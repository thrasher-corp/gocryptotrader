package gateio

import (
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const (
	gateioTradeURL                      = "https://api.gateio.ws"
	gateioFuturesTestnetTrading         = "https://fx-api-testnet.gateio.ws"
	gateioFuturesLiveTradingAlternative = "https://fx-api.gateio.ws"
	gateioAPIVersion                    = "/api/v4/"
	tradeBaseURL                        = "https://www.gate.io"

	// SubAccount Endpoints
	subAccounts = "sub_accounts"

	// Spot
	gateioSpotCurrencies  = "spot/currencies"
	gateioSpotOrders      = "spot/orders"
	gateioSpotPriceOrders = "spot/price_orders"

	// Wallets
	walletSubAccountTransfer = "/wallet/sub_account_transfers"

	// Margin
	gateioMarginCurrencyPairs   = "margin/currency_pairs"
	gateioMarginLoans           = "margin/loans"
	gateioMarginLoanRecords     = "margin/loan_records"
	gateioMarginAutoRepay       = "margin/auto_repay"
	gateioCrossMarginCurrencies = "margin/cross/currencies"
	gateioCrossMarginLoans      = "margin/cross/loans"
	gateioCrossMarginRepayments = "margin/cross/repayments"

	// Options
	gateioOptionContracts  = "options/contracts"
	gateioOptionSettlement = "options/settlements"
	gateioOptionsPosition  = "options/positions"
	gateioOptionsOrders    = "options/orders"

	// Flash Swap
	gateioFlashSwapOrders = "flash_swap/orders"

	futuresPath      = "futures/"
	deliveryPath     = "delivery/"
	ordersPath       = "orders"
	positionsPath    = "positions/"
	hedgeModePath    = "dual_comp/positions/"
	subAccountsPath  = "sub_accounts/"
	priceOrdersPaths = "price_orders"

	// Withdrawals
	withdrawal = "/withdrawals"
)

const (
	utc0TimeZone = "utc0"
	utc8TimeZone = "utc8"
)

var (
	errInvalidOrderText          = errors.New("invalid text value, requires prefix `t-`")
	errLoanTypeIsRequired        = errors.New("loan type is required")
	errUserIDRequired            = errors.New("user id is required")
	errSTPGroupNameRequired      = errors.New("self-trade prevention group name required")
	errSTPGroupIDRequired        = errors.New("self-trade prevention group id required")
	errPlanIDRequired            = errors.New("plan ID required")
	errInvalidCurrencyChain      = errors.New("name of the chain used for withdrawal must be specified")
	errNoValidResponseFromServer = errors.New("no valid response from server")
	errInvalidUnderlying         = errors.New("missing underlying")
	errInvalidOrderSize          = errors.New("invalid order size")
	errInvalidSubAccount         = errors.New("invalid or empty subaccount")
	errInvalidTransferDirection  = errors.New("invalid transfer direction")
	errDifferentAccount          = errors.New("account type must be identical for all orders")
	errNoValidParameterPassed    = errors.New("no valid parameter passed")
	errInvalidCountdown          = errors.New("invalid countdown, Countdown time, in seconds At least 5 seconds, 0 means cancel the countdown")
	errInvalidOrderStatus        = errors.New("invalid order status")
	errInvalidLoanSide           = errors.New("invalid loan side, only 'lend' and 'borrow'")
	errLoanRateIsRequired        = errors.New("loan rate is required for borrow side")
	errInvalidLoanID             = errors.New("missing loan ID")
	errInvalidRepayMode          = errors.New("invalid repay mode specified, must be 'all' or 'partial'")
	errMissingPreviewID          = errors.New("missing required parameter: preview_id")
	errChangeHasToBePositive     = errors.New("change has to be positive")
	errInvalidAutoSize           = errors.New("invalid autoSize")
	errTooManyOrderRequest       = errors.New("too many order creation request")
	errInvalidTimeout            = errors.New("invalid timeout, should be in seconds At least 5 seconds, 0 means cancel the countdown")
	errNoTickerData              = errors.New("no ticker data available")
	errInvalidTimezone           = errors.New("invalid timezone")
	errMultipleOrders            = errors.New("multiple orders passed")
	errMissingWithdrawalID       = errors.New("missing withdrawal ID")
	errInvalidSubAccountUserID   = errors.New("sub-account user id is required")
	errInvalidSettlementQuote    = errors.New("symbol quote currency does not match asset settlement currency")
	errInvalidSettlementBase     = errors.New("symbol base currency does not match asset settlement currency")
	errMissingAPIKey             = errors.New("missing API key information")
	errSingleAssetRequired       = errors.New("single asset type required")
	errMissingUnifiedAccountMode = errors.New("unified account mode is required")
	errTooManyCurrencyCodes      = errors.New("too many currency codes supplied")
	errFetchingOrderbook         = errors.New("error fetching orderbook")
	errNoSpotInstrument          = errors.New("no spot instrument available")
	errOperationTypeRequired     = errors.New("operation type required")
)

// validTimesInForce holds a list of supported time-in-force values and corresponding string representations.
// slice iteration outperforms map with this few elements
var validTimesInForce = []struct {
	String      string
	TimeInForce order.TimeInForce
}{
	{gtcTIF, order.GoodTillCancel}, {iocTIF, order.ImmediateOrCancel}, {pocTIF, order.PostOnly}, {fokTIF, order.FillOrKill},
}

func timeInForceFromString(tif string) (order.TimeInForce, error) {
	for a := range validTimesInForce {
		if validTimesInForce[a].String == tif {
			return validTimesInForce[a].TimeInForce, nil
		}
	}
	return order.UnknownTIF, fmt.Errorf("%w: %q", order.ErrUnsupportedTimeInForce, tif)
}

// Exchange implements exchange.IBotExchange and contains additional specific api methods for interacting with GateIO
type Exchange struct {
	exchange.Base

	messageIDSeq  common.Counter
	wsOBUpdateMgr *wsOBUpdateManager
	wsOBResubMgr  *wsOBResubManager
}

// ***************************************** SubAccounts ********************************

// CreateNewSubAccount creates a new sub-account
func (e *Exchange) CreateNewSubAccount(ctx context.Context, arg *SubAccountParams) (*SubAccount, error) {
	if arg.SubAccountName == "" {
		return nil, errInvalidSubAccount
	}
	var response *SubAccount
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, subAccountEPL, http.MethodPost, subAccounts, nil, &arg, &response)
}

// GetSubAccounts retrieves list of sub-accounts for given account
func (e *Exchange) GetSubAccounts(ctx context.Context) ([]*SubAccount, error) {
	var response []*SubAccount
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, subAccountEPL, http.MethodGet, subAccounts, nil, nil, &response)
}

// GetSingleSubAccount retrieves a single sub-account for given account
func (e *Exchange) GetSingleSubAccount(ctx context.Context, userID string) (*SubAccount, error) {
	if userID == "" {
		return nil, errInvalidSubAccount
	}
	var response *SubAccount
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, subAccountEPL, http.MethodGet, subAccounts+"/"+userID, nil, nil, &response)
}

// CreateAPIKeysOfSubAccount creates a sub-account for the sub-account
//
// name: Permission name (all permissions will be removed if no value is passed)
// >> wallet: wallet, spot: spot/margin, futures: perpetual contract, delivery: delivery, earn: earn, options: options
func (e *Exchange) CreateAPIKeysOfSubAccount(ctx context.Context, arg *CreateAPIKeySubAccountParams) (*APIDetailResponse, error) {
	if arg.SubAccountUserID == 0 {
		return nil, errInvalidSubAccountUserID
	}
	if arg.Body == nil {
		return nil, errors.New("sub-account key information is required")
	}
	var resp *APIDetailResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, subAccountEPL, http.MethodPost, subAccountsPath+strconv.FormatUint(arg.SubAccountUserID, 10)+"/keys", nil, &arg, &resp)
}

// GetAllAPIKeyOfSubAccount list all API Key of the sub-account
func (e *Exchange) GetAllAPIKeyOfSubAccount(ctx context.Context, userID uint64) ([]*APIDetailResponse, error) {
	if userID == 0 {
		return nil, errUserIDRequired
	}
	var resp []*APIDetailResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, subAccountEPL, http.MethodGet, subAccountsPath+strconv.FormatUint(userID, 10)+"/keys", nil, nil, &resp)
}

// UpdateAPIKeyOfSubAccount update API key of the sub-account
func (e *Exchange) UpdateAPIKeyOfSubAccount(ctx context.Context, subAccountAPIKey string, arg CreateAPIKeySubAccountParams) error {
	if arg.SubAccountUserID == 0 {
		return errInvalidSubAccountUserID
	}
	if subAccountAPIKey == "" {
		return errMissingAPIKey
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, subAccountEPL, http.MethodPut, subAccountsPath+strconv.FormatUint(arg.SubAccountUserID, 10)+"/keys/"+subAccountAPIKey, nil, &arg, nil)
}

// DeleteSubAccountAPIKeyPair deletes a subaccount API key pair
func (e *Exchange) DeleteSubAccountAPIKeyPair(ctx context.Context, subAccountUserID int64, subAccountAPIKey string) error {
	if subAccountUserID == 0 {
		return errInvalidSubAccountUserID
	}
	if subAccountAPIKey == "" {
		return errMissingAPIKey
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, subAccountEPL, http.MethodDelete, subAccountsPath+strconv.FormatInt(subAccountUserID, 10)+"/keys/"+subAccountAPIKey, nil, nil, nil)
}

// GetAPIKeyOfSubAccount retrieves the API Key of the sub-account
func (e *Exchange) GetAPIKeyOfSubAccount(ctx context.Context, subAccountUserID int64, apiKey string) (*APIDetailResponse, error) {
	if subAccountUserID == 0 {
		return nil, errInvalidSubAccountUserID
	}
	if apiKey == "" {
		return nil, errMissingAPIKey
	}
	var resp *APIDetailResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, subAccountEPL, http.MethodGet, subAccountsPath+strconv.FormatInt(subAccountUserID, 10)+"/keys/"+apiKey, nil, nil, &resp)
}

// LockSubAccount locks the sub-account
func (e *Exchange) LockSubAccount(ctx context.Context, subAccountUserID int64) error {
	if subAccountUserID == 0 {
		return errInvalidSubAccountUserID
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, subAccountEPL, http.MethodPost, subAccountsPath+strconv.FormatInt(subAccountUserID, 10)+"/lock", nil, nil, nil)
}

// UnlockSubAccount locks the sub-account
func (e *Exchange) UnlockSubAccount(ctx context.Context, subAccountUserID int64) error {
	if subAccountUserID == 0 {
		return errInvalidSubAccountUserID
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, subAccountEPL, http.MethodPost, subAccountsPath+strconv.FormatInt(subAccountUserID, 10)+"/unlock", nil, nil, nil)
}

// GetSubAccountMode retrieves sub-account mode
// Unified account mode:
//
//	classic: Classic account mode
//	multi_currency: Multi-currency margin mode
//	portfolio: Portfolio margin mode
func (e *Exchange) GetSubAccountMode(ctx context.Context) ([]*SubAccountMode, error) {
	var resp []*SubAccountMode
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodGet, "sub_accounts/unified_mode", nil, nil, &resp)
}

// *****************************************  Spot **************************************

// ListSpotCurrencies to retrieve detailed list of each currency.
func (e *Exchange) ListSpotCurrencies(ctx context.Context) ([]*CurrencyInfo, error) {
	var resp []*CurrencyInfo
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, publicCurrenciesSpotEPL, gateioSpotCurrencies, &resp)
}

// GetCurrencyDetail details of a specific currency.
func (e *Exchange) GetCurrencyDetail(ctx context.Context, ccy currency.Code) (*CurrencyInfo, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp *CurrencyInfo
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, publicCurrenciesSpotEPL, gateioSpotCurrencies+"/"+ccy.String(), &resp)
}

// ListSpotCurrencyPairs retrieve all currency pairs supported by the exchange.
func (e *Exchange) ListSpotCurrencyPairs(ctx context.Context) ([]*CurrencyPairDetail, error) {
	var resp []*CurrencyPairDetail
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, publicListCurrencyPairsSpotEPL, "spot/currency_pairs", &resp)
}

// GetCurrencyPairDetail to get details of a specific order for spot/margin accounts.
func (e *Exchange) GetCurrencyPairDetail(ctx context.Context, currencyPair currency.Pair) (*CurrencyPairDetail, error) {
	if currencyPair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var resp *CurrencyPairDetail
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, publicCurrencyPairDetailSpotEPL, "spot/currency_pairs/"+currencyPair.String(), &resp)
}

// GetTickers retrieve ticker information
// Return only related data if currency_pair is specified; otherwise return all of them
func (e *Exchange) GetTickers(ctx context.Context, currencyPair currency.Pair, timezone string) ([]*Ticker, error) {
	params := url.Values{}
	if !currencyPair.IsEmpty() {
		params.Set("currency_pair", currencyPair.String())
	}
	if timezone != "" && timezone != utc8TimeZone && timezone != utc0TimeZone {
		return nil, errInvalidTimezone
	} else if timezone != "" {
		params.Set("timezone", timezone)
	}
	var tickers []*Ticker
	return tickers, e.SendHTTPRequest(ctx, exchange.RestSpot, publicTickersSpotEPL, common.EncodeURLValues("spot/tickers", params), &tickers)
}

// GetTicker retrieves a single ticker information for a currency pair.
func (e *Exchange) GetTicker(ctx context.Context, currencyPair currency.Pair, timezone string) (*Ticker, error) {
	if currencyPair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	tickers, err := e.GetTickers(ctx, currencyPair, timezone)
	if err != nil {
		return nil, err
	}
	if len(tickers) > 0 {
		return tickers[0], err
	}
	return nil, fmt.Errorf("no ticker data found for currency pair %v", currencyPair)
}

var intervalAndStringRepresentations = []*struct {
	Interval kline.Interval
	String   string
}{
	{kline.TenMilliseconds, "10ms"},
	{kline.TwentyMilliseconds, "20ms"},
	{kline.HundredMilliseconds, "100ms"},
	{kline.TwoHundredAndFiftyMilliseconds, "250ms"},
	{kline.ThousandMilliseconds, "1000ms"},
	{kline.TenSecond, "10s"},
	{kline.ThirtySecond, "30s"},
	{kline.OneMin, "1m"},
	{kline.FiveMin, "5m"},
	{kline.FifteenMin, "15m"},
	{kline.ThirtyMin, "30m"},
	{kline.OneHour, "1h"},
	{kline.TwoHour, "2h"},
	{kline.FourHour, "4h"},
	{kline.EightHour, "8h"},
	{kline.TwelveHour, "12h"},
	{kline.OneDay, "1d"},
	{kline.SevenDay, "7d"},
	{kline.OneMonth, "30d"},
}

// getIntervalString returns a string representation of the interval according to the Gateio exchange representation
func getIntervalString(interval kline.Interval) (string, error) {
	for _, result := range intervalAndStringRepresentations {
		if result.Interval == interval {
			return result.String, nil
		}
	}
	return "", fmt.Errorf("%q: %w", interval.String(), kline.ErrUnsupportedInterval)
}

// GetOrderbook returns the orderbook data for a suppled currency pair
func (e *Exchange) GetOrderbook(ctx context.Context, pairString currency.Pair, interval string, limit uint64, withOrderbookID bool) (*Orderbook, error) {
	if pairString.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	if interval != "" {
		params.Set("interval", interval)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	params.Set("currency_pair", pairString.String())
	params.Set("with_id", strconv.FormatBool(withOrderbookID))
	var response *OrderbookData
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, publicOrderbookSpotEPL, common.EncodeURLValues("spot/order_book", params), &response); err != nil {
		return nil, err
	}
	return response.MakeOrderbook(), nil
}

// GetMarketTrades retrieve market trades
func (e *Exchange) GetMarketTrades(ctx context.Context, pairString currency.Pair, limit uint64, lastID string, reverse bool, from, to time.Time, page uint64) ([]*Trade, error) {
	if !from.IsZero() && !to.IsZero() {
		if err := common.StartEndTimeCheck(from, to); err != nil {
			return nil, err
		}
	}
	if pairString.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("currency_pair", pairString.String())
	if lastID != "" {
		params.Set("last_id", lastID)
	}
	if reverse {
		params.Set("reverse", strconv.FormatBool(reverse))
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	if page != 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var response []*Trade
	return response, e.SendHTTPRequest(ctx, exchange.RestSpot, publicMarketTradesSpotEPL, common.EncodeURLValues("spot/trades", params), &response)
}

// GetCandlesticks retrieves market candlesticks.
func (e *Exchange) GetCandlesticks(ctx context.Context, currencyPair currency.Pair, limit uint64, from, to time.Time, interval kline.Interval) ([]*Candlestick, error) {
	if currencyPair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if !from.IsZero() && !to.IsZero() {
		if err := common.StartEndTimeCheck(from, to); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if interval.Duration().Microseconds() != 0 {
		var intervalString string
		intervalString, err := getIntervalString(interval)
		if err != nil {
			return nil, err
		}
		params.Set("interval", intervalString)
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	params.Set("currency_pair", currencyPair.String())
	var candles []*Candlestick
	return candles, e.SendHTTPRequest(ctx, exchange.RestSpot, publicCandleStickSpotEPL, common.EncodeURLValues("spot/candlesticks", params), &candles)
}

// GetTradingFeeRatio retrieves user trading fee rates
func (e *Exchange) GetTradingFeeRatio(ctx context.Context, currencyPair currency.Pair) (*SpotTradingFeeRate, error) {
	params := url.Values{}
	if currencyPair.IsPopulated() {
		// specify a currency pair to retrieve precise fee rate
		params.Set("currency_pair", currencyPair.String())
	}
	var response *SpotTradingFeeRate
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotTradingFeeEPL, http.MethodGet, "spot/fee", params, nil, &response)
}

// GetAccountBatchFeeRates retrieves account fee rates
// Maximum 50 currency pairs per request
func (e *Exchange) GetAccountBatchFeeRates(ctx context.Context, currencyPairs []string) (map[string]*SpotTradingFeeRate, error) {
	if len(currencyPairs) == 0 {
		return nil, currency.ErrCurrencyPairsEmpty
	}
	params := url.Values{}
	params.Set("currency_pairs", strings.Join(currencyPairs, ","))
	var resp map[string]*SpotTradingFeeRate // map of currency pair string to trading fee rate detail.
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotTradingFeeEPL, http.MethodGet, "spot/batch_fee", params, nil, &resp)
}

// GetSpotAccounts retrieves spot account.
func (e *Exchange) GetSpotAccounts(ctx context.Context, ccy currency.Code) ([]*SpotAccount, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	var response []*SpotAccount
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotAccountsEPL, http.MethodGet, "spot/accounts", params, nil, &response)
}

// GetUnifiedAccount retrieves unified account.
func (e *Exchange) GetUnifiedAccount(ctx context.Context, ccy currency.Code, subAccountUserID string) (*UnifiedUserAccount, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	if subAccountUserID != "" {
		params.Set("sub_uid", subAccountUserID)
	}
	var resp *UnifiedUserAccount
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, privateUnifiedSpotEPL, http.MethodGet, "unified/accounts", params, nil, &resp)
}

// GetMaximumBorrowableAmountUnifiedAccount query maximum borrowable amount for unified account
func (e *Exchange) GetMaximumBorrowableAmountUnifiedAccount(ctx context.Context, ccy currency.Code) (*CurrencyAndAmount, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	var resp *CurrencyAndAmount
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, privateUnifiedSpotEPL, http.MethodGet, "unified/borrowable", params, nil, &resp)
}

// GetUnifiedAccountMaximumTransferableAmount query maximum transferable amount for unified account
func (e *Exchange) GetUnifiedAccountMaximumTransferableAmount(ctx context.Context, ccy currency.Code) (*CurrencyAndAmount, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	var resp *CurrencyAndAmount
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, privateUnifiedSpotEPL, http.MethodGet, "unified/transferable", params, nil, &resp)
}

// GetMultipleTransferableAmountForUnifiedAccounts batch query maximum transferable amount for unified accounts. Each currency shows the maximum value. After user withdrawal, the transferable amount for all currencies will change
func (e *Exchange) GetMultipleTransferableAmountForUnifiedAccounts(ctx context.Context, currencies ...currency.Code) ([]*CurrencyAndAmount, error) {
	if len(currencies) == 0 {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	currenciesString := make([]string, len(currencies))
	for x := range currencies {
		if currencies[x].IsEmpty() {
			return nil, currency.ErrCurrencyCodeEmpty
		}
		currenciesString[x] = currencies[x].String()
	}
	params := url.Values{}
	params.Set("currencies", strings.Join(currenciesString, ","))
	var resp []*CurrencyAndAmount
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, privateUnifiedSpotEPL, http.MethodGet, "unified/transferables", params, nil, &resp)
}

// GetBatchUnifiedAccountMaximumBorrowableAmount batch query unified account maximum borrowable amount
func (e *Exchange) GetBatchUnifiedAccountMaximumBorrowableAmount(ctx context.Context, currencies ...currency.Code) ([]*CurrencyAndAmount, error) {
	if len(currencies) == 0 {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	currenciesString := make([]string, len(currencies))
	for x := range currencies {
		if currencies[x].IsEmpty() {
			return nil, currency.ErrCurrencyCodeEmpty
		}
		currenciesString[x] = currencies[x].String()
	}
	params := url.Values{}
	params.Set("currencies", strings.Join(currenciesString, ","))
	var resp []*CurrencyAndAmount
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, privateUnifiedSpotEPL, http.MethodGet, "unified/batch_borrowable", params, nil, &resp)
}

