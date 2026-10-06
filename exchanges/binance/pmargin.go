package binance

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Portfolio margin (PAPI) terminology: Margin is cross margin, UM is USDⓈ-M futures and CM is COIN-M futures

var (
	errTransferSideRequired         = errors.New("transfer side is required")
	errSymbolWithPair               = errors.New("symbol and pair cannot be sent together")
	errFromIDWithPair               = errors.New("fromId and pair cannot be sent together")
	errFromIDWithTimeRange          = errors.New("fromId cannot be sent with startTime or endTime")
	errGoodTillDateRequired         = errors.New("goodTillDate is required for GTD orders")
	errStopLimitTimeInForceRequired = errors.New("stopLimitTimeInForce is required with stopLimitPrice")
	errClosePositionConflict        = errors.New("closePosition cannot be sent with quantity or reduceOnly")
)

// pmOrderSide returns the wire value of a buy or sell order side
func pmOrderSide(side order.Side) (string, error) {
	switch side {
	case order.Buy, order.Sell:
		return side.String(), nil
	default:
		return "", fmt.Errorf("%w: %q", order.ErrSideIsInvalid, side)
	}
}

// pmSymbolParam sets the symbol parameter when a pair is given
func (e *Exchange) pmSymbolParam(params url.Values, pair currency.Pair, a asset.Item) error {
	if pair.IsEmpty() {
		return nil
	}
	symbol, err := e.FormatSymbol(pair, a)
	if err != nil {
		return err
	}
	params.Set("symbol", symbol)
	return nil
}

// GetAccountBalance returns the portfolio margin account balance of one asset, or of every asset when asset is empty
func (e *Exchange) GetAccountBalance(ctx context.Context, ccy currency.Code) (AccountBalanceResponse, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("asset", ccy.Upper().String())
	}
	var resp AccountBalanceResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/balance", params, pmGetAccountBalancesRate, &resp)
}

// GetPortfolioMarginAccountInformation returns the portfolio margin account information
func (e *Exchange) GetPortfolioMarginAccountInformation(ctx context.Context) (*PMAccountInformationResponse, error) {
	var resp *PMAccountInformationResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/account", nil, pmGetAccountInformationRate, &resp)
}

// BNBTransfer transfers BNB into or out of the UM account; transferSide is TO_UM or FROM_UM. The endpoint can be called
// 10 times per 10 minutes
func (e *Exchange) BNBTransfer(ctx context.Context, amount float64, transferSide string) (*PMTransactionResponse, error) {
	return e.pmTransferBNB(ctx, exchange.RestFuturesSupplementary, "/papi/v1/bnb-transfer", amount, transferSide, pmBNBTransferRate)
}

func (e *Exchange) pmTransferBNB(ctx context.Context, ePath exchange.URL, path string, amount float64, transferSide string, f request.EndpointLimit) (*PMTransactionResponse, error) {
	if amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if transferSide == "" {
		return nil, errTransferSideRequired
	}
	params := url.Values{}
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	params.Set("transferSide", transferSide)
	var resp *PMTransactionResponse
	return resp, e.SendAuthHTTPRequest(ctx, ePath, http.MethodPost, path, params, f, &resp)
}

// GetAutoRepayFuturesStatus returns whether futures negative balances are repaid automatically
func (e *Exchange) GetAutoRepayFuturesStatus(ctx context.Context) (*PMAutoRepayFuturesStatusResponse, error) {
	var resp *PMAutoRepayFuturesStatusResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/repay-futures-switch", nil, pmGetAutoRepayFuturesStatusRate, &resp)
}

// ChangeAutoRepayFuturesStatus turns the automatic repayment of futures negative balances on or off. The endpoint can
// be called 20 times a day
func (e *Exchange) ChangeAutoRepayFuturesStatus(ctx context.Context, autoRepay bool) (*PMMessageResponse, error) {
	return e.pmChangeAutoRepayFuturesStatus(ctx, exchange.RestFuturesSupplementary, "/papi/v1/repay-futures-switch", autoRepay, pmChangeAutoRepayFuturesStatusRate)
}

func (e *Exchange) pmChangeAutoRepayFuturesStatus(ctx context.Context, ePath exchange.URL, path string, autoRepay bool, f request.EndpointLimit) (*PMMessageResponse, error) {
	params := url.Values{}
	params.Set("autoRepay", strconv.FormatBool(autoRepay))
	var resp *PMMessageResponse
	return resp, e.SendAuthHTTPRequest(ctx, ePath, http.MethodPost, path, params, f, &resp)
}

// ChangeCMInitialLeverage changes the initial leverage of a CM symbol
func (e *Exchange) ChangeCMInitialLeverage(ctx context.Context, symbol currency.Pair, leverage uint64) (*CMInitialLeverageResponse, error) {
	params, err := e.pmLeverageParams(symbol, asset.CoinMarginedFutures, leverage)
	if err != nil {
		return nil, err
	}
	var resp *CMInitialLeverageResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodPost, "/papi/v1/cm/leverage", params, pmDefaultRate, &resp)
}

func (e *Exchange) pmLeverageParams(symbol currency.Pair, a asset.Item, leverage uint64) (url.Values, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if leverage == 0 {
		return nil, errLeverageRequired
	}
	params := url.Values{}
	if err := e.pmSymbolParam(params, symbol, a); err != nil {
		return nil, err
	}
	params.Set("leverage", strconv.FormatUint(leverage, 10))
	return params, nil
}

// GetCMCurrentPositionMode returns whether hedge mode is enabled for every CM symbol
func (e *Exchange) GetCMCurrentPositionMode(ctx context.Context) (*PMPositionModeResponse, error) {
	var resp *PMPositionModeResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/cm/positionSide/dual", nil, pmGetCMCurrentPositionModeRate, &resp)
}

// ChangeCMPositionMode switches every CM symbol between hedge mode (true) and one-way mode (false); the CM position
// mode must match the UM position mode
func (e *Exchange) ChangeCMPositionMode(ctx context.Context, dualSidePosition bool) (*SuccessResponse, error) {
	return e.pmChangePositionMode(ctx, "/papi/v1/cm/positionSide/dual", dualSidePosition)
}

func (e *Exchange) pmChangePositionMode(ctx context.Context, path string, dualSidePosition bool) (*SuccessResponse, error) {
	params := url.Values{}
	params.Set("dualSidePosition", strconv.FormatBool(dualSidePosition))
	var resp *SuccessResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodPost, path, params, pmDefaultRate, &resp)
}

// ChangeUMInitialLeverage changes the initial leverage of a UM symbol
func (e *Exchange) ChangeUMInitialLeverage(ctx context.Context, symbol currency.Pair, leverage uint64) (*UMInitialLeverageResponse, error) {
	params, err := e.pmLeverageParams(symbol, asset.USDTMarginedFutures, leverage)
	if err != nil {
		return nil, err
	}
	var resp *UMInitialLeverageResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodPost, "/papi/v1/um/leverage", params, pmDefaultRate, &resp)
}

// GetUMCurrentPositionMode returns whether hedge mode is enabled for every UM symbol
func (e *Exchange) GetUMCurrentPositionMode(ctx context.Context) (*PMPositionModeResponse, error) {
	var resp *PMPositionModeResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/um/positionSide/dual", nil, pmGetUMCurrentPositionModeRate, &resp)
}

// ChangeUMPositionMode switches every UM symbol between hedge mode (true) and one-way mode (false)
func (e *Exchange) ChangeUMPositionMode(ctx context.Context, dualSidePosition bool) (*SuccessResponse, error) {
	return e.pmChangePositionMode(ctx, "/papi/v1/um/positionSide/dual", dualSidePosition)
}

// GetCMNotionalAndLeverageBrackets returns the notional and leverage brackets of one CM symbol, or of every CM symbol
// when symbol is empty
func (e *Exchange) GetCMNotionalAndLeverageBrackets(ctx context.Context, symbol currency.Pair) ([]CMNotionalAndLeverage, error) {
	params := url.Values{}
	if err := e.pmSymbolParam(params, symbol, asset.CoinMarginedFutures); err != nil {
		return nil, err
	}
	var resp []CMNotionalAndLeverage
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/cm/leverageBracket", params, pmDefaultRate, &resp)
}

