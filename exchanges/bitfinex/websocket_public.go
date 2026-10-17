package bitfinex

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	"hash/crc32"
	"math"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/buger/jsonparser"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const (
	// wsChecksumDepth is how many levels of each side Bitfinex's order book checksums cover
	wsChecksumDepth = 25
	// wsRawBookPrecision asks for a book of orders, or of funding offers, rather than of aggregated levels
	wsRawBookPrecision = "R0"
)

// wsHeartbeat starts the data of a channel's heartbeat, [CHANNEL_ID, "hb", TIME]
var wsHeartbeat = []byte(`"hb"`)

var (
	errChecksumMismatch    = errors.New("order book checksum mismatch")
	errFundingLevelUnknown = errors.New("funding book level unknown")
)

// wsHandleChannelEvent handles an event on a public channel, which the subscription its channel carries describes
func (e *Exchange) wsHandleChannelEvent(ctx context.Context, conn websocket.Connection, channelID int64, respRaw []byte) error {
	data := wsFrameData(respRaw)
	if len(data) == 0 {
		return fmt.Errorf("%w: %.256s", errWebsocketMalformedFrame, respRaw)
	}
	// Bitfinex can send a channel's heartbeat before confirming its subscription
	if bytes.HasPrefix(data, wsHeartbeat) {
		return nil
	}
	s := e.wsChannels.get(conn, channelID)
	if s == nil {
		// A channel draining until Bitfinex unsubscribes it carries nothing wanted
		if e.wsChannels.isDraining(conn, channelID) {
			return nil
		}
		return fmt.Errorf("%w: %d", errWebsocketChannelUnknown, channelID)
	}
	if len(s.Pairs) != 1 {
		return fmt.Errorf("%w: %s", errSubscriptionNotSinglePair, s)
	}
	if data[0] == '"' {
		eventType, _, _, err := jsonparser.Get(data)
		if err != nil {
			return fmt.Errorf("%w: %w: %.256s", errWebsocketMalformedFrame, err, respRaw)
		}
		switch string(eventType) {
		case "cs":
			return e.wsHandleChecksum(ctx, conn, s, respRaw)
		case "te", "fte":
			return e.wsHandleTradeExecuted(s, respRaw)
		case "tu", "ftu":
			// An execution update repeats the trade its execution event already sent
			return nil
		}
		return fmt.Errorf("%w: event type %q: %.256s", errWebsocketMalformedFrame, eventType, respRaw)
	}
	switch s.Channel {
	case subscription.TickerChannel:
		return e.wsHandleTicker(ctx, s, respRaw)
	case subscription.AllTradesChannel:
		return e.wsHandleTradesSnapshot(s, data, respRaw)
	case subscription.OrderbookChannel:
		return e.wsHandleBook(ctx, conn, s, data, respRaw)
	case subscription.CandlesChannel:
		return e.wsHandleCandles(ctx, s, data, respRaw)
	case wsStatusChannel:
		return e.wsHandleDerivativeStatus(ctx, s, respRaw)
	}
	return fmt.Errorf("%w: %s", errWebsocketChannelUnknown, s.Channel)
}

// wsHandleDerivativeStatus relays a derivative pair's status, and its funding rate as funding data
func (e *Exchange) wsHandleDerivativeStatus(ctx context.Context, s *subscription.Subscription, respRaw []byte) error {
	var f wsDataFrame[DerivativeStatus]
	if err := f.UnmarshalJSON(respRaw); err != nil {
		return err
	}
	if err := e.Websocket.DataHandler.Send(ctx, websocket.FundingData{
		Timestamp:    f.Data.Time.Time(),
		CurrencyPair: s.Pairs[0],
		AssetType:    s.Asset,
		Exchange:     e.Name,
		Rate:         f.Data.CurrentFunding,
	}); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, &WebsocketDerivativeStatus{Pair: s.Pairs[0], AssetType: s.Asset, Status: f.Data})
}

// wsFrameData returns a channel event from its data on, after the channel ID it starts with
func wsFrameData(respRaw []byte) []byte {
	for i := range respRaw {
		if respRaw[i] == ',' {
			return respRaw[skipSpace(respRaw, i+1):]
		}
	}
	return nil
}

