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
)

var _ = Describe("Config", func() {

	var (
		config            ConfigValues
		yaml              ConfigValues
		fileErr           error
		exampleConfigFile string = "../example.config.yml"
	)

	Describe("Loading runtime configuration", func() {
		Context("Loading default values", func() {
			config, _ = CreateConfig("")
			It("should contain defaults", func() {
				// Service
				Expect(config.Service.Name).ShouldNot(Equal(nil))
				Expect(config.Service.Auth).ShouldNot(Equal(nil))
				Expect(config.Service.Tokens).ShouldNot(Equal(nil))

				// Connection
				Expect(config.Connection.Tcp.Host).ShouldNot(Equal(nil))
				Expect(config.Connection.Tcp.Port).ShouldNot(Equal(nil))
				Expect(config.Connection.Udp.Host).ShouldNot(Equal(nil))
				Expect(config.Connection.Udp.Port).ShouldNot(Equal(nil))
				Expect(config.Connection.Udp.Maxpacket).ShouldNot(Equal(nil))
				Expect(config.Connection.Unix.File).ShouldNot(Equal(nil))

				// Relay
				Expect(config.Relay.Type).ShouldNot(Equal(nil))
				Expect(config.Relay.Concurrency).ShouldNot(Equal(nil))
				Expect(config.Relay.Timeout).ShouldNot(Equal(nil))
				Expect(config.Relay.Flush).ShouldNot(Equal(nil))

				// Carbon
				Expect(config.Carbon.Host).ShouldNot(Equal(nil))
				Expect(config.Carbon.Port).ShouldNot(Equal(nil))

				// Stats
				Expect(config.Stats.Percentile).ShouldNot(Equal(nil))

				// Namespace
				Expect(config.Namespace.Prefix).ShouldNot(Equal(nil))
				Expect(config.Namespace.Prefixes.Counters).ShouldNot(Equal(nil))
				Expect(config.Namespace.Prefixes.Gauges).ShouldNot(Equal(nil))
				Expect(config.Namespace.Prefixes.Rates).ShouldNot(Equal(nil))
				Expect(config.Namespace.Prefixes.Sets).ShouldNot(Equal(nil))
				Expect(config.Namespace.Prefixes.Timers).ShouldNot(Equal(nil))
				Expect(config.Namespace.Suffix).ShouldNot(Equal(nil))
				Expect(config.Namespace.Suffixes.Counters).ShouldNot(Equal(nil))
				Expect(config.Namespace.Suffixes.Gauges).ShouldNot(Equal(nil))
				Expect(config.Namespace.Suffixes.Rates).ShouldNot(Equal(nil))
				Expect(config.Namespace.Suffixes.Sets).ShouldNot(Equal(nil))
				Expect(config.Namespace.Suffixes.Timers).ShouldNot(Equal(nil))

				// Debug
				Expect(config.Debug.Verbose).ShouldNot(Equal(nil))
				Expect(config.Debug.Receipt).ShouldNot(Equal(nil))
				Expect(config.Debug.Profile).ShouldNot(Equal(nil))
				Expect(config.Debug.Relay).ShouldNot(Equal(nil))
			})
		})

		Context("Loading config file", func() {
			yaml, fileErr = CreateConfig(exampleConfigFile)
			It("should match the defaults", func() {
				Expect(fileErr).Should(BeNil())

				// Service
				Expect(yaml.Service.Name).Should(Equal(config.Service.Name))
				Expect(yaml.Service.Auth).Should(Equal(config.Service.Auth))
				Expect(yaml.Service.Tokens).Should(Equal(config.Service.Tokens))

				// Connection
				Expect(yaml.Connection.Tcp.Host).Should(Equal(config.Connection.Tcp.Host))
				Expect(yaml.Connection.Tcp.Port).Should(Equal(config.Connection.Tcp.Port))
				Expect(yaml.Connection.Udp.Host).Should(Equal(config.Connection.Udp.Host))
				Expect(yaml.Connection.Udp.Port).Should(Equal(config.Connection.Udp.Port))
				Expect(yaml.Connection.Udp.Maxpacket).Should(Equal(config.Connection.Udp.Maxpacket))
				Expect(yaml.Connection.Unix.File).Should(Equal(config.Connection.Unix.File))

				// Relay
				Expect(yaml.Relay.Type).Should(Equal(config.Relay.Type))
				Expect(yaml.Relay.Concurrency).Should(Equal(config.Relay.Concurrency))
				Expect(yaml.Relay.Timeout).Should(Equal(config.Relay.Timeout))
				Expect(yaml.Relay.Flush).Should(Equal(config.Relay.Flush))

				// Carbon
				Expect(yaml.Carbon.Host).Should(Equal(config.Carbon.Host))
				Expect(yaml.Carbon.Port).Should(Equal(config.Carbon.Port))

				// Stats
				Expect(yaml.Stats.Percentile).Should(Equal(config.Stats.Percentile))

				// Namespace
				Expect(yaml.Namespace.Prefix).Should(Equal(config.Namespace.Prefix))
				Expect(yaml.Namespace.Prefixes.Counters).Should(Equal(config.Namespace.Prefixes.Counters))
				Expect(yaml.Namespace.Prefixes.Gauges).Should(Equal(config.Namespace.Prefixes.Gauges))
				Expect(yaml.Namespace.Prefixes.Rates).Should(Equal(config.Namespace.Prefixes.Rates))
				Expect(yaml.Namespace.Prefixes.Sets).Should(Equal(config.Namespace.Prefixes.Sets))
				Expect(yaml.Namespace.Prefixes.Timers).Should(Equal(config.Namespace.Prefixes.Timers))
				Expect(yaml.Namespace.Suffix).Should(Equal(config.Namespace.Suffix))
				Expect(yaml.Namespace.Suffixes.Counters).Should(Equal(config.Namespace.Suffixes.Counters))
				Expect(yaml.Namespace.Suffixes.Gauges).Should(Equal(config.Namespace.Suffixes.Gauges))
				Expect(yaml.Namespace.Suffixes.Rates).Should(Equal(config.Namespace.Suffixes.Rates))
				Expect(yaml.Namespace.Suffixes.Sets).Should(Equal(config.Namespace.Suffixes.Sets))
				Expect(yaml.Namespace.Suffixes.Timers).Should(Equal(config.Namespace.Suffixes.Timers))

				// Debug
				Expect(yaml.Debug.Verbose).Should(Equal(config.Debug.Verbose))
				Expect(yaml.Debug.Receipt).Should(Equal(config.Debug.Receipt))
				Expect(yaml.Debug.Profile).Should(Equal(config.Debug.Profile))
				Expect(yaml.Debug.Relay).Should(Equal(config.Debug.Relay))

			})
		})

		Context("Loading bogus file", func() {
			It("should throw an error", func() {
				_, noFileErr := CreateConfig("noFile")
				Expect(noFileErr).ShouldNot(Equal(nil))
			})
		})

		Context("Re-loading file", func() {
			It("should update the values", func() {
				runtimeConf, _ := CreateConfig(exampleConfigFile)
				originalName := runtimeConf.Service.Name
				runtimeConf.Service.Name = "SomethingNew"
				runtimeConf.LoadFile(exampleConfigFile)
				Expect(runtimeConf.Service.Name).Should(Equal(originalName))
			})
		})
	})

})
