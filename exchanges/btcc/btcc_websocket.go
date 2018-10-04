package btcc

import (
	"errors"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/socketio"
)

const (
	btccSocketioAddress = "wss://ws.btcc.com"

	msgTypeHeartBeat          = "Heartbeat"
	msgTypeGetActiveContracts = "GetActiveContractsResponse"
	msgTypeQuote              = "QuoteResponse"
	msgTypeLogin              = "LoginResponse"
	msgTypeAccountInfo        = "AccountInfo"
	msgTypeExecReport         = "ExecReport"
	msgTypePlaceOrder         = "PlaceOrderResponse"
	msgTypeCancelAllOrders    = "CancelAllOrdersResponse"
	msgTypeCancelOrder        = "CancelOrderResponse"
	msgTypeCancelReplaceOrder = "CancelReplaceOrderResponse"
	msgTypeGetAccountInfo     = "GetAccountInfoResponse"
	msgTypeRetrieveOrder      = "RetrieveOrderResponse"
	msgTypeGetTrades          = "GetTradesResponse"

	msgTypeAllTickers = "AllTickersResponse"
)

// BTCCSocket is a pointer to a IO socket
var (
	BTCCSocket *socketio.SocketIO
	mtx        sync.Mutex
)

// WsOutgoing defines outgoing JSON
type WsOutgoing struct {
	Action string `json:"action"`
	Symbol string `json:"symbol,omitempty"`
	Count  int    `json:"count,omitempty"`
	Len    int    `json:"len,omitempty"`
}

// WsConnect initiates a websocket client connection
func (b *BTCC) WsConnect() error {
	if !b.Websocket.IsEnabled() || !b.IsEnabled() {
		return errors.New(exchange.WebsocketNotEnabled)
	}

	b.Websocket.ShutdownC = make(chan struct{}, 1)

	var dialer websocket.Dialer
	var err error

	if b.Websocket.GetProxyAddress() != "" {
		prxy, err := url.Parse(b.Websocket.GetProxyAddress())
		if err != nil {
			return err
		}
		dialer.Proxy = http.ProxyURL(prxy)
	}

	b.Conn, _, err = dialer.Dial(b.Websocket.GetWebsocketURL(), http.Header{})
	if err != nil {
		return err
	}

	err = b.WsSubscribeAllTickers()
	if err != nil {
		return err
	}

	var currencyUpdated bool
	for !currencyUpdated {
		quickCapture := quickcheck{}
		_, resp, err := b.Conn.ReadMessage()
		if err != nil {
			return err
		}

		err = common.JSONDecode(resp, &quickCapture)
		if err != nil {
			log.Println(string(resp))
			return err
		}

		switch quickCapture.MsgType {
		case msgTypeAllTickers:
			var tickers WsAllTickerData
			err := common.JSONDecode(resp, &tickers)
			if err != nil {
				log.Println(string(resp))
				log.Fatal(err)
			}

			var availableTickers []string

			for _, tickerData := range tickers {
				availableTickers = append(availableTickers, tickerData.Symbol)
			}

			err = b.UpdateCurrencies(availableTickers, false, true)
			if err != nil {
				log.Fatalf("%s failed to update available currencies. %s\n",
					b.Name,
					err)
			}

			err = b.WsUnSubscribeAllTickers()
			if err != nil {
				log.Fatal(err)
			}
			currencyUpdated = true

		default:
			continue
		}
	}

	err = b.WsSubscribeToOrderbook()
	if err != nil {
		return err
	}

	err = b.WsSubcribeToTicker()
	if err != nil {
		return err
	}

	err = b.WsSubcribeToTrades()
	if err != nil {
		return err
	}

	c := make(chan response, 1)
	go b.WsReadData(c)
	go b.WsHandleData(c)

	return nil
}

type response struct {
	MessageType int
	Resp        []byte
}

// WsReadData reads data from the websocket connection
func (b *BTCC) WsReadData(comm chan response) {
	log.Println("READING DATA")
	b.Websocket.Wg.Add(1)
	defer b.Websocket.Wg.Done()

	for {
		select {
		case <-b.Websocket.ShutdownC:
		default:

			mtx.Lock()
			log.Println("Attempting to Read")
			_, resp, err := b.Conn.ReadMessage()
			log.Println("Message Read")
			mtx.Unlock()
			if err != nil {
				b.Websocket.DataHandler <- err
			}
			log.Println("pushed to channel")
			comm <- response{
				// MessageType: messageType,
				Resp: resp,
			}
		}
	}
}

type quickcheck struct {
	MsgType string `json:"MsgType"`
	CRID    string `json:"CRID"`
	RC      int    `json:"RC"`
	Reason  string `json:"Reason"`
}

