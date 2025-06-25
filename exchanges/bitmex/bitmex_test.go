package bitmex

import (
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// Please supply your own keys here for due diligence testing
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var b *Bitmex

func TestMain(m *testing.M) {
	b = new(Bitmex)
	if err := testexch.Setup(b); err != nil {
		log.Fatalf("Bitmex Setup error: %s", err)
	}

	if apiKey != "" && apiSecret != "" {
		b.API.AuthenticatedSupport = true
		b.API.AuthenticatedWebsocketSupport = true
		b.SetCredentials(apiKey, apiSecret, "", "", "", "")
	}

	os.Exit(m.Run())
}

func TestGetFullFundingHistory(t *testing.T) {
	t.Parallel()
	_, err := b.GetFullFundingHistory(t.Context(),
		"", "", "", "", "", true, time.Now().Add(-time.Minute), time.Now())
	require.NoError(t, err)

	_, err = b.GetFullFundingHistory(t.Context(),
		"LTCUSD", "1", "", "", "", true, time.Now().Add(-time.Minute), time.Now())
	require.NoError(t, err)
}

func TestGetUrgentAnnouncement(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, b, canManipulateRealOrders)
	_, err := b.GetUrgentAnnouncement(t.Context())
	require.Error(t, err)
}

func TestGetAPIKeys(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, b, canManipulateRealOrders)
	_, err := b.GetAPIKeys(t.Context())
	require.Error(t, err)
}

func TestRemoveAPIKey(t *testing.T) {
	t.Parallel()

	_, err := b.RemoveAPIKey(t.Context(), APIKeyParams{APIKeyID: "1337"})
	require.Error(t, err)
}

func TestDisableAPIKey(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, b, canManipulateRealOrders)
	_, err := b.DisableAPIKey(t.Context(), APIKeyParams{APIKeyID: "1337"})
	require.Error(t, err)
}

func TestEnableAPIKey(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, b, canManipulateRealOrders)
	_, err := b.EnableAPIKey(t.Context(), APIKeyParams{APIKeyID: "1337"})
	require.Error(t, err)
}

func TestGetTrollboxMessages(t *testing.T) {
	t.Parallel()
	_, err := b.GetTrollboxMessages(t.Context(), ChatGetParams{Count: 1})
	require.NoError(t, err)
}

func TestSendTrollboxMessage(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, b, canManipulateRealOrders)
	_, err := b.SendTrollboxMessage(t.Context(),
		ChatSendParams{
			ChannelID: 1337,
			Message:   "Hello,World!",
		})
	require.Error(t, err)
}

func TestGetTrollboxChannels(t *testing.T) {
	t.Parallel()
	_, err := b.GetTrollboxChannels(t.Context())
	require.NoError(t, err)
}

func TestGetTrollboxConnectedUsers(t *testing.T) {
	t.Parallel()
	_, err := b.GetTrollboxConnectedUsers(t.Context())
	require.NoError(t, err)
}

func TestGetAccountExecutions(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, b, canManipulateRealOrders)
	_, err := b.GetAccountExecutions(t.Context(),
		&GenericRequestParams{})
	require.Error(t, err)
}

func TestGetAccountExecutionTradeHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, b, canManipulateRealOrders)
	_, err := b.GetAccountExecutionTradeHistory(t.Context(),
		&GenericRequestParams{})
	require.Error(t, err)
}

func TestGetFundingHistory(t *testing.T) {
	t.Parallel()
	_, err := b.GetAccountFundingHistory(t.Context())
	require.Error(t, err)
}

func TestGetInstruments(t *testing.T) {
	t.Parallel()
	_, err := b.GetInstruments(t.Context(),
		&GenericRequestParams{
			Symbol: "XRPUSD",
		})
	require.NoError(t, err)
}

func TestGetActiveInstruments(t *testing.T) {
	t.Parallel()
	_, err := b.GetActiveInstruments(t.Context(),
		&GenericRequestParams{})
	require.NoError(t, err)
}

func TestGetActiveAndIndexInstruments(t *testing.T) {
	t.Parallel()
	_, err := b.GetActiveAndIndexInstruments(t.Context())
	require.NoError(t, err)
}

func TestGetActiveIntervals(t *testing.T) {
	t.Parallel()
	_, err := b.GetActiveIntervals(t.Context())
	require.NoError(t, err)
}

func TestGetCompositeIndex(t *testing.T) {
	t.Parallel()
	_, err := b.GetCompositeIndex(t.Context(),
		".XBT", "", "", "", "", "", time.Time{}, time.Time{})
	require.NoError(t, err)
}

func TestGetIndices(t *testing.T) {
	t.Parallel()
	_, err := b.GetIndices(t.Context())
	require.NoError(t, err)
}

func TestGetInsuranceFundHistory(t *testing.T) {
	t.Parallel()
	_, err := b.GetInsuranceFundHistory(t.Context(),
		&GenericRequestParams{})
	require.NoError(t, err)
}

