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
	apiPath                   = "/api/"
	okcoinAPIURL              = "https://www.okcoin.com"
	okcoinAPIVersion          = apiPath + "v5/"
	okcoinExchangeName        = "Okcoin"
	okcoinWebsocketURL        = "wss://real.okcoin.com:8443/ws/v5/public"
	okcoinPrivateWebsocketURL = "wss://real.okcoin.com:8443/ws/v5/private"
)

// Okcoin is the overarching type used for Okcoin's exchange API implementation
type Okcoin struct {
	exchange.Base
	// Spot and contract market error codes
	ErrorCodes map[string]error
}

var (
	errNilArgument                            = errors.New("nil argument")
	errInvalidAmount                          = errors.New("invalid amount value")
	errInvalidPrice                           = errors.New("invalid price value")
	errAddressMustNotBeEmptyString            = errors.New("address must be a non-empty string")
	errSubAccountNameRequired                 = errors.New("sub-account name is required")
	errNoValidResponseFromServer              = errors.New("no valid response")
	errTransferIDOrClientIDRequired           = errors.New("either transfer id or client id is required")
	errInvalidWithdrawalMethod                = errors.New("withdrawal method must be specified")
	errInvalidTransactionFeeValue             = errors.New("invalid transaction fee value")
	errWithdrawalIDMissing                    = errors.New("withdrawal id is missing")
	errTradeModeIsRequired                    = errors.New("trade mode is required")
	errInstrumentTypeMissing                  = errors.New("instrument type is required")
	errChannelIDRequired                      = errors.New("channel id is required")
	errBankAccountNumberIsRequired            = errors.New("bank account number is required")
	errMissingInstrumentID                    = errors.New("missing instrument id")
	errAlgoIDRequired                         = errors.New("algo ID is required")
	errNoOrderbookData                        = errors.New("no orderbook data found")
	errOrderIDOrClientOrderIDRequired         = errors.New("order id or client order id is required")
	errSizeOrPriceRequired                    = errors.New("valid size or price has to be specified")
	errPriceRatioOrPriceSpreadRequired        = errors.New("either price ratio or price variance is required")
	errSizeLimitRequired                      = errors.New("size limit is required")
	errPriceLimitRequired                     = errors.New("price limit is required")
	errStopLossOrTakeProfitOrderPriceRequired = errors.New("either parameter 'stop loss order price' or 'take profit order price' is required")
	errStopLossTriggerPriceRequired           = errors.New("stop-loss-order-price is required")
	errTakeProfitOrderPriceRequired           = errors.New("tp-order-price is required")
	errTpTriggerOrderPriceTypeRequired        = errors.New("'take-profit-order-price-type' is required")
	errStopLossTriggerPriceTypeRequired       = errors.New("'stop-loss-trigger-price' is required")
	errCallbackRatioOrCallbackSpeedRequired   = errors.New("either Callback ration or callback spread is required")
	errTimeIntervalInformationRequired        = errors.New("time interval information is required")
	errOrderTypeRequired                      = errors.New("order type is required")
	errNoAccountDepositAddress                = errors.New("no account deposit address")
	errQuoteIDRequired                        = errors.New("quote id is required")
	errClientRequestIDRequired                = errors.New("client supplied request ID is required")
)

const (
	// endpoint types
	typeAccounts = "account"
	typeFiat     = "fiat"
	typeOtc      = "otc"
	typeAssets   = "asset"
	typeMarket   = "market"
	typePublic   = "public"
	typeSystem   = "system"
	typeTrade    = "trade"
	typeUser     = "users"
)

// GetInstruments Get market data. This endpoint provides the snapshots of market data and can be used without verifications.
// List trading pairs and get the trading limit, price, and more information of different trading pairs.
func (o *Okcoin) GetInstruments(ctx context.Context, instrumentType, instrumentID string) ([]Instrument, error) {
	params := url.Values{}
	if instrumentType == "" {
		return nil, errInstrumentTypeMissing
	}
	params.Set("instType", instrumentType)
	if instrumentID != "" {
		params.Set("instId", instrumentID)
	}
	var resp []Instrument
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, getInstrumentsEPL, http.MethodGet, typePublic, common.EncodeURLValues("instruments", params), nil, &resp, false)
}

// GetSystemStatus system maintenance status,scheduled: waiting; ongoing: processing; pre_open: pre_open; completed: completed ;canceled: canceled.
// Generally, pre_open last about 10 minutes. There will be pre_open when the time of upgrade is too long.
// If this parameter is not filled, the data with status scheduled, ongoing and pre_open will be returned by default
func (o *Okcoin) GetSystemStatus(ctx context.Context, state string) ([]SystemStatus, error) {
	params := url.Values{}
	if state != "" {
		params.Set("state", state)
	}
	var resp []SystemStatus
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, getSystemStatusEPL, http.MethodGet, typeSystem, common.EncodeURLValues("status", params), nil, &resp, false)
}

// GetSystemTime retrieve API server time.
func (o *Okcoin) GetSystemTime(ctx context.Context) (time.Time, error) {
	timestampResponse := []struct {
		Timestamp okcoinTime `json:"ts"`
	}{}
	err := o.SendHTTPRequest(ctx, exchange.RestSpot, getSystemTimeEPL, http.MethodGet, typePublic, "time", nil, &timestampResponse, false)
	if err != nil {
		return time.Time{}, err
	}
	return timestampResponse[0].Timestamp.Time(), nil
}

// GetTickers retrieve the latest price snapshot, best bid/ask price, and trading volume in the last 24 hours.
func (o *Okcoin) GetTickers(ctx context.Context, instrumentType string) ([]TickerData, error) {
	params := url.Values{}
	if instrumentType == "" {
		return nil, errInstrumentTypeMissing
	}
	params.Set("instType", instrumentType)
	var resp []TickerData
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, getTickersEPL, http.MethodGet, typeMarket, common.EncodeURLValues("tickers", params), nil, &resp, false)
}

