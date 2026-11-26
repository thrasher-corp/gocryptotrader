package bitstamp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRevokeAllAPIKeys(t *testing.T) {
	t.Parallel()
	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("RevokeAllAPIKeys is never sent live with real credentials, as the API would revoke every API key")
		}
		resp, err := e.RevokeAllAPIKeys(t.Context())
		require.NoError(t, err, "RevokeAllAPIKeys must not error")
		exp := &RevokeAPIKeysResponse{RevokedAPIKeys: []string{"rkBWB2SMjCyU4lJKQvjRNwszMr11kSRU", "beXTolq77X3GU70rpmBNjwZw3mwpuiSs"}}
		assert.Equal(t, exp, resp, "RevokeAllAPIKeys should return the revoked keys")
	})
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("The placeholder API key is refused only by the live API")
		}
		_, err := newPlaceholderCredentialsTestInstance(t).RevokeAllAPIKeys(t.Context())
		requireUnknownAPIKey(t, err)
	})
}
