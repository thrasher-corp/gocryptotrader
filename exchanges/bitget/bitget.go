package bitget

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Bitget is the overarching type across this package
type Bitget struct {
	exchange.Base
}

const (
	bitgetAPIURL = "https://api.bitget.com/api/v2/"

	// Public endpoints
	bitgetPublic         = "public/"
	bitgetAnnouncements  = "annoucements" // sic
	bitgetTime           = "time"
	bitgetCoins          = "coins"
	bitgetSymbols        = "symbols"
	bitgetMarket         = "market/"
	bitgetVIPFeeRate     = "vip-fee-rate"
	bitgetTickers        = "tickers"
	bitgetMergeDepth     = "merge-depth"
	bitgetOrderbook      = "orderbook"
	bitgetCandles        = "candles"
	bitgetHistoryCandles = "history-candles"
	bitgetFills          = "fills"
	bitgetFillsHistory   = "fills-history"

	// Mixed endpoints
	bitgetSpot = "spot/"

	// Authenticated endpoints
	bitgetCommon               = "common/"
	bitgetTradeRate            = "trade-rate"
	bitgetTax                  = "tax/"
	bitgetSpotRecord           = "spot-record"
	bitgetFutureRecord         = "future-record"
	bitgetMarginRecord         = "margin-record"
	bitgetP2PRecord            = "p2p-record"
	bitgetP2P                  = "p2p/"
	bitgetMerchantList         = "merchantList"
	bitgetMerchantInfo         = "merchantInfo"
	bitgetOrderList            = "orderList"
	bitgetAdvList              = "advList"
	bitgetUser                 = "user/"
	bitgetCreate               = "create-"
	bitgetVirtualSubaccount    = "virtual-subaccount"
	bitgetModify               = "modify-"
	bitgetBatchCreateSubAccApi = "batch-create-subaccount-and-apikey"
	bitgetList                 = "list"
	bitgetAPIKey               = "apikey"
	bitgetConvert              = "convert/"
	bitgetCurrencies           = "currencies"
	bitgetQuotedPrice          = "quoted-price"
	bitgetTrade                = "trade"
	bitgetConvertRecord        = "convert-record"
	bitgetBGBConvert           = "bgb-convert"
	bitgetConvertCoinList      = "bgb-convert-coin-list"
	bitgetBGBConvertRecords    = "bgb-convert-records"
	bitgetPlaceOrder           = "/place-order"
	bitgetCancelOrder          = "/cancel-order"
	bitgetBatchOrders          = "/batch-orders"

	// Errors
	errUnknownEndpointLimit = "unknown endpoint limit %v"
)

var (
	errBusinessTypeEmpty     = errors.New("businessType cannot be empty")
	errPairEmpty             = errors.New("currency pair cannot be empty")
	errCurrencyEmpty         = errors.New("currency cannot be empty")
	errProductTypeEmpty      = errors.New("productType cannot be empty")
	errSubAccountEmpty       = errors.New("subaccounts cannot be empty")
	errNewStatusEmpty        = errors.New("newStatus cannot be empty")
	errNewPermsEmpty         = errors.New("newPerms cannot be empty")
	errPassphraseEmpty       = errors.New("passphrase cannot be empty")
	errLabelEmpty            = errors.New("label cannot be empty")
	errAPIKeyEmpty           = errors.New("apiKey cannot be empty")
	errFromToMutex           = errors.New("exactly one of fromAmount and toAmount must be set")
	errTraceIDEmpty          = errors.New("traceID cannot be empty")
	errAmountEmpty           = errors.New("amount cannot be empty")
	errPriceEmpty            = errors.New("price cannot be empty")
	errTypeAssertTimestamp   = errors.New("unable to type assert timestamp")
	errTypeAssertOpenPrice   = errors.New("unable to type assert opening price")
	errTypeAssertHighPrice   = errors.New("unable to type assert high price")
	errTypeAssertLowPrice    = errors.New("unable to type assert low price")
	errTypeAssertClosePrice  = errors.New("unable to type assert close price")
	errTypeAssertBaseVolume  = errors.New("unable to type assert base volume")
	errTypeAssertQuoteVolume = errors.New("unable to type assert quote volume")
	errTypeAssertUSDTVolume  = errors.New("unable to type assert USDT volume")
	errGranEmpty             = errors.New("granularity cannot be empty")
	errEndTimeEmpty          = errors.New("endTime cannot be empty")
	errSideEmpty             = errors.New("side cannot be empty")
	errOrderTypeEmpty        = errors.New("orderType cannot be empty")
	errStrategyEmpty         = errors.New("strategy cannot be empty")
	errLimitPriceEmpty       = errors.New("price cannot be empty for limit orders")
	errOrderClientEmpty      = errors.New("at least one of orderID and clientOrderID must not be empty")
	errOrdersEmpty           = errors.New("orders cannot be empty")
)

