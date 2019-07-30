package ob

import (
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/thrasher-/gocryptotrader/currency"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	log "github.com/thrasher-/gocryptotrader/logger"
)

const wsOrderbookBufferLimit = 5

// Update updates a local cache using bid targets and ask targets then updates
// main cache in orderbook.go
// Volume == 0; deletion at price target
// Price target not found; append of price target
// Price target found; amend volume of price target
func (w *WebsocketOrderbookLocal) Update(orderbookUpdate *BufferUpdate) error {
	if (orderbookUpdate.Bids == nil && orderbookUpdate.Asks == nil) ||
		(len(orderbookUpdate.Bids) == 0 && len(orderbookUpdate.Asks) == 0) {
		return errors.New("cannot have bids and ask targets both nil")
	}
	if _, ok := w.orderbook[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType]; !ok {
		return fmt.Errorf("orderbook.Base could not be found for Exchange %s CurrencyPair: %s AssetType: %s",
			orderbookUpdate.ExchangeName,
			orderbookUpdate.CurrencyPair.String(),
			orderbookUpdate.AssetType)
	}

	if w.orderbookBuffer == nil {
		w.orderbookBuffer = make(map[currency.Pair]map[string][]BufferUpdate)
	}
	if w.orderbookBuffer[orderbookUpdate.CurrencyPair] == nil {
		w.orderbookBuffer[orderbookUpdate.CurrencyPair] = make(map[string][]BufferUpdate)
	}
	if len(w.orderbookBuffer[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType]) <= wsOrderbookBufferLimit {
		w.orderbookBuffer[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType] = append(w.orderbookBuffer[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType], *orderbookUpdate)
		if len(w.orderbookBuffer[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType]) < wsOrderbookBufferLimit {
			return nil
		}
	}
	// sort by last updated to ensure each update is in order
	if orderbookUpdate.OrderByIDs {
		sort.Slice(w.orderbookBuffer[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType], func(i, j int) bool {
			return w.orderbookBuffer[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType][i].UpdateID < w.orderbookBuffer[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType][j].UpdateID
		})
	} else {
		sort.Slice(w.orderbookBuffer[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType], func(i, j int) bool {
			return w.orderbookBuffer[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType][i].Updated.Before(w.orderbookBuffer[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType][j].Updated)
		})
	}
	for i := 0; i < len(w.orderbookBuffer[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType]); i++ {
		if w.orderbookBuffer[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType][i].UseUpdateIDs {
			log.Debug(w.orderbookBuffer[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType][i].Asks[0])
			w.DoTheThing(&w.orderbookBuffer[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType][i])
		} else {
			var wg sync.WaitGroup
			wg.Add(2)
			w.updateAsksByPrice(&w.orderbookBuffer[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType][i], &wg)
			w.updateBidsByPrice(&w.orderbookBuffer[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType][i], &wg)
			wg.Wait()
		}
	}
	err := w.orderbook[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType].Process()
	if err != nil {
		return err
	}
	// Reset the buffer
	w.orderbookBuffer[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType] = []BufferUpdate{}
	return nil
}

func (w *WebsocketOrderbookLocal) updateAsksByPrice(base *BufferUpdate, wg *sync.WaitGroup) {
	for j := 0; j < len(base.Asks); j++ {
		found := false
		for k := 0; k < len(w.orderbook[base.CurrencyPair][base.AssetType].Asks); k++ {
			if w.orderbook[base.CurrencyPair][base.AssetType].Asks[k].Price == base.Asks[j].Price {
				found = true
				if base.Asks[j].Amount == 0 {
					w.orderbook[base.CurrencyPair][base.AssetType].Asks = append(w.orderbook[base.CurrencyPair][base.AssetType].Asks[:k],
						w.orderbook[base.CurrencyPair][base.AssetType].Asks[k+1:]...)
					break
				}
				w.orderbook[base.CurrencyPair][base.AssetType].Asks[k].Amount = base.Asks[j].Amount
				break
			}
		}
		if !found {
			w.orderbook[base.CurrencyPair][base.AssetType].Asks = append(w.orderbook[base.CurrencyPair][base.AssetType].Asks, base.Asks[j])
		}
	}
	wg.Done()
}

