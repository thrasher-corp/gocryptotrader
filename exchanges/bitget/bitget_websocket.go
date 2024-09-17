package bitget

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	bitgetPublicWSURL  = "wss://ws.bitget.com/v2/ws/public"
	bitgetPrivateWSURL = "wss://ws.bitget.com/v2/ws/private"
)

var subscriptionNames = map[string]string{
	subscription.TickerChannel:    bitgetTicker,
	subscription.CandlesChannel:   bitgetCandleDailyChannel,
	subscription.AllOrdersChannel: bitgetTrade,
	subscription.OrderbookChannel: bitgetBookFullChannel,
	"account":                     bitgetAccount,
	subscription.AllTradesChannel: bitgetFillChannel,
}

var defaultSubscriptions = subscription.List{
	{Enabled: false, Channel: subscription.TickerChannel, Asset: asset.Spot},
	{Enabled: false, Channel: subscription.CandlesChannel, Asset: asset.Spot},
	{Enabled: false, Channel: subscription.AllOrdersChannel, Asset: asset.Spot},
	{Enabled: false, Channel: subscription.OrderbookChannel, Asset: asset.Spot},
	{Enabled: false, Channel: "account", Authenticated: true, Asset: asset.Spot},
	{Enabled: true, Channel: subscription.AllTradesChannel, Authenticated: true, Asset: asset.Spot},
}

