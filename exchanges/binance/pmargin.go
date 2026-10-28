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

// portfolioMarginOrderSide returns the wire value of a buy or sell order side
func portfolioMarginOrderSide(side order.Side) (string, error) {
	switch side {
	case order.Buy, order.Sell:
		return side.String(), nil
	default:
		return "", fmt.Errorf("%w: %q", order.ErrSideIsInvalid, side)
	}
}

// portfolioMarginSymbolParam sets the symbol parameter when a pair is given
func (e *Exchange) portfolioMarginSymbolParam(params url.Values, pair currency.Pair, a asset.Item) error {
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

// GetPortfolioMarginAccountBalance returns the portfolio margin account balance of one asset, or of every asset when
// asset is empty
func (e *Exchange) GetPortfolioMarginAccountBalance(ctx context.Context, ccy currency.Code) ([]AccountBalanceResponse, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("asset", ccy.Upper().String())
	}
	var resp objectOrArray[AccountBalanceResponse]
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/balance", params, portfolioMarginGetAccountBalancesRate, &resp)
}

// GetPortfolioMarginAccountInformation returns the portfolio margin account information
func (e *Exchange) GetPortfolioMarginAccountInformation(ctx context.Context) (*PortfolioMarginAccountInformationResponse, error) {
	var resp *PortfolioMarginAccountInformationResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/account", nil, portfolioMarginGetAccountInformationRate, &resp)
}

// PortfolioMarginBNBTransfer transfers BNB into or out of the UM account; transferSide is TO_UM or FROM_UM. The
// endpoint can be called 10 times per 10 minutes
func (e *Exchange) PortfolioMarginBNBTransfer(ctx context.Context, amount float64, transferSide string) (*PortfolioMarginTransactionResponse, error) {
	return e.portfolioMarginTransferBNB(ctx, exchange.RestFuturesSupplementary, "/papi/v1/bnb-transfer", amount, transferSide, portfolioMarginBNBTransferRate)
}

func (e *Exchange) portfolioMarginTransferBNB(ctx context.Context, ePath exchange.URL, path string, amount float64, transferSide string, f request.EndpointLimit) (*PortfolioMarginTransactionResponse, error) {
	if amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if transferSide == "" {
		return nil, errTransferSideRequired
	}
	params := url.Values{}
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	params.Set("transferSide", transferSide)
	var resp *PortfolioMarginTransactionResponse
	return resp, e.SendAuthHTTPRequest(ctx, ePath, http.MethodPost, path, params, f, &resp)
}

// GetPortfolioMarginAutoRepayFuturesStatus returns whether futures negative balances are repaid automatically
func (e *Exchange) GetPortfolioMarginAutoRepayFuturesStatus(ctx context.Context) (*PortfolioMarginAutoRepayFuturesStatusResponse, error) {
	var resp *PortfolioMarginAutoRepayFuturesStatusResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/repay-futures-switch", nil, portfolioMarginGetAutoRepayFuturesStatusRate, &resp)
}

// ChangePortfolioMarginAutoRepayFuturesStatus turns the automatic repayment of futures negative balances on or off. The
// endpoint can be called 20 times a day
func (e *Exchange) ChangePortfolioMarginAutoRepayFuturesStatus(ctx context.Context, autoRepay bool) (*PortfolioMarginMessageResponse, error) {
	return e.portfolioMarginChangeAutoRepayFuturesStatus(ctx, exchange.RestFuturesSupplementary, "/papi/v1/repay-futures-switch", autoRepay, portfolioMarginChangeAutoRepayFuturesStatusRate)
}

func (e *Exchange) portfolioMarginChangeAutoRepayFuturesStatus(ctx context.Context, ePath exchange.URL, path string, autoRepay bool, f request.EndpointLimit) (*PortfolioMarginMessageResponse, error) {
	params := url.Values{}
	params.Set("autoRepay", strconv.FormatBool(autoRepay))
	var resp *PortfolioMarginMessageResponse
	return resp, e.SendAuthHTTPRequest(ctx, ePath, http.MethodPost, path, params, f, &resp)
}

// ChangePortfolioMarginCoinMarginedInitialLeverage changes the initial leverage of a CM symbol
func (e *Exchange) ChangePortfolioMarginCoinMarginedInitialLeverage(ctx context.Context, symbol currency.Pair, leverage uint64) (*PortfolioMarginCoinMarginedInitialLeverageResponse, error) {
	params, err := e.portfolioMarginLeverageParams(symbol, asset.CoinMarginedFutures, leverage)
	if err != nil {
		return nil, err
	}
	var resp *PortfolioMarginCoinMarginedInitialLeverageResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodPost, "/papi/v1/cm/leverage", params, portfolioMarginDefaultRate, &resp)
}

func (e *Exchange) portfolioMarginLeverageParams(symbol currency.Pair, a asset.Item, leverage uint64) (url.Values, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if leverage == 0 {
		return nil, errLeverageRequired
	}
	params := url.Values{}
	if err := e.portfolioMarginSymbolParam(params, symbol, a); err != nil {
		return nil, err
	}
	params.Set("leverage", strconv.FormatUint(leverage, 10))
	return params, nil
}

// GetPortfolioMarginCoinMarginedCurrentPositionMode returns whether hedge mode is enabled for every CM symbol
func (e *Exchange) GetPortfolioMarginCoinMarginedCurrentPositionMode(ctx context.Context) (*PortfolioMarginPositionModeResponse, error) {
	var resp *PortfolioMarginPositionModeResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/cm/positionSide/dual", nil, portfolioMarginGetCoinMarginedCurrentPositionModeRate, &resp)
}

// ChangePortfolioMarginCoinMarginedPositionMode switches every CM symbol between hedge mode (true) and one-way mode
// (false); the CM position mode must match the UM position mode
func (e *Exchange) ChangePortfolioMarginCoinMarginedPositionMode(ctx context.Context, dualSidePosition bool) (*SuccessResponse, error) {
	return e.portfolioMarginChangePositionMode(ctx, "/papi/v1/cm/positionSide/dual", dualSidePosition)
}

func (e *Exchange) portfolioMarginChangePositionMode(ctx context.Context, path string, dualSidePosition bool) (*SuccessResponse, error) {
	params := url.Values{}
	params.Set("dualSidePosition", strconv.FormatBool(dualSidePosition))
	var resp *SuccessResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodPost, path, params, portfolioMarginDefaultRate, &resp)
}

// ChangePortfolioMarginUSDTMarginedInitialLeverage changes the initial leverage of a UM symbol
func (e *Exchange) ChangePortfolioMarginUSDTMarginedInitialLeverage(ctx context.Context, symbol currency.Pair, leverage uint64) (*PortfolioMarginUSDTMarginedInitialLeverageResponse, error) {
	params, err := e.portfolioMarginLeverageParams(symbol, asset.USDTMarginedFutures, leverage)
	if err != nil {
		return nil, err
	}
	var resp *PortfolioMarginUSDTMarginedInitialLeverageResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodPost, "/papi/v1/um/leverage", params, portfolioMarginDefaultRate, &resp)
}

// GetPortfolioMarginUSDTMarginedCurrentPositionMode returns whether hedge mode is enabled for every UM symbol
func (e *Exchange) GetPortfolioMarginUSDTMarginedCurrentPositionMode(ctx context.Context) (*PortfolioMarginPositionModeResponse, error) {
	var resp *PortfolioMarginPositionModeResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/um/positionSide/dual", nil, portfolioMarginGetUSDTMarginedCurrentPositionModeRate, &resp)
}

// ChangePortfolioMarginUSDTMarginedPositionMode switches every UM symbol between hedge mode (true) and one-way mode
// (false)
func (e *Exchange) ChangePortfolioMarginUSDTMarginedPositionMode(ctx context.Context, dualSidePosition bool) (*SuccessResponse, error) {
	return e.portfolioMarginChangePositionMode(ctx, "/papi/v1/um/positionSide/dual", dualSidePosition)
}