// BorrowOrRepay borrow or repay an asset in unified account
// When borrowing, ensure the borrowed amount is not below the minimum borrowing threshold for the specific cryptocurrency and does not exceed the maximum borrowing limit set by the platform and user.
// Loan interest will be automatically deducted from the account at regular intervals. Users are responsible for managing repayment of borrowed amounts.
// For repayment, use repaid_all=true to repay all available amounts
func (e *Exchange) BorrowOrRepay(ctx context.Context, arg *BorrowOrRepayParams) (string, error) {
	if arg.Currency.IsEmpty() {
		return "", currency.ErrCurrencyCodeEmpty
	}
	if arg.Type == "" {
		return "", errLoanTypeIsRequired
	}
	if arg.Amount <= 0 {
		return "", fmt.Errorf("%w: borrow or repay amount is required", order.ErrAmountIsInvalid)
	}
	var resp struct {
		TransactionID string `json:"tran_id"`
	}
	return resp.TransactionID, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, privateUnifiedSpotEPL, http.MethodPost, "unified/loans", nil, arg, &resp)
}

// GetLoans retrieves a list of borrow or repay loan actions detail
func (e *Exchange) GetLoans(ctx context.Context, ccy currency.Code, loanType string, page, limit uint64) ([]*LoadDetail, error) {
	return e.getLoans(ctx, ccy, loanType, "unified/loans", page, limit)
}

// GetLoanRecords retrieves a borrow and repay loan types records
func (e *Exchange) GetLoanRecords(ctx context.Context, ccy currency.Code, loanType string, page, limit uint64) ([]*LoadDetail, error) {
	return e.getLoans(ctx, ccy, loanType, "unified/loan_records", page, limit)
}

func (e *Exchange) getLoans(ctx context.Context, ccy currency.Code, loanType, path string, page, limit uint64) ([]*LoadDetail, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if loanType != "" {
		params.Set("type", loanType)
	}
	var resp []*LoadDetail
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, path, params, nil, &resp)
}

// GetInterestDeductionRecords retrieves interest deduction records
func (e *Exchange) GetInterestDeductionRecords(ctx context.Context, ccy currency.Code, page, limit uint64, startTime, endTime time.Time, loanType string) ([]*InterestDeductionRecord, error) {
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	if !startTime.IsZero() {
		params.Set("from", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("to", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if loanType != "" {
		params.Set("type", loanType)
	}
	var resp []*InterestDeductionRecord
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, "unified/interest_records", params, nil, &resp)
}

// GetUserRiskUnitDetails holds a user risk unit details
func (e *Exchange) GetUserRiskUnitDetails(ctx context.Context) (*UserRiskUnitDetail, error) {
	var resp *UserRiskUnitDetail
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, "unified/risk_units", nil, nil, &resp)
}

// SetUnifiedAccountMode sets unified account mode
// Possible unified account modes:
// "classic": Classic account mode
// "multi_currency": Cross-currency margin mode
// "portfolio": Portfolio margin mode
// "single_currency": Single-currency margin mode
func (e *Exchange) SetUnifiedAccountMode(ctx context.Context, arg *UnifiedAccountMode) error {
	if err := common.NilGuard(arg); err != nil {
		return err
	}
	if arg.Mode == "" {
		return errMissingUnifiedAccountMode
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodPut, "unified/unified_mode", nil, arg, nil)
}

// GetUnifiedAccountMode query mode of the unified account
func (e *Exchange) GetUnifiedAccountMode(ctx context.Context) (*UnifiedAccountMode, error) {
	var resp *UnifiedAccountMode
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, "unified/unified_mode", nil, nil, &resp)
}

// GetUnifiedAccountEstimatedInterestRate retrieves unified account estimated interest rate
// Interest rates fluctuate hourly based on lending depth, so exact rates cannot be provided. When a currency is not supported, the interest rate returned will be an empty string
func (e *Exchange) GetUnifiedAccountEstimatedInterestRate(ctx context.Context, currencies []string) (map[currency.Code]types.Number, error) {
	if len(currencies) == 0 || slices.Contains(currencies, "") {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currencies", strings.Join(currencies, ","))
	var resp map[currency.Code]types.Number
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, "unified/estimate_rate", params, nil, &resp)
}

// GetUnifiedAccountTiered retrieves unified account tiered
func (e *Exchange) GetUnifiedAccountTiered(ctx context.Context) ([]*UnifiedAccountTieredDetail, error) {
	var resp []*UnifiedAccountTieredDetail
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, "unified/currency_discount_tiers", nil, nil, &resp)
}

// GetUnifiedAccountTieredLoanMargin query unified account tiered loan margin
func (e *Exchange) GetUnifiedAccountTieredLoanMargin(ctx context.Context) ([]*UnifiedAccountLoanMargin, error) {
	var resp []*UnifiedAccountLoanMargin
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, "unified/loan_margin_tiers", nil, nil, &resp)
}

// CalculatePortfolioMargin portfolio margin calculator
func (e *Exchange) CalculatePortfolioMargin(ctx context.Context, arg *PortfolioMarginCalculatorParams) (*PortfolioMarginCalculationResponse, error) {
	for _, sb := range arg.SpotBalances {
		if sb.Currency.IsEmpty() {
			return nil, currency.ErrCurrencyCodeEmpty
		}
		if sb.Equity <= 0 {
			return nil, fmt.Errorf("%w: equity must be greater than 0", errInvalidOrderSize)
		}
	}
	for _, so := range arg.SpotOrders {
		if len(so.CurrencyPairs) == 0 {
			return nil, currency.ErrCurrencyPairEmpty
		}
		if so.OrderPrice <= 0 {
			return nil, fmt.Errorf("%w: order price must be greater than 0", limits.ErrPriceBelowMin)
		}
		if so.Left <= 0 {
			return nil, fmt.Errorf("%w: left, unfilled quantity size must be greater than 0", errInvalidOrderSize)
		}
		if so.Type == order.UnknownType {
			return nil, fmt.Errorf("%w: order type is required", order.ErrTypeIsInvalid)
		}
	}
	for _, fo := range arg.FuturesOrders {
		if fo.Contract.IsEmpty() {
			return nil, currency.ErrCurrencyPairEmpty
		}
		if fo.Size <= 0 {
			return nil, fmt.Errorf("%w: size must be greater than 0", errInvalidOrderSize)
		}
		if fo.Left <= 0 {
			return nil, fmt.Errorf("%w: left, unfilled quantity size must be greater than 0", errInvalidOrderSize)
		}
	}
	for _, op := range arg.OptionsPositions {
		if op.OptionsName.IsEmpty() {
			return nil, currency.ErrCurrencyPairEmpty
		}
		if op.Size <= 0 {
			return nil, fmt.Errorf("%w: size must be greater than 0", errInvalidOrderSize)
		}
	}
	for _, oo := range arg.OptionsOrders {
		if oo.OptionsName.IsEmpty() {
			return nil, currency.ErrCurrencyPairEmpty
		}
		if oo.Size <= 0 {
			return nil, fmt.Errorf("%w: size must be greater than 0", errInvalidOrderSize)
		}
		if oo.Left <= 0 {
			return nil, fmt.Errorf("%w: left, unfilled quantity size must be greater than 0", errInvalidOrderSize)
		}
	}
	var resp *PortfolioMarginCalculationResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodPost, "unified/portfolio_calculator", nil, arg, &resp)
}

// CreateBatchOrders Create a batch of orders Batch orders requirements: custom order field text is required At most 4 currency pairs,
// maximum 10 orders each, are allowed in one request No mixture of spot orders and margin orders, i.e. account must be identical for all orders
func (e *Exchange) CreateBatchOrders(ctx context.Context, args []CreateOrderRequest) ([]*SpotOrder, error) {
	if len(args) > 10 {
		return nil, fmt.Errorf("%w only 10 orders are canceled at once", errMultipleOrders)
	}
	for x := range args {
		if (x != 0) && args[x-1].Account != args[x].Account {
			return nil, errDifferentAccount
		}
		if args[x].Text == "" {
			return nil, order.ErrClientOrderIDMustBeSet
		} else if !strings.HasPrefix(args[x].Text, "t-") {
			return nil, errInvalidOrderText
		}
		if args[x].CurrencyPair.IsEmpty() {
			return nil, currency.ErrCurrencyPairEmpty
		}
		if args[x].Side != order.Buy && args[x].Side != order.Sell {
			return nil, order.ErrSideIsInvalid
		}
		if args[x].Account != asset.Spot &&
			args[x].Account != asset.CrossMargin &&
			args[x].Account != asset.Margin {
			return nil, fmt.Errorf("%w: only spot, margin, and cross_margin area allowed", asset.ErrInvalidAsset)
		}
		if args[x].Amount <= 0 {
			return nil, order.ErrAmountIsInvalid
		}
		if args[x].Price <= 0 {
			return nil, limits.ErrPriceBelowMin
		}
	}
	var response []*SpotOrder
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotBatchOrdersEPL, http.MethodPost, "spot/batch_orders", nil, &args, &response)
}

// GetSpotOpenOrders retrieves all open orders
// List open orders in all currency pairs.
// Note that pagination parameters affect record number in each currency pair's open order list. No pagination is applied to the number of currency pairs returned. All currency pairs with open orders will be returned.
// Spot and margin orders are returned by default. To list cross margin orders, account must be set to cross_margin
func (e *Exchange) GetSpotOpenOrders(ctx context.Context, page, limit uint64, isCrossMargin bool) ([]*SpotOrdersDetail, error) {
	params := url.Values{}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if isCrossMargin {
		params.Set("account", asset.CrossMargin.String())
	}
	var response []*SpotOrdersDetail
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotGetOpenOrdersEPL, http.MethodGet, "spot/open_orders", params, nil, &response)
}

// SpotClosePositionWhenCrossCurrencyDisabled set close position when cross-currency is disabled
func (e *Exchange) SpotClosePositionWhenCrossCurrencyDisabled(ctx context.Context, arg *ClosePositionRequestParam) (*SpotOrder, error) {
	if arg.CurrencyPair.IsInvalid() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if arg.Amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	if arg.Price <= 0 {
		return nil, limits.ErrPriceBelowMin
	}
	var response *SpotOrder
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotClosePositionEPL, http.MethodPost, "spot/cross_liquidate_orders", nil, &arg, &response)
}

// PlaceSpotOrder creates a spot order you can place orders with spot, margin or cross margin account through setting the accountfield.
// It defaults to spot, which means spot account is used to place orders.
func (e *Exchange) PlaceSpotOrder(ctx context.Context, arg *CreateOrderRequest) (*SpotOrder, error) {
	if arg.CurrencyPair.IsInvalid() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if arg.Side != order.Buy && arg.Side != order.Sell {
		return nil, order.ErrSideIsInvalid
	}
	if arg.Account != asset.Spot &&
		arg.Account != asset.CrossMargin &&
		arg.Account != asset.Margin {
		return nil, fmt.Errorf("%w: only 'spot', 'cross_margin', and 'margin' area allowed", asset.ErrInvalidAsset)
	}
	if arg.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if arg.Price < 0 {
		return nil, limits.ErrPriceBelowMin
	}
	var response *SpotOrder
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotPlaceOrderEPL, http.MethodPost, gateioSpotOrders, nil, &arg, &response)
}

// GetSpotOrders retrieves spot orders.
func (e *Exchange) GetSpotOrders(ctx context.Context, currencyPair currency.Pair, status string, page, limit uint64) ([]*SpotOrder, error) {
	if currencyPair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	if status != "" {
		params.Set("status", status)
	}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	params.Set("currency_pair", currencyPair.String())
	var response []*SpotOrder
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotGetOrdersEPL, http.MethodGet, gateioSpotOrders, params, nil, &response)
}

// CancelAllOpenOrdersSpecifiedCurrencyPair cancel all open orders in specified currency pair
func (e *Exchange) CancelAllOpenOrdersSpecifiedCurrencyPair(ctx context.Context, currencyPair currency.Pair, side order.Side, a asset.Item) ([]*SpotOrder, error) {
	if currencyPair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("currency_pair", currencyPair.String())
	if side == order.Buy || side == order.Sell {
		params.Set("side", strings.ToLower(side.Title()))
	}
	if a == asset.Spot || a == asset.Margin || a == asset.CrossMargin {
		params.Set("account", a.String())
	}
	var response []*SpotOrder
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotCancelAllOpenOrdersEPL, http.MethodDelete, gateioSpotOrders, params, nil, &response)
}

// CancelBatchOrdersWithIDList cancels batch orders specifying the order ID and currency pair information
// Multiple currency pairs can be specified, but maximum 20 orders are allowed per request
func (e *Exchange) CancelBatchOrdersWithIDList(ctx context.Context, args []CancelOrderByIDParam) ([]*CancelOrderByIDResponse, error) {
	if len(args) == 0 {
		return nil, errNoValidParameterPassed
	} else if len(args) > 20 {
		return nil, fmt.Errorf("%w maximum order size to cancel is 20", errInvalidOrderSize)
	}
	for x := range args {
		if args[x].CurrencyPair.IsEmpty() || args[x].ID == "" {
			return nil, fmt.Errorf("%w %w currency pair or order ID are required", currency.ErrCurrencyPairEmpty, order.ErrOrderIDNotSet)
		}
	}
	var response []*CancelOrderByIDResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotCancelBatchOrdersEPL, http.MethodPost, "spot/cancel_batch_orders", nil, &args, &response)
}

// GetSpotOrder retrieves a single spot order using the order id and currency pair information.
func (e *Exchange) GetSpotOrder(ctx context.Context, orderID string, currencyPair currency.Pair, a asset.Item) (*SpotOrder, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if currencyPair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("currency_pair", currencyPair.String())
	if accountType := a.String(); accountType != "" {
		params.Set("account", accountType)
	}
	var response *SpotOrder
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotGetOrderEPL, http.MethodGet, gateioSpotOrders+"/"+orderID, params, nil, &response)
}

// AmendSpotOrder amend an order
// By default, the orders of spot and margin account are updated.
// If you need to modify orders of the cross-margin account, you must specify account as cross_margin.
// For portfolio margin account, only cross_margin account is supported.
func (e *Exchange) AmendSpotOrder(ctx context.Context, orderID string, currencyPair currency.Pair, isCrossMarginAccount bool, arg *PriceAndAmount) (*SpotOrder, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if currencyPair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("currency_pair", currencyPair.String())
	if isCrossMarginAccount {
		params.Set("account", asset.CrossMargin.String())
	}
	if arg.Price <= 0 && arg.Amount <= 0 {
		return nil, fmt.Errorf("%w %w : either price or amount has to be set", order.ErrAmountIsInvalid, limits.ErrPriceBelowMin)
	}
	var resp *SpotOrder
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotAmendOrderEPL, http.MethodPatch, gateioSpotOrders+"/"+orderID, params, arg, &resp)
}

// CancelSingleSpotOrder cancels a single order
// Spot and margin orders are cancelled by default.
// If trying to cancel cross margin orders or portfolio margin account are used, account must be set to cross_margin
func (e *Exchange) CancelSingleSpotOrder(ctx context.Context, orderID string, currencyPair currency.Pair, isCrossMarginAccount bool) (*SpotOrder, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if currencyPair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	if isCrossMarginAccount {
		params.Set("account", asset.CrossMargin.String())
	}
	params.Set("currency_pair", currencyPair.String())
	var response *SpotOrder
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotCancelSingleOrderEPL, http.MethodDelete, gateioSpotOrders+"/"+orderID, params, nil, &response)
}

// GetMySpotTradingHistory retrieves personal trading history
func (e *Exchange) GetMySpotTradingHistory(ctx context.Context, p currency.Pair, orderID string, page, limit uint64, crossMargin bool, from, to time.Time) ([]*SpotPersonalTradeHistory, error) {
	if !from.IsZero() && !to.IsZero() {
		if err := common.StartEndTimeCheck(from, to); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if p.IsPopulated() {
		params.Set("currency_pair", p.String())
	}
	if orderID != "" {
		params.Set("order_id", orderID)
	}
	if crossMargin {
		params.Set("account", asset.CrossMargin.String())
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() && to.After(from) {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var response []*SpotPersonalTradeHistory
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotTradingHistoryEPL, http.MethodGet, "spot/my_trades", params, nil, &response)
}

// GetServerTime retrieves current server time
func (e *Exchange) GetServerTime(ctx context.Context, _ asset.Item) (time.Time, error) {
	var resp struct {
		ServerTime types.Time `json:"server_time"`
	}
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, publicGetServerTimeEPL, "spot/time", &resp); err != nil {
		return time.Time{}, err
	}
	return resp.ServerTime.Time(), nil
}

// CountdownCancelorders Countdown cancel orders
// When the timeout set by the user is reached, if there is no cancel or set a new countdown, the related pending orders will be automatically cancelled.
// This endpoint can be called repeatedly to set a new countdown or cancel the countdown.
func (e *Exchange) CountdownCancelorders(ctx context.Context, arg CountdownCancelOrderParam) (*TriggerTimeResponse, error) {
	if arg.Timeout <= 0 {
		return nil, errInvalidCountdown
	}
	var response *TriggerTimeResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotCountdownCancelEPL, http.MethodPost, "spot/countdown_cancel_all", nil, &arg, &response)
}

// CreatePriceTriggeredOrder create a price-triggered order
func (e *Exchange) CreatePriceTriggeredOrder(ctx context.Context, arg *PriceTriggeredOrderParam) (*OrderID, error) {
	if arg.Put.TimeInForce == "" {
		return nil, fmt.Errorf("%w: %q only 'gtc' and 'ioc' are supported", order.ErrInvalidTimeInForce, arg.Put.TimeInForce)
	}
	if arg.Symbol.IsEmpty() {
		return nil, fmt.Errorf("%w, %s", currency.ErrCurrencyPairEmpty, "field market is required")
	}
	if arg.Trigger.Price < 0 {
		return nil, fmt.Errorf("%w trigger price found %f, but expected trigger_price >=0", limits.ErrPriceBelowMin, arg.Trigger.Price)
	}
	if arg.Trigger.Rule != "<=" && arg.Trigger.Rule != ">=" {
		return nil, fmt.Errorf("invalid price trigger condition or rule %q but expected '>=' or '<='", arg.Trigger.Rule)
	}
	if arg.Put.Side != "buy" && arg.Put.Side != "sell" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.Put.Price < 0 {
		return nil, fmt.Errorf("%w, %s", limits.ErrPriceBelowMin, "put price has to be greater than 0")
	}
	if arg.Put.Amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	arg.Put.Side = strings.ToLower(arg.Put.Side)
	arg.Put.Type = strings.ToLower(arg.Put.Type)
	arg.Put.Account = strings.ToLower(arg.Put.Account)
	if arg.Put.Account == "" {
		arg.Put.Account = "normal"
	}
	var response *OrderID
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotCreateTriggerOrderEPL, http.MethodPost, gateioSpotPriceOrders, nil, &arg, &response)
}

// GetPriceTriggeredOrderList retrieves price orders created with an order detail and trigger price information.
func (e *Exchange) GetPriceTriggeredOrderList(ctx context.Context, status string, market currency.Pair, a asset.Item, offset, limit uint64) ([]*SpotPriceTriggeredOrder, error) {
	if status != statusOpen && status != statusFinished {
		return nil, fmt.Errorf("%w status %s", errInvalidOrderStatus, status)
	}
	params := url.Values{}
	params.Set("status", status)
	if market.IsPopulated() {
		params.Set("market", market.String())
	}
	if a == asset.CrossMargin {
		params.Set("account", a.String())
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	var response []*SpotPriceTriggeredOrder
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotGetTriggerOrderListEPL, http.MethodGet, gateioSpotPriceOrders, params, nil, &response)
}

// CancelMultipleSpotOpenOrders deletes price triggered orders.
func (e *Exchange) CancelMultipleSpotOpenOrders(ctx context.Context, currencyPair currency.Pair, a asset.Item) ([]*SpotPriceTriggeredOrder, error) {
	params := url.Values{}
	if currencyPair.IsPopulated() {
		params.Set("market", currencyPair.String())
	}
	switch a {
	case asset.Empty:
		return nil, asset.ErrNotSupported
	case asset.Spot:
		params.Set("account", "normal")
	default:
		params.Set("account", a.String())
	}
	var response []*SpotPriceTriggeredOrder
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotCancelTriggerOrdersEPL, http.MethodDelete, gateioSpotPriceOrders, params, nil, &response)
}

// GetSinglePriceTriggeredOrder get a single order
func (e *Exchange) GetSinglePriceTriggeredOrder(ctx context.Context, orderID string) (*SpotPriceTriggeredOrder, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var response *SpotPriceTriggeredOrder
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotGetTriggerOrderEPL, http.MethodGet, gateioSpotPriceOrders+"/"+orderID, nil, nil, &response)
}

// CancelPriceTriggeredOrder cancel a price-triggered order
func (e *Exchange) CancelPriceTriggeredOrder(ctx context.Context, orderID string) (*SpotPriceTriggeredOrder, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var response *SpotPriceTriggeredOrder
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotCancelTriggerOrderEPL, http.MethodGet, gateioSpotPriceOrders+"/"+orderID, nil, nil, &response)
}

