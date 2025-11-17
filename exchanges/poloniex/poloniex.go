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
	"sync"
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
	v3Path           = "/v3/"
	mainURL          = "https://www.poloniex.com"
	apiURL           = "https://api.poloniex.com"
	tradeSpotPath    = "/trade/"
	tradeFuturesPath = "/futures/trade/"
	marketsPath      = "/markets/"
)

var (
	errAddressRequired        = errors.New("address is required")
	errInvalidWithdrawalChain = errors.New("invalid withdrawal chain")
	errInvalidTimeout         = errors.New("invalid timeout")
	errAccountIDRequired      = errors.New("missing account ID")
	errAccountTypeRequired    = errors.New("account type required")
	errInvalidTrailingOffset  = errors.New("invalid trailing offset required for trailing stop orders")
	errInvalidOffsetLimit     = errors.New("invalid offset required for trailing stop limit orders")
)

// Exchange is the overarching type across the poloniex package
type Exchange struct {
	exchange.Base

	onceWebsocketOrderbookCache map[currency.Pair]bool
	onceWebsocketOrderbookLock  sync.Mutex
}

// GetSymbol returns symbol and trade limit info
func (e *Exchange) GetSymbol(ctx context.Context, symbol currency.Pair) ([]*SymbolDetails, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrSymbolStringEmpty
	}
	var resp []*SymbolDetails
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, unauthEPL, marketsPath+symbol.String(), &resp)
}

// GetSymbols returns all symbols and their trade limits
func (e *Exchange) GetSymbols(ctx context.Context) ([]*SymbolDetails, error) {
	var resp []*SymbolDetails
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, referenceDataEPL, "/markets", &resp)
}

// GetCurrencies retrieves currencies and their details
func (e *Exchange) GetCurrencies(ctx context.Context) ([]*Currency, error) {
	var resp []*Currency
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, referenceDataEPL, "/v2/currencies", &resp)
}

// GetCurrency retrieves a currency's details
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
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, referenceDataEPL, "/markets/price", &resp)
}

// GetMarketPrice retrieves latest trade price for a symbol
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
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, referenceDataEPL, "/markets/markPrice", &resp)
}

// GetMarkPrice retrieves latest mark price for a symbol.
func (e *Exchange) GetMarkPrice(ctx context.Context, symbol currency.Pair) (*MarkPrice, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var resp *MarkPrice
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, unauthEPL, marketsPath+symbol.String()+"/markPrice", &resp)
}

// GetMarkPriceComponents retrieves components of a mark price
func (e *Exchange) GetMarkPriceComponents(ctx context.Context, symbol currency.Pair) (*MarkPriceComponents, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var resp *MarkPriceComponents
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, unauthEPL, marketsPath+symbol.String()+"/markPriceComponents", &resp)
}

// GetOrderbook retrieves the order book for a symbol
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

// GetCandlesticks retrieves candlestick data
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
	params.Set("interval", intervalString)
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
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
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, unauthEPL, common.EncodeURLValues(marketsPath+symbol.String()+"/trades", params), &resp)
}

// GetTickers retrieves tickers for last 24 hours for all symbols.
func (e *Exchange) GetTickers(ctx context.Context) ([]*TickerData, error) {
	var resp []*TickerData
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, referenceDataEPL, "/markets/ticker24h", &resp)
}

// GetTicker retrieves ticker for last 24 hours for a symbol
func (e *Exchange) GetTicker(ctx context.Context, symbol currency.Pair) (*TickerData, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var resp *TickerData
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, unauthEPL, marketsPath+symbol.String()+"/ticker24h", &resp)
}

// GetCollateral retrieves collateral information for a currency
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
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, referenceDataEPL, "/markets/collateralInfo", &resp)
}

// GetBorrowRate retrieves borrow rates information for all tiers and currencies.
func (e *Exchange) GetBorrowRate(ctx context.Context) ([]*BorrowRateDetails, error) {
	var resp []*BorrowRateDetails
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, unauthEPL, "/markets/borrowRatesInfo", &resp)
}

