package okcoin

import (
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
)

// TickerData stores ticker data
type TickerData struct {
	InstType        string         `json:"instType"`
	InstrumentID    string         `json:"instId"`
	LastTradedPrice float64        `json:"last,string"`
	LastTradedSize  float64        `json:"lastSz,string"`
	BestAskPrice    float64        `json:"askPx,string"`
	BestAskSize     float64        `json:"askSz,string"`
	BestBidPrice    float64        `json:"bidPx,string"`
	BestBidSize     float64        `json:"bidSz,string"`
	Open24H         float64        `json:"open24h,string"`   // Open price in the past 24 hours
	High24H         float64        `json:"high24h,string"`   // Highest price in the past 24 hours
	Low24H          float64        `json:"low24h,string"`    // Lowest price in the past 24 hours
	VolCcy24H       float64        `json:"volCcy24h,string"` // 24h trading volume, with a unit of currency. The value is the quantity in quote currency.
	Vol24H          float64        `json:"vol24h,string"`    // 24h trading volume, with a unit of contract. The value is the quantity in base currency.
	Timestamp       okcoinMilliSec `json:"ts"`
	OpenPriceInUtc0 float64        `json:"sodUtc0,string"`
	OpenPriceInUtc8 float64        `json:"sodUtc8,string"`
}

// GetOrderBookResponse response data
type GetOrderBookResponse struct {
	Timestamp okcoinMilliSec `json:"ts"`
	Asks      [][4]string    `json:"asks"` // [[0]: "Price", [1]: "Size", [2]: "Num_orders"], ...
	Bids      [][4]string    `json:"bids"` // [[0]: "Price", [1]: "Size", [2]: "Num_orders"], ...
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
		Channel string `json:"channel"`
		InstID  string `json:"instId"`
	} `json:"arg"`
	Action string               `json:"action"`
	Data   []WebsocketOrderBook `json:"data"`
}

// WebsocketOrderBook holds orderbook data
type WebsocketOrderBook struct {
	Checksum  int64            `json:"checksum"`
	Asks      [][]okcoinNumber `json:"asks"` // [ Price, Quantity, depreciated, number of orders at the price ]
	Bids      [][]okcoinNumber `json:"bids"` // [ Price, Quantity, depreciated, number of orders at the price ]
	Timestamp okcoinMilliSec   `json:"ts"`
}

func (a *WebsocketOrderBook) prepareOrderbook() {
	asks := [][]okcoinNumber{}
askLoop:
	for x := range a.Asks {
		if a.Asks[x][1].Float64() != 0 {
			for i := 0; i < len(asks); i++ {
				if asks[i][0].Float64() == a.Asks[x][0].Float64() {
					if asks[i][1].Float64() > a.Asks[x][1].Float64() {
						continue askLoop
					}
					asks[i][1] = a.Asks[x][1]
					continue askLoop
				}
			}
			asks = append(asks, a.Asks[x])
		}
	}
	a.Asks = asks
	bids := [][]okcoinNumber{}
bidsLoop:
	for x := range a.Bids {
		if a.Bids[x][1].Float64() != 0 {
			for i := 0; i < len(bids); i++ {
				if bids[i][0].Float64() == a.Bids[x][0].Float64() {
					if bids[i][1].Float64() > a.Bids[x][1].Float64() {
						continue bidsLoop
					}
					bids[i][1] = a.Bids[x][1]
					continue bidsLoop
				}
			}
			bids = append(bids, a.Bids[x])
		}
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
		Title                 string         `json:"title"`
		State                 string         `json:"state"`
		End                   okcoinMilliSec `json:"end"`
		Begin                 okcoinMilliSec `json:"begin"`
		Href                  string         `json:"href"`
		ServiceType           string         `json:"serviceType"`
		System                string         `json:"system"`
		RescheduleDescription string         `json:"scheDesc"`
		Time                  okcoinMilliSec `json:"ts"`
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
			AvailableBalance   okcoinNumber   `json:"availBal"`
			AvailableEquity    string         `json:"availEq"`
			CashBalance        string         `json:"cashBal"`
			Currency           string         `json:"ccy"`
			CoinUsdPrice       string         `json:"coinUsdPrice"`
			CrossLiab          string         `json:"crossLiab"`
			DiscountEquity     string         `json:"disEq"`
			Equity             string         `json:"eq"`
			EquityUsd          string         `json:"eqUsd"`
			FixedBalance       string         `json:"fixedBal"`
			FrozenBalance      string         `json:"frozenBal"`
			Interest           string         `json:"interest"`
			IsoEquity          string         `json:"isoEq"`
			IsoLiability       string         `json:"isoLiab"`
			IsoUpl             string         `json:"isoUpl"`
			Liability          string         `json:"liab"`
			MaxLoan            string         `json:"maxLoan"`
			MgnRatio           string         `json:"mgnRatio"`
			NotionalLeverage   string         `json:"notionalLever"`
			MarginFrozenOrders string         `json:"ordFrozen"`
			SpotInUseAmount    string         `json:"spotInUseAmt"`
			StrategyEquity     string         `json:"stgyEq"`
			Twap               string         `json:"twap"`
			UPL                string         `json:"upl"` // Unrealized profit and loss
			UpdateTime         okcoinMilliSec `json:"uTime"`
		} `json:"details"`
		FrozenEquity                 string         `json:"imr"`
		IsoEquity                    string         `json:"isoEq"`
		MarginRatio                  string         `json:"mgnRatio"`
		MaintenanceMarginRequirement string         `json:"mmr"`
		NotionalUsd                  string         `json:"notionalUsd"`
		MarginOrderFrozen            string         `json:"ordFroz"`
		TotalEquity                  string         `json:"totalEq"`
		UpdateTime                   okcoinMilliSec `json:"uTime"`
	} `json:"data"`
}

