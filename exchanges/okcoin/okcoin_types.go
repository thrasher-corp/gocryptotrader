package okcoin

import (
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
)

var errorCodes = map[string]string{
	"0":     "Ok",
	"1":     "Operation failed.",
	"2":     "Bulk operation partially succeeded.",
	"50000": "Body cannot be empty.",
	"50001": "Service temporarily unavailable, please try again later.",
	"50002": "Json data format error.",
	"50004": "Endpoint request timeout (does not mean that the request was successful or failed, please check the request result).",
	"50005": "API is offline or unavailable.",
	"50006": "Invalid Content_Type, please use 'application/json' format.",
	"50007": "Account blocked.",
	"50008": "User does not exist.",
	"50009": "Account is suspended due to ongoing liquidation.",
	"50010": "User ID cannot be empty.",
	"50011": "Requests too frequent.",
	"50012": "Account status invalid.",
	"50013": "System is busy, please try again later.",
	"50026": "System error, please try again later.",
	"50027": "The account is restricted from trading.",
	"50028": "Unable to take the order, please reach out to support center for details.",
	"50030": "No permission to use this API",
	"50032": "This asset is blocked, allow its trading and try again",
	"50033": "This instrument is blocked, allow its trading and try again",
	"50035": "This endpoint requires that APIKey must be bound to IP",
	"50036": "Invalid expTime",
	"50037": "Order expired",
	"50038": "This feature is temporarily unavailable in demo trading",
	"50039": "The before parameter is not available for implementing timestamp pagination",
	"50041": "You are not currently on the whitelist, please contact customer service",
	"50100": `API frozen, please contact customer service.`,
	"50101": `APIKey does not match current environment.`,
	"50102": `Timestamp request expired.`,
	"50103": `Request header "OK-ACCESS-KEY" cannot be empty.`,
	"50104": `Request header "OK-ACCESS-PASSPHRASE" cannot be empty.`,
	"50105": `Request header "OK-ACCESS-PASSPHRASE" incorrect.`,
	"50106": `Request header "OK-ACCESS-SIGN" cannot be empty.`,
	"50107": `Request header "OK-ACCESS-TIMESTAMP" cannot be empty.`,
	"50108": `Exchange ID does not exist.`,
	"50109": `Exchange domain does not exist.`,
	"50111": `Invalid OK-ACCESS-KEY.`,
	"50112": `Invalid OK-ACCESS-TIMESTAMP.`,
	"50113": `Invalid signature.`,
	"50114": `Invalid authorization.`,
	"50115": `Invalid request method.`,
	"51001": `Instrument ID does not exist.`,
	"51003": `Either client order ID or order ID is required.`,
	"51005": `Order amount exceeds the limit.`,
	"51009": `Order placement function is blocked by the platform.`,
	"51010": `Operation is not supported under the current account mode.`,
	"51011": `Duplicated order ID.`,
	"51012": `Token does not exist.`,
	"51014": `Index does not exist.`,
	"51015": `Instrument ID does not match instrument type.`,
	"51016": `Duplicated client order ID.`,
	"51020": `Order amount should be greater than the min available amount.`,
	"51023": `Position does not exist.`,
	"51024": `Trading account is blocked.`,
	"51025": `Order count exceeds the limit.`,
	"51026": `Instrument type does not match underlying index.`,
	"51030": `Funding fee is being settled.`,
	"51031": `This order price is not within the closing price range.`,
	"51032": `Closing all positions at market price.`,
	"51033": `The total amount per order for this pair has reached the upper limit.`,
	"51037": `The current account risk status only supports you to place IOC orders that can reduce the risk of your account.`,
	"51038": `There is already an IOC order under the current risk module that reduces the risk of the account.`,
	"51046": `The take profit trigger price should be higher than the order price`,
	"51047": `The stop loss trigger price should be lower than the order price`,
	"51048": `The take profit trigger price should be lower than the order price`,
	"51049": `The stop loss trigger price should be higher than the order price`,
	"51050": `The take profit trigger price should be higher than the best ask price`,
	"51051": `The stop loss trigger price should be lower than the best ask price`,
	"51052": `The take profit trigger price should be lower than the best bid price`,
	"51053": `The stop loss trigger price should be higher than the best bid price`,
	"51054": `Getting information timed out, please try again later`,
	"51056": `Action not allowed`,
	"51058": `No available position for this algo order`,
	"51059": `Strategy for the current state does not support this operation`,
	"51101": `Entered amount exceeds the max pending order amount (Cont) per transaction.`,
	"51103": `Entered amount exceeds the max pending order count of the underlying asset.`,
	"51104": `Entered amount exceeds the max pending order amount (Cont) of the underlying asset.`,
	"51106": `Entered amount exceeds the max order amount (Cont) of the underlying asset.`,
	"51107": `Entered amount exceeds the max holding amount (Cont).`,
	"51109": `No available offer.`,
	"51110": `You can only place a limit order after Call Auction has started.`,
	"51112": `Close order size exceeds your available size.`,
	"51113": `Market-price liquidation requests too frequent.`,
	"51115": `Cancel all pending close-orders before liquidation.`,
	"51117": `Pending close-orders count exceeds limit.`,
	"51121": `Order count should be the integer multiples of the lot size.`,
	"51124": `You can only place limit orders during call auction.`,
	"51127": `Available balance is 0.`,
	"51129": `The value of the position and buy order has reached the position limit, and no further buying is allowed.`,
	"51131": `Insufficient balance.`,
	"51132": `Your position amount is negative and less than the minimum trading amount.`,
	"51134": `Closing position failed. Please check your holdings and pending orders.`,
	"51139": `Reduce-only feature is unavailable for the spot transactions by simple account.`,
	"51143": `There is no valid quotation in the market, and the order cannot be filled in USDT mode, please try to switch to currency mode`,
	"51148": `ReduceOnly cannot increase the position quantity.`,
	"51149": `Order timed out, please try again later.`,
	"51150": `The precision of the number of trades or the price exceeds the limit.`,
	"51201": `Value of per market order cannot exceed 1,000,000 USDT.`,
	"51202": `Market - order amount exceeds the max amount.`,
	"51204": `The price for the limit order cannot be empty.`,
	"51205": `Reduce-Only is not available.`,
	"51250": `Algo order price is out of the available range.`,
	"51251": `Algo order type error (when user place an iceberg order).`,
	"51252": `Algo order amount is out of the available range.`,
	"51253": `Average amount exceeds the limit of per iceberg order.`,
	"51254": `Iceberg average amount error (when user place an iceberg order).`,
	"51255": `Limit of per iceberg order: Total amount/1000 < x <= Total amount.`,
	"51256": `Iceberg order price variance error.`,
	"51257": `Trail order callback rate error.`,
	"51258": `Trail - order placement failed. The trigger price of a sell order should be higher than the last transaction price.`,
	"51259": `Trail - order placement failed. The trigger price of a buy order should be lower than the last transaction price.`,
	"51264": `Average amount exceeds the limit of per time-weighted order.`,
	"51265": `Time-weighted order limit error.`,
	"51267": `Time-weighted order strategy initiative rate error.`,
	"51268": `Time-weighted order strategy initiative range error.`,
	"51270": `The limit of time-weighted order price variance is 0 < x <= 1%.`,
	"51271": `Sweep ratio should be 0 < x <= 100%.`,
	"51272": `Price variance should be 0 < x <= 1%.`,
	"51274": `Total quantity of time-weighted order must be larger than single order limit.`,
	"51275": `The amount of single stop-market order cannot exceed the upper limit.`,
	"51276": `Stop - Market orders cannot specify a price.`,
	"51277": `TP trigger price cannot be higher than the last price.`,
	"51278": `SL trigger price cannot be lower than the last price.`,
	"51279": `TP trigger price cannot be lower than the last price.`,
	"51280": `SL trigger price cannot be higher than the last price.`,
	"51281": `trigger not support the tgtCcy parameter.`,
	"51288": `We are stopping the Bot. Please do not click it multiple times`,
	"51289": `Bot configuration does not exist. Please try again later`,
	"51290": `The Bot engine is being upgraded. Please try again later`,
	"51291": `This Bot does not exist or has been stopped`,
	"51292": `This Bot type does not exist`,
	"51293": `This Bot does not exist`,
	"51294": `This Bot cannot be created temporarily. Please try again later`,
	"51300": `TP trigger price cannot be higher than the mark price`,
	"51302": `SL trigger price cannot be lower than the mark price`,
	"51303": `TP trigger price cannot be lower than the mark price`,
	"51304": `SL trigger price cannot be higher than the mark price`,
	"51305": `TP trigger price cannot be higher than the index price`,
	"51306": `SL trigger price cannot be lower than the index price`,
	"51307": `TP trigger price cannot be lower than the index price`,
	"51308": `SL trigger price cannot be higher than the index price`,
	"51309": `Cannot create trading bot during call auction`,
	"51313": `Manual transfer in isolated mode does not support bot trading`,
	"51341": `Position closing not allowed`,
	"51342": `Closing order already exists. Please try again later`,
	"51343": `TP price must be less than the lower price`,
	"51344": `SL price must be greater than the upper price`,
	"51345": `Policy type is not grid policy`,
	"51346": `The highest price cannot be lower than the lowest price`,
	"51347": `No profit available`,
	"51348": `Stop loss price should be less than the lower price in the range`,
	"51349": `Stop profit price should be greater than the highest price in the range`,
	"51350": `No recommended parameters`,
	"51351": `Single income must be greater than 0`,
	"51400": `cancellation failed as the order does not exist.`,
	"51401": `cancellation failed as the order is already canceled.`,
	"51402": `cancellation failed as the order is already completed.`,
	"51403": `cancellation failed as the order type does not support cancellation.`,
	"51404": `Order cancellation unavailable during the second phase of call auction.`,
	"51405": `cancellation failed as you do not have any pending orders.`,
	"51407": `Either order ID or client order ID is required.`,
	"51408": `Pair ID or name does not match the order info.`,
	"51409": `Either pair ID or pair name ID is required.`,
	"51410": `cancellation pending. Duplicate order rejected.`,
	"51411": `Account does not have permission for mass cancellation.`,
	"51412": `The order has been triggered and cannot be canceled.`,
	"51413": `cancellation failed as the order type is not supported by endpoint.`,
	"51415": `Unable to place order. Spot trading only supports using the last price as trigger price. Please select "Last" and try again.`,
	"51500": `Either order price or amount is required.`,
	"51503": `Order modification failed as the order does not exist.`,
	"51506": `Order modification unavailable for the order type.`,
	"51508": `Orders are not allowed to be modified during the call auction.`,
	"51509": `Modification failed as the order has been canceled.`,
	"51510": `Modification failed as the order has been completed.`,
	"51511": `Operation failed as the order price did not meet the requirement for Post Only.`,
	"51512": `Failed to amend orders in batches. You cannot have duplicate orders in the same amend-batch-orders request.`,
	"51513": `Number of modification requests that are currently in progress for an order cannot exceed 3.`,
	"51600": `Status not found.`,
	"51601": `Order status and order ID cannot exist at the same time.`,
	"51602": `Either order status or order ID is required.`,
	"51603": `Order does not exist.`,
	"51607": `The file is generating.`,
}

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
	Asks      [][]string     `json:"asks"` // [ Price, Quantity, depreciated, number of orders at the price ]
	Bids      [][]string     `json:"bids"` // [ Price, Quantity, depreciated, number of orders at the price ]
	Timestamp okcoinMilliSec `json:"ts"`
	Checksum  int32          `json:"checksum"`
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
		State                 okcoinMilliSec `json:"state"`
		Begin                 okcoinMilliSec `json:"begin"`
		Href                  string         `json:"href"`
		End                   string         `json:"end"`
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
			AvailableBalance   string         `json:"availBal"`
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
	// Orderbook events
	okcoinWsOrderbookUpdate  = "update"
	okcoinWsOrderbookPartial = "partial"
	// API subsections
	okcoinWsSwapSubsection    = "swap/"
	okcoinWsIndexSubsection   = "index/"
	okcoinWsFuturesSubsection = "futures/"
	okcoinWsSpotSubsection    = "spot/"
	// Shared API endpoints
	okcoinWsCandle         = "candle"
	okcoinWsCandle60s      = okcoinWsCandle + "60s"
	okcoinWsCandle180s     = okcoinWsCandle + "180s"
	okcoinWsCandle300s     = okcoinWsCandle + "300s"
	okcoinWsCandle900s     = okcoinWsCandle + "900s"
	okcoinWsCandle1800s    = okcoinWsCandle + "1800s"
	okcoinWsCandle3600s    = okcoinWsCandle + "3600s"
	okcoinWsCandle7200s    = okcoinWsCandle + "7200s"
	okcoinWsCandle14400s   = okcoinWsCandle + "14400s"
	okcoinWsCandle21600s   = okcoinWsCandle + "21600"
	okcoinWsCandle43200s   = okcoinWsCandle + "43200s"
	okcoinWsCandle86400s   = okcoinWsCandle + "86400s"
	okcoinWsCandle604900s  = okcoinWsCandle + "604800s"
	okcoinWsTicker         = "ticker"
	okcoinWsTrade          = "trade"
	okcoinWsDepth          = "depth"
	okcoinWsDepth5         = "depth5"
	okcoinWsAccount        = "account"
	okcoinWsMarginAccount  = "margin_account"
	okcoinWsOrder          = "order"
	okcoinWsFundingRate    = "funding_rate"
	okcoinWsPriceRange     = "price_range"
	okcoinWsMarkPrice      = "mark_price"
	okcoinWsPosition       = "position"
	okcoinWsEstimatedPrice = "estimated_price"
	// Spot endpoints
	okcoinWsSpotTicker        = okcoinWsSpotSubsection + okcoinWsTicker
	okcoinWsSpotCandle60s     = okcoinWsSpotSubsection + okcoinWsCandle60s
	okcoinWsSpotCandle180s    = okcoinWsSpotSubsection + okcoinWsCandle180s
	okcoinWsSpotCandle300s    = okcoinWsSpotSubsection + okcoinWsCandle300s
	okcoinWsSpotCandle900s    = okcoinWsSpotSubsection + okcoinWsCandle900s
	okcoinWsSpotCandle1800s   = okcoinWsSpotSubsection + okcoinWsCandle1800s
	okcoinWsSpotCandle3600s   = okcoinWsSpotSubsection + okcoinWsCandle3600s
	okcoinWsSpotCandle7200s   = okcoinWsSpotSubsection + okcoinWsCandle7200s
	okcoinWsSpotCandle14400s  = okcoinWsSpotSubsection + okcoinWsCandle14400s
	okcoinWsSpotCandle21600s  = okcoinWsSpotSubsection + okcoinWsCandle21600s
	okcoinWsSpotCandle43200s  = okcoinWsSpotSubsection + okcoinWsCandle43200s
	okcoinWsSpotCandle86400s  = okcoinWsSpotSubsection + okcoinWsCandle86400s
	okcoinWsSpotCandle604900s = okcoinWsSpotSubsection + okcoinWsCandle604900s
	okcoinWsSpotTrade         = okcoinWsSpotSubsection + okcoinWsTrade
	okcoinWsSpotDepth         = okcoinWsSpotSubsection + okcoinWsDepth
	okcoinWsSpotDepth5        = okcoinWsSpotSubsection + okcoinWsDepth5
	okcoinWsSpotAccount       = okcoinWsSpotSubsection + okcoinWsAccount
	okcoinWsSpotMarginAccount = okcoinWsSpotSubsection + okcoinWsMarginAccount
	okcoinWsSpotOrder         = okcoinWsSpotSubsection + okcoinWsOrder
	// Swap endpoints
	okcoinWsSwapTicker        = okcoinWsSwapSubsection + okcoinWsTicker
	okcoinWsSwapCandle60s     = okcoinWsSwapSubsection + okcoinWsCandle60s
	okcoinWsSwapCandle180s    = okcoinWsSwapSubsection + okcoinWsCandle180s
	okcoinWsSwapCandle300s    = okcoinWsSwapSubsection + okcoinWsCandle300s
	okcoinWsSwapCandle900s    = okcoinWsSwapSubsection + okcoinWsCandle900s
	okcoinWsSwapCandle1800s   = okcoinWsSwapSubsection + okcoinWsCandle1800s
	okcoinWsSwapCandle3600s   = okcoinWsSwapSubsection + okcoinWsCandle3600s
	okcoinWsSwapCandle7200s   = okcoinWsSwapSubsection + okcoinWsCandle7200s
	okcoinWsSwapCandle14400s  = okcoinWsSwapSubsection + okcoinWsCandle14400s
	okcoinWsSwapCandle21600s  = okcoinWsSwapSubsection + okcoinWsCandle21600s
	okcoinWsSwapCandle43200s  = okcoinWsSwapSubsection + okcoinWsCandle43200s
	okcoinWsSwapCandle86400s  = okcoinWsSwapSubsection + okcoinWsCandle86400s
	okcoinWsSwapCandle604900s = okcoinWsSwapSubsection + okcoinWsCandle604900s
	okcoinWsSwapTrade         = okcoinWsSwapSubsection + okcoinWsTrade
	okcoinWsSwapDepth         = okcoinWsSwapSubsection + okcoinWsDepth
	okcoinWsSwapDepth5        = okcoinWsSwapSubsection + okcoinWsDepth5
	okcoinWsSwapFundingRate   = okcoinWsSwapSubsection + okcoinWsFundingRate
	okcoinWsSwapPriceRange    = okcoinWsSwapSubsection + okcoinWsPriceRange
	okcoinWsSwapMarkPrice     = okcoinWsSwapSubsection + okcoinWsMarkPrice
	okcoinWsSwapPosition      = okcoinWsSwapSubsection + okcoinWsPosition
	okcoinWsSwapAccount       = okcoinWsSwapSubsection + okcoinWsAccount
	okcoinWsSwapOrder         = okcoinWsSwapSubsection + okcoinWsOrder
	// Index endpoints
	okcoinWsIndexTicker        = okcoinWsIndexSubsection + okcoinWsTicker
	okcoinWsIndexCandle60s     = okcoinWsIndexSubsection + okcoinWsCandle60s
	okcoinWsIndexCandle180s    = okcoinWsIndexSubsection + okcoinWsCandle180s
	okcoinWsIndexCandle300s    = okcoinWsIndexSubsection + okcoinWsCandle300s
	okcoinWsIndexCandle900s    = okcoinWsIndexSubsection + okcoinWsCandle900s
	okcoinWsIndexCandle1800s   = okcoinWsIndexSubsection + okcoinWsCandle1800s
	okcoinWsIndexCandle3600s   = okcoinWsIndexSubsection + okcoinWsCandle3600s
	okcoinWsIndexCandle7200s   = okcoinWsIndexSubsection + okcoinWsCandle7200s
	okcoinWsIndexCandle14400s  = okcoinWsIndexSubsection + okcoinWsCandle14400s
	okcoinWsIndexCandle21600s  = okcoinWsIndexSubsection + okcoinWsCandle21600s
	okcoinWsIndexCandle43200s  = okcoinWsIndexSubsection + okcoinWsCandle43200s
	okcoinWsIndexCandle86400s  = okcoinWsIndexSubsection + okcoinWsCandle86400s
	okcoinWsIndexCandle604900s = okcoinWsIndexSubsection + okcoinWsCandle604900s
	// Futures endpoints
	okcoinWsFuturesTicker         = okcoinWsFuturesSubsection + okcoinWsTicker
	okcoinWsFuturesCandle60s      = okcoinWsFuturesSubsection + okcoinWsCandle60s
	okcoinWsFuturesCandle180s     = okcoinWsFuturesSubsection + okcoinWsCandle180s
	okcoinWsFuturesCandle300s     = okcoinWsFuturesSubsection + okcoinWsCandle300s
	okcoinWsFuturesCandle900s     = okcoinWsFuturesSubsection + okcoinWsCandle900s
	okcoinWsFuturesCandle1800s    = okcoinWsFuturesSubsection + okcoinWsCandle1800s
	okcoinWsFuturesCandle3600s    = okcoinWsFuturesSubsection + okcoinWsCandle3600s
	okcoinWsFuturesCandle7200s    = okcoinWsFuturesSubsection + okcoinWsCandle7200s
	okcoinWsFuturesCandle14400s   = okcoinWsFuturesSubsection + okcoinWsCandle14400s
	okcoinWsFuturesCandle21600s   = okcoinWsFuturesSubsection + okcoinWsCandle21600s
	okcoinWsFuturesCandle43200s   = okcoinWsFuturesSubsection + okcoinWsCandle43200s
	okcoinWsFuturesCandle86400s   = okcoinWsFuturesSubsection + okcoinWsCandle86400s
	okcoinWsFuturesCandle604900s  = okcoinWsFuturesSubsection + okcoinWsCandle604900s
	okcoinWsFuturesTrade          = okcoinWsFuturesSubsection + okcoinWsTrade
	okcoinWsFuturesEstimatedPrice = okcoinWsFuturesSubsection + okcoinWsTrade
	okcoinWsFuturesPriceRange     = okcoinWsFuturesSubsection + okcoinWsPriceRange
	okcoinWsFuturesDepth          = okcoinWsFuturesSubsection + okcoinWsDepth
	okcoinWsFuturesDepth5         = okcoinWsFuturesSubsection + okcoinWsDepth5
	okcoinWsFuturesMarkPrice      = okcoinWsFuturesSubsection + okcoinWsMarkPrice
	okcoinWsFuturesAccount        = okcoinWsFuturesSubsection + okcoinWsAccount
	okcoinWsFuturesPosition       = okcoinWsFuturesSubsection + okcoinWsPosition
	okcoinWsFuturesOrder          = okcoinWsFuturesSubsection + okcoinWsOrder

	okcoinWsRateLimit = 30

	allowableIterations = 25
	delimiterColon      = ":"
	delimiterDash       = "-"

	maxConnByteLen = 4096
)

