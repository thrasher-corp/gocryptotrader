package htx

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestFuturesAuthenticatedHTTPRequest(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name         string
		path         string
		statusCode   int
		body         string
		authenticate bool
		nilResult    bool
		expected     []error
	}{
		{name: "V3 failure", path: "/api/v3/contract_hisorders", statusCode: http.StatusOK, authenticate: true, body: `{"code":403,"msg":"incorrect key","data":""}`, expected: []error{request.ErrAuthRequestFailed}},
		{name: "V3 success", path: "/api/v3/contract_hisorders", statusCode: http.StatusOK, authenticate: true, body: `{"code":200,"msg":"","data":[]}`},

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
		{
			name:         "API error without message",
			statusCode:   http.StatusOK,
			body:         `{"status":"error","err_code":1001}`,
			authenticate: true,
			expected:     []error{request.ErrAuthRequestFailed},
		},
		{
			name:         "HTTP forbidden",
			statusCode:   http.StatusForbidden,
			body:         `{"status":"error","err_msg":"forbidden"}`,
			authenticate: true,
			expected:     []error{request.ErrBadStatus, request.ErrAuthRequestFailed},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			path := tc.path
			if path == "" {
				path = "/private"
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, path, r.URL.Path, "request path should match")
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
			response := new(FuturesFinancialRecordsResponse)
			var result any = response
			if tc.nilResult {
				result = nil
			}
			err := h.FuturesAuthenticatedHTTPRequest(t.Context(), exchange.RestFutures, http.MethodPost, path, nil, nil, result)
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
	require.ErrorIs(t, addV3HistoryTimeRange(make(map[string]any), 3), errInvalidLookbackDays, "addV3HistoryTimeRange must reject lookbacks over two days")
}

func TestFormatFuturesOrderStatuses(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name        string
		statuses    []order.Status
		expected    string
		expectedErr error
	}{
		{name: "empty", expected: "0"},
		{name: "single", statuses: []order.Status{order.Active}, expected: "3"},
		{name: "multiple", statuses: []order.Status{order.PartiallyCancelled, order.Active}, expected: "5,3"},
		{name: "invalid", statuses: []order.Status{order.Rejected}, expectedErr: errInvalidOrderStatus},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			actual, err := formatFuturesOrderStatuses(tc.statuses)
			if tc.expectedErr != nil {
				require.ErrorIs(t, err, tc.expectedErr, "formatFuturesOrderStatuses must return the expected error")
				assert.Contains(t, err.Error(), tc.statuses[0].String(), "error should include the invalid status")
				return
			}
			require.NoError(t, err, "formatFuturesOrderStatuses must not error")
			assert.Equal(t, tc.expected, actual, "formatted statuses should match")
		})
	}
}

func TestFuturesGetContractInfo(t *testing.T) {
	t.Parallel()
	_, err := e.FuturesGetContractInfo(t.Context(), "", "", currency.EMPTYPAIR)
	require.NoError(t, err)
}

func TestFuturesIndexPriceInfo(t *testing.T) {
	t.Parallel()
	_, err := e.FuturesIndexPriceInfo(t.Context(), currency.BTC)
	require.NoError(t, err)
}

func TestFuturesContractPriceLimitations(t *testing.T) {
	t.Parallel()
	_, err := e.FuturesContractPriceLimitations(t.Context(),
		"BTC", "this_week", currency.EMPTYPAIR)
	require.NoError(t, err)
}

func TestFuturesContractOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := e.FuturesContractOpenInterest(t.Context(), "BTC", "this_week", currency.EMPTYPAIR)
	require.NoError(t, err)
}

func TestFuturesGetEstimatedDeliveryPrice(t *testing.T) {
	t.Parallel()
	_, err := e.FuturesGetEstimatedDeliveryPrice(t.Context(), currency.BTC)
	require.NoError(t, err)
}

func TestFuturesGetMarketDepth(t *testing.T) {
	t.Parallel()
	_, err := e.FuturesGetMarketDepth(t.Context(), btccwPair, "step5")
	require.NoError(t, err)
}

func TestFuturesGetKline(t *testing.T) {
	t.Parallel()
	_, err := e.FuturesGetKline(t.Context(), btccwPair, "5min", 5, time.Now().Add(-time.Minute*5), time.Now())
	require.NoError(t, err)
	_, err = e.FuturesGetKline(t.Context(), btccwPair, "invalid", 5, time.Time{}, time.Time{})
	require.ErrorIs(t, err, common.ErrInvalidPeriod, "FGetKlineData must reject an invalid period")
	_, err = e.FuturesGetKline(t.Context(), btccwPair, "5min", 5, time.Now(), time.Time{})
	require.ErrorIs(t, err, common.ErrDateUnset, "FGetKlineData must reject a half-open interval")
}

