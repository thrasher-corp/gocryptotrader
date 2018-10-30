package kraken

import (
	"fmt"
	"strings"
	"testing"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
)

var k Kraken

// Please add your own APIkeys to do correct due diligence testing.
const (
	apiKey                  = ""
	apiSecret               = ""
	clientID                = ""
	canManipulateRealOrders = false
)

// TestSetDefaults setup func
func TestSetDefaults(t *testing.T) {
	k.SetDefaults()
}

// TestSetup setup func
func TestSetup(t *testing.T) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	krakenConfig, err := cfg.GetExchangeConfig("Kraken")
	if err != nil {
		t.Error("Test Failed - kraken Setup() init error", err)
	}
	krakenConfig.API.AuthenticatedSupport = true
	krakenConfig.API.Credentials.Key = apiKey
	krakenConfig.API.Credentials.Secret = apiSecret
	krakenConfig.API.Credentials.ClientID = clientID
	krakenConfig.API.Endpoints.WebsocketURL = k.API.Endpoints.WebsocketURL
	subscribeToDefaultChannels = false

	k.Setup(krakenConfig)
}

// TestGetServerTime API endpoint test
func TestGetServerTime(t *testing.T) {
	t.Parallel()
	_, err := k.GetServerTime()
	if err != nil {
		t.Error("Test Failed - GetServerTime() error", err)
	}
}

// TestGetAssets API endpoint test
func TestGetAssets(t *testing.T) {
	t.Parallel()
	_, err := k.GetAssets()
	if err != nil {
		t.Error("Test Failed - GetAssets() error", err)
	}
}

// TestGetAssetPairs API endpoint test
func TestGetAssetPairs(t *testing.T) {
	t.Parallel()
	_, err := k.GetAssetPairs()
	if err != nil {
		t.Error("Test Failed - GetAssetPairs() error", err)
	}
}

// TestGetTicker API endpoint test
func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := k.GetTicker("BCHEUR")
	if err != nil {
		t.Error("Test Failed - GetTicker() error", err)
	}
}

// TestGetTickers API endpoint test
func TestGetTickers(t *testing.T) {
	t.Parallel()
	_, err := k.GetTickers("LTCUSD,ETCUSD")
	if err != nil {
		t.Error("Test failed - GetTickers() error", err)
	}
}

// TestGetOHLC API endpoint test
func TestGetOHLC(t *testing.T) {
	t.Parallel()
	_, err := k.GetOHLC("BCHEUR")
	if err != nil {
		t.Error("Test Failed - GetOHLC() error", err)
	}
}

// TestGetDepth API endpoint test
func TestGetDepth(t *testing.T) {
	t.Parallel()
	_, err := k.GetDepth("BCHEUR")
	if err != nil {
		t.Error("Test Failed - GetDepth() error", err)
	}
}

// TestGetTrades API endpoint test
func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := k.GetTrades("BCHEUR")
	if err != nil {
		t.Error("Test Failed - GetTrades() error", err)
	}
}

// TestGetSpread API endpoint test
func TestGetSpread(t *testing.T) {
	t.Parallel()
	_, err := k.GetSpread("BCHEUR")
	if err != nil {
		t.Error("Test Failed - GetSpread() error", err)
	}
}

// TestGetBalance API endpoint test
func TestGetBalance(t *testing.T) {
	t.Parallel()
	_, err := k.GetBalance()
	if err == nil {
		t.Error("Test Failed - GetBalance() error", err)
	}
}

// TestGetTradeBalance API endpoint test
func TestGetTradeBalance(t *testing.T) {
	t.Parallel()
	args := TradeBalanceOptions{Asset: "ZEUR"}
	_, err := k.GetTradeBalance(args)
	if err == nil {
		t.Error("Test Failed - GetTradeBalance() error", err)
	}
}

// TestGetOpenOrders API endpoint test
func TestGetOpenOrders(t *testing.T) {
	t.Parallel()
	args := OrderInfoOptions{Trades: true}
	_, err := k.GetOpenOrders(args)
	if err == nil {
		t.Error("Test Failed - GetOpenOrders() error", err)
	}
}

// TestGetClosedOrders API endpoint test
func TestGetClosedOrders(t *testing.T) {
	t.Parallel()
	args := GetClosedOrdersOptions{Trades: true, Start: "OE4KV4-4FVQ5-V7XGPU"}
	_, err := k.GetClosedOrders(args)
	if err == nil {
		t.Error("Test Failed - GetClosedOrders() error", err)
	}
}

