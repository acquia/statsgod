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
	"net"
	"os"
	"strings"
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
}

// CreateSocket is a factory to create Socket structs.
func CreateSocket(socketType int) Socket {
	switch socketType {
	case SocketTypeUdp:
		return new(SocketUdp)
	case SocketTypeTcp:
		return new(SocketTcp)
	case SocketTypeUnix:
		return new(SocketUnix)
	default:
		panic("Unknown socket type requested.")
	}
}

// SocketTcp contains the required fields to start a TCP socket.
type SocketTcp struct {
	Host string
	Port int
}

// Listen conforms to the Socket.Listen() interface.
func (l SocketTcp) Listen(parseChannel chan string, logger Logger) {
	if l.Host == "" || l.Port == 0 {
		panic("Could not establish a TCP socket. Host and port must be specified.")
	}
	addr := fmt.Sprintf("%s:%d", l.Host, l.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		panic(fmt.Sprintf("Could not establish a TCP socket. %s", err))
	}

	logger.Info.Printf("TCP socket opened on %s", addr)
	for {
		conn, err := listener.Accept()
		if err != nil {
			logger.Error.Println("Could not accept connection", err)
			return
		}
		go readInput(conn, parseChannel, logger)
	}
}

// SocketUdp contains the fields required to start a UDP socket.
type SocketUdp struct {
	Host string
	Port int
}

// Listen conforms to the Socket.Listen() interface.
func (l SocketUdp) Listen(parseChannel chan string, logger Logger) {
	if l.Host == "" || l.Port == 0 {
		panic("Could not establish a UDP socket. Host and port must be specified.")
	}
	addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", l.Host, l.Port))
	listener, err := net.ListenUDP("udp", addr)
	if err != nil {
		panic(fmt.Sprintf("Could not establish a UDP socket. %s", err))
	}

	logger.Info.Printf("UDP socket opened on %s", addr)
	for {
		readInputUdp(*listener, parseChannel, logger)
	}
}

// SocketUnix contains the fields required to start a Unix socket.
type SocketUnix struct {
	Sock string
}

// Listen conforms to the Socket.Listen() interface.
func (l SocketUnix) Listen(parseChannel chan string, logger Logger) {
	if l.Sock == "" {
		panic("Could not establish a Unix socket. No sock file specified.")
	}
	listener, err := net.Listen("unix", l.Sock)
	defer os.Remove(l.Sock)
	if err != nil {
		panic(fmt.Sprintf("Could not establish a Unix socket. %s", err))
	}
	logger.Info.Printf("Unix socket opened at %s", l.Sock)
	for {
		conn, err := listener.Accept()
		if err != nil {
			logger.Error.Println("Could not accept connection", err)
			return
		}
		go readInput(conn, parseChannel, logger)
	}
}

// readInput parses the buffer for TCP and Unix sockets.
func readInput(conn net.Conn, parseChannel chan string, logger Logger) {
	defer conn.Close()
	// Read the data from the connection.
	buf := make([]byte, 512)
	_, err := conn.Read(buf)
	if err != nil {
		logger.Error.Println("Could not read stream.", err)
		return
	}
	if len(string(buf)) != 0 {
		parseChannel <- strings.TrimSpace(strings.Trim(string(buf), "\x00"))
	}
}

// readInputUdp parses the buffer for UDP sockets.
func readInputUdp(conn net.UDPConn, parseChannel chan string, logger Logger) {
	buf := make([]byte, 512)
	_, _, err := conn.ReadFromUDP(buf[0:])
	if err != nil {
		logger.Error.Println("Could not read stream.", err)
		return
	}
	parseChannel <- strings.TrimSpace(strings.Trim(string(buf), "\x00"))
}
