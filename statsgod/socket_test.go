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
	"net"
	"strings"
	"time"
)

// Table for running the socket tests.
var testSockets = []struct {
	socketType     int
	socketDesc     string
	socketAddr     string
	badAddr        string
	socketMessages []string
}{
	{SocketTypeTcp, "tcp", "127.0.0.1:0", "0.0.0.0", []string{"test.tcp:4|c", "test.tcp:2|c\ntest.tcp:1|c"}},
	{SocketTypeUdp, "udp", "127.0.0.1:0", "", []string{"test.udp:4|c", "test.udp:2|c"}},
	{SocketTypeUnix, "unix", "/tmp/statsgod.sock", "/dev/null", []string{"test.unix:4|c", "test.unix:2|c\ntest.unix:1|c"}},
}

var sockets = make([]Socket, 3)
var parseChannel = make(chan string)
var logger = *CreateLogger(ioutil.Discard, ioutil.Discard, ioutil.Discard, ioutil.Discard)

var _ = BeforeSuite(func() {
	for i, ts := range testSockets {
		socket := CreateSocket(ts.socketType, ts.socketAddr)
		go socket.Listen(parseChannel, logger)
		BlockForSocket(socket, time.Second)
		sockets[i] = socket
	}
})

var _ = AfterSuite(func() {
	for i, _ := range testSockets {
		sockets[i].Close(logger)
	}
})

var _ = Describe("Sockets", func() {

	Describe("Testing the Socket interface", func() {
		It("should contain the required functions", func() {
			for i, _ := range testSockets {
				_, ok := sockets[i].(interface {
					Listen(parseChannel chan string, logger Logger)
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
		It("should be able to recieve messages", func() {
			for i, ts := range testSockets {
				for _, sm := range ts.socketMessages {
					sendSocketMessage(ts.socketDesc, sockets[i], sm)
				}

				for _, sm := range ts.socketMessages {
					receivedMessages := strings.Split(sm, "\n")
					for _, rm := range receivedMessages {
						message := ""
						select {
						case message = <-parseChannel:
						case <-time.After(5 * time.Second):
							message = ""
						}
						Expect(message).Should(Equal(rm))
					}
				}
			}
		})

		It("should panic if it has a bad address", func() {
			for _, ts := range testSockets {
				socket := CreateSocket(ts.socketType, "")
				Expect(func() { socket.Listen(parseChannel, logger) }).Should(Panic())
				socket = CreateSocket(ts.socketType, ts.badAddr)
				Expect(func() { socket.Listen(parseChannel, logger) }).Should(Panic())
			}
		})

		Measure("it should receive metrics quickly.", func(b Benchmarker) {
			runtime := b.Time("runtime", func() {
				for i, ts := range testSockets {
					for _, sm := range ts.socketMessages {
						sendSocketMessage(ts.socketDesc, sockets[i], sm)
						<-parseChannel
					}
				}
			})

			Expect(runtime.Seconds()).Should(BeNumerically("<", 0.5), "it should receive metrics quickly.")
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
