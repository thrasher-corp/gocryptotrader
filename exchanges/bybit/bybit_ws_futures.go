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
	bybitWebsocketFuturesPublicV2 = "wss://stream.bybit.com/realtime"
)

var defaultFuturesSubscriptionChannels = []string{
	wsInstrument,
	wsOrder200,
	wsTrade,
	wsKlineV2,
}

var defaultFuturesAuthSubscriptionChannels = []string{
	wsWallet,
	wsOrder,
	wsStopOrder,
}

// WsFuturesConnect connects to a Futures websocket feed
func (by *Bybit) WsFuturesConnect() error {
	if !by.Websocket.IsEnabled() || !by.IsEnabled() || !by.IsAssetWebsocketSupported(asset.Futures) || by.CurrencyPairs.IsAssetEnabled(asset.Futures) != nil {
		return errors.New(stream.WebsocketNotEnabled)
	}
	assetWebsocket, err := by.Websocket.GetAssetWebsocket(asset.Futures)
	if err != nil {
		return fmt.Errorf("%w asset type: %v", err, asset.Futures)
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
		log.Debugf(log.ExchangeSys, "%s Connected to %v Websocket.\n", by.Name, asset.Futures)
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	go by.wsFuturesReadData(ctx, cancelFunc, assetWebsocket.Conn, assetWebsocket)
	by.Websocket.SetCanUseAuthenticatedEndpoints(true, asset.Futures)
	if by.Websocket.CanUseAuthenticatedEndpoints() {
		err = by.WsFuturesAuth(ctx, cancelFunc)
		if err != nil {
			by.Websocket.DataHandler <- err
			by.Websocket.SetCanUseAuthenticatedEndpoints(false, asset.Futures)
			return nil
		}
	}
	return nil
}

// wsFuturesReadData gets and passes on websocket messages for processing
func (by *Bybit) wsFuturesReadData(ctx context.Context, cancelFunc context.CancelFunc, wsConn stream.Connection, assetWebsocket *stream.Websocket) {
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

			err := by.wsFuturesHandleData(resp.Raw)
			if err != nil {
				by.Websocket.DataHandler <- err
			}
		}
	}
}

// WsFuturesAuth sends an authentication message to receive auth data
func (by *Bybit) WsFuturesAuth(ctx context.Context, cancelFunc context.CancelFunc) error {
	assetWebsocket, err := by.Websocket.GetAssetWebsocket(asset.Futures)
	if err != nil {
		return fmt.Errorf("%w asset type: %v", err, asset.Futures)
	}
	creds, err := by.GetCredentials(ctx)
	if err != nil {
		return err
	}

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
	return assetWebsocket.Conn.SendJSONMessage(req)
}

