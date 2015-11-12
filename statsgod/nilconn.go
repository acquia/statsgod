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

package statsgod

import (
	"net"
	"time"
)

// NilConnPanicMsg is the message that the NilConn implementation sends in panic.
const (
	NilConnPanicMsg = "Use of a nil connection."
)

// NilConn is an implementation of the net.Conn interface. We use it so that
// we can return a value from GetConnection. Without the NilConn, it would
// attempt to convert nil into a net.Conn which causes an error. In this case
// we can handle it gracefully and panic for any access instead of attemting
// to access a nil pointer.
type NilConn struct{}

// Read implements net.Conn.Read()
func (c NilConn) Read(b []byte) (n int, err error) {
	panic(NilConnPanicMsg)
}

// Write implements net.Conn.Write()
func (c NilConn) Write(b []byte) (n int, err error) {
	panic(NilConnPanicMsg)
}

// Close implements net.Conn.Close()
func (c NilConn) Close() error {
	panic(NilConnPanicMsg)
}

// LocalAddr implements net.Conn.LocalAddr()
func (c NilConn) LocalAddr() net.Addr {
	panic(NilConnPanicMsg)
}

// RemoteAddr implements net.Conn.RemoteAddr()
func (c NilConn) RemoteAddr() net.Addr {
	panic(NilConnPanicMsg)
}

// SetDeadline implements net.Conn.SetDeadline()
func (c NilConn) SetDeadline(t time.Time) error {
	panic(NilConnPanicMsg)
}

// SetReadDeadline implements net.Conn.SetReadDeadline()
func (c NilConn) SetReadDeadline(t time.Time) error {
	panic(NilConnPanicMsg)
}

// SetWriteDeadline implements net.Conn.SetWriteDeadline()
func (c NilConn) SetWriteDeadline(t time.Time) error {
	panic(NilConnPanicMsg)
}
