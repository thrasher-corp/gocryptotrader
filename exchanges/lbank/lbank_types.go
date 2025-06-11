package lbank

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Ticker stores the ticker price data for a currency pair
type Ticker struct {
	Change   float64 `json:"change"`
	High     float64 `json:"high"`
	Latest   float64 `json:"latest"`
	Low      float64 `json:"low"`
	Turnover float64 `json:"turnover"`
	Volume   float64 `json:"vol"`
}

// TickerResponse stores the ticker price data and timestamp for a currency pair
type TickerResponse struct {
	Symbol    currency.Pair `json:"symbol"`
	Timestamp types.Time    `json:"timestamp"`
	Ticker    Ticker        `json:"ticker"`
}

// MarketDepthResponse stores arrays for asks, bids and a timestamp for a currency pair
type MarketDepthResponse struct {
	ErrCapture
	Data struct {
		Asks      [][2]types.Number `json:"asks"`
		Bids      [][2]types.Number `json:"bids"`
		Timestamp types.Time        `json:"timestamp"`
	} `json:"data"`
}

// TradeResponse stores date_ms, amount, price, type, tid for a currency pair
type TradeResponse struct {
	DateMS types.Time `json:"date_ms"`
	Amount float64    `json:"amount"`
	Price  float64    `json:"price"`
	Type   string     `json:"type"`
	TID    string     `json:"tid"`
}

// KlineResponse stores kline info for given currency exchange
type KlineResponse struct {
	TimeStamp     time.Time `json:"timestamp"`
	OpenPrice     float64   `json:"openprice"`
	HighestPrice  float64   `json:"highestprice"`
	LowestPrice   float64   `json:"lowestprice"`
	ClosePrice    float64   `json:"closeprice"`
	TradingVolume float64   `json:"tradingvolume"`
}

// InfoResponse stores info
type InfoResponse struct {
	Freeze map[string]types.Number `json:"freeze"`
	Asset  map[string]types.Number `json:"asset"`
	Free   map[string]types.Number `json:"Free"`
}

// InfoFinalResponse stores info
type InfoFinalResponse struct {
	ErrCapture
	Info InfoResponse `json:"info"`
}

// CreateOrderResponse stores the result of the Order and
type CreateOrderResponse struct {
	ErrCapture
	OrderID string `json:"order_id"`
}

// RemoveOrderResponse stores the result when an order is cancelled
type RemoveOrderResponse struct {
	ErrCapture
	Err     string `json:"error"`
	OrderID string `json:"order_id"`
	Success string `json:"success"`
}

// OrderResponse stores the data related to the given OrderIDs
type OrderResponse struct {
	Symbol     string     `json:"symbol"`
	Amount     float64    `json:"amount"`
	CreateTime types.Time `json:"created_time"`
	Price      float64    `json:"price"`
	AvgPrice   float64    `json:"avg_price"`
	Type       string     `json:"type"`
	OrderID    string     `json:"order_id"`
	DealAmount float64    `json:"deal_amount"`
	Status     int64      `json:"status"`
}

// QueryOrderResponse stores the data from queries
type QueryOrderResponse struct {
	ErrCapture
	Orders json.RawMessage `json:"orders"`
}

// QueryOrderFinalResponse stores data from queries
type QueryOrderFinalResponse struct {
	ErrCapture
	Orders []OrderResponse
}

// OrderHistory stores data for past orders
type OrderHistory struct {
	Result      bool            `json:"result,string"`
	Total       string          `json:"total"`
	PageLength  uint8           `json:"page_length"`
	Orders      json.RawMessage `json:"orders"`
	CurrentPage uint8           `json:"current_page"`
	ErrorCode   int64           `json:"error_code"`
}

// OrderHistoryResponse stores past orders
type OrderHistoryResponse struct {
	ErrCapture
	PageLength  uint8           `json:"page_length"`
	Orders      json.RawMessage `json:"orders"`
	CurrentPage uint8           `json:"current_page"`
}

// OrderHistoryFinalResponse stores past orders
type OrderHistoryFinalResponse struct {
	ErrCapture
	PageLength  uint8
	Orders      []OrderResponse
	CurrentPage uint8
}

// PairInfoResponse stores information about trading pairs
type PairInfoResponse struct {
	MinimumQuantity  string `json:"minTranQua"`
	PriceAccuracy    string `json:"priceAccuracy"`
	QuantityAccuracy string `json:"quantityAccuracy"`
	Symbol           string `json:"symbol"`
}

