package htx

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestQuerySwapIndexPriceInfo(t *testing.T) {
	t.Parallel()
	_, err := e.QuerySwapIndexPriceInfo(t.Context(), btcusdPair)
	require.NoError(t, err)
}

func TestSwapOpenInterestInformation(t *testing.T) {
	t.Parallel()
	_, err := e.SwapOpenInterestInformation(t.Context(), btcusdPair)
	require.NoError(t, err)
}

func TestGetSwapMarketDepth(t *testing.T) {
	t.Parallel()
	_, err := e.GetSwapMarketDepth(t.Context(), btcusdPair, "step0")
	require.NoError(t, err)
}

func TestGetSwapKlineData(t *testing.T) {
	t.Parallel()
	r, err := e.GetSwapKlineData(t.Context(), btcusdPair, "5min", 5, time.Now().Add(-time.Hour), time.Now())
	require.NoError(t, err)
	assert.NotEmpty(t, r.Data, "GetSwapKlineData should return some data")
}

func TestGetSwapMarketOverview(t *testing.T) {
	t.Parallel()
	_, err := e.GetSwapMarketOverview(t.Context(), btcusdPair)
	require.NoError(t, err)
}

func TestGetLastTrade(t *testing.T) {
	t.Parallel()
	_, err := e.GetLastTrade(t.Context(), btcusdPair)
	require.NoError(t, err)
}

func TestGetBatchTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetBatchTrades(t.Context(), btcusdPair, 5)
	require.NoError(t, err)
}

func TestGetTieredAjustmentFactorInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetTieredAjustmentFactorInfo(t.Context(), btcusdPair)
	require.NoError(t, err)
}

func TestGetOpenInterestInfo(t *testing.T) {
	t.Parallel()
	updatePairsOnce(t, e)
	_, err := e.GetOpenInterestInfo(t.Context(), btcusdPair, "5min", "cryptocurrency", 50)
	require.NoError(t, err)
}

func TestGetTraderSentimentIndexAccount(t *testing.T) {
	t.Parallel()
	_, err := e.GetTraderSentimentIndexAccount(t.Context(), btcusdPair, "5min")
	require.NoError(t, err)
}

func TestGetTraderSentimentIndexPosition(t *testing.T) {
	t.Parallel()
	_, err := e.GetTraderSentimentIndexPosition(t.Context(), btcusdPair, "5min")
	require.NoError(t, err)
}

func TestGetLiquidationOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetLiquidationOrders(t.Context(), btcusdPair, "closed", time.Now().AddDate(0, 0, -2), time.Now(), "", 0)
	assert.NoError(t, err, "GetLiquidationOrders should not error")
}

func TestGetPremiumIndexKlineData(t *testing.T) {
	t.Parallel()
	_, err := e.GetPremiumIndexKlineData(t.Context(), btcusdPair, "5min", 15)
	require.NoError(t, err)
}

func TestGetEstimatedFundingRates(t *testing.T) {
	t.Parallel()
	_, err := e.GetEstimatedFundingRates(t.Context(), btcusdPair, "5min", 15)
	require.NoError(t, err)

	_, err = e.GetEstimatedFundingRates(t.Context(), btcusdPair, "invalid", 15)
	require.Error(t, err, "GetEstimatedFundingRates must reject invalid period")
}

func TestGetBasisData(t *testing.T) {
	t.Parallel()
	_, err := e.GetBasisData(t.Context(), btcusdPair, "5min", "close", 5)
	require.NoError(t, err)
}

func TestGetSystemStatusInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetSystemStatusInfo(t.Context(), btcusdPair)
	require.NoError(t, err)
}

func TestGetSwapPriceLimits(t *testing.T) {
	t.Parallel()
	_, err := e.GetSwapPriceLimits(t.Context(), btcusdPair)
	require.NoError(t, err)
}

func TestGetSwapAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSwapAccountInfo(t.Context(), ethusdPair)
	require.NoError(t, err)
}

func TestGetSwapPositionsInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSwapPositionsInfo(t.Context(), ethusdPair)
	require.NoError(t, err)
}

func TestGetSwapAssetsAndPositions(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSwapAssetsAndPositions(t.Context(), ethusdPair)
	require.NoError(t, err)
}

func TestGetSwapAllSubAccAssets(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSwapAllSubAccAssets(t.Context(), ethusdPair)
	require.NoError(t, err)
}

func TestGetSubAccPositionInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSubAccPositionInfo(t.Context(), ethusdPair, 0)
	require.NoError(t, err)
}

func TestSwapSingleSubAccAssets(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	h.API.AuthenticatedSupport = true
	_, err := h.SwapSingleSubAccAssets(t.Context(), ethusdPair, 123)
	require.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "SwapSingleSubAccAssets must return credentials error")
}

func TestGetAccountFinancialRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetAccountFinancialRecords(t.Context(), ethusdPair, "3,4", 2, 0, 0)
	require.NoError(t, err)
}

func TestGetSwapSettlementRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	r, err := e.GetSwapSettlementRecords(t.Context(), ethusdPair, time.Now().AddDate(0, -1, 0), time.Now(), 0, 0)
	require.NoError(t, err)
	assert.NotEmpty(t, r.Data, "GetSwapSettlementRecords should return some data")
}

func TestGetAvailableLeverage(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetAvailableLeverage(t.Context(), ethusdPair)
	require.NoError(t, err)
}

func TestGetSwapOrderLimitInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSwapOrderLimitInfo(t.Context(), ethusdPair, "limit")
	require.NoError(t, err)
}

func TestGetSwapTradingFeeInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSwapTradingFeeInfo(t.Context(), ethusdPair)
	require.NoError(t, err)
}

func TestGetSwapTransferLimitInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSwapTransferLimitInfo(t.Context(), ethusdPair)
	require.NoError(t, err)
}

func TestGetSwapPositionLimitInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSwapPositionLimitInfo(t.Context(), ethusdPair)
	require.NoError(t, err)
}

func TestAccountTransferData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.AccountTransferData(t.Context(), ethusdPair, "123", "master_to_sub", 15)
	require.NoError(t, err)
}

func TestAccountTransferRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.AccountTransferRecords(t.Context(), ethusdPair, "master_to_sub", 12, 0, 0)
	require.NoError(t, err)
}

func TestPlaceSwapOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.PlaceSwapOrders(t.Context(), ethusdPair, "", "buy", "open", "limit", 0.01, 1, 1)
	require.NoError(t, err)
}

func TestPlaceSwapBatchOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	var req BatchOrderRequestType
	order1 := batchOrderData{
		ContractCode:   "ETH-USD",
		ClientOrderID:  "",
		Price:          5,
		Volume:         1,
		Direction:      "buy",
		Offset:         "open",
		LeverageRate:   1,
		OrderPriceType: "limit",
	}
	order2 := batchOrderData{
		ContractCode:   "BTC-USD",
		ClientOrderID:  "",
		Price:          2.5,
		Volume:         1,
		Direction:      "buy",
		Offset:         "open",
		LeverageRate:   1,
		OrderPriceType: "limit",
	}
	req.Data = append(req.Data, order1, order2)

	_, err := e.PlaceSwapBatchOrders(t.Context(), req)
	require.NoError(t, err)
}

func TestCancelSwapOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.CancelSwapOrder(t.Context(), "test123", "", ethusdPair)
	require.NoError(t, err)
}

func TestCancelAllSwapOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.CancelAllSwapOrders(t.Context(), ethusdPair)
	require.NoError(t, err)
}

func TestPlaceLightningCloseOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.PlaceLightningCloseOrder(t.Context(), ethusdPair, "buy", "lightning", 5, 1)
	require.NoError(t, err)
}

func TestGetSwapOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSwapOrderInfo(t.Context(), ethusdPair, "123", "")
	require.NoError(t, err)
}

