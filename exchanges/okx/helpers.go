package okx

import (
	"fmt"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var (
	spotAssetType          = []asset.Item{asset.Spot, asset.Margin}
	perpetualSwapAssetType = []asset.Item{asset.PerpetualSwap}
	futuresAssetType       = []asset.Item{asset.Futures}
	optionsAssetType       = []asset.Item{asset.Options}
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

// getAssetsFromInstrumentID infers the asset type from the OKX instrument ID shape.
func (*Exchange) getAssetsFromInstrumentID(instrumentID string) ([]asset.Item, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	const swapSuffixLength = len("-SWAP")
	if len(instrumentID) > swapSuffixLength {
		suffix := instrumentID[len(instrumentID)-swapSuffixLength:]
		if suffix == "-SWAP" || suffix == "-swap" {
			return perpetualSwapAssetType, nil
		}
	}

	splitInstrumentID := strings.Split(instrumentID, "-")
	switch len(splitInstrumentID) {
	case 2:
		return spotAssetType, nil
	case 3:
		return futuresAssetType, nil
	case 5:
		switch strings.ToUpper(splitInstrumentID[len(splitInstrumentID)-1]) {
		case "C", "P":
			return optionsAssetType, nil
		default:
			return nil, fmt.Errorf("%w: unsupported option instrument ID %q", asset.ErrNotSupported, instrumentID)
		}
	default:
		if len(splitInstrumentID) < 2 {
			return nil, fmt.Errorf("%w %v", currency.ErrCurrencyNotSupported, instrumentID)
		}
		return nil, fmt.Errorf("%w: unsupported OKX instrument ID %q", asset.ErrNotSupported, instrumentID)
	}
}

// assetTypeFromInstrumentType returns an asset Item instance given and Instrument Type string
func assetTypeFromInstrumentType(instrumentType string) (asset.Item, error) {
	switch instrumentType {
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
