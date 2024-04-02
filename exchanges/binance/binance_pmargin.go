package binance

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// Definitions and Terminology
// Portfolio Margin is an advanced trading mode offered by Binance, designed for experienced traders who seek
// increased leverage and flexibility across various trading products. It incorporates a unique approach to margin
// calculations and risk management to offer a more comprehensive assessment of the trader's overall exposure.

// - Terminology
// Margin refers to Cross Margin
// UM refers to USD-M Futures
// CM refers to Coin-M Futures

// NewUMOrder send in a new USDT margined order/orders.
func (b *Binance) NewUMOrder(ctx context.Context, arg *UMOrderParam) (*UMOrder, error) {
	return b.newUMCMOrder(ctx, arg, "/papi/v1/um/order")
}

// NewCMOrder send in a new Coin margined order/orders.
func (b *Binance) NewCMOrder(ctx context.Context, arg *UMOrderParam) (*UMOrder, error) {
	return b.newUMCMOrder(ctx, arg, "/papi/v1/cm/order")
}

func (b *Binance) newUMCMOrder(ctx context.Context, arg *UMOrderParam, path string) (*UMOrder, error) {
	if arg == nil || (*arg) == (UMOrderParam{}) {
		return nil, common.ErrNilPointer
	}
	if arg.Symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.OrderType == "" {
		return nil, order.ErrTypeIsInvalid
	}
	arg.OrderType = strings.ToUpper(arg.OrderType)
	if arg.OrderType == "limit" {
		if arg.TimeInForce == "" {
			return nil, errTimestampInfoRequired
		}
		if arg.Quantity <= 0 {
			return nil, order.ErrAmountBelowMin
		}
		if arg.Price <= 0 {
			return nil, order.ErrPriceBelowMin
		}
	} else if arg.OrderType == "MARKET" {
		if arg.Quantity <= 0 {
			return nil, order.ErrAmountBelowMin
		}
	} else {
		return nil, order.ErrUnsupportedOrderType
	}
	params := url.Values{}
	params.Set("symbol", arg.Symbol)
	params.Set("side", arg.Side)
	params.Set("type", arg.OrderType)
	if arg.PositionSide != "" {
		params.Set("positionSide", arg.PositionSide)
	}
	if arg.TimeInForce != "" {
		params.Set("timeInForce", arg.TimeInForce)
	}
	if arg.Quantity > 0 {
		params.Set("quantity", strconv.FormatFloat(arg.Quantity, 'f', -1, 64))
	}
	if arg.ReduceOnly {
		params.Set("reduceOnly", "true")
	}
	if arg.Price > 0 {
		params.Set("price", strconv.FormatFloat(arg.Price, 'f', -1, 64))
	}
	if arg.NewClientOrderID != "" {
		params.Set("newClientOrderID", arg.NewClientOrderID)
	}
	if arg.NewOrderRespType != "" {
		params.Set("newOrderRespType", arg.NewOrderRespType)
	}
	if arg.SelfTradePreventionMode != "" {
		params.Set("selfTradePreventionMode", arg.SelfTradePreventionMode)
	}
	if arg.GoodTillDate > 0 {
		params.Set("goodTillDate", strconv.FormatInt(arg.GoodTillDate, 10))
	}
	var resp *UMOrder
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestOptions, http.MethodPost, path, params, spotDefaultRate, &resp)
}
