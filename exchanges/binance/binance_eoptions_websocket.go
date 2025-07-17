package binance

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
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
func (e *Exchange) WsOptionsConnect() error {
	ctx := context.Background()
	if !e.Websocket.IsEnabled() || !e.IsEnabled() {
		return websocket.ErrWebsocketNotEnabled
	}
	var err error
	var dialer gws.Dialer
	dialer.HandshakeTimeout = e.Config.HTTPTimeout
	dialer.Proxy = http.ProxyFromEnvironment
	wsURL := eoptionsWebsocketURL + "stream"
	err = e.Websocket.SetWebsocketURL(wsURL, false, false)
	if err != nil {
		e.Websocket.SetCanUseAuthenticatedEndpoints(false)
		log.Errorf(log.ExchangeSys,
			"%v unable to connect to authenticated Websocket. Error: %s", e.Name, err)
	}
	if e.Websocket.CanUseAuthenticatedEndpoints() {
		listenKey, err = e.GetEOptionsWsAuthStreamKey(context.TODO())
		switch {
		case err != nil:
			e.Websocket.SetCanUseAuthenticatedEndpoints(false)
			log.Errorf(log.ExchangeSys, "%v unable to connect to authenticated Websocket. Error: %s", e.Name, err)
		default:
			wsURL = wsURL + "ws/" + listenKey
			err = e.Websocket.SetWebsocketURL(wsURL, false, false)
			if err != nil {
				return err
			}
		}
	}
	err = e.Websocket.SetWebsocketURL(wsURL, false, false)
	if err != nil {
		return err
	}
	err = e.Websocket.Conn.Dial(ctx, &dialer, http.Header{})
	if err != nil {
		return fmt.Errorf("%v - Unable to connect to Websocket. Error: %s", e.Name, err)
	}
	e.Websocket.Wg.Add(1)
	go e.wsEOptionsFuturesReadData()

	e.Websocket.Conn.SetupPingHandler(request.UnAuth, websocket.PingHandler{
		UseGorillaHandler: true,
		MessageType:       gws.PongMessage,
		Delay:             pingDelay,
	})
	subscriptions, err := e.GenerateEOptionsDefaultSubscriptions()
	if err != nil {
		return err
	}
	return e.OptionSubscribe(subscriptions)
}

func (e *Exchange) handleEOptionsSubscriptions(operation string, subscs subscription.List) error {
	if len(subscs) == 0 {
		return common.ErrEmptyParams
	}
	params := &EOptionSubscriptionParam{
		Method: operation,
		Params: make([]string, 0, len(subscs)),
		ID:     e.Websocket.Conn.GenerateMessageID(false),
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
			intervalString := e.intervalToString(subscs[s].Interval)
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
				intervalString = "@" + e.intervalToString(subscs[s].Interval)
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

	response, err := e.Websocket.Conn.SendMessageReturnResponse(context.Background(), request.UnAuth, params.ID, params)
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
		err = e.Websocket.AddSuccessfulSubscriptions(e.Websocket.Conn, subscs...)
		if err != nil {
			return err
		}
	}
	return e.Websocket.RemoveSubscriptions(e.Websocket.Conn, subscs...)
}

// OptionSubscribe sends an european option subscription messages.
func (e *Exchange) OptionSubscribe(subscs subscription.List) error {
	return e.handleEOptionsSubscriptions("SUBSCRIBE", subscs)
}

// OptionUnsubscribe unsubscribes an option un-subscription messages.
func (e *Exchange) OptionUnsubscribe(subscs subscription.List) error {
	return e.handleEOptionsSubscriptions("UNSUBSCRIBE", subscs)
}

// GenerateEOptionsDefaultSubscriptions generates the default subscription set
func (e *Exchange) GenerateEOptionsDefaultSubscriptions() (subscription.List, error) {
	channels := defaultEOptionsSubscriptions
	var subscriptions subscription.List
	pairs, err := e.FetchTradablePairs(context.Background(), asset.Options)
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
func (e *Exchange) GetEOptionsWsAuthStreamKey(ctx context.Context) (string, error) {
	endpointPath, err := e.API.Endpoints.GetURL(exchange.RestOptions)
	if err != nil {
		return "", err
	}

	creds, err := e.GetCredentials(ctx)
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
		Verbose:       e.Verbose,
		HTTPDebugging: e.HTTPDebugging,
		HTTPRecording: e.HTTPRecording,
	}

	return resp.ListenKey, e.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
		return item, nil
	}, request.AuthenticatedRequest)
}