func TestGetLeaderboard(t *testing.T) {
	t.Parallel()

	b := new(Bitmex) //nolint:govet // Intentional shadow
	err := testexch.Setup(b)
	require.NoError(t, err, "Setup must not error")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method, "Request method should be correct")
		assert.Equal(t, "/api/v1/leaderboard", r.URL.Path, "Request path should be correct")
		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte(`[
			{
				"isRealName": true,
				"name": "Trader Joe",
				"profit": 12345.67
			},
			{
				"isRealName": false,
				"name": "CryptoKing",
				"profit": 9876.54
			}
		]`))
		assert.NoError(t, err, "Writing response to handler should not error")
	}))
	defer server.Close()

	err = b.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL+"/api/v1")
	require.NoError(t, err, "SetRunningURL must not error")

	expectedLeaderboard := []Leaderboard{
		{IsRealName: true, Name: "Trader Joe", Profit: 12345.67},
		{Name: "CryptoKing", Profit: 9876.54},
	}

	result, err := b.GetLeaderboard(t.Context(), LeaderboardGetParams{})
	require.NoError(t, err, "GetLeaderboard must not error")
	assert.Equal(t, expectedLeaderboard, result, "GetLeaderboard result should be correct")
}

func TestGetAliasOnLeaderboard(t *testing.T) {
	t.Parallel()
	_, err := b.GetAliasOnLeaderboard(t.Context())
	require.Error(t, err)
}

func TestGetLiquidationOrders(t *testing.T) {
	t.Parallel()
	_, err := b.GetLiquidationOrders(t.Context(),
		&GenericRequestParams{})
	require.NoError(t, err)
}

func TestGetCurrentNotifications(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, b, canManipulateRealOrders)
	_, err := b.GetCurrentNotifications(t.Context())
	require.Error(t, err)
}

func TestAmendOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, b, canManipulateRealOrders)
	_, err := b.AmendOrder(t.Context(), &OrderAmendParams{})
	require.Error(t, err)
}

func TestCreateOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, b, canManipulateRealOrders)
	_, err := b.CreateOrder(t.Context(),
		&OrderNewParams{
			Symbol:        "XBTM15",
			Price:         219.0,
			ClientOrderID: "mm_bitmex_1a/oemUeQ4CAJZgP3fjHsA",
			OrderQuantity: 98,
		})
	require.Error(t, err)
}

func TestCancelOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, b, canManipulateRealOrders)
	_, err := b.CancelOrders(t.Context(), &OrderCancelParams{})
	require.Error(t, err)
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, b, canManipulateRealOrders)
	_, err := b.CancelAllExistingOrders(t.Context(),
		OrderCancelAllParams{})
	require.Error(t, err)
}

func TestAmendBulkOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, b, canManipulateRealOrders)
	_, err := b.AmendBulkOrders(t.Context(), OrderAmendBulkParams{})
	require.Error(t, err)
}

func TestCreateBulkOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, b, canManipulateRealOrders)
	_, err := b.CreateBulkOrders(t.Context(), OrderNewBulkParams{})
	require.Error(t, err)
}

func TestCancelAllOrdersAfterTime(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, b, canManipulateRealOrders)
	_, err := b.CancelAllOrdersAfterTime(t.Context(),
		OrderCancelAllAfterParams{})
	require.Error(t, err)
}

func TestClosePosition(t *testing.T) {
	t.Parallel()
	_, err := b.ClosePosition(t.Context(), OrderClosePositionParams{})
	require.Error(t, err)
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := b.GetOrderbook(t.Context(),
		OrderBookGetL2Params{Symbol: "XBT"})
	require.NoError(t, err)
}

func TestGetPositions(t *testing.T) {
	t.Parallel()
	_, err := b.GetPositions(t.Context(), PositionGetParams{})
	require.Error(t, err)
}

func TestIsolatePosition(t *testing.T) {
	t.Parallel()
	_, err := b.IsolatePosition(t.Context(),
		PositionIsolateMarginParams{Symbol: "XBT"})
	require.Error(t, err)
}

func TestLeveragePosition(t *testing.T) {
	t.Parallel()
	_, err := b.LeveragePosition(t.Context(),
		PositionUpdateLeverageParams{})
	require.Error(t, err)
}

func TestUpdateRiskLimit(t *testing.T) {
	t.Parallel()
	_, err := b.UpdateRiskLimit(t.Context(),
		PositionUpdateRiskLimitParams{})
	require.Error(t, err)
}

func TestTransferMargin(t *testing.T) {
	t.Parallel()
	_, err := b.TransferMargin(t.Context(),
		PositionTransferIsolatedMarginParams{})
	require.Error(t, err)
}

func TestGetQuotesByBuckets(t *testing.T) {
	t.Parallel()
	_, err := b.GetQuotesByBuckets(t.Context(),
		&QuoteGetBucketedParams{})
	require.Error(t, err)
}

func TestGetSettlementHistory(t *testing.T) {
	t.Parallel()
	_, err := b.GetSettlementHistory(t.Context(),
		&GenericRequestParams{})
	require.NoError(t, err)
}

