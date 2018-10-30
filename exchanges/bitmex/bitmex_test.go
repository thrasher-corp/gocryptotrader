package bitmex

import (
	"sync"
	"testing"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
)

// Please supply your own keys here for due diligence testing
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var b Bitmex

func TestSetDefaults(t *testing.T) {
	b.SetDefaults()
}

func TestSetup(t *testing.T) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	bitmexConfig, err := cfg.GetExchangeConfig("Bitmex")
	if err != nil {
		t.Error("Test Failed - Bitmex Setup() init error")
	}

	bitmexConfig.API.AuthenticatedSupport = true
	bitmexConfig.API.Credentials.Key = apiKey
	bitmexConfig.API.Credentials.Secret = apiSecret

	b.Setup(bitmexConfig)
}

func TestStart(t *testing.T) {
	var testWg sync.WaitGroup
	b.Start(&testWg)
	testWg.Wait()
}

func TestGetUrgentAnnouncement(t *testing.T) {
	_, err := b.GetUrgentAnnouncement()
	if err == nil {
		t.Error("test failed - GetUrgentAnnouncement() error", err)
	}
}

func TestGetAPIKeys(t *testing.T) {
	_, err := b.GetAPIKeys()
	if err == nil {
		t.Error("test failed - GetAPIKeys() error", err)
	}
}

func TestRemoveAPIKey(t *testing.T) {
	_, err := b.RemoveAPIKey(APIKeyParams{APIKeyID: "1337"})
	if err == nil {
		t.Error("test failed - RemoveAPIKey() error", err)
	}
}

func TestDisableAPIKey(t *testing.T) {
	_, err := b.DisableAPIKey(APIKeyParams{APIKeyID: "1337"})
	if err == nil {
		t.Error("test failed - DisableAPIKey() error", err)
	}
}

func TestEnableAPIKey(t *testing.T) {
	_, err := b.EnableAPIKey(APIKeyParams{APIKeyID: "1337"})
	if err == nil {
		t.Error("test failed - EnableAPIKey() error", err)
	}
}

func TestGetTrollboxMessages(t *testing.T) {
	_, err := b.GetTrollboxMessages(ChatGetParams{Count: 5})
	if err != nil {
		t.Error("test failed - GetTrollboxMessages() error", err)
	}
}

func TestSendTrollboxMessage(t *testing.T) {
	_, err := b.SendTrollboxMessage(ChatSendParams{
		ChannelID: 1337,
		Message:   "Hello,World!"})
	if err == nil {
		t.Error("test failed - SendTrollboxMessage() error", err)
	}
}

func TestGetTrollboxChannels(t *testing.T) {
	_, err := b.GetTrollboxChannels()
	if err != nil {
		t.Error("test failed - GetTrollboxChannels() error", err)
	}
}

func TestGetTrollboxConnectedUsers(t *testing.T) {
	_, err := b.GetTrollboxConnectedUsers()
	if err == nil {
		t.Error("test failed - GetTrollboxConnectedUsers() error", err)
	}
}

func TestGetAccountExecutions(t *testing.T) {
	_, err := b.GetAccountExecutions(&GenericRequestParams{})
	if err == nil {
		t.Error("test failed - GetAccountExecutions() error", err)
	}
}

func TestGetAccountExecutionTradeHistory(t *testing.T) {
	_, err := b.GetAccountExecutionTradeHistory(&GenericRequestParams{})
	if err == nil {
		t.Error("test failed - GetAccountExecutionTradeHistory() error", err)
	}
}

func TestGetFundingHistory(t *testing.T) {
	_, err := b.GetFundingHistory()
	if err == nil {
		t.Error("test failed - GetFundingHistory() error", err)
	}
}

func TestGetInstruments(t *testing.T) {
	_, err := b.GetInstruments(&GenericRequestParams{})
	if err != nil {
		t.Error("test failed - GetInstruments() error", err)
	}
}

func TestGetActiveInstruments(t *testing.T) {
	_, err := b.GetActiveInstruments(&GenericRequestParams{})
	if err != nil {
		t.Error("test failed - GetActiveInstruments() error", err)
	}
}

func TestGetActiveAndIndexInstruments(t *testing.T) {
	_, err := b.GetActiveAndIndexInstruments()
	if err != nil {
		t.Error("test failed - GetActiveAndIndexInstruments() error", err)
	}
}

func TestGetActiveIntervals(t *testing.T) {
	_, err := b.GetActiveIntervals()
	if err == nil {
		t.Error("test failed - GetActiveIntervals() error", err)
	}
}

func TestGetCompositeIndex(t *testing.T) {
	_, err := b.GetCompositeIndex(&GenericRequestParams{})
	if err == nil {
		t.Error("test failed - GetCompositeIndex() error", err)
	}
}

func TestGetIndices(t *testing.T) {
	_, err := b.GetIndices()
	if err != nil {
		t.Error("test failed - GetIndices() error", err)
	}
}

func TestGetInsuranceFundHistory(t *testing.T) {
	_, err := b.GetInsuranceFundHistory(&GenericRequestParams{})
	if err != nil {
		t.Error("test failed - GetInsuranceFundHistory() error", err)
	}
}

