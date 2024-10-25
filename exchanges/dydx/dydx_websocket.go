package dydx

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
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
	// channels
	accountsChannel  = "v3_accounts"
	orderbookChannel = "v3_orderbook"
	tradesChannel    = "v3_trades"
	marketsChannel   = "v3_markets"
)

var defaultSubscriptions = []string{
	orderbookChannel,
	tradesChannel,
	marketsChannel,
}

// WsConnect connect to dydx websocket server.
func (dy *DYDX) WsConnect() error {
	if !dy.Websocket.IsEnabled() || !dy.IsEnabled() {
		return stream.ErrWebsocketNotEnabled
	}
	var dialer websocket.Dialer
	err := dy.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return err
	}
	if dy.Verbose {
		log.Debugf(log.ExchangeSys, "%s Connected to Websocket.\n", dy.Name)
	}
	dy.Websocket.Wg.Add(1)
	go dy.wsReadData()
	return nil
}

var orderbookSnapshootCurrencies map[string]bool

func (dy *DYDX) wsReadData() {
	orderbookSnapshootCurrencies = map[string]bool{}
	defer dy.Websocket.Wg.Done()
	for {
		resp := dy.Websocket.Conn.ReadMessage()
		if resp.Raw == nil {
			return
		}
		err := dy.wsHandleData(resp.Raw)
		if err != nil {
			dy.Websocket.DataHandler <- err
		}
	}
}

var websocketMarketTickerPushDataCounter = false

func (dy *DYDX) wsHandleData(respRaw []byte) error {
	var resp WsResponse
	err := json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	if resp.Type == "connected" {
		return nil
	}
	switch resp.Channel {
	case accountsChannel:
		switch resp.Type {
		case "subscribe":
			var resp AccountSubscriptionResponse
			err = json.Unmarshal(respRaw, &resp)
			if err != nil {
				return err
			}
			err = dy.processOrders(resp.Contents.Orders)
			if err != nil {
				return err
			}
			dy.Websocket.DataHandler <- resp.Transfers
			dy.Websocket.DataHandler <- resp.FundingPayments
		case "channel_data":
			var resp AccountChannelData
			err = json.Unmarshal(respRaw, &resp)
			if err != nil {
				return err
			}
			err = dy.processOrders(resp.Contents.Orders)
			if err != nil {
				return err
			}
			dy.Websocket.DataHandler <- resp.Contents.Accounts
			dy.Websocket.DataHandler <- resp.Contents.Positions
			dy.Websocket.DataHandler <- resp.Contents.Fills
		}
		return nil
	case orderbookChannel:
		pair, err := currency.NewPairFromString(resp.ID)
		if err != nil {
			return err
		}
		if resp.MessageID == 1 || !orderbookSnapshootCurrencies[resp.ID] {
			orderbookSnapshootCurrencies[resp.ID] = true
			var market MarketOrderbook
			err = json.Unmarshal(resp.Contents, &market)
			if err != nil {
				return err
			}
			return dy.Websocket.Orderbook.LoadSnapshot(&orderbook.Base{
				Asset:       asset.Spot,
				Asks:        market.Asks.generateOrderbookItem(),
				Bids:        market.Bids.generateOrderbookItem(),
				Pair:        pair,
				Exchange:    dy.Name,
				LastUpdated: time.Now(),
			})
		}
		var market MarketOrderbookUpdate
		err = json.Unmarshal(resp.Contents, &market)
		if err != nil {
			return err
		}
		update := &orderbook.Update{
			Asset:      asset.Spot,
			Pair:       pair,
			UpdateTime: time.Now(),
		}
		update.Asks, err = market.Asks.generateOrderbookItem()
		if err != nil {
			return err
		}
		update.Bids, err = market.Bids.generateOrderbookItem()
		if err != nil {
			return err
		}
		return dy.Websocket.Orderbook.Update(update)
	case tradesChannel:
		var myTrades MarketTrades
		err := json.Unmarshal(resp.Contents, &myTrades)
		if err != nil {
			return err
		}
		pair, err := currency.NewPairFromString(resp.ID)
		if err != nil {
			return err
		}
		trades := make([]trade.Data, len(myTrades.Trades))
		for i := range myTrades.Trades {
			side, err := order.StringToOrderSide(myTrades.Trades[i].Side)
			if err != nil {
				return err
			}
			trades[i] = trade.Data{
				AssetType:    asset.Spot,
				CurrencyPair: pair,
				Exchange:     dy.Name,
				Side:         side,
				Timestamp:    myTrades.Trades[i].CreatedAt,
				Amount:       myTrades.Trades[i].Size.Float64(),
				Price:        myTrades.Trades[i].Price.Float64(),
			}
		}
		return trade.AddTradesToBuffer(dy.Name, trades...)
	case marketsChannel:
		var market InstrumentDatas
		if !websocketMarketTickerPushDataCounter {
			err := json.Unmarshal(resp.Contents, &market)
			if err != nil {
				return err
			}
			websocketMarketTickerPushDataCounter = true
		} else {
			err := json.Unmarshal(resp.Contents, &market.Markets)
			if err != nil {
				return err
			}
		}
		pairs, err := dy.GetEnabledPairs(asset.Spot)
		if err != nil {
			return err
		}
		for key := range market.Markets {
			pair, err := currency.NewPairFromString(key)
			if err != nil {
				return err
			}
			if !pairs.Contains(pair, true) {
				continue
			}
			var tickerPrice *ticker.Price
			if market.Markets[key].Volume24H != 0 {
				tickerPrice = &ticker.Price{
					ExchangeName: dy.Name,
					Last:         market.Markets[key].IndexPrice.Float64(),
					Pair:         pair,
					AssetType:    asset.Spot,
					Open:         market.Markets[key].PriceChange24H.Float64(),
					QuoteVolume:  market.Markets[key].Volume24H.Float64(),
				}
				if market.Markets[key].OraclePrice != 0 {
					tickerPrice.Volume = market.Markets[key].Volume24H.Float64() / market.Markets[key].OraclePrice.Float64()
				}
			} else {
				tickerPrice, err = ticker.GetTicker(dy.Name, pair, asset.Spot)
				if err != nil {
					return err
				}
				tickerPrice.Last = market.Markets[key].IndexPrice.Float64()
			}
			dy.Websocket.DataHandler <- tickerPrice
		}
	default:
		dy.Websocket.DataHandler <- stream.UnhandledMessageWarning{Message: dy.Name + stream.UnhandledMessage + string(respRaw)}
		return nil
	}
	return nil
}

