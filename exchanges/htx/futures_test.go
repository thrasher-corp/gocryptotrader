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
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestFuturesHistoryEndpointPaths(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "/api/v3/contract_financial_record", fFinancialRecords, "delivery futures financial records endpoint should match HTX docs")
	assert.Equal(t, "/api/v3/contract_hisorders", fOrderHistory, "delivery futures order history endpoint should match HTX docs")
	assert.Equal(t, "/api/v3/contract_matchresults", fMatchResult, "delivery futures trade history endpoint should match HTX docs")
}

func TestFuturesAuthenticatedHTTPRequest(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name         string
		statusCode   int
		body         string
		authenticate bool
		nilResult    bool
		expected     []error
	}{
		{
			name:       "authentication required",
			statusCode: http.StatusOK,
			expected:   []error{exchange.ErrAuthenticationSupportNotEnabled},
		},
		{
			name:         "empty data",
			statusCode:   http.StatusOK,
			body:         `{"code":200,"msg":"","data":"","ts":1604312615051}`,
			authenticate: true,
		},
		{
			name:         "no content without result",
			statusCode:   http.StatusNoContent,
			authenticate: true,
			nilResult:    true,
		},
		{
			name:         "no content with result",
			statusCode:   http.StatusNoContent,
			authenticate: true,
			expected:     []error{errExpectedResponseBody, request.ErrAuthRequestFailed},
		},
		{
			name:         "unexpected response without result",
			statusCode:   http.StatusOK,
			body:         `{"code":200,"msg":"","data":[]}`,
			authenticate: true,
			nilResult:    true,
			expected:     []error{errUnexpectedResponseBody, request.ErrAuthRequestFailed},
		},
		{
			name:         "API error without result",
			statusCode:   http.StatusOK,
			body:         `{"status":"error","err_code":1001,"err_msg":"invalid request"}`,
			authenticate: true,
			nilResult:    true,
			expected:     []error{request.ErrAuthRequestFailed},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/private", r.URL.Path, "request path should match")
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"), "request content type should match")
				if tc.body != "" {
					w.Header().Set("Content-Type", "application/json")
				}
				w.WriteHeader(tc.statusCode)
				_, _ = w.Write([]byte(tc.body))
			}))
			t.Cleanup(server.Close)

			h := new(Exchange)
			require.NoError(t, testexch.Setup(h), "HTX setup must not error")
			if tc.authenticate {
				h.API.AuthenticatedSupport = true
				h.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
			}
			require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestFutures.String(), server.URL), "futures endpoint must be set")
			response := new(FFinancialRecords)
			var result any = response
			if tc.nilResult {
				result = nil
			}
			err := h.FuturesAuthenticatedHTTPRequest(t.Context(), exchange.RestFutures, http.MethodPost, "/private", nil, nil, result)
			for _, expected := range tc.expected {
				assert.ErrorIs(t, err, expected, "FuturesAuthenticatedHTTPRequest should return the expected error")
			}
			if len(tc.expected) != 0 {
				require.Error(t, err, "FuturesAuthenticatedHTTPRequest must return an error")
				return
			}
			require.NoError(t, err, "FuturesAuthenticatedHTTPRequest must not error")
			if result != nil {
				assert.Empty(t, response.Data.FinancialRecord, "empty data should produce an empty result")
			}
		})
	}
}

func TestAddV3HistoryTimeRange(t *testing.T) {
	t.Parallel()
	req := make(map[string]any)
	err := addV3HistoryTimeRange(req, 2)
	require.NoError(t, err, "addV3HistoryTimeRange must accept a two-day lookback")
	startTime, ok := req["start_time"].(int64)
	require.True(t, ok, "start time must be set")
	endTime, ok := req["end_time"].(int64)
	require.True(t, ok, "end time must be set")
	assert.Greater(t, endTime, startTime, "end time should be after start time")
	assert.InDelta(t, int64(48*time.Hour/time.Millisecond), endTime-startTime, float64(time.Minute/time.Millisecond), "lookback should cover 48 hours")

	emptyReq := make(map[string]any)
	require.NoError(t, addV3HistoryTimeRange(emptyReq, 0), "addV3HistoryTimeRange must accept a zero lookback")
	assert.Empty(t, emptyReq, "zero lookback should not set a time range")
	require.ErrorIs(t, addV3HistoryTimeRange(make(map[string]any), 3), errInvalidCreateDate, "addV3HistoryTimeRange must reject lookbacks over two days")
}

