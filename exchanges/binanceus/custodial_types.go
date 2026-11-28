package binanceus

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// CustodialAccountBalanceResponse holds the balances of the Binance.US exchange wallet and of the custodial sub-account
type CustodialAccountBalanceResponse struct {
	ExchangeWalletBalance   []CustodialBalance `json:"exchangeWalletBalance"`
	CustodialAccountBalance []CustodialBalance `json:"custodialAcctBalance"`
}

// CustodialBalance holds an asset's balance in the exchange wallet or in the custodial sub-account
type CustodialBalance struct {
	Asset           currency.Code `json:"asset"`
	Free            types.Number  `json:"free"`
	Locked          types.Number  `json:"locked"`
	InSettlement    types.Number  `json:"inSettlement"`
	LastUpdatedTime types.Time    `json:"lastUpdatedTime"`
}

// CustodialSupportedAssetListResponse holds the assets that can be transferred from the custodial partner and the
// assets that can be settled to it
type CustodialSupportedAssetListResponse struct {
	TransferEligible   []CustodialSupportedAsset `json:"transferEligible"`
	SettlementEligible []CustodialSupportedAsset `json:"settlementEligible"`
}

// CustodialSupportedAsset holds an asset supported by the custodial solution, with its precision and networks
type CustodialSupportedAsset struct {
	Asset     currency.Code `json:"asset"`
	Precision uint64        `json:"precision"`
	Network   []string      `json:"network"`
}

// CustodialAssetTransferRequest holds the parameters of a transfer to the custodial sub-account, from the exchange
// wallet or from the custodial partner
type CustodialAssetTransferRequest struct {
	Rail          string // Custodial partner in upper case, such as ANCHORAGE
	Asset         currency.Code
	Amount        float64
	ClientOrderID string // A unique reference for the transfer; generated when empty
}

// TransferFromExchangeWalletResponse holds a transfer from the exchange wallet to the custodial sub-account
type TransferFromExchangeWalletResponse struct {
	Asset         currency.Code `json:"asset"`
	Amount        float64       `json:"amount"`
	ClientOrderID string        `json:"clientOrderId"`
	TransferID    string        `json:"transferId"`
	Status        string        `json:"status"`
	CreateTime    types.Time    `json:"createTime"`
}

// TransferFromCustodianResponse holds a transfer from the custodial partner to the custodial sub-account
type TransferFromCustodianResponse struct {
	Asset              currency.Code `json:"asset"`
	Amount             float64       `json:"amount"`
	ClientOrderID      string        `json:"clientOrderId"`
	TransferID         string        `json:"transferId"`
	CustodyAccountID   string        `json:"custodyAccountId"`
	CustodyAccountName string        `json:"custodyAccountName"`
	Status             string        `json:"status"`
	CreateTime         types.Time    `json:"createTime"`
}

// UndoCustodialTransferResponse holds an undone transfer from the custodial partner
type UndoCustodialTransferResponse struct {
	TransferID string        `json:"transferId"`
	Asset      currency.Code `json:"asset"`
	Amount     float64       `json:"amount"`
}

// ExchangeWalletTransferRequest holds the parameters of an exchange wallet transfer query
type ExchangeWalletTransferRequest struct {
	Rail          string // Custodial partner in upper case, such as ANCHORAGE
	TransferID    string
	ClientOrderID string
	Asset         currency.Code
	StartTime     time.Time // 90 days ago when zero
	EndTime       time.Time // Now when zero
	Page          uint64    // 1 when zero
	Limit         uint64    // 20 when zero, at most 100
}

// ExchangeWalletTransferResponse holds a page of transfers from the exchange wallet and the number of transfers the
// query matched
type ExchangeWalletTransferResponse struct {
	Data  []ExchangeWalletTransfer `json:"data"`
	Total uint64                   `json:"total"`
}

// ExchangeWalletTransfer holds a transfer from the exchange wallet to the custodial sub-account
type ExchangeWalletTransfer struct {
	TransferID    string        `json:"transferId"`
	ClientOrderID string        `json:"clientOrderId"`
	Asset         currency.Code `json:"asset"`
	Amount        float64       `json:"amount"`
	Status        string        `json:"status"`
	CreateTime    types.Time    `json:"createTime"`
	UpdateTime    types.Time    `json:"updateTime"`
}

