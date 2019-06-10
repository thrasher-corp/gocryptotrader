package engine

import (
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/communications/base"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	log "github.com/thrasher-/gocryptotrader/logger"
)

// vars for the fund manager package
var (
	OrderManagerDelay      = time.Second * 10
	ErrOrdersAlreadyExists = errors.New("order already exists")
)

func (o *orderStore) Get() map[string][]exchange.OrderDetail {
	o.m.Lock()
	defer o.m.Unlock()
	return o.Orders
}

func (o *orderStore) exists(order *exchange.OrderDetail) bool {
	r, ok := o.Orders[order.Exchange]
	if !ok {
		return false
	}

	for x := range r {
		if r[x].ID == order.ID {
			return true
		}
	}

	return false
}

func (o *orderStore) Add(order *exchange.OrderDetail) error {
	o.m.Lock()
	defer o.m.Unlock()

	if o.exists(order) {
		return ErrOrdersAlreadyExists
	}

	orders := o.Orders[order.Exchange]
	orders = append(orders, *order)
	o.Orders[order.Exchange] = orders
	return nil
}

func (o *orderManager) Started() bool {
	return atomic.LoadInt32(&o.started) == 1
}

func (o *orderManager) Start() error {
	if atomic.AddInt32(&o.started, 1) != 1 {
		return errors.New("order manager already started")
	}

	log.Debugln("Order manager starting...")

	// test param
	o.cfg.CancelOrdersOnShutdown = true
	o.shutdown = make(chan struct{})
	o.orderStore.Orders = make(map[string][]exchange.OrderDetail)
	go o.run()
	return nil
}
func (o *orderManager) Stop() error {
	if atomic.AddInt32(&o.stopped, 1) != 1 {
		return errors.New("order manager is already stopped")
	}

	log.Debugln("Order manager shutting down...")
	close(o.shutdown)
	return nil
}

func (o *orderManager) gracefulShutdown() {
	if o.cfg.CancelOrdersOnShutdown {
		log.Debug("Order manager: Cancelling any open orders...")
		orders := o.orderStore.Get()
		if orders == nil {
			return
		}

		for k, v := range orders {
			log.Debugf("Order manager: Cancelling order(s) for exchange %s.", k)
			for y := range v {
				log.Debugf("order manager: Cancelling order ID %v [%v]",
					v[y].ID, v[y])
				err := o.Cancel(k, &exchange.OrderCancellation{
					OrderID: v[y].ID,
				})
				if err != nil {
					msg := fmt.Sprintf("Order manager: Exchange %s unable to cancel order ID=%v. Err: %s",
						k, v[y].ID, err)
					log.Debugln(msg)
					Bot.CommsRelayer.PushEvent(base.Event{
						Type:    "order",
						Message: msg,
					})
					continue
				}

				msg := fmt.Sprintf("Order manager: Exchange %s order ID=%v cancelled.",
					k, v[y].ID)
				log.Debugln(msg)
				Bot.CommsRelayer.PushEvent(base.Event{
					Type:    "order",
					Message: msg,
				})
			}
		}
	}
}

func (o *orderManager) run() {
	log.Debugln("Order manager started.")
	tick := time.NewTicker(OrderManagerDelay)
	Bot.ServicesWG.Add(1)
	defer func() {
		log.Debugf("Order manager shutdown.")
		tick.Stop()
		Bot.ServicesWG.Done()
	}()

	for {
		select {
		case <-o.shutdown:
			o.gracefulShutdown()
			return
		case <-tick.C:
			o.processOrders()
		}
	}
}

func (o *orderManager) CancelAllOrders() {}

func (o *orderManager) Cancel(exchName string, order *exchange.OrderCancellation) error {
	if exchName == "" {
		return errors.New("order exchange name is empty")
	}

	if order == nil {
		return errors.New("order cancel param is nil")
	}

	if order.OrderID == "" {
		return errors.New("order id is empty")
	}

	exch := GetExchangeByName(exchName)
	if exch == nil {
		return errors.New("unable to get exchange by name")
	}

	if order.AssetType.String() != "" && !exch.GetAssetTypes().Contains(order.AssetType) {
		return errors.New("order asset type not supported by exchange")
	}

	return exch.CancelOrder(order)
}

func (o *orderManager) Submit(exchName string, order *exchange.OrderSubmission) (*orderSubmitResponse, error) {
	if exchName == "" {
		return nil, errors.New("order exchange name must be specified")
	}

	if order == nil {
		return nil, exchange.ErrOrderSubmissionIsNil
	}

	if err := order.Validate(); err != nil {
		return nil, err
	}

	if o.cfg.EnforceLimitConfig {
		if !o.cfg.AllowMarketOrders && order.OrderType == exchange.MarketOrderType {
			return nil, errors.New("order market type is not allowed")
		}

		if o.cfg.LimitAmount > 0 && order.Amount > o.cfg.LimitAmount {
			return nil, errors.New("order limit exceeds allowed limit")
		}

		if len(o.cfg.AllowedExchanges) > 0 &&
			!common.StringDataCompareInsensitive(o.cfg.AllowedExchanges, exchName) {
			return nil, errors.New("order exchange not found in allowed list")
		}

		if len(o.cfg.AllowedPairs) > 0 && !o.cfg.AllowedPairs.Contains(order.Pair, true) {
			return nil, errors.New("order pair not found in allowed list")
		}
	}

	exch := GetExchangeByName(exchName)
	if exch == nil {
		return nil, errors.New("unable to get exchange by name")
	}

	id, err := common.GetV4UUID()
	if err != nil {
		log.Warnf("Order manager: Unable to generate UUID. Err: %s", err)
	}

	result, err := exch.SubmitOrder(order)
	if err != nil {
		return nil, err
	}

	if result.IsOrderPlaced {
		return nil, errors.New("order unable to be placed")
	}

	msg := fmt.Sprintf("Order manager: Exchange %s submitted order ID=%v [Ours: %v] pair=%v price=%v amount=%v side=%v type=%v.",
		exchName, result.OrderID, id.String(), order.Pair, order.Price, order.Amount, order.OrderSide, order.OrderType)
	log.Debugln(msg)
	Bot.CommsRelayer.PushEvent(base.Event{
		Type:    "order",
		Message: msg,
	})

	return &orderSubmitResponse{
		SubmitOrderResponse: exchange.SubmitOrderResponse{
			OrderID: result.OrderID,
		},
		OurOrderID: id.String(),
	}, nil
}

func (o *orderManager) processOrders() {
	authExchanges := GetAuthAPISupportedExchanges()
	for x := range authExchanges {
		log.Debugf("Order manager: Procesing orders for exchange %v.", authExchanges[x])
		exch := GetExchangeByName(authExchanges[x])
		req := exchange.GetOrdersRequest{
			OrderSide: exchange.AnyOrderSide,
			OrderType: exchange.AnyOrderType,
		}
		result, err := exch.GetActiveOrders(&req)
		if err != nil {
			log.Debugf("Order manager: Unable to get active orders: %s", err)
			continue
		}

		for x := range result {
			order := &result[x]
			result := o.orderStore.Add(order)
			if result != ErrOrdersAlreadyExists {
				msg := fmt.Sprintf("Order manager: Exchange %s added order ID=%v pair=%v price=%v amount=%v side=%v type=%v.",
					order.Exchange, order.ID, order.CurrencyPair, order.Price, order.Amount, order.OrderSide, order.OrderType)
				log.Debug(msg)
				Bot.CommsRelayer.PushEvent(base.Event{
					Type:    "order",
					Message: msg,
				})
				continue
			}
		}
	}
}
