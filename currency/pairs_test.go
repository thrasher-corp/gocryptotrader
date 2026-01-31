package currency

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

func TestPairsUpper(t *testing.T) {
	t.Parallel()
	pairs, err := NewPairsFromStrings([]string{"btc_usd", "btc_aud", "btc_ltc"})
	if err != nil {
		t.Fatal(err)
	}
	if expected := "BTC_USD,BTC_AUD,BTC_LTC"; pairs.Upper().Join() != expected {
		t.Errorf("Pairs Join() error expected %s but received %s",
			expected, pairs.Upper().Join())
	}
}

func TestPairsLower(t *testing.T) {
	t.Parallel()
	pairs, err := NewPairsFromStrings([]string{"BTC_USD", "BTC_AUD", "BTC_LTC"})
	if err != nil {
		t.Fatal(err)
	}
	if expected := "btc_usd,btc_aud,btc_ltc"; pairs.Lower().Join() != expected {
		t.Errorf("Pairs Join() error expected %s but received %s",
			expected, pairs.Lower().Join())
	}
}

func TestPairsString(t *testing.T) {
	t.Parallel()
	pairs, err := NewPairsFromStrings([]string{"btc_usd", "btc_aud", "btc_ltc"})
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"btc_usd", "btc_aud", "btc_ltc"}

	for i, p := range pairs {
		if p.String() != expected[i] {
			t.Errorf("Pairs String() error expected %s but received %s",
				expected, p.String())
		}
	}
}

func TestPairsFromString(t *testing.T) {
	t.Parallel()
	_, err := NewPairsFromString("", "")
	assert.ErrorIs(t, err, errNoDelimiter)
	_, err = NewPairsFromString("", ",")
	assert.ErrorIs(t, err, ErrCreatingPair)

	pairs, err := NewPairsFromString("ALGO-AUD,BAT-AUD,BCH-AUD,BSV-AUD,BTC-AUD,COMP-AUD,ENJ-AUD,ETC-AUD,ETH-AUD,ETH-BTC,GNT-AUD,LINK-AUD,LTC-AUD,LTC-BTC,MCAU-AUD,OMG-AUD,POWR-AUD,UNI-AUD,USDT-AUD,XLM-AUD,XRP-AUD,XRP-BTC", ",")
	require.NoError(t, err)

	expected := []string{
		"ALGO-AUD", "BAT-AUD", "BCH-AUD", "BSV-AUD", "BTC-AUD",
		"COMP-AUD", "ENJ-AUD", "ETC-AUD", "ETH-AUD", "ETH-BTC", "GNT-AUD",
		"LINK-AUD", "LTC-AUD", "LTC-BTC", "MCAU-AUD", "OMG-AUD", "POWR-AUD",
		"UNI-AUD", "USDT-AUD", "XLM-AUD", "XRP-AUD", "XRP-BTC",
	}

	assert.Equal(t, expected, pairs.Strings(), "NewPairsFromString should return the correct pairs")
}

func TestPairsJoin(t *testing.T) {
	t.Parallel()
	pairs, err := NewPairsFromStrings([]string{"btc_usd", "btc_aud", "btc_ltc"})
	if err != nil {
		t.Fatal(err)
	}
	expected := "btc_usd,btc_aud,btc_ltc"

	if pairs.Join() != expected {
		t.Errorf("Pairs Join() error expected %s but received %s",
			expected, pairs.Join())
	}
}

func TestPairsFormat(t *testing.T) {
	t.Parallel()
	pairs, err := NewPairsFromStrings([]string{"btc_usd", "btc_aud", "btc_ltc"})
	require.NoError(t, err, "NewPairsFromStrings must not error")
	assert.Equal(t, "BTC-USD,BTC-AUD,BTC-LTC", pairs.Format(PairFormat{Delimiter: "-", Uppercase: true}).Join(), "Format should return the correct value")
	assert.Equal(t, "btc:usd,btc:aud,btc:ltc", pairs.Format(PairFormat{Delimiter: ":", Uppercase: false}).Join(), "Format should return the correct value")
}

func TestPairsUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var unmarshalHere Pairs
	configPairs := ""
	encoded, err := json.Marshal(configPairs)
	if err != nil {
		t.Fatal("Pairs UnmarshalJSON() error", err)
	}

	err = json.Unmarshal([]byte{1, 3, 3, 7}, &unmarshalHere)
	if err == nil {
		t.Fatal("error cannot be nil")
	}

	err = json.Unmarshal(encoded, &unmarshalHere)
	if err != nil {
		t.Fatal("Pairs UnmarshalJSON() error", err)
	}

	configPairs = "btc_usd,btc_aud,btc_ltc"
	encoded, err = json.Marshal(configPairs)
	if err != nil {
		t.Fatal("Pairs UnmarshalJSON() error", err)
	}

	err = json.Unmarshal(encoded, &unmarshalHere)
	if err != nil {
		t.Fatal("Pairs UnmarshalJSON() error", err)
	}

	err = json.Unmarshal(encoded, &unmarshalHere)
	if err != nil {
		t.Fatal("Pairs UnmarshalJSON() error", err)
	}

	if unmarshalHere.Join() != configPairs {
		t.Errorf("Pairs UnmarshalJSON() error expected %s but received %s",
			configPairs, unmarshalHere.Join())
	}
}

func TestPairsMarshalJSON(t *testing.T) {
	t.Parallel()
	pairs, err := NewPairsFromStrings([]string{"btc_usd", "btc_aud", "btc_ltc"})
	if err != nil {
		t.Fatal(err)
	}

	quickstruct := struct {
		Pairs Pairs `json:"soManyPairs"`
	}{
		Pairs: pairs,
	}

	encoded, err := json.Marshal(quickstruct)
	if err != nil {
		t.Fatal("Pairs MarshalJSON() error", err)
	}

	expected := `{"soManyPairs":"btc_usd,btc_aud,btc_ltc"}`
	if string(encoded) != expected {
		t.Errorf("Pairs MarshalJSON() error expected %s but received %s",
			expected, string(encoded))
	}
}

func TestRemovePairsByFilter(t *testing.T) {
	t.Parallel()
	pairs := Pairs{
		NewBTCUSD(),
		NewPair(LTC, USD),
		NewPair(LTC, USDT),
	}

	pairs = pairs.RemovePairsByFilter(USDT)
	if pairs.Contains(NewPair(LTC, USDT), true) {
		t.Error("TestRemovePairsByFilter unexpected result")
	}
}

func TestGetPairsByFilter(t *testing.T) {
	t.Parallel()
	pairs := Pairs{
		NewBTCUSD(),
		NewPair(LTC, USD),
		NewPair(LTC, USDT),
	}

	filtered := pairs.GetPairsByFilter(LTC)
	if !filtered.Contains(NewPair(LTC, USDT), true) &&
		!filtered.Contains(NewPair(LTC, USD), true) {
		t.Error("TestRemovePairsByFilter unexpected result")
	}
}

func TestRemove(t *testing.T) {
	t.Parallel()
	oldPairs := Pairs{
		NewBTCUSD(),
		NewPair(LTC, USD),
		NewPair(LTC, USDT),
	}

	compare := slices.Clone(oldPairs)

	newPairs := oldPairs.Remove(oldPairs[:2]...)

	err := compare.ContainsAll(oldPairs, true)
	assert.NoError(t, err, "Remove should not affect the original pairs")

	require.Len(t, newPairs, 1, "Remove must remove a pair")
	require.Equal(t, oldPairs[2], newPairs[0], "Remove must leave the final pair")

	newPairs = newPairs.Remove(oldPairs[0])
	assert.Len(t, newPairs, 1, "Remove should have no effect on non-included pairs")
}