func TestFGetContractInfo(t *testing.T) {
	t.Parallel()
	_, err := e.FGetContractInfo(t.Context(), "", "", currency.EMPTYPAIR)
	require.NoError(t, err)
}

func TestFIndexPriceInfo(t *testing.T) {
	t.Parallel()
	_, err := e.FIndexPriceInfo(t.Context(), currency.BTC)
	require.NoError(t, err)
}

func TestFContractPriceLimitations(t *testing.T) {
	t.Parallel()
	_, err := e.FContractPriceLimitations(t.Context(),
		"BTC", "this_week", currency.EMPTYPAIR)
	require.NoError(t, err)
}

func TestFContractOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := e.FContractOpenInterest(t.Context(), "BTC", "this_week", currency.EMPTYPAIR)
	require.NoError(t, err)
}

func TestFGetEstimatedDeliveryPrice(t *testing.T) {
	t.Parallel()
	_, err := e.FGetEstimatedDeliveryPrice(t.Context(), currency.BTC)
	require.NoError(t, err)
}

func TestFGetMarketDepth(t *testing.T) {
	t.Parallel()
	_, err := e.FGetMarketDepth(t.Context(), btccwPair, "step5")
	require.NoError(t, err)
}

func TestFGetKlineData(t *testing.T) {
	t.Parallel()
	_, err := e.FGetKlineData(t.Context(), btccwPair, "5min", 5, time.Now().Add(-time.Minute*5), time.Now())
	require.NoError(t, err)
}

func TestFGetMarketOverviewData(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodGet, fMarketOverview, `{"status":"ok","tick":{"id":123}}`, nil)
	resp, err := h.FGetMarketOverviewData(t.Context(), btccwPair)
	require.NoError(t, err, "FGetMarketOverviewData must not error")
	assert.Equal(t, int64(123), resp.Tick.ID, "market overview ID should decode")
}

func TestFLastTradeData(t *testing.T) {
	t.Parallel()
	_, err := e.FLastTradeData(t.Context(), btccwPair)
	require.NoError(t, err)
}

func TestFRequestPublicBatchTrades(t *testing.T) {
	t.Parallel()
	_, err := e.FRequestPublicBatchTrades(t.Context(), btccwPair, 50)
	require.NoError(t, err)
}

func TestFQueryTieredAdjustmentFactor(t *testing.T) {
	t.Parallel()
	_, err := e.FQueryTieredAdjustmentFactor(t.Context(), currency.BTC)
	require.NoError(t, err)
}

func TestFQueryHisOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := e.FQueryHisOpenInterest(t.Context(), "BTC", "this_week", "60min", "cont", 3)
	require.NoError(t, err)
}

func TestFQuerySystemStatus(t *testing.T) {
	t.Parallel()
	_, err := e.FQuerySystemStatus(t.Context(), currency.BTC)
	require.NoError(t, err)
}

func TestFQueryTopAccountsRatio(t *testing.T) {
	t.Parallel()
	_, err := e.FQueryTopAccountsRatio(t.Context(), "BTC", "5min")
	require.NoError(t, err)
}

func TestFQueryTopPositionsRatio(t *testing.T) {
	t.Parallel()
	_, err := e.FQueryTopPositionsRatio(t.Context(), "BTC", "5min")
	require.NoError(t, err)
}

