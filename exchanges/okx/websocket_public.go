package okx

import (
	"context"
	"errors"
	"fmt"
	"hash/crc32"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/types"
)

var (
	errInvalidOrderbookSequence  = errors.New("invalid orderbook sequence")
	errOrderbookSnapshotPending  = errors.New("orderbook snapshot pending")
	errOrderbookChecksumMismatch = errors.New("orderbook checksum mismatch")
	errOrderbookSourceChanged    = errors.New("orderbook updated by another channel than its snapshot's")
)

// The channels which keep an instrument's order book, deepest first and, at
// one depth, tick-by-tick before 100 ms pushes and organic before consolidated
// RPI depth. A book is kept by the deepest of them its instrument is
// subscribed or being subscribed to; the pushes of the others are relayed as
// they arrive.
var (
	orderBookDepthChannels       = []string{channelOrderBooksTBT, channelOrderBooks, channelOrderBooksRPI, channelOrderBooks50TBT, channelOrderBooks5, channelBBOTBT}
	spreadOrderBookDepthChannels = []string{channelSpreadOrderBooksTBT, channelSpreadOrderBooks5, channelSpreadBBOTBT}
)

// qualifiedChannelKey finds a subscription by its exact subscription argument
type qualifiedChannelKey string

// Match implements subscription.MatchableKey
func (k qualifiedChannelKey) Match(eachKey subscription.MatchableKey) bool {
	s := eachKey.GetSubscription()
	return s != nil && s.QualifiedChannel == string(k)
}

// GetSubscription implements subscription.MatchableKey
func (k qualifiedChannelKey) GetSubscription() *subscription.Subscription {
	return &subscription.Subscription{QualifiedChannel: string(k)}
}

// String implements subscription.MatchableKey
func (k qualifiedChannelKey) String() string {
	return string(k)
}

// GetQualifiedChannel implements subscription.QualifiedChannelKey, so the
// store looks the argument up in its index
func (k qualifiedChannelKey) GetQualifiedChannel() string {
	return string(k)
}

// keepsOrderBook returns whether a push on an order book channel keeps the
// book of the instrument or spread id, which it does unless a deeper channel of
// depthChannels is subscribed, or being subscribed, for it; idField names id
// in the argument, as instId or sprdId
func (e *Exchange) keepsOrderBook(channel string, depthChannels []string, idField string, id currency.Pair) bool {
	// The arguments are built as channelArgument marshals them, since marshalling
	// on every push costs several times the lookups
	argumentEnd := `","` + idField + `":"` + id.String() + `"}`
	for _, deeper := range depthChannels {
		if deeper == channel {
			return true
		}
		argument := `{"channel":"` + deeper + argumentEnd
		if e.Websocket.GetSubscription(qualifiedChannelKey(argument)) != nil || e.subscriptionRequests.isRequested(argument) {
			return false
		}
	}
	return true
}

// orderBookArgument returns the subscription argument of an order book channel
// for an instrument or spread id, as channelArgument marshals it
func orderBookArgument(channel string, id currency.Pair) string {
	idField := "instId"
	if strings.HasPrefix(channel, "sprd-") {
		idField = "sprdId"
	}
	return `{"channel":"` + channel + `","` + idField + `":"` + id.String() + `"}`
}

// orderBookStates holds, by instrument or spread ID, the lock ordering the
// pushes applied to each order book and the channel whose snapshot the book
// was loaded from, and the order book arguments whose book needs a fresh
// snapshot on their channel's next push finding it without one: their
// recovery waits for their subscription to be tracked on its connection, or
// their snapshot was relayed while a deeper channel kept the book or was
// being subscribed
type orderBookStates struct {
	mu       sync.Mutex
	locks    map[string]*sync.Mutex
	sources  map[string]string
	deferred map[string]bool
}

// lock locks and returns the lock ordering the pushes applied to an ID's order
// books, so that the connections pushing them change a book and the channel
// recorded as its source together
func (s *orderBookStates) lock(id string) *sync.Mutex {
	s.mu.Lock()
	if s.locks == nil {
		s.locks = make(map[string]*sync.Mutex)
	}
	l, ok := s.locks[id]
	if !ok {
		l = new(sync.Mutex)
		s.locks[id] = l
	}
	s.mu.Unlock()
	l.Lock()
	return l
}

