package coinbasepro

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/buger/jsonparser"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

const (
	coinbaseproWebsocketURL = "wss://advanced-trade-ws.coinbase.com"
)

// WsConnect initiates a websocket connection
func (c *CoinbasePro) WsConnect() error {
	if !c.Websocket.IsEnabled() || !c.IsEnabled() {
		return stream.ErrWebsocketNotEnabled
	}
	var dialer websocket.Dialer
	err := c.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	c.Websocket.Wg.Add(1)
	go c.wsReadData()
	return nil
}

// wsReadData receives and passes on websocket messages for processing
func (c *CoinbasePro) wsReadData() {
	defer c.Websocket.Wg.Done()
	var seqCount uint64
	for {
		resp := c.Websocket.Conn.ReadMessage()
		if resp.Raw == nil {
			return
		}
		warn, err := c.wsHandleData(resp.Raw, seqCount)
		if err != nil {
			c.Websocket.DataHandler <- err
		}
		if warn != "" {
			c.Websocket.DataHandler <- warn
			tempStr := strings.SplitN(warn, "Out of order sequence number. Received ", 2)[1]
			tempStr = strings.SplitN(tempStr, ", expected ", 2)[0]
			tempNum, err := strconv.ParseUint(tempStr, 10, 64)
			if err != nil {
				c.Websocket.DataHandler <- err
			} else {
				seqCount = tempNum
			}
		}
		seqCount++
	}
}

