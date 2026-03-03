package bybit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
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

func TestRetExtInfoUnmarshalJSON(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		payload   string
		expectLen int
		expectErr bool
	}{
		{
			name:      "null",
			payload:   `null`,
			expectLen: 0,
		},
		{
			name:      "empty string",
			payload:   `""`,
			expectLen: 0,
		},
		{
			// Bybit occasionally returns retExtInfo as a raw string; treat as empty metadata.
			name:      "raw string",
			payload:   `"ignored"`,
			expectLen: 0,
		},
		{
			name:      "object list",
			payload:   `{"list":[{"code":10001,"msg":"bad request"}]}`,
			expectLen: 1,
		},
		{
			name:      "invalid type",
			payload:   `123`,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var info retExtInfo
			err := info.UnmarshalJSON([]byte(tt.payload))
			if tt.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, info.List, tt.expectLen)
			if tt.expectLen == 1 {
				assert.EqualValues(t, 10001, info.List[0].Code)
				assert.Equal(t, "bad request", info.List[0].Message)
			}
		})
	}
}

func TestRestResponseRetExtInfoParsing(t *testing.T) {
	t.Parallel()

	t.Run("string retExtInfo", func(t *testing.T) {
		t.Parallel()
		var resp RestResponse
		err := json.Unmarshal([]byte(`{"retCode":0,"retMsg":"OK","result":{},"retExtInfo":"", "time":1700000000000}`), &resp)
		require.NoError(t, err)
		require.Empty(t, resp.RetExtInfo.List)
	})

	t.Run("object retExtInfo", func(t *testing.T) {
		t.Parallel()
		var resp RestResponse
		err := json.Unmarshal([]byte(`{"retCode":0,"retMsg":"OK","result":{},"retExtInfo":{"list":[{"code":123,"msg":"msg"}]},"time":1700000000000}`), &resp)
		require.NoError(t, err)
		require.Len(t, resp.RetExtInfo.List, 1)
		assert.EqualValues(t, 123, resp.RetExtInfo.List[0].Code)
		assert.Equal(t, "msg", resp.RetExtInfo.List[0].Message)
	})
}
