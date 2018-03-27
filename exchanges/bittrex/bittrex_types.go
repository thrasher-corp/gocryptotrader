package bittrex

import "encoding/json"

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
		MarketCurrency     string  `json:"MarketCurrency"`
		BaseCurrency       string  `json:"BaseCurrency"`
		MarketCurrencyLong string  `json:"MarketCurrencyLong"`
		BaseCurrencyLong   string  `json:"BaseCurrencyLong"`
		MinTradeSize       float64 `json:"MinTradeSize"`
		MarketName         string  `json:"MarketName"`
		IsActive           bool    `json:"IsActive"`
		Created            string  `json:"Created"`
	} `json:"result"`
}

// Currency holds supported currency metadata
type Currency struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Result  []struct {
		Currency        string  `json:"Currency"`
		CurrencyLong    string  `json:"CurrencyLong"`
		MinConfirmation int     `json:"MinConfirmation"`
		TxFee           float64 `json:"TxFee"`
		IsActive        bool    `json:"IsActive"`
		CoinType        string  `json:"CoinType"`
		BaseAddress     string  `json:"BaseAddress"`
	} `json:"result"`
}

// Ticker holds basic ticker information
type Ticker struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Result  struct {
		Bid  float64 `json:"Bid"`
		Ask  float64 `json:"Ask"`
		Last float64 `json:"Last"`
	} `json:"result"`
}

// MarketSummary holds last 24 hour metadata of an active exchange
type MarketSummary struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Result  []struct {
		MarketName        string  `json:"MarketName"`
		High              float64 `json:"High"`
		Low               float64 `json:"Low"`
		Volume            float64 `json:"Volume"`
		Last              float64 `json:"Last"`
		BaseVolume        float64 `json:"BaseVolume"`
		TimeStamp         string  `json:"TimeStamp"`
		Bid               float64 `json:"Bid"`
		Ask               float64 `json:"Ask"`
		OpenBuyOrders     int     `json:"OpenBuyOrders"`
		OpenSellOrders    int     `json:"OpenSellOrders"`
		PrevDay           float64 `json:"PrevDay"`
		Created           string  `json:"Created"`
		DisplayMarketName string  `json:"DisplayMarketName"`
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
	Quantity float64 `json:"Quantity"`
	Rate     float64 `json:"Rate"`
}

// MarketHistory holds an executed trade's data for a market ie "BTC-LTC"
type MarketHistory struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Result  []struct {
		ID        int     `json:"Id"`
		Timestamp string  `json:"TimeStamp"`
		Quantity  float64 `json:"Quantity"`
		Price     float64 `json:"Price"`
		Total     float64 `json:"Total"`
		FillType  string  `json:"FillType"`
		OrderType string  `json:"OrderType"`
	} `json:"result"`
}

// Balance holds the balance from your account for a specified currency
type Balance struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Result  struct {
		Currency      string  `json:"Currency"`
		Balance       float64 `json:"Balance"`
		Available     float64 `json:"Available"`
		Pending       float64 `json:"Pending"`
		CryptoAddress string  `json:"CryptoAddress"`
		Requested     bool    `json:"Requested"`
		UUID          string  `json:"Uuid"`
	} `json:"result"`
}

// Balances holds the balance from your account for a specified currency
type Balances struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Result  []struct {
		Currency      string  `json:"Currency"`
		Balance       float64 `json:"Balance"`
		Available     float64 `json:"Available"`
		Pending       float64 `json:"Pending"`
		CryptoAddress string  `json:"CryptoAddress"`
		Requested     bool    `json:"Requested"`
		UUID          string  `json:"Uuid"`
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
		AccountID                  string  `json:"AccountId"`
		OrderUUID                  string  `json:"OrderUuid"`
		Exchange                   string  `json:"Exchange"`
		Type                       string  `json:"Type"`
		Quantity                   float64 `json:"Quantity"`
		QuantityRemaining          float64 `json:"QuantityRemaining"`
		Limit                      float64 `json:"Limit"`
		Reserved                   float64 `json:"Reserved"`
		ReserveRemaining           float64 `json:"ReserveRemaining"`
		CommissionReserved         float64 `json:"CommissionReserved"`
		CommissionReserveRemaining float64 `json:"CommissionReserveRemaining"`
		CommissionPaid             float64 `json:"CommissionPaid"`
		Price                      float64 `json:"Price"`
		PricePerUnit               float64 `json:"PricePerUnit"`
		Opened                     string  `json:"Opened"`
		Closed                     string  `json:"Closed"`
		IsOpen                     bool    `json:"IsOpen"`
		Sentinel                   string  `json:"Sentinel"`
		CancelInitiated            bool    `json:"CancelInitiated"`
		ImmediateOrCancel          bool    `json:"ImmediateOrCancel"`
		IsConditional              bool    `json:"IsConditional"`
		Condition                  string  `json:"Condition"`
		ConditionTarget            string  `json:"ConditionTarget"`
	} `json:"result"`
}

// WithdrawalHistory holds the Withdrawal history data
type WithdrawalHistory struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Result  []struct {
		PaymentUUID    string  `json:"PaymentUuid"`
		Currency       string  `json:"Currency"`
		Amount         float64 `json:"Amount"`
		Address        string  `json:"Address"`
		Opened         string  `json:"Opened"`
		Authorized     bool    `json:"Authorized"`
		PendingPayment bool    `json:"PendingPayment"`
		TxCost         float64 `json:"TxCost"`
		TxID           string  `json:"TxId"`
		Canceled       bool    `json:"Canceled"`
		InvalidAddress bool    `json:"InvalidAddress"`
	} `json:"result"`
}
