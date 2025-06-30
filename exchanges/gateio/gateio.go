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
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const (
	gateioTradeURL                      = "https://api.gateio.ws/" + gateioAPIVersion
	gateioFuturesTestnetTrading         = "https://fx-api-testnet.gateio.ws"
	gateioFuturesLiveTradingAlternative = "https://fx-api.gateio.ws/" + gateioAPIVersion
	gateioAPIVersion                    = "api/v4/"
	tradeBaseURL                        = "https://www.gate.io/"

	// SubAccount Endpoints
	subAccounts = "sub_accounts"

	// Spot
	gateioSpotCurrencies                                 = "spot/currencies"
	gateioSpotCurrencyPairs                              = "spot/currency_pairs"
	gateioSpotTickers                                    = "spot/tickers"
	gateioSpotOrderbook                                  = "spot/order_book"
	gateioSpotMarketTrades                               = "spot/trades"
	gateioSpotCandlesticks                               = "spot/candlesticks"
	gateioSpotFeeRate                                    = "spot/fee"
	gateioSpotAccounts                                   = "spot/accounts"
	gateioUnifiedAccounts                                = "unified/accounts"
	gateioSpotBatchOrders                                = "spot/batch_orders"
	gateioSpotOpenOrders                                 = "spot/open_orders"
	gateioSpotClosePositionWhenCrossCurrencyDisabledPath = "spot/cross_liquidate_orders"
	gateioSpotOrders                                     = "spot/orders"
	gateioSpotCancelBatchOrders                          = "spot/cancel_batch_orders"
	gateioSpotMyTrades                                   = "spot/my_trades"
	gateioSpotServerTime                                 = "spot/time"
	gateioSpotAllCountdown                               = "spot/countdown_cancel_all"
	gateioSpotPriceOrders                                = "spot/price_orders"

	// Wallets
	walletCurrencyChain                 = "wallet/currency_chains"
	walletDepositAddress                = "wallet/deposit_address"
	walletWithdrawals                   = "wallet/withdrawals"
	walletDeposits                      = "wallet/deposits"
	walletTransfer                      = "wallet/transfers"
	walletSubAccountTransfer            = "wallet/sub_account_transfers"
	walletInterSubAccountTransfer       = "wallet/sub_account_to_sub_account"
	walletWithdrawStatus                = "wallet/withdraw_status"
	walletSubAccountBalance             = "wallet/sub_account_balances"
	walletSubAccountMarginBalance       = "wallet/sub_account_margin_balances"
	walletSubAccountFuturesBalance      = "wallet/sub_account_futures_balances"
	walletSubAccountCrossMarginBalances = "wallet/sub_account_cross_margin_balances"
	walletSavedAddress                  = "wallet/saved_address"
	walletTradingFee                    = "wallet/fee"
	walletTotalBalance                  = "wallet/total_balance"

	// Margin
	gateioMarginCurrencyPairs     = "margin/currency_pairs"
	gateioMarginFundingBook       = "margin/funding_book"
	gateioMarginAccount           = "margin/accounts"
	gateioMarginAccountBook       = "margin/account_book"
	gateioMarginFundingAccounts   = "margin/funding_accounts"
	gateioMarginLoans             = "margin/loans"
	gateioMarginMergedLoans       = "margin/merged_loans"
	gateioMarginLoanRecords       = "margin/loan_records"
	gateioMarginAutoRepay         = "margin/auto_repay"
	gateioMarginTransfer          = "margin/transferable"
	gateioMarginBorrowable        = "margin/borrowable"
	gateioCrossMarginCurrencies   = "margin/cross/currencies"
	gateioCrossMarginAccounts     = "margin/cross/accounts"
	gateioCrossMarginAccountBook  = "margin/cross/account_book"
	gateioCrossMarginLoans        = "margin/cross/loans"
	gateioCrossMarginRepayments   = "margin/cross/repayments"
	gateioCrossMarginTransferable = "margin/cross/transferable"
	gateioCrossMarginBorrowable   = "margin/cross/borrowable"

	// Options
	gateioOptionUnderlyings            = "options/underlyings"
	gateioOptionExpiration             = "options/expirations"
	gateioOptionContracts              = "options/contracts"
	gateioOptionSettlement             = "options/settlements"
	gateioOptionMySettlements          = "options/my_settlements"
	gateioOptionsOrderbook             = "options/order_book"
	gateioOptionsTickers               = "options/tickers"
	gateioOptionCandlesticks           = "options/candlesticks"
	gateioOptionUnderlyingCandlesticks = "options/underlying/candlesticks"
	gateioOptionsTrades                = "options/trades"
	gateioOptionAccounts               = "options/accounts"
	gateioOptionsAccountbook           = "options/account_book"
	gateioOptionsPosition              = "options/positions"
	gateioOptionsPositionClose         = "options/position_close"
	gateioOptionsOrders                = "options/orders"
	gateioOptionsMyTrades              = "options/my_trades"

	// Flash Swap
	gateioFlashSwapCurrencies    = "flash_swap/currencies"
	gateioFlashSwapOrders        = "flash_swap/orders"
	gateioFlashSwapOrdersPreview = "flash_swap/orders/preview"

	futuresPath      = "futures/"
	deliveryPath     = "delivery/"
	ordersPath       = "/orders"
	positionsPath    = "/positions/"
	subAccountsPath  = "sub_accounts/"
	priceOrdersPaths = "/price_orders"

	// Withdrawals
	withdrawal = "withdrawals"
)

const (
	utc0TimeZone = "utc0"
	utc8TimeZone = "utc8"
)

var (
	errEmptyOrInvalidSettlementCurrency = errors.New("empty or invalid settlement currency")
	errInvalidOrMissingContractParam    = errors.New("invalid or empty contract")
	errNoValidResponseFromServer        = errors.New("no valid response from server")
	errInvalidUnderlying                = errors.New("missing underlying")
	errInvalidOrderSize                 = errors.New("invalid order size")
	errInvalidOrderID                   = errors.New("invalid order id")
	errInvalidAmount                    = errors.New("invalid amount")
	errInvalidSubAccount                = errors.New("invalid or empty subaccount")
	errInvalidTransferDirection         = errors.New("invalid transfer direction")
	errDifferentAccount                 = errors.New("account type must be identical for all orders")
	errInvalidPrice                     = errors.New("invalid price")
	errNoValidParameterPassed           = errors.New("no valid parameter passed")
	errInvalidCountdown                 = errors.New("invalid countdown, Countdown time, in seconds At least 5 seconds, 0 means cancel the countdown")
	errInvalidOrderStatus               = errors.New("invalid order status")
	errInvalidLoanSide                  = errors.New("invalid loan side, only 'lend' and 'borrow'")
	errInvalidLoanID                    = errors.New("missing loan ID")
	errInvalidRepayMode                 = errors.New("invalid repay mode specified, must be 'all' or 'partial'")
	errMissingPreviewID                 = errors.New("missing required parameter: preview_id")
	errChangeHasToBePositive            = errors.New("change has to be positive")
	errInvalidLeverageValue             = errors.New("invalid leverage value")
	errInvalidRiskLimit                 = errors.New("new position risk limit")
	errInvalidCountTotalValue           = errors.New("invalid \"count_total\" value, supported \"count_total\" values are 0 and 1")
	errInvalidAutoSizeValue             = errors.New("invalid \"auto_size\" value, only \"close_long\" and \"close_short\" are supported")
	errTooManyOrderRequest              = errors.New("too many order creation request")
	errInvalidTimeout                   = errors.New("invalid timeout, should be in seconds At least 5 seconds, 0 means cancel the countdown")
	errNoTickerData                     = errors.New("no ticker data available")
	errNilArgument                      = errors.New("null argument")
	errInvalidTimezone                  = errors.New("invalid timezone")
	errMultipleOrders                   = errors.New("multiple orders passed")
	errMissingWithdrawalID              = errors.New("missing withdrawal ID")
	errInvalidSubAccountUserID          = errors.New("sub-account user id is required")
	errInvalidSettlementQuote           = errors.New("symbol quote currency does not match asset settlement currency")
	errInvalidSettlementBase            = errors.New("symbol base currency does not match asset settlement currency")
	errMissingAPIKey                    = errors.New("missing API key information")
	errInvalidTextValue                 = errors.New("invalid text value, requires prefix `t-`")
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

// Gateio is the overarching type across this package
type Gateio struct {
	Counter common.Counter // Must be first	due to alignment requirements
	exchange.Base
	wsOBUpdateMgr *wsOBUpdateManager
}

// ***************************************** SubAccounts ********************************

// CreateNewSubAccount creates a new sub-account
func (g *Gateio) CreateNewSubAccount(ctx context.Context, arg SubAccountParams) (*SubAccount, error) {
	if arg.LoginName == "" {
		return nil, errors.New("login name can not be empty")
	}
	var response *SubAccount
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, subAccountEPL, http.MethodPost, subAccounts, nil, &arg, &response)
}

// GetSubAccounts retrieves list of sub-accounts for given account
func (g *Gateio) GetSubAccounts(ctx context.Context) ([]SubAccount, error) {
	var response []SubAccount
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, subAccountEPL, http.MethodGet, subAccounts, nil, nil, &response)
}

// GetSingleSubAccount retrieves a single sub-account for given account
func (g *Gateio) GetSingleSubAccount(ctx context.Context, userID string) (*SubAccount, error) {
	if userID == "" {
		return nil, errors.New("user ID can not be empty")
	}
	var response *SubAccount
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, subAccountEPL, http.MethodGet, subAccounts+"/"+userID, nil, nil, &response)
}

// CreateAPIKeysOfSubAccount creates a sub-account for the sub-account
//
// name: Permission name (all permissions will be removed if no value is passed)
// >> wallet: wallet, spot: spot/margin, futures: perpetual contract, delivery: delivery, earn: earn, options: options
func (g *Gateio) CreateAPIKeysOfSubAccount(ctx context.Context, arg CreateAPIKeySubAccountParams) (*CreateAPIKeyResponse, error) {
	if arg.SubAccountUserID == 0 {
		return nil, errInvalidSubAccountUserID
	}
	if arg.Body == nil {
		return nil, errors.New("sub-account key information is required")
	}
	var resp *CreateAPIKeyResponse
	return resp, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, subAccountEPL, http.MethodPost, subAccountsPath+strconv.FormatInt(arg.SubAccountUserID, 10)+"/keys", nil, &arg, &resp)
}

// GetAllAPIKeyOfSubAccount list all API Key of the sub-account
func (g *Gateio) GetAllAPIKeyOfSubAccount(ctx context.Context, userID int64) ([]CreateAPIKeyResponse, error) {
	var resp []CreateAPIKeyResponse
	return resp, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, subAccountEPL, http.MethodGet, subAccountsPath+strconv.FormatInt(userID, 10)+"/keys", nil, nil, &resp)
}

// UpdateAPIKeyOfSubAccount update API key of the sub-account
func (g *Gateio) UpdateAPIKeyOfSubAccount(ctx context.Context, subAccountAPIKey string, arg CreateAPIKeySubAccountParams) error {
	return g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, subAccountEPL, http.MethodPut, subAccountsPath+strconv.FormatInt(arg.SubAccountUserID, 10)+"/keys/"+subAccountAPIKey, nil, &arg, nil)
}

// GetAPIKeyOfSubAccount retrieves the API Key of the sub-account
func (g *Gateio) GetAPIKeyOfSubAccount(ctx context.Context, subAccountUserID int64, apiKey string) (*CreateAPIKeyResponse, error) {
	if subAccountUserID == 0 {
		return nil, errInvalidSubAccountUserID
	}
	if apiKey == "" {
		return nil, errMissingAPIKey
	}
	var resp *CreateAPIKeyResponse
	return resp, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, subAccountEPL, http.MethodGet, subAccountsPath+strconv.FormatInt(subAccountUserID, 10)+"/keys/"+apiKey, nil, nil, &resp)
}

// LockSubAccount locks the sub-account
func (g *Gateio) LockSubAccount(ctx context.Context, subAccountUserID int64) error {
	if subAccountUserID == 0 {
		return errInvalidSubAccountUserID
	}
	return g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, subAccountEPL, http.MethodPost, subAccountsPath+strconv.FormatInt(subAccountUserID, 10)+"/lock", nil, nil, nil)
}

// UnlockSubAccount locks the sub-account
func (g *Gateio) UnlockSubAccount(ctx context.Context, subAccountUserID int64) error {
	if subAccountUserID == 0 {
		return errInvalidSubAccountUserID
	}
	return g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, subAccountEPL, http.MethodPost, subAccountsPath+strconv.FormatInt(subAccountUserID, 10)+"/unlock", nil, nil, nil)
}

// *****************************************  Spot **************************************

// ListSpotCurrencies to retrieve detailed list of each currency.
func (g *Gateio) ListSpotCurrencies(ctx context.Context) ([]CurrencyInfo, error) {
	var resp []CurrencyInfo
	return resp, g.SendHTTPRequest(ctx, exchange.RestSpot, publicCurrenciesSpotEPL, gateioSpotCurrencies, &resp)
}

// GetCurrencyDetail details of a specific currency.
func (g *Gateio) GetCurrencyDetail(ctx context.Context, ccy currency.Code) (*CurrencyInfo, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp *CurrencyInfo
	return resp, g.SendHTTPRequest(ctx, exchange.RestSpot, publicCurrenciesSpotEPL, gateioSpotCurrencies+"/"+ccy.String(), &resp)
}

// ListSpotCurrencyPairs retrieve all currency pairs supported by the exchange.
func (g *Gateio) ListSpotCurrencyPairs(ctx context.Context) ([]CurrencyPairDetail, error) {
	var resp []CurrencyPairDetail
	return resp, g.SendHTTPRequest(ctx, exchange.RestSpot, publicListCurrencyPairsSpotEPL, gateioSpotCurrencyPairs, &resp)
}

