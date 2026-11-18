package websocket

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
)

func TestAuthenticationRecoveryIntegration(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                string
		mixed               bool
		dialFailure         bool
		disabled            bool
		disableAfterFailure bool
		permanent           bool
		remoteClose         bool
		removeChannels      bool
		timerDriven         bool
		mixedBatch          bool
		requestOnly         bool
		unmarkedPrivate     bool
		blockedAuth         bool
		disableDuringAuth   bool
		flushPending        bool
		blockedGeneration   bool
	}{
		{name: "private login recovers"},
		{name: "disable during subscription generation suppresses private recovery", mixed: true, blockedGeneration: true, disableDuringAuth: true},
		{name: "blocked authentication never advertises availability", mixed: true, blockedAuth: true},
		{name: "disable during authentication suppresses private recovery", mixed: true, blockedAuth: true, disableDuringAuth: true},
		{name: "flush while authentication pending preserves public", mixed: true, removeChannels: true, flushPending: true},
		{name: "unmarked mixed private subscriptions wait for login", mixed: true, unmarkedPrivate: true},
		{name: "mixed batches authenticate independently", mixed: true, mixedBatch: true},
		{name: "request-only connection recovers", requestOnly: true},
		{name: "monitor retries private login", timerDriven: true},
		{name: "private refusal and remote close preserve public", remoteClose: true},
		{name: "removed private channels stay removed", removeChannels: true},
		{name: "private dial recovers", dialFailure: true},
		{name: "mixed login recovers on public connection", mixed: true},
		{name: "permanent private refusal preserves public", permanent: true},
		{name: "configuration disables private attempts", disabled: true},
		{name: "explicit disable stops pending retries", disableAfterFailure: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			server, dialer := mockws.NewTestServer(t, mockws.CurryWsMockUpgrader(t, func(_ testing.TB, _ []byte, conn *gws.Conn) error {
				return conn.Close()
			}))
			manager := NewManager()

			setup := newDefaultSetup()
			setup.UseMultiConnectionManagement = true
			setup.Features.Unsubscribe = tc.flushPending
			setup.ExchangeConfig.ConnectionMonitorDelay = time.Hour
			if tc.timerDriven {
				setup.ExchangeConfig.ConnectionMonitorDelay = 10 * time.Millisecond
			}
			setup.ExchangeConfig.WebsocketTrafficTimeout = time.Hour
			setup.ExchangeConfig.API.AuthenticatedWebsocketSupport = !tc.disabled
			require.NoError(t, manager.Setup(setup), "manager setup must succeed")
			if tc.mixedBatch {
				manager.MaxSubscriptionsPerConnection = 1
			}
			// Recovery is driven synchronously so retry counts do not depend on timer scheduling.
			if !tc.timerDriven {
				manager.connectionMonitorRunning.Store(true)
			}
			t.Cleanup(func() {
				if !tc.timerDriven {
					manager.connectionMonitorRunning.Store(false)
				}
				cleanupManagerMonitors(t, manager)
			})
			entered, release := make(chan struct{}), make(chan struct{})
			t.Cleanup(func() {
				select {
				case <-release:
				default:
					close(release)
				}
			})
			var removed atomic.Bool
			var publicDials, privateDials, logins, publicSubscriptions, privateSubscriptions atomic.Int64
			subscribe := func(_ context.Context, conn Connection, subs subscription.List) error {
				for _, sub := range subs {
					if sub.Channel == "orders" {
						privateSubscriptions.Add(1)
					} else {
						publicSubscriptions.Add(1)
					}
				}
				return manager.AddSuccessfulSubscriptions(conn, subs...)
			}
			authenticate := func(ctx context.Context, conn Connection) error {
				attempt := logins.Add(1)
				if tc.blockedAuth && attempt == 2 {
					manager.SetCanUseAuthenticatedEndpoints(true)
					close(entered)
					<-release
				}
				if tc.remoteClose && attempt == 1 {
					require.NoError(t, conn.SendRawMessage(ctx, 0, gws.TextMessage, []byte("login")), "login request must reach refusing server")
					require.Eventually(t, func() bool { return len(manager.ReadMessageErrors) > 0 }, time.Second, time.Millisecond, "remote close must reach manager reader")
				}
				if attempt == 1 || tc.permanent {
					manager.SetCanUseAuthenticatedEndpoints(false)
					return errDastardlyReason
				}
				manager.SetCanUseAuthenticatedEndpoints(true)
				return nil
			}
			publicSetup := &ConnectionSetup{
				URL:           "ws" + strings.TrimPrefix(server.URL, "http"),
				MessageFilter: "public",
				Connector: func(ctx context.Context, conn Connection) error {
					publicDials.Add(1)
					return conn.Dial(ctx, dialer, nil, nil)
				},
				GenerateSubscriptions: func() (subscription.List, error) {
					subs := subscription.List{{Channel: "ticker"}}
					if tc.mixed && manager.CanGenerateAuthenticatedSubscriptions() && !removed.Load() {
						subs = append(subs, &subscription.Subscription{Channel: "orders", Authenticated: !tc.unmarkedPrivate})
					}
					if tc.blockedGeneration && logins.Load() >= 2 {
						close(entered)
						<-release
					}
					return subs, nil
				},
				Subscriber: subscribe,
				Unsubscriber: func(_ context.Context, _ Connection, subs subscription.List) error {
					assert.Empty(t, subs, "pending authentication should not remove public subscriptions")
					return nil
				},
				Handler: func(context.Context, Connection, []byte) error { return nil },
			}
			if tc.mixed {
				publicSetup.Authenticate = authenticate
			}
			require.NoError(t, manager.SetupNewConnection(publicSetup), "public registration must succeed")
			if !tc.mixed {
				privateSetup := *publicSetup
				privateSetup.MessageFilter = "private"
				privateSetup.Authenticated = true
				privateSetup.Authenticate = authenticate
				if tc.requestOnly {
					privateSetup.Authenticated = false
					privateSetup.SubscriptionsNotRequired = true
				}
				privateSetup.Connector = func(ctx context.Context, conn Connection) error {
					attempt := privateDials.Add(1)
					if tc.dialFailure && attempt == 1 {
						return errDastardlyReason
					}
					return conn.Dial(ctx, dialer, nil, nil)
				}
				privateSetup.GenerateSubscriptions = func() (subscription.List, error) {
					if tc.requestOnly || !manager.CanGenerateAuthenticatedSubscriptions() || removed.Load() {
						return nil, nil
					}
					return subscription.List{{Channel: "orders", Authenticated: true}}, nil
				}
				require.NoError(t, manager.SetupNewConnection(&privateSetup), "private registration must succeed")
			}
			err := manager.Connect(t.Context())
			if tc.disabled {
				require.NoError(t, err, "public connection must succeed with auth disabled")
			} else {
				require.ErrorIs(t, err, errDastardlyReason, "initial attempt must expose private failure")
			}
			if tc.mixedBatch {
				assert.EqualValues(t, 2, logins.Load(), "second physical connection should attempt authentication after first fails")
				assert.EqualValues(t, 1, privateSubscriptions.Load(), "authenticated second connection should subscribe privately")
			}
			if !tc.mixedBatch && !tc.timerDriven {
				assert.Zero(t, privateSubscriptions.Load(), "failed initial login should never subscribe to private channels")
			}
			public, err := manager.GetConnection("public")
			require.NoError(t, err, "public connection must survive private failure")
			if tc.remoteClose {
				timer := time.NewTimer(time.Hour)
				defer timer.Stop()
				require.False(t, manager.observeConnection(t.Context(), timer), "stale private fault must not stop monitor")
			}
			if tc.removeChannels {
				removed.Store(true)
			}
			if tc.disableAfterFailure {
				manager.SetAuthenticatedWebsocketSupport(false)
			}
			if tc.flushPending {
				require.ErrorIs(t, manager.FlushChannels(t.Context()), errFailedToAuthenticate, "pending authentication must remain visible to flush caller")
				assert.Zero(t, privateSubscriptions.Load(), "flush should not send pending private subscriptions")
			}
			initialLogins, initialPrivateDials := logins.Load(), privateDials.Load()
			switch {
			case tc.blockedAuth || tc.blockedGeneration:
				done := make(chan error, 1)
				go func() { done <- manager.retryAuthentication(t.Context()) }()
				select {
				case <-entered:
				case <-time.After(3 * time.Second):
					require.FailNow(t, "retry must reach the blocked callback")
				}
				assert.False(t, manager.CanUseAuthenticatedEndpoints(), "pending recovery should never advertise authenticated availability")
				assert.False(t, manager.CanUseAuthenticatedWebsocketForWrapper(), "pending recovery should preserve wrapper REST routing")
				if tc.disableDuringAuth {
					manager.SetAuthenticatedWebsocketSupport(false)
				}
				close(release)
				select {
				case retryErr := <-done:
					if !tc.disableDuringAuth {
						require.NoError(t, retryErr, "successful authentication must finish recovery")
					}
				case <-time.After(3 * time.Second):
					require.FailNow(t, "retry must finish after the callback is released")
				}
				if tc.disableDuringAuth {
					assert.False(t, manager.CanUseAuthenticatedEndpoints(), "explicit disable should remain effective after callback completion")
					require.NoError(t, manager.retryAuthentication(t.Context()), "disabled authentication must suppress later retries")
					assert.EqualValues(t, 2, logins.Load(), "disabled authentication should prevent further login attempts")
				}
			case tc.timerDriven:
				require.Eventually(t, func() bool { return privateSubscriptions.Load() == 1 }, 3*time.Second, time.Millisecond, "connection monitor must restore private subscriptions")
			default:
				for range 3 {
					err = manager.retryAuthentication(t.Context())
					if tc.permanent {
						require.ErrorIs(t, err, errDastardlyReason, "permanent refusal must remain visible")
					}
				}
			}
			currentPublic, err := manager.GetConnection("public")
			require.NoError(t, err, "public connection must remain available")
			assert.Same(t, public, currentPublic, "recovery should retain the public physical connection")
			expectedPublicDials := int64(1)
			if tc.mixedBatch {
				expectedPublicDials = 2
			}
			assert.Equal(t, expectedPublicDials, publicDials.Load(), "recovery should not redial public streams")
			assert.EqualValues(t, 1, publicSubscriptions.Load(), "recovery should not duplicate public subscriptions")
			switch {
			case tc.disableDuringAuth:
				assert.Equal(t, 1, public.Subscriptions().Len(), "disabled recovery should retain only the public subscription on its connection")
				assert.Zero(t, privateSubscriptions.Load(), "disable during recovery should prevent subsequent private subscriptions")
			case tc.disabled || tc.disableAfterFailure:
				if tc.disabled {
					assert.Zero(t, initialLogins, "configured-disabled authentication should never log in")
					assert.Zero(t, initialPrivateDials, "configured-disabled authentication should never dial privately")
				}
				assert.Equal(t, initialLogins, logins.Load(), "disabled authentication should not retry login")
				assert.Equal(t, initialPrivateDials, privateDials.Load(), "disabled authentication should not retry private dial")
				assert.Zero(t, privateSubscriptions.Load(), "disabled authentication should not subscribe privately")
			case tc.requestOnly:
				assert.GreaterOrEqual(t, logins.Load(), int64(2), "request-only login should recover")
				private, err := manager.GetConnection("private")
				require.NoError(t, err, "request-only private connection must recover")
				assert.Zero(t, private.Subscriptions().Len(), "request-only connection should own no subscriptions")
			case tc.removeChannels:
				assert.Zero(t, privateSubscriptions.Load(), "recovery should not restore removed channels")
			case tc.permanent:
				assert.Zero(t, privateSubscriptions.Load(), "refused authentication should not subscribe privately")
				assert.Greater(t, logins.Load(), int64(1), "private login should remain retryable")
			default:
				assert.EqualValues(t, 1, privateSubscriptions.Load(), "successful recovery should subscribe privately exactly once")
				filter := "private"
				expected := 1
				if tc.mixed {
					filter, expected = "public", 2
					if tc.mixedBatch {
						expected = 1
					}
				}
				private, err := manager.GetConnection(filter)
				require.NoError(t, err, "recovered private connection must be available")
				assert.Equal(t, expected, private.Subscriptions().Len(), "physical connection should own its recovered subscriptions")
			}
		})
	}
}
