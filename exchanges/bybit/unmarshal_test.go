package bybit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestUnmarshalJSONKlineItem(t *testing.T) {
	t.Parallel()
	ki := &KlineItem{}
	err := ki.UnmarshalJSON([]byte(`["1691905800000","0.000301","0.0003015","0.0002995","0.0003","213303600","64084.7623"]`))
	require.NoError(t, err)

	require.Equal(t, time.UnixMilli(1691905800000), ki.StartTime.Time())
	require.Equal(t, types.Number(0.000301), ki.Open)
	require.Equal(t, types.Number(0.0003015), ki.High)
	require.Equal(t, types.Number(0.0002995), ki.Low)
	require.Equal(t, types.Number(0.0003), ki.Close)
	require.Equal(t, types.Number(213303600), ki.TradeVolume)
	require.Equal(t, types.Number(64084.7623), ki.Turnover)
}
