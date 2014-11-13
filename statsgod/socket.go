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

// Package statsgod - this library manages the different socket listeners
// that we use to collect metrics.
package statsgod

import (
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"syscall"
	"time"
)

// Enumeration of the socket types.
const (
	SocketTypeUdp = iota
	SocketTypeTcp
	SocketTypeUnix
)

// Socket is the interface for all of our socket types.
type Socket interface {
	Listen(parseChannel chan string, logger Logger)
	Close(logger Logger)
	GetAddr() string
	SocketIsActive() bool
}

// CreateSocket is a factory to create Socket structs.
func CreateSocket(socketType int, addr string) Socket {
	switch socketType {
	case SocketTypeUdp:
		l := new(SocketUdp)
		l.Addr = addr
		return l
	case SocketTypeTcp:
		l := new(SocketTcp)
		l.Addr = addr
		return l
	case SocketTypeUnix:
		l := new(SocketUnix)
		l.Addr = addr
		return l
	default:
		panic("Unknown socket type requested.")
	}
}

// BlockForSocket blocks until the specified socket is active.
func BlockForSocket(socket Socket, timeout time.Duration) {
	start := time.Now()
	for {
		if socket.SocketIsActive() == true {
			return
		}
		time.Sleep(time.Microsecond)
		if time.Since(start) > timeout {
			return
		}
	}
}

// SocketTcp contains the required fields to start a TCP socket.
type SocketTcp struct {
	Addr     string
	Listener net.Listener
}

// Listen listens on a socket and populates a channel with received messages.
// Conforms to Socket.Listen().
func (l *SocketTcp) Listen(parseChannel chan string, logger Logger) {
	if l.Addr == "" {
		panic("Could not establish a TCP socket. Address must be specified.")
	}
	listener, err := net.Listen("tcp", l.Addr)
	if err != nil {
		panic(fmt.Sprintf("Could not establish a TCP socket. %s", err))
	}
	l.Listener = listener

	logger.Info.Printf("TCP socket opened on %s", l.Addr)
	for {
		conn, err := listener.Accept()
		if err != nil {
			logger.Error.Println("Could not accept connection", err)
			return
		}
		go readInput(conn, parseChannel, logger)
	}
}

// Close closes an open socket. Conforms to Socket.Close().
func (l *SocketTcp) Close(logger Logger) {
	logger.Info.Println("Closing TCP socket.")
	l.Listener.Close()
}

// SocketIsActive determines if the socket is listening. Conforms to Socket.SocketIsActive()
func (l *SocketTcp) SocketIsActive() bool {
	return l.Listener != nil
}

// GetAddr retrieves a net compatible address string. Conforms to Socket.GetAddr().
func (l *SocketTcp) GetAddr() string {
	return l.Listener.Addr().String()
}

// SocketUdp contains the fields required to start a UDP socket.
type SocketUdp struct {
	Addr     string
	Listener *net.UDPConn
}

// Listen listens on a socket and populates a channel with received messages.
// Conforms to Socket.Listen().
func (l *SocketUdp) Listen(parseChannel chan string, logger Logger) {
	if l.Addr == "" {
		panic("Could not establish a UDP socket. Addr must be specified.")
	}
	addr, _ := net.ResolveUDPAddr("udp4", l.Addr)
	listener, err := net.ListenUDP("udp", addr)
	if err != nil {
		panic(fmt.Sprintf("Could not establish a UDP socket. %s", err))
	}
	l.Listener = listener

	logger.Info.Printf("UDP socket opened on %s", l.Addr)
	for {
		readInputUdp(*listener, parseChannel, logger)
	}
}

// Close closes an open socket. Conforms to Socket.Close().
func (l *SocketUdp) Close(logger Logger) {
	logger.Info.Println("Closing UDP socket.")
	l.Listener.Close()
}

// SocketIsActive determines if the socket is listening. Conforms to Socket.SocketIsActive()
func (l *SocketUdp) SocketIsActive() bool {
	return l.Listener != nil
}