func TestAdd(t *testing.T) {
	t.Parallel()
	orig := Pairs{NewBTCUSD(), NewPair(LTC, USD), NewPair(LTC, USDT)}
	p := slices.Clone(orig)
	p2 := Pairs{NewBTCUSDT(), NewPair(ETH, USD), NewPair(BTC, ETH)}

	pT := p.Add(p...)
	assert.Equal(t, pT.Join(), orig.Join(), "Adding only existing pairs should return same Pairs")
	assert.Equal(t, p.Join(), orig.Join(), "Should not effect original")

	pT = p.Add(p2...)
	assert.Equal(t, pT.Join(), append(orig, p2...).Join(), "Adding new pairs should return correct Pairs")
	assert.Equal(t, p.Join(), orig.Join(), "Should not effect original")

	p = slices.Grow(slices.Clone(orig), len(p2)) // Grow so that append doesn't alloc
	pT1 := p.Add(p2[0])
	pT2 := p.Add(p2[1])
	pT1[3] = p2[2] // If Add doesn't allocate an new underlying array, this would affect PT2 as well
	assert.Equal(t, p.Join(), orig.Join(), "Pairs underlying array should not be shared with original")
	assert.Equal(t, pT2.Join(), append(orig, p2[1]).Join(), "Pairs underlying array should not be shared with siblings")
}

func TestContains(t *testing.T) {
	t.Parallel()
	pairs := Pairs{
		NewBTCUSD(),
		NewPair(LTC, USD),
		NewPair(USD, ZRX),
	}

	if !pairs.Contains(NewBTCUSD(), true) {
		t.Errorf("TestContains: Expected pair was not found")
	}

	if pairs.Contains(NewPair(USD, BTC), true) {
		t.Errorf("TestContains: Unexpected pair was found")
	}

	if !pairs.Contains(NewPair(USD, BTC), false) {
		t.Errorf("TestContains: Expected pair was not found")
	}

	if pairs.Contains(NewPair(ETH, USD), false) {
		t.Errorf("TestContains: Non-existent pair was found")
	}
}

func TestContainsAll(t *testing.T) {
	t.Parallel()
	pairs := Pairs{
		NewBTCUSD(),
		NewPair(LTC, USD),
		NewPair(USD, ZRX),
	}

	err := pairs.ContainsAll(nil, true)
	require.ErrorIs(t, err, ErrCurrencyPairsEmpty)

	err = pairs.ContainsAll(Pairs{NewBTCUSD()}, true)
	require.NoError(t, err)

	err = pairs.ContainsAll(Pairs{NewPair(USD, BTC)}, false)
	require.NoError(t, err)

	err = pairs.ContainsAll(Pairs{NewPair(XRP, BTC)}, false)
	require.ErrorIs(t, err, ErrPairNotContainedInAvailablePairs)

	err = pairs.ContainsAll(Pairs{NewPair(XRP, BTC)}, true)
	require.ErrorIs(t, err, ErrPairNotContainedInAvailablePairs)

	err = pairs.ContainsAll(pairs, true)
	require.NoError(t, err)

	err = pairs.ContainsAll(pairs, false)
	require.NoError(t, err)

	duplication := Pairs{
		NewBTCUSD(),
		NewPair(LTC, USD),
		NewPair(USD, ZRX),
		NewPair(USD, ZRX),
	}

	err = pairs.ContainsAll(duplication, false)
	require.ErrorIs(t, err, ErrPairDuplication)
}

func TestDeriveFrom(t *testing.T) {
	t.Parallel()
	_, err := Pairs{}.DeriveFrom("")
	require.ErrorIs(t, err, ErrCurrencyPairsEmpty)

	testCases := Pairs{
		NewBTCUSDT(),
		NewPair(USDC, USDT),
		NewPair(USDC, USD),
		NewPair(BTC, LTC),
		NewPair(LTC, SAFEMARS),
	}

	_, err = testCases.DeriveFrom("")
	require.ErrorIs(t, err, errSymbolEmpty)

	_, err = testCases.DeriveFrom("btcUSD")
	require.ErrorIs(t, err, ErrPairNotFound)

	got, err := testCases.DeriveFrom("USDCUSD")
	require.NoError(t, err)

	if got.Upper().String() != "USDCUSD" {
		t.Fatalf("received: '%v' but expected: '%v'", got.Upper().String(), "USDCUSD")
	}
}

