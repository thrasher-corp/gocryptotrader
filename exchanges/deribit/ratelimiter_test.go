package deribit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
	"golang.org/x/time/rate"
)

// newLimitedServerExchange returns an exchange whose REST requests the handler serves, and a context that refuses to
// wait for a rate limit. Each endpoint limit admits at once the number of requests tokens gives it, and no others, or
// every request for a negative number
func newLimitedServerExchange(t *testing.T, handler http.Handler, tokens map[request.EndpointLimit]int) (*Exchange, context.Context) {
	t.Helper()
	ex := new(Exchange)
	require.NoError(t, setupTestExchange(ex), "setupTestExchange must not error")
	ex.Name = t.Name()
	setTestCredentials(ex, &accounts.Credentials{Key: testClientID, Secret: testClientSecret})
	server := httptest.NewTestServer(t, handler)
	limits := make(request.RateLimitDefinitions, len(rateLimits))
	for key := range rateLimits {
		epl, ok := key.(request.EndpointLimit)
		require.Truef(t, ok, "rateLimits must key its limits by endpoint limit, not %T", key)
		limits[epl] = request.NewRateLimitWithWeight(0, 0, 1)
		if n := tokens[epl]; n >= 0 {
			limits[epl] = request.GetRateLimiterWithWeight(rate.NewLimiter(rate.Every(time.Hour), n), 1)
		}
	}
	requester, err := request.New(ex.Name, server.Client(), request.WithLimiter(limits))
	require.NoError(t, err, "request.New must not error")
	require.NoError(t, ex.SetRequester(requester), "SetRequester must not error")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestFutures.String(), server.URL), "SetRunningURL must not error")
	return ex, request.WithDelayNotAllowed(t.Context())
}

// newRateLimitedServerExchange returns an exchange whose REST requests all receive one fixed response body, and a
// context that refuses to wait for a rate limit. Only the non-matching engine limit admits a request, so a request sent
// with the context fails with request.ErrDelayNotAllowed unless it is limited as a non-matching engine request
func newRateLimitedServerExchange(t *testing.T, body string) (*Exchange, context.Context) {
	t.Helper()
	ex, noDelay := newLimitedServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte(body))
		assert.NoError(t, err, "Writing the response should not error")
	}), map[request.EndpointLimit]int{nonMatchingEPL: 1})
	err := ex.SendHTTPRequest(noDelay, matchingEPL, "public/get_time", nil, new(types.Time))
	require.ErrorIs(t, err, request.ErrDelayNotAllowed, "a request under an exhausted limit must not be sent")
	return ex, noDelay
}

// tickerOrderBookLimitCases are the endpoint limit capacities a ticker or order book request is tried with, and the
// requests sent and error each gives
var tickerOrderBookLimitCases = []struct {
	name   string
	tokens map[request.EndpointLimit]int
	sent   int32
	err    error
}{
	{name: "within both limits", tokens: map[request.EndpointLimit]int{tickerOrderBookEPL: 1, nonMatchingEPL: 1}, sent: 1},
	{name: "ticker and order book limit reached", tokens: map[request.EndpointLimit]int{nonMatchingEPL: 1}, err: request.ErrDelayNotAllowed},
	{name: "non-matching engine limit reached", tokens: map[request.EndpointLimit]int{tickerOrderBookEPL: 1}, err: request.ErrDelayNotAllowed},
}

// testTickerOrderBookLimits checks a ticker or order book request, sent by send and answered with body, waits on the
// ticker and order book limit and the non-matching engine limit, and on no other
func testTickerOrderBookLimits(t *testing.T, body string, send func(context.Context, *Exchange) error) {
	t.Helper()
	for _, tc := range tickerOrderBookLimitCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var sent atomic.Int32
			ex, noDelay := newLimitedServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				sent.Add(1)
				_, err := w.Write([]byte(body))
				assert.NoError(t, err, "Writing the response should not error")
			}), tc.tokens)
			assert.ErrorIs(t, send(noDelay, ex), tc.err, "the request should wait on the ticker and order book limit and the non-matching engine limit alone")
			assert.Equal(t, tc.sent, sent.Load(), "the request should be sent only within both limits")
		})
	}
}

// testWebsocketTickerOrderBookLimits checks a ticker or order book request sent over the websocket by send, which the
// mock venue answers for method with result, waits on the ticker and order book limit and the non-matching engine
// limit, and on no other
func testWebsocketTickerOrderBookLimits(t *testing.T, method, result string, send func(context.Context, *Exchange) error) {
	t.Helper()
	if !mockTests {
		t.Skip("the mock websocket venue's requests are checked in mock mode")
	}
	for _, tc := range tickerOrderBookLimitCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex, venue := newWebsocketTestExchange(t)
			limited, noDelay := newLimitedServerExchange(t, http.NotFoundHandler(), tc.tokens)
			require.NoError(t, ex.SetRequester(limited.Requester), "SetRequester must not error")
			venue.answerNext(method, result)
			assert.ErrorIs(t, send(noDelay, ex), tc.err, "the request should wait on the ticker and order book limit and the non-matching engine limit alone")
			assert.Len(t, venue.received(method), int(tc.sent), "the request should be sent only within both limits")
		})
	}
}