// GetTicker retrieve the latest price snapshot, best bid/ask price, and trading volume in the last 24 hours.
func (o *Okcoin) GetTicker(ctx context.Context, instrumentID string) (*TickerData, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	var resp []TickerData
	err := o.SendHTTPRequest(ctx, exchange.RestSpot, getTickerEPL, http.MethodGet, typeMarket, "ticker"+"?instId="+instrumentID, nil, &resp, false)
	if err != nil {
		return nil, err
	} else if len(resp) == 0 {
		return nil, errors.New("instrument not found")
	}
	return &resp[0], nil
}

// GetOrderbook retrieve order book of the instrument.
func (o *Okcoin) GetOrderbook(ctx context.Context, instrumentID string, sideDepth int64) (*GetOrderBookResponse, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	if sideDepth > 0 {
		params.Set("sz", strconv.FormatInt(sideDepth, 10))
	}
	var resp []GetOrderBookResponse
	err := o.SendHTTPRequest(ctx, exchange.RestSpot, getOrderbookEPL, http.MethodGet, typeMarket, common.EncodeURLValues("books", params), nil, &resp, false)
	if err != nil {
		return nil, err
	} else if len(resp) == 0 {
		return nil, fmt.Errorf("%w for instrument %s", errNoOrderbookData, instrumentID)
	}
	return &resp[0], nil
}

// GetCandlesticks retrieve the candlestick charts. This endpoint can retrieve the latest 1,440 data entries. Charts are returned in groups based on the requested bar.
func (o *Okcoin) GetCandlesticks(ctx context.Context, instrumentID string, interval kline.Interval, after, before time.Time, limit int64, utcOpeningPrice bool) ([]CandlestickData, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	var err error
	if interval != kline.Interval(0) {
		var intervalString string
		intervalString, err = intervalToString(interval, utcOpeningPrice)
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
	err = o.SendHTTPRequest(ctx, exchange.RestSpot, getCandlesticksEPL, http.MethodGet, typeMarket, common.EncodeURLValues("candles", params), nil, &resp, false)
	if err != nil {
		return nil, err
	}
	return ExtractCandlesticks(resp)
}

// GetCandlestickHistory retrieve history candlestick charts from recent years.
func (o *Okcoin) GetCandlestickHistory(ctx context.Context, instrumentID string, start, end time.Time, bar kline.Interval, limit int64) ([]CandlestickData, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	var err error
	if bar != kline.Interval(0) {
		var intervalString string
		intervalString, err = intervalToString(bar, true)
		if err != nil {
			return nil, err
		}
		params.Set("bar", intervalString)
	}
	if !start.IsZero() {
		params.Set("start", strconv.FormatInt(start.UnixNano(), 10))
	}
	if !end.IsZero() {
		params.Set("end", strconv.FormatInt(end.UnixNano(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []candlestickItemResponse
	err = o.SendHTTPRequest(ctx, exchange.RestSpot, getCandlestickHistoryEPL, http.MethodGet, typeMarket, common.EncodeURLValues("history-candles", params), nil, &resp, false)
	if err != nil {
		return nil, err
	}
	return ExtractCandlesticks(resp)
}

// GetTrades retrieve the recent transactions of an instrument.
func (o *Okcoin) GetTrades(ctx context.Context, instrumentID string, limit int64) ([]SpotTrade, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []SpotTrade
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, getPublicTradesEPL, http.MethodGet, typeMarket, common.EncodeURLValues("trades", params), nil, &resp, false)
}

// GetTradeHistory retrieve the recent transactions of an instrument from the last 3 months with pagination.
func (o *Okcoin) GetTradeHistory(ctx context.Context, instrumentID, paginationType string, before, after time.Time, limit int64) ([]SpotTrade, error) {
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
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, getPublicTradeHistroryEPL, http.MethodGet, typeMarket, common.EncodeURLValues("history-trades", params), nil, &resp, false)
}

// Get24HourTradingVolume returns the 24-hour trading volume is calculated on a rolling basis, using USD as the pricing unit.
func (o *Okcoin) Get24HourTradingVolume(ctx context.Context) ([]TradingVolume, error) {
	var resp []TradingVolume
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, get24HourTradingVolumeEPL, http.MethodGet, typeMarket, "platform-24-volume", nil, &resp, false)
}

// GetOracle retrieves the crypto price of signing using Open Oracle smart contract.
func (o *Okcoin) GetOracle(ctx context.Context) (*Oracle, error) {
	var resp *Oracle
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, getOracleEPL, http.MethodGet, typeMarket, "open-oracle", nil, &resp, false)
}

// GetExchangeRate provides the average exchange rate data for 2 weeks
func (o *Okcoin) GetExchangeRate(ctx context.Context) ([]ExchangeRate, error) {
	var resp []ExchangeRate
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, getExchangeRateEPL, http.MethodGet, typeMarket, "exchange-rate", nil, &resp, false)
}

func intervalToString(interval kline.Interval, utcOpeningPrice bool) (string, error) {
	intervalMap := map[kline.Interval]string{
		kline.OneMin:     "1m",
		kline.ThreeMin:   "3m",
		kline.FiveMin:    "5m",
		kline.FifteenMin: "15m",
		kline.ThirtyMin:  "30m",
		kline.OneHour:    "1H",
		kline.TwoHour:    "2H",
		kline.FourHour:   "4H",
		kline.SixHour:    "6H",
		kline.TwelveHour: "12H",
		kline.OneDay:     "1D",
		kline.TwoDay:     "2D",
		kline.ThreeDay:   "3D",
		kline.OneWeek:    "1W",
		kline.OneMonth:   "1M",
		kline.ThreeMonth: "3M",
	}
	str, ok := intervalMap[interval]
	if !ok {
		return "", kline.ErrUnsupportedInterval
	}
	if utcOpeningPrice && (interval == kline.SixHour ||
		interval == kline.TwelveHour ||
		interval == kline.OneDay ||
		interval == kline.TwoDay ||
		interval == kline.ThreeDay ||
		interval == kline.OneWeek ||
		interval == kline.OneMonth ||
		interval == kline.ThreeMonth) {
		str += "utc"
	}
	return str, nil
}

// ------------ Funding endpoints --------------------------------

// GetCurrencies retrieves all list of currencies
func (o *Okcoin) GetCurrencies(ctx context.Context, ccy currency.Code) ([]CurrencyInfo, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.Upper().String())
	}
	var resp []CurrencyInfo
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, getFundingCurrenciesEPL, http.MethodGet, typeAssets, common.EncodeURLValues("currencies", params), nil, &resp, true)
}

