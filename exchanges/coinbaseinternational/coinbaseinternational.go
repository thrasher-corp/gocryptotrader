package coinbaseinternational

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// CoinbaseInternational is the overarching type across this package
type CoinbaseInternational struct {
	exchange.Base
}

const (
	coinbaseInternationalAPIURL = "https://api.international.coinbase.com"
	coinbaseAPIVersion          = "/api/v1"

	// Public endpoints

	// Authenticated endpoints
)

var (
	errArgumentMustBeInterface = errors.New("argument must be an interface")
	errMissingPortfolioID      = errors.New("missing portfolio identification")
)

// Start implementing public and private exchange API funcs below

// --------------------------------------  Assets  --------------------------

// ListAssets returns a list of all supported assets.
func (co *CoinbaseInternational) ListAssets(ctx context.Context) ([]AssetItemInfo, error) {
	var resp []AssetItemInfo
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "assets", nil, nil, &resp, false)
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
	return &resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, &resp, false)
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
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, &resp, false)
}

// -------------------------------------- Instrument Information ----------------------------------------------------------

// GetInstruments returns all of the instruments available for trading.
func (co *CoinbaseInternational) GetInstruments(ctx context.Context) ([]InstrumentInfo, error) {
	var resp []InstrumentInfo
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "instruments", nil, nil, &resp, false)
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
	return &resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, &resp, false)
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
	return &resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, nil, &resp, false)
}

// CreateOrder creates a new order.
func (co *CoinbaseInternational) CreateOrder(ctx context.Context, arg *OrderRequestParams) (*TradeOrder, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.BaseSize <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	if arg.Price <= 0 {
		return nil, order.ErrPriceBelowMin
	}
	if arg.OrderType == "" {
		return nil, order.ErrUnsupportedOrderType
	}
	var resp TradeOrder
	return &resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "orders", nil, arg, &resp, true)
}

// GetOpenOrders returns a list of active orders resting on the order book matching the requested criteria. Does not return any rejected, cancelled, or fully filled orders as they are not active.
func (co *CoinbaseInternational) GetOpenOrders(ctx context.Context, portfolioUUID, portfolioID, instrument, clientOrderID, eventType string, RefDateTime time.Time, resultOffset, resultLimit int64) (*OrderItemDetail, error) {
	var resp OrderItemDetail
	params := url.Values{}
	switch {
	case portfolioID != "":
		params.Set("portfolio", portfolioID)
	case portfolioUUID != "":
		params.Set("portfolio", portfolioUUID)
	}
	if instrument != "" {
		params.Set("instrument", instrument)
	}
	if clientOrderID != "" {
		params.Set("client_order_id", clientOrderID)
	}
	if eventType != "" {
		params.Set("event_type", eventType)
	}
	if !RefDateTime.IsZero() {
		params.Set("ref_datetime", RefDateTime.String())
	}
	if resultOffset > 0 {
		params.Set("result_offset", strconv.FormatInt(resultOffset, 10))
	}
	if resultLimit > 0 {
		params.Set("result_limit", strconv.FormatInt(resultLimit, 10))
	}
	return &resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "orders", params, nil, &resp, true)
}

// CancelOrders cancels all orders matching the requested criteria.
func (co *CoinbaseInternational) CancelOrders(ctx context.Context, portfolioID, portfolioUUID, instrument string) ([]OrderItem, error) {
	params := url.Values{}
	switch {
	case portfolioID != "":
		params.Set("portfolio", portfolioID)
	case portfolioUUID != "":
		params.Set("portfolio", portfolioUUID)
	default:
		return nil, errMissingPortfolioID
	}
	if instrument != "" {
		params.Set("instrument", instrument)
	}
	var resp []OrderItem
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, "orders", params, nil, &resp, true)
}

// ModifyOpenOrder modifies an open order.
func (co *CoinbaseInternational) ModifyOpenOrder(ctx context.Context, orderID string) (*OrderItem, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp OrderItem
	return &resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPut, "orders/"+orderID, nil, nil, &resp, true)
}

