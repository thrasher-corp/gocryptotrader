package okx

import "github.com/thrasher-corp/gocryptotrader/encoding/json"

// UnmarshalJSON preserves whether OKX supplied a realised settlement rate.
func (r *FundingRateResponse) UnmarshalJSON(data []byte) error {
	type rate FundingRateResponse
	var decoded rate
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	var fields fundingRatePresence
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	decoded.RealisedRatePresent = fields.RealisedRate != nil && string(*fields.RealisedRate) != `""`
	*r = FundingRateResponse(decoded)
	return nil
}
