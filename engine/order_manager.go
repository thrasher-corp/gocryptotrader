package engine

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gofrs/uuid"
	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/communications/base"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// SetupOrderManager will boot up the OrderManager
func SetupOrderManager(exchangeManager iExchangeManager, communicationsManager iCommsManager, wg *sync.WaitGroup, verbose bool) (*OrderManager, error) {
	if exchangeManager == nil {
		return nil, errNilExchangeManager
	}
	if communicationsManager == nil {
		return nil, errNilCommunicationsManager
	}
	if wg == nil {
		return nil, errNilWaitGroup
	}

	return &OrderManager{
		shutdown: make(chan struct{}),
		orderStore: store{
			Orders:                    make(map[string][]*order.Detail),
			exchangeManager:           exchangeManager,
			commsManager:              communicationsManager,
			wg:                        wg,
			futuresPositionController: order.SetupPositionController(),
		},
		verbose: verbose,
	}, nil
}

// IsRunning safely checks whether the subsystem is running
func (m *OrderManager) IsRunning() bool {
	if m == nil {
		return false
	}
	return atomic.LoadInt32(&m.started) == 1
}

// Start runs the subsystem
func (m *OrderManager) Start() error {
	if m == nil {
		return fmt.Errorf("order manager %w", ErrNilSubsystem)
	}
	if !atomic.CompareAndSwapInt32(&m.started, 0, 1) {
		return fmt.Errorf("order manager %w", ErrSubSystemAlreadyStarted)
	}
	log.Debugln(log.OrderMgr, "Order manager starting...")
	m.shutdown = make(chan struct{})
	go m.run()
	return nil
}

// Stop attempts to shutdown the subsystem
func (m *OrderManager) Stop() error {
	if m == nil {
		return fmt.Errorf("order manager %w", ErrNilSubsystem)
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return fmt.Errorf("order manager %w", ErrSubSystemNotStarted)
	}

	defer func() {
		atomic.CompareAndSwapInt32(&m.started, 1, 0)
	}()

	log.Debugln(log.OrderMgr, "Order manager shutting down...")
	close(m.shutdown)
	return nil
}

// gracefulShutdown cancels all orders (if enabled) before shutting down
func (m *OrderManager) gracefulShutdown() {
	if m.cfg.CancelOrdersOnShutdown {
		log.Debugln(log.OrderMgr, "Order manager: Cancelling any open orders...")
		exchanges, err := m.orderStore.exchangeManager.GetExchanges()
		if err != nil {
			log.Errorf(log.OrderMgr, "Order manager cannot get exchanges: %v", err)
			return
		}
		m.CancelAllOrders(context.TODO(), exchanges)
	}
}

// run will periodically process orders
func (m *OrderManager) run() {
	log.Debugln(log.OrderMgr, "Order manager started.")
	m.processOrders()
	tick := time.NewTicker(orderManagerDelay)
	m.orderStore.wg.Add(1)
	defer func() {
		log.Debugln(log.OrderMgr, "Order manager shutdown.")
		tick.Stop()
		m.orderStore.wg.Done()
	}()

	for {
		select {
		case <-m.shutdown:
			m.gracefulShutdown()
			return
		case <-tick.C:
			go m.processOrders()
		}
	}
}

// CancelAllOrders iterates and cancels all orders for each exchange provided
func (m *OrderManager) CancelAllOrders(ctx context.Context, exchangeNames []exchange.IBotExchange) {
	if m == nil || atomic.LoadInt32(&m.started) == 0 {
		return
	}

	orders := m.orderStore.get()
	if orders == nil {
		return
	}

	for i := range exchangeNames {
		exchangeOrders, ok := orders[strings.ToLower(exchangeNames[i].GetName())]
		if !ok {
			continue
		}
		for j := range exchangeOrders {
			log.Debugf(log.OrderMgr, "Order manager: Cancelling order(s) for exchange %s.", exchangeNames[i].GetName())
			err := m.Cancel(ctx, &order.Cancel{
				Exchange:      exchangeOrders[j].Exchange,
				ID:            exchangeOrders[j].ID,
				AccountID:     exchangeOrders[j].AccountID,
				ClientID:      exchangeOrders[j].ClientID,
				WalletAddress: exchangeOrders[j].WalletAddress,
				Type:          exchangeOrders[j].Type,
				Side:          exchangeOrders[j].Side,
				Pair:          exchangeOrders[j].Pair,
				AssetType:     exchangeOrders[j].AssetType,
			})
			if err != nil {
				log.Error(log.OrderMgr, err)
			}
		}
	}
}

