package listener

type Listener interface {
	Listen() (chan listenerUpdatePrice, error)
	Description() string
	Close()
}
