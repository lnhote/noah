package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
)

func main() {
	serverAddr := flag.String("s", "localhost:8848", "server address")
	flag.Parse()
	// connect to server
	args := os.Args[1:]
	commandToSend := strings.Join(args, " ")
	fmt.Println("send message to server: ", commandToSend)
	net.Dial("tcp", *serverAddr)
	tcpAddr, err := net.ResolveTCPAddr("tcp", *serverAddr)
	if err != nil {
		println("ResolveTCPAddr failed:", err.Error())
		os.Exit(1)
	}

	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		println("Dial failed:", err.Error())
		os.Exit(1)
	}

	_, err = conn.Write([]byte(commandToSend))
	if err != nil {
		println("Write to server failed:", err.Error())
		os.Exit(1)
	}

	println("write to server = ", commandToSend)

	reply := make([]byte, 1024)
	_, err = conn.Read(reply)
	if err != nil {
		println("Write to server failed:", err.Error())
		os.Exit(1)
	}

	println("reply from server=", string(reply))
	conn.Close()
}
