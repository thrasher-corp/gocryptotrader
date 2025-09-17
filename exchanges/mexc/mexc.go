package mexc

import (
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Exchange is the overarching type across this package
type Exchange struct {
	exchange.Base
}

const (
	spotAPIURL   = "https://api.mexc.com/api/v3/"
	spotWSAPIURL = "https://api.mexc.com"

	contractAPIURL = "https://contract.mexc.com/api/v1/"
)

var (
	errInvalidSubAccountName      = errors.New("invalid sub-account name")
	errAPIKeyMissing              = errors.New("API key is required")
	errInvalidSubAccountNote      = errors.New("invalid sub-account note")
	errUnsupportedPermissionValue = errors.New("permission is unsupported")
	errAddressRequired            = errors.New("address is required")
	errNetworkNameRequired        = errors.New("network name required")
	errAccountTypeRequired        = errors.New("account type information required")
	errTransactionIDRequired      = errors.New("missing transaction ID")
	errLimitIsRequired            = errors.New("limit is required")
	errPageSizeRequired           = errors.New("page size is required")
	errPageNumberRequired         = errors.New("page number is required")
	errMissingLeverage            = errors.New("leverage is required")
	errPositionModeRequired       = errors.New("position mode is required")
)

// Start implementing public and private exchange API funcs below

// GetSymbols retrieves current exchange trading rules and symbol information
func (e *Exchange) GetSymbols(ctx context.Context, symbols []string) (*ExchangeConfig, error) {
	params := url.Values{}
	if len(symbols) > 1 {
		params.Set("symbols", strings.Join(symbols, ","))
	} else if len(symbols) == 1 {
		params.Set("symbol", symbols[0])
	}
	var resp *ExchangeConfig
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getSymbolsEPL, http.MethodGet, "exchangeInfo", params, nil, &resp)
}

// GetSystemTime check server time
func (e *Exchange) GetSystemTime(ctx context.Context) (types.Time, error) {
	resp := &struct {
		ServerTime types.Time `json:"serverTime"`
	}{}
	return resp.ServerTime, e.SendHTTPRequest(ctx, exchange.RestSpot, systemTimeEPL, http.MethodGet, "time", nil, nil, &resp)
}

// GetDefaultSumbols retrieves all default symbols
func (e *Exchange) GetDefaultSumbols(ctx context.Context) ([]string, error) {
	resp := &struct {
		Symbols []string `json:"data"`
	}{}
	return resp.Symbols, e.SendHTTPRequest(ctx, exchange.RestSpot, defaultSymbolsEPL, http.MethodGet, "defaultSymbols", nil, nil, &resp)
}

// GetOrderbook retrieves orderbook data of a symbol
func (e *Exchange) GetOrderbook(ctx context.Context, symbol string, limit int64) (*Orderbook, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *Orderbook
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, orderbooksEPL, http.MethodGet, "depth", params, nil, &resp)
}

// GetRecentTradesList retrieves recent trades list
func (e *Exchange) GetRecentTradesList(ctx context.Context, symbol string, limit int64) ([]TradeDetail, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []TradeDetail
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, recentTradesListEPL, http.MethodGet, "trades", params, nil, &resp)
}

// GetAggregatedTrades get compressed, aggregate trades. Trades that fill at the time, from the same order, with the same price will have the quantity aggregated.
func (e *Exchange) GetAggregatedTrades(ctx context.Context, symbol string, startTime, endTime time.Time, limit int64) ([]AggregatedTradeDetail, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []AggregatedTradeDetail
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, aggregatedTradesEPL, http.MethodGet, "aggTrades", params, nil, &resp)
}

var intervalToStringMap = map[string]map[kline.Interval]string{"wsIntervalToStringMap": {kline.HundredMilliseconds: "100ms", kline.TenMilliseconds: "10ms", kline.OneMin: "Min1", kline.FiveMin: "Min5", kline.FifteenMin: "Min15", kline.ThirtyMin: "Min30", kline.OneHour: "Min60", kline.FourHour: "Hour4", kline.EightHour: "Hour8", kline.OneDay: "Day1", kline.OneWeek: "Week1", kline.OneMonth: "Month1"}, "intervalToStringMap": {kline.HundredMilliseconds: "100ms", kline.TenMilliseconds: "10ms", kline.OneMin: "1m", kline.FiveMin: "5m", kline.FifteenMin: "15m", kline.ThirtyMin: "30m", kline.OneHour: "60m", kline.FourHour: "4h", kline.OneDay: "1d", kline.OneWeek: "1W", kline.OneMonth: "1M"}}

func intervalToString(interval kline.Interval, isWebsocket ...bool) (string, error) {
	var intervalString string
	var ok bool
	if len(isWebsocket) > 0 && isWebsocket[0] {
		intervalString, ok = intervalToStringMap["wsIntervalToStringMap"][interval]
	} else {
		intervalString, ok = intervalToStringMap["intervalToStringMap"][interval]
	}
	if !ok {
		return "", kline.ErrUnsupportedInterval
	}
	return intervalString, nil
}

// GetCandlestick retrieves kline/candlestick bars for a symbol.
// Klines are uniquely identified by their open time.
func (e *Exchange) GetCandlestick(ctx context.Context, symbol, interval string, startTime, endTime time.Time, limit uint64) ([]CandlestickData, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if interval == "" {
		return nil, kline.ErrUnsupportedInterval
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("interval", interval)
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []CandlestickData
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, candlestickEPL, http.MethodGet, "klines", params, nil, &resp)
}

