package bitget

import (
	"bytes"
	"context"
	"encoding/json"
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
	bitgetAnnouncements = "annoucements"

	// Authenticated endpoints

	// Errors
	errUnknownEndpointLimit = "unknown endpoint limit %v"
)

func (bi *Bitget) QueryAnnouncements(ctx context.Context, annType string, startTime, endTime time.Time) (AnnResp, error) {
	var resp AnnResp
	vals := url.Values{}
	vals.Set("annType", annType)
	vals.Set("startTime", strconv.FormatInt(startTime.Unix(), 10))
	vals.Set("endTime", strconv.FormatInt(endTime.Unix(), 10))
	vals.Set("language", "en_US")
	path := bitgetPublic + "/" + bitgetAnnouncements
	return resp, bi.SendHTTPRequest(ctx, exchange.RestSpot, bitgetRate20, path, vals, &resp)
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
		message := t + method + path + string(payload)
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
