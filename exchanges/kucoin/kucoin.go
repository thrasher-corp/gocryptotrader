package kucoin

import (
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// Kucoin is the overarching type across this package
type Kucoin struct {
	exchange.Base
}

const (
	kucoinAPIURL     = "https://api.kucoin.com"
	kucoinAPIVersion = "1"

	// Public endpoints

	// Authenticated endpoints
)

// Start implementing public and private exchange API funcs below
