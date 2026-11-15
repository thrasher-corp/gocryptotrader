package v20_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config/versions"
)

func TestDeploy(t *testing.T) {
	t.Parallel()
	in := []byte(`{"version":19,"exchanges":[{"name":"GateIO","api":{"urlEndpoints":{"RestFuturesURL":"https://proxy.example/gate-fx"}},"bankAccounts":[{"supportedExchanges":"GateIO"}]},{"name":"Kraken"}],"bankAccounts":[{"supportedExchanges":"Kraken,GateIO"}],"portfolioAddresses":{"addresses":[{"Address":"GateIO","CoinType":"BTC","Balance":1,"Description":"Exchange"},{"Address":"1abc","CoinType":"BTC","Description":"Personal","SupportedExchanges":"GateIO,Kraken"}]}}`)
	out, err := versions.Manager.Deploy(t.Context(), in, 20)
	require.NoError(t, err, "Deploy must not error upgrading to version 20")
	assert.JSONEq(t, `{"version":20,"exchanges":[{"name":"Gate","api":{"urlEndpoints":{"RestFuturesURL":"https://proxy.example/gate-fx"}},"bankAccounts":[{"supportedExchanges":"Gate"}]},{"name":"Kraken"}],"bankAccounts":[{"supportedExchanges":"Kraken,Gate"}],"portfolioAddresses":{"addresses":[{"Address":"Gate","CoinType":"BTC","Balance":1,"Description":"Exchange"},{"Address":"1abc","CoinType":"BTC","Description":"Personal","SupportedExchanges":"Gate,Kraken"}]}}`, string(out), "Deploy should rename GateIO to Gate")

	out, err = versions.Manager.Deploy(t.Context(), out, 19)
	require.NoError(t, err, "Deploy must not error downgrading to version 19")
	assert.JSONEq(t, string(in), string(out), "Deploy should restore the version 19 config")
}
