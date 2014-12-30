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
	"reflect"
)

var _ = Describe("Auth", func() {

	var testToken = "test-token"
	var testMetric = "foo.bar:123|g"
	var config ConfigValues

	// The boolean value is whether or not the token auth should succeed.
	var testMetrics = map[string]bool{
		"":                           false,
		"foo":                        false,
		testMetric:                   false,
		testToken + ".foo":           true,
		testToken:                    false,
		testToken + "." + testMetric: true,
	}

	Describe("Testing the authentication", func() {
		BeforeEach(func() {
			config, _ = CreateConfig("")
		})

		Context("when using the factory function", func() {
			It("should be a complete structure", func() {
				// Create an token config auth.
				config.Service.Auth = AuthTypeConfigToken
				tokenAuth := CreateAuth(config)
				Expect(tokenAuth).ShouldNot(BeNil())
				Expect(reflect.TypeOf(tokenAuth).String()).Should(Equal("*statsgod.AuthConfigToken"))

				// Create a no auth.
				config.Service.Auth = AuthTypeNone
				noAuth := CreateAuth(config)
				Expect(noAuth).ShouldNot(BeNil())
				Expect(reflect.TypeOf(noAuth).String()).Should(Equal("*statsgod.AuthNone"))

				// No auth should be default.
				config.Service.Auth = "foo"
				defaultAuth := CreateAuth(config)
				Expect(defaultAuth).ShouldNot(BeNil())
				Expect(reflect.TypeOf(defaultAuth).String()).Should(Equal("*statsgod.AuthNone"))

			})
		})

		Context("when using AuthNone", func() {
			It("should always authenticate", func() {
				config.Service.Auth = AuthTypeNone
				noAuth := CreateAuth(config)

				// No auth should always authenticate.
				for metric, _ := range testMetrics {
					auth, _ := noAuth.Authenticate(&metric)
					Expect(auth).Should(BeTrue())
				}
			})
		})

		Context("when using AuthConfigToken", func() {
			It("AuthConfigToken should only allow valid tokens", func() {
				config.Service.Auth = AuthTypeConfigToken
				tokenAuth := CreateAuth(config).(*AuthConfigToken)
				tokenAuth.Tokens = map[string]bool{testToken: true}

				// Auth should only authenticate with a valid token.
				for metric, valid := range testMetrics {
					auth, _ := tokenAuth.Authenticate(&metric)
					Expect(auth).Should(Equal(valid))
				}

				// Test that the token is stripped from the metric.
				validMetric := testToken + "." + testMetric
				tokenAuth.Authenticate(&validMetric)
				Expect(validMetric).Should(Equal(testMetric))

				// Invalidate the token to ensure that auth fails.
				tokenAuth.Tokens[testToken] = false
				invalidMetric := testToken + "." + testMetric
				invalidAuth, _ := tokenAuth.Authenticate(&invalidMetric)
				Expect(invalidAuth).Should(BeFalse())
			})
		})
	})
})