func TestGetCrypto(t *testing.T) {
	t.Parallel()
	pairs := Pairs{
		NewBTCUSD(),
		NewPair(LTC, USD),
		NewPair(USD, NZD),
		NewPair(LTC, USDT),
	}
	contains(t, []Code{BTC, LTC, USDT}, pairs.GetCrypto())
}

func TestGetFiat(t *testing.T) {
	t.Parallel()
	pairs := Pairs{
		NewBTCUSD(),
		NewPair(LTC, USD),
		NewPair(USD, NZD),
		NewPair(LTC, USDT),
	}
	contains(t, []Code{USD, NZD}, pairs.GetFiat())
}

func TestGetCurrencies(t *testing.T) {
	t.Parallel()
	pairs := Pairs{
		NewBTCUSD(),
		NewPair(LTC, USD),
		NewPair(USD, NZD),
		NewPair(LTC, USDT),
	}
	contains(t, []Code{BTC, USD, LTC, NZD, USDT}, pairs.GetCurrencies())
}

func TestGetStables(t *testing.T) {
	t.Parallel()
	pairs := Pairs{
		NewBTCUSD(),
		NewPair(LTC, USD),
		NewPair(USD, NZD),
		NewPair(LTC, USDT),
		NewPair(DAI, USDT),
		NewPair(LTC, USDC),
		NewPair(USDP, USDT),
	}
	contains(t, []Code{USDT, USDP, USDC, DAI}, pairs.GetStables())
}

func contains(t *testing.T, c1, c2 []Code) {
	t.Helper()
codes:
	for x := range c1 {
		for y := range c2 {
			if c1[x].Equal(c2[y]) {
				continue codes
			}
		}
		t.Fatalf("cannot find currency %s in returned currency list %v", c1[x], c2)
	}
}

// Current: 6176922	       260.0 ns/op	      48 B/op	       1 allocs/op
// Prior: 2575473	       474.2 ns/op	     112 B/op	       3 allocs/op
func BenchmarkGetCrypto(b *testing.B) {
	pairs := Pairs{
		NewBTCUSD(),
		NewPair(LTC, USD),
		NewPair(USD, NZD),
		NewPair(LTC, USDT),
	}

	for b.Loop() {
		_ = pairs.GetCrypto()
	}
}

func TestGetMatch(t *testing.T) {
	t.Parallel()
	pairs := Pairs{
		NewBTCUSD(),
		NewPair(LTC, USD),
		NewPair(USD, NZD),
		NewPair(LTC, USDT),
	}

	_, err := pairs.GetMatch(NewPair(BTC, WABI))
	require.ErrorIs(t, err, ErrPairNotFound)

	expected := NewBTCUSD()
	match, err := pairs.GetMatch(expected)
	require.NoError(t, err)

	if !match.Equal(expected) {
		t.Fatalf("received: '%v' but expected '%v'", match, expected)
	}

	match, err = pairs.GetMatch(NewPair(USD, BTC))
	require.NoError(t, err)

	if !match.Equal(expected) {
		t.Fatalf("received: '%v' but expected '%v'", match, expected)
	}
}

func TestGetStablesMatch(t *testing.T) {
	t.Parallel()
	pairs := Pairs{
		NewBTCUSD(),
		NewPair(LTC, USD),
		NewPair(USD, NZD),
		NewPair(LTC, USDT),
		NewPair(LTC, DAI),
		NewPair(USDT, XRP),
		NewPair(DAI, XRP),
	}

	stablePairs := pairs.GetStablesMatch(BTC)
	if len(stablePairs) != 0 {
		t.Fatal("unexpected value")
	}

	stablePairs = pairs.GetStablesMatch(USD)
	if len(stablePairs) != 0 {
		t.Fatal("unexpected value")
	}

	stablePairs = pairs.GetStablesMatch(LTC)
	if len(stablePairs) != 2 {
		t.Fatal("unexpected value")
	}

	if !stablePairs[0].Equal(NewPair(LTC, USDT)) {
		t.Fatal("unexpected value")
	}

	if !stablePairs[1].Equal(NewPair(LTC, DAI)) {
		t.Fatal("unexpected value")
	}

	stablePairs = pairs.GetStablesMatch(XRP)
	if len(stablePairs) != 2 {
		t.Fatal("unexpected value")
	}

	if !stablePairs[0].Equal(NewPair(USDT, XRP)) {
		t.Fatal("unexpected value")
	}

	if !stablePairs[1].Equal(NewPair(DAI, XRP)) {
		t.Fatal("unexpected value")
	}
}

