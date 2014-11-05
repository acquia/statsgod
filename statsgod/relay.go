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
	"strconv"
	"time"
)

const (
	RelayTypeCarbon = "carbon"
	RelayTypeMock   = "mock"
)

// MetricRelay defines the interface for a back end implementation.
type MetricRelay interface {
	Relay(metric Metric, logger Logger)
}

// CreateRelay is a factory for instantiating remote relays.
func CreateRelay(relayType string) MetricRelay {
	switch relayType {
	case RelayTypeCarbon:
		return new(CarbonRelay)
	case RelayTypeMock:
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
	quantile := float64(c.Percentile) / float64(100)
	ProcessMetric(&m, c.FlushInterval, quantile, logger)
	// @todo: are we ever setting flush time?
	stringTime := strconv.Itoa(m.FlushTime)
	var gkey string

	defer logger.Info.Println("Done sending to Graphite")

	// @todo: for metrics
	// http://blog.pkhamre.com/2012/07/24/understanding-statsd-and-graphite/
	// Ensure all of the metrics are working correctly.

	switch m.MetricType {
	case "gauge":
		gkey = fmt.Sprintf("stats.gauges.%s", m.Key)
		sendSingleMetricToGraphite(gkey, m.LastValue, stringTime, true, c, logger)
	case "counter":
		gkey = fmt.Sprintf("stats.%s", m.Key)
		sendSingleMetricToGraphite(gkey, m.ValuesPerSecond, stringTime, true, c, logger)

		gkey = fmt.Sprintf("stats_counts.%s", m.Key)
		sendSingleMetricToGraphite(gkey, m.LastValue, stringTime, true, c, logger)
	case "timer":
		// Cumulative values.
		gkey = fmt.Sprintf("stats.timers.%s.mean_value", m.Key)
		sendSingleMetricToGraphite(gkey, m.MeanValue, stringTime, true, c, logger)

		gkey = fmt.Sprintf("stats.timers.%s.median_value", m.Key)
		sendSingleMetricToGraphite(gkey, m.MedianValue, stringTime, true, c, logger)

		gkey = fmt.Sprintf("stats.timers.%s.max_value", m.Key)
		sendSingleMetricToGraphite(gkey, m.MaxValue, stringTime, true, c, logger)

		gkey = fmt.Sprintf("stats.timers.%s.min_value", m.Key)
		sendSingleMetricToGraphite(gkey, m.MinValue, stringTime, true, c, logger)

		// Quantile values.
		gkey = fmt.Sprintf("stats.timers.%s.mean_%d", m.Key, c.Percentile)
		sendSingleMetricToGraphite(gkey, m.MeanInThreshold, stringTime, true, c, logger)

		gkey = fmt.Sprintf("stats.timers.%s.upper_%d", m.Key, c.Percentile)
		sendSingleMetricToGraphite(gkey, m.MaxInThreshold, stringTime, true, c, logger)

		gkey = fmt.Sprintf("stats.timers.%s.sum_%d", m.Key, c.Percentile)
		sendSingleMetricToGraphite(gkey, m.SumInThreshold, stringTime, true, c, logger)
	}

}

// sendSingleMetricToGraphite formats a message and a value and time and sends to Graphite.
func sendSingleMetricToGraphite(key string, v float64, t string, retry bool, relay CarbonRelay, logger Logger) {
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

// MockRelay implements MetricRelay.
type MockRelay struct {
	FlushInterval time.Duration
	Percentile    int
}

// Relay implements MetricRelay::Relay().
func (c MockRelay) Relay(metric Metric, logger Logger) {
	quantile := float64(c.Percentile) / float64(100)
	ProcessMetric(&metric, c.FlushInterval, quantile, logger)
	logger.Trace.Printf(fmt.Sprintf("Mock flush: %v", metric))
}