// GetAccount retrieves all accounts of a user.
func (e *Exchange) GetAccount(ctx context.Context) ([]*AccountDetails, error) {
	var resp []*AccountDetails
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, sAccountInfoEPL, http.MethodGet, "/accounts", nil, nil, &resp)
}

// GetBalances retrieves all account balances for the authorised user
func (e *Exchange) GetBalances(ctx context.Context, accountType string) ([]*AccountBalances, error) {
	params := url.Values{}
	if accountType != "" {
		params.Set("accountType", accountType)
	}
	var resp []*AccountBalances
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, sAccountBalancesEPL, http.MethodGet, "/accounts/balances", params, nil, &resp)
}

// GetAccountBalances gets balances for an account
func (e *Exchange) GetAccountBalances(ctx context.Context, accountID, accountType string) ([]*AccountBalances, error) {
	if accountID == "" {
		return nil, errAccountIDRequired
	}
	params := url.Values{}
	if accountType != "" {
		params.Set("accountType", accountType)
	}
	var resp []*AccountBalances
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, sAccountBalancesEPL, http.MethodGet, "/accounts/"+accountID+"/balances", params, nil, &resp)
}

// GetAccountActivities retrieves a list of activities such as airdrop, rebates, staking, credit/debit adjustments, and other (historical adjustments).
func (e *Exchange) GetAccountActivities(ctx context.Context, startTime, endTime time.Time, activityType, limit, from uint64, direction string, ccy currency.Code) ([]*AccountActivity, error) {
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
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, sAccountActivityEPL, http.MethodGet, "/accounts/activity", params, nil, &resp)
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
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, sAccountsTransferEPL, http.MethodPost, "/accounts/transfer", nil, arg, &resp)
}

// GetAccountsTransferRecords gets a list of transfer records of a user
func (e *Exchange) GetAccountsTransferRecords(ctx context.Context, startTime, endTime time.Time, direction string, ccy currency.Code, from, limit uint64) ([]*AccountTransferRecord, error) {
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
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, sAccountsTransferRecordsEPL, http.MethodGet, "/accounts/transfer", params, nil, &resp)
}

// GetAccountsTransferRecordByTransferID gets a transfer record of a user.
func (e *Exchange) GetAccountsTransferRecordByTransferID(ctx context.Context, transferID string) (*AccountTransferRecord, error) {
	if transferID == "" {
		return nil, errAccountIDRequired
	}
	var resp *AccountTransferRecord
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, sAccountsTransferRecordsEPL, http.MethodGet, "/accounts/transfer/"+transferID, nil, nil, &resp)
}

// GetFeeInfo retrieves fee rate for an account
func (e *Exchange) GetFeeInfo(ctx context.Context) (*FeeInfo, error) {
	var resp *FeeInfo
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, sFeeInfoEPL, http.MethodGet, "/feeinfo", nil, nil, &resp)
}

// GetInterestHistory gets a list of interest collection records of a user
func (e *Exchange) GetInterestHistory(ctx context.Context, startTime, endTime time.Time, direction string, from, limit uint64) ([]*InterestHistory, error) {
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
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, sInterestHistoryEPL, http.MethodGet, "/accounts/interest/history", params, nil, &resp)
}

// GetSubAccount gets a list of all the accounts within an Account Group for a user.
func (e *Exchange) GetSubAccount(ctx context.Context) ([]*SubAccount, error) {
	var resp []*SubAccount
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, sGetSubAccountEPL, http.MethodGet, "/subaccounts", nil, nil, &resp)
}

// GetSubAccountBalances retrieves balances by currency and account type (SPOT or FUTURES)
// for all accounts in the group. Available only to the primary user.
// Subaccounts should use GetBalances() for SPOT and the Futures API for FUTURES.
func (e *Exchange) GetSubAccountBalances(ctx context.Context) ([]*SubAccountBalances, error) {
	var resp []*SubAccountBalances
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, sGetSubAccountBalancesEPL, http.MethodGet, "/subaccounts/balances", nil, nil, &resp)
}

