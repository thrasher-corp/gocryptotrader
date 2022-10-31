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
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
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
	if !errors.Is(err, errInvalidRetryAttempts) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidRetryAttempts)
	}

	c.RetryAttempts = 3
	err = c.Check(context.Background())
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
}

func TestConfig_GetDistrbutionAmount(t *testing.T) {
	t.Parallel()

	var c *Config
	_, err := c.GetDistrbutionAmount(0, nil)
	if !errors.Is(err, errConfigurationIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errConfigurationIsNil)
	}

	tn := time.Now()
	c = &Config{Start: tn, End: tn.Add(time.Minute)}
	_, err = c.GetDistrbutionAmount(0, nil)
	if !errors.Is(err, errExchangeIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errExchangeIsNil)
	}

	c.Exchange = &fake{}
	_, err = c.GetDistrbutionAmount(0, nil)
	if !errors.Is(err, errInvalidAllocatedAmount) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidAllocatedAmount)
	}

	_, err = c.GetDistrbutionAmount(5, nil)
	if !errors.Is(err, errOrderbookIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errOrderbookIsNil)
	}

	depth, err := orderbook.DeployDepth("test", currency.NewPair(currency.MANA, currency.CYC), asset.Spot)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	_, err = c.GetDistrbutionAmount(5, depth)
	if !errors.Is(err, kline.ErrUnsetInterval) {
		t.Fatalf("received: '%v' but expected: '%v'", err, kline.ErrUnsetInterval)
	}

	c.Interval = kline.OneMin
	_, err = c.GetDistrbutionAmount(5, depth)
	if !errors.Is(err, errInvalidOperationWindow) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidOperationWindow)
	}

	c.End = tn.Add(time.Minute * 5)
	_, err = c.GetDistrbutionAmount(5, depth)
	if !errors.Is(err, orderbook.ErrNoLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, orderbook.ErrNoLiquidity)
	}

	depth.LoadSnapshot(
		nil,
		[]orderbook.Item{{Amount: 10000000, Price: 100}},
		0,
		time.Time{},
		true,
	)

	c.Buy = true
	_, err = c.GetDistrbutionAmount(0.01, depth)
	if !errors.Is(err, errUnderMinimumAmount) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errUnderMinimumAmount)
	}

	_, err = c.GetDistrbutionAmount(5000000, depth)
	if !errors.Is(err, errOverMaximumAmount) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errOverMaximumAmount)
	}

	amount, err := c.GetDistrbutionAmount(500000, depth)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if amount != 100000 { // This will stick to the quote amount from above.
		t.Fatalf("received: '%v' but expected: '%v'", amount, 100000)
	}

	_, err = c.GetDistrbutionAmount(100000, depth)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
}