func TestGetSwapOrderDetails(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSwapOrderDetails(t.Context(), ethusdPair, "123", "10", "cancelledOrder", 0, 0)
	require.NoError(t, err)
}

func TestGetSwapOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSwapOpenOrders(t.Context(), ethusdPair, 0, 0)
	require.NoError(t, err)
}

func TestGetSwapOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetSwapOrderHistory(t.Context(), ethusdPair, "all", "all", []order.Status{order.PartiallyCancelled, order.Active}, 25, 0, 0)
	require.ErrorIs(t, err, errInvalidCreateDate, "GetSwapOrderHistory must reject lookbacks over two days")
}

func TestGetSwapOrderHistoryByTimeRange(t *testing.T) {
	t.Parallel()
	body := make(chan []byte, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload, err := io.ReadAll(r.Body)
		assert.NoError(t, err, "request body should be readable")
		body <- payload
		_, _ = w.Write([]byte(`{"code":200,"data":[{"query_id":12,"order_id":34,"order_id_str":"34","contract_code":"ETH-USD","direction":"buy","order_price_type":"limit","status":6}]}`))
	}))
	t.Cleanup(server.Close)

	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	h.API.AuthenticatedSupport = true
	h.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
	require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestFutures.String(), server.URL), "futures endpoint must be set")
	startTime := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
	endTime := startTime.Add(48 * time.Hour)
	resp, err := h.GetSwapOrderHistoryByTimeRange(t.Context(), ethusdPair, "all", "all", nil, startTime, endTime, 11, 50)
	require.NoError(t, err, "GetSwapOrderHistoryByTimeRange must not error")
	require.Len(t, resp.Data.Orders, 1, "decoded order history must be returned")
	assert.Equal(t, int64(12), resp.Data.Orders[0].QueryID, "query ID should decode")

	var requestBody map[string]any
	require.NoError(t, json.Unmarshal(<-body, &requestBody), "request body must decode")
	assert.Equal(t, float64(startTime.UnixMilli()), requestBody["start_time"], "start time should be preserved")
	assert.Equal(t, float64(endTime.UnixMilli()), requestBody["end_time"], "end time should be preserved")
	assert.Equal(t, v3HistoryDirectionNext, requestBody["direct"], "pagination direction should move forwards")
	assert.Equal(t, float64(11), requestBody["from_id"], "cursor should be preserved")
	assert.Equal(t, float64(50), requestBody["limit"], "limit should be preserved")

	_, err = h.GetSwapOrderHistoryByTimeRange(t.Context(), ethusdPair, "all", "all", nil, endTime, startTime, 0, 50)
	require.ErrorIs(t, err, errStartTimeAfterEndTime, "GetSwapOrderHistoryByTimeRange must reject reversed intervals")
}

func TestGetSwapTradeHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSwapTradeHistory(t.Context(), ethusdPair, "liquidateShort", 2, 0, 0)
	require.NoError(t, err)
}

func TestPlaceSwapTriggerOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.PlaceSwapTriggerOrder(t.Context(), ethusdPair, "greaterOrEqual", "buy", "open", "optimal_5", 5, 3, 1, 1)
	require.NoError(t, err)
}

func TestCancelSwapTriggerOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.CancelSwapTriggerOrder(t.Context(), ethusdPair, "test123")
	require.NoError(t, err)
}

func TestCancelAllSwapTriggerOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.CancelAllSwapTriggerOrders(t.Context(), ethusdPair)
	require.NoError(t, err)
}

func TestGetSwapTriggerOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetSwapTriggerOrderHistory(t.Context(), ethusdPair, "open", "all", 15, 0, 0)
	require.NoError(t, err)
}

func TestGetSwapMarkets(t *testing.T) {
	t.Parallel()
	_, err := e.GetSwapMarkets(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err)
}

func TestGetSwapFundingRates(t *testing.T) {
	t.Parallel()
	_, err := e.GetSwapFundingRates(t.Context())
	require.NoError(t, err)
}