func TestFuturesGetMarketOverview(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodGet, futuresMarketOverview, `{"status":"ok","tick":{"id":123}}`, nil)
	resp, err := h.FuturesGetMarketOverview(t.Context(), btccwPair)
	require.NoError(t, err, "FGetMarketOverviewData must not error")
	assert.Equal(t, uint64(123), resp.Tick.ID, "market overview ID should decode")
}

func TestFuturesLastTrade(t *testing.T) {
	t.Parallel()
	_, err := e.FuturesLastTrade(t.Context(), btccwPair)
	require.NoError(t, err)
}

func TestFuturesRequestPublicBatchTrades(t *testing.T) {
	t.Parallel()
	_, err := e.FuturesRequestPublicBatchTrades(t.Context(), btccwPair, 50)
	require.NoError(t, err)
}

func TestFuturesQueryTieredAdjustmentFactor(t *testing.T) {
	t.Parallel()
	_, err := e.FuturesQueryTieredAdjustmentFactor(t.Context(), currency.BTC)
	require.NoError(t, err)
}

func TestFuturesQueryHisOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := e.FuturesQueryHisOpenInterest(t.Context(), "BTC", "this_week", "60min", "cont", 3)
	require.NoError(t, err)
	for _, size := range []uint64{0, 201} {
		t.Run(strconv.FormatUint(size, 10), func(t *testing.T) {
			t.Parallel()
			_, err := e.FuturesQueryHisOpenInterest(t.Context(), "BTC", "this_week", "60min", "cont", size)
			require.ErrorIs(t, err, errInvalidSize, "FQueryHisOpenInterest must reject an invalid size")
		})
	}
}

func TestFuturesQuerySystemStatus(t *testing.T) {
	t.Parallel()
	_, err := e.FuturesQuerySystemStatus(t.Context(), currency.BTC)
	require.NoError(t, err)
}

func TestFuturesQueryTopAccountsRatio(t *testing.T) {
	t.Parallel()
	_, err := e.FuturesQueryTopAccountsRatio(t.Context(), "BTC", "5min")
	require.NoError(t, err)
}

func TestFuturesQueryTopPositionsRatio(t *testing.T) {
	t.Parallel()
	_, err := e.FuturesQueryTopPositionsRatio(t.Context(), "BTC", "5min")
	require.NoError(t, err)
}

func TestFuturesLiquidationOrders(t *testing.T) {
	t.Parallel()
	_, err := e.FuturesLiquidationOrders(t.Context(), currency.BTC, "filled", 0, 0, "", 0)
	require.NoError(t, err, "FLiquidationOrders must not error")
	for _, tc := range []struct {
		name       string
		start, end uint64
	}{
		{name: "both timestamps", start: 1604367600000, end: 1604367700000},
		{name: "start only", start: 1604367600000},
		{name: "end only", end: 1604367700000},
		{name: "no timestamps"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodGet, futuresLiquidationOrders, emptySuccessResponse, func(r *http.Request) {
				for key, timestamp := range map[string]uint64{"start_time": tc.start, "end_time": tc.end} {
					if timestamp == 0 {
						assert.NotContains(t, r.URL.Query(), key, "unset timestamps should be omitted")
					} else {
						assert.Equal(t, strconv.FormatUint(timestamp, 10), r.URL.Query().Get(key), "timestamp should match its argument")
					}
				}
			})
			_, err := h.FuturesLiquidationOrders(t.Context(), currency.BTC, "filled", tc.start, tc.end, "", 0)
			require.NoError(t, err, "FLiquidationOrders must accept optional timestamps")
		})
	}
}

func TestFuturesIndexKline(t *testing.T) {
	t.Parallel()
	_, err := e.FuturesIndexKline(t.Context(), btccwPair, "5min", 5)
	require.NoError(t, err)
}

func TestFuturesGetBasis(t *testing.T) {
	t.Parallel()
	_, err := e.FuturesGetBasis(t.Context(), btccwPair, "5min", "open", 3)
	require.NoError(t, err)
}

