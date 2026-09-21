package htx

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
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
	_, err = e.GetSwapKlineData(t.Context(), btcusdPair, "invalid", 5, time.Time{}, time.Time{})
	require.ErrorIs(t, err, common.ErrInvalidPeriod, "GetSwapKlineData must reject an invalid period")
	_, err = e.GetSwapKlineData(t.Context(), btcusdPair, "5min", 5, time.Now(), time.Time{})
	require.ErrorIs(t, err, common.ErrDateUnset, "GetSwapKlineData must reject a half-open time range")
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
	_, err = e.GetOpenInterestInfo(t.Context(), btcusdPair, "invalid", "cryptocurrency", 50)
	require.ErrorIs(t, err, common.ErrInvalidPeriod, "GetOpenInterestInfo must reject an invalid period")
	_, err = e.GetOpenInterestInfo(t.Context(), btcusdPair, "5min", "cryptocurrency", 0)
	require.ErrorIs(t, err, errInvalidSize, "GetOpenInterestInfo must reject an invalid size")
	_, err = e.GetOpenInterestInfo(t.Context(), btcusdPair, "5min", "invalid", 50)
	require.ErrorIs(t, err, errInvalidAmountType, "GetOpenInterestInfo must reject an invalid amount type")
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
	_, err = e.GetLiquidationOrders(t.Context(), btcusdPair, "invalid", time.Time{}, time.Time{}, "", 0)
	require.ErrorIs(t, err, errInvalidTradeType, "GetLiquidationOrders must reject an invalid trade type")
}

func TestGetPremiumIndexKlineData(t *testing.T) {
	t.Parallel()
	_, err := e.GetPremiumIndexKlineData(t.Context(), btcusdPair, "5min", 15)
	require.NoError(t, err)
	_, err = e.GetPremiumIndexKlineData(t.Context(), btcusdPair, "invalid", 15)
	require.ErrorIs(t, err, common.ErrInvalidPeriod, "GetPremiumIndexKlineData must reject an invalid period")
	_, err = e.GetPremiumIndexKlineData(t.Context(), btcusdPair, "5min", 0)
	require.ErrorIs(t, err, errInvalidSize, "GetPremiumIndexKlineData must reject an invalid size")
}

func TestGetEstimatedFundingRates(t *testing.T) {
	t.Parallel()
	_, err := e.GetEstimatedFundingRates(t.Context(), btcusdPair, "5min", 15)
	require.NoError(t, err)

	_, err = e.GetEstimatedFundingRates(t.Context(), btcusdPair, "invalid", 15)
	require.ErrorIs(t, err, common.ErrInvalidPeriod, "GetEstimatedFundingRates must reject invalid period")
	_, err = e.GetEstimatedFundingRates(t.Context(), btcusdPair, "5min", 0)
	require.ErrorIs(t, err, errInvalidSize, "GetEstimatedFundingRates must reject an invalid size")
}

func TestGetBasisData(t *testing.T) {
	t.Parallel()
	_, err := e.GetBasisData(t.Context(), btcusdPair, "5min", "close", 5)
	require.NoError(t, err)
	_, err = e.GetBasisData(t.Context(), btcusdPair, "invalid", "close", 5)
	require.ErrorIs(t, err, common.ErrInvalidPeriod, "GetBasisData must reject an invalid period")
	_, err = e.GetBasisData(t.Context(), btcusdPair, "5min", "close", 0)
	require.ErrorIs(t, err, errInvalidSize, "GetBasisData must reject an invalid size")
	_, err = e.GetBasisData(t.Context(), btcusdPair, "5min", "invalid", 5)
	require.ErrorIs(t, err, errInvalidBasisPriceType, "GetBasisData must reject an invalid basis price type")
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
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_account_info", emptySuccessResponse, nil)
	_, err := h.GetSwapAccountInfo(t.Context(), ethusdPair)
	require.NoError(t, err, "GetSwapAccountInfo must not error")
}

func TestGetSwapPositionsInfo(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_position_info", emptySuccessResponse, nil)
	_, err := h.GetSwapPositionsInfo(t.Context(), ethusdPair)
	require.NoError(t, err, "GetSwapPositionsInfo must not error")
}