// orderbookMutex Ensures if two entries arrive at once, only one can be
// processed at a time
var orderbookMutex sync.Mutex

var defaultSpotSubscribedChannels = []string{okcoinWsSpotDepth,
	okcoinWsSpotCandle300s,
	okcoinWsSpotTicker,
	okcoinWsSpotTrade}

var defaultFuturesSubscribedChannels = []string{okcoinWsFuturesDepth,
	okcoinWsFuturesCandle300s,
	okcoinWsFuturesTicker,
	okcoinWsFuturesTrade}

var defaultIndexSubscribedChannels = []string{okcoinWsIndexCandle300s,
	okcoinWsIndexTicker}

var defaultSwapSubscribedChannels = []string{okcoinWsSwapDepth,
	okcoinWsSwapCandle300s,
	okcoinWsSwapTicker,
	okcoinWsSwapTrade,
	okcoinWsSwapFundingRate,
	okcoinWsSwapMarkPrice}

// SetErrorDefaults sets the full error default list
func (o *OKCoin) SetErrorDefaults() {
	o.ErrorCodes = map[string]error{
		"0":     errors.New("successful"),
		"1":     errors.New("invalid parameter in url normally"),
		"30001": errors.New("request header \"OK_ACCESS_KEY\" cannot be blank"),
		"30002": errors.New("request header \"OK_ACCESS_SIGN\" cannot be blank"),
		"30003": errors.New("request header \"OK_ACCESS_TIMESTAMP\" cannot be blank"),
		"30004": errors.New("request header \"OK_ACCESS_PASSPHRASE\" cannot be blank"),
		"30005": errors.New("invalid OK_ACCESS_TIMESTAMP"),
		"30006": errors.New("invalid OK_ACCESS_KEY"),
		"30007": errors.New("invalid Content_Type, please use \"application/json\" format"),
		"30008": errors.New("timestamp request expired"),
		"30009": errors.New("system error"),
		"30010": errors.New("api validation failed"),
		"30011": errors.New("invalid IP"),
		"30012": errors.New("invalid authorization"),
		"30013": errors.New("invalid sign"),
		"30014": errors.New("request too frequent"),
		"30015": errors.New("request header \"OK_ACCESS_PASSPHRASE\" incorrect"),
		"30016": errors.New("you are using v1 apiKey, please use v1 endpoint. If you would like to use v3 endpoint, please subscribe to v3 apiKey"),
		"30017": errors.New("apikey's broker id does not match"),
		"30018": errors.New("apikey's domain does not match"),
		"30020": errors.New("body cannot be blank"),
		"30021": errors.New("json data format error"),
		"30023": errors.New("required parameter cannot be blank"),
		"30024": errors.New("parameter value error"),
		"30025": errors.New("parameter category error"),
		"30026": errors.New("requested too frequent; endpoint limit exceeded"),
		"30027": errors.New("login failure"),
		"30028": errors.New("unauthorized execution"),
		"30029": errors.New("account suspended"),
		"30030": errors.New("endpoint request failed. Please try again"),
		"30031": errors.New("token does not exist"),
		"30032": errors.New("pair does not exist"),
		"30033": errors.New("exchange domain does not exist"),
		"30034": errors.New("exchange ID does not exist"),
		"30035": errors.New("trading is not supported in this website"),
		"30036": errors.New("no relevant data"),
		"30037": errors.New("endpoint is offline or unavailable"),
		"30038": errors.New("user does not exist"),
		"32001": errors.New("futures account suspended"),
		"32002": errors.New("futures account does not exist"),
		"32003": errors.New("canceling, please wait"),
		"32004": errors.New("you have no unfilled orders"),
		"32005": errors.New("max order quantity"),
		"32006": errors.New("the order price or trigger price exceeds USD 1 million"),
		"32007": errors.New("leverage level must be the same for orders on the same side of the contract"),
		"32008": errors.New("max. positions to open (cross margin)"),
		"32009": errors.New("max. positions to open (fixed margin)"),
		"32010": errors.New("leverage cannot be changed with open positions"),
		"32011": errors.New("futures status error"),
		"32012": errors.New("futures order update error"),
		"32013": errors.New("token type is blank"),
		"32014": errors.New("your number of contracts closing is larger than the number of contracts available"),
		"32015": errors.New("margin ratio is lower than 100% before opening positions"),
		"32016": errors.New("margin ratio is lower than 100% after opening position"),
		"32017": errors.New("no BBO"),
		"32018": errors.New("the order quantity is less than 1, please try again"),
		"32019": errors.New("the order price deviates from the price of the previous minute by more than 3%"),
		"32020": errors.New("the price is not in the range of the price limit"),
		"32021": errors.New("leverage error"),
		"32022": errors.New("this function is not supported in your country or region according to the regulations"),
		"32023": errors.New("this account has outstanding loan"),
		"32024": errors.New("order cannot be placed during delivery"),
		"32025": errors.New("order cannot be placed during settlement"),
		"32026": errors.New("your account is restricted from opening positions"),
		"32027": errors.New("cancelled over 20 orders"),
		"32028": errors.New("account is suspended and liquidated"),
		"32029": errors.New("order info does not exist"),
		"33001": errors.New("margin account for this pair is not enabled yet"),
		"33002": errors.New("margin account for this pair is suspended"),
		"33003": errors.New("no loan balance"),
		"33004": errors.New("loan amount cannot be smaller than the minimum limit"),
		"33005": errors.New("repayment amount must exceed 0"),
		"33006": errors.New("loan order not found"),
		"33007": errors.New("status not found"),
		"33008": errors.New("loan amount cannot exceed the maximum limit"),
		"33009": errors.New("user ID is blank"),
		"33010": errors.New("you cannot cancel an order during session 2 of call auction"),
		"33011": errors.New("no new market data"),
		"33012": errors.New("order cancellation failed"),
		"33013": errors.New("order placement failed"),
		"33014": errors.New("order does not exist"),
		"33015": errors.New("exceeded maximum limit"),
		"33016": errors.New("margin trading is not open for this token"),
		"33017": errors.New("insufficient balance"),
		"33018": errors.New("this parameter must be smaller than 1"),
		"33020": errors.New("request not supported"),
		"33021": errors.New("token and the pair do not match"),
		"33022": errors.New("pair and the order do not match"),
		"33023": errors.New("you can only place market orders during call auction"),
		"33024": errors.New("trading amount too small"),
		"33025": errors.New("base token amount is blank"),
		"33026": errors.New("transaction completed"),
		"33027": errors.New("cancelled order or order cancelling"),
		"33028": errors.New("the decimal places of the trading price exceeded the limit"),
		"33029": errors.New("the decimal places of the trading size exceeded the limit"),
		"34001": errors.New("withdrawal suspended"),
		"34002": errors.New("please add a withdrawal address"),
		"34003": errors.New("sorry, this token cannot be withdrawn to xx at the moment"),
		"34004": errors.New("withdrawal fee is smaller than minimum limit"),
		"34005": errors.New("withdrawal fee exceeds the maximum limit"),
		"34006": errors.New("withdrawal amount is lower than the minimum limit"),
		"34007": errors.New("withdrawal amount exceeds the maximum limit"),
		"34008": errors.New("insufficient balance"),
		"34009": errors.New("your withdrawal amount exceeds the daily limit"),
		"34010": errors.New("transfer amount must be larger than 0"),
		"34011": errors.New("conditions not met"),
		"34012": errors.New("the minimum withdrawal amount for NEO is 1, and the amount must be an integer"),
		"34013": errors.New("please transfer"),
		"34014": errors.New("transfer limited"),
		"34015": errors.New("subaccount does not exist"),
		"34016": errors.New("transfer suspended"),
		"34017": errors.New("account suspended"),
		"34018": errors.New("incorrect trades password"),
		"34019": errors.New("please bind your email before withdrawal"),
		"34020": errors.New("please bind your funds password before withdrawal"),
		"34021": errors.New("not verified address"),
		"34022": errors.New("withdrawals are not available for sub accounts"),
		"35001": errors.New("contract subscribing does not exist"),
		"35002": errors.New("contract is being settled"),
		"35003": errors.New("contract is being paused"),
		"35004": errors.New("pending contract settlement"),
		"35005": errors.New("perpetual swap trading is not enabled"),
		"35008": errors.New("margin ratio too low when placing order"),
		"35010": errors.New("closing position size larger than available size"),
		"35012": errors.New("placing an order with less than 1 contract"),
		"35014": errors.New("order size is not in acceptable range"),
		"35015": errors.New("leverage level unavailable"),
		"35017": errors.New("changing leverage level"),
		"35019": errors.New("order size exceeds limit"),
		"35020": errors.New("order price exceeds limit"),
		"35021": errors.New("order size exceeds limit of the current tier"),
		"35022": errors.New("contract is paused or closed"),
		"35030": errors.New("place multiple orders"),
		"35031": errors.New("cancel multiple orders"),
		"35061": errors.New("invalid instrument_id"),
	}
}