// Cancel will find the order in the OrderManager, send a cancel request
// to the exchange and if successful, update the status of the order
func (m *OrderManager) Cancel(ctx context.Context, cancel *order.Cancel) error {
	if m == nil {
		return fmt.Errorf("order manager %w", ErrNilSubsystem)
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return fmt.Errorf("order manager %w", ErrSubSystemNotStarted)
	}
	var err error
	defer func() {
		if err != nil {
			m.orderStore.commsManager.PushEvent(base.Event{
				Type:    "order",
				Message: err.Error(),
			})
		}
	}()

	if cancel == nil {
		err = errors.New("order cancel param is nil")
		return err
	}
	if cancel.Exchange == "" {
		err = errors.New("order exchange name is empty")
		return err
	}
	if cancel.ID == "" {
		err = errors.New("order id is empty")
		return err
	}

	exch, err := m.orderStore.exchangeManager.GetExchangeByName(cancel.Exchange)
	if err != nil {
		return err
	}

	if cancel.AssetType.String() != "" && !exch.GetAssetTypes(false).Contains(cancel.AssetType) {
		err = errors.New("order asset type not supported by exchange")
		return err
	}

	log.Debugf(log.OrderMgr, "Order manager: Cancelling order ID %v [%+v]",
		cancel.ID, cancel)

	err = exch.CancelOrder(ctx, cancel)
	if err != nil {
		err = fmt.Errorf("%v - Failed to cancel order: %w", cancel.Exchange, err)
		return err
	}
	var od *order.Detail
	od, err = m.orderStore.getByExchangeAndID(cancel.Exchange, cancel.ID)
	if err != nil {
		err = fmt.Errorf("%v - Failed to retrieve order %v to update cancelled status: %w", cancel.Exchange, cancel.ID, err)
		return err
	}

	od.Status = order.Cancelled
	msg := fmt.Sprintf("Order manager: Exchange %s order ID=%v cancelled.",
		od.Exchange, od.ID)
	log.Debugln(log.OrderMgr, msg)
	m.orderStore.commsManager.PushEvent(base.Event{
		Type:    "order",
		Message: msg,
	})

	return nil
}

// GetFuturesPositionsForExchange returns futures positions stored within
// the order manager's futures position tracker that match the provided params
func (m *OrderManager) GetFuturesPositionsForExchange(exch string, item asset.Item, pair currency.Pair) ([]order.PositionStats, error) {
	if m == nil {
		return nil, fmt.Errorf("order manager %w", ErrNilSubsystem)
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return nil, fmt.Errorf("order manager %w", ErrSubSystemNotStarted)
	}
	if m.orderStore.futuresPositionController == nil {
		return nil, errFuturesTrackerNotSetup
	}
	if !item.IsFutures() {
		return nil, fmt.Errorf("%v %w", item, order.ErrNotFuturesAsset)
	}

	return m.orderStore.futuresPositionController.GetPositionsForExchange(exch, item, pair)
}

// ClearFuturesTracking will clear existing futures positions for a given exchange,
// asset, pair for the event that positions have not been tracked accurately
func (m *OrderManager) ClearFuturesTracking(exch string, item asset.Item, pair currency.Pair) error {
	if m == nil {
		return fmt.Errorf("order manager %w", ErrNilSubsystem)
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return fmt.Errorf("order manager %w", ErrSubSystemNotStarted)
	}
	if m.orderStore.futuresPositionController == nil {
		return errFuturesTrackerNotSetup
	}
	if !item.IsFutures() {
		return fmt.Errorf("%v %w", item, order.ErrNotFuturesAsset)
	}

	return m.orderStore.futuresPositionController.ClearPositionsForExchange(exch, item, pair)
}

// UpdateOpenPositionUnrealisedPNL finds an open position from
// an exchange asset pair, then calculates the unrealisedPNL
// using the latest ticker data
func (m *OrderManager) UpdateOpenPositionUnrealisedPNL(e string, item asset.Item, pair currency.Pair, last float64, updated time.Time) (decimal.Decimal, error) {
	if m == nil {
		return decimal.Zero, fmt.Errorf("order manager %w", ErrNilSubsystem)
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return decimal.Zero, fmt.Errorf("order manager %w", ErrSubSystemNotStarted)
	}
	if m.orderStore.futuresPositionController == nil {
		return decimal.Zero, errFuturesTrackerNotSetup
	}
	if !item.IsFutures() {
		return decimal.Zero, fmt.Errorf("%v %w", item, order.ErrNotFuturesAsset)
	}

	return m.orderStore.futuresPositionController.UpdateOpenPositionUnrealisedPNL(e, item, pair, last, updated)
}

