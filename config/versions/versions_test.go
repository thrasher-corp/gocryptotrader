package versions

import (
	"context"
	"testing"

	"github.com/buger/jsonparser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	v9997 "github.com/thrasher-corp/gocryptotrader/config/versions/testfixtures/v9997"
	v9998 "github.com/thrasher-corp/gocryptotrader/config/versions/testfixtures/v9998"
	v9999 "github.com/thrasher-corp/gocryptotrader/config/versions/testfixtures/v9999"
	v0 "github.com/thrasher-corp/gocryptotrader/config/versions/v0"
	v1 "github.com/thrasher-corp/gocryptotrader/config/versions/v1"
)

func TestDeploy(t *testing.T) {
	t.Parallel()
	m := manager{}
	_, err := m.Deploy(context.Background(), []byte(``))
	assert.ErrorIs(t, err, errNoVersions)

	m.registerVersion(&v9999.Version{})
	_, err = m.Deploy(context.Background(), []byte(``))
	require.ErrorIs(t, err, errVersionIncompatible)

	m.errors = nil
	m.registerVersion(&v0.Version{})
	_, err = m.Deploy(context.Background(), []byte(`not an object`))
	require.ErrorIs(t, err, jsonparser.KeyPathNotFoundError, "Must throw the correct error trying to add version to bad json")
	require.ErrorIs(t, err, common.ErrSettingField, "Must throw the correct error trying to add version to bad json")
	require.ErrorContains(t, err, "version", "Must throw the correct error trying to add version to bad json")

	_, err = m.Deploy(context.Background(), []byte(`{"version":"not an int"}`))
	require.ErrorIs(t, err, common.ErrGettingField, "Must throw the correct error trying to get version from bad json")

	in := []byte(`{"version":0,"exchanges":[{"name":"Juan"}]}`)
	j, err := m.Deploy(context.Background(), in)
	require.NoError(t, err)
	require.Equal(t, string(in), string(j))

	m.registerVersion(&v1.Version{})
	j, err = m.Deploy(context.Background(), in)
	require.NoError(t, err)
	require.Contains(t, string(j), `"version":1`)

	m.versions = m.versions[:1]
	j, err = m.Deploy(context.Background(), j)
	require.NoError(t, err)
	require.Contains(t, string(j), `"version":0`)

	m.versions = append(m.versions, &v9998.Version{ConfigErr: true, ExchErr: false}) // Bit hacky, but this will actually work
	_, err = m.Deploy(context.Background(), j)
	require.ErrorIs(t, err, v9998.ErrUpgrade)

	m.versions[1] = &v9998.Version{ConfigErr: false, ExchErr: true}
	_, err = m.Deploy(context.Background(), in)
	require.Implements(t, (*ExchangeVersion)(nil), m.versions[1])
	require.ErrorIs(t, err, v9998.ErrUpgrade)
}

// TestExchangeDeploy exercises exchangeDeploy
// There are a number of error paths we can't currently cover without exposing unacceptable risks to the hot-paths as well
func TestExchangeDeploy(t *testing.T) {
	t.Parallel()
	m := manager{}
	_, err := m.Deploy(context.Background(), []byte(``))
	assert.ErrorIs(t, err, errNoVersions)

	v := &v9998.Version{}
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

	m.registerVersion(&v0.Version{})
	require.NoError(t, m.errors)
	assert.NotEmpty(t, m.versions)

	m.registerVersion("cheese string")
	require.ErrorIs(t, m.errors, errRegisteringVersion)

	m.errors = nil
	m.registerVersion(&v9999.Version{})
	require.ErrorIs(t, m.errors, errVersionIncompatible)
	assert.ErrorContains(t, m.errors, "9999")

	m.errors = nil
	m.registerVersion(&v9998.Version{})
	assert.ErrorIs(t, m.errors, errVersionSequence)
	assert.ErrorContains(t, m.errors, "9998")

	m.errors = nil
	m.registerVersion(&v9997.Version{})
	assert.NoError(t, m.errors)
	assert.Len(t, m.versions, 1, "Disabled Versions should not be registered")
}

func TestLatest(t *testing.T) {
	t.Parallel()
	m := manager{}
	_, err := m.latest()
	require.ErrorIs(t, err, errNoVersions)

	m.registerVersion(&v0.Version{})
	m.registerVersion(&v1.Version{})
	v, err := m.latest()
	require.NoError(t, err)
	assert.Equal(t, 1, v)
}