// GetBalance retrieve the funding account balances of all the assets and the amount that is available or on hold.
func (o *Okcoin) GetBalance(ctx context.Context, ccy currency.Code) ([]CurrencyBalance, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	var resp []CurrencyBalance
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, getFundingAccountBalanceEPL, http.MethodGet, typeAssets, common.EncodeURLValues("balances", params), nil, &resp, true)
}

// GetAccountAssetValuation view account asset valuation
func (o *Okcoin) GetAccountAssetValuation(ctx context.Context, ccy currency.Code) ([]AccountAssetValuation, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	var resp []AccountAssetValuation
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, getAccountAssetValuationEPL, http.MethodGet, typeAssets, common.EncodeURLValues("asset-valuation", params), nil, &resp, true)
}

// FundsTransfer transfer of funds between your funding account and trading account, and from the master account to sub-accounts.
func (o *Okcoin) FundsTransfer(ctx context.Context, arg *FundingTransferRequest) (*FundingTransferItem, error) {
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
	err := o.SendHTTPRequest(ctx, exchange.RestSpot, fundingTransferEPL, http.MethodPost, typeAssets, "transfer", arg, &resp, true)
	if err != nil {
		return nil, err
	}
	if len(resp) == 0 {
		return nil, errNoValidResponseFromServer
	}
	return &resp[0], nil
}

// GetFundsTransferState retrieve the transfer state data of the last 2 weeks.
func (o *Okcoin) GetFundsTransferState(ctx context.Context, transferID, clientID, transferType string) ([]FundingTransferItem, error) {
	params := url.Values{}
	if transferID == "" && clientID == "" {
		return nil, errTransferIDOrClientIDRequired
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
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, getFundsTransferStateEPL, http.MethodGet, typeAssets, common.EncodeURLValues("transfer-state", params), nil, &resp, true)
}

// GetAssetBillsDetail query the billing record. You can get the latest 1 month historical data.
// Bill type 1: Deposit 2: Withdrawal 13: Canceled withdrawal 20: Transfer to sub account 21: Transfer from sub account
// 22: Transfer out from sub to master account 23: Transfer in from master to sub account 37: Transfer to spot 38: Transfer from spot
func (o *Okcoin) GetAssetBillsDetail(ctx context.Context, ccy currency.Code, billType, clientSuppliedID string, before, after time.Time, limit int64) ([]AssetBillDetail, error) {
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
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, assetBillsDetailEPL, http.MethodGet, typeAssets, common.EncodeURLValues("bills", params), nil, &resp, true)
}

// GetLightningDeposits retrieves lightning deposit instances
func (o *Okcoin) GetLightningDeposits(ctx context.Context, ccy currency.Code, amount float64, to string) ([]LightningDepositDetail, error) {
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
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, lightningDepositsEPL, http.MethodGet, typeAssets, common.EncodeURLValues("deposit-lightning", params), nil, &resp, true)
}

// GetCurrencyDepositAddresses retrieve the deposit addresses of currencies, including previously-used addresses.
func (o *Okcoin) GetCurrencyDepositAddresses(ctx context.Context, ccy currency.Code) ([]DepositAddress, error) {
	params := url.Values{}
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params.Set("ccy", ccy.String())
	var resp []DepositAddress
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, getAssetDepositAddressEPL, http.MethodGet, typeAssets, common.EncodeURLValues("deposit-address", params), nil, &resp, true)
}

// GetDepositHistory retrieve the deposit records according to the currency, deposit status, and time range in reverse chronological order. The 100 most recent records are returned by default.
func (o *Okcoin) GetDepositHistory(ctx context.Context, ccy currency.Code, depositID, transactionID, depositType, state string, after, before time.Time, limit int64) ([]DepositHistoryItem, error) {
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
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, getDepositHistoryEPL, http.MethodGet, typeAssets, common.EncodeURLValues("deposit-history", params), nil, &resp, true)
}

// Withdrawal apply withdrawal of tokens. Sub-account does not support withdrawal.
// Withdrawal method
// 3: 'internal' using email, phone or login account name.
// 4: 'on chain' a trusted crypto currency address.
func (o *Okcoin) Withdrawal(ctx context.Context, arg *WithdrawalRequest) ([]WithdrawalResponse, error) {
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
	if arg.TransactionFee < 0 {
		return nil, fmt.Errorf("%w, transaction fee: %f", errInvalidTransactionFeeValue, arg.TransactionFee)
	}
	var resp []WithdrawalResponse
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, postWithdrawalEPL, http.MethodPost, typeAssets, "withdrawal", arg, &resp, true)
}

// SubmitLightningWithdrawals the maximum withdrawal amount is 0.1 BTC per request, and 1 BTC in 24 hours.
// The minimum withdrawal amount is approximately 0.000001 BTC. Sub-account does not support withdrawal.
func (o *Okcoin) SubmitLightningWithdrawals(ctx context.Context, arg *LightningWithdrawalsRequest) ([]LightningWithdrawals, error) {
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
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, postLightningWithdrawalEPL, http.MethodPost, typeAssets, "withdrawal-lightning", arg, &resp, true)
}

// CancelWithdrawal cancel normal withdrawal requests, but you cannot cancel withdrawal requests on Lightning.
func (o *Okcoin) CancelWithdrawal(ctx context.Context, arg *WithdrawalCancellation) (*WithdrawalCancellation, error) {
	var resp []WithdrawalCancellation
	if arg.WithdrawalID == "" {
		return nil, errWithdrawalIDMissing
	}
	err := o.SendHTTPRequest(ctx, exchange.RestSpot, cancelWithdrawalEPL, http.MethodPost, typeAssets, "cancel-withdrawal", arg, &resp, true)
	if err != nil {
		return nil, err
	}
	if len(resp) == 0 {
		return nil, errNoValidResponseFromServer
	}
	return &resp[0], nil
}

