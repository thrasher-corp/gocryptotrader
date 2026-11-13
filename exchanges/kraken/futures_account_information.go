package kraken

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// GetFuturesAccounts calls Get wallets, returning the balances, margin requirements and auxiliary values, such as
// available funds and unrealised PnL, of the cash account, the multi-collateral margin account and every
// single-collateral margin account
func (e *Exchange) GetFuturesAccounts(ctx context.Context) (*FuturesAccountsResponse, error) {
	var resp *FuturesAccountsResponse
	return resp, e.SendFuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, "/api/v3/accounts", nil, nil, &resp)
}

// GetFuturesOpenPositions calls Get open positions, returning the size and average entry price of every open
// position, including positions in contracts that have matured but not yet settled
func (e *Exchange) GetFuturesOpenPositions(ctx context.Context) (*FuturesOpenPositionsResponse, error) {
	var resp *FuturesOpenPositionsResponse
	return resp, e.SendFuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, "/api/v3/openpositions", nil, nil, &resp)
}

// GetFuturesUnwindQueue calls Get position percentile of unwind queue, returning each open position's percentile rank
// in the unwind queue
func (e *Exchange) GetFuturesUnwindQueue(ctx context.Context) (*FuturesUnwindQueueResponse, error) {
	var resp *FuturesUnwindQueueResponse
	return resp, e.SendFuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, "/api/v3/unwindqueue", nil, nil, &resp)
}

// GetFuturesPortfolioMarginParameters calls Get portfolio margin parameters, returning the parameters of the
// portfolio margin calculation and the account's options trading limits. Kraken serves it only in its pre-production
// environments
func (e *Exchange) GetFuturesPortfolioMarginParameters(ctx context.Context) (*FuturesPortfolioMarginParametersResponse, error) {
	var resp *FuturesPortfolioMarginParametersResponse
	return resp, e.SendFuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, "/api/v3/portfolio-margining/parameters", nil, nil, &resp)
}

// SimulateFuturesPortfolio calls Calculate portfolio margin, pnl and greeks, returning the margin requirements, PnL
// and option greeks of a portfolio of up to 500 futures and options positions. Kraken serves it only in its
// pre-production environments
func (e *Exchange) SimulateFuturesPortfolio(ctx context.Context, positions []FuturesSimulatedPosition) (*FuturesSimulatePortfolioResponse, error) {
	if len(positions) == 0 {
		return nil, fmt.Errorf("%w: no positions", common.ErrEmptyParams)
	}
	if len(positions) > 500 {
		return nil, fmt.Errorf("%w: %d positions exceed 500", errFuturesTooManyEntries, len(positions))
	}
	portfolio := futuresPortfolioSimulationJSON{Positions: make([]futuresSimulatedPositionJSON, len(positions))}
	for i := range positions {
		instrument, err := e.futuresMarketSymbol(positions[i].Instrument)
		if err != nil {
			return nil, fmt.Errorf("position %d: %w", i, err)
		}
		portfolio.Positions[i] = futuresSimulatedPositionJSON{Instrument: instrument, Size: positions[i].Size, EntryPrice: positions[i].EntryPrice}
	}
	data, err := json.Marshal(portfolio)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("json", string(data))
	var resp *FuturesSimulatePortfolioResponse
	return resp, e.SendFuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/api/v3/portfolio-margining/simulate", params, nil, &resp)
}
