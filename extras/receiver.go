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

// Package receiver is a sample receiver app.
package main

/**
 * Test receiver used to debug statsgod that listens on localhost:2003
 */

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
)

func main() {
	ln, err := net.Listen("tcp", ":2003")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Error: ", err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(c net.Conn) {
	// TODO - this isn't working for new lines?
	status, err := bufio.NewReader(c).ReadString('\n')
	if err != nil && err != io.EOF {
		fmt.Println("Error: ", err)
		os.Exit(1)
	}
	fmt.Printf("New client: %v\n%s", c.RemoteAddr(), status)

	c.Close()
	fmt.Printf("Connection from %v closed.\n", c.RemoteAddr())
}
