package okx

import (
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// SupportCoinResponse holds the currencies the trading statistics endpoints support
type SupportCoinResponse struct {
	ContractCurrencies []currency.Code `json:"contract"`
	OptionCurrencies   []currency.Code `json:"option"`
	OptionV2Currencies []currency.Code `json:"optionV2"`
	SpotCurrencies     []currency.Code `json:"spot"`
}

// ContractStatisticsRequest holds the parameters for the contract trading statistics endpoints
type ContractStatisticsRequest struct {
	InstrumentID string
	Period       kline.Interval
	Begin        time.Time // returns records newer than this time
	End          time.Time // returns records earlier than this time
	Limit        uint64
}

// ContractOpenInterestHistoryResponse holds a contract's open interest at one time
type ContractOpenInterestHistoryResponse struct {
	Timestamp             types.Time
	OpenInterestContracts types.Number
	OpenInterestCurrency  types.Number
	OpenInterestUSD       types.Number
}

// UnmarshalJSON decodes a contract's open interest from its array form
func (c *ContractOpenInterestHistoryResponse) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[4]any{&c.Timestamp, &c.OpenInterestContracts, &c.OpenInterestCurrency, &c.OpenInterestUSD})
}

// TakerVolumeRequest holds the parameters for GetTakerVolume
type TakerVolumeRequest struct {
	Currency       currency.Code
	InstrumentType string // SPOT or CONTRACTS
	Begin          time.Time
	End            time.Time
	Period         kline.Interval
}

// TakerVolumeResponse holds the taker sell and buy volumes at one time
type TakerVolumeResponse struct {
	Timestamp  types.Time
	SellVolume types.Number
	BuyVolume  types.Number
}

// UnmarshalJSON decodes taker volumes from their array form
func (t *TakerVolumeResponse) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[3]any{&t.Timestamp, &t.SellVolume, &t.BuyVolume})
}

// ContractTakerVolumeRequest holds the parameters for GetContractTakerVolume
type ContractTakerVolumeRequest struct {
	InstrumentID string
	Period       kline.Interval
	Unit         string    // 0: crypto; 1: contracts, the default; 2: USD
	Begin        time.Time // returns records newer than this time
	End          time.Time // returns records earlier than this time
	Limit        uint64
}

// CurrencyStatisticsRequest holds the parameters for the currency trading statistics endpoints
type CurrencyStatisticsRequest struct {
	Currency currency.Code
	Begin    time.Time
	End      time.Time
	Period   kline.Interval
}

// MarginLendingRatioResponse holds the ratio of the cumulative margin loans of quote currency to base currency at one
// time
type MarginLendingRatioResponse struct {
	Timestamp types.Time
	Ratio     types.Number
}

// UnmarshalJSON decodes a margin lending ratio from its array form
func (m *MarginLendingRatioResponse) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[2]any{&m.Timestamp, &m.Ratio})
}

// LongShortRatioResponse holds a long to short ratio at one time
type LongShortRatioResponse struct {
	Timestamp types.Time
	Ratio     types.Number
}

// UnmarshalJSON decodes a long to short ratio from its array form
func (l *LongShortRatioResponse) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[2]any{&l.Timestamp, &l.Ratio})
}

// OpenInterestVolumeResponse holds the open interest and trading volume at one time
type OpenInterestVolumeResponse struct {
	Timestamp    types.Time
	OpenInterest types.Number
	Volume       types.Number
}

// UnmarshalJSON decodes open interest and trading volume from their array form
func (o *OpenInterestVolumeResponse) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[3]any{&o.Timestamp, &o.OpenInterest, &o.Volume})
}

// OptionStatisticsRequest holds the parameters for the option trading statistics endpoints
type OptionStatisticsRequest struct {
	Currency currency.Code
	Period   kline.Interval // 8H, the default, or 1D
}

// PutCallRatioResponse holds the call to put open interest and trading volume ratios at one time
type PutCallRatioResponse struct {
	Timestamp         types.Time
	OpenInterestRatio types.Number
	VolumeRatio       types.Number
}

// UnmarshalJSON decodes call to put ratios from their array form
func (p *PutCallRatioResponse) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[3]any{&p.Timestamp, &p.OpenInterestRatio, &p.VolumeRatio})
}

// OpenInterestVolumeExpiryResponse holds the call and put open interest and trading volume of one expiry, in coin
type OpenInterestVolumeExpiryResponse struct {
	Timestamp        types.Time
	ExpiryTime       time.Time
	CallOpenInterest types.Number
	PutOpenInterest  types.Number
	CallVolume       types.Number
	PutVolume        types.Number
}

// UnmarshalJSON decodes an expiry's open interest and trading volume from their array form
func (o *OpenInterestVolumeExpiryResponse) UnmarshalJSON(data []byte) error {
	var expiryTimeString string
	if err := json.Unmarshal(data, &[6]any{&o.Timestamp, &expiryTimeString, &o.CallOpenInterest, &o.PutOpenInterest, &o.CallVolume, &o.PutVolume}); err != nil {
		return err
	}
	if len(expiryTimeString) != len(expiryDateLayout) {
		return nil
	}
	expiryTime, err := time.Parse(expiryDateLayout, expiryTimeString)
	if err != nil {
		return fmt.Errorf("error parsing expiry date: %w", err)
	}
	o.ExpiryTime = expiryTime
	return nil
}

// OpenInterestVolumeStrikeRequest holds the parameters for GetOpenInterestAndVolumeStrike
type OpenInterestVolumeStrikeRequest struct {
	Currency   currency.Code
	ExpiryTime time.Time // sent as its UTC date
	Period     kline.Interval
}

// OpenInterestVolumeStrikeResponse holds the call and put open interest and trading volume of one strike, in coin
type OpenInterestVolumeStrikeResponse struct {
	Timestamp        types.Time
	Strike           types.Number
	CallOpenInterest types.Number
	PutOpenInterest  types.Number
	CallVolume       types.Number
	PutVolume        types.Number
}

// UnmarshalJSON decodes a strike's open interest and trading volume from their array form
func (o *OpenInterestVolumeStrikeResponse) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[6]any{&o.Timestamp, &o.Strike, &o.CallOpenInterest, &o.PutOpenInterest, &o.CallVolume, &o.PutVolume})
}

// TakerFlowResponse holds the option taker buy and sell volumes and block volumes, in settlement currency
type TakerFlowResponse struct {
	Timestamp       types.Time
	CallBuyVolume   types.Number
	CallSellVolume  types.Number
	PutBuyVolume    types.Number
	PutSellVolume   types.Number
	CallBlockVolume types.Number
	PutBlockVolume  types.Number
}

// UnmarshalJSON decodes the option taker flow from its array form
func (t *TakerFlowResponse) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[7]any{&t.Timestamp, &t.CallBuyVolume, &t.CallSellVolume, &t.PutBuyVolume, &t.PutSellVolume, &t.CallBlockVolume, &t.PutBlockVolume})
}
