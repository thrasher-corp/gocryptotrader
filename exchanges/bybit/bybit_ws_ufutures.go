package bybit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	// public endpoints
	wsUSDTKline = "candle"

	bybitWebsocketUSDTMarginedFuturesPublicV2  = "wss://stream.bybit.com/realtime_public"
	bybitWebsocketUSDTMarginedFuturesPrivateV2 = "wss://stream.bybit.com/realtime_private"
)

var defaultUSDTMarginedFuturesSubscriptionChannels = []string{
	wsInstrument,
	wsOrder200,
	wsTrade,
	wsUSDTKline,
}

var defaultUSDTMarginedFuturesAuthSubscriptionChannels = []string{
	wsWallet,
	wsOrder,
	wsStopOrder,
}

// WsUSDTConnect connects to USDTMarginedFutures CMF websocket feed
func (by *Bybit) WsUSDTConnect() error {
	if !by.Websocket.IsEnabled() || !by.IsEnabled() || !by.IsAssetWebsocketSupported(asset.USDTMarginedFutures) || by.CurrencyPairs.IsAssetEnabled(asset.USDTMarginedFutures) != nil {
		return errors.New(stream.WebsocketNotEnabled)
	}
	assetWebsocket, err := by.Websocket.GetAssetWebsocket(asset.USDTMarginedFutures)
	if err != nil {
		return fmt.Errorf("%w asset type: %v", err, asset.USDTMarginedFutures)
	}
	var dialer websocket.Dialer
	err = assetWebsocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}

	pingMsg, err := json.Marshal(pingRequest)
	if err != nil {
		return err
	}
	assetWebsocket.Conn.SetupPingHandler(stream.PingHandler{
		Message:     pingMsg,
		MessageType: websocket.PingMessage,
		Delay:       bybitWebsocketTimer,
	})
	if by.Verbose {
		log.Debugf(log.ExchangeSys, "%s Connected to %v Websocket.\n", by.Name, asset.USDTMarginedFutures)
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	go by.wsUSDTReadData(ctx, cancelFunc, assetWebsocket.Conn, assetWebsocket)
	by.Websocket.SetCanUseAuthenticatedEndpoints(true, asset.USDTMarginedFutures)
	if by.Websocket.CanUseAuthenticatedEndpoints() {
		err = by.WsUSDTAuth(ctx, cancelFunc)
		if err != nil {
			by.Websocket.DataHandler <- err
			by.Websocket.SetCanUseAuthenticatedEndpoints(false, asset.USDTMarginedFutures)
			return nil
		}
	}
	return nil
}

// WsUSDTAuth sends an authentication message to receive auth data
func (by *Bybit) WsUSDTAuth(ctx context.Context, cancelFunc context.CancelFunc) error {
	assetWebsocket, err := by.Websocket.GetAssetWebsocket(asset.USDTMarginedFutures)
	if err != nil {
		return fmt.Errorf("%w asset type: %v", err, asset.USDTMarginedFutures)
	}
	creds, err := by.GetCredentials(ctx)
	if err != nil {
		return err
	}
	var dialer websocket.Dialer
	err = assetWebsocket.AuthConn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	go by.wsUSDTReadData(ctx, cancelFunc, assetWebsocket.AuthConn, assetWebsocket)

	intNonce := (time.Now().Unix() + 2) * 1000
	strNonce := strconv.FormatInt(intNonce, 10)
	hmac, err := crypto.GetHMAC(
		crypto.HashSHA256,
		[]byte("GET/realtime"+strNonce),
		[]byte(creds.Secret),
	)
	if err != nil {
		return err
	}
	sign := crypto.HexEncodeToString(hmac)
	req := Authenticate{
		Operation: "auth",
		Args:      []interface{}{creds.Key, intNonce, sign},
	}
	return assetWebsocket.AuthConn.SendJSONMessage(req)
}

