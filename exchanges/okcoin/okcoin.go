package okcoin

import (
	"bytes"
	"context"
	"encoding/json"
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
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	okCoinRateInterval        = time.Second
	okCoinStandardRequestRate = 6
	apiPath                   = "/api/"
	okCoinAPIURL              = "https://www.okcoin.com"
	okCoinAPIVersion          = apiPath + "v5/"
	okCoinExchangeName        = "OKCOIN International"
	okCoinWebsocketURL        = "wss://real.okcoin.com:8443/ws/v5/public"
	okCoinPrivateWebsocketURL = "wss://real.okcoin.com:8443/ws/v5/private"
)

// OKCoin is the overarching type used for OKCoin's exchange API implementation
type OKCoin struct {
	exchange.Base
	// Spot and contract market error codes
	ErrorCodes map[string]error
}

var (
	errNilArgument                              = errors.New("nil argument")
	errInvalidAmount                            = errors.New("invalid amount value")
	errInvalidPrice                             = errors.New("invalid price value")
	errAddressMustNotBeEmptyString              = errors.New("address must be a non-empty string")
	errSubAccountNameRequired                   = errors.New("sub-account name is required")
	errNoValidResponseFromServer                = errors.New("no valid response")
	errTransferIDOrClientIDRequred              = errors.New("either transfer id or cliend id is required")
	errInvalidWithdrawalMethod                  = errors.New("withdrawal method must be specified")
	errInvalidTrasactionFeeValue                = errors.New("invalid transaction fee value")
	errWithdrawalIDMissing                      = errors.New("withdrawal id is missing")
	errTradeModeIsRequired                      = errors.New("trade mode is required")
	errInstrumentTypeMissing                    = errors.New("instrument type is required")
	errChannelIDRequired                        = errors.New("channel id is required")
	errBankAccountNumberIsRequired              = errors.New("bank account number is required")
	errMissingInstrumentID                      = errors.New("missing instrument id")
	errNoOrderbookData                          = errors.New("no orderbook data found")
	errOrderIDOrClientOrderIDRequired           = errors.New("order id or client order id is required")
	errSizeOrPriceRequired                      = errors.New("valid size or price has to be specified")
	errPriceRatioOrPriveSpreadRequired          = errors.New("either price ratio or price variance is required")
	errSizeLimitRequired                        = errors.New("size limit is required")
	errPriceLimitRequired                       = errors.New("price limit is required")
	errStopLossOrTakeProfitTriggerPriceRequired = errors.New("either parameter slTriggerPx or tpTriggerPx is required")
)

const (
	// endpoint types
	typeAccountSubsection = "account"
	typeTokenSubsection   = "spot"
	typeAccounts          = "account"
	typeFiat              = "fiat"
	typeOtc               = "otc"
	typeAssets            = "asset"
	typeMarket            = "market"
	typePublic            = "public"
	typeSystem            = "system"
	typeTrade             = "trade"

	// endpoints
	systemStatus = "status"
	systemTime   = "time"

	// market endpoints
	tickersPath            = "tickers"
	tickerData             = "ticker"
	getSpotOrderBooks      = "books"
	orderbookLitePath      = "books-lite"
	getSpotMarketData      = "candles"
	getSpotHistoricCandles = "history-candles"
	getTrades              = "trades"

	// ----------------------------------
	marginTradingSubsection   = "margin"
	ledger                    = "ledger"
	orders                    = "orders"
	batchOrders               = "batch_orders"
	cancelOrders              = "cancel_orders"
	cancelBatchOrders         = "cancel_batch_orders"
	pendingOrders             = "orders_pending"
	trades                    = "trades"
	instruments               = "instruments"
	getAccountDepositHistory  = "deposit/history"
	getSpotTransactionDetails = "fills"
	// Account based endpoints
	getAccountCurrencies        = "currencies"
	getAccountWalletInformation = "wallet"
	fundsTransfer               = "transfer"
	withdrawRequest             = "withdrawal"
	getWithdrawalFees           = "withdrawal/fee"
	getWithdrawalHistory        = "withdrawal/history"
	getDepositAddress           = "deposit/address"
	// Margin based endpoints
	getMarketAvailability = "availability"
	getLoan               = "borrow"
	getRepayment          = "repayment"
)

// ------------------------------------  New ------------------------------------------------------------

// GetInstruments Get market data. This endpoint provides the snapshots of market data and can be used without verifications.
// List trading pairs and get the trading limit, price, and more information of different trading pairs.
func (o *OKCoin) GetInstruments(ctx context.Context, instrumentType, instrumentID string) ([]Instrument, error) {
	params := url.Values{}
	if instrumentType != "" {
		params.Set("instType", instrumentType)
	}
	if instrumentID != "" {
		params.Set("instId", instrumentID)
	}
	var resp []Instrument
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, typePublic, common.EncodeURLValues(instruments, params), nil, &resp, false)
}

// GetSystemStatus system maintenance status,scheduled: waiting; ongoing: processing; pre_open: pre_open; completed: completed ;canceled: canceled.
// Generally, pre_open last about 10 minutes. There will be pre_open when the time of upgrade is too long.
// If this parameter is not filled, the data with status scheduled, ongoing and pre_open will be returned by default
func (o *OKCoin) GetSystemStatus(ctx context.Context, state string) (interface{}, error) {
	params := url.Values{}
	if state != "" {
		params.Set("state", state)
	}
	var resp []SystemStatus
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, typeSystem, common.EncodeURLValues(systemStatus, params), nil, &resp, false)
}

// GetSystemTime retrieve API server time.
func (o *OKCoin) GetSystemTime(ctx context.Context) (time.Time, error) {
	timestampResponse := []struct {
		Timestamp okcoinMilliSec `json:"ts"`
	}{}
	err := o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, typePublic, systemTime, nil, &timestampResponse, false)
	if err != nil {
		return time.Time{}, err
	}
	return timestampResponse[0].Timestamp.Time(), nil
}

// GetTickers retrieve the latest price snapshot, best bid/ask price, and trading volume in the last 24 hours.
func (o *OKCoin) GetTickers(ctx context.Context, instrumentType string) ([]TickerData, error) {
	params := url.Values{}
	if instrumentType != "" {
		params.Set("instType", instrumentType)
	}
	var resp []TickerData
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, typeMarket, common.EncodeURLValues(tickersPath, params), nil, &resp, false)
}

// GetTicker retrieve the latest price snapshot, best bid/ask price, and trading volume in the last 24 hours.
func (o *OKCoin) GetTicker(ctx context.Context, instrumentID string) (*TickerData, error) {
	var resp []TickerData
	err := o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, typeMarket, tickerData+"?instId="+instrumentID, nil, &resp, false)
	if err != nil {
		return nil, err
	} else if len(resp) == 0 {
		return nil, errors.New("instrument not found")
	}
	return &resp[0], nil
}

// GetOrderbook retrieve order book of the instrument.
func (o *OKCoin) GetOrderbook(ctx context.Context, instrumentID string, sideDepth int64) (*GetOrderBookResponse, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	if sideDepth > 0 {
		params.Set("sz", strconv.FormatInt(sideDepth, 10))
	}
	var resp []GetOrderBookResponse
	err := o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, typeMarket, common.EncodeURLValues(getSpotOrderBooks, params), nil, &resp, false)
	if err != nil {
		return nil, err
	} else if len(resp) == 0 {
		return nil, fmt.Errorf("%w for instrument %s", errNoOrderbookData, instrumentID)
	}
	return &resp[0], nil
}

// GetOrderbookLitebook retrieve order top 25 book of the instrument more quickly
func (o *OKCoin) GetOrderbookLitebook(ctx context.Context, instrumentID string) (*GetOrderBookResponse, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	var resp []GetOrderBookResponse
	err := o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, typeMarket, common.EncodeURLValues(orderbookLitePath, params), nil, &resp, false)
	if err != nil {
		return nil, err
	} else if len(resp) == 0 {
		return nil, fmt.Errorf("%w for instrument %s", errNoOrderbookData, instrumentID)
	}
	return &resp[0], nil
}

// GetCandlesticks retrieve the candlestick charts. This endpoint can retrieve the latest 1,440 data entries. Charts are returned in groups based on the requested bar.
func (o *OKCoin) GetCandlesticks(ctx context.Context, instrumentID string, interval kline.Interval, after, before time.Time, limit int64) ([]CandlestickData, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	var err error
	if interval != kline.Interval(0) {
		var intervalString string
		intervalString, err = intervalToString(interval, false)
		if err != nil {
			return nil, err
		}
		params.Set("bar", intervalString)
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []candlestickItemResponse
	err = o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, typeMarket, common.EncodeURLValues(getSpotMarketData, params), nil, &resp, false)
	if err != nil {
		return nil, err
	}
	return ExtractCandlesticks(resp)
}

// GetCandlestickHistory retrieve history candlestick charts from recent years.
func (o *OKCoin) GetCandlestickHistory(ctx context.Context, instrumentID string, after, before time.Time, bar kline.Interval, limit int64) ([]CandlestickData, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	var err error
	if bar != kline.Interval(0) {
		var intervalString string
		intervalString, err = intervalToString(bar, false)
		if err != nil {
			return nil, err
		}
		params.Set("bar", intervalString)
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []candlestickItemResponse
	err = o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, typeMarket, common.EncodeURLValues(getSpotHistoricCandles, params), nil, &resp, false)
	if err != nil {
		return nil, err
	}
	return ExtractCandlesticks(resp)
}

// GetTrades retrieve the recent transactions of an instrument.
func (o *OKCoin) GetTrades(ctx context.Context, instrumentID string, limit int64) ([]SpotTrade, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []SpotTrade
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, typeMarket, common.EncodeURLValues(getTrades, params), nil, &resp, false)
}

// GetTradeHistory retrieve the recent transactions of an instrument from the last 3 months with pagination.
func (o *OKCoin) GetTradeHistory(ctx context.Context, instrumentID, paginationType string, after, before time.Time, limit int64) ([]SpotTrade, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	if paginationType != "" {
		params.Set("type", paginationType)
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []SpotTrade
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, typeMarket, common.EncodeURLValues("history-trades", params), nil, &resp, false)
}

