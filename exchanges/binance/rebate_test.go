package binance

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetSpotRebateHistoryRecords(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetSpotRebateHistoryRecords(t.Context(), endTime, startTime, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSpotRebateHistoryRecords must reject a reversed window")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSpotRebateHistoryRecords(t.Context(), startTime, endTime, 10)
	require.NoError(t, err, "GetSpotRebateHistoryRecords must not error")
	if mockTests {
		exp := &RebateHistoryResponse{
			Status: "OK",
			Type:   "GENERAL",
			Code:   "000000000",
			Data: RebateHistoryPage{
				Page:            10,
				TotalRecords:    2000,
				TotalPageNumber: 10,
				Data: []RebateRecord{
					{
						Asset:      currency.USDT,
						Type:       1,
						Amount:     0.0001126,
						UpdateTime: types.Time(time.Unix(1744150000, 0)),
					},
					{
						Asset:      currency.BNB,
						Type:       2,
						Amount:     0.00000431,
						UpdateTime: types.Time(time.Unix(1744160000, 0)),
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetSpotRebateHistoryRecords should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSpotRebateHistoryRecords should return a result")
}
