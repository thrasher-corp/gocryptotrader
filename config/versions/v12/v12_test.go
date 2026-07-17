package v12_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v12 "github.com/thrasher-corp/gocryptotrader/config/versions/v12"
)

func TestExchanges(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []string{"Huobi", "HTX"}, new(v12.Version).Exchanges(), "Exchanges should return migrated exchange names")
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
			out, err := new(v12.Version).UpgradeExchange(t.Context(), []byte(`{"name":"`+tt.in+`"}`))
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
			out, err := new(v12.Version).DowngradeExchange(t.Context(), []byte(`{"name":"`+tt.in+`"}`))
			require.NoError(t, err, "DowngradeExchange must not error")
			require.NotEmpty(t, out, "DowngradeExchange must return output")
			assert.Equalf(t, `{"name":"`+tt.want+`"}`, string(out), "exchange name %s should migrate correctly", tt.in)
		})
	}
}