// Get24HourTradingVolume returns the 24-hour trading volume is calculated on a rolling basis, using USD as the pricing unit.
func (o *OKCoin) Get24HourTradingVolume(ctx context.Context) ([]TradingVolume, error) {
	var resp []TradingVolume
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, typeMarket, "platform-24-volume", nil, &resp, false)
}

// GetOracle retrives the crypto price of signing using Open Oracle smart contract.
func (o *OKCoin) GetOracle(ctx context.Context) (*Oracle, error) {
	var resp *Oracle
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, typeMarket, "open-oracle", nil, &resp, false)
}

// GetExchangeRate provides the average exchange rate data for 2 weeks
func (o *OKCoin) GetExchangeRate(ctx context.Context) ([]ExchangeRate, error) {
	var resp []ExchangeRate
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, typeMarket, "exchange-rate", nil, &resp, false)
}

func intervalToString(interval kline.Interval, utcOpeningPrice bool) (string, error) {
	switch interval {
	case kline.OneMin:
		return "1m", nil
	case kline.ThreeMin:
		return "3m", nil
	case kline.FiveMin:
		return "5m", nil
	case kline.FifteenMin:
		return "15m", nil
	case kline.ThirtyMin:
		return "30m", nil
	case kline.OneHour:
		return "1H", nil
	case kline.TwoHour:
		return "2H", nil
	case kline.FourHour:
		return "4H", nil
	case kline.SixHour:
		if utcOpeningPrice {
			return "6Hutc", nil
		}
		return "6H", nil
	case kline.TwelveHour:
		if utcOpeningPrice {
			return "12Hutc", nil
		}
		return "12H", nil
	case kline.OneDay:
		if utcOpeningPrice {
			return "1Dutc", nil
		}
		return "1D", nil
	case kline.TwoDay:
		if utcOpeningPrice {
			return "2Dutc", nil
		}
		return "2D", nil
	case kline.ThreeDay:
		if utcOpeningPrice {
			return "3Dutc", nil
		}
		return "3D", nil
	case kline.OneWeek:
		if utcOpeningPrice {
			return "1Wutc", nil
		}
		return "1W", nil
	case kline.OneMonth:
		if utcOpeningPrice {
			return "1Mutc", nil
		}
		return "1M", nil
	case kline.ThreeMonth:
		if utcOpeningPrice {
			return "3Mutc", nil
		}
		return "3M", nil
	default:
		return "", kline.ErrUnsupportedInterval
	}
}

// ------------ Funding endpoints --------------------------------

// GetCurrencies retrieves all list of currencies
func (o *OKCoin) GetCurrencies(ctx context.Context, ccy currency.Code) ([]CurrencyInfo, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.Upper().String())
	}
	var resp []CurrencyInfo
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, typeAssets, common.EncodeURLValues("currencies", params), nil, &resp, true)
}

// GetBalance retrieve the funding account balances of all the assets and the amount that is available or on hold.
func (o *OKCoin) GetBalance(ctx context.Context, ccy currency.Code) ([]CurrencyBalance, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	var resp []CurrencyBalance
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, typeAssets, common.EncodeURLValues("balances", params), nil, &resp, true)
}

// GetAccountAssetValuation view account asset valuation
func (o *OKCoin) GetAccountAssetValuation(ctx context.Context, ccy currency.Code) ([]AccountAssetValuation, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	var resp []AccountAssetValuation
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, typeAssets, common.EncodeURLValues("asset-valuation", params), nil, &resp, true)
}

// FundsTransfer transfer of funds between your funding account and trading account, and from the master account to sub-accounts.
func (o *OKCoin) FundsTransfer(ctx context.Context, arg *FundingTransferRequest) (*FundingTransferItem, error) {
	if arg == nil {
		return nil, errNilArgument
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Amount <= 0 {
		return nil, fmt.Errorf("%w %f", errInvalidAmount, arg.Amount)
	}
	if arg.From == "" {
		return nil, fmt.Errorf("%w, 'from' address", errAddressMustNotBeEmptyString)
	}
	if arg.To == "" {
		return nil, fmt.Errorf("%w, 'to' address", errAddressMustNotBeEmptyString)
	}
	if arg.TransferType == 1 || arg.TransferType == 2 && arg.SubAccount == "" {
		return nil, fmt.Errorf("for transfer type is 1 or 2, %w", errSubAccountNameRequired)
	}
	var resp []FundingTransferItem
	err := o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, typeAssets, "transfer", arg, &resp, true)
	if err != nil {
		return nil, err
	}
	if len(resp) == 0 {
		return nil, errNoValidResponseFromServer
	}
	return &resp[0], nil
}

// GetFundsTransferState retrieve the transfer state data of the last 2 weeks.
func (o *OKCoin) GetFundsTransferState(ctx context.Context, transferID, clientID, transferType string) ([]FundingTransferItem, error) {
	params := url.Values{}
	if transferID == "" && clientID == "" {
		return nil, errTransferIDOrClientIDRequred
	}
	if transferID != "" {
		params.Set("transId", transferID)
	}
	if clientID != "" {
		params.Set("clientId", clientID)
	}
	if transferType != "" {
		params.Set("type", transferType)
	}
	var resp []FundingTransferItem
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, typeAssets, common.EncodeURLValues("transfer-state", params), nil, &resp, true)
}

// GetAssetBilsDetail query the billing record. You can get the latest 1 month historical data.
// Bill type 1: Deposit 2: Withdrawal 13: Canceled withdrawal 20: Transfer to sub account 21: Transfer from sub account
// 22: Transfer out from sub to master account 23: Transfer in from master to sub account 37: Transfer to spot 38: Transfer from spot
func (o *OKCoin) GetAssetBilsDetail(ctx context.Context, ccy currency.Code, billType, clientSuppliedID string, before, after time.Time, limit int64) ([]AssetBillDetail, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if billType != "" {
		params.Set("type", billType)
	}
	if clientSuppliedID != "" {
		params.Set("clientId", clientSuppliedID)
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() && after.Before(before) {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []AssetBillDetail
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, typeAssets, common.EncodeURLValues("bills", params), nil, &resp, true)
}

// GetLightningDeposits retrives lightning deposit instances
func (o *OKCoin) GetLightningDeposits(ctx context.Context, ccy currency.Code, amount float64, to string) ([]LightningDepositDetail, error) {
	params := url.Values{}
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params.Set("ccy", ccy.String())
	if amount < 0.000001 || amount > 0.1 {
		return nil, fmt.Errorf("%w, deposit amount must be between 0.000001 - 0.1", errInvalidAmount)
	}
	params.Set("amt", strconv.FormatFloat(amount, 'f', -1, 64))
	if to != "" {
		params.Set("to", to)
	}
	var resp []LightningDepositDetail
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, typeAssets, common.EncodeURLValues("deposit-lightning", params), nil, &resp, true)
}

// GetCurrencyDepositAddresses retrieve the deposit addresses of currencies, including previously-used addresses.
func (o *OKCoin) GetCurrencyDepositAddresses(ctx context.Context, ccy currency.Code) ([]DepositAddress, error) {
	params := url.Values{}
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params.Set("ccy", ccy.String())
	var resp []DepositAddress
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, typeAssets, common.EncodeURLValues("deposit-address", params), nil, &resp, true)
}

// GetDepositHistory retrieve the deposit records according to the currency, deposit status, and time range in reverse chronological order. The 100 most recent records are returned by default.
func (o *OKCoin) GetDepositHistory(ctx context.Context, ccy currency.Code, depositID, transactionID, depositType, state string, after, before time.Time, limit int64) ([]DepositHistoryItem, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if depositID != "" {
		params.Set("depId", depositID)
	}
	if transactionID != "" {
		params.Set("txId", transactionID)
	}
	if depositType != "" {
		params.Set("type", depositType)
	}
	if state != "" {
		params.Set("state", state)
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() && after.Before(before) {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []DepositHistoryItem
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, typeAssets, common.EncodeURLValues("deposit-history", params), nil, &resp, true)
}

// Withdrawal apply withdrawal of tokens. Sub-account does not support withdrawal.
func (o *OKCoin) Withdrawal(ctx context.Context, arg *WithdrawalRequest) ([]WithdrawalResponse, error) {
	if arg == nil {
		return nil, errNilArgument
	}
	if arg.Ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Amount <= 0 {
		return nil, fmt.Errorf("%w %f", errInvalidAmount, arg.Amount)
	}
	if arg.WithdrawalMethod == "" {
		return nil, errInvalidWithdrawalMethod
	}
	if arg.ToAddress == "" {
		return nil, fmt.Errorf("%w, 'toAddr' address", errAddressMustNotBeEmptyString)
	}
	if arg.TransactionFee <= 0 {
		return nil, fmt.Errorf("%w, transaction fee: %f", errInvalidTrasactionFeeValue, arg.TransactionFee)
	}
	var resp []WithdrawalResponse
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, typeAssets, "withdrawal", arg, &resp, true)
}

// SubmitLightningWithdrawals the maximum withdrawal amount is 0.1 BTC per request, and 1 BTC in 24 hours.
// The minimum withdrawal amount is approximately 0.000001 BTC. Sub-account does not support withdrawal.
func (o *OKCoin) SubmitLightningWithdrawals(ctx context.Context, arg *LightningWithdrawalsRequest) ([]LightningWithdrawals, error) {
	if arg == nil {
		return nil, errNilArgument
	}
	if arg.Ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Invoice == "" {
		return nil, errors.New("missing invoice text")
	}
	var resp []LightningWithdrawals
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, typeAssets, "withdrawal-lightning", arg, resp, true)
}

// CancelWithdrawal cancel normal withdrawal requests, but you cannot cancel withdrawal requests on Lightning.
func (o *OKCoin) CancelWithdrawal(ctx context.Context, arg *WithdrawalCancelation) (*WithdrawalCancelation, error) {
	var resp []WithdrawalCancelation
	if arg.WithdrawalID == "" {
		return nil, errWithdrawalIDMissing
	}
	err := o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, typeAssets, "cancel-withdrawal", arg, &resp, true)
	if err != nil {
		return nil, err
	}
	if len(resp) == 0 {
		return nil, errNoValidResponseFromServer
	}
	return &resp[0], nil
}

