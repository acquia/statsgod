package main

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

// Creates a Metric struct with default values.
func getDefaultMetricStructure() Metric {
	var metric = new(Metric)

	return *metric
}

// Test the expected values in the Metric struct.
func TestMetricStructure(t *testing.T) {
	// @todo: log to /dev/null
	logInit(os.Stdout, os.Stdout, os.Stdout, os.Stderr)

	metric := getDefaultMetricStructure()

	// Ensure that the expected values exist.
	assert.NotNil(t, metric.key)
	assert.NotNil(t, metric.metricType)
	assert.NotNil(t, metric.totalHits)
	assert.NotNil(t, metric.lastValue)
	assert.NotNil(t, metric.flushTime)
	assert.NotNil(t, metric.lastFlushed)

	// Slices when empty evaluate to nil, check for len instead.
	assert.Equal(t, len(metric.allValues), 0)
}

// Test the aggregateMetric() function.
func TestAggregateMetric(t *testing.T) {
	// @todo: log to /dev/null
	logInit(os.Stdout, os.Stdout, os.Stdout, os.Stderr)

	metrics := make(map[string]Metric)

	// Tests adding an initial metric.
	metricOne, err := parseMetricString("test.one:1|c")
	assert.Nil(t, err)
	assert.NotNil(t, metricOne)
	aggregateMetric(metrics, *metricOne)

	// Tests adding an additional metric with the same namespace.
	metricTwo, _ := parseMetricString("test.one:2|c")
	assert.Nil(t, err)
	assert.NotNil(t, metricTwo)
	aggregateMetric(metrics, *metricTwo)

	// Tests adding a new metric with a new namespace.
	metricThree, _ := parseMetricString("test.two:1|c")
	assert.Nil(t, err)
	assert.NotNil(t, metricThree)
	aggregateMetric(metrics, *metricThree)

	// Test that we now have two metrics stored.
	assert.Equal(t, len(metrics), 2)

	// Test that the equal namespaces sum values and increment hits.
	existingMetric, metricExists := metrics["test.one"]
	assert.Equal(t, metricExists, true)
	assert.Equal(t, existingMetric.totalHits, 2)
	assert.Equal(t, existingMetric.lastValue, 3)
}

// Test the parseMetricString() function.
func TestParseMetricString(t *testing.T) {
	// @todo: log to /dev/null
	logInit(os.Stdout, os.Stdout, os.Stdout, os.Stderr)

	// Test that we can correctly parse a metric string.
	metricOne, err := parseMetricString("test.three:3|g")
	assert.Nil(t, err)
	assert.NotNil(t, metricOne)
	assert.Equal(t, metricOne.key, "test.three")
	assert.Equal(t, metricOne.lastValue, float32(3))
	assert.Equal(t, metricOne.metricType, "gauge")
}
