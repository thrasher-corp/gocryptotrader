package validator

import (
	"math/rand"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
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
func (w Wrapper) Orderbook(exch string, pair currency.Pair, item asset.Item) (*orderbook.Base, error) {
	if exch == exchError.String() {
		return nil, errTestFailed
	}

	return &orderbook.Base{
		Exchange: exch,
		Asset:    item,
		Pair:     pair,
		Bids: []orderbook.Item{
			{
				Amount: 1,
				Price:  1,
			},
		},
		Asks: []orderbook.Item{
			{
				Amount: 1,
				Price:  1,
			},
		},
	}, nil
}

// Ticker validator for test execution/scripts
func (w Wrapper) Ticker(exch string, pair currency.Pair, item asset.Item) (*ticker.Price, error) {
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

	pairs, err := currency.NewPairsFromStrings([]string{"btc_usd",
		"btc_aud",
		"btc_ltc"})
	if err != nil {
		return nil, err
	}
	return &pairs, nil
}

// QueryOrder validator for test execution/scripts
func (w Wrapper) QueryOrder(exch, _ string, _ currency.Pair, _ asset.Item) (*order.Detail, error) {
	if exch == exchError.String() {
		return nil, errTestFailed
	}

	pair, err := currency.NewPairFromString("BTCAUD")
	if err != nil {
		return nil, err
	}

	return &order.Detail{
		Exchange:        exch,
		AccountID:       "hello",
		ID:              "1",
		Pair:            pair,
		Side:            "ask",
		Type:            "limit",
		Date:            time.Now(),
		Status:          "cancelled",
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
				Type:        "limit",
				Side:        "ask",
				Fee:         0,
				Description: "",
			},
		},
	}, nil
}

// SubmitOrder validator for test execution/scripts
func (w Wrapper) SubmitOrder(o *order.Submit) (*order.SubmitResponse, error) {
	if o == nil {
		return nil, errTestFailed
	}
	if o.Exchange == exchError.String() {
		return nil, errTestFailed
	}

	tempOrder := &order.SubmitResponse{
		IsOrderPlaced: false,
		OrderID:       o.Exchange,
	}

	if o.Exchange == "true" {
		tempOrder.IsOrderPlaced = true
	}

	return tempOrder, nil
}

// CancelOrder validator for test execution/scripts
func (w Wrapper) CancelOrder(exch, orderid string, cp currency.Pair, a asset.Item) (bool, error) {
	if exch == exchError.String() {
		return false, errTestFailed
	}
	if orderid == "" {
		return false, errTestFailed
	}
	if !cp.IsEmpty() && cp.IsInvalid() {
		return false, errTestFailed
	}
	if a != "" && !a.IsValid() {
		return false, errTestFailed
	}
	return true, nil
}

// AccountInformation validator for test execution/scripts
func (w Wrapper) AccountInformation(exch string, assetType asset.Item) (account.Holdings, error) {
	if exch == exchError.String() {
		return account.Holdings{}, errTestFailed
	}

	return account.Holdings{
		Exchange: exch,
		Accounts: []account.SubAccount{
			{
				ID: exch,
				Currencies: []account.Balance{
					{
						CurrencyName: currency.Code{
							Item: &currency.Item{
								ID:         0,
								FullName:   "Bitcoin",
								Symbol:     "BTC",
								Role:       1,
								AssocChain: "",
							},
						},
						TotalValue: 100,
						Hold:       0,
					},
				},
			},
		},
	}, nil
}

// DepositAddress validator for test execution/scripts
func (w Wrapper) DepositAddress(exch string, _ currency.Code) (string, error) {
	if exch == exchError.String() {
		return exch, errTestFailed
	}

	return exch, nil
}

// WithdrawalCryptoFunds validator for test execution/scripts
func (w Wrapper) WithdrawalCryptoFunds(r *withdraw.Request) (out string, err error) {
	if r.Exchange == exchError.String() {
		return r.Exchange, errTestFailed
	}

	return "", nil
}

// WithdrawalFiatFunds validator for test execution/scripts
func (w Wrapper) WithdrawalFiatFunds(_ string, r *withdraw.Request) (out string, err error) {
	if r.Exchange == exchError.String() {
		return r.Exchange, errTestFailed
	}

	return "123", nil
}

// OHLCV returns open high low close volume candles for requested exchange/pair/asset/start & end time
func (w Wrapper) OHLCV(exch string, p currency.Pair, a asset.Item, start, end time.Time, i kline.Interval) (kline.Item, error) {
	if exch == exchError.String() {
		return kline.Item{}, errTestFailed
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
		r := validatorLow + rand.Float64()*(validatorHigh-validatorLow) // nolint:gosec // no need to import crypo/rand
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

	return kline.Item{
		Exchange: exch,
		Pair:     p,
		Asset:    a,
		Interval: i,
		Candles:  candles,
	}, nil
}
