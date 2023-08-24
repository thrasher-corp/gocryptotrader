package key

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func TestMatchesExchangeAsset(t *testing.T) {
	t.Parallel()
	cp := currency.NewPair(currency.BTC, currency.USD)
	k := ExchangePairAssetKey{
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
	k := ExchangePairAssetKey{
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
	k := ExchangePairAssetKey{
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
