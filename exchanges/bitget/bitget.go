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
	bitgetPublic        = "public"
	bitgetAnnouncements = "annoucements" // sic
	bitgetTime          = "time"

	// Authenticated endpoints
	bitgetCommon                  = "common"
	bitgetTradeRate               = "trade-rate"
	bitgetTax                     = "tax"
	bitgetSpotRecord              = "spot-record"
	bitgetFutureRecord            = "future-record"
	bitgetMarginRecord            = "margin-record"
	bitgetP2PRecord               = "p2p-record"
	bitgetP2P                     = "p2p"
	bitgetMerchantList            = "merchantList"
	bitgetMerchantInfo            = "merchantInfo"
	bitgetOrderList               = "orderList"
	bitgetAdvList                 = "advList"
	bitgetUser                    = "user"
	bitgetCreateVirtualSubaccount = "create-virtual-subaccount"
	bitgetModifyVirtualSubaccount = "modify-virtual-subaccount"
	bitgetVirtualSubaccountList   = "virtual-subaccount-list"
	bitgetCreateAPIKey            = "create-virtual-subaccount-apikey"
	bitgetModifyAPIKey            = "modify-virtual-subaccount-apikey"
	bitgetAPIKeyList              = "virtual-subaccount-apikey-list"
	bitgetConvert                 = "convert"
	bitgetCurrencies              = "currencies"
	bitgetQuotedPrice             = "quoted-price"

	// Errors
	errUnknownEndpointLimit = "unknown endpoint limit %v"
)

var (
	errBusinessTypeEmpty = errors.New("businessType cannot be empty")
	errPairEmpty         = errors.New("currency pair cannot be empty")
	errProductTypeEmpty  = errors.New("productType cannot be empty")
	errSubAccountEmpty   = errors.New("subaccounts cannot be empty")
	errNewStatusEmpty    = errors.New("newStatus cannot be empty")
	errNewPermsEmpty     = errors.New("newPerms cannot be empty")
	errPassphraseEmpty   = errors.New("passphrase cannot be empty")
	errLabelEmpty        = errors.New("label cannot be empty")
	errAPIKeyEmpty       = errors.New("apiKey cannot be empty")
	errFromToMutex       = errors.New("exactly one of fromAmount and toAmount must be set")
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
	path := bitgetPublic + "/" + bitgetAnnouncements
	var resp *AnnResp
	return resp, bi.SendHTTPRequest(ctx, exchange.RestSpot, Rate20, path, params.Values, &resp)
}

// GetTime returns the server's time
func (bi *Bitget) GetTime(ctx context.Context) (*TimeResp, error) {
	var resp *TimeResp
	path := bitgetPublic + "/" + bitgetTime
	return resp, bi.SendHTTPRequest(ctx, exchange.RestSpot, Rate20, path, nil, &resp)
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
	path := bitgetCommon + "/" + bitgetTradeRate
	var resp *TradeRateResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals,
		nil, &resp)
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
	path := bitgetTax + "/" + bitgetSpotRecord
	var resp *SpotTrResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate1, http.MethodGet, path,
		params.Values, nil, &resp)
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
	path := bitgetTax + "/" + bitgetFutureRecord
	var resp *FutureTrResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate1, http.MethodGet, path,
		params.Values, nil, &resp)
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
	path := bitgetTax + "/" + bitgetMarginRecord
	var resp *MarginTrResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate1, http.MethodGet, path,
		params.Values, nil, &resp)
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
	path := bitgetTax + "/" + bitgetP2PRecord
	var resp *P2PTrResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate1, http.MethodGet, path,
		params.Values, nil, &resp)
}

// GetP2PMerchantList returns detailed information on a particular merchant
func (bi *Bitget) GetP2PMerchantList(ctx context.Context, online, merchantID string, limit, pagination int64) (*P2PMerListResp, error) {
	vals := url.Values{}
	vals.Set("online", online)
	vals.Set("merchantId", merchantID)
	vals.Set("limit", strconv.FormatInt(limit, 10))
	vals.Set("idLessThan", strconv.FormatInt(pagination, 10))
	path := bitgetP2P + "/" + bitgetMerchantList
	var resp *P2PMerListResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path,
		vals, nil, &resp)
}

