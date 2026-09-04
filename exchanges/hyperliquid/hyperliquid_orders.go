package hyperliquid

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
)

const (
	marketPriceSignificantFigures = 5
	perpetualPriceDecimalBase     = 6
	spotPriceDecimalBase          = 8
	defaultTriggerSlippage        = 0.1
	orderGroupingNone             = "na"
	orderGroupingNormalTPSL       = "normalTpsl"
	orderStatusWaitingForFill     = "waitingForFill"
	orderStatusWaitingForTrigger  = "waitingForTrigger"
	wireTimeInForceALO            = "Alo"
	wireTimeInForceGTC            = "Gtc"
	wireTimeInForceIOC            = "Ioc"
)

var (
	errActionStatusCount         = errors.New("unexpected exchange action status count")
	errActionStatusMalformed     = errors.New("malformed exchange action status")
	errCrossMarginUnavailable    = errors.New("cross margin is unavailable for an isolated-only market")
	errGroupedOrderChildFailure  = errors.New("one or more grouped TP/SL child orders were rejected")
	errInvalidFilledSize         = errors.New("invalid filled order size")
	errInvalidMarketPrice        = errors.New("invalid market price")
	errInvalidLeverage           = errors.New("leverage must be a positive whole number within the market maximum")
	errMarketMidPriceNotFound    = errors.New("market mid price not found")
	errOrderNotModifiable        = errors.New("order is not open for modification")
	errPricePrecision            = errors.New("price exceeds market precision")
	errRiskManagementUnsupported = errors.New("unsupported risk management configuration")
	errSizePrecision             = errors.New("size exceeds market precision")
	errSlippageTolerance         = errors.New("market order slippage tolerance must be greater than 0 and less than 1")
	errTriggerOrderReduceOnly    = errors.New("trigger orders must be reduce-only perpetual orders")
	errTriggerPriceRequired      = errors.New("trigger price must be set")
	errUnsupportedOrderStatus    = errors.New("unsupported order status")
	errUnsupportedTimeInForce    = errors.New("unsupported time in force")
)

func formatOrderSize(size float64, sizeDecimals uint64) (string, error) {
	if size <= 0 || sizeDecimals > 8 {
		return "", fmt.Errorf("%w: size %v with %d decimals", errSizePrecision, size, sizeDecimals)
	}
	scale := math.Pow10(int(sizeDecimals))
	if math.Abs(math.RoundToEven(size*scale)/scale-size) >= 1e-12 {
		return "", fmt.Errorf("%w: %v allows %d decimals", errSizePrecision, size, sizeDecimals)
	}
	return floatToWire(size)
}

func deriveFilledOrderState(requested, filled float64, sizeDecimals uint64, timeInForce string) (order.Status, float64, error) {
	if _, err := formatOrderSize(requested, sizeDecimals); err != nil {
		return order.UnknownStatus, 0, fmt.Errorf("%w: invalid requested size: %w", errInvalidFilledSize, err)
	}
	if _, err := formatOrderSize(filled, sizeDecimals); err != nil {
		return order.UnknownStatus, 0, fmt.Errorf("%w: invalid reported size: %w", errInvalidFilledSize, err)
	}
	scale := math.Pow10(int(sizeDecimals)) //nolint:gosec // formatOrderSize validated sizeDecimals in the inclusive range 0..8.
	requestedUnits := math.Round(requested * scale)
	filledUnits := math.Round(filled * scale)
	if filledUnits > requestedUnits {
		return order.UnknownStatus, 0, fmt.Errorf("%w: reported %v exceeds requested %v", errInvalidFilledSize, filled, requested)
	}
	if filledUnits == requestedUnits {
		return order.Filled, 0, nil
	}
	remaining := (requestedUnits - filledUnits) / scale
	switch timeInForce {
	case wireTimeInForceGTC:
		return order.PartiallyFilled, remaining, nil
	case wireTimeInForceIOC:
		return order.PartiallyFilledCancelled, remaining, nil
	default:
		return order.UnknownStatus, 0, fmt.Errorf("%w: partial fill for %q time in force", errActionStatusMalformed, timeInForce)
	}
}

