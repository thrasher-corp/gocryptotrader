package poloniex

type PoloniexTicker struct {
	Last          float64 `json:"last,string"`
	LowestAsk     float64 `json:"lowestAsk,string"`
	HighestBid    float64 `json:"highestBid,string"`
	PercentChange float64 `json:"percentChange,string"`
	BaseVolume    float64 `json:"baseVolume,string"`
	QuoteVolume   float64 `json:"quoteVolume,string"`
	IsFrozen      int     `json:"isFrozen,string"`
	High24Hr      float64 `json:"high24hr,string"`
	Low24Hr       float64 `json:"low24hr,string"`
}

type PoloniexOrderbookResponseAll struct {
	Data map[string]PoloniexOrderbookResponse
}

type PoloniexOrderbookResponse struct {
	Asks     [][]interface{} `json:"asks"`
	Bids     [][]interface{} `json:"bids"`
	IsFrozen string          `json:"isFrozen"`
	Error    string          `json:"error"`
}

type PoloniexOrderbookItem struct {
	Price  float64
	Amount float64
}

type PoloniexOrderbookAll struct {
	Data map[string]PoloniexOrderbook
}

type PoloniexOrderbook struct {
	Asks []PoloniexOrderbookItem `json:"asks"`
	Bids []PoloniexOrderbookItem `json:"bids"`
}

type PoloniexTradeHistory struct {
	GlobalTradeID int64   `json:"globalTradeID"`
	TradeID       int64   `json:"tradeID"`
	Date          string  `json:"date"`
	Type          string  `json:"type"`
	Rate          float64 `json:"rate,string"`
	Amount        float64 `json:"amount,string"`
	Total         float64 `json:"total,string"`
}

type PoloniexChartData struct {
	Date            int     `json:"date"`
	High            float64 `json:"high"`
	Low             float64 `json:"low"`
	Open            float64 `json:"open"`
	Close           float64 `json:"close"`
	Volume          float64 `json:"volume"`
	QuoteVolume     float64 `json:"quoteVolume"`
	WeightedAverage float64 `json:"weightedAverage"`
	Error           string  `json:"error"`
}

type PoloniexCurrencies struct {
	Name               string      `json:"name"`
	MaxDailyWithdrawal string      `json:"maxDailyWithdrawal"`
	TxFee              float64     `json:"txFee,string"`
	MinConfirmations   int         `json:"minConf"`
	DepositAddresses   interface{} `json:"depositAddress"`
	Disabled           int         `json:"disabled"`
	Delisted           int         `json:"delisted"`
	Frozen             int         `json:"frozen"`
}

type PoloniexLoanOrder struct {
	Rate     float64 `json:"rate,string"`
	Amount   float64 `json:"amount,string"`
	RangeMin int     `json:"rangeMin"`
	RangeMax int     `json:"rangeMax"`
}

type PoloniexLoanOrders struct {
	Offers  []PoloniexLoanOrder `json:"offers"`
	Demands []PoloniexLoanOrder `json:"demands"`
}

type PoloniexBalance struct {
	Currency map[string]float64
}

type PoloniexCompleteBalance struct {
	Available float64
	OnOrders  float64
	BTCValue  float64
}

type PoloniexDepositAddresses struct {
	Addresses map[string]string
}

type PoloniexDepositsWithdrawals struct {
	Deposits []struct {
		Currency      string  `json:"currency"`
		Address       string  `json:"address"`
		Amount        float64 `json:"amount,string"`
		Confirmations int     `json:"confirmations"`
		TransactionID string  `json:"txid"`
		Timestamp     int64   `json:"timestamp"`
		Status        string  `json:"status"`
	} `json:"deposits"`
	Withdrawals []struct {
		WithdrawalNumber int64   `json:"withdrawalNumber"`
		Currency         string  `json:"currency"`
		Address          string  `json:"address"`
		Amount           float64 `json:"amount,string"`
		Confirmations    int     `json:"confirmations"`
		TransactionID    string  `json:"txid"`
		Timestamp        int64   `json:"timestamp"`
		Status           string  `json:"status"`
		IPAddress        string  `json:"ipAddress"`
	} `json:"withdrawals"`
}

