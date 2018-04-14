package entity

import (
	"fmt"

	"github.com/lnhote/noah/core/errmsg"
)

const (
	CmdSet = iota
	CmdGet
)

type Command struct {
	CommandType int
	Key         string
	Value       []byte
}

func (c Command) String() string {
	switch c.CommandType {
	case CmdGet:
		return fmt.Sprintf("Get|%s", c.Key)
	case CmdSet:
		return fmt.Sprintf("Set|%s|%s", c.Key, c.Value)
	default:
		return errmsg.NoSuchCommand.Error()
	}
}
