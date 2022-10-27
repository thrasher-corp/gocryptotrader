package twap

import (
	"context"
	"errors"
	"log"
	"os"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ftx"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
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

func TestMain(m *testing.M) {
	_, err := orderbook.DeployDepth("fake", btcusd, asset.Spot)
	if err != nil {
		log.Fatal(err)
	}
	os.Exit(m.Run())
}

func loadHoldingsState(freeQuote, freeBase float64) error {
	return account.Process(
		&account.Holdings{
			Exchange: "fake",
			Accounts: []account.SubAccount{
				{
					AssetType: asset.Spot,
					Currencies: []account.Balance{
						{
							CurrencyName:           currency.USD,
							AvailableWithoutBorrow: freeQuote,
						},
						{
							CurrencyName:           currency.BTC,
							AvailableWithoutBorrow: freeBase,
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
	if !errors.Is(err, errParamsAreNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errParamsAreNil)
	}

	tn := time.Now()

	c := &Config{
		Exchange: &fake{},
		Pair:     currency.NewPair(currency.AAA, currency.WABI),
		Asset:    asset.Futures,
		Interval: kline.OneMin,
		Start:    tn,
		End:      tn.Add(time.Minute * 5),
		Amount:   100001, // Quotation funding (USD)
		Buy:      true,
	}

	_, err = New(context.Background(), c)
	if !errors.Is(err, errInvalidAssetType) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidAssetType)
	}

	c.Asset = asset.Spot

	_, err = New(context.Background(), c)
	if !errors.Is(err, orderbook.ErrCannotFindOrderbook) {
		t.Fatalf("received: '%v' but expected: '%v'", err, orderbook.ErrCannotFindOrderbook)
	}

	c.Pair = btcusd

	failCtx := account.DeployCredentialsToContext(context.Background(), &account.Credentials{Key: "FAIL"})
	_, err = New(failCtx, c)
	if !errors.Is(err, errTestCredsFail) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errTestCredsFail)
	}

	err = loadHoldingsState(0, 0)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	ctx := account.DeployCredentialsToContext(context.Background(), &account.Credentials{Key: "KEY"})
	_, err = New(ctx, c)
	if !errors.Is(err, errNoBalanceFound) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoBalanceFound)
	}

	err = loadHoldingsState(500, 0)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	_, err = New(ctx, c)
	if !errors.Is(err, errExceedsFreeBalance) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errExceedsFreeBalance)
	}

	c.FullAmount = true
	c.Amount = 0
	_, err = New(ctx, c)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	err = loadHoldingsState(0, 1)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	c.Buy = false // Sell
	_, err = New(ctx, c)
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

	ctx := account.DeployCredentialsToContext(context.Background(),
		&account.Credentials{Key: "smelly old man"})

	twap, err := New(ctx, &Config{
		Exchange:                &ftx.FTX{},
		Pair:                    currency.NewPair(currency.BTC, currency.USD),
		Asset:                   asset.Spot,
		Start:                   time.Now(),
		End:                     time.Now().AddDate(0, 0, 7),
		Interval:                kline.OneDay,
		Amount:                  100000,
		Buy:                     true,
		AllowTradingPastEndTime: true,
	})
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if twap == nil {
		t.Fatal("unexpected value")
	}
}
