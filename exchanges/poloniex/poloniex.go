package poloniex

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	apiURL           = "https://api.poloniex.com"
	tradeSpotPath    = "/trade/"
	tradeFuturesPath = "/futures" + tradeSpotPath
	marketsPath      = "/markets/"
)

var (
	errAddressRequired         = errors.New("address is required")
	errInvalidWithdrawalChain  = errors.New("invalid withdrawal chain")
	errInvalidTimeout          = errors.New("invalid timeout")
	errChainsNotFound          = errors.New("chains not found")
	errAccountIDRequired       = errors.New("missing account ID")
	errAccountTypeRequired     = errors.New("account type required")
	errMarginAdjustTypeMissing = errors.New("margin adjust type invalid")
	errPositionModeInvalid     = errors.New("invalid position mode")
	errTrailingOffsetInvalid   = errors.New("invalid trailing offset required for trailing stop orders")
	errOffsetLimitInvalid      = errors.New("invalid offset required for trailing stop limit orders")
)

// Exchange is the overarching type across the poloniex package
type Exchange struct {
	exchange.Base
}

// GetSymbol returns symbol and trade limit info
func (e *Exchange) GetSymbol(ctx context.Context, symbol currency.Pair) ([]*SymbolDetails, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrSymbolStringEmpty
	}
	var resp []*SymbolDetails
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, unauthEPL, "/markets/"+symbol.String(), &resp)
}

// GetSymbols returns all symbols and their trade limits
func (e *Exchange) GetSymbols(ctx context.Context) ([]*SymbolDetails, error) {
	var resp []*SymbolDetails
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, unauthEPL, "/markets", &resp)
}

// GetCurrencies retrieves currencies and their details
func (e *Exchange) GetCurrencies(ctx context.Context) ([]*Currency, error) {
	var resp []*Currency
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, referenceDataEPL, "/v2/currencies", &resp)
}

// GetCurrency retrieves currency details for V2 API.
func (e *Exchange) GetCurrency(ctx context.Context, ccy currency.Code) (*Currency, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp *Currency
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, referenceDataEPL, "/v2/currencies/"+ccy.String(), &resp)
}

// GetSystemTimestamp retrieves current server time.
func (e *Exchange) GetSystemTimestamp(ctx context.Context) (time.Time, error) {
	var resp ServerSystemTime
	err := e.SendHTTPRequest(ctx, exchange.RestSpot, unauthEPL, "/timestamp", &resp)
	return resp.ServerTime.Time(), err
}

// GetMarketPrices retrieves latest trade price for all symbols.
func (e *Exchange) GetMarketPrices(ctx context.Context) ([]*MarketPrice, error) {
	var resp []*MarketPrice
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, unauthEPL, "/markets/price", &resp)
}

// GetMarketPrice retrieves latest trade price for symbols
func (e *Exchange) GetMarketPrice(ctx context.Context, symbol currency.Pair) (*MarketPrice, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var resp *MarketPrice
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, unauthEPL, marketsPath+symbol.String()+"/price", &resp)
}

// GetMarkPrices retrieves latest mark prices for all currencies
func (e *Exchange) GetMarkPrices(ctx context.Context) ([]*MarkPrice, error) {
	var resp []*MarkPrice
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, unauthEPL, "/markets/markPrice", &resp)
}

// GetMarkPrice retrieves latest mark price for all cross margin symbol.
func (e *Exchange) GetMarkPrice(ctx context.Context, symbol currency.Pair) (*MarkPrice, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var resp *MarkPrice
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, unauthEPL, marketsPath+symbol.String()+"/markPrice", &resp)
}

// GetMarkPriceComponents retrieves components of the mark price
func (e *Exchange) GetMarkPriceComponents(ctx context.Context, symbol currency.Pair) (*MarkPriceComponent, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var resp *MarkPriceComponent
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, unauthEPL, marketsPath+symbol.String()+"/markPriceComponents", &resp)
}

// GetOrderbook retrieves the order book for a given symbol
func (e *Exchange) GetOrderbook(ctx context.Context, symbol currency.Pair, scale, limit uint64) (*OrderbookData, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	if scale > 0 {
		params.Set("scale", strconv.FormatUint(scale, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp *OrderbookData
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, unauthEPL, common.EncodeURLValues(marketsPath+symbol.String()+"/orderBook", params), &resp)
}

// GetCandlesticks retrieves OHLC for a symbol at given timeframe (interval).
func (e *Exchange) GetCandlesticks(ctx context.Context, symbol currency.Pair, interval kline.Interval, startTime, endTime time.Time, limit uint64) ([]*CandlestickData, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	intervalString, err := intervalToString(interval)
	if err != nil {
		return nil, err
	} else if intervalString == "" {
		return nil, kline.ErrUnsupportedInterval
	}
	params := url.Values{}
	params.Set("interval", intervalString)
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	var resp []*CandlestickData
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, unauthEPL, common.EncodeURLValues(marketsPath+symbol.String()+"/candles", params), &resp)
}

// GetTrades returns a list of recent trades, request param limit is optional, its default value is 500, and max value is 1000.
func (e *Exchange) GetTrades(ctx context.Context, symbol currency.Pair, limit uint64) ([]*Trade, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []*Trade
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, referenceDataEPL, common.EncodeURLValues(marketsPath+symbol.String()+"/trades", params), &resp)
}

