package poloniex

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var (
	errIntervalRequired = errors.New("interval is required")
	errNilArgument      = errors.New("error: nil argument")
	errAddressRequired  = errors.New("address is required")
)

// Reference data endpoints.

// GetSymbolInformation all symbols and their tradeLimit info. priceScale is referring to the max number of decimals allowed for a given symbol.
func (p *Poloniex) GetSymbolInformation(ctx context.Context, symbol currency.Pair) ([]SymbolDetail, error) {
	var resp []SymbolDetail
	path := "/markets"
	if !symbol.IsEmpty() {
		path = fmt.Sprintf("%s/%s", path, symbol)
	}
	return resp, p.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
}

// GetCurrencyInformations retrieves list of currencies and theiir detailed information.
func (p *Poloniex) GetCurrencyInformations(ctx context.Context) ([]CurrencyDetail, error) {
	var resp []CurrencyDetail
	path := "/currencies"
	return resp, p.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
}

// GetCurrencyInformation retrieves currency and their detailed information.
func (p *Poloniex) GetCurrencyInformation(ctx context.Context, ccy currency.Code) (CurrencyDetail, error) {
	var resp CurrencyDetail
	path := "/currencies"
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	path = fmt.Sprintf("%s/%s", path, ccy.String())
	return resp, p.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
}

// GetV2CurrencyInformations retrieves list of currency details for V2 API.
func (p *Poloniex) GetV2CurrencyInformations(ctx context.Context) ([]CurrencyV2Information, error) {
	var resp []CurrencyV2Information
	return resp, p.SendHTTPRequest(ctx, exchange.RestSpot, "/v2/currencies", &resp)
}

// GetV2CurrencyInformations retrieves currency details for V2 API.
func (p *Poloniex) GetV2CurrencyInformation(ctx context.Context, ccy currency.Code) (*CurrencyV2Information, error) {
	var resp CurrencyV2Information
	path := "/v2/currencies"
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	path = fmt.Sprintf("%s/%s", path, ccy.String())
	return &resp, p.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
}

// GetSystemTimestamp retrieves current server time.
func (p *Poloniex) GetSystemTimestamp(ctx context.Context) (time.Time, error) {
	resp := &struct {
		ServerTime convert.ExchangeTime `json:"serverTime"`
	}{}
	return resp.ServerTime.Time(), p.SendHTTPRequest(ctx, exchange.RestSpot, "/timestamp", &resp)
}

// Marker Data endpoints.

// GetMarketPrices retrieves latest trade price for all symbols.
func (p *Poloniex) GetMarketPrices(ctx context.Context) ([]MarketPrice, error) {
	var resp []MarketPrice
	return resp, p.SendHTTPRequest(ctx, exchange.RestSpot, "/markets/price", &resp)
}

// GetMarketPrice retrieves latest trade price for all symbols.
func (p *Poloniex) GetMarketPrice(ctx context.Context, symbol currency.Pair) (*MarketPrice, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp MarketPrice
	return &resp, p.SendHTTPRequest(ctx, exchange.RestSpot, fmt.Sprintf("/markets/%s/price", symbol.String()), &resp)
}

// GetMarkPrices retrieves latest mark price for a single cross margin
func (p *Poloniex) GetMarkPrices(ctx context.Context) ([]MarkPrice, error) {
	var resp []MarkPrice
	return resp, p.SendHTTPRequest(ctx, exchange.RestSpot, "/markets/markPrice", &resp)
}

// GetMarkPrice retrieves latest mark price for all cross margin symbol.
func (p *Poloniex) GetMarkPrice(ctx context.Context, symbol currency.Pair) (*MarkPrice, error) {
	var resp MarkPrice
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	return &resp, p.SendHTTPRequest(ctx, exchange.RestSpot, fmt.Sprintf("/markets/%s/markPrice", symbol.String()), &resp)
}

// MarkPriceComponents retrieves components of the mark price for a given symbol.
func (p *Poloniex) MarkPriceComponents(ctx context.Context, symbol currency.Pair) (*MarkPriceComponent, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp MarkPriceComponent
	return &resp, p.SendHTTPRequest(ctx, exchange.RestSpot, fmt.Sprintf("/markets/%s/markPriceComponents", symbol.String()), &resp)
}

// GetOrderbook retrieves the order book for a given symbol. Scale and limit values are optional.
// For valid scale values, please refer to the scale values defined for each symbol .
// If scale is not supplied, then no grouping/aggregation will be applied.
func (p *Poloniex) GetOrderbook(ctx context.Context, symbol currency.Pair) (*OrderbookData, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var resp OrderbookData
	return &resp, p.SendHTTPRequest(ctx, exchange.RestSpot, fmt.Sprintf("/markets/%s/orderBook", symbol.String()), &resp)
}