func TestFuturesGetAccountInfo(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, futuresAccount, emptySuccessResponse, nil)
	_, err := h.FuturesGetAccountInfo(t.Context(), currency.BTC)
	require.NoError(t, err, "FGetAccountInfo must not error")
}

func TestFuturesGetPositionsInfo(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, futuresPositionInformation, emptySuccessResponse, nil)
	_, err := h.FuturesGetPositionsInfo(t.Context(), currency.BTC)
	require.NoError(t, err, "FGetPositionsInfo must not error")
}

func TestFuturesGetAllSubAccountAssets(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, futuresAllSubAccountAssets, emptySuccessResponse, nil)
	_, err := h.FuturesGetAllSubAccountAssets(t.Context(), currency.BTC)
	require.NoError(t, err, "FGetAllSubAccountAssets must not error")
}

func TestFuturesGetSingleSubAccountInfo(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, futuresSingleSubAccountAssets, emptySuccessResponse, nil)
	_, err := h.FuturesGetSingleSubAccountInfo(t.Context(), "BTC", "154263566")
	require.NoError(t, err, "FGetSingleSubAccountInfo must not error")
}

func TestFuturesGetSingleSubPositions(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, futuresSingleSubAccountPositions, emptySuccessResponse, nil)
	_, err := h.FuturesGetSingleSubPositions(t.Context(), "BTC", "154263566")
	require.NoError(t, err, "FGetSingleSubPositions must not error")
}

func TestFuturesGetFinancialRecords(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "/api/v3/contract_financial_record", futuresFinancialRecords, "delivery futures financial records endpoint should match HTX docs")
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, futuresFinancialRecords, emptySuccessResponse, func(r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err, "request body must be readable")
		var requestBody map[string]any
		require.NoError(t, json.Unmarshal(body, &requestBody), "request body must decode")
		assert.Equal(t, float64(20), requestBody["limit"], "page size should be sent as the V3 limit")
	})
	_, err := h.FuturesGetFinancialRecords(t.Context(), "BTC", "closeLong", 2, 1, 20)
	require.NoError(t, err, "FGetFinancialRecords must not error")
	_, err = h.FuturesGetFinancialRecords(t.Context(), "BTC", "closeLong", 3, 1, 20)
	require.ErrorIs(t, err, errInvalidLookbackDays, "FGetFinancialRecords must reject lookbacks over two days")
}

func TestFuturesGetSettlementRecords(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, futuresSettlementRecords, emptySuccessResponse, nil)
	start := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
	_, err := h.FuturesGetSettlementRecords(t.Context(), currency.BTC, 1, 20, start, start.Add(time.Hour))
	require.NoError(t, err, "FGetSettlementRecords must not error")
	_, err = h.FuturesGetSettlementRecords(t.Context(), currency.BTC, 1, 20, start, time.Time{})
	require.ErrorIs(t, err, common.ErrDateUnset, "FGetSettlementRecords must reject a half-open interval")
}

func TestFuturesGetOrderLimits(t *testing.T) {
	t.Parallel()
	t.Run("invalid order price type", func(t *testing.T) {
		t.Parallel()
		h := new(Exchange)
		require.NoError(t, testexch.Setup(h), "HTX setup must not error")
		_, err := h.FuturesGetOrderLimits(t.Context(), "BTC", "not-real")
		require.Error(t, err, "FGetOrderLimits must reject invalid order price type")
	})
	t.Run("success", func(t *testing.T) {
		t.Parallel()
		h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, futuresOrderLimitInfo, emptySuccessResponse, nil)
		_, err := h.FuturesGetOrderLimits(t.Context(), "BTC", "limit")
		require.NoError(t, err, "FGetOrderLimits must not error")
	})
}

func TestFuturesContractTradingFee(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, futuresContractTradingFee, emptySuccessResponse, nil)
	_, err := h.FuturesContractTradingFee(t.Context(), currency.BTC)
	require.NoError(t, err, "FuturesContractTradingFee must not error")
}

func TestFuturesGetTransferLimits(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, futuresTransferLimitInfo, emptySuccessResponse, nil)
	_, err := h.FuturesGetTransferLimits(t.Context(), currency.BTC)
	require.NoError(t, err, "FGetTransferLimits must not error")
}

func TestFuturesGetPositionLimits(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, futuresPositionLimitInfo, emptySuccessResponse, nil)
	_, err := h.FuturesGetPositionLimits(t.Context(), currency.BTC)
	require.NoError(t, err, "FGetPositionLimits must not error")
}

