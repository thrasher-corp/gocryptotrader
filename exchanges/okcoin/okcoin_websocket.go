package okcoin

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
)

const (
	okcoinWebsocketUSDRealTrades      = "ok_usd_realtrades"
	okcoinWebsocketCNYRealTrades      = "ok_cny_realtrades"
	okcoinWebsocketSpotUSDTrade       = "ok_spotusd_trade"
	okcoinWebsocketSpotCNYTrade       = "ok_spotcny_trade"
	okcoinWebsocketSpotUSDCancelOrder = "ok_spotusd_cancel_order"
	okcoinWebsocketSpotCNYCancelOrder = "ok_spotcny_cancel_order"
	okcoinWebsocketSpotUSDUserInfo    = "ok_spotusd_userinfo"
	okcoinWebsocketSpotCNYUserInfo    = "ok_spotcny_userinfo"
	okcoinWebsocketSpotUSDOrderInfo   = "ok_spotusd_order_info"
	okcoinWebsocketSpotCNYOrderInfo   = "ok_spotcny_order_info"
	okcoinWebsocketFuturesTrade       = "ok_futuresusd_trade"
	okcoinWebsocketFuturesCancelOrder = "ok_futuresusd_cancel_order"
	okcoinWebsocketFuturesRealTrades  = "ok_usd_future_realtrades"
	okcoinWebsocketFuturesUserInfo    = "ok_futureusd_userinfo"
	okcoinWebsocketFuturesOrderInfo   = "ok_futureusd_order_info"
)

