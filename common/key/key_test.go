package key

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func TestMatchesExchangeAsset(t *testing.T) {
	t.Parallel()
	cp := currency.NewPair(currency.BTC, currency.USD)
	k := ExchangePairAsset{
		Exchange: "test",
		Base:     cp.Base.Item,
		Quote:    cp.Quote.Item,
		Asset:    asset.Spot,
	}
	if !k.MatchesExchangeAsset("test", asset.Spot) {
		t.Error("expected true")
	}
	if k.MatchesExchangeAsset("TEST", asset.Futures) {
		t.Error("expected false")
	}
	if k.MatchesExchangeAsset("test", asset.Futures) {
		t.Error("expected false")
	}
	if !k.MatchesExchangeAsset("TEST", asset.Spot) {
		t.Error("expected true")
	}
}

func TestMatchesPairAsset(t *testing.T) {
	t.Parallel()
	cp := currency.NewPair(currency.BTC, currency.USD)
	k := ExchangePairAsset{
		Base:  cp.Base.Item,
		Quote: cp.Quote.Item,
		Asset: asset.Spot,
	}
	if !k.MatchesPairAsset(cp, asset.Spot) {
		t.Error("expected true")
	}
	if k.MatchesPairAsset(cp, asset.Futures) {
		t.Error("expected false")
	}
	if k.MatchesPairAsset(currency.EMPTYPAIR, asset.Futures) {
		t.Error("expected false")
	}
	if k.MatchesPairAsset(currency.NewPair(currency.BTC, currency.USDT), asset.Spot) {
		t.Error("expected false")
	}
}

func TestMatchesExchange(t *testing.T) {
	t.Parallel()
	k := ExchangePairAsset{
		Exchange: "test",
	}
	if !k.MatchesExchange("test") {
		t.Error("expected true")
	}
	if !k.MatchesExchange("TEST") {
		t.Error("expected true")
	}
	if k.MatchesExchange("tèst") {
		t.Error("expected false")
	}
	if k.MatchesExchange("") {
		t.Error("expected false")
	}
}

func TestExchangePairAsset_Pair(t *testing.T) {
	t.Parallel()
	cp := currency.NewPair(currency.BTC, currency.USD)
	k := ExchangePairAsset{
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
	cp := currency.NewPair(currency.BTC, currency.USD)
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
	k := NewExchangePairAssetKey(e, a, p)
	assert.Equal(t, e, k.Exchange)
	assert.Equal(t, p.Base.Item, k.Base)
	assert.Equal(t, p.Quote.Item, k.Quote)
	assert.Equal(t, a, k.Asset)

	// demonstration of rules that it is simply a helper and not a validator
	e = ""
	a = 0
	p = currency.EMPTYPAIR
	k = NewExchangePairAssetKey(e, a, p)
	assert.Equal(t, e, k.Exchange)
	assert.Equal(t, p.Base.Item, k.Base)
	assert.Equal(t, p.Quote.Item, k.Quote)
	assert.Equal(t, a, k.Asset)
}