func TestFuturesGetAssetsAndPositions(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, futuresQueryAssetsAndPositions, emptySuccessResponse, nil)
	_, err := h.FuturesGetAssetsAndPositions(t.Context(), currency.BTC)
	require.NoError(t, err, "FGetAssetsAndPositions must not error")
}

func TestFuturesTransfer(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, futuresTransfer, emptySuccessResponse, nil)
	_, err := h.FuturesTransfer(t.Context(), "154263566", "BTC", "sub_to_master", 5)
	require.NoError(t, err, "FTransfer must not error")
}

func TestFuturesGetTransferRecords(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, futuresTransferRecords, emptySuccessResponse, nil)
	_, err := h.FuturesGetTransferRecords(t.Context(), "BTC", "master_to_sub", 2, 1, 20)
	require.NoError(t, err, "FGetTransferRecords must not error")
}

func TestFuturesGetAvailableLeverage(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, futuresAvailableLeverage, emptySuccessResponse, nil)
	_, err := h.FuturesGetAvailableLeverage(t.Context(), currency.BTC)
	require.NoError(t, err, "FGetAvailableLeverage must not error")
}

func TestFuturesOrder(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name             string
		priceType        string
		clientOrderID    string
		expectedClientID uint64
		expectedErr      error
	}{
		{name: "limit", priceType: orderPriceTypeLimit, clientOrderID: "123", expectedClientID: 123},
		{name: "post only", priceType: orderPriceTypePostOnly, clientOrderID: "123", expectedClientID: 123},
		{name: "no client ID", priceType: orderPriceTypeLimit},
		{name: "minimum client ID", priceType: orderPriceTypeLimit, clientOrderID: "1", expectedClientID: 1},
		{name: "maximum client ID", priceType: orderPriceTypeLimit, clientOrderID: "9223372036854775807", expectedClientID: 9223372036854775807},
		{name: "zero client ID", priceType: orderPriceTypeLimit, clientOrderID: "0", expectedErr: errInvalidClientOrderID},
		{name: "negative client ID", priceType: orderPriceTypeLimit, clientOrderID: "-1", expectedErr: errInvalidClientOrderID},
		{name: "overflow client ID", priceType: orderPriceTypeLimit, clientOrderID: "9223372036854775808", expectedErr: errInvalidClientOrderID},
		{name: "invalid client ID", priceType: orderPriceTypeLimit, clientOrderID: "invalid", expectedErr: errInvalidClientOrderID},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, futuresOrder, emptySuccessResponse, func(r *http.Request) {
				assert.NoError(t, tc.expectedErr, "invalid client IDs should not reach the endpoint")
				var body map[string]json.RawMessage
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body), "order request must decode")
				assert.JSONEq(t, `"`+tc.priceType+`"`, string(body["order_price_type"]), "order price type should match")
				assert.Equal(t, "1", string(body["lever_rate"]), "leverage should match the requested integer")
				if tc.clientOrderID == "" {
					assert.NotContains(t, body, "client_order_id", "unset client IDs should be omitted")
				} else {
					assert.Equal(t, strconv.FormatUint(tc.expectedClientID, 10), string(body["client_order_id"]), "client ID should retain precision as a JSON integer")
				}
			})
			_, err := h.FuturesOrder(t.Context(), currency.EMPTYPAIR, "BTC", "quarter", tc.clientOrderID, "buy", "open", tc.priceType, 1, 1, 1)
			if tc.expectedErr != nil {
				require.ErrorIs(t, err, tc.expectedErr, "FuturesOrder must reject client IDs outside the documented range")
				return
			}
			require.NoError(t, err, "FuturesOrder must accept valid client order IDs")
		})
	}
}

