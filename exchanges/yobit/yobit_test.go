package yobit

import (
	"log"
	"math"
	"os"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

var y Yobit

// Please supply your own keys for better unit testing
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

func TestMain(m *testing.M) {
	y.SetDefaults()
	yobitConfig := config.GetConfig()
	err := yobitConfig.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal("Yobit load config error", err)
	}
	conf, err := yobitConfig.GetExchangeConfig("Yobit")
	if err != nil {
		log.Fatal("Yobit init error", err)
	}
	conf.API.Credentials.Key = apiKey
	conf.API.Credentials.Secret = apiSecret
	conf.API.AuthenticatedSupport = true

	err = y.Setup(conf)
	if err != nil {
		log.Fatal("Yobit setup error", err)
	}

	os.Exit(m.Run())
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := y.FetchTradablePairs(asset.Spot)
	if err != nil {
		t.Errorf("FetchTradablePairs err: %s", err)
	}
}

func TestGetInfo(t *testing.T) {
	t.Parallel()
	_, err := y.GetInfo()
	if err != nil {
		t.Error("GetInfo() error")
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := y.GetTicker("btc_usd")
	if err != nil {
		t.Error("GetTicker() error", err)
	}
}

func TestGetDepth(t *testing.T) {
	t.Parallel()
	_, err := y.GetDepth("btc_usd")
	if err != nil {
		t.Error("GetDepth() error", err)
	}
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := y.GetTrades("btc_usd")
	if err != nil {
		t.Error("GetTrades() error", err)
	}
}

func TestGetAccountInfo(t *testing.T) {
	t.Parallel()
	_, err := y.UpdateAccountInfo(asset.Spot)
	if err == nil {
		t.Error("GetAccountInfo() Expected error")
	}
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := y.GetOpenOrders("")
	if err == nil {
		t.Error("GetOpenOrders() Expected error")
	}
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	_, err := y.GetOrderInfo("6196974", currency.Pair{}, asset.Spot)
	if err == nil {
		t.Error("GetOrderInfo() Expected error")
	}
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	err := y.CancelExistingOrder(1337)
	if err == nil {
		t.Error("CancelOrder() Expected error")
	}
}

func TestTrade(t *testing.T) {
	t.Parallel()
	_, err := y.Trade("", order.Buy.String(), 0, 0)
	if err == nil {
		t.Error("Trade() Expected error")
	}
}

func TestWithdrawCoinsToAddress(t *testing.T) {
	t.Parallel()
	_, err := y.WithdrawCoinsToAddress("", 0, "")
	if err == nil {
		t.Error("WithdrawCoinsToAddress() Expected error")
	}
}

func TestCreateYobicode(t *testing.T) {
	t.Parallel()
	_, err := y.CreateCoupon("bla", 0)
	if err == nil {
		t.Error("CreateYobicode() Expected error")
	}
}

func TestRedeemYobicode(t *testing.T) {
	t.Parallel()
	_, err := y.RedeemCoupon("bla2")
	if err == nil {
		t.Error("RedeemYobicode() Expected error")
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	expectedResult := exchange.AutoWithdrawCryptoWithAPIPermissionText + " & " + exchange.WithdrawFiatViaWebsiteOnlyText
	withdrawPermissions := y.FormatWithdrawPermissions()
	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s", expectedResult, withdrawPermissions)
	}
}

func TestGetActiveOrders(t *testing.T) {
	var getOrdersRequest = order.GetOrdersRequest{
		Type: order.AnyType,
		Pairs: []currency.Pair{currency.NewPair(currency.LTC,
			currency.BTC)},
		AssetType: asset.Spot,
	}

	_, err := y.GetActiveOrders(&getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get open orders: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestGetOrderHistory(t *testing.T) {
	var getOrdersRequest = order.GetOrdersRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
		Pairs: []currency.Pair{currency.NewPair(currency.LTC,
			currency.BTC)},
		StartTime: time.Unix(0, 0),
		EndTime:   time.Unix(math.MaxInt64, 0),
	}

	_, err := y.GetOrderHistory(&getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get order history: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------
func areTestAPIKeysSet() bool {
	return y.ValidateAPICredentials()
}

func TestSubmitOrder(t *testing.T) {
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var orderSubmission = &order.Submit{
		Pair: currency.Pair{
			Delimiter: "_",
			Base:      currency.BTC,
			Quote:     currency.USD,
		},
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     1,
		Amount:    1,
		ClientID:  "meowOrder",
		AssetType: asset.Spot,
	}
	response, err := y.SubmitOrder(orderSubmission)
	if areTestAPIKeysSet() && (err != nil || !response.IsOrderPlaced) {
		t.Errorf("Order failed to be placed: %v", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)
	var orderCancellation = &order.Cancel{
		ID:            "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          currencyPair,
		AssetType:     asset.Spot,
	}

	err := y.CancelOrder(orderCancellation)
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)
	var orderCancellation = &order.Cancel{
		ID:            "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          currencyPair,
		AssetType:     asset.Spot,
	}

	resp, err := y.CancelAllOrders(orderCancellation)

	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}

	if len(resp.Status) > 0 {
		t.Errorf("%v orders failed to cancel", len(resp.Status))
	}
}

func TestModifyOrder(t *testing.T) {
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	_, err := y.ModifyOrder(&order.Modify{AssetType: asset.Spot})
	if err == nil {
		t.Error("ModifyOrder() Expected error")
	}
}

func TestWithdraw(t *testing.T) {
	withdrawCryptoRequest := withdraw.Request{
		Amount:      -1,
		Currency:    currency.BTC,
		Description: "WITHDRAW IT ALL",
		Crypto: withdraw.CryptoRequest{
			Address: core.BitcoinDonationAddress,
		},
	}

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	_, err := y.WithdrawCryptocurrencyFunds(&withdrawCryptoRequest)
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

func TestWithdrawFiat(t *testing.T) {
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = withdraw.Request{}
	_, err := y.WithdrawFiatFunds(&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'",
			common.ErrFunctionNotSupported,
			err)
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = withdraw.Request{}
	_, err := y.WithdrawFiatFundsToInternationalBank(&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'",
			common.ErrFunctionNotSupported,
			err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	if areTestAPIKeysSet() {
		_, err := y.GetDepositAddress(currency.BTC, "")
		if err != nil {
			t.Error("GetDepositAddress() Expected error")
		}
	} else {
		_, err := y.GetDepositAddress(currency.BTC, "")
		if err == nil {
			t.Error("GetDepositAddress() error")
		}
	}
}

func TestGetRecentTrades(t *testing.T) {
	currencyPair, err := currency.NewPairFromString("btc_usd")
	if err != nil {
		t.Fatal(err)
	}
	_, err = y.GetRecentTrades(currencyPair, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetHistoricTrades(t *testing.T) {
	currencyPair, err := currency.NewPairFromString("btc_usd")
	if err != nil {
		t.Fatal(err)
	}
	_, err = y.GetHistoricTrades(currencyPair, asset.Spot, time.Now().Add(-time.Minute*15), time.Now())
	if err != nil && err != common.ErrFunctionNotSupported {
		t.Error(err)
	}
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString("ETH_BTC")
	if err != nil {
		t.Fatal(err)
	}
	_, err = y.UpdateTicker(cp, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	err := y.UpdateTickers(asset.Spot)
	if err != nil {
		t.Error(err)
	}
}