// GetCurrentAveragePrice retrieves current average price of symbol
func (e *Exchange) GetCurrentAveragePrice(ctx context.Context, symbol string) (*SymbolAveragePrice, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *SymbolAveragePrice
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, currentAveragePriceEPL, http.MethodGet, "avgPrice", params, nil, &resp)
}

// Get24HourTickerPriceChangeStatistics retrieves ticker price change statistics
func (e *Exchange) Get24HourTickerPriceChangeStatistics(ctx context.Context, symbols []string) (TickerList, error) {
	params := url.Values{}
	if len(symbols) > 1 {
		params.Set("symbols", strings.Join(symbols, ","))
	} else if len(symbols) == 1 {
		params.Set("symbol", symbols[0])
	}
	epl := symbolsTickerPriceChangeStatEPL
	if len(symbols) == 1 {
		epl = symbolTickerPriceChangeStatEPL
	}
	var resp TickerList
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, epl, http.MethodGet, "ticker/24hr", params, nil, &resp)
}

// GetSymbolPriceTicker represents a symbol price ticker detail
func (e *Exchange) GetSymbolPriceTicker(ctx context.Context, symbols []string) ([]SymbolPriceTicker, error) {
	params := url.Values{}
	if len(symbols) > 1 {
		params.Set("symbols", strings.Join(symbols, ","))
	} else if len(symbols) == 1 {
		params.Set("symbol", symbols[0])
	}
	epl := symbolsPriceTickerEPL
	if len(symbols) == 1 {
		epl = symbolPriceTickerEPL
	}
	var resp SymbolPriceTickers
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, epl, http.MethodGet, "ticker/price", params, nil, &resp)
}

// GetSymbolOrderbookTicker represents an orderbook detail for a symbol
func (e *Exchange) GetSymbolOrderbookTicker(ctx context.Context, symbol string) ([]SymbolOrderbookTicker, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp SymbolOrderbookTickerList
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, symbolOrderbookTickerEPL, http.MethodGet, "ticker/bookTicker", params, nil, &resp)
}

// CreateSubAccount create a sub-account from the master account.
func (e *Exchange) CreateSubAccount(ctx context.Context, subAccountName, note string) (*SubAccountCreationResponse, error) {
	if subAccountName == "" {
		return nil, errInvalidSubAccountName
	}
	if note == "" {
		return nil, errInvalidSubAccountNote
	}
	params := url.Values{}
	params.Set("subAccount", subAccountName)
	params.Set("note", note)
	var resp *SubAccountCreationResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, createSubAccountEPL, http.MethodPost, "sub-account/virtualSubAccount", params, nil, &resp, true)
}

// GetSubAccountList get details of the sub-account list
func (e *Exchange) GetSubAccountList(ctx context.Context, subAccountName string, isFreeze bool, page, limit int64) (*SubAccounts, error) {
	params := url.Values{}
	if subAccountName != "" {
		params.Set("subAccount", subAccountName)
	}
	if isFreeze {
		params.Set("isFreeze", "true")
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *SubAccounts
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, subAccountListEPL, http.MethodGet, "sub-account/list", params, nil, &resp, true)
}

// CreateAPIKeyForSubAccount creates an API key for sub-account
// Permission of APIKey: SPOT_ACCOUNT_READ, SPOT_ACCOUNT_WRITE, SPOT_DEAL_READ, SPOT_DEAL_WRITE, CONTRACT_ACCOUNT_READ, CONTRACT_ACCOUNT_WRITE, CONTRACT_DEAL_READ,
// CONTRACT_DEAL_WRITE, SPOT_TRANSFER_READ, SPOT_TRANSFER_WRITE
func (e *Exchange) CreateAPIKeyForSubAccount(ctx context.Context, subAccountName, note, permissions, ip string) (*SubAccountAPIDetail, error) {
	if subAccountName == "" {
		return nil, errInvalidSubAccountName
	}
	if note == "" {
		return nil, errInvalidSubAccountNote
	}
	if permissions == "" {
		return nil, errUnsupportedPermissionValue
	}
	params := url.Values{}
	params.Set("subAccount", subAccountName)
	params.Set("note", note)
	params.Set("permissions", permissions)
	if ip != "" {
		params.Set("ip", ip)
	}
	var resp *SubAccountAPIDetail
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, createAPIKeyForSubAccountEPL, http.MethodPost, "sub-account/apiKey", params, nil, &resp, true)
}

// GetSubAccountAPIKey applies to master accounts only
func (e *Exchange) GetSubAccountAPIKey(ctx context.Context, subAccountName string) (*SubAccountsAPIs, error) {
	if subAccountName == "" {
		return nil, errInvalidSubAccountName
	}
	params := url.Values{}
	params.Set("subAccount", subAccountName)
	var resp *SubAccountsAPIs
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getSubAccountAPIKeyEPL, http.MethodGet, "sub-account/apiKey", params, nil, &resp, true)
}

// DeleteAPIKeySubAccount delete the API Key of a sub-account
func (e *Exchange) DeleteAPIKeySubAccount(ctx context.Context, subAccountName string) (string, error) {
	if subAccountName == "" {
		return "", errInvalidSubAccountName
	}
	params := url.Values{}
	params.Set("subAccount", subAccountName)
	resp := &struct {
		SubAccount string `json:"subAccount"`
	}{}
	return resp.SubAccount, e.SendHTTPRequest(ctx, exchange.RestSpot, deleteSubAccountAPIKeyEPL, http.MethodDelete, "sub-account/apiKey", params, nil, &resp, true)
}

