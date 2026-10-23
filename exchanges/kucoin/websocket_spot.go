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
	errMalformedCandleTopic     = errors.New("malformed candle topic")
	errUnknownOrderChangeType   = errors.New("unknown order change type")
	errUnknownStopOrderType     = errors.New("unknown stop order type")
	errUnknownBalanceRelation   = errors.New("unknown balance relation event")
	errBalancePushOutOfSequence = errors.New("balance push older than its account's last balance")
	errUnknownMarginTradeType   = errors.New("unknown trade type")
	errEmptyOrderbookLevelOne   = errors.New("level 1 push carries no price")
)

// processTicker merges a Ticker or All Tickers push onto the stored ticker of each asset the pair is enabled on
func (e *Exchange) processTicker(ctx context.Context, data []byte, symbol string) error {
	var resp WebsocketTicker
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
	e.tickerMergeMu.Lock()
	for _, a := range assets {
		if !e.AssetWebsocketSupport.IsAssetWebsocketSupported(a) {
			continue
		}
		t, err := e.mergeTicker(pair, a, apply)
		if err != nil {
			e.tickerMergeMu.Unlock()
			return err
		}
		tickers = append(tickers, *t)
	}
	processed, err := ticker.ProcessBatch(tickers)
	e.tickerMergeMu.Unlock()
	if len(processed) == 0 {
		return err
	}
	return common.AppendError(err, e.Websocket.DataHandler.Send(ctx, processed))
}