// GetCandlesticks retrieves OHLC for a symbol at given timeframe (interval).
func (p *Poloniex) GetCandlesticks(ctx context.Context, symbol currency.Pair, interval kline.Interval, startTime, endTime time.Time, limit int64) ([]CandlestickData, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	intervalString, err := intervalToString(interval)
	if err != nil {
		return nil, err
	} else if intervalString == "" {
		return nil, errIntervalRequired
	}
	params := url.Values{}
	params.Set("interval", intervalString)
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	var resp []CandlestickArrayData
	err = p.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(fmt.Sprintf("/markets/%s/candles", symbol.String()), params), &resp)
	if err != nil {
		return nil, err
	}
	return processCandlestickData(resp)
}

// GetTrades returns a list of recent trades, request param limit is optional, its default value is 500, and max value is 1000.
func (p *Poloniex) GetTrades(ctx context.Context, symbol currency.Pair, limit int64) ([]Trade, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []Trade
	return resp, p.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(fmt.Sprintf("/markets/%s/trades", symbol.String()), params), &resp)
}

// GetTicker retrieve ticker in last 24 hours for all symbols.
func (p *Poloniex) GetTickers(ctx context.Context) ([]TickerData, error) {
	var resp []TickerData
	return resp, p.SendHTTPRequest(ctx, exchange.RestSpot, "/markets/ticker24h", &resp)
}

// GetTicker retrieve ticker in last 24 hours for provided symbols.
func (p *Poloniex) GetTicker(ctx context.Context, symbol currency.Pair) (*TickerData, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var resp TickerData
	return &resp, p.SendHTTPRequest(ctx, exchange.RestSpot, fmt.Sprintf("/markets/%s/ticker24h", symbol.String()), &resp)
}

// Margin endpoints.

// GetCollateralInfos retrieves collateral information for all currencies.
func (p *Poloniex) GetCollateralInfos(ctx context.Context) ([]CollateralInfo, error) {
	var resp []CollateralInfo
	return resp, p.SendHTTPRequest(ctx, exchange.RestSpot, "/markets/collateralInfo", &resp)
}

// GetCollateralInfo retrieves collateral information for all currencies.
func (p *Poloniex) GetCollateralInfo(ctx context.Context, ccy currency.Code) (*CollateralInfo, error) {
	var resp CollateralInfo
	return &resp, p.SendHTTPRequest(ctx, exchange.RestSpot, fmt.Sprintf("/markets/%s/collateralInfo", ccy.String()), &resp)
}

// GetBorrowRateInfo retrieves borrow rates information for all tiers and currencies.
func (p *Poloniex) GetBorrowRateInfo(ctx context.Context) ([]BorrowRateinfo, error) {
	var resp []BorrowRateinfo
	return resp, p.SendHTTPRequest(ctx, exchange.RestSpot, "/markets/borrowRatesInfo", &resp)
}

// ---------------- Authenticated endpoints ----------------------------------

// GetAccountInformation retrieves all accounts of a user.
func (p *Poloniex) GetAccountInformation(ctx context.Context) ([]AccountInformation, error) {
	var resp []AccountInformation
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/accounts", nil, nil, &resp)
}

// GetAllBalances get a list of all accounts of a user with each account’s id, type and balances (assets).
func (p *Poloniex) GetAllBalances(ctx context.Context, accountType string) ([]AccountBalance, error) {
	params := url.Values{}
	if accountType != "" {
		params.Set("accountType", accountType)
	}
	var resp []AccountBalance
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/accounts/balances", params, nil, &resp)
}

// GetAllBalance get an accounts of a user with each account’s id, type and balances (assets).
func (p *Poloniex) GetAllBalance(ctx context.Context, accountID, accountType string) ([]AccountBalance, error) {
	if accountID == "" {
		return nil, errAccountIDRequired
	}
	params := url.Values{}
	if accountType != "" {
		params.Set("accountType", accountType)
	}
	var resp []AccountBalance
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, fmt.Sprintf("/accounts/%s/balances", accountID), params, nil, &resp)
}

