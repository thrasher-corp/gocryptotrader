package htx

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

func TestSwapWebsocketTriggerOrder(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, triggered, inserted   string
		wantTriggered, wantInserted time.Time
	}{
		{name: "executed", triggered: `1603880263722`, inserted: `1603880263723`, wantTriggered: time.UnixMilli(1603880263722), wantInserted: time.UnixMilli(1603880263723)},
		{name: "pending", triggered: `0`, inserted: `0`},
		{name: "unset", triggered: `null`, inserted: `null`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			raw := `{"op":"notify","topic":"trigger_order.BTC-USD","ts":1603880263724,"data":[{"order_id":9007199254740993,"created_at":1603880263721,"triggered_at":` + tc.triggered + `,"order_insert_at":` + tc.inserted + `}]}`
			var response SwapWebsocketTriggerOrderUpdates
			require.NoError(t, json.Unmarshal([]byte(raw), &response), "trigger notification must decode")
			require.Len(t, response.Data, 1, "notification must retain its trigger order")
			entry := response.Data[0]
			assert.Equal(t, uint64(9007199254740993), entry.OrderID, "order ID should retain all digits")
			assert.Equal(t, time.UnixMilli(1603880263721), entry.CreatedAt.Time(), "creation time should retain milliseconds")
			assert.Equal(t, tc.wantTriggered, entry.TriggeredAt.Time(), "trigger time should reflect execution state")
			assert.Equal(t, tc.wantInserted, entry.OrderInsertAt.Time(), "order insertion time should reflect execution state")
		})
	}
}