// wsHandleData handles all the websocket data coming from the websocket connection
func (c *CoinbasePro) wsHandleData(respRaw []byte, seqCount uint64) (string, error) {
	var warnString string
	seqData, _, _, err := jsonparser.Get(respRaw, "sequence_num")
	if err != nil {
		return warnString, err
	}
	seqNum, err := strconv.ParseUint(string(seqData), 10, 64)
	if err != nil {
		return warnString, err
	}
	if seqNum != seqCount {
		warnString = fmt.Sprintf(warnSequenceIssue, seqNum,
			seqCount)
	}
	channelRaw, _, _, err := jsonparser.Get(respRaw, "channel")
	if err != nil {
		return warnString, err
	}
	channel := string(channelRaw)
	if channel == "subscriptions" || channel == "heartbeats" {
		return warnString, nil
	}
	data, _, _, err := jsonparser.Get(respRaw, "events")
	if err != nil {
		return warnString, err
	}
	switch channel {
	case "status":
		wsStatus := []WebsocketProductHolder{}
		err = json.Unmarshal(data, &wsStatus)
		if err != nil {
			return warnString, err
		}
		c.Websocket.DataHandler <- wsStatus
	case "error":
		c.Websocket.DataHandler <- errors.New(string(respRaw))
	case "ticker", "ticker_batch":
		wsTicker := []WebsocketTickerHolder{}
		err = json.Unmarshal(data, &wsTicker)
		if err != nil {
			return warnString, err
		}
		sliToSend := []ticker.Price{}
		var timestamp time.Time
		timestamp, err = getTimestamp(respRaw)
		if err != nil {
			return warnString, err
		}
		for i := range wsTicker {
			for j := range wsTicker[i].Tickers {
				sliToSend = append(sliToSend, ticker.Price{
					LastUpdated:  timestamp,
					Pair:         wsTicker[i].Tickers[j].ProductID,
					AssetType:    asset.Spot,
					ExchangeName: c.Name,
					High:         wsTicker[i].Tickers[j].High24H,
					Low:          wsTicker[i].Tickers[j].Low24H,
					Last:         wsTicker[i].Tickers[j].Price,
					Volume:       wsTicker[i].Tickers[j].Volume24H,
				})
			}
		}
		c.Websocket.DataHandler <- sliToSend
	case "candles":
		wsCandles := []WebsocketCandleHolder{}
		err = json.Unmarshal(data, &wsCandles)
		if err != nil {
			return warnString, err
		}
		sliToSend := []stream.KlineData{}
		var timestamp time.Time
		timestamp, err = getTimestamp(respRaw)
		if err != nil {
			return warnString, err
		}
		for i := range wsCandles {
			for j := range wsCandles[i].Candles {
				sliToSend = append(sliToSend, stream.KlineData{
					Timestamp:  timestamp,
					Pair:       wsCandles[i].Candles[j].ProductID,
					AssetType:  asset.Spot,
					Exchange:   c.Name,
					StartTime:  wsCandles[i].Candles[j].Start.Time(),
					OpenPrice:  wsCandles[i].Candles[j].Open,
					ClosePrice: wsCandles[i].Candles[j].Close,
					HighPrice:  wsCandles[i].Candles[j].High,
					LowPrice:   wsCandles[i].Candles[j].Low,
					Volume:     wsCandles[i].Candles[j].Volume,
				})
			}
		}
		c.Websocket.DataHandler <- sliToSend
	case "market_trades":
		wsTrades := []WebsocketMarketTradeHolder{}
		err = json.Unmarshal(data, &wsTrades)
		if err != nil {
			return warnString, err
		}
		sliToSend := []trade.Data{}
		for i := range wsTrades {
			for j := range wsTrades[i].Trades {
				sliToSend = append(sliToSend, trade.Data{
					TID:          wsTrades[i].Trades[j].TradeID,
					Exchange:     c.Name,
					CurrencyPair: wsTrades[i].Trades[j].ProductID,
					AssetType:    asset.Spot,
					Side:         wsTrades[i].Trades[j].Side,
					Price:        wsTrades[i].Trades[j].Price,
					Amount:       wsTrades[i].Trades[j].Size,
					Timestamp:    wsTrades[i].Trades[j].Time,
				})
			}
		}
		c.Websocket.DataHandler <- sliToSend
	case "l2_data":
		var wsL2 []WebsocketOrderbookDataHolder
		err := json.Unmarshal(data, &wsL2)
		if err != nil {
			return warnString, err
		}
		timestamp, err := getTimestamp(respRaw)
		if err != nil {
			return warnString, err
		}
		for i := range wsL2 {
			switch wsL2[i].Type {
			case "snapshot":
				err = c.ProcessSnapshot(&wsL2[i], timestamp)
			case "update":
				err = c.ProcessUpdate(&wsL2[i], timestamp)
			default:
				err = fmt.Errorf("%w %v", errUnknownL2DataType, wsL2[i].Type)
			}
			if err != nil {
				return warnString, err
			}
		}
	case "user":
		var wsUser []WebsocketOrderDataHolder
		err := json.Unmarshal(data, &wsUser)
		if err != nil {
			return warnString, err
		}
		var sliToSend []order.Detail
		for i := range wsUser {
			for j := range wsUser[i].Orders {
				var oType order.Type
				oType, err = order.StringToOrderType(wsUser[i].Orders[j].OrderType)
				if err != nil {
					return warnString, err
				}
				var oSide order.Side
				oSide, err = order.StringToOrderSide(wsUser[i].Orders[j].OrderSide)
				if err != nil {
					return warnString, err
				}
				var oStatus order.Status
				oStatus, err = statusToStandardStatus(wsUser[i].Orders[j].Status)
				if err != nil {
					return warnString, err
				}
				sliToSend = append(sliToSend, order.Detail{
					Price:           wsUser[i].Orders[j].AveragePrice,
					Amount:          wsUser[i].Orders[j].CumulativeQuantity + wsUser[i].Orders[j].LeavesQuantity,
					ExecutedAmount:  wsUser[i].Orders[j].CumulativeQuantity,
					RemainingAmount: wsUser[i].Orders[j].LeavesQuantity,
					Fee:             wsUser[i].Orders[j].TotalFees,
					Exchange:        c.Name,
					OrderID:         wsUser[i].Orders[j].OrderID,
					ClientOrderID:   wsUser[i].Orders[j].ClientOrderID,
					Type:            oType,
					Side:            oSide,
					Status:          oStatus,
					AssetType:       asset.Spot,
					Date:            wsUser[i].Orders[j].CreationTime,
					Pair:            wsUser[i].Orders[j].ProductID,
				})
			}
		}
		c.Websocket.DataHandler <- sliToSend
	default:
		return warnString, errChannelNameUnknown
	}
	return warnString, nil
}

