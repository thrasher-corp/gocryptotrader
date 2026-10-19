package kraken

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// spotBalancesServer serves Get Extended Balance's reply to the refreshes the balances channel starts, counting them
type spotBalancesServer struct {
	mu    sync.Mutex
	reads int
	// replies answers the first reads in turn, and reply the reads after them
	replies []string
	reply   string
}

func (s *spotBalancesServer) handle(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		if !assert.Equal(t, "/0/private/BalanceEx", r.URL.Path, "only the extended balances should be read") {
			http.Error(w, "unexpected request", http.StatusNotFound)
			return
		}
		s.mu.Lock()
		s.reads++
		reply := s.reply
		if len(s.replies) > 0 {
			reply, s.replies = s.replies[0], s.replies[1:]
		}
		s.mu.Unlock()
		_, err := w.Write([]byte(reply))
		assert.NoError(t, err, "Write should not error")
	}
}

func (s *spotBalancesServer) setReply(reply string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reply = reply
}

func (s *spotBalancesServer) readCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.reads
}

// assertStoredSpotBalances waits for the stored spot balances to be exp, as balances refresh in the background
func assertStoredSpotBalances(t *testing.T, ex *Exchange, exp accounts.CurrencyBalances, msg string) {
	t.Helper()
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		got, err := ex.Accounts.CurrencyBalances(ex.GetDefaultCredentials(), asset.Spot)
		if !assert.NoError(c, err, "CurrencyBalances should not error") {
			return
		}
		for code, b := range got {
			b.UpdatedAt = time.Time{}
			got[code] = b
		}
		assert.Equal(c, exp, got, msg)
	}, 5*time.Second, time.Millisecond, msg)
}

// spotLedgerWithdrawal is a balances channel update reporting a withdrawal that leaves 1 BTC
var spotLedgerWithdrawal = &wsChannelMessage{Type: "update", Data: json.RawMessage(`[{"ledger_id":"L4UESK-KG3EQ-UFO4T5","type":"withdrawal","asset":"BTC","asset_class":"currency","wallet_type":"spot","wallet_id":"main","amount":-0.2,"fee":0,"balance":1,"timestamp":"2023-09-22T10:33:05.709993Z"}]`)}

// useConnectedRefreshers has an exchange's REST refreshes carry on as they do while its websocket is connected
func useConnectedRefreshers(ex *Exchange) {
	connected := func() bool { return true }
	ex.spotBalances.active, ex.spotOrders.active, ex.spotDayOpens.active = connected, connected, connected
}

func TestWebsocketRefresher(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var calls [][]string
	started, release := make(chan struct{}, 1), make(chan struct{})
	r := &wsRefresher{interval: time.Millisecond}
	refresh := func(_ context.Context, keys []string) []string {
		mu.Lock()
		calls = append(calls, keys)
		first := len(calls) == 1
		mu.Unlock()
		if first {
			started <- struct{}{}
			<-release
		}
		return nil
	}
	callCount := func() int {
		mu.Lock()
		defer mu.Unlock()
		return len(calls)
	}
	r.refresh(t.Context(), refresh, "a")
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "the refresh must start")
	}
	// Keys asked for while a refresh runs wait for the next, which takes them together
	for _, k := range []string{"b", "c", "b"} {
		r.refresh(t.Context(), refresh, k)
	}
	assert.Never(t, func() bool { return callCount() > 1 }, 50*time.Millisecond, time.Millisecond, "a refresh should not start while another runs")
	close(release)
	assert.Eventually(t, func() bool { return callCount() == 2 }, 5*time.Second, time.Millisecond, "the keys asked for during a refresh should be refreshed once more")
	assert.Never(t, func() bool { return callCount() > 2 }, 50*time.Millisecond, time.Millisecond, "a refresh with nothing pending should start no other")
	mu.Lock()
	assert.Equal(t, [][]string{{"a"}, {"b", "c"}}, calls, "each refresh should take the keys pending once each")
	mu.Unlock()
	assert.Eventually(t, func() bool {
		r.mu.Lock()
		defer r.mu.Unlock()
		return !r.running
	}, 5*time.Second, time.Millisecond, "the refresher should stop once nothing is pending")
}

