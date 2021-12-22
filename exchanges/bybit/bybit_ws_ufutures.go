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

// TODO
// do we need multiple connection? by.Websocket.Conn

const (
	wsCoinMarginedPath = "realtime"
	coinSubscribe      = "subscribe"
	coinUnsubscribe    = "unsubscribe"
	dot                = "."

	wsCoinOrder25           = "orderBookL2_25"
	wsCoinOrder200          = "orderBook_200"
	wsCoinTrade             = "trade"
	wsCoinInsurance         = "insurance"
	wsCoinInstrument        = "instrument_info"
	wsCoinMarket            = "klineV2"
	wsCoinLiquidation       = "liquidation"
	wsOrderbookSnapshot     = "snapshot"
	wsOrderbookDelta        = "delta"
	wsOrderbookActionDelete = "delete"
	wsOrderbookActionUpdate = "update"
	wsOrderbookActionInsert = "insert"
	wsKlineV2               = "klineV2"

	wsCoinPosition = "position"
)

var pingRequest = WsCoinReq{Topic: stream.Ping}

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
		var sub WsCoinReq
		sub.Topic = coinSubscribe

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
		var unSub WsCoinReq
		unSub.Topic = coinUnsubscribe

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
			case wsCoinOrder25, wsCoinOrder200:
				switch wsType {
				case wsOrderbookSnapshot:
					var orderbooks WsCoinOrderbook
					err := json.Unmarshal(respRaw, &orderbooks)
					if err != nil {
						return err
					}

					p, err := currency.NewPairFromString(orderbooks.OBData[0].Symbol)
					if err != nil {
						return err
					}

					a, err := by.GetPairAssetType(p)
					if err != nil {
						return err
					}
					err = by.processOrderbook(orderbooks.OBData,
						orderbooks.Type,
						p,
						a)
					if err != nil {
						return err
					}

				case wsOrderbookDelta:
					var orderbooks WsCoinDeltaOrderbook
					err := json.Unmarshal(respRaw, &orderbooks)
					if err != nil {
						return err
					}

					if len(orderbooks.OBData.Delete) > 0 {
						p, err := currency.NewPairFromString(orderbooks.OBData.Delete[0].Symbol)
						if err != nil {
							return err
						}

						a, err := by.GetPairAssetType(p)
						if err != nil {
							return err
						}
						err = by.processOrderbook(orderbooks.OBData.Delete,
							wsOrderbookActionDelete,
							p,
							a)
						if err != nil {
							return err
						}
					}

					if len(orderbooks.OBData.Update) > 0 {
						p, err := currency.NewPairFromString(orderbooks.OBData.Update[0].Symbol)
						if err != nil {
							return err
						}

						a, err := by.GetPairAssetType(p)
						if err != nil {
							return err
						}
						err = by.processOrderbook(orderbooks.OBData.Update,
							wsOrderbookActionUpdate,
							p,
							a)
						if err != nil {
							return err
						}
					}

					if len(orderbooks.OBData.Insert) > 0 {
						p, err := currency.NewPairFromString(orderbooks.OBData.Insert[0].Symbol)
						if err != nil {
							return err
						}

						a, err := by.GetPairAssetType(p)
						if err != nil {
							return err
						}
						err = by.processOrderbook(orderbooks.OBData.Insert,
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
				var data WsCoinTrade
				err := json.Unmarshal(respRaw, &data)
				if err != nil {
					return err
				}
				var trades []trade.Data
				for i := range data.TradeData {
					p, err := currency.NewPairFromString(data.TradeData[0].Symbol)
					if err != nil {
						return err
					}

					var a asset.Item
					a, err = by.GetPairAssetType(p)
					if err != nil {
						return err
					}

					var oSide order.Side
					oSide, err = order.StringToOrderSide(data.TradeData[i].Side)
					if err != nil {
						by.Websocket.DataHandler <- order.ClassificationError{
							Exchange: by.Name,
							Err:      err,
						}
					}

					trades = append(trades, trade.Data{
						TID:          data.TradeData[i].ID,
						Exchange:     by.Name,
						CurrencyPair: p,
						AssetType:    a,
						Side:         oSide,
						Price:        data.TradeData[i].Price,
						Amount:       float64(data.TradeData[i].Size),
						Timestamp:    data.TradeData[i].Time,
					})
				}
				return by.AddTradesToBuffer(trades...)

			case wsKlineV2:
				var candleData WsCoinKline
				err = json.Unmarshal(respRaw, &candleData)
				if err != nil {
					return err
				}
				newPair, err := by.getCurrencyFromWsTopic(asset.CoinMarginedFutures, candleData.Topic)
				if err != nil {
					return err
				}

				for i := range candleData.KlineData {
					by.Websocket.DataHandler <- stream.KlineData{
						Pair:       newPair,
						AssetType:  asset.CoinMarginedFutures,
						Exchange:   by.Name,
						OpenPrice:  candleData.KlineData[i].Open,
						HighPrice:  candleData.KlineData[i].High,
						LowPrice:   candleData.KlineData[i].Low,
						ClosePrice: candleData.KlineData[i].Close,
						Volume:     candleData.KlineData[i].Volume,
						Timestamp:  time.Unix(candleData.KlineData[i].Timestamp, 0),
					}
				}

			case wsCoinInsurance:
				var data WsCoinInsurance
				err = json.Unmarshal(respRaw, &data)
				if err != nil {
					return err
				}
				by.Websocket.DataHandler <- data

			case wsCoinInstrument:
				var data WsCoinTicker
				err := json.Unmarshal(respRaw, &data)
				if err != nil {
					return err
				}

				p, err := currency.NewPairFromString(data.Ticker.Symbol)
				if err != nil {
					return err
				}

				by.Websocket.DataHandler <- &ticker.Price{
					ExchangeName: by.Name,
					Last:         data.Ticker.LastPrice,
					High:         data.Ticker.HighPrice24h,
					Low:          data.Ticker.LowPrice24h,
					Bid:          data.Ticker.BidPrice,
					Ask:          data.Ticker.AskPrice,
					Volume:       float64(data.Ticker.Volume24h),
					Close:        data.Ticker.PrevPrice24h,
					LastUpdated:  data.Ticker.UpdateAt,
					AssetType:    asset.Spot,
					Pair:         p,
				}

			case wsCoinLiquidation:
				var data WsCoinLiquidation
				err = json.Unmarshal(respRaw, &data)
				if err != nil {
					return err
				}
				by.Websocket.DataHandler <- data

			case wsCoinPosition:
				var data WsCoinPosition
				err = json.Unmarshal(respRaw, &data)
				if err != nil {
					return err
				}
				by.Websocket.DataHandler <- data

			default:
				by.Websocket.DataHandler <- stream.UnhandledMessageWarning{Message: by.Name + stream.UnhandledMessage + string(respRaw)}
			}
		}
	}

	return nil
}

