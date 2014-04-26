A FastAGI parallel benchmark in Go

Copyright (C) 2013 - 2014, Lefteris Zafiris <zaf.000@gmail.com>

Usage flags:

  -arg="": Argument to pass to the FastAGI server
  -cid="Unknown": Caller ID
  -debug=false: Write detailed statistics output to csv file
  -delay=100: Delay in AGI responses to the server (milliseconds)
  -ext="100": Called extension
  -host="127.0.0.1": FAstAGI server host
  -port="4573": FastAGI server port
  -req="myagi?file=echo-test": AGI request
  -runs=1: Number of runs per second
  -sess=1: Sessions per run

This program is free software, distributed under the terms of
the GNU General Public License Version 2. See the LICENSE file
at the top of the source tree.
