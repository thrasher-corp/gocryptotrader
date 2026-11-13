package deribit

import (
	"context"
	"net/http"
	"net/http/httptest"
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

// newRateLimitedServerExchange returns an exchange whose REST requests all receive one fixed response body, and a
// context that refuses to wait for a rate limit. Every limit but the non-matching engine's has admitted its one request
// an hour already, so a request sent with the context fails with request.ErrDelayNotAllowed unless it is limited as a
// non-matching engine request
func newRateLimitedServerExchange(t *testing.T, body string) (*Exchange, context.Context) {
	t.Helper()
	ex := new(Exchange)
	require.NoError(t, setupTestExchange(ex), "setupTestExchange must not error")
	ex.Name = t.Name()
	setTestCredentials(ex, &accounts.Credentials{Key: testClientID, Secret: testClientSecret})
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte(body))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	limits := make(request.RateLimitDefinitions)
	for epl := range rateLimits {
		limits[epl] = request.NewRateLimitWithWeight(time.Hour, 1, 1)
		if epl != nonMatchingEPL {
			require.NoError(t, limits[epl].RateLimit(t.Context()), "RateLimit must admit a new limit's first request")
		}
	}
	requester, err := request.New(ex.Name, server.Client(), request.WithLimiter(limits))
	require.NoError(t, err, "request.New must not error")
	require.NoError(t, ex.SetRequester(requester), "SetRequester must not error")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestFutures.String(), server.URL), "SetRunningURL must not error")
	noDelay := request.WithDelayNotAllowed(t.Context())
	err = ex.SendHTTPRequest(noDelay, matchingEPL, "public/get_time", nil, new(types.Time))
	require.ErrorIs(t, err, request.ErrDelayNotAllowed, "a request under an exhausted limit must not be sent")
	return ex, noDelay
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
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Containsf(t, limits, tc.epl, "rateLimits must define endpoint limit %d", tc.epl)
			assert.InDeltaf(t, float64(tc.rate), float64(limits[tc.epl].Limit()), 1e-9, "rateLimits should refill endpoint limit %d at its documented rate", tc.epl)
			assert.Equalf(t, request.Weight(1), limits[tc.epl].Weight(), "rateLimits should weigh endpoint limit %d as one request", tc.epl)
		})
	}
	assert.Len(t, limits, 8, "rateLimits should define only the documented limits")
}
