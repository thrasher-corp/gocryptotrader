package kraken

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

var (
	futuresTestContractPreference = FuturesAssignmentParticipant{
		ContractType:                        "flex",
		Contract:                            "PF_XBTUSD",
		MaxSize:                             new(250000.0),
		MaxPosition:                         new(1000000.0),
		AcceptLong:                          true,
		TimeFrame:                           "WEEKDAYS",
		Enabled:                             true,
		MinimumProfitabilityLongBasisPoints: new(uint64(90)),
	}
	futuresTestWalletPreference = FuturesAssignmentParticipant{
		ContractType:                         "fi_xbtusd",
		MaxSize:                              new(1000.0),
		MaxPosition:                          new(5000.0),
		AcceptLong:                           true,
		AcceptShort:                          true,
		TimeFrame:                            "ALL",
		Enabled:                              true,
		CashAccountOpenPositionMaxNotional:   new(750000.0),
		MinimumProfitabilityShortBasisPoints: new(uint64(110)),
	}
	futuresTestPairPreference = FuturesAssignmentParticipant{
		ContractType:                         "flex",
		MaxSize:                              new(500000.0),
		MaxPosition:                          new(2000000.0),
		AcceptLong:                           true,
		AcceptShort:                          true,
		TimeFrame:                            "WEEKEND",
		Enabled:                              true,
		CashAccountOpenPositionMaxNotional:   new(3000000.0),
		BaseCurrency:                         currency.XBT,
		QuoteCurrency:                        currency.USD,
		MinimumProfitabilityLongBasisPoints:  new(uint64(120)),
		MinimumProfitabilityShortBasisPoints: new(uint64(150)),
	}
)

func TestGetFuturesAssignmentPrograms(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesAssignmentPrograms(t.Context())
	require.NoError(t, err, "GetFuturesAssignmentPrograms must not error")
	if mockTests {
		exp := &FuturesAssignmentProgramsResponse{
			Participants: []FuturesAssignmentPreference{
				{ID: 4521, Participant: futuresTestContractPreference},
				{ID: 4522, Participant: futuresTestWalletPreference},
			},
			ServerTime: time.Date(2026, 10, 9, 0, 46, 2, 415000000, time.UTC),
		}
		assert.Equal(t, exp, result, "GetFuturesAssignmentPrograms should decode every field")
		return
	}
	assert.NotZero(t, result.ServerTime, "GetFuturesAssignmentPrograms should return the server time")
}

func TestAddFuturesAssignmentPreference(t *testing.T) {
	t.Parallel()
	_, err := e.AddFuturesAssignmentPreference(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "AddFuturesAssignmentPreference must reject a nil request")
	_, err = e.AddFuturesAssignmentPreference(t.Context(), &FuturesAddAssignmentPreferenceRequest{TimeFrame: "ALL"})
	require.ErrorIs(t, err, errFuturesContractTypeEmpty, "AddFuturesAssignmentPreference must reject an empty contract type")
	_, err = e.AddFuturesAssignmentPreference(t.Context(), &FuturesAddAssignmentPreferenceRequest{ContractType: "flex"})
	require.ErrorIs(t, err, errFuturesAssignmentTimeFrameEmpty, "AddFuturesAssignmentPreference must reject an empty time frame")
	_, err = e.AddFuturesAssignmentPreference(t.Context(), &FuturesAddAssignmentPreferenceRequest{ContractType: "flex", TimeFrame: "ALL", Contract: futuresTestPair, Pair: spotTestPair})
	require.ErrorIs(t, err, errFuturesAssignmentScopeConflict, "AddFuturesAssignmentPreference must reject a contract with a currency pair")
	_, err = e.AddFuturesAssignmentPreference(t.Context(), &FuturesAddAssignmentPreferenceRequest{ContractType: "flex", TimeFrame: "ALL", Pair: currency.Pair{Base: currency.XBT}})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "AddFuturesAssignmentPreference must reject a currency pair without a quote")
	_, err = e.AddFuturesAssignmentPreference(t.Context(), &FuturesAddAssignmentPreferenceRequest{ContractType: "flex", TimeFrame: "ALL", MaxSize: -1})
	require.ErrorIs(t, err, errFuturesNegativeLimit, "AddFuturesAssignmentPreference must reject a negative max size")
	_, err = e.AddFuturesAssignmentPreference(t.Context(), &FuturesAddAssignmentPreferenceRequest{ContractType: "flex", TimeFrame: "ALL", MaxPosition: new(-1.0)})
	require.ErrorIs(t, err, errFuturesNegativeLimit, "AddFuturesAssignmentPreference must reject a negative max position")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name string
		req  *FuturesAddAssignmentPreferenceRequest
		exp  *FuturesAssignmentPreferenceResponse
	}{
		{
			name: "currency pair",
			req: &FuturesAddAssignmentPreferenceRequest{
				ContractType:                         "flex",
				Pair:                                 spotTestPair,
				MaxSize:                              500000,
				MaxPosition:                          new(2000000.0),
				AcceptLong:                           true,
				AcceptShort:                          true,
				TimeFrame:                            "WEEKEND",
				Enabled:                              true,
				CashAccountOpenPositionMaxNotional:   new(3000000.0),
				MinimumProfitabilityLongBasisPoints:  new(uint64(120)),
				MinimumProfitabilityShortBasisPoints: new(uint64(150)),
			},
			exp: &FuturesAssignmentPreferenceResponse{
				ID:          4523,
				Participant: futuresTestPairPreference,
				ServerTime:  time.Date(2026, 10, 9, 0, 46, 30, 904000000, time.UTC),
			},
		},
		{
			name: "contract opt out",
			req: &FuturesAddAssignmentPreferenceRequest{
				ContractType: "flex",
				Contract:     currency.NewPairWithDelimiter("PF", "ETHUSD", currency.UnderscoreDelimiter),
				MaxPosition:  new(0.0),
				AcceptLong:   true,
				AcceptShort:  true,
				TimeFrame:    "ALL",
			},
			exp: &FuturesAssignmentPreferenceResponse{
				ID: 4524,
				Participant: FuturesAssignmentParticipant{
					ContractType: "flex",
					Contract:     "PF_ETHUSD",
					MaxPosition:  new(0.0),
					AcceptLong:   true,
					AcceptShort:  true,
					TimeFrame:    "ALL",
				},
				ServerTime: time.Date(2026, 10, 9, 0, 46, 33, 117000000, time.UTC),
			},
		},
		{
			name: "zero minimum profitability",
			req: &FuturesAddAssignmentPreferenceRequest{
				ContractType:                         "fi_ethusd",
				AcceptShort:                          true,
				TimeFrame:                            "WEEKDAYS",
				Enabled:                              true,
				MinimumProfitabilityShortBasisPoints: new(uint64(0)),
			},
			exp: &FuturesAssignmentPreferenceResponse{
				ID: 4525,
				Participant: FuturesAssignmentParticipant{
					ContractType:                         "fi_ethusd",
					AcceptShort:                          true,
					TimeFrame:                            "WEEKDAYS",
					Enabled:                              true,
					MinimumProfitabilityShortBasisPoints: new(uint64(0)),
				},
				ServerTime: time.Date(2026, 10, 9, 0, 46, 35, 262000000, time.UTC),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.AddFuturesAssignmentPreference(t.Context(), tc.req)
			require.NoError(t, err, "AddFuturesAssignmentPreference must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "AddFuturesAssignmentPreference should decode every field")
				return
			}
			assert.Equal(t, tc.exp.Participant.TimeFrame, result.Participant.TimeFrame, "AddFuturesAssignmentPreference should return the preference added")
		})
	}
}

