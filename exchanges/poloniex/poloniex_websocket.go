package poloniex

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
)

const (
	poloniexWebsocketAddress = "wss://api2.poloniex.com"
	wsAccountNotificationID  = 1000
	wsTickerDataID           = 1002
	ws24HourExchangeVolumeID = 1003
	wsHeartbeat              = 1010
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

	go p.WsReadData()
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

	pairs := p.GetEnabledCurrencies()
	for _, nextPair := range pairs {
		fPair := exchange.FormatExchangeCurrency(p.GetName(), nextPair)

		orderbookJSON, err := common.JSONEncode(WsCommand{
			Command: "subscribe",
			Channel: fPair.String(),
		})

		err = p.WebsocketConn.WriteMessage(websocket.TextMessage, orderbookJSON)
		if err != nil {
			return err
		}
	}
	return nil
}

// WsReadData reads data from the websocket connection
func (p *Poloniex) WsReadData() {
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
			_, resp, err := p.WebsocketConn.ReadMessage()
			if err != nil {
				p.Websocket.DataHandler <- err
				return
			}

			p.Websocket.TrafficAlert <- struct{}{}
			p.Websocket.Intercomm <- exchange.WebsocketResponse{Raw: resp}
		}
	}
}

// WsHandleData handles data from the websocket connection
func (p *Poloniex) WsHandleData() {
	p.Websocket.Wg.Add(1)
	defer p.Websocket.Wg.Done()

	for {
		select {
		case <-p.Websocket.ShutdownC:
			return

		case resp := <-p.Websocket.Intercomm:
			var check []interface{}
			err := common.JSONDecode(resp.Raw, &check)
			if err != nil {
				log.Fatal("poloniex_websocket.go - ", err)
			}

			switch len(check) {
			case 1:
				if check[0].(float64) == wsHeartbeat {
					continue
				}

			case 2:
				switch check[0].(type) {
				case float64:
					subscriptionID := check[0].(float64)
					if subscriptionID == ws24HourExchangeVolumeID ||
						subscriptionID == wsAccountNotificationID ||
						subscriptionID == wsTickerDataID {
						if check[1].(float64) != 1 {
							p.Websocket.DataHandler <- errors.New("poloniex.go error - Subcription failed")
							continue
						}
						continue
					}

				case string:
					orderbookSubscriptionID := check[0].(string)
					if check[1].(float64) != 1 {
						p.Websocket.DataHandler <- fmt.Errorf("poloniex.go error - orderbook subscription failed with symbol %s",
							orderbookSubscriptionID)
						continue
					}
				}

			case 3:
				switch len(check[2].([]interface{})) {
				case 1:
					// Snapshot
					datalevel1 := check[2].([]interface{})
					datalevel2 := datalevel1[0].([]interface{})

					switch datalevel2[1].(type) {
					case float64:
						err := p.WsProcessOrderbookUpdate(datalevel2,
							CurrencyPairID[int64(check[0].(float64))])
						if err != nil {
							log.Fatal(err)
						}

						p.Websocket.DataHandler <- exchange.WebsocketOrderbookUpdate{
							Exchange: p.GetName(),
							Asset:    "SPOT",
							// Pair: pair.NewCurrencyPairFromString(currencyPair),
						}

					case map[string]interface{}:
						datalevel3 := datalevel2[1].(map[string]interface{})
						currencyPair, ok := datalevel3["currencyPair"].(string)
						if !ok {
							log.Fatal("poloniex.go error - could not find currency pair in map")
						}

						orderbookData, ok := datalevel3["orderBook"].([]interface{})
						if !ok {
							log.Fatal("poloniex.go error - could not find orderbook data in map")
						}

						err := p.WsProcessOrderbookSnapshot(orderbookData, currencyPair)
						if err != nil {
							log.Fatal(err)
						}

						p.Websocket.DataHandler <- exchange.WebsocketOrderbookUpdate{
							Exchange: p.GetName(),
							Asset:    "SPOT",
							Pair:     pair.NewCurrencyPairFromString(currencyPair),
						}
						continue
					}

				case 10:
					tickerData := check[2].([]interface{})
					var ticker WsTicker

					ticker.LastPrice, _ = tickerData[0].(float64)
					// ticker.LowestAsk, _ = strconv.ParseFloat(tickerData[1].(string), 64)
					ticker.HighestBid, _ = strconv.ParseFloat(tickerData[2].(string), 64)
					ticker.PercentageChange, _ = strconv.ParseFloat(tickerData[3].(string), 64)
					ticker.BaseCurrencyVolume24H, _ = strconv.ParseFloat(tickerData[4].(string), 64)
					ticker.QuoteCurrencyVolume24H, _ = strconv.ParseFloat(tickerData[5].(string), 64)
					frozen, _ := strconv.ParseInt(tickerData[6].(string), 10, 64)
					if frozen == 1 {
						ticker.IsFrozen = true
					}
					ticker.HighestTradeIn24H, _ = tickerData[7].(float64)
					ticker.LowestTradePrice24H, _ = strconv.ParseFloat(tickerData[8].(string), 64)

					p.Websocket.DataHandler <- exchange.TickerData{
						Timestamp: time.Now(),
						Exchange:  p.GetName(),
						AssetType: "SPOT",
						LowPrice:  ticker.LowestAsk,
						HighPrice: ticker.HighestBid,
					}

				default:
					for _, element := range check[2].([]interface{}) {
						switch element.(type) {
						case []interface{}:
							data := element.([]interface{})
							if data[0].(string) == "o" {
								p.WsProcessOrderbookUpdate(data, CurrencyPairID[int64(check[0].(float64))])
								continue
							}

							var trade WsTrade

							id, _ := strconv.ParseInt(data[0].(string), 10, 64)
							trade.Symbol = CurrencyPairID[id]
							trade.TradeID, _ = data[0].(int64)
							trade.Side, _ = data[0].(string)
							trade.Volume, _ = data[0].(float64)
							trade.Price, _ = data[0].(float64)
							trade.Timestamp, _ = data[0].(int64)

							p.Websocket.DataHandler <- exchange.TradeData{
								Timestamp: time.Unix(trade.Timestamp, 0),
								// CurrencyPair: pair.NewCurrencyPairFromString(trade.Symbol),
								Side:   trade.Side,
								Amount: trade.Volume,
								Price:  trade.Price,
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

	var newOrderbook orderbook.Base
	newOrderbook.Asks = asks
	newOrderbook.Bids = bids
	newOrderbook.AssetType = "SPOT"
	newOrderbook.CurrencyPair = symbol
	newOrderbook.LastUpdated = time.Now()
	newOrderbook.Pair = pair.NewCurrencyPairFromString(symbol)

	return p.Websocket.Orderbook.LoadSnapshot(newOrderbook, p.GetName())
}

// WsProcessOrderbookUpdate processses new orderbook updates
func (p *Poloniex) WsProcessOrderbookUpdate(target []interface{}, symbol string) error {
	sideCheck := target[1].(float64)

	cP := pair.NewCurrencyPairFromString(symbol)

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
			[]orderbook.Item{orderbook.Item{Price: price, Amount: volume}},
			cP,
			time.Now(),
			p.GetName(),
			"SPOT")
	}

	return p.Websocket.Orderbook.Update([]orderbook.Item{orderbook.Item{Price: price, Amount: volume}},
		nil,
		cP,
		time.Now(),
		p.GetName(),
		"SPOT")
}

// CurrencyPairID contains a list of IDS for currency pairs.
var CurrencyPairID = map[int64]string{
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
	182: "BTC_STRAT",
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

// CurrencyID defines IDs to a currency supported by the exchange
var CurrencyID = map[int64]string{
	1:   "1CR",
	2:   "ABY",
	3:   "AC",
	4:   "ACH",
	5:   "ADN",
	6:   "AEON",
	7:   "AERO",
	8:   "AIR",
	9:   "APH",
	10:  "AUR",
	11:  "AXIS",
	12:  "BALLS",
	13:  "BANK",
	14:  "BBL",
	15:  "BBR",
	16:  "BCC",
	17:  "BCN",
	18:  "BDC",
	19:  "BDG",
	20:  "BELA",
	21:  "BITS",
	22:  "BLK",
	23:  "BLOCK",
	24:  "BLU",
	25:  "BNS",
	26:  "BONES",
	27:  "BOST",
	28:  "BTC",
	29:  "BTCD",
	30:  "BTCS",
	31:  "BTM",
	32:  "BTS",
	33:  "BURN",
	34:  "BURST",
	35:  "C2",
	36:  "CACH",
	37:  "CAI",
	38:  "CC",
	39:  "CCN",
	40:  "CGA",
	41:  "CHA",
	42:  "CINNI",
	43:  "CLAM",
	44:  "CNL",
	45:  "CNMT",
	46:  "CNOTE",
	47:  "COMM",
	48:  "CON",
	49:  "CORG",
	50:  "CRYPT",
	51:  "CURE",
	52:  "CYC",
	53:  "DGB",
	54:  "DICE",
	55:  "DIEM",
	56:  "DIME",
	57:  "DIS",
	58:  "DNS",
	59:  "DOGE",
	60:  "DASH",
	61:  "DRKC",
	62:  "DRM",
	63:  "DSH",
	64:  "DVK",
	65:  "EAC",
	66:  "EBT",
	67:  "ECC",
	68:  "EFL",
	69:  "EMC2",
	70:  "EMO",
	71:  "ENC",
	72:  "eTOK",
	73:  "EXE",
	74:  "FAC",
	75:  "FCN",
	76:  "FIBRE",
	77:  "FLAP",
	78:  "FLDC",
	79:  "FLT",
	80:  "FOX",
	81:  "FRAC",
	82:  "FRK",
	83:  "FRQ",
	84:  "FVZ",
	85:  "FZ",
	86:  "FZN",
	87:  "GAP",
	88:  "GDN",
	89:  "GEMZ",
	90:  "GEO",
	91:  "GIAR",
	92:  "GLB",
	93:  "GAME",
	94:  "GML",
	95:  "GNS",
	96:  "GOLD",
	97:  "GPC",
	98:  "GPUC",
	99:  "GRCX",
	100: "GRS",
	101: "GUE",
	102: "H2O",
	103: "HIRO",
	104: "HOT",
	105: "HUC",
	106: "HVC",
	107: "HYP",
	108: "HZ",
	109: "IFC",
	110: "ITC",
	111: "IXC",
	112: "JLH",
	113: "JPC",
	114: "JUG",
	115: "KDC",
	116: "KEY",
	117: "LC",
	118: "LCL",
	119: "LEAF",
	120: "LGC",
	121: "LOL",
	122: "LOVE",
	123: "LQD",
	124: "LTBC",
	125: "LTC",
	126: "LTCX",
	127: "MAID",
	128: "MAST",
	129: "MAX",
	130: "MCN",
	131: "MEC",
	132: "METH",
	133: "MIL",
	134: "MIN",
	135: "MINT",
	136: "MMC",
	137: "MMNXT",
	138: "MMXIV",
	139: "MNTA",
	140: "MON",
	141: "MRC",
	142: "MRS",
	143: "OMNI",
	144: "MTS",
	145: "MUN",
	146: "MYR",
	147: "MZC",
	148: "N5X",
	149: "NAS",
	150: "NAUT",
	151: "NAV",
	152: "NBT",
	153: "NEOS",
	154: "NL",
	155: "NMC",
	156: "NOBL",
	157: "NOTE",
	158: "NOXT",
	159: "NRS",
	160: "NSR",
	161: "NTX",
	162: "NXT",
	163: "NXTI",
	164: "OPAL",
	165: "PAND",
	166: "PAWN",
	167: "PIGGY",
	168: "PINK",
	169: "PLX",
	170: "PMC",
	171: "POT",
	172: "PPC",
	173: "PRC",
	174: "PRT",
	175: "PTS",
	176: "Q2C",
	177: "QBK",
	178: "QCN",
	179: "QORA",
	180: "QTL",
	181: "RBY",
	182: "RDD",
	183: "RIC",
	184: "RZR",
	185: "SDC",
	186: "SHIBE",
	187: "SHOPX",
	188: "SILK",
	189: "SJCX",
	190: "SLR",
	191: "SMC",
	192: "SOC",
	193: "SPA",
	194: "SQL",
	195: "SRCC",
	196: "SRG",
	197: "SSD",
	198: "STR",
	199: "SUM",
	200: "SUN",
	201: "SWARM",
	202: "SXC",
	203: "SYNC",
	204: "SYS",
	205: "TAC",
	206: "TOR",
	207: "TRUST",
	208: "TWE",
	209: "UIS",
	210: "ULTC",
	211: "UNITY",
	212: "URO",
	213: "USDE",
	214: "USDT",
	215: "UTC",
	216: "UTIL",
	217: "UVC",
	218: "VIA",
	219: "VOOT",
	220: "VRC",
	221: "VTC",
	222: "WC",
	223: "WDC",
	224: "WIKI",
	225: "WOLF",
	226: "X13",
	227: "XAI",
	228: "XAP",
	229: "XBC",
	230: "XC",
	231: "XCH",
	232: "XCN",
	233: "XCP",
	234: "XCR",
	235: "XDN",
	236: "XDP",
	237: "XHC",
	238: "XLB",
	239: "XMG",
	240: "XMR",
	241: "XPB",
	242: "XPM",
	243: "XRP",
	244: "XSI",
	245: "XST",
	246: "XSV",
	247: "XUSD",
	248: "XXC",
	249: "YACC",
	250: "YANG",
	251: "YC",
	252: "YIN",
	253: "XVC",
	254: "FLO",
	256: "XEM",
	258: "ARCH",
	260: "HUGE",
	261: "GRC",
	263: "IOC",
	265: "INDEX",
	267: "ETH",
	268: "SC",
	269: "BCY",
	270: "EXP",
	271: "FCT",
	272: "BITUSD",
	273: "BITCNY",
	274: "RADS",
	275: "AMP",
	276: "VOX",
	277: "DCR",
	278: "LSK",
	279: "DAO",
	280: "LBC",
	281: "STEEM",
	282: "SBD",
	283: "ETC",
	284: "REP",
	285: "ARDR",
	286: "ZEC",
	287: "STRAT",
	288: "NXC",
	289: "PASC",
	290: "GNT",
	291: "GNO",
	292: "BCH",
	293: "ZRX",
	294: "CVC",
	295: "OMG",
	296: "GAS",
	297: "STORJ",
	298: "EOS",
	299: "USDC",
	300: "SNT",
	301: "KNC",
	302: "BAT",
	303: "LOOM",
	304: "QTUM",
}