// GetSubAccountBalance gets balances information by currency and account type (SPOT and FUTURES) for a given external accountId in the account group
func (e *Exchange) GetSubAccountBalance(ctx context.Context, subAccountID string) ([]*SubAccountBalances, error) {
	if subAccountID == "" {
		return nil, fmt.Errorf("%w: empty subAccountID", errAccountIDRequired)
	}
	var resp []*SubAccountBalances
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, sGetSubAccountBalancesEPL, http.MethodGet, "/subaccounts/"+subAccountID+"/balances", nil, nil, &resp)
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
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, sGetSubAccountTransfersEPL, http.MethodPost, "/subaccounts/transfer", nil, arg, &resp)
}

// GetSubAccountTransferRecords gets a list of transfer records of a user. Max interval for start and end time is 6 months.
func (e *Exchange) GetSubAccountTransferRecords(ctx context.Context, arg *SubAccountTransferRecordRequest) ([]*SubAccountTransfer, error) {
	if !arg.StartTime.IsZero() && !arg.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(arg.StartTime, arg.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if !arg.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(arg.StartTime.UnixMilli(), 10))
	}
	if !arg.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(arg.EndTime.UnixMilli(), 10))
	}
	if !arg.Currency.IsEmpty() {
		params.Set("currency", arg.Currency.String())
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
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, sGetSubAccountTransfersEPL, http.MethodGet, "/subaccounts/transfer", params, nil, &resp)
}

// GetSubAccountTransferRecord retrieves a subaccount transfer record.
func (e *Exchange) GetSubAccountTransferRecord(ctx context.Context, id string) ([]*SubAccountTransfer, error) {
	if id == "" {
		return nil, fmt.Errorf("%w: subAccountID is missing", errAccountIDRequired)
	}
	var resp []*SubAccountTransfer
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, sGetSubAccountTransfersEPL, http.MethodGet, "/subaccounts/transfer/"+id, nil, nil, &resp)
}

// GetDepositAddresses gets all deposit addresses for a user.
func (e *Exchange) GetDepositAddresses(ctx context.Context, ccy currency.Code) (DepositAddresses, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	var addresses DepositAddresses
	return addresses, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, sGetDepositAddressesEPL, http.MethodGet, "/wallets/addresses", params, nil, &addresses)
}

// WalletActivity returns the wallet activity between set start and end time
func (e *Exchange) WalletActivity(ctx context.Context, start, end time.Time, activityType string) (*WalletActivity, error) {
	if err := common.StartEndTimeCheck(start, end); err != nil {
		return nil, err
	}
	values := url.Values{}
	values.Set("start", strconv.FormatInt(start.Unix(), 10))
	values.Set("end", strconv.FormatInt(end.Unix(), 10))
	if activityType != "" {
		values.Set("activityType", activityType)
	}
	var resp *WalletActivity
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, sGetWalletActivityRecordsEPL, http.MethodGet, "/wallets/activity", values, nil, &resp)
}

// NewCurrencyDepositAddress creates a new deposit address for a currency.
func (e *Exchange) NewCurrencyDepositAddress(ctx context.Context, ccy currency.Code) (string, error) {
	if ccy.IsEmpty() {
		return "", currency.ErrCurrencyCodeEmpty
	}
	var resp struct {
		Address string `json:"address"`
	}
	return resp.Address, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, sGetWalletAddressesEPL, http.MethodPost, "/wallets/address", nil, map[string]string{"currency": ccy.String()}, &resp)
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
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, sWithdrawCurrencyEPL, http.MethodPost, "/v2/wallets/withdraw", nil, arg, &resp)
}

// GetAccountMargin retrieves account margin information
func (e *Exchange) GetAccountMargin(ctx context.Context, accountType string) (*AccountMargin, error) {
	if accountType == "" {
		return nil, errAccountTypeRequired
	}
	params := url.Values{}
	params.Set("accountType", accountType)
	var resp *AccountMargin
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, sAccountMarginEPL, http.MethodGet, "/margin/accountMargin", params, nil, &resp)
}

// GetBorrowStatus retrieves borrow status of currencies
func (e *Exchange) GetBorrowStatus(ctx context.Context, ccy currency.Code) ([]*BorrowStatus, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	var resp []*BorrowStatus
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, sBorrowStatusEPL, http.MethodGet, "/margin/borrowStatus", params, nil, &resp)
}

