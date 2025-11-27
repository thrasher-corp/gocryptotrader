package bitget

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"hash/crc32"
	"strconv"
	"strings"
	"text/template"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	bitgetPublicWSURL         = "wss://ws.bitget.com/v2/ws/public"
	bitgetPrivateWSURL        = "wss://ws.bitget.com/v2/ws/private"
	bitgetPublicSandboxWSUrl  = "wss://wspap.bitget.com/v2/ws/public"
	bitgetPrivateSandboxWSUrl = "wss://wspap.bitget.com/v2/ws/private"

	// Websocket endpoints
	// Unauthenticated
	bitgetCandleDailyChannel = "candle1D" // There's one of these for each time period, but we'll ignore those for now
	bitgetBookFullChannel    = "books"    // There's more of these for varying orderbook depths, ignored for now
	bitgetIndexPriceChannel  = "index-price"

	// Authenticated
	bitgetFillChannel             = "fill"
	bitgetOrdersChannel           = "orders"
	bitgetOrdersAlgoChannel       = "orders-algo"
	bitgetPositionsChannel        = "positions"
	bitgetPositionsHistoryChannel = "positions-history"
	bitgetAccountCrossedChannel   = "account-crossed"
	bitgetOrdersCrossedChannel    = "orders-crossed"
	bitgetAccountIsolatedChannel  = "account-isolated"
	bitgetOrdersIsolatedChannel   = "orders-isolated"
)

var subscriptionNames = map[asset.Item]map[string]string{
	asset.Spot: {
		subscription.TickerChannel:    bitgetTicker,
		subscription.CandlesChannel:   bitgetCandleDailyChannel,
		subscription.AllOrdersChannel: bitgetTrade,
		subscription.OrderbookChannel: bitgetBookFullChannel,
		subscription.MyTradesChannel:  bitgetFillChannel,
		subscription.MyOrdersChannel:  bitgetOrdersChannel,
		"myTriggerOrders":             bitgetOrdersAlgoChannel,
		"account":                     bitgetAccount,
	},
	asset.CoinMarginedFutures: {
		subscription.TickerChannel:    bitgetTicker,
		subscription.CandlesChannel:   bitgetCandleDailyChannel,
		subscription.AllOrdersChannel: bitgetTrade,
		subscription.OrderbookChannel: bitgetBookFullChannel,
		subscription.MyTradesChannel:  bitgetFillChannel,
		subscription.MyOrdersChannel:  bitgetOrdersChannel,
		"myTriggerOrders":             bitgetOrdersAlgoChannel,
		"account":                     bitgetAccount,
		"positions":                   bitgetPositionsChannel,
		"positionsHistory":            bitgetPositionsHistoryChannel,
	},
	asset.USDTMarginedFutures: {
		subscription.TickerChannel:    bitgetTicker,
		subscription.CandlesChannel:   bitgetCandleDailyChannel,
		subscription.AllOrdersChannel: bitgetTrade,
		subscription.OrderbookChannel: bitgetBookFullChannel,
		subscription.MyTradesChannel:  bitgetFillChannel,
		subscription.MyOrdersChannel:  bitgetOrdersChannel,
		"myTriggerOrders":             bitgetOrdersAlgoChannel,
		"account":                     bitgetAccount,
		"positions":                   bitgetPositionsChannel,
		"positionsHistory":            bitgetPositionsHistoryChannel,
	},
	asset.USDCMarginedFutures: {
		subscription.TickerChannel:    bitgetTicker,
		subscription.CandlesChannel:   bitgetCandleDailyChannel,
		subscription.AllOrdersChannel: bitgetTrade,
		subscription.OrderbookChannel: bitgetBookFullChannel,
		subscription.MyTradesChannel:  bitgetFillChannel,
		subscription.MyOrdersChannel:  bitgetOrdersChannel,
		"myTriggerOrders":             bitgetOrdersAlgoChannel,
		"account":                     bitgetAccount,
		"positions":                   bitgetPositionsChannel,
		"positionsHistory":            bitgetPositionsHistoryChannel,
	},
	asset.Margin: {
		"indexPrice":                 bitgetIndexPriceChannel,
		subscription.MyOrdersChannel: bitgetOrdersIsolatedChannel,
		"account":                    bitgetAccountIsolatedChannel,
	},
	asset.CrossMargin: {
		"indexPrice":                 bitgetIndexPriceChannel,
		subscription.MyOrdersChannel: bitgetOrdersCrossedChannel,
		"account":                    bitgetAccountCrossedChannel,
	},
}

var defaultSubscriptions = subscription.List{
	{Enabled: true, Channel: subscription.TickerChannel, Asset: asset.Spot},
	{Enabled: true, Channel: subscription.TickerChannel, Asset: asset.USDTMarginedFutures},
	{Enabled: true, Channel: subscription.CandlesChannel, Asset: asset.Spot},
	{Enabled: true, Channel: subscription.CandlesChannel, Asset: asset.USDTMarginedFutures},
	{Enabled: true, Channel: subscription.AllOrdersChannel, Asset: asset.Spot},
	{Enabled: true, Channel: subscription.AllOrdersChannel, Asset: asset.USDTMarginedFutures},
	{Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.Spot},
	{Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.USDTMarginedFutures},
	{Enabled: true, Channel: subscription.MyTradesChannel, Authenticated: true, Asset: asset.Spot},
	{Enabled: true, Channel: subscription.MyTradesChannel, Authenticated: true, Asset: asset.USDTMarginedFutures},
	{Enabled: true, Channel: subscription.MyOrdersChannel, Authenticated: true, Asset: asset.Spot},
	{Enabled: true, Channel: subscription.MyOrdersChannel, Authenticated: true, Asset: asset.USDTMarginedFutures},
	{Enabled: true, Channel: subscription.MyOrdersChannel, Authenticated: true, Asset: asset.Margin},
	{Enabled: true, Channel: subscription.MyOrdersChannel, Authenticated: true, Asset: asset.CrossMargin},
	{Enabled: true, Channel: "myTriggerOrders", Authenticated: true, Asset: asset.Spot},
	{Enabled: true, Channel: "myTriggerOrders", Authenticated: true, Asset: asset.USDTMarginedFutures},
	{Enabled: true, Channel: "account", Authenticated: true, Asset: asset.Spot},
	{Enabled: true, Channel: "account", Authenticated: true, Asset: asset.USDTMarginedFutures},
	{Enabled: true, Channel: "account", Authenticated: true, Asset: asset.Margin},
	{Enabled: true, Channel: "account", Authenticated: true, Asset: asset.CrossMargin},
	{Enabled: true, Channel: "positions", Authenticated: true, Asset: asset.USDTMarginedFutures},
	{Enabled: true, Channel: "positionsHistory", Authenticated: true, Asset: asset.USDTMarginedFutures},
	{Enabled: true, Channel: "indexPrice", Asset: asset.Margin},
}

