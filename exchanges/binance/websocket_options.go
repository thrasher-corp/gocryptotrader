package binance

import (
	"context"
	"fmt"
	"strconv"

	"github.com/buger/jsonparser"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	optionsPublicFilter  = "options-public"
	optionsMarketFilter  = "options-market"
	optionsPrivateFilter = "options-private"
)

// Default options subscriptions, expanded to every enabled pair; candles belong to /market, the rest to /public. A
// connection takes at most 200 streams
var (
	defaultOptionsPublicSubscriptions = subscription.List{
		{Enabled: true, Channel: subscription.TickerChannel},
		{Enabled: true, Channel: subscription.AllTradesChannel},
		{Enabled: true, Channel: subscription.OrderbookChannel, Interval: kline.HundredMilliseconds},
	}
	defaultOptionsMarketSubscriptions = subscription.List{
		{Enabled: true, Channel: subscription.CandlesChannel, Interval: kline.OneMin},
	}
)

// optionsKlineIntervals are the candle intervals the options kline stream accepts
var optionsKlineIntervals = []kline.Interval{
	kline.OneMin, kline.ThreeMin, kline.FiveMin, kline.FifteenMin, kline.ThirtyMin, kline.OneHour, kline.TwoHour,
	kline.FourHour, kline.SixHour, kline.TwelveHour, kline.OneDay, kline.ThreeDay, kline.OneWeek,
}

// generateOptionsPublicSubscriptions returns the default options /public stream subscriptions
func (e *Exchange) generateOptionsPublicSubscriptions() (subscription.List, error) {
	return e.generateDerivativesSubscriptions(asset.Options, defaultOptionsPublicSubscriptions)
}

// generateOptionsMarketSubscriptions returns the default options /market stream subscriptions
func (e *Exchange) generateOptionsMarketSubscriptions() (subscription.List, error) {
	return e.generateDerivativesSubscriptions(asset.Options, defaultOptionsMarketSubscriptions)
}

// generateOptionsUserDataSubscriptions returns the options user data stream subscription when it can be used
func (e *Exchange) generateOptionsUserDataSubscriptions() (subscription.List, error) {
	return e.generateUserDataSubscriptions(asset.Options)
}

// wsHandleOptionsData routes options /public and /market stream frames
func (e *Exchange) wsHandleOptionsData(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
	if id, err := jsonparser.GetInt(respRaw, "id"); err == nil {
		return conn.RequireMatchWithData(id, respRaw)
	}
	stream, data, event, err := decodeStreamFrame(respRaw)
	if err != nil || event == "" {
		return err
	}
	switch event {
	case "depthUpdate":
		return e.processOptionsDepth(ctx, stream, data)
	case "bookTicker":
		return sendDecoded[OptionsBookTicker](ctx, e, data)
	case "24hrTicker":
		return e.processOptionsTicker(ctx, data)
	case "trade":
		return e.processOptionsTrade(data)
	case "indexPrice":
		return sendDecodedArray[OptionsIndexPrice](ctx, e, data)
	case "kline":
		return e.processOptionsKline(ctx, data)
	case "markPrice":
		return sendDecodedArray[OptionsMarkPrice](ctx, e, data)
	case "optionSymbol":
		return sendDecoded[OptionsNewSymbol](ctx, e, data)
	case "openInterest":
		return sendDecodedArray[WsOptionsOpenInterest](ctx, e, data)
	default:
		return fmt.Errorf("%w %q on stream %q", errUnhandledStreamEvent, event, stream)
	}
}

// processOptionsDepth applies a depth event: diffs through the REST synchronised order book, partial depth as a
// snapshot of the top levels
func (e *Exchange) processOptionsDepth(ctx context.Context, stream string, data []byte) error {
	var resp OptionsDepthUpdate
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	pair, enabled, err := e.matchDerivativesPair(resp.Symbol, asset.Options)
	if err != nil || !enabled {
		return err
	}
	if isPartialDepthStream(stream) {
		return e.Websocket.Orderbook.LoadSnapshot(ctx, &orderbook.Book{
			Bids:              resp.Bids.Levels(),
			Asks:              resp.Asks.Levels(),
			Exchange:          e.Name,
			Pair:              pair,
			Asset:             asset.Options,
			LastUpdated:       resp.TransactionTime.Time(),
			LastPushed:        resp.EventTime.Time(),
			LastUpdateID:      resp.FinalUpdateID,
			ValidateOrderbook: e.ValidateOrderbook,
		})
	}
	return e.processDerivativesDepthUpdate(ctx, resp.PreviousFinalUpdateID, &orderbook.Update{
		UpdateID:   resp.FinalUpdateID,
		UpdateTime: resp.TransactionTime.Time(),
		LastPushed: resp.EventTime.Time(),
		Asset:      asset.Options,
		Bids:       resp.Bids.Levels(),
		Asks:       resp.Asks.Levels(),
		Pair:       pair,
		AllowEmpty: true,
	})
}

