package bitstamp

import (
	"errors"
	"fmt"
	"log"
	"strings"

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

// findPairFromChannel extracts the capitalized trading pair from the channel and returns it only if enabled in the config
func (b *Bitstamp) findPairFromChannel(channelName string) (string, error) {
	split := strings.Split(channelName, "_")
	tradingPair := strings.ToUpper(split[len(split)-1])

	for _, enabledPair := range b.EnabledPairs {
		if enabledPair == tradingPair {
			return tradingPair, nil
		}
	}

	return "", errors.New("Could not find trading pair")
}

// PusherClient starts the push mechanism
func (b *Bitstamp) PusherClient() {
	for b.Enabled && b.Websocket {
		// hold the mapping of channel:tradingPair in order not to always compute it
		seenTradingPairs := map[string]string{}

		pusherClient, err := pusher.NewClient(BitstampPusherKey)
		if err != nil {
			log.Printf("%s Unable to connect to Websocket. Error: %s\n", b.GetName(), err)
			continue
		}

		for _, pair := range b.EnabledPairs {
			err = pusherClient.Subscribe(fmt.Sprintf("live_trades_%s", strings.ToLower(pair)))
			if err != nil {
				log.Printf("%s Websocket Trade subscription error: %s\n", b.GetName(), err)
			}

			err = pusherClient.Subscribe(fmt.Sprintf("order_book_%s", strings.ToLower(pair)))
			if err != nil {
				log.Printf("%s Websocket Trade subscription error: %s\n", b.GetName(), err)
			}
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
				var channelTradingPair string
				var ok bool

				if channelTradingPair, ok = seenTradingPairs[data.Channel]; !ok {
					if foundTradingPair, noPair := b.findPairFromChannel(data.Channel); noPair == nil {
						seenTradingPairs[data.Channel] = foundTradingPair
					} else {
						log.Printf("%s Pair from Channel: %s does not seem to be enabled or found", b.GetName(), data.Channel)
						continue
					}
				}

				log.Printf("%s Pusher: received ticker for Pair: %s\n", b.GetName(), channelTradingPair)

				if err != nil {
					log.Println(err)
				}
			case trade := <-tradeChannelTrade:
				result := PusherTrade{}
				err := common.JSONDecode([]byte(trade.Data), &result)

				if err != nil {
					log.Println(err)
				}

				var channelTradingPair string
				var ok bool

				if channelTradingPair, ok = seenTradingPairs[trade.Channel]; !ok {
					if foundTradingPair, noPair := b.findPairFromChannel(trade.Channel); noPair == nil {
						seenTradingPairs[trade.Channel] = foundTradingPair
					} else {
						log.Printf("%s LiveTrade Pair from Channel: %s does not seem to be enabled or found", b.GetName(), trade.Channel)
						continue
					}
				}

				log.Println(trade.Channel)
				log.Printf("%s Pusher trade: Pair: %s Price: %f Amount: %f\n", b.GetName(), channelTradingPair, result.Price, result.Amount)
			}
		}
	}
}
