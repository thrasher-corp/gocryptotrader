package key

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func TestMatchesExchangeAsset(t *testing.T) {
	t.Parallel()
	cp := currency.NewBTCUSD()
	k := ExchangeAssetPair{
		Exchange: "test",
		Base:     cp.Base.Item,
		Quote:    cp.Quote.Item,
		Asset:    asset.Spot,
	}
	assert.True(t, k.MatchesExchangeAsset("test", asset.Spot))
	assert.False(t, k.MatchesExchangeAsset("TEST", asset.Futures))
	assert.False(t, k.MatchesExchangeAsset("test", asset.Futures))
	assert.False(t, k.MatchesExchangeAsset("TEST", asset.Spot))
}

func TestMatchesPairAsset(t *testing.T) {
	t.Parallel()
	cp := currency.NewBTCUSD()
	k := ExchangeAssetPair{
		Base:  cp.Base.Item,
		Quote: cp.Quote.Item,
		Asset: asset.Spot,
	}
	assert.True(t, k.MatchesPairAsset(cp, asset.Spot))
	assert.False(t, k.MatchesPairAsset(cp, asset.Futures))
	assert.False(t, k.MatchesPairAsset(currency.EMPTYPAIR, asset.Futures))
	assert.False(t, k.MatchesPairAsset(currency.NewBTCUSDT(), asset.Spot))
}

func TestMatchesExchange(t *testing.T) {
	t.Parallel()
	k := ExchangeAssetPair{
		Exchange: "test",
	}
	assert.True(t, k.MatchesExchange("test"))
	assert.False(t, k.MatchesExchange("TEST"))
	assert.False(t, k.MatchesExchange("tèst"))
	assert.False(t, k.MatchesExchange(""))
}

func TestExchangePairAsset_Pair(t *testing.T) {
	t.Parallel()
	cp := currency.NewBTCUSD()
	k := ExchangeAssetPair{
		Base:  currency.BTC.Item,
		Quote: currency.USD.Item,
		Asset: asset.Spot,
	}
	assert.Equal(t, cp, k.Pair())
	cp = currency.NewPair(currency.BTC, currency.EMPTYCODE)
	k.Quote = currency.EMPTYCODE.Item
	assert.Equal(t, cp, k.Pair())
}

func TestPairAsset_Pair(t *testing.T) {
	t.Parallel()
	cp := currency.NewBTCUSD()
	k := PairAsset{
		Base:  currency.BTC.Item,
		Quote: currency.USD.Item,
		Asset: asset.Spot,
	}
	assert.Equal(t, cp, k.Pair())
	cp = currency.NewPair(currency.BTC, currency.EMPTYCODE)
	k.Quote = currency.EMPTYCODE.Item
	assert.Equal(t, cp, k.Pair())
}

func TestNewExchangePairAssetKey(t *testing.T) {
	t.Parallel()
	e := "test"
	a := asset.Spot
	p := currency.NewBTCUSDT()
	k := NewExchangeAssetPair(e, a, p)
	assert.Equal(t, e, k.Exchange)
	assert.Equal(t, p.Base.Item, k.Base)
	assert.Equal(t, p.Quote.Item, k.Quote)
	assert.Equal(t, a, k.Asset)

	e = ""
	a = 0
	p = currency.EMPTYPAIR
	k = NewExchangeAssetPair(e, a, p)
	assert.Equal(t, a, k.Asset, "NewExchangeAssetPair should not alter an invalid asset")
}