// load records the channel whose snapshot a book was loaded from; the book
// needs no deferred recovery for the channel's argument
func (s *orderBookStates) load(id, channel, argument string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sources == nil {
		s.sources = make(map[string]string)
	}
	s.sources[id] = channel
	delete(s.deferred, argument)
}

// handOver records a channel updating a book as the channel it is loaded from,
// and returns the channel whose snapshot it held when that was another
func (s *orderBookStates) handOver(id, channel string) (source string, changed bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	source, ok := s.sources[id]
	if !ok || source == channel {
		return "", false
	}
	s.sources[id] = channel
	return source, true
}

// deferRecovery records that the book of an argument needs a fresh snapshot on
// its channel's next push finding the book without one
func (s *orderBookStates) deferRecovery(argument string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.deferred == nil {
		s.deferred = make(map[string]bool)
	}
	s.deferred[argument] = true
}

// takeDeferredRecovery returns whether the book of an argument needs a
// deferred recovery, which it no longer needs once taken
func (s *orderBookStates) takeDeferredRecovery(argument string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	deferred := s.deferred[argument]
	delete(s.deferred, argument)
	return deferred
}

// pushAssets returns the assets a push is for: the asset of its instrument
// type, or every enabled asset holding its instrument
func (e *Exchange) pushAssets(arg *SubscriptionArgument) ([]asset.Item, error) {
	if arg.InstrumentType != "" {
		a, err := assetTypeFromInstrumentType(arg.InstrumentType)
		if err != nil {
			return nil, err
		}
		return []asset.Item{a}, nil
	}
	return e.getAssetsFromInstrumentID(arg.InstrumentID.String())
}

// wsProcessTickers processes tickers into the ticker store and relays those
// which processed; a pre-open instrument's ticker may quote its best ask
// below its best bid
func (e *Exchange) wsProcessTickers(ctx context.Context, data []byte) error {
	var resp WebsocketTickersPush
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	tickerPrices := make([]ticker.Price, 0, len(resp.Data))
	for i := range resp.Data {
		assets, err := e.pushAssets(&resp.Argument)
		if err != nil {
			return err
		}
		preOpen := e.instrumentPreOpen(resp.Data[i].InstrumentID.String(), resp.Data[i].TickerDataGenerationTime.Time())
		for _, a := range assets {
			baseVolume, quoteVolume := tickerVolumes(&resp.Data[i], a)
			tickerPrices = append(tickerPrices, ticker.Price{
				ExchangeName: e.Name,
				Open:         resp.Data[i].OpenPrice24Hour.Float64(),
				BaseVolume:   baseVolume,
				QuoteVolume:  quoteVolume,
				High:         resp.Data[i].HighestPrice24Hour.Float64(),
				Low:          resp.Data[i].LowestPrice24Hour.Float64(),
				Bid:          resp.Data[i].BestBidPrice.Float64(),
				Ask:          resp.Data[i].BestAskPrice.Float64(),
				BidSize:      resp.Data[i].BestBidSize.Float64(),
				AskSize:      resp.Data[i].BestAskSize.Float64(),
				Last:         resp.Data[i].LastTradePrice.Float64(),
				AssetType:    a,
				Pair:         resp.Data[i].InstrumentID,
				LastUpdated:  resp.Data[i].TickerDataGenerationTime.Time(),
				PreOpen:      preOpen,
			})
		}
	}
	processed, err := ticker.ProcessBatch(tickerPrices)
	if len(processed) == 0 {
		return err
	}
	return common.AppendError(err, e.Websocket.DataHandler.Send(ctx, processed))
}

