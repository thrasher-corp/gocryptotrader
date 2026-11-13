package lbank

import (
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// beijingTime is the UTC+8 zone LBank gives some times in
var beijingTime = time.FixedZone("UTC+8", 8*60*60)

// v2Response is the envelope LBank wraps its replies in
type v2Response struct {
	// Result is a pointer so an absent or null result stays distinct from an explicit false: several successful replies
	// omit result and send only data, and a non-pointer types.Boolean would read those successes as failures. LBank sends
	// it as a JSON boolean on some endpoints and as a JSON string on others, which types.Boolean accepts both of
	Result *types.Boolean `json:"result"`
	// ErrorCode is negative for the contract API's system exception (-99); the contract docs type it as a string while
	// it arrives as a number, and types.Number accepts both
	ErrorCode types.Number    `json:"error_code"`
	Message   string          `json:"msg"`
	Data      json.RawMessage `json:"data"`
}

// requestError is a failure an LBank reply reports, with LBank's error code
type requestError struct {
	code    int64
	message string
}

func (r *requestError) Error() string {
	return fmt.Sprintf("%s: %s (error_code %d)", errRequestFailed, r.message, r.code)
}

// Unwrap returns errRequestFailed, the category every reported failure falls under
func (r *requestError) Unwrap() error {
	return errRequestFailed
}
