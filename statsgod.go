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

// Package main: statsgod is an experimental implementation of statsd.
package main

import (
	"errors"
	"flag"
	"fmt"
	"math"
	"net"
	"regexp"
	"sort"
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
	key         string    // Name of the metric.
	metricType  string    // What type of metric is it (gauge, counter, timer)
	totalHits   int       // Number of times it has been used.
	lastValue   float32   // The last value stored.
	allValues   []float32 // All of the values.
	flushTime   int       // What time are we sending Graphite?
	lastFlushed int       // When did we last flush this out?
}

// MetricStore is storage for the metrics with locking.
type MetricStore struct {
	//Map from the key of the metric to the int value.
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
var flushTime = flag.Duration("t", 5*time.Second, "Flush time")
var percentile = flag.Int("e", 90, "Percentile")

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

	// Every X seconds we want to flush the metrics
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
		var metric, val, metricType string
		buf := make([]byte, 512)
		_, err := conn.Read(buf)
		if err != nil {
			checkError(err, "Reading Connection", false)
			return
		}
		defer conn.Close()

		logger(fmt.Sprintf("Got from client: %s", buf))

		msg := regexp.MustCompile(`(.*)\:(.*)\|(.*)`)
		bits := msg.FindAllStringSubmatch(string(buf), 1)
		if len(bits) != 0 {
			metric = bits[0][1]
			val = bits[0][2]
			tmpMetricType := bits[0][3]
			tmpMetricType = strings.TrimSpace(tmpMetricType)
			tmpMetricType = strings.Trim(tmpMetricType, "\x00")
			metricType, err = shortTypeToLong(tmpMetricType)
			if err != nil {
				fmt.Println("Problem handling metric of type: ", tmpMetricType)
			}
		} else {
			fmt.Println("Error processing client message: ", string(buf))
			return
		}

		// TODO - this float parsing is ugly.
		value, err := strconv.ParseFloat(val, 32)
		checkError(err, "Converting Value", false)

		logger(fmt.Sprintf("(%s) %s => %f", metricType, metric, value))

		store.Set(metric, metricType, float32(value))
	}
}

