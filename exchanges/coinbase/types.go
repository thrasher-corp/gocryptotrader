package coinbase

import (
	"maps"
	"slices"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Product types, which select products in requests and identify them in responses
const (
	ProductTypeSpot   = "SPOT"
	ProductTypeFuture = "FUTURE"
	// ProductTypeEquity selects US stocks and ETFs, which trade against USDC in whole or fractional shares
	ProductTypeEquity = "EQUITY"
)

// Product venues, where a product trades
const (
	// ProductVenueCBE is Coinbase Exchange, where spot trades
	ProductVenueCBE = "CBE"
	// ProductVenueFCM is Coinbase Financial Markets, the futures commission merchant US derivatives trade through
	ProductVenueFCM = "FCM"
	// ProductVenueINTX is Coinbase International Exchange, whose perpetuals stopped trading through Advanced Trade on 1
	// October 2026
	ProductVenueINTX = "INTX"
	// ProductVenueDeribit lists the Global Derivatives gateway's instruments, which Advanced Trade only lists; they
	// trade through GlobalDerivatives
	ProductVenueDeribit = "DERIBIT"
)

// Order sides
const (
	OrderSideBuy  = "BUY"
	OrderSideSell = "SELL"
)

// Amount is an amount of a currency
type Amount struct {
	Value    types.Number  `json:"value"`
	Currency currency.Code `json:"currency"`
	// CBRN names the currency as a Coinbase resource name, which convert and US derivatives responses send
	CBRN string `json:"cbrn"`
}

// pairAliases holds the products Coinbase treats as one: a USDC pair other than USDT-USDC and EURC-USDC trades on the
// same book as its USD pair, so the websocket streams it under the USD product's ID
type pairAliases struct {
	associatedAliases map[currency.Pair]currency.Pairs
	m                 sync.RWMutex
}

// aliasKey returns the key a pair's aliases are held under, whatever its delimiter
func aliasKey(p currency.Pair) currency.Pair {
	return currency.Pair{Base: p.Base.Upper(), Quote: p.Quote.Upper()}
}

// GetAlias returns the pairs that are the same product as a pair, the pair itself among them
func (a *pairAliases) GetAlias(p currency.Pair) currency.Pairs {
	a.m.RLock()
	defer a.m.RUnlock()
	return slices.Clone(a.associatedAliases[aliasKey(p)])
}

// GetAliases returns every pair's aliases
func (a *pairAliases) GetAliases() map[currency.Pair]currency.Pairs {
	a.m.RLock()
	defer a.m.RUnlock()
	return maps.Clone(a.associatedAliases)
}

// canonical returns the pair whose product a pair streams as: the USD pair a USDC alias shares its book with, or the
// pair itself. It reports false until product data has loaded the aliases
func (a *pairAliases) canonical(p currency.Pair) (currency.Pair, bool) {
	a.m.RLock()
	defer a.m.RUnlock()
	if len(a.associatedAliases) == 0 {
		return p, false
	}
	for product, aliases := range a.associatedAliases {
		if !product.Equal(p) && aliases.Contains(p, true) {
			return product, true
		}
	}
	return p, true
}

// Load adds aliases to the pairs they belong to
func (a *pairAliases) Load(aliases map[currency.Pair]currency.Pairs) {
	a.m.Lock()
	defer a.m.Unlock()
	if a.associatedAliases == nil {
		a.associatedAliases = make(map[currency.Pair]currency.Pairs)
	}
	for k, v := range aliases {
		k = aliasKey(k)
		a.associatedAliases[k] = a.associatedAliases[k].Add(v...)
	}
}
