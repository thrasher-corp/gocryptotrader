package fixer

// Rates contains the data fields for the currencies you have requested.
type Rates struct {
	Success bool `json:"success"`
	Error   struct {
		Code int    `json:"code"`
		Type string `json:"type"`
		Info string `json:"info"`
	} `json:"error"`
	Historical bool               `json:"historical"`
	Timestamp  int64              `json:"timestamp"`
	Base       string             `json:"base"`
	Date       string             `json:"date"`
	Rates      map[string]float64 `json:"rates"`
}

// Conversion contains data for currency conversion
type Conversion struct {
	Success bool `json:"success"`
	Error   struct {
		Code int    `json:"code"`
		Type string `json:"type"`
		Info string `json:"info"`
	} `json:"error"`
	Query struct {
		From   string  `json:"from"`
		To     string  `json:"to"`
		Amount float64 `json:"amount"`
	} `json:"query"`
	Info struct {
		Timestamp int64   `json:"timestamp"`
		Rate      float64 `json:"rate"`
	} `json:"info"`
	Historical bool    `json:"historical"`
	Date       string  `json:"date"`
	Result     float64 `json:"result"`
}

// TimeSeries holds timeseries data
type TimeSeries struct {
	Success bool `json:"success"`
	Error   struct {
		Code int    `json:"code"`
		Type string `json:"type"`
		Info string `json:"info"`
	} `json:"error"`
	Timeseries bool                   `json:"timeseries"`
	StartDate  string                 `json:"start_date"`
	EndDate    string                 `json:"end_date"`
	Base       string                 `json:"base"`
	Rates      map[string]interface{} `json:"rates"`
}

// Fluctuation holds fluctuation data
type Fluctuation struct {
	Success bool `json:"success"`
	Error   struct {
		Code int    `json:"code"`
		Type string `json:"type"`
		Info string `json:"info"`
	} `json:"error"`
	Fluctuation bool            `json:"fluctuation"`
	StartDate   string          `json:"start_date"`
	EndDate     string          `json:"end_date"`
	Base        string          `json:"base"`
	Rates       map[string]Flux `json:"rates"`
}

// Flux is a sub type holding fluctation data
type Flux struct {
	StartRate float64 `json:"start_rate"`
	EndRate   float64 `json:"end_rate"`
	Change    float64 `json:"change"`
	ChangePCT float64 `json:"change_pct"`
}