// GetMarginBuySellAmounts gets the maximum and available margin buy/sell amount for a given symbol
func (e *Exchange) GetMarginBuySellAmounts(ctx context.Context, symbol currency.Pair) (*MarginBuySellAmount, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	var resp *MarginBuySellAmount
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, sMaxMarginSizeEPL, http.MethodGet, "/margin/maxSize", params, nil, &resp)
}

func validateOrderRequest(arg *PlaceOrderRequest) error {
	if arg.Symbol.IsEmpty() {
		return currency.ErrCurrencyPairEmpty
	}
	if arg.Side == "" {
		return fmt.Errorf("%w: %s", order.ErrSideIsInvalid, arg.Side)
	}
	isMarket := arg.Type == OrderType(order.Market) || arg.Type == OrderType(order.UnknownType)
	if !isMarket && arg.Price <= 0 {
		return fmt.Errorf("%w: price is required for non-market orders", limits.ErrPriceBelowMin)
	}
	if (arg.Type == OrderType(order.Limit) && arg.Quantity <= 0) ||
		(isMarket && strings.EqualFold(arg.Side, "SELL") && arg.Quantity <= 0) {
		return fmt.Errorf("%w: base quantity is required for market sell or limit orders", limits.ErrAmountBelowMin)
	}
	if isMarket && strings.EqualFold(arg.Side, "BUY") && arg.Amount <= 0 {
		return fmt.Errorf("%w: quote amount is required for market buy orders", limits.ErrAmountBelowMin)
	}
	return nil
}

// PlaceOrder places an order
func (e *Exchange) PlaceOrder(ctx context.Context, arg *PlaceOrderRequest) (*PlaceOrderResponse, error) {
	if err := validateOrderRequest(arg); err != nil {
		return nil, err
	}
	var resp PlaceOrderResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, sCreateOrderEPL, http.MethodPost, "/orders", nil, arg, &resp); err != nil {
		return nil, fmt.Errorf("%w: %w", order.ErrPlaceFailed, err)
	}
	return &resp, nil
}

// PlaceBatchOrders places a batch of orders
func (e *Exchange) PlaceBatchOrders(ctx context.Context, args []PlaceOrderRequest) ([]*PlaceBatchOrderItem, error) {
	if len(args) == 0 {
		return nil, common.ErrNilPointer
	}
	for x := range args {
		if err := validateOrderRequest(&args[x]); err != nil {
			return nil, err
		}
	}
	var resp []*PlaceBatchOrderItem
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, sBatchOrderEPL, http.MethodPost, "/orders/batch", nil, args, &resp)
}

// CancelReplaceOrder cancels an existing active order, new or partially filled, and places a new order
func (e *Exchange) CancelReplaceOrder(ctx context.Context, arg *CancelReplaceOrderRequest) (*CancelReplaceOrderResponse, error) {
	path, err := orderPath(arg.OrderID, "/orders/", arg.ClientOrderID, "/orders/cid:")
	if err != nil {
		return nil, err
	}
	var resp *CancelReplaceOrderResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, sCancelReplaceOrderEPL, http.MethodPut, path, nil, arg, &resp); err != nil {
		return nil, fmt.Errorf("%w: %w", order.ErrCancelFailed, err)
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
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, sGetOpenOrdersEPL, http.MethodGet, "/orders", params, nil, &resp)
}

// GetOrder gets order details by orderId or clientOrderId
func (e *Exchange) GetOrder(ctx context.Context, id, clientOrderID string) (*TradeOrder, error) {
	path, err := orderPath(id, "/orders/", clientOrderID, "/orders/cid:")
	if err != nil {
		return nil, err
	}
	var resp TradeOrder
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, sGetOpenOrderDetailEPL, http.MethodGet, path, nil, nil, &resp); err != nil {
		return nil, fmt.Errorf("%w: %w", order.ErrGetFailed, err)
	}
	return &resp, nil
}