// WsConnect connects to a websocket feed
func (bi *Bitget) WsConnect() error {
	if !bi.Websocket.IsEnabled() || !bi.IsEnabled() {
		return stream.ErrWebsocketNotEnabled
	}
	var dialer websocket.Dialer
	err := bi.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	if bi.Verbose {
		log.Debugf(log.ExchangeSys, "%s connected to Websocket.\n", bi.Name)
	}
	bi.Websocket.Wg.Add(1)
	go bi.wsReadData(bi.Websocket.Conn)
	stream.Connection.SetupPingHandler(bi.Websocket.Conn, stream.PingHandler{
		Websocket:   true,
		Message:     []byte(`ping`),
		MessageType: websocket.TextMessage,
		Delay:       time.Second * 25,
	})
	if bi.IsWebsocketAuthenticationSupported() {
		var authDialer websocket.Dialer
		err = bi.WsAuth(context.TODO(), &authDialer)
		if err != nil {
			log.Errorf(log.ExchangeSys, "Error connecting auth socket: %s\n", err.Error())
			bi.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
	}
	return nil
}

// WsAuth sends an authentication message to the websocket
func (bi *Bitget) WsAuth(ctx context.Context, dialer *websocket.Dialer) error {
	if !bi.Websocket.CanUseAuthenticatedEndpoints() {
		return fmt.Errorf(errAuthenticatedWebsocketDisabled, bi.Name)
	}
	err := bi.Websocket.AuthConn.Dial(dialer, http.Header{})
	if err != nil {
		return err
	}
	bi.Websocket.Wg.Add(1)
	go bi.wsReadData(bi.Websocket.AuthConn)
	stream.Connection.SetupPingHandler(bi.Websocket.Conn, stream.PingHandler{
		Websocket:   true,
		Message:     []byte(`ping`),
		MessageType: websocket.TextMessage,
		Delay:       time.Second * 25,
	})
	creds, err := bi.GetCredentials(ctx)
	if err != nil {
		return err
	}
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	message := timestamp + "GET" + "/user/verify"
	hmac, err := crypto.GetHMAC(crypto.HashSHA256, []byte(message), []byte(creds.Secret))
	if err != nil {
		return err
	}
	base64Sign := crypto.Base64Encode(hmac)
	payload := WsLogin{
		Operation: "login",
		Arguments: []WsLoginArgument{
			{
				APIKey:     creds.Key,
				Signature:  base64Sign,
				Timestamp:  timestamp,
				Passphrase: creds.ClientID,
			},
		},
	}
	err = bi.Websocket.AuthConn.SendJSONMessage(payload)
	if err != nil {
		return err
	}
	return nil
}

// wsReadData receives and passes on websocket messages for processing
func (bi *Bitget) wsReadData(ws stream.Connection) {
	defer bi.Websocket.Wg.Done()
	for {
		resp := ws.ReadMessage()
		if resp.Raw == nil {
			return
		}
		err := bi.wsHandleData(resp.Raw)
		if err != nil {
			bi.Websocket.DataHandler <- err
		}
	}
}

// wsHandleData handles data from the websocket connection
func (bi *Bitget) wsHandleData(respRaw []byte) error {
	var wsResponse WsResponse
	if respRaw != nil && string(respRaw[:4]) == "pong" {
		if bi.Verbose {
			log.Debugf(log.ExchangeSys, "%v - Websocket pong received\n", bi.Name)
		}
		return nil
	}
	err := json.Unmarshal(respRaw, &wsResponse)
	if err != nil {
		return err
	}
	// Under the assumption that the exchange only ever sends one of these. If both can be sent, this will need to
	// be made more complicated
	toCheck := wsResponse.Event + wsResponse.Action
	switch toCheck {
	case "subscribe":
		if bi.Verbose {
			log.Debugf(log.ExchangeSys, "%v - Websocket %v succeeded for %v\n", bi.Name, wsResponse.Event,
				wsResponse.Arg)
		}
	case "error":
		return fmt.Errorf("%v - Websocket error, code: %v message: %v", bi.Name, wsResponse.Code, wsResponse.Message)
	case "login":
		if wsResponse.Code != 0 {
			return fmt.Errorf("%v - Websocket login failed: %v", bi.Name, wsResponse.Message)
		}
		if bi.Verbose {
			log.Debugf(log.ExchangeSys, "%v - Websocket login succeeded\n", bi.Name)
		}
	case "snapshot":
		switch wsResponse.Arg.Channel {
		case bitgetTicker:
			var ticks []WsTickerSnapshot
			err := json.Unmarshal(wsResponse.Data, &ticks)
			if err != nil {
				return err
			}
			for i := range ticks {
				pair, err := pairFromStringHelper(ticks[i].InstrumentID)
				if err != nil {
					return err
				}
				bi.Websocket.DataHandler <- &ticker.Price{
					Last:         ticks[i].LastPrice,
					High:         ticks[i].High24H,
					Low:          ticks[i].Low24H,
					Bid:          ticks[i].BidPrice,
					Ask:          ticks[i].AskPrice,
					Volume:       ticks[i].BaseVolume,
					QuoteVolume:  ticks[i].QuoteVolume,
					Open:         ticks[i].Open24H,
					Pair:         pair,
					ExchangeName: bi.Name,
					AssetType:    asset.Spot,
					LastUpdated:  ticks[i].Timestamp.Time(),
				}
			}
		case bitgetCandleDailyChannel:
			var candles [][8]string
			err := json.Unmarshal(wsResponse.Data, &candles)
			if err != nil {
				return err
			}
			pair, err := pairFromStringHelper(wsResponse.Arg.InstrumentID)
			if err != nil {
				return err
			}
			resp := make([]stream.KlineData, len(candles))
			for i := range candles {
				ts, err := strconv.ParseInt(candles[i][0], 10, 64)
				if err != nil {
					return err
				}
				open, err := strconv.ParseFloat(candles[i][1], 64)
				if err != nil {
					return err
				}
				close, err := strconv.ParseFloat(candles[i][4], 64)
				if err != nil {
					return err
				}
				high, err := strconv.ParseFloat(candles[i][2], 64)
				if err != nil {
					return err
				}
				low, err := strconv.ParseFloat(candles[i][3], 64)
				if err != nil {
					return err
				}
				volume, err := strconv.ParseFloat(candles[i][5], 64)
				if err != nil {
					return err
				}
				resp[i] = stream.KlineData{
					Timestamp:  wsResponse.Timestamp.Time(),
					Pair:       pair,
					AssetType:  asset.Spot,
					Exchange:   bi.Name,
					StartTime:  time.UnixMilli(ts),
					CloseTime:  time.UnixMilli(ts).Add(time.Hour * 24),
					Interval:   "1d",
					OpenPrice:  open,
					ClosePrice: close,
					HighPrice:  high,
					LowPrice:   low,
					Volume:     volume,
				}
			}
			bi.Websocket.DataHandler <- resp
		case bitgetTrade:
			resp, err := bi.tradeDataHandler(wsResponse)
			if err != nil {
				return err
			}
			bi.Websocket.DataHandler <- resp
		case bitgetBookFullChannel:
			err := bi.orderbookDataHandler(wsResponse)
			if err != nil {
				return err
			}
		case bitgetAccount:
			var acc []WsAccountResponse
			err := json.Unmarshal(wsResponse.Data, &acc)
			if err != nil {
				return err
			}
			var hold account.Holdings
			hold.Exchange = bi.Name
			var sub account.SubAccount
			sub.AssetType = asset.Spot
			sub.Currencies = make([]account.Balance, len(acc))
			for i := range acc {
				sub.Currencies[i] = account.Balance{
					Currency: currency.NewCode(acc[i].Coin),
					Hold:     acc[i].Frozen + acc[i].Locked,
					Free:     acc[i].Available,
					Total:    sub.Currencies[i].Hold + sub.Currencies[i].Free,
				}
			}
			// Plan to add handling of account.Holdings on websocketDataHandler side in a later PR
			bi.Websocket.DataHandler <- hold
		default:
			bi.Websocket.DataHandler <- stream.UnhandledMessageWarning{Message: bi.Name + stream.UnhandledMessage +
				string(respRaw)}
		}
	case "update":
		switch wsResponse.Arg.Channel {
		case bitgetCandleDailyChannel:
			var candles [][8]string
			err := json.Unmarshal(wsResponse.Data, &candles)
			if err != nil {
				return err
			}
			pair, err := pairFromStringHelper(wsResponse.Arg.InstrumentID)
			if err != nil {
				return err
			}
			resp := make([]stream.IncompleteKline, len(candles))
			for i := range candles {
				ts, err := strconv.ParseInt(candles[i][0], 10, 64)
				if err != nil {
					return err
				}
				open, err := strconv.ParseFloat(candles[i][1], 64)
				if err != nil {
					return err
				}
				close, err := strconv.ParseFloat(candles[i][4], 64)
				if err != nil {
					return err
				}
				high, err := strconv.ParseFloat(candles[i][2], 64)
				if err != nil {
					return err
				}
				low, err := strconv.ParseFloat(candles[i][3], 64)
				if err != nil {
					return err
				}
				volume, err := strconv.ParseFloat(candles[i][5], 64)
				if err != nil {
					return err
				}
				resp[i] = stream.IncompleteKline{
					Timestamp:  wsResponse.Timestamp.Time(),
					Pair:       pair,
					AssetType:  asset.Spot,
					Exchange:   bi.Name,
					StartTime:  time.UnixMilli(ts),
					CloseTime:  time.UnixMilli(ts).Add(time.Hour * 24),
					Interval:   "1d",
					OpenPrice:  open,
					ClosePrice: close,
					HighPrice:  high,
					LowPrice:   low,
					Volume:     volume,
				}
			}
			bi.Websocket.DataHandler <- resp
		case bitgetTrade:
			resp, err := bi.tradeDataHandler(wsResponse)
			if err != nil {
				return err
			}
			bi.Websocket.DataHandler <- resp
		case bitgetBookFullChannel:
			err := bi.orderbookDataHandler(wsResponse)
			if err != nil {
				return err
			}
		case bitgetAccount:
			var acc []WsAccountResponse
			err := json.Unmarshal(wsResponse.Data, &acc)
			if err != nil {
				return err
			}
			resp := make([]account.Change, len(acc))
			for i := range acc {
				resp[i] = account.Change{
					Exchange: bi.Name,
					Currency: currency.NewCode(acc[i].Coin),
					Asset:    asset.Spot,
					Amount:   acc[i].Available,
				}
			}
			bi.Websocket.DataHandler <- resp
		default:
			bi.Websocket.DataHandler <- stream.UnhandledMessageWarning{Message: bi.Name + stream.UnhandledMessage +
				string(respRaw)}
		}
	default:
		bi.Websocket.DataHandler <- stream.UnhandledMessageWarning{Message: bi.Name + stream.UnhandledMessage +
			string(respRaw)}
	}
	return nil
}

// TradeDataHandler handles trade data, as functionality is shared between updates and snapshots
func (bi *Bitget) tradeDataHandler(wsResponse WsResponse) ([]trade.Data, error) {
	var trades []WsTradeResponse
	pair, err := pairFromStringHelper(wsResponse.Arg.InstrumentID)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(wsResponse.Data, &trades)
	if err != nil {
		return nil, err
	}
	resp := make([]trade.Data, len(trades))
	for i := range trades {
		resp[i] = trade.Data{
			Timestamp:    trades[i].Timestamp.Time(),
			CurrencyPair: pair,
			AssetType:    asset.Spot,
			Exchange:     bi.Name,
			Price:        trades[i].Price,
			Amount:       trades[i].Size,
			Side:         sideDecoder(trades[i].Side),
			TID:          strconv.FormatInt(trades[i].TradeID, 10),
		}
	}
	return resp, nil
}

// OrderbookDataHandler handles orderbook data, as functionality is shared between updates and snapshots
func (bi *Bitget) orderbookDataHandler(wsResponse WsResponse) error {
	var ob []WsOrderBookResponse
	pair, err := pairFromStringHelper(wsResponse.Arg.InstrumentID)
	if err != nil {
		return err
	}
	err = json.Unmarshal(wsResponse.Data, &ob)
	if err != nil {
		return err
	}
	if len(ob) == 0 {
		return errReturnEmpty
	}
	bids, err := trancheConstructor(ob[0].Bids)
	if err != nil {
		return err
	}
	asks, err := trancheConstructor(ob[0].Asks)
	if err != nil {
		return err
	}
	if wsResponse.Action[0] == 's' {
		orderbook := orderbook.Base{
			Pair:                   pair,
			Asset:                  asset.Spot,
			Bids:                   bids,
			Asks:                   asks,
			LastUpdated:            wsResponse.Timestamp.Time(),
			Exchange:               bi.Name,
			VerifyOrderbook:        bi.CanVerifyOrderbook,
			ChecksumStringRequired: true,
		}
		err = bi.Websocket.Orderbook.LoadSnapshot(&orderbook)
		if err != nil {
			return err
		}
	} else {
		update := orderbook.Update{
			Bids:       bids,
			Asks:       asks,
			Pair:       pair,
			UpdateTime: wsResponse.Timestamp.Time(),
			Asset:      asset.Spot,
			Checksum:   uint32(ob[0].Checksum),
		}
		err = bi.Websocket.Orderbook.Update(&update)
		if err != nil {
			return err
		}
	}
	return nil
}

// TrancheConstructor turns the exchange's orderbook data into a standardised format for the engine
func trancheConstructor(data [][2]string) ([]orderbook.Tranche, error) {
	resp := make([]orderbook.Tranche, len(data))
	var err error
	for i := range data {
		resp[i] = orderbook.Tranche{
			StrPrice:  data[i][0],
			StrAmount: data[i][1],
		}
		resp[i].Price, err = strconv.ParseFloat(data[i][0], 64)
		if err != nil {
			return nil, err
		}
		resp[i].Amount, err = strconv.ParseFloat(data[i][1], 64)
		if err != nil {
			return nil, err
		}
	}
	return resp, nil
}

func (bi *Bitget) CalculateUpdateOrderbookChecksum(orderbookData *orderbook.Base, checksumVal uint32) error {
	var builder strings.Builder
	for i := 0; i < 25; i++ {
		if len(orderbookData.Bids) > i {
			builder.WriteString(orderbookData.Bids[i].StrPrice + ":" + orderbookData.Bids[i].StrAmount + ":")
		}
		if len(orderbookData.Asks) > i {
			builder.WriteString(orderbookData.Asks[i].StrPrice + ":" + orderbookData.Asks[i].StrAmount + ":")
		}
	}
	check := builder.String()
	if check != "" {
		check = check[:len(check)-1]
	}
	if crc32.ChecksumIEEE([]byte(check)) != checksumVal {
		return errInvalidChecksum
	}
	return nil
}

// GenerateDefaultSubscriptions generates default subscriptions
func (bi *Bitget) generateDefaultSubscriptions() (subscription.List, error) {
	enabledPairs, err := bi.GetEnabledPairs(asset.Spot)
	if err != nil {
		return nil, err
	}
	subs := make(subscription.List, 0, len(defaultSubscriptions))
	for _, sub := range defaultSubscriptions {
		if sub.Enabled {
			subs = append(subs, sub)
		}
	}
	subs = subs[:len(subs):len(subs)]
	for i := range subs {
		subs[i].Pairs = enabledPairs
		subs[i].Channel = subscriptionNames[subs[i].Channel]
	}
	return subs, nil
}

// Subscribe sends a websocket message to receive data from the channel
func (bi *Bitget) Subscribe(subs subscription.List) error {
	return bi.websocketMessage(subs, "subscribe")
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (bi *Bitget) Unsubscribe(subs subscription.List) error {
	return bi.websocketMessage(subs, "unsubscribe")
}

// ReqSplitter splits a request into multiple requests to avoid going over the byte limit
func reqSplitter(req *WsRequest) []WsRequest {
	cap := (len(req.Arguments) / 47) + 1
	reqs := make([]WsRequest, cap)
	for i := 0; i < cap; i++ {
		reqs[i].Operation = req.Operation
		if i == cap-1 {
			reqs[i].Arguments = req.Arguments[i*47:]
			break
		}
		reqs[i].Arguments = req.Arguments[i*47 : (i+1)*47]
	}
	return reqs
}

// ReqBuilder builds a request in the manner the exchange expects
func (bi *Bitget) reqBuilder(req *WsRequest, sub *subscription.Subscription) {
	for i := range sub.Pairs {
		form, _ := bi.GetPairFormat(asset.Spot, true)
		sub.Pairs[i] = sub.Pairs[i].Format(form)
		req.Arguments = append(req.Arguments, WsArgument{
			Channel:        sub.Channel,
			InstrumentType: strings.ToUpper(sub.Asset.String()),
			InstrumentID:   sub.Pairs[i].String(),
		})
	}
}

// websocketMessage sends a websocket message
func (bi *Bitget) websocketMessage(subs subscription.List, op string) error {
	unauthBase := &WsRequest{
		Operation: op,
	}
	authBase := &WsRequest{
		Operation: op,
	}
	for _, s := range subs {
		switch s.Channel {
		case bitgetAccount, bitgetFillChannel:
			authBase.Arguments = append(authBase.Arguments, WsArgument{
				Channel:        s.Channel,
				InstrumentType: strings.ToUpper(s.Asset.String()),
				Coin:           "default",
			})
		}
		if s.Authenticated {
			bi.reqBuilder(authBase, s)
		} else {
			bi.reqBuilder(unauthBase, s)
		}
	}
	unauthReq := reqSplitter(unauthBase)
	authReq := reqSplitter(authBase)
	for i := range unauthReq {
		if len(unauthReq[i].Arguments) != 0 {
			err := bi.Websocket.Conn.SendJSONMessage(unauthReq[i])
			if err != nil {
				return err
			}
		}
	}
	for i := range authReq {
		if len(authReq[i].Arguments) != 0 {
			err := bi.Websocket.AuthConn.SendJSONMessage(authReq[i])
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// GetSubscriptionTemplate returns a subscription channel template
func (bi *Bitget) GetSubscriptionTemplate(_ *subscription.Subscription) (*template.Template, error) {
	return template.New("master.tmpl").Funcs(template.FuncMap{"channelName": channelName}).Parse(subTplText)
}

func channelName(s *subscription.Subscription) string {
	if n, ok := subscriptionNames[s.Channel]; ok {
		return n
	}
	// Replace error with subscription.ErrNotSupported after merge
	panic(fmt.Errorf("error not supported: %s", s.Channel))
}

const subTplText = `
{{ range $asset, $pairs := $.AssetPairs }}
	{{- channelName $.S -}}
	{{- $.AssetSeparator }}
{{- end }}
`
