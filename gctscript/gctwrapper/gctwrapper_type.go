package gctwrapper

import "github.com/thrasher-corp/gocryptotrader/gctscript/gctwrapper/exchange"

// Wrapper struct
type Wrapper struct {
	*exchange.Exchange
}
