package fee

import (
	"errors"
	"fmt"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var (
	// ErrExchangeNameEmpty is an error for when the exchange name is empty
	ErrExchangeNameEmpty = errors.New("exchange name is empty")
	// ErrFeeRateNotFound is an error for when the fee rate is not found
	ErrFeeRateNotFound = errors.New("fee rate not found")
	exchangeFees       = Fees{all: make(map[key.ExchangePairAsset]Rates)}
)

// Fees holds the fees for each exchange.
type Fees struct {
	all map[key.ExchangePairAsset]Rates
	mtx sync.RWMutex
}

// Rates holds the maker and taker fee rates for an exchange, pair and asset
type Rates struct {
	Maker float64
	Taker float64
}

// Load loads the percentage fee rate for a specific exchange, pair and asset
// type. Care must be taken as fee rates can be a rebate (negative) or zero as
// an introductory rate.
func Load(exch string, pair currency.Pair, a asset.Item, makerRate, takerRate float64) error {
	if exch == "" {
		return ErrExchangeNameEmpty
	}
	if pair.IsEmpty() {
		return currency.ErrCurrencyPairEmpty
	}
	if !a.IsValid() {
		return asset.ErrNotSupported
	}
	exchangeFees.mtx.Lock()
	exchangeFees.all[key.ExchangePairAsset{Exchange: exch, Base: pair.Base.Item, Quote: pair.Quote.Item, Asset: a}] = Rates{
		Maker: makerRate,
		Taker: takerRate,
	}
	exchangeFees.mtx.Unlock()
	return nil
}

// RetrievePercentageRates returns the fee for a specific exchange, pair and
// asset type
// TODO: Add credentials support to differentiate between keys.
func RetrievePercentageRates(exch string, pair currency.Pair, a asset.Item) (Rates, error) {
	if exch == "" {
		return Rates{}, ErrExchangeNameEmpty
	}
	if pair.IsEmpty() {
		return Rates{}, currency.ErrCurrencyPairEmpty
	}
	if !a.IsValid() {
		return Rates{}, fmt.Errorf("[%s]: %w", a, asset.ErrNotSupported)
	}

	rates, ok := exchangeFees.all[key.ExchangePairAsset{Exchange: exch, Base: pair.Base.Item, Quote: pair.Quote.Item, Asset: a}]
	if !ok {
		return Rates{}, fmt.Errorf("%w for %v %v %v", ErrFeeRateNotFound, exch, pair, a)
	}
	return rates, nil
}