// GetOrderInfo calls the exchange's wrapper GetOrderInfo function
// and stores the result in the order manager
func (m *OrderManager) GetOrderInfo(ctx context.Context, exchangeName, orderID string, cp currency.Pair, a asset.Item) (order.Detail, error) {
	if m == nil {
		return order.Detail{}, fmt.Errorf("order manager %w", ErrNilSubsystem)
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return order.Detail{}, fmt.Errorf("order manager %w", ErrSubSystemNotStarted)
	}

	if orderID == "" {
		return order.Detail{}, ErrOrderIDCannotBeEmpty
	}

	exch, err := m.orderStore.exchangeManager.GetExchangeByName(exchangeName)
	if err != nil {
		return order.Detail{}, err
	}
	result, err := exch.GetOrderInfo(ctx, orderID, cp, a)
	if err != nil {
		return order.Detail{}, err
	}

	upsertResponse, err := m.orderStore.upsert(&result)
	if err != nil {
		return order.Detail{}, err
	}

	return upsertResponse.OrderDetails, nil
}

// validate ensures a submitted order is valid before adding to the manager
func (m *OrderManager) validate(newOrder *order.Submit) error {
	if newOrder == nil {
		return errNilOrder
	}

	if newOrder.Exchange == "" {
		return ErrExchangeNameIsEmpty
	}

	if err := newOrder.Validate(); err != nil {
		return fmt.Errorf("order manager: %w", err)
	}

	if m.cfg.EnforceLimitConfig {
		if !m.cfg.AllowMarketOrders && newOrder.Type == order.Market {
			return errors.New("order market type is not allowed")
		}

		if m.cfg.LimitAmount > 0 && newOrder.Amount > m.cfg.LimitAmount {
			return errors.New("order limit exceeds allowed limit")
		}

		if len(m.cfg.AllowedExchanges) > 0 &&
			!common.StringDataCompareInsensitive(m.cfg.AllowedExchanges, newOrder.Exchange) {
			return errors.New("order exchange not found in allowed list")
		}

		if len(m.cfg.AllowedPairs) > 0 && !m.cfg.AllowedPairs.Contains(newOrder.Pair, true) {
			return errors.New("order pair not found in allowed list")
		}
	}
	return nil
}

// Modify depends on the order.Modify.ID and order.Modify.Exchange fields to uniquely
// identify an order to modify.
func (m *OrderManager) Modify(ctx context.Context, mod *order.Modify) (*order.ModifyResponse, error) {
	if m == nil {
		return nil, fmt.Errorf("order manager %w", ErrNilSubsystem)
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return nil, fmt.Errorf("order manager %w", ErrSubSystemNotStarted)
	}

	// Fetch details from locally managed order store.
	det, err := m.orderStore.getByExchangeAndID(mod.Exchange, mod.ID)
	if det == nil || err != nil {
		return nil, fmt.Errorf("order does not exist: %w", err)
	}

	// Populate additional Modify fields as some of them are required by various
	// exchange implementations.
	mod.Pair = det.Pair                           // Used by Bithumb.
	mod.Side = det.Side                           // Used by Bithumb.
	mod.PostOnly = det.PostOnly                   // Used by Poloniex.
	mod.ImmediateOrCancel = det.ImmediateOrCancel // Used by Poloniex.

	// Following is just a precaution to not modify orders by mistake if exchange
	// implementations do not check fields of the Modify struct for zero values.
	if mod.Amount == 0 {
		mod.Amount = det.Amount
	}
	if mod.Price == 0 {
		mod.Price = det.Price
	}

	// Get exchange instance and submit order modification request.
	exch, err := m.orderStore.exchangeManager.GetExchangeByName(mod.Exchange)
	if err != nil {
		return nil, err
	}
	res, err := exch.ModifyOrder(ctx, mod)
	if err != nil {
		message := fmt.Sprintf(
			"Order manager: Exchange %s order ID=%v: failed to modify",
			mod.Exchange,
			mod.ID,
		)
		m.orderStore.commsManager.PushEvent(base.Event{
			Type:    "order",
			Message: message,
		})
		return nil, err
	}

	// If modification is successful, apply changes to local order store.
	//
	// XXX: This comes with a race condition, because [request -> changes] are not
	// atomic.
	err = m.orderStore.modifyExisting(mod.ID, &res)

	// Notify observers.
	var message string
	if err != nil {
		message = "Order manager: Exchange %s order ID=%v: modified on exchange, but failed to modify locally"
	} else {
		message = "Order manager: Exchange %s order ID=%v: modified successfully"
	}
	m.orderStore.commsManager.PushEvent(base.Event{
		Type:    "order",
		Message: fmt.Sprintf(message, mod.Exchange, res.ID),
	})
	return &order.ModifyResponse{OrderID: res.ID}, err
}

