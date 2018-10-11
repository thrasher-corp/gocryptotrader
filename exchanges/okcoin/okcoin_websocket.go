package okcoin

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges"
)

const (
	wsSubTicker         = "ok_sub_spot_%s_ticker"
	wsSubDepthIncrement = "ok_sub_spot_%s_depth"
	wsSubDepthFull      = "ok_sub_spot_%s_depth_%s"
	wsSubTrades         = "ok_sub_spot_%s_deals"
	wsSubKline          = "ok_sub_spot_%s_kline_%s"
)

// PingHandler handles the keep alive
func (o *OKCoin) PingHandler(message string) error {
	return o.WebsocketConn.WriteControl(websocket.PingMessage,
		[]byte("{'event':'ping'}"),
		time.Now().Add(time.Second))
}

// AddChannel adds a new channel on the websocket client
func (o *OKCoin) AddChannel(channel string) error {
	event := WebsocketEvent{"addChannel", channel}
	json, err := common.JSONEncode(event)
	if err != nil {
		return err
	}

	return o.WebsocketConn.WriteMessage(websocket.TextMessage, json)
}

// WsConnect initiates a websocket connection
func (o *OKCoin) WsConnect() error {
	if !o.Websocket.IsEnabled() || !o.IsEnabled() {
		return errors.New(exchange.WebsocketNotEnabled)
	}

	klineValues := []string{"1min", "3min", "5min", "15min", "30min", "1hour",
		"2hour", "4hour", "6hour", "12hour", "day", "3day", "week"}

	var dialer websocket.Dialer

	if o.Websocket.GetProxyAddress() != "" {
		proxy, err := url.Parse(o.Websocket.GetProxyAddress())
		if err != nil {
			return err
		}

		dialer.Proxy = http.ProxyURL(proxy)
	}

	var err error
	o.WebsocketConn, _, err = dialer.Dial(o.Websocket.GetWebsocketURL(),
		http.Header{})
	if err != nil {
		return err
	}

	o.WebsocketConn.SetPingHandler(o.PingHandler)

	go o.WsReadData()
	go o.WsHandleData()

	for _, p := range o.GetEnabledCurrencies() {
		fPair := exchange.FormatExchangeCurrency(o.GetName(), p)

		o.AddChannel(fmt.Sprintf(wsSubDepthFull, fPair.String(), "20"))
		o.AddChannel(fmt.Sprintf(wsSubKline, fPair.String(), klineValues[0]))
		o.AddChannel(fmt.Sprintf(wsSubTicker, fPair.String()))
		o.AddChannel(fmt.Sprintf(wsSubTrades, fPair.String()))
	}

	return nil
}

// WsReadData reads from the websocket connection
func (o *OKCoin) WsReadData() {
	o.Websocket.Wg.Add(1)

	defer func() {
		err := o.WebsocketConn.Close()
		if err != nil {
			o.Websocket.DataHandler <- fmt.Errorf("okcoin_websocket.go - Unable to to close Websocket connection. Error: %s",
				err)
		}
		o.Websocket.Wg.Done()
	}()

	for {
		select {
		case <-o.Websocket.ShutdownC:
			return

		default:
			_, resp, err := o.WebsocketConn.ReadMessage()
			if err != nil {
				o.Websocket.DataHandler <- err
				return
			}

			o.Websocket.TrafficAlert <- struct{}{}
			o.Websocket.Intercomm <- exchange.WebsocketResponse{Raw: resp}
		}
	}

}

