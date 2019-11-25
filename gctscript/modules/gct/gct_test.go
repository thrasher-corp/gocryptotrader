package gct

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/d5/tengo/objects"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/gctscript/modules"
)

var (
	exch = &objects.String{
		Value: "BTC Markets",
	}
	exchError = &objects.String{
		Value: "error",
	}
	currencyPair = &objects.String{
		Value: "BTC-AUD",
	}
	delimiter = &objects.String{
		Value: "-",
	}
	assetType = &objects.String{
		Value: "SPOT",
	}
	orderID = &objects.String{
		Value: "1235",
	}

	tv            = objects.TrueValue
	fv            = objects.FalseValue
	errTestFailed = errors.New("test failed")
)

func TestMain(m *testing.M) {
	modules.SetModuleWrapper(Wrapper{})
	os.Exit(m.Run())
}

func TestExchangeOrderbook(t *testing.T) {
	t.Parallel()
	_, err := ExchangeOrderbook(exch, currencyPair, delimiter, assetType)
	if err != nil {
		t.Fatal(err)
	}

	_, err = ExchangeOrderbook(exchError, currencyPair, delimiter, assetType)
	if err != nil && errors.Is(err, errTestFailed) {
		t.Fatal(err)
	}

	_, err = ExchangeOrderbook()
	if !errors.Is(err, objects.ErrWrongNumArguments) {
		t.Fatal(err)
	}
}

func TestExchangeTicker(t *testing.T) {
	t.Parallel()
	_, err := ExchangeTicker(exch, currencyPair, delimiter, assetType)
	if err != nil {
		t.Fatal(err)
	}

	_, err = ExchangeTicker(exchError, currencyPair, delimiter, assetType)
	if err != nil && errors.Is(err, errTestFailed) {
		t.Fatal(err)
	}

	_, err = ExchangeTicker()
	if !errors.Is(err, objects.ErrWrongNumArguments) {
		t.Fatal(err)
	}
}

func TestExchangeExchanges(t *testing.T) {
	t.Parallel()

	_, err := ExchangeExchanges(tv)
	if err != nil {
		t.Fatal(err)
	}

	_, err = ExchangeExchanges(fv)
	if err != nil {
		t.Fatal(err)
	}

	_, err = ExchangeExchanges()
	if !errors.Is(err, objects.ErrWrongNumArguments) {
		t.Fatal(err)
	}
}

func TestExchangePairs(t *testing.T) {
	t.Parallel()

	_, err := ExchangePairs(exch, tv, assetType)
	if err != nil {
		t.Fatal(err)
	}

	_, err = ExchangePairs(exchError, tv, assetType)
	if err != nil && errors.Is(err, errTestFailed) {
		t.Fatal(err)
	}

	_, err = ExchangePairs()
	if !errors.Is(err, objects.ErrWrongNumArguments) {
		t.Fatal(err)
	}
}

func TestAccountInfo(t *testing.T) {
	t.Parallel()

	_, err := ExchangeAccountInfo()
	if !errors.Is(err, objects.ErrWrongNumArguments) {
		t.Fatal(err)
	}

	_, err = ExchangeAccountInfo(exch)
	if err != nil {
		t.Fatal(err)
	}

	_, err = ExchangeAccountInfo(exchError)
	if err != nil && !errors.Is(err, errTestFailed) {
		t.Fatal(err)
	}
}

func TestExchangeOrderQuery(t *testing.T) {
	t.Parallel()

	_, err := ExchangeOrderQuery()
	if !errors.Is(err, objects.ErrWrongNumArguments) {
		t.Fatal(err)
	}

	_, err = ExchangeOrderQuery(exch, orderID)
	if err != nil {
		t.Fatal(err)
	}

	_, err = ExchangeOrderQuery(exchError, orderID)
	if err != nil && !errors.Is(err, errTestFailed) {
		t.Fatal(err)
	}
}

func TestExchangeOrderCancel(t *testing.T) {
	_, err := ExchangeOrderCancel()
	if !errors.Is(err, objects.ErrWrongNumArguments) {
		t.Fatal(err)
	}

	_, err = ExchangeOrderCancel(exch, orderID)
	if err != nil {
		t.Fatal(err)
	}

	_, err = ExchangeOrderCancel(exch, objects.FalseValue)
	if err != nil {
		t.Fatal(err)
	}

	_, err = ExchangeOrderCancel(exchError, orderID)
	if err != nil && !errors.Is(err, errTestFailed) {
		t.Fatal(err)
	}
}

