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
	"math/rand"
	"net"
	"strings"
	"time"
)

var socketBenchmarkTimeLimit = 0.75

var shortMetricString = generateMetricString(2, "short")
var mediumMetricString = generateMetricString(10, "medium")
var longMetricString = generateMetricString(100, "long")

// Table for running the socket tests.
var testSockets = []struct {
	socketType     int
	socketDesc     string
	socketAddr     string
	badAddr        string
	socketMessages []string
}{
	{SocketTypeTcp, "tcp", "127.0.0.1:0", "0.0.0.0", []string{"test.tcp:4|c", shortMetricString, mediumMetricString, longMetricString}},
	{SocketTypeUdp, "udp", "127.0.0.1:0", "", []string{"test.udp:4|c", shortMetricString, mediumMetricString}},
	{SocketTypeUnix, "unix", "/tmp/statsgod.sock", "/dev/null", []string{"test.unix:4|c", shortMetricString, mediumMetricString, longMetricString}},
}

var sockets = make([]Socket, 3)
var parseChannel = make(chan string)
var logger = *CreateLogger(ioutil.Discard, ioutil.Discard, ioutil.Discard, ioutil.Discard)
var config, _ = CreateConfig("")

var _ = Describe("Sockets", func() {

	Describe("Testing the Socket interface", func() {
		It("should contain the required functions", func() {
			for i, _ := range testSockets {
				_, ok := sockets[i].(interface {
					Listen(parseChannel chan string, logger Logger, config *ConfigValues)
					Close(logger Logger)
					GetAddr() string
					SocketIsActive() bool
				})
				Expect(ok).Should(Equal(true))
			}

		})

		It("should panic for unknown socket types", func() {
			defer GinkgoRecover()
			Expect(func() { CreateSocket(99, "null") }).Should(Panic())
		})
	})

	Describe("Testing the Socket functionality", func() {
		It("should be able to receive messages", func() {
			for i, ts := range testSockets {
				for _, sm := range ts.socketMessages {
					sendSocketMessage(ts.socketDesc, sockets[i], sm)

					receivedMessages := strings.Split(sm, "\n")
					for _, rm := range receivedMessages {
						message := ""
						select {
						case message = <-parseChannel:
						case <-time.After(5 * time.Second):
							message = ""
						}
						Expect(strings.TrimSpace(message)).Should(Equal(strings.TrimSpace(rm)))
					}
				}
			}
		})

		It("should ignore empty values", func() {
			message := ""
			for i, ts := range testSockets {
				sendSocketMessage(ts.socketDesc, sockets[i], "\x00")
				sendSocketMessage(ts.socketDesc, sockets[i], "\n")
				sendSocketMessage(ts.socketDesc, sockets[i], "")
				select {
				case message = <-parseChannel:
				case <-time.After(100 * time.Microsecond):
					message = ""
				}
				Expect(message).Should(Equal(""))
			}
		})

		It("should panic if it has a bad address", func() {
			for _, ts := range testSockets {
				socket := CreateSocket(ts.socketType, "")
				Expect(func() { socket.Listen(parseChannel, logger, &config) }).Should(Panic())
				socket = CreateSocket(ts.socketType, ts.badAddr)
				Expect(func() { socket.Listen(parseChannel, logger, &config) }).Should(Panic())
			}
		})

		It("should block while waiting for a socket to become active", func() {
			socket := CreateSocket(SocketTypeTcp, "127.0.0.1:0")
			start := time.Now()
			BlockForSocket(socket, time.Second)
			Expect(time.Now()).Should(BeTemporally(">=", start, time.Second))
		})

		It("should panic if it is already listening on an address.", func() {
			for i, ts := range testSockets {
				socket := CreateSocket(ts.socketType, sockets[i].GetAddr())
				Expect(func() { socket.Listen(parseChannel, logger, &config) }).Should(Panic())
			}
		})

		Measure("it should receive metrics quickly.", func(b Benchmarker) {
			runtime := b.Time("runtime", func() {
				for i, ts := range testSockets {
					sendSocketMessage(ts.socketDesc, sockets[i], ts.socketMessages[0])
					<-parseChannel
				}
			})

			Expect(runtime.Seconds()).Should(BeNumerically("<", socketBenchmarkTimeLimit), "it should receive metrics quickly.")
		}, 1000)

	})
})

// sendSocketMessage will connect to a specified socket type send a message and close.
func sendSocketMessage(socketType string, socket Socket, message string) {
	conn, err := net.Dial(socketType, socket.GetAddr())
	if err != nil {
		panic(fmt.Sprintf("Dial failed: %v", err))
	}
	defer conn.Close()
	_, err = conn.Write([]byte(message))
}

// generateMetricString generates a string that can be used to send multiple metrics.
func generateMetricString(count int, prefix string) (metricString string) {
	rand.Seed(time.Now().UnixNano())

	for i := 0; i < count; i++ {
		metricString += fmt.Sprintf("test.%s:%d|c\n", prefix, rand.Intn(100))
	}

	return strings.Trim(metricString, "\n")
}
