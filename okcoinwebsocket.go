package main 

import (
	"log"
	"net/http"
	"net/url"
	"time"
	"fmt"
	"strings"
	"strconv"
	"github.com/gorilla/websocket"
)

var OKConnWebsocket *websocket.Conn

const (
	OKCOIN_WEBSOCKET_USD_REALTRADES = "ok_usd_realtrades"
	OKCOIN_WEBSOCKET_CNY_REALTRADES = "ok_cny_realtrades"
	OKCOIN_WEBSOCKET_SPOTUSD_TRADE = "ok_spotusd_trade"
	OKCOIN_WEBSOCKET_SPOTCNY_TRADE = "ok_spotcny_trade"
	OKCOIN_WEBSOCKET_SPOTUSD_CANCEL_ORDER = "ok_spotusd_cancel_order"
	OKCOIN_WEBSOCKET_SPOTCNY_CANCEL_ORDER = "ok_spotcny_cancel_order"
	OKCOIN_WEBSOCKET_SPOTUSD_USERINFO = "ok_spotusd_userinfo"
	OKCOIN_WEBSOCKET_SPOTCNY_USERINFO = "ok_spotcny_userinfo"
	OKCOIN_WEBSOCKET_SPOTUSD_ORDER_INFO = "ok_spotusd_order_info"
	OKCOIN_WEBSOCKET_SPOTCNY_ORDER_INFO = "ok_spotcny_order_info"
)

type OKCoinWebsocketTicker struct {
	Timestamp int64 `json:"timestamp,string"`
	Vol string `json:"vol"`
	Buy float64 `json:"buy,string"`
	High float64 `json:"high,string"`
	Last float64 `json:"last,string"`
	Low float64 `json:"low,string"`
	Sell float64 `json:"sell,string"`
}

type OKCoinWebsocketOrderbook struct {
	Asks [][]float64 `json:"asks"`
	Bids [][]float64 `json:"bids"`
	Timestamp int64 `json:"timestamp,string"`
}

type OKCoinWebsocketUserinfo struct {
	Info struct {
		Funds struct {
			Asset struct {
				Net float64 `json:"net,string"`
				Total float64 `json:"total,string"`
			} `json:"asset"`
			Free struct {
				BTC float64 `json:"btc,string"`
				LTC float64 `json:"ltc,string"`
				USD float64 `json:"usd,string"`
				CNY float64 `json:"cny,string"`
			} `json:"free"`
			Frozen struct {
				BTC float64 `json:"btc,string"`
				LTC float64 `json:"ltc,string"`
				USD float64 `json:"usd,string"`
				CNY float64 `json:"cny,string"`
			} `json:"freezed"`
		} `json:"funds"`
	} `json:"info"`
	Result bool `json:"result"`
}

type OKCoinWebsocketOrder struct {
	Amount float64 `json:"amount"`
	AvgPrice float64 `json:"avg_price"`
	DateCreated float64 `json:"create_date"`
	TradeAmount float64 `json:"deal_amount"`
	OrderID float64 `json:"order_id"`
	OrdersID float64 `json:"orders_id"`
	Price float64 `json:"price"`
	Status int64 `json:"status"`
	Symbol string `json:"symbol"`
	OrderType string `json:"type"`
}

type OKCoinWebsocketRealtrades struct {
	AveragePrice float64 `json:"averagePrice,string"`
	CompletedTradeAmount float64 `json:"completedTradeAmount,string"`
	DateCreated float64 `json:"createdDate"`
	ID float64 `json:"id"`
	OrderID float64 `json:"orderId"`
	SigTradeAmount float64 `json:"sigTradeAmount,string"`
	SigTradePrice float64 `json:"sigTradePrice,string"`
	Status int64 `json:"status"`
	Symbol string `json:"symbol"`
	TradeAmount float64 `json:"tradeAmount,string"`
	TradePrice float64 `json:"buy,string"`
	TradeType string `json:"tradeType"`
	TradeUnitPrice float64 `json:"tradeUnitPrice,string"`
	UnTrade float64 `json:"unTrade,string"`
}

type OKCoinWebsocketEvent struct {
	Event string `json:"event"`
	Channel string `json:"channel"`
}

type OKCoinWebsocketResponse struct {
	Channel string `json:"channel"`
	Data interface{} `json:"data"`
}

type OKCoinWebsocketEventAuth struct {
	Event string `json:"event"`
	Channel string `json:"channel"`
	Parameters map[string]string `json:"parameters"`
}

type OKCoinWebsocketEventAuthRemove struct {
	Event string `json:"event"`
	Channel string `json:"channel"`
	Parameters map[string]string `json:"parameters"`
}

func (o *OKCoin) PingHandler(message string) (error) {
	err := OKConnWebsocket.WriteControl(websocket.PingMessage, []byte("{'event':'ping'}"), time.Now().Add(time.Second))

	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}	

