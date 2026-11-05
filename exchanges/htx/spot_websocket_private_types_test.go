package htx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

func TestWebsocketAccountUpdateJSON(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name     string
		sequence string
	}
	for _, tc := range []testCase{
		{name: "numeric sequence", sequence: `9007199254740993`},
		{name: "quoted sequence", sequence: `"9007199254740993"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var update WebsocketAccountUpdate
			require.NoError(t, json.Unmarshal([]byte(`{"accountId":9007199254740993,"seqNum":`+tc.sequence+`}`), &update), "account update must decode both documented sequence representations")
			assert.Equal(t, uint64(9007199254740993), update.AccountID, "account ID should retain its exact value")
			assert.Equal(t, "9007199254740993", update.SequenceNumber.String(), "sequence number should retain its exact value")
		})
	}
}

func TestWebsocketTradeUpdateJSON(t *testing.T) {
	t.Parallel()
	var update WebsocketTradeUpdate
	require.NoError(t, json.Unmarshal([]byte(`{"orderId":9007199254740993,"tradeId":102523573486,"feeCurrency":"btc","transactFee":"-0.0001"}`), &update), "trade clearing update must decode")
	assert.Equal(t, uint64(9007199254740993), update.OrderID, "order ID should retain its exact value")
	assert.True(t, currency.BTC.Equal(update.FeeCurrency), "fee currency should identify the single fee asset")
	assert.Equal(t, -0.0001, update.TransactFee.Float64(), "fee rebate should retain its negative sign")
}

func TestWebsocketOrderUpdateErrorCodes(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name, key string
		response  any
	}
	for _, tc := range []testCase{
		{name: "WebsocketOrderUpdate", key: "errCode", response: new(WebsocketOrderUpdate)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.NoError(t, json.Unmarshal([]byte(`{"`+tc.key+`":1001}`), tc.response), "documented positive error code must decode")
			require.Error(t, json.Unmarshal([]byte(`{"`+tc.key+`":-1}`), tc.response), "negative error code must not decode into an unsigned field")
		})
	}
}
