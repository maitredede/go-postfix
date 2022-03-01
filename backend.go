package postfix

type Backend interface {
	Lookup(client Client, dict string, key string) (ReplyType, string, error)
}
