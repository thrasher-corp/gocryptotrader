package exchange

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
)

func TestAutomaticPreflightCheck(t *testing.T) {
	require.Nil(t, AutomaticPreFlightCheck(nil))
	fake := &FakeBase{}
	require.Empty(t, AutomaticPreFlightCheck(fake), "no assets supported")
	require.NoError(t, fake.CurrencyPairs.Store(asset.Spot, &currency.PairStore{}))
	require.NoError(t, fake.CurrencyPairs.SetAssetEnabled(asset.Spot, true))
	set := AutomaticPreFlightCheck(fake)
	require.NotEmpty(t, set, "assets supported")
	restSpot := set[protocol.Target{Asset: asset.Spot, Protocol: protocol.REST}]
	require.True(t, restSpot.SubmitOrder, "submit order should be functional for this protocol and this asset")
}
