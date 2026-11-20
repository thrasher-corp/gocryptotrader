package bingx

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"math"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/types"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

// Order request limits: the order history spans and page sizes each product's endpoint takes, and the furthest record
// spot order history pages reach
const (
	perpetualOrderHistorySpan    = 7*24*time.Hour - time.Millisecond
	perpetualOrderHistoryLimit   = 1000
	spotOrderHistoryPageSize     = 100
	spotOrderHistoryPagesPerSpan = spotOrderHistoryMaxPageWindow / spotOrderHistoryPageSize
)

var (
	errInvalidOrderID               = errors.New("invalid order ID")
	errInvalidPrice                 = errors.New("invalid price")
	errTriggerPriceRequired         = errors.New("trigger price required")
	errTrailingValueRequired        = errors.New("trailing stops need a positive tracking value")
	errCancelReplaceFailed          = errors.New("cancel-replace failed")
	errModifyAmountNotAboveExecuted = errors.New("modified amount is not above the amount already executed")
	errCancelledOrderExecuted       = errors.New("order cancelled after executing its desired total")
	errCancelOutcomeUnknown         = errors.New("cancellation outcome unknown")
	errOrderNotOpen                 = errors.New("order is no longer open")
	errBatchCancelMixedAssets       = errors.New("batch cancellation orders must share an asset")
	errBatchCancelMixedPairs        = errors.New("batch cancellation orders must share a pair")
	errBatchCancelResultMissing     = errors.New("cancellation result missing from the reply")
	errOrderHistoryLimit            = errors.New("more orders than BingX pages in the smallest time window")
	errOrderHistoryTruncated        = errors.New("order history truncated at the furthest page BingX serves")
	errInvalidOrderLeverage         = errors.New("invalid order leverage")
	errCoinMarginedReduceOnlyOneWay = errors.New("coin-M orders cannot be reduce only in one-way mode")
	errSpotReduceOnly               = errors.New("spot orders cannot be reduce only")
)

// positionModeCache holds the position mode of each USDT-M or coin-M account orders were placed for, keyed by the
// product's endpoint and API key, as context credentials can switch the account per call and a demo trading endpoint
// holds accounts of its own. A lookup keeps the mode it fetched only when the account's mode was not forgotten while it
// was in flight, so a lookup overlapping a change of mode cannot keep the mode the change replaced
type positionModeCache struct {
	mu          sync.Mutex
	hedgeMode   map[string]bool
	generations map[string]uint64
}

// get returns the account's cached mode, whether it has one, and the generation a lookup started now keeps its mode at
func (c *positionModeCache) get(account string) (hedgeMode, ok bool, generation uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	hedgeMode, ok = c.hedgeMode[account]
	return hedgeMode, ok, c.generations[account]
}

// set keeps the mode a lookup started at generation fetched, unless the account's mode was forgotten since
func (c *positionModeCache) set(account string, generation uint64, hedgeMode bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.generations[account] != generation {
		return
	}
	if c.hedgeMode == nil {
		c.hedgeMode = make(map[string]bool)
	}
	c.hedgeMode[account] = hedgeMode
}

// forget drops the account's cached mode and the mode of any lookup in flight
func (c *positionModeCache) forget(account string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.hedgeMode, account)
	if c.generations == nil {
		c.generations = make(map[string]uint64)
	}
	c.generations[account]++
}

// orderModificationLocks serialises the modifications of each order, so a modification looks the order up only once any
// other modification of it has finished, and finds an order another replaced closed rather than replacing it again. An
// order's entry is kept only while a modification holds or waits for its lock
type orderModificationLocks struct {
	mu    sync.Mutex
	locks map[orderModificationKey]*orderModificationLock
}

// orderModificationKey names the order a modification changes: its account on its product, as productAccount keys it,
// its asset, and its order ID, which modifiedOrderID resolves however the modification names the order
type orderModificationKey struct {
	account string
	asset   asset.Item
	orderID uint64
}

// orderModificationLock is the lock of one order: held is full while a modification holds it, and holders counts the
// modifications holding or waiting for it
type orderModificationLock struct {
	held    chan struct{}
	holders int
}

// lock waits until no other modification holds k's lock, or until ctx ends, and returns the function releasing it
func (l *orderModificationLocks) lock(ctx context.Context, k orderModificationKey) (unlock func(), err error) {
	l.mu.Lock()
	entry, ok := l.locks[k]
	if !ok {
		if l.locks == nil {
			l.locks = make(map[orderModificationKey]*orderModificationLock)
		}
		entry = &orderModificationLock{held: make(chan struct{}, 1)}
		l.locks[k] = entry
	}
	entry.holders++
	l.mu.Unlock()
	leave := func() {
		l.mu.Lock()
		defer l.mu.Unlock()
		entry.holders--
		if entry.holders == 0 {
			delete(l.locks, k)
		}
	}
	select {
	case entry.held <- struct{}{}:
	case <-ctx.Done():
		leave()
		return nil, ctx.Err()
	}
	return func() {
		<-entry.held
		leave()
	}, nil
}

// credentialsContext returns ctx carrying the credentials it names now, so every request an operation sends with it
// goes to one account although the exchange's default credentials change meanwhile
func (e *Exchange) credentialsContext(ctx context.Context) (context.Context, error) {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return nil, err
	}
	return accounts.DeployCredentialsToContext(ctx, creds), nil
}

// productAccount returns the key of ctx's account on the product whose endpoint ePath names, by which position modes and
// order modifications are kept
func (e *Exchange) productAccount(ctx context.Context, ePath exchange.URL) (string, error) {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return "", err
	}
	endpoint, err := e.API.Endpoints.GetURL(ePath)
	if err != nil {
		return "", err
	}
	return endpoint + " " + creds.Key, nil
}

// swapHedgeMode returns whether ctx's USDT-M account is in hedge mode, from the cache or else from Query Position Mode
func (e *Exchange) swapHedgeMode(ctx context.Context) (bool, error) {
	account, err := e.productAccount(ctx, exchange.RestUSDTMargined)
	if err != nil {
		return false, err
	}
	hedgeMode, ok, generation := e.positionModes.get(account)
	if ok {
		return hedgeMode, nil
	}
	resp, err := e.GetSwapPositionMode(ctx)
	if err != nil {
		return false, fmt.Errorf("error fetching position mode: %w", err)
	}
	e.positionModes.set(account, generation, resp.DualSidePosition.Bool())
	return resp.DualSidePosition.Bool(), nil
}

// forgetSwapPositionMode drops the cached position mode of ctx's USDT-M account, so the next order looks it up
func (e *Exchange) forgetSwapPositionMode(ctx context.Context) {
	if account, err := e.productAccount(ctx, exchange.RestUSDTMargined); err == nil {
		e.positionModes.forget(account)
	}
}

// coinMarginedHedgeMode returns whether ctx's coin-M account is taken to be in hedge mode. BingX offers no coin-M
// position mode query, so an account is taken to be in hedge mode until BingX refuses an order for its position side
func (e *Exchange) coinMarginedHedgeMode(ctx context.Context) (bool, error) {
	account, err := e.productAccount(ctx, exchange.RestCoinMargined)
	if err != nil {
		return false, err
	}
	hedgeMode, ok, _ := e.coinMarginedPositionModes.get(account)
	return hedgeMode || !ok, nil
}

// recordCoinMarginedPositionMode records ctx's coin-M account as in hedge mode or one-way mode, as an order BingX
// accepted or refused for its position side shows
func (e *Exchange) recordCoinMarginedPositionMode(ctx context.Context, hedgeMode bool) {
	account, err := e.productAccount(ctx, exchange.RestCoinMargined)
	if err != nil {
		return
	}
	_, _, generation := e.coinMarginedPositionModes.get(account)
	e.coinMarginedPositionModes.set(account, generation, hedgeMode)
}

// perpetualPositionSide returns the position side and reduce only flag a perpetual order is sent with. BingX takes an
// order without a position side as one on the LONG side, so every order carries one: BOTH in one-way mode, with reduce
// only as asked, and in hedge mode the side of the position the order opens, LONG for Buy or Long and SHORT for Sell
// or Short, or with ReduceOnly the side it closes, without reduceOnly, which hedge mode refuses. A USDT-M account's
// mode is looked up, while coin-M accounts, which have no position mode query, are taken to be in hedge mode until
// BingX refuses an order for its position side
func (e *Exchange) perpetualPositionSide(ctx context.Context, a asset.Item, side order.Side, reduceOnly bool) (PositionSide, bool, error) {
	var hedgeMode bool
	var err error
	if a == asset.USDTMarginedFutures {
		hedgeMode, err = e.swapHedgeMode(ctx)
	} else {
		hedgeMode, err = e.coinMarginedHedgeMode(ctx)
	}
	if err != nil {
		return "", false, err
	}
	if !hedgeMode {
		return PositionSideBoth, reduceOnly, nil
	}
	return hedgePositionSide(side, reduceOnly), false, nil
}

// hedgePositionSide returns the hedge mode position side an order trades: the side of the position it opens, or with
// reduceOnly the other side, the position it closes
func hedgePositionSide(side order.Side, reduceOnly bool) PositionSide {
	if side.IsLong() != reduceOnly {
		return PositionSideLong
	}
	return PositionSideShort
}

// isPositionModeRejection reports whether BingX refused a perpetual order, placing nothing, for a position side or
// reduce only flag the account's position mode does not take, which it reports as a parameter error naming them
func isPositionModeRejection(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Code != 109400 {
		return false
	}
	msg := strings.ToLower(apiErr.Message)
	for _, term := range []string{"positionside", "position side", "reduceonly", "reduce only", "position mode"} {
		if strings.Contains(msg, term) {
			return true
		}
	}
	return false
}

// parseOrderID converts an order ID into BingX's numeric form; an empty ID is 0, which requests leave out
func parseOrderID(orderID string) (uint64, error) {
	if orderID == "" {
		return 0, nil
	}
	id, err := strconv.ParseUint(orderID, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w %q: %w", errInvalidOrderID, orderID, err)
	}
	return id, nil
}

// remainingAmount returns an order's unfilled amount, in decimal arithmetic so it is as exact as BingX's amounts
func remainingAmount(original, executed types.Number) float64 {
	return max(original.Decimal().Sub(executed.Decimal()).InexactFloat64(), 0)
}

// parseLeverage returns the leverage of an order BingX reports as text such as 20X; empty text is no leverage
func parseLeverage(leverage string) (float64, error) {
	value := strings.TrimSuffix(strings.ToUpper(leverage), "X")
	if value == "" {
		return 0, nil
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("%w %q", errInvalidOrderLeverage, leverage)
	}
	return parsed, nil
}

// logOrderIssue logs a value of an order that could not be converted, which leaves that value unset: a listing still
// returns the order, and a placement or modification BingX accepted is not reported as failed
func (e *Exchange) logOrderIssue(a asset.Item, orderID string, err error) {
	log.Warnf(log.ExchangeSys, "%s %s order %s: %v", e.Name, a, orderID, err)
}