func TestGetSwapAssetsAndPositions(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_account_position_info", emptySuccessResponse, nil)
	_, err := h.GetSwapAssetsAndPositions(t.Context(), ethusdPair)
	require.NoError(t, err, "GetSwapAssetsAndPositions must not error")
}

func TestGetSwapAllSubAccAssets(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_sub_account_list", emptySuccessResponse, nil)
	_, err := h.GetSwapAllSubAccAssets(t.Context(), ethusdPair)
	require.NoError(t, err, "GetSwapAllSubAccAssets must not error")
}

func TestGetSubAccPositionInfo(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_sub_position_info", emptySuccessResponse, nil)
	_, err := h.GetSubAccPositionInfo(t.Context(), ethusdPair, 123)
	require.NoError(t, err, "GetSubAccPositionInfo must not error")
}

func TestSwapSingleSubAccAssets(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_sub_account_info", emptySuccessResponse, nil)
	_, err := h.SwapSingleSubAccAssets(t.Context(), ethusdPair, 123)
	require.NoError(t, err, "SwapSingleSubAccAssets must not error")
}

func TestGetAccountFinancialRecords(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/swap-api/v3/swap_financial_record", emptySuccessResponse, func(r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err, "request body must be readable")
		var requestBody map[string]any
		require.NoError(t, json.Unmarshal(body, &requestBody), "request body must decode")
		assert.Equal(t, float64(20), requestBody["limit"], "page size should be sent as the V3 limit")
	})
	_, err := h.GetAccountFinancialRecords(t.Context(), ethusdPair, "3,4", 2, 1, 20)
	require.NoError(t, err, "GetAccountFinancialRecords must not error")
	_, err = h.GetAccountFinancialRecords(t.Context(), ethusdPair, "3,4", 3, 1, 20)
	require.ErrorIs(t, err, errInvalidLookbackDays, "GetAccountFinancialRecords must reject lookbacks over two days")
}

func TestGetSwapSettlementRecords(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_user_settlement_records", emptySuccessResponse, nil)
	start := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
	_, err := h.GetSwapSettlementRecords(t.Context(), ethusdPair, start, start.Add(time.Hour), 1, 20)
	require.NoError(t, err, "GetSwapSettlementRecords must not error")
	_, err = h.GetSwapSettlementRecords(t.Context(), ethusdPair, start, time.Time{}, 1, 20)
	require.ErrorIs(t, err, common.ErrDateUnset, "GetSwapSettlementRecords must reject a half-open interval")
}

func TestGetAvailableLeverage(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_available_level_rate", emptySuccessResponse, nil)
	_, err := h.GetAvailableLeverage(t.Context(), ethusdPair)
	require.NoError(t, err, "GetAvailableLeverage must not error")
}

func TestGetSwapOrderLimitInfo(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_order_limit", emptySuccessResponse, nil)
	_, err := h.GetSwapOrderLimitInfo(t.Context(), ethusdPair, "limit")
	require.NoError(t, err, "GetSwapOrderLimitInfo must not error")
}

func TestGetSwapTradingFeeInfo(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_fee", emptySuccessResponse, nil)
	_, err := h.GetSwapTradingFeeInfo(t.Context(), ethusdPair)
	require.NoError(t, err, "GetSwapTradingFeeInfo must not error")
}

func TestGetSwapTransferLimitInfo(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_transfer_limit", emptySuccessResponse, nil)
	_, err := h.GetSwapTransferLimitInfo(t.Context(), ethusdPair)
	require.NoError(t, err, "GetSwapTransferLimitInfo must not error")
}

func TestGetSwapPositionLimitInfo(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_position_limit", emptySuccessResponse, nil)
	_, err := h.GetSwapPositionLimitInfo(t.Context(), ethusdPair)
	require.NoError(t, err, "GetSwapPositionLimitInfo must not error")
}

