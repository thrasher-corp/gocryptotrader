package bithumb

import (
	"encoding/json"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fee"
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
		Timestamp       int64  `json:"timestamp,string"`
		OrderCurrency   string `json:"order_currency"`
		PaymentCurrency string `json:"payment_currency"`
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
		ContNumber      int64   `json:"cont_no,string"`
		TransactionDate string  `json:"transaction_date"`
		Type            string  `json:"type"`
		UnitsTraded     float64 `json:"units_traded,string"`
		Price           float64 `json:"price,string"`
		Total           float64 `json:"total,string"`
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
	Status  string                 `json:"status"`
	Data    map[string]interface{} `json:"data"`
	Message string                 `json:"message"`
}

// WalletAddressRes contains wallet address information
type WalletAddressRes struct {
	Status string `json:"status"`
	Data   struct {
		WalletAddress string `json:"wallet_address"`
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
	OrderID         string  `json:"order_id"`
	OrderCurrency   string  `json:"order_currency"`
	OrderDate       int64   `json:"order_date"`
	PaymentCurrency string  `json:"payment_currency"`
	Type            string  `json:"type"`
	Status          string  `json:"status"`
	Units           float64 `json:"units,string"`
	UnitsRemaining  float64 `json:"units_remaining,string"`
	Price           float64 `json:"price,string"`
	Fee             float64 `json:"fee,string"`
	Total           float64 `json:"total,string"`
	DateCompleted   int64   `json:"date_completed"`
}

// UserTransactions holds users full transaction list
type UserTransactions struct {
	Status string `json:"status"`
	Data   []struct {
		Search       string  `json:"search"`
		TransferDate int64   `json:"transfer_date"`
		Units        string  `json:"units"`
		Price        float64 `json:"price,string"`
		BTC1KRW      float64 `json:"btc1krw,string"`
		Fee          string  `json:"fee"`
		BTCRemain    float64 `json:"btc_remain,string"`
		KRWRemain    float64 `json:"krw_remain,string"`
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

// transferFees the large list of predefined fees. Prone to change.
var transferFees = map[asset.Item]map[currency.Code]fee.Transfer{
	asset.Spot: {
		currency.KRW:   {Withdrawal: 1000},
		currency.BTC:   {Withdrawal: 0.001, Deposit: 0}, // TODO: Add functionality to express below
		currency.ETH:   {Withdrawal: 0.01},
		currency.DASH:  {Withdrawal: 0.01},
		currency.LTC:   {Withdrawal: 0.01},
		currency.ETC:   {Withdrawal: 0.01},
		currency.XRP:   {Withdrawal: 1},
		currency.BCH:   {Withdrawal: 0.001},
		currency.XMR:   {Withdrawal: 0.05},
		currency.ZEC:   {Withdrawal: 0.001},
		currency.QTUM:  {Withdrawal: 0.05},
		currency.BTG:   {Withdrawal: 0.001},
		currency.ICX:   {Withdrawal: 1},
		currency.TRX:   {Withdrawal: 5},
		currency.ELF:   {Withdrawal: 5},
		currency.MITH:  {Withdrawal: 5},
		currency.MCO:   {Withdrawal: 0.5},
		currency.OMG:   {Withdrawal: 0.4},
		currency.KNC:   {Withdrawal: 3},
		currency.GNT:   {Withdrawal: 12},
		currency.HSR:   {Withdrawal: 0.2},
		currency.ZIL:   {Withdrawal: 30},
		currency.ETHOS: {Withdrawal: 2},
		currency.PAY:   {Withdrawal: 2.4},
		currency.WAX:   {Withdrawal: 5},
		currency.POWR:  {Withdrawal: 5},
		currency.LRC:   {Withdrawal: 10},
		currency.GTO:   {Withdrawal: 15},
		currency.STEEM: {Withdrawal: 0.01},
		currency.STRAT: {Withdrawal: 0.2},
		currency.PPT:   {Withdrawal: 0.5},
		currency.CTXC:  {Withdrawal: 4},
		currency.CMT:   {Withdrawal: 20},
		currency.THETA: {Withdrawal: 24},
		currency.WTC:   {Withdrawal: 0.7},
		currency.ITC:   {Withdrawal: 5},
		currency.TRUE:  {Withdrawal: 4},
		currency.ABT:   {Withdrawal: 5},
		currency.RNT:   {Withdrawal: 20},
		currency.PLY:   {Withdrawal: 20},
		currency.WAVES: {Withdrawal: 0.01},
		currency.LINK:  {Withdrawal: 10},
		currency.ENJ:   {Withdrawal: 35},
		currency.PST:   {Withdrawal: 30},
	},
}

// TODO: Add small deposit fee below to above
// // getDepositFee returns fee on a currency when depositing small amounts to bithumb
// func getDepositFee(c currency.Code, amount float64) float64 {
// 	var f float64

// 	switch c {
// 	case currency.BTC:
// 		if amount <= 0.005 {
// 			f = 0.001
// 		}
// 	case currency.LTC:
// 		if amount <= 0.3 {
// 			f = 0.01
// 		}
// 	case currency.DASH:
// 		if amount <= 0.04 {
// 			f = 0.01
// 		}
// 	case currency.BCH:
// 		if amount <= 0.03 {
// 			f = 0.001
// 		}
// 	case currency.ZEC:
// 		if amount <= 0.02 {
// 			f = 0.001
// 		}
// 	case currency.BTG:
// 		if amount <= 0.15 {
// 			f = 0.001
// 		}
// 	}

// 	return f
// }

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
	Status string           `json:"status"`
	Data   [][6]interface{} `json:"data"`
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
