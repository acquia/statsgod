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
	)

	Describe("Testing the basic structure", func() {
		BeforeEach(func() {
			logger = *CreateLogger(ioutil.Discard, ioutil.Discard, ioutil.Discard, ioutil.Discard)
			tmpPort = StartTemporaryListener()
		})

		AfterEach(func() {
			StopTemporaryListener()
		})

		Context("when using the factory function", func() {
			It("should be a complete structure", func() {
				// Tests that we can get a mock relay.
				mockRelay := CreateRelay("mock")
				Expect(mockRelay).ShouldNot(Equal(nil))
				Expect(reflect.TypeOf(mockRelay).String()).Should(Equal("*statsgod.MockRelay"))

				// Tests that we can get a carbon relay.
				carbonRelay := CreateRelay("carbon")
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
				backendRelay := CreateRelay("carbon").(*CarbonRelay)
				Expect(backendRelay.FlushInterval).ShouldNot(Equal(nil))
				Expect(backendRelay.Percentile).ShouldNot(Equal(nil))
				// At this point the connection pool has not been established.
				Expect(backendRelay.ConnectionPool).ShouldNot(Equal(nil))

				// Test the Relay() function.
				logger := *CreateLogger(ioutil.Discard, ioutil.Discard, ioutil.Discard, ioutil.Discard)
				pool, _ := CreateConnectionPool(1, "127.0.0.1", tmpPort, 10*time.Second, logger)
				backendRelay.ConnectionPool = pool
				metricOne, _ := ParseMetricString("test.one:3|c")
				metricTwo, _ := ParseMetricString("test.two:3|ms")
				metricThree, _ := ParseMetricString("test.three:3|g")
				backendRelay.Relay(*metricOne, logger)
				backendRelay.Relay(*metricTwo, logger)
				backendRelay.Relay(*metricThree, logger)
			})
		})

		Context("when creating a MockRelay", func() {
			It("should be a complete structure", func() {
				backendRelay := CreateRelay("mock").(*MockRelay)
				metricOne, _ := ParseMetricString("test.one:3|c")
				logger := *CreateLogger(ioutil.Discard, ioutil.Discard, ioutil.Discard, ioutil.Discard)
				backendRelay.Relay(*metricOne, logger)
			})
		})
	})
})