// GetTickers retrieve ticker in last 24 hours for all symbols.
func (e *Exchange) GetTickers(ctx context.Context) ([]*TickerData, error) {
	var resp []*TickerData
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, referenceDataEPL, "/markets/ticker24h", &resp)
}

// GetTicker retrieve ticker in last 24 hours for provided symbols.
func (e *Exchange) GetTicker(ctx context.Context, symbol currency.Pair) (*TickerData, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var resp *TickerData
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, referenceDataEPL, marketsPath+symbol.String()+"/ticker24h", &resp)
}

// GetCollateral retrieves collateral information of a single currency
func (e *Exchange) GetCollateral(ctx context.Context, ccy currency.Code) (*CollateralDetails, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp *CollateralDetails
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, unauthEPL, marketsPath+ccy.String()+"/collateralInfo", &resp)
}

// GetCollaterals retrieves account's collaterals information
func (e *Exchange) GetCollaterals(ctx context.Context) ([]*CollateralDetails, error) {
	var resp []*CollateralDetails
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, unauthEPL, "/markets/collateralInfo", &resp)
}

// GetBorrowRate retrieves borrow rates information for all tiers and currencies.
func (e *Exchange) GetBorrowRate(ctx context.Context) ([]*BorrowRateInfo, error) {
	var resp []*BorrowRateInfo
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, unauthEPL, "/markets/borrowRatesInfo", &resp)
}

// GetAccount retrieves all accounts of a user.
func (e *Exchange) GetAccount(ctx context.Context) ([]*AccountDetails, error) {
	var resp []*AccountDetails
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, authNonResourceIntensiveEPL, http.MethodGet, "/accounts", nil, nil, &resp)
}

// GetBalances get a list of all accounts of a user with each account’s id, type and balances (assets).
func (e *Exchange) GetBalances(ctx context.Context, accountType string) ([]*AccountBalance, error) {
	params := url.Values{}
	if accountType != "" {
		params.Set("accountType", accountType)
	}
	var resp []*AccountBalance
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, authNonResourceIntensiveEPL, http.MethodGet, "/accounts/balances", params, nil, &resp)
}

// GetBalancesByID get an accounts of a user with each account’s id, type and balances (assets).
func (e *Exchange) GetBalancesByID(ctx context.Context, accountID, accountType string) ([]*AccountBalance, error) {
	if accountID == "" {
		return nil, errAccountIDRequired
	}
	params := url.Values{}
	if accountType != "" {
		params.Set("accountType", accountType)
	}
	var resp []*AccountBalance
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, authNonResourceIntensiveEPL, http.MethodGet, "/accounts/"+accountID+"/balances", params, nil, &resp)
}

// GetAccountActivities retrieves a list of activities such as airdrop, rebates, staking, credit/debit adjustments, and other (historical adjustments).
func (e *Exchange) GetAccountActivities(ctx context.Context, startTime, endTime time.Time, activityType, limit, from uint64, direction string, ccy currency.Code) ([]*AccountActivity, error) {
	params := url.Values{}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if activityType != 0 {
		params.Set("activityType", strconv.FormatUint(activityType, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if from != 0 {
		params.Set("from", strconv.FormatUint(from, 10))
	}
	if direction != "" {
		params.Set("direction", direction)
	}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	var resp []*AccountActivity
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, authResourceIntensiveEPL, http.MethodGet, "/accounts/activity", params, nil, &resp)
}

// AccountsTransfer transfers currencies between accounts
func (e *Exchange) AccountsTransfer(ctx context.Context, arg *AccountTransferRequest) (*AccountTransferResponse, error) {
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Amount <= 0 {
		return nil, fmt.Errorf("%w, amount has to be greater than zero", order.ErrAmountIsInvalid)
	}
	if arg.FromAccount == "" {
		return nil, fmt.Errorf("%w: FromAccount", errAddressRequired)
	}
	if arg.ToAccount == "" {
		return nil, fmt.Errorf("%w: ToAccount", errAddressRequired)
	}
	var resp *AccountTransferResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, authResourceIntensiveEPL, http.MethodPost, "/accounts/transfer", nil, arg, &resp)
}

// GetAccountsTransferRecords gets a list of transfer records of a user
func (e *Exchange) GetAccountsTransferRecords(ctx context.Context, startTime, endTime time.Time, direction string, ccy currency.Code, from, limit uint64) ([]*AccountTransferRecord, error) {
	params := url.Values{}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	if direction != "" {
		params.Set("direction", direction)
	}
	if from != 0 {
		params.Set("from", strconv.FormatUint(from, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []*AccountTransferRecord
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, authNonResourceIntensiveEPL, http.MethodGet, "/accounts/transfer", params, nil, &resp)
}

// GetAccountsTransferRecordByTransferID gets a transfer record of a user.
func (e *Exchange) GetAccountsTransferRecordByTransferID(ctx context.Context, transferID string) ([]*AccountTransferRecord, error) {
	if transferID == "" {
		return nil, errAccountIDRequired
	}
	var resp AccountTransferRecords
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, authNonResourceIntensiveEPL, http.MethodGet, "/accounts/transfer/"+transferID, nil, nil, &resp)
}

// GetFeeInfo retrieves fee rate for an account
func (e *Exchange) GetFeeInfo(ctx context.Context) (*FeeInfo, error) {
	var resp *FeeInfo
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, authResourceIntensiveEPL, http.MethodGet, "/feeinfo", nil, nil, &resp)
}

// GetInterestHistory get a list of interest collection records of a user.
func (e *Exchange) GetInterestHistory(ctx context.Context, startTime, endTime time.Time, direction string, from, limit uint64) ([]*InterestHistory, error) {
	params := url.Values{}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if direction != "" {
		params.Set("direction", direction)
	}
	if from != 0 {
		params.Set("from", strconv.FormatUint(from, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []*InterestHistory
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, authNonResourceIntensiveEPL, http.MethodGet, "/accounts/interest/history", params, nil, &resp)
}

// GetSubAccount get a list of all the accounts within an Account Group for a user.
func (e *Exchange) GetSubAccount(ctx context.Context) ([]*SubAccount, error) {
	var resp []*SubAccount
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, authNonResourceIntensiveEPL, http.MethodGet, "/subaccounts", nil, nil, &resp)
}

// GetSubAccountBalances retrieves balances by currency and account type (SPOT or FUTURES)
// for all accounts in the group. Available only to the primary user.
// Subaccounts should use GetBalances() for SPOT and the Futures API for FUTURES.
func (e *Exchange) GetSubAccountBalances(ctx context.Context) ([]*SubAccountBalances, error) {
	var resp []*SubAccountBalances
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, authResourceIntensiveEPL, http.MethodGet, "/subaccounts/balances", nil, nil, &resp)
}

// GetSubAccountBalance get balances information by currency and account type (SPOT and FUTURES) for each account in the account group.
func (e *Exchange) GetSubAccountBalance(ctx context.Context, subAccountID string) ([]*SubAccountBalances, error) {
	if subAccountID == "" {
		return nil, fmt.Errorf("%w: empty subAccountID", errAccountIDRequired)
	}
	var resp []*SubAccountBalances
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, authNonResourceIntensiveEPL, http.MethodGet, "/subaccounts/"+subAccountID+"/balances", nil, nil, &resp)
}