// GetWithdrawalHistory retrieve the withdrawal records according to the currency, withdrawal status, and time range in reverse chronological order. The 100 most recent records are returned by default.
func (o *OKCoin) GetWithdrawalHistory(ctx context.Context, ccy currency.Code, withdrawalID, clientID, transactionID, withdrawalType, state string, after, before time.Time, limit int64) ([]WithdrawalOrderItem, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if withdrawalID != "" {
		params.Set("wdId", withdrawalID)
	}
	if clientID != "" {
		params.Set("clientId", clientID)
	}
	if transactionID != "" {
		params.Set("txId", transactionID)
	}
	if withdrawalType != "" {
		params.Set("type", withdrawalType)
	}
	if state != "" {
		params.Set("state", state)
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	var resp []WithdrawalOrderItem
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, typeAssets, common.EncodeURLValues("withdrawal-history", params), nil, &resp, true)
}

// ------------------------ Account Endpoints --------------------

// GetAccountBalance retrieve a list of assets (with non-zero balance), remaining balance, and available amount in the trading account.
func (o *OKCoin) GetAccountBalance(ctx context.Context, currencies ...currency.Code) ([]AccountBalanceInformation, error) {
	params := url.Values{}
	if len(currencies) > 0 {
		currencyString := ""
		found := false
		for x := range currencies {
			if found {
				currencyString += ","
			}
			if !currencies[x].IsEmpty() {
				currencyString += currencies[x].String()
				found = true
			}
		}
		if found {
			params.Set("ccy", currencyString)
		}
	}
	var resp []AccountBalanceInformation
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, typeAccounts, common.EncodeURLValues("balance", params), nil, &resp, true)
}

// GetBillsDetails retrieve the bills of the account. The bill refers to all transaction records that result in changing the balance of an account. Pagination is supported, and the response is sorted with the most recent first.
// For the last 7 days.
func (o *OKCoin) GetBillsDetails(ctx context.Context, ccy currency.Code, instrumentType, billType, billSubType, afterBillID, beforeBillID string, begin, end time.Time, limit int64) ([]BillsDetail, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if instrumentType != "" {
		params.Set("instType", instrumentType)
	}
	if billType != "" {
		params.Set("type", billType)
	}
	if billSubType != "" {
		params.Set("subType", billSubType)
	}
	if afterBillID != "" {
		params.Set("after", afterBillID)
	}
	if beforeBillID != "" {
		params.Set("before", beforeBillID)
	}
	if !begin.IsZero() {
		params.Set("before", strconv.FormatInt(begin.UnixMilli(), 10))
	}
	if !end.IsZero() {
		params.Set("end", strconv.FormatInt(end.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []BillsDetail
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, typeAccounts, common.EncodeURLValues("bills", params), nil, &resp, true)
}

// GetBillsDetailsFor3Moonths retrieve the bills of the account. The bill refers to all transaction records that result in changing the balance of an account. Pagination is supported, and the response is sorted with the most recent first.
// For the last 3 months.
func (o *OKCoin) GetBillsDetailsFor3Months(ctx context.Context, ccy currency.Code, instrumentType, billType, billSubType, afterBillID, beforeBillID string, begin, end time.Time, limit int64) ([]BillsDetail, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if instrumentType != "" {
		params.Set("instType", instrumentType)
	}
	if billType != "" {
		params.Set("type", billType)
	}
	if billSubType != "" {
		params.Set("subType", billSubType)
	}
	if afterBillID != "" {
		params.Set("after", afterBillID)
	}
	if beforeBillID != "" {
		params.Set("before", beforeBillID)
	}
	if !begin.IsZero() {
		params.Set("before", strconv.FormatInt(begin.UnixMilli(), 10))
	}
	if !end.IsZero() {
		params.Set("end", strconv.FormatInt(end.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []BillsDetail
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, typeAccounts, common.EncodeURLValues("bills-archive", params), nil, &resp, true)
}

// GetAccountConfigurations retrives current account configuration informations.
func (o *OKCoin) GetAccountConfigurations(ctx context.Context) ([]AccountConfiguration, error) {
	var resp []AccountConfiguration
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, typeAccounts, "config", nil, &resp, true)
}

// GetMaximumBuySellOrOpenAmount retrives maximum buy, sell, or open amount information.
// Single instrument or multiple instruments (no more than 5) separated with comma, e.g. BTC-USD
// Trade mode 'cash'
// Price When the price is not specified, it will be calculated according to the last traded price.
// optional parameter
func (o *OKCoin) GetMaximumBuySellOrOpenAmount(ctx context.Context, instrumentID, tradeMode string, price float64) ([]MaxBuySellResp, error) {
	params := url.Values{}
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	params.Set("instId", instrumentID)
	if tradeMode == "" {
		return nil, errTradeModeIsRequired
	}
	params.Set("tdMode", tradeMode)
	if price != 0 {
		params.Set("price", strconv.FormatFloat(price, 'f', -1, 64))
	}
	var resp []MaxBuySellResp
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, typeAccounts, common.EncodeURLValues("max-size", params), nil, &resp, true)
}

// GetMaximumAvailableTradableAmount retrives maximum available tradable amount.
// Single instrument or multiple instruments (no more than 5) separated with comma, e.g. BTC-USDT,ETH-USDT
// Trade mode 'cash'
func (o *OKCoin) GetMaximumAvailableTradableAmount(ctx context.Context, tradeMode string, instrumentIDs ...string) ([]AvailableTradableAmount, error) {
	params := url.Values{}
	if len(instrumentIDs) == 0 {
		return nil, errMissingInstrumentID
	}
	params.Set("instId", strings.Join(instrumentIDs, ","))
	if tradeMode == "" {
		return nil, errTradeModeIsRequired
	}
	params.Set("tdMode", tradeMode)
	var resp []AvailableTradableAmount
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, typeAccounts, common.EncodeURLValues("max-avail-size", params), nil, &resp, true)
}

// GetFeeRates retrives instrument trading fee information.
func (o *OKCoin) GetFeeRates(ctx context.Context, instrumentType, instrumentID string) ([]FeeRate, error) {
	params := url.Values{}
	if instrumentType == "" {
		return nil, errInstrumentTypeMissing
	}
	params.Set("instType", instrumentType)
	if instrumentID != "" {
		params.Set("instId", instrumentID)
	}
	var resp []FeeRate
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, typeAccounts, common.EncodeURLValues("trade-fee", params), nil, &resp, true)
}

// GetMaximumWithdrawals retrieve the maximum transferable amount from trading account to funding account. If no currency is specified, the transferable amount of all owned currencies will be returned.
func (o *OKCoin) GetMaximumWithdrawals(ctx context.Context, ccy currency.Code) ([]MaximumWithdrawal, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	var resp []MaximumWithdrawal
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, typeAccounts, common.EncodeURLValues("max-withdrawal", params), nil, &resp, true)
}

// ------------------------------------ OTC-Desk RFQ --------------------------------

// GetAvailableRFQPairs retrives a list of RFQ instruments.
func (o *OKCoin) GetAvailableRFQPairs(ctx context.Context) ([]AvailableRFQPair, error) {
	var resp []AvailableRFQPair
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, typeOtc, "rfq/instruments", nil, &resp, true)
}

// RequestQuote query current market quotation information
func (o *OKCoin) RequestQuote(ctx context.Context, arg *QuoteRequestArg) ([]RFQQuoteResponse, error) {
	if arg == nil {
		return nil, errNilArgument
	}
	if arg.BaseCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w, base currency", currency.ErrCurrencyCodeEmpty)
	}
	if arg.QuoteCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w, quote currency", currency.ErrCurrencyCodeEmpty)
	}
	if arg.Side == "" {
		return nil, fmt.Errorf("%w, empty order side", order.ErrSideIsInvalid)
	}
	if arg.RfqSize <= 0 {
		return nil, fmt.Errorf("%w, RFQ size must be greater than 0", errInvalidAmount)
	}
	if arg.RfqSzCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w, rfqSzCurrency currency", currency.ErrCurrencyCodeEmpty)
	}
	var resp []RFQQuoteResponse
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, typeOtc, "rfq/quote", arg, &resp, true)
}

// PlaceRFQOrder submit RFQ order
func (o *OKCoin) PlaceRFQOrder(ctx context.Context, arg *PlaceRFQOrderRequest) ([]RFQOrderResponse, error) {
	if arg == nil {
		return nil, errNilArgument
	}
	if arg.ClientDefinedTradeRequestID == "" {
		return nil, errors.New("client supplied request ID is required")
	}
	if arg.QuoteID == "" {
		return nil, errors.New("quoteid is required")
	}
	if arg.BaseCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w, base currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if arg.QuoteCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w, quote currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if arg.Side == "" {
		return nil, fmt.Errorf("%w, order side is required", order.ErrSideIsInvalid)
	}
	if arg.Size <= 0 {
		return nil, fmt.Errorf("%w, size can not be less than or equal to zero", errInvalidAmount)
	}
	if arg.SizeCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w, token is required", currency.ErrCurrencyCodeEmpty)
	}
	var resp []RFQOrderResponse
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, typeAccounts, "", arg, &resp, true)
}

// GetRFQOrderDetails retrives an RFQ order details.
func (o *OKCoin) GetRFQOrderDetails(ctx context.Context, clientDefinedID, tradeOrderID string) ([]RFQOrderDetail, error) {
	params := url.Values{}
	if clientDefinedID != "" {
		params.Set("clTReqId", clientDefinedID)
	}
	if tradeOrderID != "" {
		params.Set("tradeId", tradeOrderID)
	}
	var resp []RFQOrderDetail
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, typeOtc, common.EncodeURLValues("rfq/trade", params), nil, &resp, true)
}

// GetRFQOrderHistory retrives an RFQ order history
func (o *OKCoin) GetRFQOrderHistory(ctx context.Context, begin, end time.Time, pageSize, pageIndex int64) ([]RFQOrderHistoryItem, error) {
	params := url.Values{}
	if !begin.IsZero() {
		params.Set("begin", strconv.FormatInt(begin.UnixMilli(), 10))
	}
	if !end.IsZero() {
		params.Set("end", strconv.FormatInt(end.UnixMilli(), 10))
	}
	if pageSize > 0 {
		params.Set("pageSz", strconv.FormatInt(pageSize, 10))
	}
	if pageIndex > 0 {
		params.Set("pageIdx", strconv.FormatInt(pageIndex, 10))
	}
	var resp []RFQOrderHistoryItem
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, typeOtc, common.EncodeURLValues("rfq/history", params), nil, &resp, true)
}

