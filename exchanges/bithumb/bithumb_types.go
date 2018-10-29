package bithumb

import "github.com/thrasher-/gocryptotrader/currency/symbol"

// Ticker holds ticker data
type Ticker struct {
	OpeningPrice float64 `json:"opening_price,string"`
	ClosingPrice float64 `json:"closing_price,string"`
	MinPrice     float64 `json:"min_price,string"`
	MaxPrice     float64 `json:"max_price,string"`
	AveragePrice float64 `json:"average_price,string"`
	UnitsTraded  float64 `json:"units_traded,string"`
	Volume1Day   float64 `json:"volume_1day,string"`
	Volume7Day   float64 `json:"volume_7day,string"`
	BuyPrice     float64 `json:"buy_price,string"`
	SellPrice    float64 `json:"sell_price,string"`
	ActionStatus
	//	Date         int64   `json:"date,string"`
}

// TickerResponse holds the standard ticker response
type TickerResponse struct {
	Status  string `json:"status"`
	Data    Ticker `json:"data"`
	Message string `json:"message"`
}

// TickersResponse holds the standard ticker response
type TickersResponse struct {
	Status  string            `json:"status"`
	Data    map[string]Ticker `json:"data"`
	Message string            `json:"message"`
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
	Status string `json:"status"`
	Data   struct {
		TotalBTC     float64 `json:"total_btc,string"`
		TotalKRW     float64 `json:"total_krw"`
		InUseBTC     float64 `json:"in_use_btc,string"`
		InUseKRW     float64 `json:"in_use_krw"`
		AvailableBTC float64 `json:"available_btc,string"`
		AvailableKRW float64 `json:"available_krw"`
		MisuKRW      float64 `json:"misu_krw"`
		MisuBTC      float64 `json:"misu_btc,string"`
		XcoinLast    float64 `json:"xcoin_last,string"`
	} `json:"data"`
	Message string `json:"message"`
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
	Status string `json:"status"`
	Data   []struct {
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
	} `json:"data"`
	Message string `json:"message"`
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

// WithdrawalFees the large list of predefined withdrawal fees
// Prone to change
var WithdrawalFees = map[string]float64{
	symbol.KRW:   1000,
	symbol.BTC:   0.001,
	symbol.ETH:   0.01,
	symbol.DASH:  0.01,
	symbol.LTC:   0.01,
	symbol.ETC:   0.01,
	symbol.XRP:   1,
	symbol.BCH:   0.001,
	symbol.XMR:   0.05,
	symbol.ZEC:   0.001,
	symbol.QTUM:  0.05,
	symbol.BTG:   0.001,
	symbol.ICX:   1,
	symbol.TRX:   5,
	symbol.ELF:   5,
	symbol.MITH:  5,
	symbol.MCO:   0.5,
	symbol.OMG:   0.4,
	symbol.KNC:   3,
	symbol.GNT:   12,
	symbol.HSR:   0.2,
	symbol.ZIL:   30,
	symbol.ETHOS: 2,
	symbol.PAY:   2.4,
	symbol.WAX:   5,
	symbol.POWR:  5,
	symbol.LRC:   10,
	symbol.GTO:   15,
	symbol.STEEM: 0.01,
	symbol.STRAT: 0.2,
	symbol.PPT:   0.5,
	symbol.CTXC:  4,
	symbol.CMT:   20,
	symbol.THETA: 24,
	symbol.WTC:   0.7,
	symbol.ITC:   5,
	symbol.TRUE:  4,
	symbol.ABT:   5,
	symbol.RNT:   20,
	symbol.PLY:   20,
	symbol.WAVES: 0.01,
	symbol.LINK:  10,
	symbol.ENJ:   35,
	symbol.PST:   30,
}
