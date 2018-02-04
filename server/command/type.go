package command

const (
	CmdSet = iota
	CmdGet
)

type Command struct {
	CommandType int
	Key         string
	Value       string
}
