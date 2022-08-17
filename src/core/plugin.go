package core

type Plugin interface {
	Init(MessagePipeInterface)
	Close()
	Process(*Message)
	Info() *Info
	Subscriptions() []string
}
