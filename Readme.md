### A FastAGI benchmarking and debugging tool
---

'agistress' acts as a FastAGI client that can connect to servers
using the Asterisk Gateway Interface (AGI).
It can be used as a test tool and/or a traffic generator for FastAGI
applications. It can load and use user-defined payloads from config files.

#### Usage:
agistress [ Flags ] [ AGI arguments ]

#### Usage flags:

- conf   : Configuration file with the AGI playload in JSON format
- single : Connect and run only once displaying detailed AGI debug info. Default: false
- runs   : Number of runs per second. Default: 1
- sess   : Sessions per run. Default: 1
- host   : FAstAGI server host. Default: 127.0.0.1
- port   : FastAGI server port. Default: 4573
- req    : AGI request. Default: "myagi?file=echo-test"
- cid    : Caller ID. Default: "Unknown"
- debug  : Write detailed statistics output to csv file. Default: false
- delay  : Wait time in milliseconds before replying to server requests. Default: 50
- ext    : Called extension. Default: 100
- tls    : Enable AGI over TLS. Default : false

---

Copyright (C) 2013 - 2015, Lefteris Zafiris <zaf@fastmail.com>

This program is free software, distributed under the terms of
the GNU General Public License Version 3. See the LICENSE file
at the top of the source tree.
