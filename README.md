statsgod
========

[![Build Status](https://magnum.travis-ci.com/acquia/statsgod.svg?token=V11Dcpsz9xGpCipC8pBD&branch=master)](https://magnum.travis-ci.com/acquia/statsgod)

Statsgod is an experimental Go implementation (or deviation) of Etsy's statsd service.

## Usage
---
```
Usage:  statsgod [args]
  -config="config.yml": YAML config file path
  -debug=false: Debugging mode
```

### Example:
1.  start the daemon.
	
	go run statsgod.go

2. Start a testing receiver.

	go run test_receiver.go

3. Send data to the daemon. Set a gauge to 3 for the_magic_number

	echo "the_magic_number:3|g" | nc localhost 8125 # TCP
	echo "the_magic_number:3|g" | nc -4u -w0 localhost 8126 # UDP
	echo "the_magic_number:3|g" | nc -U /tmp/statsgod.sock # Unix Socket

## Configuration
All runtime options are specified in a YAML file. e.g. ```go run statsgod.go -config=/etc/statsgod.yml```. See config.yml for an example with all default/configurable values.

## Development
To download all dependencies and compile statsgod

	go get -u github.com/mattn/gom
	mkdir -p $GOPATH/src/github.com/acquia/statsgod
	git clone https://github.com/acquia/statsgod $GOPATH/src/github.com/acquia/statsgod
	cd $GOPATH/src/github.com/acquia/statsgod
	gom install
	gom build -o $GOPATH/bin/statsgod

## Testing
We are using github.com/stretchr/testify to do basic assertions on top of the testing package. The tests are mostly focused on units, but behavior tests are equally welcome. Example:

	make test

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
