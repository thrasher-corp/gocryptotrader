package okx

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetSystemStatus(t *testing.T) {
	t.Parallel()
	result, err := e.GetSystemStatus(contextGenerate(), "completed")
	require.NoError(t, err, "GetSystemStatus must not error")
	if mockTests {
		exp := []SystemStatusResponse{
			{
				Title:           "Copy trading system unscheduled maintenance",
				State:           "completed",
				Begin:           types.Time(time.UnixMilli(1794902400000)),
				End:             types.Time(time.UnixMilli(1794905400000)),
				ServiceType:     "11",
				System:          "unified",
				MaintenanceType: "2",
				Environment:     "1",
			},
			{
				Title:           "Copy trading system scheduled maintenance",
				State:           "completed",
				Begin:           types.Time(time.UnixMilli(1792656000000)),
				End:             types.Time(time.UnixMilli(1792657200000)),
				ServiceType:     "11",
				System:          "unified",
				MaintenanceType: "1",
				Environment:     "1",
			},
		}
		assert.Equal(t, exp, result, "GetSystemStatus should decode every field")
	} else {
		assert.NotNil(t, result, "GetSystemStatus should return a list")
	}
}
