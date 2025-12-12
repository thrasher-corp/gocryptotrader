//go:build config_versions

// This test is run independently from CI for developer convenience when developing out-of-sequence versions
// Called from a separate github workflow to prevent a PR from being merged without failing the main unit tests

package versions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersionContinuity(t *testing.T) {
	t.Parallel()
	for ver, v := range Manager.versions {
		assert.NotNilf(t, v, "Version %d should not be empty", ver)
	}
}
