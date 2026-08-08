package mexc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

// TestFuturesContractsListUnmarshalJSON asserts the endpoint's two shapes both decode: it returns a
// list when queried for every contract and a bare object when queried for one.
func TestFuturesContractsListUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var list FuturesContractsList
	require.NoError(t, json.Unmarshal([]byte(`[{"symbol":"BTC_USDT"},{"symbol":"ETH_USDT"}]`), &list), "a list payload must decode")
	require.Len(t, list, 2, "both contracts must be decoded")
	assert.Equal(t, "BTC_USDT", list[0].Symbol, "the first symbol should be correct")
	assert.Equal(t, "ETH_USDT", list[1].Symbol, "the second symbol should be correct")

	var single FuturesContractsList
	require.NoError(t, json.Unmarshal([]byte(`{"symbol":"BTC_USDT"}`), &single), "a single object payload must decode")
	require.Len(t, single, 1, "a bare object must decode to a single element list")
	assert.Equal(t, "BTC_USDT", single[0].Symbol, "the symbol should be correct")

	var bad FuturesContractsList
	assert.Error(t, json.Unmarshal([]byte(`"not-a-contract"`), &bad), "an unexpected shape should be reported")
}

// TestContractTickersListUnmarshalJSON asserts the same dual shape for contract tickers.
func TestContractTickersListUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var list ContractTickersList
	require.NoError(t, json.Unmarshal([]byte(`[{"symbol":"BTC_USDT"},{"symbol":"ETH_USDT"}]`), &list), "a list payload must decode")
	require.Len(t, list, 2, "both tickers must be decoded")
	assert.Equal(t, "BTC_USDT", list[0].Symbol, "the first symbol should be correct")

	var single ContractTickersList
	require.NoError(t, json.Unmarshal([]byte(`{"symbol":"BTC_USDT"}`), &single), "a single object payload must decode")
	require.Len(t, single, 1, "a bare object must decode to a single element list")
	assert.Equal(t, "BTC_USDT", single[0].Symbol, "the symbol should be correct")

	var bad ContractTickersList
	assert.Error(t, json.Unmarshal([]byte(`"not-a-ticker"`), &bad), "an unexpected shape should be reported")
}

// TestOrderbookDataWithDepthUnmarshalJSON asserts the positional [price, amount, orders] tuple is
// mapped to the right fields.
func TestOrderbookDataWithDepthUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var level OrderbookDataWithDepth
	require.NoError(t, json.Unmarshal([]byte(`[93180.18,0.21976424,5]`), &level), "a depth tuple must decode")
	assert.Equal(t, 93180.18, level.Price, "the first element should be the price")
	assert.Equal(t, 0.21976424, level.Amount, "the second element should be the amount")
	assert.Equal(t, int64(5), level.OrderCount, "the third element should be the order count")

	var bad OrderbookDataWithDepth
	assert.Error(t, json.Unmarshal([]byte(`{"price":1}`), &bad), "a non tuple payload should be reported")
}