// FundAutoCollection collects every asset except BNB from the futures accounts into the margin account. The endpoint
// can be called 500 times per hour
func (e *Exchange) FundAutoCollection(ctx context.Context) (*PMMessageResponse, error) {
	var resp *PMMessageResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodPost, "/papi/v1/auto-collection", nil, pmFundAutoCollectionRate, &resp)
}

// FundCollectionByAsset collects an asset other than BNB from the futures accounts into the margin account
func (e *Exchange) FundCollectionByAsset(ctx context.Context, ccy currency.Code) (*PMMessageResponse, error) {
	return e.pmFundCollectionByAsset(ctx, exchange.RestFuturesSupplementary, "/papi/v1/asset-collection", ccy, pmFundCollectionByAssetRate)
}

func (e *Exchange) pmFundCollectionByAsset(ctx context.Context, ePath exchange.URL, path string, ccy currency.Code, f request.EndpointLimit) (*PMMessageResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("asset", ccy.Upper().String())
	var resp *PMMessageResponse
	return resp, e.SendAuthHTTPRequest(ctx, ePath, http.MethodPost, path, params, f, &resp)
}

// GetCMAccountDetail returns the CM account assets and positions
func (e *Exchange) GetCMAccountDetail(ctx context.Context) (*CMAccountDetailResponse, error) {
	var resp *CMAccountDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/cm/account", nil, pmGetCMAccountDetailRate, &resp)
}

// GetCMIncomeHistory returns CM income history; without a time range the last 200 days are returned, and a range
// cannot exceed 200 days
func (e *Exchange) GetCMIncomeHistory(ctx context.Context, arg *PMIncomeHistoryRequest) ([]IncomeItem, error) {
	return e.pmIncomeHistory(ctx, "/papi/v1/cm/income", asset.CoinMarginedFutures, arg, pmGetCMIncomeHistoryRate)
}

func (e *Exchange) pmIncomeHistory(ctx context.Context, path string, a asset.Item, arg *PMIncomeHistoryRequest, f request.EndpointLimit) ([]IncomeItem, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params := url.Values{}
	if err := e.pmSymbolParam(params, arg.Symbol, a); err != nil {
		return nil, err
	}
	if arg.IncomeType != "" {
		params.Set("incomeType", arg.IncomeType)
	}
	if err := setTimeRangeParams(params, arg.StartTime, arg.EndTime); err != nil {
		return nil, err
	}
	if arg.Page > 0 {
		params.Set("page", strconv.FormatUint(arg.Page, 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []IncomeItem
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, path, params, f, &resp)
}

// GetMarginBorrowOrLoanInterestHistory returns margin borrow and loan interest records, newest first; without a time
// range the last 7 days are returned, and a range cannot exceed 30 days
func (e *Exchange) GetMarginBorrowOrLoanInterestHistory(ctx context.Context, arg *PMMarginInterestHistoryRequest) (*PMMarginInterestHistoryResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params := url.Values{}
	if !arg.Asset.IsEmpty() {
		params.Set("asset", arg.Asset.Upper().String())
	}
	if err := setTimeRangeParams(params, arg.StartTime, arg.EndTime); err != nil {
		return nil, err
	}
	if arg.Current > 0 {
		params.Set("current", strconv.FormatUint(arg.Current, 10))
	}
	if arg.Size > 0 {
		params.Set("size", strconv.FormatUint(arg.Size, 10))
	}
	if arg.Archived {
		params.Set("archived", "true")
	}
	var resp *PMMarginInterestHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/margin/marginInterestHistory", params, pmDefaultRate, &resp)
}

// GetUMAccountDetail returns the UM account assets and positions
func (e *Exchange) GetUMAccountDetail(ctx context.Context) (*UMAccountDetailResponse, error) {
	var resp *UMAccountDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/um/account", nil, pmGetUMAccountDetailRate, &resp)
}

// GetUMIncomeHistory returns UM income history of the last three months; without a time range the last 7 days are
// returned
func (e *Exchange) GetUMIncomeHistory(ctx context.Context, arg *PMIncomeHistoryRequest) ([]IncomeItem, error) {
	return e.pmIncomeHistory(ctx, "/papi/v1/um/income", asset.USDTMarginedFutures, arg, pmGetUMIncomeHistoryRate)
}

// GetCMUserCommissionRate returns the user's commission rates for a CM symbol
func (e *Exchange) GetCMUserCommissionRate(ctx context.Context, symbol currency.Pair) (*PMCommissionRateResponse, error) {
	return e.pmUserCommissionRate(ctx, "/papi/v1/cm/commissionRate", symbol, asset.CoinMarginedFutures, pmGetCMUserCommissionRate)
}

func (e *Exchange) pmUserCommissionRate(ctx context.Context, path string, symbol currency.Pair, a asset.Item, f request.EndpointLimit) (*PMCommissionRateResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	if err := e.pmSymbolParam(params, symbol, a); err != nil {
		return nil, err
	}
	var resp *PMCommissionRateResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, path, params, f, &resp)
}

// GetUMUserCommissionRate returns the user's commission rates for a UM symbol
func (e *Exchange) GetUMUserCommissionRate(ctx context.Context, symbol currency.Pair) (*PMCommissionRateResponse, error) {
	return e.pmUserCommissionRate(ctx, "/papi/v1/um/commissionRate", symbol, asset.USDTMarginedFutures, pmGetUMUserCommissionRate)
}

// GetPMMarginMaxBorrow returns the maximum amount of an asset the margin account can borrow
func (e *Exchange) GetPMMarginMaxBorrow(ctx context.Context, ccy currency.Code) (*PMMarginMaxBorrowResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("asset", ccy.Upper().String())
	var resp *PMMarginMaxBorrowResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/margin/maxBorrowable", params, pmMarginMaxBorrowRate, &resp)
}

// GetPortfolioMarginUMTradingQuantitativeRulesIndicator returns the UM trading quantitative rules indicators of one
// symbol, or of every symbol when symbol is empty
func (e *Exchange) GetPortfolioMarginUMTradingQuantitativeRulesIndicator(ctx context.Context, symbol currency.Pair) (*PMTradingQuantitativeRulesIndicatorsResponse, error) {
	params := url.Values{}
	f := pmUMTradingQuantitativeRulesIndicatorsRate
	if !symbol.IsEmpty() {
		if err := e.pmSymbolParam(params, symbol, asset.USDTMarginedFutures); err != nil {
			return nil, err
		}
		f = pmDefaultRate
	}
	var resp *PMTradingQuantitativeRulesIndicatorsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/um/apiTradingStatus", params, f, &resp)
}

// GetCMPositionInformation returns CM positions, filtered by margin asset or by pair (such as BTCUSD, the base of a
// COIN-M contract pair); without either the positions of every trading symbol are returned
func (e *Exchange) GetCMPositionInformation(ctx context.Context, marginAsset, pair currency.Code) ([]CMPositionInformation, error) {
	params := url.Values{}
	if !marginAsset.IsEmpty() {
		params.Set("marginAsset", marginAsset.Upper().String())
	}
	if !pair.IsEmpty() {
		params.Set("pair", pair.Upper().String())
	}
	var resp []CMPositionInformation
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/cm/positionRisk", params, pmDefaultRate, &resp)
}

// GetMarginLoanRecord returns margin loan records, newest first; a transaction ID or start time is required, and a
// time range cannot exceed 30 days
func (e *Exchange) GetMarginLoanRecord(ctx context.Context, arg *PMMarginLoanRepayRecordRequest) (*PMMarginLoanRecordResponse, error) {
	params, err := pmLoanRepayRecordParams(arg)
	if err != nil {
		return nil, err
	}
	var resp *PMMarginLoanRecordResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/margin/marginLoan", params, pmGetMarginLoanRecordRate, &resp)
}

func pmLoanRepayRecordParams(arg *PMMarginLoanRepayRecordRequest) (url.Values, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Asset.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.TransactionID == 0 && arg.StartTime.IsZero() {
		return nil, fmt.Errorf("%w when no transaction ID is given", errStartTimeRequired)
	}
	params := url.Values{}
	params.Set("asset", arg.Asset.Upper().String())
	if arg.TransactionID > 0 {
		params.Set("txId", strconv.FormatUint(arg.TransactionID, 10))
	}
	if err := setTimeRangeParams(params, arg.StartTime, arg.EndTime); err != nil {
		return nil, err
	}
	if arg.Current > 0 {
		params.Set("current", strconv.FormatUint(arg.Current, 10))
	}
	if arg.Size > 0 {
		params.Set("size", strconv.FormatUint(arg.Size, 10))
	}
	if arg.Archived {
		params.Set("archived", "true")
	}
	return params, nil
}

// GetMarginMaxWithdrawal returns the maximum amount of an asset that can be withdrawn from the margin account
func (e *Exchange) GetMarginMaxWithdrawal(ctx context.Context, ccy currency.Code) (*PMMarginMaxWithdrawResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("asset", ccy.Upper().String())
	var resp *PMMarginMaxWithdrawResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/margin/maxWithdraw", params, pmGetMarginMaxWithdrawalRate, &resp)
}

// GetMarginRepayRecord returns margin repay records, newest first; a transaction ID or start time is required, and a
// time range cannot exceed 30 days
func (e *Exchange) GetMarginRepayRecord(ctx context.Context, arg *PMMarginLoanRepayRecordRequest) (*PMMarginRepayRecordResponse, error) {
	params, err := pmLoanRepayRecordParams(arg)
	if err != nil {
		return nil, err
	}
	var resp *PMMarginRepayRecordResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/margin/repayLoan", params, pmGetMarginRepayRecordRate, &resp)
}

// GetPortfolioMarginNegativeBalanceInterestHistory returns the interest charged on negative balances, newest first;
// without a time range the last 7 days are returned, and a range cannot exceed 30 days
func (e *Exchange) GetPortfolioMarginNegativeBalanceInterestHistory(ctx context.Context, arg *PMNegativeBalanceInterestHistoryRequest) ([]PortfolioMarginNegativeBalanceInterest, error) {
	params, err := pmNegativeBalanceInterestHistoryParams(arg)
	if err != nil {
		return nil, err
	}
	var resp []PortfolioMarginNegativeBalanceInterest
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/portfolio/interest-history", params, pmGetPortfolioMarginNegativeBalanceInterestHistoryRate, &resp)
}

func pmNegativeBalanceInterestHistoryParams(arg *PMNegativeBalanceInterestHistoryRequest) (url.Values, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params := url.Values{}
	if !arg.Asset.IsEmpty() {
		params.Set("asset", arg.Asset.Upper().String())
	}
	if err := setTimeRangeParams(params, arg.StartTime, arg.EndTime); err != nil {
		return nil, err
	}
	if arg.Size > 0 {
		params.Set("size", strconv.FormatUint(arg.Size, 10))
	}
	return params, nil
}

// GetUMPositionInformation returns the UM positions of one symbol, or of every symbol when symbol is empty; one-way
// mode accounts only have BOTH positions and hedge mode accounts LONG and SHORT positions
func (e *Exchange) GetUMPositionInformation(ctx context.Context, symbol currency.Pair) ([]UMPositionInformation, error) {
	params := url.Values{}
	if err := e.pmSymbolParam(params, symbol, asset.USDTMarginedFutures); err != nil {
		return nil, err
	}
	var resp []UMPositionInformation
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/um/positionRisk", params, pmGetUMPositionInformationRate, &resp)
}

// GetUserNegativeBalanceAutoExchangeRecord returns negative balance auto exchange records, newest first; the time
// range is required and cannot exceed three months
func (e *Exchange) GetUserNegativeBalanceAutoExchangeRecord(ctx context.Context, startTime, endTime time.Time) (*PMNegativeBalanceExchangeRecordResponse, error) {
	if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	var resp *PMNegativeBalanceExchangeRecordResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/portfolio/negative-balance-exchange-record", params, pmNegativeBalanceExchangeRecordRate, &resp)
}

// GetUserRateLimits returns the user's portfolio margin order rate limits
func (e *Exchange) GetUserRateLimits(ctx context.Context) ([]PMRateLimit, error) {
	var resp []PMRateLimit
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/rateLimit/order", nil, pmDefaultRate, &resp)
}

// RepayFuturesNegativeBalance repays futures negative balances
func (e *Exchange) RepayFuturesNegativeBalance(ctx context.Context) (*PMMessageResponse, error) {
	var resp *PMMessageResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodPost, "/papi/v1/repay-futures-negative-balance", nil, pmRepayFuturesNegativeBalanceRate, &resp)
}