// processOrderbook processes orderbook updates
func (by *Bybit) processOrderbook(data []WsCoinOrderbookData, action string, p currency.Pair, a asset.Item) error {
	if len(data) < 1 {
		return errors.New("no orderbook data")
	}

	switch action {
	case wsOrderbookSnapshot:
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

func (by *Bybit) getCurrencyFromWsTopic(assetType asset.Item, channelTopic string) (cp currency.Pair, err error) {
	var format currency.PairFormat
	format, err = by.GetPairFormat(assetType, true)
	if err != nil {
		return cp, err
	}

	var pairs currency.Pairs
	pairs, err = by.GetEnabledPairs(assetType)
	if err != nil {
		return cp, err
	}
	// channel topics are formatted as "spot/orderbook.BTCUSDT"
	channelSplit := strings.Split(channelTopic, ".")
	if len(channelSplit) == 1 {
		return currency.Pair{}, errors.New("no currency found in topic " + channelTopic)
	}
	cp, err = currency.MatchPairsWithNoDelimiter(channelSplit[len(channelSplit)-1], pairs, format)
	if err != nil {
		return cp, err
	}
	if !pairs.Contains(cp, true) {
		return cp, fmt.Errorf("currency %s not found in enabled pairs", cp.String())
	}
	return cp, nil
}
