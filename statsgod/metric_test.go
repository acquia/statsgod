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

package statsgod_test

import (
	"fmt"
	. "github.com/acquia/statsgod/statsgod"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"math"
	"time"
)

var metricBenchmarkTimeLimit = 0.25

// Creates a Metric struct with default values.
func getDefaultMetricStructure() Metric {
	var metric = new(Metric)

	return *metric
}

var _ = Describe("Metrics", func() {

	Describe("Testing the basic structure", func() {
		metric := getDefaultMetricStructure()
		It("should contain values", func() {
			// Ensure that the expected values exist.
			Expect(metric.Key).ShouldNot(Equal(nil))
			Expect(metric.MetricType).ShouldNot(Equal(nil))
			Expect(metric.TotalHits).ShouldNot(Equal(nil))
			Expect(metric.LastValue).ShouldNot(Equal(nil))
			Expect(metric.FlushTime).ShouldNot(Equal(nil))
			Expect(metric.LastFlushed).ShouldNot(Equal(nil))

			// Slices when empty evaluate to nil, check for len instead.
			Expect(len(metric.AllValues)).Should(Equal(0))
		})
	})

	Describe("Testing the functionality", func() {
		Context("when we parse metrics", func() {
			// Test that we can correctly parse a metric string.
			It("should contain correct values", func() {
				metricOne, err := ParseMetricString("test.three:3|g")
				Expect(err).Should(BeNil())
				Expect(metricOne).ShouldNot(Equal(nil))
				Expect(metricOne.Key).Should(Equal("test.three"))
				Expect(metricOne.LastValue).Should(Equal(float64(3)))
				Expect(metricOne.MetricType).Should(Equal(MetricTypeGauge))
			})

			// Test that incorrect strings trigger errors.
			It("should throw errors on malformed strings", func() {
				_, errOne := ParseMetricString("test3|g")
				Expect(errOne).ShouldNot(Equal(nil))
				_, errTwo := ParseMetricString("test:3g")
				Expect(errTwo).ShouldNot(Equal(nil))
				_, errThree := ParseMetricString("test:3|gauge")
				Expect(errThree).ShouldNot(Equal(nil))
				_, errFour := ParseMetricString("test:three|g")
				Expect(errFour).ShouldNot(Equal(nil))
			})

			Measure("it should be able to parse metric strings quickly.", func(b Benchmarker) {
				runtime := b.Time("runtime", func() {
					metric, _ := ParseMetricString("test:1|g")
					metric, _ = ParseMetricString("test:2|c")
					metric, _ = ParseMetricString("test:3|ms")
					Expect(metric).ShouldNot(Equal(nil))
				})

				Expect(runtime.Seconds()).Should(BeNumerically("<", metricBenchmarkTimeLimit), "it should be able to parse metric strings quickly.")

			}, 100000)

		})

		Context("when we create simple metrics", func() {
			It("should create a metric with the correct values", func() {
				key := "test"
				value := float64(123)
				metricType := MetricTypeGauge
				metric := CreateSimpleMetric(key, value, metricType)
				Expect(metric.Key).Should(Equal(key))
				Expect(metric.MetricType).Should(Equal(metricType))
				Expect(metric.LastValue).Should(Equal(value))
				Expect(metric.AllValues[0]).Should(Equal(value))
				Expect(metric.TotalHits).Should(Equal(1))
			})
		})

		Context("when we aggregate metrics", func() {
			metrics := make(map[string]Metric)
			testMetrics := []string{
				"test.one:1|c",
				"test.one:1|c",
				"test.one:1|c",
				"test.one:1|c",
				"test.one:1|c",
				"test.two:3|c",
				"test.three:45|g",
				"test.three:33|g",
				"test.four:100|ms",
				"test.four:150|ms",
			}

			for _, metricString := range testMetrics {
				m, _ := ParseMetricString(metricString)
				AggregateMetric(metrics, *m)
			}

			It("should combine counters and keep last for metrics and timers", func() {
				// Test that we now have two metrics stored.
				Expect(len(metrics)).Should(Equal(4))

				// Test that the equal namespaces sum values and increment hits.
				existingMetric, metricExists := metrics["test.one"]
				Expect(metricExists).Should(Equal(true))
				Expect(len(existingMetric.AllValues)).Should(Equal(5))
				Expect(existingMetric.TotalHits).Should(Equal(5))
				Expect(existingMetric.LastValue).Should(Equal(float64(5)))

				existingMetric, metricExists = metrics["test.two"]
				Expect(metricExists).Should(Equal(true))
				Expect(len(existingMetric.AllValues)).Should(Equal(1))
				Expect(existingMetric.TotalHits).Should(Equal(1))
				Expect(existingMetric.LastValue).Should(Equal(float64(3)))

				existingMetric, metricExists = metrics["test.three"]
				Expect(metricExists).Should(Equal(true))
				Expect(len(existingMetric.AllValues)).Should(Equal(2))
				Expect(existingMetric.TotalHits).Should(Equal(2))
				Expect(existingMetric.LastValue).Should(Equal(float64(33)))
			})

			Measure("it should aggregate metrics quickly.", func(b Benchmarker) {
				metrics := make(map[string]Metric)
				metric, _ := ParseMetricString("test:1|c")
				runtime := b.Time("runtime", func() {
					AggregateMetric(metrics, *metric)
				})

				Expect(runtime.Seconds()).Should(BeNumerically("<", metricBenchmarkTimeLimit), "it should aggregate metrics quickly.")
			}, 100000)
		})

		Context("when we process metrics", func() {
			logger := *CreateLogger(ioutil.Discard, ioutil.Discard, ioutil.Discard, ioutil.Discard)
			metrics := make(map[string]Metric)

			// Test that counters aggregate and process properly.
			for i := 0; i < 11; i++ {
				metricCount, _ := ParseMetricString("test.count:1|c")
				AggregateMetric(metrics, *metricCount)
			}
			It("should calculate the total and rate properly", func() {
				metricCount := metrics["test.count"]
				ProcessMetric(&metricCount, time.Second*10, []int{80}, logger)
				Expect(metricCount.ValuesPerSecond).Should(Equal(1.1))
				Expect(metricCount.LastValue).Should(Equal(float64(11)))
			})

			// Test that gauges average properly.
			for i := 1; i < 11; i++ {
				gauge := float64(i) * float64(i)
				metricGauge, _ := ParseMetricString(fmt.Sprintf("test.gauge:%f|g", gauge))
				AggregateMetric(metrics, *metricGauge)
			}
			It("should average gauges properly", func() {
				metricGauge := metrics["test.gauge"]
				ProcessMetric(&metricGauge, time.Second*10, []int{80}, logger)
				Expect(metricGauge.MedianValue).Should(Equal(30.5))
				Expect(metricGauge.MeanValue).Should(Equal(38.5))
				Expect(metricGauge.LastValue).Should(Equal(100.0))
			})

			// Test all of the timer calculations.
			for i := 3; i < 14; i++ {
				timer := float64(i) * float64(i)
				metricTimer, _ := ParseMetricString(fmt.Sprintf("test.timer:%f|ms", timer))
				AggregateMetric(metrics, *metricTimer)
			}
			It("should calculate timer values properly", func() {
				metricTimer := metrics["test.timer"]
				ProcessMetric(&metricTimer, time.Second*10, []int{50, 75, 90}, logger)

				Expect(len(metricTimer.Quantiles)).Should(Equal(3))

				Expect(metricTimer.MinValue).Should(Equal(float64(9)))
				Expect(metricTimer.MaxValue).Should(Equal(float64(169)))
				Expect(metricTimer.MeanValue).Should(Equal(float64(74)))
				Expect(metricTimer.MedianValue).Should(Equal(float64(64)))
				Expect(metricTimer.LastValue).Should(Equal(float64(169)))
				// Quantiles
				q := metricTimer.Quantiles[0]
				Expect(q.Mean).Should(Equal(float64(33.166666666666664)))
				Expect(q.Median).Should(Equal(float64(30.5)))
				Expect(q.Max).Should(Equal(float64(64)))
				Expect(q.Sum).Should(Equal(float64(199)))

				q = metricTimer.Quantiles[1]
				GinkgoWriter.Write([]byte(fmt.Sprintf("%v", q)))
				Expect(q.Mean).Should(Equal(float64(47.5)))
				Expect(q.Median).Should(Equal(float64(42.5)))
				Expect(q.Max).Should(Equal(float64(100)))
				Expect(q.Sum).Should(Equal(float64(380)))

				q = metricTimer.Quantiles[2]
				Expect(q.Mean).Should(Equal(float64(64.5)))
				Expect(q.Median).Should(Equal(float64(56.5)))
				Expect(q.Max).Should(Equal(float64(144)))
				Expect(q.Sum).Should(Equal(float64(645)))
			})

			// Test the set-metric-type unique values.
			for i := 1; i < 21; i++ {
				set := math.Floor(float64(i) / float64(2))
				metricSet, _ := ParseMetricString(fmt.Sprintf("test.set:%f|s", set))
				AggregateMetric(metrics, *metricSet)
			}
			It("should calculate set unique values properly", func() {
				metricSet := metrics["test.set"]
				ProcessMetric(&metricSet, time.Second*10, []int{90}, logger)
				Expect(metricSet.LastValue).Should(Equal(float64(11)))
			})

			Measure("it should process metrics quickly.", func(b Benchmarker) {
				metric := metrics["test.gauge"]
				runtime := b.Time("runtime", func() {
					ProcessMetric(&metric, time.Second*10, []int{80}, logger)
				})

				Expect(runtime.Seconds()).Should(BeNumerically("<", metricBenchmarkTimeLimit), "it should process metrics quickly.")
			}, 100000)

		})
	})

})
