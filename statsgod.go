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
 * Set     (s): a count of unique values sent during a flush period.
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
	"regexp"
	"runtime"
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
	// Megabyte represents the number of bytes in a megabyte.
	Megabyte = 1048576
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

// Track the host of this program.
var hostname string

func main() {
	// Load command line options.
	flag.Parse()

	var logger statsgod.Logger

	// Load the YAML config.
	var config, _ = statsgod.LoadConfig(*configFile)

	var hostnameErr error
	hostname, hostnameErr = os.Hostname()
	if hostnameErr != nil {
		hostname = "unknown"
	} else {
		re := regexp.MustCompile("[^a-zA-Z0-9]")
		hostname = re.ReplaceAllString(hostname, "-")
	}

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
	case statsgod.RelayTypeCarbon:
		// Create a relay to carbon.
		relay := statsgod.CreateRelay(statsgod.RelayTypeCarbon).(*statsgod.CarbonRelay)
		relay.FlushInterval = config.Relay.Flush
		relay.Percentile = config.Stats.Percentile
		// Create a connection pool for the relay to use.
		pool, err := statsgod.CreateConnectionPool(config.Relay.Concurrency, fmt.Sprintf("%s:%d", config.Carbon.Host, config.Carbon.Port), statsgod.ConnPoolTypeTcp, config.Relay.Timeout, logger)
		checkError(err, "Creating connection pool", true)
		relay.ConnectionPool = pool
		backendRelay = statsgod.MetricRelay(relay)
		logger.Info.Println("Relaying metrics to carbon backend")
	case statsgod.RelayTypeMock:
		fallthrough
	default:
		relay := statsgod.CreateRelay(statsgod.RelayTypeMock).(*statsgod.MockRelay)
		relay.FlushInterval = config.Relay.Flush
		relay.Percentile = config.Stats.Percentile
		backendRelay = statsgod.MetricRelay(relay)
		logger.Info.Println("Relaying metrics to mock backend")
	}

	// Set up the authentication.
	var auth statsgod.Auth
	switch config.Service.Auth {
	case statsgod.AuthTypeConfigToken:
		tokenAuth := statsgod.CreateAuth(statsgod.AuthTypeConfigToken).(*statsgod.AuthConfigToken)
		tokenAuth.Tokens = config.Service.Tokens
		auth = tokenAuth
	case statsgod.AuthTypeNone:
		fallthrough
	default:
		auth = statsgod.CreateAuth(statsgod.AuthTypeNone).(*statsgod.AuthNone)
	}

	// Parse the incoming messages and convert to metrics.
	go parseMetrics(logger, auth)

	// Flush the metrics to the remote stats collector.
	for i := 0; i < config.Relay.Concurrency; i++ {
		go flushMetrics(logger, config)
	}

	tcpAddr := fmt.Sprintf("%s:%d", config.Connection.Tcp.Host, config.Connection.Tcp.Port)
	socketTcp := statsgod.CreateSocket(statsgod.SocketTypeTcp, tcpAddr).(*statsgod.SocketTcp)
	go socketTcp.Listen(parseChannel, logger, config)

	udpAddr := fmt.Sprintf("%s:%d", config.Connection.Udp.Host, config.Connection.Udp.Port)
	socketUdp := statsgod.CreateSocket(statsgod.SocketTypeUdp, udpAddr).(*statsgod.SocketUdp)
	go socketUdp.Listen(parseChannel, logger, config)

	socketUnix := statsgod.CreateSocket(statsgod.SocketTypeUnix, config.Connection.Unix.File).(*statsgod.SocketUnix)
	go socketUnix.Listen(parseChannel, logger, config)

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
		finishChannel <- 1
	}()

	select {
	case <-finishChannel:
		logger.Info.Println("Exiting program.")
		socketTcp.Close(logger)
		socketUdp.Close(logger)
		socketUnix.Close(logger)
	}
}

