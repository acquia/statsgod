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
	"errors"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"
)

// tmpListener tracks a local dummy tcp connection.
var tmpListener net.Listener

// TestConnectionPoolStructure tests the ConnectionPool struct.
func TestConnectionPoolStructure(t *testing.T) {
	var pool = new(ConnectionPool)

	assert.NotNil(t, pool.Size)
	assert.NotNil(t, pool.Host)
	assert.NotNil(t, pool.Port)
	assert.NotNil(t, pool.Timeout)
	assert.NotNil(t, pool.ErrorCount)

	assert.Equal(t, len(pool.Connections), 0)
}

// startTemporaryListener starts a dummy tcp listener.
func startTemporaryListener(t *testing.T) int {
	// Temporarily listen for the test connection
	conn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	tmpListener = conn
	laddr := strings.Split(conn.Addr().String(), ":")
	if len(laddr) < 2 {
		t.Fatal(errors.New("Could not get port of listener."))
	}

	port, err := strconv.ParseInt(laddr[1], 10, 32)

	if err != nil {
		t.Fatal(errors.New("Could not get port of listener."))
	}

	return int(port)
}

// stopTemporaryListener stops the dummy tcp listener.
func stopTemporaryListener() {
	tmpListener.Close()
}

// TestConnectionPoolFactory tests that we can create ConnectionPool structs.
func TestConnectionPoolFactory(t *testing.T) {
	maxConnections := 5
	host := "127.0.0.1"
	port := startTemporaryListener(t)
	timeout := 1 * time.Second
	logger := *CreateLogger(ioutil.Discard, ioutil.Discard, ioutil.Discard, ioutil.Discard)
	pool, _ := CreateConnectionPool(maxConnections, host, int(port), timeout, logger)

	assert.Equal(t, pool.Size, maxConnections)
	assert.Equal(t, pool.Host, host)
	assert.Equal(t, pool.Port, port)
	assert.Equal(t, pool.Timeout, timeout)
	assert.Equal(t, pool.ErrorCount, 0)
	assert.Equal(t, cap(pool.Connections), maxConnections)
	assert.Equal(t, len(pool.Connections), maxConnections)

	stopTemporaryListener()
}

// TestGetReleaseConnection tests GetConnection and ReleaseConnection.
func TestGetReleaseConnection(t *testing.T) {
	maxConnections := 2
	host := "127.0.0.1"
	port := startTemporaryListener(t)
	timeout := 1 * time.Second
	logger := *CreateLogger(ioutil.Discard, ioutil.Discard, ioutil.Discard, ioutil.Discard)
	pool, _ := CreateConnectionPool(maxConnections, host, port, timeout, logger)

	// Check that we established the correct number of connections.
	assert.Equal(t, len(pool.Connections), maxConnections)

	// Check one out and ensure that the length of the channel changes.
	connOne, _ := pool.GetConnection(logger)
	assert.Equal(t, len(pool.Connections), maxConnections-1)

	// Check another one out and ensure that the length of the channel changes.
	connTwo, _ := pool.GetConnection(logger)
	assert.Equal(t, len(pool.Connections), maxConnections-2)

	// Test that we timeout if there are no available connections.
	_, err := pool.GetConnection(logger)
	assert.NotNil(t, err)

	// Release the connections and check that we are again at max connections.
	pool.ReleaseConnection(connOne, false, logger)
	pool.ReleaseConnection(connTwo, false, logger)
	assert.Equal(t, len(pool.Connections), maxConnections)

	// Test that we can recreate connections
	connThree, _ := pool.GetConnection(logger)
	connThree.Close()
	pool.ReleaseConnection(connThree, true, logger)
	assert.Equal(t, len(pool.Connections), maxConnections)

	// Test that we cannot create more connections than the pool allows.
	_, err = pool.CreateConnection(logger)
	assert.NotNil(t, err)
}