// GenerateSignature returns hash for authenticated requests
func (e *Exchange) GenerateSignature(secret, method, path, query string, body any, dtime time.Time) (string, error) {
	rawQuery, err := url.QueryUnescape(query)
	if err != nil {
		return "", err
	}

	h := sha512.New()
	if body != nil {
		val, err := json.Marshal(body)
		if err != nil {
			return "", err
		}
		h.Write(val)
	}

	h.Write(nil)
	hashedPayload := hex.EncodeToString(h.Sum(nil))
	msg := method + "\n" + path + "\n" + rawQuery + "\n" + hashedPayload + "\n" + strconv.FormatInt(dtime.Unix(), 10)
	mac := hmac.New(sha512.New, []byte(secret))
	mac.Write([]byte(msg))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

// *********************************** Withdrawals ******************************

// WithdrawCurrency to withdraw a currency.
func (e *Exchange) WithdrawCurrency(ctx context.Context, arg *WithdrawalRequestParam) (*WithdrawalResponse, error) {
	if arg.Amount <= 0 {
		return nil, fmt.Errorf("%w currency amount must be greater than zero", order.ErrAmountIsInvalid)
	}
	if arg.Currency.IsEmpty() {
		return nil, fmt.Errorf("%w currency to be withdrawal nust be specified", currency.ErrCurrencyCodeEmpty)
	}
	if arg.Chain == "" {
		return nil, errInvalidCurrencyChain
	}
	var resp *WithdrawalResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletWithdrawEPL, http.MethodPost, withdrawal, nil, &arg, &resp)
}

// TransferBetweenSubAccountsByUID transfers between main spot accounts. Both parties cannot be sub-accounts
func (e *Exchange) TransferBetweenSubAccountsByUID(ctx context.Context, arg *SubAccountTransfer) (*OrderID, error) {
	if arg.Amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	if arg.ReceiveUID <= 0 {
		return nil, errInvalidSubAccountUserID
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp *OrderID
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodPost, "withdrawals/push", nil, &arg, &resp)
}

// CancelWithdrawalWithSpecifiedID cancels withdrawal with specified ID.
func (e *Exchange) CancelWithdrawalWithSpecifiedID(ctx context.Context, withdrawalID string) (*WithdrawalResponse, error) {
	if withdrawalID == "" {
		return nil, errMissingWithdrawalID
	}
	var response *WithdrawalResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletCancelWithdrawEPL, http.MethodDelete, withdrawal+"/"+withdrawalID, nil, nil, &response)
}

// *********************************** Wallet ***********************************

// ListCurrencyChain retrieves a list of currency chain name
func (e *Exchange) ListCurrencyChain(ctx context.Context, ccy currency.Code) ([]*CurrencyChain, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	var resp []*CurrencyChain
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, publicListCurrencyChainEPL, common.EncodeURLValues("wallet/currency_chains", params), &resp)
}

// GenerateCurrencyDepositAddress generate currency deposit address
func (e *Exchange) GenerateCurrencyDepositAddress(ctx context.Context, ccy currency.Code) (*CurrencyDepositAddressInfo, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	var response *CurrencyDepositAddressInfo
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletDepositAddressEPL, http.MethodGet, "wallet/deposit_address", params, nil, &response)
}

// GetWithdrawalRecords retrieves withdrawal records. Record time range cannot exceed 30 days
func (e *Exchange) GetWithdrawalRecords(ctx context.Context, ccy currency.Code, from, to time.Time, offset, limit uint64) ([]*WithdrawalResponse, error) {
	if !from.IsZero() && !to.IsZero() {
		if err := common.StartEndTimeCheck(from, to); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	var withdrawals []*WithdrawalResponse
	return withdrawals, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletWithdrawalRecordsEPL, http.MethodGet, "wallet/withdrawals", params, nil, &withdrawals)
}

// GetDepositRecords retrieves deposit records. Record time range cannot exceed 30 days
func (e *Exchange) GetDepositRecords(ctx context.Context, ccy currency.Code, from, to time.Time, offset, limit uint64) ([]*DepositRecord, error) {
	if !from.IsZero() && !to.IsZero() {
		if err := common.StartEndTimeCheck(from, to); err != nil && !to.IsZero() {
			return nil, err
		}
	}
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	var depositHistories []*DepositRecord
	return depositHistories, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletDepositRecordsEPL, http.MethodGet, "wallet/deposits", params, nil, &depositHistories)
}

// TransferCurrency Transfer between different accounts. Currently support transfers between the following:
// spot - margin, spot - futures(perpetual), spot - delivery
// spot - cross margin, spot - options
func (e *Exchange) TransferCurrency(ctx context.Context, arg *TransferCurrencyParam) (*TransactionIDResponse, error) {
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.From == asset.Empty || !arg.From.IsValid() {
		return nil, fmt.Errorf("%w: source account address(From) is required ", asset.ErrInvalidAsset)
	}
	if arg.To == asset.Empty || !arg.To.IsValid() {
		return nil, fmt.Errorf("%w: destination account address(To) is required ", asset.ErrInvalidAsset)
	}
	if (arg.To == asset.Margin || arg.From == asset.Margin) && arg.CurrencyPair.IsEmpty() {
		return nil, fmt.Errorf("%w: currency pair is required for margin account transfer", currency.ErrCurrencyPairEmpty)
	}
	if (arg.To == asset.Futures || arg.From == asset.Futures) && arg.Settle.IsEmpty() {
		return nil, fmt.Errorf("%w: settle is required for futures account transfer", currency.ErrCurrencyCodeEmpty)
	}
	if arg.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	var response *TransactionIDResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletTransferCurrencyEPL, http.MethodPost, "wallet/transfers", nil, &arg, &response)
}

// SubAccountTransfer to transfer between main and sub accounts
// Support transferring with sub user's spot or futures account. Note that only main user's spot account is used no matter which sub user's account is operated.
func (e *Exchange) SubAccountTransfer(ctx context.Context, arg *SubAccountTransferParam) error {
	if arg.Currency.IsEmpty() {
		return currency.ErrCurrencyCodeEmpty
	}
	if arg.SubAccount == "" {
		return errInvalidSubAccount
	}
	arg.Direction = strings.ToLower(arg.Direction)
	if arg.Direction != "to" && arg.Direction != "from" {
		return errInvalidTransferDirection
	}
	if arg.Amount <= 0 {
		return order.ErrAmountIsInvalid
	}
	switch arg.SubAccountType {
	case "", "spot", "futures", "delivery":
	default:
		return fmt.Errorf("%w %q for SubAccountTransfer; Supported: [spot, futures, delivery]", asset.ErrNotSupported, arg.SubAccountType)
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletSubAccountTransferEPL, http.MethodPost, walletSubAccountTransfer, nil, &arg, nil)
}

// GetSubAccountTransferHistory retrieve transfer records between main and sub accounts.
// retrieve transfer records between main and sub accounts. Record time range cannot exceed 30 days
// Note: only records after 2020-04-10 can be retrieved // TODO:
func (e *Exchange) GetSubAccountTransferHistory(ctx context.Context, subAccountUserID string, from, to time.Time, offset, limit uint64) ([]*SubAccountTransferResponse, error) {
	startingTime, err := time.Parse("2006-Jan-02", "2020-Apr-10")
	if err != nil {
		return nil, err
	}
	if !from.IsZero() && !to.IsZero() {
		if err := common.StartEndTimeCheck(startingTime, from); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	if subAccountUserID != "" {
		params.Set("sub_uid", subAccountUserID)
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var response []*SubAccountTransferResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletSubAccountTransferHistoryEPL, http.MethodGet, walletSubAccountTransfer, params, nil, &response)
}

// SubAccountTransferToSubAccount performs sub-account transfers to sub-account
func (e *Exchange) SubAccountTransferToSubAccount(ctx context.Context, arg *InterSubAccountTransferParams) (*TransactionIDResponse, error) {
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.SubAccountFromUserID == "" {
		return nil, fmt.Errorf("%w: sub-account from user-id is required", errInvalidSubAccountUserID)
	}
	if arg.SubAccountFromAssetType == asset.Empty {
		return nil, fmt.Errorf("%w: sub-account to transfer the asset from is required", asset.ErrInvalidAsset)
	}
	if arg.SubAccountToUserID == "" {
		return nil, fmt.Errorf("%w: sub-account to user-id is required", errInvalidSubAccountUserID)
	}
	if arg.SubAccountToAssetType == asset.Empty {
		return nil, fmt.Errorf("%w: sub-account to transfer to is required", asset.ErrInvalidAsset)
	}
	if arg.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	var resp *TransactionIDResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletSubAccountToSubAccountTransferEPL, http.MethodPost, "wallet/sub_account_to_sub_account", nil, &arg, &resp)
}

// GetWithdrawalStatus retrieves withdrawal status
func (e *Exchange) GetWithdrawalStatus(ctx context.Context, ccy currency.Code) ([]*WithdrawalStatus, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	var response []*WithdrawalStatus
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletWithdrawStatusEPL, http.MethodGet, "wallet/withdraw_status", params, nil, &response)
}

// GetSubAccountBalances retrieve sub account balances
func (e *Exchange) GetSubAccountBalances(ctx context.Context, subAccountUserID string) (*SubAccountBalances, error) {
	params := url.Values{}
	if subAccountUserID != "" {
		params.Set("sub_uid", subAccountUserID)
	}
	var response *SubAccountBalances
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletSubAccountBalancesEPL, http.MethodGet, "wallet/sub_account_balances", params, nil, &response)
}

// GetSubAccountMarginBalances query sub accounts' margin balances
func (e *Exchange) GetSubAccountMarginBalances(ctx context.Context, subAccountUserID string) ([]*SubAccountMarginBalance, error) {
	params := url.Values{}
	if subAccountUserID != "" {
		params.Set("sub_uid", subAccountUserID)
	}
	var response []*SubAccountMarginBalance
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletSubAccountMarginBalancesEPL, http.MethodGet, "wallet/sub_account_margin_balances", params, nil, &response)
}

// GetTransferOrderStatus supports querying transfer status based on user-defined client_order_id or tx_id returned by the transfer interface
func (e *Exchange) GetTransferOrderStatus(ctx context.Context, clientOrderID, transactionID string) (*TransferStatus, error) {
	params := url.Values{}
	if clientOrderID != "" {
		params.Set("client_order_id", clientOrderID)
	}
	if transactionID != "" {
		params.Set("tx_id", transactionID)
	}
	var resp *TransferStatus
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodGet, "wallet/order_status", params, nil, &resp)
}

// GetSubAccountFuturesBalances retrieves sub accounts' futures account balances
func (e *Exchange) GetSubAccountFuturesBalances(ctx context.Context, subAccountUserID string, settle currency.Code) ([]*FuturesSubAccountBalance, error) {
	params := url.Values{}
	if subAccountUserID != "" {
		params.Set("sub_uid", subAccountUserID)
	}
	if !settle.IsEmpty() {
		params.Set("settle", settle.Item.Lower)
	}
	var response []*FuturesSubAccountBalance
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletSubAccountFuturesBalancesEPL, http.MethodGet, "wallet/sub_account_futures_balances", params, nil, &response)
}

// GetSubAccountCrossMarginBalances query subaccount's cross_margin account info
func (e *Exchange) GetSubAccountCrossMarginBalances(ctx context.Context, subAccountUserID string) ([]*SubAccountCrossMarginInfo, error) {
	params := url.Values{}
	if subAccountUserID != "" {
		params.Set("sub_uid", subAccountUserID)
	}
	var response []*SubAccountCrossMarginInfo
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletSubAccountCrossMarginBalancesEPL, http.MethodGet, "wallet/sub_account_cross_margin_balances", params, nil, &response)
}

// GetSavedAddresses retrieves saved currency address info and related details.
func (e *Exchange) GetSavedAddresses(ctx context.Context, ccy currency.Code, chain string, limit, page uint64) ([]*WalletSavedAddress, error) {
	params := url.Values{}
	if ccy.IsEmpty() {
		return nil, fmt.Errorf("%w address is required", currency.ErrCurrencyPairEmpty)
	}
	params.Set("currency", ccy.String())
	if chain != "" {
		params.Set("chain", chain)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	var response []*WalletSavedAddress
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletSavedAddressesEPL, http.MethodGet, "wallet/saved_address", params, nil, &response)
}

// GetPersonalTradingFee retrieves personal trading fee
func (e *Exchange) GetPersonalTradingFee(ctx context.Context, currencyPair currency.Pair, settle currency.Code) (*PersonalTradingFee, error) {
	params := url.Values{}
	if currencyPair.IsPopulated() {
		// specify a currency pair to retrieve precise fee rate
		params.Set("currency_pair", currencyPair.String())
	}
	if !settle.IsEmpty() {
		params.Set("settle", settle.Item.Lower)
	}
	var response *PersonalTradingFee
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletTradingFeeEPL, http.MethodGet, "wallet/fee", params, nil, &response)
}

// GetUsersTotalBalance retrieves user's total balances
func (e *Exchange) GetUsersTotalBalance(ctx context.Context, ccy currency.Code) (*UsersAllAccountBalance, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	var response *UsersAllAccountBalance
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletTotalBalanceEPL, http.MethodGet, "wallet/total_balance", params, nil, &response)
}

// ConvertSmallBalances converts small balances of provided currencies into GT.
// If no currencies are provided, all supported currencies will be converted
// See [this documentation](https://www.gate.io/help/guide/functional_guidelines/22367) for details and restrictions.
func (e *Exchange) ConvertSmallBalances(ctx context.Context, currs ...currency.Code) error {
	currencyList := make([]string, len(currs))
	for i := range currs {
		if currs[i].IsEmpty() {
			return currency.ErrCurrencyCodeEmpty
		}
		currencyList[i] = currs[i].Upper().String()
	}

	payload := struct {
		Currency []string `json:"currency"`
		IsAll    bool     `json:"is_all"`
	}{
		Currency: currencyList,
		IsAll:    len(currs) == 0,
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletConvertSmallBalancesEPL, http.MethodPost, "wallet/small_balance", nil, payload, nil)
}

// GetConvertibleSmallBalanceCurrencyHistory get convertible small balance currency history
func (e *Exchange) GetConvertibleSmallBalanceCurrencyHistory(ctx context.Context, ccy currency.Code, page, limit int64) ([]*SmallCurrencyBalance, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []*SmallCurrencyBalance
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodGet, "wallet/small_balance_history", params, nil, &resp)
}

// ********************************* Margin *******************************************

// GetEstimatedInterestRate retrieves estimated interest rate for provided currencies
func (e *Exchange) GetEstimatedInterestRate(ctx context.Context, currencies []currency.Code) (map[string]types.Number, error) {
	if len(currencies) == 0 {
		return nil, currency.ErrCurrencyCodesEmpty
	}
	if len(currencies) > 10 {
		return nil, fmt.Errorf("%w: maximum 10", errTooManyCurrencyCodes)
	}
	var currStr strings.Builder
	for i := range currencies {
		if currencies[i].IsEmpty() {
			return nil, currency.ErrCurrencyCodeEmpty
		}
		if i != 0 {
			currStr.WriteString(",")
		}
		currStr.WriteString(currencies[i].String())
	}
	params := url.Values{}
	params.Set("currencies", currStr.String())

	var response map[string]types.Number
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginEstimateRateEPL, http.MethodGet, "margin/uni/estimate_rate", params, nil, &response)
}

// GetMarginSupportedCurrencyPairs retrieves margin supported currency pairs.
func (e *Exchange) GetMarginSupportedCurrencyPairs(ctx context.Context) ([]*MarginCurrencyPairInfo, error) {
	var resp []*MarginCurrencyPairInfo
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, publicCurrencyPairsMarginEPL, gateioMarginCurrencyPairs, &resp)
}

// GetSingleMarginSupportedCurrencyPair retrieves margin supported currency pair detail given the currency pair.
func (e *Exchange) GetSingleMarginSupportedCurrencyPair(ctx context.Context, market currency.Pair) (*MarginCurrencyPairInfo, error) {
	if market.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var currencyPairInfo *MarginCurrencyPairInfo
	return currencyPairInfo, e.SendHTTPRequest(ctx, exchange.RestSpot, publicCurrencyPairsMarginEPL, gateioMarginCurrencyPairs+"/"+market.String(), &currencyPairInfo)
}

// GetOrderbookOfLendingLoans retrieves order book of lending loans for specific currency
func (e *Exchange) GetOrderbookOfLendingLoans(ctx context.Context, ccy currency.Code) ([]*OrderbookOfLendingLoan, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var lendingLoans []*OrderbookOfLendingLoan
	return lendingLoans, e.SendHTTPRequest(ctx, exchange.RestSpot, publicOrderbookMarginEPL, "margin/funding_book?currency="+ccy.String(), &lendingLoans)
}

// GetMarginAccountList margin account list
func (e *Exchange) GetMarginAccountList(ctx context.Context, currencyPair currency.Pair) ([]*MarginAccountItem, error) {
	params := url.Values{}
	if currencyPair.IsPopulated() {
		params.Set("currency_pair", currencyPair.String())
	}
	var response []*MarginAccountItem
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginAccountListEPL, http.MethodGet, "margin/accounts", params, nil, &response)
}

// ListMarginAccountBalanceChangeHistory retrieves margin account balance change history
// Only transferals from and to margin account are provided for now. Time range allows 30 days at most
func (e *Exchange) ListMarginAccountBalanceChangeHistory(ctx context.Context, ccy currency.Code, currencyPair currency.Pair, from, to time.Time, page, limit uint64) ([]*MarginAccountBalanceChangeInfo, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	if currencyPair.IsPopulated() {
		params.Set("currency_pair", currencyPair.String())
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() && ((!from.IsZero() && to.After(from)) || from.IsZero()) {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var response []*MarginAccountBalanceChangeInfo
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginAccountBalanceEPL, http.MethodGet, "margin/account_book", params, nil, &response)
}

// GetMarginFundingAccountList retrieves funding account list
func (e *Exchange) GetMarginFundingAccountList(ctx context.Context, ccy currency.Code) ([]*MarginFundingAccountItem, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	var response []*MarginFundingAccountItem
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginFundingAccountListEPL, http.MethodGet, "margin/funding_accounts", params, nil, &response)
}

// MarginLoan represents lend or borrow request
func (e *Exchange) MarginLoan(ctx context.Context, arg *MarginLoanRequestParam) (*MarginLoanResponse, error) {
	if arg.Side != sideLend && arg.Side != sideBorrow {
		return nil, errInvalidLoanSide
	}
	if arg.Side == sideBorrow && arg.Rate == 0 {
		return nil, errLoanRateIsRequired
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	var response *MarginLoanResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginLendBorrowEPL, http.MethodPost, gateioMarginLoans, nil, &arg, &response)
}

// GetMarginAllLoans retrieves all loans (borrow and lending) orders.
func (e *Exchange) GetMarginAllLoans(ctx context.Context, status, side, sortBy string, ccy currency.Code, currencyPair currency.Pair, reverseSort bool, page, limit uint64) ([]*MarginLoanResponse, error) {
	if side != sideLend && side != sideBorrow {
		return nil, fmt.Errorf("%w, only 'lend' and 'borrow' are supported", order.ErrSideIsInvalid)
	}
	if status == "" {
		return nil, errInvalidLoanSide
	}
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	if currencyPair.IsPopulated() {
		params.Set("currency_pair", currencyPair.String())
	}
	if sortBy == "create_time" || sortBy == "rate" {
		params.Set("sort_by", sortBy)
	}
	if reverseSort {
		params.Set("reverse_sort", strconv.FormatBool(reverseSort))
	}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}

	params.Set("status", status)
	var response []*MarginLoanResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginAllLoansEPL, http.MethodGet, gateioMarginLoans, params, nil, &response)
}

// MergeMultipleLendingLoans merge multiple lending loans
func (e *Exchange) MergeMultipleLendingLoans(ctx context.Context, ccy currency.Code, ids []string) (*MarginLoanResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if len(ids) < 2 || len(ids) > 20 {
		return nil, errors.New("number of loans to be merged must be between [2-20], inclusive")
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	params.Set("ids", strings.Join(ids, ","))
	var response *MarginLoanResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginMergeLendingLoansEPL, http.MethodPost, "margin/merged_loans", params, nil, &response)
}

// RetriveOneSingleLoanDetail retrieve one single loan detail
// "side" represents loan side: Lend or Borrow
func (e *Exchange) RetriveOneSingleLoanDetail(ctx context.Context, side, loanID string) (*MarginLoanResponse, error) {
	if side != sideBorrow && side != sideLend {
		return nil, errInvalidLoanSide
	}
	if loanID == "" {
		return nil, errInvalidLoanID
	}
	params := url.Values{}
	params.Set("side", side)
	var response *MarginLoanResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginGetLoanEPL, http.MethodGet, gateioMarginLoans+"/"+loanID+"/", params, nil, &response)
}

// ModifyALoan Modify a loan
// only auto_renew modification is supported currently
func (e *Exchange) ModifyALoan(ctx context.Context, loanID string, arg *ModifyLoanRequestParam) (*MarginLoanResponse, error) {
	if loanID == "" {
		return nil, fmt.Errorf("%w, %v", errInvalidLoanID, " loan_id is required")
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Side != sideBorrow && arg.Side != sideLend {
		return nil, errInvalidLoanSide
	}
	if arg.CurrencyPair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var response *MarginLoanResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginModifyLoanEPL, http.MethodPatch, gateioMarginLoans+"/"+loanID, nil, &arg, &response)
}

