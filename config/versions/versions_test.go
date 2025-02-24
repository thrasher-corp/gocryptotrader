package versions

import (
	"context"
	"testing"

	"github.com/buger/jsonparser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
)

func TestDeploy(t *testing.T) {
	t.Parallel()
	m := manager{}
	_, err := m.Deploy(context.Background(), []byte(``), LatestVersion)
	assert.ErrorIs(t, err, errNoVersions)

	m.registerVersion(1, &TestVersion1{})
	_, err = m.Deploy(context.Background(), []byte(``), LatestVersion)
	require.ErrorIs(t, err, errVersionIncompatible)

	m = manager{}

	m.registerVersion(0, &Version0{})
	_, err = m.Deploy(context.Background(), []byte(`not an object`), LatestVersion)
	require.ErrorIs(t, err, jsonparser.KeyPathNotFoundError, "Must throw the correct error trying to add version to bad json")
	require.ErrorIs(t, err, common.ErrSettingField, "Must throw the correct error trying to add version to bad json")
	require.ErrorContains(t, err, "version", "Must throw the correct error trying to add version to bad json")

	_, err = m.Deploy(context.Background(), []byte(`{"version":"not an int"}`), LatestVersion)
	require.ErrorIs(t, err, common.ErrGettingField, "Must throw the correct error trying to get version from bad json")

	in := []byte(`{"version":0,"exchanges":[{"name":"Juan"}]}`)
	j, err := m.Deploy(context.Background(), in, LatestVersion)
	require.NoError(t, err)
	assert.Equal(t, string(in), string(j))

	m.registerVersion(1, &Version1{})
	j, err = m.Deploy(context.Background(), in, LatestVersion)
	require.NoError(t, err)
	assert.Contains(t, string(j), `"version": 1`)

	_, err = m.Deploy(context.Background(), j, 2)
	assert.ErrorIs(t, err, errTargetVersion, "Downgrade to a unregistered version should not be allowed")

	m.versions = append(m.versions, &TestVersion2{ConfigErr: true, ExchErr: false})
	_, err = m.Deploy(context.Background(), j, LatestVersion)
	require.ErrorIs(t, err, errUpgrade)

	m.versions[len(m.versions)-1] = &TestVersion2{ConfigErr: false, ExchErr: true}
	_, err = m.Deploy(context.Background(), in, LatestVersion)
	require.Implements(t, (*ExchangeVersion)(nil), m.versions[1])
	require.ErrorIs(t, err, errUpgrade)

	j2, err := m.Deploy(context.Background(), j, 0)
	require.NoError(t, err)
	assert.Contains(t, string(j2), `"version": 0`, "Explicit downgrade should work correctly")

	m.versions = m.versions[:1]
	_, err = m.Deploy(context.Background(), j, LatestVersion)
	assert.ErrorIs(t, err, errConfigVersion, "Config version ahead of latest version should error")

	_, err = m.Deploy(context.Background(), j, 0)
	assert.ErrorIs(t, err, errConfigVersion, "Config version ahead of latest version should error")
}

// TestExchangeDeploy exercises exchangeDeploy
// There are a number of error paths we can't currently cover without exposing unacceptable risks to the hot-paths as well
func TestExchangeDeploy(t *testing.T) {
	t.Parallel()
	m := manager{}
	_, err := m.Deploy(context.Background(), []byte(``), LatestVersion)
	assert.ErrorIs(t, err, errNoVersions)

	v := &TestVersion2{}
	in := []byte(`{"version":0,"exchanges":[{}]}`)
	_, err = exchangeDeploy(context.Background(), v, ExchangeVersion.UpgradeExchange, in)
	require.ErrorIs(t, err, errModifyingExchange)
	require.ErrorIs(t, err, common.ErrGettingField)
	require.ErrorIs(t, err, jsonparser.KeyPathNotFoundError)
	require.ErrorContains(t, err, "`name`")

	in = []byte(`{"version":0,"exchanges":[{"name":"Juan"},{"name":"Megashaft"}]}`)
	_, err = exchangeDeploy(context.Background(), v, ExchangeVersion.UpgradeExchange, in)
	require.NoError(t, err)
}

func TestRegisterVersion(t *testing.T) {
	t.Parallel()
	m := manager{}

	m.registerVersion(0, &Version0{})
	assert.NotEmpty(t, m.versions)

	m.registerVersion(2, &TestVersion2{})
	require.Equal(t, 3, len(m.versions), "Must allocate a space for missing version 1")
	require.NotNil(t, m.versions[2], "Must put Version 2 in the correct slot")
	require.Nil(t, m.versions[1], "Must leave Version 1 alone")

	m.registerVersion(1, &TestVersion1{})
	require.Equal(t, 3, len(m.versions), "Must leave len alone when registering out-of-sequence")
	require.NotNil(t, m.versions[1], "Must put Version 1 in the correct slot")
}

func TestLatest(t *testing.T) {
	t.Parallel()
	m := manager{}
	_, err := m.latest()
	require.ErrorIs(t, err, errNoVersions)

	m.registerVersion(0, &Version0{})
	m.registerVersion(1, &Version1{})
	v, err := m.latest()
	require.NoError(t, err)
	assert.Equal(t, 1, v)

	m.registerVersion(2, &Version2{})
	v, err = m.latest()
	require.NoError(t, err)
	assert.Equal(t, 2, v)
}
