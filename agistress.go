/*
	A FastAGI parallel benchmark in go

	Copyright (C) 2013 - 2014, Lefteris Zafiris <zaf.000@gmail.com>

	This program is free software, distributed under the terms of
	the GNU General Public License Version 3. See the LICENSE file
	at the top of the source tree.
*/

package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// Benchmark parameters and default values
var (
	shutdown int32
	file     *os.File
	writer   *bufio.Writer
	conf     = flag.String("conf", "", "Configuration file")
	debug    = flag.Bool("debug", false, "Write detailed statistics output to csv file")
	host     = flag.String("host", "127.0.0.1", "FAstAGI server host")
	port     = flag.String("port", "4573", "FastAGI server port")
	runs     = flag.Float64("runs", 1, "Number of runs per second")
	sess     = flag.Int("sess", 1, "Sessions per run")
	delay    = flag.Int("delay", 100, "Delay in AGI responses to the server (milliseconds)")
	req      = flag.String("req", "myagi?file=echo-test", "AGI request")
	arg      = flag.String("arg", "", "Argument to pass to the FastAGI server")
	cid      = flag.String("cid", "Unknown", "Caller ID")
	ext      = flag.String("ext", "100", "Called extension")
)

// Benchmark session data
type Bench struct {
	Count      int64
	Active     int64
	Fail       int64
	AvrDur     int64
	LogChan    chan string
	TimeChan   chan int64
	RunDelay   time.Duration
	ReplyDelay time.Duration
	Env        []byte
}

// Configuration data
type Config struct {
	AGIEnv []string
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()
	if *debug {
		// Open log file for writing
		file, err := os.Create("bench-" + strconv.FormatInt(time.Now().Unix(), 10) + ".csv")
		if err != nil {
			log.Fatalln("Failed to create file:", err)
		}
		writer = bufio.NewWriter(file)
		defer func() {
			writer.Flush()
			file.Close()
		}()
		writer.WriteString("#Started benchmark at: " + time.Now().String() + "\n")
		writer.WriteString(fmt.Sprintf("#Host: %v\n#Port: %v\n#Runs: %v\n", *host, *port, *runs))
		writer.WriteString(fmt.Sprintf("#Sessions: %v\n#Delay: %v\n#Reguest: %v\n", *sess, *delay, *req))
		writer.WriteString("#completed,active,duration\n")
	}
	rand.Seed(time.Now().UTC().UnixNano())
	wg := new(sync.WaitGroup)
	wg.Add(1)
	// Start benchmark and wait for users input to stop
	go agiBench(wg)
	bufio.NewReader(os.Stdin).ReadString('\n')
	atomic.StoreInt32(&shutdown, 1)
	wg.Wait()
	if *debug {
		writer.WriteString("#Stopped benchmark at: " + time.Now().String() + "\n")
	}
}

func agiBench(wg *sync.WaitGroup) {
	defer wg.Done()
	b := benchInit()
	wg1 := new(sync.WaitGroup)
	wg1.Add(*sess)
	// Spawn pool of paraller runs
	for i := 0; i < *sess; i++ {
		ticker := time.Tick(b.RunDelay)
		go session(ticker, b, wg1)
	}
	wg2 := new(sync.WaitGroup)
	// Write to log file
	if *debug {
		wg2.Add(1)
		go logger(b, wg2)
	}
	// Calculate session duration and display some output on the console
	wg2.Add(2)
	go calcAvrg(b, wg2)
	go consoleOutput(b, wg2)
	// Wait all FastAGI sessions to end
	wg1.Wait()
	close(b.LogChan)
	close(b.TimeChan)
	// Wait logger and console output to finish
	wg2.Wait()
}

//
func session(ticker <-chan time.Time, b *Bench, wg *sync.WaitGroup) {
	defer wg.Done()
	wg1 := new(sync.WaitGroup)
	for atomic.LoadInt32(&shutdown) == 0 {
		<-ticker
		wg1.Add(1)
		// Spawn Connections to the AGI server
		go func() {
			defer wg1.Done()
			start := time.Now()
			conn, err := net.Dial("tcp", net.JoinHostPort(*host, *port))
			if err != nil {
				atomic.AddInt64(&b.Fail, 1)
				if *debug {
					b.LogChan <- fmt.Sprintf("# %s\n", err)
				}
				return
			}
			atomic.AddInt64(&b.Active, 1)
			scanner := bufio.NewScanner(conn)
			// Send AGI initialisation data
			conn.Write(b.Env)
			// Reply with '200' to all messages from the AGI server until it hangs up
			for scanner.Scan() {
				time.Sleep(b.ReplyDelay)
				conn.Write([]byte("200 result=0\n"))
				if scanner.Text() == "HANGUP" {
					conn.Write([]byte("200 result=0\nHANGUP\n"))
					break
				}
			}
			conn.Close()
			elapsed := time.Since(start)
			b.TimeChan <- elapsed.Nanoseconds()
			atomic.AddInt64(&b.Active, -1)
			atomic.AddInt64(&b.Count, 1)
			if *debug {
				b.LogChan <- fmt.Sprintf("%d,%d,%d\n", atomic.LoadInt64(&b.Count),
					atomic.LoadInt64(&b.Active), elapsed.Nanoseconds())
			}
		}()

	}
	wg1.Wait()
}