// GetWithdrawalHistory retrieve the withdrawal records according to the currency, withdrawal status, and time range in reverse chronological order. The 100 most recent records are returned by default.
func (o *Okcoin) GetWithdrawalHistory(ctx context.Context, ccy currency.Code, withdrawalID, clientID, transactionID, withdrawalType, state string, after, before time.Time, limit int64) ([]WithdrawalOrderItem, error) {
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
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []WithdrawalOrderItem
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, getAssetWithdrawalHistoryEPL, http.MethodGet, typeAssets, common.EncodeURLValues("withdrawal-history", params), nil, &resp, true)
}

// ------------------------ Account Endpoints --------------------

// GetAccountBalance retrieve a list of assets (with non-zero balance), remaining balance, and available amount in the trading account.
func (o *Okcoin) GetAccountBalance(ctx context.Context, currencies ...currency.Code) ([]AccountBalanceInformation, error) {
	params := url.Values{}
	if len(currencies) > 0 {
		currencyString := ""
		var x int
		for x = range currencies {
			if x > 0 {
				currencyString += ","
			}
			if currencies[x].IsEmpty() {
				return nil, currency.ErrCurrencyPairEmpty
			}
		}
		if len(currencies) > 0 {
			params.Set("ccy", currencyString)
		}
	}
	var resp []AccountBalanceInformation
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, getAccountBalanceEPL, http.MethodGet, typeAccounts, common.EncodeURLValues("balance", params), nil, &resp, true)
}

// GetBillsDetails retrieve the bills of the account. The bill refers to all transaction records that result in changing the balance of an account. Pagination is supported, and the response is sorted with the most recent first.
// For the last 7 days.
func (o *Okcoin) GetBillsDetails(ctx context.Context, ccy currency.Code, instrumentType, billType, billSubType, afterBillID, beforeBillID string, begin, end time.Time, limit int64) ([]BillsDetail, error) {
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
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, getBillsDetailLast3MonthEPL, http.MethodGet, typeAccounts, common.EncodeURLValues("bills", params), nil, &resp, true)
}

// GetBillsDetailsFor3Months retrieve the bills of the account. The bill refers to all transaction records that result in changing the balance of an account. Pagination is supported, and the response is sorted with the most recent first.
// For the last 3 months.
func (o *Okcoin) GetBillsDetailsFor3Months(ctx context.Context, ccy currency.Code, instrumentType, billType, billSubType, afterBillID, beforeBillID string, begin, end time.Time, limit int64) ([]BillsDetail, error) {
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
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, getBillsDetailEPL, http.MethodGet, typeAccounts, common.EncodeURLValues("bills-archive", params), nil, &resp, true)
}

// GetAccountConfigurations retrieves current account configuration information.
func (o *Okcoin) GetAccountConfigurations(ctx context.Context) ([]AccountConfiguration, error) {
	var resp []AccountConfiguration
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, getAccountConfigurationEPL, http.MethodGet, typeAccounts, "config", nil, &resp, true)
}

// GetMaximumBuySellOrOpenAmount retrieves maximum buy, sell, or open amount information.
// Single instrument or multiple instruments (no more than 5) separated with comma, e.g. BTC-USD
// Trade mode 'cash'
// Price When the price is not specified, it will be calculated according to the last traded price.
// optional parameter
func (o *Okcoin) GetMaximumBuySellOrOpenAmount(ctx context.Context, instrumentID, tradeMode string, price float64) ([]MaxBuySellResp, error) {
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
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, getMaxBuySellAmountOpenAmountEPL, http.MethodGet, typeAccounts, common.EncodeURLValues("max-size", params), nil, &resp, true)
}

// GetMaximumAvailableTradableAmount retrieves maximum available tradable amount.
// Single instrument or multiple instruments (no more than 5) separated with comma, e.g. BTC-USDT,ETH-USDT
// Trade mode 'cash'
func (o *Okcoin) GetMaximumAvailableTradableAmount(ctx context.Context, tradeMode string, instrumentIDs ...string) ([]AvailableTradableAmount, error) {
	params := url.Values{}
	if len(instrumentIDs) == 0 {
		return nil, errMissingInstrumentID
	} else if len(instrumentIDs) > 5 {
		return nil, errors.New("instrument IDs can not be more than 5")
	}
	params.Set("instId", strings.Join(instrumentIDs, ","))
	if tradeMode == "" {
		return nil, errTradeModeIsRequired
	}
	params.Set("tdMode", tradeMode)
	var resp []AvailableTradableAmount
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, getMaxAvailableTradableAmountEPL, http.MethodGet, typeAccounts, common.EncodeURLValues("max-avail-size", params), nil, &resp, true)
}

// GetFeeRates retrieves instrument trading fee information.
func (o *Okcoin) GetFeeRates(ctx context.Context, instrumentType, instrumentID string) ([]FeeRate, error) {
	params := url.Values{}
	if instrumentType == "" {
		return nil, errInstrumentTypeMissing
	}
	params.Set("instType", instrumentType)
	if instrumentID != "" {
		params.Set("instId", instrumentID)
	}
	var resp []FeeRate
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, getFeeRatesEPL, http.MethodGet, typeAccounts, common.EncodeURLValues("trade-fee", params), nil, &resp, true)
}

// GetMaximumWithdrawals retrieve the maximum transferable amount from trading account to funding account. If no currency is specified, the transferable amount of all owned currencies will be returned.
func (o *Okcoin) GetMaximumWithdrawals(ctx context.Context, ccy currency.Code) ([]MaximumWithdrawal, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	var resp []MaximumWithdrawal
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, getMaxWithdrawalsEPL, http.MethodGet, typeAccounts, common.EncodeURLValues("max-withdrawal", params), nil, &resp, true)
}

// ------------------------------------ OTC-Desk RFQ --------------------------------

// GetAvailableRFQPairs retrieves a list of RFQ instruments.
func (o *Okcoin) GetAvailableRFQPairs(ctx context.Context) ([]AvailableRFQPair, error) {
	var resp []AvailableRFQPair
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, getAvailablePairsEPL, http.MethodGet, typeOtc, "rfq/instruments", nil, &resp, true)
}

// RequestQuote query current market quotation information
func (o *Okcoin) RequestQuote(ctx context.Context, arg *QuoteRequestArg) ([]RFQQuoteResponse, error) {
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
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, requestQuotesEPL, http.MethodPost, typeOtc, "rfq/quote", arg, &resp, true)
}