// SubAccountUniversalTransfer requires SPOT_TRANSFER_WRITE permission
func (e *Exchange) SubAccountUniversalTransfer(ctx context.Context, fromAccount, toAccount string, fromAccountType, toAccountType asset.Item, ccy currency.Code, amount float64) (*AssetTransferResponse, error) {
	if !e.SupportsAsset(fromAccountType) {
		return nil, fmt.Errorf("%w fromAccountType %v", asset.ErrNotSupported, fromAccountType)
	}
	if !e.SupportsAsset(toAccountType) {
		return nil, fmt.Errorf("%w toAccountType %v", asset.ErrNotSupported, fromAccountType)
	}
	if ccy.IsEmpty() {
		return nil, fmt.Errorf("%w, asset %v", currency.ErrCurrencyCodeEmpty, ccy)
	}
	if amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("fromAccountType", fromAccountType.String())
	params.Set("toAccountType", toAccountType.String())
	params.Set("asset", ccy.String())
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	if fromAccount != "" {
		params.Set("fromAccount", fromAccount)
	}
	if toAccount != "" {
		params.Set("toAccount", toAccount)
	}
	var resp *AssetTransferResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, subAccountUniversalTransferEPL, http.MethodPost, "capital/sub-account/universalTransfer", params, nil, &resp, true)
}

// GetSubAccountUnversalTransferHistory retrieves universal assets transfer history of master account
func (e *Exchange) GetSubAccountUnversalTransferHistory(ctx context.Context, fromAccount, toAccount string, fromAccountType, toAccountType asset.Item, startTime, endTime time.Time, page, limit int64) (*UniversalTransferHistoryData, error) {
	if !e.SupportsAsset(fromAccountType) {
		return nil, fmt.Errorf("%w fromAccountType %v", asset.ErrNotSupported, fromAccountType)
	}
	if !e.SupportsAsset(toAccountType) {
		return nil, fmt.Errorf("%w toAccountType %v", asset.ErrNotSupported, fromAccountType)
	}
	params := url.Values{}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	params.Set("fromAccountType", fromAccountType.String())
	params.Set("toAccountType", toAccountType.String())
	if fromAccount != "" {
		params.Set("fromAccount", fromAccount)
	}
	if toAccount != "" {
		params.Set("toAccount", toAccount)
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *UniversalTransferHistoryData
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getSubaccUnversalTransfersEPL, http.MethodGet, "capital/sub-account/universalTransfer", params, resp, true)
}

// GetSubAccountAsset represents a sub-account asset balance detail
func (e *Exchange) GetSubAccountAsset(ctx context.Context, subAccount string, accountType asset.Item) (*SubAccountAssetBalances, error) {
	if subAccount == "" {
		return nil, errInvalidSubAccountName
	}
	if accountType == asset.Empty {
		return nil, asset.ErrNotSupported
	}
	params := url.Values{}
	params.Set("subAccount", subAccount)
	params.Set("accountType", accountType.String())
	var resp *SubAccountAssetBalances
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getSubAccountAssetEPL, http.MethodGet, "sub-account/asset", params, nil, &resp, true)
}

// GetKYCStatus retrieves accounts KYC(know your customer) status
func (e *Exchange) GetKYCStatus(ctx context.Context) (*KYCStatusInfo, error) {
	var resp *KYCStatusInfo
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getKYCStatusEPL, http.MethodGet, "kyc/status", nil, nil, &resp, true)
}

// UseAPIDefaultSymbols retrieves a default user API symbols
func (e *Exchange) UseAPIDefaultSymbols(ctx context.Context) ([]string, error) {
	resp := &struct {
		Data []string `json:"data"`
	}{}
	return resp.Data, e.SendHTTPRequest(ctx, exchange.RestSpot, selfSymbolsEPL, http.MethodGet, "selfSymbols", nil, nil, &resp, true)
}

// GetCurrencyInformation get currency information
func (e *Exchange) GetCurrencyInformation(ctx context.Context) ([]CurrencyInformation, error) {
	var resp []CurrencyInformation
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getCurrencyInformationEPL, http.MethodGet, "capital/config/getall", nil, nil, &resp, true)
}

// WithdrawCapital withdraws an asset through chains
func (e *Exchange) WithdrawCapital(ctx context.Context, amount float64, coin currency.Code, withdrawID, network, contractAddress, address, memo, remark string) (*IDResponse, error) {
	if coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if address == "" {
		return nil, fmt.Errorf("%w, withdrawal address 'address' is unset", errAddressRequired)
	}
	if amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("coin", coin.String())
	params.Set("address", address)
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	if withdrawID != "" {
		params.Set("withdrawOrderId", withdrawID)
	}
	if network != "" {
		params.Set("netWork", network)
	}
	if contractAddress != "" {
		params.Set("contractAddress", contractAddress)
	}
	if memo != "" {
		params.Set("memo", memo)
	}
	if remark != "" {
		params.Set("remark", remark)
	}
	var resp *IDResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, withdrawCapitalEPL, http.MethodPost, "capital/withdraw", params, nil, &resp, true)
}

// CancelWithdrawal cancels an pending withdrawal order
func (e *Exchange) CancelWithdrawal(ctx context.Context, id string) (*IDResponse, error) {
	if id == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("id", id)
	var resp *IDResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, cancelWithdrawalEPL, http.MethodDelete, "capital/withdraw", params, nil, &resp, true)
}

