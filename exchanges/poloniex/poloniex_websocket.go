package poloniex

import (
	"log"
	"strconv"

	"github.com/beatgammit/turnpike"
)

const (
	poloniexWebsocketAddress  = "wss://api.poloniex.com"
	poloniexWebsocketRealm    = "realm1"
	poloniexWebsocketTicker   = "ticker"
	poloniexWebsocketTrollbox = "trollbox"
)

// OnTicker converts ticker data to a websocketTicker
func OnTicker(args []interface{}, kwargs map[string]interface{}) {
	ticker := WebsocketTicker{}
	ticker.CurrencyPair = args[0].(string)
	ticker.Last, _ = strconv.ParseFloat(args[1].(string), 64)
	ticker.LowestAsk, _ = strconv.ParseFloat(args[2].(string), 64)
	ticker.HighestBid, _ = strconv.ParseFloat(args[3].(string), 64)
	ticker.PercentChange, _ = strconv.ParseFloat(args[4].(string), 64)
	ticker.BaseVolume, _ = strconv.ParseFloat(args[5].(string), 64)
	ticker.QuoteVolume, _ = strconv.ParseFloat(args[6].(string), 64)

	if args[7].(float64) != 0 {
		ticker.IsFrozen = true
	} else {
		ticker.IsFrozen = false
	}

	ticker.High, _ = strconv.ParseFloat(args[8].(string), 64)
	ticker.Low, _ = strconv.ParseFloat(args[9].(string), 64)
}

// OnTrollbox handles trollbox messages
func OnTrollbox(args []interface{}, kwargs map[string]interface{}) {
	message := WebsocketTrollboxMessage{}
	message.MessageNumber, _ = args[1].(float64)
	message.Username = args[2].(string)
	message.Message = args[3].(string)
	if len(args) == 5 {
		message.Reputation = args[4].(float64)
	}
}

// OnDepthOrTrade handles orderbook depth and trade events
func OnDepthOrTrade(args []interface{}, kwargs map[string]interface{}) {
	for x := range args {
		data := args[x].(map[string]interface{})
		msgData := data["data"].(map[string]interface{})
		msgType := data["type"].(string)

		switch msgType {
		case "orderBookModify":
			{
				type PoloniexWebsocketOrderbookModify struct {
					Type   string
					Rate   float64
					Amount float64
				}

				orderModify := PoloniexWebsocketOrderbookModify{}
				orderModify.Type = msgData["type"].(string)

				rateStr := msgData["rate"].(string)
				orderModify.Rate, _ = strconv.ParseFloat(rateStr, 64)

				amountStr := msgData["amount"].(string)
				orderModify.Amount, _ = strconv.ParseFloat(amountStr, 64)
			}
		case "orderBookRemove":
			{
				type PoloniexWebsocketOrderbookRemove struct {
					Type string
					Rate float64
				}

				orderRemoval := PoloniexWebsocketOrderbookRemove{}
				orderRemoval.Type = msgData["type"].(string)

				rateStr := msgData["rate"].(string)
				orderRemoval.Rate, _ = strconv.ParseFloat(rateStr, 64)
			}
		case "newTrade":
			{
				type PoloniexWebsocketNewTrade struct {
					Type    string
					TradeID int64
					Rate    float64
					Amount  float64
					Date    string
					Total   float64
				}

				trade := PoloniexWebsocketNewTrade{}
				trade.Type = msgData["type"].(string)

				tradeIDstr := msgData["tradeID"].(string)
				trade.TradeID, _ = strconv.ParseInt(tradeIDstr, 10, 64)

				rateStr := msgData["rate"].(string)
				trade.Rate, _ = strconv.ParseFloat(rateStr, 64)

				amountStr := msgData["amount"].(string)
				trade.Amount, _ = strconv.ParseFloat(amountStr, 64)

				totalStr := msgData["total"].(string)
				trade.Rate, _ = strconv.ParseFloat(totalStr, 64)

				trade.Date = msgData["date"].(string)
			}
		}
	}
}

// WebsocketClient creates a new websocket client
func (p *Poloniex) WebsocketClient() {
	for p.Enabled && p.Websocket {
		c, err := turnpike.NewWebsocketClient(turnpike.JSON, poloniexWebsocketAddress, nil)
		if err != nil {
			log.Printf("%s Unable to connect to Websocket. Error: %s\n", p.GetName(), err)
			continue
		}

		if p.Verbose {
			log.Printf("%s Connected to Websocket.\n", p.GetName())
		}

		_, err = c.JoinRealm(poloniexWebsocketRealm, nil)
		if err != nil {
			log.Printf("%s Unable to join realm. Error: %s\n", p.GetName(), err)
			continue
		}

		if p.Verbose {
			log.Printf("%s Joined Websocket realm.\n", p.GetName())
		}

		c.ReceiveDone = make(chan bool)

		if err := c.Subscribe(poloniexWebsocketTicker, OnTicker); err != nil {
			log.Printf("%s Error subscribing to ticker channel: %s\n", p.GetName(), err)
		}

		if err := c.Subscribe(poloniexWebsocketTrollbox, OnTrollbox); err != nil {
			log.Printf("%s Error subscribing to trollbox channel: %s\n", p.GetName(), err)
		}

		for x := range p.EnabledPairs {
			currency := p.EnabledPairs[x]
			if err := c.Subscribe(currency, OnDepthOrTrade); err != nil {
				log.Printf("%s Error subscribing to %s channel: %s\n", p.GetName(), currency, err)
			}
		}

		if p.Verbose {
			log.Printf("%s Subscribed to websocket channels.\n", p.GetName())
		}

		<-c.ReceiveDone
		log.Printf("%s Websocket client disconnected.\n", p.GetName())
	}
}
