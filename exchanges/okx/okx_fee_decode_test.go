package okx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

func TestTradeFeeRateUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, payload string
		present       bool
		invalid       bool
	}{
		{name: "explicit zero strings", payload: `{"maker":"0","taker":"0","makerU":"0","takerU":"0","makerUSDC":"0","takerUSDC":"0"}`, present: true},
		{name: "explicit zero numbers", payload: `{"maker":0,"taker":0,"makerU":0,"takerU":0,"makerUSDC":0,"takerUSDC":0}`, present: true},
		{name: "present charges", payload: `{"maker":"-0.001","taker":"-0.002","makerU":"-0.001","takerU":"-0.002","makerUSDC":"-0.001","takerUSDC":"-0.002"}`, present: true},
		{name: "empty", payload: `{"maker":"","taker":"","makerU":"","takerU":"","makerUSDC":"","takerUSDC":""}`},
		{name: "null", payload: `{"maker":null,"taker":null,"makerU":null,"takerU":null,"makerUSDC":null,"takerUSDC":null}`},
		{name: "missing", payload: `{}`},
		{name: "invalid", payload: `{"maker":"invalid"}`, invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			row := TradeFeeRate{MakerPresent: true, TakerPresent: true, MakerUSDTPresent: true, TakerUSDTPresent: true, MakerUSDCPresent: true, TakerUSDCPresent: true}
			err := json.Unmarshal([]byte(tc.payload), &row)
			if tc.invalid {
				require.Error(t, err, "invalid fees must fail decoding")
				return
			}
			require.NoError(t, err, "valid fees must decode")
			assert.Equal(t, tc.present, row.MakerPresent, "presence should match the response")
			assert.Equal(t, tc.present, row.TakerPresent, "presence should match the response")
			assert.Equal(t, tc.present, row.MakerUSDTPresent, "presence should match the response")
			assert.Equal(t, tc.present, row.TakerUSDTPresent, "presence should match the response")
			assert.Equal(t, tc.present, row.MakerUSDCPresent, "USDC maker presence should match")
			assert.Equal(t, tc.present, row.TakerUSDCPresent, "USDC taker presence should match")
		})
	}
	t.Run("modern fee group", func(t *testing.T) {
		t.Parallel()
		var row TradeFeeRate
		require.NoError(t, json.Unmarshal([]byte(`{"feeGroup":[{"groupId":"4","maker":"0","taker":"-0.0005"}]}`), &row), "fee group must decode")
		require.Len(t, row.FeeGroup, 1, "fee group must decode")
		assert.Equal(t, TradeFeeGroup{GroupID: "4", MakerRate: 0, TakerRate: -0.0005, MakerPresent: true, TakerPresent: true}, row.FeeGroup[0], "fee group should preserve rates and presence")
		var instrument Instrument
		require.NoError(t, json.Unmarshal([]byte(`{"instId":"BTC-USDT-SWAP","groupId":"4"}`), &instrument), "instrument must decode")
		assert.Equal(t, "4", instrument.TradingFeeGroupID, "instrument should retain its fee group")
	})
}

func TestTradeFeeGroupUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, payload    string
		present, invalid bool
	}{
		{name: "explicit zero strings", payload: `{"maker":"0","taker":"0"}`, present: true},
		{name: "explicit zero numbers", payload: `{"maker":0,"taker":0}`, present: true},
		{name: "charges and rebates", payload: `{"maker":"0.001","taker":"-0.002"}`, present: true},
		{name: "absent", payload: `{}`},
		{name: "null", payload: `{"maker":null,"taker":null}`},
		{name: "empty", payload: `{"maker":"","taker":""}`},
		{name: "invalid", payload: `{"maker":"invalid","taker":"0"}`, invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			row := TradeFeeGroup{MakerPresent: true, TakerPresent: true}
			err := json.Unmarshal([]byte(tc.payload), &row)
			if tc.invalid {
				require.Error(t, err, "invalid fee values must fail decoding")
				return
			}
			require.NoError(t, err, "valid fee responses must decode")
			assert.Equal(t, tc.present, row.MakerPresent, "maker presence should match the response")
			assert.Equal(t, tc.present, row.TakerPresent, "taker presence should match the response")
		})
	}
}
