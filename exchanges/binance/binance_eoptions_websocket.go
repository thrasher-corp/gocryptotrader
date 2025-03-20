package binance

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	eoptionsWebsocketURL = "wss://nbstream.binance.com/eoptions/"

	// For convention, we use the @channel_type pattern to represents channels that use underlying asset like ETH otherwise they use symbols or none

	cnlTrade                    = "trade"  // <symbol>@trade
	cnlTradeWithUnderlyingAsset = "@trade" // <underlyingAsset>@trade eg. ETH@trade
	cnlIndex                    = "index"
	cnlMarkPrice                = "@markPrice"
	cnlKline                    = "kline"
	cnlTicker                   = "ticker"
	cnlTickerWithExpiration     = "@ticker@"
	cnlOpenInterest             = "@openInterest@"
	cnlDepth                    = "depth"
	cnlOptionPair               = "option_pair"
)

// defaultEOptionsSubscriptions list of default subscription channels
var defaultEOptionsSubscriptions = []string{
	cnlTicker,
	cnlKline,
	cnlDepth,
}

// WsOptionsConnect initiates a websocket connection to coin margined futures websocket
func (b *Binance) WsOptionsConnect() error {
	if !b.Websocket.IsEnabled() || !b.IsEnabled() {
		return stream.ErrWebsocketNotEnabled
	}
	var err error
	var dialer websocket.Dialer
	dialer.HandshakeTimeout = b.Config.HTTPTimeout
	dialer.Proxy = http.ProxyFromEnvironment
	wsURL := eoptionsWebsocketURL + "stream"
	err = b.Websocket.SetWebsocketURL(wsURL, false, false)
	if err != nil {
		b.Websocket.SetCanUseAuthenticatedEndpoints(false)
		log.Errorf(log.ExchangeSys,
			"%v unable to connect to authenticated Websocket. Error: %s", b.Name, err)
	}
	if b.Websocket.CanUseAuthenticatedEndpoints() {
		listenKey, err = b.GetEOptionsWsAuthStreamKey(context.TODO())
		switch {
		case err != nil:
			b.Websocket.SetCanUseAuthenticatedEndpoints(false)
			log.Errorf(log.ExchangeSys, "%v unable to connect to authenticated Websocket. Error: %s", b.Name, err)
		default:
			wsURL = wsURL + "ws/" + listenKey
			err = b.Websocket.SetWebsocketURL(wsURL, false, false)
			if err != nil {
				return err
			}
		}
	}
	err = b.Websocket.SetWebsocketURL(wsURL, false, false)
	if err != nil {
		return err
	}
	err = b.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return fmt.Errorf("%v - Unable to connect to Websocket. Error: %s", b.Name, err)
	}
	b.Websocket.Wg.Add(1)
	go b.wsEOptionsFuturesReadData()

	b.Websocket.Conn.SetupPingHandler(request.UnAuth, stream.PingHandler{
		UseGorillaHandler: true,
		MessageType:       websocket.PongMessage,
		Delay:             pingDelay,
	})
	subscriptions, err := b.GenerateEOptionsDefaultSubscriptions()
	if err != nil {
		return err
	}
	return b.OptionSubscribe(subscriptions)
}

