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
	"syscall"
	"time"
)

func catchSignal(finishChannel chan int) bool {
	select {
	case <-finishChannel:
		return true
	case <-time.After(time.Second):
		return false
	}
}

var _ = Describe("Signals", func() {

	Describe("Testing the signal handling", func() {
		configFile := ""
		config, _ := CreateConfig(configFile)
		finishChannel := make(chan int)
		logger := *CreateLogger(ioutil.Discard, ioutil.Discard, ioutil.Discard, ioutil.Discard)
		ListenForSignals(finishChannel, &config, &configFile, logger)

		config.Service.Name = "testing"
		syscall.Kill(syscall.Getpid(), syscall.SIGHUP)

		It("should stop the parser when a quit signal is sent", func() {
			ListenForSignals(finishChannel, &config, &configFile, logger)
			parseChannel := make(chan string, 2)
			relayChannel := make(chan *Metric, 2)
			auth := CreateAuth(config)
			quit := false
			go ParseMetrics(parseChannel, relayChannel, auth, logger, &quit)
			syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
			quit = true
		})

		It("should catch the 'reload' signal", func() {
			Expect(config.Service.Name).Should(Equal("statsgod"))
		})

		It("should catch the 'quit' signals", func() {
			signals := []syscall.Signal{
				syscall.SIGABRT,
				//syscall.SIGINT, // @TODO SIGINT kills the test for some reason.
				syscall.SIGTERM,
				syscall.SIGQUIT,
			}
			for _, signal := range signals {
				syscall.Kill(syscall.Getpid(), signal)
				Expect(catchSignal(finishChannel)).Should(BeTrue())
				ListenForSignals(finishChannel, &config, &configFile, logger)
			}
		})

	})
})
