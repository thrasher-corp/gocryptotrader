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
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	wsCoinMarginedPath = "realtime"
	subscribe          = "subscribe"
	unsubscribe        = "unsubscribe"
	dot                = "."

	// public endpoints
	wsOrder25     = "orderBookL2_25"
	wsOrder200    = "orderBook_200"
	wsTrade       = "trade"
	wsInsurance   = "insurance"
	wsInstrument  = "instrument_info"
	wsCoinMarket  = "klineV2"
	wsLiquidation = "liquidation"

	wsOperationSnapshot     = "snapshot"
	wsOperationDelta        = "delta"
	wsOrderbookActionDelete = "delete"
	wsOrderbookActionUpdate = "update"
	wsOrderbookActionInsert = "insert"
	wsKlineV2               = "klineV2"

	// private endpoints
	wsPosition  = "position"
	wsExecution = "execution"
	wsOrder     = "order"
	wsStopOrder = "stop_order"
	wsWallet    = "wallet"
)

var pingRequest = WsFuturesReq{Topic: stream.Ping}

// WsCoinConnect connects to a CMF websocket feed
func (by *Bybit) WsCoinConnect() error {
	if !by.Websocket.IsEnabled() || !by.IsEnabled() {
		return errors.New(stream.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
	err := by.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}

	pingMsg, err := json.Marshal(pingRequest)
	if err != nil {
		return err
	}
	by.Websocket.Conn.SetupPingHandler(stream.PingHandler{
		Message:     pingMsg,
		MessageType: websocket.PingMessage,
		Delay:       bybitWebsocketTimer,
	})
	if by.Verbose {
		log.Debugf(log.ExchangeSys, "%s Connected to Websocket.\n", by.Name)
	}

	by.Websocket.Wg.Add(1)
	go by.wsCoinReadData()
	if by.IsWebsocketAuthenticationSupported() {
		err = by.WsCoinAuth(context.TODO())
		if err != nil {
			by.Websocket.DataHandler <- err
			by.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
	}

	by.Websocket.Wg.Add(1)
	go by.WsDataHandler()
	return nil
}

// WsCoinAuth sends an authentication message to receive auth data
func (by *Bybit) WsCoinAuth(ctx context.Context) error {
	creds, err := by.GetCredentials(ctx)
	if err != nil {
		return err
	}

	intNonce := (time.Now().Unix() + 1) * 1000
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
	return by.Websocket.Conn.SendJSONMessage(req)
}

// SubscribeCoin sends a websocket message to receive data from the channel
func (by *Bybit) SubscribeCoin(channelsToSubscribe []stream.ChannelSubscription) error {
	var errs error
	for i := range channelsToSubscribe {
		var sub WsFuturesReq
		sub.Topic = subscribe

		sub.Args = append(sub.Args, formatArgs(channelsToSubscribe[i].Channel, channelsToSubscribe[i].Params))
		err := by.Websocket.Conn.SendJSONMessage(sub)
		if err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		by.Websocket.AddSuccessfulSubscriptions(channelsToSubscribe[i])
	}
	if errs != nil {
		return errs
	}
	return nil
}

func formatArgs(channel string, params map[string]interface{}) string {
	argStr := channel
	for x := range params {
		argStr += dot + fmt.Sprintf("%v", params[x])
	}
	return argStr
}

// UnsubscribeCoin sends a websocket message to stop receiving data from the channel
func (by *Bybit) UnsubscribeCoin(channelsToUnsubscribe []stream.ChannelSubscription) error {
	var errs error
	for i := range channelsToUnsubscribe {
		var unSub WsFuturesReq
		unSub.Topic = unsubscribe

		formattedPair, err := by.FormatExchangeCurrency(channelsToUnsubscribe[i].Currency, asset.CoinMarginedFutures)
		if err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		unSub.Args = append(unSub.Args, channelsToUnsubscribe[i].Channel+dot+formattedPair.String())
		err = by.Websocket.Conn.SendJSONMessage(unSub)
		if err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		by.Websocket.RemoveSuccessfulUnsubscriptions(channelsToUnsubscribe[i])
	}
	return errs
}

// wsCoinReadData gets and passes on websocket messages for processing
func (by *Bybit) wsCoinReadData() {
	by.Websocket.Wg.Add(1)
	defer by.Websocket.Wg.Done()

	for {
		select {
		case <-by.Websocket.ShutdownC:
			return
		default:
			resp := by.Websocket.Conn.ReadMessage()
			if resp.Raw == nil {
				return
			}

			err := by.wsCoinHandleData(resp.Raw)
			if err != nil {
				by.Websocket.DataHandler <- err
			}
		}
	}
}

func (by *Bybit) wsCoinHandleData(respRaw []byte) error {
	var multiStreamData map[string]interface{}
	err := json.Unmarshal(respRaw, &multiStreamData)
	if err != nil {
		return err
	}

	t, ok := multiStreamData["topic"].(string)
	if !ok {
		log.Errorf(log.ExchangeSys, "%s Received unhandle message on websocket: %v\n", by.Name, multiStreamData)
		return nil
	}

	topics := strings.Split(t, dot)
	if len(topics) < 1 {
		return errors.New(by.Name + " - topic could not be extracted from response")
	}

	if wsType, ok := multiStreamData["type"].(string); ok {
		switch topics[0] {
		case wsOrder25, wsOrder200:
			switch wsType {
			case wsOperationSnapshot:
				var response WsFuturesOrderbook
				err = json.Unmarshal(respRaw, &response)
				if err != nil {
					return err
				}

				var p currency.Pair
				p, err = by.extractCurrencyPair(response.OBData[0].Symbol, asset.CoinMarginedFutures)
				if err != nil {
					return err
				}

				err = by.processOrderbook(response.OBData,
					response.Type,
					p,
					asset.CoinMarginedFutures)
				if err != nil {
					return err
				}

			case wsOperationDelta:
				var response WsCoinDeltaOrderbook
				err = json.Unmarshal(respRaw, &response)
				if err != nil {
					return err
				}

				if len(response.OBData.Delete) > 0 {
					var p currency.Pair
					p, err = by.extractCurrencyPair(response.OBData.Delete[0].Symbol, asset.CoinMarginedFutures)
					if err != nil {
						return err
					}

					err = by.processOrderbook(response.OBData.Delete,
						wsOrderbookActionDelete,
						p,
						asset.CoinMarginedFutures)
					if err != nil {
						return err
					}
				}

				if len(response.OBData.Update) > 0 {
					var p currency.Pair
					p, err = by.extractCurrencyPair(response.OBData.Update[0].Symbol, asset.CoinMarginedFutures)
					if err != nil {
						return err
					}

					err = by.processOrderbook(response.OBData.Update,
						wsOrderbookActionUpdate,
						p,
						asset.CoinMarginedFutures)
					if err != nil {
						return err
					}
				}

				if len(response.OBData.Insert) > 0 {
					var p currency.Pair
					p, err = by.extractCurrencyPair(response.OBData.Insert[0].Symbol, asset.CoinMarginedFutures)
					if err != nil {
						return err
					}

					err = by.processOrderbook(response.OBData.Insert,
						wsOrderbookActionInsert,
						p,
						asset.CoinMarginedFutures)
					if err != nil {
						return err
					}
				}
			default:
				by.Websocket.DataHandler <- stream.UnhandledMessageWarning{Message: by.Name + stream.UnhandledMessage + "unsupported orderbook operation"}
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
				p, err = by.extractCurrencyPair(response.TradeData[0].Symbol, asset.CoinMarginedFutures)
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
				}

				trades[counter] = trade.Data{
					TID:          response.TradeData[i].ID,
					Exchange:     by.Name,
					CurrencyPair: p,
					AssetType:    asset.CoinMarginedFutures,
					Side:         oSide,
					Price:        response.TradeData[i].Price,
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
			p, err = by.extractCurrencyPair(topics[len(topics)-1], asset.CoinMarginedFutures)
			if err != nil {
				return err
			}

			for i := range response.KlineData {
				by.Websocket.DataHandler <- stream.KlineData{
					Pair:       p,
					AssetType:  asset.CoinMarginedFutures,
					Exchange:   by.Name,
					OpenPrice:  response.KlineData[i].Open,
					HighPrice:  response.KlineData[i].High,
					LowPrice:   response.KlineData[i].Low,
					ClosePrice: response.KlineData[i].Close,
					Volume:     response.KlineData[i].Volume,
					Timestamp:  response.KlineData[i].Timestamp.Time(),
				}
			}

		case wsInsurance:
			var response WsInsurance
			err = json.Unmarshal(respRaw, &response)
			if err != nil {
				return err
			}
			by.Websocket.DataHandler <- response.Data

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
					p, err = by.extractCurrencyPair(response.Ticker.Symbol, asset.CoinMarginedFutures)
					if err != nil {
						return err
					}

					by.Websocket.DataHandler <- &ticker.Price{
						ExchangeName: by.Name,
						Last:         response.Ticker.LastPrice.Float64(),
						High:         response.Ticker.HighPrice24h.Float64(),
						Low:          response.Ticker.LowPrice24h.Float64(),
						Bid:          response.Ticker.BidPrice,
						Ask:          response.Ticker.AskPrice,
						Volume:       response.Ticker.Volume24h,
						Close:        response.Ticker.PrevPrice24h.Float64(),
						LastUpdated:  response.Ticker.UpdateAt,
						AssetType:    asset.CoinMarginedFutures,
						Pair:         p,
					}

				case wsOperationDelta:
					var response WsDeltaTicker
					err = json.Unmarshal(respRaw, &response)
					if err != nil {
						return err
					}

					if len(response.Data.Delete) > 0 {
						for x := range response.Data.Delete {
							var p currency.Pair
							p, err = by.extractCurrencyPair(response.Data.Delete[x].Symbol, asset.CoinMarginedFutures)
							if err != nil {
								return err
							}

							by.Websocket.DataHandler <- &ticker.Price{
								ExchangeName: by.Name,
								Last:         response.Data.Delete[x].LastPrice.Float64(),
								High:         response.Data.Delete[x].HighPrice24h.Float64(),
								Low:          response.Data.Delete[x].LowPrice24h.Float64(),
								Bid:          response.Data.Delete[x].BidPrice,
								Ask:          response.Data.Delete[x].AskPrice,
								Volume:       response.Data.Delete[x].Volume24h,
								Close:        response.Data.Delete[x].PrevPrice24h.Float64(),
								LastUpdated:  response.Data.Delete[x].UpdateAt,
								AssetType:    asset.CoinMarginedFutures,
								Pair:         p,
							}
						}
					}

					if len(response.Data.Update) > 0 {
						for x := range response.Data.Update {
							var p currency.Pair
							p, err = by.extractCurrencyPair(response.Data.Update[x].Symbol, asset.CoinMarginedFutures)
							if err != nil {
								return err
							}

							by.Websocket.DataHandler <- &ticker.Price{
								ExchangeName: by.Name,
								Last:         response.Data.Update[x].LastPrice.Float64(),
								High:         response.Data.Update[x].HighPrice24h.Float64(),
								Low:          response.Data.Update[x].LowPrice24h.Float64(),
								Bid:          response.Data.Update[x].BidPrice,
								Ask:          response.Data.Update[x].AskPrice,
								Volume:       response.Data.Update[x].Volume24h,
								Close:        response.Data.Update[x].PrevPrice24h.Float64(),
								LastUpdated:  response.Data.Update[x].UpdateAt,
								AssetType:    asset.CoinMarginedFutures,
								Pair:         p,
							}
						}
					}

					if len(response.Data.Insert) > 0 {
						for x := range response.Data.Insert {
							var p currency.Pair
							p, err = by.extractCurrencyPair(response.Data.Insert[x].Symbol, asset.CoinMarginedFutures)
							if err != nil {
								return err
							}

							by.Websocket.DataHandler <- &ticker.Price{
								ExchangeName: by.Name,
								Last:         response.Data.Insert[x].LastPrice.Float64(),
								High:         response.Data.Insert[x].HighPrice24h.Float64(),
								Low:          response.Data.Insert[x].LowPrice24h.Float64(),
								Bid:          response.Data.Insert[x].BidPrice,
								Ask:          response.Data.Insert[x].AskPrice,
								Volume:       response.Data.Insert[x].Volume24h,
								Close:        response.Data.Insert[x].PrevPrice24h.Float64(),
								LastUpdated:  response.Data.Insert[x].UpdateAt,
								AssetType:    asset.CoinMarginedFutures,
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
				p, err = by.extractCurrencyPair(response.Data[i].Symbol, asset.CoinMarginedFutures)
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
					AssetType: asset.CoinMarginedFutures,
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
				p, err = by.extractCurrencyPair(response.Data[x].Symbol, asset.CoinMarginedFutures)
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
					AssetType: asset.CoinMarginedFutures,
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
				p, err = by.extractCurrencyPair(response.Data[x].Symbol, asset.CoinMarginedFutures)
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
					AssetType: asset.CoinMarginedFutures,
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
	}
	return nil
}

// processOrderbook processes orderbook updates
func (by *Bybit) processOrderbook(data []WsFuturesOrderbookData, action string, p currency.Pair, a asset.Item) error {
	if len(data) < 1 {
		return errors.New("no orderbook data")
	}

	switch action {
	case wsOperationSnapshot:
		var book orderbook.Base
		for i := range data {
			item := orderbook.Item{
				Price:  data[i].Price.Float64(),
				Amount: data[i].Size,
				ID:     data[i].ID,
			}
			switch {
			case strings.EqualFold(data[i].Side, sideSell):
				book.Asks = append(book.Asks, item)
			case strings.EqualFold(data[i].Side, sideBuy):
				book.Bids = append(book.Bids, item)
			default:
				return fmt.Errorf("could not process websocket orderbook update, order side could not be matched for %s",
					data[i].Side)
			}
		}
		book.Asset = a
		book.Pair = p
		book.Exchange = by.Name
		book.VerifyOrderbook = by.CanVerifyOrderbook

		err := by.Websocket.Orderbook.LoadSnapshot(&book)
		if err != nil {
			return fmt.Errorf("process orderbook error -  %s", err)
		}
	default:
		updateAction, err := by.GetActionFromString(action)
		if err != nil {
			return err
		}

		var asks, bids []orderbook.Item
		for i := range data {
			item := orderbook.Item{
				Price:  data[i].Price.Float64(),
				Amount: data[i].Size,
				ID:     data[i].ID,
			}

			switch {
			case strings.EqualFold(data[i].Side, sideSell):
				asks = append(asks, item)
			case strings.EqualFold(data[i].Side, sideBuy):
				bids = append(bids, item)
			default:
				return fmt.Errorf("could not process websocket orderbook update, order side could not be matched for %s",
					data[i].Side)
			}
		}

		err = by.Websocket.Orderbook.Update(&orderbook.Update{
			Bids:   bids,
			Asks:   asks,
			Pair:   p,
			Asset:  a,
			Action: updateAction,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// GetActionFromString matches a string action to an internal action.
func (by *Bybit) GetActionFromString(s string) (orderbook.Action, error) {
	switch s {
	case wsOrderbookActionUpdate:
		return orderbook.Amend, nil
	case wsOrderbookActionDelete:
		return orderbook.Delete, nil
	case wsOrderbookActionInsert:
		return orderbook.Insert, nil
	}
	return 0, fmt.Errorf("%s %w", s, orderbook.ErrInvalidAction)
}
