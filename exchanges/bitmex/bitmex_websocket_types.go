package bitmex

import (
	"time"
)

// WebsocketRequest is the main request type
type WebsocketRequest struct {
	Command   string `json:"op"`
	Arguments []any  `json:"args"`
}

// WebsocketErrorResponse main error response
type WebsocketErrorResponse struct {
	Status  int64            `json:"status"`
	Error   string           `json:"error"`
	Meta    any              `json:"meta"`
	Request WebsocketRequest `json:"request"`
}

// WebsocketWelcome initial welcome type
type WebsocketWelcome struct {
	Info      string `json:"info"`
	Version   string `json:"version"`
	Timestamp string `json:"timestamp"`
	Docs      string `json:"docs"`
	Limit     struct {
		Remaining int64 `json:"remaining"`
	} `json:"limit"`
}

// WebsocketSubscribeResp is a response that occurs after a subscription
type WebsocketSubscribeResp struct {
	Success   bool             `json:"success"`
	Subscribe string           `json:"subscribe"`
	Request   WebsocketRequest `json:"request"`
}

// WebsocketMainResponse main table defined response
type WebsocketMainResponse struct {
	Table string   `json:"table"`
	Keys  []string `json:"keys"`
	Types struct {
		ID     string `json:"id"`
		Price  string `json:"price"`
		Side   string `json:"side"`
		Size   string `json:"size"`
		Symbol string `json:"symbol"`
	} `json:"types"`
	ForeignKeys struct {
		Side   string `json:"side"`
		Symbol string `json:"symbol"`
	} `json:"foreignKeys"`
	Attributes struct {
		ID     string `json:"id"`
		Symbol string `json:"symbol"`
	} `json:"Attributes"`
	Action string `json:"action,omitempty"`
}

// OrderBookData contains orderbook resp data with action to be taken
type OrderBookData struct {
	Data   []OrderBookL2 `json:"data"`
	Action string        `json:"action"`
}

// TradeData contains trade resp data with action to be taken
type TradeData struct {
	Data   []*Trade `json:"data"`
	Action string   `json:"action"`
}

// AnnouncementData contains announcement resp data with action to be taken
type AnnouncementData struct {
	Data   []Announcement `json:"data"`
	Action string         `json:"action"`
}

// WsAffiliateResponse private api response
type WsAffiliateResponse struct {
	WsDataResponse
	ForeignKeys any                           `json:"foreignKeys"`
	Attributes  WsAffiliateResponseAttributes `json:"attributes"`
	Filter      WsAffiliateResponseFilter     `json:"filter"`
	Data        []any                         `json:"data"`
}

// WsAffiliateResponseAttributes private api data
type WsAffiliateResponseAttributes struct {
	Account  string `json:"account"`
	Currency string `json:"currency"`
}

// WsAffiliateResponseFilter private api data
type WsAffiliateResponseFilter struct {
	Account int64 `json:"account"`
}

// WsOrderResponse private api response
type WsOrderResponse struct {
	WsDataResponse
	ForeignKeys WsOrderResponseForeignKeys `json:"foreignKeys"`
	Attributes  WsOrderResponseAttributes  `json:"attributes"`
	Filter      WsOrderResponseFilter      `json:"filter"`
	Data        []OrderInsertData          `json:"data"`
}

