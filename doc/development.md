## Development
To download all dependencies and compile statsgod

	go get -u github.com/mattn/gom
	mkdir -p $GOPATH/src/github.com/acquia/statsgod
	git clone https://github.com/acquia/statsgod $GOPATH/src/github.com/acquia/statsgod
	cd $GOPATH/src/github.com/acquia/statsgod
	gom install
	gom build -o $GOPATH/bin/statsgod


To build the debian package in a docker container

	docker build -t statsgod .
	docker run -v $(pwd)/:/usr/share/go/src/github.com/acquia/statsgod -it statsgod /bin/bash -c "make && make deb"

## Testing
For automated tests we are using http://onsi.github.io/ginkgo/ and http://onsi.github.io/gomega/. We have a combination of unit, integration and benchmark tests which are executed by Travis.ci.

	make test

## Load Testing and QA
For load testing and manual QA we have a script in /extras/loadtest.go that can be used to soak the system. There are a lot of options with invocation, so take a moment to read through and understand them. The most used flags specify the number of metrics, amount of concurrency and type of connection to test (tcp, tcp with connection pool, udp, unix and unix with connection pool).

Here is an example using the -logSent flag which will emit a line for each metric string sent. For statsgod.go, use the config option config.debug.receipt: true to emit the received/parsed metrics. Using the log\_compare.go utility you can compare each metric sent to each received/parsed metric received to ensure that nothing was lost.

1. Start statsgod.go with config.debug.receipt set to true and redirect to a file:

		gom exec go run statsgod.go 2>&1 > /tmp/statsgod.output

2. Start loadtest.go with the -logSent flag and redirect to a file:

		gom exec go run extras/loadtest.go [options] -logSent=true 2>&1 > /tmp/statsgod.input

3. After collecting input and output, compare using the log\_compare.go utility:

		gom exec go run extras/log_compare.go -in=/tmp/statsgod.input -out=/tmp/statsgod.output

## Profiling
To profile this program we have been using the runtime.pprof library. Build a binary, start the daemon and run it through a real-world set of tests to fully exercise the code. After the program exits it will create a file in the same directory called "statsgod.prof". This profile, along with the binary, can be used to learn more about the runtime.

Config:
	debug.profile = true

	gom build -o sg # Build a binary called "sg"
	./sg -config=/path/to/config # start the daemon and don't forget to set config.debug.profile = true
	# run some tests and stop the daemon. "sg" is the binary and "statsgod.prof" is the profile.
	go tool pprof --help # See all of the options.
	go tool pprof --dot sg statsgod.prof # Generate a dot file for graphviz.
	go tool pprof --gif sg statsgod.prof # Generate a gif of the visualization.
	go tool pprof sg statsgod.prof # Interactive mode:
	(pprof) top30 -cum # View the top 30 functions sorted by cumulative sample counts.


