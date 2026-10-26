package okx

import (
	"bytes"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// UnmarshalJSON decoder for OpenInterestResponse instance.
func (a *OpenInterest) UnmarshalJSON(data []byte) error {
	type Alias OpenInterest
	chil := &struct {
		*Alias
		InstrumentType string `json:"instType"`
	}{Alias: (*Alias)(a)}
	err := json.Unmarshal(data, chil)
	if err != nil {
		return err
	}
	chil.InstrumentType = strings.ToUpper(chil.InstrumentType)
	a.InstrumentType, err = assetTypeFromInstrumentType(chil.InstrumentType)
	return err
}

// MarshalJSON serialised QuoteLeg instance into bytes
func (a *QuoteLeg) MarshalJSON() ([]byte, error) {
	type Alias QuoteLeg
	chil := &struct {
		*Alias
		Side string `json:"side"`
	}{
		Alias: (*Alias)(a),
	}
	if a.Side == order.Buy {
		chil.Side = "buy"
	} else {
		chil.Side = "sell"
	}
	return json.Marshal(chil)
}

// UnmarshalJSON decodes OKX's numeric strings while retaining explicit zero values.
func (n *OptionalNumber) UnmarshalJSON(data []byte) error {
	var value types.Number
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	data = bytes.TrimSpace(data)
	n.Value = nil
	if !bytes.Equal(data, []byte(`null`)) && !bytes.Equal(data, []byte(`""`)) {
		n.Value = &value
	}
	return nil
}

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