// Write to log file
func logger(b *Bench, wg *sync.WaitGroup) {
	defer wg.Done()
	for logMsg := range b.LogChan {
		writer.WriteString(logMsg)
	}
}

// Calculate Average session duration for the last 1000 sessions
func calcAvrg(b *Bench, wg *sync.WaitGroup) {
	defer wg.Done()
	var sessions int64
	for dur := range b.TimeChan {
		sessions++
		atomic.StoreInt64(&b.AvrDur, (atomic.LoadInt64(&b.AvrDur)*(sessions-1)+dur)/sessions)
		if sessions >= 10000 {
			sessions = 0
		}
	}
}

// Display pretty output to the user
func consoleOutput(b *Bench, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		fmt.Print("\033[2J\033[H") //Clear screen
		fmt.Println("Running paraller AGI bench to:", net.JoinHostPort(*host, *port),
			"\nPress Enter to stop.\n",
			"\nA new run each:", b.RunDelay,
			"\nSessions per run:", *sess,
			"\nReply delay:", b.ReplyDelay,
			"\n\nFastAGI Sessions\nActive:", atomic.LoadInt64(&b.Active),
			"\nCompleted:", atomic.LoadInt64(&b.Count),
			"\nDuration:", atomic.LoadInt64(&b.AvrDur), "ns (last 10000 sessions average)",
			"\nFailed:", atomic.LoadInt64(&b.Fail))
		if atomic.LoadInt32(&shutdown) != 0 {
			fmt.Println("Stopping...")
			if atomic.LoadInt64(&b.Active) == 0 {
				break
			}
		}
		time.Sleep(1 * time.Second)
	}
}

// Initialize benchmark session
func benchInit() *Bench {
	b := new(Bench)
	b.LogChan = make(chan string, *sess*2)
	b.TimeChan = make(chan int64, *sess*2)
	b.RunDelay = time.Duration(1e9 / *runs) * time.Nanosecond
	b.ReplyDelay = time.Duration(*delay) * time.Millisecond
	b.Env = agiInit()
	return b
}

// Generate AGI initialisation data
func agiInit() []byte {
	agiData := make([]byte, 0, 512)
	if *conf != "" {
		// Parse config file
		file, err := os.Open(*conf)
		if err != nil {
			log.Println("Failed to open config file:", err, "\nUsing default settings")
			goto DEF
		}
		var confData Config
		decoder := json.NewDecoder(file)
		err = decoder.Decode(&confData)
		if err != nil {
			log.Println("Failed to parse config file:", err, "\nUsing default settings")
			goto DEF
		}
		for _, par := range confData.AGIEnv {
			agiData = append(agiData, par+"\n"...)
		}
		agiData = append(agiData, "\n"...)
		return agiData
	}

	//Deafult AGI environment settings
DEF:
	agiData = append(agiData, "agi_network: yes\n"...)
	if len(*req) > 0 {
		agiData = append(agiData, "agi_network_script: "+*req+"\n"...)
		agiData = append(agiData, "agi_request: agi://"+*host+"/"+*req+"\n"...)
	} else {
		agiData = append(agiData, "agi_request: agi://"+*host+"\n"...)
	}
	agiData = append(agiData, "agi_channel: SIP/1234-00000000\n"...)
	agiData = append(agiData, "agi_language: en\n"...)
	agiData = append(agiData, "agi_type: SIP\n"...)
	agiData = append(agiData, "agi_uniqueid: "+strconv.Itoa(1e8+rand.Intn(9e8-1))+"\n"...)
	agiData = append(agiData, "agi_version: 10.1.1.0\n"...)
	agiData = append(agiData, "agi_callerid: "+*cid+"\n"...)
	agiData = append(agiData, "agi_calleridname: "+*cid+"\n"...)
	agiData = append(agiData, "agi_callingpres: 67\n"...)
	agiData = append(agiData, "agi_callingani2: 0\n"...)
	agiData = append(agiData, "agi_callington: 0\n"...)
	agiData = append(agiData, "agi_callingtns: 0\n"...)
	agiData = append(agiData, "agi_dnid: "+*ext+"\n"...)
	agiData = append(agiData, "agi_rdnis: unknown\n"...)
	agiData = append(agiData, "agi_context: default\n"...)
	agiData = append(agiData, "agi_extension: "+*ext+"\n"...)
	agiData = append(agiData, "agi_priority: 1\n"...)
	agiData = append(agiData, "agi_enhanced: 0.0\n"...)
	agiData = append(agiData, "agi_accountcode: \n"...)
	agiData = append(agiData, "agi_threadid: "+strconv.Itoa(1e8+rand.Intn(9e8-1))+"\n"...)
	if len(*arg) > 0 {
		agiData = append(agiData, "agi_arg_1: "+*arg+"\n\n"...)
	} else {
		agiData = append(agiData, "\n"...)
	}
	return agiData
}
