package kucoin

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

var (
	errMalformedCandleTopic   = errors.New("malformed candle topic")
	errUnknownOrderChangeType = errors.New("unknown order change type")
	errUnknownStopOrderType   = errors.New("unknown stop order type")
	errUnknownBalanceRelation = errors.New("unknown balance relation event")
	errUnknownMarginTradeType = errors.New("unknown trade type")
	errEmptyOrderbookLevelOne = errors.New("level 1 push carries no price")
)

// processTicker merges a Ticker or All Tickers push onto the stored ticker of each asset the pair is enabled on
func (e *Exchange) processTicker(ctx context.Context, data []byte, symbol string) error {
	var resp WsTicker
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	pair, err := currency.NewPairFromString(symbol)
	if err != nil {
		return err
	}
	return e.processSpotTickers(ctx, pair, marketTickerChannel, func(t *ticker.Price) {
		t.Last = resp.Price.Float64()
		t.LastSize = resp.Size.Float64()
		t.Bid = resp.BestBid.Float64()
		t.BidSize = resp.BestBidSize.Float64()
		t.Ask = resp.BestAsk.Float64()
		t.AskSize = resp.BestAskSize.Float64()
		t.LastUpdated = resp.Time.Time()
	})
}

// processSpotTickers merges a push onto the stored ticker of each asset the pair is enabled on, stores the tickers and
// relays the ones stored
func (e *Exchange) processSpotTickers(ctx context.Context, pair currency.Pair, channel string, apply func(*ticker.Price)) error {
	assets, err := e.CalculateAssets(channel, pair)
	if err != nil {
		return err
	}
	tickers := make([]ticker.Price, 0, len(assets))
	for _, a := range assets {
		if !e.AssetWebsocketSupport.IsAssetWebsocketSupported(a) {
			continue
		}
		t, err := e.mergeTicker(pair, a, apply)
		if err != nil {
			return err
		}
		tickers = append(tickers, *t)
	}
	processed, err := ticker.ProcessBatch(tickers)
	if len(processed) == 0 {
		return err
	}
	return common.AppendError(err, e.Websocket.DataHandler.Send(ctx, processed))
}

// processOrderbookLevel1 loads an Orderbook - Level 1 push as a one level book for each asset the pair is enabled on
func (e *Exchange) processOrderbookLevel1(ctx context.Context, data []byte, symbol string) error {
	var resp WsOrderbookLevel1
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	pair, err := currency.NewPairFromString(symbol)
	if err != nil {
		return err
	}
	if resp.Asks[0].Float64() == 0 && resp.Bids[0].Float64() == 0 {
		return fmt.Errorf("%w: %s", errEmptyOrderbookLevelOne, symbol)
	}
	assets, err := e.CalculateAssets(marketOrderbookDepth1Channel, pair)
	if err != nil {
		return err
	}
	for _, a := range assets {
		book := &orderbook.Book{
			Exchange:    e.Name,
			Pair:        pair,
			Asset:       a,
			LastUpdated: resp.Timestamp.Time(),
		}
		// A side with no orders arrives without a price, and a level without a price is not a level
		if resp.Asks[0].Float64() != 0 {
			book.Asks = orderbook.Levels{{Price: resp.Asks[0].Float64(), Amount: resp.Asks[1].Float64()}}
		}
		if resp.Bids[0].Float64() != 0 {
			book.Bids = orderbook.Levels{{Price: resp.Bids[0].Float64(), Amount: resp.Bids[1].Float64()}}
		}
		if err := e.Websocket.Orderbook.LoadSnapshot(ctx, book); err != nil {
			return err
		}
	}
	return nil
}

// processOrderbook loads an Orderbook - Level 5 or Level 50 push for each asset the pair is enabled on
func (e *Exchange) processOrderbook(ctx context.Context, data []byte, symbol, channel string) error {
	var resp WsOrderbookDepth
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	pair, err := currency.NewPairFromString(symbol)
	if err != nil {
		return err
	}
	assets, err := e.CalculateAssets(channel, pair)
	if err != nil {
		return err
	}
	lastUpdated := resp.Timestamp.Time()
	if lastUpdated.IsZero() {
		lastUpdated = time.Now()
	}
	asks := mergeRoundedOrderbookLevels(resp.Asks.Levels())
	bids := mergeRoundedOrderbookLevels(resp.Bids.Levels())
	for _, a := range assets {
		if err := e.Websocket.Orderbook.LoadSnapshot(ctx, &orderbook.Book{
			Exchange:    e.Name,
			Asks:        asks,
			Bids:        bids,
			Pair:        pair,
			Asset:       a,
			LastUpdated: lastUpdated,
		}); err != nil {
			return err
		}
	}
	return nil
}

