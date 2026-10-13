package kucoin

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

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
	errMalformedOrderbookChange = errors.New("malformed orderbook change")
	errUnknownMarginMode        = errors.New("unknown margin mode")
)

// processFuturesTickerV2 merges a Ticker V2 push onto the stored ticker. Ticker V2 carries only the best bid and ask,
// and the ticker's other fields come from its companion topics
func (e *Exchange) processFuturesTickerV2(ctx context.Context, data []byte) error {
	var resp WsFuturesTickerV2
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	pair, err := e.futuresPair(resp.Symbol)
	if err != nil {
		return err
	}
	return e.processMergedTicker(ctx, pair, asset.Futures, func(t *ticker.Price) {
		t.Bid = resp.BestBidPrice.Float64()
		t.BidSize = float64(resp.BestBidSize)
		t.Ask = resp.BestAskPrice.Float64()
		t.AskSize = float64(resp.BestAskSize)
		t.LastUpdated = resp.Timestamp.Time()
	})
}

// processFuturesTickerV1 merges the last fill of a Ticker V1 push onto the stored ticker. Its best bid and ask are
// those at the fill and often older than Ticker V2's, so they are left to Ticker V2
func (e *Exchange) processFuturesTickerV1(ctx context.Context, data []byte) error {
	var resp WsFuturesTickerV1
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	pair, err := e.futuresPair(resp.Symbol)
	if err != nil {
		return err
	}
	return e.processMergedTicker(ctx, pair, asset.Futures, func(t *ticker.Price) {
		t.Last = resp.Price.Float64()
		t.LastSize = float64(resp.Size)
		t.LastUpdated = resp.Timestamp.Time()
	})
}

// processFuturesSnapshot merges the 24 hour statistics of a Symbol Snapshot push onto the stored ticker. Its last price
// is up to 5 seconds old, so the last price is left to Ticker V1
func (e *Exchange) processFuturesSnapshot(ctx context.Context, data []byte) error {
	var resp WsFuturesSnapshot
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	pair, err := e.futuresPair(resp.Symbol)
	if err != nil {
		return err
	}
	baseVolume, quoteVolume := resp.Volume, resp.Turnover
	if isInverseContract(pair) {
		// An inverse contract is worth one unit of its quote currency, so its volume is counted in the quote currency
		baseVolume, quoteVolume = quoteVolume, baseVolume
	}
	return e.processMergedTicker(ctx, pair, asset.Futures, func(t *ticker.Price) {
		t.High = resp.HighPrice
		t.Low = resp.LowPrice
		t.Open24Hour = resp.Price24HoursBefore
		t.BaseVolume = baseVolume
		t.QuoteVolume = quoteVolume
		t.LastUpdated = resp.Timestamp.Time()
	})
}

// isInverseContract returns whether a futures pair is an inverse contract; KuCoin's linear contracts are quoted in
// USDTM or USDCM and its inverse ones in USDM or by delivery month
func isInverseContract(pair currency.Pair) bool {
	return !pair.Quote.Equal(currency.USDTM) && !pair.Quote.Equal(currency.USDCM)
}

// processFuturesOrderbookSnapshot loads a futures Orderbook - Level 5 or Level 50 push
func (e *Exchange) processFuturesOrderbookSnapshot(ctx context.Context, data []byte, symbol string) error {
	var resp WsFuturesOrderbookDepth
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	pair, err := e.futuresPair(symbol)
	if err != nil {
		return err
	}
	return e.Websocket.Orderbook.LoadSnapshot(ctx, &orderbook.Book{
		Exchange:     e.Name,
		LastUpdateID: resp.Sequence,
		LastUpdated:  resp.Timestamp.Time(),
		LastPushed:   resp.PushTimestamp.Time(),
		Asset:        asset.Futures,
		Bids:         mergeRoundedOrderbookLevels(resp.Bids.Levels()),
		Asks:         mergeRoundedOrderbookLevels(resp.Asks.Levels()),
		Pair:         pair,
	})
}

