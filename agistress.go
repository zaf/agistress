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
	"flag"
	"fmt"
	"math/rand"
	"net"
	"os"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

//benchmark parameters and default values
var (
	shutdown int32
	file     *os.File
	writer   *bufio.Writer
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

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()
	if *debug {
		//Open log file for writing
		file, err := os.Create("bench-" + strconv.FormatInt(time.Now().Unix(), 10) + ".csv")
		if err != nil {
			fmt.Println("Failed to create file:", err)
			os.Exit(1)
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
	//Start benchmark and wait for users input to stop
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
	var active, count, fail int64
	var avrDur int64
	logChan := make(chan string, *sess*2)
	timeChan := make(chan int64, *sess*2)
	runDelay := time.Duration(1e9 / *runs) * time.Nanosecond
	replyDelay := time.Duration(*delay) * time.Millisecond
	wg1 := new(sync.WaitGroup)
	wg1.Add(*sess)
	//Spawn pool of paraller runs
	for i := 0; i < *sess; i++ {
		ticker := time.Tick(runDelay)
		env := agiInit()
		go func(ticker <-chan time.Time) {
			defer wg1.Done()
			wg2 := new(sync.WaitGroup)
			for atomic.LoadInt32(&shutdown) == 0 {
				<-ticker
				wg2.Add(1)
				//Spawn Connections to the AGI server
				go func() {
					defer wg2.Done()
					start := time.Now()
					conn, err := net.Dial("tcp", net.JoinHostPort(*host, *port))
					if err != nil {
						atomic.AddInt64(&fail, 1)
						if *debug {
							logChan <- fmt.Sprintf("# %s\n", err)
						}
						return
					}
					atomic.AddInt64(&active, 1)
					scanner := bufio.NewScanner(conn)
					//Send AGI initialisation data
					conn.Write(env)
					//Reply with '200' to all messages from the AGI server until it hangs up
					for scanner.Scan() {
						time.Sleep(replyDelay)
						conn.Write([]byte("200 result=0\n"))
						if scanner.Text() == "HANGUP" {
							break
						}
					}
					conn.Close()
					elapsed := time.Since(start)
					timeChan <- elapsed.Nanoseconds()
					atomic.AddInt64(&active, -1)
					atomic.AddInt64(&count, 1)
					if *debug {
						logChan <- fmt.Sprintf("%d,%d,%d\n", atomic.LoadInt64(&count),
							atomic.LoadInt64(&active), elapsed.Nanoseconds())
					}
				}()

			}
			wg2.Wait()
		}(ticker)
	}
	wg3 := new(sync.WaitGroup)
	//Write to log file
	if *debug {
		wg3.Add(1)
		go func() {
			defer wg3.Done()
			for logMsg := range logChan {
				writer.WriteString(logMsg)
			}
		}()
	}
	//Calculate Average session duration for the last 1000 sessions
	wg3.Add(1)
	go func() {
		defer wg3.Done()
		var sessions int64
		for dur := range timeChan {
			sessions++
			atomic.StoreInt64(&avrDur, (atomic.LoadInt64(&avrDur)*(sessions-1)+dur)/sessions)
			if sessions >= 10000 {
				sessions = 0
			}
		}
	}()
	//Display pretty output to the user
	wg3.Add(1)
	go func() {
		defer wg3.Done()
		for {
			fmt.Print("\033[2J\033[H") //Clear screen
			fmt.Println("Running paraller AGI bench to:", net.JoinHostPort(*host, *port),
				"\nPress Enter to stop.\n",
				"\nA new run each:", runDelay,
				"\nSessions per run:", *sess,
				"\nReply delay:", replyDelay,
				"\n\nFastAGI Sessions\nActive:", atomic.LoadInt64(&active),
				"\nCompleted:", atomic.LoadInt64(&count),
				"\nDuration:", atomic.LoadInt64(&avrDur), "ns (last 10000 sessions average)",
				"\nFailed:", atomic.LoadInt64(&fail))
			if atomic.LoadInt32(&shutdown) != 0 {
				fmt.Println("Stopping...")
				if atomic.LoadInt64(&active) == 0 {
					break
				}
			}
			time.Sleep(1 * time.Second)
		}
	}()
	//Wait all FastAGI sessions to end
	wg1.Wait()
	close(logChan)
	close(timeChan)
	//Wait writing to log file to finish
	wg3.Wait()
}

func agiInit() []byte {
	//Generate AGI initialisation data
	agiData := make([]byte, 0, 512)
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
	agiData = append(agiData, "agi_version: 0.1\n"...)
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