// SubmitOrder submits a new order. Spot market buys spend QuoteAmount and other spot orders trade Amount, or QuoteAmount
// for a limit order without one, and a spot order sized by QuoteAmount reports the base amount BingX's reply gives,
// or the amount it executed; spot stop orders are BingX's TAKE_STOP orders, which trigger in either direction.
// Perpetual orders trade Amount, which counts contracts on coin-M, and a bare Stop or TakeProfit is placed as its limit
// form when Price is set and its market form otherwise, triggered at TriggerPrice on the price TriggerPriceType names.
// USDT-M trailing stops trail by TrackingValue, a price distance or a percentage, from TriggerPrice or the latest
// price, and only USDT-M LIMIT and MARKET orders take a ClientOrderID, others being refused before anything is sent.
// Perpetual orders carry the position side perpetualPositionSide returns, and an order BingX refuses for its
// position mode is placed once more: a USDT-M one after looking the mode up again, and a coin-M one for the other mode,
// as coin-M has no position mode query. Coin-M orders have no reduce only flag, so a reduce only coin-M order is placed
// on the hedge mode side it closes whichever mode the account is taken to be in: BingX accepting it shows hedge mode,
// and refusing it for its position side shows one-way mode, which cannot mark it reduce only, and returns an error. A
// perpetual order's position mode lookup and placements all go to the account of the credentials ctx names when it is
// submitted. Orders carry no margin type, so a MarginType is refused other than a spot order's NoMargin, as is a reduce
// only spot order
func (e *Exchange) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if err := s.Validate(e.GetTradingRequirements()); err != nil {
		return nil, err
	}
	if s.Leverage != 0 && s.Leverage != 1 {
		return nil, fmt.Errorf("%w: received %v, set it with SetLeverage", order.ErrSubmitLeverageNotSupported, s.Leverage)
	}
	switch {
	case s.MarginType == margin.Unset, s.AssetType == asset.Spot && s.MarginType == margin.NoMargin:
	case s.AssetType == asset.Spot:
		return nil, fmt.Errorf("%w: spot orders are not margined, received %s", margin.ErrMarginTypeUnsupported, s.MarginType)
	default:
		return nil, fmt.Errorf("%w: received %s, set it with SetMarginType", margin.ErrMarginTypeUnsupported, s.MarginType)
	}
	if s.ReduceOnly && s.AssetType == asset.Spot {
		return nil, errSpotReduceOnly
	}
	if t := resolveOrderType(s.Type, s.Price); t != s.Type {
		// The response reports the type placed, which later reads of the order report
		resolved := *s
		resolved.Type = t
		s = &resolved
	}
	switch s.AssetType {
	case asset.Spot:
		return e.submitSpotOrder(ctx, s)
	case asset.USDTMarginedFutures, asset.CoinMarginedFutures:
	default:
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, s.AssetType)
	}
	ctx, err := e.credentialsContext(ctx)
	if err != nil {
		return nil, err
	}
	if s.AssetType == asset.USDTMarginedFutures {
		return e.submitSwapOrder(ctx, s)
	}
	return e.submitCoinMarginedOrder(ctx, s)
}

// spotOrderRequest returns the spot order a resolved submission places. A market buy spends QuoteAmount and other
// orders trade Amount, or a limit order without one QuoteAmount; priced types take Price, and stop types trigger at
// TriggerPrice
func spotOrderRequest(s *order.Submit) (*PlaceSpotOrderRequest, error) {
	side, err := formatOrderSide(s.Side)
	if err != nil {
		return nil, err
	}
	orderType, err := formatOrderType(asset.Spot, s.Type)
	if err != nil {
		return nil, err
	}
	timeInForce, err := formatTimeInForce(s.TimeInForce)
	if err != nil {
		return nil, err
	}
	req := &PlaceSpotOrderRequest{
		Symbol:           s.Pair,
		Side:             side,
		Type:             orderType,
		NewClientOrderID: s.ClientOrderID,
		TimeInForce:      timeInForce,
	}
	if orderTypeTakesPrice(orderType) {
		if s.Price <= 0 {
			return nil, order.ErrPriceMustBeSetIfLimitOrder
		}
		req.Price = s.Price
	}
	if orderTypeTakesTrigger(orderType) {
		if s.TriggerPrice <= 0 {
			return nil, errTriggerPriceRequired
		}
		req.StopPrice = s.TriggerPrice
	}
	switch {
	case orderType == OrderTypeMarket && side == SideBuy:
		req.QuoteOrderQuantity = s.QuoteAmount
	case s.Amount > 0:
		req.Quantity = s.Amount
	case orderType == OrderTypeLimit:
		req.QuoteOrderQuantity = s.QuoteAmount
	default:
		return nil, fmt.Errorf("%w: %s orders take a base amount", order.ErrAmountIsInvalid, orderType)
	}
	return req, nil
}

// submitSpotOrder places a spot order
func (e *Exchange) submitSpotOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	req, err := spotOrderRequest(s)
	if err != nil {
		return nil, err
	}
	placed, err := e.PlaceSpotOrder(ctx, req)
	if err != nil {
		return nil, err
	}
	resp, err := s.DeriveSubmitResponse(placed.OrderID.String())
	if err != nil {
		return nil, err
	}
	resp.ClientOrderID = placed.ClientOrderID
	resp.Date = placed.TransactTime.Time()
	resp.LastUpdated = resp.Date
	resp.Status = order.New
	e.setPlacedOrderStatus(&resp.Status, resp.AssetType, resp.OrderID, string(placed.Status))
	if req.Quantity == 0 {
		// An order sized in the quote currency has the base amount the reply reports, which the order store settles it by
		resp.Amount = cmp.Or(placed.OriginalQuantity, placed.ExecutedQuantity).Float64()
	}
	resp.RemainingAmount = remainingAmount(placed.OriginalQuantity, placed.ExecutedQuantity)
	if placed.ExecutedQuantity > 0 {
		resp.Cost = placed.CummulativeQuoteQuantity.Float64()
		resp.AverageExecutedPrice = placed.CummulativeQuoteQuantity.Decimal().Div(placed.ExecutedQuantity.Decimal()).InexactFloat64()
	}
	reportExecutedPlacementNew(&resp.Status, resp.Amount, placed.ExecutedQuantity)
	return resp, nil
}

// setPlacedOrderStatus sets status, a submitted order's or a replacement's, to the one its placement reply reports, and
// keeps it when the reply reports none or one that cannot be converted
func (e *Exchange) setPlacedOrderStatus(status *order.Status, a asset.Item, orderID, reported string) {
	if reported == "" {
		return
	}
	converted, err := convertOrderStatus(reported)
	if err != nil {
		e.logOrderIssue(a, orderID, err)
		return
	}
	*status = converted
}

// reportExecutedPlacementNew reports an order of amount BingX finished at placement after executing any of it as New,
// for the order manager to settle from a later query: submit and modify responses carry no executed amount, and BingX's
// placement replies carry no fills
func reportExecutedPlacementNew(status *order.Status, amount float64, executed types.Number) {
	if amount > 0 && executed > 0 && status.IsInactive() {
		*status = order.New
	}
}

// perpetualOrderValues returns the values a resolved perpetual submission shares across products: its side and type,
// its price and trigger where the type takes them, and its time in force
func perpetualOrderValues(s *order.Submit) (side Side, orderType OrderType, timeInForce TimeInForce, err error) {
	if s.Amount <= 0 {
		return "", "", "", fmt.Errorf("%w: perpetual orders take a base amount", order.ErrAmountIsInvalid)
	}
	if side, err = formatOrderSide(s.Side); err != nil {
		return "", "", "", err
	}
	if orderType, err = formatOrderType(s.AssetType, s.Type); err != nil {
		return "", "", "", err
	}
	if timeInForce, err = formatTimeInForce(s.TimeInForce); err != nil {
		return "", "", "", err
	}
	if orderTypeTakesPrice(orderType) && s.Price <= 0 {
		return "", "", "", order.ErrPriceMustBeSetIfLimitOrder
	}
	if orderTypeTakesTrigger(orderType) && s.TriggerPrice <= 0 {
		return "", "", "", errTriggerPriceRequired
	}
	return side, orderType, timeInForce, nil
}

// swapOrderRequest returns the USDT-M order a resolved submission places
func (e *Exchange) swapOrderRequest(ctx context.Context, s *order.Submit) (*SwapOrderRequest, error) {
	side, orderType, timeInForce, err := perpetualOrderValues(s)
	if err != nil {
		return nil, err
	}
	// Checked here as well as on placement, so a refused order sends no position mode lookup either
	if err := swapTradeCheckClientOrderID(orderType, s.ClientOrderID); err != nil {
		return nil, err
	}
	req := &SwapOrderRequest{
		Symbol:        s.Pair,
		Type:          orderType,
		Side:          side,
		Quantity:      s.Amount,
		ClientOrderID: s.ClientOrderID,
		TimeInForce:   timeInForce,
	}
	if orderTypeTakesPrice(orderType) {
		req.Price = s.Price
	}
	if orderTypeTakesTrigger(orderType) {
		req.StopPrice = s.TriggerPrice
	}
	if orderType == OrderTypeTrailingStopMarket {
		if s.TrackingValue <= 0 {
			return nil, errTrailingValueRequired
		}
		switch s.TrackingMode {
		case order.Distance:
			req.Price = s.TrackingValue
		case order.Percentage:
			// BingX takes the rate as a fraction
			req.PriceRate = s.TrackingValue / 100
		default:
			return nil, fmt.Errorf("%w: trailing stops trail by a distance or a percentage", order.ErrUnknownTrackingMode)
		}
		req.ActivationPrice = s.TriggerPrice
	}
	if orderTypeTakesTrigger(orderType) || orderType == OrderTypeTrailingStopMarket {
		if req.WorkingType, err = formatWorkingType(s.TriggerPriceType); err != nil {
			return nil, err
		}
	}
	if req.PositionSide, req.ReduceOnly, err = e.perpetualPositionSide(ctx, asset.USDTMarginedFutures, s.Side, s.ReduceOnly); err != nil {
		return nil, err
	}
	return req, nil
}