// WebsocketOrder represents and order information. Data will not be pushed when first subscribed.
type WebsocketOrder struct {
	Arg struct {
		Channel        string `json:"channel"`
		InstrumentType string `json:"instType"`
		InstID         string `json:"instId"`
		UID            string `json:"uid"`
	} `json:"arg"`
	Data []struct {
		AccFillSize                float64        `json:"accFillSz"`
		AmendResult                string         `json:"amendResult"`
		AveragePrice               float64        `json:"avgPx"`
		CreateTime                 okcoinMilliSec `json:"cTime"`
		Category                   string         `json:"category"`
		Currency                   string         `json:"ccy"`
		ClientOrdID                string         `json:"clOrdId"`
		Code                       string         `json:"code"`
		ExecType                   string         `json:"execType"`
		Fee                        float64        `json:"fee"`
		FeeCurrency                string         `json:"feeCcy"`
		FillFee                    string         `json:"fillFee"`
		FillFeeCurrency            string         `json:"fillFeeCcy"`
		FillNotionalUsd            string         `json:"fillNotionalUsd"`
		FillPrice                  float64        `json:"fillPx"`
		FillSize                   float64        `json:"fillSz"`
		FillTime                   okcoinMilliSec `json:"fillTime"`
		InstrumentID               string         `json:"instId"`
		InstrumentType             string         `json:"instType"`
		Leverage                   string         `json:"lever"`
		ErrorMessage               string         `json:"msg"`
		NotionalUsd                string         `json:"notionalUsd"`
		OrderID                    string         `json:"ordId"`
		OrderType                  string         `json:"ordType"`
		ProfitAndLoss              string         `json:"pnl"`
		PositionSide               string         `json:"posSide"`
		Price                      string         `json:"px"`
		Rebate                     string         `json:"rebate"`
		RebateCurrency             string         `json:"rebateCcy"`
		ReduceOnly                 string         `json:"reduceOnly"`
		ClientRequestID            string         `json:"reqId"`
		Side                       string         `json:"side"`
		StopLossOrderPrice         float64        `json:"slOrdPx"`
		StopLossTriggerPrice       float64        `json:"slTriggerPx"`
		StopLossTriggerPriceType   string         `json:"slTriggerPxType"`
		Source                     string         `json:"source"`
		State                      string         `json:"state"`
		Size                       float64        `json:"sz"`
		Tag                        string         `json:"tag"`
		TradeMode                  string         `json:"tdMode"`
		TargetCurrency             string         `json:"tgtCcy"`
		TakeProfitOrdPrice         float64        `json:"tpOrdPx"`
		TakeProfitTriggerPrice     float64        `json:"tpTriggerPx"`
		TakeProfitTriggerPriceType string         `json:"tpTriggerPxType"`
		TradeID                    string         `json:"tradeId"`
		UTime                      okcoinMilliSec `json:"uTime"`
	} `json:"data"`
}

// WebsocketAlgoOrder represents algo orders (includes trigger order, oco order, conditional order).
type WebsocketAlgoOrder struct {
	Arg struct {
		Channel  string `json:"channel"`
		UID      string `json:"uid"`
		InstType string `json:"instType"`
		InstID   string `json:"instId"`
	} `json:"arg"`
	Data []struct {
		InstrumentType             string         `json:"instType"`
		InstrumentID               string         `json:"instId"`
		OrderID                    string         `json:"ordId"`
		Currency                   string         `json:"ccy"`
		ClientOrderID              string         `json:"clOrdId"`
		AlgoID                     string         `json:"algoId"`
		Price                      float64        `json:"px,string"`
		Size                       float64        `json:"sz,string"`
		TradeMode                  string         `json:"tdMode"`
		TgtCurrency                string         `json:"tgtCcy"`
		NotionalUsd                string         `json:"notionalUsd"`
		OrderType                  string         `json:"ordType"`
		Side                       string         `json:"side"`
		PositionSide               string         `json:"posSide"`
		State                      string         `json:"state"`
		Leverage                   float64        `json:"lever"`
		TakeProfitTriggerPrice     float64        `json:"tpTriggerPx,string"`
		TakeProfitTriggerPriceType string         `json:"tpTriggerPxType"`
		TakeProfitOrdPrice         float64        `json:"tpOrdPx,string"`
		SlTriggerPrice             float64        `json:"slTriggerPx,string"`
		SlTriggerPriceType         string         `json:"slTriggerPxType"`
		TriggerPxType              string         `json:"triggerPxType"`
		TriggerPrice               float64        `json:"triggerPx,string"`
		OrderPrice                 float64        `json:"ordPx,string"`
		Tag                        string         `json:"tag"`
		ActualSize                 float64        `json:"actualSz,string"`
		ActualPrice                float64        `json:"actualPx,string"`
		ActualSide                 string         `json:"actualSide"`
		TriggerTime                okcoinMilliSec `json:"triggerTime"`
		CreateTime                 okcoinMilliSec `json:"cTime"`
	} `json:"data"`
}

// WebsocketAdvancedAlgoOrder represents advance algo orders (including Iceberg order, TWAP order, Trailing order).
type WebsocketAdvancedAlgoOrder struct {
	Arg struct {
		Channel  string `json:"channel"`
		UID      string `json:"uid"`
		InstType string `json:"instType"`
		InstID   string `json:"instId"`
	} `json:"arg"`
	Data []struct {
		ActualPx       string `json:"actualPx"`
		ActualSide     string `json:"actualSide"`
		ActualSz       string `json:"actualSz"`
		AlgoID         string `json:"algoId"`
		CTime          string `json:"cTime"`
		Ccy            string `json:"ccy"`
		ClOrdID        string `json:"clOrdId"`
		Count          string `json:"count"`
		InstID         string `json:"instId"`
		InstType       string `json:"instType"`
		Lever          string `json:"lever"`
		NotionalUsd    string `json:"notionalUsd"`
		OrdPx          string `json:"ordPx"`
		OrdType        string `json:"ordType"`
		PTime          string `json:"pTime"`
		PosSide        string `json:"posSide"`
		PxLimit        string `json:"pxLimit"`
		PxSpread       string `json:"pxSpread"`
		PxVar          string `json:"pxVar"`
		Side           string `json:"side"`
		SlOrdPx        string `json:"slOrdPx"`
		SlTriggerPx    string `json:"slTriggerPx"`
		State          string `json:"state"`
		Sz             string `json:"sz"`
		SzLimit        string `json:"szLimit"`
		TdMode         string `json:"tdMode"`
		TimeInterval   string `json:"timeInterval"`
		TpOrdPx        string `json:"tpOrdPx"`
		TpTriggerPx    string `json:"tpTriggerPx"`
		Tag            string `json:"tag"`
		TriggerPx      string `json:"triggerPx"`
		TriggerTime    string `json:"triggerTime"`
		CallbackRatio  string `json:"callbackRatio"`
		CallbackSpread string `json:"callbackSpread"`
		ActivePx       string `json:"activePx"`
		MoveTriggerPx  string `json:"moveTriggerPx"`
	} `json:"data"`
}