// GetFundDepositHistory retrieves a list of fund deposit transaction details
func (e *Exchange) GetFundDepositHistory(ctx context.Context, coin currency.Code, status string, startTime, endTime time.Time, limit int64) ([]FundDepositInfo, error) {
	params := url.Values{}
	if !coin.IsEmpty() {
		params.Set("coin", coin.String())
	}
	if status != "" {
		params.Set("status", status)
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []FundDepositInfo
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getFundDepositHistoryEPL, http.MethodGet, "capital/deposit/hisrec", params, nil, &resp, true)
}

// GetWithdrawalHistory represents currency withdrawal history possible values of withdraw status
// 1:APPLY,2:AUDITING,3:WAIT,4:PROCESSING,5:WAIT_PACKAGING,6:WAIT_CONFIRM,7:SUCCESS,8:FAILED,9:CANCEL,10:MANUAL
func (e *Exchange) GetWithdrawalHistory(ctx context.Context, coin currency.Code, startTime, endTime time.Time, status, limit int64) ([]WithdrawalInfo, error) {
	params := url.Values{}
	if !coin.IsEmpty() {
		params.Set("coin", coin.String())
	}
	if status != 0 {
		params.Set("status", strconv.FormatInt(status, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []WithdrawalInfo
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getWithdrawalHistoryEPL, http.MethodGet, "capital/withdraw/history", params, nil, &resp, true)
}

// GenerateDepositAddress generates a deposit address given the currency code and network name
func (e *Exchange) GenerateDepositAddress(ctx context.Context, coin currency.Code, network string) ([]DepositAddressInfo, error) {
	if coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if network == "" {
		return nil, errNetworkNameRequired
	}
	params := url.Values{}
	params.Set("coin", coin.String())
	params.Set("network", network)
	var resp []DepositAddressInfo
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, generateDepositAddressEPL, http.MethodPost, "capital/deposit/address", params, nil, &resp, true)
}

// GetDepositAddressOfCoin retrieves a deposit address detail of an asset
func (e *Exchange) GetDepositAddressOfCoin(ctx context.Context, coin currency.Code, network string) ([]DepositAddressInfo, error) {
	if coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("coin", coin.String())
	if network != "" {
		params.Set("network", network)
	}
	var resp []DepositAddressInfo
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getDepositAddressEPL, http.MethodGet, "capital/deposit/address", params, nil, &resp, true)
}

// GetWithdrawalAddress retrieves a list of previously used deposit addresses
func (e *Exchange) GetWithdrawalAddress(ctx context.Context, coin currency.Code, page, limit int64) (*WithdrawalAddressesDetail, error) {
	params := url.Values{}
	if !coin.IsEmpty() {
		params.Set("coin", coin.String())
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *WithdrawalAddressesDetail
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getWithdrawalAddressEPL, http.MethodGet, "capital/withdraw/address", params, nil, &resp, true)
}

// UserUniversalTransfer transfers an asset transfer between account types of same account
func (e *Exchange) UserUniversalTransfer(ctx context.Context, fromAccountType, toAccountType string, ccy currency.Code, amount float64) ([]UserUniversalTransferResponse, error) {
	if fromAccountType == "" {
		return nil, fmt.Errorf("%w, fromAccountType is required", errAccountTypeRequired)
	}
	if toAccountType == "" {
		return nil, fmt.Errorf("%w, toAccountType is required", errAccountTypeRequired)
	}
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	params.Set("fromAccountType", fromAccountType)
	params.Set("toAccountType", toAccountType)
	params.Set("asset", ccy.String())
	var resp []UserUniversalTransferResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, userUniversalTransferEPL, http.MethodPost, "capital/transfer", params, nil, &resp, true)
}

// GetUniversalTransferHistory retrieves users universal asset transfer history
func (e *Exchange) GetUniversalTransferHistory(ctx context.Context, fromAccountType, toAccountType string, startTime, endTime time.Time, page, size int64) (*UniversalTransferHistoryResponse, error) {
	if fromAccountType == "" {
		return nil, fmt.Errorf("%w, fromAccountType is required", errAccountTypeRequired)
	}
	if toAccountType == "" {
		return nil, fmt.Errorf("%w, toAccountType is required", errAccountTypeRequired)
	}
	params := url.Values{}
	params.Set("fromAccountType", fromAccountType)
	params.Set("toAccountType", toAccountType)
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if size > 0 {
		params.Set("size", strconv.FormatInt(size, 10))
	}

	var resp *UniversalTransferHistoryResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getUniversalTransferhistoryEPL, http.MethodGet, "capital/transfer", params, nil, &resp, true)
}

// GetUniversalTransferDetailByID retrieves a universal asset transfer history item detail
func (e *Exchange) GetUniversalTransferDetailByID(ctx context.Context, transactionID string) (*UniversalTransferHistoryData, error) {
	if transactionID == "" {
		return nil, errTransactionIDRequired
	}
	params := url.Values{}
	params.Set("tranId", transactionID)
	var resp *UniversalTransferHistoryData
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getUniversalTransferDetailByIDEPL, http.MethodGet, "capital/transfer/tranId", params, nil, &resp, true)
}

// GetAssetThatCanBeConvertedintoMX represents an asset that can be converted into an MX asset
func (e *Exchange) GetAssetThatCanBeConvertedintoMX(ctx context.Context) ([]AssetConvertableToMX, error) {
	var resp []AssetConvertableToMX
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getAssetConvertedMXEPL, http.MethodGet, "capital/convert/list", nil, nil, &resp, true)
}