// Current: 5594431	       217.4 ns/op	     168 B/op	       8 allocs/op
// Prev:  3490366	       373.4 ns/op	     296 B/op	      11 allocs/op
func BenchmarkPairsString(b *testing.B) {
	pairs := Pairs{
		NewBTCUSD(),
		NewPair(LTC, USD),
		NewPair(USD, NZD),
		NewPair(LTC, USDT),
		NewPair(LTC, DAI),
		NewPair(USDT, XRP),
		NewPair(DAI, XRP),
	}

	for b.Loop() {
		_ = pairs.Strings()
	}
}

// Current:  6691011	       184.6 ns/op	     352 B/op	       1 allocs/op
// Prev:  3746151	       317.1 ns/op	     720 B/op	       4 allocs/op
func BenchmarkPairsFormat(b *testing.B) {
	pairs := Pairs{
		NewBTCUSD(),
		NewPair(LTC, USD),
		NewPair(USD, NZD),
		NewPair(LTC, USDT),
		NewPair(LTC, DAI),
		NewPair(USDT, XRP),
		NewPair(DAI, XRP),
	}

	formatting := PairFormat{Delimiter: "/", Uppercase: false}

	for b.Loop() {
		_ = pairs.Format(formatting)
	}
}

// current: 13075897	       100.4 ns/op	     352 B/op	       1 allocs/o
// prev: 8188616	       148.0 ns/op	     336 B/op	       3 allocs/op
func BenchmarkRemovePairsByFilter(b *testing.B) {
	pairs := Pairs{
		NewBTCUSD(),
		NewPair(LTC, USD),
		NewPair(USD, NZD),
		NewPair(LTC, USDT),
		NewPair(LTC, DAI),
		NewPair(USDT, XRP),
		NewPair(DAI, XRP),
	}

	for b.Loop() {
		_ = pairs.RemovePairsByFilter(USD)
	}
}

func TestPairsContainsCurrency(t *testing.T) {
	t.Parallel()
	pairs := Pairs{
		NewBTCUSD(),
		NewPair(LTC, USD),
		NewPair(USD, NZD),
		NewPair(LTC, USDT),
		NewPair(LTC, DAI),
		NewPair(USDT, XRP),
		NewPair(DAI, XRP),
	}

	if !pairs.ContainsCurrency(BTC) {
		t.Fatalf("expected %s to be %v", BTC, true)
	}
	if !pairs.ContainsCurrency(USD) {
		t.Fatalf("expected %s to be %v", USD, true)
	}
	if !pairs.ContainsCurrency(LTC) {
		t.Fatalf("expected %s to be %v", LTC, true)
	}
	if !pairs.ContainsCurrency(DAI) {
		t.Fatalf("expected %s to be %v", DAI, true)
	}
	if !pairs.ContainsCurrency(XRP) {
		t.Fatalf("expected %s to be %v", XRP, true)
	}
	if pairs.ContainsCurrency(ATOM3L) {
		t.Fatalf("expected %s to be %v", ATOM3L, false)
	}
}

func TestGetPairsByCurrencies(t *testing.T) {
	t.Parallel()
	available := Pairs{
		NewBTCUSD(),
		NewPair(LTC, USD),
		NewPair(USD, NZD),
		NewPair(LTC, USDT),
		NewPair(LTC, DAI),
		NewPair(USDT, XRP),
		NewPair(DAI, XRP),
	}

	enabled := available.GetPairsByCurrencies(Currencies{USD})
	if len(enabled) != 0 {
		t.Fatalf("received %v but expected %v", enabled, "no pairs")
	}

	enabled = available.GetPairsByCurrencies(Currencies{USD, BTC})
	if !enabled.Contains(NewBTCUSD(), true) {
		t.Fatalf("received %v but expected to contain %v", enabled, NewBTCUSD())
	}

	enabled = available.GetPairsByCurrencies(Currencies{USD, BTC, LTC, NZD, USDT, DAI})
	if len(enabled) != 5 {
		t.Fatalf("received %v but expected  %v", enabled, 5)
	}
}

