package okex

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/idoall/gocryptotrader/common"
)

const (
	okexDefaultWebsocketURL = "wss://real.okex.com:10440/websocket/okexapi"
)

func (o *OKEX) writeToWebsocket(message string) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	return o.WebsocketConn.WriteMessage(websocket.TextMessage, []byte(message))
}

func (o *OKEX) websocketConnect() {
	var Dialer websocket.Dialer
	var err error
	myEnabledSubscriptionChannels := []string{}

	for _, pair := range o.EnabledPairs {
		myEnabledSubscriptionChannels = append(myEnabledSubscriptionChannels, fmt.Sprintf("{'event':'addChannel','channel':'ok_sub_spot_%s_ticker'}", pair))
		myEnabledSubscriptionChannels = append(myEnabledSubscriptionChannels, fmt.Sprintf("{'event':'addChannel','channel':'ok_sub_spot_%s_depth'}", pair))
		myEnabledSubscriptionChannels = append(myEnabledSubscriptionChannels, fmt.Sprintf("{'event':'addChannel','channel':'ok_sub_spot_%s_deals'}", pair))
		myEnabledSubscriptionChannels = append(myEnabledSubscriptionChannels, fmt.Sprintf("{'event':'addChannel','channel':'ok_sub_spot_%s_kline_1min'}", pair))
	}

	mySubscriptionString := "[" + strings.Join(myEnabledSubscriptionChannels, ",") + "]"

	o.WebsocketConn, _, err = Dialer.Dial(okexDefaultWebsocketURL, http.Header{})

	if err != nil {
		log.Printf("%s Unable to connect to Websocket. Error: %s\n", o.Name, err)
		return
	}

	if o.Verbose {
		log.Printf("%s Connected to Websocket.\n", o.Name)
		log.Printf("Subscription String is %s\n", mySubscriptionString)
	}

	log.Printf("Subscription String is %s\n", mySubscriptionString)

	// subscribe to all the desired subscriptions
	err = o.writeToWebsocket(mySubscriptionString)

	if err != nil {
		log.Printf("Error: Could not subscribe to the OKEX websocket %s", err)
		return
	}
}

// WebsocketClient the main function handling the OKEX websocket
// Documentation URL: https://github.com/okcoin-okex/API-docs-OKEx.com/blob/master/API-For-Spot-EN/WEBSOCKET%20API%20for%20SPOT.md
func (o *OKEX) WebsocketClient() {
	for o.Enabled && o.Websocket {
		o.websocketConnect()

		go func() {
			for {
				time.Sleep(time.Second * 27)
				o.writeToWebsocket("{'event':'ping'}")
				log.Printf("%s sent Ping message\n", o.GetName())
			}
		}()

		for o.Enabled && o.Websocket {
			msgType, resp, err := o.WebsocketConn.ReadMessage()

			if err != nil {
				log.Printf("Error: Could not read from the OKEX websocket %s", err)
				o.websocketConnect()
				continue
			}

			switch msgType {
			case websocket.TextMessage:
				multiStreamDataArr := []MultiStreamData{}

				err = common.JSONDecode(resp, &multiStreamDataArr)

				if err != nil {
					if strings.Contains(string(resp), "pong") {
						log.Printf("%s received Pong message\n", o.GetName())
					} else {
						log.Printf("%s some other error happened: %s", o.GetName(), err)
						continue
					}
				}

				for _, multiStreamData := range multiStreamDataArr {
					if strings.Contains(multiStreamData.Channel, "ticker") {
						// ticker data
						ticker := TickerStreamData{}
						tickerDecodeError := common.JSONDecode(multiStreamData.Data, &ticker)

						if tickerDecodeError != nil {
							log.Printf("OKEX Ticker Decode Error: %s", tickerDecodeError)
							continue
						}

						log.Printf("OKEX Channel: %s\tData: %s\n", multiStreamData.Channel, multiStreamData.Data)
					} else if strings.Contains(multiStreamData.Channel, "deals") {
						// orderbook data
						deals := DealsStreamData{}
						decodeError := common.JSONDecode(multiStreamData.Data, &deals)

						if decodeError != nil {
							log.Printf("OKEX Deals Decode Error: %s", decodeError)
							continue
						}

						log.Printf("OKEX Channel: %s\tData: %s\n", multiStreamData.Channel, multiStreamData.Data)
					} else if strings.Contains(multiStreamData.Channel, "kline") {
						// 1 min kline data
						klines := KlineStreamData{}
						decodeError := common.JSONDecode(multiStreamData.Data, &klines)

						if decodeError != nil {
							log.Printf("OKEX Klines Decode Error: %s", decodeError)
							continue
						}

						log.Printf("OKEX Channel: %s\tData: %s\n", multiStreamData.Channel, multiStreamData.Data)
					} else if strings.Contains(multiStreamData.Channel, "depth") {
						// market depth data
						depth := DepthStreamData{}
						decodeError := common.JSONDecode(multiStreamData.Data, &depth)

						if decodeError != nil {
							log.Printf("OKEX Depth Decode Error: %s", decodeError)
							continue
						}

						log.Printf("OKEX Channel: %s\tData: %s\n", multiStreamData.Channel, multiStreamData.Data)
					}
				}
			}
		}
	}
}
