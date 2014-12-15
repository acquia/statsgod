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
	// NamespaceTypeCounter is an enum for counter namespacing.
	NamespaceTypeCounter = iota
	// NamespaceTypeGauge is an enum for gauge namespacing.
	NamespaceTypeGauge
	// NamespaceTypeRate is an enum for rate namespacing.
	NamespaceTypeRate
	// NamespaceTypeSet is an enum for rate namespacing.
	NamespaceTypeSet
	// NamespaceTypeTimer is an enum for timer namespacing.
	NamespaceTypeTimer
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
	Prefixes       map[int]string
	Suffixes       map[int]string
}

// SetPrefixesAndSuffixes is a helper to set the prefixes and suffixes from
// config when relaying data.
func (c *CarbonRelay) SetPrefixesAndSuffixes(config ConfigValues) {
	prefix := ""
	suffix := ""
	c.Prefixes = map[int]string{}
	c.Suffixes = map[int]string{}

	configPrefixes := map[int]string{
		NamespaceTypeCounter: config.Namespace.Prefixes.Counters,
		NamespaceTypeGauge:   config.Namespace.Prefixes.Gauges,
		NamespaceTypeRate:    config.Namespace.Prefixes.Rates,
		NamespaceTypeSet:     config.Namespace.Prefixes.Sets,
		NamespaceTypeTimer:   config.Namespace.Prefixes.Timers,
	}

	configSuffixes := map[int]string{
		NamespaceTypeCounter: config.Namespace.Suffixes.Counters,
		NamespaceTypeGauge:   config.Namespace.Suffixes.Gauges,
		NamespaceTypeRate:    config.Namespace.Suffixes.Rates,
		NamespaceTypeSet:     config.Namespace.Suffixes.Sets,
		NamespaceTypeTimer:   config.Namespace.Suffixes.Timers,
	}

	// Global prefix.
	if config.Namespace.Prefix != "" {
		prefix = config.Namespace.Prefix + "."
	}

	// Type prefixes.
	for metricType, typePrefix := range configPrefixes {
		c.Prefixes[metricType] = prefix
		if typePrefix != "" {
			c.Prefixes[metricType] = prefix + typePrefix + "."
		}
	}

	// Global suffix.
	if config.Namespace.Suffix != "" {
		suffix = "." + config.Namespace.Suffix
	}

	// Type suffixes
	for metricType, typeSuffix := range configSuffixes {
		c.Suffixes[metricType] = suffix
		if typeSuffix != "" {
			c.Suffixes[metricType] = "." + typeSuffix + suffix
		}
	}
}

// ApplyPrefixAndSuffix interpolates the configured prefix and suffix with the
// metric namespace string.
func (c CarbonRelay) ApplyPrefixAndSuffix(namespace string, metricType int) string {
	return fmt.Sprintf("%s%s%s", c.Prefixes[metricType], namespace, c.Suffixes[metricType])
}

// Relay implements MetricRelay::Relay().
func (c CarbonRelay) Relay(metric Metric, logger Logger) bool {
	ProcessMetric(&metric, c.FlushInterval, c.Percentile, logger)
	// @todo: are we ever setting flush time?
	stringTime := strconv.Itoa(metric.FlushTime)
	var key string
	var qkey string

	defer logger.Info.Println("Done sending to Graphite")

	switch metric.MetricType {
	case MetricTypeGauge:
		key = c.ApplyPrefixAndSuffix(metric.Key, NamespaceTypeGauge)
		sendCarbonMetric(key, metric.LastValue, stringTime, true, c, logger)
	case MetricTypeCounter:
		key = c.ApplyPrefixAndSuffix(metric.Key, NamespaceTypeRate)
		sendCarbonMetric(key, metric.ValuesPerSecond, stringTime, true, c, logger)

		key = c.ApplyPrefixAndSuffix(metric.Key, NamespaceTypeCounter)
		sendCarbonMetric(key, metric.LastValue, stringTime, true, c, logger)
	case MetricTypeSet:
		key = c.ApplyPrefixAndSuffix(metric.Key, NamespaceTypeSet)
		sendCarbonMetric(key, metric.LastValue, stringTime, true, c, logger)
	case MetricTypeTimer:
		// Cumulative values.
		key = c.ApplyPrefixAndSuffix(metric.Key+".mean_value", NamespaceTypeTimer)
		sendCarbonMetric(key, metric.MeanValue, stringTime, true, c, logger)

		key = c.ApplyPrefixAndSuffix(metric.Key+".median_value", NamespaceTypeTimer)
		sendCarbonMetric(key, metric.MedianValue, stringTime, true, c, logger)

		key = c.ApplyPrefixAndSuffix(metric.Key+".max_value", NamespaceTypeTimer)
		sendCarbonMetric(key, metric.MaxValue, stringTime, true, c, logger)

		key = c.ApplyPrefixAndSuffix(metric.Key+".min_value", NamespaceTypeTimer)
		sendCarbonMetric(key, metric.MinValue, stringTime, true, c, logger)

		// Quantile values.
		for _, q := range metric.Quantiles {
			qkey = strconv.FormatInt(int64(q.Quantile), 10)

			key = c.ApplyPrefixAndSuffix(metric.Key+".mean_"+qkey, NamespaceTypeTimer)
			sendCarbonMetric(key, q.Mean, stringTime, true, c, logger)

			key = c.ApplyPrefixAndSuffix(metric.Key+".median_"+qkey, NamespaceTypeTimer)
			sendCarbonMetric(key, q.Median, stringTime, true, c, logger)

			key = c.ApplyPrefixAndSuffix(metric.Key+".upper_"+qkey, NamespaceTypeTimer)
			sendCarbonMetric(key, q.Max, stringTime, true, c, logger)

			key = c.ApplyPrefixAndSuffix(metric.Key+".sum_"+qkey, NamespaceTypeTimer)
			sendCarbonMetric(key, q.Sum, stringTime, true, c, logger)
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
