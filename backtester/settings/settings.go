package settings

import (
	"time"
)

// Settings holds all initial command line settings when running the application
type Settings struct {
	StartTime                  string
	EndTime                    string
	Interval                   time.Duration
	InitialFunds               float64
	OrderSize                  float64
	MaximumOrderSize           float64
	IsOrderSizePercentageBased bool
	ExchangeName               string
	CurrencyPair               string
	AssetType                  string
	RunName                    string
	StrategyName               string
	DataSource                 string
	DataPath                   string
	DataType                   string
}

type DataSource string

const (
	FileSource     = "file"
	ExchangeSource = "exchange"
	DatabaseSource = "database"
)

type DataType string

const (
	KlineType = "kline"
)