// TestQueryOrdersInfo API endpoint test
func TestQueryOrdersInfo(t *testing.T) {
	t.Parallel()
	args := OrderInfoOptions{Trades: true}
	_, err := k.QueryOrdersInfo(args, "OR6ZFV-AA6TT-CKFFIW", "OAMUAJ-HLVKG-D3QJ5F")
	if err == nil {
		t.Error("Test Failed - QueryOrdersInfo() error", err)
	}
}

// TestGetTradesHistory API endpoint test
func TestGetTradesHistory(t *testing.T) {
	t.Parallel()
	args := GetTradesHistoryOptions{Trades: true, Start: "TMZEDR-VBJN2-NGY6DX", End: "TVRXG2-R62VE-RWP3UW"}
	_, err := k.GetTradesHistory(args)
	if err == nil {
		t.Error("Test Failed - GetTradesHistory() error", err)
	}
}

// TestQueryTrades API endpoint test
func TestQueryTrades(t *testing.T) {
	t.Parallel()
	_, err := k.QueryTrades(true, "TMZEDR-VBJN2-NGY6DX", "TFLWIB-KTT7L-4TWR3L", "TDVRAH-2H6OS-SLSXRX")
	if err == nil {
		t.Error("Test Failed - QueryTrades() error", err)
	}
}

// TestOpenPositions API endpoint test
func TestOpenPositions(t *testing.T) {
	t.Parallel()
	_, err := k.OpenPositions(false)
	if err == nil {
		t.Error("Test Failed - OpenPositions() error", err)
	}
}

// TestGetLedgers API endpoint test
func TestGetLedgers(t *testing.T) {
	t.Parallel()
	args := GetLedgersOptions{Start: "LRUHXI-IWECY-K4JYGO", End: "L5NIY7-JZQJD-3J4M2V", Ofs: 15}
	_, err := k.GetLedgers(args)
	if err == nil {
		t.Error("Test Failed - GetLedgers() error", err)
	}
}

// TestQueryLedgers API endpoint test
func TestQueryLedgers(t *testing.T) {
	t.Parallel()
	_, err := k.QueryLedgers("LVTSFS-NHZVM-EXNZ5M")
	if err == nil {
		t.Error("Test Failed - QueryLedgers() error", err)
	}
}

// TestGetTradeVolume API endpoint test
func TestGetTradeVolume(t *testing.T) {
	t.Parallel()
	_, err := k.GetTradeVolume(true, "OAVY7T-MV5VK-KHDF5X")
	if err == nil {
		t.Error("Test Failed - GetTradeVolume() error", err)
	}
}

// TestAddOrder API endpoint test
func TestAddOrder(t *testing.T) {
	t.Parallel()
	args := AddOrderOptions{Oflags: "fcib"}
	_, err := k.AddOrder("XXBTZUSD",
		exchange.SellOrderSide.ToLower().ToString(), exchange.LimitOrderType.ToLower().ToString(),
		0.00000001, 0, 0, 0, &args)
	if err == nil {
		t.Error("Test Failed - AddOrder() error", err)
	}
}

// TestCancelExistingOrder API endpoint test
func TestCancelExistingOrder(t *testing.T) {
	t.Parallel()
	_, err := k.CancelExistingOrder("OAVY7T-MV5VK-KHDF5X")
	if err == nil {
		t.Error("Test Failed - CancelExistingOrder() error", err)
	}
}

func setFeeBuilder() *exchange.FeeBuilder {
	return &exchange.FeeBuilder{
		Amount:              1,
		FeeType:             exchange.CryptocurrencyTradeFee,
		Pair:                currency.NewPair(currency.XXBT, currency.ZUSD),
		PurchasePrice:       1,
		FiatCurrency:        currency.USD,
		BankTransactionType: exchange.WireTransfer,
	}
}

// TestGetFee logic test