// submitSwapOrder places a USDT-M order, once more after looking the position mode up again when BingX refuses it for
// the mode, which a change made elsewhere leaves stale in the cache
func (e *Exchange) submitSwapOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	req, err := e.swapOrderRequest(ctx, s)
	if err != nil {
		return nil, err
	}
	placed, err := e.PlaceSwapOrder(ctx, req)
	if isPositionModeRejection(err) {
		e.forgetSwapPositionMode(ctx)
		if req, err = e.swapOrderRequest(ctx, s); err != nil {
			return nil, err
		}
		placed, err = e.PlaceSwapOrder(ctx, req)
	}
	if err != nil {
		return nil, err
	}
	resp, err := s.DeriveSubmitResponse(placed.OrderID.String())
	if err != nil {
		return nil, err
	}
	resp.ClientOrderID = placed.ClientOrderID
	resp.Status = order.New
	e.setPlacedOrderStatus(&resp.Status, resp.AssetType, resp.OrderID, placed.Status)
	resp.RemainingAmount = max(resp.Amount-placed.ExecutedQuantity.Float64(), 0)
	if placed.ExecutedQuantity > 0 {
		resp.AverageExecutedPrice = placed.AveragePrice.Float64()
		resp.Cost = placed.AveragePrice.Decimal().Mul(placed.ExecutedQuantity.Decimal()).InexactFloat64()
	}
	reportExecutedPlacementNew(&resp.Status, resp.Amount, placed.ExecutedQuantity)
	return resp, nil
}

// coinMarginedOrderRequest returns the coin-M order a resolved submission places
func (e *Exchange) coinMarginedOrderRequest(ctx context.Context, s *order.Submit) (*PlaceCoinMarginedOrderRequest, error) {
	side, orderType, timeInForce, err := perpetualOrderValues(s)
	if err != nil {
		return nil, err
	}
	req := &PlaceCoinMarginedOrderRequest{
		Symbol:        s.Pair,
		Type:          orderType,
		Side:          side,
		Quantity:      s.Amount,
		ClientOrderID: s.ClientOrderID,
		TimeInForce:   timeInForce,
	}
	if orderTypeTakesPrice(orderType) {
		req.Price = s.Price
	}
	if orderTypeTakesTrigger(orderType) {
		req.StopPrice = s.TriggerPrice
		if req.WorkingType, err = formatWorkingType(s.TriggerPriceType); err != nil {
			return nil, err
		}
	}
	if s.ReduceOnly {
		// Coin-M orders have no reduceOnly parameter, so only a hedge mode position side can mark an order as closing,
		// and a cached one-way mode may be stale
		req.PositionSide = hedgePositionSide(s.Side, true)
		return req, nil
	}
	if req.PositionSide, _, err = e.perpetualPositionSide(ctx, asset.CoinMarginedFutures, s.Side, false); err != nil {
		return nil, err
	}
	return req, nil
}

// submitCoinMarginedOrder places a coin-M order, whose reply carries no status, so the order is reported as new until
// an order event or query reports its fills. An order BingX refuses for its position side shows the account is in the
// other position mode, which it is taken to be in from then on: an ordinary order is placed once more for that mode,
// and a reduce only order, placed on its hedge mode side, returns errCoinMarginedReduceOnlyOneWay. A reduce only order
// BingX accepts shows hedge mode
func (e *Exchange) submitCoinMarginedOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	req, err := e.coinMarginedOrderRequest(ctx, s)
	if err != nil {
		return nil, err
	}
	placed, err := e.PlaceCoinMarginedOrder(ctx, req)
	if isPositionModeRejection(err) {
		hedgeMode := req.PositionSide == PositionSideBoth
		e.recordCoinMarginedPositionMode(ctx, hedgeMode)
		if s.ReduceOnly {
			return nil, fmt.Errorf("%w: %w", errCoinMarginedReduceOnlyOneWay, err)
		}
		req.PositionSide = PositionSideBoth
		if hedgeMode {
			req.PositionSide = hedgePositionSide(s.Side, false)
		}
		placed, err = e.PlaceCoinMarginedOrder(ctx, req)
	}
	if err != nil {
		return nil, err
	}
	if s.ReduceOnly {
		e.recordCoinMarginedPositionMode(ctx, true)
	}
	resp, err := s.DeriveSubmitResponse(placed.OrderID.String())
	if err != nil {
		return nil, err
	}
	resp.ClientOrderID = placed.ClientOrderID
	resp.Status = order.New
	return resp, nil
}

// ModifyOrder modifies an order's price, amount, trigger price or time in force. Amount is the order's desired total
// quantity, which the order manager fills with the order's original total when a modification leaves it out. A USDT-M
// LIMIT order whose amount alone changes is amended in place to that total. Other USDT-M orders and spot orders are
// cancelled, and once the order is cancelled a replacement is placed for the desired total, or the existing order's
// total without one, less the quantity the cancelled order had executed, taking every value the modification leaves
// unchanged from the existing order; the replacement's order ID is returned. The cancellation is sent once, and one
// that fails is checked by looking the order up again. A cancellation that was refused shows it cancelled nothing,
// whether BingX refused it with an error code, an HTTP 4xx status without one turned it away, as a firewall or rate
// limit page does, or it was never sent, as when the rate limiter refuses a wait past the context's deadline: a filled
// order returns an error saying it executed, one closed otherwise an error saying it is no longer open, and one still
// open the cancellation's error, each placing nothing. A cancellation without an answer showing its outcome, such as
// one that timed out, one answered with an HTTP 5xx status such as a gateway timeout, or one BingX answered with a code
// reporting a busy, internal or network failure (100500, 109500, 109701 or 110500), may have cancelled the order, so an
// order closed without filling is replaced, a filled one returns an error saying it executed, and one still open an
// error saying the outcome is unknown, as the cancellation may still land. A failed lookup returns both errors, the
// outcome being unknown. Modifications of one order run one at a time, whether they name it by its order ID or by its
// client order ID, which is looked up first, and each looks the order up by its order ID once its turn comes, so one
// waiting while another replaces the order finds it closed. An order the lookup shows closed, and a total not above the
// quantity the order's lookup reports executed, are refused before anything is cancelled, and a total not above the
// quantity executed by the cancellation places nothing and returns an error saying the order was cancelled. A
// replacement that fails after the cancellation is reported with the outcome of both. A trailing stop's trigger price
// is its activation price. A USDT-M order that closes a position keeps its position ID, and one whose separate isolated
// position its lookup does not name is refused before anything is cancelled, as is a NewClientOrderID for a USDT-M
// order other than a LIMIT or MARKET order. Spot orders report no time in force, so a spot replacement takes the
// modification's, or else BingX's default, GTC. Every request a modification sends goes to the account of the
// credentials ctx names when it starts. Coin-M orders cannot be modified
func (e *Exchange) ModifyOrder(ctx context.Context, action *order.Modify) (*order.ModifyResponse, error) {
	if err := action.Validate(); err != nil {
		return nil, err
	}
	// Refused before anything is cancelled, as a replacement BingX refuses for its price leaves the order cancelled
	for _, price := range []float64{action.Price, action.TriggerPrice} {
		if price < 0 || math.IsNaN(price) || math.IsInf(price, 0) {
			return nil, fmt.Errorf("%w: %v", errInvalidPrice, price)
		}
	}
	var ePath exchange.URL
	switch action.AssetType {
	case asset.Spot:
		ePath = exchange.RestSpot
	case asset.USDTMarginedFutures:
		ePath = exchange.RestUSDTMargined
	case asset.CoinMarginedFutures:
		return nil, fmt.Errorf("%w: BingX modifies no %s orders", common.ErrFunctionNotSupported, action.AssetType)
	default:
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, action.AssetType)
	}
	ctx, err := e.credentialsContext(ctx)
	if err != nil {
		return nil, err
	}
	account, err := e.productAccount(ctx, ePath)
	if err != nil {
		return nil, err
	}
	orderID, err := e.modifiedOrderID(ctx, action)
	if err != nil {
		return nil, err
	}
	unlock, err := e.orderModifications.lock(ctx, orderModificationKey{account: account, asset: action.AssetType, orderID: orderID})
	if err != nil {
		return nil, err
	}
	defer unlock()
	if action.AssetType == asset.Spot {
		return e.replaceSpotOrder(ctx, action, orderID)
	}
	return e.amendOrReplaceSwapOrder(ctx, action, orderID)
}

// modifiedOrderID returns the order ID of the order a spot or USDT-M modification names: its order ID, or without one
// the order ID a lookup by its client order ID returns, so modifications naming one order either way take one lock
func (e *Exchange) modifiedOrderID(ctx context.Context, action *order.Modify) (uint64, error) {
	orderID, err := parseOrderID(action.OrderID)
	if err != nil || orderID != 0 {
		return orderID, err
	}
	if action.AssetType == asset.Spot {
		o, err := e.GetSpotOrderDetails(ctx, &SpotOrderDetailsRequest{Symbol: action.Pair, ClientOrderID: action.ClientOrderID})
		if err != nil {
			return 0, fmt.Errorf("error fetching order to modify: %w", err)
		}
		return uint64(o.OrderID), nil
	}
	o, err := e.GetSwapOrderDetails(ctx, action.Pair, 0, action.ClientOrderID)
	if err != nil {
		return 0, fmt.Errorf("error fetching order to modify: %w", err)
	}
	return uint64(o.OrderID), nil
}

// replacementQuantity returns the quantity a cancel-replace places: the desired total, the modification's amount or
// else the existing order's original quantity, less the quantity the existing order has executed
func replacementQuantity(amount float64, original, executed types.Number) (float64, error) {
	if amount < 0 || math.IsNaN(amount) || math.IsInf(amount, 0) {
		return 0, fmt.Errorf("%w: %v", order.ErrAmountIsInvalid, amount)
	}
	total := original.Decimal()
	if amount > 0 {
		total = decimal.MustFromFloat(amount)
	}
	remaining := total.Sub(executed.Decimal())
	if !remaining.IsPositive() {
		return 0, fmt.Errorf("%w: %s, of which %s is executed", errModifyAmountNotAboveExecuted, total, executed.Decimal())
	}
	return remaining.InexactFloat64(), nil
}

// cancelledReplacementQuantity returns the quantity a replacement places once the order it replaces is cancelled: the
// desired total less the quantity the cancellation reports executed, or the lookup's when that is larger, as executed
// quantities only grow. A total not above it returns errCancelledOrderExecuted, the order being cancelled without a
// replacement
func cancelledReplacementQuantity(orderID string, amount float64, original, lookedUp, cancelled types.Number) (float64, error) {
	executed := max(lookedUp, cancelled)
	quantity, err := replacementQuantity(amount, original, executed)
	if errors.Is(err, errModifyAmountNotAboveExecuted) {
		return 0, fmt.Errorf("%w: order %s had executed %v when cancelled, and no replacement was placed", errCancelledOrderExecuted, orderID, executed.Float64())
	}
	return quantity, err
}

// orderOpenToModify returns errOrderNotOpen for an order whose lookup shows it closed, before anything is cancelled: a
// failed cancellation is checked by looking the order up again, and a closed order would then read as one the
// cancellation closed and be replaced. A status that cannot be converted is left to that check, which places nothing
// for it
func orderOpenToModify(orderID, status string) error {
	if converted, err := convertOrderStatus(status); err == nil && converted.IsInactive() {
		return fmt.Errorf("%w: order %s is %s", errOrderNotOpen, orderID, converted)
	}
	return nil
}