func TestFuturesPlaceBatchOrder(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, futuresBatchOrder, emptySuccessResponse, nil)
	_, err := h.FuturesPlaceBatchOrder(t.Context(), []FuturesBatchOrderRequest{{
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

func TestFuturesCancelOrder(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name           string
		orderIDs       string
		clientOrderIDs string
		expectedErr    error
	}{
		{name: "one order", orderIDs: "123"},
		{name: "one client order", clientOrderIDs: "456"},
		{name: "ten orders", orderIDs: "1,2,3,4,5,6,7,8,9,10"},
		{name: "ten client orders", clientOrderIDs: "1,2,3,4,5,6,7,8,9,10"},
		{name: "missing identifiers", expectedErr: order.ErrOrderIDNotSet},
		{name: "mixed identifiers", orderIDs: "123", clientOrderIDs: "456", expectedErr: errMixedOrderIdentifiers},
		{name: "too many orders", orderIDs: "1,2,3,4,5,6,7,8,9,10,11", expectedErr: errCancelOrderLimitExceeded},
		{name: "too many client orders", clientOrderIDs: "1,2,3,4,5,6,7,8,9,10,11", expectedErr: errCancelOrderLimitExceeded},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, futuresCancelOrder, emptySuccessResponse, func(r *http.Request) {
				require.NoError(t, tc.expectedErr, "invalid cancellation must not send a request")
				var request map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&request), "cancellation request must decode")
				assert.Equal(t, "BTC", request["symbol"], "symbol should match")
				if tc.orderIDs != "" {
					assert.Equal(t, tc.orderIDs, request["order_id"], "order IDs should be sent in one request")
					assert.NotContains(t, request, "client_order_id", "order cancellation should omit client IDs")
				} else {
					assert.Equal(t, tc.clientOrderIDs, request["client_order_id"], "client IDs should be sent in one request")
					assert.NotContains(t, request, "order_id", "client cancellation should omit order IDs")
				}
			})
			_, err := h.FuturesCancelOrder(t.Context(), currency.BTC, tc.orderIDs, tc.clientOrderIDs)
			if tc.expectedErr != nil {
				require.ErrorIs(t, err, tc.expectedErr, "FuturesCancelOrder must reject invalid identifiers")
				return
			}
			require.NoError(t, err, "FuturesCancelOrder must send one cancellation request")
		})
	}
}

func TestFuturesCancelAllOrders(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, futuresCancelAllOrders, emptySuccessResponse, nil)
	_, err := h.FuturesCancelAllOrders(t.Context(), currency.EMPTYPAIR, "BTC", "quarter")
	require.NoError(t, err, "FCancelAllOrders must not error")
}

func TestFuturesFlashCloseOrder(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, futuresFlashCloseOrder, emptySuccessResponse, nil)
	_, err := h.FuturesFlashCloseOrder(t.Context(), currency.EMPTYPAIR, "BTC", "quarter", "buy", "lightning", "", 1)
	require.NoError(t, err, "FFlashCloseOrder must not error")
}

func TestFuturesGetOrderInfo(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, futuresOrderInfo, emptySuccessResponse, nil)
	_, err := h.FuturesGetOrderInfo(t.Context(), "BTC", "", "123")
	require.NoError(t, err, "FGetOrderInfo must not error")
}

func TestFuturesOrderDetails(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, futuresOrderDetails, emptySuccessResponse, nil)
	_, err := h.FuturesOrderDetails(t.Context(), "BTC", "123", "quotation", time.Now().Add(-time.Hour), 1, 20)
	require.NoError(t, err, "FuturesOrderDetails must not error")
}

func TestFuturesGetOpenOrders(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, futuresQueryOpenOrders, emptySuccessResponse, nil)
	_, err := h.FuturesGetOpenOrders(t.Context(), currency.BTC, 1, 20)
	require.NoError(t, err, "FGetOpenOrders must not error")
}

func TestFuturesGetOrderHistory(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "/api/v3/contract_hisorders", futuresOrderHistory, "delivery futures order history endpoint should match HTX docs")
	_, err := e.FuturesGetOrderHistory(t.Context(),
		currency.EMPTYPAIR, "BTC",
		"all", "all", "limit",
		[]order.Status{},
		3, 0, 0)
	require.ErrorIs(t, err, errInvalidLookbackDays, "FGetOrderHistory must reject lookbacks over two days")
}

