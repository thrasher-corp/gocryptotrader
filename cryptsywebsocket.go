package main

import (
	"github.com/toorop/go-pusher"
	"log"
	"strings"
)

type CryptsyPusherTrade struct {
	Channel string `json:"channel"`
	Trade   struct {
		Datetime   string  `json:"datetime"`
		MarketID   string  `json:"marketid"`
		MarketName string  `json:"marketname"`
		Price      float64 `json:"price,string"`
		Quantity   float64 `json:"quantity,string"`
		Timestamp  int64   `json:"timestamp"`
		Total      string  `json:"total"`
		Type       string  `json:"type"`
	} `json:"trade"`
}

type CryptsyPusherTicker struct {
	Channel string `json:"channel"`
	Trade   struct {
		Datetime  string `json:"datetime"`
		MarketID  string `json:"marketid"`
		Timestamp int64  `json:"timestamp"`
		TopBuy    struct {
			Price     float64 `json:"price,string"`
			Quantitiy float64 `json:"quantity,string"`
		} `json:"topbuy"`
		TopSell struct {
			Price    float64 `json:"price,string"`
			Quantity float64 `json:"quantity,string"`
		} `json:"topsell"`
	} `json:"trade"`
}

const (
	CRYPTSY_PUSHER_KEY = "cb65d0a7a72cd94adf1f"
)

func (c *Cryptsy) PusherClient(marketID []string) {
	for c.Enabled && c.Websocket {
		pusherClient, err := pusher.NewClient(CRYPTSY_PUSHER_KEY)
		if err != nil {
			log.Printf("%s Unable to connect to Websocket. Error: %s\n", c.GetName(), err)
			continue
		}

		for i := 0; i < len(marketID); i++ {
			err = pusherClient.Subscribe("trade." + marketID[i])

			if err != nil {
				log.Printf("%s Websocket Trade subscription error: %s\n", c.GetName(), err)
			}

			err = pusherClient.Subscribe("ticker." + marketID[i])
			if err != nil {
				log.Printf("%s Websocket Trade subscription error: %s\n", c.GetName(), err)
			}
		}

		dataChannel, err := pusherClient.Bind("message")
		if err != nil {
			log.Printf("%s Websocket Bind error: ", c.GetName(), err)
			continue
		}
		log.Printf("%s Pusher client connected.\n", c.GetName())

		for c.Enabled && c.Websocket {
			select {
			case data := <-dataChannel:
				if strings.Contains(data.Data, "topbuy") {
					result := CryptsyPusherTicker{}
					err := JSONDecode([]byte(data.Data), &result)
					if err != nil {
						log.Println(err)
						continue
					}
				} else {
					result := CryptsyPusherTrade{}
					err := JSONDecode([]byte(data.Data), &result)
					if err != nil {
						log.Println(err)
						continue
					}
					log.Printf("%s Pusher trade - market %s - Price %f Amount %f Type %s\n", c.GetName(), result.Channel, result.Trade.Price, result.Trade.Quantity, result.Trade.Type)
				}
			}
		}
	}
}
