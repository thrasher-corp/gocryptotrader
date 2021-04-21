package ordermanager

import (
	"errors"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/subsystems/exchangemanager"
)

// get returns all orders for all exchanges
// should not be exported as it can have large impact if used improperly
func (s *store) get() map[string][]*order.Detail {
	s.m.RLock()
	orders := s.Orders
	s.m.RUnlock()
	return orders
}

// getByExchangeAndID returns a specific order by exchange and id
func (s *store) getByExchangeAndID(exchange, id string) (*order.Detail, error) {
	s.m.RLock()
	defer s.m.RUnlock()
	r, ok := s.Orders[strings.ToLower(exchange)]
	if !ok {
		return nil, exchangemanager.ErrExchangeNotFound
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
	s.m.RLock()
	defer s.m.RUnlock()
	r, ok := s.Orders[strings.ToLower(od.Exchange)]
	if !ok {
		return exchangemanager.ErrExchangeNotFound
	}
	for x := range r {
		if r[x].ID == od.ID {
			r[x] = od
			return nil
		}
	}

	return ErrOrderNotFound
}

func (s *store) upsert(od *order.Detail) error {
	lName := strings.ToLower(od.Exchange)
	exch := s.exchangeManager.GetExchangeByName(lName)
	if exch == nil {
		return exchangemanager.ErrExchangeNotFound
	}
	s.m.Lock()
	defer s.m.Unlock()
	r, ok := s.Orders[lName]
	if !ok {
		s.Orders[lName] = []*order.Detail{od}
		return nil
	}
	for x := range r {
		if r[x].ID == od.ID {
			r[x] = od
			return nil
		}
	}
	s.Orders[lName] = append(s.Orders[lName], od)
	return nil
}

// getByExchange returns orders by exchange
func (s *store) getByExchange(exchange string) ([]*order.Detail, error) {
	s.m.RLock()
	defer s.m.RUnlock()
	r, ok := s.Orders[strings.ToLower(exchange)]
	if !ok {
		return nil, exchangemanager.ErrExchangeNotFound
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
		return errors.New("order store: Order is nil")
	}
	exch := s.exchangeManager.GetExchangeByName(det.Exchange)
	if exch == nil {
		return exchangemanager.ErrExchangeNotFound
	}
	if s.exists(det) {
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
	s.m.Lock()
	defer s.m.Unlock()
	orders := s.Orders[strings.ToLower(det.Exchange)]
	orders = append(orders, det)
	s.Orders[strings.ToLower(det.Exchange)] = orders

	return nil
}
