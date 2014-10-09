/**
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
	"fmt"
	"net"
	"time"
)

// ConnectionPool maintains a channel of connections to a remote host.
type ConnectionPool struct {
	// Size indicates the number of connections to keep open.
	Size int
	// Connections is the channel to push new/reused connections onto.
	Connections chan net.Conn
	// Host is the remote connection hostname to connect to.
	Host string
	// Port is the remote connection port to connect to.
	Port int
	// Timeout is the amount of time to wait for a connection.
	Timeout time.Duration
	// ErrorCount tracks the number of connection errors that have occured.
	ErrorCount int
}

// CreateConnectionPool creates instances of ConnectionPool.
func CreateConnectionPool(size int, host string, port int, timeout time.Duration, logger Logger) (*ConnectionPool, error) {
	var pool = new(ConnectionPool)
	pool.Size = size
	pool.Host = host
	pool.Port = port
	pool.Timeout = timeout
	pool.ErrorCount = 0
	pool.Connections = make(chan net.Conn, size)

	errorCount := 0
	for i := 0; i < size; i++ {
		added, err := pool.CreateConnection(logger)
		if !added || err != nil {
			errorCount++
		}
	}

	if errorCount > 0 {
		err := fmt.Errorf("%d connections failed", errorCount)
		return pool, err
	}

	return pool, nil
}

// CreateConnection attempts to contact the remote relay host.
func (pool *ConnectionPool) CreateConnection(logger Logger) (bool, error) {

	if len(pool.Connections) < pool.Size {
		logger.Info.Printf("Connecting to %s on port %d", pool.Host, pool.Port)
		// Establish a new connection and set the timeout accordingly.
		conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", pool.Host, pool.Port))
		if err != nil {
			pool.ErrorCount++
			logger.Error.Println("Connection Error.", err)
			return false, err
		}
		conn.SetDeadline(time.Now().Add(pool.Timeout))
		pool.Connections <- conn
		return true, nil
	}

	err := errors.New("Attempt to add too many connections to the pool.")
	return false, err
}

// GetConnection retrieves a connection from the pool.
func (pool *ConnectionPool) GetConnection(logger Logger) (net.Conn, error) {
	select {
	case conn := <-pool.Connections:
		return conn, nil
	case <-time.After(pool.Timeout):
		logger.Error.Println("No connections available.")
		err := errors.New("Connection timeout.")
		return nil, err
	}
}

// ReleaseConnection releases a connection back to the pool.
func (pool *ConnectionPool) ReleaseConnection(conn net.Conn, recreate bool, logger Logger) (bool, error) {
	// recreate signifies that there was something wrong with the connection and
	// that we should make a new one.
	if recreate {
		conn.Close()
		added, err := pool.CreateConnection(logger)
		if !added || err != nil {
			logger.Error.Println("Could not release connection.", err)
			return false, err
		}
		return true, nil
	}

	// Reset the timeout and put it back on the channel.
	conn.SetDeadline(time.Now().Add(pool.Timeout))
	pool.Connections <- conn
	return true, nil
}
