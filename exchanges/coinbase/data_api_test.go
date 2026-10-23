package coinbase

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

func TestGetAPIKeyPermissions(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	resp, err := e.GetAPIKeyPermissions(t.Context())
	require.NoError(t, err, "GetAPIKeyPermissions must not error")
	if mockTests {
		exp := &APIKeyPermissionsResponse{
			CanView:       true,
			CanTrade:      true,
			PortfolioUUID: "b87a2d3f-8a1e-49b3-a4ea-402d8c389aca",
			PortfolioType: "DEFAULT",
		}
		assert.Equal(t, exp, resp, "GetAPIKeyPermissions should decode every field")
	} else {
		assert.NotEmpty(t, resp.PortfolioUUID, "GetAPIKeyPermissions should return the key's portfolio")
	}

	_, err = newJWTCheckingServerExchange(t, `null`).GetAPIKeyPermissions(t.Context())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetAPIKeyPermissions should error for a null response")
	_, err = newUnavailableServerExchange(t).GetAPIKeyPermissions(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetAPIKeyPermissions should return the request failure")
}
