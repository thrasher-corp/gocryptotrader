package key

import (
	"strings"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// ExchangePairAsset is a unique map key signature for exchange, currency pair and asset
type ExchangePairAsset struct {
	Exchange string
	Base     *currency.Item
	Quote    *currency.Item
	Asset    asset.Item
}

// NewExchangePairAssetKey is a helper to reduce the amount of code needed to create a key
func NewExchangePairAssetKey(exch string, a asset.Item, cp currency.Pair) ExchangePairAsset {
	return ExchangePairAsset{
		Exchange: strings.ToLower(exch),
		Base:     cp.Base.Item,
		Quote:    cp.Quote.Item,
		Asset:    a,
	}
}

// Pair combines the base and quote into a pair
func (k ExchangePairAsset) Pair() currency.Pair {
	if k.Base == nil && k.Quote == nil {
		return currency.EMPTYPAIR
	}
	return currency.NewPair(k.Base.Currency(), k.Quote.Currency())
}

// MatchesExchangeAsset checks if the key matches the exchange and asset
func (k ExchangePairAsset) MatchesExchangeAsset(exch string, item asset.Item) bool {
	return strings.EqualFold(k.Exchange, exch) && k.Asset == item
}

// MatchesPairAsset checks if the key matches the pair and asset
func (k ExchangePairAsset) MatchesPairAsset(pair currency.Pair, item asset.Item) bool {
	return k.Base == pair.Base.Item &&
		k.Quote == pair.Quote.Item &&
		k.Asset == item
}

// MatchesExchange checks if the exchange matches
func (k ExchangePairAsset) MatchesExchange(exch string) bool {
	return strings.EqualFold(k.Exchange, exch)
}

// ExchangeAsset is a unique map key signature for exchange and asset
type ExchangeAsset struct {
	Exchange string
	Asset    asset.Item
}

// PairAsset is a unique map key signature for currency pair and asset
type PairAsset struct {
	Base  *currency.Item
	Quote *currency.Item
	Asset asset.Item
}

// Pair combines the base and quote into a pair
func (k PairAsset) Pair() currency.Pair {
	if k.Base == nil && k.Quote == nil {
		return currency.EMPTYPAIR
	}
	return currency.NewPair(k.Base.Currency(), k.Quote.Currency())
}

// SubAccountAsset is a unique map key signature for subaccount and asset
type SubAccountAsset struct {
	SubAccount string
	Asset      asset.Item
}