// ---------- Fiat ----------------------------------------------------------------

// Deposit posts a fiat deposit to an account
func (o *OKCoin) Deposit(ctx context.Context, arg *FiatDepositRequestArg) ([]FiatDepositResponse, error) {
	if arg == nil {
		return nil, errNilArgument
	}
	if arg.ChannelID == "" {
		return nil, errChannelIDRequired
	}
	if arg.Amount <= 0 {
		return nil, fmt.Errorf("%w, amount %f", errInvalidAmount, arg.Amount)
	}
	if arg.BankAccountNumber == "" {
		return nil, errBankAccountNumberIsRequired
	}
	var resp []FiatDepositResponse
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, typeFiat, "deposit", arg, &resp, true)
}

// CancelFiatDeposit cancels pending deposit requests.
func (o *OKCoin) CancelFiatDeposit(ctx context.Context, depositID string) (*CancelDepositAddressResp, error) {
	if depositID == "" {
		return nil, errors.New("deposit address required")
	}
	var resp []CancelDepositAddressResp
	err := o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, typeFiat, "cancel-deposit", map[string]string{"depId": depositID}, &resp, true)
	if err != nil {
		return nil, err
	}
	if len(resp) == 0 {
		return nil, errNoValidResponseFromServer
	}
	return &resp[0], nil
}

// GetFiatDepositHistory deposit history query requests can be filtered by the different elements, such as channels, deposit status, and currencies.
// Paging is also available during query and is stored in reverse order based on the transaction time, with the latest one at the top.
func (o *OKCoin) GetFiatDepositHistory(ctx context.Context, ccy currency.Code, channelID, depositState, depositID string, after, before time.Time, limit int64) ([]DepositHistoryResponse, error) {
	params := url.Values{}
	if channelID != "" {
		params.Set("chanId", channelID)
	}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if depositState != "" {
		params.Set("state", depositState)
	}
	if depositID != "" {
		params.Set("depId", depositID)
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []DepositHistoryResponse
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, typeFiat, common.EncodeURLValues("deposit-history", params), nil, &resp, true)
}

// FiatWithdrawal submit fiat withdrawal operations.
func (o *OKCoin) FiatWithdrawal(ctx context.Context, arg *FiatWithdrawalParam) (*FiatWithdrawalResponse, error) {
	if arg == nil {
		return nil, errNilArgument
	}
	if arg.ChannelID == "" {
		return nil, errChannelIDRequired
	}
	if arg.Amount <= 0 {
		return nil, fmt.Errorf("%w, amount must be greater than 0", errInvalidAmount)
	}
	if arg.BankAcctNumber == "" {
		return nil, errBankAccountNumberIsRequired
	}
	var resp []FiatWithdrawalResponse
	err := o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, typeFiat, "withdrawal", arg, &resp, true)
	if err != nil {
		return nil, err
	}
	if len(resp) == 0 {
		return nil, errNoValidResponseFromServer
	}
	return &resp[0], nil
}

// FiatCancelWithdrawal canceles fiat withdrawal request
func (o *OKCoin) FiatCancelWithdrawal(ctx context.Context, withdrawalID string) (string, error) {
	if withdrawalID == "" {
		return "", errWithdrawalIDMissing
	}
	var resp []struct {
		WithdrawalID string `json:"wdId"`
	}
	err := o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, typeFiat, "cancel-withdrawal", &map[string]string{"wdId": withdrawalID}, &resp, true)
	if err != nil {
		return "", err
	}
	if len(resp) == 0 {
		return "", errNoValidResponseFromServer
	}
	return resp[0].WithdrawalID, nil
}

// GetFiatWithdrawalHistory retrives a fiat withdrawal orders list
// Channel ID used in the transaction.  9:PrimeX; 28:PrimeX US; 21:PrimeX Europe; 3:Silvergate SEN; 27:Silvergate SEN HK
// Withdrawal state. -2:User canceled the order；-1:Withdrawal attempt has failed; 0:Withdrawal request submitted; 1:Withdrawal request is pending; 2:Withdrawal has been credited
func (o *OKCoin) GetFiatWithdrawalHistory(ctx context.Context, ccy currency.Code, channelID, withdrawalState, withdrawalID string, after, before time.Time, limit int64) ([]FiatWithdrawalHistoryItem, error) {
	params := url.Values{}
	if channelID != "" {
		params.Set("chanId", channelID)
	}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if withdrawalState != "" {
		params.Set("state", withdrawalState)
	}
	if withdrawalID != "" {
		params.Set("depId", withdrawalID)
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []FiatWithdrawalHistoryItem
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, typeFiat, common.EncodeURLValues("withdrawal-history", params), nil, &resp, true)
}

// GetChannelInfo retrives channel detailed information given channel id.
func (o *OKCoin) GetChannelInfo(ctx context.Context, channelID string) ([]ChannelInfo, error) {
	params := url.Values{}
	if channelID != "" {
		params.Set("chanId", channelID)
	}
	var resp []ChannelInfo
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, typeFiat, common.EncodeURLValues("channel", params), nil, &resp, true)
}

// --------------------- Trade endpoints ---------------------------

func (a *PlaceTradeOrderParam) validateTradeOrderParameter() error {
	if a == nil {
		return errNilArgument
	}
	if a.InstrumentID.IsEmpty() {
		return fmt.Errorf("%w, instrument id is required", currency.ErrCurrencyPairEmpty)
	}
	if a.TradeMode == "" {
		return errTradeModeIsRequired
	}
	if a.Side == "" {
		return fmt.Errorf("%w, empty order side", order.ErrSideIsInvalid)
	}
	if a.OrderType == "" {
		return fmt.Errorf("%w, empty order type", order.ErrTypeIsInvalid)
	}
	if a.Size <= 0 {
		return fmt.Errorf("%w, size: %f", errInvalidAmount, a.Size)
	}
	return nil
}

// PlaceOrder to place a trade order.
func (o *OKCoin) PlaceOrder(ctx context.Context, arg *PlaceTradeOrderParam) (*TradeOrderResponse, error) {
	err := arg.validateTradeOrderParameter()
	if err != nil {
		return nil, err
	}
	var resp []TradeOrderResponse
	err = o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, typeTrade, "order", &arg, &resp, true)
	if err != nil {
		return nil, err
	}
	if len(resp) == 0 {
		return nil, errNoValidResponseFromServer
	}
	if resp[0].SCode != "0" {
		return nil, fmt.Errorf("code: %s msg: %s", resp[0].SCode, resp[0].SMsg)
	}
	return &resp[0], nil
}

// PlaceMultipleOrder place orders in batches. Maximum 20 orders can be placed per request. Request parameters should be passed in the form of an array.
func (o *OKCoin) PlaceMultipleOrder(ctx context.Context, args []PlaceTradeOrderParam) ([]TradeOrderResponse, error) {
	var err error
	if len(args) == 0 {
		return nil, fmt.Errorf("%w, 0 length place order requests", errNilArgument)
	}
	for x := range args {
		err = args[x].validateTradeOrderParameter()
		if err != nil {
			return nil, err
		}
	}
	var resp []TradeOrderResponse
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, typeTrade, "batch-orders", &args, &resp, true)
}

// CancelTradeOrder cancels a single trade order
func (o *OKCoin) CancelTradeOrder(ctx context.Context, arg *CancelTradeOrderRequest) (*TradeOrderResponse, error) {
	if arg == nil {
		return nil, errNilArgument
	}
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	if arg.OrderID == "" && arg.ClientOrderID == "" {
		return nil, errOrderIDOrClientOrderIDRequired
	}
	var resp []TradeOrderResponse
	err := o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, typeTrade, "cancel-order", &arg, &resp, true)
	if err != nil {
		return nil, err
	}
	if len(resp) == 0 {
		return nil, errNoValidResponseFromServer
	}
	if resp[0].SCode != "0" {
		return nil, fmt.Errorf("code: %s msg: %s", resp[0].SCode, resp[0].SMsg)
	}
	return &resp[0], nil
}

func (arg *CancelTradeOrderRequest) validate() error {
	if arg == nil {
		return errNilArgument
	}
	if arg.InstrumentID == "" {
		return errMissingInstrumentID
	}
	if arg.OrderID == "" && arg.ClientOrderID == "" {
		return errOrderIDOrClientOrderIDRequired
	}
	return nil
}

// CancelMultipleOrders cancel incomplete orders in batches. Maximum 20 orders can be canceled per request.
// Request parameters should be passed in the form of an array.
func (o *OKCoin) CancelMultipleOrders(ctx context.Context, args []CancelTradeOrderRequest) ([]TradeOrderResponse, error) {
	var err error
	if len(args) == 0 {
		return nil, fmt.Errorf("%w, 0 length place order requests", errNilArgument)
	}
	for x := range args {
		err = args[x].validate()
		if err != nil {
			return nil, err
		}
	}
	var resp []TradeOrderResponse
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, typeTrade, "cancel-batch-orders", args, &resp, true)
}

func (a *AmendTradeOrderRequestParam) validate() error {
	if a == nil {
		return errNilArgument
	}
	if a.InstrumentID == "" {
		return errMissingInstrumentID
	}
	if a.OrderID == "" && a.ClientOrderID == "" {
		return errOrderIDOrClientOrderIDRequired
	}
	if a.NewSize <= 0 && a.NewPrice <= 0 {
		return errSizeOrPriceRequired
	}
	return nil
}

// AmendOrder amends an incomplete order.
func (o *OKCoin) AmendOrder(ctx context.Context, arg *AmendTradeOrderRequestParam) (*AmendTradeOrderResponse, error) {
	err := arg.validate()
	if err != nil {
		return nil, err
	}
	var resp []AmendTradeOrderResponse
	err = o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, typeTrade, "amend-order", &arg, &resp, true)
	if err != nil {
		return nil, err
	}
	if len(resp) == 0 {
		return nil, errNoValidResponseFromServer
	}
	if resp[0].StatusCode != "0" {
		return nil, fmt.Errorf("code: %s msg: %s", resp[0].StatusCode, resp[0].StatusMessage)
	}
	return &resp[0], nil
}