// wsConnect connects to a websocket feed
func (e *Exchange) wsConnect(ctx context.Context, conn websocket.Connection) error {
	if err := conn.Dial(ctx, &gws.Dialer{}, nil); err != nil {
		return err
	}
	conn.SetupPingHandler(request.Unset, websocket.PingHandler{
		Message:     []byte(`ping`),
		MessageType: gws.TextMessage,
		Delay:       time.Second * 25,
	})
	return nil
}

// wsAuthenticate sends an authentication message to the websocket
func (e *Exchange) wsAuthenticate(ctx context.Context, conn websocket.Connection) error {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	hmac, err := crypto.GetHMAC(crypto.HashSHA256, []byte(timestamp+"GET"+"/user/verify"), []byte(creds.Secret))
	if err != nil {
		return err
	}
	payload := WsLogin{
		Operation: "login",
		Arguments: []WsLoginArgument{{
			APIKey:     creds.Key,
			Signature:  base64.StdEncoding.EncodeToString(hmac),
			Timestamp:  timestamp,
			Passphrase: creds.ClientID,
		}},
	}
	resp, err := conn.SendMessageReturnResponse(ctx, request.Unset, "login-response", payload)
	if err != nil {
		return err
	}
	var wsResp WsResponse
	if err := json.Unmarshal(resp, &wsResp); err != nil {
		return err
	}
	if wsResp.Code != 0 {
		return fmt.Errorf(errWebsocketLoginFailed, e.Name, wsResp.Message)
	}
	return nil
}

// wsHandleData handles data from the websocket connection
// TODO: break up into public and private handlers to reduce complexity
func (e *Exchange) wsHandleData(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
	if string(respRaw[:4]) == "pong" {
		if e.Verbose {
			log.Debugf(log.ExchangeSys, "%v - Websocket pong received\n", e.Name)
		}
		return nil
	}
	var wsResponse *WsResponse
	if err := json.Unmarshal(respRaw, &wsResponse); err != nil {
		return err
	}
	// Under the assumption that the exchange only ever sends one of these. If both can be sent, this will need to be made more complicated
	switch wsResponse.Event + wsResponse.Action {
	case "subscribe":
		if e.Verbose {
			log.Debugf(log.ExchangeSys, "%v - Websocket %v succeeded for %v\n", e.Name, wsResponse.Event, wsResponse.Arg)
		}
	case "error":
		if conn.IncomingWithData("login-response", respRaw) {
			return nil
		}
		var args []map[string]any
		if err := json.Unmarshal(wsResponse.Arg, &args); err != nil {
			return err
		}
		if len(args) == 1 {
			if id, ok := args[0]["id"]; ok {
				if idStr, ok := id.(string); ok {
					return conn.RequireMatchWithData(idStr, respRaw)
				}
			}
		}
		return fmt.Errorf(errWebsocketGeneric, e.Name, wsResponse.Code, wsResponse.Message)
	case "login":
		return conn.RequireMatchWithData("login-response", respRaw)
	case "snapshot":
		var arg WsArgument
		if err := json.Unmarshal(wsResponse.Arg, &arg); err != nil {
			return err
		}
		switch arg.Channel {
		case bitgetTicker:
			return e.tickerDataHandler(wsResponse, arg.InstrumentType)
		case bitgetCandleDailyChannel:
			return e.candleDataHandler(wsResponse, arg.InstrumentID, arg.InstrumentType)
		case bitgetTrade:
			return e.tradeDataHandler(wsResponse, arg.InstrumentID, arg.InstrumentType)
		case bitgetBookFullChannel:
			return e.orderbookDataHandler(wsResponse, arg.InstrumentID, arg.InstrumentType)
		case bitgetAccount:
			return e.accountSnapshotDataHandler(ctx, wsResponse, arg.InstrumentType)
		case bitgetFillChannel:
			return e.fillDataHandler(wsResponse, arg.InstrumentType)
		case bitgetOrdersChannel:
			return e.genOrderDataHandler(wsResponse, arg.InstrumentType)
		case bitgetOrdersAlgoChannel:
			return e.triggerOrderDataHandler(wsResponse, arg.InstrumentType)
		case bitgetPositionsChannel:
			return e.positionsDataHandler(wsResponse, arg.InstrumentType)
		case bitgetPositionsHistoryChannel:
			return e.positionsHistoryDataHandler(wsResponse)
		case bitgetIndexPriceChannel:
			return e.indexPriceDataHandler(wsResponse, arg.InstrumentType)
		case bitgetAccountCrossedChannel:
			return e.crossAccountDataHandler(ctx, wsResponse)
		case bitgetOrdersCrossedChannel, bitgetOrdersIsolatedChannel:
			return e.marginOrderDataHandler(wsResponse, arg.InstrumentType, arg.Channel)
		case bitgetAccountIsolatedChannel:
			return e.isolatedAccountDataHandler(ctx, wsResponse)
		default:
			e.Websocket.DataHandler <- websocket.UnhandledMessageWarning{Message: e.Name + websocket.UnhandledMessage + string(respRaw)}
		}
	case "update":
		var arg WsArgument
		if err := json.Unmarshal(wsResponse.Arg, &arg); err != nil {
			return err
		}
		switch arg.Channel {
		case bitgetCandleDailyChannel:
			return e.candleDataHandler(wsResponse, arg.InstrumentID, arg.InstrumentType)
		case bitgetTrade:
			return e.tradeDataHandler(wsResponse, arg.InstrumentID, arg.InstrumentType)
		case bitgetBookFullChannel:
			return e.orderbookDataHandler(wsResponse, arg.InstrumentID, arg.InstrumentType)
		case bitgetAccount:
			return e.accountUpdateDataHandler(ctx, wsResponse, arg.InstrumentType)
		default:
			e.Websocket.DataHandler <- websocket.UnhandledMessageWarning{Message: e.Name + websocket.UnhandledMessage + string(respRaw)}
		}
	case "trade":
		var args []map[string]any
		if err := json.Unmarshal(wsResponse.Arg, &args); err != nil {
			return err
		}
		if len(args) == 1 {
			if id, ok := args[0]["id"]; ok {
				if idStr, ok := id.(string); ok {
					return conn.RequireMatchWithData(idStr, respRaw)
				}
			}
		}
		return errors.New("unable to correlate trade response")
	default:
		e.Websocket.DataHandler <- websocket.UnhandledMessageWarning{Message: e.Name + websocket.UnhandledMessage + string(respRaw)}
	}
	return nil
}

