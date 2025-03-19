package exchange

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
)

type evenFakerBase struct {
	FakeBase
}

func (e *evenFakerBase) SubmitOrder(context.Context, *order.Submit) (*order.SubmitResponse, error) {
	return nil, errors.New("random error that denotes functionality is here but outbound requests are not possible")
}

func TestGenerateSupportedFunctionality(t *testing.T) {
	require.Nil(t, GenerateSupportedFunctionality(nil))
	fake := &evenFakerBase{}
	require.Empty(t, GenerateSupportedFunctionality(fake), "no assets supported")
	require.NoError(t, fake.CurrencyPairs.Store(asset.Spot, &currency.PairStore{}))
	require.NoError(t, fake.CurrencyPairs.SetAssetEnabled(asset.Spot, true))
	set := GenerateSupportedFunctionality(fake)
	require.NotEmpty(t, set, "assets supported")

	_, ok := set[protocol.Target{Asset: asset.Spot, Protocol: protocol.Websocket}]
	require.False(t, ok, "no websocket support for this asset")
	restSpotFunctionality, ok := set[protocol.Target{Asset: asset.Spot, Protocol: protocol.REST}]
	require.True(t, ok, "rest support for this asset")
	require.True(t, restSpotFunctionality["SubmitOrder"], "submit order must be functional for this protocol and this asset")
}
