package twap

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

func TestConfig_Check(t *testing.T) {
	t.Parallel()

	var c *Config
	err := c.Check(context.Background())
	if !errors.Is(err, errParamsAreNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errParamsAreNil)
	}

	c = &Config{}
	err = c.Check(context.Background())
	if !errors.Is(err, errExchangeIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errExchangeIsNil)
	}

	c.Exchange = &fake{}
	err = c.Check(context.Background())
	if !errors.Is(err, currency.ErrPairIsEmpty) {
		t.Fatalf("received: '%v' but expected: '%v'", err, currency.ErrPairIsEmpty)
	}

	c.Pair = btcusd
	err = c.Check(context.Background())
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received: '%v' but expected: '%v'", err, asset.ErrNotSupported)
	}

	c.Asset = asset.Spot
	err = c.Check(context.Background())
	if !errors.Is(err, common.ErrDateUnset) {
		t.Fatalf("received: '%v' but expected: '%v'", err, common.ErrDateUnset)
	}

	c.Start = time.Now().Add(-time.Hour * 2)
	c.End = time.Now().Add(-time.Hour)
	err = c.Check(context.Background())
	if !errors.Is(err, errEndBeforeNow) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errEndBeforeNow)
	}

	c.Start = time.Now()
	c.End = c.Start.AddDate(0, 0, 7)
	err = c.Check(context.Background())
	if !errors.Is(err, kline.ErrUnsetInterval) {
		t.Fatalf("received: '%v' but expected: '%v'", err, kline.ErrUnsetInterval)
	}

	c.Interval = kline.OneDay
	c.FullAmount = true
	c.Amount = 1
	err = c.Check(context.Background())
	if !errors.Is(err, errCannotSetAmount) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errCannotSetAmount)
	}

	c.Amount = 0
	c.FullAmount = false
	err = c.Check(context.Background())
	if !errors.Is(err, errInvalidVolume) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidVolume)
	}

	c.Amount = 100000
	c.MaxImpactSlippage = -1
	err = c.Check(context.Background())
	if !errors.Is(err, errInvalidMaxSlippageValue) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidMaxSlippageValue)
	}

	c.MaxImpactSlippage = 0
	c.MaxNominalSlippage = -1
	err = c.Check(context.Background())
	if !errors.Is(err, errInvalidMaxSlippageValue) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidMaxSlippageValue)
	}

	c.MaxNominalSlippage = 0
	c.PriceLimit = -1
	err = c.Check(context.Background())
	if !errors.Is(err, errInvalidPriceLimit) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidPriceLimit)
	}

	c.PriceLimit = 0
	c.MaxSpreadPercentage = -1
	err = c.Check(context.Background())
	if !errors.Is(err, errInvalidMaxSpreadPercentage) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidMaxSpreadPercentage)
	}

	c.MaxSpreadPercentage = 0
	err = c.Check(context.Background())
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
}

func TestConfig_GetDistrbutionAmount(t *testing.T) {
	t.Parallel()

	var c *Config
	_, err := c.GetDistrbutionAmount(0, nil, false)
	if !errors.Is(err, errConfigurationIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errConfigurationIsNil)
	}

	_, err = c.GetDistrbutionAmount(0, nil, false)
	if !errors.Is(err, errInvalidAllocatedAmount) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidAllocatedAmount)
	}

	tn := time.Now()
	c = &Config{Start: tn, End: tn.Add(time.Minute)}
	_, err = c.GetDistrbutionAmount(5, nil, false)
	if !errors.Is(err, kline.ErrUnsetInterval) {
		t.Fatalf("received: '%v' but expected: '%v'", err, kline.ErrUnsetInterval)
	}

	c.Interval = kline.OneMin
	_, err = c.GetDistrbutionAmount(5, nil, false)
	if !errors.Is(err, errInvalidOperationWindow) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidOperationWindow)
	}

	c.End = tn.Add(time.Minute * 5)
	_, err = c.GetDistrbutionAmount(5, nil, false)
	if !errors.Is(err, errInvalidOperationWindow) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidOperationWindow)
	}
}
