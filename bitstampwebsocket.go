package main

import (
	"github.com/toorop/go-pusher"
	"log"
)

type BitstampPusherOrderbook struct {
	Asks[][] string `json:"asks"`
	Bids[][] string `json:"bids"`
}
type BitstampPusherTrade struct {
	Price float64 `json:"price"`
	Amount float64 `json:"amount"`
	ID int64 `json:"id"`
}

const (
	BITSTAMP_PUSHER_KEY = "de504dc5763aeef9ff52" 
)

func (b *Bitstamp) PusherClient() {
	pusherClient, err := pusher.NewClient(BITSTAMP_PUSHER_KEY)
	if err != nil {
		log.Fatalln(err)
		return
	}

	err = pusherClient.Subscribe("live_trades")
	if err != nil {
		log.Println("Subscription error : ", err)
		return
	}

	err = pusherClient.Subscribe("order_book")
	if err != nil {
		log.Println("Subscription error : ", err)
		return
	}

	dataChannelTrade, err := pusherClient.Bind("data")
	if err != nil {
		log.Println("Bind error: ", err)
		return
	}
	tradeChannelTrade, err := pusherClient.Bind("trade")
	if err != nil {
		log.Println("Bind error: ", err)
		return
	}

	log.Printf("%s Pusher client ready.\n", b.GetName())

	for b.Websocket {
		select {
		case data := <-dataChannelTrade:
			result := BitstampPusherOrderbook{}
			err := JSONDecode([]byte(data.Data), &result)
			if err != nil {
				log.Println(err)
			}
		case trade := <-tradeChannelTrade:
			result := BitstampPusherTrade{}
			err := JSONDecode([]byte(trade.Data), &result)
			if err != nil {
				log.Println(err)
			}
			log.Printf("%s Pusher trade: Price: %f Amount: %f\n", b.GetName(), result.Price, result.Amount)
		}
	}
}