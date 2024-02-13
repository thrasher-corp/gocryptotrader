package binance

import (
	"context"
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
)

// ProcessUpdate processes the websocket orderbook update
func (b *Binance) ProcessUpdate(ctx context.Context, ws *WebsocketDepthStream) error {
	pair, enabled, err := b.MatchSymbolCheckEnabled(ws.Pair, asset.Spot, false)
	if err != nil {
		return err
	}
	if !enabled {
		return nil
	}
	updateBid := make([]orderbook.Item, len(ws.UpdateBids))
	for i := range ws.UpdateBids {
		updateBid[i] = orderbook.Item{
			Price:  ws.UpdateBids[i][0].Float64(),
			Amount: ws.UpdateBids[i][1].Float64(),
		}
	}
	updateAsk := make([]orderbook.Item, len(ws.UpdateAsks))
	for i := range ws.UpdateAsks {
		updateAsk[i] = orderbook.Item{
			Price:  ws.UpdateAsks[i][0].Float64(),
			Amount: ws.UpdateAsks[i][1].Float64(),
		}
	}
	return b.OrderbookBuilder.Process(ctx, &orderbook.Update{
		Bids:       updateBid,
		Asks:       updateAsk,
		Pair:       pair,
		UpdateID:   ws.LastUpdateID,
		UpdateTime: ws.Timestamp,
		Asset:      asset.Spot,
	})
}

// GetBuildableBook fetches an orderbook to build a local cache for websocket
// streaming
func (b *Binance) GetBuildableBook(ctx context.Context, p currency.Pair, a asset.Item) (*orderbook.Base, error) {
	ob, err := b.GetOrderBook(ctx, OrderBookDataRequestParams{p, 1000})
	if err != nil {
		return nil, err
	}
	bids := make([]orderbook.Item, len(ob.Bids))
	for i := range ob.Bids {
		bids[i].Amount = ob.Bids[i].Quantity
		bids[i].Price = ob.Bids[i].Price
	}
	asks := make([]orderbook.Item, len(ob.Asks))
	for i := range ob.Asks {
		asks[i].Amount = ob.Asks[i].Quantity
		asks[i].Price = ob.Asks[i].Price
	}
	return &orderbook.Base{
		Pair:            p,
		Asset:           asset.Spot,
		Exchange:        b.Name,
		LastUpdateID:    ob.LastUpdateID,
		VerifyOrderbook: b.CanVerifyOrderbook,
		Bids:            bids,
		Asks:            asks,
		LastUpdated:     time.Now(), // Time not provided in REST book.
	}, nil
}

func (b *Binance) Validate(loaded *orderbook.Base, incoming *orderbook.Update, initialSync bool) (skip bool, err error) {
	if incoming.UpdateID <= loaded.LastUpdateID {
		// Drop any event where u is <= lastUpdateId in the snapshot.
		return false, nil
	}

	if initialSync {
		return false, nil
	}

	id := loaded.LastUpdateID + 1
	// The first processed event should have U <= lastUpdateId+1 AND
	// u >= lastUpdateId+1.
	if incoming.FirstUpdateID > id || updt.LastUpdateID < id {
		return false, fmt.Errorf("initial websocket orderbook sync failure for pair %s and asset %s",
			incoming.Pair,
			incoming.Asset)
	}
	u.initialSync = false

	return false, nil
}