func TestFLiquidationOrders(t *testing.T) {
	t.Parallel()
	if _, err := e.FLiquidationOrders(t.Context(), currency.BTC, "filled", 0, 0, "", 0); err != nil {
		t.Error(err)
	}
}

func TestFIndexKline(t *testing.T) {
	t.Parallel()
	_, err := e.FIndexKline(t.Context(), btccwPair, "5min", 5)
	require.NoError(t, err)
}

func TestFGetBasisData(t *testing.T) {
	t.Parallel()
	_, err := e.FGetBasisData(t.Context(), btccwPair, "5min", "open", 3)
	require.NoError(t, err)
}

func TestFGetAccountInfo(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, fAccountData, emptySuccessResponse, nil)
	_, err := h.FGetAccountInfo(t.Context(), currency.BTC)
	require.NoError(t, err, "FGetAccountInfo must not error")
}

func TestFGetPositionsInfo(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, fPositionInformation, emptySuccessResponse, nil)
	_, err := h.FGetPositionsInfo(t.Context(), currency.BTC)
	require.NoError(t, err, "FGetPositionsInfo must not error")
}

func TestFGetAllSubAccountAssets(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, fAllSubAccountAssets, emptySuccessResponse, nil)
	_, err := h.FGetAllSubAccountAssets(t.Context(), currency.BTC)
	require.NoError(t, err, "FGetAllSubAccountAssets must not error")
}

func TestFGetSingleSubAccountInfo(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, fSingleSubAccountAssets, emptySuccessResponse, nil)
	_, err := h.FGetSingleSubAccountInfo(t.Context(), "BTC", "154263566")
	require.NoError(t, err, "FGetSingleSubAccountInfo must not error")
}

func TestFGetSingleSubPositions(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, fSingleSubAccountPositions, emptySuccessResponse, nil)
	_, err := h.FGetSingleSubPositions(t.Context(), "BTC", "154263566")
	require.NoError(t, err, "FGetSingleSubPositions must not error")
}

func TestFGetFinancialRecords(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, fFinancialRecords, emptySuccessResponse, nil)
	_, err := h.FGetFinancialRecords(t.Context(), "BTC", "closeLong", 2, 1, 20)
	require.NoError(t, err, "FGetFinancialRecords must not error")
}

func TestFGetSettlementRecords(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, fSettlementRecords, emptySuccessResponse, nil)
	start := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
	_, err := h.FGetSettlementRecords(t.Context(), currency.BTC, 1, 20, start, start.Add(time.Hour))
	require.NoError(t, err, "FGetSettlementRecords must not error")
}

func TestFGetOrderLimits(t *testing.T) {
	t.Parallel()
	t.Run("invalid order price type", func(t *testing.T) {
		t.Parallel()
		h := new(Exchange)
		require.NoError(t, testexch.Setup(h), "HTX setup must not error")
		_, err := h.FGetOrderLimits(t.Context(), "BTC", "not-real")
		require.Error(t, err, "FGetOrderLimits must reject invalid order price type")
	})
	t.Run("success", func(t *testing.T) {
		t.Parallel()
		h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, fOrderLimitInfo, emptySuccessResponse, nil)
		_, err := h.FGetOrderLimits(t.Context(), "BTC", "limit")
		require.NoError(t, err, "FGetOrderLimits must not error")
	})
}

func TestFContractTradingFee(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, fContractTradingFee, emptySuccessResponse, nil)
	_, err := h.FContractTradingFee(t.Context(), currency.BTC)
	require.NoError(t, err, "FContractTradingFee must not error")
}

func TestFGetTransferLimits(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, fTransferLimitInfo, emptySuccessResponse, nil)
	_, err := h.FGetTransferLimits(t.Context(), currency.BTC)
	require.NoError(t, err, "FGetTransferLimits must not error")
}

func TestFGetPositionLimits(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, fPositionLimitInfo, emptySuccessResponse, nil)
	_, err := h.FGetPositionLimits(t.Context(), currency.BTC)
	require.NoError(t, err, "FGetPositionLimits must not error")
}