// Submit will take in an order struct, send it to the exchange and
// populate it in the OrderManager if successful
func (m *OrderManager) Submit(ctx context.Context, newOrder *order.Submit) (*OrderSubmitResponse, error) {
	if m == nil {
		return nil, fmt.Errorf("order manager %w", ErrNilSubsystem)
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return nil, fmt.Errorf("order manager %w", ErrSubSystemNotStarted)
	}

	err := m.validate(newOrder)
	if err != nil {
		return nil, err
	}
	exch, err := m.orderStore.exchangeManager.GetExchangeByName(newOrder.Exchange)
	if err != nil {
		return nil, err
	}

	// Checks for exchange min max limits for order amounts before order
	// execution can occur
	err = exch.CheckOrderExecutionLimits(newOrder.AssetType,
		newOrder.Pair,
		newOrder.Price,
		newOrder.Amount,
		newOrder.Type)
	if err != nil {
		return nil, fmt.Errorf("order manager: exchange %s unable to place order: %w",
			newOrder.Exchange,
			err)
	}

	// Determines if current trading activity is turned off by the exchange for
	// the currency pair
	err = exch.CanTradePair(newOrder.Pair, newOrder.AssetType)
	if err != nil {
		return nil, fmt.Errorf("order manager: exchange %s cannot trade pair %s %s: %w",
			newOrder.Exchange,
			newOrder.Pair,
			newOrder.AssetType,
			err)
	}

	result, err := exch.SubmitOrder(ctx, newOrder)
	if err != nil {
		return nil, err
	}

	return m.processSubmittedOrder(newOrder, result)
}

// SubmitFakeOrder runs through the same process as order submission
// but does not touch live endpoints
func (m *OrderManager) SubmitFakeOrder(newOrder *order.Submit, resultingOrder order.SubmitResponse, checkExchangeLimits bool) (*OrderSubmitResponse, error) {
	if m == nil {
		return nil, fmt.Errorf("order manager %w", ErrNilSubsystem)
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return nil, fmt.Errorf("order manager %w", ErrSubSystemNotStarted)
	}

	err := m.validate(newOrder)
	if err != nil {
		return nil, err
	}
	exch, err := m.orderStore.exchangeManager.GetExchangeByName(newOrder.Exchange)
	if err != nil {
		return nil, err
	}

	if checkExchangeLimits {
		// Checks for exchange min max limits for order amounts before order
		// execution can occur
		err = exch.CheckOrderExecutionLimits(newOrder.AssetType,
			newOrder.Pair,
			newOrder.Price,
			newOrder.Amount,
			newOrder.Type)
		if err != nil {
			return nil, fmt.Errorf("order manager: exchange %s unable to place order: %w",
				newOrder.Exchange,
				err)
		}
	}
	return m.processSubmittedOrder(newOrder, resultingOrder)
}

// GetOrdersSnapshot returns a snapshot of all orders in the orderstore. It optionally filters any orders that do not match the status
// but a status of "" or ANY will include all
// the time adds contexts for when the snapshot is relevant for
func (m *OrderManager) GetOrdersSnapshot(s order.Status) []order.Detail {
	if m == nil || atomic.LoadInt32(&m.started) == 0 {
		return nil
	}
	var os []order.Detail
	for _, v := range m.orderStore.Orders {
		for i := range v {
			if s != v[i].Status &&
				s != order.AnyStatus &&
				s != "" {
				continue
			}
			os = append(os, *v[i])
		}
	}

	return os
}

// GetOrdersFiltered returns a snapshot of all orders in the order store.
// Filtering is applied based on the order.Filter unless entries are empty
func (m *OrderManager) GetOrdersFiltered(f *order.Filter) ([]order.Detail, error) {
	if m == nil {
		return nil, fmt.Errorf("order manager %w", ErrNilSubsystem)
	}
	if f == nil {
		return nil, fmt.Errorf("order manager, GetOrdersFiltered: Filter is nil")
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return nil, fmt.Errorf("order manager %w", ErrSubSystemNotStarted)
	}
	return m.orderStore.getFilteredOrders(f)
}

// GetOrdersActive returns a snapshot of all orders in the order store
// that have a status that indicates it's currently tradable
func (m *OrderManager) GetOrdersActive(f *order.Filter) ([]order.Detail, error) {
	if m == nil {
		return nil, fmt.Errorf("order manager %w", ErrNilSubsystem)
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return nil, fmt.Errorf("order manager %w", ErrSubSystemNotStarted)
	}
	return m.orderStore.getActiveOrders(f), nil
}