// QueryAnnouncement returns announcements from the exchange, filtered by type and time
func (bi *Bitget) QueryAnnouncements(ctx context.Context, annType string, startTime, endTime time.Time) (*AnnResp, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, true)
	if err != nil {
		return nil, err
	}
	params.Values.Set("annType", annType)
	params.Values.Set("language", "en_US")
	var resp *AnnResp
	return resp, bi.SendHTTPRequest(ctx, exchange.RestSpot, Rate20, bitgetPublic+bitgetAnnouncements, params.Values,
		&resp)
}

// GetTime returns the server's time
func (bi *Bitget) GetTime(ctx context.Context) (*TimeResp, error) {
	var resp *TimeResp
	return resp, bi.SendHTTPRequest(ctx, exchange.RestSpot, Rate20, bitgetPublic+bitgetTime, nil, &resp)
}

// GetTradeRate returns the fees the user would face for trading a given symbol
func (bi *Bitget) GetTradeRate(ctx context.Context, pair, businessType string) (*TradeRateResp, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	if businessType == "" {
		return nil, errBusinessTypeEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair)
	vals.Set("businessType", businessType)
	var resp *TradeRateResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet,
		bitgetCommon+bitgetTradeRate, vals, nil, &resp)
}

// GetSpotTransactionRecords returns the user's spot transaction records
func (bi *Bitget) GetSpotTransactionRecords(ctx context.Context, currency string, startTime, endTime time.Time, limit, pagination int64) (*SpotTrResp, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, false)
	if err != nil {
		return nil, err
	}
	params.Values.Set("coin", currency)
	params.Values.Set("limit", strconv.FormatInt(limit, 10))
	params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	var resp *SpotTrResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate1, http.MethodGet,
		bitgetTax+bitgetSpotRecord, params.Values, nil, &resp)
}

// GetFuturesTransactionRecords returns the user's futures transaction records
func (bi *Bitget) GetFuturesTransactionRecords(ctx context.Context, productType, currency string, startTime, endTime time.Time, limit, pagination int64) (*FutureTrResp, error) {
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, false)
	if err != nil {
		return nil, err
	}
	params.Values.Set("productType", productType)
	params.Values.Set("marginCoin", currency)
	params.Values.Set("limit", strconv.FormatInt(limit, 10))
	params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	var resp *FutureTrResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate1, http.MethodGet,
		bitgetTax+bitgetFutureRecord, params.Values, nil, &resp)
}

// GetMarginTransactionRecords returns the user's margin transaction records
func (bi *Bitget) GetMarginTransactionRecords(ctx context.Context, marginType, currency string, startTime, endTime time.Time, limit, pagination int64) (*MarginTrResp, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, false)
	if err != nil {
		return nil, err
	}
	params.Values.Set("marginType", marginType)
	params.Values.Set("coin", currency)
	params.Values.Set("limit", strconv.FormatInt(limit, 10))
	params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	var resp *MarginTrResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate1, http.MethodGet,
		bitgetTax+bitgetMarginRecord, params.Values, nil, &resp)
}

// GetP2PTransactionRecords returns the user's P2P transaction records
func (bi *Bitget) GetP2PTransactionRecords(ctx context.Context, currency string, startTime, endTime time.Time, limit, pagination int64) (*P2PTrResp, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, false)
	if err != nil {
		return nil, err
	}
	params.Values.Set("coin", currency)
	params.Values.Set("limit", strconv.FormatInt(limit, 10))
	params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	var resp *P2PTrResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate1, http.MethodGet,
		bitgetTax+bitgetP2PRecord, params.Values, nil, &resp)
}

// GetP2PMerchantList returns detailed information on a particular merchant
func (bi *Bitget) GetP2PMerchantList(ctx context.Context, online, merchantID string, limit, pagination int64) (*P2PMerListResp, error) {
	vals := url.Values{}
	vals.Set("online", online)
	vals.Set("merchantId", merchantID)
	vals.Set("limit", strconv.FormatInt(limit, 10))
	vals.Set("idLessThan", strconv.FormatInt(pagination, 10))
	var resp *P2PMerListResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet,
		bitgetP2P+bitgetMerchantList, vals, nil, &resp)
}

