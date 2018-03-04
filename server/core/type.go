package core

import "fmt"

const (
	CmdSet = iota
	CmdGet
)

type Command struct {
	CommandType int
	Key         string
	Value       []byte
}

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

type LogEntry struct {
	Command *Command
	Index   int
	Term    int
}

type ClientResponse struct {
	Code int
	Data map[string]interface{}
}
