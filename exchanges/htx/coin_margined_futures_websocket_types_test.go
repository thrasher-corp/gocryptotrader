package htx

import (
	"testing"
	"time"

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

func TestSwapWebsocketTradesResponse(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, price, amount string }{
		{name: "quoted decimals", price: `"13689.8"`, amount: `"2"`},
		{name: "bare decimals", price: `13689.8`, amount: `2`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			raw := `{"id":"id8","rep":"market.BTC-USD.trade.detail","status":"ok","ts":1603876266723,"data":[{"id":9007199254740993,"price":` + tc.price + `,"amount":` + tc.amount + `,"direction":"sell","ts":1603876250774}]}`
			var response SwapWebsocketTradesResponse
			require.NoError(t, json.Unmarshal([]byte(raw), &response), "requested trade response must decode")
			assert.Equal(t, "id8", response.ID, "client correlation ID should retain its text")
			assert.Equal(t, "market.BTC-USD.trade.detail", response.Rep, "response topic should decode")
			assert.Equal(t, time.UnixMilli(1603876266723), response.Timestamp.Time(), "response time should retain milliseconds")
			require.Len(t, response.Data, 1, "response must retain its trade")
			trade := response.Data[0]
			assert.Equal(t, uint64(9007199254740993), trade.ID, "trade ID should retain all digits")
			assert.Equal(t, 13689.8, trade.Price.Float64(), "trade price should decode")
			assert.Equal(t, 2.0, trade.Amount.Float64(), "trade amount should decode")
			assert.Equal(t, "sell", trade.Direction, "taker direction should decode")
			assert.Equal(t, time.UnixMilli(1603876250774), trade.Timestamp.Time(), "trade time should retain milliseconds")
		})
	}
}

func TestSwapWebsocketKlineResponse(t *testing.T) {
	t.Parallel()
	var response SwapWebsocketKlineResponse
	require.NoError(t, json.Unmarshal([]byte(`{"id":"id4","rep":"market.BTC-USD.kline.15min","wsid":498385304,"status":"ok","data":[{"id":1599667200,"open":10262.31,"close":10244.93,"low":10234.84,"high":10282,"amount":1849.5,"vol":189634,"count":5342}]}`), &response), "requested candle envelope must decode")
	assert.Equal(t, "id4", response.ID, "client correlation ID should decode")
	assert.Equal(t, "market.BTC-USD.kline.15min", response.Rep, "response topic should decode")
	assert.Equal(t, uint64(498385304), response.WebsocketID, "connection ID should decode")
	require.Len(t, response.Data, 1, "data array must retain its candle")
	assert.Equal(t, SwapWebsocketCandle{ID: 1599667200, Open: 10262.31, Close: 10244.93, Low: 10234.84, High: 10282, Amount: 1849.5, Volume: 189634, Count: 5342}, response.Data[0], "candle values should decode from data")
}

func TestSwapWebsocketContract(t *testing.T) {
	t.Parallel()
	var response SwapWebsocketContractInfo
	require.NoError(t, json.Unmarshal([]byte(`{"op":"notify","topic":"public.BTC-USD.contract_info","ts":1603879734428,"event":"snapshot","data":[{"symbol":"BTC","contract_code":"BTC-USD","contract_size":100,"price_tick":0.1,"settlement_date":"1603900800000","create_date":"20200325","contract_status":1}]}`), &response), "contract notification must decode")
	require.Len(t, response.ContractData, 1, "notification must retain its contract")
	assert.Equal(t, time.UnixMilli(1603900800000), response.ContractData[0].SettlementDate.Time(), "settlement date should retain milliseconds")
	assert.Equal(t, time.Date(2020, time.March, 25, 0, 0, 0, 0, time.UTC), response.ContractData[0].CreateDate.Time(), "listing date should decode the documented calendar format")
}