func TestValidateAndConform(t *testing.T) {
	t.Parallel()

	conformMe := Pairs{
		NewBTCUSD(),
		NewPair(LTC, USD),
		NewPair(USD, NZD),
		NewPair(LTC, USDT),
		NewPair(LTC, DAI),
		NewPair(USDT, XRP),
		NewPair(EMPTYCODE, EMPTYCODE),
	}

	_, err := conformMe.ValidateAndConform(EMPTYFORMAT, false)
	require.ErrorIs(t, err, ErrCurrencyPairEmpty)

	duplication, err := NewPairFromString("linkusdt")
	if err != nil {
		t.Fatal(err)
	}

	conformMe = Pairs{
		NewBTCUSD(),
		NewPair(LTC, USD),
		NewPair(LINK, USDT),
		NewPair(USD, NZD),
		NewPair(LTC, USDT),
		NewPair(LTC, DAI),
		NewPair(USDT, XRP),
		duplication,
	}

	_, err = conformMe.ValidateAndConform(EMPTYFORMAT, false)
	require.ErrorIs(t, err, ErrPairDuplication)

	conformMe = Pairs{
		NewBTCUSD(),
		NewPair(LTC, USD),
		NewPair(LINK, USDT),
		NewPair(USD, NZD),
		NewPair(LTC, USDT),
		NewPair(LTC, DAI),
		NewPair(USDT, XRP),
	}

	formatted, err := conformMe.ValidateAndConform(EMPTYFORMAT, false)
	require.NoError(t, err)

	expected := "btcusd,ltcusd,linkusdt,usdnzd,ltcusdt,ltcdai,usdtxrp"

	if formatted.Join() != expected {
		t.Fatalf("received: '%v' but expected '%v'", formatted.Join(), expected)
	}

	formatted, err = formatted.ValidateAndConform(PairFormat{Delimiter: DashDelimiter, Uppercase: true}, false)
	require.NoError(t, err)

	expected = "BTC-USD,LTC-USD,LINK-USDT,USD-NZD,LTC-USDT,LTC-DAI,USDT-XRP"

	if formatted.Join() != expected {
		t.Fatalf("received: '%v' but expected '%v'", formatted.Join(), expected)
	}

	formatted, err = formatted.ValidateAndConform(PairFormat{
		Delimiter: UnderscoreDelimiter,
		Uppercase: false,
	},
		true)
	require.NoError(t, err)

	expected = "BTC-USD,LTC-USD,LINK-USDT,USD-NZD,LTC-USDT,LTC-DAI,USDT-XRP"

	if formatted.Join() != expected {
		t.Fatalf("received: '%v' but expected '%v'", formatted.Join(), expected)
	}
}