// CustodianTransferRequest holds the parameters of a custodian transfer query
type CustodianTransferRequest struct {
	Rail                 string // Custodial partner in upper case, such as ANCHORAGE
	TransferID           string
	ClientOrderID        string
	ExpressTradeTransfer bool
	Asset                currency.Code
	StartTime            time.Time // 90 days ago when zero
	EndTime              time.Time // Now when zero
	Page                 uint64    // 1 when zero
	Limit                uint64    // 20 when zero, at most 100
}

// CustodianTransferResponse holds a page of transfers from the custodial partner and the number of transfers the query
// matched
type CustodianTransferResponse struct {
	Data  []CustodianTransfer `json:"data"`
	Total uint64              `json:"total"`
}

// CustodianTransfer holds a transfer from the custodial partner to the custodial sub-account: a custodian transfer, an
// ExpressTrade transfer or an undone transfer
type CustodianTransfer struct {
	TransferID    string        `json:"transferId"`
	ClientOrderID string        `json:"clientOrderId"`
	Asset         currency.Code `json:"asset"`
	Amount        float64       `json:"amount"`
	Status        string        `json:"status"`
	ExpressTrade  bool          `json:"expressTrade"`
	CreateTime    types.Time    `json:"createTime"`
	UpdateTime    types.Time    `json:"updateTime"`
}

// CustodialNewOrderRequest holds the parameters of a custodial order. Which quantity, price and stop parameters are
// mandatory depends on Type
type CustodialNewOrderRequest struct {
	Rail               string // Custodial partner in upper case, such as ANCHORAGE
	Symbol             currency.Pair
	Side               Side
	Type               OrderType
	TimeInForce        TimeInForce
	Quantity           float64
	QuoteOrderQuantity float64 // Quote asset amount a MARKET order spends or receives, instead of Quantity
	Price              float64
	StopPrice          float64
	IcebergQuantity    float64
	// Asset is the asset an ExpressTrade order sells
	Asset currency.Code
	// AllowExpressTrade transfers the order's full amount from the custodial partner when the custodial sub-account's
	// balance is smaller than the order
	AllowExpressTrade bool
}

// CustodialNewOrderResponse holds a new custodial order
type CustodialNewOrderResponse struct {
	Symbol                  string       `json:"symbol"`
	OrderID                 uint64       `json:"orderId"`
	OrderListID             int64        `json:"orderListId"` // -1 unless the order is part of an order list
	ClientOrderID           string       `json:"clientOrderId"`
	Price                   types.Number `json:"price"`
	OriginalQuantity        types.Number `json:"origQty"`
	ExecutedQuantity        types.Number `json:"executedQty"`
	CumulativeQuoteQuantity types.Number `json:"cummulativeQuoteQty"`
	Status                  string       `json:"status"`
	TimeInForce             TimeInForce  `json:"timeInForce"`
	Type                    OrderType    `json:"type"`
	Side                    Side         `json:"side"`
	ExpressTradeFlag        bool         `json:"expressTradeFlag"`
}

// CustodialNewOneCancelsTheOtherOrderRequest holds the parameters of a custodial one-cancels-the-other (OCO) order list:
// a limit order at Price and a stop loss order triggered at StopPrice, which is a stop loss limit order at
// StopLimitPrice when that is set. A sell's Price must be above the last price and its StopPrice below; a buy's the
// reverse
type CustodialNewOneCancelsTheOtherOrderRequest struct {
	Rail                 string // Custodial partner in upper case, such as ANCHORAGE
	Symbol               currency.Pair
	Side                 Side
	Quantity             float64 // The quantity of each order
	LimitClientOrderID   string  // Identifies the limit order
	Price                float64
	LimitIcebergQuantity float64
	StopClientOrderID    string // Identifies the stop order
	StopPrice            float64
	StopLimitPrice       float64
	StopIcebergQuantity  float64
	// StopLimitTimeInForce is required with StopLimitPrice
	StopLimitTimeInForce TimeInForce
	// Asset is the asset an ExpressTrade order list sells
	Asset currency.Code
	// AllowExpressTrade transfers the order list's full amount from the custodial partner when the custodial
	// sub-account's balance is smaller than the order list
	AllowExpressTrade bool
}

