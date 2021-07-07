package bitfinex

import (
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// Please supply your own keys here to do better tests
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var b Bitfinex
var wsAuthExecuted bool

func TestMain(m *testing.M) {
	b.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal("Bitfinex load config error", err)
	}
	bfxConfig, err := cfg.GetExchangeConfig("Bitfinex")
	if err != nil {
		log.Fatal("Bitfinex Setup() init error")
	}
	b.Websocket = sharedtestvalues.NewTestWebsocket()
	err = b.Setup(bfxConfig)
	if err != nil {
		log.Fatal("Bitfinex setup error", err)
	}
	b.API.Credentials.Key = apiKey
	b.API.Credentials.Secret = apiSecret
	if !b.Enabled || b.API.AuthenticatedSupport ||
		b.Verbose || b.Websocket.IsEnabled() || len(b.BaseCurrencies) < 1 {
		log.Fatal("Bitfinex Setup values not set correctly")
	}

	if areTestAPIKeysSet() {
		b.API.AuthenticatedSupport = true
		b.API.AuthenticatedWebsocketSupport = true
	}
	b.WebsocketSubdChannels = make(map[int]WebsocketChanInfo)
	os.Exit(m.Run())
}

func TestGetV2MarginFunding(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip("api keys are not set or invalid")
	}
	_, err := b.GetV2MarginFunding("fUSD", "2", 2)
	if err != nil {
		t.Error(err)
	}
}

func TestGetV2MarginInfo(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip("api keys are not set or invalid")
	}
	_, err := b.GetV2MarginInfo("base")
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetV2MarginInfo("tBTCUSD")
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetV2MarginInfo("sym_all")
	if err != nil {
		t.Error(err)
	}
}

func TestGetAccountInfoV2(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("api keys are not set or invalid")
	}
	_, err := b.GetAccountInfoV2()
	if err != nil {
		t.Error(err)
	}
}

func TestGetV2FundingInfo(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip("api keys are not set or invalid")
	}
	_, err := b.GetV2FundingInfo("fUST")
	if err != nil {
		t.Error(err)
	}
}

func TestGetV2Balances(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("api keys are not set or invalid")
	}
	_, err := b.GetV2Balances()
	if err != nil {
		t.Error(err)
	}
}

