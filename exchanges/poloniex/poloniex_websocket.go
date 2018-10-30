package poloniex

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
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
	poloniexWebsocketAddress = "wss://api2.poloniex.com"
	wsAccountNotificationID  = 1000
	wsTickerDataID           = 1002
	ws24HourExchangeVolumeID = 1003
	wsHeartbeat              = 1010
)

var (
	// CurrencyIDMap stores a map of currencies associated with their ID
	CurrencyIDMap map[string]int
)

// WsConnect initiates a websocket connection
func (p *Poloniex) WsConnect() error {
	if !p.Websocket.IsEnabled() || !p.IsEnabled() {
		return errors.New(exchange.WebsocketNotEnabled)
	}

	var dialer websocket.Dialer
	if p.Websocket.GetProxyAddress() != "" {
		proxy, err := url.Parse(p.Websocket.GetProxyAddress())
		if err != nil {
			return err
		}

		dialer.Proxy = http.ProxyURL(proxy)
	}

	var err error
	p.WebsocketConn, _, err = dialer.Dial(p.Websocket.GetWebsocketURL(),
		http.Header{})
	if err != nil {
		return err
	}

	if CurrencyIDMap == nil {
		CurrencyIDMap = make(map[string]int)
		resp, err := p.GetCurrencies()
		if err != nil {
			return err
		}

		for k, v := range resp {
			CurrencyIDMap[k] = v.ID
		}
	}

	go p.WsHandleData()

	return p.WsSubscribe()
}

// WsSubscribe subscribes to the websocket feeds
func (p *Poloniex) WsSubscribe() error {
	tickerJSON, err := common.JSONEncode(WsCommand{
		Command: "subscribe",
		Channel: wsTickerDataID})
	if err != nil {
		return err
	}

	err = p.WebsocketConn.WriteMessage(websocket.TextMessage, tickerJSON)
	if err != nil {
		return err
	}

	pairs := p.GetEnabledPairs(assets.AssetTypeSpot)
	for _, nextPair := range pairs {
		fPair := p.FormatExchangeCurrency(nextPair, assets.AssetTypeSpot)

		orderbookJSON, err := common.JSONEncode(WsCommand{
			Command: "subscribe",
			Channel: fPair.String(),
		})
		if err != nil {
			return err
		}

		err = p.WebsocketConn.WriteMessage(websocket.TextMessage, orderbookJSON)
		if err != nil {
			return err
		}
	}
	return nil
}

// WsReadData reads data from the websocket connection
func (p *Poloniex) WsReadData() (exchange.WebsocketResponse, error) {
	_, resp, err := p.WebsocketConn.ReadMessage()
	if err != nil {
		return exchange.WebsocketResponse{}, err
	}

	p.Websocket.TrafficAlert <- struct{}{}
	return exchange.WebsocketResponse{Raw: resp}, nil
}

func getWSDataType(data interface{}) string {
	subData := data.([]interface{})
	dataType := subData[0].(string)
	return dataType
}

func checkSubscriptionSuccess(data []interface{}) bool {
	return data[1].(float64) == 1
}

