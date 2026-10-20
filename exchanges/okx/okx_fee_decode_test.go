package okx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
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
			previous := types.Number(1)
			row := TradeFeeRate{FeeRateMaker: &previous, FeeRateTaker: &previous, FeeRateMakerUSDT: &previous, FeeRateTakerUSDT: &previous, FeeRateMakerUSDC: &previous, FeeRateTakerUSDC: &previous}
			err := json.Unmarshal([]byte(tc.payload), &row)
			if tc.invalid {
				require.Error(t, err, "invalid fees must fail decoding")
				return
			}
			require.NoError(t, err, "valid fees must decode")
			assert.Equal(t, tc.present, row.FeeRateMaker != nil, "presence should match the response")
			assert.Equal(t, tc.present, row.FeeRateTaker != nil, "presence should match the response")
			assert.Equal(t, tc.present, row.FeeRateMakerUSDT != nil, "presence should match the response")
			assert.Equal(t, tc.present, row.FeeRateTakerUSDT != nil, "presence should match the response")
			assert.Equal(t, tc.present, row.FeeRateMakerUSDC != nil, "USDC maker presence should match")
			assert.Equal(t, tc.present, row.FeeRateTakerUSDC != nil, "USDC taker presence should match")
		})
	}
	t.Run("independent legacy fields", func(t *testing.T) {
		t.Parallel()
		var row TradeFeeRate
		require.NoError(t, json.Unmarshal([]byte(`{"maker":"-0.001","taker":null,"makerU":"","takerU":"-0.002","makerUSDC":"0.003","takerUSDC":"0"}`), &row), "mixed legacy fields must decode")
		maker, takerUSDT, makerUSDC, takerUSDC := types.Number(-0.001), types.Number(-0.002), types.Number(0.003), types.Number(0)
		assert.Equal(t, TradeFeeRate{
			FeeRateMaker: &maker, FeeRateTakerUSDT: &takerUSDT, FeeRateMakerUSDC: &makerUSDC, FeeRateTakerUSDC: &takerUSDC,
		}, row, "each legacy rate and presence flag should follow its own field")
	})
	t.Run("modern fee group", func(t *testing.T) {
		t.Parallel()
		var row TradeFeeRate
		require.NoError(t, json.Unmarshal([]byte(`{"feeGroup":[{"groupId":"4","maker":"0","taker":"-0.0005"}]}`), &row), "fee group must decode")
		require.Len(t, row.FeeGroup, 1, "fee group must decode")
		maker, taker := types.Number(0), types.Number(-0.0005)
		assert.Equal(t, TradeFeeGroup{GroupID: "4", MakerRate: &maker, TakerRate: &taker}, row.FeeGroup[0], "fee group should preserve rates and presence")
		var instrument Instrument
		require.NoError(t, json.Unmarshal([]byte(`{"instId":"BTC-USDT-SWAP","groupId":"4"}`), &instrument), "instrument must decode")
		assert.Equal(t, "4", instrument.TradingFeeGroupID, "instrument should retain its fee group")
	})
}

func TestTradeFeeGroupUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, payload                       string
		makerPresent, takerPresent, invalid bool
		maker, taker                        float64
	}{
		{name: "explicit zero strings", payload: `{"maker":"0","taker":"0"}`, makerPresent: true, takerPresent: true},
		{name: "explicit zero numbers", payload: `{"maker":0,"taker":0}`, makerPresent: true, takerPresent: true},
		{name: "charges and rebates", payload: `{"maker":"0.001","taker":"-0.002"}`, makerPresent: true, takerPresent: true, maker: 0.001, taker: -0.002},
		{name: "maker only", payload: `{"maker":"0","taker":""}`, makerPresent: true},
		{name: "taker only", payload: `{"maker":null,"taker":"-0.002"}`, takerPresent: true, taker: -0.002},
		{name: "absent", payload: `{}`},
		{name: "null", payload: `{"maker":null,"taker":null}`},
		{name: "empty", payload: `{"maker":"","taker":""}`},
		{name: "invalid", payload: `{"maker":"invalid","taker":"0"}`, invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			previous := types.Number(1)
			row := TradeFeeGroup{MakerRate: &previous, TakerRate: &previous}
			err := json.Unmarshal([]byte(tc.payload), &row)
			if tc.invalid {
				require.Error(t, err, "invalid fee values must fail decoding")
				return
			}
			require.NoError(t, err, "valid fee responses must decode")
			assert.Equal(t, tc.makerPresent, row.MakerRate != nil, "maker presence should match the response")
			if tc.makerPresent {
				require.NotNil(t, row.MakerRate, "maker rate must be available")
				assert.Equal(t, tc.maker, row.MakerRate.Float64(), "maker rate should match the response")
			}
			assert.Equal(t, tc.takerPresent, row.TakerRate != nil, "taker presence should match the response")
			if tc.takerPresent {
				require.NotNil(t, row.TakerRate, "taker rate must be available")
				assert.Equal(t, tc.taker, row.TakerRate.Float64(), "taker rate should match the response")
			}
		})
	}
}
