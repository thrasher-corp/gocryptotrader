package exchange

import (
	"errors"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var btcusd = currency.NewPair(currency.BTC, currency.USD)
var ltcusd = currency.NewPair(currency.LTC, currency.USD)
var btcltc = currency.NewPair(currency.BTC, currency.LTC)

func TestLoadTolerances(t *testing.T) {
	e := ExecutionTolerance{}
	err := e.LoadTolerances(nil)
	if !errors.Is(err, errCannotLoadTolerance) {
		t.Fatalf("expected error %v but received %v", errCannotLoadTolerance, err)
	}

	newLimits := []MinMaxLevel{
		{
			Pair:      btcusd,
			Asset:     asset.Spot,
			MinPrice:  100000,
			MaxPrice:  1000000,
			MinAmount: 1,
			MaxAmount: 10,
		},
	}

	err = e.LoadTolerances(newLimits)
	if !errors.Is(err, nil) {
		t.Fatalf("expected error %v but received %v", nil, err)
	}

	badLimit := []MinMaxLevel{
		{
			Pair:      btcusd,
			Asset:     asset.Spot,
			MinPrice:  2,
			MaxPrice:  1,
			MinAmount: 1,
			MaxAmount: 10,
		},
	}

	err = e.LoadTolerances(badLimit)
	if !errors.Is(err, errInvalidPriceLevels) {
		t.Fatalf("expected error %v but received %v", errInvalidPriceLevels, err)
	}

	badLimit = []MinMaxLevel{
		{
			Pair:      btcusd,
			Asset:     asset.Spot,
			MinPrice:  1,
			MaxPrice:  2,
			MinAmount: 10,
			MaxAmount: 9,
		},
	}

	err = e.LoadTolerances(badLimit)
	if !errors.Is(err, errInvalidAmountLevels) {
		t.Fatalf("expected error %v but received %v", errInvalidPriceLevels, err)
	}
}

func TestGetTolerance(t *testing.T) {
	e := ExecutionTolerance{}
	_, err := e.GetTolerance(asset.Spot, btcusd)
	if !errors.Is(err, ErrExchangeToleranceNotLoaded) {
		t.Fatalf("expected error %v but received %v", ErrExchangeToleranceNotLoaded, err)
	}

	newLimits := []MinMaxLevel{
		{
			Pair:      btcusd,
			Asset:     asset.Spot,
			MinPrice:  100000,
			MaxPrice:  1000000,
			MinAmount: 1,
			MaxAmount: 10,
		},
	}

	err = e.LoadTolerances(newLimits)
	if !errors.Is(err, nil) {
		t.Fatalf("expected error %v but received %v", errCannotLoadTolerance, err)
	}

	_, err = e.GetTolerance(asset.Futures, ltcusd)
	if !errors.Is(err, errExchangeToleranceAsset) {
		t.Fatalf("expected error %v but received %v", errExchangeToleranceAsset, err)
	}

	_, err = e.GetTolerance(asset.Spot, ltcusd)
	if !errors.Is(err, errExchangeToleranceBase) {
		t.Fatalf("expected error %v but received %v", errExchangeToleranceBase, err)
	}

	_, err = e.GetTolerance(asset.Spot, btcltc)
	if !errors.Is(err, errExchangeToleranceQuote) {
		t.Fatalf("expected error %v but received %v", errExchangeToleranceQuote, err)
	}

	tt, err := e.GetTolerance(asset.Spot, btcusd)
	if !errors.Is(err, nil) {
		t.Fatalf("expected error %v but received %v", nil, err)
	}

	if tt.maxAmount != newLimits[0].MaxAmount ||
		tt.minAmount != newLimits[0].MinAmount ||
		tt.maxPrice != newLimits[0].MaxPrice ||
		tt.minPrice != newLimits[0].MinPrice {
		t.Fatal("unexpected values")
	}
}

func TestCheckTolerance(t *testing.T) {
	e := ExecutionTolerance{}
	err := e.CheckTolerance(asset.Spot, btcusd, 1337, 1337)
	if !errors.Is(err, nil) {
		t.Fatalf("expected error %v but received %v", nil, err)
	}

	newLimits := []MinMaxLevel{
		{
			Pair:      btcusd,
			Asset:     asset.Spot,
			MinPrice:  100000,
			MaxPrice:  1000000,
			MinAmount: 1,
			MaxAmount: 10,
		},
	}

	err = e.LoadTolerances(newLimits)
	if !errors.Is(err, nil) {
		t.Fatalf("expected error %v but received %v", errCannotLoadTolerance, err)
	}

	err = e.CheckTolerance(asset.Futures, ltcusd, 1337, 1337)
	if !errors.Is(err, errCannotValidateAsset) {
		t.Fatalf("expected error %v but received %v", errCannotValidateAsset, err)
	}

	err = e.CheckTolerance(asset.Spot, ltcusd, 1337, 1337)
	if !errors.Is(err, errCannotValidateBaseCurrency) {
		t.Fatalf("expected error %v but received %v", errCannotValidateBaseCurrency, err)
	}

	err = e.CheckTolerance(asset.Spot, btcltc, 1337, 1337)
	if !errors.Is(err, errCannotValidateQuoteCurrency) {
		t.Fatalf("expected error %v but received %v", errCannotValidateQuoteCurrency, err)
	}

	err = e.CheckTolerance(asset.Spot, btcusd, 1337, 1337)
	if !errors.Is(err, ErrPriceExceedsMin) {
		t.Fatalf("expected error %v but received %v", ErrPriceExceedsMin, err)
	}

	err = e.CheckTolerance(asset.Spot, btcusd, 1000001, 1337)
	if !errors.Is(err, ErrPriceExceedsMax) {
		t.Fatalf("expected error %v but received %v", ErrPriceExceedsMax, err)
	}

	err = e.CheckTolerance(asset.Spot, btcusd, 999999, .5)
	if !errors.Is(err, ErrAmountExceedsMin) {
		t.Fatalf("expected error %v but received %v", ErrAmountExceedsMin, err)
	}

	err = e.CheckTolerance(asset.Spot, btcusd, 999999, 11)
	if !errors.Is(err, ErrAmountExceedsMax) {
		t.Fatalf("expected error %v but received %v", ErrAmountExceedsMax, err)
	}

	err = e.CheckTolerance(asset.Spot, btcusd, 999999, 7)
	if !errors.Is(err, nil) {
		t.Fatalf("expected error %v but received %v", nil, err)
	}
}

func TestConforms(t *testing.T) {
	var tt *Tolerance
	err := tt.Conforms(0, 0)
	if err != nil {
		t.Fatal(err)
	}

	tt = &Tolerance{
		minNotional: 100,
	}

	err = tt.Conforms(1, 1)
	if !errors.Is(err, ErrNotionalValue) {
		t.Fatalf("expected error %v but received %v", ErrNotionalValue, err)
	}

	err = tt.Conforms(200, .5)
	if !errors.Is(err, nil) {
		t.Fatalf("expected error %v but received %v", nil, err)
	}

	tt.stepSizePrice = 0.001
	err = tt.Conforms(200.0001, .5)
	if !errors.Is(err, ErrPriceExceedsStep) {
		t.Fatalf("expected error %v but received %v", ErrPriceExceedsStep, err)
	}
	err = tt.Conforms(200.004, .5)
	if !errors.Is(err, nil) {
		t.Fatalf("expected error %v but received %v", nil, err)
	}

	tt.stepSizeAmount = 0.001
	err = tt.Conforms(200, .0002)
	if !errors.Is(err, ErrAmountExceedsStep) {
		t.Fatalf("expected error %v but received %v", ErrAmountExceedsStep, err)
	}
	err = tt.Conforms(200000, .003)
	if !errors.Is(err, nil) {
		t.Fatalf("expected error %v but received %v", nil, err)
	}
}

func TestConformToAmount(t *testing.T) {
	tt := &Tolerance{}
	val := tt.ConformToAmount(1)
	if val != 0 {
		t.Fatal("unexpected amount")
	}

	tt.stepSizeAmount = 0.001
	val = tt.ConformToAmount(1)
	if val != 1 {
		t.Fatal("unexpected amount")
	}

	val = tt.ConformToAmount(0.7777)
	if val != 0.777 {
		t.Fatal("unexpected amount", val)
	}

	tt.stepSizeAmount = 100
	val = tt.ConformToAmount(100)
	if val != 100 {
		t.Fatal("unexpected amount", val)
	}

	val = tt.ConformToAmount(200)
	if val != 200 {
		t.Fatal("unexpected amount", val)
	}
	val = tt.ConformToAmount(150)
	if val != 100 {
		t.Fatal("unexpected amount", val)
	}
}