// CancelLendingLoan cancels lending loans. only lent loans can be canceled.
func (e *Exchange) CancelLendingLoan(ctx context.Context, ccy currency.Code, loanID string) (*MarginLoanResponse, error) {
	if loanID == "" {
		return nil, fmt.Errorf("%w, %s", errInvalidLoanID, " loan_id is required")
	}
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	var response *MarginLoanResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginCancelLoanEPL, http.MethodDelete, gateioMarginLoans+"/"+loanID, params, nil, &response)
}

// RepayALoan execute a loan repay.
func (e *Exchange) RepayALoan(ctx context.Context, loanID string, arg *RepayLoanRequestParam) (*MarginLoanResponse, error) {
	if loanID == "" {
		return nil, fmt.Errorf("%w, %v", errInvalidLoanID, " loan_id is required")
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.CurrencyPair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if arg.Mode != "all" && arg.Mode != "partial" {
		return nil, errInvalidRepayMode
	}
	if arg.Mode == "partial" && arg.Amount <= 0 {
		return nil, fmt.Errorf("%w, repay amount for partial repay mode must be greater than 0", order.ErrAmountIsInvalid)
	}
	var response *MarginLoanResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginRepayLoanEPL, http.MethodPost, gateioMarginLoans+"/"+loanID+"/repayment", nil, &arg, &response)
}

// ListLoanRepaymentRecords retrieves loan repayment records for specified loan ID
func (e *Exchange) ListLoanRepaymentRecords(ctx context.Context, loanID string) ([]*LoanRepaymentRecord, error) {
	if loanID == "" {
		return nil, fmt.Errorf("%w, %v", errInvalidLoanID, " loan_id is required")
	}
	var response []*LoanRepaymentRecord
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginListLoansEPL, http.MethodGet, gateioMarginLoans+"/"+loanID+"/repayment", nil, nil, &response)
}

// ListRepaymentRecordsOfSpecificLoan retrieves repayment records of specific loan
func (e *Exchange) ListRepaymentRecordsOfSpecificLoan(ctx context.Context, loanID, status string, page, limit uint64) ([]*LoanRecord, error) {
	if loanID == "" {
		return nil, fmt.Errorf("%w, %v", errInvalidLoanID, " loan_id is required")
	}
	params := url.Values{}
	params.Set("loan_id", loanID)
	if status == statusLoaned || status == statusFinished {
		params.Set("status", status)
	}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var response []*LoanRecord
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginRepaymentRecordEPL, http.MethodGet, gateioMarginLoanRecords, params, nil, &response)
}

// GetOneSingleLoanRecord get one single loan record
func (e *Exchange) GetOneSingleLoanRecord(ctx context.Context, loanID, loanRecordID string) (*LoanRecord, error) {
	if loanID == "" {
		return nil, fmt.Errorf("%w, %v", errInvalidLoanID, " loan_id is required")
	}
	if loanRecordID == "" {
		return nil, fmt.Errorf("%w, %v", errInvalidLoanID, " loan_record_id is required")
	}
	params := url.Values{}
	params.Set("loan_id", loanID)
	var response *LoanRecord
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginSingleRecordEPL, http.MethodGet, gateioMarginLoanRecords+"/"+loanRecordID, params, nil, &response)
}

// ModifyLoanRecord modify a loan record
// Only auto_renew modification is supported currently
func (e *Exchange) ModifyLoanRecord(ctx context.Context, loanRecordID string, arg *ModifyLoanRequestParam) (*LoanRecord, error) {
	if loanRecordID == "" {
		return nil, fmt.Errorf("%w, %v", errInvalidLoanID, " loan_record_id is required")
	}
	if arg.LoanID == "" {
		return nil, fmt.Errorf("%w, %v", errInvalidLoanID, " loan_id is required")
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Side != sideBorrow && arg.Side != sideLend {
		return nil, errInvalidLoanSide
	}
	var response *LoanRecord
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginModifyLoanRecordEPL, http.MethodPatch, gateioMarginLoanRecords+"/"+loanRecordID, nil, &arg, &response)
}

// UpdateUsersAutoRepaymentSetting represents update user's auto repayment setting
func (e *Exchange) UpdateUsersAutoRepaymentSetting(ctx context.Context, statusOn bool) (*OnOffStatus, error) {
	statusStr := "off"
	if statusOn {
		statusStr = "on"
	}
	params := url.Values{}
	params.Set("status", statusStr)
	var response *OnOffStatus
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginAutoRepayEPL, http.MethodPost, gateioMarginAutoRepay, params, nil, &response)
}

// GetUserAutoRepaymentSetting retrieve user auto repayment setting
func (e *Exchange) GetUserAutoRepaymentSetting(ctx context.Context) (*OnOffStatus, error) {
	var response *OnOffStatus
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginGetAutoRepaySettingsEPL, http.MethodGet, gateioMarginAutoRepay, nil, nil, &response)
}

// GetMaxTransferableAmountForSpecificMarginCurrency get the max transferable amount for a specific margin currency.
func (e *Exchange) GetMaxTransferableAmountForSpecificMarginCurrency(ctx context.Context, ccy currency.Code, currencyPair currency.Pair) (*MaxTransferAndLoanAmount, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	if currencyPair.IsPopulated() {
		params.Set("currency_pair", currencyPair.String())
	}
	params.Set("currency", ccy.String())
	var response *MaxTransferAndLoanAmount
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginGetMaxTransferEPL, http.MethodGet, "margin/transferable", params, nil, &response)
}

// GetMaxBorrowableAmountForSpecificMarginCurrency retrieves the max borrowble amount for specific currency
func (e *Exchange) GetMaxBorrowableAmountForSpecificMarginCurrency(ctx context.Context, ccy currency.Code, currencyPair currency.Pair) (*MaxTransferAndLoanAmount, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	if currencyPair.IsPopulated() {
		params.Set("currency_pair", currencyPair.String())
	}
	params.Set("currency", ccy.String())
	var response *MaxTransferAndLoanAmount
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginGetMaxBorrowEPL, http.MethodGet, "margin/borrowable", params, nil, &response)
}

// CurrencySupportedByCrossMargin currencies supported by cross margin.
func (e *Exchange) CurrencySupportedByCrossMargin(ctx context.Context) ([]*CrossMarginCurrencies, error) {
	var response []*CrossMarginCurrencies
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginSupportedCurrencyCrossListEPL, http.MethodGet, gateioCrossMarginCurrencies, nil, nil, &response)
}

// GetCrossMarginSupportedCurrencyDetail retrieve detail of one single currency supported by cross margin
func (e *Exchange) GetCrossMarginSupportedCurrencyDetail(ctx context.Context, ccy currency.Code) (*CrossMarginCurrencies, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var response *CrossMarginCurrencies
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginSupportedCurrencyCrossEPL, http.MethodGet, gateioCrossMarginCurrencies+"/"+ccy.String(), nil, nil, &response)
}

// GetCrossMarginAccounts retrieve cross margin account
func (e *Exchange) GetCrossMarginAccounts(ctx context.Context) (*CrossMarginAccount, error) {
	var response *CrossMarginAccount
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginAccountsEPL, http.MethodGet, "margin/cross/accounts", nil, nil, &response)
}

// GetCrossMarginAccountChangeHistory retrieve cross margin account change history
// Record time range cannot exceed 30 days
// possible values of account change types are "in", "out", "repay", "new_order", "order_fill", "referral_fee", "order_fee", and "unknown"
func (e *Exchange) GetCrossMarginAccountChangeHistory(ctx context.Context, ccy currency.Code, from, to time.Time, page, limit uint64, accountChangeType string) ([]*CrossMarginAccountHistoryItem, error) {
	if !from.IsZero() && !to.IsZero() {
		if err := common.StartEndTimeCheck(from, to); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if accountChangeType != "" {
		params.Set("type", accountChangeType)
	}
	var response []*CrossMarginAccountHistoryItem
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginAccountHistoryEPL, http.MethodGet, "margin/cross/account_book", params, nil, &response)
}

// CreateCrossMarginBorrowLoan create a cross margin borrow loan
// Borrow amount cannot be less than currency minimum borrow amount
func (e *Exchange) CreateCrossMarginBorrowLoan(ctx context.Context, arg CrossMarginBorrowLoanParams) (*CrossMarginLoanResponse, error) {
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Amount <= 0 {
		return nil, fmt.Errorf("%w, borrow amount must be greater than 0", order.ErrAmountIsInvalid)
	}
	var response CrossMarginLoanResponse
	return &response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginCreateCrossBorrowLoanEPL, http.MethodPost, gateioCrossMarginLoans, nil, &arg, &response)
}

// ExecuteRepayment when the liquidity of the currency is insufficient and the transaction risk is high, the currency will be disabled,
// and funds cannot be transferred.When the available balance of cross-margin is insufficient, the balance of the spot account can be used for repayment.
// Please ensure that the balance of the spot account is sufficient, and system uses cross-margin account for repayment first
func (e *Exchange) ExecuteRepayment(ctx context.Context, arg CurrencyAndAmount) ([]*CrossMarginLoanResponse, error) {
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Amount <= 0 {
		return nil, fmt.Errorf("%w, repay amount must be greater than 0", order.ErrAmountIsInvalid)
	}
	var response []*CrossMarginLoanResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginExecuteRepaymentsEPL, http.MethodPost, gateioCrossMarginRepayments, nil, &arg, &response)
}

// GetCrossMarginRepayments retrieves list of cross margin repayments
func (e *Exchange) GetCrossMarginRepayments(ctx context.Context, ccy currency.Code, loanID string, limit, offset uint64, reverse bool) ([]*CrossMarginLoanResponse, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	if loanID != "" {
		params.Set("loanId", loanID)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if reverse {
		params.Set("reverse", "true")
	}
	var response []*CrossMarginLoanResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginGetCrossMarginRepaymentsEPL, http.MethodGet, gateioCrossMarginRepayments, params, nil, &response)
}

// GetMaxTransferableAmountForSpecificCrossMarginCurrency get the max transferable amount for a specific cross margin currency
func (e *Exchange) GetMaxTransferableAmountForSpecificCrossMarginCurrency(ctx context.Context, ccy currency.Code) (*CurrencyAndAmount, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	var response *CurrencyAndAmount
	params.Set("currency", ccy.String())
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginGetMaxTransferCrossEPL, http.MethodGet, "margin/cross/transferable", params, nil, &response)
}

// GetMaxBorrowableAmountForSpecificCrossMarginCurrency returns the max borrowable amount for a specific cross margin currency
func (e *Exchange) GetMaxBorrowableAmountForSpecificCrossMarginCurrency(ctx context.Context, ccy currency.Code) (*CurrencyAndAmount, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	var response *CurrencyAndAmount
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginGetMaxBorrowCrossEPL, http.MethodGet, "margin/cross/borrowable", params, nil, &response)
}

// GetCrossMarginBorrowHistory retrieves cross margin borrow history sorted by creation time in descending order by default.
// Set reverse=false to return ascending results.
func (e *Exchange) GetCrossMarginBorrowHistory(ctx context.Context, status uint64, ccy currency.Code, limit, offset uint64, reverse bool) ([]*CrossMarginLoanResponse, error) {
	if status < 1 || status > 3 {
		return nil, fmt.Errorf("%s %w, only allowed status values are 1:failed, 2:borrowed, and 3:repayment", e.Name, errInvalidOrderStatus)
	}
	params := url.Values{}
	params.Set("status", strconv.FormatUint(status, 10))
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if reverse {
		params.Set("reverse", strconv.FormatBool(reverse))
	}
	var response []*CrossMarginLoanResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginGetCrossBorrowHistoryEPL, http.MethodGet, gateioCrossMarginLoans, params, nil, &response)
}

// GetSingleBorrowLoanDetail retrieve single borrow loan detail
func (e *Exchange) GetSingleBorrowLoanDetail(ctx context.Context, loanID string) (*CrossMarginLoanResponse, error) {
	if loanID == "" {
		return nil, errInvalidLoanID
	}
	var response *CrossMarginLoanResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginGetBorrowEPL, http.MethodGet, gateioCrossMarginLoans+"/"+loanID, nil, nil, &response)
}

// *********************************Futures***************************************

// GetAllFutureContracts retrieves list all futures contracts
func (e *Exchange) GetAllFutureContracts(ctx context.Context, settle currency.Code) ([]*FuturesContract, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	var contracts []*FuturesContract
	return contracts, e.SendHTTPRequest(ctx, exchange.RestSpot, publicFuturesContractsEPL, futuresPath+settle.Item.Lower+"/contracts", &contracts)
}

// GetFuturesContract returns a single futures contract info for the specified settle and Currency Pair (contract << in this case)
func (e *Exchange) GetFuturesContract(ctx context.Context, settle currency.Code, contract string) (*FuturesContract, error) {
	if contract == "" {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	var futureContract *FuturesContract
	return futureContract, e.SendHTTPRequest(ctx, exchange.RestSpot, publicFuturesContractsEPL, futuresPath+settle.Item.Lower+"/contracts/"+contract, &futureContract)
}

// GetFuturesOrderbook retrieves futures order book data
func (e *Exchange) GetFuturesOrderbook(ctx context.Context, settle currency.Code, contract currency.Pair, interval string, limit uint64, withOrderbookID bool) (*Orderbook, error) {
	if contract.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	params.Set("contract", contract.String())
	if interval != "" {
		params.Set("interval", interval)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if withOrderbookID {
		params.Set("with_id", "true")
	}
	var response *Orderbook
	return response, e.SendHTTPRequest(ctx, exchange.RestSpot, publicOrderbookFuturesEPL, common.EncodeURLValues(futuresPath+settle.Item.Lower+"/order_book", params), &response)
}

// GetFuturesTradingHistory retrieves futures trading history
func (e *Exchange) GetFuturesTradingHistory(ctx context.Context, settle currency.Code, contract currency.Pair, limit, offset uint64, lastID string, from, to time.Time) ([]*TradingHistoryItem, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("contract", contract.Upper().String())
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if lastID != "" {
		params.Set("last_id", lastID)
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	var response []*TradingHistoryItem
	return response, e.SendHTTPRequest(ctx, exchange.RestSpot, publicTradingHistoryFuturesEPL, common.EncodeURLValues(futuresPath+settle.Item.Lower+"/trades", params), &response)
}

// GetFuturesCandlesticks retrieves specified contract candlesticks.
func (e *Exchange) GetFuturesCandlesticks(ctx context.Context, settle currency.Code, contract currency.Pair, from, to time.Time, limit uint64, interval kline.Interval) ([]*FuturesCandlestick, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("contract", contract.String())
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if interval.Duration().Microseconds() != 0 {
		intervalString, err := getIntervalString(interval)
		if err != nil {
			return nil, err
		}
		params.Set("interval", intervalString)
	}
	var candlesticks []*FuturesCandlestick
	return candlesticks, e.SendHTTPRequest(ctx, exchange.RestFutures, publicCandleSticksFuturesEPL, common.EncodeURLValues(futuresPath+settle.Item.Lower+"/candlesticks", params), &candlesticks)
}

// PremiumIndexKLine retrieves premium Index K-Line
// Maximum of 1000 points can be returned in a query. Be sure not to exceed the limit when specifying from, to and interval
func (e *Exchange) PremiumIndexKLine(ctx context.Context, settleCurrency currency.Code, contract currency.Pair, from, to time.Time, limit int64, interval kline.Interval) ([]*FuturesPremiumIndexKLineResponse, error) {
	if settleCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("contract", contract.String())
	if from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	intervalString, err := getIntervalString(interval)
	if err != nil {
		return nil, err
	}
	params.Set("interval", intervalString)
	var resp []*FuturesPremiumIndexKLineResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, publicPremiumIndexEPL, common.EncodeURLValues(futuresPath+settleCurrency.Item.Lower+"/premium_index", params), &resp)
}

// GetFuturesTickers retrieves futures ticker information for a specific settle and contract info.
func (e *Exchange) GetFuturesTickers(ctx context.Context, settle currency.Code, contract currency.Pair) ([]*FuturesTicker, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	var tickers []*FuturesTicker
	return tickers, e.SendHTTPRequest(ctx, exchange.RestSpot, publicTickersFuturesEPL, common.EncodeURLValues(futuresPath+settle.Item.Lower+"/tickers", params), &tickers)
}

// GetFutureFundingRates retrieves funding rate information.
func (e *Exchange) GetFutureFundingRates(ctx context.Context, settle currency.Code, contract currency.Pair, limit uint64) ([]*FuturesFundingRate, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsInvalid() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("contract", contract.String())
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var rates []*FuturesFundingRate
	return rates, e.SendHTTPRequest(ctx, exchange.RestSpot, publicFundingRatesEPL, common.EncodeURLValues(futuresPath+settle.Item.Lower+"/funding_rate", params), &rates)
}

// GetFuturesInsuranceBalanceHistory retrieves futures insurance balance history
func (e *Exchange) GetFuturesInsuranceBalanceHistory(ctx context.Context, settle currency.Code, limit uint64) ([]*InsuranceBalance, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var balances []*InsuranceBalance
	return balances, e.SendHTTPRequest(ctx, exchange.RestSpot, publicInsuranceFuturesEPL, common.EncodeURLValues(futuresPath+settle.Item.Lower+"/insurance", params), &balances)
}

// GetFutureStats retrieves futures stats
func (e *Exchange) GetFutureStats(ctx context.Context, settle currency.Code, contract currency.Pair, from time.Time, interval kline.Interval, limit uint64) ([]*ContractStat, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w: settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsInvalid() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("contract", contract.String())
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if int64(interval) != 0 {
		intervalString, err := getIntervalString(interval)
		if err != nil {
			return nil, err
		}
		params.Set("interval", intervalString)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var stats []*ContractStat
	return stats, e.SendHTTPRequest(ctx, exchange.RestSpot, publicStatsFuturesEPL, common.EncodeURLValues(futuresPath+settle.Item.Lower+"/contract_stats", params), &stats)
}

// GetIndexConstituent retrieves index constituents
func (e *Exchange) GetIndexConstituent(ctx context.Context, settle currency.Code, index string) (*IndexConstituent, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if index == "" {
		return nil, fmt.Errorf("%w: index pair string is required", currency.ErrCurrencyPairEmpty)
	}
	indexString := strings.ToUpper(index)
	var constituents *IndexConstituent
	return constituents, e.SendHTTPRequest(ctx, exchange.RestSpot, publicIndexConstituentsEPL, futuresPath+settle.Item.Lower+"/index_constituents/"+indexString, &constituents)
}

// GetLiquidationHistory retrieves liqudiation history
func (e *Exchange) GetLiquidationHistory(ctx context.Context, settle currency.Code, contract currency.Pair, from, to time.Time, limit uint64) ([]*LiquidationHistory, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	if !from.IsZero() && !to.IsZero() {
		if err := common.StartEndTimeCheck(from, to); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("contract", contract.String())
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var histories []*LiquidationHistory
	return histories, e.SendHTTPRequest(ctx, exchange.RestSpot, publicLiquidationHistoryEPL, common.EncodeURLValues(futuresPath+settle.Lower().String()+"/liq_orders", params), &histories)
}

// GetRiskLimitTiers retrieves risk limit tiers
// When the 'contract' parameter is not passed, the default is to query the risk limits for the top 100 markets.
// 'Limit' and 'offset' correspond to pagination queries at the market level, not to the length of the returned array.
func (e *Exchange) GetRiskLimitTiers(ctx context.Context, settle currency.Code, contract currency.Pair, offset, limit uint64) ([]*RiskLimitTier, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	params := url.Values{}
	params.Set("contract", contract.String())
	if offset > 0 {
		params.Set("from", strconv.FormatUint(offset, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []*RiskLimitTier
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, publicLiquidationHistoryEPL, common.EncodeURLValues(futuresPath+settle.Lower().String()+"/risk_limit_tiers", params), &resp)
}

// QueryFuturesAccount retrieves futures account
func (e *Exchange) QueryFuturesAccount(ctx context.Context, settle currency.Code) (*FuturesAccount, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w: settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	var response *FuturesAccount
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualAccountEPL, http.MethodGet, futuresPath+settle.Item.Lower+"/accounts", nil, nil, &response)
}

// GetFuturesAccountBooks retrieves account books
func (e *Exchange) GetFuturesAccountBooks(ctx context.Context, settle currency.Code, limit uint64, from, to time.Time, changingType string) ([]*AccountBookItem, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if !from.IsZero() && !to.IsZero() {
		if err := common.StartEndTimeCheck(from, to); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	if changingType != "" {
		params.Set("type", changingType)
	}
	var response []*AccountBookItem
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualAccountBooksEPL, http.MethodGet, futuresPath+settle.Item.Lower+"/account_book", params, nil, &response)
}

// GetAllFuturesPositionsOfUsers list all positions of users.
func (e *Exchange) GetAllFuturesPositionsOfUsers(ctx context.Context, settle currency.Code, realPositionsOnly bool) ([]*Position, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if realPositionsOnly {
		params.Set("holding", "true")
	}
	var response []*Position
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualPositionsEPL, http.MethodGet, futuresPath+settle.Item.Lower+"/positions", params, nil, &response)
}

// GetSinglePosition returns a single position
func (e *Exchange) GetSinglePosition(ctx context.Context, settle currency.Code, contract currency.Pair) (*Position, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	var response *Position
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualPositionEPL, http.MethodPost, futuresPath+settle.Item.Lower+positionsPath+contract.String(), nil, nil, &response)
}

// UpdateFuturesPositionMargin represents account position margin for a futures contract.
func (e *Exchange) UpdateFuturesPositionMargin(ctx context.Context, settle currency.Code, change float64, contract currency.Pair) (*Position, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	if change <= 0 {
		return nil, fmt.Errorf("%w, futures margin change must be positive", errChangeHasToBePositive)
	}
	params := url.Values{}
	params.Set("change", strconv.FormatFloat(change, 'f', -1, 64))
	var response *Position
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualUpdateMarginEPL, http.MethodPost, futuresPath+settle.Item.Lower+positionsPath+contract.String()+"/margin", params, nil, &response)
}

// UpdateFuturesPositionLeverage update position leverage
func (e *Exchange) UpdateFuturesPositionLeverage(ctx context.Context, settle currency.Code, contract currency.Pair, leverage, crossLeverageLimit float64) (*Position, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	if leverage < 0 {
		return nil, fmt.Errorf("%w: %f", order.ErrSubmitLeverageNotSupported, leverage)
	}
	params := url.Values{}
	params.Set("leverage", strconv.FormatFloat(leverage, 'f', -1, 64))
	if leverage == 0 && crossLeverageLimit > 0 {
		params.Set("cross_leverage_limit", strconv.FormatFloat(crossLeverageLimit, 'f', -1, 64))
	}
	var resp Position
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualUpdateLeverageEPL, http.MethodPost, futuresPath+settle.Item.Lower+positionsPath+contract.String()+"/leverage", params, nil, &[1]*Position{&resp})
}

// UpdateFuturesPositionRiskLimit updates the position risk limit
func (e *Exchange) UpdateFuturesPositionRiskLimit(ctx context.Context, settle currency.Code, contract currency.Pair, riskLimit uint64) (*Position, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	params := url.Values{}
	params.Set("risk_limit", strconv.FormatUint(riskLimit, 10))
	var response *Position
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualUpdateRiskEPL, http.MethodPost, futuresPath+settle.Item.Lower+positionsPath+contract.String()+"/risk_limit", params, nil, &response)
}

