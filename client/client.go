package main

import (
	"flag"
	"fmt"
	"net/rpc"
	"os"
	"strings"

	"github.com/lnhote/noah/core/entity"
	"github.com/lnhote/noah/core/raftrpc"
)

func main() {
	serverAddr := flag.String("s", "127.0.0.1:8851", "server address")
	flag.Parse()
	// connect to server
	args := os.Args[1:]
	fmt.Printf("send message to server %s: %s\n", *serverAddr, strings.Join(args, " "))

	//client, err := rpc.DialHTTP("tcp", *serverAddr)
	//if err != nil {
	//	println("Dial failed:", err.Error())
	//	os.Exit(1)
	//}

	var client, err = rpc.Dial("tcp", *serverAddr)
	if err != nil {
		println("Dial failed:", err.Error())
	}

	var resp raftrpc.ClientResponse
	cmd, err := parseCommand(args)
	if err != nil {
		fmt.Printf("Unkown Command: %s\n", args[0])
		os.Exit(1)
	}
	var method string
	switch cmd.CommandType {
	case entity.CmdGet:
		method = "NoahCommandServer.Get"
	case entity.CmdSet:
		method = "NoahCommandServer.Set"
	default:
		fmt.Printf("Unkown Command: %s\n", args[0])
		os.Exit(1)
	}
	err = client.Call(method, cmd, &resp)
	if err != nil {
		fmt.Printf("%s Fail: %s", method, err.Error())
		os.Exit(1)
	}
	fmt.Printf("%s Success: %+v\n", method, resp)
	os.Exit(0)
}

func parseCommand(commandParts []string) (*entity.Command, error) {
	if len(commandParts) == 0 {
		return nil, fmt.Errorf("EmptyCommand")
	}
	rawCommandStr := strings.Join(commandParts, " ")
	cmd := strings.ToLower(commandParts[0])
	if cmd == "get" {
		if len(commandParts) != 2 {
			return nil, fmt.Errorf("WrongCommand||%s", rawCommandStr)
		}
		return &entity.Command{CommandType: entity.CmdGet, Key: commandParts[1]}, nil
	}
	if cmd == "set" {
		if len(commandParts) != 3 {
			return nil, fmt.Errorf("WrongCommand||%s", rawCommandStr)
		}
		return &entity.Command{CommandType: entity.CmdSet, Key: commandParts[1], Value: []byte(commandParts[2])}, nil
	}
	return nil, fmt.Errorf("CommandNotSupported||%s", rawCommandStr)
}