// processFuturesOrderbookIncrement applies a futures Orderbook - Increment push
func (e *Exchange) processFuturesOrderbookIncrement(ctx context.Context, data []byte, symbol string) error {
	pair, err := e.futuresPair(symbol)
	if err != nil {
		return err
	}
	var resp WsFuturesOrderbookIncrement
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	parts := strings.Split(resp.Change, ",")
	if len(parts) != 3 {
		return fmt.Errorf("%w: %q", errMalformedOrderbookChange, resp.Change)
	}
	price, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return fmt.Errorf("%w: price %q: %w", errMalformedOrderbookChange, resp.Change, err)
	}
	amount, err := strconv.ParseFloat(parts[2], 64)
	if err != nil {
		return fmt.Errorf("%w: size %q: %w", errMalformedOrderbookChange, resp.Change, err)
	}
	var bids, asks []orderbook.Level
	switch parts[1] {
	case "buy":
		bids = []orderbook.Level{{Price: price, Amount: amount, ID: resp.Sequence}}
	case "sell":
		asks = []orderbook.Level{{Price: price, Amount: amount, ID: resp.Sequence}}
	default:
		return fmt.Errorf("%w: side %q", errMalformedOrderbookChange, resp.Change)
	}
	return e.wsOBUpdateMgr.ProcessOrderbookUpdate(ctx, resp.Sequence, &orderbook.Update{
		UpdateTime: resp.Timestamp.Time(),
		LastPushed: resp.Timestamp.Time(),
		UpdateID:   resp.Sequence,
		Pair:       pair,
		Asset:      asset.Futures,
		Asks:       asks,
		Bids:       bids,
	})
}

// processFuturesKline relays a futures Klines push as a candle. Live candles carry the turnover where the docs place
// the volume, which they call incorrect, and the traded contracts where they place the amount; the volume relayed is
// the contracts, as the REST klines report it
func (e *Exchange) processFuturesKline(ctx context.Context, data []byte, symbolAndInterval string) error {
	symbol, intervalString, ok := strings.Cut(symbolAndInterval, currency.UnderscoreDelimiter)
	if !ok {
		return fmt.Errorf("%w: %q", errMalformedCandleTopic, symbolAndInterval)
	}
	interval, err := IntervalFromString(intervalString)
	if err != nil {
		return err
	}
	pair, err := e.futuresPair(symbol)
	if err != nil {
		return err
	}
	var resp WsCandlestick
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, &kline.Item{
		Asset:    asset.Futures,
		Exchange: e.Name,
		Pair:     pair,
		Interval: interval,
		Candles: []kline.Candle{{
			Time:   resp.Candles.StartTime.Time(),
			Open:   resp.Candles.OpenPrice.Float64(),
			Close:  resp.Candles.ClosePrice.Float64(),
			High:   resp.Candles.HighPrice.Float64(),
			Low:    resp.Candles.LowPrice.Float64(),
			Volume: resp.Candles.TransactionAmount.Float64(),
		}},
	})
}

// processFuturesExecution saves a futures Trade push
func (e *Exchange) processFuturesExecution(data []byte) error {
	var resp WsFuturesExecution
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	saveTradeData := e.IsSaveTradeDataEnabled()
	if !saveTradeData && !e.IsTradeFeedEnabled() {
		return nil
	}
	pair, err := e.futuresPair(resp.Symbol)
	if err != nil {
		return err
	}
	side, err := order.StringToOrderSide(resp.Side)
	if err != nil {
		return err
	}
	return e.Websocket.Trade.Update(saveTradeData, trade.Data{
		CurrencyPair: pair,
		Timestamp:    resp.Timestamp.Time(),
		Price:        resp.Price.Float64(),
		Amount:       float64(resp.Size),
		Side:         side,
		Exchange:     e.Name,
		TID:          resp.TradeID,
		AssetType:    asset.Futures,
	})
}

// processFuturesInstrument relays an Instrument push, naming the symbol from the topic as the push does not carry it
func (e *Exchange) processFuturesInstrument(ctx context.Context, msg *WsMessage, symbol string) error {
	var resp any
	switch msg.Subject {
	case "mark.index.price":
		resp = &WsFuturesMarkIndexPrice{Symbol: symbol}
	case "funding.rate":
		resp = &WsFuturesFundingRate{Symbol: symbol}
	default:
		return unhandledSubject(msg)
	}
	if err := json.Unmarshal(msg.Data, resp); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, resp)
}

// processFuturesAnnouncement relays a Funding Fee Settlement push
func (e *Exchange) processFuturesAnnouncement(ctx context.Context, msg *WsMessage) error {
	switch msg.Subject {
	case "funding.begin":
		return processPayload[WsFuturesFundingBegin](ctx, e, msg.Data)
	case "funding.end":
		return processPayload[WsFuturesFundingEnd](ctx, e, msg.Data)
	}
	return unhandledSubject(msg)
}

