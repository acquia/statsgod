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
	"errors"
	"flag"
	"fmt"
	"gopkg.in/yaml.v1"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	// Trace log level.
	Trace *log.Logger
	// Info log level.
	Info *log.Logger
	// Warning log level.
	Warning *log.Logger
	// Error log level.
	Error *log.Logger
)

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

const (
	// AvailableMemory is amount of available memory for the process.
	AvailableMemory = 10 << 20 // 10 MB, for example
	// AverageMemoryPerRequest is how much memory we want to use per request.
	AverageMemoryPerRequest = 10 << 10 // 10 KB
	// MAXREQS is how many requests.
	MAXREQS = AvailableMemory / AverageMemoryPerRequest
	// SeparatorNamespaceValue is the character separating the namespace and value
	// in the metric string.
	SeparatorNamespaceValue = ":"
	// SeparatorValueType is the character separating the value and metric type in
	// the metric string.
	SeparatorValueType = "|"
)

// The channel containing received metric strings.
var parseChannel = make(chan string, MAXREQS)

// The channel containing the Metric objects.
var flushChannel = make(chan *Metric, MAXREQS)

// CLI flags.
var config = flag.String("config", "config.yml", "YAML config file path")
var debug = flag.Bool("debug", false, "Debugging mode")
var host = flag.String("host", "localhost", "Hostname")
var port = flag.Int("port", 8125, "Port")
var graphiteHost = flag.String("graphiteHost", "localhost", "Graphite Hostname")
var graphitePort = flag.Int("graphitePort", 5001, "Graphite Port")
var flushInterval = flag.Duration("flushInterval", 10*time.Second, "Flush time")
var percentile = flag.Int("percentile", 90, "Percentile")

func main() {
	// Load command line options.
	flag.Parse()

	if *debug {
		logInit(os.Stdout, os.Stdout, os.Stdout, os.Stderr)
		Info.Println("Debugging mode enabled")
	} else {
		logInit(ioutil.Discard, os.Stdout, os.Stdout, os.Stderr)
	}

	// Load the YAML config.
	c := loadConfig(*config)
	Info.Printf("Loaded Config: %v", c)

	addr := fmt.Sprintf("%s:%d", *host, *port)
	Info.Printf("Starting stats server on %s", addr)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		checkError(err, "Starting Server", true)
	}

	// Parse the incoming messages and convert to metrics.
	go parseMetrics()

	// Flush the metrics to the remote stats collector.
	go flushMetrics()

	for {
		conn, err := listener.Accept()
		// @todo: handle errors with one client gracefully.
		if err != nil {
			checkError(err, "Accepting Connection", false)
		}
		go handleTcpRequest(conn)
	}
}

func logInit(
	traceHandle io.Writer,
	infoHandle io.Writer,
	warningHandle io.Writer,
	errorHandle io.Writer) {

	Trace = log.New(traceHandle,
		"TRACE: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Info = log.New(infoHandle,
		"INFO: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Warning = log.New(warningHandle,
		"WARNING: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Error = log.New(errorHandle,
		"ERROR: ",
		log.Ldate|log.Ltime|log.Lshortfile)
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

	if m["graphiteHost"] != nil && touchedFlags["graphiteHost"] != 1 {
		*graphiteHost = m["graphiteHost"].(string)
	}

	if m["graphitePort"] != nil && touchedFlags["graphitePort"] != 1 {
		*graphitePort = m["graphitePort"].(int)
	}

	if m["percentile"] != nil && touchedFlags["percentile"] != 1 {
		*percentile = m["percentile"].(int)
	}

	return m
}

// Processes the TCP data as quickly as possible, moving it onto a channel
// and returning immediately. At this phase, we aren't worried about malformed
// strings, we just want to collect them.
func handleTcpRequest(conn net.Conn) {

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
		Warning.Printf("Error processing client message: %s", string(buf))
		return
	}

	conn.Write([]byte(""))
}

// Parses the strings received from clients and creates Metric structures.
func parseMetrics() {

	for {
		// Process the channel as soon as requests come in. If they are valid Metric
		// structures, we move them to a new channel to be flushed on an interval.
		select {
		case metricString := <-parseChannel:
			metric, err := parseMetricString(metricString)
			if err != nil {
				Trace.Printf("Invalid metric: %v", metricString)
				continue
			}
			Trace.Printf("Metric push: %v", metric)
			// Push the metric onto the channel to be aggregated and flushed.
			flushChannel <- metric
		}

	}
}

// Parses a metric string, and if it is properly constructed, create a Metric
// structure. Expects the format [namespace]:[value]|[type]
func parseMetricString(metricString string) (*Metric, error) {
	var metric = new(Metric)

	// First extract the first element which is the namespace.
	split1 := strings.Split(metricString, SeparatorNamespaceValue)
	if len(split1) == 1 {
		// We didn't find the ":" separator.
		return metric, errors.New("Invalid data string")
	}

	// Next split the remaining string wich is the value and type.
	split2 := strings.Split(split1[1], SeparatorValueType)
	if len(split2) == 1 {
		// We didn't find the "|" separator.
		return metric, errors.New("Invalid data string")
	}

	// Locate the metric type.
	metricType, err := shortTypeToLong(strings.TrimSpace(split2[1]))
	if err != nil {
		// We were unable to discern a metric type.
		return metric, errors.New("Invalid data string")
	}

	// Parse the value as a float.
	parsedValue, err := strconv.ParseFloat(split2[0], 32)
	if err != nil {
		// We were unable to find a numeric value.
		return metric, errors.New("Invalid data string")
	}

	// The string was successfully parsed. Convert to a Metric structure.
	metric.key = strings.TrimSpace(split1[0])
	metric.metricType = metricType
	metric.lastValue = float32(parsedValue)
	metric.totalHits = 1

	return metric, nil
}

