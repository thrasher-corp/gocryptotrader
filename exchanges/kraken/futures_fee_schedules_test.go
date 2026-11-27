package kraken

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

func TestGetFuturesFeeSchedules(t *testing.T) {
	t.Parallel()
	result, err := e.GetFuturesFeeSchedules(t.Context())
	require.NoError(t, err, "GetFuturesFeeSchedules must not error")
	if mockTests {
		exp := &FuturesFeeSchedulesResponse{
			FeeSchedules: []FuturesFeeSchedule{
				{
					UID:  "4845910e-81a7-4578-bdeb-8faada1b972f",
					Name: "Incentive Rebate MTF Linear Fees",
					Tiers: []FuturesFeeTier{
						{MakerFee: 0.02, TakerFee: 0.05},
						{MakerFee: 0.0175, TakerFee: 0.045, USDVolume: 5000000},
					},
				},
				{
					UID:   "535cacd9-217d-442b-98e3-489c81019eb9",
					Name:  "Consumer MTF Linear Fees",
					Tiers: []FuturesFeeTier{{MakerFee: 0.25, TakerFee: 0.26}},
				},
			},
			ServerTime: time.Date(2026, 10, 9, 0, 34, 18, 715000000, time.UTC),
		}
		assert.Equal(t, exp, result, "GetFuturesFeeSchedules should decode every field")
		return
	}
	assert.NotEmpty(t, result.FeeSchedules, "GetFuturesFeeSchedules should return fee schedules")
}

func TestGetFuturesFeeScheduleVolumes(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesFeeScheduleVolumes(t.Context())
	require.NoError(t, err, "GetFuturesFeeScheduleVolumes must not error")
	if mockTests {
		exp := &FuturesFeeScheduleVolumesResponse{
			VolumesByFeeSchedule: map[string]float64{
				"4845910e-81a7-4578-bdeb-8faada1b972f": 53823.71,
				"535cacd9-217d-442b-98e3-489c81019eb9": 1250,
			},
			ServerTime: time.Date(2026, 10, 9, 0, 34, 19, 120000000, time.UTC),
		}
		assert.Equal(t, exp, result, "GetFuturesFeeScheduleVolumes should decode every field")
		return
	}
	assert.NotNil(t, result.VolumesByFeeSchedule, "GetFuturesFeeScheduleVolumes should return the volumes")
}
