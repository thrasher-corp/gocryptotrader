package settings

import (
	"time"
)

// Settings holds all initial command line settings when running the application
type Settings struct {
	StartTime    string
	EndTime      string
	Interval     time.Duration
	InitialFunds float64
	ExchangeName string
	CurrencyPair string
	AssetType    string
	RunName      string
	StrategyName string
}