func TestExchangeOrderSubmit(t *testing.T) {
	_, err := ExchangeOrderSubmit()
	if !errors.Is(err, objects.ErrWrongNumArguments) {
		t.Fatal(err)
	}

	orderSide := &objects.String{Value: "ASK"}
	orderType := &objects.String{Value: "LIMIT"}
	orderPrice := &objects.Float{Value: 1}
	orderAmount := &objects.Float{Value: 1}

	_, err = ExchangeOrderSubmit(exch, currencyPair, delimiter,
		orderType, orderSide, orderPrice, orderAmount, orderID)
	if err != nil && !errors.Is(err, errTestFailed) {
		t.Fatal(err)
	}

	_, err = ExchangeOrderSubmit(exch, currencyPair, delimiter,
		orderType, orderSide, orderPrice, orderAmount, orderID)
	if err != nil {
		t.Fatal(err)
	}

	_, err = ExchangeOrderSubmit(objects.TrueValue, currencyPair, delimiter,
		orderType, orderSide, orderPrice, orderAmount, orderID)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAllModuleNames(t *testing.T) {
	x := AllModuleNames()
	xType := reflect.TypeOf(x).Kind()
	t.Log(xType)
	if xType != reflect.Slice {
		t.Fatalf("AllModuleNames() should return slice instead received: %v", x)
	}
}

func TestExchangeDepositAddress(t *testing.T) {
	_, err := ExchangeDepositAddress()
	if !errors.Is(err, objects.ErrWrongNumArguments) {
		t.Fatal(err)
	}

	currCode := &objects.String{Value: "BTC"}
	_, err = ExchangeDepositAddress(exch, currCode, orderID)
	if err != nil {
		t.Fatal(err)
	}

	_, err = ExchangeDepositAddress(exchError, currCode, orderID)
	if err != nil && !errors.Is(err, errTestFailed) {
		t.Fatal(err)
	}
}

type Wrapper struct {
}

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

func (w Wrapper) IsEnabled(exch string) (v bool) {
	if exch == exchError.String() {
		return
	}
	return true
}

func (w Wrapper) Orderbook(exch string, pair currency.Pair, item asset.Item) (*orderbook.Base, error) {
	if exch == exchError.String() {
		return nil, errTestFailed
	}

	return &orderbook.Base{
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

func (w Wrapper) Pairs(exch string, enabledOnly bool, item asset.Item) (*currency.Pairs, error) {
	if exch == exchError.String() {
		return nil, errTestFailed
	}

	pairs := currency.NewPairsFromStrings([]string{"btc_usd", "btc_aud", "btc_ltc"})
	return &pairs, nil
}

func (w Wrapper) QueryOrder(exch, orderid string) (*order.Detail, error) {
	if exch == exchError.String() {
		return nil, errTestFailed
	}
	return &order.Detail{
		Exchange:        exch,
		AccountID:       "hello",
		ID:              "1",
		CurrencyPair:    currency.NewPairFromString("BTCAUD"),
		OrderSide:       "ask",
		OrderType:       "limit",
		OrderDate:       time.Now(),
		Status:          "cancelled",
		Price:           1,
		Amount:          2,
		ExecutedAmount:  1,
		RemainingAmount: 0,
		Fee:             0,
		Trades: []order.TradeHistory{
			{
				Timestamp:   time.Now(),
				TID:         0,
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

func (w Wrapper) SubmitOrder(exch string, submit *order.Submit) (*order.SubmitResponse, error) {
	if exch == exchError.String() {
		return nil, errTestFailed
	}

	tempOrder := &order.SubmitResponse{
		IsOrderPlaced: false,
		OrderID:       exch,
	}

	if exch == "true" {
		tempOrder.IsOrderPlaced = true
	}

	fmt.Println(tempOrder)
	return tempOrder, nil
}

func (w Wrapper) CancelOrder(exch, orderid string) (bool, error) {
	if exch == exchError.String() {
		return false, errTestFailed
	}
	if orderid == "false" {
		return false, nil
	}
	return true, nil
}

func (w Wrapper) AccountInformation(exch string) (modules.AccountInfo, error) {
	if exch == exchError.String() {
		return modules.AccountInfo{}, errTestFailed
	}

	return modules.AccountInfo{
		Exchange: exch,
		Accounts: []modules.Account{
			{
				ID: exch,
				Currencies: []modules.AccountCurrencyInfo{
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

func (w Wrapper) DepositAddress(exch string, currencyCode currency.Code, accountID string) (string, error) {
	if exch == exchError.String() {
		return exch, errTestFailed
	}
	return exch, nil
}

func (w Wrapper) WithdrawalFunds() error {
	return nil
}
