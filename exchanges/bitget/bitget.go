package bitget

import (
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// Bitget is the overarching type across this package
type Bitget struct {
	exchange.Base
}

const (
	bitgetAPIURL     = ""
	bitgetAPIVersion = ""

	// Public endpoints

	// Authenticated endpoints
)

// Start implementing public and private exchange API funcs below

func (b *Bitget) SendAuthenticatedHTTPRequest() error {

	return nil
}