// TransactionTemp stores details about transactions
type TransactionTemp struct {
	TxUUID       string     `json:"txUuid"`
	OrderUUID    string     `json:"orderUuid"`
	TradeType    string     `json:"tradeType"`
	DealTime     types.Time `json:"dealTime"`
	DealPrice    float64    `json:"dealPrice"`
	DealQuantity float64    `json:"dealQuantity"`
	DealVolPrice float64    `json:"dealVolumePrice"`
	TradeFee     float64    `json:"tradeFee"`
	TradeFeeRate float64    `json:"tradeFeeRate"`
}

// TransactionHistoryResp stores details about past transactions
type TransactionHistoryResp struct {
	ErrCapture
	Transaction []TransactionTemp `json:"transaction"`
}

// OpenOrderResponse stores information about the opening orders
type OpenOrderResponse struct {
	ErrCapture
	PageLength uint8           `json:"page_length"`
	PageNumber uint8           `json:"page_number"`
	Total      string          `json:"total"`
	Orders     json.RawMessage `json:"orders"`
}

// OpenOrderFinalResponse stores the unmarshalled value of OpenOrderResponse
type OpenOrderFinalResponse struct {
	ErrCapture
	PageLength uint8
	PageNumber uint8
	Total      string
	Orders     []OrderResponse
}

// ExchangeRateResponse stores information about USD-RMB rate
type ExchangeRateResponse struct {
	USD2CNY string `json:"USD2CNY"`
}

// WithdrawConfigResponse stores info about withdrawal configurations
type WithdrawConfigResponse struct {
	AmountScale         int64         `json:"amountScale,string"`
	Chain               string        `json:"chain"`
	AssetCode           currency.Code `json:"assetCode"`
	Minimum             float64       `json:"min,string"`
	TransferAmountScale int64         `json:"transferAmtScale,string"`
	CanWithdraw         bool          `json:"canWithDraw"`
	Fee                 float64       `json:"fee"`
	MinimumTransfer     float64       `json:"minTransfer,string"`
	Type                int64         `json:"type,string"`
}

// WithdrawResponse stores info about the withdrawal
type WithdrawResponse struct {
	ErrCapture
	WithdrawID string  `json:"withdrawId"`
	Fee        float64 `json:"fee"`
}

// RevokeWithdrawResponse stores info about the revoked withdrawal
type RevokeWithdrawResponse struct {
	ErrCapture
	WithdrawID string `json:"string"`
}

// ListDataResponse contains some of withdrawal data
type ListDataResponse struct {
	ErrCapture
	Amount    float64    `json:"amount"`
	AssetCode string     `json:"assetCode"`
	Address   string     `json:"address"`
	Fee       float64    `json:"fee"`
	ID        int64      `json:"id"`
	Time      types.Time `json:"time"`
	TXHash    string     `json:"txhash"`
	Status    string     `json:"status"`
}

// WithdrawalResponse stores data for withdrawals
type WithdrawalResponse struct {
	ErrCapture
	TotalPages int64              `json:"totalPages"`
	PageSize   int64              `json:"pageSize"`
	PageNo     int64              `json:"pageNo"`
	List       []ListDataResponse `json:"list"`
}

// ErrCapture helps with error info
type ErrCapture struct {
	Error  int64 `json:"error_code"`
	Result bool  `json:"result,string"`
}

// GetAllOpenIDResp stores orderIds and currency pairs for open orders
type GetAllOpenIDResp struct {
	CurrencyPair string
	OrderID      string
}

// TimestampResponse holds timestamp data
type TimestampResponse struct {
	Timestamp types.Time `json:"data"`
}

var errorCodes = map[int64]string{
	10000: "Internal error",
	10001: "The required parameters can not be empty",
	10002: "Validation Failed",
	10003: "Invalid parameter",
	10004: "Request too frequent",
	10005: "Secret key does not exist",
	10006: "User does not exist",
	10007: "Invalid signature",
	10008: "Invalid Trading Pair",
	10009: "Price and/or Amount are required for limit order",
	10010: "Price and/or Amount must be more than 0",
	10013: "The amount is too small",
	10014: "Insufficient amount of money in account",
	10015: "Invalid order type",
	10016: "Insufficient account balance",
	10017: "Server Error",
	10018: "Page size should be between 1 and 50",
	10019: "Cancel NO more than 3 orders in one request",
	10020: "Volume < 0.001",
	10021: "Price < 0.01",
	10022: "Access denied",
	10023: "Market Order is not supported yet.",
	10024: "User cannot trade on this pair",
	10025: "Order has been filled",
	10026: "Order has been cancelld",
	10027: "Order is cancelling",
	10028: "Wrong query time",
	10029: "'from' is not in the query time",
	10030: "'from' does not match the transaction type of inqury",
	10100: "Has no privilege to withdraw",
	10101: "Invalid fee rate to withdraw",
	10102: "Too little to withdraw",
	10103: "Exceed daily limitation of withdraw",
	10104: "Cancel was rejected",
	10105: "Request has been cancelled",
}
