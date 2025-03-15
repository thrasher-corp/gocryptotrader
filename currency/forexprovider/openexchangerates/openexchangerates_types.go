package openexchangerates

import (
	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/base"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// These consts contain endpoint information
const (
	APIDeveloperAccess = iota
	APIEnterpriseAccess
	APIUnlimitedAccess

	APIURL                = "https://openexchangerates.org/api/"
	APIEndpointLatest     = "latest.json"
	APIEndpointHistorical = "historical/%s.json"
	APIEndpointCurrencies = "currencies.json"
	APIEndpointTimeSeries = "time-series.json"
	APIEndpointConvert    = "convert/%s/%s/%s"
	APIEndpointOHLC       = "ohlc.json"
	APIEndpointUsage      = "usage.json"

	oxrSupportedCurrencies = "AED,AFN,ALL,AMD,ANG,AOA,ARS,AUD,AWG,AZN,BAM,BBD," +
		"BDT,BGN,BHD,BIF,BMD,BND,BOB,BRL,BSD,BTC,BTN,BWP,BYN,BYR,BZD,CAD,CDF," +
		"CHF,CLF,CLP,CNH,CNY,COP,CRC,CUC,CUP,CVE,CZK,DJF,DKK,DOP,DZD,EEK,EGP," +
		"ERN,ETB,EUR,FJD,FKP,GBP,GEL,GGP,GHS,GIP,GMD,GNF,GTQ,GYD,HKD,HNL,HRK," +
		"HTG,HUF,IDR,ILS,IMP,INR,IQD,IRR,ISK,JEP,JMD,JOD,JPY,KES,KGS,KHR,KMF," +
		"KPW,KRW,KWD,KYD,KZT,LAK,LBP,LKR,LRD,LSL,LYD,MAD,MDL,MGA,MKD,MMK,MNT," +
		"MOP,MRO,MRU,MTL,MUR,MVR,MWK,MXN,MYR,MZN,NAD,NGN,NIO,NOK,NPR,NZD,OMR," +
		"PAB,PEN,PGK,PHP,PKR,PLN,PYG,QAR,RON,RSD,RUB,RWF,SAR,SBD,SCR,SDG,SEK," +
		"SGD,SHP,SLL,SOS,SRD,SSP,STD,STN,SVC,SYP,SZL,THB,TJS,TMT,TND,TOP,TRY," +
		"TTD,TWD,TZS,UAH,UGX,USD,UYU,UZS,VEF,VND,VUV,WST,XAF,XAG,XAU,XCD,XDR," +
		"XOF,XPD,XPF,XPT,YER,ZAR,ZMK,ZMW"
)

// OXR is a foreign exchange rate provider at https://openexchangerates.org/
// this is the overarching type across this package
// DOCs : https://docs.openexchangerates.org/docs
type OXR struct {
	base.Base
	Requester *request.Requester
}

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
	Disclaimer  string         `json:"disclaimer"`
	License     string         `json:"license"`
	StartDate   string         `json:"start_date"`
	EndDate     string         `json:"end_date"`
	Base        string         `json:"base"`
	Rates       map[string]any `json:"rates"`
	Error       bool           `json:"error"`
	Status      int            `json:"status"`
	Message     string         `json:"message"`
	Description string         `json:"description"`
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
	Disclaimer  string         `json:"disclaimer"`
	License     string         `json:"license"`
	StartDate   string         `json:"start_date"`
	EndDate     string         `json:"end_date"`
	Base        string         `json:"base"`
	Rates       map[string]any `json:"rates"`
	Error       bool           `json:"error"`
	Status      int            `json:"status"`
	Message     string         `json:"message"`
	Description string         `json:"description"`
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
