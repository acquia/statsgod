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

// Metric is our main data type.
type Metric struct {
	Key             string     // Name of the metric.
	MetricType      string     // What type of metric is it (gauge, counter, timer)
	TotalHits       int        // Number of times it has been used.
	LastValue       float64    // The last value stored.
	ValuesPerSecond float64    // The number of values per second.
	MinValue        float64    // The min value.
	MaxValue        float64    // The max value.
	MeanValue       float64    // The cumulative mean.
	MedianValue     float64    // The cumulative median.
	MeanInThreshold float64    // The mean average within the threshold.
	MaxInThreshold  float64    // The max value within the threshold.
	SumInThreshold  float64    // The total within the threshold
	AllValues       ValueSlice // All of the values.
	FlushTime       int        // What time are we sending Graphite?
	LastFlushed     int        // When did we last flush this out?
}

// ParseMetricString parses a metric string, and if it is properly constructed,
// create a Metric structure. Expects the format [namespace]:[value]|[type]
func ParseMetricString(metricString string) (*Metric, error) {
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
	MetricType, err := shortTypeToLong(strings.TrimSpace(split2[1]))
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
	metric.Key = strings.TrimSpace(split1[0])
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
		case metric.MetricType == "gauge":
			existingMetric.LastValue = metric.LastValue
		case metric.MetricType == "counter":
			existingMetric.LastValue += metric.LastValue
		case metric.MetricType == "timer":
			existingMetric.LastValue = metric.LastValue
		}

		metrics[metric.Key] = existingMetric
	} else {
		metrics[metric.Key] = metric
	}
}

// ProcessMetric will create additional calculations based on the type of metric.
func ProcessMetric(metric *Metric, flushDuration time.Duration, quantile float64, logger Logger) {
	flushInterval := flushDuration / time.Second

	sort.Sort(metric.AllValues)
	switch metric.MetricType {
	case "counter":
		metric.ValuesPerSecond = float64(len(metric.AllValues)) / float64(flushInterval)
	case "timer":
		metric.MinValue, metric.MaxValue, _ = metric.AllValues.Minmax()

		// Make calculations based on the desired quantile.
		quantileValue := metric.AllValues.Quantile(quantile)
		quantileTotal := float64(0)
		quantileCount := int(0)
		for _, value := range metric.AllValues {
			if value > quantileValue {
				break
			}
			quantileTotal += value
			quantileCount++
		}
		metric.MaxInThreshold = quantileValue
		metric.MeanInThreshold = quantileTotal / float64(quantileCount)
		metric.SumInThreshold = quantileTotal
		fallthrough
	default:
		metric.MedianValue = metric.AllValues.Median()
		metric.MeanValue = metric.AllValues.Mean()
	}
}

// shortTypeToLong converts a single-character metric format to a full term.
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
