package currency

import (
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func initTest(t *testing.T) *PairsManager {
	t.Helper()
	spotAvailable, err := NewPairsFromStrings([]string{"BTC-USD", "LTC-USD"})
	if err != nil {
		t.Fatal(err)
	}

	spotEnabled, err := NewPairsFromStrings([]string{"BTC-USD"})
	if err != nil {
		t.Fatal(err)
	}

	spot := &PairStore{
		AssetEnabled:  true,
		Available:     spotAvailable,
		Enabled:       spotEnabled,
		RequestFormat: &PairFormat{Uppercase: true},
		ConfigFormat:  &PairFormat{Uppercase: true, Delimiter: "-"},
	}

	futures := &PairStore{
		AssetEnabled:  false,
		Available:     spotAvailable,
		Enabled:       spotEnabled,
		RequestFormat: &PairFormat{Uppercase: true},
		ConfigFormat:  &PairFormat{Uppercase: true, Delimiter: "-"},
	}

	var p PairsManager

	err = p.Store(asset.Spot, spot)
	if err != nil {
		t.Fatal(err)
	}
	err = p.Store(asset.Futures, futures)
	if err != nil {
		t.Fatal(err)
	}

	return &p
}

// testCode returns a code outside the shared code store
func testCode(symbol string) Code {
	return Code{Item: &Item{Symbol: symbol, Lower: strings.ToLower(symbol)}}
}

// matcherKey identifies one PairsManager matcher entry
type matcherKey struct {
	Symbol string
	Asset  asset.Item
}

func (k matcherKey) lookup(pm *PairsManager) (*Pair, bool) {
	p, ok := pm.matcher[k.Asset][k.Symbol]
	return p, ok
}

func TestGetAssetTypes(t *testing.T) {
	t.Parallel()
	p := initTest(t)

	a := p.GetAssetTypes(false)
	if len(a) != 2 {
		t.Errorf("expected 2 but received: %d", len(a))
	}

	a = p.GetAssetTypes(true)
	if len(a) != 1 {
		t.Errorf("GetAssetTypes shouldn't be nil")
	}

	if !a.Contains(asset.Spot) {
		t.Errorf("AssetTypeSpot should be in the assets list")
	}
}

func TestGet(t *testing.T) {
	t.Parallel()
	p := initTest(t)

	_, err := p.Get(asset.Spot)
	if err != nil {
		t.Error(err)
	}

	_, err = p.Get(asset.Empty)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = p.Get(asset.CoinMarginedFutures)
	require.ErrorIs(t, err, asset.ErrNotSupported)
}

func TestPairsManagerMatch(t *testing.T) {
	t.Parallel()

	p := &PairsManager{}

	_, err := p.Match("", 1337)
	require.ErrorIs(t, err, ErrSymbolStringEmpty)

	_, err = p.Match("sillyBilly", 1337)
	require.ErrorIs(t, err, errPairMatcherIsNil)

	p = initTest(t)

	_, err = p.Match("sillyBilly", 1337)
	require.ErrorIs(t, err, ErrPairNotFound)

	_, err = p.Match("sillyBilly", asset.Spot)
	require.ErrorIs(t, err, ErrPairNotFound)

	whatIgot, err := p.Match("bTCuSD", asset.Spot)
	require.NoError(t, err)

	whatIwant, err := NewPairFromString("btc-usd")
	if err != nil {
		t.Fatal(err)
	}

	if !whatIgot.Equal(whatIwant) {
		t.Fatal("expected btc-usd")
	}

	matcher := new(PairsManager)
	long := NewPair(testCode(strings.Repeat("LONG", 10)), testCode(strings.Repeat("QUOTE", 6)))
	accented := NewPair(testCode("ÄPFEL"), USDT)
	require.NoError(t, matcher.StorePairs(asset.Spot, Pairs{NewBTCUSD(), long, accented}, false), "StorePairs must not error")
	for _, tc := range []struct {
		name   string
		symbol string
		want   Pair
	}{
		{name: "lower case symbol", symbol: "btcusd", want: NewBTCUSD()},
		{name: "upper case symbol", symbol: "BTCUSD", want: NewBTCUSD()},
		{name: "symbol longer than the key buffer", symbol: strings.ToUpper(long.Base.String() + long.Quote.String()), want: long},
		{name: "non-ASCII upper case symbol", symbol: "ÄPFELUSDT", want: accented},
		{name: "non-ASCII lower case symbol", symbol: "äpfelusdt", want: accented},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := matcher.Match(tc.symbol, asset.Spot)
			require.NoError(t, err, "Match must not error")
			assert.True(t, got.Equal(tc.want), "Match should return the matching pair")
		})
	}

	for _, tc := range []struct {
		name   string
		symbol string
		key    string
	}{
		{name: "missing pair", symbol: "SILLYBILLY", key: "sillybilly"},
		{name: "missing non-ASCII pair", symbol: "ÄBCUSDT", key: "äbcusdt"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := matcher.Match(tc.symbol, asset.Spot)
			require.ErrorIs(t, err, ErrPairNotFound, "Match must error for a missing pair")
			assert.ErrorContains(t, err, tc.key, "Match's error should name the lower case symbol")
		})
	}
}

func TestPairsManagerMatchDelimited(t *testing.T) {
	t.Parallel()

	p := new(PairsManager)
	_, err := p.MatchDelimited("", asset.Spot)
	require.ErrorIs(t, err, ErrSymbolStringEmpty, "MatchDelimited must error for an empty symbol")
	_, err = p.MatchDelimited("BTC-USD", asset.Spot)
	require.ErrorIs(t, err, errPairMatcherIsNil, "MatchDelimited must error without a matcher")

	option := NewPair(BTC, testCode("26NOV24-92000-C"))
	accented := NewPair(testCode("ÄPFEL"), USDT)
	require.NoError(t, p.StorePairs(asset.Spot, Pairs{NewBTCUSD(), accented}, false), "StorePairs must not error")
	require.NoError(t, p.StorePairs(asset.Options, Pairs{option}, false), "StorePairs must not error")
	for _, tc := range []struct {
		name   string
		symbol string
		a      asset.Item
		want   Pair
	}{
		{name: "dash delimited upper case symbol", symbol: "BTC-USD", a: asset.Spot, want: NewBTCUSD()},
		{name: "underscore delimited lower case symbol", symbol: "btc_usd", a: asset.Spot, want: NewBTCUSD()},
		{name: "symbol without a delimiter", symbol: "btcusd", a: asset.Spot, want: NewBTCUSD()},
		{name: "only the first punctuation character is dropped", symbol: "BTC-26NOV24-92000-C", a: asset.Options, want: option},
		{name: "non-ASCII delimited symbol", symbol: "ÄPFEL/USDT", a: asset.Spot, want: accented},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := p.MatchDelimited(tc.symbol, tc.a)
			require.NoError(t, err, "MatchDelimited must not error")
			assert.True(t, got.Equal(tc.want), "MatchDelimited should return the matching pair")
		})
	}

	t.Run("missing pair", func(t *testing.T) {
		t.Parallel()
		_, err := p.MatchDelimited("BTC-XYZ", asset.Spot)
		require.ErrorIs(t, err, ErrPairNotFound, "MatchDelimited must error for a missing pair")
		assert.ErrorContains(t, err, "btcxyz", "MatchDelimited's error should name the symbol's key")
	})
}

