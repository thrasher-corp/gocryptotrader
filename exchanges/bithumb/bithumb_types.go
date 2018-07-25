package bithumb

import "github.com/kempeng/gocryptotrader/decimal"

// Ticker holds ticker data
type Ticker struct {
	OpeningPrice decimal.Decimal `json:"opening_price,string"`
	ClosingPrice decimal.Decimal `json:"closing_price,string"`
	MinPrice     decimal.Decimal `json:"min_price,string"`
	MaxPrice     decimal.Decimal `json:"max_price,string"`
	AveragePrice decimal.Decimal `json:"average_price,string"`
	UnitsTraded  decimal.Decimal `json:"units_traded,string"`
	Volume1Day   decimal.Decimal `json:"volume_1day,string"`
	Volume7Day   decimal.Decimal `json:"volume_7day,string"`
	BuyPrice     decimal.Decimal `json:"buy_price,string"`
	SellPrice    decimal.Decimal `json:"sell_price,string"`
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
			Quantity decimal.Decimal `json:"quantity,string"`
			Price    decimal.Decimal `json:"price,string"`
		} `json:"bids"`
		Asks []struct {
			Quantity decimal.Decimal `json:"quantity,string"`
			Price    decimal.Decimal `json:"price,string"`
		} `json:"asks"`
	} `json:"data"`
	Message string `json:"message"`
}

// TransactionHistory holds history of completed transaction data
type TransactionHistory struct {
	Status string `json:"status"`
	Data   []struct {
		ContNumber      int64           `json:"cont_no,string"`
		TransactionDate string          `json:"transaction_date"`
		Type            string          `json:"type"`
		UnitsTraded     decimal.Decimal `json:"units_traded,string"`
		Price           decimal.Decimal `json:"price,string"`
		Total           decimal.Decimal `json:"total,string"`
	} `json:"data"`
	Message string `json:"message"`
}

// Account holds account details
type Account struct {
	Status string `json:"status"`
	Data   struct {
		Created   int64           `json:"created,string"`
		AccountID string          `json:"account_id"`
		TradeFee  decimal.Decimal `json:"trade_fee,string"`
		Balance   decimal.Decimal `json:"balance,string"`
	} `json:"data"`
	Message string `json:"message"`
}

// Balance holds balance details
type Balance struct {
	Status string `json:"status"`
	Data   struct {
		TotalBTC     decimal.Decimal `json:"total_btc,string"`
		TotalKRW     decimal.Decimal `json:"total_krw"`
		InUseBTC     decimal.Decimal `json:"in_use_btc,string"`
		InUseKRW     decimal.Decimal `json:"in_use_krw"`
		AvailableBTC decimal.Decimal `json:"available_btc,string"`
		AvailableKRW decimal.Decimal `json:"available_krw"`
		MisuKRW      decimal.Decimal `json:"misu_krw"`
		MisuBTC      decimal.Decimal `json:"misu_btc,string"`
		XcoinLast    decimal.Decimal `json:"xcoin_last,string"`
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
		OpeningPrice decimal.Decimal `json:"opening_price,string"`
		ClosingPrice decimal.Decimal `json:"closing_price,string"`
		MinPrice     decimal.Decimal `json:"min_price,string"`
		MaxPrice     decimal.Decimal `json:"max_price,string"`
		AveragePrice decimal.Decimal `json:"average_price,string"`
		UnitsTraded  decimal.Decimal `json:"units_traded,string"`
		Volume1Day   decimal.Decimal `json:"volume_1day,string"`
		Volume7Day   decimal.Decimal `json:"volume_7day,string"`
		BuyPrice     int64           `json:"buy_price,string"`
		SellPrice    int64           `json:"sell_price,string"`
		Date         int64           `json:"date,string"`
	} `json:"data"`
	Message string `json:"message"`
}

// Orders contains information about your current orders
type Orders struct {
	Status string `json:"status"`
	Data   []struct {
		OrderID         string          `json:"order_id"`
		OrderCurrency   string          `json:"order_currency"`
		OrderDate       int64           `json:"order_date"`
		PaymentCurrency string          `json:"payment_currency"`
		Type            string          `json:"type"`
		Status          string          `json:"status"`
		Units           decimal.Decimal `json:"units,string"`
		UnitsRemaining  decimal.Decimal `json:"units_remaining,string"`
		Price           decimal.Decimal `json:"price,string"`
		Fee             decimal.Decimal `json:"fee,string"`
		Total           decimal.Decimal `json:"total,string"`
		DateCompleted   int64           `json:"date_completed"`
	} `json:"data"`
	Message string `json:"message"`
}

// UserTransactions holds users full transaction list
type UserTransactions struct {
	Status string `json:"status"`
	Data   []struct {
		Search       string          `json:"search"`
		TransferDate int64           `json:"transfer_date"`
		Units        string          `json:"units"`
		Price        decimal.Decimal `json:"price,string"`
		BTC1KRW      decimal.Decimal `json:"btc1krw,string"`
		Fee          string          `json:"fee"`
		BTCRemain    decimal.Decimal `json:"btc_remain,string"`
		KRWRemain    decimal.Decimal `json:"krw_remain,string"`
	} `json:"data"`
	Message string `json:"message"`
}

// OrderPlace contains order information
type OrderPlace struct {
	Status string `json:"status"`
	Data   []struct {
		ContID string          `json:"cont_id"`
		Units  decimal.Decimal `json:"units,string"`
		Price  decimal.Decimal `json:"price,string"`
		Total  decimal.Decimal `json:"total,string"`
		Fee    decimal.Decimal `json:"fee,string"`
	} `json:"data"`
	Message string `json:"message"`
}

// OrderDetails contains specific order information
type OrderDetails struct {
	Status string `json:"status"`
	Data   []struct {
		TransactionDate int64           `json:"transaction_date,string"`
		Type            string          `json:"type"`
		OrderCurrency   string          `json:"order_currency"`
		PaymentCurrency string          `json:"payment_currency"`
		UnitsTraded     decimal.Decimal `json:"units_traded,string"`
		Price           decimal.Decimal `json:"price,string"`
		Total           decimal.Decimal `json:"total,string"`
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
		ContID string          `json:"cont_id"`
		Units  decimal.Decimal `json:"units,string"`
		Price  decimal.Decimal `json:"price,string"`
		Total  decimal.Decimal `json:"total,string"`
		Fee    decimal.Decimal `json:"fee,string"`
	} `json:"data"`
	Message string `json:"message"`
}

// MarketSell holds market buy order information
type MarketSell struct {
	Status  string `json:"status"`
	OrderID string `json:"order_id"`
	Data    []struct {
		ContID string          `json:"cont_id"`
		Units  decimal.Decimal `json:"units,string"`
		Price  decimal.Decimal `json:"price,string"`
		Total  decimal.Decimal `json:"total,string"`
		Fee    decimal.Decimal `json:"fee,string"`
	} `json:"data"`
	Message string `json:"message"`
}
