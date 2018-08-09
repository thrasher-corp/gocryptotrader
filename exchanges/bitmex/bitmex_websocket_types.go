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