// isSnapshot reports whether a channel event's data is a snapshot, an array of records, rather than one record
func isSnapshot(data []byte) bool {
	if len(data) == 0 || data[0] != '[' {
		return false
	}
	i := skipSpace(data, 1)
	return i < len(data) && (data[i] == '[' || data[i] == ']')
}

// wsHandleTicker stores and relays a trading pair's or funding currency's ticker, as of the time Bitfinex sent it
func (e *Exchange) wsHandleTicker(ctx context.Context, s *subscription.Subscription, respRaw []byte) error {
	var t *ticker.Price
	if s.Asset == asset.MarginFunding {
		var f wsDataFrame[FundingTicker]
		if err := f.UnmarshalJSON(respRaw); err != nil {
			return err
		}
		t = e.fundingTickerPrice(&f.Data, s.Pairs[0], s.Asset)
		t.LastUpdated = f.Time.Time()
	} else {
		var f wsDataFrame[Ticker]
		if err := f.UnmarshalJSON(respRaw); err != nil {
			return err
		}
		t = e.tickerPrice(&f.Data, s.Pairs[0], s.Asset)
		t.LastUpdated = f.Time.Time()
	}
	err := e.wsStoreTicker(ctx, t)
	if !e.wsSharedChannels.carriesMargin(s) {
		return err
	}
	m := *t
	m.AssetType = asset.Margin
	return common.AppendError(err, e.wsStoreTicker(ctx, &m))
}

// wsStoreTicker stores a ticker, then relays it
func (e *Exchange) wsStoreTicker(ctx context.Context, t *ticker.Price) error {
	if err := ticker.ProcessTicker(t); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, t)
}

// wsHandleTradesSnapshot relays the trades Bitfinex sends on subscribing, its most recent
func (e *Exchange) wsHandleTradesSnapshot(s *subscription.Subscription, data, respRaw []byte) error {
	if !e.IsTradeFeedEnabled() && !e.IsSaveTradeDataEnabled() {
		return nil
	}
	if !isSnapshot(data) {
		return fmt.Errorf("%w: trades event is not a snapshot: %.256s", errWebsocketMalformedFrame, respRaw)
	}
	if s.Asset == asset.MarginFunding {
		var f wsDataFrame[[]PublicFundingTrade]
		if err := f.UnmarshalJSON(respRaw); err != nil {
			return err
		}
		trades := make([]trade.Data, len(f.Data))
		for i := range f.Data {
			trades[i] = e.fundingTradeData(s.Pairs[0], s.Asset, &f.Data[i])
		}
		return e.Websocket.Trade.Update(e.IsSaveTradeDataEnabled(), trades...)
	}
	var f wsDataFrame[[]PublicTrade]
	if err := f.UnmarshalJSON(respRaw); err != nil {
		return err
	}
	trades := make([]trade.Data, len(f.Data))
	for i := range f.Data {
		trades[i] = e.tradeData(s.Pairs[0], s.Asset, &f.Data[i])
	}
	return e.Websocket.Trade.Update(e.IsSaveTradeDataEnabled(), e.wsSharedTrades(s, trades)...)
}

// wsHandleTradeExecuted relays a trade as it executes; its execution update, which follows, is ignored
func (e *Exchange) wsHandleTradeExecuted(s *subscription.Subscription, respRaw []byte) error {
	if s.Channel != subscription.AllTradesChannel {
		return fmt.Errorf("%w: trade event on %s: %.256s", errWebsocketMalformedFrame, s.Channel, respRaw)
	}
	if !e.IsTradeFeedEnabled() && !e.IsSaveTradeDataEnabled() {
		return nil
	}
	if s.Asset == asset.MarginFunding {
		var f wsTypedFrame[PublicFundingTrade]
		if err := f.UnmarshalJSON(respRaw); err != nil {
			return err
		}
		return e.Websocket.Trade.Update(e.IsSaveTradeDataEnabled(), e.fundingTradeData(s.Pairs[0], s.Asset, &f.Data))
	}
	var f wsTypedFrame[PublicTrade]
	if err := f.UnmarshalJSON(respRaw); err != nil {
		return err
	}
	return e.Websocket.Trade.Update(e.IsSaveTradeDataEnabled(), e.wsSharedTrades(s, []trade.Data{e.tradeData(s.Pairs[0], s.Asset, &f.Data)})...)
}