func TestDeleteFuturesAssignmentPreference(t *testing.T) {
	t.Parallel()
	_, err := e.DeleteFuturesAssignmentPreference(t.Context(), 0)
	require.ErrorIs(t, err, errFuturesAssignmentIDEmpty, "DeleteFuturesAssignmentPreference must reject an empty ID")

	id := uint64(4522)
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		// A disabled preference opting one contract out is harmless to add, so the test never deletes a real preference
		added, err := e.AddFuturesAssignmentPreference(t.Context(), &FuturesAddAssignmentPreferenceRequest{ContractType: "flex", Contract: futuresTestPair, MaxPosition: new(0.0), TimeFrame: "ALL"})
		require.NoError(t, err, "AddFuturesAssignmentPreference must not error")
		id = added.ID
	}
	result, err := e.DeleteFuturesAssignmentPreference(t.Context(), id)
	require.NoError(t, err, "DeleteFuturesAssignmentPreference must not error")
	if mockTests {
		exp := &FuturesAssignmentPreferenceResponse{
			ID:          4522,
			Participant: futuresTestWalletPreference,
			ServerTime:  time.Date(2026, 10, 9, 0, 47, 12, 586000000, time.UTC),
		}
		assert.Equal(t, exp, result, "DeleteFuturesAssignmentPreference should decode every field")
		return
	}
	assert.Equal(t, id, result.ID, "DeleteFuturesAssignmentPreference should return the preference deleted")
}

func TestGetFuturesAssignmentPreferencesHistory(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesAssignmentPreferencesHistory(t.Context())
	require.NoError(t, err, "GetFuturesAssignmentPreferencesHistory must not error")
	if mockTests {
		exp := &FuturesAssignmentPreferencesHistoryResponse{
			Participants: []FuturesAssignmentPreferenceChange{
				{Participant: futuresTestPairPreference, Timestamp: time.Date(2026, 10, 8, 9, 15, 22, 0, time.UTC)},
				{Deleted: true, Participant: futuresTestWalletPreference, Timestamp: time.Date(2026, 10, 9, 0, 47, 12, 586000000, time.UTC)},
			},
			ServerTime: time.Date(2026, 10, 9, 0, 47, 40, 33000000, time.UTC),
		}
		assert.Equal(t, exp, result, "GetFuturesAssignmentPreferencesHistory should decode every field")
		return
	}
	assert.NotZero(t, result.ServerTime, "GetFuturesAssignmentPreferencesHistory should return the server time")
}
