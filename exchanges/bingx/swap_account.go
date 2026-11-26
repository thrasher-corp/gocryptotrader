package bingx

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// GetSwapAccountData returns the balance, equity and margins of each asset of the USDT- and USDC-settled perpetual
// account from Query account data
func (e *Exchange) GetSwapAccountData(ctx context.Context) ([]SwapAccountDataResponse, error) {
	var resp []SwapAccountDataResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/openApi/swap/v3/user/balance", nil, swapBalanceRate, &resp)
}

// GetSwapPositionData returns the account's perpetual positions from Query position data: a symbol's, a settlement
// currency's, or with neither set every position. BingX reports a failed query as code 109500, so an empty list means
// no position matched
func (e *Exchange) GetSwapPositionData(ctx context.Context, arg *SwapPositionDataRequest) ([]SwapPositionDataResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params, err := e.swapMarketSymbolParams(arg.Symbol)
	if err != nil {
		return nil, err
	}
	if !arg.Currency.IsEmpty() {
		params.Set("currency", arg.Currency.Upper().String())
	}
	var resp []SwapPositionDataResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/openApi/swap/v2/user/positions", params, swapPositionsRate, &resp)
}

// GetSwapFundFlow returns the account's perpetual fund flow records from Get Account Profit and Loss Fund Flow, for one
// settlement currency. Without a time range the last seven days are returned, and only the last three months are kept
func (e *Exchange) GetSwapFundFlow(ctx context.Context, arg *SwapFundFlowRequest) ([]SwapFundFlowResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params, err := e.swapMarketSymbolParams(arg.Symbol)
	if err != nil {
		return nil, err
	}
	if err := setTimeRange(params, "startTime", "endTime", arg.StartTime, arg.EndTime); err != nil {
		return nil, err
	}
	if !arg.Currency.IsEmpty() {
		params.Set("currency", arg.Currency.Upper().String())
	}
	if arg.IncomeType != "" {
		params.Set("incomeType", string(arg.IncomeType))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []SwapFundFlowResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/openApi/swap/v2/user/income", params, swapIncomeRate, &resp)
}

// GetSwapTradingCommissionRate returns the account's perpetual taker and maker fee rates from Query Trading Commission
// Rate
func (e *Exchange) GetSwapTradingCommissionRate(ctx context.Context) (*SwapTradingCommissionRateResponse, error) {
	var resp *SwapTradingCommissionRateResponse
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/openApi/swap/v2/user/commissionRate", nil, swapCommissionRateRate, &resp); err != nil {
		return nil, err
	}
	if resp.Commission == nil {
		return nil, fmt.Errorf("%w: commission", common.ErrNoResponse)
	}
	return resp, nil
}