// wsSharedTrades returns a channel's trades, with each repeated for margin when the channel carries margin's data too
func (e *Exchange) wsSharedTrades(s *subscription.Subscription, trades []trade.Data) []trade.Data {
	if !e.wsSharedChannels.carriesMargin(s) {
		return trades
	}
	for i, n := 0, len(trades); i < n; i++ {
		t := trades[i]
		t.AssetType = asset.Margin
		trades = append(trades, t)
	}
	return trades
}

// tradeData returns a trading pair's trade as GCT records it: a positive amount bought, a negative one sold
func (e *Exchange) tradeData(p currency.Pair, a asset.Item, t *PublicTrade) trade.Data {
	side := order.Buy
	if t.Amount < 0 {
		side = order.Sell
	}
	return trade.Data{
		Exchange:     e.Name,
		TID:          strconv.FormatUint(t.ID, 10),
		CurrencyPair: p,
		AssetType:    a,
		Side:         side,
		Price:        t.Price,
		Amount:       math.Abs(t.Amount),
		Timestamp:    t.Time.Time(),
	}
}

// fundingTradeData returns a funding currency's trade as GCT records it, its rate as the price
func (e *Exchange) fundingTradeData(p currency.Pair, a asset.Item, t *PublicFundingTrade) trade.Data {
	side := order.Buy
	if t.Amount < 0 {
		side = order.Sell
	}
	return trade.Data{
		Exchange:     e.Name,
		TID:          strconv.FormatUint(t.ID, 10),
		CurrencyPair: p,
		AssetType:    a,
		Side:         side,
		Price:        t.Rate,
		Amount:       math.Abs(t.Amount),
		Timestamp:    t.Time.Time(),
	}
}

// wsHandleCandles relays a snapshot of candles, oldest first, or the update of one
func (e *Exchange) wsHandleCandles(ctx context.Context, s *subscription.Subscription, data, respRaw []byte) error {
	var candles []Candle
	if isSnapshot(data) {
		var f wsDataFrame[[]Candle]
		if err := f.UnmarshalJSON(respRaw); err != nil {
			return err
		}
		candles = f.Data
	} else {
		var f wsDataFrame[Candle]
		if err := f.UnmarshalJSON(respRaw); err != nil {
			return err
		}
		candles = []Candle{f.Data}
	}
	if len(candles) == 0 {
		return nil
	}
	item := kline.Item{
		Exchange: e.Name,
		Pair:     s.Pairs[0],
		Asset:    s.Asset,
		Interval: s.Interval,
		Candles:  make([]kline.Candle, len(candles)),
	}
	// Bitfinex sends a snapshot newest first
	for i := range candles {
		c := &candles[len(candles)-1-i]
		item.Candles[i] = kline.Candle{Time: c.Time.Time(), Open: c.Open, High: c.High, Low: c.Low, Close: c.Close, Volume: c.Volume}
	}
	err := e.Websocket.DataHandler.Send(ctx, item)
	if !e.wsSharedChannels.carriesMargin(s) {
		return err
	}
	item.Asset = asset.Margin
	item.Candles = slices.Clone(item.Candles)
	return common.AppendError(err, e.Websocket.DataHandler.Send(ctx, item))
}

// isRawBook reports whether a book subscription asks for a book of orders, or of funding offers, rather than of
// aggregated levels
func isRawBook(s *subscription.Subscription) bool {
	precision, _ := s.Params["prec"].(string)
	return strings.EqualFold(precision, wsRawBookPrecision)
}

// wsBookLevel converts an entry of a book Bitfinex sends into a level, and tells which side holds it and whether the
// entry removes it
type wsBookLevel interface {
	level() (l orderbook.Level, bid, remove bool)
}

