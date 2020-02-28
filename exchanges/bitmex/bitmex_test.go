package bitmex

import (
	"log"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// Please supply your own keys here for due diligence testing
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var b Bitmex

func TestMain(m *testing.M) {
	b.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal("Bitmex load config error", err)
	}
	bitmexConfig, err := cfg.GetExchangeConfig("Bitmex")
	if err != nil {
		log.Fatal("Bitmex Setup() init error")
	}

	bitmexConfig.API.AuthenticatedSupport = true
	bitmexConfig.API.AuthenticatedWebsocketSupport = true
	bitmexConfig.API.Credentials.Key = apiKey
	bitmexConfig.API.Credentials.Secret = apiSecret

	err = b.Setup(bitmexConfig)
	if err != nil {
		log.Fatal("Bitmex setup error", err)
	}
	os.Exit(m.Run())
}

func TestStart(t *testing.T) {
	var testWg sync.WaitGroup
	b.Start(&testWg)
	testWg.Wait()
}

func TestGetUrgentAnnouncement(t *testing.T) {
	_, err := b.GetUrgentAnnouncement()
	if err == nil {
		t.Error("GetUrgentAnnouncement() Expected error")
	}
}

func TestGetAPIKeys(t *testing.T) {
	_, err := b.GetAPIKeys()
	if err == nil {
		t.Error("GetAPIKeys() Expected error")
	}
}

func TestRemoveAPIKey(t *testing.T) {
	_, err := b.RemoveAPIKey(APIKeyParams{APIKeyID: "1337"})
	if err == nil {
		t.Error("RemoveAPIKey() Expected error")
	}
}

func TestDisableAPIKey(t *testing.T) {
	_, err := b.DisableAPIKey(APIKeyParams{APIKeyID: "1337"})
	if err == nil {
		t.Error("DisableAPIKey() Expected error")
	}
}

func TestEnableAPIKey(t *testing.T) {
	_, err := b.EnableAPIKey(APIKeyParams{APIKeyID: "1337"})
	if err == nil {
		t.Error("EnableAPIKey() Expected error")
	}
}

func TestGetTrollboxMessages(t *testing.T) {
	_, err := b.GetTrollboxMessages(ChatGetParams{Count: 5})
	if err != nil {
		t.Error("GetTrollboxMessages() error", err)
	}
}

func TestSendTrollboxMessage(t *testing.T) {
	_, err := b.SendTrollboxMessage(ChatSendParams{
		ChannelID: 1337,
		Message:   "Hello,World!"})
	if err == nil {
		t.Error("SendTrollboxMessage() Expected error")
	}
}

func TestGetTrollboxChannels(t *testing.T) {
	_, err := b.GetTrollboxChannels()
	if err != nil {
		t.Error("GetTrollboxChannels() error", err)
	}
}

func TestGetTrollboxConnectedUsers(t *testing.T) {
	_, err := b.GetTrollboxConnectedUsers()
	if err == nil {
		t.Error("GetTrollboxConnectedUsers() Expected error")
	}
}

func TestGetAccountExecutions(t *testing.T) {
	_, err := b.GetAccountExecutions(&GenericRequestParams{})
	if err == nil {
		t.Error("GetAccountExecutions() Expected error")
	}
}

func TestGetAccountExecutionTradeHistory(t *testing.T) {
	_, err := b.GetAccountExecutionTradeHistory(&GenericRequestParams{})
	if err == nil {
		t.Error("GetAccountExecutionTradeHistory() Expected error")
	}
}

func TestGetFundingHistory(t *testing.T) {
	_, err := b.GetFundingHistory()
	if err == nil {
		t.Error("GetFundingHistory() Expected error")
	}
}

func TestGetInstruments(t *testing.T) {
	_, err := b.GetInstruments(&GenericRequestParams{})
	if err != nil {
		t.Error("GetInstruments() error", err)
	}
}

func TestGetActiveInstruments(t *testing.T) {
	_, err := b.GetActiveInstruments(&GenericRequestParams{})
	if err != nil {
		t.Error("GetActiveInstruments() error", err)
	}
}

func TestGetActiveAndIndexInstruments(t *testing.T) {
	_, err := b.GetActiveAndIndexInstruments()
	if err != nil {
		t.Error("GetActiveAndIndexInstruments() error", err)
	}
}

