package htx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

func TestWebsocketTradeIDUnmarshalJSON(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name        string
		input       []byte
		expected    WebsocketTradeID
		expectedErr error
		wantErr     bool
	}
	for _, tc := range []testCase{
		{name: "numeric", input: []byte(`137005445109359290000000000`), expected: "137005445109359290000000000"},
		{name: "quoted", input: []byte(`"137005445109359290000000000"`), expected: "137005445109359290000000000"},
		{name: "whitespace", input: []byte(" 123 "), expected: "123"},
		{name: "empty", expectedErr: errInvalidTradeID},
		{name: "non-integer", input: []byte(`12.3`), expectedErr: errInvalidTradeID},
		{name: "malformed string", input: []byte(`"123`), wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var id WebsocketTradeID
			err := id.UnmarshalJSON(tc.input)
			if tc.wantErr || tc.expectedErr != nil {
				if tc.expectedErr != nil {
					require.ErrorIs(t, err, tc.expectedErr, "UnmarshalJSON must return the expected error")
				} else {
					require.Error(t, err, "UnmarshalJSON must reject malformed input")
				}
				return
			}
			require.NoError(t, err, "UnmarshalJSON must decode the trade ID")
			assert.Equal(t, tc.expected, id, "trade ID should retain its exact value")
		})
	}

	t.Run("feed payload", func(t *testing.T) {
		t.Parallel()
		var response WebsocketTrade
		require.NoError(t, json.Unmarshal([]byte(`{"tick":{"data":[{"id":137005445109359290000000000}]}}`), &response), "large derivative trade ID must decode")
		require.Len(t, response.Tick.Trades, 1, "one trade must decode")
		assert.Equal(t, WebsocketTradeID("137005445109359290000000000"), response.Tick.Trades[0].ID, "trade ID should retain full precision")
	})
}