// DustTransfer transfer near-worthless crypto assets whose value is smaller than transaction fees
func (e *Exchange) DustTransfer(ctx context.Context, assets []currency.Code) (*DustConvertResponse, error) {
	if len(assets) == 0 {
		return nil, fmt.Errorf("%w: at least one asset must be specified", currency.ErrCurrencyCodeEmpty)
	}
	assetsString := ""
	for a := range assets {
		if assets[a].IsEmpty() {
			return nil, currency.ErrCurrencyCodeEmpty
		}
		assetsString += assets[a].String() + ","
	}
	assetsString = strings.Trim(assetsString, ",")
	params := url.Values{}
	params.Set("asset", assetsString)
	var resp *DustConvertResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, dustTransferEPL, http.MethodPost, "capital/convert", params, nil, &resp, true)
}

// DustLog retrieves a dust conversion history
func (e *Exchange) DustLog(ctx context.Context, startTime, endTime time.Time, page, limit int64) (*DustLogDetail, error) {
	params := url.Values{}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if limit <= 0 {
		return nil, errLimitIsRequired
	}
	params.Set("limit", strconv.FormatInt(limit, 10))
	var resp *DustLogDetail
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, dustLogEPL, http.MethodGet, "capital/convert", params, nil, &resp, true)
}

// InternalTransfer allows an internal asset transfer between assets.
func (e *Exchange) InternalTransfer(ctx context.Context, toAccountType, toAccount, areaCode string, ccy currency.Code, amount float64) (*AssetTransferResponse, error) {
	if toAccountType == "" {
		return nil, fmt.Errorf("%w: toAccountType is required", errAccountTypeRequired)
	}
	if toAccount == "" {
		return nil, fmt.Errorf("%w: toAccount is required", errAddressRequired)
	}
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("toAccountType", toAccountType)
	params.Set("toAccount", toAccount)
	params.Set("asset", ccy.String())
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	if areaCode != "" {
		params.Set("areaCode", areaCode)
	}
	var resp *AssetTransferResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, internalTransferEPL, http.MethodPost, "capital/transfer/internal", params, nil, &resp, true)
}

// GetInternalTransferHistory retrieves an internal asset transfer history
func (e *Exchange) GetInternalTransferHistory(ctx context.Context, transferID string, startTime, endTime time.Time, page, limit int64) (*InternalTransferDetail, error) {
	params := url.Values{}
	if transferID != "" {
		params.Set("tranId", transferID)
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	var resp *InternalTransferDetail
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getInternalTransferHistoryEPL, http.MethodGet, "capital/transfer/internal", params, nil, &resp, true)
}

// CapitalWithdrawal withdraws an asset through a network
func (e *Exchange) CapitalWithdrawal(ctx context.Context, coin currency.Code, withdrawOrderID, network, address, memo, remark string, amount float64) ([]IDResponse, error) {
	if coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if address == "" {
		return nil, errAddressRequired
	}
	if amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("coin", coin.String())
	params.Set("address", address)
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	if withdrawOrderID != "" {
		params.Set("withdrawOrderId", withdrawOrderID)
	}
	if network != "" {
		params.Set("network", network)
	}
	if memo != "" {
		params.Set("memo", memo)
	}
	if remark != "" {
		params.Set("remark", remark)
	}
	var resp []IDResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, capitalWithdrawalEPL, http.MethodPost, "capital/withdraw/apply", params, nil, &resp, true)
}

// NewTestOrder creates and validates a new order but does not send it into the matching engine.
func (e *Exchange) NewTestOrder(ctx context.Context, symbol, newClientOrderID, side, orderType string, quantity, quoteOrderQty, price float64) (*OrderDetail, error) {
	return e.newOrder(ctx, symbol, newClientOrderID, side, orderType, "order/test", quantity, quoteOrderQty, price)
}

// SpotOrderStringFromOrderTypeAndTimeInForce returns an order type string from order.Type and order.TimeInForce instance.
func SpotOrderStringFromOrderTypeAndTimeInForce(oType order.Type, tif order.TimeInForce) (string, error) {
	switch oType {
	case order.Limit:
		if tif == order.PostOnly {
			return typeLimitMaker, nil
		}
		return typeLimit, nil
	case order.Market:
		switch tif {
		case order.ImmediateOrCancel:
			return typeImmediateOrCancel, nil
		case order.FillOrKill:
			return typeFillOrKill, nil
		}
		return typeMarket, nil
	default:
		switch tif {
		case order.PostOnly:
			return typeLimitMaker, nil
		case order.ImmediateOrCancel:
			return typeImmediateOrCancel, nil
		case order.FillOrKill:
			return typeFillOrKill, nil
		default:
			return "", order.ErrTypeIsInvalid
		}
	}
}

// NewOrder creates a new order
func (e *Exchange) NewOrder(ctx context.Context, symbol, newClientOrderID, side, orderType string, quantity, quoteOrderQty, price float64) (*OrderDetail, error) {
	return e.newOrder(ctx, symbol, newClientOrderID, side, orderType, "order", quantity, quoteOrderQty, price)
}