// processSpotOrderbookIncrement applies an Orderbook - Increment push to the book of each asset the pair is enabled on
func (e *Exchange) processSpotOrderbookIncrement(ctx context.Context, data []byte, symbol string) error {
	pair, err := currency.NewPairFromString(symbol)
	if err != nil {
		return err
	}
	var resp WsOrderbookIncrement
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	bids := make([]orderbook.Level, len(resp.Changes.Bids))
	for i := range resp.Changes.Bids {
		bids[i] = orderbook.Level{
			Price:  resp.Changes.Bids[i][0].Float64(),
			Amount: resp.Changes.Bids[i][1].Float64(),
			ID:     resp.Changes.Bids[i][2].Int64(),
		}
	}
	asks := make([]orderbook.Level, len(resp.Changes.Asks))
	for i := range resp.Changes.Asks {
		asks[i] = orderbook.Level{
			Price:  resp.Changes.Asks[i][0].Float64(),
			Amount: resp.Changes.Asks[i][1].Float64(),
			ID:     resp.Changes.Asks[i][2].Int64(),
		}
	}
	assets, err := e.CalculateAssets(marketOrderbookChannel, pair)
	if err != nil {
		return err
	}
	if len(assets) == 0 {
		// A subscribed pair outside the enabled lists has always been booked as spot.
		assets = []asset.Item{asset.Spot}
	}
	// Each asset gets its own level slices to prevent sharing mutable state.
	for _, a := range assets {
		if err := e.wsOBUpdateMgr.ProcessOrderbookUpdate(ctx, resp.SequenceStart, &orderbook.Update{
			UpdateID:   resp.SequenceEnd,
			UpdateTime: resp.Time.Time(),
			LastPushed: resp.Time.Time(), // Realtime so this is pushed when a change occurs
			Asset:      a,
			Bids:       slices.Clone(bids),
			Asks:       slices.Clone(asks),
			Pair:       pair,
		}); err != nil {
			return err
		}
	}
	return nil
}

// processCallAuctionOrderbook relays a Call Auction Orderbook - Level 50 push; the indicative book of an auction is
// not a book the store can hold
func (e *Exchange) processCallAuctionOrderbook(ctx context.Context, data []byte, symbol string) error {
	resp := &WsCallAuctionOrderbook{Symbol: symbol}
	if err := json.Unmarshal(data, resp); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, resp)
}

// processCallAuctionData relays a Call Auction Data push
func (e *Exchange) processCallAuctionData(ctx context.Context, data []byte) error {
	return processPayload[WsCallAuctionInfo](ctx, e, data)
}

// processCandlesticks relays a spot Klines push as a candle for each asset the pair is enabled on
func (e *Exchange) processCandlesticks(ctx context.Context, data []byte, symbolAndInterval string) error {
	symbol, intervalString, ok := strings.Cut(symbolAndInterval, currency.UnderscoreDelimiter)
	if !ok {
		return fmt.Errorf("%w: %q", errMalformedCandleTopic, symbolAndInterval)
	}
	interval, err := IntervalFromString(intervalString)
	if err != nil {
		return err
	}
	pair, err := currency.NewPairFromString(symbol)
	if err != nil {
		return err
	}
	var resp WsCandlestick
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	assets, err := e.CalculateAssets(marketCandlesChannel, pair)
	if err != nil {
		return err
	}
	for _, a := range assets {
		if !e.AssetWebsocketSupport.IsAssetWebsocketSupported(a) {
			continue
		}
		if err := e.Websocket.DataHandler.Send(ctx, &kline.Item{
			Pair:     pair,
			Asset:    a,
			Exchange: e.Name,
			Interval: interval,
			Candles: []kline.Candle{{
				Time:   resp.Candles.StartTime.Time(),
				Open:   resp.Candles.OpenPrice.Float64(),
				Close:  resp.Candles.ClosePrice.Float64(),
				High:   resp.Candles.HighPrice.Float64(),
				Low:    resp.Candles.LowPrice.Float64(),
				Volume: resp.Candles.TransactionVolume.Float64(),
			}},
		}); err != nil {
			return err
		}
	}
	return nil
}

