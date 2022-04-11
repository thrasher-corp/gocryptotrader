package common

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

const (
	// CandleStr is a config readable data type to tell the backtester to retrieve candle data
	CandleStr = "candle"
	// TradeStr is a config readable data type to tell the backtester to retrieve trade data
	TradeStr = "trade"
)

// DataCandle is an int64 representation of a candle data type
const (
	DataCandle = iota
	DataTrade
)

var (
	// ErrNilArguments is a common error response to highlight that nils were passed in
	// when they should not have been
	ErrNilArguments = errors.New("received nil argument(s)")
	// ErrNilEvent is a common error for whenever a nil event occurs when it shouldn't have
	ErrNilEvent = errors.New("nil event received")
	// ErrInvalidDataType occurs when an invalid data type is defined in the config
	ErrInvalidDataType = errors.New("invalid datatype received")
)

// EventHandler interface implements required GetTime() & Pair() return
type EventHandler interface {
	GetOffset() int64
	SetOffset(int64)
	IsEvent() bool
	GetTime() time.Time
	Pair() currency.Pair
	GetExchange() string
	GetInterval() kline.Interval
	GetAssetType() asset.Item
	GetReason() string
	AppendReason(string)
}

// DataEventHandler interface used for loading and interacting with Data
type DataEventHandler interface {
	EventHandler
	GetClosePrice() decimal.Decimal
	GetHighPrice() decimal.Decimal
	GetLowPrice() decimal.Decimal
	GetOpenPrice() decimal.Decimal
}

// Directioner dictates the side of an order
type Directioner interface {
	SetDirection(side order.Side)
	GetDirection() order.Side
}

// ASCIILogo is a sweet logo that is optionally printed to the command line window
const ASCIILogo = `
                                                                                
                               @@@@@@@@@@@@@@@@@                                
                            @@@@@@@@@@@@@@@@@@@@@@@    ,,,,,,                   
                           @@@@@@@@,,,,,    @@@@@@@@@,,,,,,,,                   
                         @@@@@@@@,,,,,,,       @@@@@@@,,,,,,,                   
                         @@@@@@(,,,,,,,,      ,,@@@@@@@,,,,,,                   
                       ,,@@@@@@,,,,,,,,,   #,,,,,,,,,,,,,,,,,                   
                    ,,,,*@@@@@@,,,,,,,,,,,,,,,,,,,,,,,,,,%%%%%%%                
                 ,,,,,,,*@@@@@@,,,,,,,,,,,,,,%%%%%,,,,,,%%%%%%%%                
                ,,,,,,,,*@@@@@@,,,,,,,,,,,%%%%%%%%%%%%%%%%%%#%%                 
                  ,,,,,,*@@@@@@,,,,,,,,,%%%,,,,,%%%%%%%%,,,,,                   
                     ,,,*@@@@@@,,,,,,%%,  ,,,,,,,@*%%,@,,,,,,                   
                        *@@@@@@,,,,,,,,,     ,,,,@@@@@@,,,,,,                   
                         @@@@@@,,,,,,,,,        @@@@@@@,,,,,,                   
                         @@@@@@@@,,,,,,,       @@@@@@@,,,,,,,                   
                           @@@@@@@@@,,,,    @@@@@@@@@#,,,,,,,                   
                            @@@@@@@@@@@@@@@@@@@@@@@     *,,,,                   
                                @@@@@@@@@@@@@@@@                                
                                                                               
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