// GetCurrencyPairDetail to get details of a specific order for spot/margin accounts.
func (g *Gateio) GetCurrencyPairDetail(ctx context.Context, currencyPair string) (*CurrencyPairDetail, error) {
	if currencyPair == "" {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var resp *CurrencyPairDetail
	return resp, g.SendHTTPRequest(ctx, exchange.RestSpot, publicCurrencyPairDetailSpotEPL, gateioSpotCurrencyPairs+"/"+currencyPair, &resp)
}

// GetTickers retrieve ticker information
// Return only related data if currency_pair is specified; otherwise return all of them
func (g *Gateio) GetTickers(ctx context.Context, currencyPair, timezone string) ([]Ticker, error) {
	params := url.Values{}
	if currencyPair != "" {
		params.Set("currency_pair", currencyPair)
	}
	if timezone != "" && timezone != utc8TimeZone && timezone != utc0TimeZone {
		return nil, errInvalidTimezone
	} else if timezone != "" {
		params.Set("timezone", timezone)
	}
	var tickers []Ticker
	return tickers, g.SendHTTPRequest(ctx, exchange.RestSpot, publicTickersSpotEPL, common.EncodeURLValues(gateioSpotTickers, params), &tickers)
}

// GetTicker retrieves a single ticker information for a currency pair.
func (g *Gateio) GetTicker(ctx context.Context, currencyPair, timezone string) (*Ticker, error) {
	if currencyPair == "" {
		return nil, currency.ErrCurrencyPairEmpty
	}
	tickers, err := g.GetTickers(ctx, currencyPair, timezone)
	if err != nil {
		return nil, err
	}
	if len(tickers) > 0 {
		return &tickers[0], err
	}
	return nil, fmt.Errorf("no ticker data found for currency pair %v", currencyPair)
}

// getIntervalString returns a string representation of the interval according to the Gateio exchange representation
func getIntervalString(interval kline.Interval) (string, error) {
	switch interval {
	case kline.ThousandMilliseconds:
		return "1000ms", nil
	case kline.OneDay:
		return "1d", nil
	case kline.SevenDay:
		return "7d", nil
	case kline.OneMonth:
		return "30d", nil
	case kline.TenMilliseconds, kline.TwentyMilliseconds, kline.HundredMilliseconds, kline.TwoHundredAndFiftyMilliseconds,
		kline.TenSecond, kline.ThirtySecond, kline.OneMin, kline.FiveMin, kline.FifteenMin, kline.ThirtyMin,
		kline.OneHour, kline.TwoHour, kline.FourHour, kline.EightHour, kline.TwelveHour:
		return interval.Short(), nil
	default:
		return "", fmt.Errorf("%q: %w", interval.String(), kline.ErrUnsupportedInterval)
	}
}

// GetIntervalFromString returns a kline.Interval representation of the interval string
func (g *Gateio) GetIntervalFromString(interval string) (kline.Interval, error) {
	switch interval {
	case "10s":
		return kline.TenSecond, nil
	case "30s":
		return kline.ThirtySecond, nil
	case "1m":
		return kline.OneMin, nil
	case "5m":
		return kline.FiveMin, nil
	case "15m":
		return kline.FifteenMin, nil
	case "30m":
		return kline.ThirtyMin, nil
	case "1h":
		return kline.OneHour, nil
	case "2h":
		return kline.TwoHour, nil
	case "4h":
		return kline.FourHour, nil
	case "8h":
		return kline.EightHour, nil
	case "12h":
		return kline.TwelveHour, nil
	case "1d":
		return kline.OneDay, nil
	case "7d":
		return kline.SevenDay, nil
	case "30d":
		return kline.OneMonth, nil
	case "100ms":
		return kline.HundredMilliseconds, nil
	case "1000ms":
		return kline.ThousandMilliseconds, nil
	default:
		return kline.Interval(0), kline.ErrInvalidInterval
	}
}

// GetOrderbook returns the orderbook data for a suppled currency pair
func (g *Gateio) GetOrderbook(ctx context.Context, pairString, interval string, limit uint64, withOrderbookID bool) (*Orderbook, error) {
	if pairString == "" {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("currency_pair", pairString)
	if interval != "" {
		params.Set("interval", interval)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	params.Set("with_id", strconv.FormatBool(withOrderbookID))
	var response *OrderbookData
	if err := g.SendHTTPRequest(ctx, exchange.RestSpot, publicOrderbookSpotEPL, common.EncodeURLValues(gateioSpotOrderbook, params), &response); err != nil {
		return nil, err
	}
	return response.MakeOrderbook(), nil
}

// GetMarketTrades retrieve market trades
func (g *Gateio) GetMarketTrades(ctx context.Context, pairString currency.Pair, limit uint64, lastID string, reverse bool, from, to time.Time, page uint64) ([]Trade, error) {
	params := url.Values{}
	if pairString.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params.Set("currency_pair", pairString.String())
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
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
	var response []Trade
	return response, g.SendHTTPRequest(ctx, exchange.RestSpot, publicMarketTradesSpotEPL, common.EncodeURLValues(gateioSpotMarketTrades, params), &response)
}

// GetCandlesticks retrieves market candlesticks.
func (g *Gateio) GetCandlesticks(ctx context.Context, currencyPair currency.Pair, limit uint64, from, to time.Time, interval kline.Interval) ([]Candlestick, error) {
	params := url.Values{}
	if currencyPair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params.Set("currency_pair", currencyPair.String())
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var err error
	if interval.Duration().Microseconds() != 0 {
		var intervalString string
		intervalString, err = getIntervalString(interval)
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
	var candles []Candlestick
	return candles, g.SendHTTPRequest(ctx, exchange.RestSpot, publicCandleStickSpotEPL, common.EncodeURLValues(gateioSpotCandlesticks, params), &candles)
}

// GetTradingFeeRatio retrieves user trading fee rates
func (g *Gateio) GetTradingFeeRatio(ctx context.Context, currencyPair currency.Pair) (*SpotTradingFeeRate, error) {
	params := url.Values{}
	if currencyPair.IsPopulated() {
		// specify a currency pair to retrieve precise fee rate
		params.Set("currency_pair", currencyPair.String())
	}
	var response *SpotTradingFeeRate
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotTradingFeeEPL, http.MethodGet, gateioSpotFeeRate, params, nil, &response)
}

// GetSpotAccounts retrieves spot account.
func (g *Gateio) GetSpotAccounts(ctx context.Context, ccy currency.Code) ([]SpotAccount, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	var response []SpotAccount
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotAccountsEPL, http.MethodGet, gateioSpotAccounts, params, nil, &response)
}

// GetUnifiedAccount retrieves unified account.
func (g *Gateio) GetUnifiedAccount(ctx context.Context, ccy currency.Code) (*UnifiedUserAccount, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	var response UnifiedUserAccount
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, privateUnifiedSpotEPL, http.MethodGet, gateioUnifiedAccounts, params, nil, &response)
}

// CreateBatchOrders Create a batch of orders Batch orders requirements: custom order field text is required At most 4 currency pairs,
// maximum 10 orders each, are allowed in one request No mixture of spot orders and margin orders, i.e. account must be identical for all orders
func (g *Gateio) CreateBatchOrders(ctx context.Context, args []CreateOrderRequest) ([]SpotOrder, error) {
	if len(args) > 10 {
		return nil, fmt.Errorf("%w only 10 orders are canceled at once", errMultipleOrders)
	}
	for x := range args {
		if (x != 0) && args[x-1].Account != args[x].Account {
			return nil, errDifferentAccount
		}
		if args[x].CurrencyPair.IsEmpty() {
			return nil, currency.ErrCurrencyPairEmpty
		}
		if args[x].Type != "limit" {
			return nil, errors.New("only order type limit is allowed")
		}
		args[x].Side = strings.ToLower(args[x].Side)
		if args[x].Side != "buy" && args[x].Side != "sell" {
			return nil, order.ErrSideIsInvalid
		}
		if !strings.EqualFold(args[x].Account, asset.Spot.String()) &&
			!strings.EqualFold(args[x].Account, asset.CrossMargin.String()) &&
			!strings.EqualFold(args[x].Account, asset.Margin.String()) {
			return nil, errors.New("only spot, margin, and cross_margin area allowed")
		}
		if args[x].Amount <= 0 {
			return nil, errInvalidAmount
		}
		if args[x].Price <= 0 {
			return nil, errInvalidPrice
		}
	}
	var response []SpotOrder
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotBatchOrdersEPL, http.MethodPost, gateioSpotBatchOrders, nil, &args, &response)
}

// GetSpotOpenOrders retrieves all open orders
// List open orders in all currency pairs.
// Note that pagination parameters affect record number in each currency pair's open order list. No pagination is applied to the number of currency pairs returned. All currency pairs with open orders will be returned.
// Spot and margin orders are returned by default. To list cross margin orders, account must be set to cross_margin
func (g *Gateio) GetSpotOpenOrders(ctx context.Context, page, limit uint64, isCrossMargin bool) ([]SpotOrdersDetail, error) {
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
	var response []SpotOrdersDetail
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotGetOpenOrdersEPL, http.MethodGet, gateioSpotOpenOrders, params, nil, &response)
}

// SpotClosePositionWhenCrossCurrencyDisabled set close position when cross-currency is disabled
func (g *Gateio) SpotClosePositionWhenCrossCurrencyDisabled(ctx context.Context, arg *ClosePositionRequestParam) (*SpotOrder, error) {
	if arg == nil {
		return nil, errNilArgument
	}
	if arg.CurrencyPair.IsInvalid() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if arg.Amount <= 0 {
		return nil, errInvalidAmount
	}
	if arg.Price <= 0 {
		return nil, errInvalidPrice
	}
	var response *SpotOrder
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotClosePositionEPL, http.MethodPost, gateioSpotClosePositionWhenCrossCurrencyDisabledPath, nil, &arg, &response)
}

// PlaceSpotOrder creates a spot order you can place orders with spot, margin or cross margin account through setting the accountfield.
// It defaults to spot, which means spot account is used to place orders.
func (g *Gateio) PlaceSpotOrder(ctx context.Context, arg *CreateOrderRequest) (*SpotOrder, error) {
	if arg == nil {
		return nil, errNilArgument
	}
	if arg.CurrencyPair.IsInvalid() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	arg.Side = strings.ToLower(arg.Side)
	if arg.Side != "buy" && arg.Side != "sell" {
		return nil, order.ErrSideIsInvalid
	}
	if !strings.EqualFold(arg.Account, asset.Spot.String()) &&
		!strings.EqualFold(arg.Account, asset.CrossMargin.String()) &&
		!strings.EqualFold(arg.Account, asset.Margin.String()) {
		return nil, errors.New("only 'spot', 'cross_margin', and 'margin' area allowed")
	}
	if arg.Amount <= 0 {
		return nil, errInvalidAmount
	}
	if arg.Price < 0 {
		return nil, errInvalidPrice
	}
	var response *SpotOrder
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotPlaceOrderEPL, http.MethodPost, gateioSpotOrders, nil, &arg, &response)
}

// GetSpotOrders retrieves spot orders.
func (g *Gateio) GetSpotOrders(ctx context.Context, currencyPair currency.Pair, status string, page, limit uint64) ([]SpotOrder, error) {
	if currencyPair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("currency_pair", currencyPair.String())
	if status != "" {
		params.Set("status", status)
	}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var response []SpotOrder
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotGetOrdersEPL, http.MethodGet, gateioSpotOrders, params, nil, &response)
}

// CancelAllOpenOrdersSpecifiedCurrencyPair cancel all open orders in specified currency pair
func (g *Gateio) CancelAllOpenOrdersSpecifiedCurrencyPair(ctx context.Context, currencyPair currency.Pair, side order.Side, account asset.Item) ([]SpotOrder, error) {
	if currencyPair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("currency_pair", currencyPair.String())
	if side == order.Buy || side == order.Sell {
		params.Set("side", strings.ToLower(side.Title()))
	}
	if account == asset.Spot || account == asset.Margin || account == asset.CrossMargin {
		params.Set("account", account.String())
	}
	var response []SpotOrder
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotCancelAllOpenOrdersEPL, http.MethodDelete, gateioSpotOrders, params, nil, &response)
}

// CancelBatchOrdersWithIDList cancels batch orders specifying the order ID and currency pair information
// Multiple currency pairs can be specified, but maximum 20 orders are allowed per request
func (g *Gateio) CancelBatchOrdersWithIDList(ctx context.Context, args []CancelOrderByIDParam) ([]CancelOrderByIDResponse, error) {
	var response []CancelOrderByIDResponse
	if len(args) == 0 {
		return nil, errNoValidParameterPassed
	} else if len(args) > 20 {
		return nil, fmt.Errorf("%w maximum order size to cancel is 20", errInvalidOrderSize)
	}
	for x := range args {
		if args[x].CurrencyPair.IsEmpty() || args[x].ID == "" {
			return nil, errors.New("currency pair and order ID are required")
		}
	}
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotCancelBatchOrdersEPL, http.MethodPost, gateioSpotCancelBatchOrders, nil, &args, &response)
}

// GetSpotOrder retrieves a single spot order using the order id and currency pair information.
func (g *Gateio) GetSpotOrder(ctx context.Context, orderID string, currencyPair currency.Pair, account asset.Item) (*SpotOrder, error) {
	if orderID == "" {
		return nil, errInvalidOrderID
	}
	if currencyPair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("currency_pair", currencyPair.String())
	if accountType := account.String(); accountType != "" {
		params.Set("account", accountType)
	}
	var response *SpotOrder
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotGetOrderEPL, http.MethodGet, gateioSpotOrders+"/"+orderID, params, nil, &response)
}

// AmendSpotOrder amend an order
// By default, the orders of spot and margin account are updated.
// If you need to modify orders of the cross-margin account, you must specify account as cross_margin.
// For portfolio margin account, only cross_margin account is supported.
func (g *Gateio) AmendSpotOrder(ctx context.Context, orderID string, currencyPair currency.Pair, isCrossMarginAccount bool, arg *PriceAndAmount) (*SpotOrder, error) {
	if arg == nil {
		return nil, errNilArgument
	}
	if orderID == "" {
		return nil, errInvalidOrderID
	}
	if currencyPair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("currency_pair", currencyPair.String())
	if isCrossMarginAccount {
		params.Set("account", asset.CrossMargin.String())
	}
	if arg.Amount != 0 && arg.Price != 0 {
		return nil, errors.New("only can chose one of amount or price")
	}
	var resp *SpotOrder
	return resp, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotAmendOrderEPL, http.MethodPatch, gateioSpotOrders+"/"+orderID, params, arg, &resp)
}

// CancelSingleSpotOrder cancels a single order
// Spot and margin orders are cancelled by default.
// If trying to cancel cross margin orders or portfolio margin account are used, account must be set to cross_margin
func (g *Gateio) CancelSingleSpotOrder(ctx context.Context, orderID, currencyPair string, isCrossMarginAccount bool) (*SpotOrder, error) {
	if orderID == "" {
		return nil, errInvalidOrderID
	}
	if currencyPair == "" {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("currency_pair", currencyPair)
	if isCrossMarginAccount {
		params.Set("account", asset.CrossMargin.String())
	}
	var response *SpotOrder
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotCancelSingleOrderEPL, http.MethodDelete, gateioSpotOrders+"/"+orderID, params, nil, &response)
}

// GetMySpotTradingHistory retrieves personal trading history
func (g *Gateio) GetMySpotTradingHistory(ctx context.Context, p currency.Pair, orderID string, page, limit uint64, crossMargin bool, from, to time.Time) ([]SpotPersonalTradeHistory, error) {
	params := url.Values{}
	if p.IsPopulated() {
		params.Set("currency_pair", p.String())
	}
	if orderID != "" {
		params.Set("order_id", orderID)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
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
	var response []SpotPersonalTradeHistory
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotTradingHistoryEPL, http.MethodGet, gateioSpotMyTrades, params, nil, &response)
}

// GetServerTime retrieves current server time
func (g *Gateio) GetServerTime(ctx context.Context, _ asset.Item) (time.Time, error) {
	var resp struct {
		ServerTime types.Time `json:"server_time"`
	}
	if err := g.SendHTTPRequest(ctx, exchange.RestSpot, publicGetServerTimeEPL, gateioSpotServerTime, &resp); err != nil {
		return time.Time{}, err
	}
	return resp.ServerTime.Time(), nil
}

// CountdownCancelorders Countdown cancel orders
// When the timeout set by the user is reached, if there is no cancel or set a new countdown, the related pending orders will be automatically cancelled.
// This endpoint can be called repeatedly to set a new countdown or cancel the countdown.
func (g *Gateio) CountdownCancelorders(ctx context.Context, arg CountdownCancelOrderParam) (*TriggerTimeResponse, error) {
	if arg.Timeout <= 0 {
		return nil, errInvalidCountdown
	}
	var response *TriggerTimeResponse
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotCountdownCancelEPL, http.MethodPost, gateioSpotAllCountdown, nil, &arg, &response)
}