// GetMerchantInfo returns detailed information on the user as a merchant
func (bi *Bitget) GetMerchantInfo(ctx context.Context) (*P2PMerInfoResp, error) {
	var resp *P2PMerInfoResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet,
		bitgetP2P+bitgetMerchantInfo, nil, nil, &resp)
}

// GetMerchantP2POrders returns information on the user's P2P orders
func (bi *Bitget) GetMerchantP2POrders(ctx context.Context, startTime, endTime time.Time, limit, pagination, adNum, ordNum int64, status, side, cryptoCurrency, fiatCurrency string) (*P2POrdersResp, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, false)
	if err != nil {
		return nil, err
	}
	params.Values.Set("limit", strconv.FormatInt(limit, 10))
	params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	params.Values.Set("advNo", strconv.FormatInt(adNum, 10))
	params.Values.Set("orderNo", strconv.FormatInt(ordNum, 10))
	params.Values.Set("status", status)
	params.Values.Set("side", side)
	params.Values.Set("coin", cryptoCurrency)
	// params.Values.Set("language", "en-US")
	params.Values.Set("fiat", fiatCurrency)
	var resp *P2POrdersResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet,
		bitgetP2P+bitgetOrderList, params.Values, nil, &resp)
}

// GetMerchantAdvertisementList returns information on a variety of merchant advertisements
func (bi *Bitget) GetMerchantAdvertisementList(ctx context.Context, startTime, endTime time.Time, limit, pagination, adNum, payMethodID int64, status, side, cryptoCurrency, fiatCurrency, orderBy, sourceType string) (*P2PAdListResp, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, false)
	if err != nil {
		return nil, err
	}
	params.Values.Set("limit", strconv.FormatInt(limit, 10))
	params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	params.Values.Set("advNo", strconv.FormatInt(adNum, 10))
	params.Values.Set("payMethodId", strconv.FormatInt(payMethodID, 10))
	params.Values.Set("status", status)
	params.Values.Set("side", side)
	params.Values.Set("coin", cryptoCurrency)
	// params.Values.Set("language", "en-US")
	params.Values.Set("fiat", fiatCurrency)
	params.Values.Set("orderBy", orderBy)
	params.Values.Set("sourceType", sourceType)
	var resp *P2PAdListResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet,
		bitgetP2P+bitgetAdvList, params.Values, nil, &resp)
}

// CreateVirtualSubaccounts creates a batch of virtual subaccounts. These names must use English letters,
// no spaces, no numbers, and be exactly 8 characters long.
func (bi *Bitget) CreateVirtualSubaccounts(ctx context.Context, subaccounts []string) (*CrVirSubResp, error) {
	if len(subaccounts) == 0 {
		return nil, errSubAccountEmpty
	}
	path := bitgetUser + bitgetCreate + bitgetVirtualSubaccount
	req := map[string]interface{}{
		"subAccountList": subaccounts,
	}
	var resp *CrVirSubResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate5, http.MethodPost, path,
		nil, req, &resp)
}

// ModifyVirtualSubaccount changes the permissions and/or status of a virtual subaccount
func (bi *Bitget) ModifyVirtualSubaccount(ctx context.Context, subaccountID, newStatus string, newPerms []string) (*ModVirSubResp, error) {
	if subaccountID == "" {
		return nil, errSubAccountEmpty
	}
	if newStatus == "" {
		return nil, errNewStatusEmpty
	}
	if len(newPerms) == 0 {
		return nil, errNewPermsEmpty
	}
	path := bitgetUser + bitgetModify + bitgetVirtualSubaccount
	req := map[string]interface{}{
		"subAccountUid": subaccountID,
		"status":        newStatus,
		"permList":      newPerms,
	}
	var resp *ModVirSubResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate5, http.MethodPost, path,
		nil, req, &resp)
}

// CreateSubaccountAndAPIKey creates a subaccounts and an API key. Every account can have up to 20 sub-accounts,
// and every API key can have up to 10 API keys. The name of the sub-account must be exactly 8 English letters.
// The passphrase of the API key must be 8-32 letters and/or numbers. The label must be 20 or fewer characters.
// A maximum of 30 IPs can be a part of the whitelist.
func (bi *Bitget) CreateSubaccountAndAPIKey(ctx context.Context, subaccountName, passphrase, label string, whiteList, permList []string) (*CrSubAccAPIKeyResp, error) {
	if subaccountName == "" {
		return nil, errSubAccountEmpty
	}
	req := map[string]interface{}{
		"subAccountName": subaccountName,
		"passphrase":     passphrase,
		"label":          label,
		"ipList":         whiteList,
		"permList":       permList,
	}
	var resp *CrSubAccAPIKeyResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate1, http.MethodPost,
		bitgetUser+bitgetBatchCreateSubAccApi, nil, req, &resp)
}