// PingHandler handles the keep alive
func (o *OKCoin) PingHandler(message string) error {
	err := o.WebsocketConn.WriteControl(websocket.PingMessage, []byte("{'event':'ping'}"), time.Now().Add(time.Second))

	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

// AddChannel adds a new channel on the websocket client
func (o *OKCoin) AddChannel(channel string) {
	event := WebsocketEvent{"addChannel", channel}
	json, err := common.JSONEncode(event)
	if err != nil {
		log.Println(err)
		return
	}
	err = o.WebsocketConn.WriteMessage(websocket.TextMessage, json)

	if err != nil {
		log.Println(err)
		return
	}

	if o.Verbose {
		log.Printf("%s Adding channel: %s\n", o.GetName(), channel)
	}
}

// RemoveChannel removes a channel on the websocket client
func (o *OKCoin) RemoveChannel(channel string) {
	event := WebsocketEvent{"removeChannel", channel}
	json, err := common.JSONEncode(event)
	if err != nil {
		log.Println(err)
		return
	}
	err = o.WebsocketConn.WriteMessage(websocket.TextMessage, json)

	if err != nil {
		log.Println(err)
		return
	}

	if o.Verbose {
		log.Printf("%s Removing channel: %s\n", o.GetName(), channel)
	}
}

// WebsocketSpotTrade handles spot trade request on the websocket client
func (o *OKCoin) WebsocketSpotTrade(symbol, orderType string, price, amount float64) {
	values := make(map[string]string)
	values["symbol"] = symbol
	values["type"] = orderType
	values["price"] = strconv.FormatFloat(price, 'f', -1, 64)
	values["amount"] = strconv.FormatFloat(amount, 'f', -1, 64)

	channel := okcoinWebsocketSpotUSDTrade
	if o.WebsocketURL == okcoinWebsocketURLChina {
		channel = okcoinWebsocketSpotCNYTrade
	}

	o.AddChannelAuthenticated(channel, values)
}

// WebsocketFuturesTrade handles a futures trade on the websocket client
func (o *OKCoin) WebsocketFuturesTrade(symbol, contractType string, price, amount float64, orderType, matchPrice, leverage int) {
	values := make(map[string]string)
	values["symbol"] = symbol
	values["contract_type"] = contractType
	values["price"] = strconv.FormatFloat(price, 'f', -1, 64)
	values["amount"] = strconv.FormatFloat(amount, 'f', -1, 64)
	values["type"] = strconv.Itoa(orderType)
	values["match_price"] = strconv.Itoa(matchPrice)
	values["lever_rate"] = strconv.Itoa(orderType)
	o.AddChannelAuthenticated(okcoinWebsocketFuturesTrade, values)
}

// WebsocketSpotCancel cancels a spot trade on the websocket client
func (o *OKCoin) WebsocketSpotCancel(symbol string, orderID int64) {
	values := make(map[string]string)
	values["symbol"] = symbol
	values["order_id"] = strconv.FormatInt(orderID, 10)

	channel := okcoinWebsocketSpotUSDCancelOrder
	if o.WebsocketURL == okcoinWebsocketURLChina {
		channel = okcoinWebsocketSpotCNYCancelOrder
	}

	o.AddChannelAuthenticated(channel, values)
}

// WebsocketFuturesCancel cancels a futures contract on the websocket client
func (o *OKCoin) WebsocketFuturesCancel(symbol, contractType string, orderID int64) {
	values := make(map[string]string)
	values["symbol"] = symbol
	values["order_id"] = strconv.FormatInt(orderID, 10)
	values["contract_type"] = contractType
	o.AddChannelAuthenticated(okcoinWebsocketFuturesCancelOrder, values)
}

// WebsocketSpotOrderInfo request information on an order on the websocket
// client
func (o *OKCoin) WebsocketSpotOrderInfo(symbol string, orderID int64) {
	values := make(map[string]string)
	values["symbol"] = symbol
	values["order_id"] = strconv.FormatInt(orderID, 10)

	channel := okcoinWebsocketSpotUSDOrderInfo
	if o.WebsocketURL == okcoinWebsocketURLChina {
		channel = okcoinWebsocketSpotCNYOrderInfo
	}

	o.AddChannelAuthenticated(channel, values)
}

// WebsocketFuturesOrderInfo requests futures order info on the websocket client
func (o *OKCoin) WebsocketFuturesOrderInfo(symbol, contractType string, orderID int64, orderStatus, currentPage, pageLength int) {
	values := make(map[string]string)
	values["symbol"] = symbol
	values["order_id"] = strconv.FormatInt(orderID, 10)
	values["contract_type"] = contractType
	values["status"] = strconv.Itoa(orderStatus)
	values["current_page"] = strconv.Itoa(currentPage)
	values["page_length"] = strconv.Itoa(pageLength)
	o.AddChannelAuthenticated(okcoinWebsocketFuturesOrderInfo, values)
}

// ConvertToURLValues converts values to url.Values
func (o *OKCoin) ConvertToURLValues(values map[string]string) url.Values {
	urlVals := url.Values{}
	for i, x := range values {
		urlVals.Set(i, x)
	}
	return urlVals
}

// WebsocketSign signs values on the webcoket client
func (o *OKCoin) WebsocketSign(values map[string]string) string {
	values["api_key"] = o.APIKey
	urlVals := o.ConvertToURLValues(values)
	return strings.ToUpper(common.HexEncodeToString(common.GetMD5([]byte(urlVals.Encode() + "&secret_key=" + o.APISecret))))
}

// AddChannelAuthenticated adds an authenticated channel on the websocket client
func (o *OKCoin) AddChannelAuthenticated(channel string, values map[string]string) {
	values["sign"] = o.WebsocketSign(values)
	event := WebsocketEventAuth{"addChannel", channel, values}
	json, err := common.JSONEncode(event)
	if err != nil {
		log.Println(err)
		return
	}
	err = o.WebsocketConn.WriteMessage(websocket.TextMessage, json)

	if err != nil {
		log.Println(err)
		return
	}

	if o.Verbose {
		log.Printf("%s Adding authenticated channel: %s\n", o.GetName(), channel)
	}
}

// RemoveChannelAuthenticated removes the added authenticated channel on the
// websocket client
func (o *OKCoin) RemoveChannelAuthenticated(conn *websocket.Conn, channel string, values map[string]string) {
	values["sign"] = o.WebsocketSign(values)
	event := WebsocketEventAuthRemove{"removeChannel", channel, values}
	json, err := common.JSONEncode(event)
	if err != nil {
		log.Println(err)
		return
	}
	err = o.WebsocketConn.WriteMessage(websocket.TextMessage, json)

	if err != nil {
		log.Println(err)
		return
	}

	if o.Verbose {
		log.Printf("%s Removing authenticated channel: %s\n", o.GetName(), channel)
	}
}

// WebsocketClient starts a websocket client
func (o *OKCoin) WebsocketClient() {
	klineValues := []string{"1min", "3min", "5min", "15min", "30min", "1hour", "2hour", "4hour", "6hour", "12hour", "day", "3day", "week"}
	var currencyChan, userinfoChan string

	if o.WebsocketURL == okcoinWebsocketURLChina {
		currencyChan = okcoinWebsocketCNYRealTrades
		userinfoChan = okcoinWebsocketSpotCNYUserInfo
	} else {
		currencyChan = okcoinWebsocketUSDRealTrades
		userinfoChan = okcoinWebsocketSpotUSDUserInfo
	}

	for o.Enabled && o.Websocket {
		var Dialer websocket.Dialer
		var err error
		o.WebsocketConn, _, err = Dialer.Dial(o.WebsocketURL, http.Header{})

		if err != nil {
			log.Printf("%s Unable to connect to Websocket. Error: %s\n", o.GetName(), err)
			continue
		}

		if o.Verbose {
			log.Printf("%s Connected to Websocket.\n", o.GetName())
		}

		o.WebsocketConn.SetPingHandler(o.PingHandler)

		if o.AuthenticatedAPISupport {
			if o.WebsocketURL == okcoinWebsocketURL {
				o.AddChannelAuthenticated(okcoinWebsocketFuturesRealTrades, map[string]string{})
				o.AddChannelAuthenticated(okcoinWebsocketFuturesUserInfo, map[string]string{})
			}
			o.AddChannelAuthenticated(currencyChan, map[string]string{})
			o.AddChannelAuthenticated(userinfoChan, map[string]string{})
		}

		for _, x := range o.EnabledPairs {
			currency := common.StringToLower(x)
			currencyUL := currency[0:3] + "_" + currency[3:]
			if o.AuthenticatedAPISupport {
				o.WebsocketSpotOrderInfo(currencyUL, -1)
			}
			if o.WebsocketURL == okcoinWebsocketURL {
				o.AddChannel(fmt.Sprintf("ok_%s_future_index", currency))
				for _, y := range o.FuturesValues {
					if o.AuthenticatedAPISupport {
						o.WebsocketFuturesOrderInfo(currencyUL, y, -1, 1, 1, 50)
					}
					o.AddChannel(fmt.Sprintf("ok_%s_future_ticker_%s", currency, y))
					o.AddChannel(fmt.Sprintf("ok_%s_future_depth_%s_60", currency, y))
					o.AddChannel(fmt.Sprintf("ok_%s_future_trade_v1_%s", currency, y))
					for _, z := range klineValues {
						o.AddChannel(fmt.Sprintf("ok_future_%s_kline_%s_%s", currency, y, z))
					}
				}
			} else {
				o.AddChannel(fmt.Sprintf("ok_%s_ticker", currency))
				o.AddChannel(fmt.Sprintf("ok_%s_depth60", currency))
				o.AddChannel(fmt.Sprintf("ok_%s_trades_v1", currency))

				for _, y := range klineValues {
					o.AddChannel(fmt.Sprintf("ok_%s_kline_%s", currency, y))
				}
			}
		}

		for o.Enabled && o.Websocket {
			msgType, resp, err := o.WebsocketConn.ReadMessage()
			if err != nil {
				log.Println(err)
				break
			}
			switch msgType {
			case websocket.TextMessage:
				response := []interface{}{}
				err = common.JSONDecode(resp, &response)

				if err != nil {
					log.Println(err)
					continue
				}

				for _, y := range response {
					z := y.(map[string]interface{})
					channel := z["channel"]
					data := z["data"]
					success := z["success"]
					errorcode := z["errorcode"]
					channelStr, ok := channel.(string)

					if !ok {
						log.Println("Unable to convert channel to string")
						continue
					}

					if success != "true" && success != nil {
						errorCodeStr, ok := errorcode.(string)
						if !ok {
							log.Printf("%s Websocket: Unable to convert errorcode to string.\n", o.GetName())
							log.Printf("%s Websocket: channel %s error code: %s.\n", o.GetName(), channelStr, errorcode)
						} else {
							log.Printf("%s Websocket: channel %s error: %s.\n", o.GetName(), channelStr, o.WebsocketErrors[errorCodeStr])
						}
						continue
					}

					if success == "true" {
						if data == nil {
							continue
						}
					}

					dataJSON, err := common.JSONEncode(data)

					if err != nil {
						log.Println(err)
						continue
					}

					switch true {
					case common.StringContains(channelStr, "ticker") && !common.StringContains(channelStr, "future"):
						tickerValues := []string{"buy", "high", "last", "low", "sell", "timestamp"}
						tickerMap := data.(map[string]interface{})
						ticker := WebsocketTicker{}
						ticker.Vol = tickerMap["vol"].(string)

						for _, z := range tickerValues {
							result := reflect.TypeOf(tickerMap[z]).String()
							if result == "string" {
								value, errTickVals := strconv.ParseFloat(tickerMap[z].(string), 64)
								if errTickVals != nil {
									log.Println(errTickVals)
									continue
								}

								switch z {
								case "buy":
									ticker.Buy = value
								case "high":
									ticker.High = value
								case "last":
									ticker.Last = value
								case "low":
									ticker.Low = value
								case "sell":
									ticker.Sell = value
								case "timestamp":
									ticker.Timestamp = value
								}

							} else if result == "float64" {
								switch z {
								case "buy":
									ticker.Buy = tickerMap[z].(float64)
								case "high":
									ticker.High = tickerMap[z].(float64)
								case "last":
									ticker.Last = tickerMap[z].(float64)
								case "low":
									ticker.Low = tickerMap[z].(float64)
								case "sell":
									ticker.Sell = tickerMap[z].(float64)
								case "timestamp":
									ticker.Timestamp = tickerMap[z].(float64)
								}
							}
						}
					case common.StringContains(channelStr, "ticker") && common.StringContains(channelStr, "future"):
						ticker := WebsocketFuturesTicker{}
						err = common.JSONDecode(dataJSON, &ticker)

						if err != nil {
							log.Println(err)
							continue
						}
					case common.StringContains(channelStr, "depth"):
						orderbook := WebsocketOrderbook{}
						err = common.JSONDecode(dataJSON, &orderbook)

						if err != nil {
							log.Println(err)
							continue
						}
					case common.StringContains(channelStr, "trades_v1") || common.StringContains(channelStr, "trade_v1"):
						type TradeResponse struct {
							Data [][]string
						}

						trades := TradeResponse{}
						err = common.JSONDecode(dataJSON, &trades.Data)

						if err != nil {
							log.Println(err)
							continue
						}
						// to-do: convert from string array to trade struct
					case common.StringContains(channelStr, "kline"):
						klines := []interface{}{}

						err = common.JSONDecode(dataJSON, &klines)
						if err != nil {
							log.Println(err)
							continue
						}
					case common.StringContains(channelStr, "spot") && common.StringContains(channelStr, "realtrades"):
						if string(dataJSON) == "null" {
							continue
						}
						realtrades := WebsocketRealtrades{}

						err = common.JSONDecode(dataJSON, &realtrades)
						if err != nil {
							log.Println(err)
							continue
						}
					case common.StringContains(channelStr, "future") && common.StringContains(channelStr, "realtrades"):
						if string(dataJSON) == "null" {
							continue
						}
						realtrades := WebsocketFuturesRealtrades{}

						err = common.JSONDecode(dataJSON, &realtrades)
						if err != nil {
							log.Println(err)
							continue
						}
					case common.StringContains(channelStr, "spot") && common.StringContains(channelStr, "trade") || common.StringContains(channelStr, "futures") && common.StringContains(channelStr, "trade"):
						tradeOrder := WebsocketTradeOrderResponse{}

						err = common.JSONDecode(dataJSON, &tradeOrder)
						if err != nil {
							log.Println(err)
							continue
						}
					case common.StringContains(channelStr, "cancel_order"):
						cancelOrder := WebsocketTradeOrderResponse{}

						err = common.JSONDecode(dataJSON, &cancelOrder)
						if err != nil {
							log.Println(err)
							continue
						}
					case common.StringContains(channelStr, "spot") && common.StringContains(channelStr, "userinfo"):
						userinfo := WebsocketUserinfo{}

						err = common.JSONDecode(dataJSON, &userinfo)
						if err != nil {
							log.Println(err)
							continue
						}
					case common.StringContains(channelStr, "futureusd_userinfo"):
						userinfo := WebsocketFuturesUserInfo{}

						err = common.JSONDecode(dataJSON, &userinfo)
						if err != nil {
							log.Println(err)
							continue
						}
					case common.StringContains(channelStr, "spot") && common.StringContains(channelStr, "order_info"):
						type OrderInfoResponse struct {
							Result bool             `json:"result"`
							Orders []WebsocketOrder `json:"orders"`
						}
						var orders OrderInfoResponse

						err = common.JSONDecode(dataJSON, &orders)
						if err != nil {
							log.Println(err)
							continue
						}
					case common.StringContains(channelStr, "futureusd_order_info"):
						type OrderInfoResponse struct {
							Result bool                    `json:"result"`
							Orders []WebsocketFuturesOrder `json:"orders"`
						}
						var orders OrderInfoResponse

						err = common.JSONDecode(dataJSON, &orders)
						if err != nil {
							log.Println(err)
							continue
						}
					case common.StringContains(channelStr, "future_index"):
						index := WebsocketFutureIndex{}

						err = common.JSONDecode(dataJSON, &index)
						if err != nil {
							log.Println(err)
							continue
						}
					}
				}
			}
		}
		o.WebsocketConn.Close()
		log.Printf("%s Websocket client disconnected.", o.GetName())
	}
}

// SetWebsocketErrorDefaults sets default errors for websocket
func (o *OKCoin) SetWebsocketErrorDefaults() {
	o.WebsocketErrors = map[string]string{
		"10001": "Illegal parameters",
		"10002": "Authentication failure",
		"10003": "This connection has requested other user data",
		"10004": "This connection did not request this user data",
		"10005": "System error",
		"10009": "Order does not exist",
		"10010": "Insufficient funds",
		"10011": "Order quantity too low",
		"10012": "Only support btc_usd/btc_cny ltc_usd/ltc_cny",
		"10014": "Order price must be between 0 - 1,000,000",
		"10015": "Channel subscription temporally not available",
		"10016": "Insufficient coins",
		"10017": "WebSocket authorization error",
		"10100": "User frozen",
		"10216": "Non-public API",
		"20001": "User does not exist",
		"20002": "User frozen",
		"20003": "Frozen due to force liquidation",
		"20004": "Future account frozen",
		"20005": "User future account does not exist",
		"20006": "Required field can not be null",
		"20007": "Illegal parameter",
		"20008": "Future account fund balance is zero",
		"20009": "Future contract status error",
		"20010": "Risk rate information does not exist",
		"20011": `Risk rate bigger than 90% before opening position`,
		"20012": `Risk rate bigger than 90% after opening position`,
		"20013": "Temporally no counter party price",
		"20014": "System error",
		"20015": "Order does not exist",
		"20016": "Liquidation quantity bigger than holding",
		"20017": "Not authorized/illegal order ID",
		"20018": `Order price higher than 105% or lower than 95% of the price of last minute`,
		"20019": "IP restrained to access the resource",
		"20020": "Secret key does not exist",
		"20021": "Index information does not exist",
		"20022": "Wrong API interface",
		"20023": "Fixed margin user",
		"20024": "Signature does not match",
		"20025": "Leverage rate error",
	}
}