// wsEOptionsFuturesReadData receives and passes on websocket messages for processing
// for European Options instruments.
func (e *Exchange) wsEOptionsFuturesReadData() {
	defer e.Websocket.Wg.Done()
	for {
		resp := e.Websocket.Conn.ReadMessage()
		if resp.Raw == nil {
			return
		}
		err := e.wsHandleEOptionsData(resp.Raw)
		if err != nil {
			e.Websocket.DataHandler <- err
		}
	}
}

func (e *Exchange) wsHandleEOptionsData(respRaw []byte) error {
	var result WsOptionIncomingResps
	err := json.Unmarshal(respRaw, &result)
	if err != nil {
		return err
	}
	if result.Instances[0].EventType == "" || (result.Instances[0].ID != 0 && result.Instances[0].Result != nil) {
		if !e.Websocket.Match.IncomingWithData(result.Instances[0].ID, respRaw) {
			return errors.New("Unhandled data: " + string(respRaw))
		}
		return nil
	}
	switch result.Instances[0].Stream {
	case cnlTrade:
		return e.processOptionsTradeStream(respRaw)
	case cnlIndex:
		return e.processOptionsIndexPrice(respRaw)
	case "24hrTicker":
		return e.processOptionsTicker(respRaw, result.IsSlice)
	case "markPrice":
		return e.processOptionsMarkPrices(respRaw)
	case "kline":
		return e.processOptionsKline(respRaw)
	case "openInterest":
		return e.processOptionsOpenInterest(respRaw)
	case "option_pair":
		return e.processOptionsPair(respRaw)
	case "depth":
		return e.processOptionsOrderbook(respRaw)
	default:
		e.Websocket.DataHandler <- websocket.UnhandledMessageWarning{
			Message: string(respRaw),
		}
		return fmt.Errorf("unhandled stream data %s", string(respRaw))
	}
}

// orderbookSnapshotLoadedPairsMap used for validation of whether the symbol has snapshot orderbook data in the buffer or not.
var orderbookSnapshotLoadedPairsMap = map[string]bool{}

func (e *Exchange) processOptionsOrderbook(data []byte) error {
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
		err = e.Websocket.Orderbook.LoadSnapshot(&orderbook.Book{
			Pair:         pair,
			Exchange:     e.Name,
			Asset:        asset.Options,
			LastUpdated:  resp.TransactionTime.Time(),
			LastUpdateID: resp.UpdateID,
			Asks:         orderbook.Levels(resp.Asks),
			Bids:         orderbook.Levels(resp.Bids),
		})
		if err != nil {
			return err
		}
	} else {
		err = e.Websocket.Orderbook.Update(&orderbook.Update{
			Pair:       pair,
			Asks:       orderbook.Levels(resp.Asks),
			Bids:       orderbook.Levels(resp.Bids),
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
func (e *Exchange) processOptionsPair(data []byte) error {
	var resp WsOptionsNewPair
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	e.Websocket.DataHandler <- resp
	return nil
}

func (e *Exchange) processOptionsOpenInterest(data []byte) error {
	var resp []WsOpenInterest
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	e.Websocket.DataHandler <- resp
	return nil
}

func (e *Exchange) processOptionsKline(data []byte) error {
	var resp WsOptionsKlineData
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	pair, err := currency.NewPairFromString(resp.KlineData.Symbol)
	if err != nil {
		return err
	}
	e.Websocket.DataHandler <- websocket.KlineData{
		Timestamp:  resp.EventTime.Time(),
		Pair:       pair,
		AssetType:  asset.Options,
		Exchange:   e.Name,
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

func (e *Exchange) processOptionsMarkPrices(data []byte) error {
	var resp []WsOptionsMarkPrice
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	e.Websocket.DataHandler <- resp
	return nil
}

func (e *Exchange) processOptionsIndexPrice(data []byte) error {
	var resp OptionsIndexInfo
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	e.Websocket.DataHandler <- resp
	return nil
}

func (e *Exchange) processOptionsTicker(data []byte, isSlice bool) error {
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
		e.Websocket.DataHandler <- ticker.Price{
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
			ExchangeName: e.Name,
			AssetType:    asset.Options,
			LastUpdated:  resp[a].EventTime.Time(),
		}
	}
	return nil
}

func (e *Exchange) processOptionsTradeStream(data []byte) error {
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
	e.Websocket.DataHandler <- trade.Data{
		TID:          strconv.FormatInt(resp.TradeID, 10),
		Exchange:     e.Name,
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
	kline.SixHour: "6h", kline.TwelveHour: "12h", kline.OneDay: "1d", kline.ThreeDay: "3d", kline.OneWeek: "1w",
}

func (e *Exchange) intervalToString(interval kline.Interval) string {
	intervalString, okay := intervalsMap[interval]
	if !okay {
		return ""
	}
	return intervalString
}
