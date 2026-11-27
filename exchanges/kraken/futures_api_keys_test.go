package kraken

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

func TestCheckFuturesAPIKey(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.CheckFuturesAPIKey(t.Context())
	require.NoError(t, err, "CheckFuturesAPIKey must not error")
	if mockTests {
		exp := &FuturesCheckAPIKeyResponse{
			APIKey:            "mock-api-key",
			AccountUID:        futuresTestMasterAccountUID,
			IIBAN:             "AA08 N84G CPAR UN7A",
			CreatedAt:         time.Date(2019, 8, 24, 14, 15, 22, 0, time.UTC),
			Permissions:       FuturesAPIKeyPermissions{General: "FULL_ACCESS", Transfer: "READ_ONLY"},
			AllowedCIDRBlock:  "192.168.0.0/16",
			AllowedCIDRBlocks: []string{"192.168.0.0/16", "10.20.0.0/24"},
		}
		assert.Equal(t, exp, result, "CheckFuturesAPIKey should decode every field")
		return
	}
	// The key is a credential, so the test checks the account rather than printing the key in a failure
	assert.NotEmpty(t, result.AccountUID, "CheckFuturesAPIKey should return the key's account")
}
