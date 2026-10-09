package kraken

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// GetPreTradeData calls Pre-Trade Data, returning the top 10 aggregated levels of a pair's order book
func (e *Exchange) GetPreTradeData(ctx context.Context, pair currency.Pair) (*PreTradeDataResponse, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	symbol, err := e.FormatSymbol(pair, asset.Spot)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *PreTradeDataResponse
	return resp, e.SendHTTPRequest(ctx, common.EncodeURLValues("/0/public/PreTrade", params), &resp)
}

// GetPostTradeData calls Post-Trade Data, returning trades in ascending time order. A nil request returns the last
// 1000 trades of every pair
func (e *Exchange) GetPostTradeData(ctx context.Context, req *PostTradeDataRequest) (*PostTradeDataResponse, error) {
	params := url.Values{}
	if req != nil {
		if req.Count > 1000 {
			return nil, fmt.Errorf("%w: %d exceeds 1000", errInvalidCount, req.Count)
		}
		if !req.From.IsZero() && !req.To.IsZero() && req.From.After(req.To) {
			return nil, common.ErrStartAfterEnd
		}
		if !req.Pair.IsEmpty() {
			symbol, err := e.FormatSymbol(req.Pair, asset.Spot)
			if err != nil {
				return nil, err
			}
			params.Set("symbol", symbol)
		}
		if !req.From.IsZero() {
			params.Set("from_ts", req.From.UTC().Format(time.RFC3339Nano))
		}
		if !req.To.IsZero() {
			params.Set("to_ts", req.To.UTC().Format(time.RFC3339Nano))
		}
		if req.Count != 0 {
			params.Set("count", strconv.FormatUint(req.Count, 10))
		}
	}
	var resp *PostTradeDataResponse
	return resp, e.SendHTTPRequest(ctx, common.EncodeURLValues("/0/public/PostTrade", params), &resp)
}
