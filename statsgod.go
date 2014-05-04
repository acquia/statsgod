/**
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

// Package statsgod is an experimental implementation of statsd.
package statsgod

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Gauge   (g):  constant metric, repeats this gauge until stats server is restarted
// Counter (c):  increment/decrement a given method
// Timer   (ms): a timer that calculates average, 90% percentile, etc.

// Metric is our main data type.
type Metric struct {
	key         string  // Name of the metric.
	metricType  string  // What type of metric is it (gauge, counter, timer)
	totalHits   int     // Number of times it has been used.
	lastValue   float32 // The last value stored.
	avgValue    float32 // What is the average over all the hits?
	maxValue    float32 // What is the biggest value we've seen?
	minValue    float32 // What is the minimum value we've seen?
	flushTime   int     // What time are we sending Graphite?
	lastFlushed int     // When did we last flush this out?
}

// MetricStore is storage for the metrics with locking.
type MetricStore struct {
	//Map from the kef of the metric to the int value.
	metrics map[string]Metric
	mu      sync.RWMutex
}

const (
	// AvailableMemory is amount of available memory for the process.
	AvailableMemory = 10 << 20 // 10 MB, for example
	// AverageMemoryPerRequest is how much memory we want to use per request.
	AverageMemoryPerRequest = 10 << 10 // 10 KB
	// MAXREQS is how many requests.
	MAXREQS = AvailableMemory / AverageMemoryPerRequest
)

var graphitePipeline = make(chan Metric, MAXREQS)

var debug = flag.Bool("d", false, "Debugging mode")
var host = flag.String("h", "localhost", "Hostname")
var port = flag.String("p", "8125", "Port")

func main() {
	// Load command line options.
	flag.Parse()

	addr := fmt.Sprintf("%s:%s", *host, *port)
	fmt.Println("Starting stats server on ", addr)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		checkError(err, "Starting Server", true)
	}

	var store = NewMetricStore()

	// Every 5 seconds we want to flush the metrics
	go flushMetrics(store)

	// Constantly process background Graphite queue.
	go handleGraphiteQueue(store)

	for {
		conn, err := listener.Accept()
		// TODO: handle errors with one client gracefully.
		if err != nil {
			checkError(err, "Accepting Connection", false)
		}
		go handleRequest(conn, store)
	}
}

func handleRequest(conn net.Conn, store *MetricStore) {
	for {
		var metric, val, validOp string
		buf := make([]byte, 512)
		_, err := conn.Read(buf)
		if err != nil {
			checkError(err, "Reading Connection", false)
			return
		}
		defer conn.Close()

		msg := regexp.MustCompile(`(.*)\:(.*)\|(.*)`)
		bits := msg.FindAllStringSubmatch(string(buf), 1)
		if len(bits) != 0 {
			metric = bits[0][1]
			val = bits[0][2]
			operation := bits[0][3]
			validOp, err = shortTypeToLong(operation)
			if err != nil {
				fmt.Println("Problem handling metric of type: ", operation)
			}
		} else {
			fmt.Println("Error processing client message: ", string(buf))
			return
		}

		// TODO - this float parsing is ugly.
		value, err := strconv.ParseFloat(val, 32)
		checkError(err, "Converting Value", false)

		logger(fmt.Sprintf("(%s) %s => %f", validOp, metric, value))

		store.Set(metric, float32(value))
	}
}

func flushMetrics(store *MetricStore) {
	flushTicker := time.Tick(5e9)

	for {
		select {
		case <-flushTicker:
			fmt.Println("Tick...")
			for index, metric := range store.metrics {
				logger(fmt.Sprintf("[%s] %s: %g", index, metric.key, metric.lastValue))
			}

			for _, metric := range store.metrics {
				flushTime := int(time.Now().Unix())
				metric.flushTime = flushTime
				graphitePipeline <- metric
			}
		}
	}

}

func handleGraphiteQueue(store *MetricStore) {
	for {
		metric := <-graphitePipeline
		go sendToGraphite(metric.key, metric.lastValue, metric.flushTime)
	}
}

func sendToGraphite(key string, val float32, timer int) {
	stringTime := strconv.Itoa(timer)

	strVal := strconv.FormatFloat(float64(val), 'f', 6, 32)
	logger(fmt.Sprintf("Sending to Graphite: %s@%s => %s", key, stringTime, strVal))
	conn, err := net.Dial("tcp", "localhost:5001")
	if err != nil {
		fmt.Println("Could not connect to remote graphite server")
		return
	}

	//Determine why this checkError wasn't working.
	//checkError(err, "Problem sending to graphite", false)

	payload := fmt.Sprintf("%s %s %s", key, string_time, str_val)
	fmt.Fprintf(conn, payload)
	conn.Close()

	logger("Done sending to Graphite")
}

// NewMetricStore Initialize the metric store.
func NewMetricStore() *MetricStore {
	return &MetricStore{metrics: make(map[string]Metric)}
}

// Get will return a metric from inside the store.
func (s *MetricStore) Get(key string) Metric {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m := s.metrics[key]
	return m
}

// Set will store or update a metric.
func (s *MetricStore) Set(key string, val float32) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	m, existingMetric := s.metrics[key]

	if !existingMetric {
		m.key = key
		m.totalHits = 1
		m.lastValue = val
		m.avgValue = val
		m.minValue = val
		m.maxValue = val
	} else {
		if val < m.minValue {
			m.minValue = val
		}
		if val > m.maxValue {
			m.maxValue = val
		}
		m.avgValue = ((float32(m.totalHits) * m.avgValue) + val) / float32(m.totalHits+1)
		m.totalHits++
		m.lastValue = val

	}
	s.metrics[key] = m

	return false
}

func shortTypeToLong(short string) (string, error) {
	switch {
	case "c" == short:
		return "counter", nil
	case "g" == short:
		return "gauge", nil
	case "ms" == short:
		return "timer", nil
	}
	return "", errors.New("unknown metric type")
}

func logger(msg string) {
	fmt.Println(msg)
}

// Split a string by a separate and get the left and right.
func splitString(raw, separator string) (string, string) {
	split := strings.Split(raw, separator)
	return split[0], split[1]
}

func checkError(err error, info string, panicOnError bool) {
	if err != nil {
		var errString = "ERROR: " + info + " " + err.Error()
		if panicOnError {
			panic(errString)
		}
	}
}
