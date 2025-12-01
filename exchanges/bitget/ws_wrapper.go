package bitget

import (
	"context"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// WebsocketSubmitOrder submits an order to the exchange via a websocket connection
func (e *Exchange) WebsocketSubmitOrder(ctx context.Context, submit *order.Submit) (*order.SubmitResponse, error) {
	if err := submit.Validate(e.GetTradingRequirements()); err != nil { // TODO: rm validate function as its just doubling up on checks
		return nil, err
	}

	tif, err := strategyTruthTable(submit.TimeInForce)
	if err != nil {
		return nil, err
	}

	oType, err := formatOrderType(submit.Type)
	if err != nil {
		return nil, err
	}

	side, err := formatOrderSide(submit.Side)
	if err != nil {
		return nil, err
	}

	switch submit.AssetType {
	case asset.Spot:
		amount := submit.Amount
		if side == "buy" && oType == "market" {
			if submit.QuoteAmount <= 0 {
				return nil, fmt.Errorf("%w: quote amount must be set for market buy orders", order.ErrAmountMustBeSet)
			}
			amount = submit.QuoteAmount
		}
		resp, err := e.WebsocketSpotPlaceOrder(ctx, &WebsocketSpotPlaceOrderRequest{
			Pair:          submit.Pair,
			OrderType:     oType,
			Side:          side,
			Size:          amount,
			TimeInForce:   tif,
			Price:         submit.Price,
			ClientOrderID: submit.ClientOrderID,
		})
		if err != nil {
			return nil, err
		}
		sr, err := submit.DeriveSubmitResponse(resp.Params.OrderID)
		if err != nil {
			return nil, err
		}
		sr.ClientOrderID = resp.Params.ClientOrderID
		return sr, nil
	case asset.USDTMarginedFutures, asset.CoinMarginedFutures, asset.USDCMarginedFutures:
		if submit.SettlementCurrency.IsEmpty() {
			return nil, fmt.Errorf("%w: %s", currency.ErrCurrencyCodeEmpty, "settlement currency must be set for futures orders")
		}
		if !submit.MarginType.Valid() {
			return nil, fmt.Errorf("%w: %q", margin.ErrInvalidMarginType, submit.MarginType)
		}

		// TODO:
		// * Determine trade side for hedge-mode or one-way-position mode. For now assume hedge-mode, restriction can come later
		// * Link take profit and stop loss values
		// * Self trade prevention implementation
		tradeSide := "open"
		if submit.ReduceOnly {
			tradeSide = "close"
		}
		resp, err := e.WebsocketFuturesPlaceOrder(ctx, &WebsocketFuturesOrderRequest{
			Contract:       submit.Pair,
			InstrumentType: itemEncoder(submit.AssetType),
			OrderType:      oType,
			Side:           side,
			ContractSize:   submit.Amount,
			TimeInForce:    tif,
			Price:          submit.Price,
			ClientOrderID:  submit.ClientOrderID,
			MarginCoin:     submit.SettlementCurrency,
			MarginMode:     submit.MarginType.String(),
			TradeSide:      tradeSide,
		})
		if err != nil {
			return nil, err
		}
		sr, err := submit.DeriveSubmitResponse(resp.ID)
		if err != nil {
			return nil, err
		}
		return sr, nil
	default:
		return nil, fmt.Errorf("%w: %q", asset.ErrNotSupported, submit.AssetType)
	}
}

// WebsocketCancelOrder cancels an order via the websocket connection
func (e *Exchange) WebsocketCancelOrder(ctx context.Context, cancel *order.Cancel) error {
	if err := cancel.Validate(); err != nil { // TODO: rm validate function as its just doubling up on checks
		return err
	}
	switch cancel.AssetType {
	case asset.Spot:
		_, err := e.WebsocketSpotCancelOrder(ctx, cancel.Pair, cancel.OrderID, cancel.ClientOrderID)
		return err
	case asset.USDTMarginedFutures, asset.CoinMarginedFutures, asset.USDCMarginedFutures:
		_, err := e.WebsocketFuturesCancelOrder(ctx, cancel.Pair, itemEncoder(cancel.AssetType), cancel.OrderID, cancel.ClientOrderID)
		return err
	default:
		return fmt.Errorf("%w: %q", asset.ErrNotSupported, cancel.AssetType)
	}
}

func formatOrderType(o order.Type) (string, error) {
	if lc := o.Lower(); lc == "limit" || lc == "market" {
		return lc, nil
	}
	return "", fmt.Errorf("%w: %q", order.ErrTypeIsInvalid, o)
}

func formatOrderSide(s order.Side) (string, error) {
	if s.IsLong() {
		return "buy", nil
	}
	if s.IsShort() {
		return "sell", nil
	}
	return "", fmt.Errorf("%w: %q", order.ErrSideIsInvalid, s)
}
