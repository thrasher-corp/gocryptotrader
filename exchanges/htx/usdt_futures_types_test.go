package htx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

func TestV5BooleanUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		input    string
		expected V5Boolean
		wantErr  bool
	}{
		{name: "empty string", input: `""`},
		{name: "boolean", input: `true`, expected: true},
		{name: "invalid", input: `null`, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var value V5Boolean
			err := value.UnmarshalJSON([]byte(tc.input))
			if tc.wantErr {
				require.Error(t, err, "UnmarshalJSON must reject an invalid V5 boolean")
				return
			}
			require.NoError(t, err, "UnmarshalJSON must decode a V5 boolean")
			assert.Equal(t, tc.expected, value, "decoded boolean should match")
		})
	}
}

func TestV5BooleanBool(t *testing.T) {
	t.Parallel()
	assert.True(t, V5Boolean(true).Bool(), "Bool should return true")
	assert.False(t, V5Boolean(false).Bool(), "Bool should return false")
}

func TestV5OrderStateUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name        string
		input       string
		expected    V5OrderState
		expectedErr error
		wantErr     bool
	}{
		{name: "named", input: `"filled"`, expected: "filled"},
		{name: "numeric", input: `3`, expected: "ACTIVE"},
		{name: "malformed named", input: `"filled`, wantErr: true},
		{name: "non-numeric", input: `true`, wantErr: true},
		{name: "unsupported numeric", input: `99`, expectedErr: errInvalidOrderStatus},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var state V5OrderState
			err := state.UnmarshalJSON([]byte(tc.input))
			if tc.wantErr || tc.expectedErr != nil {
				if tc.expectedErr != nil {
					require.ErrorIs(t, err, tc.expectedErr, "UnmarshalJSON must return the expected state error")
				} else {
					require.Error(t, err, "UnmarshalJSON must reject invalid state data")
				}
				return
			}
			require.NoError(t, err, "UnmarshalJSON must decode a V5 order state")
			assert.Equal(t, tc.expected, state, "decoded state should match")
		})
	}
}

func TestV5OrderQueryResponseUnmarshal(t *testing.T) {
	t.Parallel()
	var resp *V5OrderQueryResponse
	err := json.Unmarshal([]byte(`{"code":200,"message":"Success","data":{"order_id":"1","contract_code":"BTC-USDT","side":"buy","type":"limit","price":"5000","volume":"1","trade_volume":"0.25","trade_turnover":"1250","fee":"0.1","lever_rate":10,"state":3,"reduce_only":"","price_protect":"","created_time":"1769076510922","updated_time":"1769076510922"}}`), &resp)
	require.NoError(t, err, "Unmarshal must decode HTX V5 order response")
	require.NotNil(t, resp, "response must not be nil")
	assert.Equal(t, 5000.0, resp.Data.Price.Float64(), "price should decode from quoted number")
	assert.Equal(t, 0.25, resp.Data.TradeVolume.Float64(), "trade volume should decode from quoted number")
	assert.Equal(t, 10.0, resp.Data.LeverageRate.Float64(), "leverage should decode from bare number")
	assert.Equal(t, V5OrderState("ACTIVE"), resp.Data.State, "numeric order state should decode")
	assert.False(t, resp.Data.ReduceOnly.Bool(), "empty reduce-only should decode as false")
	assert.False(t, resp.Data.PriceProtect.Bool(), "empty price-protect should decode as false")
}

func TestV5OrderResponseDataUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name          string
		payload       string
		orderID       string
		clientOrderID string
	}{
		{name: "string identifiers", payload: `{"order_id":"123","client_order_id":"456"}`, orderID: "123", clientOrderID: "456"},
		{name: "numeric identifiers", payload: `{"order_id":1358944503420903424,"client_order_id":1358944503420903425}`, orderID: "1358944503420903424", clientOrderID: "1358944503420903425"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var response V5OrderResponseData
			require.NoError(t, json.Unmarshal([]byte(tc.payload), &response), "V5OrderResponseData unmarshal must not error")
			assert.Equal(t, tc.orderID, response.OrderID, "order ID should retain full precision")
			assert.Equal(t, tc.clientOrderID, response.ClientOrderID, "client order ID should retain full precision")
		})
	}
}

func TestV5OpenInterestResponseUnmarshal(t *testing.T) {
	t.Parallel()
	var resp *V5OpenInterestResponse
	err := json.Unmarshal([]byte(`{"code":200,"data":{"amount":"244.004","volume":"244004","value":"29275599.92","contract_code":"BTC-USDT","trade_amount":"9.838","trade_volume":"9838","trade_turnover":"1091416.458752"},"message":null,"success":true}`), &resp)
	require.NoError(t, err, "Unmarshal must decode HTX V5 open interest response")
	require.NotNil(t, resp, "response must not be nil")
	assert.True(t, resp.Success, "success should decode")
	assert.Equal(t, 244.004, resp.Data.Amount.Float64(), "amount should decode from quoted number")
	assert.Equal(t, 1091416.458752, resp.Data.TradeTurnover.Float64(), "trade turnover should decode from quoted number")
}
