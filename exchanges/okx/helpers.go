package okx

import (
	"fmt"
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
	default:
		return order.UnknownType, order.UnknownTIF, fmt.Errorf("%w %q", order.ErrTypeIsInvalid, orderType)
	}
}

// orderTypeString returns a string representation of order.Type instance
func orderTypeString(orderType order.Type, tif order.TimeInForce) (string, error) {
	switch orderType {
	case order.MarketMakerProtection:
		if tif == order.PostOnly {
			return orderMarketMakerProtectionAndPostOnly, nil
		}
		return orderMarketMakerProtection, nil
	case order.OptimalLimit:
		return orderOptimalLimitIOC, nil
	case order.Limit:
		if tif == order.PostOnly {
			return orderPostOnly, nil
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
	case order.ConditionalStop:
		return orderConditional, nil
	case order.TrailingStop:
		return orderMoveOrderStop, nil
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
