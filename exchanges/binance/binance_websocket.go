package binance

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/astaxie/beego/logs"

	"github.com/agtorre/gocolorize"
	"github.com/gorilla/websocket"
	"github.com/idoall/TokenExchangeCommon/commonutils"
	"github.com/idoall/gocryptotrader/common"
)

const (
	binanceDefaultWebsocketURL = "wss://stream.binance.com:9443"
	binancePingPeriod          = 20 * time.Second
)

// WebsocketClient starts and handles the websocket client connection
func (b *Binance) WebsocketClient() {
	for b.Enabled && b.Websocket {
		var Dialer websocket.Dialer
		var err error
		// myenabledPairs := strings.ToLower(strings.Replace(strings.Join(b.EnabledPairs, "@ticker/"), "-", "", -1)) + "@trade"

		myenabledPairsTicker := strings.ToLower(strings.Replace(strings.Join(b.EnabledPairs, "@ticker/"), "-", "", -1)) + "@ticker"
		myenabledPairsTrade := strings.ToLower(strings.Replace(strings.Join(b.EnabledPairs, "@trade/"), "-", "", -1)) + "@trade"
		myenabledPairsKline := strings.ToLower(strings.Replace(strings.Join(b.EnabledPairs, "@kline_1m/"), "-", "", -1)) + "@kline_1m"
		wsurl := b.WebsocketURL + "/stream?streams=" + myenabledPairsTicker + "/" + myenabledPairsTrade + "/" + myenabledPairsKline

		// b.WebsocketConn, _, err = Dialer.Dial(binanceDefaultWebsocketURL+myenabledPairs, http.Header{})
		b.WebsocketConn, _, err = Dialer.Dial(wsurl, http.Header{})

		if err != nil {
			log.Printf("%s Unable to connect to Websocket. Error: %s\n", b.Name, err)
			continue
		}

		if b.Verbose {
			log.Printf("%s Connected to Websocket.\n", b.Name)
		}

		for b.Enabled && b.Websocket {
			msgType, resp, err := b.WebsocketConn.ReadMessage()
			if err != nil {
				log.Println(err)
				break
			}

			switch msgType {
			case websocket.TextMessage:
				multiStreamData := MultiStreamData{}

				err := common.JSONDecode(resp, &multiStreamData)

				if err != nil {
					log.Println("Could not load multi stream data.", string(resp))
					continue
				}

				if strings.Contains(multiStreamData.Stream, "trade") {
					trade := TradeStream{}
					err := common.JSONDecode(multiStreamData.Data, &trade)

					if err != nil {
						log.Println("Could not convert to a TradeStream structure")
						continue
					}
					log.Println("Trade received", trade.Symbol, trade.TimeStamp, trade.TradeID, trade.Price, trade.Quantity)
				} else if strings.Contains(multiStreamData.Stream, "ticker") {
					ticker := TickerStream{}

					err := common.JSONDecode(multiStreamData.Data, &ticker)
					if err != nil {
						log.Println("Could not convert to a TickerStream structure")
						continue
					}

					log.Println("Ticker received", ticker.Symbol, ticker.EventTime, ticker.TotalTradedVolume, ticker.LastTradeID)
				} else if strings.Contains(multiStreamData.Stream, "kline") {
					kline := KlineStream{}

					err := common.JSONDecode(multiStreamData.Data, &kline)
					if err != nil {
						log.Println("Could not convert to a KlineStream structure")
						continue
					}

					log.Println("Kline received", kline.Symbol, kline.EventTime, kline.Kline.HighPrice, kline.Kline.LowPrice)
				}
				type MsgType struct {
					MessageType string `json:"messageType"`
				}
			}
		}
		b.WebsocketConn.Close()
		log.Printf("%s Websocket client disconnected.", b.Name)
	}
}

