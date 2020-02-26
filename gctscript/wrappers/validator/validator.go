package validator

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
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
		ExchangeName: exch,
		AssetType:    item,
		Pair:         pair,
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

	pairs := currency.NewPairsFromStrings([]string{"btc_usd", "btc_aud", "btc_ltc"})
	return &pairs, nil
}

// QueryOrder validator for test execution/scripts
func (w Wrapper) QueryOrder(exch, _ string) (*order.Detail, error) {
	if exch == exchError.String() {
		return nil, errTestFailed
	}
	return &order.Detail{
		Exchange:        exch,
		AccountID:       "hello",
		ID:              "1",
		Pair:            currency.NewPairFromString("BTCAUD"),
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
				Timestamp:   time.Now(),
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
func (w Wrapper) CancelOrder(exch, orderid string) (bool, error) {
	if exch == exchError.String() {
		return false, errTestFailed
	}
	return orderid != "false", nil
}

// AccountInformation validator for test execution/scripts
func (w Wrapper) AccountInformation(exch string) (account.Holdings, error) {
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
								ID:            0,
								FullName:      "Bitcoin",
								Symbol:        "BTC",
								Role:          1,
								AssocChain:    "",
								AssocExchange: nil,
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
func (w Wrapper) WithdrawalCryptoFunds(exch string, _ *withdraw.Request) (out string, err error) {
	if exch == exchError.String() {
		return exch, errTestFailed
	}

	return "", nil
}

// WithdrawalFiatFunds validator for test execution/scripts
func (w Wrapper) WithdrawalFiatFunds(exch, _ string, _ *withdraw.Request) (out string, err error) {
	if exch == exchError.String() {
		return exch, errTestFailed
	}

	return "123", nil
}
