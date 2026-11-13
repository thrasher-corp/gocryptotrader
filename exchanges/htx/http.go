package htx

import (
	"fmt"
	"net/http"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// UnmarshalJSON decodes the response directly without retaining an untyped JSON payload.
func (r *httpResponse) UnmarshalJSON(payload []byte) error {
	r.Received = true
	errorKind := errAPIResponse
	if r.Authenticated {
		errorKind = request.ErrAuthRequestFailed
	}
	if !r.Authenticated || r.Futures {
		var status errorCapture
		if err := json.Unmarshal(payload, &status); err == nil {
			if status.ErrMsgType1 != "" {
				return fmt.Errorf("%w error code: %v error message: %s", errorKind, status.CodeType1, status.ErrMsgType1)
			}
			if status.ErrMsgType2 != "" {
				return fmt.Errorf("%w error code: %v error message: %s", errorKind, status.CodeType2, status.ErrMsgType2)
			}
			if status.Status == htxStatusError {
				return fmt.Errorf("%w: status %s", errorKind, status.Status)
			}
		}
	} else if !r.Version2 {
		var status Response
		if err := json.Unmarshal(payload, &status); err == nil && status.Status == htxStatusError {
			return fmt.Errorf("%w error code: %s error message: %s", errorKind, status.ErrorCode, status.ErrorMessage)
		}
	}
	if r.Version2 {
		var status ResponseV2
		if err := json.Unmarshal(payload, &status); err == nil && status.Code != 0 && status.Code != http.StatusOK {
			return fmt.Errorf("%w error code: %d error message: %s", errorKind, status.Code, status.Message)
		}
	} else if r.Version5 {
		status := new(V5Response)
		if err := json.Unmarshal(payload, status); err == nil && status.Code != 0 && status.Code != http.StatusOK {
			message := status.Message
			if message == "" {
				message = status.V3Message
			}
			return fmt.Errorf("%w error code: %d error message: %s", errorKind, status.Code, message)
		}
	}
	if err := unmarshalResponse(payload, r.Result); err != nil {
		if r.Authenticated {
			return common.AppendError(err, request.ErrAuthRequestFailed)
		}
		return err
	}
	return nil
}
