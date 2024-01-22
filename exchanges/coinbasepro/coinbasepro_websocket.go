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
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

const (
	coinbaseproWebsocketURL = "wss://advanced-trade-ws.coinbase.com"
)

// WsConnect initiates a websocket connection
func (c *CoinbasePro) WsConnect() error {
	if !c.Websocket.IsEnabled() || !c.IsEnabled() {
		return errors.New(stream.WebsocketNotEnabled)
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

	for {
		resp := c.Websocket.Conn.ReadMessage()
		if resp.Raw == nil {
			return
		}
		err := c.wsHandleData(resp.Raw)
		if err != nil {
			c.Websocket.DataHandler <- err
		}
	}
}

func (c *CoinbasePro) wsHandleData(respRaw []byte) error {
	// fmt.Println("WHADDUP:", string(respRaw))
	genData := wsGen{}
	err := json.Unmarshal(respRaw, &genData)
	if err != nil {
		return err
	}

	if genData.Channel == "subscriptions" || genData.Channel == "heartbeats" {
		return nil
	}

	// fmt.Printf("=== OH NOO LOOK AT THIS DATA WE HAVE TO DEAL WITH: %s ===\n", genData.Events)

	// data, _, _, err := jsonparser.Get(respRaw, "events")
	// if err != nil {
	// 	return err
	// }
	// specData := []WebsocketTickerHolder{}
	// err = json.Unmarshal(data, &specData)
	// if err != nil {
	// 	return err
	// }
	// fmt.Printf("===== AWESOME, WE'VE GOT THE GOOD DATA: %v =====\n", specData)

	if len(genData.Events) == 0 {
		return errNoEventsWS
	}

	data, _, _, err := jsonparser.Get(respRaw, "events")
	if err != nil {
		return err
	}
	// fmt.Printf("==== WEEWOO WE'VE GOT THE NASTY DATA: %s ====\n", data)

	switch genData.Channel {
	case "status":
		wsStatus := []WebsocketProductHolder{}

		err = json.Unmarshal(data, &wsStatus)
		if err != nil {
			return err
		}

	case "error":
		c.Websocket.DataHandler <- errors.New(string(respRaw))
	case "ticker", "ticker_batch":
		wsTicker := []WebsocketTickerHolder{}

		err = json.Unmarshal(data, &wsTicker)
		if err != nil {
			return err
		}

		for i := range wsTicker {
			for j := range wsTicker[i].Tickers {
				c.Websocket.DataHandler <- &ticker.Price{
					LastUpdated:  genData.Timestamp,
					Pair:         wsTicker[i].Tickers[j].ProductID,
					AssetType:    asset.Spot,
					ExchangeName: c.Name,
					High:         wsTicker[i].Tickers[j].High24H,
					Low:          wsTicker[i].Tickers[j].Low24H,
					Last:         wsTicker[i].Tickers[j].Price,
					Volume:       wsTicker[i].Tickers[j].Volume24H,
				}
			}
		}
		// fmt.Printf("=== WOOT, IT WORKED ===\n")
	case "candles":
		wsCandles := []WebsocketCandleHolder{}

		err = json.Unmarshal(data, &wsCandles)
		if err != nil {
			return err
		}

		for i := range wsCandles {
			for j := range wsCandles[i].Candles {
				c.Websocket.DataHandler <- &stream.KlineData{
					Timestamp:  genData.Timestamp,
					Pair:       wsCandles[i].Candles[j].ProductID,
					AssetType:  asset.Spot,
					Exchange:   c.Name,
					StartTime:  wsCandles[i].Candles[j].Start.Time(),
					OpenPrice:  wsCandles[i].Candles[j].Open,
					ClosePrice: wsCandles[i].Candles[j].Close,
					HighPrice:  wsCandles[i].Candles[j].High,
					LowPrice:   wsCandles[i].Candles[j].Low,
					Volume:     wsCandles[i].Candles[j].Volume,
				}
			}
		}
	// fmt.Print("=== RECEIVED AND PROCESSED ===\n")
	case "market_trades":
		wsTrades := []WebsocketMarketTradeHolder{}

		err = json.Unmarshal(data, &wsTrades)
		if err != nil {
			return err
		}

		for i := range wsTrades {
			for j := range wsTrades[i].Trades {
				c.Websocket.DataHandler <- &trade.Data{
					TID:          wsTrades[i].Trades[j].TradeID,
					Exchange:     c.Name,
					CurrencyPair: wsTrades[i].Trades[j].ProductID,
					AssetType:    asset.Spot,
					Side:         wsTrades[i].Trades[j].Side,
					Price:        wsTrades[i].Trades[j].Price,
					Amount:       wsTrades[i].Trades[j].Size,
					Timestamp:    wsTrades[i].Trades[j].Time,
				}
			}
		}
		// fmt.Print("=== RECEIVED AND PROCESSED ===\n")
	case "l2_data":
		var wsL2 []WebsocketOrderbookDataHolder
		err := json.Unmarshal(data, &wsL2)
		if err != nil {
			return err
		}

		for i := range wsL2 {
			// fmt.Printf("======== DATA THAT WE JUST HIT: %v ========\n", wsL2[i])
			switch wsL2[i].Type {
			case "snapshot":
				err = c.ProcessSnapshot(wsL2[i], genData.Timestamp)
			case "update":
				err = c.ProcessUpdate(wsL2[i], genData.Timestamp)
			default:
				err = errors.Errorf(errUnknownL2DataType, wsL2[i].Type)
			}
			if err != nil {
				return err
			}

		}
	case "user":
		var wsUser []WebsocketOrderDataHolder
		err := json.Unmarshal(data, &wsUser)
		if err != nil {
			return err
		}

		for i := range wsUser {
			for j := range wsUser[i].Orders {
				var oType order.Type
				oType, err = order.StringToOrderType(wsUser[i].Orders[j].OrderType)
				if err != nil {
					c.Websocket.DataHandler <- stream.UnhandledMessageWarning{Message: c.Name + stream.UnhandledMessage + string(respRaw)}
					continue
				}

				var oSide order.Side
				oSide, err = order.StringToOrderSide(wsUser[i].Orders[j].OrderSide)
				if err != nil {
					c.Websocket.DataHandler <- stream.UnhandledMessageWarning{Message: c.Name + stream.UnhandledMessage + string(respRaw)}
					continue
				}

				var oStatus order.Status
				oStatus, err = statusToStandardStatus(wsUser[i].Orders[j].Status)
				if err != nil {
					c.Websocket.DataHandler <- stream.UnhandledMessageWarning{Message: c.Name + stream.UnhandledMessage + string(respRaw)}
					continue
				}

				c.Websocket.DataHandler <- &order.Detail{
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
				}
			}
		}

	default:
		c.Websocket.DataHandler <- stream.UnhandledMessageWarning{Message: c.Name + stream.UnhandledMessage + string(respRaw)}
		return nil
	}
	return nil
}

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
		return order.UnknownStatus, fmt.Errorf("%s not recognised as status type", stat)
	}
}