func TestGetActiveIntervals(t *testing.T) {
	_, err := b.GetActiveIntervals()
	if err == nil {
		t.Error("GetActiveIntervals() Expected error")
	}
}

func TestGetCompositeIndex(t *testing.T) {
	_, err := b.GetCompositeIndex(&GenericRequestParams{})
	if err == nil {
		t.Error("GetCompositeIndex() Expected error")
	}
}

func TestGetIndices(t *testing.T) {
	_, err := b.GetIndices()
	if err != nil {
		t.Error("GetIndices() error", err)
	}
}

func TestGetInsuranceFundHistory(t *testing.T) {
	_, err := b.GetInsuranceFundHistory(&GenericRequestParams{})
	if err != nil {
		t.Error("GetInsuranceFundHistory() error", err)
	}
}

func TestGetLeaderboard(t *testing.T) {
	_, err := b.GetLeaderboard(LeaderboardGetParams{})
	if err != nil {
		t.Error("GetLeaderboard() error", err)
	}
}

func TestGetAliasOnLeaderboard(t *testing.T) {
	_, err := b.GetAliasOnLeaderboard()
	if err == nil {
		t.Error("GetAliasOnLeaderboard() Expected error")
	}
}

func TestGetLiquidationOrders(t *testing.T) {
	_, err := b.GetLiquidationOrders(&GenericRequestParams{})
	if err != nil {
		t.Error("GetLiquidationOrders() error", err)
	}
}

func TestGetCurrentNotifications(t *testing.T) {
	_, err := b.GetCurrentNotifications()
	if err == nil {
		t.Error("GetCurrentNotifications() Expected error")
	}
}

func TestAmendOrder(t *testing.T) {
	_, err := b.AmendOrder(&OrderAmendParams{})
	if err == nil {
		t.Error("AmendOrder() Expected error")
	}
}

func TestCreateOrder(t *testing.T) {
	_, err := b.CreateOrder(&OrderNewParams{Symbol: "XBTM15",
		Price:    219.0,
		ClOrdID:  "mm_bitmex_1a/oemUeQ4CAJZgP3fjHsA",
		OrderQty: 98})
	if err == nil {
		t.Error("CreateOrder() Expected error")
	}
}

func TestCancelOrders(t *testing.T) {
	_, err := b.CancelOrders(&OrderCancelParams{})
	if err == nil {
		t.Error("CancelOrders() Expected error")
	}
}

func TestCancelAllOrders(t *testing.T) {
	_, err := b.CancelAllExistingOrders(OrderCancelAllParams{})
	if err == nil {
		t.Error("CancelAllOrders(orderCancellation *order.Cancel) (order.CancelAllResponse, error)", err)
	}
}

func TestAmendBulkOrders(t *testing.T) {
	_, err := b.AmendBulkOrders(OrderAmendBulkParams{})
	if err == nil {
		t.Error("AmendBulkOrders() Expected error")
	}
}

func TestCreateBulkOrders(t *testing.T) {
	_, err := b.CreateBulkOrders(OrderNewBulkParams{})
	if err == nil {
		t.Error("CreateBulkOrders() Expected error")
	}
}

func TestCancelAllOrdersAfterTime(t *testing.T) {
	_, err := b.CancelAllOrdersAfterTime(OrderCancelAllAfterParams{})
	if err == nil {
		t.Error("CancelAllOrdersAfterTime() Expected error")
	}
}

func TestClosePosition(t *testing.T) {
	_, err := b.ClosePosition(OrderClosePositionParams{})
	if err == nil {
		t.Error("ClosePosition() Expected error")
	}
}

func TestGetOrderbook(t *testing.T) {
	_, err := b.GetOrderbook(OrderBookGetL2Params{Symbol: "XBT"})
	if err != nil {
		t.Error("GetOrderbook() error", err)
	}
}

func TestGetPositions(t *testing.T) {
	_, err := b.GetPositions(PositionGetParams{})
	if err == nil {
		t.Error("GetPositions() Expected error")
	}
}

func TestIsolatePosition(t *testing.T) {
	_, err := b.IsolatePosition(PositionIsolateMarginParams{Symbol: "XBT"})
	if err == nil {
		t.Error("IsolatePosition() Expected error")
	}
}

