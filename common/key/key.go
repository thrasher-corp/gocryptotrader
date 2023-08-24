package key

import (
	"strings"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

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

// SubAccountCurrencyAssetKey is a unique map key signature for subaccount, currency code and asset
type SubAccountCurrencyAssetKey struct {
	SubAccount string
	Currency   *currency.Item
	Asset      asset.Item
}

// MatchesExchangeAsset checks if the key matches the exchange and asset
func (k *ExchangePairAssetKey) MatchesExchangeAsset(exch string, item asset.Item) bool {
	return strings.EqualFold(k.Exchange, exch) && k.Asset == item
}

// MatchesPairAsset checks if the key matches the pair and asset
func (k *ExchangePairAssetKey) MatchesPairAsset(pair currency.Pair, item asset.Item) bool {
	return k.Base == pair.Base.Item &&
		k.Quote == pair.Quote.Item &&
		k.Asset == item
}

// MatchesExchange checks if the exchange matches
func (k *ExchangePairAssetKey) MatchesExchange(exch string) bool {
	return strings.EqualFold(k.Exchange, exch)
}
