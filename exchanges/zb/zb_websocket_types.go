package zb

import (
	"encoding/json"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

// Subscription defines an initial subscription type to be sent
type Subscription struct {
	Event   string `json:"event"`
	Channel string `json:"channel"`
	No      int64  `json:"no,string,omitempty"`
}

// Generic defines a generic fields associated with many return types
type Generic struct {
	Code    int64           `json:"code"`
	Channel string          `json:"channel"`
	Message interface{}     `json:"message"`
	No      int64           `json:"no,string,omitempty"`
	Data    json.RawMessage `json:"data"`
}

// Markets defines market data
type Markets map[string]struct {
	AmountScale int64 `json:"amountScale"`
	PriceScale  int64 `json:"priceScale"`
}

// WsTicker defines websocket ticker data
type WsTicker struct {
	Date int64 `json:"date,string"`
	Data struct {
		Volume24Hr float64 `json:"vol,string"`
		High       float64 `json:"high,string"`
		Low        float64 `json:"low,string"`
		Last       float64 `json:"last,string"`
		Buy        float64 `json:"buy,string"`
		Sell       float64 `json:"sell,string"`
	} `json:"ticker"`
}

// WsDepth defines websocket orderbook data
type WsDepth struct {
	Timestamp int64           `json:"timestamp"`
	Asks      [][]interface{} `json:"asks"`
	Bids      [][]interface{} `json:"bids"`
}

// WsTrades defines websocket trade data
type WsTrades struct {
	Data []struct {
		Amount    float64 `json:"amount,string"`
		Price     float64 `json:"price,string"`
		TID       int64   `json:"tid"`
		Date      int64   `json:"date"`
		Type      string  `json:"type"`
		TradeType string  `json:"trade_type"`
	} `json:"data"`
}

// WsAuthenticatedRequest base request type
type WsAuthenticatedRequest struct {
	Accesskey string `json:"accesskey"`
	Channel   string `json:"channel"`
	Event     string `json:"event"`
	No        int64  `json:"no,string,omitempty"`
	Sign      string `json:"sign,omitempty"`
}

// WsAddSubUserRequest data to add sub users
type WsAddSubUserRequest struct {
	Accesskey   string `json:"accesskey"`
	Channel     string `json:"channel"`
	Event       string `json:"event"`
	Memo        string `json:"memo"`
	Password    string `json:"password"`
	SubUserName string `json:"subUserName"`
	No          int64  `json:"no,string,omitempty"`
	Sign        string `json:"sign,omitempty"`
}

// WsCreateSubUserKeyRequest data to add sub user keys
type WsCreateSubUserKeyRequest struct {
	Accesskey   string `json:"accesskey"`
	AssetPerm   bool   `json:"assetPerm,string"`
	Channel     string `json:"channel"`
	EntrustPerm bool   `json:"entrustPerm,string"`
	Event       string `json:"event"`
	KeyName     string `json:"keyName"`
	LeverPerm   bool   `json:"leverPerm,string"`
	MoneyPerm   bool   `json:"moneyPerm,string"`
	No          int64  `json:"no,string,omitempty"`
	Sign        string `json:"sign,omitempty"`
	ToUserID    string `json:"toUserId"`
}

// WsDoTransferFundsRequest data to transfer funds
type WsDoTransferFundsRequest struct {
	Accesskey    string        `json:"accesskey"`
	Amount       float64       `json:"amount,string"`
	Channel      string        `json:"channel"`
	Currency     currency.Code `json:"currency"`
	Event        string        `json:"event"`
	FromUserName string        `json:"fromUserName"`
	No           int64         `json:"no,string"`
	Sign         string        `json:"sign,omitempty"`
	ToUserName   string        `json:"toUserName"`
}

// WsGetSubUserListResponse data response from GetSubUserList
type WsGetSubUserListResponse struct {
	Success bool                           `json:"success"`
	Code    int64                          `json:"code"`
	Channel string                         `json:"channel"`
	Message []WsGetSubUserListResponseData `json:"message"`
	No      int64                          `json:"no,string"`
}

// WsGetSubUserListResponseData user data
type WsGetSubUserListResponseData struct {
	IsOpenAPI bool   `json:"isOpenApi,omitempty"`
	Memo      string `json:"memo,omitempty"`
	UserName  string `json:"userName,omitempty"`
	UserID    int64  `json:"userId,omitempty"`
	IsFreez   bool   `json:"isFreez,omitempty"`
}

// WsRequestResponse generic response data
type WsRequestResponse struct {
	Success bool        `json:"success"`
	Code    int64       `json:"code"`
	Channel string      `json:"channel"`
	Message interface{} `json:"message"`
	No      int64       `json:"no,string"`
}

// WsSubmitOrderRequest creates an order via ws
type WsSubmitOrderRequest struct {
	Accesskey string  `json:"accesskey"`
	Amount    float64 `json:"amount,string"`
	Channel   string  `json:"channel"`
	Event     string  `json:"event"`
	No        int64   `json:"no,string,omitempty"`
	Price     float64 `json:"price,string"`
	Sign      string  `json:"sign,omitempty"`
	TradeType int64   `json:"tradeType,string"`
}

// WsSubmitOrderResponse data about submitted order
type WsSubmitOrderResponse struct {
	Message string `json:"message"`
	No      int64  `json:"no,string"`
	Data    struct {
		EntrustID int64 `json:"intrustID"`
	} `json:"data"`
	Code    int64  `json:"code"`
	Channel string `json:"channel"`
	Success bool   `json:"success"`
}

// WsCancelOrderRequest order cancel request
type WsCancelOrderRequest struct {
	Accesskey string `json:"accesskey"`
	Channel   string `json:"channel"`
	Event     string `json:"event"`
	ID        int64  `json:"id"`
	Sign      string `json:"sign,omitempty"`
	No        int64  `json:"no,string"`
}

// WsCancelOrderResponse order cancel response
type WsCancelOrderResponse struct {
	Message string `json:"message"`
	No      int64  `json:"no,string"`
	Code    int64  `json:"code"`
	Channel string `json:"channel"`
	Success bool   `json:"success"`
}

// WsGetOrderRequest Get specific order details
type WsGetOrderRequest struct {
	Accesskey string `json:"accesskey"`
	Channel   string `json:"channel"`
	Event     string `json:"event"`
	ID        int64  `json:"id"`
	Sign      string `json:"sign,omitempty"`
	No        int64  `json:"no,string"`
}

// WsGetOrderResponse contains order data
type WsGetOrderResponse struct {
	Message string  `json:"message"`
	No      int64   `json:"no,string"`
	Code    int64   `json:"code"`
	Channel string  `json:"channel"`
	Success bool    `json:"success"`
	Data    []Order `json:"data"`
}

// WsGetOrdersRequest get more orders, with no orderID filtering
type WsGetOrdersRequest struct {
	Accesskey string `json:"accesskey"`
	Channel   string `json:"channel"`
	Event     string `json:"event"`
	No        int64  `json:"no,string"`
	PageIndex int64  `json:"pageIndex"`
	TradeType int64  `json:"tradeType"`
	Sign      string `json:"sign,omitempty"`
}

// WsGetOrdersResponse contains orders data
type WsGetOrdersResponse struct {
	Message string  `json:"message"`
	No      int64   `json:"no,string"`
	Code    int64   `json:"code"`
	Channel string  `json:"channel"`
	Success bool    `json:"success"`
	Data    []Order `json:"data"`
}

// WsGetOrdersIgnoreTradeTypeRequest ws request
type WsGetOrdersIgnoreTradeTypeRequest struct {
	Accesskey string `json:"accesskey"`
	Channel   string `json:"channel"`
	Event     string `json:"event"`
	No        int64  `json:"no,string"`
	PageIndex int64  `json:"pageIndex"`
	PageSize  int64  `json:"pageSize"`
	Sign      string `json:"sign,omitempty"`
}

// WsGetOrdersIgnoreTradeTypeResponse contains orders data
type WsGetOrdersIgnoreTradeTypeResponse struct {
	Message string  `json:"message"`
	No      int64   `json:"no,string"`
	Code    int64   `json:"code"`
	Channel string  `json:"channel"`
	Success bool    `json:"success"`
	Data    []Order `json:"data"`
}

// WsGetAccountInfoResponse contains account data
type WsGetAccountInfoResponse struct {
	Message string `json:"message"`
	No      int64  `json:"no,string"`
	Data    struct {
		Coins []AccountsResponseCoin `json:"coins"`
		Base  AccountsBaseResponse   `json:"base"`
	} `json:"data"`
	Code    int64  `json:"code"`
	Channel string `json:"channel"`
	Success bool   `json:"success"`
}

var wsErrCodes = map[int64]string{
	1000: "Successful call",
	1001: "General error message",
	1002: "internal error",
	1003: "Verification failed",
	1004: "Financial security password lock",
	1005: "The fund security password is incorrect. Please confirm and re-enter.",
	1006: "Real-name certification is awaiting review or review",
	1007: "Channel is empty",
	1008: "Event is empty",
	1009: "This interface is being maintained",
	1011: "Not open yet",
	1012: "Insufficient permissions",
	1013: "Can not trade, if you have any questions, please contact online customer service",
	1014: "Cannot be sold during the pre-sale period",
	2002: "Insufficient balance in Bitcoin account",
	2003: "Insufficient balance of Litecoin account",
	2005: "Insufficient balance in Ethereum account",
	2006: "Insufficient balance in ETC currency account",
	2007: "Insufficient balance of BTS currency account",
	2008: "Insufficient balance in EOS currency account",
	2009: "Insufficient account balance",
	3001: "Pending order not found",
	3002: "Invalid amount",
	3003: "Invalid quantity",
	3004: "User does not exist",
	3005: "Invalid parameter",
	3006: "Invalid IP or inconsistent with the bound IP",
	3007: "Request time has expired",
	3008: "Transaction history not found",
	4001: "API interface is locked",
	4002: "Request too frequently",
}
