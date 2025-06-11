package bithumb

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Ticker holds ticker data
type Ticker struct {
	OpeningPrice              float64 `json:"opening_price,string"`
	ClosingPrice              float64 `json:"closing_price,string"`
	MinPrice                  float64 `json:"min_price,string"`
	MaxPrice                  float64 `json:"max_price,string"`
	UnitsTraded               float64 `json:"units_traded,string"`
	AccumulatedTradeValue     float64 `json:"acc_trade_value,string"`
	PreviousClosingPrice      float64 `json:"prev_closing_price,string"`
	UnitsTraded24Hr           float64 `json:"units_traded_24H,string"`
	AccumulatedTradeValue24hr float64 `json:"acc_trade_value_24H,string"`
	Fluctuate24Hr             float64 `json:"fluctate_24H,string"`
	FluctuateRate24hr         float64 `json:"fluctate_rate_24H,string"`
	Date                      int64   `json:"date,string"`
}

// TickerResponse holds the standard ticker response
type TickerResponse struct {
	Status  string `json:"status"`
	Data    Ticker `json:"data"`
	Message string `json:"message"`
}

// TickersResponse holds the standard ticker response
type TickersResponse struct {
	Status  string                     `json:"status"`
	Data    map[string]json.RawMessage `json:"data"`
	Message string                     `json:"message"`
}

// Orderbook holds full range of order book information
type Orderbook struct {
	Status string `json:"status"`
	Data   struct {
		Timestamp       types.Time `json:"timestamp"`
		OrderCurrency   string     `json:"order_currency"`
		PaymentCurrency string     `json:"payment_currency"`
		Bids            []struct {
			Quantity float64 `json:"quantity,string"`
			Price    float64 `json:"price,string"`
		} `json:"bids"`
		Asks []struct {
			Quantity float64 `json:"quantity,string"`
			Price    float64 `json:"price,string"`
		} `json:"asks"`
	} `json:"data"`
	Message string `json:"message"`
}

// TransactionHistory holds history of completed transaction data
type TransactionHistory struct {
	Status string `json:"status"`
	Data   []struct {
		ContNumber      int64          `json:"cont_no,string"`
		TransactionDate types.DateTime `json:"transaction_date"`
		Type            string         `json:"type"`
		UnitsTraded     float64        `json:"units_traded,string"`
		Price           float64        `json:"price,string"`
		Total           float64        `json:"total,string"`
	} `json:"data"`
	Message string `json:"message"`
}

// Account holds account details
type Account struct {
	Status string `json:"status"`
	Data   struct {
		Created   int64   `json:"created,string"`
		AccountID string  `json:"account_id"`
		TradeFee  float64 `json:"trade_fee,string"`
		Balance   float64 `json:"balance,string"`
	} `json:"data"`
	Message string `json:"message"`
}

// Balance holds balance details
type Balance struct {
	Status  string                  `json:"status"`
	Data    map[string]types.Number `json:"data"`
	Message string                  `json:"message"`
}

// WalletAddressRes contains wallet address information
type WalletAddressRes struct {
	Status string `json:"status"`
	Data   struct {
		WalletAddress string `json:"wallet_address"`
		Tag           string // custom field we populate
		Currency      string `json:"currency"`
	} `json:"data"`
	Message string `json:"message"`
}

// LastTransactionTicker holds customer last transaction information
type LastTransactionTicker struct {
	Status string `json:"status"`
	Data   struct {
		OpeningPrice float64 `json:"opening_price,string"`
		ClosingPrice float64 `json:"closing_price,string"`
		MinPrice     float64 `json:"min_price,string"`
		MaxPrice     float64 `json:"max_price,string"`
		AveragePrice float64 `json:"average_price,string"`
		UnitsTraded  float64 `json:"units_traded,string"`
		Volume1Day   float64 `json:"volume_1day,string"`
		Volume7Day   float64 `json:"volume_7day,string"`
		BuyPrice     int64   `json:"buy_price,string"`
		SellPrice    int64   `json:"sell_price,string"`
		Date         int64   `json:"date,string"`
	} `json:"data"`
	Message string `json:"message"`
}

// Orders contains information about your current orders
type Orders struct {
	Status  string      `json:"status"`
	Data    []OrderData `json:"data"`
	Message string      `json:"message"`
}

// OrderData contains all individual order details
type OrderData struct {
	OrderID         string     `json:"order_id"`
	OrderCurrency   string     `json:"order_currency"`
	OrderDate       types.Time `json:"order_date"`
	PaymentCurrency string     `json:"payment_currency"`
	Type            string     `json:"type"`
	Status          string     `json:"status"`
	Units           float64    `json:"units,string"`
	UnitsRemaining  float64    `json:"units_remaining,string"`
	Price           float64    `json:"price,string"`
	Fee             float64    `json:"fee,string"`
	Total           float64    `json:"total,string"`
}

// UserTransactions holds users full transaction list
type UserTransactions struct {
	Status string `json:"status"`
	Data   []struct {
		Search          int64         `json:"search,string"`
		TransferDate    types.Time    `json:"transfer_date"`
		OrderCurrency   currency.Code `json:"order_currency"`
		PaymentCurrency currency.Code `json:"payment_currency"`
		Units           float64       `json:"units,string"`
		Price           float64       `json:"price,string"`
		Amount          float64       `json:"amount,string"`
		FeeCurrency     currency.Code `json:"fee_currency"`
		Fee             float64       `json:"fee,string"`
		OrderBalance    float64       `json:"order_balance,string"`
		PaymentBalance  float64       `json:"payment_balance,string"`
	} `json:"data"`
	Message string `json:"message"`
}

