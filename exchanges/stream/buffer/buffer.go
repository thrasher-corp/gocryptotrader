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
	return w.updateByPrice(o, u)
}

func (w *Orderbook) updateByPrice(o *orderbook.Base, u *Update) error {
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

			if o.Asks[k].Price > u.Asks[j].Price && u.Asks[j].Amount > 0 {
				insertItem(u.Asks[j], &o.Asks, k)
				continue askUpdates
			}
		}
	}

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

			if o.Bids[k].Price < u.Bids[j].Price && u.Bids[j].Amount > 0 {
				insertItem(u.Bids[j], &o.Bids, k)
				continue bidUpdates
			}
		}
	}
	return nil
}

// updateByIDAndAction will receive an action to execute against the orderbook
// it will then match by IDs instead of price to perform the action
func (w *Orderbook) updateByIDAndAction(o *orderbook.Base, u *Update) (err error) {
	fmt.Printf("%+v\n", u)
	switch u.Action {
	case Amend:
		fmt.Println("amend")
		err = applyUpdates(&u.Bids, &o.Bids)
		if err != nil {
			return err
		}
		err = applyUpdates(&u.Asks, &o.Asks)
		if err != nil {
			return err
		}
	case Delete:
		fmt.Println("delete")
		err = deleteUpdates(&u.Bids, &o.Bids)
		if err != nil {
			return err
		}
		err = deleteUpdates(&u.Asks, &o.Asks)
		if err != nil {
			return err
		}
	case Insert:
		fmt.Println("insert")
		insertUpdatesBid(&u.Bids, &o.Bids)
		insertUpdatesAsk(&u.Asks, &o.Asks)
	case UpdateInsert:
		fmt.Println("update insert")
	updateBids:
		for x := range u.Bids {
			for y := range o.Bids {
				if o.Bids[y].ID == u.Bids[x].ID {
					o.Bids[y].Amount = u.Bids[x].Amount
					continue updateBids
				}

				if o.Bids[y].Price > u.Bids[x].Price {
					insertItem(u.Bids[x], &o.Bids, y)
					continue updateBids
				}
			}
			return errors.New("good ness")
		}

	updateAsks:
		for x := range u.Asks {
			for y := range o.Asks {
				if o.Asks[y].ID == u.Asks[x].ID {
					o.Asks[y].Amount = u.Asks[x].Amount
					continue updateAsks
				}

				if o.Asks[y].Price < u.Asks[x].Price {
					insertItem(u.Asks[x], &o.Asks, y)
					continue updateAsks
				}
			}
			return errors.New("good ness")
		}

	default:
		return fmt.Errorf("invalid action [%s]", u.Action)
	}
	return nil
}

func applyUpdates(u, b *[]orderbook.Item) error {
updates:
	for x := range *u {
		for y := range *b {
			if ([]orderbook.Item)(*b)[y].ID == ([]orderbook.Item)(*u)[x].ID {
				([]orderbook.Item)(*b)[y].Amount = ([]orderbook.Item)(*u)[x].Amount
				continue updates
			}
		}
		return fmt.Errorf("update cannot be applied id: %d not found",
			([]orderbook.Item)(*u)[x].ID)
	}
	return nil
}

func deleteUpdates(u, b *[]orderbook.Item) error {
updates:
	for x := range *u {
		for y := range *b {
			if ([]orderbook.Item)(*b)[y].ID == ([]orderbook.Item)(*u)[x].ID {
				*b = append(([]orderbook.Item)(*b)[:y], ([]orderbook.Item)(*b)[y+1:]...)
				continue updates
			}
		}
		return fmt.Errorf("update cannot be deleted id: %d not found",
			([]orderbook.Item)(*u)[x].ID)
	}
	return nil
}

// insertUpdatesBid inserts on correctly aligned book at price level
func insertUpdatesBid(u, b *[]orderbook.Item) {
	for x := range *u {
		fmt.Println("PEW BIDS")
		for y := range *b {
			if ([]orderbook.Item)(*u)[x].ID == ([]orderbook.Item)(*b)[y].ID {
				return
			}
			if ([]orderbook.Item)(*u)[x].Price > ([]orderbook.Item)(*b)[y].Price {
				if ([]orderbook.Item)(*u)[x].ID == ([]orderbook.Item)(*b)[y+1].ID {
					return
				}
				insertItem(([]orderbook.Item)(*u)[x], b, y)
			}
		}
	}
}

// insertUpdatesBid inserts on correctly aligned book at price level
func insertUpdatesAsk(u, b *[]orderbook.Item) {
	for x := range *u {
		fmt.Println("PEW ASKS")
		for y := range *b {
			if ([]orderbook.Item)(*u)[x].ID == ([]orderbook.Item)(*b)[y].ID {
				return
			}

		}
		for y := range *b {
			if ([]orderbook.Item)(*u)[x].Price < ([]orderbook.Item)(*b)[y].Price {
				insertItem(([]orderbook.Item)(*u)[x], b, y)
			}
		}
	}
}

func insertItem(update orderbook.Item, book *[]orderbook.Item, target int) {
	*book = append(*book, orderbook.Item{})
	copy(([]orderbook.Item)(*book)[target+1:], ([]orderbook.Item)(*book)[target:])
	([]orderbook.Item)(*book)[target] = update
}

// LoadSnapshot loads initial snapshot of ob data from websocket
func (w *Orderbook) LoadSnapshot(newOrderbook *orderbook.Base) error {
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
	m3.Bids = newOrderbook.Bids
	m3.Asks = newOrderbook.Asks

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
	book, ok := w.ob[p.Base][p.Quote][a]
	if !ok {
		return fmt.Errorf("orderbook not associated with pair: [%s] and asset [%s]", p, a)
	}
	book.Bids = nil
	book.Asks = nil
	return nil
}