func TestGetSwapFundingRate(t *testing.T) {
	t.Parallel()
	_, err := e.GetSwapFundingRate(t.Context(), btcusdPair)
	require.NoError(t, err, "GetSwapFundingRate must not error")
}

func TestCoinMarginedAuthenticatedEndpoints(t *testing.T) {
	t.Parallel()
	contractCode := currency.NewPairWithDelimiter("ETH", "USD", "-")
	startTime := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name          string
		path          string
		expectedField string
		expectedValue any
		call          func(*Exchange) error
	}{
		{
			name: "GetSwapAccountInfo",
			path: "/swap-api/v1/swap_account_info",
			call: func(h *Exchange) error {
				_, err := h.GetSwapAccountInfo(t.Context(), contractCode)
				return err
			},
		},
		{
			name: "GetSwapPositionsInfo",
			path: "/swap-api/v1/swap_position_info",
			call: func(h *Exchange) error {
				_, err := h.GetSwapPositionsInfo(t.Context(), contractCode)
				return err
			},
		},
		{
			name: "GetSwapAssetsAndPositions",
			path: "/swap-api/v1/swap_account_position_info",
			call: func(h *Exchange) error {
				_, err := h.GetSwapAssetsAndPositions(t.Context(), contractCode)
				return err
			},
		},
		{
			name: "GetSwapAllSubAccAssets",
			path: "/swap-api/v1/swap_sub_account_list",
			call: func(h *Exchange) error {
				_, err := h.GetSwapAllSubAccAssets(t.Context(), contractCode)
				return err
			},
		},
		{
			name: "SwapSingleSubAccAssets",
			path: "/swap-api/v1/swap_sub_account_info",
			call: func(h *Exchange) error {
				_, err := h.SwapSingleSubAccAssets(t.Context(), contractCode, 123)
				return err
			},
		},
		{
			name: "GetSubAccPositionInfo",
			path: "/swap-api/v1/swap_sub_position_info",
			call: func(h *Exchange) error {
				_, err := h.GetSubAccPositionInfo(t.Context(), contractCode, 123)
				return err
			},
		},
		{
			name: "GetAccountFinancialRecords",
			path: "/swap-api/v3/swap_financial_record",
			call: func(h *Exchange) error {
				_, err := h.GetAccountFinancialRecords(t.Context(), contractCode, "3,4", 2, 1, 20)
				return err
			},
		},
		{
			name: "GetSwapSettlementRecords",
			path: "/swap-api/v1/swap_user_settlement_records",
			call: func(h *Exchange) error {
				_, err := h.GetSwapSettlementRecords(t.Context(), contractCode, startTime, startTime.Add(time.Hour), 1, 20)
				return err
			},
		},
		{
			name: "GetAvailableLeverage",
			path: "/swap-api/v1/swap_available_level_rate",
			call: func(h *Exchange) error {
				_, err := h.GetAvailableLeverage(t.Context(), contractCode)
				return err
			},
		},
		{
			name: "SwitchCoinMarginedLeverage",
			path: "/swap-api/v1/swap_switch_lever_rate",
			call: func(h *Exchange) error {
				return h.SwitchCoinMarginedLeverage(t.Context(), contractCode, 5)
			},
		},
		{
			name: "GetSwapOrderLimitInfo",
			path: "/swap-api/v1/swap_order_limit",
			call: func(h *Exchange) error {
				_, err := h.GetSwapOrderLimitInfo(t.Context(), contractCode, "limit")
				return err
			},
		},
		{
			name: "GetSwapTradingFeeInfo",
			path: "/swap-api/v1/swap_fee",
			call: func(h *Exchange) error {
				_, err := h.GetSwapTradingFeeInfo(t.Context(), contractCode)
				return err
			},
		},
		{
			name: "GetSwapTransferLimitInfo",
			path: "/swap-api/v1/swap_transfer_limit",
			call: func(h *Exchange) error {
				_, err := h.GetSwapTransferLimitInfo(t.Context(), contractCode)
				return err
			},
		},
		{
			name: "GetSwapPositionLimitInfo",
			path: "/swap-api/v1/swap_position_limit",
			call: func(h *Exchange) error {
				_, err := h.GetSwapPositionLimitInfo(t.Context(), contractCode)
				return err
			},
		},
		{
			name: "AccountTransferData",
			path: "/swap-api/v1/swap_master_sub_transfer",
			call: func(h *Exchange) error {
				_, err := h.AccountTransferData(t.Context(), contractCode, "123", "master_to_sub", 1)
				return err
			},
		},
		{
			name: "AccountTransferRecords",
			path: "/swap-api/v1/swap_master_sub_transfer_record",
			call: func(h *Exchange) error {
				_, err := h.AccountTransferRecords(t.Context(), contractCode, "master_to_sub", 2, 1, 20)
				return err
			},
		},
		{
			name: "PlaceSwapOrders",
			path: "/swap-api/v1/swap_order",
			call: func(h *Exchange) error {
				_, err := h.PlaceSwapOrders(t.Context(), contractCode, "", "buy", "open", "limit", 1, 1, 1)
				return err
			},
		},
		{
			name: "PlaceSwapBatchOrders",
			path: "/swap-api/v1/swap_batchorder",
			call: func(h *Exchange) error {
				_, err := h.PlaceSwapBatchOrders(t.Context(), BatchOrderRequestType{Data: []batchOrderData{{
					ContractCode:   "ETH-USD",
					Price:          1,
					Volume:         1,
					Direction:      "buy",
					Offset:         "open",
					LeverageRate:   1,
					OrderPriceType: "limit",
				}}})
				return err
			},
		},
		{
			name: "CancelSwapOrder",
			path: "/swap-api/v1/swap_cancel",
			call: func(h *Exchange) error {
				_, err := h.CancelSwapOrder(t.Context(), "123", "", contractCode)
				return err
			},
		},
		{
			name: "CancelAllSwapOrders",
			path: "/swap-api/v1/swap_cancelall",
			call: func(h *Exchange) error {
				_, err := h.CancelAllSwapOrders(t.Context(), contractCode)
				return err
			},
		},
		{
			name: "PlaceLightningCloseOrder",
			path: "/swap-api/v1/swap_lightning_close_position",
			call: func(h *Exchange) error {
				_, err := h.PlaceLightningCloseOrder(t.Context(), contractCode, "buy", "lightning", 1, 123)
				return err
			},
		},
		{
			name: "GetSwapOrderDetails",
			path: "/swap-api/v1/swap_order_detail",
			call: func(h *Exchange) error {
				_, err := h.GetSwapOrderDetails(t.Context(), contractCode, "123", "10", "cancelledOrder", 1, 20)
				return err
			},
		},
		{
			name: "GetSwapOrderInfo",
			path: "/swap-api/v1/swap_order_info",
			call: func(h *Exchange) error {
				_, err := h.GetSwapOrderInfo(t.Context(), contractCode, "123", "")
				return err
			},
		},
		{
			name: "GetSwapOpenOrders",
			path: "/swap-api/v1/swap_openorders",
			call: func(h *Exchange) error {
				_, err := h.GetSwapOpenOrders(t.Context(), contractCode, 1, 20)
				return err
			},
		},
		{
			name: "GetSwapTradeHistory",
			path: "/swap-api/v3/swap_matchresults",
			call: func(h *Exchange) error {
				_, err := h.GetSwapTradeHistory(t.Context(), contractCode, "liquidateShort", 2, 1, 20)
				return err
			},
		},
		{
			name: "PlaceSwapTriggerOrder",
			path: "/swap-api/v1/swap_trigger_order",
			call: func(h *Exchange) error {
				_, err := h.PlaceSwapTriggerOrder(t.Context(), contractCode, "greaterOrEqual", "buy", "open", "optimal_5", 2, 1, 1, 1)
				return err
			},
		},
		{
			name:          "CancelSwapTriggerOrder",
			path:          "/swap-api/v1/swap_trigger_cancel",
			expectedField: "contract_code",
			expectedValue: "ETH-USD",
			call: func(h *Exchange) error {
				_, err := h.CancelSwapTriggerOrder(t.Context(), contractCode, "123")
				return err
			},
		},
		{
			name: "CancelAllSwapTriggerOrders",
			path: "/swap-api/v1/swap_trigger_cancelall",
			call: func(h *Exchange) error {
				_, err := h.CancelAllSwapTriggerOrders(t.Context(), contractCode)
				return err
			},
		},
		{
			name: "GetSwapTriggerOrderHistory",
			path: "/swap-api/v1/swap_trigger_hisorders",
			call: func(h *Exchange) error {
				_, err := h.GetSwapTriggerOrderHistory(t.Context(), contractCode, "open", "all", 2, 1, 20)
				return err
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method, "authenticated coin-margined method should match")
				assert.Equal(t, tc.path, r.URL.Path, "authenticated coin-margined path should match HTX documentation")
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"), "authenticated coin-margined content type should match")
				if tc.expectedField != "" {
					payload, err := io.ReadAll(r.Body)
					if !assert.NoError(t, err, "request body should be readable") {
						return
					}
					var body map[string]any
					if !assert.NoError(t, json.Unmarshal(payload, &body), "request body should decode") {
						return
					}
					assert.Equal(t, tc.expectedValue, body[tc.expectedField], "request body should contain the documented value")
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","code":200,"msg":"","data":null}`))
			}))
			t.Cleanup(server.Close)

			h := new(Exchange)
			require.NoError(t, testexch.Setup(h), "HTX setup must not error")
			h.API.AuthenticatedSupport = true
			h.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
			require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestFutures.String(), server.URL), "coin-margined endpoint must be set")
			require.NoError(t, tc.call(h), "authenticated coin-margined endpoint must not error")
		})
	}
}