// WebsocketInstrumentData contains formatted data for instruments related websocket responses
type WebsocketInstrumentData struct {
	Alias                 string         `json:"alias"`
	BaseCurrency          string         `json:"baseCcy"`
	Category              string         `json:"category"`
	ContractMultiplier    string         `json:"ctMult"`
	ContractType          string         `json:"ctType"`
	ContractValue         string         `json:"ctVal"`
	ContractValueCurrency string         `json:"ctValCcy"`
	ExpiryTime            okcoinMilliSec `json:"expTime"`
	InstrumentFamily      string         `json:"instFamily"`
	InstrumentID          string         `json:"instId"`
	InstrumentType        string         `json:"instType"`
	Leverage              string         `json:"lever"`
	ListTime              okcoinMilliSec `json:"listTime"`
	LotSize               string         `json:"lotSz"`
	MaxIcebergSize        float64        `json:"maxIcebergSz,string"`
	MaxLimitSize          float64        `json:"maxLmtSz,string"`
	MaxMarketSize         float64        `json:"maxMktSz,string"`
	MaxStopSize           float64        `json:"maxStopSz,string"`
	MaxTriggerSize        float64        `json:"maxTriggerSz,string"`
	MaxTwapSize           float64        `json:"maxTwapSz,string"`
	MinimumOrderSize      float64        `json:"minSz,string"`
	OptionType            string         `json:"optType"`
	QuoteCurrency         string         `json:"quoteCcy"`
	SettleCurrency        string         `json:"settleCcy"`
	State                 string         `json:"state"`
	StrikePrice           string         `json:"stk"`
	TickSize              float64        `json:"tickSz,string"`
	Underlying            string         `json:"uly"`
}

// WsTickerData contains formatted data for ticker related websocket responses
type WsTickerData struct {
	InstrumentType string         `json:"instType"`
	InstrumentID   string         `json:"instId"`
	Last           float64        `json:"last,string"`
	LastSize       float64        `json:"lastSz,string"`
	AskPrice       float64        `json:"askPx,string"`
	AskSize        float64        `json:"askSz,string"`
	BidPrice       float64        `json:"bidPx,string"`
	BidSize        float64        `json:"bidSz,string"`
	Open24H        float64        `json:"open24h,string"`
	High24H        float64        `json:"high24h,string"`
	Low24H         float64        `json:"low24h,string"`
	SodUtc0        string         `json:"sodUtc0"`
	SodUtc8        string         `json:"sodUtc8"`
	VolCcy24H      float64        `json:"volCcy24h,string"`
	Vol24H         float64        `json:"vol24h,string"`
	Timestamp      okcoinMilliSec `json:"ts"`
}

// WebsocketTradeResponse contains formatted data for trade related websocket responses
type WebsocketTradeResponse struct {
	Arg struct {
		Channel      string `json:"channel"`
		InstrumentID string `json:"instId"`
	} `json:"arg"`
	Data []struct {
		InstrumentID string         `json:"instId"`
		TradeID      string         `json:"tradeId"`
		Price        float64        `json:"px,string"`
		Size         float64        `json:"sz,string"`
		Side         string         `json:"side"`
		Timestamp    okcoinMilliSec `json:"ts"`
	} `json:"data"`
}

// WebsocketCandlesResponse represents a candlestick response data.
type WebsocketCandlesResponse struct {
	Arg struct {
		Channel string `json:"channel"`
		InstID  string `json:"instId"`
	} `json:"arg"`
	Data [][]string `json:"data"`
}

// GetCandlesData represents a candlestick instances list.
func (a *WebsocketCandlesResponse) GetCandlesData(exchangeName string) ([]stream.KlineData, error) {
	candlesticks := make([]stream.KlineData, len(a.Data))
	cp, err := currency.NewPairFromString(a.Arg.InstID)
	if err != nil {
		return nil, err
	}
	for x := range a.Data {
		var timestamp int64
		timestamp, err = strconv.ParseInt(a.Data[x][0], 10, 64)
		if err != nil {
			return nil, err
		}
		candlesticks[x] = stream.KlineData{
			AssetType: asset.Spot,
			Pair:      cp,
			Timestamp: time.UnixMilli(timestamp),
			Exchange:  exchangeName,
		}
		candlesticks[x].OpenPrice, err = strconv.ParseFloat(a.Data[x][1], 64)
		if err != nil {
			return nil, err
		}
		candlesticks[x].HighPrice, err = strconv.ParseFloat(a.Data[x][2], 64)
		if err != nil {
			return nil, err
		}
		candlesticks[x].LowPrice, err = strconv.ParseFloat(a.Data[x][3], 64)
		if err != nil {
			return nil, err
		}
		candlesticks[x].ClosePrice, err = strconv.ParseFloat(a.Data[x][4], 64)
		if err != nil {
			return nil, err
		}
		candlesticks[x].Volume, err = strconv.ParseFloat(a.Data[x][5], 64)
		if err != nil {
			return nil, err
		}
	}
	return candlesticks, nil
}

// WebsocketOrderBooksData is the full websocket response containing orderbook data
type WebsocketOrderBooksData struct {
	Table  string               `json:"table"`
	Action string               `json:"action"`
	Data   []WebsocketOrderBook `json:"data"`
}