// GetOrderDetails retrieves a single order. The order retrieved can be either active or inactive.
func (co *CoinbaseInternational) GetOrderDetails(ctx context.Context, orderID string) (*OrderItem, error) {
	var resp OrderItem
	return &resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "orders/"+orderID, nil, nil, &resp, true)
}

// CancelTradeOrder cancels a single open order.
func (co *CoinbaseInternational) CancelTradeOrder(ctx context.Context, orderID string) (*OrderItem, error) {
	var resp OrderItem
	return &resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, "orders/"+orderID, nil, nil, &resp, true)
}

//  ----- 	------------------------------- ------------------------------- ------------------------------- ------------------------------- ------------------------------- ------------------------------- ------------------------------- -------------------------------

// GetAllUserPortfolios returns all of the user's portfolios.
func (co *CoinbaseInternational) GetAllUserPortfolios(ctx context.Context) ([]PortfolioItem, error) {
	var resp []PortfolioItem
	return resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "portfolios", nil, nil, &resp, true)
}

// GetPortfolioDetails retrieves the summary, positions, and balances of a portfolio.
func (co *CoinbaseInternational) GetPortfolioDetails(ctx context.Context, portfolioID, portfolioUUID string) (*PortfolioDetail, error) {
	var resp PortfolioDetail
	if portfolioID == "" {
		portfolioID = portfolioUUID
	} else if portfolioUUID == "" {
		return nil, errMissingPortfolioID
	}
	return &resp, co.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "portfolios/"+portfolioID+"/detail", nil, nil, &resp, true)
}

// SendHTTPRequest sends a public HTTP request.
func (co *CoinbaseInternational) SendHTTPRequest(ctx context.Context, ep exchange.URL, method, path string, params url.Values, data, result interface{}, authenticated bool) error {
	endpoint, err := co.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	urlPath := endpoint + coinbaseAPIVersion + "/" + path
	if params != nil {
		urlPath = common.EncodeURLValues(urlPath, params)
	}
	requestType := request.AuthType(request.UnauthenticatedRequest)
	var creds *account.Credentials
	if authenticated {
		creds, err = co.GetCredentials(ctx)
		if err != nil {
			return err
		}
		requestType = request.AuthenticatedRequest
	}
	value := reflect.ValueOf(data)
	var payload []byte
	if value != (reflect.Value{}) && !value.IsNil() && value.Kind() != reflect.Ptr {
		return errArgumentMustBeInterface
	} else if value != (reflect.Value{}) && !value.IsNil() {
		payload, err = json.Marshal(data)
		if err != nil {
			return err
		}
	}
	return co.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
		timestamp := time.Now()
		headers := make(map[string]string)
		headers["Content-Type"] = "application/json"
		headers["Accept"] = "application/json"

		if authenticated {
			headers["CB-ACCESS-KEY"] = creds.Key
			headers["CB-ACCESS-PASSPHRASE"] = creds.ClientID
			headers["CB-ACCESS-TIMESTAMP"] = strconv.FormatInt(timestamp.Unix(), 10)
			signatureString := headers["CB-ACCESS-TIMESTAMP"] + method + coinbaseAPIVersion + "/" + path + string(payload)
			var hmac []byte
			secretBytes, err := crypto.Base64Decode(creds.Secret)
			if err != nil {
				return nil, err
			}
			hmac, err = crypto.GetHMAC(crypto.HashSHA256,
				[]byte(signatureString),
				secretBytes)
			if err != nil {
				return nil, err
			}
			headers["CB-ACCESS-SIGN"] = crypto.Base64Encode(hmac)
		}

		return &request.Item{
			Method:        method,
			Path:          urlPath,
			Headers:       headers,
			Result:        result,
			Body:          bytes.NewBuffer(payload),
			Verbose:       co.Verbose,
			HTTPDebugging: co.HTTPDebugging,
			HTTPRecording: co.HTTPRecording,
		}, nil
	}, requestType)
}

func orderTypeString(oType order.Type) (string, error) {
	switch oType {
	case order.Limit, order.Market, order.Stop:
		return oType.String(), nil
	case order.StopLimit:
		return "STOP_LIMIT", nil
	default:
		return "", order.ErrUnsupportedOrderType
	}
}