// GenerateUSDTMarginedFuturesDefaultSubscriptions returns channel subscriptions for futures instruments
func (by *Bybit) GenerateUSDTMarginedFuturesDefaultSubscriptions() ([]stream.ChannelSubscription, error) {
	channels := defaultUSDTMarginedFuturesSubscriptionChannels
	subscriptions := []stream.ChannelSubscription{}
	pairs, err := by.GetEnabledPairs(asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	pairFormat, err := by.GetPairFormat(asset.USDTMarginedFutures, true)
	if err != nil {
		return nil, err
	}
	pairs = pairs.Format(pairFormat)
	for x := range channels {
		switch channels[x] {
		case wsInsurance, wsLiquidation, wsPosition,
			wsExecution, wsOrder, wsStopOrder, wsWallet:
			subscriptions = append(subscriptions, stream.ChannelSubscription{
				Asset:   asset.USDTMarginedFutures,
				Channel: channels[x],
			})
		case wsOrder25, wsTrade:
			for p := range pairs {
				subscriptions = append(subscriptions, stream.ChannelSubscription{
					Asset:    asset.USDTMarginedFutures,
					Channel:  channels[x],
					Currency: pairs[p],
				})
			}
		case wsKlineV2, wsUSDTKline:
			for p := range pairs {
				subscriptions = append(subscriptions, stream.ChannelSubscription{
					Asset:    asset.USDTMarginedFutures,
					Channel:  channels[x],
					Currency: pairs[p],
					Params: map[string]interface{}{
						"interval": "1",
					},
				})
			}
		case wsInstrument, wsOrder200:
			for p := range pairs {
				subscriptions = append(subscriptions, stream.ChannelSubscription{
					Asset:    asset.USDTMarginedFutures,
					Channel:  channels[x],
					Currency: pairs[p],
					Params: map[string]interface{}{
						"frequency_interval": "100ms",
					},
				})
			}
		}
	}
	return subscriptions, nil
}

// GenerateWsUSDTDefaultAuthSubscriptions returns channel subscriptions for futures instruments
func (by *Bybit) GenerateWsUSDTDefaultAuthSubscriptions() ([]stream.ChannelSubscription, error) {
	if !by.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, nil
	}
	channels := defaultUSDTMarginedFuturesAuthSubscriptionChannels
	subscriptions := []stream.ChannelSubscription{}
	for x := range channels {
		switch channels[x] {
		case wsInsurance, wsLiquidation, wsPosition,
			wsExecution, wsOrder, wsStopOrder, wsWallet:
			subscriptions = append(subscriptions, stream.ChannelSubscription{
				Asset:   asset.USDTMarginedFutures,
				Channel: channels[x],
			})
		}
	}
	return subscriptions, nil
}

// SubscribeUSDT sends a websocket message to receive data from the channel
func (by *Bybit) SubscribeUSDT(channelsToSubscribe []stream.ChannelSubscription) error {
	assetWebsocket, err := by.Websocket.GetAssetWebsocket(asset.USDTMarginedFutures)
	if err != nil {
		return fmt.Errorf("%w asset type: %v", err, asset.USDTMarginedFutures)
	}
	var errs error
	for i := range channelsToSubscribe {
		var sub WsFuturesReq
		sub.Topic = wsSubscribe

		argStr := formatArgs(channelsToSubscribe[i].Channel, channelsToSubscribe[i].Params)
		switch channelsToSubscribe[i].Channel {
		case wsOrder25, wsKlineV2, wsUSDTKline, wsInstrument, wsOrder200, wsTrade:
			var formattedPair currency.Pair
			formattedPair, err = by.FormatExchangeCurrency(channelsToSubscribe[i].Currency, channelsToSubscribe[i].Asset)
			if err != nil {
				errs = common.AppendError(errs, err)
				continue
			}
			argStr += dot + formattedPair.String()
		}
		sub.Args = append(sub.Args, argStr)

		switch channelsToSubscribe[i].Channel {
		case wsPosition, wsExecution, wsOrder, wsStopOrder, wsWallet:
			err = assetWebsocket.AuthConn.SendJSONMessage(sub)
		default:
			err = assetWebsocket.Conn.SendJSONMessage(sub)
		}
		if err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		assetWebsocket.AddSuccessfulSubscriptions(channelsToSubscribe[i])
	}
	return errs
}

