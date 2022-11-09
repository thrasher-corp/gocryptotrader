package gateio

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

const (
	optionsWebsocketURL        = "wss://op-ws.gateio.live/v4/ws"
	optionsWebsocketTestnetURL = "wss://op-ws-testnet.gateio.live/v4/ws"

	// channels
	optionsPingChannel                   = "options.ping"
	optionsContractTickersChannel        = "options.contract_tickers"
	optionsUnderlyingTickersChannel      = "options.ul_tickers"
	optionsTradesChannel                 = "options.trades"
	optionsUnderlyingTradesChannel       = "options.ul_trades"
	optionsUnderlyingPriceChannel        = "options.ul_price"
	optionsMarkPriceChannel              = "options.mark_price"
	optionsSettlementChannel             = "options.settlements"
	optionsContractsChannel              = "options.contracts"
	optionsContractCandlesticksChannel   = "options.contract_candlesticks"
	optionsUnderlyingCandlesticksChannel = "options.ul_candlesticks"
	optionsOrderbookChannel              = "options.order_book"
	optionsOrderbookTickerChannel        = "options.book_ticker"
	optionsOrderbookUpdateChannel        = "options.order_book_update"
	optionsOrdersChannel                 = "options.orders"
	optionsUserTradesChannel             = "options.usertrades"
	optionsLiquidatesChannel             = "options.liquidates"
	optionsUserSettlementChannel         = "options.user_settlements"
	optionsPositionCloseChannel          = "options.position_closes"
	optionsBalancesChannel               = "options.balances"
	optionsPositionsChannel              = "options.positions"
)

var defaultOptionsSubscriptions = []string{
	optionsContractTickersChannel,
	optionsUnderlyingTickersChannel,
	optionsTradesChannel,
	optionsUnderlyingTradesChannel,
	optionsContractCandlesticksChannel,
	optionsUnderlyingCandlesticksChannel,
	optionsOrderbookChannel,
	optionsOrderbookUpdateChannel,
}