// GetVirtualSubaccounts returns a list of the user's virtual sub-accounts
func (bi *Bitget) GetVirtualSubaccounts(ctx context.Context, limit, pagination int64, status string) (*GetVirSubResp, error) {
	vals := url.Values{}
	vals.Set("limit", strconv.FormatInt(limit, 10))
	vals.Set("idLessThan", strconv.FormatInt(pagination, 10))
	vals.Set("status", status)
	path := bitgetUser + bitgetVirtualSubaccount + "-" + bitgetList
	var resp *GetVirSubResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals,
		nil, &resp)
}

// CreateAPIKey creates an API key for the selected virtual sub-account
func (bi *Bitget) CreateAPIKey(ctx context.Context, subaccountID, passphrase, label string, whiteList, permList []string) (*AlterAPIKeyResp, error) {
	if subaccountID == "" {
		return nil, errSubAccountEmpty
	}
	if passphrase == "" {
		return nil, errPassphraseEmpty
	}
	if label == "" {
		return nil, errLabelEmpty
	}
	path := bitgetUser + bitgetCreate + bitgetVirtualSubaccount + "-" + bitgetAPIKey
	req := map[string]interface{}{
		"subAccountUid": subaccountID,
		"passphrase":    passphrase,
		"label":         label,
		"ipList":        whiteList,
		"permList":      permList,
	}
	var resp *AlterAPIKeyResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate5, http.MethodPost, path,
		nil, req, &resp)
}

// ModifyAPIKey modifies the label, IP whitelist, and/or permissions of the API key associated with the selected
// virtual sub-account
func (bi *Bitget) ModifyAPIKey(ctx context.Context, subaccountID, passphrase, label, apiKey string, whiteList, permList []string) (*AlterAPIKeyResp, error) {
	if apiKey == "" {
		return nil, errAPIKeyEmpty
	}
	if passphrase == "" {
		return nil, errPassphraseEmpty
	}
	if label == "" {
		return nil, errLabelEmpty
	}
	if subaccountID == "" {
		return nil, errSubAccountEmpty
	}
	path := bitgetUser + bitgetModify + bitgetVirtualSubaccount + "-" + bitgetAPIKey
	req := make(map[string]interface{})
	req["subAccountUid"] = subaccountID
	req["passphrase"] = passphrase
	req["label"] = label
	req["subAccountApiKey"] = apiKey
	req["ipList"] = whiteList
	req["permList"] = permList
	var resp *AlterAPIKeyResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate5, http.MethodPost, path,
		nil, req, &resp)
}

// GetAPIKeys lists the API keys associated with the selected virtual sub-account
func (bi *Bitget) GetAPIKeys(ctx context.Context, subaccountID string) (*GetAPIKeyResp, error) {
	if subaccountID == "" {
		return nil, errSubAccountEmpty
	}
	vals := url.Values{}
	vals.Set("subAccountUid", subaccountID)
	path := bitgetUser + bitgetVirtualSubaccount + "-" + bitgetAPIKey + "-" + bitgetList
	var resp *GetAPIKeyResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals,
		nil, &resp)
}

// GetConvertCoins returns a list of supported currencies, your balance in those currencies, and the maximum and
// minimum tradable amounts of those currencies
func (bi *Bitget) GetConvertCoins(ctx context.Context) (*ConvertCoinsResp, error) {
	var resp *ConvertCoinsResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet,
		bitgetConvert+bitgetCurrencies, nil, nil, &resp)
}

// GetQuotedPrice returns the price of a given amount of one currency in terms of another currency, and an
// ID for this quote, to be used in a subsequent conversion
func (bi *Bitget) GetQuotedPrice(ctx context.Context, fromCurrency, toCurrency string, fromAmount, toAmount float64) (*QuotedPriceResp, error) {
	if fromCurrency == "" || toCurrency == "" {
		return nil, errCurrencyEmpty
	}
	if (fromAmount == 0 && toAmount == 0) || (fromAmount != 0 && toAmount != 0) {
		return nil, errFromToMutex
	}
	vals := url.Values{}
	vals.Set("fromCoin", fromCurrency)
	vals.Set("toCoin", toCurrency)
	if fromAmount != 0 {
		vals.Set("fromCoinSize", strconv.FormatFloat(fromAmount, 'f', -1, 64))
	} else {
		vals.Set("toCoinSize", strconv.FormatFloat(toAmount, 'f', -1, 64))
	}
	var resp *QuotedPriceResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet,
		bitgetConvert+bitgetQuotedPrice, vals, nil, &resp)
}