// cancellationRefused reports whether a cancellation failed in a way showing it cancelled nothing: it was never sent, as
// when the rate limiter refuses a wait past the context's deadline, BingX replied with an error code other than one
// reporting a failure that leaves the outcome unknown, or an HTTP 4xx status without a code turned it away, as a
// firewall or rate limit page does. Any other failure, such as a timeout, a lost connection or a 5xx status like a
// gateway timeout, leaves the outcome unknown
func cancellationRefused(err error) bool {
	if errors.Is(err, errRequestNotSent) {
		return true
	}
	if apiErr, ok := errors.AsType[*APIError](err); ok {
		return !apiErr.outcomeUnknown()
	}
	status, ok := badStatusCode(err)
	return ok && status >= http.StatusBadRequest && status < http.StatusInternalServerError
}

// failedCancellationExecuted returns the quantity an order to be replaced had executed when cancelled, as a lookup made
// after its cancellation failed with cancelErr reports in status and executed, or failed with lookupErr. The
// cancellation is sent once, so a refusal, as cancellationRefused reports, shows it cancelled nothing: an order the
// lookup shows filled returns errCancelledOrderExecuted, one closed otherwise errOrderNotOpen, something else having
// closed it, and one still open, or whose status does not show whether it is, errCancelReplaceFailed. A cancellation
// without an answer showing its outcome, such as one that timed out, one answered with an HTTP 5xx status or one BingX
// answered with a code reporting a busy, internal or network failure, may have cancelled the order: one closed without
// filling returns its executed quantity, to size its replacement by, one filled errCancelledOrderExecuted, and one
// still open, or whose status does not show whether it is, errCancelOutcomeUnknown, as the cancellation may still land.
// A failed lookup returns errCancelOutcomeUnknown with both errors
func failedCancellationExecuted(orderID, status string, executed types.Number, cancelErr, lookupErr error) (types.Number, error) {
	if lookupErr != nil {
		return 0, fmt.Errorf("%w: %w: cancelling order %s failed, and so did looking it up again, so no replacement was placed: %w; %w", errCancelReplaceFailed, errCancelOutcomeUnknown, orderID, cancelErr, lookupErr)
	}
	refused := cancellationRefused(cancelErr)
	converted, err := convertOrderStatus(status)
	unknown := err != nil || converted == order.UnknownStatus
	switch {
	case unknown && refused:
		return 0, fmt.Errorf("%w: cancelling order %s was refused, and its status %q does not show whether it is still open, so no replacement was placed: %w", errCancelReplaceFailed, orderID, status, cancelErr)
	case unknown:
		return 0, fmt.Errorf("%w: %w: cancelling order %s got no answer showing its outcome, and its status %q does not show whether it was cancelled, so no replacement was placed: %w", errCancelReplaceFailed, errCancelOutcomeUnknown, orderID, status, cancelErr)
	case converted == order.Filled:
		return 0, fmt.Errorf("%w: order %s filled before it could be cancelled, and no replacement was placed: %w", errCancelledOrderExecuted, orderID, cancelErr)
	case converted.IsInactive() && refused:
		return 0, fmt.Errorf("%w: cancelling order %s was refused, and the order is %s, so no replacement was placed: %w", errOrderNotOpen, orderID, converted, cancelErr)
	case converted.IsInactive():
		return executed, nil
	case refused:
		return 0, fmt.Errorf("%w: cancelling order %s was refused, so no replacement was placed: %w", errCancelReplaceFailed, orderID, cancelErr)
	default:
		return 0, fmt.Errorf("%w: %w: cancelling order %s got no answer showing its outcome while the order is still open, so no replacement was placed, although the cancellation may still land: %w", errCancelReplaceFailed, errCancelOutcomeUnknown, orderID, cancelErr)
	}
}

// cancelSpotOrderForReplacement cancels a spot order to be replaced and returns the quantity it had executed when
// cancelled. The cancellation is sent once, so how it failed shows whether it can have cancelled the order, and one
// that fails is checked by looking the order up again, as failedCancellationExecuted sets out
func (e *Exchange) cancelSpotOrderForReplacement(ctx context.Context, pair currency.Pair, orderID NumericID) (types.Number, error) {
	cancelled, err := e.CancelSpotOrder(request.WithRetryNotAllowed(ctx), &CancelSpotOrderRequest{Symbol: pair, OrderID: uint64(orderID)})
	if err == nil {
		return cancelled.ExecutedQuantity, nil
	}
	lookedUp, lookupErr := e.GetSpotOrderDetails(ctx, &SpotOrderDetailsRequest{Symbol: pair, OrderID: uint64(orderID)})
	if lookupErr != nil {
		return failedCancellationExecuted(orderID.String(), "", 0, err, lookupErr)
	}
	return failedCancellationExecuted(orderID.String(), string(lookedUp.Status), lookedUp.ExecutedQuantity, err, nil)
}

// replaceSpotOrder cancels spot order orderID, the order a modification names, and then places its replacement
func (e *Exchange) replaceSpotOrder(ctx context.Context, action *order.Modify, orderID uint64) (*order.ModifyResponse, error) {
	timeInForce, err := formatTimeInForce(action.TimeInForce)
	if err != nil {
		return nil, err
	}
	existing, err := e.GetSpotOrderDetails(ctx, &SpotOrderDetailsRequest{Symbol: action.Pair, OrderID: orderID})
	if err != nil {
		return nil, fmt.Errorf("error fetching order to modify: %w", err)
	}
	if err := orderOpenToModify(existing.OrderID.String(), string(existing.Status)); err != nil {
		return nil, err
	}
	if _, err := replacementQuantity(action.Amount, existing.OriginalQuantity, existing.ExecutedQuantity); err != nil {
		return nil, err
	}
	resp, err := action.DeriveModifyResponse()
	if err != nil {
		return nil, err
	}
	req := &PlaceSpotOrderRequest{
		Symbol:           action.Pair,
		Side:             existing.Side,
		Type:             existing.Type,
		Price:            cmp.Or(action.Price, existing.Price.Float64()),
		StopPrice:        cmp.Or(action.TriggerPrice, existing.StopPrice.Float64()),
		NewClientOrderID: action.NewClientOrderID,
		TimeInForce:      timeInForce,
	}
	if !orderTypeTakesPrice(req.Type) {
		req.Price = 0
	}
	if !orderTypeTakesTrigger(req.Type) {
		req.StopPrice = 0
	}
	cancelledExecuted, err := e.cancelSpotOrderForReplacement(ctx, action.Pair, existing.OrderID)
	if err != nil {
		return nil, err
	}
	if req.Quantity, err = cancelledReplacementQuantity(existing.OrderID.String(), action.Amount, existing.OriginalQuantity, existing.ExecutedQuantity, cancelledExecuted); err != nil {
		return nil, err
	}
	placed, err := e.PlaceSpotOrder(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%w: order %s was cancelled, but placing its replacement failed: %w", errCancelReplaceFailed, existing.OrderID, err)
	}
	resp.OrderID = placed.OrderID.String()
	resp.ClientOrderID = placed.ClientOrderID
	resp.Price = placed.Price.Float64()
	resp.Amount = placed.OriginalQuantity.Float64()
	resp.TriggerPrice = placed.StopPrice.Float64()
	resp.RemainingAmount = remainingAmount(placed.OriginalQuantity, placed.ExecutedQuantity)
	resp.LastUpdated = placed.TransactTime.Time()
	var convertErr error
	if resp.Type, convertErr = convertOrderType(placed.Type); convertErr != nil {
		e.logOrderIssue(action.AssetType, resp.OrderID, convertErr)
	}
	if resp.Side, convertErr = convertOrderSide(placed.Side); convertErr != nil {
		e.logOrderIssue(action.AssetType, resp.OrderID, convertErr)
	}
	resp.Status = order.New
	e.setPlacedOrderStatus(&resp.Status, action.AssetType, resp.OrderID, string(placed.Status))
	reportExecutedPlacementNew(&resp.Status, resp.Amount, placed.ExecutedQuantity)
	return resp, nil
}

// swapAmountOnlyChange reports whether a modification changes an order's amount and nothing else
func swapAmountOnlyChange(existing *SwapOrderResponse, action *order.Modify) bool {
	return action.Amount > 0 && action.Amount != existing.OriginalQuantity.Float64() &&
		(action.Price == 0 || action.Price == existing.Price.Float64()) &&
		(action.TriggerPrice == 0 || action.TriggerPrice == existing.StopPrice.Float64()) &&
		(action.TimeInForce == order.UnknownTIF || action.TimeInForce == swapOrderTimeInForce(existing)) &&
		action.NewClientOrderID == ""
}

// swapOrderTimeInForce returns the time in force of a USDT-M order, which the order queries report only as postOnly;
// a resting order without it is GTC, as IOC and FOK orders never rest
func swapOrderTimeInForce(o *SwapOrderResponse) order.TimeInForce {
	if o.PostOnly {
		return order.PostOnly
	}
	return order.GoodTillCancel
}

// swapAttachedOrder returns a USDT-M order's attached take profit or stop loss as a request, or nil without one
func swapAttachedOrder(attached *SwapTakeProfitStopLossResponse) *SwapTakeProfitStopLossRequest {
	if attached.Type == "" {
		return nil
	}
	return &SwapTakeProfitStopLossRequest{
		Type:           attached.Type,
		StopPrice:      attached.StopPrice,
		Price:          attached.Price,
		WorkingType:    attached.WorkingType,
		Quantity:       attached.Quantity,
		StopGuaranteed: attached.StopGuaranteed,
	}
}

// swapOrderCloses reports whether a USDT-M order closes a position: a reduce only or close position order, or in hedge
// mode a sell on the LONG side or a buy on the SHORT side
func swapOrderCloses(o *SwapOrderResponse) bool {
	switch {
	case o.ReduceOnly.Bool(), o.ClosePosition.Bool():
		return true
	case o.PositionSide == PositionSideLong:
		return o.Side == SideSell
	case o.PositionSide == PositionSideShort:
		return o.Side == SideBuy
	default:
		return false
	}
}

// cancelSwapOrderForReplacement cancels a USDT-M order to be replaced and returns the quantity it had executed when
// cancelled. CancelSwapOrder sends the cancellation once, so how it failed shows whether it can have cancelled the
// order, and one that fails is checked by looking the order up again, as failedCancellationExecuted sets out
func (e *Exchange) cancelSwapOrderForReplacement(ctx context.Context, pair currency.Pair, orderID NumericID) (types.Number, error) {
	cancelled, err := e.CancelSwapOrder(ctx, pair, uint64(orderID), "")
	if err == nil {
		return cancelled.ExecutedQuantity, nil
	}
	lookedUp, lookupErr := e.GetSwapOrderDetails(ctx, pair, uint64(orderID), "")
	if lookupErr != nil {
		return failedCancellationExecuted(orderID.String(), "", 0, err, lookupErr)
	}
	return failedCancellationExecuted(orderID.String(), lookedUp.Status, lookedUp.ExecutedQuantity, err, nil)
}

