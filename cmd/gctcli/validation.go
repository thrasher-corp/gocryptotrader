package main

import (
	"errors"
	"strings"

	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var (
	errInvalidPair     = errors.New("invalid currency pair supplied")
	errInvalidExchange = errors.New("invalid exchange supplied")
	errInvalidAsset    = errors.New("invalid asset supplied")
)

func validPair(pair string) bool {
	return strings.Contains(pair, pairDelimiter)
}

func validExchange(exch string) bool {
	return exchange.IsSupported(exch)
}

func validAsset(i string) bool {
	_, err := asset.New(i)
	return err == nil
}