func (b *Binance) handleEOptionsSubscriptions(operation string, subscs subscription.List) error {
	if len(subscs) == 0 {
		return common.ErrEmptyParams
	}
	params := &EOptionSubscriptionParam{
		Method: operation,
		Params: make([]string, 0, len(subscs)),
		ID:     b.Websocket.Conn.GenerateMessageID(false),
	}
	for s := range subscs {
		switch subscs[s].Channel {
		case cnlTrade, cnlIndex, cnlTicker: // subscriptions with <symbol>@channel pattern
			for p := range subscs[s].Pairs {
				params.Params = append(params.Params, subscs[s].Pairs[p].String()+"@"+subscs[s].Channel)
			}
		case cnlTradeWithUnderlyingAsset, cnlMarkPrice: // subscriptions with <underlyingAsset>@channel
			for p := range subscs[s].Pairs {
				params.Params = append(params.Params, subscs[s].Pairs[p].Base.String()+"@"+subscs[s].Channel)
			}
		case cnlKline: // subscriptions with <symbol>@channel<interval> pattern
			intervalString := b.intervalToString(subscs[s].Interval)
			if intervalString == "" {
				intervalString = "15m"
			}
			for p := range subscs[s].Pairs {
				params.Params = append(params.Params, subscs[s].Pairs[p].String()+"@"+subscs[s].Channel+"_"+intervalString)
			}
		case cnlTickerWithExpiration, cnlOpenInterest: // subscriptions with <underlyingAsset>@channel@<expirationDate>
			var expirationTime time.Time
			expirationTimeInterface, okay := subscs[s].Params["expiration"]
			if !okay {
				// default: five day expiration time
				expirationTime = time.Now().Add(time.Hour * 24 * 5)
			} else {
				expirationTime, okay = expirationTimeInterface.(time.Time)
				if !okay {
					// default: five day expiration time
					expirationTime = time.Now().Add(time.Hour * 24 * 5)
				}
			}
			expirationTimeString := fmt.Sprintf("%2d%2d%2d", expirationTime.Year(), expirationTime.Month(), expirationTime.Day())
			for p := range subscs[s].Pairs {
				params.Params = append(params.Params, subscs[s].Pairs[p].String()+subscs[s].Channel+expirationTimeString)
			}
		case cnlDepth:
			level, okay := subscs[s].Params["level"].(string)
			if !okay {
				// deefault level set to 50
				level = "10"
			}
			var intervalString string
			if subscs[s].Interval != kline.Interval(0) {
				intervalString = "@" + b.intervalToString(subscs[s].Interval)
			}
			for p := range subscs[s].Pairs {
				params.Params = append(params.Params, subscs[s].Pairs[p].String()+"@"+subscs[s].Channel+"@"+level+intervalString)
			}
		case cnlOptionPair:
			params.Params = append(params.Params, subscs[s].Channel)
		default:
			return errors.New("unsupported channel")
		}
	}

	response, err := b.Websocket.Conn.SendMessageReturnResponse(context.Background(), request.UnAuth, params.ID, params)
	if err != nil {
		return err
	}
	var resp EOptionsOperationResponse
	err = json.Unmarshal(response, &resp)
	if err != nil {
		return err
	} else if resp.Error.Code != 0 {
		return fmt.Errorf("err: code: %d, msg: %s", resp.Error.Code, resp.Error.Message)
	}
	if operation == "SUBSCRIBE" {
		err = b.Websocket.AddSuccessfulSubscriptions(b.Websocket.Conn, subscs...)
		if err != nil {
			return err
		}
	}
	return b.Websocket.RemoveSubscriptions(b.Websocket.Conn, subscs...)
}

// OptionSubscribe sends an european option subscription messages.
func (b *Binance) OptionSubscribe(subscs subscription.List) error {
	return b.handleEOptionsSubscriptions("SUBSCRIBE", subscs)
}

// OptionUnsubscribe unsubscribes an option un-subscription messages.
func (b *Binance) OptionUnsubscribe(subscs subscription.List) error {
	return b.handleEOptionsSubscriptions("UNSUBSCRIBE", subscs)
}

// GenerateEOptionsDefaultSubscriptions generates the default subscription set
func (b *Binance) GenerateEOptionsDefaultSubscriptions() (subscription.List, error) {
	channels := defaultEOptionsSubscriptions
	var subscriptions subscription.List
	pairs, err := b.FetchTradablePairs(context.Background(), asset.Options)
	if err != nil {
		return nil, err
	}
	if len(pairs) > 4 {
		pairs = pairs[:3]
	}

	for z := range channels {
		switch channels[z] {
		case cnlTrade, cnlMarkPrice, cnlIndex, cnlTicker, cnlTradeWithUnderlyingAsset:
			subscriptions = append(subscriptions, &subscription.Subscription{
				Channel: channels[z],
				Pairs:   pairs,
				Asset:   asset.Options,
			})
		case cnlKline:
			subscriptions = append(subscriptions, &subscription.Subscription{
				Channel:  cnlKline,
				Pairs:    pairs,
				Asset:    asset.Options,
				Interval: kline.FiveMin,
			})
		case cnlTickerWithExpiration, cnlOpenInterest:
			subscriptions = append(subscriptions, &subscription.Subscription{
				Channel: channels[z],
				Pairs:   pairs,
				Asset:   asset.Options,
				Params: map[string]interface{}{
					"expiration": time.Now().Add(time.Hour * 24 * 5),
				},
			})
		case cnlDepth:
			subscriptions = append(subscriptions, &subscription.Subscription{
				Channel:  cnlDepth,
				Pairs:    pairs,
				Asset:    asset.Options,
				Interval: kline.FiveHundredMilliseconds,
				Params: map[string]interface{}{
					"level": 50, // Valid levels are 10, 20, 50, 100.
				},
			})
		case cnlOptionPair:
			subscriptions = append(subscriptions, &subscription.Subscription{
				Channel: cnlOptionPair,
			})
		default:
			return nil, errors.New("unsupported subscription")
		}
	}
	return subscriptions, nil
}