// amendOrReplaceSwapOrder amends USDT-M order orderID, the order a modification names, when it is a LIMIT order whose
// amount alone changes, and otherwise cancels and replaces it
func (e *Exchange) amendOrReplaceSwapOrder(ctx context.Context, action *order.Modify, orderID uint64) (*order.ModifyResponse, error) {
	existing, err := e.GetSwapOrderDetails(ctx, action.Pair, orderID, "")
	if err != nil {
		return nil, fmt.Errorf("error fetching order to modify: %w", err)
	}
	if err := orderOpenToModify(existing.OrderID.String(), existing.Status); err != nil {
		return nil, err
	}
	resp, err := action.DeriveModifyResponse()
	if err != nil {
		return nil, err
	}
	if existing.Type == OrderTypeLimit && swapAmountOnlyChange(existing, action) {
		amended, err := e.ModifySwapOrder(ctx, &ModifySwapOrderRequest{Symbol: action.Pair, OrderID: uint64(existing.OrderID), Quantity: action.Amount})
		if err != nil {
			return nil, err
		}
		resp.OrderID = amended.OrderID.String()
		resp.ClientOrderID = amended.ClientOrderID
		resp.Amount = amended.Quantity
		resp.Price = existing.Price.Float64()
		resp.RemainingAmount = max(amended.Quantity-existing.ExecutedQuantity.Float64(), 0)
		resp.TimeInForce = swapOrderTimeInForce(existing)
		resp.Type = order.Limit
		if resp.Side, err = convertOrderSide(existing.Side); err != nil {
			e.logOrderIssue(action.AssetType, resp.OrderID, err)
		}
		if resp.Status, err = convertOrderStatus(existing.Status); err != nil {
			e.logOrderIssue(action.AssetType, resp.OrderID, err)
		}
		return resp, nil
	}
	timeInForce := cmp.Or(action.TimeInForce, swapOrderTimeInForce(existing))
	formattedTimeInForce, err := formatTimeInForce(timeInForce)
	if err != nil {
		return nil, err
	}
	if _, err := replacementQuantity(action.Amount, existing.OriginalQuantity, existing.ExecutedQuantity); err != nil {
		return nil, err
	}
	// Checked before the cancellation, after which the replacement would be refused
	if err := swapTradeCheckClientOrderID(existing.Type, action.NewClientOrderID); err != nil {
		return nil, err
	}
	closes := swapOrderCloses(existing)
	if existing.PositionID == 0 && closes {
		// Closing a separate isolated position needs its ID, and a replacement refused for the lack of it would leave
		// the order cancelled
		marginType, err := e.GetSwapMarginType(ctx, action.Pair)
		if err != nil {
			return nil, fmt.Errorf("error fetching margin type of order to modify: %w", err)
		}
		if marginType.MarginType == MarginTypeSeparateIsolated {
			return nil, fmt.Errorf("%w: order %s closes a separate isolated position its lookup did not name", errSwapPositionIDRequired, existing.OrderID)
		}
	}
	workingType := existing.WorkingType
	// LastPrice is the price type's zero value, so only another price type shows the modification asks for a change
	if action.TriggerPriceType != order.LastPrice && (orderTypeTakesTrigger(existing.Type) || orderTypeTrails(existing.Type)) {
		if workingType, err = formatWorkingType(action.TriggerPriceType); err != nil {
			return nil, err
		}
	}
	closePosition := existing.ClosePosition.Bool()
	req := &SwapOrderRequest{
		Symbol:       action.Pair,
		Type:         existing.Type,
		Side:         existing.Side,
		PositionSide: existing.PositionSide,
		// Hedge mode orders and close position orders take no reduce only flag
		ReduceOnly:      existing.ReduceOnly.Bool() && existing.PositionSide == PositionSideBoth && !closePosition,
		Price:           cmp.Or(action.Price, existing.Price.Float64()),
		StopPrice:       cmp.Or(action.TriggerPrice, existing.StopPrice.Float64()),
		PriceRate:       existing.PriceRate.Float64(),
		WorkingType:     workingType,
		StopLoss:        swapAttachedOrder(&existing.StopLoss),
		TakeProfit:      swapAttachedOrder(&existing.TakeProfit),
		ClientOrderID:   action.NewClientOrderID,
		TimeInForce:     formattedTimeInForce,
		ClosePosition:   closePosition,
		ActivationPrice: existing.ActivationPrice.Float64(),
		StopGuaranteed:  existing.StopGuaranteed,
	}
	if closes {
		// BingX needs a position ID only to close a separate isolated position, while lookups report one for other
		// orders too
		req.PositionID = uint64(existing.PositionID)
	}
	if orderTypeTrails(req.Type) {
		req.StopPrice = 0
		req.ActivationPrice = cmp.Or(action.TriggerPrice, existing.ActivationPrice.Float64())
	}
	cancelledExecuted, err := e.cancelSwapOrderForReplacement(ctx, action.Pair, existing.OrderID)
	if err != nil {
		return nil, err
	}
	if req.Quantity, err = cancelledReplacementQuantity(existing.OrderID.String(), action.Amount, existing.OriginalQuantity, existing.ExecutedQuantity, cancelledExecuted); err != nil {
		return nil, err
	}
	placed, err := e.PlaceSwapOrder(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%w: order %s was cancelled, but placing its replacement failed: %w", errCancelReplaceFailed, existing.OrderID, err)
	}
	// Place Order replies carry no price, quantity or time in force, so the replacement is reported as requested
	resp.OrderID = placed.OrderID.String()
	resp.ClientOrderID = placed.ClientOrderID
	resp.Price = req.Price
	resp.Amount = req.Quantity
	resp.TriggerPrice = req.StopPrice
	if orderTypeTrails(req.Type) {
		// Reported as swapOrderDetail reports a trailing order, whose price is a trailing distance
		resp.Price = 0
		resp.TriggerPrice = req.ActivationPrice
	}
	resp.RemainingAmount = max(req.Quantity-placed.ExecutedQuantity.Float64(), 0)
	resp.TimeInForce = timeInForce
	resp.Status = order.New
	e.setPlacedOrderStatus(&resp.Status, action.AssetType, resp.OrderID, placed.Status)
	reportExecutedPlacementNew(&resp.Status, resp.Amount, placed.ExecutedQuantity)
	if resp.Type, err = convertOrderType(placed.Type); err != nil {
		e.logOrderIssue(action.AssetType, resp.OrderID, err)
	}
	if resp.Side, err = convertOrderSide(placed.Side); err != nil {
		e.logOrderIssue(action.AssetType, resp.OrderID, err)
	}
	return resp, nil
}

// CancelOrder cancels an order by its order ID, or by its client order ID without one
func (e *Exchange) CancelOrder(ctx context.Context, o *order.Cancel) error {
	if err := o.Validate(o.PairAssetRequired()); err != nil {
		return err
	}
	orderID, err := parseOrderID(o.OrderID)
	if err != nil {
		return err
	}
	if orderID == 0 && o.ClientOrderID == "" {
		return order.ErrOrderIDNotSet
	}
	clientOrderID := o.ClientOrderID
	if orderID != 0 {
		// The order ID identifies the order alone, as BingX takes either
		clientOrderID = ""
	}
	switch o.AssetType {
	case asset.Spot:
		_, err = e.CancelSpotOrder(ctx, &CancelSpotOrderRequest{Symbol: o.Pair, OrderID: orderID, ClientOrderID: clientOrderID})
	case asset.USDTMarginedFutures:
		_, err = e.CancelSwapOrder(ctx, o.Pair, orderID, clientOrderID)
	case asset.CoinMarginedFutures:
		_, err = e.CancelCoinMarginedOrder(ctx, o.Pair, orderID, clientOrderID)
	default:
		return fmt.Errorf("%w: %s", asset.ErrNotSupported, o.AssetType)
	}
	return err
}

// batchCancelGroup holds the orders of one pair a batch cancellation names, by order ID or by client order ID
type batchCancelGroup struct {
	pair           currency.Pair
	orderIDs       []uint64
	clientOrderIDs []string
}