// AmendMultipleOrder amends multple trade orders.
func (o *OKCoin) AmendMultipleOrder(ctx context.Context, args []AmendTradeOrderRequestParam) ([]AmendTradeOrderResponse, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("%w, please provide at least one trade order amendment request", errNilArgument)
	}
	for x := range args {
		err := args[x].validate()
		if err != nil {
			return nil, err
		}
	}
	var resp []AmendTradeOrderResponse
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, typeTrade, "amend-batch-orders", &args, &resp, true)
}

// GetPersonalOrderDetail retrives an order detail
func (o *OKCoin) GetPersonalOrderDetail(ctx context.Context, instrumentID, orderID, clientOrderID string) (*TradeOrder, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	if orderID == "" && clientOrderID == "" {
		return nil, errOrderIDOrClientOrderIDRequired
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	if orderID != "" {
		params.Set("ordId", orderID)
	}
	if clientOrderID != "" {
		params.Set("clOrdId", clientOrderID)
	}
	var resp []TradeOrder
	err := o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, typeTrade, common.EncodeURLValues("order", params), nil, &resp, true)
	if err != nil {
		return nil, err
	}
	if len(resp) == 0 {
		return nil, err
	}
	return &resp[0], nil
}

func tradeOrderParamsFill(instrumentType, instrumentID, orderType, state string, after, before time.Time, limit int64) url.Values {
	params := url.Values{}
	if instrumentType != "" {
		params.Set("instType", instrumentType)
	}
	if instrumentID != "" {
		params.Set("instId", instrumentID)
	}
	if orderType != "" {
		params.Set("ordType", orderType)
	}
	if state != "" {
		params.Set("state", state)
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	return params
}

// GetPersonalOrderList retrieve all incomplete orders under the current account.
func (o *OKCoin) GetPersonalOrderList(ctx context.Context, instrumentType, instrumentID, orderType, state string, after, before time.Time, limit int64) ([]TradeOrder, error) {
	params := tradeOrderParamsFill(instrumentType, instrumentID, orderType, state, after, before, limit)
	var resp []TradeOrder
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, typeTrade, common.EncodeURLValues("orders-pending", params), nil, &resp, true)
}

// GetOrderHistory7Days retrieve the completed order data for the last 7 days, and the incomplete orders that have been canceled are only reserved for 2 hours.
func (o *OKCoin) GetOrderHistory7Days(ctx context.Context, instrumentType, instrumentID, orderType, state string, after, before time.Time, limit int64) ([]TradeOrder, error) {
	params := tradeOrderParamsFill(instrumentType, instrumentID, orderType, state, after, before, limit)
	var resp []TradeOrder
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, typeTrade, common.EncodeURLValues("orders-history", params), nil, &resp, true)
}

// GetOrderHistory3Months retrieve the completed order data of the last 3 months.
func (o *OKCoin) GetOrderHistory3MonthsDays(ctx context.Context, instrumentType, instrumentID, orderType, state string, after, before time.Time, limit int64) ([]TradeOrder, error) {
	params := tradeOrderParamsFill(instrumentType, instrumentID, orderType, state, after, before, limit)
	var resp []TradeOrder
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, typeTrade, common.EncodeURLValues("orders-history-archive", params), nil, &resp, true)
}

func transactionFillParams(instrumentType, instrumentID, orderID, afterBillID, beforeBillID string, begin, end time.Time, limit int64) url.Values {
	params := url.Values{}
	if instrumentType != "" {
		params.Set("instType", instrumentType)
	}
	if instrumentID != "" {
		params.Set("instId", instrumentID)
	}
	if orderID != "" {
		params.Set("ordId", orderID)
	}
	if afterBillID != "" {
		params.Set("after", afterBillID)
	}
	if beforeBillID != "" {
		params.Set("before", beforeBillID)
	}
	if !begin.IsZero() {
		params.Set("begin", strconv.FormatInt(begin.UnixMilli(), 10))
	}
	if !end.IsZero() {
		params.Set("end", strconv.FormatInt(end.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	return params
}

// GetRecentTransactionDetail retrieve recently-filled transaction details in the last 3 day.
func (o *OKCoin) GetRecentTransactionDetail(ctx context.Context, instrumentType, instrumentID, orderID, afterBillID, beforeBillID string, begin, end time.Time, limit int64) ([]TransactionFillItem, error) {
	params := transactionFillParams(instrumentType, instrumentID, orderID, afterBillID, beforeBillID, begin, end, limit)
	var resp []TransactionFillItem
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, typeTrade, common.EncodeURLValues("fills", params), nil, &resp, true)
}

// GetTransactionDetails3Months retrives recently filled transaction detail in the last 3-months
func (o *OKCoin) GetTransactionDetails3Months(ctx context.Context, instrumentType, instrumentID, orderID, beforeBillID, afterBillID string, begin, end time.Time, limit int64) ([]TransactionFillItem, error) {
	params := transactionFillParams(instrumentType, instrumentID, orderID, afterBillID, beforeBillID, begin, end, limit)
	var resp []TransactionFillItem
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, typeTrade, common.EncodeURLValues("fills-history", params), nil, &resp, true)
}

func (arg *AlgoOrderRequestParam) validateAlgoOrder() error {
	if arg == nil {
		return errNilArgument
	}
	if arg.InstrumentID == "" {
		return errMissingInstrumentID
	}
	if arg.TradeMode == "" {
		return errTradeModeIsRequired
	}
	if arg.Side == "" {
		return fmt.Errorf("%w, empty order side", order.ErrSideIsInvalid)
	}
	// conditional: One-way stop order, 'oco': One-cancels-the-other order, 'trigger': Trigger order
	// 'move_order_stop': Trailing order 'iceberg': Iceberg order 'twap': TWAP order
	if arg.OrderType == "" {
		return fmt.Errorf("%w, empty order type", order.ErrTypeIsInvalid)
	}
	if arg.Size <= 0 {
		return fmt.Errorf("%w, please specify a valid size, size >0", errInvalidAmount)
	}
	switch arg.OrderType {
	case "conditional":
		//  One-way stop order
		// When placing net stop order (ordType=conditional) and both take-profit and stop-loss parameters are sent,
		// only stop-loss logic will be performed and take-profit logic will be ignored.
		if arg.StopLossTriggerPrice <= 0 && arg.TpTriggerPrice <= 0 {
			return errStopLossOrTakeProfitTriggerPriceRequired
		}
	case "oco":
		//  One-cancels-the-other order
	case "trigger":
		//  Trigger order
		if arg.TriggerPrice <= 0 {
			return fmt.Errorf("%w, trigger price is required for order type %s", errInvalidPrice, arg.OrderType)
		}
		if arg.OrderPrice <= 0 {
			return fmt.Errorf("%w, order price is required for order type %s", errInvalidPrice, arg.OrderType)
		}
	case "move_order_stop":
		//  Trailing order
		if arg.CallbackRatio <= 0 && arg.CallbackSpread == "" {
			return errors.New("either Callback ration or callback spread is required")
		}
	case "iceberg":
		//  Iceberg order
		if arg.PriceRatio <= 0 && arg.PriceSpread <= 0 {
			return errPriceRatioOrPriveSpreadRequired
		}
		if arg.SizeLimit <= 0 {
			return fmt.Errorf("%w, order type %s", errSizeLimitRequired, arg.OrderType)
		}
		if arg.PriceLimit <= 0 {
			return fmt.Errorf("%w, order type %s", errPriceLimitRequired, arg.OrderType)
		}
	case "twap":
		//  TWAP order
		if arg.PriceRatio <= 0 && arg.PriceSpread <= 0 {
			return errPriceRatioOrPriveSpreadRequired
		}
		if arg.SizeLimit <= 0 {
			return fmt.Errorf("%w, order type %s", errSizeLimitRequired, arg.OrderType)
		}
		if arg.PriceLimit <= 0 {
			return fmt.Errorf("%w, order type %s", errPriceLimitRequired, arg.OrderType)
		}
		if arg.TimeInterval == "" {
			return errors.New("time interval information is required")
		}
	}
	return nil
}

// PlaceAlgoOrder places an algo order.
// The algo order includes trigger order, oco order, conditional order,iceberg order, twap order and trailing order.
func (o *OKCoin) PlaceAlgoOrder(ctx context.Context, arg *AlgoOrderRequestParam) (*AlgoOrderResponse, error) {
	err := arg.validateAlgoOrder()
	if err != nil {
		return nil, err
	}
	var resp []AlgoOrderResponse
	err = o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, typeTrade, "order-algo", &arg, &resp, true)
	if err != nil {
		return nil, err
	}
	if len(resp) == 0 {
		return nil, errNoValidResponseFromServer
	}
	if resp[0].StatusCode != "0" {
		return nil, fmt.Errorf("code: %s msg: %s", resp[0].StatusCode, resp[0].StatusMsg)
	}
	return &resp[0], nil
}

// ------------------------------------  Old ------------------------------------------------------------

// GetAccountCurrencies returns a list of tradable spot instruments and their properties
func (o *OKCoin) GetAccountCurrencies(ctx context.Context) ([]GetAccountCurrenciesResponse, error) {
	//	var respData []struct {
	//		Name          string `json:"name"`
	//		Currency      string `json:"currency"`
	//		Chain         string `json:"chain"`
	//		CanInternal   int64  `json:"can_internal,string"`
	//		CanWithdraw   int64  `json:"can_withdraw,string"`
	//		CanDeposit    int64  `json:"can_deposit,string"`
	//		MinWithdrawal string `json:"min_withdrawal"`
	//	}
	//
	// err := o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, accountSubsection, getAccountCurrencies, nil, &respData, true)
	//
	//	if err != nil {
	//		return nil, err
	//	}
	//
	// resp := make([]GetAccountCurrenciesResponse, len(respData))
	//
	//	for i := range respData {
	//		var mw float64
	//		if respData[i].MinWithdrawal != "" {
	//			mw, err = strconv.ParseFloat(respData[i].MinWithdrawal, 64)
	//			if err != nil {
	//				return nil, err
	//			}
	//		}
	//		resp[i] = GetAccountCurrenciesResponse{
	//			Name:          respData[i].Name,
	//			Currency:      respData[i].Currency,
	//			Chain:         respData[i].Chain,
	//			CanInternal:   respData[i].CanInternal == 1,
	//			CanWithdraw:   respData[i].CanWithdraw == 1,
	//			CanDeposit:    respData[i].CanDeposit == 1,
	//			MinWithdrawal: mw,
	//		}
	//	}
	//
	// return resp, nil
	return nil, nil
}

