package engine

import (
	"errors"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	websocketSyncQueueSize = 4096
)

var errUnsupportedWebsocketSyncData = errors.New("unsupported websocket sync data")

// startWebsocketSyncWorkers launches workers that consume websocket sync updates.
func (m *SyncManager) startWebsocketSyncWorkers() {
	if m.wsUpdateQueue == nil {
		m.wsUpdateQueue = make(chan websocketSyncUpdate, websocketSyncQueueSize)
	}

	workerCount := m.config.NumWorkers
	if workerCount <= 0 {
		workerCount = 1
	}

	for range workerCount {
		m.wsSyncWG.Go(m.websocketSyncWorker)
	}
}

// websocketSyncWorker consumes queued websocket updates and routes each update for processing.
func (m *SyncManager) websocketSyncWorker() {
	for {
		select {
		case <-m.shutdown:
			return
		case update := <-m.wsUpdateQueue:
			if err := m.handleWebsocketSyncUpdate(update); err != nil {
				log.Errorln(log.SyncMgr, err)
			}
		}
	}
}

// EnqueueWebsocketUpdate queues websocket updates for asynchronous sync processing.
func (m *SyncManager) EnqueueWebsocketUpdate(exchangeName string, data any) error {
	if m == nil {
		return fmt.Errorf("exchange CurrencyPairSyncer %w", ErrNilSubsystem)
	}
	if m.wsUpdateQueue == nil {
		return nil
	}

	update := websocketSyncUpdate{
		exchangeName: exchangeName,
	}

	switch d := data.(type) {
	case *ticker.Price:
		if d == nil {
			return nil
		}
		p := *d
		update.ticker = &p
	case []ticker.Price:
		update.tickerBatch = make([]ticker.Price, len(d))
		copy(update.tickerBatch, d)
	case *orderbook.Depth:
		if d == nil {
			return nil
		}
		update.orderbook = d
	default:
		return fmt.Errorf("%w: %T", errUnsupportedWebsocketSyncData, data)
	}

	select {
	case m.wsUpdateQueue <- update:
		return nil
	default:
		// Keep websocket data handling responsive by dropping sync updates under back pressure.
		log.Warnf(log.SyncMgr, "%s websocket sync queue is full; dropping update", exchangeName)
		return nil
	}
}

func (m *SyncManager) handleWebsocketSyncUpdate(update websocketSyncUpdate) error {
	switch {
	case update.ticker != nil:
		return m.handleWebsocketTicker(update.exchangeName, update.ticker)
	case len(update.tickerBatch) > 0:
		return m.handleWebsocketTickerBatch(update.exchangeName, update.tickerBatch)
	case update.orderbook != nil:
		return m.handleWebsocketOrderbook(update.exchangeName, update.orderbook)
	default:
		return nil
	}
}

// handleWebsocketTicker processes a single websocket ticker update when the pair is tracked.
func (m *SyncManager) handleWebsocketTicker(exchangeName string, t *ticker.Price) error {
	if !m.isTrackedPair(exchangeName, t.Pair, t.AssetType) {
		return nil
	}

	err := m.WebsocketUpdate(exchangeName, t.Pair, t.AssetType, SyncItemTicker, nil)
	m.PrintTickerSummary(t, "websocket", err)
	return err
}

// handleWebsocketTickerBatch processes websocket ticker batch updates for tracked pairs.
func (m *SyncManager) handleWebsocketTickerBatch(exchangeName string, batch []ticker.Price) error {
	for i := range batch {
		if !m.isTrackedPair(exchangeName, batch[i].Pair, batch[i].AssetType) {
			continue
		}

		err := m.WebsocketUpdate(exchangeName, batch[i].Pair, batch[i].AssetType, SyncItemTicker, nil)
		m.PrintTickerSummary(&batch[i], "websocket", err)
		if err != nil {
			return err
		}
	}
	return nil
}

// handleWebsocketOrderbook processes a websocket orderbook update when the pair is tracked.
func (m *SyncManager) handleWebsocketOrderbook(exchangeName string, d *orderbook.Depth) error {
	base, err := d.Retrieve()
	if err != nil {
		return err
	}

	if !m.isTrackedPair(exchangeName, base.Pair, base.Asset) {
		return nil
	}

	if err := m.WebsocketUpdate(exchangeName, base.Pair, base.Asset, SyncItemOrderbook, nil); err != nil {
		return err
	}
	m.PrintOrderbookSummary(base, "websocket", nil)
	return nil
}

// isTrackedPair reports whether the sync manager currently tracks an exchange/asset/pair key.
func (m *SyncManager) isTrackedPair(exchangeName string, pair currency.Pair, a asset.Item) bool {
	return m.get(key.NewExchangeAssetPair(exchangeName, a, pair)) != nil
}