func TestAccountTransferData(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_master_sub_transfer", emptySuccessResponse, nil)
	_, err := h.AccountTransferData(t.Context(), ethusdPair, "123", "master_to_sub", 15)
	require.NoError(t, err, "AccountTransferData must not error")
	_, err = h.AccountTransferData(t.Context(), ethusdPair, "123", "invalid", 15)
	require.ErrorIs(t, err, errInvalidTransferType, "AccountTransferData must reject an invalid transfer type")
}

func TestAccountTransferRecords(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_master_sub_transfer_record", emptySuccessResponse, nil)
	_, err := h.AccountTransferRecords(t.Context(), ethusdPair, "master_to_sub", 12, 1, 20)
	require.NoError(t, err, "AccountTransferRecords must not error")
	_, err = h.AccountTransferRecords(t.Context(), ethusdPair, "invalid", 12, 1, 20)
	require.ErrorIs(t, err, errInvalidTransferType, "AccountTransferRecords must reject an invalid transfer type")
	_, err = h.AccountTransferRecords(t.Context(), ethusdPair, "master_to_sub", 91, 1, 20)
	require.ErrorIs(t, err, errInvalidCreateDate, "AccountTransferRecords must reject a lookback over 90 days")
}

func TestPlaceSwapOrders(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_order", emptySuccessResponse, nil)
	_, err := h.PlaceSwapOrders(t.Context(), ethusdPair, "", "buy", "open", "limit", 0.01, 1, 1)
	require.NoError(t, err, "PlaceSwapOrders must not error")
	_, err = h.PlaceSwapOrders(t.Context(), ethusdPair, "", "buy", "open", "invalid", 0.01, 1, 1)
	require.ErrorIs(t, err, errInvalidOrderType, "PlaceSwapOrders must reject an invalid order type")
}

func TestPlaceSwapBatchOrders(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_batchorder", emptySuccessResponse, nil)
	req := BatchOrderRequestType{Data: []batchOrderData{{
		ContractCode:   "ETH-USD",
		Price:          5,
		Volume:         1,
		Direction:      "buy",
		Offset:         "open",
		LeverageRate:   1,
		OrderPriceType: "limit",
	}}}
	_, err := h.PlaceSwapBatchOrders(t.Context(), req)
	require.NoError(t, err, "PlaceSwapBatchOrders must not error")
	_, err = h.PlaceSwapBatchOrders(t.Context(), BatchOrderRequestType{})
	require.ErrorIs(t, err, errBatchOrderLimitExceeded, "PlaceSwapBatchOrders must reject an empty batch")
	_, err = h.PlaceSwapBatchOrders(t.Context(), BatchOrderRequestType{Data: make([]batchOrderData, 11)})
	require.ErrorIs(t, err, errBatchOrderLimitExceeded, "PlaceSwapBatchOrders must reject more than ten orders")
}

func TestCancelSwapOrder(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_cancel", emptySuccessResponse, nil)
	_, err := h.CancelSwapOrder(t.Context(), "test123", "", ethusdPair)
	require.NoError(t, err, "CancelSwapOrder must not error")
}

func TestCancelAllSwapOrders(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_cancelall", emptySuccessResponse, nil)
	_, err := h.CancelAllSwapOrders(t.Context(), ethusdPair)
	require.NoError(t, err, "CancelAllSwapOrders must not error")
}

func TestPlaceLightningCloseOrder(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_lightning_close_position", emptySuccessResponse, nil)
	_, err := h.PlaceLightningCloseOrder(t.Context(), ethusdPair, "buy", "lightning", 5, 1)
	require.NoError(t, err, "PlaceLightningCloseOrder must not error")
	_, err = h.PlaceLightningCloseOrder(t.Context(), ethusdPair, "buy", "invalid", 5, 1)
	require.ErrorIs(t, err, errInvalidOrderPriceType, "PlaceLightningCloseOrder must reject an invalid order price type")
}

func TestGetSwapOrderInfo(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_order_info", emptySuccessResponse, nil)
	_, err := h.GetSwapOrderInfo(t.Context(), ethusdPair, "123", "")
	require.NoError(t, err, "GetSwapOrderInfo must not error")
}

