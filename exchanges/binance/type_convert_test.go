package binance

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

// These UnmarshalJSON helpers all accept either a single JSON object or an array
// of objects from the exchange, so each test exercises both the array branch and
// the fallback single-object branch, plus the invalid-input error path.

func TestPriceChanges_UnmarshalJSON(t *testing.T) {
	t.Parallel()
	var arr PriceChanges
	require.NoError(t, json.Unmarshal([]byte(`[{"symbol":"BTCUSDT","lastPrice":"100.5"},{"symbol":"ETHUSDT","lastPrice":"50.25"}]`), &arr))
	require.Len(t, arr, 2, "array input must unmarshal to two elements")
	assert.Equal(t, "BTCUSDT", arr[0].Symbol, "first symbol should match")
	assert.Equal(t, 50.25, arr[1].LastPrice.Float64(), "second last price should match")

	var single PriceChanges
	require.NoError(t, json.Unmarshal([]byte(`{"symbol":"BTCUSDT","lastPrice":"100.5"}`), &single))
	require.Len(t, single, 1, "single object input must unmarshal to one element")
	assert.Equal(t, "BTCUSDT", single[0].Symbol, "symbol should match")

	var bad PriceChanges
	assert.Error(t, json.Unmarshal([]byte(`"not an object"`), &bad), "invalid input should error")
}

func TestSymbolTickers_UnmarshalJSON(t *testing.T) {
	t.Parallel()
	var arr SymbolTickers
	require.NoError(t, json.Unmarshal([]byte(`[{"symbol":"BTCUSDT","price":"100.5"},{"symbol":"ETHUSDT","price":"50.25"}]`), &arr))
	require.Len(t, arr, 2, "array input must unmarshal to two elements")
	assert.Equal(t, 100.5, arr[0].Price.Float64(), "first price should match")

	var single SymbolTickers
	require.NoError(t, json.Unmarshal([]byte(`{"symbol":"BTCUSDT","price":"100.5"}`), &single))
	require.Len(t, single, 1, "single object input must unmarshal to one element")
	assert.Equal(t, "BTCUSDT", single[0].Symbol, "symbol should match")

	var bad SymbolTickers
	assert.Error(t, json.Unmarshal([]byte(`12345`), &bad), "invalid input should error")
}

func TestWsOrderbookTickers_UnmarshalJSON(t *testing.T) {
	t.Parallel()
	var arr WsOrderbookTickers
	require.NoError(t, json.Unmarshal([]byte(`[{"symbol":"BTCUSDT","bidPrice":"99","askPrice":"101"},{"symbol":"ETHUSDT","bidPrice":"49","askPrice":"51"}]`), &arr))
	require.Len(t, arr, 2, "array input must unmarshal to two elements")
	assert.Equal(t, 101.0, arr[0].AskPrice.Float64(), "first ask price should match")

	var single WsOrderbookTickers
	require.NoError(t, json.Unmarshal([]byte(`{"symbol":"BTCUSDT","bidPrice":"99","askPrice":"101"}`), &single))
	require.Len(t, single, 1, "single object input must unmarshal to one element")
	assert.Equal(t, 99.0, single[0].BidPrice.Float64(), "bid price should match")

	var bad WsOrderbookTickers
	assert.Error(t, json.Unmarshal([]byte(`false`), &bad), "invalid input should error")
}

func TestPriceChangesWrapper_UnmarshalJSON(t *testing.T) {
	t.Parallel()
	var single PriceChangesWrapper
	require.NoError(t, json.Unmarshal([]byte(`{"symbol":"BTCUSDT","lastPrice":"100.5"}`), &single))
	require.Len(t, single, 1, "single object input must unmarshal to one element")
	assert.Equal(t, "BTCUSDT", single[0].Symbol, "symbol should match")

	var arr PriceChangesWrapper
	require.NoError(t, json.Unmarshal([]byte(`[{"symbol":"BTCUSDT","lastPrice":"100.5"},{"symbol":"ETHUSDT","lastPrice":"50.25"}]`), &arr))
	require.Len(t, arr, 2, "array input must unmarshal to two elements")
	assert.Equal(t, "ETHUSDT", arr[1].Symbol, "second symbol should match")

	var bad PriceChangesWrapper
	assert.Error(t, json.Unmarshal([]byte(`"oops"`), &bad), "invalid input should error")
}

func TestWsOptionIncomingResps_UnmarshalJSON(t *testing.T) {
	t.Parallel()
	var arr WsOptionIncomingResps
	require.NoError(t, json.Unmarshal([]byte(`[{"id":1,"e":"depth"},{"id":2,"e":"ticker"}]`), &arr))
	require.Len(t, arr.Instances, 2, "array input must unmarshal to two instances")
	assert.True(t, arr.IsSlice, "IsSlice should be true for array input")
	assert.Equal(t, int64(2), arr.Instances[1].ID, "second instance id should match")

	var single WsOptionIncomingResps
	require.NoError(t, json.Unmarshal([]byte(`{"id":1,"e":"depth"}`), &single))
	require.Len(t, single.Instances, 1, "single object input must unmarshal to one instance")
	assert.False(t, single.IsSlice, "IsSlice should be false for single object input")
	assert.Equal(t, "depth", single.Instances[0].EventType, "event type should match")

	var bad WsOptionIncomingResps
	assert.Error(t, json.Unmarshal([]byte(`"oops"`), &bad), "invalid input should error")
}

func TestAssetIndexResponse_UnmarshalJSON(t *testing.T) {
	t.Parallel()
	var arr AssetIndexResponse
	require.NoError(t, json.Unmarshal([]byte(`[{"symbol":"BTCUSDT","index":"100"},{"symbol":"ETHUSDT","index":"50"}]`), &arr))
	require.Len(t, arr, 2, "array input must unmarshal to two elements")
	assert.Equal(t, 100.0, arr[0].Index.Float64(), "first index should match")

	var single AssetIndexResponse
	require.NoError(t, json.Unmarshal([]byte(`{"symbol":"BTCUSDT","index":"100"}`), &single))
	require.Len(t, single, 1, "single object input must unmarshal to one element")
	assert.Equal(t, "BTCUSDT", single[0].Symbol, "symbol should match")

	var bad AssetIndexResponse
	assert.Error(t, json.Unmarshal([]byte(`"oops"`), &bad), "invalid input should error")
}

func TestAccountBalanceResponse_UnmarshalJSON(t *testing.T) {
	t.Parallel()
	var arr AccountBalanceResponse
	require.NoError(t, json.Unmarshal([]byte(`[{"asset":"BTC","totalWalletBalance":"1.5"},{"asset":"ETH","totalWalletBalance":"10"}]`), &arr))
	require.Len(t, arr, 2, "array input must unmarshal to two elements")
	assert.Equal(t, 1.5, arr[0].TotalWalletBalance.Float64(), "first wallet balance should match")

	var single AccountBalanceResponse
	require.NoError(t, json.Unmarshal([]byte(`{"asset":"BTC","totalWalletBalance":"1.5"}`), &single))
	require.Len(t, single, 1, "single object input must unmarshal to one element")
	assert.Equal(t, "BTC", single[0].Asset, "asset should match")

	var bad AccountBalanceResponse
	assert.Error(t, json.Unmarshal([]byte(`"oops"`), &bad), "invalid input should error")
}
