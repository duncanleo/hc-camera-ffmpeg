package hds

const (
	StatusSuccess               int64 = 0
	StatusOutOfMemory           int64 = 1
	StatusTimeout               int64 = 2
	StatusHeaderError           int64 = 3
	StatusPayloadError          int64 = 4
	StatusMissingProtocol       int64 = 5
	StatusProtocolSpecificError int64 = 6
)