func TestMatchKey(t *testing.T) {
	t.Parallel()

	option := NewPair(BTC, testCode("26NOV24-92000-C"))
	accented := NewPair(testCode("ÄPFEL"), USDT)
	accentedQuote := NewPair(testCode("X"), testCode("ÄPFEL"))
	long := NewPair(testCode(strings.Repeat("LONG", 10)), testCode(strings.Repeat("QUOTE", 6)))
	symbols := make(map[string]*Pair)
	for _, pair := range []Pair{NewBTCUSD(), option, accented, accentedQuote, long} {
		symbols[pair.Base.Lower().String()+pair.Quote.Lower().String()] = &pair
	}

	for _, tc := range []struct {
		name         string
		symbol       string
		start        int
		hasDelimiter bool
		want         Pair
	}{
		{name: "upper case symbol", symbol: "BTCUSD", want: NewBTCUSD()},
		{name: "keeps the bytes before start", symbol: "btcUSD", start: 3, want: NewBTCUSD()},
		{name: "drops only the first delimiter", symbol: "BTC-26NOV24-92000-C", hasDelimiter: true, want: option},
		{name: "non-ASCII symbol", symbol: "ÄPFELUSDT", want: accented},
		{name: "delimiter before a non-ASCII byte", symbol: "X-ÄPFEL", hasDelimiter: true, want: accentedQuote},
		{name: "key longer than the buffer", symbol: strings.ToUpper(long.Base.String() + long.Quote.String()), want: long},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := matchKey(symbols, tc.symbol, tc.start, asset.Spot, tc.hasDelimiter)
			require.NoError(t, err, "matchKey must not error")
			assert.True(t, got.Equal(tc.want), "matchKey should return the matching pair")
		})
	}

	for _, tc := range []struct {
		name         string
		symbol       string
		hasDelimiter bool
		key          string
	}{
		{name: "delimiter kept without hasDelimiter", symbol: "BTC-USD", key: "btc-usd"},
		{name: "missing non-ASCII pair", symbol: "ÄBC-USDT", hasDelimiter: true, key: "äbcusdt"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := matchKey(symbols, tc.symbol, 0, asset.Spot, tc.hasDelimiter)
			require.ErrorIs(t, err, ErrPairNotFound, "matchKey must error for a missing pair")
			assert.ErrorContains(t, err, tc.key, "matchKey's error should name the key")
		})
	}
}

func TestWithoutDelimiter(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name         string
		symbol       string
		hasDelimiter bool
		want         string
	}{
		{name: "without hasDelimiter", symbol: "BTC-USD", want: "BTC-USD"},
		{name: "no punctuation", symbol: "BTCUSD", hasDelimiter: true, want: "BTCUSD"},
		{name: "first of several punctuation characters", symbol: "BTC-26NOV24-92000-C", hasDelimiter: true, want: "BTC26NOV24-92000-C"},
		{name: "underscore", symbol: "btc_usd", hasDelimiter: true, want: "btcusd"},
		{name: "after a non-ASCII character", symbol: "ÄPFEL/USDT", hasDelimiter: true, want: "ÄPFELUSDT"},
		{name: "multi-byte character before the delimiter", symbol: "grán-usdt", hasDelimiter: true, want: "gránusdt"},
		{name: "symbol characters are not punctuation", symbol: "BTC$USD", hasDelimiter: true, want: "BTC$USD"},
		{name: "empty symbol", hasDelimiter: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, withoutDelimiter(tc.symbol, tc.hasDelimiter), "withoutDelimiter should return the expected symbol")
		})
	}
}

func TestKeyBytes(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		bytes string
		class uint8
	}{
		{name: "upper case letters", bytes: "AMZ", class: keyByteChanges},
		{name: "non-ASCII bytes", bytes: "\x80\xc3\xff", class: keyByteChanges},
		{name: "punctuation", bytes: "-_/.:", class: keyByteDelimits},
		{name: "lower case letters, digits and symbols", bytes: "amz09$+^", class: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			for i := range len(tc.bytes) {
				assert.Equalf(t, tc.class, keyBytes[tc.bytes[i]], "keyBytes should classify %q", tc.bytes[i])
			}
		})
	}
}

