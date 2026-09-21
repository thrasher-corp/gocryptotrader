package kraken

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

type mockAuthSubConnection struct {
	websocket.Connection
	responses [][]byte
	expected  int
}

func (m *mockAuthSubConnection) SendMessageReturnResponses(_ context.Context, _ request.EndpointLimit, _, _ any, expected int) ([][]byte, error) {
	m.expected = expected
	return m.responses, nil
}

func TestManageSubs(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name             string
		channel          string
		qualifiedChannel string
		response         []byte
		responseCount    int
		errIs            error
		errContains      string
	}{
		{name: "own trades", channel: subscription.MyTradesChannel, qualifiedChannel: krakenWsOwnTrades, response: []byte(`{"channelName":"ownTrades","event":"subscriptionStatus","reqid":3,"status":"subscribed","subscription":{"name":"ownTrades"}}`), responseCount: 1},
		{name: "open orders", channel: subscription.MyOrdersChannel, qualifiedChannel: krakenWsOpenOrders, response: []byte(`{"channelName":"openOrders","event":"subscriptionStatus","reqid":3,"status":"subscribed","subscription":{"name":"openOrders"}}`), responseCount: 1},
		{name: "requires single response", channel: subscription.MyTradesChannel, qualifiedChannel: krakenWsOwnTrades, responseCount: 0, errIs: errExpectedOneSubResponse, errContains: "got 0; Channel: myTrades"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "Setup Instance must not error")

			conn := &mockAuthSubConnection{expected: -1}
			if tc.responseCount > 0 {
				conn.responses = [][]byte{tc.response}
			}
			ex.Websocket.AuthConn = conn

			err := ex.manageSubs(t.Context(), krakenWsSubscribe, subscription.List{{
				Channel:          tc.channel,
				QualifiedChannel: tc.qualifiedChannel,
				Authenticated:    true,
			}})
			if tc.errIs != nil {
				require.ErrorIs(t, err, tc.errIs)
				require.ErrorContains(t, err, tc.errContains)
			} else {
				require.NoError(t, err, "auth subscription without pairs must not error")
			}
			assert.Equal(t, 1, conn.expected, "auth subscription without pairs waits for one response")
		})
	}
}

func TestWsProcessSubStatusInvalidPair(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup Instance must not error")
	s := &subscription.Subscription{
		Channel: subscription.TickerChannel,
		Pairs:   currency.Pairs{currency.NewBTCUSD()},
	}
	require.NoError(t, ex.Websocket.AddSubscriptions(nil, s), "subscription must be added in subscribing state")

	ex.wsProcessSubStatus([]byte(`{"channelName":"ticker","event":"subscriptionStatus","pair":"not-a-pair","status":"subscribed","subscription":{"name":"ticker"}}`))
	assert.Equal(t, subscription.SubscribingState, s.State(), "invalid websocket subscription pair should leave the subscription state unchanged")
}

// TestWsProcessTickers covers the v1 ticker's [today, last 24 hours] arrays. UpdateTickers records
// the 24 hour volume and range for the same pair, so reading today's wrote a different window into
// the same store entry
func TestWsProcessTickers(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")

	// The payload of a live wss://ws.kraken.com ticker message for XBT/USD
	data := json.RawMessage(`{"a":["78165.70000",0,"0.34373236"],"b":["78165.60000",0,"0.63986310"],"c":["78165.60000","0.00028359"],"v":["439.88105355","2394.31405834"],"p":["78299.32832","78599.42339"],"t":[23720,101224],"l":["77932.40000","77754.90000"],"h":["78500.00000","79739.10000"],"o":["78288.60000","79224.50000"]}`)
	pair := currency.NewPairWithDelimiter("XBT", "USD", "/")
	require.NoError(t, ex.wsProcessTickers(t.Context(), data, pair), "wsProcessTickers must not error")

	select {
	case msg := <-ex.Websocket.DataHandler.C:
		got, ok := msg.Data.(*ticker.Price)
		require.True(t, ok, "wsProcessTickers must send a ticker price")
		assert.Equal(t, &ticker.Price{
			ExchangeName: ex.Name,
			Pair:         pair,
			AssetType:    asset.Spot,
			Ask:          78165.7,
			AskSize:      0.34373236,
			Bid:          78165.6,
			BidSize:      0.6398631,
			Last:         78165.6,
			Close:        78165.6,
			BaseVolume:   2394.31405834,
			Low:          77754.9,
			High:         79739.1,
			Open:         78288.6,
		}, got, "the ticker should record what UpdateTickers records for the same pair")
	default:
		require.Fail(t, "no ticker price sent", "wsProcessTickers must send a ticker price")
	}
}
