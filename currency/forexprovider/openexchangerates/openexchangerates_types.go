package openexchangerates

// Latest holds latest rate data
type Latest struct {
	Disclaimer  string             `json:"disclaimer"`
	License     string             `json:"license"`
	Timestamp   int64              `json:"timestamp"`
	Base        string             `json:"base"`
	Rates       map[string]float64 `json:"rates"`
	Error       bool               `json:"error"`
	Status      int                `json:"status"`
	Message     string             `json:"message"`
	Description string             `json:"description"`
}

// Historical holds historic rate data
type Historical struct {
	Disclaimer  string             `json:"disclaimer"`
	License     string             `json:"license"`
	Timestamp   int64              `json:"timestamp"`
	Base        string             `json:"base"`
	Rates       map[string]float64 `json:"rates"`
	Error       bool               `json:"error"`
	Status      int                `json:"status"`
	Message     string             `json:"message"`
	Description string             `json:"description"`
}

// TimeSeries holds historic rate data
type TimeSeries struct {
	Disclaimer  string                 `json:"disclaimer"`
	License     string                 `json:"license"`
	StartDate   string                 `json:"start_date"`
	EndDate     string                 `json:"end_date"`
	Base        string                 `json:"base"`
	Rates       map[string]interface{} `json:"rates"`
	Error       bool                   `json:"error"`
	Status      int                    `json:"status"`
	Message     string                 `json:"message"`
	Description string                 `json:"description"`
}

// Convert holds historic rate data
type Convert struct {
	Disclaimer string `json:"disclaimer"`
	License    string `json:"license"`
	Request    struct {
		Query  string  `json:"query"`
		Amount float64 `json:"amount"`
		From   string  `json:"from"`
		To     string  `json:"to"`
	} `json:"request"`
	Meta struct {
		Timestamp int64   `json:"timestamp"`
		Rate      float64 `json:"rate"`
	}
	Response    float64 `json:"response"`
	Error       bool    `json:"error"`
	Status      int     `json:"status"`
	Message     string  `json:"message"`
	Description string  `json:"description"`
}

// OHLC holds open high low close values
type OHLC struct {
	Disclaimer  string                 `json:"disclaimer"`
	License     string                 `json:"license"`
	StartDate   string                 `json:"start_date"`
	EndDate     string                 `json:"end_date"`
	Base        string                 `json:"base"`
	Rates       map[string]interface{} `json:"rates"`
	Error       bool                   `json:"error"`
	Status      int                    `json:"status"`
	Message     string                 `json:"message"`
	Description string                 `json:"description"`
}

// Usage holds usage statistical data
type Usage struct {
	Status int `json:"status"`
	Data   struct {
		AppID  string `json:"app_id"`
		Status string `json:"status"`
		Plan   struct {
			Name            string `json:"name"`
			Quota           string `json:"quota"`
			UpdateFrequency string `json:"update_frequency"`
			Features        struct {
				Base         bool `json:"base"`
				Symbols      bool `json:"symbols"`
				Experimental bool `json:"experimental"`
				Timeseries   bool `json:"time-series"`
				Convert      bool `json:"convert"`
			} `json:"features"`
		} `json:"plaab"`
	} `json:"data"`
	Usages struct {
		Requests          int64 `json:"requests"`
		RequestQuota      int   `json:"requests_quota"`
		RequestsRemaining int   `json:"requests_remaining"`
		DaysElapsed       int   `json:"days_elapsed"`
		DaysRemaining     int   `json:"days_remaining"`
		DailyAverage      int   `json:"daily_average"`
	}
	Error       bool   `json:"error"`
	Message     string `json:"message"`
	Description string `json:"description"`
}