// GenerateFuturesDefaultSubscriptions returns channel subscriptions for futures instruments
func (by *Bybit) GenerateFuturesDefaultSubscriptions() ([]stream.ChannelSubscription, error) {
	channels := defaultFuturesSubscriptionChannels

	subscriptions := []stream.ChannelSubscription{}
	pairs, err := by.GetEnabledPairs(asset.Futures)
	if err != nil {
		return nil, err
	}
	pairFormat, err := by.GetPairFormat(asset.Futures, true)
	if err != nil {
		return nil, err
	}
	pairs = pairs.Format(pairFormat)
	for x := range channels {
		switch channels[x] {
		case wsInsurance, wsLiquidation, wsPosition,
			wsExecution, wsOrder, wsStopOrder, wsWallet:
			subscriptions = append(subscriptions, stream.ChannelSubscription{
				Asset:   asset.Futures,
				Channel: channels[x],
			})
		case wsOrder25, wsTrade:
			for p := range pairs {
				subscriptions = append(subscriptions, stream.ChannelSubscription{
					Asset:    asset.Futures,
					Channel:  channels[x],
					Currency: pairs[p],
				})
			}
		case wsKlineV2:
			for p := range pairs {
				subscriptions = append(subscriptions, stream.ChannelSubscription{
					Asset:    asset.Futures,
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
					Asset:    asset.Futures,
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

// GenerateWsFuturesDefaultAuthSubscriptions returns channel subscriptions for futures instruments
func (by *Bybit) GenerateWsFuturesDefaultAuthSubscriptions() ([]stream.ChannelSubscription, error) {
	if !by.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, nil
	}
	channels := defaultFuturesAuthSubscriptionChannels
	subscriptions := []stream.ChannelSubscription{}
	for x := range channels {
		switch channels[x] {
		case wsInsurance, wsLiquidation, wsPosition,
			wsExecution, wsOrder, wsStopOrder, wsWallet:
			subscriptions = append(subscriptions, stream.ChannelSubscription{
				Asset:   asset.Futures,
				Channel: channels[x],
			})
		}
	}
	return subscriptions, nil
}

// SubscribeFutures sends a websocket message to receive data from the channel
func (by *Bybit) SubscribeFutures(channelsToSubscribe []stream.ChannelSubscription) error {
	assetWebsocket, err := by.Websocket.GetAssetWebsocket(asset.Futures)
	if err != nil {
		return fmt.Errorf("%w asset type: %v", err, asset.Futures)
	}
	var errs error
	for i := range channelsToSubscribe {
		var sub WsFuturesReq
		sub.Topic = wsSubscribe

		argStr := formatArgs(channelsToSubscribe[i].Channel, channelsToSubscribe[i].Params)
		switch channelsToSubscribe[i].Channel {
		case wsOrder25, wsKlineV2, wsInstrument, wsOrder200, wsTrade:
			var formattedPair currency.Pair
			formattedPair, err = by.FormatExchangeCurrency(channelsToSubscribe[i].Currency, channelsToSubscribe[i].Asset)
			if err != nil {
				errs = common.AppendError(errs, err)
				continue
			}
			argStr += dot + formattedPair.String()
		}
		sub.Args = append(sub.Args, argStr)

		err = assetWebsocket.Conn.SendJSONMessage(sub)
		if err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		assetWebsocket.AddSuccessfulSubscriptions(channelsToSubscribe[i])
	}
	return errs
}

// UnsubscribeFutures sends a websocket message to stop receiving data from the channel
func (by *Bybit) UnsubscribeFutures(channelsToUnsubscribe []stream.ChannelSubscription) error {
	assetWebsocket, err := by.Websocket.GetAssetWebsocket(asset.Futures)
	if err != nil {
		return fmt.Errorf("%w asset type: %v", err, asset.Futures)
	}
	var errs error
	for i := range channelsToUnsubscribe {
		var unSub WsFuturesReq
		unSub.Topic = wsUnsubscribe

		formattedPair, err := by.FormatExchangeCurrency(channelsToUnsubscribe[i].Currency, asset.Futures)
		if err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		unSub.Args = append(unSub.Args, channelsToUnsubscribe[i].Channel+dot+formattedPair.String())
		err = assetWebsocket.Conn.SendJSONMessage(sub)
		if err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		assetWebsocket.RemoveSuccessfulUnsubscriptions(channelsToUnsubscribe[i])
	}
	return errs
}

// wsFuturesHandleResp handles response messages from ws requests
func (by *Bybit) wsFuturesHandleResp(wsFuturesResp *WsFuturesResp) error {
	switch wsFuturesResp.Request.Topic {
	case wsAuth:
		if !wsFuturesResp.Success {
			assetWebsocket, err := by.Websocket.GetAssetWebsocket(asset.Futures)
			if err != nil {
				return fmt.Errorf("%w asset type: %v", err, asset.Futures)
			}
			switch wsFuturesResp.RetMsg {
			case "error:request expired":
				// Consider handling the error by sending the authentication message again
				log.Errorf(log.ExchangeSys, "%s Asset Type %v Authentication request expired: %v", by.Name, asset.Futures, wsFuturesResp.RetMsg)
			default:
				log.Errorf(log.ExchangeSys, "%s Asset Type %v Authentication failed with message: %v - disabling authenticated endpoint", by.Name, asset.Futures, wsFuturesResp.RetMsg)
			}
			assetWebsocket.SetCanUseAuthenticatedEndpoints(false)
			return nil
		}
		authSubs, err := by.GenerateWsFuturesDefaultAuthSubscriptions()
		if err != nil {
			return err
		}
		err = by.SubscribeFutures(authSubs)
		if err != nil {
			return err
		}
		if by.Verbose {
			log.Debugf(log.ExchangeSys, "%s Asset Type %v Authentication succeeded", by.Name, asset.Futures)
			return nil
		}
	case wsSubscribe:
		if !wsFuturesResp.Success {
			log.Errorf(log.ExchangeSys, "%s Asset Type %v Subscription failed: %v", by.Name, asset.Futures, wsFuturesResp.Request.Args)
		}
	case wsUnsubscribe:
		if !wsFuturesResp.Success {
			log.Errorf(log.ExchangeSys, "%s Asset Type %v Unsubscription failed: %v", by.Name, asset.Futures, wsFuturesResp.Request.Args)
		}
	default:
		log.Errorf(log.ExchangeSys, "%s Asset Type %v Unhandled response", by.Name, asset.Futures)
	}
	return nil
}

func (by *Bybit) wsFuturesHandleData(respRaw []byte) error {
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
				log.Warnf(log.ExchangeSys, "%s Asset Type %v Received unhandled message on websocket: %v\n", by.Name, asset.Futures, multiStreamData)
			}
			return nil
		}
		return by.wsFuturesHandleResp(&wsFuturesResp)
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
				p, err = by.extractCurrencyPair(response.Data[0].Symbol, asset.Futures)
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
					asset.Futures)
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
					p, err = by.extractCurrencyPair(response.OBData.Delete[0].Symbol, asset.Futures)
					if err != nil {
						return err
					}

					err = by.processOrderbook(response.OBData.Delete,
						wsOrderbookActionDelete,
						p,
						asset.Futures)
					if err != nil {
						return err
					}
				}

				if len(response.OBData.Update) > 0 {
					var p currency.Pair
					p, err = by.extractCurrencyPair(response.OBData.Update[0].Symbol, asset.Futures)
					if err != nil {
						return err
					}

					err = by.processOrderbook(response.OBData.Update,
						wsOrderbookActionUpdate,
						p,
						asset.Futures)
					if err != nil {
						return err
					}
				}

				if len(response.OBData.Insert) > 0 {
					var p currency.Pair
					p, err = by.extractCurrencyPair(response.OBData.Insert[0].Symbol, asset.Futures)
					if err != nil {
						return err
					}

					err = by.processOrderbook(response.OBData.Insert,
						wsOrderbookActionInsert,
						p,
						asset.Futures)
					if err != nil {
						return err
					}
				}
			default:
				by.Websocket.DataHandler <- stream.UnhandledMessageWarning{Message: by.Name + stream.UnhandledMessage + "unsupported orderbook operation"}
			}
		}

	case wsTrades:
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
			p, err = by.extractCurrencyPair(response.TradeData[0].Symbol, asset.Futures)
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
				AssetType:    asset.Futures,
				Side:         oSide,
				Price:        response.TradeData[i].Price.Float64(),
				Amount:       response.TradeData[i].Size,
				Timestamp:    response.TradeData[i].Time,
			}
			counter++
		}
		return by.AddTradesToBuffer(trades...)

	case wsKlineV2:
		var response WsFuturesKline
		err = json.Unmarshal(respRaw, &response)
		if err != nil {
			return err
		}

		var p currency.Pair
		p, err = by.extractCurrencyPair(topics[len(topics)-1], asset.Futures)
		if err != nil {
			return err
		}

		for i := range response.KlineData {
			by.Websocket.DataHandler <- stream.KlineData{
				Pair:       p,
				AssetType:  asset.Futures,
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
				var response WsTicker
				err = json.Unmarshal(respRaw, &response)
				if err != nil {
					return err
				}

				var p currency.Pair
				p, err = by.extractCurrencyPair(response.Ticker.Symbol, asset.Futures)
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
					Volume:       response.Ticker.Volume24h,
					Close:        response.Ticker.PrevPrice24h.Float64(),
					LastUpdated:  response.Ticker.UpdateAt,
					AssetType:    asset.Futures,
					Pair:         p,
				}

			case wsOperationDelta:
				var response WsDeltaTicker
				err = json.Unmarshal(respRaw, &response)
				if err != nil {
					return err
				}

				if len(response.Data.Update) > 0 {
					for x := range response.Data.Update {
						if response.Data.Update[x] == (WsTickerData{}) {
							continue
						}
						var p currency.Pair
						p, err = by.extractCurrencyPair(response.Data.Update[x].Symbol, asset.Futures)
						if err != nil {
							return err
						}
						var tickerData *ticker.Price
						tickerData, err = by.FetchTicker(context.Background(), p, asset.Futures)
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
							Volume:       compareAndSet(tickerData.Volume, response.Data.Update[x].Volume24h),
							Close:        compareAndSet(tickerData.Close, response.Data.Update[x].PrevPrice24h.Float64()),
							LastUpdated:  response.Data.Update[x].UpdateAt,
							AssetType:    asset.Futures,
							Pair:         p,
						}
					}
				}
			default:
				by.Websocket.DataHandler <- stream.UnhandledMessageWarning{Message: by.Name + stream.UnhandledMessage + "unsupported ticker operation"}
			}
		}

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
			p, err = by.extractCurrencyPair(response.Data[i].Symbol, asset.Futures)
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
				AssetType: asset.Futures,
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
			p, err = by.extractCurrencyPair(response.Data[x].Symbol, asset.Futures)
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
				AssetType: asset.Futures,
				Date:      response.Data[x].Time,
				Pair:      p,
				Trades: []order.TradeHistory{
					{
						Price:     response.Data[x].Price.Float64(),
						Amount:    response.Data[x].OrderQty,
						Exchange:  by.Name,
						Side:      oSide,
						Timestamp: response.Data[x].Time,
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
			p, err = by.extractCurrencyPair(response.Data[x].Symbol, asset.Futures)
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
				AssetType: asset.Futures,
				Date:      response.Data[x].Time,
				Pair:      p,
				Trades: []order.TradeHistory{
					{
						Price:     response.Data[x].Price.Float64(),
						Amount:    response.Data[x].OrderQty,
						Exchange:  by.Name,
						Side:      oSide,
						Timestamp: response.Data[x].Time,
					},
				},
			}
		}

	default:
		by.Websocket.DataHandler <- stream.UnhandledMessageWarning{Message: by.Name + stream.UnhandledMessage + string(respRaw)}
	}

	return nil
}
