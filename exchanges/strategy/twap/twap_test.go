package twap

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ftx"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

func TestCheck(t *testing.T) {
	t.Parallel()

	var p *Config
	err := p.Check(context.Background())
	if !errors.Is(err, errParamsAreNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errParamsAreNil)
	}

	p = &Config{}
	err = p.Check(context.Background())
	if !errors.Is(err, currency.ErrPairIsEmpty) {
		t.Fatalf("received: '%v' but expected: '%v'", err, currency.ErrPairIsEmpty)
	}

	p.Pair = currency.NewPair(currency.BTC, currency.USD)
	err = p.Check(context.Background())
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received: '%v' but expected: '%v'", err, asset.ErrNotSupported)
	}

	p.Asset = asset.Spot
	err = p.Check(context.Background())
	if !errors.Is(err, common.ErrDateUnset) {
		t.Fatalf("received: '%v' but expected: '%v'", err, common.ErrDateUnset)
	}

	p.Start = time.Now()
	p.End = p.Start.AddDate(0, 0, 7)
	err = p.Check(context.Background())
	if !errors.Is(err, kline.ErrUnsetInterval) {
		t.Fatalf("received: '%v' but expected: '%v'", err, kline.ErrUnsetInterval)
	}

	p.Interval = kline.OneDay
	err = p.Check(context.Background())
	if !errors.Is(err, errInvalidVolume) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidVolume)
	}

	p.Volume = 100000
	p.MaxSlippage = -1
	err = p.Check(context.Background())
	if !errors.Is(err, errInvalidMaxSlippageValue) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidMaxSlippageValue)
	}

	p.MaxSlippage = 0
	err = p.Check(context.Background())
	if !errors.Is(err, errExchangeIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errExchangeIsNil)
	}

	p.Exchange = &ftx.FTX{}
	err = p.Check(context.Background())
	if !errors.Is(err, exchange.ErrCredentialsAreEmpty) {
		t.Fatalf("received: '%v' but expected: '%v'", err, exchange.ErrCredentialsAreEmpty)
	}

	p.Exchange.GetBase().API.SetKey("sweet cheeks")
	err = p.Check(context.Background())
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
}

func TestGetTWAP(t *testing.T) {
	t.Parallel()
	_, err := New(context.Background(), nil)
	if !errors.Is(err, errParamsAreNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errParamsAreNil)
	}

	ctx := exchange.DeployCredentialsToContext(context.Background(),
		&exchange.Credentials{Key: "smelly old man"})

	twap, err := New(ctx, &Config{
		Exchange:                &ftx.FTX{},
		Pair:                    currency.NewPair(currency.BTC, currency.USD),
		Asset:                   asset.Spot,
		Start:                   time.Now(),
		End:                     time.Now().AddDate(0, 0, 7),
		Interval:                kline.OneDay,
		Volume:                  100000,
		Accumulation:            true,
		AllowTradingPastEndTime: true,
	})
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if twap == nil {
		t.Fatal("unexpected value")
	}
}
