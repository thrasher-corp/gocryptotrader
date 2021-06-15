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
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
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
	b.Websocket = sharedtestvalues.NewTestWebsocket()
	err = b.Setup(bitmexConfig)
	if err != nil {
		log.Fatal("Bitmex setup error", err)
	}
	os.Exit(m.Run())
}

func TestStart(t *testing.T) {
	t.Parallel()
	var testWg sync.WaitGroup
	b.Start(&testWg)
	testWg.Wait()
}

func TestGetFullFundingHistory(t *testing.T) {
	t.Parallel()
	_, err := b.GetFullFundingHistory("", "", "", "", "", true, time.Now().Add(-time.Minute), time.Now())
	if err != nil {
		t.Error(err)
	}

	_, err = b.GetFullFundingHistory("LTCUSD", "1", "", "", "", true, time.Now().Add(-time.Minute), time.Now())
	if err != nil {
		t.Error(err)
	}
}

func TestGetUrgentAnnouncement(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	_, err := b.GetUrgentAnnouncement()
	if err == nil {
		t.Error("GetUrgentAnnouncement() Expected error")
	}
}

func TestGetAPIKeys(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	_, err := b.GetAPIKeys()
	if err == nil {
		t.Error("GetAPIKeys() Expected error")
	}
}

func TestRemoveAPIKey(t *testing.T) {
	t.Parallel()

	_, err := b.RemoveAPIKey(APIKeyParams{APIKeyID: "1337"})
	if err == nil {
		t.Error("RemoveAPIKey() Expected error")
	}
}

func TestDisableAPIKey(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	_, err := b.DisableAPIKey(APIKeyParams{APIKeyID: "1337"})
	if err == nil {
		t.Error("DisableAPIKey() Expected error")
	}
}

func TestEnableAPIKey(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	_, err := b.EnableAPIKey(APIKeyParams{APIKeyID: "1337"})
	if err == nil {
		t.Error("EnableAPIKey() Expected error")
	}
}

func TestGetTrollboxMessages(t *testing.T) {
	t.Parallel()
	_, err := b.GetTrollboxMessages(ChatGetParams{Count: 1})
	if err != nil {
		t.Error("GetTrollboxMessages() error", err)
	}
}

func TestSendTrollboxMessage(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	_, err := b.SendTrollboxMessage(ChatSendParams{
		ChannelID: 1337,
		Message:   "Hello,World!"})
	if err == nil {
		t.Error("SendTrollboxMessage() Expected error")
	}
}

func TestGetTrollboxChannels(t *testing.T) {
	t.Parallel()
	_, err := b.GetTrollboxChannels()
	if err != nil {
		t.Error("GetTrollboxChannels() error", err)
	}
}

func TestGetTrollboxConnectedUsers(t *testing.T) {
	t.Parallel()
	_, err := b.GetTrollboxConnectedUsers()
	if err == nil {
		t.Error("GetTrollboxConnectedUsers() Expected error")
	}
}

func TestGetAccountExecutions(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	_, err := b.GetAccountExecutions(&GenericRequestParams{})
	if err == nil {
		t.Error("GetAccountExecutions() Expected error")
	}
}

func TestGetAccountExecutionTradeHistory(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	_, err := b.GetAccountExecutionTradeHistory(&GenericRequestParams{})
	if err == nil {
		t.Error("GetAccountExecutionTradeHistory() Expected error")
	}
}

func TestGetFundingHistory(t *testing.T) {
	t.Parallel()
	_, err := b.GetFundingHistory()
	if err == nil {
		t.Error("GetFundingHistory() Expected error")
	}
}

func TestGetInstruments(t *testing.T) {
	t.Parallel()
	_, err := b.GetInstruments(&GenericRequestParams{
		Symbol: "XRPUSD",
	})
	if err != nil {
		t.Error("GetInstruments() error", err)
	}
}

func TestGetActiveInstruments(t *testing.T) {
	t.Parallel()
	_, err := b.GetActiveInstruments(&GenericRequestParams{})
	if err != nil {
		t.Error("GetActiveInstruments() error", err)
	}
}

