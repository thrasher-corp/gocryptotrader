package bybit

import (
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// RestResponse is the envelope Bybit wraps every REST reply in. P2P endpoints use snake case keys for the same fields
type RestResponse struct {
	ReturnCode         int64           `json:"retCode"` // Bybit's error codes include negative ones, such as -1 and -2015
	ReturnMessage      string          `json:"retMsg"`
	Result             json.RawMessage `json:"result"`
	ReturnExtendedInfo json.RawMessage `json:"retExtInfo"`
	Time               types.Time      `json:"time"`
	SnakeReturnCode    int64           `json:"ret_code"` // Bybit's error codes include negative ones, such as -1 and -2015
	SnakeReturnMessage string          `json:"ret_msg"`
	ExtendedCode       string          `json:"ext_code"`
	ExtendedInfo       json.RawMessage `json:"ext_info"`
	TimeNow            types.Number    `json:"time_now"`
}
