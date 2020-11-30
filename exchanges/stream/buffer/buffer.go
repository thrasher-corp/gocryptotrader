package buffer

import (
	"errors"
	"fmt"
	"sort"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
)

const packageError = "websocket orderbook buffer error: %w"

var (
	errUnsetExchangeName            = errors.New("exchange name unset")
	errUnsetDataHandler             = errors.New("datahandler unset")
	errIssueBufferEnabledButNoLimit = errors.New("buffer enabled but no limit set")
	errUpdateIsNil                  = errors.New("update is nil")
	errUpdateNoTargets              = errors.New("update bid/ask targets cannot be nil")
)

// Setup sets private variables
func (w *Orderbook) Setup(obBufferLimit int,
	bufferEnabled,
	sortBuffer,
	sortBufferByUpdateIDs,
	updateEntriesByID bool, exchangeName string, dataHandler chan interface{}) error {
	if exchangeName == "" {
		return fmt.Errorf(packageError, errUnsetExchangeName)
	}
	if dataHandler == nil {
		return fmt.Errorf(packageError, errUnsetDataHandler)
	}
	if bufferEnabled && obBufferLimit < 1 {
		return fmt.Errorf(packageError, errIssueBufferEnabledButNoLimit)
	}
	w.obBufferLimit = obBufferLimit
	w.bufferEnabled = bufferEnabled
	w.sortBuffer = sortBuffer
	w.sortBufferByUpdateIDs = sortBufferByUpdateIDs
	w.updateEntriesByID = updateEntriesByID
	w.exchangeName = exchangeName
	w.dataHandler = dataHandler
	w.ob = make(map[currency.Code]map[currency.Code]map[asset.Item]*orderbook.Base)
	w.buffer = make(map[currency.Code]map[currency.Code]map[asset.Item]*[]Update)
	return nil
}

// validate validates update against setup values
func (w *Orderbook) validate(u *Update) error {
	if u == nil {
		return fmt.Errorf(packageError, errUpdateIsNil)
	}
	if len(u.Bids) == 0 && len(u.Asks) == 0 {
		return fmt.Errorf(packageError, errUpdateNoTargets)
	}
	return nil
}

// Update updates a local buffer using bid targets and ask targets then updates
// main orderbook
// Volume == 0; deletion at price target
// Price target not found; append of price target
// Price target found; amend volume of price target
func (w *Orderbook) Update(u *Update) error {
	if err := w.validate(u); err != nil {
		return err
	}
	w.m.Lock()
	defer w.m.Unlock()
	obLookup, ok := w.ob[u.Pair.Base][u.Pair.Quote][u.Asset]
	if !ok {
		return fmt.Errorf("ob.Base could not be found for Exchange %s CurrencyPair: %s AssetType: %s",
			w.exchangeName,
			u.Pair,
			u.Asset)
	}

	if w.bufferEnabled {
		processed, err := w.processBufferUpdate(obLookup, u)
		if err != nil {
			return err
		}

		if !processed {
			return nil
		}
	} else {
		err := w.processObUpdate(obLookup, u)
		if err != nil {
			return err
		}
	}

	err := obLookup.Process()
	if err != nil {
		return err
	}

	// Process in data handler
	w.dataHandler <- obLookup
	return nil
}

func (w *Orderbook) processBufferUpdate(o *orderbook.Base, u *Update) (bool, error) {
	m1, ok := w.buffer[u.Pair.Base]
	if !ok {
		m1 = make(map[currency.Code]map[asset.Item]*[]Update)
		w.buffer[u.Pair.Base] = m1
	}
	m2, ok := m1[u.Pair.Quote]
	if !ok {
		m2 = make(map[asset.Item]*[]Update)
		w.buffer[u.Pair.Base][u.Pair.Quote] = m2
	}

	buffer, ok := m2[u.Asset]
	if !ok {
		buffer = new([]Update)
		m2[u.Asset] = buffer
	}

	if len(*buffer)+1 < w.obBufferLimit {
		*buffer = append(*buffer, *u)
		return false, nil
	}

	tmp := append(*buffer, *u)
	if w.sortBuffer {
		// sort by last updated to ensure each update is in order
		if w.sortBufferByUpdateIDs {
			sort.Slice(tmp, func(i, j int) bool {
				return tmp[i].UpdateID < tmp[j].UpdateID
			})
		} else {
			sort.Slice(tmp, func(i, j int) bool {
				return tmp[i].UpdateTime.Before(tmp[j].UpdateTime)
			})
		}
	}
	for i := range tmp {
		err := w.processObUpdate(o, &tmp[i])
		if err != nil {
			return false, err
		}
	}
	// clear buffer of old updates
	*buffer = nil
	return true, nil
}

func (w *Orderbook) processObUpdate(o *orderbook.Base, u *Update) error {
	o.LastUpdateID = u.UpdateID
	if w.updateEntriesByID {
		return w.updateByIDAndAction(o, u)
	}
	w.updateByPrice(o, u)
	return nil
}

