package htx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

func TestSwapWebsocketPremiumKlineResponse(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		raw  string
	}{
		{name: "quoted", raw: `{"data":[{"id":1603296000,"open":"-0.0001","close":"0.0002","low":"-0.0003","high":"0.0004","vol":"0","count":"0","amount":"0"}]}`},
		{name: "bare", raw: `{"data":[{"id":1603296000,"open":-0.0001,"close":0.0002,"low":-0.0003,"high":0.0004,"vol":0,"count":0,"amount":0}]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var response SwapWebsocketPremiumKlineResponse
			require.NoError(t, json.Unmarshal([]byte(tc.raw), &response), "premium index response must decode decimal values")
			require.Len(t, response.Data, 1, "response must retain its candle")
			assert.Equal(t, uint64(1603296000), response.Data[0].ID, "candle ID should decode")
			assert.Equal(t, -0.0001, response.Data[0].Open.Float64(), "negative premium should retain its sign")
			assert.Equal(t, 0.0002, response.Data[0].Close.Float64(), "premium close should decode")
			assert.Equal(t, -0.0003, response.Data[0].Low.Float64(), "premium low should decode")
			assert.Equal(t, 0.0004, response.Data[0].High.Float64(), "premium high should decode")
		})
	}
}

func TestSwapWebsocketBasisResponse(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		raw  string
	}{
		{name: "quoted", raw: `{"data":[{"id":1603296000,"contract_price":"12845.7","index_price":"12846.13","basis":"-0.43","basis_rate":"-0.00003"}]}`},
		{name: "bare", raw: `{"data":[{"id":1603296000,"contract_price":12845.7,"index_price":12846.13,"basis":-0.43,"basis_rate":-0.00003}]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var response SwapWebsocketBasisResponse
			require.NoError(t, json.Unmarshal([]byte(tc.raw), &response), "basis response must decode decimal values")
			require.Len(t, response.Data, 1, "response must retain its basis snapshot")
			assert.Equal(t, uint64(1603296000), response.Data[0].ID, "snapshot ID should decode")
			assert.Equal(t, 12845.7, response.Data[0].ContractPrice.Float64(), "contract price should decode")
			assert.Equal(t, 12846.13, response.Data[0].IndexPrice.Float64(), "index price should decode")
			assert.Equal(t, -0.43, response.Data[0].Basis.Float64(), "negative basis should retain its sign")
			assert.Equal(t, -0.00003, response.Data[0].BasisRate.Float64(), "negative basis rate should retain its sign")
		})
	}
}