// level returns the order as a level; an order leaving the book is priced zero, and its amount's sign gives its side
func (b *RawBookLevel) level() (l orderbook.Level, bid, remove bool) {
	return orderbook.Level{ID: int64(b.OrderID), Price: b.Price, Amount: math.Abs(b.Amount)}, b.Amount > 0, b.Price == 0 //nolint:gosec // Bitfinex numbers orders well within int64
}

// level returns the price level; a level leaving the book counts no orders, and its amount's sign gives its side
func (b *BookLevel) level() (l orderbook.Level, bid, remove bool) {
	return orderbook.Level{Price: b.Price, Amount: math.Abs(b.Amount), OrderCount: int64(b.Count)}, b.Amount > 0, b.Count == 0 //nolint:gosec // A level counts few orders
}

// level returns the offer as a level; an offer leaving the book is rated zero, and its amount's sign gives its side, a
// negative amount bidding for funding
func (b *RawFundingBookLevel) level() (l orderbook.Level, bid, remove bool) {
	return orderbook.Level{ID: int64(b.OfferID), Price: b.Rate, Period: int64(b.Period), Amount: math.Abs(b.Amount)}, b.Amount < 0, b.Rate == 0 //nolint:gosec // Bitfinex numbers offers well within int64, and periods are days
}

// level returns the rate and period level; a level leaving the book counts no offers, and its amount's sign gives its
// side, a negative amount bidding for funding
func (b *FundingBookLevel) level() (l orderbook.Level, bid, remove bool) {
	return orderbook.Level{Price: b.Rate, Period: int64(b.Period), Amount: math.Abs(b.Amount), OrderCount: int64(b.Count)}, b.Amount < 0, b.Count == 0 //nolint:gosec // Periods are days, and a level counts few offers
}

// wsHandleBook loads a book's snapshot or applies one of its updates, decoding the entries of the book the subscription
// asked for
func (e *Exchange) wsHandleBook(ctx context.Context, conn websocket.Connection, s *subscription.Subscription, data, respRaw []byte) error {
	raw := isRawBook(s)
	switch {
	case s.Asset == asset.MarginFunding && raw:
		return wsProcessBook[RawFundingBookLevel](ctx, e, conn, s, data, respRaw)
	case s.Asset == asset.MarginFunding:
		return wsProcessBook[FundingBookLevel](ctx, e, conn, s, data, respRaw)
	case raw:
		return wsProcessBook[RawBookLevel](ctx, e, conn, s, data, respRaw)
	}
	return wsProcessBook[BookLevel](ctx, e, conn, s, data, respRaw)
}

