package twap

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	strategy "github.com/thrasher-corp/gocryptotrader/exchanges/strategy/common"
)

func TestConfig_Check(t *testing.T) {
	t.Parallel()

	var c *Config
	err := c.Check(context.Background())
	if !errors.Is(err, strategy.ErrConfigIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, strategy.ErrConfigIsNil)
	}

	c = &Config{}
	err = c.Check(context.Background())
	if !errors.Is(err, strategy.ErrExchangeIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, strategy.ErrExchangeIsNil)
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
	c.End = time.Now().Add(-time.Hour)
	err = c.Check(context.Background())
	if !errors.Is(err, strategy.ErrEndBeforeTimeNow) {
		t.Fatalf("received: '%v' but expected: '%v'", err, strategy.ErrEndBeforeTimeNow)
	}

	c.Start = time.Now().Add(-time.Hour * 2)
	c.End = time.Now().Add(-time.Hour)
	err = c.Check(context.Background())
	if !errors.Is(err, strategy.ErrEndBeforeTimeNow) {
		t.Fatalf("received: '%v' but expected: '%v'", err, strategy.ErrEndBeforeTimeNow)
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
	if !errors.Is(err, kline.ErrUnsetInterval) {
		t.Fatalf("received: '%v' but expected: '%v'", err, kline.ErrUnsetInterval)
	}

	c.TWAP = kline.OneHour
	err = c.Check(context.Background())
	if !errors.Is(err, strategy.ErrCannotSetAmount) {
		t.Fatalf("received: '%v' but expected: '%v'", err, strategy.ErrCannotSetAmount)
	}

	c.Amount = 0
	c.FullAmount = false
	err = c.Check(context.Background())
	if !errors.Is(err, strategy.ErrInvalidAmount) {
		t.Fatalf("received: '%v' but expected: '%v'", err, strategy.ErrInvalidAmount)
	}

	c.Amount = 100000
	c.MaxImpactSlippage = -1
	err = c.Check(context.Background())
	if !errors.Is(err, strategy.ErrInvalidSlippage) {
		t.Fatalf("received: '%v' but expected: '%v'", err, strategy.ErrInvalidSlippage)
	}

	c.MaxImpactSlippage = 0
	c.MaxNominalSlippage = -1
	err = c.Check(context.Background())
	if !errors.Is(err, strategy.ErrInvalidSlippage) {
		t.Fatalf("received: '%v' but expected: '%v'", err, strategy.ErrInvalidSlippage)
	}

	c.MaxNominalSlippage = 0
	c.PriceLimit = -1
	err = c.Check(context.Background())
	if !errors.Is(err, strategy.ErrInvalidPriceLimit) {
		t.Fatalf("received: '%v' but expected: '%v'", err, strategy.ErrInvalidPriceLimit)
	}

	c.PriceLimit = 0
	c.MaxSpreadPercentage = -1
	err = c.Check(context.Background())
	if !errors.Is(err, strategy.ErrInvalidSpread) {
		t.Fatalf("received: '%v' but expected: '%v'", err, strategy.ErrInvalidSpread)
	}

	c.MaxSpreadPercentage = 0
	err = c.Check(context.Background())
	if !errors.Is(err, strategy.ErrInvalidRetryAttempts) {
		t.Fatalf("received: '%v' but expected: '%v'", err, strategy.ErrInvalidRetryAttempts)
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
	_, err := c.GetDistrbutionAmount(context.Background(), 0, nil)
	if !errors.Is(err, strategy.ErrConfigIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, strategy.ErrConfigIsNil)
	}

	tn := time.Now()
	c = &Config{Start: tn, End: tn.Add(time.Minute)}
	_, err = c.GetDistrbutionAmount(context.Background(), 0, nil)
	if !errors.Is(err, strategy.ErrExchangeIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, strategy.ErrExchangeIsNil)
	}

	c.Exchange = &fake{}
	_, err = c.GetDistrbutionAmount(context.Background(), 0, nil)
	if !errors.Is(err, strategy.ErrInvalidAmount) {
		t.Fatalf("received: '%v' but expected: '%v'", err, strategy.ErrInvalidAmount)
	}

	_, err = c.GetDistrbutionAmount(context.Background(), 5, nil)
	if !errors.Is(err, strategy.ErrOrderbookIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, strategy.ErrOrderbookIsNil)
	}

	depth, err := orderbook.DeployDepth("test", currency.NewPair(currency.MANA, currency.CYC), asset.Spot)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	_, err = c.GetDistrbutionAmount(context.Background(), 5, depth)
	if !errors.Is(err, kline.ErrUnsetInterval) {
		t.Fatalf("received: '%v' but expected: '%v'", err, kline.ErrUnsetInterval)
	}

	c.Interval = kline.OneMin
	_, err = c.GetDistrbutionAmount(context.Background(), 5, depth)
	if !errors.Is(err, strategy.ErrInvalidOperatingWindow) {
		t.Fatalf("received: '%v' but expected: '%v'", err, strategy.ErrInvalidOperatingWindow)
	}

	c.End = tn.Add(time.Minute * 5)
	_, err = c.GetDistrbutionAmount(context.Background(), 5, depth)
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
	_, err = c.GetDistrbutionAmount(context.Background(), 0.01, depth)
	if !errors.Is(err, strategy.ErrUnderMinimumAmount) {
		t.Fatalf("received: '%v' but expected: '%v'", err, strategy.ErrUnderMinimumAmount)
	}

	_, err = c.GetDistrbutionAmount(context.Background(), 5000000, depth)
	if !errors.Is(err, strategy.ErrOverMaximumAmount) {
		t.Fatalf("received: '%v' but expected: '%v'", err, strategy.ErrOverMaximumAmount)
	}

	amount, err := c.GetDistrbutionAmount(context.Background(), 500000, depth)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if amount.Deployment != 100000 { // This will stick to the quote amount from above.
		t.Fatalf("received: '%v' but expected: '%v'", amount, 100000)
	}

	_, err = c.GetDistrbutionAmount(context.Background(), 100000, depth)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
}