func TestGetLeaderboard(t *testing.T) {
	_, err := b.GetLeaderboard(LeaderboardGetParams{})
	if err != nil {
		t.Error("test failed - GetLeaderboard() error", err)
	}
}

func TestGetAliasOnLeaderboard(t *testing.T) {
	_, err := b.GetAliasOnLeaderboard()
	if err == nil {
		t.Error("test failed - GetAliasOnLeaderboard() error", err)
	}
}

func TestGetLiquidationOrders(t *testing.T) {
	_, err := b.GetLiquidationOrders(&GenericRequestParams{})
	if err != nil {
		t.Error("test failed - GetLiquidationOrders() error", err)
	}
}

func TestGetCurrentNotifications(t *testing.T) {
	_, err := b.GetCurrentNotifications()
	if err == nil {
		t.Error("test failed - GetCurrentNotifications() error", err)
	}
}

func TestAmendOrder(t *testing.T) {
	_, err := b.AmendOrder(&OrderAmendParams{})
	if err == nil {
		t.Error("test failed - AmendOrder() error", err)
	}
}

func TestCreateOrder(t *testing.T) {
	_, err := b.CreateOrder(&OrderNewParams{Symbol: "XBTM15",
		Price:    219.0,
		ClOrdID:  "mm_bitmex_1a/oemUeQ4CAJZgP3fjHsA",
		OrderQty: 98})
	if err == nil {
		t.Error("test failed - CreateOrder() error", err)
	}
}

func TestCancelOrders(t *testing.T) {
	_, err := b.CancelOrders(&OrderCancelParams{})
	if err == nil {
		t.Error("test failed - CancelOrders() error", err)
	}
}

func TestCancelAllOrders(t *testing.T) {
	_, err := b.CancelAllExistingOrders(OrderCancelAllParams{})
	if err == nil {
		t.Error("test failed - CancelAllOrders(orderCancellation *exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error)", err)
	}
}

func TestAmendBulkOrders(t *testing.T) {
	_, err := b.AmendBulkOrders(OrderAmendBulkParams{})
	if err == nil {
		t.Error("test failed - AmendBulkOrders() error", err)
	}
}

func TestCreateBulkOrders(t *testing.T) {
	_, err := b.CreateBulkOrders(OrderNewBulkParams{})
	if err == nil {
		t.Error("test failed - CreateBulkOrders() error", err)
	}
}

func TestCancelAllOrdersAfterTime(t *testing.T) {
	_, err := b.CancelAllOrdersAfterTime(OrderCancelAllAfterParams{})
	if err == nil {
		t.Error("test failed - CancelAllOrdersAfterTime() error", err)
	}
}

func TestClosePosition(t *testing.T) {
	_, err := b.ClosePosition(OrderClosePositionParams{})
	if err == nil {
		t.Error("test failed - ClosePosition() error", err)
	}
}

func TestGetOrderbook(t *testing.T) {
	_, err := b.GetOrderbook(OrderBookGetL2Params{Symbol: "XBT"})
	if err != nil {
		t.Error("test failed - GetOrderbook() error", err)
	}
}

func TestGetPositions(t *testing.T) {
	_, err := b.GetPositions(PositionGetParams{})
	if err == nil {
		t.Error("test failed - GetPositions() error", err)
	}
}

func TestIsolatePosition(t *testing.T) {
	_, err := b.IsolatePosition(PositionIsolateMarginParams{Symbol: "XBT"})
	if err == nil {
		t.Error("test failed - IsolatePosition() error", err)
	}
}

func TestLeveragePosition(t *testing.T) {
	_, err := b.LeveragePosition(PositionUpdateLeverageParams{})
	if err == nil {
		t.Error("test failed - LeveragePosition() error", err)
	}
}

func TestUpdateRiskLimit(t *testing.T) {
	_, err := b.UpdateRiskLimit(PositionUpdateRiskLimitParams{})
	if err == nil {
		t.Error("test failed - UpdateRiskLimit() error", err)
	}
}

func TestTransferMargin(t *testing.T) {
	_, err := b.TransferMargin(PositionTransferIsolatedMarginParams{})
	if err == nil {
		t.Error("test failed - TransferMargin() error", err)
	}
}

func TestGetQuotesByBuckets(t *testing.T) {
	_, err := b.GetQuotesByBuckets(&QuoteGetBucketedParams{})
	if err == nil {
		t.Error("test failed - GetQuotesByBuckets() error", err)
	}
}

func TestGetSettlementHistory(t *testing.T) {
	_, err := b.GetSettlementHistory(&GenericRequestParams{})
	if err != nil {
		t.Error("test failed - GetSettlementHistory() error", err)
	}
}

func TestGetStats(t *testing.T) {
	_, err := b.GetStats()
	if err != nil {
		t.Error("test failed - GetStats() error", err)
	}
}

func TestGetStatsHistorical(t *testing.T) {
	_, err := b.GetStatsHistorical()
	if err != nil {
		t.Error("test failed - GetStatsHistorical() error", err)
	}
}

