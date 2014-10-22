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

package statsgod

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net"
	"testing"
	"time"
)

// Table for running the socket tests.
var testSockets = []struct {
	socketType    int
	socketDesc    string
	socketAddr    string
	socketMessage string
}{
	{SocketTypeTcp, "tcp", "127.0.0.1:0", "test.tcp:4|c"},
	{SocketTypeUdp, "udp", "127.0.0.1:0", "test.udp:4|c"},
	{SocketTypeUnix, "unix", "/tmp/statsgod.sock", "test.unix:4|c"},
}

// ensureSocketInterface Tests that each Socket conforms to the interface.
func ensureSocketInterface(socketInterface Socket, t *testing.T) {
	// This tests that it conforms to the Socket interface.
	if s, ok := socketInterface.(interface {
		Listen(parseChannel chan string, logger Logger)
	}); ok {
		// If it contains the Listen function, we are good.
		assert.NotNil(t, s)
	} else {
		// Else we need to implement Listen().
		assert.NotNil(t, nil)
	}
}

// TestCreateSocket will create each socket type and ensure that it conforms to
// the Socket interface.
func TestCreateSocket(t *testing.T) {
	for _, ts := range testSockets {
		socket := CreateSocket(ts.socketType, ts.socketAddr)
		ensureSocketInterface(socket, t)
	}
}

// TestListenClose is a functional test that we can listen, receive and close on our sockets.
func TestListenClose(t *testing.T) {
	parseChannel := make(chan string)
	logger := *CreateLogger(ioutil.Discard, ioutil.Discard, ioutil.Discard, ioutil.Discard)

	for _, ts := range testSockets {
		socket := CreateSocket(ts.socketType, ts.socketAddr)
		assert.NotNil(t, socket)
		go socket.Listen(parseChannel, logger)
		BlockForSocket(socket, time.Second)
		sendSocketMessage(ts.socketDesc, socket, ts.socketMessage, t)
		message := ""
		select {
		case message = <-parseChannel:
		case <-time.After(10 * time.Second):
			message = ""
		}
		assert.Equal(t, message, ts.socketMessage)
		socket.Close(logger)
	}
}

// sendSocketMessage will connect to a specified socket type send a message and close.
func sendSocketMessage(socketType string, socket Socket, message string, t *testing.T) {
	conn, err := net.Dial(socketType, socket.GetAddr())
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer conn.Close()
	_, err = conn.Write([]byte(message))
}