func TestGetHistoricalFundingRatesForPair(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/swap-api/v1/swap_historical_funding_rate", r.URL.Path, "coin-margined funding history path should match")
		assert.Equal(t, "25", r.URL.Query().Get("page_size"), "page size should use the pageSize argument")
		assert.Equal(t, "2", r.URL.Query().Get("page_index"), "page index should use the pageIndex argument")
		_, _ = w.Write([]byte(`{"status":"ok","data":{"total_page":1,"current_page":1,"total_size":1,"data":[{"funding_rate":"0.001","funding_time":"1604312615051","contract_code":"BTC-USD"}]}}`))
	}))
	t.Cleanup(server.Close)

	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestFutures.String(), server.URL), "futures endpoint must be set")
	got, err := h.GetHistoricalFundingRatesForPair(t.Context(), btcusdPair, 25, 2)
	require.NoError(t, err, "GetHistoricalFundingRatesForPair must not error")
	require.Len(t, got.Data.Data, 1, "one funding rate must be returned")
	assert.Equal(t, time.UnixMilli(1604312615051), got.Data.Data[0].FundingTime.Time(), "funding time should match")
}

func TestSwitchCoinMarginedLeverage(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/swap-api/v1/swap_switch_lever_rate", r.URL.Path, "coin-margined leverage path should match")
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err, "request body should be readable")
		assert.JSONEq(t, `{"contract_code":"BTC-USD","lever_rate":5}`, string(body), "coin-margined leverage body should match")
		_, _ = w.Write([]byte(`{"status":"ok","data":{"contract_code":"BTC-USD","lever_rate":5}}`))
	}))
	t.Cleanup(server.Close)

	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	h.API.AuthenticatedSupport = true
	h.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
	require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestFutures.String(), server.URL), "futures endpoint must be set")
	require.NoError(t, h.SwitchCoinMarginedLeverage(t.Context(), btcusdPair, 5), "SwitchCoinMarginedLeverage must not error")
}