// TestGetFeeByTypeOfflineTradeFee logic test
func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	var feeBuilder = setFeeBuilder()
	k.GetFeeByType(feeBuilder)
	if apiKey == "" || apiSecret == "" {
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
	k.SetDefaults()
	TestSetup(t)
	var feeBuilder = setFeeBuilder()

	if areTestAPIKeysSet() {
		// CryptocurrencyTradeFee Basic
		if resp, err := k.GetFee(feeBuilder); resp != float64(0.0026) || err != nil {
			t.Error(err)
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0.0026), resp)
		}

		// CryptocurrencyTradeFee High quantity
		feeBuilder = setFeeBuilder()
		feeBuilder.Amount = 1000
		feeBuilder.PurchasePrice = 1000
		if resp, err := k.GetFee(feeBuilder); resp != float64(2600) || err != nil {
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(2600), resp)
			t.Error(err)
		}

		// CryptocurrencyTradeFee IsMaker
		feeBuilder = setFeeBuilder()
		feeBuilder.IsMaker = true
		if resp, err := k.GetFee(feeBuilder); resp != float64(0.0016) || err != nil {
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0.0016), resp)
			t.Error(err)
		}

		// CryptocurrencyTradeFee Negative purchase price
		feeBuilder = setFeeBuilder()
		feeBuilder.PurchasePrice = -1000
		if resp, err := k.GetFee(feeBuilder); resp != float64(0) || err != nil {
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
			t.Error(err)
		}

		// InternationalBankDepositFee Basic
		feeBuilder = setFeeBuilder()
		feeBuilder.FeeType = exchange.InternationalBankDepositFee
		if resp, err := k.GetFee(feeBuilder); resp != float64(5) || err != nil {
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(5), resp)
			t.Error(err)
		}
	}

	// CyptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CyptocurrencyDepositFee
	feeBuilder.Pair.Base = currency.XXBT
	if resp, err := k.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(5), resp)
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := k.GetFee(feeBuilder); resp != float64(0.0005) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0.0005), resp)
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Invalid currency
	feeBuilder = setFeeBuilder()
	feeBuilder.Pair.Base = currency.NewCode("hello")
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := k.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.USD
	if resp, err := k.GetFee(feeBuilder); resp != float64(5) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(5), resp)
		t.Error(err)
	}
}

// TestFormatWithdrawPermissions logic test
func TestFormatWithdrawPermissions(t *testing.T) {
	k.SetDefaults()
	expectedResult := exchange.AutoWithdrawCryptoWithSetupText + " & " + exchange.WithdrawCryptoWith2FAText + " & " + exchange.AutoWithdrawFiatWithSetupText + " & " + exchange.WithdrawFiatWith2FAText

	withdrawPermissions := k.FormatWithdrawPermissions()

	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s", expectedResult, withdrawPermissions)
	}
}