func TestGetStats(t *testing.T) {
	t.Parallel()
	_, err := b.GetStats(t.Context())
	require.NoError(t, err)
}

func TestGetStatsHistorical(t *testing.T) {
	t.Parallel()

	b := new(Bitmex) //nolint:govet // Intentional shadow
	err := testexch.Setup(b)
	require.NoError(t, err, "Setup must not error")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method, "Request method should be correct")
		assert.Equal(t, "/api/v1/stats/history", r.URL.Path, "Request path should be correct")
		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte(`[
			{
				"currency": "XBt",
				"date": "2023-10-26T00:00:00.000Z",
				"rootSymbol": "XBT",
				"turnover": 5000000000,
				"volume": 100000
			},
			{
				"currency": "XBt",
				"date": "2023-10-25T10:35:42.123Z",
				"rootSymbol": "XBT",
				"turnover": 4500000000,
				"volume": 90000
			}
		]`)) // Bitmex sends XBt as the currency for BTC, so we mimic it exactly
		assert.NoError(t, err, "Writing response to handler should not error")
	}))
	defer server.Close()

	err = b.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL+"/api/v1")
	require.NoError(t, err, "SetRunningURL must not error")

	expectedStats := []StatsHistory{
		{Currency: "XBt", Date: time.Date(2023, 10, 26, 0, 0, 0, 0, time.UTC), RootSymbol: "XBT", Turnover: 5000000000, Volume: 100000},
		{Currency: "XBt", Date: time.Date(2023, 10, 25, 10, 35, 42, 123000000, time.UTC), RootSymbol: "XBT", Turnover: 4500000000, Volume: 90000},
	}

	result, err := b.GetStatsHistorical(t.Context())
	require.NoError(t, err, "GetStatsHistorical must not error")
	assert.Equal(t, expectedStats, result, "GetStatsHistorical result should be correct")
}

func TestGetStatSummary(t *testing.T) {
	t.Parallel()
	_, err := b.GetStatSummary(t.Context())
	require.NoError(t, err)
}

func TestGetTrade(t *testing.T) {
	t.Parallel()
	_, err := b.GetTrade(t.Context(),
		&GenericRequestParams{
			Symbol:    "XBT",
			Reverse:   false,
			StartTime: time.Now().Add(-time.Minute).Format(time.RFC3339),
		})
	require.NoError(t, err)
}

func TestGetPreviousTrades(t *testing.T) {
	t.Parallel()
	_, err := b.GetPreviousTrades(t.Context(),
		&TradeGetBucketedParams{
			Symbol:  "XBTBTC",
			Start:   time.Now().Add(-time.Hour).Unix(),
			Columns: "open,high,low,close,volume",
		})
	require.Error(t, err)
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
	feeBuilder := setFeeBuilder()
	_, err := b.GetFeeByType(t.Context(), feeBuilder)
	require.NoError(t, err)
	if !sharedtestvalues.AreAPICredentialsSet(b) {
		assert.Equal(t, exchange.OfflineTradeFee, feeBuilder.FeeType)
	} else {
		assert.Equal(t, exchange.CryptocurrencyTradeFee, feeBuilder.FeeType)
	}
}

func TestGetFee(t *testing.T) {
	t.Parallel()
	feeBuilder := setFeeBuilder()
	// CryptocurrencyTradeFee Basic
	_, err := b.GetFee(feeBuilder)
	require.NoError(t, err)

	// CryptocurrencyTradeFee High quantity
	feeBuilder = setFeeBuilder()
	feeBuilder.Amount = 1000
	feeBuilder.PurchasePrice = 1000
	_, err = b.GetFee(feeBuilder)
	require.NoError(t, err)

	// CryptocurrencyTradeFee IsMaker
	feeBuilder = setFeeBuilder()
	feeBuilder.IsMaker = true
	_, err = b.GetFee(feeBuilder)
	require.NoError(t, err)

	// CryptocurrencyTradeFee Negative purchase price
	feeBuilder = setFeeBuilder()
	feeBuilder.PurchasePrice = -1000
	_, err = b.GetFee(feeBuilder)
	require.NoError(t, err)

	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	_, err = b.GetFee(feeBuilder)
	require.NoError(t, err)

	// CryptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyDepositFee
	_, err = b.GetFee(feeBuilder)
	require.NoError(t, err)

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	feeBuilder.FiatCurrency = currency.HKD
	_, err = b.GetFee(feeBuilder)
	require.NoError(t, err)

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.HKD
	_, err = b.GetFee(feeBuilder)
	require.NoError(t, err)
}