// OrderPlace contains order information
type OrderPlace struct {
	Status string `json:"status"`
	Data   []struct {
		ContID string  `json:"cont_id"`
		Units  float64 `json:"units,string"`
		Price  float64 `json:"price,string"`
		Total  float64 `json:"total,string"`
		Fee    float64 `json:"fee,string"`
	} `json:"data"`
	Message string `json:"message"`
}

// OrderDetails contains specific order information
type OrderDetails struct {
	Status string `json:"status"`
	Data   []struct {
		TransactionDate int64   `json:"transaction_date,string"`
		Type            string  `json:"type"`
		OrderCurrency   string  `json:"order_currency"`
		PaymentCurrency string  `json:"payment_currency"`
		UnitsTraded     float64 `json:"units_traded,string"`
		Price           float64 `json:"price,string"`
		Total           float64 `json:"total,string"`
	} `json:"data"`
	Message string `json:"message"`
}

// ActionStatus holds the return status
type ActionStatus struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// KRWDeposit resp type for a KRW deposit
type KRWDeposit struct {
	Status   string `json:"status"`
	Account  string `json:"account"`
	Bank     string `json:"bank"`
	BankUser string `json:"BankUser"`
	Message  string `json:"message"`
}

// MarketBuy holds market buy order information
type MarketBuy struct {
	Status  string `json:"status"`
	OrderID string `json:"order_id"`
	Data    []struct {
		ContID string  `json:"cont_id"`
		Units  float64 `json:"units,string"`
		Price  float64 `json:"price,string"`
		Total  float64 `json:"total,string"`
		Fee    float64 `json:"fee,string"`
	} `json:"data"`
	Message string `json:"message"`
}

// MarketSell holds market buy order information
type MarketSell struct {
	Status  string `json:"status"`
	OrderID string `json:"order_id"`
	Data    []struct {
		ContID string  `json:"cont_id"`
		Units  float64 `json:"units,string"`
		Price  float64 `json:"price,string"`
		Total  float64 `json:"total,string"`
		Fee    float64 `json:"fee,string"`
	} `json:"data"`
	Message string `json:"message"`
}

// WithdrawalFees the large list of predefined withdrawal fees
// Prone to change
var WithdrawalFees = map[currency.Code]float64{
	currency.KRW:   1000,
	currency.BTC:   0.001,
	currency.ETH:   0.01,
	currency.DASH:  0.01,
	currency.LTC:   0.01,
	currency.ETC:   0.01,
	currency.XRP:   1,
	currency.BCH:   0.001,
	currency.XMR:   0.05,
	currency.ZEC:   0.001,
	currency.QTUM:  0.05,
	currency.BTG:   0.001,
	currency.ICX:   1,
	currency.TRX:   5,
	currency.ELF:   5,
	currency.MITH:  5,
	currency.MCO:   0.5,
	currency.OMG:   0.4,
	currency.KNC:   3,
	currency.GNT:   12,
	currency.HSR:   0.2,
	currency.ZIL:   30,
	currency.ETHOS: 2,
	currency.PAY:   2.4,
	currency.WAX:   5,
	currency.POWR:  5,
	currency.LRC:   10,
	currency.GTO:   15,
	currency.STEEM: 0.01,
	currency.STRAT: 0.2, //nolint:misspell // Not a misspelling
	currency.PPT:   0.5,
	currency.CTXC:  4,
	currency.CMT:   20,
	currency.THETA: 24,
	currency.WTC:   0.7,
	currency.ITC:   5,
	currency.TRUE:  4,
	currency.ABT:   5,
	currency.RNT:   20,
	currency.PLY:   20,
	currency.WAVES: 0.01,
	currency.LINK:  10,
	currency.ENJ:   35,
	currency.PST:   30,
}

// FullBalance defines a return type with full balance data
type FullBalance struct {
	InUse     map[string]float64
	Misu      map[string]float64
	Total     map[string]float64
	Xcoin     map[string]float64
	Available map[string]float64
}

// OHLCVResponse holds returned kline data
type OHLCVResponse struct {
	Status string      `json:"status"`
	Data   []OHLCVItem `json:"data"`
}

// OHLCVItem holds a single kline item
type OHLCVItem struct {
	Timestamp types.Time
	Open      types.Number
	High      types.Number
	Low       types.Number
	Close     types.Number
	Volume    types.Number
}

// UnmarshalJSON unmarshals OHLCV
func (o *OHLCVItem) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[6]any{&o.Timestamp, &o.Open, &o.High, &o.Low, &o.Close, &o.Volume})
}

// Status defines the current exchange allowance to deposit or withdraw a
// currency
type Status struct {
	Status string `json:"status"`
	Data   struct {
		DepositStatus    int64 `json:"deposit_status"`
		WithdrawalStatus int64 `json:"withdrawal_status"`
	} `json:"data"`
	Message string `json:"message"`
}

// StatusAll defines the current exchange allowance to deposit or withdraw a
// currency
type StatusAll struct {
	Status string `json:"status"`
	Data   map[string]struct {
		DepositStatus    int64 `json:"deposit_status"`
		WithdrawalStatus int64 `json:"withdrawal_status"`
	} `json:"data"`
	Message string `json:"message"`
}