// WsHandleData handles data from the websocket connection
func (p *Poloniex) WsHandleData() {
	p.Websocket.Wg.Add(1)

	defer func() {
		err := p.WebsocketConn.Close()
		if err != nil {
			p.Websocket.DataHandler <- fmt.Errorf("poloniex_websocket.go - Unable to to close Websocket connection. Error: %s",
				err)
		}
		p.Websocket.Wg.Done()
	}()

	for {
		select {
		case <-p.Websocket.ShutdownC:
			return

		default:
			resp, err := p.WsReadData()
			if err != nil {
				p.Websocket.DataHandler <- err
				return
			}

			var result interface{}
			err = common.JSONDecode(resp.Raw, &result)
			if err != nil {
				p.Websocket.DataHandler <- err
				continue
			}

			data := result.([]interface{})
			chanID := int(data[0].(float64))

			if len(data) == 2 && chanID != wsHeartbeat {
				if checkSubscriptionSuccess(data) {
					if p.Verbose {
						log.Debugf("poloniex websocket subscribed to channel successfully. %d", chanID)
					}
				} else {
					if p.Verbose {
						log.Debugf("poloniex websocket subscription to channel failed. %d", chanID)
					}
				}
				continue
			}

			switch chanID {
			case wsAccountNotificationID:
			case wsTickerDataID:
				tickerData := data[2].([]interface{})
				var t WsTicker
				currencyPairNum := tickerData[0].(float64)
				var currencyPair currency.Pair
				cp, ok := CurrencyPairID[int(currencyPairNum)]
				if ok {
					currencyPair = currency.NewPairDelimiter(cp, "_")

				}

				t.LastPrice, _ = strconv.ParseFloat(tickerData[1].(string), 64)
				t.LowestAsk, _ = strconv.ParseFloat(tickerData[2].(string), 64)
				t.HighestBid, _ = strconv.ParseFloat(tickerData[3].(string), 64)
				t.PercentageChange, _ = strconv.ParseFloat(tickerData[4].(string), 64)
				t.BaseCurrencyVolume24H, _ = strconv.ParseFloat(tickerData[5].(string), 64)
				t.QuoteCurrencyVolume24H, _ = strconv.ParseFloat(tickerData[6].(string), 64)
				isFrozen := false
				if tickerData[7].(float64) == 1 {
					isFrozen = true
				}
				t.IsFrozen = isFrozen
				t.HighestTradeIn24H, _ = strconv.ParseFloat(tickerData[8].(string), 64)
				t.LowestTradePrice24H, _ = strconv.ParseFloat(tickerData[9].(string), 64)

				p.Websocket.DataHandler <- exchange.TickerData{
					Timestamp:  time.Now(),
					Pair:       currencyPair,
					AssetType:  assets.AssetTypeSpot,
					Exchange:   p.GetName(),
					LowPrice:   t.LowestAsk,
					HighPrice:  t.HighestBid,
					ClosePrice: t.LastPrice,
					Quantity:   t.QuoteCurrencyVolume24H,
				}
			case ws24HourExchangeVolumeID:
			case wsHeartbeat:
			default:
				if len(data) > 2 {
					subData := data[2].([]interface{})

					for x := range subData {
						dataL2 := subData[x]
						dataL3 := dataL2.([]interface{})

						switch getWSDataType(dataL2) {
						case "i":
							dataL3map := dataL3[1].(map[string]interface{})
							currencyPair, ok := dataL3map["currencyPair"].(string)
							if !ok {
								p.Websocket.DataHandler <- errors.New("poloniex.go error - could not find currency pair in map")
								continue
							}

							orderbookData, ok := dataL3map["orderBook"].([]interface{})
							if !ok {
								p.Websocket.DataHandler <- errors.New("poloniex.go error - could not find orderbook data in map")
								continue
							}

							err := p.WsProcessOrderbookSnapshot(orderbookData, currencyPair)
							if err != nil {
								p.Websocket.DataHandler <- err
								continue
							}

							p.Websocket.DataHandler <- exchange.WebsocketOrderbookUpdate{
								Exchange: p.GetName(),
								Asset:    assets.AssetTypeSpot,
								Pair:     currency.NewPairFromString(currencyPair),
							}
						case "o":
							currencyPair := CurrencyPairID[chanID]
							err := p.WsProcessOrderbookUpdate(dataL3, currencyPair)
							if err != nil {
								p.Websocket.DataHandler <- err
								continue
							}

							p.Websocket.DataHandler <- exchange.WebsocketOrderbookUpdate{
								Exchange: p.GetName(),
								Asset:    assets.AssetTypeSpot,
								Pair:     currency.NewPairFromString(currencyPair),
							}
						case "t":
							currencyPair := CurrencyPairID[chanID]
							var trade WsTrade
							trade.Symbol = CurrencyPairID[chanID]
							trade.TradeID, _ = strconv.ParseInt(dataL3[1].(string), 10, 64)
							// 1 for buy 0 for sell
							side := exchange.BuyOrderSide.ToLower().ToString()
							if dataL3[2].(float64) != 1 {
								side = exchange.SellOrderSide.ToLower().ToString()
							}
							trade.Side = side
							trade.Volume, _ = strconv.ParseFloat(dataL3[3].(string), 64)
							trade.Price, _ = strconv.ParseFloat(dataL3[4].(string), 64)
							trade.Timestamp = int64(dataL3[5].(float64))

							p.Websocket.DataHandler <- exchange.TradeData{
								Timestamp:    time.Unix(trade.Timestamp, 0),
								CurrencyPair: currency.NewPairFromString(currencyPair),
								Side:         trade.Side,
								Amount:       trade.Volume,
								Price:        trade.Price,
							}
						}
					}
				}
			}
		}
	}
}

// WsProcessOrderbookSnapshot processes a new orderbook snapshot into a local
// of orderbooks
func (p *Poloniex) WsProcessOrderbookSnapshot(ob []interface{}, symbol string) error {
	askdata := ob[0].(map[string]interface{})
	var asks []orderbook.Item
	for price, volume := range askdata {
		assetPrice, err := strconv.ParseFloat(price, 64)
		if err != nil {
			return err
		}

		assetVolume, err := strconv.ParseFloat(volume.(string), 64)
		if err != nil {
			return err
		}

		asks = append(asks, orderbook.Item{
			Price:  assetPrice,
			Amount: assetVolume,
		})
	}

	bidData := ob[1].(map[string]interface{})
	var bids []orderbook.Item
	for price, volume := range bidData {
		assetPrice, err := strconv.ParseFloat(price, 64)
		if err != nil {
			return err
		}

		assetVolume, err := strconv.ParseFloat(volume.(string), 64)
		if err != nil {
			return err
		}

		bids = append(bids, orderbook.Item{
			Price:  assetPrice,
			Amount: assetVolume,
		})
	}

	var newOrderBook orderbook.Base
	newOrderBook.Asks = asks
	newOrderBook.Bids = bids
	newOrderBook.AssetType = assets.AssetTypeSpot
	newOrderBook.LastUpdated = time.Now()
	newOrderBook.Pair = currency.NewPairFromString(symbol)

	return p.Websocket.Orderbook.LoadSnapshot(&newOrderBook, p.GetName(), false)
}