// SystemStatus represents system status
type SystemStatus struct {
	Title       string `json:"title"`
	State       string `json:"state"`
	Begin       string `json:"begin"`
	End         string `json:"end"`
	Href        string `json:"href"`
	ServiceType string `json:"serviceType"`
	System      string `json:"system"`
	ScheDesc    string `json:"scheDesc"`
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
	ExpTime        string         `json:"expTime"`
	InstFamily     string         `json:"instFamily"`
	InstrumentID   string         `json:"instId"`
	InstrumentType string         `json:"instType"`
	Leverage       string         `json:"lever"`
	ListTime       okcoinMilliSec `json:"listTime"`
	LotSize        string         `json:"lotSz"`
	MaxIcebergSz   string         `json:"maxIcebergSz"`
	MaxLimitSize   float64        `json:"maxLmtSz,string"`
	MaxMarketSize  float64        `json:"maxMktSz,string"`
	MaxStopSize    float64        `json:"maxStopSz,string"`
	MaxTwapSize    float64        `json:"maxTwapSz,string"`
	MaxTriggerSize float64        `json:"maxTriggerSz,string"`
	MinSize        float64        `json:"minSz,string"`
	QuoteCurrency  string         `json:"quoteCcy"`
	OptionType     string         `json:"optType"`
	SettleCurrency string         `json:"settleCcy"`
	State          string         `json:"state"`
	StrikePrice    string         `json:"stk"`
	TickSize       float64        `json:"tickSz,string"`
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
	Bal       float64        `json:"bal,string"`
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
	State               int64          `json:"state,string"`
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
	State         int64          `json:"state,string"`
	Timestamp     okcoinMilliSec `json:"ts"`
	WithdrawalID  string         `json:"wdId"`
}