// TickerDataHandler handles incoming ticker data for websockets
func (e *Exchange) tickerDataHandler(wsResponse *WsResponse, instrumentType string) error {
	respAsset := itemDecoder(instrumentType)
	switch respAsset {
	case asset.Spot:
		var ticks []WsTickerSnapshotSpot
		if err := json.Unmarshal(wsResponse.Data, &ticks); err != nil {
			return err
		}
		for i := range ticks {
			pair, err := pairFromStringHelper(ticks[i].InstrumentID)
			if err != nil {
				return err
			}
			e.Websocket.DataHandler <- &ticker.Price{
				Last:         ticks[i].LastPrice.Float64(),
				High:         ticks[i].High24H.Float64(),
				Low:          ticks[i].Low24H.Float64(),
				Bid:          ticks[i].BidPrice.Float64(),
				Ask:          ticks[i].AskPrice.Float64(),
				Volume:       ticks[i].BaseVolume.Float64(),
				QuoteVolume:  ticks[i].QuoteVolume.Float64(),
				Open:         ticks[i].Open24H.Float64(),
				Pair:         pair,
				ExchangeName: e.Name,
				AssetType:    itemDecoder(instrumentType),
				LastUpdated:  ticks[i].Timestamp.Time(),
			}
		}
	case asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.USDCMarginedFutures:
		var ticks []WsTickerSnapshotFutures
		if err := json.Unmarshal(wsResponse.Data, &ticks); err != nil {
			return err
		}
		for i := range ticks {
			pair, err := pairFromStringHelper(ticks[i].InstrumentID)
			if err != nil {
				return err
			}
			e.Websocket.DataHandler <- &ticker.Price{
				Last:         ticks[i].LastPrice.Float64(),
				High:         ticks[i].High24H.Float64(),
				Low:          ticks[i].Low24H.Float64(),
				Bid:          ticks[i].BidPrice.Float64(),
				Ask:          ticks[i].AskPrice.Float64(),
				Volume:       ticks[i].BaseVolume.Float64(),
				QuoteVolume:  ticks[i].QuoteVolume.Float64(),
				Open:         ticks[i].Open24H.Float64(),
				MarkPrice:    ticks[i].MarkPrice.Float64(),
				IndexPrice:   ticks[i].IndexPrice.Float64(),
				Pair:         pair,
				ExchangeName: e.Name,
				AssetType:    itemDecoder(instrumentType),
				LastUpdated:  ticks[i].Timestamp.Time(),
			}
		}
	default:
		e.Websocket.DataHandler <- fmt.Errorf("%s %s %w %s", e.Name, respAsset, asset.ErrNotSupported, instrumentType)
	}
	return nil
}

// CandleDataHandler handles candle data, as functionality is shared between updates and snapshots
func (e *Exchange) candleDataHandler(wsResponse *WsResponse, instrumentID, instrumentType string) error {
	var candles [][8]string
	if err := json.Unmarshal(wsResponse.Data, &candles); err != nil {
		return err
	}
	pair, err := pairFromStringHelper(instrumentID)
	if err != nil {
		return err
	}
	resp := make([]websocket.KlineData, len(candles))
	for i := range candles {
		ts, err := strconv.ParseInt(candles[i][0], 10, 64)
		if err != nil {
			return err
		}
		open, err := strconv.ParseFloat(candles[i][1], 64)
		if err != nil {
			return err
		}
		closePrice, err := strconv.ParseFloat(candles[i][4], 64)
		if err != nil {
			return err
		}
		high, err := strconv.ParseFloat(candles[i][2], 64)
		if err != nil {
			return err
		}
		low, err := strconv.ParseFloat(candles[i][3], 64)
		if err != nil {
			return err
		}
		volume, err := strconv.ParseFloat(candles[i][5], 64)
		if err != nil {
			return err
		}
		resp[i] = websocket.KlineData{
			Timestamp:  wsResponse.Timestamp.Time(),
			Pair:       pair,
			AssetType:  itemDecoder(instrumentType),
			Exchange:   e.Name,
			StartTime:  time.UnixMilli(ts),
			CloseTime:  time.UnixMilli(ts).Add(time.Hour * 24),
			Interval:   "1d",
			OpenPrice:  open,
			ClosePrice: closePrice,
			HighPrice:  high,
			LowPrice:   low,
			Volume:     volume,
		}
	}
	e.Websocket.DataHandler <- resp
	return nil
}