// ProcessSnapshot processes the initial orderbook snap shot
func (c *CoinbasePro) ProcessSnapshot(snapshot WebsocketOrderbookDataHolder, timestamp time.Time) error {
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
func (c *CoinbasePro) ProcessUpdate(update WebsocketOrderbookDataHolder, timestamp time.Time) error {
	// fmt.Printf("====== DATA THAT WE'RE USING TO UPDATE: %v ======\n", update)

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

	// fmt.Printf("===== WE'RE ABOUT TO UPDATE THE ORDERBOOK: %v =====\n", obU)

	return c.Websocket.Orderbook.Update(&obU)
}

// processBidAskArray is a helper function that turns WebsocketOrderbookDataHolder into arrays
// of bids and asks
func processBidAskArray(data WebsocketOrderbookDataHolder) ([]orderbook.Item, []orderbook.Item, error) {
	var bids, asks []orderbook.Item
	for i := range data.Changes {
		switch data.Changes[i].Side {
		case "bid":
			bids = append(bids, orderbook.Item{
				Price:  data.Changes[i].PriceLevel,
				Amount: data.Changes[i].NewQuantity,
			})
		case "offer":
			asks = append(asks, orderbook.Item{
				Price:  data.Changes[i].PriceLevel,
				Amount: data.Changes[i].NewQuantity,
			})
		default:
			return nil, nil, errors.Errorf(errUnknownSide, data.Changes[i].Side)
		}
	}
	return bids, asks, nil
}

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be handled by ManageSubscriptions()
func (c *CoinbasePro) GenerateDefaultSubscriptions() ([]stream.ChannelSubscription, error) {
	var channels = []string{
		"heartbeats",
		"ticker",
		"ticker_batch",
		"candles",
		"market_trades",
		"level2",
		"user",
	}
	enabledCurrencies, err := c.GetEnabledPairs(asset.Spot)
	if err != nil {
		return nil, err
	}
	var subscriptions []stream.ChannelSubscription
	for i := range channels {
		for j := range enabledCurrencies {
			fPair, err := c.FormatExchangeCurrency(enabledCurrencies[j],
				asset.Spot)
			if err != nil {
				return nil, err
			}
			subscriptions = append(subscriptions, stream.ChannelSubscription{
				Channel:  channels[i],
				Currency: fPair,
				Asset:    asset.Spot,
			})
		}
	}
	return subscriptions, nil
}

func (c *CoinbasePro) sendRequest(msgType, channel string, productID currency.Pair, rLim RateLimit) error {
	creds, err := c.GetCredentials(context.Background())
	if err != nil {
		return err
	}

	n := strconv.FormatInt(time.Now().Unix(), 10)

	message := n + channel + productID.String()

	hmac, err := crypto.GetHMAC(crypto.HashSHA256,
		[]byte(message),
		[]byte(creds.Secret))
	if err != nil {
		return err
	}

	// jwt, err := c.GetJWT(context.Background(), "")
	// if err != nil {
	// 	return err
	// }

	req := WebsocketSubscribe{
		Type:       msgType,
		ProductIDs: []string{productID.String()},
		Channel:    channel,
		Signature:  hex.EncodeToString(hmac),
		// JWT:       jwt,
		Key:       creds.Key,
		Timestamp: n,
	}

	reqMarshal, _ := json.Marshal(req)

	fmt.Print(string(reqMarshal))

	err = rLim.Limit(context.Background(), WSRate)
	if err != nil {
		return err
	}
	err = c.Websocket.Conn.SendJSONMessage(req)
	if err != nil {
		return err
	}
	return nil
}

// Subscribe sends a websocket message to receive data from the channel
func (c *CoinbasePro) Subscribe(channelsToSubscribe []stream.ChannelSubscription) error {

	fmt.Printf("SUBSCRIBE: %v\n", channelsToSubscribe)
	// 	var creds *account.Credentials
	// 	var err error
	// 	if c.IsWebsocketAuthenticationSupported() {
	// 		creds, err = c.GetCredentials(context.TODO())
	// 		if err != nil {
	// 			return err
	// 		}
	// 	}

	// 	subscribe := WebsocketSubscribe{
	// 		Type: "subscribe",
	// 	}

	// subscriptions:
	// 	for i := range channelsToSubscribe {
	// 		p := channelsToSubscribe[i].Currency.String()
	// 		if !common.StringDataCompare(subscribe.ProductIDs, p) && p != "" {
	// 			subscribe.ProductIDs = append(subscribe.ProductIDs, p)
	// 		}

	// 		if subscribe.Channel == channelsToSubscribe[i].Channel {
	// 			continue subscriptions
	// 		}

	// 		subscribe.Channel = channelsToSubscribe[i].Channel

	// 		if (channelsToSubscribe[i].Channel == "user" ||
	// 			channelsToSubscribe[i].Channel == "full") && creds != nil {
	// 			n := strconv.FormatInt(time.Now().Unix(), 10)
	// 			message := n + http.MethodGet + "/users/self/verify"
	// 			var hmac []byte
	// 			hmac, err = crypto.GetHMAC(crypto.HashSHA256,
	// 				[]byte(message),
	// 				[]byte(creds.Secret))
	// 			if err != nil {
	// 				return err
	// 			}
	// 			subscribe.Signature = crypto.Base64Encode(hmac)
	// 			subscribe.Key = creds.Key
	// 			subscribe.Timestamp = n
	// 		}
	// 	}

	var rLim RateLimit

	rLim.RateLimWS = request.NewRateLimit(coinbaseWSInterval, coinbaseWSRate)

	for i := range channelsToSubscribe {
		err := c.sendRequest("subscribe", channelsToSubscribe[i].Channel, channelsToSubscribe[i].Currency, rLim)
		if err != nil {
			return err
		}
	}

	c.Websocket.AddSuccessfulSubscriptions(channelsToSubscribe...)
	return nil
}

// Unsubscribe sends a websocket message to stop receiving data from the channel
func (c *CoinbasePro) Unsubscribe(channelsToUnsubscribe []stream.ChannelSubscription) error {
	unsubscribe := WebsocketSubscribe{
		Type: "unsubscribe",
	}

	for i := range channelsToUnsubscribe {
		p := channelsToUnsubscribe[i].Currency.String()
		if !common.StringDataCompare(unsubscribe.ProductIDs, p) && p != "" {
			unsubscribe.ProductIDs = append(unsubscribe.ProductIDs, p)
		}

		if unsubscribe.Channel == channelsToUnsubscribe[i].Channel {
			unsubscribe.Channel = channelsToUnsubscribe[i].Channel

		}

	}
	var rLim RateLimit

	rLim.RateLimWS = request.NewRateLimit(coinbaseWSInterval, coinbaseWSRate)

	err := rLim.Limit(context.Background(), WSRate)
	if err != nil {
		return err
	}

	err = c.Websocket.Conn.SendJSONMessage(unsubscribe)
	if err != nil {
		return err
	}

	c.Websocket.RemoveSubscriptions(channelsToUnsubscribe...)
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
	headJson, err := json.Marshal(head)
	if err != nil {
		return "", err
	}
	headEncode := Base64URLEncode(headJson)

	c.jwtLastRegen = time.Now()

	body := map[string]interface{}{"iss": "coinbase-cloud", "nbf": time.Now().Unix(),
		"exp": time.Now().Add(time.Minute * 2).Unix(), "sub": creds.ClientID, "aud": "retail_rest_api_proxy"}
	if uri != "" {
		body["uri"] = uri
	}
	bodyJson, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	bodyEncode := Base64URLEncode(bodyJson)

	hash := sha256.Sum256([]byte(headEncode + "." + bodyEncode))

	sig, err := ecdsa.SignASN1(rand.Reader, key, hash[:])
	if err != nil {
		return "", err
	}
	sigEncode := Base64URLEncode(sig)

	return headEncode + "." + bodyEncode + "." + sigEncode, nil
}

// Base64URLEncode is a helper function that does some tweaks to standard Base64 encoding, in a way
// which JWT requires
func Base64URLEncode(b []byte) string {
	s := crypto.Base64Encode(b)
	s = strings.Split(s, "=")[0]
	s = strings.ReplaceAll(s, "+", "-")
	s = strings.ReplaceAll(s, "/", "_")
	return s
}