// OrderInsertData holds order data from an order response
type OrderInsertData struct {
	WorkingIndicator      bool      `json:"workingIndicator"`
	Account               int64     `json:"account"`
	AveragePrice          float64   `json:"avgPx"`
	Commission            float64   `json:"commission"`
	FilledQuantity        float64   `json:"cumQty"`
	DisplayQuantity       float64   `json:"displayQty"`
	ExecComm              float64   `json:"execComm"`
	ExecCost              float64   `json:"execCost"`
	ForeignNotional       float64   `json:"foreignNotional"`
	HomeNotional          float64   `json:"homeNotional"`
	LastPrice             float64   `json:"lastPx"`
	LastQuantity          float64   `json:"lastQty"`
	LeavesQuantity        float64   `json:"leavesQty"`
	OrderQuantity         float64   `json:"orderQty"`
	PegOffsetValue        float64   `json:"pegOffsetValue"`
	Price                 float64   `json:"price"`
	SimpleFilledQuantity  float64   `json:"simpleCumQty"`
	SimpleLeavesQuantity  float64   `json:"simpleLeavesQty"`
	SimpleOrderQuantity   float64   `json:"simpleOrderQty"`
	StopPrice             float64   `json:"stopPx"`
	ExDestination         string    `json:"exDestination"`
	ContingencyType       string    `json:"contingencyType"`
	Currency              string    `json:"currency"`
	ExecutionID           string    `json:"execID"`
	ExecutionInstance     string    `json:"execInst"`
	ExecutionType         string    `json:"execType"`
	LastLiquidityInd      string    `json:"lastLiquidityInd"`
	LastMkt               string    `json:"lastMkt"`
	UnderlyingLastPrice   float64   `json:"underlyingLastPx"`
	MultiLegReportingType string    `json:"multiLegReportingType"`
	OrderRejectedReason   string    `json:"ordRejReason"`
	OrderStatus           string    `json:"ordStatus"`
	OrderType             string    `json:"ordType"`
	OrderID               string    `json:"orderID"`
	PegPriceType          string    `json:"pegPriceType"`
	ClientOrderID         string    `json:"clOrdID"`
	ClientOrderLinkID     string    `json:"clOrdLinkID"`
	Symbol                string    `json:"symbol"`
	Text                  string    `json:"text"`
	TimeInForce           string    `json:"timeInForce"`
	Timestamp             time.Time `json:"timestamp"`
	TradePublishIndicator string    `json:"tradePublishIndicator"`
	TransactTime          time.Time `json:"transactTime"`
	TradingMatchID        string    `json:"trdMatchID"`
	Triggered             string    `json:"triggered"`
	SettleCurrency        string    `json:"settlCurrency"`
	Side                  string    `json:"side"`
}

// WsOrderResponseAttributes private api data
type WsOrderResponseAttributes struct {
	OrderID          string `json:"orderID"`
	Account          string `json:"account"`
	OrdStatus        string `json:"ordStatus"`
	WorkingIndicator string `json:"workingIndicator"`
}

// WsOrderResponseFilter private api data
type WsOrderResponseFilter struct {
	Account int64 `json:"account"`
}

// WsOrderResponseForeignKeys private api data
type WsOrderResponseForeignKeys struct {
	Symbol    string `json:"symbol"`
	Side      string `json:"side"`
	OrdStatus string `json:"ordStatus"`
}

// WsTransactResponse private api response
type WsTransactResponse struct {
	WsDataResponse
	ForeignKeys any                          `json:"foreignKeys"`
	Attributes  WsTransactResponseAttributes `json:"attributes"`
	Filter      WsTransactResponseFilter     `json:"filter"`
	Data        []any                        `json:"data"`
}

// WsTransactResponseAttributes private api data
type WsTransactResponseAttributes struct {
	TransactID   string `json:"transactID"`
	TransactTime string `json:"transactTime"`
}

// WsTransactResponseFilter private api data
type WsTransactResponseFilter struct {
	Account int64 `json:"account"`
}

// WsWalletResponse private api response
type WsWalletResponse struct {
	WsDataResponse
	ForeignKeys any                        `json:"foreignKeys"`
	Attributes  WsWalletResponseAttributes `json:"attributes"`
	Filter      WsWalletResponseFilter     `json:"filter"`
	Data        []WsWalletResponseData     `json:"data"`
}

// WsWalletResponseAttributes private api data
type WsWalletResponseAttributes struct {
	Account  string `json:"account"`
	Currency string `json:"currency"`
}