// TradeDataHandler handles trade data, as functionality is shared between updates and snapshots
func (e *Exchange) tradeDataHandler(wsResponse *WsResponse, instrumentID, instrumentType string) error {
	pair, err := pairFromStringHelper(instrumentID)
	if err != nil {
		return err
	}
	var trades []WsTradeResponse
	if err := json.Unmarshal(wsResponse.Data, &trades); err != nil {
		return err
	}
	resp := make([]trade.Data, len(trades))
	for i := range trades {
		resp[i] = trade.Data{
			Timestamp:    trades[i].Timestamp.Time(),
			CurrencyPair: pair,
			AssetType:    itemDecoder(instrumentType),
			Exchange:     e.Name,
			Price:        trades[i].Price.Float64(),
			Amount:       trades[i].Size.Float64(),
			Side:         sideDecoder(trades[i].Side),
			TID:          strconv.FormatInt(trades[i].TradeID, 10),
		}
	}
	e.Websocket.DataHandler <- resp
	return nil
}

// OrderbookDataHandler handles orderbook data, as functionality is shared between updates and snapshots
func (e *Exchange) orderbookDataHandler(wsResponse *WsResponse, instrumentID, instrumentType string) error {
	pair, err := pairFromStringHelper(instrumentID)
	if err != nil {
		return err
	}
	var ob []WsOrderBookResponse
	if err := json.Unmarshal(wsResponse.Data, &ob); err != nil {
		return err
	}
	if len(ob) == 0 {
		return common.ErrNoResults
	}
	bids, err := levelConstructor(ob[0].Bids)
	if err != nil {
		return err
	}
	asks, err := levelConstructor(ob[0].Asks)
	if err != nil {
		return err
	}
	if wsResponse.Action[0] == 's' {
		if len(bids) > 0 && bids[len(bids)-1].Price == 0 {
			// Bid depths periodically contain a zero priced entry that needs to be removed. This might clash with the
			// checksum validation when liquidity/levels are low and would need to be a strict rule to be applied when
			// loading a snapshot if this becomes a trading problem.
			bids = bids[:len(bids)-1]
		}
		return e.Websocket.Orderbook.LoadSnapshot(&orderbook.Book{
			Pair:                   pair,
			Asset:                  itemDecoder(instrumentType),
			Bids:                   bids,
			Asks:                   asks,
			LastUpdated:            wsResponse.Timestamp.Time(),
			Exchange:               e.Name,
			ValidateOrderbook:      e.ValidateOrderbook,
			ChecksumStringRequired: true,
		})
	}
	update := orderbook.Update{
		Bids:             bids,
		Asks:             asks,
		Pair:             pair,
		UpdateTime:       wsResponse.Timestamp.Time(),
		Asset:            itemDecoder(instrumentType),
		GenerateChecksum: calculateUpdateOrderbookChecksum,
		ExpectedChecksum: uint32(ob[0].Checksum), //nolint:gosec // The exchange sends it as ints expecting overflows to be handled as Go does by default
		AllowEmpty:       true,
	}
	// TODO: Need to have resub manager to handle checksum failures. See Gateio implementation for reference #2045
	// Can copy that code almost verbatim with minor adjustments.
	return e.Websocket.Orderbook.Update(&update)
}

// AccountSnapshotDataHandler handles account snapshot data
func (e *Exchange) accountSnapshotDataHandler(ctx context.Context, wsResponse *WsResponse, instrumentType string) error {
	var subAcc *accounts.SubAccount
	switch a := itemDecoder(instrumentType); a {
	case asset.Spot:
		var acc []WsAccountSpotResponse
		if err := json.Unmarshal(wsResponse.Data, &acc); err != nil {
			return err
		}
		subAcc = accounts.NewSubAccount(a, "")
		for i := range acc {
			subAcc.Balances.Set(acc[i].Coin, accounts.Balance{
				Total: acc[i].Available.Float64() + acc[i].Frozen.Float64() + acc[i].Locked.Float64(),
				Free:  acc[i].Available.Float64(),
				Hold:  acc[i].Frozen.Float64() + acc[i].Locked.Float64(),
			})
		}
	case asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.USDCMarginedFutures:
		var acc []WsAccountFuturesResponse
		if err := json.Unmarshal(wsResponse.Data, &acc); err != nil {
			return err
		}
		subAcc = accounts.NewSubAccount(a, "")
		for i := range acc {
			subAcc.Balances.Set(acc[i].MarginCoin, accounts.Balance{
				Total: acc[i].Available.Float64() + acc[i].Frozen.Float64(),
				Free:  acc[i].Available.Float64(),
				Hold:  acc[i].Frozen.Float64(),
			})
		}
	default:
		return fmt.Errorf("%s %s %w %s", e.Name, a, asset.ErrNotSupported, instrumentType)
	}
	subAccts := accounts.SubAccounts{subAcc}
	if err := e.Accounts.Save(ctx, subAccts, false); err != nil {
		return err
	}
	e.Websocket.DataHandler <- subAccts
	return nil
}

func (e *Exchange) fillDataHandler(wsResponse *WsResponse, instrumentType string) error {
	respAsset := itemDecoder(instrumentType)
	switch respAsset {
	case asset.Spot:
		var fil []WsFillSpotResponse
		if err := json.Unmarshal(wsResponse.Data, &fil); err != nil {
			return err
		}
		resp := make([]fill.Data, len(fil))
		for i := range fil {
			pair, err := pairFromStringHelper(fil[i].Symbol)
			if err != nil {
				return err
			}
			resp[i] = fill.Data{
				ID:           strconv.FormatInt(fil[i].TradeID, 10),
				Timestamp:    fil[i].CreationTime.Time(),
				Exchange:     e.Name,
				AssetType:    asset.Spot,
				CurrencyPair: pair,
				Side:         sideDecoder(fil[i].Side),
				OrderID:      strconv.FormatInt(fil[i].OrderID, 10),
				TradeID:      strconv.FormatInt(fil[i].TradeID, 10),
				Price:        fil[i].PriceAverage.Float64(),
				Amount:       fil[i].Size.Float64(),
			}
		}
		e.Websocket.DataHandler <- resp
	case asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.USDCMarginedFutures:
		var fil []WsFillFuturesResponse
		if err := json.Unmarshal(wsResponse.Data, &fil); err != nil {
			return err
		}
		resp := make([]fill.Data, len(fil))
		for i := range fil {
			pair, err := pairFromStringHelper(fil[i].Symbol)
			if err != nil {
				return err
			}
			resp[i] = fill.Data{
				Exchange:     e.Name,
				CurrencyPair: pair,
				OrderID:      strconv.FormatInt(fil[i].OrderID, 10),
				TradeID:      strconv.FormatInt(fil[i].TradeID, 10),
				Side:         sideDecoder(fil[i].Side),
				Price:        fil[i].Price.Float64(),
				Amount:       fil[i].BaseVolume.Float64(),
				Timestamp:    fil[i].CreationTime.Time(),
			}
		}
		e.Websocket.DataHandler <- resp
	default:
		e.Websocket.DataHandler <- fmt.Errorf("%s %s %w %s", e.Name, respAsset, asset.ErrNotSupported, instrumentType)
	}
	return nil
}