// WsHandleData handles read data
func (b *BTCC) WsHandleData(comm chan response) {
	log.Println("HANDLING DATA")
	b.Websocket.Wg.Add(1)
	defer b.Websocket.Wg.Done()

	for {
		select {
		case <-b.Websocket.ShutdownC:
			return
		case yay := <-comm:
			log.Println("something came in")
			var Result quickcheck
			err := common.JSONDecode(yay.Resp, &Result)
			if err != nil {
				log.Fatal(err)
			}

			switch Result.MsgType {
			case msgTypeHeartBeat:
				log.Println("HeartBeat")
				log.Fatal(string(yay.Resp))

			case msgTypeGetActiveContracts:
				log.Println("Active Contracts")
				log.Fatal(string(yay.Resp))

			case msgTypeQuote:
				log.Println("Quotes")
				log.Fatal(string(yay.Resp))

			case msgTypeLogin:
				log.Println("Login")
				log.Fatal(string(yay.Resp))

			case msgTypeAccountInfo:
				log.Println("Account info")
				log.Fatal(string(yay.Resp))

			case msgTypeExecReport:
				log.Println("Exec Report")
				log.Fatal(string(yay.Resp))

			case msgTypePlaceOrder:
				log.Println("Place order")
				log.Fatal(string(yay.Resp))

			case msgTypeCancelAllOrders:
				log.Println("Cancel All orders")
				log.Fatal(string(yay.Resp))

			case msgTypeCancelOrder:
				log.Println("Cancel order")
				log.Fatal(string(yay.Resp))

			case msgTypeCancelReplaceOrder:
				log.Println("Replace order")
				log.Fatal(string(yay.Resp))

			case msgTypeGetAccountInfo:
				log.Println("Account info")
				log.Fatal(string(yay.Resp))

			case msgTypeRetrieveOrder:
				log.Println("Retrieve order")
				log.Fatal(string(yay.Resp))

			case msgTypeGetTrades:
				log.Println("Get trades")
				log.Fatal(string(yay.Resp))

				// for _, data := range tickers {
				// 	if common.StringDataCompare(b.EnabledPairs, data.Symbol) {
				// 		b.Websocket.DataHandler <- exchange.TickerData{
				// 			Timestamp:         time.Unix(0, data.Timestamp),
				// 			Pair:              pair.NewCurrencyPairFromString(data.Symbol),
				// 			OpenPrice:         data.Open,
				// 			HighPrice:         data.High,
				// 			BestAskPrice:      data.AskPrice,
				// 			BestBidPrice:      data.BidPrice,
				// 			TotalTradedVolume: data.Volume,
				// 		}
				// 	}
				// }

			case msgTypeAllTickers:

			default:
				log.Println("edgecase")
				if common.StringContains(Result.MsgType, "OrderBook") {
					log.Println("ORDER BOOK!!!!!")
					log.Fatal(string(yay.Resp))
				}
				log.Fatal("edge case:", Result.MsgType)
			}
		}
	}
}

// WsShutdown closes websocket connection and routines
func (b *BTCC) WsShutdown() error {
	log.Println("Shutdown captured")
	timer := time.NewTimer(5 * time.Second)
	c := make(chan struct{}, 1)

	go func(c chan struct{}) {
		close(b.Websocket.ShutdownC)
		b.Websocket.Wg.Wait()
		c <- struct{}{}
	}(c)

	select {
	case <-timer.C:
		return errors.New("routines did not shut down")
	case <-c:
		return b.Conn.Close()
	}
}

// WsGetActiveContracts return
func (b *BTCC) WsGetActiveContracts() (interface{}, error) {
	return nil, nil
}

// WsGetTrades returns trade data
func (b *BTCC) WsGetTrades() (interface{}, error) {
	return nil, nil
}

// WsSubscribeAllTickers subscribes to a ticker channel
func (b *BTCC) WsSubscribeAllTickers() error {
	mtx.Lock()
	defer mtx.Unlock()
	return b.Conn.WriteJSON(WsOutgoing{
		Action: "SubscribeAllTickers",
	})
}

// WsUnSubscribeAllTickers unsubscribes from a ticker channel
func (b *BTCC) WsUnSubscribeAllTickers() error {
	mtx.Lock()
	defer mtx.Unlock()
	return b.Conn.WriteJSON(WsOutgoing{
		Action: "UnSubscribeAllTickers",
	})
}

// WsSubscribeToOrderbook subscribes to an orderbook channel
func (b *BTCC) WsSubscribeToOrderbook() error {
	mtx.Lock()
	defer mtx.Unlock()

	for _, pair := range b.EnabledPairs {
		err := b.Conn.WriteJSON(WsOutgoing{
			Action: "SubOrderBook",
			Symbol: pair,
			Len:    100})
		if err != nil {
			return err
		}
	}
	return nil
}

// WsSubcribeToTicker subscribes to a ticker channel
func (b *BTCC) WsSubcribeToTicker() error {
	mtx.Lock()
	defer mtx.Unlock()

	for _, pair := range b.EnabledPairs {
		err := b.Conn.WriteJSON(WsOutgoing{
			Action: "Subscribe",
			Symbol: pair})
		if err != nil {
			return err
		}
	}
	return nil
}

// WsSubcribeToTrades subscribes to a trade channel
func (b *BTCC) WsSubcribeToTrades() error {
	mtx.Lock()
	defer mtx.Unlock()

	for _, pair := range b.EnabledPairs {
		err := b.Conn.WriteJSON(WsOutgoing{
			Action: "GetTrades",
			Symbol: pair,
			Count:  100,
		})
		if err != nil {
			return err
		}
	}
	return nil
}
