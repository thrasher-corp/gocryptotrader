package binance

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// GetC2CTradeHistory calls Get C2C Trade History. The API documents this USER_DATA endpoint as unsigned, taking only
// the API key; without a window it returns the last 30 days, and it only reaches 6 months back
func (e *Exchange) GetC2CTradeHistory(ctx context.Context, arg *C2CTradeHistoryRequest) (*C2CTradeHistoryResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if !arg.StartTime.IsZero() && !arg.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(arg.StartTime, arg.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if arg.TradeType != "" {
		params.Set("tradeType", arg.TradeType)
	}
	if !arg.StartTime.IsZero() {
		params.Set("startTimestamp", strconv.FormatInt(arg.StartTime.UnixMilli(), 10))
	}
	if !arg.EndTime.IsZero() {
		params.Set("endTimestamp", strconv.FormatInt(arg.EndTime.UnixMilli(), 10))
	}
	if arg.Page > 0 {
		params.Set("page", strconv.FormatUint(arg.Page, 10))
	}
	if arg.Rows > 0 {
		params.Set("rows", strconv.FormatUint(arg.Rows, 10))
	}
	var resp *C2CTradeHistoryResponse
	return resp, e.SendAPIKeyHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, common.EncodeURLValues("/sapi/v1/c2c/orderMatch/listUserOrderHistory", params), sapiDefaultRate, &resp)
}