// WebsocketUserSwapPositionResponse contains formatted data for user position data
type WebsocketUserSwapPositionResponse struct {
	InstrumentID string                                 `json:"instrument_id"`
	Timestamp    time.Time                              `json:"timestamp,omitempty"`
	Holding      []WebsocketUserSwapPositionHoldingData `json:"holding,omitempty"`
}

// WebsocketUserSwapPositionHoldingData contains formatted data for user position holding data
type WebsocketUserSwapPositionHoldingData struct {
	AvailablePosition float64   `json:"avail_position,string,omitempty"`
	AverageCost       float64   `json:"avg_cost,string,omitempty"`
	Leverage          float64   `json:"leverage,string,omitempty"`
	LiquidationPrice  float64   `json:"liquidation_price,string,omitempty"`
	Margin            float64   `json:"margin,string,omitempty"`
	Position          float64   `json:"position,string,omitempty"`
	RealizedPnl       float64   `json:"realized_pnl,string,omitempty"`
	SettlementPrice   float64   `json:"settlement_price,string,omitempty"`
	Side              string    `json:"side,omitempty"`
	Timestamp         time.Time `json:"timestamp,omitempty"`
}

// WebsocketSpotOrderResponse contains formatted data for spot user orders
type WebsocketSpotOrderResponse struct {
	Table string `json:"table"`
	Data  []struct {
		ClientOid      string    `json:"client_oid"`
		CreatedAt      time.Time `json:"created_at"`
		FilledNotional float64   `json:"filled_notional,string"`
		FilledSize     float64   `json:"filled_size,string"`
		InstrumentID   string    `json:"instrument_id"`
		LastFillPx     float64   `json:"last_fill_px,string"`
		LastFillQty    float64   `json:"last_fill_qty,string"`
		LastFillTime   time.Time `json:"last_fill_time"`
		MarginTrading  int64     `json:"margin_trading,string"`
		Notional       string    `json:"notional"`
		OrderID        string    `json:"order_id"`
		OrderType      int64     `json:"order_type,string"`
		Price          float64   `json:"price,string"`
		Side           string    `json:"side"`
		Size           float64   `json:"size,string"`
		State          int64     `json:"state,string"`
		Status         string    `json:"status"`
		Timestamp      time.Time `json:"timestamp"`
		Type           string    `json:"type"`
	} `json:"data"`
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
	Title       string         `json:"title"`
	State       string         `json:"state"`
	Begin       okcoinMilliSec `json:"begin"`
	End         okcoinMilliSec `json:"end"`
	Href        string         `json:"href"`
	ServiceType string         `json:"serviceType"`
	System      string         `json:"system"`
	ScheDesc    string         `json:"scheDesc"`
}

// Instrument represents an instrument in an open contract.
type Instrument struct {
	Alias          string         `json:"alias"`
	BaseCurrency   string         `json:"baseCcy"`
	Category       string         `json:"category"`
	CtMult         string         `json:"ctMult"`
	CtType         string         `json:"ctType"`
	CtVal          string         `json:"ctVal"`
	CtValCurrency  string         `json:"ctValCcy"`
	ExpTime        okcoinMilliSec `json:"expTime"`
	InstFamily     string         `json:"instFamily"`
	InstrumentID   string         `json:"instId"`
	InstrumentType string         `json:"instType"`
	Leverage       string         `json:"lever"`
	ListTime       okcoinMilliSec `json:"listTime"`
	LotSize        string         `json:"lotSz"`
	MaxIcebergSz   okcoinNumber   `json:"maxIcebergSz"`
	MaxLimitSize   okcoinNumber   `json:"maxLmtSz"`
	MaxMarketSize  okcoinNumber   `json:"maxMktSz"`
	MaxStopSize    okcoinNumber   `json:"maxStopSz"`
	MaxTwapSize    okcoinNumber   `json:"maxTwapSz"`
	MaxTriggerSize okcoinNumber   `json:"maxTriggerSz"`
	MinSize        okcoinNumber   `json:"minSz"`
	QuoteCurrency  string         `json:"quoteCcy"`
	OptionType     string         `json:"optType"`
	SettleCurrency string         `json:"settleCcy"`
	State          string         `json:"state"`
	StrikePrice    okcoinNumber   `json:"stk"`
	TickSize       okcoinNumber   `json:"tickSz"`
	Underlying     string         `json:"uly"`
}

type candlestickItemResponse [9]string

// CandlestickData represents the candlestick chart
type CandlestickData struct {
	Timestamp            okcoinMilliSec
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
	InstID     string         `json:"instId"`
	Side       string         `json:"side"`
	TradeSize  float64        `json:"sz,string"`
	TradePrice float64        `json:"px,string"`
	TradeID    string         `json:"tradeId"`
	Timestamp  okcoinMilliSec `json:"ts"`
}

// TradingVolume represents the trading volume of the platform in 24 hours
type TradingVolume struct {
	VolCny    float64        `json:"volCny,string"`
	VolUsd    float64        `json:"volUsd,string"`
	Timestamp okcoinMilliSec `json:"ts"`
}

// Oracle represents crypto price of signing using Open Oracle smart contract.
type Oracle []struct {
	Messages   []string          `json:"messages"`
	Prices     map[string]string `json:"prices"`
	Signatures []string          `json:"signatures"`
	Timestamp  okcoinMilliSec    `json:"timestamp"`
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
	CanDep                     bool    `json:"canDep"`
	CanInternal                bool    `json:"canInternal"`
	CanWd                      bool    `json:"canWd"`
	Currency                   string  `json:"ccy"`
	Chain                      string  `json:"chain"`
	DepQuotaFixed              string  `json:"depQuotaFixed"`
	DepQuoteDailyLayer2        string  `json:"depQuoteDailyLayer2"`
	LogoLink                   string  `json:"logoLink"`
	MainNet                    bool    `json:"mainNet"`
	MaxFee                     float64 `json:"maxFee,string"`
	MaxWithdrawal              float64 `json:"maxWd,string"`
	MinDeposit                 float64 `json:"minDep,string"`
	MinDepArrivalConfirm       string  `json:"minDepArrivalConfirm"`
	MinFee                     float64 `json:"minFee,string"`
	MinWithdrawal              float64 `json:"minWd,string"`
	MinWithdrawalUnlockConfirm string  `json:"minWdUnlockConfirm"`
	Name                       string  `json:"name"`
	NeedTag                    bool    `json:"needTag"`
	UsedDepQuotaFixed          string  `json:"usedDepQuotaFixed"`
	UsedWdQuota                string  `json:"usedWdQuota"`
	WithdrawalQuota            string  `json:"wdQuota"`
	WithdrawalTickSize         float64 `json:"wdTickSz,string"`
}