func TestGetActiveAndIndexInstruments(t *testing.T) {
	t.Parallel()
	_, err := b.GetActiveAndIndexInstruments()
	if err != nil {
		t.Error("GetActiveAndIndexInstruments() error", err)
	}
}

func TestGetActiveIntervals(t *testing.T) {
	t.Parallel()
	_, err := b.GetActiveIntervals()
	if err == nil {
		t.Error("GetActiveIntervals() Expected error")
	}
}

func TestGetCompositeIndex(t *testing.T) {
	t.Parallel()
	_, err := b.GetCompositeIndex(".XBT", "", "", "", "", "", time.Time{}, time.Time{})
	if err != nil {
		t.Error("GetCompositeIndex() Expected error", err)
	}
}

func TestGetIndices(t *testing.T) {
	t.Parallel()
	_, err := b.GetIndices()
	if err != nil {
		t.Error("GetIndices() error", err)
	}
}

func TestGetInsuranceFundHistory(t *testing.T) {
	t.Parallel()
	_, err := b.GetInsuranceFundHistory(&GenericRequestParams{})
	if err != nil {
		t.Error("GetInsuranceFundHistory() error", err)
	}
}

func TestGetLeaderboard(t *testing.T) {
	t.Parallel()
	_, err := b.GetLeaderboard(LeaderboardGetParams{})
	if err != nil {
		t.Error("GetLeaderboard() error", err)
	}
}

func TestGetAliasOnLeaderboard(t *testing.T) {
	t.Parallel()
	_, err := b.GetAliasOnLeaderboard()
	if err == nil {
		t.Error("GetAliasOnLeaderboard() Expected error")
	}
}

func TestGetLiquidationOrders(t *testing.T) {
	t.Parallel()
	_, err := b.GetLiquidationOrders(&GenericRequestParams{})
	if err != nil {
		t.Error("GetLiquidationOrders() error", err)
	}
}

func TestGetCurrentNotifications(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	_, err := b.GetCurrentNotifications()
	if err == nil {
		t.Error("GetCurrentNotifications() Expected error")
	}
}

func TestAmendOrder(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	_, err := b.AmendOrder(&OrderAmendParams{})
	if err == nil {
		t.Error("AmendOrder() Expected error")
	}
}

func TestCreateOrder(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	_, err := b.CreateOrder(&OrderNewParams{Symbol: "XBTM15",
		Price:         219.0,
		ClientOrderID: "mm_bitmex_1a/oemUeQ4CAJZgP3fjHsA",
		OrderQuantity: 98})
	if err == nil {
		t.Error("CreateOrder() Expected error")
	}
}

func TestCancelOrders(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	_, err := b.CancelOrders(&OrderCancelParams{})
	if err == nil {
		t.Error("CancelOrders() Expected error")
	}
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	_, err := b.CancelAllExistingOrders(OrderCancelAllParams{})
	if err == nil {
		t.Error("CancelAllOrders(orderCancellation *order.Cancel) (order.CancelAllResponse, error)", err)
	}
}

func TestAmendBulkOrders(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	_, err := b.AmendBulkOrders(OrderAmendBulkParams{})
	if err == nil {
		t.Error("AmendBulkOrders() Expected error")
	}
}

func TestCreateBulkOrders(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	_, err := b.CreateBulkOrders(OrderNewBulkParams{})
	if err == nil {
		t.Error("CreateBulkOrders() Expected error")
	}
}

func TestCancelAllOrdersAfterTime(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	_, err := b.CancelAllOrdersAfterTime(OrderCancelAllAfterParams{})
	if err == nil {
		t.Error("CancelAllOrdersAfterTime() Expected error")
	}
}