// CancelBatchOrders cancels orders of one asset and pair in one request, as each of BingX's batch cancellations takes
// one symbol: spot orders, whose request reports each order's outcome, and USDT-M orders, at most ten by order ID and
// ten by client order ID. BingX has no coin-M batch cancellation, so coin-M orders are refused. Each order's status is
// its cancelled status or why its cancellation failed, keyed by the ID it was requested with; a failed request's error
// is returned with the response
func (e *Exchange) CancelBatchOrders(ctx context.Context, cancels []order.Cancel) (*order.CancelBatchResponse, error) {
	if len(cancels) == 0 {
		return nil, order.ErrCancelOrderIsNil
	}
	a := cancels[0].AssetType
	switch a {
	case asset.Spot, asset.USDTMarginedFutures:
	case asset.CoinMarginedFutures:
		return nil, fmt.Errorf("%w: BingX has no %s batch cancellation", common.ErrFunctionNotSupported, a)
	default:
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	g := &batchCancelGroup{pair: cancels[0].Pair}
	for i := range cancels {
		c := &cancels[i]
		if c.AssetType != a {
			return nil, errBatchCancelMixedAssets
		}
		if c.Pair.IsEmpty() {
			return nil, currency.ErrCurrencyPairEmpty
		}
		if !c.Pair.Equal(g.pair) {
			return nil, errBatchCancelMixedPairs
		}
		orderID, err := parseOrderID(c.OrderID)
		if err != nil {
			return nil, err
		}
		if orderID == 0 && c.ClientOrderID == "" {
			return nil, order.ErrOrderIDNotSet
		}
		if orderID != 0 {
			g.orderIDs = append(g.orderIDs, orderID)
		} else {
			g.clientOrderIDs = append(g.clientOrderIDs, c.ClientOrderID)
		}
	}
	if a == asset.USDTMarginedFutures {
		if err := swapTradeCheckListLength("order IDs", len(g.orderIDs), swapTradeBatchCancelLimit); err != nil {
			return nil, err
		}
		if err := swapTradeCheckListLength("client order IDs", len(g.clientOrderIDs), swapTradeBatchCancelLimit); err != nil {
			return nil, err
		}
	}
	resp := &order.CancelBatchResponse{Status: make(map[string]string, len(cancels))}
	if a == asset.Spot {
		return resp, e.cancelSpotBatch(ctx, g, resp)
	}
	return resp, e.cancelSwapBatch(ctx, g.pair, g.orderIDs, g.clientOrderIDs, resp)
}

// recordBatchRequestError records a whole cancellation request's error as the status of each order it covered, and
// returns it naming them
func recordBatchRequestError(resp *order.CancelBatchResponse, keys []string, err error) error {
	for _, k := range keys {
		resp.Status[k] = err.Error()
	}
	return fmt.Errorf("error cancelling orders %v: %w", keys, err)
}

// recordMissingBatchResults records the orders a cancellation reply left out as missing from it
func recordMissingBatchResults(resp *order.CancelBatchResponse, keys []string) {
	for _, k := range keys {
		if _, ok := resp.Status[k]; !ok {
			resp.Status[k] = errBatchCancelResultMissing.Error()
		}
	}
}

// batchKeys returns the status keys of a batch's order IDs followed by its client order IDs
func batchKeys(orderIDs []uint64, clientOrderIDs []string) []string {
	keys := make([]string, 0, len(orderIDs)+len(clientOrderIDs))
	for _, id := range orderIDs {
		keys = append(keys, strconv.FormatUint(id, 10))
	}
	return append(keys, clientOrderIDs...)
}

// cancelSpotBatch cancels a pair's spot orders in one request, which reports the orders cancelled and, as process=1
// asks, those it could not cancel instead of failing them all
func (e *Exchange) cancelSpotBatch(ctx context.Context, g *batchCancelGroup, resp *order.CancelBatchResponse) error {
	keys := batchKeys(g.orderIDs, g.clientOrderIDs)
	cancelled, err := e.CancelSpotBatchOrders(ctx, &CancelSpotBatchOrdersRequest{Symbol: g.pair, Process: true, OrderIDs: g.orderIDs, ClientOrderIDs: g.clientOrderIDs})
	if err != nil {
		return recordBatchRequestError(resp, keys, err)
	}
	clientOrderIDs := make(map[string]bool, len(g.clientOrderIDs))
	for _, id := range g.clientOrderIDs {
		clientOrderIDs[id] = true
	}
	for i := range cancelled.Orders {
		o := &cancelled.Orders[i]
		k := o.OrderID.String()
		if clientOrderIDs[o.ClientOrderID] {
			k = o.ClientOrderID
		}
		resp.Status[k] = string(o.Status)
	}
	for i := range cancelled.Fails {
		// A failure names its order by order ID alone, so one requested by client order ID is reported as missing
		if cancelled.Fails[i].OrderID != 0 {
			resp.Status[cancelled.Fails[i].OrderID.String()] = cancelled.Fails[i].Error
		}
	}
	recordMissingBatchResults(resp, keys)
	return nil
}

// cancelSwapBatch cancels up to ten of a pair's USDT-M orders by order ID and ten by client order ID in one request
func (e *Exchange) cancelSwapBatch(ctx context.Context, pair currency.Pair, orderIDs []uint64, clientOrderIDs []string, resp *order.CancelBatchResponse) error {
	keys := batchKeys(orderIDs, clientOrderIDs)
	cancelled, err := e.CancelSwapBatchOrders(ctx, pair, orderIDs, clientOrderIDs)
	if err != nil {
		return recordBatchRequestError(resp, keys, err)
	}
	// BingX converts client order IDs to lower case, so the outcomes are matched to the IDs as requested
	requested := make(map[string]string, len(clientOrderIDs))
	for _, id := range clientOrderIDs {
		requested[strings.ToLower(id)] = id
	}
	statusKey := func(orderID NumericID, clientOrderID string) string {
		if id, ok := requested[strings.ToLower(clientOrderID)]; ok && clientOrderID != "" {
			return id
		}
		return orderID.String()
	}
	for i := range cancelled.Success {
		resp.Status[statusKey(cancelled.Success[i].OrderID, cancelled.Success[i].ClientOrderID)] = cancelled.Success[i].Status
	}
	for i := range cancelled.Failed {
		f := &cancelled.Failed[i]
		resp.Status[statusKey(f.OrderID, f.ClientOrderID)] = (&APIError{Code: f.ErrorCode, Message: f.ErrorMessage}).Error()
	}
	recordMissingBatchResults(resp, keys)
	return nil
}

// cancelMatchedOrders cancels the open orders a narrowed CancelAllOrders matched in the batches each product takes: a
// pair's spot orders in one request, its USDT-M orders ten a request, and coin-M orders, which BingX has no batch
// cancellation for, one a request. Each order's status is recorded by its order ID
func (e *Exchange) cancelMatchedOrders(ctx context.Context, a asset.Item, matched []order.Cancel, status map[string]string) error {
	var errs error
	if a == asset.CoinMarginedFutures {
		for i := range matched {
			errs = common.AppendError(errs, e.cancelCoinMarginedOpenOrder(ctx, &matched[i], status))
		}
		return errs
	}
	for _, pairOrders := range ordersByPair(matched) {
		size := len(pairOrders)
		if a == asset.USDTMarginedFutures {
			size = swapTradeBatchCancelLimit
		}
		for batch := range slices.Chunk(pairOrders, size) {
			cancelled, err := e.CancelBatchOrders(ctx, batch)
			if cancelled != nil {
				maps.Copy(status, cancelled.Status)
			}
			errs = common.AppendError(errs, err)
		}
	}
	return errs
}

// ordersByPair groups orders by pair, in the order each pair first appears
func ordersByPair(cancels []order.Cancel) [][]order.Cancel {
	var groups [][]order.Cancel
	for i := range cancels {
		j := slices.IndexFunc(groups, func(g []order.Cancel) bool { return g[0].Pair.Equal(cancels[i].Pair) })
		if j < 0 {
			groups = append(groups, nil)
			j = len(groups) - 1
		}
		groups[j] = append(groups[j], cancels[i])
	}
	return groups
}

// cancelCoinMarginedOpenOrder cancels a coin-M order by its order ID, recording its cancelled status, or why its
// cancellation failed, and returning that error naming the order
func (e *Exchange) cancelCoinMarginedOpenOrder(ctx context.Context, c *order.Cancel, status map[string]string) error {
	orderID, err := parseOrderID(c.OrderID)
	var cancelled *CoinMarginedOrderResponse
	if err == nil {
		cancelled, err = e.CancelCoinMarginedOrder(ctx, c.Pair, orderID, "")
	}
	if err != nil {
		status[c.OrderID] = err.Error()
		return fmt.Errorf("error cancelling order %s: %w", c.OrderID, err)
	}
	status[c.OrderID] = cancelled.Status
	return nil
}

// CancelAllOrders cancels the open orders of a pair, or of every symbol when the pair is empty. A Side or Type narrows it
// to the open orders GetActiveOrders reports with that side and type, which are cancelled by order ID, as BingX's
// cancel-all endpoints take no side; the listing and its cancellations go to the account of the credentials ctx names
// when it starts. Each order's status is its cancelled status or why its cancellation failed
func (e *Exchange) CancelAllOrders(ctx context.Context, req *order.Cancel) (order.CancelAllResponse, error) {
	if err := req.Validate(); err != nil {
		return order.CancelAllResponse{}, err
	}
	switch req.AssetType {
	case asset.Spot, asset.USDTMarginedFutures, asset.CoinMarginedFutures:
	default:
		return order.CancelAllResponse{}, fmt.Errorf("%w: %s", asset.ErrNotSupported, req.AssetType)
	}
	if (req.Side != order.UnknownSide && req.Side != order.AnySide) || (req.Type != order.UnknownType && req.Type != order.AnyType) {
		var err error
		if ctx, err = e.credentialsContext(ctx); err != nil {
			return order.CancelAllResponse{}, err
		}
		listing := &order.MultiOrderRequest{AssetType: req.AssetType, Side: order.AnySide, Type: order.AnyType}
		if !req.Pair.IsEmpty() {
			listing.Pairs = currency.Pairs{req.Pair}
		}
		open, err := e.GetActiveOrders(ctx, listing)
		if err != nil {
			return order.CancelAllResponse{}, err
		}
		// GetActiveOrders' own filters keep orders whose side or type is unknown, which a narrowed cancellation leaves open
		filter := &order.Filter{Side: req.Side, Type: req.Type}
		var cancels []order.Cancel
		for i := range open {
			if open[i].MatchFilter(filter) {
				cancels = append(cancels, order.Cancel{Exchange: e.Name, OrderID: open[i].OrderID, Pair: open[i].Pair, AssetType: req.AssetType})
			}
		}
		resp := order.CancelAllResponse{Status: make(map[string]string, len(cancels))}
		return resp, e.cancelMatchedOrders(ctx, req.AssetType, cancels, resp.Status)
	}
	resp := order.CancelAllResponse{Status: make(map[string]string)}
	switch req.AssetType {
	case asset.Spot:
		cancelled, err := e.CancelSpotOpenOrders(ctx, req.Pair)
		if err != nil {
			return resp, err
		}
		for i := range cancelled {
			resp.Add(cancelled[i].OrderID.String(), string(cancelled[i].Status))
		}
	case asset.USDTMarginedFutures:
		cancelled, err := e.CancelAllSwapOpenOrders(ctx, req.Pair, "")
		if err != nil {
			return resp, err
		}
		for i := range cancelled.Success {
			resp.Add(cancelled.Success[i].OrderID.String(), cancelled.Success[i].Status)
		}
		for i := range cancelled.Failed {
			f := &cancelled.Failed[i]
			resp.Add(f.OrderID.String(), (&APIError{Code: f.ErrorCode, Message: f.ErrorMessage}).Error())
		}
	case asset.CoinMarginedFutures:
		cancelled, err := e.CancelAllCoinMarginedOrders(ctx, req.Pair)
		if err != nil {
			return resp, err
		}
		for i := range cancelled.Success {
			resp.Add(cancelled.Success[i].OrderID.String(), cancelled.Success[i].Status)
		}
		for i := range cancelled.Failed {
			f := &cancelled.Failed[i]
			resp.Add(f.OrderID.String(), (&APIError{Code: int64(min(f.Code, math.MaxInt64)), Message: f.Message}).Error())
		}
	}
	return resp, nil
}

// GetOrderInfo returns an order's details. A spot order BingX does not hold returns order.ErrOrderNotFound
func (e *Exchange) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, a asset.Item) (*order.Detail, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	id, err := parseOrderID(orderID)
	if err != nil {
		return nil, err
	}
	var fees *perpetualOrderFeeRead
	if a == asset.USDTMarginedFutures || a == asset.CoinMarginedFutures {
		ctx, fees = e.beginPerpetualOrderFeeRead(ctx, a)
		defer fees.end()
	}
	var d order.Detail
	switch a {
	case asset.Spot:
		o, err := e.GetSpotOrderDetails(ctx, &SpotOrderDetailsRequest{Symbol: pair, OrderID: id})
		if err != nil {
			return nil, err
		}
		d, err = e.spotOrderDetail(o, pair)
		if err != nil {
			return nil, fmt.Errorf("error converting %s order %s: %w", a, orderID, err)
		}
	case asset.USDTMarginedFutures:
		o, err := e.GetSwapOrderDetails(ctx, pair, id, "")
		if err != nil {
			return nil, err
		}
		d, err = e.swapOrderDetail(o, pair)
		if err != nil {
			return nil, fmt.Errorf("error converting %s order %s: %w", a, orderID, err)
		}
		fees.seed(&d, o.ExecutedQuantity, o.Commission)
	case asset.CoinMarginedFutures:
		o, err := e.GetCoinMarginedOrderDetail(ctx, pair, id, "")
		if err != nil {
			return nil, err
		}
		d, err = e.coinMarginedOrderDetail(o, pair)
		if err != nil {
			return nil, fmt.Errorf("error converting %s order %s: %w", a, orderID, err)
		}
		fees.seed(&d, o.ExecutedQuantity, o.Commission)
	default:
		return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
	return &d, nil
}

// spotOrderDetail converts a spot order. Values that cannot be converted are left unset and returned as the error.
// BingX reports a charged fee as negative on fills and in order events but documents no sign for the order queries,
// so the fee is taken as paid whatever its sign
func (e *Exchange) spotOrderDetail(o *SpotOrderDetailResponse, pair currency.Pair) (order.Detail, error) {
	status, statusErr := convertOrderStatus(string(o.Status))
	orderType, typeErr := convertOrderType(o.Type)
	side, sideErr := convertOrderSide(o.Side)
	d := order.Detail{
		Price:                o.Price.Float64(),
		Amount:               o.OriginalQuantity.Float64(),
		TriggerPrice:         o.StopPrice.Float64(),
		AverageExecutedPrice: o.AveragePrice.Float64(),
		QuoteAmount:          o.OriginalQuoteOrderQuantity.Float64(),
		ExecutedAmount:       o.ExecutedQuantity.Float64(),
		RemainingAmount:      remainingAmount(o.OriginalQuantity, o.ExecutedQuantity),
		Cost:                 o.CummulativeQuoteQuantity.Float64(),
		CostAsset:            pair.Quote,
		Fee:                  math.Abs(o.Fee.Float64()),
		FeeAsset:             o.FeeAsset,
		Exchange:             e.Name,
		OrderID:              o.OrderID.String(),
		ClientOrderID:        o.ClientOrderID,
		Type:                 orderType,
		Side:                 side,
		Status:               status,
		AssetType:            asset.Spot,
		Date:                 o.Time.Time(),
		LastUpdated:          o.UpdateTime.Time(),
		Pair:                 pair,
	}
	return d, common.AppendError(common.AppendError(statusErr, typeErr), sideErr)
}

// swapOrderDetail converts a USDT-M order, whose cost and fee are in its settlement currency, the pair's quote. A
// trailing stop's price is its trailing distance, which order details hold no field for, and it triggers at its
// activation price. Values that cannot be converted are left unset and returned as the error
func (e *Exchange) swapOrderDetail(o *SwapOrderResponse, pair currency.Pair) (order.Detail, error) {
	status, statusErr := convertOrderStatus(o.Status)
	orderType, typeErr := convertOrderType(o.Type)
	side, sideErr := convertOrderSide(o.Side)
	leverage, leverageErr := parseLeverage(o.Leverage)
	d := order.Detail{
		TimeInForce:          swapOrderTimeInForce(o),
		ReduceOnly:           o.ReduceOnly.Bool() || o.ClosePosition.Bool(),
		Leverage:             leverage,
		Price:                o.Price.Float64(),
		Amount:               o.OriginalQuantity.Float64(),
		TriggerPrice:         o.StopPrice.Float64(),
		AverageExecutedPrice: o.AveragePrice.Float64(),
		ExecutedAmount:       o.ExecutedQuantity.Float64(),
		RemainingAmount:      remainingAmount(o.OriginalQuantity, o.ExecutedQuantity),
		Cost:                 o.CumulativeQuote.Float64(),
		CostAsset:            pair.Quote,
		Fee:                  math.Abs(o.Commission.Float64()),
		FeeAsset:             pair.Quote,
		Exchange:             e.Name,
		OrderID:              o.OrderID.String(),
		ClientOrderID:        o.ClientOrderID,
		Type:                 orderType,
		Side:                 side,
		Status:               status,
		AssetType:            asset.USDTMarginedFutures,
		Date:                 o.Time.Time(),
		LastUpdated:          o.UpdateTime.Time(),
		Pair:                 pair,
		SettlementCurrency:   pair.Quote,
	}
	if orderTypeTrails(o.Type) {
		d.Price = 0
		d.TriggerPrice = o.ActivationPrice.Float64()
	}
	return d, common.AppendError(common.AppendError(common.AppendError(statusErr, typeErr), sideErr), leverageErr)
}

// coinMarginedOrderDetail converts a coin-M order, whose amounts count contracts and whose fee is in its margin coin,
// the pair's base. Some replies send the order's contracts as quantity rather than origQty, and spell its status in
// mixed case. Values that cannot be converted are left unset and returned as the error
func (e *Exchange) coinMarginedOrderDetail(o *CoinMarginedOrderResponse, pair currency.Pair) (order.Detail, error) {
	status, statusErr := convertCoinMarginedOrderStatus(o.Status)
	orderType, typeErr := convertOrderType(o.Type)
	side, sideErr := convertOrderSide(o.Side)
	timeInForce, timeInForceErr := convertTimeInForce(o.TimeInForce)
	leverage, leverageErr := parseLeverage(o.Leverage)
	if o.PostOnly {
		timeInForce = order.PostOnly
	}
	amount := o.OriginalQuantity
	if amount == 0 {
		amount = types.Number(o.Quantity)
	}
	d := order.Detail{
		TimeInForce:          timeInForce,
		ReduceOnly:           o.ReduceOnly.Bool(),
		Leverage:             leverage,
		Price:                o.Price.Float64(),
		Amount:               amount.Float64(),
		TriggerPrice:         o.StopPrice.Float64(),
		AverageExecutedPrice: o.AveragePrice.Float64(),
		ExecutedAmount:       o.ExecutedQuantity.Float64(),
		RemainingAmount:      remainingAmount(amount, o.ExecutedQuantity),
		Fee:                  math.Abs(o.Commission.Float64()),
		FeeAsset:             pair.Base,
		Exchange:             e.Name,
		OrderID:              o.OrderID.String(),
		ClientOrderID:        o.ClientOrderID,
		Type:                 orderType,
		Side:                 side,
		Status:               status,
		AssetType:            asset.CoinMarginedFutures,
		Date:                 o.Time.Time(),
		LastUpdated:          o.UpdateTime.Time(),
		Pair:                 pair,
		SettlementCurrency:   pair.Base,
	}
	errs := common.AppendError(common.AppendError(statusErr, typeErr), sideErr)
	return d, common.AppendError(common.AppendError(errs, timeInForceErr), leverageErr)
}

// appendListedOrder adds an order of a listing to orders: the pair it was requested for, or the pair its symbol names.
// An order whose symbol cannot be read is left out, and one with values that cannot be converted is kept without
// them, each with a logged warning, so one odd order does not fail the listing
func (e *Exchange) appendListedOrder(orders []order.Detail, a asset.Item, orderID string, requested currency.Pair, symbol string, convert func(currency.Pair) (order.Detail, error)) []order.Detail {
	pair := requested
	if pair.IsEmpty() {
		var err error
		if pair, err = e.wsAccountPair(symbol, a); err != nil {
			e.logOrderIssue(a, orderID, err)
			return orders
		}
	}
	d, err := convert(pair)
	if err != nil {
		e.logOrderIssue(a, orderID, err)
	}
	return append(orders, d)
}

// listingPairs returns the pairs to request orders for: the one pair requested, or every symbol in one request when
// several or none are requested, as each request lists every symbol's orders as cheaply as one symbol's
func listingPairs(pairs currency.Pairs) currency.Pairs {
	if len(pairs) == 1 {
		return pairs
	}
	return currency.Pairs{currency.EMPTYPAIR}
}

// GetActiveOrders returns the open orders of the requested pairs, or of every symbol when none are requested
func (e *Exchange) GetActiveOrders(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	var fees *perpetualOrderFeeRead
	if req.AssetType == asset.USDTMarginedFutures || req.AssetType == asset.CoinMarginedFutures {
		ctx, fees = e.beginPerpetualOrderFeeRead(ctx, req.AssetType)
		defer fees.end()
	}
	var orders []order.Detail
	for _, p := range listingPairs(req.Pairs) {
		switch req.AssetType {
		case asset.Spot:
			open, err := e.GetSpotOpenOrders(ctx, p)
			if err != nil {
				return nil, err
			}
			for i := range open {
				o := &open[i]
				orders = e.appendListedOrder(orders, req.AssetType, o.OrderID.String(), p, o.Symbol, func(pair currency.Pair) (order.Detail, error) {
					return e.spotOrderDetail(o, pair)
				})
			}
		case asset.USDTMarginedFutures:
			open, err := e.GetSwapOpenOrders(ctx, p, "")
			if err != nil {
				return nil, err
			}
			for i := range open {
				o := &open[i]
				orders = e.appendListedOrder(orders, req.AssetType, o.OrderID.String(), p, o.Symbol, func(pair currency.Pair) (order.Detail, error) {
					d, err := e.swapOrderDetail(o, pair)
					fees.seed(&d, o.ExecutedQuantity, o.Commission)
					return d, err
				})
			}
		case asset.CoinMarginedFutures:
			open, err := e.GetCoinMarginedOpenOrders(ctx, p)
			if err != nil {
				return nil, err
			}
			for i := range open {
				o := &open[i]
				orders = e.appendListedOrder(orders, req.AssetType, o.OrderID.String(), p, o.Symbol, func(pair currency.Pair) (order.Detail, error) {
					d, err := e.coinMarginedOrderDetail(o, pair)
					fees.seed(&d, o.ExecutedQuantity, o.Commission)
					return d, err
				})
			}
		default:
			return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, req.AssetType)
		}
	}
	return req.Filter(e.Name, orders), nil
}

