package huobi

import (
	"log"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/stats"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

func (h *HUOBI) Start() {
	go h.Run()
}

func (h *HUOBI) Run() {
	if h.Verbose {
		log.Printf("%s Websocket: %s (url: %s).\n", h.GetName(), common.IsEnabled(h.Websocket), HUOBI_SOCKETIO_ADDRESS)
		log.Printf("%s polling delay: %ds.\n", h.GetName(), h.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", h.GetName(), len(h.EnabledPairs), h.EnabledPairs)
	}

	if h.Websocket {
		go h.WebsocketClient()
	}

	for h.Enabled {
		for _, x := range h.EnabledPairs {
			curr := common.StringToLower(x[0:3])
			go func() {
				ticker, err := h.GetTickerPrice(curr)
				if err != nil {
					log.Println(err)
					return
				}
				HuobiLastUSD, _ := currency.ConvertCurrency(ticker.Last, "CNY", "USD")
				HuobiHighUSD, _ := currency.ConvertCurrency(ticker.High, "CNY", "USD")
				HuobiLowUSD, _ := currency.ConvertCurrency(ticker.Low, "CNY", "USD")
				log.Printf("Huobi %s: Last %f (%f) High %f (%f) Low %f (%f) Volume %f\n", curr, HuobiLastUSD, ticker.Last, HuobiHighUSD, ticker.High, HuobiLowUSD, ticker.Low, ticker.Volume)
				stats.AddExchangeInfo(h.GetName(), common.StringToUpper(curr[0:3]), common.StringToUpper(curr[3:]), ticker.Last, ticker.Volume)
				stats.AddExchangeInfo(h.GetName(), common.StringToUpper(curr[0:3]), "USD", HuobiLastUSD, ticker.Volume)
			}()
		}
		time.Sleep(time.Second * h.RESTPollingDelay)
	}
}

func (h *HUOBI) GetTickerPrice(currency string) (ticker.TickerPrice, error) {
	tickerNew, err := ticker.GetTicker(h.GetName(), common.StringToUpper(currency[0:3]), common.StringToUpper(currency[3:]))
	if err == nil {
		return tickerNew, nil
	}

	var tickerPrice ticker.TickerPrice
	tick, err := h.GetTicker(currency)
	if err != nil {
		return tickerPrice, err
	}
	tickerPrice.Ask = tick.Sell
	tickerPrice.Bid = tick.Buy
	tickerPrice.FirstCurrency = common.StringToUpper(currency[0:3])
	tickerPrice.SecondCurrency = common.StringToUpper(currency[3:])
	tickerPrice.Low = tick.Low
	tickerPrice.Last = tick.Last
	tickerPrice.Volume = tick.Vol
	tickerPrice.High = tick.High
	ticker.ProcessTicker(h.GetName(), tickerPrice.FirstCurrency, tickerPrice.SecondCurrency, tickerPrice)
	return tickerPrice, nil
}

func (h *HUOBI) GetOrderbookEx(currency string) (orderbook.OrderbookBase, error) {
	ob, err := orderbook.GetOrderbook(h.GetName(), common.StringToUpper(currency[0:3]), common.StringToUpper(currency[3:]))
	if err == nil {
		return ob, nil
	}

	var orderBook orderbook.OrderbookBase
	orderbookNew, err := h.GetOrderBook(currency)
	if err != nil {
		return orderBook, err
	}

	for x, _ := range orderbookNew.Bids {
		data := orderbookNew.Bids[x]
		orderBook.Bids = append(orderBook.Bids, orderbook.OrderbookItem{Amount: data[1], Price: data[0]})
	}

	for x, _ := range orderbookNew.Asks {
		data := orderbookNew.Asks[x]
		orderBook.Asks = append(orderBook.Asks, orderbook.OrderbookItem{Amount: data[1], Price: data[0]})
	}
	orderBook.FirstCurrency = common.StringToUpper(currency[0:3])
	orderBook.SecondCurrency = common.StringToUpper(currency[3:])
	orderbook.ProcessOrderbook(h.GetName(), orderBook.FirstCurrency, orderBook.SecondCurrency, orderBook)
	return orderBook, nil
}

//TODO: retrieve HUOBI balance info
//GetExchangeAccountInfo : Retrieves balances for all enabled currencies for the HUOBI exchange
func (e *HUOBI) GetExchangeAccountInfo() (exchange.ExchangeAccountInfo, error) {
	var response exchange.ExchangeAccountInfo
	response.ExchangeName = e.GetName()
	return response, nil
}
