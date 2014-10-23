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
	"time"
)

var _ = Describe("Sockets", func() {
	// Table for running the socket tests.
	var testSockets = []struct {
		socketType    int
		socketDesc    string
		socketAddr    string
		badAddr       string
		socketMessage string
	}{
		{SocketTypeTcp, "tcp", "127.0.0.1:0", "0.0.0.0", "test.tcp:4|c"},
		{SocketTypeUdp, "udp", "127.0.0.1:0", "", "test.udp:4|c"},
		{SocketTypeUnix, "unix", "/tmp/statsgod.sock", "/dev/null", "test.unix:4|c"},
	}

	Describe("Testing the Socket interface", func() {
		It("should contain the required functions", func() {
			for _, ts := range testSockets {
				socket := CreateSocket(ts.socketType, ts.socketAddr)
				_, ok := socket.(interface {
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
		parseChannel := make(chan string)
		logger := *CreateLogger(ioutil.Discard, ioutil.Discard, ioutil.Discard, ioutil.Discard)
		It("should be able to recieve messages", func() {
			for _, ts := range testSockets {
				socket := CreateSocket(ts.socketType, ts.socketAddr)
				go socket.Listen(parseChannel, logger)
				BlockForSocket(socket, time.Second)
				sendSocketMessage(ts.socketDesc, socket, ts.socketMessage)
				message := ""
				select {
				case message = <-parseChannel:
				case <-time.After(10 * time.Second):
					message = ""
				}
				Expect(message).Should(Equal(ts.socketMessage))
				socket.Close(logger)
			}
		})

		It("should be able to block until timeout", func() {
			socket := CreateSocket(SocketTypeTcp, "127.0.0.1:0")
			BlockForSocket(socket, time.Microsecond)
		})

		It("should panic if it has a bad address", func() {
			for _, ts := range testSockets {
				socket := CreateSocket(ts.socketType, "")
				Expect(func() { socket.Listen(parseChannel, logger) }).Should(Panic())
				socket = CreateSocket(ts.socketType, ts.badAddr)
				Expect(func() { socket.Listen(parseChannel, logger) }).Should(Panic())
			}
		})

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
