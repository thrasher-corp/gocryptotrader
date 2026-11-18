package binance

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// GetSpotRebateHistoryRecords calls Get Spot Rebate History Records for a window of at most 30 days from 2020-06-10
// onwards; without a window the API returns the last 7 days, at most 200 records a page
func (e *Exchange) GetSpotRebateHistoryRecords(ctx context.Context, startTime, endTime time.Time, page uint64) (*RebateHistoryResponse, error) {
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	var resp *RebateHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/rebate/taxQuery", params, spotRebateHistoryRate, &resp)
}