// genOrderDataHandler handles generic order data
func (e *Exchange) genOrderDataHandler(wsResponse *WsResponse, instrumentType string) error {
	respAsset := itemDecoder(instrumentType)
	switch respAsset {
	case asset.Spot:
		var orders []WsOrderSpotResponse
		if err := json.Unmarshal(wsResponse.Data, &orders); err != nil {
			return err
		}
		resp := make([]order.Detail, len(orders))
		for i := range orders {
			pair, err := pairFromStringHelper(orders[i].InstrumentID)
			if err != nil {
				return err
			}
			var baseAmount, quoteAmount float64
			side := sideDecoder(orders[i].Side)
			if side == order.Buy {
				quoteAmount = orders[i].Size.Float64()
			}
			if side == order.Sell {
				baseAmount = orders[i].Size.Float64()
			}
			orderType := typeDecoder(orders[i].OrderType)
			if orderType == order.Limit {
				baseAmount = orders[i].NewSize.Float64()
			}
			resp[i] = order.Detail{
				Exchange:             e.Name,
				AssetType:            asset.Spot,
				Pair:                 pair,
				OrderID:              strconv.FormatInt(orders[i].OrderID, 10),
				ClientOrderID:        orders[i].ClientOrderID,
				Price:                orders[i].PriceAverage.Float64(),
				Amount:               baseAmount,
				QuoteAmount:          quoteAmount,
				Type:                 orderType,
				TimeInForce:          strategyDecoder(orders[i].Force),
				Side:                 side,
				AverageExecutedPrice: orders[i].PriceAverage.Float64(),
				Status:               statusDecoder(orders[i].Status),
				Date:                 orders[i].CreationTime.Time(),
				LastUpdated:          orders[i].UpdateTime.Time(),
			}
			for x := range orders[i].FeeDetail {
				resp[i].Fee += orders[i].FeeDetail[x].TotalFee.Float64()
				resp[i].FeeAsset = orders[i].FeeDetail[x].FeeCoin
			}
		}
		e.Websocket.DataHandler <- resp
	case asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.USDCMarginedFutures:
		var orders []WsOrderFuturesResponse
		if err := json.Unmarshal(wsResponse.Data, &orders); err != nil {
			return err
		}
		resp := make([]order.Detail, len(orders))
		for i := range orders {
			pair, err := pairFromStringHelper(orders[i].InstrumentID)
			if err != nil {
				return err
			}
			var baseAmount, quoteAmount float64
			side := sideDecoder(orders[i].Side)
			if side == order.Buy {
				quoteAmount = orders[i].Size.Float64()
			}
			if side == order.Sell {
				baseAmount = orders[i].Size.Float64()
			}
			orderType := typeDecoder(orders[i].OrderType)
			if orderType == order.Limit {
				baseAmount = orders[i].BaseVolume.Float64()
			}
			resp[i] = order.Detail{
				Exchange:             e.Name,
				AssetType:            respAsset,
				Pair:                 pair,
				Amount:               baseAmount,
				QuoteAmount:          quoteAmount,
				Type:                 orderType,
				TimeInForce:          strategyDecoder(orders[i].Force),
				Side:                 side,
				ExecutedAmount:       orders[i].FilledQuantity.Float64(),
				Date:                 orders[i].CreationTime.Time(),
				ClientOrderID:        orders[i].ClientOrderID,
				Leverage:             orders[i].Leverage.Float64(),
				MarginType:           marginDecoder(orders[i].MarginMode),
				OrderID:              strconv.FormatInt(orders[i].OrderID, 10),
				Price:                orders[i].Price.Float64(),
				AverageExecutedPrice: orders[i].PriceAverage.Float64(),
				ReduceOnly:           bool(orders[i].ReduceOnly),
				Status:               statusDecoder(orders[i].Status),
				LimitPriceLower:      orders[i].PresetStopSurplusPrice.Float64(),
				LimitPriceUpper:      orders[i].PresetStopLossPrice.Float64(),
				LastUpdated:          orders[i].UpdateTime.Time(),
			}
			for x := range orders[i].FeeDetail {
				resp[i].Fee += orders[i].FeeDetail[x].Fee.Float64()
				resp[i].FeeAsset = orders[i].FeeDetail[x].FeeCoin
			}
		}
		e.Websocket.DataHandler <- resp
	default:
		e.Websocket.DataHandler <- fmt.Errorf("%s %s %w %s", e.Name, respAsset, asset.ErrNotSupported, instrumentType)
	}
	return nil
}