// CurrencyBalance represents a currency balance information.
type CurrencyBalance struct {
	AvailableBalance float64 `json:"availBal,string"`
	Balance          float64 `json:"bal,string"`
	Currency         string  `json:"ccy"`
	FrozenBalance    float64 `json:"frozenBal,string"`
}

// AccountAssetValuation represents account asset valuation
type AccountAssetValuation struct {
	Details struct {
		Classic float64 `json:"classic,string"`
		Earn    float64 `json:"earn,string"`
		Funding float64 `json:"funding,string"`
		Trading float64 `json:"trading,string"`
	} `json:"details"`
	TotalBalance float64        `json:"totalBal,string"`
	Timestamp    okcoinMilliSec `json:"ts"`
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
	TransferID string  `json:"transId"`
	Currency   string  `json:"ccy"`
	ClientID   string  `json:"clientId"`
	From       string  `json:"from"`
	Amount     float64 `json:"amt,string"`
	InstID     string  `json:"instId"`
	State      string  `json:"state"`
	SubAcct    string  `json:"subAcct"`
	To         string  `json:"to"`
	ToInstID   string  `json:"toInstId"`
	Type       string  `json:"type"`
}

// AssetBillDetail represents the billing record.
type AssetBillDetail struct {
	BillID    string         `json:"billId"`
	Currency  string         `json:"ccy"`
	ClientID  string         `json:"clientId"`
	BalChange float64        `json:"balChg,string"`
	Balance   float64        `json:"bal,string"`
	Type      string         `json:"type"`
	Timestamp okcoinMilliSec `json:"ts"`
}

// LightningDepositDetail represents a lightning deposit instance detail
type LightningDepositDetail struct {
	CreationTime okcoinMilliSec `json:"cTime"`
	Invoice      string         `json:"invoice"`
}

// DepositAddress represents a currency deposit address detail
type DepositAddress struct {
	Chain                     string `json:"chain"`
	ContractAddr              string `json:"ctAddr"`
	Ccy                       string `json:"ccy"`
	To                        string `json:"to"`
	Address                   string `json:"addr"`
	Selected                  bool   `json:"selected"`
	Tag                       string `json:"tag"`
	Memo                      string `json:"memo"`
	DepositPaymentID          string `json:"pmtId"`
	DepositAddressAttachement string `json:"addrEx"`
}

// DepositHistoryItem represents deposit records according to the currency, deposit status, and time range in reverse chronological order.
type DepositHistoryItem struct {
	ActualDepBlkConfirm string         `json:"actualDepBlkConfirm"` // ActualDepBlkConfirm actual amount of blockchain confirm in a single deposit
	Amount              float64        `json:"amt,string"`
	Currency            string         `json:"ccy"`
	Chain               string         `json:"chain"`
	DepositID           string         `json:"depId"`
	From                string         `json:"from"`
	State               string         `json:"state"`
	To                  string         `json:"to"`
	Timestamp           okcoinMilliSec `json:"ts"`
	TransactionID       string         `json:"txId"`
}

// WithdrawalRequest represents withdrawal of tokens request.
type WithdrawalRequest struct {
	Amount           float64       `json:"amt,string,omitempty"`
	TransactionFee   float64       `json:"fee,string,omitempty"`
	WithdrawalMethod string        `json:"dest,omitempty"` // Withdrawal method 3: internal  4: on chain
	Ccy              currency.Code `json:"ccy,omitempty"`
	Chain            string        `json:"chain,omitempty"`
	ClientID         string        `json:"clientId,omitempty"`
	ToAddress        string        `json:"toAddr,omitempty"`
}

// WithdrawalResponse represents withdrawal of tokens response.
type WithdrawalResponse struct {
	Amt      string `json:"amt"`
	WdID     string `json:"wdId"`
	Ccy      string `json:"ccy"`
	ClientID string `json:"clientId"`
	Chain    string `json:"chain"`
}

// LightningWithdrawalsRequest represents lightning withdrawal request params
type LightningWithdrawalsRequest struct {
	Ccy     currency.Code `json:"ccy"`
	Invoice string        `json:"invoice"`
	Memo    string        `json:"memo,omitempty"`
}

// LightningWithdrawals the minimum withdrawal amount is approximately 0.000001 BTC. Sub-account does not support withdrawal.
type LightningWithdrawals struct {
	WithdrawalID string         `json:"wdId"`
	CreationTime okcoinMilliSec `json:"cTime"`
}

// WithdrawalCancellation represents a request parameter for withdrawal cancellation
type WithdrawalCancellation struct {
	WithdrawalID string `json:"wdId"`
}

// WithdrawalOrderItem represents a withdrawal instance item
type WithdrawalOrderItem struct {
	Chain         string         `json:"chain"`
	Fee           float64        `json:"fee,string"`
	Ccy           string         `json:"ccy"`
	ClientID      string         `json:"clientId"`
	Amt           float64        `json:"amt,string"`
	TransactionID string         `json:"txId"`
	From          string         `json:"from"`
	To            string         `json:"to"`
	State         string         `json:"state"`
	Timestamp     okcoinMilliSec `json:"ts"`
	WithdrawalID  string         `json:"wdId"`
}