func TestWebsocketRefresherAttempts(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	attempts := make(map[string]int)
	r := &wsRefresher{interval: time.Millisecond}
	// Settles nothing, as REST still lagging the websocket would
	refresh := func(_ context.Context, keys []string) []string {
		mu.Lock()
		defer mu.Unlock()
		for _, k := range keys {
			attempts[k]++
		}
		return keys
	}
	attemptsOf := func(k string) int {
		mu.Lock()
		defer mu.Unlock()
		return attempts[k]
	}
	idle := func() bool {
		r.mu.Lock()
		defer r.mu.Unlock()
		return !r.running
	}
	r.refresh(t.Context(), refresh, "a")
	require.Eventually(t, idle, 5*time.Second, time.Millisecond, "the refresher must stop once a key has had its attempts")
	assert.Equal(t, wsRefreshAttempts, attemptsOf("a"), "an unsettled key should be refreshed wsRefreshAttempts times")

	// A key asked for again has its attempts afresh
	r.refresh(t.Context(), refresh, "a")
	require.Eventually(t, idle, 5*time.Second, time.Millisecond, "the refresher must stop once a key has had its attempts")
	assert.Equal(t, 2*wsRefreshAttempts, attemptsOf("a"), "a key asked for again should have its attempts afresh")

	// So does a key asked for again while it waits for another attempt, as a run leaves a key refreshed twice
	r.mu.Lock()
	r.running, r.pending = true, map[string]int{"b": 2}
	r.mu.Unlock()
	r.refresh(t.Context(), refresh, "b")
	go r.run(t.Context(), refresh, time.Millisecond)
	require.Eventually(t, idle, 5*time.Second, time.Millisecond, "the refresher must stop once a key has had its attempts")
	assert.Equal(t, wsRefreshAttempts, attemptsOf("b"), "a key asked for again while it waits for another attempt should have its attempts afresh")

	// And a key asked for again while it is being refreshed
	started, resume := make(chan struct{}), make(chan struct{})
	var once sync.Once
	blocking := func(ctx context.Context, keys []string) []string {
		once.Do(func() {
			close(started)
			<-resume
		})
		return refresh(ctx, keys)
	}
	r.refresh(t.Context(), blocking, "c")
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "the refresh must start")
	}
	r.refresh(t.Context(), blocking, "c")
	close(resume)
	require.Eventually(t, idle, 5*time.Second, time.Millisecond, "the refresher must stop once a key has had its attempts")
	assert.Equal(t, 1+wsRefreshAttempts, attemptsOf("c"), "a key asked for again while it is being refreshed should have its attempts afresh")
}

func TestWebsocketRefresherSettledRetention(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var calls [][]string
	started, release := make(chan struct{}, 1), make(chan struct{})
	r := &wsRefresher{interval: time.Millisecond, settledRetention: time.Hour}
	refresh := func(_ context.Context, keys []string) []string {
		mu.Lock()
		calls = append(calls, keys)
		first := len(calls) == 1
		mu.Unlock()
		if first {
			started <- struct{}{}
			<-release
		}
		return nil
	}
	callCount := func() int {
		mu.Lock()
		defer mu.Unlock()
		return len(calls)
	}
	idle := func() bool {
		r.mu.Lock()
		defer r.mu.Unlock()
		return !r.running
	}
	r.refresh(t.Context(), refresh, "a")
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "the refresh must start")
	}
	// Asked for again while it is being settled, the key has its answer once settled
	r.refresh(t.Context(), refresh, "a")
	close(release)
	require.Eventually(t, idle, 5*time.Second, time.Millisecond, "the refresher must stop once the key is settled")
	assert.Equal(t, 1, callCount(), "a key asked for again while it was being settled should not be refreshed again")

	r.refresh(t.Context(), refresh, "a")
	assert.Never(t, func() bool { return callCount() > 1 }, 20*time.Millisecond, time.Millisecond, "a key settled within the retention should be passed over")
	r.mu.Lock()
	r.settled["a"] = time.Now().Add(-2 * time.Hour)
	r.mu.Unlock()
	r.refresh(t.Context(), refresh, "a")
	assert.Eventually(t, func() bool { return callCount() == 2 }, 5*time.Second, time.Millisecond, "a key settled before the retention should be refreshed again")
}

func TestRefresherWiring(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	for name, r := range map[string]*wsRefresher{"spot balances": &ex.spotBalances, "spot orders": &ex.spotOrders, "spot day opens": &ex.spotDayOpens} {
		require.NotNilf(t, r.active, "the %s refresher must follow the websocket's connection", name)
		assert.Equalf(t, ex.websocketActive(), r.active(), "the %s refresher should report the websocket's connection", name)
	}
	assert.Equal(t, spotOrderReadRetention, ex.spotOrders.settledRetention, "the spot orders refresher should read a finished order once")
	assert.Zero(t, ex.spotBalances.settledRetention, "the spot balances refresher should refresh whenever asked")
	assert.Equal(t, dayOpensReadRetention, ex.spotDayOpens.settledRetention, "the spot day opens refresher should read a day's opening prices once")
}