type PoloniexOrder struct {
	OrderNumber int64   `json:"orderNumber,string"`
	Type        string  `json:"type"`
	Rate        float64 `json:"rate,string"`
	Amount      float64 `json:"amount,string"`
	Total       float64 `json:"total,string"`
	Date        string  `json:"date"`
	Margin      float64 `json:"margin"`
}

type PoloniexOpenOrdersResponseAll struct {
	Data map[string][]PoloniexOrder
}

type PoloniexOpenOrdersResponse struct {
	Data []PoloniexOrder
}

type PoloniexAuthentictedTradeHistory struct {
	GlobalTradeID int64   `json:"globalTradeID"`
	TradeID       int64   `json:"tradeID,string"`
	Date          string  `json:"date"`
	Rate          float64 `json:"rate,string"`
	Amount        float64 `json:"amount,string"`
	Total         float64 `json:"total,string"`
	Fee           float64 `json:"fee,string"`
	OrderNumber   int64   `json:"orderNumber,string"`
	Type          string  `json:"type"`
	Category      string  `json:"category"`
}

type PoloniexAuthenticatedTradeHistoryAll struct {
	Data map[string][]PoloniexAuthentictedTradeHistory
}

type PoloniexAuthenticatedTradeHistoryResponse struct {
	Data []PoloniexAuthentictedTradeHistory
}

type PoloniexResultingTrades struct {
	Amount  float64 `json:"amount,string"`
	Date    string  `json:"date"`
	Rate    float64 `json:"rate,string"`
	Total   float64 `json:"total,string"`
	TradeID int64   `json:"tradeID,string"`
	Type    string  `json:"type"`
}

type PoloniexOrderResponse struct {
	OrderNumber int64                     `json:"orderNumber,string"`
	Trades      []PoloniexResultingTrades `json:"resultingTrades"`
}

type PoloniexGenericResponse struct {
	Success int    `json:"success"`
	Error   string `json:"error"`
}

type PoloniexMoveOrderResponse struct {
	Success     int                                  `json:"success"`
	Error       string                               `json:"error"`
	OrderNumber int64                                `json:"orderNumber,string"`
	Trades      map[string][]PoloniexResultingTrades `json:"resultingTrades"`
}

type PoloniexWithdraw struct {
	Response string `json:"response"`
	Error    string `json:"error"`
}

type PoloniexFee struct {
	MakerFee        float64 `json:"makerFee,string"`
	TakerFee        float64 `json:"takerFee,string"`
	ThirtyDayVolume float64 `json:"thirtyDayVolume,string"`
	NextTier        float64 `json:"nextTier,string"`
}

type PoloniexMargin struct {
	TotalValue    float64 `json:"totalValue,string"`
	ProfitLoss    float64 `json:"pl,string"`
	LendingFees   float64 `json:"lendingFees,string"`
	NetValue      float64 `json:"netValue,string"`
	BorrowedValue float64 `json:"totalBorrowedValue,string"`
	CurrentMargin float64 `json:"currentMargin,string"`
}

type PoloniexMarginPosition struct {
	Amount            float64 `json:"amount,string"`
	Total             float64 `json:"total,string"`
	BasePrice         float64 `json:"basePrice,string"`
	LiquidiationPrice float64 `json:"liquidiationPrice"`
	ProfitLoss        float64 `json:"pl,string"`
	LendingFees       float64 `json:"lendingFees,string"`
	Type              string  `json:"type"`
}

type PoloniexLoanOffer struct {
	ID        int64   `json:"id"`
	Rate      float64 `json:"rate,string"`
	Amount    float64 `json:"amount,string"`
	Duration  int     `json:"duration"`
	AutoRenew bool    `json:"autoRenew,int"`
	Date      string  `json:"date"`
}

type PoloniexActiveLoans struct {
	Provided []PoloniexLoanOffer `json:"provided"`
	Used     []PoloniexLoanOffer `json:"used"`
}

type PoloniexLendingHistory struct {
	ID       int64   `json:"id"`
	Currency string  `json:"currency"`
	Rate     float64 `json:"rate,string"`
	Amount   float64 `json:"amount,string"`
	Duration float64 `json:"duration,string"`
	Interest float64 `json:"interest,string"`
	Fee      float64 `json:"fee,string"`
	Earned   float64 `json:"earned,string"`
	Open     string  `json:"open"`
	Close    string  `json:"close"`
}