// AccountBalanceInformation represents currency balance information.
type AccountBalanceInformation struct {
	AdjustedEquity string `json:"adjEq"` // Adjusted / Effective equity in USD . Not enabled. Please disregard.
	Details        []struct {
		AvailableBalance                 okcoinNumber   `json:"availBal"`
		AvaileEquity                     string         `json:"availEq"`
		CashBalance                      okcoinNumber   `json:"cashBal"`
		Currency                         string         `json:"ccy"`
		CrossLiability                   string         `json:"crossLiab"`
		DiscountEqutity                  string         `json:"disEq"`
		Equity                           string         `json:"eq"`
		EquityUsd                        string         `json:"eqUsd"`
		FixedBalance                     okcoinNumber   `json:"fixedBal"`
		FrozenBalance                    okcoinNumber   `json:"frozenBal"`
		Interest                         okcoinNumber   `json:"interest"`
		IsolatedEquity                   okcoinNumber   `json:"isoEq"`
		IsolatedLiability                okcoinNumber   `json:"isoLiab"`
		IsolatedUpl                      string         `json:"isoUpl"` // Isolated unrealized profit and loss of the currency. Not enabled. Please disregard.
		Liability                        string         `json:"liab"`
		MaxLoan                          string         `json:"maxLoan"`
		MarginRatio                      string         `json:"mgnRatio"`
		NotionalLever                    string         `json:"notionalLever"`
		OrdFrozen                        string         `json:"ordFrozen"`
		SpotInUseAmount                  string         `json:"spotInUseAmt"`
		StrategyEquity                   string         `json:"stgyEq"`
		Twap                             string         `json:"twap"`
		UpdateTime                       okcoinMilliSec `json:"uTime"`
		UnrealizedProfitAndLoss          string         `json:"upl"`
		UnrealizedProfitAndLossLiability string         `json:"uplLiab"`
	} `json:"details"`
	IMR             string         `json:"imr"` // Frozen equity for open positions and pending orders in USD.
	IsolatedEqutity string         `json:"isoEq"`
	MarginRatio     okcoinNumber   `json:"mgnRatio"`
	Mmr             string         `json:"mmr"` // Maintenance margin requirement in USD.
	NotionalUsd     string         `json:"notionalUsd"`
	OrdFroz         string         `json:"ordFroz"`
	TotalEq         string         `json:"totalEq"`
	UpdateTime      okcoinMilliSec `json:"uTime"`
}

// BillsDetail represents a bill
type BillsDetail struct {
	Balance          float64        `json:"bal,string"`
	BalanceChange    float64        `json:"balChg,string"`
	BillID           string         `json:"billId"`
	Currency         string         `json:"ccy"`
	ExecType         string         `json:"execType"`
	Fee              okcoinNumber   `json:"fee"`
	From             string         `json:"from"`
	InstrumentID     string         `json:"instId"`
	InstrumentType   string         `json:"instType"`
	MarginMode       string         `json:"mgnMode"`
	Notes            string         `json:"notes"`
	OrderID          string         `json:"ordId"`
	ProfitAndLoss    string         `json:"pnl"`
	PosBalance       float64        `json:"posBal,string"`
	PosBalanceChange float64        `json:"posBalChg,string"`
	BillSubType      string         `json:"subType"`
	Size             float64        `json:"sz,string"`
	To               string         `json:"to"`
	BillType         string         `json:"type"`
	Timestamp        okcoinMilliSec `json:"ts"`
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
	Currency     string  `json:"ccy"`
	InstrumentID string  `json:"instId"`
	MaxBuy       float64 `json:"maxBuy,string"`
	MaxSell      float64 `json:"maxSell,string"`
}

// AvailableTradableAmount represents maximum available tradable amount information
type AvailableTradableAmount struct {
	AvailableBuy  float64 `json:"availBuy,string"`
	AvailableSell float64 `json:"availSell,string"`
	InstrumentID  string  `json:"instId"`
}

// FeeRate represents instrument trading fee information.
type FeeRate struct {
	Category       string         `json:"category"`
	Delivery       string         `json:"delivery"`
	Exercise       string         `json:"exercise"`
	InstrumentType string         `json:"instType"`
	Level          string         `json:"level"`
	MakerFeeRate   float64        `json:"maker,string"`
	MakerU         string         `json:"makerU"`
	MakerUSDC      string         `json:"makerUSDC"`
	TakerFeeRate   float64        `json:"taker,string"`
	TakerU         string         `json:"takerU"`
	TakerUSDC      string         `json:"takerUSDC"`
	Timestamp      okcoinMilliSec `json:"ts"`
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
		BaseSingleMin     okcoinNumber `json:"baseSingleMin"`
		InstrumentID      string       `json:"instId"`
		QuoteCurrency     string       `json:"quoteCcy"`
		QuoteCurrencyIcon string       `json:"quoteCcyIcon"`
		QuoteSingleMin    okcoinNumber `json:"quoteSingleMin"`
	} `json:"instruments"`
	Timestamp okcoinMilliSec `json:"ts"`
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
	QuoteTimesamp okcoinMilliSec `json:"quoteTs"`
	TTLMs         string         `json:"ttlMs"`
	ClQReqID      string         `json:"clQReqId"`
	QuoteID       string         `json:"quoteId"`
	BaseCurrency  string         `json:"baseCcy"`
	QuoteCurrency string         `json:"quoteCcy"`
	Side          string         `json:"side"`
	OrigRfqSize   float64        `json:"origRfqSz"`
	RfqSize       float64        `json:"rfqSz"`
	RfqSzCurrency string         `json:"rfqSzCcy"`
	BidPrice      float64        `json:"bidPx,string"`
	BidBaseSize   float64        `json:"bidBaseSz,string"`
	BidQuoteSize  float64        `json:"bidQuoteSz,string"`
	AskPx         float64        `json:"askPx,string"`
	AskBaseSize   float64        `json:"askBaseSz,string"`
	AskQuoteSize  float64        `json:"askQuoteSz,string"`
}

// PlaceRFQOrderRequest represents a place RFQ request order.
type PlaceRFQOrderRequest struct {
	ClientDefinedTradeRequestID string        `json:"clTReqId"`
	QuoteID                     string        `json:"quoteId"`
	BaseCurrency                currency.Code `json:"baseCcy"`
	QuoteCurrency               currency.Code `json:"quoteCcy"`
	Side                        string        `json:"side"`
	Size                        float64       `json:"Sz,string"`
	SizeCurrency                currency.Code `json:"szCcy"`
}