func (o *OKCoin) AddChannel(channel string) {
	event := OKCoinWebsocketEvent{"addChannel", channel}
	json, err := JSONEncode(event)
	if err != nil {
		log.Println(err)
		return
	}
	err = OKConnWebsocket.WriteMessage(websocket.TextMessage, json)

	if err != nil {
		log.Println(err)
		return
	}

	if o.Verbose {
		log.Printf("%s Adding channel: %s\n", o.GetName(), channel)
	}
}

func (o* OKCoin) RemoveChannel(channel string) {
	event := OKCoinWebsocketEvent{"removeChannel", channel}
	json, err := JSONEncode(event)
	if err != nil {
		log.Println(err)
		return
	}
	err = OKConnWebsocket.WriteMessage(websocket.TextMessage, json)

	if err != nil {
		log.Println(err)
		return
	}

	if o.Verbose {
		log.Printf("%s Removing channel: %s\n", o.GetName(), channel)
	}
}

func (o *OKCoin) WebsocketSpotTrade(symbol, orderType string, price, amount float64) {
	values := make(map[string]string)
	values["symbol"] = symbol
	values["type"] = orderType
	values["price"] = strconv.FormatFloat(price, 'f', 8, 64)
	values["amount"] = strconv.FormatFloat(amount, 'f', 8, 64)
	channel := ""

	if o.WebsocketURL == OKCOIN_WEBSOCKET_URL_CHINA {
		channel = OKCOIN_WEBSOCKET_SPOTCNY_TRADE
	} else {
		channel = OKCOIN_WEBSOCKET_SPOTUSD_TRADE
	}

	o.AddChannelAuthenticated(channel, values)
}

func (o *OKCoin) WebsocketSpotCancel(symbol string, orderID int64) {
	values := make(map[string]string)
	values["symbol"] = symbol
	values["order_id"] = strconv.FormatInt(orderID, 10)
	channel := ""

	if o.WebsocketURL == OKCOIN_WEBSOCKET_URL_CHINA {
		channel = OKCOIN_WEBSOCKET_SPOTCNY_CANCEL_ORDER
	} else {
		channel = OKCOIN_WEBSOCKET_SPOTUSD_CANCEL_ORDER
	}

	o.AddChannelAuthenticated(channel, values)
}

func (o *OKCoin) WebsocketSpotOrderInfo(symbol string, orderID int64) {
	values := make(map[string]string)
	values["symbol"] = symbol
	values["order_id"] = strconv.FormatInt(orderID, 10)
	channel := ""

	if o.WebsocketURL == OKCOIN_WEBSOCKET_URL_CHINA {
		channel = OKCOIN_WEBSOCKET_SPOTCNY_ORDER_INFO
	} else {
		channel = OKCOIN_WEBSOCKET_SPOTUSD_ORDER_INFO
	}

	o.AddChannelAuthenticated(channel, values)
}

func (o *OKCoin) ConvertToURLValues(values map[string]string) (url.Values) {
	urlVals := url.Values{}
	for i, x := range values {
		urlVals.Set(i, x)
	}
	return urlVals
}

func (o *OKCoin) WebsocketSign(values map[string]string) (string) {
	values["api_key"] = o.PartnerID
	urlVals := o.ConvertToURLValues(values)
	return strings.ToUpper(HexEncodeToString(GetMD5([]byte(urlVals.Encode() + "&secret_key=" + o.SecretKey))))
}

func (o *OKCoin) AddChannelAuthenticated(channel string, values map[string]string) {
	values["sign"] = o.WebsocketSign(values)
	event := OKCoinWebsocketEventAuth{"addChannel", channel, values}
	json, err := JSONEncode(event)
	if err != nil {
		log.Println(err)
		return
	}
	err = OKConnWebsocket.WriteMessage(websocket.TextMessage, json)

	if err != nil {
		log.Println(err)
		return
	}

	if o.Verbose {
		log.Printf("%s Adding authenticated channel: %s\n", o.GetName(), channel)
	}
}

func (o *OKCoin) RemoveChannelAuthenticated(conn *websocket.Conn, channel string, values map[string]string) {
	values["sign"] = o.WebsocketSign(values)
	event := OKCoinWebsocketEventAuthRemove{"removeChannel", channel, values}
	json, err := JSONEncode(event)
	if err != nil {
		log.Println(err)
		return
	}
	err = OKConnWebsocket.WriteMessage(websocket.TextMessage, json)

	if err != nil {
		log.Println(err)
		return
	}

	if o.Verbose {
		log.Printf("%s Removing authenticated channel: %s\n", o.GetName(), channel)
	}
}