// PlaceRFQOrder submit RFQ order
func (o *Okcoin) PlaceRFQOrder(ctx context.Context, arg *PlaceRFQOrderRequest) ([]RFQOrderResponse, error) {
	if arg == nil {
		return nil, errNilArgument
	}
	if arg.ClientDefinedTradeRequestID == "" {
		return nil, errClientRequestIDRequired
	}
	if arg.QuoteID == "" {
		return nil, errQuoteIDRequired
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
	if arg.ClientRFQSendingTime == 0 {
		arg.ClientRFQSendingTime = time.Now().UnixMilli()
	}
	var resp []RFQOrderResponse
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, placeRFQOrderEPL, http.MethodPost, typeOtc, "rfq/trade", arg, &resp, true)
}

// GetRFQOrderDetails retrieves an RFQ order details.
func (o *Okcoin) GetRFQOrderDetails(ctx context.Context, clientDefinedID, tradeOrderID string) ([]RFQOrderDetail, error) {
	params := url.Values{}
	if clientDefinedID != "" {
		params.Set("clTReqId", clientDefinedID)
	}
	if tradeOrderID != "" {
		params.Set("tradeId", tradeOrderID)
	}
	var resp []RFQOrderDetail
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, getRFQTradeOrderDetailsEPL, http.MethodGet, typeOtc, common.EncodeURLValues("rfq/trade", params), nil, &resp, true)
}

// GetRFQOrderHistory retrieves an RFQ order history
func (o *Okcoin) GetRFQOrderHistory(ctx context.Context, begin, end time.Time, pageSize, pageIndex int64) ([]RFQOrderHistoryItem, error) {
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
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, getRFQTradeOrderHistoryEPL, http.MethodGet, typeOtc, common.EncodeURLValues("rfq/history", params), nil, &resp, true)
}

// ---------- Fiat ----------------------------------------------------------------

// Deposit posts a fiat deposit to an account
func (o *Okcoin) Deposit(ctx context.Context, arg *FiatDepositRequestArg) ([]FiatDepositResponse, error) {
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
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, fiatDepositEPL, http.MethodPost, typeFiat, "deposit", arg, &resp, true)
}

// CancelFiatDeposit cancels pending deposit requests.
func (o *Okcoin) CancelFiatDeposit(ctx context.Context, depositID string) (*CancelDepositAddressResp, error) {
	if depositID == "" {
		return nil, errors.New("deposit address required")
	}
	var resp []CancelDepositAddressResp
	err := o.SendHTTPRequest(ctx, exchange.RestSpot, fiatCancelDepositEPL, http.MethodPost, typeFiat, "cancel-deposit", map[string]string{"depId": depositID}, &resp, true)
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
func (o *Okcoin) GetFiatDepositHistory(ctx context.Context, ccy currency.Code, channelID, depositState, depositID string, after, before time.Time, limit int64) ([]DepositHistoryResponse, error) {
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
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, fiatDepositHistoryEPL, http.MethodGet, typeFiat, common.EncodeURLValues("deposit-history", params), nil, &resp, true)
}

// FiatWithdrawal submit fiat withdrawal operations.
func (o *Okcoin) FiatWithdrawal(ctx context.Context, arg *FiatWithdrawalParam) (*FiatWithdrawalResponse, error) {
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
	err := o.SendHTTPRequest(ctx, exchange.RestSpot, fiatWithdrawalEPL, http.MethodPost, typeFiat, "withdrawal", arg, &resp, true)
	if err != nil {
		return nil, err
	}
	if len(resp) == 0 {
		return nil, errNoValidResponseFromServer
	}
	return &resp[0], nil
}

// FiatCancelWithdrawal cancels fiat withdrawal request
func (o *Okcoin) FiatCancelWithdrawal(ctx context.Context, withdrawalID string) (string, error) {
	if withdrawalID == "" {
		return "", errWithdrawalIDMissing
	}
	var resp []struct {
		WithdrawalID string `json:"wdId"`
	}
	err := o.SendHTTPRequest(ctx, exchange.RestSpot, fiatCancelWithdrawalEPL, http.MethodPost, typeFiat, "cancel-withdrawal", map[string]string{"wdId": withdrawalID}, &resp, true)
	if err != nil {
		return "", err
	}
	if len(resp) == 0 {
		return "", errNoValidResponseFromServer
	}
	return resp[0].WithdrawalID, nil
}

// GetFiatWithdrawalHistory retrieves a fiat withdrawal orders list
// Channel ID used in the transaction.  9:PrimeX; 28:PrimeX US; 21:PrimeX Europe; 3:Silvergate SEN; 27:Silvergate SEN HK
// Withdrawal state. -2:User canceled the order；-1:Withdrawal attempt has failed; 0:Withdrawal request submitted; 1:Withdrawal request is pending; 2:Withdrawal has been credited
func (o *Okcoin) GetFiatWithdrawalHistory(ctx context.Context, ccy currency.Code, channelID, withdrawalState, withdrawalID string, after, before time.Time, limit int64) ([]FiatWithdrawalHistoryItem, error) {
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
		params.Set("wdId", withdrawalID)
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
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, fiatGetWithdrawalsEPL, http.MethodGet, typeFiat, common.EncodeURLValues("withdrawal-history", params), nil, &resp, true)
}

// GetChannelInfo retrieves channel detailed information given channel id.
func (o *Okcoin) GetChannelInfo(ctx context.Context, channelID string) ([]ChannelInfo, error) {
	params := url.Values{}
	if channelID != "" {
		params.Set("chanId", channelID)
	}
	var resp []ChannelInfo
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, fiatGetChannelInfoEPL, http.MethodGet, typeFiat, common.EncodeURLValues("channel", params), nil, &resp, true)
}

// ---------------------- Sub Account --------------------

// GetSubAccounts lists the sub-accounts detail.
// Applies to master accounts only
func (o *Okcoin) GetSubAccounts(ctx context.Context, enable bool, subAccountName string, after, before time.Time, limit int64) ([]SubAccountInfo, error) {
	params := url.Values{}
	if enable {
		params.Set("enable", "true")
	}
	if subAccountName != "" {
		params.Set("subAcct", subAccountName)
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
	var resp []SubAccountInfo
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, subAccountsListEPL, http.MethodGet, typeUser, common.EncodeURLValues("subaccount/list", params), nil, &resp, true)
}

