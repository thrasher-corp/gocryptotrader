package htx

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestV5WebsocketRateLimitUnmarshalJSON(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name    string
		raw     string
		empty   bool
		wantErr bool
	}
	for _, tc := range []testCase{
		{name: "object", raw: `{"limit":"24","interval":"3000","remaining":"19","reset":"1772779491561"}`},
		{name: "encoded string", raw: `"{\"limit\":\"24\",\"interval\":\"3000\",\"remaining\":\"19\",\"reset\":\"1772779491561\"}"`},
		{name: "null", raw: `null`, empty: true},
		{name: "empty string", raw: `""`, wantErr: true},
		{name: "invalid string", raw: `"not JSON"`, wantErr: true},
		{name: "array", raw: `[]`, wantErr: true},
		{name: "malformed", raw: `{`, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			state := V5WebsocketRateLimit{Limit: 99}
			err := state.UnmarshalJSON([]byte(tc.raw))
			if tc.wantErr {
				require.Error(t, err, "invalid state must return a decoding error")
				assert.Equal(t, types.Number(99), state.Limit, "failed decode should preserve existing state")
				return
			}
			require.NoError(t, err, "rate limit must decode")
			if tc.empty {
				assert.Equal(t, V5WebsocketRateLimit{}, state, "null should clear state")
				return
			}
			assert.Equal(t, types.Number(24), state.Limit)
			assert.Equal(t, types.Number(3000), state.Interval)
			assert.Equal(t, types.Number(19), state.Remaining)
			assert.Equal(t, time.UnixMilli(1772779491561), state.Reset.Time())
			var response V5WebsocketOrderResponse
			require.NoError(t, json.Unmarshal([]byte(`{"rate_limit":`+tc.raw+`}`), &response), "embedded trade response must decode")
			assert.Equal(t, state, response.RateLimit, "embedded rate limit should match standalone state")
		})
	}
}

func TestV5WebsocketOrder(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name, raw string
		want      time.Time
	}
	for _, tc := range []testCase{
		{name: "milliseconds", raw: `1604312615051`, want: time.UnixMilli(1604312615051)},
		{name: "quoted milliseconds", raw: `"1604312615051"`, want: time.UnixMilli(1604312615051)},
		{name: "zero", raw: `0`},
		{name: "empty", raw: `""`},
		{name: "null", raw: `null`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var response V5WebsocketOrder
			require.NoError(t, json.Unmarshal([]byte(`{"created_time":`+tc.raw+`,"updated_time":`+tc.raw+`}`), &response), "websocket timestamps must decode")
			assert.Equal(t, tc.want, response.CreatedTime.Time(), "CreatedTime should preserve the timestamp or remain unset")
			assert.Equal(t, tc.want, response.UpdatedTime.Time(), "UpdatedTime should preserve the timestamp or remain unset")
		})
	}
}

func TestV5WebsocketMatchOrder(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name, raw string
		want      time.Time
	}
	for _, tc := range []testCase{
		{name: "milliseconds", raw: `1604312615051`, want: time.UnixMilli(1604312615051)},
		{name: "quoted milliseconds", raw: `"1604312615051"`, want: time.UnixMilli(1604312615051)},
		{name: "zero", raw: `0`},
		{name: "empty", raw: `""`},
		{name: "null", raw: `null`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var response V5WebsocketMatchOrder
			require.NoError(t, json.Unmarshal([]byte(`{"match_time":`+tc.raw+`}`), &response), "websocket timestamps must decode")
			assert.Equal(t, tc.want, response.MatchTime.Time(), "MatchTime should preserve the timestamp or remain unset")
		})
	}
}