func TestPairs_GetFormatting(t *testing.T) {
	t.Parallel()
	pFmt, err := Pairs{NewBTCUSDT()}.GetFormatting()
	require.NoError(t, err)
	assert.True(t, pFmt.Uppercase)
	assert.Empty(t, pFmt.Delimiter)

	pFmt, err = Pairs{NewPairWithDelimiter("eth", "usdt", "/")}.GetFormatting()
	require.NoError(t, err)
	assert.False(t, pFmt.Uppercase)
	assert.Equal(t, "/", pFmt.Delimiter)

	_, err = Pairs{NewBTCUSDT(), NewPairWithDelimiter("eth", "usdt", "/")}.GetFormatting()
	require.ErrorIs(t, err, errPairFormattingInconsistent)

	_, err = Pairs{NewPairWithDelimiter("eth", "USDT", "/")}.GetFormatting()
	require.ErrorIs(t, err, errPairFormattingInconsistent)

	_, err = Pairs{NewPairWithDelimiter("eth", "usdt", "/"), NewPairWithDelimiter("eth", "usdt", "/")}.GetFormatting()
	require.NoError(t, err)

	_, err = Pairs{NewPairWithDelimiter("eth", "usdt", "/"), NewPairWithDelimiter("eth", "usdt", "|")}.GetFormatting()
	require.ErrorIs(t, err, errPairFormattingInconsistent)

	_, err = Pairs{NewPairWithDelimiter("eth", "420", "/"), NewPairWithDelimiter("eth", "420", "/")}.GetFormatting()
	require.NoError(t, err)
	_, err = Pairs{NewPairWithDelimiter("ETH", "420", "/"), NewPairWithDelimiter("ETH", "420", "/")}.GetFormatting()
	require.NoError(t, err)
	_, err = Pairs{NewPairWithDelimiter("420", "ETH", "/"), NewPairWithDelimiter("420", "ETH", "/")}.GetFormatting()
	require.NoError(t, err)
	_, err = Pairs{NewPairWithDelimiter("420", "eth", "/"), NewPairWithDelimiter("420", "eth", "/")}.GetFormatting()
	require.NoError(t, err)
	_, err = Pairs{NewPairWithDelimiter("420", "eth", "/"), NewPairWithDelimiter("eth", "420", "/")}.GetFormatting()
	require.NoError(t, err)
	_, err = Pairs{NewPairWithDelimiter("420", "ETH", "/"), NewPairWithDelimiter("ETH", "420", "/")}.GetFormatting()
	require.NoError(t, err)
}

func TestGetPairsByQuote(t *testing.T) {
	t.Parallel()

	var available Pairs
	_, err := available.GetPairsByQuote(EMPTYCODE)
	require.ErrorIs(t, err, ErrCurrencyPairsEmpty)

	available = Pairs{
		NewBTCUSD(),
		NewPair(LTC, USD),
		NewPair(USD, NZD),
		NewPair(LTC, USDT),
		NewPair(LTC, DAI),
		NewPair(USDT, XRP),
		NewPair(DAI, XRP),
	}

	_, err = available.GetPairsByQuote(EMPTYCODE)
	require.ErrorIs(t, err, ErrCurrencyCodeEmpty)

	got, err := available.GetPairsByQuote(USD)
	require.NoError(t, err)

	if len(got) != 2 {
		t.Fatalf("received: '%v' but expected '%v'", len(got), 2)
	}

	got, err = available.GetPairsByQuote(BTC)
	require.NoError(t, err)

	if len(got) != 0 {
		t.Fatalf("received: '%v' but expected '%v'", len(got), 0)
	}
}

func TestGetPairsByBase(t *testing.T) {
	t.Parallel()

	var available Pairs
	_, err := available.GetPairsByBase(EMPTYCODE)
	require.ErrorIs(t, err, ErrCurrencyPairsEmpty)

	available = Pairs{
		NewBTCUSD(),
		NewPair(LTC, USD),
		NewPair(USD, NZD),
		NewPair(LTC, USDT),
		NewPair(LTC, DAI),
		NewPair(USDT, XRP),
		NewPair(DAI, XRP),
	}

	_, err = available.GetPairsByBase(EMPTYCODE)
	require.ErrorIs(t, err, ErrCurrencyCodeEmpty)

	got, err := available.GetPairsByBase(USD)
	require.NoError(t, err)

	if len(got) != 1 {
		t.Fatalf("received: '%v' but expected '%v'", len(got), 1)
	}

	got, err = available.GetPairsByBase(LTC)
	require.NoError(t, err)

	if len(got) != 3 {
		t.Fatalf("received: '%v' but expected '%v'", len(got), 3)
	}
}