// EnableOrDisableDualMode enable or disable dual mode
// Before setting dual mode, make sure all positions are closed and no orders are open
func (e *Exchange) EnableOrDisableDualMode(ctx context.Context, settle currency.Code, dualMode bool) (*DualModeResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	params.Set("dual_mode", strconv.FormatBool(dualMode))
	var response *DualModeResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualToggleDualModeEPL, http.MethodGet, futuresPath+settle.Item.Lower+"/dual_mode", params, nil, &response)
}

// GetPositionDetailInDualMode retrieve position detail in dual mode
func (e *Exchange) GetPositionDetailInDualMode(ctx context.Context, settle currency.Code, contract currency.Pair) ([]*Position, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	var response []*Position
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualPositionsDualModeEPL, http.MethodGet, futuresPath+settle.Item.Lower+hedgeModePath+contract.String(), nil, nil, &response)
}

// UpdatePositionMarginInDualMode update position margin in dual mode
func (e *Exchange) UpdatePositionMarginInDualMode(ctx context.Context, settle currency.Code, contract currency.Pair, change float64, dualSide string) ([]*Position, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	if dualSide != "dual_long" && dualSide != "dual_short" {
		return nil, fmt.Errorf("%w: 'dual_side' should be 'dual_short' or 'dual_long'", order.ErrSideIsInvalid)
	}
	params := url.Values{}
	params.Set("change", strconv.FormatFloat(change, 'f', -1, 64))
	params.Set("dual_side", dualSide)
	var response []*Position
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualUpdateMarginDualModeEPL, http.MethodPost, futuresPath+settle.Item.Lower+hedgeModePath+contract.String()+"/margin", params, nil, &response)
}

// UpdatePositionLeverageInDualMode update position leverage in dual mode
func (e *Exchange) UpdatePositionLeverageInDualMode(ctx context.Context, settle currency.Code, contract currency.Pair, leverage, crossLeverageLimit float64) (*Position, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	if leverage < 0 {
		return nil, fmt.Errorf("%w: %f", order.ErrSubmitLeverageNotSupported, leverage)
	}
	params := url.Values{}
	params.Set("leverage", strconv.FormatFloat(leverage, 'f', -1, 64))
	if leverage == 0 && crossLeverageLimit > 0 {
		params.Set("cross_leverage_limit", strconv.FormatFloat(crossLeverageLimit, 'f', -1, 64))
	}
	var response *Position
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualUpdateLeverageDualModeEPL, http.MethodPost, futuresPath+settle.Item.Lower+"/dual_comp/positions/"+contract.String()+"/leverage", params, nil, &response)
}

// UpdatePositionRiskLimitInDualMode update position risk limit in dual mode
// Risk Limit as of GateIO: https://www.gate.com/en/help/futures/futures-logic/22162
func (e *Exchange) UpdatePositionRiskLimitInDualMode(ctx context.Context, settle currency.Code, contract currency.Pair, riskLimit float64) ([]*Position, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	if riskLimit <= 0 {
		return nil, errInvalidRiskLimit
	}
	params := url.Values{}
	params.Set("risk_limit", strconv.FormatFloat(riskLimit, 'f', -1, 64))
	var response []*Position
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualUpdateRiskDualModeEPL, http.MethodPost, futuresPath+settle.Item.Lower+"/dual_comp/positions/"+contract.String()+"/risk_limit", params, nil, &response)
}

// PlaceFuturesOrder creates futures order
// Create a futures order
// Creating futures orders requires size, which is number of contracts instead of currency amount. You can use quanto_multiplier in contract detail response to know how much currency 1 size contract represents
// Zero-filled order cannot be retrieved 10 minutes after order cancellation. You will get a 404 not found for such orders
// Set reduce_only to true can keep the position from changing side when reducing position size
// In single position mode, to close a position, you need to set size to 0 and close to true
// In dual position mode, to close one side position, you need to set auto_size side, reduce_only to true and size to 0
func (e *Exchange) PlaceFuturesOrder(ctx context.Context, arg *ContractOrderCreateParams) (*Order, error) {
	if err := arg.validate(true); err != nil {
		return nil, err
	}
	var response *Order
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualSubmitOrderEPL, http.MethodPost, futuresPath+arg.Settle.Item.Lower+ordersPath, nil, &arg, &response)
}

// GetFuturesOrders retrieves list of futures orders
// Zero-filled order cannot be retrieved 10 minutes after order cancellation
func (e *Exchange) GetFuturesOrders(ctx context.Context, contract currency.Pair, status, lastID string, settle currency.Code, limit, offset uint64, countTotal bool) ([]*Order, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if status != statusOpen && status != statusFinished {
		return nil, fmt.Errorf("%w, only 'open' and 'finished' status are supported", errInvalidOrderStatus)
	}
	params := url.Values{}
	params.Set("status", status)
	if !contract.IsEmpty() {
		params.Set("contract", contract.String())
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if lastID != "" {
		params.Set("last_id", lastID)
	}
	if countTotal && status != statusOpen {
		params.Set("count_total", "1")
	}
	var response []*Order
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualGetOrdersEPL, http.MethodGet, futuresPath+settle.Item.Lower+ordersPath, params, nil, &response)
}

// CancelMultipleFuturesOpenOrders ancel all open orders
// Zero-filled order cannot be retrieved 10 minutes after order cancellation
func (e *Exchange) CancelMultipleFuturesOpenOrders(ctx context.Context, contract currency.Pair, side string, settle currency.Code) ([]*Order, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	params := url.Values{}
	if side != "" {
		params.Set("side", side)
	}
	params.Set("contract", contract.String())
	var response []*Order
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualGetOrdersEPL, http.MethodDelete, futuresPath+settle.Item.Lower+ordersPath, params, nil, &response)
}

// PlaceBatchFuturesOrders creates a list of futures orders
// Up to 10 orders per request
// If any of the order's parameters are missing or in the wrong format, all of them will not be executed, and a http status 400 error will be returned directly
// If the parameters are checked and passed, all are executed. Even if there is a business logic error in the middle (such as insufficient funds), it will not affect other execution orders
// The returned result is in array format, and the order corresponds to the orders in the request body
// In the returned result, the succeeded field of type bool indicates whether the execution was successful or not
// If the execution is successful, the normal order content is included; if the execution fails, the label field is included to indicate the cause of the error
// In the rate limiting, each order is counted individually
func (e *Exchange) PlaceBatchFuturesOrders(ctx context.Context, settle currency.Code, args []ContractOrderCreateParams) ([]*Order, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if len(args) > 10 {
		return nil, errTooManyOrderRequest
	}
	for x := range args {
		if err := args[x].validate(true); err != nil {
			return nil, err
		}
	}
	var response []*Order
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualSubmitBatchOrdersEPL, http.MethodPost, futuresPath+settle.Item.Lower+"/batch_orders", nil, &args, &response)
}

// GetSingleFuturesOrder retrieves a single order by its identifier
func (e *Exchange) GetSingleFuturesOrder(ctx context.Context, settle currency.Code, orderID string) (*Order, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if orderID == "" {
		return nil, fmt.Errorf("%w, 'order_id' cannot be empty", order.ErrOrderIDNotSet)
	}
	var response *Order
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualFetchOrderEPL, http.MethodGet, futuresPath+settle.Item.Lower+"/orders/"+orderID, nil, nil, &response)
}

// CancelSingleFuturesOrder cancel a single order
func (e *Exchange) CancelSingleFuturesOrder(ctx context.Context, settle currency.Code, orderID string) (*Order, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if orderID == "" {
		return nil, fmt.Errorf("%w, 'order_id' cannot be empty", order.ErrOrderIDNotSet)
	}
	var response *Order
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualCancelOrderEPL, http.MethodDelete, futuresPath+settle.Item.Lower+"/orders/"+orderID, nil, nil, &response)
}

// AmendFuturesOrder amends an existing futures order
func (e *Exchange) AmendFuturesOrder(ctx context.Context, settle currency.Code, orderID string, arg AmendFuturesOrderParam) (*Order, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if orderID == "" {
		return nil, fmt.Errorf("%w, 'order_id' cannot be empty", order.ErrOrderIDNotSet)
	}
	if arg.Size <= 0 && arg.Price <= 0 {
		return nil, errors.New("missing update 'size' or 'price', please specify 'size' or 'price' or both information")
	}
	var response *Order
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualAmendOrderEPL, http.MethodPut, futuresPath+settle.Item.Lower+"/orders/"+orderID, nil, &arg, &response)
}

// GetMyFuturesTradingHistory retrieves authenticated account's futures trading history
func (e *Exchange) GetMyFuturesTradingHistory(ctx context.Context, settle currency.Code, lastID, orderID string, contract currency.Pair, limit, offset, countTotal uint64) ([]*TradingHistoryItem, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	if orderID != "" {
		params.Set("order", orderID)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if lastID != "" {
		params.Set("last_id", lastID)
	}
	if countTotal == 1 {
		params.Set("count_total", strconv.FormatUint(countTotal, 10))
	}
	var response []*TradingHistoryItem
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualTradingHistoryEPL, http.MethodGet, futuresPath+settle.Item.Lower+"/my_trades", params, nil, &response)
}

// GetFuturesPositionCloseHistory lists position close history
func (e *Exchange) GetFuturesPositionCloseHistory(ctx context.Context, settle currency.Code, contract currency.Pair, limit, offset uint64, from, to time.Time) ([]*PositionCloseHistoryResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	var response []*PositionCloseHistoryResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualClosePositionEPL, http.MethodGet, futuresPath+settle.Item.Lower+"/position_close", params, nil, &response)
}

// GetFuturesLiquidationHistory list liquidation history
func (e *Exchange) GetFuturesLiquidationHistory(ctx context.Context, settle currency.Code, contract currency.Pair, limit uint64, at time.Time) ([]*LiquidationHistoryItem, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if !at.IsZero() {
		params.Set("at", strconv.FormatInt(at.Unix(), 10))
	}
	var response []*LiquidationHistoryItem
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualLiquidationHistoryEPL, http.MethodGet, futuresPath+settle.Item.Lower+"/liquidates", params, nil, &response)
}

// CountdownCancelOrders represents a trigger time response
func (e *Exchange) CountdownCancelOrders(ctx context.Context, settle currency.Code, arg CountdownParams) (*TriggerTimeResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if arg.Timeout < 0 {
		return nil, errInvalidTimeout
	}
	var response *TriggerTimeResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualCancelTriggerOrdersEPL, http.MethodPost, futuresPath+settle.Item.Lower+"/countdown_cancel_all", nil, &arg, &response)
}

// CreatePriceTriggeredFuturesOrder create a price-triggered order
func (e *Exchange) CreatePriceTriggeredFuturesOrder(ctx context.Context, settle currency.Code, arg *FuturesPriceTriggeredOrderParam) (*OrderID, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if arg.Initial.Contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	if arg.Initial.Price < 0 {
		return nil, fmt.Errorf("%w, price must be greater than 0", limits.ErrPriceBelowMin)
	}
	if arg.Initial.TimeInForce != "" && arg.Initial.TimeInForce != gtcTIF && arg.Initial.TimeInForce != iocTIF {
		return nil, fmt.Errorf("%w: %q; only 'gtc' and 'ioc' are allowed", order.ErrInvalidTimeInForce, arg.Initial.TimeInForce)
	}
	if arg.Trigger.StrategyType != 0 && arg.Trigger.StrategyType != 1 {
		return nil, errors.New("strategy type must be 0 or 1, 0: by price, and 1: by price gap")
	}
	if arg.Trigger.Rule != 1 && arg.Trigger.Rule != 2 {
		return nil, errors.New("invalid trigger condition('rule') value, rule must be 1 or 2")
	}
	if arg.Trigger.PriceType != 0 && arg.Trigger.PriceType != 1 && arg.Trigger.PriceType != 2 {
		return nil, errors.New("price type must be 0, 1 or 2")
	}
	if arg.Trigger.OrderType != "" &&
		arg.Trigger.OrderType != "close-long-order" &&
		arg.Trigger.OrderType != "close-short-order" &&
		arg.Trigger.OrderType != "close-long-position" &&
		arg.Trigger.OrderType != "close-short-position" &&
		arg.Trigger.OrderType != "plan-close-long-position" &&
		arg.Trigger.OrderType != "plan-close-short-position" {
		return nil, errors.New("invalid order type, only 'close-long-order', 'close-short-order', 'close-long-position', 'close-short-position', 'plan-close-long-position', and 'plan-close-short-position'")
	}
	var response *OrderID
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualSubmitTriggerOrderEPL, http.MethodPost, futuresPath+settle.Item.Lower+priceOrdersPaths, nil, &arg, &response)
}

// ListAllFuturesAutoOrders lists all open orders
func (e *Exchange) ListAllFuturesAutoOrders(ctx context.Context, status string, settle currency.Code, contract currency.Pair, limit, offset uint64) ([]*PriceTriggeredOrder, error) {
	if status != statusOpen && status != statusFinished {
		return nil, fmt.Errorf("%w status: %s", errInvalidOrderStatus, status)
	}
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	params.Set("status", status)
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	var response []*PriceTriggeredOrder
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualListOpenOrdersEPL, http.MethodGet, futuresPath+settle.Item.Lower+priceOrdersPaths, params, nil, &response)
}

// CancelAllFuturesOpenOrders cancels all futures open orders
func (e *Exchange) CancelAllFuturesOpenOrders(ctx context.Context, settle currency.Code, contract currency.Pair) ([]*PriceTriggeredOrder, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	params := url.Values{}
	params.Set("contract", contract.String())
	var response []*PriceTriggeredOrder
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualCancelOpenOrdersEPL, http.MethodDelete, futuresPath+settle.Item.Lower+priceOrdersPaths, params, nil, &response)
}

// GetSingleFuturesPriceTriggeredOrder retrieves a single price triggered order
func (e *Exchange) GetSingleFuturesPriceTriggeredOrder(ctx context.Context, settle currency.Code, orderID string) (*PriceTriggeredOrder, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var response *PriceTriggeredOrder
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualGetTriggerOrderEPL, http.MethodGet, futuresPath+settle.Item.Lower+"/price_orders/"+orderID, nil, nil, &response)
}

// CancelFuturesPriceTriggeredOrder cancel a price-triggered order
func (e *Exchange) CancelFuturesPriceTriggeredOrder(ctx context.Context, settle currency.Code, orderID string) (*PriceTriggeredOrder, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var response *PriceTriggeredOrder
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualCancelTriggerOrderEPL, http.MethodDelete, futuresPath+settle.Item.Lower+"/price_orders/"+orderID, nil, nil, &response)
}

// *************************************** Delivery ***************************************

// GetAllDeliveryContracts retrieves all futures contracts
func (e *Exchange) GetAllDeliveryContracts(ctx context.Context, settle currency.Code) ([]*DeliveryContract, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	var contracts []*DeliveryContract
	return contracts, e.SendHTTPRequest(ctx, exchange.RestSpot, publicDeliveryContractsEPL, deliveryPath+settle.Item.Lower+"/contracts", &contracts)
}

// GetDeliveryContract retrieves a single delivery contract instance
func (e *Exchange) GetDeliveryContract(ctx context.Context, settle currency.Code, contract currency.Pair) (*DeliveryContract, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	var deliveryContract *DeliveryContract
	return deliveryContract, e.SendHTTPRequest(ctx, exchange.RestSpot, publicDeliveryContractsEPL, deliveryPath+settle.Item.Lower+"/contracts/"+contract.String(), &deliveryContract)
}

// GetDeliveryOrderbook delivery orderbook
func (e *Exchange) GetDeliveryOrderbook(ctx context.Context, settle currency.Code, interval string, contract currency.Pair, limit uint64, withOrderbookID bool) (*Orderbook, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	params := url.Values{}
	params.Set("contract", contract.String())
	if interval != "" {
		params.Set("interval", interval)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if withOrderbookID {
		params.Set("with_id", strconv.FormatBool(withOrderbookID))
	}
	var orderbook *Orderbook
	return orderbook, e.SendHTTPRequest(ctx, exchange.RestSpot, publicOrderbookDeliveryEPL, common.EncodeURLValues(deliveryPath+settle.Item.Lower+"/order_book", params), &orderbook)
}

// GetDeliveryTradingHistory retrieves futures trading history
func (e *Exchange) GetDeliveryTradingHistory(ctx context.Context, settle currency.Code, lastID string, contract currency.Pair, limit uint64, from, to time.Time) ([]*TradingHistoryItem, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	params := url.Values{}
	params.Set("contract", contract.String())
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if lastID != "" {
		params.Set("last_id", lastID)
	}
	var histories []*TradingHistoryItem
	return histories, e.SendHTTPRequest(ctx, exchange.RestSpot, publicTradingHistoryDeliveryEPL, common.EncodeURLValues(deliveryPath+settle.Item.Lower+"/trades", params), &histories)
}

// GetDeliveryFuturesCandlesticks retrieves specified contract candlesticks
func (e *Exchange) GetDeliveryFuturesCandlesticks(ctx context.Context, settle currency.Code, contract currency.Pair, from, to time.Time, limit uint64, interval kline.Interval) ([]*FuturesCandlestick, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	if !from.IsZero() && !to.IsZero() {
		if err := common.StartEndTimeCheck(from, to); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if int64(interval) != 0 {
		intervalString, err := getIntervalString(interval)
		if err != nil {
			return nil, err
		}
		params.Set("interval", intervalString)
	}
	params.Set("contract", contract.Upper().String())
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var candlesticks []*FuturesCandlestick
	return candlesticks, e.SendHTTPRequest(ctx, exchange.RestSpot, publicCandleSticksDeliveryEPL, common.EncodeURLValues(deliveryPath+settle.Item.Lower+"/candlesticks", params), &candlesticks)
}

// GetDeliveryFutureTickers retrieves futures ticker information for a specific settle and contract info.
func (e *Exchange) GetDeliveryFutureTickers(ctx context.Context, settle currency.Code, contract currency.Pair) ([]*FuturesTicker, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	var tickers []*FuturesTicker
	return tickers, e.SendHTTPRequest(ctx, exchange.RestSpot, publicTickersDeliveryEPL, common.EncodeURLValues(deliveryPath+settle.Item.Lower+"/tickers", params), &tickers)
}

// GetDeliveryInsuranceBalanceHistory retrieves delivery futures insurance balance history
func (e *Exchange) GetDeliveryInsuranceBalanceHistory(ctx context.Context, settle currency.Code, limit uint64) ([]*InsuranceBalance, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var balances []*InsuranceBalance
	return balances, e.SendHTTPRequest(ctx, exchange.RestSpot, publicInsuranceDeliveryEPL, common.EncodeURLValues(deliveryPath+settle.Item.Lower+"/insurance", params), &balances)
}

// GetDeliveryFuturesAccounts retrieves futures account
func (e *Exchange) GetDeliveryFuturesAccounts(ctx context.Context, settle currency.Code) (*FuturesAccount, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	var response *FuturesAccount
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryAccountEPL, http.MethodGet, deliveryPath+settle.Item.Lower+"/accounts", nil, nil, &response)
}

// GetDeliveryAccountBooks retrieves account books
func (e *Exchange) GetDeliveryAccountBooks(ctx context.Context, settle currency.Code, limit uint64, from, to time.Time, changingType string) ([]*AccountBookItem, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if !from.IsZero() && !to.IsZero() {
		if err := common.StartEndTimeCheck(from, to); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	if changingType != "" {
		params.Set("type", changingType)
	}
	var response []*AccountBookItem
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryAccountBooksEPL, http.MethodGet, deliveryPath+settle.Item.Lower+"/account_book", params, nil, &response)
}

// GetAllDeliveryPositionsOfUser retrieves all positions of user
func (e *Exchange) GetAllDeliveryPositionsOfUser(ctx context.Context, settle currency.Code) (*Position, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	var response *Position
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryPositionsEPL, http.MethodGet, deliveryPath+settle.Item.Lower+"/positions", nil, nil, &response)
}

