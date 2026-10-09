package kraken

import (
	"io"
	"math"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestListEarnStrategies(t *testing.T) {
	t.Parallel()
	_, err := e.ListEarnStrategies(t.Context(), &EarnStrategiesRequest{Limit: math.MaxUint16 + 1})
	require.ErrorIs(t, err, errInvalidCount, "ListEarnStrategies must reject a limit above 65535")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *EarnStrategiesRequest
		exp  *EarnStrategiesResponse
	}{
		{
			name: "filtered",
			req:  &EarnStrategiesRequest{Ascending: true, Asset: currency.DOT, Cursor: "1", Limit: 10},
			exp: &EarnStrategiesResponse{
				Items: []EarnStrategy{
					{
						AllocationFee:             0.001,
						AllocationRestrictionInfo: []string{},
						APREstimate:               &EarnAPREstimate{High: 12, Low: 8},
						Asset:                     currency.DOT,
						AutoCompound:              EarnAutoCompound{Type: "enabled"},
						CanAllocate:               true,
						CanDeallocate:             true,
						DeallocationFee:           0.002,
						ID:                        "ESRFUO3-Q62XD-WIOIL7",
						LockType:                  EarnLockType{PayoutFrequencySeconds: 604800, Type: "instant"},
						UserCap:                   new(types.Number(50000)),
						UserMinimumAllocation:     0.01,
						YieldSource:               EarnYieldSource{Type: "staking"},
					},
				},
				NextCursor: "2",
			},
		},
		{
			name: "every strategy",
			exp: &EarnStrategiesResponse{
				Items: []EarnStrategy{
					{
						AllocationFee:             0.25,
						AllocationRestrictionInfo: []string{"tier"},
						APREstimate:               &EarnAPREstimate{High: 5, Low: 2.5},
						Asset:                     currency.ETH,
						AutoCompound:              EarnAutoCompound{Default: true, Type: "optional"},
						CanDeallocate:             true,
						DeallocationFee:           0.5,
						ID:                        "ESDQCOL-WTZEU-NU55QF",
						LockType: EarnLockType{
							BondingPeriodSeconds:    3456000,
							BondingPeriodVariable:   true,
							BondingRewards:          true,
							ExitQueuePeriodSeconds:  1296000,
							PayoutFrequencySeconds:  604800,
							Type:                    "bonded",
							UnbondingPeriodSeconds:  1209600,
							UnbondingPeriodVariable: true,
							UnbondingRewards:        true,
						},
						UserCap:               new(types.Number(0)),
						UserMinimumAllocation: 1,
						YieldSource:           EarnYieldSource{Type: "staking"},
					},
					{
						AllocationFee:             0.1,
						AllocationRestrictionInfo: []string{},
						APREstimate:               &EarnAPREstimate{High: 7.5, Low: 6},
						Asset:                     currency.SOL,
						AutoCompound:              EarnAutoCompound{Type: "optional"},
						CanAllocate:               true,
						DeallocationFee:           0.2,
						ID:                        "ESMWVX6-JAPVY-23L3CV",
						LockType: EarnLockType{
							BondingPeriodSeconds:    345600,
							PayoutFrequencySeconds:  172800,
							Type:                    "bonded",
							UnbondingPeriodSeconds:  259200,
							UnbondingPeriodVariable: true,
							UnbondingRewards:        true,
						},
						UserCap:               new(types.Number(1000)),
						UserMinimumAllocation: 5,
						YieldSource:           EarnYieldSource{Type: "staking"},
					},
					{
						AllocationRestrictionInfo: []string{},
						Asset:                     currency.USDC,
						AutoCompound:              EarnAutoCompound{Type: "disabled"},
						ID:                        "ESXUM7H-SJHQ6-KOQNNI",
						LockType:                  EarnLockType{Type: "flex"},
						UserMinimumAllocation:     1,
						YieldSource:               EarnYieldSource{Type: "off_chain"},
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.ListEarnStrategies(t.Context(), tc.req)
			require.NoError(t, err, "ListEarnStrategies must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "ListEarnStrategies should decode every field")
				return
			}
			assert.NotNil(t, result, "ListEarnStrategies should return strategies")
		})
	}
}

// TestListEarnStrategiesBody checks against a local server that the lock types are sent as a JSON array, which the VCR
// server cannot match in a recorded body, beside the largest limit Kraken documents
func TestListEarnStrategiesBody(t *testing.T) {
	t.Parallel()
	ex := newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/0/private/Earn/Strategies", r.URL.Path, "ListEarnStrategies should request Earn/Strategies")
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err, "ReadAll should not error")
		var payload map[string]any
		assert.NoError(t, json.Unmarshal(body, &payload), "Unmarshal should not error")
		assert.Contains(t, payload, "nonce", "body should carry the nonce")
		delete(payload, "nonce")
		exp := map[string]any{"limit": float64(math.MaxUint16), "lock_type": []any{"bonded", "instant"}}
		assert.Equal(t, exp, payload, "body should carry the limit and the lock types as an array")
		_, _ = w.Write([]byte(`{"error":[],"result":{"items":[],"next_cursor":null}}`))
	})
	result, err := ex.ListEarnStrategies(t.Context(), &EarnStrategiesRequest{Limit: math.MaxUint16, LockTypes: []string{"bonded", "instant"}})
	require.NoError(t, err, "ListEarnStrategies must not error for the largest limit")
	assert.Equal(t, &EarnStrategiesResponse{Items: []EarnStrategy{}}, result, "ListEarnStrategies should decode an empty page")
}

