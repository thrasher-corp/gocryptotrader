package bybit

import (
	"context"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
)

const (

	// public endpoint
	usdcfuturesGetOrderbook = "/perpetual/usdc/openapi/public/v1/order-book"
	usdcfuturesGetContracts = "/perpetual/usdc/openapi/public/v1/symbols"
	usdcfuturesGetSymbols   = "/perpetual/usdc/openapi/public/v1/tick"
	usdcfutures             = ""

	// auth endpoint
	usdcfuturesPlaceOrder = "/perpetual/usdc/openapi/private/v1/place-order"
)

// GetUSDCFuturesOrderbook gets orderbook data for USDCMarginedFutures.
func (by *Bybit) GetUSDCFuturesOrderbook(ctx context.Context, symbol currency.Pair) (Orderbook, error) {
	var resp Orderbook
	data := struct {
		Result []USDCOrderbookData `json:"result"`
		Error
	}{}

	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDCMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)

	err = by.SendHTTPRequest(ctx, exchange.RestUSDCMargined, common.EncodeURLValues(usdcfuturesGetOrderbook, params), publicFuturesRate, &data)
	if err != nil {
		return resp, err
	}

	for x := range data.Result {
		switch data.Result[x].Side {
		case sideBuy:
			resp.Bids = append(resp.Bids, orderbook.Item{
				Price:  data.Result[x].Price,
				Amount: data.Result[x].Size,
			})
		case sideSell:
			resp.Asks = append(resp.Asks, orderbook.Item{
				Price:  data.Result[x].Price,
				Amount: data.Result[x].Size,
			})
		default:
			return resp, errInvalidSide
		}
	}
	return resp, nil
}

// GetUSDCContracts gets all contract information for USDCMarginedFutures.
func (by *Bybit) GetUSDCContracts(ctx context.Context, symbol currency.Pair, direction string, limit int64) ([]USDCContract, error) {
	resp := struct {
		Data []USDCContract `json:"result"`
		Error
	}{}

	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := by.FormatSymbol(symbol, asset.USDCMarginedFutures)
		if err != nil {
			return resp.Data, err
		}
		params.Set("symbol", symbolValue)
	}

	if direction != "" {
		params.Set("direction", direction)
	}

	if limit <= 0 || limit > 200 {
		limit = 200
	}
	params.Set("limit", strconv.FormatInt(limit, 10))

	return resp.Data, by.SendHTTPRequest(ctx, exchange.RestUSDCMargined, common.EncodeURLValues(usdcfuturesGetContracts, params), publicFuturesRate, &resp)
}

// GetUSDCSymbols gets all symbol information for USDCMarginedFutures.
func (by *Bybit) GetUSDCSymbols(ctx context.Context, symbol currency.Pair) (USDCSymbol, error) {
	resp := struct {
		Data USDCSymbol `json:"result"`
		Error
	}{}

	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := by.FormatSymbol(symbol, asset.USDCMarginedFutures)
		if err != nil {
			return resp.Data, err
		}
		params.Set("symbol", symbolValue)
	} else {
		return USDCSymbol{}, errSymbolMissing
	}

	return resp.Data, by.SendHTTPRequest(ctx, exchange.RestUSDCMargined, common.EncodeURLValues(usdcfuturesGetSymbols, params), publicFuturesRate, &resp)
}