// GetEOptionsWsAuthStreamKey will retrieve a key to use for authorised WS streaming
func (b *Binance) GetEOptionsWsAuthStreamKey(ctx context.Context) (string, error) {
	endpointPath, err := b.API.Endpoints.GetURL(exchange.RestOptions)
	if err != nil {
		return "", err
	}

	creds, err := b.GetCredentials(ctx)
	if err != nil {
		return "", err
	}

	var resp UserAccountStream
	headers := make(map[string]string)
	headers["X-MBX-APIKEY"] = creds.Key
	item := &request.Item{
		Method:        http.MethodPost,
		Path:          endpointPath + "/eapi/v1/listenKey",
		Headers:       headers,
		Result:        &resp,
		Verbose:       b.Verbose,
		HTTPDebugging: b.HTTPDebugging,
		HTTPRecording: b.HTTPRecording,
	}

	return resp.ListenKey, b.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
		return item, nil
	}, request.AuthenticatedRequest)
}

// wsEOptionsFuturesReadData receives and passes on websocket messages for processing
// for European Options instruments.
func (b *Binance) wsEOptionsFuturesReadData() {
	defer b.Websocket.Wg.Done()
	for {
		resp := b.Websocket.Conn.ReadMessage()
		if resp.Raw == nil {
			return
		}
		err := b.wsHandleEOptionsData(resp.Raw)
		if err != nil {
			b.Websocket.DataHandler <- err
		}
	}
}

func (b *Binance) wsHandleEOptionsData(respRaw []byte) error {
	var result WsOptionIncomingResps
	err := json.Unmarshal(respRaw, &result)
	if err != nil {
		return err
	}
	if result.Instances[0].EventType == "" || (result.Instances[0].ID != 0 && result.Instances[0].Result != nil) {
		if !b.Websocket.Match.IncomingWithData(result.Instances[0].ID, respRaw) {
			return errors.New("Unhandled data: " + string(respRaw))
		}
		return nil
	}
	switch result.Instances[0].Stream {
	case cnlTrade:
		return b.processOptionsTradeStream(respRaw)
	case cnlIndex:
		return b.processOptionsIndexPrice(respRaw)
	case "24hrTicker":
		return b.processOptionsTicker(respRaw, result.IsSlice)
	case "markPrice":
		return b.processOptionsMarkPrices(respRaw)
	case "kline":
		return b.processOptionsKline(respRaw)
	case "openInterest":
		return b.processOptionsOpenInterest(respRaw)
	case "option_pair":
		return b.processOptionsPair(respRaw)
	case "depth":
		return b.processOptionsOrderbook(respRaw)
	default:
		b.Websocket.DataHandler <- stream.UnhandledMessageWarning{
			Message: string(respRaw),
		}
		return fmt.Errorf("unhandled stream data %s", string(respRaw))
	}
}

// orderbookSnapshotLoadedPairsMap used for validation of whether the symbol has snapshot orderbook data in the buffer or not.
var orderbookSnapshotLoadedPairsMap = map[string]bool{}