// GetAllAccountActivities retrieves a list of activities such as airdrop, rebates, staking, credit/debit adjustments, and other (historical adjustments).
// Type of activity: ALL: 200, AIRDROP: 201, COMMISSION_REBATE: 202, STAKING: 203, REFERAL_REBATE: 204, CREDIT_ADJUSTMENT: 104, DEBIT_ADJUSTMENT: 105, OTHER: 199
func (p *Poloniex) GetAllAccountActivities(ctx context.Context, startTime, endTime time.Time,
	activityType, limit, from int64, direction string, ccy currency.Code) ([]AccountActivity, error) {
	params := url.Values{}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if activityType != 0 {
		params.Set("activityType", strconv.FormatInt(activityType, 10))
	}
	if limit != 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if from != 0 {
		params.Set("from", strconv.FormatInt(from, 10))
	}
	if direction != "" {
		params.Set("direction", direction)
	}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	var resp []AccountActivity
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/accounts/activity", params, nil, &resp)
}

// AccountsTransfer transfer amount of currency from an account to another account for a user.
func (p *Poloniex) AccountsTransfer(ctx context.Context, arg *AccountTransferParams) (*AccountTransferResponse, error) {
	if arg == nil {
		return nil, errNilArgument
	}
	if arg.Ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Amount <= 0 {
		return nil, fmt.Errorf("%w, amount has to be greater than zero", order.ErrAmountIsInvalid)
	}
	if arg.FromAccount == "" {
		return nil, fmt.Errorf("%w, fromAccount=''", errAddressRequired)
	}
	if arg.ToAccount == "" {
		return nil, fmt.Errorf("%w, toAccount=''", errAddressRequired)
	}
	var resp AccountTransferResponse
	return &resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/accounts/transfer", nil, arg, &resp)
}