func (e *Exchange) newOrder(ctx context.Context, symbol, newClientOrderID, side, orderType, path string, quantity, quoteOrderQty, price float64) (*OrderDetail, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if orderType == "" {
		return nil, order.ErrTypeIsInvalid
	}
	orderType = strings.ToUpper(orderType)
	switch orderType {
	case typeLimit, typeLimitMaker:
		if quantity <= 0 {
			return nil, fmt.Errorf("%w, quantity %v", limits.ErrAmountBelowMin, quantity)
		}
		if price <= 0 {
			return nil, fmt.Errorf("%w, price %v", limits.ErrPriceBelowMin, price)
		}
	case typeMarket, typeImmediateOrCancel, typeFillOrKill:
		if quantity <= 0 && quoteOrderQty <= 0 {
			return nil, fmt.Errorf("%w, either quantity or quote order quantity must be filled", limits.ErrAmountBelowMin)
		}
	default:
		return nil, fmt.Errorf("%w, order type %s", order.ErrUnsupportedOrderType, orderType)
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("side", side)
	params.Set("type", orderType)
	if quantity > 0 {
		params.Set("quantity", strconv.FormatFloat(quantity, 'f', -1, 64))
	}
	if quoteOrderQty > 0 {
		params.Set("quoteOrderQty", strconv.FormatFloat(quoteOrderQty, 'f', -1, 64))
	}
	if price != 0 {
		params.Set("price", strconv.FormatFloat(price, 'f', -1, 64))
	}
	if newClientOrderID != "" {
		params.Set("newClientOrderId", newClientOrderID)
	}
	var resp *OrderDetail
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, newOrderEPL, http.MethodPost, path, params, nil, &resp, true)
}

// OrderTypeStringFromOrderTypeAndTimeInForce returns a string representation of an order.Type instance.
func (e *Exchange) OrderTypeStringFromOrderTypeAndTimeInForce(oType order.Type, tif order.TimeInForce) (string, error) {
	switch oType {
	case order.Limit:
		if tif == order.PostOnly {
			return typePostOnly, nil
		}
		return typeLimit, nil
	case order.Market:
		switch tif {
		case order.ImmediateOrCancel:
			return typeImmediateOrCancel, nil
		case order.FillOrKill:
			return typeFillOrKill, nil
		}
		return typeMarket, nil
	case order.StopLimit:
		return typeStopLimit, nil
	case order.UnknownType:
		switch tif {
		case order.PostOnly:
			return typePostOnly, nil
		case order.ImmediateOrCancel:
			return typeImmediateOrCancel, nil
		case order.FillOrKill:
			return typeFillOrKill, nil
		}
	}
	return "", fmt.Errorf("%w %w", order.ErrUnsupportedTimeInForce, order.ErrUnsupportedOrderType)
}

// StringToOrderTypeAndTimeInForce returns an order type from string
func (e *Exchange) StringToOrderTypeAndTimeInForce(oType string) (order.Type, order.TimeInForce, error) {
	switch oType {
	case typeLimit:
		return order.Limit, order.GoodTillCancel, nil
	case typeMarket:
		return order.Market, order.UnknownTIF, nil
	case typeLimitMaker:
		return order.Limit, order.PostOnly, nil
	case typePostOnly:
		return order.Limit, order.PostOnly, nil
	case typeImmediateOrCancel:
		return order.Market, order.ImmediateOrCancel, nil
	case typeFillOrKill:
		return order.Market, order.FillOrKill, nil
	case typeStopLimit:
		return order.StopLimit, order.UnknownTIF, nil
	default:
		return order.UnknownType, order.UnknownTIF, order.ErrUnsupportedOrderType
	}
}

// CreateBatchOrder creates utmost 30 orders with a same symbol in a batch,rate limit:2 times/s.
func (e *Exchange) CreateBatchOrder(ctx context.Context, args []BatchOrderCreationParam) ([]OrderDetail, error) {
	if len(args) == 0 {
		return nil, common.ErrEmptyParams
	}
	for a := range args {
		if args[a] == (BatchOrderCreationParam{}) {
			return nil, common.ErrEmptyParams
		}
		if args[a].Symbol == "" {
			return nil, currency.ErrSymbolStringEmpty
		}
		if args[a].Side == "" {
			return nil, order.ErrSideIsInvalid
		}
		args[a].OrderType = strings.ToUpper(args[a].OrderType)
		switch args[a].OrderType {
		case typeLimit:
			if args[a].Quantity <= 0 {
				return nil, fmt.Errorf("%w, quantity %v", limits.ErrAmountBelowMin, args[a].Quantity)
			}
			if args[a].Price <= 0 {
				return nil, fmt.Errorf("%w, price %v", limits.ErrPriceBelowMin, args[a].Price)
			}
		case typeMarket:
			if args[a].Quantity <= 0 && args[a].QuoteOrderQty <= 0 {
				return nil, fmt.Errorf("%w, either quantity or quote order quantity must be filled", limits.ErrAmountBelowMin)
			}
		default:
			return nil, fmt.Errorf("%w, order type %s", order.ErrUnsupportedOrderType, args[a].OrderType)
		}
	}
	jsonString, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("batchOrders", string(jsonString))
	var resp []OrderDetail
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, createBatchOrdersEPL, http.MethodPost, "batchOrders", params, nil, &resp, true)
}

// CancelTradeOrder cancels an order
func (e *Exchange) CancelTradeOrder(ctx context.Context, symbol, orderID, clientOrderID, newClientOrderID string) (*OrderDetail, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if orderID == "" && clientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if clientOrderID != "" {
		params.Set("origClientOrderId", clientOrderID)
	}
	if newClientOrderID != "" {
		params.Set("newClientOrderId", newClientOrderID)
	}
	var resp *OrderDetail
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, cancelTradeOrderEPL, http.MethodDelete, "order", params, nil, &resp, true)
}

// CancelAllOpenOrdersBySymbol cancel all pending orders for a single symbol, including OCO pending orders.
func (e *Exchange) CancelAllOpenOrdersBySymbol(ctx context.Context, symbol string) ([]OrderDetail, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp []OrderDetail
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, cancelAllOpenOrdersBySymbolEPL, http.MethodDelete, "openOrders", params, nil, &resp, true)
}

