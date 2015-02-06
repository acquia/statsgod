statsgod
========


Statsgod is a metric aggregation service inspired by the statsd project. Written in Golang, it increases performance and can be deployed without dependencies. This project uses the same metric string format as statsd, but adds new features like alternate sockets, authentication, etc.

## Usage

	Usage:  statsgod [args]
	  -config="config.yml": YAML config file path

### Example:
1.  start the daemon.

		gom exec go run statsgod.go

2. Start a testing receiver.

		gom exec go run test_receiver.go

3. Send data to the daemon. Set a gauge to 3 for your.metric.name

		echo "your.metric.name:3|g" | nc localhost 8125 # TCP
		echo "your.metric.name:3|g" | nc -4u -w0 localhost 8126 # UDP
		echo "your.metric.name:3|g" | nc -U /tmp/statsgod.sock # Unix Socket

### Metric Format
Data is sent over a socket connection as a string using the format: [namespace]:[value]|[type] where the namespace is a dot-delimeted string like "user.login.success". Values are floating point numbers represented as strings. The metric type uses the following values:

- Gauge   (g):  constant metric, keeps the last value.
- Counter (c):  increment/decrement a given namespace.
- Timer   (ms): a timer that calculates averages (see below).
- Set     (s): a count of unique values sent during a flush period.

Optionally you may denote that a metric has been sampled by adding "|@0.75" (where 0.75 is the sample rate as a float). Counters will inflate the value accordingly so that it can be accurately used to calculate a rate.

An example data string would be "user.login.success:123|c|@0.9"

### Sending Metrics
Client code can send metrics via any one of three sockets which listen concurrently:

1. TCP
	- Allows multiple metrics to be sent over a connection, separated by a newline character.
	- Connection will remain open until closed by the client.
	- Config:
		- connection.udp.enabled
		- connection.udp.host
		- connection.udp.port

2. UDP
	- Allows multiple metrics to be sent over a connection, separated by a newline character. Note, you should be careful to not exceed the maximum packet size (default 1024 bytes).
	- Config:
		- connection.udp.enabled
		- connection.udp.host
		- connection.udp.port
		- connection.udp.maxpacket (buffer size to read incoming packets)

3. Unix Domain Socket
	- Allows multiple metrics to be sent over a connection, separated by a newline character.
	- Connection will remain open until closed by the client.
	- Config:
		- connection.unix.enabled
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
		[prefix].my.counter.[suffix] [timestamp] 3
		[prefix].my.counter.[suffix] [timestamp] [3/(duration of flush interval in seconds)]

2. Gauges - these are a "last in" measurement which discards all previously sent values:

		my.gauge:1|g
		my.gauge:2|g
		my.gauge:3|g
		# flush only sends the last value:
		[prefix].my.gauge.[suffix] [timestamp] 3

3. Timers - these are timed values measured in milliseconds. Statsgod provides several calculated values based on the sent metrics:

		my.timer:100|ms
		my.timer:200|ms
		my.timer:300|ms
		# flush produces several calculated fields:
		[prefix].my.timer.mean_value.[suffix] [timestamp] [mean]
		[prefix].my.timer.median_value.[suffix] [timestamp] [median]
		[prefix].my.timer.min_value.[suffix] [timestamp] [min]
		[prefix].my.timer.max_value.[suffix] [timestamp] [max]
		[prefix].my.timer.mean_90.[suffix] [timestamp] [mean in 90th percentile]
		[prefix].my.timer.upper_90.[suffix] [timestamp] [upper in 90th percentile]
		[prefix].my.timer.sum_90.[suffix] [timestamp] [sum in 90th percentile]

4. Sets - these track the number of unique values sent during a flush interval:

		my.unique:1|s
		my.unique:2|s
		my.unique:2|s
		my.unique:1|s
		# flush produces a single value counting the unique metrics sent:
		[prefix].my.unique.[suffix] [timestamp] 2

## Prefix/Suffix
Prefixes and suffixes noted above can be customized in the configuration. Metrics will render as [prefix].[type prefix].[metric namespace].[type suffix].[suffix]. You may also use empty strings in the config for any values you do not wish statsgod to prefix/suffix before relaying.

	namespace:
		prefix: "stats"
		prefixes:
			counters: "counts"
			gauges: "gauges"
			rates: "rates"
			sets: "sets"
			timers: "timers"
		suffix: ""
		suffixes:
			counters: ""
			gauges: ""
			rates: ""
			sets: ""
			timers: ""



## Authentication
Auth is handled via the statsgod.Auth interface. Currently there are two types of authentication: no-auth and token-auth, which are specified in the configuration file:

1. No auth

		# config.yml
		service:
			auth: "none"

Works as you might expect, all metrics strings are parsed without authentication or manipulation. This is the default behavior.

2. Token auth

		# config.yml
		service:
			auth: "token"
			tokens:
				"token-name": false
				"32a3c4970093": true

"token" checks the configuration file for a valid auth token. The config file may specify as many tokens as needed in the service.tokens map. These are written as "string": bool where the string is the token and the bool is whether or not the token is valid. Please note that these are read into memory when the proces is started, so changes to the token map require a restart.

When sending metrics, the token is specified at the beginning of the metric namespace followed by a dot. For example, a metric "32a3c4970093.my.metric:123|g" would look in the config tokens for the string "32a3c4970093" and see if that is set to true. If valid, the process will strip the token from the namespace, only parsing and aggregating "my.metric:123|g". NOTE: since metric namespaces are dot-delimited, you cannot use a dot in a token.

## Signal handling
The statsgod service is equipped to handle the following signals:

1. Shut down the sockets and clean up before exiting.
	- SIGABRT
	- SIGINT
	- SIGTERM
	- SIGQUIT

2. Reload\* the configuration without restarting.
	- SIGHUP

\* When reloading configuration, not all values will affect the current runtime. The following are only available on start up and not currently reloadable:

- connection.\*
- relay.\*
- stats.percentile
- debug.verbose
- debug.profile

## Development
[Read more about the development process.](doc/development.md)

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