// GetAccountWalletInformation returns a list of wallets and their properties
func (o *OKCoin) GetAccountWalletInformation(ctx context.Context, currency string) ([]WalletInformationResponse, error) {
	// requestURL := getAccountWalletInformation
	//
	//	if currency != "" {
	//		requestURL += "/" + currency
	//	}
	//
	// var resp []WalletInformationResponse
	// return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, accountSubsection, requestURL, nil, &resp, true)
	return nil, nil
}

// TransferAccountFunds  the transfer of funds between wallet, trading accounts, main account and subaccounts
func (o *OKCoin) TransferAccountFunds(ctx context.Context, request *TransferAccountFundsRequest) (*TransferAccountFundsResponse, error) {
	// var resp *TransferAccountFundsResponse
	// return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, accountSubsection, fundsTransfer, request, &resp, true)
	return nil, nil
}

// AccountWithdraw withdrawal of tokens to OKCoin International or other addresses.
func (o *OKCoin) AccountWithdraw(ctx context.Context, request *AccountWithdrawRequest) (*AccountWithdrawResponse, error) {
	// 	var resp *AccountWithdrawResponse
	// 	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, accountSubsection, withdrawRequest, request, &resp, true)
	return nil, nil
}

// GetAccountWithdrawalFee retrieves the information about the recommended network transaction fee for withdrawals to digital asset addresses. The higher the fees are, the sooner the confirmations you will get.
func (o *OKCoin) GetAccountWithdrawalFee(ctx context.Context, currency string) ([]GetAccountWithdrawalFeeResponse, error) {
	// 	requestURL := getWithdrawalFees
	// 	if currency != "" {
	// 		requestURL += "?currency=" + currency
	// 	}

	// 	var resp []GetAccountWithdrawalFeeResponse
	// 	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, accountSubsection, requestURL, nil, &resp, true)
	// }

	// // GetAccountWithdrawalHistory retrieves all recent withdrawal records.
	// func (o *OKCoin) GetAccountWithdrawalHistory(ctx context.Context, currency string) ([]WithdrawalHistoryResponse, error) {
	// 	requestURL := getWithdrawalHistory
	// 	if currency != "" {
	// 		requestURL += "/" + currency
	// 	}
	// 	var resp []WithdrawalHistoryResponse
	// 	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, accountSubsection, requestURL, nil, &resp, true)
	return nil, nil
}

// GetAccountBillDetails retrieves the bill details of the wallet. All the information will be paged and sorted in reverse chronological order,
// which means the latest will be at the top. Please refer to the pagination section for additional records after the first page.
// 3 months recent records will be returned at maximum
func (o *OKCoin) GetAccountBillDetails(ctx context.Context, request *GetAccountBillDetailsRequest) ([]GetAccountBillDetailsResponse, error) {
	// 	encodedRequest, err := encodeRequest(request)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	requestURL := ledger + encodedRequest
	// 	var resp []GetAccountBillDetailsResponse
	// 	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, accountSubsection, requestURL, nil, &resp, true)
	return nil, nil
}

// GetAccountDepositAddressForCurrency retrieves the deposit addresses of different tokens, including previously used addresses.
func (o *OKCoin) GetAccountDepositAddressForCurrency(ctx context.Context, currency string) ([]GetDepositAddressResponse, error) {
	// 	urlValues := url.Values{}
	// 	urlValues.Set("currency", currency)
	// 	requestURL := getDepositAddress + "?" + urlValues.Encode()
	// 	var resp []GetDepositAddressResponse
	// 	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, accountSubsection, requestURL, nil, &resp, true)
	return nil, nil
}

// // GetAccountDepositHistory retrieves the deposit history of all tokens.100 recent records will be returned at maximum
// func (o *OKCoin) GetAccountDepositHistory(ctx context.Context, currency string) ([]GetAccountDepositHistoryResponse, error) {
// 	requestURL := getAccountDepositHistory
// 	if currency != "" {
// 		requestURL += "/" + currency
// 	}
// 	var resp []GetAccountDepositHistoryResponse
// 	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, accountSubsection, requestURL, nil, &resp, true)
// }

// // GetSpotTradingAccounts retrieves the list of assets(only show pairs with balance larger than 0), the balances, amount available/on hold in spot accounts.
// func (o *OKCoin) GetSpotTradingAccounts(ctx context.Context) ([]GetSpotTradingAccountResponse, error) {
// 	var resp []GetSpotTradingAccountResponse
// 	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, tokenSubsection, accounts, nil, &resp, true)
// }

// // GetSpotTradingAccountForCurrency This endpoint supports getting the balance, amount available/on hold of a token in spot account.
// func (o *OKCoin) GetSpotTradingAccountForCurrency(ctx context.Context, currency string) (*GetSpotTradingAccountResponse, error) {
// 	requestURL := accounts + "/" + currency
// 	var resp *GetSpotTradingAccountResponse
// 	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, tokenSubsection, requestURL, nil, &resp, true)
// }

// // GetSpotBillDetailsForCurrency This endpoint supports getting the balance, amount available/on hold of a token in spot account.
// func (o *OKCoin) GetSpotBillDetailsForCurrency(ctx context.Context, request *GetSpotBillDetailsForCurrencyRequest) ([]GetSpotBillDetailsForCurrencyResponse, error) {
// 	encodedRequest, err := encodeRequest(request)
// 	if err != nil {
// 		return nil, err
// 	}
// 	requestURL := accounts + "/" + request.Currency + "/" + ledger + encodedRequest
// 	var resp []GetSpotBillDetailsForCurrencyResponse
// 	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, tokenSubsection, requestURL, nil, &resp, true)
// }

// // PlaceSpotOrder token trading only supports limit and market orders (more order types will become available in the future).
// // You can place an order only if you have enough funds.
// // Once your order is placed, the amount will be put on hold.
// func (o *OKCoin) PlaceSpotOrder(ctx context.Context, request *PlaceOrderRequest) (*PlaceOrderResponse, error) {
// 	if request.OrderType == "" {
// 		request.OrderType = "0"
// 	}
// 	var resp *PlaceOrderResponse
// 	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, tokenSubsection, orders, request, &resp, true)
// }

// // PlaceMultipleSpotOrders supports placing multiple orders for specific trading pairs
// // up to 4 trading pairs, maximum 4 orders for each pair
// func (o *OKCoin) PlaceMultipleSpotOrders(ctx context.Context, request []PlaceOrderRequest) (map[string][]PlaceOrderResponse, []error) {
// 	currencyPairOrders := make(map[string]int)
// 	resp := make(map[string][]PlaceOrderResponse)

// 	for i := range request {
// 		if request[i].OrderType == "" {
// 			request[i].OrderType = "0"
// 		}
// 		currencyPairOrders[request[i].InstrumentID]++
// 	}

// 	if len(currencyPairOrders) > 4 {
// 		return resp, []error{errors.New("up to 4 trading pairs")}
// 	}
// 	for _, orderCount := range currencyPairOrders {
// 		if orderCount > 4 {
// 			return resp, []error{errors.New("maximum 4 orders for each pair")}
// 		}
// 	}

// 	err := o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, tokenSubsection, batchOrders, request, &resp, true)
// 	if err != nil {
// 		return resp, []error{err}
// 	}

// 	var orderErrors []error
// 	for currency, orderResponse := range resp {
// 		for i := range orderResponse {
// 			if !orderResponse[i].Result {
// 				orderErrors = append(orderErrors, fmt.Errorf("order for currency %v failed to be placed", currency))
// 			}
// 		}
// 	}

// 	return resp, orderErrors
// }

// // CancelSpotOrder Cancelling an unfilled order.
// func (o *OKCoin) CancelSpotOrder(ctx context.Context, request *CancelSpotOrderRequest) (*CancelSpotOrderResponse, error) {
// 	requestURL := cancelOrders + "/" + strconv.FormatInt(request.OrderID, 10)
// 	var resp *CancelSpotOrderResponse
// 	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, tokenSubsection, requestURL, request, &resp, true)
// }

// // CancelMultipleSpotOrders Cancelling multiple unfilled orders.
// func (o *OKCoin) CancelMultipleSpotOrders(ctx context.Context, request *CancelMultipleSpotOrdersRequest) (map[string][]CancelMultipleSpotOrdersResponse, error) {
// 	if len(request.OrderIDs) > 4 {
// 		return nil, errors.New("maximum 4 order cancellations for each pair")
// 	}

// 	resp := make(map[string][]CancelMultipleSpotOrdersResponse)
// 	err := o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, tokenSubsection, cancelBatchOrders, []CancelMultipleSpotOrdersRequest{*request}, &resp, true)
// 	if err != nil {
// 		return nil, err
// 	}
// 	for currency, orderResponse := range resp {
// 		for i := range orderResponse {
// 			cancellationResponse := CancelMultipleSpotOrdersResponse{
// 				OrderID:   orderResponse[i].OrderID,
// 				Result:    orderResponse[i].Result,
// 				ClientOID: orderResponse[i].ClientOID,
// 			}

// 			if !orderResponse[i].Result {
// 				cancellationResponse.Error = fmt.Errorf("order %v for currency %v failed to be cancelled", orderResponse[i].OrderID, currency)
// 			}

// 			resp[currency] = append(resp[currency], cancellationResponse)
// 		}
// 	}

// 	return resp, nil
// }

// // GetSpotOrders List your orders. Cursor pagination is used.
// // All paginated requests return the latest information (newest) as the first page sorted by newest (in chronological time) first.
// func (o *OKCoin) GetSpotOrders(ctx context.Context, request *GetSpotOrdersRequest) ([]GetSpotOrderResponse, error) {
// 	encodedRequest, err := encodeRequest(request)
// 	if err != nil {
// 		return nil, err
// 	}
// 	requestURL := orders + encodedRequest
// 	var resp []GetSpotOrderResponse
// 	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, tokenSubsection, requestURL, nil, &resp, true)
// }

// // GetSpotOpenOrders List all your current open orders. Cursor pagination is used.
// // All paginated requests return the latest information (newest) as the first page sorted by newest (in chronological time) first.
// func (o *OKCoin) GetSpotOpenOrders(ctx context.Context, request *GetSpotOpenOrdersRequest) ([]GetSpotOrderResponse, error) {
// 	encodedRequest, err := encodeRequest(request)
// 	if err != nil {
// 		return nil, err
// 	}
// 	requestURL := pendingOrders + encodedRequest
// 	var resp []GetSpotOrderResponse
// 	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, tokenSubsection, requestURL, nil, &resp, true)
// }

