package htx

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

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
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetV5Leverage(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodGet, "/v5/position/lever", `{"code":200,"data":[{"contract_code":"BTC-USDT","lever_rate":5,"available_lever":[1,5,10]}]}`, func(r *http.Request) {
		assert.Equal(t, "cross", r.URL.Query().Get("margin_mode"), "margin mode should be sent")
	})
	resp, err := h.GetV5Leverage(t.Context(), btcusdtPair, "cross", "long")
	require.NoError(t, err, "GetV5Leverage must not error")
	require.Len(t, resp.Leverage, 1, "one leverage entry must decode")
	assert.Equal(t, []uint64{1, 5, 10}, resp.Leverage[0].AvailableLeverage, "available leverage should decode")
}

func TestSetV5Leverage(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodPost, "/v5/position/lever", `{"code":200,"data":{"contract_code":"BTC-USDT","lever_rate":"5"}}`, nil)
	_, err := h.SetV5Leverage(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SetV5Leverage must reject nil request")
	resp, err := h.SetV5Leverage(t.Context(), &V5SetLeverageRequest{ContractCode: "BTC-USDT", MarginMode: "cross", LeverageRate: 5})
	require.NoError(t, err, "SetV5Leverage must not error")
	assert.Equal(t, types.Number(5), resp.Leverage.LeverageRate, "leverage should decode")
}

func TestAdjustV5PositionMargin(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodPost, "/v5/position/margin", `{"code":200,"message":"Success","data":null}`, nil)
	_, err := h.AdjustV5PositionMargin(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "AdjustV5PositionMargin must reject nil request")
	resp, err := h.AdjustV5PositionMargin(t.Context(), &V5AdjustPositionMarginRequest{ContractCode: "BTC-USDT", PositionSide: "long", Type: "add", Amount: 10})
	require.NoError(t, err, "AdjustV5PositionMargin must not error")
	assert.Equal(t, int64(200), resp.Code, "response code should decode")
}

func TestGetV5PositionMode(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodGet, "/v5/position/mode", `{"code":200,"data":{"position_mode":"dual_side"}}`, nil)
	resp, err := h.GetV5PositionMode(t.Context())
	require.NoError(t, err, "GetV5PositionMode must not error")
	assert.Equal(t, "dual_side", resp.Mode.PositionMode, "position mode should decode")
}

func TestSetV5PositionMode(t *testing.T) {
	t.Parallel()
	t.Run("exclude overlapping submissions and mode changes", func(t *testing.T) {
		t.Parallel()
		started, release := make(chan struct{}), make(chan struct{})
		var calls atomic.Int32
		h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodPost, "/v5/position/mode", `{"code":200,"data":{"position_mode":"single_side"}}`, func(*http.Request) {
			if calls.Add(1) == 1 {
				close(started)
				<-release
			}
		})
		// Registered after the server cleanup so a failed assertion cannot strand its handler.
		var releaseOnce sync.Once
		t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })
		h.Websocket.SetCanUseAuthenticatedEndpoints(true)
		result := make(chan error, 1)
		go func() { _, err := h.SetV5PositionMode(t.Context(), "single_side"); result <- err }()
		select {
		case <-started:
		case <-t.Context().Done():
			require.FailNow(t, "mode request must start")
		}
		submission := &order.Submit{Exchange: h.Name, Pair: btcusdtPair, AssetType: asset.USDTMarginedFutures, Side: order.Buy, Type: order.Limit, Price: 100, Amount: 1}
		_, err := h.SubmitOrder(t.Context(), submission)
		require.ErrorIs(t, err, errPositionModeInUse, "REST submission must not race a mode change")
		_, err = h.WebsocketSubmitOrder(t.Context(), submission)
		require.ErrorIs(t, err, errPositionModeInUse, "websocket submission must not race a mode change")
		_, err = h.WebsocketSubmitOrders(t.Context(), []*order.Submit{submission})
		require.ErrorIs(t, err, errPositionModeInUse, "batch submission must not race a mode change")
		_, err = h.SetV5PositionMode(t.Context(), "dual_side")
		require.ErrorIs(t, err, errPositionModeInUse, "mode changes must exclude each other")
		releaseOnce.Do(func() { close(release) })
		require.NoError(t, <-result, "initial mode change must complete")
		assert.False(t, h.positionModeInUse.Load(), "successful mode change should release the guard")
		_, err = h.SetV5PositionMode(t.Context(), "dual_side")
		require.NoError(t, err, "later mode changes must remain usable")
	})
	t.Run("release after failure", func(t *testing.T) {
		t.Parallel()
		h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodPost, "/v5/position/mode", `{"code":400,"message":"rejected"}`, nil)
		_, err := h.SetV5PositionMode(t.Context(), "dual_side")
		require.Error(t, err, "rejected mode change must fail")
		assert.False(t, h.positionModeInUse.Load(), "failed mode change should release the guard")
	})

	h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodPost, "/v5/position/mode", `{"code":200,"data":{"position_mode":"single_side"}}`, nil)
	resp, err := h.SetV5PositionMode(t.Context(), "single_side")
	require.NoError(t, err, "SetV5PositionMode must not error")
	assert.Equal(t, "single_side", resp.Mode.PositionMode, "position mode should decode")
}