// processOptionsTicker processes a 24 hour ticker of an enabled pair into the ticker store and relays it. The
// underlying form of the stream, <underlying>@optionTicker, pushes every option of that underlying
func (e *Exchange) processOptionsTicker(ctx context.Context, data []byte) error {
	var resp OptionsTicker
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	pair, enabled, err := e.matchDerivativesPair(resp.Symbol, asset.Options)
	if err != nil || !enabled {
		return err
	}
	return e.processAndSendTicker(ctx, &ticker.Price{
		Last:                       resp.LastPrice.Float64(),
		LastSize:                   resp.LastQuantity.Float64(),
		VolumeWeightedAveragePrice: resp.WeightedAveragePrice.Float64(),
		High:                       resp.HighPrice.Float64(),
		Low:                        resp.LowPrice.Float64(),
		BaseVolume:                 resp.TradingVolume.Float64(),
		QuoteVolume:                resp.TradeAmount.Float64(),
		Open:                       resp.OpenPrice.Float64(),
		PercentChange24Hour:        resp.PriceChangePercent.Float64() * 100,
		Pair:                       pair,
		ExchangeName:               e.Name,
		AssetType:                  asset.Options,
		LastUpdated:                resp.EventTime.Time(),
	})
}

// processOptionsTrade relays a trade of an enabled pair. The direction is the taker's side
func (e *Exchange) processOptionsTrade(data []byte) error {
	saveTradeData := e.IsSaveTradeDataEnabled()
	if !saveTradeData && !e.IsTradeFeedEnabled() {
		return nil
	}
	var resp WsOptionsTrade
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	pair, enabled, err := e.matchDerivativesPair(resp.Symbol, asset.Options)
	if err != nil || !enabled {
		return err
	}
	side, err := order.StringToOrderSide(resp.Direction)
	if err != nil {
		return err
	}
	return e.Websocket.Trade.Update(saveTradeData, trade.Data{
		TID:          strconv.FormatUint(resp.TradeID, 10),
		Exchange:     e.Name,
		CurrencyPair: pair,
		AssetType:    asset.Options,
		Side:         side,
		Price:        resp.Price.Float64(),
		Amount:       resp.Quantity.Float64(),
		Timestamp:    resp.TradeCompletedTime.Time(),
	})
}

// processOptionsKline relays a candle of an enabled pair
func (e *Exchange) processOptionsKline(ctx context.Context, data []byte) error {
	var resp OptionsKline
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	pair, enabled, err := e.matchDerivativesPair(resp.Symbol, asset.Options)
	if err != nil || !enabled {
		return err
	}
	interval, err := formatToInterval(resp.Kline.CandlePeriod)
	if err != nil {
		return err
	}
	var validationIssues string
	if !resp.Kline.IsCompleted {
		validationIssues = kline.PartialCandle
	}
	return e.Websocket.DataHandler.Send(ctx, kline.Item{
		Exchange: e.Name,
		Pair:     pair,
		Asset:    asset.Options,
		Interval: interval,
		Candles: []kline.Candle{{
			Time:             resp.Kline.StartTime.Time(),
			Open:             resp.Kline.Open.Float64(),
			High:             resp.Kline.High.Float64(),
			Low:              resp.Kline.Low.Float64(),
			Close:            resp.Kline.Close.Float64(),
			Volume:           resp.Kline.Volume.Float64(),
			ValidationIssues: validationIssues,
		}},
	})
}

// wsHandleOptionsUserData routes options user data stream frames. Their stream name is the listen key, so frames are
// routed on their event type and errors never quote the frame
func (e *Exchange) wsHandleOptionsUserData(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
	if id, err := jsonparser.GetInt(respRaw, "id"); err == nil {
		return conn.RequireMatchWithData(id, respRaw)
	}
	data, _, _, err := jsonparser.Get(respRaw, "data")
	if err != nil {
		return fmt.Errorf("%w: options user data frame without data: %w", errUnhandledStreamEvent, err)
	}
	event, err := jsonparser.GetString(data, "e")
	if err != nil {
		return fmt.Errorf("%w: options user data frame without an event type: %w", errUnhandledStreamEvent, err)
	}
	switch event {
	case "ACCOUNT_UPDATE":
		return sendDecoded[OptionsAccountUpdate](ctx, e, data)
	case "BALANCE_POSITION_UPDATE":
		return e.processOptionsBalancePositionUpdate(ctx, data)
	case "ORDER_TRADE_UPDATE":
		return e.processOptionsOrderTradeUpdate(ctx, data)
	case "GREEK_UPDATE":
		return sendDecoded[OptionsGreekUpdate](ctx, e, data)
	case "RISK_LEVEL_CHANGE":
		return sendDecoded[OptionsRiskLevelChange](ctx, e, data)
	case "listenKeyExpired":
		return e.processListenKeyExpired(ctx, conn, data, asset.Options)
	default:
		return fmt.Errorf("%w %q on the %s user data stream", errUnhandledStreamEvent, event, asset.Options)
	}
}