// GetSingleDeliveryPosition get single position
func (e *Exchange) GetSingleDeliveryPosition(ctx context.Context, settle currency.Code, contract currency.Pair) (*Position, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	var response *Position
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryPositionsEPL, http.MethodGet, deliveryPath+settle.Item.Lower+positionsPath+contract.String(), nil, nil, &response)
}

// UpdateDeliveryPositionMargin updates position margin
func (e *Exchange) UpdateDeliveryPositionMargin(ctx context.Context, settle currency.Code, change float64, contract currency.Pair) (*Position, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	if change <= 0 {
		return nil, fmt.Errorf("%w, futures margin change must be positive", errChangeHasToBePositive)
	}
	params := url.Values{}
	params.Set("change", strconv.FormatFloat(change, 'f', -1, 64))
	var response *Position
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryUpdateMarginEPL, http.MethodPost, deliveryPath+settle.Item.Lower+positionsPath+contract.String()+"/margin", params, nil, &response)
}

// UpdateDeliveryPositionLeverage updates position leverage
func (e *Exchange) UpdateDeliveryPositionLeverage(ctx context.Context, settle currency.Code, contract currency.Pair, leverage float64) (*Position, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	if leverage < 0 {
		return nil, order.ErrSubmitLeverageNotSupported
	}
	params := url.Values{}
	params.Set("leverage", strconv.FormatFloat(leverage, 'f', -1, 64))
	var response *Position
	return response, e.SendAuthenticatedHTTPRequest(ctx,
		exchange.RestSpot, deliveryUpdateLeverageEPL, http.MethodPost, deliveryPath+settle.Item.Lower+positionsPath+contract.String()+"/leverage", params, nil, &response)
}

// UpdateDeliveryPositionRiskLimit update position risk limit
func (e *Exchange) UpdateDeliveryPositionRiskLimit(ctx context.Context, settle currency.Code, contract currency.Pair, riskLimit uint64) (*Position, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	params := url.Values{}
	params.Set("risk_limit", strconv.FormatUint(riskLimit, 10))
	var response *Position
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryUpdateRiskLimitEPL, http.MethodPost, deliveryPath+settle.Item.Lower+positionsPath+contract.String()+"/risk_limit", params, nil, &response)
}

// PlaceDeliveryOrder create a futures order
// Zero-filled order cannot be retrieved 10 minutes after order cancellation
func (e *Exchange) PlaceDeliveryOrder(ctx context.Context, arg *ContractOrderCreateParams) (*Order, error) {
	if err := arg.validate(true); err != nil {
		return nil, err
	}
	var response *Order
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliverySubmitOrderEPL, http.MethodPost, deliveryPath+arg.Settle.Item.Lower+ordersPath, nil, &arg, &response)
}

// GetDeliveryOrders list futures orders
// Zero-filled order cannot be retrieved 10 minutes after order cancellation
func (e *Exchange) GetDeliveryOrders(ctx context.Context, contract currency.Pair, settle currency.Code, status, lastID string, limit, offset uint64, countTotal bool) ([]*Order, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if status != statusOpen && status != statusFinished {
		return nil, fmt.Errorf("%w, only 'open' and 'finished' status are supported", errInvalidOrderStatus)
	}
	params := url.Values{}
	if !contract.IsEmpty() {
		params.Set("contract", contract.String())
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if lastID != "" {
		params.Set("last_id", lastID)
	}
	if countTotal && status != statusOpen {
		params.Set("count_total", "1")
	}
	params.Set("status", status)
	var response []*Order
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryGetOrdersEPL, http.MethodGet, deliveryPath+settle.Item.Lower+ordersPath, params, nil, &response)
}

// CancelMultipleDeliveryOrders cancel all open orders matched
// Zero-filled order cannot be retrieved 10 minutes after order cancellation
func (e *Exchange) CancelMultipleDeliveryOrders(ctx context.Context, contract currency.Pair, side string, settle currency.Code) ([]*Order, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsInvalid() {
		return nil, fmt.Errorf("%w, currency pair for contract must not be empty", currency.ErrCurrencyPairEmpty)
	}
	params := url.Values{}
	if side == order.Ask.Lower() || side == order.Bid.Lower() {
		params.Set("side", side)
	}
	params.Set("contract", contract.String())
	var response []*Order
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryCancelOrdersEPL, http.MethodDelete, deliveryPath+settle.Item.Lower+ordersPath, params, nil, &response)
}

// GetSingleDeliveryOrder Get a single order
// Zero-filled order cannot be retrieved 10 minutes after order cancellation
func (e *Exchange) GetSingleDeliveryOrder(ctx context.Context, settle currency.Code, orderID string) (*Order, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if orderID == "" {
		return nil, fmt.Errorf("%w, 'order_id' cannot be empty", order.ErrOrderIDNotSet)
	}
	var response *Order
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryGetOrderEPL, http.MethodGet, deliveryPath+settle.Item.Lower+"/orders/"+orderID, nil, nil, &response)
}

// CancelSingleDeliveryOrder cancel a single order
func (e *Exchange) CancelSingleDeliveryOrder(ctx context.Context, settle currency.Code, orderID string) (*Order, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if orderID == "" {
		return nil, fmt.Errorf("%w, 'order_id' cannot be empty", order.ErrOrderIDNotSet)
	}
	var response *Order
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryCancelOrderEPL, http.MethodDelete, deliveryPath+settle.Item.Lower+"/orders/"+orderID, nil, nil, &response)
}

// GetMyDeliveryTradingHistory retrieves authenticated account delivery futures trading history
func (e *Exchange) GetMyDeliveryTradingHistory(ctx context.Context, settle currency.Code, orderID string, contract currency.Pair, limit, offset, countTotal uint64, lastID string) ([]*TradingHistoryItem, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	if orderID != "" {
		params.Set("order", orderID)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if lastID != "" {
		params.Set("last_id", lastID)
	}
	if countTotal == 1 {
		params.Set("count_total", strconv.FormatUint(countTotal, 10))
	}
	var response []*TradingHistoryItem
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryTradingHistoryEPL, http.MethodGet, deliveryPath+settle.Item.Lower+"/my_trades", params, nil, &response)
}

// GetDeliveryPositionCloseHistory retrieves position history
func (e *Exchange) GetDeliveryPositionCloseHistory(ctx context.Context, settle currency.Code, contract currency.Pair, limit, offset uint64, from, to time.Time) ([]*PositionCloseHistoryResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	var response []*PositionCloseHistoryResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryCloseHistoryEPL, http.MethodGet, deliveryPath+settle.Item.Lower+"/position_close", params, nil, &response)
}

// GetDeliveryLiquidationHistory lists liquidation history
func (e *Exchange) GetDeliveryLiquidationHistory(ctx context.Context, settle currency.Code, contract currency.Pair, limit uint64, at time.Time) ([]*LiquidationHistoryItem, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if !at.IsZero() {
		params.Set("at", strconv.FormatInt(at.Unix(), 10))
	}
	var response []*LiquidationHistoryItem
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryLiquidationHistoryEPL, http.MethodGet, deliveryPath+settle.Item.Lower+"/liquidates", params, nil, &response)
}

// GetDeliverySettlementHistory retrieves settlement history
func (e *Exchange) GetDeliverySettlementHistory(ctx context.Context, settle currency.Code, contract currency.Pair, limit uint64, at time.Time) ([]*SettlementHistoryItem, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if !at.IsZero() {
		params.Set("at", strconv.FormatInt(at.Unix(), 10))
	}
	var response []*SettlementHistoryItem
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliverySettlementHistoryEPL, http.MethodGet, deliveryPath+settle.Item.Lower+"/settlements", params, nil, &response)
}

// GetDeliveryPriceTriggeredOrder creates a price-triggered order
func (e *Exchange) GetDeliveryPriceTriggeredOrder(ctx context.Context, settle currency.Code, arg *FuturesPriceTriggeredOrderParam) (*OrderID, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if arg.Initial.Contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	if arg.Initial.Price < 0 {
		return nil, fmt.Errorf("%w, price must be greater than 0", limits.ErrPriceBelowMin)
	}
	if arg.Initial.Size <= 0 {
		return nil, fmt.Errorf("%w: initial.size out of range", limits.ErrAmountBelowMin)
	}
	if arg.Initial.TimeInForce != "" &&
		arg.Initial.TimeInForce != gtcTIF && arg.Initial.TimeInForce != iocTIF {
		return nil, fmt.Errorf("%w: %q; only 'gtc' and 'ioc' are allowed", order.ErrUnsupportedTimeInForce, arg.Initial.TimeInForce)
	}
	if arg.Trigger.StrategyType != 0 && arg.Trigger.StrategyType != 1 {
		return nil, errors.New("strategy type must be 0 or 1, 0: by price, and 1: by price gap")
	}
	if arg.Trigger.Rule != 1 && arg.Trigger.Rule != 2 {
		return nil, errors.New("invalid trigger condition('rule') value, rule must be 1 or 2")
	}
	if arg.Trigger.PriceType != 0 && arg.Trigger.PriceType != 1 && arg.Trigger.PriceType != 2 {
		return nil, errors.New("price type must be 0 or 1 or 2")
	}
	if arg.Trigger.Price <= 0 {
		return nil, fmt.Errorf("%w: trigger.price", limits.ErrPriceBelowMin)
	}
	if arg.Trigger.OrderType != "" &&
		arg.Trigger.OrderType != "close-long-order" &&
		arg.Trigger.OrderType != "close-short-order" &&
		arg.Trigger.OrderType != "close-long-position" &&
		arg.Trigger.OrderType != "close-short-position" &&
		arg.Trigger.OrderType != "plan-close-long-position" &&
		arg.Trigger.OrderType != "plan-close-short-position" {
		return nil, errors.New("invalid order type, only 'close-long-order', 'close-short-order', 'close-long-position', 'close-short-position', 'plan-close-long-position', and 'plan-close-short-position'")
	}
	var response *OrderID
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryGetTriggerOrderEPL, http.MethodPost, deliveryPath+settle.Item.Lower+priceOrdersPaths, nil, &arg, &response)
}

// GetDeliveryAllAutoOrder retrieves all auto orders
func (e *Exchange) GetDeliveryAllAutoOrder(ctx context.Context, status string, settle currency.Code, contract currency.Pair, limit, offset uint64) ([]*PriceTriggeredOrder, error) {
	if status != statusOpen && status != statusFinished {
		return nil, fmt.Errorf("%w status %s", errInvalidOrderStatus, status)
	}
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	params.Set("status", status)
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	var response []*PriceTriggeredOrder
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryAutoOrdersEPL, http.MethodGet, deliveryPath+settle.Item.Lower+priceOrdersPaths, params, nil, &response)
}

// CancelAllDeliveryPriceTriggeredOrder cancels all delivery price triggered orders
func (e *Exchange) CancelAllDeliveryPriceTriggeredOrder(ctx context.Context, settle currency.Code, contract currency.Pair) ([]*PriceTriggeredOrder, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	params := url.Values{}
	params.Set("contract", contract.String())
	var response []*PriceTriggeredOrder
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryCancelTriggerOrdersEPL, http.MethodDelete, deliveryPath+settle.Item.Lower+priceOrdersPaths, params, nil, &response)
}

// GetSingleDeliveryPriceTriggeredOrder retrieves a single price triggered order
func (e *Exchange) GetSingleDeliveryPriceTriggeredOrder(ctx context.Context, settle currency.Code, orderID string) (*PriceTriggeredOrder, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var response *PriceTriggeredOrder
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryGetTriggerOrderEPL, http.MethodGet, deliveryPath+settle.Item.Lower+"/price_orders/"+orderID, nil, nil, &response)
}

// CancelDeliveryPriceTriggeredOrder cancel a price-triggered order
func (e *Exchange) CancelDeliveryPriceTriggeredOrder(ctx context.Context, settle currency.Code, orderID string) (*PriceTriggeredOrder, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var response *PriceTriggeredOrder
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryCancelTriggerOrderEPL, http.MethodDelete, deliveryPath+settle.Item.Lower+"/price_orders/"+orderID, nil, nil, &response)
}

// ********************************** Options ***************************************************

// GetAllOptionsUnderlyings retrieves all option underlyings
func (e *Exchange) GetAllOptionsUnderlyings(ctx context.Context) ([]*OptionUnderlying, error) {
	var response []*OptionUnderlying
	return response, e.SendHTTPRequest(ctx, exchange.RestSpot, publicUnderlyingOptionsEPL, "options/underlyings", &response)
}

// GetExpirationTime return the expiration time for the provided underlying.
func (e *Exchange) GetExpirationTime(ctx context.Context, underlying string) (time.Time, error) {
	if underlying == "" {
		return time.Time{}, errInvalidUnderlying
	}
	var timestamps []types.Time
	err := e.SendHTTPRequest(ctx, exchange.RestSpot, publicExpirationOptionsEPL, "options/expirations?underlying="+underlying, &timestamps)
	if err != nil {
		return time.Time{}, err
	}
	if len(timestamps) == 0 {
		return time.Time{}, errNoValidResponseFromServer
	}
	return timestamps[0].Time(), nil
}

// GetAllContractOfUnderlyingWithinExpiryDate retrieves list of contracts of the specified underlying and expiry time.
func (e *Exchange) GetAllContractOfUnderlyingWithinExpiryDate(ctx context.Context, underlying string, expTime time.Time) ([]*OptionContract, error) {
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	params := url.Values{}
	params.Set("underlying", underlying)
	if !expTime.IsZero() {
		params.Set("expires", strconv.FormatInt(expTime.Unix(), 10))
	}
	var contracts []*OptionContract
	return contracts, e.SendHTTPRequest(ctx, exchange.RestSpot, publicContractsOptionsEPL, common.EncodeURLValues(gateioOptionContracts, params), &contracts)
}

// GetOptionsSpecifiedContractDetail query specified contract detail
func (e *Exchange) GetOptionsSpecifiedContractDetail(ctx context.Context, contract currency.Pair) (*OptionContract, error) {
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	var contr *OptionContract
	return contr, e.SendHTTPRequest(ctx, exchange.RestSpot, publicContractsOptionsEPL, gateioOptionContracts+"/"+contract.String(), &contr)
}

// GetSettlementHistory retrieves list of settlement history
func (e *Exchange) GetSettlementHistory(ctx context.Context, underlying string, offset, limit uint64, from, to time.Time) ([]*OptionSettlement, error) {
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	if !from.IsZero() && !to.IsZero() {
		if err := common.StartEndTimeCheck(from, to); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("underlying", underlying)
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	var settlements []*OptionSettlement
	return settlements, e.SendHTTPRequest(ctx, exchange.RestSpot, publicSettlementOptionsEPL, common.EncodeURLValues(gateioOptionSettlement, params), &settlements)
}

// GetOptionsSpecifiedContractsSettlement retrieve a single contract settlement detail passing the underlying and contract name
func (e *Exchange) GetOptionsSpecifiedContractsSettlement(ctx context.Context, contract currency.Pair, underlying string, at int64) (*OptionSettlement, error) {
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	params := url.Values{}
	params.Set("underlying", underlying)
	params.Set("at", strconv.FormatInt(at, 10))
	var settlement *OptionSettlement
	return settlement, e.SendHTTPRequest(ctx, exchange.RestSpot, publicSettlementOptionsEPL, common.EncodeURLValues(gateioOptionSettlement+"/"+contract.String(), params), &settlement)
}

// GetMyOptionsSettlements retrieves accounts option settlements.
func (e *Exchange) GetMyOptionsSettlements(ctx context.Context, underlying string, contract currency.Pair, offset, limit uint64, to time.Time) ([]*MyOptionSettlement, error) {
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	params := url.Values{}
	params.Set("underlying", underlying)
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	if to.After(time.Now()) {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var settlements []*MyOptionSettlement
	return settlements, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsSettlementsEPL, http.MethodGet, "options/my_settlements", params, nil, &settlements)
}

// GetOptionsOrderbook returns the orderbook data for the given contract.
func (e *Exchange) GetOptionsOrderbook(ctx context.Context, contract currency.Pair, interval string, limit uint64, withOrderbookID bool) (*Orderbook, error) {
	if contract.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("contract", strings.ToUpper(contract.String()))
	if interval != "" {
		params.Set("interval", interval)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	params.Set("with_id", strconv.FormatBool(withOrderbookID))
	var response *Orderbook
	return response, e.SendHTTPRequest(ctx, exchange.RestSpot, publicOrderbookOptionsEPL, common.EncodeURLValues("options/order_book", params), &response)
}

// GetOptionAccounts lists option accounts
func (e *Exchange) GetOptionAccounts(ctx context.Context) (*OptionAccount, error) {
	var resp *OptionAccount
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsAccountsEPL, http.MethodGet, "options/accounts", nil, nil, &resp)
}

// GetAccountChangingHistory retrieves list of account changing history
// possible change type values are: "dnw": Deposit & Withdrawal, "prem": Trading premium, "fee": Trading fee, "refr": Referrer rebate, and "set": Settlement P&L
func (e *Exchange) GetAccountChangingHistory(ctx context.Context, offset, limit uint64, from, to time.Time, changingType string) ([]*AccountBook, error) {
	if !from.IsZero() && !to.IsZero() {
		if err := common.StartEndTimeCheck(from, to); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() && ((!from.IsZero() && to.After(from)) || to.Before(time.Now())) {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	if changingType != "" {
		params.Set("type", changingType)
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var accountBook []*AccountBook
	return accountBook, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsAccountBooksEPL, http.MethodGet, "options/account_book", params, nil, &accountBook)
}

// GetUsersPositionSpecifiedUnderlying lists user's positions of specified underlying
func (e *Exchange) GetUsersPositionSpecifiedUnderlying(ctx context.Context, underlying string) ([]*UsersPositionForUnderlying, error) {
	params := url.Values{}
	if underlying != "" {
		params.Set("underlying", underlying)
	}
	var response []*UsersPositionForUnderlying
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsPositions, http.MethodGet, gateioOptionsPosition, params, nil, &response)
}

// GetSpecifiedContractPosition retrieves specified contract position
func (e *Exchange) GetSpecifiedContractPosition(ctx context.Context, contract currency.Pair) (*UsersPositionForUnderlying, error) {
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	var response *UsersPositionForUnderlying
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsPositions, http.MethodGet, gateioOptionsPosition+"/"+contract.String(), nil, nil, &response)
}

// GetUsersLiquidationHistoryForSpecifiedUnderlying retrieves user's liquidation history of specified underlying
func (e *Exchange) GetUsersLiquidationHistoryForSpecifiedUnderlying(ctx context.Context, underlying string, contract currency.Pair) ([]*ContractClosePosition, error) {
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	params := url.Values{}
	params.Set("underlying", underlying)
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	var response []*ContractClosePosition
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsLiquidationHistoryEPL, http.MethodGet, "options/position_close", params, nil, &response)
}

// PlaceOptionOrder creates an options order
func (e *Exchange) PlaceOptionOrder(ctx context.Context, arg *OptionOrderParam) (*OptionOrderResponse, error) {
	if arg.Contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	if arg.OrderSize <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	var response *OptionOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsSubmitOrderEPL, http.MethodPost, gateioOptionsOrders, nil, &arg, &response)
}

// GetOptionFuturesOrders retrieves futures orders
func (e *Exchange) GetOptionFuturesOrders(ctx context.Context, contract currency.Pair, underlying, status string, offset, limit uint64, from, to time.Time) ([]*OptionOrderResponse, error) {
	if !from.IsZero() && !to.IsZero() {
		if err := common.StartEndTimeCheck(from, to); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	if underlying != "" {
		params.Set("underlying", underlying)
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	status = strings.ToLower(status)
	if status == statusOpen || status == statusFinished {
		params.Set("status", status)
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var response []*OptionOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsOrdersEPL, http.MethodGet, gateioOptionsOrders, params, nil, &response)
}

// CancelMultipleOptionOpenOrders cancels all open orders matched
func (e *Exchange) CancelMultipleOptionOpenOrders(ctx context.Context, contract currency.Pair, underlying, side string) ([]*OptionOrderResponse, error) {
	params := url.Values{}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	if underlying != "" {
		params.Set("underlying", underlying)
	}
	if side != "" {
		params.Set("side", side)
	}
	var response []*OptionOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsCancelOrdersEPL, http.MethodDelete, gateioOptionsOrders, params, nil, &response)
}

// GetSingleOptionOrder retrieves a single option order
func (e *Exchange) GetSingleOptionOrder(ctx context.Context, orderID string) (*OptionOrderResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var o *OptionOrderResponse
	return o, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsOrderEPL, http.MethodGet, gateioOptionsOrders+"/"+orderID, nil, nil, &o)
}

