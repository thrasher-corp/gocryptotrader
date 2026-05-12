//go:build integration

package okx

import (
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

// TestPublicWSSubscribeBBOTBTBTCUSDT connects to OKX public websocket and
// subscribes to bbo-tbt for BTC-USDT so payload cadence and shape can be observed.
func TestPublicWSSubscribeBBOTBTBTCUSDT(t *testing.T) {
	t.Parallel()
	const sampleCount = 60
	conn, resp, err := gws.DefaultDialer.Dial(apiWebsocketPublicURL, nil)
	if resp != nil && resp.Body != nil {
		t.Cleanup(func() {
			require.NoError(t, resp.Body.Close(), "response body close must not error")
		})
	}
	require.NoError(t, err, "websocket dial must not error")
	t.Cleanup(func() {
		require.NoError(t, conn.Close(), "websocket close must not error")
	})

	subReq := map[string]any{
		"op": "subscribe",
		"args": []map[string]string{{
			"channel": "bbo-tbt",
			"instId":  "BTC-USDT",
		}},
	}
	require.NoError(t, conn.WriteJSON(subReq), "subscribe request must not error")

	type envelope struct {
		Event string `json:"event"`
		Code  string `json:"code"`
		Msg   string `json:"msg"`
	}

	timestamps := make([]time.Time, 0, sampleCount)
	samples := make([]string, 0, sampleCount)

	deadline := time.Now().Add(time.Duration(sampleCount) * time.Second)
	for len(samples) < sampleCount && time.Now().Before(deadline) {
		require.NoError(t, conn.SetReadDeadline(time.Now().Add(time.Duration(sampleCount)*time.Second)), "setting read deadline must not error")
		_, message, readErr := conn.ReadMessage()
		require.NoError(t, readErr, "reading websocket message must not error")

		var event envelope
		if err := json.Unmarshal(message, &event); err == nil {
			if event.Event == "error" {
				t.Fatalf("subscription must not error: code=%s msg=%s", event.Code, event.Msg)
			}
			if event.Event == "subscribe" {
				continue
			}
		}

		timestamps = append(timestamps, time.Now())
		samples = append(samples, string(message))
	}

	require.Len(t, samples, sampleCount, "must capture bbo-tbt payload messages")

	t.Log("captured bbo-tbt samples:")
	for i := range samples {
		if i == 0 {
			t.Logf("sample %d: %s", i+1, samples[i])
			continue
		}
		delta := timestamps[i].Sub(timestamps[i-1])
		t.Logf("sample %d (+%s): %s", i+1, delta.Truncate(time.Microsecond), samples[i])
	}

	if len(timestamps) > 1 {
		var total time.Duration
		for i := 1; i < len(timestamps); i++ {
			total += timestamps[i].Sub(timestamps[i-1])
		}
		avg := total / time.Duration(len(timestamps)-1)
		t.Logf("average inter-message delta across %d intervals: %s", len(timestamps)-1, avg.Truncate(time.Microsecond))
	}
}