func (w *Orderbook) updateByPrice(o *orderbook.Base, u *Update) {
askUpdates:
	for j := range u.Asks {
		for k := range o.Asks {
			if o.Asks[k].Price == u.Asks[j].Price {
				if u.Asks[j].Amount <= 0 {
					o.Asks = append(o.Asks[:k], o.Asks[k+1:]...)
					continue askUpdates
				}
				o.Asks[k].Amount = u.Asks[j].Amount
				continue askUpdates
			}
		}
		if u.Asks[j].Amount == 0 {
			continue
		}
		o.Asks = append(o.Asks, u.Asks[j])
	}
	_ = orderbook.SortAsks(o.Asks)

bidUpdates:
	for j := range u.Bids {
		for k := range o.Bids {
			if o.Bids[k].Price == u.Bids[j].Price {
				if u.Bids[j].Amount <= 0 {
					o.Bids = append(o.Bids[:k], o.Bids[k+1:]...)
					continue bidUpdates
				}
				o.Bids[k].Amount = u.Bids[j].Amount
				continue bidUpdates
			}
		}
		if u.Bids[j].Amount == 0 {
			continue
		}
		o.Bids = append(o.Bids, u.Bids[j])
	}
	_ = orderbook.SortBids(o.Bids)
}

// updateByIDAndAction will receive an action to execute against the orderbook
// it will then match by IDs instead of price to perform the action
func (w *Orderbook) updateByIDAndAction(o *orderbook.Base, u *Update) error {
	switch u.Action {
	case "update":
		for x := range u.Bids {
			for y := range o.Bids {
				if o.Bids[y].ID == u.Bids[x].ID {
					o.Bids[y].Amount = u.Bids[x].Amount
					break
				}
			}
		}
		for x := range u.Asks {
			for y := range o.Asks {
				if o.Asks[y].ID == u.Asks[x].ID {
					o.Asks[y].Amount = u.Asks[x].Amount
					break
				}
			}
		}
	case "delete":
		for x := range u.Bids {
			for y := 0; y < len(o.Bids); y++ {
				if o.Bids[y].ID == u.Bids[x].ID {
					o.Bids = append(o.Bids[:y], o.Bids[y+1:]...)
					break
				}
			}
		}
		for x := range u.Asks {
			for y := 0; y < len(o.Asks); y++ {
				if o.Asks[y].ID == u.Asks[x].ID {
					o.Asks = append(o.Asks[:y], o.Asks[y+1:]...)
					break
				}
			}
		}
	case "insert":
		o.Bids = append(o.Bids, u.Bids...)
		_ = orderbook.SortBids(o.Bids)

		o.Asks = append(o.Asks, u.Asks...)
		_ = orderbook.SortAsks(o.Asks)

	case "update/insert":
	updateBids:
		for x := range u.Bids {
			for y := range o.Bids {
				if o.Bids[y].ID == u.Bids[x].ID {
					o.Bids[y].Amount = u.Bids[x].Amount
					continue updateBids
				}
			}
			o.Bids = append(o.Bids, u.Bids[x])
		}
		_ = orderbook.SortBids(o.Bids)

	updateAsks:
		for x := range u.Asks {
			for y := range o.Asks {
				if o.Asks[y].ID == u.Asks[x].ID {
					o.Asks[y].Amount = u.Asks[x].Amount
					continue updateAsks
				}
			}
			o.Asks = append(o.Asks, u.Asks[x])
		}
		_ = orderbook.SortAsks(o.Asks)

	default:
		return fmt.Errorf("invalid action [%s]", u.Action)
	}
	return nil
}

// LoadSnapshot loads initial snapshot of ob data from websocket
func (w *Orderbook) LoadSnapshot(newOrderbook *orderbook.Base) error {
	// segragate bid/ask slice so there is no potential reference in the
	// orderbook package
	bids := append(newOrderbook.Bids[:0:0], newOrderbook.Bids...)
	asks := append(newOrderbook.Asks[:0:0], newOrderbook.Asks...)

	w.m.Lock()
	defer w.m.Unlock()

	err := newOrderbook.Process()
	if err != nil {
		return err
	}

	m1, ok := w.ob[newOrderbook.Pair.Base]
	if !ok {
		m1 = make(map[currency.Code]map[asset.Item]*orderbook.Base)
		w.ob[newOrderbook.Pair.Base] = m1
	}
	m2, ok := m1[newOrderbook.Pair.Quote]
	if !ok {
		m2 = make(map[asset.Item]*orderbook.Base)
		m1[newOrderbook.Pair.Quote] = m2
	}
	m3, ok := m2[newOrderbook.AssetType]
	if !ok {
		m3 = &orderbook.Base{
			Pair:         newOrderbook.Pair,
			LastUpdated:  newOrderbook.LastUpdated,
			LastUpdateID: newOrderbook.LastUpdateID,
			AssetType:    newOrderbook.AssetType,
			ExchangeName: newOrderbook.ExchangeName,
		}
		m2[newOrderbook.AssetType] = m3
	}
	m3.Bids = bids
	m3.Asks = asks

	w.dataHandler <- newOrderbook
	return nil
}

// GetOrderbook use sparingly. Modifying anything here will ruin hash
// calculation and cause problems
func (w *Orderbook) GetOrderbook(p currency.Pair, a asset.Item) *orderbook.Base {
	w.m.Lock()
	ob := w.ob[p.Base][p.Quote][a]
	w.m.Unlock()
	return ob
}

// FlushBuffer flushes w.ob data to be garbage collected and refreshed when a
// connection is lost and reconnected
func (w *Orderbook) FlushBuffer() {
	w.m.Lock()
	w.ob = nil
	w.buffer = nil
	w.m.Unlock()
}

// FlushOrderbook flushes independent orderbook
func (w *Orderbook) FlushOrderbook(p currency.Pair, a asset.Item) error {
	w.m.Lock()
	defer w.m.Unlock()
	_, ok := w.ob[p.Base][p.Quote][a]
	if !ok {
		return fmt.Errorf("orderbook not associated with pair: [%s] and asset [%s]", p, a)
	}
	w.ob[p.Base][p.Quote][a] = nil
	return nil
}