// WsWalletResponseData private api data
type WsWalletResponseData struct {
	Account          int64   `json:"account"`
	Currency         string  `json:"currency"`
	PrevDeposited    float64 `json:"prevDeposited"`
	PrevWithdrawn    float64 `json:"prevWithdrawn"`
	PrevTransferIn   float64 `json:"prevTransferIn"`
	PrevTransferOut  float64 `json:"prevTransferOut"`
	PrevAmount       float64 `json:"prevAmount"`
	PrevTimestamp    string  `json:"prevTimestamp"`
	DeltaDeposited   float64 `json:"deltaDeposited"`
	DeltaWithdrawn   float64 `json:"deltaWithdrawn"`
	DeltaTransferIn  float64 `json:"deltaTransferIn"`
	DeltaTransferOut float64 `json:"deltaTransferOut"`
	DeltaAmount      float64 `json:"deltaAmount"`
	Deposited        float64 `json:"deposited"`
	Withdrawn        float64 `json:"withdrawn"`
	TransferIn       float64 `json:"transferIn"`
	TransferOut      float64 `json:"transferOut"`
	Amount           float64 `json:"amount"`
	PendingCredit    float64 `json:"pendingCredit"`
	PendingDebit     float64 `json:"pendingDebit"`
	ConfirmedDebit   int64   `json:"confirmedDebit"`
	Timestamp        string  `json:"timestamp"`
	Addr             string  `json:"addr"`
	Script           string  `json:"script"`
	WithdrawalLock   []any   `json:"withdrawalLock"`
}

// WsWalletResponseFilter private api data
type WsWalletResponseFilter struct {
	Account int64 `json:"account"`
}

// WsExecutionResponse private api response
type WsExecutionResponse struct {
	WsDataResponse
	ForeignKeys WsExecutionResponseForeignKeys `json:"foreignKeys"`
	Attributes  WsExecutionResponseAttributes  `json:"attributes"`
	Filter      WsExecutionResponseFilter      `json:"filter"`
	Data        []wsExecutionData              `json:"data"`
}

type wsExecutionData struct {
	WorkingIndicator      bool      `json:"workingIndicator"`
	Account               int64     `json:"account"`
	AvgPx                 float64   `json:"avgPx"`
	Commission            float64   `json:"commission"`
	FilledQuantity        float64   `json:"cumQty"`
	DisplayQuantity       float64   `json:"displayQty"`
	ExecComm              float64   `json:"execComm"`
	ExecCost              float64   `json:"execCost"`
	ForeignNotional       float64   `json:"foreignNotional"`
	HomeNotional          float64   `json:"homeNotional"`
	LastPx                float64   `json:"lastPx"`
	LastQuantity          float64   `json:"lastQty"`
	LeavesQuantity        float64   `json:"leavesQty"`
	OrderQuantity         float64   `json:"orderQty"`
	PegOffsetValue        float64   `json:"pegOffsetValue"`
	Price                 float64   `json:"price"`
	SimpleFilledQuantity  float64   `json:"simpleCumQty"`
	SimpleLeavesQuantity  float64   `json:"simpleLeavesQty"`
	SimpleOrderQuantity   float64   `json:"simpleOrderQty"`
	StopPx                float64   `json:"stopPx"`
	UnderlyingLastPx      float64   `json:"underlyingLastPx"`
	PegPriceType          string    `json:"pegPriceType"`
	Symbol                string    `json:"symbol"`
	Text                  string    `json:"text"`
	TimeInForce           string    `json:"timeInForce"`
	Timestamp             time.Time `json:"timestamp"`
	TradePublishIndicator string    `json:"tradePublishIndicator"`
	TransactTime          time.Time `json:"transactTime"`
	TrdMatchID            string    `json:"trdMatchID"`
	Triggered             string    `json:"triggered"`
	ClOrdID               string    `json:"clOrdID"`
	ClOrdLinkID           string    `json:"clOrdLinkID"`
	SettlCurrency         string    `json:"settlCurrency"`
	Side                  string    `json:"side"`
	MultiLegReportingType string    `json:"multiLegReportingType"`
	OrdRejReason          string    `json:"ordRejReason"`
	OrdStatus             string    `json:"ordStatus"`
	OrdType               string    `json:"ordType"`
	OrderID               string    `json:"orderID"`
	LastLiquidityInd      string    `json:"lastLiquidityInd"`
	LastMkt               string    `json:"lastMkt"`
	ExecID                string    `json:"execID"`
	ExecInst              string    `json:"execInst"`
	ExecType              string    `json:"execType"`
	ExDestination         string    `json:"exDestination"`
	Currency              string    `json:"currency"`
	ContingencyType       string    `json:"contingencyType"`
}

