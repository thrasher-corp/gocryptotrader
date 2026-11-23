package allratestoday

import (
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/base"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
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
	Time   Timestamp `json:"time"`
}

// Timestamp is a rate's time. The rates endpoint may send its offset without a
// colon (2026-10-22T23:07:01+0000), which time.Time cannot decode, or as RFC 3339,
// so both forms are accepted
type Timestamp time.Time

// UnmarshalJSON deserialises a rate's timestamp
func (t *Timestamp) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}

	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("%w %s: %w", errInvalidTimestamp, data, err)
	}
	if value == "" {
		return nil
	}

	for _, layout := range []string{"2006-01-02T15:04:05Z0700", time.RFC3339Nano} {
		if parsed, err := time.Parse(layout, value); err == nil {
			*t = Timestamp(parsed)
			return nil
		}
	}
	return fmt.Errorf("%w: %q", errInvalidTimestamp, value)
}

// MarshalJSON serialises the timestamp in RFC 3339 format
func (t Timestamp) MarshalJSON() ([]byte, error) {
	return t.Time().MarshalJSON()
}

// Time converts Timestamp to time.Time
func (t Timestamp) Time() time.Time {
	return time.Time(t)
}
