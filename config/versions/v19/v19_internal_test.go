package v19

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenameExchangeConflict(t *testing.T) {
	t.Parallel()
	// Neither exchange config can be dropped without losing a user's settings, so the version must not advance
	in := `{"version":18,"exchanges":[{"name":"GateIO","enabled":true},{"name":"gate","enabled":false}]}`
	out, err := new(Version).UpgradeConfig(t.Context(), []byte(in))
	assert.ErrorIs(t, err, errExchangeNameConflict, "UpgradeConfig should refuse a config holding both GateIO and Gate")
	assert.JSONEq(t, in, string(out), "UpgradeConfig should leave a conflicting config unchanged")

	in = `{"version":19,"exchanges":[{"name":"Gate"},{"name":"GATEIO"}]}`
	out, err = new(Version).DowngradeConfig(t.Context(), []byte(in))
	assert.ErrorIs(t, err, errExchangeNameConflict, "DowngradeConfig should refuse a config holding both Gate and GateIO")
	assert.JSONEq(t, in, string(out), "DowngradeConfig should leave a conflicting config unchanged")
}
