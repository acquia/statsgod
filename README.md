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

## Configuration
All runtime options are specified in a YAML file. e.g.

	go run statsgod.go -config=/etc/statsgod.yml

See config.yml for an example with all default/configurable values.

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