// CancelOrderByID cancels an active order
func (e *Exchange) CancelOrderByID(ctx context.Context, id string) (*CancelOrderResponse, error) {
	if id == "" {
		return nil, fmt.Errorf("%w; order 'id' is required", order.ErrOrderIDNotSet)
	}
	var resp CancelOrderResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, sCancelOrderByIDEPL, http.MethodDelete, "/orders/"+id, nil, nil, &resp); err != nil {
		return nil, fmt.Errorf("%w: %w", order.ErrPlaceFailed, err)
	}
	return &resp, nil
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
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, sCancelBatchOrdersEPL, http.MethodDelete, "/orders/cancelByIds", nil, params, &resp)
}

// CancelTradeOrders batch cancel all orders in an account
func (e *Exchange) CancelTradeOrders(ctx context.Context, symbols []string, accountTypes []AccountType) ([]*CancelOrderResponse, error) {
	args := make(map[string]any)
	if len(symbols) != 0 {
		args["symbols"] = symbols
	}
	if len(accountTypes) > 0 {
		args["accountTypes"] = accountTypes
	}
	var resp []*CancelOrderResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, sCancelAllOrdersEPL, http.MethodDelete, "/orders", nil, args, &resp)
}

// KillSwitch set a timer that cancels all regular and smartorders after the timeout has expired.
// Timeout can be reset by calling this command again with a new timeout value.
// timeout value in seconds; range is 10 seconds to 10 minutes or 600 seconds
func (e *Exchange) KillSwitch(ctx context.Context, timeout time.Duration) (*KillSwitchStatus, error) {
	var timeoutString string
	if timeout < time.Second*10 || timeout > time.Minute*10 {
		return nil, fmt.Errorf("%w: must be between 10 seconds and 10 minutes", errInvalidTimeout)
	}
	timeoutString = strconv.FormatInt(int64(timeout.Seconds()), 10)
	var resp *KillSwitchStatus
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, sKillSwitchEPL, http.MethodPost, "/orders/killSwitch", nil, map[string]any{"timeout": timeoutString}, &resp)
}

// DisableKillSwitch disables the timer to cancels all regular and smartorders
func (e *Exchange) DisableKillSwitch(ctx context.Context) (*KillSwitchStatus, error) {
	var resp *KillSwitchStatus
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, sKillSwitchEPL, http.MethodPost, "/orders/killSwitch", nil, map[string]any{"timeout": "-1"}, &resp)
}

// GetKillSwitchStatus gets status of kill switch
func (e *Exchange) GetKillSwitchStatus(ctx context.Context) (*KillSwitchStatus, error) {
	var resp *KillSwitchStatus
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, sGetKillSwitchStatusEPL, http.MethodGet, "/orders/killSwitchStatus", nil, nil, &resp)
}

// CreateSmartOrder create a smart order for an account. Funds will only be frozen when the smart order triggers, not upon smart order creation
func (e *Exchange) CreateSmartOrder(ctx context.Context, arg *SmartOrderRequest) (*PlaceOrderResponse, error) {
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if arg.Side != order.Buy && arg.Side != order.Sell {
		return nil, order.ErrSideIsInvalid
	}
	if arg.Quantity <= 0 {
		return nil, fmt.Errorf("%w; base quantity is required", limits.ErrAmountBelowMin)
	}
	if arg.Type == OrderType(order.StopLimit) && arg.Price <= 0 {
		return nil, fmt.Errorf("%w %w", order.ErrPriceMustBeSetIfLimitOrder, limits.ErrPriceBelowMin)
	}
	if (order.Type(arg.Type)&order.TrailingStop == order.TrailingStop) && arg.TrailingOffset == "" {
		return nil, errInvalidTrailingOffset
	}
	if arg.Type == OrderType(order.TrailingStopLimit) && arg.LimitOffset == "" {
		return nil, errInvalidOffsetLimit
	}
	var resp *PlaceOrderResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, sCreateSmartOrdersEPL, http.MethodPost, "/smartorders", nil, arg, &resp); err != nil {
		return nil, fmt.Errorf("%w: %w", order.ErrPlaceFailed, err)
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
func (e *Exchange) CancelReplaceSmartOrder(ctx context.Context, arg *CancelReplaceSmartOrderRequest) (*CancelReplaceSmartOrder, error) {
	path, err := orderPath(arg.OrderID, "/smartorders/", arg.ClientOrderID, "/smartorders/cid:")
	if err != nil {
		return nil, err
	}
	var smartOrderResponse *CancelReplaceSmartOrder
	resp := &V3ResponseWrapper{
		Data: &smartOrderResponse,
	}
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, sCreateReplaceSmartOrdersEPL, http.MethodPut, path, nil, arg, &resp); err != nil {
		return nil, fmt.Errorf("%w: %w", order.ErrCancelFailed, err)
	}
	if smartOrderResponse == nil {
		return nil, order.ErrCancelFailed
	} else if smartOrderResponse.Code != 0 && smartOrderResponse.Code != 200 {
		ordID := arg.OrderID
		if ordID == "" {
			ordID = arg.ClientOrderID
		}
		return nil, fmt.Errorf("%w: order ID: %s code: %d message: %s", order.ErrCancelFailed, ordID, smartOrderResponse.Code, smartOrderResponse.Message)
	}
	return smartOrderResponse, nil
}

