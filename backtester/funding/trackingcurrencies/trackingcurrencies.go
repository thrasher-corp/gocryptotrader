package trackingcurrencies

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/backtester/config"
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
	errNilPairs                = errors.New("cannot assess with nil available pairs")
	errNoMatchingPairUSDFound  = errors.New("currency pair has no USD back equivalent, cannot track price")
	errCurrencyNotFoundInPairs = errors.New("currency does not exist in available pairs")
	errNoMatchingBaseUSDFound  = errors.New("base currency has no USD back equivalent, cannot track price")
	errNoMatchingQuoteUSDFound = errors.New("quote currency has no USD back equivalent, cannot track price")
	errCurrencyContainsUSD     = errors.New("currency already contains a USD equivalent")
)

func DoEverything(cs []config.CurrencySettings) ([]config.CurrencySettings, error) {
	em := engine.SetupExchangeManager()
	var emm = make(map[string]exchange.IBotExchange)
	var resp []config.CurrencySettings
	// exchange, asset, base, quote
	var maperoo = make(map[string]map[string]map[string]map[string]config.CurrencySettings)
	var err error
	for i := range cs {
		if _, ok := maperoo[cs[i].ExchangeName]; !ok {
			maperoo[cs[i].ExchangeName] = make(map[string]map[string]map[string]config.CurrencySettings)
		}
		var exch exchange.IBotExchange
		var ok bool
		if exch, ok = emm[strings.ToLower(cs[i].ExchangeName)]; !ok {
			exch, err = em.NewExchangeByName(cs[i].ExchangeName)
			if err != nil {
				return nil, err
			}
			exch.SetDefaults()
			err = exch.UpdateTradablePairs(context.Background(), true)
			if err != nil {
				return nil, err
			}
			emm[exch.GetName()] = exch
		}

		if _, ok := maperoo[cs[i].ExchangeName][cs[i].Asset]; !ok {
			maperoo[cs[i].ExchangeName][cs[i].Asset] = make(map[string]map[string]config.CurrencySettings)
		}
		if _, ok := maperoo[cs[i].ExchangeName][cs[i].Asset][cs[i].Base]; !ok {
			maperoo[cs[i].ExchangeName][cs[i].Asset][cs[i].Base] = make(map[string]config.CurrencySettings)
		}
		pair, err := currency.NewPairFromStrings(cs[i].Base, cs[i].Quote)
		if err != nil {
			return nil, err
		}
		if PairContainsUSD(pair) {
			resp = append(resp, cs[i])
		} else {
			b := exch.GetBase()
			a, err := asset.New(cs[i].Asset)
			if err != nil {
				return nil, err
			}
			pairs := b.CurrencyPairs.Pairs[a]
			basePair, quotePair, err := FindMatchingUSDPairs(pair, pairs)
			if err != nil {
				return nil, err
			}
			resp = append(resp, cs[i])
			resp = append(resp, config.CurrencySettings{
				ExchangeName:      cs[i].ExchangeName,
				Asset:             cs[i].Asset,
				Base:              basePair.Base.String(),
				Quote:             basePair.Quote.String(),
				PriceTrackingOnly: true,
			})
			resp = append(resp, config.CurrencySettings{
				ExchangeName:      cs[i].ExchangeName,
				Asset:             cs[i].Asset,
				Base:              quotePair.Base.String(),
				Quote:             quotePair.Quote.String(),
				PriceTrackingOnly: true,
			})
		}
	}
	return resp, nil
}

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
		return currency.Pair{}, currency.Pair{}, errCurrencyContainsUSD
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
