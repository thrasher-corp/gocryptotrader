package order

import (
	"errors"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var btcusd = currency.NewPair(currency.BTC, currency.USD)
var ltcusd = currency.NewPair(currency.LTC, currency.USD)
var btcltc = currency.NewPair(currency.BTC, currency.LTC)

func TestLoadLimits(t *testing.T) {
	t.Parallel()
	e := ExecutionLimits{}
	err := e.LoadLimits(nil)
	if !errors.Is(err, errCannotLoadLimit) {
		t.Fatalf("expected error %v but received %v", errCannotLoadLimit, err)
	}

	invalidAsset := []MinMaxLevel{
		{
			Pair:      btcusd,
			MinPrice:  100000,
			MaxPrice:  1000000,
			MinAmount: 1,
			MaxAmount: 10,
		},
	}
	err = e.LoadLimits(invalidAsset)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("expected error %v but received %v",
			asset.ErrNotSupported,
			err)
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

	err = e.LoadLimits(newLimits)
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

	err = e.LoadLimits(badLimit)
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

	err = e.LoadLimits(badLimit)
	if !errors.Is(err, errInvalidAmountLevels) {
		t.Fatalf("expected error %v but received %v", errInvalidPriceLevels, err)
	}

	goodLimit := []MinMaxLevel{
		{
			Pair:  btcusd,
			Asset: asset.Spot,
		},
	}

	err = e.LoadLimits(goodLimit)
	if !errors.Is(err, nil) {
		t.Fatalf("expected error %v but received %v", nil, err)
	}

	noCompare := []MinMaxLevel{
		{
			Pair:      btcusd,
			Asset:     asset.Spot,
			MinAmount: 10,
		},
	}

	err = e.LoadLimits(noCompare)
	if !errors.Is(err, nil) {
		t.Fatalf("expected error %v but received %v", nil, err)
	}

	noCompare = []MinMaxLevel{
		{
			Pair:     btcusd,
			Asset:    asset.Spot,
			MinPrice: 10,
		},
	}

	err = e.LoadLimits(noCompare)
	if !errors.Is(err, nil) {
		t.Fatalf("expected error %v but received %v", nil, err)
	}
}

func TestGetOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	e := ExecutionLimits{}
	_, err := e.GetOrderExecutionLimits(asset.Spot, btcusd)
	if !errors.Is(err, ErrExchangeLimitNotLoaded) {
		t.Fatalf("expected error %v but received %v", ErrExchangeLimitNotLoaded, err)
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

	err = e.LoadLimits(newLimits)
	if !errors.Is(err, nil) {
		t.Fatalf("expected error %v but received %v", errCannotLoadLimit, err)
	}

	_, err = e.GetOrderExecutionLimits(asset.Futures, ltcusd)
	if !errors.Is(err, errExchangeLimitAsset) {
		t.Fatalf("expected error %v but received %v", errExchangeLimitAsset, err)
	}

	_, err = e.GetOrderExecutionLimits(asset.Spot, ltcusd)
	if !errors.Is(err, errExchangeLimitBase) {
		t.Fatalf("expected error %v but received %v", errExchangeLimitBase, err)
	}

	_, err = e.GetOrderExecutionLimits(asset.Spot, btcltc)
	if !errors.Is(err, errExchangeLimitQuote) {
		t.Fatalf("expected error %v but received %v", errExchangeLimitQuote, err)
	}

	tt, err := e.GetOrderExecutionLimits(asset.Spot, btcusd)
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

func TestCheckLimit(t *testing.T) {
	t.Parallel()
	e := ExecutionLimits{}
	err := e.CheckOrderExecutionLimits(asset.Spot, btcusd, 1337, 1337, Limit)
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

	err = e.LoadLimits(newLimits)
	if !errors.Is(err, nil) {
		t.Fatalf("expected error %v but received %v", errCannotLoadLimit, err)
	}

	err = e.CheckOrderExecutionLimits(asset.Futures, ltcusd, 1337, 1337, Limit)
	if !errors.Is(err, errCannotValidateAsset) {
		t.Fatalf("expected error %v but received %v", errCannotValidateAsset, err)
	}

	err = e.CheckOrderExecutionLimits(asset.Spot, ltcusd, 1337, 1337, Limit)
	if !errors.Is(err, errCannotValidateBaseCurrency) {
		t.Fatalf("expected error %v but received %v", errCannotValidateBaseCurrency, err)
	}

	err = e.CheckOrderExecutionLimits(asset.Spot, btcltc, 1337, 1337, Limit)
	if !errors.Is(err, errCannotValidateQuoteCurrency) {
		t.Fatalf("expected error %v but received %v", errCannotValidateQuoteCurrency, err)
	}

	err = e.CheckOrderExecutionLimits(asset.Spot, btcusd, 1337, 9, Limit)
	if !errors.Is(err, ErrPriceBelowMin) {
		t.Fatalf("expected error %v but received %v", ErrPriceBelowMin, err)
	}

	err = e.CheckOrderExecutionLimits(asset.Spot, btcusd, 1000001, 9, Limit)
	if !errors.Is(err, ErrPriceExceedsMax) {
		t.Fatalf("expected error %v but received %v", ErrPriceExceedsMax, err)
	}

	err = e.CheckOrderExecutionLimits(asset.Spot, btcusd, 999999, .5, Limit)
	if !errors.Is(err, ErrAmountBelowMin) {
		t.Fatalf("expected error %v but received %v", ErrAmountBelowMin, err)
	}

	err = e.CheckOrderExecutionLimits(asset.Spot, btcusd, 999999, 11, Limit)
	if !errors.Is(err, ErrAmountExceedsMax) {
		t.Fatalf("expected error %v but received %v", ErrAmountExceedsMax, err)
	}

	err = e.CheckOrderExecutionLimits(asset.Spot, btcusd, 999999, 7, Limit)
	if !errors.Is(err, nil) {
		t.Fatalf("expected error %v but received %v", nil, err)
	}

	err = e.CheckOrderExecutionLimits(asset.Spot, btcusd, 999999, 7, Market)
	if !errors.Is(err, nil) {
		t.Fatalf("expected error %v but received %v", nil, err)
	}
}

func TestConforms(t *testing.T) {
	t.Parallel()
	var tt *Limits
	err := tt.Conforms(0, 0, Limit)
	if err != nil {
		t.Fatal(err)
	}

	tt = &Limits{
		minNotional: 100,
	}

	err = tt.Conforms(1, 1, Limit)
	if !errors.Is(err, ErrNotionalValue) {
		t.Fatalf("expected error %v but received %v", ErrNotionalValue, err)
	}

	err = tt.Conforms(200, .5, Limit)
	if !errors.Is(err, nil) {
		t.Fatalf("expected error %v but received %v", nil, err)
	}

	tt.stepIncrementSizePrice = 0.001
	err = tt.Conforms(200.0001, .5, Limit)
	if !errors.Is(err, ErrPriceExceedsStep) {
		t.Fatalf("expected error %v but received %v", ErrPriceExceedsStep, err)
	}
	err = tt.Conforms(200.004, .5, Limit)
	if !errors.Is(err, nil) {
		t.Fatalf("expected error %v but received %v", nil, err)
	}

	tt.stepIncrementSizeAmount = 0.001
	err = tt.Conforms(200, .0002, Limit)
	if !errors.Is(err, ErrAmountExceedsStep) {
		t.Fatalf("expected error %v but received %v", ErrAmountExceedsStep, err)
	}
	err = tt.Conforms(200000, .003, Limit)
	if !errors.Is(err, nil) {
		t.Fatalf("expected error %v but received %v", nil, err)
	}

	tt.minAmount = 1
	tt.maxAmount = 10
	tt.marketMinQty = 1.1
	tt.marketMaxQty = 9.9

	err = tt.Conforms(200000, 1, Market)
	if !errors.Is(err, ErrMarketAmountBelowMin) {
		t.Fatalf("expected error %v but received: %v", ErrMarketAmountBelowMin, err)
	}

	err = tt.Conforms(200000, 10, Market)
	if !errors.Is(err, ErrMarketAmountExceedsMax) {
		t.Fatalf("expected error %v but received: %v", ErrMarketAmountExceedsMax, err)
	}

	tt.marketStepIncrementSize = 10
	err = tt.Conforms(200000, 9.1, Market)
	if !errors.Is(err, ErrMarketAmountExceedsStep) {
		t.Fatalf("expected error %v but received: %v", ErrMarketAmountExceedsStep, err)
	}
	tt.marketStepIncrementSize = 1
	err = tt.Conforms(200000, 9.1, Market)
	if !errors.Is(err, nil) {
		t.Fatalf("expected error %v but received: %v", nil, err)
	}
}

func TestConformToAmount(t *testing.T) {
	t.Parallel()
	var tt *Limits
	if tt.ConformToAmount(1.001) != 1.001 {
		t.Fatal("value should not be changed")
	}

	tt = &Limits{}
	val := tt.ConformToAmount(1)
	if val != 1 { // If there is no step amount set this should not change
		// the inputted amount
		t.Fatal("unexpected amount")
	}

	tt.stepIncrementSizeAmount = 0.001
	val = tt.ConformToAmount(1.001)
	if val != 1.001 {
		t.Error("unexpected amount", val)
	}

	val = tt.ConformToAmount(0.0001)
	if val != 0 {
		t.Error("unexpected amount", val)
	}

	val = tt.ConformToAmount(0.7777)
	if val != 0.777 {
		t.Error("unexpected amount", val)
	}

	tt.stepIncrementSizeAmount = 100
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