// GetMerchantInfo returns detailed information on the user as a merchant
func (bi *Bitget) GetMerchantInfo(ctx context.Context) (*P2PMerInfoResp, error) {
	path := bitgetP2P + "/" + bitgetMerchantInfo
	var resp *P2PMerInfoResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, nil, nil, &resp)
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
	path := bitgetP2P + "/" + bitgetOrderList
	var resp *P2POrdersResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path,
		params.Values, nil, &resp)
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
	path := bitgetP2P + "/" + bitgetAdvList
	var resp *P2PAdListResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path,
		params.Values, nil, &resp)
}

// CreateVirtualSubaccounts creates a batch of virtual subaccounts. These names must use English letters,
// no spaces, no numbers, and be exactly 8 characters long.
func (bi *Bitget) CreateVirtualSubaccounts(ctx context.Context, subaccounts []string) (*CrVirSubResp, error) {
	if len(subaccounts) == 0 {
		return nil, errSubAccountEmpty
	}
	path := bitgetUser + "/" + bitgetCreateVirtualSubaccount
	req := make(map[string]interface{})
	req["subAccountList"] = subaccounts
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
	path := bitgetUser + "/" + bitgetModifyVirtualSubaccount
	req := make(map[string]interface{})
	req["subAccountUid"] = subaccountID
	req["status"] = newStatus
	req["permList"] = newPerms
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
	// if passphrase == "" {
	// 	return nil, errPassphraseEmpty
	// }
	// if label == "" {
	// 	return nil, errLabelEmpty
	// }
	path := bitgetUser + "/" + bitgetCreateVirtualSubaccount
	req := make(map[string]interface{})
	req["subAccountName"] = subaccountName
	req["passphrase"] = passphrase
	req["label"] = label
	req["ipList"] = whiteList
	req["permList"] = permList
	var resp *CrSubAccAPIKeyResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate1, http.MethodPost, path,
		nil, req, &resp)
}

// GetVirtualSubaccounts returns a list of the user's virtual sub-accounts
func (bi *Bitget) GetVirtualSubaccounts(ctx context.Context, limit, pagination int64, status string) (*GetVirSubResp, error) {
	vals := url.Values{}
	vals.Set("limit", strconv.FormatInt(limit, 10))
	vals.Set("idLessThan", strconv.FormatInt(pagination, 10))
	vals.Set("status", status)
	path := bitgetUser + "/" + bitgetVirtualSubaccountList
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
	path := bitgetUser + "/" + bitgetCreateAPIKey
	req := make(map[string]interface{})
	req["subAccountUid"] = subaccountID
	req["passphrase"] = passphrase
	req["label"] = label
	req["ipList"] = whiteList
	req["permList"] = permList
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
	path := bitgetUser + "/" + bitgetModifyAPIKey
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
	path := bitgetUser + "/" + bitgetAPIKeyList
	var resp *GetAPIKeyResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals,
		nil, &resp)
}

// GetConvertCoins returns a list of supported currencies, your balance in those currencies, and the maximum and
// minimum tradable amounts of those currencies
func (bi *Bitget) GetConvertCoins(ctx context.Context) (*ConvertCoinsResp, error) {
	path := bitgetConvert + "/" + bitgetCurrencies
	var resp *ConvertCoinsResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, nil,
		nil, &resp)
}

// GetQuotedPrice
func (bi *Bitget) GetQuotedPrice(ctx context.Context, fromCurrency, toCurrency string, fromAmount, toAmount float64) (*QuotedPriceResp, error) {
	if fromCurrency == "" || toCurrency == "" {
		return nil, errPairEmpty
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
	path := bitgetConvert + "/" + bitgetQuotedPrice
	var resp *QuotedPriceResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals,
		nil, &resp)
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