// processSubmittedOrder adds a new order to the manager
func (m *OrderManager) processSubmittedOrder(newOrder *order.Submit, result order.SubmitResponse) (*OrderSubmitResponse, error) {
	if !result.IsOrderPlaced {
		return nil, errUnableToPlaceOrder
	}

	id, err := uuid.NewV4()
	if err != nil {
		log.Warnf(log.OrderMgr,
			"Order manager: Unable to generate UUID. Err: %s",
			err)
	}
	if newOrder.Date.IsZero() {
		newOrder.Date = time.Now()
	}
	msg := fmt.Sprintf("Order manager: Exchange %s submitted order ID=%v [Ours: %v] pair=%v price=%v amount=%v quoteAmount=%v side=%v type=%v for time %v.",
		newOrder.Exchange,
		result.OrderID,
		id.String(),
		newOrder.Pair,
		newOrder.Price,
		newOrder.Amount,
		newOrder.QuoteAmount,
		newOrder.Side,
		newOrder.Type,
		newOrder.Date)

	log.Debugln(log.OrderMgr, msg)
	m.orderStore.commsManager.PushEvent(base.Event{
		Type:    "order",
		Message: msg,
	})
	status := order.New
	if result.FullyMatched {
		status = order.Filled
	}
	err = m.orderStore.add(&order.Detail{
		ImmediateOrCancel: newOrder.ImmediateOrCancel,
		HiddenOrder:       newOrder.HiddenOrder,
		FillOrKill:        newOrder.FillOrKill,
		PostOnly:          newOrder.PostOnly,
		Price:             newOrder.Price,
		Amount:            newOrder.Amount,
		LimitPriceUpper:   newOrder.LimitPriceUpper,
		LimitPriceLower:   newOrder.LimitPriceLower,
		TriggerPrice:      newOrder.TriggerPrice,
		QuoteAmount:       newOrder.QuoteAmount,
		ExecutedAmount:    newOrder.ExecutedAmount,
		RemainingAmount:   newOrder.RemainingAmount,
		Fee:               newOrder.Fee,
		Exchange:          newOrder.Exchange,
		InternalOrderID:   id.String(),
		ID:                result.OrderID,
		AccountID:         newOrder.AccountID,
		ClientID:          newOrder.ClientID,
		ClientOrderID:     newOrder.ClientOrderID,
		WalletAddress:     newOrder.WalletAddress,
		Type:              newOrder.Type,
		Side:              newOrder.Side,
		Status:            status,
		AssetType:         newOrder.AssetType,
		Date:              time.Now(),
		LastUpdated:       time.Now(),
		Pair:              newOrder.Pair,
		Leverage:          newOrder.Leverage,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to add %v order %v to orderStore: %s", newOrder.Exchange, result.OrderID, err)
	}

	return &OrderSubmitResponse{
		SubmitResponse: order.SubmitResponse{
			IsOrderPlaced: result.IsOrderPlaced,
			OrderID:       result.OrderID,
		},
		InternalOrderID: id.String(),
	}, nil
}

// processOrders iterates over all exchange orders via API
// and adds them to the internal order store
func (m *OrderManager) processOrders() {
	if !atomic.CompareAndSwapInt32(&m.processingOrders, 0, 1) {
		return
	}
	defer func() {
		atomic.StoreInt32(&m.processingOrders, 0)
	}()
	exchanges, err := m.orderStore.exchangeManager.GetExchanges()
	if err != nil {
		log.Errorf(log.OrderMgr, "Order manager cannot get exchanges: %v", err)
		return
	}
	var wg sync.WaitGroup
	for i := range exchanges {
		if !exchanges[i].GetAuthenticatedAPISupport(exchange.RestAuthentication) {
			continue
		}
		log.Debugf(log.OrderMgr,
			"Order manager: Processing orders for exchange %v.",
			exchanges[i].GetName())

		supportedAssets := exchanges[i].GetAssetTypes(true)
		for y := range supportedAssets {
			pairs, err := exchanges[i].GetEnabledPairs(supportedAssets[y])
			if err != nil {
				log.Errorf(log.OrderMgr,
					"Order manager: Unable to get enabled pairs for %s and asset type %s: %s",
					exchanges[i].GetName(),
					supportedAssets[y],
					err)
				continue
			}

			if len(pairs) == 0 {
				if m.verbose {
					log.Debugf(log.OrderMgr,
						"Order manager: No pairs enabled for %s and asset type %s, skipping...",
						exchanges[i].GetName(),
						supportedAssets[y])
				}
				continue
			}

			filter := &order.Filter{
				Exchange: exchanges[i].GetName(),
			}
			orders := m.orderStore.getActiveOrders(filter)
			order.FilterOrdersByCurrencies(&orders, pairs)
			requiresProcessing := make(map[string]bool, len(orders))
			for x := range orders {
				requiresProcessing[orders[x].InternalOrderID] = true
			}

			req := order.GetOrdersRequest{
				Side:      order.AnySide,
				Type:      order.AnyType,
				Pairs:     pairs,
				AssetType: supportedAssets[y],
			}
			result, err := exchanges[i].GetActiveOrders(context.TODO(), &req)
			if err != nil {
				log.Errorf(log.OrderMgr,
					"Order manager: Unable to get active orders for %s and asset type %s: %s",
					exchanges[i].GetName(),
					supportedAssets[y],
					err)
				continue
			}
			if len(orders) == 0 && len(result) == 0 {
				continue
			}

			for z := range result {
				upsertResponse, err := m.UpsertOrder(&result[z])
				if err != nil {
					log.Error(log.OrderMgr, err)
				} else {
					requiresProcessing[upsertResponse.OrderDetails.InternalOrderID] = false
				}
			}
			if !exchanges[i].GetBase().GetSupportedFeatures().RESTCapabilities.GetOrder {
				continue
			}
			wg.Add(1)
			go m.processMatchingOrders(exchanges[i], orders, requiresProcessing, &wg)
		}
	}
	wg.Wait()
}

func (m *OrderManager) processMatchingOrders(exch exchange.IBotExchange, orders []order.Detail, requiresProcessing map[string]bool, wg *sync.WaitGroup) {
	defer func() {
		if wg != nil {
			wg.Done()
		}
	}()
	for x := range orders {
		if time.Since(orders[x].LastUpdated) < time.Minute {
			continue
		}
		if requiresProcessing[orders[x].InternalOrderID] {
			err := m.FetchAndUpdateExchangeOrder(exch, &orders[x], orders[x].AssetType)
			if err != nil {
				log.Error(log.OrderMgr, err)
			}
		}
	}
}

// FetchAndUpdateExchangeOrder calls the exchange to upsert an order to the order store
func (m *OrderManager) FetchAndUpdateExchangeOrder(exch exchange.IBotExchange, ord *order.Detail, assetType asset.Item) error {
	if ord == nil {
		return errors.New("order manager: Order is nil")
	}
	fetchedOrder, err := exch.GetOrderInfo(context.TODO(), ord.ID, ord.Pair, assetType)
	if err != nil {
		ord.Status = order.UnknownStatus
		return err
	}
	fetchedOrder.LastUpdated = time.Now()
	_, err = m.UpsertOrder(&fetchedOrder)
	return err
}

// Exists checks whether an order exists in the order store
func (m *OrderManager) Exists(o *order.Detail) bool {
	if m == nil || atomic.LoadInt32(&m.started) == 0 {
		return false
	}

	return m.orderStore.exists(o)
}

// Add adds an order to the orderstore
func (m *OrderManager) Add(o *order.Detail) error {
	if m == nil {
		return fmt.Errorf("order manager %w", ErrNilSubsystem)
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return fmt.Errorf("order manager %w", ErrSubSystemNotStarted)
	}

	return m.orderStore.add(o)
}

// GetByExchangeAndID returns a copy of an order from an exchange if it matches the ID
func (m *OrderManager) GetByExchangeAndID(exchangeName, id string) (*order.Detail, error) {
	if m == nil {
		return nil, fmt.Errorf("order manager %w", ErrNilSubsystem)
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return nil, fmt.Errorf("order manager %w", ErrSubSystemNotStarted)
	}

	o, err := m.orderStore.getByExchangeAndID(exchangeName, id)
	if err != nil {
		return nil, err
	}
	var cpy order.Detail
	cpy.UpdateOrderFromDetail(o)
	return &cpy, nil
}

// UpdateExistingOrder will update an existing order in the orderstore
func (m *OrderManager) UpdateExistingOrder(od *order.Detail) error {
	if m == nil {
		return fmt.Errorf("order manager %w", ErrNilSubsystem)
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return fmt.Errorf("order manager %w", ErrSubSystemNotStarted)
	}
	return m.orderStore.updateExisting(od)
}

// UpsertOrder updates an existing order or adds a new one to the orderstore
func (m *OrderManager) UpsertOrder(od *order.Detail) (resp *OrderUpsertResponse, err error) {
	if m == nil {
		return nil, fmt.Errorf("order manager %w", ErrNilSubsystem)
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return nil, fmt.Errorf("order manager %w", ErrSubSystemNotStarted)
	}
	if od == nil {
		return nil, errNilOrder
	}
	var msg string
	defer func(message *string) {
		if message == nil {
			log.Errorf(log.OrderMgr, "UpsertOrder: produced nil order event message\n")
			return
		}
		m.orderStore.commsManager.PushEvent(base.Event{
			Type:    "order",
			Message: *message,
		})
	}(&msg)

	upsertResponse, err := m.orderStore.upsert(od)
	if err != nil {
		msg = fmt.Sprintf(
			"Order manager: Exchange %s unable to upsert order ID=%v internal ID=%v pair=%v price=%.8f amount=%.8f side=%v type=%v status=%v: %s",
			od.Exchange, od.ID, od.InternalOrderID, od.Pair, od.Price, od.Amount, od.Side, od.Type, od.Status, err)
		return nil, err
	}

	status := "updated"
	if upsertResponse.IsNewOrder {
		status = "added"
	}
	msg = fmt.Sprintf("Order manager: Exchange %s %s order ID=%v internal ID=%v pair=%v price=%.8f amount=%.8f side=%v type=%v status=%v.",
		upsertResponse.OrderDetails.Exchange, status, upsertResponse.OrderDetails.ID, upsertResponse.OrderDetails.InternalOrderID,
		upsertResponse.OrderDetails.Pair, upsertResponse.OrderDetails.Price, upsertResponse.OrderDetails.Amount,
		upsertResponse.OrderDetails.Side, upsertResponse.OrderDetails.Type, upsertResponse.OrderDetails.Status)
	if upsertResponse.IsNewOrder {
		log.Info(log.OrderMgr, msg)
		return upsertResponse, nil
	}
	log.Debug(log.OrderMgr, msg)
	return upsertResponse, nil
}

// get returns all orders for all exchanges
// should not be exported as it can have large impact if used improperly
func (s *store) get() map[string][]*order.Detail {
	s.m.Lock()
	orders := s.Orders
	s.m.Unlock()
	return orders
}

// getByExchangeAndID returns a specific order by exchange and id
func (s *store) getByExchangeAndID(exchange, id string) (*order.Detail, error) {
	s.m.Lock()
	defer s.m.Unlock()
	r, ok := s.Orders[strings.ToLower(exchange)]
	if !ok {
		return nil, ErrExchangeNotFound
	}

	for x := range r {
		if r[x].ID == id {
			return r[x], nil
		}
	}
	return nil, ErrOrderNotFound
}

// updateExisting checks if an order exists in the orderstore
// and then updates it
func (s *store) updateExisting(od *order.Detail) error {
	if od == nil {
		return errNilOrder
	}
	s.m.Lock()
	defer s.m.Unlock()
	r, ok := s.Orders[strings.ToLower(od.Exchange)]
	if !ok {
		return ErrExchangeNotFound
	}
	for x := range r {
		if r[x].ID == od.ID {
			r[x].UpdateOrderFromDetail(od)
			if r[x].AssetType.IsFutures() {
				err := s.futuresPositionController.TrackNewOrder(r[x])
				if err != nil {
					if !errors.Is(err, order.ErrPositionClosed) {
						return err
					}
				}
			}
			return nil
		}
	}

	return ErrOrderNotFound
}

// modifyExisting depends on mod.Exchange and given ID to uniquely identify an order and
// modify it.
func (s *store) modifyExisting(id string, mod *order.Modify) error {
	s.m.Lock()
	defer s.m.Unlock()
	r, ok := s.Orders[strings.ToLower(mod.Exchange)]
	if !ok {
		return ErrExchangeNotFound
	}
	for x := range r {
		if r[x].ID == id {
			r[x].UpdateOrderFromModify(mod)
			if r[x].AssetType.IsFutures() {
				err := s.futuresPositionController.TrackNewOrder(r[x])
				if err != nil {
					if !errors.Is(err, order.ErrPositionClosed) {
						return err
					}
				}
			}
			return nil
		}
	}
	return ErrOrderNotFound
}

// upsert (1) checks if such an exchange exists in the exchangeManager, (2) checks if
// order exists and updates/creates it.
func (s *store) upsert(od *order.Detail) (resp *OrderUpsertResponse, err error) {
	if od == nil {
		return nil, errNilOrder
	}
	lName := strings.ToLower(od.Exchange)
	_, err = s.exchangeManager.GetExchangeByName(lName)
	if err != nil {
		return nil, err
	}
	s.m.Lock()
	defer s.m.Unlock()
	if od.AssetType.IsFutures() {
		err = s.futuresPositionController.TrackNewOrder(od)
		if err != nil {
			if !errors.Is(err, order.ErrPositionClosed) {
				return nil, err
			}
		}
	}
	r, ok := s.Orders[lName]
	if !ok {
		od.GenerateInternalOrderID()
		s.Orders[lName] = []*order.Detail{od}
		resp = &OrderUpsertResponse{
			OrderDetails: od.Copy(),
			IsNewOrder:   true,
		}
		return resp, nil
	}
	for x := range r {
		if r[x].ID == od.ID {
			r[x].UpdateOrderFromDetail(od)
			resp = &OrderUpsertResponse{
				OrderDetails: r[x].Copy(),
				IsNewOrder:   false,
			}
			return resp, nil
		}
	}
	// Untracked websocket orders will not have internalIDs yet
	od.GenerateInternalOrderID()
	s.Orders[lName] = append(s.Orders[lName], od)
	resp = &OrderUpsertResponse{
		OrderDetails: od.Copy(),
		IsNewOrder:   true,
	}
	return resp, nil
}

// getByExchange returns orders by exchange
func (s *store) getByExchange(exchange string) ([]*order.Detail, error) {
	s.m.RLock()
	defer s.m.RUnlock()
	r, ok := s.Orders[strings.ToLower(exchange)]
	if !ok {
		return nil, ErrExchangeNotFound
	}
	return r, nil
}

// getByInternalOrderID will search all orders for our internal orderID
// and return the order
func (s *store) getByInternalOrderID(internalOrderID string) (*order.Detail, error) {
	s.m.RLock()
	defer s.m.RUnlock()
	for _, v := range s.Orders {
		for x := range v {
			if v[x].InternalOrderID == internalOrderID {
				return v[x], nil
			}
		}
	}
	return nil, ErrOrderNotFound
}

// exists verifies if the orderstore contains the provided order
func (s *store) exists(det *order.Detail) bool {
	if det == nil {
		return false
	}
	s.m.RLock()
	defer s.m.RUnlock()
	r, ok := s.Orders[strings.ToLower(det.Exchange)]
	if !ok {
		return false
	}

	for x := range r {
		if r[x].ID == det.ID {
			return true
		}
	}
	return false
}

// Add Adds an order to the orderStore for tracking the lifecycle
func (s *store) add(det *order.Detail) error {
	if det == nil {
		return errNilOrder
	}
	_, err := s.exchangeManager.GetExchangeByName(det.Exchange)
	if err != nil {
		return err
	}
	if s.exists(det) {
		return ErrOrdersAlreadyExists
	}
	// Untracked websocket orders will not have internalIDs yet
	det.GenerateInternalOrderID()
	s.m.Lock()
	defer s.m.Unlock()
	orders := s.Orders[strings.ToLower(det.Exchange)]
	orders = append(orders, det)
	s.Orders[strings.ToLower(det.Exchange)] = orders

	if det.AssetType.IsFutures() {
		err = s.futuresPositionController.TrackNewOrder(det)
		if err != nil {
			return err
		}
	}
	return nil
}

// getFilteredOrders returns a filtered copy of the orders
func (s *store) getFilteredOrders(f *order.Filter) ([]order.Detail, error) {
	if f == nil {
		return nil, errors.New("filter is nil")
	}
	s.m.RLock()
	defer s.m.RUnlock()

	var os []order.Detail
	// optimization if Exchange is filtered
	if f.Exchange != "" {
		if e, ok := s.Orders[strings.ToLower(f.Exchange)]; ok {
			for i := range e {
				if !e[i].MatchFilter(f) {
					continue
				}
				os = append(os, e[i].Copy())
			}
		}
	} else {
		for _, e := range s.Orders {
			for i := range e {
				if !e[i].MatchFilter(f) {
					continue
				}
				os = append(os, e[i].Copy())
			}
		}
	}
	return os, nil
}

// getActiveOrders returns copy of the orders that are active
func (s *store) getActiveOrders(f *order.Filter) []order.Detail {
	s.m.RLock()
	defer s.m.RUnlock()

	var orders []order.Detail
	switch {
	case f == nil:
		for _, e := range s.Orders {
			for i := range e {
				if !e[i].IsActive() {
					continue
				}
				orders = append(orders, e[i].Copy())
			}
		}
	case f.Exchange != "":
		// optimization if Exchange is filtered
		if e, ok := s.Orders[strings.ToLower(f.Exchange)]; ok {
			for i := range e {
				if !e[i].IsActive() || !e[i].MatchFilter(f) {
					continue
				}
				orders = append(orders, e[i].Copy())
			}
		}
	default:
		for _, e := range s.Orders {
			for i := range e {
				if !e[i].IsActive() || !e[i].MatchFilter(f) {
					continue
				}
				orders = append(orders, e[i].Copy())
			}
		}
	}

	return orders
}