// GetAPIKeyOfSubAccount retrieves sub-account's API Key information.
func (o *Okcoin) GetAPIKeyOfSubAccount(ctx context.Context, subAccountName, apiKey string) ([]SubAccountAPIKey, error) {
	params := url.Values{}
	if subAccountName == "" {
		return nil, errSubAccountNameRequired
	}
	params.Set("subAcct", subAccountName)
	if apiKey != "" {
		params.Set("apiKey", apiKey)
	}
	var resp []SubAccountAPIKey
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, getAPIKeyOfASubAccountEPL, http.MethodGet, typeUser, common.EncodeURLValues("subaccount/apikey", params), nil, &resp, true)
}

// GetSubAccountTradingBalance retrieves detailed balance info of Trading Account of a sub-account via the master account (applies to master accounts only).
func (o *Okcoin) GetSubAccountTradingBalance(ctx context.Context, subAccountName string) ([]SubAccountTradingBalance, error) {
	if subAccountName == "" {
		return nil, errSubAccountNameRequired
	}
	var resp []SubAccountTradingBalance
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, getSubAccountTradingBalanceEPL, http.MethodGet, typeAccounts, "subaccount/balances?subAcct="+subAccountName, nil, &resp, true)
}

// GetSubAccountFundingBalance retrieves detailed balance info of Funding Account of a sub-account via the master account (applies to master accounts only)
func (o *Okcoin) GetSubAccountFundingBalance(ctx context.Context, subAccountName string, currencies ...string) ([]SubAccountFundingBalance, error) {
	params := url.Values{}
	if subAccountName == "" {
		return nil, errSubAccountNameRequired
	}
	params.Set("subAcct", subAccountName)
	if len(currencies) > 0 {
		params.Set("ccy", strings.Join(currencies, ","))
	}
	var resp []SubAccountFundingBalance
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, getSubAccountFundingBalanceEPL, http.MethodGet, typeAssets, common.EncodeURLValues("subaccount/balances", params), nil, &resp, true)
}

// SubAccountTransferHistory retrieve the transfer data for the last 3 months.
// Applies to master accounts only.
// 0: Transfers from master account to sub-account ;
// 1 : Transfers from sub-account to master account.
func (o *Okcoin) SubAccountTransferHistory(ctx context.Context, subAccountName, currency, transferType string, after, before time.Time, limit int64) ([]SubAccountTransferInfo, error) {
	params := url.Values{}
	if currency != "" {
		params.Set("currency", currency)
	}
	if transferType != "" {
		params.Set("type", transferType)
	}
	if subAccountName != "" {
		params.Set("subAcct", subAccountName)
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
	var resp []SubAccountTransferInfo
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, subAccountTransferHistoryEPL, http.MethodGet, typeAssets, common.EncodeURLValues("subaccount/bills", params), nil, &resp, true)
}