// SubAccountTransfer transfers currencies between accounts in the account group
// Primary account can transfer to and from any subaccounts as well as transfer between 2 subaccounts across account types.
// Subaccount can only transfer to the primary account across account types.
func (e *Exchange) SubAccountTransfer(ctx context.Context, arg *SubAccountTransferRequest) (*AccountTransferResponse, error) {
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	if arg.FromAccountID == "" {
		return nil, fmt.Errorf("%w: FromAccountID", errAccountIDRequired)
	}
	if arg.ToAccountID == "" {
		return nil, fmt.Errorf("%w: ToAccountID", errAccountIDRequired)
	}
	if arg.FromAccountType == "" {
		return nil, fmt.Errorf("%w: FromAccountType", errAccountTypeRequired)
	}
	if arg.ToAccountType == "" {
		return nil, fmt.Errorf("%w: ToAccountType", errAccountTypeRequired)
	}
	var resp *AccountTransferResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, authNonResourceIntensiveEPL, http.MethodPost, "/subaccounts/transfer", nil, arg, &resp)
}

// GetSubAccountTransferRecords gets a list of transfer records of a user. Max interval for start and end time is 6 months.
func (e *Exchange) GetSubAccountTransferRecords(ctx context.Context, arg *SubAccountTransferRecordRequest) ([]*SubAccountTransfer, error) {
	params := url.Values{}
	if !arg.Currency.IsEmpty() {
		params.Set("currency", arg.Currency.String())
	}
	if !arg.StartTime.IsZero() && !arg.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(arg.StartTime, arg.EndTime); err != nil {
			return nil, err
		}
		params.Set("startTime", arg.StartTime.String())
		params.Set("endTime", arg.EndTime.String())
	}
	if arg.FromAccountID != "" {
		params.Set("fromAccountID", arg.FromAccountID)
	}
	if arg.ToAccountID != "" {
		params.Set("toAccountID", arg.ToAccountID)
	}
	if arg.FromAccountType != "" {
		params.Set("fromAccountType", arg.FromAccountType)
	}
	if arg.ToAccountType != "" {
		params.Set("toAccountType", arg.ToAccountType)
	}
	if arg.Direction != "" {
		params.Set("direction", arg.Direction)
	}
	if arg.From > 0 {
		params.Set("from", strconv.FormatUint(arg.From, 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []*SubAccountTransfer
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, authResourceIntensiveEPL, http.MethodGet, "/subaccounts/transfer", params, nil, &resp)
}

// GetSubAccountTransferRecord retrieves a subaccount transfer record.
func (e *Exchange) GetSubAccountTransferRecord(ctx context.Context, id string) ([]*SubAccountTransfer, error) {
	if id == "" {
		return nil, fmt.Errorf("%w: subAccountID is missing", errAccountIDRequired)
	}
	var resp []*SubAccountTransfer
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, authResourceIntensiveEPL, http.MethodGet, "/subaccounts/transfer/"+id, nil, nil, &resp)
}

// GetDepositAddresses get all deposit addresses for a user.
func (e *Exchange) GetDepositAddresses(ctx context.Context, ccy currency.Code) (DepositAddresses, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	var addresses DepositAddresses
	return addresses, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, authResourceIntensiveEPL, http.MethodGet, "/wallets/addresses", params, nil, &addresses)
}

