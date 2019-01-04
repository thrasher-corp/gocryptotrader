package bitmex

import (
	"sync"
	"testing"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/currency/symbol"
	"github.com/thrasher-/gocryptotrader/exchanges"

	"github.com/thrasher-/gocryptotrader/config"
)

// Please supply your own keys here for due diligence testing
const (
	testAPIKey              = ""
	testAPISecret           = ""
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

	bitmexConfig.AuthenticatedAPISupport = true
	bitmexConfig.APIKey = testAPIKey
	bitmexConfig.APISecret = testAPISecret

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
	_, err := b.GetAccountExecutions(GenericRequestParams{})
	if err == nil {
		t.Error("test failed - GetAccountExecutions() error", err)
	}
}

func TestGetAccountExecutionTradeHistory(t *testing.T) {
	_, err := b.GetAccountExecutionTradeHistory(GenericRequestParams{})
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
	_, err := b.GetInstruments(GenericRequestParams{})
	if err != nil {
		t.Error("test failed - GetInstruments() error", err)
	}
}

func TestGetActiveInstruments(t *testing.T) {
	_, err := b.GetActiveInstruments(GenericRequestParams{})
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
	_, err := b.GetCompositeIndex(GenericRequestParams{})
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
	_, err := b.GetInsuranceFundHistory(GenericRequestParams{})
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
	_, err := b.GetLiquidationOrders(GenericRequestParams{})
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

func TestGetOrders(t *testing.T) {
	_, err := b.GetOrders(GenericRequestParams{})
	if err == nil {
		t.Error("test failed - GetOrders() error", err)
	}
}

func TestAmendOrder(t *testing.T) {
	_, err := b.AmendOrder(OrderAmendParams{})
	if err == nil {
		t.Error("test failed - AmendOrder() error", err)
	}
}

func TestCreateOrder(t *testing.T) {
	_, err := b.CreateOrder(OrderNewParams{Symbol: "XBTM15",
		Price:    219.0,
		ClOrdID:  "mm_bitmex_1a/oemUeQ4CAJZgP3fjHsA",
		OrderQty: 98})
	if err == nil {
		t.Error("test failed - CreateOrder() error", err)
	}
}

func TestCancelOrders(t *testing.T) {
	_, err := b.CancelOrders(OrderCancelParams{})
	if err == nil {
		t.Error("test failed - CancelOrders() error", err)
	}
}

func TestCancelAllOrders(t *testing.T) {
	_, err := b.CancelAllExistingOrders(OrderCancelAllParams{})
	if err == nil {
		t.Error("test failed - CancelAllOrders(orderCancellation exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error)", err)
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
	_, err := b.GetQuotesByBuckets(QuoteGetBucketedParams{})
	if err == nil {
		t.Error("test failed - GetQuotesByBuckets() error", err)
	}
}

func TestGetSettlementHistory(t *testing.T) {
	_, err := b.GetSettlementHistory(GenericRequestParams{})
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
	_, err := b.GetTrade(GenericRequestParams{
		Symbol:    "XBTUSD",
		StartTime: time.Now().Format(time.RFC3339),
		Reverse:   true})
	if err != nil {
		t.Error("test failed - GetTrade() error", err)
	}
}

func TestGetPreviousTrades(t *testing.T) {
	_, err := b.GetPreviousTrades(TradeGetBucketedParams{})
	if err == nil {
		t.Error("test failed - GetPreviousTrades() error", err)
	}
}

func setFeeBuilder() exchange.FeeBuilder {
	return exchange.FeeBuilder{
		Amount:         1,
		Delimiter:      "",
		FeeType:        exchange.CryptocurrencyTradeFee,
		FirstCurrency:  symbol.BTC,
		SecondCurrency: symbol.LTC,
		IsMaker:        false,
		PurchasePrice:  1,
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
	feeBuilder.CurrencyItem = symbol.HKD
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.CurrencyItem = symbol.HKD
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	// Arrange
	b.SetDefaults()
	expectedResult := exchange.AutoWithdrawCryptoWithAPIPermissionText + " & " + exchange.WithdrawCryptoWith2FAText + " & " + exchange.WithdrawCryptoWithEmailText
	// Act
	withdrawPermissions := b.FormatWithdrawPermissions()
	// Assert
	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s", expectedResult, withdrawPermissions)
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------
func areTestAPIKeysSet() bool {
	if b.APIKey != "" && b.APIKey != "Key" &&
		b.APISecret != "" && b.APISecret != "Secret" {
		return true
	}
	return false
}

func TestSubmitOrder(t *testing.T) {
	b.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var p = pair.CurrencyPair{
		Delimiter:      "",
		FirstCurrency:  symbol.XBT,
		SecondCurrency: symbol.USD,
	}
	response, err := b.SubmitOrder(p, exchange.Buy, exchange.Market, 1, 1, "clientId")
	if areTestAPIKeysSet() && (err != nil || !response.IsOrderPlaced) {
		t.Errorf("Order failed to be placed: %v", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	// Arrange
	b.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	currencyPair := pair.NewCurrencyPair(symbol.LTC, symbol.BTC)

	var orderCancellation = exchange.OrderCancellation{
		OrderID:       "123456789012345678901234567890123456",
		WalletAddress: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		AccountID:     "1",
		CurrencyPair:  currencyPair,
	}

	// Act
	err := b.CancelOrder(orderCancellation)

	// Assert
	if !areTestAPIKeysSet() && err == nil {
		t.Errorf("Expecting an error when no keys are set: %v", err)
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	// Arrange
	b.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	currencyPair := pair.NewCurrencyPair(symbol.LTC, symbol.BTC)

	var orderCancellation = exchange.OrderCancellation{
		OrderID:       "123456789012345678901234567890123456",
		WalletAddress: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		AccountID:     "1",
		CurrencyPair:  currencyPair,
	}

	// Act
	resp, err := b.CancelAllOrders(orderCancellation)

	// Assert
	if !areTestAPIKeysSet() && err == nil {
		t.Errorf("Expecting an error when no keys are set: %v", err)
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}

	if len(resp.OrderStatus) > 0 {
		t.Errorf("%v orders failed to cancel", len(resp.OrderStatus))
	}
}

func TestGetAccountInfo(t *testing.T) {
	if testAPIKey != "" || testAPISecret != "" {
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
	_, err := b.ModifyOrder(exchange.ModifyOrder{OrderID: "1337"})
	if err == nil {
		t.Error("Test Failed - ModifyOrder() error")
	}
}

func TestWithdraw(t *testing.T) {
	b.SetDefaults()
	TestSetup(t)
	var withdrawCryptoRequest = exchange.WithdrawRequest{
		Amount:          100,
		Currency:        "XBt",
		Address:         "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		Description:     "WITHDRAW IT ALL",
		OneTimePassword: 000000,
	}

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	_, err := b.WithdrawCryptocurrencyFunds(withdrawCryptoRequest)
	if !areTestAPIKeysSet() && err == nil {
		t.Errorf("Expecting an error when no keys are set: %v", err)
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

	var withdrawFiatRequest = exchange.WithdrawRequest{
		Amount:       100,
		Currency:     symbol.BTC,
		Description:  "WITHDRAW IT ALL",
		BankAddress:  "123 Fake St",
		BankCity:     "Tarry Town",
		BankCountry:  "Hyrule",
		BankName:     "Federal Reserve Bank",
		WireCurrency: symbol.AUD,
		SwiftCode:    "Taylor",
	}

	_, err := b.WithdrawFiatFunds(withdrawFiatRequest)
	if err != common.ErrNotYetImplemented {
		t.Errorf("Expected '%v', recieved: '%v'", common.ErrNotYetImplemented, err)
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	b.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = exchange.WithdrawRequest{
		Amount:       100,
		Currency:     symbol.BTC,
		Description:  "WITHDRAW IT ALL",
		BankAddress:  "123 Fake St",
		BankCity:     "Tarry Town",
		BankCountry:  "Hyrule",
		BankName:     "Federal Reserve Bank",
		WireCurrency: symbol.AUD,
		SwiftCode:    "Taylor",
	}

	_, err := b.WithdrawFiatFundsToInternationalBank(withdrawFiatRequest)
	if err != common.ErrNotYetImplemented {
		t.Errorf("Expected '%v', recieved: '%v'", common.ErrNotYetImplemented, err)
	}
}
