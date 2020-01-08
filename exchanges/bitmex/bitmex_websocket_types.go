package bitmex

// WebsocketRequest is the main request type
type WebsocketRequest struct {
	Command   string        `json:"op"`
	Arguments []interface{} `json:"args"`
}

// WebsocketErrorResponse main error response
type WebsocketErrorResponse struct {
	Status  int              `json:"status"`
	Error   string           `json:"error"`
	Meta    interface{}      `json:"meta"`
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
}

// OrderBookData contains orderbook resp data with action to be taken
type OrderBookData struct {
	Data   []OrderBookL2 `json:"data"`
	Action string        `json:"action"`
}

// TradeData contains trade resp data with action to be taken
type TradeData struct {
	Data   []Trade `json:"data"`
	Action string  `json:"action"`
}

// AnnouncementData contains announcement resp data with action to be taken
type AnnouncementData struct {
	Data   []Announcement `json:"data"`
	Action string         `json:"action"`
}

// WsAffiliateResponse private api response
type WsAffiliateResponse struct {
	WsDataResponse
	ForeignKeys interface{}                   `json:"foreignKeys"`
	Attributes  WsAffiliateResponseAttributes `json:"attributes"`
	Filter      WsAffiliateResponseFilter     `json:"filter"`
	Data        []interface{}                 `json:"data"`
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
	Data        []OrderInsert              `json:"data"`
}

