package websocket

import (
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var upgrader = websocket.Upgrader{CheckOrigin: func(_ *http.Request) bool { return true }}

// WsMockFunc is a websocket handler to be called with each websocket message
type WsMockFunc func(testing.TB, []byte, *websocket.Conn) error

// CurryWsMockUpgrader curries a WsMockUpgrader with a testing.TB and a mock func
// bridging the gap between information known before the Server is created and during a request
func CurryWsMockUpgrader(tb testing.TB, wsHandler WsMockFunc) http.HandlerFunc {
	tb.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		WsMockUpgrader(tb, w, r, wsHandler)
	}
}

// WsMockUpgrader handles upgrading an initial HTTP request to WS, and then runs a for loop calling the mock func on each input
func WsMockUpgrader(tb testing.TB, w http.ResponseWriter, r *http.Request, wsHandler WsMockFunc) {
	tb.Helper()
	c, err := upgrader.Upgrade(w, r, nil)
	require.NoError(tb, err, "Upgrade connection must not error")
	defer c.Close()
	for {
		_, p, err := c.ReadMessage()
		if err != nil {
			// Any error here is likely due to the connection closing
			return
		}
		err = wsHandler(tb, p, c)
		assert.NoError(tb, err, "WS Mock Function should not error")
	}
}

// EchoHandler is a simple echo function after a read, this doesn't need to worry if writing to the connection fails
func EchoHandler(_ testing.TB, p []byte, c *websocket.Conn) error {
	time.Sleep(time.Nanosecond) // Shift clock to simulate time passing
	_ = c.WriteMessage(websocket.TextMessage, p)
	return nil
}