func TestSaveSpotBalances(t *testing.T) {
	t.Parallel()
	ex := newHTTPTestExchange(t, func(http.ResponseWriter, *http.Request) {
		assert.Fail(t, "saveSpotBalances should make no request")
	})
	balances := func(total float64) accounts.SubAccounts {
		return accounts.SubAccounts{{AssetType: asset.Spot, Balances: accounts.CurrencyBalances{currency.USD: {Total: total, Free: total}}}}
	}
	require.NoError(t, ex.saveSpotBalances(t.Context(), 2, balances(100)), "saveSpotBalances must not error")
	// A read requested before the stored one finished after it, so it is older and is not stored
	require.NoError(t, ex.saveSpotBalances(t.Context(), 1, balances(50)), "saveSpotBalances must not error for an older read")
	b, err := ex.Accounts.GetBalance("", ex.GetDefaultCredentials(), asset.Spot, currency.USD)
	require.NoError(t, err, "GetBalance must not error")
	assert.Equal(t, 100.0, b.Total, "saveSpotBalances should not store a read requested before the stored one")
	require.NoError(t, ex.saveSpotBalances(t.Context(), 3, balances(75)), "saveSpotBalances must not error for a newer read")
	b, err = ex.Accounts.GetBalance("", ex.GetDefaultCredentials(), asset.Spot, currency.USD)
	require.NoError(t, err, "GetBalance must not error")
	assert.Equal(t, 75.0, b.Total, "saveSpotBalances should store a read requested after the stored one")

	unsupported := accounts.SubAccounts{{AssetType: asset.Empty, Balances: accounts.CurrencyBalances{currency.USD: {Total: 25, Free: 25}}}}
	assert.ErrorIs(t, ex.saveSpotBalances(t.Context(), 5, unsupported), asset.ErrNotSupported, "saveSpotBalances should return the error saving the read")
	require.NoError(t, ex.saveSpotBalances(t.Context(), 4, balances(60)), "saveSpotBalances must not error for a read requested before one it could not save")
	b, err = ex.Accounts.GetBalance("", ex.GetDefaultCredentials(), asset.Spot, currency.USD)
	require.NoError(t, err, "GetBalance must not error")
	assert.Equal(t, 60.0, b.Total, "saveSpotBalances should not count a read it could not save as stored")
	require.NoError(t, ex.saveSpotBalances(t.Context(), 4, balances(65)), "saveSpotBalances must not error for a read numbered as the stored one")
	b, err = ex.Accounts.GetBalance("", ex.GetDefaultCredentials(), asset.Spot, currency.USD)
	require.NoError(t, err, "GetBalance must not error")
	assert.Equal(t, 65.0, b.Total, "saveSpotBalances should store a read numbered as the stored one")

	noCreds := newTestExchange(t)
	noCreds.SkipAuthCheck = false
	noCreds.API.AuthenticatedSupport = true
	noCreds.SetCredentials(&accounts.Credentials{})
	assert.ErrorIs(t, noCreds.saveSpotBalances(t.Context(), 1, balances(100)), exchange.ErrCredentialsAreEmpty, "saveSpotBalances should reject missing credentials")
}

func TestRefreshSpotBalancesRetries(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	reads := 0
	ex := newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/0/private/BalanceEx", r.URL.Path, "only the extended balances should be read")
		mu.Lock()
		reads++
		mu.Unlock()
		_, err := w.Write([]byte(`{"error":["EService:Unavailable"]}`))
		assert.NoError(t, err, "Write should not error")
	})
	seedTestAssetNames(ex)
	ex.spotBalances.interval = time.Millisecond
	useConnectedRefreshers(ex)
	readCount := func() int {
		mu.Lock()
		defer mu.Unlock()
		return reads
	}
	require.NoError(t, ex.wsProcessBalances(t.Context(), &wsChannelMessage{Type: wsTypeSnapshot, Data: json.RawMessage(`[]`)}), "wsProcessBalances must not error")
	assert.Eventually(t, func() bool { return readCount() == wsRefreshAttempts }, 5*time.Second, time.Millisecond, "a failing refresh should be retried until it has had its attempts")
	assert.Never(t, func() bool { return readCount() > wsRefreshAttempts }, 50*time.Millisecond, time.Millisecond, "a failing refresh should stop after its attempts")
	require.Eventually(t, func() bool {
		ex.spotBalances.mu.Lock()
		defer ex.spotBalances.mu.Unlock()
		return !ex.spotBalances.running
	}, 5*time.Second, time.Millisecond, "the refresher must stop once the refresh has had its attempts")

	// The exchange's own websocket is not connected, so a failed refresh is not made again
	ex.spotBalances.mu.Lock()
	ex.spotBalances.active = ex.websocketActive
	ex.spotBalances.mu.Unlock()
	require.NoError(t, ex.wsProcessBalances(t.Context(), &wsChannelMessage{Type: wsTypeSnapshot, Data: json.RawMessage(`[]`)}), "wsProcessBalances must not error")
	assert.Eventually(t, func() bool { return readCount() == wsRefreshAttempts+1 }, 5*time.Second, time.Millisecond, "a balances message should start a refresh")
	assert.Never(t, func() bool { return readCount() > wsRefreshAttempts+1 }, 50*time.Millisecond, time.Millisecond, "a refresh should not be made again once the websocket has disconnected")
}

