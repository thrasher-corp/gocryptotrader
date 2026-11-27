package bybit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

func TestSetAPIRateLimit(t *testing.T) {
	t.Parallel()
	_, err := e.SetAPIRateLimit(t.Context(), nil)
	require.ErrorIs(t, err, errNoAPIRateLimits, "SetAPIRateLimit must reject no limits")
	_, err = e.SetAPIRateLimit(t.Context(), []SetAPIRateLimitRequest{{BusinessType: "SPOT", Rate: 600}})
	require.ErrorIs(t, err, errMembersIDsNotSet, "SetAPIRateLimit must reject missing UIDs")
	_, err = e.SetAPIRateLimit(t.Context(), []SetAPIRateLimitRequest{{UIDs: []string{"100000002"}, Rate: 600}})
	require.ErrorIs(t, err, errBusinessTypeRequired, "SetAPIRateLimit must reject a missing business type")
	_, err = e.SetAPIRateLimit(t.Context(), []SetAPIRateLimitRequest{{UIDs: []string{"100000002"}, BusinessType: "SPOT"}})
	require.ErrorIs(t, err, errRateRequired, "SetAPIRateLimit must reject a missing rate")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SetAPIRateLimit(t.Context(), []SetAPIRateLimitRequest{
		{UIDs: []string{"100000002", "100000003"}, BusinessType: "SPOT", Rate: 600},
		{UIDs: []string{"100000004"}, BusinessType: "DERIVATIVES", Rate: 2000},
	})
	require.NoError(t, err, "SetAPIRateLimit must not error")
	if mockTests {
		exp := &SetAPIRateLimitResponse{
			Result: []SetAPIRateLimitResult{
				{
					UIDs:         "100000002,100000003",
					BusinessType: "SPOT",
					Rate:         600,
					Success:      true,
					Message:      "API limit updated successfully",
				},
				{
					UIDs:         "100000004",
					BusinessType: "DERIVATIVES",
					Rate:         2000,
					Message:      "Requested limit exceeds maximum allowed per user",
				},
			},
		}
		assert.Equal(t, exp, result, "SetAPIRateLimit should decode every field")
		return
	}
	assert.Len(t, result.Result, 2, "SetAPIRateLimit should return each limit's outcome")
}

func TestGetAPIRateLimit(t *testing.T) {
	t.Parallel()
	_, err := e.GetAPIRateLimit(t.Context())
	require.ErrorIs(t, err, errMembersIDsNotSet, "GetAPIRateLimit must reject missing UIDs")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAPIRateLimit(t.Context(), "100000002")
	require.NoError(t, err, "GetAPIRateLimit must not error")
	if mockTests {
		exp := &APIRateLimitResponse{
			List: []APIRateLimit{
				{
					UIDs:         "100000002",
					BusinessType: "SPOT",
					Rate:         600,
				},
				{
					UIDs:         "100000002",
					BusinessType: "DERIVATIVES",
					Rate:         400,
				},
			},
		}
		assert.Equal(t, exp, result, "GetAPIRateLimit should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAPIRateLimit should return a result")
}

func TestGetAPIRateLimitCap(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAPIRateLimitCap(t.Context())
	require.NoError(t, err, "GetAPIRateLimitCap must not error")
	if mockTests {
		exp := &APIRateLimitCapResponse{
			List: []APIRateLimitCap{
				{
					BusinessType:     "SPOT",
					TotalRate:        29882,
					InstitutionalCap: 30000,
					UIDCap:           600,
				},
				{
					BusinessType:     "OPTIONS",
					TotalRate:        29882,
					InstitutionalCap: 30000,
					UIDCap:           600,
				},
				{
					BusinessType:     "DERIVATIVES",
					TotalRate:        39932,
					InstitutionalCap: 40000,
					UIDCap:           800,
				},
			},
		}
		assert.Equal(t, exp, result, "GetAPIRateLimitCap should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAPIRateLimitCap should return a result")
}

func TestGetAllAPIRateLimits(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllAPIRateLimits(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAllAPIRateLimits must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAllAPIRateLimits(t.Context(), &AllAPIRateLimitsRequest{Limit: 2, Cursor: "MTAwMDAwMDAx", UIDs: []string{"100000002", "100000005"}})
	require.NoError(t, err, "GetAllAPIRateLimits must not error")
	if mockTests {
		exp := &AllAPIRateLimitsResponse{
			NextPageCursor: "MTAwMDAwMDA1",
			List: []APIRateLimit{
				{
					UIDs:         "100000002",
					BusinessType: "SPOT",
					Rate:         223,
				},
				{
					UIDs:         "100000005",
					BusinessType: "OPTIONS",
					Rate:         298,
				},
			},
		}
		assert.Equal(t, exp, result, "GetAllAPIRateLimits should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAllAPIRateLimits should return a result")
}