func TestGetV5PositionRiskLimit(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodGet, "/v5/position/risk/limit", `{"code":200,"data":[{"contract_code":"BTC-USDT","max_volume":"100"}]}`, nil)
	resp, err := h.GetV5PositionRiskLimit(t.Context(), currency.EMPTYPAIR, "cross", "long")
	require.NoError(t, err, "GetV5PositionRiskLimit must not error")
	require.Len(t, resp.RiskLimits, 1, "one risk limit must decode")
	assert.Equal(t, types.Number(100), resp.RiskLimits[0].MaximumVolume, "maximum volume should decode")
}

func TestGetV5PositionRiskLimitTiers(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodGet, "/v5/position/risk/limit_tier", `{"code":200,"data":[{"contract_code":"BTC-USDT","tier":"1"}]}`, nil)
	resp, err := h.GetV5PositionRiskLimitTiers(t.Context(), btcusdtPair, "cross")
	require.NoError(t, err, "GetV5PositionRiskLimitTiers must not error")
	require.Len(t, resp.RiskLimits, 1, "one risk tier must decode")
	assert.Equal(t, types.Number(1), resp.RiskLimits[0].Tier, "risk tier should decode")
}

func TestSwitchLinearSwapLeverage(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name         string
		mode         string
		crossMargin  bool
		side         order.Side
		wantSide     string
		wantErr      error
		modeResponse string
	}
	for _, tc := range []testCase{
		{name: "one-way isolated buy", mode: "single_side", side: order.Buy, wantSide: "both"},
		{name: "one-way isolated sell", mode: "single_side", side: order.Sell, wantSide: "both"},
		{name: "one-way cross buy", mode: "single_side", crossMargin: true, side: order.Buy, wantSide: "both"},
		{name: "one-way cross sell", mode: "single_side", crossMargin: true, side: order.Sell, wantSide: "both"},
		{name: "one-way unspecified isolated side", mode: "single_side", wantSide: "both"},
		{name: "one-way unspecified cross side", mode: "single_side", crossMargin: true, wantSide: "both"},
		{name: "hedge isolated buy", mode: "dual_side", side: order.Buy, wantSide: "long"},
		{name: "hedge isolated sell", mode: "dual_side", side: order.Sell, wantSide: "short"},
		{name: "hedge cross buy", mode: "dual_side", crossMargin: true, side: order.Buy, wantSide: "long"},
		{name: "hedge cross sell", mode: "dual_side", crossMargin: true, side: order.Sell, wantSide: "short"},
		{name: "hedge long", mode: "dual_side", side: order.Long, wantSide: "long"},
		{name: "hedge short", mode: "dual_side", side: order.Short, wantSide: "short"},
		{name: "hedge unspecified cross side", mode: "dual_side", crossMargin: true},
		{name: "hedge unspecified isolated side", mode: "dual_side", wantErr: order.ErrSideIsInvalid},
		{name: "invalid side", mode: "single_side", side: order.AnySide, wantErr: order.ErrSideIsInvalid},
		{name: "mode request failure", modeResponse: `{"code":400,"message":"mode unavailable"}`, wantErr: request.ErrAuthRequestFailed},
		{name: "null mode response", modeResponse: `null`, wantErr: errEmptyResult},
		{name: "unsupported mode", mode: "unsupported", wantErr: errInvalidPositionMode},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/v5/position/mode" {
					assert.Equal(t, http.MethodGet, r.Method, "position mode should be queried")
					response := tc.modeResponse
					if response == "" {
						response = `{"code":200,"data":{"position_mode":"` + tc.mode + `"}}`
					}
					_, _ = w.Write([]byte(response))
					return
				}
				calls.Add(1)
				assert.Equal(t, "/v5/position/lever", r.URL.Path, "setter endpoint should match")
				assert.Equal(t, http.MethodPost, r.Method, "leverage should be set with POST")
				var payload V5SetLeverageRequest
				if assert.NoError(t, json.NewDecoder(r.Body).Decode(&payload), "leverage request should decode") {
					marginMode := "isolated"
					if tc.crossMargin {
						marginMode = "cross"
					}
					assert.Equal(t, V5SetLeverageRequest{ContractCode: "BTC-USDT", MarginMode: marginMode, PositionSide: tc.wantSide, LeverageRate: 5}, payload, "leverage request should target the account's position mode")
				}
				_, _ = w.Write([]byte(`{"code":200,"data":{}}`))
			}))
			t.Cleanup(server.Close)
			h := new(Exchange)
			require.NoError(t, testexch.Setup(h), "HTX setup must succeed")
			h.API.AuthenticatedSupport = true
			h.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
			require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestUSDTMargined.String(), server.URL), "mock endpoint must be set")
			err := h.SwitchLinearSwapLeverage(t.Context(), btcusdtPair, 5, tc.crossMargin, tc.side)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr, "leverage change must return the expected error")
				assert.Zero(t, calls.Load(), "failed validation or mode lookup should prevent a leverage change")
				return
			}
			require.NoError(t, err, "leverage change must succeed")
			assert.Equal(t, int32(1), calls.Load(), "exactly one leverage change should be sent")
		})
	}
}
