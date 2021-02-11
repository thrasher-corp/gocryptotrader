package common

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

const (
	// DoNothing is an explicit signal for the backtester to not perform an action
	// based upon indicator results
	DoNothing order.Side = "DO NOTHING"
	// CouldNotBuy/Sell is flagged when a BUY/SELL signal is raised in the strategy/signal phase, but the
	// portfolio manager or exchange cannot place an order
	CouldNotBuy  order.Side = "COULD NOT BUY"
	CouldNotSell order.Side = "COULD NOT SELL"
	// Missing Data is signalled during the strategy/signal phase when data has been identified as missing
	// No buy or sell events can occur
	MissingData order.Side = "MISSING DATA"
	// used to identify the type of data in a config
	CandleStr = "candle"
	TradeStr  = "trade"

	// DataCandle is an int64 representation of a candle data type
	DataCandle = iota
	DataTrade
)

// EventHandler interface implements required GetTime() & Pair() return
type EventHandler interface {
	IsEvent() bool
	GetTime() time.Time
	Pair() currency.Pair
	GetExchange() string
	GetInterval() kline.Interval
	GetAssetType() asset.Item

	GetWhy() string
	AppendWhy(string)
}

// DataHandler interface used for loading and interacting with Data
type DataEventHandler interface {
	EventHandler
	ClosePrice() float64
	HighPrice() float64
	LowPrice() float64
	OpenPrice() float64
}

// Directioner dictates the side of an order
type Directioner interface {
	SetDirection(side order.Side)
	GetDirection() order.Side
}

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

                 ____             __   __            __           
                / __ )____ ______/ /__/ /____  _____/ /____  _____
               / __  / __  / ___/ //_/ __/ _ \/ ___/ __/ _ \/ ___/
              / /_/ / /_/ / /__/ ,< / /_/  __(__  ) /_/  __/ /
             /_____/\__,_/\___/_/|_|\__/\___/____/\__/\___/_/
`