func TestRefreshSpotBalancesAfterLedgerUpdates(t *testing.T) {
	t.Parallel()
	server := &spotBalancesServer{
		// REST answers the read made at once with the balance from before the withdrawal the ledger update reports
		replies: []string{`{"error":[],"result":{"XXBT":{"balance":"1.2","hold_trade":"0"}}}`},
		reply:   `{"error":[],"result":{"XXBT":{"balance":"1","hold_trade":"0"}}}`,
	}
	ex := newHTTPTestExchange(t, server.handle(t))
	seedTestAssetNames(ex)
	ex.spotBalances.interval = time.Millisecond
	useConnectedRefreshers(ex)
	require.NoError(t, ex.wsProcessBalances(t.Context(), spotLedgerWithdrawal), "wsProcessBalances must not error")
	assertStoredSpotBalances(t, ex, accounts.CurrencyBalances{
		currency.XBT: {Currency: currency.XBT, Total: 1, Free: 1, AvailableWithoutBorrow: 1},
	}, "a refresh after a ledger update should read the balances once more, as REST can lag the update")
	assert.Never(t, func() bool { return server.readCount() > 2 }, 100*time.Millisecond, time.Millisecond, "a ledger update should cost two reads")

	// A snapshot is the balances as they are, so its refresh reads them once
	snapshot := &wsChannelMessage{Type: wsTypeSnapshot, Data: json.RawMessage(`[]`)}
	require.NoError(t, ex.wsProcessBalances(t.Context(), snapshot), "wsProcessBalances must not error")
	assert.Eventually(t, func() bool { return server.readCount() == 3 }, 5*time.Second, time.Millisecond, "a snapshot should start a refresh")
	assert.Never(t, func() bool { return server.readCount() > 3 }, 100*time.Millisecond, time.Millisecond, "a snapshot should cost one read")
}

func TestRefreshSpotBalancesConfirmAfterFailure(t *testing.T) {
	t.Parallel()
	server := &spotBalancesServer{
		// The read made at once fails, and the read retrying it answers with the balance from before the withdrawal
		replies: []string{`{"error":["EService:Unavailable"]}`, `{"error":[],"result":{"XXBT":{"balance":"1.2","hold_trade":"0"}}}`},
		reply:   `{"error":[],"result":{"XXBT":{"balance":"1","hold_trade":"0"}}}`,
	}
	ex := newHTTPTestExchange(t, server.handle(t))
	seedTestAssetNames(ex)
	ex.spotBalances.interval = time.Millisecond
	useConnectedRefreshers(ex)
	require.NoError(t, ex.wsProcessBalances(t.Context(), spotLedgerWithdrawal), "wsProcessBalances must not error")
	assertStoredSpotBalances(t, ex, accounts.CurrencyBalances{
		currency.XBT: {Currency: currency.XBT, Total: 1, Free: 1, AvailableWithoutBorrow: 1},
	}, "a failed read should not stand in for the read confirming a ledger update")
	assert.Never(t, func() bool { return server.readCount() > 3 }, 50*time.Millisecond, time.Millisecond, "a ledger update should cost its failed read and two reads")
}

func TestRefreshSpotBalancesLedgerUpdateDuringRead(t *testing.T) {
	t.Parallel()
	server := &spotBalancesServer{
		// The read in flight when the ledger update arrives, and the read after it, answer with the balance from before
		// the withdrawal
		replies: []string{`{"error":[],"result":{"XXBT":{"balance":"1.2","hold_trade":"0"}}}`, `{"error":[],"result":{"XXBT":{"balance":"1.2","hold_trade":"0"}}}`},
		reply:   `{"error":[],"result":{"XXBT":{"balance":"1","hold_trade":"0"}}}`,
	}
	firstRead, resume := make(chan struct{}), make(chan struct{})
	var requests atomic.Int32
	handle := server.handle(t)
	ex := newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) == 1 {
			close(firstRead)
			<-resume
		}
		handle(w, r)
	})
	var resumeOnce sync.Once
	resumeReads := func() { resumeOnce.Do(func() { close(resume) }) }
	// Cleanups run last first, so a held read is let go before the server closes
	t.Cleanup(resumeReads)
	seedTestAssetNames(ex)
	ex.spotBalances.interval = time.Millisecond
	useConnectedRefreshers(ex)
	require.NoError(t, ex.wsProcessBalances(t.Context(), &wsChannelMessage{Type: wsTypeSnapshot, Data: json.RawMessage(`[]`)}), "wsProcessBalances must not error")
	select {
	case <-firstRead:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "a snapshot must start a refresh")
	}
	require.NoError(t, ex.wsProcessBalances(t.Context(), spotLedgerWithdrawal), "wsProcessBalances must not error")
	resumeReads()
	assertStoredSpotBalances(t, ex, accounts.CurrencyBalances{
		currency.XBT: {Currency: currency.XBT, Total: 1, Free: 1, AvailableWithoutBorrow: 1},
	}, "a read started before a ledger update should not stand in for the reads after it")
	assert.Never(t, func() bool { return server.readCount() > 3 }, 50*time.Millisecond, time.Millisecond, "a ledger update during a read should cost two reads after it")
}

