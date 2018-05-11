package errmsg

type Errmsg string

func (e Errmsg) Error() string {
	return string(e)
}

const (
	NoSuchCommand       = Errmsg("NoSuchCommand")
	CRCShortWrite       = Errmsg("CRCShortWrite")
	CRCError            = Errmsg("CRCError")
	ShortWrite          = Errmsg("ShortWrite")
	OpenWALError        = Errmsg("OpenWALError")
	EncodingRecordError = Errmsg("EncodingRecordError")
	ReplicateLogFail    = Errmsg("ReplicateLogFail")
	ServerNotFound      = Errmsg("ServerNotFound")
	EncodingError       = Errmsg("EncodingError")
	DecodeError         = Errmsg("DecodeError")
	DataTooShort        = Errmsg("DataTooShort")
	DBConnectErr        = Errmsg("DBConnectErr")
	EOF                 = Errmsg("EOF")
)