// WsExecutionResponseAttributes private api data
type WsExecutionResponseAttributes struct {
	ExecID       string `json:"execID"`
	Account      string `json:"account"`
	ExecType     string `json:"execType"`
	TransactTime string `json:"transactTime"`
}

// WsExecutionResponseFilter private api data
type WsExecutionResponseFilter struct {
	Account int64  `json:"account"`
	Symbol  string `json:"symbol"`
}

// WsExecutionResponseForeignKeys private api data
type WsExecutionResponseForeignKeys struct {
	Symbol    string `json:"symbol"`
	Side      string `json:"side"`
	OrdStatus string `json:"ordStatus"`
}

// WsDataResponse contains common elements
type WsDataResponse struct {
	Table  string            `json:"table"`
	Action string            `json:"action"`
	Keys   []string          `json:"keys"`
	Types  map[string]string `json:"types"`
}

// WsMarginResponse private api response
type WsMarginResponse struct {
	WsDataResponse
	ForeignKeys any                        `json:"foreignKeys"`
	Attributes  WsMarginResponseAttributes `json:"attributes"`
	Filter      WsMarginResponseFilter     `json:"filter"`
	Data        []WsMarginResponseData     `json:"data"`
}

// WsMarginResponseAttributes private api data
type WsMarginResponseAttributes struct {
	Account  string `json:"account"`
	Currency string `json:"currency"`
}

// WsMarginResponseData private api data
type WsMarginResponseData struct {
	Account            int64     `json:"account"`
	Currency           string    `json:"currency"`
	RiskLimit          float64   `json:"riskLimit"`
	PrevState          string    `json:"prevState"`
	State              string    `json:"state"`
	Action             string    `json:"action"`
	Amount             float64   `json:"amount"`
	PendingCredit      float64   `json:"pendingCredit"`
	PendingDebit       float64   `json:"pendingDebit"`
	ConfirmedDebit     float64   `json:"confirmedDebit"`
	PrevRealisedPnl    float64   `json:"prevRealisedPnl"`
	PrevUnrealisedPnl  float64   `json:"prevUnrealisedPnl"`
	GrossComm          float64   `json:"grossComm"`
	GrossOpenCost      float64   `json:"grossOpenCost"`
	GrossOpenPremium   float64   `json:"grossOpenPremium"`
	GrossExecCost      float64   `json:"grossExecCost"`
	GrossMarkValue     float64   `json:"grossMarkValue"`
	RiskValue          float64   `json:"riskValue"`
	TaxableMargin      float64   `json:"taxableMargin"`
	InitMargin         float64   `json:"initMargin"`
	MaintMargin        float64   `json:"maintMargin"`
	SessionMargin      float64   `json:"sessionMargin"`
	TargetExcessMargin float64   `json:"targetExcessMargin"`
	VarMargin          float64   `json:"varMargin"`
	RealisedPnl        float64   `json:"realisedPnl"`
	UnrealisedPnl      float64   `json:"unrealisedPnl"`
	IndicativeTax      float64   `json:"indicativeTax"`
	UnrealisedProfit   float64   `json:"unrealisedProfit"`
	SyntheticMargin    any       `json:"syntheticMargin"`
	WalletBalance      float64   `json:"walletBalance"`
	MarginBalance      float64   `json:"marginBalance"`
	MarginBalancePcnt  float64   `json:"marginBalancePcnt"`
	MarginLeverage     float64   `json:"marginLeverage"`
	MarginUsedPcnt     float64   `json:"marginUsedPcnt"`
	ExcessMargin       float64   `json:"excessMargin"`
	ExcessMarginPcnt   float64   `json:"excessMarginPcnt"`
	AvailableMargin    float64   `json:"availableMargin"`
	WithdrawableMargin float64   `json:"withdrawableMargin"`
	Timestamp          time.Time `json:"timestamp"`
	GrossLastValue     float64   `json:"grossLastValue"`
	Commission         any       `json:"commission"`
}

// WsMarginResponseFilter private api data
type WsMarginResponseFilter struct {
	Account int64 `json:"account"`
}