// wsProcessCandles relays each candlestick of a candlestick push for every
// asset its instrument is enabled for
func (e *Exchange) wsProcessCandles(ctx context.Context, respRaw []byte) error {
	var resp WebsocketCandlesticksPush
	if err := json.Unmarshal(respRaw, &resp); err != nil {
		return err
	}
	if len(resp.Data) == 0 {
		return kline.ErrNoTimeSeriesDataToConvert
	}
	interval, err := candleChannelInterval(resp.Argument.Channel, channelCandle)
	if err != nil {
		return err
	}
	assets, err := e.pushAssets(&resp.Argument)
	if err != nil {
		return err
	}
	for i := range resp.Data {
		for _, a := range assets {
			if err := e.Websocket.DataHandler.Send(ctx, kline.Item{
				Pair:     resp.Argument.InstrumentID,
				Asset:    a,
				Exchange: e.Name,
				Interval: interval,
				Candles: []kline.Candle{{
					Time:   resp.Data[i].Timestamp.Time(),
					Open:   resp.Data[i].OpenPrice.Float64(),
					High:   resp.Data[i].HighestPrice.Float64(),
					Low:    resp.Data[i].LowestPrice.Float64(),
					Close:  resp.Data[i].ClosePrice.Float64(),
					Volume: candleVolume(resp.Data[i].Volume, resp.Data[i].VolumeCurrency, a),
				}},
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

// relayTrades relays trades to the trade feed and saves them, as the exchange
// is configured to; a relay which cannot deliver does not stop them saving.
// The trades are relayed first because saving normalises them in place.
func (e *Exchange) relayTrades(ctx context.Context, trades []trade.Data) error {
	var relayErr error
	if e.IsTradeFeedEnabled() {
		for i := range trades {
			if relayErr = e.Websocket.DataHandler.Send(ctx, trades[i]); relayErr != nil {
				break
			}
		}
	}
	if !e.IsSaveTradeDataEnabled() {
		return relayErr
	}
	return common.AppendError(relayErr, trade.AddTradesToBuffer(trades...))
}

// wsProcessTrades processes a trades or trades-all push
func (e *Exchange) wsProcessTrades(ctx context.Context, data []byte) error {
	if !e.IsSaveTradeDataEnabled() && !e.IsTradeFeedEnabled() {
		return nil
	}
	var resp WebsocketTradesPush
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	assets, err := e.pushAssets(&resp.Argument)
	if err != nil {
		return err
	}
	trades := make([]trade.Data, 0, len(resp.Data)*len(assets))
	for i := range resp.Data {
		for _, a := range assets {
			trades = append(trades, trade.Data{
				Amount:       resp.Data[i].Size.Float64(),
				AssetType:    a,
				CurrencyPair: resp.Data[i].InstrumentID,
				Exchange:     e.Name,
				Side:         resp.Data[i].Side,
				Timestamp:    resp.Data[i].Timestamp.Time().UTC(),
				TID:          resp.Data[i].TradeID,
				Price:        resp.Data[i].Price.Float64(),
			})
		}
	}
	return e.relayTrades(ctx, trades)
}

// wsProcessOptionTrades processes an option-trades push
func (e *Exchange) wsProcessOptionTrades(ctx context.Context, data []byte) error {
	if !e.IsSaveTradeDataEnabled() && !e.IsTradeFeedEnabled() {
		return nil
	}
	var resp WebsocketOptionTradesPush
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	trades := make([]trade.Data, len(resp.Data))
	for i := range resp.Data {
		trades[i] = trade.Data{
			Amount:       resp.Data[i].Size.Float64(),
			AssetType:    asset.Options,
			CurrencyPair: resp.Data[i].InstrumentID,
			Exchange:     e.Name,
			Side:         resp.Data[i].Side,
			Timestamp:    resp.Data[i].Timestamp.Time(),
			TID:          resp.Data[i].TradeID,
			Price:        resp.Data[i].Price.Float64(),
		}
	}
	return e.relayTrades(ctx, trades)
}

// wsProcessPublicBlockTrades processes a public-block-trades push, whose
// trades OKX pushes 15 minutes after execution
func (e *Exchange) wsProcessPublicBlockTrades(ctx context.Context, data []byte) error {
	if !e.IsSaveTradeDataEnabled() && !e.IsTradeFeedEnabled() {
		return nil
	}
	var resp WebsocketPublicBlockTradesPush
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	assets, err := e.pushAssets(&resp.Argument)
	if err != nil {
		return err
	}
	trades := make([]trade.Data, 0, len(resp.Data)*len(assets))
	for i := range resp.Data {
		for _, a := range assets {
			trades = append(trades, trade.Data{
				Amount:       resp.Data[i].Size.Float64(),
				AssetType:    a,
				CurrencyPair: resp.Data[i].InstrumentID,
				Exchange:     e.Name,
				Side:         resp.Data[i].Side,
				Timestamp:    resp.Data[i].Timestamp.Time(),
				TID:          resp.Data[i].TradeID,
				Price:        resp.Data[i].Price.Float64(),
			})
		}
	}
	return e.relayTrades(ctx, trades)
}

// orderBookPush holds the levels, sequence and checksum of an order book push;
// the checksum is OKX's signed CRC32 as its bits, and zero when OKX sends none
type orderBookPush struct {
	asks, bids         orderbook.Levels
	updated            time.Time
	previousSequenceID int64
	sequenceID         int64
	checksum           uint32
}

// heldBy returns whether a book at lastUpdateID already holds the push: a stale
// update, or a keep-alive, which OKX pushes with an unchanged sequence ID
func (p *orderBookPush) heldBy(lastUpdateID int64) bool {
	return p.sequenceID == lastUpdateID && p.previousSequenceID <= lastUpdateID
}

// orderBookChecksum returns the CRC32 checksum OKX documents for an order book:
// the price and size, as sent, of its first 25 bids and asks, alternating from
// the best bid and joined by colons
func orderBookChecksum(b *orderbook.Book) uint32 {
	var s strings.Builder
	for i := range 25 {
		for _, levels := range [2]orderbook.Levels{b.Bids, b.Asks} {
			if i >= len(levels) {
				continue
			}
			if s.Len() != 0 {
				s.WriteByte(':')
			}
			s.WriteString(levels[i].StrPrice)
			s.WriteByte(':')
			s.WriteString(levels[i].StrAmount)
		}
	}
	return crc32.ChecksumIEEE([]byte(s.String()))
}

// orderBookLevels converts order book levels of price, size, a deprecated or
// RPI size, and order count. Only a snapshot keeps the count: an update to a
// level's size keeps the level's count in the stored book, so it would go
// stale.
func orderBookLevels(levels [][4]types.Number, withCount bool) orderbook.Levels {
	l := make(orderbook.Levels, len(levels))
	for i := range levels {
		l[i] = orderbook.Level{Price: levels[i][0].Float64(), Amount: levels[i][1].Float64()}
		if withCount {
			l[i].OrderCount = levels[i][3].Int64()
		}
	}
	return l
}

// spreadOrderBookLevels converts spread order book levels of price, size and
// order count, keeping the count as orderBookLevels does, and the price and
// size as sent, which the checksum is calculated from
func spreadOrderBookLevels(levels [][3]types.PreciseNumber, withCount bool) orderbook.Levels {
	l := make(orderbook.Levels, len(levels))
	for i := range levels {
		l[i] = orderbook.Level{
			Price:     levels[i][0].Float64(),
			StrPrice:  levels[i][0].String(),
			Amount:    levels[i][1].Float64(),
			StrAmount: levels[i][1].String(),
		}
		if withCount {
			l[i].OrderCount = levels[i][2].Int64()
		}
	}
	return l
}

// wsProcessOrderBooks processes an instrument's order book push: books5 and
// bbo-tbt push snapshots without an action, the other order book channels a
// snapshot followed by updates
func (e *Exchange) wsProcessOrderBooks(ctx context.Context, conn websocket.Connection, data []byte) error {
	var resp WebsocketOrderBookPush
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	snapshotOnly := resp.Argument.Channel == channelOrderBooks5 || resp.Argument.Channel == channelBBOTBT
	if (snapshotOnly && resp.Action != "" && resp.Action != wsOrderbookSnapshot) ||
		(!snapshotOnly && resp.Action != wsOrderbookUpdate && resp.Action != wsOrderbookSnapshot) {
		return fmt.Errorf("%w, %s", orderbook.ErrInvalidAction, resp.Action)
	}
	if !resp.Argument.InstrumentID.IsPopulated() {
		return currency.ErrCurrencyPairsEmpty
	}
	assets, err := e.pushAssets(&resp.Argument)
	if err != nil {
		return err
	}
	pair := resp.Argument.InstrumentID
	pair.Delimiter = currency.DashDelimiter
	books := e.orderBookStates.lock(pair.String())
	defer books.Unlock()
	if !e.keepsOrderBook(resp.Argument.Channel, orderBookDepthChannels, "instId", pair) {
		if !snapshotOnly && resp.Action == wsOrderbookSnapshot {
			e.orderBookStates.deferRecovery(orderBookArgument(resp.Argument.Channel, pair))
		}
		return e.Websocket.DataHandler.Send(ctx, &resp)
	}
	for i := range resp.Data {
		push := &orderBookPush{
			asks:               orderBookLevels(resp.Data[i].Asks, snapshotOnly),
			bids:               orderBookLevels(resp.Data[i].Bids, snapshotOnly),
			updated:            resp.Data[i].Timestamp.Time(),
			previousSequenceID: resp.Data[i].PreviousSequenceID,
			sequenceID:         resp.Data[i].SequenceID,
		}
		if err := e.applyOrderBookPush(ctx, conn, resp.Argument.Channel, pair, assets, push, snapshotOnly || resp.Action == wsOrderbookSnapshot); err != nil {
			return err
		}
	}
	return nil
}

// wsProcessSpreadOrderBooks processes a spread's order book push: sprd-books5
// and sprd-bbo-tbt push snapshots without an action, sprd-books-l2-tbt a
// snapshot followed by updates
func (e *Exchange) wsProcessSpreadOrderBooks(ctx context.Context, conn websocket.Connection, data []byte) error {
	var resp WebsocketSpreadOrderBookPush
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	snapshotOnly := resp.Argument.Channel != channelSpreadOrderBooksTBT
	if (snapshotOnly && resp.Action != "" && resp.Action != wsOrderbookSnapshot) ||
		(!snapshotOnly && resp.Action != wsOrderbookUpdate && resp.Action != wsOrderbookSnapshot) {
		return fmt.Errorf("%w, %s", orderbook.ErrInvalidAction, resp.Action)
	}
	if !resp.Argument.SpreadID.IsPopulated() {
		return currency.ErrCurrencyPairsEmpty
	}
	books := e.orderBookStates.lock(resp.Argument.SpreadID.String())
	defer books.Unlock()
	if !e.keepsOrderBook(resp.Argument.Channel, spreadOrderBookDepthChannels, "sprdId", resp.Argument.SpreadID) {
		return e.Websocket.DataHandler.Send(ctx, &resp)
	}
	for i := range resp.Data {
		push := &orderBookPush{
			asks:               spreadOrderBookLevels(resp.Data[i].Asks, snapshotOnly),
			bids:               spreadOrderBookLevels(resp.Data[i].Bids, snapshotOnly),
			updated:            resp.Data[i].Timestamp.Time(),
			previousSequenceID: resp.Data[i].PreviousSequenceID,
			sequenceID:         resp.Data[i].SequenceID,
			checksum:           uint32(resp.Data[i].Checksum), //nolint:gosec // The signed CRC32 keeps its 32 bits
		}
		if err := e.applyOrderBookPush(ctx, conn, resp.Argument.Channel, resp.Argument.SpreadID, []asset.Item{asset.Spread}, push, snapshotOnly || resp.Action == wsOrderbookSnapshot); err != nil {
			return err
		}
	}
	return nil
}

// applyOrderBookPush loads a snapshot or applies an update to the order books
// of a pair's assets. A sequence gap, an invalid book or an update from
// another channel than the one whose snapshot the book holds resubscribes the
// channel for a fresh snapshot, and the errors relaying the books updated are
// returned beside it. A channel's snapshot which does not load leaves the
// books as they were, unless they hold another channel's snapshot: then they
// are invalidated and the channel resubscribed. A recovery whose subscription
// is not yet tracked on the connection runs on the channel's next push once
// it is.
func (e *Exchange) applyOrderBookPush(ctx context.Context, conn websocket.Connection, channel string, pair currency.Pair, assets []asset.Item, push *orderBookPush, snapshot bool) error {
	argument := orderBookArgument(channel, pair)
	var dispatchErr, err error
	if snapshot {
		if dispatchErr, err = e.wsProcessSnapshotOrderBook(ctx, push, pair, assets); err == nil {
			e.orderBookStates.load(pair.String(), channel, argument)
			return dispatchErr
		}
		if _, changed := e.orderBookStates.handOver(pair.String(), channel); changed && !errors.Is(err, orderbook.ErrOrderbookInvalid) {
			err = e.invalidateOrderbooks(pair, assets, err)
			return common.AppendError(err, e.recoverOrderBook(ctx, conn, channel, pair, err))
		}
	} else if source, changed := e.orderBookStates.handOver(pair.String(), channel); changed {
		err = e.invalidateOrderbooks(pair, assets, fmt.Errorf("%w: %v loaded from %s, updated by %s", errOrderbookSourceChanged, pair, source, channel))
	} else {
		dispatchErr, err = e.wsProcessUpdateOrderBook(ctx, push, pair, assets)
	}
	switch {
	case err == nil:
		return dispatchErr
	case errors.Is(err, errOrderbookSnapshotPending), errors.Is(err, orderbook.ErrDepthNotFound):
		if !e.orderBookStates.takeDeferredRecovery(argument) {
			if errors.Is(err, errOrderbookSnapshotPending) {
				return dispatchErr
			}
			return common.AppendError(dispatchErr, err)
		}
	case !errors.Is(err, errInvalidOrderbookSequence) && !errors.Is(err, orderbook.ErrOrderbookInvalid) && !errors.Is(err, errOrderbookSourceChanged):
		return common.AppendError(dispatchErr, err)
	}
	return common.AppendError(dispatchErr, e.recoverOrderBook(ctx, conn, channel, pair, err))
}

// recoverOrderBook resubscribes the subscriptions to an order book channel for
// a pair on conn, for a fresh snapshot of the book, which any recovery
// deferred for the channel's argument awaits too. A recovery whose
// subscription is being subscribed, or tracked but not yet on the
// connection, is deferred to the channel's next push.
func (e *Exchange) recoverOrderBook(ctx context.Context, conn websocket.Connection, channel string, pair currency.Pair, cause error) error {
	argument := orderBookArgument(channel, pair)
	matches := func(sub *subscription.Subscription) bool {
		name, err := channelName(sub, sub.Asset)
		return err == nil && name == channel && sub.Pairs.Contains(pair, true)
	}
	var resubscribe subscription.List
	if conn != nil {
		for _, sub := range conn.Subscriptions().List() {
			if matches(sub) {
				resubscribe = append(resubscribe, sub)
			}
		}
	}
	if len(resubscribe) == 0 {
		if len(e.subscriptionRequests.subscribing(conn, matches)) == 0 && e.Websocket.GetSubscription(qualifiedChannelKey(argument)) == nil {
			return fmt.Errorf("%w: %s %s", subscription.ErrNotFound, channel, pair)
		}
		e.orderBookStates.deferRecovery(argument)
		return nil
	}
	e.orderBookStates.takeDeferredRecovery(argument)
	if e.Verbose {
		log.Debugf(log.ExchangeSys, "Resubscribing to %v due to orderbook error: %v", resubscribe.Strings(), cause)
	}
	go func() {
		if err := e.Websocket.ResubscribeFromConnection(ctx, conn, resubscribe); err != nil {
			log.Errorf(log.ExchangeSys, "Failed to resubscribe %v: %v", resubscribe.Strings(), err)
		}
	}()
	return nil
}

// wsProcessSnapshotOrderBook loads an order book snapshot for each asset, once
// the snapshot matches its checksum when it has one and books are validated,
// and returns the errors relaying the books loaded apart from the error which
// left every book unloaded. The snapshot is first checked as loading checks a
// book, so a book fails to load only before any does, and every asset loads
// even when one fails to relay.
func (e *Exchange) wsProcessSnapshotOrderBook(ctx context.Context, push *orderBookPush, pair currency.Pair, assets []asset.Item) (dispatchErr, err error) {
	if push.checksum != 0 && e.ValidateOrderbook {
		if checksum := orderBookChecksum(&orderbook.Book{Bids: push.bids, Asks: push.asks}); checksum != push.checksum {
			return nil, e.invalidateOrderbooks(pair, assets, fmt.Errorf("%w: %w %v: pushed %d, calculated %d", orderbook.ErrOrderbookInvalid, errOrderbookChecksumMismatch, pair, push.checksum, checksum))
		}
	}
	if push.updated.IsZero() {
		return nil, fmt.Errorf("%w: %v snapshot", orderbook.ErrLastUpdatedNotSet, pair)
	}
	books := make([]*orderbook.Book, len(assets))
	for i, a := range assets {
		books[i] = &orderbook.Book{
			LastUpdateID:      push.sequenceID,
			Asset:             a,
			Asks:              push.asks,
			Bids:              push.bids,
			LastUpdated:       push.updated,
			Pair:              pair,
			Exchange:          e.Name,
			ValidateOrderbook: e.ValidateOrderbook,
		}
		if err := books[i].Validate(); err != nil {
			return nil, err
		}
	}
	for _, b := range books {
		dispatchErr = common.AppendError(dispatchErr, e.Websocket.Orderbook.LoadSnapshot(ctx, b))
	}
	return dispatchErr, nil
}

// wsProcessUpdateOrderBook applies an order book update to each asset's book,
// returning the errors relaying the books it updated apart from the error
// applying it. OKX can reset sequence IDs to a lower value while retaining
// continuity through prevSeqId; see https://www.okx.com/docs-v5/en/#order-book-trading-market-data-ws-order-book-channel
func (e *Exchange) wsProcessUpdateOrderBook(ctx context.Context, push *orderBookPush, pair currency.Pair, assets []asset.Item) (dispatchErr, err error) {
	// A push without instType can map one instrument to several assets, such
	// as spot and margin, so every book is checked before any is changed.
	lastUpdateIDs := make([]int64, len(assets))
	for i, a := range assets {
		lastUpdateID, err := e.Websocket.Orderbook.LastUpdateID(pair, a)
		if err != nil {
			if errors.Is(err, orderbook.ErrOrderbookInvalid) {
				err = fmt.Errorf("%w %v %v: %w", errOrderbookSnapshotPending, pair, a, err)
			}
			return nil, e.invalidateOrderbooks(pair, assets, err)
		}
		lastUpdateIDs[i] = lastUpdateID
	}
	for i, a := range assets {
		if !push.heldBy(lastUpdateIDs[i]) && push.previousSequenceID != lastUpdateIDs[i] {
			return nil, e.invalidateOrderbooks(pair, assets, fmt.Errorf("%w %v %v: previous sequence ID %d, last update ID %d", errInvalidOrderbookSequence, pair, a, push.previousSequenceID, lastUpdateIDs[i]))
		}
	}
	for i, a := range assets {
		if push.heldBy(lastUpdateIDs[i]) {
			continue
		}
		if err := e.Websocket.Orderbook.Update(ctx, &orderbook.Update{
			UpdateID:         push.sequenceID,
			Pair:             pair,
			Asset:            a,
			UpdateTime:       push.updated,
			LastPushed:       push.updated,
			Asks:             push.asks,
			Bids:             push.bids,
			ExpectedChecksum: push.checksum,
			GenerateChecksum: orderBookChecksum,
			AllowEmpty:       true, // An update without levels still advances the sequence
		}); err != nil {
			if !errors.Is(err, orderbook.ErrOrderbookInvalid) {
				dispatchErr = common.AppendError(dispatchErr, err)
				continue
			}
			return dispatchErr, e.invalidateOrderbooks(pair, assets, err)
		}
	}
	return dispatchErr, nil
}

// invalidateOrderbooks invalidates a pair's book for each asset which has one,
// so no stale depth stays in service, and returns err with any failure to
// invalidate appended
func (e *Exchange) invalidateOrderbooks(pair currency.Pair, assets []asset.Item, err error) error {
	for _, a := range assets {
		if invalidateErr := e.Websocket.Orderbook.InvalidateOrderbook(pair, a); !errors.Is(invalidateErr, orderbook.ErrDepthNotFound) {
			err = common.AppendError(err, invalidateErr)
		}
	}
	return err
}

// refreshEquivalentOrderbookSnapshot copies the order book of a spot or
// margin subscription's equivalent asset, which shares its outbound
// subscription, to the subscription's asset, in order with the pushes applied
// to the instrument's books
func (e *Exchange) refreshEquivalentOrderbookSnapshot(ctx context.Context, sub *subscription.Subscription) error {
	if sub == nil || sub.Channel != subscription.OrderbookChannel || len(sub.Pairs) == 0 {
		return nil
	}
	inverse, ok := inverseSpotMarginSubscription(sub)
	if !ok {
		return nil
	}
	instrument := sub.Pairs[0]
	instrument.Delimiter = currency.DashDelimiter
	books := e.orderBookStates.lock(instrument.String())
	defer books.Unlock()
	book, err := e.Websocket.Orderbook.GetOrderbook(sub.Pairs[0], inverse.Asset)
	if err != nil {
		if errors.Is(err, orderbook.ErrDepthNotFound) {
			return nil
		}
		return err
	}
	cloned := *book
	cloned.Asset = sub.Asset
	cloned.Pair = sub.Pairs[0]
	return e.Websocket.Orderbook.LoadSnapshot(ctx, &cloned)
}

// wsProcessSpreadPublicTrades processes a sprd-public-trades push
func (e *Exchange) wsProcessSpreadPublicTrades(ctx context.Context, respRaw []byte) error {
	if !e.IsSaveTradeDataEnabled() && !e.IsTradeFeedEnabled() {
		return nil
	}
	var resp WebsocketSpreadPublicTradesPush
	if err := json.Unmarshal(respRaw, &resp); err != nil {
		return err
	}
	trades := make([]trade.Data, len(resp.Data))
	for i := range resp.Data {
		trades[i] = trade.Data{
			TID:          resp.Data[i].TradeID,
			Exchange:     e.Name,
			CurrencyPair: resp.Data[i].SpreadID,
			AssetType:    asset.Spread,
			Side:         resp.Data[i].Side,
			Price:        resp.Data[i].Price.Float64(),
			Amount:       resp.Data[i].Size.Float64(),
			Timestamp:    resp.Data[i].Timestamp.Time(),
		}
	}
	return e.relayTrades(ctx, trades)
}

// wsProcessSpreadTickers processes spread tickers into the ticker store and
// relays those which processed
func (e *Exchange) wsProcessSpreadTickers(ctx context.Context, respRaw []byte) error {
	var resp WebsocketSpreadTickersPush
	if err := json.Unmarshal(respRaw, &resp); err != nil {
		return err
	}
	tickers := make([]ticker.Price, len(resp.Data))
	for i := range resp.Data {
		// vol24h is deliberately not mapped: OKX reports it in USD on an
		// inverse spread, so it is not a base volume, mirroring the REST path.
		tickers[i] = ticker.Price{
			Last:         resp.Data[i].LastPrice.Float64(),
			Bid:          resp.Data[i].BestBidPrice.Float64(),
			BidSize:      resp.Data[i].BestBidSize.Float64(),
			Ask:          resp.Data[i].BestAskPrice.Float64(),
			AskSize:      resp.Data[i].BestAskSize.Float64(),
			Open:         resp.Data[i].OpenPrice24Hour.Float64(),
			High:         resp.Data[i].HighestPrice24Hour.Float64(),
			Low:          resp.Data[i].LowestPrice24Hour.Float64(),
			Pair:         resp.Data[i].SpreadID,
			ExchangeName: e.Name,
			AssetType:    asset.Spread,
			LastUpdated:  resp.Data[i].Timestamp.Time(),
		}
	}
	processed, err := ticker.ProcessBatch(tickers)
	if len(processed) == 0 {
		return err
	}
	return common.AppendError(err, e.Websocket.DataHandler.Send(ctx, processed))
}

// wsProcessSpreadCandles relays each candlestick of a spread candlestick push
func (e *Exchange) wsProcessSpreadCandles(ctx context.Context, respRaw []byte) error {
	var resp WebsocketSpreadCandlesticksPush
	if err := json.Unmarshal(respRaw, &resp); err != nil {
		return err
	}
	if len(resp.Data) == 0 {
		return kline.ErrNoTimeSeriesDataToConvert
	}
	interval, err := candleChannelInterval(resp.Argument.Channel, channelSpreadCandle)
	if err != nil {
		return err
	}
	for i := range resp.Data {
		if err := e.Websocket.DataHandler.Send(ctx, kline.Item{
			Pair:     resp.Argument.SpreadID,
			Asset:    asset.Spread,
			Exchange: e.Name,
			Interval: interval,
			Candles: []kline.Candle{{
				Time:   resp.Data[i].Timestamp.Time(),
				Open:   resp.Data[i].OpenPrice.Float64(),
				High:   resp.Data[i].HighestPrice.Float64(),
				Low:    resp.Data[i].LowestPrice.Float64(),
				Close:  resp.Data[i].ClosePrice.Float64(),
				Volume: resp.Data[i].Volume.Float64(),
			}},
		}); err != nil {
			return err
		}
	}
	return nil
}

// wsProcessInstruments caches the instrument ID codes and pre-open states of
// an instruments push and relays it. The cache updates first, so a relay which
// cannot deliver leaves no stale code for the websocket order operations to
// send.
func (e *Exchange) wsProcessInstruments(ctx context.Context, respRaw []byte) error {
	var resp WebsocketInstrumentsPush
	if err := json.Unmarshal(respRaw, &resp); err != nil {
		return err
	}
	e.cacheInstrumentPush(resp.Data)
	return e.Websocket.DataHandler.Send(ctx, &resp)
}
