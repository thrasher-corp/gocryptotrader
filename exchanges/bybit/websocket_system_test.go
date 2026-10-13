package bybit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestSystemStatusConnectionSetup(t *testing.T) {
	t.Parallel()
	ex := websocketTestInstance(t)
	setup := ex.SystemStatusConnectionSetup(websocketSystemStatus)
	require.NotNil(t, setup, "SystemStatusConnectionSetup must return a setup")
	assert.Equal(t, websocketSystemStatus, setup.URL, "setup should connect to the system status stream")
	assert.Equal(t, SystemStatusConnection, setup.MessageFilter, "setup should be found by its message filter")
	assert.Equal(t, ex.Config.WebsocketResponseCheckTimeout, setup.ResponseCheckTimeout, "setup should take the configured response check timeout")
	assert.Equal(t, ex.Config.WebsocketResponseMaxLimit, setup.ResponseMaxLimit, "setup should take the configured response limit")
	for name, set := range map[string]bool{"Connector": setup.Connector != nil, "GenerateSubscriptions": setup.GenerateSubscriptions != nil, "Subscriber": setup.Subscriber != nil, "Unsubscriber": setup.Unsubscriber != nil, "Handler": setup.Handler != nil} {
		assert.Truef(t, set, "setup should set %s", name)
	}
	subs, err := setup.GenerateSubscriptions()
	require.NoError(t, err, "GenerateSubscriptions must not error")
	assert.Empty(t, subs, "the system status connection should not subscribe unless configured to")
	ex.Features.Subscriptions = append(ex.Features.Subscriptions, &subscription.Subscription{Enabled: true, Channel: chanSystemStatus})
	subs, err = setup.GenerateSubscriptions()
	require.NoError(t, err, "GenerateSubscriptions must not error")
	assert.Equal(t, []string{chanSystemStatus}, subs.QualifiedChannels(), "a configured system.status subscription should be generated")

	unconfigured := new(Exchange)
	setup = unconfigured.SystemStatusConnectionSetup(websocketSystemStatus)
	assert.Equal(t, exchange.DefaultWebsocketResponseCheckTimeout, setup.ResponseCheckTimeout, "setup should fall back to the default response check timeout")
	assert.Equal(t, exchange.DefaultWebsocketResponseMaxLimit, setup.ResponseMaxLimit, "setup should fall back to the default response limit")
}

func TestGenerateSystemStatusSubscriptions(t *testing.T) {
	t.Parallel()
	ex := websocketTestInstance(t)
	ex.Features.Subscriptions = subscription.List{{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel}}
	subs, err := ex.generateSystemStatusSubscriptions()
	require.NoError(t, err, "generateSystemStatusSubscriptions must not error")
	assert.Nil(t, subs, "generateSystemStatusSubscriptions should generate nothing unless system.status is configured")

	configured := &subscription.Subscription{Enabled: true, Channel: chanSystemStatus}
	ex.Features.Subscriptions = append(ex.Features.Subscriptions, configured)
	subs, err = ex.generateSystemStatusSubscriptions()
	require.NoError(t, err, "generateSystemStatusSubscriptions must not error")
	exp := subscription.List{{Enabled: true, Channel: chanSystemStatus, QualifiedChannel: chanSystemStatus}}
	assert.Equal(t, exp, subs, "generateSystemStatusSubscriptions should generate the configured subscription qualified by its topic")
	require.Len(t, subs, 1, "generateSystemStatusSubscriptions must generate one subscription")
	assert.NotSame(t, configured, subs[0], "generateSystemStatusSubscriptions should generate a copy of the configured subscription")
	assert.Empty(t, configured.QualifiedChannel, "generateSystemStatusSubscriptions should leave the configured subscription unchanged")
}

