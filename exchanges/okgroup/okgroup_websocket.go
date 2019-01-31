package okgroup

import (
	"bytes"
	"compress/flate"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	log "github.com/thrasher-/gocryptotrader/logger"
)

const (
	okexDefaultWebsocketURL = "wss://real.okex.com:10440/websocket/okexapi"
)

func (o *OKGroup) writeToWebsocket(message string) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	return o.WebsocketConn.WriteMessage(websocket.TextMessage, []byte(message))
}

// WsConnect initiates a websocket connection
func (o *OKGroup) WsConnect() error {
	if !o.Websocket.IsEnabled() || !o.IsEnabled() {
		return errors.New(exchange.WebsocketNotEnabled)
	}

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
		return fmt.Errorf("%s Unable to connect to Websocket. Error: %s",
			o.Name,
			err)
	}

	go o.WsHandleData()
	go o.wsPingHandler()

	err = o.WsSubscribe()
	if err != nil {
		return fmt.Errorf("Error: Could not subscribe to the OKEX websocket %s",
			err)
	}

	return nil
}

// WsSubscribe subscribes to the websocket channels
func (o *OKGroup) WsSubscribe() error {
	myEnabledSubscriptionChannels := []string{}

	for _, pair := range o.EnabledPairs {

		// ----------- deprecate when usd pairs are upgraded to usdt ----------
		checkSymbol := common.SplitStrings(pair, "_")
		for i := range checkSymbol {
			if common.StringContains(checkSymbol[i], "usdt") {
				break
			}
			if common.StringContains(checkSymbol[i], "usd") {
				checkSymbol[i] = "usdt"
			}
		}

		symbolRedone := common.JoinStrings(checkSymbol, "_")
		// ----------- deprecate when usd pairs are upgraded to usdt ----------

		myEnabledSubscriptionChannels = append(myEnabledSubscriptionChannels,
			fmt.Sprintf("{'event':'addChannel','channel':'ok_sub_spot_%s_ticker'}",
				symbolRedone))

		myEnabledSubscriptionChannels = append(myEnabledSubscriptionChannels,
			fmt.Sprintf("{'event':'addChannel','channel':'ok_sub_spot_%s_depth'}",
				symbolRedone))

		myEnabledSubscriptionChannels = append(myEnabledSubscriptionChannels,
			fmt.Sprintf("{'event':'addChannel','channel':'ok_sub_spot_%s_deals'}",
				symbolRedone))

		myEnabledSubscriptionChannels = append(myEnabledSubscriptionChannels,
			fmt.Sprintf("{'event':'addChannel','channel':'ok_sub_spot_%s_kline_1min'}",
				symbolRedone))
	}

	for _, outgoing := range myEnabledSubscriptionChannels {
		err := o.writeToWebsocket(outgoing)
		if err != nil {
			return err
		}
	}

	return nil
}

// WsReadData reads data from the websocket connection
func (o *OKGroup) WsReadData() (exchange.WebsocketResponse, error) {
	mType, resp, err := o.WebsocketConn.ReadMessage()
	if err != nil {
		return exchange.WebsocketResponse{}, err
	}

	o.Websocket.TrafficAlert <- struct{}{}

	var standardMessage []byte

	switch mType {
	case websocket.TextMessage:
		standardMessage = resp

	case websocket.BinaryMessage:
		reader := flate.NewReader(bytes.NewReader(resp))
		standardMessage, err = ioutil.ReadAll(reader)
		reader.Close()
		if err != nil {
			return exchange.WebsocketResponse{}, err
		}
	}

	return exchange.WebsocketResponse{Raw: standardMessage}, nil
}

func (o *OKGroup) wsPingHandler() {
	o.Websocket.Wg.Add(1)
	defer o.Websocket.Wg.Done()

	ticker := time.NewTicker(time.Second * 27)

	for {
		select {
		case <-o.Websocket.ShutdownC:
			return

		case <-ticker.C:
			err := o.writeToWebsocket("{'event':'ping'}")
			if err != nil {
				o.Websocket.DataHandler <- err
				return
			}
		}
	}
}