// WalletActivity returns the wallet activity between set start and end time
func (e *Exchange) WalletActivity(ctx context.Context, start, end time.Time, activityType string) (*WalletActivity, error) {
	values := url.Values{}
	if err := common.StartEndTimeCheck(start, end); err != nil {
		return nil, err
	}
	values.Set("start", strconv.FormatInt(start.Unix(), 10))
	values.Set("end", strconv.FormatInt(end.Unix(), 10))
	if activityType != "" {
		values.Set("activityType", activityType)
	}
	var resp *WalletActivity
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, authResourceIntensiveEPL, http.MethodGet, "/wallets/activity", values, nil, &resp)
}

// NewCurrencyDepositAddress creates a new deposit address for a currency.
func (e *Exchange) NewCurrencyDepositAddress(ctx context.Context, ccy currency.Code) (string, error) {
	if ccy.IsEmpty() {
		return "", currency.ErrCurrencyCodeEmpty
	}
	var resp struct {
		Address string `json:"address"`
	}
	return resp.Address, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, authResourceIntensiveEPL, http.MethodPost, "/wallets/address", nil, map[string]string{"currency": ccy.String()}, &resp)
}

var supportedIntervals = []struct {
	key string
	val kline.Interval
}{
	{key: "MINUTE_1", val: kline.OneMin},
	{key: "MINUTE_5", val: kline.FiveMin},
	{key: "MINUTE_10", val: kline.TenMin},
	{key: "MINUTE_15", val: kline.FifteenMin},
	{key: "MINUTE_30", val: kline.ThirtyMin},
	{key: "HOUR_1", val: kline.OneHour},
	{key: "HOUR_2", val: kline.TwoHour},
	{key: "HOUR_4", val: kline.FourHour},
	{key: "HOUR_6", val: kline.SixHour},
	{key: "HOUR_12", val: kline.TwelveHour},
	{key: "DAY_1", val: kline.OneDay},
	{key: "DAY_3", val: kline.ThreeDay},
	{key: "WEEK_1", val: kline.SevenDay},
	{key: "MONTH_1", val: kline.OneMonth},
}

func stringToInterval(interval string) (kline.Interval, error) {
	interval = strings.ToUpper(interval)
	for x := range supportedIntervals {
		if supportedIntervals[x].key == interval {
			return supportedIntervals[x].val, nil
		}
	}
	return kline.Interval(0), fmt.Errorf("%w: %q", kline.ErrUnsupportedInterval, interval)
}

func intervalToString(interval kline.Interval) (string, error) {
	for x := range supportedIntervals {
		if supportedIntervals[x].val == interval {
			return supportedIntervals[x].key, nil
		}
	}
	return "", kline.ErrUnsupportedInterval
}

// WithdrawCurrency withdraws a currency to a specific delegated address
func (e *Exchange) WithdrawCurrency(ctx context.Context, arg *WithdrawCurrencyRequest) (*Withdraw, error) {
	if arg.Coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if arg.Network == "" {
		return nil, errInvalidWithdrawalChain
	}
	if arg.Address == "" {
		return nil, errAddressRequired
	}
	var resp *Withdraw
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, authNonResourceIntensiveEPL, http.MethodPost, "/v2/wallets/withdraw", nil, arg, &resp)
}

// GetAccountMargin retrieves account margin information
func (e *Exchange) GetAccountMargin(ctx context.Context, accountType string) (*AccountMargin, error) {
	params := url.Values{}
	if accountType != "" {
		params.Set("accountType", accountType)
	}
	var resp *AccountMargin
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, authNonResourceIntensiveEPL, http.MethodGet, "/margin/accountMargin", params, nil, &resp)
}

// GetBorrowStatus retrieves borrow status of currencies
func (e *Exchange) GetBorrowStatus(ctx context.Context, ccy currency.Code) ([]*BorrowStatus, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	var resp []*BorrowStatus
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, authNonResourceIntensiveEPL, http.MethodGet, "/margin/borrowStatus", params, nil, &resp)
}

// MaximumBuySellAmount get maximum and available buy/sell amount for a given symbol.
func (e *Exchange) MaximumBuySellAmount(ctx context.Context, symbol currency.Pair) (*MaxBuySellAmount, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	var resp *MaxBuySellAmount
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, authNonResourceIntensiveEPL, http.MethodGet, "/margin/maxSize", params, nil, &resp)
}

// PlaceOrder places an order
func (e *Exchange) PlaceOrder(ctx context.Context, arg *PlaceOrderRequest) (*PlaceOrderResponse, error) {
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	var resp *PlaceOrderResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, authResourceIntensiveEPL, http.MethodPost, "/orders", nil, arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	} else if resp.Code != 0 && resp.Code != 200 {
		return resp, fmt.Errorf("%w: code: %d message: %s", common.ErrNoResponse, resp.Code, resp.Message)
	}
	return resp, nil
}

