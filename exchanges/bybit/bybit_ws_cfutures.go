package bybit

import (
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
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream/buffer"
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

	go by.wsCoinReadData()
	if by.GetAuthenticatedAPISupport(exchange.WebsocketAuthentication) {
		err = by.WsCoinAuth()
		if err != nil {
			by.Websocket.DataHandler <- err
			by.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
	}

	return nil
}

// WsCoinAuth sends an authentication message to receive auth data
func (by *Bybit) WsCoinAuth() error {
	intNonce := (time.Now().Unix() + 1) * 1000
	strNonce := strconv.FormatInt(intNonce, 10)
	hmac := crypto.GetHMAC(
		crypto.HashSHA256,
		[]byte("GET/realtime"+strNonce),
		[]byte(by.API.Credentials.Secret),
	)
	sign := crypto.HexEncodeToString(hmac)
	req := Authenticate{
		Operation: "auth",
		Args:      []string{by.API.Credentials.Key, strNonce, sign},
	}
	return by.Websocket.Conn.SendJSONMessage(req)
}

// SubscribeCoin sends a websocket message to receive data from the channel
func (by *Bybit) SubscribeCoin(channelsToSubscribe []stream.ChannelSubscription) error {
	var errs common.Errors
	for i := range channelsToSubscribe {
		var sub WsFuturesReq
		sub.Topic = subscribe

		a, err := by.GetPairAssetType(channelsToSubscribe[i].Currency)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		formattedPair, err := by.FormatExchangeCurrency(channelsToSubscribe[i].Currency, a)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		sub.Args = append(sub.Args, formatArgs(channelsToSubscribe[i].Channel, formattedPair.String(), channelsToSubscribe[i].Params))
		err = by.Websocket.Conn.SendJSONMessage(sub)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		by.Websocket.AddSuccessfulSubscriptions(channelsToSubscribe[i])
	}
	if errs != nil {
		return errs
	}
	return nil
}

func formatArgs(channel, pair string, params map[string]interface{}) string {
	argStr := channel
	for _, param := range params {
		argStr += dot + fmt.Sprintf("%v", param)
	}
	return argStr
}

// UnsubscribeCoin sends a websocket message to stop receiving data from the channel
func (by *Bybit) UnsubscribeCoin(channelsToUnsubscribe []stream.ChannelSubscription) error {
	var errs common.Errors

	for i := range channelsToUnsubscribe {
		var unSub WsFuturesReq
		unSub.Topic = unsubscribe

		a, err := by.GetPairAssetType(channelsToUnsubscribe[i].Currency)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		formattedPair, err := by.FormatExchangeCurrency(channelsToUnsubscribe[i].Currency, a)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		unSub.Args = append(unSub.Args, channelsToUnsubscribe[i].Channel+dot+formattedPair.String())
		err = by.Websocket.Conn.SendJSONMessage(unSub)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		by.Websocket.RemoveSuccessfulUnsubscriptions(channelsToUnsubscribe[i])
	}
	if errs != nil {
		return errs
	}
	return nil
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

	if t, ok := multiStreamData["topic"].(string); ok {
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
					err := json.Unmarshal(respRaw, &response)
					if err != nil {
						return err
					}

					p, err := currency.NewPairFromString(response.OBData[0].Symbol)
					if err != nil {
						return err
					}

					a, err := by.GetPairAssetType(p)
					if err != nil {
						return err
					}
					err = by.processOrderbook(response.OBData,
						response.Type,
						p,
						a)
					if err != nil {
						return err
					}

				case wsOperationDelta:
					var response WsCoinDeltaOrderbook
					err := json.Unmarshal(respRaw, &response)
					if err != nil {
						return err
					}

					if len(response.OBData.Delete) > 0 {
						p, err := currency.NewPairFromString(response.OBData.Delete[0].Symbol)
						if err != nil {
							return err
						}

						a, err := by.GetPairAssetType(p)
						if err != nil {
							return err
						}
						err = by.processOrderbook(response.OBData.Delete,
							wsOrderbookActionDelete,
							p,
							a)
						if err != nil {
							return err
						}
					}

					if len(response.OBData.Update) > 0 {
						p, err := currency.NewPairFromString(response.OBData.Update[0].Symbol)
						if err != nil {
							return err
						}

						a, err := by.GetPairAssetType(p)
						if err != nil {
							return err
						}
						err = by.processOrderbook(response.OBData.Update,
							wsOrderbookActionUpdate,
							p,
							a)
						if err != nil {
							return err
						}
					}

					if len(response.OBData.Insert) > 0 {
						p, err := currency.NewPairFromString(response.OBData.Insert[0].Symbol)
						if err != nil {
							return err
						}

						a, err := by.GetPairAssetType(p)
						if err != nil {
							return err
						}
						err = by.processOrderbook(response.OBData.Insert,
							wsOrderbookActionInsert,
							p,
							a)
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
				err := json.Unmarshal(respRaw, &response)
				if err != nil {
					return err
				}
				var trades []trade.Data
				for i := range response.TradeData {
					p, err := currency.NewPairFromString(response.TradeData[0].Symbol)
					if err != nil {
						return err
					}

					var a asset.Item
					a, err = by.GetPairAssetType(p)
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

					trades = append(trades, trade.Data{
						TID:          response.TradeData[i].ID,
						Exchange:     by.Name,
						CurrencyPair: p,
						AssetType:    a,
						Side:         oSide,
						Price:        response.TradeData[i].Price,
						Amount:       float64(response.TradeData[i].Size),
						Timestamp:    response.TradeData[i].Time,
					})
				}
				return by.AddTradesToBuffer(trades...)

			case wsKlineV2:
				var response WsFuturesKline
				err = json.Unmarshal(respRaw, &response)
				if err != nil {
					return err
				}

				p, err := currency.NewPairFromString(topics[len(topics)-1])
				if err != nil {
					return err
				}

				var a asset.Item
				a, err = by.GetPairAssetType(p)
				if err != nil {
					return err
				}
				for i := range response.KlineData {
					by.Websocket.DataHandler <- stream.KlineData{
						Pair:       p,
						AssetType:  a,
						Exchange:   by.Name,
						OpenPrice:  response.KlineData[i].Open,
						HighPrice:  response.KlineData[i].High,
						LowPrice:   response.KlineData[i].Low,
						ClosePrice: response.KlineData[i].Close,
						Volume:     response.KlineData[i].Volume,
						Timestamp:  time.Unix(response.KlineData[i].Timestamp, 0),
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
						err := json.Unmarshal(respRaw, &response)
						if err != nil {
							return err
						}

						p, err := currency.NewPairFromString(response.Ticker.Symbol)
						if err != nil {
							return err
						}

						var a asset.Item
						a, err = by.GetPairAssetType(p)
						if err != nil {
							return err
						}
						by.Websocket.DataHandler <- &ticker.Price{
							ExchangeName: by.Name,
							Last:         response.Ticker.LastPrice,
							High:         response.Ticker.HighPrice24h,
							Low:          response.Ticker.LowPrice24h,
							Bid:          response.Ticker.BidPrice,
							Ask:          response.Ticker.AskPrice,
							Volume:       float64(response.Ticker.Volume24h),
							Close:        response.Ticker.PrevPrice24h,
							LastUpdated:  response.Ticker.UpdateAt,
							AssetType:    a,
							Pair:         p,
						}

					case wsOperationDelta:
						var response WsDeltaTicker
						err := json.Unmarshal(respRaw, &response)
						if err != nil {
							return err
						}

						if len(response.Data.Delete) > 0 {
							for _, t := range response.Data.Delete {
								p, err := currency.NewPairFromString(t.Symbol)
								if err != nil {
									return err
								}

								var a asset.Item
								a, err = by.GetPairAssetType(p)
								if err != nil {
									return err
								}

								by.Websocket.DataHandler <- &ticker.Price{
									ExchangeName: by.Name,
									Last:         t.LastPrice,
									High:         t.HighPrice24h,
									Low:          t.LowPrice24h,
									Bid:          t.BidPrice,
									Ask:          t.AskPrice,
									Volume:       float64(t.Volume24h),
									Close:        t.PrevPrice24h,
									LastUpdated:  t.UpdateAt,
									AssetType:    a,
									Pair:         p,
								}
							}
						}

						if len(response.Data.Update) > 0 {
							for _, t := range response.Data.Update {
								p, err := currency.NewPairFromString(t.Symbol)
								if err != nil {
									return err
								}

								var a asset.Item
								a, err = by.GetPairAssetType(p)
								if err != nil {
									return err
								}

								by.Websocket.DataHandler <- &ticker.Price{
									ExchangeName: by.Name,
									Last:         t.LastPrice,
									High:         t.HighPrice24h,
									Low:          t.LowPrice24h,
									Bid:          t.BidPrice,
									Ask:          t.AskPrice,
									Volume:       float64(t.Volume24h),
									Close:        t.PrevPrice24h,
									LastUpdated:  t.UpdateAt,
									AssetType:    a,
									Pair:         p,
								}
							}
						}

						if len(response.Data.Insert) > 0 {
							for _, t := range response.Data.Insert {
								p, err := currency.NewPairFromString(t.Symbol)
								if err != nil {
									return err
								}

								var a asset.Item
								a, err = by.GetPairAssetType(p)
								if err != nil {
									return err
								}

								by.Websocket.DataHandler <- &ticker.Price{
									ExchangeName: by.Name,
									Last:         t.LastPrice,
									High:         t.HighPrice24h,
									Low:          t.LowPrice24h,
									Bid:          t.BidPrice,
									Ask:          t.AskPrice,
									Volume:       float64(t.Volume24h),
									Close:        t.PrevPrice24h,
									LastUpdated:  t.UpdateAt,
									AssetType:    a,
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
					p, err = currency.NewPairFromString(response.Data[i].Symbol)
					if err != nil {
						return err
					}

					var a asset.Item
					a, err = by.GetPairAssetType(p)
					if err != nil {
						return err
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
					var oSide order.Side
					oSide, err = order.StringToOrderSide(response.Data[i].Side)
					if err != nil {
						by.Websocket.DataHandler <- order.ClassificationError{
							Exchange: by.Name,
							OrderID:  response.Data[i].OrderID,
							Err:      err,
						}
					}
					by.Websocket.DataHandler <- &order.Modify{
						Exchange:  by.Name,
						ID:        response.Data[i].OrderID,
						AssetType: a,
						Pair:      p,
						Status:    oStatus,
						Trades: []order.TradeHistory{
							{
								Price:     response.Data[i].Price,
								Amount:    float64(response.Data[i].OrderQty),
								Exchange:  by.Name,
								TID:       response.Data[i].ExecutionID,
								Side:      oSide,
								Timestamp: response.Data[i].Time,
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
					var a asset.Item
					p, a, err = by.GetRequestFormattedPairAndAssetType(response.Data[x].Symbol)
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
						Price:     response.Data[x].Price,
						Amount:    float64(response.Data[x].OrderQty),
						Exchange:  by.Name,
						ID:        response.Data[x].OrderID,
						Type:      oType,
						Side:      oSide,
						Status:    oStatus,
						AssetType: a,
						Date:      response.Data[x].Time,
						Pair:      p,
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
					var a asset.Item
					p, a, err = by.GetRequestFormattedPairAndAssetType(response.Data[x].Symbol)
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
						Price:     response.Data[x].Price,
						Amount:    float64(response.Data[x].OrderQty),
						Exchange:  by.Name,
						ID:        response.Data[x].OrderID,
						AccountID: strconv.FormatInt(response.Data[x].UserID, 10),
						Type:      oType,
						Side:      oSide,
						Status:    oStatus,
						AssetType: a,
						Date:      response.Data[x].Time,
						Pair:      p,
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
			target, err := strconv.ParseFloat(data[i].Price, 64)
			if err != nil {
				by.Websocket.DataHandler <- err
				continue
			}

			item := orderbook.Item{
				Price:  target,
				Amount: float64(data[i].Size),
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
		var asks, bids []orderbook.Item
		for i := range data {
			target, err := strconv.ParseFloat(data[i].Price, 64)
			if err != nil {
				by.Websocket.DataHandler <- err
				continue
			}

			item := orderbook.Item{
				Price:  target,
				Amount: float64(data[i].Size),
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

		err := by.Websocket.Orderbook.Update(&buffer.Update{
			Bids:   bids,
			Asks:   asks,
			Pair:   p,
			Asset:  a,
			Action: buffer.Action(action),
		})
		if err != nil {
			return err
		}
	}
	return nil
}