func TestFGetAssetsAndPositions(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, fQueryAssetsAndPositions, emptySuccessResponse, nil)
	_, err := h.FGetAssetsAndPositions(t.Context(), currency.BTC)
	require.NoError(t, err, "FGetAssetsAndPositions must not error")
}

func TestFTransfer(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, fTransfer, emptySuccessResponse, nil)
	_, err := h.FTransfer(t.Context(), "154263566", "BTC", "sub_to_master", 5)
	require.NoError(t, err, "FTransfer must not error")
}

func TestFGetTransferRecords(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, fTransferRecords, emptySuccessResponse, nil)
	_, err := h.FGetTransferRecords(t.Context(), "BTC", "master_to_sub", 2, 1, 20)
	require.NoError(t, err, "FGetTransferRecords must not error")
}

func TestFGetAvailableLeverage(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, fAvailableLeverage, emptySuccessResponse, nil)
	_, err := h.FGetAvailableLeverage(t.Context(), currency.BTC)
	require.NoError(t, err, "FGetAvailableLeverage must not error")
}

func TestFOrder(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, fOrder, emptySuccessResponse, nil)
	_, err := h.FOrder(t.Context(), currency.EMPTYPAIR, "BTC", "quarter", "123", "buy", "open", "limit", 1, 1, 1)
	require.NoError(t, err, "FOrder must not error")
}

func TestFPlaceBatchOrder(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, fBatchOrder, emptySuccessResponse, nil)
	_, err := h.FPlaceBatchOrder(t.Context(), []fBatchOrderData{{
		Symbol:         "BTC",
		ContractType:   "quarter",
		Price:          5,
		Volume:         1,
		Direction:      "buy",
		Offset:         "open",
		LeverageRate:   1,
		OrderPriceType: "limit",
	}})
	require.NoError(t, err, "FPlaceBatchOrder must not error")
}

func TestFCancelOrder(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, fCancelOrder, emptySuccessResponse, nil)
	_, err := h.FCancelOrder(t.Context(), currency.BTC, "123", "")
	require.NoError(t, err, "FCancelOrder must not error")
}

func TestFCancelAllOrders(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, fCancelAllOrders, emptySuccessResponse, nil)
	_, err := h.FCancelAllOrders(t.Context(), currency.EMPTYPAIR, "BTC", "quarter")
	require.NoError(t, err, "FCancelAllOrders must not error")
}

func TestFFlashCloseOrder(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, fFlashCloseOrder, emptySuccessResponse, nil)
	_, err := h.FFlashCloseOrder(t.Context(), currency.EMPTYPAIR, "BTC", "quarter", "buy", "lightning", "", 1)
	require.NoError(t, err, "FFlashCloseOrder must not error")
}

func TestFGetOrderInfo(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, fOrderInfo, emptySuccessResponse, nil)
	_, err := h.FGetOrderInfo(t.Context(), "BTC", "", "123")
	require.NoError(t, err, "FGetOrderInfo must not error")
}

func TestFOrderDetails(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, fOrderDetails, emptySuccessResponse, nil)
	_, err := h.FOrderDetails(t.Context(), "BTC", "123", "quotation", time.Now().Add(-time.Hour), 1, 20)
	require.NoError(t, err, "FOrderDetails must not error")
}

func TestFGetOpenOrders(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, fQueryOpenOrders, emptySuccessResponse, nil)
	_, err := h.FGetOpenOrders(t.Context(), currency.BTC, 1, 20)
	require.NoError(t, err, "FGetOpenOrders must not error")
}

func TestFGetOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.FGetOrderHistory(t.Context(),
		currency.EMPTYPAIR, "BTC",
		"all", "all", "limit",
		[]order.Status{},
		3, 0, 0)
	require.ErrorIs(t, err, errInvalidCreateDate, "FGetOrderHistory must reject lookbacks over two days")
}