func TestListEarnAllocations(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *EarnAllocationsRequest
		exp  *EarnAllocationsResponse
	}{
		{
			name: "filtered",
			req:  &EarnAllocationsRequest{Ascending: true, ConvertedAsset: currency.EUR, HideZeroAllocations: true},
			exp: &EarnAllocationsResponse{
				ConvertedAsset: currency.EUR,
				Items: []EarnAllocation{
					{
						AmountAllocated: EarnAmountAllocated{
							Bonding: &EarnAllocationState{
								AllocationCount: 2,
								Allocations: []EarnAllocationTransition{
									{
										ConvertedAmount: 1.8602,
										CreatedAt:       time.Date(2026, 9, 6, 10, 52, 5, 0, time.UTC),
										Expires:         time.Date(2026, 10, 19, 2, 34, 5, 807000000, time.UTC),
										NativeAmount:    0.001,
									},
									{
										ConvertedAmount: 37.2043,
										CreatedAt:       time.Date(2026, 10, 1, 11, 25, 52, 0, time.UTC),
										Expires:         time.Date(2026, 11, 6, 7, 55, 52, 648000000, time.UTC),
										NativeAmount:    0.02,
									},
								},
								ConvertedAmount: 39.0645,
								NativeAmount:    0.021,
							},
							ExitQueue: &EarnAllocationState{
								AllocationCount: 1,
								Allocations: []EarnAllocationTransition{
									{
										ConvertedAmount: 5.5806,
										CreatedAt:       time.Date(2026, 10, 2, 8, 14, 31, 0, time.UTC),
										Expires:         time.Date(2026, 11, 1, 8, 14, 31, 125000000, time.UTC),
										NativeAmount:    0.003,
									},
								},
								ConvertedAmount: 5.5806,
								NativeAmount:    0.003,
							},
							Pending: &EarnAmount{ConvertedAmount: -1.8602, NativeAmount: -0.001},
							Total:   EarnAmount{ConvertedAmount: 49.2398, NativeAmount: 0.0265},
							Unbonding: &EarnAllocationState{
								AllocationCount: 1,
								Allocations: []EarnAllocationTransition{
									{
										ConvertedAmount: 4.6505,
										CreatedAt:       time.Date(2026, 10, 5, 16, 40, 12, 0, time.UTC),
										Expires:         time.Date(2026, 10, 19, 16, 40, 12, 379000000, time.UTC),
										NativeAmount:    0.0025,
									},
								},
								ConvertedAmount: 4.6505,
								NativeAmount:    0.0025,
							},
						},
						NativeAsset: currency.ETH,
						Payout: &EarnPayout{
							AccumulatedReward: EarnAmount{ConvertedAmount: 0.0223, NativeAmount: 0.000012},
							EstimatedReward:   EarnAmount{ConvertedAmount: 0.0632, NativeAmount: 0.000034},
							PeriodEnd:         time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC),
							PeriodStart:       time.Date(2026, 10, 4, 0, 0, 0, 0, time.UTC),
						},
						StrategyID:    "ESDQCOL-WTZEU-NU55QF",
						TotalRewarded: EarnAmount{ConvertedAmount: 0.0675, NativeAmount: 0.0000363},
					},
				},
				TotalAllocated: 49.2398,
				TotalRewarded:  0.0675,
			},
		},
		{
			name: "every allocation",
			exp: &EarnAllocationsResponse{
				ConvertedAsset: currency.USD,
				Items: []EarnAllocation{
					{
						NativeAsset:   currency.DOT,
						StrategyID:    "ESRFUO3-Q62XD-WIOIL7",
						TotalRewarded: EarnAmount{ConvertedAmount: 1.25, NativeAmount: 0.3051},
					},
				},
				TotalRewarded: 1.25,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.ListEarnAllocations(t.Context(), tc.req)
			require.NoError(t, err, "ListEarnAllocations must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "ListEarnAllocations should decode every field")
				return
			}
			assert.NotNil(t, result, "ListEarnAllocations should return allocations")
		})
	}
}