// GetUMNotionalAndLeverageBrackets returns the notional and leverage brackets of one UM symbol, or of every UM symbol
// when symbol is empty
func (e *Exchange) GetUMNotionalAndLeverageBrackets(ctx context.Context, symbol currency.Pair) ([]UMNotionalAndLeverage, error) {
	params := url.Values{}
	if err := e.pmSymbolParam(params, symbol, asset.USDTMarginedFutures); err != nil {
		return nil, err
	}
	var resp []UMNotionalAndLeverage
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/um/leverageBracket", params, pmDefaultRate, &resp)
}

// CancelAllCMOpenConditionalOrders cancels every open CM conditional order of a symbol
func (e *Exchange) CancelAllCMOpenConditionalOrders(ctx context.Context, symbol currency.Pair) (*SuccessResponse, error) {
	return e.pmCancelAllOpenOrders(ctx, "/papi/v1/cm/conditional/allOpenOrders", symbol, asset.CoinMarginedFutures)
}

func (e *Exchange) pmCancelAllOpenOrders(ctx context.Context, path string, symbol currency.Pair, a asset.Item) (*SuccessResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	if err := e.pmSymbolParam(params, symbol, a); err != nil {
		return nil, err
	}
	var resp *SuccessResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodDelete, path, params, pmDefaultRate, &resp)
}

// CancelAllCMOrders cancels every open CM LIMIT order of a symbol
func (e *Exchange) CancelAllCMOrders(ctx context.Context, symbol currency.Pair) (*SuccessResponse, error) {
	return e.pmCancelAllOpenOrders(ctx, "/papi/v1/cm/allOpenOrders", symbol, asset.CoinMarginedFutures)
}

// CancelAllUMOrders cancels every open UM LIMIT order of a symbol
func (e *Exchange) CancelAllUMOrders(ctx context.Context, symbol currency.Pair) (*SuccessResponse, error) {
	return e.pmCancelAllOpenOrders(ctx, "/papi/v1/um/allOpenOrders", symbol, asset.USDTMarginedFutures)
}

// NewCMConditionalOrder places a CM conditional order
func (e *Exchange) NewCMConditionalOrder(ctx context.Context, arg *CMConditionalOrderRequest) (*NewCMConditionalOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	side, err := pmOrderSide(arg.Side)
	if err != nil {
		return nil, err
	}
	if arg.StrategyType == "" {
		return nil, errStrategyTypeRequired
	}
	params := url.Values{}
	if err := e.pmSymbolParam(params, arg.Symbol, asset.CoinMarginedFutures); err != nil {
		return nil, err
	}
	params.Set("side", side)
	if arg.PositionSide != "" {
		params.Set("positionSide", arg.PositionSide)
	}
	params.Set("strategyType", arg.StrategyType)
	if arg.TimeInForce != "" {
		params.Set("timeInForce", arg.TimeInForce)
	}
	if arg.Quantity > 0 {
		params.Set("quantity", strconv.FormatFloat(arg.Quantity, 'f', -1, 64))
	}
	if arg.ReduceOnly {
		params.Set("reduceOnly", "true")
	}
	if arg.Price > 0 {
		params.Set("price", strconv.FormatFloat(arg.Price, 'f', -1, 64))
	}
	if arg.WorkingType != "" {
		params.Set("workingType", arg.WorkingType)
	}
	if arg.PriceProtect {
		params.Set("priceProtect", "true")
	}
	if arg.NewClientStrategyID != "" {
		params.Set("newClientStrategyId", arg.NewClientStrategyID)
	}
	if arg.StopPrice > 0 {
		params.Set("stopPrice", strconv.FormatFloat(arg.StopPrice, 'f', -1, 64))
	}
	if arg.ActivationPrice > 0 {
		params.Set("activationPrice", strconv.FormatFloat(arg.ActivationPrice, 'f', -1, 64))
	}
	if arg.CallbackRate > 0 {
		params.Set("callbackRate", strconv.FormatFloat(arg.CallbackRate, 'f', -1, 64))
	}
	var resp *NewCMConditionalOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodPost, "/papi/v1/cm/conditional/order", params, pmOrderRate, &resp)
}