func TestFGetOrderHistoryByTimeRange(t *testing.T) {
	t.Parallel()
	body := make(chan []byte, 1)
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, fOrderHistory, `{"code":200,"data":[{"query_id":12,"order_id":34,"order_id_str":"34","contract_code":"BTC-USD","direction":"buy","order_price_type":"limit","status":6}]}`, func(r *http.Request) {
		payload, err := io.ReadAll(r.Body)
		assert.NoError(t, err, "request body should be readable")
		body <- payload
	})
	startTime := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
	endTime := startTime.Add(48 * time.Hour)
	resp, err := h.FGetOrderHistoryByTimeRange(t.Context(), currency.EMPTYPAIR, "BTC", "all", "all", "limit", nil, startTime, endTime, 11, 50)
	require.NoError(t, err, "FGetOrderHistoryByTimeRange must not error")
	require.Len(t, resp.Data.Orders, 1, "decoded order history must be returned")
	assert.Equal(t, int64(12), resp.Data.Orders[0].QueryID, "query ID should decode")

	var requestBody map[string]any
	require.NoError(t, json.Unmarshal(<-body, &requestBody), "request body must decode")
	assert.Equal(t, float64(startTime.UnixMilli()), requestBody["start_time"], "start time should be preserved")
	assert.Equal(t, float64(endTime.UnixMilli()), requestBody["end_time"], "end time should be preserved")
	assert.Equal(t, v3HistoryDirectionNext, requestBody["direct"], "pagination direction should move forwards")
	assert.Equal(t, float64(11), requestBody["from_id"], "cursor should be preserved")
	assert.Equal(t, float64(50), requestBody["limit"], "limit should be preserved")

	_, err = h.FGetOrderHistoryByTimeRange(t.Context(), currency.EMPTYPAIR, "BTC", "all", "all", "limit", nil, startTime, endTime.Add(time.Millisecond), 0, 50)
	require.ErrorIs(t, err, errHistoryTimeRangeExceeded, "FGetOrderHistoryByTimeRange must reject intervals over 48 hours")
}

func TestFTradeHistory(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, fMatchResult, emptySuccessResponse, nil)
	_, err := h.FTradeHistory(t.Context(), currency.EMPTYPAIR, "BTC", "all", 2, 1, 20)
	require.NoError(t, err, "FTradeHistory must not error")
}

func TestFPlaceTriggerOrder(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, fTriggerOrder, emptySuccessResponse, nil)
	_, err := h.FPlaceTriggerOrder(t.Context(), currency.EMPTYPAIR, "BTC", "quarter", "greaterOrEqual", "limit", "buy", "close", 1.1, 1.05, 5, 2)
	require.NoError(t, err, "FPlaceTriggerOrder must not error")
}

func TestFCancelTriggerOrder(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, fCancelTriggerOrder, emptySuccessResponse, nil)
	_, err := h.FCancelTriggerOrder(t.Context(), "BTC", "123")
	require.NoError(t, err, "FCancelTriggerOrder must not error")
}

func TestFCancelAllTriggerOrders(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, fCancelAllTriggerOrders, emptySuccessResponse, nil)
	_, err := h.FCancelAllTriggerOrders(t.Context(), currency.EMPTYPAIR, "BTC", "this_week")
	require.NoError(t, err, "FCancelAllTriggerOrders must not error")
}

func TestFQueryTriggerOpenOrders(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, fTriggerOpenOrders, emptySuccessResponse, nil)
	_, err := h.FQueryTriggerOpenOrders(t.Context(), currency.EMPTYPAIR, "BTC", 1, 20)
	require.NoError(t, err, "FQueryTriggerOpenOrders must not error")
}