// GetSmartOpenOrders gets a list of pending smart orders for an account
func (e *Exchange) GetSmartOpenOrders(ctx context.Context, limit uint64, types []string) ([]*SmartOrder, error) {
	params := url.Values{}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if len(types) > 0 {
		params.Set("types", strings.Join(types, ","))
	}
	var resp []*SmartOrder
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, sGetSmartOrdersEPL, http.MethodGet, "/smartorders", params, nil, &resp)
}

// GetSmartOrderDetails retrieves a smart order's detail
func (e *Exchange) GetSmartOrderDetails(ctx context.Context, orderID, clientSuppliedID string) ([]*SmartOrderDetails, error) {
	path, err := orderPath(orderID, "/smartorders/", clientSuppliedID, "/smartorders/cid:")
	if err != nil {
		return nil, err
	}
	var smartOrders []*SmartOrderDetails
	resp := &V3ResponseWrapper{
		Data: &smartOrders,
	}
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, sSmartOrderDetailEPL, http.MethodGet, path, nil, nil, &resp); err != nil {
		return nil, fmt.Errorf("%w: %w", request.ErrAuthRequestFailed, err)
	}
	return smartOrders, nil
}

// CancelSmartOrderByID cancel a smart order by its id.
func (e *Exchange) CancelSmartOrderByID(ctx context.Context, id, clientSuppliedID string) (*CancelSmartOrderResponse, error) {
	path, err := orderPath(id, "/smartorders/", clientSuppliedID, "/smartorders/cid:")
	if err != nil {
		return nil, err
	}
	var cancelSmartOrderResponse *CancelSmartOrderResponse
	resp := &V3ResponseWrapper{
		Data: &cancelSmartOrderResponse,
	}
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, sCancelSmartOrderByIDEPL, http.MethodDelete, path, nil, nil, &resp); err != nil {
		return nil, fmt.Errorf("%w: %w", order.ErrCancelFailed, err)
	}
	if cancelSmartOrderResponse == nil {
		return nil, common.ErrNoResponse
	} else if cancelSmartOrderResponse.Code != 0 && cancelSmartOrderResponse.Code != 200 {
		ordID := id
		if ordID == "" {
			ordID = clientSuppliedID
		}
		return cancelSmartOrderResponse, fmt.Errorf("%w: order ID: %s code: %d message: %s", order.ErrCancelFailed, ordID, cancelSmartOrderResponse.Code, cancelSmartOrderResponse.Message)
	}
	return cancelSmartOrderResponse, nil
}

// CancelMultipleSmartOrders performs a batch cancel one or many smart orders in an account by IDs.
func (e *Exchange) CancelMultipleSmartOrders(ctx context.Context, args *CancelOrdersRequest) ([]*CancelOrderResponse, error) {
	if args == nil {
		return nil, common.ErrNilPointer
	}
	if len(args.ClientOrderIDs) == 0 && len(args.OrderIDs) == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	var cancelResponses []*CancelOrderResponse
	resp := &V3ResponseWrapper{
		Data: &cancelResponses,
	}
	return cancelResponses, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, sCancelSmartOrdersByIDEPL, http.MethodDelete, "/smartorders/cancelByIds", nil, args, &resp)
}