// WsOptionsConnect initiates a websocket connection to options websocket endpoints.
func (g *Gateio) WsOptionsConnect() error {
	if !g.Websocket.IsEnabled() || !g.IsEnabled() {
		return errors.New(stream.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
	err := g.Websocket.SetWebsocketURL(optionsWebsocketURL, false, true)
	if err != nil {
		return err
	}
	err = g.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	pingMessage, err := json.Marshal(WsInput{
		ID:      g.Websocket.Conn.GenerateMessageID(false),
		Time:    time.Now().Unix(),
		Channel: optionsPingChannel,
	})
	if err != nil {
		return err
	}
	g.Websocket.Wg.Add(1)
	go g.wsReadConnData()
	go g.WsChannelsMultiplexer.Run()
	g.Websocket.Conn.SetupPingHandler(stream.PingHandler{
		Websocket:   true,
		Delay:       time.Second * 5,
		MessageType: websocket.PingMessage,
		Message:     pingMessage,
	})
	return nil
}

// GenerateOptionsDefaultSubscriptions generates list of channel subscriptions for options asset type.
func (g *Gateio) GenerateOptionsDefaultSubscriptions() ([]stream.ChannelSubscription, error) {
	channelsToSubscribe := defaultOptionsSubscriptions
	if g.Websocket.CanUseAuthenticatedEndpoints() {
		channelsToSubscribe = append(channelsToSubscribe,
			optionsUserTradesChannel,
			optionsBalancesChannel,
		)
	}
	var subscriptions []stream.ChannelSubscription
	pairs, err := g.GetEnabledPairs(asset.Options)
	if err != nil {
		return nil, err
	}
	for i := range channelsToSubscribe {
		for j := range pairs {
			params := make(map[string]interface{})
			switch channelsToSubscribe[i] {
			case optionsOrderbookChannel:
				params["accuracy"] = "0"
				params["level"] = "20"
			case optionsContractCandlesticksChannel, optionsUnderlyingCandlesticksChannel:
				params["interval"] = kline.FiveMin
			case optionsOrderbookUpdateChannel:
				params["interval"] = kline.ThousandMilliseconds
				params["level"] = 20
			}
			fpair, err := g.FormatExchangeCurrency(pairs[j], asset.Options)
			if err != nil {
				return nil, err
			}
			subscriptions = append(subscriptions, stream.ChannelSubscription{
				Channel:  channelsToSubscribe[i],
				Currency: fpair.Upper(),
				Params:   params,
			})
		}
	}
	return subscriptions, nil
}

func (g *Gateio) processOptionsContractTickers(data []byte) error {
	var response WsResponse
	tickerData := OptionsTicker{}
	response.Result = &tickerData
	err := json.Unmarshal(data, &response)
	if err != nil {
		return err
	}
	currencyPair, err := currency.NewPairFromString(tickerData.Name)
	if err != nil {
		return err
	}
	g.Websocket.DataHandler <- &ticker.Price{
		Pair:         currencyPair,
		Last:         tickerData.LastPrice,
		Bid:          tickerData.Bid1Price,
		Ask:          tickerData.Ask1Price,
		AskSize:      tickerData.Ask1Size,
		BidSize:      tickerData.Bid1Size,
		ExchangeName: g.Name,
		AssetType:    asset.Options,
	}
	return nil
}

func (g *Gateio) processOptionsUnderlyingTicker(data []byte) error {
	var response WsResponse
	response.Result = &WsOptionUnderlyingTicker{}
	err := json.Unmarshal(data, &response)
	if err != nil {
		return err
	}
	g.Websocket.DataHandler <- &response
	return nil
}

func (g *Gateio) processOptionsTradesPushData(data []byte) error {
	saveTradeData := g.IsSaveTradeDataEnabled()
	if !saveTradeData &&
		!g.IsTradeFeedEnabled() {
		return nil
	}
	resp := struct {
		Time    int64             `json:"time"`
		Channel string            `json:"channel"`
		Event   string            `json:"event"`
		Result  []WsOptionsTrades `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	trades := make([]trade.Data, len(resp.Result))
	for x := range resp.Result {
		currencyPair, err := currency.NewPairFromString(resp.Result[x].Contract)
		if err != nil {
			return err
		}
		trades[x] = trade.Data{
			Timestamp:    time.UnixMilli(resp.Result[x].CreateTimeMs),
			CurrencyPair: currencyPair,
			AssetType:    asset.Options,
			Exchange:     g.Name,
			Price:        resp.Result[x].Price,
			Amount:       resp.Result[x].Size,
			TID:          strconv.FormatInt(resp.Result[x].ID, 10),
		}
	}
	return g.Websocket.Trade.Update(saveTradeData, trades...)
}

func (g *Gateio) processOptionsUnderlyingPricePushData(data []byte) error {
	var response WsResponse
	priceD := WsOptionsUnderlyingPrice{}
	response.Result = &priceD
	err := json.Unmarshal(data, &response)
	if err != nil {
		return err
	}
	g.Websocket.DataHandler <- &response
	return nil
}

func (g *Gateio) processOptionsMarkPrice(data []byte) error {
	var response WsResponse
	markPrice := WsOptionsMarkPrice{}
	response.Result = &markPrice
	err := json.Unmarshal(data, &response)
	if err != nil {
		return err
	}
	g.Websocket.DataHandler <- &response
	return nil
}

func (g *Gateio) processOptionsSettlementPushData(data []byte) error {
	var response WsResponse
	settlementData := WsOptionsSettlement{}
	response.Result = &settlementData
	err := json.Unmarshal(data, &response)
	if err != nil {
		return err
	}
	g.Websocket.DataHandler <- &response
	return nil
}

func (g *Gateio) processOptionsContractPushData(data []byte) error {
	var response WsResponse
	contractData := WsOptionsContract{}
	response.Result = &contractData
	err := json.Unmarshal(data, &response)
	if err != nil {
		return err
	}
	g.Websocket.DataHandler <- &response
	return nil
}

func (g *Gateio) processOptionsCandlestickPushData(data []byte) error {
	resp := struct {
		Time    int64                          `json:"time"`
		Channel string                         `json:"channel"`
		Event   string                         `json:"event"`
		Result  []WsOptionsContractCandlestick `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	for x := range resp.Result {
		icp := strings.Split(resp.Result[x].NameOfSubscription, currency.UnderscoreDelimiter)
		if len(icp) < 3 {
			return errors.New("malformed futures candlestick websocket push data")
		}
		currencyPair, err := currency.NewPairFromString(strings.Join(icp[1:], currency.UnderscoreDelimiter))
		if err != nil {
			return err
		}
		g.Websocket.DataHandler <- stream.KlineData{
			Pair:       currencyPair,
			AssetType:  asset.Options,
			Exchange:   g.Name,
			StartTime:  time.Unix(resp.Result[x].Timestamp, 0),
			Interval:   icp[0],
			OpenPrice:  resp.Result[x].OpenPrice,
			ClosePrice: resp.Result[x].ClosePrice,
			HighPrice:  resp.Result[x].HighestPrice,
			LowPrice:   resp.Result[x].LowestPrice,
			Volume:     resp.Result[x].Amount,
		}
	}
	return nil
}

func (g *Gateio) processOrderbookTickerPushData(data []byte) error {
	var response WsResponse
	orderbookTicker := WsOptionsOrderbookTicker{}
	response.Result = &orderbookTicker
	err := json.Unmarshal(data, &orderbookTicker)
	if err != nil {
		return err
	}
	g.Websocket.DataHandler <- &response
	return nil
}

func (g *Gateio) processOptionsOrderbookSnapshotPushData(event string, data []byte) error {
	if event == "all" {
		var response WsResponse
		snapshot := WsOptionsOrderbookSnapshot{}
		response.Result = &snapshot
		err := json.Unmarshal(data, &response)
		if err != nil {
			return err
		}
		pair, err := currency.NewPairFromString(snapshot.Contract)
		if err != nil {
			return err
		}
		base := orderbook.Base{
			Asset:           asset.Options,
			Exchange:        g.Name,
			Pair:            pair,
			LastUpdated:     time.Unix(snapshot.Timestamp, 0),
			VerifyOrderbook: g.CanVerifyOrderbook,
		}
		base.Asks = make([]orderbook.Item, len(snapshot.Asks))
		base.Bids = make([]orderbook.Item, len(snapshot.Bids))
		for x := range base.Asks {
			base.Asks[x] = orderbook.Item{
				Amount: snapshot.Asks[x].Size,
				Price:  snapshot.Asks[x].Price,
			}
		}
		for x := range base.Bids {
			base.Bids[x] = orderbook.Item{
				Amount: snapshot.Bids[x].Size,
				Price:  snapshot.Bids[x].Price,
			}
		}
		return g.Websocket.Orderbook.LoadSnapshot(&base)
	}
	resp := struct {
		Time    int64                           `json:"time"`
		Channel string                          `json:"channel"`
		Event   string                          `json:"event"`
		Result  []WsFuturesOrderbookUpdateEvent `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	dataMap := map[string][2][]orderbook.Item{}
	for x := range resp.Result {
		ab, ok := dataMap[resp.Result[x].CurrencyPair]
		if !ok {
			ab = [2][]orderbook.Item{}
		}
		if resp.Result[x].Amount > 0 {
			ab[1] = append(ab[1], orderbook.Item{
				Price:  resp.Result[x].Price,
				Amount: resp.Result[x].Amount,
			})
		} else {
			ab[0] = append(ab[0], orderbook.Item{
				Price:  resp.Result[x].Price,
				Amount: -resp.Result[x].Amount,
			})
		}
		if !ok {
			dataMap[resp.Result[x].CurrencyPair] = ab
		}
	}
	if len(dataMap) == 0 {
		return errors.New("missing orderbook ask and bid data")
	}
	for key, ab := range dataMap {
		currencyPair, err := currency.NewPairFromString(key)
		if err != nil {
			return err
		}
		err = g.Websocket.Orderbook.LoadSnapshot(&orderbook.Base{
			Asks:            ab[0],
			Bids:            ab[1],
			Asset:           asset.Options,
			Exchange:        g.Name,
			Pair:            currencyPair,
			LastUpdated:     time.Unix(resp.Time, 0),
			VerifyOrderbook: g.CanVerifyOrderbook,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (g *Gateio) processOptionsOrderPushData(data []byte) error {
	resp := struct {
		Time    int64            `json:"time"`
		Channel string           `json:"channel"`
		Event   string           `json:"event"`
		Result  []WsOptionsOrder `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	for x := range resp.Result {
		currencyPair, err := currency.NewPairFromString(resp.Result[x].Contract)
		if err != nil {
			return err
		}
		status, err := order.StringToOrderStatus(func() string {
			if resp.Result[x].Status == "finished" {
				return "cancelled"
			}
			return resp.Result[x].Status
		}())
		if err != nil {
			return err
		}
		g.Websocket.DataHandler <- &order.Detail{
			Amount:         resp.Result[x].Size,
			Exchange:       g.Name,
			OrderID:        strconv.FormatInt(resp.Result[x].ID, 10),
			Status:         status,
			Pair:           currencyPair,
			Date:           time.UnixMilli(resp.Result[x].CreationTimeMs),
			ExecutedAmount: resp.Result[x].Size - resp.Result[x].Left,
			Price:          resp.Result[x].Price,
			AssetType:      asset.Options,
			AccountID:      resp.Result[x].User,
		}
	}
	return nil
}

func (g *Gateio) procesOptionsUserTradesPushData(data []byte) error {
	resp := struct {
		Time    int64                `json:"time"`
		Channel string               `json:"channel"`
		Event   string               `json:"event"`
		Result  []WsOptionsUserTrade `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	fills := make([]fill.Data, len(resp.Result))
	for x := range resp.Result {
		currencyPair, err := currency.NewPairFromString(resp.Result[x].Contract)
		if err != nil {
			return err
		}
		fills[x] = fill.Data{
			Timestamp:    time.UnixMilli(resp.Result[x].CreateTimeMs),
			Exchange:     g.Name,
			CurrencyPair: currencyPair,
			OrderID:      resp.Result[x].OrderID,
			TradeID:      resp.Result[x].ID,
			Price:        resp.Result[x].Price,
			Amount:       resp.Result[x].Size,
		}
	}
	return g.Websocket.Fills.Update(fills...)
}

func (g *Gateio) processOptionsLiquidatesPushData(data []byte) error {
	resp := struct {
		Time    int64                 `json:"time"`
		Channel string                `json:"channel"`
		Event   string                `json:"event"`
		Result  []WsOptionsLiquidates `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	g.Websocket.DataHandler <- &resp
	return nil
}

func (g *Gateio) processOptionsUsersPersonalSettlementsPushData(data []byte) error {
	resp := struct {
		Time    int64                     `json:"time"`
		Channel string                    `json:"channel"`
		Event   string                    `json:"event"`
		Result  []WsOptionsUserSettlement `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	g.Websocket.DataHandler <- &resp
	return nil
}

func (g *Gateio) processOptionsPositionPushData(data []byte) error {
	resp := struct {
		Time    int64               `json:"time"`
		Channel string              `json:"channel"`
		Event   string              `json:"event"`
		Result  []WsOptionsPosition `json:"result"`
	}{}
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	g.Websocket.DataHandler <- &resp
	return nil
}
