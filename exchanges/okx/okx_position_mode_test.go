package okx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

// positionModeWaitContext signals when a reader starts its context-aware wait.
//
//nolint:containedctx // This test context instruments waits while preserving cancellation and deadlines.
type positionModeWaitContext struct {
	context.Context
	waiting chan struct{}
}

func (c *positionModeWaitContext) Done() <-chan struct{} {
	select {
	case c.waiting <- struct{}{}:
	default:
	}
	return c.Context.Done()
}

func TestContractPositionMode(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		asset      asset.Item
		body, want string
		err        error
	}{
		{name: "spot needs no lookup", asset: asset.Spot},
		{name: "options needs no lookup", asset: asset.Options},
		{name: "spread needs no lookup", asset: asset.Spread},
		{name: "futures net", asset: asset.Futures, body: `{"code":"0","data":[{"posMode":"net_mode"}]}`, want: "net_mode"},
		{name: "swap hedge", asset: asset.PerpetualSwap, body: `{"code":"0","data":[{"posMode":"long_short_mode"}]}`, want: "long_short_mode"},
		{name: "missing mode", asset: asset.Futures, body: `{"code":"0","data":[{}]}`, err: errInvalidPositionMode},
		{name: "invalid mode", asset: asset.PerpetualSwap, body: `{"code":"0","data":[{"posMode":"invalid"}]}`, err: errInvalidPositionMode},
		{name: "no configuration", asset: asset.Futures, body: `{"code":"0","data":null}`, err: common.ErrNoResponse},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "Setup must succeed")
			ex.API.AuthenticatedSupport = true
			ex.SkipAuthCheck = true
			var requests atomic.Int64
			server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				assert.True(t, strings.HasSuffix(r.URL.Path, "account/config"), "position mode should come from the account configuration")
				_, err := w.Write([]byte(tc.body))
				assert.NoError(t, err, "configuration response should write")
			}))
			require.NoError(t, ex.SetHTTPClient(server.Client()), "mock client must configure")
			require.NoError(t, ex.API.Endpoints.SetRunningURL("RestSpotURL", server.URL+"/"), "mock endpoint must configure")
			actual, err := ex.contractPositionMode(t.Context(), tc.asset)
			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err, "invalid configuration should return its sentinel error")
			} else {
				require.NoError(t, err, "valid configuration must resolve")
				assert.Equal(t, tc.want, actual, "position mode should match the account")
			}
			wantRequests := int64(1)
			if tc.body == "" {
				wantRequests = 0
			}
			assert.Equal(t, wantRequests, requests.Load(), "only contracts should query the account configuration")
		})
	}
	t.Run("external position mode changes are observed", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must succeed")
		ex.API.AuthenticatedSupport = true
		ex.SkipAuthCheck = true
		var requests atomic.Int64
		server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			mode := "net_mode"
			if requests.Add(1) > 1 {
				mode = "long_short_mode"
			}
			_, err := w.Write([]byte(`{"code":"0","data":[{"posMode":"` + mode + `"}]}`))
			assert.NoError(t, err, "configuration response should write")
		}))
		require.NoError(t, ex.SetHTTPClient(server.Client()), "mock client must configure")
		require.NoError(t, ex.API.Endpoints.SetRunningURL("RestSpotURL", server.URL+"/"), "mock endpoint must configure")
		for _, want := range []string{"net_mode", "long_short_mode"} {
			if want == "long_short_mode" {
				ex.positionModeMu.Lock()
				for _, entry := range ex.positionModes {
					entry.expires = time.Now().Add(-time.Second)
				}
				ex.positionModeMu.Unlock()
			}
			actual, err := ex.contractPositionMode(t.Context(), asset.Futures)
			require.NoError(t, err, "updated account configuration must resolve")
			assert.Equal(t, want, actual, "fresh position mode should observe external account changes")
		}
		assert.Equal(t, int64(2), requests.Load(), "expired cache should refresh the account position mode")
	})
	t.Run("lookup errors retain their cause", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must succeed")
		ex.API.AuthenticatedSupport = true
		ex.SkipAuthCheck = true
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		_, err := ex.contractPositionMode(ctx, asset.Futures)
		assert.ErrorIs(t, err, context.Canceled, "failed account lookup should retain the cancellation cause")
	})
	t.Run("coalesces lookups and permits waiter cancellation", func(t *testing.T) {
		t.Parallel()
		entered, release := make(chan struct{}), make(chan struct{})
		var requests atomic.Int64
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "setup must succeed")
		ex.API.AuthenticatedSupport, ex.SkipAuthCheck = true, true
		if err := ex.DisableRateLimiter(); err != nil {
			require.ErrorIs(t, err, request.ErrRateLimiterAlreadyDisabled, "mock limiter must only report already disabled")
		}
		server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if requests.Add(1) == 1 {
				close(entered)
			}
			select {
			case <-release:
			case <-r.Context().Done():
				return
			}
			_, err := w.Write([]byte(`{"code":"0","data":[{"posMode":"net_mode"}]}`))
			assert.NoError(t, err, "response should write")
		}))
		require.NoError(t, ex.SetHTTPClient(server.Client()), "client must configure")
		require.NoError(t, ex.API.Endpoints.SetRunningURL("RestSpotURL", server.URL+"/"), "endpoint must configure")
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()
		results := make(chan error, 20)
		for range 20 {
			go func() {
				mode, err := ex.contractPositionMode(ctx, asset.Futures)
				if err == nil {
					assert.Equal(t, "net_mode", mode, "all callers should share the result")
				}
				results <- err
			}()
		}
		select {
		case <-entered:
		case <-ctx.Done():
			require.FailNow(t, "lookup must start")
		}
		waitCtx, cancelWait := context.WithCancel(ctx)
		waitResult := make(chan error, 1)
		go func() { _, err := ex.contractPositionMode(waitCtx, asset.Futures); waitResult <- err }()
		cancelWait()
		select {
		case err := <-waitResult:
			assert.ErrorIs(t, err, context.Canceled, "waiter cancellation should return promptly")
		case <-ctx.Done():
			require.FailNow(t, "cancelled waiter must return")
		}
		close(release)
		for range 20 {
			select {
			case err := <-results:
				require.NoError(t, err, "coalesced lookup must succeed")
			case <-ctx.Done():
				require.FailNow(t, "all readers must finish")
			}
		}
		assert.Equal(t, int64(1), requests.Load(), "concurrent readers should share one GET")
	})
	t.Run("credentials isolate cached account modes", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "setup must succeed")
		ex.API.AuthenticatedSupport, ex.SkipAuthCheck = true, true
		if err := ex.DisableRateLimiter(); err != nil {
			require.ErrorIs(t, err, request.ErrRateLimiterAlreadyDisabled, "mock limiter must only report already disabled")
		}
		var requests atomic.Int64
		server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			mode := "net_mode"
			if requests.Add(1)%2 == 0 {
				mode = "long_short_mode"
			}
			_, err := w.Write([]byte(`{"code":"0","data":[{"posMode":"` + mode + `"}]}`))
			assert.NoError(t, err, "response should write")
		}))
		require.NoError(t, ex.SetHTTPClient(server.Client()), "client must configure")
		require.NoError(t, ex.API.Endpoints.SetRunningURL("RestSpotURL", server.URL+"/"), "endpoint must configure")
		for i, creds := range []accounts.Credentials{{Key: "a", Secret: "one", ClientID: "pass"}, {Key: "a", Secret: "two", ClientID: "pass"}, {Key: "a", Secret: "two", ClientID: "other"}, {Key: "b", Secret: "two", ClientID: "other"}} {
			store := &accounts.ContextCredentialsStore{}
			store.Load(&creds)
			ctx := context.WithValue(t.Context(), accounts.ContextCredentialsFlag, store)
			want := "net_mode"
			if i%2 != 0 {
				want = "long_short_mode"
			}
			for range 2 {
				mode, err := ex.contractPositionMode(ctx, asset.Futures)
				require.NoError(t, err, "credential-specific mode must resolve")
				assert.Equal(t, want, mode, "each complete credential set should retain its own mode")
			}
		}
		assert.Equal(t, int64(4), requests.Load(), "each complete credential set should fetch once")
	})
	t.Run("failed configuration is not cached", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "setup must succeed")
		ex.API.AuthenticatedSupport, ex.SkipAuthCheck = true, true
		if err := ex.DisableRateLimiter(); err != nil {
			require.ErrorIs(t, err, request.ErrRateLimiterAlreadyDisabled, "mock limiter must only report already disabled")
		}
		var requests atomic.Int64
		server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			mode := "invalid"
			if requests.Add(1) > 1 {
				mode = "net_mode"
			}
			_, err := w.Write([]byte(`{"code":"0","data":[{"posMode":"` + mode + `"}]}`))
			assert.NoError(t, err, "response should write")
		}))
		require.NoError(t, ex.SetHTTPClient(server.Client()), "client must configure")
		require.NoError(t, ex.API.Endpoints.SetRunningURL("RestSpotURL", server.URL+"/"), "endpoint must configure")
		mode, err := ex.contractPositionMode(t.Context(), asset.Futures)
		assert.ErrorIs(t, err, errInvalidPositionMode, "invalid mode should retain its sentinel")
		assert.Empty(t, mode, "invalid mode should not escape on error")
		mode, err = ex.contractPositionMode(t.Context(), asset.Futures)
		require.NoError(t, err, "next reader must retry failed configuration")
		assert.Equal(t, "net_mode", mode, "successful retry should supply the mode")
		assert.Equal(t, int64(2), requests.Load(), "failed results should not suppress a retry")
	})
	t.Run("coalesced failure reaches existing waiters", func(t *testing.T) {
		t.Parallel()
		entered, release := make(chan struct{}), make(chan struct{})
		var requests atomic.Int64
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "setup must succeed")
		ex.API.AuthenticatedSupport, ex.SkipAuthCheck = true, true
		if err := ex.DisableRateLimiter(); err != nil {
			require.ErrorIs(t, err, request.ErrRateLimiterAlreadyDisabled, "mock limiter must only report already disabled")
		}
		server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if requests.Add(1) == 1 {
				close(entered)
				select {
				case <-release:
				case <-r.Context().Done():
					return
				}
			}
			_, err := w.Write([]byte(`{"code":"0","data":[{"posMode":"invalid"}]}`))
			assert.NoError(t, err, "response should write")
		}))
		require.NoError(t, ex.SetHTTPClient(server.Client()), "client must configure")
		require.NoError(t, ex.API.Endpoints.SetRunningURL("RestSpotURL", server.URL+"/"), "endpoint must configure")
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()
		results := make(chan error, 2)
		go func() { _, err := ex.contractPositionMode(ctx, asset.Futures); results <- err }()
		select {
		case <-entered:
		case <-ctx.Done():
			require.FailNow(t, "first lookup must start")
		}
		waitCtx := &positionModeWaitContext{Context: ctx, waiting: make(chan struct{}, 1)}
		go func() { _, err := ex.contractPositionMode(waitCtx, asset.Futures); results <- err }()
		select {
		case <-waitCtx.waiting:
		case <-ctx.Done():
			require.FailNow(t, "second reader must join the flight")
		}
		close(release)
		for range 2 {
			select {
			case err := <-results:
				assert.ErrorIs(t, err, errInvalidPositionMode, "existing readers should receive the shared failure")
			case <-ctx.Done():
				require.FailNow(t, "shared failure must return")
			}
		}
		assert.Equal(t, int64(1), requests.Load(), "existing waiters should not issue serial retries")
		_, err := ex.contractPositionMode(ctx, asset.Futures)
		assert.ErrorIs(t, err, errInvalidPositionMode, "a new caller should retry the failed lookup")
		assert.Equal(t, int64(2), requests.Load(), "errors should not be cached for new callers")
	})
	t.Run("cancelled flight releases other readers", func(t *testing.T) {
		t.Parallel()
		entered := make(chan struct{})
		var requests atomic.Int64
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "setup must succeed")
		ex.API.AuthenticatedSupport, ex.SkipAuthCheck = true, true
		if err := ex.DisableRateLimiter(); err != nil {
			require.ErrorIs(t, err, request.ErrRateLimiterAlreadyDisabled, "mock limiter must only report already disabled")
		}
		server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if requests.Add(1) == 1 {
				close(entered)
				<-r.Context().Done()
				return
			}
			_, err := w.Write([]byte(`{"code":"0","data":[{"posMode":"net_mode"}]}`))
			assert.NoError(t, err, "response should write")
		}))
		require.NoError(t, ex.SetHTTPClient(server.Client()), "client must configure")
		require.NoError(t, ex.API.Endpoints.SetRunningURL("RestSpotURL", server.URL+"/"), "endpoint must configure")
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()
		leaderCtx, cancelLeader := context.WithCancel(ctx)
		defer cancelLeader()
		leaderResult, result := make(chan error, 1), make(chan error, 1)
		go func() { _, err := ex.contractPositionMode(leaderCtx, asset.Futures); leaderResult <- err }()
		select {
		case <-entered:
		case <-ctx.Done():
			require.FailNow(t, "first lookup must start")
		}
		waitCtx := &positionModeWaitContext{Context: ctx, waiting: make(chan struct{}, 1)}
		go func() {
			mode, err := ex.contractPositionMode(waitCtx, asset.Futures)
			assert.Equal(t, "net_mode", mode, "remaining reader should obtain a fresh mode")
			result <- err
		}()
		select {
		case <-waitCtx.waiting:
		case <-ctx.Done():
			require.FailNow(t, "second reader must join the flight")
		}
		cancelLeader()
		select {
		case err := <-leaderResult:
			assert.ErrorIs(t, err, context.Canceled, "leader should retain cancellation")
		case <-ctx.Done():
			require.FailNow(t, "cancelled leader must return")
		}
		select {
		case err := <-result:
			require.NoError(t, err, "remaining reader must retry a cancelled flight")
		case <-ctx.Done():
			require.FailNow(t, "remaining reader must not remain stranded")
		}
	})
	t.Run("mode mutations block reads and supersede earlier flights", func(t *testing.T) {
		t.Parallel()
		getEntered, oldRelease := make(chan struct{}), make(chan struct{})
		posts := []chan struct{}{make(chan struct{}), make(chan struct{})}
		postEntered := make(chan int, 2)
		var gets, mutations atomic.Int64
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "setup must succeed")
		ex.API.AuthenticatedSupport, ex.SkipAuthCheck = true, true
		if err := ex.DisableRateLimiter(); err != nil {
			require.ErrorIs(t, err, request.ErrRateLimiterAlreadyDisabled, "mock limiter must only report already disabled")
		}
		server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mode := "long_short_mode"
			if strings.HasSuffix(r.URL.Path, "set-position-mode") {
				i := int(mutations.Add(1) - 1)
				postEntered <- i
				select {
				case <-posts[i]:
				case <-r.Context().Done():
					return
				}
			} else if gets.Add(1) == 1 {
				mode = "net_mode"
				close(getEntered)
				select {
				case <-oldRelease:
				case <-r.Context().Done():
					return
				}
			}
			_, err := w.Write([]byte(`{"code":"0","data":[{"posMode":"` + mode + `"}]}`))
			assert.NoError(t, err, "response should write")
		}))
		require.NoError(t, ex.SetHTTPClient(server.Client()), "client must configure")
		require.NoError(t, ex.API.Endpoints.SetRunningURL("RestSpotURL", server.URL+"/"), "endpoint must configure")
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()
		type result struct {
			mode string
			err  error
		}
		reads := make(chan result, 2)
		changed := make(chan error, 2)
		go func() { mode, err := ex.contractPositionMode(ctx, asset.Futures); reads <- result{mode, err} }()
		select {
		case <-getEntered:
		case <-ctx.Done():
			require.FailNow(t, "old configuration lookup must begin")
		}
		for range 2 {
			go func() { _, err := ex.SetPositionMode(ctx, "long_short_mode"); changed <- err }()
		}
		for range 2 {
			select {
			case <-postEntered:
			case <-ctx.Done():
				require.FailNow(t, "both mutations must start")
			}
		}
		waitCtx := &positionModeWaitContext{Context: ctx, waiting: make(chan struct{}, 1)}
		go func() { mode, err := ex.contractPositionMode(waitCtx, asset.Futures); reads <- result{mode, err} }()
		select {
		case <-waitCtx.waiting:
		case <-ctx.Done():
			require.FailNow(t, "new reader must wait for mutation")
		}
		close(oldRelease)
		close(posts[0])
		select {
		case err := <-changed:
			require.NoError(t, err, "first mutation must complete")
		case <-ctx.Done():
			require.FailNow(t, "first mutation must return")
		}
		select {
		case <-reads:
			require.FailNow(t, "reads must remain blocked while another mutation is pending")
		default:
		}
		close(posts[1])
		select {
		case err := <-changed:
			require.NoError(t, err, "second mutation must complete")
		case <-ctx.Done():
			require.FailNow(t, "second mutation must return")
		}
		for range 2 {
			select {
			case got := <-reads:
				require.NoError(t, got.err, "read must resume after mutation")
				assert.Equal(t, "long_short_mode", got.mode, "superseded lookup should not repopulate the old mode")
			case <-ctx.Done():
				require.FailNow(t, "reads must resume")
			}
		}
		assert.Equal(t, int64(2), gets.Load(), "new readers should coalesce the post-mutation refresh")
	})
	t.Run("failed mode mutation invalidates cached configuration", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "setup must succeed")
		ex.API.AuthenticatedSupport, ex.SkipAuthCheck = true, true
		if err := ex.DisableRateLimiter(); err != nil {
			require.ErrorIs(t, err, request.ErrRateLimiterAlreadyDisabled, "mock limiter must only report already disabled")
		}
		var gets atomic.Int64
		server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body := `{"code":"50000","msg":"mode mutation failed","data":[]}`
			if strings.HasSuffix(r.URL.Path, "account/config") {
				mode := "net_mode"
				if gets.Add(1) > 1 {
					mode = "long_short_mode"
				}
				body = `{"code":"0","data":[{"posMode":"` + mode + `"}]}`
			}
			_, err := w.Write([]byte(body))
			assert.NoError(t, err, "response should write")
		}))
		require.NoError(t, ex.SetHTTPClient(server.Client()), "client must configure")
		require.NoError(t, ex.API.Endpoints.SetRunningURL("RestSpotURL", server.URL+"/"), "endpoint must configure")
		mode, err := ex.contractPositionMode(t.Context(), asset.Futures)
		require.NoError(t, err, "initial mode must resolve")
		assert.Equal(t, "net_mode", mode, "initial account should be net")
		_, err = ex.SetPositionMode(t.Context(), "long_short_mode")
		assert.Error(t, err, "mutation should return its venue failure")
		mode, err = ex.contractPositionMode(t.Context(), asset.Futures)
		require.NoError(t, err, "mode must refresh after an uncertain mutation")
		assert.Equal(t, "long_short_mode", mode, "failed request should not retain a potentially stale mode")
		assert.Equal(t, int64(2), gets.Load(), "mutation should invalidate the previous cache")
	})
}