func validateLimitPrice(price float64, a asset.Item, sizeDecimals uint64) error {
	if price <= 0 || math.IsNaN(price) || math.IsInf(price, 0) {
		return errInvalidMarketPrice
	}
	var decimalBase uint64
	switch a {
	case asset.Spot:
		decimalBase = spotPriceDecimalBase
	case asset.PerpetualContract:
		decimalBase = perpetualPriceDecimalBase
	default:
		return fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	if sizeDecimals > decimalBase {
		return fmt.Errorf("%w: %s prices allow at most %d size decimals, got %d", errSizePrecision, a, decimalBase, sizeDecimals)
	}
	wire, err := floatToWire(price)
	if err != nil {
		return err
	}
	whole, fractional, hasFraction := strings.Cut(wire, ".")
	if !hasFraction {
		return nil // Hyperliquid permits integer prices with any number of significant figures.
	}
	if uint64(len(fractional)) > decimalBase-sizeDecimals {
		return fmt.Errorf("%w: %s allows at most %d price decimals", errPricePrecision, a, decimalBase-sizeDecimals)
	}
	significant := strings.TrimLeft(whole+fractional, "0")
	if len(significant) > marketPriceSignificantFigures {
		return fmt.Errorf("%w: got %d significant figures", errPricePrecision, len(significant))
	}
	return nil
}

func roundMarketPrice(price float64, a asset.Item, sizeDecimals uint64) (float64, error) {
	if price <= 0 || math.IsNaN(price) || math.IsInf(price, 0) {
		return 0, errInvalidMarketPrice
	}
	var decimalBase uint64
	switch a {
	case asset.Spot:
		decimalBase = spotPriceDecimalBase
	case asset.PerpetualContract:
		decimalBase = perpetualPriceDecimalBase
	default:
		return 0, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	if sizeDecimals > decimalBase {
		return 0, fmt.Errorf("%w: %s prices allow at most %d size decimals, got %d", errSizePrecision, a, decimalBase, sizeDecimals)
	}
	significantText := strconv.FormatFloat(price, 'g', marketPriceSignificantFigures, 64)
	significantPrice, _ := strconv.ParseFloat(significantText, 64)
	scale := math.Pow10(int(decimalBase - sizeDecimals)) //nolint:gosec // The checked difference is in the inclusive range 0..8.
	rounded := math.RoundToEven(significantPrice*scale) / scale
	if rounded <= 0 {
		return 0, errInvalidMarketPrice
	}
	return rounded, nil
}

func formatOrderTimeInForce(timeInForce order.TimeInForce) (string, error) {
	switch timeInForce {
	case order.UnknownTIF, order.GoodTillCancel:
		return wireTimeInForceGTC, nil
	case order.PostOnly, order.GoodTillCancel | order.PostOnly:
		return wireTimeInForceALO, nil
	case order.ImmediateOrCancel:
		return wireTimeInForceIOC, nil
	default:
		return "", fmt.Errorf("%w: %s", errUnsupportedTimeInForce, timeInForce)
	}
}

func (e *Exchange) buildOrderWire(ctx context.Context, p currency.Pair, a asset.Item, orderType order.Type, side order.Side, timeInForce order.TimeInForce, amount, price, triggerPrice, slippage float64, reduceOnly bool, clientOrderID string) (orderWire, pairMapping, error) {
	mapping, err := e.getPairMapping(ctx, p, a)
	if err != nil {
		return orderWire{}, pairMapping{}, err
	}
	size, err := formatOrderSize(amount, mapping.sizeDecimals)
	if err != nil {
		return orderWire{}, mapping, err
	}
	if clientOrderID != "" {
		if err := validateClientOrderID(clientOrderID); err != nil {
			return orderWire{}, mapping, err
		}
		clientOrderID = strings.ToLower(clientOrderID)
	}
	var wireType orderTypeWire
	switch orderType {
	case order.Limit:
		if triggerPrice != 0 {
			return orderWire{}, mapping, fmt.Errorf("%w: trigger price requires a trigger order type", errRiskManagementUnsupported)
		}
		if err := validateLimitPrice(price, a, mapping.sizeDecimals); err != nil {
			return orderWire{}, mapping, err
		}
		tif, err := formatOrderTimeInForce(timeInForce)
		if err != nil {
			return orderWire{}, mapping, err
		}
		wireType.Limit = &limitOrderTypeWire{TimeInForce: tif}
	case order.Market:
		if triggerPrice != 0 {
			return orderWire{}, mapping, fmt.Errorf("%w: trigger price requires a trigger order type", errRiskManagementUnsupported)
		}
		if slippage <= 0 || slippage >= 1 {
			return orderWire{}, mapping, errSlippageTolerance
		}
		mids, err := e.GetAllMids(ctx, mapping.dex)
		if err != nil {
			return orderWire{}, mapping, err
		}
		mid, ok := mids[mapping.coin]
		if !ok || mid.Float64() <= 0 {
			return orderWire{}, mapping, fmt.Errorf("%w: %s", errMarketMidPriceNotFound, mapping.coin)
		}
		price = mid.Float64()
		if side.IsLong() {
			price *= 1 + slippage
		} else {
			price *= 1 - slippage
		}
		price, err = roundMarketPrice(price, a, mapping.sizeDecimals)
		if err != nil {
			return orderWire{}, mapping, err
		}
		wireType.Limit = &limitOrderTypeWire{TimeInForce: wireTimeInForceIOC}
	case order.Stop, order.StopLimit, order.StopMarket, order.TakeProfit, order.TakeProfitMarket:
		if a != asset.PerpetualContract || !reduceOnly {
			return orderWire{}, mapping, errTriggerOrderReduceOnly
		}
		if triggerPrice <= 0 {
			return orderWire{}, mapping, errTriggerPriceRequired
		}
		if err := validateLimitPrice(triggerPrice, a, mapping.sizeDecimals); err != nil {
			return orderWire{}, mapping, fmt.Errorf("invalid trigger price: %w", err)
		}
		isMarket := orderType == order.Stop || orderType == order.StopMarket || orderType == order.TakeProfitMarket
		if isMarket && price == 0 {
			if slippage <= 0 || slippage >= 1 {
				return orderWire{}, mapping, errSlippageTolerance
			}
			price = triggerPrice
			if side.IsLong() {
				price *= 1 + slippage
			} else {
				price *= 1 - slippage
			}
			price, err = roundMarketPrice(price, a, mapping.sizeDecimals)
			if err != nil {
				return orderWire{}, mapping, err
			}
		} else if err := validateLimitPrice(price, a, mapping.sizeDecimals); err != nil {
			return orderWire{}, mapping, err
		}
		triggerPriceWire, _ := floatToWire(triggerPrice) // Trigger validation guarantees a wire-safe number.
		tpsl := "sl"
		if orderType == order.TakeProfit || orderType == order.TakeProfitMarket {
			tpsl = "tp"
		}
		wireType.Trigger = &triggerOrderTypeWire{
			IsMarket:           isMarket,
			TriggerPrice:       triggerPriceWire,
			TakeProfitStopLoss: tpsl,
		}
	default:
		return orderWire{}, mapping, fmt.Errorf("%w: %s", order.ErrTypeIsInvalid, orderType)
	}
	priceWire, _ := floatToWire(price) // Limit/trigger validation and market rounding guarantee wire-safe precision.
	return orderWire{
		AssetID:       mapping.assetID,
		IsBuy:         side.IsLong(),
		Price:         priceWire,
		Size:          size,
		ReduceOnly:    reduceOnly,
		Type:          wireType,
		ClientOrderID: clientOrderID,
	}, mapping, nil
}

func (e *Exchange) buildOrderWires(ctx context.Context, submit *order.Submit) ([]orderWire, pairMapping, string, error) {
	switch submit.Type {
	case order.Stop, order.StopLimit, order.StopMarket, order.TakeProfit, order.TakeProfitMarket:
		if submit.TriggerPriceType != order.MarkPrice {
			return nil, pairMapping{}, "", fmt.Errorf("%w: Hyperliquid triggers use mark price", errRiskManagementUnsupported)
		}
	}
	parent, mapping, err := e.buildOrderWire(ctx,
		submit.Pair,
		submit.AssetType,
		submit.Type,
		submit.Side,
		submit.TimeInForce,
		submit.Amount,
		submit.Price,
		submit.TriggerPrice,
		submit.SlippageTolerance,
		submit.ReduceOnly,
		submit.ClientOrderID)
	if err != nil {
		return nil, mapping, "", err
	}
	risk := submit.RiskManagementModes
	if !risk.TakeProfit.Enabled && !risk.StopLoss.Enabled && !risk.StopEntry.Enabled {
		return []orderWire{parent}, mapping, orderGroupingNone, nil
	}
	if submit.AssetType != asset.PerpetualContract ||
		risk.StopEntry.Enabled ||
		(risk.Mode != "" && risk.Mode != orderGroupingNormalTPSL) {
		return nil, mapping, "", errRiskManagementUnsupported
	}
	wires := []orderWire{parent}
	for _, child := range []struct {
		riskManagement order.RiskManagement
		takeProfit     bool
	}{
		{riskManagement: risk.TakeProfit, takeProfit: true},
		{riskManagement: risk.StopLoss},
	} {
		if !child.riskManagement.Enabled {
			continue
		}
		if child.riskManagement.Price <= 0 {
			return nil, mapping, "", errTriggerPriceRequired
		}
		if child.riskManagement.TriggerPriceType != order.MarkPrice {
			return nil, mapping, "", fmt.Errorf("%w: Hyperliquid TP/SL triggers use mark price", errRiskManagementUnsupported)
		}
		var childType order.Type
		switch {
		case child.takeProfit && (child.riskManagement.OrderType == order.UnknownType || child.riskManagement.OrderType == order.Market || child.riskManagement.OrderType == order.TakeProfitMarket):
			childType = order.TakeProfitMarket
		case child.takeProfit && (child.riskManagement.OrderType == order.Limit || child.riskManagement.OrderType == order.TakeProfit):
			childType = order.TakeProfit
		case !child.takeProfit && (child.riskManagement.OrderType == order.UnknownType || child.riskManagement.OrderType == order.Market || child.riskManagement.OrderType == order.Stop || child.riskManagement.OrderType == order.StopMarket):
			childType = order.StopMarket
		case !child.takeProfit && (child.riskManagement.OrderType == order.Limit || child.riskManagement.OrderType == order.StopLimit):
			childType = order.StopLimit
		default:
			return nil, mapping, "", fmt.Errorf("%w: %s child type %s", errRiskManagementUnsupported, map[bool]string{true: "take-profit", false: "stop-loss"}[child.takeProfit], child.riskManagement.OrderType)
		}
		slippage := submit.SlippageTolerance
		if (childType == order.StopMarket || childType == order.TakeProfitMarket) && child.riskManagement.LimitPrice == 0 && slippage == 0 {
			slippage = defaultTriggerSlippage
		}
		childSide := order.Buy
		if submit.Side.IsLong() {
			childSide = order.Sell
		}
		childWire, _, err := e.buildOrderWire(ctx,
			submit.Pair,
			submit.AssetType,
			childType,
			childSide,
			order.UnknownTIF,
			submit.Amount,
			child.riskManagement.LimitPrice,
			child.riskManagement.Price,
			slippage,
			true,
			"")
		if err != nil {
			return nil, mapping, "", err
		}
		wires = append(wires, childWire)
	}
	return wires, mapping, orderGroupingNormalTPSL, nil
}

func parseOrderActionStatuses(response *exchangeActionResponse, expected int) ([]orderActionStatus, error) {
	if response == nil {
		return nil, common.ErrNilPointer
	}
	var actionData exchangeActionData
	if err := json.Unmarshal(response.Response, &actionData); err != nil {
		return nil, err
	}
	if len(actionData.Data.Statuses) == 1 && expected > 1 {
		var failure struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal(actionData.Data.Statuses[0], &failure); err == nil && failure.Error != "" {
			actionData.Data.Statuses = slices.Repeat(actionData.Data.Statuses, expected)
		}
	}
	if len(actionData.Data.Statuses) != expected {
		return nil, fmt.Errorf("%w: expected %d, got %d", errActionStatusCount, expected, len(actionData.Data.Statuses))
	}
	statuses := make([]orderActionStatus, len(actionData.Data.Statuses))
	for i := range actionData.Data.Statuses {
		var deferred string
		if err := json.Unmarshal(actionData.Data.Statuses[i], &deferred); err == nil {
			switch deferred {
			case orderStatusWaitingForFill, orderStatusWaitingForTrigger:
				statuses[i].Deferred = deferred
				continue
			default:
				return nil, fmt.Errorf("%w at index %d: unknown deferred status %q", errActionStatusMalformed, i, deferred)
			}
		}
		if err := json.Unmarshal(actionData.Data.Statuses[i], &statuses[i]); err != nil {
			return nil, err
		}
		variants := 0
		if statuses[i].Resting != nil {
			if statuses[i].Resting.OrderID == 0 {
				return nil, fmt.Errorf("%w at index %d: resting order ID is zero", errActionStatusMalformed, i)
			}
			variants++
		}
		if statuses[i].Filled != nil {
			if statuses[i].Filled.OrderID == 0 {
				return nil, fmt.Errorf("%w at index %d: filled order ID is zero", errActionStatusMalformed, i)
			}
			variants++
		}
		if statuses[i].Error != "" {
			variants++
		}
		if variants != 1 {
			return nil, fmt.Errorf("%w at index %d", errActionStatusMalformed, i)
		}
	}
	return statuses, nil
}

func (e *Exchange) submitOrder(ctx context.Context, submit *order.Submit) (*order.SubmitResponse, error) {
	if err := submit.Validate(protocol.TradingRequirements{}); err != nil {
		return nil, err
	}
	if _, _, err := e.getSigningCredentials(ctx); err != nil {
		return nil, err
	}
	wires, mapping, grouping, err := e.buildOrderWires(ctx, submit)
	if err != nil {
		return nil, err
	}
	action := orderAction{Type: "order", Orders: wires, Grouping: grouping}
	var response exchangeActionResponse
	if err := e.sendSignedAction(ctx, action, len(wires), &response); err != nil {
		return nil, err
	}
	statuses, err := parseOrderActionStatuses(&response, len(wires))
	if err != nil {
		return nil, err
	}
	if statuses[0].Error != "" {
		return nil, fmt.Errorf("%w: %s", order.ErrUnableToPlaceOrder, statuses[0].Error)
	}
	if statuses[0].Deferred != "" {
		return nil, fmt.Errorf("%w at index 0: parent order cannot be %s", errActionStatusMalformed, statuses[0].Deferred)
	}
	wireTimeInForce := ""
	if wires[0].Type.Limit != nil {
		wireTimeInForce = wires[0].Type.Limit.TimeInForce
	}
	var orderID uint64
	var status order.Status
	var remainingAmount float64
	switch {
	case statuses[0].Resting != nil:
		orderID = statuses[0].Resting.OrderID
		status = order.New
		remainingAmount = submit.Amount
	case statuses[0].Filled != nil:
		orderID = statuses[0].Filled.OrderID
		status, remainingAmount, err = deriveFilledOrderState(submit.Amount, statuses[0].Filled.TotalSize.Float64(), mapping.sizeDecimals, wireTimeInForce)
		if err != nil {
			return nil, err
		}
	}
	result, _ := submit.DeriveSubmitResponse(strconv.FormatUint(orderID, 10)) // A parsed action status always supplies a non-zero order ID.
	result.Status = status
	result.Price, _ = strconv.ParseFloat(wires[0].Price, 64) // buildOrderWire guarantees a valid wire-format number.
	result.TimeInForce = order.UnknownTIF
	if wireTimeInForce != "" {
		result.TimeInForce, _ = classifyHyperliquidTimeInForce(wireTimeInForce)
	}
	if statuses[0].Filled != nil {
		result.AverageExecutedPrice = statuses[0].Filled.AveragePrice.Float64()
	}
	result.RemainingAmount = remainingAmount
	var childErrors error
	for i := 1; i < len(statuses); i++ {
		if statuses[i].Error != "" {
			childErrors = common.AppendError(childErrors, fmt.Errorf("child %d: %s", i, statuses[i].Error))
		}
	}
	if childErrors != nil {
		result.SubmissionError = fmt.Errorf("%w: %w", errGroupedOrderChildFailure, childErrors)
	}
	return result, nil
}

func parseCancelActionStatuses(response *exchangeActionResponse, identifiers []string) (map[string]string, error) {
	if response == nil {
		return nil, common.ErrNilPointer
	}
	var actionData exchangeActionData
	if err := json.Unmarshal(response.Response, &actionData); err != nil {
		return nil, err
	}
	if len(actionData.Data.Statuses) != len(identifiers) {
		return nil, fmt.Errorf("%w: expected %d, got %d", errActionStatusCount, len(identifiers), len(actionData.Data.Statuses))
	}
	statuses := make(map[string]string, len(identifiers))
	var errs error
	for i := range actionData.Data.Statuses {
		var success string
		if err := json.Unmarshal(actionData.Data.Statuses[i], &success); err == nil && success != "" {
			statuses[identifiers[i]] = success
			continue
		}
		var failure struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal(actionData.Data.Statuses[i], &failure); err != nil {
			return nil, err
		}
		if failure.Error == "" {
			return nil, fmt.Errorf("%w at index %d", errActionStatusMalformed, i)
		}
		statuses[identifiers[i]] = failure.Error
		errs = common.AppendError(errs, fmt.Errorf("%s: %w: %s", identifiers[i], errActionResponse, failure.Error))
	}
	return statuses, errs
}

func (e *Exchange) cancelOrders(ctx context.Context, cancels []order.Cancel) (map[string]string, error) {
	if len(cancels) == 0 {
		return map[string]string{}, nil
	}
	if len(cancels) > maximumActionBatchSize {
		return nil, fmt.Errorf("%w: maximum %d, got %d", errActionBatchTooLarge, maximumActionBatchSize, len(cancels))
	}
	numericWires := make([]cancelWire, 0, len(cancels))
	numericIDs := make([]string, 0, len(cancels))
	clientWires := make([]cancelByClientOrderIDWire, 0, len(cancels))
	clientIDs := make([]string, 0, len(cancels))
	for i := range cancels {
		if err := cancels[i].Validate(cancels[i].PairAssetRequired()); err != nil {
			return nil, err
		}
		mapping, err := e.getPairMapping(ctx, cancels[i].Pair, cancels[i].AssetType)
		if err != nil {
			return nil, err
		}
		switch {
		case cancels[i].OrderID != "":
			id, err := strconv.ParseUint(cancels[i].OrderID, 10, 64)
			if err != nil || id == 0 {
				return nil, fmt.Errorf("%w: %q", order.ErrOrderIDNotSet, cancels[i].OrderID)
			}
			numericWires = append(numericWires, cancelWire{AssetID: mapping.assetID, OrderID: id})
			numericIDs = append(numericIDs, cancels[i].OrderID)
		case cancels[i].ClientOrderID != "":
			if err := validateClientOrderID(cancels[i].ClientOrderID); err != nil {
				return nil, err
			}
			clientOrderID := strings.ToLower(cancels[i].ClientOrderID)
			clientWires = append(clientWires, cancelByClientOrderIDWire{AssetID: mapping.assetID, ClientOrderID: clientOrderID})
			clientIDs = append(clientIDs, cancels[i].ClientOrderID)
		default:
			return nil, order.ErrOrderIDNotSet
		}
	}
	statuses := make(map[string]string, len(cancels))
	var errs error
	if len(numericWires) != 0 {
		var response exchangeActionResponse
		err := e.sendSignedAction(ctx, cancelAction{Type: "cancel", Cancels: numericWires}, len(numericWires), &response)
		if err == nil {
			var parsed map[string]string
			parsed, err = parseCancelActionStatuses(&response, numericIDs)
			maps.Copy(statuses, parsed)
		}
		errs = common.AppendError(errs, err)
	}
	if len(clientWires) != 0 {
		var response exchangeActionResponse
		err := e.sendSignedAction(ctx, cancelByClientOrderIDAction{Type: "cancelByCloid", Cancels: clientWires}, len(clientWires), &response)
		if err == nil {
			var parsed map[string]string
			parsed, err = parseCancelActionStatuses(&response, clientIDs)
			maps.Copy(statuses, parsed)
		}
		errs = common.AppendError(errs, err)
	}
	return statuses, errs
}

func classifyHyperliquidOrderStatus(status string) (order.Status, error) {
	switch status {
	case "open":
		return order.Open, nil
	case "filled":
		return order.Filled, nil
	case "canceled", "scheduledCancel":
		return order.Cancelled, nil
	case "triggered":
		return order.Closed, nil
	case "marginCanceled", "vaultWithdrawalCanceled", "openInterestCapCanceled", "selfTradeCanceled", "reduceOnlyCanceled", "siblingFilledCanceled", "delistedCanceled", "liquidatedCanceled":
		return order.Cancelled, nil
	case "rejected", "tickRejected", "minTradeNtlRejected", "perpMarginRejected", "reduceOnlyRejected", "badAloPxRejected", "iocCancelRejected", "badTriggerPxRejected", "marketOrderNoLiquidityRejected", "positionIncreaseAtOpenInterestCapRejected", "positionFlipAtOpenInterestCapRejected", "tooAggressiveAtOpenInterestCapRejected", "openInterestIncreaseRejected", "insufficientSpotBalanceRejected", "oracleRejected", "perpMaxPositionRejected":
		return order.Rejected, nil
	default:
		return order.UnknownStatus, fmt.Errorf("%w: %s", errUnsupportedOrderStatus, status)
	}
}

func classifyHyperliquidOrderType(orderType string, isTrigger bool) (order.Type, error) {
	lowerOrderType := strings.ToLower(orderType)
	switch {
	case isTrigger && strings.Contains(lowerOrderType, "take profit") && strings.Contains(lowerOrderType, "market"):
		return order.TakeProfitMarket, nil
	case isTrigger && strings.Contains(lowerOrderType, "take profit"):
		return order.TakeProfit, nil
	case isTrigger && strings.Contains(lowerOrderType, "market"):
		return order.StopMarket, nil
	case isTrigger && strings.Contains(lowerOrderType, "limit"):
		return order.StopLimit, nil
	case isTrigger:
		return order.Stop, nil
	case strings.Contains(lowerOrderType, "market"):
		return order.Market, nil
	case strings.Contains(lowerOrderType, "limit"):
		return order.Limit, nil
	default:
		return order.UnknownType, fmt.Errorf("%w: %s", order.ErrTypeIsInvalid, orderType)
	}
}

func classifyHyperliquidTimeInForce(timeInForce string) (order.TimeInForce, error) {
	switch strings.ToLower(timeInForce) {
	case "", "gtc":
		return order.GoodTillCancel, nil
	case "alo":
		return order.PostOnly, nil
	case "ioc", "frontendmarket":
		return order.ImmediateOrCancel, nil
	default:
		return order.UnknownTIF, fmt.Errorf("%w: %s", order.ErrInvalidTimeInForce, timeInForce)
	}
}

// SetLeverage changes a perpetual market on any registered DEX between cross
// or isolated margin and applies the requested whole-number leverage.
func (e *Exchange) SetLeverage(ctx context.Context, a asset.Item, p currency.Pair, marginType margin.Type, amount float64, _ order.Side) error {
	if a != asset.PerpetualContract {
		return fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	mapping, err := e.getPairMapping(ctx, p, a)
	if err != nil {
		return err
	}
	if amount <= 0 ||
		math.IsNaN(amount) ||
		math.IsInf(amount, 0) ||
		math.Trunc(amount) != amount ||
		mapping.maxLeverage == 0 ||
		amount > float64(mapping.maxLeverage) {
		return fmt.Errorf("%w: requested %v, maximum %d", errInvalidLeverage, amount, mapping.maxLeverage)
	}
	var isCross bool
	switch marginType {
	case margin.Unset, margin.Multi:
		if mapping.onlyIsolated {
			return fmt.Errorf("%w: %w", margin.ErrMarginTypeUnsupported, errCrossMarginUnavailable)
		}
		isCross = true
	case margin.Isolated:
	default:
		return fmt.Errorf("%w: %s", margin.ErrMarginTypeUnsupported, marginType)
	}
	var result exchangeActionResponse
	return e.sendSignedAction(ctx, updateLeverageAction{
		Type:     "updateLeverage",
		AssetID:  mapping.assetID,
		IsCross:  isCross,
		Leverage: uint64(amount),
	}, 1, &result)
}

func (e *Exchange) convertOrder(ctx context.Context, source *OpenOrder, status string, statusTimestamp time.Time) (order.Detail, error) {
	if source == nil {
		return order.Detail{}, common.ErrNilPointer
	}
	mapping, a, err := e.getPairMappingByCoin(ctx, source.Coin)
	if err != nil {
		return order.Detail{}, err
	}
	return e.convertOrderFromMapping(source, status, statusTimestamp, &mapping, a)
}

func (e *Exchange) convertOrderFromMapping(source *OpenOrder, status string, statusTimestamp time.Time, mapping *pairMapping, a asset.Item) (order.Detail, error) {
	if source == nil || mapping == nil {
		return order.Detail{}, common.ErrNilPointer
	}
	var side order.Side
	switch source.Side {
	case "A":
		side = order.Sell
	case "B":
		side = order.Buy
	default:
		return order.Detail{}, fmt.Errorf("%w: %q", order.ErrSideIsInvalid, source.Side)
	}
	orderType, err := classifyHyperliquidOrderType(source.OrderType, source.IsTrigger)
	if err != nil {
		return order.Detail{}, err
	}
	timeInForce, err := classifyHyperliquidTimeInForce(source.TimeInForce)
	if err != nil {
		return order.Detail{}, err
	}
	orderStatus, err := classifyHyperliquidOrderStatus(status)
	if err != nil {
		return order.Detail{}, err
	}
	amount := source.OriginalSize.Float64()
	if amount == 0 {
		amount = source.Size.Float64()
	}
	clientOrderID := ""
	if source.ClientOrderID != nil {
		clientOrderID = *source.ClientOrderID
	}
	lastUpdated := statusTimestamp.UTC()
	if lastUpdated.IsZero() {
		lastUpdated = source.Timestamp.Time().UTC()
	}
	return order.Detail{
		TimeInForce:     timeInForce,
		ReduceOnly:      source.ReduceOnly,
		Price:           source.LimitPrice.Float64(),
		Amount:          amount,
		TriggerPrice:    source.TriggerPrice.Float64(),
		ExecutedAmount:  max(0, amount-source.Size.Float64()),
		RemainingAmount: source.Size.Float64(),
		Exchange:        e.Name,
		OrderID:         strconv.FormatUint(source.OrderID, 10),
		ClientOrderID:   clientOrderID,
		Type:            orderType,
		Side:            side,
		Status:          orderStatus,
		AssetType:       a,
		Date:            source.Timestamp.Time().UTC(),
		LastUpdated:     lastUpdated,
		Pair:            mapping.pair,
	}, nil
}
