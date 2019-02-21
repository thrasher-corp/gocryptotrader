package currency

// TradeSupport defines a bitmask for what an exchange can do
type TradeSupport uint8

// Conts here define market exchange trade support structure
const (
	Offline       TradeSupport = 0
	SpotTrading   TradeSupport = 1 << 0
	PerpetualSwap TradeSupport = 1 << 1
	Contracts     TradeSupport = 1 << 2
	Other         TradeSupport = 1 << 3
)

// Market defines a cryptocurrency exchange market
type Market struct {
	ID      int64
	Name    string
	Support TradeSupport
}
