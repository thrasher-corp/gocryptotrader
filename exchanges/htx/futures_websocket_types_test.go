package htx

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

func TestFuturesWebsocketKlineResponse(t *testing.T) {
	t.Parallel()
	var response FuturesWebsocketKlineResponse
	require.NoError(t, json.Unmarshal([]byte(`{"id":"id4","rep":"market.BTC_CQ.kline.15min","wsid":498385304,"status":"ok","data":[{"id":1599667200,"open":10262.31,"close":10244.93,"low":10234.84,"high":10282,"amount":1849.5,"vol":189634,"count":5342}]}`), &response), "requested candle envelope must decode")
	assert.Equal(t, "id4", response.ID, "client correlation ID should decode")
	assert.Equal(t, "market.BTC_CQ.kline.15min", response.Rep, "response topic should decode")
	assert.Equal(t, uint64(498385304), response.WebsocketID, "connection ID should decode")
	require.Len(t, response.Data, 1, "data array must retain its candle")
	assert.Equal(t, FuturesWebsocketKline{ID: 1599667200, Open: 10262.31, Close: 10244.93, Low: 10234.84, High: 10282, Amount: 1849.5, Volume: 189634, Count: 5342}, response.Data[0], "candle values should decode from data")
}

func TestFuturesWebsocketTradeDetailResponse(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, price, amount string }{
		{name: "quoted decimals", price: `"13586.25"`, amount: `"4"`},
		{name: "bare decimals", price: `13586.25`, amount: `4`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			raw := `{"id":"id8","rep":"market.BTC_CQ.trade.detail","status":"ok","ts":1604386202755,"data":[{"id":9007199254740993,"price":` + tc.price + `,"amount":` + tc.amount + `,"direction":"buy","ts":1604386167285}]}`
			var response FuturesWebsocketTradeDetailResponse
			require.NoError(t, json.Unmarshal([]byte(raw), &response), "requested delivery trade envelope must decode")
			assert.Equal(t, "id8", response.ID, "client correlation ID should retain its text")
			assert.Equal(t, time.UnixMilli(1604386202755), response.Timestamp.Time(), "response time should retain milliseconds")
			require.Len(t, response.Data, 1, "response must retain its execution")
			assert.Equal(t, uint64(9007199254740993), response.Data[0].ID, "trade ID should retain all digits")
			assert.Equal(t, 13586.25, response.Data[0].Price.Float64(), "trade price should decode")
			assert.Equal(t, 4.0, response.Data[0].Amount.Float64(), "trade amount should decode")
			assert.Equal(t, time.UnixMilli(1604386167285), response.Data[0].Timestamp.Time(), "trade time should retain milliseconds")
		})
	}
}

func TestFuturesWebsocketContractInfo(t *testing.T) {
	t.Parallel()
	var response FuturesWebsocketSubContractInfo
	require.NoError(t, json.Unmarshal([]byte(`{"op":"notify","topic":"public.BTC.contract_info","ts":1603879734428,"event":"snapshot","data":[{"symbol":"BTC","contract_code":"BTC201225","contract_type":"quarter","contract_size":100,"price_tick":0.1,"delivery_date":"20201225","create_date":"20200925","contract_status":1}]}`), &response), "delivery contract notification must decode")
	require.Len(t, response.ContractData, 1, "notification must retain its contract")
	assert.Equal(t, time.Date(2020, time.September, 25, 0, 0, 0, 0, time.UTC), response.ContractData[0].CreateDate.Time(), "listing date should decode its calendar representation")
	assert.Equal(t, time.Date(2020, time.December, 25, 0, 0, 0, 0, time.UTC), response.ContractData[0].DeliveryDate.Time(), "delivery date should decode its calendar representation")
}