func TestFormatWithdrawPermissions(t *testing.T) {
	t.Parallel()
	expectedResult := exchange.AutoWithdrawCryptoWithAPIPermissionText + " & " + exchange.WithdrawCryptoWith2FAText +
		" & " + exchange.WithdrawCryptoWithEmailText + " & " + exchange.NoFiatWithdrawalsText
	withdrawPermissions := b.FormatWithdrawPermissions()
	assert.Equal(t, expectedResult, withdrawPermissions)
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	getOrdersRequest := order.MultiOrderRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
		Side:      order.AnySide,
	}

	_, err := b.GetActiveOrders(t.Context(), &getOrdersRequest)
	if sharedtestvalues.AreAPICredentialsSet(b) {
		require.NoError(t, err)
	} else {
		require.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled)
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	getOrdersRequest := order.MultiOrderRequest{
		Type:      order.AnyType,
		Pairs:     []currency.Pair{currency.NewPair(currency.LTC, currency.BTC)},
		AssetType: asset.Spot,
		Side:      order.AnySide,
	}

	_, err := b.GetOrderHistory(t.Context(), &getOrdersRequest)
	if sharedtestvalues.AreAPICredentialsSet(b) {
		require.NoError(t, err)
	} else {
		require.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled)
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, b, canManipulateRealOrders)

	orderSubmission := &order.Submit{
		Exchange: b.Name,
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
	response, err := b.SubmitOrder(t.Context(), orderSubmission)
	if sharedtestvalues.AreAPICredentialsSet(b) {
		require.NoError(t, err)
		assert.Equal(t, order.New, response.Status)
	} else {
		require.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled)
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, b, canManipulateRealOrders)

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)
	orderCancellation := &order.Cancel{
		OrderID:   "123456789012345678901234567890123456",
		AccountID: "1",
		Pair:      currencyPair,
		AssetType: asset.Futures,
	}

	err := b.CancelOrder(t.Context(), orderCancellation)
	if sharedtestvalues.AreAPICredentialsSet(b) {
		require.NoError(t, err)
	} else {
		require.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, b, canManipulateRealOrders)

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)
	orderCancellation := &order.Cancel{
		OrderID:   "123456789012345678901234567890123456",
		AccountID: "1",
		Pair:      currencyPair,
		AssetType: asset.Futures,
	}

	resp, err := b.CancelAllOrders(t.Context(), orderCancellation)
	if sharedtestvalues.AreAPICredentialsSet(b) {
		require.NoError(t, err)
		require.Empty(t, resp.Status, "CancelAllOrders must not fail to cancel orders")
	} else {
		require.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled)
	}
}

func TestUpdateAccountInfo(t *testing.T) {
	t.Parallel()
	if sharedtestvalues.AreAPICredentialsSet(b) {
		_, err := b.UpdateAccountInfo(t.Context(), asset.Spot)
		require.NoError(t, err)

		_, err = b.UpdateAccountInfo(t.Context(), asset.Futures)
		require.NoError(t, err)
	} else {
		_, err := b.UpdateAccountInfo(t.Context(), asset.Spot)
		require.Error(t, err)

		_, err = b.UpdateAccountInfo(t.Context(), asset.Futures)
		require.Error(t, err)
	}
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, b, canManipulateRealOrders)
	_, err := b.ModifyOrder(t.Context(),
		&order.Modify{OrderID: "1337", AssetType: asset.Futures})
	require.Error(t, err)
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, b, canManipulateRealOrders)

	withdrawCryptoRequest := withdraw.Request{
		Exchange: b.Name,
		Crypto: withdraw.CryptoRequest{
			Address: core.BitcoinDonationAddress,
		},
		Amount:          -1,
		Currency:        currency.BTC,
		Description:     "WITHDRAW IT ALL",
		OneTimePassword: 696969,
	}

	_, err := b.WithdrawCryptocurrencyFunds(t.Context(), &withdrawCryptoRequest)
	if sharedtestvalues.AreAPICredentialsSet(b) {
		require.NoError(t, err)
	} else {
		require.Error(t, err)
	}
}

func TestWithdrawFiat(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, b, canManipulateRealOrders)

	withdrawFiatRequest := withdraw.Request{}
	_, err := b.WithdrawFiatFunds(t.Context(), &withdrawFiatRequest)
	require.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestWithdrawInternationalBank(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, b, canManipulateRealOrders)

	withdrawFiatRequest := withdraw.Request{}
	_, err := b.WithdrawFiatFundsToInternationalBank(t.Context(),
		&withdrawFiatRequest)
	require.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	if sharedtestvalues.AreAPICredentialsSet(b) {
		_, err := b.GetDepositAddress(t.Context(), currency.BTC, "", "")
		require.NoError(t, err)
	} else {
		_, err := b.GetDepositAddress(t.Context(), currency.BTC, "", "")
		require.Error(t, err)
	}
}

