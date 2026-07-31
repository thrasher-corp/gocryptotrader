package htx

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetV5Leverage(t *testing.T) {
	t.Parallel()
	h := setupV5HTTPTest(t, http.MethodGet, "/v5/position/lever", `{"code":200,"data":[{"contract_code":"BTC-USDT","lever_rate":5,"available_lever":[1,5,10]}]}`, func(r *http.Request) {
		assert.Equal(t, "cross", r.URL.Query().Get("margin_mode"), "margin mode should be sent")
	})
	resp, err := h.GetV5Leverage(t.Context(), btcusdtPair, "cross", "long")
	require.NoError(t, err, "GetV5Leverage must not error")
	require.Len(t, resp.Data, 1, "one leverage entry must decode")
	assert.Equal(t, []uint64{1, 5, 10}, resp.Data[0].AvailableLeverage, "available leverage should decode")
}

func TestSetV5Leverage(t *testing.T) {
	t.Parallel()
	h := setupV5HTTPTest(t, http.MethodPost, "/v5/position/lever", `{"code":200,"data":{"contract_code":"BTC-USDT","lever_rate":"5"}}`, nil)
	_, err := h.SetV5Leverage(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SetV5Leverage must reject nil request")
	resp, err := h.SetV5Leverage(t.Context(), &V5SetLeverageRequest{ContractCode: "BTC-USDT", MarginMode: "cross", LeverageRate: 5})
	require.NoError(t, err, "SetV5Leverage must not error")
	assert.Equal(t, types.Number(5), resp.Data.LeverageRate, "leverage should decode")
}

func TestAdjustV5PositionMargin(t *testing.T) {
	t.Parallel()
	h := setupV5HTTPTest(t, http.MethodPost, "/v5/position/margin", `{"code":200,"message":"Success","data":null}`, nil)
	_, err := h.AdjustV5PositionMargin(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "AdjustV5PositionMargin must reject nil request")
	resp, err := h.AdjustV5PositionMargin(t.Context(), &V5AdjustPositionMarginRequest{ContractCode: "BTC-USDT", PositionSide: "long", Type: "add", Amount: 10})
	require.NoError(t, err, "AdjustV5PositionMargin must not error")
	assert.Equal(t, int64(200), resp.Code, "response code should decode")
}

func TestGetV5PositionMode(t *testing.T) {
	t.Parallel()
	h := setupV5HTTPTest(t, http.MethodGet, "/v5/position/mode", `{"code":200,"data":{"position_mode":"dual_side"}}`, nil)
	resp, err := h.GetV5PositionMode(t.Context())
	require.NoError(t, err, "GetV5PositionMode must not error")
	assert.Equal(t, "dual_side", resp.Data.PositionMode, "position mode should decode")
}

func TestSetV5PositionMode(t *testing.T) {
	t.Parallel()
	h := setupV5HTTPTest(t, http.MethodPost, "/v5/position/mode", `{"code":200,"data":{"position_mode":"single_side"}}`, nil)
	resp, err := h.SetV5PositionMode(t.Context(), "single_side")
	require.NoError(t, err, "SetV5PositionMode must not error")
	assert.Equal(t, "single_side", resp.Data.PositionMode, "position mode should decode")
}

func TestGetV5PositionRiskLimit(t *testing.T) {
	t.Parallel()
	h := setupV5HTTPTest(t, http.MethodGet, "/v5/position/risk/limit", `{"code":200,"data":[{"contract_code":"BTC-USDT","max_volume":"100"}]}`, nil)
	resp, err := h.GetV5PositionRiskLimit(t.Context(), currency.EMPTYPAIR, "cross", "long")
	require.NoError(t, err, "GetV5PositionRiskLimit must not error")
	require.Len(t, resp.Data, 1, "one risk limit must decode")
	assert.Equal(t, types.Number(100), resp.Data[0].MaximumVolume, "maximum volume should decode")
}

func TestGetV5PositionRiskLimitTiers(t *testing.T) {
	t.Parallel()
	h := setupV5HTTPTest(t, http.MethodGet, "/v5/position/risk/limit_tier", `{"code":200,"data":[{"contract_code":"BTC-USDT","tier":"1"}]}`, nil)
	resp, err := h.GetV5PositionRiskLimitTiers(t.Context(), btcusdtPair, "cross")
	require.NoError(t, err, "GetV5PositionRiskLimitTiers must not error")
	require.Len(t, resp.Data, 1, "one risk tier must decode")
	assert.Equal(t, types.Number(1), resp.Data[0].Tier, "risk tier should decode")
}