// processTradeData saves a spot Trade push for each asset the pair is enabled on
func (e *Exchange) processTradeData(data []byte, symbol string) error {
	var resp WsTrade
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	saveTradeData := e.IsSaveTradeDataEnabled()
	if !saveTradeData && !e.IsTradeFeedEnabled() {
		return nil
	}
	pair, err := currency.NewPairFromString(symbol)
	if err != nil {
		return err
	}
	side, err := order.StringToOrderSide(resp.Side)
	if err != nil {
		return err
	}
	assets, err := e.CalculateAssets(marketMatchChannel, pair)
	if err != nil {
		return err
	}
	for _, a := range assets {
		if err := e.Websocket.Trade.Update(saveTradeData, trade.Data{
			CurrencyPair: pair,
			Timestamp:    resp.Time.Time(),
			Price:        resp.Price.Float64(),
			Amount:       resp.Size.Float64(),
			Side:         side,
			Exchange:     e.Name,
			TID:          resp.TradeID,
			AssetType:    a,
		}); err != nil {
			return err
		}
	}
	return nil
}

// processMarketSnapshot merges a Symbol Snapshot or Market Snapshot push onto the stored ticker of each asset the pair
// is enabled on
func (e *Exchange) processMarketSnapshot(ctx context.Context, data []byte) error {
	var resp WsSnapshot
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	pair, err := currency.NewPairFromString(resp.Data.Symbol)
	if err != nil {
		return err
	}
	return e.processSpotTickers(ctx, pair, marketSnapshotChannel, func(t *ticker.Price) {
		t.Last = resp.Data.LastTradedPrice
		t.LastSize = resp.Data.LastSize
		t.Bid = resp.Data.Buy
		t.BidSize = resp.Data.BidSize
		t.Ask = resp.Data.Sell
		t.AskSize = resp.Data.AskSize
		t.High = resp.Data.High
		t.Low = resp.Data.Low
		t.Open = resp.Data.Open
		t.Close = resp.Data.Close
		t.BaseVolume = resp.Data.Volume
		t.QuoteVolume = resp.Data.VolumeValue
		t.LastUpdated = resp.Data.Datetime.Time()
	})
}

// processOrderChange relays an Order V2 or Order V1 push as an order update
func (e *Exchange) processOrderChange(ctx context.Context, data []byte) error {
	var resp WsSpotOrderChange
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	pair, err := currency.NewPairFromString(resp.Symbol)
	if err != nil {
		return err
	}
	oType, err := order.StringToOrderType(resp.OrderType)
	if err != nil {
		return err
	}
	side, err := order.StringToOrderSide(resp.Side)
	if err != nil {
		return err
	}
	status, err := orderChangeStatus(resp.Status, resp.Type, resp.FilledSize.Float64(), resp.RemainSize.Float64())
	if err != nil {
		return err
	}
	amount := resp.Size.Float64()
	if amount == 0 {
		// A received message only carries the size the order was placed with
		amount = resp.OriginSize.Float64()
	}
	detail := &order.Detail{
		Price:           resp.Price.Float64(),
		Amount:          amount,
		ExecutedAmount:  resp.FilledSize.Float64(),
		RemainingAmount: resp.RemainSize.Float64(),
		Exchange:        e.Name,
		OrderID:         resp.OrderID,
		ClientOrderID:   resp.ClientOrderID,
		Type:            oType,
		Side:            side,
		Status:          status,
		AssetType:       e.spotOrderAsset(pair),
		Date:            resp.OrderTime.Time(),
		LastUpdated:     resp.Timestamp.Time(),
		Pair:            pair,
	}
	if resp.Type == "match" {
		detail.Trades = []order.TradeHistory{{
			Price:     resp.MatchPrice.Float64(),
			Amount:    resp.MatchSize.Float64(),
			Exchange:  e.Name,
			TID:       resp.TradeID,
			Type:      oType,
			Side:      side,
			Timestamp: resp.Timestamp.Time(),
			IsMaker:   resp.Liquidity == "maker",
		}}
	}
	return e.Websocket.DataHandler.Send(ctx, detail)
}

// spotOrderAsset returns the asset of a spot or margin order push, which shares one topic and does not say which it
// is: spot when the pair is enabled for spot, else margin when enabled for margin
func (e *Exchange) spotOrderAsset(pair currency.Pair) asset.Item {
	if enabled, err := e.IsPairEnabled(pair, asset.Spot); err == nil && enabled {
		return asset.Spot
	}
	if enabled, err := e.IsPairEnabled(pair, asset.Margin); err == nil && enabled {
		return asset.Margin
	}
	return asset.Spot
}

