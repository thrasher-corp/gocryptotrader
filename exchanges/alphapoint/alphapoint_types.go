package alphapoint

import (
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Response contains general responses from the exchange
type Response struct {
	IsAccepted    bool       `json:"isAccepted"`
	RejectReason  string     `json:"rejectReason"`
	Fee           float64    `json:"fee"`
	FeeProduct    string     `json:"feeProduct"`
	CancelOrderID int64      `json:"cancelOrderId"`
	ServerOrderID int64      `json:"serverOrderId"`
	DateTimeUTC   types.Time `json:"dateTimeUtc"`
	ModifyOrderID int64      `json:"modifyOrderId"`
	Addresses     []DepositAddresses
}

// Ticker holds ticker information
type Ticker struct {
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

// Trades holds trade information
type Trades struct {
	IsAccepted   bool       `json:"isAccepted"`
	RejectReason string     `json:"rejectReason"`
	DateTimeUTC  types.Time `json:"dateTimeUtc"`
	Instrument   string     `json:"ins"`
	StartIndex   int        `json:"startIndex"`
	Count        int        `json:"count"`
	StartDate    int64      `json:"startDate"`
	EndDate      int64      `json:"endDate"`
	Trades       []Trade    `json:"trades"`
}

// Trade is a sub-type which holds the singular trade that occurred in the past
type Trade struct {
	TID                   int64   `json:"tid"`
	Price                 float64 `json:"px"`
	Quantity              float64 `json:"qty"`
	Unixtime              int     `json:"unixtime"`
	UTCTicks              int64   `json:"utcticks"`
	IncomingOrderSide     int     `json:"incomingOrderSide"`
	IncomingServerOrderID int     `json:"incomingServerOrderId"`
	BookServerOrderID     int     `json:"bookServerOrderId"`
}

// Orderbook holds the total Bids and Asks on the exchange
type Orderbook struct {
	Bids         []OrderbookEntry `json:"bids"`
	Asks         []OrderbookEntry `json:"asks"`
	IsAccepted   bool             `json:"isAccepted"`
	RejectReason string           `json:"rejectReason"`
}

// OrderbookEntry is a sub-type that takes has the individual quantity and price
type OrderbookEntry struct {
	Quantity float64 `json:"qty"`
	Price    float64 `json:"px"`
}

// ProductPairs holds the full range of product pairs that the exchange can
// trade between
type ProductPairs struct {
	ProductPairs []ProductPair `json:"productPairs"`
	IsAccepted   bool          `json:"isAccepted"`
	RejectReason string        `json:"rejectReason"`
}

// ProductPair holds the individual product pairs that are currently traded
type ProductPair struct {
	Name                  string `json:"name"`
	Productpaircode       int    `json:"productPairCode"`
	Product1Label         string `json:"product1Label"`
	Product1Decimalplaces int    `json:"product1DecimalPlaces"`
	Product2Label         string `json:"product2Label"`
	Product2Decimalplaces int    `json:"product2DecimalPlaces"`
}

// Products holds the full range of supported currency products
type Products struct {
	Products     []Product `json:"products"`
	IsAccepted   bool      `json:"isAccepted"`
	RejectReason string    `json:"rejectReason"`
}

// Product holds the a single currency product that is supported
type Product struct {
	Name          string `json:"name"`
	IsDigital     bool   `json:"isDigital"`
	ProductCode   int    `json:"productCode"`
	DecimalPlaces int    `json:"decimalPlaces"`
	FullName      string `json:"fullName"`
}

// UserInfo holds current user information associated with the apiKey details
type UserInfo struct {
	UserInforKVPs []UserInfoKVP `json:"userInfoKVP"`
	IsAccepted    bool          `json:"isAccepted"`
	RejectReason  string        `json:"rejectReason"`
}

// UserInfoKVP is a sub-type that holds key value pairs
type UserInfoKVP struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// UserInfoSet is the returned response from set user information request
type UserInfoSet struct {
	IsAccepted        string `json:"isAccepted"`
	RejectReason      string `json:"rejectReason"`
	RequireAuthy2FA   bool   `json:"requireAuthy2FA"`
	Val2FaRequestCode string `json:"val2FaRequestCode"`
}

// AccountInfo holds your current account information like balances, trade count
// and volume
type AccountInfo struct {
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

// Order is a generalised order type
type Order struct {
	ServerOrderID int        `json:"ServerOrderId"`
	AccountID     int        `json:"AccountId"`
	Price         float64    `json:"Price"`
	QtyTotal      float64    `json:"QtyTotal"`
	QtyRemaining  float64    `json:"QtyRemaining"`
	ReceiveTime   types.Time `json:"ReceiveTime"`
	Side          int64      `json:"Side"`
	State         int        `json:"orderState"`
	OrderType     int        `json:"orderType"`
}

// OpenOrders holds the full range of orders by instrument
type OpenOrders struct {
	Instrument string  `json:"ins"`
	OpenOrders []Order `json:"openOrders"`
}

// OrderInfo holds all open orders across the entire range of all instruments
type OrderInfo struct {
	OpenOrders   []OpenOrders `json:"openOrdersInfo"`
	IsAccepted   bool         `json:"isAccepted"`
	DateTimeUTC  types.Time   `json:"dateTimeUtc"`
	RejectReason string       `json:"rejectReason"`
}

// DepositAddresses holds information about the generated deposit address for
// a specific currency
type DepositAddresses struct {
	Name           string `json:"name"`
	DepositAddress string `json:"depositAddress"`
}

// WebsocketTicker holds current up to date ticker information
type WebsocketTicker struct {
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

// orderSideMap holds order type info based on Alphapoint data
var orderSideMap = map[int64]order.Side{
	1: order.Buy,
	2: order.Sell,
}

// orderTypeMap holds order type info based on Alphapoint data
var orderTypeMap = map[int]order.Type{
	1: order.Market,
	2: order.Limit,
	3: order.Stop,
	6: order.TrailingStop,
}