// WsHandleData handles stream data from the websocket connection
func (o *OKCoin) WsHandleData() {
	o.Websocket.Wg.Add(1)
	defer o.Websocket.Wg.Done()

	for {
		select {
		case <-o.Websocket.ShutdownC:
			return

		case resp := <-o.Websocket.Intercomm:
			var init []WsResponse
			err := common.JSONDecode(resp.Raw, &init)
			if err != nil {
				log.Fatal(err)
			}

			if init[0].ErrorCode != "" {
				log.Fatal(o.WebsocketErrors[init[0].ErrorCode])
			}

			if init[0].Success {
				if init[0].Data == nil {
					continue
				}
			}

			if init[0].Channel == "addChannel" {
				continue
			}

			var currencyPairSlice []string
			splitChar := common.SplitStrings(init[0].Channel, "_")
			currencyPairSlice = append(currencyPairSlice,
				common.StringToUpper(splitChar[3]))
			currencyPairSlice = append(currencyPairSlice,
				common.StringToUpper(splitChar[4]))
			currencyPair := common.JoinStrings(currencyPairSlice, "-")

			assetType := common.StringToUpper(splitChar[2])

			switch {
			case common.StringContains(init[0].Channel, "ticker") &&
				common.StringContains(init[0].Channel, "spot"):
				var ticker WsTicker

				err = common.JSONDecode(init[0].Data, &ticker)
				if err != nil {
					log.Fatal(err)

				}

				o.Websocket.DataHandler <- exchange.TickerData{
					Timestamp:  time.Unix(0, ticker.Timestamp),
					Pair:       pair.NewCurrencyPairFromString(currencyPair),
					AssetType:  assetType,
					Exchange:   o.GetName(),
					ClosePrice: ticker.Close,
					OpenPrice:  ticker.Open,
					HighPrice:  ticker.Last,
					LowPrice:   ticker.Low,
					Quantity:   ticker.Volume,
				}

			case common.StringContains(init[0].Channel, "depth"):
				var orderbook WsOrderbook

				err = common.JSONDecode(init[0].Data, &orderbook)
				if err != nil {
					log.Fatal(err)
				}

				o.Websocket.DataHandler <- exchange.WebsocketOrderbookUpdate{
					Pair:     pair.NewCurrencyPairFromString(currencyPair),
					Exchange: o.GetName(),
					Asset:    assetType,
				}

			case common.StringContains(init[0].Channel, "kline"):
				var klineData [][]interface{}

				err = common.JSONDecode(init[0].Data, &klineData)
				if err != nil {
					log.Fatal(err)
				}

				var klines []WsKlines
				for _, data := range klineData {
					var newKline WsKlines

					newKline.Timestamp, _ = strconv.ParseInt(data[0].(string), 10, 64)
					newKline.Open, _ = strconv.ParseFloat(data[1].(string), 64)
					newKline.High, _ = strconv.ParseFloat(data[1].(string), 64)
					newKline.Low, _ = strconv.ParseFloat(data[1].(string), 64)
					newKline.Close, _ = strconv.ParseFloat(data[1].(string), 64)
					newKline.Volume, _ = strconv.ParseFloat(data[1].(string), 64)

					klines = append(klines, newKline)
				}

				for _, data := range klines {
					o.Websocket.DataHandler <- exchange.KlineData{
						Timestamp:  time.Unix(0, data.Timestamp),
						Pair:       pair.NewCurrencyPairFromString(currencyPair),
						AssetType:  assetType,
						Exchange:   o.GetName(),
						OpenPrice:  data.Open,
						ClosePrice: data.Close,
						HighPrice:  data.High,
						LowPrice:   data.Low,
						Volume:     data.Volume,
					}
				}

			case common.StringContains(init[0].Channel, "spot") &&
				common.StringContains(init[0].Channel, "deals"):
				var dealsData [][]interface{}
				err = common.JSONDecode(init[0].Data, &dealsData)
				if err != nil {
					log.Fatal(err)
				}

				var deals []WsDeals
				for _, data := range dealsData {
					var newDeal WsDeals
					newDeal.TID, _ = strconv.ParseInt(data[0].(string), 10, 64)
					newDeal.Price, _ = strconv.ParseFloat(data[1].(string), 64)
					newDeal.Amount, _ = strconv.ParseFloat(data[2].(string), 64)
					newDeal.Timestamp, _ = data[3].(string)
					newDeal.Type, _ = data[4].(string)

					deals = append(deals, newDeal)
				}
			}
		}
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

// WsOrderbook defines orderbook data from websocket connection
type WsOrderbook struct {
	Asks      [][]string `json:"asks"`
	Bids      [][]string `json:"bids"`
	Timestamp int64      `json:"timestamp"`
}

// WsResponse defines initial response stream
type WsResponse struct {
	Channel   string          `json:"channel"`
	Result    bool            `json:"result"`
	Success   bool            `json:"success"`
	ErrorCode string          `json:"errorcode"`
	Data      json.RawMessage `json:"data"`
}

// WsKlines defines a Kline response data from the websocket connection
type WsKlines struct {
	Timestamp int64
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
}

// WsTicker holds ticker data for websocket
type WsTicker struct {
	High      float64 `json:"high,string"`
	Volume    float64 `json:"vol,string"`
	Last      float64 `json:"last,string"`
	Low       float64 `json:"low,string"`
	Buy       float64 `json:"buy,string"`
	Change    float64 `json:"change,string"`
	Sell      float64 `json:"sell,string"`
	DayLow    float64 `json:"dayLow,string"`
	Close     float64 `json:"close,string"`
	DayHigh   float64 `json:"dayHigh,string"`
	Open      float64 `json:"open,string"`
	Timestamp int64   `json:"timestamp"`
}

// WsDeals defines a deal response from the websocket connection
type WsDeals struct {
	TID       int64
	Price     float64
	Amount    float64
	Timestamp string
	Type      string
}