// processFuturesOrderChange relays a futures Orders push as an order update
func (e *Exchange) processFuturesOrderChange(ctx context.Context, data []byte) error {
	var resp WsFuturesOrderChange
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	pair, err := e.futuresPair(resp.Symbol)
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
	marginType, err := futuresMarginType(resp.MarginMode)
	if err != nil {
		return err
	}
	detail := &order.Detail{
		Price:           resp.Price.Float64(),
		Amount:          resp.Size.Float64(),
		ExecutedAmount:  resp.FilledSize.Float64(),
		RemainingAmount: resp.RemainSize.Float64(),
		Exchange:        e.Name,
		OrderID:         resp.OrderID,
		ClientOrderID:   resp.ClientOrderID,
		Type:            oType,
		Side:            side,
		Status:          status,
		AssetType:       asset.Futures,
		MarginType:      marginType,
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

// futuresMarginType returns the margin type of a futures margin mode
func futuresMarginType(marginMode string) (margin.Type, error) {
	switch marginMode {
	case "ISOLATED":
		return margin.Isolated, nil
	case "CROSS":
		return margin.Multi, nil
	case "":
		return margin.Unset, nil
	}
	return margin.Unknown, fmt.Errorf("%w: %q", errUnknownMarginMode, marginMode)
}

// processFuturesWallet saves the balance of a futures Balance push and relays it. The order margin and withdrawal hold
// subjects, deprecated once an account first switches margin mode, are relayed as they come
func (e *Exchange) processFuturesWallet(ctx context.Context, msg *WsMessage) error {
	switch msg.Subject {
	case "walletBalance.change":
		var resp WsFuturesWalletBalance
		if err := json.Unmarshal(msg.Data, &resp); err != nil {
			return err
		}
		return e.saveFuturesBalance(ctx, resp.Currency, resp.AvailableBalance.Float64(), resp.HoldBalance.Float64(), resp.Timestamp.Time())
	case "availableBalance.change":
		var resp WsFuturesAvailableBalance
		if err := json.Unmarshal(msg.Data, &resp); err != nil {
			return err
		}
		return e.saveFuturesBalance(ctx, resp.Currency, resp.AvailableBalance, resp.HoldBalance, resp.Timestamp.Time())
	case "orderMargin.change":
		return processPayload[WsFuturesOrderMargin](ctx, e, msg.Data)
	case "withdrawHold.change":
		return processPayload[WsFuturesWithdrawHold](ctx, e, msg.Data)
	}
	return unhandledSubject(msg)
}

// saveFuturesBalance saves and relays a futures balance the way the REST balances store it: the total is the
// available and the held balance
func (e *Exchange) saveFuturesBalance(ctx context.Context, code currency.Code, available, hold float64, updatedAt time.Time) error {
	subAccts := accounts.SubAccounts{accounts.NewSubAccount(asset.Futures, "")}
	subAccts[0].Balances.Set(code, accounts.Balance{
		Total:     available + hold,
		Hold:      hold,
		Free:      available,
		UpdatedAt: updatedAt,
	})
	if err := e.Accounts.Save(ctx, subAccts, false); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, subAccts)
}

// processFuturesPosition relays a Positions push
func (e *Exchange) processFuturesPosition(ctx context.Context, msg *WsMessage) error {
	switch msg.Subject {
	case "position.change":
		return processPayload[WsFuturesPosition](ctx, e, msg.Data)
	case "position.settlement":
		return processPayload[WsFuturesPositionSettlement](ctx, e, msg.Data)
	case "position.adjustRiskLimit":
		return processPayload[WsFuturesRiskLimitAdjustment](ctx, e, msg.Data)
	}
	return unhandledSubject(msg)
}

// processFuturesStopOrder relays a futures Stop Orders push as an order update
func (e *Exchange) processFuturesStopOrder(ctx context.Context, data []byte) error {
	var resp WsFuturesStopOrder
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	pair, err := e.futuresPair(resp.Symbol)
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
	marginType, err := futuresMarginType(resp.MarginMode)
	if err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, &order.Detail{
		Price:        resp.OrderPrice.Float64(),
		TriggerPrice: resp.StopPrice.Float64(),
		Amount:       float64(resp.Size),
		Exchange:     e.Name,
		OrderID:      resp.OrderID,
		Type:         oType,
		Side:         side,
		Status:       status,
		AssetType:    asset.Futures,
		MarginType:   marginType,
		Date:         resp.CreatedAt.Time(),
		LastUpdated:  resp.Timestamp.Time(),
		Pair:         pair,
	})
}