func TestWebsocketActive(t *testing.T) {
	t.Parallel()
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		<-release
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)
	ex := newTestExchange(t)
	seedTestAssetNames(ex)
	useTestWebsocket(t, ex, "ws"+strings.TrimPrefix(server.URL, "http"), wsPublicConnection)
	ex.Websocket.SetSubscriptionsNotRequired()
	var releaseOnce sync.Once
	releaseServer := func() { releaseOnce.Do(func() { close(release) }) }
	// Cleanups run last first, so the held dial is let go before the websocket is disabled and the server closes
	t.Cleanup(releaseServer)
	assert.False(t, ex.websocketActive(), "websocketActive should report a websocket not connected inactive")

	connectErr := make(chan error, 1)
	go func() { connectErr <- ex.Websocket.Connect(t.Context()) }()
	require.Eventually(t, ex.Websocket.IsConnecting, 5*time.Second, time.Millisecond, "the websocket must be connecting while the server holds its connection")
	// The manager is connecting until its last connection connects, while the connections before it stream messages
	assert.True(t, ex.websocketActive(), "websocketActive should report a websocket still connecting active")
	releaseServer()
	select {
	case err := <-connectErr:
		require.Error(t, err, "Connect must error when the server refuses the connection")
	case <-time.After(5 * time.Second):
		require.FailNow(t, "Connect must return once the server refuses the connection")
	}
	assert.False(t, ex.websocketActive(), "websocketActive should report a websocket that failed to connect inactive")

	connected := newTestExchange(t)
	seedTestAssetNames(connected)
	useTestWebsocket(t, connected, newMockWebsocketURL(t, func(testing.TB, []byte, *gws.Conn) error { return nil }), wsPublicConnection)
	connected.Websocket.SetSubscriptionsNotRequired()
	require.NoError(t, connected.Websocket.Connect(t.Context()), "Connect must not error")
	assert.True(t, connected.websocketActive(), "websocketActive should report a connected websocket active")
	require.NoError(t, connected.Websocket.Shutdown(), "Shutdown must not error")
	assert.False(t, connected.websocketActive(), "websocketActive should report a websocket shut down inactive")
}

func TestWebsocketRefresherStopsOnDisconnect(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	calls := 0
	connected := false
	r := &wsRefresher{interval: time.Millisecond, active: func() bool {
		mu.Lock()
		defer mu.Unlock()
		return connected
	}}
	refresh := func(_ context.Context, keys []string) []string {
		mu.Lock()
		defer mu.Unlock()
		calls++
		return keys
	}
	callCount := func() int {
		mu.Lock()
		defer mu.Unlock()
		return calls
	}
	r.refresh(t.Context(), refresh, "a")
	require.Eventually(t, func() bool {
		r.mu.Lock()
		defer r.mu.Unlock()
		return !r.running
	}, 5*time.Second, time.Millisecond, "the refresher must stop once the websocket has disconnected")
	assert.Equal(t, 1, callCount(), "a refresh should be made once while the websocket is disconnected, for the message that started it")
	r.mu.Lock()
	assert.Empty(t, r.pending, "the refresher should drop the keys pending when it stops")
	r.mu.Unlock()

	// A message on a new connection starts a refresh again
	mu.Lock()
	connected = true
	mu.Unlock()
	r.refresh(t.Context(), refresh, "a")
	assert.Eventually(t, func() bool { return callCount() == 1+wsRefreshAttempts }, 5*time.Second, time.Millisecond, "a refresh should be made again while the websocket is connected")
}

// finishedSpotOrdersServer serves Query Orders Info's replies, one for each read, recording the orders each read asks
// for
type finishedSpotOrdersServer struct {
	mu      sync.Mutex
	replies []string
	reads   [][]string
	// hold, when set, holds every reply until it is closed, each read being recorded first
	hold chan struct{}
}

func (s *finishedSpotOrdersServer) handle(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		if !assert.Equal(t, "/0/private/QueryOrders", r.URL.Path, "only orders should be read") {
			http.Error(w, "unexpected request", http.StatusNotFound)
			return
		}
		var body struct {
			OrderIDs string `json:"txid"`
		}
		assert.NoError(t, json.Unmarshal([]byte(tradingRequestBody(t, r)), &body), "Unmarshal should not error")
		s.mu.Lock()
		s.reads = append(s.reads, strings.Split(body.OrderIDs, ","))
		reply := `{"error":["EGeneral:Internal error"]}`
		if len(s.replies) != 0 {
			reply, s.replies = s.replies[0], s.replies[1:]
		}
		s.mu.Unlock()
		if s.hold != nil {
			select {
			case <-s.hold:
			case <-r.Context().Done():
				return
			}
		}
		_, err := w.Write([]byte(reply))
		assert.NoError(t, err, "Write should not error")
	}
}

func (s *finishedSpotOrdersServer) readOrders() [][]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.reads)
}

// finishedSpotOrderID is the order the finished spot order tests read
const finishedSpotOrderID = "OK4GJX-KSTLS-7DZZO5"