// Parses the strings received from clients and creates Metric structures.
func parseMetrics(logger statsgod.Logger, auth statsgod.Auth) {

	var authOk bool
	var authErr error

	for {
		// Process the channel as soon as requests come in. If they are valid Metric
		// structures, we move them to a new channel to be flushed on an interval.
		select {
		case metricString := <-parseChannel:
			// Authenticate the metric.
			authOk, authErr = auth.Authenticate(&metricString)
			if authErr != nil || !authOk {
				logger.Error.Printf("Auth Error: %v, %s", authOk, authErr)
				continue
			}

			metric, err := statsgod.ParseMetricString(metricString)
			if err != nil {
				logger.Error.Printf("Invalid metric: %s, %s", metricString, err)
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

	// Track the flush cycle metrics.
	var flushStart time.Time
	var flushStop time.Time
	var flushCount int

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

			// Prepare the runtime metrics.
			prepareRuntimeMetrics(metrics, config)

			// Time and flush the received metrics.
			flushCount = len(metrics)
			flushStart = time.Now()
			flushAllMetrics(metrics, logger)
			flushStop = time.Now()

			// Prepare and flush the internal metrics.
			prepareFlushMetrics(metrics, config, flushStart, flushStop, flushCount)
			flushAllMetrics(metrics, logger)
		default:
			// Flush interval hasn't passed yet.
		}
	}
}

// flushAllMetrics is a helper to iterate over a Metric map and flush all to the relay.
func flushAllMetrics(metrics map[string]statsgod.Metric, logger statsgod.Logger) {
	for key, metric := range metrics {
		metric.FlushTime = int(time.Now().Unix())
		backendRelay.Relay(metric, logger)
		delete(metrics, key)
	}
}

// prepareRuntimeMetrics creates key runtime metrics to monitor the health of the system.
func prepareRuntimeMetrics(metrics map[string]statsgod.Metric, config statsgod.ConfigValues) {
	if config.Debug.Relay {
		memStats := &runtime.MemStats{}
		runtime.ReadMemStats(memStats)

		// Prepare a metric for the memory allocated.
		heapAllocKey := fmt.Sprintf("statsgod.%s.runtime.memory.heapalloc", hostname)
		heapAllocValue := float64(memStats.HeapAlloc) / float64(Megabyte)
		heapAllocMetric := statsgod.CreateSimpleMetric(heapAllocKey, heapAllocValue, statsgod.MetricTypeGauge)
		metrics[heapAllocMetric.Key] = *heapAllocMetric

		// Prepare a metric for the memory allocated that is still in use.
		allocKey := fmt.Sprintf("statsgod.%s.runtime.memory.alloc", hostname)
		allocValue := float64(memStats.Alloc) / float64(Megabyte)
		allocMetric := statsgod.CreateSimpleMetric(allocKey, allocValue, statsgod.MetricTypeGauge)
		metrics[allocMetric.Key] = *allocMetric

		// Prepare a metric for the memory obtained from the system.
		sysKey := fmt.Sprintf("statsgod.%s.runtime.memory.sys", hostname)
		sysValue := float64(memStats.Sys) / float64(Megabyte)
		sysMetric := statsgod.CreateSimpleMetric(sysKey, sysValue, statsgod.MetricTypeGauge)
		metrics[sysMetric.Key] = *sysMetric
	}
}

// prepareFlushMetrics creates metrics that represent the speed and size of the flushes.
func prepareFlushMetrics(metrics map[string]statsgod.Metric, config statsgod.ConfigValues, flushStart time.Time, flushStop time.Time, flushCount int) {
	if config.Debug.Relay {
		// Prepare the duration metric.
		durationKey := fmt.Sprintf("statsgod.%s.flush.duration", hostname)
		durationValue := float64(int64(flushStop.Sub(flushStart).Nanoseconds()) / int64(time.Millisecond))
		durationMetric := statsgod.CreateSimpleMetric(durationKey, durationValue, statsgod.MetricTypeTimer)
		metrics[durationMetric.Key] = *durationMetric

		// Prepare the counter metric.
		countKey := fmt.Sprintf("statsgod.%s.flush.count", hostname)
		countMetric := statsgod.CreateSimpleMetric(countKey, float64(flushCount), statsgod.MetricTypeGauge)
		metrics[countMetric.Key] = *countMetric
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
