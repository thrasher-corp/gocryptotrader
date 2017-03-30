package bitstamp

import (
	"log"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/toorop/go-pusher"
)

type BitstampPusherOrderbook struct {
	Asks [][]string `json:"asks"`
	Bids [][]string `json:"bids"`
}
type BitstampPusherTrade struct {
	Price  float64 `json:"price"`
	Amount float64 `json:"amount"`
	ID     int64   `json:"id"`
}

const (
	BITSTAMP_PUSHER_KEY = "de504dc5763aeef9ff52"
)

func (b *Bitstamp) PusherClient() {
	for b.Enabled && b.Websocket {
		pusherClient, err := pusher.NewClient(BITSTAMP_PUSHER_KEY)
		if err != nil {
			log.Printf("%s Unable to connect to Websocket. Error: %s\n", b.GetName(), err)
			continue
		}

		err = pusherClient.Subscribe("live_trades")
		if err != nil {
			log.Printf("%s Websocket Trade subscription error: %s\n", b.GetName(), err)
		}

		err = pusherClient.Subscribe("order_book")
		if err != nil {
			log.Printf("%s Websocket Trade subscription error: %s\n", b.GetName(), err)
		}

		dataChannelTrade, err := pusherClient.Bind("data")
		if err != nil {
			log.Printf("%s Websocket Bind error: %s\n", b.GetName(), err)
			continue
		}
		tradeChannelTrade, err := pusherClient.Bind("trade")
		if err != nil {
			log.Printf("%s Websocket Bind error: %s\n", b.GetName(), err)
			continue
		}

		log.Printf("%s Pusher client connected.\n", b.GetName())

		for b.Websocket {
			select {
			case data := <-dataChannelTrade:
				result := BitstampPusherOrderbook{}
				err := common.JSONDecode([]byte(data.Data), &result)
				if err != nil {
					log.Println(err)
				}
			case trade := <-tradeChannelTrade:
				result := BitstampPusherTrade{}
				err := common.JSONDecode([]byte(trade.Data), &result)
				if err != nil {
					log.Println(err)
				}
				log.Printf("%s Pusher trade: Price: %f Amount: %f\n", b.GetName(), result.Price, result.Amount)
			}
		}
	}
}
