package okx

import "github.com/thrasher-corp/gocryptotrader/encoding/json"

// UnmarshalJSON preserves presence of legacy fee fields, including explicit zero rates.
func (r *TradeFeeRate) UnmarshalJSON(data []byte) error {
	type rate TradeFeeRate
	var decoded rate
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	var fields tradeFeePresence
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	decoded.MakerPresent = fields.Maker != nil && string(*fields.Maker) != `""`
	decoded.TakerPresent = fields.Taker != nil && string(*fields.Taker) != `""`
	decoded.MakerUSDTPresent = fields.MakerUSDT != nil && string(*fields.MakerUSDT) != `""`
	decoded.TakerUSDTPresent = fields.TakerUSDT != nil && string(*fields.TakerUSDT) != `""`
	decoded.MakerUSDCPresent = fields.MakerUSDC != nil && string(*fields.MakerUSDC) != `""`
	decoded.TakerUSDCPresent = fields.TakerUSDC != nil && string(*fields.TakerUSDC) != `""`
	*r = TradeFeeRate(decoded)
	return nil
}

// UnmarshalJSON retains explicit zero rates without treating absent fields as free trading.
func (r *TradeFeeGroup) UnmarshalJSON(data []byte) error {
	type feeGroup TradeFeeGroup
	var decoded feeGroup
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	var fields tradeFeePresence
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	decoded.MakerPresent = fields.Maker != nil && string(*fields.Maker) != `""`
	decoded.TakerPresent = fields.Taker != nil && string(*fields.Taker) != `""`
	*r = TradeFeeGroup(decoded)
	return nil
}
