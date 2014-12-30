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

// Package statsgod - This library is responsible for parsing and defining what
// a "metric" is that we are going to be aggregating.
package statsgod

import (
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	// SeparatorNamespaceValue is the character separating the namespace and value
	// in the metric string.
	SeparatorNamespaceValue = ":"
	// SeparatorValueType is the character separating the value and metric type in
	// the metric string.
	SeparatorValueType = "|"
)

const (
	// MetricTypeCounter describes a counter, sent as "c"
	MetricTypeCounter = 0
	// MetricTypeGauge describes a gauge, sent as "g"
	MetricTypeGauge = 1
	// MetricTypeSet describes a set, set as "s"
	MetricTypeSet = 2
	// MetricTypeTimer describes a timer, set as "ms"
	MetricTypeTimer = 3
	// MetricTypeUnknown describes a malformed metric type
	MetricTypeUnknown = 4
)

// MetricQuantile tracks a specified quantile measurement.
type MetricQuantile struct {
	Quantile  int        // The specified percentile.
	Boundary  float64    // The calculated quantile value.
	AllValues ValueSlice // All of the values.
	Mean      float64    // The mean value within the quantile.
	Median    float64    // The median value within the quantile.
	Max       float64    // The maxumum value within the quantile.
	Sum       float64    // The sum value within the quantile.
}

// Metric is our main data type.
type Metric struct {
	Key             string           // Name of the metric.
	MetricType      int              // What type of metric is it (gauge, counter, timer)
	TotalHits       int              // Number of times it has been used.
	LastValue       float64          // The last value stored.
	ValuesPerSecond float64          // The number of values per second.
	MinValue        float64          // The min value.
	MaxValue        float64          // The max value.
	MeanValue       float64          // The cumulative mean.
	MedianValue     float64          // The cumulative median.
	Quantiles       []MetricQuantile // A list of quantile calculations.
	AllValues       ValueSlice       // All of the values.
	FlushTime       int              // What time are we sending Graphite?
	LastFlushed     int              // When did we last flush this out?
}

// CreateSimpleMetric is a helper to quickly create a metric with the minimum information.
func CreateSimpleMetric(key string, value float64, metricType int) *Metric {
	metric := new(Metric)
	metric.Key = key
	metric.MetricType = metricType
	metric.LastValue = value
	metric.AllValues = append(metric.AllValues, metric.LastValue)
	metric.TotalHits = 1

	return metric
}

// ParseMetricString parses a metric string, and if it is properly constructed,
// create a Metric structure. Expects the format [namespace]:[value]|[type]
func ParseMetricString(metricString string) (*Metric, error) {
	var metric = new(Metric)

	// First extract the first element which is the namespace.
	split1 := strings.Split(strings.TrimSpace(strings.Trim(metricString, "\x00")), SeparatorNamespaceValue)
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
	MetricType, err := getMetricType(strings.TrimSpace(split2[1]))
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
	metric.Key = split1[0]
	metric.MetricType = MetricType
	metric.LastValue = parsedValue
	metric.AllValues = append(metric.AllValues, metric.LastValue)
	metric.TotalHits = 1

	return metric, nil
}

// AggregateMetric adds the metric to the specified storage map or aggregates
// it with an existing metric which has the same namespace.
func AggregateMetric(metrics map[string]Metric, metric Metric) {
	existingMetric, metricExists := metrics[metric.Key]

	// If the metric exists in the specified map, we either take the last value or
	// sum the values and increment the hit count.
	if metricExists {
		existingMetric.TotalHits++

		existingMetric.AllValues = append(existingMetric.AllValues, metric.LastValue)

		switch {
		case metric.MetricType == MetricTypeCounter:
			existingMetric.LastValue += metric.LastValue
		case metric.MetricType == MetricTypeGauge:
			existingMetric.LastValue = metric.LastValue
		case metric.MetricType == MetricTypeSet:
			existingMetric.LastValue = metric.LastValue
		case metric.MetricType == MetricTypeTimer:
			existingMetric.LastValue = metric.LastValue
		}

		metrics[metric.Key] = existingMetric
	} else {
		metrics[metric.Key] = metric
	}
}

// ProcessMetric will create additional calculations based on the type of metric.
func ProcessMetric(metric *Metric, flushDuration time.Duration, quantiles []int, logger Logger) {
	flushInterval := flushDuration / time.Second

	sort.Sort(metric.AllValues)
	switch metric.MetricType {
	case MetricTypeCounter:
		metric.ValuesPerSecond = float64(len(metric.AllValues)) / float64(flushInterval)
	case MetricTypeSet:
		metric.LastValue = float64(metric.AllValues.UniqueCount())
	case MetricTypeTimer:
		metric.MinValue, metric.MaxValue, _ = metric.AllValues.Minmax()

		metric.Quantiles = make([]MetricQuantile, 0)
		for _, q := range quantiles {
			percentile := float64(q) / float64(100)
			quantile := new(MetricQuantile)
			quantile.Quantile = q

			// Make calculations based on the desired quantile.
			quantile.Boundary = metric.AllValues.Quantile(percentile)
			for _, value := range metric.AllValues {
				if value > quantile.Boundary {
					break
				}
				quantile.AllValues = append(quantile.AllValues, value)
			}
			_, quantile.Max, _ = quantile.AllValues.Minmax()
			quantile.Mean = quantile.AllValues.Mean()
			quantile.Median = quantile.AllValues.Median()
			quantile.Sum = quantile.AllValues.Sum()
			metric.Quantiles = append(metric.Quantiles, *quantile)
		}
		fallthrough
	default:
		metric.MedianValue = metric.AllValues.Median()
		metric.MeanValue = metric.AllValues.Mean()
	}
}

// getMetricType converts a single-character metric format to a full term.
func getMetricType(short string) (int, error) {
	switch {
	case "c" == short:
		return MetricTypeCounter, nil
	case "g" == short:
		return MetricTypeGauge, nil
	case "s" == short:
		return MetricTypeSet, nil
	case "ms" == short:
		return MetricTypeTimer, nil
	}
	return MetricTypeUnknown, errors.New("unknown metric type")
}

// ParseMetrics parses the strings received from clients and creates Metric structures.
func ParseMetrics(parseChannel chan string, relayChannel chan *Metric, auth Auth, logger Logger, quit *bool) {

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

			metric, err := ParseMetricString(metricString)
			if err != nil {
				logger.Error.Printf("Invalid metric: %s, %s", metricString, err)
				continue
			}
			// Push the metric onto the channel to be aggregated and flushed.
			relayChannel <- metric
		case <-time.After(time.Second):
			// Test for a quit signal.
		}
		if *quit {
			break
		}
	}
}
