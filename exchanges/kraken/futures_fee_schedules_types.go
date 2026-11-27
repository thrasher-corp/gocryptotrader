package kraken

import "time"

// FuturesFeeSchedulesResponse holds the fee schedules
type FuturesFeeSchedulesResponse struct {
	FeeSchedules []FuturesFeeSchedule `json:"feeSchedules"`
	ServerTime   time.Time            `json:"serverTime"`
}

// FuturesFeeSchedule holds a fee schedule's tiers
type FuturesFeeSchedule struct {
	UID   string           `json:"uid"`
	Name  string           `json:"name"`
	Tiers []FuturesFeeTier `json:"tiers"`
}

// FuturesFeeTier holds a fee tier's maker and taker fees, in percent
type FuturesFeeTier struct {
	MakerFee float64 `json:"makerFee"`
	TakerFee float64 `json:"takerFee"`
	// USDVolume is the 30 day volume, in USD, from which the tier applies
	USDVolume float64 `json:"usdVolume"`
}

// FuturesFeeScheduleVolumesResponse holds the account's 30 day volume for each fee schedule
type FuturesFeeScheduleVolumesResponse struct {
	// VolumesByFeeSchedule maps each fee schedule's UID to the account's 30 day volume
	VolumesByFeeSchedule map[string]float64 `json:"volumesByFeeSchedule"`
	ServerTime           time.Time          `json:"serverTime"`
}
