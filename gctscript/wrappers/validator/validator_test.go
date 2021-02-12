package validator

import (
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

const (
	exchName      = "BTC Markets" // change to test on another exchange
	exchAPIKEY    = ""
	exchAPISECRET = ""
	exchClientID  = ""
	pairs         = "BTC-AUD" // change to test another currency pair
	delimiter     = "-"
	assetType     = asset.Spot
	orderID       = "1234"
	orderType     = order.Limit
	orderSide     = order.Buy
	orderClientID = ""
	orderPrice    = 1
	orderAmount   = 1
)

var (
	currencyPair, _ = currency.NewPairFromString("BTCAUD")
	testWrapper     = Wrapper{}
)

func TestWrapper_Exchanges(t *testing.T) {
	t.Parallel()
	x := testWrapper.Exchanges(false)
	y := len(x)
	if y != 1 {
		t.Fatalf("expected 1 received %v", y)
	}

	x = testWrapper.Exchanges(true)
	y = len(x)
	if y != 1 {
		t.Fatalf("expected 1 received %v", y)
	}
}

func TestWrapper_IsEnabled(t *testing.T) {
	t.Parallel()
	f := testWrapper.IsEnabled("hello")
	if !f {
		t.Fatal("expected IsEnabled to return true for enabled exchange")
	}

	f = testWrapper.IsEnabled(exchError.String())
	if f {
		t.Fatal("expected IsEnabled to return false for disabled exchange")
	}
}

func TestWrapper_AccountInformation(t *testing.T) {
	t.Parallel()

	_, err := testWrapper.AccountInformation(exchName, asset.Spot)
	if err != nil {
		t.Fatal(err)
	}

	_, err = testWrapper.AccountInformation(exchError.String(), asset.Spot)
	if err == nil {
		t.Fatal("expected AccountInformation to return error on invalid name")
	}
}

func TestWrapper_CancelOrder(t *testing.T) {
	t.Parallel()
	cp := currency.NewPair(currency.BTC, currency.USD)
	_, err := testWrapper.CancelOrder(exchName, orderID, cp, assetType)
	if err != nil {
		t.Error(err)
	}

	_, err = testWrapper.CancelOrder(exchError.String(), orderID, cp, assetType)
	if err == nil {
		t.Error("expected CancelOrder to return error on invalid name")
	}

	_, err = testWrapper.CancelOrder(exchName, "", cp, assetType)
	if err == nil {
		t.Error("expected CancelOrder to return error on invalid name")
	}

	_, err = testWrapper.CancelOrder(exchName, orderID, currency.Pair{}, assetType)
	if err != nil {
		t.Error(err)
	}

	_, err = testWrapper.CancelOrder(exchName, orderID, cp, "")
	if err != nil {
		t.Error(err)
	}
}

func TestWrapper_DepositAddress(t *testing.T) {
	_, err := testWrapper.DepositAddress(exchError.String(), currency.NewCode("BTC"))
	if err == nil {
		t.Fatal("expected DepositAddress to return error on invalid name")
	}

	_, err = testWrapper.DepositAddress(exchName, currency.NewCode("BTC"))
	if err != nil {
		t.Fatal(err)
	}
}

func TestWrapper_Orderbook(t *testing.T) {
	t.Parallel()
	c, err := currency.NewPairDelimiter(pairs, delimiter)
	if err != nil {
		t.Fatal(err)
	}
	_, err = testWrapper.Orderbook(exchName, c, assetType)
	if err != nil {
		t.Fatal(err)
	}

	_, err = testWrapper.Orderbook(exchError.String(), currencyPair, asset.Spot)
	if err == nil {
		t.Fatal("expected Orderbook to return error with invalid name")
	}
}

func TestWrapper_Pairs(t *testing.T) {
	t.Parallel()
	_, err := testWrapper.Pairs(exchName, false, assetType)
	if err != nil {
		t.Fatal(err)
	}
	_, err = testWrapper.Pairs(exchName, true, assetType)
	if err != nil {
		t.Fatal(err)
	}

	_, err = testWrapper.Pairs(exchError.String(), false, asset.Spot)
	if err == nil {
		t.Fatal("expected Pairs to return error on invalid name")
	}
}

func TestWrapper_QueryOrder(t *testing.T) {
	t.Parallel()

	_, err := testWrapper.QueryOrder(exchName, orderID, currency.Pair{}, assetType)
	if err != nil {
		t.Fatal(err)
	}

	_, err = testWrapper.QueryOrder(exchError.String(), "", currency.Pair{}, assetType)
	if err == nil {
		t.Fatal("expected QueryOrder to return error on invalid name")
	}
}

func TestWrapper_SubmitOrder(t *testing.T) {
	t.Parallel()
	c, err := currency.NewPairDelimiter(pairs, delimiter)
	if err != nil {
		t.Fatal(err)
	}
	tempOrder := &order.Submit{
		Pair:         c,
		Type:         orderType,
		Side:         orderSide,
		TriggerPrice: 0,
		TargetAmount: 0,
		Price:        orderPrice,
		Amount:       orderAmount,
		ClientID:     orderClientID,
		Exchange:     "true",
		AssetType:    asset.Spot,
	}
	_, err = testWrapper.SubmitOrder(tempOrder)
	if err != nil {
		t.Fatal(err)
	}

	_, err = testWrapper.SubmitOrder(nil)
	if err == nil {
		t.Fatal("expected SubmitOrder to return error with invalid name")
	}
}

func TestWrapper_Ticker(t *testing.T) {
	t.Parallel()
	c, err := currency.NewPairDelimiter(pairs, delimiter)
	if err != nil {
		t.Fatal(err)
	}
	_, err = testWrapper.Ticker(exchName, c, assetType)
	if err != nil {
		t.Fatal(err)
	}

	_, err = testWrapper.Ticker(exchError.String(), currencyPair, asset.Spot)
	if err == nil {
		t.Fatal("expected Ticker to return error with invalid name")
	}
}

func TestWrapper_WithdrawalCryptoFunds(t *testing.T) {
	_, err := testWrapper.WithdrawalCryptoFunds(&withdraw.Request{Exchange: exchError.String()})
	if err == nil {
		t.Fatal("expected WithdrawalCryptoFunds to return error with invalid name")
	}

	_, err = testWrapper.WithdrawalCryptoFunds(&withdraw.Request{Exchange: exchName})
	if err != nil {
		t.Fatal("expected WithdrawalCryptoFunds to return error with invalid name")
	}
}

func TestWrapper_WithdrawalFiatFunds(t *testing.T) {
	_, err := testWrapper.WithdrawalFiatFunds("", &withdraw.Request{Exchange: exchError.String()})
	if err == nil {
		t.Fatal("expected WithdrawalFiatFunds to return error with invalid name")
	}

	_, err = testWrapper.WithdrawalFiatFunds("", &withdraw.Request{Exchange: exchName})
	if err != nil {
		t.Fatal("expected WithdrawalCryptoFunds to return error with invalid name")
	}
}

func TestWrapper_OHLCV(t *testing.T) {
	c, err := currency.NewPairDelimiter(pairs, delimiter)
	if err != nil {
		t.Fatal(err)
	}
	_, err = testWrapper.OHLCV("test", c, asset.Spot, time.Now().Add(-24*time.Hour), time.Now(), kline.OneDay)
	if err != nil {
		t.Fatal(err)
	}
	_, err = testWrapper.OHLCV(exchError.String(), c, asset.Spot, time.Now().Add(-24*time.Hour), time.Now(), kline.OneDay)
	if err == nil {
		t.Fatal("expected OHLCV to return error with invalid name")
	}
}
