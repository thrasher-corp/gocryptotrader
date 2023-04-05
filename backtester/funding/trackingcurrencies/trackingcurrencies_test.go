package trackingcurrencies

import (
	"errors"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var (
	eName = "binance"
	a     = asset.Spot
	b     = currency.BTC
	q     = currency.USDT
)

func TestCreateUSDTrackingPairs(t *testing.T) {
	t.Parallel()

	_, err := CreateUSDTrackingPairs(nil, nil)
	if !errors.Is(err, errNilPairsReceived) {
		t.Errorf("received '%v' expected '%v'", err, errNilPairsReceived)
	}

	_, err = CreateUSDTrackingPairs([]TrackingPair{{}}, nil)
	if !errors.Is(err, errExchangeManagerRequired) {
		t.Errorf("received '%v' expected '%v'", err, errExchangeManagerRequired)
	}

	em := engine.NewExchangeManager()
	_, err = CreateUSDTrackingPairs([]TrackingPair{{Exchange: eName}}, em)
	if !errors.Is(err, engine.ErrExchangeNotFound) {
		t.Errorf("received '%v' expected '%v'", err, engine.ErrExchangeNotFound)
	}

	s1 := TrackingPair{
		Exchange: eName,
		Asset:    a,
		Base:     b,
		Quote:    q,
	}

	exch, err := em.NewExchangeByName(eName)
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	cp := currency.NewPair(s1.Base, s1.Quote)
	cp2 := currency.NewPair(currency.LTC, currency.USDT)
	cp3 := currency.NewPair(currency.LTC, currency.BTC)
	exchB := exch.GetBase()
	eba := exchB.CurrencyPairs.Pairs[a]
	eba.Available = eba.Available.Add(cp)
	eba.Enabled = eba.Enabled.Add(cp)
	eba.Available = eba.Available.Add(cp2)
	eba.Enabled = eba.Enabled.Add(cp2)
	eba.Available = eba.Available.Add(cp3)
	eba.Enabled = eba.Enabled.Add(cp3)
	eba.AssetEnabled = convert.BoolPtr(true)

	err = em.Add(exch)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
	resp, err := CreateUSDTrackingPairs([]TrackingPair{s1}, em)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if len(resp) != 1 {
		t.Error("expected 1 currency setting as it contains a USDT equiv")
	}
	s1.Base = currency.LTC
	s1.Quote = currency.BTC

	resp, err = CreateUSDTrackingPairs([]TrackingPair{s1}, em)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if len(resp) != 3 {
		t.Error("expected 3 currency settings as it did not contain a USDT equiv")
	}
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
			description:    "already has USDT",
			initialPair:    currency.NewPair(currency.BTC, currency.USDT),
			availablePairs: &currency.PairStore{Available: currency.Pairs{currency.NewPair(currency.BTC, currency.USDT)}},
			basePair:       currency.EMPTYPAIR,
			quotePair:      currency.EMPTYPAIR,
			expectedErr:    ErrCurrencyContainsUSD,
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
			description:    "quote currency has no matching USDT pair",
			initialPair:    currency.NewPair(currency.BTC, currency.LTC),
			availablePairs: &currency.PairStore{Available: currency.Pairs{currency.NewPair(currency.BTC, currency.LTC), currency.NewPair(currency.BTC, currency.DAI)}},
			basePair:       currency.NewPair(currency.BTC, currency.DAI),
			quotePair:      currency.EMPTYPAIR,
			expectedErr:    errNoMatchingQuoteUSDFound,
		},
		{
			description:    "base currency has no matching USDT pair",
			initialPair:    currency.NewPair(currency.BTC, currency.LTC),
			availablePairs: &currency.PairStore{Available: currency.Pairs{currency.NewPair(currency.BTC, currency.LTC), currency.NewPair(currency.LTC, currency.USDT)}},
			basePair:       currency.EMPTYPAIR,
			quotePair:      currency.NewPair(currency.LTC, currency.USDT),
			expectedErr:    errNoMatchingBaseUSDFound,
		},
		{
			description:    "both base and quote don't have USDT pairs",
			initialPair:    currency.NewPair(currency.BTC, currency.LTC),
			availablePairs: &currency.PairStore{Available: currency.Pairs{currency.NewPair(currency.BTC, currency.LTC)}},
			basePair:       currency.EMPTYPAIR,
			quotePair:      currency.EMPTYPAIR,
			expectedErr:    errNoMatchingPairUSDFound,
		},
		{
			description:    "currency doesn't exist in available pairs",
			initialPair:    currency.NewPair(currency.BTC, currency.LTC),
			availablePairs: &currency.PairStore{Available: currency.Pairs{currency.NewPair(currency.BTC, currency.DOGE)}},
			basePair:       currency.EMPTYPAIR,
			quotePair:      currency.EMPTYPAIR,
			expectedErr:    errCurrencyNotFoundInPairs,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.description, func(t *testing.T) {
			t.Parallel()
			basePair, quotePair, err := findMatchingUSDPairs(tt.initialPair, tt.availablePairs)
			if !errors.Is(err, tt.expectedErr) {
				t.Fatalf("'%v' received '%v' expected '%v'", tt.description, err, tt.expectedErr)
			}
			if basePair != tt.basePair {
				t.Fatalf("'%v' received '%v' expected '%v'", tt.description, basePair, tt.basePair)
			}
			if quotePair != tt.quotePair {
				t.Fatalf("'%v' received '%v' expected '%v'", tt.description, quotePair, tt.quotePair)
			}
		})
	}
}

func TestPairContainsUSD(t *testing.T) {
	t.Parallel()
	type testPair struct {
		description string
		expected    bool
		pair        currency.Pair
	}
	pairs := []testPair{
		{
			"btcusdt",
			true,
			currency.NewPair(currency.BTC, currency.USDT),
		},
		{
			"btcdoge",
			false,
			currency.NewPair(currency.BTC, currency.DOGE),
		},
		{
			"usdltc",
			true,
			currency.NewPair(currency.USDT, currency.LTC),
		},
		{
			"btcdai",
			true,
			currency.NewPair(currency.BTC, currency.DAI),
		},
		{
			"btcbusd",
			true,
			currency.NewPair(currency.BTC, currency.BUSD),
		},
		{
			"btcusd",
			true,
			currency.NewPair(currency.BTC, currency.USDT),
		},
		{
			"btcaud",
			false,
			currency.NewPair(currency.BTC, currency.AUD),
		},
		{
			"btcusdc",
			true,
			currency.NewPair(currency.BTC, currency.USDC),
		},
		{
			"btctusd",
			true,
			currency.NewPair(currency.BTC, currency.TUSD),
		},
		{
			"btczusd",
			true,
			currency.NewPair(currency.BTC, currency.ZUSD),
		},
		{
			"btcpax",
			true,
			currency.NewPair(currency.BTC, currency.PAX),
		},
	}
	for i := range pairs {
		tt := pairs[i]
		t.Run(tt.description, func(t *testing.T) {
			t.Parallel()
			resp := pairContainsUSD(tt.pair)
			if resp != tt.expected {
				t.Errorf("expected %v received %v", tt, resp)
			}
		})
	}
}
