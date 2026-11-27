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

// orderTypeString returns the ordType an order of the type places with the time in force, refusing a time in force the
// order type cannot honour rather than placing an order that ignores it. Expiry instructions are left to the
// submission's validation. A trigger order's fill-or-kill or immediate-or-cancel instruction is sent as its
// advanceOrdType, which triggerAdvanceOrderType returns.
func orderTypeString(orderType order.Type, tif order.TimeInForce) (string, error) {
	if !tif.IsValid() {
		return "", fmt.Errorf("%w: %q", order.ErrInvalidTimeInForce, tif)
	}
	immediate := tif.Is(order.ImmediateOrCancel) || tif.Is(order.FillOrKill)
	switch orderType {
	case order.MarketMakerProtection:
		if tif.Is(order.PostOnly) {
			return orderMarketMakerProtectionAndPostOnly, nil
		}
		if immediate {
			return "", fmt.Errorf("%w: a market maker protection order rests on the book and cannot be %s", order.ErrUnsupportedTimeInForce, tif)
		}
		return orderMarketMakerProtection, nil
	case order.OptimalLimit:
		if tif != order.UnknownTIF && tif != order.ImmediateOrCancel {
			return "", fmt.Errorf("%w: an optimal limit order is immediate or cancel and cannot be %s", order.ErrUnsupportedTimeInForce, tif)
		}
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
		if tif.Is(order.PostOnly) {
			return "", fmt.Errorf("%w: a market order takes liquidity and cannot be %s", order.ErrUnsupportedTimeInForce, tif)
		}
		switch tif {
		case order.FillOrKill:
			return orderFOK, nil
		case order.ImmediateOrCancel:
			return orderIOC, nil
		}
		return orderMarket, nil
	case order.Trigger:
		if tif.Is(order.PostOnly) {
			return "", fmt.Errorf("%w: a trigger order places no post only order", order.ErrUnsupportedTimeInForce)
		}
		return orderTrigger, nil
	case order.Chase:
		// A chase order keeps a post only order at the best bid or ask, so it is post only already.
		if immediate {
			return "", fmt.Errorf("%w: a chase order rests on the book and cannot be %s", order.ErrUnsupportedTimeInForce, tif)
		}
		return orderChase, nil
	case order.TWAP, order.OCO, order.ConditionalStop, order.TrailingStop:
		if immediate || tif.Is(order.PostOnly) {
			return "", fmt.Errorf("%w: a %s order takes no %s instruction", order.ErrUnsupportedTimeInForce, orderType, tif)
		}
		switch orderType {
		case order.ConditionalStop:
			return orderConditional, nil
		case order.TrailingStop:
			return orderMoveOrderStop, nil
		}
		return orderType.Lower(), nil
	case order.LimitMaker:
		// A LimitMaker order must never take liquidity, so a time in force
		// that fills immediately contradicts it.
		if tif == order.ImmediateOrCancel || tif == order.FillOrKill {
			return "", fmt.Errorf("%w: %q with %q", order.ErrUnsupportedOrderType, orderType, tif)
		}
		return orderPostOnly, nil
	default:
		return "", fmt.Errorf("%w: %q", order.ErrUnsupportedOrderType, orderType)
	}
}

// triggerAdvanceOrderType returns the advanceOrdType a trigger order sends for its time in force: OKX places the
// triggered order as a fill-or-kill or immediate-or-cancel order at the trigger order's price, so one triggering at
// market is refused either instruction.
func triggerAdvanceOrderType(tif order.TimeInForce, orderPrice float64) (string, error) {
	var advanceOrderType string
	switch tif {
	case order.FillOrKill:
		advanceOrderType = orderFOK
	case order.ImmediateOrCancel:
		advanceOrderType = orderIOC
	default:
		return "", nil
	}
	if orderPrice == marketOrderPrice {
		return "", fmt.Errorf("%w: a trigger order executing at market cannot be %s", order.ErrUnsupportedTimeInForce, tif)
	}
	return advanceOrderType, nil
}

