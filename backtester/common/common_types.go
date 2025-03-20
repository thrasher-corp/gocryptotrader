package common

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	// CandleStr is a config readable data type to tell the backtester to retrieve candle data
	CandleStr = "candle"
	// TradeStr is a config readable data type to tell the backtester to retrieve trade data
	TradeStr = "trade"

	// DataCandle is an int64 representation of a candle data type
	DataCandle int64 = iota
	// DataTrade is an int64 representation of a trade data type
	DataTrade
)

var (
	// ErrNilEvent is a common error for whenever a nil event occurs when it shouldn't have
	ErrNilEvent = errors.New("nil event received")
	// ErrInvalidDataType occurs when an invalid data type is defined in the config
	ErrInvalidDataType = errors.New("invalid datatype received")
	// ErrFileNotFound returned when the file is not found
	ErrFileNotFound = errors.New("file not found")

	errCannotGenerateFileName = errors.New("cannot generate filename")
)

// Event interface implements required GetTime() & Pair() return
type Event interface {
	GetBase() *event.Base
	GetOffset() int64
	SetOffset(int64)
	IsEvent() bool
	GetTime() time.Time
	Pair() currency.Pair
	GetUnderlyingPair() currency.Pair
	GetExchange() string
	GetInterval() kline.Interval
	GetAssetType() asset.Item
	GetConcatReasons() string
	GetReasons() []string
	GetClosePrice() decimal.Decimal
	AppendReason(string)
	AppendReasonf(string, ...any)
}

// custom subloggers for backtester use
var (
	Backtester         *log.SubLogger
	LiveStrategy       *log.SubLogger
	Setup              *log.SubLogger
	Strategy           *log.SubLogger
	Config             *log.SubLogger
	Portfolio          *log.SubLogger
	Exchange           *log.SubLogger
	Fill               *log.SubLogger
	Report             *log.SubLogger
	Statistics         *log.SubLogger
	CurrencyStatistics *log.SubLogger
	FundingStatistics  *log.SubLogger
	Holdings           *log.SubLogger
	Data               *log.SubLogger
	FundManager        *log.SubLogger
)

// Directioner dictates the side of an order
type Directioner interface {
	SetDirection(side order.Side)
	GetDirection() order.Side
}

// Colours defines colour types for CMD output
type Colours struct {
	Default  string
	Green    string
	White    string
	Grey     string
	DarkGrey string
	H1       string
	H2       string
	H3       string
	H4       string
	Success  string
	Info     string
	Debug    string
	Warn     string
	Error    string
}

// CMDColours holds colour information for CMD output
var CMDColours = Colours{
	Default:  "\u001b[0m",
	Green:    "\033[38;5;157m",
	White:    "\033[38;5;255m",
	Grey:     "\033[38;5;240m",
	DarkGrey: "\033[38;5;243m",
	H1:       "\033[38;5;33m",
	H2:       "\033[38;5;39m",
	H3:       "\033[38;5;45m",
	H4:       "\033[38;5;51m",
	Success:  "\033[38;5;40m",
	Info:     "\u001B[32m",
	Debug:    "\u001B[34m",
	Warn:     "\u001B[33m",
	Error:    "\033[38;5;196m",
}

// ASCIILogo is a sweet logo that is optionally printed to the command line window
const ASCIILogo = `
   ______      ______                 __      ______               __         
  / ____/___  / ____/______  ______  / /_____/_  __/________ _____/ /__  _____
 / / __/ __ \/ /   / ___/ / / / __ \/ __/ __ \/ / / ___/ __  / __  / _ \/ ___/
/ /_/ / /_/ / /___/ /  / /_/ / /_/ / /_/ /_/ / / / /  / /_/ / /_/ /  __/ /
\____/\____/\____/_/   \__, / .___/\__/\____/_/ /_/   \__,_/\__,_/\___/_/
                       /___/
                 ____             __   __            __           
                / __ )____ ______/ /__/ /____  _____/ /____  _____
               / __  / __  / ___/ //_/ __/ _ \/ ___/ __/ _ \/ ___/
              / /_/ / /_/ / /__/ ,< / /_/  __(__  ) /_/  __/ /
             /_____/\__,_/\___/_/|_|\__/\___/____/\__/\___/_/

`
