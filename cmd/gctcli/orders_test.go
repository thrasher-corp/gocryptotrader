package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/urfave/cli/v2"
)

func TestSubmitOrderRequest(t *testing.T) {
	// pairDelimiter is a CLI singleton.
	previous := pairDelimiter
	pairDelimiter = "-"
	t.Cleanup(func() { pairDelimiter = previous })
	for _, tc := range []struct{ name, clientID, clientOrderID string }{
		{name: "separate identifiers", clientID: "account", clientOrderID: "order-42"},
		{name: "account only", clientID: "account"},
		{name: "order only", clientOrderID: "order-42"},
		{name: "neither"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			command := *submitOrderCommand
			command.Action = func(c *cli.Context) error {
				req, err := submitOrderRequest(c)
				require.NoError(t, err, "valid order flags must build a request")
				assert.Equal(t, tc.clientID, req.ClientId, "account identifier should remain separate")
				assert.Equal(t, tc.clientOrderID, req.ClientOrderId, "client order identifier should reach its own RPC field")
				assert.Equal(t, "BTC", req.Pair.Base, "base should decode")
				assert.Equal(t, "USD", req.Pair.Quote, "quote should decode")
				assert.Equal(t, "spot", req.AssetType, "asset should be normalised")
				assert.Equal(t, 2.0, req.Amount, "amount should reach RPC")
				assert.Equal(t, 100.0, req.Price, "price should reach RPC")
				return nil
			}
			app := &cli.App{Commands: []*cli.Command{&command}}
			require.NoError(t, app.Run([]string{"gctcli", "submitorder", "--exchange", "HTX", "--pair", "BTC-USD", "--side", "buy", "--type", "limit", "--amount", "2", "--price", "100", "--asset", "SPOT", "--client_id", tc.clientID, "--client_order_id", tc.clientOrderID}), "command flags must parse")
		})
	}
	for _, tc := range []struct {
		name, flag, value string
		want              error
	}{
		{name: "missing side", flag: sideFlag, want: errSubmitOrderSideRequired},
		{name: "missing type", flag: "type", want: errSubmitOrderTypeRequired},
		{name: "zero amount", flag: amountFlag, value: "0", want: errSubmitOrderAmountRequired},
		{name: "invalid pair", flag: pairFlag, value: "X", want: errInvalidPair},
		{name: "invalid asset", flag: assetFlag, value: "bogus", want: errInvalidAsset},
		{name: "invalid margin", flag: "margintype", value: "bogus", want: margin.ErrInvalidMarginType},
	} {
		t.Run(tc.name, func(t *testing.T) {
			command := *submitOrderCommand
			command.Action = func(c *cli.Context) error {
				_, err := submitOrderRequest(c)
				require.ErrorIs(t, err, tc.want, "invalid order flags must return the expected sentinel")
				return nil
			}
			app := &cli.App{Commands: []*cli.Command{&command}}
			require.NoError(t, app.Run([]string{"gctcli", "submitorder", "--exchange", "HTX", "--pair", "BTC-USD", "--side", "buy", "--type", "limit", "--amount", "2", "--asset", "spot", "--" + tc.flag, tc.value}), "invalid value must reach request validation")
		})
	}
}
