package btcc

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/assets"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	log "github.com/thrasher-/gocryptotrader/logger"
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

var (
	mtx sync.Mutex
)

// WsConnect initiates a websocket client connection
func (b *BTCC) WsConnect() error {
	if !b.Websocket.IsEnabled() || !b.IsEnabled() {
		return errors.New(exchange.WebsocketNotEnabled)
	}

	var dialer websocket.Dialer
	var err error

	if b.Websocket.GetProxyAddress() != "" {
		var proxy *url.URL
		proxy, err = url.Parse(b.Websocket.GetProxyAddress())
		if err != nil {
			return err
		}
		dialer.Proxy = http.ProxyURL(proxy)
	}

	b.Conn, _, err = dialer.Dial(b.Websocket.GetWebsocketURL(), http.Header{})
	if err != nil {
		return err
	}

	err = b.WsUpdateCurrencyPairs()
	if err != nil {
		return err
	}

	go b.WsHandleData()

	err = b.WsSubscribeToOrderbook()
	if err != nil {
		return err
	}

	err = b.WsSubcribeToTicker()
	if err != nil {
		return err
	}

	return b.WsSubcribeToTrades()
}

// WsReadData reads data from the websocket connection
func (b *BTCC) WsReadData() (exchange.WebsocketResponse, error) {
	mtx.Lock()
	_, resp, err := b.Conn.ReadMessage()
	mtx.Unlock()
	if err != nil {
		return exchange.WebsocketResponse{}, err
	}

	b.Websocket.TrafficAlert <- struct{}{}

	return exchange.WebsocketResponse{
		Raw: resp,
	}, nil
}

