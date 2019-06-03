package zb

import (
	"encoding/json"

	"github.com/thrasher-/gocryptotrader/currency"
)

// Subscription defines an initial subscription type to be sent
type Subscription struct {
	Event   string `json:"event"`
	Channel string `json:"channel"`
}

// Generic defines a generic fields associated with many return types
type Generic struct {
	Code    int64           `json:"code"`
	Success bool            `json:"success"`
	Channel string          `json:"channel"`
	Message interface{}     `json:"message"`
	No      string          `json:"no"`
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
	Timestamp int64         `json:"timestamp"`
	Asks      []interface{} `json:"asks"`
	Bids      []interface{} `json:"bids"`
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
	Sign      string `json:"sign,omitempty"`
}

// WsAddSubUserRequest data to add sub users
type WsAddSubUserRequest struct {
	WsAuthenticatedRequest
	Memo        string `json:"memo"`
	Password    string `json:"password"`
	SubUserName string `json:"subUserName"`
}

// WsCreateSubUserKeyRequest data to add sub user keys
type WsCreateSubUserKeyRequest struct {
	WsAuthenticatedRequest
	AssetPerm   bool   `json:"assetPerm,string"`
	EntrustPerm bool   `json:"entrustPerm,string"`
	KeyName     string `json:"keyName"`
	LeverPerm   bool   `json:"leverPerm,string"`
	MoneyPerm   bool   `json:"moneyPerm,string"`
	ToUserID    string `json:"toUserId"`
}

// WsDoTransferFundsRequest data to transfer funds
type WsDoTransferFundsRequest struct {
	WsAuthenticatedRequest
	Amount       float64       `json:"amount,string"`
	Currency     currency.Pair `json:"currency"`
	FromUserName string        `json:"fromUserName"`
	ToUserName   string        `json:"toUserName"`
}

// WsGetSubUserListResponse data response from GetSubUserList
type WsGetSubUserListResponse struct {
	Success bool                           `json:"success"`
	Code    int64                          `json:"code"`
	Channel string                         `json:"channel"`
	Message []WsGetSubUserListResponseData `json:"message"` /*[] `json:"message,string"`*/
	No      string                         `json:"no"`
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
	No      string      `json:"no"`
}

// WsSubmitOrderRequest creates an order via ws
type WsSubmitOrderRequest struct {
	WsAuthenticatedRequest
	Amount      float64 `json:"amount"`
	Price       float64 `json:"price"`
	TradeType   int64   `json:"tradeType"`
	AccountType int64   `json:"acccType"`
}

// WsSubmitOrderResponse data about submitted order
type WsSubmitOrderResponse struct {
	Message string `json:"message"`
	No      string `json:"no"`
	Data    struct {
		EntrustID int64 `json:"intrustID"`
	} `json:"data"`
	Code    int64  `json:"code"`
	Channel string `json:"channel"`
	Success bool   `json:"success"`
}

// WsCancelOrderRequest order cancel request
type WsCancelOrderRequest struct {
	WsAuthenticatedRequest
	ID int64 `json:"id"`
}

// WsCancelOrderResponse order cancel response
type WsCancelOrderResponse struct {
	Message string `json:"message"`
	No      string `json:"no"`
	Code    int64  `json:"code"`
	Channel string `json:"channel"`
	Success bool   `json:"success"`
}

// WsGetOrderRequest Get specific order details
type WsGetOrderRequest struct {
	WsAuthenticatedRequest
	ID int64 `json:"id"`
}

// WsGetOrderResponse contains order data
type WsGetOrderResponse struct {
	Message string                 `json:"message"`
	No      string                 `json:"no"`
	Code    int64                  `json:"code"`
	Channel string                 `json:"channel"`
	Success bool                   `json:"success"`
	Data    WsGetOrderResponseData `json:"data"`
}

// WsGetOrderResponseData Detailed order data
type WsGetOrderResponseData struct {
	Currency    string  `json:"currency"`
	Fees        int64   `json:"fees"`
	ID          string  `json:"id"`
	Price       int64   `json:"price"`
	Status      int64   `json:"status"`
	TotalAmount float64 `json:"total_amount"`
	TradeAmount int64   `json:"trade_amount"`
	TradePrice  int64   `json:"trade_price"`
	TradeDate   int64   `json:"trade_date"`
	TradeMoney  int64   `json:"trade_money"`
	Type        int64   `json:"type"`
}

// WsGetOrdersRequest get more orders, with no orderID filtering
type WsGetOrdersRequest struct {
	WsAuthenticatedRequest
	PageIndex int64 `json:"pageIndex"`
	TradeType int64 `json:"tradeType"`
}

// WsGetOrdersResponse contains orders data
type WsGetOrdersResponse struct {
	Message string                   `json:"message"`
	No      string                   `json:"no"`
	Code    int64                    `json:"code"`
	Channel string                   `json:"channel"`
	Success bool                     `json:"success"`
	Data    []WsGetOrderResponseData `json:"data"`
}

// WsGetOrdersIgnoreTradeTypeRequest request weirdly requires tradetype
type WsGetOrdersIgnoreTradeTypeRequest struct {
	WsAuthenticatedRequest
	PageIndex int64 `json:"pageIndex"`
	PageSize  int64 `json:"pageSize"`
	ID        int64 `json:"id"`
	TradeType int64 `json:"tradeType"`
}

// WsGetOrdersIgnoreTradeTypeResponse contains orders data
type WsGetOrdersIgnoreTradeTypeResponse struct {
	Message string                   `json:"message"`
	No      string                   `json:"no"`
	Code    int64                    `json:"code"`
	Channel string                   `json:"channel"`
	Success bool                     `json:"success"`
	Data    []WsGetOrderResponseData `json:"data"`
}

// WsGetAccountInfoResponse contains account data
type WsGetAccountInfoResponse struct {
	Message string `json:"message"`
	No      string `json:"no"`
	Data    struct {
		Coins []struct {
			Freez       string `json:"freez"`
			EnName      string `json:"enName"`
			UnitDecimal int    `json:"unitDecimal"`
			CnName      string `json:"cnName"`
			UnitTag     string `json:"unitTag"`
			Available   string `json:"available"`
			Key         string `json:"key"`
		} `json:"coins"`
		Base struct {
			Username             string `json:"username"`
			TradePasswordEnabled bool   `json:"trade_password_enabled"`
			AuthGoogleEnabled    bool   `json:"auth_google_enabled"`
			AuthMobileEnabled    bool   `json:"auth_mobile_enabled"`
		} `json:"base"`
	} `json:"data"`
	Code    int    `json:"code"`
	Channel string `json:"channel"`
	Success bool   `json:"success"`
}