// PlaceBatchOrders places a batch of orders
func (e *Exchange) PlaceBatchOrders(ctx context.Context, args []PlaceOrderRequest) ([]*PlaceBatchOrderItem, error) {
	if len(args) == 0 {
		return nil, common.ErrNilPointer
	}
	for x := range args {
		if args[x].Symbol.IsEmpty() {
			return nil, currency.ErrCurrencyPairEmpty
		}
		if args[x].Side == "" {
			return nil, order.ErrSideIsInvalid
		}
		if args[x].Amount <= 0 {
			return nil, limits.ErrAmountBelowMin
		}
	}
	var resp []*PlaceBatchOrderItem
	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, authResourceIntensiveEPL, http.MethodPost, "/orders/batch", nil, args, &resp)
	if err != nil {
		return nil, err
	}
	for _, r := range resp {
		if r.Code != 0 && r.Code != 200 {
			err = common.AppendError(err, fmt.Errorf("%w: code: %d message: %s", common.ErrNoResponse, r.Code, r.Message))
		}
	}
	return resp, err
}

// CancelReplaceOrder cancels an existing active order, new or partially filled, and places a new order
func (e *Exchange) CancelReplaceOrder(ctx context.Context, arg *CancelReplaceOrderRequest) (*CancelReplaceOrderResponse, error) {
	if arg.orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp *CancelReplaceOrderResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, authNonResourceIntensiveEPL, http.MethodPut, "/orders/"+arg.orderID, nil, arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	} else if resp.Code != 0 && resp.Code != 200 {
		return resp, fmt.Errorf("%w: code: %d message: %s", common.ErrNoResponse, resp.Code, resp.Message)
	}
	return resp, nil
}

// GetOpenOrders retrieves a list of active orders
func (e *Exchange) GetOpenOrders(ctx context.Context, symbol currency.Pair, side, direction, fromOrderID string, limit uint64) ([]*TradeOrder, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		params.Set("symbol", symbol.String())
	}
	if side != "" {
		params.Set("side", side)
	}
	if fromOrderID != "" {
		params.Set("from", fromOrderID)
	}
	if direction != "" {
		params.Set("direction", direction)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []*TradeOrder
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, authResourceIntensiveEPL, http.MethodGet, "/orders", params, nil, &resp)
}

// GetOrder gets an order’s status by orderId or clientOrderId
func (e *Exchange) GetOrder(ctx context.Context, id, clientOrderID string) (*TradeOrder, error) {
	var path string
	switch {
	case id != "":
		path = "/orders/" + id
	case clientOrderID != "":
		path = "/orders/cid:" + id
	default:
		return nil, fmt.Errorf("%w, orderid or client order id is required", order.ErrOrderIDNotSet)
	}
	var resp *TradeOrder
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, authResourceIntensiveEPL, http.MethodGet, path, nil, nil, &resp)
}

// CancelOrderByID cancels an active order
func (e *Exchange) CancelOrderByID(ctx context.Context, id string) (*CancelOrderResponse, error) {
	if id == "" {
		return nil, fmt.Errorf("%w; order 'id' is required", order.ErrOrderIDNotSet)
	}
	var resp *CancelOrderResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, authResourceIntensiveEPL, http.MethodDelete, "/orders/"+id, nil, nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	} else if resp.Code != 0 && resp.Code != 200 {
		return resp, fmt.Errorf("%w: code: %d message: %s", common.ErrNoResponse, resp.Code, resp.Message)
	}
	return resp, nil
}

// CancelOrdersByIDs cancels multiple orders
func (e *Exchange) CancelOrdersByIDs(ctx context.Context, orderIDs, clientOrderIDs []string) ([]*CancelOrderResponse, error) {
	if len(orderIDs) == 0 && len(clientOrderIDs) == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	params := make(map[string]any)
	if len(orderIDs) > 0 {
		params["orderIds"] = orderIDs
	}
	if len(clientOrderIDs) > 0 {
		params["clientOrderIds"] = clientOrderIDs
	}
	var resp []*CancelOrderResponse
	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, authResourceIntensiveEPL, http.MethodDelete, "/orders/cancelByIds", nil, params, &resp)
	if err != nil {
		return nil, err
	}
	for _, r := range resp {
		if r.Code != 0 && r.Code != 200 {
			err = common.AppendError(err, fmt.Errorf("%w: code: %d message: %s", common.ErrNoResponse, r.Code, r.Message))
		}
	}
	return resp, err
}

// CancelTradeOrders batch cancel all orders in an account
func (e *Exchange) CancelTradeOrders(ctx context.Context, symbols []string, accountTypes []accountType) ([]*CancelOrderResponse, error) {
	args := make(map[string]any)
	if len(symbols) != 0 {
		args["symbols"] = symbols
	}
	if len(accountTypes) > 0 {
		args["accountTypes"] = accountTypes
	}
	var resp []*CancelOrderResponse
	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, authResourceIntensiveEPL, http.MethodDelete, "/orders", nil, args, &resp)
	if err != nil {
		return nil, err
	}
	for _, r := range resp {
		if r.Code != 0 && r.Code != 200 {
			err = common.AppendError(err, fmt.Errorf("%w: code: %d message: %s", common.ErrNoResponse, r.Code, r.Message))
		}
	}
	return resp, err
}

// KillSwitch set a timer that cancels all regular and smartorders after the timeout has expired.
// Timeout can be reset by calling this command again with a new timeout value.
// timeout value in seconds; range is 10 seconds to 10 minutes or 600 seconds
func (e *Exchange) KillSwitch(ctx context.Context, timeout time.Duration) (*KillSwitchStatus, error) {
	var timeoutString string
	if timeout < time.Second*10 || timeout > time.Minute*10 {
		return nil, fmt.Errorf("%w: timeout possible values must be between 10 second to 10 minute", errInvalidTimeout)
	}
	timeoutString = strconv.FormatInt(int64(timeout.Seconds()), 10)
	var resp *KillSwitchStatus
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, authNonResourceIntensiveEPL, http.MethodPost, "/orders/killSwitch", nil, map[string]any{"timeout": timeoutString}, &resp)
}

