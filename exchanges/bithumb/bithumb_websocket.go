package bithumb

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

const wsEndpoint = "wss://pubwss.bithumb.com/pub/ws"

var (
	wsDefaultTickTypes = []string{"30M"} // alternatives "1H", "12H", "24H", "MID"
	tickerTimeLayout   = "20060102150405"
	tradeTimeLayout    = "2006-01-02 15:04:05.000000"
)

var location *time.Location

func init() {
	var err error
	location, err = time.LoadLocation("Asia/Seoul")
	if err != nil {
		panic(err)
	}
}

// WsConnect initiates a websocket connection
func (b *Bithumb) WsConnect() error {
	if !b.Websocket.IsEnabled() || !b.IsEnabled() {
		return errors.New(stream.WebsocketNotEnabled)
	}

	var dialer websocket.Dialer
	dialer.HandshakeTimeout = b.Config.HTTPTimeout
	dialer.Proxy = http.ProxyFromEnvironment

	err := b.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return fmt.Errorf("%v - Unable to connect to Websocket. Error: %w",
			b.Name,
			err)
	}
	go b.wsReadData()
	b.setupOrderbookManager()
	return nil
}

// wsReadData receives and passes on websocket messages for processing
func (b *Bithumb) wsReadData() {
	b.Websocket.Wg.Add(1)
	defer b.Websocket.Wg.Done()

	for {
		resp := b.Websocket.Conn.ReadMessage()
		if resp.Raw == nil {
			return
		}
		err := b.wsHandleData(resp.Raw)
		if err != nil {
			b.Websocket.DataHandler <- err
		}
	}
}

func (b *Bithumb) wsHandleData(respRaw []byte) error {
	var resp WsReponse
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}

	if len(resp.Status) > 0 {
		if resp.Status == "0000" {
			return nil
		}
		return errors.New(resp.ResponseMessage)
	}

	switch resp.Type {
	case "ticker":
		var tick WsTicker
		err = json.Unmarshal(resp.Content, &tick)
		if err != nil {
			return err
		}
		var lu time.Time
		lu, err = time.ParseInLocation(tickerTimeLayout,
			tick.Date+tick.Time,
			location)
		if err != nil {
			return err
		}
		b.Websocket.DataHandler <- &ticker.Price{
			ExchangeName: b.Name,
			AssetType:    asset.Spot,
			Last:         tick.PreviousClosePrice,
			Pair:         tick.Symbol,
			Open:         tick.OpenPrice,
			Close:        tick.ClosePrice,
			Low:          tick.LowPrice,
			High:         tick.HighPrice,
			QuoteVolume:  tick.Value,
			Volume:       tick.Volume,
			LastUpdated:  lu,
		}
	case "transaction":
		if !b.IsSaveTradeDataEnabled() {
			return nil
		}

		var trades WsTransactions
		err = json.Unmarshal(resp.Content, &trades)
		if err != nil {
			return err
		}

		var lu time.Time
		for x := range trades.List {
			lu, err = time.ParseInLocation(tradeTimeLayout,
				trades.List[x].ContractTime,
				location)
			if err != nil {
				return err
			}

			err = b.AddTradesToBuffer(trade.Data{
				Exchange:     b.Name,
				AssetType:    asset.Spot,
				CurrencyPair: trades.List[x].Symbol,
				Timestamp:    lu,
				Price:        trades.List[x].ContractPrice,
				Amount:       trades.List[x].ContractAmount,
			})
			if err != nil {
				return err
			}
		}
	case "orderbookdepth":
		var orderbooks WsOrderbooks
		err = json.Unmarshal(resp.Content, &orderbooks)
		if err != nil {
			return err
		}
		init, err := b.UpdateLocalBuffer(&orderbooks)
		if err != nil && !init {
			return fmt.Errorf("%v - UpdateLocalCache error: %s", b.Name, err)
		}
		return nil
	default:
		return fmt.Errorf("unhandled response type %s", resp.Type)
	}

	return nil
}

// GenerateSubscriptions generates the default subscription set
func (b *Bithumb) GenerateSubscriptions() ([]stream.ChannelSubscription, error) {
	var channels = []string{"ticker", "transaction", "orderbookdepth"}
	var subscriptions []stream.ChannelSubscription
	pairs, err := b.GetEnabledPairs(asset.Spot)
	if err != nil {
		return nil, err
	}

	for x := range pairs {
		for y := range channels {
			subscriptions = append(subscriptions, stream.ChannelSubscription{
				Channel:  channels[y],
				Currency: pairs[x].Format("_", true),
				Asset:    asset.Spot,
			})
		}
	}
	return subscriptions, nil
}

// Subscribe subscribes to a set of channels
func (b *Bithumb) Subscribe(channelsToSubscribe []stream.ChannelSubscription) error {
	subs := make(map[string]*WsSubscribe)
	for i := range channelsToSubscribe {
		s, ok := subs[channelsToSubscribe[i].Channel]
		if !ok {
			s = &WsSubscribe{
				Type: channelsToSubscribe[i].Channel,
			}
			subs[channelsToSubscribe[i].Channel] = s
		}
		s.Symbols = append(s.Symbols, channelsToSubscribe[i].Currency)
	}

	tSub, ok := subs["ticker"]
	if ok {
		tSub.TickTypes = wsDefaultTickTypes
	}

	for _, s := range subs {
		err := b.Websocket.Conn.SendJSONMessage(s)
		if err != nil {
			return err
		}
		time.Sleep(time.Second) // Undocumented rate limiting.
	}
	b.Websocket.AddSuccessfulSubscriptions(channelsToSubscribe...)
	return nil
}
