package bitget

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	bitgetPublicWSURL  = "wss://ws.bitget.com/v2/ws/public"
	bitgetPrivateWSURL = "wss://ws.bitget.com/v2/ws/private"
)

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
		case bitgetTickerChannel:
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
		case bitgetAccountChannel:
			var account []WsAccountResponse
			err := json.Unmarshal(wsResponse.Data, &account)
			if err != nil {
				return err
			}
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
		case bitgetAccountChannel:
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

func (bi *Bitget) generateDefaultSubscriptions() (subscription.List, error) {
	channels := []string{bitgetAccountChannel}
	// channels := []string{bitgetTickerChannel}
	enabledPairs, err := bi.GetEnabledPairs(asset.Spot)
	if err != nil {
		return nil, err
	}
	var subscriptions subscription.List
	for i := range channels {
		subscriptions = append(subscriptions, &subscription.Subscription{
			Channel: channels[i],
			Pairs:   enabledPairs,
			Asset:   asset.Spot,
		})
	}
	return subscriptions, nil
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
		case bitgetTickerChannel, bitgetCandleDailyChannel:
			bi.reqBuilder(unauthBase, s)
		case bitgetAccountChannel:
			authBase.Arguments = append(authBase.Arguments, WsArgument{
				Channel:        s.Channel,
				InstrumentType: strings.ToUpper(s.Asset.String()),
				Coin:           "default",
			})
		default:
			bi.reqBuilder(authBase, s)
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

// SendJSONMessage sends a JSON message to the connected websocket
