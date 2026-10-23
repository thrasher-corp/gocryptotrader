package htx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

func TestWSFuturesTimestampUnmarshalJSON(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name, input, want string
		wantErr           bool
	}
	for _, tc := range []testCase{
		{name: "numeric", input: `1492420473027`, want: `1492420473027`},
		{name: "quoted", input: `"1492420473027"`, want: `"1492420473027"`},
		{name: "invalid", input: `1`, wantErr: true},
		{name: "zero", input: `null`, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var timestamp wsFuturesTimestamp
			err := timestamp.UnmarshalJSON([]byte(tc.input))
			if tc.wantErr {
				require.Error(t, err, "malformed futures timestamp must be rejected")
				return
			}
			require.NoError(t, err, "valid futures timestamp must decode")
			assert.Equal(t, wsFuturesTimestamp(tc.want), timestamp, "timestamp should preserve its JSON representation")
		})
	}
}

func TestWSFuturesTimestampMarshalJSON(t *testing.T) {
	t.Parallel()
	for _, input := range []string{`1492420473027`, `"1492420473027"`} {
		t.Run(input, func(t *testing.T) {
			t.Parallel()
			encoded, err := json.Marshal(wsFuturesTimestamp(input))
			require.NoError(t, err, "futures timestamp must encode")
			assert.Equal(t, input, string(encoded), "futures timestamp should retain its JSON type")
		})
	}
}

func TestFuturesWebsocketErrorCodes(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name, key string
		response  any
	}
	for _, tc := range []testCase{
		{name: "FuturesWebsocketTriggerOrder", key: "fail_code", response: new(FuturesWebsocketTriggerOrder)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.NoError(t, json.Unmarshal([]byte(`{"`+tc.key+`":1001}`), tc.response), "documented positive error code must decode")
			require.Error(t, json.Unmarshal([]byte(`{"`+tc.key+`":-1}`), tc.response), "negative error code must not decode into an unsigned field")
		})
	}
}