// GetPortfolioMarginCoinMarginedNotionalAndLeverageBrackets returns the notional and leverage brackets of one CM
// symbol, or of every CM symbol when symbol is empty
func (e *Exchange) GetPortfolioMarginCoinMarginedNotionalAndLeverageBrackets(ctx context.Context, symbol currency.Pair) ([]PortfolioMarginCoinMarginedNotionalAndLeverageResponse, error) {
	params := url.Values{}
	if err := e.portfolioMarginSymbolParam(params, symbol, asset.CoinMarginedFutures); err != nil {
		return nil, err
	}
	var resp []PortfolioMarginCoinMarginedNotionalAndLeverageResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/cm/leverageBracket", params, portfolioMarginDefaultRate, &resp)
}

// PortfolioMarginFundAutoCollection collects every asset except BNB from the futures accounts into the margin account.
// The endpoint can be called 500 times per hour
func (e *Exchange) PortfolioMarginFundAutoCollection(ctx context.Context) (*PortfolioMarginMessageResponse, error) {
	var resp *PortfolioMarginMessageResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodPost, "/papi/v1/auto-collection", nil, portfolioMarginFundAutoCollectionRate, &resp)
}

// PortfolioMarginFundCollectionByAsset collects an asset other than BNB from the futures accounts into the margin
// account
func (e *Exchange) PortfolioMarginFundCollectionByAsset(ctx context.Context, ccy currency.Code) (*PortfolioMarginMessageResponse, error) {
	return e.portfolioMarginFundCollectionByAsset(ctx, exchange.RestFuturesSupplementary, "/papi/v1/asset-collection", ccy, portfolioMarginFundCollectionByAssetRate)
}

func (e *Exchange) portfolioMarginFundCollectionByAsset(ctx context.Context, ePath exchange.URL, path string, ccy currency.Code, f request.EndpointLimit) (*PortfolioMarginMessageResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("asset", ccy.Upper().String())
	var resp *PortfolioMarginMessageResponse
	return resp, e.SendAuthHTTPRequest(ctx, ePath, http.MethodPost, path, params, f, &resp)
}

// GetPortfolioMarginCoinMarginedAccountDetail returns the CM account assets and positions
func (e *Exchange) GetPortfolioMarginCoinMarginedAccountDetail(ctx context.Context) (*PortfolioMarginCoinMarginedAccountDetailResponse, error) {
	var resp *PortfolioMarginCoinMarginedAccountDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/cm/account", nil, portfolioMarginGetCoinMarginedAccountDetailRate, &resp)
}

// GetPortfolioMarginCoinMarginedIncomeHistory returns CM income history; without a time range the last 200 days are
// returned, and a range cannot exceed 200 days
func (e *Exchange) GetPortfolioMarginCoinMarginedIncomeHistory(ctx context.Context, arg *PortfolioMarginIncomeHistoryRequest) ([]IncomeItemResponse, error) {
	return e.portfolioMarginIncomeHistory(ctx, "/papi/v1/cm/income", asset.CoinMarginedFutures, arg, portfolioMarginGetCoinMarginedIncomeHistoryRate)
}

func (e *Exchange) portfolioMarginIncomeHistory(ctx context.Context, path string, a asset.Item, arg *PortfolioMarginIncomeHistoryRequest, f request.EndpointLimit) ([]IncomeItemResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params := url.Values{}
	if err := e.portfolioMarginSymbolParam(params, arg.Symbol, a); err != nil {
		return nil, err
	}
	if arg.IncomeType != "" {
		params.Set("incomeType", string(arg.IncomeType))
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
	var resp []IncomeItemResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, path, params, f, &resp)
}

// GetPortfolioMarginMarginBorrowOrLoanInterestHistory returns margin borrow and loan interest records, newest first;
// without a time range the last 7 days are returned, and a range cannot exceed 30 days
func (e *Exchange) GetPortfolioMarginMarginBorrowOrLoanInterestHistory(ctx context.Context, arg *PortfolioMarginMarginInterestHistoryRequest) (*PortfolioMarginMarginInterestHistoryResponse, error) {
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
	var resp *PortfolioMarginMarginInterestHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/margin/marginInterestHistory", params, portfolioMarginDefaultRate, &resp)
}

// GetPortfolioMarginUSDTMarginedAccountDetail returns the UM account assets and positions
func (e *Exchange) GetPortfolioMarginUSDTMarginedAccountDetail(ctx context.Context) (*PortfolioMarginUSDTMarginedAccountDetailResponse, error) {
	var resp *PortfolioMarginUSDTMarginedAccountDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/um/account", nil, portfolioMarginGetUSDTMarginedAccountDetailRate, &resp)
}

// GetPortfolioMarginUSDTMarginedIncomeHistory returns UM income history of the last three months; without a time range
// the last 7 days are returned
func (e *Exchange) GetPortfolioMarginUSDTMarginedIncomeHistory(ctx context.Context, arg *PortfolioMarginIncomeHistoryRequest) ([]IncomeItemResponse, error) {
	return e.portfolioMarginIncomeHistory(ctx, "/papi/v1/um/income", asset.USDTMarginedFutures, arg, portfolioMarginGetUSDTMarginedIncomeHistoryRate)
}

// GetPortfolioMarginCoinMarginedUserCommissionRate returns the user's commission rates for a CM symbol
func (e *Exchange) GetPortfolioMarginCoinMarginedUserCommissionRate(ctx context.Context, symbol currency.Pair) (*PortfolioMarginCommissionRateResponse, error) {
	return e.portfolioMarginUserCommissionRate(ctx, "/papi/v1/cm/commissionRate", symbol, asset.CoinMarginedFutures, portfolioMarginGetCoinMarginedUserCommissionRate)
}

func (e *Exchange) portfolioMarginUserCommissionRate(ctx context.Context, path string, symbol currency.Pair, a asset.Item, f request.EndpointLimit) (*PortfolioMarginCommissionRateResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	if err := e.portfolioMarginSymbolParam(params, symbol, a); err != nil {
		return nil, err
	}
	var resp *PortfolioMarginCommissionRateResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, path, params, f, &resp)
}

// GetPortfolioMarginUSDTMarginedUserCommissionRate returns the user's commission rates for a UM symbol
func (e *Exchange) GetPortfolioMarginUSDTMarginedUserCommissionRate(ctx context.Context, symbol currency.Pair) (*PortfolioMarginCommissionRateResponse, error) {
	return e.portfolioMarginUserCommissionRate(ctx, "/papi/v1/um/commissionRate", symbol, asset.USDTMarginedFutures, portfolioMarginGetUSDTMarginedUserCommissionRate)
}

// GetPortfolioMarginMarginMaxBorrow returns the maximum amount of an asset the margin account can borrow
func (e *Exchange) GetPortfolioMarginMarginMaxBorrow(ctx context.Context, ccy currency.Code) (*PortfolioMarginMarginMaxBorrowResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("asset", ccy.Upper().String())
	var resp *PortfolioMarginMarginMaxBorrowResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/margin/maxBorrowable", params, portfolioMarginMarginMaxBorrowRate, &resp)
}

// GetPortfolioMarginUSDTMarginedTradingQuantitativeRulesIndicator returns the UM trading quantitative rules indicators
// of one symbol, or of every symbol when symbol is empty
func (e *Exchange) GetPortfolioMarginUSDTMarginedTradingQuantitativeRulesIndicator(ctx context.Context, symbol currency.Pair) (*PortfolioMarginTradingQuantitativeRulesIndicatorsResponse, error) {
	params := url.Values{}
	f := portfolioMarginUSDTMarginedTradingQuantitativeRulesIndicatorsRate
	if !symbol.IsEmpty() {
		if err := e.portfolioMarginSymbolParam(params, symbol, asset.USDTMarginedFutures); err != nil {
			return nil, err
		}
		f = portfolioMarginDefaultRate
	}
	var resp *PortfolioMarginTradingQuantitativeRulesIndicatorsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/um/apiTradingStatus", params, f, &resp)
}

