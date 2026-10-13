package bybit

import (
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// RestResponse is the envelope Bybit wraps every REST reply in. P2P endpoints use snake case keys for the same fields
type RestResponse struct {
	RetCode      int64           `json:"retCode"`
	RetMsg       string          `json:"retMsg"`
	Result       json.RawMessage `json:"result"`
	RetExtInfo   json.RawMessage `json:"retExtInfo"`
	Time         types.Time      `json:"time"`
	SnakeRetCode int64           `json:"ret_code"`
	SnakeRetMsg  string          `json:"ret_msg"`
	ExtCode      string          `json:"ext_code"`
	ExtInfo      json.RawMessage `json:"ext_info"`
	TimeNow      types.Number    `json:"time_now"`
}

// ErrorMessage represents an error message item
type ErrorMessage struct {
	Code    int64  `json:"code"`
	Message string `json:"msg"`
}
