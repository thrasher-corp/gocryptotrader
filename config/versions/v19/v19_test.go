package v19_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config/versions"
	v19 "github.com/thrasher-corp/gocryptotrader/config/versions/v19"
)

func TestUpgradeConfig(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, input, expected string
	}{
		{name: "missing exchanges", input: `{"name":"gocryptotrader"}`, expected: `{"name":"gocryptotrader"}`},
		{name: "non-array exchanges", input: `{"exchanges":null}`, expected: `{"exchanges":null}`},
		{name: "no GateIO", input: `{"exchanges":[{"name":"Kraken"},{"name":"GateIOX"}]}`, expected: `{"exchanges":[{"name":"Kraken"},{"name":"GateIOX"}]}`},
		{
			name:     "GateIO renamed with its settings kept",
			input:    `{"exchanges":[{"name":"Kraken","enabled":true},{"name":"GateIO","enabled":true,"api":{"urlEndpoints":{"RestSpotURL":"https://api.gateio.ws"}}}]}`,
			expected: `{"exchanges":[{"name":"Kraken","enabled":true},{"name":"Gate","enabled":true,"api":{"urlEndpoints":{"RestSpotURL":"https://api.gateio.ws"}}}]}`,
		},
		{name: "lower case", input: `{"exchanges":[{"name":"gateio"}]}`, expected: `{"exchanges":[{"name":"Gate"}]}`},
		{name: "upper case", input: `{"exchanges":[{"name":"GATEIO"}]}`, expected: `{"exchanges":[{"name":"Gate"}]}`},
		{name: "unnamed exchange", input: `{"exchanges":[{"enabled":false},{"name":"GateIO"}]}`, expected: `{"exchanges":[{"enabled":false},{"name":"Gate"}]}`},
		{
			name:     "exchange bank accounts",
			input:    `{"exchanges":[{"name":"GateIO","bankAccounts":[{"supportedExchanges":"GateIO"}]},{"name":"Kraken","bankAccounts":[{"supportedExchanges":"Kraken,GATEIO"},{"supportedExchanges":"ALL"}]}]}`,
			expected: `{"exchanges":[{"name":"Gate","bankAccounts":[{"supportedExchanges":"Gate"}]},{"name":"Kraken","bankAccounts":[{"supportedExchanges":"Kraken,Gate"},{"supportedExchanges":"ALL"}]}]}`,
		},
		{
			name:     "client bank accounts",
			input:    `{"bankAccounts":[{"bankName":"a","supportedExchanges":"Kraken, gateio ,Bitstamp"},{"supportedExchanges":"GateIOX,Gateway"},{"supportedExchanges":null},{"bankName":"no exchange list"}]}`,
			expected: `{"bankAccounts":[{"bankName":"a","supportedExchanges":"Kraken, Gate ,Bitstamp"},{"supportedExchanges":"GateIOX,Gateway"},{"supportedExchanges":null},{"bankName":"no exchange list"}]}`,
		},
		{
			// The portfolio names an exchange balance after the exchange exactly, so a balance under another case was
			// never the exchange's and stays as it is
			name:     "portfolio addresses",
			input:    `{"portfolioAddresses":{"addresses":[{"Address":"GateIO","CoinType":"BTC","Balance":1,"Description":"Exchange"},{"Address":"gateio","CoinType":"ETH","Description":"Exchange"},{"Address":"1abc","CoinType":"BTC","Description":"Personal","SupportedExchanges":"Kraken,GateIO"},{"Address":"GateIO","CoinType":"LTC","Description":"Personal"}],"verbose":false}}`,
			expected: `{"portfolioAddresses":{"addresses":[{"Address":"Gate","CoinType":"BTC","Balance":1,"Description":"Exchange"},{"Address":"gateio","CoinType":"ETH","Description":"Exchange"},{"Address":"1abc","CoinType":"BTC","Description":"Personal","SupportedExchanges":"Kraken,Gate"},{"Address":"GateIO","CoinType":"LTC","Description":"Personal"}],"verbose":false}}`,
		},
		{
			name:     "portfolio balance already held under the new name",
			input:    `{"portfolioAddresses":{"addresses":[{"Address":"GateIO","CoinType":"BTC","Balance":1,"Description":"Exchange"},{"Address":"Gate","CoinType":"btc","Balance":2,"Description":"Exchange"},{"Address":"GateIO","CoinType":"ETH","Balance":3,"Description":"Exchange"}]}}`,
			expected: `{"portfolioAddresses":{"addresses":[{"Address":"Gate","CoinType":"btc","Balance":2,"Description":"Exchange"},{"Address":"Gate","CoinType":"ETH","Balance":3,"Description":"Exchange"}]}}`,
		},
		{
			name:     "portfolio without addresses",
			input:    `{"portfolioAddresses":{"addresses":null,"verbose":true}}`,
			expected: `{"portfolioAddresses":{"addresses":null,"verbose":true}}`,
		},
		{
			// The config loader reads keys in any case, so a hand-written config may spell them otherwise
			name:     "keys in another case",
			input:    `{"Exchanges":[{"Name":"GateIO","BankAccounts":[{"SupportedExchanges":"GateIO"}]}],"BANKACCOUNTS":[{"supportedexchanges":"Kraken,GateIO"}],"PortfolioAddresses":{"Addresses":[{"address":"GateIO","coinType":"BTC","description":"Exchange"},{"address":"1abc","description":"Personal","supportedExchanges":"GateIO"}]}}`,
			expected: `{"Exchanges":[{"Name":"Gate","BankAccounts":[{"SupportedExchanges":"Gate"}]}],"BANKACCOUNTS":[{"supportedexchanges":"Kraken,Gate"}],"PortfolioAddresses":{"Addresses":[{"address":"Gate","coinType":"BTC","description":"Exchange"},{"address":"1abc","description":"Personal","supportedExchanges":"Gate"}]}}`,
		},
		{
			// The loader keeps the last of several keys matching in any case
			name:     "duplicate keys in different cases",
			input:    `{"exchanges":[{"name":"Kraken","Name":"GateIO"}]}`,
			expected: `{"exchanges":[{"name":"Kraken","Name":"Gate"}]}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out, err := new(v19.Version).UpgradeConfig(t.Context(), []byte(tc.input))
			require.NoError(t, err, "UpgradeConfig must not error")
			assert.JSONEq(t, tc.expected, string(out), "UpgradeConfig should rename only GateIO")
		})
	}

	for _, input := range []string{
		`{"exchanges":[{"name":42}]}`,
		`{"exchanges":[`,
		`{"bankAccounts":[{"supportedExchanges":"GateIO"}`,
		`{"portfolioAddresses":{"addresses":[{"Address":"GateIO","Description":"Exchange"}`,
		`{"portfolioAddresses":{"addresses":[{"Address":"GateIO","Description":"Exchange","SupportedExchanges":"\u"}]}}`,
	} {
		_, err := new(v19.Version).UpgradeConfig(t.Context(), []byte(input))
		assert.Errorf(t, err, "UpgradeConfig should error on %s", input)
	}
}

func TestDowngradeConfig(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, input, expected string
	}{
		{name: "missing exchanges", input: `{"name":"gocryptotrader"}`, expected: `{"name":"gocryptotrader"}`},
		{name: "Gate renamed", input: `{"exchanges":[{"name":"Gate","enabled":true},{"name":"Kraken"}]}`, expected: `{"exchanges":[{"name":"GateIO","enabled":true},{"name":"Kraken"}]}`},
		{name: "lower case", input: `{"exchanges":[{"name":"gate"}]}`, expected: `{"exchanges":[{"name":"GateIO"}]}`},
		{name: "no Gate", input: `{"exchanges":[{"name":"Gateway"}]}`, expected: `{"exchanges":[{"name":"Gateway"}]}`},
		{
			name:     "bank accounts",
			input:    `{"bankAccounts":[{"supportedExchanges":"Kraken,gate"},{"supportedExchanges":"Gateway"}],"exchanges":[{"name":"Gate","bankAccounts":[{"supportedExchanges":"Gate"}]}]}`,
			expected: `{"bankAccounts":[{"supportedExchanges":"Kraken,GateIO"},{"supportedExchanges":"Gateway"}],"exchanges":[{"name":"GateIO","bankAccounts":[{"supportedExchanges":"GateIO"}]}]}`,
		},
		{
			name:     "portfolio addresses",
			input:    `{"portfolioAddresses":{"addresses":[{"Address":"Gate","CoinType":"BTC","Balance":1,"Description":"Exchange"},{"Address":"GateIO","CoinType":"ETH","Balance":2,"Description":"Exchange"},{"Address":"Gate","CoinType":"ETH","Balance":3,"Description":"Exchange"},{"Address":"1abc","CoinType":"BTC","Description":"Personal","SupportedExchanges":"Gate"}]}}`,
			expected: `{"portfolioAddresses":{"addresses":[{"Address":"GateIO","CoinType":"BTC","Balance":1,"Description":"Exchange"},{"Address":"GateIO","CoinType":"ETH","Balance":2,"Description":"Exchange"},{"Address":"1abc","CoinType":"BTC","Description":"Personal","SupportedExchanges":"GateIO"}]}}`,
		},
		{
			name:     "keys in another case",
			input:    `{"EXCHANGES":[{"NAME":"gate"}],"portfolioaddresses":{"ADDRESSES":[{"ADDRESS":"Gate","DESCRIPTION":"Exchange","COINTYPE":"BTC"}]}}`,
			expected: `{"EXCHANGES":[{"NAME":"GateIO"}],"portfolioaddresses":{"ADDRESSES":[{"ADDRESS":"GateIO","DESCRIPTION":"Exchange","COINTYPE":"BTC"}]}}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out, err := new(v19.Version).DowngradeConfig(t.Context(), []byte(tc.input))
			require.NoError(t, err, "DowngradeConfig must not error")
			assert.JSONEq(t, tc.expected, string(out), "DowngradeConfig should rename only Gate")
		})
	}
}

func TestDeploy(t *testing.T) {
	t.Parallel()
	in := []byte(`{"version":18,"exchanges":[{"name":"GateIO","api":{"urlEndpoints":{"RestFuturesURL":"https://proxy.example/gate-fx"}},"bankAccounts":[{"supportedExchanges":"GateIO"}]},{"name":"Kraken"}],"bankAccounts":[{"supportedExchanges":"Kraken,GateIO"}],"portfolioAddresses":{"addresses":[{"Address":"GateIO","CoinType":"BTC","Balance":1,"Description":"Exchange"},{"Address":"1abc","CoinType":"BTC","Description":"Personal","SupportedExchanges":"GateIO,Kraken"}]}}`)
	out, err := versions.Manager.Deploy(t.Context(), in, 19)
	require.NoError(t, err, "Deploy must not error upgrading to version 19")
	assert.JSONEq(t, `{"version":19,"exchanges":[{"name":"Gate","api":{"urlEndpoints":{"RestFuturesURL":"https://proxy.example/gate-fx"}},"bankAccounts":[{"supportedExchanges":"Gate"}]},{"name":"Kraken"}],"bankAccounts":[{"supportedExchanges":"Kraken,Gate"}],"portfolioAddresses":{"addresses":[{"Address":"Gate","CoinType":"BTC","Balance":1,"Description":"Exchange"},{"Address":"1abc","CoinType":"BTC","Description":"Personal","SupportedExchanges":"Gate,Kraken"}]}}`, string(out), "Deploy should rename GateIO to Gate")

	out, err = versions.Manager.Deploy(t.Context(), out, 18)
	require.NoError(t, err, "Deploy must not error downgrading to version 18")
	assert.JSONEq(t, string(in), string(out), "Deploy should restore the version 18 config")
}