func TestV5WebsocketPositionUpdate(t *testing.T) {
	t.Parallel()
	var response V5WebsocketPositionUpdate
	err := json.Unmarshal([]byte(`{"op":"notify","topic":"positions","contract_code":"DOGE-USDT","ts":1783421319399,"uid":"562637220","event":"snapshot","data":[{"margin":"7.4806","contract_code":"DOGE-USDT","symbol":"DOGE","position_mode":"dual_side","position_side":"long","direction":"buy","margin_mode":"cross","open_avg_price":"0.074821","volume":"1","available":"1","fee":"0.00448926","break_even_price":"0.0749107852","total_trade_fee":"0.00448926","lever_rate":1,"adl_risk_percent":5,"liquidation_price":"-0.071400533059835393","initial_margin":"7.4806","maintenance_margin":"0.04937196","profit_unreal":"-0.0015","profit":"0","profit_rate":"-0.0002","margin_rate":"0.0033","state":"normal","funding_fee":"0","mark_price":"0.074806","last_price":"0.074807","contract_type":"swap","version":11,"created_time":"1783421181489","updated_time":"1783421181489"}]}`), &response)
	require.NoError(t, err, "documented position notification must decode")
	require.Len(t, response.Positions, 1, "position notification must contain a concrete position")
	position := response.Positions[0]
	assert.Equal(t, "DOGE-USDT", position.ContractCode, "position contract should decode")
	assert.Equal(t, "snapshot", response.Event, "position event should decode")
	assert.Equal(t, types.Number(1), position.LeverageRate, "numeric leverage should decode")
	assert.Equal(t, uint64(5), position.ADLRiskPercent, "risk level should decode")
	assert.Equal(t, types.Number(-0.071400533059835393), position.LiquidationPrice, "liquidation price should retain its sign")
	assert.Equal(t, types.Number(0.00448926), position.TotalTradeFee, "trade fee should decode")
	assert.Equal(t, types.Number(0.0749107852), position.BreakEvenPrice, "break-even price should decode")
	assert.Equal(t, time.UnixMilli(1783421181489), position.CreatedTime.Time(), "quoted timestamp should retain milliseconds")
	t.Run("quoted leverage", func(t *testing.T) {
		t.Parallel()
		var quoted V5WebsocketPositionUpdate
		require.NoError(t, json.Unmarshal([]byte(`{"data":[{"lever_rate":"5"}]}`), &quoted), "documented string leverage must decode")
		require.Len(t, quoted.Positions, 1, "notification must retain its position")
		assert.Equal(t, types.Number(5), quoted.Positions[0].LeverageRate, "quoted leverage should decode")
	})
}

func TestV5WebsocketAlgoOrderUpdate(t *testing.T) {
	t.Parallel()
	var response V5WebsocketAlgoOrderUpdate
	err := json.Unmarshal([]byte(`{"op":"notify","topic":"algo_orders","contract_code":"ETH-USDT","ts":1767600751568,"uid":"413618269","data":[{"id":"","volume":"1","state":"active","price":"","type":"trailing_stop","algo_id":"1457768739645739008","algo_client_order_id":"","contract_code":"ETH-USDT","position_side":"long","margin_mode":"isolated","side":"sell","created_time":"1767600751565","updated_time":"","order_source":"api","relation_order_id":"","actual_volume":"1","active_price":"100500","order_price_type":"optimal_5","callback_rate":"0.05","reduce_only":true,"actual_price":"","actual_time":""}]}`), &response)
	require.NoError(t, err, "documented algoorder notification must decode")
	require.Len(t, response.Orders, 1, "algo notification must contain a concrete order")
	algo := response.Orders[0]
	assert.Equal(t, "1457768739645739008", algo.AlgoID, "algo identifier should retain its precision")
	assert.Equal(t, "ETH-USDT", algo.ContractCode, "algo contract should decode")
	assert.Equal(t, "trailing_stop", algo.Type, "algo type should decode")
	assert.Equal(t, types.Number(100500), algo.ActivationPrice, "activation price should decode")
	assert.Equal(t, types.Number(0.05), algo.CallbackRate, "callback rate should decode")
	assert.True(t, algo.ReduceOnly.Bool(), "reduce-only flag should decode")
	assert.Equal(t, time.UnixMilli(1767600751565), algo.CreatedTime.Time(), "quoted timestamp should retain milliseconds")
	assert.True(t, algo.UpdatedTime.Time().IsZero(), "empty timestamp should remain unset")
}