// WsPositionResponse private api response
type WsPositionResponse struct {
	WsDataResponse
	ForeignKeys WsPositionResponseForeignKeys `json:"foreignKeys"`
	Attributes  WsPositionResponseAttributes  `json:"attributes"`
	Filter      WsPositionResponseFilter      `json:"filter"`
	Data        []wsPositionData              `json:"data"`
}

type wsPositionData struct {
	IsOpen               bool      `json:"isOpen"`
	Account              int64     `json:"account"`
	CurrentQuantity      float64   `json:"currentQty"`
	HomeNotional         float64   `json:"homeNotional"`
	LiquidationPrice     float64   `json:"liquidationPrice"`
	MaintMargin          float64   `json:"maintMargin"`
	MarkPrice            float64   `json:"markPrice"`
	MarkValue            float64   `json:"markValue"`
	RiskValue            float64   `json:"riskValue"`
	SimpleQuantity       float64   `json:"simpleQty"`
	UnrealisedGrossPnl   float64   `json:"unrealisedGrossPnl"`
	UnrealisedPnl        float64   `json:"unrealisedPnl"`
	UnrealisedPnlPcnt    float64   `json:"unrealisedPnlPcnt"`
	UnrealisedRoePcnt    float64   `json:"unrealisedRoePcnt"`
	BankruptPrice        float64   `json:"bankruptPrice"`
	AvgCostPrice         float64   `json:"avgCostPrice"`
	AvgEntryPrice        float64   `json:"avgEntryPrice"`
	BreakEvenPrice       float64   `json:"breakEvenPrice"`
	CurrentComm          float64   `json:"currentComm"`
	CurrentCost          float64   `json:"currentCost"`
	DeleveragePercentile float64   `json:"deleveragePercentile"`
	ExecComm             float64   `json:"execComm"`
	ExecCost             float64   `json:"execCost"`
	ExecQuantity         float64   `json:"execQty"`
	ExecSellCost         float64   `json:"execSellCost"`
	ExecSellQuantity     float64   `json:"execSellQty"`
	ForeignNotional      float64   `json:"foreignNotional"`
	GrossExecCost        float64   `json:"grossExecCost"`
	MarginCallPrice      float64   `json:"marginCallPrice"`
	PosComm              float64   `json:"posComm"`
	PosCost              float64   `json:"posCost"`
	PosCost2             float64   `json:"posCost2"`
	PosInit              float64   `json:"posInit"`
	PosMaint             float64   `json:"posMaint"`
	PosMargin            float64   `json:"posMargin"`
	PrevRealisedPnl      float64   `json:"prevRealisedPnl"`
	RealisedCost         float64   `json:"realisedCost"`
	RealisedGrossPnl     float64   `json:"realisedGrossPnl"`
	RealisedPnl          float64   `json:"realisedPnl"`
	RebalancedPnl        float64   `json:"rebalancedPnl"`
	SimpleCost           float64   `json:"simpleCost"`
	SimpleValue          float64   `json:"simpleValue"`
	UnrealisedCost       float64   `json:"unrealisedCost"`
	Currency             string    `json:"currency"`
	CurrentTimestamp     time.Time `json:"currentTimestamp"`
	Symbol               string    `json:"symbol"`
	Timestamp            time.Time `json:"timestamp"`
	PosState             string    `json:"posState"`
}

// WsPositionResponseAttributes private api data
type WsPositionResponseAttributes struct {
	Account       string `json:"account"`
	Symbol        string `json:"symbol"`
	Currency      string `json:"currency"`
	Underlying    string `json:"underlying"`
	QuoteCurrency string `json:"quoteCurrency"`
}

// WsPositionResponseFilter private api data
type WsPositionResponseFilter struct {
	Account int64  `json:"account"`
	Symbol  string `json:"symbol"`
}

// WsPositionResponseForeignKeys private api data
type WsPositionResponseForeignKeys struct {
	Symbol string `json:"symbol"`
}

// WsPrivateNotificationsResponse private api response
type WsPrivateNotificationsResponse struct {
	Table  string `json:"table"`
	Action string `json:"action"`
	Data   []any  `json:"data"`
}