// CustodialOneCancelsTheOtherOrderResponse holds a custodial order list as placed or cancelled; a placement alone sends
// ExpressTradeFlag
type CustodialOneCancelsTheOtherOrderResponse struct {
	OrderListID       uint64                    `json:"orderListId"`
	ContingencyType   string                    `json:"contingencyType"`
	ListStatusType    string                    `json:"listStatusType"`
	ListOrderStatus   string                    `json:"listOrderStatus"`
	ListClientOrderID string                    `json:"listClientOrderId"`
	TransactionTime   types.Time                `json:"transactionTime"`
	Symbol            string                    `json:"symbol"`
	ExpressTradeFlag  bool                      `json:"expressTradeFlag"`
	Orders            []CustodialOrderListOrder `json:"orders"`
	OrderReports      []CustodialOrderReport    `json:"orderReports"`
}

// CustodialOrderListOrder identifies an order of a custodial order list
type CustodialOrderListOrder struct {
	Symbol        string `json:"symbol"`
	OrderID       uint64 `json:"orderId"`
	ClientOrderID string `json:"clientOrderId"`
}

// CustodialOrderReport holds an order report of a custodial order list placement, which alone sends TransactTime, or
// of a cancellation, which alone sends OriginalClientOrderID
type CustodialOrderReport struct {
	Symbol                  string       `json:"symbol"`
	OriginalClientOrderID   string       `json:"origClientOrderId"`
	OrderID                 uint64       `json:"orderId"`
	OrderListID             int64        `json:"orderListId"` // -1 unless the order is part of an order list
	ClientOrderID           string       `json:"clientOrderId"`
	TransactTime            types.Time   `json:"transactTime"`
	Price                   types.Number `json:"price"`
	OriginalQuantity        types.Number `json:"origQty"`
	ExecutedQuantity        types.Number `json:"executedQty"`
	CumulativeQuoteQuantity types.Number `json:"cummulativeQuoteQty"`
	Status                  string       `json:"status"`
	TimeInForce             TimeInForce  `json:"timeInForce"`
	Type                    OrderType    `json:"type"`
	Side                    Side         `json:"side"`
	StopPrice               types.Number `json:"stopPrice"`
}

// CustodialOrderRequest holds the parameters of a custodial order query
type CustodialOrderRequest struct {
	Rail    string // Custodial partner in upper case, such as ANCHORAGE
	Symbol  currency.Pair
	OrderID uint64
}

// CustodialOrderResponse holds a custodial order as queried
type CustodialOrderResponse struct {
	Symbol                     string       `json:"symbol"`
	OrderID                    uint64       `json:"orderId"`
	OrderListID                int64        `json:"orderListId"` // -1 unless the order is part of an order list
	ClientOrderID              string       `json:"clientOrderId"`
	Price                      types.Number `json:"price"`
	OriginalQuantity           types.Number `json:"origQty"`
	ExecutedQuantity           types.Number `json:"executedQty"`
	CumulativeQuoteQuantity    types.Number `json:"cummulativeQuoteQty"`
	Status                     string       `json:"status"`
	TimeInForce                TimeInForce  `json:"timeInForce"`
	Type                       OrderType    `json:"type"`
	Side                       Side         `json:"side"`
	StopPrice                  types.Number `json:"stopPrice"`
	IcebergQuantity            types.Number `json:"icebergQty"`
	Time                       types.Time   `json:"time"`
	UpdateTime                 types.Time   `json:"updateTime"`
	IsWorking                  bool         `json:"isWorking"`
	OriginalQuoteOrderQuantity types.Number `json:"origQuoteOrderQty"`
	ExpressTradeFlag           bool         `json:"expressTradeFlag"`
}

// CustodialOrderHistoryRequest holds the parameters of a custodial order history query
type CustodialOrderHistoryRequest struct {
	Rail      string        // Custodial partner in upper case, such as ANCHORAGE
	Symbol    currency.Pair // Every symbol's orders when empty
	StartTime time.Time
	EndTime   time.Time
	FromID    uint64 // Orders from this order ID on; 1 when zero
	Limit     uint64 // 200 when zero
}

// CustodialTradeHistoryRequest holds the parameters of a custodial trade history query
type CustodialTradeHistoryRequest struct {
	Rail      string        // Custodial partner in upper case, such as ANCHORAGE
	Symbol    currency.Pair // Every symbol's trades when empty
	OrderID   uint64
	StartTime time.Time
	EndTime   time.Time
	FromID    uint64 // Trades from this trade ID on; the most recent trades when zero
	Limit     uint64 // 200 when zero
}

