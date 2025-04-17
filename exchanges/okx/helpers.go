package okx

import (
	"fmt"
	"slices"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// orderTypeFromString returns order.Type instance from string
func orderTypeFromString(orderType string) (order.Type, order.TimeInForce, error) {
	orderType = strings.ToLower(orderType)
	switch orderType {
	case orderMarket:
		return order.Market, order.UnsetTIF, nil
	case orderLimit:
		return order.Limit, order.UnsetTIF, nil
	case orderPostOnly:
		return order.Limit, order.PostOnly, nil
	case orderFOK:
		return order.Limit, order.FillOrKill, nil
	case orderIOC:
		return order.Limit, order.ImmediateOrCancel, nil
	case orderOptimalLimitIOC:
		return order.OptimalLimit, order.ImmediateOrCancel, nil
	case "mmp":
		return order.MarketMakerProtection, order.UnsetTIF, nil
	case "mmp_and_post_only":
		return order.MarketMakerProtection, order.PostOnly, nil
	case "twap":
		return order.TWAP, order.UnsetTIF, nil
	case "move_order_stop":
		return order.TrailingStop, 0, nil
	case "chase":
		return order.Chase, order.UnsetTIF, nil
	default:
		return order.UnknownType, order.UnsetTIF, fmt.Errorf("%w %v", order.ErrTypeIsInvalid, orderType)
	}
}

// orderTypeString returns a string representation of order.Type instance
func orderTypeString(orderType order.Type, tif order.TimeInForce) (string, error) {
	switch orderType {
	case order.MarketMakerProtection:
		if tif == order.PostOnly {
			return "mmp_and_post_only", nil
		}
		return "mmp", nil
	case order.OptimalLimit:
		return "optimal_limit_ioc", nil
	case order.Limit:
		if tif == order.ImmediateOrCancel {
			return orderIOC, nil
		}
		return orderType.Lower(), nil
	case order.Market,
		order.Trigger,
		order.Chase,
		order.TWAP,
		order.OCO:
		return orderType.Lower(), nil
	case order.ConditionalStop:
		return "conditional", nil
	case order.TrailingStop:
		return "move_order_stop", nil
	default:
		switch tif {
		case order.PostOnly:
			return orderPostOnly, nil
		case order.FillOrKill:
			return orderFOK, nil
		case order.ImmediateOrCancel:
			return orderIOC, nil
		}
		return "", fmt.Errorf("%w: `%v`", order.ErrUnsupportedOrderType, orderType)
	}
}

// getAssetsFromInstrumentID parses an instrument ID and returns a list of assets types
// that the instrument is associated with
func (ok *Okx) getAssetsFromInstrumentID(instrumentID string) ([]asset.Item, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	pf, err := ok.CurrencyPairs.GetFormat(asset.Spot, true)
	if err != nil {
		return nil, err
	}
	splitSymbol := strings.Split(instrumentID, pf.Delimiter)
	if len(splitSymbol) <= 1 {
		return nil, fmt.Errorf("%w %v", currency.ErrCurrencyNotSupported, instrumentID)
	}
	pair, err := currency.NewPairDelimiter(instrumentID, pf.Delimiter)
	if err != nil {
		return nil, fmt.Errorf("%w: `%s`", err, instrumentID)
	}
	switch {
	case len(splitSymbol) == 2:
		resp := make([]asset.Item, 0, 2)
		enabled, err := ok.IsPairEnabled(pair, asset.Spot)
		if err != nil {
			return nil, err
		}
		if enabled {
			resp = append(resp, asset.Spot)
		}
		enabled, err = ok.IsPairEnabled(pair, asset.Margin)
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
		enabled, err := ok.IsPairEnabled(pair, aType)
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

func (ok *Okx) validatePlaceOrderParams(arg *PlaceOrderRequestParam) error {
	if arg == nil {
		return common.ErrNilPointer
	}
	if arg.InstrumentID == "" {
		return errMissingInstrumentID
	}
	if arg.AssetType == asset.Spot || arg.AssetType == asset.Margin || arg.AssetType == asset.Empty {
		arg.Side = strings.ToLower(arg.Side)
		if arg.Side != order.Buy.Lower() && arg.Side != order.Sell.Lower() {
			return fmt.Errorf("%w %s", order.ErrSideIsInvalid, arg.Side)
		}
	}
	if !slices.Contains([]string{"", TradeModeCross, TradeModeIsolated, TradeModeCash}, arg.TradeMode) {
		return fmt.Errorf("%w %s", errInvalidTradeModeValue, arg.TradeMode)
	}
	if arg.AssetType == asset.Futures || arg.AssetType == asset.PerpetualSwap {
		arg.PositionSide = strings.ToLower(arg.PositionSide)
		if !slices.Contains([]string{"long", "short"}, arg.PositionSide) {
			return fmt.Errorf("%w: `%s`, 'long' or 'short' supported", order.ErrSideIsInvalid, arg.PositionSide)
		}
	}
	arg.OrderType = strings.ToLower(arg.OrderType)
	if !slices.Contains([]string{orderMarket, orderLimit, orderPostOnly, orderFOK, orderIOC, orderOptimalLimitIOC, "mmp", "mmp_and_post_only"}, arg.OrderType) {
		return fmt.Errorf("%w: '%v'", order.ErrTypeIsInvalid, arg.OrderType)
	}
	if arg.Amount <= 0 {
		return order.ErrAmountBelowMin
	}
	if !slices.Contains([]string{"", "base_ccy", "quote_ccy"}, arg.QuantityType) {
		return errCurrencyQuantityTypeRequired
	}
	return nil
}

// assetTypeString returns a string representation of asset type
func assetTypeString(assetType asset.Item) (string, error) {
	switch assetType {
	case asset.Spot:
		return "SPOT", nil
	case asset.Margin:
		return "MARGIN", nil
	case asset.Futures:
		return "FUTURES", nil
	case asset.Options:
		return "OPTION", nil
	case asset.PerpetualSwap:
		return "SWAP", nil
	default:
		return "", asset.ErrNotSupported
	}
}