// OrderInsert
type OrderInsert struct {
	Account               int64       `json:"account"`
	AvgPx                 float64     `json:"avgPx"`
	ClOrdID               string      `json:"clOrdID"`
	ClOrdLinkID           string      `json:"clOrdLinkID"`
	Commission            float64     `json:"commission"`
	ContingencyType       string      `json:"contingencyType"`
	CumQty                int64       `json:"cumQty"`
	Currency              string      `json:"currency"`
	DisplayQty            int64       `json:"displayQty"`
	ExDestination         string      `json:"exDestination"`
	ExecComm              int64       `json:"execComm"`
	ExecCost              int64       `json:"execCost"`
	ExecID                string      `json:"execID"`
	ExecInst              string      `json:"execInst"`
	ExecType              string      `json:"execType"`
	ForeignNotional       int64       `json:"foreignNotional"`
	HomeNotional          float64     `json:"homeNotional"`
	LastLiquidityInd      string      `json:"lastLiquidityInd"`
	LastMkt               string      `json:"lastMkt"`
	LastPx                float64     `json:"lastPx"`
	LastQty               int64       `json:"lastQty"`
	LeavesQty             int64       `json:"leavesQty"`
	MultiLegReportingType string      `json:"multiLegReportingType"`
	OrdRejReason          string      `json:"ordRejReason"`
	OrdStatus             string      `json:"ordStatus"`
	OrdType               string      `json:"ordType"`
	OrderID               string      `json:"orderID"`
	OrderQty              int64       `json:"orderQty"`
	PegOffsetValue        float64     `json:"pegOffsetValue"`
	PegPriceType          string      `json:"pegPriceType"`
	Price                 float64     `json:"price"`
	SettlCurrency         string      `json:"settlCurrency"`
	Side                  string      `json:"side"`
	SimpleCumQty          float64     `json:"simpleCumQty"`
	SimpleLeavesQty       int64       `json:"simpleLeavesQty"`
	SimpleOrderQty        int64       `json:"simpleOrderQty"`
	StopPx                float64     `json:"stopPx"`
	Symbol                string      `json:"symbol"`
	Text                  string      `json:"text"`
	TimeInForce           string      `json:"timeInForce"`
	Timestamp             string      `json:"timestamp"`
	TradePublishIndicator string      `json:"tradePublishIndicator"`
	TransactTime          string      `json:"transactTime"`
	TrdMatchID            string      `json:"trdMatchID"`
	Triggered             string      `json:"triggered"`
	UnderlyingLastPx      interface{} `json:"underlyingLastPx"`
	WorkingIndicator      bool        `json:"workingIndicator"`
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
	ForeignKeys interface{}                  `json:"foreignKeys"`
	Attributes  WsTransactResponseAttributes `json:"attributes"`
	Filter      WsTransactResponseFilter     `json:"filter"`
	Data        []interface{}                `json:"data"`
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
	ForeignKeys interface{}                `json:"foreignKeys"`
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
	Account          int64         `json:"account"`
	Currency         string        `json:"currency"`
	PrevDeposited    float64       `json:"prevDeposited"`
	PrevWithdrawn    float64       `json:"prevWithdrawn"`
	PrevTransferIn   float64       `json:"prevTransferIn"`
	PrevTransferOut  float64       `json:"prevTransferOut"`
	PrevAmount       float64       `json:"prevAmount"`
	PrevTimestamp    string        `json:"prevTimestamp"`
	DeltaDeposited   float64       `json:"deltaDeposited"`
	DeltaWithdrawn   float64       `json:"deltaWithdrawn"`
	DeltaTransferIn  float64       `json:"deltaTransferIn"`
	DeltaTransferOut float64       `json:"deltaTransferOut"`
	DeltaAmount      float64       `json:"deltaAmount"`
	Deposited        float64       `json:"deposited"`
	Withdrawn        float64       `json:"withdrawn"`
	TransferIn       float64       `json:"transferIn"`
	TransferOut      float64       `json:"transferOut"`
	Amount           float64       `json:"amount"`
	PendingCredit    float64       `json:"pendingCredit"`
	PendingDebit     float64       `json:"pendingDebit"`
	ConfirmedDebit   int64         `json:"confirmedDebit"`
	Timestamp        string        `json:"timestamp"`
	Addr             string        `json:"addr"`
	Script           string        `json:"script"`
	WithdrawalLock   []interface{} `json:"withdrawalLock"`
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
	Data        []interface{}                  `json:"data"`
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
	ForeignKeys interface{}                `json:"foreignKeys"`
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
	Account            int64       `json:"account"`
	Currency           string      `json:"currency"`
	RiskLimit          float64     `json:"riskLimit"`
	PrevState          string      `json:"prevState"`
	State              string      `json:"state"`
	Action             string      `json:"action"`
	Amount             float64     `json:"amount"`
	PendingCredit      float64     `json:"pendingCredit"`
	PendingDebit       float64     `json:"pendingDebit"`
	ConfirmedDebit     float64     `json:"confirmedDebit"`
	PrevRealisedPnl    float64     `json:"prevRealisedPnl"`
	PrevUnrealisedPnl  float64     `json:"prevUnrealisedPnl"`
	GrossComm          float64     `json:"grossComm"`
	GrossOpenCost      float64     `json:"grossOpenCost"`
	GrossOpenPremium   float64     `json:"grossOpenPremium"`
	GrossExecCost      float64     `json:"grossExecCost"`
	GrossMarkValue     float64     `json:"grossMarkValue"`
	RiskValue          float64     `json:"riskValue"`
	TaxableMargin      float64     `json:"taxableMargin"`
	InitMargin         float64     `json:"initMargin"`
	MaintMargin        float64     `json:"maintMargin"`
	SessionMargin      float64     `json:"sessionMargin"`
	TargetExcessMargin float64     `json:"targetExcessMargin"`
	VarMargin          float64     `json:"varMargin"`
	RealisedPnl        float64     `json:"realisedPnl"`
	UnrealisedPnl      float64     `json:"unrealisedPnl"`
	IndicativeTax      float64     `json:"indicativeTax"`
	UnrealisedProfit   float64     `json:"unrealisedProfit"`
	SyntheticMargin    interface{} `json:"syntheticMargin"`
	WalletBalance      float64     `json:"walletBalance"`
	MarginBalance      float64     `json:"marginBalance"`
	MarginBalancePcnt  float64     `json:"marginBalancePcnt"`
	MarginLeverage     float64     `json:"marginLeverage"`
	MarginUsedPcnt     float64     `json:"marginUsedPcnt"`
	ExcessMargin       float64     `json:"excessMargin"`
	ExcessMarginPcnt   float64     `json:"excessMarginPcnt"`
	AvailableMargin    float64     `json:"availableMargin"`
	WithdrawableMargin float64     `json:"withdrawableMargin"`
	Timestamp          string      `json:"timestamp"`
	GrossLastValue     float64     `json:"grossLastValue"`
	Commission         interface{} `json:"commission"`
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
	Data        []interface{}                 `json:"data"`
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
	Table  string        `json:"table"`
	Action string        `json:"action"`
	Data   []interface{} `json:"data"`
}
