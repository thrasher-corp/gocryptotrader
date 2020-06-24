package buffer

import (
	"errors"
	"fmt"
	"sort"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
)

// Setup sets private variables
func (w *Orderbook) Setup(obBufferLimit int, bufferEnabled, sortBuffer, sortBufferByUpdateIDs, updateEntriesByID bool, exchangeName string, dataHandler chan interface{}) {
	w.obBufferLimit = obBufferLimit
	w.bufferEnabled = bufferEnabled
	w.sortBuffer = sortBuffer
	w.sortBufferByUpdateIDs = sortBufferByUpdateIDs
	w.updateEntriesByID = updateEntriesByID
	w.exchangeName = exchangeName
	w.dataHandler = dataHandler
}

// Update updates a local buffer using bid targets and ask targets then updates
// main orderbook
// Volume == 0; deletion at price target
// Price target not found; append of price target
// Price target found; amend volume of price target
func (w *Orderbook) Update(u *Update) error {
	if (u.Bids == nil && u.Asks == nil) || (len(u.Bids) == 0 && len(u.Asks) == 0) {
		return fmt.Errorf("%v cannot have bids and ask targets both nil",
			w.exchangeName)
	}
	w.m.Lock()
	defer w.m.Unlock()
	obLookup, ok := w.ob[u.Pair][u.Asset]
	if !ok {
		return fmt.Errorf("ob.Base could not be found for Exchange %s CurrencyPair: %s AssetType: %s",
			w.exchangeName,
			u.Pair,
			u.Asset)
	}

	if w.bufferEnabled {
		overBufferLimit := w.processBufferUpdate(obLookup, u)
		if !overBufferLimit {
			return nil
		}
	} else {
		w.processObUpdate(obLookup, u)
	}
	err := obLookup.Process()
	if err != nil {
		return err
	}

	if w.bufferEnabled {
		// Reset the buffer
		w.buffer[u.Pair][u.Asset] = nil
	}

	// Process in data handler
	w.dataHandler <- obLookup
	return nil
}

func (w *Orderbook) processBufferUpdate(o *orderbook.Base, u *Update) bool {
	if w.buffer == nil {
		w.buffer = make(map[currency.Pair]map[asset.Item][]*Update)
	}
	if w.buffer[u.Pair] == nil {
		w.buffer[u.Pair] = make(map[asset.Item][]*Update)
	}
	bufferLookup := w.buffer[u.Pair][u.Asset]
	if len(bufferLookup) <= w.obBufferLimit {
		bufferLookup = append(bufferLookup, u)
		if len(bufferLookup) < w.obBufferLimit {
			w.buffer[u.Pair][u.Asset] = bufferLookup
			return false
		}
	}
	if w.sortBuffer {
		// sort by last updated to ensure each update is in order
		if w.sortBufferByUpdateIDs {
			sort.Slice(bufferLookup, func(i, j int) bool {
				return bufferLookup[i].UpdateID < bufferLookup[j].UpdateID
			})
		} else {
			sort.Slice(bufferLookup, func(i, j int) bool {
				return bufferLookup[i].UpdateTime.Before(bufferLookup[j].UpdateTime)
			})
		}
	}
	for i := range bufferLookup {
		w.processObUpdate(o, bufferLookup[i])
	}
	w.buffer[u.Pair][u.Asset] = bufferLookup
	return true
}

func (w *Orderbook) processObUpdate(o *orderbook.Base, u *Update) {
	o.LastUpdateID = u.UpdateID

	if w.updateEntriesByID {
		w.updateByIDAndAction(o, u)
	} else {
		w.updateAsksByPrice(o, u)
		w.updateBidsByPrice(o, u)
	}
}

func (w *Orderbook) updateAsksByPrice(o *orderbook.Base, u *Update) {
updates:
	for j := range u.Asks {
		for k := range o.Asks {
			if o.Asks[k].Price == u.Asks[j].Price {
				if u.Asks[j].Amount <= 0 {
					o.Asks = append(o.Asks[:k], o.Asks[k+1:]...)
					continue updates
				}
				o.Asks[k].Amount = u.Asks[j].Amount
				continue updates
			}
		}
		if u.Asks[j].Amount == 0 {
			continue
		}
		o.Asks = append(o.Asks, u.Asks[j])
	}
	sort.Slice(o.Asks, func(i, j int) bool {
		return o.Asks[i].Price < o.Asks[j].Price
	})
}

func (w *Orderbook) updateBidsByPrice(o *orderbook.Base, u *Update) {
updates:
	for j := range u.Bids {
		for k := range o.Bids {
			if o.Bids[k].Price == u.Bids[j].Price {
				if u.Bids[j].Amount <= 0 {
					o.Bids = append(o.Bids[:k], o.Bids[k+1:]...)
					continue updates
				}
				o.Bids[k].Amount = u.Bids[j].Amount
				continue updates
			}
		}
		if u.Bids[j].Amount == 0 {
			continue
		}
		o.Bids = append(o.Bids, u.Bids[j])
	}
	sort.Slice(o.Bids, func(i, j int) bool {
		return o.Bids[i].Price > o.Bids[j].Price
	})
}

// updateByIDAndAction will receive an action to execute against the orderbook
// it will then match by IDs instead of price to perform the action
func (w *Orderbook) updateByIDAndAction(o *orderbook.Base, u *Update) {
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
		sort.Slice(o.Bids, func(i, j int) bool {
			return o.Bids[i].Price > o.Bids[j].Price
		})

		o.Asks = append(o.Asks, u.Asks...)
		sort.Slice(o.Asks, func(i, j int) bool {
			return o.Asks[i].Price < o.Asks[j].Price
		})

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
	}
}

// LoadSnapshot loads initial snapshot of ob data, overwrite allows full
// ob to be completely rewritten because the exchange is a doing a full
// update not an incremental one
func (w *Orderbook) LoadSnapshot(newOrderbook *orderbook.Base) error {
	if len(newOrderbook.Asks) == 0 || len(newOrderbook.Bids) == 0 {
		return fmt.Errorf("%v snapshot ask and bids are nil", w.exchangeName)
	}

	if newOrderbook.Pair.IsEmpty() {
		return errors.New("websocket orderbook pair unset")
	}

	if newOrderbook.AssetType.String() == "" {
		return errors.New("websocket orderbook asset type unset")
	}

	if newOrderbook.ExchangeName == "" {
		return errors.New("websocket orderbook exchange name unset")
	}

	w.m.Lock()
	defer w.m.Unlock()
	if w.ob == nil {
		w.ob = make(map[currency.Pair]map[asset.Item]*orderbook.Base)
	}
	if w.ob[newOrderbook.Pair] == nil {
		w.ob[newOrderbook.Pair] = make(map[asset.Item]*orderbook.Base)
	}

	w.ob[newOrderbook.Pair][newOrderbook.AssetType] = newOrderbook
	err := newOrderbook.Process()
	if err != nil {
		return err
	}

	w.dataHandler <- newOrderbook
	return nil
}

// GetOrderbook use sparingly. Modifying anything here will ruin hash
// calculation and cause problems
func (w *Orderbook) GetOrderbook(p currency.Pair, a asset.Item) *orderbook.Base {
	w.m.Lock()
	ob := w.ob[p][a]
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