// AccountBalanceInformation represents currency balance information.
type AccountBalanceInformation struct {
	AdjustedEquity string `json:"adjEq"` // Adjusted / Effective equity in USD . Not enabled. Please disregard.
	Details        []struct {
		AvailableBalance                 string         `json:"availBal"`
		AvaileEquity                     string         `json:"availEq"`
		CashBalance                      string         `json:"cashBal"`
		Currency                         string         `json:"ccy"`
		CrossLiability                   string         `json:"crossLiab"`
		DiscountEqutity                  string         `json:"disEq"`
		Equity                           string         `json:"eq"`
		EquityUsd                        string         `json:"eqUsd"`
		FixedBalance                     string         `json:"fixedBal"`
		FrozenBalance                    string         `json:"frozenBal"`
		Interest                         string         `json:"interest"`
		IsolatedEquity                   string         `json:"isoEq"`
		IsolatedLiability                string         `json:"isoLiab"`
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
	MarginRatio     string         `json:"mgnRatio"`
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
	Fee              string         `json:"fee"`
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
		BaseCurrency      string `json:"baseCcy"`
		BaseCurrencyIcon  string `json:"baseCcyIcon"`
		BaseSingleMin     string `json:"baseSingleMin"`
		InstrumentID      string `json:"instId"`
		QuoteCurrency     string `json:"quoteCcy"`
		QuoteCurrencyIcon string `json:"quoteCcyIcon"`
		QuoteSingleMin    string `json:"quoteSingleMin"`
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
	AskPx         string         `json:"askPx"`
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
	Price          string         `json:"px,string"`
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
	ChannelID            string `json:"chanId"`
	Currency             string `json:"ccy"`
	DepositQuota         string `json:"depQuota"`
	MinDeposit           string `json:"minDep"`
	WithdrawalQuota      string `json:"wdQuota"`
	MinWithdrawal        string `json:"minWd"`
	UsedWithdrawalQuota  string `json:"usedWdQuota"`
	ValidWithdrawalQuota string `json:"validWdQuota"`
	BankAccountInfo      []struct {
		BankAccountName   string `json:"bankAcctName"`
		BankAccountNum    string `json:"bankAcctNum"`
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
	ExpiryTime okcoinMilliSec `json:"expTime,string"`
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
	CancelOOrderIfAmendFail bool    `json:"cxlOnFail,omitempty"` // whether the order needs to be automatically canceled when the order amendment fails
	ClientOrderID           string  `json:"clOrdId,omitempty"`
	ClientRequestID         string  `json:"reqId,omitempty"`
	NewSize                 float64 `json:"newSz,string,omitempty"` // Conditional
	NewPrice                float64 `json:"newPx,string,omitempty"` // Conditional
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
	Fee                        float64        `json:"fee,string"`
	FeeCurrency                string         `json:"feeCcy"`
	FillPrice                  float64        `json:"fillPx,string"`
	FillSize                   float64        `json:"fillSz,string"`
	FillTime                   okcoinMilliSec `json:"fillTime"`
	InstrumentID               string         `json:"instId"`
	InstrumentType             string         `json:"instType"`
	Leverage                   float64        `json:"lever,string"`
	OrderID                    string         `json:"ordId"`
	OrderType                  string         `json:"ordType"`
	ProfitAndLoss              string         `json:"pnl"`
	PosSide                    string         `json:"posSide"`
	Price                      float64        `json:"px,string"`
	Rebate                     string         `json:"rebate"`
	RebateCurrency             string         `json:"rebateCcy"`
	ReduceOnly                 bool           `json:"reduceOnly,string"`
	Side                       string         `json:"side"`
	StopLossOrdPrice           string         `json:"slOrdPx"`
	StopLossTriggerPrice       string         `json:"slTriggerPx"`
	StopLossTriggerPriceType   string         `json:"slTriggerPxType"`
	Source                     string         `json:"source"`
	State                      string         `json:"state"`
	Size                       float64        `json:"sz,string"`
	Tag                        string         `json:"tag"`
	TradeMode                  string         `json:"tdMode"`
	TargetCurrency             string         `json:"tgtCcy"`
	TakeProfitOrderPrice       string         `json:"tpOrdPx"`
	TakeProfitTriggerPrice     string         `json:"tpTriggerPx"`
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
	Fee            float64        `json:"fee,string"`
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
	ActivePrice              string         `json:"activePx"`
	ActualPrice              string         `json:"actualPx"`
	ActualSide               string         `json:"actualSide"`
	ActualSize               string         `json:"actualSz"`
	AlgoID                   string         `json:"algoId"`
	CreateTime               okcoinMilliSec `json:"cTime"`
	CallbackRatio            string         `json:"callbackRatio"`
	CallbackSpread           string         `json:"callbackSpread"`
	Currency                 string         `json:"ccy"`
	ClientOrderID            string         `json:"clOrdId"`
	InstrumentID             string         `json:"instId"`
	InstrumentType           string         `json:"instType"`
	Leverage                 string         `json:"lever"`
	MoveTriggerPrice         string         `json:"moveTriggerPx"`
	OrderID                  string         `json:"ordId"`
	OrdPrice                 string         `json:"ordPx"`
	OrderType                string         `json:"ordType"`
	PosSide                  string         `json:"posSide"`
	PriceLimit               string         `json:"pxLimit"`
	PriceSpread              string         `json:"pxSpread"`
	PriceVar                 string         `json:"pxVar"`
	Side                     string         `json:"side"`
	StopLossOrdPrice         string         `json:"slOrdPx"`
	StopLossTriggerPrice     string         `json:"slTriggerPx"`
	StopLossTriggerPriceType string         `json:"slTriggerPxType"`
	State                    string         `json:"state"`
	Size                     string         `json:"sz"`
	SizeLimit                string         `json:"szLimit"`
	Tag                      string         `json:"tag"`
	TdMode                   string         `json:"tdMode"`
	TgtCcy                   string         `json:"tgtCcy"`
	TimeInterval             string         `json:"timeInterval"`
	TpOrdPrice               string         `json:"tpOrdPx"`
	TpTriggerPrice           string         `json:"tpTriggerPx"`
	TpTriggerPriceType       string         `json:"tpTriggerPxType"`
	TriggerPrice             string         `json:"triggerPx"`
	TriggerPriceType         string         `json:"triggerPxType"`
	TriggerTime              okcoinMilliSec `json:"triggerTime"`
}
