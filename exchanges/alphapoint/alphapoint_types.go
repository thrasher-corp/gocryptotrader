package alphapoint

type AlphapointTrade struct {
	TID                   int64   `json:"tid"`
	Price                 float64 `json:"px"`
	Quantity              float64 `json:"qty"`
	Unixtime              int     `json:"unixtime"`
	UTCTicks              int64   `json:"utcticks"`
	IncomingOrderSide     int     `json:"incomingOrderSide"`
	IncomingServerOrderID int     `json:"incomingServerOrderId"`
	BookServerOrderID     int     `json:"bookServerOrderId"`
}

type AlphapointTrades struct {
	IsAccepted   bool              `json:"isAccepted"`
	RejectReason string            `json:"rejectReason"`
	DateTimeUTC  int64             `json:"dateTimeUtc"`
	Instrument   string            `json:"ins"`
	StartIndex   int               `json:"startIndex"`
	Count        int               `json:"count"`
	Trades       []AlphapointTrade `json:"trades"`
}

type AlphapointTradesByDate struct {
	IsAccepted   bool              `json:"isAccepted"`
	RejectReason string            `json:"rejectReason"`
	DateTimeUTC  int64             `json:"dateTimeUtc"`
	Instrument   string            `json:"ins"`
	StartDate    int64             `json:"startDate"`
	EndDate      int64             `json:"endDate"`
	Trades       []AlphapointTrade `json:"trades"`
}

type AlphapointTicker struct {
	High               float64 `json:"high"`
	Last               float64 `json:"last"`
	Bid                float64 `json:"bid"`
	Volume             float64 `json:"volume"`
	Low                float64 `json:"low"`
	Ask                float64 `json:"ask"`
	Total24HrQtyTraded float64 `json:"Total24HrQtyTraded"`
	Total24HrNumTrades float64 `json:"Total24HrNumTrades"`
	SellOrderCount     float64 `json:"sellOrderCount"`
	BuyOrderCount      float64 `json:"buyOrderCount"`
	NumOfCreateOrders  float64 `json:"numOfCreateOrders"`
	IsAccepted         bool    `json:"isAccepted"`
	RejectReason       string  `json:"rejectReason"`
}

type AlphapointOrderbookEntry struct {
	Quantity float64 `json:"qty"`
	Price    float64 `json:"px"`
}

type AlphapointOrderbook struct {
	Bids         []AlphapointOrderbookEntry `json:"bids"`
	Asks         []AlphapointOrderbookEntry `json:"asks"`
	IsAccepted   bool                       `json:"isAccepted"`
	RejectReason string                     `json:"rejectReason"`
}

type AlphapointProductPair struct {
	Name                  string `json:"name"`
	Productpaircode       int    `json:"productPairCode"`
	Product1Label         string `json:"product1Label"`
	Product1Decimalplaces int    `json:"product1DecimalPlaces"`
	Product2Label         string `json:"product2Label"`
	Product2Decimalplaces int    `json:"product2DecimalPlaces"`
}

type AlphapointProductPairs struct {
	ProductPairs []AlphapointProductPair `json:"productPairs"`
	IsAccepted   bool                    `json:"isAccepted"`
	RejectReason string                  `json:"rejectReason"`
}

type AlphapointProduct struct {
	Name          string `json:"name"`
	IsDigital     bool   `json:"isDigital"`
	ProductCode   int    `json:"productCode"`
	DecimalPlaces int    `json:"decimalPlaces"`
	FullName      string `json:"fullName"`
}

type AlphapointProducts struct {
	Products     []AlphapointProduct `json:"products"`
	IsAccepted   bool                `json:"isAccepted"`
	RejectReason string              `json:"rejectReason"`
}

type AlphapointUserInfo struct {
	UserInfoKVP []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	} `json:"userInfoKVP"`
	IsAccepted   bool   `json:"isAccepted"`
	RejectReason string `json:"rejectReason"`
}

type AlphapointAccountInfo struct {
	Currencies []struct {
		Name    string `json:"name"`
		Balance int    `json:"balance"`
		Hold    int    `json:"hold"`
	} `json:"currencies"`
	ProductPairs []struct {
		ProductPairName string `json:"productPairName"`
		ProductPairCode int    `json:"productPairCode"`
		TradeCount      int    `json:"tradeCount"`
		TradeVolume     int    `json:"tradeVolume"`
	} `json:"productPairs"`
	IsAccepted   bool   `json:"isAccepted"`
	RejectReason string `json:"rejectReason"`
}

type AlphapointOrder struct {
	Serverorderid int   `json:"ServerOrderId"`
	AccountID     int   `json:"AccountId"`
	Price         int   `json:"Price"`
	QtyTotal      int   `json:"QtyTotal"`
	QtyRemaining  int   `json:"QtyRemaining"`
	ReceiveTime   int64 `json:"ReceiveTime"`
	Side          int   `json:"Side"`
}

type AlphapointOpenOrders struct {
	Instrument string            `json:"ins"`
	Openorders []AlphapointOrder `json:"openOrders"`
}

type AlphapointOrderInfo struct {
	OpenOrders   []AlphapointOpenOrders `json:"openOrdersInfo"`
	IsAccepted   bool                   `json:"isAccepted"`
	DateTimeUTC  int64                  `json:"dateTimeUtc"`
	RejectReason string                 `json:"rejectReason"`
}

type AlphapointDepositAddresses struct {
	Name           string `json:"name"`
	DepositAddress string `json:"depositAddress"`
}

type AlphapointWebsocketTicker struct {
	MessageType             string  `json:"messageType"`
	ProductPair             string  `json:"prodPair"`
	High                    float64 `json:"high"`
	Low                     float64 `json:"low"`
	Last                    float64 `json:"last"`
	Volume                  float64 `json:"volume"`
	Volume24Hrs             float64 `json:"volume24hrs"`
	Volume24HrsProduct2     float64 `json:"volume24hrsProduct2"`
	Total24HrQtyTraded      float64 `json:"Total24HrQtyTraded"`
	Total24HrProduct2Traded float64 `json:"Total24HrProduct2Traded"`
	Total24HrNumTrades      float64 `json:"Total24HrNumTrades"`
	Bid                     float64 `json:"bid"`
	Ask                     float64 `json:"ask"`
	BuyOrderCount           int     `json:"buyOrderCount"`
	SellOrderCount          int     `json:"sellOrderCount"`
}