// DisableKillSwitch disables the timer to cancels all regular and smartorders
func (e *Exchange) DisableKillSwitch(ctx context.Context) (*KillSwitchStatus, error) {
	var resp *KillSwitchStatus
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, authNonResourceIntensiveEPL, http.MethodPost, "/orders/killSwitch", nil, map[string]any{"timeout": "-1"}, &resp)
}

// GetKillSwitchStatus get status of kill switch
func (e *Exchange) GetKillSwitchStatus(ctx context.Context) (*KillSwitchStatus, error) {
	var resp *KillSwitchStatus
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, authNonResourceIntensiveEPL, http.MethodGet, "/orders/killSwitchStatus", nil, nil, &resp)
}

// CreateSmartOrder create a smart order for an account. Funds will only be frozen when the smart order triggers, not upon smart order creation
func (e *Exchange) CreateSmartOrder(ctx context.Context, arg *SmartOrderRequestRequest) (*PlaceOrderResponse, error) {
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if arg.Side != order.Buy && arg.Side != order.Sell {
		return nil, order.ErrSideIsInvalid
	}
	if arg.Quantity <= 0 {
		return nil, fmt.Errorf("%w; base quantity is required", limits.ErrAmountBelowMin)
	}
	if strings.EqualFold(arg.Type, "STOP_LIMIT") && arg.Price <= 0 {
		return nil, fmt.Errorf("%w %w", order.ErrPriceMustBeSetIfLimitOrder, limits.ErrPriceBelowMin)
	}
	if (strings.EqualFold(arg.Type, "TRAILING_STOP") || strings.EqualFold(arg.Type, "TRAILING_STOP_LIMIT")) && arg.TrailingOffset == "" {
		return nil, errTrailingOffsetInvalid
	}
	if strings.EqualFold(arg.Type, "TRAILING_STOP_LIMIT") && arg.LimitOffset == "" {
		return nil, errOffsetLimitInvalid
	}
	var resp *PlaceOrderResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, authNonResourceIntensiveEPL, http.MethodPost, "/smartorders", nil, arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	} else if resp.Code != 0 && resp.Code != 200 {
		return resp, fmt.Errorf("%w: code: %d message: %s", common.ErrNoResponse, resp.Code, resp.Message)
	}
	return resp, nil
}

func orderPath(orderID, idPath, clientOrderID, clientIDPath string) (string, error) {
	switch {
	case orderID != "":
		return idPath + orderID, nil
	case clientOrderID != "":
		return clientIDPath + clientOrderID, nil
	default:
		return "", order.ErrOrderIDNotSet
	}
}

// CancelReplaceSmartOrder cancel an existing untriggered smart order and place a new smart order on the same symbol with details from existing smart order unless amended by new parameters
func (e *Exchange) CancelReplaceSmartOrder(ctx context.Context, arg *CancelReplaceSmartOrderRequest) (*CancelReplaceSmartOrderResponse, error) {
	path, err := orderPath(arg.orderID, "/smartorders/", arg.ClientOrderID, "/smartorders/cid:")
	if err != nil {
		return nil, err
	}
	var resp *CancelReplaceSmartOrderResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, authResourceIntensiveEPL, http.MethodPut, path, nil, arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	} else if resp.Code != 0 && resp.Code != 200 {
		return resp, fmt.Errorf("%w: code: %d message: %s", common.ErrNoResponse, resp.Code, resp.Message)
	}
	return resp, nil
}

// GetSmartOpenOrders get a list of (pending) smart orders for an account
func (e *Exchange) GetSmartOpenOrders(ctx context.Context, limit uint64, types []string) ([]*SmartOrder, error) {
	params := url.Values{}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if len(types) > 0 {
		params.Set("types", strings.Join(types, ","))
	}
	var resp []*SmartOrder
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, authResourceIntensiveEPL, http.MethodGet, "/smartorders", params, nil, &resp)
}

// GetSmartOrderDetails retrieves a smart order's detail
func (e *Exchange) GetSmartOrderDetails(ctx context.Context, orderID, clientSuppliedID string) ([]*SmartOrderDetails, error) {
	path, err := orderPath(orderID, "/smartorders/", clientSuppliedID, "/smartorders/cid:")
	if err != nil {
		return nil, err
	}
	var resp []*SmartOrderDetails
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, authNonResourceIntensiveEPL, http.MethodGet, path, nil, nil, &resp)
}

// CancelSmartOrderByID cancel a smart order by its id.
func (e *Exchange) CancelSmartOrderByID(ctx context.Context, id, clientSuppliedID string) (*CancelSmartOrderResponse, error) {
	path, err := orderPath(id, "/smartorders/", clientSuppliedID, "/smartorders/cid:")
	if err != nil {
		return nil, err
	}
	var resp *CancelSmartOrderResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, authNonResourceIntensiveEPL, http.MethodDelete, path, nil, nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	} else if resp.Code != 0 && resp.Code != 200 {
		return resp, fmt.Errorf("%w: code: %d message: %s", common.ErrNoResponse, resp.Code, resp.Message)
	}
	return resp, nil
}

