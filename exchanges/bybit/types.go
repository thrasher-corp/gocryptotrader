package bybit

import (
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// RestResponse is the envelope Bybit wraps every REST reply in. P2P endpoints use snake case keys for the same fields
type RestResponse struct {
	ReturnCode         int64           `json:"retCode"`
	ReturnMessage      string          `json:"retMsg"`
	Result             json.RawMessage `json:"result"`
	ReturnExtendedInfo json.RawMessage `json:"retExtInfo"`
	Time               types.Time      `json:"time"`
	SnakeReturnCode    int64           `json:"ret_code"`
	SnakeReturnMessage string          `json:"ret_msg"`
	ExtendedCode       string          `json:"ext_code"`
	ExtendedInfo       json.RawMessage `json:"ext_info"`
	TimeNow            types.Number    `json:"time_now"`
}
