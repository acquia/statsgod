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

package statsgod

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

// Creates a Metric struct with default values.
func getDefaultMetricStructure() Metric {
	var metric = new(Metric)

	return *metric
}

// Test the expected values in the Metric struct.
func TestMetricStructure(t *testing.T) {
	metric := getDefaultMetricStructure()

	// Ensure that the expected values exist.
	assert.NotNil(t, metric.Key)
	assert.NotNil(t, metric.MetricType)
	assert.NotNil(t, metric.TotalHits)
	assert.NotNil(t, metric.LastValue)
	assert.NotNil(t, metric.FlushTime)
	assert.NotNil(t, metric.LastFlushed)

	// Slices when empty evaluate to nil, check for len instead.
	assert.Equal(t, len(metric.AllValues), 0)
}

// Test the AggregateMetric() function.
func TestAggregateMetric(t *testing.T) {
	metrics := make(map[string]Metric)

	// Tests adding an initial metric.
	metricOne, err := ParseMetricString("test.one:1|c")
	assert.Nil(t, err)
	assert.NotNil(t, metricOne)
	AggregateMetric(metrics, *metricOne)

	// Tests adding an additional metric with the same namespace.
	metricTwo, _ := ParseMetricString("test.one:2|c")
	assert.Nil(t, err)
	assert.NotNil(t, metricTwo)
	AggregateMetric(metrics, *metricTwo)

	// Tests adding a new metric with a new namespace.
	metricThree, _ := ParseMetricString("test.two:1|c")
	assert.Nil(t, err)
	assert.NotNil(t, metricThree)
	AggregateMetric(metrics, *metricThree)

	// Test that we now have two metrics stored.
	assert.Equal(t, len(metrics), 2)

	// Test that the equal namespaces sum values and increment hits.
	existingMetric, metricExists := metrics["test.one"]
	assert.Equal(t, metricExists, true)
	assert.Equal(t, existingMetric.TotalHits, 2)
	assert.Equal(t, existingMetric.LastValue, 3)
	assert.Equal(t, len(existingMetric.AllValues), 2)
}

// Test the ParseMetricString() function.
func TestParseMetricString(t *testing.T) {
	// Test that we can correctly parse a metric string.
	metricOne, err := ParseMetricString("test.three:3|g")
	assert.Nil(t, err)
	assert.NotNil(t, metricOne)
	assert.Equal(t, metricOne.Key, "test.three")
	assert.Equal(t, metricOne.LastValue, float32(3))
	assert.Equal(t, metricOne.MetricType, "gauge")
}