// TestPairsEqual exercises Pairs.Equal
func TestPairsEqual(t *testing.T) {
	t.Parallel()
	orig := Pairs{NewPairWithDelimiter("USDT", "BTC", "-"), NewPair(DAI, XRP), NewPair(DAI, BTC)}
	assert.True(t, orig.Equal(Pairs{NewPair(DAI, XRP), NewPair(DAI, BTC), NewPair(USDT, BTC)}), "Equal Pairs should return true")
	assert.Equal(t, "USDT-BTC", orig[0].String(), "Equal Pairs should not effect original order or format")
	assert.False(t, orig.Equal(Pairs{NewPair(DAI, XRP), NewPair(DAI, BTC), NewPair(USD, LTC)}), "UnEqual Pairs should return false")
}

func TestFindPairDifferences(t *testing.T) {
	pairList, err := NewPairsFromStrings([]string{defaultPairWDelimiter, "ETH-USD", "LTC-USD"})
	require.NoError(t, err)

	dash, err := NewPairsFromStrings([]string{"DASH-USD"})
	require.NoError(t, err)

	// Test new pair update
	diff, err := pairList.FindDifferences(dash, PairFormat{Delimiter: DashDelimiter, Uppercase: true})
	require.NoError(t, err)
	assert.Len(t, diff.New, 1)
	assert.Len(t, diff.Remove, 3)

	diff, err = pairList.FindDifferences(Pairs{}, EMPTYFORMAT)
	require.NoError(t, err)
	assert.Empty(t, diff.New)
	assert.Len(t, diff.Remove, 3)
	assert.True(t, diff.FormatDifference)

	diff, err = Pairs{}.FindDifferences(pairList, EMPTYFORMAT)
	require.NoError(t, err)
	assert.Len(t, diff.New, 3)
	assert.Empty(t, diff.Remove)
	assert.True(t, diff.FormatDifference)

	// Test that the supplied pair lists are the same, so
	// no newPairs or removedPairs
	diff, err = pairList.FindDifferences(pairList, PairFormat{Delimiter: DashDelimiter, Uppercase: true})
	require.NoError(t, err)
	assert.Empty(t, diff.New)
	assert.Empty(t, diff.Remove)
	assert.False(t, diff.FormatDifference)

	_, err = pairList.FindDifferences(Pairs{EMPTYPAIR}, EMPTYFORMAT)
	require.ErrorIs(t, err, ErrCurrencyPairEmpty)

	_, err = Pairs{EMPTYPAIR}.FindDifferences(pairList, EMPTYFORMAT)
	require.ErrorIs(t, err, ErrCurrencyPairEmpty)

	// Test duplication
	duplication, err := NewPairsFromStrings([]string{defaultPairWDelimiter, "ETH-USD", "LTC-USD", "ETH-USD"})
	require.NoError(t, err)

	_, err = pairList.FindDifferences(duplication, EMPTYFORMAT)
	require.ErrorIs(t, err, ErrPairDuplication)

	// This will allow for the removal of the duplicated item to be returned if
	// contained in the original list.
	diff, err = duplication.FindDifferences(pairList, EMPTYFORMAT)
	require.NoError(t, err)
	require.Len(t, diff.Remove, 1)
	require.True(t, diff.Remove[0].Equal(pairList[1]))

	original, err := NewPairsFromStrings([]string{"ETH-USD", "LTC-USD", "ETH-USD"})
	require.NoError(t, err)

	compare, err := NewPairsFromStrings([]string{"ETH-123", "LTC-123", "MEOW-123"})
	require.NoError(t, err)

	diff, err = original.FindDifferences(compare, PairFormat{Delimiter: DashDelimiter, Uppercase: true})
	require.NoError(t, err)
	require.False(t, diff.FormatDifference)
}

// 2208139	       509.3 ns/op	     288 B/op	       2 allocs/op (current)
//
// 1614865	       712.5 ns/op	     336 B/op	       8 allocs/op (prev)
func BenchmarkFindDifferences(b *testing.B) {
	original, err := NewPairsFromStrings([]string{"ETH-USD", "LTC-USD", "ETH-USD"})
	require.NoError(b, err)

	compare, err := NewPairsFromStrings([]string{"ETH-123", "LTC-123", "MEOW-123"})
	require.NoError(b, err)

	for b.Loop() {
		_, err = original.FindDifferences(compare, EMPTYFORMAT)
		require.NoError(b, err)
	}
}
