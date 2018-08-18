package currencylayer

import "github.com/thrasher-/gocryptotrader/decimal"

// LiveRates is a response type holding rates priced now.
type LiveRates struct {
	Success bool `json:"success"`
	Error   struct {
		Code int    `json:"code"`
		Info string `json:"info"`
	} `json:"error"`
	Terms     string                     `json:"terms"`
	Privacy   string                     `json:"privacy"`
	Timestamp int64                      `json:"timestamp"`
	Source    string                     `json:"source"`
	Quotes    map[string]decimal.Decimal `json:"quotes"`
}

// SupportedCurrencies holds supported currency information
type SupportedCurrencies struct {
	Success bool `json:"success"`
	Error   struct {
		Code int    `json:"code"`
		Info string `json:"info"`
	} `json:"error"`
	Terms      string            `json:"terms"`
	Privacy    string            `json:"privacy"`
	Currencies map[string]string `json:"currencies"`
}

// HistoricalRates is a response type holding rates priced from the past.
type HistoricalRates struct {
	Success bool `json:"success"`
	Error   struct {
		Code int    `json:"code"`
		Info string `json:"info"`
	} `json:"error"`
	Terms      string                     `json:"terms"`
	Privacy    string                     `json:"privacy"`
	Historical bool                       `json:"historical"`
	Date       string                     `json:"date"`
	Timestamp  int64                      `json:"timestamp"`
	Source     string                     `json:"source"`
	Quotes     map[string]decimal.Decimal `json:"quotes"`
}

// ConversionRate is a response type holding a converted rate.
type ConversionRate struct {
	Success bool `json:"success"`
	Error   struct {
		Code int    `json:"code"`
		Info string `json:"info"`
	} `json:"error"`
	Privacy string `json:"privacy"`
	Terms   string `json:"terms"`
	Query   struct {
		From   string          `json:"from"`
		To     string          `json:"to"`
		Amount decimal.Decimal `json:"amount"`
	} `json:"query"`
	Info struct {
		Timestamp int64           `json:"timestamp"`
		Quote     decimal.Decimal `json:"quote"`
	} `json:"info"`
	Historical bool            `json:"historical"`
	Date       string          `json:"date"`
	Result     decimal.Decimal `json:"result"`
}

// TimeFrame is a response type holding exchange rates for a time period
type TimeFrame struct {
	Success bool `json:"success"`
	Error   struct {
		Code int    `json:"code"`
		Info string `json:"info"`
	} `json:"error"`
	Terms     string                 `json:"terms"`
	Privacy   string                 `json:"privacy"`
	Timeframe bool                   `json:"timeframe"`
	StartDate string                 `json:"start_date"`
	EndDate   string                 `json:"end_date"`
	Source    string                 `json:"source"`
	Quotes    map[string]interface{} `json:"quotes"`
}

// ChangeRate is the response type that holds rate change data.
type ChangeRate struct {
	Success bool `json:"success"`
	Error   struct {
		Code int    `json:"code"`
		Info string `json:"info"`
	} `json:"error"`
	Terms     string             `json:"terms"`
	Privacy   string             `json:"privacy"`
	Change    bool               `json:"change"`
	StartDate string             `json:"start_date"`
	EndDate   string             `json:"end_date"`
	Source    string             `json:"source"`
	Quotes    map[string]Changes `json:"quotes"`
}

// Changes is a sub-type of ChangeRate that holds the actual changes of rates.
type Changes struct {
	StartRate decimal.Decimal `json:"start_rate"`
	EndRate   decimal.Decimal `json:"end_rate"`
	Change    decimal.Decimal `json:"change"`
	ChangePCT decimal.Decimal `json:"change_pct"`
}