// processOptionsBalancePositionUpdate stores changed balances and relays them with the changed positions. The event
// carries account balances only, so the margin held from the last REST snapshot is kept and the free balance follows
// the account balance. A position whose symbol has no pair is left out, so it cannot hold back the others
func (e *Exchange) processOptionsBalancePositionUpdate(ctx context.Context, data []byte) error {
	var resp OptionsBalancePositionUpdate
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	if len(resp.Balances) > 0 {
		subAccount := accounts.NewSubAccount(asset.Options, "")
		for _, b := range resp.Balances {
			balance, err := e.Accounts.UpdateBalance(ctx, "", asset.Options, b.MarginAsset, func(balance *accounts.Balance) {
				balance.Total = b.AccountBalance.Float64()
				balance.Free = max(balance.Total-balance.Hold, 0)
			})
			if err != nil {
				return err
			}
			subAccount.Balances.Set(b.MarginAsset, balance)
		}
		if err := e.Websocket.DataHandler.Send(ctx, accounts.SubAccounts{subAccount}); err != nil {
			return err
		}
	}
	positions := make([]futures.Position, 0, len(resp.Positions))
	for i := range resp.Positions {
		p := &resp.Positions[i]
		pair, err := e.userDataPair(p.Symbol, asset.Options)
		if err != nil {
			log.Warnf(log.WebsocketMgr, "%s %s balance and position update: skipping a position: %v", e.Name, asset.Options, err)
			continue
		}
		position := futuresPosition(e.Name, asset.Options, pair, "", p.PositionQuantity.Decimal(), resp.TransactionTime.Time())
		position.OpeningPrice = p.AverageEntryPrice.Decimal()
		position.NotionalSize = p.PositionValue.Decimal().Abs()
		positions = append(positions, position)
	}
	if len(positions) == 0 {
		return nil
	}
	return e.Websocket.DataHandler.Send(ctx, positions)
}

// processOptionsOrderTradeUpdate relays an order update as an order detail. The event carries no creation time, so the
// order is dated by the update, which order stores keep from the first update they see. A fill's commission is that
// fill's alone, so it is reported with the fill as the order's trade rather than as the order's fee
func (e *Exchange) processOptionsOrderTradeUpdate(ctx context.Context, data []byte) error {
	var resp OptionsOrderTradeUpdate
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	o := &resp.Order
	pair, err := e.userDataPair(o.Symbol, asset.Options)
	if err != nil {
		log.Warnf(log.WebsocketMgr, "%s %s order update: skipping order %d: %v", e.Name, asset.Options, o.OrderID, err)
		return nil
	}
	side, err := order.StringToOrderSide(o.Side)
	if err != nil {
		return err
	}
	orderType, err := derivativesOrderType(o.OrderType)
	if err != nil {
		return err
	}
	status, err := derivativesOrderStatus(o.OrderStatus)
	if err != nil {
		return err
	}
	tif, err := derivativesTimeInForce(o.TimeInForce)
	if err != nil {
		return err
	}
	d := &order.Detail{
		TimeInForce:          tif,
		ReduceOnly:           o.IsReduceOnly,
		Price:                o.OriginalPrice.Float64(),
		Amount:               o.OriginalQuantity.Float64(),
		AverageExecutedPrice: o.AveragePrice.Float64(),
		ExecutedAmount:       o.OrderFilledAccumulatedQuantity.Float64(),
		Cost:                 o.AveragePrice.Decimal().Mul(o.OrderFilledAccumulatedQuantity.Decimal()).InexactFloat64(),
		RemainingAmount:      remainingAmount(o.OriginalQuantity, o.OrderFilledAccumulatedQuantity),
		FeeAsset:             o.CommissionAsset,
		Exchange:             e.Name,
		OrderID:              strconv.FormatUint(o.OrderID, 10),
		ClientOrderID:        o.ClientOrderID,
		Type:                 orderType,
		Side:                 side,
		Status:               status,
		AssetType:            asset.Options,
		Date:                 o.OrderTradeTime.Time(),
		LastUpdated:          o.OrderTradeTime.Time(),
		Pair:                 pair,
	}
	if o.ExecutionType == "TRADE" {
		d.Trades = []order.TradeHistory{{
			Price:     o.LastFilledPrice.Float64(),
			Amount:    o.OrderLastFilledQuantity.Float64(),
			Fee:       -o.Commission.Float64(), // Options report a charged fee as a negative commission
			Exchange:  e.Name,
			TID:       strconv.FormatUint(o.TradeID, 10),
			Type:      orderType,
			Side:      side,
			Timestamp: o.OrderTradeTime.Time(),
			IsMaker:   o.IsMaker,
			FeeAsset:  o.CommissionAsset.String(),
		}}
	}
	return e.Websocket.DataHandler.Send(ctx, d)
}