func TestChangePositionModeCache(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	creds := accounts.Credentials{Key: "test"}
	ex.changePositionModeCache(&creds, true)
	entry := ex.positionModes[creds]
	first := entry.changed
	ex.changePositionModeCache(&creds, true)
	assert.Equal(t, 2, entry.changing, "overlapping mutations should share the barrier")
	assert.Same(t, entry, ex.positionModes[creds], "overlapping mutations should retain the entry")
	ex.changePositionModeCache(&creds, false)
	select {
	case <-first:
		require.FailNow(t, "first completion must not release other pending mutations")
	default:
	}
	ex.changePositionModeCache(&creds, false)
	select {
	case <-first:
	default:
		require.FailNow(t, "last completion must release readers")
	}
	assert.Nil(t, entry.changed, "completed barrier should be removed")
	assert.Zero(t, entry.expires, "mode changes should invalidate cached configuration")
	assert.Equal(t, uint64(4), entry.generation, "each mutation boundary should supersede earlier lookups")
}

func TestTrimPositionModeCache(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	ex.positionModes = make(map[accounts.Credentials]*positionModeCacheEntry)
	for i := range positionModeCacheLimit + 3 {
		ex.positionModes[accounts.Credentials{Key: strconv.Itoa(i)}] = &positionModeCacheEntry{expires: time.Now().Add(time.Duration(i) * time.Second)}
	}
	active := accounts.Credentials{Key: "active"}
	ex.positionModes[active] = &positionModeCacheEntry{ready: make(chan struct{})}
	changing := accounts.Credentials{Key: "changing"}
	ex.positionModes[changing] = &positionModeCacheEntry{changing: 1}
	ex.trimPositionModeCache()
	assert.Len(t, ex.positionModes, positionModeCacheLimit, "completed entries should fit the retention limit")
	assert.Contains(t, ex.positionModes, active, "pending lookups should remain reachable")
	assert.Contains(t, ex.positionModes, changing, "pending mutations should remain reachable")
	ex.positionModes = make(map[accounts.Credentials]*positionModeCacheEntry)
	for i := range positionModeCacheLimit + 1 {
		ex.positionModes[accounts.Credentials{Key: strconv.Itoa(i)}] = &positionModeCacheEntry{ready: make(chan struct{})}
	}
	ex.trimPositionModeCache()
	assert.Len(t, ex.positionModes, positionModeCacheLimit+1, "active requests should not be evicted merely to enforce retention")
}

func TestPositionModeWaitContext(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	waitCtx := &positionModeWaitContext{Context: ctx, waiting: make(chan struct{}, 1)}
	assert.Equal(t, ctx.Done(), waitCtx.Done(), "instrumented context should preserve cancellation")
	select {
	case <-waitCtx.waiting:
	default:
		require.FailNow(t, "Done must signal the wait")
	}
}
