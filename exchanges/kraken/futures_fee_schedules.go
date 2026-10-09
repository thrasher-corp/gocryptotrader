package kraken

import (
	"context"
	"net/http"

	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// GetFuturesFeeSchedules calls Get fee schedules. Kraken deprecated it on 22 June 2026, since when its fees no longer
// reflect the fees charged on futures trades; GetTradeVolume returns those
func (e *Exchange) GetFuturesFeeSchedules(ctx context.Context) (*FuturesFeeSchedulesResponse, error) {
	var resp *FuturesFeeSchedulesResponse
	return resp, e.SendFuturesHTTPRequest(ctx, exchange.RestFutures, "/api/v3/feeschedules", &resp)
}

// GetFuturesFeeScheduleVolumes calls Get fee schedule volumes, returning the account's 30 day volume for each fee
// schedule. Kraken deprecated it on 22 June 2026, since when its volumes no longer determine the fees charged on
// futures trades; GetTradeVolume returns those
func (e *Exchange) GetFuturesFeeScheduleVolumes(ctx context.Context) (*FuturesFeeScheduleVolumesResponse, error) {
	var resp *FuturesFeeScheduleVolumesResponse
	return resp, e.SendFuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, "/api/v3/feeschedules/volumes", nil, nil, &resp)
}
