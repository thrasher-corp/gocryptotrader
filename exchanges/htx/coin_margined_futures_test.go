package htx

import (
	"io"
	"maps"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
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
	for _, tc := range []struct {
		name       string
		start, end time.Time
		want       error
	}{
		{name: "missing start", end: time.Now(), want: common.ErrDateUnset},
		{name: "reversed range", start: time.Unix(200, 0), end: time.Unix(100, 0), want: common.ErrStartAfterEnd},
		{name: "equal range", start: time.Unix(100, 0), end: time.Unix(100, 0), want: common.ErrStartEqualsEnd},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetSwapKlineData(t.Context(), btcusdPair, "5min", 5, tc.start, tc.end)
			require.ErrorIs(t, err, tc.want, "invalid ranges must fail before transport")
		})
	}

	r, err := e.GetSwapKlineData(t.Context(), btcusdPair, "5min", 5, time.Now().Add(-time.Hour), time.Now())
	require.NoError(t, err)
	assert.NotEmpty(t, r.Candles, "GetSwapKlineData should return some data")
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
	t.Run("above maximum size", func(t *testing.T) {
		t.Parallel()
		_, err := e.GetOpenInterestInfo(t.Context(), btcusdPair, "5min", "cryptocurrency", 1201)
		require.ErrorIs(t, err, errInvalidSize, "size above the maximum must be rejected")
	})
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
	t.Run("invalid period", func(t *testing.T) {
		t.Parallel()
		_, err := e.GetTraderSentimentIndexAccount(t.Context(), btcusdPair, "bogus")
		require.ErrorIs(t, err, common.ErrInvalidPeriod, "invalid periods must be rejected")
	})
	_, err := e.GetTraderSentimentIndexAccount(t.Context(), btcusdPair, "5min")
	require.NoError(t, err)
}

func TestGetTraderSentimentIndexPosition(t *testing.T) {
	t.Parallel()
	t.Run("invalid period", func(t *testing.T) {
		t.Parallel()
		_, err := e.GetTraderSentimentIndexPosition(t.Context(), btcusdPair, "bogus")
		require.ErrorIs(t, err, common.ErrInvalidPeriod, "invalid periods must be rejected")
	})
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
	t.Run("above maximum size", func(t *testing.T) {
		t.Parallel()
		_, err := e.GetPremiumIndexKlineData(t.Context(), btcusdPair, "5min", 1201)
		require.ErrorIs(t, err, errInvalidSize, "size above the maximum must be rejected")
	})
	_, err := e.GetPremiumIndexKlineData(t.Context(), btcusdPair, "5min", 15)
	require.NoError(t, err)
	_, err = e.GetPremiumIndexKlineData(t.Context(), btcusdPair, "invalid", 15)
	require.ErrorIs(t, err, common.ErrInvalidPeriod, "GetPremiumIndexKlineData must reject an invalid period")
	_, err = e.GetPremiumIndexKlineData(t.Context(), btcusdPair, "5min", 0)
	require.ErrorIs(t, err, errInvalidSize, "GetPremiumIndexKlineData must reject an invalid size")
}

func TestGetEstimatedFundingRates(t *testing.T) {
	t.Parallel()
	t.Run("above maximum size", func(t *testing.T) {
		t.Parallel()
		_, err := e.GetEstimatedFundingRates(t.Context(), btcusdPair, "5min", 1201)
		require.ErrorIs(t, err, errInvalidSize, "size above the maximum must be rejected")
	})
	_, err := e.GetEstimatedFundingRates(t.Context(), btcusdPair, "5min", 15)
	require.NoError(t, err)

	_, err = e.GetEstimatedFundingRates(t.Context(), btcusdPair, "invalid", 15)
	require.ErrorIs(t, err, common.ErrInvalidPeriod, "GetEstimatedFundingRates must reject invalid period")
	_, err = e.GetEstimatedFundingRates(t.Context(), btcusdPair, "5min", 0)
	require.ErrorIs(t, err, errInvalidSize, "GetEstimatedFundingRates must reject an invalid size")
}

