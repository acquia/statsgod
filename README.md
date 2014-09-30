statsgod
========

[![Build Status](https://travis-ci.org/syrneus/statsgod.svg?branch=master)](https://travis-ci.org/syrneus/statsgod)

Statsgod is an experimental Go implementation (or deviation) of Etsy's statsd service.  It does not yet support all statsd configuration options or other capabilities and has not yet been performance tested. All of the metric types are not yet thoroughly tested.

The original statsd was written in node.js. This version is written in Go and utilizes capabilities such as Go channels to improve overall concurrency and scalability.


## Usage
---
```
Usage:  statsgod [args]
 -config configfile		configuration yaml file to process
 -host host				host to listen on
 -port port				port to listen on
 -graphiteHost 			remotehost: remote host to send data to
 -graphitePort 			remoteport: remote port to send data to
 -flushTime 			flushtime: time for the metrics to be flushed. default 10s
 -percentile 			percentile: which percentile we want to track for metrics
 -debug 				debug mode
```

### Example:
1.  start the daemon.
	
		go run statsgod.go

2. Start a testing receiver.

		go run test_receiver.go

3. Send data to the daemon. Set a gauge to 3 for the_magic_number

		echo "the_magic_number:3|g" | nc localhost 5000


## Development
To download all dependencies and compile statsgod

	go get -u github.com/mattn/gom
	mkdir -p $GOPATH/src/github.com/syrneus/statsgod
	git clone https://github.com/syrneus/statsgod $GOPATH/src/github.com/syrneus/statsgod
	cd $GOPATH/src/github.com/syrneus/statsgod
	gom install
	gom build -o $GOPATH/bin/statsgod


## TODO
---
Statsgod is very much a work in progress and has a number of missing features, including, but not limited to:

* Support multiple percentiles
* Working configuration file
* Unit tests
* Operate with UDP or TCP
* Load test and soak test
* Pickle support for sending to Graphite in parallel?
* Performance tuning and tunable channels
* Have the metric types be pluggable?
* Pluggable storage backend?

## License
---
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