func (w *WebsocketOrderbookLocal) updateBidsByPrice(base *BufferUpdate, wg *sync.WaitGroup) {
	for j := 0; j < len(base.Bids); j++ {
		found := false
		for k := 0; k < len(w.orderbook[base.CurrencyPair][base.AssetType].Bids); k++ {
			if w.orderbook[base.CurrencyPair][base.AssetType].Bids[k].Price == base.Bids[j].Price {

				found = true
				if base.Bids[j].Amount == 0 {
					w.orderbook[base.CurrencyPair][base.AssetType].Bids = append(w.orderbook[base.CurrencyPair][base.AssetType].Bids[:k],
						w.orderbook[base.CurrencyPair][base.AssetType].Bids[k+1:]...)
					break
				}
				w.orderbook[base.CurrencyPair][base.AssetType].Bids[k].Amount = base.Bids[j].Amount
				break
			}
		}
		if !found {
			w.orderbook[base.CurrencyPair][base.AssetType].Bids = append(w.orderbook[base.CurrencyPair][base.AssetType].Bids, base.Bids[j])
		}
	}
	wg.Done()
}

// LoadSnapshot loads initial snapshot of orderbook data, overite allows full
// orderbook to be completely rewritten because the exchange is a doing a full
// update not an incremental one
func (w *WebsocketOrderbookLocal) LoadSnapshot(newOrderbook *orderbook.Base, exchName string, overwrite bool) error {
	if len(newOrderbook.Asks) == 0 || len(newOrderbook.Bids) == 0 {
		return errors.New("snapshot ask and bids are nil")
	}
	w.m.Lock()
	defer w.m.Unlock()
	if w.orderbook == nil {
		w.orderbook = make(map[currency.Pair]map[string]*orderbook.Base)
	}
	if w.orderbook[newOrderbook.Pair] == nil {
		w.orderbook[newOrderbook.Pair] = make(map[string]*orderbook.Base)
	}
	if w.orderbook[newOrderbook.Pair][newOrderbook.AssetType] != nil &&
		(len(w.orderbook[newOrderbook.Pair][newOrderbook.AssetType].Asks) > 0 ||
			len(w.orderbook[newOrderbook.Pair][newOrderbook.AssetType].Bids) > 0) {
		if overwrite {
			w.orderbook[newOrderbook.Pair][newOrderbook.AssetType] = newOrderbook
			return newOrderbook.Process()
		}
		return errors.New("snapshot instance already found")
	}
	w.orderbook[newOrderbook.Pair][newOrderbook.AssetType] = newOrderbook
	return newOrderbook.Process()
}

// DoTheThing studies the thing,
// understands its true purpose,
// reflects on how it impacts the world around us.
//
// Then fucking does it
func (w *WebsocketOrderbookLocal) DoTheThing(orderbookUpdate *BufferUpdate) {
	switch orderbookUpdate.Action {
	case "update":
		for _, target := range orderbookUpdate.Bids {
			for i := range w.orderbook[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType].Bids {
				if w.orderbook[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType].Bids[i].ID == target.ID {
					w.orderbook[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType].Bids[i].Amount = target.Amount
					break
				}
			}
		}

		for _, target := range orderbookUpdate.Asks {
			for i := range w.orderbook[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType].Asks {
				if w.orderbook[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType].Asks[i].ID == target.ID {
					w.orderbook[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType].Asks[i].Amount = target.Amount
					break
				}
			}
		}

	case "delete":
		for _, target := range orderbookUpdate.Bids {
			for i := range w.orderbook[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType].Bids {
				if w.orderbook[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType].Bids[i].ID == target.ID {
					w.orderbook[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType].Bids = append(w.orderbook[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType].Bids[:i],
						w.orderbook[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType].Bids[i+1:]...)
					i--
					break
				}
			}
		}

		for _, target := range orderbookUpdate.Asks {
			for i := range w.orderbook[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType].Asks {
				if w.orderbook[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType].Asks[i].ID == target.ID {
					w.orderbook[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType].Asks = append(w.orderbook[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType].Asks[:i],
						w.orderbook[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType].Asks[i+1:]...)
					i--
					break
				}
			}
		}

	case "insert":
		w.orderbook[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType].Bids = append(w.orderbook[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType].Bids, orderbookUpdate.Bids...)
		w.orderbook[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType].Asks = append(w.orderbook[orderbookUpdate.CurrencyPair][orderbookUpdate.AssetType].Asks, orderbookUpdate.Asks...)
	}
}

// FlushCache flushes w.ob data to be garbage collected and refreshed when a
// connection is lost and reconnected
func (w *WebsocketOrderbookLocal) FlushCache() {
	w.m.Lock()
	w.orderbook = nil
	w.orderbookBuffer = nil
	w.m.Unlock()
}