// RFQOrderResponse represents an RFQ
type RFQOrderResponse struct {
	Timestamp      okcoinMilliSec `json:"ts"`
	TradeID        string         `json:"tradeId"`
	QuoteID        string         `json:"quoteId"`
	ClTReqID       string         `json:"clTReqId"` // user-defined ID
	State          string         `json:"state"`
	InstrumentID   string         `json:"instId"`
	BaseCurrency   string         `json:"baseCcy"`
	QuoteCurrency  string         `json:"quoteCcy"`
	Side           string         `json:"side"`
	Price          float64        `json:"px,string"`
	FilledBaseSize float64        `json:"filledBaseSz,string"`
	FilledTermSize float64        `json:"filledTermSz,string"`
}

// RFQOrderDetail represents an rfq order detail
type RFQOrderDetail struct {
	Timestamp      okcoinMilliSec `json:"ts"`
	TradeID        string         `json:"tradeId"`
	QuoteID        string         `json:"quoteId"`
	ClTReqID       string         `json:"clTReqId"`
	State          string         `json:"state"`
	InstID         string         `json:"instId"`
	BaseCurrency   string         `json:"baseCcy"`
	QuoteCurrency  string         `json:"quoteCcy"`
	Side           string         `json:"side"`
	Price          float64        `json:"px,string"`
	FilledBaseSize float64        `json:"filledBaseSz,string"`
	FilledTermSize float64        `json:"filledTermSz,string"`
}

