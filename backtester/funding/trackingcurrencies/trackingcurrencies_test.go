package trackingcurrencies

import (
	"errors"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

func TestDoEverything(t *testing.T) {
	t.Parallel()

}

func TestFindMatchingUSDPairs(t *testing.T) {
	t.Parallel()
	type testPair struct {
		description    string
		initialPair    currency.Pair
		availablePairs *currency.PairStore
		basePair       currency.Pair
		quotePair      currency.Pair
		expectedErr    error
	}
	tests := []testPair{
		{
			description:    "already has USD",
			initialPair:    currency.NewPair(currency.BTC, currency.USDT),
			availablePairs: &currency.PairStore{Available: currency.Pairs{currency.NewPair(currency.BTC, currency.USDT)}},
			basePair:       currency.Pair{},
			quotePair:      currency.Pair{},
			expectedErr:    errCurrencyContainsUSD,
		},
		{
			description:    "successful",
			initialPair:    currency.NewPair(currency.BTC, currency.LTC),
			availablePairs: &currency.PairStore{Available: currency.Pairs{currency.NewPair(currency.BTC, currency.LTC), currency.NewPair(currency.BTC, currency.USDT), currency.NewPair(currency.LTC, currency.TUSD)}},
			basePair:       currency.NewPair(currency.BTC, currency.USDT),
			quotePair:      currency.NewPair(currency.LTC, currency.TUSD),
			expectedErr:    nil,
		},
		{
			description:    "quote currency has no matching USD pair",
			initialPair:    currency.NewPair(currency.BTC, currency.LTC),
			availablePairs: &currency.PairStore{Available: currency.Pairs{currency.NewPair(currency.BTC, currency.LTC), currency.NewPair(currency.BTC, currency.DAI)}},
			basePair:       currency.NewPair(currency.BTC, currency.DAI),
			quotePair:      currency.Pair{},
			expectedErr:    errNoMatchingQuoteUSDFound,
		},
		{
			description:    "base currency has no matching USD pair",
			initialPair:    currency.NewPair(currency.BTC, currency.LTC),
			availablePairs: &currency.PairStore{Available: currency.Pairs{currency.NewPair(currency.BTC, currency.LTC), currency.NewPair(currency.LTC, currency.USDT)}},
			basePair:       currency.Pair{},
			quotePair:      currency.NewPair(currency.LTC, currency.USDT),
			expectedErr:    errNoMatchingBaseUSDFound,
		},
		{
			description:    "both base and quote don't have USD pairs",
			initialPair:    currency.NewPair(currency.BTC, currency.LTC),
			availablePairs: &currency.PairStore{Available: currency.Pairs{currency.NewPair(currency.BTC, currency.LTC)}},
			basePair:       currency.Pair{},
			quotePair:      currency.Pair{},
			expectedErr:    errNoMatchingPairUSDFound,
		},
		{
			description:    "currency doesnt exist in available pairs",
			initialPair:    currency.NewPair(currency.BTC, currency.LTC),
			availablePairs: &currency.PairStore{Available: currency.Pairs{currency.NewPair(currency.BTC, currency.DOGE)}},
			basePair:       currency.Pair{},
			quotePair:      currency.Pair{},
			expectedErr:    errCurrencyNotFoundInPairs,
		},
	}
	for i := range tests {
		basePair, quotePair, err := FindMatchingUSDPairs(tests[i].initialPair, tests[i].availablePairs)
		if !errors.Is(err, tests[i].expectedErr) {
			t.Fatalf("'%v' received '%v' expected '%v'", tests[i].description, err, tests[i].expectedErr)
		}
		if basePair != tests[i].basePair {
			t.Fatalf("'%v' received '%v' expected '%v'", tests[i].description, basePair, tests[i].basePair)
		}
		if quotePair != tests[i].quotePair {
			t.Fatalf("'%v' received '%v' expected '%v'", tests[i].description, quotePair, tests[i].quotePair)
		}
	}
}

func TestPairContainsUSD(t *testing.T) {
	t.Parallel()
	type testPair struct {
		expected bool
		pair     currency.Pair
	}
	pairs := []testPair{
		{
			true,
			currency.NewPair(currency.BTC, currency.USDT),
		},
		{
			false,
			currency.NewPair(currency.BTC, currency.DOGE),
		},
		{
			true,
			currency.NewPair(currency.USD, currency.LTC),
		},
		{
			true,
			currency.NewPair(currency.BTC, currency.DAI),
		},
		{
			true,
			currency.NewPair(currency.BTC, currency.BUSD),
		},
		{
			true,
			currency.NewPair(currency.BTC, currency.USD),
		},
		{
			false,
			currency.NewPair(currency.BTC, currency.AUD),
		},
		{
			true,
			currency.NewPair(currency.BTC, currency.USDC),
		},
		{
			true,
			currency.NewPair(currency.BTC, currency.TUSD),
		},
		{
			true,
			currency.NewPair(currency.BTC, currency.ZUSD),
		},
		{
			true,
			currency.NewPair(currency.BTC, currency.PAX),
		},
	}
	var resp bool
	for i := range pairs {
		resp = PairContainsUSD(pairs[i].pair)
		if resp != pairs[i].expected {
			t.Errorf("expected %v received %v", pairs[i], resp)
		}
	}
}
