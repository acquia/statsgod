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

// Package statsgod - This library handles the relaying of data to a backend
// storage. This backend could be a webservice like carbon or a filesystem
// or a mock for testing. All backend implementations should conform to the
// MetricRelay interface.
package statsgod

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"time"
)

// MetricRelay defines the interface for a back end implementation.
type MetricRelay interface {
	Relay(metric Metric, logger Logger)
}

// CreateRelay is a factory for instantiating remote relays.
func CreateRelay(relayType string) MetricRelay {
	switch relayType {
	case "carbon":
		return new(CarbonRelay)
	case "mock":
		return new(MockRelay)
	}
	return new(MockRelay)
}

// CarbonRelay implements MetricRelay.
type CarbonRelay struct {
	FlushInterval  time.Duration
	Percentile     int
	ConnectionPool *ConnectionPool
}

// Relay implements MetricRelay::Relay().
func (c CarbonRelay) Relay(metric Metric, logger Logger) {
	sendToGraphite(metric, c, logger)
}

// sendToGraphite sends the metric to the specified host/port.
func sendToGraphite(m Metric, c CarbonRelay, logger Logger) {
	stringTime := strconv.Itoa(m.FlushTime)
	var gkey string

	defer logger.Info.Println("Done sending to Graphite")

	// @todo: for metrics
	// http://blog.pkhamre.com/2012/07/24/understanding-statsd-and-graphite/
	// Ensure all of the metrics are working correctly.

	if m.MetricType == "gauge" {
		gkey = fmt.Sprintf("stats.gauges.%s.avg_value", m.Key)
		sendSingleMetricToGraphite(gkey, m.LastValue, stringTime, true, c, logger)
	} else if m.MetricType == "counter" {
		flushSeconds := time.Duration.Seconds(c.FlushInterval)
		valuePerSec := m.LastValue / float32(flushSeconds)

		gkey = fmt.Sprintf("stats.%s", m.Key)
		sendSingleMetricToGraphite(gkey, valuePerSec, stringTime, true, c, logger)

		gkey = fmt.Sprintf("stats_counts.%s", m.Key)
		sendSingleMetricToGraphite(gkey, m.LastValue, stringTime, true, c, logger)
	}

	sendSingleMetricToGraphite(m.Key, m.LastValue, stringTime, true, c, logger)

	if m.MetricType != "timer" {
		logger.Trace.Println("Not a timer, so skipping additional graphite points")
		return
	}

	// Handle timer specific calls.
	sort.Sort(ByFloat32(m.AllValues))
	logger.Trace.Printf("Sorted Vals: %v", m.AllValues)

	// Calculate the math values for the timer.
	minValue := m.AllValues[0]
	maxValue := m.AllValues[len(m.AllValues)-1]

	sum := float32(0)
	cumulativeValues := []float32{minValue}
	for idx, value := range m.AllValues {
		sum += value

		if idx != 0 {
			cumulativeValues = append(cumulativeValues, cumulativeValues[idx-1]+value)
		}
	}
	avgValue := sum / float32(m.TotalHits)

	gkey = fmt.Sprintf("stats.timers.%s.avg_value", m.Key)
	sendSingleMetricToGraphite(gkey, avgValue, stringTime, true, c, logger)

	gkey = fmt.Sprintf("stats.timers.%s.max_value", m.Key)
	sendSingleMetricToGraphite(gkey, maxValue, stringTime, true, c, logger)

	gkey = fmt.Sprintf("stats.timers.%s.min_value", m.Key)
	sendSingleMetricToGraphite(gkey, minValue, stringTime, true, c, logger)
	// All of the percentile based value calculations.

	thresholdIndex := int(math.Floor((((100 - float64(c.Percentile)) / 100) * float64(m.TotalHits)) + 0.5))
	numInThreshold := m.TotalHits - thresholdIndex

	maxAtThreshold := m.AllValues[numInThreshold-1]
	logger.Trace.Printf("Key: %s | Total Vals: %d | Threshold IDX: %d | How many in threshold? %d | Max at threshold: %f", m.Key, m.TotalHits, thresholdIndex, numInThreshold, maxAtThreshold)

	logger.Trace.Printf("Cumultative Values: %v", cumulativeValues)

	// Take the cumulative at the threshold and divide by the threshold idx.
	meanAtPercentile := cumulativeValues[numInThreshold-1] / float32(numInThreshold)

	gkey = fmt.Sprintf("stats.timers.%s.mean_%d", m.Key, c.Percentile)
	sendSingleMetricToGraphite(gkey, meanAtPercentile, stringTime, true, c, logger)

	gkey = fmt.Sprintf("stats.timers.%s.upper_%d", m.Key, c.Percentile)
	sendSingleMetricToGraphite(gkey, maxAtThreshold, stringTime, true, c, logger)

	gkey = fmt.Sprintf("stats.timers.%s.sum_%d", m.Key, c.Percentile)
	sendSingleMetricToGraphite(gkey, cumulativeValues[numInThreshold-1], stringTime, true, c, logger)
}

// sendSingleMetricToGraphite formats a message and a value and time and sends to Graphite.
func sendSingleMetricToGraphite(key string, v float32, t string, retry bool, relay CarbonRelay, logger Logger) {
	var releaseErr error
	dataSent := false

	sv := strconv.FormatFloat(float64(v), 'f', 6, 32)
	payload := fmt.Sprintf("%s %s %s", key, sv, t)

	// Send to the remote host.
	conn, connErr := relay.ConnectionPool.GetConnection(logger)

	if connErr != nil {
		logger.Error.Println("Could not connect to remote host.", connErr)
		// If there was an error connecting, recreate the connection and retry.
		_, releaseErr = relay.ConnectionPool.ReleaseConnection(conn, true, logger)
	} else {
		// Write to the connection.
		_, writeErr := fmt.Fprintf(conn, fmt.Sprintf("%s\n", payload))
		if writeErr != nil {
			// If there was an error writing, recreate the connection and retry.
			logger.Error.Printf("%v", writeErr)
			_, releaseErr = relay.ConnectionPool.ReleaseConnection(conn, true, logger)
		} else {
			// If the metric was written, just release that connection back.
			logger.Trace.Printf("Payload: %v", payload)
			_, releaseErr = relay.ConnectionPool.ReleaseConnection(conn, false, logger)
			dataSent = true
		}
	}

	// For some reason we were unable to release this connection.
	if releaseErr != nil {
		logger.Error.Println("Relay connection not released.", releaseErr)
	}

	// If data was not sent, likely a socket timeout, we'll retry one time.
	if !dataSent && retry {
		logger.Error.Printf("Metric not sent, retrying %v", payload)
		sendSingleMetricToGraphite(key, v, t, false, relay, logger)
	}
}

// ByFloat32 implements sort.Interface for []Float32.
type ByFloat32 []float32

func (a ByFloat32) Len() int           { return len(a) }
func (a ByFloat32) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByFloat32) Less(i, j int) bool { return a[i] < a[j] }

func logger(msg string) {
	fmt.Println(msg)
}

// MockRelay implements MetricRelay.
type MockRelay struct {
}

// Relay implements MetricRelay::Relay().
func (c MockRelay) Relay(metric Metric, logger Logger) {
	logger.Trace.Printf(fmt.Sprintf("Mock flush: %s %v %s", metric.Key, metric.LastValue, metric.MetricType))
}