// WsHandleData handles the read data from the websocket connection
func (o *OKGroup) WsHandleData() {
	o.Websocket.Wg.Add(1)

	defer func() {
		err := o.WebsocketConn.Close()
		if err != nil {
			o.Websocket.DataHandler <- fmt.Errorf("okex_websocket.go - Unable to to close Websocket connection. Error: %s",
				err)
		}
		o.Websocket.Wg.Done()
	}()

	for {
		select {
		case <-o.Websocket.ShutdownC:
			return

		default:
			resp, err := o.WsReadData()
			if err != nil {
				o.Websocket.DataHandler <- err
				return
			}

			multiStreamDataArr := []MultiStreamData{}

			err = common.JSONDecode(resp.Raw, &multiStreamDataArr)
			if err != nil {
				if strings.Contains(string(resp.Raw), "pong") {
					continue
				} else {
					o.Websocket.DataHandler <- err
					continue
				}
			}

			for _, multiStreamData := range multiStreamDataArr {
				var errResponse ErrorResponse
				if common.StringContains(string(resp.Raw), "error_msg") {
					err = common.JSONDecode(resp.Raw, &errResponse)
					if err != nil {
						log.Error(err)
					}
					o.Websocket.DataHandler <- fmt.Errorf("okex.go error - %s resp: %s ",
						errResponse.ErrorMsg,
						string(resp.Raw))
					continue
				}

				var newPair string
				var assetType string
				currencyPairSlice := common.SplitStrings(multiStreamData.Channel, "_")
				if len(currencyPairSlice) > 5 {
					newPair = currencyPairSlice[3] + "_" + currencyPairSlice[4]
					assetType = currencyPairSlice[2]
				}

				if strings.Contains(multiStreamData.Channel, "ticker") {
					var ticker TickerStreamData

					err = common.JSONDecode(multiStreamData.Data, &ticker)
					if err != nil {
						o.Websocket.DataHandler <- err
						continue
					}

					o.Websocket.DataHandler <- exchange.TickerData{
						Timestamp: time.Unix(0, int64(ticker.Timestamp)),
						Exchange:  o.GetName(),
						AssetType: assetType,
					}

				} else if strings.Contains(multiStreamData.Channel, "deals") {
					var deals DealsStreamData

					err = common.JSONDecode(multiStreamData.Data, &deals)
					if err != nil {
						o.Websocket.DataHandler <- err
						continue
					}

					for _, trade := range deals {
						price, _ := strconv.ParseFloat(trade[1], 64)
						amount, _ := strconv.ParseFloat(trade[2], 64)
						time, _ := time.Parse(time.RFC3339, trade[3])

						o.Websocket.DataHandler <- exchange.TradeData{
							Timestamp:    time,
							Exchange:     o.GetName(),
							AssetType:    assetType,
							CurrencyPair: pair.NewCurrencyPairFromString(newPair),
							Price:        price,
							Amount:       amount,
							EventType:    trade[4],
						}
					}

				} else if strings.Contains(multiStreamData.Channel, "kline") {
					var klines KlineStreamData

					err := common.JSONDecode(multiStreamData.Data, &klines)
					if err != nil {
						o.Websocket.DataHandler <- err
						continue
					}

					for _, kline := range klines {
						ntime, _ := strconv.ParseInt(kline[0], 10, 64)
						open, _ := strconv.ParseFloat(kline[1], 64)
						high, _ := strconv.ParseFloat(kline[2], 64)
						low, _ := strconv.ParseFloat(kline[3], 64)
						close, _ := strconv.ParseFloat(kline[4], 64)
						volume, _ := strconv.ParseFloat(kline[5], 64)

						o.Websocket.DataHandler <- exchange.KlineData{
							Timestamp:  time.Unix(ntime, 0),
							Pair:       pair.NewCurrencyPairFromString(newPair),
							AssetType:  assetType,
							Exchange:   o.GetName(),
							OpenPrice:  open,
							HighPrice:  high,
							LowPrice:   low,
							ClosePrice: close,
							Volume:     volume,
						}
					}

				} else if strings.Contains(multiStreamData.Channel, "depth") {
					var depth DepthStreamData

					err := common.JSONDecode(multiStreamData.Data, &depth)
					if err != nil {
						o.Websocket.DataHandler <- err
						continue
					}

					o.Websocket.DataHandler <- exchange.WebsocketOrderbookUpdate{
						Exchange: o.GetName(),
						Asset:    assetType,
						Pair:     pair.NewCurrencyPairFromString(newPair),
					}
				}
			}
		}
	}
}

// ErrorResponse defines an error response type from the websocket connection
type ErrorResponse struct {
	Result    bool   `json:"result"`
	ErrorMsg  string `json:"error_msg"`
	ErrorCode int64  `json:"error_code"`
}

// Request defines the JSON request structure to the websocket server
type Request struct {
	Event      string `json:"event"`
	Channel    string `json:"channel"`
	Parameters string `json:"parameters,omitempty"`
}