func TestRateLimits(t *testing.T) {
	t.Parallel()
	limits := rateLimits
	for _, tc := range []struct {
		name string
		epl  request.EndpointLimit
		rate rate.Limit
	}{
		{name: "non-matching engine", epl: nonMatchingEPL, rate: 20},
		{name: "matching engine", epl: matchingEPL, rate: 5},
		{name: "instruments", epl: getInstrumentsEPL, rate: 1},
		{name: "subscriptions", epl: subscribeEPL, rate: 10.0 / 3},
		{name: "position moves", epl: movePositionsEPL, rate: 6.0 / 60},
		{name: "transaction log", epl: transactionLogEPL, rate: 1},
		{name: "mass quotes", epl: massQuoteEPL, rate: 10},
		{name: "portfolio simulation", epl: simulatePortfolioEPL, rate: 1},
		{name: "tickers and order books", epl: tickerOrderBookEPL, rate: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Containsf(t, limits, tc.epl, "rateLimits must define endpoint limit %d", tc.epl)
			assert.InDeltaf(t, float64(tc.rate), float64(limits[tc.epl].Limit()), 1e-9, "rateLimits should refill endpoint limit %d at its documented rate", tc.epl)
			assert.Equalf(t, request.Weight(1), limits[tc.epl].Weight(), "rateLimits should weigh endpoint limit %d as one request", tc.epl)
		})
	}
	assert.Len(t, limits, 9, "rateLimits should define only the documented limits")
}

func TestAttemptRateLimits(t *testing.T) {
	t.Parallel()
	for key := range rateLimits {
		epl, ok := key.(request.EndpointLimit)
		require.Truef(t, ok, "rateLimits must key its limits by endpoint limit, not %T", key)
		t.Run(strconv.FormatUint(uint64(epl), 10), func(t *testing.T) {
			t.Parallel()
			exp := []request.EndpointLimit{epl}
			if epl == tickerOrderBookEPL {
				exp = []request.EndpointLimit{nonMatchingEPL, tickerOrderBookEPL}
			}
			assert.Equal(t, exp, attemptRateLimits(epl), "attemptRateLimits should have a ticker or order book request wait on the non-matching engine limit and then on their shared limit, and any other request on its own limit")
		})
	}
}

func TestInitiateRateLimits(t *testing.T) {
	t.Parallel()

	t.Run("each limit in turn", func(t *testing.T) {
		t.Parallel()
		ex, noDelay := newLimitedServerExchange(t, http.NotFoundHandler(), map[request.EndpointLimit]int{nonMatchingEPL: 1, tickerOrderBookEPL: 1})
		require.NoError(t, ex.initiateRateLimits(noDelay, []request.EndpointLimit{nonMatchingEPL, tickerOrderBookEPL}), "initiateRateLimits must admit a request both limits admit")
		assert.ErrorIs(t, ex.initiateRateLimits(noDelay, []request.EndpointLimit{nonMatchingEPL}), request.ErrDelayNotAllowed, "initiateRateLimits should take the first limit's capacity")
		assert.ErrorIs(t, ex.initiateRateLimits(noDelay, []request.EndpointLimit{tickerOrderBookEPL}), request.ErrDelayNotAllowed, "initiateRateLimits should take the second limit's capacity")
	})

	t.Run("stopped by a refusal", func(t *testing.T) {
		t.Parallel()
		ex, noDelay := newLimitedServerExchange(t, http.NotFoundHandler(), map[request.EndpointLimit]int{nonMatchingEPL: 0, tickerOrderBookEPL: 1})
		assert.ErrorIs(t, ex.initiateRateLimits(noDelay, []request.EndpointLimit{nonMatchingEPL, tickerOrderBookEPL}), request.ErrDelayNotAllowed, "initiateRateLimits should return the first limit's refusal")
		assert.NoError(t, ex.initiateRateLimits(noDelay, []request.EndpointLimit{tickerOrderBookEPL}), "initiateRateLimits should leave a later limit's capacity once an earlier limit refuses")
	})

	t.Run("no limits", func(t *testing.T) {
		t.Parallel()
		ex, noDelay := newLimitedServerExchange(t, http.NotFoundHandler(), nil)
		assert.NoError(t, ex.initiateRateLimits(noDelay, nil), "initiateRateLimits should wait on nothing without limits")
	})

	t.Run("rate limiter disabled", func(t *testing.T) {
		t.Parallel()
		ex, noDelay := newLimitedServerExchange(t, http.NotFoundHandler(), nil)
		require.NoError(t, ex.DisableRateLimiter(), "DisableRateLimiter must not error")
		assert.NoError(t, ex.initiateRateLimits(noDelay, []request.EndpointLimit{nonMatchingEPL, tickerOrderBookEPL}), "initiateRateLimits should not wait once the rate limiter is disabled")
	})

	t.Run("no rate limits", func(t *testing.T) {
		t.Parallel()
		ex, noDelay := newLimitedServerExchange(t, http.NotFoundHandler(), nil)
		requester, err := request.New(ex.Name, new(http.Client))
		require.NoError(t, err, "request.New must not error")
		require.NoError(t, ex.SetRequester(requester), "SetRequester must not error")
		assert.NoError(t, ex.initiateRateLimits(noDelay, []request.EndpointLimit{nonMatchingEPL, tickerOrderBookEPL}), "initiateRateLimits should not wait without rate limits, as the requester sends without them")
	})
}