// CreatePriceTriggeredOrder create a price-triggered order
func (g *Gateio) CreatePriceTriggeredOrder(ctx context.Context, arg *PriceTriggeredOrderParam) (*OrderID, error) {
	if arg == nil {
		return nil, errNilArgument
	}
	if arg.Put.TimeInForce != gtcTIF && arg.Put.TimeInForce != iocTIF {
		return nil, fmt.Errorf("%w: %q only 'gct' and 'ioc' are supported", order.ErrUnsupportedTimeInForce, arg.Put.TimeInForce)
	}
	if arg.Market.IsEmpty() {
		return nil, fmt.Errorf("%w, %s", currency.ErrCurrencyPairEmpty, "field market is required")
	}
	if arg.Trigger.Price < 0 {
		return nil, fmt.Errorf("%w trigger price found %f, but expected trigger_price >=0", errInvalidPrice, arg.Trigger.Price)
	}
	if arg.Trigger.Rule != "<=" && arg.Trigger.Rule != ">=" {
		return nil, fmt.Errorf("invalid price trigger condition or rule %q but expected '>=' or '<='", arg.Trigger.Rule)
	}
	if arg.Trigger.Expiration <= 0 {
		return nil, errors.New("invalid expiration(seconds to wait for the condition to be triggered before cancelling the order)")
	}
	arg.Put.Side = strings.ToLower(arg.Put.Side)
	arg.Put.Type = strings.ToLower(arg.Put.Type)
	if arg.Put.Type != "limit" {
		return nil, errors.New("invalid order type, only order type 'limit' is allowed")
	}
	if arg.Put.Side != "buy" && arg.Put.Side != "sell" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.Put.Price < 0 {
		return nil, fmt.Errorf("%w, %s", errInvalidPrice, "put price has to be greater than 0")
	}
	if arg.Put.Amount <= 0 {
		return nil, errInvalidAmount
	}
	arg.Put.Account = strings.ToLower(arg.Put.Account)
	if arg.Put.Account == "" {
		arg.Put.Account = "normal"
	}
	var response *OrderID
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotCreateTriggerOrderEPL, http.MethodPost, gateioSpotPriceOrders, nil, &arg, &response)
}

// GetPriceTriggeredOrderList retrieves price orders created with an order detail and trigger price information.
func (g *Gateio) GetPriceTriggeredOrderList(ctx context.Context, status string, market currency.Pair, account asset.Item, offset, limit uint64) ([]SpotPriceTriggeredOrder, error) {
	if status != statusOpen && status != statusFinished {
		return nil, fmt.Errorf("%w status %s", errInvalidOrderStatus, status)
	}
	params := url.Values{}
	params.Set("status", status)
	if market.IsPopulated() {
		params.Set("market", market.String())
	}
	if account == asset.CrossMargin {
		params.Set("account", account.String())
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	var response []SpotPriceTriggeredOrder
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotGetTriggerOrderListEPL, http.MethodGet, gateioSpotPriceOrders, params, nil, &response)
}

// CancelMultipleSpotOpenOrders deletes price triggered orders.
func (g *Gateio) CancelMultipleSpotOpenOrders(ctx context.Context, currencyPair currency.Pair, account asset.Item) ([]SpotPriceTriggeredOrder, error) {
	params := url.Values{}
	if currencyPair.IsPopulated() {
		params.Set("market", currencyPair.String())
	}
	switch account {
	case asset.Empty:
		return nil, asset.ErrNotSupported
	case asset.Spot:
		params.Set("account", "normal")
	default:
		params.Set("account", account.String())
	}
	var response []SpotPriceTriggeredOrder
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotCancelTriggerOrdersEPL, http.MethodDelete, gateioSpotPriceOrders, params, nil, &response)
}

// GetSinglePriceTriggeredOrder get a single order
func (g *Gateio) GetSinglePriceTriggeredOrder(ctx context.Context, orderID string) (*SpotPriceTriggeredOrder, error) {
	if orderID == "" {
		return nil, errInvalidOrderID
	}
	var response *SpotPriceTriggeredOrder
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotGetTriggerOrderEPL, http.MethodGet, gateioSpotPriceOrders+"/"+orderID, nil, nil, &response)
}

// CancelPriceTriggeredOrder cancel a price-triggered order
func (g *Gateio) CancelPriceTriggeredOrder(ctx context.Context, orderID string) (*SpotPriceTriggeredOrder, error) {
	if orderID == "" {
		return nil, errInvalidOrderID
	}
	var response *SpotPriceTriggeredOrder
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotCancelTriggerOrderEPL, http.MethodGet, gateioSpotPriceOrders+"/"+orderID, nil, nil, &response)
}

// GenerateSignature returns hash for authenticated requests
func (g *Gateio) GenerateSignature(secret, method, path, query string, body any, dtime time.Time) (string, error) {
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

// SendAuthenticatedHTTPRequest sends authenticated requests to the Gateio API
// To use this you must setup an APIKey and APISecret from the exchange
func (g *Gateio) SendAuthenticatedHTTPRequest(ctx context.Context, ep exchange.URL, epl request.EndpointLimit, method, endpoint string, param url.Values, data, result any) error {
	creds, err := g.GetCredentials(ctx)
	if err != nil {
		return err
	}
	ePoint, err := g.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	var intermediary json.RawMessage
	err = g.SendPayload(ctx, epl, func() (*request.Item, error) {
		headers := make(map[string]string)
		urlPath := endpoint
		timestamp := time.Now()
		var paramValue string
		if param != nil {
			paramValue = param.Encode()
		}
		var sig string
		sig, err = g.GenerateSignature(creds.Secret, method, "/"+gateioAPIVersion+endpoint, paramValue, data, timestamp)
		if err != nil {
			return nil, err
		}
		headers["Content-Type"] = "application/json"
		headers["KEY"] = creds.Key
		headers["TIMESTAMP"] = strconv.FormatInt(timestamp.Unix(), 10)
		headers["Accept"] = "application/json"
		headers["SIGN"] = sig
		urlPath = ePoint + urlPath
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
			Method:        method,
			Path:          urlPath,
			Headers:       headers,
			Body:          strings.NewReader(payload),
			Result:        &intermediary,
			Verbose:       g.Verbose,
			HTTPDebugging: g.HTTPDebugging,
			HTTPRecording: g.HTTPRecording,
		}, nil
	}, request.AuthenticatedRequest)
	if err != nil {
		return err
	}
	errCap := struct {
		Label   string `json:"label"`
		Code    string `json:"code"`
		Message string `json:"message"`
	}{}

	if err := json.Unmarshal(intermediary, &errCap); err == nil && errCap.Code != "" {
		return fmt.Errorf("%s auth request error, code: %s message: %s",
			g.Name,
			errCap.Label,
			errCap.Message)
	}
	if result == nil {
		return nil
	}
	return json.Unmarshal(intermediary, result)
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (g *Gateio) SendHTTPRequest(ctx context.Context, ep exchange.URL, epl request.EndpointLimit, path string, result any) error {
	endpoint, err := g.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	item := &request.Item{
		Method:        http.MethodGet,
		Path:          endpoint + path,
		Result:        result,
		Verbose:       g.Verbose,
		HTTPDebugging: g.HTTPDebugging,
		HTTPRecording: g.HTTPRecording,
	}
	return g.SendPayload(ctx, epl, func() (*request.Item, error) {
		return item, nil
	}, request.UnauthenticatedRequest)
}

// *********************************** Withdrawals ******************************

// WithdrawCurrency to withdraw a currency.
func (g *Gateio) WithdrawCurrency(ctx context.Context, arg WithdrawalRequestParam) (*WithdrawalResponse, error) {
	if arg.Amount <= 0 {
		return nil, fmt.Errorf("%w currency amount must be greater than zero", errInvalidAmount)
	}
	if arg.Currency.IsEmpty() {
		return nil, fmt.Errorf("%w currency to be withdrawal nust be specified", currency.ErrCurrencyCodeEmpty)
	}
	if arg.Chain == "" {
		return nil, errors.New("name of the chain used for withdrawal must be specified")
	}
	var response *WithdrawalResponse
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletWithdrawEPL, http.MethodPost, withdrawal, nil, &arg, &response)
}

// CancelWithdrawalWithSpecifiedID cancels withdrawal with specified ID.
func (g *Gateio) CancelWithdrawalWithSpecifiedID(ctx context.Context, withdrawalID string) (*WithdrawalResponse, error) {
	if withdrawalID == "" {
		return nil, errMissingWithdrawalID
	}
	var response *WithdrawalResponse
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletCancelWithdrawEPL, http.MethodDelete, withdrawal+"/"+withdrawalID, nil, nil, &response)
}

// *********************************** Wallet ***********************************

// ListCurrencyChain retrieves a list of currency chain name
func (g *Gateio) ListCurrencyChain(ctx context.Context, ccy currency.Code) ([]CurrencyChain, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	var resp []CurrencyChain
	return resp, g.SendHTTPRequest(ctx, exchange.RestSpot, publicListCurrencyChainEPL, common.EncodeURLValues(walletCurrencyChain, params), &resp)
}

// GenerateCurrencyDepositAddress generate currency deposit address
func (g *Gateio) GenerateCurrencyDepositAddress(ctx context.Context, ccy currency.Code) (*CurrencyDepositAddressInfo, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	var response *CurrencyDepositAddressInfo
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletDepositAddressEPL, http.MethodGet, walletDepositAddress, params, nil, &response)
}

// GetWithdrawalRecords retrieves withdrawal records. Record time range cannot exceed 30 days
func (g *Gateio) GetWithdrawalRecords(ctx context.Context, ccy currency.Code, from, to time.Time, offset, limit uint64) ([]WithdrawalResponse, error) {
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
		if err := common.StartEndTimeCheck(from, to); err != nil && !to.IsZero() {
			return nil, err
		} else if !to.IsZero() {
			params.Set("to", strconv.FormatInt(to.Unix(), 10))
		}
	}
	var withdrawals []WithdrawalResponse
	return withdrawals, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletWithdrawalRecordsEPL, http.MethodGet, walletWithdrawals, params, nil, &withdrawals)
}

// GetDepositRecords retrieves deposit records. Record time range cannot exceed 30 days
func (g *Gateio) GetDepositRecords(ctx context.Context, ccy currency.Code, from, to time.Time, offset, limit uint64) ([]DepositRecord, error) {
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
		if err := common.StartEndTimeCheck(from, to); err != nil {
			params.Set("to", strconv.FormatInt(to.Unix(), 10))
		}
	}
	var depositHistories []DepositRecord
	return depositHistories, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletDepositRecordsEPL, http.MethodGet, walletDeposits, params, nil, &depositHistories)
}

// TransferCurrency Transfer between different accounts. Currently support transfers between the following:
// spot - margin, spot - futures(perpetual), spot - delivery
// spot - cross margin, spot - options
func (g *Gateio) TransferCurrency(ctx context.Context, arg *TransferCurrencyParam) (*TransactionIDResponse, error) {
	if arg == nil {
		return nil, errNilArgument
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.From == "" {
		return nil, errors.New("from account is required")
	}
	if arg.To == "" {
		return nil, errors.New("to account is required")
	}
	if arg.To == arg.From {
		return nil, errors.New("from and to account cannot be the same")
	}
	if (arg.To == "margin" || arg.From == "margin") && arg.CurrencyPair.IsEmpty() {
		return nil, errors.New("currency pair is required for margin account transfer")
	}
	if (arg.To == "futures" || arg.From == "futures") && arg.Settle == "" {
		return nil, errors.New("settle is required for futures account transfer")
	}
	if arg.Amount <= 0 {
		return nil, errInvalidAmount
	}
	var response *TransactionIDResponse
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletTransferCurrencyEPL, http.MethodPost, walletTransfer, nil, &arg, &response)
}

func (g *Gateio) assetTypeToString(acc asset.Item) string {
	if acc == asset.Options {
		return "options"
	}
	return acc.String()
}

// SubAccountTransfer to transfer between main and sub accounts
// Support transferring with sub user's spot or futures account. Note that only main user's spot account is used no matter which sub user's account is operated.
func (g *Gateio) SubAccountTransfer(ctx context.Context, arg SubAccountTransferParam) error {
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
		return errInvalidAmount
	}
	switch arg.SubAccountType {
	case "", "spot", "futures", "delivery":
	default:
		return fmt.Errorf("%w %q for SubAccountTransfer; Supported: [spot, futures, delivery]", asset.ErrNotSupported, arg.SubAccountType)
	}
	return g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletSubAccountTransferEPL, http.MethodPost, walletSubAccountTransfer, nil, &arg, nil)
}

// GetSubAccountTransferHistory retrieve transfer records between main and sub accounts.
// retrieve transfer records between main and sub accounts. Record time range cannot exceed 30 days
// Note: only records after 2020-04-10 can be retrieved
func (g *Gateio) GetSubAccountTransferHistory(ctx context.Context, subAccountUserID string, from, to time.Time, offset, limit uint64) ([]SubAccountTransferResponse, error) {
	params := url.Values{}
	if subAccountUserID != "" {
		params.Set("sub_uid", subAccountUserID)
	}
	startingTime, err := time.Parse("2006-Jan-02", "2020-Apr-10")
	if err != nil {
		return nil, err
	}
	if err := common.StartEndTimeCheck(startingTime, from); err == nil {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if err := common.StartEndTimeCheck(from, to); err == nil {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var response []SubAccountTransferResponse
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletSubAccountTransferHistoryEPL, http.MethodGet, walletSubAccountTransfer, params, nil, &response)
}

// SubAccountTransferToSubAccount performs sub-account transfers to sub-account
func (g *Gateio) SubAccountTransferToSubAccount(ctx context.Context, arg *InterSubAccountTransferParams) error {
	if arg.Currency.IsEmpty() {
		return currency.ErrCurrencyCodeEmpty
	}
	if arg.SubAccountFromUserID == "" {
		return errors.New("sub-account from user-id is required")
	}
	if arg.SubAccountFromAssetType == asset.Empty {
		return errors.New("sub-account to transfer the asset from is required")
	}
	if arg.SubAccountToUserID == "" {
		return errors.New("sub-account to user-id is required")
	}
	if arg.SubAccountToAssetType == asset.Empty {
		return errors.New("sub-account to transfer to is required")
	}
	if arg.Amount <= 0 {
		return errInvalidAmount
	}
	return g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletSubAccountToSubAccountTransferEPL, http.MethodPost, walletInterSubAccountTransfer, nil, &arg, nil)
}

// GetWithdrawalStatus retrieves withdrawal status
func (g *Gateio) GetWithdrawalStatus(ctx context.Context, ccy currency.Code) ([]WithdrawalStatus, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	var response []WithdrawalStatus
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletWithdrawStatusEPL, http.MethodGet, walletWithdrawStatus, params, nil, &response)
}

// GetSubAccountBalances retrieve sub account balances
func (g *Gateio) GetSubAccountBalances(ctx context.Context, subAccountUserID string) ([]FuturesSubAccountBalance, error) {
	params := url.Values{}
	if subAccountUserID != "" {
		params.Set("sub_uid", subAccountUserID)
	}
	var response []FuturesSubAccountBalance
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletSubAccountBalancesEPL, http.MethodGet, walletSubAccountBalance, params, nil, &response)
}

// GetSubAccountMarginBalances query sub accounts' margin balances
func (g *Gateio) GetSubAccountMarginBalances(ctx context.Context, subAccountUserID string) ([]SubAccountMarginBalance, error) {
	params := url.Values{}
	if subAccountUserID != "" {
		params.Set("sub_uid", subAccountUserID)
	}
	var response []SubAccountMarginBalance
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletSubAccountMarginBalancesEPL, http.MethodGet, walletSubAccountMarginBalance, params, nil, &response)
}

// GetSubAccountFuturesBalances retrieves sub accounts' futures account balances
func (g *Gateio) GetSubAccountFuturesBalances(ctx context.Context, subAccountUserID string, settle currency.Code) ([]FuturesSubAccountBalance, error) {
	params := url.Values{}
	if subAccountUserID != "" {
		params.Set("sub_uid", subAccountUserID)
	}
	if !settle.IsEmpty() {
		params.Set("settle", settle.Item.Lower)
	}
	var response []FuturesSubAccountBalance
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletSubAccountFuturesBalancesEPL, http.MethodGet, walletSubAccountFuturesBalance, params, nil, &response)
}

// GetSubAccountCrossMarginBalances query subaccount's cross_margin account info
func (g *Gateio) GetSubAccountCrossMarginBalances(ctx context.Context, subAccountUserID string) ([]SubAccountCrossMarginInfo, error) {
	params := url.Values{}
	if subAccountUserID != "" {
		params.Set("sub_uid", subAccountUserID)
	}
	var response []SubAccountCrossMarginInfo
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletSubAccountCrossMarginBalancesEPL, http.MethodGet, walletSubAccountCrossMarginBalances, params, nil, &response)
}