// ProcessSnapshot processes the initial orderbook snap shot
func (c *CoinbasePro) ProcessSnapshot(snapshot *WebsocketOrderbookDataHolder, timestamp time.Time) error {
	bids, asks, err := processBidAskArray(snapshot)
	if err != nil {
		return err
	}
	return c.Websocket.Orderbook.LoadSnapshot(&orderbook.Base{
		Bids:            bids,
		Asks:            asks,
		Exchange:        c.Name,
		Pair:            snapshot.ProductID,
		Asset:           asset.Spot,
		LastUpdated:     timestamp,
		VerifyOrderbook: c.CanVerifyOrderbook,
	})
}

// ProcessUpdate updates the orderbook local cache
func (c *CoinbasePro) ProcessUpdate(update *WebsocketOrderbookDataHolder, timestamp time.Time) error {
	bids, asks, err := processBidAskArray(update)
	if err != nil {
		return err
	}
	obU := orderbook.Update{
		Bids:       bids,
		Asks:       asks,
		Pair:       update.ProductID,
		UpdateTime: timestamp,
		Asset:      asset.Spot,
	}
	return c.Websocket.Orderbook.Update(&obU)
}

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be handled by ManageSubscriptions()
func (c *CoinbasePro) generateSubscriptions() (subscription.List, error) {
	var channels = []string{
		"heartbeats",
		"status",
		"ticker",
		"ticker_batch",
		"candles",
		"market_trades",
		"level2",
		"user",
	}
	enabledPairs, err := c.GetEnabledPairs(asset.Spot)
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
func (c *CoinbasePro) Subscribe(channelsToSubscribe subscription.List) error {
	chanKeys := make(map[string]currency.Pairs)
	for i := range channelsToSubscribe {
		chanKeys[channelsToSubscribe[i].Channel] =
			chanKeys[channelsToSubscribe[i].Channel].Add(channelsToSubscribe[i].Pairs...)
	}
	for s := range chanKeys {
		err := c.sendRequest("subscribe", s, chanKeys[s])
		if err != nil {
			return err
		}
		time.Sleep(time.Millisecond * 10)
	}
	return nil
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (c *CoinbasePro) Unsubscribe(channelsToUnsubscribe subscription.List) error {
	chanKeys := make(map[string]currency.Pairs)
	for i := range channelsToUnsubscribe {
		chanKeys[channelsToUnsubscribe[i].Channel] =
			chanKeys[channelsToUnsubscribe[i].Channel].Add(channelsToUnsubscribe[i].Pairs...)
	}
	for s := range chanKeys {
		err := c.sendRequest("unsubscribe", s, chanKeys[s])
		if err != nil {
			return err
		}
		time.Sleep(time.Millisecond * 10)
	}
	return nil
}

// GetJWT checks if the current JWT is valid, returns it if it is, generates a new one if it isn't
// Also suitable for use in REST requests, by checking for the presence of a URI, and always generating
// a new JWT if one is not provided
func (c *CoinbasePro) GetJWT(ctx context.Context, uri string) (string, error) {
	if c.jwtLastRegen.Add(time.Minute*2).After(time.Now()) && uri != "" {
		return c.jwt, nil
	}
	creds, err := c.GetCredentials(ctx)
	if err != nil {
		return "", err
	}
	block, _ := pem.Decode([]byte(creds.Secret))
	if block == nil {
		return "", errCantDecodePrivKey
	}
	key, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return "", err
	}
	nonce, err := common.GenerateRandomString(64, "1234567890ABCDEF")
	if err != nil {
		return "", err
	}
	head := map[string]interface{}{"kid": creds.ClientID, "typ": "JWT", "alg": "ES256", "nonce": nonce}
	headJSON, err := json.Marshal(head)
	if err != nil {
		return "", err
	}
	headEncode := base64URLEncode(headJSON)
	c.jwtLastRegen = time.Now()
	body := map[string]interface{}{"iss": "coinbase-cloud", "nbf": time.Now().Unix(),
		"exp": time.Now().Add(time.Minute * 2).Unix(), "sub": creds.ClientID, "aud": "retail_rest_api_proxy"}
	if uri != "" {
		body["uri"] = uri
	}
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	bodyEncode := base64URLEncode(bodyJSON)
	hash := sha256.Sum256([]byte(headEncode + "." + bodyEncode))
	sig, err := ecdsa.SignASN1(rand.Reader, key, hash[:])
	if err != nil {
		return "", err
	}
	sigEncode := base64URLEncode(sig)
	return headEncode + "." + bodyEncode + "." + sigEncode, nil
}

// getTimestamp is a helper function which pulls a RFC3339-formatted timestamp from a byte slice
// of JSON data
func getTimestamp(rawData []byte) (time.Time, error) {
	data, _, _, err := jsonparser.Get(rawData, "timestamp")
	if err != nil {
		return time.Time{}, err
	}
	timestamp, err := time.Parse(time.RFC3339, string(data))
	if err != nil {
		return time.Time{}, err
	}
	return timestamp, nil
}

// sendRequest is a helper function which sends a websocket message to the Coinbase server
func (c *CoinbasePro) sendRequest(msgType, channel string, productIDs currency.Pairs) error {
	authenticated := true
	creds, err := c.GetCredentials(context.Background())
	if err != nil {
		if errors.Is(err, exchange.ErrCredentialsAreEmpty) ||
			errors.Is(err, exchange.ErrAuthenticationSupportNotEnabled) {
			authenticated = false
			if channel == "user" {
				return errNoCredsUser
			}
		} else {
			return err
		}
	}
	n := strconv.FormatInt(time.Now().Unix(), 10)
	req := WebsocketRequest{
		Type:       msgType,
		ProductIDs: productIDs.Strings(),
		Channel:    channel,
		Timestamp:  n,
	}
	if authenticated {
		message := n + channel + productIDs.Join()
		var hmac []byte
		hmac, err = crypto.GetHMAC(crypto.HashSHA256,
			[]byte(message),
			[]byte(creds.Secret))
		if err != nil {
			return err
		}
		// TODO: Implement JWT authentication once our REST implementation moves to it, or if there's
		// an exchange-wide reform to enable multiple sets of authentication credentials
		req.Key = creds.Key
		req.Signature = hex.EncodeToString(hmac)
		err = c.InitiateRateLimit(context.Background(), WSAuthRate)
	} else {
		err = c.InitiateRateLimit(context.Background(), WSUnauthRate)
	}
	if err != nil {
		return fmt.Errorf("failed to rate limit websocket request: %w", err)
	}
	return c.Websocket.Conn.SendJSONMessage(req)
}

// processBidAskArray is a helper function that turns WebsocketOrderbookDataHolder into arrays
// of bids and asks
func processBidAskArray(data *WebsocketOrderbookDataHolder) (bids, asks orderbook.Tranches, err error) {
	bids = make(orderbook.Tranches, 0, len(data.Changes))
	asks = make(orderbook.Tranches, 0, len(data.Changes))
	for i := range data.Changes {
		change := orderbook.Tranche{Price: data.Changes[i].PriceLevel, Amount: data.Changes[i].NewQuantity}
		switch data.Changes[i].Side {
		case "bid":
			bids = append(bids, change)
		case "offer":
			asks = append(asks, change)
		default:
			return nil, nil, fmt.Errorf("%w %v", errUnknownSide, data.Changes[i].Side)
		}
	}
	bids.SortBids()
	asks.SortAsks()
	return bids, asks, nil
}

// statusToStandardStatus is a helper function that converts a Coinbase Pro status string to a
// standardised order.Status type
func statusToStandardStatus(stat string) (order.Status, error) {
	switch stat {
	case "received":
		return order.New, nil
	case "open":
		return order.Active, nil
	case "done":
		return order.Filled, nil
	case "match":
		return order.PartiallyFilled, nil
	case "change", "activate":
		return order.Active, nil
	default:
		return order.UnknownStatus, fmt.Errorf("%w %v", errUnrecognisedStatusType, stat)
	}
}

// Base64URLEncode is a helper function that does some tweaks to standard Base64 encoding, in a way
// which JWT requires
func base64URLEncode(b []byte) string {
	s := crypto.Base64Encode(b)
	s = strings.Split(s, "=")[0]
	s = strings.ReplaceAll(s, "+", "-")
	s = strings.ReplaceAll(s, "/", "_")
	return s
}