// GetOrderHistory returns the orders of the requested pairs, or of every symbol when none are requested: those from
// FromOrderID on, filtered to the request's time window, or those within the window, collected in windows the
// endpoints take (spot pages of 100 up to the 10,000th order, USDT-M and coin-M spans of seven days of up to 1000
// orders), each split while it holds more than the endpoint returns. Orders from FromOrderID on are paged: spot up to
// its 10,000th order, beyond which the history is reported truncated, and USDT-M and coin-M from the highest order ID
// each page returns until a page comes back short or adds no later order. Without a window or an order ID, BingX's
// most recent orders are returned. USDT-M listings of every symbol cover USDT- and USDC-settled contracts
func (e *Exchange) GetOrderHistory(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil && !errors.Is(err, common.ErrDateUnset) {
		return nil, err
	}
	fromID, err := parseOrderID(req.FromOrderID)
	if err != nil {
		return nil, err
	}
	if !req.StartTime.IsZero() && req.EndTime.IsZero() {
		windowed := *req
		windowed.EndTime = time.Now()
		req = &windowed
	}
	var orders []order.Detail
	for _, p := range listingPairs(req.Pairs) {
		var listed []order.Detail
		switch req.AssetType {
		case asset.Spot:
			listed, err = e.spotOrderHistory(ctx, req, p, fromID)
		case asset.USDTMarginedFutures:
			listed, err = e.swapOrderHistory(ctx, req, p, fromID)
		case asset.CoinMarginedFutures:
			listed, err = e.coinMarginedOrderHistory(ctx, req, p, fromID)
		default:
			return nil, fmt.Errorf("%w: %s", asset.ErrNotSupported, req.AssetType)
		}
		if err != nil {
			return nil, err
		}
		orders = append(orders, listed...)
	}
	return req.Filter(e.Name, orders), nil
}

