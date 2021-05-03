package engine

import (
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/communications/base"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine/subsystem"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// vars for the fund manager package
var (
	orderManagerDelay      = time.Second * 10
	ErrOrdersAlreadyExists = errors.New("order already exists")
	ErrOrderNotFound       = errors.New("order does not exist")
)

// get returns all orders for all exchanges
// should not be exported as it can have large impact if used improperly
func (o *orderStore) get() map[string][]*order.Detail {
	o.m.RLock()
	orders := o.Orders
	o.m.RUnlock()
	return orders
}

// GetByExchangeAndID returns a specific order by exchange and id
func (o *orderStore) GetByExchangeAndID(exchange, id string) (*order.Detail, error) {
	o.m.RLock()
	defer o.m.RUnlock()
	r, ok := o.Orders[strings.ToLower(exchange)]
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

// GetByExchange returns orders by exchange
func (o *orderStore) GetByExchange(exchange string) ([]*order.Detail, error) {
	o.m.RLock()
	defer o.m.RUnlock()
	r, ok := o.Orders[strings.ToLower(exchange)]
	if !ok {
		return nil, ErrExchangeNotFound
	}
	return r, nil
}

// GetByInternalOrderID will search all orders for our internal orderID
// and return the order
func (o *orderStore) GetByInternalOrderID(internalOrderID string) (*order.Detail, error) {
	o.m.RLock()
	defer o.m.RUnlock()
	for _, v := range o.Orders {
		for x := range v {
			if v[x].InternalOrderID == internalOrderID {
				return v[x], nil
			}
		}
	}
	return nil, ErrOrderNotFound
}

func (o *orderStore) exists(det *order.Detail) bool {
	if det == nil {
		return false
	}
	o.m.RLock()
	defer o.m.RUnlock()
	r, ok := o.Orders[strings.ToLower(det.Exchange)]
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
func (o *orderStore) Add(det *order.Detail) error {
	if det == nil {
		return errors.New("order store: Order is nil")
	}
	exch := o.bot.GetExchangeByName(det.Exchange)
	if exch == nil {
		return ErrExchangeNotFound
	}
	if o.exists(det) {
		return ErrOrdersAlreadyExists
	}
	// Untracked websocket orders will not have internalIDs yet
	if det.InternalOrderID == "" {
		id, err := uuid.NewV4()
		if err != nil {
			log.Warnf(log.OrderMgr,
				"Order manager: Unable to generate UUID. Err: %s",
				err)
		} else {
			det.InternalOrderID = id.String()
		}
	}
	o.m.Lock()
	defer o.m.Unlock()
	orders := o.Orders[strings.ToLower(det.Exchange)]
	orders = append(orders, det)
	o.Orders[strings.ToLower(det.Exchange)] = orders

	return nil
}

// Started returns the status of the orderManager
func (o *orderManager) Started() bool {
	return atomic.LoadInt32(&o.started) == 1
}

// Start will boot up the orderManager
func (o *orderManager) Start(bot *Engine) error {
	if bot == nil {
		return errors.New("cannot start with nil bot")
	}
	if !atomic.CompareAndSwapInt32(&o.started, 0, 1) {
		return fmt.Errorf("order manager %w", subsystem.ErrSubSystemAlreadyStarted)
	}
	log.Debugln(log.OrderBook, "Order manager starting...")

	o.shutdown = make(chan struct{})
	o.orderStore.Orders = make(map[string][]*order.Detail)
	o.orderStore.bot = bot

	go o.run()
	return nil
}

// Stop will attempt to shutdown the orderManager
func (o *orderManager) Stop() error {
	if atomic.LoadInt32(&o.started) == 0 {
		return fmt.Errorf("order manager %w", subsystem.ErrSubSystemNotStarted)
	}

	defer func() {
		atomic.CompareAndSwapInt32(&o.started, 1, 0)
	}()

	log.Debugln(log.OrderBook, "Order manager shutting down...")
	close(o.shutdown)
	return nil
}

func (o *orderManager) gracefulShutdown() {
	if o.cfg.CancelOrdersOnShutdown {
		log.Debugln(log.OrderMgr, "Order manager: Cancelling any open orders...")
		o.CancelAllOrders(o.orderStore.bot.Config.GetEnabledExchanges())
	}
}

func (o *orderManager) run() {
	log.Debugln(log.OrderBook, "Order manager started.")
	tick := time.NewTicker(orderManagerDelay)
	o.orderStore.bot.ServicesWG.Add(1)
	defer func() {
		log.Debugln(log.OrderMgr, "Order manager shutdown.")
		tick.Stop()
		o.orderStore.bot.ServicesWG.Done()
	}()

	for {
		select {
		case <-o.shutdown:
			o.gracefulShutdown()
			return
		case <-tick.C:
			go o.processOrders()
		}
	}
}

// CancelAllOrders iterates and cancels all orders for each exchange provided
func (o *orderManager) CancelAllOrders(exchangeNames []string) {
	orders := o.orderStore.get()
	if orders == nil {
		return
	}

	for k, v := range orders {
		log.Debugf(log.OrderMgr, "Order manager: Cancelling order(s) for exchange %s.", k)
		if !common.StringDataCompareInsensitive(exchangeNames, k) {
			continue
		}

		for y := range v {
			err := o.Cancel(&order.Cancel{
				Exchange:      k,
				ID:            v[y].ID,
				AccountID:     v[y].AccountID,
				ClientID:      v[y].ClientID,
				WalletAddress: v[y].WalletAddress,
				Type:          v[y].Type,
				Side:          v[y].Side,
				Pair:          v[y].Pair,
				AssetType:     v[y].AssetType,
			})
			if err != nil {
				log.Error(log.OrderMgr, err)
				continue
			}
		}
	}
}

// Cancel will find the order in the orderManager, send a cancel request
// to the exchange and if successful, update the status of the order
func (o *orderManager) Cancel(cancel *order.Cancel) error {
	var err error
	defer func() {
		if err != nil {
			o.orderStore.bot.CommsManager.PushEvent(base.Event{
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

	exch := o.orderStore.bot.GetExchangeByName(cancel.Exchange)
	if exch == nil {
		err = ErrExchangeNotFound
		return err
	}

	if cancel.AssetType.String() != "" && !exch.GetAssetTypes().Contains(cancel.AssetType) {
		err = errors.New("order asset type not supported by exchange")
		return err
	}

	log.Debugf(log.OrderMgr, "Order manager: Cancelling order ID %v [%+v]",
		cancel.ID, cancel)

	err = exch.CancelOrder(cancel)
	if err != nil {
		err = fmt.Errorf("%v - Failed to cancel order: %v", cancel.Exchange, err)
		return err
	}
	var od *order.Detail
	od, err = o.orderStore.GetByExchangeAndID(cancel.Exchange, cancel.ID)
	if err != nil {
		err = fmt.Errorf("%v - Failed to retrieve order %v to update cancelled status: %v", cancel.Exchange, cancel.ID, err)
		return err
	}

	od.Status = order.Cancelled
	msg := fmt.Sprintf("Order manager: Exchange %s order ID=%v cancelled.",
		od.Exchange, od.ID)
	log.Debugln(log.OrderMgr, msg)
	o.orderStore.bot.CommsManager.PushEvent(base.Event{
		Type:    "order",
		Message: msg,
	})

	return nil
}

// GetOrderInfo calls the exchange's wrapper GetOrderInfo function
// and stores the result in the order manager
func (o *orderManager) GetOrderInfo(exchangeName, orderID string, cp currency.Pair, a asset.Item) (order.Detail, error) {
	if orderID == "" {
		return order.Detail{}, errOrderIDCannotBeEmpty
	}

	exch := o.orderStore.bot.GetExchangeByName(exchangeName)
	if exch == nil {
		return order.Detail{}, ErrExchangeNotFound
	}
	result, err := exch.GetOrderInfo(orderID, cp, a)
	if err != nil {
		return order.Detail{}, err
	}

	err = o.orderStore.Add(&result)
	if err != nil && err != ErrOrdersAlreadyExists {
		return order.Detail{}, err
	}

	return result, nil
}

func (o *orderManager) validate(newOrder *order.Submit) error {
	if newOrder == nil {
		return errors.New("order cannot be nil")
	}

	if newOrder.Exchange == "" {
		return errors.New("order exchange name must be specified")
	}

	if err := newOrder.Validate(); err != nil {
		return fmt.Errorf("order manager: %w", err)
	}

	if o.cfg.EnforceLimitConfig {
		if !o.cfg.AllowMarketOrders && newOrder.Type == order.Market {
			return errors.New("order market type is not allowed")
		}

		if o.cfg.LimitAmount > 0 && newOrder.Amount > o.cfg.LimitAmount {
			return errors.New("order limit exceeds allowed limit")
		}

		if len(o.cfg.AllowedExchanges) > 0 &&
			!common.StringDataCompareInsensitive(o.cfg.AllowedExchanges, newOrder.Exchange) {
			return errors.New("order exchange not found in allowed list")
		}

		if len(o.cfg.AllowedPairs) > 0 && !o.cfg.AllowedPairs.Contains(newOrder.Pair, true) {
			return errors.New("order pair not found in allowed list")
		}
	}
	return nil
}

// Submit will take in an order struct, send it to the exchange and
// populate it in the orderManager if successful
func (o *orderManager) Submit(newOrder *order.Submit) (*orderSubmitResponse, error) {
	err := o.validate(newOrder)
	if err != nil {
		return nil, err
	}
	exch := o.orderStore.bot.GetExchangeByName(newOrder.Exchange)
	if exch == nil {
		return nil, ErrExchangeNotFound
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

	result, err := exch.SubmitOrder(newOrder)
	if err != nil {
		return nil, err
	}

	return o.processSubmittedOrder(newOrder, result)
}

// SubmitFakeOrder runs through the same process as order submission
// but does not touch live endpoints
func (o *orderManager) SubmitFakeOrder(newOrder *order.Submit, resultingOrder order.SubmitResponse, checkExchangeLimits bool) (*orderSubmitResponse, error) {
	err := o.validate(newOrder)
	if err != nil {
		return nil, err
	}
	exch := o.orderStore.bot.GetExchangeByName(newOrder.Exchange)
	if exch == nil {
		return nil, ErrExchangeNotFound
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
	return o.processSubmittedOrder(newOrder, resultingOrder)
}

// GetOrdersSnapshot returns a snapshot of all orders in the orderstore. It optionally filters any orders that do not match the status
// but a status of "" or ANY will include all
// the time adds contexts for the when the snapshot is relevant for
func (o *orderManager) GetOrdersSnapshot(s order.Status) ([]order.Detail, time.Time) {
	var os []order.Detail
	var latestUpdate time.Time
	for _, v := range o.orderStore.Orders {
		for i := range v {
			if s != v[i].Status &&
				s != order.AnyStatus &&
				s != "" {
				continue
			}
			if v[i].LastUpdated.After(latestUpdate) {
				latestUpdate = v[i].LastUpdated
			}

			cpy := *v[i]
			os = append(os, cpy)
		}
	}

	return os, latestUpdate
}

func (o *orderManager) processSubmittedOrder(newOrder *order.Submit, result order.SubmitResponse) (*orderSubmitResponse, error) {
	if !result.IsOrderPlaced {
		return nil, errors.New("order unable to be placed")
	}

	id, err := uuid.NewV4()
	if err != nil {
		log.Warnf(log.OrderMgr,
			"Order manager: Unable to generate UUID. Err: %s",
			err)
	}
	msg := fmt.Sprintf("Order manager: Exchange %s submitted order ID=%v [Ours: %v] pair=%v price=%v amount=%v side=%v type=%v for time %v.",
		newOrder.Exchange,
		result.OrderID,
		id.String(),
		newOrder.Pair,
		newOrder.Price,
		newOrder.Amount,
		newOrder.Side,
		newOrder.Type,
		newOrder.Date)

	log.Debugln(log.OrderMgr, msg)
	o.orderStore.bot.CommsManager.PushEvent(base.Event{
		Type:    "order",
		Message: msg,
	})
	status := order.New
	if result.FullyMatched {
		status = order.Filled
	}
	err = o.orderStore.Add(&order.Detail{
		ImmediateOrCancel: newOrder.ImmediateOrCancel,
		HiddenOrder:       newOrder.HiddenOrder,
		FillOrKill:        newOrder.FillOrKill,
		PostOnly:          newOrder.PostOnly,
		Price:             newOrder.Price,
		Amount:            newOrder.Amount,
		LimitPriceUpper:   newOrder.LimitPriceUpper,
		LimitPriceLower:   newOrder.LimitPriceLower,
		TriggerPrice:      newOrder.TriggerPrice,
		TargetAmount:      newOrder.TargetAmount,
		ExecutedAmount:    newOrder.ExecutedAmount,
		RemainingAmount:   newOrder.RemainingAmount,
		Fee:               newOrder.Fee,
		Exchange:          newOrder.Exchange,
		InternalOrderID:   id.String(),
		ID:                result.OrderID,
		AccountID:         newOrder.AccountID,
		ClientID:          newOrder.ClientID,
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

	return &orderSubmitResponse{
		SubmitResponse: order.SubmitResponse{
			IsOrderPlaced: result.IsOrderPlaced,
			OrderID:       result.OrderID,
		},
		InternalOrderID: id.String(),
	}, nil
}

func (o *orderManager) processOrders() {
	authExchanges := o.orderStore.bot.GetAuthAPISupportedExchanges()
	for x := range authExchanges {
		log.Debugf(log.OrderMgr,
			"Order manager: Processing orders for exchange %v.",
			authExchanges[x])

		exch := o.orderStore.bot.GetExchangeByName(authExchanges[x])
		supportedAssets := exch.GetAssetTypes()
		for y := range supportedAssets {
			pairs, err := exch.GetEnabledPairs(supportedAssets[y])
			if err != nil {
				log.Errorf(log.OrderMgr,
					"Order manager: Unable to get enabled pairs for %s and asset type %s: %s",
					authExchanges[x],
					supportedAssets[y],
					err)
				continue
			}

			if len(pairs) == 0 {
				if o.orderStore.bot.Settings.Verbose {
					log.Debugf(log.OrderMgr,
						"Order manager: No pairs enabled for %s and asset type %s, skipping...",
						authExchanges[x],
						supportedAssets[y])
				}
				continue
			}

			req := order.GetOrdersRequest{
				Side:      order.AnySide,
				Type:      order.AnyType,
				Pairs:     pairs,
				AssetType: supportedAssets[y],
			}
			result, err := exch.GetActiveOrders(&req)
			if err != nil {
				log.Warnf(log.OrderMgr,
					"Order manager: Unable to get active orders for %s and asset type %s: %s",
					authExchanges[x],
					supportedAssets[y],
					err)
				continue
			}

			for z := range result {
				ord := &result[z]
				result := o.orderStore.Add(ord)
				if result != ErrOrdersAlreadyExists {
					msg := fmt.Sprintf("Order manager: Exchange %s added order ID=%v pair=%v price=%v amount=%v side=%v type=%v.",
						ord.Exchange, ord.ID, ord.Pair, ord.Price, ord.Amount, ord.Side, ord.Type)
					log.Debugf(log.OrderMgr, "%v", msg)
					o.orderStore.bot.CommsManager.PushEvent(base.Event{
						Type:    "order",
						Message: msg,
					})
					continue
				}
			}
		}
	}
}
