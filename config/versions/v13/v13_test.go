package v13_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config/versions"
	v13 "github.com/thrasher-corp/gocryptotrader/config/versions/v13"
)

func TestExchanges(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []string{"Huobi", "HTX"}, new(v13.Version).Exchanges(), "Exchanges should return migrated exchange names")
}

func TestUpgradeExchange(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		in   string
		want string
	}{
		{name: "legacy", in: "Huobi", want: "HTX"},
		{name: "unrelated", in: "Kraken", want: "Kraken"},
		{name: "current", in: "HTX", want: "HTX"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			out, err := new(v13.Version).UpgradeExchange(t.Context(), []byte(`{"name":"`+tt.in+`"}`))
			require.NoError(t, err, "UpgradeExchange must not error")
			require.NotEmpty(t, out, "UpgradeExchange must return output")
			assert.Equalf(t, `{"name":"`+tt.want+`"}`, string(out), "exchange name %s should migrate correctly", tt.in)
		})
	}
}

func TestDowngradeExchange(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		in   string
		want string
	}{
		{name: "current", in: "HTX", want: "Huobi"},
		{name: "unrelated", in: "Kraken", want: "Kraken"},
		{name: "legacy", in: "Huobi", want: "Huobi"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			out, err := new(v13.Version).DowngradeExchange(t.Context(), []byte(`{"name":"`+tt.in+`"}`))
			require.NoError(t, err, "DowngradeExchange must not error")
			require.NotEmpty(t, out, "DowngradeExchange must return output")
			assert.Equalf(t, `{"name":"`+tt.want+`"}`, string(out), "exchange name %s should migrate correctly", tt.in)
		})
	}
}

func TestRegisteredUpgrade(t *testing.T) {
	t.Parallel()
	input := []byte(`{"version":11,"exchanges":[{"name":"EXMO"},{"name":"Huobi","enabled":true},{"name":"Kraken","enabled":true}]}`)
	out, err := versions.Manager.Deploy(t.Context(), input, 13)
	require.NoError(t, err, "Deploy must apply the registered v12 and v13 upgrades")
	assert.JSONEq(t, `{"version":13,"exchanges":[{"name":"HTX","enabled":true},{"name":"Kraken","enabled":true}]}`, string(out), "Deploy should remove EXMO before renaming Huobi to HTX")
}
