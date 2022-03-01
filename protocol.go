package postfix

type ReplyType string

const (
	ReplyTypeOK       ReplyType = "OK"
	ReplyTypeTEMP     ReplyType = "TEMP"
	ReplyTypeNOTFOUND ReplyType = "NOTFOUND"
	ReplyTypeTIMEOUT  ReplyType = "TIMEOUT"
	ReplyTypePERM     ReplyType = "PERM"
)