func TestGetStatSummary(t *testing.T) {
	_, err := b.GetStatSummary()
	if err != nil {
		t.Error("test failed - GetStatSummary() error", err)
	}
}

func TestGetTrade(t *testing.T) {
	_, err := b.GetTrade(&GenericRequestParams{
		Symbol:    "XBTUSD",
		StartTime: time.Now().Format(time.RFC3339),
		Reverse:   true})
	if err != nil {
		t.Error("test failed - GetTrade() error", err)
	}
}

func TestGetPreviousTrades(t *testing.T) {
	_, err := b.GetPreviousTrades(&TradeGetBucketedParams{})
	if err == nil {
		t.Error("test failed - GetPreviousTrades() error", err)
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
	b.SetDefaults()
	TestSetup(t)

	var feeBuilder = setFeeBuilder()
	// CryptocurrencyTradeFee Basic
	if resp, err := b.GetFee(feeBuilder); resp != float64(0.00075) || err != nil {
		t.Error(err)
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0.00075), resp)
	}

	// CryptocurrencyTradeFee High quantity
	feeBuilder = setFeeBuilder()
	feeBuilder.Amount = 1000
	feeBuilder.PurchasePrice = 1000
	if resp, err := b.GetFee(feeBuilder); resp != float64(750) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(750), resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee IsMaker
	feeBuilder = setFeeBuilder()
	feeBuilder.IsMaker = true
	if resp, err := b.GetFee(feeBuilder); resp != float64(0.0005) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0.0005), resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee Negative purchase price
	feeBuilder = setFeeBuilder()
	feeBuilder.PurchasePrice = -1000
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// CyptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CyptocurrencyDepositFee
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	feeBuilder.FiatCurrency = currency.HKD
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.HKD
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	b.SetDefaults()
	expectedResult := exchange.AutoWithdrawCryptoWithAPIPermissionText + " & " + exchange.WithdrawCryptoWith2FAText +
		" & " + exchange.WithdrawCryptoWithEmailText + " & " + exchange.NoFiatWithdrawalsText

	withdrawPermissions := b.FormatWithdrawPermissions()

	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s", expectedResult, withdrawPermissions)
	}
}

func TestGetActiveOrders(t *testing.T) {
	b.SetDefaults()
	TestSetup(t)

	var getOrdersRequest = exchange.GetOrdersRequest{
		OrderType: exchange.AnyOrderType,
	}

	_, err := b.GetActiveOrders(&getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get open orders: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestGetOrderHistory(t *testing.T) {
	b.SetDefaults()
	TestSetup(t)

	var getOrdersRequest = exchange.GetOrdersRequest{
		OrderType: exchange.AnyOrderType,
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
	b.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var p = currency.Pair{
		Delimiter: "",
		Base:      currency.XBT,
		Quote:     currency.USD,
	}
	response, err := b.SubmitOrder(p, exchange.BuyOrderSide, exchange.LimitOrderType, 1, 1, "clientId")
	if areTestAPIKeysSet() && (err != nil || !response.IsOrderPlaced) {
		t.Errorf("Order failed to be placed: %v", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	b.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)

	var orderCancellation = &exchange.OrderCancellation{
		OrderID:       "123456789012345678901234567890123456",
		WalletAddress: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
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
	b.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)

	var orderCancellation = &exchange.OrderCancellation{
		OrderID:       "123456789012345678901234567890123456",
		WalletAddress: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
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

	if len(resp.OrderStatus) > 0 {
		t.Errorf("%v orders failed to cancel", len(resp.OrderStatus))
	}
}

func TestGetAccountInfo(t *testing.T) {
	if apiKey != "" || apiSecret != "" {
		_, err := b.GetAccountInfo()
		if err != nil {
			t.Error("Test Failed - GetAccountInfo() error", err)
		}
	} else {
		_, err := b.GetAccountInfo()
		if err == nil {
			t.Error("Test Failed - GetAccountInfo() error")
		}
	}
}

func TestModifyOrder(t *testing.T) {
	_, err := b.ModifyOrder(&exchange.ModifyOrder{OrderID: "1337"})
	if err == nil {
		t.Error("Test Failed - ModifyOrder() error")
	}
}

func TestWithdraw(t *testing.T) {
	b.SetDefaults()
	TestSetup(t)
	var withdrawCryptoRequest = exchange.WithdrawRequest{
		Amount:          100,
		Currency:        currency.XBT,
		Address:         "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
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
	b.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = exchange.WithdrawRequest{}

	_, err := b.WithdrawFiatFunds(&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	b.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = exchange.WithdrawRequest{}

	_, err := b.WithdrawFiatFundsToInternationalBank(&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	if areTestAPIKeysSet() {
		_, err := b.GetDepositAddress(currency.BTC, "")
		if err != nil {
			t.Error("Test Failed - GetDepositAddress() error", err)
		}
	} else {
		_, err := b.GetDepositAddress(currency.BTC, "")
		if err == nil {
			t.Error("Test Failed - GetDepositAddress() error cannot be nil")
		}
	}
}