func TestGetBasisData(t *testing.T) {
	t.Parallel()
	t.Run("above maximum size", func(t *testing.T) {
		t.Parallel()
		_, err := e.GetBasisData(t.Context(), btcusdPair, "5min", "open", 1201)
		require.ErrorIs(t, err, errInvalidSize, "size above the maximum must be rejected")
	})
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
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_sub_account_list", `{"status":"ok","data":[{"sub_uid":123,"list":[{"symbol":"ETH","contract_code":"ETH-USD","margin_balance":0.125}]}]}`, nil)
	response, err := h.GetSwapAllSubAccAssets(t.Context(), ethusdPair)
	require.NoError(t, err, "GetSwapAllSubAccAssets must decode decimal balances")
	require.Len(t, response.Assets, 1, "response must contain the subaccount")
	require.Len(t, response.Assets[0].Balances, 1, "subaccount must contain its balance")
	assert.Equal(t, uint64(123), response.Assets[0].SubUID, "subaccount ID should decode")
	assert.Equal(t, 0.125, response.Assets[0].Balances[0].MarginBalance, "fractional margin balance should retain its decimals")
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
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_master_sub_transfer", `{"status":"ok","ts":1603872000000,"data":{"order_id":"123"}}`, nil)
	response, err := h.AccountTransferData(t.Context(), ethusdPair, "123", "master_to_sub", 15)
	require.NoError(t, err, "AccountTransferData must not error")
	require.NotNil(t, response, "transfer response must be returned as a pointer")
	assert.Equal(t, time.UnixMilli(1603872000000), response.Timestamp.Time(), "transfer timestamp should retain milliseconds")
	assert.Equal(t, "123", response.Transfer.OrderID, "transfer identifier should decode")
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
	_, err = h.PlaceSwapOrders(t.Context(), ethusdPair, "", "buy", "open", orderPriceTypePostOnly, 0.01, 1, 1)
	require.NoError(t, err, "post-only orders must be accepted")
	_, err = h.PlaceSwapOrders(t.Context(), ethusdPair, "", "buy", "open", "invalid", 0.01, 1, 1)
	require.ErrorIs(t, err, errInvalidOrderType, "PlaceSwapOrders must reject an invalid order type")
}

func TestPlaceSwapBatchOrders(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name       string
		count      int
		nilRequest bool
		nilOrders  bool
		priceType  string
		wantErr    error
	}
	for _, tc := range []testCase{
		{name: "nil request", nilRequest: true, wantErr: common.ErrNilPointer},
		{name: "nil orders", nilOrders: true, wantErr: errEmptyOrderBatch},
		{name: "empty batch", wantErr: errEmptyOrderBatch},
		{name: "too many orders", count: 11, wantErr: errBatchOrderLimitExceeded},
		{name: "limit order", count: 1, priceType: "limit"},
		{name: "post-only order", count: 1, priceType: "post_only"},
		{name: "maximum batch", count: 10, priceType: "limit"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var calls atomic.Int32
			h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_batchorder", emptySuccessResponse, func(r *http.Request) {
				calls.Add(1)
				type orderPayload struct {
					Orders []map[string]any `json:"orders_data"`
				}
				var payload orderPayload
				if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&payload), "batch payload should decode") {
					return
				}
				assert.Len(t, payload.Orders, tc.count, "batch payload should retain every order")
				for _, row := range payload.Orders {
					assert.Equal(t, "ETH-USD", row["contract_code"], "contract code should retain its exchange format")
					assert.Equal(t, float64(1), row["lever_rate"], "leverage should use the documented key")
					assert.Equal(t, tc.priceType, row["order_price_type"], "price instruction should use the documented key")
					assert.Equal(t, float64(5), row["price"], "order price should be retained")
					assert.Equal(t, float64(1), row["volume"], "order volume should be retained")
					assert.Equal(t, "buy", row["direction"], "order direction should be retained")
					assert.Equal(t, "open", row["offset"], "order offset should be retained")
					assert.NotContains(t, row, "leverRate", "undocumented leverage key should not be sent")
					assert.NotContains(t, row, "orderPriceType", "undocumented price instruction key should not be sent")
				}
			})
			var req *SwapBatchOrderRequest
			if !tc.nilRequest {
				req = &SwapBatchOrderRequest{Orders: make([]SwapBatchOrder, tc.count)}
				if tc.nilOrders {
					req.Orders = nil
				}
				for i := range req.Orders {
					req.Orders[i] = SwapBatchOrder{ContractCode: "ETH-USD", Price: 5, Volume: 1, Direction: "buy", Offset: "open", LeverageRate: 1, OrderPriceType: tc.priceType}
				}
			}
			_, err := h.PlaceSwapBatchOrders(t.Context(), req)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr, "invalid batch must return the expected validation error")
				assert.Zero(t, calls.Load(), "invalid batch should not send a request")
				return
			}
			require.NoError(t, err, "valid batch must be accepted")
			assert.Equal(t, int32(1), calls.Load(), "one batch should send exactly one request")
		})
	}
}