// CommitConversion commits a conversion previously quoted by GetQuotedPrice. This quote has to have been issued
// within the last 8 seconds.
func (bi *Bitget) CommitConversion(ctx context.Context, fromCurrency, toCurrency, traceID string, fromAmount, toAmount, price float64) (*CommitConvResp, error) {
	if fromCurrency == "" || toCurrency == "" {
		return nil, errCurrencyEmpty
	}
	if traceID == "" {
		return nil, errTraceIDEmpty
	}
	if fromAmount == 0 || toAmount == 0 {
		return nil, errAmountEmpty
	}
	if price == 0 {
		return nil, errPriceEmpty
	}
	req := map[string]interface{}{
		"fromCoin":     fromCurrency,
		"toCoin":       toCurrency,
		"traceId":      traceID,
		"fromCoinSize": strconv.FormatFloat(fromAmount, 'f', -1, 64),
		"toCoinSize":   strconv.FormatFloat(toAmount, 'f', -1, 64),
		"cnvtPrice":    strconv.FormatFloat(price, 'f', -1, 64),
	}
	var resp *CommitConvResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost,
		bitgetConvert+bitgetTrade, nil, req, &resp)
}

// GetConvertHistory returns a list of the user's previous conversions
func (bi *Bitget) GetConvertHistory(ctx context.Context, startTime, endTime time.Time, limit, pagination int64) (*ConvHistResp, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, false)
	if err != nil {
		return nil, err
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	var resp *ConvHistResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet,
		bitgetConvert+bitgetConvertRecord, params.Values, nil, &resp)
}

// GetBGBConvertCoins returns a list of available currencies, with information on converting them to BGB
func (bi *Bitget) GetBGBConvertCoins(ctx context.Context) (*BGBConvertCoinsResp, error) {
	var resp *BGBConvertCoinsResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet,
		bitgetConvert+bitgetConvertCoinList, nil, nil, &resp)
}

// ConvertBGB converts all funds in the listed currencies to BGB
func (bi *Bitget) ConvertBGB(ctx context.Context, currencies []string) (*ConvertBGBResp, error) {
	if len(currencies) == 0 {
		return nil, errCurrencyEmpty
	}
	req := map[string]interface{}{
		"coinList": currencies,
	}
	var resp *ConvertBGBResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost,
		bitgetConvert+bitgetBGBConvert, nil, req, &resp)
}

// GetBGBConvertHistory returns a list of the user's previous BGB conversions
func (bi *Bitget) GetBGBConvertHistory(ctx context.Context, orderID, limit, pagination int64, startTime, endTime time.Time) (*BGBConvHistResp, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, true)
	if err != nil {
		return nil, err
	}
	params.Values.Set("orderId", strconv.FormatInt(orderID, 10))
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	var resp *BGBConvHistResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet,
		bitgetConvert+bitgetBGBConvertRecords, params.Values, nil, &resp)
}

// GetCoinInfo returns information on all supported spot currencies, or a single currency of the user's choice
func (bi *Bitget) GetCoinInfo(ctx context.Context, currency string) (*CoinInfoResp, error) {
	vals := url.Values{}
	vals.Set("coin", currency)
	path := bitgetSpot + bitgetPublic + bitgetCoins
	var resp *CoinInfoResp
	return resp, bi.SendHTTPRequest(ctx, exchange.RestSpot, Rate3, path, vals, &resp)
}

// GetSymbolInfo returns information on all supported spot trading pairs, or a single pair of the user's choice
func (bi *Bitget) GetSymbolInfo(ctx context.Context, pair string) (*SymbolInfoResp, error) {
	vals := url.Values{}
	vals.Set("symbol", pair)
	path := bitgetSpot + bitgetPublic + bitgetSymbols
	var resp *SymbolInfoResp
	return resp, bi.SendHTTPRequest(ctx, exchange.RestSpot, Rate20, path, vals, &resp)
}

// GetVIPFeeRate returns the different levels of VIP fee rates
func (bi *Bitget) GetVIPFeeRate(ctx context.Context) (*VIPFeeRateResp, error) {
	path := bitgetSpot + bitgetMarket + bitgetVIPFeeRate
	var resp *VIPFeeRateResp
	return resp, bi.SendHTTPRequest(ctx, exchange.RestSpot, Rate10, path, nil, &resp)
}

