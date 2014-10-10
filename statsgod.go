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
	"gopkg.in/yaml.v1"
	"io/ioutil"
	"net"
	"os"
	"strings"
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

// CLI flags.
var config = flag.String("config", "config.yml", "YAML config file path")
var debug = flag.Bool("debug", false, "Debugging mode")
var host = flag.String("host", "localhost", "Hostname")
var port = flag.Int("port", 8125, "Port")
var carbonHost = flag.String("carbonHost", "localhost", "Carbon Hostname")
var carbonPort = flag.Int("carbonPort", 5001, "Carbon Port")
var relayConcurrency = flag.Int("relayConcurrency", 1, "Simultaneous Relay Connections")
var relayTimeout = flag.Duration("relayTimeout", 20*time.Second, "Socket timeout to carbon relay.")
var flushInterval = flag.Duration("flushInterval", 10*time.Second, "Flush time")
var percentile = flag.Int("percentile", 90, "Percentile")
var relay = flag.String("relay", "carbon", "Relay type, one of 'carbon' or 'mock'")

var backendRelay statsgod.MetricRelay

func main() {
	// Load command line options.
	flag.Parse()

	var logger statsgod.Logger

	if *debug {
		logger = *statsgod.CreateLogger(os.Stdout, os.Stdout, os.Stdout, os.Stderr)
		logger.Info.Println("Debugging mode enabled")
	} else {
		logger = *statsgod.CreateLogger(ioutil.Discard, os.Stdout, os.Stdout, os.Stderr)
	}

	// Load the YAML config.
	c := loadConfig(*config)
	logger.Info.Printf("Loaded Config: %v", c)

	// Set up the backend relay.
	switch *relay {
	case "carbon":
		// Create a relay to carbon.
		relay := statsgod.CreateRelay("carbon").(*statsgod.CarbonRelay)
		relay.FlushInterval = *flushInterval
		relay.Percentile = *percentile
		// Create a connection pool for the relay to use.
		pool, err := statsgod.CreateConnectionPool(*relayConcurrency, *carbonHost, *carbonPort, *relayTimeout, logger)
		checkError(err, "Creating connection pool", true)
		relay.ConnectionPool = pool
		backendRelay = statsgod.MetricRelay(relay)
		logger.Info.Println("Relaying metrics to carbon backend")
	default:
		relay := statsgod.CreateRelay("mock").(*statsgod.MockRelay)
		backendRelay = statsgod.MetricRelay(relay)
		logger.Info.Println("Relaying metrics to mock backend")
	}

	addr := fmt.Sprintf("%s:%d", *host, *port)
	logger.Info.Printf("Starting stats server on %s", addr)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		checkError(err, "Starting Server", true)
	}

	// Parse the incoming messages and convert to metrics.
	go parseMetrics(logger)

	// Flush the metrics to the remote stats collector.
	for i := 0; i < *relayConcurrency; i++ {
		go flushMetrics(logger)
	}

	for {
		conn, err := listener.Accept()
		// @todo: handle errors with one client gracefully.
		if err != nil {
			checkError(err, "Accepting Connection", false)
		}
		go handleTcpRequest(conn, logger)
	}
}

func loadConfig(c string) map[interface{}]interface{} {
	m := make(map[interface{}]interface{})

	contents, err := ioutil.ReadFile(c)
	checkError(err, "Config file could not be read", true)

	err = yaml.Unmarshal([]byte(contents), &m)
	checkError(err, "YAML error processing config file", true)

	if m["debug"] != nil {
		*debug = m["debug"].(bool)
	}

	touchedFlags := make(map[string]int)
	flag.Visit(
		func(f *flag.Flag) {
			touchedFlags[f.Name] = 1
		})

	if m["flushInterval"] != nil && touchedFlags["flushInterval"] != 1 {
		ft, err := time.ParseDuration(m["flushInterval"].(string))
		checkError(err, "Could not parse flushInterval", true)
		*flushInterval = ft
	}

	if m["host"] != nil && touchedFlags["host"] != 1 {
		*host = m["host"].(string)
	}

	if m["port"] != nil && touchedFlags["port"] != 1 {
		*port = m["port"].(int)
	}

	if m["carbonHost"] != nil && touchedFlags["carbonHost"] != 1 {
		*carbonHost = m["carbonHost"].(string)
	}

	if m["carbonPort"] != nil && touchedFlags["carbonPort"] != 1 {
		*carbonPort = m["carbonPort"].(int)
	}

	if m["relayConcurrency"] != nil && touchedFlags["relayConcurrency"] != 1 {
		*relayConcurrency = m["relayConcurrency"].(int)
	}

	if m["relayTimeout"] != nil && touchedFlags["relayTimeout"] != 1 {
		rt, err := time.ParseDuration(m["relayTimeout"].(string))
		checkError(err, "Could not parse relayTimeout", true)
		*relayTimeout = rt
	}

	if m["percentile"] != nil && touchedFlags["percentile"] != 1 {
		*percentile = m["percentile"].(int)
	}

	if m["relay"] != nil && touchedFlags["relay"] != 1 {
		*relay = m["relay"].(string)
	}

	return m
}

// Processes the TCP data as quickly as possible, moving it onto a channel
// and returning immediately. At this phase, we aren't worried about malformed
// strings, we just want to collect them.
func handleTcpRequest(conn net.Conn, logger statsgod.Logger) {

	// Read the data from the connection.
	buf := make([]byte, 512)
	_, err := conn.Read(buf)
	if err != nil {
		checkError(err, "Reading Connection", false)
		return
	}
	defer conn.Close()

	// As long as we have some data, submit it for processing Don't
	// validate or parse it yet, just get the message on the channel asap.
	// so that we can free up the connection.
	if len(string(buf)) != 0 {
		parseChannel <- strings.TrimSpace(strings.Trim(string(buf), "\x00"))
	} else {
		logger.Warning.Printf("Error processing client message: %s", string(buf))
		return
	}

	conn.Write([]byte(""))
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
				logger.Trace.Printf("Invalid metric: %v", metricString)
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
func flushMetrics(logger statsgod.Logger) {
	// Use a tick channel to determine if a flush message has arrived.
	tick := time.Tick(*flushInterval)
	logger.Info.Printf("Flushing every %v", flushInterval)

	// Internal storage.
	metrics := make(map[string]statsgod.Metric)

	for {
		// Process the metrics as soon as they arrive on the channel. If nothing has
		// been added during the flush interval duration, continue the loop to allow
		// it to flush the data.
		select {
		case metric := <-flushChannel:
			statsgod.AggregateMetric(metrics, *metric)
			logger.Trace.Printf("Metric: %v", metrics[metric.Key])
		case <-time.After(*flushInterval):
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
