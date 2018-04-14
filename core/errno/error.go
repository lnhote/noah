package errno

const (
	// 200 - OK
	Success = 200

	// 300 – Ask someone else.
	NotLeader = 301

	// 400 – You fucked up.
	MissingParam      = 401
	InvalidParamValue = 402
	ServiceNotFound   = 404
	MethodNotAllowed  = 405
	RequestTimeout    = 408
	TooManyRequests   = 429

	// 500 – I fucked up.
	InternalServerError = 500
	ValueNotAccepcted   = 501
	ReplicateLogFail    = 502
	ServiceBusy         = 503
	GatewayTimeout      = 504
)