// GetTickerInformation returns the ticker information for all trading pairs, or a single pair of the user's choice
func (bi *Bitget) GetTickerInformation(ctx context.Context, pair string) (*TickerResp, error) {
	vals := url.Values{}
	vals.Set("symbol", pair)
	path := bitgetSpot + bitgetMarket + bitgetTickers
	var resp *TickerResp
	return resp, bi.SendHTTPRequest(ctx, exchange.RestSpot, Rate20, path, vals, &resp)
}

// GetMergeDepth returns part of the orderbook, with options to merge orders of similar price levels together,
// and to change how many results are returned. Limit's a string instead of the typical int64 because the API
// will accept a value of "max"
func (bi *Bitget) GetMergeDepth(ctx context.Context, pair, precision, limit string) (*DepthResp, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair)
	vals.Set("precision", precision)
	vals.Set("limit", limit)
	path := bitgetSpot + bitgetMarket + bitgetMergeDepth
	var resp *DepthResp
	return resp, bi.SendHTTPRequest(ctx, exchange.RestSpot, Rate20, path, vals, &resp)
}

// GetOrderbookDepth returns the orderbook for a given trading pair, with options to merge orders of similar price
// levels together, and to change how many results are returned.
func (bi *Bitget) GetOrderbookDepth(ctx context.Context, pair, step string, limit uint8) (*OrderbookResp, error) {
	vals := url.Values{}
	vals.Set("symbol", pair)
	vals.Set("type", step)
	vals.Set("limit", strconv.FormatUint(uint64(limit), 10))
	path := bitgetSpot + bitgetMarket + bitgetOrderbook
	var resp *OrderbookResp
	return resp, bi.SendHTTPRequest(ctx, exchange.RestSpot, Rate20, path, vals, &resp)
}

// GetCandlestickData returns candlestick data for a given trading pair
func (bi *Bitget) GetCandlestickData(ctx context.Context, pair, granularity string, startTime, endTime time.Time, limit uint16, historic bool) (*CandleData, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	if granularity == "" {
		return nil, errGranEmpty
	}
	var path string
	var params Params
	params.Values = make(url.Values)
	if historic {
		if endTime.IsZero() || endTime.Equal(time.Unix(0, 0)) {
			return nil, errEndTimeEmpty
		}
		path = bitgetSpot + bitgetMarket + bitgetHistoryCandles
		params.Values.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	} else {
		path = bitgetSpot + bitgetMarket + bitgetCandles
		err := params.prepareDateString(startTime, endTime, true)
		if err != nil {
			return nil, err
		}
	}
	params.Values.Set("symbol", pair)
	params.Values.Set("granularity", granularity)
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatUint(uint64(limit), 10))
	}
	var resp *CandleResponse
	err := bi.SendHTTPRequest(ctx, exchange.RestSpot, Rate20, path, params.Values, &resp)
	if err != nil {
		return nil, err
	}
	var data CandleData
	data.Candles = make([]OneCandle, len(resp.Data))
	for i := range resp.Data {
		timeTemp, ok := resp.Data[i][0].(string)
		if !ok {
			return nil, errTypeAssertTimestamp
		}
		timeTemp = (timeTemp)[1 : len(timeTemp)-1]
		timeTemp2, err := strconv.ParseInt(timeTemp, 10, 64)
		if err != nil {
			return nil, err
		}
		data.Candles[i].Timestamp = time.Time(UnixTimestamp(time.UnixMilli(timeTemp2).UTC()))
		openTemp, ok := resp.Data[i][1].(string)
		if !ok {
			return nil, errTypeAssertOpenPrice
		}
		data.Candles[i].Open, err = strconv.ParseFloat(openTemp, 64)
		if err != nil {
			return nil, err
		}
		highTemp, ok := resp.Data[i][2].(string)
		if !ok {
			return nil, errTypeAssertHighPrice
		}
		data.Candles[i].High, err = strconv.ParseFloat(highTemp, 64)
		if err != nil {
			return nil, err
		}
		lowTemp, ok := resp.Data[i][3].(string)
		if !ok {
			return nil, errTypeAssertLowPrice
		}
		data.Candles[i].Low, err = strconv.ParseFloat(lowTemp, 64)
		if err != nil {
			return nil, err
		}
		closeTemp, ok := resp.Data[i][4].(string)
		if !ok {
			return nil, errTypeAssertClosePrice
		}
		data.Candles[i].Close, err = strconv.ParseFloat(closeTemp, 64)
		if err != nil {
			return nil, err
		}
		baseVolumeTemp := resp.Data[i][5].(string)
		if !ok {
			return nil, errTypeAssertBaseVolume
		}
		data.Candles[i].BaseVolume, err = strconv.ParseFloat(baseVolumeTemp, 64)
		if err != nil {
			return nil, err
		}
		quoteVolumeTemp := resp.Data[i][6].(string)
		if !ok {
			return nil, errTypeAssertQuoteVolume
		}
		data.Candles[i].QuoteVolume, err = strconv.ParseFloat(quoteVolumeTemp, 64)
		if err != nil {
			return nil, err
		}
		usdtVolumeTemp := resp.Data[i][7].(string)
		if !ok {
			return nil, errTypeAssertUSDTVolume
		}
		data.Candles[i].USDTVolume, err = strconv.ParseFloat(usdtVolumeTemp, 64)
		if err != nil {
			return nil, err
		}
	}
	return &data, nil
}

