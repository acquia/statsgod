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

// acceptablePrecision is used to round values to an acceptable test precision.
var acceptablePrecision = float64(10000)

// Creates a Metric struct with default values.
func getDefaultMetricStructure() Metric {
	var metric = new(Metric)

	return *metric
}

// Adjusts values to an acceptable precision for testing.
func testPrecision(value float64) float64 {
	return math.Floor(value*acceptablePrecision) / acceptablePrecision
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
			Expect(metric.MinValue).ShouldNot(Equal(nil))
			Expect(metric.MaxValue).ShouldNot(Equal(nil))
			Expect(metric.MeanValue).ShouldNot(Equal(nil))
			Expect(metric.MedianValue).ShouldNot(Equal(nil))
			Expect(metric.FlushTime).ShouldNot(Equal(nil))
			Expect(metric.LastFlushed).ShouldNot(Equal(nil))
			Expect(metric.SampleRate).ShouldNot(Equal(nil))

			// Slices when empty evaluate to nil, check for len instead.
			Expect(len(metric.Quantiles)).Should(Equal(0))
			Expect(len(metric.AllValues)).Should(Equal(0))
		})
	})

	Describe("Testing the functionality", func() {
		Context("when we parse metrics", func() {
			// Test that we can correctly parse a metric string.
			It("should contain correct values", func() {
				metricOne, err := ParseMetricString("test.three:3|g|@0.75")
				Expect(err).Should(BeNil())
				Expect(metricOne).ShouldNot(Equal(nil))
				Expect(metricOne.Key).Should(Equal("test.three"))
				Expect(metricOne.LastValue).Should(Equal(float64(3)))
				Expect(metricOne.MetricType).Should(Equal(MetricTypeGauge))
				Expect(metricOne.SampleRate).Should(Equal(float64(0.75)))
			})

			It("should contain reasonable defaults", func() {
				metricOne, _ := ParseMetricString("test.three:3|g")
				Expect(metricOne).ShouldNot(Equal(nil))
				Expect(metricOne.TotalHits).Should(Equal(float64(1.0)))
				Expect(metricOne.SampleRate).Should(Equal(float64(1.0)))
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
					metric, _ := ParseMetricString("test:1|g|@0.25")
					metric, _ = ParseMetricString("test:2|c|@0.5")
					metric, _ = ParseMetricString("test:3|ms|@0.75")
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
				Expect(metric.TotalHits).Should(Equal(float64(1.0)))
			})
		})

		Context("when we aggregate metrics", func() {
			metrics := make(map[string]Metric)
			testMetrics := []string{
				"test.one:1|c",
				"test.one:2|c|@0.25",
				"test.one:3|c",
				"test.one:1|c",
				"test.one:1|c|@0.5",
				"test.one:1|c|@0.95",
				"test.one:5|c",
				"test.one:1|c|@0.75",
				"test.two:3|c",
				"test.three:45|g",
				"test.three:35|g",
				"test.three:33|g",
				"test.four:100|ms|@0.25",
				"test.four:150|ms|@0.5",
				"test.four:250|ms",
				"test.four:50|ms|@0.95",
				"test.four:150|ms|@0.95",
				"test.four:120|ms|@0.75",
				"test.four:130|ms|@0.95",
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
				Expect(len(existingMetric.AllValues)).Should(Equal(8))
				Expect(testPrecision(existingMetric.TotalHits)).Should(Equal(float64(12.3859)))
				Expect(testPrecision(existingMetric.LastValue)).Should(Equal(float64(22.3859)))

				existingMetric, metricExists = metrics["test.two"]
				Expect(metricExists).Should(Equal(true))
				Expect(len(existingMetric.AllValues)).Should(Equal(1))
				Expect(existingMetric.TotalHits).Should(Equal(float64(1.0)))
				Expect(existingMetric.LastValue).Should(Equal(float64(3.0)))

				existingMetric, metricExists = metrics["test.three"]
				Expect(metricExists).Should(Equal(true))
				Expect(len(existingMetric.AllValues)).Should(Equal(3))
				Expect(existingMetric.TotalHits).Should(Equal(float64(3.0)))
				Expect(existingMetric.LastValue).Should(Equal(float64(33.0)))

				existingMetric, metricExists = metrics["test.four"]
				Expect(metricExists).Should(Equal(true))
				Expect(len(existingMetric.AllValues)).Should(Equal(7))
				Expect(testPrecision(existingMetric.TotalHits)).Should(Equal(float64(11.4912)))
				Expect(existingMetric.LastValue).Should(Equal(float64(130.0)))
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
			// This is a full test of the goroutine that takes input strings
			// from the socket and converts them to Metrics.
			It("should parse from the channel and populate the relay channel", func() {
				config, _ := CreateConfig("")
				config.Service.Auth = AuthTypeConfigToken
				config.Service.Tokens["test-token"] = true

				parseChannel := make(chan string, 2)
				relayChannel := make(chan *Metric, 2)
				auth := CreateAuth(config)
				quit := false
				go ParseMetrics(parseChannel, relayChannel, auth, logger, &quit)
				parseChannel <- "test-token.test.one:123|c"
				parseChannel <- "test-token.test.two:234|g"
				parseChannel <- "bad-metric"
				parseChannel <- "test-token.bad-metric|g"
				parseChannel <- "bad-token.test.two:234|g"
				for len(parseChannel) > 0 {
					// Wait for the channel to be emptied.
					time.Sleep(time.Microsecond)
				}
				quit = true
				Expect(len(parseChannel)).Should(Equal(0))
				Expect(len(relayChannel)).Should(Equal(2))
				metricOne := <-relayChannel
				metricTwo := <-relayChannel
				Expect(metricOne.LastValue).Should(Equal(float64(123)))
				Expect(metricTwo.LastValue).Should(Equal(float64(234)))
			})

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

			It("should magnify sample rates properly", func() {
				metricSample, _ := ParseMetricString("test.count:1|c|@0.1")
				Expect(testPrecision(metricSample.TotalHits)).Should(Equal(float64(9.9999)))
				metricSample, _ = ParseMetricString("test.count:1|c|@0.25")
				Expect(metricSample.TotalHits).Should(Equal(float64(4.0)))
				metricSample, _ = ParseMetricString("test.count:3|c|@0.5")
				Expect(metricSample.TotalHits).Should(Equal(float64(2.0)))
				metricSample, _ = ParseMetricString("test.count:4|c|@0.75")
				Expect(testPrecision(metricSample.TotalHits)).Should(Equal(float64(1.3333)))
				metricSample, _ = ParseMetricString("test.count:3|c|@0.9")
				Expect(testPrecision(metricSample.TotalHits)).Should(Equal(float64(1.1111)))
				metricSample, _ = ParseMetricString("test.count:3|c|@0.95")
				Expect(testPrecision(metricSample.TotalHits)).Should(Equal(float64(1.0526)))
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
				metricTimer, _ := ParseMetricString(fmt.Sprintf("test.timer:%f|ms|@0.9", timer))
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
				Expect(testPrecision(metricTimer.TotalHits)).Should(Equal(float64(12.2222)))
				Expect(testPrecision(metricTimer.ValuesPerSecond)).Should(Equal(float64(1.2222)))
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
