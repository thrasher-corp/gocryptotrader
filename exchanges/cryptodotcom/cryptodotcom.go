package cryptodotcom

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Cryptodotcom is the overarching type across this package
type Cryptodotcom struct {
	exchange.Base
}

const (
	cryptodotcomAPIURL       = "https://api.crypto.com"
	cryptodotcomAPIVersion   = "/v1/"
	cryptodotcomWebsocketURL = "wss://ws.crypto.com/kline-api/ws"

	// Public endpoints
	marketSymbols     = "symbols"
	marketTicker      = "ticker"
	marketTickers     = "tickers"
	marketKlines      = "klines"
	marketTrades      = "trades"
	marketTickerPrice = "ticker/price"
	marketOrderbook   = "depth"

	// Authenticated endpoints
	userAccountBalance  = "account"
	userCreateOrder     = "order"
	userShowOrder       = "showOrder"
	userOrdersCancel    = "orders/cancel"
	userCancelAllOrders = "cancelAllOrders"
	userOpenOrders      = "openOrders"
	userAllOrders       = "allOrders"
	userExecutedOrders  = "myTrades"
)

// GetSymbols retrives all market symbols.
func (cr *Cryptodotcom) GetSymbols(ctx context.Context) ([]MarketSymbol, error) {
	var resp []MarketSymbol
	return resp, cr.SendHTTPRequest(ctx, exchange.RestSpot, marketSymbols, request.Unset, &resp)
}

// GetTickersInAllAvailableMarkets retrives tickers in all available markets.
func (cr *Cryptodotcom) GetTickersInAllAvailableMarkets(ctx context.Context) (*TickerDetail, error) {
	var resp *TickerDetail
	return resp, cr.SendHTTPRequest(ctx, exchange.RestSpot, marketTicker, request.Unset, &resp)
}

// GetTickerForParticularMarket represents ticker for a particular market.
func (cr *Cryptodotcom) GetTickerForParticularMarket(ctx context.Context, symbol string) (*MarketTickerItem, error) {
	var resp *MarketTickerItem
	if symbol == "" {
		return nil, errSymbolIsRequired
	}
	return resp, cr.SendHTTPRequest(ctx, exchange.RestSpot, marketTicker+"?symbol="+symbol, request.Unset, &resp)
}

// intervalToString returns a string representation of interval.
func intervalToString(interval kline.Interval) (string, error) {
	switch interval {
	case kline.OneMin:
		return "1", nil
	case kline.FiveMin:
		return "5", nil
	case kline.FifteenDay:
		return "15", nil
	case kline.ThirtyMin:
		return "30", nil
	case kline.OneHour:
		return "60", nil
	case kline.OneDay:
		return "1440", nil
	case kline.SevenDay:
		return "10080", nil
	case kline.OneMonth:
		return "43200", nil
	default:
		return "", fmt.Errorf("%v interval:%v", kline.ErrUnsupportedInterval, interval)
	}
}

// stringToInterval converts a string representation to kline.Interval instance.
func stringToInterval(interval string) (kline.Interval, error) {
	switch interval {
	case "1":
		return kline.OneMin, nil
	case "5":
		return kline.FiveMin, nil
	case "15":
		return kline.FifteenDay, nil
	case "30":
		return kline.ThirtyMin, nil
	case "60":
		return kline.OneHour, nil
	case "1440":
		return kline.OneDay, nil
	case "10080":
		return kline.SevenDay, nil
	case "43200":
		return kline.OneMonth, nil
	default:
		return 0, fmt.Errorf("invalid interval string: %s", interval)
	}
}

// GetKlineDataOverSpecifiedPeriod K-line data for a symbol with in a specified period of time.
func (cr *Cryptodotcom) GetKlineDataOverSpecifiedPeriod(ctx context.Context, period kline.Interval, symbol string) ([]KlineItem, error) {
	var intervalString string
	var err error
	if intervalString, err = intervalToString(period); err != nil {
		return nil, err
	}
	if symbol == "" {
		return nil, errSymbolIsRequired
	}
	params := url.Values{}
	params.Set("period", intervalString)
	params.Set("symbol", symbol)
	var resp [][6]float64
	err = cr.SendHTTPRequest(ctx, exchange.RestSpot, marketKlines, request.Unset, &resp)
	if err != nil {
		return nil, err
	}
	response := make([]KlineItem, len(resp))
	for x := range resp {
		response[x] = KlineItem{
			Timestamp:    time.UnixMilli(int64(resp[x][0])),
			OpeningPrice: resp[x][1],
			HighestPrice: resp[x][2],
			MinimumPrice: resp[x][3],
			ClosingPrice: resp[x][4],
			Volume:       resp[x][5],
		}
	}
	return response, nil
}

// SendHTTPRequest send requests for un-authenticated market endpoints.
func (cr *Cryptodotcom) SendHTTPRequest(ctx context.Context, ePath exchange.URL, path string, f request.EndpointLimit, result interface{}) error {
	endpointPath, err := cr.API.Endpoints.GetURL(ePath)
	if err != nil {
		return err
	}
	response := &struct {
		Code    string          `json:"code"`
		Message string          `json:"msg"`
		Data    json.RawMessage `json:"data"`
	}{}
	println(endpointPath + cryptodotcomAPIVersion + path)
	err = cr.SendPayload(ctx, f, func() (*request.Item, error) {
		return &request.Item{
			Method:        http.MethodGet,
			Path:          endpointPath + cryptodotcomAPIVersion + path,
			Result:        response,
			Verbose:       cr.Verbose,
			HTTPDebugging: cr.HTTPDebugging,
			HTTPRecording: cr.HTTPRecording}, nil
	})
	if err != nil {
		return err
	}
	if response.Code != "0" || response.Message != "suc" {
		return fmt.Errorf("error code %s: Message: %s", response.Code, response.Message)
	}
	return json.Unmarshal(response.Data, &result)
}