func (b *Binance) processOptionsOrderbook(data []byte) error {
	var resp WsOptionsOrderbook
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	pair, err := currency.NewPairFromString(resp.OptionSymbol)
	if err != nil {
		return err
	}
	if len(resp.Asks) == 0 && len(resp.Bids) == 0 {
		return nil
	}
	okay := orderbookSnapshotLoadedPairsMap[resp.OptionSymbol]
	if !okay {
		err = b.Websocket.Orderbook.LoadSnapshot(&orderbook.Base{
			Pair:         pair,
			Exchange:     b.Name,
			Asset:        asset.Options,
			LastUpdated:  resp.TransactionTime.Time(),
			LastUpdateID: resp.UpdateID,
			Asks:         orderbook.Tranches(resp.Asks),
			Bids:         orderbook.Tranches(resp.Bids),
		})
		if err != nil {
			return err
		}
	} else {
		err = b.Websocket.Orderbook.Update(&orderbook.Update{
			Pair:       pair,
			Asks:       orderbook.Tranches(resp.Asks),
			Bids:       orderbook.Tranches(resp.Bids),
			Asset:      asset.Options,
			UpdateID:   resp.UpdateID,
			UpdateTime: resp.TransactionTime.Time(),
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// processOptionsPair new symbol listing stream
func (b *Binance) processOptionsPair(data []byte) error {
	var resp WsOptionsNewPair
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	b.Websocket.DataHandler <- resp
	return nil
}

func (b *Binance) processOptionsOpenInterest(data []byte) error {
	var resp []WsOpenInterest
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	b.Websocket.DataHandler <- resp
	return nil
}

func (b *Binance) processOptionsKline(data []byte) error {
	var resp WsOptionsKlineData
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	pair, err := currency.NewPairFromString(resp.KlineData.Symbol)
	if err != nil {
		return err
	}
	b.Websocket.DataHandler <- stream.KlineData{
		Timestamp:  resp.EventTime.Time(),
		Pair:       pair,
		AssetType:  asset.Options,
		Exchange:   b.Name,
		StartTime:  resp.KlineData.StartTime.Time(),
		CloseTime:  resp.KlineData.EndTime.Time(),
		Interval:   strings.Split(resp.EventType, "_")[1],
		OpenPrice:  resp.KlineData.Open.Float64(),
		ClosePrice: resp.KlineData.Close.Float64(),
		HighPrice:  resp.KlineData.High.Float64(),
		LowPrice:   resp.KlineData.Low.Float64(),
		Volume:     resp.KlineData.ContractVolume.Float64(),
	}
	return nil
}

func (b *Binance) processOptionsMarkPrices(data []byte) error {
	var resp []WsOptionsMarkPrice
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	b.Websocket.DataHandler <- resp
	return nil
}

func (b *Binance) processOptionsIndexPrice(data []byte) error {
	var resp OptionsIndexInfo
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	b.Websocket.DataHandler <- resp
	return nil
}

func (b *Binance) processOptionsTicker(data []byte, isSlice bool) error {
	var resp []OptionsTicker24Hr
	if isSlice {
		err := json.Unmarshal(data, &resp)
		if err != nil {
			return err
		}
	} else {
		respSingle := OptionsTicker24Hr{}
		err := json.Unmarshal(data, &resp)
		if err != nil {
			return err
		}
		resp = append(resp, respSingle)
	}
	for a := range resp {
		pair, err := currency.NewPairFromString(resp[a].Symbol)
		if err != nil {
			return err
		}
		b.Websocket.DataHandler <- ticker.Price{
			High:         resp[a].HightPrice.Float64(),
			Low:          resp[a].LowPrice.Float64(),
			Bid:          resp[a].BestBuyPrice.Float64(),
			Ask:          resp[a].BestSellPrice.Float64(),
			Volume:       resp[a].TradingVolume.Float64(),
			QuoteVolume:  resp[a].TradingAmount.Float64(),
			Open:         resp[a].OpeningPrice.Float64(),
			Close:        resp[a].ClosingPrice.Float64(),
			MarkPrice:    resp[a].MarkPrice.Float64(),
			Pair:         pair,
			ExchangeName: b.Name,
			AssetType:    asset.Options,
			LastUpdated:  resp[a].EventTime.Time(),
		}
	}
	return nil
}

func (b *Binance) processOptionsTradeStream(data []byte) error {
	var resp *EOptionsWsTrade
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	pair, err := currency.NewPairFromString(resp.Symbol)
	if err != nil {
		return err
	}
	var side order.Side
	if resp.Direction == "-1" {
		side = order.Sell
	} else {
		side = order.Buy
	}
	b.Websocket.DataHandler <- trade.Data{
		TID:          strconv.FormatInt(resp.TradeID, 10),
		Exchange:     b.Name,
		CurrencyPair: pair,
		AssetType:    asset.Options,
		Side:         side,
		Price:        resp.Price.Float64(),
		Amount:       resp.Quantity.Float64(),
		Timestamp:    resp.TradeCompletedTime.Time(),
	}
	return nil
}

var intervalsMap = map[kline.Interval]string{
	// Intervals used by the orderbook depth
	kline.HundredMilliseconds: "100ms", kline.FiveHundredMilliseconds: "500ms",
	kline.ThousandMilliseconds: "1000ms",

	// other intervals
	kline.OneMin: "1m", kline.ThreeMin: "3m", kline.FiveMin: "5m", kline.FifteenMin: "15m",
	kline.ThirtyMin: "30m", kline.OneHour: "1h", kline.TwoHour: "2h", kline.FourHour: "4h",
	kline.SixHour: "6h", kline.TwelveHour: "12h", kline.OneDay: "1d", kline.ThreeDay: "3d",
	kline.OneWeek: "1w"}

func (b *Binance) intervalToString(interval kline.Interval) string {
	intervalString, okay := intervalsMap[interval]
	if !okay {
		return ""
	}
	return intervalString
}