// GetSavedAddresses retrieves saved currency address info and related details.
func (g *Gateio) GetSavedAddresses(ctx context.Context, ccy currency.Code, chain string, limit uint64) ([]WalletSavedAddress, error) {
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
	var response []WalletSavedAddress
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletSavedAddressesEPL, http.MethodGet, walletSavedAddress, params, nil, &response)
}

// GetPersonalTradingFee retrieves personal trading fee
func (g *Gateio) GetPersonalTradingFee(ctx context.Context, currencyPair currency.Pair, settle currency.Code) (*PersonalTradingFee, error) {
	params := url.Values{}
	if currencyPair.IsPopulated() {
		// specify a currency pair to retrieve precise fee rate
		params.Set("currency_pair", currencyPair.String())
	}
	if !settle.IsEmpty() {
		params.Set("settle", settle.Item.Lower)
	}
	var response *PersonalTradingFee
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletTradingFeeEPL, http.MethodGet, walletTradingFee, params, nil, &response)
}

// GetUsersTotalBalance retrieves user's total balances
func (g *Gateio) GetUsersTotalBalance(ctx context.Context, ccy currency.Code) (*UsersAllAccountBalance, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	var response *UsersAllAccountBalance
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletTotalBalanceEPL, http.MethodGet, walletTotalBalance, params, nil, &response)
}

// ConvertSmallBalances converts small balances of provided currencies into GT.
// If no currencies are provided, all supported currencies will be converted
// See [this documentation](https://www.gate.io/help/guide/functional_guidelines/22367) for details and restrictions.
func (g *Gateio) ConvertSmallBalances(ctx context.Context, currs ...currency.Code) error {
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
	return g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletConvertSmallBalancesEPL, http.MethodPost, "wallet/small_balance", nil, payload, nil)
}

// ********************************* Margin *******************************************

// GetMarginSupportedCurrencyPairs retrieves margin supported currency pairs.
func (g *Gateio) GetMarginSupportedCurrencyPairs(ctx context.Context) ([]MarginCurrencyPairInfo, error) {
	var currenciePairsInfo []MarginCurrencyPairInfo
	return currenciePairsInfo, g.SendHTTPRequest(ctx, exchange.RestSpot, publicCurrencyPairsMarginEPL, gateioMarginCurrencyPairs, &currenciePairsInfo)
}

// GetSingleMarginSupportedCurrencyPair retrieves margin supported currency pair detail given the currency pair.
func (g *Gateio) GetSingleMarginSupportedCurrencyPair(ctx context.Context, market currency.Pair) (*MarginCurrencyPairInfo, error) {
	if market.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var currencyPairInfo *MarginCurrencyPairInfo
	return currencyPairInfo, g.SendHTTPRequest(ctx, exchange.RestSpot, publicCurrencyPairsMarginEPL, gateioMarginCurrencyPairs+"/"+market.String(), &currencyPairInfo)
}

// GetOrderbookOfLendingLoans retrieves order book of lending loans for specific currency
func (g *Gateio) GetOrderbookOfLendingLoans(ctx context.Context, ccy currency.Code) ([]OrderbookOfLendingLoan, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var lendingLoans []OrderbookOfLendingLoan
	return lendingLoans, g.SendHTTPRequest(ctx, exchange.RestSpot, publicOrderbookMarginEPL, gateioMarginFundingBook+"?currency="+ccy.String(), &lendingLoans)
}

// GetMarginAccountList margin account list
func (g *Gateio) GetMarginAccountList(ctx context.Context, currencyPair currency.Pair) ([]MarginAccountItem, error) {
	params := url.Values{}
	if currencyPair.IsPopulated() {
		params.Set("currency_pair", currencyPair.String())
	}
	var response []MarginAccountItem
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginAccountListEPL, http.MethodGet, gateioMarginAccount, params, nil, &response)
}

// ListMarginAccountBalanceChangeHistory retrieves margin account balance change history
// Only transferals from and to margin account are provided for now. Time range allows 30 days at most
func (g *Gateio) ListMarginAccountBalanceChangeHistory(ctx context.Context, ccy currency.Code, currencyPair currency.Pair, from, to time.Time, page, limit uint64) ([]MarginAccountBalanceChangeInfo, error) {
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
	var response []MarginAccountBalanceChangeInfo
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginAccountBalanceEPL, http.MethodGet, gateioMarginAccountBook, params, nil, &response)
}

// GetMarginFundingAccountList retrieves funding account list
func (g *Gateio) GetMarginFundingAccountList(ctx context.Context, ccy currency.Code) ([]MarginFundingAccountItem, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	var response []MarginFundingAccountItem
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginFundingAccountListEPL, http.MethodGet, gateioMarginFundingAccounts, params, nil, &response)
}

// MarginLoan represents lend or borrow request
func (g *Gateio) MarginLoan(ctx context.Context, arg *MarginLoanRequestParam) (*MarginLoanResponse, error) {
	if arg == nil {
		return nil, errNilArgument
	}
	if arg.Side != sideLend && arg.Side != sideBorrow {
		return nil, errInvalidLoanSide
	}
	if arg.Side == sideBorrow && arg.Rate == 0 {
		return nil, errors.New("`rate` is required in borrowing")
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Amount <= 0 {
		return nil, errInvalidAmount
	}
	if arg.Rate != 0 && arg.Rate > 0.002 || arg.Rate < 0.0002 {
		return nil, errors.New("invalid loan rate, rate must be between 0.0002 and 0.002")
	}
	var response *MarginLoanResponse
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginLendBorrowEPL, http.MethodPost, gateioMarginLoans, nil, &arg, &response)
}

// GetMarginAllLoans retrieves all loans (borrow and lending) orders.
func (g *Gateio) GetMarginAllLoans(ctx context.Context, status, side, sortBy string, ccy currency.Code, currencyPair currency.Pair, reverseSort bool, page, limit uint64) ([]MarginLoanResponse, error) {
	if side != sideLend && side != sideBorrow {
		return nil, fmt.Errorf("%w, only 'lend' and 'borrow' are supported", order.ErrSideIsInvalid)
	}
	params := url.Values{}
	params.Set("side", side)
	if status == statusOpen || status == "loaned" || status == statusFinished || status == "auto_repair" {
		params.Set("status", status)
	} else {
		return nil, errors.New("loan status \"status\" is required")
	}
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
	var response []MarginLoanResponse
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginAllLoansEPL, http.MethodGet, gateioMarginLoans, params, nil, &response)
}

// MergeMultipleLendingLoans merge multiple lending loans
func (g *Gateio) MergeMultipleLendingLoans(ctx context.Context, ccy currency.Code, ids []string) (*MarginLoanResponse, error) {
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
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginMergeLendingLoansEPL, http.MethodPost, gateioMarginMergedLoans, params, nil, &response)
}

// RetriveOneSingleLoanDetail retrieve one single loan detail
// "side" represents loan side: Lend or Borrow
func (g *Gateio) RetriveOneSingleLoanDetail(ctx context.Context, side, loanID string) (*MarginLoanResponse, error) {
	if side != sideBorrow && side != sideLend {
		return nil, errInvalidLoanSide
	}
	if loanID == "" {
		return nil, errInvalidLoanID
	}
	params := url.Values{}
	params.Set("side", side)
	var response *MarginLoanResponse
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginGetLoanEPL, http.MethodGet, gateioMarginLoans+"/"+loanID+"/", params, nil, &response)
}

// ModifyALoan Modify a loan
// only auto_renew modification is supported currently
func (g *Gateio) ModifyALoan(ctx context.Context, loanID string, arg *ModifyLoanRequestParam) (*MarginLoanResponse, error) {
	if arg == nil {
		return nil, errNilArgument
	}
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
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginModifyLoanEPL, http.MethodPatch, gateioMarginLoans+"/"+loanID, nil, &arg, &response)
}

// CancelLendingLoan cancels lending loans. only lent loans can be canceled.
func (g *Gateio) CancelLendingLoan(ctx context.Context, ccy currency.Code, loanID string) (*MarginLoanResponse, error) {
	if loanID == "" {
		return nil, fmt.Errorf("%w, %s", errInvalidLoanID, " loan_id is required")
	}
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	var response *MarginLoanResponse
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginCancelLoanEPL, http.MethodDelete, gateioMarginLoans+"/"+loanID, params, nil, &response)
}

// RepayALoan execute a loan repay.
func (g *Gateio) RepayALoan(ctx context.Context, loanID string, arg *RepayLoanRequestParam) (*MarginLoanResponse, error) {
	if arg == nil {
		return nil, errNilArgument
	}
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
		return nil, fmt.Errorf("%w, repay amount for partial repay mode must be greater than 0", errInvalidAmount)
	}
	var response *MarginLoanResponse
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginRepayLoanEPL, http.MethodPost, gateioMarginLoans+"/"+loanID+"/repayment", nil, &arg, &response)
}

// ListLoanRepaymentRecords retrieves loan repayment records for specified loan ID
func (g *Gateio) ListLoanRepaymentRecords(ctx context.Context, loanID string) ([]LoanRepaymentRecord, error) {
	if loanID == "" {
		return nil, fmt.Errorf("%w, %v", errInvalidLoanID, " loan_id is required")
	}
	var response []LoanRepaymentRecord
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginListLoansEPL, http.MethodGet, gateioMarginLoans+"/"+loanID+"/repayment", nil, nil, &response)
}

// ListRepaymentRecordsOfSpecificLoan retrieves repayment records of specific loan
func (g *Gateio) ListRepaymentRecordsOfSpecificLoan(ctx context.Context, loanID, status string, page, limit uint64) ([]LoanRecord, error) {
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
	var response []LoanRecord
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginRepaymentRecordEPL, http.MethodGet, gateioMarginLoanRecords, params, nil, &response)
}

// GetOneSingleLoanRecord get one single loan record
func (g *Gateio) GetOneSingleLoanRecord(ctx context.Context, loanID, loanRecordID string) (*LoanRecord, error) {
	if loanID == "" {
		return nil, fmt.Errorf("%w, %v", errInvalidLoanID, " loan_id is required")
	}
	if loanRecordID == "" {
		return nil, fmt.Errorf("%w, %v", errInvalidLoanID, " loan_record_id is required")
	}
	params := url.Values{}
	params.Set("loan_id", loanID)
	var response *LoanRecord
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginSingleRecordEPL, http.MethodGet, gateioMarginLoanRecords+"/"+loanRecordID, params, nil, &response)
}

// ModifyALoanRecord modify a loan record
// Only auto_renew modification is supported currently
func (g *Gateio) ModifyALoanRecord(ctx context.Context, loanRecordID string, arg *ModifyLoanRequestParam) (*LoanRecord, error) {
	if arg == nil {
		return nil, errNilArgument
	}
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
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginModifyLoanRecordEPL, http.MethodPatch, gateioMarginLoanRecords+"/"+loanRecordID, nil, &arg, &response)
}

// UpdateUsersAutoRepaymentSetting represents update user's auto repayment setting
func (g *Gateio) UpdateUsersAutoRepaymentSetting(ctx context.Context, statusOn bool) (*OnOffStatus, error) {
	var statusStr string
	if statusOn {
		statusStr = "on"
	} else {
		statusStr = "off"
	}
	params := url.Values{}
	params.Set("status", statusStr)
	var response *OnOffStatus
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginAutoRepayEPL, http.MethodPost, gateioMarginAutoRepay, params, nil, &response)
}

// GetUserAutoRepaymentSetting retrieve user auto repayment setting
func (g *Gateio) GetUserAutoRepaymentSetting(ctx context.Context) (*OnOffStatus, error) {
	var response *OnOffStatus
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginGetAutoRepaySettingsEPL, http.MethodGet, gateioMarginAutoRepay, nil, nil, &response)
}

// GetMaxTransferableAmountForSpecificMarginCurrency get the max transferable amount for a specific margin currency.
func (g *Gateio) GetMaxTransferableAmountForSpecificMarginCurrency(ctx context.Context, ccy currency.Code, currencyPair currency.Pair) (*MaxTransferAndLoanAmount, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	if currencyPair.IsPopulated() {
		params.Set("currency_pair", currencyPair.String())
	}
	params.Set("currency", ccy.String())
	var response *MaxTransferAndLoanAmount
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginGetMaxTransferEPL, http.MethodGet, gateioMarginTransfer, params, nil, &response)
}

// GetMaxBorrowableAmountForSpecificMarginCurrency retrieves the max borrowble amount for specific currency
func (g *Gateio) GetMaxBorrowableAmountForSpecificMarginCurrency(ctx context.Context, ccy currency.Code, currencyPair currency.Pair) (*MaxTransferAndLoanAmount, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	if currencyPair.IsPopulated() {
		params.Set("currency_pair", currencyPair.String())
	}
	params.Set("currency", ccy.String())
	var response *MaxTransferAndLoanAmount
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginGetMaxBorrowEPL, http.MethodGet, gateioMarginBorrowable, params, nil, &response)
}

// CurrencySupportedByCrossMargin currencies supported by cross margin.
func (g *Gateio) CurrencySupportedByCrossMargin(ctx context.Context) ([]CrossMarginCurrencies, error) {
	var response []CrossMarginCurrencies
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginSupportedCurrencyCrossListEPL, http.MethodGet, gateioCrossMarginCurrencies, nil, nil, &response)
}

// GetCrossMarginSupportedCurrencyDetail retrieve detail of one single currency supported by cross margin
func (g *Gateio) GetCrossMarginSupportedCurrencyDetail(ctx context.Context, ccy currency.Code) (*CrossMarginCurrencies, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var response *CrossMarginCurrencies
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginSupportedCurrencyCrossEPL, http.MethodGet, gateioCrossMarginCurrencies+"/"+ccy.String(), nil, nil, &response)
}

// GetCrossMarginAccounts retrieve cross margin account
func (g *Gateio) GetCrossMarginAccounts(ctx context.Context) (*CrossMarginAccount, error) {
	var response *CrossMarginAccount
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginAccountsEPL, http.MethodGet, gateioCrossMarginAccounts, nil, nil, &response)
}

// GetCrossMarginAccountChangeHistory retrieve cross margin account change history
// Record time range cannot exceed 30 days
func (g *Gateio) GetCrossMarginAccountChangeHistory(ctx context.Context, ccy currency.Code, from, to time.Time, page, limit uint64, accountChangeType string) ([]CrossMarginAccountHistoryItem, error) {
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
	if accountChangeType != "" { // "in", "out", "repay", "new_order", "order_fill", "referral_fee", "order_fee", "unknown" are supported
		params.Set("type", accountChangeType)
	}
	var response []CrossMarginAccountHistoryItem
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginAccountHistoryEPL, http.MethodGet, gateioCrossMarginAccountBook, params, nil, &response)
}

// CreateCrossMarginBorrowLoan create a cross margin borrow loan
// Borrow amount cannot be less than currency minimum borrow amount
func (g *Gateio) CreateCrossMarginBorrowLoan(ctx context.Context, arg CrossMarginBorrowLoanParams) (*CrossMarginLoanResponse, error) {
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Amount <= 0 {
		return nil, fmt.Errorf("%w, borrow amount must be greater than 0", errInvalidAmount)
	}
	var response CrossMarginLoanResponse
	return &response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginCreateCrossBorrowLoanEPL, http.MethodPost, gateioCrossMarginLoans, nil, &arg, &response)
}

