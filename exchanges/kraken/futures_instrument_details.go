package kraken

import (
	"context"
	"net/http"
	"net/url"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// GetFuturesInstruments calls Get instruments, returning the specifications of the listed markets. A nil request
// returns every listed futures market, which excludes options
func (e *Exchange) GetFuturesInstruments(ctx context.Context, req *FuturesInstrumentsRequest) (*FuturesInstrumentsResponse, error) {
	var params url.Values
	if req != nil {
		params = futuresContractTypeParams(req.ContractTypes)
		if req.Expired {
			params.Set("expired", "true")
		}
	}
	var resp *FuturesInstrumentsResponse
	return resp, e.SendFuturesHTTPRequest(ctx, exchange.RestFutures, common.EncodeURLValues("/api/v3/instruments", params), &resp)
}

// GetFuturesTradingInstruments calls Get trading instruments, returning the specifications of the markets the account
// can access with the margin levels that apply to it. A nil request returns every futures market, which excludes
// options
func (e *Exchange) GetFuturesTradingInstruments(ctx context.Context, req *FuturesTradingInstrumentsRequest) (*FuturesTradingInstrumentsResponse, error) {
	var params url.Values
	if req != nil {
		params = futuresContractTypeParams(req.ContractTypes)
	}
	var resp *FuturesTradingInstrumentsResponse
	return resp, e.SendFuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, "/api/v3/trading/instruments", params, nil, &resp)
}

// GetFuturesInstrumentStatusList calls Get instrument status list, returning each market's price dislocation and
// volatility status. A nil request returns every futures market's status, which excludes options
func (e *Exchange) GetFuturesInstrumentStatusList(ctx context.Context, req *FuturesInstrumentStatusListRequest) (*FuturesInstrumentStatusListResponse, error) {
	var params url.Values
	if req != nil {
		params = futuresContractTypeParams(req.ContractTypes)
	}
	var resp *FuturesInstrumentStatusListResponse
	return resp, e.SendFuturesHTTPRequest(ctx, exchange.RestFutures, common.EncodeURLValues("/api/v3/instruments/status", params), &resp)
}

// GetFuturesInstrumentStatus calls Get instrument status, returning a market's price dislocation and volatility status
func (e *Exchange) GetFuturesInstrumentStatus(ctx context.Context, pair currency.Pair) (*FuturesInstrumentStatusResponse, error) {
	symbol, err := e.futuresMarketSymbol(pair)
	if err != nil {
		return nil, err
	}
	var resp *FuturesInstrumentStatusResponse
	return resp, e.SendFuturesHTTPRequest(ctx, exchange.RestFutures, "/api/v3/instruments/"+url.PathEscape(symbol)+"/status", &resp)
}
