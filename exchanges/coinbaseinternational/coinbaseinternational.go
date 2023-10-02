package coinbaseinternational

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Coinbaseinternational is the overarching type across this package
type CoinbaseInternational struct {
	exchange.Base
}

const (
	coinbaseInternationalAPIURL = "https://api.international.coinbase.com"
	coinbaseAPIVersion          = "/api/v1"

	// Public endpoints

	// Authenticated endpoints
)

// Start implementing public and private exchange API funcs below

// --------------------------------------  Assets  --------------------------

// ListAssets returns a list of all supported assets.
func (co *CoinbaseInternational) ListAssets(ctx context.Context) ([]AssetItemInfo, error) {
	var resp []AssetItemInfo
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, "assets", nil, &resp)
}

// GetAssetDetails retrieves information for a specific asset.
func (co *CoinbaseInternational) GetAssetDetails(ctx context.Context, assetName currency.Code, assetUUID, assetID string) (*AssetItemInfo, error) {
	var resp AssetItemInfo
	path := "assets/"
	switch {
	case !assetName.IsEmpty():
		path += assetName.String()
	case assetUUID != "":
		path += assetUUID
	case assetID != "":
		path += assetID
	default:
		return nil, errors.New("missing asset information; ")
	}
	return &resp, co.SendHTTPRequest(ctx, exchange.RestSpot, path, nil, &resp)
}

// GetSupportedNetworkPerAsset returns a list of supported networks and network information for a specific asset.
func (co *CoinbaseInternational) GetSupportedNetworksPerAsset(ctx context.Context, assetName currency.Code, assetUUID, assetID string) ([]AssetInfoWithSupportedNetwork, error) {
	var resp []AssetInfoWithSupportedNetwork
	path := "assets/"
	switch {
	case !assetName.IsEmpty():
		path += assetName.String()
	case assetUUID != "":
		path += assetUUID
	case assetID != "":
		path += assetID
	default:
		return nil, errors.New("missing asset information; ")
	}
	path += "/networks"
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, path, nil, &resp)
}

// -------------------------------------- Instrument Information ----------------------------------------------------------

// GetInstruments returns all of the instruments available for trading.
func (co *CoinbaseInternational) GetInstruments(ctx context.Context) ([]InstrumentInfo, error) {
	var resp []InstrumentInfo
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, "instruments", nil, &resp)
}

// GetInstrumentDetails retrieves market information for a specific instrument.
func (co *CoinbaseInternational) GetInstrumentDetails(ctx context.Context, instrumentName, instrumentUUID, instrumentID string) (*InstrumentInfo, error) {
	var resp InstrumentInfo
	path := "instruments/"
	switch {
	case instrumentName != "":
		path += instrumentName
	case instrumentUUID != "":
		path += instrumentUUID
	case instrumentID != "":
		path += instrumentID
	default:
		return nil, errors.New("instrument information is required")
	}
	return &resp, co.SendHTTPRequest(ctx, exchange.RestSpot, path, nil, &resp)
}

// GetQuotePerInstrument retrieves the current quote for a specific instrument.
func (co *CoinbaseInternational) GetQuotePerInstrument(ctx context.Context, instrumentName, instrumentUUID, instrumentID string) (*InstrumentQuoteInformation, error) {
	var resp InstrumentQuoteInformation
	path := "instruments/"
	switch {
	case instrumentName != "":
		path += instrumentName
	case instrumentUUID != "":
		path += instrumentUUID
	case instrumentID != "":
		path += instrumentID
	default:
		return nil, errors.New("instrument information is required")
	}
	return &resp, co.SendHTTPRequest(ctx, exchange.RestSpot, path, nil, &resp)
}

// SendHTTPRequest sends a public HTTP request.
func (co *CoinbaseInternational) SendHTTPRequest(ctx context.Context, ep exchange.URL, path string, params url.Values, result interface{}) error {
	endpoint, err := co.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	urlPath := endpoint + coinbaseAPIVersion + "/" + path
	if params != nil {
		urlPath = common.EncodeURLValues(urlPath, params)
	}
	return co.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
		headers := make(map[string]string)
		headers["Content-Type"] = "application/json"
		headers["Accept"] = "application/json"
		return &request.Item{
			Method:        http.MethodGet,
			Path:          urlPath,
			Headers:       headers,
			Result:        result,
			Verbose:       co.Verbose,
			HTTPDebugging: co.HTTPDebugging,
			HTTPRecording: co.HTTPRecording,
		}, nil
	}, request.UnauthenticatedRequest)
}
