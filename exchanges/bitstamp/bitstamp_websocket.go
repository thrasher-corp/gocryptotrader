package bitstamp

import (
	"log"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/toorop/go-pusher"
)

// PusherOrderbook holds order book information to be pushed
type PusherOrderbook struct {
	Asks [][]string `json:"asks"`
	Bids [][]string `json:"bids"`
}

// PusherTrade holds trade information to be pushed
type PusherTrade struct {
	Price  float64 `json:"price"`
	Amount float64 `json:"amount"`
	ID     int64   `json:"id"`
}

const (
	// BitstampPusherKey holds the current pusher key
	BitstampPusherKey = "de504dc5763aeef9ff52"
)

// PusherClient starts the push mechanism
func (b *Bitstamp) PusherClient() {
	for b.Enabled && b.Websocket {
		pusherClient, err := pusher.NewClient(BitstampPusherKey)
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
				result := PusherOrderbook{}
				err := common.JSONDecode([]byte(data.Data), &result)
				if err != nil {
					log.Println(err)
				}
			case trade := <-tradeChannelTrade:
				result := PusherTrade{}
				err := common.JSONDecode([]byte(trade.Data), &result)
				if err != nil {
					log.Println(err)
				}
				log.Printf("%s Pusher trade: Price: %f Amount: %f\n", b.GetName(), result.Price, result.Amount)
			}
		}
	}
}
