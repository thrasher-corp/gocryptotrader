package gateio

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// Please supply your own APIKEYS here for due diligence testing

const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var g Gateio
var wsSetupRan bool

func TestMain(m *testing.M) {
	g.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal("GateIO load config error", err)
	}
	gConf, err := cfg.GetExchangeConfig("GateIO")
	if err != nil {
		log.Fatal("GateIO Setup() init error")
	}
	gConf.API.AuthenticatedSupport = true
	gConf.API.AuthenticatedWebsocketSupport = true
	gConf.API.Credentials.Key = apiKey
	gConf.API.Credentials.Secret = apiSecret
	g.Websocket = sharedtestvalues.NewTestWebsocket()
	err = g.Setup(gConf)
	if err != nil {
		log.Fatal("GateIO setup error", err)
	}

	os.Exit(m.Run())
}

func TestStart(t *testing.T) {
	t.Parallel()
	err := g.Start(nil)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Fatalf("received: '%v' but expected: '%v'", err, common.ErrNilPointer)
	}
	var testWg sync.WaitGroup
	err = g.Start(&testWg)
	if err != nil {
		t.Fatal(err)
	}
	testWg.Wait()
}

func TestGetSymbols(t *testing.T) {
	t.Parallel()
	_, err := g.GetSymbols(context.Background())
	if err != nil {
		t.Errorf("Gateio TestGetSymbols: %s", err)
	}
}

func TestGetMarketInfo(t *testing.T) {
	t.Parallel()
	_, err := g.GetMarketInfo(context.Background())
	if err != nil {
		t.Errorf("Gateio GetMarketInfo: %s", err)
	}
}

func TestSpotNewOrder(t *testing.T) {
	t.Parallel()

	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip()
	}

	_, err := g.SpotNewOrder(context.Background(),
		SpotNewOrderRequestParams{
			Symbol: "btc_usdt",
			Amount: -1,
			Price:  100000,
			Type:   order.Sell.Lower(),
		})
	if err != nil {
		t.Errorf("Gateio SpotNewOrder: %s", err)
	}
}

func TestCancelExistingOrder(t *testing.T) {
	t.Parallel()

	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip()
	}

	_, err := g.CancelExistingOrder(context.Background(), 917591554, "btc_usdt")
	if err != nil {
		t.Errorf("Gateio CancelExistingOrder: %s", err)
	}
}

func TestGetBalances(t *testing.T) {
	t.Parallel()

	if !areTestAPIKeysSet() {
		t.Skip()
	}

	_, err := g.GetBalances(context.Background())
	if err != nil {
		t.Errorf("Gateio GetBalances: %s", err)
	}
}