// UnsubscribeUSDT sends asset.USDTMarginedFutures websocket message to stop receiving data from the channel
func (by *Bybit) UnsubscribeUSDT(channelsToUnsubscribe []stream.ChannelSubscription) error {
	assetWebsocket, err := by.Websocket.GetAssetWebsocket(asset.USDTMarginedFutures)
	if err != nil {
		return fmt.Errorf("%w asset type: %v", err, asset.USDTMarginedFutures)
	}
	var errs error
	for i := range channelsToUnsubscribe {
		var unSub WsFuturesReq
		unSub.Topic = wsUnsubscribe

		formattedPair, err := by.FormatExchangeCurrency(channelsToUnsubscribe[i].Currency, asset.USDTMarginedFutures)
		if err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		unSub.Args = append(unSub.Args, channelsToUnsubscribe[i].Channel+dot+formattedPair.String())

		switch channelsToUnsubscribe[i].Channel {
		case wsPosition, wsExecution, wsOrder, wsStopOrder, wsWallet:
			err = assetWebsocket.AuthConn.SendJSONMessage(sub)
		default:
			err = assetWebsocket.Conn.SendJSONMessage(sub)
		}
		if err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		assetWebsocket.RemoveSuccessfulUnsubscriptions(channelsToUnsubscribe[i])
	}
	return errs
}

// wsUSDTReadData gets and passes on websocket messages for processing
func (by *Bybit) wsUSDTReadData(ctx context.Context, cancelFunc context.CancelFunc, wsConn stream.Connection, assetWebsocket *stream.Websocket) {
	assetWebsocket.Wg.Add(1)
	defer func() {
		assetWebsocket.Wg.Done()
	}()
	for {
		select {
		case <-ctx.Done():
			// received termination signal
			return
		case <-assetWebsocket.ShutdownC:
			return
		default:
			resp := wsConn.ReadMessage()
			if resp.Raw == nil {
				cancelFunc()
				return
			}

			err := by.wsUSDTHandleData(resp.Raw)
			if err != nil {
				by.Websocket.DataHandler <- err
			}
		}
	}
}

// wsUSDTHandleResp handles response messages from ws requests
func (by *Bybit) wsUSDTHandleResp(wsFuturesResp *WsFuturesResp) error {
	switch wsFuturesResp.Request.Topic {
	case wsAuth:
		assetWebsocket, err := by.Websocket.GetAssetWebsocket(asset.USDTMarginedFutures)
		if err != nil {
			return fmt.Errorf("%w asset type: %v", err, asset.USDTMarginedFutures)
		}
		if !wsFuturesResp.Success {
			switch wsFuturesResp.RetMsg {
			case "error:request expired":
				log.Errorf(log.ExchangeSys, "%s Asset Type %v Authentication request expired: %v", by.Name, asset.USDTMarginedFutures, wsFuturesResp.RetMsg)
				return nil
			default:
				log.Errorf(log.ExchangeSys, "%s Asset Type %v Authentication failed with message: %v - disabling authenticated endpoint", by.Name, asset.USDTMarginedFutures, wsFuturesResp.RetMsg)
				assetWebsocket.SetCanUseAuthenticatedEndpoints(false)
				return nil
			}
		}
		pingMsg, err := json.Marshal(pingRequest)
		if err != nil {
			return err
		}
		assetWebsocket.AuthConn.SetupPingHandler(stream.PingHandler{
			Message:     pingMsg,
			MessageType: websocket.PingMessage,
			Delay:       bybitWebsocketTimer,
		})
		authSubs, err := by.GenerateWsUSDTDefaultAuthSubscriptions()
		if err != nil {
			return err
		}
		err = by.SubscribeUSDT(authSubs)
		if err != nil {
			return err
		}
		if by.Verbose {
			log.Debugf(log.ExchangeSys, "%s Asset Type %v Authentication succeeded", by.Name, asset.USDTMarginedFutures)
			return nil
		}
	case wsSubscribe:
		if !wsFuturesResp.Success {
			log.Errorf(log.ExchangeSys, "%s Asset Type %v Subscription failed: %v", by.Name, asset.USDTMarginedFutures, wsFuturesResp.Request.Args)
		}
	case wsUnsubscribe:
		if !wsFuturesResp.Success {
			log.Errorf(log.ExchangeSys, "%s Asset Type %v Unsubscription failed: %v", by.Name, asset.USDTMarginedFutures, wsFuturesResp.Request.Args)
		}
	default:
		log.Errorf(log.ExchangeSys, "%s Asset Type %v Unhandled response", by.Name, asset.USDTMarginedFutures)
	}
	return nil
}

