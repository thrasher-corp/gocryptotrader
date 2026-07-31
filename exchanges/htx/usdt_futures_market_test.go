package htx

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetV5AssetsDeductionCurrencies(t *testing.T) {
	t.Parallel()
	h := setupV5HTTPTest(t, http.MethodGet, "/v5/market/assets_deduction_currency", `{"code":200,"data":{"currency":["USDT","HTX"]}}`, nil)
	resp, err := h.GetV5AssetsDeductionCurrencies(t.Context())
	require.NoError(t, err, "GetV5AssetsDeductionCurrencies must not error")
	assert.Equal(t, []string{"USDT", "HTX"}, resp.Data.Currencies, "currencies should decode")

	errorExchange := setupV5HTTPTest(t, http.MethodGet, "/v5/market/assets_deduction_currency", `{"code":400,"message":"invalid request"}`, nil)
	_, err = errorExchange.GetV5AssetsDeductionCurrencies(t.Context())
	require.ErrorContains(t, err, "invalid request", "GetV5AssetsDeductionCurrencies must return V5 API errors")
}

func TestGetV5MultiAssetsMarginCurrencies(t *testing.T) {
	t.Parallel()
	h := setupV5HTTPTest(t, http.MethodGet, "/v5/market/multi_assets_margin", `{"code":200,"data":{"multi_assets":["USDT","BTC"]}}`, nil)
	resp, err := h.GetV5MultiAssetsMarginCurrencies(t.Context())
	require.NoError(t, err, "GetV5MultiAssetsMarginCurrencies must not error")
	assert.Equal(t, []string{"USDT", "BTC"}, resp.Data.Currencies, "currencies should decode")
}

func TestGetV5EliteAccountRatio(t *testing.T) {
	t.Parallel()
	h := setupV5HTTPTest(t, http.MethodGet, "/v5/market/elite_account_ratio", `{"code":200,"data":[{"contract_code":"BTC-USDT","buy_ratio":"0.6"}]}`, func(r *http.Request) {
		assert.Equal(t, "BTC-USDT", r.URL.Query().Get("contract_code"), "contract code should be sent")
		assert.Equal(t, "5min", r.URL.Query().Get("period"), "period should be sent")
	})
	resp, err := h.GetV5EliteAccountRatio(t.Context(), btcusdtPair, "5min")
	require.NoError(t, err, "GetV5EliteAccountRatio must not error")
	require.Len(t, resp.Data, 1, "one ratio must decode")
	assert.Equal(t, types.Number(0.6), resp.Data[0].BuyRatio, "buy ratio should decode")
}

func TestGetV5ElitePositionRatio(t *testing.T) {
	t.Parallel()
	h := setupV5HTTPTest(t, http.MethodGet, "/v5/market/elite_position_ratio", `{"code":200,"data":[{"contract_code":"BTC-USDT","sell_ratio":"0.4"}]}`, nil)
	resp, err := h.GetV5ElitePositionRatio(t.Context(), btcusdtPair, "5min")
	require.NoError(t, err, "GetV5ElitePositionRatio must not error")
	require.Len(t, resp.Data, 1, "one ratio must decode")
	assert.Equal(t, types.Number(0.4), resp.Data[0].SellRatio, "sell ratio should decode")
}

func TestGetV5EstimatedSettlementPrice(t *testing.T) {
	t.Parallel()
	h := setupV5HTTPTest(t, http.MethodGet, "/v5/market/estimated_settlement_price", `{"code":200,"data":[{"contract_code":"BTC-USDT","estimated_settlement_price":"10"}]}`, nil)
	resp, err := h.GetV5EstimatedSettlementPrice(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err, "GetV5EstimatedSettlementPrice must not error")
	require.Len(t, resp.Data, 1, "one estimate must decode")
	assert.Equal(t, types.Number(10), resp.Data[0].EstimatedSettlementPrice, "settlement price should decode")
}

func TestGetV5LiquidationOrders(t *testing.T) {
	t.Parallel()
	h := setupV5HTTPTest(t, http.MethodGet, "/v5/market/liquidation_orders", `{"code":200,"data":[{"id":"1","volume":"2"}]}`, func(r *http.Request) {
		assert.Equal(t, "BTC-USDT", r.URL.Query().Get("contract_code"), "contract code should be sent")
		assert.Equal(t, "10", r.URL.Query().Get("limit"), "limit should be sent")
	})
	_, err := h.GetV5LiquidationOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetV5LiquidationOrders must reject nil request")
	resp, err := h.GetV5LiquidationOrders(t.Context(), &V5LiquidationOrdersRequest{
		ContractCode: "BTC-USDT",
		Pair:         "BTC-USDT",
		StartTime:    time.UnixMilli(1),
		EndTime:      time.UnixMilli(2),
		Direction:    "next",
		From:         "1",
		Limit:        10,
	})
	require.NoError(t, err, "GetV5LiquidationOrders must not error")
	require.Len(t, resp.Data, 1, "one liquidation must decode")
	assert.Equal(t, types.Number(2), resp.Data[0].Volume, "volume should decode")
}

func TestGetV5MarketRiskLimit(t *testing.T) {
	t.Parallel()
	h := setupV5HTTPTest(t, http.MethodGet, "/v5/market/risk/limit", `{"code":200,"data":[{"contract_code":"BTC-USDT","max_lever":"20"}]}`, nil)
	resp, err := h.GetV5MarketRiskLimit(t.Context(), btcusdtPair, "cross", "1")
	require.NoError(t, err, "GetV5MarketRiskLimit must not error")
	require.Len(t, resp.Data, 1, "one risk limit must decode")
	assert.Equal(t, types.Number(20), resp.Data[0].MaximumLeverage, "maximum leverage should decode")
}

func TestGetV5SettlementHistory(t *testing.T) {
	t.Parallel()
	h := setupV5HTTPTest(t, http.MethodGet, "/v5/market/settlement_history", `{"code":200,"data":[{"id":"1","settlement_price":"10"}]}`, func(r *http.Request) {
		assert.Equal(t, "BTC-USDT", r.URL.Query().Get("contract_code"), "contract code should be sent")
	})
	_, err := h.GetV5SettlementHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetV5SettlementHistory must reject nil request")
	resp, err := h.GetV5SettlementHistory(t.Context(), &V5SettlementHistoryRequest{
		ContractCode: "BTC-USDT",
		StartTime:    time.UnixMilli(1),
		EndTime:      time.UnixMilli(2),
		Direction:    "next",
		From:         "1",
		Limit:        10,
	})
	require.NoError(t, err, "GetV5SettlementHistory must not error")
	require.Len(t, resp.Data, 1, "one settlement must decode")
	assert.Equal(t, types.Number(10), resp.Data[0].SettlementPrice, "settlement price should decode")
}
