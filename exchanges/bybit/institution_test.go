package bybit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

func TestGetInstitutionWhitelistIP(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetInstitutionWhitelistIP(t.Context())
	require.NoError(t, err, "GetInstitutionWhitelistIP must not error")
	if mockTests {
		exp := []InstitutionWhitelistIPResponse{
			{
				Name:           "Example Client",
				EndpointDomain: "gw-client.example.com",
				IPList: []string{
					"192.0.2.1",
					"198.51.100.1",
				},
			},
			{
				Name:           "Example Client",
				EndpointDomain: "ws-client.example.com",
				IPList: []string{
					"192.0.2.1",
					"198.51.100.1",
					"198.51.100.2",
				},
			},
		}
		assert.Equal(t, exp, result, "GetInstitutionWhitelistIP should decode every field")
		return
	}
	assert.NotNil(t, result, "GetInstitutionWhitelistIP should return a result")
}

func TestGetInstitutionIPChangeLog(t *testing.T) {
	t.Parallel()
	_, err := e.GetInstitutionIPChangeLog(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetInstitutionIPChangeLog must reject a nil request")
	_, err = e.GetInstitutionIPChangeLog(t.Context(), &InstitutionIPChangeLogRequest{StartTime: time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC), EndTime: time.Date(2026, 5, 21, 0, 0, 0, 0, time.UTC)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetInstitutionIPChangeLog must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetInstitutionIPChangeLog(t.Context(), &InstitutionIPChangeLogRequest{
		StartTime: time.Date(2026, 5, 21, 0, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC),
		Limit:     3,
		Cursor:    "eyJpZCI6NDA0LCJjcmVhdGVkQXQiOiIyMDI2LTA1LTIxVDA3OjAwOjAwIn0=",
	})
	require.NoError(t, err, "GetInstitutionIPChangeLog must not error")
	if mockTests {
		exp := &InstitutionIPChangeLogResponse{
			List: []InstitutionIPChange{
				{
					OperationType: "DISABLE",
					IPAddress:     "192.0.2.10",
					DomainType:    "WS",
					ChangeReason:  "do not meet the requirement",
					CreatedAt:     time.Date(2026, 5, 21, 6, 2, 36, 0, time.UTC),
				},
				{
					OperationType: "ADD",
					IPAddress:     "198.51.100.0/25",
					DomainType:    "GW",
					ChangeReason:  "client request",
					CreatedAt:     time.Date(2026, 5, 21, 5, 30, 0, 0, time.UTC),
				},
			},
			NextPageCursor: "eyJpZCI6NDE0LCJjcmVhdGVkQXQiOiIyMDI2LTA1LTIxVDA2OjAyOjM2In0=",
		}
		assert.Equal(t, exp, result, "GetInstitutionIPChangeLog should decode every field")
		return
	}
	assert.NotNil(t, result, "GetInstitutionIPChangeLog should return a result")
}
