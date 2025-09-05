package bybit

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
)

func TestWSHandleTradeData(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		input []byte
		match string
		err   error
	}{
		{input: []byte(`{"reqId":"12345"}`), match: "12345", err: nil},
		{input: []byte(`{"op":"auth"}`), match: "auth", err: nil},
		{input: []byte(`{"op":"pong"}`), err: nil},
		{input: []byte(`{"op":"pewpewpew"}`), err: errUnhandledStreamData},
	} {
		conn := &FixtureConnection{match: websocket.NewMatch()}
		var ch <-chan []byte
		if tc.match != "" {
			var err error
			ch, err = conn.match.Set(tc.match, 1)
			require.NoError(t, err, "match.Set must not error")
		}
		err := e.wsHandleTradeData(conn, tc.input)
		if tc.err != nil {
			require.ErrorIs(t, err, tc.err)
			continue
		}
		require.NoError(t, err)
		if tc.match != "" {
			require.Len(t, ch, 1, "must receive 1 message from channel")
			require.Equal(t, tc.input, <-ch, "must be correct")
		}
	}
}
