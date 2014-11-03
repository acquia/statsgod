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
 * Package main: statsgod is an experimental metrics aggregator inspired by
 * statsd. The intent is to provide a server which accepts metrics over time,
 * aggregates them and forwards on to permanent storage.
 *
 * Data is sent over a TCP socket in the format [namespace]:[value]|[type]
 * where the namespace is a dot-delimeted string like "user.login.success".
 * Values are floating point numbers represented as strings. The metric type
 * uses the following values:
 *
 * Gauge   (g):  constant metric, value persists until the server is restarted.
 * Counter (c):  increment/decrement a given namespace.
 * Timer   (ms): a timer that calculates average, 90% percentile, etc.
 *
 * An example data string would be "user.login.success:123|c"
 */
package main

import (
	"flag"
	"fmt"
	"github.com/acquia/statsgod/statsgod"
	"io/ioutil"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"
	"time"
)

const (
	// AvailableMemory is amount of available memory for the process.
	AvailableMemory = 10 << 20 // 10 MB, for example
	// AverageMemoryPerRequest is how much memory we want to use per request.
	AverageMemoryPerRequest = 10 << 10 // 10 KB
	// MaxReqs is how many requests.
	MaxReqs = AvailableMemory / AverageMemoryPerRequest
)

// The channel containing received metric strings.
var parseChannel = make(chan string, MaxReqs)

// The channel containing the Metric objects.
var flushChannel = make(chan *statsgod.Metric, MaxReqs)

var finishChannel = make(chan int)

// CLI flags.
var configFile = flag.String("config", "/etc/statsgod/config.yml", "YAML config file path")
var debug = flag.Bool("debug", false, "Debugging mode")
var profile = flag.Bool("profile", false, "Profiling mode")

var backendRelay statsgod.MetricRelay

func main() {
	// Load command line options.
	flag.Parse()

	var logger statsgod.Logger

	// Load the YAML config.
	var config, _ = statsgod.LoadConfig(*configFile)

	if *profile || config.Debug.Profile {
		// Allow the flag to override the config.
		config.Debug.Profile = true
		f, err := os.Create("statsgod.prof")
		if err != nil {
			panic(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	if *debug || config.Debug.Verbose {
		// Allow the debug flag to override config.
		config.Debug.Verbose = true
		logger = *statsgod.CreateLogger(os.Stdout, os.Stdout, os.Stdout, os.Stderr)
		logger.Info.Println("Debugging mode enabled")
	} else {
		logger = *statsgod.CreateLogger(ioutil.Discard, os.Stdout, os.Stdout, os.Stderr)
	}

	logger.Info.Printf("Loaded Config: %v", config)

	// Set up the backend relay.
	switch config.Relay.Type {
	case "carbon":
		// Create a relay to carbon.
		relay := statsgod.CreateRelay("carbon").(*statsgod.CarbonRelay)
		relay.FlushInterval = config.Relay.Flush
		relay.Percentile = config.Stats.Percentile
		// Create a connection pool for the relay to use.
		pool, err := statsgod.CreateConnectionPool(config.Relay.Concurrency, config.Carbon.Host, config.Carbon.Port, config.Relay.Timeout, logger)
		checkError(err, "Creating connection pool", true)
		relay.ConnectionPool = pool
		backendRelay = statsgod.MetricRelay(relay)
		logger.Info.Println("Relaying metrics to carbon backend")
	default:
		relay := statsgod.CreateRelay("mock").(*statsgod.MockRelay)
		relay.FlushInterval = config.Relay.Flush
		relay.Percentile = config.Stats.Percentile
		backendRelay = statsgod.MetricRelay(relay)
		logger.Info.Println("Relaying metrics to mock backend")
	}

	// Parse the incoming messages and convert to metrics.
	go parseMetrics(logger)

	// Flush the metrics to the remote stats collector.
	for i := 0; i < config.Relay.Concurrency; i++ {
		go flushMetrics(logger, config)
	}

	tcpAddr := fmt.Sprintf("%s:%d", config.Connection.Tcp.Host, config.Connection.Tcp.Port)
	socketTcp := statsgod.CreateSocket(statsgod.SocketTypeTcp, tcpAddr).(*statsgod.SocketTcp)
	go socketTcp.Listen(parseChannel, logger)

	udpAddr := fmt.Sprintf("%s:%d", config.Connection.Udp.Host, config.Connection.Udp.Port)
	socketUdp := statsgod.CreateSocket(statsgod.SocketTypeUdp, udpAddr).(*statsgod.SocketUdp)
	go socketUdp.Listen(parseChannel, logger)

	socketUnix := statsgod.CreateSocket(statsgod.SocketTypeUnix, config.Connection.Unix.File).(*statsgod.SocketUnix)
	go socketUnix.Listen(parseChannel, logger)

	// Signal handling. Any signal that should cause this program to stop needs to
	// also do some cleanup before exiting.
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel,
		syscall.SIGABRT,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	go func() {
		s := <-signalChannel
		logger.Info.Printf("Processed signal %v", s)
		socketTcp.Close(logger)
		socketUdp.Close(logger)
		socketUnix.Close(logger)
		finishChannel <- 1
	}()

	select {
	case <-finishChannel:
		logger.Info.Println("Exiting program.")
	}
}

// Parses the strings received from clients and creates Metric structures.
func parseMetrics(logger statsgod.Logger) {

	for {
		// Process the channel as soon as requests come in. If they are valid Metric
		// structures, we move them to a new channel to be flushed on an interval.
		select {
		case metricString := <-parseChannel:
			metric, err := statsgod.ParseMetricString(metricString)
			if err != nil {
				logger.Error.Printf("Invalid metric: %v", metricString)
				continue
			}
			// Push the metric onto the channel to be aggregated and flushed.
			flushChannel <- metric
		}

	}
}

// Flushes the metrics in-memory to the permanent storage facility. At this
// point we are receiving Metric structures from a channel that need to be
// aggregated by the specified namespace. We do this immediately, then when
// the specified flush interval passes, we send aggregated metrics to storage.
func flushMetrics(logger statsgod.Logger, config statsgod.ConfigValues) {
	// Use a tick channel to determine if a flush message has arrived.
	tick := time.Tick(config.Relay.Flush)
	logger.Info.Printf("Flushing every %v", config.Relay.Flush)

	// Internal storage.
	metrics := make(map[string]statsgod.Metric)

	for {
		// Process the metrics as soon as they arrive on the channel. If nothing has
		// been added during the flush interval duration, continue the loop to allow
		// it to flush the data.
		select {
		case metric := <-flushChannel:
			statsgod.AggregateMetric(metrics, *metric)
			if config.Debug.Receipt {
				logger.Info.Printf("Metric: %v", metrics[metric.Key])
			}
		case <-time.After(config.Relay.Flush):
			logger.Trace.Println("Metric Channel Timeout")
			// Nothing to read, attempt to flush.
		}

		// After reading from the metrics channel, we check the ticks channel. If there
		// is a tick, flush the in-memory metrics.
		select {
		case <-tick:
			logger.Trace.Println("Tick...")
			for key, metric := range metrics {
				metric.FlushTime = int(time.Now().Unix())
				backendRelay.Relay(metric, logger)
				// @todo: should we delete gauges?
				delete(metrics, key)
			}
		default:
			// Flush interval hasn't passed yet.
		}
	}

}

func checkError(err error, info string, panicOnError bool) {
	if err != nil {
		var errString = "ERROR: " + info + " " + err.Error()
		if panicOnError {
			panic(errString)
		}
	}
}