// AccountBalanceTransfer posts an account transfer between master and sub-account transfers.
func (o *Okcoin) AccountBalanceTransfer(ctx context.Context, arg *IntraAccountTransferParam) (*SubAccountTransferResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Ccy == "" {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Amount <= 0 {
		return nil, fmt.Errorf("%w amount: %f", errInvalidAmount, arg.Amount)
	}
	// 6:Funding Account 18:Trading account
	if arg.From != "6" && arg.From != "18" {
		return nil, fmt.Errorf("invalid source account type %s, 6:Funding Account 18:Trading account", arg.From)
	}
	// 6:Funding Account 18:Trading account
	if arg.To != "6" && arg.To != "18" {
		return nil, fmt.Errorf("invalid destination account type %s, 6:Funding Account 18:Trading account", arg.To)
	}
	if arg.FromSubAccount == "" {
		return nil, fmt.Errorf("%w, source subaccount must be specified", errSubAccountNameRequired)
	}
	if arg.ToSubAccount == "" {
		return nil, fmt.Errorf("%w, destination subaccount must be specified", errSubAccountNameRequired)
	}
	var resp []SubAccountTransferResponse
	err := o.SendHTTPRequest(ctx, exchange.RestSpot, masterAccountsManageTransfersBetweenSubaccountEPL, http.MethodPost, typeAssets, "subaccount/transfer", arg, &resp, true)
	if err != nil {
		return nil, err
	}
	if len(resp) == 0 {
		return nil, errNoValidResponseFromServer
	}
	return &resp[0], nil
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
func (o *Okcoin) PlaceOrder(ctx context.Context, arg *PlaceTradeOrderParam) (*TradeOrderResponse, error) {
	err := arg.validateTradeOrderParameter()
	if err != nil {
		return nil, err
	}
	var resp []TradeOrderResponse
	err = o.SendHTTPRequest(ctx, exchange.RestSpot, placeTradeOrderEPL, http.MethodPost, typeTrade, "order", arg, &resp, true)
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
func (o *Okcoin) PlaceMultipleOrder(ctx context.Context, args []PlaceTradeOrderParam) ([]TradeOrderResponse, error) {
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
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, placeTradeMultipleOrdersEPL, http.MethodPost, typeTrade, "batch-orders", &args, &resp, true)
}

// CancelTradeOrder cancels a single trade order
func (o *Okcoin) CancelTradeOrder(ctx context.Context, arg *CancelTradeOrderRequest) (*TradeOrderResponse, error) {
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
	err := o.SendHTTPRequest(ctx, exchange.RestSpot, cancelTradeOrderEPL, http.MethodPost, typeTrade, "cancel-order", arg, &resp, true)
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
func (o *Okcoin) CancelMultipleOrders(ctx context.Context, args []CancelTradeOrderRequest) ([]TradeOrderResponse, error) {
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
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, cancelMultipleOrderEPL, http.MethodPost, typeTrade, "cancel-batch-orders", &args, &resp, true)
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
func (o *Okcoin) AmendOrder(ctx context.Context, arg *AmendTradeOrderRequestParam) (*AmendTradeOrderResponse, error) {
	err := arg.validate()
	if err != nil {
		return nil, err
	}
	var resp []AmendTradeOrderResponse
	err = o.SendHTTPRequest(ctx, exchange.RestSpot, amendTradeOrderEPL, http.MethodPost, typeTrade, "amend-order", arg, &resp, true)
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

// AmendMultipleOrder amends multiple trade orders.
func (o *Okcoin) AmendMultipleOrder(ctx context.Context, args []AmendTradeOrderRequestParam) ([]AmendTradeOrderResponse, error) {
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
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, amendMultipleOrdersEPL, http.MethodPost, typeTrade, "amend-batch-orders", &args, &resp, true)
}

// GetPersonalOrderDetail retrieves an order detail
func (o *Okcoin) GetPersonalOrderDetail(ctx context.Context, instrumentID, orderID, clientOrderID string) (*TradeOrder, error) {
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
	err := o.SendHTTPRequest(ctx, exchange.RestSpot, getOrderDetailsEPL, http.MethodGet, typeTrade, common.EncodeURLValues("order", params), nil, &resp, true)
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
func (o *Okcoin) GetPersonalOrderList(ctx context.Context, instrumentType, instrumentID, orderType, state string, before, after time.Time, limit int64) ([]TradeOrder, error) {
	params := tradeOrderParamsFill(instrumentType, instrumentID, orderType, state, after, before, limit)
	var resp []TradeOrder
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, getOrderListEPL, http.MethodGet, typeTrade, common.EncodeURLValues("orders-pending", params), nil, &resp, true)
}

// GetOrderHistory7Days retrieve the completed order data for the last 7 days, and the incomplete orders that have been canceled are only reserved for 2 hours.
func (o *Okcoin) GetOrderHistory7Days(ctx context.Context, instrumentType, instrumentID, orderType, state string, before, after time.Time, limit int64) ([]TradeOrder, error) {
	params := tradeOrderParamsFill(instrumentType, instrumentID, orderType, state, after, before, limit)
	var resp []TradeOrder
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, getOrderHistoryEPL, http.MethodGet, typeTrade, common.EncodeURLValues("orders-history", params), nil, &resp, true)
}

// GetOrderHistory3Months retrieve the completed order data of the last 3 months.
func (o *Okcoin) GetOrderHistory3Months(ctx context.Context, instrumentType, instrumentID, orderType, state string, before, after time.Time, limit int64) ([]TradeOrder, error) {
	params := tradeOrderParamsFill(instrumentType, instrumentID, orderType, state, after, before, limit)
	var resp []TradeOrder
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, getOrderhistory3MonthsEPL, http.MethodGet, typeTrade, common.EncodeURLValues("orders-history-archive", params), nil, &resp, true)
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
func (o *Okcoin) GetRecentTransactionDetail(ctx context.Context, instrumentType, instrumentID, orderID, afterBillID, beforeBillID string, begin, end time.Time, limit int64) ([]TransactionFillItem, error) {
	params := transactionFillParams(instrumentType, instrumentID, orderID, afterBillID, beforeBillID, begin, end, limit)
	var resp []TransactionFillItem
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, getTransactionDetails3DaysEPL, http.MethodGet, typeTrade, common.EncodeURLValues("fills", params), nil, &resp, true)
}

// GetTransactionDetails3Months retrieves recently filled transaction detail in the last 3-months
func (o *Okcoin) GetTransactionDetails3Months(ctx context.Context, instrumentType, instrumentID, orderID, beforeBillID, afterBillID string, begin, end time.Time, limit int64) ([]TransactionFillItem, error) {
	params := transactionFillParams(instrumentType, instrumentID, orderID, afterBillID, beforeBillID, begin, end, limit)
	var resp []TransactionFillItem
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, getTransactionDetails3MonthsEPL, http.MethodGet, typeTrade, common.EncodeURLValues("fills-history", params), nil, &resp, true)
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
		if arg.StopLossOrderPrice <= 0 && arg.TpOrderPrice <= 0 {
			return errStopLossOrTakeProfitOrderPriceRequired
		}
		if arg.StopLossOrderPrice > 0 {
			if arg.StopLossTriggerPrice <= 0 {
				return fmt.Errorf("for you specify 'stop-loss-order-price', %w", errStopLossTriggerPriceRequired)
			}
			if arg.StopLossTriggerPriceType == "" {
				return errStopLossTriggerPriceTypeRequired
			}
		} else if arg.TpOrderPrice > 0 {
			if arg.TpTriggerPrice <= 0 {
				return fmt.Errorf("for you specify 'take-profit-order-price', %w", errTakeProfitOrderPriceRequired)
			}
			if arg.TpTriggerOrderPriceType == "" {
				return errTpTriggerOrderPriceTypeRequired
			}
		}
	case "oco":
		//  One-cancels-the-other order
		if arg.TpOrderPrice <= 0 {
			return errTakeProfitOrderPriceRequired
		}
		if arg.TpTriggerOrderPriceType == "" {
			return errTpTriggerOrderPriceTypeRequired
		}
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
			return errCallbackRatioOrCallbackSpeedRequired
		}
	case "iceberg":
		//  Iceberg order
		if arg.PriceRatio <= 0 && arg.PriceSpread <= 0 {
			return errPriceRatioOrPriceSpreadRequired
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
			return errPriceRatioOrPriceSpreadRequired
		}
		if arg.SizeLimit <= 0 {
			return fmt.Errorf("%w, order type %s", errSizeLimitRequired, arg.OrderType)
		}
		if arg.PriceLimit <= 0 {
			return fmt.Errorf("%w, order type %s", errPriceLimitRequired, arg.OrderType)
		}
		if arg.TimeInterval == "" {
			return errTimeIntervalInformationRequired
		}
	}
	return nil
}

