package binance

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

func TestRateLimit_Limit(t *testing.T) {
	t.Parallel()
	symbol := "BTC-USDT"

	testTable := map[string]struct {
		Expected request.EndpointLimit
		Limit    request.EndpointLimit
		Deadline time.Time
	}{
		"Open Orders":          {Expected: spotOpenOrdersSpecificRate, Limit: openOrdersLimit(symbol)},
		"Orderbook Depth 5":    {Expected: spotOrderbookDepth100Rate, Limit: orderbookLimit(5)},
		"Orderbook Depth 10":   {Expected: spotOrderbookDepth100Rate, Limit: orderbookLimit(10)},
		"Orderbook Depth 20":   {Expected: spotOrderbookDepth100Rate, Limit: orderbookLimit(20)},
		"Orderbook Depth 50":   {Expected: spotOrderbookDepth100Rate, Limit: orderbookLimit(50)},
		"Orderbook Depth 100":  {Expected: spotOrderbookDepth100Rate, Limit: orderbookLimit(100)},
		"Orderbook Depth 500":  {Expected: spotOrderbookDepth500Rate, Limit: orderbookLimit(500)},
		"Orderbook Depth 1000": {Expected: spotOrderbookDepth1000Rate, Limit: orderbookLimit(1000)},
		"Orderbook Depth 5000": {Expected: spotOrderbookDepth5000Rate, Limit: orderbookLimit(5000)},
		"Exceeds deadline":     {Expected: spotOrderbookDepth5000Rate, Limit: orderbookLimit(5000), Deadline: time.Now().Add(time.Nanosecond)},
	}

	rl, err := request.New("rateLimitTest", &http.Client{}, request.WithLimiter(GetRateLimits()))
	require.NoError(t, err)

	for name, tt := range testTable {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			exp, got := tt.Expected, tt.Limit
			if exp != got {
				t.Fatalf("incorrect limit applied.\nexp: %v\ngot: %v", exp, got)
			}

			ctx := t.Context()
			if !tt.Deadline.IsZero() {
				var cancel context.CancelFunc
				ctx, cancel = context.WithDeadline(ctx, tt.Deadline)
				defer cancel()
			}

			err := rl.InitiateRateLimit(ctx, tt.Limit)
			require.Truef(t, err == nil || errors.Is(err, context.DeadlineExceeded), "InitiateRateLimit must not error: %s", err)
		})
	}
}

func TestRateLimit_LimitStatic(t *testing.T) {
	t.Parallel()
	testTable := map[string]request.EndpointLimit{
		"Default":           spotDefaultRate,
		"Historical Trades": spotHistoricalTradesRate,
		"All Price Changes": spotTickerAllRate,
		"All Orders":        spotAllOrdersRate,
	}

	rl, err := request.New("rateLimitTest2", http.DefaultClient, request.WithLimiter(GetRateLimits()))
	require.NoError(t, err)

	for name, tt := range testTable {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if err := rl.InitiateRateLimit(t.Context(), tt); err != nil {
				t.Fatalf("error applying rate limit: %v", err)
			}
		})
	}
}