func TestCancelSwapOrder(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name           string
		orderIDs       string
		clientOrderIDs string
		wantErr        error
	}
	for _, tc := range []testCase{
		{name: "order ID", orderIDs: "123"},
		{name: "client order ID", clientOrderIDs: "123"},
		{name: "ten order IDs", orderIDs: strings.Repeat("123,", 9) + "123"},
		{name: "ten client order IDs", clientOrderIDs: strings.Repeat("123,", 9) + "123"},
		{name: "missing IDs", wantErr: order.ErrOrderIDNotSet},
		{name: "mixed IDs", orderIDs: "123", clientOrderIDs: "456", wantErr: errMixedOrderIdentifiers},
		{name: "eleven order IDs", orderIDs: strings.Repeat("123,", 10) + "123", wantErr: errCancelOrderLimitExceeded},
		{name: "eleven client order IDs", clientOrderIDs: strings.Repeat("123,", 10) + "123", wantErr: errCancelOrderLimitExceeded},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			calls := make(chan bool, 1)
			h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_cancel", emptySuccessResponse, func(r *http.Request) {
				calls <- true
				var payload map[string]string
				require.NoError(t, json.NewDecoder(r.Body).Decode(&payload), "cancellation body must decode")
				assert.Equal(t, tc.orderIDs, payload["order_id"], "order IDs should be retained")
				assert.Equal(t, tc.clientOrderIDs, payload["client_order_id"], "client order IDs should be retained")
			})
			_, err := h.CancelSwapOrder(t.Context(), tc.orderIDs, tc.clientOrderIDs, ethusdPair)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr, "CancelSwapOrder must reject invalid identifier combinations")
				assert.Empty(t, calls, "invalid cancellations should not send a request")
				return
			}
			require.NoError(t, err, "CancelSwapOrder must accept a supported batch")
			assert.Len(t, calls, 1, "a cancellation batch should send one request")
		})
	}
}

func TestCancelAllSwapOrders(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_cancelall", emptySuccessResponse, nil)
	_, err := h.CancelAllSwapOrders(t.Context(), ethusdPair)
	require.NoError(t, err, "CancelAllSwapOrders must not error")
}

func TestPlaceLightningCloseOrder(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name          string
		clientOrderID uint64
		priceType     string
		wantErr       error
	}
	for _, tc := range []testCase{
		{name: "client order ID", clientOrderID: 1, priceType: "lightning"},
		{name: "omitted client order ID", priceType: "lightning"},
		{name: "maximum client order ID", clientOrderID: math.MaxInt64, priceType: "lightning"},
		{name: "client order ID over maximum", clientOrderID: math.MaxInt64 + 1, priceType: "lightning", wantErr: errInvalidClientOrderID},
		{name: "invalid price type", clientOrderID: 1, priceType: "invalid", wantErr: errInvalidOrderPriceType},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			calls := make(chan bool, 1)
			h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_lightning_close_position", emptySuccessResponse, func(r *http.Request) {
				calls <- true
				type lightningRequest struct {
					ClientOrderID *uint64 `json:"client_order_id"`
				}
				var payload lightningRequest
				require.NoError(t, json.NewDecoder(r.Body).Decode(&payload), "lightning close body must decode")
				if tc.clientOrderID == 0 {
					assert.Nil(t, payload.ClientOrderID, "zero client order ID should be omitted")
					return
				}
				require.NotNil(t, payload.ClientOrderID, "client order ID must be present")
				assert.Equal(t, tc.clientOrderID, *payload.ClientOrderID, "client order ID should be retained")
			})
			_, err := h.PlaceLightningCloseOrder(t.Context(), ethusdPair, "buy", tc.priceType, 5, tc.clientOrderID)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr, "PlaceLightningCloseOrder must reject invalid parameters")
				assert.Empty(t, calls, "invalid lightning close orders should not send a request")
				return
			}
			require.NoError(t, err, "PlaceLightningCloseOrder must accept valid parameters")
			assert.Len(t, calls, 1, "lightning close should send one request")
		})
	}
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
	_, err = e.GetSwapOrderHistory(t.Context(), ethusdPair, "all", "all", nil, 3, 0, 0)
	require.ErrorIs(t, err, errInvalidLookbackDays, "GetSwapOrderHistory must reject a three-day lookback")
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
	require.Len(t, resp.OrderHistory.Orders, 1, "decoded order history must be returned")
	assert.Equal(t, uint64(12), resp.OrderHistory.Orders[0].QueryID, "query ID should decode")

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
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodGet, "/swap-api/v1/swap_contract_info", `{"status":"ok","data":[{"contract_code":"BTC-USD","create_date":"20200325"}]}`, nil)
	response, err := h.GetSwapMarkets(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err, "GetSwapMarkets must decode market metadata")
	require.Len(t, response, 1, "decoded response must retain its contract")
	assert.Equal(t, "BTC-USD", response[0].ContractCode, "contract code should decode")
	assert.Equal(t, time.Date(2020, time.March, 25, 0, 0, 0, 0, time.UTC), response[0].CreateDate.Time(), "contract listing date should decode")
}

