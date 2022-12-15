package dca

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	strategy "github.com/thrasher-corp/gocryptotrader/exchanges/strategy/common"
)

var btcusd = currency.NewPair(currency.BTC, currency.USD)
var errTestCredsFail = errors.New("fail on creds")

type fake struct {
	fields exchange.Base
	exchange.IBotExchange
}

func (f *fake) GetCredentials(ctx context.Context) (*account.Credentials, error) {
	creds, err := f.fields.GetCredentials(ctx)
	if err != nil {
		return nil, err
	}
	if creds.Key == "FAIL" {
		return nil, errTestCredsFail
	}
	return creds, nil
}

func (f *fake) GetName() string {
	return "fake"
}

func (f *fake) GetOrderExecutionLimits(asset.Item, currency.Pair) (order.MinMaxLevel, error) {
	return order.MinMaxLevel{MinAmount: 0.0001, MaxAmount: 1000}, nil
}

func (f *fake) SubmitOrder(_ context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	return s.DeriveSubmitResponse(strategy.Simulation)
}

func loadHoldingsState(pair currency.Pair, freeQuote, freeBase float64) error {
	if pair.IsEmpty() {
		return errors.New("pair is empty")
	}
	return account.Process(
		&account.Holdings{
			Exchange: "fake",
			Accounts: []account.SubAccount{
				{
					AssetType: asset.Spot,
					Currencies: []account.Balance{
						{
							CurrencyName: pair.Quote,
							Free:         freeQuote,
						},
						{
							CurrencyName: pair.Base,
							Free:         freeBase,
							// TODO: Upgrade to allow for no balance loaded.
						},
					},
				},
			},
		},
		&account.Credentials{Key: "KEY"},
	)
}

func TestNew(t *testing.T) {
	t.Parallel()

	_, err := New(context.Background(), nil)
	if !errors.Is(err, strategy.ErrConfigIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, strategy.ErrConfigIsNil)
	}

	tn := time.Now()

	c := &Config{
		Exchange:      &fake{},
		Pair:          currency.NewPair(currency.AAA, currency.WABI),
		Asset:         asset.Futures,
		Interval:      kline.OneMin,
		Start:         tn,
		End:           tn.Add(time.Minute * 5),
		Amount:        100001, // Quotation funding (USD)
		Buy:           true,
		RetryAttempts: 3,
	}

	_, err = New(context.Background(), c)
	if !errors.Is(err, strategy.ErrInvalidAssetType) {
		t.Fatalf("received: '%v' but expected: '%v'", err, strategy.ErrInvalidAssetType)
	}

	c.Asset = asset.Spot

	_, err = New(context.Background(), c)
	if !errors.Is(err, orderbook.ErrCannotFindOrderbook) {
		t.Fatalf("received: '%v' but expected: '%v'", err, orderbook.ErrCannotFindOrderbook)
	}

	c.Pair = btcusd
	depth, err := orderbook.DeployDepth("fake", btcusd, asset.Spot)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	failCtx := account.DeployCredentialsToContext(context.Background(), &account.Credentials{Key: "FAIL"})
	_, err = New(failCtx, c)
	if !errors.Is(err, errTestCredsFail) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errTestCredsFail)
	}

	err = loadHoldingsState(btcusd, 0, 0)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	ctx := account.DeployCredentialsToContext(context.Background(), &account.Credentials{Key: "KEY"})
	_, err = New(ctx, c)
	if !errors.Is(err, strategy.ErrNoBalance) {
		t.Fatalf("received: '%v' but expected: '%v'", err, strategy.ErrNoBalance)
	}

	err = loadHoldingsState(btcusd, 500, 0)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	_, err = New(ctx, c)
	if !errors.Is(err, strategy.ErrExceedsFreeBalance) {
		t.Fatalf("received: '%v' but expected: '%v'", err, strategy.ErrExceedsFreeBalance)
	}

	c.FullAmount = true
	c.Amount = 0
	depth.LoadSnapshot(
		[]orderbook.Item{{Amount: 10000000, Price: 99}},
		[]orderbook.Item{{Amount: 10000000, Price: 100}},
		0,
		time.Time{},
		true)

	err = loadHoldingsState(btcusd, 50000, 0)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	c.Buy = true
	_, err = New(ctx, c)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	err = loadHoldingsState(btcusd, 0, 1)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	c.Buy = false // Sell
	_, err = New(ctx, c)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
}

func TestStrategy_CheckAndSubmit(t *testing.T) {
	t.Parallel()

	var s *Strategy
	err := s.checkAndSubmit(context.Background())
	if !errors.Is(err, strategy.ErrIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, strategy.ErrIsNil)
	}

	pair := currency.NewPair(currency.B20, currency.F16)

	depth, err := orderbook.DeployDepth("fake", pair, asset.Spot)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	depth.LoadSnapshot(
		[]orderbook.Item{{Amount: 10000000, Price: 99}},
		[]orderbook.Item{{Amount: 10000000, Price: 100}},
		0,
		time.Time{},
		true,
	)

	err = loadHoldingsState(pair, 500, 500)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	ctx := account.DeployCredentialsToContext(context.Background(), &account.Credentials{Key: "KEY"})

	strate, err := New(ctx, &Config{
		Exchange:      &fake{},
		Pair:          pair,
		Asset:         asset.Spot,
		Interval:      kline.OneMin,
		Start:         time.Now(),
		End:           time.Now().Add(time.Minute * 5),
		Amount:        1,
		Buy:           true,
		Simulate:      true,
		RetryAttempts: 3,
	})
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	var ok bool
	s, ok = strate.(*Strategy)
	if !ok {
		t.Fatal("type assertion failed")
	}
	s.allocation.Deployment = 0
	err = s.checkAndSubmit(context.Background())
	if !errors.Is(err, strategy.ErrInvalidAmount) {
		t.Fatalf("received: '%v' but expected: '%v'", err, strategy.ErrInvalidAmount)
	}

	s.allocation.Deployment = .2
	err = s.checkAndSubmit(context.Background())
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
}
