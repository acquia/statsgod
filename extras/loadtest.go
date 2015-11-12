/**
 * Copyright 2015 Acquia, Inc.
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

package main

import (
	"flag"
	"fmt"
	"github.com/acquia/statsgod/statsgod"
	"github.com/jmcvetta/randutil"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"time"
)

var statsHost = flag.String("statsHost", "localhost", "Statsgod Hostname.")
var statsPortTcp = flag.Int("statsPortTcp", 8125, "Statsgod TCP Port.")
var statsPortUdp = flag.Int("statsPortUdp", 8126, "Statsgod UDP Port.")
var statsSock = flag.String("statsSock", "/var/run/statsgod/statsgod.sock", "The location of the socket file.")
var statsPoolCount = flag.Int("statsPoolCount", 5, "How many active connections to maintain.")
var numMetrics = flag.Int("numMetrics", 10, "Number of metrics per thread.")
var flushTime = flag.Duration("flushTime", 1*time.Second, "How frequently to send metrics.")
var concurrency = flag.Int("concurrency", 1, "How many concurrent generators to run.")
var runTime = flag.Duration("runTime", 30*time.Second, "How long to run the test.")
var connType = flag.Int("connType", 0, "0 for all, 1 for TCP, 2 for TCP Pool, 3 for UDP, 4 for Unix, 5 for Unix Pool.")
var logSent = flag.Bool("logSent", false, "Log each metric sent.")
var token = flag.String("token", "", "An auth token to prepend to the metric string.")

// Metric is our main data type.
type Metric struct {
	key            string // Name of the metric.
	metricType     string // What type of metric is it (gauge, counter, set, timer)
	metricValue    int    // The value of the metric to send.
	connectionType int    // Whether we are connecting TCP, UDP or Unix.
}

// Connection Types enum.
const (
	ConnectionTypeTcp      = 1
	ConnectionTypeTcpPool  = 2
	ConnectionTypeUdp      = 3
	ConnectionTypeUnix     = 4
	ConnectionTypeUnixPool = 5
)

// Track connections/errors.
var (
	connectionCountTcpPool  int
	connectionCountTcp      int
	connectionCountUdp      int
	connectionCountUnixPool int
	connectionCountUnix     int
	connectionErrorTcpPool  int
	connectionErrorTcp      int
	connectionErrorUdp      int
	connectionErrorUnixPool int
	connectionErrorUnix     int
	tcpPool                 *statsgod.ConnectionPool
	unixPool                *statsgod.ConnectionPool
)

var logger = *statsgod.CreateLogger(ioutil.Discard, os.Stdout, os.Stdout, os.Stderr)

func main() {
	flag.Parse()
	fmt.Printf("Starting test with %d metrics on %d concurrent threads for %s.\n", *numMetrics, *concurrency, *runTime)

	tcpPool, _ = statsgod.CreateConnectionPool(*statsPoolCount, fmt.Sprintf("%s:%d", *statsHost, *statsPortTcp), statsgod.ConnPoolTypeTcp, 10*time.Second, logger)
	unixPool, _ = statsgod.CreateConnectionPool(*statsPoolCount, *statsSock, statsgod.ConnPoolTypeUnix, 10*time.Second, logger)

	startTime := time.Now()

	finishChannel := make(chan int)
	flushChannel := make(chan Metric)

	// If the user specified an auth token, construct it here.
	metricToken := ""
	if *token != "" {
		metricToken = *token + "."
	}

	// Establish threads to send data concurrently.
	for i := 0; i < *concurrency; i++ {
		var store = make([]Metric, 0)
		store = generateMetricNames(*numMetrics, store, metricToken)
		go sendTestMetrics(store, flushChannel, finishChannel)
		go flushMetrics(flushChannel, finishChannel)
	}

	finishTicker := time.Tick(*runTime)
	flushTicker := time.Tick(*flushTime)
runloop:
	for {
		select {
		case <-flushTicker:
			if !*logSent {
				fmt.Printf("-")
			}
		case <-finishTicker:
			break runloop
		}
	}
	close(finishChannel)

	totalTime := time.Since(startTime)

	// Print the output.
	printReqPerSecond("TCP", connectionCountTcp, connectionErrorTcp, totalTime)
	printReqPerSecond("TCP (pool)", connectionCountTcpPool, connectionErrorTcpPool, totalTime)
	printReqPerSecond("UDP", connectionCountUdp, connectionErrorUdp, totalTime)
	printReqPerSecond("Unix", connectionCountUnix, connectionErrorUnix, totalTime)
	printReqPerSecond("Unix (pool)", connectionCountUnixPool, connectionErrorUnixPool, totalTime)
	totalCount := connectionCountTcpPool + connectionCountTcp + connectionCountUdp + connectionCountUnixPool + connectionCountUnix
	totalError := connectionErrorTcpPool + connectionErrorTcp + connectionErrorUdp + connectionErrorUnixPool + connectionErrorUnix
	printReqPerSecond("Total", totalCount, totalError, totalTime)
}

// printReqPerSecond prints the results in a human-readable format.
func printReqPerSecond(title string, total int, errors int, runTime time.Duration) {
	rate := (total - errors) / (int(runTime / time.Second))
	errRate := float64(errors) / float64(total)
	fmt.Printf("\n%s:\n-Connections: %d\n-Errors: %d\n-Error Rate: %.6f\n-Req/sec: %d\n", title, total, errors, errRate, rate)
}

// sendTestMetrics reads from the store and send all metrics to the flush channel on a timer.
func sendTestMetrics(store []Metric, flushChannel chan Metric, finishChannel chan int) {
	flushTicker := time.Tick(*flushTime)

	for {
		select {
		case <-flushTicker:
			for _, metric := range store {
				flushChannel <- metric
			}
		case <-finishChannel:
			return
		}
	}

}

// flushMetrics continually reads from the flush channel and sends metrics.
func flushMetrics(flushChannel chan Metric, finishChannel chan int) {

	for {
		select {
		case metric := <-flushChannel:
			sendMetricToStats(metric)
		case <-finishChannel:
			return
		}
	}
}

// generateMetricNames generates a specified number of random metric types.
func generateMetricNames(numMetrics int, store []Metric, metricToken string) []Metric {
	metricTypes := []string{
		"c",
		"g",
		"ms",
		"s",
	}

	rand.Seed(time.Now().UnixNano())
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < numMetrics; i++ {
		newMetricName, _ := randutil.String(20, randutil.Alphabet)
		newMetricNS := fmt.Sprintf("%sstatsgod.test.%s", metricToken, newMetricName)
		newMetricCT := 1
		if *connType > 0 && *connType < 6 {
			newMetricCT = *connType
		} else {
			newMetricCT, _ = randutil.IntRange(1, 6)
		}

		metricType := metricTypes[r.Intn(len(metricTypes))]

		store = append(store, Metric{
			key:            newMetricNS,
			metricType:     metricType,
			metricValue:    0,
			connectionType: newMetricCT,
		})
	}

	return store
}

// sendMetricToStats formats a message and a value and time and sends to statsgod.
func sendMetricToStats(metric Metric) {

	rand.Seed(time.Now().UnixNano())
	var metricValue int
	switch metric.metricType {
	case "c":
		metricValue = 1
	case "g":
		metricValue = rand.Intn(100)
	case "ms":
		metricValue = rand.Intn(1000)
	case "s":
		metricValue = rand.Intn(100)
	}
	stringValue := fmt.Sprintf("%s:%d|%s", metric.key, metricValue, metric.metricType)
	// Send to the designated connection
	switch metric.connectionType {
	case ConnectionTypeTcp:
		c, err := net.Dial("tcp", fmt.Sprintf("%s:%d", *statsHost, *statsPortTcp))
		connectionCountTcp++
		if err == nil {
			c.Write([]byte(stringValue + "\n"))
			logSentMetric(stringValue)
			defer c.Close()
		} else {
			connectionErrorTcp++
			return
		}
	case ConnectionTypeTcpPool:
		c, err := tcpPool.GetConnection(logger)
		connectionCountTcpPool++
		if err == nil {
			_, err := c.Write([]byte(stringValue + "\n"))
			logSentMetric(stringValue)
			if err != nil {
				connectionErrorTcpPool++
				defer tcpPool.ReleaseConnection(c, true, logger)
			} else {
				defer tcpPool.ReleaseConnection(c, false, logger)
			}
		} else {
			connectionErrorTcp++
			return
		}
	case ConnectionTypeUdp:
		c, err := net.Dial("udp", fmt.Sprintf("%s:%d", *statsHost, *statsPortUdp))
		connectionCountUdp++
		if err == nil {
			c.Write([]byte(stringValue))
			logSentMetric(stringValue)
			defer c.Close()
		} else {
			connectionErrorUdp++
			return
		}
	case ConnectionTypeUnix:
		c, err := net.Dial("unix", *statsSock)
		connectionCountUnix++
		if err == nil {
			c.Write([]byte(stringValue))
			logSentMetric(stringValue)
			defer c.Close()
		} else {
			connectionErrorUnix++
			return
		}
	case ConnectionTypeUnixPool:
		c, err := unixPool.GetConnection(logger)
		connectionCountUnixPool++
		if err == nil {
			_, err := c.Write([]byte(stringValue + "\n"))
			logSentMetric(stringValue)
			if err != nil {
				connectionErrorUnixPool++
				defer unixPool.ReleaseConnection(c, true, logger)
			} else {
				defer unixPool.ReleaseConnection(c, false, logger)
			}
		} else {
			connectionErrorUnixPool++
			return
		}
	}
}

// logSentMetric will log the metric string for QA spot checking.
func logSentMetric(metric string) {
	if *logSent {
		fmt.Printf("%s\n", metric)
	}
}
