package bitstamp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRevokeAllAPIKeys(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("RevokeAllAPIKeys is never tested live as it revokes every API key")
	}
	resp, err := e.RevokeAllAPIKeys(t.Context())
	require.NoError(t, err, "RevokeAllAPIKeys must not error")
	exp := &RevokeAPIKeysResponse{RevokedAPIKeys: []string{"rkBWB2SMjCyU4lJKQvjRNwszMr11kSRU", "beXTolq77X3GU70rpmBNjwZw3mwpuiSs"}}
	assert.Equal(t, exp, resp, "RevokeAllAPIKeys should return the revoked keys")
}