// orderChangeStatus returns the status of an order change push. The message type says whether a done order filled or
// was cancelled, so it is read with the status, and partial fills follow from the filled and remaining sizes
func orderChangeStatus(status, messageType string, filled, remaining float64) (order.Status, error) {
	switch messageType {
	case "received":
		return order.New, nil
	case "open":
		return order.Open, nil
	case "match":
		if remaining == 0 {
			return order.Filled, nil
		}
		return order.PartiallyFilled, nil
	case "filled":
		return order.Filled, nil
	case "canceled":
		if filled > 0 {
			return order.PartiallyFilledCancelled, nil
		}
		return order.Cancelled, nil
	case "update":
		switch {
		case status == "done" && filled > 0:
			return order.PartiallyFilledCancelled, nil
		case status == "done":
			return order.Cancelled, nil
		case filled > 0:
			return order.PartiallyFilled, nil
		default:
			return order.Open, nil
		}
	}
	return order.UnknownStatus, fmt.Errorf("%w: %q", errUnknownOrderChangeType, messageType)
}

// processAccountBalance saves a Balance push of the spot (trade, trade_hf) or cross margin (margin, marginV2) account and
// relays it; changes to other accounts, which have no sub-account here, are relayed as they come
func (e *Exchange) processAccountBalance(ctx context.Context, data []byte) error {
	var resp WsAccountBalance
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	a, err := balanceAsset(resp.RelationEvent)
	if err != nil {
		return err
	}
	if a == asset.Empty {
		return e.Websocket.DataHandler.Send(ctx, &resp)
	}
	subAccts := accounts.SubAccounts{accounts.NewSubAccount(a, "")}
	subAccts[0].Balances.Set(resp.Currency, accounts.Balance{
		Total:     resp.Total.Float64(),
		Hold:      resp.Hold.Float64(),
		Free:      resp.Available.Float64(),
		UpdatedAt: resp.Time.Time(),
	})
	if err := e.Accounts.Save(ctx, subAccts, false); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, subAccts)
}

// balanceAsset returns the asset of the account a balance relation event belongs to, or no asset for the main,
// isolated margin and other accounts. The account is the event's prefix: trade.hold, isolated_BTC-USDT.hold
func balanceAsset(relationEvent string) (asset.Item, error) {
	account, _, ok := strings.Cut(relationEvent, ".")
	switch {
	case account == "trade", account == "trade_hf":
		return asset.Spot, nil
	case account == "margin", account == "marginV2":
		return asset.Margin, nil
	case account == "main", account == "other" && !ok, strings.HasPrefix(account, "isolated_"), strings.HasPrefix(account, "isolatedV2_"):
		return asset.Empty, nil
	}
	return asset.Empty, fmt.Errorf("%w: %q", errUnknownBalanceRelation, relationEvent)
}

// processStopOrder relays a spot or margin Stop Order push as an order update
func (e *Exchange) processStopOrder(ctx context.Context, data []byte) error {
	var resp WsStopOrder
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	pair, err := currency.NewPairFromString(resp.Symbol)
	if err != nil {
		return err
	}
	oType, err := order.StringToOrderType(resp.OrderType)
	if err != nil {
		return err
	}
	side, err := order.StringToOrderSide(resp.Side)
	if err != nil {
		return err
	}
	status, err := stopOrderStatus(resp.Type)
	if err != nil {
		return err
	}
	var a asset.Item
	var marginType margin.Type
	switch resp.TradeType {
	case "TRADE":
		a = asset.Spot
	case "MARGIN_TRADE":
		a, marginType = asset.Margin, margin.Multi
	case "MARGIN_ISOLATED_TRADE":
		a, marginType = asset.Margin, margin.Isolated
	default:
		return fmt.Errorf("%w: %q", errUnknownMarginTradeType, resp.TradeType)
	}
	return e.Websocket.DataHandler.Send(ctx, &order.Detail{
		Price:        resp.OrderPrice.Float64(),
		TriggerPrice: resp.StopPrice.Float64(),
		Amount:       resp.Size.Float64(),
		Exchange:     e.Name,
		OrderID:      resp.OrderID,
		Type:         oType,
		Side:         side,
		Status:       status,
		AssetType:    a,
		MarginType:   marginType,
		Date:         resp.CreatedAt.Time(),
		LastUpdated:  resp.Timestamp.Time(),
		Pair:         pair,
	})
}

// stopOrderStatus returns the status of a stop order push: a triggered stop order is done, and the order it places is
// reported on the order topics
func stopOrderStatus(messageType string) (order.Status, error) {
	switch messageType {
	case "open":
		return order.Open, nil
	case "triggered":
		return order.Closed, nil
	case "cancel":
		return order.Cancelled, nil
	}
	return order.UnknownStatus, fmt.Errorf("%w: %q", errUnknownStopOrderType, messageType)
}
