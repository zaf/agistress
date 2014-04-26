### A FastAGI parallel benchmark in Go
---

Copyright (C) 2013 - 2014, Lefteris Zafiris <zaf.000@gmail.com>

#### Usage flags:

- runs  : Number of runs per second. Default: 1
- sess  : Sessions per run. Default: 1
- host  : FAstAGI server host. Default: 127.0.0.1
- port  : FastAGI server port. Default: 4573
- req   : AGI request. Default: "myagi?file=echo-test"
- arg   : Argument to pass to the FastAGI server.
- cid   : Caller ID. Default: "Unknown"
- debug : Write detailed statistics output to csv file. Default: off
- delay : Delay in AGI responses to the server (milliseconds). Default: 100
- ext   : Called extension. Default: 100

This program is free software, distributed under the terms of
the GNU General Public License Version 3. See the LICENSE file
at the top of the source tree.
