/**
 * Copyright 2014 Acquia, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

/**
 * Package main: statsgod is a metrics aggregator inspired by statsd. The
 * main feature is to provide a server which accepts metrics over time,
 * aggregates them and forwards on to permanent storage.
 */
package main

import (
	"flag"
	"fmt"
	"github.com/acquia/statsgod/statsgod"
	"io/ioutil"
	"os"
)

const (
	// AvailableMemory is amount of available memory for the process.
	AvailableMemory = 10 << 20 // 10 MB, for example
	// AverageMemoryPerRequest is how much memory we want to use per request.
	AverageMemoryPerRequest = 10 << 10 // 10 KB
	// MaxReqs is how many requests.
	MaxReqs = AvailableMemory / AverageMemoryPerRequest
)

// parseChannel containing received metric strings.
var parseChannel = make(chan string, MaxReqs)

// relayChannel containing the Metric objects.
var relayChannel = make(chan *statsgod.Metric, MaxReqs)

// finishChannel is used to respond to a quit signal.
var finishChannel = make(chan int)

// quit is a shared value to instruct looping goroutines to stop.
var quit = false

// CLI flags.
var configFile = flag.String("config", "/etc/statsgod/config.yml", "YAML config file path")

func main() {
	// Load command line options.
	flag.Parse()

	// Load the YAML config.
	var config, _ = statsgod.CreateConfig(*configFile)

	// Set up the logger based on the configuration.
	var logger statsgod.Logger
	if config.Debug.Verbose {
		logger = *statsgod.CreateLogger(os.Stdout, os.Stdout, os.Stdout, os.Stderr)
		logger.Info.Println("Debugging mode enabled")
		logger.Info.Printf("Loaded Config: %v", config)
	} else {
		logger = *statsgod.CreateLogger(ioutil.Discard, os.Stdout, os.Stdout, os.Stderr)
	}

	// Set up the backend relay.
	relay := statsgod.CreateRelay(config, logger)

	// Set up the authentication.
	auth := statsgod.CreateAuth(config)

	// Parse the incoming messages and convert to metrics.
	go statsgod.ParseMetrics(parseChannel, relayChannel, auth, logger, &quit)

	// Flush the metrics to the remote stats collector.
	for i := 0; i < config.Relay.Concurrency; i++ {
		go statsgod.RelayMetrics(relay, relayChannel, logger, &config, &quit)
	}

	// Listen on the TCP socket.
	tcpAddr := fmt.Sprintf("%s:%d", config.Connection.Tcp.Host, config.Connection.Tcp.Port)
	socketTcp := statsgod.CreateSocket(statsgod.SocketTypeTcp, tcpAddr).(*statsgod.SocketTcp)
	go socketTcp.Listen(parseChannel, logger, &config)

	// Listen on the UDP socket.
	udpAddr := fmt.Sprintf("%s:%d", config.Connection.Udp.Host, config.Connection.Udp.Port)
	socketUdp := statsgod.CreateSocket(statsgod.SocketTypeUdp, udpAddr).(*statsgod.SocketUdp)
	go socketUdp.Listen(parseChannel, logger, &config)

	// Listen on the Unix socket.
	socketUnix := statsgod.CreateSocket(statsgod.SocketTypeUnix, config.Connection.Unix.File).(*statsgod.SocketUnix)
	go socketUnix.Listen(parseChannel, logger, &config)

	// Listen for OS signals.
	statsgod.ListenForSignals(finishChannel, &config, configFile, logger)

	// Wait until the program is finished.
	select {
	case <-finishChannel:
		logger.Info.Println("Exiting program.")
		quit = true
		socketTcp.Close(logger)
		socketUdp.Close(logger)
		socketUnix.Close(logger)
	}
}
