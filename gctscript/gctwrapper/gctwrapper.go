package gctwrapper

import "github.com/thrasher-corp/gocryptotrader/gctscript/gctwrapper/exchange"

// Setup returns a Wrapper
func Setup() *Wrapper {
	return &Wrapper{
		&exchange.Exchange{},
	}
}
