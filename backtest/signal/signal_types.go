package signal

type Direction uint8

const (
	BUY Direction = iota
	SELL
	HOLD
	EXIT
)

type Signal struct {

}