func TestGetDerivativeStatusInfo(t *testing.T) {
	t.Parallel()
	_, err := b.GetDerivativeStatusInfo("ALL", "", "", 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarginPairs(t *testing.T) {
	t.Parallel()
	_, err := b.GetMarginPairs()
	if err != nil {
		t.Error(err)
	}
}

func TestAppendOptionalDelimiter(t *testing.T) {
	t.Parallel()
	curr1, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	b.appendOptionalDelimiter(&curr1)
	if curr1.Delimiter != "" {
		t.Errorf("Expected no delimiter, received %v", curr1.Delimiter)
	}
	curr2, err := currency.NewPairFromString("DUSK:USD")
	if err != nil {
		t.Fatal(err)
	}

	curr2.Delimiter = ""
	b.appendOptionalDelimiter(&curr2)
	if curr2.Delimiter != ":" {
		t.Errorf("Expected \":\" as a delimiter, received %v", curr2.Delimiter)
	}
}

func TestGetPlatformStatus(t *testing.T) {
	t.Parallel()
	result, err := b.GetPlatformStatus()
	if err != nil {
		t.Errorf("TestGetPlatformStatus error: %s", err)
	}

	if result != bitfinexOperativeMode && result != bitfinexMaintenanceMode {
		t.Errorf("TestGetPlatformStatus unexpected response code")
	}
}

func TestGetTickerBatch(t *testing.T) {
	t.Parallel()
	_, err := b.GetTickerBatch()
	if err != nil {
		t.Error(err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := b.GetTicker("tBTCUSD")
	if err != nil {
		t.Error(err)
	}

	_, err = b.GetTicker("fUSD")
	if err != nil {
		t.Error(err)
	}
}

func TestGetTrades(t *testing.T) {
	t.Parallel()

	_, err := b.GetTrades("tBTCUSD", 5, 0, 0, false)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := b.GetOrderbook("tBTCUSD", "R0", 1)
	if err != nil {
		t.Error(err)
	}

	_, err = b.GetOrderbook("fUSD", "R0", 1)
	if err != nil {
		t.Error(err)
	}

	_, err = b.GetOrderbook("tBTCUSD", "P0", 1)
	if err != nil {
		t.Error(err)
	}

	_, err = b.GetOrderbook("fUSD", "P0", 1)
	if err != nil {
		t.Error(err)
	}

	_, err = b.GetOrderbook("tLINK:UST", "P0", 1)
	if err != nil {
		t.Error(err)
	}
}

func TestGetStats(t *testing.T) {
	t.Parallel()
	_, err := b.GetStats("btcusd")
	if err != nil {
		t.Error(err)
	}
}

func TestGetFundingBook(t *testing.T) {
	t.Parallel()
	_, err := b.GetFundingBook("usd")
	if err != nil {
		t.Error(err)
	}
}

func TestGetLends(t *testing.T) {
	t.Parallel()
	_, err := b.GetLends("usd", nil)
	if err != nil {
		t.Error(err)
	}
}

func TestGetCandles(t *testing.T) {
	t.Parallel()
	_, err := b.GetCandles("fUSD", "1m", 0, 0, 10, true)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetLeaderboard(t *testing.T) {
	t.Parallel()
	// Test invalid key
	_, err := b.GetLeaderboard("", "", "", 0, 0, "", "")
	if err == nil {
		t.Error("an error should have been thrown for an invalid key")
	}
	// Test default
	_, err = b.GetLeaderboard(LeaderboardUnrealisedProfitInception,
		"1M",
		"tGLOBAL:USD",
		0,
		0,
		"",
		"")
	if err != nil {
		t.Fatal(err)
	}
	// Test params
	var result []LeaderboardEntry
	result, err = b.GetLeaderboard(LeaderboardUnrealisedProfitInception,
		"1M",
		"tGLOBAL:USD",
		-1,
		1000,
		"1582695181661",
		"1583299981661")
	if err != nil {
		t.Fatal(err)
	}
	if len(result) == 0 {
		t.Error("should have retrieved leaderboard data")
	}
}

func TestGetAccountFees(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.UpdateAccountInfo(asset.Spot)
	if err != nil {
		t.Error("GetAccountInfo error", err)
	}
}

func TestGetWithdrawalFee(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	t.Parallel()
	_, err := b.GetWithdrawalFees()
	if err != nil {
		t.Error("GetAccountInfo error", err)
	}
}

func TestGetAccountSummary(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.GetAccountSummary()
	if err == nil {
		t.Error("GetAccountSummary() Expected error")
	}
}

func TestNewDeposit(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()
	_, err := b.NewDeposit("blabla", "testwallet", 0)
	if err == nil {
		t.Error("NewDeposit() Expected error")
	}

	_, err = b.NewDeposit("bitcoin", "testwallet", 0)
	if err == nil {
		t.Error("NewDeposit() Expected error")
	}

	_, err = b.NewDeposit("bitcoin", "exchange", 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetKeyPermissions(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.GetKeyPermissions()
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarginInfo(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.GetMarginInfo()
	if err != nil {
		t.Error(err)
	}
}

func TestGetAccountBalance(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.GetAccountBalance()
	if err != nil {
		t.Error(err)
	}
}

func TestGetAccountInfo(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.FetchAccountInfo(asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestWalletTransfer(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.WalletTransfer(0.01, "btc", "bla", "bla")
	if err == nil {
		t.Error("error cannot be nil")
	}
}

func TestNewOrder(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.NewOrder("BTCUSD",
		order.Limit.Lower(),
		-1,
		2,
		false,
		true)
	if err == nil {
		t.Error(err)
	}
}

func TestUpdateTicker(t *testing.T) {
	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.UpdateTicker(pair, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestNewOrderMulti(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	newOrder := []PlaceOrder{
		{
			Symbol:   "BTCUSD",
			Amount:   1,
			Price:    1,
			Exchange: "bitfinex",
			Side:     order.Buy.Lower(),
			Type:     order.Limit.Lower(),
		},
	}

	_, err := b.NewOrderMulti(newOrder)
	if err == nil {
		t.Error("NewOrderMulti() Expected error")
	}
}

func TestCancelOrder(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.CancelExistingOrder(1337)
	if err == nil {
		t.Error("CancelExistingOrder() Expected error")
	}
}

func TestCancelMultipleOrders(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.CancelMultipleOrders([]int64{1337, 1336})
	if err == nil {
		t.Error("CancelMultipleOrders() Expected error")
	}
}

func TestCancelAllOrders(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.CancelAllExistingOrders()
	if err == nil {
		t.Error("CancelAllExistingOrders() Expected error")
	}
}

func TestReplaceOrder(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.ReplaceOrder(1337, "BTCUSD",
		1, 1, true, order.Limit.Lower(), false)
	if err == nil {
		t.Error("ReplaceOrder() Expected error")
	}
}

func TestGetOrderStatus(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.GetOrderStatus(1337)
	if err == nil {
		t.Error("GetOrderStatus() Expected error")
	}
}

func TestGetOpenOrders(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.GetOpenOrders()
	if err == nil {
		t.Error("GetOpenOrders() Expectederror")
	}
}

func TestGetActivePositions(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.GetActivePositions()
	if err == nil {
		t.Error("GetActivePositions() Expected error")
	}
}

func TestClaimPosition(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.ClaimPosition(1337)
	if err == nil {
		t.Error("ClaimPosition() Expected error")
	}
}

func TestGetBalanceHistory(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()
	_, err := b.GetBalanceHistory("USD", time.Time{}, time.Time{}, 1, "deposit")
	if err == nil {
		t.Error("GetBalanceHistory() Expected error")
	}
}

func TestGetMovementHistory(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.GetMovementHistory("USD", "bitcoin", time.Time{}, time.Time{}, 1)
	if err == nil {
		t.Error("GetMovementHistory() Expected error")
	}
}

func TestGetTradeHistory(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()
	_, err := b.GetTradeHistory("BTCUSD", time.Time{}, time.Time{}, 1, 0)
	if err == nil {
		t.Error("GetTradeHistory() Expected error")
	}
}

func TestNewOffer(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.NewOffer("BTC", 1, 1, 1, "loan")
	if err == nil {
		t.Error("NewOffer() Expected error")
	}
}

func TestCancelOffer(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.CancelOffer(1337)
	if err == nil {
		t.Error("CancelOffer() Expected error")
	}
}

func TestGetOfferStatus(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.GetOfferStatus(1337)
	if err == nil {
		t.Error("NewOffer() Expected error")
	}
}

func TestGetActiveCredits(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.GetActiveCredits()
	if err == nil {
		t.Error("GetActiveCredits() Expected error")
	}
}

func TestGetActiveOffers(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.GetActiveOffers()
	if err == nil {
		t.Error("GetActiveOffers() Expected error")
	}
}

func TestGetActiveMarginFunding(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.GetActiveMarginFunding()
	if err == nil {
		t.Error("GetActiveMarginFunding() Expected error")
	}
}

func TestGetUnusedMarginFunds(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.GetUnusedMarginFunds()
	if err == nil {
		t.Error("GetUnusedMarginFunds() Expected error")
	}
}

func TestGetMarginTotalTakenFunds(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.GetMarginTotalTakenFunds()
	if err == nil {
		t.Error("GetMarginTotalTakenFunds() Expected error")
	}
}

func TestCloseMarginFunding(t *testing.T) {
	if !b.ValidateAPICredentials() {
		t.SkipNow()
	}
	t.Parallel()

	_, err := b.CloseMarginFunding(1337)
	if err == nil {
		t.Error("CloseMarginFunding() Expected error")
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
	t.Parallel()

	if areTestAPIKeysSet() {
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
	expectedResult := exchange.AutoWithdrawCryptoWithAPIPermissionText + " & " + exchange.AutoWithdrawFiatWithAPIPermissionText
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
		Type:      order.AnyType,
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
			Delimiter: "_",
			Base:      currency.XRP,
			Quote:     currency.USD,
		},
		AssetType: asset.Spot,
		Side:      order.Sell,
		Type:      order.Limit,
		Price:     1000,
		Amount:    20,
		ClientID:  "meowOrder",
	}
	response, err := b.SubmitOrder(orderSubmission)

	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not place order: %v", err)
	}
	if areTestAPIKeysSet() && !response.IsOrderPlaced {
		t.Error("Order not placed")
	}
	if !areTestAPIKeysSet() && err == nil {
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
		ID:            "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          currencyPair,
		AssetType:     asset.Spot,
	}

	err := b.CancelOrder(orderCancellation)
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}
}

func TestCancelAllExchangeOrdera(t *testing.T) {
	t.Parallel()
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

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	_, err := b.ModifyOrder(&order.Modify{AssetType: asset.Spot})
	if err == nil {
		t.Error("ModifyOrder() Expected error")
	}
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	withdrawCryptoRequest := withdraw.Request{
		Amount:      -1,
		Currency:    currency.BTC,
		Description: "WITHDRAW IT ALL",
		Crypto: withdraw.CryptoRequest{
			Address: core.BitcoinDonationAddress,
		},
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

	var withdrawFiatRequest = withdraw.Request{
		Amount:      -1,
		Currency:    currency.USD,
		Description: "WITHDRAW IT ALL",
		Fiat: withdraw.FiatRequest{
			WireCurrency: currency.USD.String(),
		},
	}

	_, err := b.WithdrawFiatFunds(&withdrawFiatRequest)
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = withdraw.Request{
		Amount:      -1,
		Currency:    currency.BTC,
		Description: "WITHDRAW IT ALL",
		Fiat: withdraw.FiatRequest{
			WireCurrency:                  currency.USD.String(),
			RequiresIntermediaryBank:      true,
			IsExpressWire:                 false,
			IntermediaryBankAccountNumber: 12345,
			IntermediaryBankAddress:       "123 Fake St",
			IntermediaryBankCity:          "Tarry Town",
			IntermediaryBankCountry:       "Hyrule",
			IntermediaryBankName:          "Federal Reserve Bank",
			IntermediarySwiftCode:         "Taylor",
		},
	}

	_, err := b.WithdrawFiatFundsToInternationalBank(&withdrawFiatRequest)
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() {
		_, err := b.GetDepositAddress(currency.BTC, "deposit")
		if err != nil {
			t.Error("GetDepositAddress() error", err)
		}
	} else {
		_, err := b.GetDepositAddress(currency.BTC, "deposit")
		if err == nil {
			t.Error("GetDepositAddress() error cannot be nil")
		}
	}
}

func setupWs() {
	var dialer websocket.Dialer
	err := b.Websocket.AuthConn.Dial(&dialer, http.Header{})
	if err != nil {
		log.Fatal(err)
	}
	go b.wsReadData(b.Websocket.AuthConn)
	go b.WsDataHandler()
}

// TestWsAuth dials websocket, sends login request.
func TestWsAuth(t *testing.T) {
	if !b.Websocket.IsEnabled() && !b.API.AuthenticatedWebsocketSupport || !areTestAPIKeysSet() {
		t.Skip("API keys not set, skipping")
	}
	runAuth(t)
}

func runAuth(t *testing.T) {
	setupWs()
	err := b.WsSendAuth()
	if err != nil {
		t.Error(err)
	}
	timer := time.NewTimer(sharedtestvalues.WebsocketResponseDefaultTimeout)
	select {
	case resp := <-b.Websocket.DataHandler:
		if logResponse, ok := resp.(map[string]interface{}); ok {
			if logResponse["event"] != "auth" && logResponse["status"] != "OK" {
				t.Error("expected successful login")
			}
		} else {
			t.Error("Unexpected response")
		}
	case <-timer.C:
		t.Error("Have not received a response")
	}
	timer.Stop()
	wsAuthExecuted = true
}

// TestWsPlaceOrder dials websocket, sends order request.
func TestWsPlaceOrder(t *testing.T) {
	if !b.Websocket.IsEnabled() && !b.API.AuthenticatedWebsocketSupport || !areTestAPIKeysSet() {
		t.Skip("API keys not set, skipping")
	}
	if !wsAuthExecuted {
		runAuth(t)
	}

	_, err := b.WsNewOrder(&WsNewOrderRequest{
		GroupID: 1,
		Type:    "EXCHANGE LIMIT",
		Symbol:  "tXRPUSD",
		Amount:  -20,
		Price:   1000,
	})
	if err != nil {
		t.Error(err)
	}
}

// TestWsCancelOrder dials websocket, sends cancel request.
func TestWsCancelOrder(t *testing.T) {
	if !b.Websocket.IsEnabled() && !b.API.AuthenticatedWebsocketSupport || !areTestAPIKeysSet() {
		t.Skip("API keys not set, skipping")
	}
	if !wsAuthExecuted {
		runAuth(t)
	}
	err := b.WsCancelOrder(1234)
	if err != nil {
		t.Error(err)
	}
}

// TestWsCancelOrder dials websocket, sends modify request.
func TestWsUpdateOrder(t *testing.T) {
	if !b.Websocket.IsEnabled() && !b.API.AuthenticatedWebsocketSupport || !areTestAPIKeysSet() {
		t.Skip("API keys not set, skipping")
	}
	if !wsAuthExecuted {
		runAuth(t)
	}
	err := b.WsModifyOrder(&WsUpdateOrderRequest{
		OrderID: 1234,
		Price:   -111,
		Amount:  111,
	})
	if err != nil {
		t.Error(err)
	}
}

// TestWsCancelAllOrders dials websocket, sends cancel all request.
func TestWsCancelAllOrders(t *testing.T) {
	if !b.Websocket.IsEnabled() && !b.API.AuthenticatedWebsocketSupport || !areTestAPIKeysSet() {
		t.Skip("API keys not set, skipping")
	}
	if !wsAuthExecuted {
		runAuth(t)
	}
	err := b.WsCancelAllOrders()
	if err != nil {
		t.Error(err)
	}
}

// TestWsCancelAllOrders dials websocket, sends cancel all request.
func TestWsCancelMultiOrders(t *testing.T) {
	if !b.Websocket.IsEnabled() && !b.API.AuthenticatedWebsocketSupport || !areTestAPIKeysSet() {
		t.Skip("API keys not set, skipping")
	}
	if !wsAuthExecuted {
		runAuth(t)
	}
	err := b.WsCancelMultiOrders([]int64{1, 2, 3, 4})
	if err != nil {
		t.Error(err)
	}
}

// TestWsNewOffer dials websocket, sends new offer request.
func TestWsNewOffer(t *testing.T) {
	if !b.Websocket.IsEnabled() && !b.API.AuthenticatedWebsocketSupport || !areTestAPIKeysSet() {
		t.Skip("API keys not set, skipping")
	}
	if !wsAuthExecuted {
		runAuth(t)
	}
	err := b.WsNewOffer(&WsNewOfferRequest{
		Type:   order.Limit.String(),
		Symbol: "fBTC",
		Amount: -10,
		Rate:   10,
		Period: 30,
	})
	if err != nil {
		t.Error(err)
	}
}

// TestWsCancelOffer dials websocket, sends cancel offer request.
func TestWsCancelOffer(t *testing.T) {
	if !b.Websocket.IsEnabled() && !b.API.AuthenticatedWebsocketSupport || !areTestAPIKeysSet() {
		t.Skip("API keys not set, skipping")
	}
	if !wsAuthExecuted {
		runAuth(t)
	}
	err := b.WsCancelOffer(1234)
	if err != nil {
		t.Error(err)
	}
}

func TestConvertSymbolToDepositMethod(t *testing.T) {
	s, err := b.ConvertSymbolToDepositMethod(currency.BTC)
	if err != nil {
		log.Fatal(err)
	}
	if s != "bitcoin" {
		t.Errorf("expected bitcoin but received %s", s)
	}

	_, err = b.ConvertSymbolToDepositMethod(currency.NewCode("CATS!"))
	if err == nil {
		log.Fatal("error cannot be nil")
	}
}

func TestUpdateTradablePairs(t *testing.T) {
	err := b.UpdateTradablePairs(false)
	if err != nil {
		t.Error(err)
	}
}

func TestWsSubscribedResponse(t *testing.T) {
	pressXToJSON := `{"event":"subscribed","channel":"ticker","chanId":224555,"symbol":"tBTCUSD","pair":"BTCUSD"}`
	err := b.wsHandleData([]byte(pressXToJSON))
	if err != nil {
		t.Error(err)
	}
}

func TestWsTradingPairSnapshot(t *testing.T) {
	b.WebsocketSubdChannels[23405] = WebsocketChanInfo{Pair: "BTCUSD", Channel: wsBook}
	pressXToJSON := `[23405,[[38334303613,9348.8,0.53],[38334308111,9348.8,5.98979404],[38331335157,9344.1,1.28965787],[38334302803,9343.8,0.08230094],[38334279092,9343,0.8],[38334307036,9342.938663676,0.8],[38332749107,9342.9,0.2],[38332277330,9342.8,0.85],[38329406786,9342,0.1432012],[38332841570,9341.947288638,0.3],[38332163238,9341.7,0.3],[38334303384,9341.6,0.324],[38332464840,9341.4,0.5],[38331935870,9341.2,0.5],[38334312082,9340.9,0.02126899],[38334261292,9340.8,0.26763],[38334138680,9340.625455254,0.12],[38333896802,9339.8,0.85],[38331627527,9338.9,1.57863959],[38334186713,9338.9,0.26769],[38334305819,9338.8,2.999],[38334211180,9338.75285796,3.999],[38334310699,9337.8,0.10679883],[38334307414,9337.5,1],[38334179822,9337.1,0.26773],[38334306600,9336.659955102,1.79],[38334299667,9336.6,1.1],[38334306452,9336.6,0.13979771],[38325672859,9336.3,1.25],[38334311646,9336.2,1],[38334258509,9336.1,0.37],[38334310592,9336,1.79],[38334310378,9335.6,1.43],[38334132444,9335.2,0.26777],[38331367325,9335,0.07],[38334310703,9335,0.10680562],[38334298209,9334.7,0.08757301],[38334304857,9334.456899462,0.291],[38334309940,9334.088390727,0.0725],[38334310377,9333.7,1.2868],[38334297615,9333.607784,0.1108],[38334095188,9333.3,0.26785],[38334228913,9332.7,0.40861186],[38334300526,9332.363996604,0.3884],[38334310701,9332.2,0.10680562],[38334303548,9332.005382871,0.07],[38334311798,9331.8,0.41285228],[38334301012,9331.7,1.7952],[38334089877,9331.4,0.2679],[38321942150,9331.2,0.2],[38334310670,9330,1.069],[38334063096,9329.6,0.26796],[38334310700,9329.4,0.10680562],[38334310404,9329.3,1],[38334281630,9329.1,6.57150597],[38334036864,9327.7,0.26801],[38334310702,9326.6,0.10680562],[38334311799,9326.1,0.50220625],[38334164163,9326,0.219638],[38334309722,9326,1.5],[38333051682,9325.8,0.26807],[38334302027,9325.7,0.75],[38334203435,9325.366592,0.32397696],[38321967613,9325,0.05],[38334298787,9324.9,0.3],[38334301719,9324.8,3.6227592],[38331316716,9324.763454646,0.71442],[38334310698,9323.8,0.10680562],[38334035499,9323.7,0.23431017],[38334223472,9322.670551788,0.42150603],[38334163459,9322.560399006,0.143967],[38321825171,9320.8,2],[38334075805,9320.467496148,0.30772633],[38334075800,9319.916732238,0.61457592],[38333682302,9319.7,0.0011],[38331323088,9319.116771762,0.12913],[38333677480,9319,0.0199],[38334277797,9318.6,0.89],[38325235155,9318.041088,1.20249],[38334310910,9317.82382938,1.79],[38334311811,9317.2,0.61079138],[38334311812,9317.2,0.71937652],[38333298214,9317.1,50],[38334306359,9317,1.79],[38325531545,9316.382823951,0.21263],[38333727253,9316.3,0.02316372],[38333298213,9316.1,45],[38333836479,9316,2.135],[38324520465,9315.9,2.7681],[38334307411,9315.5,1],[38330313617,9315.3,0.84455],[38334077770,9315.294024,0.01248397],[38334286663,9315.294024,1],[38325533762,9315.290315394,2.40498],[38334310018,9315.2,3],[38333682617,9314.6,0.0011],[38334304794,9314.6,0.76364676],[38334304798,9314.3,0.69242113],[38332915733,9313.8,0.0199],[38334084411,9312.8,1],[38334311893,9350.1,-1.015],[38334302734,9350.3,-0.26737],[38334300732,9350.8,-5.2],[38333957619,9351,-0.90677089],[38334300521,9351,-1.6457],[38334301600,9351.012829557,-0.0523],[38334308878,9351.7,-2.5],[38334299570,9351.921544,-0.1015],[38334279367,9352.1,-0.26732],[38334299569,9352.411802928,-0.4036],[38334202773,9353.4,-0.02139404],[38333918472,9353.7,-1.96412776],[38334278782,9354,-0.26731],[38334278606,9355,-1.2785],[38334302105,9355.439221251,-0.79191542],[38313897370,9355.569409242,-0.43363],[38334292995,9355.584296,-0.0979],[38334216989,9355.8,-0.03686414],[38333894025,9355.9,-0.26721],[38334293798,9355.936691952,-0.4311],[38331159479,9356,-0.4204022],[38333918888,9356.1,-1.10885563],[38334298205,9356.4,-0.20124428],[38328427481,9356.5,-0.1],[38333343289,9356.6,-0.41034213],[38334297205,9356.6,-0.08835018],[38334277927,9356.741101161,-0.0737],[38334311645,9356.8,-0.5],[38334309002,9356.9,-5],[38334309736,9357,-0.10680107],[38334306448,9357.4,-0.18645275],[38333693302,9357.7,-0.2672],[38332815159,9357.8,-0.0011],[38331239824,9358.2,-0.02],[38334271608,9358.3,-2.999],[38334311971,9358.4,-0.55],[38333919260,9358.5,-1.9972841],[38334265365,9358.5,-1.7841],[38334277960,9359,-3],[38334274601,9359.020969848,-3],[38326848839,9359.1,-0.84],[38334291080,9359.247048,-0.16199869],[38326848844,9359.4,-1.84],[38333680200,9359.6,-0.26713],[38331326606,9359.8,-0.84454],[38334309738,9359.8,-0.10680107],[38331314707,9359.9,-0.2],[38333919803,9360.9,-1.41177599],[38323651149,9361.33417827,-0.71442],[38333656906,9361.5,-0.26705],[38334035500,9361.5,-0.40861586],[38334091886,9362.4,-6.85940815],[38334269617,9362.5,-4],[38323629409,9362.545858872,-2.40497],[38334309737,9362.7,-0.10680107],[38334312380,9362.7,-3],[38325280830,9362.8,-1.75123],[38326622800,9362.8,-1.05145],[38333175230,9363,-0.0011],[38326848745,9363.2,-0.79],[38334308960,9363.206775564,-0.12],[38333920234,9363.3,-1.25318113],[38326848843,9363.4,-1.29],[38331239823,9363.4,-0.02],[38333209613,9363.4,-0.26719],[38334299964,9364,-0.05583123],[38323470224,9364.161816648,-0.12912],[38334284711,9365,-0.21346019],[38334299594,9365,-2.6757062],[38323211816,9365.073132585,-0.21262],[38334312456,9365.1,-0.11167861],[38333209612,9365.2,-0.26719],[38327770474,9365.3,-0.0073],[38334298788,9365.3,-0.3],[38334075803,9365.409831204,-0.30772637],[38334309740,9365.5,-0.10680107],[38326608767,9365.7,-2.76809],[38333920657,9365.7,-1.25848083],[38329594226,9366.6,-0.02587],[38334311813,9366.7,-4.72290945],[38316386301,9367.39258128,-2.37581],[38334302026,9367.4,-4.5],[38334228915,9367.9,-0.81725458],[38333921381,9368.1,-1.72213641],[38333175678,9368.2,-0.0011],[38334301150,9368.2,-2.654604],[38334297208,9368.3,-0.78036466],[38334309739,9368.3,-0.10680107],[38331227515,9368.7,-0.02],[38331184470,9369,-0.003975],[38334203436,9369.319616,-0.32397695],[38334269964,9369.7,-0.5],[38328386732,9370,-4.11759935],[38332719555,9370,-0.025],[38333921935,9370.5,-1.2224398],[38334258511,9370.5,-0.35],[38326848842,9370.8,-0.34],[38333985038,9370.9,-0.8551502],[38334283018,9370.9,-1],[38326848744,9371,-1.34]],5]`
	err := b.wsHandleData([]byte(pressXToJSON))
	if err != nil {
		t.Error(err)
	}
	pressXToJSON = `[23405,[7617,52.98726298,7617.1,53.601795929999994,-550.9,-0.0674,7617,8318.92961981,8257.8,7500],6]`
	err = b.wsHandleData([]byte(pressXToJSON))
	if err != nil {
		t.Error(err)
	}
}

func TestWsTradeResponse(t *testing.T) {
	b.WebsocketSubdChannels[18788] = WebsocketChanInfo{Pair: "BTCUSD", Channel: wsTrades}
	pressXToJSON := `[18788,[[412685577,1580268444802,11.1998,176.3],[412685575,1580268444802,5,176.29952759],[412685574,1580268374717,1.99069999,176.41],[412685573,1580268374717,1.00930001,176.41],[412685572,1580268358760,0.9907,176.47],[412685571,1580268324362,0.5505,176.44],[412685570,1580268297270,-0.39040819,176.39],[412685568,1580268297270,-0.39780162,176.46475676],[412685567,1580268283470,-0.09,176.41],[412685566,1580268256536,-2.31310783,176.48],[412685565,1580268256536,-0.59669217,176.49],[412685564,1580268256536,-0.9902,176.49],[412685562,1580268194474,0.9902,176.55],[412685561,1580268186215,0.1,176.6],[412685560,1580268185964,-2.17096773,176.5],[412685559,1580268185964,-1.82903227,176.51],[412685558,1580268181215,2.098914,176.53],[412685557,1580268169844,16.7302,176.55],[412685556,1580268169844,3.25,176.54],[412685555,1580268155725,0.23576115,176.45],[412685553,1580268155725,3,176.44596249],[412685552,1580268155725,3.25,176.44],[412685551,1580268155725,5,176.44],[412685550,1580268155725,0.65830078,176.41],[412685549,1580268155725,0.45063807,176.41],[412685548,1580268153825,-0.67604704,176.39],[412685547,1580268145713,2.5883,176.41],[412685543,1580268087513,12.92927,176.33],[412685542,1580268087513,0.40083,176.33],[412685533,1580268005756,-0.17096773,176.32]]]`
	err := b.wsHandleData([]byte(pressXToJSON))
	if err != nil {
		t.Error(err)
	}
}

func TestWsTickerResponse(t *testing.T) {
	b.WebsocketSubdChannels[11534] = WebsocketChanInfo{Pair: "BTCUSD", Channel: wsTicker}
	pressXToJSON := `[11534,[61.304,2228.36155358,61.305,1323.2442970500003,0.395,0.0065,61.371,50973.3020771,62.5,57.421]]`
	err := b.wsHandleData([]byte(pressXToJSON))
	if err != nil {
		t.Error(err)
	}
	b.WebsocketSubdChannels[123412] = WebsocketChanInfo{Pair: "XAUTF0:USTF0", Channel: wsTicker}
	pressXToJSON = `[123412,[61.304,2228.36155358,61.305,1323.2442970500003,0.395,0.0065,61.371,50973.3020771,62.5,57.421]]`
	err = b.wsHandleData([]byte(pressXToJSON))
	if err != nil {
		t.Error(err)
	}
	b.WebsocketSubdChannels[123413] = WebsocketChanInfo{Pair: "trade:1m:tXRPUSD", Channel: wsTicker}
	pressXToJSON = `[123413,[61.304,2228.36155358,61.305,1323.2442970500003,0.395,0.0065,61.371,50973.3020771,62.5,57.421]]`
	err = b.wsHandleData([]byte(pressXToJSON))
	if err != nil {
		t.Error(err)
	}
	b.WebsocketSubdChannels[123414] = WebsocketChanInfo{Pair: "trade:1m:fZRX:p30", Channel: wsTicker}
	pressXToJSON = `[123414,[61.304,2228.36155358,61.305,1323.2442970500003,0.395,0.0065,61.371,50973.3020771,62.5,57.421]]`
	err = b.wsHandleData([]byte(pressXToJSON))
	if err != nil {
		t.Error(err)
	}
}

func TestWsCandleResponse(t *testing.T) {
	b.WebsocketSubdChannels[343351] = WebsocketChanInfo{Pair: "BTCUSD", Channel: wsCandles}
	pressXToJSON := `[343351,[[1574698260000,7379.785503,7383.8,7388.3,7379.785503,1.68829482]]]`
	err := b.wsHandleData([]byte(pressXToJSON))
	if err != nil {
		t.Error(err)
	}
	pressXToJSON = `[343351,[1574698200000,7399.9,7379.7,7399.9,7371.8,41.63633658]]`
	err = b.wsHandleData([]byte(pressXToJSON))
	if err != nil {
		t.Error(err)
	}
}

func TestWsOrderSnapshot(t *testing.T) {
	b.WsAddSubscriptionChannel(0, "account", "N/A")
	pressXToJSON := `[0,"os",[[34930659963,null,1574955083558,"tETHUSD",1574955083558,1574955083573,0.201104,0.201104,"EXCHANGE LIMIT",null,null,null,0,"ACTIVE",null,null,120,0,0,0,null,null,null,0,0,null,null,null,"BFX",null,null,null]]]`
	err := b.wsHandleData([]byte(pressXToJSON))
	if err != nil {
		t.Error(err)
	}
	pressXToJSON = `[0,"oc",[34930659963,null,1574955083558,"tETHUSD",1574955083558,1574955354487,0.201104,0.201104,"EXCHANGE LIMIT",null,null,null,0,"CANCELED",null,null,120,0,0,0,null,null,null,0,0,null,null,null,"BFX",null,null,null]]`
	err = b.wsHandleData([]byte(pressXToJSON))
	if err != nil {
		t.Error(err)
	}
}

func TestWsNotifications(t *testing.T) {
	pressXToJSON := `[0,"n",[1575282446099,"fon-req",null,null,[41238905,null,null,null,-1000,null,null,null,null,null,null,null,null,null,0.002,2,null,null,null,null,null],null,"SUCCESS","Submitting funding bid of 1000.0 USD at 0.2000 for 2 days."]]`
	err := b.wsHandleData([]byte(pressXToJSON))
	if err != nil {
		t.Error(err)
	}

	pressXToJSON = `[0,"n",[1575287438.515,"on-req",null,null,[1185815098,null,1575287436979,"tETHUSD",1575287438515,1575287438515,-2.5,-2.5,"LIMIT",null,null,null,0,"ACTIVE",null,null,230,0,0,0,null,null,null,0,null,null,null,null,"API>BFX",null,null,null],null,"SUCCESS","Submitting limit sell order for -2.5 ETH."]]`
	err = b.wsHandleData([]byte(pressXToJSON))
	if err != nil {
		t.Error(err)
	}
}

func TestGetHistoricCandles(t *testing.T) {
	currencyPair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}
	startTime := time.Now().Add(-time.Hour * 24)
	endTime := time.Now().Add(-time.Hour * 20)
	_, err = b.GetHistoricCandles(currencyPair, asset.Spot, startTime, endTime, kline.OneHour)
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetHistoricCandles(currencyPair, asset.Spot, startTime, time.Now(), kline.OneMin*1337)
	if err == nil {
		t.Fatal(err)
	}
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	currencyPair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}
	startTime := time.Now().Add(-time.Hour * 24)
	endTime := time.Now().Add(-time.Hour * 20)
	_, err = b.GetHistoricCandlesExtended(currencyPair, asset.Spot, startTime, endTime, kline.OneHour)
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetHistoricCandlesExtended(currencyPair, asset.Spot, startTime, endTime, kline.OneMin*1337)
	if err == nil {
		t.Fatal(err)
	}
}

func TestFixCasing(t *testing.T) {
	pair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}
	ret, err := b.fixCasing(pair, asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	if ret != "tBTCUSD" {
		t.Errorf("unexpected result: %v", ret)
	}
	pair, err = currency.NewPairFromString("TBTCUSD")
	if err != nil {
		t.Fatal(err)
	}
	ret, err = b.fixCasing(pair, asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	if ret != "tBTCUSD" {
		t.Errorf("unexpected result: %v", ret)
	}
	pair, err = currency.NewPairFromString("tBTCUSD")
	if err != nil {
		t.Fatal(err)
	}
	ret, err = b.fixCasing(pair, asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	if err != nil {
		t.Fatal(err)
	}
	if ret != "tBTCUSD" {
		t.Errorf("unexpected result: %v", ret)
	}
	pair, err = currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}
	ret, err = b.fixCasing(pair, asset.Margin)
	if err != nil {
		t.Fatal(err)
	}
	if ret != "tBTC:USD" {
		t.Errorf("unexpected result: %v", ret)
	}
	pair, err = currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}
	ret, err = b.fixCasing(pair, asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	if ret != "tBTCUSD" {
		t.Errorf("unexpected result: %v", ret)
	}
	pair, err = currency.NewPairFromString("FUNETH")
	if err != nil {
		t.Fatal(err)
	}
	ret, err = b.fixCasing(pair, asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	if ret != "tFUNETH" {
		t.Errorf("unexpected result: %v", ret)
	}
	pair, err = currency.NewPairFromString("TNBUSD")
	if err != nil {
		t.Fatal(err)
	}
	ret, err = b.fixCasing(pair, asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	if ret != "tTNBUSD" {
		t.Errorf("unexpected result: %v", ret)
	}

	pair, err = currency.NewPairFromString("tTNBUSD")
	if err != nil {
		t.Fatal(err)
	}
	ret, err = b.fixCasing(pair, asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	if ret != "tTNBUSD" {
		t.Errorf("unexpected result: %v", ret)
	}
	pair, err = currency.NewPairFromStrings("fUSD", "")
	if err != nil {
		t.Fatal(err)
	}
	ret, err = b.fixCasing(pair, asset.MarginFunding)
	if err != nil {
		t.Fatal(err)
	}
	if ret != "fUSD" {
		t.Errorf("unexpected result: %v", ret)
	}
	pair, err = currency.NewPairFromStrings("USD", "")
	if err != nil {
		t.Fatal(err)
	}
	ret, err = b.fixCasing(pair, asset.MarginFunding)
	if err != nil {
		t.Fatal(err)
	}
	if ret != "fUSD" {
		t.Errorf("unexpected result: %v", ret)
	}

	pair, err = currency.NewPairFromStrings("FUSD", "")
	if err != nil {
		t.Fatal(err)
	}
	ret, err = b.fixCasing(pair, asset.MarginFunding)
	if err != nil {
		t.Fatal(err)
	}
	if ret != "fUSD" {
		t.Errorf("unexpected result: %v", ret)
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
			"1m",
		},
		{
			"OneDay",
			kline.OneDay,
			"1D",
		},
		{
			"OneWeek",
			kline.OneWeek,
			"7D",
		},
		{
			"TwoWeeks",
			kline.OneWeek * 2,
			"14D",
		},
	}

	for x := range testCases {
		test := testCases[x]
		t.Run(test.name, func(t *testing.T) {
			ret := b.FormatExchangeKlineInterval(test.interval)
			if ret != test.output {
				t.Fatalf("unexpected result return expected: %v received: %v", test.output, ret)
			}
		})
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	currencyPair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.GetRecentTrades(currencyPair, asset.Spot)
	if err != nil {
		t.Error(err)
	}

	currencyPair, err = currency.NewPairFromString("USD")
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.GetRecentTrades(currencyPair, asset.Margin)
	if err != nil {
		t.Error(err)
	}
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	currencyPair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.GetHistoricTrades(currencyPair, asset.Spot, time.Now().Add(-time.Minute*15), time.Now())
	if err != nil {
		t.Error(err)
	}

	// longer term test
	_, err = b.GetHistoricTrades(currencyPair, asset.Spot, time.Now().Add(-time.Hour*100), time.Now().Add(-time.Hour*99))
	if err != nil {
		t.Error(err)
	}
}

var testOb = orderbook.Base{
	Asks: []orderbook.Item{
		{Price: 0.05005, Amount: 0.00000500},
		{Price: 0.05010, Amount: 0.00000500},
		{Price: 0.05015, Amount: 0.00000500},
		{Price: 0.05020, Amount: 0.00000500},
		{Price: 0.05025, Amount: 0.00000500},
		{Price: 0.05030, Amount: 0.00000500},
		{Price: 0.05035, Amount: 0.00000500},
		{Price: 0.05040, Amount: 0.00000500},
		{Price: 0.05045, Amount: 0.00000500},
		{Price: 0.05050, Amount: 0.00000500},
	},
	Bids: []orderbook.Item{
		{Price: 0.05000, Amount: 0.00000500},
		{Price: 0.04995, Amount: 0.00000500},
		{Price: 0.04990, Amount: 0.00000500},
		{Price: 0.04980, Amount: 0.00000500},
		{Price: 0.04975, Amount: 0.00000500},
		{Price: 0.04970, Amount: 0.00000500},
		{Price: 0.04965, Amount: 0.00000500},
		{Price: 0.04960, Amount: 0.00000500},
		{Price: 0.04955, Amount: 0.00000500},
		{Price: 0.04950, Amount: 0.00000500},
	},
}

func TestChecksum(t *testing.T) {
	err := validateCRC32(&testOb, 190468240)
	if err != nil {
		t.Fatal(err)
	}
}

func TestReOrderbyID(t *testing.T) {
	asks := []orderbook.Item{
		{ID: 4, Price: 100, Amount: 0.00000500},
		{ID: 3, Price: 100, Amount: 0.00000500},
		{ID: 2, Price: 100, Amount: 0.00000500},
		{ID: 1, Price: 100, Amount: 0.00000500},
		{ID: 5, Price: 101, Amount: 0.00000500},
		{ID: 6, Price: 102, Amount: 0.00000500},
		{ID: 8, Price: 103, Amount: 0.00000500},
		{ID: 7, Price: 103, Amount: 0.00000500},
		{ID: 9, Price: 104, Amount: 0.00000500},
		{ID: 10, Price: 105, Amount: 0.00000500},
	}
	reOrderByID(asks)

	for i := range asks {
		if asks[i].ID != int64(i+1) {
			t.Fatal("order by ID failure")
		}
	}

	bids := []orderbook.Item{
		{ID: 4, Price: 100, Amount: 0.00000500},
		{ID: 3, Price: 100, Amount: 0.00000500},
		{ID: 2, Price: 100, Amount: 0.00000500},
		{ID: 1, Price: 100, Amount: 0.00000500},
		{ID: 5, Price: 99, Amount: 0.00000500},
		{ID: 6, Price: 98, Amount: 0.00000500},
		{ID: 8, Price: 97, Amount: 0.00000500},
		{ID: 7, Price: 97, Amount: 0.00000500},
		{ID: 9, Price: 96, Amount: 0.00000500},
		{ID: 10, Price: 95, Amount: 0.00000500},
	}
	reOrderByID(bids)

	for i := range bids {
		if bids[i].ID != int64(i+1) {
			t.Fatal("order by ID failure")
		}
	}
}