// RFQOrderHistoryItem represents otc rfq order instance.
type RFQOrderHistoryItem struct {
	Timestamp        okcoinMilliSec `json:"ts"`
	PageIdx          int64          `json:"pageIdx,string"`
	TotalPageCount   int64          `json:"totalPageCnt,string"`
	TotalRecordCount int64          `json:"totalRecordCnt,string"`
	Trades           []struct {
		Timestamp      okcoinMilliSec `json:"ts"`
		TradeID        string         `json:"tradeId"`
		TradeTimestamp okcoinMilliSec `json:"tradeTs"`
		ClTRequestID   string         `json:"clTReqId"`
		InstrumentID   string         `json:"instId"`
		Side           string         `json:"side"`
		Price          float64        `json:"px,string"`
		BaseCurrency   string         `json:"baseCcy"`
		BaseSize       float64        `json:"baseSz,string"`
		QuoteCurrency  string         `json:"quoteCcy"`
		QuoteSize      float64        `json:"quoteSz,string"`
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
	DepositID    string         `json:"depId"`
	CreationTime okcoinMilliSec `json:"cTime"`
}

// CancelDepositAddressResp represents a deposit address id response after cancelling.
type CancelDepositAddressResp struct {
	DepositAddressID string `json:"depId"`
}

// DepositHistoryResponse represents a deposit history instance detail.
type DepositHistoryResponse struct {
	DepositID         string         `json:"depId"`
	ChannelID         string         `json:"chanId"`
	BillID            string         `json:"billId"`
	BankAccountName   string         `json:"bankAcctName"`
	BankAccountNumber string         `json:"bankAcctNum"`
	Amount            float64        `json:"amt,string"`
	State             string         `json:"state"`
	Currency          string         `json:"ccy"`
	CreationTime      okcoinMilliSec `json:"cTime"`
	UpdatedTime       okcoinMilliSec `json:"uTime"`
}

// FiatWithdrawalParam represents a fiat withdrawal parameters
type FiatWithdrawalParam struct {
	ChannelID      string  `json:"chanId"`
	BankAcctNumber string  `json:"bankAcctNum"`
	Amount         float64 `json:"amt,string"`
}

// FiatWithdrawalResponse represents a fiat withdrawal
type FiatWithdrawalResponse struct {
	DepositID    string         `json:"depId"`
	Fee          float64        `json:"fee,string"`
	CreationTime okcoinMilliSec `json:"cTime"`
}

// FiatWithdrawalHistoryItem represents fiat withdrawal history item.
type FiatWithdrawalHistoryItem struct {
	WithdrawalID    string         `json:"wdId"`
	ChannelID       string         `json:"chanId"`
	BillID          string         `json:"billId"`
	BankAccountName string         `json:"bankAcctName"`
	BankAcctNumber  string         `json:"bankAcctNum"`
	Amount          float64        `json:"amt,string"`
	Fee             float64        `json:"fee,string"`
	State           string         `json:"state"`
	Ccy             string         `json:"ccy"`
	CreationTime    okcoinMilliSec `json:"cTime"`
	UpdateTime      okcoinMilliSec `json:"uTime"`
}

// ChannelInfo represents a channel information
type ChannelInfo struct {
	ChannelID            string       `json:"chanId"`
	Currency             string       `json:"ccy"`
	DepositQuota         string       `json:"depQuota"`
	MinDeposit           okcoinNumber `json:"minDep"`
	WithdrawalQuota      string       `json:"wdQuota"`
	MinWithdrawal        string       `json:"minWd"`
	UsedWithdrawalQuota  string       `json:"usedWdQuota"`
	ValidWithdrawalQuota string       `json:"validWdQuota"`
	BankAccountInfo      []struct {
		BankAccountName   string `json:"bankAcctName"`
		BankAccountNumber string `json:"bankAcctNum"`
		InstrumentName    string `json:"instName"`
		MaskAccountNumber string `json:"maskAcctNum"`
	} `json:"bankAcctInfo"`
}

// PlaceTradeOrderParam represents a trade order arguments.
type PlaceTradeOrderParam struct {
	InstrumentID   currency.Pair `json:"instId"`
	TradeMode      string        `json:"tdMode"` // Trade mode --> Margin mode: 'cross','isolated' Non-Margin mode: 'cash'
	ClientOrderID  string        `json:"clOrdId"`
	Side           string        `json:"side"`                // Order side, buy sell
	OrderType      string        `json:"ordType"`             // Order type 'market': Market order 'limit': Limit order 'post_only': Post-only order 'fok': Fill-or-kill order 'ioc': Immediate-or-cancel order
	Price          float64       `json:"px,string,omitempty"` // Order price. Only applicable to limit,post_only,fok,ioc order.
	Size           float64       `json:"sz,string"`
	OrderTag       string        `json:"tag,omitempty"` // Order tag A combination of case-sensitive alphanumerics, all numbers, or all letters of up to 16 characters.
	BanAmend       bool          `json:"banAmend,omitempty"`
	TargetCurrency string        `json:"tgtCcy,omitempty"` // Whether the target currency uses the quote or base currency.

	// ExpiryTime is the request effective deadline.
	ExpiryTime okcoinMilliSec `json:"expTime"`
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
	OrderID       string `json:"ordId"`
	ClientOrderID string `json:"clOrdId"`
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
	AccFillSize                float64        `json:"accFillSz,string"`
	AveragePrice               float64        `json:"avgPx,string"`
	CreationTime               okcoinMilliSec `json:"cTime"`
	Category                   string         `json:"category"`
	Currency                   string         `json:"ccy"`
	ClientOrdID                string         `json:"clOrdId"`
	Fee                        okcoinNumber   `json:"fee"`
	FeeCurrency                string         `json:"feeCcy"`
	FillPrice                  float64        `json:"fillPx,string"`
	FillSize                   float64        `json:"fillSz,string"`
	FillTime                   okcoinMilliSec `json:"fillTime"`
	InstrumentID               string         `json:"instId"`
	InstrumentType             string         `json:"instType"`
	Leverage                   okcoinNumber   `json:"lever"`
	OrderID                    string         `json:"ordId"`
	OrderType                  string         `json:"ordType"`
	ProfitAndLoss              string         `json:"pnl"`
	PosSide                    string         `json:"posSide"`
	Price                      float64        `json:"px,string"`
	Rebate                     string         `json:"rebate"`
	RebateCurrency             string         `json:"rebateCcy"`
	ReduceOnly                 bool           `json:"reduceOnly,string"`
	Side                       string         `json:"side"`
	StopLossOrdPrice           okcoinNumber   `json:"slOrdPx"`
	StopLossTriggerPrice       okcoinNumber   `json:"slTriggerPx"`
	StopLossTriggerPriceType   string         `json:"slTriggerPxType"`
	Source                     string         `json:"source"`
	State                      string         `json:"state"`
	Size                       float64        `json:"sz,string"`
	Tag                        string         `json:"tag"`
	TradeMode                  string         `json:"tdMode"`
	TargetCurrency             string         `json:"tgtCcy"`
	TakeProfitOrderPrice       okcoinNumber   `json:"tpOrdPx"`
	TakeProfitTriggerPrice     okcoinNumber   `json:"tpTriggerPx"`
	TakeProfitTriggerPriceType string         `json:"tpTriggerPxType"`
	TradeID                    string         `json:"tradeId"`
	UpdateTime                 okcoinMilliSec `json:"uTime"`
}

// TransactionFillItem represents recently filled transactions
type TransactionFillItem struct {
	InstrumentType string         `json:"instType"`
	InstrumentID   string         `json:"instId"`
	TradeID        string         `json:"tradeId"`
	OrderID        string         `json:"ordId"`
	ClientOrderID  string         `json:"clOrdId"`
	BillID         string         `json:"billId"`
	Tag            string         `json:"tag"`
	FillSize       float64        `json:"fillSz,string"`
	FillPrice      float64        `json:"fillPx,string"`
	Side           string         `json:"side"`
	PosSide        string         `json:"posSide"`
	ExecType       string         `json:"execType"`
	FeeCurrency    string         `json:"feeCcy"`
	Fee            okcoinNumber   `json:"fee"`
	Timestamp      okcoinMilliSec `json:"ts"`
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
	ActivePrice              okcoinNumber   `json:"activePx"`
	ActualPrice              okcoinNumber   `json:"actualPx"`
	ActualSide               string         `json:"actualSide"`
	ActualSize               okcoinNumber   `json:"actualSz"`
	AlgoID                   string         `json:"algoId"`
	CreateTime               okcoinMilliSec `json:"cTime"`
	CallbackRatio            okcoinNumber   `json:"callbackRatio"`
	CallbackSpread           string         `json:"callbackSpread"`
	Currency                 string         `json:"ccy"`
	ClientOrderID            string         `json:"clOrdId"`
	InstrumentID             string         `json:"instId"`
	InstrumentType           string         `json:"instType"`
	Leverage                 string         `json:"lever"`
	MoveTriggerPrice         okcoinNumber   `json:"moveTriggerPx"`
	OrderID                  string         `json:"ordId"`
	OrdPrice                 okcoinNumber   `json:"ordPx"`
	OrderType                string         `json:"ordType"`
	PosSide                  string         `json:"posSide"`
	PriceLimit               okcoinNumber   `json:"pxLimit"`
	PriceSpread              string         `json:"pxSpread"`
	PriceVar                 string         `json:"pxVar"`
	Side                     string         `json:"side"`
	StopLossOrdPrice         okcoinNumber   `json:"slOrdPx"`
	StopLossTriggerPrice     okcoinNumber   `json:"slTriggerPx"`
	StopLossTriggerPriceType string         `json:"slTriggerPxType"`
	State                    string         `json:"state"`
	Size                     okcoinNumber   `json:"sz"`
	SizeLimit                okcoinNumber   `json:"szLimit"`
	Tag                      string         `json:"tag"`
	TdMode                   string         `json:"tdMode"`
	TgtCcy                   string         `json:"tgtCcy"`
	TimeInterval             string         `json:"timeInterval"`
	TpOrdPrice               okcoinNumber   `json:"tpOrdPx"`
	TpTriggerPrice           okcoinNumber   `json:"tpTriggerPx"`
	TpTriggerPriceType       string         `json:"tpTriggerPxType"`
	TriggerPrice             okcoinNumber   `json:"triggerPx"`
	TriggerPriceType         string         `json:"triggerPxType"`
	TriggerTime              okcoinMilliSec `json:"triggerTime"`
}