// GetOrderByID retrieves a single order
func (e *Exchange) GetOrderByID(ctx context.Context, symbol, clientOrderID, orderID string) (*OrderDetail, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if clientOrderID == "" && orderID == "" {
		return nil, order.ErrClientOrderIDMustBeSet
	}
	if clientOrderID != "" {
		params.Set("origClientOrderId", clientOrderID)
	}
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	var resp *OrderDetail
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getOrderByIDEPL, http.MethodGet, "order", params, nil, &resp, true)
}

// GetOpenOrders retrieves all open orders on a symbol. Careful when accessing this with no symbol.
func (e *Exchange) GetOpenOrders(ctx context.Context, symbol string) ([]OrderDetail, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp []OrderDetail
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getOpenOrdersEPL, http.MethodGet, "openOrders", params, nil, &resp, true)
}

// GetAllOrders retrieves all account orders including active, cancelled or completed orders(the query period is the latest 24 hours by default).
// You can query a maximum of the latest 7 days.
func (e *Exchange) GetAllOrders(ctx context.Context, symbol string, startTime, endTime time.Time, limit int64) ([]OrderDetail, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []OrderDetail
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, allOrdersEPL, http.MethodGet, "allOrders", params, nil, &resp, true)
}

// GetAccountInformation retrieves current account information,rate limit:2 times/s.
func (e *Exchange) GetAccountInformation(ctx context.Context) (*AccountDetail, error) {
	var resp *AccountDetail
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, accountInformationEPL, http.MethodGet, "account", nil, nil, &resp, true)
}

// GetAccountTradeList retrieves trades for a specific account and symbol,Only the transaction records in the past 1 month can be queried.
func (e *Exchange) GetAccountTradeList(ctx context.Context, symbol, orderID string, startTime, endTime time.Time, limit int64) ([]AccountTrade, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []AccountTrade
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, accountTradeListEPL, http.MethodGet, "myTrades", params, nil, &resp, true)
}

// EnableMXDeduct enable or disable MX deduct for spot commission fee
func (e *Exchange) EnableMXDeduct(ctx context.Context, mxDeductEnable bool) (*MXDeductResponse, error) {
	params := url.Values{}
	if mxDeductEnable {
		params.Set("mxDeductEnable", "true")
	} else {
		params.Set("mxDeductEnable", "false")
	}
	var resp *MXDeductResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, enableMXDeductEPL, http.MethodPost, "mxDeduct/enable", params, nil, &resp, true)
}

// GetMXDeductStatus retrieves MX deduct status detail
func (e *Exchange) GetMXDeductStatus(ctx context.Context) (*MXDeductResponse, error) {
	var resp *MXDeductResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getMXDeductStatusEPL, http.MethodGet, "mxDeduct/enable", nil, nil, &resp, true)
}

// GetSymbolTradingFee retrieves symbol commissions
func (e *Exchange) GetSymbolTradingFee(ctx context.Context, symbol string) (*SymbolCommissionFee, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *SymbolCommissionFee
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getSymbolTradingFeeEPL, http.MethodGet, "tradeFee", params, nil, &resp, true)
}

// GetRebateHistoryRecords retrieves a rebate history record
func (e *Exchange) GetRebateHistoryRecords(ctx context.Context, startTime, endTime time.Time, page int64) (*RebateHistory, error) {
	params := url.Values{}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	var resp *RebateHistory
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getUserRebateHistoryEPL, http.MethodGet, "rebate/taxQuery", params, nil, &resp, true)
}

// GetRebateRecordsDetail retrieves a rebate record detail
func (e *Exchange) GetRebateRecordsDetail(ctx context.Context, startTime, endTime time.Time, page int64) (*RebateRecordDetail, error) {
	params := url.Values{}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	var resp *RebateRecordDetail
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getRebateRecordsDetailEPL, http.MethodGet, "rebate/detail", params, nil, &resp, true)
}

// GetSelfRebateRecordsDetail retrieves self rebate records details
func (e *Exchange) GetSelfRebateRecordsDetail(ctx context.Context, startTime, endTime time.Time, page int64) (*RebateRecordDetail, error) {
	params := url.Values{}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	var resp *RebateRecordDetail
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, selfRebateRecordsDetailsEPL, http.MethodGet, "rebate/detail/kickback", params, nil, &resp, true)
}

// GetReferCode retrieves refer code
func (e *Exchange) GetReferCode(ctx context.Context) (*ReferCode, error) {
	var resp *ReferCode
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getReferCodeEPL, http.MethodGet, "rebate/referCode", nil, nil, &resp, true)
}

// GetAffiliateCommissionRecord retrieves affiliate commission record
func (e *Exchange) GetAffiliateCommissionRecord(ctx context.Context, startTime, endTime time.Time, inviteCode string, page, pageSize int64) (*AffiliateCommissionRecord, error) {
	params := url.Values{}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if inviteCode != "" {
		params.Set("inviteCode", inviteCode)
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if pageSize > 0 {
		params.Set("pageSize", strconv.FormatInt(pageSize, 10))
	}
	var resp *AffiliateCommissionRecord
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getAffilateCommissionRecordEPL, http.MethodGet, "rebate/affiliate/commission", params, nil, &resp, true)
}