// WsHandleData handles read data
func (b *BTCC) WsHandleData() {
	b.Websocket.Wg.Add(1)

	defer func() {
		err := b.Conn.Close()
		if err != nil {
			b.Websocket.DataHandler <- fmt.Errorf("btcc_websocket.go - Unable to close Websocket connection. Error: %s",
				err)
		}
		b.Websocket.Wg.Done()
	}()

	for {
		select {
		case <-b.Websocket.ShutdownC:
			return

		default:
			resp, err := b.WsReadData()
			if err != nil {
				b.Websocket.DataHandler <- err
				return
			}

			var Result WsResponseMain
			err = common.JSONDecode(resp.Raw, &Result)
			if err != nil {
				b.Websocket.DataHandler <- err
				continue
			}

			switch Result.MsgType {
			case msgTypeHeartBeat:

			case msgTypeGetActiveContracts:
				log.Debugf("Active Contracts: %s", resp.Raw)

			case msgTypeQuote:
				log.Debugf("Quotes: %s", resp.Raw)

			case msgTypeLogin:
				log.Debugf("Login: %s", resp.Raw)

			case msgTypeAccountInfo:
				log.Debugf("Account info: %s", resp.Raw)

			case msgTypeExecReport:
				log.Debugf("Exec Report: %s", resp.Raw)

			case msgTypePlaceOrder:
				log.Debugf("Place order: %s", resp.Raw)

			case msgTypeCancelAllOrders:
				log.Debugf("Cancel All orders: %s", resp.Raw)

			case msgTypeCancelOrder:
				log.Debugf("Cancel order: %s", resp.Raw)

			case msgTypeCancelReplaceOrder:
				log.Debugf("Replace order: %s", resp.Raw)

			case msgTypeGetAccountInfo:
				log.Debugf("Account info: %s", resp.Raw)

			case msgTypeRetrieveOrder:
				log.Debugf("Retrieve order: %s", resp.Raw)

			case msgTypeGetTrades:
				var trades WsTrades

				err = common.JSONDecode(resp.Raw, &trades)
				if err != nil {
					b.Websocket.DataHandler <- err
					continue
				}

			case "OrderBook":
				// NOTE: This seems to be a websocket update not reflected in
				// current API docs, this comes in conjunction with the other
				// orderbook feeds
				var orderbook WsOrderbookSnapshot

				err = common.JSONDecode(resp.Raw, &orderbook)
				if err != nil {
					b.Websocket.DataHandler <- err
					continue
				}

				switch orderbook.Type {
				case "F":
					err = b.WsProcessOrderbookSnapshot(&orderbook)
					if err != nil {
						b.Websocket.DataHandler <- err
					}

				case "I":
					err = b.WsProcessOrderbookUpdate(&orderbook)
					if err != nil {
						b.Websocket.DataHandler <- err
					}
				}

			case "SubOrderBookResponse":

			case "Ticker":
				var ticker WsTicker

				err = common.JSONDecode(resp.Raw, &ticker)
				if err != nil {
					b.Websocket.DataHandler <- err
					continue
				}

				tick := exchange.TickerData{}
				tick.AssetType = assets.AssetTypeSpot
				tick.ClosePrice = ticker.PrevCls
				tick.Exchange = b.GetName()
				tick.HighPrice = ticker.High
				tick.LowPrice = ticker.Low
				tick.OpenPrice = ticker.Open
				tick.Pair = currency.NewPairFromString(ticker.Symbol)
				tick.Quantity = ticker.Volume
				timestamp := time.Unix(ticker.Timestamp, 0)
				tick.Timestamp = timestamp

				b.Websocket.DataHandler <- tick

			default:

				if common.StringContains(Result.MsgType, "OrderBook") {
					var oldOrderbookType WsOrderbookSnapshotOld
					err = common.JSONDecode(resp.Raw, &oldOrderbookType)
					if err != nil {
						b.Websocket.DataHandler <- err
						continue
					}

					symbol := common.SplitStrings(Result.MsgType, ".")
					err = b.WsProcessOldOrderbookSnapshot(oldOrderbookType, symbol[1])
					if err != nil {
						b.Websocket.DataHandler <- err
						continue
					}
					continue
				}
			}
		}
	}
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

// WsUpdateCurrencyPairs updates currency pairs from the websocket connection
func (b *BTCC) WsUpdateCurrencyPairs() error {
	err := b.WsSubscribeAllTickers()
	if err != nil {
		return err
	}

	var currencyResponse WsResponseMain
	for {
		_, resp, err := b.Conn.ReadMessage()
		if err != nil {
			return err
		}

		b.Websocket.TrafficAlert <- struct{}{}

		err = common.JSONDecode(resp, &currencyResponse)
		if err != nil {
			return err
		}

		switch currencyResponse.MsgType {
		case msgTypeAllTickers:
			var tickers WsAllTickerData
			err := common.JSONDecode(currencyResponse.Data, &tickers)
			if err != nil {
				return err
			}

			var availableTickers currency.Pairs
			for i := range tickers {
				availableTickers = append(availableTickers,
					currency.NewPairFromString(tickers[i].Symbol))
			}

			err = b.UpdatePairs(availableTickers, assets.AssetTypeSpot, false, true)
			if err != nil {
				return fmt.Errorf("%s failed to update available currencies. %s",
					b.Name,
					err)
			}

			return b.WsUnSubscribeAllTickers()

		case "Heartbeat":

		default:
			return fmt.Errorf("btcc_websocket.go error - Updating currency pairs resp incorrect: %s",
				string(resp))
		}
	}
}

// WsSubscribeToOrderbook subscribes to an orderbook channel
func (b *BTCC) WsSubscribeToOrderbook() error {
	mtx.Lock()
	defer mtx.Unlock()

	for _, pair := range b.GetEnabledPairs(assets.AssetTypeSpot) {
		formattedPair := b.FormatExchangeCurrency(pair, assets.AssetTypeSpot)
		err := b.Conn.WriteJSON(WsOutgoing{
			Action: "SubOrderBook",
			Symbol: formattedPair.String(),
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

	for _, pair := range b.GetEnabledPairs(assets.AssetTypeSpot) {
		formattedPair := b.FormatExchangeCurrency(pair, assets.AssetTypeSpot)
		err := b.Conn.WriteJSON(WsOutgoing{
			Action: "Subscribe",
			Symbol: formattedPair.String(),
		})
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

	for _, pair := range b.GetEnabledPairs(assets.AssetTypeSpot) {
		formattedPair := b.FormatExchangeCurrency(pair, assets.AssetTypeSpot)
		err := b.Conn.WriteJSON(WsOutgoing{
			Action: "GetTrades",
			Symbol: formattedPair.String(),
			Count:  100,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// WsProcessOrderbookSnapshot processes a new orderbook snapshot
func (b *BTCC) WsProcessOrderbookSnapshot(ob *WsOrderbookSnapshot) error {
	var asks, bids []orderbook.Item
	for _, data := range ob.List {
		var newSize float64
		switch result := data.Size.(type) {
		case float64:
			newSize = result
		case string:
			var err error
			newSize, err = strconv.ParseFloat(result, 64)
			if err != nil {
				return err
			}
		}

		if data.Side == "1" {
			asks = append(asks, orderbook.Item{Price: data.Price, Amount: newSize})
			continue
		}

		bids = append(bids, orderbook.Item{Price: data.Price, Amount: newSize})
	}

	var newOrderBook orderbook.Base

	newOrderBook.Asks = asks
	newOrderBook.AssetType = assets.AssetTypeSpot
	newOrderBook.Bids = bids
	newOrderBook.Pair = currency.NewPairFromString(ob.Symbol)

	err := b.Websocket.Orderbook.LoadSnapshot(&newOrderBook, b.GetName(), false)
	if err != nil {
		return err
	}

	b.Websocket.DataHandler <- exchange.WebsocketOrderbookUpdate{
		Exchange: b.GetName(),
		Asset:    assets.AssetTypeSpot,
		Pair:     currency.NewPairFromString(ob.Symbol),
	}

	return nil
}

// WsProcessOrderbookUpdate processes an orderbook update
func (b *BTCC) WsProcessOrderbookUpdate(ob *WsOrderbookSnapshot) error {
	var asks, bids []orderbook.Item
	for _, data := range ob.List {
		var newSize float64
		switch d := data.Size.(type) {
		case float64:
			newSize = d
		case string:
			var err error
			newSize, err = strconv.ParseFloat(d, 64)
			if err != nil {
				return err
			}
		}

		if data.Side == "1" {
			if newSize < 0 {
				asks = append(asks, orderbook.Item{Price: data.Price, Amount: 0})
				continue
			}
			asks = append(asks, orderbook.Item{Price: data.Price, Amount: newSize})
			continue
		}

		if newSize < 0 {
			bids = append(bids, orderbook.Item{Price: data.Price, Amount: 0})
			continue
		}

		bids = append(bids, orderbook.Item{Price: data.Price, Amount: newSize})
	}

	p := currency.NewPairFromString(ob.Symbol)

	err := b.Websocket.Orderbook.Update(bids, asks, p, time.Now(), b.GetName(), assets.AssetTypeSpot)
	if err != nil {
		return err
	}

	b.Websocket.DataHandler <- exchange.WebsocketOrderbookUpdate{
		Exchange: b.GetName(),
		Asset:    assets.AssetTypeSpot,
		Pair:     currency.NewPairFromString(ob.Symbol),
	}

	return nil
}

// WsProcessOldOrderbookSnapshot processes an old orderbook snapshot
func (b *BTCC) WsProcessOldOrderbookSnapshot(ob WsOrderbookSnapshotOld, symbol string) error {
	var asks, bids []orderbook.Item

	askData := ob.Data["Asks"]
	bidData := ob.Data["Bids"]

	for _, ask := range askData {
		data := ask.([]interface{})
		var price, amount float64

		switch d := data[0].(type) {
		case string:
			var err error
			price, err = strconv.ParseFloat(d, 64)
			if err != nil {
				return err
			}
		case float64:
			price = d
		}

		switch d := data[0].(type) {
		case string:
			var err error
			amount, err = strconv.ParseFloat(d, 64)
			if err != nil {
				return err
			}
		case float64:
			amount = d
		}

		asks = append(asks, orderbook.Item{
			Price:  price,
			Amount: amount,
		})
	}

	for _, bid := range bidData {
		data := bid.([]interface{})
		var price, amount float64

		switch d := data[1].(type) {
		case string:
			var err error
			price, err = strconv.ParseFloat(d, 64)
			if err != nil {
				return err
			}
		case float64:
			price = d
		}

		switch d := data[1].(type) {
		case string:
			var err error
			amount, err = strconv.ParseFloat(d, 64)
			if err != nil {
				return err
			}
		case float64:
			amount = d
		}

		bids = append(bids, orderbook.Item{
			Price:  price,
			Amount: amount,
		})
	}

	p := currency.NewPairFromString(symbol)
	err := b.Websocket.Orderbook.Update(bids, asks, p, time.Now(), b.GetName(), assets.AssetTypeSpot)
	if err != nil {
		return err
	}

	b.Websocket.DataHandler <- exchange.WebsocketOrderbookUpdate{
		Exchange: b.GetName(),
		Pair:     p,
		Asset:    assets.AssetTypeSpot,
	}

	return nil
}