// GetPortfolioMarginCoinMarginedPositionInformation returns CM positions, filtered by margin asset or by pair (such as
// BTCUSD, the base of a COIN-M contract pair); without either the positions of every trading symbol are returned
func (e *Exchange) GetPortfolioMarginCoinMarginedPositionInformation(ctx context.Context, marginAsset, pair currency.Code) ([]PortfolioMarginCoinMarginedPositionInformationResponse, error) {
	params := url.Values{}
	if !marginAsset.IsEmpty() {
		params.Set("marginAsset", marginAsset.Upper().String())
	}
	if !pair.IsEmpty() {
		params.Set("pair", pair.Upper().String())
	}
	var resp []PortfolioMarginCoinMarginedPositionInformationResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/cm/positionRisk", params, portfolioMarginDefaultRate, &resp)
}

// GetPortfolioMarginMarginLoanRecord returns margin loan records, newest first; a transaction ID or start time is
// required, and a time range cannot exceed 30 days
func (e *Exchange) GetPortfolioMarginMarginLoanRecord(ctx context.Context, arg *PortfolioMarginMarginLoanRepayRecordRequest) (*PortfolioMarginMarginLoanRecordResponse, error) {
	params, err := portfolioMarginLoanRepayRecordParams(arg)
	if err != nil {
		return nil, err
	}
	var resp *PortfolioMarginMarginLoanRecordResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/margin/marginLoan", params, portfolioMarginGetMarginLoanRecordRate, &resp)
}

func portfolioMarginLoanRepayRecordParams(arg *PortfolioMarginMarginLoanRepayRecordRequest) (url.Values, error) {
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

// GetPortfolioMarginMarginMaxWithdrawal returns the maximum amount of an asset that can be withdrawn from the margin
// account
func (e *Exchange) GetPortfolioMarginMarginMaxWithdrawal(ctx context.Context, ccy currency.Code) (*PortfolioMarginMarginMaxWithdrawResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("asset", ccy.Upper().String())
	var resp *PortfolioMarginMarginMaxWithdrawResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/margin/maxWithdraw", params, portfolioMarginGetMarginMaxWithdrawalRate, &resp)
}

// GetPortfolioMarginMarginRepayRecord returns margin repay records, newest first; a transaction ID or start time is
// required, and a time range cannot exceed 30 days
func (e *Exchange) GetPortfolioMarginMarginRepayRecord(ctx context.Context, arg *PortfolioMarginMarginLoanRepayRecordRequest) (*PortfolioMarginMarginRepayRecordResponse, error) {
	params, err := portfolioMarginLoanRepayRecordParams(arg)
	if err != nil {
		return nil, err
	}
	var resp *PortfolioMarginMarginRepayRecordResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/margin/repayLoan", params, portfolioMarginGetMarginRepayRecordRate, &resp)
}

// GetPortfolioMarginNegativeBalanceInterestHistory returns the interest charged on negative balances, newest first;
// without a time range the last 7 days are returned, and a range cannot exceed 30 days
func (e *Exchange) GetPortfolioMarginNegativeBalanceInterestHistory(ctx context.Context, arg *PortfolioMarginNegativeBalanceInterestHistoryRequest) ([]PortfolioMarginNegativeBalanceInterestResponse, error) {
	params, err := portfolioMarginNegativeBalanceInterestHistoryParams(arg)
	if err != nil {
		return nil, err
	}
	var resp []PortfolioMarginNegativeBalanceInterestResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/portfolio/interest-history", params, portfolioMarginGetNegativeBalanceInterestHistoryRate, &resp)
}

func portfolioMarginNegativeBalanceInterestHistoryParams(arg *PortfolioMarginNegativeBalanceInterestHistoryRequest) (url.Values, error) {
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

// GetPortfolioMarginUSDTMarginedPositionInformation returns the UM positions of one symbol, or of every symbol when
// symbol is empty; one-way mode accounts only have BOTH positions and hedge mode accounts LONG and SHORT positions
func (e *Exchange) GetPortfolioMarginUSDTMarginedPositionInformation(ctx context.Context, symbol currency.Pair) ([]PortfolioMarginUSDTMarginedPositionInformationResponse, error) {
	params := url.Values{}
	if err := e.portfolioMarginSymbolParam(params, symbol, asset.USDTMarginedFutures); err != nil {
		return nil, err
	}
	var resp []PortfolioMarginUSDTMarginedPositionInformationResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/um/positionRisk", params, portfolioMarginGetUSDTMarginedPositionInformationRate, &resp)
}

// GetPortfolioMarginUserNegativeBalanceAutoExchangeRecord returns negative balance auto exchange records, newest first;
// the time range is required and cannot exceed three months
func (e *Exchange) GetPortfolioMarginUserNegativeBalanceAutoExchangeRecord(ctx context.Context, startTime, endTime time.Time) (*PortfolioMarginNegativeBalanceExchangeRecordResponse, error) {
	if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	var resp *PortfolioMarginNegativeBalanceExchangeRecordResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/portfolio/negative-balance-exchange-record", params, portfolioMarginNegativeBalanceExchangeRecordRate, &resp)
}

// GetPortfolioMarginUserRateLimits returns the user's portfolio margin order rate limits
func (e *Exchange) GetPortfolioMarginUserRateLimits(ctx context.Context) ([]PortfolioMarginRateLimitResponse, error) {
	var resp []PortfolioMarginRateLimitResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/rateLimit/order", nil, portfolioMarginDefaultRate, &resp)
}

// PortfolioMarginRepayFuturesNegativeBalance repays futures negative balances
func (e *Exchange) PortfolioMarginRepayFuturesNegativeBalance(ctx context.Context) (*PortfolioMarginMessageResponse, error) {
	var resp *PortfolioMarginMessageResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodPost, "/papi/v1/repay-futures-negative-balance", nil, portfolioMarginRepayFuturesNegativeBalanceRate, &resp)
}

// GetPortfolioMarginUSDTMarginedNotionalAndLeverageBrackets returns the notional and leverage brackets of one UM
// symbol, or of every UM symbol when symbol is empty
func (e *Exchange) GetPortfolioMarginUSDTMarginedNotionalAndLeverageBrackets(ctx context.Context, symbol currency.Pair) ([]PortfolioMarginUSDTMarginedNotionalAndLeverageResponse, error) {
	params := url.Values{}
	if err := e.portfolioMarginSymbolParam(params, symbol, asset.USDTMarginedFutures); err != nil {
		return nil, err
	}
	var resp []PortfolioMarginUSDTMarginedNotionalAndLeverageResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/um/leverageBracket", params, portfolioMarginDefaultRate, &resp)
}

// CancelAllPortfolioMarginCoinMarginedOpenConditionalOrders cancels every open CM conditional order of a symbol
func (e *Exchange) CancelAllPortfolioMarginCoinMarginedOpenConditionalOrders(ctx context.Context, symbol currency.Pair) (*SuccessResponse, error) {
	return e.portfolioMarginCancelAllOpenOrders(ctx, "/papi/v1/cm/conditional/allOpenOrders", symbol, asset.CoinMarginedFutures)
}

func (e *Exchange) portfolioMarginCancelAllOpenOrders(ctx context.Context, path string, symbol currency.Pair, a asset.Item) (*SuccessResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	if err := e.portfolioMarginSymbolParam(params, symbol, a); err != nil {
		return nil, err
	}
	var resp *SuccessResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodDelete, path, params, portfolioMarginDefaultRate, &resp)
}

// CancelAllPortfolioMarginCoinMarginedOrders cancels every open CM LIMIT order of a symbol
func (e *Exchange) CancelAllPortfolioMarginCoinMarginedOrders(ctx context.Context, symbol currency.Pair) (*SuccessResponse, error) {
	return e.portfolioMarginCancelAllOpenOrders(ctx, "/papi/v1/cm/allOpenOrders", symbol, asset.CoinMarginedFutures)
}