// // GetSpotOrder Get order details by order ID.
// func (o *OKCoin) GetSpotOrder(ctx context.Context, request *GetSpotOrderRequest) (*GetSpotOrderResponse, error) {
// 	encodedRequest, err := encodeRequest(request)
// 	if err != nil {
// 		return nil, err
// 	}
// 	requestURL := orders + "/" + request.OrderID + encodedRequest
// 	var resp *GetSpotOrderResponse
// 	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, tokenSubsection, requestURL, request, &resp, true)
// }

// // GetSpotTransactionDetails Get details of the recent filled orders. Cursor pagination is used.
// // All paginated requests return the latest information (newest) as the first page sorted by newest (in chronological time) first.
// func (o *OKCoin) GetSpotTransactionDetails(ctx context.Context, request *GetSpotTransactionDetailsRequest) ([]GetSpotTransactionDetailsResponse, error) {
// 	encodedRequest, err := encodeRequest(request)
// 	if err != nil {
// 		return nil, err
// 	}
// 	requestURL := getSpotTransactionDetails + encodedRequest
// 	var resp []GetSpotTransactionDetailsResponse
// 	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, tokenSubsection, requestURL, nil, &resp, true)
// }

// // GetOrderBook Getting the order book of a trading pair. Pagination is not
// // supported here. The whole book will be returned for one request. Websocket is
// // recommended here.
// func (o *OKCoin) GetOrderBook(ctx context.Context, request *GetOrderBookRequest, a asset.Item) (*GetOrderBookResponse, error) {
// 	var resp *GetOrderBookResponse
// 	if a != asset.Spot {
// 		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, a)
// 	}
// 	encodedRequest, err := encodeRequest(request)
// 	if err != nil {
// 		return nil, err
// 	}
// 	requestURL := instruments + "/" + request.InstrumentID + "/" + getSpotOrderBook + encodedRequest
// 	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, tokenSubsection, requestURL, nil, &resp, false)
// }

// // GetSpotAllTokenPairsInformationForCurrency Get the last traded price, best bid/ask price, 24 hour trading volume and more info of a currency
// func (o *OKCoin) GetSpotAllTokenPairsInformationForCurrency(ctx context.Context, currency string) (*GetSpotTokenPairsInformationResponse, error) {
// 	requestURL := instruments + "/" + currency + "/" + tickerData
// 	var resp *GetSpotTokenPairsInformationResponse
// 	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, tokenSubsection, requestURL, nil, &resp, false)
// }

// // GetSpotFilledOrdersInformation Get the recent 60 transactions of all trading pairs.
// // Cursor pagination is used. All paginated requests return the latest information (newest) as the first page sorted by newest (in chronological time) first.
// func (o *OKCoin) GetSpotFilledOrdersInformation(ctx context.Context, request *GetSpotFilledOrdersInformationRequest) ([]GetSpotFilledOrdersInformationResponse, error) {
// 	encodedRequest, err := encodeRequest(request)
// 	if err != nil {
// 		return nil, err
// 	}
// 	requestURL := instruments + "/" + request.InstrumentID + "/" + trades + encodedRequest
// 	var resp []GetSpotFilledOrdersInformationResponse
// 	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, tokenSubsection, requestURL, nil, &resp, false)
// }

// // GetMarketData Get the charts of the trading pairs. Charts are returned in grouped buckets based on requested granularity.
// func (o *OKCoin) GetMarketData(ctx context.Context, request *GetMarketDataRequest) ([]kline.Candle, error) {
// 	encodedRequest, err := encodeRequest(request)
// 	if err != nil {
// 		return nil, err
// 	}
// 	requestURL := instruments + "/" + request.InstrumentID + "/" + getSpotMarketData + encodedRequest
// 	if request.Asset != asset.Spot && request.Asset != asset.Margin {
// 		return nil, asset.ErrNotSupported
// 	}
// 	var resp []interface{}
// 	err = o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, tokenSubsection, requestURL, nil, &resp, false)
// 	if err != nil {
// 		return nil, err
// 	}
// 	candles := make([]kline.Candle, len(resp))
// 	for x := range resp {
// 		t, ok := resp[x].([]interface{})
// 		if !ok {
// 			return nil, common.GetAssertError("[]interface{}", resp[x])
// 		}
// 		if len(t) < 6 {
// 			return nil, fmt.Errorf("%w expteced %v received %v", errIncorrectCandleDataLength, 6, len(t))
// 		}
// 		v, ok := t[0].(string)
// 		if !ok {
// 			return nil, common.GetAssertError("string", t[0])
// 		}
// 		var tempCandle kline.Candle
// 		if tempCandle.Time, err = time.Parse(time.RFC3339, v); err != nil {
// 			return nil, err
// 		}
// 		if tempCandle.Open, err = convert.FloatFromString(t[1]); err != nil {
// 			return nil, err
// 		}
// 		if tempCandle.High, err = convert.FloatFromString(t[2]); err != nil {
// 			return nil, err
// 		}
// 		if tempCandle.Low, err = convert.FloatFromString(t[3]); err != nil {
// 			return nil, err
// 		}
// 		if tempCandle.Close, err = convert.FloatFromString(t[4]); err != nil {
// 			return nil, err
// 		}
// 		if tempCandle.Volume, err = convert.FloatFromString(t[5]); err != nil {
// 			return nil, err
// 		}
// 		candles[x] = tempCandle
// 	}
// 	return candles, nil
// }

// // GetMarginTradingAccounts List all assets under token margin trading account, including information such as balance, amount on hold and more.
// func (o *OKCoin) GetMarginTradingAccounts(ctx context.Context) ([]GetMarginAccountsResponse, error) {
// 	var resp []GetMarginAccountsResponse
// 	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, marginTradingSubsection, accounts, nil, &resp, true)
// }

// // GetMarginTradingAccountsForCurrency Get the balance, amount on hold and more useful information.
// func (o *OKCoin) GetMarginTradingAccountsForCurrency(ctx context.Context, currency string) (*GetMarginAccountsResponse, error) {
// 	requestURL := accounts + "/" + currency
// 	var resp *GetMarginAccountsResponse
// 	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, marginTradingSubsection, requestURL, nil, &resp, true)
// }

// // GetMarginBillDetails List all bill details. Pagination is used here.
// // before and after cursor arguments should not be confused with before and after in chronological time.
// // Most paginated requests return the latest information (newest) as the first page sorted by newest (in chronological time) first.
// func (o *OKCoin) GetMarginBillDetails(ctx context.Context, request *GetMarginBillDetailsRequest) ([]GetSpotBillDetailsForCurrencyResponse, error) {
// 	encodedRequest, err := encodeRequest(request)
// 	if err != nil {
// 		return nil, err
// 	}
// 	requestURL := accounts + "/" + request.InstrumentID + "/" + ledger + encodedRequest
// 	var resp []GetSpotBillDetailsForCurrencyResponse
// 	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, marginTradingSubsection, requestURL, nil, &resp, true)
// }

// // GetMarginAccountSettings Get all information of the margin trading account,
// // including the maximum loan amount, interest rate, and maximum leverage.
// func (o *OKCoin) GetMarginAccountSettings(ctx context.Context, currency string) ([]GetMarginAccountSettingsResponse, error) {
// 	requestURL := accounts + "/" + getMarketAvailability
// 	if currency != "" {
// 		requestURL = accounts + "/" + currency + "/" + getMarketAvailability
// 	}
// 	var resp []GetMarginAccountSettingsResponse
// 	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, marginTradingSubsection, requestURL, nil, &resp, true)
// }

// // GetMarginLoanHistory Get loan history of the margin trading account.
// // Pagination is used here. before and after cursor arguments should not be confused with before and after in chronological time.
// // Most paginated requests return the latest information (newest) as the first page sorted by newest (in chronological time) first.
// func (o *OKCoin) GetMarginLoanHistory(ctx context.Context, request *GetMarginLoanHistoryRequest) ([]GetMarginLoanHistoryResponse, error) {
// 	requestURL := accounts + "/" + getLoan
// 	if len(request.InstrumentID) > 0 {
// 		requestURL = accounts + "/" + request.InstrumentID + "/" + getLoan
// 	}
// 	var resp []GetMarginLoanHistoryResponse
// 	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, marginTradingSubsection, requestURL, nil, &resp, true)
// }

// // OpenMarginLoan Borrowing tokens in a margin trading account.
// func (o *OKCoin) OpenMarginLoan(ctx context.Context, request *OpenMarginLoanRequest) (*OpenMarginLoanResponse, error) {
// 	requestURL := accounts + "/" + getLoan
// 	var resp *OpenMarginLoanResponse
// 	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, marginTradingSubsection, requestURL, request, &resp, true)
// }

// // RepayMarginLoan Repaying tokens in a margin trading account.
// func (o *OKCoin) RepayMarginLoan(ctx context.Context, request *RepayMarginLoanRequest) (*RepayMarginLoanResponse, error) {
// 	requestURL := accounts + "/" + getRepayment
// 	var resp *RepayMarginLoanResponse
// 	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, marginTradingSubsection, requestURL, request, &resp, true)
// }

// // PlaceMarginOrder You can place an order only if you have enough funds. Once your order is placed, the amount will be put on hold.
// func (o *OKCoin) PlaceMarginOrder(ctx context.Context, request *PlaceOrderRequest) (*PlaceOrderResponse, error) {
// 	var resp *PlaceOrderResponse
// 	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, marginTradingSubsection, orders, request, &resp, true)
// }

// // PlaceMultipleMarginOrders Place multiple orders for specific trading pairs (up to 4 trading pairs, maximum 4 orders each)
// func (o *OKCoin) PlaceMultipleMarginOrders(ctx context.Context, request []PlaceOrderRequest) (map[string][]PlaceOrderResponse, []error) {
// 	currencyPairOrders := make(map[string]int)
// 	resp := make(map[string][]PlaceOrderResponse)
// 	for i := range request {
// 		currencyPairOrders[request[i].InstrumentID]++
// 	}
// 	if len(currencyPairOrders) > 4 {
// 		return resp, []error{errors.New("up to 4 trading pairs")}
// 	}
// 	for _, orderCount := range currencyPairOrders {
// 		if orderCount > 4 {
// 			return resp, []error{errors.New("maximum 4 orders for each pair")}
// 		}
// 	}