func (b *Binance) UserDataWebsocket(urwr UserDataWebsocketRequest) (chan *AccountEvent, chan struct{}, error) {

	url := fmt.Sprintf("%s/ws/%s", binanceDefaultWebsocketURL, urwr.ListenKey)
	if b.Verbose {
		logs.Info(url)
	}
	c, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Fatal("dial:", err)
	}

	done := make(chan struct{})
	aech := make(chan *AccountEvent)

	go func() {
		defer c.Close()
		defer close(done)
		for {
			select {
			case <-done:
				fmt.Fprintln(os.Stdout, gocolorize.NewColor("green").Paint("closing reader"))
				// level.Info(as.Logger).Log("closing reader")
				return
			default:
				_, message, err := c.ReadMessage()
				if err != nil {
					// fmt.Fprintln(os.Stdout, gocolorize.NewColor("green").Paint("closing reader"))
					// level.Error(as.Logger).Log("wsRead", err)
					fmt.Println("wsRead", err)
					return
				}
				rawAccount := struct {
					Type            string  `json:"e"`
					Time            float64 `json:"E"`
					OpenTime        float64 `json:"t"`
					MakerCommision  int     `json:"m"`
					TakerCommision  int     `json:"t"`
					BuyerCommision  int     `json:"b"`
					SellerCommision int     `json:"s"`
					CanTrade        bool    `json:"T"`
					CanWithdraw     bool    `json:"W"`
					CanDeposit      bool    `json:"D"`
					Balances        []struct {
						Asset            string `json:"a"`
						AvailableBalance string `json:"f"`
						Locked           string `json:"l"`
					} `json:"B"`
				}{}
				if err := json.Unmarshal(message, &rawAccount); err != nil {
					fmt.Fprintln(os.Stdout, gocolorize.NewColor("red").Paint("wsUnmarshal", err, "body", string(message)))
					// level.Error(as.Logger).Log("wsUnmarshal", err, "body", string(message))
					return
				}
				t, err := commonutils.TimeFromUnixTimestampFloat(rawAccount.Time)
				if err != nil {
					fmt.Fprintln(os.Stdout, gocolorize.NewColor("red").Paint("wsUnmarshal", err, "body", rawAccount.Time))
					// level.Error(as.Logger).Log("wsUnmarshal", err, "body", rawAccount.Time)
					return
				}

				ae := &AccountEvent{
					WSEvent: WSEvent{
						Type: rawAccount.Type,
						Time: t,
					},
					Account: Account{
						MakerCommission:  rawAccount.MakerCommision,
						TakerCommission:  rawAccount.TakerCommision,
						BuyerCommission:  rawAccount.BuyerCommision,
						SellerCommission: rawAccount.SellerCommision,
						// MakerCommision:  rawAccount.MakerCommision,
						// TakerCommision:  rawAccount.TakerCommision,
						// BuyerCommision:  rawAccount.BuyerCommision,
						// SellerCommision: rawAccount.SellerCommision,
						CanTrade:    rawAccount.CanTrade,
						CanWithdraw: rawAccount.CanWithdraw,
						CanDeposit:  rawAccount.CanDeposit,
					},
				}
				for _, b := range rawAccount.Balances {
					free, err := commonutils.FloatFromString(b.AvailableBalance)
					if err != nil {
						fmt.Fprintln(os.Stdout, gocolorize.NewColor("red").Paint("wsUnmarshal", err, "body", b.AvailableBalance))

						// level.Error(as.Logger).Log("wsUnmarshal", err, "body", b.AvailableBalance)
						return
					}
					locked, err := commonutils.FloatFromString(b.Locked)
					if err != nil {
						fmt.Fprintln(os.Stdout, gocolorize.NewColor("red").Paint("wsUnmarshal", err, "body", b.Locked))
						// level.Error(as.Logger).Log("wsUnmarshal", err, "body", b.Locked)
						return
					}
					ae.Balances = append(ae.Balances, Balance{
						Asset:  b.Asset,
						Free:   fmt.Sprintf("%f", free),
						Locked: fmt.Sprintf("%f", locked),
					})
				}
				aech <- ae
			}
		}
	}()

	go b.exitHandler(c, done)
	return aech, done, nil
}

func (b *Binance) exitHandler(c *websocket.Conn, done chan struct{}) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	defer c.Close()

	for {
		select {
		case t := <-ticker.C:
			err := c.WriteMessage(websocket.TextMessage, []byte(t.String()))
			if err != nil {
				fmt.Fprintln(os.Stdout, gocolorize.NewColor("red").Paint("wsWrite", err))
				// level.Error(as.Logger).Log("wsWrite", err)
				return
			}
		case <-done:
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			fmt.Fprintln(os.Stdout, gocolorize.NewColor("red").Paint("closing connection"))
			// level.Info(as.Logger).Log("closing connection")
			return
		}
	}
}