// ExecuteRepayment when the liquidity of the currency is insufficient and the transaction risk is high, the currency will be disabled,
// and funds cannot be transferred.When the available balance of cross-margin is insufficient, the balance of the spot account can be used for repayment.
// Please ensure that the balance of the spot account is sufficient, and system uses cross-margin account for repayment first
func (g *Gateio) ExecuteRepayment(ctx context.Context, arg CurrencyAndAmount) ([]CrossMarginLoanResponse, error) {
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Amount <= 0 {
		return nil, fmt.Errorf("%w, repay amount must be greater than 0", errInvalidAmount)
	}
	var response []CrossMarginLoanResponse
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginExecuteRepaymentsEPL, http.MethodPost, gateioCrossMarginRepayments, nil, &arg, &response)
}

// GetCrossMarginRepayments retrieves list of cross margin repayments
func (g *Gateio) GetCrossMarginRepayments(ctx context.Context, ccy currency.Code, loanID string, limit, offset uint64, reverse bool) ([]CrossMarginLoanResponse, error) {
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
	var response []CrossMarginLoanResponse
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginGetCrossMarginRepaymentsEPL, http.MethodGet, gateioCrossMarginRepayments, params, nil, &response)
}

// GetMaxTransferableAmountForSpecificCrossMarginCurrency get the max transferable amount for a specific cross margin currency
func (g *Gateio) GetMaxTransferableAmountForSpecificCrossMarginCurrency(ctx context.Context, ccy currency.Code) (*CurrencyAndAmount, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	var response *CurrencyAndAmount
	params.Set("currency", ccy.String())
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginGetMaxTransferCrossEPL, http.MethodGet, gateioCrossMarginTransferable, params, nil, &response)
}

// GetMaxBorrowableAmountForSpecificCrossMarginCurrency returns the max borrowable amount for a specific cross margin currency
func (g *Gateio) GetMaxBorrowableAmountForSpecificCrossMarginCurrency(ctx context.Context, ccy currency.Code) (*CurrencyAndAmount, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	var response *CurrencyAndAmount
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginGetMaxBorrowCrossEPL, http.MethodGet, gateioCrossMarginBorrowable, params, nil, &response)
}

// GetCrossMarginBorrowHistory retrieves cross margin borrow history sorted by creation time in descending order by default.
// Set reverse=false to return ascending results.
func (g *Gateio) GetCrossMarginBorrowHistory(ctx context.Context, status uint64, ccy currency.Code, limit, offset uint64, reverse bool) ([]CrossMarginLoanResponse, error) {
	if status < 1 || status > 3 {
		return nil, fmt.Errorf("%s %v, only allowed status values are 1:failed, 2:borrowed, and 3:repayment", g.Name, errInvalidOrderStatus)
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
	var response []CrossMarginLoanResponse
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginGetCrossBorrowHistoryEPL, http.MethodGet, gateioCrossMarginLoans, params, nil, &response)
}

// GetSingleBorrowLoanDetail retrieve single borrow loan detail
func (g *Gateio) GetSingleBorrowLoanDetail(ctx context.Context, loanID string) (*CrossMarginLoanResponse, error) {
	if loanID == "" {
		return nil, errInvalidLoanID
	}
	var response *CrossMarginLoanResponse
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginGetBorrowEPL, http.MethodGet, gateioCrossMarginLoans+"/"+loanID, nil, nil, &response)
}

// *********************************Futures***************************************

// GetAllFutureContracts retrieves list all futures contracts
func (g *Gateio) GetAllFutureContracts(ctx context.Context, settle currency.Code) ([]FuturesContract, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}
	var contracts []FuturesContract
	return contracts, g.SendHTTPRequest(ctx, exchange.RestSpot, publicFuturesContractsEPL, futuresPath+settle.Item.Lower+"/contracts", &contracts)
}

// GetFuturesContract returns a single futures contract info for the specified settle and Currency Pair (contract << in this case)
func (g *Gateio) GetFuturesContract(ctx context.Context, settle currency.Code, contract string) (*FuturesContract, error) {
	if contract == "" {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}
	var futureContract *FuturesContract
	return futureContract, g.SendHTTPRequest(ctx, exchange.RestSpot, publicFuturesContractsEPL, futuresPath+settle.Item.Lower+"/contracts/"+contract, &futureContract)
}

// GetFuturesOrderbook retrieves futures order book data
func (g *Gateio) GetFuturesOrderbook(ctx context.Context, settle currency.Code, contract, interval string, limit uint64, withOrderbookID bool) (*Orderbook, error) {
	if contract == "" {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}
	params := url.Values{}
	params.Set("contract", contract)
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
	return response, g.SendHTTPRequest(ctx, exchange.RestSpot, publicOrderbookFuturesEPL, common.EncodeURLValues(futuresPath+settle.Item.Lower+"/order_book", params), &response)
}

// GetFuturesTradingHistory retrieves futures trading history
func (g *Gateio) GetFuturesTradingHistory(ctx context.Context, settle currency.Code, contract currency.Pair, limit, offset uint64, lastID string, from, to time.Time) ([]TradingHistoryItem, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
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
	var response []TradingHistoryItem
	return response, g.SendHTTPRequest(ctx, exchange.RestSpot, publicTradingHistoryFuturesEPL, common.EncodeURLValues(futuresPath+settle.Item.Lower+"/trades", params), &response)
}

// GetFuturesCandlesticks retrieves specified contract candlesticks.
func (g *Gateio) GetFuturesCandlesticks(ctx context.Context, settle currency.Code, contract string, from, to time.Time, limit uint64, interval kline.Interval) ([]FuturesCandlestick, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}
	if contract == "" {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("contract", strings.ToUpper(contract))
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
	var candlesticks []FuturesCandlestick
	return candlesticks, g.SendHTTPRequest(ctx, exchange.RestFutures, publicCandleSticksFuturesEPL, common.EncodeURLValues(futuresPath+settle.Item.Lower+"/candlesticks", params), &candlesticks)
}

// PremiumIndexKLine retrieves premium Index K-Line
// Maximum of 1000 points can be returned in a query. Be sure not to exceed the limit when specifying from, to and interval
func (g *Gateio) PremiumIndexKLine(ctx context.Context, settleCurrency currency.Code, contract currency.Pair, from, to time.Time, limit int64, interval kline.Interval) ([]FuturesPremiumIndexKLineResponse, error) {
	if settleCurrency.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
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
	var resp []FuturesPremiumIndexKLineResponse
	return resp, g.SendHTTPRequest(ctx, exchange.RestSpot, publicPremiumIndexEPL, common.EncodeURLValues(futuresPath+settleCurrency.Item.Lower+"/premium_index", params), &resp)
}

// GetFuturesTickers retrieves futures ticker information for a specific settle and contract info.
func (g *Gateio) GetFuturesTickers(ctx context.Context, settle currency.Code, contract currency.Pair) ([]FuturesTicker, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}
	params := url.Values{}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	var tickers []FuturesTicker
	return tickers, g.SendHTTPRequest(ctx, exchange.RestSpot, publicTickersFuturesEPL, common.EncodeURLValues(futuresPath+settle.Item.Lower+"/tickers", params), &tickers)
}

// GetFutureFundingRates retrieves funding rate information.
func (g *Gateio) GetFutureFundingRates(ctx context.Context, settle currency.Code, contract currency.Pair, limit uint64) ([]FuturesFundingRate, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}
	if contract.IsInvalid() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("contract", contract.String())
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var rates []FuturesFundingRate
	return rates, g.SendHTTPRequest(ctx, exchange.RestSpot, publicFundingRatesEPL, common.EncodeURLValues(futuresPath+settle.Item.Lower+"/funding_rate", params), &rates)
}

// GetFuturesInsuranceBalanceHistory retrieves futures insurance balance history
func (g *Gateio) GetFuturesInsuranceBalanceHistory(ctx context.Context, settle currency.Code, limit uint64) ([]InsuranceBalance, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}
	params := url.Values{}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var balances []InsuranceBalance
	return balances, g.SendHTTPRequest(ctx, exchange.RestSpot, publicInsuranceFuturesEPL, common.EncodeURLValues(futuresPath+settle.Item.Lower+"/insurance", params), &balances)
}

// GetFutureStats retrieves futures stats
func (g *Gateio) GetFutureStats(ctx context.Context, settle currency.Code, contract currency.Pair, from time.Time, interval kline.Interval, limit uint64) ([]ContractStat, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
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
	var stats []ContractStat
	return stats, g.SendHTTPRequest(ctx, exchange.RestSpot, publicStatsFuturesEPL, common.EncodeURLValues(futuresPath+settle.Item.Lower+"/contract_stats", params), &stats)
}

// GetIndexConstituent retrieves index constituents
func (g *Gateio) GetIndexConstituent(ctx context.Context, settle currency.Code, index string) (*IndexConstituent, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}
	if index == "" {
		return nil, currency.ErrCurrencyPairEmpty
	}
	indexString := strings.ToUpper(index)
	var constituents *IndexConstituent
	return constituents, g.SendHTTPRequest(ctx, exchange.RestSpot, publicIndexConstituentsEPL, futuresPath+settle.Item.Lower+"/index_constituents/"+indexString, &constituents)
}

// GetLiquidationHistory retrieves liqudiation history
func (g *Gateio) GetLiquidationHistory(ctx context.Context, settle currency.Code, contract currency.Pair, from, to time.Time, limit uint64) ([]LiquidationHistory, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}
	if contract.IsInvalid() {
		return nil, errInvalidOrMissingContractParam
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
	var histories []LiquidationHistory
	return histories, g.SendHTTPRequest(ctx, exchange.RestSpot, publicLiquidationHistoryEPL, common.EncodeURLValues(futuresPath+settle.Item.Lower+"/liq_orders", params), &histories)
}

// QueryFuturesAccount retrieves futures account
func (g *Gateio) QueryFuturesAccount(ctx context.Context, settle currency.Code) (*FuturesAccount, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}
	var response *FuturesAccount
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualAccountEPL, http.MethodGet, futuresPath+settle.Item.Lower+"/accounts", nil, nil, &response)
}

// GetFuturesAccountBooks retrieves account books
func (g *Gateio) GetFuturesAccountBooks(ctx context.Context, settle currency.Code, limit uint64, from, to time.Time, changingType string) ([]AccountBookItem, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
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
	var response []AccountBookItem
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualAccountBooksEPL, http.MethodGet, futuresPath+settle.Item.Lower+"/account_book", params, nil, &response)
}

// GetAllFuturesPositionsOfUsers list all positions of users.
func (g *Gateio) GetAllFuturesPositionsOfUsers(ctx context.Context, settle currency.Code, realPositionsOnly bool) ([]Position, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}
	params := url.Values{}
	if realPositionsOnly {
		params.Set("holding", "true")
	}
	var response []Position
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualPositionsEPL, http.MethodGet, futuresPath+settle.Item.Lower+"/positions", params, nil, &response)
}

// GetSinglePosition returns a single position
func (g *Gateio) GetSinglePosition(ctx context.Context, settle currency.Code, contract currency.Pair) (*Position, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}
	if contract.IsInvalid() {
		return nil, fmt.Errorf("%w, currency pair for contract must not be empty", errInvalidOrMissingContractParam)
	}
	var response *Position
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualPositionEPL, http.MethodPost, futuresPath+settle.Item.Lower+positionsPath+contract.String(), nil, nil, &response)
}

// UpdateFuturesPositionMargin represents account position margin for a futures contract.
func (g *Gateio) UpdateFuturesPositionMargin(ctx context.Context, settle currency.Code, change float64, contract currency.Pair) (*Position, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}
	if contract.IsInvalid() {
		return nil, fmt.Errorf("%w, currency pair for contract must not be empty", errInvalidOrMissingContractParam)
	}
	if change <= 0 {
		return nil, fmt.Errorf("%w, futures margin change must be positive", errChangeHasToBePositive)
	}
	params := url.Values{}
	params.Set("change", strconv.FormatFloat(change, 'f', -1, 64))
	var response *Position
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualUpdateMarginEPL, http.MethodPost, futuresPath+settle.Item.Lower+positionsPath+contract.String()+"/margin", params, nil, &response)
}

// UpdateFuturesPositionLeverage update position leverage
func (g *Gateio) UpdateFuturesPositionLeverage(ctx context.Context, settle currency.Code, contract currency.Pair, leverage, crossLeverageLimit float64) (*Position, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}
	if contract.IsInvalid() {
		return nil, fmt.Errorf("%w, currency pair for contract must not be empty", errInvalidOrMissingContractParam)
	}
	if leverage < 0 {
		return nil, errInvalidLeverageValue
	}
	params := url.Values{}
	params.Set("leverage", strconv.FormatFloat(leverage, 'f', -1, 64))
	if leverage == 0 && crossLeverageLimit > 0 {
		params.Set("cross_leverage_limit", strconv.FormatFloat(crossLeverageLimit, 'f', -1, 64))
	}
	var response *Position
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualUpdateLeverageEPL, http.MethodPost, futuresPath+settle.Item.Lower+positionsPath+contract.String()+"/leverage", params, nil, &response)
}

// UpdateFuturesPositionRiskLimit updates the position risk limit
func (g *Gateio) UpdateFuturesPositionRiskLimit(ctx context.Context, settle currency.Code, contract currency.Pair, riskLimit uint64) (*Position, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}
	if contract.IsInvalid() {
		return nil, fmt.Errorf("%w, currency pair for contract must not be empty", errInvalidOrMissingContractParam)
	}
	params := url.Values{}
	params.Set("risk_limit", strconv.FormatUint(riskLimit, 10))
	var response *Position
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualUpdateRiskEPL, http.MethodPost, futuresPath+settle.Item.Lower+positionsPath+contract.String()+"/risk_limit", params, nil, &response)
}

// EnableOrDisableDualMode enable or disable dual mode
// Before setting dual mode, make sure all positions are closed and no orders are open
func (g *Gateio) EnableOrDisableDualMode(ctx context.Context, settle currency.Code, dualMode bool) (*DualModeResponse, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}
	params := url.Values{}
	params.Set("dual_mode", strconv.FormatBool(dualMode))
	var response *DualModeResponse
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualToggleDualModeEPL, http.MethodGet, futuresPath+settle.Item.Lower+"/dual_mode", params, nil, &response)
}

// RetrivePositionDetailInDualMode retrieve position detail in dual mode
func (g *Gateio) RetrivePositionDetailInDualMode(ctx context.Context, settle currency.Code, contract currency.Pair) ([]Position, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}
	if contract.IsInvalid() {
		return nil, fmt.Errorf("%w, currency pair for contract must not be empty", errInvalidOrMissingContractParam)
	}
	var response []Position
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualPositionsDualModeEPL, http.MethodGet, futuresPath+settle.Item.Lower+"/dual_comp/positions/"+contract.String(), nil, nil, &response)
}

// UpdatePositionMarginInDualMode update position margin in dual mode
func (g *Gateio) UpdatePositionMarginInDualMode(ctx context.Context, settle currency.Code, contract currency.Pair, change float64, dualSide string) ([]Position, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}
	if contract.IsInvalid() {
		return nil, fmt.Errorf("%w, currency pair for contract must not be empty", errInvalidOrMissingContractParam)
	}
	params := url.Values{}
	params.Set("change", strconv.FormatFloat(change, 'f', -1, 64))
	if dualSide != "dual_long" && dualSide != "dual_short" {
		return nil, errors.New("invalid 'dual_side' should be 'dual_short' or 'dual_long'")
	}
	params.Set("dual_side", dualSide)
	var response []Position
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualUpdateMarginDualModeEPL, http.MethodPost, futuresPath+settle.Item.Lower+"/dual_comp/positions/"+contract.String()+"/margin", params, nil, &response)
}

