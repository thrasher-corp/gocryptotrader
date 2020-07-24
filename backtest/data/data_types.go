package data

type Handler interface {
	Load([]string) error
	Reset() error

	Stream
}

type Stream interface {
	Next()
	Stream()
	History()
	Last()
	List()
}