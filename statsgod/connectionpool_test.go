/**
 * Copyright 2015 Acquia, Inc.
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
	"strconv"
	"strings"
	"time"
)

var _ = Describe("Connection Pool", func() {
	var (
		tmpPort        int
		logger         Logger
		maxConnections int           = 2
		host           string        = "127.0.0.1"
		timeout        time.Duration = 1 * time.Second
	)

	Describe("Testing the basic structure", func() {
		It("should contain values", func() {
			var pool = new(ConnectionPool)
			Expect(pool.Size).ShouldNot(Equal(nil))
			Expect(pool.Addr).ShouldNot(Equal(nil))
			Expect(pool.Timeout).ShouldNot(Equal(nil))
			Expect(pool.ErrorCount).ShouldNot(Equal(nil))

			Expect(len(pool.Connections)).Should(Equal(0))

		})
	})

	Describe("Testing the connection pool functionality", func() {
		BeforeEach(func() {
			logger = *CreateLogger(ioutil.Discard, ioutil.Discard, ioutil.Discard, ioutil.Discard)
			tmpPort = StartTemporaryListener()
		})

		AfterEach(func() {
			StopTemporaryListener()
		})

		Context("when we create a new connection pool", func() {
			It("should contain values", func() {
				addr := fmt.Sprintf("%s:%d", host, tmpPort)
				pool, _ := CreateConnectionPool(maxConnections, addr, ConnPoolTypeTcp, timeout, logger)
				Expect(pool.Size).Should(Equal(maxConnections))
				Expect(pool.Addr).Should(Equal(addr))
				Expect(pool.Timeout).Should(Equal(timeout))
				Expect(pool.ErrorCount).Should(Equal(0))
				Expect(cap(pool.Connections)).Should(Equal(maxConnections))
				Expect(len(pool.Connections)).Should(Equal(maxConnections))
			})

			// Test that we get an error if there is no listener.
			It("should throw an error if there is no listener", func() {
				addr := fmt.Sprintf("%s:%d", host, tmpPort)
				StopTemporaryListener()
				_, errTcp := CreateConnectionPool(maxConnections, addr, ConnPoolTypeTcp, timeout, logger)
				Expect(errTcp).ShouldNot(Equal(nil))

				_, errUnix := CreateConnectionPool(maxConnections, "/dev/null", ConnPoolTypeUnix, timeout, logger)
				Expect(errUnix).ShouldNot(BeNil())

				_, errType := CreateConnectionPool(maxConnections, "/dev/null", ConnPoolTypeNone, timeout, logger)
				Expect(errType).ShouldNot(BeNil())
			})

		})

		Context("when we use the connection pool", func() {
			It("should contain the right number of connections", func() {
				addr := fmt.Sprintf("%s:%d", host, tmpPort)
				pool, _ := CreateConnectionPool(maxConnections, addr, ConnPoolTypeTcp, timeout, logger)

				// Check that we established the correct number of connections.
				Expect(len(pool.Connections)).Should(Equal(maxConnections))

				// Check one out and ensure that the length of the channel changes.
				connOne, _ := pool.GetConnection(logger)
				Expect(len(pool.Connections)).Should(Equal(maxConnections - 1))

				// Check another one out and ensure that the length of the channel changes.
				connTwo, _ := pool.GetConnection(logger)
				Expect(len(pool.Connections)).Should(Equal(maxConnections - 2))

				// Test that we timeout if there are no available connections.
				_, err := pool.GetConnection(logger)
				Expect(err).ShouldNot(Equal(nil))

				// Release the connections and check that we are again at max connections.
				pool.ReleaseConnection(connOne, false, logger)
				pool.ReleaseConnection(connTwo, false, logger)
				Expect(len(pool.Connections)).Should(Equal(maxConnections))

				// Test that we can recreate connections
				connThree, _ := pool.GetConnection(logger)
				connThree.Close()
				pool.ReleaseConnection(connThree, true, logger)
				Expect(len(pool.Connections)).Should(Equal(maxConnections))

				// Test that we cannot create more connections than the pool allows.
				_, err = pool.CreateConnection(logger)
				Expect(err).ShouldNot(Equal(nil))
			})

			It("should throw an error if there is no listener.", func() {
				addr := fmt.Sprintf("%s:%d", host, tmpPort)
				pool, _ := CreateConnectionPool(maxConnections, addr, ConnPoolTypeTcp, timeout, logger)
				StopTemporaryListener()

				// Test that we get an error if there is no listener.
				badConnection, _ := pool.GetConnection(logger)
				_, releaseErr := pool.ReleaseConnection(badConnection, true, logger)
				Expect(releaseErr).ShouldNot(Equal(nil))
			})

		})
	})
})

// tmpListener tracks a local dummy tcp connection.
var tmpListener net.Listener

// StartTemporaryListener starts a dummy tcp listener.
func StartTemporaryListener() int {
	// @todo: move this to a setup/teardown (Issue #29)
	// Temporarily listen for the test connection
	conn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	tmpListener = conn
	laddr := strings.Split(conn.Addr().String(), ":")
	if len(laddr) < 2 {
		panic("Could not get port of listener.")
	}

	port, err := strconv.ParseInt(laddr[1], 10, 32)

	if err != nil {
		panic("Could not get port of listener.")
	}

	return int(port)
}

// StopTemporaryListener stops the dummy tcp listener.
func StopTemporaryListener() {
	if tmpListener != nil {
		tmpListener.Close()
	}
}