// 	err := o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, marginTradingSubsection, batchOrders, request, &resp, true)
// 	if err != nil {
// 		return resp, []error{err}
// 	}

// 	var orderErrors []error
// 	for currency, orderResponse := range resp {
// 		for i := range orderResponse {
// 			if !orderResponse[i].Result {
// 				orderErrors = append(orderErrors, fmt.Errorf("order for currency %v failed to be placed", currency))
// 			}
// 		}
// 	}

// 	return resp, orderErrors
// }

// // CancelMarginOrder Cancelling an unfilled order.
// func (o *OKCoin) CancelMarginOrder(ctx context.Context, request *CancelSpotOrderRequest) (*CancelSpotOrderResponse, error) {
// 	requestURL := cancelOrders + "/" + strconv.FormatInt(request.OrderID, 10)
// 	var resp *CancelSpotOrderResponse
// 	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, marginTradingSubsection, requestURL, request, &resp, true)
// }

// // CancelMultipleMarginOrders Cancelling multiple unfilled orders.
// func (o *OKCoin) CancelMultipleMarginOrders(ctx context.Context, request *CancelMultipleSpotOrdersRequest) (map[string][]CancelMultipleSpotOrdersResponse, []error) {
// 	resp := make(map[string][]CancelMultipleSpotOrdersResponse)
// 	if len(request.OrderIDs) > 4 {
// 		return resp, []error{errors.New("maximum 4 order cancellations for each pair")}
// 	}

// 	err := o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, marginTradingSubsection, cancelBatchOrders, []CancelMultipleSpotOrdersRequest{*request}, &resp, true)
// 	if err != nil {
// 		return resp, []error{err}
// 	}

// 	var orderErrors []error
// 	for currency, orderResponse := range resp {
// 		for i := range orderResponse {
// 			if !orderResponse[i].Result {
// 				orderErrors = append(orderErrors, fmt.Errorf("order %v for currency %v failed to be cancelled", orderResponse[i].OrderID, currency))
// 			}
// 		}
// 	}

// 	return resp, orderErrors
// }

// // GetMarginOrders List your orders. Cursor pagination is used. All paginated requests return the latest information (newest) as the first page sorted by newest (in chronological time) first.
// func (o *OKCoin) GetMarginOrders(ctx context.Context, request *GetSpotOrdersRequest) ([]GetSpotOrderResponse, error) {
// 	encodedRequest, err := encodeRequest(request)
// 	if err != nil {
// 		return nil, err
// 	}
// 	requestURL := orders + encodedRequest
// 	var resp []GetSpotOrderResponse
// 	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, marginTradingSubsection, requestURL, nil, &resp, true)
// }

// // GetMarginOpenOrders List all your current open orders. Cursor pagination is used. All paginated requests return the latest information (newest) as the first page sorted by newest (in chronological time) first.
// func (o *OKCoin) GetMarginOpenOrders(ctx context.Context, request *GetSpotOpenOrdersRequest) ([]GetSpotOrderResponse, error) {
// 	encodedRequest, err := encodeRequest(request)
// 	if err != nil {
// 		return nil, err
// 	}
// 	requestURL := pendingOrders + encodedRequest
// 	var resp []GetSpotOrderResponse
// 	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, marginTradingSubsection, requestURL, nil, &resp, true)
// }

// // GetMarginOrder Get order details by order ID.
// func (o *OKCoin) GetMarginOrder(ctx context.Context, request *GetSpotOrderRequest) (*GetSpotOrderResponse, error) {
// 	encodedRequest, err := encodeRequest(request)
// 	if err != nil {
// 		return nil, err
// 	}
// 	requestURL := orders + "/" + request.OrderID + encodedRequest
// 	var resp *GetSpotOrderResponse
// 	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, marginTradingSubsection, requestURL, request, &resp, true)
// }

// // GetMarginTransactionDetails Get details of the recent filled orders. Cursor pagination is used.
// // All paginated requests return the latest information (newest) as the first page sorted by newest (in chronological time) first.
// func (o *OKCoin) GetMarginTransactionDetails(ctx context.Context, request *GetSpotTransactionDetailsRequest) ([]GetSpotTransactionDetailsResponse, error) {
// 	encodedRequest, err := encodeRequest(request)
// 	if err != nil {
// 		return nil, err
// 	}
// 	requestURL := getSpotTransactionDetails + encodedRequest
// 	var resp []GetSpotTransactionDetailsResponse
// 	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, marginTradingSubsection, requestURL, nil, &resp, true)
// }

// // encodeRequest Formats URL parameters, useful for optional parameters due to OKCoin signature check
// func encodeRequest(request interface{}) (string, error) {
// 	v, err := query.Values(request)
// 	if err != nil {
// 		return "", err
// 	}
// 	resp := v.Encode()
// 	if resp == "" {
// 		return resp, nil
// 	}
// 	return "?" + resp, nil
// }

// // GetErrorCode returns an error code
// func (o *OKCoin) GetErrorCode(code interface{}) error {
// 	var assertedCode string

// 	switch d := code.(type) {
// 	case float64:
// 		assertedCode = strconv.FormatFloat(d, 'f', -1, 64)
// 	case string:
// 		assertedCode = d
// 	default:
// 		return errors.New("unusual type returned")
// 	}

// 	if i, ok := o.ErrorCodes[assertedCode]; ok {
// 		return i
// 	}
// 	return errors.New("unable to find SPOT error code")
// }

// SendHTTPRequest sends an authenticated http request to a desired
// path with a JSON payload (of present)
// URL arguments must be in the request path and not as url.URL values
func (o *OKCoin) SendHTTPRequest(ctx context.Context, ep exchange.URL, httpMethod, requestType, requestPath string, data, result interface{}, authenticated bool) error {
	endpoint, err := o.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	resp := &struct {
		Code    int64       `json:"code,string"`
		Message string      `json:"msg"`
		Data    interface{} `json:"data"`
	}{
		Data: result,
	}
	var intermediary json.RawMessage
	newRequest := func() (*request.Item, error) {
		utcTime := time.Now().UTC().Format(time.RFC3339)
		payload := []byte("")

		if data != nil {
			payload, err = json.Marshal(data)
			if err != nil {
				return nil, err
			}
		}
		path := endpoint + okCoinAPIVersion + requestType + "/" + requestPath
		headers := make(map[string]string)
		headers["Content-Type"] = "application/json"
		if authenticated {
			var creds *account.Credentials
			creds, err = o.GetCredentials(ctx)
			if err != nil {
				return nil, err
			}
			signPath := okCoinAPIVersion + requestType + "/" + requestPath

			var hmac []byte
			hmac, err = crypto.GetHMAC(crypto.HashSHA256,
				[]byte(utcTime+httpMethod+signPath+string(payload)),
				[]byte(creds.Secret))
			if err != nil {
				return nil, err
			}
			headers["OK-ACCESS-KEY"] = creds.Key
			headers["OK-ACCESS-SIGN"] = crypto.Base64Encode(hmac)
			headers["OK-ACCESS-TIMESTAMP"] = utcTime
			headers["OK-ACCESS-PASSPHRASE"] = creds.ClientID
		}

		return &request.Item{
			Method:        strings.ToUpper(httpMethod),
			Path:          path,
			Headers:       headers,
			Body:          bytes.NewBuffer(payload),
			Result:        &intermediary,
			AuthRequest:   authenticated,
			Verbose:       o.Verbose,
			HTTPDebugging: o.HTTPDebugging,
			HTTPRecording: o.HTTPRecording,
		}, nil
	}

	err = o.SendPayload(ctx, request.Unset, newRequest)
	if err != nil {
		return err
	}

	type errCapFormat struct {
		Error        int64  `json:"error_code"`
		ErrorMessage string `json:"error_message"`
		Result       bool   `json:"result,string"`
	}
	errCap := errCapFormat{Result: true}
	err = json.Unmarshal(intermediary, &errCap)
	if err == nil {
		if errCap.Error > 0 {
			return fmt.Errorf("sendHTTPRequest error - %s", o.ErrorCodes[strconv.FormatInt(errCap.Error, 10)])
		}
		if errCap.ErrorMessage != "" {
			return fmt.Errorf("error: %v", errCap.ErrorMessage)
		}
		if !errCap.Result {
			return errors.New("unspecified error occurred")
		}
	}
	err = json.Unmarshal(intermediary, resp)
	if err != nil {
		return err
	}
	if resp.Code > 2 {
		return fmt.Errorf("sendHTTPRequest error - code: %d message: %s", resp.Code, resp.Message)
	}
	return nil
}

// // GetFee returns an estimate of fee based on type of transaction
// func (o *OKCoin) GetFee(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
// 	var fee float64
// 	switch feeBuilder.FeeType {
// 	case exchange.CryptocurrencyTradeFee:
// 		fee = calculateTradingFee(feeBuilder.PurchasePrice, feeBuilder.Amount, feeBuilder.IsMaker)
// 	case exchange.CryptocurrencyWithdrawalFee:
// 		withdrawFees, err := o.GetAccountWithdrawalFee(ctx, feeBuilder.FiatCurrency.String())
// 		if err != nil {
// 			return -1, err
// 		}
// 		for _, withdrawFee := range withdrawFees {
// 			if withdrawFee.Currency == feeBuilder.FiatCurrency.String() {
// 				fee = withdrawFee.MinFee
// 				break
// 			}
// 		}
// 	case exchange.OfflineTradeFee:
// 		fee = getOfflineTradeFee(feeBuilder.PurchasePrice, feeBuilder.Amount)
// 	}
// 	if fee < 0 {
// 		fee = 0
// 	}

// 	return fee, nil
// }

// // getOfflineTradeFee calculates the worst case-scenario trading fee
// func getOfflineTradeFee(price, amount float64) float64 {
// 	return 0.0015 * price * amount
// }

// func calculateTradingFee(purchasePrice, amount float64, isMaker bool) (fee float64) {
// 	// TODO volume based fees
// 	if isMaker {
// 		fee = 0.0005
// 	} else {
// 		fee = 0.0015
// 	}
// 	return fee * amount * purchasePrice
// }
