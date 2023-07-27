package bybit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	bybitWSBaseURL      = "wss://stream.bybit.com/"
	wsSpotPublicTopicV2 = "spot/quote/ws/v2"
	wsSpotPrivate       = "spot/ws"
	bybitWebsocketTimer = 20 * time.Second
	wsOrderbook         = "depth"
	wsTicker            = "bookTicker"
	wsRealtimes         = "realtimes"
	wsKlines            = "kline"

	wsAccountInfo    = "outboundAccountInfo"
	wsOrderExecution = "executionReport"
	wsTicketInfo     = "ticketInfo"

	sub    = "sub"    // event for subscribe
	cancel = "cancel" // event for unsubscribe
)

var defaultSpotSubscriptionChannels = []string{
	wsTicker,
	//	wsRealtimes,
	wsOrderbook,
	wsKlines,
}

// WsSpotConnect connects to a websocket feed
func (by *Bybit) WsSpotConnect() error {
	if !by.Websocket.IsEnabled() || !by.IsEnabled() || !by.IsAssetWebsocketSupported(asset.Spot) || by.CurrencyPairs.IsAssetEnabled(asset.Spot) != nil {
		return errors.New(stream.WebsocketNotEnabled)
	}
	assetWebsocket, err := by.Websocket.GetAssetWebsocket(asset.Spot)
	if err != nil {
		return fmt.Errorf("%w asset type: %v", err, asset.Spot)
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
		log.Debugf(log.ExchangeSys, "%s Connected to %v Websocket.\n", by.Name, asset.Spot)
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	go by.wsSpotReadData(ctx, cancelFunc, assetWebsocket.Conn, assetWebsocket)
	if by.Websocket.CanUseAuthenticatedEndpoints() {
		err = by.WsSpotAuth(ctx, cancelFunc)
		if err != nil {
			by.Websocket.DataHandler <- err
			by.Websocket.SetCanUseAuthenticatedEndpoints(false, asset.Spot)
			return nil
		}
	}
	return nil
}

// WsSpotAuth sends an authentication message to receive auth data
func (by *Bybit) WsSpotAuth(ctx context.Context, cancelFunc context.CancelFunc) error {
	assetWebsocket, err := by.Websocket.GetAssetWebsocket(asset.Spot)
	if err != nil {
		return fmt.Errorf("%w asset type: %v", err, asset.Spot)
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
	go by.wsSpotReadData(ctx, cancelFunc, assetWebsocket.AuthConn, assetWebsocket)
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

// GenerateSpotDefaultSubscriptions generates default subscription
func (by *Bybit) GenerateSpotDefaultSubscriptions() ([]stream.ChannelSubscription, error) {
	channels := defaultSpotSubscriptionChannels
	var subscriptions []stream.ChannelSubscription
	pairs, err := by.GetEnabledPairs(asset.Spot)
	if err != nil {
		return nil, err
	}
	pairFormat, err := by.GetPairFormat(asset.Spot, true)
	if err != nil {
		return nil, err
	}
	pairs = pairs.Format(pairFormat)
	for x := range channels {
		for p := range pairs {
			subscriptions = append(subscriptions,
				stream.ChannelSubscription{
					Channel:  channels[x],
					Currency: pairs[p],
					Asset:    asset.Spot,
				})
		}
	}
	return subscriptions, nil
}

// SubscribeSpot sends a websocket message to receive data from the channel
func (by *Bybit) SubscribeSpot(channelsToSubscribe []stream.ChannelSubscription) error {
	assetWebsocket, err := by.Websocket.GetAssetWebsocket(asset.Spot)
	if err != nil {
		return fmt.Errorf("%w asset type: %v", err, asset.Spot)
	}
	var errs error
	for i := range channelsToSubscribe {
		var subReq WsReq
		subReq.Topic = channelsToSubscribe[i].Channel
		subReq.Event = sub

		formattedPair, err := by.FormatExchangeCurrency(channelsToSubscribe[i].Currency, asset.Spot)
		if err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		if channelsToSubscribe[i].Channel == wsKlines {
			subReq.Parameters = WsParams{
				Symbol:    formattedPair.String(),
				IsBinary:  false,
				KlineType: "1m",
			}
		} else {
			subReq.Parameters = WsParams{
				Symbol:   formattedPair.String(),
				IsBinary: false,
			}
		}
		err = assetWebsocket.Conn.SendJSONMessage(subReq)
		if err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		assetWebsocket.AddSuccessfulSubscriptions(channelsToSubscribe[i])
	}
	return errs
}

// UnsubscribeSpot sends a websocket message to stop receiving data from the channel
func (by *Bybit) UnsubscribeSpot(channelsToUnsubscribe []stream.ChannelSubscription) error {
	assetWebsocket, err := by.Websocket.GetAssetWebsocket(asset.Spot)
	if err != nil {
		return fmt.Errorf("%w asset type: %v", err, asset.Spot)
	}
	var errs error
	for i := range channelsToUnsubscribe {
		var unSub WsReq
		unSub.Event = cancel
		unSub.Topic = channelsToUnsubscribe[i].Channel
		formattedPair, err := by.FormatExchangeCurrency(channelsToUnsubscribe[i].Currency, asset.Spot)
		if err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		unSub.Parameters = WsParams{
			Symbol: formattedPair.String(),
		}
		err = assetWebsocket.Conn.SendJSONMessage(unSub)
		if err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		assetWebsocket.RemoveSuccessfulUnsubscriptions(channelsToUnsubscribe[i])
	}
	return errs
}

// wsSpotReadData gets and passes on websocket messages for processing
func (by *Bybit) wsSpotReadData(ctx context.Context, cancelFunc context.CancelFunc, wsConn stream.Connection, assetWebsocket *stream.Websocket) {
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

			err := by.wsSpotHandleData(resp.Raw)
			if err != nil {
				by.Websocket.DataHandler <- err
			}
		}
	}
}

// wsSpotHandleResp handles response messages from ws requests
func (by *Bybit) wsSpotHandleResp(wsResp *WsResp) error {
	switch wsResp.Event {
	case sub:
		if wsResp.Code != 0 {
			log.Errorf(log.ExchangeSys, "%s Asset Type %v Subscription failed: %v", by.Name, asset.Spot, wsResp.Parameters)
		}
	case cancel:
		if wsResp.Code != 0 {
			log.Errorf(log.ExchangeSys, "%s Asset Type %v Unsubscription failed: %v", by.Name, asset.Spot, wsResp.Parameters)
		}
	default:
		log.Errorf(log.ExchangeSys, "%s Asset Type %v Unhandled response", by.Name, asset.Spot)
	}
	return nil
}

func (by *Bybit) wsSpotHandleData(respRaw []byte) error {
	var result interface{}
	err := json.Unmarshal(respRaw, &result)
	if err != nil {
		return err
	}
	switch d := result.(type) {
	case map[string]interface{}:
		if _, ok := d["event"].(string); ok {
			var wsResp WsResp
			err = json.Unmarshal(respRaw, &wsResp)
			if err != nil {
				if by.Verbose {
					log.Warnf(log.ExchangeSys, "%s Asset Type %v Received unhandled message on websocket: %v\n", by.Name, asset.Spot, result)
				}
				return nil
			}
			return by.wsSpotHandleResp(&wsResp)
		}

		if t, ok := d["topic"].(string); ok {
			switch t {
			case wsOrderbook:
				var data WsOrderbook
				err := json.Unmarshal(respRaw, &data)
				if err != nil {
					return err
				}
				p, err := by.extractCurrencyPair(data.OBData.Symbol, asset.Spot)
				if err != nil {
					return err
				}

				err = by.wsUpdateOrderbook(&data.OBData, p, asset.Spot)
				if err != nil {
					return err
				}
				return nil
			case wsTrade:
				if !by.IsSaveTradeDataEnabled() {
					return nil
				}
				var data WsTrade
				err := json.Unmarshal(respRaw, &data)
				if err != nil {
					return err
				}

				p, err := by.extractCurrencyPair(data.Parameters.Symbol, asset.Spot)
				if err != nil {
					return err
				}

				side := order.Sell
				if data.TradeData.Side {
					side = order.Buy
				}

				return trade.AddTradesToBuffer(by.Name, trade.Data{
					Timestamp:    data.TradeData.Time.Time(),
					CurrencyPair: p,
					AssetType:    asset.Spot,
					Exchange:     by.Name,
					Price:        data.TradeData.Price.Float64(),
					Amount:       data.TradeData.Size.Float64(),
					Side:         side,
					TID:          data.TradeData.ID,
				})
			case wsTicker:
				var data WsSpotTicker
				err := json.Unmarshal(respRaw, &data)
				if err != nil {
					return err
				}

				p, err := by.extractCurrencyPair(data.Ticker.Symbol, asset.Spot)
				if err != nil {
					return err
				}

				by.Websocket.DataHandler <- &ticker.Price{
					ExchangeName: by.Name,
					Bid:          data.Ticker.Bid.Float64(),
					Ask:          data.Ticker.Ask.Float64(),
					LastUpdated:  data.Ticker.Time.Time(),
					AssetType:    asset.Spot,
					Pair:         p,
				}
				return nil
			case wsKlines:
				var data KlineStream
				err := json.Unmarshal(respRaw, &data)
				if err != nil {
					return err
				}

				p, err := by.extractCurrencyPair(data.Kline.Symbol, asset.Spot)
				if err != nil {
					return err
				}

				by.Websocket.DataHandler <- stream.KlineData{
					Pair:       p,
					AssetType:  asset.Spot,
					Exchange:   by.Name,
					StartTime:  data.Kline.StartTime.Time(),
					Interval:   data.Parameters.KlineType,
					OpenPrice:  data.Kline.OpenPrice.Float64(),
					ClosePrice: data.Kline.ClosePrice.Float64(),
					HighPrice:  data.Kline.HighPrice.Float64(),
					LowPrice:   data.Kline.LowPrice.Float64(),
					Volume:     data.Kline.Volume.Float64(),
				}
				return nil
			default:
				by.Websocket.DataHandler <- stream.UnhandledMessageWarning{Message: by.Name + stream.UnhandledMessage + string(respRaw)}
			}
		}

		if m, ok := d["auth"].(string); ok {
			log.Infof(log.WebsocketMgr, "%v received auth response: %v", by.Name, m)
			assetWebsocket, err := by.Websocket.GetAssetWebsocket(asset.Spot)
			if err != nil {
				return fmt.Errorf("%w asset type: %v", err, asset.Spot)
			}
			if m == "fail" {
				log.Errorf(log.ExchangeSys, "%s Asset Type %v Authentication failed with message: %v - disabling authenticated endpoint", by.Name, asset.Spot, m)
				assetWebsocket.SetCanUseAuthenticatedEndpoints(false)
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
			// Sending the authentication message automatically subscribes you to all 4 private topics.
			if by.Verbose {
				log.Debugf(log.ExchangeSys, "%s Asset Type %v Authentication succeeded", by.Name, asset.Spot)
				return nil
			}
			return nil
		}

		if m, ok := d["pong"]; ok {
			log.Infof(log.WebsocketMgr, "%v received pong: %v", by.Name, m)
			return nil
		}
	case []interface{}:
		for i := range d {
			obj, ok := d[i].(map[string]interface{})
			if !ok {
				return common.GetTypeAssertError("map[string]interface{}", d[i])
			}
			e, ok := obj["e"].(string)
			if !ok {
				return common.GetTypeAssertError("string", obj["e"])
			}

			switch e {
			case wsAccountInfo:
				var data []wsAccount
				err := json.Unmarshal(respRaw, &data)
				if err != nil {
					return fmt.Errorf("%v - Could not convert to outboundAccountInfo structure %w",
						by.Name,
						err)
				}
				by.Websocket.DataHandler <- data
				return nil
			case wsOrderExecution:
				var data []wsOrderUpdate
				err := json.Unmarshal(respRaw, &data)
				if err != nil {
					return fmt.Errorf("%v - Could not convert to executionReport structure %w",
						by.Name,
						err)
				}

				for j := range data {
					oType, err := order.StringToOrderType(data[j].OrderType)
					if err != nil {
						by.Websocket.DataHandler <- order.ClassificationError{
							Exchange: by.Name,
							OrderID:  data[j].OrderID,
							Err:      err,
						}
					}
					var oSide order.Side
					oSide, err = order.StringToOrderSide(data[j].Side)
					if err != nil {
						by.Websocket.DataHandler <- order.ClassificationError{
							Exchange: by.Name,
							OrderID:  data[j].OrderID,
							Err:      err,
						}
					}
					var oStatus order.Status
					oStatus, err = stringToOrderStatus(data[j].OrderStatus)
					if err != nil {
						by.Websocket.DataHandler <- order.ClassificationError{
							Exchange: by.Name,
							OrderID:  data[j].OrderID,
							Err:      err,
						}
					}

					p, err := by.extractCurrencyPair(data[j].Symbol, asset.Spot)
					if err != nil {
						return err
					}

					by.Websocket.DataHandler <- order.Detail{
						Price:           data[j].Price.Float64(),
						Amount:          data[j].Quantity.Float64(),
						ExecutedAmount:  data[j].CumulativeFilledQuantity.Float64(),
						RemainingAmount: data[j].Quantity.Float64() - data[j].CumulativeFilledQuantity.Float64(),
						Exchange:        by.Name,
						OrderID:         data[j].OrderID,
						Type:            oType,
						Side:            oSide,
						Status:          oStatus,
						AssetType:       asset.Spot,
						Date:            data[j].OrderCreationTime.Time(),
						Pair:            p,
						ClientOrderID:   data[j].ClientOrderID,
						Trades: []order.TradeHistory{
							{
								Price:     data[j].Price.Float64(),
								Amount:    data[j].Quantity.Float64(),
								Exchange:  by.Name,
								Timestamp: data[j].OrderCreationTime.Time(),
							},
						},
					}
				}
				return nil
			case wsTicketInfo:
				var data []wsOrderFilled
				err := json.Unmarshal(respRaw, &data)
				if err != nil {
					return fmt.Errorf("%v - Could not convert to ticketInfo structure %w",
						by.Name,
						err)
				}

				for j := range data {
					var oSide order.Side
					oSide, err = order.StringToOrderSide(data[j].Side)
					if err != nil {
						by.Websocket.DataHandler <- order.ClassificationError{
							Exchange: by.Name,
							OrderID:  data[j].OrderID,
							Err:      err,
						}
					}

					p, err := by.extractCurrencyPair(data[j].Symbol, asset.Spot)
					if err != nil {
						return err
					}

					by.Websocket.DataHandler <- &order.Detail{
						Exchange:  by.Name,
						OrderID:   data[j].OrderID,
						Side:      oSide,
						AssetType: asset.Spot,
						Pair:      p,
						Price:     data[j].Price.Float64(),
						Amount:    data[j].Quantity.Float64(),
						Date:      data[j].Timestamp.Time(),
						Trades: []order.TradeHistory{
							{
								Price:     data[j].Price.Float64(),
								Amount:    data[j].Quantity.Float64(),
								Exchange:  by.Name,
								Timestamp: data[j].Timestamp.Time(),
								TID:       data[j].TradeID,
								IsMaker:   data[j].IsMaker,
							},
						},
					}
				}
				return nil
			}
		}
	}

	return fmt.Errorf("unhandled stream data %s", string(respRaw))
}

func (by *Bybit) wsUpdateOrderbook(update *WsOrderbookData, p currency.Pair, assetType asset.Item) error {
	if update == nil || (len(update.Asks) == 0 && len(update.Bids) == 0) {
		return errors.New("no orderbook data")
	}
	asks := make([]orderbook.Item, len(update.Asks))
	for i := range update.Asks {
		target, err := strconv.ParseFloat(update.Asks[i][0], 64)
		if err != nil {
			return err
		}
		amount, err := strconv.ParseFloat(update.Asks[i][1], 64)
		if err != nil {
			return err
		}
		asks[i] = orderbook.Item{Price: target, Amount: amount}
	}
	bids := make([]orderbook.Item, len(update.Bids))
	for i := range update.Bids {
		target, err := strconv.ParseFloat(update.Bids[i][0], 64)
		if err != nil {
			return err
		}
		amount, err := strconv.ParseFloat(update.Bids[i][1], 64)
		if err != nil {
			return err
		}
		bids[i] = orderbook.Item{Price: target, Amount: amount}
	}
	return by.Websocket.Orderbook.LoadSnapshot(&orderbook.Base{
		Bids:            bids,
		Asks:            asks,
		Pair:            p,
		LastUpdated:     update.Time.Time(),
		Asset:           assetType,
		Exchange:        by.Name,
		VerifyOrderbook: by.CanVerifyOrderbook,
	})
}

func stringToOrderStatus(status string) (order.Status, error) {
	switch status {
	case "NEW":
		return order.New, nil
	case "CANCELED":
		return order.Cancelled, nil
	case "REJECTED":
		return order.Rejected, nil
	case "TRADE":
		return order.PartiallyFilled, nil
	case "EXPIRED":
		return order.Expired, nil
	default:
		return order.UnknownStatus, errors.New(status + " not recognised as order status")
	}
}