// WsProcessOrderbookUpdate processses new orderbook updates
func (p *Poloniex) WsProcessOrderbookUpdate(target []interface{}, symbol string) error {
	sideCheck := target[1].(float64)

	cP := currency.NewPairFromString(symbol)

	price, err := strconv.ParseFloat(target[2].(string), 64)
	if err != nil {
		return err
	}

	volume, err := strconv.ParseFloat(target[3].(string), 64)
	if err != nil {
		return err
	}

	if sideCheck == 0 {
		return p.Websocket.Orderbook.Update(nil,
			[]orderbook.Item{{Price: price, Amount: volume}},
			cP,
			time.Now(),
			p.GetName(),
			assets.AssetTypeSpot)
	}

	return p.Websocket.Orderbook.Update([]orderbook.Item{{Price: price, Amount: volume}},
		nil,
		cP,
		time.Now(),
		p.GetName(),
		assets.AssetTypeSpot)
}

// CurrencyPairID contains a list of IDS for currency pairs.
var CurrencyPairID = map[int]string{
	7:   "BTC_BCN",
	14:  "BTC_BTS",
	15:  "BTC_BURST",
	20:  "BTC_CLAM",
	25:  "BTC_DGB",
	27:  "BTC_DOGE",
	24:  "BTC_DASH",
	38:  "BTC_GAME",
	43:  "BTC_HUC",
	50:  "BTC_LTC",
	51:  "BTC_MAID",
	58:  "BTC_OMNI",
	61:  "BTC_NAV",
	64:  "BTC_NMC",
	69:  "BTC_NXT",
	75:  "BTC_PPC",
	89:  "BTC_STR",
	92:  "BTC_SYS",
	97:  "BTC_VIA",
	100: "BTC_VTC",
	108: "BTC_XCP",
	114: "BTC_XMR",
	116: "BTC_XPM",
	117: "BTC_XRP",
	112: "BTC_XEM",
	148: "BTC_ETH",
	150: "BTC_SC",
	153: "BTC_EXP",
	155: "BTC_FCT",
	160: "BTC_AMP",
	162: "BTC_DCR",
	163: "BTC_LSK",
	167: "BTC_LBC",
	168: "BTC_STEEM",
	170: "BTC_SBD",
	171: "BTC_ETC",
	174: "BTC_REP",
	177: "BTC_ARDR",
	178: "BTC_ZEC",
	182: "BTC_STRAT", // nolint: misspell
	184: "BTC_PASC",
	185: "BTC_GNT",
	187: "BTC_GNO",
	189: "BTC_BCH",
	192: "BTC_ZRX",
	194: "BTC_CVC",
	196: "BTC_OMG",
	198: "BTC_GAS",
	200: "BTC_STORJ",
	201: "BTC_EOS",
	204: "BTC_SNT",
	207: "BTC_KNC",
	210: "BTC_BAT",
	213: "BTC_LOOM",
	221: "BTC_QTUM",
	121: "USDT_BTC",
	216: "USDT_DOGE",
	122: "USDT_DASH",
	123: "USDT_LTC",
	124: "USDT_NXT",
	125: "USDT_STR",
	126: "USDT_XMR",
	127: "USDT_XRP",
	149: "USDT_ETH",
	219: "USDT_SC",
	218: "USDT_LSK",
	173: "USDT_ETC",
	175: "USDT_REP",
	180: "USDT_ZEC",
	217: "USDT_GNT",
	191: "USDT_BCH",
	220: "USDT_ZRX",
	203: "USDT_EOS",
	206: "USDT_SNT",
	209: "USDT_KNC",
	212: "USDT_BAT",
	215: "USDT_LOOM",
	223: "USDT_QTUM",
	129: "XMR_BCN",
	132: "XMR_DASH",
	137: "XMR_LTC",
	138: "XMR_MAID",
	140: "XMR_NXT",
	181: "XMR_ZEC",
	166: "ETH_LSK",
	169: "ETH_STEEM",
	172: "ETH_ETC",
	176: "ETH_REP",
	179: "ETH_ZEC",
	186: "ETH_GNT",
	188: "ETH_GNO",
	190: "ETH_BCH",
	193: "ETH_ZRX",
	195: "ETH_CVC",
	197: "ETH_OMG",
	199: "ETH_GAS",
	202: "ETH_EOS",
	205: "ETH_SNT",
	208: "ETH_KNC",
	211: "ETH_BAT",
	214: "ETH_LOOM",
	222: "ETH_QTUM",
	224: "USDC_BTC",
	226: "USDC_USDT",
	225: "USDC_ETH",
}
