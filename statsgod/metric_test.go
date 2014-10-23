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
	"time"
)

// Creates a Metric struct with default values.
func getDefaultMetricStructure() Metric {
	var metric = new(Metric)

	return *metric
}

var _ = Describe("Connection Pool", func() {

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
				Expect(metricOne.MetricType).Should(Equal("gauge"))
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
				ProcessMetric(&metricCount, time.Second*10, float64(0.8), logger)
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
				ProcessMetric(&metricGauge, time.Second*10, float64(0.8), logger)
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
				ProcessMetric(&metricTimer, time.Second*10, float64(0.9), logger)
				Expect(metricTimer.MinValue).Should(Equal(float64(9)))
				Expect(metricTimer.MaxValue).Should(Equal(float64(169)))
				Expect(metricTimer.MeanValue).Should(Equal(float64(74)))
				Expect(metricTimer.MedianValue).Should(Equal(float64(64)))
				Expect(metricTimer.MeanInThreshold).Should(Equal(float64(64.5)))
				Expect(metricTimer.MaxInThreshold).Should(Equal(float64(144)))
				Expect(metricTimer.SumInThreshold).Should(Equal(float64(645)))
				Expect(metricTimer.LastValue).Should(Equal(float64(169)))
			})

		})
	})

})