// CancelSmartOrders cancels all smart orders in an account.
func (e *Exchange) CancelSmartOrders(ctx context.Context, symbols []currency.Pair, accountTypes []AccountType, orderTypes []OrderType) ([]*CancelOrderResponse, error) {
	args := make(map[string]any)
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
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, sCancelAllSmartOrdersEPL, http.MethodDelete, "/smartorders", nil, args, &resp)
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
	if arg.Side != order.UnknownSide && arg.Side != order.AnySide {
		params.Set("side", arg.Side.String())
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
	}
	if !arg.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(arg.StartTime.UnixMilli(), 10))
	}
	if !arg.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(arg.EndTime.UnixMilli(), 10))
	}
	return params, nil
}

// GetOrdersHistory gets a list of historical orders in an account
func (e *Exchange) GetOrdersHistory(ctx context.Context, arg *OrdersHistoryRequest) ([]*TradeOrder, error) {
	params, err := orderFillParams(arg)
	if err != nil {
		return nil, err
	}
	var resp []*TradeOrder
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, sGetOrderHistoryEPL, http.MethodGet, "/orders/history", params, nil, &resp)
}

// GetSmartOrderHistory gets a list of historical smart orders in an account
func (e *Exchange) GetSmartOrderHistory(ctx context.Context, arg *OrdersHistoryRequest) ([]*SmartOrder, error) {
	params, err := orderFillParams(arg)
	if err != nil {
		return nil, err
	}
	var resp []*SmartOrder
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, sGetSmartOrderHistoryEPL, http.MethodGet, "/smartorders/history", params, nil, &resp)
}

// GetTradeHistory gets a list of all trades for an account
func (e *Exchange) GetTradeHistory(ctx context.Context, symbols currency.Pairs, direction string, from, limit uint64, startTime, endTime time.Time) ([]*TradeHistory, error) {
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
	if len(symbols) != 0 {
		params.Set("symbols", symbols.Join())
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
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, sGetTradesEPL, http.MethodGet, "/trades", params, nil, &resp)
}

// GetTradesByOrderID gets trades for an order
func (e *Exchange) GetTradesByOrderID(ctx context.Context, orderID string) ([]*TradeHistory, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp []*TradeHistory
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, sGetTradeDetailEPL, http.MethodGet, "/orders/"+orderID+"/trades", nil, nil, &resp)
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (e *Exchange) SendHTTPRequest(ctx context.Context, ep exchange.URL, epl request.EndpointLimit, path string, result any) error {
	endpoint, err := e.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	resp := result
	requiresWrapper := strings.HasPrefix(path, v3Path)
	if requiresWrapper {
		resp = &V3ResponseWrapper{
			Data: result,
		}
	}
	if err := e.SendPayload(ctx, epl, func() (*request.Item, error) {
		return &request.Item{
			Method:                 http.MethodGet,
			Path:                   endpoint + path,
			Result:                 resp,
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}, nil
	}, request.UnauthenticatedRequest); err != nil {
		return err
	}
	if result == nil {
		return common.ErrNoResponse
	}
	if requiresWrapper {
		if val, ok := resp.(*V3ResponseWrapper); ok {
			if val.Code != 0 && val.Code != 200 {
				return fmt.Errorf("code: %d message: %s", val.Code, val.Message)
			}
		}
	}
	if errType, ok := result.(interface{ Error() error }); ok {
		return errType.Error()
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
	requiresWrapper := strings.HasPrefix(path, v3Path)
	if requiresWrapper {
		resp = &V3ResponseWrapper{
			Data: result,
		}
	}
	requestFunc := func() (*request.Item, error) {
		headers := make(map[string]string)
		headers["Content-Type"] = "application/json"
		headers["key"] = creds.Key
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
		values.Del("signTimestamp") // The signature timestamp has been removed from the query string as it is now included in the request header.

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
				return fmt.Errorf("%w code: %d message: %s", request.ErrAuthRequestFailed, val.Code, val.Message)
			}
		}
	}
	if errType, ok := result.(interface{ Error() error }); ok {
		return errType.Error()
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
