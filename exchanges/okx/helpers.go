package okx

import (
	"fmt"
	"slices"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// orderTypeFromString returns the order Type and TimeInForce for okx order type strings
func orderTypeFromString(orderType string) (order.Type, order.TimeInForce, error) {
	orderType = strings.ToLower(orderType)
	switch orderType {
	case orderMarket:
		return order.Market, order.UnknownTIF, nil
	case orderLimit:
		return order.Limit, order.UnknownTIF, nil
	case orderPostOnly:
		return order.Limit, order.PostOnly, nil
	case orderFOK:
		return order.Limit, order.FillOrKill, nil
	case orderIOC:
		return order.Limit, order.ImmediateOrCancel, nil
	case orderOptimalLimitIOC:
		return order.OptimalLimit, order.ImmediateOrCancel, nil
	case orderMarketMakerProtection:
		return order.MarketMakerProtection, order.UnknownTIF, nil
	case orderMarketMakerProtectionAndPostOnly:
		return order.MarketMakerProtection, order.PostOnly, nil
	case orderTWAP:
		return order.TWAP, order.UnknownTIF, nil
	case orderMoveOrderStop:
		return order.TrailingStop, order.UnknownTIF, nil
	case orderChase:
		return order.Chase, order.UnknownTIF, nil
	case orderRPI:
		return order.Limit, order.UnknownTIF, nil
	case orderELP:
		return order.Limit, order.UnknownTIF, nil
	case orderOptionFOK:
		return order.Limit, order.FillOrKill, nil
	default:
		return order.UnknownType, order.UnknownTIF, fmt.Errorf("%w %q", order.ErrTypeIsInvalid, orderType)
	}
}

// orderTypeString returns a string representation of order.Type instance
func orderTypeString(orderType order.Type, tif order.TimeInForce) (string, error) {
	switch orderType {
	case order.MarketMakerProtection:
		if tif.Is(order.PostOnly) {
			return orderMarketMakerProtectionAndPostOnly, nil
		}
		return orderMarketMakerProtection, nil
	case order.OptimalLimit:
		return orderOptimalLimitIOC, nil
	case order.Limit:
		// TimeInForce.IsValid lets PostOnly combine with flags such as
		// GoodTillCancel, and the order must still place as post-only.
		if tif.Is(order.PostOnly) {
			return orderPostOnly, nil
		}
		switch tif {
		case order.FillOrKill:
			return orderFOK, nil
		case order.ImmediateOrCancel:
			return orderIOC, nil
		}
		return orderLimit, nil
	case order.Market:
		switch tif {
		case order.FillOrKill:
			return orderFOK, nil
		case order.ImmediateOrCancel:
			return orderIOC, nil
		}
		return orderMarket, nil
	case order.Trigger,
		order.Chase,
		order.TWAP,
		order.OCO:
		return orderType.Lower(), nil
	case order.LimitMaker:
		// A LimitMaker order must never take liquidity, so a time in force
		// that fills immediately contradicts it.
		if tif == order.ImmediateOrCancel || tif == order.FillOrKill {
			return "", fmt.Errorf("%w: %q with %q", order.ErrUnsupportedOrderType, orderType, tif)
		}
		return orderPostOnly, nil
	case order.ConditionalStop:
		return orderConditional, nil
	case order.TrailingStop:
		return orderMoveOrderStop, nil
	case order.Stop, order.StopLimit, order.StopMarket, order.TakeProfit, order.TakeProfitMarket, order.TrailingStopLimit, order.Bracket, order.Liquidation:
		// A trigger order cannot ride the time-in-force fallback below: it
		// would reach OKX as a plain limit-style order with no trigger
		// attached.
		return "", fmt.Errorf("%w: %q", order.ErrUnsupportedOrderType, orderType)
	default:
		switch tif {
		case order.PostOnly:
			return orderPostOnly, nil
		case order.FillOrKill:
			return orderFOK, nil
		case order.ImmediateOrCancel:
			return orderIOC, nil
		}
		return "", fmt.Errorf("%w: %q", order.ErrUnsupportedOrderType, orderType)
	}
}

// spreadOrderTypeString returns the ordType a spread order places for the
// order type and time in force. The spread endpoints document only market,
// limit, post_only and ioc: fok is rejected outright rather than silently
// downgraded to a resting limit order, and order types the spread book does
// not list, such as trigger-style orders, are rejected too.
func spreadOrderTypeString(orderType order.Type, tif order.TimeInForce) (string, error) {
	switch orderType {
	case order.Market:
		if tif == order.FillOrKill {
			return "", fmt.Errorf("%w: %q", order.ErrUnsupportedOrderType, orderType)
		}
		return orderMarket, nil // an ioc market order is already immediate
	case order.LimitMaker:
		// A LimitMaker order must never take liquidity, so it places as
		// post_only, and a time in force that fills immediately is refused.
		switch tif {
		case order.UnknownTIF, order.GoodTillCancel, order.GoodTillDay, order.PostOnly:
			return orderPostOnly, nil
		}
	case order.Limit:
		switch tif {
		case order.PostOnly:
			return orderPostOnly, nil
		case order.ImmediateOrCancel:
			return orderIOC, nil
		case order.FillOrKill:
			return "", fmt.Errorf("%w: %q", order.ErrUnsupportedOrderType, orderType)
		case order.UnknownTIF, order.GoodTillCancel, order.GoodTillDay:
			return orderLimit, nil
		}
	}
	return "", fmt.Errorf("%w: %q", order.ErrUnsupportedOrderType, orderType)
}

// orderTypeFilter returns the ordType filter for the OKX order types that
// orderTypeFromString reads back as orderType, and as tif when one is set, as
// the comma-separated list OKX accepts: a limit order without a time in force
// spans limit, post_only, fok, ioc, op_fok and rpi. A pair no OKX order type
// reads back as, such as a limit order with GoodTillCancel, falls back to
// orderTypeString. elp is left out: OKX retires it on 31 October 2026 as the
// old name of rpi.
func orderTypeFilter(orderType order.Type, tif order.TimeInForce) (string, error) {
	var oTypes []string
	for _, oType := range []string{orderMarket, orderLimit, orderPostOnly, orderFOK, orderIOC, orderOptimalLimitIOC, orderMarketMakerProtection, orderMarketMakerProtectionAndPostOnly, orderOptionFOK, orderRPI} {
		if t, f, _ := orderTypeFromString(oType); t == orderType && (tif == order.UnknownTIF || f == tif) {
			oTypes = append(oTypes, oType)
		}
	}
	if len(oTypes) > 0 {
		return strings.Join(oTypes, ","), nil
	}
	return orderTypeString(orderType, tif)
}

// spreadOrderTypeFilter returns the ordType filter for the spread order
// endpoints, which document only market, limit, post_only and ioc as single
// values: the comma-separated lists the ordinary order endpoints accept are
// rejected with 51000 there. One matching type is sent as-is; several matches
// (limit, post_only and ioc all read back as Limit) send no filter and leave
// the request filter to narrow by type. No match filters on the type spread
// placement uses when it accepts the request, as a market order with
// ImmediateOrCancel places as market, and otherwise falls back to
// orderTypeString and rejects values outside the four spread types, such as
// the fok a market or limit order with FillOrKill maps to: spread orders
// cannot be fill-or-kill.
func spreadOrderTypeFilter(orderType order.Type, tif order.TimeInForce) (string, error) {
	spreadTypes := []string{orderMarket, orderLimit, orderPostOnly, orderIOC}
	var oTypes []string
	for _, oType := range spreadTypes {
		if t, f, _ := orderTypeFromString(oType); t == orderType && (tif == order.UnknownTIF || f == tif) {
			oTypes = append(oTypes, oType)
		}
	}
	switch len(oTypes) {
	case 0:
		if placed, err := spreadOrderTypeString(orderType, tif); err == nil {
			return placed, nil
		}
		fallback, err := orderTypeString(orderType, tif)
		if err != nil {
			return "", err
		}
		if !slices.Contains(spreadTypes, fallback) {
			return "", fmt.Errorf("%w: %q", order.ErrUnsupportedOrderType, fallback)
		}
		return fallback, nil
	case 1:
		return oTypes[0], nil
	default:
		return "", nil
	}
}

// getAssetsFromInstrumentID parses an instrument ID and returns a list of assets types
// that the instrument is associated with
func (e *Exchange) getAssetsFromInstrumentID(instrumentID string) ([]asset.Item, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	pf, err := e.CurrencyPairs.GetFormat(asset.Spot, true)
	if err != nil {
		return nil, err
	}
	splitSymbol := strings.Split(instrumentID, pf.Delimiter)
	if len(splitSymbol) <= 1 {
		return nil, fmt.Errorf("%w %v", currency.ErrCurrencyNotSupported, instrumentID)
	}
	pair, err := currency.NewPairDelimiter(instrumentID, pf.Delimiter)
	if err != nil {
		return nil, fmt.Errorf("%w: %q", err, instrumentID)
	}
	switch {
	case len(splitSymbol) == 2:
		resp := make([]asset.Item, 0, 2)
		enabled, err := e.IsPairEnabled(pair, asset.Spot)
		if err != nil {
			return nil, err
		}
		if enabled {
			resp = append(resp, asset.Spot)
		}
		enabled, err = e.IsPairEnabled(pair, asset.Margin)
		if err != nil {
			return nil, err
		}
		if enabled {
			resp = append(resp, asset.Margin)
		}
		if len(resp) > 0 {
			return resp, nil
		}
	case len(splitSymbol) > 2:
		var aType asset.Item
		switch strings.ToLower(splitSymbol[len(splitSymbol)-1]) {
		case "swap":
			aType = asset.PerpetualSwap
		case "c", "p":
			aType = asset.Options
		default:
			aType = asset.Futures
		}
		enabled, err := e.IsPairEnabled(pair, aType)
		if err != nil {
			return nil, err
		} else if enabled {
			return []asset.Item{aType}, nil
		}
	}
	return nil, fmt.Errorf("%w: no asset enabled with instrument ID `%v`", asset.ErrNotEnabled, instrumentID)
}

// assetTypeFromInstrumentType returns an asset Item instance given and Instrument Type string
func assetTypeFromInstrumentType(instrumentType string) (asset.Item, error) {
	switch strings.ToUpper(instrumentType) {
	case instTypeSwap, instTypeContract:
		return asset.PerpetualSwap, nil
	case instTypeSpot:
		return asset.Spot, nil
	case instTypeMargin:
		return asset.Margin, nil
	case instTypeFutures:
		return asset.Futures, nil
	case instTypeOption:
		return asset.Options, nil
	case "":
		return asset.Empty, nil
	default:
		return asset.Empty, asset.ErrNotSupported
	}
}

// assetTypeString returns a string representation of asset type
func assetTypeString(assetType asset.Item) (string, error) {
	switch assetType {
	case asset.Spot:
		return instTypeSpot, nil
	case asset.Margin:
		return instTypeMargin, nil
	case asset.Futures:
		return instTypeFutures, nil
	case asset.Options:
		return instTypeOption, nil
	case asset.PerpetualSwap:
		return instTypeSwap, nil
	default:
		return "", asset.ErrNotSupported
	}
}