// GetAccountTransferRecords gets a list of transfer records of a user. Max interval for start and end time is 6 months. If no start/end time params are specified then records for last 7 days will be returned.
func (p *Poloniex) GetAccountTransferRecords(ctx context.Context, startTime, endTime time.Time, direction string, ccy currency.Code, from, limit int64) ([]AccountTransferRecord, error) {
	params := url.Values{}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	if direction != "" {
		params.Set("direction", direction)
	}
	if from != 0 {
		params.Set("from", strconv.FormatInt(from, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []AccountTransferRecord
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/accounts/transfer", params, nil, &resp)
}

// GetAccountTransferRecord gets a transfer records of a user.
func (p *Poloniex) GetAccountTransferRecord(ctx context.Context, accountID string) ([]AccountTransferRecord, error) {
	var resp []AccountTransferRecord
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, fmt.Sprintf("/accounts/transfer/%s", accountID), nil, nil, &resp)
}

// GetFeeInfo retrieves fee rate for an account
func (p *Poloniex) GetFeeInfo(ctx context.Context) (*FeeInfo, error) {
	var resp FeeInfo
	return &resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/feeinfo", nil, nil, &resp)
}

// GetInterestHistory get a list of interest collection records of a user.
// Max interval for start and end time is 90 days.
// If no start/end time params are specified then records for last 7 days will be returned.
func (p *Poloniex) GetInterestHistory(ctx context.Context, startTime, endTime time.Time, direction string, from, limit int64) ([]InterestHistory, error) {
	params := url.Values{}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if direction != "" {
		params.Set("direction", direction)
	}
	if from != 0 {
		params.Set("from", strconv.FormatInt(from, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []InterestHistory
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/accounts/interest/history", params, nil, &resp)
}

// ------------------------------------  Sub-Accounts endpoints ----------------------------

// GetSubAccountInformations get a list of all the accounts within an Account Group for a user.
func (p *Poloniex) GetSubAccountInformations(ctx context.Context) ([]SubAccount, error) {
	var resp []SubAccount
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/subaccounts", nil, nil, &resp)
}

// GetSubAccountBalances get balances information by currency and account type (SPOT and FUTURES) for each account in the account group.
// This is only functional for a primary user.
// A subaccount user can call /accounts/balances for SPOT account type and the futures API overview for its FUTURES balances.
func (p *Poloniex) GetSubAccountBalances(ctx context.Context) ([]SubAccountBalance, error) {
	var resp []SubAccountBalance
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/subaccounts/balances", nil, nil, &resp)
}

// GetSubAccountBalance get balances information by currency and account type (SPOT and FUTURES) for each account in the account group.
func (p *Poloniex) GetSubAccountBalance(ctx context.Context, id string) ([]SubAccountBalance, error) {
	var resp []SubAccountBalance
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, fmt.Sprintf("/subaccounts/%s/balances", id), nil, nil, &resp)
}

// SubAccountTransfer transfer amount of currency from an account and account type to another account and account type among the accounts in the account group.
// Primary account can transfer to and from any subaccounts as well as transfer between 2 subaccounts across account types.
// Subaccount can only transfer to the primary account across account types.
func (p *Poloniex) SubAccountTransfer(ctx context.Context, arg *SubAccountTransferParam) (*AccountTransferResponse, error) {
	if arg == nil {
		return nil, errNilArgument
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	if arg.FromAccountID == "" {
		return nil, fmt.Errorf("%w, fromAccountID=''", errAccountIDRequired)
	}
	if arg.ToAccountID == "" {
		return nil, fmt.Errorf("%w, toAccountID=''", errAccountIDRequired)
	}
	if arg.FromAccountType == "" {
		return nil, fmt.Errorf("%w, fromAccountType=''", errAccountTypeRequired)
	}
	if arg.ToAccountType == "" {
		return nil, fmt.Errorf("%w, toAccountType=''", errAccountTypeRequired)
	}
	var resp AccountTransferResponse
	return &resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/subaccounts/transfer", nil, arg, &resp)
}

// GetSubAccountTransferRecords get a list of transfer records of a user. Max interval for start and end time is 6 months. If no start/end time params are specified then records for last 7 days will be returned.
func (p *Poloniex) GetSubAccountTransferRecords(ctx context.Context, ccy currency.Code, startTime,
	endTime time.Time, fromAccountID, toAccountID, fromAccountType,
	toAccountType, direction string, from, limit int64) ([]SubAccountTransfer, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	if !startTime.IsZero() {
		params.Set("startTime", startTime.String())
	}
	if !endTime.IsZero() {
		params.Set("endTime", endTime.String())
	}
	if fromAccountID != "" {
		params.Set("fromAccountID", fromAccountID)
	}
	if toAccountID != "" {
		params.Set("toAccountID", toAccountID)
	}
	if fromAccountType != "" {
		params.Set("fromAccountType", fromAccountType)
	}
	if toAccountType != "" {
		params.Set("toAccountType", toAccountType)
	}
	if direction != "" {
		params.Set("direction", direction)
	}
	if from > 0 {
		params.Set("from", strconv.FormatInt(from, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []SubAccountTransfer
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/subaccounts/transfer", params, nil, &resp)
}

// GetSubAccountTransferRecord retrieves a subaccount transfer record.
func (p *Poloniex) GetSubAccountTransferRecord(ctx context.Context, id string) ([]SubAccountTransfer, error) {
	var resp []SubAccountTransfer
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, fmt.Sprintf("/subaccounts/transfer/%s", id), nil, nil, &resp)
}

// -------------------------------------  Wallet sub-accounts  ---------------------------------------

// ---------------------------------------------- End ------------------------------------------------

func intervalToString(interval kline.Interval) (string, error) {
	intervalMap := map[kline.Interval]string{
		kline.OneMin:     "MINUTE_1",
		kline.FiveMin:    "MINUTE_5",
		kline.TenMin:     "MINUTE_10",
		kline.FifteenMin: "MINUTE_15",
		kline.ThirtyMin:  "MINUTE_30",
		kline.OneHour:    "HOUR_1",
		kline.TwoHour:    "HOUR_2",
		kline.FourHour:   "HOUR_4",
		kline.SixHour:    "HOUR_6",
		kline.TwelveHour: "HOUR_12",
		kline.OneDay:     "DAY_1",
		kline.ThreeDay:   "DAY_3",
		kline.SevenDay:   "WEEK_1",
		kline.OneMonth:   "MONTH_1",
	}
	intervalString, okay := intervalMap[interval]
	if okay {
		return intervalString, nil
	}
	return "", kline.ErrUnsupportedInterval
}

func stringToInterval(interval string) (kline.Interval, error) {
	intervalMap := map[string]kline.Interval{
		"MINUTE_1":  kline.OneMin,
		"MINUTE_5":  kline.FiveMin,
		"MINUTE_10": kline.TenMin,
		"MINUTE_15": kline.FifteenMin,
		"MINUTE_30": kline.ThirtyMin,
		"HOUR_1":    kline.OneHour,
		"HOUR_2":    kline.TwoHour,
		"HOUR_4":    kline.FourHour,
		"HOUR_6":    kline.SixHour,
		"HOUR_12":   kline.TwelveHour,
		"DAY_1":     kline.OneDay,
		"DAY_3":     kline.ThreeDay,
		"WEEK_1":    kline.SevenDay,
		"MONTH_1":   kline.OneMonth,
	}
	intervalInstance, okay := intervalMap[interval]
	if okay {
		return intervalInstance, nil
	}
	return kline.Interval(0), kline.ErrUnsupportedInterval
}