// wsProcessBook loads a book's snapshot or applies one of its updates, to margin's book as well as spot's when the
// channel carries both. A raw book's orders or offers are matched by their IDs. A trading pair's aggregated levels are
// matched by price, and a funding currency's by rate and period, which the levels' IDs stand for
func wsProcessBook[T any, PT interface {
	*T
	wsBookLevel
	UnmarshalJSON([]byte) error
}](ctx context.Context, e *Exchange, conn websocket.Connection, s *subscription.Subscription, data, respRaw []byte) error {
	p, a := s.Pairs[0], s.Asset
	bookKey := key.PairAsset{Base: p.Base.Item, Quote: p.Quote.Item, Asset: a}
	raw := isRawBook(s)
	fundingLevels := a == asset.MarginFunding && !raw
	margin := e.wsSharedChannels.carriesMargin(s)
	if isSnapshot(data) {
		var f wsDataFrame[[]T]
		if err := f.UnmarshalJSON(respRaw); err != nil {
			return err
		}
		book := &orderbook.Book{
			Exchange: e.Name,
			Pair:     p,
			Asset:    a,
			// A raw book holds every order at a price, and a funding book a level per period at a rate
			PriceDuplication:  raw || a == asset.MarginFunding,
			IsFundingRate:     a == asset.MarginFunding,
			ValidateOrderbook: e.ValidateOrderbook,
			LastUpdated:       wsEventTime(f.Time),
			LastPushed:        wsEventTime(f.Time),
		}
		for i := range f.Data {
			l, bid, _ := PT(&f.Data[i]).level()
			if bid {
				book.Bids = append(book.Bids, l)
			} else {
				book.Asks = append(book.Asks, l)
			}
		}
		if fundingLevels {
			e.wsFundingBookIDs.load(bookKey, book.Bids, book.Asks)
		}
		err := e.Websocket.Orderbook.LoadSnapshot(ctx, book)
		if margin {
			// Loading a book copies its levels, so margin's shares spot's
			m := *book
			m.Asset = asset.Margin
			err = common.AppendError(err, e.Websocket.Orderbook.LoadSnapshot(ctx, &m))
		}
		// A book loaded whose relay failed is fresh, and one that failed to load is still invalid, which its next
		// checksum renews it for
		e.wsBookResync.finish(bookKey)
		return err
	}
	var f wsDataFrame[T]
	if err := f.UnmarshalJSON(respRaw); err != nil {
		return err
	}
	l, bid, remove := PT(&f.Data).level()
	u := &orderbook.Update{
		Pair:       p,
		Asset:      a,
		UpdateTime: wsEventTime(f.Time),
		LastPushed: wsEventTime(f.Time),
		Action:     orderbook.UpdateOrInsertAction,
	}
	switch {
	case fundingLevels:
		id, err := e.wsFundingBookIDs.id(bookKey, l.Price, l.Period, remove)
		if err != nil {
			return fmt.Errorf("%w: %s %s rate %v period %d", err, a, p, l.Price, l.Period)
		}
		l.ID = id
		if remove {
			u.Action = orderbook.DeleteAction
		}
	case raw:
		if remove {
			u.Action = orderbook.DeleteAction
		}
	default:
		// A trading pair's aggregated levels are matched by price, and a level without orders is removed
		u.Action = orderbook.UnknownAction
		if remove {
			l.Amount = 0
		}
	}
	if bid {
		u.Bids = orderbook.Levels{l}
	} else {
		u.Asks = orderbook.Levels{l}
	}
	err := e.Websocket.Orderbook.Update(ctx, u)
	unaffected := asset.Margin
	// A book whose update was applied but not relayed is current, so margin's book takes the update too
	if margin && !errors.Is(err, orderbook.ErrOrderbookInvalid) {
		m := *u
		m.Asset = asset.Margin
		m.Bids, m.Asks = slices.Clone(u.Bids), slices.Clone(u.Asks)
		marginErr := e.Websocket.Orderbook.Update(ctx, &m)
		// A pair enabled for margin after its channel's snapshot has no margin book until the channel's next snapshot
		if errors.Is(marginErr, orderbook.ErrDepthNotFound) {
			marginErr = fmt.Errorf("%w: %w", orderbook.ErrOrderbookInvalid, marginErr)
		}
		if errors.Is(marginErr, orderbook.ErrOrderbookInvalid) {
			unaffected = a
		}
		err = common.AppendError(err, marginErr)
	}
	if err != nil {
		if !errors.Is(err, orderbook.ErrOrderbookInvalid) {
			return err
		}
		// An update a book cannot take invalidates it; renewing its channel sends a fresh snapshot, which the channel's
		// other book awaits too, and the updates before it are dropped
		resync, ok := e.wsBookResync.start(bookKey)
		if !ok {
			return nil
		}
		if margin {
			_ = e.Websocket.Orderbook.InvalidateOrderbook(p, unaffected)
		}
		e.resubscribeBook(ctx, conn, s, bookKey, resync)
		return err
	}
	return nil
}

// wsEventTime returns when Bitfinex sent an event, or now for an event without its time
func wsEventTime(t types.Time) time.Time {
	if sent := t.Time(); !sent.IsZero() {
		return sent
	}
	return time.Now()
}