// processOrderbookLevel1 loads an Orderbook - Level 1 push as a one level book for each asset the pair is enabled on
func (e *Exchange) processOrderbookLevel1(ctx context.Context, data []byte, symbol string) error {
	var resp WebsocketOrderbookLevel1
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
	var resp WebsocketOrderbookDepth
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
	var resp WebsocketOrderbookIncrement
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
		if err := e.wsOBUpdateMgr.ProcessOrderbookUpdate(ctx, orderbookUpdateID(resp.SequenceStart), &orderbook.Update{
			UpdateID:   orderbookUpdateID(resp.SequenceEnd),
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
	resp := &WebsocketCallAuctionOrderbook{Symbol: symbol}
	if err := json.Unmarshal(data, resp); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, resp)
}

// processCallAuctionData relays a Call Auction Data push
func (e *Exchange) processCallAuctionData(ctx context.Context, data []byte) error {
	return processPayload[WebsocketCallAuctionInfo](ctx, e, data)
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
	var resp WebsocketCandlestick
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
	var resp WebsocketTrade
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
	var resp WebsocketSnapshot
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
	var resp WebsocketSpotOrderChange
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
	// A margin order being placed, whose push came before the reply naming its order ID, is read as margin by its client
	// order ID
	placing := resp.ClientOrderID != "" && e.notePlacingMarginOrder(resp.ClientOrderID, resp.OrderID, resp.Status == "done")
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
		AssetType:       e.spotOrderAsset(pair, resp.OrderID, placing),
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
	if resp.Status == "done" {
		e.marginOrderIDs.Delete(resp.OrderID)
	} else {
		// A margin stop order's pushes here report it past its trigger
		e.marginOrderIDs.CompareAndSwap(resp.OrderID, false, true)
	}
	return e.Websocket.DataHandler.Send(ctx, detail)
}

// spotOrderAsset returns the asset of a spot or margin order push, which shares one topic and does not say which it
// is: margin for a margin order this exchange placed or listed, or is placing, while the pair is enabled for margin,
// otherwise spot when the pair is enabled for spot, else margin when enabled for margin
func (e *Exchange) spotOrderAsset(pair currency.Pair, orderID string, placing bool) asset.Item {
	if _, ok := e.marginOrderIDs.Load(orderID); ok || placing {
		if enabled, err := e.IsPairEnabled(pair, asset.Margin); err == nil && enabled {
			return asset.Margin
		}
	}
	if enabled, err := e.IsPairEnabled(pair, asset.Spot); err == nil && enabled {
		return asset.Spot
	}
	if enabled, err := e.IsPairEnabled(pair, asset.Margin); err == nil && enabled {
		return asset.Margin
	}
	return asset.Spot
}

// marginPlacement is a client order ID's margin orders being placed: how many requests using it are out, and the IDs of
// the orders whose done push came before the reply, which is not to remember them as live
type marginPlacement struct {
	requests     int
	doneOrderIDs map[string]struct{}
}

// rememberMarginOrder places a margin order through place, which returns the order's ID, and remembers it as a live
// margin order, as spot and margin order pushes share one topic. A push can arrive before KuCoin's reply names the
// order's ID, so the client order ID is held as being placed while a request using it is out: a push carrying it is
// read as margin, and a done one is noted, so the reply does not remember a finished order. A margin stop order
// cancelled before the reply stays remembered, as its stop push names no client order ID, which costs only its entry
func (e *Exchange) rememberMarginOrder(clientOrderID string, place func() (orderID string, err error)) error {
	e.marginPlacementsMu.Lock()
	p := e.marginPlacements[clientOrderID]
	if p == nil {
		if e.marginPlacements == nil {
			e.marginPlacements = make(map[string]*marginPlacement)
		}
		p = &marginPlacement{doneOrderIDs: make(map[string]struct{})}
		e.marginPlacements[clientOrderID] = p
	}
	p.requests++
	e.marginPlacementsMu.Unlock()

	orderID, err := place()

	e.marginPlacementsMu.Lock()
	defer e.marginPlacementsMu.Unlock()
	if _, done := p.doneOrderIDs[orderID]; err == nil && orderID != "" && !done {
		e.marginOrderIDs.LoadOrStore(orderID, false)
	}
	p.requests--
	if p.requests == 0 {
		delete(e.marginPlacements, clientOrderID)
	}
	return err
}

// notePlacingMarginOrder reports whether a push's client order ID is being placed as a margin order, and if so
// remembers the order as live, or notes that it is done
func (e *Exchange) notePlacingMarginOrder(clientOrderID, orderID string, done bool) bool {
	e.marginPlacementsMu.Lock()
	defer e.marginPlacementsMu.Unlock()
	p := e.marginPlacements[clientOrderID]
	if p == nil {
		return false
	}
	if done {
		p.doneOrderIDs[orderID] = struct{}{}
	} else if _, finished := p.doneOrderIDs[orderID]; !finished {
		e.marginOrderIDs.Store(orderID, true)
	}
	return true
}

// orderChangeStatus returns the status of an order change push. The message type says whether a done order filled or
// was cancelled, so it is read with the status, and partial fills follow from the filled and remaining sizes. A taker
// order that fills in part opens on the book with what it filled, as REST reports it
func orderChangeStatus(status, messageType string, filled, remaining float64) (order.Status, error) {
	switch messageType {
	case "received":
		return order.New, nil
	case "open":
		if filled > 0 {
			return order.PartiallyFilled, nil
		}
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

// processAccountBalance saves a Balance push of the trade (trade, trade_hf) or main account as that account's share of
// the spot balance, or of a cross (margin, marginV2) or isolated (isolated_<symbol>, isolatedV2_<symbol>) margin
// account as that margin sub-account's balance, and relays it; changes to other accounts, which have no sub-account
// here, are relayed as they come. A margin push carries no liability, so the one saved is kept
func (e *Exchange) processAccountBalance(ctx context.Context, data []byte) error {
	var resp WebsocketAccountBalance
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	a, account, err := balanceAccount(resp.RelationEvent)
	if err != nil {
		return err
	}
	b := accounts.Balance{
		Currency:  resp.Currency,
		Total:     resp.Total.Float64(),
		Hold:      resp.Hold.Float64(),
		Free:      resp.Available.Float64(),
		UpdatedAt: resp.Time.Time(),
	}
	var subAccts accounts.SubAccounts
	switch a {
	case asset.Empty:
		return e.Websocket.DataHandler.Send(ctx, &resp)
	case asset.Spot:
		if subAccts, err = e.saveSpotAccountBalance(ctx, account, &b); err != nil {
			return err
		}
	default: // Margin, the only other asset balanceAccount returns
		if subAccts, err = e.saveMarginBalance(ctx, account, &b); err != nil {
			return err
		}
	}
	return e.Websocket.DataHandler.Send(ctx, subAccts)
}

// saveMarginBalance saves a balance push of a margin sub-account, "" for the cross margin account or an isolated margin
// account's symbol, keeping the liability saved, as the push carries none. The read and the save are one step against
// REST's margin snapshots, so a snapshot saved between them cannot be undone
func (e *Exchange) saveMarginBalance(ctx context.Context, subAccount string, b *accounts.Balance) (accounts.SubAccounts, error) {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return nil, err
	}
	e.marginBalanceMu.Lock()
	defer e.marginBalanceMu.Unlock()
	saved, err := e.Accounts.GetBalance(subAccount, creds, asset.Margin, b.Currency)
	if err != nil && !errors.Is(err, accounts.ErrNoBalances) {
		return nil, err
	}
	// A listing holds the margin accounts as of when it was asked for, so a push older than the listing is out of date
	// for a currency or isolated margin account the listing left out too
	if err != nil && e.marginAccountsListedAt[*creds].After(b.UpdatedAt) {
		return nil, fmt.Errorf("%w: margin %q %s at %s", errBalancePushOutOfSequence, subAccount, b.Currency, b.UpdatedAt)
	}
	b.Borrowed = saved.Borrowed
	subAccts := accounts.SubAccounts{accounts.NewSubAccount(asset.Margin, subAccount)}
	subAccts[0].Balances.Set(b.Currency, *b)
	if err := e.Accounts.Save(ctx, subAccts, false); err != nil {
		return nil, err
	}
	return subAccts, nil
}

// saveSpotAccountBalance saves an account's balance of a currency as its share of the spot balance, which is every
// account's share together as of the latest of their times, and returns the spot balance saved
func (e *Exchange) saveSpotAccountBalance(ctx context.Context, account string, b *accounts.Balance) (accounts.SubAccounts, error) {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return nil, err
	}
	e.spotAccountBalancesMu.Lock()
	defer e.spotAccountBalancesMu.Unlock()
	shares := e.spotAccountBalances[*creds]
	saved, ok := shares[account][b.Currency.Item]
	// A listing holds each account's balances as of when it was asked for, so a push older than the listing is out of
	// date for a currency the listing left out of its account too
	if (ok && saved.UpdatedAt.After(b.UpdatedAt)) || (!ok && e.spotAccountsListedAt[*creds].After(b.UpdatedAt)) {
		return nil, fmt.Errorf("%w: %s %s at %s", errBalancePushOutOfSequence, account, b.Currency, b.UpdatedAt)
	}
	spot := *b
	for other, saved := range shares {
		if share, ok := saved[b.Currency.Item]; ok && other != account {
			spot = spot.Add(share)
		}
	}
	subAccts := accounts.SubAccounts{accounts.NewSubAccount(asset.Spot, "")}
	subAccts[0].Balances.Set(b.Currency, spot)
	if err := e.Accounts.Save(ctx, subAccts, false); err != nil {
		return nil, err
	}
	if shares == nil {
		if e.spotAccountBalances == nil {
			e.spotAccountBalances = make(map[accounts.Credentials]spotAccountShares)
		}
		shares = make(spotAccountShares)
		e.spotAccountBalances[*creds] = shares
	}
	if shares[account] == nil {
		shares[account] = make(map[*currency.Item]accounts.Balance)
	}
	shares[account][b.Currency.Item] = *b
	return subAccts, nil
}

// balanceAccount returns the asset a balance relation event's account is saved under and which account it is: trade or
// main for spot, and for margin "" for the cross margin account or the symbol of an isolated margin account; other
// accounts have no asset here. The account is the event's prefix: trade.hold, isolated_BTC-USDT.hold
func balanceAccount(relationEvent string) (a asset.Item, subAccount string, err error) {
	account, _, ok := strings.Cut(relationEvent, ".")
	switch {
	case account == "trade", account == "trade_hf":
		return asset.Spot, "trade", nil
	case account == "main":
		return asset.Spot, "main", nil
	case account == "margin", account == "marginV2":
		return asset.Margin, "", nil
	case account == "other" && !ok:
		return asset.Empty, "", nil
	}
	for _, prefix := range []string{"isolated_", "isolatedV2_"} {
		// A symbol is required, as the cross margin account is sub-account ""
		if symbol, isolated := strings.CutPrefix(account, prefix); isolated && symbol != "" {
			return asset.Margin, symbol, nil
		}
	}
	return asset.Empty, "", fmt.Errorf("%w: %q", errUnknownBalanceRelation, relationEvent)
}

// processStopOrder relays a spot or margin Stop Order push as an order update
func (e *Exchange) processStopOrder(ctx context.Context, data []byte) error {
	var resp WebsocketStopOrder
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	pair, err := currency.NewPairFromString(resp.Symbol)
	if err != nil {
		return err
	}
	oType, err := stopPushOrderType(resp.OrderType, resp.OrderPrice.Float64())
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
	if status == order.Open {
		// A spot or margin stop order waiting for its trigger is NEW, as Get Stop Order reports it
		status = order.New
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
	if a == asset.Margin {
		// A margin stop order places its order under its own order ID once triggered, which the Order V2 pushes follow
		switch resp.Type {
		case "cancel":
			e.marginOrderIDs.Delete(resp.OrderID)
		case "triggered":
			// From then on it is the limit or market order it placed, as Get Order By OrderId and the Order V2 pushes
			// report it. On its own topic the triggered push can arrive after those pushes or a REST read have reported
			// the order, when it could reopen a finished one, so it is relayed only as the first report of the trigger
			// of an order remembered as waiting for it
			if !e.marginOrderIDs.CompareAndSwap(resp.OrderID, false, true) {
				e.marginOrderIDs.LoadOrStore(resp.OrderID, true)
				return nil
			}
			status = order.Open
			switch oType {
			case order.StopLimit:
				oType = order.Limit
			case order.StopMarket:
				oType = order.Market
			}
		default:
			e.marginOrderIDs.LoadOrStore(resp.OrderID, false)
		}
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

// stopPushOrderType returns the order type of a stop order push. The order type stop, which the docs' examples show, is
// read as a stop limit order when the push carries an order price and as a stop market order when not, matching the
// types REST reports for stop orders; any other order type is read as named
func stopPushOrderType(orderType string, orderPrice float64) (order.Type, error) {
	oType, err := order.StringToOrderType(orderType)
	if err != nil || oType != order.Stop {
		return oType, err
	}
	if orderPrice > 0 {
		return order.StopLimit, nil
	}
	return order.StopMarket, nil
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
