package htx

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestFuturesAuthenticatedEndpoints(t *testing.T) {
	t.Parallel()
	startTime := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name string
		path string
		call func(*Exchange) error
	}{
		{
			name: "FGetAccountInfo",
			path: fAccountData,
			call: func(h *Exchange) error {
				_, err := h.FGetAccountInfo(t.Context(), currency.BTC)
				return err
			},
		},
		{
			name: "FGetPositionsInfo",
			path: fPositionInformation,
			call: func(h *Exchange) error {
				_, err := h.FGetPositionsInfo(t.Context(), currency.BTC)
				return err
			},
		},
		{
			name: "FGetAllSubAccountAssets",
			path: fAllSubAccountAssets,
			call: func(h *Exchange) error {
				_, err := h.FGetAllSubAccountAssets(t.Context(), currency.BTC)
				return err
			},
		},
		{
			name: "FGetSingleSubAccountInfo",
			path: fSingleSubAccountAssets,
			call: func(h *Exchange) error {
				_, err := h.FGetSingleSubAccountInfo(t.Context(), "BTC", "123")
				return err
			},
		},
		{
			name: "FGetSingleSubPositions",
			path: fSingleSubAccountPositions,
			call: func(h *Exchange) error {
				_, err := h.FGetSingleSubPositions(t.Context(), "BTC", "123")
				return err
			},
		},
		{
			name: "FGetFinancialRecords",
			path: fFinancialRecords,
			call: func(h *Exchange) error {
				_, err := h.FGetFinancialRecords(t.Context(), "BTC", "closeLong", 2, 1, 20)
				return err
			},
		},
		{
			name: "FGetSettlementRecords",
			path: fSettlementRecords,
			call: func(h *Exchange) error {
				_, err := h.FGetSettlementRecords(t.Context(), currency.BTC, 1, 20, startTime, startTime.Add(time.Hour))
				return err
			},
		},
		{
			name: "FContractTradingFee",
			path: fContractTradingFee,
			call: func(h *Exchange) error {
				_, err := h.FContractTradingFee(t.Context(), currency.BTC)
				return err
			},
		},
		{
			name: "FGetTransferLimits",
			path: fTransferLimitInfo,
			call: func(h *Exchange) error {
				_, err := h.FGetTransferLimits(t.Context(), currency.BTC)
				return err
			},
		},
		{
			name: "FGetPositionLimits",
			path: fPositionLimitInfo,
			call: func(h *Exchange) error {
				_, err := h.FGetPositionLimits(t.Context(), currency.BTC)
				return err
			},
		},
		{
			name: "FGetAssetsAndPositions",
			path: fQueryAssetsAndPositions,
			call: func(h *Exchange) error {
				_, err := h.FGetAssetsAndPositions(t.Context(), currency.BTC)
				return err
			},
		},
		{
			name: "FTransfer",
			path: fTransfer,
			call: func(h *Exchange) error {
				_, err := h.FTransfer(t.Context(), "123", "BTC", "master_to_sub", 1)
				return err
			},
		},
		{
			name: "FGetTransferRecords",
			path: fTransferRecords,
			call: func(h *Exchange) error {
				_, err := h.FGetTransferRecords(t.Context(), "BTC", "master_to_sub", 2, 1, 20)
				return err
			},
		},
		{
			name: "FGetAvailableLeverage",
			path: fAvailableLeverage,
			call: func(h *Exchange) error {
				_, err := h.FGetAvailableLeverage(t.Context(), currency.BTC)
				return err
			},
		},
		{
			name: "FOrder",
			path: fOrder,
			call: func(h *Exchange) error {
				_, err := h.FOrder(t.Context(), currency.EMPTYPAIR, "BTC", "quarter", "123", "buy", "open", "limit", 1, 1, 1)
				return err
			},
		},
		{
			name: "FPlaceBatchOrder",
			path: fBatchOrder,
			call: func(h *Exchange) error {
				_, err := h.FPlaceBatchOrder(t.Context(), []fBatchOrderData{{
					Symbol:         "BTC",
					ContractType:   "quarter",
					Price:          1,
					Volume:         1,
					Direction:      "buy",
					Offset:         "open",
					LeverageRate:   1,
					OrderPriceType: "limit",
				}})
				return err
			},
		},
		{
			name: "FCancelOrder",
			path: fCancelOrder,
			call: func(h *Exchange) error {
				_, err := h.FCancelOrder(t.Context(), currency.BTC, "123", "")
				return err
			},
		},
		{
			name: "FCancelAllOrders",
			path: fCancelAllOrders,
			call: func(h *Exchange) error {
				_, err := h.FCancelAllOrders(t.Context(), currency.EMPTYPAIR, "BTC", "quarter")
				return err
			},
		},
		{
			name: "FFlashCloseOrder",
			path: fFlashCloseOrder,
			call: func(h *Exchange) error {
				_, err := h.FFlashCloseOrder(t.Context(), currency.EMPTYPAIR, "BTC", "quarter", "buy", "lightning", "", 1)
				return err
			},
		},
		{
			name: "FGetOrderInfo",
			path: fOrderInfo,
			call: func(h *Exchange) error {
				_, err := h.FGetOrderInfo(t.Context(), "BTC", "", "123")
				return err
			},
		},
		{
			name: "FOrderDetails",
			path: fOrderDetails,
			call: func(h *Exchange) error {
				_, err := h.FOrderDetails(t.Context(), "BTC", "123", "quotation", startTime, 1, 20)
				return err
			},
		},
		{
			name: "FGetOpenOrders",
			path: fQueryOpenOrders,
			call: func(h *Exchange) error {
				_, err := h.FGetOpenOrders(t.Context(), currency.BTC, 1, 20)
				return err
			},
		},
		{
			name: "FTradeHistory",
			path: fMatchResult,
			call: func(h *Exchange) error {
				_, err := h.FTradeHistory(t.Context(), currency.EMPTYPAIR, "BTC", "all", 2, 1, 20)
				return err
			},
		},
		{
			name: "FPlaceTriggerOrder",
			path: fTriggerOrder,
			call: func(h *Exchange) error {
				_, err := h.FPlaceTriggerOrder(t.Context(), currency.EMPTYPAIR, "BTC", "quarter", "greaterOrEqual", "limit", "buy", "open", 2, 1, 1, 1)
				return err
			},
		},
		{
			name: "FCancelTriggerOrder",
			path: fCancelTriggerOrder,
			call: func(h *Exchange) error {
				_, err := h.FCancelTriggerOrder(t.Context(), "BTC", "123")
				return err
			},
		},
		{
			name: "FCancelAllTriggerOrders",
			path: fCancelAllTriggerOrders,
			call: func(h *Exchange) error {
				_, err := h.FCancelAllTriggerOrders(t.Context(), currency.EMPTYPAIR, "BTC", "quarter")
				return err
			},
		},
		{
			name: "FQueryTriggerOpenOrders",
			path: fTriggerOpenOrders,
			call: func(h *Exchange) error {
				_, err := h.FQueryTriggerOpenOrders(t.Context(), currency.EMPTYPAIR, "BTC", 1, 20)
				return err
			},
		},
		{
			name: "FQueryTriggerOrderHistory",
			path: fTriggerOrderHistory,
			call: func(h *Exchange) error {
				_, err := h.FQueryTriggerOrderHistory(t.Context(), currency.EMPTYPAIR, "BTC", "all", "all", 2, 1, 20)
				return err
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method, "authenticated futures method should match")
				assert.Equal(t, tc.path, r.URL.Path, "authenticated futures path should match HTX documentation")
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"), "authenticated futures content type should match")
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","code":200,"msg":"","data":null}`))
			}))
			t.Cleanup(server.Close)

			h := new(Exchange)
			require.NoError(t, testexch.Setup(h), "HTX setup must not error")
			h.API.AuthenticatedSupport = true
			h.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
			require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestFutures.String(), server.URL), "futures endpoint must be set")
			require.NoError(t, tc.call(h), "authenticated futures endpoint must not error")
		})
	}
}