// GetRecentFills returns the most recent trades for a given pair
func (bi *Bitget) GetRecentFills(ctx context.Context, pair string, limit uint16) (*FillsResp, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair)
	vals.Set("limit", strconv.FormatInt(int64(limit), 10))
	path := bitgetSpot + bitgetMarket + bitgetFills
	var resp *FillsResp
	return resp, bi.SendHTTPRequest(ctx, exchange.RestSpot, Rate10, path, vals, &resp)
}

// GetMarketTrades returns trades for a given pair within a particular time range, and/or before a certain ID
func (bi *Bitget) GetMarketTrades(ctx context.Context, pair string, startTime, endTime time.Time, limit, pagination int64) (*FillsResp, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, true)
	if err != nil {
		return nil, err
	}
	params.Values.Set("symbol", pair)
	params.Values.Set("limit", strconv.FormatInt(limit, 10))
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	path := bitgetSpot + bitgetMarket + bitgetFillsHistory
	var resp *FillsResp
	return resp, bi.SendHTTPRequest(ctx, exchange.RestSpot, Rate10, path, params.Values, &resp)
}

// PlaceOrder places an order on the exchange
func (bi *Bitget) PlaceOrder(ctx context.Context, pair, side, orderType, strategy, clientOrderID string, price, amount float64) (*OrderResp, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	if side == "" {
		return nil, errSideEmpty
	}
	if orderType == "" {
		return nil, errOrderTypeEmpty
	}
	if strategy == "" {
		return nil, errStrategyEmpty
	}
	if orderType == "limit" && price == 0 {
		return nil, errLimitPriceEmpty
	}
	if amount == 0 {
		return nil, errAmountEmpty
	}
	req := map[string]interface{}{
		"symbol":    pair,
		"side":      side,
		"orderType": orderType,
		"force":     strategy,
		"price":     strconv.FormatFloat(price, 'f', -1, 64),
		"size":      strconv.FormatFloat(amount, 'f', -1, 64),
		"clientOid": clientOrderID,
	}
	path := bitgetSpot + bitgetTrade + bitgetPlaceOrder
	var resp *OrderResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req,
		&resp)
}

// CancelOrderByID cancels an order on the exchange
func (bi *Bitget) CancelOrderByID(ctx context.Context, pair, clientOrderID string, orderID int64) (*OrderResp, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	if orderID == 0 && clientOrderID == "" {
		return nil, errOrderClientEmpty
	}
	req := map[string]interface{}{
		"symbol":    pair,
		"orderId":   strconv.FormatInt(orderID, 10),
		"clientOid": clientOrderID,
	}
	path := bitgetSpot + bitgetTrade + bitgetCancelOrder
	var resp *OrderResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req,
		&resp)
}

// BatchPlaceOrders places up to fifty orders on the exchange
func (bi *Bitget) BatchPlaceOrder(ctx context.Context, pair string, orders []PlaceOrderStruct) (*BatchOrderResp, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	if len(orders) == 0 {
		return nil, errOrdersEmpty
	}
	req := map[string]interface{}{
		"symbol":    pair,
		"orderData": orders,
	}
	path := bitgetSpot + bitgetTrade + bitgetBatchOrders
	var resp *BatchOrderResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate5, http.MethodPost, path, nil, req,
		&resp)
}

