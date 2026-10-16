package deribit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestAPIErrorError(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		err *APIError
		exp string
	}{
		{err: &APIError{Code: 11094, Message: "internal_server_error"}, exp: "code 11094: internal_server_error"},
		{err: &APIError{Code: -32602, Message: "Invalid params", Data: json.RawMessage(`{"reason":"invalid index","param":"index_name"}`)}, exp: `code -32602: Invalid params {"reason":"invalid index","param":"index_name"}`},
		{err: &APIError{Code: 10028, Message: "too_many_requests", Data: json.RawMessage(`null`)}, exp: "code 10028: too_many_requests"},
	} {
		assert.Equal(t, tc.exp, tc.err.Error(), "Error should return the code and message, and any details")
	}
}

func TestOrderPriceUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		input string
		exp   OrderPrice
	}{
		{input: `85000.5`, exp: OrderPrice{Value: 85000.5}},
		{input: `"85000.5"`, exp: OrderPrice{Value: 85000.5}},
		{input: `"market_price"`, exp: OrderPrice{Market: true}},
		{input: `null`},
	} {
		var price OrderPrice
		require.NoErrorf(t, json.Unmarshal([]byte(tc.input), &price), "Unmarshal must not error for %s", tc.input)
		assert.Equalf(t, tc.exp, price, "UnmarshalJSON should decode %s", tc.input)
	}
	for _, input := range []string{`"limit"`, `true`, `{}`} {
		var price OrderPrice
		assert.ErrorIsf(t, json.Unmarshal([]byte(input), &price), errInvalidOrderPrice, "UnmarshalJSON should reject %s", input)
	}
}

func TestObscuredUserIDUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		input string
		exp   ObscuredUserID
	}{
		{input: `"***009"`, exp: "***009"},
		{input: `57874`, exp: "57874"},
		{input: `null`},
	} {
		var userID ObscuredUserID
		require.NoErrorf(t, json.Unmarshal([]byte(tc.input), &userID), "Unmarshal must not error for %s", tc.input)
		assert.Equalf(t, tc.exp, userID, "UnmarshalJSON should decode %s", tc.input)
	}
	for _, input := range []string{`-1`, `1.5`, `true`, `{}`} {
		var userID ObscuredUserID
		assert.ErrorIsf(t, json.Unmarshal([]byte(input), &userID), errInvalidObscuredUserID, "UnmarshalJSON should reject %s", input)
	}
}

func TestTradeAllocationObscuredUserID(t *testing.T) {
	t.Parallel()
	var allocations []TradeAllocation
	require.NoError(t, json.Unmarshal([]byte(`[{"user_id":"***009","amount":10000,"client_info":{"client_id":1,"client_link_id":2,"name":"Client"}},{"user_id":57874,"amount":5000,"fee":0.0001}]`), &allocations), "Unmarshal must not error")
	exp := []TradeAllocation{
		{UserID: "***009", Amount: 10000, ClientInfo: &TradeAllocationClientInfo{ClientID: 1, ClientLinkID: 2, Name: "Client"}},
		{UserID: "57874", Amount: 5000, Fee: 0.0001},
	}
	assert.Equal(t, exp, allocations, "Unmarshal should decode a broker's obscured user ID and a plain user ID")
}

func TestUnmarshalTuple(t *testing.T) {
	t.Parallel()
	var timestamp types.Time
	var price float64
	require.NoError(t, unmarshalTuple([]byte(`[1573228800000,8751.7138636]`), &timestamp, &price), "unmarshalTuple must not error")
	assert.Equal(t, marketDataMilli(1573228800000), timestamp, "unmarshalTuple should decode the first element into the first target")
	assert.Equal(t, 8751.7138636, price, "unmarshalTuple should decode the second element into the second target")

	require.NoError(t, unmarshalTuple([]byte(` null `), &timestamp, &price), "unmarshalTuple must not error for null")
	assert.Equal(t, 8751.7138636, price, "unmarshalTuple should leave the targets unchanged for null")

	for _, input := range []string{`[1573228800000]`, `[1573228800000,8751.7138636,1]`} {
		assert.ErrorIsf(t, unmarshalTuple([]byte(input), &timestamp, &price), errInvalidTuple, "unmarshalTuple should reject %s", input)
	}
	assert.ErrorIs(t, unmarshalTuple([]byte(`{}`), &timestamp, &price), errInvalidTuple, "unmarshalTuple should reject an object")
	assert.ErrorIs(t, unmarshalTuple([]byte(`[1573228800000,"price"]`), &timestamp, &price), errInvalidTuple, "unmarshalTuple should reject an element its target cannot hold")
}
