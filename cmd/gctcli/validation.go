package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
)

var (
	errInvalidPair  = errors.New("invalid currency pair supplied")
	errInvalidAsset = errors.New("invalid asset supplied")
)

func validPair(pair string) bool {
	return strings.Contains(pair, pairDelimiter)
}

func validAsset(i string) bool {
	_, err := asset.New(i)
	return err == nil
}

func isFuturesAsset(a string) error {
	i, err := asset.New(a)
	if err != nil {
		return err
	}
	if !i.IsFutures() {
		return fmt.Errorf("%w %q", futures.ErrNotFuturesAsset, a)
	}
	return nil
}