// GetAffiliateWithdrawRecord retrieves affiliate withdrawal records
func (e *Exchange) GetAffiliateWithdrawRecord(ctx context.Context, startTime, endTime time.Time, page, pageSize int64) (*AffiliateWithdrawRecords, error) {
	params := url.Values{}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if pageSize > 0 {
		params.Set("pageSize", strconv.FormatInt(pageSize, 10))
	}
	var resp *AffiliateWithdrawRecords
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getAffilateWithdrawRecordEPL, http.MethodGet, "rebate/affiliate/withdraw", params, nil, &resp, true)
}

// GetAffiliateCommissionDetailRecord retrieves an affiliate commission detail record
// Commission type possible values: '1':spot,'2':futures, and '3':ETF
func (e *Exchange) GetAffiliateCommissionDetailRecord(ctx context.Context, startTime, endTime time.Time, inviteCode, commissionType string, page, pageSize int64) (*RebateAffiliateCommissionDetail, error) {
	params := url.Values{}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if inviteCode != "" {
		params.Set("inviteCode", inviteCode)
	}
	if commissionType != "" {
		params.Set("commissionType", commissionType)
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if pageSize > 0 {
		params.Set("pageSize", strconv.FormatInt(pageSize, 10))
	}
	var resp *RebateAffiliateCommissionDetail
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getAffiliateConnissionDetailEPL, http.MethodGet, "rebate/affiliate/commission/detail", params, nil, &resp, true)
}

// GetAffiliateCampaignData retrieves an affiliate campaign data
func (e *Exchange) GetAffiliateCampaignData(ctx context.Context, startTime, endTime time.Time, page, pageSize int64) (*AffiliateCampaignData, error) {
	params := url.Values{}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if pageSize > 0 {
		params.Set("pageSize", strconv.FormatInt(pageSize, 10))
	}
	var resp *AffiliateCampaignData
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, affiliateCampaignDataEPL, http.MethodGet, "rebate/affiliate/campaign", params, nil, &resp, true)
}

// GetAffiliateReferralData retrieves an affiliate referral data
func (e *Exchange) GetAffiliateReferralData(ctx context.Context, startTime, endTime time.Time, uid, inviteCode string, page, pageSize int64) (*AffiliateReferralData, error) {
	params := url.Values{}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if inviteCode != "" {
		params.Set("inviteCode", inviteCode)
	}
	if uid != "" {
		params.Set("uid", uid)
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if pageSize > 0 {
		params.Set("pageSize", strconv.FormatInt(pageSize, 10))
	}
	var resp *AffiliateReferralData
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, affiliateReferralDataEPL, http.MethodGet, "rebate/affiliate/referral", params, nil, &resp, true)
}

// GetSubAffiliateData retrieve a sub-affiliate data
func (e *Exchange) GetSubAffiliateData(ctx context.Context, startTime, endTime time.Time, inviteCode string, page, pageSize int64) (*SubAffiliateData, error) {
	params := url.Values{}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if inviteCode != "" {
		params.Set("inviteCode", inviteCode)
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if pageSize > 0 {
		params.Set("pageSize", strconv.FormatInt(pageSize, 10))
	}
	var resp *SubAffiliateData
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, subAffiliateDataEPL, http.MethodGet, "rebate/affiliate/subaffiliates", params, nil, &resp, true)
}

// GenerateListenKey starts a new data stream. The stream will close 60 minutes after creation unless a keepalive is sent.
func (e *Exchange) GenerateListenKey(ctx context.Context) (string, error) {
	resp := &struct {
		ListenKey string `json:"listenKey"`
	}{}
	return resp.ListenKey, e.SendHTTPRequest(ctx, exchange.RestSpot, request.Auth, http.MethodPost, "userDataStream", nil, nil, &resp, true)
}

// SendHTTPRequest sends an http request to a desired path with a JSON payload (of present)
func (e *Exchange) SendHTTPRequest(ctx context.Context, ep exchange.URL, epl request.EndpointLimit, method, requestPath string, values url.Values, arg, result interface{}, auth ...bool) error {
	ePoint, err := e.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	headers := make(map[string]string)
	headers["Content-Type"] = "application/json"
	var authType request.AuthType
	authType = request.UnauthenticatedRequest
	if len(auth) > 0 && auth[0] {
		authType = request.AuthenticatedRequest
		creds, err := e.GetCredentials(ctx)
		if err != nil {
			return err
		}
		headers["X-MEXC-APIKEY"] = creds.Key
		if values == nil {
			values = url.Values{}
		}
		values.Set("recvWindow", "5000")
		values.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
		hmac, err := crypto.GetHMAC(crypto.HashSHA256,
			[]byte(values.Encode()),
			[]byte(creds.Secret))
		if err != nil {
			return err
		}
		values.Set("signature", base64.StdEncoding.EncodeToString(hmac))
	}
	var payload string
	if arg != nil {
		var byteData []byte
		byteData, err = json.Marshal(arg)
		if err != nil {
			return err
		}
		payload = string(byteData)
	}
	err = e.SendPayload(ctx, epl, func() (*request.Item, error) {
		return &request.Item{
			Method:        method,
			Path:          ePoint + common.EncodeURLValues(requestPath, values),
			Headers:       headers,
			Body:          strings.NewReader(payload),
			Result:        result,
			NonceEnabled:  true,
			Verbose:       e.Verbose,
			HTTPDebugging: e.HTTPDebugging,
			HTTPRecording: e.HTTPRecording,
		}, nil
	}, authType)
	if err != nil && len(auth) > 0 && auth[0] {
		return fmt.Errorf("%w %v", request.ErrAuthRequestFailed, err)
	}
	return err
}
