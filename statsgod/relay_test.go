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
	"reflect"
	"time"
)

var _ = Describe("Relay", func() {
	var (
		tmpPort int
		logger  Logger
		config  ConfigValues
	)

	Describe("Testing the basic structure", func() {
		BeforeEach(func() {
			logger = *CreateLogger(ioutil.Discard, ioutil.Discard, ioutil.Discard, ioutil.Discard)
			config, _ = CreateConfig("")
			config.Relay.Type = "mock"
			tmpPort = StartTemporaryListener()
		})

		AfterEach(func() {
			StopTemporaryListener()
		})

		Context("when using the factory function", func() {
			It("should be a complete structure", func() {
				// Tests that we can get a mock relay.
				mockRelay := CreateRelay(RelayTypeMock)
				Expect(mockRelay).ShouldNot(Equal(nil))
				Expect(reflect.TypeOf(mockRelay).String()).Should(Equal("*statsgod.MockRelay"))

				// Tests that we can get a carbon relay.
				carbonRelay := CreateRelay(RelayTypeCarbon)
				Expect(carbonRelay).ShouldNot(Equal(nil))
				Expect(reflect.TypeOf(carbonRelay).String()).Should(Equal("*statsgod.CarbonRelay"))

				// Tests that we can get a mock relay as the default value
				fooRelay := CreateRelay("foo")
				Expect(fooRelay).ShouldNot(Equal(nil))
				Expect(reflect.TypeOf(fooRelay).String()).Should(Equal("*statsgod.MockRelay"))
			})
		})

		Context("when creating a CarbonRelay", func() {
			It("should be a complete structure", func() {
				backendRelay := CreateRelay(RelayTypeCarbon).(*CarbonRelay)
				backendRelay.Percentile = []int{50, 75, 90}
				backendRelay.SetPrefixesAndSuffixes(config)
				Expect(backendRelay.Prefixes[NamespaceTypeCounter]).Should(Equal("stats.counts."))
				Expect(backendRelay.Prefixes[NamespaceTypeGauge]).Should(Equal("stats.gauges."))
				Expect(backendRelay.Prefixes[NamespaceTypeRate]).Should(Equal("stats.rates."))
				Expect(backendRelay.Prefixes[NamespaceTypeSet]).Should(Equal("stats.sets."))
				Expect(backendRelay.Prefixes[NamespaceTypeTimer]).Should(Equal("stats.timers."))
				Expect(backendRelay.Suffixes[NamespaceTypeCounter]).Should(Equal(""))
				Expect(backendRelay.Suffixes[NamespaceTypeGauge]).Should(Equal(""))
				Expect(backendRelay.Suffixes[NamespaceTypeRate]).Should(Equal(""))
				Expect(backendRelay.Suffixes[NamespaceTypeSet]).Should(Equal(""))
				Expect(backendRelay.Suffixes[NamespaceTypeTimer]).Should(Equal(""))
				Expect(backendRelay.FlushInterval).ShouldNot(Equal(nil))
				Expect(backendRelay.Percentile).ShouldNot(Equal(nil))
				// At this point the connection pool has not been established.
				Expect(backendRelay.ConnectionPool).ShouldNot(Equal(nil))

				// Test the Relay() function.
				logger := *CreateLogger(ioutil.Discard, ioutil.Discard, ioutil.Discard, ioutil.Discard)
				pool, _ := CreateConnectionPool(1, fmt.Sprintf("127.0.0.1:%d", tmpPort), ConnPoolTypeTcp, 10*time.Second, logger)
				backendRelay.ConnectionPool = pool
				testMetrics := []string{
					"test.one:3|c",
					"test.two:3|g",
					"test.three:3|ms",
					"test.four:3|s",
				}

				var metric *Metric
				for _, testMetric := range testMetrics {
					metric, _ = ParseMetricString(testMetric)
					backendRelay.Relay(*metric, logger)
				}

				// Test prefixes and suffixes.
				config.Namespace.Prefix = "p"
				config.Namespace.Prefixes.Counters = "c"
				config.Namespace.Prefixes.Gauges = "g"
				config.Namespace.Prefixes.Rates = "r"
				config.Namespace.Prefixes.Sets = "s"
				config.Namespace.Prefixes.Timers = "t"
				config.Namespace.Suffix = "s"
				config.Namespace.Suffixes.Counters = "c"
				config.Namespace.Suffixes.Gauges = "g"
				config.Namespace.Suffixes.Rates = "r"
				config.Namespace.Suffixes.Sets = "s"
				config.Namespace.Suffixes.Timers = "t"
				backendRelay.SetPrefixesAndSuffixes(config)
				Expect(backendRelay.ApplyPrefixAndSuffix("m", NamespaceTypeCounter)).Should(Equal("p.c.m.c.s"))
				Expect(backendRelay.ApplyPrefixAndSuffix("m", NamespaceTypeGauge)).Should(Equal("p.g.m.g.s"))
				Expect(backendRelay.ApplyPrefixAndSuffix("m", NamespaceTypeRate)).Should(Equal("p.r.m.r.s"))
				Expect(backendRelay.ApplyPrefixAndSuffix("m", NamespaceTypeSet)).Should(Equal("p.s.m.s.s"))
				Expect(backendRelay.ApplyPrefixAndSuffix("m", NamespaceTypeTimer)).Should(Equal("p.t.m.t.s"))

				// Test a broken relay.
				StopTemporaryListener()

				Expect(func() { backendRelay.Relay(*metric, logger) }).Should(Panic())

				// Restart the broken relay to shutdown properly.
				tmpPort = StartTemporaryListener()
			})
		})

		Context("when creating a MockRelay", func() {
			It("should be a complete structure", func() {
				backendRelay := CreateRelay(RelayTypeMock).(*MockRelay)
				metricOne, _ := ParseMetricString("test.one:3|c")
				logger := *CreateLogger(ioutil.Discard, ioutil.Discard, ioutil.Discard, ioutil.Discard)
				backendRelay.Relay(*metricOne, logger)
			})
		})
	})
})