// SendAuthenticatedHTTPRequest sends an authenticated HTTP request
func (bi *Bitget) SendAuthenticatedHTTPRequest(ctx context.Context, ep exchange.URL, rateLim request.EndpointLimit, method, path string, queryParams url.Values, bodyParams map[string]interface{}, result interface{}) error {
	creds, err := bi.GetCredentials(ctx)
	if err != nil {
		return err
	}
	endpoint, err := bi.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	newRequest := func() (*request.Item, error) {
		payload := []byte("")
		if bodyParams != nil {
			payload, err = json.Marshal(bodyParams)
			if err != nil {
				return nil, err
			}
		}
		path = common.EncodeURLValues(path, queryParams)
		t := strconv.FormatInt(time.Now().UnixMilli(), 10)
		message := t + method + "/api/v2/" + path + string(payload)
		// The exchange also supports user-generated RSA keys, but we haven't implemented that yet
		var hmac []byte
		hmac, err = crypto.GetHMAC(crypto.HashSHA256, []byte(message), []byte(creds.Secret))
		if err != nil {
			return nil, err
		}
		headers := make(map[string]string)
		headers["ACCESS-KEY"] = creds.Key
		headers["ACCESS-SIGN"] = crypto.Base64Encode(hmac)
		headers["ACCESS-TIMESTAMP"] = t
		headers["ACCESS-PASSPHRASE"] = creds.ClientID
		headers["Content-Type"] = "application/json"
		headers["locale"] = "en-US"
		return &request.Item{
			Method:        method,
			Path:          endpoint + path,
			Headers:       headers,
			Body:          bytes.NewBuffer(payload),
			Result:        &result,
			Verbose:       bi.Verbose,
			HTTPDebugging: bi.HTTPDebugging,
			HTTPRecording: bi.HTTPRecording,
		}, nil
	}
	return bi.SendPayload(ctx, rateLim, newRequest, request.AuthenticatedRequest)
}

// SendHTTPRequest sends an unauthenticated HTTP request, with a few assumptions about the request;
// namely that it is a GET request with no body
func (bi *Bitget) SendHTTPRequest(ctx context.Context, ep exchange.URL, rateLim request.EndpointLimit, path string, queryParams url.Values, result interface{}) error {
	endpoint, err := bi.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	newRequest := func() (*request.Item, error) {
		path = common.EncodeURLValues(path, queryParams)
		return &request.Item{
			Method:        "GET",
			Path:          endpoint + path,
			Result:        &result,
			Verbose:       bi.Verbose,
			HTTPDebugging: bi.HTTPDebugging,
			HTTPRecording: bi.HTTPRecording,
		}, nil
	}
	return bi.SendPayload(ctx, rateLim, newRequest, request.UnauthenticatedRequest)
}

// PrepareDateString encodes a set of parameters indicating start & end dates
func (p *Params) prepareDateString(startDate, endDate time.Time, ignoreUnset bool) error {
	err := common.StartEndTimeCheck(startDate, endDate)
	if err != nil {
		if errors.Is(err, common.ErrDateUnset) && ignoreUnset {
			return nil
		}
		return err
	}
	p.Values.Set("startTime", strconv.FormatInt(startDate.UnixMilli(), 10))
	p.Values.Set("endTime", strconv.FormatInt(endDate.UnixMilli(), 10))
	return nil
}

// UnmarshalJSON unmarshals the JSON input into a UnixTimestamp type
func (t *UnixTimestamp) UnmarshalJSON(b []byte) error {
	var timestampStr string
	err := json.Unmarshal(b, &timestampStr)
	if err != nil {
		return err
	}
	if timestampStr == "" {
		*t = UnixTimestamp(time.Time{})
		return nil
	}
	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return err
	}
	*t = UnixTimestamp(time.UnixMilli(timestamp).UTC())
	return nil
}

// String implements the stringer interface
func (t *UnixTimestamp) String() string {
	return t.Time().String()
}

// Time returns the time.Time representation of the UnixTimestamp
func (t *UnixTimestamp) Time() time.Time {
	return time.Time(*t)
}

// UnmarshalJSON unmarshals the JSON input into a YesNoBool type
func (y *YesNoBool) UnmarshalJSON(b []byte) error {
	var yn string
	err := json.Unmarshal(b, &yn)
	if err != nil {
		return err
	}
	switch yn {
	case "yes":
		*y = true
	case "no":
		*y = false
	}
	return nil
}

// UnmarshalJSON unmarshals the JSON input into a SuccessBool type
func (s *SuccessBool) UnmarshalJSON(b []byte) error {
	var success string
	err := json.Unmarshal(b, &success)
	if err != nil {
		return err
	}
	switch success {
	case "success":
		*s = true
	case "failure":
		*s = false
	}
	return nil
}