// processOrders processes incoming orders with push data.
func (dy *DYDX) processOrders(orders []Order) error {
	orderDetails := make([]order.Detail, len(orders))
	for x := range orders {
		orderType, err := order.StringToOrderType(orders[x].Type)
		if err != nil {
			return err
		}
		orderSide, err := order.StringToOrderSide(orders[x].Side)
		if err != nil {
			return err
		}
		orderStatus, err := order.StringToOrderStatus(orders[x].Status)
		if err != nil {
			return err
		}
		cp, err := currency.NewPairFromString(orders[x].Market)
		if err != nil {
			return err
		}
		orderDetails[x] = order.Detail{
			Price:           orders[x].Price.Float64(),
			Amount:          orders[x].Size.Float64(),
			ExecutedAmount:  orders[x].Size.Float64() - orders[x].RemainingSize.Float64(),
			Fee:             orders[x].LimitFee.Float64(),
			Exchange:        dy.Name,
			OrderID:         orders[x].ID,
			Type:            orderType,
			Status:          orderStatus,
			Side:            orderSide,
			AssetType:       asset.Spot,
			Date:            orders[x].CreatedAt,
			Pair:            cp,
			RemainingAmount: orders[x].RemainingSize.Float64(),
		}
	}
	dy.Websocket.DataHandler <- orderDetails
	return nil
}

// GenerateDefaultSubscriptions Adds default subscriptions to websocket to be handled by ManageSubscriptions
func (dy *DYDX) GenerateDefaultSubscriptions() (subscription.List, error) {
	channels := defaultSubscriptions
	if dy.Websocket.CanUseAuthenticatedEndpoints() {
		channels = append(channels, accountsChannel)
	}
	subscriptions := subscription.List{}
	enabledPairs, err := dy.GetEnabledPairs(asset.Spot)
	if err != nil {
		return nil, err
	}
	for x := range channels {
		if channels[x] == accountsChannel || channels[x] == marketsChannel {
			subscriptions = append(subscriptions, &subscription.Subscription{
				Channel: channels[x],
			})
			continue
		}
		subscriptions = append(subscriptions, &subscription.Subscription{
			Channel: channels[x],
			Pairs:   enabledPairs,
		})
	}
	return subscriptions, nil
}

func (dy *DYDX) generateSubscriptionPayload(subscriptions subscription.List, operation string) ([]WsInput, error) {
	payloads := make([]WsInput, len(subscriptions))
	for x := range subscriptions {
		for a := range subscriptions[x].Pairs {
			payloads[x] = WsInput{
				Type:    operation,
				Channel: subscriptions[x].Channel,
				ID:      subscriptions[x].Pairs[a].String(),
			}
		}

		if subscriptions[x].Channel == accountsChannel {
			payloads[x].ID = ""
			creds, err := dy.GetCredentials(context.Background())
			if err != nil {
				return nil, err
			}
			payloads[x].AccountNumber = "0"
			timestamp := time.Now().UTC().Format(timeFormat)
			message := fmt.Sprintf("%s%s%s%s", timestamp, http.MethodGet, "/ws/accounts", "")
			secret, _ := base64.URLEncoding.DecodeString(creds.Secret)
			h := hmac.New(sha256.New, secret)
			h.Write([]byte(message))

			payloads[x].APIKey = creds.Key
			payloads[x].Passphrase = creds.PEMKey
			payloads[x].Signature = base64.URLEncoding.EncodeToString(h.Sum(nil))
			payloads[x].Timestamp = timestamp
		}
	}
	return payloads, nil
}

func (dy *DYDX) handleSubscriptions(subscriptions subscription.List, operation string) error {
	payloads, err := dy.generateSubscriptionPayload(subscriptions, operation)
	if err != nil {
		return err
	}
	var errs error
	for x := range payloads {
		err = dy.Websocket.Conn.SendJSONMessage(context.Background(), request.UnAuth, payloads[x])
		if err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		dy.Websocket.AddSuccessfulSubscriptions(subscriptions[x])
	}
	return errs
}

// Subscribe sends a subscriptions requests through the websocket connection.
func (dy *DYDX) Subscribe(subscriptions subscription.List) error {
	return dy.handleSubscriptions(subscriptions, "subscribe")
}

// Unsubscribe sends unsubscription to channels through the websocket connection.
func (dy *DYDX) Unsubscribe(subscriptions subscription.List) error {
	return dy.handleSubscriptions(subscriptions, "unsubscribe")
}