// triggerTimeInForce returns the time in force of a trigger order: the fill-or-kill or immediate-or-cancel its
// advanceOrdType gives the order it places once triggered, as triggerAdvanceOrderType sends them
func triggerTimeInForce(advanceOrderType string) order.TimeInForce {
	switch strings.ToLower(advanceOrderType) {
	case orderFOK:
		return order.FillOrKill
	case orderIOC:
		return order.ImmediateOrCancel
	default:
		return order.UnknownTIF
	}
}

// spreadOrderTypeString returns the ordType a spread order places for the
// order type and time in force. The spread endpoints document only market,
// limit, post_only and ioc: fok is rejected outright rather than silently
// downgraded to a resting limit order, as is a post only market order, and
// order types the spread book does not list, such as trigger-style orders, are
// rejected too.
func spreadOrderTypeString(orderType order.Type, tif order.TimeInForce) (string, error) {
	switch orderType {
	case order.Market:
		if tif.Is(order.PostOnly) {
			return "", fmt.Errorf("%w: a market order takes liquidity and cannot be %s", order.ErrUnsupportedTimeInForce, tif)
		}
		if tif == order.FillOrKill || !tif.IsValid() {
			return "", fmt.Errorf("%w: %q", order.ErrUnsupportedOrderType, orderType)
		}
		return orderMarket, nil // an ioc market order is already immediate
	case order.LimitMaker:
		// A LimitMaker order must never take liquidity, so it places as
		// post_only, and a time in force that fills immediately is refused.
		if tif.IsValid() && tif.Is(order.PostOnly) {
			return orderPostOnly, nil
		}
		switch tif {
		case order.UnknownTIF, order.GoodTillCancel, order.GoodTillDay:
			return orderPostOnly, nil
		}
	case order.Limit:
		// TimeInForce.IsValid lets PostOnly combine with flags such as
		// GoodTillCancel, and the order must still place as post-only.
		if tif.IsValid() && tif.Is(order.PostOnly) {
			return orderPostOnly, nil
		}
		switch tif {
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

// filteredOrderTypes are the OKX order types the ordType filters select from.
// elp is left out: OKX retires it on 31 October 2026 as the old name of rpi.
var filteredOrderTypes = []string{orderMarket, orderLimit, orderPostOnly, orderFOK, orderIOC, orderOptimalLimitIOC, orderMarketMakerProtection, orderMarketMakerProtectionAndPostOnly, orderOptionFOK, orderRPI}

// orderTypeFilter returns the ordType filter for the OKX order types that
// orderTypeFromString reads back as orderType, and as tif when one is set, as
// the comma-separated list OKX accepts: a limit order without a time in force
// spans limit, post_only, fok, ioc, op_fok and rpi. A time in force no pending
// order of the type reads back with, such as GoodTillDay on a limit order or
// ImmediateOrCancel on a market order, is refused, as any filter sent for it
// would select orders without it. LimitMaker selects the post_only orders
// listings report as it, so it takes no time in force but PostOnly, and an
// order type no OKX order type reads back as, such as an algo order type,
// falls back to the ordType orderTypeString places it as.
func orderTypeFilter(orderType order.Type, tif order.TimeInForce) (string, error) {
	if orderType == order.LimitMaker {
		placed, err := orderTypeString(orderType, tif)
		if err != nil {
			return "", err
		}
		if !filterTimeInForceMatches(order.Limit, order.PostOnly, tif&^order.GoodTillCancel) {
			return "", fmt.Errorf("%w: %s, a limit maker order reads back as post only", order.ErrUnsupportedTimeInForce, tif)
		}
		return placed, nil
	}
	var oTypes []string
	var readsBack bool
	for _, oType := range filteredOrderTypes {
		t, f, _ := orderTypeFromString(oType)
		if t != orderType {
			continue
		}
		readsBack = true
		if filterTimeInForceMatches(t, f, tif) {
			oTypes = append(oTypes, oType)
		}
	}
	switch {
	case len(oTypes) > 0:
		return strings.Join(oTypes, ","), nil
	case readsBack:
		return "", fmt.Errorf("%w: %s, no pending %s order reads back with it", order.ErrUnsupportedTimeInForce, tif, orderType)
	}
	return orderTypeString(orderType, tif)
}

// listingOrderTypeFilter returns the ordType filter an order listing sends:
// orderTypeFilter's, refusing a type and time in force no pending order reads
// back as, such as a market order with any time in force, as a cancel-all
// does, except that a listing of any order type narrows to its time in force
// alone
func listingOrderTypeFilter(orderType order.Type, tif order.TimeInForce) (string, error) {
	if orderType == order.AnyType {
		return timeInForceFilter(tif)
	}
	return orderTypeFilter(orderType, tif)
}

// timeInForceFilter returns the ordType filter for the OKX order types of every
// order type that orderTypeFromString reads back with the time in force, as
// the comma-separated list OKX accepts, refusing a time in force no pending
// order reads back with, such as GoodTillCancel
func timeInForceFilter(tif order.TimeInForce) (string, error) {
	var oTypes []string
	for _, oType := range filteredOrderTypes {
		if t, f, _ := orderTypeFromString(oType); f == tif || tif != order.UnknownTIF && filterTimeInForceMatches(t, f, tif) {
			oTypes = append(oTypes, oType)
		}
	}
	if len(oTypes) == 0 {
		return "", fmt.Errorf("%w: %s, no pending order reads back with it", order.ErrUnsupportedTimeInForce, tif)
	}
	return strings.Join(oTypes, ","), nil
}

// filterTimeInForceMatches reports whether a pending order read back as the
// given type and time in force matches a filter's time in force. OKX orders
// rest until cancelled unless their type says otherwise, so good till cancel
// matches a resting order read back with no time in force, and beside another
// flag selects by that flag.
func filterTimeInForceMatches(readType order.Type, readBack, requested order.TimeInForce) bool {
	if requested == order.UnknownTIF {
		return true
	}
	if !requested.IsValid() {
		return false
	}
	if !requested.Is(order.GoodTillCancel) {
		return readBack == requested
	}
	if others := requested &^ order.GoodTillCancel; others != order.UnknownTIF {
		return readBack == others
	}
	return readBack == order.UnknownTIF && readType != order.Market
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
// cannot be fill-or-kill. A listing of any order type narrows to its time in
// force as a limit listing does, as only spread limit orders read back with one.
func spreadOrderTypeFilter(orderType order.Type, tif order.TimeInForce) (string, error) {
	if orderType == order.AnyType {
		orderType = order.Limit
	}
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
		// The case is reachable: limit, post_only and ioc all read back as
		// Limit, so a query without a time in force cannot narrow to one wire
		// type. No filter is sent, leaving the request filter to narrow by
		// type, which keeps every matching row.
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

// GetInstrumentTypeFromAssetItem returns a string representation of asset.Item; which is an equivalent term for InstrumentType in Okx exchange
func GetInstrumentTypeFromAssetItem(a asset.Item) string {
	switch a {
	case asset.PerpetualSwap:
		return instTypeSwap
	case asset.Options:
		return instTypeOption
	default:
		return strings.ToUpper(a.String())
	}
}

// GetUnderlying returns the instrument ID for the corresponding asset pairs and asset type( Instrument Type )
func (e *Exchange) GetUnderlying(pair currency.Pair, a asset.Item) (string, error) {
	if !pair.IsPopulated() {
		return "", currency.ErrCurrencyPairsEmpty
	}
	format, err := e.GetPairFormat(a, false)
	if err != nil {
		return "", err
	}
	return pair.Base.String() + format.Delimiter + pair.Quote.String(), nil
}

// GetPairFromInstrumentID returns a currency pair give an instrument ID and asset Item, which represents the instrument type
func (e *Exchange) GetPairFromInstrumentID(instrumentID string) (currency.Pair, error) {
	codes := strings.Split(instrumentID, currency.DashDelimiter)
	if len(codes) >= 2 {
		instrumentID = codes[0] + currency.DashDelimiter + strings.Join(codes[1:], currency.DashDelimiter)
	}
	return currency.NewPairFromString(instrumentID)
}
