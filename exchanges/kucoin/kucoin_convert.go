package kucoin

import (
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

// UnmarshalJSON valid data to SubAccountsResponse of return nil if the data is empty list.
// this is added to handle the empty list returned when there are no accounts.
func (a *SubAccountsResponse) UnmarshalJSON(data []byte) error {
	var result any
	err := json.Unmarshal(data, &result)
	if err != nil {
		return err
	}
	var ok bool
	if a, ok = result.(*SubAccountsResponse); ok {
		if a == nil {
			return errNoValidResponseFromServer
		}
		return nil
	} else if _, ok := result.([]any); ok {
		return nil
	}
	return fmt.Errorf("%w can not unmarshal to SubAccountsResponse", common.ErrMalformedData)
}