func TestFuturesGetOrderHistoryByTimeRange(t *testing.T) {
	t.Parallel()
	body := make(chan []byte, 1)
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, futuresOrderHistory, `{"code":200,"data":[{"query_id":12,"order_id":34,"order_id_str":"34","contract_code":"BTC-USD","direction":"buy","order_price_type":"limit","status":6}]}`, func(r *http.Request) {
		payload, err := io.ReadAll(r.Body)
		assert.NoError(t, err, "request body should be readable")
		body <- payload
	})
	startTime := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
	endTime := startTime.Add(48 * time.Hour)
	resp, err := h.FuturesGetOrderHistoryByTimeRange(t.Context(), currency.EMPTYPAIR, "BTC", "all", "all", "limit", nil, startTime, endTime, 11, 50)
	require.NoError(t, err, "FGetOrderHistoryByTimeRange must not error")
	require.Len(t, resp.Data.Orders, 1, "decoded order history must be returned")
	assert.Equal(t, uint64(12), resp.Data.Orders[0].QueryID, "query ID should decode")

	var requestBody map[string]any
	require.NoError(t, json.Unmarshal(<-body, &requestBody), "request body must decode")
	assert.Equal(t, float64(startTime.UnixMilli()), requestBody["start_time"], "start time should be preserved")
	assert.Equal(t, float64(endTime.UnixMilli()), requestBody["end_time"], "end time should be preserved")
	assert.Equal(t, v3HistoryDirectionNext, requestBody["direct"], "pagination direction should move forwards")
	assert.Equal(t, float64(11), requestBody["from_id"], "cursor should be preserved")
	assert.Equal(t, float64(50), requestBody["limit"], "limit should be preserved")

	_, err = h.FuturesGetOrderHistoryByTimeRange(t.Context(), currency.EMPTYPAIR, "BTC", "all", "all", "limit", nil, startTime, endTime.Add(time.Millisecond), 0, 50)
	require.ErrorIs(t, err, errHistoryTimeRangeExceeded, "FGetOrderHistoryByTimeRange must reject intervals over 48 hours")
	_, err = h.FuturesGetOrderHistoryByTimeRange(t.Context(), currency.EMPTYPAIR, "BTC", "all", "all", "limit", nil, endTime, startTime, 0, 50)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "FGetOrderHistoryByTimeRange must reject reversed intervals")
}

func TestFuturesTradeHistory(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "/api/v3/contract_matchresults", futuresMatchResult, "delivery futures trade history endpoint should match HTX docs")
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, futuresMatchResult, emptySuccessResponse, func(r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err, "request body must be readable")
		var requestBody map[string]any
		require.NoError(t, json.Unmarshal(body, &requestBody), "request body must decode")
		assert.Equal(t, float64(20), requestBody["limit"], "page size should be sent as the V3 limit")
	})
	_, err := h.FuturesTradeHistory(t.Context(), currency.EMPTYPAIR, "BTC", "all", 2, 1, 20)
	require.NoError(t, err, "FTradeHistory must not error")
}

func TestFuturesPlaceTriggerOrder(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, futuresTriggerOrder, emptySuccessResponse, nil)
	_, err := h.FuturesPlaceTriggerOrder(t.Context(), currency.EMPTYPAIR, "BTC", "quarter", "greaterOrEqual", "limit", "buy", "close", 1.1, 1.05, 5, 2)
	require.NoError(t, err, "FPlaceTriggerOrder must not error")
}

func TestFuturesCancelTriggerOrder(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, futuresCancelTriggerOrder, emptySuccessResponse, nil)
	_, err := h.FuturesCancelTriggerOrder(t.Context(), "BTC", "123")
	require.NoError(t, err, "FCancelTriggerOrder must not error")
}

func TestFuturesCancelAllTriggerOrders(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, futuresCancelAllTriggerOrders, emptySuccessResponse, nil)
	_, err := h.FuturesCancelAllTriggerOrders(t.Context(), currency.EMPTYPAIR, "BTC", "this_week")
	require.NoError(t, err, "FCancelAllTriggerOrders must not error")
}

func TestFuturesQueryTriggerOpenOrders(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, futuresTriggerOpenOrders, emptySuccessResponse, nil)
	_, err := h.FuturesQueryTriggerOpenOrders(t.Context(), currency.EMPTYPAIR, "BTC", 1, 20)
	require.NoError(t, err, "FQueryTriggerOpenOrders must not error")
}

func TestFuturesQueryTriggerOrderHistory(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, futuresTriggerOrderHistory, emptySuccessResponse, nil)
	_, err := h.FuturesQueryTriggerOrderHistory(t.Context(), currency.EMPTYPAIR, "BTC", "all", "all", 10, 1, 20)
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

func TestFuturesSwitchLeverage(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, futuresSwitchLeverage, `{"status":"ok","data":{"symbol":"BTC","lever_rate":5}}`, func(r *http.Request) {
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err, "request body should be readable")
		assert.JSONEq(t, `{"symbol":"BTC","lever_rate":5}`, string(body), "delivery leverage body should match")
	})
	require.NoError(t, h.FuturesSwitchLeverage(t.Context(), currency.BTC, 5), "FSwitchLeverage must not error")
}