func TestLeveragePosition(t *testing.T) {
	_, err := b.LeveragePosition(PositionUpdateLeverageParams{})
	if err == nil {
		t.Error("LeveragePosition() Expected error")
	}
}

func TestUpdateRiskLimit(t *testing.T) {
	_, err := b.UpdateRiskLimit(PositionUpdateRiskLimitParams{})
	if err == nil {
		t.Error("UpdateRiskLimit() Expected error")
	}
}

func TestTransferMargin(t *testing.T) {
	_, err := b.TransferMargin(PositionTransferIsolatedMarginParams{})
	if err == nil {
		t.Error("TransferMargin() Expected error")
	}
}

func TestGetQuotesByBuckets(t *testing.T) {
	_, err := b.GetQuotesByBuckets(&QuoteGetBucketedParams{})
	if err == nil {
		t.Error("GetQuotesByBuckets() Expected error")
	}
}

func TestGetSettlementHistory(t *testing.T) {
	_, err := b.GetSettlementHistory(&GenericRequestParams{})
	if err != nil {
		t.Error("GetSettlementHistory() error", err)
	}
}

func TestGetStats(t *testing.T) {
	_, err := b.GetStats()
	if err != nil {
		t.Error("GetStats() error", err)
	}
}

func TestGetStatsHistorical(t *testing.T) {
	_, err := b.GetStatsHistorical()
	if err != nil {
		t.Error("GetStatsHistorical() error", err)
	}
}

func TestGetStatSummary(t *testing.T) {
	_, err := b.GetStatSummary()
	if err != nil {
		t.Error("GetStatSummary() error", err)
	}
}

func TestGetTrade(t *testing.T) {
	_, err := b.GetTrade(&GenericRequestParams{
		Symbol:    "XBTUSD",
		StartTime: time.Now().Format(time.RFC3339),
		Reverse:   true})
	if err != nil {
		t.Error("GetTrade() error", err)
	}
}

func TestGetPreviousTrades(t *testing.T) {
	_, err := b.GetPreviousTrades(&TradeGetBucketedParams{})
	if err == nil {
		t.Error("GetPreviousTrades() Expected error")
	}
}

func setFeeBuilder() *exchange.FeeBuilder {
	return &exchange.FeeBuilder{
		Amount:        1,
		FeeType:       exchange.CryptocurrencyTradeFee,
		Pair:          currency.NewPair(currency.BTC, currency.LTC),
		PurchasePrice: 1,
	}
}

// TestGetFeeByTypeOfflineTradeFee logic test
func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	var feeBuilder = setFeeBuilder()
	b.GetFeeByType(feeBuilder)
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
	// CryptocurrencyTradeFee Basic
	if resp, err := b.GetFee(feeBuilder); resp != float64(0.00075) || err != nil {
		t.Error(err)
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0.00075), resp)
	}

	// CryptocurrencyTradeFee High quantity
	feeBuilder = setFeeBuilder()
	feeBuilder.Amount = 1000
	feeBuilder.PurchasePrice = 1000
	if resp, err := b.GetFee(feeBuilder); resp != float64(750) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(750), resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee IsMaker
	feeBuilder = setFeeBuilder()
	feeBuilder.IsMaker = true
	if resp, err := b.GetFee(feeBuilder); resp != float64(0.0005) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0.0005), resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee Negative purchase price
	feeBuilder = setFeeBuilder()
	feeBuilder.PurchasePrice = -1000
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// CyptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CyptocurrencyDepositFee
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	feeBuilder.FiatCurrency = currency.HKD
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.HKD
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	expectedResult := exchange.AutoWithdrawCryptoWithAPIPermissionText + " & " + exchange.WithdrawCryptoWith2FAText +
		" & " + exchange.WithdrawCryptoWithEmailText + " & " + exchange.NoFiatWithdrawalsText
	withdrawPermissions := b.FormatWithdrawPermissions()
	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s", expectedResult, withdrawPermissions)
	}
}