// TestWsAuth dials websocket, sends login request.
func TestWsAuth(t *testing.T) {
	t.Parallel()
	if !b.Websocket.IsEnabled() && !b.API.AuthenticatedWebsocketSupport || !sharedtestvalues.AreAPICredentialsSet(b) {
		t.Skip(websocket.ErrWebsocketNotEnabled.Error())
	}
	var dialer gws.Dialer
	err := b.Websocket.Conn.Dial(t.Context(), &dialer, http.Header{})
	require.NoError(t, err)

	go b.wsReadData()
	err = b.websocketSendAuth(t.Context())
	require.NoError(t, err)
	timer := time.NewTimer(sharedtestvalues.WebsocketResponseDefaultTimeout)
	select {
	case resp := <-b.Websocket.DataHandler:
		sub, ok := resp.(WebsocketSubscribeResp)
		if !ok {
			t.Fatal("unable to type assert WebsocketSubscribeResp")
		}
		if !sub.Success {
			t.Error("Expected successful subscription")
		}
	case <-timer.C:
		t.Error("Have not received a response")
	}
	timer.Stop()
}

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	err := b.UpdateTradablePairs(t.Context(), true)
	require.NoError(t, err)
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
	require.NoError(t, err)
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
	require.NoError(t, err)
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
	require.NoError(t, err)
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
	require.NoError(t, err)
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
        {"symbol":"ETHUSD","id":17999992000,"side":"Sell","size":100,"price":80,"timestamp":"2017-04-04T22:16:38.461Z"},
        {"symbol":"ETHUSD","id":17999993000,"side":"Sell","size":20,"price":70},
        {"symbol":"ETHUSD","id":17999994000,"side":"Sell","size":10,"price":60},
        {"symbol":"ETHUSD","id":17999995000,"side":"Buy","size":10,"price":50},
        {"symbol":"ETHUSD","id":17999996000,"side":"Buy","size":20,"price":40},
        {"symbol":"ETHUSD","id":17999997000,"side":"Buy","size":100,"price":30}
      ]}`)
	err := b.wsHandleData(pressXToJSON)
	require.NoError(t, err)

	pressXToJSON = []byte(`{
      "table":"orderBookL2_25",
      "action":"update",
      "data":[
        {"symbol":"ETHUSD","id":17999995000,"side":"Buy","size":5,"timestamp":"2017-04-04T22:16:38.461Z"}
      ]}`)
	err = b.wsHandleData(pressXToJSON)
	require.NoError(t, err)

	pressXToJSON = []byte(`{
      "table":"orderBookL2_25",
      "action":"update",
      "data":[]}`)
	err = b.wsHandleData(pressXToJSON)
	require.ErrorContains(t, err, "empty orderbook")

	pressXToJSON = []byte(`{
      "table":"orderBookL2_25",
      "action":"delete",
      "data":[
        {"symbol":"ETHUSD","id":17999995000,"side":"Buy","timestamp":"2017-04-04T22:16:38.461Z"}
      ]}`)
	err = b.wsHandleData(pressXToJSON)
	require.NoError(t, err)

	pressXToJSON = []byte(`{
      "table":"orderBookL2_25",
      "action":"delete",
      "data":[
        {"symbol":"ETHUSD","id":17999995000,"side":"Buy","timestamp":"2017-04-04T22:16:38.461Z"}
      ]}`)
	err = b.wsHandleData(pressXToJSON)
	assert.ErrorIs(t, err, orderbook.ErrOrderbookInvalid)
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
	require.NoError(t, err)

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
	require.NoError(t, err)
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
	require.NoError(t, err)
}

func TestWsTrades(t *testing.T) {
	t.Parallel()
	b := new(Bitmex) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes
	require.NoError(t, testexch.Setup(b), "Test instance Setup must not error")
	b.SetSaveTradeDataStatus(true)
	msg := []byte(`{"table":"trade","action":"insert","data":[{"timestamp":"2020-02-17T01:35:36.442Z","symbol":"ETHUSD","side":"Sell","size":100,"price":258.3,"tickDirection":"MinusTick","trdMatchID":"c427f7a0-6b26-1e10-5c4e-1bd74daf2a73","grossValue":2583000,"homeNotional":0.9904912836767037,"foreignNotional":255.84389857369254},{"timestamp":"2020-02-17T01:35:36.442Z","symbol":"ETHUSD","side":"Sell","size":100,"price":258.3,"tickDirection":"ZeroMinusTick","trdMatchID":"95eb9155-b58c-70e9-44b7-34efe50302e0","grossValue":2583000,"homeNotional":0.9904912836767037,"foreignNotional":255.84389857369254},{"timestamp":"2020-02-17T01:35:36.442Z","symbol":"ETHUSD","side":"Sell","size":100,"price":258.3,"tickDirection":"ZeroMinusTick","trdMatchID":"e607c187-f25c-86bc-cb39-8afff7aaf2d9","grossValue":2583000,"homeNotional":0.9904912836767037,"foreignNotional":255.84389857369254},{"timestamp":"2020-02-17T01:35:36.442Z","symbol":"ETHUSD","side":"Sell","size":17,"price":258.3,"tickDirection":"ZeroMinusTick","trdMatchID":"0f076814-a57d-9a59-8063-ad6b823a80ac","grossValue":439110,"homeNotional":0.1683835182250396,"foreignNotional":43.49346275752773},{"timestamp":"2020-02-17T01:35:36.442Z","symbol":"ETHUSD","side":"Sell","size":100,"price":258.25,"tickDirection":"MinusTick","trdMatchID":"f4ef3dfd-51c4-538f-37c1-e5071ba1c75d","grossValue":2582500,"homeNotional":0.9904912836767037,"foreignNotional":255.79437400950872},{"timestamp":"2020-02-17T01:35:36.442Z","symbol":"ETHUSD","side":"Sell","size":100,"price":258.25,"tickDirection":"ZeroMinusTick","trdMatchID":"81ef136b-8f4a-b1cf-78a8-fffbfa89bf40","grossValue":2582500,"homeNotional":0.9904912836767037,"foreignNotional":255.79437400950872},{"timestamp":"2020-02-17T01:35:36.442Z","symbol":"ETHUSD","side":"Sell","size":100,"price":258.25,"tickDirection":"ZeroMinusTick","trdMatchID":"65a87e8c-7563-34a4-d040-94e8513c5401","grossValue":2582500,"homeNotional":0.9904912836767037,"foreignNotional":255.79437400950872},{"timestamp":"2020-02-17T01:35:36.442Z","symbol":"ETHUSD","side":"Sell","size":15,"price":258.25,"tickDirection":"ZeroMinusTick","trdMatchID":"1d11a74e-a157-3f33-036d-35a101fba50b","grossValue":387375,"homeNotional":0.14857369255150554,"foreignNotional":38.369156101426306},{"timestamp":"2020-02-17T01:35:36.442Z","symbol":"ETHUSD","side":"Sell","size":1,"price":258.25,"tickDirection":"ZeroMinusTick","trdMatchID":"40d49df1-f018-f66f-4ca5-31d4997641d7","grossValue":25825,"homeNotional":0.009904912836767036,"foreignNotional":2.5579437400950873},{"timestamp":"2020-02-17T01:35:36.442Z","symbol":"ETHUSD","side":"Sell","size":100,"price":258.2,"tickDirection":"MinusTick","trdMatchID":"36135b51-73e5-c007-362b-a55be5830c6b","grossValue":2582000,"homeNotional":0.9904912836767037,"foreignNotional":255.7448494453249},{"timestamp":"2020-02-17T01:35:36.442Z","symbol":"ETHUSD","side":"Sell","size":100,"price":258.2,"tickDirection":"ZeroMinusTick","trdMatchID":"6ee19edb-99aa-3030-ba63-933ffb347ade","grossValue":2582000,"homeNotional":0.9904912836767037,"foreignNotional":255.7448494453249},{"timestamp":"2020-02-17T01:35:36.442Z","symbol":"ETHUSD","side":"Sell","size":100,"price":258.2,"tickDirection":"ZeroMinusTick","trdMatchID":"d44be603-cdb8-d676-e3e2-f91fb12b2a70","grossValue":2582000,"homeNotional":0.9904912836767037,"foreignNotional":255.7448494453249},{"timestamp":"2020-02-17T01:35:36.442Z","symbol":"ETHUSD","side":"Sell","size":5,"price":258.2,"tickDirection":"ZeroMinusTick","trdMatchID":"a14b43b3-50b4-c075-c54d-dfb0165de33d","grossValue":129100,"homeNotional":0.04952456418383518,"foreignNotional":12.787242472266245},{"timestamp":"2020-02-17T01:35:36.442Z","symbol":"ETHUSD","side":"Sell","size":8,"price":258.2,"tickDirection":"ZeroMinusTick","trdMatchID":"3c30e175-5194-320c-8f8c-01636c2f4a32","grossValue":206560,"homeNotional":0.07923930269413629,"foreignNotional":20.45958795562599},{"timestamp":"2020-02-17T01:35:36.442Z","symbol":"ETHUSD","side":"Sell","size":50,"price":258.2,"tickDirection":"ZeroMinusTick","trdMatchID":"5b803378-760b-4919-21fc-bfb275d39ace","grossValue":1291000,"homeNotional":0.49524564183835185,"foreignNotional":127.87242472266244},{"timestamp":"2020-02-17T01:35:36.442Z","symbol":"ETHUSD","side":"Sell","size":244,"price":258.2,"tickDirection":"ZeroMinusTick","trdMatchID":"cf57fec1-c444-b9e5-5e2d-4fb643f4fdb7","grossValue":6300080,"homeNotional":2.416798732171157,"foreignNotional":624.0174326465927}]}`)
	require.NoError(t, b.wsHandleData(msg), "Must not error handling a standard stream of trades")

	msg = []byte(`{"table":"trade","action":"insert","data":[{"timestamp":"2020-02-17T01:35:36.442Z","symbol":".BGCT","size":14,"price":258.2,"side":"sell"}]}`)
	require.ErrorIs(t, b.wsHandleData(msg), exchange.ErrSymbolCannotBeMatched, "Must error correctly with an unknown symbol")

	msg = []byte(`{"table":"trade","action":"insert","data":[{"timestamp":"2020-02-17T01:35:36.442Z","symbol":".BGCT","size":0,"price":258.2,"side":"sell"}]}`)
	require.NoError(t, b.wsHandleData(msg), "Must not error that symbol is unknown when index trade is ignored due to zero size")
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	err := b.UpdateTradablePairs(t.Context(), false)
	require.NoError(t, err)
	currencyPair := b.CurrencyPairs.Pairs[asset.Futures].Available[0]
	_, err = b.GetRecentTrades(t.Context(), currencyPair, asset.Futures)
	require.NoError(t, err)
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	err := b.UpdateTradablePairs(t.Context(), false)
	require.NoError(t, err)
	currencyPair := b.CurrencyPairs.Pairs[asset.Futures].Available[0]
	_, err = b.GetHistoricTrades(t.Context(), currencyPair, asset.Futures, time.Now().Add(-time.Minute), time.Now())
	require.NoError(t, err)
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	cp := currency.NewPair(currency.ETH, currency.USD)
	_, err := b.UpdateTicker(t.Context(), cp, asset.PerpetualContract)
	require.NoError(t, err)
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	err := b.UpdateTickers(t.Context(), asset.PerpetualContract)
	require.NoError(t, err)
}

func TestNormalizeWalletInfo(t *testing.T) {
	w := &WalletInfo{
		Currency: "XBt",
		Amount:   1e+08,
	}

	normalizeWalletInfo(w)

	assert.Equal(t, "BTC", w.Currency, "Currency should be correct")
	assert.Equal(t, 1.0, w.Amount, "Amount should be correct")
}

func TestGetOrderType(t *testing.T) {
	t.Parallel()
	_, err := b.getOrderType(0)
	assert.ErrorIs(t, err, order.ErrTypeIsInvalid)

	o, err := b.getOrderType(1)
	require.NoError(t, err)
	assert.Equal(t, order.Market, o)
}

func TestGetActionFromString(t *testing.T) {
	t.Parallel()
	_, err := b.GetActionFromString("meow")
	assert.ErrorIs(t, err, orderbook.ErrInvalidAction)

	action, err := b.GetActionFromString("update")
	require.NoError(t, err)
	assert.Equal(t, orderbook.UpdateAction, action)

	action, err = b.GetActionFromString("delete")
	require.NoError(t, err)
	assert.Equal(t, orderbook.DeleteAction, action)

	action, err = b.GetActionFromString("insert")
	require.NoError(t, err)
	assert.Equal(t, orderbook.InsertAction, action)

	action, err = b.GetActionFromString("update/insert")
	require.NoError(t, err)
	assert.Equal(t, orderbook.UpdateOrInsertAction, action)
}

func TestGetAccountFundingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetAccountFundingHistory(t.Context())
	require.NoError(t, err)
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	_, err := b.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Spot)
	require.NoError(t, err)
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	_, err := b.GetOrderInfo(t.Context(), "1234", currency.NewBTCUSD(), asset.Spot)
	require.NoError(t, err)
}

func TestCancelBatchOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.CancelBatchOrders(t.Context(), []order.Cancel{
		{
			OrderID:   "1234",
			AssetType: asset.Spot,
			Pair:      currency.NewBTCUSD(),
		},
	})
	require.NoError(t, err)
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	_, err := b.GetFuturesContractDetails(t.Context(), asset.Spot)
	assert.ErrorIs(t, err, futures.ErrNotFuturesAsset)

	_, err = b.GetFuturesContractDetails(t.Context(), asset.USDTMarginedFutures)
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = b.GetFuturesContractDetails(t.Context(), asset.Futures)
	assert.NoError(t, err)

	_, err = b.GetFuturesContractDetails(t.Context(), asset.PerpetualContract)
	assert.NoError(t, err)
}

func TestGetLatestFundingRates(t *testing.T) {
	t.Parallel()
	_, err := b.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset:                asset.USDTMarginedFutures,
		Pair:                 currency.NewBTCUSDT(),
		IncludePredictedRate: true,
	})
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported)

	_, err = b.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset: asset.Futures,
		Pair:  currency.NewPair(currency.BTC, currency.KLAY),
	})
	assert.ErrorIs(t, err, futures.ErrNotPerpetualFuture)

	_, err = b.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset: asset.PerpetualContract,
	})
	require.NoError(t, err)

	cp, err := currency.NewPairFromString("ETHUSD")
	require.NoError(t, err)
	_, err = b.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset: asset.PerpetualContract,
		Pair:  cp,
	})
	require.NoError(t, err)
}

func TestIsPerpetualFutureCurrency(t *testing.T) {
	t.Parallel()
	isPerp, err := b.IsPerpetualFutureCurrency(asset.Futures, currency.NewBTCUSD())
	require.NoError(t, err)
	require.False(t, isPerp)

	isPerp, err = b.IsPerpetualFutureCurrency(asset.PerpetualContract, currency.NewBTCUSD())
	require.NoError(t, err)
	require.True(t, isPerp)
}

func TestGetOpenInterest(t *testing.T) {
	t.Parallel()
	cp1 := currency.NewPair(currency.XBT, currency.USD)
	cp2 := currency.NewPair(currency.DOGE, currency.USD)
	sharedtestvalues.SetupCurrencyPairsForExchangeAsset(t, b, asset.PerpetualContract, cp1, cp2)

	resp, err := b.GetOpenInterest(t.Context(), key.PairAsset{
		Base:  currency.XBT.Item,
		Quote: currency.USD.Item,
		Asset: asset.PerpetualContract,
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)

	resp, err = b.GetOpenInterest(t.Context(),
		key.PairAsset{
			Base:  currency.XBT.Item,
			Quote: currency.USD.Item,
			Asset: asset.PerpetualContract,
		},
		key.PairAsset{
			Base:  currency.DOGE.Item,
			Quote: currency.USD.Item,
			Asset: asset.PerpetualContract,
		})
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)

	resp, err = b.GetOpenInterest(t.Context())
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)

	_, err = b.GetOpenInterest(t.Context(), key.PairAsset{
		Base:  currency.BTC.Item,
		Quote: currency.USDT.Item,
		Asset: asset.Spot,
	})
	assert.ErrorIs(t, err, asset.ErrNotSupported)
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, b)
	for _, a := range b.GetAssetTypes(false) {
		pairs, err := b.CurrencyPairs.GetPairs(a, false)
		require.NoErrorf(t, err, "cannot get pairs for %s", a)
		require.NotEmptyf(t, pairs, "no pairs for %s", a)
		resp, err := b.GetCurrencyTradeURL(t.Context(), a, pairs[0])
		require.NoError(t, err)
		assert.NotEmpty(t, resp)
	}
}

func TestGenerateSubscriptions(t *testing.T) {
	t.Parallel()

	b := new(Bitmex)
	require.NoError(t, testexch.Setup(b), "Test instance Setup must not error")

	p := currency.Pairs{
		currency.NewPair(currency.ETH, currency.USD),
		currency.NewPair(currency.BCH, currency.NewCode("Z19")),
	}

	exp := subscription.List{
		{QualifiedChannel: bitmexWSOrderbookL2 + ":" + p[1].String(), Channel: bitmexWSOrderbookL2, Asset: asset.Futures, Pairs: p[1:2]},
		{QualifiedChannel: bitmexWSOrderbookL2 + ":" + p[0].String(), Channel: bitmexWSOrderbookL2, Asset: asset.PerpetualContract, Pairs: p[:1]},
		{QualifiedChannel: bitmexWSTrade + ":" + p[1].String(), Channel: bitmexWSTrade, Asset: asset.Futures, Pairs: p[1:2]},
		{QualifiedChannel: bitmexWSTrade + ":" + p[0].String(), Channel: bitmexWSTrade, Asset: asset.PerpetualContract, Pairs: p[:1]},
		{QualifiedChannel: bitmexWSAffiliate, Channel: bitmexWSAffiliate, Authenticated: true},
		{QualifiedChannel: bitmexWSOrder, Channel: bitmexWSOrder, Authenticated: true},
		{QualifiedChannel: bitmexWSMargin, Channel: bitmexWSMargin, Authenticated: true},
		{QualifiedChannel: bitmexWSTransact, Channel: bitmexWSTransact, Authenticated: true},
		{QualifiedChannel: bitmexWSWallet, Channel: bitmexWSWallet, Authenticated: true},
		{QualifiedChannel: bitmexWSExecution + ":" + p[0].String(), Channel: bitmexWSExecution, Authenticated: true, Asset: asset.PerpetualContract, Pairs: p[:1]},
		{QualifiedChannel: bitmexWSPosition + ":" + p[0].String(), Channel: bitmexWSPosition, Authenticated: true, Asset: asset.PerpetualContract, Pairs: p[:1]},
	}

	b.Websocket.SetCanUseAuthenticatedEndpoints(true)
	subs, err := b.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	testsubs.EqualLists(t, exp, subs)

	for _, a := range b.GetAssetTypes(true) {
		require.NoErrorf(t, b.CurrencyPairs.SetAssetEnabled(a, false), "SetAssetEnabled must not error for %s", a)
	}
	_, err = b.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error when no assets are enabled")
}

func TestSubscribe(t *testing.T) {
	t.Parallel()
	b := new(Bitmex)
	require.NoError(t, testexch.Setup(b), "Test instance Setup must not error")
	subs, err := b.generateSubscriptions() // Note: We grab this before it's overwritten by SetupWs
	require.NoError(t, err, "generateSubscriptions must not error")
	testexch.SetupWs(t, b)
	err = b.Subscribe(subs)
	require.NoError(t, err, "Subscribe must not error")
	for _, s := range subs {
		assert.Equalf(t, subscription.SubscribedState, s.State(), "%s state should be subscribed", s.QualifiedChannel)
	}
	err = b.Unsubscribe(subs)
	require.NoError(t, err, "Unsubscribe must not error")
	for _, s := range subs {
		assert.Equalf(t, subscription.UnsubscribedState, s.State(), "%s state should be unsusbscribed", s.QualifiedChannel)
	}

	err = b.Subscribe(subscription.List{{QualifiedChannel: "wibble", Channel: "wibble", Asset: asset.Spot}})
	require.ErrorContains(t, err, "Unknown table: wibble", "Subscribe must receive errors through websocket.Match on request json")
}
