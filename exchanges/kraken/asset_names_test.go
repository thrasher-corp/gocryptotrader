package kraken

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

// seedTestAssetNames seeds the asset names of the pairs the websocket tests use, as Kraken's assets list names them
func seedTestAssetNames(ex *Exchange) {
	ex.assetNames.seed(
		map[string]AssetInfo{"XXBT": {AlternativeName: "XBT"}, "ZUSD": {AlternativeName: "USD"}, "XXDG": {AlternativeName: "XDG"}},
		map[string]AssetInfo{"BTC": {AlternativeName: "XBT"}, "USD": {AlternativeName: "USD"}, "DOGE": {AlternativeName: "XDG"}},
	)
}

func TestSeedAssets(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	require.False(t, ex.assetNames.seeded(), "asset names must not be seeded before SeedAssets")
	require.NoError(t, ex.SeedAssets(t.Context()), "SeedAssets must not error")
	assert.True(t, ex.assetNames.seeded(), "SeedAssets should seed the asset names")
	for name, exp := range map[string]string{
		"XXBT":  "XBT",
		"BTC":   "XBT",
		"ZUSD":  "USD",
		"USD":   "USD",
		"XXDG":  "XDG",
		"DOGE":  "XDG",
		"XETH":  "ETH",
		"ETH":   "ETH",
		"XBT.M": "XBT.M",
		"BTC.M": "XBT.M",
	} {
		alt, ok := ex.assetNames.alternativeName(name)
		assert.Truef(t, ok, "alternativeName should know %s", name)
		assert.Equalf(t, exp, alt, "alternativeName should return the alternative name of %s", name)
	}
	for alt, exp := range map[string]string{"XBT": "BTC", "XDG": "DOGE", "XBT.M": "BTC.M", "ETH": "ETH", "USD": "USD"} {
		assert.Equalf(t, exp, ex.assetNames.displayName(alt), "displayName should return the display name of %s", alt)
	}
}

func TestAssetNames(t *testing.T) {
	t.Parallel()
	var a assetNames
	assert.False(t, a.seeded(), "seeded should be false before seeding")
	_, ok := a.alternativeName("BTC")
	assert.False(t, ok, "alternativeName should not know a name before seeding")
	assert.Equal(t, "XBT", a.displayName("XBT"), "displayName should return the alternative name before seeding")

	a.seed(
		map[string]AssetInfo{"XXBT": {AlternativeName: "XBT"}, "ZUSD": {AlternativeName: "USD"}},
		map[string]AssetInfo{"BTC": {AlternativeName: "XBT"}, "USD": {AlternativeName: "USD"}},
	)
	assert.True(t, a.seeded(), "seeded should be true after seeding")
	assert.Equal(t, map[string]string{"XXBT": "XBT", "ZUSD": "USD", "BTC": "XBT", "USD": "USD"}, a.alternative, "seed should map internal and display names to alternative names")
	assert.Equal(t, map[string]string{"XBT": "BTC"}, a.display, "seed should map only the alternative names whose display names differ")
	_, ok = a.alternativeName("ETH")
	assert.False(t, ok, "alternativeName should not know an unseeded name")
	assert.Equal(t, "ETH", a.displayName("ETH"), "displayName should return an unseeded alternative name as it is")
}

func TestDisplaySymbol(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	seedTestAssetNames(ex)
	for p, exp := range map[currency.Pair]string{
		currency.NewPair(currency.XBT, currency.USD):            "BTC/USD",
		currency.NewPairWithDelimiter("xdg", "usd", "_"):        "DOGE/USD",
		currency.NewPair(currency.ETH, currency.XBT):            "ETH/BTC",
		currency.NewPair(currency.NewCode("NEW"), currency.USD): "NEW/USD",
	} {
		assert.Equalf(t, exp, ex.displaySymbol(p), "displaySymbol should return the display symbol of %s", p)
	}
}

func TestPairFromDisplaySymbol(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	seedTestAssetNames(ex)
	for symbol, exp := range map[string]currency.Pair{
		"BTC/USD":  currency.NewPairWithDelimiter("XBT", "USD", "_"),
		"DOGE/BTC": currency.NewPairWithDelimiter("XDG", "XBT", "_"),
		"NEW/USD":  currency.NewPairWithDelimiter("NEW", "USD", "_"),
	} {
		p, err := ex.pairFromDisplaySymbol(symbol)
		require.NoErrorf(t, err, "pairFromDisplaySymbol must not error for %s", symbol)
		assert.Equalf(t, exp, p, "pairFromDisplaySymbol should return %s in the configured format", symbol)
	}
	for _, symbol := range []string{"BTCUSD", "/USD", "BTC/", ""} {
		_, err := ex.pairFromDisplaySymbol(symbol)
		assert.ErrorIsf(t, err, errInvalidSymbol, "pairFromDisplaySymbol should reject %q", symbol)
	}
}
