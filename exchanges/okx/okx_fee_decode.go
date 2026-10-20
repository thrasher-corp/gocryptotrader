package okx

import "github.com/thrasher-corp/gocryptotrader/encoding/json"

// UnmarshalJSON preserves presence of legacy fee fields, including explicit zero rates.
func (r *TradeFeeRate) UnmarshalJSON(data []byte) error {
	type rate TradeFeeRate
	type response struct {
		rate
		Maker     OptionalNumber `json:"maker"`
		Taker     OptionalNumber `json:"taker"`
		MakerUSDT OptionalNumber `json:"makerU"`
		TakerUSDT OptionalNumber `json:"takerU"`
		MakerUSDC OptionalNumber `json:"makerUSDC"`
		TakerUSDC OptionalNumber `json:"takerUSDC"`
	}
	var decoded response
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	decoded.FeeRateMaker = decoded.Maker.Value
	decoded.FeeRateTaker = decoded.Taker.Value
	decoded.FeeRateMakerUSDT = decoded.MakerUSDT.Value
	decoded.FeeRateTakerUSDT = decoded.TakerUSDT.Value
	decoded.FeeRateMakerUSDC = decoded.MakerUSDC.Value
	decoded.FeeRateTakerUSDC = decoded.TakerUSDC.Value
	*r = TradeFeeRate(decoded.rate)
	return nil
}

// UnmarshalJSON retains explicit zero rates without treating absent fields as free trading.
func (r *TradeFeeGroup) UnmarshalJSON(data []byte) error {
	type feeGroup TradeFeeGroup
	type response struct {
		feeGroup
		Maker OptionalNumber `json:"maker"`
		Taker OptionalNumber `json:"taker"`
	}
	var decoded response
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	decoded.MakerRate = decoded.Maker.Value
	decoded.TakerRate = decoded.Taker.Value
	*r = TradeFeeGroup(decoded.feeGroup)
	return nil
}
