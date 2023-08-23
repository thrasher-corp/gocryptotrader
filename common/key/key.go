package key

import (
	"errors"
	"fmt"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var ErrExchangeNameIsEmpty = errors.New("exchange name is empty")

// ExchangePairAssetKey is a unique map key signature for exchange, currency pair and asset
type ExchangePairAssetKey struct {
	Exchange string
	Base     *currency.Item
	Quote    *currency.Item
	Asset    asset.Item
}

// PairAssetKey is a unique map key signature for currency pair and asset
type PairAssetKey struct {
	Base  *currency.Item
	Quote *currency.Item
	Asset asset.Item
}

// ExchangeCurrencyAssetKey is a unique map key signature for exchange, currency code and asset
type ExchangeCurrencyAssetKey struct {
	Exchange string
	Currency *currency.Item
	Asset    asset.Item
}

// SubAccountCurrencyAssetKey is a unique map key signature for subaccount, currency code and asset
type SubAccountCurrencyAssetKey struct {
	SubAccount string
	Currency   *currency.Item
	Asset      asset.Item
}

// GeneratePairAssetKey is a helper function to generate a unique map key
// and don't want to write validation yourself
// Note it's better to do the validation yourself and inline the declaration
// of this Key
func GeneratePairAssetKey(pair currency.Pair, item asset.Item) (PairAssetKey, error) {
	if pair.IsEmpty() {
		return PairAssetKey{}, currency.ErrCurrencyPairEmpty
	}
	if !item.IsValid() {
		return PairAssetKey{}, fmt.Errorf("%w %v", asset.ErrInvalidAsset, item)
	}
	return PairAssetKey{
		Base:  pair.Base.Item,
		Quote: pair.Quote.Item,
		Asset: item,
	}, nil
}

// GenerateExchangePairAssetKey is a helper function to generate a unique map key with an exchange name
// and don't want to write validation yourself
func GenerateExchangePairAssetKey(exch string, pair currency.Pair, item asset.Item) (ExchangePairAssetKey, error) {
	if pair.IsEmpty() {
		return ExchangePairAssetKey{}, currency.ErrCurrencyPairEmpty
	}
	if !item.IsValid() {
		return ExchangePairAssetKey{}, fmt.Errorf("%w %v", asset.ErrInvalidAsset, item)
	}
	if exch == "" {
		return ExchangePairAssetKey{}, ErrExchangeNameIsEmpty
	}
	return ExchangePairAssetKey{
		Exchange: strings.ToLower(exch),
		Base:     pair.Base.Item,
		Quote:    pair.Quote.Item,
		Asset:    item,
	}, nil
}

func GenerateExchangeCurrencyAssetKey(exch string, curr currency.Code, item asset.Item) (ExchangeCurrencyAssetKey, error) {
	if curr.IsEmpty() {
		return ExchangeCurrencyAssetKey{}, currency.ErrCurrencyPairEmpty
	}
	if !item.IsValid() {
		return ExchangeCurrencyAssetKey{}, fmt.Errorf("%w %v", asset.ErrInvalidAsset, item)
	}
	if exch == "" {
		return ExchangeCurrencyAssetKey{}, ErrExchangeNameIsEmpty
	}
	return ExchangeCurrencyAssetKey{
		Exchange: strings.ToLower(exch),
		Currency: curr.Item,
		Asset:    item,
	}, nil
}

// MatchesExchangeAsset checks if the key matches the exchange and asset
// used in Backtester funding statistics
func (k *ExchangePairAssetKey) MatchesExchangeAsset(exch string, item asset.Item) bool {
	return strings.ToLower(k.Exchange) == strings.ToLower(exch) && k.Asset == item
}

// MatchesPairAsset checks if the key matches the pair and asset
// used in Ticker and Orderbook when the exchange doesn't matter
func (k *ExchangePairAssetKey) MatchesPairAsset(pair currency.Pair, item asset.Item) bool {
	return k.Base == pair.Base.Item &&
		k.Quote == pair.Quote.Item &&
		k.Asset == item
}

func (k *ExchangePairAssetKey) MatchesExchange(exch string) bool {
	return strings.ToLower(k.Exchange) == strings.ToLower(exch)
}