func TestAllocateEarnFunds(t *testing.T) {
	t.Parallel()
	_, err := e.AllocateEarnFunds(t.Context(), "", 4.3)
	require.ErrorIs(t, err, errEarnStrategyIDRequired, "AllocateEarnFunds must reject an empty strategy ID")
	_, err = e.AllocateEarnFunds(t.Context(), "ESRFUO3-Q62XD-WIOIL7", 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "AllocateEarnFunds must reject a zero amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.AllocateEarnFunds(t.Context(), "ESRFUO3-Q62XD-WIOIL7", 4.3)
	require.NoError(t, err, "AllocateEarnFunds must not error")
	assert.True(t, result, "AllocateEarnFunds should report the allocation was dispatched")
}

func TestDeallocateEarnFunds(t *testing.T) {
	t.Parallel()
	_, err := e.DeallocateEarnFunds(t.Context(), "", 1.25)
	require.ErrorIs(t, err, errEarnStrategyIDRequired, "DeallocateEarnFunds must reject an empty strategy ID")
	_, err = e.DeallocateEarnFunds(t.Context(), "ESRFUO3-Q62XD-WIOIL7", 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "DeallocateEarnFunds must reject a zero amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.DeallocateEarnFunds(t.Context(), "ESRFUO3-Q62XD-WIOIL7", 1.25)
	require.NoError(t, err, "DeallocateEarnFunds must not error")
	assert.True(t, result, "DeallocateEarnFunds should report the deallocation was dispatched")
}

func TestGetEarnAllocationStatus(t *testing.T) {
	t.Parallel()
	_, err := e.GetEarnAllocationStatus(t.Context(), "")
	require.ErrorIs(t, err, errEarnStrategyIDRequired, "GetEarnAllocationStatus must reject an empty strategy ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetEarnAllocationStatus(t.Context(), "ESRFUO3-Q62XD-WIOIL7")
	require.NoError(t, err, "GetEarnAllocationStatus must not error")
	if mockTests {
		assert.Equal(t, &EarnAllocationStatusResponse{Pending: true}, result, "GetEarnAllocationStatus should decode every field")
		return
	}
	assert.NotNil(t, result, "GetEarnAllocationStatus should return a status")
}

func TestGetEarnDeallocationStatus(t *testing.T) {
	t.Parallel()
	_, err := e.GetEarnDeallocationStatus(t.Context(), "")
	require.ErrorIs(t, err, errEarnStrategyIDRequired, "GetEarnDeallocationStatus must reject an empty strategy ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetEarnDeallocationStatus(t.Context(), "ESDQCOL-WTZEU-NU55QF")
	require.NoError(t, err, "GetEarnDeallocationStatus must not error")
	if mockTests {
		assert.Equal(t, &EarnDeallocationStatusResponse{Pending: true}, result, "GetEarnDeallocationStatus should decode every field")
		return
	}
	assert.NotNil(t, result, "GetEarnDeallocationStatus should return a status")
}

func TestEarnOperationBody(t *testing.T) {
	t.Parallel()
	_, err := earnOperationBody("", 1)
	assert.ErrorIs(t, err, errEarnStrategyIDRequired, "earnOperationBody should reject an empty strategy ID")
	_, err = earnOperationBody("ESRFUO3-Q62XD-WIOIL7", 0)
	assert.ErrorIs(t, err, limits.ErrAmountBelowMin, "earnOperationBody should reject a zero amount")
	body, err := earnOperationBody("ESRFUO3-Q62XD-WIOIL7", 0.5)
	require.NoError(t, err, "earnOperationBody must not error")
	assert.Equal(t, map[string]any{"amount": "0.5", "strategy_id": "ESRFUO3-Q62XD-WIOIL7"}, body, "earnOperationBody should send the amount and strategy ID")
}
