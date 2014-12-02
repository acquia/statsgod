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
	// RelayTypeCarbon is an enum describing a carbon backend relay.
	RelayTypeCarbon = "carbon"
	// RelayTypeMock is an enum describing a mock backend relay.
	RelayTypeMock = "mock"
)

// MetricRelay defines the interface for a back end implementation.
type MetricRelay interface {
	Relay(metric Metric, logger Logger) bool
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
	Percentile     []int
	ConnectionPool *ConnectionPool
}

// Relay implements MetricRelay::Relay().
func (c CarbonRelay) Relay(metric Metric, logger Logger) bool {
	ProcessMetric(&metric, c.FlushInterval, c.Percentile, logger)
	// @todo: are we ever setting flush time?
	stringTime := strconv.Itoa(metric.FlushTime)
	var gkey string

	defer logger.Info.Println("Done sending to Graphite")

	// @todo: for metrics
	// http://blog.pkhamre.com/2012/07/24/understanding-statsd-and-graphite/
	// Ensure all of the metrics are working correctly.

	switch metric.MetricType {
	case "gauge":
		gkey = fmt.Sprintf("stats.gauges.%s", metric.Key)
		sendCarbonMetric(gkey, metric.LastValue, stringTime, true, c, logger)
	case "counter":
		gkey = fmt.Sprintf("stats.rates.%s", metric.Key)
		sendCarbonMetric(gkey, metric.ValuesPerSecond, stringTime, true, c, logger)

		gkey = fmt.Sprintf("stats.counts.%s", metric.Key)
		sendCarbonMetric(gkey, metric.LastValue, stringTime, true, c, logger)
	case "set":
		gkey = fmt.Sprintf("stats.sets.%s", metric.Key)
		sendCarbonMetric(gkey, metric.LastValue, stringTime, true, c, logger)
	case "timer":
		// Cumulative values.
		gkey = fmt.Sprintf("stats.timers.%s.mean_value", metric.Key)
		sendCarbonMetric(gkey, metric.MeanValue, stringTime, true, c, logger)

		gkey = fmt.Sprintf("stats.timers.%s.median_value", metric.Key)
		sendCarbonMetric(gkey, metric.MedianValue, stringTime, true, c, logger)

		gkey = fmt.Sprintf("stats.timers.%s.max_value", metric.Key)
		sendCarbonMetric(gkey, metric.MaxValue, stringTime, true, c, logger)

		gkey = fmt.Sprintf("stats.timers.%s.min_value", metric.Key)
		sendCarbonMetric(gkey, metric.MinValue, stringTime, true, c, logger)

		// Quantile values.
		for _, q := range metric.Quantiles {
			gkey = fmt.Sprintf("stats.timers.%s.mean_%d", metric.Key, q.Quantile)
			sendCarbonMetric(gkey, q.Mean, stringTime, true, c, logger)

			gkey = fmt.Sprintf("stats.timers.%s.median%d", metric.Key, q.Quantile)
			sendCarbonMetric(gkey, q.Median, stringTime, true, c, logger)

			gkey = fmt.Sprintf("stats.timers.%s.upper_%d", metric.Key, q.Quantile)
			sendCarbonMetric(gkey, q.Max, stringTime, true, c, logger)

			gkey = fmt.Sprintf("stats.timers.%s.sum_%d", metric.Key, q.Quantile)
			sendCarbonMetric(gkey, q.Sum, stringTime, true, c, logger)
		}
	}
	return true
}

// sendCarbonMetric formats a message and a value and time and sends to Graphite.
func sendCarbonMetric(key string, v float64, t string, retry bool, relay CarbonRelay, logger Logger) bool {
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
		dataSent = sendCarbonMetric(key, v, t, false, relay, logger)
	}

	return dataSent
}

// MockRelay implements MetricRelay.
type MockRelay struct {
	FlushInterval time.Duration
	Percentile    []int
}

// Relay implements MetricRelay::Relay().
func (c MockRelay) Relay(metric Metric, logger Logger) bool {
	ProcessMetric(&metric, c.FlushInterval, c.Percentile, logger)
	logger.Trace.Printf(fmt.Sprintf("Mock flush: %v", metric))
	return true
}
