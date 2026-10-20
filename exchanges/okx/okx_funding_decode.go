package okx

import "github.com/thrasher-corp/gocryptotrader/encoding/json"

// UnmarshalJSON preserves whether OKX supplied a realised settlement rate.
func (r *FundingRateResponse) UnmarshalJSON(data []byte) error {
	type rate FundingRateResponse
	type response struct {
		rate
		RealisedRate OptionalNumber `json:"realizedRate"`
	}
	var decoded response
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	decoded.rate.RealisedRate = decoded.RealisedRate.Value
	*r = FundingRateResponse(decoded.rate)
	return nil
}