// GetAddr retrieves a net compatible address string. Conforms to Socket.GetAddr().
func (l *SocketUdp) GetAddr() string {
	return l.Listener.LocalAddr().String()
}

// SocketUnix contains the fields required to start a Unix socket.
type SocketUnix struct {
	Addr     string
	Listener net.Listener
}

// Listen listens on a socket and populates a channel with received messages.
// Conforms to Socket.Listen().
func (l *SocketUnix) Listen(parseChannel chan string, logger Logger) {
	if l.Addr == "" {
		panic("Could not establish a Unix socket. No sock file specified.")
	}
	oldMask := syscall.Umask(0011)
	listener, err := net.Listen("unix", l.Addr)
	_ = syscall.Umask(oldMask)
	if err != nil {
		panic(fmt.Sprintf("Could not establish a Unix socket. %s", err))
	}
	l.Listener = listener
	logger.Info.Printf("Unix socket opened at %s", l.Addr)
	for {
		conn, err := l.Listener.Accept()
		if err != nil {
			logger.Error.Println("Could not accept connection", err)
			return
		}
		go readInput(conn, parseChannel, logger)
	}
}

// Close closes an open socket. Conforms to Socket.Close().
func (l *SocketUnix) Close(logger Logger) {
	defer os.Remove(l.Addr)
	logger.Info.Println("Closing Unix socket.")
	l.Listener.Close()
}

// SocketIsActive determines if the socket is listening. Conforms to Socket.SocketIsActive()
func (l *SocketUnix) SocketIsActive() bool {
	return l.Listener != nil
}

// GetAddr retrieves a net compatible address string. Conforms to Socket.GetAddr().
func (l *SocketUnix) GetAddr() string {
	return l.Addr
}

// readInput parses the buffer for TCP and Unix sockets.
func readInput(conn net.Conn, parseChannel chan string, logger Logger) {
	defer conn.Close()
	// readLength is the length of our buffer.
	readLength := 512
	// metricCount is how many metrics to parse per read.
	metricCount := 0
	// overflow tracks any messages that span two buffers.
	overflow := ""
	// buf is a reusable buffer for reading from the connection.
	buf := make([]byte, readLength)

	// Read data from the connection until it is closed.
	for {
		length, err := conn.Read(buf)
		if err != nil {
			// EOF will present as an error, but it could just signal a hangup.
			if err != io.EOF {
				logger.Error.Println("Could not read stream.", err)
			}
			break
		}
		conn.Write([]byte(""))
		if length == 0 {
			logger.Info.Println("Connection closed by peer.")
			break
		} else {
			// Check for multiple metrics delimited by a newline character.
			metrics := strings.Split(overflow+strings.TrimSpace(strings.Trim(string(buf), "\x00")), "\n")
			// If the buffer is full, the last metric likely has not fully been sent
			// yet. Try to parse it and if it throws an error, we'll prepend it to the
			// next connection read presuming that it will span to the next data read.
			metricCount = len(metrics)
			overflow = ""
			if length == readLength {
				// Attempt to parse the last metric. If that fails parse, we'll
				// reduce the size by one and save the overflow for the next read.
				_, err = ParseMetricString(metrics[len(metrics)-1])
				if err != nil {
					overflow = metrics[len(metrics)-1]
					metricCount = len(metrics) - 1
				}
			}

			// Send the metrics to be parsed.
			for i := 0; i < metricCount; i++ {
				parseChannel <- metrics[i]
			}
		}

		// Zero out the buffer for the next read.
		for b := range buf {
			buf[b] = 0
		}
	}
}

// readInputUdp parses the buffer for UDP sockets.
func readInputUdp(conn net.UDPConn, parseChannel chan string, logger Logger) {
	buf := make([]byte, 512)
	length, _, err := conn.ReadFromUDP(buf[0:])
	if err != nil {
		logger.Error.Println("Could not read stream.", err)
		return
	}
	if length != 0 {
		parseChannel <- string(buf[:length])
	}
}