// TriggerOrderDataHandler handles trigger order data
func (e *Exchange) triggerOrderDataHandler(wsResponse *WsResponse, instrumentType string) error {
	respAsset := itemDecoder(instrumentType)
	switch respAsset {
	case asset.Spot:
		var orders []WsTriggerOrderSpotResponse
		if err := json.Unmarshal(wsResponse.Data, &orders); err != nil {
			return err
		}
		resp := make([]order.Detail, len(orders))
		for i := range orders {
			pair, err := pairFromStringHelper(orders[i].InstrumentID)
			if err != nil {
				return err
			}
			resp[i] = order.Detail{
				Exchange:      e.Name,
				AssetType:     asset.Spot,
				Pair:          pair,
				OrderID:       strconv.FormatInt(orders[i].OrderID, 10),
				ClientOrderID: orders[i].ClientOrderID,
				TriggerPrice:  orders[i].TriggerPrice.Float64(),
				Price:         orders[i].Price.Float64(),
				Amount:        orders[i].Size.Float64(),
				Type:          typeDecoder(orders[i].OrderType),
				Side:          sideDecoder(orders[i].Side),
				Status:        statusDecoder(orders[i].Status),
				Date:          orders[i].CreationTime.Time(),
				LastUpdated:   orders[i].UpdateTime.Time(),
			}
		}
		e.Websocket.DataHandler <- resp
	case asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.USDCMarginedFutures:
		var orders []WsTriggerOrderFuturesResponse
		if err := json.Unmarshal(wsResponse.Data, &orders); err != nil {
			return err
		}
		resp := make([]order.Detail, len(orders))
		for i := range orders {
			pair, err := pairFromStringHelper(orders[i].InstrumentID)
			if err != nil {
				return err
			}
			resp[i] = order.Detail{
				Exchange:             e.Name,
				AssetType:            respAsset,
				Pair:                 pair,
				OrderID:              strconv.FormatInt(orders[i].OrderID, 10),
				ClientOrderID:        orders[i].ClientOrderID,
				TriggerPrice:         orders[i].TriggerPrice.Float64(),
				Price:                orders[i].Price.Float64(),
				AverageExecutedPrice: orders[i].ExecutePrice.Float64(),
				Amount:               orders[i].Size.Float64(),
				Type:                 typeDecoder(orders[i].OrderType),
				Side:                 sideDecoder(orders[i].Side),
				Status:               statusDecoder(orders[i].Status),
				Date:                 orders[i].CreationTime.Time(),
				LastUpdated:          orders[i].UpdateTime.Time(),
			}
		}
		e.Websocket.DataHandler <- resp
	default:
		e.Websocket.DataHandler <- fmt.Errorf("%s %s %w %s", e.Name, respAsset, asset.ErrNotSupported, instrumentType)
	}
	return nil
}

// PositionsDataHandler handles data on futures positions
func (e *Exchange) positionsDataHandler(wsResponse *WsResponse, instrumentType string) error {
	var positions []WsPositionResponse
	if err := json.Unmarshal(wsResponse.Data, &positions); err != nil {
		return err
	}
	resp := make([]order.Detail, len(positions))
	for i := range positions {
		pair, err := pairFromStringHelper(positions[i].InstrumentID)
		if err != nil {
			return err
		}
		resp[i] = order.Detail{
			Exchange:             e.Name,
			AssetType:            itemDecoder(instrumentType),
			Pair:                 pair,
			OrderID:              strconv.FormatInt(positions[i].PositionID, 10),
			MarginType:           marginDecoder(positions[i].MarginMode),
			Side:                 sideDecoder(positions[i].HoldSide),
			Amount:               positions[i].Total.Float64(),
			AverageExecutedPrice: positions[i].OpenPriceAverage.Float64(),
			Leverage:             positions[i].Leverage.Float64(),
			Date:                 positions[i].CreationTime.Time(),
			Fee:                  positions[i].TotalFee.Float64(),
			LastUpdated:          positions[i].UpdateTime.Time(),
		}
	}
	e.Websocket.DataHandler <- resp
	return nil
}

// PositionsHistoryDataHandler handles data on futures positions history
func (e *Exchange) positionsHistoryDataHandler(wsResponse *WsResponse) error {
	var positions []WsPositionHistoryResponse
	if err := json.Unmarshal(wsResponse.Data, &positions); err != nil {
		return err
	}
	resp := make([]futures.PositionHistory, len(positions))
	for i := range positions {
		pair, err := pairFromStringHelper(positions[i].InstrumentID)
		if err != nil {
			return err
		}
		resp[i] = futures.PositionHistory{
			Exchange:          e.Name,
			PositionID:        strconv.FormatInt(positions[i].PositionID, 10),
			Pair:              pair,
			MarginCoin:        positions[i].MarginCoin,
			MarginType:        marginDecoder(positions[i].MarginMode),
			Side:              sideDecoder(positions[i].HoldSide),
			PositionMode:      positionModeDecoder(positions[i].PositionMode),
			OpenAveragePrice:  positions[i].OpenPriceAverage.Float64(),
			CloseAveragePrice: positions[i].ClosePriceAverage.Float64(),
			OpenSize:          positions[i].OpenSize.Float64(),
			CloseSize:         positions[i].CloseSize.Float64(),
			RealisedPnl:       positions[i].AchievedProfits.Float64(),
			SettlementFee:     positions[i].SettleFee.Float64(),
			OpenFee:           positions[i].OpenFee.Float64(),
			CloseFee:          positions[i].CloseFee.Float64(),
			StartDate:         positions[i].CreationTime.Time(),
			LastUpdated:       positions[i].UpdateTime.Time(),
		}
	}
	// Implement a better handler for this once work on account.Holdings begins
	e.Websocket.DataHandler <- resp
	return nil
}

// IndexPriceDataHandler handles index price data
func (e *Exchange) indexPriceDataHandler(wsResponse *WsResponse, instrumentType string) error {
	var indexPrice []WsIndexPriceResponse
	if err := json.Unmarshal(wsResponse.Data, &indexPrice); err != nil {
		return err
	}
	resp := make([]ticker.Price, len(indexPrice))
	var cur int
	for i := range indexPrice {
		as := itemDecoder(instrumentType)
		pair, enabled, err := e.MatchSymbolCheckEnabled(indexPrice[i].Symbol, as, false)
		// The exchange sometimes returns unavailable pairs such as "USDT/USDT" which should be ignored
		if !enabled || err != nil {
			continue
		}
		resp[cur] = ticker.Price{
			ExchangeName: e.Name,
			AssetType:    as,
			Pair:         pair,
			Last:         indexPrice[i].IndexPrice.Float64(),
			LastUpdated:  indexPrice[i].Timestamp.Time(),
		}
		cur++
	}
	resp = resp[:cur]
	e.Websocket.DataHandler <- resp
	return nil
}