func flushMetrics(store *MetricStore) {
	flushTicker := time.Tick(*flushTime)
	logger(fmt.Sprintf("Flushing every %s", *flushTime))

	for {
		select {
		case <-flushTicker:
			fmt.Println("Tick...")
			for index, metric := range store.metrics {
				logger(fmt.Sprintf("%s (%s) => %g %v", index, metric.metricType, metric.lastValue, metric.allValues))
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
		go sendToGraphite(metric)
		if metric.metricType != "gauge" {
			delete(store.metrics, metric.key)
		}
	}
}

func sendToGraphite(m Metric) {
	stringTime := strconv.Itoa(m.flushTime)
	var gkey string

	defer logger("Done sending to Graphite")

	//Determine why this checkError wasn't working.
	//checkError(err, "Problem sending to graphite", false)

	// TODO for metrics
	// http://blog.pkhamre.com/2012/07/24/understanding-statsd-and-graphite/
	// Ensure all of the metrics are working correctly.

	if m.metricType == "gauge" {
		gkey = fmt.Sprintf("stats.gauges.%s.avg_value", m.key)
		sendSingleMetricToGraphite(gkey, m.lastValue, stringTime)
	} else if m.metricType == "counter" {
		flushSeconds := time.Duration.Seconds(*flushTime)
		valuePerSec := m.lastValue / float32(flushSeconds)

		gkey = fmt.Sprintf("stats.%s", m.key)
		sendSingleMetricToGraphite(gkey, valuePerSec, stringTime)

		gkey = fmt.Sprintf("stats_counts.%s", m.key)
		sendSingleMetricToGraphite(gkey, m.lastValue, stringTime)
	}

	sendSingleMetricToGraphite(m.key, m.lastValue, stringTime)

	if m.metricType != "timer" {
		logger("Not a timer, so skipping additional graphite points")
		return
	}

	// Handle timer specific calls.
	sort.Sort(ByFloat32(m.allValues))
	logger(fmt.Sprintf("Sorted Vals: %v", m.allValues))

	// Calculate the math values for the timer.
	minValue := m.allValues[0]
	maxValue := m.allValues[len(m.allValues)-1]

	sum := float32(0)
	cumulativeValues := []float32{minValue}
	for idx, value := range m.allValues {
		sum += value

		if idx != 0 {
			cumulativeValues = append(cumulativeValues, cumulativeValues[idx-1]+value)
		}
	}
	avgValue := sum / float32(m.totalHits)

	gkey = fmt.Sprintf("stats.timers.%s.avg_value", m.key)
	sendSingleMetricToGraphite(gkey, avgValue, stringTime)

	gkey = fmt.Sprintf("stats.timers.%s.max_value", m.key)
	sendSingleMetricToGraphite(gkey, maxValue, stringTime)

	gkey = fmt.Sprintf("stats.timers.%s.min_value", m.key)
	sendSingleMetricToGraphite(gkey, minValue, stringTime)
	// All of the percentile based value calculations.

	thresholdIndex := int(math.Floor((((100 - float64(*percentile)) / 100) * float64(m.totalHits)) + 0.5))
	numInThreshold := m.totalHits - thresholdIndex

	maxAtThreshold := m.allValues[numInThreshold-1]
	logger(fmt.Sprintf("Key: %s | Total Vals: %d | Threshold IDX: %d | How many in threshold? %d | Max at threshold: %f", m.key, m.totalHits, thresholdIndex, numInThreshold, maxAtThreshold))

	logger(fmt.Sprintf("Cumultative Values: %v", cumulativeValues))

	// Take the cumulative at the threshold and divide by the threshold idx.
	meanAtPercentile := cumulativeValues[numInThreshold-1] / float32(numInThreshold)

	gkey = fmt.Sprintf("stats.timers.%s.mean_%d", m.key, *percentile)
	sendSingleMetricToGraphite(gkey, meanAtPercentile, stringTime)

	gkey = fmt.Sprintf("stats.timers.%s.upper_%d", m.key, *percentile)
	sendSingleMetricToGraphite(gkey, maxAtThreshold, stringTime)

	gkey = fmt.Sprintf("stats.timers.%s.sum_%d", m.key, *percentile)
	sendSingleMetricToGraphite(gkey, cumulativeValues[numInThreshold-1], stringTime)
}

// NewMetricStore Initialize the metric store.
func NewMetricStore() *MetricStore {
	return &MetricStore{metrics: make(map[string]Metric)}
}

// Get will return a metric from inside the store.
func (s *MetricStore) Get(key string) Metric {
	// TODO - investigate this never running. NOTE: Set doesn't run Get.
	s.mu.RLock()
	defer s.mu.RUnlock()
	m := s.metrics[key]
	return m
}

// Set will store or update a metric.
func (s *MetricStore) Set(key string, metricType string, val float32) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	m, existingMetric := s.metrics[key]

	if !existingMetric {
		m.key = key
		m.totalHits = 1
		m.lastValue = val
		m.metricType = metricType

		switch {
		case metricType == "gauge":
		case metricType == "counter":
		case metricType == "timer":
		}
	} else {
		m.totalHits++

		switch {
		case metricType == "gauge":
			m.lastValue = val
		case metricType == "counter":
			m.lastValue += val
		case metricType == "timer":
			m.lastValue = val
		}

	}

	// TODO: should we bother trackin this for counters?
	m.allValues = append(m.allValues, val)
	s.metrics[key] = m

	return false
}

// sendSingleMetricToGraphite formats a message and a value and time and sends to Graphite.
func sendSingleMetricToGraphite(key string, v float32, t string) {
	c, err := net.Dial("tcp", "localhost:5001")
	if err != nil {
		fmt.Println("Could not connect to remote graphite server")
		return
	}

	defer c.Close()

	sv := strconv.FormatFloat(float64(v), 'f', 6, 32)
	payload := fmt.Sprintf("%s %s %s", key, sv, t)
	logger(payload)
	fmt.Fprintf(c, fmt.Sprintf("%s %v %s\n", key, sv, t))
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

// ByFloat32 implements sort.Interface for []Float32.
type ByFloat32 []float32

func (a ByFloat32) Len() int           { return len(a) }
func (a ByFloat32) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByFloat32) Less(i, j int) bool { return a[i] < a[j] }

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