func TestSystemStatusSubscribe(t *testing.T) {
	t.Parallel()
	ex := websocketTestInstance(t)
	subs := subscription.List{{Enabled: true, Channel: chanSystemStatus}}
	conn := &FixtureConnection{reply: optionsReply}
	require.NoError(t, ex.SystemStatusSubscribe(t.Context(), conn, subs), "SystemStatusSubscribe must not error")
	require.Len(t, conn.sent, 1, "SystemStatusSubscribe must send one request")
	req, ok := conn.sent[0].(*SubscriptionRequest)
	require.True(t, ok, "SystemStatusSubscribe must send a SubscriptionRequest")
	assert.Equal(t, []string{chanSystemStatus}, req.Arguments, "SystemStatusSubscribe should send the system.status topic")
	assert.Equal(t, []string{chanSystemStatus}, ex.Websocket.GetSubscriptions().QualifiedChannels(), "SystemStatusSubscribe should store the subscription")
	require.NoError(t, ex.SystemStatusUnsubscribe(t.Context(), conn, subs), "SystemStatusUnsubscribe must not error")
	assert.Empty(t, ex.Websocket.GetSubscriptions(), "SystemStatusUnsubscribe should remove the subscription")
}

func TestWebsocketSystemStatus(t *testing.T) {
	t.Parallel()
	ex := websocketTestInstance(t)
	if !mockTests {
		// Maintenance notices are pushed only around maintenance, so a quiet window proves only the subscription
		frames, reply := liveStreamFrames(t, websocketSystemStatus, 5*time.Second, 1, chanSystemStatus)
		var resp SubscriptionResponse
		require.NoError(t, json.Unmarshal(reply, &resp), "subscription reply must decode")
		require.True(t, resp.Success, "system.status must be accepted")
		assert.Equal(t, []string{chanSystemStatus}, resp.Data.SuccessTopics, "system.status should be applied")
		for _, frame := range frames {
			require.NoError(t, ex.wsHandleSystemStatus(t.Context(), nil, frame), "wsHandleSystemStatus must not error")
		}
		return
	}
	frames := fixtureFrames(t, "testdata/wsSystemStatus.json")
	pushes := decodeFramesStrictly[WebsocketSystemStatusResponse](t, frames)
	expPushes := []WebsocketSystemStatusResponse{
		{
			Topic:     "system.status",
			Timestamp: types.Time(time.UnixMilli(1751858399649)),
			Data: []WebsocketSystemStatus{
				{
					ID:    "4d95b2a0-587f-11f0-bcc9-56f28c94d6ea",
					Title: "t06",
					State: "completed",
					Begin: types.Time(time.Unix(1751596902, 0)),
					End:   types.Time(time.Unix(1751597011, 0)),
					Href:  "https://announcements.bybit.com/en/article/system-maintenance",
					ServiceTypes: []uint64{
						2,
						3,
						4,
						5,
					},
					Product: []uint64{
						1,
						2,
					},
					UIDSuffix: []uint64{
						3,
						7,
					},
					MaintainType: 1,
					Env:          1,
				},
			},
		},
	}
	assert.Equal(t, expPushes, pushes, "fixture frames should decode every system status field")
	for i, frame := range frames {
		require.NoErrorf(t, ex.wsHandleSystemStatus(t.Context(), nil, frame), "wsHandleSystemStatus must not error for frame %d", i)
	}
	data := relayed(ex)
	require.Len(t, data, len(frames), "every push must be relayed")
	for i := range pushes {
		assert.Equalf(t, &pushes[i], data[i], "push %d should be relayed decoded", i)
	}

	err := ex.wsHandleSystemStatus(t.Context(), nil, []byte(`{"topic":"system.bogus","ts":1751858399649,"data":[]}`))
	require.ErrorIs(t, err, errUnhandledStreamData, "wsHandleSystemStatus must error for an unknown topic")
	err = ex.wsHandleSystemStatus(t.Context(), nil, []byte(`{"topic":1}`))
	require.Error(t, err, "wsHandleSystemStatus must error for a topic that is not a string")
	require.NoError(t, ex.wsHandleSystemStatus(t.Context(), &FixtureConnection{}, []byte(`{"args":["1791534274878"],"op":"pong"}`)), "wsHandleSystemStatus must handle a pong")
}
