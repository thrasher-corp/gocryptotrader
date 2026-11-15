package v20

import (
	"encoding/json" //nolint:depguard // v20 decodes with the standard library, so its tests need the same JSON types
	"errors"
	"testing"

	"github.com/buger/jsonparser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpgradeConfig(t *testing.T) {
	t.Parallel()

	t.Run("refuses a config holding both GateIO and Gate", func(t *testing.T) {
		t.Parallel()
		// Neither exchange config can be dropped without losing a user's settings, so the version must not advance
		in := `{"version":19,"exchanges":[{"name":"GateIO","enabled":true},{"name":"gate","enabled":false}]}`
		out, err := new(Version).UpgradeConfig(t.Context(), []byte(in))
		assert.ErrorIs(t, err, errExchangeNameConflict, "UpgradeConfig should refuse a config holding both GateIO and Gate")
		assert.JSONEq(t, in, string(out), "UpgradeConfig should leave a conflicting config unchanged")
	})

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
			out, err := new(Version).UpgradeConfig(t.Context(), []byte(tc.input))
			require.NoError(t, err, "UpgradeConfig must not error")
			assert.JSONEq(t, tc.expected, string(out), "UpgradeConfig should rename only GateIO")
		})
	}

	for _, tc := range []struct {
		name, input string
	}{
		{name: "non-string exchange name", input: `{"exchanges":[{"name":42}]}`},
		{name: "unterminated exchanges", input: `{"exchanges":[`},
		{name: "unterminated bank accounts", input: `{"bankAccounts":[{"supportedExchanges":"GateIO"}`},
		{name: "unterminated portfolio addresses", input: `{"portfolioAddresses":{"addresses":[{"Address":"GateIO","Description":"Exchange"}`},
		{name: "invalid escape in a portfolio address", input: `{"portfolioAddresses":{"addresses":[{"Address":"GateIO","Description":"Exchange","SupportedExchanges":"\u"}]}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := new(Version).UpgradeConfig(t.Context(), []byte(tc.input))
			assert.Errorf(t, err, "UpgradeConfig should error on %s", tc.input)
		})
	}
}

func TestDowngradeConfig(t *testing.T) {
	t.Parallel()

	t.Run("refuses a config holding both Gate and GateIO", func(t *testing.T) {
		t.Parallel()
		in := `{"version":20,"exchanges":[{"name":"Gate"},{"name":"GATEIO"}]}`
		out, err := new(Version).DowngradeConfig(t.Context(), []byte(in))
		assert.ErrorIs(t, err, errExchangeNameConflict, "DowngradeConfig should refuse a config holding both Gate and GateIO")
		assert.JSONEq(t, in, string(out), "DowngradeConfig should leave a conflicting config unchanged")
	})

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
			out, err := new(Version).DowngradeConfig(t.Context(), []byte(tc.input))
			require.NoError(t, err, "DowngradeConfig must not error")
			assert.JSONEq(t, tc.expected, string(out), "DowngradeConfig should rename only Gate")
		})
	}
}

func TestRenameExchange(t *testing.T) {
	t.Parallel()

	t.Run("non-string exchange name", func(t *testing.T) {
		t.Parallel()
		in := `{"exchanges":[{"name":42}]}`
		out, err := renameExchange([]byte(in), oldName, newName)
		var typeErr *json.UnmarshalTypeError
		require.ErrorAs(t, err, &typeErr, "renameExchange must return the JSON type error")
		assert.Equal(t, in, string(out), "renameExchange should return the config unchanged on error")
	})

	for _, tc := range []struct {
		name, input, from, to, expected string
		err                             error
	}{
		{
			name:     "exchange named in any case",
			input:    `{"exchanges":[{"name":"Kraken"},{"name":"gateio"}]}`,
			from:     oldName,
			to:       newName,
			expected: `{"exchanges":[{"name":"Kraken"},{"name":"Gate"}]}`,
		},
		{
			name:     "exchange bank accounts",
			input:    `{"exchanges":[{"name":"Kraken","bankAccounts":[{"supportedExchanges":"Kraken,GateIO"},{"supportedExchanges":"ALL"}]}]}`,
			from:     oldName,
			to:       newName,
			expected: `{"exchanges":[{"name":"Kraken","bankAccounts":[{"supportedExchanges":"Kraken,Gate"},{"supportedExchanges":"ALL"}]}]}`,
		},
		{
			name:     "client bank accounts",
			input:    `{"bankAccounts":[{"supportedExchanges":"GATEIO"}]}`,
			from:     oldName,
			to:       newName,
			expected: `{"bankAccounts":[{"supportedExchanges":"Gate"}]}`,
		},
		{
			name:     "portfolio addresses",
			input:    `{"portfolioAddresses":{"addresses":[{"Address":"GateIO","CoinType":"BTC","Description":"Exchange"}]}}`,
			from:     oldName,
			to:       newName,
			expected: `{"portfolioAddresses":{"addresses":[{"Address":"Gate","CoinType":"BTC","Description":"Exchange"}]}}`,
		},
		{
			name:     "only the new name",
			input:    `{"exchanges":[{"name":"Gate"}],"bankAccounts":[{"supportedExchanges":"Gate"}]}`,
			from:     oldName,
			to:       newName,
			expected: `{"exchanges":[{"name":"Gate"}],"bankAccounts":[{"supportedExchanges":"Gate"}]}`,
		},
		{
			name:     "Gate back to GateIO",
			input:    `{"exchanges":[{"name":"gate"}],"bankAccounts":[{"supportedExchanges":"Gate"}]}`,
			from:     newName,
			to:       oldName,
			expected: `{"exchanges":[{"name":"GateIO"}],"bankAccounts":[{"supportedExchanges":"GateIO"}]}`,
		},
		{
			name:  "both names",
			input: `{"exchanges":[{"name":"GateIO"},{"name":"Gate"}]}`,
			from:  oldName,
			to:    newName,
			err:   errExchangeNameConflict,
		},
		{
			name:  "unterminated exchanges",
			input: `{"exchanges":[`,
			from:  oldName,
			to:    newName,
			err:   jsonparser.MalformedArrayError,
		},
		{
			name:  "unterminated bank accounts",
			input: `{"bankAccounts":[`,
			from:  oldName,
			to:    newName,
			err:   jsonparser.MalformedArrayError,
		},
		{
			name:  "unterminated portfolio addresses",
			input: `{"portfolioAddresses":{"addresses":[`,
			from:  oldName,
			to:    newName,
			err:   jsonparser.MalformedArrayError,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out, err := renameExchange([]byte(tc.input), tc.from, tc.to)
			require.ErrorIs(t, err, tc.err, "renameExchange must return the expected error")
			if tc.err != nil {
				assert.Equal(t, tc.input, string(out), "renameExchange should return the config unchanged on error")
				return
			}
			assert.JSONEq(t, tc.expected, string(out), "renameExchange should rename the exchange wherever the config lists it")
		})
	}
}

func TestRenamePortfolioAddresses(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, input, expected string
		err                   error
	}{
		{name: "missing addresses", input: `{"portfolioAddresses":{"verbose":true}}`},
		{name: "empty addresses", input: `{"portfolioAddresses":{"addresses":[]}}`},
		{
			name:     "exchange balance",
			input:    `{"portfolioAddresses":{"addresses":[{"Address":"GateIO","CoinType":"BTC","Balance":1,"Description":"Exchange"}]}}`,
			expected: `{"portfolioAddresses":{"addresses":[{"Address":"Gate","CoinType":"BTC","Balance":1,"Description":"Exchange"}]}}`,
		},
		{
			name:  "exchange balance named in another case",
			input: `{"portfolioAddresses":{"addresses":[{"Address":"gateio","CoinType":"BTC","Description":"Exchange"}]}}`,
		},
		{
			name:  "personal address named after the exchange",
			input: `{"portfolioAddresses":{"addresses":[{"Address":"GateIO","CoinType":"BTC","Description":"Personal"}]}}`,
		},
		{
			name:     "balance already held under the new name",
			input:    `{"portfolioAddresses":{"addresses":[{"Address":"GateIO","CoinType":"BTC","Balance":1,"Description":"Exchange"},{"Address":"Gate","CoinType":"btc","Balance":2,"Description":"Exchange"},{"Address":"GateIO","CoinType":"ETH","Balance":3,"Description":"Exchange"}]}}`,
			expected: `{"portfolioAddresses":{"addresses":[{"Address":"Gate","CoinType":"btc","Balance":2,"Description":"Exchange"},{"Address":"Gate","CoinType":"ETH","Balance":3,"Description":"Exchange"}]}}`,
		},
		{
			name:     "supported exchanges",
			input:    `{"portfolioAddresses":{"addresses":[{"Address":"1abc","CoinType":"BTC","Description":"Personal","SupportedExchanges":"Kraken,gateio"}]}}`,
			expected: `{"portfolioAddresses":{"addresses":[{"Address":"1abc","CoinType":"BTC","Description":"Personal","SupportedExchanges":"Kraken,Gate"}]}}`,
		},
		{
			name:     "keys in another case",
			input:    `{"PortfolioAddresses":{"Addresses":[{"address":"GateIO","coinType":"BTC","description":"Exchange"}]}}`,
			expected: `{"PortfolioAddresses":{"Addresses":[{"address":"Gate","coinType":"BTC","description":"Exchange"}]}}`,
		},
		{name: "unterminated addresses", input: `{"portfolioAddresses":{"addresses":[`, err: jsonparser.MalformedArrayError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out, err := renamePortfolioAddresses([]byte(tc.input), oldName, newName)
			require.ErrorIs(t, err, tc.err, "renamePortfolioAddresses must return the expected error")
			if tc.expected == "" {
				assert.Equal(t, tc.input, string(out), "renamePortfolioAddresses should return the config unchanged")
				return
			}
			assert.JSONEq(t, tc.expected, string(out), "renamePortfolioAddresses should rename the exchange's balances and address lists")
		})
	}
}

func TestExchangeBalance(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, address, exchange, coin string
		ok                            bool
	}{
		{name: "exchange balance", address: `{"Address":"GateIO","CoinType":"BTC","Balance":1,"Description":"Exchange"}`, exchange: "GateIO", coin: "BTC", ok: true},
		{name: "keys in another case", address: `{"address":"GateIO","cointype":"BTC","DESCRIPTION":"Exchange"}`, exchange: "GateIO", coin: "BTC", ok: true},
		{name: "missing currency", address: `{"Address":"GateIO","Description":"Exchange"}`, exchange: "GateIO", ok: true},
		{name: "personal address", address: `{"Address":"1abc","CoinType":"BTC","Description":"Personal"}`},
		{name: "missing description", address: `{"Address":"GateIO","CoinType":"BTC"}`},
		{name: "missing address", address: `{"CoinType":"BTC","Description":"Exchange"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			exchange, coin, ok := exchangeBalance([]byte(tc.address))
			assert.Equal(t, tc.exchange, exchange, "exchangeBalance should return the exchange")
			assert.Equal(t, tc.coin, coin, "exchangeBalance should return the currency")
			assert.Equal(t, tc.ok, ok, "exchangeBalance should report whether the address holds an exchange balance")
		})
	}
}

func TestRenameListed(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, element, expected string
		err                     error
	}{
		{name: "named in any case with its spacing kept", element: `{"supportedExchanges":"Kraken, gateio ,Bitstamp"}`, expected: `{"supportedExchanges":"Kraken, Gate ,Bitstamp"}`},
		{name: "named more than once", element: `{"supportedExchanges":"GateIO,GATEIO"}`, expected: `{"supportedExchanges":"Gate,Gate"}`},
		{name: "key in another case", element: `{"SupportedExchanges":"GateIO"}`, expected: `{"SupportedExchanges":"Gate"}`},
		{name: "similar names", element: `{"supportedExchanges":"GateIOX,Gateway"}`},
		{name: "missing list", element: `{"bankName":"a"}`},
		{name: "null list", element: `{"supportedExchanges":null}`},
		{name: "non-string list", element: `{"supportedExchanges":["GateIO"]}`},
		{name: "unterminated list", element: `{"supportedExchanges":"GateIO`, err: jsonparser.MalformedStringError},
		{name: "invalid escape", element: `{"supportedExchanges":"\u"}`, err: jsonparser.MalformedValueError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out, err := renameListed([]byte(tc.element), "supportedExchanges", oldName, newName)
			require.ErrorIs(t, err, tc.err, "renameListed must return the expected error")
			if tc.expected == "" {
				assert.Equal(t, tc.element, string(out), "renameListed should return the element unchanged")
				return
			}
			assert.JSONEq(t, tc.expected, string(out), "renameListed should rename the exchange in the list")
		})
	}
}

func TestEditArray(t *testing.T) {
	t.Parallel()
	errEdit := errors.New("edit failed")
	edit := func(element []byte) ([]byte, error) {
		switch string(element) {
		case `"fail"`:
			return nil, errEdit
		case `"old"`:
			return []byte(`"new"`), nil
		default:
			return element, nil
		}
	}
	for _, tc := range []struct {
		name, input string
		path        []string
		expected    string
		err         error
	}{
		{name: "changed elements", input: `{"list":["old","kept","old"]}`, path: []string{"list"}, expected: `{"list":["new","kept","new"]}`},
		{name: "keys in another case", input: `{"Outer":{"List":["old"]}}`, path: []string{"outer", "list"}, expected: `{"Outer":{"List":["new"]}}`},
		{name: "unchanged elements", input: `{"list":["kept"]}`, path: []string{"list"}},
		{name: "missing array", input: `{"other":["old"]}`, path: []string{"list"}},
		{name: "non-array value", input: `{"list":"old"}`, path: []string{"list"}},
		{name: "edit error", input: `{"list":["fail","old"]}`, path: []string{"list"}, err: errEdit},
		{name: "unterminated array", input: `{"list":[`, path: []string{"list"}, err: jsonparser.MalformedArrayError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out, err := editArray([]byte(tc.input), edit, tc.path...)
			require.ErrorIs(t, err, tc.err, "editArray must return the expected error")
			if tc.expected == "" {
				assert.Equal(t, tc.input, string(out), "editArray should return the config unchanged")
				return
			}
			assert.JSONEq(t, tc.expected, string(out), "editArray should set each element the edit changes")
		})
	}
}

func TestGetArray(t *testing.T) {
	t.Parallel()

	t.Run("invalid element", func(t *testing.T) {
		t.Parallel()
		_, err := getArray([]byte(`{"list":[1,]}`), "list")
		var syntaxErr *json.SyntaxError
		assert.ErrorAs(t, err, &syntaxErr, "getArray should return the JSON syntax error")
	})

	for _, tc := range []struct {
		name     string
		input    string
		path     []string
		expected []json.RawMessage
		err      error
	}{
		{name: "array", input: `{"list":[1,{"a":"b"},"c"]}`, path: []string{"list"}, expected: []json.RawMessage{json.RawMessage(`1`), json.RawMessage(`{"a":"b"}`), json.RawMessage(`"c"`)}},
		{name: "keys in another case", input: `{"Outer":{"LIST":[1]}}`, path: []string{"outer", "list"}, expected: []json.RawMessage{json.RawMessage(`1`)}},
		{name: "array index in path", input: `{"exchanges":[{"bankAccounts":[1]}]}`, path: []string{"exchanges", "[0]", "bankAccounts"}, expected: []json.RawMessage{json.RawMessage(`1`)}},
		{name: "missing array", input: `{}`, path: []string{"list"}},
		{name: "null", input: `{"list":null}`, path: []string{"list"}},
		{name: "object", input: `{"list":{"a":1}}`, path: []string{"list"}},
		{name: "unterminated array", input: `{"list":[`, path: []string{"list"}, err: jsonparser.MalformedArrayError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := getArray([]byte(tc.input), tc.path...)
			require.ErrorIs(t, err, tc.err, "getArray must return the expected error")
			assert.Equal(t, tc.expected, got, "getArray should return the elements of the array at path")
		})
	}
}

func TestSetString(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, input, value string
		path               []string
		expected           string
		err                error
	}{
		{name: "value written as a JSON string", input: `{"name":"GateIO","enabled":true}`, value: `Gate "io"`, path: []string{"name"}, expected: `{"name":"Gate \"io\"","enabled":true}`},
		{name: "key in another case", input: `{"Name":"GateIO"}`, value: newName, path: []string{"name"}, expected: `{"Name":"Gate"}`},
		{name: "not an object", input: `not json`, value: newName, path: []string{"name"}, err: jsonparser.KeyPathNotFoundError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out, err := setString([]byte(tc.input), tc.value, tc.path...)
			require.ErrorIs(t, err, tc.err, "setString must return the expected error")
			if tc.err != nil {
				return
			}
			assert.JSONEq(t, tc.expected, string(out), "setString should set the value at path as a JSON string")
		})
	}
}

func TestSet(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, input, value string
		path               []string
		expected           string
		err                error
	}{
		{name: "existing key", input: `{"a":1,"b":2}`, value: `{"c":3}`, path: []string{"a"}, expected: `{"a":{"c":3},"b":2}`},
		{name: "keys in another case", input: `{"Outer":{"Inner":1}}`, value: `2`, path: []string{"outer", "inner"}, expected: `{"Outer":{"Inner":2}}`},
		{name: "array element", input: `{"list":[1,2]}`, value: `3`, path: []string{"list", "[1]"}, expected: `{"list":[1,3]}`},
		{name: "missing key", input: `{"a":1}`, value: `2`, path: []string{"b"}, expected: `{"a":1,"b":2}`},
		{name: "not an object", input: `not json`, value: `1`, path: []string{"a"}, err: jsonparser.KeyPathNotFoundError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out, err := set([]byte(tc.input), []byte(tc.value), tc.path...)
			require.ErrorIs(t, err, tc.err, "set must return the expected error")
			if tc.err != nil {
				return
			}
			assert.JSONEq(t, tc.expected, string(out), "set should set the value at path")
		})
	}
}

func TestResolve(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, input    string
		path, expected []string
	}{
		{name: "same spelling", input: `{"exchanges":[{"name":"GateIO"}]}`, path: []string{"exchanges", "[0]", "name"}, expected: []string{"exchanges", "[0]", "name"}},
		{name: "keys in another case", input: `{"Exchanges":[{"NAME":"GateIO"}]}`, path: []string{"exchanges", "[0]", "name"}, expected: []string{"Exchanges", "[0]", "NAME"}},
		{name: "last of several matching keys", input: `{"name":"Kraken","Name":"GateIO"}`, path: []string{"name"}, expected: []string{"Name"}},
		{name: "stops at a missing key", input: `{"Outer":{"Deep":1}}`, path: []string{"outer", "inner", "deep"}, expected: []string{"Outer", "inner", "deep"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, resolve([]byte(tc.input), tc.path...), "resolve should spell each key of path as the data does")
		})
	}
}

func TestIndex(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "[12]", index(12), "index should write an array index path element")
}