func TestGetSwapOrderDetails(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_order_detail", emptySuccessResponse, nil)
	_, err := h.GetSwapOrderDetails(t.Context(), ethusdPair, "123", "10", "cancelledOrder", 1, 20)
	require.NoError(t, err, "GetSwapOrderDetails must not error")
	_, err = h.GetSwapOrderDetails(t.Context(), ethusdPair, "123", "10", "invalid", 1, 20)
	require.ErrorIs(t, err, errInvalidOrderType, "GetSwapOrderDetails must reject an invalid order type")
}

func TestGetSwapOpenOrders(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_openorders", emptySuccessResponse, nil)
	_, err := h.GetSwapOpenOrders(t.Context(), ethusdPair, 1, 20)
	require.NoError(t, err, "GetSwapOpenOrders must not error")
}

func TestGetSwapOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetSwapOrderHistory(t.Context(), ethusdPair, "all", "all", []order.Status{order.PartiallyCancelled, order.Active}, 25, 0, 0)
	require.ErrorIs(t, err, errInvalidLookbackDays, "GetSwapOrderHistory must reject lookbacks over two days")
	_, err = e.GetSwapOrderHistory(t.Context(), ethusdPair, "all", "all", nil, -1, 0, 0)
	require.ErrorIs(t, err, errInvalidLookbackDays, "GetSwapOrderHistory must reject negative lookbacks")
}

func TestGetSwapOrderHistoryByTimeRange(t *testing.T) {
	t.Parallel()
	body := make(chan []byte, 1)
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/swap-api/v3/swap_hisorders", `{"code":200,"data":[{"query_id":12,"order_id":34,"order_id_str":"34","contract_code":"ETH-USD","direction":"buy","order_price_type":"limit","status":6}]}`, func(r *http.Request) {
		payload, err := io.ReadAll(r.Body)
		assert.NoError(t, err, "request body should be readable")
		body <- payload
	})
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
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSwapOrderHistoryByTimeRange must reject reversed intervals")
	_, err = h.GetSwapOrderHistoryByTimeRange(t.Context(), ethusdPair, "invalid", "all", nil, startTime, endTime, 0, 50)
	require.ErrorIs(t, err, errInvalidTradeType, "GetSwapOrderHistoryByTimeRange must reject an invalid trade type")
	_, err = h.GetSwapOrderHistoryByTimeRange(t.Context(), ethusdPair, "all", "invalid", nil, startTime, endTime, 0, 50)
	require.ErrorIs(t, err, errInvalidRequestType, "GetSwapOrderHistoryByTimeRange must reject an invalid request type")
	_, err = h.GetSwapOrderHistoryByTimeRange(t.Context(), ethusdPair, "all", "all", []order.Status{order.Rejected}, startTime, endTime, 0, 50)
	require.ErrorIs(t, err, errInvalidOrderStatus, "GetSwapOrderHistoryByTimeRange must reject an invalid status")
	assert.Contains(t, err.Error(), order.Rejected.String(), "status error should include the invalid status")
	_, err = h.GetSwapOrderHistoryByTimeRange(t.Context(), ethusdPair, "all", "all", nil, startTime, time.Time{}, 0, 50)
	require.ErrorIs(t, err, errInvalidCreateDate, "GetSwapOrderHistoryByTimeRange must reject a half-open interval")
	_, err = h.GetSwapOrderHistoryByTimeRange(t.Context(), ethusdPair, "all", "all", nil, startTime, startTime, 0, 50)
	require.ErrorIs(t, err, common.ErrStartEqualsEnd, "GetSwapOrderHistoryByTimeRange must reject an empty interval")
	_, err = h.GetSwapOrderHistoryByTimeRange(t.Context(), ethusdPair, "all", "all", nil, startTime, endTime.Add(time.Millisecond), 0, 50)
	require.ErrorIs(t, err, errHistoryTimeRangeExceeded, "GetSwapOrderHistoryByTimeRange must reject intervals over 48 hours")
}