func TestConfig_VerifyBookDeployment(t *testing.T) {
	t.Parallel()

	var c *Config
	_, _, err := c.VerifyBookDeployment(nil, 0)
	if !errors.Is(err, errConfigurationIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errConfigurationIsNil)
	}

	c = &Config{
		MaxImpactSlippage:   0.1,
		MaxNominalSlippage:  0.1,
		MaxSpreadPercentage: 0.1,
		PriceLimit:          75,
	}
	_, _, err = c.VerifyBookDeployment(nil, 0)
	if !errors.Is(err, errOrderbookIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errOrderbookIsNil)
	}

	depth, err := orderbook.DeployDepth("test", currency.NewPair(currency.MANA, currency.C2), asset.Spot)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	_, _, err = c.VerifyBookDeployment(depth, 0)
	if !errors.Is(err, errInvalidAllocatedAmount) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidAllocatedAmount)
	}

	depth.LoadSnapshot(
		[]orderbook.Item{{Amount: 1, Price: 50}, {Amount: 1, Price: 25}},
		[]orderbook.Item{{Amount: 10000000, Price: 100}},
		0,
		time.Time{},
		true,
	)

	_, _, err = c.VerifyBookDeployment(depth, 2)
	if !errors.Is(err, errExceedsTotalBookLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errExceedsTotalBookLiquidity)
	}

	_, _, err = c.VerifyBookDeployment(depth, 1.5)
	if !errors.Is(err, errMaxImpactPercentageExceeded) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errMaxImpactPercentageExceeded)
	}

	c.MaxImpactSlippage = 0
	_, _, err = c.VerifyBookDeployment(depth, 1.5)
	if !errors.Is(err, errMaxNominalPercentageExceeded) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errMaxNominalPercentageExceeded)
	}

	c.MaxNominalSlippage = 0
	_, _, err = c.VerifyBookDeployment(depth, 1.5)
	if !errors.Is(err, errMaxPriceLimitExceeded) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errMaxPriceLimitExceeded)
	}

	c.Buy = true
	_, _, err = c.VerifyBookDeployment(depth, 1.5)
	if !errors.Is(err, errMaxPriceLimitExceeded) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errMaxPriceLimitExceeded)
	}

	c.Buy = false
	c.PriceLimit = 0
	_, _, err = c.VerifyBookDeployment(depth, 1.5)
	if !errors.Is(err, errMaxSpreadPercentageExceeded) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errMaxSpreadPercentageExceeded)
	}

	c.MaxSpreadPercentage = 0
	amount, _, err := c.VerifyBookDeployment(depth, 2.5)
	if !errors.Is(err, errBookSmallerThanDeploymentAmount) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errBookSmallerThanDeploymentAmount)
	}

	if amount != 2 {
		t.Fatalf("received: '%v' but expected: '%v'", amount, 2)
	}

	_, _, err = c.VerifyBookDeployment(depth, 2)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
}

func TestConfig_VerifyExecutionLimits(t *testing.T) {
	t.Parallel()

	var c *Config
	_, err := c.VerifyExecutionLimitsReturnConformed(0)
	if !errors.Is(err, errConfigurationIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errConfigurationIsNil)
	}

	c = &Config{Exchange: &fake{}}
	_, err = c.VerifyExecutionLimitsReturnConformed(0)
	if !errors.Is(err, errInvalidAllocatedAmount) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidAllocatedAmount)
	}

	_, err = c.VerifyExecutionLimitsReturnConformed(0.00001)
	if !errors.Is(err, errUnderMinimumAmount) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errUnderMinimumAmount)
	}

	_, err = c.VerifyExecutionLimitsReturnConformed(1000000)
	if !errors.Is(err, errOverMaximumAmount) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errOverMaximumAmount)
	}

	conformed, err := c.VerifyExecutionLimitsReturnConformed(1)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if conformed != 1 {
		t.Fatalf("received: '%v' but expected: '%v'", conformed, 1)
	}
}

func TestConfig_GetNextSchedule(t *testing.T) {
	t.Parallel()

	var c *Config
	_, err := c.GetNextSchedule(time.Now())
	if !errors.Is(err, errConfigurationIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errConfigurationIsNil)
	}

	c = &Config{}
	_, err = c.GetNextSchedule(time.Now())
	if !errors.Is(err, kline.ErrUnsetInterval) {
		t.Fatalf("received: '%v' but expected: '%v'", err, kline.ErrUnsetInterval)
	}

	tn := time.Now()
	c.Interval = kline.Interval(time.Minute)
	dur, err := c.GetNextSchedule(tn)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if dur != time.Minute {
		t.Fatalf("received: '%v' but expected: '%v'", dur, time.Minute)
	}

	c.CandleStickAligned = true
	dur, err = c.GetNextSchedule(tn)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if dur > time.Minute {
		t.Fatalf("received: '%v' but expected to be within or equal to '%v'", dur, time.Minute)
	}
}

func TestConfig_SetTimer(t *testing.T) {
	t.Parallel()

	var c *Config
	err := c.SetTimer(nil)
	if !errors.Is(err, errConfigurationIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errConfigurationIsNil)
	}

	c = &Config{Interval: kline.Interval(time.Minute)}
	err = c.SetTimer(nil)
	if !errors.Is(err, errTimerIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errTimerIsNil)
	}

	timer := time.NewTimer(0)
	err = c.SetTimer(timer)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if !timer.Stop() {
		<-timer.C
	}
}