// queryOrdersReply returns Query Orders Info's reply for finishedSpotOrderID, an order of 10 XBT/USD bought at 26599.9,
// with its status, executed volume and fee
func queryOrdersReply(t *testing.T, status string, executed, fee float64) string {
	t.Helper()
	b, err := json.Marshal(map[string]any{"error": []string{}, "result": map[string]any{finishedSpotOrderID: map[string]any{
		"status": status, "opentm": 1695378785.7, "closetm": 1695378786.7, "vol": "10", "vol_exec": executed, "cost": executed * 26599.9,
		"fee": fee, "price": "26599.9", "descr": map[string]any{"pair": "XBTUSD", "type": "buy", "ordertype": "limit", "price": "26599.9"},
	}}})
	require.NoError(t, err, "Marshal must not error")
	return string(b)
}

func TestFinishSpotOrder(t *testing.T) {
	t.Parallel()
	const orderID = finishedSpotOrderID
	server := &finishedSpotOrdersServer{replies: []string{
		// REST can still report the order open after the websocket reported it filled
		queryOrdersReply(t, "open", 6, 0.2),
		queryOrdersReply(t, "closed", 10, 0.6916),
	}, hold: make(chan struct{})}
	ex := newHTTPTestExchange(t, server.handle(t))
	// The reads are held until the updates the executions relay are checked, as the read starts as soon as they are
	// processed. Cleanups run last first, so a held read is let go before the server closes
	release := sync.OnceFunc(func() { close(server.hold) })
	t.Cleanup(release)
	seedTestAssetNames(ex)
	ex.spotOrders.interval = time.Millisecond
	useConnectedRefreshers(ex)

	// The order as the order store holds it once placed, which merges each update relayed for it
	stored := &order.Detail{Exchange: ex.Name, AssetType: asset.Spot, OrderID: orderID, Pair: spotTestPair, Side: order.Buy, Type: order.Limit, Status: order.New, Amount: 10, Price: 26599.9}
	var read *order.Detail
	merge := func() {
		for _, payload := range relayed(ex) {
			if d, ok := payload.(*order.Detail); ok {
				require.NoError(t, stored.UpdateOrderFromDetail(d), "UpdateOrderFromDetail must not error")
				if d.Fee != 0 {
					read = d
				}
			}
		}
	}
	// The last fill reports the order filled, and the filled event reports it again
	msg := &wsChannelMessage{Data: json.RawMessage(`[{"order_id":"` + orderID + `","exec_id":"TGBB7L-HT5LX-J3BZ4A","exec_type":"trade","trade_id":62887576,"symbol":"BTC/USD","side":"buy","last_qty":10,"last_price":26599.9,"liquidity_ind":"t","cost":265999,"order_type":"limit","timestamp":"2023-09-22T10:33:05.709993Z","order_status":"filled","cum_qty":10,"cum_cost":265999,"avg_price":26599.9,"fees":[{"asset":"USD","qty":0.4916}]},` +
		`{"order_id":"` + orderID + `","exec_type":"filled","order_status":"filled","cum_qty":10,"timestamp":"2023-09-22T10:33:05.709994Z"}]`)}
	require.NoError(t, ex.wsProcessExecutions(t.Context(), msg), "wsProcessExecutions must not error")
	merge()
	assert.Zero(t, stored.Fee, "an order's trade events should leave its fee to the read")
	release()
	assert.Eventually(t, func() bool { return len(server.readOrders()) == 2 }, 5*time.Second, time.Millisecond, "the order should be read again while REST reports it open")
	assert.Eventually(t, func() bool {
		merge()
		return stored.Fee != 0
	}, 5*time.Second, time.Millisecond, "the read should relay the order's fee")
	assert.Equal(t, [][]string{{orderID}, {orderID}}, server.readOrders(), "an order reported finished twice should be read once, and again only while REST reports it open")
	assert.Equal(t, order.Filled, stored.Status, "the order should be stored filled")
	assert.Equal(t, 0.6916, stored.Fee, "the order should be stored with the fee Kraken reports for the whole order")
	assert.Equal(t, 10.0, stored.ExecutedAmount, "the order should be stored filled in full")
	require.NotNil(t, read, "the read must relay the order")
	assert.Equal(t, currency.USD, read.FeeAsset, "the read should relay the quote currency as the order's fee currency")
}

func TestFinishSpotOrderReadRetried(t *testing.T) {
	t.Parallel()
	const orderID = finishedSpotOrderID
	server := &finishedSpotOrdersServer{replies: []string{`{"error":["EService:Unavailable"]}`, queryOrdersReply(t, "closed", 10, 0.6916)}}
	ex := newHTTPTestExchange(t, server.handle(t))
	seedTestAssetNames(ex)
	ex.spotOrders.interval = time.Millisecond
	useConnectedRefreshers(ex)
	msg := &wsChannelMessage{Data: json.RawMessage(`[{"order_id":"` + orderID + `","exec_type":"filled","order_status":"filled","cum_qty":10,"timestamp":"2023-09-22T10:33:05.709993Z"}]`)}
	require.NoError(t, ex.wsProcessExecutions(t.Context(), msg), "wsProcessExecutions must not error")
	var fee float64
	assert.Eventually(t, func() bool {
		for _, payload := range relayed(ex) {
			if d, ok := payload.(*order.Detail); ok && d.Fee != 0 {
				fee = d.Fee
			}
		}
		return fee != 0
	}, 5*time.Second, time.Millisecond, "an order whose read failed should be read again")
	assert.Equal(t, 0.6916, fee, "the read made after a failed one should relay the order's fee")
	assert.Len(t, server.readOrders(), 2, "the order should be read once more after a failed read")
}

