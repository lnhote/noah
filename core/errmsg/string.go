package errmsg

type Errmsg string

func (e Errmsg) Error() string {
	return string(e)
}

const (
	NoSuchCommand    = Errmsg("NoSuchCommand")
	ReplicateLogFail = Errmsg("ReplicateLogFail")
	ServerNotFound   = Errmsg("ServerNotFound")
)
