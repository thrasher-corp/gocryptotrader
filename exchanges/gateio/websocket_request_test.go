package gateio

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func TestWebsocketLogin(t *testing.T) {
	t.Parallel()
	_, err := g.WebsocketLogin(context.Background(), asset.Futures)
	require.ErrorIs(t, err, common.ErrNotYetImplemented)

	require.NoError(t, g.UpdateTradablePairs(context.Background(), false))
	for _, a := range g.GetAssetTypes(true) {
		avail, err := g.GetAvailablePairs(a)
		require.NoError(t, err)
		if len(avail) > 1 {
			avail = avail[:1]
		}
		require.NoError(t, g.SetPairs(avail, a, true))
	}
	require.NoError(t, g.Websocket.Connect())
	g.GetBase().API.AuthenticatedSupport = true
	g.GetBase().API.AuthenticatedWebsocketSupport = true

	got, err := g.WebsocketLogin(context.Background(), asset.Spot)
	require.NoError(t, err)

	fmt.Println(got)
}
