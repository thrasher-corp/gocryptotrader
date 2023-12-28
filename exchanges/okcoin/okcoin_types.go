package okcoin

import (
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

var withdrawalFeeMaps = map[currency.Code]float64{
	currency.BTC:     0.00007,
	currency.ETH:     0.00096,
	currency.USDC:    3.5044432,
	currency.DOGE:    4,
	currency.ADA:     0.8,
	currency.SOL:     0.008,
	currency.TRX:     0.8,
	currency.MATIC:   7.46726546,
	currency.DOT:     0.08,
	currency.LTC:     0.001,
	currency.SHIB:    290655.61757874,
	currency.WBTC:    0.00017668,
	currency.UNI:     0.59184286,
	currency.AVAX:    0.008,
	currency.LINK:    0.5480391,
	currency.DAI:     0.5480391,
	currency.ATOM:    0.004,
	currency.HBAR:    0.8,
	currency.ICP:     0.0003,
	currency.LDO:     3.46374201,
	currency.CRO:     52.74539796,
	currency.NEAR:    0.008,
	currency.OP:      0.1,
	currency.MKR:     0.00463947,
	currency.GRT:     34.50290634,
	currency.AAVE:    0.07203909,
	currency.ALGO:    0.008,
	currency.SAND:    4.62930946,
	currency.SNX:     1.85469272,
	currency.STX:     0.5,
	currency.IMX:     7.94166437,
	currency.EOS:     0.08,
	currency.EGLD:    0.001,
	currency.AXS:     0.26108549,
	currency.XTZ:     0.08,
	currency.APE:     0.8,
	currency.MANA:    4.58638989,
	currency.FTM:     5,
	currency.FTM:     5,
	currency.FLOW:    0.008,
	currency.CRV:     3.45323741,
	currency.CHZ:     16.62405247,
	currency.LUNC:    1000,
	currency.MINA:    0.4,
	currency.COMP:    0.07717078,
	currency.DYDX:    2.26134119,
	currency.ZIL:     0.16,
	currency.AR:      0.01,
	currency.FLR:     1,
	currency.OneINCH: 5.61490912,
	currency.ENJ:     6.98777134,
	currency.MASK:    0.98622576,
	currency.LRC:     10.02915512,
	currency.CELO:    0.0008,
	currency.CVX:     0.92875557,
	currency.ASTR:    1,
	currency.ZEC:     0.0008,
	currency.ENS:     1.4,
	currency.RVN:     0.8,
	currency.YFI:     0.00040394,
	currency.ICX:     0.016,
	currency.KSM:     0.008,
	currency.LUNA:    0.05,
	currency.ANT:     1.48614813,
	currency.LPT:     0.34101202,
	currency.WAXP:    0.01,
	currency.USTC:    10,
	currency.ONE:     0,
	currency.SUSHI:   3.37202198,
	currency.SKL:     328,
	currency.KDA:     0,
	currency.UMA:     2.77,
	currency.KNC:     3.2,
	currency.API3:    2.13391062,
	currency.XNO:     0.1,
	currency.NMR:     0.176,
	currency.BNT:     8.40922032,
	currency.SLP:     770.4,
	currency.METIS:   0.13742175,
	currency.SPELL:   0.13742175,
	currency.PHA:     24.72661591,
	currency.AGLD:    10.74779497,
	currency.GHST:    3.93441153,
	currency.BADGER:  3.93441153,
	currency.STORJ:   9.85835481,
	currency.PERP:    8.48902141,
	currency.LAT:     0.8,
	currency.YFII:    0.00393526,
	currency.TRB:     0.24234508,
	currency.FORTH:   1.13699584,
	currency.DIA:     9.74318345,
	currency.KP3R:    0.0336,
	currency.CLV:     54.8817387,
	currency.ZRX:     16.27968018,
	currency.BRWL:    879.4987,
	currency.BRL:     0,
}

// TickerData stores ticker data
type TickerData struct {
	InstType        string       `json:"instType"`
	InstrumentID    string       `json:"instId"`
	LastTradedPrice types.Number `json:"last"`
	LastTradedSize  types.Number `json:"lastSz"`
	BestAskPrice    types.Number `json:"askPx"`
	BestAskSize     types.Number `json:"askSz"`
	BestBidPrice    types.Number `json:"bidPx"`
	BestBidSize     types.Number `json:"bidSz"`
	Open24H         types.Number `json:"open24h"`   // Open price in the past 24 hours
	High24H         types.Number `json:"high24h"`   // Highest price in the past 24 hours
	Low24H          types.Number `json:"low24h"`    // Lowest price in the past 24 hours
	VolCcy24H       types.Number `json:"volCcy24h"` // 24h trading volume, with a unit of currency. The value is the quantity in quote currency.
	Vol24H          types.Number `json:"vol24h"`    // 24h trading volume, with a unit of contract. The value is the quantity in base currency.
	Timestamp       okcoinTime   `json:"ts"`
	OpenPriceInUtc0 types.Number `json:"sodUtc0"`
	OpenPriceInUtc8 types.Number `json:"sodUtc8"`
}

// GetOrderBookResponse response data
type GetOrderBookResponse struct {
	Timestamp okcoinTime  `json:"ts"`
	Asks      [][4]string `json:"asks"` // [[0]: "Price", [1]: "Size", [2]: "Num_orders"], ...
	Bids      [][4]string `json:"bids"` // [[0]: "Price", [1]: "Size", [2]: "Num_orders"], ...
}

// WebsocketEventRequest contains event data for a websocket channel
type WebsocketEventRequest struct {
	Operation string              `json:"op"`   // 1--subscribe 2--unsubscribe 3--login
	Arguments []map[string]string `json:"args"` // args: the value is the channel name, which can be one or more channels
}

// WebsocketEventResponse contains event data for a websocket channel
type WebsocketEventResponse struct {
	Event   string `json:"event"`
	Channel string `json:"channel,omitempty"`
	Message string `json:"msg"`
	Code    string `json:"code"`
}

// WebsocketOrderbookResponse formats orderbook data for a websocket push data
type WebsocketOrderbookResponse struct {
	Arg struct {
		Channel      string `json:"channel"`
		InstrumentID string `json:"instId"`
	} `json:"arg"`
	Action string               `json:"action"`
	Data   []WebsocketOrderBook `json:"data"`
}

// WebsocketOrderBook holds orderbook data
type WebsocketOrderBook struct {
	Checksum  int64             `json:"checksum"`
	Asks      [][2]types.Number `json:"asks"` // [ Price, Quantity, depreciated, number of orders at the price ]
	Bids      [][2]types.Number `json:"bids"` // [ Price, Quantity, depreciated, number of orders at the price ]
	Timestamp okcoinTime        `json:"ts"`
}

func (a *WebsocketOrderBook) prepareOrderbook() {
	asks := [][2]types.Number{}
	for x := range a.Asks {
		if len(asks) > 0 && asks[len(asks)-1][0].Float64() == a.Asks[x][0].Float64() {
			if a.Asks[x][1].Float64() != 0 {
				if asks[len(asks)-1][1].Float64() > a.Asks[x][1].Float64() {
					asks[len(asks)-1], a.Asks[x] = a.Asks[x], asks[len(asks)-1]
				}
			} else if a.Asks[x][1] == 0 {
				continue
			}
		}
		asks = append(asks, a.Asks[x])
	}
	a.Asks = asks
	bids := [][2]types.Number{}
	for x := range a.Bids {
		if len(bids) > 0 && bids[len(bids)-1][0].Float64() == a.Bids[x][0].Float64() {
			if a.Bids[x][1].Float64() != 0 {
				if bids[len(bids)-1][1].Float64() < a.Bids[x][1].Float64() {
					bids[len(bids)-1], a.Bids[x] = a.Bids[x], bids[len(bids)-1]
				}
			} else if a.Bids[x][1] == 0 {
				continue
			}
		}
		bids = append(bids, a.Bids[x])
	}
	a.Bids = bids
}

// WebsocketDataResponse formats all response data for a websocket event
type WebsocketDataResponse struct {
	ID        string `json:"id"`
	Operation string `json:"op"`
	Message   string `json:"msg"`
	Arguments struct {
		Channel        string `json:"channel"`
		InstrumentID   string `json:"instId"`
		InstrumentType string `json:"instType"`
	} `json:"arg"`
	Action string        `json:"action"`
	Data   []interface{} `json:"data"`
}

// WebsocketDataResponseReciever formats all response data for a websocket events and used to unmarshal push data into corresponding instance
type WebsocketDataResponseReciever struct {
	Arguments struct {
		Channel        string `json:"channel"`
		InstrumentID   string `json:"instId"`
		InstrumentType string `json:"instType"`
	} `json:"arg"`
	Action string      `json:"action"`
	Data   interface{} `json:"data"`
}

// WebsocketStatus represents a system status response.
type WebsocketStatus struct {
	Arg struct {
		Channel string `json:"channel"`
	} `json:"arg"`
	Data []struct {
		Title                 string     `json:"title"`
		State                 string     `json:"state"`
		End                   okcoinTime `json:"end"`
		Begin                 okcoinTime `json:"begin"`
		Href                  string     `json:"href"`
		ServiceType           int64      `json:"serviceType,string"`
		System                string     `json:"system"`
		RescheduleDescription string     `json:"scheDesc"`
		Time                  okcoinTime `json:"ts"`
	} `json:"data"`
}

// WebsocketAccount represents an account information. Data will be pushed when triggered by events such as placing order, canceling order, transaction execution
type WebsocketAccount struct {
	Arg struct {
		Channel string `json:"channel"`
		UID     string `json:"uid"`
	} `json:"arg"`
	Data []struct {
		AdjustedEquity string `json:"adjEq"`
		Details        []struct {
			AvailableBalance   types.Number `json:"availBal"`
			AvailableEquity    types.Number `json:"availEq"`
			CashBalance        types.Number `json:"cashBal"`
			Currency           string       `json:"ccy"`
			CoinUsdPrice       types.Number `json:"coinUsdPrice"`
			CrossLiab          string       `json:"crossLiab"`
			DiscountEquity     string       `json:"disEq"`
			Equity             string       `json:"eq"`
			EquityUsd          string       `json:"eqUsd"`
			FixedBalance       types.Number `json:"fixedBal"`
			FrozenBalance      types.Number `json:"frozenBal"`
			Interest           types.Number `json:"interest"`
			IsoEquity          string       `json:"isoEq"`
			IsoLiability       string       `json:"isoLiab"`
			IsoUpl             string       `json:"isoUpl"`
			Liability          types.Number `json:"liab"`
			MaxLoan            types.Number `json:"maxLoan"`
			MgnRatio           types.Number `json:"mgnRatio"`
			NotionalLeverage   string       `json:"notionalLever"`
			MarginFrozenOrders string       `json:"ordFrozen"`
			SpotInUseAmount    string       `json:"spotInUseAmt"`
			StrategyEquity     string       `json:"stgyEq"`
			Twap               string       `json:"twap"`
			UPL                string       `json:"upl"` // Unrealized profit and loss
			UpdateTime         okcoinTime   `json:"uTime"`
		} `json:"details"`
		FrozenEquity                 string       `json:"imr"`
		IsoEquity                    string       `json:"isoEq"`
		MarginRatio                  string       `json:"mgnRatio"`
		MaintenanceMarginRequirement string       `json:"mmr"`
		NotionalUsd                  types.Number `json:"notionalUsd"`
		MarginOrderFrozen            string       `json:"ordFroz"`
		TotalEquity                  string       `json:"totalEq"`
		UpdateTime                   okcoinTime   `json:"uTime"`
	} `json:"data"`
}

// WebsocketOrder represents and order information. Data will not be pushed when first subscribed.
type WebsocketOrder struct {
	Arg struct {
		Channel        string `json:"channel"`
		InstrumentType string `json:"instType"`
		InstrumentID   string `json:"instId"`
		UID            string `json:"uid"`
	} `json:"arg"`
	Data []struct {
		AccFillSize                types.Number `json:"accFillSz"`
		AmendResult                string       `json:"amendResult"`
		AveragePrice               types.Number `json:"avgPx"`
		CreateTime                 okcoinTime   `json:"cTime"`
		Category                   string       `json:"category"`
		Currency                   string       `json:"ccy"`
		ClientOrdID                string       `json:"clOrdId"`
		Code                       string       `json:"code"`
		ExecType                   string       `json:"execType"`
		Fee                        types.Number `json:"fee"`
		FeeCurrency                string       `json:"feeCcy"`
		FillFee                    types.Number `json:"fillFee"`
		FillFeeCurrency            string       `json:"fillFeeCcy"`
		FillNotionalUsd            types.Number `json:"fillNotionalUsd"`
		FillPrice                  types.Number `json:"fillPx"`
		FillSize                   types.Number `json:"fillSz"`
		FillTime                   okcoinTime   `json:"fillTime"`
		InstrumentID               string       `json:"instId"`
		InstrumentType             string       `json:"instType"`
		Leverage                   types.Number `json:"lever"`
		ErrorMessage               string       `json:"msg"`
		NotionalUsd                types.Number `json:"notionalUsd"`
		OrderID                    string       `json:"ordId"`
		OrderType                  string       `json:"ordType"`
		ProfitAndLoss              types.Number `json:"pnl"`
		PositionSide               string       `json:"posSide"`
		Price                      types.Number `json:"px"`
		Rebate                     string       `json:"rebate"`
		RebateCurrency             string       `json:"rebateCcy"`
		ReduceOnly                 bool         `json:"reduceOnly,string"`
		ClientRequestID            string       `json:"reqId"`
		Side                       string       `json:"side"`
		StopLossOrderPrice         types.Number `json:"slOrdPx"`
		StopLossTriggerPrice       types.Number `json:"slTriggerPx"`
		StopLossTriggerPriceType   string       `json:"slTriggerPxType"`
		Source                     string       `json:"source"`
		State                      string       `json:"state"`
		Size                       types.Number `json:"sz"`
		Tag                        string       `json:"tag"`
		TradeMode                  string       `json:"tdMode"`
		TargetCurrency             string       `json:"tgtCcy"`
		TakeProfitOrdPrice         types.Number `json:"tpOrdPx"`
		TakeProfitTriggerPrice     types.Number `json:"tpTriggerPx"`
		TakeProfitTriggerPriceType string       `json:"tpTriggerPxType"`
		TradeID                    string       `json:"tradeId"`
		UpdateTime                 okcoinTime   `json:"uTime"`
	} `json:"data"`
}

// WebsocketAlgoOrder represents algo orders (includes trigger order, oco order, conditional order).
type WebsocketAlgoOrder struct {
	Arg struct {
		Channel      string `json:"channel"`
		UID          string `json:"uid"`
		InstType     string `json:"instType"`
		InstrumentID string `json:"instId"`
	} `json:"arg"`
	Data []struct {
		InstrumentType             string       `json:"instType"`
		InstrumentID               string       `json:"instId"`
		OrderID                    string       `json:"ordId"`
		Currency                   string       `json:"ccy"`
		ClientOrderID              string       `json:"clOrdId"`
		AlgoID                     string       `json:"algoId"`
		Price                      types.Number `json:"px"`
		Size                       types.Number `json:"sz"`
		TradeMode                  string       `json:"tdMode"`
		TgtCurrency                string       `json:"tgtCcy"`
		NotionalUsd                types.Number `json:"notionalUsd"`
		OrderType                  string       `json:"ordType"`
		Side                       string       `json:"side"`
		PositionSide               string       `json:"posSide"`
		State                      string       `json:"state"`
		Leverage                   float64      `json:"lever"`
		TakeProfitTriggerPrice     types.Number `json:"tpTriggerPx"`
		TakeProfitTriggerPriceType string       `json:"tpTriggerPxType"`
		TakeProfitOrdPrice         types.Number `json:"tpOrdPx"`
		SlTriggerPrice             types.Number `json:"slTriggerPx"`
		SlTriggerPriceType         string       `json:"slTriggerPxType"`
		TriggerPxType              string       `json:"triggerPxType"`
		TriggerPrice               types.Number `json:"triggerPx"`
		OrderPrice                 types.Number `json:"ordPx"`
		Tag                        string       `json:"tag"`
		ActualSize                 types.Number `json:"actualSz"`
		ActualPrice                types.Number `json:"actualPx"`
		ActualSide                 string       `json:"actualSide"`
		TriggerTime                okcoinTime   `json:"triggerTime"`
		CreateTime                 okcoinTime   `json:"cTime"`
	} `json:"data"`
}

// WebsocketAdvancedAlgoOrder represents advance algo orders (including Iceberg order, TWAP order, Trailing order).
type WebsocketAdvancedAlgoOrder struct {
	Arg struct {
		Channel      string `json:"channel"`
		UID          string `json:"uid"`
		InstType     string `json:"instType"`
		InstrumentID string `json:"instId"`
	} `json:"arg"`
	Data []struct {
		ActualPx             types.Number `json:"actualPx"`
		ActualSide           string       `json:"actualSide"`
		ActualSz             types.Number `json:"actualSz"`
		AlgoID               string       `json:"algoId"`
		CreationTime         okcoinTime   `json:"cTime"`
		Ccy                  string       `json:"ccy"`
		ClOrdID              string       `json:"clOrdId"`
		Count                string       `json:"count"`
		InstrumentID         string       `json:"instId"`
		InstType             string       `json:"instType"`
		Lever                types.Number `json:"lever"`
		NotionalUsd          types.Number `json:"notionalUsd"`
		OrderPrice           types.Number `json:"ordPx"`
		OrderType            string       `json:"ordType"`
		PushTime             okcoinTime   `json:"pTime"`
		PosSide              string       `json:"posSide"`
		PriceLimit           types.Number `json:"pxLimit"`
		PriceSpread          types.Number `json:"pxSpread"`
		PriceVar             types.Number `json:"pxVar"`
		Side                 string       `json:"side"`
		StopLossOrdPrice     string       `json:"slOrdPx"`
		StopLossTriggerPrice string       `json:"slTriggerPx"`
		State                string       `json:"state"`
		Size                 types.Number `json:"sz"`
		SizeLimit            types.Number `json:"szLimit"`
		TradeMode            string       `json:"tdMode"`
		TimeInterval         string       `json:"timeInterval"`
		TakeProfitOrdPx      types.Number `json:"tpOrdPx"`
		TakeProfitTriggerPx  types.Number `json:"tpTriggerPx"`
		Tag                  string       `json:"tag"`
		TriggerPrice         types.Number `json:"triggerPx"`
		TriggerTime          string       `json:"triggerTime"`
		CallbackRatio        string       `json:"callbackRatio"`
		CallbackSpread       string       `json:"callbackSpread"`
		ActivePrice          types.Number `json:"activePx"`
		MoveTriggerPrice     types.Number `json:"moveTriggerPx"`
	} `json:"data"`
}

// WebsocketInstrumentData contains formatted data for instruments related websocket responses
type WebsocketInstrumentData struct {
	Alias                 string       `json:"alias"`
	BaseCurrency          string       `json:"baseCcy"`
	Category              string       `json:"category"`
	ContractMultiplier    string       `json:"ctMult"`
	ContractType          string       `json:"ctType"`
	ContractValue         string       `json:"ctVal"`
	ContractValueCurrency string       `json:"ctValCcy"`
	ExpiryTime            okcoinTime   `json:"expTime"`
	InstrumentFamily      string       `json:"instFamily"`
	InstrumentID          string       `json:"instId"`
	InstrumentType        string       `json:"instType"`
	Leverage              types.Number `json:"lever"`
	ListTime              okcoinTime   `json:"listTime"`
	LotSize               types.Number `json:"lotSz"`
	MaxIcebergSize        types.Number `json:"maxIcebergSz"`
	MaxLimitSize          types.Number `json:"maxLmtSz"`
	MaxMarketSize         types.Number `json:"maxMktSz"`
	MaxStopSize           types.Number `json:"maxStopSz"`
	MaxTriggerSize        types.Number `json:"maxTriggerSz"`
	MaxTwapSize           types.Number `json:"maxTwapSz"`
	MinimumOrderSize      types.Number `json:"minSz"`
	OptionType            string       `json:"optType"`
	QuoteCurrency         string       `json:"quoteCcy"`
	SettleCurrency        string       `json:"settleCcy"`
	State                 string       `json:"state"`
	StrikePrice           types.Number `json:"stk"`
	TickSize              types.Number `json:"tickSz"`
	Underlying            string       `json:"uly"`
}

// WsTickerData contains formatted data for ticker related websocket responses
type WsTickerData struct {
	InstrumentType string       `json:"instType"`
	InstrumentID   string       `json:"instId"`
	Last           types.Number `json:"last"`
	LastSize       types.Number `json:"lastSz"`
	AskPrice       types.Number `json:"askPx"`
	AskSize        types.Number `json:"askSz"`
	BidPrice       types.Number `json:"bidPx"`
	BidSize        types.Number `json:"bidSz"`
	Open24H        types.Number `json:"open24h"`
	High24H        types.Number `json:"high24h"`
	Low24H         types.Number `json:"low24h"`
	SodUtc0        string       `json:"sodUtc0"`
	SodUtc8        string       `json:"sodUtc8"`
	VolCcy24H      types.Number `json:"volCcy24h"`
	Vol24H         types.Number `json:"vol24h"`
	Timestamp      okcoinTime   `json:"ts"`
}

// WebsocketTradeResponse contains formatted data for trade related websocket responses
type WebsocketTradeResponse struct {
	Arg struct {
		Channel      string `json:"channel"`
		InstrumentID string `json:"instId"`
	} `json:"arg"`
	Data []struct {
		InstrumentID string       `json:"instId"`
		TradeID      string       `json:"tradeId"`
		Price        types.Number `json:"px"`
		Size         types.Number `json:"sz"`
		Side         string       `json:"side"`
		Timestamp    okcoinTime   `json:"ts"`
	} `json:"data"`
}

// WebsocketCandlesResponse represents a candlestick response data.
type WebsocketCandlesResponse struct {
	Arg struct {
		Channel      string `json:"channel"`
		InstrumentID string `json:"instId"`
	} `json:"arg"`
	Data [][]string `json:"data"`
}

// WebsocketOrderBooksData is the full websocket response containing orderbook data
type WebsocketOrderBooksData struct {
	Table  string               `json:"table"`
	Action string               `json:"action"`
	Data   []WebsocketOrderBook `json:"data"`
}

// WebsocketErrorResponse yo
type WebsocketErrorResponse struct {
	Event     string `json:"event"`
	Message   string `json:"message"`
	ErrorCode int64  `json:"errorCode"`
}

// List of all websocket channels to subscribe to
const (
	okcoinWsRateLimit   = 30
	allowableIterations = 25
	maxConnByteLen      = 4096
)

// SystemStatus represents system status
type SystemStatus struct {
	Title       string     `json:"title"`
	State       string     `json:"state"`
	Begin       okcoinTime `json:"begin"`
	End         okcoinTime `json:"end"`
	Href        string     `json:"href"`
	ServiceType string     `json:"serviceType"`
	System      string     `json:"system"`
	ScheDesc    string     `json:"scheDesc"`
}

// Instrument represents an instrument in an open contract.
type Instrument struct {
	Alias          string       `json:"alias"`
	BaseCurrency   string       `json:"baseCcy"`
	Category       string       `json:"category"`
	CtMult         string       `json:"ctMult"`
	CtType         string       `json:"ctType"`
	CtVal          string       `json:"ctVal"`
	CtValCurrency  string       `json:"ctValCcy"`
	ExpTime        okcoinTime   `json:"expTime"`
	InstFamily     string       `json:"instFamily"`
	InstrumentID   string       `json:"instId"`
	InstrumentType string       `json:"instType"`
	Leverage       types.Number `json:"lever"`
	ListTime       okcoinTime   `json:"listTime"`
	LotSize        types.Number `json:"lotSz"`
	MaxIcebergSz   types.Number `json:"maxIcebergSz"`
	MaxLimitSize   types.Number `json:"maxLmtSz"`
	MaxMarketSize  types.Number `json:"maxMktSz"`
	MaxStopSize    types.Number `json:"maxStopSz"`
	MaxTwapSize    types.Number `json:"maxTwapSz"`
	MaxTriggerSize types.Number `json:"maxTriggerSz"`
	MinSize        types.Number `json:"minSz"` // Minimum order size
	QuoteCurrency  string       `json:"quoteCcy"`
	OptionType     string       `json:"optType"`
	SettleCurrency string       `json:"settleCcy"`
	State          string       `json:"state"`
	StrikePrice    types.Number `json:"stk"`
	TickSize       types.Number `json:"tickSz"`
	Underlying     string       `json:"uly"`
}

type candlestickItemResponse [9]string

// CandlestickData represents the candlestick chart
type CandlestickData struct {
	Timestamp            okcoinTime
	OpenPrice            float64
	HighestPrice         float64
	LowestPrice          float64
	ClosePrice           float64
	TradingVolume        float64
	QuoteTradingVolume   float64
	TradingVolumeInQuote float64
	Confirm              string
}

// SpotTrade represents spot trades
type SpotTrade struct {
	InstrumentID string       `json:"instId"`
	Side         string       `json:"side"`
	TradeSize    types.Number `json:"sz"`
	TradePrice   types.Number `json:"px"`
	TradeID      string       `json:"tradeId"`
	Timestamp    okcoinTime   `json:"ts"`
}

// TradingVolume represents the trading volume of the platform in 24 hours
type TradingVolume struct {
	VolCny    types.Number `json:"volCny"`
	VolUsd    types.Number `json:"volUsd"`
	Timestamp okcoinTime   `json:"ts"`
}

// Oracle represents crypto price of signing using Open Oracle smart contract.
type Oracle []struct {
	Messages   []string          `json:"messages"`
	Prices     map[string]string `json:"prices"`
	Signatures []string          `json:"signatures"`
	Timestamp  okcoinTime        `json:"timestamp"`
}

// ExchangeRate represents average exchange rate data
type ExchangeRate struct {
	UsdCny string `json:"usdCny"`
}

// ToExtract returns a CandlestickData instance from []string
func (c *candlestickItemResponse) ToExtract() (CandlestickData, error) {
	var candle CandlestickData
	err := candle.Timestamp.UnmarshalJSON([]byte(c[0]))
	if err != nil {
		return candle, err
	}
	candle.OpenPrice, err = strconv.ParseFloat(c[1], 64)
	if err != nil {
		return candle, err
	}
	candle.HighestPrice, err = strconv.ParseFloat(c[2], 64)
	if err != nil {
		return candle, err
	}
	candle.LowestPrice, err = strconv.ParseFloat(c[3], 64)
	if err != nil {
		return candle, err
	}
	candle.ClosePrice, err = strconv.ParseFloat(c[4], 64)
	if err != nil {
		return candle, err
	}
	candle.TradingVolume, err = strconv.ParseFloat(c[5], 64)
	if err != nil {
		return candle, err
	}
	if c[6] != "" {
		candle.QuoteTradingVolume, err = strconv.ParseFloat(c[6], 64)
		if err != nil {
			return candle, err
		}
	}
	if c[7] != "" {
		candle.TradingVolumeInQuote, err = strconv.ParseFloat(c[7], 64)
		if err != nil {
			return candle, err
		}
	}
	candle.Confirm = c[8]
	return candle, nil
}

// ExtractCandlesticks retrieves a list of CandlestickData
func ExtractCandlesticks(candles []candlestickItemResponse) ([]CandlestickData, error) {
	candlestickData := make([]CandlestickData, len(candles))
	var err error
	for x := range candles {
		candlestickData[x], err = candles[x].ToExtract()
		if err != nil {
			return nil, err
		}
	}
	return candlestickData, nil
}

// CurrencyInfo represents a currency instance detailed information
type CurrencyInfo struct {
	CanDep                     bool         `json:"canDep"`
	CanInternal                bool         `json:"canInternal"`
	CanWd                      bool         `json:"canWd"`
	Currency                   string       `json:"ccy"`
	Chain                      string       `json:"chain"`
	DepQuotaFixed              string       `json:"depQuotaFixed"`
	DepQuoteDailyLayer2        string       `json:"depQuoteDailyLayer2"`
	LogoLink                   string       `json:"logoLink"`
	MainNet                    bool         `json:"mainNet"`
	MaxFee                     types.Number `json:"maxFee"`
	MaxWithdrawal              types.Number `json:"maxWd"`
	MinDeposit                 types.Number `json:"minDep"`
	MinDepArrivalConfirm       string       `json:"minDepArrivalConfirm"`
	MinFee                     types.Number `json:"minFee"`
	MinWithdrawal              types.Number `json:"minWd"`
	MinWithdrawalUnlockConfirm string       `json:"minWdUnlockConfirm"`
	Name                       string       `json:"name"`
	NeedTag                    bool         `json:"needTag"`
	UsedDepQuotaFixed          string       `json:"usedDepQuotaFixed"`
	UsedWdQuota                string       `json:"usedWdQuota"`
	WithdrawalQuota            string       `json:"wdQuota"`
	WithdrawalTickSize         types.Number `json:"wdTickSz"`
}

// CurrencyBalance represents a currency balance information.
type CurrencyBalance struct {
	AvailableBalance types.Number `json:"availBal"`
	Balance          types.Number `json:"bal"`
	Currency         string       `json:"ccy"`
	FrozenBalance    types.Number `json:"frozenBal"`
}

// AccountAssetValuation represents account asset valuation
type AccountAssetValuation struct {
	Details struct {
		Classic types.Number `json:"classic"`
		Earn    types.Number `json:"earn"`
		Funding types.Number `json:"funding"`
		Trading types.Number `json:"trading"`
	} `json:"details"`
	TotalBalance types.Number `json:"totalBal"`
	Timestamp    okcoinTime   `json:"ts"`
}

// FundingTransferRequest represents a transfer of funds between your funding account and trading account
type FundingTransferRequest struct {
	Currency currency.Code `json:"ccy"`
	// Transfer type
	// 0: transfer within account
	// 1: master account to sub-account (Only applicable to APIKey from master account)
	// 2: sub-account to master account (Only applicable to APIKey from master account)
	// 3: sub-account to master account (Only applicable to APIKey from sub-account)
	// The default is 0.
	Amount       float64 `json:"amt,string"`
	From         string  `json:"from"`
	To           string  `json:"to"`
	TransferType int32   `json:"type,string,omitempty"`
	SubAccount   string  `json:"subAcct,omitempty"`
	ClientID     string  `json:"clientId,omitempty"`
}

// FundingTransferItem represents a response for a transfer of funds between your funding account and trading account
type FundingTransferItem struct {
	TransferID     string       `json:"transId"`
	Currency       string       `json:"ccy"`
	ClientID       string       `json:"clientId"`
	From           string       `json:"from"`
	Amount         types.Number `json:"amt"`
	InstrumentID   string       `json:"instId"`
	State          string       `json:"state"`
	SubAcct        string       `json:"subAcct"`
	To             string       `json:"to"`
	ToInstrumentID string       `json:"toInstId"`
	Type           string       `json:"type"`
}

// AssetBillDetail represents the billing record.
type AssetBillDetail struct {
	BillID    string       `json:"billId"`
	Currency  string       `json:"ccy"`
	ClientID  string       `json:"clientId"`
	BalChange types.Number `json:"balChg"`
	Balance   types.Number `json:"bal"`
	Type      string       `json:"type"`
	Timestamp okcoinTime   `json:"ts"`
}

// LightningDepositDetail represents a lightning deposit instance detail
type LightningDepositDetail struct {
	CreationTime okcoinTime `json:"cTime"`
	Invoice      string     `json:"invoice"`
}

// DepositAddress represents a currency deposit address detail
type DepositAddress struct {
	Chain                    string `json:"chain"`
	ContractAddr             string `json:"ctAddr"`
	Ccy                      string `json:"ccy"`
	To                       string `json:"to"`
	Address                  string `json:"addr"`
	Selected                 bool   `json:"selected"`
	Tag                      string `json:"tag"`
	Memo                     string `json:"memo"`
	DepositPaymentID         string `json:"pmtId"`
	DepositAddressAttachment string `json:"addrEx"`
}

// DepositHistoryItem represents deposit records according to the currency, deposit status, and time range in reverse chronological order.
type DepositHistoryItem struct {
	ActualDepBlkConfirm string       `json:"actualDepBlkConfirm"` // ActualDepBlkConfirm actual amount of blockchain confirm in a single deposit
	Amount              types.Number `json:"amt"`
	Currency            string       `json:"ccy"`
	Chain               string       `json:"chain"`
	DepositID           string       `json:"depId"`
	From                string       `json:"from"`
	State               string       `json:"state"`
	To                  string       `json:"to"`
	Timestamp           okcoinTime   `json:"ts"`
	TransactionID       string       `json:"txId"`
}

// WithdrawalRequest represents withdrawal of tokens request.
type WithdrawalRequest struct {
	Amount           float64       `json:"amt,string,omitempty"`
	TransactionFee   float64       `json:"fee,string"`
	WithdrawalMethod string        `json:"dest,omitempty"` // Withdrawal method 3: internal  4: on chain
	Ccy              currency.Code `json:"ccy,omitempty"`
	Chain            string        `json:"chain,omitempty"`
	ClientID         string        `json:"clientId,omitempty"`
	ToAddress        string        `json:"toAddr,omitempty"`
}

// WithdrawalResponse represents withdrawal of tokens response.
type WithdrawalResponse struct {
	Amt      types.Number `json:"amt"`
	WdID     string       `json:"wdId"`
	Currency string       `json:"ccy"`
	ClientID string       `json:"clientId"`
	Chain    string       `json:"chain"`
}

// LightningWithdrawalsRequest represents lightning withdrawal request params
type LightningWithdrawalsRequest struct {
	Ccy     currency.Code `json:"ccy"`
	Invoice string        `json:"invoice"`
	Memo    string        `json:"memo,omitempty"`
}

// LightningWithdrawals the minimum withdrawal amount is approximately 0.000001 BTC. Sub-account does not support withdrawal.
type LightningWithdrawals struct {
	WithdrawalID string     `json:"wdId"`
	CreationTime okcoinTime `json:"cTime"`
}

// WithdrawalCancellation represents a request parameter for withdrawal cancellation
type WithdrawalCancellation struct {
	WithdrawalID string `json:"wdId"`
}

// WithdrawalOrderItem represents a withdrawal instance item
type WithdrawalOrderItem struct {
	Ccy                         string       `json:"ccy"`
	Chain                       string       `json:"chain"`
	Amount                      types.Number `json:"amt"`
	Timestamp                   okcoinTime   `json:"ts"`
	RemittingAddress            string       `json:"from"`
	ReceivingAddress            string       `json:"to"`
	Tag                         string       `json:"tag"`
	PaymentID                   string       `json:"pmtId"`
	Memo                        string       `json:"memo"`
	WithdrawalAddressAttachment string       `json:"addrEx"`
	TransactionID               string       `json:"txId"`
	Fee                         types.Number `json:"fee"`
	State                       string       `json:"state"`
	WithdrawalID                string       `json:"wdId"`
	ClientID                    string       `json:"clientId"`
}

// AccountBalanceInformation represents currency balance information.
type AccountBalanceInformation struct {
	AdjustedEquity string `json:"adjEq"` // Adjusted / Effective equity in USD . Not enabled. Please disregard.
	Details        []struct {
		AvailableBalance                 types.Number `json:"availBal"`
		AvailableEquity                  types.Number `json:"availEq"`
		CashBalance                      types.Number `json:"cashBal"`
		Currency                         string       `json:"ccy"`
		CrossLiability                   string       `json:"crossLiab"`
		DiscountEquity                   string       `json:"disEq"`
		Equity                           types.Number `json:"eq"`
		EquityUsd                        types.Number `json:"eqUsd"`
		FixedBalance                     types.Number `json:"fixedBal"`
		FrozenBalance                    types.Number `json:"frozenBal"`
		Interest                         types.Number `json:"interest"`
		IsolatedEquity                   types.Number `json:"isoEq"`
		IsolatedLiability                types.Number `json:"isoLiab"`
		IsolatedUpl                      string       `json:"isoUpl"` // Isolated unrealized profit and loss of the currency. Not enabled. Please disregard.
		Liability                        types.Number `json:"liab"`
		MaxLoan                          types.Number `json:"maxLoan"`
		MarginRatio                      types.Number `json:"mgnRatio"`
		NotionalLever                    types.Number `json:"notionalLever"`
		OrderFrozen                      types.Number `json:"ordFrozen"`
		SpotInUseAmount                  types.Number `json:"spotInUseAmt"`
		StrategyEquity                   string       `json:"stgyEq"`
		Twap                             string       `json:"twap"`
		UpdateTime                       okcoinTime   `json:"uTime"`
		UnrealizedProfitAndLoss          types.Number `json:"upl"`
		UnrealizedProfitAndLossLiability string       `json:"uplLiab"`
	} `json:"details"`
	IMR            string       `json:"imr"` // Frozen equity for open positions and pending orders in USD.
	IsolatedEquity string       `json:"isoEq"`
	MarginRatio    types.Number `json:"mgnRatio"`
	Mmr            string       `json:"mmr"` // Maintenance margin requirement in USD.
	NotionalUsd    types.Number `json:"notionalUsd"`
	OrdFroz        string       `json:"ordFroz"`
	TotalEq        string       `json:"totalEq"`
	UpdateTime     okcoinTime   `json:"uTime"`
}

// BillsDetail represents a bill
type BillsDetail struct {
	Balance          types.Number `json:"bal"`
	BalanceChange    types.Number `json:"balChg"`
	BillID           string       `json:"billId"`
	Currency         string       `json:"ccy"`
	ExecType         string       `json:"execType"`
	Fee              types.Number `json:"fee"`
	From             string       `json:"from"`
	InstrumentID     string       `json:"instId"`
	InstrumentType   string       `json:"instType"`
	MarginMode       string       `json:"mgnMode"`
	Notes            string       `json:"notes"`
	OrderID          string       `json:"ordId"`
	ProfitAndLoss    types.Number `json:"pnl"`
	PosBalance       types.Number `json:"posBal"`
	PosBalanceChange types.Number `json:"posBalChg"`
	BillSubType      string       `json:"subType"`
	Size             types.Number `json:"sz"`
	To               string       `json:"to"`
	BillType         string       `json:"type"`
	Timestamp        okcoinTime   `json:"ts"`
}

// AccountConfiguration represents account configuration information.
type AccountConfiguration struct {
	AccountLevel         string `json:"acctLv"`
	AutoLoan             bool   `json:"autoLoan"`
	ContractIsolatedMode string `json:"ctIsoMode"`
	GreeksType           string `json:"greeksType"`
	Level                string `json:"level"`
	LevelTemporary       string `json:"levelTmp"`
	MarginIsolatedMode   string `json:"mgnIsoMode"`
	PositionMode         string `json:"posMode"`
	SpotOffsetType       string `json:"spotOffsetType"`
	UID                  string `json:"uid"`
}

// MaxBuySellResp represent a maximum buy sell or open amount information.
type MaxBuySellResp struct {
	Currency     string       `json:"ccy"`
	InstrumentID string       `json:"instId"`
	MaxBuy       types.Number `json:"maxBuy"`
	MaxSell      types.Number `json:"maxSell"`
}

// AvailableTradableAmount represents maximum available tradable amount information
type AvailableTradableAmount struct {
	AvailableBuy  types.Number `json:"availBuy"`
	AvailableSell types.Number `json:"availSell"`
	InstrumentID  string       `json:"instId"`
}

// FeeRate represents instrument trading fee information.
type FeeRate struct {
	Category       string       `json:"category"`
	Delivery       string       `json:"delivery"`
	Exercise       string       `json:"exercise"`
	InstrumentType string       `json:"instType"`
	Level          string       `json:"level"`
	MakerFeeRate   types.Number `json:"maker"`
	MakerU         types.Number `json:"makerU"`
	MakerUSDC      types.Number `json:"makerUSDC"`
	TakerFeeRate   types.Number `json:"taker"`
	TakerU         types.Number `json:"takerU"`
	TakerUSDC      types.Number `json:"takerUSDC"`
	Timestamp      okcoinTime   `json:"ts"`
}

// MaximumWithdrawal represents maximum withdrawal information for currency.
type MaximumWithdrawal struct {
	Currency          string `json:"ccy"`
	MaxWithdrawal     string `json:"maxWd"`
	MaxWithdrawalEx   string `json:"maxWdEx"`
	SpotOffsetMaxWd   string `json:"spotOffsetMaxWd"`
	SpotOffsetMaxWdEx string `json:"spotOffsetMaxWdEx"`
}

// AvailableRFQPair represents list of instruments and
type AvailableRFQPair struct {
	Instruments []struct {
		BaseCurrency      string       `json:"baseCcy"`
		BaseCurrencyIcon  string       `json:"baseCcyIcon"`
		BaseSingleMin     types.Number `json:"baseSingleMin"`
		InstrumentID      string       `json:"instId"`
		QuoteCurrency     string       `json:"quoteCcy"`
		QuoteCurrencyIcon string       `json:"quoteCcyIcon"`
		QuoteSingleMin    types.Number `json:"quoteSingleMin"`
	} `json:"instruments"`
	Timestamp okcoinTime `json:"ts"`
}

// QuoteRequestArg market quotation information
type QuoteRequestArg struct {
	BaseCurrency                currency.Code `json:"baseCcy"`
	QuoteCurrency               currency.Code `json:"quoteCcy"`
	Side                        string        `json:"side"`
	RfqSize                     float64       `json:"rfqSz,string"` // Amount
	RfqSzCurrency               currency.Code `json:"rfqSzCcy"`     // Token
	ClientDefinedQuoteRequestID string        `json:"clQReqId,omitempty"`
	ClientRequestTimestamp      string        `json:"clQReqTs,omitempty"`
}

// RFQQuoteResponse query current market quotation information
type RFQQuoteResponse struct {
	QuoteTimestamp okcoinTime   `json:"quoteTs"`
	TTLMs          string       `json:"ttlMs"`
	ClQReqID       string       `json:"clQReqId"`
	QuoteID        string       `json:"quoteId"`
	BaseCurrency   string       `json:"baseCcy"`
	QuoteCurrency  string       `json:"quoteCcy"`
	Side           string       `json:"side"`
	OrigRfqSize    float64      `json:"origRfqSz"`
	RfqSize        float64      `json:"rfqSz"`
	RfqSzCurrency  string       `json:"rfqSzCcy"`
	BidPrice       types.Number `json:"bidPx"`
	BidBaseSize    types.Number `json:"bidBaseSz"`
	BidQuoteSize   types.Number `json:"bidQuoteSz"`
	AskPx          types.Number `json:"askPx"`
	AskBaseSize    types.Number `json:"askBaseSz"`
	AskQuoteSize   types.Number `json:"askQuoteSz"`
}

// PlaceRFQOrderRequest represents a place RFQ request order.
type PlaceRFQOrderRequest struct {
	ClientDefinedTradeRequestID string        `json:"clTReqId"`
	ClientRFQSendingTime        int64         `json:"clTReqTs"`
	QuoteID                     string        `json:"quoteId"`
	BaseCurrency                currency.Code `json:"baseCcy"`
	QuoteCurrency               currency.Code `json:"quoteCcy"`
	Side                        string        `json:"side"`
	Size                        float64       `json:"Sz,string"`
	SizeCurrency                currency.Code `json:"szCcy"`
}

// RFQOrderResponse represents an RFQ
type RFQOrderResponse struct {
	Timestamp      okcoinTime   `json:"ts"`
	TradeID        string       `json:"tradeId"`
	QuoteID        string       `json:"quoteId"`
	ClTReqID       string       `json:"clTReqId"` // user-defined ID
	State          string       `json:"state"`
	InstrumentID   string       `json:"instId"`
	BaseCurrency   string       `json:"baseCcy"`
	QuoteCurrency  string       `json:"quoteCcy"`
	Side           string       `json:"side"`
	Price          types.Number `json:"px"`
	FilledBaseSize types.Number `json:"filledBaseSz"`
	FilledTermSize types.Number `json:"filledTermSz"`
}

// RFQOrderDetail represents an rfq order detail
type RFQOrderDetail struct {
	Timestamp      okcoinTime   `json:"ts"`
	TradeID        string       `json:"tradeId"`
	QuoteID        string       `json:"quoteId"`
	ClTReqID       string       `json:"clTReqId"`
	State          string       `json:"state"`
	InstrumentID   string       `json:"instId"`
	BaseCurrency   string       `json:"baseCcy"`
	QuoteCurrency  string       `json:"quoteCcy"`
	Side           string       `json:"side"`
	Price          types.Number `json:"px"`
	FilledBaseSize types.Number `json:"filledBaseSz"`
	FilledTermSize types.Number `json:"filledTermSz"`
}

// RFQOrderHistoryItem represents otc rfq order instance.
type RFQOrderHistoryItem struct {
	Timestamp        okcoinTime `json:"ts"`
	PageIdx          int64      `json:"pageIdx,string"`
	TotalPageCount   int64      `json:"totalPageCnt,string"`
	TotalRecordCount int64      `json:"totalRecordCnt,string"`
	Trades           []struct {
		Timestamp      okcoinTime   `json:"ts"`
		TradeID        string       `json:"tradeId"`
		TradeTimestamp okcoinTime   `json:"tradeTs"`
		ClTRequestID   string       `json:"clTReqId"`
		InstrumentID   string       `json:"instId"`
		Side           string       `json:"side"`
		Price          types.Number `json:"px"`
		BaseCurrency   string       `json:"baseCcy"`
		BaseSize       types.Number `json:"baseSz"`
		QuoteCurrency  string       `json:"quoteCcy"`
		QuoteSize      types.Number `json:"quoteSz"`
	} `json:"trades"`
}

// FiatDepositRequestArg represents
type FiatDepositRequestArg struct {
	ChannelID         string  `json:"chanId"` // Channel ID. 9:PrimeX; 28:PrimeX US; 21:PrimeX Europe; 3:Silvergate SEN; 27:Silvergate SEN HK; 24:ACH
	BankAccountNumber string  `json:"bankAcctNum"`
	Amount            float64 `json:"amt,string"`
	To                float64 `json:"to,omitempty,string"` // Amount to deposit. Recharge to the account: funding:Funding Account
}

// FiatDepositResponse represents a fiat deposit response data
type FiatDepositResponse struct {
	DepositID    string     `json:"depId"`
	CreationTime okcoinTime `json:"cTime"`
}

// CancelDepositAddressResp represents a deposit address id response after cancelling.
type CancelDepositAddressResp struct {
	DepositAddressID string `json:"depId"`
}

// DepositHistoryResponse represents a deposit history instance detail.
type DepositHistoryResponse struct {
	DepositID         string       `json:"depId"`
	ChannelID         string       `json:"chanId"`
	BillID            string       `json:"billId"`
	BankAccountName   string       `json:"bankAcctName"`
	BankAccountNumber string       `json:"bankAcctNum"`
	Amount            types.Number `json:"amt"`
	State             string       `json:"state"`
	Currency          string       `json:"ccy"`
	CreationTime      okcoinTime   `json:"cTime"`
	UpdatedTime       okcoinTime   `json:"uTime"`
}

// FiatWithdrawalParam represents a fiat withdrawal parameters
type FiatWithdrawalParam struct {
	ChannelID      string  `json:"chanId"`
	BankAcctNumber string  `json:"bankAcctNum"`
	Amount         float64 `json:"amt,string"`
}

// FiatWithdrawalResponse represents a fiat withdrawal
type FiatWithdrawalResponse struct {
	DepositID    string       `json:"depId"`
	Fee          types.Number `json:"fee"`
	CreationTime okcoinTime   `json:"cTime"`
}

// FiatWithdrawalHistoryItem represents fiat withdrawal history item.
type FiatWithdrawalHistoryItem struct {
	WithdrawalID    string       `json:"wdId"`
	ChannelID       string       `json:"chanId"`
	BillID          string       `json:"billId"`
	BankAccountName string       `json:"bankAcctName"`
	BankAcctNumber  string       `json:"bankAcctNum"`
	Amount          types.Number `json:"amt"`
	Fee             types.Number `json:"fee"`
	State           string       `json:"state"`
	Ccy             string       `json:"ccy"`
	CreationTime    okcoinTime   `json:"cTime"`
	UpdateTime      okcoinTime   `json:"uTime"`
}

// ChannelInfo represents a channel information
type ChannelInfo struct {
	ChannelID            string       `json:"chanId"`
	Currency             string       `json:"ccy"`
	DepositQuota         string       `json:"depQuota"`
	MinDeposit           types.Number `json:"minDep"`
	WithdrawalQuota      types.Number `json:"wdQuota"`
	MinWithdrawal        types.Number `json:"minWd"`
	UsedWithdrawalQuota  types.Number `json:"usedWdQuota"`
	ValidWithdrawalQuota types.Number `json:"validWdQuota"`
	BankAccountInfo      []struct {
		BankAccountName   string `json:"bankAcctName"`
		BankAccountNumber string `json:"bankAcctNum"`
		InstrumentName    string `json:"instName"`
		MaskAccountNumber string `json:"maskAcctNum"`
	} `json:"bankAcctInfo"`
}

// SubAccountInfo represents a single sub-account info.
type SubAccountInfo struct {
	SubAccountEnable  bool       `json:"enable"`
	SubAccountName    string     `json:"subAcct"`
	SubAccountType    string     `json:"type"`
	Label             string     `json:"label"`
	MobileNumber      string     `json:"mobile"`
	GoogleAuthEnabled bool       `json:"gAuth"`
	CanTransferOut    bool       `json:"canTransOut"`
	CreationTimestamp okcoinTime `json:"ts"`
}

// PlaceTradeOrderParam represents a trade order arguments.
type PlaceTradeOrderParam struct {
	InstrumentID   currency.Pair `json:"instId"`
	TradeMode      string        `json:"tdMode"` // Trade mode --> Margin mode: 'cross','isolated' Non-Margin mode: 'cash'
	ClientOrderID  string        `json:"clOrdId,omitempty"`
	Side           string        `json:"side"`                // Order side, buy sell
	OrderType      string        `json:"ordType"`             // Order type 'market': Market order 'limit': Limit order 'post_only': Post-only order 'fok': Fill-or-kill order 'ioc': Immediate-or-cancel order
	Price          float64       `json:"px,string,omitempty"` // Order price. Only applicable to limit,post_only,fok,ioc order.
	Size           float64       `json:"sz,string"`
	OrderTag       string        `json:"tag,omitempty"` // Order tag A combination of case-sensitive alphanumerics, all numbers, or all letters of up to 16 characters.
	BanAmend       bool          `json:"banAmend,omitempty"`
	TargetCurrency string        `json:"tgtCcy,omitempty"` // Whether the target currency uses the quote or base currency.

	// ExpiryTime is the request effective deadline.
	ExpiryTime int64 `json:"expTime,omitempty,string"`
}

// TradeOrderResponse represents a single trade order information
type TradeOrderResponse struct {
	ClientOrderID string `json:"clOrdId"`
	OrderID       string `json:"ordId"`
	Tag           string `json:"tag"`
	SCode         string `json:"sCode"`
	SMsg          string `json:"sMsg"`
}

// CancelTradeOrderRequest represents a cancel trade order request body
type CancelTradeOrderRequest struct {
	InstrumentID  string `json:"instId"`
	OrderID       string `json:"ordId,omitempty"`
	ClientOrderID string `json:"clOrdId,omitempty"`
}

// AmendTradeOrderRequestParam represents an order cancellation request parameter
type AmendTradeOrderRequestParam struct {
	OrderID                 string  `json:"ordId,omitempty"`
	InstrumentID            string  `json:"instId"`
	ClientOrderID           string  `json:"clOrdId,omitempty"`
	ClientRequestID         string  `json:"reqId,omitempty"`
	NewSize                 float64 `json:"newSz,string,omitempty"` // Conditional
	NewPrice                float64 `json:"newPx,string,omitempty"` // Conditional
	CancelOOrderIfAmendFail bool    `json:"cxlOnFail,omitempty"`    // whether the order needs to be automatically canceled when the order amendment fails
}

// AmendTradeOrderResponse represents a request parameter to amend an incomplete order.
type AmendTradeOrderResponse struct {
	ClientOrderID string `json:"clOrdId"`
	OrderID       string `json:"ordId"`
	RequestID     string `json:"reqId"`
	StatusCode    string `json:"sCode"`
	StatusMessage string `json:"sMsg"`
}

// TradeOrder represents a trade order detail
type TradeOrder struct {
	AccFillSize                types.Number `json:"accFillSz"`
	AveragePrice               types.Number `json:"avgPx"`
	CreationTime               okcoinTime   `json:"cTime"`
	Category                   string       `json:"category"`
	Currency                   string       `json:"ccy"`
	ClientOrdID                string       `json:"clOrdId"`
	Fee                        types.Number `json:"fee"`
	FeeCurrency                string       `json:"feeCcy"`
	FillPrice                  types.Number `json:"fillPx"`
	FillSize                   types.Number `json:"fillSz"`
	FillTime                   okcoinTime   `json:"fillTime"`
	InstrumentID               string       `json:"instId"`
	InstrumentType             string       `json:"instType"`
	Leverage                   types.Number `json:"lever"`
	OrderID                    string       `json:"ordId"`
	OrderType                  string       `json:"ordType"`
	ProfitAndLoss              types.Number `json:"pnl"`
	PosSide                    string       `json:"posSide"`
	Price                      types.Number `json:"px"`
	Rebate                     types.Number `json:"rebate"`
	RebateCurrency             string       `json:"rebateCcy"`
	ReduceOnly                 bool         `json:"reduceOnly,string"`
	Side                       string       `json:"side"`
	StopLossOrdPrice           types.Number `json:"slOrdPx"`
	StopLossTriggerPrice       types.Number `json:"slTriggerPx"`
	StopLossTriggerPriceType   string       `json:"slTriggerPxType"`
	Source                     string       `json:"source"`
	State                      string       `json:"state"`
	Size                       types.Number `json:"sz"`
	Tag                        string       `json:"tag"`
	TradeMode                  string       `json:"tdMode"`
	TargetCurrency             string       `json:"tgtCcy"`
	TakeProfitOrderPrice       types.Number `json:"tpOrdPx"`
	TakeProfitTriggerPrice     types.Number `json:"tpTriggerPx"`
	TakeProfitTriggerPriceType string       `json:"tpTriggerPxType"`
	TradeID                    string       `json:"tradeId"`
	UpdateTime                 okcoinTime   `json:"uTime"`
}

// TransactionFillItem represents recently filled transactions
type TransactionFillItem struct {
	InstrumentType string       `json:"instType"`
	InstrumentID   string       `json:"instId"`
	TradeID        string       `json:"tradeId"`
	OrderID        string       `json:"ordId"`
	ClientOrderID  string       `json:"clOrdId"`
	BillID         string       `json:"billId"`
	Tag            string       `json:"tag"`
	FillSize       types.Number `json:"fillSz"`
	FillPrice      types.Number `json:"fillPx"`
	Side           string       `json:"side"`
	PosSide        string       `json:"posSide"`
	ExecType       string       `json:"execType"`
	FeeCurrency    string       `json:"feeCcy"`
	Fee            types.Number `json:"fee"`
	Timestamp      okcoinTime   `json:"ts"`
}

// AlgoOrderRequestParam represents algo order request parameters.
type AlgoOrderRequestParam struct {
	InstrumentID             string  `json:"instId"`
	TradeMode                string  `json:"tdMode"`
	Side                     string  `json:"side"`
	OrderType                string  `json:"ordType"` // Order type'conditional': One-way stop order'oco': One-cancels-the-other order'trigger': Trigger order'move_order_stop': Trailing order'iceberg': Iceberg order'twap': TWAP order
	Size                     float64 `json:"sz,string"`
	TpTriggerPrice           float64 `json:"tpTriggerPx,string,omitempty"`
	TpOrderPrice             float64 `json:"tpOrdPx,string,omitempty"`
	TpTriggerOrderPriceType  string  `json:"tpTriggerPxType,omitempty"`
	StopLossTriggerPrice     float64 `json:"slTriggerPx,string,omitempty"`
	StopLossOrderPrice       float64 `json:"slOrdPx,string,omitempty"`
	StopLossTriggerPriceType string  `json:"slTriggerPxType,omitempty"`
	TargetCurrency           string  `json:"tgtCcy,omitempty"`
	Tag                      string  `json:"tag,omitempty"`
	ClientOrderID            string  `json:"clOrdId,omitempty"`

	// Trigger Order
	TriggerPrice     float64 `json:"triggerPx,omitempty,string"`
	OrderPrice       float64 `json:"orderPx,omitempty,string"`
	TriggerPriceType string  `json:"triggerPxType,omitempty"`

	// Trailing Stop Order
	CallbackRatio  float64 `json:"callbackRatio,string,omitempty"` // Either callbackRatio or callbackSpread is allowed to be passed.
	CallbackSpread string  `json:"callbackSpread"`
	ActivePrice    float64 `json:"activePx,string,omitempty"`

	// Iceberg Order
	PriceRatio  float64 `json:"pxVar,string,omitempty"`
	PriceSpread float64 `json:"pxSpread,string,omitempty"`
	SizeLimit   float64 `json:"szLimit,string,omitempty"` // Average amount
	PriceLimit  float64 `json:"pxLimit,string,omitempty"`

	TimeInterval string `json:"timeInterval,omitempty"`
}

// AlgoOrderResponse represents a response data for creating algo order.
type AlgoOrderResponse struct {
	AlgoID        string `json:"algoId"`
	ClientOrderID string `json:"clOrdId"`
	StatusCode    string `json:"sCode"`
	StatusMsg     string `json:"sMsg"`
}

// CancelAlgoOrderRequestParam represents a algo order cancellation request parameter
type CancelAlgoOrderRequestParam struct {
	AlgoOrderID  string `json:"algoId"`
	InstrumentID string `json:"instId"`
}

// AlgoOrderDetail represents an algo-order detailed information
type AlgoOrderDetail struct {
	ActivePrice              types.Number `json:"activePx"`
	ActualPrice              types.Number `json:"actualPx"`
	ActualSide               string       `json:"actualSide"`
	ActualSize               types.Number `json:"actualSz"`
	AlgoID                   string       `json:"algoId"`
	CreateTime               okcoinTime   `json:"cTime"`
	CallbackRatio            types.Number `json:"callbackRatio"`
	CallbackSpread           string       `json:"callbackSpread"`
	Currency                 string       `json:"ccy"`
	ClientOrderID            string       `json:"clOrdId"`
	InstrumentID             string       `json:"instId"`
	InstrumentType           string       `json:"instType"`
	Leverage                 types.Number `json:"lever"`
	MoveTriggerPrice         types.Number `json:"moveTriggerPx"`
	OrderID                  string       `json:"ordId"`
	OrdPrice                 types.Number `json:"ordPx"`
	OrderType                string       `json:"ordType"`
	PosSide                  string       `json:"posSide"`
	PriceLimit               types.Number `json:"pxLimit"`
	PriceSpread              types.Number `json:"pxSpread"`
	PriceVar                 types.Number `json:"pxVar"`
	Side                     string       `json:"side"`
	StopLossOrdPrice         types.Number `json:"slOrdPx"`
	StopLossTriggerPrice     types.Number `json:"slTriggerPx"`
	StopLossTriggerPriceType string       `json:"slTriggerPxType"`
	State                    string       `json:"state"`
	Size                     types.Number `json:"sz"`
	SizeLimit                types.Number `json:"szLimit"`
	Tag                      string       `json:"tag"`
	TdMode                   string       `json:"tdMode"`
	TgtCcy                   string       `json:"tgtCcy"`
	TimeInterval             string       `json:"timeInterval"`
	TpOrdPrice               types.Number `json:"tpOrdPx"`
	TpTriggerPrice           types.Number `json:"tpTriggerPx"`
	TpTriggerPriceType       string       `json:"tpTriggerPxType"`
	TriggerPrice             types.Number `json:"triggerPx"`
	TriggerPriceType         string       `json:"triggerPxType"`
	TriggerTime              okcoinTime   `json:"triggerTime"`
}

// SubAccountAPIKey retrieves sub-account API key.
type SubAccountAPIKey struct {
	Label        string     `json:"label"`
	APIKey       string     `json:"apiKey"`
	Permissions  string     `json:"perm"`
	LinkedIP     string     `json:"ip"`
	CreationTime okcoinTime `json:"ts"`
}

// SubAccountTradingBalance represents a sub-account trading detail.
type SubAccountTradingBalance struct {
	AdjEq   types.Number `json:"adjEq"` // Adjusted / Effective equity in USD. Not enabled. Please disregard.
	Details []struct {
		AvailableBal            types.Number `json:"availBal"`
		AvailableEquity         types.Number `json:"availEq"`
		CashBalance             types.Number `json:"cashBal"`
		Currency                string       `json:"ccy"`
		CrossLiab               types.Number `json:"crossLiab"`
		DiscountEquity          types.Number `json:"disEq"`
		Equity                  types.Number `json:"eq"`
		EquityUSD               types.Number `json:"eqUsd"`
		FrozenBalance           types.Number `json:"frozenBal"`
		Interest                types.Number `json:"interest"`
		IsolatedEquity          types.Number `json:"isoEq"`
		IsolatedLiability       types.Number `json:"isoLiab"`
		Liability               types.Number `json:"liab"`
		MaxLoan                 types.Number `json:"maxLoan"`
		MarginRatio             types.Number `json:"mgnRatio"`
		NotionalLeverage        types.Number `json:"notionalLever"`
		OrderMarginFrozen       types.Number `json:"ordFrozen"`
		Twap                    types.Number `json:"twap"`
		UTime                   types.Number `json:"uTime"`
		UnrealizedProfitAndLoss types.Number `json:"upl"`
		UPLLiability            types.Number `json:"uplLiab"`
	} `json:"details"`
	IMR            types.Number `json:"imr"` // Frozen equity for open positions and pending orders in USD. Not enabled. Please disregard.
	IsolatedEquity types.Number `json:"isoEq"`
	MarginRatio    types.Number `json:"mgnRatio"`
	MMR            types.Number `json:"mmr"` // Maintenance margin requirement in USD. Not enabled. Please disregard
	NotionalUsd    types.Number `json:"notionalUsd"`
	OrdFrozen      types.Number `json:"ordFroz"`
	TotalEquity    types.Number `json:"totalEq"`
	UpdatedTime    okcoinTime   `json:"uTime"`
}

// SubAccountFundingBalance represents a sub-account funding balance for a currency.
type SubAccountFundingBalance struct {
	AvailBal  types.Number `json:"availBal"`
	Bal       types.Number `json:"bal"`
	Currency  string       `json:"ccy"`
	FrozenBal types.Number `json:"frozenBal"`
}

// SubAccountTransferInfo represents a sub-account transfer information.
type SubAccountTransferInfo struct {
	BillID            string       `json:"billId"`
	Type              string       `json:"type"`
	Ccy               string       `json:"ccy"`
	Amount            types.Number `json:"amt"`
	SubAccount        string       `json:"subAcct"`
	CreationTimestamp okcoinTime   `json:"ts"`
}

// SubAccountTransferResponse represents a transfer operation response.
type SubAccountTransferResponse struct {
	TransferID string `json:"transId"`
}

// IntraAccountTransferParam represents an intra account transfer request parameters.
type IntraAccountTransferParam struct {
	Ccy            string  `json:"ccy"`
	Amount         float64 `json:"amt,string"`
	From           string  `json:"from"`
	To             string  `json:"to"`
	FromSubAccount string  `json:"fromSubAccount"`
	ToSubAccount   string  `json:"toSubAccount"`
}