// CancelAllPortfolioMarginUSDTMarginedOrders cancels every open UM LIMIT order of a symbol
func (e *Exchange) CancelAllPortfolioMarginUSDTMarginedOrders(ctx context.Context, symbol currency.Pair) (*SuccessResponse, error) {
	return e.portfolioMarginCancelAllOpenOrders(ctx, "/papi/v1/um/allOpenOrders", symbol, asset.USDTMarginedFutures)
}

// NewPortfolioMarginCoinMarginedConditionalOrder places a CM conditional order
func (e *Exchange) NewPortfolioMarginCoinMarginedConditionalOrder(ctx context.Context, arg *PortfolioMarginCoinMarginedConditionalOrderRequest) (*NewPortfolioMarginCoinMarginedConditionalOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	side, err := portfolioMarginOrderSide(arg.Side)
	if err != nil {
		return nil, err
	}
	if arg.StrategyType == "" {
		return nil, errStrategyTypeRequired
	}
	params := url.Values{}
	if err := e.portfolioMarginSymbolParam(params, arg.Symbol, asset.CoinMarginedFutures); err != nil {
		return nil, err
	}
	params.Set("side", side)
	if arg.PositionSide != "" {
		params.Set("positionSide", string(arg.PositionSide))
	}
	params.Set("strategyType", string(arg.StrategyType))
	if arg.TimeInForce != "" {
		params.Set("timeInForce", string(arg.TimeInForce))
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
		params.Set("workingType", string(arg.WorkingType))
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
	var resp *NewPortfolioMarginCoinMarginedConditionalOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodPost, "/papi/v1/cm/conditional/order", params, portfolioMarginOrderRate, &resp)
}

// CancelPortfolioMarginCoinMarginedConditionalOrder cancels a CM conditional order identified by its strategy ID or
// client strategy ID
func (e *Exchange) CancelPortfolioMarginCoinMarginedConditionalOrder(ctx context.Context, symbol currency.Pair, strategyID uint64, newClientStrategyID string) (*CancelPortfolioMarginCoinMarginedConditionalOrderResponse, error) {
	params, err := e.portfolioMarginConditionalOrderParams(symbol, strategyID, newClientStrategyID)
	if err != nil {
		return nil, err
	}
	var resp *CancelPortfolioMarginCoinMarginedConditionalOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodDelete, "/papi/v1/cm/conditional/order", params, portfolioMarginDefaultRate, &resp)
}