func TestClosePosition(t *testing.T) {
	t.Parallel()
	_, err := b.ClosePosition(OrderClosePositionParams{})
	if err == nil {
		t.Error("ClosePosition() Expected error")
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := b.GetOrderbook(OrderBookGetL2Params{Symbol: "XBT"})
	if err != nil {
		t.Error("GetOrderbook() error", err)
	}
}

func TestGetPositions(t *testing.T) {
	t.Parallel()
	_, err := b.GetPositions(PositionGetParams{})
	if err == nil {
		t.Error("GetPositions() Expected error")
	}
}

func TestIsolatePosition(t *testing.T) {
	t.Parallel()
	_, err := b.IsolatePosition(PositionIsolateMarginParams{Symbol: "XBT"})
	if err == nil {
		t.Error("IsolatePosition() Expected error")
	}
}

func TestLeveragePosition(t *testing.T) {
	t.Parallel()
	_, err := b.LeveragePosition(PositionUpdateLeverageParams{})
	if err == nil {
		t.Error("LeveragePosition() Expected error")
	}
}

func TestUpdateRiskLimit(t *testing.T) {
	t.Parallel()
	_, err := b.UpdateRiskLimit(PositionUpdateRiskLimitParams{})
	if err == nil {
		t.Error("UpdateRiskLimit() Expected error")
	}
}

func TestTransferMargin(t *testing.T) {
	t.Parallel()
	_, err := b.TransferMargin(PositionTransferIsolatedMarginParams{})
	if err == nil {
		t.Error("TransferMargin() Expected error")
	}
}

func TestGetQuotesByBuckets(t *testing.T) {
	t.Parallel()
	_, err := b.GetQuotesByBuckets(&QuoteGetBucketedParams{})
	if err == nil {
		t.Error("GetQuotesByBuckets() Expected error")
	}
}

func TestGetSettlementHistory(t *testing.T) {
	t.Parallel()
	_, err := b.GetSettlementHistory(&GenericRequestParams{})
	if err != nil {
		t.Error("GetSettlementHistory() error", err)
	}
}

func TestGetStats(t *testing.T) {
	t.Parallel()
	_, err := b.GetStats()
	if err != nil {
		t.Error("GetStats() error", err)
	}
}

func TestGetStatsHistorical(t *testing.T) {
	t.Parallel()
	_, err := b.GetStatsHistorical()
	if err != nil {
		t.Error("GetStatsHistorical() error", err)
	}
}

func TestGetStatSummary(t *testing.T) {
	t.Parallel()
	_, err := b.GetStatSummary()
	if err != nil {
		t.Error("GetStatSummary() error", err)
	}
}

func TestGetTrade(t *testing.T) {
	t.Parallel()
	_, err := b.GetTrade(&GenericRequestParams{
		Symbol:    "XBT",
		Reverse:   false,
		StartTime: time.Now().Add(-time.Minute).Format(time.RFC3339),
	})
	if err != nil {
		t.Error("GetTrade() error", err)
	}
}

func TestGetPreviousTrades(t *testing.T) {
	t.Parallel()
	_, err := b.GetPreviousTrades(&TradeGetBucketedParams{
		Symbol:  "XBTBTC",
		Start:   int32(time.Now().Add(-time.Hour).Unix()),
		Columns: "open,high,low,close,volume",
	})
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
	t.Parallel()
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
	t.Parallel()
	var feeBuilder = setFeeBuilder()
	// CryptocurrencyTradeFee Basic
	if _, err := b.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// CryptocurrencyTradeFee High quantity
	feeBuilder = setFeeBuilder()
	feeBuilder.Amount = 1000
	feeBuilder.PurchasePrice = 1000
	if _, err := b.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// CryptocurrencyTradeFee IsMaker
	feeBuilder = setFeeBuilder()
	feeBuilder.IsMaker = true
	if _, err := b.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// CryptocurrencyTradeFee Negative purchase price
	feeBuilder = setFeeBuilder()
	feeBuilder.PurchasePrice = -1000
	if _, err := b.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if _, err := b.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// CryptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyDepositFee
	if _, err := b.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	feeBuilder.FiatCurrency = currency.HKD
	if _, err := b.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.HKD
	if _, err := b.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	t.Parallel()
	expectedResult := exchange.AutoWithdrawCryptoWithAPIPermissionText + " & " + exchange.WithdrawCryptoWith2FAText +
		" & " + exchange.WithdrawCryptoWithEmailText + " & " + exchange.NoFiatWithdrawalsText
	withdrawPermissions := b.FormatWithdrawPermissions()
	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s", expectedResult, withdrawPermissions)
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	var getOrdersRequest = order.GetOrdersRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
	}

	_, err := b.GetActiveOrders(&getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get open orders: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	var getOrdersRequest = order.GetOrdersRequest{
		Type: order.AnyType,
		Pairs: []currency.Pair{currency.NewPair(currency.LTC,
			currency.BTC)},
		AssetType: asset.Spot,
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
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var orderSubmission = &order.Submit{
		Pair: currency.Pair{
			Base:  currency.XBT,
			Quote: currency.USD,
		},
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     1,
		Amount:    1,
		ClientID:  "meowOrder",
		AssetType: asset.Futures,
	}
	response, err := b.SubmitOrder(orderSubmission)
	if areTestAPIKeysSet() && (err != nil || !response.IsOrderPlaced) {
		t.Errorf("Order failed to be placed: %v", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)
	var orderCancellation = &order.Cancel{
		ID:            "123456789012345678901234567890123456",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          currencyPair,
		AssetType:     asset.Futures,
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
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)
	var orderCancellation = &order.Cancel{
		ID:            "123456789012345678901234567890123456",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          currencyPair,
		AssetType:     asset.Futures,
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
	t.Parallel()
	if areTestAPIKeysSet() {
		_, err := b.UpdateAccountInfo(asset.Spot)
		if err != nil {
			t.Error("GetAccountInfo() error", err)
		}
	} else {
		_, err := b.UpdateAccountInfo(asset.Spot)
		if err == nil {
			t.Error("GetAccountInfo() error")
		}
	}
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	_, err := b.ModifyOrder(&order.Modify{ID: "1337", AssetType: asset.Futures})
	if err == nil {
		t.Error("ModifyOrder() error")
	}
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	withdrawCryptoRequest := withdraw.Request{
		Crypto: withdraw.CryptoRequest{
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
	if !b.Websocket.IsEnabled() && !b.API.AuthenticatedWebsocketSupport || !areTestAPIKeysSet() {
		t.Skip(stream.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
	err := b.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		t.Fatal(err)
	}

	go b.wsReadData()
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

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	err := b.UpdateTradablePairs(true)
	if err != nil {
		t.Fatal(err)
	}
}

func TestWsPositionUpdate(t *testing.T) {
	t.Parallel()
	pressXToJSON := []byte(`{"table":"position",
   "action":"update",
   "data":[{
    "account":2,"symbol":"ETHUSD","currency":"XBt",
    "currentTimestamp":"2017-04-04T22:07:42.442Z", "currentQty":1,"markPrice":1136.88,"markValue":-87960,
    "riskValue":87960,"homeNotional":0.0008796,"posState":"Liquidation","maintMargin":263,
    "unrealisedGrossPnl":-677,"unrealisedPnl":-677,"unrealisedPnlPcnt":-0.0078,"unrealisedRoePcnt":-0.7756,
    "simpleQty":0.001,"liquidationPrice":1140.1, "timestamp":"2017-04-04T22:07:45.442Z"
   }]}`)
	err := b.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsInsertExectuionUpdate(t *testing.T) {
	t.Parallel()
	pressXToJSON := []byte(`{"table":"execution",
   "action":"insert",
   "data":[{
    "execID":"0193e879-cb6f-2891-d099-2c4eb40fee21",
    "orderID":"00000000-0000-0000-0000-000000000000","clOrdID":"","clOrdLinkID":"","account":2,"symbol":"ETHUSD",
    "side":"Sell","lastQty":1,"lastPx":1134.37,"underlyingLastPx":null,"lastMkt":"XBME",
    "lastLiquidityInd":"RemovedLiquidity", "simpleOrderQty":null,"orderQty":1,"price":1134.37,"displayQty":null,
    "stopPx":null,"pegOffsetValue":null,"pegPriceType":"","currency":"USD","settlCurrency":"XBt",
    "execType":"Trade","ordType":"Limit","timeInForce":"ImmediateOrCancel","execInst":"",
    "contingencyType":"","exDestination":"XBME","ordStatus":"Filled","triggered":"","workingIndicator":false,
    "ordRejReason":"","simpleLeavesQty":0,"leavesQty":0,"simpleCumQty":0.001,"cumQty":1,"avgPx":1134.37,
    "commission":0.00075,"tradePublishIndicator":"DoNotPublishTrade","multiLegReportingType":"SingleSecurity",
    "text":"Liquidation","trdMatchID":"7f4ab7f6-0006-3234-76f4-ae1385aad00f","execCost":88155,"execComm":66,
    "homeNotional":-0.00088155,"foreignNotional":1,"transactTime":"2017-04-04T22:07:46.035Z",
    "timestamp":"2017-04-04T22:07:46.035Z"
   }]}`)
	err := b.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWSConnectionHandling(t *testing.T) {
	t.Parallel()
	pressXToJSON := []byte(`{"info":"Welcome to the BitMEX Realtime API.","version":"1.1.0",
     "timestamp":"2015-01-18T10:14:06.802Z","docs":"https://www.bitmex.com/app/wsAPI","heartbeatEnabled":false}`)
	err := b.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWSSubscriptionHandling(t *testing.T) {
	t.Parallel()
	pressXToJSON := []byte(`{"success":true,"subscribe":"trade:ETHUSD",
     "request":{"op":"subscribe","args":["trade:ETHUSD","instrument:ETHUSD"]}}`)
	err := b.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWSPositionUpdateHandling(t *testing.T) {
	t.Parallel()
	pressXToJSON := []byte(`{"table":"position",
   "action":"update",
   "data":[{
    "account":2,"symbol":"ETHUSD","currency":"XBt","currentQty":1,
    "markPrice":1136.88,"posState":"Liquidated","simpleQty":0.001,"liquidationPrice":1140.1,"bankruptPrice":1134.37,
    "timestamp":"2017-04-04T22:07:46.019Z"
   }]}`)
	err := b.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
	pressXToJSON = []byte(`{"table":"position",
   "action":"update",
   "data":[{
    "account":2,"symbol":"ETHUSD","currency":"XBt",
    "deleveragePercentile":null,"rebalancedPnl":1003,"prevRealisedPnl":-1003,"execSellQty":1,
    "execSellCost":88155,"execQty":0,"execCost":872,"execComm":131,"currentTimestamp":"2017-04-04T22:07:46.140Z",
    "currentQty":0,"currentCost":872,"currentComm":131,"realisedCost":872,"unrealisedCost":0,"grossExecCost":0,
    "isOpen":false,"markPrice":null,"markValue":0,"riskValue":0,"homeNotional":0,"foreignNotional":0,"posState":"",
    "posCost":0,"posCost2":0,"posInit":0,"posComm":0,"posMargin":0,"posMaint":0,"maintMargin":0,
    "realisedGrossPnl":-872,"realisedPnl":-1003,"unrealisedGrossPnl":0,"unrealisedPnl":0,
    "unrealisedPnlPcnt":0,"unrealisedRoePcnt":0,"simpleQty":0,"simpleCost":0,"simpleValue":0,"avgCostPrice":null,
    "avgEntryPrice":null,"breakEvenPrice":null,"marginCallPrice":null,"liquidationPrice":null,"bankruptPrice":null,
    "timestamp":"2017-04-04T22:07:46.140Z"
   }]}`)
	err = b.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWSOrderbookHandling(t *testing.T) {
	t.Parallel()
	pressXToJSON := []byte(`{
      "table":"orderBookL2_25",
      "keys":["symbol","id","side"],
      "types":{"id":"long","price":"float","side":"symbol","size":"long","symbol":"symbol"},
      "foreignKeys":{"side":"side","symbol":"instrument"},
      "attributes":{"id":"sorted","symbol":"grouped"},
      "action":"partial",
      "data":[
        {"symbol":"ETHUSD","id":17999992000,"side":"Sell","size":100,"price":80},
        {"symbol":"ETHUSD","id":17999993000,"side":"Sell","size":20,"price":70},
        {"symbol":"ETHUSD","id":17999994000,"side":"Sell","size":10,"price":60},
        {"symbol":"ETHUSD","id":17999995000,"side":"Buy","size":10,"price":50},
        {"symbol":"ETHUSD","id":17999996000,"side":"Buy","size":20,"price":40},
        {"symbol":"ETHUSD","id":17999997000,"side":"Buy","size":100,"price":30}
      ]
    }`)
	err := b.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}

	pressXToJSON = []byte(`{
      "table":"orderBookL2_25",
      "action":"update",
      "data":[
        {"symbol":"ETHUSD","id":17999995000,"side":"Buy","size":5}
      ]
    }`)
	err = b.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}

	pressXToJSON = []byte(`{
      "table":"orderBookL2_25",
      "action":"update",
      "data":[
      ]
    }`)
	err = b.wsHandleData(pressXToJSON)
	if err == nil {
		t.Error("Expected error")
	}

	pressXToJSON = []byte(`{
      "table":"orderBookL2_25",
      "action":"delete",
      "data":[
        {"symbol":"ETHUSD","id":17999995000,"side":"Buy"}
      ]
    }`)
	err = b.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}

	pressXToJSON = []byte(`{
      "table":"orderBookL2_25",
      "action":"delete",
      "data":[
        {"symbol":"ETHUSD","id":17999995000,"side":"Buy"}
      ]
    }`)
	err = b.wsHandleData(pressXToJSON)
	if err != nil && err.Error() != "delete error: cannot match ID on linked list 17999995000 not found" {
		t.Error(err)
	}
	if err == nil {
		t.Error("expecting error")
	}
}

func TestWSDeleveragePositionUpdateHandling(t *testing.T) {
	t.Parallel()
	pressXToJSON := []byte(`{"table":"position",
   "action":"update",
   "data":[{
    "account":2,"symbol":"ETHUSD","currency":"XBt","currentQty":2000,
    "markPrice":1160.72,"posState":"Deleverage","simpleQty":1.746,"liquidationPrice":1140.1,
    "timestamp":"2017-04-04T22:16:38.460Z"
   }]}`)
	err := b.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}

	pressXToJSON = []byte(`{"table":"position",
   "action":"update",
   "data":[{
    "account":2,"symbol":"ETHUSD","currency":"XBt",
    "deleveragePercentile":null,"rebalancedPnl":-2171150,"prevRealisedPnl":2172153,"execSellQty":2001,
    "execSellCost":172394155,"execQty":0,"execCost":-2259128,"execComm":87978,
    "currentTimestamp":"2017-04-04T22:16:38.547Z","currentQty":0,"currentCost":-2259128,
    "currentComm":87978,"realisedCost":-2259128,"unrealisedCost":0,"grossExecCost":0,"isOpen":false,
    "markPrice":null,"markValue":0,"riskValue":0,"homeNotional":0,"foreignNotional":0,"posState":"","posCost":0,
    "posCost2":0,"posInit":0,"posComm":0,"posMargin":0,"posMaint":0,"maintMargin":0,"realisedGrossPnl":2259128,
    "realisedPnl":2171150,"unrealisedGrossPnl":0,"unrealisedPnl":0,"unrealisedPnlPcnt":0,"unrealisedRoePcnt":0,
    "simpleQty":0,"simpleCost":0,"simpleValue":0,"simplePnl":0,"simplePnlPcnt":0,"avgCostPrice":null,
    "avgEntryPrice":null,"breakEvenPrice":null,"marginCallPrice":null,"liquidationPrice":null,"bankruptPrice":null,
    "timestamp":"2017-04-04T22:16:38.547Z"
   }]}`)
	err = b.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWSDeleverageExecutionInsertHandling(t *testing.T) {
	t.Parallel()
	pressXToJSON := []byte(`{"table":"execution",
   "action":"insert",
   "data":[{
    "execID":"20ad1ff4-c110-a4f2-dd31-f94eaa0701fd",
    "orderID":"00000000-0000-0000-0000-000000000000","clOrdID":"","clOrdLinkID":"","account":2,"symbol":"ETHUSD",
    "side":"Sell","lastQty":2000,"lastPx":1160.72,"underlyingLastPx":null,"lastMkt":"XBME",
    "lastLiquidityInd":"AddedLiquidity","simpleOrderQty":null,"orderQty":2000,"price":1160.72,"displayQty":null,
    "stopPx":null,"pegOffsetValue":null,"pegPriceType":"","currency":"USD","settlCurrency":"XBt","execType":"Trade",
    "ordType":"Limit","timeInForce":"GoodTillCancel","execInst":"","contingencyType":"","exDestination":"XBME",
    "ordStatus":"Filled","triggered":"","workingIndicator":false,"ordRejReason":"",
    "simpleLeavesQty":0,"leavesQty":0,"simpleCumQty":1.746,"cumQty":2000,"avgPx":1160.72,"commission":-0.00025,
    "tradePublishIndicator":"PublishTrade","multiLegReportingType":"SingleSecurity","text":"Deleverage",
    "trdMatchID":"1e849b8a-7e88-3c67-a93f-cc654d40e8ba","execCost":172306000,"execComm":-43077,
    "homeNotional":-1.72306,"foreignNotional":2000,"transactTime":"2017-04-04T22:16:38.472Z",
    "timestamp":"2017-04-04T22:16:38.472Z"
   }]}`)
	err := b.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsTrades(t *testing.T) {
	t.Parallel()
	pressXToJSON := []byte(`{"table":"trade","action":"insert","data":[{"timestamp":"2020-02-17T01:35:36.442Z","symbol":"ETHUSD","side":"Sell","size":100,"price":258.3,"tickDirection":"MinusTick","trdMatchID":"c427f7a0-6b26-1e10-5c4e-1bd74daf2a73","grossValue":2583000,"homeNotional":0.9904912836767037,"foreignNotional":255.84389857369254},{"timestamp":"2020-02-17T01:35:36.442Z","symbol":"ETHUSD","side":"Sell","size":100,"price":258.3,"tickDirection":"ZeroMinusTick","trdMatchID":"95eb9155-b58c-70e9-44b7-34efe50302e0","grossValue":2583000,"homeNotional":0.9904912836767037,"foreignNotional":255.84389857369254},{"timestamp":"2020-02-17T01:35:36.442Z","symbol":"ETHUSD","side":"Sell","size":100,"price":258.3,"tickDirection":"ZeroMinusTick","trdMatchID":"e607c187-f25c-86bc-cb39-8afff7aaf2d9","grossValue":2583000,"homeNotional":0.9904912836767037,"foreignNotional":255.84389857369254},{"timestamp":"2020-02-17T01:35:36.442Z","symbol":"ETHUSD","side":"Sell","size":17,"price":258.3,"tickDirection":"ZeroMinusTick","trdMatchID":"0f076814-a57d-9a59-8063-ad6b823a80ac","grossValue":439110,"homeNotional":0.1683835182250396,"foreignNotional":43.49346275752773},{"timestamp":"2020-02-17T01:35:36.442Z","symbol":"ETHUSD","side":"Sell","size":100,"price":258.25,"tickDirection":"MinusTick","trdMatchID":"f4ef3dfd-51c4-538f-37c1-e5071ba1c75d","grossValue":2582500,"homeNotional":0.9904912836767037,"foreignNotional":255.79437400950872},{"timestamp":"2020-02-17T01:35:36.442Z","symbol":"ETHUSD","side":"Sell","size":100,"price":258.25,"tickDirection":"ZeroMinusTick","trdMatchID":"81ef136b-8f4a-b1cf-78a8-fffbfa89bf40","grossValue":2582500,"homeNotional":0.9904912836767037,"foreignNotional":255.79437400950872},{"timestamp":"2020-02-17T01:35:36.442Z","symbol":"ETHUSD","side":"Sell","size":100,"price":258.25,"tickDirection":"ZeroMinusTick","trdMatchID":"65a87e8c-7563-34a4-d040-94e8513c5401","grossValue":2582500,"homeNotional":0.9904912836767037,"foreignNotional":255.79437400950872},{"timestamp":"2020-02-17T01:35:36.442Z","symbol":"ETHUSD","side":"Sell","size":15,"price":258.25,"tickDirection":"ZeroMinusTick","trdMatchID":"1d11a74e-a157-3f33-036d-35a101fba50b","grossValue":387375,"homeNotional":0.14857369255150554,"foreignNotional":38.369156101426306},{"timestamp":"2020-02-17T01:35:36.442Z","symbol":"ETHUSD","side":"Sell","size":1,"price":258.25,"tickDirection":"ZeroMinusTick","trdMatchID":"40d49df1-f018-f66f-4ca5-31d4997641d7","grossValue":25825,"homeNotional":0.009904912836767036,"foreignNotional":2.5579437400950873},{"timestamp":"2020-02-17T01:35:36.442Z","symbol":"ETHUSD","side":"Sell","size":100,"price":258.2,"tickDirection":"MinusTick","trdMatchID":"36135b51-73e5-c007-362b-a55be5830c6b","grossValue":2582000,"homeNotional":0.9904912836767037,"foreignNotional":255.7448494453249},{"timestamp":"2020-02-17T01:35:36.442Z","symbol":"ETHUSD","side":"Sell","size":100,"price":258.2,"tickDirection":"ZeroMinusTick","trdMatchID":"6ee19edb-99aa-3030-ba63-933ffb347ade","grossValue":2582000,"homeNotional":0.9904912836767037,"foreignNotional":255.7448494453249},{"timestamp":"2020-02-17T01:35:36.442Z","symbol":"ETHUSD","side":"Sell","size":100,"price":258.2,"tickDirection":"ZeroMinusTick","trdMatchID":"d44be603-cdb8-d676-e3e2-f91fb12b2a70","grossValue":2582000,"homeNotional":0.9904912836767037,"foreignNotional":255.7448494453249},{"timestamp":"2020-02-17T01:35:36.442Z","symbol":"ETHUSD","side":"Sell","size":5,"price":258.2,"tickDirection":"ZeroMinusTick","trdMatchID":"a14b43b3-50b4-c075-c54d-dfb0165de33d","grossValue":129100,"homeNotional":0.04952456418383518,"foreignNotional":12.787242472266245},{"timestamp":"2020-02-17T01:35:36.442Z","symbol":"ETHUSD","side":"Sell","size":8,"price":258.2,"tickDirection":"ZeroMinusTick","trdMatchID":"3c30e175-5194-320c-8f8c-01636c2f4a32","grossValue":206560,"homeNotional":0.07923930269413629,"foreignNotional":20.45958795562599},{"timestamp":"2020-02-17T01:35:36.442Z","symbol":"ETHUSD","side":"Sell","size":50,"price":258.2,"tickDirection":"ZeroMinusTick","trdMatchID":"5b803378-760b-4919-21fc-bfb275d39ace","grossValue":1291000,"homeNotional":0.49524564183835185,"foreignNotional":127.87242472266244},{"timestamp":"2020-02-17T01:35:36.442Z","symbol":"ETHUSD","side":"Sell","size":244,"price":258.2,"tickDirection":"ZeroMinusTick","trdMatchID":"cf57fec1-c444-b9e5-5e2d-4fb643f4fdb7","grossValue":6300080,"homeNotional":2.416798732171157,"foreignNotional":624.0174326465927}]}`)
	err := b.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	err := b.UpdateTradablePairs(false)
	if err != nil {
		t.Fatal(err)
	}
	currencyPair := b.CurrencyPairs.Pairs[asset.Futures].Available[0]
	_, err = b.GetRecentTrades(currencyPair, asset.Futures)
	if err != nil {
		t.Error(err)
	}
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	err := b.UpdateTradablePairs(false)
	if err != nil {
		t.Fatal(err)
	}
	currencyPair := b.CurrencyPairs.Pairs[asset.Futures].Available[0]
	_, err = b.GetHistoricTrades(currencyPair, asset.Futures, time.Now().Add(-time.Minute), time.Now())
	if err != nil {
		t.Error(err)
	}
}
