package bitget

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	bitgetCommon     = "common"
	bitgetTradeRate  = "trade-rate"
	bitgetTax        = "tax"
	bitgetSpotRecord = "spot-record"

	// Errors
	errUnknownEndpointLimit = "unknown endpoint limit %v"
)

var (
	errBusinessTypeEmpty = errors.New("businessType cannot be empty")
	errPairEmpty         = errors.New("currency pair cannot be empty")
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
	return resp, bi.SendHTTPRequest(ctx, exchange.RestSpot, bitgetRate20, path, params.Values, &resp)
}

// GetTime returns the server's time
func (bi *Bitget) GetTime(ctx context.Context) (*TimeResp, error) {
	var resp *TimeResp
	path := bitgetPublic + "/" + bitgetTime
	return resp, bi.SendHTTPRequest(ctx, exchange.RestSpot, bitgetRate20, path, nil, &resp)
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
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, bitgetRate10, "GET", path, vals, nil, &resp)
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
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, bitgetRate10, "GET", path, params.Values, nil, &resp)
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
