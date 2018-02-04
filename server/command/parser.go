package command

import (
	"fmt"
	"strings"
)

func (c Command) ToLog() (string, error) {
	switch c.CommandType {
	case CmdGet:
		return fmt.Sprintf("GET|%s", c.Key), nil
	case CmdSet:
		return fmt.Sprintf("SET|%s|%s", c.Key, c.Value), nil
	default:
		return "", fmt.Errorf("NoSuchCommand||%d", c.CommandType)
	}
}

func ParseCommand(rawCommand []byte) (*Command, error) {
	rawCommandStr := string(rawCommand)
	commandParts := strings.Split(rawCommandStr, " ")
	if len(commandParts) == 0 {
		return nil, fmt.Errorf("EmptyCommand||%s", rawCommandStr)
	}
	cmd := strings.ToLower(commandParts[0])
	if cmd == "get" {
		if len(commandParts) != 2 {
			return nil, fmt.Errorf("WrongCommand||%s", rawCommandStr)
		}
		return &Command{CommandType: CmdGet, Key: commandParts[1]}, nil
	}
	if cmd == "set" {
		if len(commandParts) != 3 {
			return nil, fmt.Errorf("WrongCommand||%s", rawCommandStr)
		}
		return &Command{CommandType: CmdSet, Key: commandParts[1], Value: commandParts[2]}, nil
	}
	return nil, fmt.Errorf("CommandNotSupported||%s", rawCommandStr)
}