// CancelOptionSingleOrder cancel a single order.
func (e *Exchange) CancelOptionSingleOrder(ctx context.Context, orderID string) (*OptionOrderResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var response *OptionOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsCancelOrderEPL, http.MethodDelete, "options/orders/"+orderID, nil, nil, &response)
}

// GetMyOptionsTradingHistory retrieves authenticated account's option trading history
func (e *Exchange) GetMyOptionsTradingHistory(ctx context.Context, underlying string, contract currency.Pair, offset, limit uint64, from, to time.Time) ([]*OptionTradingHistory, error) {
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	if !from.IsZero() && !to.IsZero() {
		if err := common.StartEndTimeCheck(from, to); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("underlying", underlying)
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() && ((!from.IsZero() && to.After(from)) || to.Before(time.Now())) {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	var resp []*OptionTradingHistory
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsTradingHistoryEPL, http.MethodGet, "options/my_trades", params, nil, &resp)
}

// GetOptionsTickers lists  tickers of options contracts
func (e *Exchange) GetOptionsTickers(ctx context.Context, underlying string) ([]*OptionsTicker, error) {
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	underlying = strings.ToUpper(underlying)
	var response []*OptionsTicker
	return response, e.SendHTTPRequest(ctx, exchange.RestSpot, publicTickerOptionsEPL, "options/tickers?underlying="+underlying, &response)
}

// GetOptionUnderlyingTickers retrieves options underlying ticker
func (e *Exchange) GetOptionUnderlyingTickers(ctx context.Context, underlying string) (*OptionsUnderlyingTicker, error) {
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	var respos *OptionsUnderlyingTicker
	return respos, e.SendHTTPRequest(ctx, exchange.RestSpot, publicUnderlyingTickerOptionsEPL, "options/underlying/tickers/"+underlying, &respos)
}

// GetOptionFuturesCandlesticks retrieves option futures candlesticks
func (e *Exchange) GetOptionFuturesCandlesticks(ctx context.Context, contract currency.Pair, interval kline.Interval, from, to time.Time, limit uint64) ([]*FuturesCandlestick, error) {
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	if !from.IsZero() && !to.IsZero() {
		if err := common.StartEndTimeCheck(from, to); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("contract", contract.String())
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	intervalString, err := getIntervalString(interval)
	if err != nil {
		return nil, err
	}
	params.Set("interval", intervalString)
	var candles []*FuturesCandlestick
	return candles, e.SendHTTPRequest(ctx, exchange.RestSpot, publicCandleSticksOptionsEPL, common.EncodeURLValues("options/candlesticks", params), &candles)
}

// GetOptionFuturesMarkPriceCandlesticks retrieves mark price candlesticks of an underlying
func (e *Exchange) GetOptionFuturesMarkPriceCandlesticks(ctx context.Context, underlying string, limit uint64, from, to time.Time, interval kline.Interval) ([]*FuturesCandlestick, error) {
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	if !from.IsZero() && !to.IsZero() {
		if err := common.StartEndTimeCheck(from, to); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("underlying", underlying)
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	if int64(interval) != 0 {
		intervalString, err := getIntervalString(interval)
		if err != nil {
			return nil, err
		}
		params.Set("interval", intervalString)
	}
	var candles []*FuturesCandlestick
	return candles, e.SendHTTPRequest(ctx, exchange.RestSpot, publicMarkpriceCandleSticksOptionsEPL, common.EncodeURLValues("options/underlying/candlesticks", params), &candles)
}

// GetOptionsTradeHistory retrieves options trade history
func (e *Exchange) GetOptionsTradeHistory(ctx context.Context, contract currency.Pair, callType string, offset, limit uint64, from, to time.Time) ([]*TradingHistoryItem, error) {
	if !from.IsZero() && !to.IsZero() {
		if err := common.StartEndTimeCheck(from, to); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	callType = strings.ToUpper(callType)
	if callType == "C" || callType == "P" {
		params.Set("type", callType)
	}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	var trades []*TradingHistoryItem
	return trades, e.SendHTTPRequest(ctx, exchange.RestSpot, publicTradeHistoryOptionsEPL, common.EncodeURLValues("options/trades", params), &trades)
}

// ********************************** Flash_SWAP *************************

// GetSupportedFlashSwapCurrencies retrieves all supported currencies in flash swap
func (e *Exchange) GetSupportedFlashSwapCurrencies(ctx context.Context) ([]*SwapCurrencies, error) {
	var currencies []*SwapCurrencies
	return currencies, e.SendHTTPRequest(ctx, exchange.RestSpot, publicFlashSwapEPL, "flash_swap/currencies", &currencies)
}

// CreateFlashSwapOrder creates a new flash swap order
// initiate a flash swap preview in advance because order creation requires a preview result
func (e *Exchange) CreateFlashSwapOrder(ctx context.Context, arg FlashSwapOrderParams) (*FlashSwapOrderResponse, error) {
	if arg.PreviewID == "" {
		return nil, errMissingPreviewID
	}
	if arg.BuyCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w, buy currency can not empty", currency.ErrCurrencyCodeEmpty)
	}
	if arg.SellCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w, sell currency can not empty", currency.ErrCurrencyCodeEmpty)
	}
	if arg.SellAmount <= 0 {
		return nil, fmt.Errorf("%w, sell_amount can not be less than or equal to 0", order.ErrAmountIsInvalid)
	}
	if arg.BuyAmount <= 0 {
		return nil, fmt.Errorf("%w, buy_amount amount can not be less than or equal to 0", order.ErrAmountIsInvalid)
	}
	var response *FlashSwapOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, flashSwapOrderEPL, http.MethodPost, gateioFlashSwapOrders, nil, &arg, &response)
}

// GetAllFlashSwapOrders retrieves list of flash swap orders filtered by the params
func (e *Exchange) GetAllFlashSwapOrders(ctx context.Context, status int, sellCurrency, buyCurrency currency.Code, reverse bool, limit, page uint64) ([]*FlashSwapOrderResponse, error) {
	params := url.Values{}
	if status == 1 || status == 2 {
		params.Set("status", strconv.Itoa(status))
	}
	if !sellCurrency.IsEmpty() {
		params.Set("sell_currency", sellCurrency.String())
	}
	if !buyCurrency.IsEmpty() {
		params.Set("buy_currency", buyCurrency.String())
	}
	params.Set("reverse", strconv.FormatBool(reverse))
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var response []*FlashSwapOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, flashGetOrdersEPL, http.MethodGet, gateioFlashSwapOrders, params, nil, &response)
}

// GetSingleFlashSwapOrder get a single flash swap order's detail
func (e *Exchange) GetSingleFlashSwapOrder(ctx context.Context, orderID string) (*FlashSwapOrderResponse, error) {
	if orderID == "" {
		return nil, fmt.Errorf("%w, flash order order_id must not be empty", order.ErrOrderIDNotSet)
	}
	var response *FlashSwapOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, flashGetOrderEPL, http.MethodGet, gateioFlashSwapOrders+"/"+orderID, nil, nil, &response)
}

// InitiateFlashSwapOrderReview initiate a flash swap order preview
func (e *Exchange) InitiateFlashSwapOrderReview(ctx context.Context, arg *FlashSwapOrderParams) (*InitFlashSwapOrderPreviewResponse, error) {
	if arg.PreviewID == "" {
		return nil, errMissingPreviewID
	}
	if arg.BuyCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w, buy currency can not empty", currency.ErrCurrencyCodeEmpty)
	}
	if arg.SellCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w, sell currency can not empty", currency.ErrCurrencyCodeEmpty)
	}
	var response *InitFlashSwapOrderPreviewResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, flashOrderReviewEPL, http.MethodPost, "flash_swap/orders/preview", nil, &arg, &response)
}

// IsValidPairString returns true if the string represents a valid currency pair
func (e *Exchange) IsValidPairString(currencyPairString string) bool {
	if len(currencyPairString) < 3 {
		return false
	}
	pf, err := e.CurrencyPairs.GetFormat(asset.Spot, true)
	if err != nil {
		return false
	}
	if strings.Contains(currencyPairString, pf.Delimiter) {
		result := strings.Split(currencyPairString, pf.Delimiter)
		return len(result) >= 2
	}
	return false
}

// SwapETH2 swaps ETH2
// 1-Forward Swap (ETH -> ETH2), 2-Reverse Swap (ETH2 -> ETH
func (e *Exchange) SwapETH2(ctx context.Context, arg *SwapETHParam) error {
	if err := common.NilGuard(arg); err != nil {
		return err
	}
	if arg.Side == "" {
		return order.ErrSideIsInvalid
	}
	if arg.Amount <= 0 {
		return order.ErrAmountIsInvalid
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotAccountsEPL, http.MethodPost, "earn/staking/eth2/swap", nil, arg, nil)
}

// GetETH2HistoricalReturnRate gets ETH2 historical return rate
// Query ETH earnings rate records for the last 31 days
func (e *Exchange) GetETH2HistoricalReturnRate(ctx context.Context) ([]*ETH2ReturnRate, error) {
	var resp []*ETH2ReturnRate
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotAccountsEPL, http.MethodGet, "earn/staking/eth2/rate_records", nil, nil, &resp)
}

// GetDualInvestmentProductList dual Investment product list
func (e *Exchange) GetDualInvestmentProductList(ctx context.Context, planID uint64) ([]*DualInvestmentPlan, error) {
	params := url.Values{}
	if planID != 0 {
		params.Set("plan_id", strconv.FormatUint(planID, 10))
	}
	var resp []*DualInvestmentPlan
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotAccountsEPL, http.MethodGet, "earn/dual/investment_plan", params, nil, &resp)
}

// GetDualInvestmentOrderList dual Investment order list
func (e *Exchange) GetDualInvestmentOrderList(ctx context.Context, from, to time.Time, page, limit int64) ([]*DualInvestmentOrderDetail, error) {
	if !from.IsZero() && !to.IsZero() {
		if err := common.StartEndTimeCheck(from, to); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.UnixMilli(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	var resp []*DualInvestmentOrderDetail
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotAccountsEPL, http.MethodGet, "earn/dual/orders", params, nil, &resp)
}

// PlaceDualInvestmentOrder place a dual investment order
func (e *Exchange) PlaceDualInvestmentOrder(ctx context.Context, arg *DualInvestmentOrderParam) error {
	if err := common.NilGuard(arg); err != nil {
		return err
	}
	if arg.PlanID == "" {
		return errPlanIDRequired
	}
	if arg.Amount <= 0 {
		return order.ErrAmountIsInvalid
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotAccountsEPL, http.MethodPost, "earn/dual/orders", nil, arg, nil)
}

// GetStructuredProductList retrieves a structured Product List
func (e *Exchange) GetStructuredProductList(ctx context.Context, productType, status string, page, limit int64) ([]*StructuredProductDetail, error) {
	params := url.Values{}
	if productType != "" {
		params.Set("type", productType)
	}
	if status != "" {
		params.Set("status", status)
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []*StructuredProductDetail
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotAccountsEPL, http.MethodGet, "earn/structured/products", params, nil, &resp)
}

// GetStructuredProductOrderList retrieves structured product order list
func (e *Exchange) GetStructuredProductOrderList(ctx context.Context, from, to time.Time, page, limit int64) ([]*StructuredProductOrderDetail, error) {
	if !from.IsZero() && !to.IsZero() {
		if err := common.StartEndTimeCheck(from, to); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.UnixMilli(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.UnixMilli(), 10))
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []*StructuredProductOrderDetail
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotAccountsEPL, http.MethodGet, "earn/structured/orders", params, nil, &resp)
}

// PlaceStructuredProductOrder retrieves structured product orders
func (e *Exchange) PlaceStructuredProductOrder(ctx context.Context, arg *StructuredOrder) (*StructuredOrder, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	var resp *StructuredOrder
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotAccountsEPL, http.MethodPost, "earn/structured/orders", nil, arg, &resp)
}

// ********************************* Trading Fee calculation ********************************

// GetFee returns an estimate of fee based on type of transaction
func (e *Exchange) GetFee(ctx context.Context, feeBuilder *exchange.FeeBuilder) (fee float64, err error) {
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		feePairs, err := e.GetPersonalTradingFee(ctx, feeBuilder.Pair, currency.EMPTYCODE)
		if err != nil {
			return 0, err
		}
		if feeBuilder.IsMaker {
			fee = calculateTradingFee(feePairs.MakerFee.Float64(),
				feeBuilder.PurchasePrice,
				feeBuilder.Amount)
		} else {
			fee = calculateTradingFee(feePairs.TakerFee.Float64(),
				feeBuilder.PurchasePrice,
				feeBuilder.Amount)
		}
	case exchange.CryptocurrencyWithdrawalFee:
		fee = getCryptocurrencyWithdrawalFee(feeBuilder.Pair.Base)
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

func calculateTradingFee(feeForPair, purchasePrice, amount float64) float64 {
	return feeForPair * purchasePrice * amount
}

func getCryptocurrencyWithdrawalFee(c currency.Code) float64 {
	return WithdrawalFees[c]
}

// GetUnderlyingFromCurrencyPair returns an underlying string from a currency pair
func (e *Exchange) GetUnderlyingFromCurrencyPair(p currency.Pair) (currency.Pair, error) {
	pairString := strings.ReplaceAll(p.Upper().String(), currency.DashDelimiter, currency.UnderscoreDelimiter)
	ccies := strings.Split(pairString, currency.UnderscoreDelimiter)
	if len(ccies) < 2 {
		return currency.EMPTYPAIR, fmt.Errorf("invalid currency pair %v", p)
	}
	return currency.Pair{Base: currency.NewCode(ccies[0]), Delimiter: currency.UnderscoreDelimiter, Quote: currency.NewCode(ccies[1])}, nil
}

// GetAccountDetails retrieves account details
func (e *Exchange) GetAccountDetails(ctx context.Context) (*AccountDetails, error) {
	var resp *AccountDetails
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotAccountsEPL, http.MethodGet, "account/detail", nil, nil, &resp)
}

// GetUserTransactionRateLimitInfo retrieves user transaction rate limit info
func (e *Exchange) GetUserTransactionRateLimitInfo(ctx context.Context) ([]*UserTransactionRateLimitInfo, error) {
	var resp []*UserTransactionRateLimitInfo
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotAccountsEPL, http.MethodGet, "account/rate_limit", nil, nil, &resp)
}

// CreateSelfTradePreventionUserGroup create STP user group
// only the main account is allowed to create a new STP user group
func (e *Exchange) CreateSelfTradePreventionUserGroup(ctx context.Context, arg *STPUserGroup) (*STPUserGroup, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Name == "" {
		return nil, errSTPGroupNameRequired
	}
	var resp *STPUserGroup
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotAccountsEPL, http.MethodPost, "account/stp_groups", nil, arg, &resp)
}

// GetUserSelfTradePreventionGroups query STP user groups created by the user
// Only query STP user groups created by the current main account
func (e *Exchange) GetUserSelfTradePreventionGroups(ctx context.Context, name string) ([]*STPUserGroup, error) {
	params := url.Values{}
	if name != "" {
		params.Set("name", name)
	}
	var resp []*STPUserGroup
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotAccountsEPL, http.MethodGet, "account/stp_groups", params, nil, &resp)
}

// GetUsersInSTPUserGroup query users in the STP user group
// Only the main account that created this STP group can query the account ID list in the current STP group
func (e *Exchange) GetUsersInSTPUserGroup(ctx context.Context, stpID string) ([]*STPUserGroupMember, error) {
	if stpID == "" {
		return nil, errSTPGroupIDRequired
	}
	var resp []*STPUserGroupMember
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotAccountsEPL, http.MethodGet, "account/stp_groups/"+stpID+"/users", nil, nil, &resp)
}

// AddUsersToSTPUserGroup add users to the STP user group
// Only the main account that created this STP group can add users to the STP user group
// Only accounts under the current main account are allowed, cross-main account is not permitted
func (e *Exchange) AddUsersToSTPUserGroup(ctx context.Context, stpID string, usersID []uint64) ([]*STPUserGroupMember, error) {
	if stpID == "" {
		return nil, errSTPGroupIDRequired
	}
	if len(usersID) == 0 {
		return nil, errUserIDRequired
	}
	var resp []*STPUserGroupMember
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotAccountsEPL, http.MethodPost, "account/stp_groups/"+stpID+"/users", nil, usersID, &resp)
}

// DeleteUserFromSTPUserGroup delete users from the STP user group
// Only the main account that created this STP group is allowed to delete users from the STP user group
func (e *Exchange) DeleteUserFromSTPUserGroup(ctx context.Context, stpID string, userID uint64) ([]*STPUserGroupMember, error) {
	if stpID == "" {
		return nil, errSTPGroupIDRequired
	}
	if userID == 0 {
		return nil, errUserIDRequired
	}
	params := url.Values{}
	params.Set("user_id", strconv.FormatUint(userID, 10))
	var resp []*STPUserGroupMember
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotAccountsEPL, http.MethodDelete, "account/stp_groups/"+stpID+"/users", params, nil, &resp)
}

// ConfigureGTFeeDeduction configure GT fee deduction
// enable or disable GT fee deduction for the current account
func (e *Exchange) ConfigureGTFeeDeduction(ctx context.Context, setEnabled bool) error {
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotAccountsEPL, http.MethodPost, "account/debit_fee", nil, &map[string]bool{"enabled": setEnabled}, nil)
}

// GetGTFeeDeductionConfiguration query GT fee deduction configuration
// Query the GT fee deduction configuration for the current account
func (e *Exchange) GetGTFeeDeductionConfiguration(ctx context.Context) (bool, error) {
	var resp struct {
		Enabled bool `json:"enabled"`
	}
	return resp.Enabled, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotAccountsEPL, http.MethodGet, "account/debit_fee", nil, nil, &resp)
}

// PlaceMultiCollateralLoanOrder place multi-currency collateral order
func (e *Exchange) PlaceMultiCollateralLoanOrder(ctx context.Context, arg *MultiCollateralLoanOrderParam) (orderID uint64, err error) {
	if err := common.NilGuard(arg); err != nil {
		return 0, err
	}
	if arg.BorrowCurrency.IsEmpty() {
		return 0, currency.ErrCurrencyCodeEmpty
	}
	if arg.BorrowAmount <= 0 {
		return 0, order.ErrAmountIsInvalid
	}
	var resp struct {
		OrderID uint64 `json:"order_id"`
	}
	return resp.OrderID, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotAccountsEPL, http.MethodPost, "loan/multi_collateral/orders", nil, arg, &resp)
}

// GetOrderDetails query order details
func (e *Exchange) GetOrderDetails(ctx context.Context, orderID string) (*MultiCollateralLoanOrderDetail, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp *MultiCollateralLoanOrderDetail
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotAccountsEPL, http.MethodGet, "loan/multi_collateral/orders/"+orderID, nil, nil, &resp)
}

// RepayMultiCollateraLoan multi-currency collateral repayment
func (e *Exchange) RepayMultiCollateraLoan(ctx context.Context, arg *MultiCollateralLoanRepaymentParams) (*MultiCollateralLoanRepayment, error) {
	if arg.OrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if len(arg.RepayItems) == 0 {
		return nil, currency.ErrCurrencyNotSupported
	}
	var resp *MultiCollateralLoanRepayment
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotAccountsEPL, http.MethodPost, "loan/multi_collateral/repay", nil, nil, &resp)
}

// GetMultiCurrencyCollateralRepaymentRecords query multi-currency collateral repayment records
func (e *Exchange) GetMultiCurrencyCollateralRepaymentRecords(ctx context.Context, operationType string, borrowCurrency currency.Code, page, limit uint64, from, to time.Time) ([]*MultiCurrencyCollateralRepayment, error) {
	if operationType == "" {
		return nil, errLoanTypeIsRequired
	}
	if !from.IsZero() && !to.IsZero() {
		if err := common.StartEndTimeCheck(from, to); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("type", operationType)
	if !borrowCurrency.IsEmpty() {
		params.Set("borrow_currency", borrowCurrency.String())
	}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.UnixMilli(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.UnixMilli(), 10))
	}
	var resp []*MultiCurrencyCollateralRepayment
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotAccountsEPL, http.MethodGet, "loan/multi_collateral/repay", params, nil, &resp)
}

// AddOrWithdrawCollateral add or withdraw collateral
func (e *Exchange) AddOrWithdrawCollateral(ctx context.Context, arg *AddOrWithdrawCollateralParams) (*CollateralAddOrRemoveResponse, error) {
	if arg.OrderID == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	if arg.OperationType == "" {
		return nil, errLoanTypeIsRequired
	}
	var resp *CollateralAddOrRemoveResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotAccountsEPL, http.MethodPost, "loan/multi_collateral/mortgage", nil, arg, &resp)
}

// GetMultiCollateralAdjustmentRecords retrieves a multi-collateral adjustment records
func (e *Exchange) GetMultiCollateralAdjustmentRecords(ctx context.Context, page, limit uint64, from, to time.Time, collateralCcy currency.Code) ([]*MultiCollateralAdjustmentRecord, error) {
	if !from.IsZero() && !to.IsZero() {
		if err := common.StartEndTimeCheck(from, to); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if !collateralCcy.IsEmpty() {
		params.Set("collateral_currency", collateralCcy.String())
	}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.UnixMilli(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.UnixMilli(), 10))
	}
	var resp []*MultiCollateralAdjustmentRecord
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodGet, "loan/multi_collateral/mortgage", params, nil, &resp)
}