func TestFQueryTriggerOrderHistory(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, fTriggerOrderHistory, emptySuccessResponse, nil)
	_, err := h.FQueryTriggerOrderHistory(t.Context(), currency.EMPTYPAIR, "BTC", "all", "all", 10, 1, 20)
	require.NoError(t, err, "FQueryTriggerOrderHistory must not error")
}

func TestFormatFuturesPair(t *testing.T) {
	t.Parallel()
	updatePairsOnce(t, e)

	r, err := e.formatFuturesPair(btccwPair, false)
	require.NoError(t, err)
	assert.Equal(t, "BTC_CW", r)

	// pair in the format of BTC210827 but make it lower case to test correct formatting
	r, err = e.formatFuturesPair(btcFutureDatedPair.Lower(), false)
	require.NoError(t, err)
	assert.Len(t, r, 9, "Should be an 9 character string")
	assert.Equal(t, "BTC2", r[0:4], "Should start with btc and a date this millennium")

	r, err = e.formatFuturesPair(btccwPair, true)
	require.NoError(t, err)
	assert.Len(t, r, 9, "Should be an 9 character string")
	assert.Equal(t, "BTC2", r[0:4], "Should start with btc and a date this millennium")

	r, err = e.formatFuturesPair(currency.NewBTCUSDT(), false)
	require.NoError(t, err)
	assert.Equal(t, "BTC-USDT", r)
}

func TestFormatFuturesCode(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	formatted, err := h.formatFuturesCode(currency.NewCode("btc"))
	require.NoError(t, err, "formatFuturesCode must not error")
	assert.Equal(t, "BTC", formatted, "futures code should use the configured case")
}

var expiryWindows = map[string]uint{
	"CW": 14,
	"NW": 21,
	"CQ": 190,
	"NQ": 282,
}

// TestPairFromContractExpiryCode ensures at least some contract codes are available and loaded with sane dates
// Expectations are relaxed because dates are unpredictable and codes disappear intermittently
func TestPairFromContractExpiryCode(t *testing.T) {
	t.Parallel()

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test Instance Setup must not fail")

	_, err := e.FetchTradablePairs(t.Context(), asset.Futures)
	require.NoError(t, err)

	tz, err := time.LoadLocation("Asia/Singapore") // HTX HQ and apparent local time for when codes become effective
	require.NoError(t, err, "LoadLocation must not error")

	today := time.Now()
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, tz) // Do not use Truncate; https://github.com/golang/go/issues/55921

	require.NotEmpty(t, e.futureContractCodes, "At least one contract code must be loaded")

	for cType, cachedContract := range e.futureContractCodes {
		t.Run(cType, func(t *testing.T) {
			t.Parallel()
			p, err := e.pairFromContractExpiryCode(currency.Pair{
				Base:  currency.BTC,
				Quote: currency.NewCode(cType),
			})
			require.NoError(t, err)
			assert.Equal(t, currency.BTC, p.Base, "pair Base should be BTC")
			assert.Equal(t, cachedContract, p.Quote, "pair Quote should match futureContractCodes value")
			exp, err := time.ParseInLocation("060102", p.Quote.String(), tz)
			require.NoError(t, err, "currency code must be a parsable date")
			require.Falsef(t, exp.Before(today), "expiry must be today or after; Got: %q", exp)
			diff := uint(exp.Sub(today).Hours() / 24)
			require.LessOrEqualf(t, diff, expiryWindows[cType], "expiry must be within expected update window; Today: %q, Expiry: %q",
				today.Format(time.DateOnly),
				exp.Format(time.DateOnly),
			)
		})
	}
}

func TestFSwitchLeverage(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, fSwitchLeverage, `{"status":"ok","data":{"symbol":"BTC","lever_rate":5}}`, func(r *http.Request) {
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err, "request body should be readable")
		assert.JSONEq(t, `{"symbol":"BTC","lever_rate":5}`, string(body), "delivery leverage body should match")
	})
	require.NoError(t, h.FSwitchLeverage(t.Context(), currency.BTC, 5), "FSwitchLeverage must not error")
}