// CrossAccountDataHandler handles cross margin account data
func (e *Exchange) crossAccountDataHandler(ctx context.Context, wsResponse *WsResponse) error {
	var acc []WsAccountCrossMarginResponse
	if err := json.Unmarshal(wsResponse.Data, &acc); err != nil {
		return err
	}
	subAcct := accounts.NewSubAccount(asset.CrossMargin, "")
	for i := range acc {
		subAcct.Balances.Set(acc[i].Coin, accounts.Balance{
			Total:                  acc[i].Available.Float64() + acc[i].Frozen.Float64() + acc[i].Borrow.Float64() + acc[i].Interest.Float64() + acc[i].Coupon.Float64(),
			Free:                   acc[i].Available.Float64(),
			Hold:                   acc[i].Frozen.Float64(),
			Borrowed:               acc[i].Borrow.Float64(),
			AvailableWithoutBorrow: acc[i].Available.Float64(),
		})
	}
	subAccts := accounts.SubAccounts{subAcct}
	if err := e.Accounts.Save(ctx, subAccts, false); err != nil {
		return err
	}
	e.Websocket.DataHandler <- subAccts
	return nil
}

// MarginOrderDataHandler handles margin order data
func (e *Exchange) marginOrderDataHandler(wsResponse *WsResponse, instrumentType, channel string) error {
	var orders []WsOrderMarginResponse
	if err := json.Unmarshal(wsResponse.Data, &orders); err != nil {
		return err
	}
	resp := make([]order.Detail, len(orders))
	pair, err := pairFromStringHelper(instrumentType)
	if err != nil {
		return err
	}
	for i := range orders {
		resp[i] = order.Detail{
			Exchange:             e.Name,
			Pair:                 pair,
			OrderID:              strconv.FormatInt(orders[i].OrderID, 10),
			ClientOrderID:        orders[i].ClientOrderID,
			AverageExecutedPrice: orders[i].FillPrice.Float64(),
			Price:                orders[i].Price.Float64(),
			Amount:               orders[i].BaseSize.Float64(),
			QuoteAmount:          orders[i].QuoteSize.Float64(),
			Type:                 typeDecoder(orders[i].OrderType),
			TimeInForce:          strategyDecoder(orders[i].Force),
			Side:                 sideDecoder(orders[i].Side),
			Status:               statusDecoder(orders[i].Status),
			Date:                 orders[i].CreationTime.Time(),
		}
		for x := range orders[i].FeeDetail {
			resp[i].Fee += orders[i].FeeDetail[x].TotalFee.Float64()
			resp[i].FeeAsset = orders[i].FeeDetail[x].FeeCoin
		}
		if channel == bitgetOrdersIsolatedChannel {
			resp[i].AssetType = asset.Margin
		} else {
			resp[i].AssetType = asset.CrossMargin
		}
	}
	e.Websocket.DataHandler <- resp
	return nil
}

// IsolatedAccountDataHandler handles isolated margin account data
func (e *Exchange) isolatedAccountDataHandler(ctx context.Context, wsResponse *WsResponse) error {
	var acc []WsAccountIsolatedMarginResponse
	if err := json.Unmarshal(wsResponse.Data, &acc); err != nil {
		return err
	}
	subAcct := accounts.NewSubAccount(asset.Margin, "")
	for i := range acc {
		subAcct.Balances.Set(acc[i].Coin, accounts.Balance{
			Total:                  acc[i].Available.Float64() + acc[i].Frozen.Float64() + acc[i].Borrow.Float64() + acc[i].Interest.Float64() + acc[i].Coupon.Float64(),
			Free:                   acc[i].Available.Float64(),
			Hold:                   acc[i].Frozen.Float64(),
			Borrowed:               acc[i].Borrow.Float64(),
			AvailableWithoutBorrow: acc[i].Available.Float64(),
		})
	}
	subAccts := accounts.SubAccounts{subAcct}
	if err := e.Accounts.Save(ctx, subAccts, false); err != nil {
		return err
	}
	e.Websocket.DataHandler <- subAccts
	return nil
}

// AccountUpdateDataHandler
func (e *Exchange) accountUpdateDataHandler(ctx context.Context, wsResponse *WsResponse, instrumentType string) error {
	var subAcc *accounts.SubAccount
	switch a := itemDecoder(instrumentType); a {
	case asset.Spot:
		var acc []WsAccountSpotResponse
		if err := json.Unmarshal(wsResponse.Data, &acc); err != nil {
			return err
		}
		subAcc = accounts.NewSubAccount(asset.Spot, "")
		for i := range acc {
			subAcc.Balances.Set(acc[i].Coin, accounts.Balance{
				Total:     acc[i].Available.Float64() + acc[i].Frozen.Float64() + acc[i].Locked.Float64(),
				Free:      acc[i].Available.Float64(),
				Hold:      acc[i].Frozen.Float64() + acc[i].Locked.Float64(),
				UpdatedAt: acc[i].UpdateTime.Time(),
			})
		}
	case asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.USDCMarginedFutures:
		var acc []WsAccountFuturesResponse
		if err := json.Unmarshal(wsResponse.Data, &acc); err != nil {
			return err
		}
		subAcc = accounts.NewSubAccount(a, "")
		for i := range acc {
			subAcc.Balances.Set(acc[i].MarginCoin, accounts.Balance{
				Total: acc[i].Available.Float64() + acc[i].Frozen.Float64(),
				Free:  acc[i].Available.Float64(),
				Hold:  acc[i].Frozen.Float64(),
			})
		}
	default:
		return fmt.Errorf("%s %s %w %s", e.Name, a, asset.ErrNotSupported, instrumentType)
	}
	subAccts := accounts.SubAccounts{subAcc}
	if err := e.Accounts.Save(ctx, subAccts, false); err != nil {
		return err
	}
	e.Websocket.DataHandler <- subAccts
	return nil
}

