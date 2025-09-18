package key

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// ExchangeAssetPair is a unique map key signature for exchange, currency pair and asset
type ExchangeAssetPair struct {
	Exchange string
	Asset    asset.Item
	Base     *currency.Item
	Quote    *currency.Item
}

// NewExchangeAssetPair is a helper function to expand a Pair into an ExchangeAssetPair
func NewExchangeAssetPair(exch string, a asset.Item, cp currency.Pair) ExchangeAssetPair {
	return ExchangeAssetPair{
		Exchange: exch,
		Base:     cp.Base.Item,
		Quote:    cp.Quote.Item,
		Asset:    a,
	}
}

// Pair combines the base and quote into a pair
func (k ExchangeAssetPair) Pair() currency.Pair {
	return currency.NewPair(k.Base.Currency(), k.Quote.Currency())
}

// MatchesExchangeAsset checks if the key matches the exchange and asset
func (k ExchangeAssetPair) MatchesExchangeAsset(exch string, item asset.Item) bool {
	return k.Exchange == exch && k.Asset == item
}

// MatchesPairAsset checks if the key matches the pair and asset
func (k ExchangeAssetPair) MatchesPairAsset(pair currency.Pair, item asset.Item) bool {
	return k.Base == pair.Base.Item &&
		k.Quote == pair.Quote.Item &&
		k.Asset == item
}

// MatchesExchange checks if the exchange matches
func (k ExchangeAssetPair) MatchesExchange(exch string) bool {
	return k.Exchange == exch
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
	return currency.NewPair(k.Base.Currency(), k.Quote.Currency())
}

// SubAccountAsset is a unique map key signature for subaccount and asset
type SubAccountAsset struct {
	SubAccount string
	Asset      asset.Item
}