func (e *Exchange) portfolioMarginConditionalOrderParams(symbol currency.Pair, strategyID uint64, newClientStrategyID string) (url.Values, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if strategyID == 0 && newClientStrategyID == "" {
		return nil, fmt.Errorf("%w: strategyId or newClientStrategyId is required", order.ErrOrderIDNotSet)
	}
	params := url.Values{}
	if err := e.portfolioMarginSymbolParam(params, symbol, asset.CoinMarginedFutures); err != nil {
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

// GetPortfolioMarginCoinMarginedOrder returns a CM order identified by its order ID or client order ID; cancelled or
// expired orders without fills are not found three days after creation
func (e *Exchange) GetPortfolioMarginCoinMarginedOrder(ctx context.Context, symbol currency.Pair, orderID uint64, origClientOrderID string) (*PortfolioMarginCoinMarginedOrderDetailResponse, error) {
	params, err := e.portfolioMarginOrderParams(symbol, asset.CoinMarginedFutures, orderID, origClientOrderID)
	if err != nil {
		return nil, err
	}
	var resp *PortfolioMarginCoinMarginedOrderDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/cm/order", params, portfolioMarginDefaultRate, &resp)
}

// portfolioMarginOrderParams builds the parameters identifying a single order by its order ID or original client order
// ID
func (e *Exchange) portfolioMarginOrderParams(symbol currency.Pair, a asset.Item, orderID uint64, origClientOrderID string) (url.Values, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if orderID == 0 && origClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	if err := e.portfolioMarginSymbolParam(params, symbol, a); err != nil {
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

// NewPortfolioMarginCoinMarginedOrder places a CM order
func (e *Exchange) NewPortfolioMarginCoinMarginedOrder(ctx context.Context, arg *PortfolioMarginCoinMarginedOrderRequest) (*PortfolioMarginCoinMarginedOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params, err := e.portfolioMarginNewFuturesOrderParams(arg.Symbol, asset.CoinMarginedFutures, arg.Side, string(arg.OrderType), arg.Price, string(arg.PriceMatch))
	if err != nil {
		return nil, err
	}
	if arg.PositionSide != "" {
		params.Set("positionSide", string(arg.PositionSide))
	}
	if arg.TimeInForce != "" {
		params.Set("timeInForce", string(arg.TimeInForce))
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
	if arg.NewOrderResponseType != "" {
		params.Set("newOrderRespType", string(arg.NewOrderResponseType))
	}
	var resp *PortfolioMarginCoinMarginedOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodPost, "/papi/v1/cm/order", params, portfolioMarginOrderRate, &resp)
}

// portfolioMarginNewFuturesOrderParams validates and sets the parameters shared by new UM and CM orders
func (e *Exchange) portfolioMarginNewFuturesOrderParams(symbol currency.Pair, a asset.Item, orderSide order.Side, orderType string, price float64, priceMatch string) (url.Values, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	side, err := portfolioMarginOrderSide(orderSide)
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
	if err := e.portfolioMarginSymbolParam(params, symbol, a); err != nil {
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

// CancelPortfolioMarginCoinMarginedOrder cancels an open CM LIMIT order identified by its order ID or client order ID
func (e *Exchange) CancelPortfolioMarginCoinMarginedOrder(ctx context.Context, symbol currency.Pair, orderID uint64, origClientOrderID string) (*PortfolioMarginCoinMarginedOrderResponse, error) {
	params, err := e.portfolioMarginOrderParams(symbol, asset.CoinMarginedFutures, orderID, origClientOrderID)
	if err != nil {
		return nil, err
	}
	var resp *PortfolioMarginCoinMarginedOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodDelete, "/papi/v1/cm/order", params, portfolioMarginDefaultRate, &resp)
}

// CancelAllPortfolioMarginMarginOpenOrdersBySymbol cancels every open margin order and OCO order list of a symbol
func (e *Exchange) CancelAllPortfolioMarginMarginOpenOrdersBySymbol(ctx context.Context, symbol currency.Pair) ([]PortfolioMarginCancelledMarginOrderResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	if err := e.portfolioMarginSymbolParam(params, symbol, asset.Margin); err != nil {
		return nil, err
	}
	var resp []PortfolioMarginCancelledMarginOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodDelete, "/papi/v1/margin/allOpenOrders", params, portfolioMarginCancelMarginAccountOpenOrdersOnSymbolRate, &resp)
}

// GetPortfolioMarginMarginAccountOneCancelsTheOtherOrder returns a margin OCO order list identified by its order list
// ID or original client order ID
func (e *Exchange) GetPortfolioMarginMarginAccountOneCancelsTheOtherOrder(ctx context.Context, orderListID uint64, origClientOrderID string) (*PortfolioMarginMarginOneCancelsTheOtherOrderResponse, error) {
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
	var resp *PortfolioMarginMarginOneCancelsTheOtherOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/margin/orderList", params, portfolioMarginGetMarginAccountOneCancelsTheOtherRate, &resp)
}

// CancelPortfolioMarginMarginAccountOneCancelsTheOtherOrders cancels a margin OCO order list; cancelling either leg
// cancels the whole list
func (e *Exchange) CancelPortfolioMarginMarginAccountOneCancelsTheOtherOrders(ctx context.Context, arg *PortfolioMarginCancelMarginOneCancelsTheOtherOrderRequest) (*PortfolioMarginCancelMarginOneCancelsTheOtherOrderResponse, error) {
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
	if err := e.portfolioMarginSymbolParam(params, arg.Symbol, asset.Margin); err != nil {
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
	var resp *PortfolioMarginCancelMarginOneCancelsTheOtherOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodDelete, "/papi/v1/margin/orderList", params, portfolioMarginCancelMarginAccountOneCancelsTheOtherRate, &resp)
}

// GetPortfolioMarginMarginAccountOrder returns a margin order identified by its order ID or client order ID
func (e *Exchange) GetPortfolioMarginMarginAccountOrder(ctx context.Context, symbol currency.Pair, orderID uint64, origClientOrderID string) (*PortfolioMarginMarginOrderResponse, error) {
	params, err := e.portfolioMarginOrderParams(symbol, asset.Margin, orderID, origClientOrderID)
	if err != nil {
		return nil, err
	}
	var resp *PortfolioMarginMarginOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/margin/order", params, portfolioMarginGetMarginAccountOrderRate, &resp)
}

// NewPortfolioMarginMarginOrder places a margin order
func (e *Exchange) NewPortfolioMarginMarginOrder(ctx context.Context, arg *PortfolioMarginMarginOrderRequest) (*PortfolioMarginNewMarginOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	side, err := portfolioMarginOrderSide(arg.Side)
	if err != nil {
		return nil, err
	}
	if arg.OrderType == "" {
		return nil, order.ErrTypeIsInvalid
	}
	params := url.Values{}
	if err := e.portfolioMarginSymbolParam(params, arg.Symbol, asset.Margin); err != nil {
		return nil, err
	}
	params.Set("side", side)
	params.Set("type", string(arg.OrderType))
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
	if arg.NewOrderResponseType != "" {
		params.Set("newOrderRespType", string(arg.NewOrderResponseType))
	}
	if arg.IcebergQuantity > 0 {
		params.Set("icebergQty", strconv.FormatFloat(arg.IcebergQuantity, 'f', -1, 64))
	}
	if arg.SideEffectType != "" {
		params.Set("sideEffectType", string(arg.SideEffectType))
	}
	if arg.TimeInForce != "" {
		params.Set("timeInForce", string(arg.TimeInForce))
	}
	if arg.SelfTradePreventionMode != "" {
		params.Set("selfTradePreventionMode", string(arg.SelfTradePreventionMode))
	}
	if arg.AutoRepayAtCancel != nil {
		params.Set("autoRepayAtCancel", strconv.FormatBool(*arg.AutoRepayAtCancel))
	}
	var resp *PortfolioMarginNewMarginOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodPost, "/papi/v1/margin/order", params, portfolioMarginOrderRate, &resp)
}

// PortfolioMarginCancelMarginAccountOrder cancels a margin order identified by its order ID or client order ID
func (e *Exchange) PortfolioMarginCancelMarginAccountOrder(ctx context.Context, arg *PortfolioMarginCancelMarginOrderRequest) (*PortfolioMarginCancelMarginOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params, err := e.portfolioMarginOrderParams(arg.Symbol, asset.Margin, arg.OrderID, arg.OriginalClientOrderID)
	if err != nil {
		return nil, err
	}
	if arg.NewClientOrderID != "" {
		params.Set("newClientOrderId", arg.NewClientOrderID)
	}
	var resp *PortfolioMarginCancelMarginOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodDelete, "/papi/v1/margin/order", params, portfolioMarginCancelMarginAccountOrderRate, &resp)
}

// GetPortfolioMarginUSDTMarginedOrder returns a UM order identified by its order ID or client order ID; cancelled or
// expired orders without fills are not found three days after creation
func (e *Exchange) GetPortfolioMarginUSDTMarginedOrder(ctx context.Context, symbol currency.Pair, orderID uint64, origClientOrderID string) (*PortfolioMarginUSDTMarginedOrderDetailResponse, error) {
	params, err := e.portfolioMarginOrderParams(symbol, asset.USDTMarginedFutures, orderID, origClientOrderID)
	if err != nil {
		return nil, err
	}
	var resp *PortfolioMarginUSDTMarginedOrderDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/um/order", params, portfolioMarginDefaultRate, &resp)
}

// NewPortfolioMarginUSDTMarginedOrder places a UM order
func (e *Exchange) NewPortfolioMarginUSDTMarginedOrder(ctx context.Context, arg *PortfolioMarginUSDTMarginedOrderRequest) (*PortfolioMarginUSDTMarginedOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params, err := e.portfolioMarginNewFuturesOrderParams(arg.Symbol, asset.USDTMarginedFutures, arg.Side, string(arg.OrderType), arg.Price, string(arg.PriceMatch))
	if err != nil {
		return nil, err
	}
	if arg.TimeInForce == "GTD" && arg.GoodTillDate.IsZero() {
		return nil, errGoodTillDateRequired
	}
	if arg.PositionSide != "" {
		params.Set("positionSide", string(arg.PositionSide))
	}
	if arg.TimeInForce != "" {
		params.Set("timeInForce", string(arg.TimeInForce))
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
	if arg.NewOrderResponseType != "" {
		params.Set("newOrderRespType", string(arg.NewOrderResponseType))
	}
	if arg.SelfTradePreventionMode != "" {
		params.Set("selfTradePreventionMode", string(arg.SelfTradePreventionMode))
	}
	if !arg.GoodTillDate.IsZero() {
		params.Set("goodTillDate", strconv.FormatInt(arg.GoodTillDate.UnixMilli(), 10))
	}
	var resp *PortfolioMarginUSDTMarginedOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodPost, "/papi/v1/um/order", params, portfolioMarginOrderRate, &resp)
}

// CancelPortfolioMarginUSDTMarginedOrder cancels an open UM LIMIT order identified by its order ID or client order ID
func (e *Exchange) CancelPortfolioMarginUSDTMarginedOrder(ctx context.Context, symbol currency.Pair, orderID uint64, origClientOrderID string) (*PortfolioMarginUSDTMarginedOrderResponse, error) {
	params, err := e.portfolioMarginOrderParams(symbol, asset.USDTMarginedFutures, orderID, origClientOrderID)
	if err != nil {
		return nil, err
	}
	var resp *PortfolioMarginUSDTMarginedOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodDelete, "/papi/v1/um/order", params, portfolioMarginDefaultRate, &resp)
}

// GetPortfolioMarginCoinMarginedAccountTradeList returns CM account trades of a symbol, or of every symbol of a pair;
// without a time range the last 24 hours are returned, and a range cannot exceed 24 hours
func (e *Exchange) GetPortfolioMarginCoinMarginedAccountTradeList(ctx context.Context, arg *PortfolioMarginCoinMarginedAccountTradeListRequest) ([]PortfolioMarginCoinMarginedAccountTradeResponse, error) {
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
	f := portfolioMarginGetCoinMarginedAccountTradeListWithPairRate
	if !arg.Symbol.IsEmpty() {
		if err := e.portfolioMarginSymbolParam(params, arg.Symbol, asset.CoinMarginedFutures); err != nil {
			return nil, err
		}
		f = portfolioMarginGetCoinMarginedAccountTradeListWithSymbolRate
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
	var resp []PortfolioMarginCoinMarginedAccountTradeResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/cm/userTrades", params, f, &resp)
}

// GetPortfolioMarginCoinMarginedPositionAutoDeleveragingQuantileEstimation returns the auto-deleveraging quantiles of
// one CM symbol's positions, or of every CM symbol when symbol is empty; values update every 30 seconds
func (e *Exchange) GetPortfolioMarginCoinMarginedPositionAutoDeleveragingQuantileEstimation(ctx context.Context, symbol currency.Pair) ([]PortfolioMarginCoinMarginedAutoDeleveragingQuantileEstimationResponse, error) {
	params := url.Values{}
	if err := e.portfolioMarginSymbolParam(params, symbol, asset.CoinMarginedFutures); err != nil {
		return nil, err
	}
	var resp []PortfolioMarginCoinMarginedAutoDeleveragingQuantileEstimationResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/cm/adlQuantile", params, portfolioMarginGetCoinMarginedPositionAutoDeleveragingQuantileEstimationRate, &resp)
}

// PortfolioMarginMarginAccountBorrow borrows an asset into the margin account
func (e *Exchange) PortfolioMarginMarginAccountBorrow(ctx context.Context, ccy currency.Code, amount float64) (*PortfolioMarginTransactionResponse, error) {
	return e.portfolioMarginMarginAccountBorrowRepay(ctx, "/papi/v1/marginLoan", ccy, amount)
}

func (e *Exchange) portfolioMarginMarginAccountBorrowRepay(ctx context.Context, path string, ccy currency.Code, amount float64) (*PortfolioMarginTransactionResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("asset", ccy.Upper().String())
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	var resp *PortfolioMarginTransactionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodPost, path, params, portfolioMarginMarginAccountLoanAndRepayRate, &resp)
}

// PortfolioMarginMarginAccountNewOneCancelsTheOtherOrder places a margin OCO order list; both legs share the quantity
// and the list counts as two orders against the order rate limit
func (e *Exchange) PortfolioMarginMarginAccountNewOneCancelsTheOtherOrder(ctx context.Context, arg *PortfolioMarginMarginOneCancelsTheOtherOrderRequest) (*PortfolioMarginNewMarginOneCancelsTheOtherOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	side, err := portfolioMarginOrderSide(arg.Side)
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
	if err := e.portfolioMarginSymbolParam(params, arg.Symbol, asset.Margin); err != nil {
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
		params.Set("stopLimitTimeInForce", string(arg.StopLimitTimeInForce))
	}
	if arg.NewOrderResponseType != "" {
		params.Set("newOrderRespType", string(arg.NewOrderResponseType))
	}
	if arg.SideEffectType != "" {
		params.Set("sideEffectType", string(arg.SideEffectType))
	}
	var resp *PortfolioMarginNewMarginOneCancelsTheOtherOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodPost, "/papi/v1/margin/order/oco", params, portfolioMarginOneCancelsTheOtherOrderRate, &resp)
}

