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
	Data        []interface{}              `json:"data"`
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
	PrevDeposited    int64         `json:"prevDeposited"`
	PrevWithdrawn    int64         `json:"prevWithdrawn"`
	PrevTransferIn   int64         `json:"prevTransferIn"`
	PrevTransferOut  int64         `json:"prevTransferOut"`
	PrevAmount       int64         `json:"prevAmount"`
	PrevTimestamp    string        `json:"prevTimestamp"`
	DeltaDeposited   int64         `json:"deltaDeposited"`
	DeltaWithdrawn   int64         `json:"deltaWithdrawn"`
	DeltaTransferIn  int64         `json:"deltaTransferIn"`
	DeltaTransferOut int64         `json:"deltaTransferOut"`
	DeltaAmount      int64         `json:"deltaAmount"`
	Deposited        int64         `json:"deposited"`
	Withdrawn        int64         `json:"withdrawn"`
	TransferIn       int64         `json:"transferIn"`
	TransferOut      int64         `json:"transferOut"`
	Amount           int64         `json:"amount"`
	PendingCredit    int64         `json:"pendingCredit"`
	PendingDebit     int64         `json:"pendingDebit"`
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
	RiskLimit          int64       `json:"riskLimit"`
	PrevState          string      `json:"prevState"`
	State              string      `json:"state"`
	Action             string      `json:"action"`
	Amount             int64       `json:"amount"`
	PendingCredit      int64       `json:"pendingCredit"`
	PendingDebit       int64       `json:"pendingDebit"`
	ConfirmedDebit     int64       `json:"confirmedDebit"`
	PrevRealisedPnl    int64       `json:"prevRealisedPnl"`
	PrevUnrealisedPnl  int64       `json:"prevUnrealisedPnl"`
	GrossComm          int64       `json:"grossComm"`
	GrossOpenCost      int64       `json:"grossOpenCost"`
	GrossOpenPremium   int64       `json:"grossOpenPremium"`
	GrossExecCost      int64       `json:"grossExecCost"`
	GrossMarkValue     int64       `json:"grossMarkValue"`
	RiskValue          int64       `json:"riskValue"`
	TaxableMargin      int64       `json:"taxableMargin"`
	InitMargin         int64       `json:"initMargin"`
	MaintMargin        int64       `json:"maintMargin"`
	SessionMargin      int64       `json:"sessionMargin"`
	TargetExcessMargin int64       `json:"targetExcessMargin"`
	VarMargin          int64       `json:"varMargin"`
	RealisedPnl        int64       `json:"realisedPnl"`
	UnrealisedPnl      int64       `json:"unrealisedPnl"`
	IndicativeTax      int64       `json:"indicativeTax"`
	UnrealisedProfit   int64       `json:"unrealisedProfit"`
	SyntheticMargin    interface{} `json:"syntheticMargin"`
	WalletBalance      int64       `json:"walletBalance"`
	MarginBalance      int64       `json:"marginBalance"`
	MarginBalancePcnt  int64       `json:"marginBalancePcnt"`
	MarginLeverage     int64       `json:"marginLeverage"`
	MarginUsedPcnt     int64       `json:"marginUsedPcnt"`
	ExcessMargin       int64       `json:"excessMargin"`
	ExcessMarginPcnt   int64       `json:"excessMarginPcnt"`
	AvailableMargin    int64       `json:"availableMargin"`
	WithdrawableMargin int64       `json:"withdrawableMargin"`
	Timestamp          string      `json:"timestamp"`
	GrossLastValue     int64       `json:"grossLastValue"`
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
