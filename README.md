statsgod
========

[![Build Status](https://magnum.travis-ci.com/acquia/statsgod.svg?token=V11Dcpsz9xGpCipC8pBD&branch=master)](https://magnum.travis-ci.com/acquia/statsgod)

Statsgod is an experimental Go implementation (or deviation) of Etsy's statsd service.

## Usage

	Usage:  statsgod [args]
	  -config="config.yml": YAML config file path
	  -debug=false: Debugging mode

### Example:
1.  start the daemon.

		gom exec go run statsgod.go

2. Start a testing receiver.

		gom exec go run test_receiver.go

3. Send data to the daemon. Set a gauge to 3 for the_magic_number

		echo "the_magic_number:3|g" | nc localhost 8125 # TCP
		echo "the_magic_number:3|g" | nc -4u -w0 localhost 8126 # UDP
		echo "the_magic_number:3|g" | nc -U /tmp/statsgod.sock # Unix Socket

### Sending Metrics
Client code can send metrics via any one of three sockets which listen concurrently:

1. TCP
	- Allows multiple metrics to be sent over a connection, separated by a newline character.
	- Connection will remain open until closed by the client.
	- Config:
		- connection.udp.host
		- connection.udp.port

2. UDP
	- Allows multiple metrics to be sent over a connection, separated by a newline character. Note, you should be careful to not exceed the maximum packet size (default 1024 bytes).
	- Config:
		- connection.udp.host
		- connection.udp.port
		- connection.udp.maxpacket (buffer size to read incoming packets)

3. Unix Domain Socket
	- Allows multiple metrics to be sent over a connection, separated by a newline character.
	- Connection will remain open until closed by the client.
	- Config:
		- config: connection.unix.file (path to the sock file)

## Configuration
All runtime options are specified in a YAML file. Please see example.config.yml for defaults. e.g.

	go run statsgod.go -config=/etc/statsgod.yml

## Stats Types
Statsgod provides support for the following metric types.

1. Counters - these are cumulative values that calculate the sum of all metrics sent. A rate is also calculated to determine how many values were sent during the flush interval:

		my.counter:1|c
		my.counter:1|c
		my.counter:1|c
		# flush produces a count and a rate:
		[stats prefix].my.counter [timestamp] 3
		[rate prefix].my.counter [timestamp] [3/(duration of flush interval in seconds)]

2. Gauges - these are a "last in" measurement which discards all previously sent values:

		my.gauge:1|g
		my.gauge:2|g
		my.gauge:3|g
		# flush only sends the last value:
		[gauge prefix].my.gauge [timestamp] 3

3. Timers - these are timed values measured in milliseconds. Statsgod provides several calculated values based on the sent metrics:

		my.timer:100|ms
		my.timer:200|ms
		my.timer:300|ms
		# flush produces several calculated fields:
		[timer prefix].my.timer.mean_value [timestamp] [mean]
		[timer prefix].my.timer.median_value [timestamp] [median]
		[timer prefix].my.timer.min_value [timestamp] [min]
		[timer prefix].my.timer.max_value [timestamp] [max]
		[timer prefix].my.timer.mean_90 [timestamp] [mean in 90th percentile]
		[timer prefix].my.timer.upper_90 [timestamp] [upper in 90th percentile]
		[timer prefix].my.timer.sum_90 [timestamp] [sum in 90th percentile]

4. Sets - these track the number of unique values sent during a flush interval:

		my.unique:1|s
		my.unique:2|s
		my.unique:2|s
		my.unique:1|s
		# flush produces a single value counting the unique metrics sent:
		[set prefix].my.unique [timestamp] 2

Note that the prefixes noted above can be customized in the configuration. Prefixes will render as [global].[type].[metric namespace]. You may also use empty strings in the config if you do not wish statsgod to prefix before relaying.

	stats:
		prefix:
			counters: "counts"
			gauges: "gauges"
			global: "stats"
			rates: "rates"
			sets: "sets"
			timers: "timers"

## Development
To download all dependencies and compile statsgod

	go get -u github.com/mattn/gom
	mkdir -p $GOPATH/src/github.com/acquia/statsgod
	git clone https://github.com/acquia/statsgod $GOPATH/src/github.com/acquia/statsgod
	cd $GOPATH/src/github.com/acquia/statsgod
	gom install
	gom build -o $GOPATH/bin/statsgod

## Testing
For automated tests we are using http://onsi.github.io/ginkgo/ and http://onsi.github.io/gomega/. We have a combination of unit, integration and benchmark tests which are executed by Travis.ci.

	make test

## Load Testing and QA
For load testing and manual QA we have a script in /extras/loadtest.go that can be used to soak the system. There are a lot of options with invocation, so take a moment to read through and understand them. The most used flags specify the number of metrics, amount of concurrency and type of connection to test (tcp, tcp with connection pool, udp, unix and unix with connection pool).

Here is an example using the -logSent flag which will emit a line for each metric string sent. For statsgod.go, use the config option config.debug.receipt: true to emit the received/parsed metrics. Using the log\_compare.go utility you can compare each metric sent to each received/parsed metric received to ensure that nothing was lost.

1. Start statsgod.go with config.debug.receipt set to true and redirect to a file:

		gom exec go run statsgod.go 2>&1 /tmp/statsgod.output

2. Start loadtest.go with the -logSent flag and redirect to a file:

		gom exec go run extras/loadtest.go [options] -logSent=true 2>&1 /tmp/statsgod.input

3. After collecting input and output, compare using the log\_compare.go utility:

		gom exec go run extras/log_compare.go -in=/tmp/statsgod.input -out=/tmp/statsgod.output

## License
Except as otherwise noted this software is licensed under the [Apache License, Version 2.0](http://www.apache.org/licenses/LICENSE-2.0.html)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