// ------------------------ Broker Rebate Endpoints ------------------------

// GetBrokerTransactionHistory retrieves broker obtains transaction history of recommended users
// Record query time range cannot exceed 30 days
func (e *Exchange) GetBrokerTransactionHistory(ctx context.Context, currencyPair currency.Pair, userID uint64, from, to time.Time, offset, limit int) (*BrokerRebateHistory, error) {
	if !from.IsZero() && !to.IsZero() {
		if err := common.StartEndTimeCheck(from, to); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if !currencyPair.IsEmpty() {
		params.Set("currency_pair", currencyPair.String())
	}
	if userID != 0 {
		params.Set("user_id", strconv.FormatUint(userID, 10))
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.UnixMilli(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		params.Set("offset", strconv.Itoa(offset))
	}
	var resp *BrokerRebateHistory
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotAccountsEPL, http.MethodGet, "rebate/agency/transaction_history", params, nil, &resp)
}

// GetBrokerRebateHistory broker obtains rebate history of recommended users
// Record query time range cannot exceed 30 days
func (e *Exchange) GetBrokerRebateHistory(ctx context.Context, ccy currency.Code, userID uint64, from, to time.Time, offset, limit int) (*BrokerRebateHistory, error) {
	if !from.IsZero() && !to.IsZero() {
		if err := common.StartEndTimeCheck(from, to); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	if userID != 0 {
		params.Set("user_id", strconv.FormatUint(userID, 10))
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.UnixMilli(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		params.Set("offset", strconv.Itoa(offset))
	}
	var resp *BrokerRebateHistory
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotAccountsEPL, http.MethodGet, "rebate/agency/commission_history", params, nil, &resp)
}

// GetPartnerRebateRecordsRecommendedUsers partner obtains rebate records of recommended users
// Record query time range cannot exceed 30 days
func (e *Exchange) GetPartnerRebateRecordsRecommendedUsers(ctx context.Context, ccy currency.Code, from, to time.Time, userID uint64, limit, offset uint8) (*UsersRebateRecords, error) {
	if !from.IsZero() && !to.IsZero() {
		if err := common.StartEndTimeCheck(from, to); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	if userID != 0 {
		params.Set("user_id", strconv.FormatUint(userID, 10))
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.UnixMilli(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(int(limit)))
	}
	if offset > 0 {
		params.Set("offset", strconv.Itoa(int(offset)))
	}
	var resp *UsersRebateRecords
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotAccountsEPL, http.MethodGet, "rebate/partner/commission_history", params, nil, &resp)
}

// GetPartnerSubordinateList partner subordinate list
// Including sub-agents, direct customers, and indirect customers
func (e *Exchange) GetPartnerSubordinateList(ctx context.Context, userID uint64, offset, limit int64) (*PartnerSubordinateList, error) {
	params := url.Values{}
	if userID != 0 {
		params.Set("user_id", strconv.FormatUint(userID, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatInt(offset, 10))
	}
	var resp *PartnerSubordinateList
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotAccountsEPL, http.MethodGet, "rebate/partner/sub_list", params, nil, &resp)
}

// BrokerObtainsUserRebateRecords broker obtains user's rebate records
// Record query time range cannot exceed 30 days
func (e *Exchange) BrokerObtainsUserRebateRecords(ctx context.Context, userID uint64, from, to time.Time, limit, offset int64) (*BrokerCommissionHistory, error) {
	if !from.IsZero() && !to.IsZero() {
		if err := common.StartEndTimeCheck(from, to); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if userID != 0 {
		params.Set("user_id", strconv.FormatUint(userID, 10))
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.UnixMilli(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatInt(offset, 10))
	}
	var resp *BrokerCommissionHistory
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotAccountsEPL, http.MethodGet, "rebate/broker/commission_history", params, nil, &resp)
}

// GetRebateBrokerTransactionHistory retrieves broker obtains user's trading history
// Record query time range cannot exceed 30 days
func (e *Exchange) GetRebateBrokerTransactionHistory(ctx context.Context, userID uint64, from, to time.Time, limit, offset int64) (*BrokerRebateUserTradingHistory, error) {
	if !from.IsZero() && !to.IsZero() {
		if err := common.StartEndTimeCheck(from, to); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if userID != 0 {
		params.Set("user_id", strconv.FormatUint(userID, 10))
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.UnixMilli(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.UnixMilli(), 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatInt(offset, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *BrokerRebateUserTradingHistory
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotAccountsEPL, http.MethodGet, "rebate/broker/transaction_history", params, nil, &resp)
}

// GetUserRebateInformation retrieves user obtains rebate information
func (e *Exchange) GetUserRebateInformation(ctx context.Context) (uint64, error) {
	var resp struct {
		InviteUID uint64 `json:"invite_uid"`
	}
	return resp.InviteUID, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotAccountsEPL, http.MethodGet, "rebate/user/info", nil, nil, &resp)
}

// GetUserSubordinateRelationship retrieves user subordinate relationships
func (e *Exchange) GetUserSubordinateRelationship(ctx context.Context, userIDList []string) (*UserRebaseSubRelation, error) {
	if len(userIDList) == 0 {
		return nil, errUserIDRequired
	}
	params := url.Values{}
	params.Set("user_id_list", strings.Join(userIDList, ","))
	var resp *UserRebaseSubRelation
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotAccountsEPL, http.MethodGet, "rebate/user/sub_relation", params, nil, &resp)
}

// validate validates the ContractOrderCreateParams
func (c *ContractOrderCreateParams) validate(isRest bool) error {
	if err := common.NilGuard(c); err != nil {
		return err
	}
	if c.Contract.IsEmpty() {
		return currency.ErrCurrencyPairEmpty
	}
	if c.Size == 0 && c.AutoSize == "" {
		return errInvalidOrderSize
	}
	if c.TimeInForce != "" {
		if _, err := timeInForceFromString(c.TimeInForce); err != nil {
			return err
		}
	}
	if c.Price == 0 && c.TimeInForce != iocTIF && c.TimeInForce != fokTIF {
		return fmt.Errorf("%w: %q; only 'ioc' and 'fok' allowed for market order", order.ErrUnsupportedTimeInForce, c.TimeInForce)
	}
	if c.Text != "" && !strings.HasPrefix(c.Text, "t-") {
		return errInvalidOrderText
	}
	if c.AutoSize != "" {
		if c.AutoSize != "close_long" && c.AutoSize != "close_short" {
			return fmt.Errorf("%w: %q", errInvalidAutoSize, c.AutoSize)
		}
	}
	// REST requests require a settlement currency, but it can be anything
	// Websocket requests may have an empty settlement currency, or it must be BTC or USDT
	if (isRest && c.Settle.IsEmpty()) ||
		(!isRest && !c.Settle.IsEmpty() && !c.Settle.Equal(currency.BTC) && !c.Settle.Equal(currency.USDT)) {
		return currency.ErrCurrencyCodeEmpty
	}
	return nil
}

// SendAuthenticatedHTTPRequest sends authenticated requests to the Gateio API
// To use this you must setup an APIKey and APISecret from the exchange
func (e *Exchange) SendAuthenticatedHTTPRequest(ctx context.Context, ep exchange.URL, epl request.EndpointLimit, method, endpoint string, param url.Values, data, result any) error {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	ePoint, err := e.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	var intermediary json.RawMessage
	if err := e.SendPayload(ctx, epl, func() (*request.Item, error) {
		headers := make(map[string]string)
		urlPath := endpoint
		timestamp := time.Now()
		var paramValue string
		if param != nil {
			paramValue = param.Encode()
		}
		var sig string
		sig, err = e.GenerateSignature(creds.Secret, method, "/"+gateioAPIVersion+endpoint, paramValue, data, timestamp)
		if err != nil {
			return nil, err
		}
		headers["Content-Type"] = "application/json"
		headers["KEY"] = creds.Key
		headers["TIMESTAMP"] = strconv.FormatInt(timestamp.Unix(), 10)
		headers["Accept"] = "application/json"
		headers["SIGN"] = sig
		urlPath = ePoint + gateioAPIVersion + urlPath
		if param != nil {
			urlPath = common.EncodeURLValues(urlPath, param)
		}
		var payload string
		if data != nil {
			var byteData []byte
			byteData, err = json.Marshal(data)
			if err != nil {
				return nil, err
			}
			payload = string(byteData)
		}
		return &request.Item{
			Method:                 method,
			Path:                   urlPath,
			Headers:                headers,
			Body:                   strings.NewReader(payload),
			Result:                 &intermediary,
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}, nil
	}, request.AuthenticatedRequest); err != nil {
		return err
	}
	if result == nil {
		return nil
	}
	var errCap struct {
		Label   string `json:"label"`
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(intermediary, &errCap); err == nil && errCap.Code != "" {
		return fmt.Errorf("%s auth request error, code: %s message: %s",
			e.Name,
			errCap.Label,
			errCap.Message)
	}
	if err := json.Unmarshal(intermediary, result); err != nil {
		return fmt.Errorf("%w: %w", request.ErrAuthRequestFailed, err)
	}
	if errType, ok := result.(interface{ Error() error }); ok && errType.Error() != nil {
		return fmt.Errorf("%w: %w", request.ErrAuthRequestFailed, errType.Error())
	}
	return nil
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (e *Exchange) SendHTTPRequest(ctx context.Context, ep exchange.URL, epl request.EndpointLimit, path string, result any) error {
	endpoint, err := e.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	return e.SendPayload(ctx, epl, func() (*request.Item, error) {
		return &request.Item{
			Method:                 http.MethodGet,
			Path:                   endpoint + gateioAPIVersion + path,
			Result:                 result,
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}, nil
	}, request.UnauthenticatedRequest)
}

// ----- Earning endpoints -----------------

// GetLendingCurrencyList retrieves lending currency list
func (e *Exchange) GetLendingCurrencyList(ctx context.Context) ([]*LendingCurrencyDetail, error) {
	var resp []*LendingCurrencyDetail
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, "earn/uni/currencies", &resp)
}

// GetLendingCurrencyDetail retrieves a single lending currency detail
func (e *Exchange) GetLendingCurrencyDetail(ctx context.Context, ccy currency.Code) (*LendingCurrencyDetail, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp *LendingCurrencyDetail
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, "earn/uni/currencies/"+ccy.Item.Lower, &resp)
}

// CreateLendingOrRedemption creates lending or redemption
// possible values of lending, redemption type are 'lend'  and 'redeem' respectively.
func (e *Exchange) CreateLendingOrRedemption(ctx context.Context, arg *LendingOrRedemptionRequest) error {
	if err := common.NilGuard(arg); err != nil {
		return err
	}
	if arg.Currency.IsEmpty() {
		return currency.ErrCurrencyCodeEmpty
	}
	if arg.Amount <= 0 {
		return fmt.Errorf("%w: minimum lending or redemption amount is required", limits.ErrAmountBelowMin)
	}
	if arg.Type == "" {
		return errLoanTypeIsRequired
	}
	if arg.MinRate <= 0 {
		return fmt.Errorf("%w: minimum interest rate is required", limits.ErrAmountBelowMin)
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodPost, "earn/uni/lends", nil, arg, nil)
}

// GetUserLendingOrderList get user's lending order list
func (e *Exchange) GetUserLendingOrderList(ctx context.Context, ccy currency.Code, page, limit uint64) ([]*LendOrderDetail, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []*LendOrderDetail
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodGet, "earn/uni/lends", params, nil, &resp)
}

// AmendUserLendingInformation amends user lending information
func (e *Exchange) AmendUserLendingInformation(ctx context.Context, ccy currency.Code, minRate float64) error {
	if ccy.IsEmpty() {
		return currency.ErrCurrencyCodeEmpty
	}
	if minRate <= 0 {
		return fmt.Errorf("%w: minimum interest rate is required", limits.ErrAmountBelowMin)
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodPatch, "earn/uni/lends", nil, map[string]string{
		"currency": ccy.String(),
		"min_rate": strconv.FormatFloat(minRate, 'f', 10, 64),
	}, nil)
}

// GetLendingTransactionRecords retrieves lending transaction records
func (e *Exchange) GetLendingTransactionRecords(ctx context.Context, ccy currency.Code, page, limit uint64, from, to time.Time, operationType string) ([]*LendingTransactionRecord, error) {
	if !from.IsZero() && !to.IsZero() {
		if err := common.StartEndTimeCheck(from, to); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.UnixMilli(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.UnixMilli(), 10))
	}
	if operationType != "" {
		params.Set("type", operationType)
	}
	var resp []*LendingTransactionRecord
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodGet, "earn/uni/lend_records", params, nil, &resp)
}

// GetUserTotalInterestIncomePerCurrency retrieves user's total interest income for specified currency
func (e *Exchange) GetUserTotalInterestIncomePerCurrency(ctx context.Context, ccy currency.Code) (*CurrencyAndInterestIncome, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	var resp *CurrencyAndInterestIncome
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodGet, "earn/uni/interests/"+ccy.Item.Lower, params, nil, &resp)
}

// GetUserDividendRecords retrieves user dividend records
func (e *Exchange) GetUserDividendRecords(ctx context.Context, ccy currency.Code, page, limit uint64, from, to time.Time) ([]*UserDividendRecords, error) {
	if !from.IsZero() && !to.IsZero() {
		if err := common.StartEndTimeCheck(from, to); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.UnixMilli(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.UnixMilli(), 10))
	}
	var resp []*UserDividendRecords
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodGet, "earn/uni/interest_records", params, nil, &resp)
}

// GetCurrencyInterestCompoundingStatus retrieves a currency code interest compounding status
func (e *Exchange) GetCurrencyInterestCompoundingStatus(ctx context.Context, ccy currency.Code) (*CurrencyInterestStatus, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp *CurrencyInterestStatus
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodGet, "earn/uni/interest_status/"+ccy.Lower().String(), nil, nil, &resp)
}

// GetUniLoanCurrencyAnnualizedTrendChart retrieves UniLoan currency annualized trend chart
func (e *Exchange) GetUniLoanCurrencyAnnualizedTrendChart(ctx context.Context, from, to time.Time, ccy currency.Code) ([]*UniLoanAssetData, error) {
	if !from.IsZero() && !to.IsZero() {
		if err := common.StartEndTimeCheck(from, to); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("asset", ccy.String())
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.UnixMilli(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.UnixMilli(), 10))
	}
	var resp []*UniLoanAssetData
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodGet, "earn/uni/chart", params, nil, &resp)
}

// GetCurrencyEstimatedAnnualizedInterestRate retrieves user's account estimated annulaized interest rate for each currency
func (e *Exchange) GetCurrencyEstimatedAnnualizedInterestRate(ctx context.Context) ([]*CurrencyEstimatedAnnualInterestRate, error) {
	var resp []*CurrencyEstimatedAnnualInterestRate
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodGet, "earn/uni/rate", nil, nil, &resp)
}

// -------------- Collateral Loan endpoints ------------------------------

// PlaceCollateralLoanOrder places a collateral loan order detail
func (e *Exchange) PlaceCollateralLoanOrder(ctx context.Context, arg *PlaceColateralLoanRequest) (*OrderIDResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.CollateralCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w: collateral currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if arg.CollateralAmount <= 0 {
		return nil, fmt.Errorf("%w: collateral asset amount is required", limits.ErrAmountBelowMin)
	}
	if arg.BorrowCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w: borrow currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if arg.BorrowAmount <= 0 {
		return nil, fmt.Errorf("%w: borrow asset amount is required", limits.ErrAmountBelowMin)
	}
	var resp *OrderIDResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodPost, "loan/collateral/orders", nil, arg, &resp)
}

// GetCollateralLoanOrderList retrieves collateral loan order list
func (e *Exchange) GetCollateralLoanOrderList(ctx context.Context, page, limit uint64, collateralCcy, borrowCcy currency.Code) ([]*CollateralLoanOrderDetail, error) {
	params := url.Values{}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if !collateralCcy.IsEmpty() {
		params.Set("collateral_currency", collateralCcy.String())
	}
	if !borrowCcy.IsEmpty() {
		params.Set("borrow_currency", borrowCcy.String())
	}
	var resp []*CollateralLoanOrderDetail
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodGet, "loan/collateral/orders", params, nil, &resp)
}

// GetSingleCollateralOrderDetail retrieves a single collateral loan order by ID
func (e *Exchange) GetSingleCollateralOrderDetail(ctx context.Context, orderID string) (*CollateralLoanOrderDetail, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp *CollateralLoanOrderDetail
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodGet, "loan/collateral/orders/"+orderID, nil, nil, &resp)
}

// RepayCollateralLoan repays a loan backed by a collateral
func (e *Exchange) RepayCollateralLoan(ctx context.Context, arg *CollateralLoanRepayRequest) (*CollateralLoanRepaymentResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.OrderID == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	if arg.RepayAmount <= 0 {
		return nil, fmt.Errorf("%w: repayment amount is required", limits.ErrAmountBelowMin)
	}
	var resp *CollateralLoanRepaymentResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodPost, "loan/collateral/repay", nil, arg, &resp)
}

// GetCollateralLoanRepaymentRecords retrieves collateral loan repayment records
func (e *Exchange) GetCollateralLoanRepaymentRecords(ctx context.Context, operationType string, page, limit uint64, collateralCcy, borrowCcy currency.Code, from, to time.Time) ([]*CollateralLoanRepaymentOrderDetail, error) {
	if operationType == "" {
		return nil, errOperationTypeRequired
	}
	if !from.IsZero() && !to.IsZero() {
		if err := common.StartEndTimeCheck(from, to); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("source", operationType)
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.UnixMilli(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.UnixMilli(), 10))
	}
	if !collateralCcy.IsEmpty() {
		params.Set("collateral_currency", collateralCcy.String())
	}
	if !borrowCcy.IsEmpty() {
		params.Set("borrow_currency", borrowCcy.String())
	}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []*CollateralLoanRepaymentOrderDetail
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodGet, "loan/collateral/repay_records", params, nil, &resp)
}

// IncreaseOrRedeemCollateral increases or redeem collateral
func (e *Exchange) IncreaseOrRedeemCollateral(ctx context.Context, arg *IncreaseOrRedeemCollateralRequest) error {
	if err := common.NilGuard(arg); err != nil {
		return err
	}
	if arg.OrderID == 0 {
		return order.ErrOrderIDNotSet
	}
	if arg.CollateralCurrency.IsEmpty() {
		return fmt.Errorf("%w: collateral currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if arg.CollateralAmount <= 0 {
		return fmt.Errorf("%w: collateral asset amount is required", limits.ErrAmountBelowMin)
	}
	if arg.OperationType == "" {
		return errOperationTypeRequired
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodPost, "loan/collateral/collaterals", nil, arg, nil)
}

// GetCollateralAdjustmentRecords retrieves collateral adjustment records
func (e *Exchange) GetCollateralAdjustmentRecords(ctx context.Context, page, limit uint64, collateralCcy, borrowCcy currency.Code, from, to time.Time) ([]*CollateralAdjustmentRecord, error) {
	if !from.IsZero() && !to.IsZero() {
		if err := common.StartEndTimeCheck(from, to); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.UnixMilli(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.UnixMilli(), 10))
	}
	if !collateralCcy.IsEmpty() {
		params.Set("collateral_currency", collateralCcy.String())
	}
	if !borrowCcy.IsEmpty() {
		params.Set("borrow_currency", borrowCcy.String())
	}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []*CollateralAdjustmentRecord
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodGet, "loan/collateral/collaterals", params, nil, &resp)
}

// GetUserTotalBorrowingAndCollateralAmount query user's total borrowing and collateral amount
func (e *Exchange) GetUserTotalBorrowingAndCollateralAmount(ctx context.Context) (*TotalBorrowingAndCollateralAmount, error) {
	var resp *TotalBorrowingAndCollateralAmount
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodGet, "loan/collateral/total_amount", nil, nil, &resp)
}

// GetUserCollateralizationRatioAndRemainingBorrowables retrieves user's collateralization ratio and remaining borrowable currencies
func (e *Exchange) GetUserCollateralizationRatioAndRemainingBorrowables(ctx context.Context, collateralCcy, borrowCcy currency.Code) (*UserCollateralRatioAndRemainingBorrowable, error) {
	if collateralCcy.IsEmpty() {
		return nil, fmt.Errorf("%w: collateral currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if borrowCcy.IsEmpty() {
		return nil, fmt.Errorf("%w: borrow currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	params.Set("collateral_currency", collateralCcy.String())
	params.Set("borrow_currency", borrowCcy.String())
	var resp *UserCollateralRatioAndRemainingBorrowable
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodGet, "loan/collateral/ltv", params, nil, &resp)
}

// GetSupportedBorrowingAndCollateralCurrencies retrieves supported borrowing and collateral currencies
func (e *Exchange) GetSupportedBorrowingAndCollateralCurrencies(ctx context.Context, loanCurrency currency.Code) ([]*SupportedBorrowingAndCollateralCurrencies, error) {
	if loanCurrency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("loan_currency", loanCurrency.String())
	var resp []*SupportedBorrowingAndCollateralCurrencies
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodGet, "loan/collateral/currencies", params, nil, &resp)
}
