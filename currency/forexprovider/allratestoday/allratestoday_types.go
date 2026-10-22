package allratestoday

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/base"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	// APIURL is the default AllRatesToday API endpoint.
	APIURL = "https://allratestoday.com/api/v1/"

	supportedCurrencies = "AED,AFN,ALL,AMD,ANG,AOA,ARS,AUD,AWG,AZN,BAM,BBD," +
		"BDT,BGN,BHD,BIF,BMD,BND,BOB,BRL,BSD,BTN,BWP,BYN,BZD,CAD,CDF,CHF,CLP," +
		"CNH,CNY,COP,CRC,CUP,CVE,CZK,DJF,DKK,DOP,DZD,EGP,ERN,ETB,EUR,FJD,FKP," +
		"GBP,GEL,GGP,GHS,GIP,GMD,GNF,GTQ,GYD,HKD,HNL,HRK,HTG,HUF,IDR,ILS,IMP," +
		"INR,IQD,IRR,ISK,JEP,JMD,JOD,JPY,KES,KGS,KHR,KMF,KPW,KRW,KWD,KYD,KZT," +
		"LAK,LBP,LKR,LRD,LSL,LYD,MAD,MDL,MGA,MKD,MMK,MNT,MOP,MRU,MUR,MVR,MWK," +
		"MXN,MYR,MZN,NAD,NGN,NIO,NOK,NPR,NZD,OMR,PAB,PEN,PGK,PHP,PKR,PLN,PYG," +
		"QAR,RON,RSD,RUB,RWF,SAR,SBD,SCR,SDG,SEK,SGD,SHP,SLE,SLL,SOS,SRD,SSP," +
		"STN,SVC,SYP,SZL,THB,TJS,TMT,TND,TOP,TRY,TTD,TWD,TZS,UAH,UGX,USD,UYU," +
		"UZS,VES,VND,VUV,WST,XAF,XCD,XCG,XOF,XPF,YER,ZAR,ZMW,ZWG,ZWL"
)

// AllRatesToday is a foreign exchange rate provider at https://allratestoday.com/
// DOCs : https://allratestoday.com/docs/
type AllRatesToday struct {
	base.Base
	Requester *request.Requester
	APIURL    string
}

// RateResponse is a single rate returned by the rates endpoint
type RateResponse struct {
	Source string    `json:"source"`
	Target string    `json:"target"`
	Rate   float64   `json:"rate"`
	Time   time.Time `json:"time"`
}