// CancelMultipleSmartOrders performs a batch cancel one or many smart orders in an account by IDs.
func (e *Exchange) CancelMultipleSmartOrders(ctx context.Context, args *CancelOrdersRequest) ([]*CancelOrderResponse, error) {
	if args == nil {
		return nil, common.ErrNilPointer
	}
	if len(args.ClientOrderIDs) == 0 && len(args.OrderIDs) == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	var resp []*CancelOrderResponse
	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, authResourceIntensiveEPL, http.MethodDelete, "/smartorders/cancelByIds", nil, args, &resp)
	if err != nil {
		return nil, err
	}
	for _, r := range resp {
		if r.Code != 0 && r.Code != 200 {
			err = common.AppendError(err, fmt.Errorf("%w: code: %d message:%s", common.ErrNoResponse, r.Code, r.Message))
		}
	}
	return resp, err
}

// CancelSmartOrders cancels all smart orders in an account.
func (e *Exchange) CancelSmartOrders(ctx context.Context, symbols, accountTypes, orderTypes []string) ([]*CancelOrderResponse, error) {
	args := make(map[string][]string)
	if len(symbols) != 0 {
		args["symbols"] = symbols
	}
	if len(accountTypes) > 0 {
		args["accountTypes"] = accountTypes
	}
	if len(orderTypes) > 0 {
		args["orderTypes"] = orderTypes
	}
	var resp []*CancelOrderResponse
	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, authResourceIntensiveEPL, http.MethodDelete, "/smartorders", nil, args, &resp)
	if err != nil {
		return nil, err
	}
	for _, r := range resp {
		if r.Code != 0 && r.Code != 200 {
			err = common.AppendError(err, fmt.Errorf("%w: code: %d message:%s", common.ErrNoResponse, r.Code, r.Message))
		}
	}
	return resp, err
}

func orderFillParams(arg *OrdersHistoryRequest) (url.Values, error) {
	params := url.Values{}
	if arg.AccountType != "" {
		params.Set("accountType", arg.AccountType)
	}
	if arg.OrderType != "" {
		params.Set("type", arg.OrderType)
	}
	if len(arg.OrderTypes) > 0 {
		params.Set("types", strings.Join(arg.OrderTypes, ","))
	}
	if arg.Side != "" {
		params.Set("side", arg.Side)
	}
	if !arg.Symbol.IsEmpty() {
		params.Set("symbol", arg.Symbol.String())
	}
	if arg.From > 0 {
		params.Set("from", strconv.FormatInt(arg.From, 10))
	}
	if arg.Direction != "" {
		params.Set("direction", arg.Direction)
	}
	if arg.States != "" {
		params.Set("states", arg.States)
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatInt(arg.Limit, 10))
	}
	if arg.HideCancel {
		params.Set("hideCancel", "true")
	}
	if !arg.StartTime.IsZero() && !arg.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(arg.StartTime, arg.EndTime); err != nil {
			return nil, err
		}
		params.Set("startTime", arg.StartTime.String())
		params.Set("endTime", arg.EndTime.String())
	}
	return params, nil
}

// GetOrdersHistory get a list of historical orders in an account
func (e *Exchange) GetOrdersHistory(ctx context.Context, arg *OrdersHistoryRequest) ([]*TradeOrder, error) {
	params, err := orderFillParams(arg)
	if err != nil {
		return nil, err
	}
	var resp []*TradeOrder
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, authResourceIntensiveEPL, http.MethodGet, "/orders/history", params, nil, &resp)
}

// GetSmartOrderHistory get a list of historical smart orders in an account
func (e *Exchange) GetSmartOrderHistory(ctx context.Context, arg *OrdersHistoryRequest) ([]*SmartOrder, error) {
	params, err := orderFillParams(arg)
	if err != nil {
		return nil, err
	}
	var resp []*SmartOrder
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, authResourceIntensiveEPL, http.MethodGet, "/smartorders/history", params, nil, &resp)
}