// CancelCMConditionalOrder cancels a CM conditional order identified by its strategy ID or client strategy ID
func (e *Exchange) CancelCMConditionalOrder(ctx context.Context, symbol currency.Pair, strategyID uint64, newClientStrategyID string) (*CancelCMConditionalOrderResponse, error) {
	params, err := e.pmConditionalOrderParams(symbol, strategyID, newClientStrategyID)
	if err != nil {
		return nil, err
	}
	var resp *CancelCMConditionalOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodDelete, "/papi/v1/cm/conditional/order", params, pmDefaultRate, &resp)
}

func (e *Exchange) pmConditionalOrderParams(symbol currency.Pair, strategyID uint64, newClientStrategyID string) (url.Values, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if strategyID == 0 && newClientStrategyID == "" {
		return nil, fmt.Errorf("%w: strategyId or newClientStrategyId is required", order.ErrOrderIDNotSet)
	}
	params := url.Values{}
	if err := e.pmSymbolParam(params, symbol, asset.CoinMarginedFutures); err != nil {
		return nil, err
	}
	if strategyID > 0 {
		params.Set("strategyId", strconv.FormatUint(strategyID, 10))
	}
	if newClientStrategyID != "" {
		params.Set("newClientStrategyId", newClientStrategyID)
	}
	return params, nil
}

// GetCMOrder returns a CM order identified by its order ID or client order ID; cancelled or expired orders without
// fills are not found three days after creation
func (e *Exchange) GetCMOrder(ctx context.Context, symbol currency.Pair, orderID uint64, origClientOrderID string) (*CMOrderDetailResponse, error) {
	params, err := e.pmOrderParams(symbol, asset.CoinMarginedFutures, orderID, origClientOrderID)
	if err != nil {
		return nil, err
	}
	var resp *CMOrderDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/cm/order", params, pmDefaultRate, &resp)
}

// pmOrderParams builds the parameters identifying a single order by its order ID or original client order ID
func (e *Exchange) pmOrderParams(symbol currency.Pair, a asset.Item, orderID uint64, origClientOrderID string) (url.Values, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if orderID == 0 && origClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	if err := e.pmSymbolParam(params, symbol, a); err != nil {
		return nil, err
	}
	if orderID > 0 {
		params.Set("orderId", strconv.FormatUint(orderID, 10))
	}
	if origClientOrderID != "" {
		params.Set("origClientOrderId", origClientOrderID)
	}
	return params, nil
}

// NewCMOrder places a CM order
func (e *Exchange) NewCMOrder(ctx context.Context, arg *CMOrderRequest) (*CMOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params, err := e.pmNewFuturesOrderParams(arg.Symbol, asset.CoinMarginedFutures, arg.Side, arg.OrderType, arg.Price, arg.PriceMatch)
	if err != nil {
		return nil, err
	}
	if arg.PositionSide != "" {
		params.Set("positionSide", arg.PositionSide)
	}
	if arg.TimeInForce != "" {
		params.Set("timeInForce", arg.TimeInForce)
	}
	if arg.Quantity > 0 {
		params.Set("quantity", strconv.FormatFloat(arg.Quantity, 'f', -1, 64))
	}
	if arg.ReduceOnly {
		params.Set("reduceOnly", "true")
	}
	if arg.NewClientOrderID != "" {
		params.Set("newClientOrderId", arg.NewClientOrderID)
	}
	if arg.NewOrderRespType != "" {
		params.Set("newOrderRespType", arg.NewOrderRespType)
	}
	var resp *CMOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodPost, "/papi/v1/cm/order", params, pmOrderRate, &resp)
}

// pmNewFuturesOrderParams validates and sets the parameters shared by new UM and CM orders
func (e *Exchange) pmNewFuturesOrderParams(symbol currency.Pair, a asset.Item, orderSide order.Side, orderType string, price float64, priceMatch string) (url.Values, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	side, err := pmOrderSide(orderSide)
	if err != nil {
		return nil, err
	}
	if orderType == "" {
		return nil, order.ErrTypeIsInvalid
	}
	if price > 0 && priceMatch != "" {
		return nil, errPriceMatchWithPrice
	}
	params := url.Values{}
	if err := e.pmSymbolParam(params, symbol, a); err != nil {
		return nil, err
	}
	params.Set("side", side)
	params.Set("type", orderType)
	if price > 0 {
		params.Set("price", strconv.FormatFloat(price, 'f', -1, 64))
	}
	if priceMatch != "" {
		params.Set("priceMatch", priceMatch)
	}
	return params, nil
}

// CancelCMOrder cancels an open CM LIMIT order identified by its order ID or client order ID
func (e *Exchange) CancelCMOrder(ctx context.Context, symbol currency.Pair, orderID uint64, origClientOrderID string) (*CMOrderResponse, error) {
	params, err := e.pmOrderParams(symbol, asset.CoinMarginedFutures, orderID, origClientOrderID)
	if err != nil {
		return nil, err
	}
	var resp *CMOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodDelete, "/papi/v1/cm/order", params, pmDefaultRate, &resp)
}

// CancelAllMarginOpenOrdersBySymbol cancels every open margin order and OCO order list of a symbol
func (e *Exchange) CancelAllMarginOpenOrdersBySymbol(ctx context.Context, symbol currency.Pair) ([]PMCancelledMarginOrder, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	if err := e.pmSymbolParam(params, symbol, asset.Margin); err != nil {
		return nil, err
	}
	var resp []PMCancelledMarginOrder
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodDelete, "/papi/v1/margin/allOpenOrders", params, pmCancelMarginAccountOpenOrdersOnSymbolRate, &resp)
}

// GetMarginAccountOCO returns a margin OCO order list identified by its order list ID or original client order ID
func (e *Exchange) GetMarginAccountOCO(ctx context.Context, orderListID uint64, origClientOrderID string) (*PMMarginOCOOrderResponse, error) {
	if orderListID == 0 && origClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	if orderListID > 0 {
		params.Set("orderListId", strconv.FormatUint(orderListID, 10))
	}
	if origClientOrderID != "" {
		params.Set("origClientOrderId", origClientOrderID)
	}
	var resp *PMMarginOCOOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/margin/orderList", params, pmGetMarginAccountOCORate, &resp)
}

// CancelMarginAccountOCOOrders cancels a margin OCO order list; cancelling either leg cancels the whole list
func (e *Exchange) CancelMarginAccountOCOOrders(ctx context.Context, arg *PMCancelMarginOCOOrderRequest) (*PMCancelMarginOCOOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if arg.OrderListID == 0 && arg.ListClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	if err := e.pmSymbolParam(params, arg.Symbol, asset.Margin); err != nil {
		return nil, err
	}
	if arg.OrderListID > 0 {
		params.Set("orderListId", strconv.FormatUint(arg.OrderListID, 10))
	}
	if arg.ListClientOrderID != "" {
		params.Set("listClientOrderId", arg.ListClientOrderID)
	}
	if arg.NewClientOrderID != "" {
		params.Set("newClientOrderId", arg.NewClientOrderID)
	}
	var resp *PMCancelMarginOCOOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodDelete, "/papi/v1/margin/orderList", params, pmCancelMarginAccountOCORate, &resp)
}

// GetMarginAccountOrder returns a margin order identified by its order ID or client order ID
func (e *Exchange) GetMarginAccountOrder(ctx context.Context, symbol currency.Pair, orderID uint64, origClientOrderID string) (*PMMarginOrderResponse, error) {
	params, err := e.pmOrderParams(symbol, asset.Margin, orderID, origClientOrderID)
	if err != nil {
		return nil, err
	}
	var resp *PMMarginOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/margin/order", params, pmGetMarginAccountOrderRate, &resp)
}

