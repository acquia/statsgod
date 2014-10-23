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
		config ConfigValues
		yaml   ConfigValues
	)

	Describe("Loading runtime configuration", func() {
		Context("Loading default values", func() {
			config, _ = LoadConfig("")
			It("should contain defaults", func() {
				Expect(config.Service.Name).ShouldNot(Equal(nil))
				Expect(config.Service.Debug).ShouldNot(Equal(nil))
				Expect(config.Connection.Tcp.Host).ShouldNot(Equal(nil))
				Expect(config.Connection.Tcp.Port).ShouldNot(Equal(nil))
				Expect(config.Connection.Udp.Host).ShouldNot(Equal(nil))
				Expect(config.Connection.Udp.Port).ShouldNot(Equal(nil))
				Expect(config.Connection.Unix.File).ShouldNot(Equal(nil))
				Expect(config.Relay.Type).ShouldNot(Equal(nil))
				Expect(config.Relay.Concurrency).ShouldNot(Equal(nil))
				Expect(config.Relay.Timeout).ShouldNot(Equal(nil))
				Expect(config.Relay.Flush).ShouldNot(Equal(nil))
				Expect(config.Carbon.Host).ShouldNot(Equal(nil))
				Expect(config.Carbon.Port).ShouldNot(Equal(nil))
				Expect(config.Stats.Percentile).ShouldNot(Equal(nil))

			})
		})

		Context("Loading config file", func() {
			yaml, _ = LoadConfig("../config.yml")
			It("should match the defaults", func() {
				Expect(yaml.Service.Name).Should(Equal(config.Service.Name))
				Expect(yaml.Service.Debug).Should(Equal(config.Service.Debug))
				Expect(yaml.Connection.Tcp.Host).Should(Equal(config.Connection.Tcp.Host))
				Expect(yaml.Connection.Tcp.Port).Should(Equal(config.Connection.Tcp.Port))
				Expect(yaml.Connection.Udp.Host).Should(Equal(config.Connection.Udp.Host))
				Expect(yaml.Connection.Udp.Port).Should(Equal(config.Connection.Udp.Port))
				Expect(yaml.Connection.Unix.File).Should(Equal(config.Connection.Unix.File))
				Expect(yaml.Relay.Type).Should(Equal(config.Relay.Type))
				Expect(yaml.Relay.Concurrency).Should(Equal(config.Relay.Concurrency))
				Expect(yaml.Relay.Timeout).Should(Equal(config.Relay.Timeout))
				Expect(yaml.Relay.Flush).Should(Equal(config.Relay.Flush))
				Expect(yaml.Carbon.Host).Should(Equal(config.Carbon.Host))
				Expect(yaml.Carbon.Port).Should(Equal(config.Carbon.Port))
				Expect(yaml.Stats.Percentile).Should(Equal(config.Stats.Percentile))

			})
		})

		Context("Loading bogus file", func() {
			It("should throw an error", func() {
				_, noFileErr := LoadConfig("noFile")
				Expect(noFileErr).ShouldNot(Equal(nil))
			})
		})
	})

})
