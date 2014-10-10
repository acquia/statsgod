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
	"strconv"
	"strings"
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
	Key         string    // Name of the metric.
	MetricType  string    // What type of metric is it (gauge, counter, timer)
	TotalHits   int       // Number of times it has been used.
	LastValue   float32   // The last value stored.
	AllValues   []float32 // All of the values.
	FlushTime   int       // What time are we sending Graphite?
	LastFlushed int       // When did we last flush this out?
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
	metric.LastValue = float32(parsedValue)
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