// NewMarginOrder places a margin order
func (e *Exchange) NewMarginOrder(ctx context.Context, arg *PMMarginOrderRequest) (*PMNewMarginOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	side, err := pmOrderSide(arg.Side)
	if err != nil {
		return nil, err
	}
	if arg.OrderType == "" {
		return nil, order.ErrTypeIsInvalid
	}
	params := url.Values{}
	if err := e.pmSymbolParam(params, arg.Symbol, asset.Margin); err != nil {
		return nil, err
	}
	params.Set("side", side)
	params.Set("type", arg.OrderType)
	if arg.Quantity > 0 {
		params.Set("quantity", strconv.FormatFloat(arg.Quantity, 'f', -1, 64))
	}
	if arg.QuoteOrderQuantity > 0 {
		params.Set("quoteOrderQty", strconv.FormatFloat(arg.QuoteOrderQuantity, 'f', -1, 64))
	}
	if arg.Price > 0 {
		params.Set("price", strconv.FormatFloat(arg.Price, 'f', -1, 64))
	}
	if arg.StopPrice > 0 {
		params.Set("stopPrice", strconv.FormatFloat(arg.StopPrice, 'f', -1, 64))
	}
	if arg.NewClientOrderID != "" {
		params.Set("newClientOrderId", arg.NewClientOrderID)
	}
	if arg.NewOrderRespType != "" {
		params.Set("newOrderRespType", arg.NewOrderRespType)
	}
	if arg.IcebergQuantity > 0 {
		params.Set("icebergQty", strconv.FormatFloat(arg.IcebergQuantity, 'f', -1, 64))
	}
	if arg.SideEffectType != "" {
		params.Set("sideEffectType", arg.SideEffectType)
	}
	if arg.TimeInForce != "" {
		params.Set("timeInForce", arg.TimeInForce)
	}
	if arg.SelfTradePreventionMode != "" {
		params.Set("selfTradePreventionMode", arg.SelfTradePreventionMode)
	}
	if arg.AutoRepayAtCancel != nil {
		params.Set("autoRepayAtCancel", strconv.FormatBool(*arg.AutoRepayAtCancel))
	}
	var resp *PMNewMarginOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodPost, "/papi/v1/margin/order", params, pmOrderRate, &resp)
}

// PMCancelMarginAccountOrder cancels a margin order identified by its order ID or client order ID
func (e *Exchange) PMCancelMarginAccountOrder(ctx context.Context, arg *PMCancelMarginOrderRequest) (*PMCancelMarginOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params, err := e.pmOrderParams(arg.Symbol, asset.Margin, arg.OrderID, arg.OriginalClientOrderID)
	if err != nil {
		return nil, err
	}
	if arg.NewClientOrderID != "" {
		params.Set("newClientOrderId", arg.NewClientOrderID)
	}
	var resp *PMCancelMarginOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodDelete, "/papi/v1/margin/order", params, pmCancelMarginAccountOrderRate, &resp)
}

// GetUMOrder returns a UM order identified by its order ID or client order ID; cancelled or expired orders without
// fills are not found three days after creation
func (e *Exchange) GetUMOrder(ctx context.Context, symbol currency.Pair, orderID uint64, origClientOrderID string) (*UMOrderDetailResponse, error) {
	params, err := e.pmOrderParams(symbol, asset.USDTMarginedFutures, orderID, origClientOrderID)
	if err != nil {
		return nil, err
	}
	var resp *UMOrderDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/um/order", params, pmDefaultRate, &resp)
}

// NewUMOrder places a UM order
func (e *Exchange) NewUMOrder(ctx context.Context, arg *UMOrderRequest) (*UMOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params, err := e.pmNewFuturesOrderParams(arg.Symbol, asset.USDTMarginedFutures, arg.Side, arg.OrderType, arg.Price, arg.PriceMatch)
	if err != nil {
		return nil, err
	}
	if arg.TimeInForce == "GTD" && arg.GoodTillDate.IsZero() {
		return nil, errGoodTillDateRequired
	}
	if arg.PositionSide != "" {
		params.Set("positionSide", arg.PositionSide)
	}
	if arg.TimeInForce != "" {
		params.Set("timeInForce", arg.TimeInForce)
	}
	if arg.Quantity > 0 {
		params.Set("quantity", strconv.FormatFloat(arg.Quantity, 'f', -1, 64))
	}
	if arg.ReduceOnly {
		params.Set("reduceOnly", "true")
	}
	if arg.NewClientOrderID != "" {
		params.Set("newClientOrderId", arg.NewClientOrderID)
	}
	if arg.NewOrderRespType != "" {
		params.Set("newOrderRespType", arg.NewOrderRespType)
	}
	if arg.SelfTradePreventionMode != "" {
		params.Set("selfTradePreventionMode", arg.SelfTradePreventionMode)
	}
	if !arg.GoodTillDate.IsZero() {
		params.Set("goodTillDate", strconv.FormatInt(arg.GoodTillDate.UnixMilli(), 10))
	}
	var resp *UMOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodPost, "/papi/v1/um/order", params, pmOrderRate, &resp)
}

// CancelUMOrder cancels an open UM LIMIT order identified by its order ID or client order ID
func (e *Exchange) CancelUMOrder(ctx context.Context, symbol currency.Pair, orderID uint64, origClientOrderID string) (*UMOrderResponse, error) {
	params, err := e.pmOrderParams(symbol, asset.USDTMarginedFutures, orderID, origClientOrderID)
	if err != nil {
		return nil, err
	}
	var resp *UMOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodDelete, "/papi/v1/um/order", params, pmDefaultRate, &resp)
}

