package validator

import (
	"context"
	"math/rand"
	"time"

	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deposit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

const (
	validatorOpen  float64 = 5000
	validatorHigh  float64 = 6000
	validatorLow   float64 = 5500
	validatorClose float64 = 5700
	validatorVol   float64 = 10
)

// Exchanges validator for test execution/scripts
func (w Wrapper) Exchanges(enabledOnly bool) []string {
	if enabledOnly {
		return []string{
			"hello world",
		}
	}
	return []string{
		"nope",
	}
}

// IsEnabled returns if requested exchange is enabled or disabled
func (w Wrapper) IsEnabled(exch string) (v bool) {
	if exch == exchError.String() {
		return
	}
	return true
}

// Orderbook validator for test execution/scripts
func (w Wrapper) Orderbook(_ context.Context, exch string, pair currency.Pair, item asset.Item) (*orderbook.Book, error) {
	if exch == exchError.String() {
		return nil, errTestFailed
	}

	return &orderbook.Book{
		Exchange: exch,
		Asset:    item,
		Pair:     pair,
		Bids: []orderbook.Level{
			{
				Amount: 1,
				Price:  1,
			},
		},
		Asks: []orderbook.Level{
			{
				Amount: 1,
				Price:  1,
			},
		},
	}, nil
}

// Ticker validator for test execution/scripts
func (w Wrapper) Ticker(_ context.Context, exch string, pair currency.Pair, item asset.Item) (*ticker.Price, error) {
	if exch == exchError.String() {
		return nil, errTestFailed
	}
	return &ticker.Price{
		Last:         1,
		High:         2,
		Low:          3,
		Bid:          4,
		Ask:          5,
		Volume:       6,
		QuoteVolume:  7,
		PriceATH:     8,
		Open:         9,
		Close:        10,
		Pair:         pair,
		ExchangeName: exch,
		AssetType:    item,
		LastUpdated:  time.Now(),
	}, nil
}

// Pairs validator for test execution/scripts
func (w Wrapper) Pairs(exch string, _ bool, _ asset.Item) (*currency.Pairs, error) {
	if exch == exchError.String() {
		return nil, errTestFailed
	}

	pairs, err := currency.NewPairsFromStrings([]string{
		"btc_usd",
		"btc_aud",
		"btc_ltc",
	})
	if err != nil {
		return nil, err
	}
	return &pairs, nil
}

// QueryOrder validator for test execution/scripts
func (w Wrapper) QueryOrder(_ context.Context, exch, _ string, _ currency.Pair, _ asset.Item) (*order.Detail, error) {
	if exch == exchError.String() {
		return nil, errTestFailed
	}

	return &order.Detail{
		Exchange:        exch,
		AccountID:       "hello",
		OrderID:         "1",
		Pair:            currency.NewBTCUSD(),
		Side:            order.Ask,
		Type:            order.Limit,
		Date:            time.Now(),
		Status:          order.Cancelled,
		Price:           1,
		Amount:          2,
		ExecutedAmount:  1,
		RemainingAmount: 0,
		Fee:             0,
		Trades: []order.TradeHistory{
			{
				TID:         "",
				Price:       1,
				Amount:      2,
				Exchange:    exch,
				Type:        order.Limit,
				Side:        order.Ask,
				Fee:         0,
				Description: "",
			},
		},
	}, nil
}

// SubmitOrder validator for test execution/scripts
func (w Wrapper) SubmitOrder(_ context.Context, o *order.Submit) (*order.SubmitResponse, error) {
	if o == nil {
		return nil, errTestFailed
	}
	if o.Exchange == exchError.String() {
		return nil, errTestFailed
	}

	resp, err := o.DeriveSubmitResponse(o.Exchange)
	if err != nil {
		return nil, err
	}

	resp.Status = order.Rejected
	if o.Exchange == "true" {
		resp.Status = order.New
	}

	return resp, nil
}

// CancelOrder validator for test execution/scripts
func (w Wrapper) CancelOrder(_ context.Context, exch, orderid string, cp currency.Pair, a asset.Item) (bool, error) {
	if exch == exchError.String() {
		return false, errTestFailed
	}
	if orderid == "" {
		return false, errTestFailed
	}
	if !cp.IsEmpty() && cp.IsInvalid() {
		return false, errTestFailed
	}
	if a != asset.Empty && !a.IsValid() {
		return false, errTestFailed
	}
	return true, nil
}

// AccountBalances validator for test execution/scripts
func (w Wrapper) AccountBalances(_ context.Context, exch string, assetType asset.Item) (accounts.SubAccounts, error) {
	if exch == exchError.String() {
		return nil, errTestFailed
	}
	c := currency.Code{
		Item: &currency.Item{
			ID:         0,
			FullName:   "Bitcoin",
			Symbol:     "BTC",
			Role:       1,
			AssocChain: "",
		},
	}
	return accounts.SubAccounts{
		{
			ID:        "subacct1",
			AssetType: assetType,
			Balances: accounts.CurrencyBalances{
				c: accounts.Balance{
					Currency: c,
					Total:    100,
					Hold:     0,
				},
			},
		},
	}, nil
}

// DepositAddress validator for test execution/scripts
func (w Wrapper) DepositAddress(exch, _ string, _ currency.Code) (*deposit.Address, error) {
	if exch == exchError.String() {
		return nil, errTestFailed
	}

	return &deposit.Address{Address: core.BitcoinDonationAddress}, nil
}

// WithdrawalCryptoFunds validator for test execution/scripts
func (w Wrapper) WithdrawalCryptoFunds(_ context.Context, r *withdraw.Request) (out string, err error) {
	if r.Exchange == exchError.String() {
		return r.Exchange, errTestFailed
	}

	return "", nil
}

// WithdrawalFiatFunds validator for test execution/scripts
func (w Wrapper) WithdrawalFiatFunds(_ context.Context, _ string, r *withdraw.Request) (out string, err error) {
	if r.Exchange == exchError.String() {
		return r.Exchange, errTestFailed
	}

	return "123", nil
}

// OHLCV returns open high low close volume candles for requested exchange/pair/asset/start & end time
func (w Wrapper) OHLCV(_ context.Context, exch string, p currency.Pair, a asset.Item, start, _ time.Time, i kline.Interval) (*kline.Item, error) {
	if exch == exchError.String() {
		return nil, errTestFailed
	}
	var candles []kline.Candle

	candles = append(candles, kline.Candle{
		Time:   start,
		Open:   validatorOpen,
		High:   validatorHigh,
		Low:    validatorLow,
		Close:  validatorClose,
		Volume: validatorVol,
	})

	for x := 1; x < 200; x++ {
		r := validatorLow + rand.Float64()*(validatorHigh-validatorLow) //nolint:gosec // no need to import crypo/rand
		candle := kline.Candle{
			Time:   candles[x-1].Time.Add(-i.Duration()),
			Open:   r,
			High:   r,
			Low:    r,
			Close:  r,
			Volume: r,
		}
		candles = append(candles, candle)
	}

	return &kline.Item{
		Exchange: exch,
		Pair:     p,
		Asset:    a,
		Interval: i,
		Candles:  candles,
	}, nil
}