// wsUSDTHandleData will read websocket raw data and pass to appropriate handler
func (by *Bybit) wsUSDTHandleData(respRaw []byte) error {
	var multiStreamData map[string]interface{}
	err := json.Unmarshal(respRaw, &multiStreamData)
	if err != nil {
		return err
	}

	t, ok := multiStreamData["topic"].(string)
	if !ok {
		var wsFuturesResp WsFuturesResp
		err = json.Unmarshal(respRaw, &wsFuturesResp)
		if err != nil {
			if by.Verbose {
				log.Warnf(log.ExchangeSys, "%s Asset Type %v Received unhandled message on websocket: %v\n", by.Name, asset.USDTMarginedFutures, multiStreamData)
			}
			return nil
		}
		return by.wsUSDTHandleResp(&wsFuturesResp)
	}

	topics := strings.Split(t, dot)
	if len(topics) < 1 {
		return errors.New(by.Name + " - topic could not be extracted from response")
	}

	switch topics[0] {
	case wsOrder25, wsOrder200:
		if wsType, ok := multiStreamData["type"].(string); ok {
			switch wsType {
			case wsOperationSnapshot:
				var response WsFuturesOrderbook
				err = json.Unmarshal(respRaw, &response)
				if err != nil {
					return err
				}

				var p currency.Pair
				p, err = by.extractCurrencyPair(response.Data[0].Symbol, asset.USDTMarginedFutures)
				if err != nil {
					return err
				}

				var format currency.PairFormat
				format, err = by.GetPairFormat(asset.Futures, false)
				if err != nil {
					return err
				}
				p = p.Format(format)
				if err != nil {
					return err
				}
				err = by.processOrderbook(response.Data,
					wsOperationSnapshot,
					p,
					asset.USDTMarginedFutures)
				if err != nil {
					return err
				}

			case wsOperationDelta:
				var response WsFuturesDeltaOrderbook
				err = json.Unmarshal(respRaw, &response)
				if err != nil {
					return err
				}

				if len(response.OBData.Delete) > 0 {
					var p currency.Pair
					p, err = by.extractCurrencyPair(response.OBData.Delete[0].Symbol, asset.USDTMarginedFutures)
					if err != nil {
						return err
					}

					err = by.processOrderbook(response.OBData.Delete,
						wsOrderbookActionDelete,
						p,
						asset.USDTMarginedFutures)
					if err != nil {
						return err
					}
				}

				if len(response.OBData.Update) > 0 {
					var p currency.Pair
					p, err = by.extractCurrencyPair(response.OBData.Update[0].Symbol, asset.USDTMarginedFutures)
					if err != nil {
						return err
					}

					err = by.processOrderbook(response.OBData.Update,
						wsOrderbookActionUpdate,
						p,
						asset.USDTMarginedFutures)
					if err != nil {
						return err
					}
				}

				if len(response.OBData.Insert) > 0 {
					var p currency.Pair
					p, err = by.extractCurrencyPair(response.OBData.Insert[0].Symbol, asset.USDTMarginedFutures)
					if err != nil {
						return err
					}

					err = by.processOrderbook(response.OBData.Insert,
						wsOrderbookActionInsert,
						p,
						asset.USDTMarginedFutures)
					if err != nil {
						return err
					}
				}
			default:
				by.Websocket.DataHandler <- stream.UnhandledMessageWarning{Message: by.Name + stream.UnhandledMessage + "unsupported orderbook operation"}
			}
		}

	case wsTrade:
		if !by.IsSaveTradeDataEnabled() {
			return nil
		}
		var response WsFuturesTrade
		err = json.Unmarshal(respRaw, &response)
		if err != nil {
			return err
		}
		counter := 0
		trades := make([]trade.Data, len(response.TradeData))
		for i := range response.TradeData {
			var p currency.Pair
			p, err = by.extractCurrencyPair(response.TradeData[0].Symbol, asset.USDTMarginedFutures)
			if err != nil {
				return err
			}

			var oSide order.Side
			oSide, err = order.StringToOrderSide(response.TradeData[i].Side)
			if err != nil {
				by.Websocket.DataHandler <- order.ClassificationError{
					Exchange: by.Name,
					Err:      err,
				}
				continue
			}

			trades[counter] = trade.Data{
				TID:          response.TradeData[i].ID,
				Exchange:     by.Name,
				CurrencyPair: p,
				AssetType:    asset.USDTMarginedFutures,
				Side:         oSide,
				Price:        response.TradeData[i].Price.Float64(),
				Amount:       response.TradeData[i].Size,
				Timestamp:    response.TradeData[i].Time,
			}
			counter++
		}
		return by.AddTradesToBuffer(trades...)

	case wsKlineV2, wsUSDTKline:
		var response WsFuturesKline
		err = json.Unmarshal(respRaw, &response)
		if err != nil {
			return err
		}

		var p currency.Pair
		p, err = by.extractCurrencyPair(topics[len(topics)-1], asset.USDTMarginedFutures)
		if err != nil {
			return err
		}

		for i := range response.KlineData {
			by.Websocket.DataHandler <- stream.KlineData{
				Pair:       p,
				AssetType:  asset.USDTMarginedFutures,
				Exchange:   by.Name,
				OpenPrice:  response.KlineData[i].Open.Float64(),
				HighPrice:  response.KlineData[i].High.Float64(),
				LowPrice:   response.KlineData[i].Low.Float64(),
				ClosePrice: response.KlineData[i].Close.Float64(),
				Volume:     response.KlineData[i].Volume.Float64(),
				Timestamp:  response.KlineData[i].Timestamp.Time(),
			}
		}

	case wsInstrument:
		if wsType, ok := multiStreamData["type"].(string); ok {
			switch wsType {
			case wsOperationSnapshot:
				var response WsFuturesTicker
				err = json.Unmarshal(respRaw, &response)
				if err != nil {
					return err
				}

				var p currency.Pair
				p, err = by.extractCurrencyPair(response.Ticker.Symbol, asset.USDTMarginedFutures)
				if err != nil {
					return err
				}

				by.Websocket.DataHandler <- &ticker.Price{
					ExchangeName: by.Name,
					Last:         response.Ticker.LastPrice.Float64(),
					High:         response.Ticker.HighPrice24h.Float64(),
					Low:          response.Ticker.LowPrice24h.Float64(),
					Bid:          response.Ticker.BidPrice.Float64(),
					Ask:          response.Ticker.AskPrice.Float64(),
					Volume:       response.Ticker.GetVolume24h(),
					Close:        response.Ticker.PrevPrice24h.Float64(),
					LastUpdated:  response.Ticker.UpdateAt,
					AssetType:    asset.USDTMarginedFutures,
					Pair:         p,
				}

			case wsOperationDelta:
				var response WsDeltaFuturesTicker
				err = json.Unmarshal(respRaw, &response)
				if err != nil {
					return err
				}

				if len(response.Data.Update) > 0 {
					for x := range response.Data.Update {
						if response.Data.Update[x] == (WsFuturesTickerData{}) {
							continue
						}
						var p currency.Pair
						p, err = by.extractCurrencyPair(response.Data.Update[x].Symbol, asset.USDTMarginedFutures)
						if err != nil {
							return err
						}
						var tickerData *ticker.Price
						tickerData, err = by.FetchTicker(context.Background(), p, asset.USDTMarginedFutures)
						if err != nil {
							return err
						}
						by.Websocket.DataHandler <- &ticker.Price{
							ExchangeName: by.Name,
							Last:         compareAndSet(tickerData.Last, response.Data.Update[x].LastPrice.Float64()),
							High:         compareAndSet(tickerData.High, response.Data.Update[x].HighPrice24h.Float64()),
							Low:          compareAndSet(tickerData.Low, response.Data.Update[x].LowPrice24h.Float64()),
							Bid:          compareAndSet(tickerData.Bid, response.Data.Update[x].BidPrice.Float64()),
							Ask:          compareAndSet(tickerData.Ask, response.Data.Update[x].AskPrice.Float64()),
							Volume:       compareAndSet(tickerData.Volume, response.Data.Update[x].GetVolume24h()),
							Close:        compareAndSet(tickerData.Close, response.Data.Update[x].PrevPrice24h.Float64()),
							LastUpdated:  response.Data.Update[x].UpdateAt,
							AssetType:    asset.USDTMarginedFutures,
							Pair:         p,
						}
					}
				}
			default:
				by.Websocket.DataHandler <- stream.UnhandledMessageWarning{Message: by.Name + stream.UnhandledMessage + "unsupported ticker operation"}
			}
		}

	case wsLiquidation:
		var response WsFuturesLiquidation
		err = json.Unmarshal(respRaw, &response)
		if err != nil {
			return err
		}
		by.Websocket.DataHandler <- response.Data

	case wsInsurance:
		var response WsInsurance
		err = json.Unmarshal(respRaw, &response)
		if err != nil {
			return err
		}
		by.Websocket.DataHandler <- response.Data

	case wsPosition:
		var response WsFuturesPosition
		err = json.Unmarshal(respRaw, &response)
		if err != nil {
			return err
		}
		by.Websocket.DataHandler <- response.Data

	case wsExecution:
		var response WsFuturesExecution
		err = json.Unmarshal(respRaw, &response)
		if err != nil {
			return err
		}

		for i := range response.Data {
			var p currency.Pair
			p, err = by.extractCurrencyPair(response.Data[i].Symbol, asset.USDTMarginedFutures)
			if err != nil {
				return err
			}

			var oSide order.Side
			oSide, err = order.StringToOrderSide(response.Data[i].Side)
			if err != nil {
				by.Websocket.DataHandler <- order.ClassificationError{
					Exchange: by.Name,
					OrderID:  response.Data[i].OrderID,
					Err:      err,
				}
			}

			var oStatus order.Status
			oStatus, err = order.StringToOrderStatus(response.Data[i].ExecutionType)
			if err != nil {
				by.Websocket.DataHandler <- order.ClassificationError{
					Exchange: by.Name,
					OrderID:  response.Data[i].OrderID,
					Err:      err,
				}
			}

			by.Websocket.DataHandler <- &order.Detail{
				Exchange:  by.Name,
				OrderID:   response.Data[i].OrderID,
				AssetType: asset.USDTMarginedFutures,
				Pair:      p,
				Price:     response.Data[i].Price.Float64(),
				Amount:    response.Data[i].OrderQty,
				Side:      oSide,
				Status:    oStatus,
				Trades: []order.TradeHistory{
					{
						Price:     response.Data[i].Price.Float64(),
						Amount:    response.Data[i].OrderQty,
						Exchange:  by.Name,
						Side:      oSide,
						Timestamp: response.Data[i].Time,
						TID:       response.Data[i].ExecutionID,
						IsMaker:   response.Data[i].IsMaker,
					},
				},
			}
		}

	case wsOrder:
		var response WsOrder
		err = json.Unmarshal(respRaw, &response)
		if err != nil {
			return err
		}
		for x := range response.Data {
			var p currency.Pair
			p, err = by.extractCurrencyPair(response.Data[x].Symbol, asset.USDTMarginedFutures)
			if err != nil {
				return err
			}
			var oSide order.Side
			oSide, err = order.StringToOrderSide(response.Data[x].Side)
			if err != nil {
				by.Websocket.DataHandler <- order.ClassificationError{
					Exchange: by.Name,
					OrderID:  response.Data[x].OrderID,
					Err:      err,
				}
			}
			var oType order.Type
			oType, err = order.StringToOrderType(response.Data[x].OrderType)
			if err != nil {
				by.Websocket.DataHandler <- order.ClassificationError{
					Exchange: by.Name,
					OrderID:  response.Data[x].OrderID,
					Err:      err,
				}
			}
			var oStatus order.Status
			oStatus, err = order.StringToOrderStatus(response.Data[x].OrderStatus)
			if err != nil {
				by.Websocket.DataHandler <- order.ClassificationError{
					Exchange: by.Name,
					OrderID:  response.Data[x].OrderID,
					Err:      err,
				}
			}
			by.Websocket.DataHandler <- &order.Detail{
				Price:     response.Data[x].Price.Float64(),
				Amount:    response.Data[x].OrderQty,
				Exchange:  by.Name,
				OrderID:   response.Data[x].OrderID,
				Type:      oType,
				Side:      oSide,
				Status:    oStatus,
				AssetType: asset.USDTMarginedFutures,
				Date:      response.Data[x].GetTime(asset.USDTMarginedFutures),
				Pair:      p,
				Trades: []order.TradeHistory{
					{
						Price:     response.Data[x].Price.Float64(),
						Amount:    response.Data[x].OrderQty,
						Exchange:  by.Name,
						Side:      oSide,
						Timestamp: response.Data[x].GetTime(asset.USDTMarginedFutures),
					},
				},
			}
		}

	case wsStopOrder:
		var response WsFuturesStopOrder
		err = json.Unmarshal(respRaw, &response)
		if err != nil {
			return err
		}
		for x := range response.Data {
			var p currency.Pair
			p, err = by.extractCurrencyPair(response.Data[x].Symbol, asset.USDTMarginedFutures)
			if err != nil {
				return err
			}
			var oSide order.Side
			oSide, err = order.StringToOrderSide(response.Data[x].Side)
			if err != nil {
				by.Websocket.DataHandler <- order.ClassificationError{
					Exchange: by.Name,
					OrderID:  response.Data[x].OrderID,
					Err:      err,
				}
			}
			var oType order.Type
			oType, err = order.StringToOrderType(response.Data[x].OrderType)
			if err != nil {
				by.Websocket.DataHandler <- order.ClassificationError{
					Exchange: by.Name,
					OrderID:  response.Data[x].OrderID,
					Err:      err,
				}
			}
			var oStatus order.Status
			oStatus, err = order.StringToOrderStatus(response.Data[x].OrderStatus)
			if err != nil {
				by.Websocket.DataHandler <- order.ClassificationError{
					Exchange: by.Name,
					OrderID:  response.Data[x].OrderID,
					Err:      err,
				}
			}
			by.Websocket.DataHandler <- &order.Detail{
				Price:     response.Data[x].Price.Float64(),
				Amount:    response.Data[x].OrderQty,
				Exchange:  by.Name,
				OrderID:   response.Data[x].OrderID,
				AccountID: strconv.FormatInt(response.Data[x].UserID, 10),
				Type:      oType,
				Side:      oSide,
				Status:    oStatus,
				AssetType: asset.USDTMarginedFutures,
				Date:      response.Data[x].GetTime(asset.USDTMarginedFutures),
				Pair:      p,
				Trades: []order.TradeHistory{
					{
						Price:     response.Data[x].Price.Float64(),
						Amount:    response.Data[x].OrderQty,
						Exchange:  by.Name,
						Side:      oSide,
						Timestamp: response.Data[x].GetTime(asset.USDTMarginedFutures),
					},
				},
			}
		}

	case wsWallet:
		var response WsFuturesWallet
		err = json.Unmarshal(respRaw, &response)
		if err != nil {
			return err
		}
		by.Websocket.DataHandler <- response.Data

	default:
		by.Websocket.DataHandler <- stream.UnhandledMessageWarning{Message: by.Name + stream.UnhandledMessage + string(respRaw)}
	}
	return nil
}
