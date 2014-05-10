# Statsgod

Statsgod is an experimental Go implementation (or deviation) of Etsy's statsd service.  It does not yet support all statsd metric types, configuration options or other capabilities and has not yet been thoroughly tested.

The original statsd was written in node.js. This version is written in Go and utilizes capabilities such as Go channels to improve overall concurrency and scalability.


## Usage
---
```
Usage:  statsgod [args]
 -c configfile		configuration yaml file to process
 -h host			host to listen on
 -p port			port to listen on
 -r remotehost		remote host to send data to
 -q remoteport		remote port to send data to
 -t flushtime		time for the metrics to be flushed. default 5s
 -e percentile		which percentile we want to track for metrics
 -d 				debug mode

# Example, first start the daemon.
go run statsgod.go

# Start a testing receiver.
go run test_receiver.go

# Send data to the daemon. Set a gauge to 3 for the_magic_number
echo "the_magic_number:3|g" | nc localhost 5000
```


## TODO
---
Statsgod is very much a work in progress and has a number of missing features, including, but not limited to:

* Support for all statsd metrics (timers, gauges, counters, etc.) - In Progress
* Correctly support the timers with working percentile support
* Working debug subsystem
* Ensure the channel is optimized
* Working configuration file
* Unit tests
* Operate with UDP or TCP
* Configurable flush time
* Performance runing and tunable channels
* Have the metric types by pluggable?
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