// CustodialTradeResponse holds a trade of the custodial sub-account
type CustodialTradeResponse struct {
	Symbol          string        `json:"symbol"`
	Price           types.Number  `json:"price"`
	Quantity        types.Number  `json:"qty"`
	QuoteQuantity   types.Number  `json:"quoteQty"`
	Time            types.Time    `json:"time"`
	IsBuyer         bool          `json:"isBuyer"`
	IsMaker         bool          `json:"isMaker"`
	IsBestMatch     bool          `json:"isBestMatch"`
	OrderID         uint64        `json:"orderId"`
	OrderListID     int64         `json:"orderListId"` // -1 unless the order is part of an order list
	Commission      types.Number  `json:"commission"`
	CommissionAsset currency.Code `json:"commissionAsset"`
}

// CancelCustodialOrderRequest holds the parameters of a custodial order cancellation; OrderID or OriginalClientOrderID
// is required
type CancelCustodialOrderRequest struct {
	Rail                  string // Custodial partner in upper case, such as ANCHORAGE
	Symbol                currency.Pair
	OrderID               uint64
	OriginalClientOrderID string
	NewClientOrderID      string // Identifies the cancellation; generated when empty
}

// CancelCustodialOrderResponse holds a cancelled custodial order
type CancelCustodialOrderResponse struct {
	Symbol                  string       `json:"symbol"`
	OriginalClientOrderID   string       `json:"origClientOrderId"`
	OrderID                 uint64       `json:"orderId"`
	OrderListID             int64        `json:"orderListId"` // -1 unless the order is part of an order list
	ClientOrderID           string       `json:"clientOrderId"`
	Price                   types.Number `json:"price"`
	OriginalQuantity        types.Number `json:"origQty"`
	ExecutedQuantity        types.Number `json:"executedQty"`
	CumulativeQuoteQuantity types.Number `json:"cummulativeQuoteQty"`
	Status                  string       `json:"status"`
	TimeInForce             TimeInForce  `json:"timeInForce"`
	Type                    OrderType    `json:"type"`
	Side                    Side         `json:"side"`
}

// CancelCustodialOneCancelsTheOtherOrderRequest holds the parameters of a custodial order list cancellation
type CancelCustodialOneCancelsTheOtherOrderRequest struct {
	Rail   string // Custodial partner in upper case, such as ANCHORAGE
	Symbol currency.Pair
	// OrderListID is required, and a pointer, as 0 is a valid order list ID
	OrderListID       *uint64
	ListClientOrderID string
	NewClientOrderID  string // Identifies the cancellation; generated when empty
}

// CustodialSettlementSettingsResponse holds the settlement settings: whether settlement is active, its schedule and its
// next trigger time
type CustodialSettlementSettingsResponse struct {
	SettlementActive bool       `json:"settlementActive"`
	FrequencyInHours uint64     `json:"frequencyInHours"`
	NextTriggerTime  types.Time `json:"nextTriggerTime"`
}

// CustodialSettlementHistoryRequest holds the parameters of a settlement history query
type CustodialSettlementHistoryRequest struct {
	Rail      string // Custodial partner in upper case, such as ANCHORAGE
	StartTime time.Time
	EndTime   time.Time
	Limit     uint64 // 5 when zero, at most 100
	Page      uint64 // 1 when zero
}

// CustodialSettlementHistoryResponse holds a page of settlements and the number of settlements the query matched
type CustodialSettlementHistoryResponse struct {
	Records []CustodialSettlementRecord `json:"records"`
	Total   uint64                      `json:"total"`
}

// CustodialSettlementRecord holds a settlement and the assets it settled
type CustodialSettlementRecord struct {
	Status           string                     `json:"status"`
	TriggerTime      types.Time                 `json:"triggerTime"`
	SettlementID     string                     `json:"settlementId"`
	SettlementAssets []CustodialSettlementAsset `json:"settlementAssets"`
}

// CustodialSettlementAsset holds an amount of an asset a settlement sent to the custodial partner
type CustodialSettlementAsset struct {
	Asset    currency.Code `json:"asset"`
	SettleTo string        `json:"settleTo"`
	Network  string        `json:"network"`
	Amount   types.Number  `json:"amount"`
}