// wsHandleChecksum checks the books a channel carries against the checksum Bitfinex sends of its top levels, after
// the updates before it. Books that no longer match, or were left invalid, such as by a renewal that gave up, are
// invalidated and their channel renewed, which sends a fresh snapshot
func (e *Exchange) wsHandleChecksum(ctx context.Context, conn websocket.Connection, s *subscription.Subscription, respRaw []byte) error {
	if s.Channel != subscription.OrderbookChannel {
		return fmt.Errorf("%w: checksum on %s: %.256s", errWebsocketMalformedFrame, s.Channel, respRaw)
	}
	var f wsTypedFrame[int64]
	if err := f.UnmarshalJSON(respRaw); err != nil {
		return err
	}
	p, a := s.Pairs[0], s.Asset
	bookKey := key.PairAsset{Base: p.Base.Item, Quote: p.Quote.Item, Asset: a}
	if e.wsBookResync.pending(bookKey) {
		return nil
	}
	// Bitfinex sends the checksum as a signed 32 bit integer
	want := uint32(f.Data) //nolint:gosec // The checksum's bits are kept, whatever its sign
	raw := isRawBook(s)
	err := e.wsCheckBook(p, a, raw, want)
	margin := e.wsSharedChannels.carriesMargin(s)
	if err == nil && margin {
		// A pair enabled for margin after its channel's snapshot has no margin book until the channel's next snapshot
		if err = e.wsCheckBook(p, asset.Margin, raw, want); errors.Is(err, orderbook.ErrDepthNotFound) {
			err = fmt.Errorf("%w: %w", orderbook.ErrOrderbookInvalid, err)
		}
	}
	if !errors.Is(err, errChecksumMismatch) && !errors.Is(err, orderbook.ErrOrderbookInvalid) {
		return err
	}
	if resync, ok := e.wsBookResync.start(bookKey); ok {
		// A book not loaded has nothing to invalidate
		_ = e.Websocket.Orderbook.InvalidateOrderbook(p, a)
		if margin {
			_ = e.Websocket.Orderbook.InvalidateOrderbook(p, asset.Margin)
		}
		e.resubscribeBook(ctx, conn, s, bookKey, resync)
	}
	return err
}

// wsCheckBook checks a book against the checksum Bitfinex sent of it
func (e *Exchange) wsCheckBook(p currency.Pair, a asset.Item, raw bool, want uint32) error {
	book, err := e.Websocket.Orderbook.GetOrderbook(p, a)
	if err != nil {
		return err
	}
	if got := bookChecksum(book, raw); got != want {
		return fmt.Errorf("%w: %s %s: Bitfinex sent %d, the book gives %d", errChecksumMismatch, a, p, want, got)
	}
	return nil
}

// bookChecksum returns the CRC32 Bitfinex checks a book with: the top levels of each side in turn, a bid then an ask,
// each as its ID, or a level's price or rate, and its signed amount, joined with colons. A raw book's orders sharing a
// price are taken by ascending ID; a funding currency's levels sharing a rate stay in the order Bitfinex sent them
func bookChecksum(b *orderbook.Book, raw bool) uint32 {
	bidSign, askSign := 1.0, -1.0
	if b.IsFundingRate {
		bidSign, askSign = -1, 1
	}
	if raw {
		slices.SortStableFunc(b.Bids, func(x, y orderbook.Level) int {
			return cmp.Or(cmp.Compare(y.Price, x.Price), cmp.Compare(x.ID, y.ID))
		})
		slices.SortStableFunc(b.Asks, func(x, y orderbook.Level) int {
			return cmp.Or(cmp.Compare(x.Price, y.Price), cmp.Compare(x.ID, y.ID))
		})
	}
	buf := make([]byte, 0, 4*wsChecksumDepth*24)
	appendLevel := func(l *orderbook.Level, sign float64) {
		if len(buf) > 0 {
			buf = append(buf, ':')
		}
		if raw {
			buf = strconv.AppendInt(buf, l.ID, 10)
		} else {
			buf = appendJSNumber(buf, l.Price)
		}
		buf = append(buf, ':')
		buf = appendJSNumber(buf, sign*l.Amount)
	}
	for i := range wsChecksumDepth {
		if i < len(b.Bids) {
			appendLevel(&b.Bids[i], bidSign)
		}
		if i < len(b.Asks) {
			appendLevel(&b.Asks[i], askSign)
		}
	}
	return crc32.ChecksumIEEE(buf)
}