// TestGetActiveOrders wrapper test
func TestGetActiveOrders(t *testing.T) {
	k.SetDefaults()
	TestSetup(t)

	var getOrdersRequest = exchange.GetOrdersRequest{
		OrderType: exchange.AnyOrderType,
	}

	_, err := k.GetActiveOrders(&getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get open orders: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

// TestGetOrderHistory wrapper test
func TestGetOrderHistory(t *testing.T) {
	k.SetDefaults()
	TestSetup(t)

	var getOrdersRequest = exchange.GetOrdersRequest{
		OrderType: exchange.AnyOrderType,
	}

	_, err := k.GetOrderHistory(&getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get order history: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------
func areTestAPIKeysSet() bool {
	return k.ValidateAPICredentials()
}

// TestSubmitOrder wrapper test
func TestSubmitOrder(t *testing.T) {
	k.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var p = currency.Pair{
		Delimiter: "",
		Base:      currency.XBT,
		Quote:     currency.CAD,
	}
	response, err := k.SubmitOrder(p, exchange.BuyOrderSide, exchange.LimitOrderType, 1, 10, "hi")
	if areTestAPIKeysSet() && (err != nil || !response.IsOrderPlaced) {
		t.Errorf("Order failed to be placed: %v", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

// TestCancelExchangeOrder wrapper test
func TestCancelExchangeOrder(t *testing.T) {
	k.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)

	var orderCancellation = &exchange.OrderCancellation{
		OrderID:       "1",
		WalletAddress: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		AccountID:     "1",
		CurrencyPair:  currencyPair,
	}

	err := k.CancelOrder(orderCancellation)
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}
}

// TestCancelAllExchangeOrders wrapper test
func TestCancelAllExchangeOrders(t *testing.T) {
	k.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)

	var orderCancellation = &exchange.OrderCancellation{
		OrderID:       "1",
		WalletAddress: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		AccountID:     "1",
		CurrencyPair:  currencyPair,
	}

	resp, err := k.CancelAllOrders(orderCancellation)

	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}

	if len(resp.OrderStatus) > 0 {
		t.Errorf("%v orders failed to cancel", len(resp.OrderStatus))
	}
}

// TestGetAccountInfo wrapper test
func TestGetAccountInfo(t *testing.T) {
	if apiKey != "" || apiSecret != "" || clientID != "" {
		_, err := k.GetAccountInfo()
		if err != nil {
			t.Error("Test Failed - GetAccountInfo() error", err)
		}
	} else {
		_, err := k.GetAccountInfo()
		if err == nil {
			t.Error("Test Failed - GetAccountInfo() error")
		}
	}
}

// TestModifyOrder wrapper test
func TestModifyOrder(t *testing.T) {
	_, err := k.ModifyOrder(&exchange.ModifyOrder{})
	if err == nil {
		t.Error("Test failed - ModifyOrder() error")
	}
}

// TestWithdraw wrapper test
func TestWithdraw(t *testing.T) {
	k.SetDefaults()
	TestSetup(t)
	var withdrawCryptoRequest = exchange.WithdrawRequest{
		Amount:        100,
		Currency:      currency.XXBT,
		Address:       "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		Description:   "donation",
		TradePassword: "Key",
	}

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	_, err := k.WithdrawCryptocurrencyFunds(&withdrawCryptoRequest)
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

// TestWithdrawFiat wrapper test
func TestWithdrawFiat(t *testing.T) {
	k.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = exchange.WithdrawRequest{
		Amount:        100,
		Currency:      currency.EUR,
		Address:       "",
		Description:   "donation",
		TradePassword: "someBank",
	}

	_, err := k.WithdrawFiatFunds(&withdrawFiatRequest)
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

// TestWithdrawInternationalBank wrapper test
func TestWithdrawInternationalBank(t *testing.T) {
	k.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = exchange.WithdrawRequest{
		Amount:        100,
		Currency:      currency.EUR,
		Address:       "",
		Description:   "donation",
		TradePassword: "someBank",
	}

	_, err := k.WithdrawFiatFundsToInternationalBank(&withdrawFiatRequest)
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

// TestGetDepositAddress wrapper test
func TestGetDepositAddress(t *testing.T) {
	if areTestAPIKeysSet() {
		_, err := k.GetDepositAddress(currency.BTC, "")
		if err != nil {
			t.Error("Test Failed - GetDepositAddress() error", err)
		}
	} else {
		_, err := k.GetDepositAddress(currency.BTC, "")
		if err == nil {
			t.Error("Test Failed - GetDepositAddress() error can not be nil")
		}
	}
}

// TestWithdrawStatus wrapper test
func TestWithdrawStatus(t *testing.T) {
	k.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() {
		_, err := k.WithdrawStatus(currency.BTC, "")
		if err != nil {
			t.Error("Test Failed - WithdrawStatus() error", err)
		}
	} else {
		_, err := k.WithdrawStatus(currency.BTC, "")
		if err == nil {
			t.Error("Test Failed - GetDepositAddress() error can not be nil", err)
		}
	}
}

// TestWithdrawCancel wrapper test
func TestWithdrawCancel(t *testing.T) {
	k.SetDefaults()
	TestSetup(t)
	_, err := k.WithdrawCancel(currency.BTC, "")
	if areTestAPIKeysSet() && err == nil {
		t.Error("Test Failed - WithdrawCancel() error cannot be nil")
	} else if !areTestAPIKeysSet() && err == nil {
		t.Errorf("Test Failed - WithdrawCancel() error - expecting an error when no keys are set but received nil")
	}
}

// ---------------------------- Websocket tests -----------------------------------------

// TestOrderbookBufferReset websocket test
func TestOrderbookBufferReset(t *testing.T) {
	if k.Name == "" {
		k.SetDefaults()
		TestSetup(t)
	}
	if !k.Websocket.IsEnabled() {
		t.Skip("Websocket not enabled, skipping")
	}
	if k.WebsocketConn == nil {
		k.Websocket.Connect()
	}
	var obUpdates []string
	obpartial := `[0,{"as":[["5541.30000","2.50700000","0"]],"bs":[["5541.20000","1.52900000","0"]]}]`
	for i := 1; i < orderbookBufferLimit+2; i++ {
		obUpdates = append(obUpdates, fmt.Sprintf(`[0,{"a":[["5541.30000","2.50700000","%v"]],"b":[["5541.30000","1.00000000","%v"]]}]`, i, i))
	}
	k.Websocket.DataHandler = make(chan interface{}, 10)
	var dataResponse WebsocketDataResponse
	err := common.JSONDecode([]byte(obpartial), &dataResponse)
	if err != nil {
		t.Errorf("Could not parse, %v", err)
	}
	obData := dataResponse[1].(map[string]interface{})
	channelData := WebsocketChannelData{
		ChannelID:    0,
		Subscription: "orderbook",
		Pair:         currency.NewPairWithDelimiter("XBT", "USD", "/"),
	}

	k.wsProcessOrderBookPartial(
		&channelData,
		obData,
	)

	for i := 0; i < len(obUpdates); i++ {
		err = common.JSONDecode([]byte(obUpdates[i]), &dataResponse)
		if err != nil {
			t.Errorf("Could not parse, %v", err)
		}
		obData = dataResponse[1].(map[string]interface{})
		if i < len(obUpdates)-1 {
			k.wsProcessOrderBookBuffer(&channelData, obData)
		} else if i == len(obUpdates)-1 {
			k.wsProcessOrderBookUpdate(&channelData)
			k.wsProcessOrderBookBuffer(&channelData, obData)
			if len(orderbookBuffer[channelData.ChannelID]) != 1 {
				t.Error("Buffer should have 1 entry after being reset")
			}
		}
	}
}

// TestOrderbookBufferReset websocket test
func TestOrderBookOutOfOrder(t *testing.T) {
	if k.Name == "" {
		k.SetDefaults()
		TestSetup(t)
	}
	if !k.Websocket.IsEnabled() {
		t.Skip("Websocket not enabled, skipping")
	}
	if k.WebsocketConn == nil {
		k.Websocket.Connect()
	}
	obpartial := `[0,{"as":[["5541.30000","2.50700000","0"]],"bs":[["5541.20000","1.52900000","5"]]}]`
	obupdate1 := `[0,{"a":[["5541.30000","0.00000000","1"]],"b":[["5541.30000","0.00000000","3"]]}]`
	obupdate2 := `[0,{"a":[["5541.30000","2.50700000","2"]],"b":[["5541.30000","0.00000000","1"]]}]`

	k.Websocket.DataHandler = make(chan interface{}, 10)
	var dataResponse WebsocketDataResponse
	err := common.JSONDecode([]byte(obpartial), &dataResponse)
	if err != nil {
		t.Errorf("Could not parse, %v", err)
	}
	obData := dataResponse[1].(map[string]interface{})
	channelData := WebsocketChannelData{
		ChannelID:    0,
		Subscription: "orderbook",
		Pair:         currency.NewPairWithDelimiter("XBT", "USD", "/"),
	}

	k.wsProcessOrderBookPartial(
		&channelData,
		obData,
	)

	err = common.JSONDecode([]byte(obupdate1), &dataResponse)
	if err != nil {
		t.Errorf("Could not parse, %v", err)
	}
	obData = dataResponse[1].(map[string]interface{})
	k.wsProcessOrderBookBuffer(&channelData, obData)

	err = common.JSONDecode([]byte(obupdate2), &dataResponse)
	if err != nil {
		t.Errorf("Could not parse, %v", err)
	}
	obData = dataResponse[1].(map[string]interface{})
	k.wsProcessOrderBookBuffer(&channelData, obData)

	err = k.wsProcessOrderBookUpdate(&channelData)
	if !strings.Contains(err.Error(), "orderbook update out of order") {
		t.Error("Expected out of order orderbook error")
	}
}

// TestSubscribeToChannel websocket test
func TestSubscribeToChannel(t *testing.T) {
	if k.Name == "" {
		k.SetDefaults()
		TestSetup(t)
	}
	if !k.Websocket.IsEnabled() {
		t.Skip("Websocket not enabled, skipping")
	}
	if k.WebsocketConn == nil {
		k.Websocket.Connect()
	}

	err := k.WsSubscribeToChannel("ticker", []string{"XTZ/USD"}, 1)
	if err != nil {
		t.Error(err)
	}
}

// TestSubscribeToNonExistentChannel websocket test
func TestSubscribeToNonExistentChannel(t *testing.T) {
	if k.Name == "" {
		k.SetDefaults()
		TestSetup(t)
	}
	if !k.Websocket.IsEnabled() {
		t.Skip("Websocket not enabled, skipping")
	}
	if k.WebsocketConn == nil {
		k.Websocket.Connect()
	}
	err := k.WsSubscribeToChannel("ticker", []string{"pewdiepie"}, 1)
	if err != nil {
		t.Error(err)
	}
	subscriptionError := false
	for i := 0; i < 7; i++ {
		response := <-k.Websocket.DataHandler
		if err, ok := response.(error); ok && err != nil {
			subscriptionError = true
			break
		}
	}
	if !subscriptionError {
		t.Error("Expected error")
	}
}

// TestSubscribeUnsubscribeToChannel websocket test
func TestSubscribeUnsubscribeToChannel(t *testing.T) {
	if k.Name == "" {
		k.SetDefaults()
		TestSetup(t)
	}
	if !k.Websocket.IsEnabled() {
		t.Skip("Websocket not enabled, skipping")
	}
	if k.WebsocketConn == nil {
		k.Websocket.Connect()
	}
	err := k.WsSubscribeToChannel("ticker", []string{"XRP/JPY"}, 1)
	if err != nil {
		t.Error(err)
	}
	err = k.WsUnsubscribeToChannel("ticker", []string{"XRP/JPY"}, 2)
	if err != nil {
		t.Error(err)
	}
}

// TestUnsubscribeWithoutSubscription websocket test
func TestUnsubscribeWithoutSubscription(t *testing.T) {
	if k.Name == "" {
		k.SetDefaults()
		TestSetup(t)
	}
	if !k.Websocket.IsEnabled() {
		t.Skip("Websocket not enabled, skipping")
	}
	if k.WebsocketConn == nil {
		k.Websocket.Connect()
	}
	err := k.WsUnsubscribeToChannel("ticker", []string{"QTUM/EUR"}, 3)
	if err != nil {
		t.Error(err)
	}
	unsubscriptionError := false
	for i := 0; i < 5; i++ {
		response := <-k.Websocket.DataHandler
		t.Log(response)
		if err, ok := response.(error); ok && err != nil {
			if err.Error() == "requestID: '3'. Error: Subscription Not Found" {
				unsubscriptionError = true
				break
			}
		}
	}
	if !unsubscriptionError {
		t.Error("Expected error")
	}
}

// TestUnsubscribeWithChannelID websocket test
func TestUnsubscribeWithChannelID(t *testing.T) {
	if k.Name == "" {
		k.SetDefaults()
		TestSetup(t)
	}
	if !k.Websocket.IsEnabled() {
		t.Skip("Websocket not enabled, skipping")
	}
	if k.WebsocketConn == nil {
		k.Websocket.Connect()
	}
	err := k.WsUnsubscribeToChannelByChannelID(100)
	if err != nil {
		t.Error(err)
	}
	unsubscriptionError := false
	for i := 0; i < 5; i++ {
		response := <-k.Websocket.DataHandler
		if err, ok := response.(error); ok && err != nil {
			if err.Error() == "Not subscribed to the requested channelID" {
				unsubscriptionError = true
				break
			}
		}
	}
	if !unsubscriptionError {
		t.Error("Expected error")
	}
}

// TestUnsubscribeFromNonExistentChannel websocket test
func TestUnsubscribeFromNonExistentChannel(t *testing.T) {
	if k.Name == "" {
		k.SetDefaults()
		TestSetup(t)
	}
	if !k.Websocket.IsEnabled() {
		t.Skip("Websocket not enabled, skipping")
	}
	if k.WebsocketConn == nil {
		k.Websocket.Connect()
	}
	err := k.WsUnsubscribeToChannel("ticker", []string{"tseries"}, 0)
	if err != nil {
		t.Error(err)
	}
	unsubscriptionError := false
	for i := 0; i < 5; i++ {
		response := <-k.Websocket.DataHandler
		if err, ok := response.(error); ok && err != nil {
			if err.Error() == "Currency pair not in ISO 4217-A3 format tseries" {
				unsubscriptionError = true
				break
			}
		}
	}
	if !unsubscriptionError {
		t.Error("Expected error")
	}
}