// GetCMAccountTradeList returns CM account trades of a symbol, or of every symbol of a pair; without a time range the
// last 24 hours are returned, and a range cannot exceed 24 hours
func (e *Exchange) GetCMAccountTradeList(ctx context.Context, arg *CMAccountTradeListRequest) ([]CMAccountTrade, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Symbol.IsEmpty() && arg.Pair.IsEmpty() {
		return nil, fmt.Errorf("%w: symbol or pair is required", currency.ErrCurrencyPairEmpty)
	}
	if !arg.Symbol.IsEmpty() && !arg.Pair.IsEmpty() {
		return nil, errSymbolWithPair
	}
	if arg.FromID > 0 && !arg.Pair.IsEmpty() {
		return nil, errFromIDWithPair
	}
	if arg.FromID > 0 && (!arg.StartTime.IsZero() || !arg.EndTime.IsZero()) {
		return nil, errFromIDWithTimeRange
	}
	params := url.Values{}
	f := pmGetCMAccountTradeListWithPairRate
	if !arg.Symbol.IsEmpty() {
		if err := e.pmSymbolParam(params, arg.Symbol, asset.CoinMarginedFutures); err != nil {
			return nil, err
		}
		f = pmGetCMAccountTradeListWithSymbolRate
	}
	if !arg.Pair.IsEmpty() {
		params.Set("pair", arg.Pair.Upper().String())
	}
	if err := setTimeRangeParams(params, arg.StartTime, arg.EndTime); err != nil {
		return nil, err
	}
	if arg.FromID > 0 {
		params.Set("fromId", strconv.FormatUint(arg.FromID, 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []CMAccountTrade
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/cm/userTrades", params, f, &resp)
}

// GetCMPositionADLQuantileEstimation returns the auto-deleveraging quantiles of one CM symbol's positions, or of every
// CM symbol when symbol is empty; values update every 30 seconds
func (e *Exchange) GetCMPositionADLQuantileEstimation(ctx context.Context, symbol currency.Pair) ([]CMADLQuantileEstimation, error) {
	params := url.Values{}
	if err := e.pmSymbolParam(params, symbol, asset.CoinMarginedFutures); err != nil {
		return nil, err
	}
	var resp []CMADLQuantileEstimation
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/cm/adlQuantile", params, pmGetCMPositionADLQuantileEstimationRate, &resp)
}

// MarginAccountBorrow borrows an asset into the margin account
func (e *Exchange) MarginAccountBorrow(ctx context.Context, ccy currency.Code, amount float64) (*PMTransactionResponse, error) {
	return e.pmMarginAccountBorrowRepay(ctx, "/papi/v1/marginLoan", ccy, amount)
}

func (e *Exchange) pmMarginAccountBorrowRepay(ctx context.Context, path string, ccy currency.Code, amount float64) (*PMTransactionResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("asset", ccy.Upper().String())
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	var resp *PMTransactionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodPost, path, params, pmMarginAccountLoanAndRepayRate, &resp)
}

// MarginAccountNewOCO places a margin OCO order list; both legs share the quantity and the list counts as two orders
// against the order rate limit
func (e *Exchange) MarginAccountNewOCO(ctx context.Context, arg *PMMarginOCOOrderRequest) (*PMNewMarginOCOOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	side, err := pmOrderSide(arg.Side)
	if err != nil {
		return nil, err
	}
	if arg.Quantity <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if arg.Price <= 0 {
		return nil, limits.ErrPriceBelowMin
	}
	if arg.StopPrice <= 0 {
		return nil, fmt.Errorf("%w: stopPrice is required", limits.ErrPriceBelowMin)
	}
	if arg.StopLimitPrice > 0 && arg.StopLimitTimeInForce == "" {
		return nil, errStopLimitTimeInForceRequired
	}
	params := url.Values{}
	if err := e.pmSymbolParam(params, arg.Symbol, asset.Margin); err != nil {
		return nil, err
	}
	if arg.ListClientOrderID != "" {
		params.Set("listClientOrderId", arg.ListClientOrderID)
	}
	params.Set("side", side)
	params.Set("quantity", strconv.FormatFloat(arg.Quantity, 'f', -1, 64))
	if arg.LimitClientOrderID != "" {
		params.Set("limitClientOrderId", arg.LimitClientOrderID)
	}
	params.Set("price", strconv.FormatFloat(arg.Price, 'f', -1, 64))
	if arg.LimitIcebergQuantity > 0 {
		params.Set("limitIcebergQty", strconv.FormatFloat(arg.LimitIcebergQuantity, 'f', -1, 64))
	}
	if arg.StopClientOrderID != "" {
		params.Set("stopClientOrderId", arg.StopClientOrderID)
	}
	params.Set("stopPrice", strconv.FormatFloat(arg.StopPrice, 'f', -1, 64))
	if arg.StopLimitPrice > 0 {
		params.Set("stopLimitPrice", strconv.FormatFloat(arg.StopLimitPrice, 'f', -1, 64))
	}
	if arg.StopIcebergQuantity > 0 {
		params.Set("stopIcebergQty", strconv.FormatFloat(arg.StopIcebergQuantity, 'f', -1, 64))
	}
	if arg.StopLimitTimeInForce != "" {
		params.Set("stopLimitTimeInForce", arg.StopLimitTimeInForce)
	}
	if arg.NewOrderRespType != "" {
		params.Set("newOrderRespType", arg.NewOrderRespType)
	}
	if arg.SideEffectType != "" {
		params.Set("sideEffectType", arg.SideEffectType)
	}
	var resp *PMNewMarginOCOOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodPost, "/papi/v1/margin/order/oco", params, pmOCOOrderRate, &resp)
}

// MarginAccountRepay repays a margin loan
func (e *Exchange) MarginAccountRepay(ctx context.Context, ccy currency.Code, amount float64) (*PMTransactionResponse, error) {
	return e.pmMarginAccountBorrowRepay(ctx, "/papi/v1/repayLoan", ccy, amount)
}

// GetPMMarginAccountTradeList returns margin account trades of a symbol, from FromID onwards when it is set; a time
// range must be shorter than 24 hours
func (e *Exchange) GetPMMarginAccountTradeList(ctx context.Context, arg *PMMarginTradeListRequest) ([]PMMarginTrade, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	if err := e.pmSymbolParam(params, arg.Symbol, asset.Margin); err != nil {
		return nil, err
	}
	if arg.OrderID > 0 {
		params.Set("orderId", strconv.FormatUint(arg.OrderID, 10))
	}
	if err := setTimeRangeParams(params, arg.StartTime, arg.EndTime); err != nil {
		return nil, err
	}
	if arg.FromID > 0 {
		params.Set("fromId", strconv.FormatUint(arg.FromID, 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []PMMarginTrade
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/margin/myTrades", params, pmGetMarginAccountTradeListRate, &resp)
}

// GetAllCMConditionalOrders returns CM conditional orders of one symbol, or of every symbol when the symbol is empty;
// a time range cannot exceed 7 days, which is also the default
func (e *Exchange) GetAllCMConditionalOrders(ctx context.Context, arg *CMConditionalOrdersRequest) ([]CMConditionalOrderRecord, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params := url.Values{}
	f := pmAllCMConditionalOrderWithoutSymbolRate
	if !arg.Symbol.IsEmpty() {
		if err := e.pmSymbolParam(params, arg.Symbol, asset.CoinMarginedFutures); err != nil {
			return nil, err
		}
		f = pmDefaultRate
	}
	if arg.StrategyID > 0 {
		params.Set("strategyId", strconv.FormatUint(arg.StrategyID, 10))
	}
	if err := setTimeRangeParams(params, arg.StartTime, arg.EndTime); err != nil {
		return nil, err
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []CMConditionalOrderRecord
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/cm/conditional/allOrders", params, f, &resp)
}

// GetAllCMOrders returns active, cancelled and filled CM orders of a symbol or pair; cancelled or expired orders
// without fills are not found three days after creation
func (e *Exchange) GetAllCMOrders(ctx context.Context, arg *CMOrdersRequest) ([]CMOrderDetailResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Symbol.IsEmpty() && arg.Pair.IsEmpty() {
		return nil, fmt.Errorf("%w: symbol or pair is required", currency.ErrCurrencyPairEmpty)
	}
	params := url.Values{}
	f := pmAllCMOrderWithPairRate
	if !arg.Symbol.IsEmpty() {
		if err := e.pmSymbolParam(params, arg.Symbol, asset.CoinMarginedFutures); err != nil {
			return nil, err
		}
		f = pmAllCMOrderWithSymbolRate
	}
	if !arg.Pair.IsEmpty() {
		params.Set("pair", arg.Pair.Upper().String())
	}
	if arg.OrderID > 0 {
		params.Set("orderId", strconv.FormatUint(arg.OrderID, 10))
	}
	if err := setTimeRangeParams(params, arg.StartTime, arg.EndTime); err != nil {
		return nil, err
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []CMOrderDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/cm/allOrders", params, f, &resp)
}

// GetAllCMOpenConditionalOrders returns the open CM conditional orders of one symbol, or of every symbol when symbol
// is empty
func (e *Exchange) GetAllCMOpenConditionalOrders(ctx context.Context, symbol currency.Pair) ([]CMConditionalOrderResponse, error) {
	params := url.Values{}
	f := pmAllCMOpenConditionalOrdersWithoutSymbolRate
	if !symbol.IsEmpty() {
		if err := e.pmSymbolParam(params, symbol, asset.CoinMarginedFutures); err != nil {
			return nil, err
		}
		f = pmDefaultRate
	}
	var resp []CMConditionalOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/cm/conditional/openOrders", params, f, &resp)
}

// GetAllCMOpenOrders returns the open CM orders of a symbol or of a pair (such as BTCUSD, the base of a COIN-M contract
// pair), or of every symbol when both are empty
func (e *Exchange) GetAllCMOpenOrders(ctx context.Context, symbol currency.Pair, pair currency.Code) ([]CMOrderDetailResponse, error) {
	params := url.Values{}
	f := pmRetrieveAllCMOpenOrdersForAllSymbolRate
	if !symbol.IsEmpty() {
		if err := e.pmSymbolParam(params, symbol, asset.CoinMarginedFutures); err != nil {
			return nil, err
		}
		f = pmDefaultRate
	}
	if !pair.IsEmpty() {
		params.Set("pair", pair.Upper().String())
	}
	var resp []CMOrderDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/cm/openOrders", params, f, &resp)
}

// GetAllUMOpenOrders returns the open UM orders of one symbol, or of every symbol when symbol is empty
func (e *Exchange) GetAllUMOpenOrders(ctx context.Context, symbol currency.Pair) ([]UMOrderDetailResponse, error) {
	params := url.Values{}
	f := pmRetrieveAllUMOpenOrdersForAllSymbolRate
	if !symbol.IsEmpty() {
		if err := e.pmSymbolParam(params, symbol, asset.USDTMarginedFutures); err != nil {
			return nil, err
		}
		f = pmDefaultRate
	}
	var resp []UMOrderDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/um/openOrders", params, f, &resp)
}

// GetAllMarginAccountOrders returns margin orders of a symbol, from OrderID onwards when it is set
func (e *Exchange) GetAllMarginAccountOrders(ctx context.Context, arg *PMMarginOrdersRequest) ([]PMMarginOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params, err := e.pmOrdersParams(arg.Symbol, asset.Margin, arg.OrderID, arg.StartTime, arg.EndTime, arg.Limit)
	if err != nil {
		return nil, err
	}
	var resp []PMMarginOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/margin/allOrders", params, pmAllMarginAccountOrdersRate, &resp)
}

// pmOrdersParams builds the parameters of the order history queries that require a symbol
func (e *Exchange) pmOrdersParams(symbol currency.Pair, a asset.Item, orderID uint64, startTime, endTime time.Time, limit uint64) (url.Values, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	if err := e.pmSymbolParam(params, symbol, a); err != nil {
		return nil, err
	}
	if orderID > 0 {
		params.Set("orderId", strconv.FormatUint(orderID, 10))
	}
	if err := setTimeRangeParams(params, startTime, endTime); err != nil {
		return nil, err
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	return params, nil
}

// GetAllUMOrders returns active, cancelled and filled UM orders of a symbol; a time range cannot exceed 7 days, which
// is also the default
func (e *Exchange) GetAllUMOrders(ctx context.Context, arg *UMOrdersRequest) ([]UMOrderDetailResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params, err := e.pmOrdersParams(arg.Symbol, asset.USDTMarginedFutures, arg.OrderID, arg.StartTime, arg.EndTime, arg.Limit)
	if err != nil {
		return nil, err
	}
	var resp []UMOrderDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/um/allOrders", params, pmGetAllUMOrdersRate, &resp)
}

// GetCMConditionalOrderHistory returns a triggered, cancelled or expired CM conditional order identified by its
// strategy ID or client strategy ID; NEW orders are not found
func (e *Exchange) GetCMConditionalOrderHistory(ctx context.Context, symbol currency.Pair, strategyID uint64, newClientStrategyID string) (*CMConditionalOrderHistoryResponse, error) {
	params, err := e.pmConditionalOrderParams(symbol, strategyID, newClientStrategyID)
	if err != nil {
		return nil, err
	}
	var resp *CMConditionalOrderHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/cm/conditional/orderHistory", params, pmDefaultRate, &resp)
}

// GetOpenCMConditionalOrder returns an open CM conditional order identified by its strategy ID or client strategy ID
func (e *Exchange) GetOpenCMConditionalOrder(ctx context.Context, symbol currency.Pair, strategyID uint64, newClientStrategyID string) (*CMConditionalOrderResponse, error) {
	params, err := e.pmConditionalOrderParams(symbol, strategyID, newClientStrategyID)
	if err != nil {
		return nil, err
	}
	var resp *CMConditionalOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/cm/conditional/openOrder", params, pmDefaultRate, &resp)
}

// GetCMOpenOrder returns an open CM order identified by its order ID or client order ID; unlike its UM counterpart the
// endpoint answers with an array
func (e *Exchange) GetCMOpenOrder(ctx context.Context, symbol currency.Pair, orderID uint64, origClientOrderID string) ([]CMOrderDetailResponse, error) {
	params, err := e.pmOrderParams(symbol, asset.CoinMarginedFutures, orderID, origClientOrderID)
	if err != nil {
		return nil, err
	}
	var resp []CMOrderDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/cm/openOrder", params, pmDefaultRate, &resp)
}

// GetCurrentMarginOpenOrders returns the open margin orders of one symbol, or of every symbol when symbol is empty,
// which costs a request per trading symbol against the rate limiter
func (e *Exchange) GetCurrentMarginOpenOrders(ctx context.Context, symbol currency.Pair) ([]PMMarginOrderResponse, error) {
	params := url.Values{}
	if err := e.pmSymbolParam(params, symbol, asset.Margin); err != nil {
		return nil, err
	}
	var resp []PMMarginOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/margin/openOrders", params, pmCurrentMarginOpenOrderRate, &resp)
}

// GetUMOpenOrder returns an open UM order identified by its order ID or client order ID
func (e *Exchange) GetUMOpenOrder(ctx context.Context, symbol currency.Pair, orderID uint64, origClientOrderID string) (*UMOrderDetailResponse, error) {
	params, err := e.pmOrderParams(symbol, asset.USDTMarginedFutures, orderID, origClientOrderID)
	if err != nil {
		return nil, err
	}
	var resp *UMOrderDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/um/openOrder", params, pmDefaultRate, &resp)
}

// GetPMMarginAccountAllOCO returns margin OCO order lists, from FromID onwards when it is set
func (e *Exchange) GetPMMarginAccountAllOCO(ctx context.Context, arg *PMMarginAllOCORequest) ([]PMMarginOCOOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params := url.Values{}
	if arg.FromID > 0 {
		params.Set("fromId", strconv.FormatUint(arg.FromID, 10))
	}
	if err := setTimeRangeParams(params, arg.StartTime, arg.EndTime); err != nil {
		return nil, err
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []PMMarginOCOOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/margin/allOrderList", params, pmGetMarginAccountsAllOCOOrdersRate, &resp)
}

// GetMarginAccountsOpenOCO returns the open margin OCO order lists
func (e *Exchange) GetMarginAccountsOpenOCO(ctx context.Context) ([]PMMarginOCOOrderResponse, error) {
	var resp []PMMarginOCOOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/margin/openOrderList", nil, pmGetMarginAccountsOpenOCOOrdersRate, &resp)
}

// GetUsersCMForceOrders returns the user's CM liquidation and auto-deleveraging orders of the last 90 days
func (e *Exchange) GetUsersCMForceOrders(ctx context.Context, arg *PMForceOrdersRequest) ([]CMForceOrder, error) {
	params, f, err := e.pmForceOrdersParams(arg, asset.CoinMarginedFutures, pmGetUserCMForceOrdersWithSymbolRate, pmGetUserCMForceOrdersWithoutSymbolRate)
	if err != nil {
		return nil, err
	}
	var resp []CMForceOrder
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/cm/forceOrders", params, f, &resp)
}

// pmForceOrdersParams builds the parameters of a UM or CM force orders query and selects its weight
func (e *Exchange) pmForceOrdersParams(arg *PMForceOrdersRequest, a asset.Item, withSymbol, withoutSymbol request.EndpointLimit) (url.Values, request.EndpointLimit, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, 0, err
	}
	params := url.Values{}
	f := withoutSymbol
	if !arg.Symbol.IsEmpty() {
		if err := e.pmSymbolParam(params, arg.Symbol, a); err != nil {
			return nil, 0, err
		}
		f = withSymbol
	}
	if arg.AutoCloseType != "" {
		params.Set("autoCloseType", arg.AutoCloseType)
	}
	if err := setTimeRangeParams(params, arg.StartTime, arg.EndTime); err != nil {
		return nil, 0, err
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	return params, f, nil
}

// GetUsersMarginForceOrders returns the user's margin liquidation orders; accounts liquidated through risk-based
// liquidation adjustment have no records here
func (e *Exchange) GetUsersMarginForceOrders(ctx context.Context, arg *PMMarginForceOrdersRequest) (*PMMarginForceOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params := url.Values{}
	if err := setTimeRangeParams(params, arg.StartTime, arg.EndTime); err != nil {
		return nil, err
	}
	if arg.Current > 0 {
		params.Set("current", strconv.FormatUint(arg.Current, 10))
	}
	if arg.Size > 0 {
		params.Set("size", strconv.FormatUint(arg.Size, 10))
	}
	var resp *PMMarginForceOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/margin/forceOrders", params, pmDefaultRate, &resp)
}

// GetUsersUMForceOrders returns the user's UM liquidation and auto-deleveraging orders of the last 90 days
func (e *Exchange) GetUsersUMForceOrders(ctx context.Context, arg *PMForceOrdersRequest) ([]UMForceOrder, error) {
	params, f, err := e.pmForceOrdersParams(arg, asset.USDTMarginedFutures, pmGetUserUMForceOrdersWithSymbolRate, pmGetUserUMForceOrdersWithoutSymbolRate)
	if err != nil {
		return nil, err
	}
	var resp []UMForceOrder
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/um/forceOrders", params, f, &resp)
}

// GetUMAccountTradeList returns UM account trades of a symbol; without a time range the last 7 days are returned, and
// a range cannot exceed 7 days
func (e *Exchange) GetUMAccountTradeList(ctx context.Context, arg *UMAccountTradeListRequest) ([]UMAccountTrade, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if arg.FromID > 0 && (!arg.StartTime.IsZero() || !arg.EndTime.IsZero()) {
		return nil, errFromIDWithTimeRange
	}
	params := url.Values{}
	if err := e.pmSymbolParam(params, arg.Symbol, asset.USDTMarginedFutures); err != nil {
		return nil, err
	}
	if err := setTimeRangeParams(params, arg.StartTime, arg.EndTime); err != nil {
		return nil, err
	}
	if arg.FromID > 0 {
		params.Set("fromId", strconv.FormatUint(arg.FromID, 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []UMAccountTrade
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/um/userTrades", params, pmGetUMAccountTradeListRate, &resp)
}

// GetUMPositionADLQuantileEstimation returns the auto-deleveraging quantiles of one UM symbol's positions, or of every
// UM symbol when symbol is empty; values update every 30 seconds
func (e *Exchange) GetUMPositionADLQuantileEstimation(ctx context.Context, symbol currency.Pair) ([]UMADLQuantileEstimation, error) {
	params := url.Values{}
	if err := e.pmSymbolParam(params, symbol, asset.USDTMarginedFutures); err != nil {
		return nil, err
	}
	var resp []UMADLQuantileEstimation
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/um/adlQuantile", params, pmGetUMPositionADLQuantileEstimationRate, &resp)
}

// NewUMAlgoOrder places a UM conditional order through the algo service, which replaced the retired UM conditional
// order endpoints on 2026-04-28
func (e *Exchange) NewUMAlgoOrder(ctx context.Context, arg *UMAlgoOrderRequest) (*UMAlgoOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params, err := e.pmNewFuturesOrderParams(arg.Symbol, asset.USDTMarginedFutures, arg.Side, arg.OrderType, arg.Price, arg.PriceMatch)
	if err != nil {
		return nil, err
	}
	if arg.AlgoType == "" {
		return nil, errAlgoTypeRequired
	}
	if arg.TimeInForce == "GTD" && arg.GoodTillDate.IsZero() {
		return nil, errGoodTillDateRequired
	}
	if arg.ClosePosition && (arg.Quantity > 0 || arg.ReduceOnly) {
		return nil, errClosePositionConflict
	}
	params.Set("algoType", arg.AlgoType)
	if arg.PositionSide != "" {
		params.Set("positionSide", arg.PositionSide)
	}
	if arg.TimeInForce != "" {
		params.Set("timeInForce", arg.TimeInForce)
	}
	if arg.Quantity > 0 {
		params.Set("quantity", strconv.FormatFloat(arg.Quantity, 'f', -1, 64))
	}
	if arg.TriggerPrice > 0 {
		params.Set("triggerPrice", strconv.FormatFloat(arg.TriggerPrice, 'f', -1, 64))
	}
	if arg.WorkingType != "" {
		params.Set("workingType", arg.WorkingType)
	}
	if arg.ClosePosition {
		params.Set("closePosition", "true")
	}
	if arg.PriceProtect {
		params.Set("priceProtect", "true")
	}
	if arg.ReduceOnly {
		params.Set("reduceOnly", "true")
	}
	if arg.ActivatePrice > 0 {
		params.Set("activatePrice", strconv.FormatFloat(arg.ActivatePrice, 'f', -1, 64))
	}
	if arg.CallbackRate > 0 {
		params.Set("callbackRate", strconv.FormatFloat(arg.CallbackRate, 'f', -1, 64))
	}
	if arg.ClientAlgoID != "" {
		params.Set("clientAlgoId", arg.ClientAlgoID)
	}
	if arg.NewOrderRespType != "" {
		params.Set("newOrderRespType", arg.NewOrderRespType)
	}
	if arg.SelfTradePreventionMode != "" {
		params.Set("selfTradePreventionMode", arg.SelfTradePreventionMode)
	}
	if !arg.GoodTillDate.IsZero() {
		params.Set("goodTillDate", strconv.FormatInt(arg.GoodTillDate.UnixMilli(), 10))
	}
	var resp *UMAlgoOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodPost, "/papi/v1/um/algo/order", params, pmOrderRate, &resp)
}

// CancelUMAlgoOrder cancels an open UM algo order identified by its algo ID or client algo ID
func (e *Exchange) CancelUMAlgoOrder(ctx context.Context, algoID uint64, clientAlgoID string) (*CancelUMAlgoOrderResponse, error) {
	if algoID == 0 && clientAlgoID == "" {
		return nil, fmt.Errorf("%w: algoId or clientAlgoId is required", order.ErrOrderIDNotSet)
	}
	params := url.Values{}
	if algoID > 0 {
		params.Set("algoId", strconv.FormatUint(algoID, 10))
	}
	if clientAlgoID != "" {
		params.Set("clientAlgoId", clientAlgoID)
	}
	var resp *CancelUMAlgoOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodDelete, "/papi/v1/um/algo/order", params, pmDefaultRate, &resp)
}

// CancelAllUMAlgoOpenOrders cancels every open UM algo order of a symbol
func (e *Exchange) CancelAllUMAlgoOpenOrders(ctx context.Context, symbol currency.Pair) (*SuccessResponse, error) {
	return e.pmCancelAllOpenOrders(ctx, "/papi/v1/um/algo/allOpenOrders", symbol, asset.USDTMarginedFutures)
}

// GetUMAlgoOrder returns a UM algo order identified by its algo ID or client algo ID. Binance drops cancelled or
// expired orders without fills after three days, and every order after 90 days
func (e *Exchange) GetUMAlgoOrder(ctx context.Context, algoID uint64, clientAlgoID string) (*UMAlgoOrderDetailResponse, error) {
	if algoID == 0 && clientAlgoID == "" {
		return nil, fmt.Errorf("%w: algoId or clientAlgoId is required", order.ErrOrderIDNotSet)
	}
	params := url.Values{}
	if algoID > 0 {
		params.Set("algoId", strconv.FormatUint(algoID, 10))
	}
	if clientAlgoID != "" {
		params.Set("clientAlgoId", clientAlgoID)
	}
	var resp *UMAlgoOrderDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/um/algo/algoOrder", params, pmDefaultRate, &resp)
}

// GetUMOpenAlgoOrders returns the open UM algo orders of a symbol, or of every symbol when symbol is empty; algoType
// and algoID narrow them when set
func (e *Exchange) GetUMOpenAlgoOrders(ctx context.Context, symbol currency.Pair, algoType string, algoID uint64) ([]UMAlgoOrder, error) {
	params := url.Values{}
	if err := e.pmSymbolParam(params, symbol, asset.USDTMarginedFutures); err != nil {
		return nil, err
	}
	if algoType != "" {
		params.Set("algoType", algoType)
	}
	if algoID > 0 {
		params.Set("algoId", strconv.FormatUint(algoID, 10))
	}
	var resp []UMAlgoOrder
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/um/algo/openAlgoOrders", params, pmDefaultRate, &resp)
}

// GetUMAlgoOrderHistory returns a symbol's UM algo orders whatever their status, from AlgoID on when it is set
func (e *Exchange) GetUMAlgoOrderHistory(ctx context.Context, arg *UMAlgoOrderHistoryRequest) ([]UMAlgoOrderDetailResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if !arg.StartTime.IsZero() && !arg.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(arg.StartTime, arg.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if err := e.pmSymbolParam(params, arg.Symbol, asset.USDTMarginedFutures); err != nil {
		return nil, err
	}
	if arg.AlgoID > 0 {
		params.Set("algoId", strconv.FormatUint(arg.AlgoID, 10))
	}
	if !arg.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(arg.StartTime.UnixMilli(), 10))
	}
	if !arg.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(arg.EndTime.UnixMilli(), 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []UMAlgoOrderDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/um/algo/allAlgoOrders", params, pmGetUMAlgoOrderHistoryRate, &resp)
}
