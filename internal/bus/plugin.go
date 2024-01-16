package bus

type Info struct {
	Name string
}

type Plugin interface {
	Init(*MessagePipe)
	Close()
	Info() *Info
	Process(*Message)
	Subscriptions() []string
}