// PortfolioMarginMarginAccountRepay repays a margin loan
func (e *Exchange) PortfolioMarginMarginAccountRepay(ctx context.Context, ccy currency.Code, amount float64) (*PortfolioMarginTransactionResponse, error) {
	return e.portfolioMarginMarginAccountBorrowRepay(ctx, "/papi/v1/repayLoan", ccy, amount)
}

// GetPortfolioMarginMarginAccountTradeList returns margin account trades of a symbol, from FromID onwards when it is
// set; a time range must be shorter than 24 hours
func (e *Exchange) GetPortfolioMarginMarginAccountTradeList(ctx context.Context, arg *PortfolioMarginMarginTradeListRequest) ([]PortfolioMarginMarginTradeResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	if err := e.portfolioMarginSymbolParam(params, arg.Symbol, asset.Margin); err != nil {
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
	var resp []PortfolioMarginMarginTradeResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/margin/myTrades", params, portfolioMarginGetMarginAccountTradeListRate, &resp)
}

// GetAllPortfolioMarginCoinMarginedConditionalOrders returns CM conditional orders of one symbol, or of every symbol
// when the symbol is empty; a time range cannot exceed 7 days, which is also the default
func (e *Exchange) GetAllPortfolioMarginCoinMarginedConditionalOrders(ctx context.Context, arg *PortfolioMarginCoinMarginedConditionalOrdersRequest) ([]PortfolioMarginCoinMarginedConditionalOrderRecordResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params := url.Values{}
	f := portfolioMarginAllCoinMarginedConditionalOrderWithoutSymbolRate
	if !arg.Symbol.IsEmpty() {
		if err := e.portfolioMarginSymbolParam(params, arg.Symbol, asset.CoinMarginedFutures); err != nil {
			return nil, err
		}
		f = portfolioMarginDefaultRate
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
	var resp []PortfolioMarginCoinMarginedConditionalOrderRecordResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/cm/conditional/allOrders", params, f, &resp)
}

// GetAllPortfolioMarginCoinMarginedOrders returns active, cancelled and filled CM orders of a symbol or pair; cancelled
// or expired orders without fills are not found three days after creation
func (e *Exchange) GetAllPortfolioMarginCoinMarginedOrders(ctx context.Context, arg *PortfolioMarginCoinMarginedOrdersRequest) ([]PortfolioMarginCoinMarginedOrderDetailResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Symbol.IsEmpty() && arg.Pair.IsEmpty() {
		return nil, fmt.Errorf("%w: symbol or pair is required", currency.ErrCurrencyPairEmpty)
	}
	params := url.Values{}
	f := portfolioMarginAllCoinMarginedOrderWithPairRate
	if !arg.Symbol.IsEmpty() {
		if err := e.portfolioMarginSymbolParam(params, arg.Symbol, asset.CoinMarginedFutures); err != nil {
			return nil, err
		}
		f = portfolioMarginAllCoinMarginedOrderWithSymbolRate
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
	var resp []PortfolioMarginCoinMarginedOrderDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/cm/allOrders", params, f, &resp)
}

// GetAllPortfolioMarginCoinMarginedOpenConditionalOrders returns the open CM conditional orders of one symbol, or of
// every symbol when symbol is empty
func (e *Exchange) GetAllPortfolioMarginCoinMarginedOpenConditionalOrders(ctx context.Context, symbol currency.Pair) ([]PortfolioMarginCoinMarginedConditionalOrderResponse, error) {
	params := url.Values{}
	f := portfolioMarginAllCoinMarginedOpenConditionalOrdersWithoutSymbolRate
	if !symbol.IsEmpty() {
		if err := e.portfolioMarginSymbolParam(params, symbol, asset.CoinMarginedFutures); err != nil {
			return nil, err
		}
		f = portfolioMarginDefaultRate
	}
	var resp []PortfolioMarginCoinMarginedConditionalOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/cm/conditional/openOrders", params, f, &resp)
}

// GetAllPortfolioMarginCoinMarginedOpenOrders returns the open CM orders of a symbol or of a pair (such as BTCUSD, the
// base of a COIN-M contract pair), or of every symbol when both are empty
func (e *Exchange) GetAllPortfolioMarginCoinMarginedOpenOrders(ctx context.Context, symbol currency.Pair, pair currency.Code) ([]PortfolioMarginCoinMarginedOrderDetailResponse, error) {
	params := url.Values{}
	f := portfolioMarginRetrieveAllCoinMarginedOpenOrdersForAllSymbolRate
	if !symbol.IsEmpty() {
		if err := e.portfolioMarginSymbolParam(params, symbol, asset.CoinMarginedFutures); err != nil {
			return nil, err
		}
		f = portfolioMarginDefaultRate
	}
	if !pair.IsEmpty() {
		params.Set("pair", pair.Upper().String())
	}
	var resp []PortfolioMarginCoinMarginedOrderDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/cm/openOrders", params, f, &resp)
}

// GetAllPortfolioMarginUSDTMarginedOpenOrders returns the open UM orders of one symbol, or of every symbol when symbol
// is empty
func (e *Exchange) GetAllPortfolioMarginUSDTMarginedOpenOrders(ctx context.Context, symbol currency.Pair) ([]PortfolioMarginUSDTMarginedOrderDetailResponse, error) {
	params := url.Values{}
	f := portfolioMarginRetrieveAllUSDTMarginedOpenOrdersForAllSymbolRate
	if !symbol.IsEmpty() {
		if err := e.portfolioMarginSymbolParam(params, symbol, asset.USDTMarginedFutures); err != nil {
			return nil, err
		}
		f = portfolioMarginDefaultRate
	}
	var resp []PortfolioMarginUSDTMarginedOrderDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/um/openOrders", params, f, &resp)
}

// GetAllPortfolioMarginMarginAccountOrders returns margin orders of a symbol, from OrderID onwards when it is set
func (e *Exchange) GetAllPortfolioMarginMarginAccountOrders(ctx context.Context, arg *PortfolioMarginMarginOrdersRequest) ([]PortfolioMarginMarginOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params, err := e.portfolioMarginOrdersParams(arg.Symbol, asset.Margin, arg.OrderID, arg.StartTime, arg.EndTime, arg.Limit)
	if err != nil {
		return nil, err
	}
	var resp []PortfolioMarginMarginOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/margin/allOrders", params, portfolioMarginAllMarginAccountOrdersRate, &resp)
}

// portfolioMarginOrdersParams builds the parameters of the order history queries that require a symbol
func (e *Exchange) portfolioMarginOrdersParams(symbol currency.Pair, a asset.Item, orderID uint64, startTime, endTime time.Time, limit uint64) (url.Values, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	if err := e.portfolioMarginSymbolParam(params, symbol, a); err != nil {
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

// GetAllPortfolioMarginUSDTMarginedOrders returns active, cancelled and filled UM orders of a symbol; a time range
// cannot exceed 7 days, which is also the default
func (e *Exchange) GetAllPortfolioMarginUSDTMarginedOrders(ctx context.Context, arg *PortfolioMarginUSDTMarginedOrdersRequest) ([]PortfolioMarginUSDTMarginedOrderDetailResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params, err := e.portfolioMarginOrdersParams(arg.Symbol, asset.USDTMarginedFutures, arg.OrderID, arg.StartTime, arg.EndTime, arg.Limit)
	if err != nil {
		return nil, err
	}
	var resp []PortfolioMarginUSDTMarginedOrderDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/um/allOrders", params, portfolioMarginGetAllUSDTMarginedOrdersRate, &resp)
}

// GetPortfolioMarginCoinMarginedConditionalOrderHistory returns a triggered, cancelled or expired CM conditional order
// identified by its strategy ID or client strategy ID; NEW orders are not found
func (e *Exchange) GetPortfolioMarginCoinMarginedConditionalOrderHistory(ctx context.Context, symbol currency.Pair, strategyID uint64, newClientStrategyID string) (*PortfolioMarginCoinMarginedConditionalOrderHistoryResponse, error) {
	params, err := e.portfolioMarginConditionalOrderParams(symbol, strategyID, newClientStrategyID)
	if err != nil {
		return nil, err
	}
	var resp *PortfolioMarginCoinMarginedConditionalOrderHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/cm/conditional/orderHistory", params, portfolioMarginDefaultRate, &resp)
}

// GetOpenPortfolioMarginCoinMarginedConditionalOrder returns an open CM conditional order identified by its strategy ID
// or client strategy ID
func (e *Exchange) GetOpenPortfolioMarginCoinMarginedConditionalOrder(ctx context.Context, symbol currency.Pair, strategyID uint64, newClientStrategyID string) (*PortfolioMarginCoinMarginedConditionalOrderResponse, error) {
	params, err := e.portfolioMarginConditionalOrderParams(symbol, strategyID, newClientStrategyID)
	if err != nil {
		return nil, err
	}
	var resp *PortfolioMarginCoinMarginedConditionalOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/cm/conditional/openOrder", params, portfolioMarginDefaultRate, &resp)
}

// GetPortfolioMarginCoinMarginedOpenOrder returns an open CM order identified by its order ID or client order ID;
// unlike its UM counterpart the endpoint answers with an array
func (e *Exchange) GetPortfolioMarginCoinMarginedOpenOrder(ctx context.Context, symbol currency.Pair, orderID uint64, origClientOrderID string) ([]PortfolioMarginCoinMarginedOrderDetailResponse, error) {
	params, err := e.portfolioMarginOrderParams(symbol, asset.CoinMarginedFutures, orderID, origClientOrderID)
	if err != nil {
		return nil, err
	}
	var resp []PortfolioMarginCoinMarginedOrderDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/cm/openOrder", params, portfolioMarginDefaultRate, &resp)
}

// GetPortfolioMarginCurrentMarginOpenOrders returns the open margin orders of one symbol, or of every symbol when
// symbol is empty, which costs a request per trading symbol against the rate limiter
func (e *Exchange) GetPortfolioMarginCurrentMarginOpenOrders(ctx context.Context, symbol currency.Pair) ([]PortfolioMarginMarginOrderResponse, error) {
	params := url.Values{}
	if err := e.portfolioMarginSymbolParam(params, symbol, asset.Margin); err != nil {
		return nil, err
	}
	var resp []PortfolioMarginMarginOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/margin/openOrders", params, portfolioMarginCurrentMarginOpenOrderRate, &resp)
}

// GetPortfolioMarginUSDTMarginedOpenOrder returns an open UM order identified by its order ID or client order ID
func (e *Exchange) GetPortfolioMarginUSDTMarginedOpenOrder(ctx context.Context, symbol currency.Pair, orderID uint64, origClientOrderID string) (*PortfolioMarginUSDTMarginedOrderDetailResponse, error) {
	params, err := e.portfolioMarginOrderParams(symbol, asset.USDTMarginedFutures, orderID, origClientOrderID)
	if err != nil {
		return nil, err
	}
	var resp *PortfolioMarginUSDTMarginedOrderDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/um/openOrder", params, portfolioMarginDefaultRate, &resp)
}

// GetPortfolioMarginMarginAccountAllOneCancelsTheOtherOrders returns margin OCO order lists, from FromID onwards when
// it is set
func (e *Exchange) GetPortfolioMarginMarginAccountAllOneCancelsTheOtherOrders(ctx context.Context, arg *PortfolioMarginMarginAllOneCancelsTheOtherOrdersRequest) ([]PortfolioMarginMarginOneCancelsTheOtherOrderResponse, error) {
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
	var resp []PortfolioMarginMarginOneCancelsTheOtherOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/margin/allOrderList", params, portfolioMarginGetMarginAccountsAllOneCancelsTheOtherOrdersRate, &resp)
}

// GetPortfolioMarginMarginAccountsOpenOneCancelsTheOtherOrders returns the open margin OCO order lists
func (e *Exchange) GetPortfolioMarginMarginAccountsOpenOneCancelsTheOtherOrders(ctx context.Context) ([]PortfolioMarginMarginOneCancelsTheOtherOrderResponse, error) {
	var resp []PortfolioMarginMarginOneCancelsTheOtherOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/margin/openOrderList", nil, portfolioMarginGetMarginAccountsOpenOneCancelsTheOtherOrdersRate, &resp)
}

// GetUsersPortfolioMarginCoinMarginedForceOrders returns the user's CM liquidation and auto-deleveraging orders of the
// last 90 days
func (e *Exchange) GetUsersPortfolioMarginCoinMarginedForceOrders(ctx context.Context, arg *PortfolioMarginForceOrdersRequest) ([]PortfolioMarginCoinMarginedForceOrderResponse, error) {
	params, f, err := e.portfolioMarginForceOrdersParams(arg, asset.CoinMarginedFutures, portfolioMarginGetUserCoinMarginedForceOrdersWithSymbolRate, portfolioMarginGetUserCoinMarginedForceOrdersWithoutSymbolRate)
	if err != nil {
		return nil, err
	}
	var resp []PortfolioMarginCoinMarginedForceOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/cm/forceOrders", params, f, &resp)
}

// portfolioMarginForceOrdersParams builds the parameters of a UM or CM force orders query and selects its weight
func (e *Exchange) portfolioMarginForceOrdersParams(arg *PortfolioMarginForceOrdersRequest, a asset.Item, withSymbol, withoutSymbol request.EndpointLimit) (url.Values, request.EndpointLimit, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, 0, err
	}
	params := url.Values{}
	f := withoutSymbol
	if !arg.Symbol.IsEmpty() {
		if err := e.portfolioMarginSymbolParam(params, arg.Symbol, a); err != nil {
			return nil, 0, err
		}
		f = withSymbol
	}
	if arg.AutoCloseType != "" {
		params.Set("autoCloseType", string(arg.AutoCloseType))
	}
	if err := setTimeRangeParams(params, arg.StartTime, arg.EndTime); err != nil {
		return nil, 0, err
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	return params, f, nil
}

// GetUsersPortfolioMarginMarginForceOrders returns the user's margin liquidation orders; accounts liquidated through
// risk-based liquidation adjustment have no records here
func (e *Exchange) GetUsersPortfolioMarginMarginForceOrders(ctx context.Context, arg *PortfolioMarginMarginForceOrdersRequest) (*PortfolioMarginMarginForceOrderResponse, error) {
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
	var resp *PortfolioMarginMarginForceOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/margin/forceOrders", params, portfolioMarginDefaultRate, &resp)
}

// GetUsersPortfolioMarginUSDTMarginedForceOrders returns the user's UM liquidation and auto-deleveraging orders of the
// last 90 days
func (e *Exchange) GetUsersPortfolioMarginUSDTMarginedForceOrders(ctx context.Context, arg *PortfolioMarginForceOrdersRequest) ([]PortfolioMarginUSDTMarginedForceOrderResponse, error) {
	params, f, err := e.portfolioMarginForceOrdersParams(arg, asset.USDTMarginedFutures, portfolioMarginGetUserUSDTMarginedForceOrdersWithSymbolRate, portfolioMarginGetUserUSDTMarginedForceOrdersWithoutSymbolRate)
	if err != nil {
		return nil, err
	}
	var resp []PortfolioMarginUSDTMarginedForceOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/um/forceOrders", params, f, &resp)
}

// GetPortfolioMarginUSDTMarginedAccountTradeList returns UM account trades of a symbol; without a time range the last 7
// days are returned, and a range cannot exceed 7 days
func (e *Exchange) GetPortfolioMarginUSDTMarginedAccountTradeList(ctx context.Context, arg *PortfolioMarginUSDTMarginedAccountTradeListRequest) ([]PortfolioMarginUSDTMarginedAccountTradeResponse, error) {
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
	if err := e.portfolioMarginSymbolParam(params, arg.Symbol, asset.USDTMarginedFutures); err != nil {
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
	var resp []PortfolioMarginUSDTMarginedAccountTradeResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/um/userTrades", params, portfolioMarginGetUSDTMarginedAccountTradeListRate, &resp)
}

// GetPortfolioMarginUSDTMarginedPositionAutoDeleveragingQuantileEstimation returns the auto-deleveraging quantiles of
// one UM symbol's positions, or of every UM symbol when symbol is empty; values update every 30 seconds
func (e *Exchange) GetPortfolioMarginUSDTMarginedPositionAutoDeleveragingQuantileEstimation(ctx context.Context, symbol currency.Pair) ([]PortfolioMarginUSDTMarginedAutoDeleveragingQuantileEstimationResponse, error) {
	params := url.Values{}
	if err := e.portfolioMarginSymbolParam(params, symbol, asset.USDTMarginedFutures); err != nil {
		return nil, err
	}
	var resp []PortfolioMarginUSDTMarginedAutoDeleveragingQuantileEstimationResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/um/adlQuantile", params, portfolioMarginGetUSDTMarginedPositionAutoDeleveragingQuantileEstimationRate, &resp)
}

// NewPortfolioMarginUSDTMarginedAlgoOrder places a UM conditional order through the algo service, which replaced the
// retired UM conditional order endpoints on 2026-04-28
func (e *Exchange) NewPortfolioMarginUSDTMarginedAlgoOrder(ctx context.Context, arg *PortfolioMarginUSDTMarginedAlgoOrderRequest) (*PortfolioMarginUSDTMarginedAlgoOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params, err := e.portfolioMarginNewFuturesOrderParams(arg.Symbol, asset.USDTMarginedFutures, arg.Side, string(arg.OrderType), arg.Price, string(arg.PriceMatch))
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
	params.Set("algoType", string(arg.AlgoType))
	if arg.PositionSide != "" {
		params.Set("positionSide", string(arg.PositionSide))
	}
	if arg.TimeInForce != "" {
		params.Set("timeInForce", string(arg.TimeInForce))
	}
	if arg.Quantity > 0 {
		params.Set("quantity", strconv.FormatFloat(arg.Quantity, 'f', -1, 64))
	}
	if arg.TriggerPrice > 0 {
		params.Set("triggerPrice", strconv.FormatFloat(arg.TriggerPrice, 'f', -1, 64))
	}
	if arg.WorkingType != "" {
		params.Set("workingType", string(arg.WorkingType))
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
	if arg.NewOrderResponseType != "" {
		params.Set("newOrderRespType", string(arg.NewOrderResponseType))
	}
	if arg.SelfTradePreventionMode != "" {
		params.Set("selfTradePreventionMode", string(arg.SelfTradePreventionMode))
	}
	if !arg.GoodTillDate.IsZero() {
		params.Set("goodTillDate", strconv.FormatInt(arg.GoodTillDate.UnixMilli(), 10))
	}
	var resp *PortfolioMarginUSDTMarginedAlgoOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodPost, "/papi/v1/um/algo/order", params, portfolioMarginOrderRate, &resp)
}

// CancelPortfolioMarginUSDTMarginedAlgoOrder cancels an open UM algo order identified by its algo ID or client algo ID
func (e *Exchange) CancelPortfolioMarginUSDTMarginedAlgoOrder(ctx context.Context, algoID uint64, clientAlgoID string) (*CancelPortfolioMarginUSDTMarginedAlgoOrderResponse, error) {
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
	var resp *CancelPortfolioMarginUSDTMarginedAlgoOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodDelete, "/papi/v1/um/algo/order", params, portfolioMarginDefaultRate, &resp)
}

// CancelAllPortfolioMarginUSDTMarginedAlgoOpenOrders cancels every open UM algo order of a symbol
func (e *Exchange) CancelAllPortfolioMarginUSDTMarginedAlgoOpenOrders(ctx context.Context, symbol currency.Pair) (*SuccessResponse, error) {
	return e.portfolioMarginCancelAllOpenOrders(ctx, "/papi/v1/um/algo/allOpenOrders", symbol, asset.USDTMarginedFutures)
}

// GetPortfolioMarginUSDTMarginedAlgoOrder returns a UM algo order identified by its algo ID or client algo ID. Binance
// drops cancelled or expired orders without fills after three days, and every order after 90 days
func (e *Exchange) GetPortfolioMarginUSDTMarginedAlgoOrder(ctx context.Context, algoID uint64, clientAlgoID string) (*PortfolioMarginUSDTMarginedAlgoOrderDetailResponse, error) {
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
	var resp *PortfolioMarginUSDTMarginedAlgoOrderDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/um/algo/algoOrder", params, portfolioMarginDefaultRate, &resp)
}

// GetPortfolioMarginUSDTMarginedOpenAlgoOrders returns the open UM algo orders of a symbol, or of every symbol when
// symbol is empty; algoType and algoID narrow them when set
func (e *Exchange) GetPortfolioMarginUSDTMarginedOpenAlgoOrders(ctx context.Context, symbol currency.Pair, algoType AlgoType, algoID uint64) ([]PortfolioMarginUSDTMarginedAlgoOrderResponse, error) {
	params := url.Values{}
	if err := e.portfolioMarginSymbolParam(params, symbol, asset.USDTMarginedFutures); err != nil {
		return nil, err
	}
	if algoType != "" {
		params.Set("algoType", string(algoType))
	}
	if algoID > 0 {
		params.Set("algoId", strconv.FormatUint(algoID, 10))
	}
	var resp []PortfolioMarginUSDTMarginedAlgoOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/um/algo/openAlgoOrders", params, portfolioMarginDefaultRate, &resp)
}

// GetPortfolioMarginUSDTMarginedAlgoOrderHistory returns a symbol's UM algo orders whatever their status, from AlgoID
// on when it is set
func (e *Exchange) GetPortfolioMarginUSDTMarginedAlgoOrderHistory(ctx context.Context, arg *PortfolioMarginUSDTMarginedAlgoOrderHistoryRequest) ([]PortfolioMarginUSDTMarginedAlgoOrderDetailResponse, error) {
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
	if err := e.portfolioMarginSymbolParam(params, arg.Symbol, asset.USDTMarginedFutures); err != nil {
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
	var resp []PortfolioMarginUSDTMarginedAlgoOrderDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/um/algo/allAlgoOrders", params, portfolioMarginGetUSDTMarginedAlgoOrderHistoryRate, &resp)
}
