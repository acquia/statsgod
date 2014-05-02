package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
)

func main() {
	ln, err := net.Listen("tcp", ":5001")
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
	status, err := bufio.NewReader(c).ReadString('\n')
	if err != nil && err != io.EOF {
		fmt.Println("Error: ", err)
		os.Exit(1)
	}
	fmt.Println("From client: ", status)

	c.Close()
	fmt.Printf("Connection from %v closed.\n", c.RemoteAddr())
}