func TestConfig_VerifyBookDeployment(t *testing.T) {
	t.Parallel()

	var c *Config
	_, _, err := c.VerifyBookDeployment(nil, 0, 0)
	if !errors.Is(err, strategy.ErrConfigIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, strategy.ErrConfigIsNil)
	}

	c = &Config{
		MaxImpactSlippage:   0.1,
		MaxNominalSlippage:  0.1,
		MaxSpreadPercentage: 0.1,
		PriceLimit:          75,
	}
	_, _, err = c.VerifyBookDeployment(nil, 0, 0)
	if !errors.Is(err, strategy.ErrOrderbookIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, strategy.ErrOrderbookIsNil)
	}

	depth, err := orderbook.DeployDepth("test", currency.NewPair(currency.MANA, currency.C2), asset.Spot)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	_, _, err = c.VerifyBookDeployment(depth, 0, 0)
	if !errors.Is(err, strategy.ErrInvalidAmount) {
		t.Fatalf("received: '%v' but expected: '%v'", err, strategy.ErrInvalidAmount)
	}

	depth.LoadSnapshot(
		[]orderbook.Item{{Amount: 1, Price: 50}, {Amount: 1, Price: 25}},
		[]orderbook.Item{{Amount: 10000000, Price: 100}},
		0,
		time.Time{},
		true,
	)

	_, _, err = c.VerifyBookDeployment(depth, 2, 0)
	if !errors.Is(err, strategy.ErrExceedsLiquidity) {
		t.Fatalf("received: '%v' but expected: '%v'", err, strategy.ErrExceedsLiquidity)
	}

	_, _, err = c.VerifyBookDeployment(depth, 1.5, 0)
	if !errors.Is(err, strategy.ErrMaxNominalExceeded) {
		t.Fatalf("received: '%v' but expected: '%v'", err, strategy.ErrMaxNominalExceeded)
	}

	c.MaxImpactSlippage = 0
	_, _, err = c.VerifyBookDeployment(depth, 1.5, 0)
	if !errors.Is(err, strategy.ErrMaxNominalExceeded) {
		t.Fatalf("received: '%v' but expected: '%v'", err, strategy.ErrMaxNominalExceeded)
	}

	c.MaxNominalSlippage = 0
	_, _, err = c.VerifyBookDeployment(depth, 1.5, 0)
	if !errors.Is(err, strategy.ErrPriceLimitExceeded) {
		t.Fatalf("received: '%v' but expected: '%v'", err, strategy.ErrPriceLimitExceeded)
	}

	c.Buy = true
	_, _, err = c.VerifyBookDeployment(depth, 1.5, 0)
	if !errors.Is(err, strategy.ErrPriceLimitExceeded) {
		t.Fatalf("received: '%v' but expected: '%v'", err, strategy.ErrPriceLimitExceeded)
	}

	c.Buy = false
	c.PriceLimit = 0
	_, _, err = c.VerifyBookDeployment(depth, 1.5, 0)
	if !errors.Is(err, strategy.ErrMaxSpreadExceeded) {
		t.Fatalf("received: '%v' but expected: '%v'", err, strategy.ErrMaxSpreadExceeded)
	}

	c.MaxSpreadPercentage = 0
	_, _, err = c.VerifyBookDeployment(depth, 1.5, 0)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
}

func TestConfig_VerifyExecutionLimits(t *testing.T) {
	t.Parallel()

	var c *Config
	_, err := c.VerifyExecutionLimitsReturnConformed(0)
	if !errors.Is(err, strategy.ErrConfigIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, strategy.ErrConfigIsNil)
	}

	c = &Config{Exchange: &fake{}}
	_, err = c.VerifyExecutionLimitsReturnConformed(0)
	if !errors.Is(err, strategy.ErrInvalidAmount) {
		t.Fatalf("received: '%v' but expected: '%v'", err, strategy.ErrInvalidAmount)
	}

	_, err = c.VerifyExecutionLimitsReturnConformed(0.00001)
	if !errors.Is(err, strategy.ErrUnderMinimumAmount) {
		t.Fatalf("received: '%v' but expected: '%v'", err, strategy.ErrUnderMinimumAmount)
	}

	_, err = c.VerifyExecutionLimitsReturnConformed(1000000)
	if !errors.Is(err, strategy.ErrOverMaximumAmount) {
		t.Fatalf("received: '%v' but expected: '%v'", err, strategy.ErrOverMaximumAmount)
	}

	conformed, err := c.VerifyExecutionLimitsReturnConformed(1)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if conformed != 1 {
		t.Fatalf("received: '%v' but expected: '%v'", conformed, 1)
	}
}
