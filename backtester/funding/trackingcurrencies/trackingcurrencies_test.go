package trackingcurrencies

import (
	"errors"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/backtester/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
)

var (
	exch = "binance"
	a    = "spot"
	b    = "BTC"
	q    = "USDT"
)

func TestCreateUSDTrackingPairs(t *testing.T) {
	t.Parallel()

	_, err := CreateUSDTrackingPairs(nil)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	_, err = CreateUSDTrackingPairs([]config.CurrencySettings{})
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	_, err = CreateUSDTrackingPairs([]config.CurrencySettings{{}})
	if !errors.Is(err, engine.ErrExchangeNotFound) {
		t.Errorf("received '%v' expected '%v'", err, engine.ErrExchangeNotFound)
	}

	s1 := config.CurrencySettings{
		ExchangeName: exch,
		Asset:        a,
		Base:         b,
		Quote:        q,
	}
	resp, err := CreateUSDTrackingPairs([]config.CurrencySettings{s1})
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if len(resp) != 1 {
		t.Error("expected 1 currency setting as it contains a USD equiv")
	}
	s1.Base = "LTC"
	s1.Quote = "BTC"
	resp, err = CreateUSDTrackingPairs([]config.CurrencySettings{s1})
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if len(resp) != 3 {
		t.Error("expected 3 currency settings as it did not contain a USD equiv")
	}
}

func TestFindMatchingUSDPairs(t *testing.T) {
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
		tt := tests[i]
		t.Run(tt.description, func(t *testing.T) {
			t.Parallel()
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
		})
	}
}

func TestPairContainsUSD(t *testing.T) {
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
		tt := pairs[i]
		t.Run(tt.pair.String(), func(t *testing.T) {
			t.Parallel()
			resp = PairContainsUSD(pairs[i].pair)
			if resp != pairs[i].expected {
				t.Errorf("expected %v received %v", pairs[i], resp)
			}
		})
	}
}
