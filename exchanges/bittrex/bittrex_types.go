package bittrex

import (
	"encoding/json"

	"github.com/shopspring/decimal"
)

// Response is the generalised response type for Bittrex
type Response struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Result  json.RawMessage `json:"result"`
}

// Market holds current market metadata
type Market struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Result  []struct {
		MarketCurrency     string          `json:"MarketCurrency"`
		BaseCurrency       string          `json:"BaseCurrency"`
		MarketCurrencyLong string          `json:"MarketCurrencyLong"`
		BaseCurrencyLong   string          `json:"BaseCurrencyLong"`
		MinTradeSize       decimal.Decimal `json:"MinTradeSize"`
		MarketName         string          `json:"MarketName"`
		IsActive           bool            `json:"IsActive"`
		Created            string          `json:"Created"`
	} `json:"result"`
}

// Currency holds supported currency metadata
type Currency struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Result  []struct {
		Currency        string          `json:"Currency"`
		CurrencyLong    string          `json:"CurrencyLong"`
		MinConfirmation int             `json:"MinConfirmation"`
		TxFee           decimal.Decimal `json:"TxFee"`
		IsActive        bool            `json:"IsActive"`
		CoinType        string          `json:"CoinType"`
		BaseAddress     string          `json:"BaseAddress"`
	} `json:"result"`
}

// Ticker holds basic ticker information
type Ticker struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Result  struct {
		Bid  decimal.Decimal `json:"Bid"`
		Ask  decimal.Decimal `json:"Ask"`
		Last decimal.Decimal `json:"Last"`
	} `json:"result"`
}

// MarketSummary holds last 24 hour metadata of an active exchange
type MarketSummary struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Result  []struct {
		MarketName        string          `json:"MarketName"`
		High              decimal.Decimal `json:"High"`
		Low               decimal.Decimal `json:"Low"`
		Volume            decimal.Decimal `json:"Volume"`
		Last              decimal.Decimal `json:"Last"`
		BaseVolume        decimal.Decimal `json:"BaseVolume"`
		TimeStamp         string          `json:"TimeStamp"`
		Bid               decimal.Decimal `json:"Bid"`
		Ask               decimal.Decimal `json:"Ask"`
		OpenBuyOrders     int             `json:"OpenBuyOrders"`
		OpenSellOrders    int             `json:"OpenSellOrders"`
		PrevDay           decimal.Decimal `json:"PrevDay"`
		Created           string          `json:"Created"`
		DisplayMarketName string          `json:"DisplayMarketName"`
	} `json:"result"`
}

// OrderBooks holds an array of buy & sell orders held on the exchange
type OrderBooks struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Result  struct {
		Buy  []OrderBook `json:"buy"`
		Sell []OrderBook `json:"sell"`
	} `json:"result"`
}

// OrderBook holds a singular order on an exchange
type OrderBook struct {
	Quantity decimal.Decimal `json:"Quantity"`
	Rate     decimal.Decimal `json:"Rate"`
}

// MarketHistory holds an executed trade's data for a market ie "BTC-LTC"
type MarketHistory struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Result  []struct {
		ID        int             `json:"Id"`
		Timestamp string          `json:"TimeStamp"`
		Quantity  decimal.Decimal `json:"Quantity"`
		Price     decimal.Decimal `json:"Price"`
		Total     decimal.Decimal `json:"Total"`
		FillType  string          `json:"FillType"`
		OrderType string          `json:"OrderType"`
	} `json:"result"`
}

// Balance holds the balance from your account for a specified currency
type Balance struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Result  struct {
		Currency      string          `json:"Currency"`
		Balance       decimal.Decimal `json:"Balance"`
		Available     decimal.Decimal `json:"Available"`
		Pending       decimal.Decimal `json:"Pending"`
		CryptoAddress string          `json:"CryptoAddress"`
		Requested     bool            `json:"Requested"`
		UUID          string          `json:"Uuid"`
	} `json:"result"`
}

// Balances holds the balance from your account for a specified currency
type Balances struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Result  []struct {
		Currency      string          `json:"Currency"`
		Balance       decimal.Decimal `json:"Balance"`
		Available     decimal.Decimal `json:"Available"`
		Pending       decimal.Decimal `json:"Pending"`
		CryptoAddress string          `json:"CryptoAddress"`
		Requested     bool            `json:"Requested"`
		UUID          string          `json:"Uuid"`
	} `json:"result"`
}

// DepositAddress holds a generated address to send specific coins to the
// exchange
type DepositAddress struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Result  struct {
		Currency string `json:"Currency"`
		Address  string `json:"Address"`
	} `json:"result"`
}

// UUID contains the universal unique identifier for one or multiple
// transactions on the exchange
type UUID struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Result  []struct {
		ID string `json:"uuid"`
	} `json:"result"`
}

// Order holds the full order information associated with the UUID supplied
type Order struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Result  []struct {
		AccountID                  string          `json:"AccountId"`
		OrderUUID                  string          `json:"OrderUuid"`
		Exchange                   string          `json:"Exchange"`
		Type                       string          `json:"Type"`
		Quantity                   decimal.Decimal `json:"Quantity"`
		QuantityRemaining          decimal.Decimal `json:"QuantityRemaining"`
		Limit                      decimal.Decimal `json:"Limit"`
		Reserved                   decimal.Decimal `json:"Reserved"`
		ReserveRemaining           decimal.Decimal `json:"ReserveRemaining"`
		CommissionReserved         decimal.Decimal `json:"CommissionReserved"`
		CommissionReserveRemaining decimal.Decimal `json:"CommissionReserveRemaining"`
		CommissionPaid             decimal.Decimal `json:"CommissionPaid"`
		Price                      decimal.Decimal `json:"Price"`
		PricePerUnit               decimal.Decimal `json:"PricePerUnit"`
		Opened                     string          `json:"Opened"`
		Closed                     string          `json:"Closed"`
		IsOpen                     bool            `json:"IsOpen"`
		Sentinel                   string          `json:"Sentinel"`
		CancelInitiated            bool            `json:"CancelInitiated"`
		ImmediateOrCancel          bool            `json:"ImmediateOrCancel"`
		IsConditional              bool            `json:"IsConditional"`
		Condition                  string          `json:"Condition"`
		ConditionTarget            string          `json:"ConditionTarget"`
	} `json:"result"`
}

// WithdrawalHistory holds the Withdrawal history data
type WithdrawalHistory struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Result  []struct {
		PaymentUUID    string          `json:"PaymentUuid"`
		Currency       string          `json:"Currency"`
		Amount         decimal.Decimal `json:"Amount"`
		Address        string          `json:"Address"`
		Opened         string          `json:"Opened"`
		Authorized     bool            `json:"Authorized"`
		PendingPayment bool            `json:"PendingPayment"`
		TxCost         decimal.Decimal `json:"TxCost"`
		TxID           string          `json:"TxId"`
		Canceled       bool            `json:"Canceled"`
		InvalidAddress bool            `json:"InvalidAddress"`
	} `json:"result"`
}