// PlaceAlgoOrder places an algo order.
// The algo order includes trigger order, oco order, conditional order,iceberg order, twap order and trailing order.
func (o *Okcoin) PlaceAlgoOrder(ctx context.Context, arg *AlgoOrderRequestParam) (*AlgoOrderResponse, error) {
	err := arg.validateAlgoOrder()
	if err != nil {
		return nil, err
	}
	var resp []AlgoOrderResponse
	err = o.SendHTTPRequest(ctx, exchange.RestSpot, placeAlgoOrderEPL, http.MethodPost, typeTrade, "order-algo", arg, &resp, true)
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

// CancelAlgoOrder cancel unfilled algo orders (not including Iceberg order, TWAP order, Trailing Stop order).
// A maximum of 10 orders can be canceled per request. Request parameters should be passed in the form of an array.
func (o *Okcoin) CancelAlgoOrder(ctx context.Context, args []CancelAlgoOrderRequestParam) ([]AlgoOrderResponse, error) {
	if len(args) == 0 {
		return nil, errNilArgument
	}
	for a := range args {
		if args[a].InstrumentID == "" {
			return nil, errMissingInstrumentID
		}
		if args[a].AlgoOrderID == "" {
			return nil, errAlgoIDRequired
		}
	}
	var resp []AlgoOrderResponse
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, cancelAlgoOrderEPL, http.MethodPost, typeTrade, "cancel-algos", &args, &resp, true)
}

// CancelAdvancedAlgoOrder cancel unfilled algo orders (including Iceberg order, TWAP order, Trailing Stop order).
// A maximum of 10 orders can be canceled per request. Request parameters should be passed in the form of an array.
func (o *Okcoin) CancelAdvancedAlgoOrder(ctx context.Context, args []CancelAlgoOrderRequestParam) ([]AlgoOrderResponse, error) {
	if len(args) == 0 {
		return nil, errNilArgument
	}
	for a := range args {
		if args[a].InstrumentID == "" {
			return nil, errMissingInstrumentID
		}
		if args[a].AlgoOrderID == "" {
			return nil, errAlgoIDRequired
		}
	}
	var resp []AlgoOrderResponse
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, cancelAdvancedAlgoOrderEPL, http.MethodPost, typeTrade, "cancel-advance-algos", &args, &resp, true)
}

// GetAlgoOrderList retrieve a list of untriggered Algo orders under the current account.
func (o *Okcoin) GetAlgoOrderList(ctx context.Context, orderType, algoOrderID, clientOrderID, instrumentType, instrumentID, afterAlgoID, beforeAlgoID string, limit int64) ([]AlgoOrderDetail, error) {
	if orderType == "" {
		return nil, errOrderTypeRequired
	}
	params := url.Values{}
	params.Set("ordType", orderType)
	if algoOrderID != "" {
		params.Set("algoId", algoOrderID)
	}
	if clientOrderID != "" {
		params.Set("clOrdId", clientOrderID)
	}
	if instrumentType != "" {
		params.Set("instType", instrumentType)
	}
	if instrumentID != "" {
		params.Set("instId", instrumentID)
	}
	if afterAlgoID != "" {
		params.Set("after", afterAlgoID)
	}
	if beforeAlgoID != "" {
		params.Set("before", beforeAlgoID)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []AlgoOrderDetail
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, getAlgoOrderListEPL, http.MethodGet, typeTrade, common.EncodeURLValues("orders-algo-pending", params), nil, &resp, true)
}

// GetAlgoOrderHistory retrieve a list of all algo orders under the current account in the last 3 months.
func (o *Okcoin) GetAlgoOrderHistory(ctx context.Context, orderType, state, algoOrderID, instrumentType, instrumentID, afterAlgoID, beforeAlgoID string, limit int64) ([]AlgoOrderDetail, error) {
	if orderType == "" {
		return nil, errOrderTypeRequired
	}
	params := url.Values{}
	params.Set("ordType", orderType)
	if algoOrderID != "" {
		params.Set("algoId", algoOrderID)
	}
	if state != "" {
		params.Set("state", state)
	}
	if instrumentType != "" {
		params.Set("instType", instrumentType)
	}
	if instrumentID != "" {
		params.Set("instId", instrumentID)
	}
	if afterAlgoID != "" {
		params.Set("after", afterAlgoID)
	}
	if beforeAlgoID != "" {
		params.Set("before", beforeAlgoID)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []AlgoOrderDetail
	return resp, o.SendHTTPRequest(ctx, exchange.RestSpot, getAlgoOrderHistoryEPL, http.MethodGet, typeTrade, common.EncodeURLValues("orders-algo-history", params), nil, &resp, true)
}

// SendHTTPRequest sends an authenticated http request to a desired
// path with a JSON payload (of present)
// URL arguments must be in the request path and not as url.URL values
func (o *Okcoin) SendHTTPRequest(ctx context.Context, ep exchange.URL, epl request.EndpointLimit, httpMethod, requestType, requestPath string, data, result interface{}, authenticated bool) error {
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
	rType := request.AuthType(request.UnauthenticatedRequest)
	if authenticated {
		rType = request.AuthenticatedRequest
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
		path := endpoint + okcoinAPIVersion + requestType + "/" + requestPath
		headers := make(map[string]string)
		headers["Content-Type"] = "application/json"
		if authenticated {
			var creds *account.Credentials
			creds, err = o.GetCredentials(ctx)
			if err != nil {
				return nil, err
			}
			signPath := okcoinAPIVersion + requestType + "/" + requestPath

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
			Verbose:       o.Verbose,
			HTTPDebugging: o.HTTPDebugging,
			HTTPRecording: o.HTTPRecording,
		}, nil
	}

	err = o.SendPayload(ctx, epl, newRequest, rType)
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
		if resp.Message == "" {
			resp.Message = websocketErrorCodes[strconv.FormatInt(resp.Code, 10)]
		}
		return fmt.Errorf("sendHTTPRequest error - code: %d message: %s", resp.Code, resp.Message)
	}
	return nil
}

// GetFee returns an estimate of fee based on type of transaction
func (o *Okcoin) GetFee(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		rate, err := o.GetFeeRates(ctx, "SPOT", feeBuilder.Pair.String())
		if err != nil {
			return 0, err
		}
		if feeBuilder.IsMaker {
			fee = rate[0].MakerFeeRate.Float64() * feeBuilder.Amount * feeBuilder.PurchasePrice
		} else {
			fee = rate[0].TakerFeeRate.Float64() * feeBuilder.Amount * feeBuilder.PurchasePrice
		}
	case exchange.CryptocurrencyWithdrawalFee:
		var okay bool
		fee, okay = withdrawalFeeMaps[feeBuilder.Pair.Quote]
		if !okay {
			fee = getOfflineTradeFee(feeBuilder.PurchasePrice, feeBuilder.Amount)
		}
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