// UpdatePositionLeverageInDualMode update position leverage in dual mode
func (g *Gateio) UpdatePositionLeverageInDualMode(ctx context.Context, settle currency.Code, contract currency.Pair, leverage, crossLeverageLimit float64) (*Position, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}
	if contract.IsInvalid() {
		return nil, fmt.Errorf("%w, currency pair for contract must not be empty", errInvalidOrMissingContractParam)
	}
	if leverage < 0 {
		return nil, errInvalidLeverageValue
	}
	params := url.Values{}
	params.Set("leverage", strconv.FormatFloat(leverage, 'f', -1, 64))
	if leverage == 0 && crossLeverageLimit > 0 {
		params.Set("cross_leverage_limit", strconv.FormatFloat(crossLeverageLimit, 'f', -1, 64))
	}
	var response *Position
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualUpdateLeverageDualModeEPL, http.MethodPost, futuresPath+settle.Item.Lower+"/dual_comp/positions/"+contract.String()+"/leverage", params, nil, &response)
}

// UpdatePositionRiskLimitInDualMode update position risk limit in dual mode
func (g *Gateio) UpdatePositionRiskLimitInDualMode(ctx context.Context, settle currency.Code, contract currency.Pair, riskLimit float64) ([]Position, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}
	if contract.IsInvalid() {
		return nil, fmt.Errorf("%w, currency pair for contract must not be empty", errInvalidOrMissingContractParam)
	}
	if riskLimit < 0 {
		return nil, errInvalidRiskLimit
	}
	params := url.Values{}
	params.Set("risk_limit", strconv.FormatFloat(riskLimit, 'f', -1, 64))
	var response []Position
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualUpdateRiskDualModeEPL, http.MethodPost, futuresPath+settle.Item.Lower+"/dual_comp/positions/"+contract.String()+"/risk_limit", params, nil, &response)
}

// PlaceFuturesOrder creates futures order
// Create a futures order
// Creating futures orders requires size, which is number of contracts instead of currency amount. You can use quanto_multiplier in contract detail response to know how much currency 1 size contract represents
// Zero-filled order cannot be retrieved 10 minutes after order cancellation. You will get a 404 not found for such orders
// Set reduce_only to true can keep the position from changing side when reducing position size
// In single position mode, to close a position, you need to set size to 0 and close to true
// In dual position mode, to close one side position, you need to set auto_size side, reduce_only to true and size to 0
func (g *Gateio) PlaceFuturesOrder(ctx context.Context, arg *ContractOrderCreateParams) (*Order, error) {
	if arg == nil {
		return nil, errNilArgument
	}
	if arg.Contract.IsEmpty() {
		return nil, fmt.Errorf("%w, currency pair for contract must not be empty", errInvalidOrMissingContractParam)
	}
	if arg.Size == 0 {
		return nil, fmt.Errorf("%w, specify positive number to make a bid, and negative number to ask", order.ErrSideIsInvalid)
	}
	if _, err := timeInForceFromString(arg.TimeInForce); err != nil {
		return nil, err
	}
	if arg.Price == "" {
		return nil, errInvalidPrice
	}
	if arg.Price == "0" && arg.TimeInForce != iocTIF && arg.TimeInForce != fokTIF {
		return nil, fmt.Errorf("%w: %q; only 'IOC' and 'FOK' allowed for market order", order.ErrUnsupportedTimeInForce, arg.TimeInForce)
	}
	if arg.AutoSize != "" && (arg.AutoSize == "close_long" || arg.AutoSize == "close_short") {
		return nil, errInvalidAutoSizeValue
	}
	if arg.Settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}

	var response *Order
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualSubmitOrderEPL, http.MethodPost, futuresPath+arg.Settle.Item.Lower+ordersPath, nil, &arg, &response)
}

// GetFuturesOrders retrieves list of futures orders
// Zero-filled order cannot be retrieved 10 minutes after order cancellation
func (g *Gateio) GetFuturesOrders(ctx context.Context, contract currency.Pair, status, lastID string, settle currency.Code, limit, offset uint64, countTotal int64) ([]Order, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}
	params := url.Values{}
	if !contract.IsEmpty() {
		params.Set("contract", contract.String())
	}
	if status != statusOpen && status != statusFinished {
		return nil, fmt.Errorf("%w, only 'open' and 'finished' status are supported", errInvalidOrderStatus)
	}
	params.Set("status", status)
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if lastID != "" {
		params.Set("last_id", lastID)
	}
	if countTotal == 1 && status != statusOpen {
		params.Set("count_total", strconv.FormatInt(countTotal, 10))
	} else if countTotal != 0 && countTotal != 1 {
		return nil, errInvalidCountTotalValue
	}
	var response []Order
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualGetOrdersEPL, http.MethodGet, futuresPath+settle.Item.Lower+ordersPath, params, nil, &response)
}

// CancelMultipleFuturesOpenOrders ancel all open orders
// Zero-filled order cannot be retrieved 10 minutes after order cancellation
func (g *Gateio) CancelMultipleFuturesOpenOrders(ctx context.Context, contract currency.Pair, side string, settle currency.Code) ([]Order, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}
	if contract.IsInvalid() {
		return nil, fmt.Errorf("%w, currency pair for contract must not be empty", errInvalidOrMissingContractParam)
	}
	params := url.Values{}
	if side != "" {
		params.Set("side", side)
	}
	params.Set("contract", contract.String())
	var response []Order
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualGetOrdersEPL, http.MethodDelete, futuresPath+settle.Item.Lower+ordersPath, params, nil, &response)
}

// PlaceBatchFuturesOrders creates a list of futures orders
// Up to 10 orders per request
// If any of the order's parameters are missing or in the wrong format, all of them will not be executed, and a http status 400 error will be returned directly
// If the parameters are checked and passed, all are executed. Even if there is a business logic error in the middle (such as insufficient funds), it will not affect other execution orders
// The returned result is in array format, and the order corresponds to the orders in the request body
// In the returned result, the succeeded field of type bool indicates whether the execution was successful or not
// If the execution is successful, the normal order content is included; if the execution fails, the label field is included to indicate the cause of the error
// In the rate limiting, each order is counted individually
func (g *Gateio) PlaceBatchFuturesOrders(ctx context.Context, settle currency.Code, args []ContractOrderCreateParams) ([]Order, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}
	if len(args) > 10 {
		return nil, errTooManyOrderRequest
	}
	for x := range args {
		if args[x].Size == 0 {
			return nil, fmt.Errorf("%w, specify positive number to make a bid, and negative number to ask", order.ErrSideIsInvalid)
		}
		if _, err := timeInForceFromString(args[x].TimeInForce); err != nil {
			return nil, err
		}
		if args[x].Price == "" {
			return nil, errInvalidPrice
		}
		if args[x].Price == "0" && args[x].TimeInForce != iocTIF && args[x].TimeInForce != fokTIF {
			return nil, fmt.Errorf("%w: %q; only 'ioc' and 'fok' allowed for market order", order.ErrUnsupportedTimeInForce, args[x].TimeInForce)
		}
		if args[x].Text != "" && !strings.HasPrefix(args[x].Text, "t-") {
			return nil, errInvalidTextValue
		}
		if args[x].AutoSize != "" && (args[x].AutoSize == "close_long" || args[x].AutoSize == "close_short") {
			return nil, errInvalidAutoSizeValue
		}
		if !args[x].Settle.Equal(currency.BTC) && !args[x].Settle.Equal(currency.USDT) {
			return nil, errEmptyOrInvalidSettlementCurrency
		}
	}
	var response []Order
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualSubmitBatchOrdersEPL, http.MethodPost, futuresPath+settle.Item.Lower+"/batch_orders", nil, &args, &response)
}

// GetSingleFuturesOrder retrieves a single order by its identifier
func (g *Gateio) GetSingleFuturesOrder(ctx context.Context, settle currency.Code, orderID string) (*Order, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}
	if orderID == "" {
		return nil, fmt.Errorf("%w, 'order_id' cannot be empty", errInvalidOrderID)
	}
	var response *Order
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualFetchOrderEPL, http.MethodGet, futuresPath+settle.Item.Lower+"/orders/"+orderID, nil, nil, &response)
}

// CancelSingleFuturesOrder cancel a single order
func (g *Gateio) CancelSingleFuturesOrder(ctx context.Context, settle currency.Code, orderID string) (*Order, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}
	if orderID == "" {
		return nil, fmt.Errorf("%w, 'order_id' cannot be empty", errInvalidOrderID)
	}
	var response *Order
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualCancelOrderEPL, http.MethodDelete, futuresPath+settle.Item.Lower+"/orders/"+orderID, nil, nil, &response)
}

// AmendFuturesOrder amends an existing futures order
func (g *Gateio) AmendFuturesOrder(ctx context.Context, settle currency.Code, orderID string, arg AmendFuturesOrderParam) (*Order, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}
	if orderID == "" {
		return nil, fmt.Errorf("%w, 'order_id' cannot be empty", errInvalidOrderID)
	}
	if arg.Size <= 0 && arg.Price <= 0 {
		return nil, errors.New("missing update 'size' or 'price', please specify 'size' or 'price' or both information")
	}
	var response *Order
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualAmendOrderEPL, http.MethodPut, futuresPath+settle.Item.Lower+"/orders/"+orderID, nil, &arg, &response)
}

// GetMyFuturesTradingHistory retrieves authenticated account's futures trading history
func (g *Gateio) GetMyFuturesTradingHistory(ctx context.Context, settle currency.Code, lastID, orderID string, contract currency.Pair, limit, offset, countTotal uint64) ([]TradingHistoryItem, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
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
	var response []TradingHistoryItem
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualTradingHistoryEPL, http.MethodGet, futuresPath+settle.Item.Lower+"/my_trades", params, nil, &response)
}

// GetFuturesPositionCloseHistory lists position close history
func (g *Gateio) GetFuturesPositionCloseHistory(ctx context.Context, settle currency.Code, contract currency.Pair, limit, offset uint64, from, to time.Time) ([]PositionCloseHistoryResponse, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
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
	var response []PositionCloseHistoryResponse
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualClosePositionEPL, http.MethodGet, futuresPath+settle.Item.Lower+"/position_close", params, nil, &response)
}

// GetFuturesLiquidationHistory list liquidation history
func (g *Gateio) GetFuturesLiquidationHistory(ctx context.Context, settle currency.Code, contract currency.Pair, limit uint64, at time.Time) ([]LiquidationHistoryItem, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
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
	var response []LiquidationHistoryItem
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualLiquidationHistoryEPL, http.MethodGet, futuresPath+settle.Item.Lower+"/liquidates", params, nil, &response)
}

// CountdownCancelOrders represents a trigger time response
func (g *Gateio) CountdownCancelOrders(ctx context.Context, settle currency.Code, arg CountdownParams) (*TriggerTimeResponse, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}
	if arg.Timeout < 0 {
		return nil, errInvalidTimeout
	}
	var response *TriggerTimeResponse
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualCancelTriggerOrdersEPL, http.MethodPost, futuresPath+settle.Item.Lower+"/countdown_cancel_all", nil, &arg, &response)
}

// CreatePriceTriggeredFuturesOrder create a price-triggered order
func (g *Gateio) CreatePriceTriggeredFuturesOrder(ctx context.Context, settle currency.Code, arg *FuturesPriceTriggeredOrderParam) (*OrderID, error) {
	if arg == nil {
		return nil, errNilArgument
	}
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}
	if arg.Initial.Contract.IsEmpty() {
		return nil, fmt.Errorf("%w, currency pair for contract must not be empty", errInvalidOrMissingContractParam)
	}
	if arg.Initial.Price < 0 {
		return nil, fmt.Errorf("%w, price must be greater than 0", errInvalidPrice)
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
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualSubmitTriggerOrderEPL, http.MethodPost, futuresPath+settle.Item.Lower+priceOrdersPaths, nil, &arg, &response)
}

// ListAllFuturesAutoOrders lists all open orders
func (g *Gateio) ListAllFuturesAutoOrders(ctx context.Context, status string, settle currency.Code, contract currency.Pair, limit, offset uint64) ([]PriceTriggeredOrder, error) {
	if status != statusOpen && status != statusFinished {
		return nil, fmt.Errorf("%w status: %s", errInvalidOrderStatus, status)
	}
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
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
	var response []PriceTriggeredOrder
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualListOpenOrdersEPL, http.MethodGet, futuresPath+settle.Item.Lower+priceOrdersPaths, params, nil, &response)
}

// CancelAllFuturesOpenOrders cancels all futures open orders
func (g *Gateio) CancelAllFuturesOpenOrders(ctx context.Context, settle currency.Code, contract currency.Pair) ([]PriceTriggeredOrder, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}
	if contract.IsInvalid() {
		return nil, fmt.Errorf("%w, currency pair for contract must not be empty", errInvalidOrMissingContractParam)
	}
	params := url.Values{}
	params.Set("contract", contract.String())
	var response []PriceTriggeredOrder
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualCancelOpenOrdersEPL, http.MethodDelete, futuresPath+settle.Item.Lower+priceOrdersPaths, params, nil, &response)
}

// GetSingleFuturesPriceTriggeredOrder retrieves a single price triggered order
func (g *Gateio) GetSingleFuturesPriceTriggeredOrder(ctx context.Context, settle currency.Code, orderID string) (*PriceTriggeredOrder, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}
	if orderID == "" {
		return nil, errInvalidOrderID
	}
	var response *PriceTriggeredOrder
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualGetTriggerOrderEPL, http.MethodGet, futuresPath+settle.Item.Lower+"/price_orders/"+orderID, nil, nil, &response)
}

// CancelFuturesPriceTriggeredOrder cancel a price-triggered order
func (g *Gateio) CancelFuturesPriceTriggeredOrder(ctx context.Context, settle currency.Code, orderID string) (*PriceTriggeredOrder, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}
	if orderID == "" {
		return nil, errInvalidOrderID
	}
	var response *PriceTriggeredOrder
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualCancelTriggerOrderEPL, http.MethodDelete, futuresPath+settle.Item.Lower+"/price_orders/"+orderID, nil, nil, &response)
}

// *************************************** Delivery ***************************************

// GetAllDeliveryContracts retrieves all futures contracts
func (g *Gateio) GetAllDeliveryContracts(ctx context.Context, settle currency.Code) ([]DeliveryContract, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}
	var contracts []DeliveryContract
	return contracts, g.SendHTTPRequest(ctx, exchange.RestSpot, publicDeliveryContractsEPL, deliveryPath+settle.Item.Lower+"/contracts", &contracts)
}

// GetDeliveryContract retrieves a single delivery contract instance
func (g *Gateio) GetDeliveryContract(ctx context.Context, settle currency.Code, contract currency.Pair) (*DeliveryContract, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}
	var deliveryContract *DeliveryContract
	return deliveryContract, g.SendHTTPRequest(ctx, exchange.RestSpot, publicDeliveryContractsEPL, deliveryPath+settle.Item.Lower+"/contracts/"+contract.String(), &deliveryContract)
}

// GetDeliveryOrderbook delivery orderbook
func (g *Gateio) GetDeliveryOrderbook(ctx context.Context, settle currency.Code, interval string, contract currency.Pair, limit uint64, withOrderbookID bool) (*Orderbook, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}
	if contract.IsInvalid() {
		return nil, errInvalidOrMissingContractParam
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
	return orderbook, g.SendHTTPRequest(ctx, exchange.RestSpot, publicOrderbookDeliveryEPL, common.EncodeURLValues(deliveryPath+settle.Item.Lower+"/order_book", params), &orderbook)
}

// GetDeliveryTradingHistory retrieves futures trading history
func (g *Gateio) GetDeliveryTradingHistory(ctx context.Context, settle currency.Code, lastID string, contract currency.Pair, limit uint64, from, to time.Time) ([]TradingHistoryItem, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}
	if contract.IsInvalid() {
		return nil, errInvalidOrMissingContractParam
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
	var histories []TradingHistoryItem
	return histories, g.SendHTTPRequest(ctx, exchange.RestSpot, publicTradingHistoryDeliveryEPL, common.EncodeURLValues(deliveryPath+settle.Item.Lower+"/trades", params), &histories)
}