func TestFinishSpotOrderMissingFromREST(t *testing.T) {
	t.Parallel()
	const orderID = finishedSpotOrderID
	// REST can leave out an order the websocket has just reported
	server := &finishedSpotOrdersServer{replies: []string{`{"error":[],"result":{}}`, queryOrdersReply(t, "closed", 10, 0.6916)}}
	ex := newHTTPTestExchange(t, server.handle(t))
	seedTestAssetNames(ex)
	ex.spotOrders.interval = time.Millisecond
	useConnectedRefreshers(ex)
	ex.finishSpotOrder(t.Context(), &WebsocketExecution{OrderID: orderID, ExecutionType: "filled", OrderStatus: "filled", CumulativeQuantity: 10})
	var fee float64
	assert.Eventually(t, func() bool {
		for _, payload := range relayed(ex) {
			if d, ok := payload.(*order.Detail); ok && d.Fee != 0 {
				fee = d.Fee
			}
		}
		return fee != 0
	}, 5*time.Second, time.Millisecond, "an order REST left out should be read again")
	assert.Equal(t, 0.6916, fee, "the read made after REST left the order out should relay its fee")
	assert.Len(t, server.readOrders(), 2, "the order should be read once more after REST left it out")
}

func TestFinishSpotOrderRelayFull(t *testing.T) {
	t.Parallel()
	const orderID = finishedSpotOrderID
	server := &finishedSpotOrdersServer{replies: []string{queryOrdersReply(t, "closed", 10, 0.6916), queryOrdersReply(t, "closed", 10, 0.6916)}}
	// The second read waits until the test has emptied the relay the first read's relay found full
	secondRead, resume := make(chan struct{}), make(chan struct{})
	var requests atomic.Int32
	handle := server.handle(t)
	ex := newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) == 2 {
			close(secondRead)
			<-resume
		}
		handle(w, r)
	})
	var resumeOnce sync.Once
	resumeReads := func() { resumeOnce.Do(func() { close(resume) }) }
	// Cleanups run last first, so a held read is let go before the server closes
	t.Cleanup(resumeReads)
	seedTestAssetNames(ex)
	ex.spotOrders.interval = time.Millisecond
	useConnectedRefreshers(ex)
	var relayErr error
	for relayErr == nil {
		relayErr = ex.Websocket.DataHandler.Send(t.Context(), nil)
	}
	ex.finishSpotOrder(t.Context(), &WebsocketExecution{OrderID: orderID, ExecutionType: "filled", OrderStatus: "filled", CumulativeQuantity: 10})
	select {
	case <-secondRead:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "an order whose relay failed must be read again")
	}
	relayed(ex)
	resumeReads()
	var fee float64
	assert.Eventually(t, func() bool {
		for _, payload := range relayed(ex) {
			if d, ok := payload.(*order.Detail); ok && d.Fee != 0 {
				fee = d.Fee
			}
		}
		return fee != 0
	}, 5*time.Second, time.Millisecond, "an order whose relay failed should be read and relayed again")
	assert.Equal(t, 0.6916, fee, "the order should be relayed with its fee once the relay has room")
	assert.Len(t, server.readOrders(), 2, "the order should be read once more after its relay failed")
}

func TestFinishSpotOrderReadOnce(t *testing.T) {
	t.Parallel()
	const orderID = finishedSpotOrderID
	server := &finishedSpotOrdersServer{replies: []string{queryOrdersReply(t, "closed", 10, 0.6916), queryOrdersReply(t, "closed", 10, 0.6916)}}
	ex := newHTTPTestExchange(t, server.handle(t))
	seedTestAssetNames(ex)
	ex.spotOrders.interval = time.Millisecond
	useConnectedRefreshers(ex)
	finished := func(execType string) *wsChannelMessage {
		return &wsChannelMessage{Data: json.RawMessage(`[{"order_id":"` + orderID + `","exec_type":"` + execType + `","order_status":"filled","cum_qty":10,"timestamp":"2023-09-22T10:33:05.709993Z"}]`)}
	}
	require.NoError(t, ex.wsProcessExecutions(t.Context(), finished("trade")), "wsProcessExecutions must not error")
	assert.Eventually(t, func() bool {
		ex.spotOrders.mu.Lock()
		defer ex.spotOrders.mu.Unlock()
		return len(server.readOrders()) == 1 && !ex.spotOrders.running
	}, 5*time.Second, time.Millisecond, "the finished order should be read")
	// The filled event can follow the last fill once the order has already been read
	require.NoError(t, ex.wsProcessExecutions(t.Context(), finished("filled")), "wsProcessExecutions must not error")
	assert.Never(t, func() bool { return len(server.readOrders()) > 1 }, 50*time.Millisecond, time.Millisecond, "an order reported finished again should not be read again")
}