func TestGetSwapFundingRates(t *testing.T) {
	t.Parallel()
	_, err := e.GetSwapFundingRates(t.Context())
	require.NoError(t, err)
}

func TestGetSwapFundingRate(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodGet, "/swap-api/v1/swap_funding_rate", `{"status":"ok","data":{"contract_code":"BTC-USD","funding_rate":"0.001","funding_time":"1603872000000"}}`, nil)
	response, err := h.GetSwapFundingRate(t.Context(), btcusdPair)
	require.NoError(t, err, "GetSwapFundingRate must decode the funding response")
	require.NotNil(t, response, "funding result must be returned as a pointer")
	assert.Equal(t, "BTC-USD", response.ContractCode, "funding contract code should decode")
	assert.Equal(t, 0.001, response.FundingRate.Float64(), "quoted funding rate should decode")
	assert.Equal(t, time.UnixMilli(1603872000000), response.FundingTime.Time(), "funding time should retain milliseconds")
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
	require.Len(t, got.FundingHistory.FundingRates, 1, "one funding rate must be returned")
	assert.Equal(t, time.UnixMilli(1604312615051), got.FundingHistory.FundingRates[0].FundingTime.Time(), "funding time should match")
}

func TestSwitchCoinMarginedLeverage(t *testing.T) {
	t.Parallel()
	t.Run("dedicated rate limit", func(t *testing.T) {
		t.Parallel()
		var sent atomic.Int32
		h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_switch_lever_rate", `{"status":"ok","data":{}}`, func(*http.Request) { sent.Add(1) })
		// A long refill interval makes routing observable without timing assertions or sleeps.
		limits := maps.Clone(h.Requester.GetRateLimiterDefinitions())
		request.WithLimiter(limits)(h.Requester)
		limits[htxSwapSwitchLeverage] = request.NewRateLimitWithWeight(time.Hour, 1, 1)
		require.NoError(t, limits[htxSwapSwitchLeverage].RateLimit(t.Context()), "dedicated allowance must be exhausted")
		err := h.SwitchCoinMarginedLeverage(request.WithDelayNotAllowed(t.Context()), btcusdPair, 5)
		require.ErrorIs(t, err, request.ErrDelayNotAllowed, "endpoint must use its exhausted dedicated allowance")
		assert.Zero(t, sent.Load(), "rate-limited request should not reach the server")
	})
	h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_switch_lever_rate", `{"status":"ok","data":{"contract_code":"BTC-USD","lever_rate":5}}`, func(r *http.Request) {
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err, "request body should be readable")
		assert.JSONEq(t, `{"contract_code":"BTC-USD","lever_rate":5}`, string(body), "coin-margined leverage body should match")
	})
	require.NoError(t, h.SwitchCoinMarginedLeverage(t.Context(), btcusdPair, 5), "SwitchCoinMarginedLeverage must not error")
}
