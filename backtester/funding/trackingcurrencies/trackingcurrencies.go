package trackingcurrencies

import (
	"errors"
	"fmt"
	"slices"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var (
	// ErrCurrencyContainsUSD is raised when the currency already contains a USD equivalent
	ErrCurrencyContainsUSD = errors.New("currency already contains a USD equivalent")
	// ErrCurrencyDoesNotContainUSD is raised when the currency does not contain a USD equivalent
	ErrCurrencyDoesNotContainUSD = errors.New("currency does not contain a USD equivalent")
	errNilPairs                  = errors.New("cannot assess with nil available pairs")
	errNoMatchingPairUSDFound    = errors.New("currency pair has no USD backed equivalent, cannot track price")
	errCurrencyNotFoundInPairs   = errors.New("currency does not exist in available pairs")
	errNoMatchingBaseUSDFound    = errors.New("base currency has no USD back equivalent, cannot track price")
	errNoMatchingQuoteUSDFound   = errors.New("quote currency has no USD back equivalent, cannot track price")
	errNilPairsReceived          = errors.New("nil tracking pairs received")
	errExchangeManagerRequired   = errors.New("exchange manager required")
)

// rankedUSDs is a slice of USD tracked currencies
// to allow for totals tracking across a backtesting run
var rankedUSDs = []currency.Code{
	currency.USDT,
	currency.BUSD,
	currency.USDC,
	currency.DAI,
	currency.USD,
	currency.TUSD,
	currency.ZUSD,
	currency.PAX,
}

// TrackingPair is basic pair data used
// to create more pairs based whether they contain
// a USD equivalent
type TrackingPair struct {
	Exchange string
	Asset    asset.Item
	Base     currency.Code
	Quote    currency.Code
}

// CreateUSDTrackingPairs is responsible for loading exchanges,
// ensuring the exchange have the latest currency pairs and
// if a pair doesn't have a USD currency to track price, to add those settings
func CreateUSDTrackingPairs(tp []TrackingPair, em *engine.ExchangeManager) ([]TrackingPair, error) {
	if len(tp) == 0 {
		return nil, errNilPairsReceived
	}
	if em == nil {
		return nil, errExchangeManagerRequired
	}

	var resp []TrackingPair
	for i := range tp {
		exch, err := em.GetExchangeByName(tp[i].Exchange)
		if err != nil {
			return nil, err
		}
		pair := currency.NewPair(tp[i].Base, tp[i].Quote)
		if pairContainsUSD(pair) {
			resp = append(resp, tp[i])
		} else {
			b := exch.GetBase()
			a := tp[i].Asset
			if a.IsFutures() {
				// futures matches to spot, not like this
				continue
			}
			pairs := b.CurrencyPairs.Pairs[a]
			basePair, quotePair, err := findMatchingUSDPairs(pair, pairs)
			if err != nil {
				return nil, err
			}
			resp = append(resp,
				tp[i],
				TrackingPair{
					Exchange: tp[i].Exchange,
					Asset:    tp[i].Asset,
					Base:     basePair.Base,
					Quote:    basePair.Quote,
				},
				TrackingPair{
					Exchange: tp[i].Exchange,
					Asset:    tp[i].Asset,
					Base:     quotePair.Base,
					Quote:    quotePair.Quote,
				},
			)
		}
	}
	return resp, nil
}

// CurrencyIsUSDTracked checks if the currency passed in
// tracks against USD value, ie is in rankedUSDs
func CurrencyIsUSDTracked(code currency.Code) bool {
	return slices.ContainsFunc(rankedUSDs, func(c currency.Code) bool {
		return c.Equal(code)
	})
}

// pairContainsUSD is a simple check to ensure that the currency pair
// has some sort of matching USD currency
func pairContainsUSD(pair currency.Pair) bool {
	return CurrencyIsUSDTracked(pair.Base) || CurrencyIsUSDTracked(pair.Quote)
}

// findMatchingUSDPairs will return a USD pair for both the base and quote currency provided
// this will allow for data retrieval and total tracking on backtesting runs
func findMatchingUSDPairs(pair currency.Pair, pairs *currency.PairStore) (basePair, quotePair currency.Pair, err error) {
	if pairs == nil {
		return currency.EMPTYPAIR, currency.EMPTYPAIR, errNilPairs
	}
	if pairContainsUSD(pair) {
		return currency.EMPTYPAIR, currency.EMPTYPAIR, ErrCurrencyContainsUSD
	}
	if !pairs.Available.Contains(pair, true) {
		return currency.EMPTYPAIR, currency.EMPTYPAIR, fmt.Errorf("%v %w", pair, errCurrencyNotFoundInPairs)
	}
	var baseFound, quoteFound bool

	for i := range rankedUSDs {
		if !baseFound && pairs.Available.Contains(currency.NewPair(pair.Base, rankedUSDs[i]), true) {
			baseFound = true
			basePair = currency.NewPair(pair.Base, rankedUSDs[i])
		}
		if !quoteFound && pairs.Available.Contains(currency.NewPair(pair.Quote, rankedUSDs[i]), true) {
			quoteFound = true
			quotePair = currency.NewPair(pair.Quote, rankedUSDs[i])
		}
	}
	if !baseFound {
		err = fmt.Errorf("%v %w", pair.Base, errNoMatchingBaseUSDFound)
	}
	if !quoteFound {
		err = fmt.Errorf("%v %w", pair.Quote, errNoMatchingQuoteUSDFound)
	}
	if !baseFound && !quoteFound {
		err = fmt.Errorf("%v %v %w", pair.Base, pair.Quote, errNoMatchingPairUSDFound)
	}
	return basePair, quotePair, err
}