// GetDeliveryFuturesCandlesticks retrieves specified contract candlesticks
func (g *Gateio) GetDeliveryFuturesCandlesticks(ctx context.Context, settle currency.Code, contract currency.Pair, from, to time.Time, limit uint64, interval kline.Interval) ([]FuturesCandlestick, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}
	if contract.IsInvalid() {
		return nil, errInvalidOrMissingContractParam
	}
	params := url.Values{}
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
	if int64(interval) != 0 {
		intervalString, err := getIntervalString(interval)
		if err != nil {
			return nil, err
		}
		params.Set("interval", intervalString)
	}
	var candlesticks []FuturesCandlestick
	return candlesticks, g.SendHTTPRequest(ctx, exchange.RestSpot, publicCandleSticksDeliveryEPL, common.EncodeURLValues(deliveryPath+settle.Item.Lower+"/candlesticks", params), &candlesticks)
}

// GetDeliveryFutureTickers retrieves futures ticker information for a specific settle and contract info.
func (g *Gateio) GetDeliveryFutureTickers(ctx context.Context, settle currency.Code, contract currency.Pair) ([]FuturesTicker, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}
	params := url.Values{}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	var tickers []FuturesTicker
	return tickers, g.SendHTTPRequest(ctx, exchange.RestSpot, publicTickersDeliveryEPL, common.EncodeURLValues(deliveryPath+settle.Item.Lower+"/tickers", params), &tickers)
}

// GetDeliveryInsuranceBalanceHistory retrieves delivery futures insurance balance history
func (g *Gateio) GetDeliveryInsuranceBalanceHistory(ctx context.Context, settle currency.Code, limit uint64) ([]InsuranceBalance, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}
	params := url.Values{}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var balances []InsuranceBalance
	return balances, g.SendHTTPRequest(ctx, exchange.RestSpot, publicInsuranceDeliveryEPL, common.EncodeURLValues(deliveryPath+settle.Item.Lower+"/insurance", params), &balances)
}

// GetDeliveryFuturesAccounts retrieves futures account
func (g *Gateio) GetDeliveryFuturesAccounts(ctx context.Context, settle currency.Code) (*FuturesAccount, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}
	var response *FuturesAccount
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryAccountEPL, http.MethodGet, deliveryPath+settle.Item.Lower+"/accounts", nil, nil, &response)
}

// GetDeliveryAccountBooks retrieves account books
func (g *Gateio) GetDeliveryAccountBooks(ctx context.Context, settle currency.Code, limit uint64, from, to time.Time, changingType string) ([]AccountBookItem, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
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
	var response []AccountBookItem
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryAccountBooksEPL, http.MethodGet, deliveryPath+settle.Item.Lower+"/account_book", params, nil, &response)
}

// GetAllDeliveryPositionsOfUser retrieves all positions of user
func (g *Gateio) GetAllDeliveryPositionsOfUser(ctx context.Context, settle currency.Code) (*Position, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}
	var response *Position
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryPositionsEPL, http.MethodGet, deliveryPath+settle.Item.Lower+"/positions", nil, nil, &response)
}

// GetSingleDeliveryPosition get single position
func (g *Gateio) GetSingleDeliveryPosition(ctx context.Context, settle currency.Code, contract currency.Pair) (*Position, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}
	if contract.IsInvalid() {
		return nil, fmt.Errorf("%w, currency pair for contract must not be empty", errInvalidOrMissingContractParam)
	}
	var response *Position
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryPositionsEPL, http.MethodGet, deliveryPath+settle.Item.Lower+positionsPath+contract.String(), nil, nil, &response)
}

// UpdateDeliveryPositionMargin updates position margin
func (g *Gateio) UpdateDeliveryPositionMargin(ctx context.Context, settle currency.Code, change float64, contract currency.Pair) (*Position, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}
	if contract.IsInvalid() {
		return nil, fmt.Errorf("%w, currency pair for contract must not be empty", errInvalidOrMissingContractParam)
	}
	if change <= 0 {
		return nil, fmt.Errorf("%w, futures margin change must be positive", errChangeHasToBePositive)
	}
	params := url.Values{}
	params.Set("change", strconv.FormatFloat(change, 'f', -1, 64))
	var response *Position
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryUpdateMarginEPL, http.MethodPost, deliveryPath+settle.Item.Lower+positionsPath+contract.String()+"/margin", params, nil, &response)
}

// UpdateDeliveryPositionLeverage updates position leverage
func (g *Gateio) UpdateDeliveryPositionLeverage(ctx context.Context, settle currency.Code, contract currency.Pair, leverage float64) (*Position, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}
	if contract.IsInvalid() {
		return nil, fmt.Errorf("%w, currency pair for contract must not be empty", errInvalidOrMissingContractParam)
	}
	if leverage < 0 {
		return nil, errInvalidLeverageValue
	}
	params := url.Values{}
	params.Set("leverage", strconv.FormatFloat(leverage, 'f', -1, 64))
	var response *Position
	return response, g.SendAuthenticatedHTTPRequest(ctx,
		exchange.RestSpot, deliveryUpdateLeverageEPL, http.MethodPost, deliveryPath+settle.Item.Lower+positionsPath+contract.String()+"/leverage", params, nil, &response)
}

// UpdateDeliveryPositionRiskLimit update position risk limit
func (g *Gateio) UpdateDeliveryPositionRiskLimit(ctx context.Context, settle currency.Code, contract currency.Pair, riskLimit uint64) (*Position, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}
	if contract.IsInvalid() {
		return nil, fmt.Errorf("%w, currency pair for contract must not be empty", errInvalidOrMissingContractParam)
	}
	params := url.Values{}
	params.Set("risk_limit", strconv.FormatUint(riskLimit, 10))
	var response *Position
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryUpdateRiskLimitEPL, http.MethodPost, deliveryPath+settle.Item.Lower+positionsPath+contract.String()+"/risk_limit", params, nil, &response)
}

// PlaceDeliveryOrder create a futures order
// Zero-filled order cannot be retrieved 10 minutes after order cancellation
func (g *Gateio) PlaceDeliveryOrder(ctx context.Context, arg *ContractOrderCreateParams) (*Order, error) {
	if arg == nil {
		return nil, errNilArgument
	}
	if arg.Contract.IsEmpty() {
		return nil, fmt.Errorf("%w, currency pair for contract must not be empty", errInvalidOrMissingContractParam)
	}
	if arg.Size == 0 {
		return nil, fmt.Errorf("%w, specify positive number to make a bid, and negative number to ask", order.ErrSideIsInvalid)
	}
	if _, err := timeInForceFromString(arg.TimeInForce); err != nil {
		return nil, err
	}
	if arg.Price == "" {
		return nil, errInvalidPrice
	}
	if arg.AutoSize != "" && (arg.AutoSize == "close_long" || arg.AutoSize == "close_short") {
		return nil, errInvalidAutoSizeValue
	}
	if arg.Settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}
	var response *Order
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliverySubmitOrderEPL, http.MethodPost, deliveryPath+arg.Settle.Item.Lower+ordersPath, nil, &arg, &response)
}

// GetDeliveryOrders list futures orders
// Zero-filled order cannot be retrieved 10 minutes after order cancellation
func (g *Gateio) GetDeliveryOrders(ctx context.Context, contract currency.Pair, status string, settle currency.Code, lastID string, limit, offset uint64, countTotal int64) ([]Order, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}
	params := url.Values{}
	if !contract.IsEmpty() {
		params.Set("contract", contract.String())
	}
	if status != statusOpen && status != statusFinished {
		return nil, fmt.Errorf("%w, only 'open' and 'finished' status are supported", errInvalidOrderStatus)
	}
	params.Set("status", status)
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if lastID != "" {
		params.Set("last_id", lastID)
	}
	if countTotal == 1 && status != statusOpen {
		params.Set("count_total", strconv.FormatInt(countTotal, 10))
	} else if countTotal != 0 && countTotal != 1 {
		return nil, errInvalidCountTotalValue
	}
	var response []Order
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryGetOrdersEPL, http.MethodGet, deliveryPath+settle.Item.Lower+ordersPath, params, nil, &response)
}

// CancelMultipleDeliveryOrders cancel all open orders matched
// Zero-filled order cannot be retrieved 10 minutes after order cancellation
func (g *Gateio) CancelMultipleDeliveryOrders(ctx context.Context, contract currency.Pair, side string, settle currency.Code) ([]Order, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}
	if contract.IsInvalid() {
		return nil, fmt.Errorf("%w, currency pair for contract must not be empty", errInvalidOrMissingContractParam)
	}
	params := url.Values{}
	if side == "ask" || side == "bid" {
		params.Set("side", side)
	}
	params.Set("contract", contract.String())
	var response []Order
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryCancelOrdersEPL, http.MethodDelete, deliveryPath+settle.Item.Lower+ordersPath, params, nil, &response)
}

// GetSingleDeliveryOrder Get a single order
// Zero-filled order cannot be retrieved 10 minutes after order cancellation
func (g *Gateio) GetSingleDeliveryOrder(ctx context.Context, settle currency.Code, orderID string) (*Order, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}
	if orderID == "" {
		return nil, fmt.Errorf("%w, 'order_id' cannot be empty", errInvalidOrderID)
	}
	var response *Order
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryGetOrderEPL, http.MethodGet, deliveryPath+settle.Item.Lower+"/orders/"+orderID, nil, nil, &response)
}

// CancelSingleDeliveryOrder cancel a single order
func (g *Gateio) CancelSingleDeliveryOrder(ctx context.Context, settle currency.Code, orderID string) (*Order, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}
	if orderID == "" {
		return nil, fmt.Errorf("%w, 'order_id' cannot be empty", errInvalidOrderID)
	}
	var response *Order
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryCancelOrderEPL, http.MethodDelete, deliveryPath+settle.Item.Lower+"/orders/"+orderID, nil, nil, &response)
}

// GetMyDeliveryTradingHistory retrieves authenticated account delivery futures trading history
func (g *Gateio) GetMyDeliveryTradingHistory(ctx context.Context, settle currency.Code, orderID string, contract currency.Pair, limit, offset, countTotal uint64, lastID string) ([]TradingHistoryItem, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
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
	var response []TradingHistoryItem
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryTradingHistoryEPL, http.MethodGet, deliveryPath+settle.Item.Lower+"/my_trades", params, nil, &response)
}

// GetDeliveryPositionCloseHistory retrieves position history
func (g *Gateio) GetDeliveryPositionCloseHistory(ctx context.Context, settle currency.Code, contract currency.Pair, limit, offset uint64, from, to time.Time) ([]PositionCloseHistoryResponse, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
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
	var response []PositionCloseHistoryResponse
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryCloseHistoryEPL, http.MethodGet, deliveryPath+settle.Item.Lower+"/position_close", params, nil, &response)
}

// GetDeliveryLiquidationHistory lists liquidation history
func (g *Gateio) GetDeliveryLiquidationHistory(ctx context.Context, settle currency.Code, contract currency.Pair, limit uint64, at time.Time) ([]LiquidationHistoryItem, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
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
	var response []LiquidationHistoryItem
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryLiquidationHistoryEPL, http.MethodGet, deliveryPath+settle.Item.Lower+"/liquidates", params, nil, &response)
}

// GetDeliverySettlementHistory retrieves settlement history
func (g *Gateio) GetDeliverySettlementHistory(ctx context.Context, settle currency.Code, contract currency.Pair, limit uint64, at time.Time) ([]SettlementHistoryItem, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
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
	var response []SettlementHistoryItem
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliverySettlementHistoryEPL, http.MethodGet, deliveryPath+settle.Item.Lower+"/settlements", params, nil, &response)
}

// GetDeliveryPriceTriggeredOrder creates a price-triggered order
func (g *Gateio) GetDeliveryPriceTriggeredOrder(ctx context.Context, settle currency.Code, arg *FuturesPriceTriggeredOrderParam) (*OrderID, error) {
	if arg == nil {
		return nil, errNilArgument
	}
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}
	if arg.Initial.Contract.IsEmpty() {
		return nil, fmt.Errorf("%w, currency pair for contract must not be empty", errInvalidOrMissingContractParam)
	}
	if arg.Initial.Price < 0 {
		return nil, fmt.Errorf("%w, price must be greater than 0", errInvalidPrice)
	}
	if arg.Initial.Size <= 0 {
		return nil, errors.New("invalid argument: initial.size out of range")
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
		return nil, errors.New("invalid argument: trigger.price")
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
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryGetTriggerOrderEPL, http.MethodPost, deliveryPath+settle.Item.Lower+priceOrdersPaths, nil, &arg, &response)
}

// GetDeliveryAllAutoOrder retrieves all auto orders
func (g *Gateio) GetDeliveryAllAutoOrder(ctx context.Context, status string, settle currency.Code, contract currency.Pair, limit, offset uint64) ([]PriceTriggeredOrder, error) {
	if status != statusOpen && status != statusFinished {
		return nil, fmt.Errorf("%w status %s", errInvalidOrderStatus, status)
	}
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
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
	var response []PriceTriggeredOrder
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryAutoOrdersEPL, http.MethodGet, deliveryPath+settle.Item.Lower+priceOrdersPaths, params, nil, &response)
}

// CancelAllDeliveryPriceTriggeredOrder cancels all delivery price triggered orders
func (g *Gateio) CancelAllDeliveryPriceTriggeredOrder(ctx context.Context, settle currency.Code, contract currency.Pair) ([]PriceTriggeredOrder, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}
	if contract.IsInvalid() {
		return nil, fmt.Errorf("%w, currency pair for contract must not be empty", errInvalidOrMissingContractParam)
	}
	params := url.Values{}
	params.Set("contract", contract.String())
	var response []PriceTriggeredOrder
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryCancelTriggerOrdersEPL, http.MethodDelete, deliveryPath+settle.Item.Lower+priceOrdersPaths, params, nil, &response)
}

// GetSingleDeliveryPriceTriggeredOrder retrieves a single price triggered order
func (g *Gateio) GetSingleDeliveryPriceTriggeredOrder(ctx context.Context, settle currency.Code, orderID string) (*PriceTriggeredOrder, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}
	if orderID == "" {
		return nil, errInvalidOrderID
	}
	var response *PriceTriggeredOrder
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryGetTriggerOrderEPL, http.MethodGet, deliveryPath+settle.Item.Lower+"/price_orders/"+orderID, nil, nil, &response)
}

// CancelDeliveryPriceTriggeredOrder cancel a price-triggered order
func (g *Gateio) CancelDeliveryPriceTriggeredOrder(ctx context.Context, settle currency.Code, orderID string) (*PriceTriggeredOrder, error) {
	if settle.IsEmpty() {
		return nil, errEmptyOrInvalidSettlementCurrency
	}
	if orderID == "" {
		return nil, errInvalidOrderID
	}
	var response *PriceTriggeredOrder
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryCancelTriggerOrderEPL, http.MethodDelete, deliveryPath+settle.Item.Lower+"/price_orders/"+orderID, nil, nil, &response)
}

// ********************************** Options ***************************************************

// GetAllOptionsUnderlyings retrieves all option underlyings
func (g *Gateio) GetAllOptionsUnderlyings(ctx context.Context) ([]OptionUnderlying, error) {
	var response []OptionUnderlying
	return response, g.SendHTTPRequest(ctx, exchange.RestSpot, publicUnderlyingOptionsEPL, gateioOptionUnderlyings, &response)
}

// GetExpirationTime return the expiration time for the provided underlying.
func (g *Gateio) GetExpirationTime(ctx context.Context, underlying string) (time.Time, error) {
	if underlying == "" {
		return time.Time{}, errInvalidUnderlying
	}
	var timestamps []types.Time
	err := g.SendHTTPRequest(ctx, exchange.RestSpot, publicExpirationOptionsEPL, gateioOptionExpiration+"?underlying="+underlying, &timestamps)
	if err != nil {
		return time.Time{}, err
	}
	if len(timestamps) == 0 {
		return time.Time{}, errNoValidResponseFromServer
	}
	return timestamps[0].Time(), nil
}

