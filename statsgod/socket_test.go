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
	"reflect"
	"testing"
)

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

func TestCreateSocket(t *testing.T) {
	// Test the UDP socket and required fields.
	socketUdpInterface := CreateSocket(SocketTypeUdp)
	ensureSocketInterface(socketUdpInterface, t)
	socketUdp := socketUdpInterface.(*SocketUdp)
	assert.Equal(t, reflect.TypeOf(socketUdp).String(), "*statsgod.SocketUdp")
	assert.NotNil(t, socketUdp.Host)
	assert.NotNil(t, socketUdp.Port)

	// Test the TCP socket and required fields.
	socketTcpInterface := CreateSocket(SocketTypeTcp)
	ensureSocketInterface(socketTcpInterface, t)
	socketTcp := socketTcpInterface.(*SocketTcp)
	assert.NotNil(t, socketTcp)
	assert.Equal(t, reflect.TypeOf(socketTcp).String(), "*statsgod.SocketTcp")
	assert.NotNil(t, socketTcp.Host)
	assert.NotNil(t, socketTcp.Port)

	// Test the Unix socket and required fields.
	socketUnixInterface := CreateSocket(SocketTypeUnix)
	ensureSocketInterface(socketUnixInterface, t)
	socketUnix := socketUnixInterface.(*SocketUnix)
	assert.NotNil(t, socketUnix)
	assert.Equal(t, reflect.TypeOf(socketUnix).String(), "*statsgod.SocketUnix")
	assert.NotNil(t, socketUnix.Sock)
}