func TestGetSwapTradeHistory(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/swap-api/v3/swap_matchresults", emptySuccessResponse, func(r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err, "request body must be readable")
		var requestBody map[string]any
		require.NoError(t, json.Unmarshal(body, &requestBody), "request body must decode")
		assert.Equal(t, float64(20), requestBody["limit"], "page size should be sent as the V3 limit")
	})
	_, err := h.GetSwapTradeHistory(t.Context(), ethusdPair, "liquidateShort", 2, 1, 20)
	require.NoError(t, err, "GetSwapTradeHistory must not error")
	_, err = h.GetSwapTradeHistory(t.Context(), ethusdPair, "invalid", 2, 1, 20)
	require.ErrorIs(t, err, errInvalidTradeType, "GetSwapTradeHistory must reject an invalid trade type")
	_, err = h.GetSwapTradeHistory(t.Context(), ethusdPair, "liquidateShort", 3, 1, 20)
	require.ErrorIs(t, err, errInvalidLookbackDays, "GetSwapTradeHistory must reject lookbacks over two days")
}

func TestPlaceSwapTriggerOrder(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_trigger_order", emptySuccessResponse, nil)
	_, err := h.PlaceSwapTriggerOrder(t.Context(), ethusdPair, "greaterOrEqual", "buy", "open", "optimal_5", 5, 3, 1, 1)
	require.NoError(t, err, "PlaceSwapTriggerOrder must not error")
	_, err = h.PlaceSwapTriggerOrder(t.Context(), ethusdPair, "invalid", "buy", "open", "optimal_5", 5, 3, 1, 1)
	require.ErrorIs(t, err, errInvalidTriggerType, "PlaceSwapTriggerOrder must reject an invalid trigger type")
	_, err = h.PlaceSwapTriggerOrder(t.Context(), ethusdPair, "greaterOrEqual", "buy", "open", "invalid", 5, 3, 1, 1)
	require.ErrorIs(t, err, errInvalidOrderPriceType, "PlaceSwapTriggerOrder must reject an invalid order price type")
}

func TestCancelSwapTriggerOrder(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_trigger_cancel", emptySuccessResponse, func(r *http.Request) {
		payload, err := io.ReadAll(r.Body)
		if !assert.NoError(t, err, "request body should be readable") {
			return
		}
		var body map[string]any
		if !assert.NoError(t, json.Unmarshal(payload, &body), "request body should decode") {
			return
		}
		assert.Equal(t, "ETH-USD", body["contract_code"], "contract code should be included")
	})
	_, err := h.CancelSwapTriggerOrder(t.Context(), ethusdPair, "test123")
	require.NoError(t, err, "CancelSwapTriggerOrder must not error")
}

func TestCancelAllSwapTriggerOrders(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_trigger_cancelall", emptySuccessResponse, nil)
	_, err := h.CancelAllSwapTriggerOrders(t.Context(), ethusdPair)
	require.NoError(t, err, "CancelAllSwapTriggerOrders must not error")
}

func TestGetSwapTriggerOrderHistory(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_trigger_hisorders", emptySuccessResponse, nil)
	_, err := h.GetSwapTriggerOrderHistory(t.Context(), ethusdPair, "open", "all", 15, 1, 20)
	require.NoError(t, err, "GetSwapTriggerOrderHistory must not error")
	_, err = h.GetSwapTriggerOrderHistory(t.Context(), ethusdPair, "open", "invalid", 15, 1, 20)
	require.ErrorIs(t, err, errInvalidTradeType, "GetSwapTriggerOrderHistory must reject an invalid trade type")
	_, err = h.GetSwapTriggerOrderHistory(t.Context(), ethusdPair, "open", "all", 91, 1, 20)
	require.ErrorIs(t, err, errInvalidCreateDate, "GetSwapTriggerOrderHistory must reject a lookback over 90 days")
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
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_switch_lever_rate", `{"status":"ok","data":{"contract_code":"BTC-USD","lever_rate":5}}`, func(r *http.Request) {
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err, "request body should be readable")
		assert.JSONEq(t, `{"contract_code":"BTC-USD","lever_rate":5}`, string(body), "coin-margined leverage body should match")
	})
	require.NoError(t, h.SwitchCoinMarginedLeverage(t.Context(), btcusdPair, 5), "SwitchCoinMarginedLeverage must not error")
}