// GetTradeHistory get a list of all trades for an account
func (e *Exchange) GetTradeHistory(ctx context.Context, symbols currency.Pairs, direction string, from, limit uint64, startTime, endTime time.Time) ([]*TradeHistory, error) {
	params := url.Values{}
	if len(symbols) != 0 {
		params.Set("symbols", symbols.Join())
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if from > 0 {
		params.Set("from", strconv.FormatUint(from, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if direction != "" {
		params.Set("direction", direction)
	}
	var resp []*TradeHistory
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, authResourceIntensiveEPL, http.MethodGet, "/trades", params, nil, &resp)
}

// GetTradesByOrderID gets trades for an order
func (e *Exchange) GetTradesByOrderID(ctx context.Context, orderID string) ([]*TradeHistory, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp []*TradeHistory
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, authNonResourceIntensiveEPL, http.MethodGet, "/orders/"+orderID+"/trades", nil, nil, &resp)
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (e *Exchange) SendHTTPRequest(ctx context.Context, ep exchange.URL, epl request.EndpointLimit, path string, result any) error {
	endpoint, err := e.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	resp := result
	if strings.HasPrefix(path, v3Path) {
		resp = &struct {
			Code int64  `json:"code"`
			Msg  string `json:"msg"`
			Data any    `json:"data"`
		}{
			Data: result,
		}
	}
	item := &request.Item{
		Method:                 http.MethodGet,
		Path:                   endpoint + path,
		Result:                 &resp,
		Verbose:                e.Verbose,
		HTTPDebugging:          e.HTTPDebugging,
		HTTPRecording:          e.HTTPRecording,
		HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
	}
	if err := e.SendPayload(ctx, epl, func() (*request.Item, error) {
		return item, nil
	}, request.UnauthenticatedRequest); err != nil {
		return err
	}
	if result == nil {
		return common.ErrNoResponse
	}
	if strings.HasPrefix(path, v3Path) || strings.HasPrefix(path, "/smartorders/") {
		if val, ok := resp.(*V3ResponseWrapper); ok {
			if val.Code != 0 && val.Code != 200 {
				return fmt.Errorf("code: %d message: %s", val.Code, val.Msg)
			}
		}
	}
	return nil
}

// SendAuthenticatedHTTPRequest sends an authenticated HTTP request
func (e *Exchange) SendAuthenticatedHTTPRequest(ctx context.Context, ep exchange.URL, epl request.EndpointLimit, method, path string, values url.Values, body, result any) error {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	endpoint, err := e.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	resp := result
	requiresWrapper := strings.HasPrefix(path, v3Path) || strings.HasPrefix(path, "/smartorders/")
	if requiresWrapper {
		resp = &V3ResponseWrapper{
			Data: result,
		}
	}
	requestFunc := func() (*request.Item, error) {
		headers := make(map[string]string)
		headers["Content-Type"] = "application/json"
		headers["key"] = creds.Key
		headers["recvWindow"] = strconv.FormatInt(1500, 10)
		if values == nil {
			values = url.Values{}
		}
		timestamp := time.Now()
		bodyPayload := []byte("{}")
		signTimestamp := strconv.FormatInt(timestamp.UnixMilli(), 10)
		values.Set("signTimestamp", signTimestamp)
		var signatureStrings string
		switch method {
		case http.MethodGet, "get":
			signatureStrings = http.MethodGet + "\n" + path + "\n" + values.Encode()
		default:
			if body != nil {
				bodyPayload, err = json.Marshal(body)
				if err != nil {
					return nil, err
				}
			}
			if string(bodyPayload) != "{}" {
				signatureStrings = method + "\n" + path + "\n" + "requestBody=" + string(bodyPayload) + "&" + values.Encode()
			} else {
				signatureStrings = method + "\n" + path + "\n" + values.Encode()
			}
		}
		var hmac []byte
		hmac, err = crypto.GetHMAC(crypto.HashSHA256,
			[]byte(signatureStrings),
			[]byte(creds.Secret))
		if err != nil {
			return nil, err
		}

		headers["signatureMethod"] = "hmacSHA256"
		headers["signature"] = base64.StdEncoding.EncodeToString(hmac)
		headers["signTimestamp"] = signTimestamp
		values.Del("signTimestamp")

		req := &request.Item{
			Method:        method,
			Path:          common.EncodeURLValues(endpoint+path, values),
			Result:        resp,
			Headers:       headers,
			Verbose:       e.Verbose,
			HTTPDebugging: e.HTTPDebugging,
			HTTPRecording: e.HTTPRecording,
		}
		if method != http.MethodGet && len(bodyPayload) > 0 && string(bodyPayload) != "{}" {
			req.Body = bytes.NewBuffer(bodyPayload)
		}
		return req, nil
	}
	if err := e.SendPayload(ctx, epl, requestFunc, request.AuthenticatedRequest); err != nil {
		return fmt.Errorf("%w %w", request.ErrAuthRequestFailed, err)
	} else if result == nil {
		return common.ErrNoResponse
	}
	if requiresWrapper {
		if val, ok := resp.(*V3ResponseWrapper); ok {
			if val.Code != 0 && val.Code != 200 {
				return fmt.Errorf("%w code: %d message: %s", request.ErrAuthRequestFailed, val.Code, val.Msg)
			}
		}
	}
	return nil
}

// GetFee returns an estimate of fee based on type of transaction
func (e *Exchange) GetFee(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		feeInfo, err := e.GetFeeInfo(ctx)
		if err != nil {
			return 0, err
		}
		fee = calculateTradingFee(feeInfo,
			feeBuilder.PurchasePrice,
			feeBuilder.Amount,
			feeBuilder.IsMaker)

	case exchange.CryptocurrencyWithdrawalFee:
		fee = getWithdrawalFee(feeBuilder.Pair.Base)
	case exchange.OfflineTradeFee:
		fee = getOfflineTradeFee(feeBuilder.PurchasePrice, feeBuilder.Amount)
	}
	if fee < 0 {
		fee = 0
	}
	return fee, nil
}

// getOfflineTradeFee calculates the worst case-scenario trading fee
func getOfflineTradeFee(price, amount float64) float64 {
	return 0.002 * price * amount
}

func calculateTradingFee(feeInfo *FeeInfo, purchasePrice, amount float64, isMaker bool) (fee float64) {
	if isMaker {
		fee = feeInfo.MakerRate.Float64()
	} else {
		fee = feeInfo.TakerRate.Float64()
	}
	return fee * amount * purchasePrice
}

func getWithdrawalFee(c currency.Code) float64 {
	return WithdrawalFees[c]
}
