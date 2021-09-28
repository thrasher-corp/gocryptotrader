package trackingcurrencies

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	gctconfig "github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
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

var (
	errNilPairs                   = errors.New("cannot assess with nil available pairs")
	errNoMatchingPairUSDFound     = errors.New("currency pair has no USD backed equivalent, cannot track price")
	errCurrencyNotFoundInPairs    = errors.New("currency does not exist in available pairs")
	errNoMatchingBaseUSDFound     = errors.New("base currency has no USD back equivalent, cannot track price")
	errNoMatchingQuoteUSDFound    = errors.New("quote currency has no USD back equivalent, cannot track price")
	ErrCurrencyContainsUSD        = errors.New("currency already contains a USD equivalent")
	ErrCurrencyDoesNotContainsUSD = errors.New("currency does not contains a USD equivalent")
)

type TrackingPair struct {
	Exchange string
	Asset    string
	Base     string
	Quote    string
}

// CreateUSDTrackingPairs is responsible for loading exchanges,
// ensuring the exchange have the latest currency pairs and
// if a pair doesn't have a USD currency to track price, to add those settings
func CreateUSDTrackingPairs(tp []TrackingPair) ([]TrackingPair, error) {
	em := engine.SetupExchangeManager()
	var emm = make(map[string]exchange.IBotExchange)
	var wg sync.WaitGroup
	var err error
	for i := range tp {
		emm[tp[i].Exchange] = nil
	}
	wg.Add(len(emm))
	for k := range emm {
		go func(key string) {
			defer wg.Done()
			var exch exchange.IBotExchange
			exch, err = em.NewExchangeByName(key)
			if err != nil {
				return
			}
			var conf *gctconfig.ExchangeConfig
			conf, err = exch.GetDefaultConfig()
			if err != nil {
				return
			}
			exch.SetDefaults()
			err = exch.Setup(conf)
			if err != nil {
				return
			}
			err = exch.UpdateTradablePairs(context.Background(), true)
			if err != nil {
				return
			}
			emm[key] = exch
		}(k)
	}
	wg.Wait()

	var resp []TrackingPair
	for i := range tp {
		exch := emm[strings.ToLower(tp[i].Exchange)]
		if exch == nil {
			return nil, fmt.Errorf("%v %w", tp[i].Exchange, engine.ErrExchangeNotFound)
		}
		pair, err := currency.NewPairFromStrings(tp[i].Base, tp[i].Quote)
		if err != nil {
			return nil, err
		}
		if PairContainsUSD(pair) {
			resp = append(resp, tp[i])
		} else {
			b := exch.GetBase()
			a, err := asset.New(tp[i].Asset)
			if err != nil {
				return nil, err
			}
			pairs := b.CurrencyPairs.Pairs[a]
			basePair, quotePair, err := FindMatchingUSDPairs(pair, pairs)
			if err != nil {
				return nil, err
			}
			resp = append(resp, tp[i])
			resp = append(resp, TrackingPair{
				Exchange: tp[i].Exchange,
				Asset:    tp[i].Asset,
				Base:     basePair.Base.String(),
				Quote:    basePair.Quote.String(),
			})
			resp = append(resp, TrackingPair{
				Exchange: tp[i].Exchange,
				Asset:    tp[i].Asset,
				Base:     quotePair.Base.String(),
				Quote:    quotePair.Quote.String(),
			})
		}
	}
	return resp, nil
}

// CurrencyIsUSDTracked checks if the currency passed in
// tracks against USD value, ie is in rankedUSDs
func CurrencyIsUSDTracked(code currency.Code) bool {
	for i := range rankedUSDs {
		if code == rankedUSDs[i] {
			return true
		}
	}
	return false
}

// PairContainsUSD is a simple check to ensure that the currency pair
// has some sort of matching USD currency
func PairContainsUSD(pair currency.Pair) bool {
	for i := range rankedUSDs {
		if rankedUSDs[i] == pair.Base {
			return true
		}
		if rankedUSDs[i] == pair.Quote {
			return true
		}
	}
	return false
}

// FindMatchingUSDPairs will return a USD pair for both the base and quote currency provided
// this will allow for data retrieval and total tracking on backtesting runs
func FindMatchingUSDPairs(pair currency.Pair, pairs *currency.PairStore) (basePair currency.Pair, quotePair currency.Pair, err error) {
	if pairs == nil {
		return currency.Pair{}, currency.Pair{}, errNilPairs
	}
	if PairContainsUSD(pair) {
		return currency.Pair{}, currency.Pair{}, ErrCurrencyContainsUSD
	}
	if !pairs.Available.Contains(pair, true) {
		return currency.Pair{}, currency.Pair{}, fmt.Errorf("%v %w", pair, errCurrencyNotFoundInPairs)
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