func TestGetActiveOrders(t *testing.T) {
	var getOrdersRequest = order.GetOrdersRequest{
		OrderType: order.AnyType,
	}

	_, err := b.GetActiveOrders(&getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get open orders: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestGetOrderHistory(t *testing.T) {
	var getOrdersRequest = order.GetOrdersRequest{
		OrderType: order.AnyType,
		Currencies: []currency.Pair{currency.NewPair(currency.LTC,
			currency.BTC)},
	}

	_, err := b.GetOrderHistory(&getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get order history: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------
func areTestAPIKeysSet() bool {
	return b.ValidateAPICredentials()
}

func TestSubmitOrder(t *testing.T) {
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var orderSubmission = &order.Submit{
		Pair: currency.Pair{
			Base:  currency.XBT,
			Quote: currency.USD,
		},
		OrderSide: order.Buy,
		OrderType: order.Limit,
		Price:     1,
		Amount:    1,
		ClientID:  "meowOrder",
	}
	response, err := b.SubmitOrder(orderSubmission)
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
		OrderID:       "123456789012345678901234567890123456",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		CurrencyPair:  currencyPair,
	}

	err := b.CancelOrder(orderCancellation)
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
		OrderID:       "123456789012345678901234567890123456",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		CurrencyPair:  currencyPair,
	}

	resp, err := b.CancelAllOrders(orderCancellation)

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
	if areTestAPIKeysSet() {
		_, err := b.UpdateAccountInfo()
		if err != nil {
			t.Error("GetAccountInfo() error", err)
		}
	} else {
		_, err := b.UpdateAccountInfo()
		if err == nil {
			t.Error("GetAccountInfo() error")
		}
	}
}

func TestModifyOrder(t *testing.T) {
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	_, err := b.ModifyOrder(&order.Modify{OrderID: "1337"})
	if err == nil {
		t.Error("ModifyOrder() error")
	}
}

func TestWithdraw(t *testing.T) {
	withdrawCryptoRequest := withdraw.Request{
		Crypto: &withdraw.CryptoRequest{
			Address: core.BitcoinDonationAddress,
		},
		Amount:          -1,
		Currency:        currency.BTC,
		Description:     "WITHDRAW IT ALL",
		OneTimePassword: 000000,
	}

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	_, err := b.WithdrawCryptocurrencyFunds(&withdrawCryptoRequest)
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
	_, err := b.WithdrawFiatFunds(&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = withdraw.Request{}
	_, err := b.WithdrawFiatFundsToInternationalBank(&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	if areTestAPIKeysSet() {
		_, err := b.GetDepositAddress(currency.BTC, "")
		if err != nil {
			t.Error("GetDepositAddress() error", err)
		}
	} else {
		_, err := b.GetDepositAddress(currency.BTC, "")
		if err == nil {
			t.Error("GetDepositAddress() error cannot be nil")
		}
	}
}

// TestWsAuth dials websocket, sends login request.
func TestWsAuth(t *testing.T) {
	if !b.Websocket.IsEnabled() && !b.API.AuthenticatedWebsocketSupport || !areTestAPIKeysSet() {
		t.Skip(wshandler.WebsocketNotEnabled)
	}
	b.WebsocketConn = &wshandler.WebsocketConnection{
		ExchangeName:         b.Name,
		URL:                  b.Websocket.GetWebsocketURL(),
		Verbose:              b.Verbose,
		ResponseMaxLimit:     exchange.DefaultWebsocketResponseMaxLimit,
		ResponseCheckTimeout: exchange.DefaultWebsocketResponseCheckTimeout,
	}
	var dialer websocket.Dialer
	err := b.WebsocketConn.Dial(&dialer, http.Header{})
	if err != nil {
		t.Fatal(err)
	}
	b.Websocket.DataHandler = sharedtestvalues.GetWebsocketInterfaceChannelOverride()
	b.Websocket.TrafficAlert = sharedtestvalues.GetWebsocketStructChannelOverride()
	go b.wsHandleIncomingData()
	err = b.websocketSendAuth()
	if err != nil {
		t.Fatal(err)
	}
	timer := time.NewTimer(sharedtestvalues.WebsocketResponseDefaultTimeout)
	select {
	case resp := <-b.Websocket.DataHandler:
		if !resp.(WebsocketSubscribeResp).Success {
			t.Error("Expected successful subscription")
		}
	case <-timer.C:
		t.Error("Have not received a response")
	}
	timer.Stop()
}