func (o *OKCoin) WebsocketClient(currencies []string) {
	if len(currencies) == 0 {
		log.Println("No currencies for Websocket client specified.")
		return
	}

	var Dialer websocket.Dialer
	var err error
	var resp *http.Response
	OKConnWebsocket, resp, err = Dialer.Dial(o.WebsocketURL, http.Header{})

	if err != nil {
		log.Println(err)
		return
	}

	if o.Verbose {
		log.Printf("%s Connected to Websocket.", o.GetName())
		log.Println(resp)
	}

	OKConnWebsocket.SetPingHandler(o.PingHandler)
	
	if o.Verbose {
		log.Printf("%s Collecting order and userinfo.\n")
	}

	currencyChan, userinfoChan := "", ""
	if o.WebsocketURL == OKCOIN_WEBSOCKET_URL_CHINA {
		currencyChan = OKCOIN_WEBSOCKET_CNY_REALTRADES
		userinfoChan = OKCOIN_WEBSOCKET_SPOTCNY_USERINFO
		o.WebsocketSpotOrderInfo("btc_cny", -1)
		o.WebsocketSpotOrderInfo("ltc_cny", -1)
	} else {
		currencyChan = OKCOIN_WEBSOCKET_USD_REALTRADES
		userinfoChan = OKCOIN_WEBSOCKET_SPOTUSD_USERINFO
		o.WebsocketSpotOrderInfo("btc_usd", -1)
		o.WebsocketSpotOrderInfo("ltc_usd", -1)
	}
	o.AddChannelAuthenticated(currencyChan, map[string]string{})
	o.AddChannelAuthenticated(userinfoChan, map[string]string{})

	klineValues := []string{"1min", "3min", "5min", "15min", "30min", "1hour", "2hour", "4hour", "6hour", "12hour", "day", "3day", "week"}
	for _, x := range currencies {
		o.AddChannel(fmt.Sprintf("ok_%s_ticker", x))
		o.AddChannel(fmt.Sprintf("ok_%s_depth60", x))
		o.AddChannel(fmt.Sprintf("ok_%s_trades_v1", x))

		for _, y := range klineValues {
			o.AddChannel(fmt.Sprintf("ok_%s_kline_%s", x, y))
		}
	}

	for {
		msgType, resp, err := OKConnWebsocket.ReadMessage()
		if err != nil {
			log.Println(err)
			break
		}
		switch msgType {
		case websocket.TextMessage:
			response := []interface{}{}
			err = JSONDecode(resp, &response)

			if err != nil {
				log.Println(err)
				break
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

				if o.Verbose {
					log.Printf("%s Websocket channel message: %s\n", o.GetName(), channelStr)
				}

				dataJSON, err := JSONEncode(data)

				if err != nil {
					log.Println(err)
					continue
				}

				switch true {
				case strings.Contains(channelStr, "ticker"): 
					ticker := OKCoinWebsocketTicker{}
					err = JSONDecode(dataJSON, &ticker)

					if err != nil {
						log.Println(err)
						continue
					}
				case strings.Contains(channelStr, "depth60"): 
					orderbook := OKCoinWebsocketOrderbook{}
					err = JSONDecode(dataJSON, &orderbook)

					if err != nil {
						log.Println(err)
						continue
					}
				case strings.Contains(channelStr, "trades_v1"): 
					type TradeResponse struct {
						Data [][]string
					}

					trades := TradeResponse{}
					err = JSONDecode(dataJSON, &trades.Data)

					if err != nil {
						log.Println(err)
						continue
					}
					// to-do: convert from string array to trade struct
				case strings.Contains(channelStr, "kline"): 
					// to-do
				case strings.Contains(channelStr, "realtrades"):
					if success == "false" {
						log.Printf("Error subscribing to real trades channel, error code: %s", errorcode)
					} else {
						if string(dataJSON) == "null" {
							continue
						}
						realtrades := OKCoinWebsocketRealtrades{}
						err := JSONDecode(dataJSON, &realtrades)

						if err != nil {
							log.Println(err)
							continue
						}
					}
				case strings.Contains(channelStr, "spot") && strings.Contains(channelStr, "trade"):
					if success == "false" {
						log.Printf("Error placing trade, error code: %s", errorcode)
					} else {
						type TradeOrderResponse struct {
							OrderID int64 `json:"order_id,string"`
							Result bool `json:"result"`
						}
						tradeOrder := TradeOrderResponse{}
						err := JSONDecode(dataJSON, &tradeOrder)

						if err != nil {
							log.Println(err)
							continue
						}
					}
				case strings.Contains(channelStr, "userinfo"):
					if success == "false" {
						log.Printf("Error fetching user info, error code: %s", errorcode)
					} else {
						userinfo := OKCoinWebsocketUserinfo{}
						err = JSONDecode(dataJSON, &userinfo)

						if err != nil {
							log.Println(err)
							continue
						}
					}
				case strings.Contains(channelStr, "order_info"):
					if success == "false" {
						log.Printf("Error fetching order info, error code: %s", errorcode)
					} else {
						type OrderInfoResponse struct {
							Result bool `json:"result"`
							Orders []OKCoinWebsocketOrder `json:"orders"`
						}
						var orders OrderInfoResponse
						err := JSONDecode(dataJSON, &orders)

						if err != nil {
							log.Println(err)
							continue
						}
					}
				}
			}
		}
	}
	OKConnWebsocket.Close()
	log.Printf("%s Websocket client disconnected.", o.GetName())
}