func TestStore(t *testing.T) {
	t.Parallel()
	availPairs, err := NewPairsFromStrings([]string{"BTC-USD", "LTC-USD"})
	if err != nil {
		t.Fatal(err)
	}

	enabledPairs, err := NewPairsFromStrings([]string{"BTC-USD"})
	if err != nil {
		t.Fatal(err)
	}

	p := initTest(t)

	err = p.Store(asset.Futures,
		&PairStore{
			Available: availPairs,
			Enabled:   enabledPairs,
			RequestFormat: &PairFormat{
				Uppercase: true,
			},
			ConfigFormat: &PairFormat{
				Uppercase: true,
				Delimiter: "-",
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	f, err := p.Get(asset.Futures)
	if err != nil {
		t.Fatal(err)
	}

	if f == nil {
		t.Error("Futures assets shouldn't be nil")
	}

	err = p.Store(asset.Empty, nil)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	err = p.Store(asset.Futures, nil)
	require.ErrorIs(t, err, errPairStoreIsNil)

	t.Run("re-indexes only the stored asset", func(t *testing.T) {
		t.Parallel()
		p := initTest(t)
		sentinel := NewPairWithDelimiter("SENT", "INEL", "-")
		p.matcher[asset.Spot]["sentinel"] = &sentinel
		require.NoError(t, p.Store(asset.Futures, &PairStore{Available: Pairs{NewPairWithDelimiter("ETH", "USD", "-")}}), "Store must not error")
		_, err := p.Match("ethusd", asset.Futures)
		assert.NoError(t, err, "Match should find a pair stored by Store")
		_, err = p.Match("ltcusd", asset.Futures)
		assert.ErrorIs(t, err, ErrPairNotFound, "Match should not find a pair replaced by Store")
		_, err = p.Match("sentinel", asset.Spot)
		assert.NoError(t, err, "Store should leave other assets' matcher entries untouched")
	})
}

func TestDelete(t *testing.T) {
	t.Parallel()
	p := initTest(t)

	p.Pairs = nil
	p.Delete(asset.Spot)

	btcusdPairs, err := NewPairsFromStrings([]string{"BTC-USD"})
	if err != nil {
		t.Fatal(err)
	}

	err = p.Store(asset.Spot, &PairStore{Available: btcusdPairs})
	if err != nil {
		t.Fatal(err)
	}

	p.Delete(asset.UpsideProfitContract)
	spotPS, err := p.Get(asset.Spot)
	if err != nil {
		t.Fatal(err)
	}

	if spotPS == nil {
		t.Error("AssetTypeSpot should exist")
	}

	p.Delete(asset.Spot)

	if _, err := p.Get(asset.Spot); err == nil {
		t.Error("Delete should have deleted AssetTypeSpot")
	}
	_, err = p.Match("btcusd", asset.Spot)
	assert.ErrorIs(t, err, ErrPairNotFound, "Match should not find pairs of a deleted asset")
}

func TestGetPairs(t *testing.T) {
	t.Parallel()
	p := initTest(t)

	p.Pairs = nil
	pairs, err := p.GetPairs(asset.Spot, true)
	if err != nil {
		t.Fatal(err)
	}

	if pairs != nil {
		t.Fatal("pairs shouldn't be populated")
	}

	p = initTest(t)
	pairs, err = p.GetPairs(asset.Spot, true)
	if err != nil {
		t.Fatal(err)
	}
	if pairs == nil {
		t.Fatal("pairs should be populated")
	}

	pairs, err = p.GetPairs(asset.Empty, true)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	if pairs != nil {
		t.Fatal("pairs shouldn't be populated")
	}

	superfluous := NewPair(DASH, USDT)
	newPairs := p.Pairs[asset.Spot].Enabled.Add(superfluous)
	p.Pairs[asset.Spot].Enabled = newPairs

	_, err = p.GetPairs(asset.Spot, true)
	if err == nil {
		t.Fatal("error cannot be nil")
	}
}

func TestStoreFormat(t *testing.T) {
	t.Parallel()
	p := &PairsManager{}

	err := p.StoreFormat(0, &PairFormat{Delimiter: "~"}, true)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	err = p.StoreFormat(asset.Spot, nil, true)
	require.ErrorIs(t, err, ErrPairFormatIsNil)

	err = p.StoreFormat(asset.Spot, &PairFormat{Delimiter: "~"}, true)
	require.NoError(t, err)

	ps, err := p.Get(asset.Spot)
	if err != nil {
		t.Fatal(err)
	}

	if ps.ConfigFormat.Delimiter != "~" {
		t.Fatal("unexpected value")
	}

	err = p.StoreFormat(asset.Spot, &PairFormat{Delimiter: "/"}, false)
	require.NoError(t, err)

	ps, err = p.Get(asset.Spot)
	if err != nil {
		t.Fatal(err)
	}

	if ps.RequestFormat.Delimiter != "/" {
		t.Fatal("unexpected value")
	}
}

func TestStorePairs(t *testing.T) {
	t.Parallel()
	p := initTest(t)

	err := p.StorePairs(0, nil, false)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	p.Pairs = nil

	ethusdPairs, err := NewPairsFromStrings([]string{"ETH-USD"})
	if err != nil {
		t.Fatal(err)
	}

	err = p.StorePairs(asset.Spot, ethusdPairs, false)
	require.NoError(t, err)

	pairs, err := p.GetPairs(asset.Spot, false)
	if err != nil {
		t.Fatal(err)
	}

	ethusd, err := NewPairFromString("ETH-USD")
	if err != nil {
		t.Fatal(err)
	}

	if !pairs.Contains(ethusd, true) {
		t.Errorf("TestStorePairs failed, unexpected result")
	}

	p = initTest(t)
	err = p.StorePairs(asset.Spot, ethusdPairs, false)
	require.NoError(t, err)

	pairs, err = p.GetPairs(asset.Spot, false)
	if err != nil {
		t.Fatal(err)
	}

	if pairs == nil {
		t.Errorf("pairs should be populated")
	}

	if !pairs.Contains(ethusd, true) {
		t.Errorf("TestStorePairs failed, unexpected result")
	}

	ethkrwPairs, err := NewPairsFromStrings([]string{"ETH-KRW"})
	if err != nil {
		t.Error(err)
	}

	err = p.StorePairs(asset.Futures, ethkrwPairs, true)
	require.NoError(t, err)

	err = p.StorePairs(asset.Futures, ethkrwPairs, false)
	require.NoError(t, err)

	pairs, err = p.GetPairs(asset.Futures, true)
	if err != nil {
		t.Fatal(err)
	}

	if pairs == nil {
		t.Errorf("pairs futures should be populated")
	}

	ethkrw, err := NewPairFromString("ETH-KRW")
	if err != nil {
		t.Error(err)
	}

	if !pairs.Contains(ethkrw, true) {
		t.Errorf("TestStorePairs failed, unexpected result")
	}

	t.Run("available pairs re-index only their asset", func(t *testing.T) {
		t.Parallel()
		p := initTest(t)
		sentinel := NewPairWithDelimiter("SENT", "INEL", "-")
		p.matcher[asset.Futures]["sentinel"] = &sentinel
		require.NoError(t, p.StorePairs(asset.Spot, ethusdPairs, false), "StorePairs must not error")
		_, err := p.Match("ethusd", asset.Spot)
		assert.NoError(t, err, "Match should find a pair stored by StorePairs")
		_, err = p.Match("btcusd", asset.Spot)
		assert.ErrorIs(t, err, ErrPairNotFound, "Match should not find a pair replaced by StorePairs")
		_, err = p.Match("sentinel", asset.Futures)
		assert.NoError(t, err, "StorePairs should leave other assets' matcher entries untouched")
	})

	t.Run("enabled pairs missing from the available pairs are added", func(t *testing.T) {
		t.Parallel()
		p := initTest(t)
		require.NoError(t, p.StorePairs(asset.Spot, ethkrwPairs, true), "StorePairs must not error")
		available, err := p.GetPairs(asset.Spot, false)
		require.NoError(t, err, "GetPairs must not error for available pairs")
		assert.True(t, available.Contains(ethkrw, true), "StorePairs should add an enabled pair to the available pairs")
		assert.Len(t, available, 3, "StorePairs should keep the existing available pairs")
		_, err = p.GetPairs(asset.Spot, true)
		assert.NoError(t, err, "GetPairs should find every enabled pair available")
		_, err = p.Match("ethkrw", asset.Spot)
		assert.NoError(t, err, "Match should find an enabled pair added to the available pairs")
	})

	t.Run("enabled pairs already available leave the available pairs unchanged", func(t *testing.T) {
		t.Parallel()
		p := initTest(t)
		sentinel := NewPairWithDelimiter("SENT", "INEL", "-")
		p.matcher[asset.Spot]["sentinel"] = &sentinel
		require.NoError(t, p.StorePairs(asset.Spot, Pairs{NewPairWithDelimiter("BTC", "USD", "-")}, true), "StorePairs must not error")
		available, err := p.GetPairs(asset.Spot, false)
		require.NoError(t, err, "GetPairs must not error for available pairs")
		assert.Len(t, available, 2, "StorePairs should not add an enabled pair already available")
		_, err = p.Match("sentinel", asset.Spot)
		assert.NoError(t, err, "StorePairs should not re-index when every enabled pair is already available")
	})
}

func TestDisablePair(t *testing.T) {
	t.Parallel()
	p := initTest(t)

	err := p.DisablePair(asset.Empty, EMPTYPAIR)
	assert.ErrorIs(t, err, asset.ErrNotSupported, "Empty asset should error")

	err = p.DisablePair(asset.Spot, EMPTYPAIR)
	assert.ErrorIs(t, err, ErrCurrencyPairEmpty, "Empty pair should error")

	p.Pairs = nil
	err = p.DisablePair(asset.Spot, NewBTCUSD())
	assert.ErrorIs(t, err, ErrPairManagerNotInitialised, "Uninitialised PairManager should error")

	p = initTest(t)
	err = p.DisablePair(asset.CoinMarginedFutures, EMPTYPAIR)
	assert.ErrorIs(t, err, ErrCurrencyPairEmpty, "Non-existent asset type should error")

	p.Pairs[asset.Spot] = nil
	err = p.DisablePair(asset.Spot, EMPTYPAIR)
	assert.ErrorIs(t, err, ErrCurrencyPairEmpty, "Empty pair store should error")

	p = initTest(t)
	err = p.DisablePair(asset.Spot, NewPair(LTC, USD))
	assert.ErrorIs(t, err, ErrPairNotFound, "Not Enabled pair should error")

	err = p.DisablePair(asset.Spot, NewBTCUSD())
	assert.NoError(t, err, "DisablePair should not error")
}

func TestEnablePair(t *testing.T) {
	t.Parallel()
	p := initTest(t)
	require.ErrorIs(t, p.EnablePair(asset.Empty, NewBTCUSD()), asset.ErrNotSupported)

	p.Pairs = nil
	// Test enabling a pair when the pair manager is not initialised
	if err := p.EnablePair(asset.Spot, NewBTCUSD()); err == nil {
		t.Error("unexpected result")
	}

	// Test asset type which doesn't exist
	p = initTest(t)
	if err := p.EnablePair(asset.Futures, EMPTYPAIR); err == nil {
		t.Error("unexpected result")
	}

	// Test asset type which has an empty pair store
	p.Pairs[asset.Spot] = nil
	if err := p.EnablePair(asset.Spot, EMPTYPAIR); err == nil {
		t.Error("unexpected result")
	}

	// Test enabling a pair which isn't in the list of available pairs
	p = initTest(t)
	if err := p.EnablePair(asset.Spot, NewPair(ETH, USD)); err == nil {
		t.Error("unexpected result")
	}

	// Test enabling a pair which already is enabled
	if err := p.EnablePair(asset.Spot, NewBTCUSD()); err == nil {
		t.Error("unexpected result")
	}

	// Test enabling a valid pair
	if err := p.EnablePair(asset.Spot, NewPair(LTC, USD)); err != nil {
		t.Error("unexpected result")
	}
}

func TestAssetEnabled(t *testing.T) {
	t.Parallel()
	p := initTest(t)

	err := p.IsAssetEnabled(asset.Empty)
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	p.Pairs = nil

	// Test enabling a pair when the pair manager is not initialised
	err = p.IsAssetEnabled(asset.Spot)
	assert.ErrorIs(t, err, ErrPairManagerNotInitialised)

	err = p.SetAssetEnabled(asset.Spot, true)
	assert.ErrorIs(t, err, ErrPairManagerNotInitialised)

	// Test asset type which doesn't exist
	p = initTest(t)

	p.Pairs[asset.Spot].AssetEnabled = false

	err = p.IsAssetEnabled(asset.Spot)
	assert.ErrorIs(t, err, asset.ErrNotEnabled)

	err = p.SetAssetEnabled(asset.Spot, false)
	assert.NoError(t, err)

	err = p.SetAssetEnabled(asset.Spot, false)
	assert.NoError(t, err, "Setting to disabled twice should not error")

	err = p.IsAssetEnabled(asset.Spot)
	assert.ErrorIs(t, err, asset.ErrNotEnabled)

	err = p.SetAssetEnabled(asset.Spot, true)
	assert.NoError(t, err)

	err = p.SetAssetEnabled(asset.Spot, true)
	assert.NoError(t, err, "Setting to enabled twice should not error")

	err = p.IsAssetEnabled(asset.Spot)
	assert.NoError(t, err, "IsAssetEnabled should not error")
}

// TestFullStoreUnmarshalMarshal tests JSON Marshal and Unmarshal
func TestFullStoreUnmarshalMarshal(t *testing.T) {
	t.Parallel()
	um := make(FullStore)
	um[asset.Spot] = &PairStore{AssetEnabled: true}

	data, err := json.Marshal(um)
	if err != nil {
		t.Fatal(err)
	}

	if string(data) != `{"spot":{"assetEnabled":true,"enabled":"","available":""}}` {
		t.Fatal("unexpected value")
	}

	var another FullStore
	err = json.Unmarshal(data, &another)
	require.NoError(t, err)

	if _, ok := another[asset.Spot]; !ok {
		t.Fatal("expected values to be associated with spot")
	}

	data = []byte(`{123:{"assetEnabled":null,"enabled":"","available":""}}`)
	err = json.Unmarshal(data, &another)
	assert.Error(t, err, "Unmarshal should error with invalid asset type")

	data = []byte(`{"bro":{"assetEnabled":null,"enabled":"","available":""}}`)
	err = json.Unmarshal(data, &another)
	require.ErrorIs(t, err, asset.ErrNotSupported)
}

func TestIsPairAvailable(t *testing.T) {
	t.Parallel()
	pm := initTest(t)
	cp := NewPairWithDelimiter("BTC", "USD", "-")
	ok, err := pm.IsPairAvailable(cp, asset.Spot)
	require.NoError(t, err, "IsPairAvailable must not error")
	assert.True(t, ok, "IsPairAvailable should return correct value for an available and enabled pair")

	ok, err = pm.IsPairAvailable(NewPair(SAFE, MOONRISE), asset.Spot)
	require.NoError(t, err, "IsPairAvailable must not error")
	assert.False(t, ok, "IsPairAvailable should return correct value for an non-existent")

	ok, err = pm.IsPairAvailable(cp, asset.Futures)
	require.NoError(t, err, "IsPairAvailable must not error")
	assert.False(t, ok, "IsPairAvailable should return false for a disabled asset type")

	cp = NewPairWithDelimiter("XRP", "DOGE", "-")
	ok, err = pm.IsPairAvailable(cp, asset.Spot)
	require.NoError(t, err, "IsPairAvailable must not error")
	assert.False(t, ok, "IsPairAvailable should return false for non-existent pair")

	_, err = pm.IsPairAvailable(cp, asset.PerpetualSwap)
	assert.ErrorIs(t, err, ErrAssetNotFound, "Should error when asset is not found")

	_, err = pm.IsPairAvailable(cp, asset.Item(1337))
	assert.ErrorIs(t, err, asset.ErrNotSupported, "Should error when asset is not supported")

	pm.Pairs[asset.PerpetualSwap] = &PairStore{}
	_, err = pm.IsPairAvailable(cp, asset.PerpetualSwap)
	require.NoError(t, err, "Must not error when store is empty")

	_, err = pm.IsPairAvailable(EMPTYPAIR, asset.PerpetualSwap)
	assert.ErrorIs(t, err, ErrCurrencyPairEmpty, "Should error when currency pair is empty")
}

func TestIsPairEnabled(t *testing.T) {
	t.Parallel()
	pm := initTest(t)
	cp := NewPairWithDelimiter("BTC", "USD", "-")
	enabled, err := pm.IsPairEnabled(cp, asset.Spot)
	require.NoError(t, err)
	assert.True(t, enabled, "IsPairEnabled should return true when pair is enabled")

	enabled, err = pm.IsPairEnabled(NewPair(SAFE, MOONRISE), asset.Spot)
	require.NoError(t, err)
	assert.False(t, enabled, "IsPairEnabled should return false when pair does not exist")

	enabled, err = pm.IsPairEnabled(cp, asset.Futures)
	require.NoError(t, err)
	assert.False(t, enabled, "IsPairEnabled should return false when asset not enabled")

	cp = NewPairWithDelimiter("XRP", "DOGE", "-")
	enabled, err = pm.IsPairEnabled(cp, asset.Spot)
	require.NoError(t, err)
	assert.False(t, enabled, "IsPairEnabled should return false when pair not in enabled list")

	_, err = pm.IsPairEnabled(cp, asset.PerpetualSwap)
	assert.ErrorIs(t, err, ErrAssetNotFound)

	_, err = pm.IsPairEnabled(cp, asset.Item(1337))
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	pm.Pairs[asset.PerpetualSwap] = &PairStore{}
	enabled, err = pm.IsPairEnabled(cp, asset.PerpetualSwap)
	require.NoError(t, err, "Must not error when store is empty")
	assert.False(t, enabled, "Should return false when store is empty")

	_, err = pm.IsPairEnabled(EMPTYPAIR, asset.PerpetualSwap)
	assert.ErrorIs(t, err, ErrCurrencyPairEmpty)
}

func TestEnsureOnePairEnabled(t *testing.T) {
	t.Parallel()
	p := NewBTCUSDT()
	pm := PairsManager{
		Pairs: map[asset.Item]*PairStore{
			asset.Futures: {},
			asset.Spot: {
				AssetEnabled: true,
				Available: []Pair{
					p,
				},
			},
		},
	}
	pair, item, err := pm.EnsureOnePairEnabled()
	assert.NoError(t, err)

	if len(pm.Pairs[asset.Spot].Enabled) != 1 {
		t.Errorf("received: '%v' but expected: '%v'", len(pm.Pairs[asset.Spot].Enabled), 1)
	}
	if item != asset.Spot || !pair.Equal(p) {
		t.Errorf("received: '%v %v' but expected: '%v %v'", item, p, asset.Spot, p)
	}

	pair, item, err = pm.EnsureOnePairEnabled()
	assert.NoError(t, err)

	if len(pm.Pairs[asset.Spot].Enabled) != 1 {
		t.Errorf("received: '%v' but expected: '%v'", len(pm.Pairs[asset.Spot].Enabled), 1)
	}

	if item != asset.Empty || !pair.Equal(EMPTYPAIR) {
		t.Errorf("received: '%v %v' but expected: '%v %v'", item, p, asset.Empty, EMPTYPAIR)
	}

	pm = PairsManager{
		Pairs: map[asset.Item]*PairStore{
			asset.Futures: {
				AssetEnabled: true,
				Available: []Pair{
					NewPair(BTC, USDC),
				},
			},
			asset.Spot: {
				AssetEnabled: true,
				Enabled: []Pair{
					p,
				},
				Available: []Pair{
					p,
				},
			},
		},
	}
	pair, item, err = pm.EnsureOnePairEnabled()
	assert.NoError(t, err)

	if len(pm.Pairs[asset.Spot].Enabled) != 1 {
		t.Errorf("received: '%v' but expected: '%v'", len(pm.Pairs[asset.Spot].Enabled), 1)
	}
	if len(pm.Pairs[asset.Futures].Enabled) != 0 {
		t.Errorf("received: '%v' but expected: '%v'", len(pm.Pairs[asset.Futures].Enabled), 0)
	}
	if item != asset.Empty || !pair.Equal(EMPTYPAIR) {
		t.Errorf("received: '%v %v' but expected: '%v %v'", item, p, asset.Empty, EMPTYPAIR)
	}

	pm = PairsManager{
		Pairs: map[asset.Item]*PairStore{
			asset.Futures: {
				AssetEnabled: true,
				Available:    []Pair{p},
			},
			asset.Options: {
				AssetEnabled: true,
				Available:    []Pair{},
			},
		},
	}
	pair, item, err = pm.EnsureOnePairEnabled()
	assert.NoError(t, err)

	if len(pm.Pairs[asset.Futures].Enabled) != 1 {
		t.Errorf("received: '%v' but expected: '%v'", len(pm.Pairs[asset.Futures].Enabled), 1)
	}
	if item != asset.Futures || !pair.Equal(p) {
		t.Errorf("received: '%v %v' but expected: '%v %v'", item, p, asset.Futures, p)
	}

	pm = PairsManager{
		Pairs: map[asset.Item]*PairStore{},
	}
	_, _, err = pm.EnsureOnePairEnabled()
	assert.ErrorIs(t, err, ErrCurrencyPairsEmpty)
}

func TestLoad(t *testing.T) {
	t.Parallel()
	base := PairsManager{}
	fmt1 := &PairFormat{Uppercase: true}
	fmt2 := &PairFormat{Uppercase: true, Delimiter: DashDelimiter}
	p := NewBTCUSDT()
	tt := int64(1337)
	seed := PairsManager{
		LastUpdated:     tt,
		UseGlobalFormat: true,
		ConfigFormat:    fmt1,
		RequestFormat:   fmt2,
		Pairs: map[asset.Item]*PairStore{
			asset.Futures: {
				AssetEnabled: true,
				Available:    []Pair{p},
			},
			asset.Options: {
				AssetEnabled: false,
				Available:    []Pair{},
			},
		},
	}

	base.Load(&seed)
	assert.True(t, base.Pairs[asset.Futures].AssetEnabled, "Futures AssetEnabled should be true")
	assert.True(t, base.Pairs[asset.Futures].Available.Contains(p, true), "Futures Available Pairs should contain BTCUSDT")
	assert.False(t, base.Pairs[asset.Options].AssetEnabled, "Options AssetEnabled should be false")
	assert.Equal(t, tt, base.LastUpdated, "Last Updated should be correct")
	assert.Equal(t, fmt1.Uppercase, base.ConfigFormat.Uppercase, "ConfigFormat Uppercase should be correct")
	assert.Equal(t, fmt2.Delimiter, base.RequestFormat.Delimiter, "RequestFormat Delimiter should be correct")
	found, err := base.Match("BTCUSDT", asset.Futures)
	require.NoError(t, err, "Match must not error")
	assert.Equal(t, p, found, "Should find the right pair")
}

func TestReindex(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name           string
		setup          func() *PairsManager
		wantKeys       []matcherKey
		wantAbsentKeys []matcherKey
	}{
		{
			name: "indexes available pairs across multiple assets",
			setup: func() *PairsManager {
				return &PairsManager{
					Pairs: FullStore{
						asset.Spot: {
							Available: Pairs{
								NewPairWithDelimiter("BTC", "USD", "-"),
								NewPairWithDelimiter("ETH", "USD", "-"),
							},
						},
						asset.Futures: {
							Available: Pairs{
								NewPairWithDelimiter("BTC", "USD", "-"),
							},
						},
					},
				}
			},
			wantKeys: []matcherKey{
				{Symbol: "btcusd", Asset: asset.Spot},
				{Symbol: "ethusd", Asset: asset.Spot},
				{Symbol: "btcusd", Asset: asset.Futures},
			},
		},
		{
			name: "skips nil pair stores without panic",
			setup: func() *PairsManager {
				return &PairsManager{
					Pairs: FullStore{
						asset.Spot:    nil,
						asset.Futures: {Available: Pairs{NewBTCUSD()}},
					},
				}
			},
			wantKeys: []matcherKey{
				{Symbol: "btcusd", Asset: asset.Futures},
			},
			wantAbsentKeys: []matcherKey{
				{Symbol: "btcusd", Asset: asset.Spot},
			},
		},
		{
			name: "empty store produces empty matcher",
			setup: func() *PairsManager {
				return &PairsManager{
					Pairs: FullStore{
						asset.Spot: {Available: Pairs{}},
					},
				}
			},
		},
		{
			name: "replaces stale matcher entries on reindex",
			setup: func() *PairsManager {
				return &PairsManager{
					Pairs: FullStore{
						asset.Spot: {
							Available: Pairs{NewPairWithDelimiter("BTC", "USD", "-")},
						},
					},
					matcher: map[asset.Item]map[string]*Pair{
						asset.Spot: {"oldpair": new(NewPairWithDelimiter("OLD", "PAIR", "-"))},
					},
				}
			},
			wantKeys: []matcherKey{
				{Symbol: "btcusd", Asset: asset.Spot},
			},
			wantAbsentKeys: []matcherKey{
				{Symbol: "oldpair", Asset: asset.Spot},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			pm := tc.setup()
			assert.NotPanics(t, pm.reindex, "reindex should not panic")

			for _, k := range tc.wantKeys {
				p, ok := k.lookup(pm)
				assert.Truef(t, ok, "matcher should contain key %v", k)
				if ok {
					assert.NotNilf(t, p, "matcher entry for key %v should not be nil", k)
				}
			}

			for _, k := range tc.wantAbsentKeys {
				_, ok := k.lookup(pm)
				assert.Falsef(t, ok, "matcher should not contain key %v", k)
			}
		})
	}
}

func TestIndexAsset(t *testing.T) {
	t.Parallel()

	eth := NewPairWithDelimiter("ETH", "USD", "-")
	old := NewPairWithDelimiter("OLD", "PAIR", "-")
	for _, tc := range []struct {
		name           string
		setup          func() *PairsManager
		wantKeys       []matcherKey
		wantAbsentKeys []matcherKey
	}{
		{
			name: "creates the matcher",
			setup: func() *PairsManager {
				return &PairsManager{Pairs: FullStore{asset.Spot: {Available: Pairs{NewBTCUSD()}}}}
			},
			wantKeys: []matcherKey{{Symbol: "btcusd", Asset: asset.Spot}},
		},
		{
			name: "replaces the asset's entries",
			setup: func() *PairsManager {
				return &PairsManager{
					Pairs:   FullStore{asset.Spot: {Available: Pairs{NewBTCUSD()}}},
					matcher: map[asset.Item]map[string]*Pair{asset.Spot: {"oldpair": &old}},
				}
			},
			wantKeys:       []matcherKey{{Symbol: "btcusd", Asset: asset.Spot}},
			wantAbsentKeys: []matcherKey{{Symbol: "oldpair", Asset: asset.Spot}},
		},
		{
			name: "leaves other assets' entries",
			setup: func() *PairsManager {
				return &PairsManager{
					Pairs:   FullStore{asset.Spot: {Available: Pairs{NewBTCUSD()}}},
					matcher: map[asset.Item]map[string]*Pair{asset.Futures: {"ethusd": &eth}},
				}
			},
			wantKeys: []matcherKey{{Symbol: "btcusd", Asset: asset.Spot}, {Symbol: "ethusd", Asset: asset.Futures}},
		},
		{
			name: "removes the entries of a nil store",
			setup: func() *PairsManager {
				return &PairsManager{
					Pairs:   FullStore{asset.Spot: nil},
					matcher: map[asset.Item]map[string]*Pair{asset.Spot: {"oldpair": &old}},
				}
			},
			wantAbsentKeys: []matcherKey{{Symbol: "oldpair", Asset: asset.Spot}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			pm := tc.setup()
			pm.indexAsset(asset.Spot)
			for _, k := range tc.wantKeys {
				p, ok := k.lookup(pm)
				if assert.Truef(t, ok, "matcher should contain key %v", k) && k.Asset == asset.Spot {
					assert.Same(t, &pm.Pairs[asset.Spot].Available[0], p, "matcher entry should point at the stored pair")
				}
			}
			for _, k := range tc.wantAbsentKeys {
				_, ok := k.lookup(pm)
				assert.Falsef(t, ok, "matcher should not contain key %v", k)
			}
		})
	}
}

func TestSortAvailablePairs(t *testing.T) {
	t.Parallel()

	sentinelKey := matcherKey{Symbol: "sentinel", Asset: asset.Spot}
	sentinelPair := NewPairWithDelimiter("SENT", "INEL", "-")
	futuresKey := matcherKey{Symbol: "btcusd", Asset: asset.Futures}

	for _, tc := range []struct {
		name            string
		setup           func() *PairsManager
		nilReceiver     bool
		checkAvailable  bool
		wantAvailable   string
		checkMatch      bool
		wantMatch       string
		checkSentinel   bool
		wantSentinel    bool
		checkFuturesKey bool
		wantFuturesKey  bool
		checkMatcherLen bool
		wantMatcherLen  int
	}{
		{
			name: "sorts and reindexes unsorted available pairs",
			setup: func() *PairsManager {
				pm := &PairsManager{
					Pairs: FullStore{
						asset.Spot: {
							Available: Pairs{
								NewPairWithDelimiter("ETH", "USD", "-"),
								NewPairWithDelimiter("ADA", "USD", "-"),
								NewPairWithDelimiter("BTC", "USD", "-"),
							},
						},
					},
				}
				pm.reindex()
				return pm
			},
			checkAvailable: true,
			wantAvailable:  "ADA-USD,BTC-USD,ETH-USD",
			checkMatch:     true,
			wantMatch:      "BTC-USD",
		},
		{
			name: "noop when no store has more than one available pair",
			setup: func() *PairsManager {
				return &PairsManager{
					Pairs: FullStore{
						asset.Spot:    nil,
						asset.Futures: {Available: Pairs{NewBTCUSD()}},
					},
					matcher: map[asset.Item]map[string]*Pair{
						sentinelKey.Asset: {sentinelKey.Symbol: &sentinelPair},
					},
				}
			},
			checkSentinel:   true,
			wantSentinel:    true,
			checkFuturesKey: true,
			wantFuturesKey:  false,
		},
		{
			name: "already sorted available pairs do not reindex",
			setup: func() *PairsManager {
				return &PairsManager{
					Pairs: FullStore{
						asset.Spot: {
							Available: Pairs{
								NewPairWithDelimiter("ADA", "USD", "-"),
								NewPairWithDelimiter("BTC", "USD", "-"),
							},
						},
					},
					matcher: map[asset.Item]map[string]*Pair{
						sentinelKey.Asset: {sentinelKey.Symbol: &sentinelPair},
					},
				}
			},
			checkAvailable:  true,
			wantAvailable:   "ADA-USD,BTC-USD",
			checkSentinel:   true,
			wantSentinel:    true,
			checkMatcherLen: true,
			wantMatcherLen:  1,
		},
		{
			name: "nil receiver does not panic",
			setup: func() *PairsManager {
				return nil
			},
			nilReceiver: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			pm := tc.setup()
			if tc.nilReceiver {
				assert.NotPanics(t, func() {
					pm.SortAvailablePairs()
				}, "SortAvailablePairs should not panic for nil receivers")
				return
			}

			pm.SortAvailablePairs()

			if tc.checkAvailable {
				got, err := pm.GetPairs(asset.Spot, false)
				require.NoError(t, err, "GetPairs must not error for sorted available pairs")
				assert.Equal(t, tc.wantAvailable, got.Join(), "Available pairs should match expected ordering")
			}

			if tc.checkMatch {
				matched, err := pm.Match("btcusd", asset.Spot)
				require.NoError(t, err, "Match must not error after sorting and reindexing available pairs")
				assert.Equal(t, tc.wantMatch, matched.String(), "Match should return the expected pair after sorting")
			}

			if tc.checkSentinel {
				_, ok := sentinelKey.lookup(pm)
				assert.Equal(t, tc.wantSentinel, ok, "SortAvailablePairs should not alter sentinel matcher entries unexpectedly")
			}

			if tc.checkFuturesKey {
				_, ok := futuresKey.lookup(pm)
				assert.Equal(t, tc.wantFuturesKey, ok, "SortAvailablePairs should only adjust matcher entries when reindexing is required")
			}

			if tc.checkMatcherLen {
				assert.Len(t, pm.matcher[asset.Spot], tc.wantMatcherLen, "Matcher length should remain unchanged when no reindex is required")
			}
		})
	}
}

func TestPairsManagerMarshalJSON(t *testing.T) {
	t.Parallel()

	var nilPM *PairsManager
	data, err := nilPM.MarshalJSON()
	require.NoError(t, err, "MarshalJSON must not error for nil receiver")
	assert.Equal(t, "null", string(data), "MarshalJSON should return null for nil receiver")

	pm := initTest(t)
	pm.UseGlobalFormat = true
	pm.LastUpdated = 1700000000

	data, err = pm.MarshalJSON()
	require.NoError(t, err, "MarshalJSON must not error for populated manager")

	var decoded PairsManager
	require.NoError(t, json.Unmarshal(data, &decoded), "Unmarshal must not error for MarshalJSON output")
	assert.True(t, decoded.UseGlobalFormat, "UseGlobalFormat should be preserved")
	assert.Equal(t, pm.LastUpdated, decoded.LastUpdated, "LastUpdated should be preserved")
	assert.Equal(t, pm.Pairs, decoded.Pairs, "Pairs should be preserved")
}

func checkPairDelimiter(tb testing.TB, p *PairsManager, err error, d, msg string) {
	tb.Helper()
	if assert.NoError(tb, err, "UnmarshalJSON should not error") {
		err := p.SetDelimitersFromConfig()
		assert.NoError(tb, err, "SetDelimitersFromConfig should not error")
		s := p.Pairs[asset.Spot]
		if assert.NotNil(tb, s, "Spot asset should not be nil") {
			for _, ps := range []Pairs{s.Enabled, s.Available} {
				if assert.NotEmpty(tb, ps, "PairStore should not be empty") {
					for _, p := range ps {
						assert.Equalf(tb, d, p.Delimiter, msg)
					}
				}
			}
		}
	}
}

// TestPairManagerSetDelimitersFromConfig tests behaviour expectations:
// * Should error with no ConfigFormat ( `CheckPairConsistency` catches that )
// * Asset pair config should take precedent
// * Global store pair config should apply when no Asset pair config
func TestPairManagerSetDelimitersFromConfig(t *testing.T) {
	t.Parallel()
	p := new(PairsManager)

	err := json.Unmarshal([]byte(`{"pairs":{"spot":{"enabled":"BTC-USD,M_PIT-USDT","available":"BTC-USD,M_PIT-USD"}}}`), p)
	if assert.NoError(t, err, "UnmarshalJSON should not error") {
		err = p.SetDelimitersFromConfig()
		assert.ErrorIs(t, err, errPairConfigFormatNil, "SetDelimitersFromConfig should error correctly")
	}

	err = json.Unmarshal([]byte(`{"pairs":{"spot":{"configFormat":{"delimiter":"-"},"enabled":"BTC-USD,M_PIT-USDT","available":"BTC-USD,M_PIT-USD"}}}`), p)
	checkPairDelimiter(t, p, err, "-", "Delimiter should be correct with only bottom level configFormat")

	err = json.Unmarshal([]byte(`{"configFormat":{"delimiter":"/"},"pairs":{"spot":{"enabled":"BTC/USD,M_PIT/USDT","available":"BTC/USD,M_PIT/USD"}}}`), p)
	checkPairDelimiter(t, p, err, "/", "Delimiter should be correct with top level configFormat")

	err = json.Unmarshal([]byte(`{"configFormat":{"delimiter":"_"},"pairs":{"spot":{"configFormat":{"delimiter":"/"},"enabled":"BTC/USD,M_PIT/USDT","available":"BTC/USD,M_PIT/USD"}}}`), p)
	checkPairDelimiter(t, p, err, "/", "Delimiter should be correct with bottom level configFormat")

	err = json.Unmarshal([]byte(`{"pairs":{"spot":{"configFormat":{"delimiter":"_"},"enabled":"BTC-USDT","available":"BTC-USDT"}}}`), p)
	if assert.NoError(t, err, "UnmarshalJSON should not error") {
		err := p.SetDelimitersFromConfig()
		assert.ErrorIs(t, err, errDelimiterNotFound, "SetDelimitersFromConfig should error correctly")
	}
}

// TestGetFormat exercises PairsManager GetFormat
func TestGetFormat(t *testing.T) {
	t.Parallel()

	m := PairsManager{
		UseGlobalFormat: true,
		ConfigFormat: &PairFormat{
			Uppercase: true,
			Delimiter: "🦄",
		},
		RequestFormat: &PairFormat{
			Delimiter: "~",
		},
	}

	pFmt, err := m.GetFormat(asset.Spot, true)
	require.NoError(t, err)
	assert.Equal(t, "~", pFmt.Delimiter, "Global Format Delimiter should be correct")
	assert.False(t, pFmt.Uppercase, "Global Format Uppercase should be correct")

	pFmt, err = m.GetFormat(asset.Spot, false)
	require.NoError(t, err)
	assert.Equal(t, "🦄", pFmt.Delimiter, "Global Format Delimiter should be special")
	assert.True(t, pFmt.Uppercase, "Global Format Uppercase should be correct")

	m.ConfigFormat = nil
	m.RequestFormat = nil
	_, err = m.GetFormat(asset.Spot, true)
	assert.ErrorIs(t, err, ErrPairFormatIsNil, "Global GetFormat Should error correctly on nil request format")
	_, err = m.GetFormat(asset.Spot, false)
	assert.ErrorIs(t, err, ErrPairFormatIsNil, "Global GetFormat Should error correctly on nil config format")

	m.UseGlobalFormat = false
	err = m.Store(asset.Spot, &PairStore{
		ConfigFormat:  &pFmt,
		RequestFormat: &PairFormat{Delimiter: "/", Uppercase: true},
	})
	require.NoError(t, err, "Store must not error")

	pFmt, err = m.GetFormat(asset.Spot, false)
	require.NoError(t, err)
	assert.Equal(t, "🦄", pFmt.Delimiter, "Per Asset Format Delimiter should be correct")
	assert.True(t, pFmt.Uppercase, "Per Asset Format Uppercase should be correct")

	pFmt, err = m.GetFormat(asset.Spot, true)
	require.NoError(t, err)
	assert.Equal(t, "/", pFmt.Delimiter, "Per Asset Format Delimiter should be correct")
	assert.True(t, pFmt.Uppercase, "Per Asset Format Uppercase should be correct")

	err = m.Store(asset.Spot, &PairStore{})
	require.NoError(t, err, "Store must not error")
	_, err = m.GetFormat(asset.Spot, true)
	assert.ErrorIs(t, err, ErrPairFormatIsNil, "Per Asset GetFormat Should error correctly on nil request format")
	_, err = m.GetFormat(asset.Spot, false)
	assert.ErrorIs(t, err, ErrPairFormatIsNil, "Per Asset GetFormat Should error correctly on nil config format")
}

// TestIsAssetSupported exercises IsAssetSupported
func TestIsAssetSupported(t *testing.T) {
	t.Parallel()
	p := PairsManager{
		Pairs: FullStore{
			asset.Spot: {
				AssetEnabled: false,
			},
		},
	}
	assert.True(t, p.IsAssetSupported(asset.Spot), "Spot should be supported")
	assert.False(t, p.IsAssetSupported(asset.Index), "Index should not be supported")
}

// BenchmarkStorePairs stores each asset's available pairs in turn, as loading a config or updating tradable pairs does
func BenchmarkStorePairs(b *testing.B) {
	assets := asset.Items{asset.Spot, asset.Margin, asset.PerpetualSwap, asset.Futures, asset.Options, asset.Spread}
	sizes := []int{2500, 800, 300, 400, 3000, 50}
	available := make([]Pairs, len(assets))
	for i := range assets {
		available[i] = make(Pairs, sizes[i])
		for j := range available[i] {
			available[i][j] = NewPair(testCode("BENCH"+strconv.Itoa(i)+"X"+strconv.Itoa(j)), USDT)
		}
	}

	for b.Loop() {
		var p PairsManager
		for i, a := range assets {
			if err := p.StorePairs(a, available[i], false); err != nil {
				b.Fatal(err)
			}
		}
	}
}

// BenchmarkMatch matches symbols in the forms exchanges send them
func BenchmarkMatch(b *testing.B) {
	pairs := make(Pairs, 2000)
	for i := range pairs {
		pairs[i] = NewPair(testCode("BENCH"+strconv.Itoa(i)), USDT)
	}
	p := new(PairsManager)
	if err := p.StorePairs(asset.Spot, pairs, false); err != nil {
		b.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		symbol string
		match  func(string, asset.Item) (Pair, error)
	}{
		{name: "lower", symbol: "bench1234usdt", match: p.Match},
		{name: "upper", symbol: "BENCH1234USDT", match: p.Match},
		{name: "delimited", symbol: "BENCH1234-USDT", match: p.MatchDelimited},
	} {
		b.Run(tc.name, func(b *testing.B) {
			for b.Loop() {
				if _, err := tc.match(tc.symbol, asset.Spot); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
