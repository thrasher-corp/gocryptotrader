package versions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v0 "github.com/thrasher-corp/gocryptotrader/config/versions/v0"
	v1 "github.com/thrasher-corp/gocryptotrader/config/versions/v1"
	v10 "github.com/thrasher-corp/gocryptotrader/config/versions/v10"
	v11 "github.com/thrasher-corp/gocryptotrader/config/versions/v11"
	v12 "github.com/thrasher-corp/gocryptotrader/config/versions/v12"
	v13 "github.com/thrasher-corp/gocryptotrader/config/versions/v13"
	v2 "github.com/thrasher-corp/gocryptotrader/config/versions/v2"
	v3 "github.com/thrasher-corp/gocryptotrader/config/versions/v3"
	v4 "github.com/thrasher-corp/gocryptotrader/config/versions/v4"
	v5 "github.com/thrasher-corp/gocryptotrader/config/versions/v5"
	v6 "github.com/thrasher-corp/gocryptotrader/config/versions/v6"
	v7 "github.com/thrasher-corp/gocryptotrader/config/versions/v7"
	v8 "github.com/thrasher-corp/gocryptotrader/config/versions/v8"
	v9 "github.com/thrasher-corp/gocryptotrader/config/versions/v9"
)

func TestNewManager(t *testing.T) {
	t.Parallel()

	m := newManager()
	for _, tc := range []struct {
		version  uint16
		expected any
	}{
		{version: 0, expected: new(v0.Version)},
		{version: 1, expected: new(v1.Version)},
		{version: 2, expected: new(v2.Version)},
		{version: 3, expected: new(v3.Version)},
		{version: 4, expected: new(v4.Version)},
		{version: 5, expected: new(v5.Version)},
		{version: 6, expected: new(v6.Version)},
		{version: 7, expected: new(v7.Version)},
		{version: 8, expected: new(v8.Version)},
		{version: 9, expected: new(v9.Version)},
		{version: 10, expected: new(v10.Version)},
		{version: 11, expected: new(v11.Version)},
		{version: 12, expected: new(v12.Version)},
		{version: 13, expected: new(v13.Version)},
	} {
		assert.IsTypef(t, tc.expected, m.Version(tc.version), "Version %d should use the expected implementation", tc.version)
	}
	assert.Nil(t, m.Version(14), "Unregistered version should not be returned")

	latest, err := m.latest()
	require.NoError(t, err, "Latest version lookup must not error")
	assert.Equal(t, uint16(13), latest, "Latest version should be 13")
}