func TestFinishedSpotOrdersReadTogether(t *testing.T) {
	t.Parallel()
	server := &finishedSpotOrdersServer{replies: []string{queryOrdersReply(t, "closed", 10, 0.6916)}, hold: make(chan struct{})}
	ex := newHTTPTestExchange(t, server.handle(t))
	// Cleanups run last first, so a held read is let go before the server closes
	release := sync.OnceFunc(func() { close(server.hold) })
	t.Cleanup(release)
	seedTestAssetNames(ex)
	ex.spotOrders.interval = time.Millisecond
	useConnectedRefreshers(ex)
	finished := func(id string) *WebsocketExecution {
		return &WebsocketExecution{OrderID: id, ExecutionType: "filled", OrderStatus: "filled", CumulativeQuantity: 10}
	}
	ex.finishSpotOrder(t.Context(), finished(finishedSpotOrderID))
	// Orders finishing while a read is held wait for the next read
	require.Eventually(t, func() bool { return len(server.readOrders()) == 1 }, 5*time.Second, time.Millisecond, "the first order must be read")
	ex.finishSpotOrder(t.Context(), finished("OZJW0T-MIQ3C-5NATAU"))
	ex.finishSpotOrder(t.Context(), finished("OAIYAU-LGI3M-PFM5VW"))
	release()
	require.Eventually(t, func() bool { return len(server.readOrders()) >= 2 }, 5*time.Second, time.Millisecond, "the orders finished during a read must be read")
	assert.Equal(t, []string{"OAIYAU-LGI3M-PFM5VW", "OZJW0T-MIQ3C-5NATAU"}, server.readOrders()[1], "the orders finished during a read should be read in one request, in order")
}

func TestFinishSpotOrderWithoutFills(t *testing.T) {
	t.Parallel()
	server := &finishedSpotOrdersServer{}
	ex := newHTTPTestExchange(t, server.handle(t))
	seedTestAssetNames(ex)
	ex.spotOrders.interval = time.Millisecond
	useConnectedRefreshers(ex)
	for _, x := range []string{
		`{"order_id":"OK4GJX-KSTLS-7DZZO5","exec_type":"canceled","order_status":"canceled","cum_qty":0,"timestamp":"2023-09-22T10:33:06Z"}`,
		`{"order_id":"OZJW0T-MIQ3C-5NATAU","exec_type":"expired","order_status":"expired","timestamp":"2023-09-22T10:33:06Z"}`,
		`{"order_id":"OAIYAU-LGI3M-PFM5VW","exec_type":"trade","order_status":"partially_filled","cum_qty":4,"last_qty":4,"timestamp":"2023-09-22T10:33:06Z"}`,
	} {
		require.NoError(t, ex.wsProcessExecutions(t.Context(), &wsChannelMessage{Data: json.RawMessage(`[` + x + `]`)}), "wsProcessExecutions must not error")
	}
	assert.Never(t, func() bool { return len(server.readOrders()) != 0 }, 50*time.Millisecond, time.Millisecond, "an order finished without fills, or still open, should not be read")

	require.NoError(t, ex.wsProcessExecutions(t.Context(), &wsChannelMessage{Data: json.RawMessage(`[{"order_id":"OAIYAU-LGI3M-PFM5VW","exec_type":"canceled","order_status":"canceled","cum_qty":4,"timestamp":"2023-09-22T10:33:07Z"}]`)}), "wsProcessExecutions must not error")
	assert.Eventually(t, func() bool { return len(server.readOrders()) != 0 }, 5*time.Second, time.Millisecond, "an order cancelled after fills should be read")
	assert.Equal(t, []string{"OAIYAU-LGI3M-PFM5VW"}, server.readOrders()[0], "the read should ask for the cancelled order")
}

func TestSpotExecutionFinishedAfterFills(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		execution WebsocketExecution
		exp       bool
	}{
		{WebsocketExecution{ExecutionType: "filled"}, true},
		{WebsocketExecution{ExecutionType: "trade", OrderStatus: "filled"}, true},
		{WebsocketExecution{ExecutionType: "trade", OrderStatus: "partially_filled", CumulativeQuantity: 4}, false},
		{WebsocketExecution{ExecutionType: "canceled", OrderStatus: "canceled", CumulativeQuantity: 4}, true},
		{WebsocketExecution{ExecutionType: "canceled", OrderStatus: "canceled"}, false},
		{WebsocketExecution{ExecutionType: "expired", CumulativeQuantity: 0.0001}, true},
		{WebsocketExecution{OrderStatus: "expired"}, false},
		{WebsocketExecution{ExecutionType: "new", OrderStatus: "new"}, false},
	} {
		assert.Equalf(t, tc.exp, spotExecutionFinishedAfterFills(&tc.execution), "spotExecutionFinishedAfterFills should report %t for %+v", tc.exp, tc.execution)
	}
}