// Flushes the metrics in-memory to the permanent storage facility. At this
// point we are receiving Metric structures from a channel that need to be
// aggregated by the specified namespace. We do this immediately, then when
// the specified flush interval passes, we send aggregated metrics to storage.
func flushMetrics() {
	// Use a tick channel to determine if a flush message has arrived.
	tick := time.Tick(*flushInterval)
	Info.Printf("Flushing every %v", flushInterval)

	// Internal storage.
	metrics := make(map[string]Metric)

	for {
		// Process the metrics as soon as they arrive on the channel. If nothing has
		// been added during the flush interval duration, continue the loop to allow
		// it to flush the data.
		select {
		case metric := <-flushChannel:
			Trace.Printf("Metric: %v", metric)
			aggregateMetric(metrics, *metric)
		case <-time.After(*flushInterval):
			Trace.Println("Metric Channel Timeout")
			// Nothing to read, attempt to flush.
		}

		// After reading from the metrics channel, we check the ticks channel. If there
		// is a tick, flush the in-memory metrics.
		select {
		case <-tick:
			Trace.Println("Tick...")
			for key, metric := range metrics {
				metric.flushTime = int(time.Now().Unix())
				sendToGraphite(metric)
				// @todo: should we delete gauges?
				delete(metrics, key)
			}
		default:
			// Flush interval hasn't passed yet.
		}
	}

}

// Adds the metric to the specified storage map or aggregates it with
// an existing metric which has the same namespace.
func aggregateMetric(metrics map[string]Metric, metric Metric) {
	existingMetric, metricExists := metrics[metric.key]

	// If the metric exists in the specified map, we either take the last value or
	// sum the values and increment the hit count.
	if metricExists {
		existingMetric.totalHits++

		switch {
		case metric.metricType == "gauge":
			existingMetric.lastValue = metric.lastValue
		case metric.metricType == "counter":
			existingMetric.lastValue += metric.lastValue
		case metric.metricType == "timer":
			existingMetric.lastValue = metric.lastValue
		}

		metrics[metric.key] = existingMetric
	} else {
		metrics[metric.key] = metric
	}

	Trace.Printf("Aggregating %v", metrics)
}

func sendToGraphite(m Metric) {
	stringTime := strconv.Itoa(m.flushTime)
	var gkey string

	defer Info.Println("Done sending to Graphite")

	//Determine why this checkError wasn't working.
	//checkError(err, "Problem sending to graphite", false)

	// @todo: for metrics
	// http://blog.pkhamre.com/2012/07/24/understanding-statsd-and-graphite/
	// Ensure all of the metrics are working correctly.

	if m.metricType == "gauge" {
		gkey = fmt.Sprintf("stats.gauges.%s.avg_value", m.key)
		sendSingleMetricToGraphite(gkey, m.lastValue, stringTime)
	} else if m.metricType == "counter" {
		flushSeconds := time.Duration.Seconds(*flushInterval)
		valuePerSec := m.lastValue / float32(flushSeconds)

		gkey = fmt.Sprintf("stats.%s", m.key)
		sendSingleMetricToGraphite(gkey, valuePerSec, stringTime)

		gkey = fmt.Sprintf("stats_counts.%s", m.key)
		sendSingleMetricToGraphite(gkey, m.lastValue, stringTime)
	}

	sendSingleMetricToGraphite(m.key, m.lastValue, stringTime)

	if m.metricType != "timer" {
		Trace.Println("Not a timer, so skipping additional graphite points")
		return
	}

	// Handle timer specific calls.
	sort.Sort(ByFloat32(m.allValues))
	Trace.Printf("Sorted Vals: %v", m.allValues)

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
	Trace.Printf("Key: %s | Total Vals: %d | Threshold IDX: %d | How many in threshold? %d | Max at threshold: %f", m.key, m.totalHits, thresholdIndex, numInThreshold, maxAtThreshold)

	Trace.Printf("Cumultative Values: %v", cumulativeValues)

	// Take the cumulative at the threshold and divide by the threshold idx.
	meanAtPercentile := cumulativeValues[numInThreshold-1] / float32(numInThreshold)

	gkey = fmt.Sprintf("stats.timers.%s.mean_%d", m.key, *percentile)
	sendSingleMetricToGraphite(gkey, meanAtPercentile, stringTime)

	gkey = fmt.Sprintf("stats.timers.%s.upper_%d", m.key, *percentile)
	sendSingleMetricToGraphite(gkey, maxAtThreshold, stringTime)

	gkey = fmt.Sprintf("stats.timers.%s.sum_%d", m.key, *percentile)
	sendSingleMetricToGraphite(gkey, cumulativeValues[numInThreshold-1], stringTime)
}

// sendSingleMetricToGraphite formats a message and a value and time and sends to Graphite.
func sendSingleMetricToGraphite(key string, v float32, t string) {
	c, err := net.Dial("tcp", fmt.Sprintf("%s:%d", *graphiteHost, *graphitePort))
	if err != nil {
		Error.Println("Could not connect to remote graphite server")
		return
	}

	defer c.Close()

	sv := strconv.FormatFloat(float64(v), 'f', 6, 32)
	payload := fmt.Sprintf("%s %s %s", key, sv, t)
	Trace.Printf("Payload: %v", payload)

	// Send to the connection
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
	return "unknown", errors.New("unknown metric type")
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