func TestGetLatestSpotPrice(t *testing.T) {
	t.Parallel()
	_, err := g.GetLatestSpotPrice(context.Background(), "btc_usdt")
	if err != nil {
		t.Errorf("Gateio GetLatestSpotPrice: %s", err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := g.GetTicker(context.Background(), "btc_usdt")
	if err != nil {
		t.Errorf("Gateio GetTicker: %s", err)
	}
}

func TestGetTickers(t *testing.T) {
	t.Parallel()
	_, err := g.GetTickers(context.Background())
	if err != nil {
		t.Errorf("Gateio GetTicker: %s", err)
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := g.GetOrderbook(context.Background(), "btc_usdt")
	if err != nil {
		t.Errorf("Gateio GetTicker: %s", err)
	}
}

func TestGetSpotKline(t *testing.T) {
	t.Parallel()
	_, err := g.GetSpotKline(context.Background(),
		KlinesRequestParams{
			Symbol:   "btc_usdt",
			GroupSec: "5", // 5 minutes or less
			HourSize: 1,   // 1 hour data
		})

	if err != nil {
		t.Errorf("Gateio GetSpotKline: %s", err)
	}
}

func setFeeBuilder() *exchange.FeeBuilder {
	return &exchange.FeeBuilder{
		Amount:  1,
		FeeType: exchange.CryptocurrencyTradeFee,
		Pair: currency.NewPairWithDelimiter(currency.BTC.String(),
			currency.USDT.String(), "_"),
		IsMaker:             false,
		PurchasePrice:       1,
		FiatCurrency:        currency.USD,
		BankTransactionType: exchange.WireTransfer,
	}
}

func TestGetTradeHistory(t *testing.T) {
	_, err := g.GetTrades(context.Background(),
		currency.NewPairWithDelimiter(currency.BTC.String(),
			currency.USDT.String(), "_").String())
	if err != nil {
		t.Error(err)
	}
}

// TestGetFeeByTypeOfflineTradeFee logic test
func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	var feeBuilder = setFeeBuilder()
	_, err := g.GetFeeByType(context.Background(), feeBuilder)
	if err != nil {
		t.Fatal(err)
	}
	if !areTestAPIKeysSet() {
		if feeBuilder.FeeType != exchange.OfflineTradeFee {
			t.Errorf("Expected %v, received %v", exchange.OfflineTradeFee, feeBuilder.FeeType)
		}
	} else {
		if feeBuilder.FeeType != exchange.CryptocurrencyTradeFee {
			t.Errorf("Expected %v, received %v", exchange.CryptocurrencyTradeFee, feeBuilder.FeeType)
		}
	}
}

func TestGetFee(t *testing.T) {
	var feeBuilder = setFeeBuilder()
	if areTestAPIKeysSet() {
		// CryptocurrencyTradeFee Basic
		if _, err := g.GetFee(context.Background(), feeBuilder); err != nil {
			t.Error(err)
		}

		// CryptocurrencyTradeFee High quantity
		feeBuilder = setFeeBuilder()
		feeBuilder.Amount = 1000
		feeBuilder.PurchasePrice = 1000
		if _, err := g.GetFee(context.Background(), feeBuilder); err != nil {
			t.Error(err)
		}

		// CryptocurrencyTradeFee IsMaker
		feeBuilder = setFeeBuilder()
		feeBuilder.IsMaker = true
		if _, err := g.GetFee(context.Background(), feeBuilder); err != nil {
			t.Error(err)
		}

		// CryptocurrencyTradeFee Negative purchase price
		feeBuilder = setFeeBuilder()
		feeBuilder.PurchasePrice = -1000
		if _, err := g.GetFee(context.Background(), feeBuilder); err != nil {
			t.Error(err)
		}
	}
	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if _, err := g.GetFee(context.Background(), feeBuilder); err != nil {
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Invalid currency
	feeBuilder = setFeeBuilder()
	feeBuilder.Pair.Base = currency.NewCode("hello")
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if _, err := g.GetFee(context.Background(), feeBuilder); err != nil {
		t.Error(err)
	}

	// CryptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyDepositFee
	if _, err := g.GetFee(context.Background(), feeBuilder); err != nil {
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	if _, err := g.GetFee(context.Background(), feeBuilder); err != nil {
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.USD
	if _, err := g.GetFee(context.Background(), feeBuilder); err != nil {
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	expectedResult := exchange.AutoWithdrawCryptoText + " & " + exchange.NoFiatWithdrawalsText
	withdrawPermissions := g.FormatWithdrawPermissions()
	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s", expectedResult, withdrawPermissions)
	}
}

func TestGetActiveOrders(t *testing.T) {
	var getOrdersRequest = order.GetOrdersRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
	}

	_, err := g.GetActiveOrders(context.Background(), &getOrdersRequest)
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
	}

	currPair := currency.NewPair(currency.LTC, currency.BTC)
	currPair.Delimiter = "_"
	getOrdersRequest.Pairs = []currency.Pair{currPair}

	_, err := g.GetOrderHistory(context.Background(), &getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get order history: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------
func areTestAPIKeysSet() bool {
	return g.ValidateAPICredentials(g.GetDefaultCredentials()) == nil
}

func TestSubmitOrder(t *testing.T) {
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip()
	}

	var orderSubmission = &order.Submit{
		Pair: currency.Pair{
			Delimiter: "_",
			Base:      currency.LTC,
			Quote:     currency.BTC,
		},
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     1,
		Amount:    1,
		ClientID:  "meowOrder",
		AssetType: asset.Spot,
	}
	response, err := g.SubmitOrder(context.Background(), orderSubmission)
	if areTestAPIKeysSet() && (err != nil || !response.IsOrderPlaced) {
		t.Errorf("Order failed to be placed: %v", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip()
	}

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)
	var orderCancellation = &order.Cancel{
		ID:            "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          currencyPair,
		AssetType:     asset.Spot,
	}

	err := g.CancelOrder(context.Background(), orderCancellation)
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip()
	}

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)
	var orderCancellation = &order.Cancel{
		ID:            "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          currencyPair,
		AssetType:     asset.Spot,
	}

	resp, err := g.CancelAllOrders(context.Background(), orderCancellation)

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

func TestGetAccountInfo(t *testing.T) {
	if apiSecret == "" || apiKey == "" {
		_, err := g.UpdateAccountInfo(context.Background(), asset.Spot)
		if err == nil {
			t.Error("GetAccountInfo() Expected error")
		}
	} else {
		_, err := g.UpdateAccountInfo(context.Background(), asset.Spot)
		if err != nil {
			t.Error("GetAccountInfo() error", err)
		}
	}
}

func TestModifyOrder(t *testing.T) {
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	_, err := g.ModifyOrder(context.Background(),
		&order.Modify{AssetType: asset.Spot})
	if err == nil {
		t.Error("ModifyOrder() Expected error")
	}
}

func TestWithdraw(t *testing.T) {
	withdrawCryptoRequest := withdraw.Request{
		Exchange:    g.Name,
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

	_, err := g.WithdrawCryptocurrencyFunds(context.Background(),
		&withdrawCryptoRequest)
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
	_, err := g.WithdrawFiatFunds(context.Background(), &withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = withdraw.Request{}
	_, err := g.WithdrawFiatFundsToInternationalBank(context.Background(),
		&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	if areTestAPIKeysSet() {
		_, err := g.GetDepositAddress(context.Background(), currency.USDT, "", "TRX")
		if err != nil {
			t.Error("Test Fail - GetDepositAddress error", err)
		}
	} else {
		_, err := g.GetDepositAddress(context.Background(), currency.ETC, "", "")
		if err == nil {
			t.Error("Test Fail - GetDepositAddress error cannot be nil")
		}
	}
}
func TestGetOrderInfo(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip("no API keys set skipping test")
	}

	_, err := g.GetOrderInfo(context.Background(),
		"917591554", currency.EMPTYPAIR, asset.Spot)
	if err != nil {
		if err.Error() != "no order found with id 917591554" && err.Error() != "failed to get open orders" {
			t.Fatalf("GetOrderInfo() returned an error skipping test: %v", err)
		}
	}
}

// TestWsGetBalance dials websocket, sends balance request.
func TestWsGetBalance(t *testing.T) {
	if !g.Websocket.IsEnabled() && !g.API.AuthenticatedWebsocketSupport || !areTestAPIKeysSet() {
		t.Skip(stream.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
	err := g.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		t.Fatal(err)
	}
	go g.wsReadData()
	err = g.wsServerSignIn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	_, err = g.wsGetBalance([]string{"EOS", "BTC"})
	if err != nil {
		t.Error(err)
	}
	_, err = g.wsGetBalance([]string{})
	if err != nil {
		t.Error(err)
	}
}

// TestWsGetOrderInfo dials websocket, sends order info request.
func TestWsGetOrderInfo(t *testing.T) {
	if !g.Websocket.IsEnabled() && !g.API.AuthenticatedWebsocketSupport || !areTestAPIKeysSet() {
		t.Skip(stream.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
	err := g.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		t.Fatal(err)
	}
	go g.wsReadData()
	err = g.wsServerSignIn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	_, err = g.wsGetOrderInfo("EOS_USDT", 0, 100)
	if err != nil {
		t.Error(err)
	}
}

func setupWSTestAuth(t *testing.T) {
	t.Helper()
	if wsSetupRan {
		return
	}
	if !g.Websocket.IsEnabled() && !g.API.AuthenticatedWebsocketSupport || !areTestAPIKeysSet() {
		t.Skip(stream.WebsocketNotEnabled)
	}
	if err := g.Websocket.Connect(); err != nil {
		t.Fatal(err)
	}
	wsSetupRan = true
}

// TestWsSubscribe dials websocket, sends a subscribe request.
func TestWsSubscribe(t *testing.T) {
	setupWSTestAuth(t)
	err := g.Subscribe([]stream.ChannelSubscription{
		{
			Channel:  "ticker.subscribe",
			Currency: currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), "_"),
		},
	})
	if err != nil {
		t.Error(err)
	}

	err = g.Unsubscribe([]stream.ChannelSubscription{
		{
			Channel:  "ticker.subscribe",
			Currency: currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), "_"),
		},
	})
	if err != nil {
		t.Error(err)
	}
}

func TestWsTicker(t *testing.T) {
	pressXToJSON := []byte(`{
    "method": "ticker.update",
    "params":
        [
            "BTC_USDT",
                {
                    "period": 86400,
                    "open": "0",
                    "close": "0",
                    "high": "0",
                    "low": "0",
                    "last": "0.2844",
                    "change": "0",
                    "quoteVolume": "0",
                    "baseVolume": "0"
                }
     ],
     "id": null
}`)
	err := g.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsTrade(t *testing.T) {
	pressXToJSON := []byte(`{
    "method": "trades.update",
    "params":
        [
             "BTC_USDT",
             [
                 {
                 "id": 7172173,
                 "time": 1523339279.761838,
                 "price": "398.59",
                 "amount": "0.027",
                 "type": "buy"
                 }
             ]
         ],
     "id": null
 }
`)
	err := g.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsDepth(t *testing.T) {
	pressXToJSON := []byte(`{
    "method": "depth.update",
    "params": [
        true,
        {
            "asks": [
                [
                    "8000.00",
                    "9.6250"
                ]
            ],
            "bids": [
                [
                    "8000.00",
                    "9.6250"
                ]
            ]
         },
         "BTC_USDT"
    ],
    "id": null
 }`)
	err := g.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsKLine(t *testing.T) {
	pressXToJSON := []byte(`{
    "method": "kline.update",
    "params":
        [
            [
                1492358400,
                "7000.00",
                "8000.0",
                "8100.00",
                "6800.00",
                "1000.00",
                "123456.00",
                "BTC_USDT"
            ]
        ],
    "id": null
}`)
	err := g.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsOrderUpdate(t *testing.T) {
	pressXToJSON := []byte(`{
  "method": "order.update",
  "params": [
    3,
    {
      "id": 34628963,
      "market": "BTC_USDT",
      "orderType": 1,
      "type": 2,
      "user": 602123,
      "ctime": 1523013969.6271579,
      "mtime": 1523013969.6271579,
      "price": "0.1",
      "amount": "1000",
      "left": "1000",
      "filledAmount": "0",
      "filledTotal": "0",
      "dealFee": "0"
    }
  ],
  "id": null
}`)
	err := g.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsBalanceUpdate(t *testing.T) {
	pressXToJSON := []byte(`{
    "method": "balance.update",
    "params": [{"EOS": {"available": "96.765323611874", "freeze": "11"}}],
    "id": 1234
}`)
	err := g.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestParseTime(t *testing.T) {
	// Test REST example
	r := convert.TimeFromUnixTimestampDecimal(1574846296.995313).UTC()
	if r.Year() != 2019 ||
		r.Month().String() != "November" ||
		r.Day() != 27 {
		t.Error("unexpected result")
	}

	// Test websocket example
	r = convert.TimeFromUnixTimestampDecimal(1523887354.256974).UTC()
	if r.Year() != 2018 ||
		r.Month().String() != "April" ||
		r.Day() != 16 {
		t.Error("unexpected result")
	}
}

func TestGetHistoricCandles(t *testing.T) {
	currencyPair, err := currency.NewPairFromString("BTC_USDT")
	if err != nil {
		t.Fatal(err)
	}
	startTime := time.Now().Add(-time.Hour * 6)
	_, err = g.GetHistoricCandles(context.Background(),
		currencyPair, asset.Spot, startTime, time.Now(), kline.OneMin)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	currencyPair, err := currency.NewPairFromString("BTC_USDT")
	if err != nil {
		t.Fatal(err)
	}
	startTime := time.Now().Add(-time.Minute * 2)
	_, err = g.GetHistoricCandlesExtended(context.Background(),
		currencyPair, asset.Spot, startTime, time.Now(), kline.OneMin)
	if err != nil {
		t.Fatal(err)
	}
}

func Test_FormatExchangeKlineInterval(t *testing.T) {
	testCases := []struct {
		name     string
		interval kline.Interval
		output   string
	}{
		{
			"OneMin",
			kline.OneMin,
			"60",
		},
		{
			"OneDay",
			kline.OneDay,
			"86400",
		},
	}

	for x := range testCases {
		test := testCases[x]

		t.Run(test.name, func(t *testing.T) {
			ret := g.FormatExchangeKlineInterval(test.interval)

			if ret != test.output {
				t.Fatalf("unexpected result return expected: %v received: %v", test.output, ret)
			}
		})
	}
}

func TestGenerateDefaultSubscriptions(t *testing.T) {
	err := g.CurrencyPairs.EnablePair(asset.Spot, currency.NewPair(
		currency.LTC,
		currency.USDT,
	))
	if err != nil {
		t.Fatal(err)
	}
	subs, err := g.GenerateDefaultSubscriptions()
	if err != nil {
		t.Fatal(err)
	}

	payload, err := g.generatePayload(subs)
	if err != nil {
		t.Fatal(err)
	}

	if len(payload) != 4 {
		t.Fatal("unexpected payload length")
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	currencyPair, err := currency.NewPairFromString("btc_usdt")
	if err != nil {
		t.Fatal(err)
	}
	_, err = g.GetRecentTrades(context.Background(), currencyPair, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	currencyPair, err := currency.NewPairFromString("btc_usdt")
	if err != nil {
		t.Fatal(err)
	}
	_, err = g.GetHistoricTrades(context.Background(),
		currencyPair, asset.Spot, time.Now().Add(-time.Minute*15), time.Now())
	if err != nil && err != common.ErrFunctionNotSupported {
		t.Error(err)
	}
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString("btc_usdt")
	if err != nil {
		t.Fatal(err)
	}
	_, err = g.UpdateTicker(context.Background(), cp, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateTickers(t *testing.T) {
	err := g.UpdateTickers(context.Background(), asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetCryptoDepositAddress(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("api keys not set")
	}
	_, err := g.GetCryptoDepositAddress(context.Background(), currency.USDT.String())
	if err != nil {
		t.Error(err)
	}
}

func TestGetAvailableTransferTrains(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("api keys not set")
	}
	_, err := g.GetAvailableTransferChains(context.Background(), currency.USDT)
	if err != nil {
		t.Error(err)
	}
}
