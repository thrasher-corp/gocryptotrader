package main

import (
	"errors"
	"strings"

	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

var (
	errInvalidPair     = errors.New("invalid currency pair supplied")
	errInvalidExchange = errors.New("invalid exchange supplied")
)

func validPair(pair string) bool {
	return strings.Contains(pair, pairDelimiter)
}

func validExchange(exch string) bool {
	return exchange.IsSupported(exch)
}