// windowedOrders collects every order from start to end through fetch, which reports whether a window's orders were
// cut short by the endpoint's limit. A range longer than maxSpan, when set, is split into windows of maxSpan, and a
// window whose orders were cut short is halved until each part's fit. Windows share their boundary millisecond, as
// the endpoints do not document whether their bounds are inclusive, and orders fetched twice are dropped by ID
func windowedOrders[T any](start, end time.Time, maxSpan time.Duration, fetch func(start, end time.Time) ([]T, bool, error), idOf func(*T) string) ([]T, error) {
	start, end = start.Truncate(time.Millisecond), end.Truncate(time.Millisecond)
	var orders []T
	seen := make(map[string]bool)
	var collect func(start, end time.Time) error
	collect = func(windowStart, windowEnd time.Time) error {
		if maxSpan > 0 && windowEnd.Sub(windowStart) > maxSpan {
			for spanStart := windowStart; spanStart.Before(windowEnd); spanStart = spanStart.Add(maxSpan) {
				if err := collect(spanStart, earlierTime(spanStart.Add(maxSpan), windowEnd)); err != nil {
					return err
				}
			}
			return nil
		}
		page, cutShort, err := fetch(windowStart, windowEnd)
		if err != nil {
			return err
		}
		if cutShort {
			if windowEnd.Sub(windowStart) < 2*time.Millisecond {
				return fmt.Errorf("%w: %s to %s", errOrderHistoryLimit, windowStart.UTC(), windowEnd.UTC())
			}
			middle := windowStart.Add(windowEnd.Sub(windowStart) / 2).Truncate(time.Millisecond)
			if err := collect(windowStart, middle); err != nil {
				return err
			}
			return collect(middle, windowEnd)
		}
		for i := range page {
			if id := idOf(&page[i]); !seen[id] {
				seen[id] = true
				orders = append(orders, page[i])
			}
		}
		return nil
	}
	return orders, collect(start, end)
}

// cursorOrders collects every order from an order ID on through fetch, which returns up to limit orders from the ID it
// is given, that order included or not. Each further page is fetched from the highest order ID collected, and an order
// fetched twice, as one at an inclusive cursor is, is kept once. Paging stops at a page shorter than limit, or at one
// that leaves the cursor where it was, once that page's orders are collected
func cursorOrders[T any](fromID uint64, limit int, fetch func(fromID uint64) ([]T, error), idOf func(*T) uint64) ([]T, error) {
	var orders []T
	seen := make(map[uint64]bool)
	for cursor := fromID; ; {
		page, err := fetch(cursor)
		if err != nil {
			return nil, err
		}
		next := cursor
		for i := range page {
			id := idOf(&page[i])
			if !seen[id] {
				seen[id] = true
				orders = append(orders, page[i])
			}
			next = max(next, id)
		}
		if len(page) < limit || next == cursor {
			return orders, nil
		}
		cursor = next
	}
}

// earlierTime returns the earlier of two times
func earlierTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

// spotOrderPages returns a spot order history query's orders, a page of 100 at a time while pages are full, and
// whether the furthest page BingX serves was full, which leaves later orders out
func (e *Exchange) spotOrderPages(ctx context.Context, req *SpotOrderHistoryRequest) ([]SpotOrderDetailResponse, bool, error) {
	var orders []SpotOrderDetailResponse
	for pageIndex := uint64(1); pageIndex <= spotOrderHistoryPagesPerSpan; pageIndex++ {
		page := *req
		page.PageIndex, page.PageSize = pageIndex, spotOrderHistoryPageSize
		resp, err := e.GetSpotOrderHistory(ctx, &page)
		if err != nil {
			return nil, false, err
		}
		orders = append(orders, resp...)
		if len(resp) < spotOrderHistoryPageSize {
			return orders, false, nil
		}
	}
	return orders, true, nil
}

// spotOrderHistory returns a pair's spot orders, or every symbol's for an empty pair
func (e *Exchange) spotOrderHistory(ctx context.Context, req *order.MultiOrderRequest, pair currency.Pair, fromID uint64) ([]order.Detail, error) {
	var history []SpotOrderDetailResponse
	var err error
	if fromID != 0 || req.StartTime.IsZero() || req.EndTime.IsZero() {
		// An order ID or the most recent orders are paged without a window, which the request's filter applies
		var truncated bool
		history, truncated, err = e.spotOrderPages(ctx, &SpotOrderHistoryRequest{Symbol: pair, OrderID: fromID})
		if err == nil && truncated && fromID != 0 {
			return nil, fmt.Errorf("%w: from order %d, %d orders", errOrderHistoryTruncated, fromID, len(history))
		}
	} else {
		history, err = windowedOrders(req.StartTime, req.EndTime, 0, func(start, end time.Time) ([]SpotOrderDetailResponse, bool, error) {
			return e.spotOrderPages(ctx, &SpotOrderHistoryRequest{Symbol: pair, StartTime: start, EndTime: end})
		}, func(o *SpotOrderDetailResponse) string { return o.OrderID.String() })
	}
	if err != nil {
		return nil, err
	}
	orders := make([]order.Detail, 0, len(history))
	for i := range history {
		o := &history[i]
		orders = e.appendListedOrder(orders, asset.Spot, o.OrderID.String(), pair, o.Symbol, func(p currency.Pair) (order.Detail, error) {
			return e.spotOrderDetail(o, p)
		})
	}
	return orders, nil
}

// swapSettlementCurrencies returns the settlement currencies a USDT-M query covers: a pair's, its quote, as every
// contract's symbol ends in its settlement currency, or both USDT and USDC for every symbol
func swapSettlementCurrencies(pair currency.Pair) []currency.Code {
	if pair.IsEmpty() {
		return []currency.Code{currency.USDT, currency.USDC}
	}
	return []currency.Code{pair.Quote}
}

// swapOrderHistory returns a pair's filled and cancelled USDT-M orders, or every symbol's for an empty pair
func (e *Exchange) swapOrderHistory(ctx context.Context, req *order.MultiOrderRequest, pair currency.Pair, fromID uint64) ([]order.Detail, error) {
	var history []SwapOrderResponse
	for _, settlement := range swapSettlementCurrencies(pair) {
		var listed []SwapOrderResponse
		var err error
		switch {
		case fromID != 0:
			listed, err = cursorOrders(fromID, perpetualOrderHistoryLimit, func(cursor uint64) ([]SwapOrderResponse, error) {
				return e.GetSwapOrderHistory(ctx, &SwapOrderHistoryRequest{Symbol: pair, Currency: settlement, OrderID: cursor, Limit: perpetualOrderHistoryLimit})
			}, func(o *SwapOrderResponse) uint64 { return uint64(o.OrderID) })
		case req.StartTime.IsZero() || req.EndTime.IsZero():
			listed, err = e.GetSwapOrderHistory(ctx, &SwapOrderHistoryRequest{Symbol: pair, Currency: settlement, Limit: perpetualOrderHistoryLimit})
		default:
			listed, err = windowedOrders(req.StartTime, req.EndTime, perpetualOrderHistorySpan, func(start, end time.Time) ([]SwapOrderResponse, bool, error) {
				page, err := e.GetSwapOrderHistory(ctx, &SwapOrderHistoryRequest{Symbol: pair, Currency: settlement, StartTime: start, EndTime: end, Limit: perpetualOrderHistoryLimit})
				return page, len(page) >= perpetualOrderHistoryLimit, err
			}, func(o *SwapOrderResponse) string { return o.OrderID.String() })
		}
		if err != nil {
			return nil, err
		}
		history = append(history, listed...)
	}
	orders := make([]order.Detail, 0, len(history))
	for i := range history {
		o := &history[i]
		orders = e.appendListedOrder(orders, asset.USDTMarginedFutures, o.OrderID.String(), pair, o.Symbol, func(p currency.Pair) (order.Detail, error) {
			return e.swapOrderDetail(o, p)
		})
	}
	return orders, nil
}

// coinMarginedOrderHistory returns a pair's filled and cancelled coin-M orders, or every symbol's for an empty pair
func (e *Exchange) coinMarginedOrderHistory(ctx context.Context, req *order.MultiOrderRequest, pair currency.Pair, fromID uint64) ([]order.Detail, error) {
	var history []CoinMarginedOrderResponse
	var err error
	switch {
	case fromID != 0:
		history, err = cursorOrders(fromID, perpetualOrderHistoryLimit, func(cursor uint64) ([]CoinMarginedOrderResponse, error) {
			return e.GetCoinMarginedOrderHistory(ctx, &CoinMarginedOrderHistoryRequest{Symbol: pair, OrderID: cursor, Limit: perpetualOrderHistoryLimit})
		}, func(o *CoinMarginedOrderResponse) uint64 { return uint64(o.OrderID) })
	case req.StartTime.IsZero() || req.EndTime.IsZero():
		history, err = e.GetCoinMarginedOrderHistory(ctx, &CoinMarginedOrderHistoryRequest{Symbol: pair, Limit: perpetualOrderHistoryLimit})
	default:
		history, err = windowedOrders(req.StartTime, req.EndTime, perpetualOrderHistorySpan, func(start, end time.Time) ([]CoinMarginedOrderResponse, bool, error) {
			page, err := e.GetCoinMarginedOrderHistory(ctx, &CoinMarginedOrderHistoryRequest{Symbol: pair, StartTime: start, EndTime: end, Limit: perpetualOrderHistoryLimit})
			return page, len(page) >= perpetualOrderHistoryLimit, err
		}, func(o *CoinMarginedOrderResponse) string { return o.OrderID.String() })
	}
	if err != nil {
		return nil, err
	}
	orders := make([]order.Detail, 0, len(history))
	for i := range history {
		o := &history[i]
		orders = e.appendListedOrder(orders, asset.CoinMarginedFutures, o.OrderID.String(), pair, o.Symbol, func(p currency.Pair) (order.Detail, error) {
			return e.coinMarginedOrderDetail(o, p)
		})
	}
	return orders, nil
}