// appendJSNumber appends a number as JavaScript writes it, as Bitfinex builds its checksums in JavaScript: the shortest
// digits that read back as the number, in exponent form below 1e-6 and from 1e21, such as 1e-7 or 1.5e+21
func appendJSNumber(buf []byte, f float64) []byte {
	if f == 0 {
		return append(buf, '0')
	}
	if abs := math.Abs(f); abs >= 1e-6 && abs < 1e21 {
		return strconv.AppendFloat(buf, f, 'f', -1, 64)
	}
	start := len(buf)
	buf = strconv.AppendFloat(buf, f, 'e', -1, 64)
	// Go writes the exponent with at least two digits, JavaScript with as few as it needs
	exponent := slices.Index(buf[start:], 'e') + start + 2
	if buf[exponent] == '0' {
		buf = append(buf[:exponent], buf[exponent+1:]...)
	}
	return buf
}

// fundingLevelKey identifies a level of a funding currency's aggregated book
type fundingLevelKey struct {
	rate   float64
	period int64
}

// fundingLevelIDs numbers a funding currency's aggregated book's levels, which Bitfinex identifies by rate and period
// together, so that each level's updates find it among those sharing its rate
type fundingLevelIDs struct {
	ids  map[fundingLevelKey]int64
	next int64
}

// wsFundingBookIDs holds the level IDs of each funding currency's aggregated book
type wsFundingBookIDs struct {
	mu    sync.Mutex
	books map[key.PairAsset]*fundingLevelIDs
}

// load numbers the levels of a book's snapshot afresh, setting their IDs
func (f *wsFundingBookIDs) load(k key.PairAsset, sides ...orderbook.Levels) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.books == nil {
		f.books = make(map[key.PairAsset]*fundingLevelIDs)
	}
	book := &fundingLevelIDs{ids: make(map[fundingLevelKey]int64)}
	f.books[k] = book
	for _, levels := range sides {
		for i := range levels {
			book.next++
			book.ids[fundingLevelKey{rate: levels[i].Price, period: levels[i].Period}] = book.next
			levels[i].ID = book.next
		}
	}
}

// id returns a level's ID, numbering a level the book has not held, or forgetting a level being removed
func (f *wsFundingBookIDs) id(k key.PairAsset, rate float64, period int64, remove bool) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	book, ok := f.books[k]
	if !ok {
		return 0, fmt.Errorf("%w: no snapshot", orderbook.ErrDepthNotFound)
	}
	levelKey := fundingLevelKey{rate: rate, period: period}
	id, ok := book.ids[levelKey]
	switch {
	case remove && !ok:
		return 0, errFundingLevelUnknown
	case remove:
		delete(book.ids, levelKey)
		return id, nil
	case ok:
		return id, nil
	}
	book.next++
	book.ids[levelKey] = book.next
	return book.next, nil
}

// wsBookResync holds the books being resubscribed, from the update or checksum they failed until their fresh snapshot
// loads, so that each is resubscribed once and the updates in between are dropped. Each resync is numbered, so that a
// renewal outlasting its own resync leaves a later one alone
type wsBookResync struct {
	mu    sync.Mutex
	books map[key.PairAsset]uint64
	last  uint64
}

// start reports whether a book's resync should start, as none is pending, numbering it
func (r *wsBookResync) start(k key.PairAsset) (uint64, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.books[k]; ok {
		return 0, false
	}
	if r.books == nil {
		r.books = make(map[key.PairAsset]uint64)
	}
	r.last++
	r.books[k] = r.last
	return r.last, true
}

// active reports whether a book's pending resync is the one numbered n
func (r *wsBookResync) active(k key.PairAsset, n uint64) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	current, ok := r.books[k]
	return ok && current == n
}

// abandon ends a book's resync numbered n, its renewal having given up, leaving any later one
func (r *wsBookResync) abandon(k key.PairAsset, n uint64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if current, ok := r.books[k]; ok && current == n {
		delete(r.books, k)
	}
}

// pending reports whether a book is awaiting its fresh snapshot
func (r *wsBookResync) pending(k key.PairAsset) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.books[k]
	return ok
}

// finish forgets a book's resubscription once its fresh snapshot loads
func (r *wsBookResync) finish(k key.PairAsset) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.books, k)
}