// GetAllContractOfUnderlyingWithinExpiryDate retrieves list of contracts of the specified underlying and expiry time.
func (g *Gateio) GetAllContractOfUnderlyingWithinExpiryDate(ctx context.Context, underlying string, expTime time.Time) ([]OptionContract, error) {
	params := url.Values{}
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	params.Set("underlying", underlying)
	if !expTime.IsZero() {
		params.Set("expires", strconv.FormatInt(expTime.Unix(), 10))
	}
	var contracts []OptionContract
	return contracts, g.SendHTTPRequest(ctx, exchange.RestSpot, publicContractsOptionsEPL, common.EncodeURLValues(gateioOptionContracts, params), &contracts)
}

// GetOptionsSpecifiedContractDetail query specified contract detail
func (g *Gateio) GetOptionsSpecifiedContractDetail(ctx context.Context, contract currency.Pair) (*OptionContract, error) {
	if contract.IsInvalid() {
		return nil, errInvalidOrMissingContractParam
	}
	var contr *OptionContract
	return contr, g.SendHTTPRequest(ctx, exchange.RestSpot, publicContractsOptionsEPL, gateioOptionContracts+"/"+contract.String(), &contr)
}

// GetSettlementHistory retrieves list of settlement history
func (g *Gateio) GetSettlementHistory(ctx context.Context, underlying string, offset, limit uint64, from, to time.Time) ([]OptionSettlement, error) {
	if underlying == "" {
		return nil, errInvalidUnderlying
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
	var settlements []OptionSettlement
	return settlements, g.SendHTTPRequest(ctx, exchange.RestSpot, publicSettlementOptionsEPL, common.EncodeURLValues(gateioOptionSettlement, params), &settlements)
}

// GetOptionsSpecifiedContractsSettlement retrieve a single contract settlement detail passing the underlying and contract name
func (g *Gateio) GetOptionsSpecifiedContractsSettlement(ctx context.Context, contract currency.Pair, underlying string, at int64) (*OptionSettlement, error) {
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	if contract.IsInvalid() {
		return nil, errInvalidOrMissingContractParam
	}
	params := url.Values{}
	params.Set("underlying", underlying)
	params.Set("at", strconv.FormatInt(at, 10))
	var settlement *OptionSettlement
	return settlement, g.SendHTTPRequest(ctx, exchange.RestSpot, publicSettlementOptionsEPL, common.EncodeURLValues(gateioOptionSettlement+"/"+contract.String(), params), &settlement)
}

// GetMyOptionsSettlements retrieves accounts option settlements.
func (g *Gateio) GetMyOptionsSettlements(ctx context.Context, underlying string, contract currency.Pair, offset, limit uint64, to time.Time) ([]MyOptionSettlement, error) {
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
	var settlements []MyOptionSettlement
	return settlements, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsSettlementsEPL, http.MethodGet, gateioOptionMySettlements, params, nil, &settlements)
}

// GetOptionsOrderbook returns the orderbook data for the given contract.
func (g *Gateio) GetOptionsOrderbook(ctx context.Context, contract currency.Pair, interval string, limit uint64, withOrderbookID bool) (*Orderbook, error) {
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
	return response, g.SendHTTPRequest(ctx, exchange.RestSpot, publicOrderbookOptionsEPL, common.EncodeURLValues(gateioOptionsOrderbook, params), &response)
}

// GetOptionAccounts lists option accounts
func (g *Gateio) GetOptionAccounts(ctx context.Context) (*OptionAccount, error) {
	var resp *OptionAccount
	return resp, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsAccountsEPL, http.MethodGet, gateioOptionAccounts, nil, nil, &resp)
}

// GetAccountChangingHistory retrieves list of account changing history
func (g *Gateio) GetAccountChangingHistory(ctx context.Context, offset, limit uint64, from, to time.Time, changingType string) ([]AccountBook, error) {
	params := url.Values{}
	if changingType != "" {
		params.Set("type", changingType)
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
	var accountBook []AccountBook
	return accountBook, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsAccountBooksEPL, http.MethodGet, gateioOptionsAccountbook, params, nil, &accountBook)
}

// GetUsersPositionSpecifiedUnderlying lists user's positions of specified underlying
func (g *Gateio) GetUsersPositionSpecifiedUnderlying(ctx context.Context, underlying string) ([]UsersPositionForUnderlying, error) {
	params := url.Values{}
	if underlying != "" {
		params.Set("underlying", underlying)
	}
	var response []UsersPositionForUnderlying
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsPositions, http.MethodGet, gateioOptionsPosition, params, nil, &response)
}

// GetSpecifiedContractPosition retrieves specified contract position
func (g *Gateio) GetSpecifiedContractPosition(ctx context.Context, contract currency.Pair) (*UsersPositionForUnderlying, error) {
	if contract.IsInvalid() {
		return nil, errInvalidOrMissingContractParam
	}
	var response *UsersPositionForUnderlying
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsPositions, http.MethodGet, gateioOptionsPosition+"/"+contract.String(), nil, nil, &response)
}

// GetUsersLiquidationHistoryForSpecifiedUnderlying retrieves user's liquidation history of specified underlying
func (g *Gateio) GetUsersLiquidationHistoryForSpecifiedUnderlying(ctx context.Context, underlying string, contract currency.Pair) ([]ContractClosePosition, error) {
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	params := url.Values{}
	params.Set("underlying", underlying)
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	var response []ContractClosePosition
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsLiquidationHistoryEPL, http.MethodGet, gateioOptionsPositionClose, params, nil, &response)
}

// PlaceOptionOrder creates an options order
func (g *Gateio) PlaceOptionOrder(ctx context.Context, arg *OptionOrderParam) (*OptionOrderResponse, error) {
	if arg.Contract == "" {
		return nil, errInvalidOrMissingContractParam
	}
	if arg.OrderSize == 0 {
		return nil, errInvalidOrderSize
	}
	if arg.Iceberg < 0 {
		arg.Iceberg = 0
	}
	if arg.TimeInForce != gtcTIF && arg.TimeInForce != iocTIF && arg.TimeInForce != pocTIF {
		arg.TimeInForce = ""
	}
	if arg.TimeInForce == iocTIF || arg.Price < 0 {
		arg.Price = 0
	}
	var response *OptionOrderResponse
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsSubmitOrderEPL, http.MethodPost, gateioOptionsOrders, nil, &arg, &response)
}

// GetOptionFuturesOrders retrieves futures orders
func (g *Gateio) GetOptionFuturesOrders(ctx context.Context, contract currency.Pair, underlying, status string, offset, limit uint64, from, to time.Time) ([]OptionOrderResponse, error) {
	params := url.Values{}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	if underlying != "" {
		params.Set("underlying", underlying)
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
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() && ((!from.IsZero() && to.After(from)) || to.Before(time.Now())) {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	var response []OptionOrderResponse
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsOrdersEPL, http.MethodGet, gateioOptionsOrders, params, nil, &response)
}

// CancelMultipleOptionOpenOrders cancels all open orders matched
func (g *Gateio) CancelMultipleOptionOpenOrders(ctx context.Context, contract currency.Pair, underlying, side string) ([]OptionOrderResponse, error) {
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
	var response []OptionOrderResponse
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsCancelOrdersEPL, http.MethodDelete, gateioOptionsOrders, params, nil, &response)
}

// GetSingleOptionOrder retrieves a single option order
func (g *Gateio) GetSingleOptionOrder(ctx context.Context, orderID string) (*OptionOrderResponse, error) {
	if orderID == "" {
		return nil, errInvalidOrderID
	}
	var o *OptionOrderResponse
	return o, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsOrderEPL, http.MethodGet, gateioOptionsOrders+"/"+orderID, nil, nil, &o)
}

// CancelOptionSingleOrder cancel a single order.
func (g *Gateio) CancelOptionSingleOrder(ctx context.Context, orderID string) (*OptionOrderResponse, error) {
	if orderID == "" {
		return nil, errInvalidOrderID
	}
	var response *OptionOrderResponse
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsCancelOrderEPL, http.MethodDelete, "options/orders/"+orderID, nil, nil, &response)
}

// GetMyOptionsTradingHistory retrieves authenticated account's option trading history
func (g *Gateio) GetMyOptionsTradingHistory(ctx context.Context, underlying string, contract currency.Pair, offset, limit uint64, from, to time.Time) ([]OptionTradingHistory, error) {
	if underlying == "" {
		return nil, errInvalidUnderlying
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
	var resp []OptionTradingHistory
	return resp, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsTradingHistoryEPL, http.MethodGet, gateioOptionsMyTrades, params, nil, &resp)
}

// GetOptionsTickers lists  tickers of options contracts
func (g *Gateio) GetOptionsTickers(ctx context.Context, underlying string) ([]OptionsTicker, error) {
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	underlying = strings.ToUpper(underlying)
	var response []OptionsTicker
	return response, g.SendHTTPRequest(ctx, exchange.RestSpot, publicTickerOptionsEPL, gateioOptionsTickers+"?underlying="+underlying, &response)
}

// GetOptionUnderlyingTickers retrieves options underlying ticker
func (g *Gateio) GetOptionUnderlyingTickers(ctx context.Context, underlying string) (*OptionsUnderlyingTicker, error) {
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	var respos *OptionsUnderlyingTicker
	return respos, g.SendHTTPRequest(ctx, exchange.RestSpot, publicUnderlyingTickerOptionsEPL, "options/underlying/tickers/"+underlying, &respos)
}

// GetOptionFuturesCandlesticks retrieves option futures candlesticks
func (g *Gateio) GetOptionFuturesCandlesticks(ctx context.Context, contract currency.Pair, limit uint64, from, to time.Time, interval kline.Interval) ([]FuturesCandlestick, error) {
	if contract.IsInvalid() {
		return nil, errInvalidOrMissingContractParam
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
	var candles []FuturesCandlestick
	return candles, g.SendHTTPRequest(ctx, exchange.RestSpot, publicCandleSticksOptionsEPL, common.EncodeURLValues(gateioOptionCandlesticks, params), &candles)
}

// GetOptionFuturesMarkPriceCandlesticks retrieves mark price candlesticks of an underlying
func (g *Gateio) GetOptionFuturesMarkPriceCandlesticks(ctx context.Context, underlying string, limit uint64, from, to time.Time, interval kline.Interval) ([]FuturesCandlestick, error) {
	if underlying == "" {
		return nil, errInvalidUnderlying
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
	var candles []FuturesCandlestick
	return candles, g.SendHTTPRequest(ctx, exchange.RestSpot, publicMarkpriceCandleSticksOptionsEPL, common.EncodeURLValues(gateioOptionUnderlyingCandlesticks, params), &candles)
}

// GetOptionsTradeHistory retrieves options trade history
func (g *Gateio) GetOptionsTradeHistory(ctx context.Context, contract currency.Pair, callType string, offset, limit uint64, from, to time.Time) ([]TradingHistoryItem, error) {
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
	var trades []TradingHistoryItem
	return trades, g.SendHTTPRequest(ctx, exchange.RestSpot, publicTradeHistoryOptionsEPL, common.EncodeURLValues(gateioOptionsTrades, params), &trades)
}

// ********************************** Flash_SWAP *************************

// GetSupportedFlashSwapCurrencies retrieves all supported currencies in flash swap
func (g *Gateio) GetSupportedFlashSwapCurrencies(ctx context.Context) ([]SwapCurrencies, error) {
	var currencies []SwapCurrencies
	return currencies, g.SendHTTPRequest(ctx, exchange.RestSpot, publicFlashSwapEPL, gateioFlashSwapCurrencies, &currencies)
}

// CreateFlashSwapOrder creates a new flash swap order
// initiate a flash swap preview in advance because order creation requires a preview result
func (g *Gateio) CreateFlashSwapOrder(ctx context.Context, arg FlashSwapOrderParams) (*FlashSwapOrderResponse, error) {
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
		return nil, fmt.Errorf("%w, sell_amount can not be less than or equal to 0", errInvalidAmount)
	}
	if arg.BuyAmount <= 0 {
		return nil, fmt.Errorf("%w, buy_amount amount can not be less than or equal to 0", errInvalidAmount)
	}
	var response *FlashSwapOrderResponse
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, flashSwapOrderEPL, http.MethodPost, gateioFlashSwapOrders, nil, &arg, &response)
}

// GetAllFlashSwapOrders retrieves list of flash swap orders filtered by the params
func (g *Gateio) GetAllFlashSwapOrders(ctx context.Context, status int, sellCurrency, buyCurrency currency.Code, reverse bool, limit, page uint64) ([]FlashSwapOrderResponse, error) {
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
	var response []FlashSwapOrderResponse
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, flashGetOrdersEPL, http.MethodGet, gateioFlashSwapOrders, params, nil, &response)
}

// GetSingleFlashSwapOrder get a single flash swap order's detail
func (g *Gateio) GetSingleFlashSwapOrder(ctx context.Context, orderID string) (*FlashSwapOrderResponse, error) {
	if orderID == "" {
		return nil, fmt.Errorf("%w, flash order order_id must not be empty", errInvalidOrderID)
	}
	var response *FlashSwapOrderResponse
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, flashGetOrderEPL, http.MethodGet, gateioFlashSwapOrders+"/"+orderID, nil, nil, &response)
}

// InitiateFlashSwapOrderReview initiate a flash swap order preview
func (g *Gateio) InitiateFlashSwapOrderReview(ctx context.Context, arg FlashSwapOrderParams) (*InitFlashSwapOrderPreviewResponse, error) {
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
	return response, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, flashOrderReviewEPL, http.MethodPost, gateioFlashSwapOrdersPreview, nil, &arg, &response)
}

// IsValidPairString returns true if the string represents a valid currency pair
func (g *Gateio) IsValidPairString(currencyPair string) bool {
	if len(currencyPair) < 3 {
		return false
	}
	pf, err := g.CurrencyPairs.GetFormat(asset.Spot, true)
	if err != nil {
		return false
	}
	if strings.Contains(currencyPair, pf.Delimiter) {
		result := strings.Split(currencyPair, pf.Delimiter)
		return len(result) >= 2
	}
	return false
}

// ********************************* Trading Fee calculation ********************************

// GetFee returns an estimate of fee based on type of transaction
func (g *Gateio) GetFee(ctx context.Context, feeBuilder *exchange.FeeBuilder) (fee float64, err error) {
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		feePairs, err := g.GetPersonalTradingFee(ctx, feeBuilder.Pair, currency.EMPTYCODE)
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
func (g *Gateio) GetUnderlyingFromCurrencyPair(p currency.Pair) (currency.Pair, error) {
	pairString := strings.ReplaceAll(p.Upper().String(), currency.DashDelimiter, currency.UnderscoreDelimiter)
	ccies := strings.Split(pairString, currency.UnderscoreDelimiter)
	if len(ccies) < 2 {
		return currency.EMPTYPAIR, fmt.Errorf("invalid currency pair %v", p)
	}
	return currency.Pair{Base: currency.NewCode(ccies[0]), Delimiter: currency.UnderscoreDelimiter, Quote: currency.NewCode(ccies[1])}, nil
}

// GetAccountDetails retrieves account details
func (g *Gateio) GetAccountDetails(ctx context.Context) (*AccountDetails, error) {
	var resp *AccountDetails
	return resp, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotAccountsEPL, http.MethodGet, "account/detail", nil, nil, &resp)
}

// GetUserTransactionRateLimitInfo retrieves user transaction rate limit info
func (g *Gateio) GetUserTransactionRateLimitInfo(ctx context.Context) ([]UserTransactionRateLimitInfo, error) {
	var resp []UserTransactionRateLimitInfo
	return resp, g.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotAccountsEPL, http.MethodGet, "account/rate_limit", nil, nil, &resp)
}