// LevelConstructor turns the exchange's orderbook data into a standardised format for the engine
func levelConstructor(data [][2]string) ([]orderbook.Level, error) {
	resp := make([]orderbook.Level, len(data))
	var err error
	for i := range data {
		if resp[i].Price, err = strconv.ParseFloat(data[i][0], 64); err != nil {
			return nil, err
		}
		if resp[i].Amount, err = strconv.ParseFloat(data[i][1], 64); err != nil {
			return nil, err
		}
		resp[i].StrPrice = data[i][0]
		resp[i].StrAmount = data[i][1]
	}
	return resp, nil
}

// calculateUpdateOrderbookChecksum calculates the checksum of the orderbook data
func calculateUpdateOrderbookChecksum(orderbookData *orderbook.Book) uint32 {
	var builder strings.Builder
	for i := range 25 {
		if len(orderbookData.Bids) > i {
			builder.WriteString(orderbookData.Bids[i].StrPrice + ":" + orderbookData.Bids[i].StrAmount + ":")
		}
		if len(orderbookData.Asks) > i {
			builder.WriteString(orderbookData.Asks[i].StrPrice + ":" + orderbookData.Asks[i].StrAmount + ":")
		}
	}
	check := builder.String()
	if check != "" {
		check = check[:len(check)-1]
	}
	return crc32.ChecksumIEEE([]byte(check))
}

// wsGenerateSubscriptions expands the default subscriptions list
func (e *Exchange) wsGenerateSubscriptions(public bool) (subscription.List, error) {
	assetPairs := make(map[asset.Item]currency.Pairs)
	for _, a := range e.GetAssetTypes(true) {
		pairs, err := e.GetEnabledPairs(a)
		if err != nil {
			return nil, err
		}
		assetPairs[a] = pairs
	}
	subs := make(subscription.List, 0, len(e.Features.Subscriptions))
	for _, sub := range e.Features.Subscriptions {
		_, ok := assetPairs[sub.Asset]
		if !ok && !sub.Authenticated {
			continue
		}
		if sub.Enabled && public == !sub.Authenticated {
			subs = append(subs, sub.Clone()) // Slow, consider this a placeholder until templating support is finished
		}
	}
	for i := range subs {
		subs[i].Channel = subscriptionNames[subs[i].Asset][subs[i].Channel]
		switch subs[i].Channel {
		case bitgetAccount, bitgetFillChannel, bitgetPositionsChannel, bitgetPositionsHistoryChannel, bitgetIndexPriceChannel, bitgetAccountCrossedChannel, bitgetAccountIsolatedChannel:
		default:
			subs[i].Pairs = assetPairs[subs[i].Asset]
		}
	}
	return subs, nil
}

// Subscribe sends a websocket message to receive data from the channel
func (e *Exchange) Subscribe(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	return e.manageSubs(ctx, conn, "subscribe", subs)
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (e *Exchange) Unsubscribe(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	return e.manageSubs(ctx, conn, "unsubscribe", subs)
}

// ReqSplitter splits a request into multiple requests to avoid going over the byte limit
func reqSplitter(req *WsRequest) []WsRequest {
	capacity := (len(req.Arguments) / 47) + 1
	reqs := make([]WsRequest, capacity)
	for i := range capacity {
		reqs[i].Operation = req.Operation
		if i == capacity-1 {
			reqs[i].Arguments = req.Arguments[i*47:]
			break
		}
		reqs[i].Arguments = req.Arguments[i*47 : (i+1)*47]
	}
	return reqs
}

// ReqBuilder builds a request in the manner the exchange expects
func (e *Exchange) reqBuilder(req *WsRequest, sub *subscription.Subscription) error {
	for i := range sub.Pairs {
		form, err := e.GetPairFormat(asset.Spot, true)
		if err != nil {
			return err
		}
		sub.Pairs[i] = sub.Pairs[i].Format(form)
		req.Arguments = append(req.Arguments, WsArgument{
			Channel:        sub.Channel,
			InstrumentType: itemEncoder(sub.Asset),
			InstrumentID:   sub.Pairs[i].String(),
		})
	}
	if len(sub.Pairs) == 0 {
		req.Arguments = append(req.Arguments, WsArgument{
			Channel:        sub.Channel,
			InstrumentType: itemEncoder(sub.Asset),
			Coin:           currency.NewCode("default"),
			InstrumentID:   "default",
		})
	}
	return nil
}

// manageSubs subscribes or unsubscribes from a list of websocket channels
func (e *Exchange) manageSubs(ctx context.Context, conn websocket.Connection, op string, subs subscription.List) error {
	unsplitRequest := &WsRequest{Operation: op}
	for _, s := range subs {
		if err := e.reqBuilder(unsplitRequest, s); err != nil {
			return err
		}
	}
	req := reqSplitter(unsplitRequest)
	var errs common.ErrorCollector
	for i := range req {
		if len(req[i].Arguments) == 0 {
			return fmt.Errorf("%w: no arguments in request", websocket.ErrSubscriptionFailure)
		}
		errs.Go(func() error { return conn.SendJSONMessage(ctx, rateSubscription, req[i]) })
	}
	if err := errs.Collect(); err != nil {
		return err
	}

	if op == "subscribe" {
		return e.Websocket.AddSuccessfulSubscriptions(conn, subs...)
	}
	return e.Websocket.RemoveSubscriptions(conn, subs...)
}

// GetSubscriptionTemplate returns a subscription channel template
func (e *Exchange) GetSubscriptionTemplate(*subscription.Subscription) (*template.Template, error) {
	return template.New("master.tmpl").Funcs(template.FuncMap{"channelName": channelName}).Parse(subTplText)
}

func channelName(s *subscription.Subscription) (string, error) {
	if n, ok := subscriptionNames[s.Asset][s.Channel]; ok {
		return n, nil
	}
	return "", fmt.Errorf("%w: %s", subscription.ErrNotSupported, s.Channel)
}

const subTplText = `
{{ range $asset, $pairs := $.AssetPairs }}
	{{- channelName $.S -}}
	{{- $.AssetSeparator }}
{{- end }}
`
