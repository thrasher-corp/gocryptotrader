package itbit

import "time"

// GeneralReturn is a generalized return type to capture any errors
type GeneralReturn struct {
	Code        int    `json:"code"`
	Description string `json:"description"`
	RequestID   string `json:"requestId"`
}

// Ticker holds returned ticker information
type Ticker struct {
	Pair          string    `json:"pair"`
	Bid           float64   `json:"bid,string"`
	BidAmt        float64   `json:"bidAmt,string"`
	Ask           float64   `json:"ask,string"`
	AskAmt        float64   `json:"askAmt,string"`
	LastPrice     float64   `json:"lastPrice,string"`
	LastAmt       float64   `json:"lastAmt,string"`
	Volume24h     float64   `json:"volume24h,string"`
	VolumeToday   float64   `json:"volumeToday,string"`
	High24h       float64   `json:"high24h,string"`
	Low24h        float64   `json:"low24h,string"`
	HighToday     float64   `json:"highToday,string"`
	LowToday      float64   `json:"lowToday,string"`
	OpenToday     float64   `json:"openToday,string"`
	VwapToday     float64   `json:"vwapToday,string"`
	Vwap24h       float64   `json:"vwap24h,string"`
	ServertimeUTC time.Time `json:"serverTimeUTC"`
}

// OrderbookResponse contains multi-arrayed strings of bid and ask side
// information
type OrderbookResponse struct {
	Bids [][]string `json:"bids"`
	Asks [][]string `json:"asks"`
}

// Trades holds recent trades with associated information
type Trades struct {
	RecentTrades []struct {
		Timestamp   time.Time `json:"timestamp"`
		MatchNumber string    `json:"matchNumber"`
		Price       float64   `json:"price,string"`
		Amount      float64   `json:"amount,string"`
	} `json:"recentTrades"`
}

// Wallet contains specific wallet information
type Wallet struct {
	ID          string    `json:"id"`
	UserID      string    `json:"userId"`
	Name        string    `json:"name"`
	Balances    []Balance `json:"balances"`
	Description string    `json:"description"`
}

// Balance is a sub type holding balance information
type Balance struct {
	Currency         string  `json:"currency"`
	AvailableBalance float64 `json:"availableBalance,string"`
	TotalBalance     float64 `json:"totalBalance,string"`
	Description      string  `json:"description"`
}

// Records embodies records of trade history information
type Records struct {
	TotalNumberOfRecords int            `json:"totalNumberOfRecords,string"`
	CurrentPageNumber    int            `json:"currentPageNumber,string"`
	LatestExecutedID     int64          `json:"latestExecutionId,string"`
	RecordsPerPage       int            `json:"recordsPerPage,string"`
	TradingHistory       []TradeHistory `json:"tradingHistory"`
	Description          string         `json:"description"`
}

// TradeHistory stores historic trade values
type TradeHistory struct {
	OrderID            string  `json:"orderId"`
	Timestamp          string  `json:"timestamp"`
	Instrument         string  `json:"instrument"`
	Direction          string  `json:"direction"`
	CurrencyOne        string  `json:"currency1"`
	CurrencyOneAmount  float64 `json:"currency1Amount,string"`
	CurrencyTwo        string  `json:"currency2"`
	CurrencyTwoAmount  float64 `json:"currency2Amount"`
	Rate               float64 `json:"rate,string"`
	CommissionPaid     float64 `json:"commissionPaid,string"`
	CommissionCurrency string  `json:"commissionCurrency"`
	RebatesApplied     float64 `json:"rebatesApplied,string"`
	RebateCurrency     string  `json:"rebateCurrency"`
}

// FundingRecords embodies records of fund history information
type FundingRecords struct {
	TotalNumberOfRecords int           `json:"totalNumberOfRecords,string"`
	CurrentPageNumber    int           `json:"currentPageNumber,string"`
	LatestExecutedID     int64         `json:"latestExecutionId,string"`
	RecordsPerPage       int           `json:"recordsPerPage,string"`
	FundingHistory       []FundHistory `json:"fundingHistory"`
	Description          string        `json:"description"`
}

// FundHistory stores historic funding transactions
type FundHistory struct {
	BankName                    string  `json:"bankName"`
	WithdrawalID                int64   `json:"withdrawalId"`
	HoldingPeriodCompletionDate string  `json:"holdingPeriodCompletionDate"`
	DestinationAddress          string  `json:"destinationAddress"`
	TxnHash                     string  `json:"txnHash"`
	Time                        string  `json:"time"`
	Currency                    string  `json:"currency"`
	TransactionType             string  `json:"transactionType"`
	Amount                      float64 `json:"amount,string"`
	WalletName                  string  `json:"walletName"`
	Status                      string  `json:"status"`
}

// Order holds order information
type Order struct {
	ID                         string      `json:"id"`
	WalletID                   string      `json:"walletId"`
	Side                       string      `json:"side"`
	Instrument                 string      `json:"instrument"`
	Type                       string      `json:"type"`
	Currency                   string      `json:"currency"`
	Amount                     float64     `json:"amount,string"`
	Price                      float64     `json:"price,string"`
	AmountFilled               float64     `json:"amountFilled,string"`
	VolumeWeightedAveragePrice float64     `json:"volumeWeightedAveragePrice,string"`
	CreatedTime                string      `json:"createdTime"`
	Status                     string      `json:"Status"`
	Metadata                   interface{} `json:"metadata"`
	ClientOrderIdentifier      string      `json:"clientOrderIdentifier"`
	Description                string      `json:"description"`
}

// CryptoCurrencyDeposit holds information about a new wallet
type CryptoCurrencyDeposit struct {
	ID             int         `json:"id"`
	WalletID       string      `json:"walletID"`
	DepositAddress string      `json:"depositAddress"`
	Metadata       interface{} `json:"metadata"`
	Description    string      `json:"description"`
}

// WalletTransfer holds wallet transfer information
type WalletTransfer struct {
	SourceWalletID      string  `json:"sourceWalletId"`
	DestinationWalletID string  `json:"destinationWalletId"`
	Amount              float64 `json:"amount,string"`
	CurrencyCode        string  `json:"currencyCode"`
	Description         string  `json:"description"`
}
