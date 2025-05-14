package kraken

import (
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

var (
	validOrderTypes = map[order.Type]string{
		order.Limit:      "lmt",
		order.Stop:       "stp",
		order.TakeProfit: "take_profit",
		order.Market:     "mkt",
	}

	validSide = []string{"buy", "sell"}

	validTriggerSignal = []string{"mark", "index", "last"}

	validReduceOnly = []string{"true", "false"}

	validBatchOrderType = []string{
		"edit", "cancel", "send",
	}
)

// WSFuturesTickerData stores ws ticker data for futures websocket
type WSFuturesTickerData struct {
	Time                          types.Time `json:"time"`
	Feed                          string     `json:"feed"`
	ProductID                     string     `json:"product_id"`
	Bid                           float64    `json:"bid"`
	Ask                           float64    `json:"ask"`
	BidSize                       float64    `json:"bid_size"`
	AskSize                       float64    `json:"ask_size"`
	Volume                        float64    `json:"volume"`
	DTM                           float64    `json:"dtm"`
	Leverage                      string     `json:"leverage"`
	Index                         float64    `json:"index"`
	Premium                       float64    `json:"premium"`
	Last                          float64    `json:"last"`
	Change                        float64    `json:"change"`
	Suspended                     bool       `json:"suspended"`
	Tag                           string     `json:"tag"`
	Pair                          string     `json:"pair"`
	OpenInterest                  float64    `json:"openinterest"`
	MarkPrice                     float64    `json:"markPrice"`
	MaturityTime                  types.Time `json:"maturityTime"`
	FundingRate                   float64    `json:"funding_rate"`
	FundingRatePrediction         float64    `json:"funding_rate_prediction"`
	RelativeFundingRate           float64    `json:"relative_funding_rate"`
	RelativeFundingRatePrediction float64    `json:"relative_funding_rate_prediction"`
	NextFundingRateTime           types.Time `json:"next_funding_rate_time"`
}

// WsFuturesTradeData stores public trade data for futures websocket
type WsFuturesTradeData struct {
	Feed      string `json:"feed"`
	ProductID string `json:"product_id"`
	Trades    []struct {
		Feed        string     `json:"feed"`
		ProductID   string     `json:"product_id"`
		Side        string     `json:"side"`
		ProductType string     `json:"type"`
		Seq         int64      `json:"seq"`
		Time        types.Time `json:"time"`
		Quantity    float64    `json:"qty"`
		Price       float64    `json:"price"`
	} `json:"trades"`
}

// WsFuturesTickerLite stores ticker lite data for futures websocket
type WsFuturesTickerLite struct {
	Feed      string  `json:"feed"`
	ProductID string  `json:"product_id"`
	Bid       float64 `json:"bid"`
	Ask       float64 `json:"ask"`
	Change    float64 `json:"change"`
	Premium   float64 `json:"premium"`
	Volume    float64 `json:"volume"`
	Tag       string  `json:"tag"`
	Pair      string  `json:"pair"`
	DTM       float64 `json:"dtm"`
}

// WsFuturesOB stores orderbook data for futures websocket
type WsFuturesOB struct {
	Feed      string     `json:"feed"`
	ProductID string     `json:"product_id"`
	Seq       int64      `json:"seq"`
	Bids      []wsOBItem `json:"bids"`
	Asks      []wsOBItem `json:"asks"`
}

type wsOBItem struct {
	Price    float64 `json:"price"`
	Quantity float64 `json:"qty"`
}

// WsVerboseOpenOrders stores verbose open orders data for futures websocket
type WsVerboseOpenOrders struct {
	Feed    string `json:"feed"`
	Account string `json:"account"`
	Orders  []struct {
		Instrument     string     `json:"instrument"`
		Time           types.Time `json:"time"`
		LastUpdateTime types.Time `json:"last_update_time"`
		Qty            float64    `json:"qty"`
		Filled         float64    `json:"filled"`
		LimitPrice     float64    `json:"limit_price"`
		StopPrice      float64    `json:"stop_price"`
		OrderType      string     `json:"type"`
		OrderID        string     `json:"order_id"`
		Direction      int64      `json:"direction"`
		ReduceOnly     bool       `json:"reduce_only"`
	} `json:"orders"`
}

// WsOpenPositions stores open positions data for futures websocket
type WsOpenPositions struct {
	Feed      string `json:"feed"`
	Account   string `json:"account"`
	Positions []struct {
		Instrument    string  `json:"instrument"`
		Balance       float64 `json:"balance"`
		EntryPrice    float64 `json:"entry_price"`
		MarkPrice     float64 `json:"mark_price"`
		IndexPrice    float64 `json:"index_price"`
		ProfitAndLoss float64 `json:"pnl"`
	} `json:"positions"`
}

// WsFuturesAccountLog stores account log data for futures websocket
type WsFuturesAccountLog struct {
	Feed string `json:"feed"`
	Logs []struct {
		ID              int64   `json:"id"`
		Date            string  `json:"date"`
		Asset           string  `json:"asset"`
		Info            string  `json:"info"`
		BookingUID      string  `json:"booking_uid"`
		MarginAccount   string  `json:"margin_account"`
		OldBalance      float64 `json:"old_balance"`
		NewBalance      float64 `json:"new_balance"`
		OldAverageEntry float64 `json:"old_average_entry"`
		NewAverageEntry float64 `json:"new_average_entry"`
		TradePrice      float64 `json:"trade_price"`
		MarkPrice       float64 `json:"mark_price"`
		RealizedPNL     float64 `json:"realized_pnl"`
		Fee             float64 `json:"fee"`
		Execution       string  `json:"execution"`
		Collateral      string  `json:"collateral"`
		FundingRate     float64 `json:"funding_rate"`
		RealizedFunding float64 `json:"realized_funding"`
	} `json:"logs"`
}

// WsFuturesFillsData stores fills data for futures websocket
type WsFuturesFillsData struct {
	Feed    string `json:"feed"`
	Account string `json:"account"`
	Fills   []struct {
		Instrument    string     `json:"instrument"`
		Time          types.Time `json:"time"`
		Price         float64    `json:"price"`
		Seq           int64      `json:"seq"`
		Buy           bool       `json:"buy"`
		Quantity      float64    `json:"qty"`
		OrderID       string     `json:"order_id"`
		ClientOrderID string     `json:"cli_order_id"`
		FillID        string     `json:"fill_id"`
		FillType      string     `json:"fill_type"`
	} `json:"fills"`
}

// WsFuturesOpenOrders stores open orders data for futures websocket
type WsFuturesOpenOrders struct {
	Feed    string `json:"feed"`
	Account string `json:"account"`
	Orders  []struct {
		Instrument     string     `json:"instrument"`
		Time           types.Time `json:"time"`
		LastUpdateTime types.Time `json:"last_update_time"`
		Qty            float64    `json:"qty"`
		Filled         float64    `json:"filled"`
		LimitPrice     float64    `json:"limit_price"`
		StopPrice      float64    `json:"stop_price"`
		OrderType      string     `json:"order_type"`
		OrderID        string     `json:"order_id"`
		Direction      string     `json:"direction"`
		ReduceOnly     bool       `json:"reduce_only"`
	} `json:"orders"`
}

// WsAccountBalancesAndMargin stores account balances and margin data for futures websocket
type WsAccountBalancesAndMargin struct {
	Seq            int64  `json:"seq"`
	Feed           string `json:"feed"`
	Account        string `json:"account"`
	MarginAccounts []struct {
		Name              string  `json:"name"`
		PortfolioValue    float64 `json:"pv"`
		Balance           float64 `json:"balance"`
		Funding           float64 `json:"funding"`
		MaintenanceMargin float64 `json:"mm"`
		ProfitAndLoss     float64 `json:"pnl"`
		InitialMargin     float64 `json:"im"`
		AM                float64 `json:"am"`
	} `json:"margin_accounts"`
}

// WsFuturesNotifications stores notifications data for futures websocket
type WsFuturesNotifications struct {
	Feed          string `json:"feed"`
	Notifications []struct {
		ID               int64      `json:"id"`
		NotificationType string     `json:"notificationType"`
		Priority         string     `json:"priority"`
		Note             string     `json:"note"`
		EffectiveTime    types.Time `json:"effective_time"`
	}
}

type assetTranslatorStore struct {
	l      sync.RWMutex
	Assets map[string]string
}

// FuturesOrderbookData stores orderbook data for futures
type FuturesOrderbookData struct {
	ServerTime time.Time `json:"serverTime"`
	Orderbook  struct {
		Bids [][2]float64 `json:"bids"`
		Asks [][2]float64 `json:"asks"`
	} `json:"orderBook"`
}

// TimeResponse type
type TimeResponse struct {
	Unixtime types.Time `json:"unixtime"`
	Rfc1123  string     `json:"rfc1123"`
}

// FuturesInstrumentData stores info for futures market
type FuturesInstrumentData struct {
	Instruments []struct {
		Symbol          string    `json:"symbol"`
		FutureType      string    `json:"type"`
		Underlying      string    `json:"underlying"`
		OpeningDate     time.Time `json:"openingDate"`
		LastTradingTime time.Time `json:"lastTradingTime"`
		TickSize        float64   `json:"tickSize"`
		ContractSize    float64   `json:"contractSize"`
		Tradable        bool      `json:"tradeable"`
		MarginLevels    []struct {
			Contracts         float64 `json:"contracts"`
			InitialMargin     float64 `json:"initialMargin"`
			MaintenanceMargin float64 `json:"maintenanceMargin"`
		} `json:"marginLevels"`
	} `json:"instruments"`
}

// FuturesTradeHistoryData stores trade history data for futures
type FuturesTradeHistoryData struct {
	History []struct {
		Time      time.Time `json:"time"`
		TradeID   int64     `json:"trade_id"`
		Price     float64   `json:"price"`
		Size      float64   `json:"size"`
		Side      string    `json:"side"`
		TradeType string    `json:"type"`
	} `json:"history"`
}

// FuturesTickersData stores info for futures tickers
type FuturesTickersData struct {
	Tickers    []FuturesTicker `json:"tickers"`
	ServerTime time.Time       `json:"serverTime"`
}

// FuturesTickerData stores info for futures ticker
type FuturesTickerData struct {
	Ticker     FuturesTicker `json:"ticker"`
	ServerTime time.Time     `json:"serverTime"`
}

// FuturesEditedOrderData stores an edited order's data
type FuturesEditedOrderData struct {
	ServerTime time.Time `json:"serverTime"`
	EditStatus struct {
		Status       string    `json:"status"`
		OrderID      string    `json:"orderId"`
		ReceivedTime time.Time `json:"receivedTime"`
		OrderEvents  []struct {
			Old FuturesOrderData `json:"old"`
			New FuturesOrderData `json:"new"`
		} `json:"orderEvents"`
		ReduceQuantity string `json:"reduceQuantity"`
		DataType       string `json:"type"`
	} `json:"editStatus"`
}

// FuturesTicker  holds futures ticker data
type FuturesTicker struct {
	Tag                   string        `json:"tag"`
	Pair                  string        `json:"pair"`
	Symbol                currency.Pair `json:"symbol"`
	MarkPrice             float64       `json:"markPrice"`
	Bid                   float64       `json:"bid"`
	BidSize               float64       `json:"bidSize"`
	Ask                   float64       `json:"ask"`
	AskSize               float64       `json:"askSize"`
	Vol24h                float64       `json:"vol24h"`
	OpenInterest          float64       `json:"openInterest"`
	Open24H               float64       `json:"open24h"`
	Last                  float64       `json:"last"`
	LastTime              time.Time     `json:"lastTime"`
	LastSize              float64       `json:"lastSize"`
	Suspended             bool          `json:"suspended"`
	FundingRate           float64       `json:"fundingRate"`
	FundingRatePrediction float64       `json:"fundingRatePrediction"`
	IndexPrice            float64       `json:"indexPrice"`
	PostOnly              bool          `json:"postOnly"`
	Change24H             float64       `json:"change24h"`
}

// FuturesSendOrderData stores send order data
type FuturesSendOrderData struct {
	SendStatus struct {
		OrderID      string    `json:"order_id"`
		Status       string    `json:"status"`
		ReceivedTime time.Time `json:"receivedTime"`
		OrderEvents  []struct {
			UID      string           `json:"uid"`
			Order    FuturesOrderData `json:"order"`
			Reason   string           `json:"reason"`
			DataType string           `json:"type"`
		} `json:"orderEvents"`
	} `json:"sendStatus"`
	ServerTime time.Time `json:"serverTime"`
}

// FuturesOrderData stores order data
type FuturesOrderData struct {
	OrderID             string    `json:"orderId"`
	ClientOrderID       string    `json:"cliOrderId"`
	OrderType           string    `json:"type"`
	Symbol              string    `json:"symbol"`
	Side                string    `json:"side"`
	Quantity            float64   `json:"quantity"`
	Filled              float64   `json:"filled"`
	LimitPrice          float64   `json:"limitPrice"`
	ReduceOnly          bool      `json:"reduceOnly"`
	Timestamp           time.Time `json:"timestamp"`
	LastUpdateTimestamp time.Time `json:"lastUpdateTimestamp"`
}

// FuturesCancelOrderData stores cancel order data for futures
type FuturesCancelOrderData struct {
	CancelStatus struct {
		Status       string    `json:"status"`
		OrderID      string    `json:"order_id"`
		ReceivedTime time.Time `json:"receivedTime"`
		OrderEvents  []struct {
			UID      string           `json:"uid"`
			Order    FuturesOrderData `json:"order"`
			DataType string           `json:"type"`
		} `json:"orderEvents"`
	} `json:"cancelStatus"`
	ServerTime time.Time `json:"serverTime"`
}

// FuturesFillsData stores fills data
type FuturesFillsData struct {
	Fills []struct {
		FillID   string    `json:"fill_id"`
		Symbol   string    `json:"symbol"`
		Side     string    `json:"side"`
		OrderID  string    `json:"order_id"`
		Size     float64   `json:"size"`
		Price    float64   `json:"price"`
		FillTime time.Time `json:"fillTime"`
		FillType string    `json:"fillType"`
	} `json:"fills"`
	ServerTime time.Time `json:"serverTime"`
}

// FuturesTransferData stores transfer data
type FuturesTransferData struct {
	Result     string    `json:"result"`
	ServerTime time.Time `json:"serverTime"`
}

// FuturesOpenPositions stores open positions data for futures
type FuturesOpenPositions struct {
	OpenPositions []struct {
		Side              string    `json:"side"`
		Symbol            string    `json:"symbol"`
		Price             float64   `json:"price"`
		FillTime          time.Time `json:"fillTime"`
		Size              float64   `json:"size"`
		UnrealizedFunding float64   `json:"unrealizedFunding"`
	} `json:"openPositions"`
	ServerTime time.Time `json:"serverTime"`
}

// FuturesNotificationData stores notification data
type FuturesNotificationData struct {
	Notifications []struct {
		NotificationType string `json:"type"`
		Priority         string `json:"priority"`
		Note             string `json:"note"`
		EffectiveTime    string `json:"effectiveTime"`
	} `json:"notifications"`
	ServerTime time.Time `json:"serverTime"`
}

// FuturesAccountsData stores account data
type FuturesAccountsData struct {
	ServerTime string                   `json:"serverTime"`
	Accounts   map[string]*AccountsData `json:"accounts"`
}

// AccountsData stores data of an account
type AccountsData struct {
	AccType   string             `json:"type"`
	Currency  string             `json:"currency"`
	Balances  map[string]float64 `json:"balances"`
	Auxiliary struct {
		AvailableFunds float64 `json:"af"`
		ProfitAndLoss  float64 `json:"pnl"`
		PortfolioValue float64 `json:"pv"`
	} `json:"auxiliary"`
	MarginRequirements struct {
		InitialMargin        float64 `json:"im"`
		MaintenanceMargin    float64 `json:"mm"`
		LiquidationThreshold float64 `json:"lt"`
		TerminationThreshold float64 `json:"tt"`
	} `json:"marginRequirements"`
	TriggerEstimates struct {
		InitialMargin        float64 `json:"im"`
		MaintenanceMargin    float64 `json:"mm"`
		LiquidationThreshold float64 `json:"lt"`
		TerminationThreshold float64 `json:"tt"`
	} `json:"triggerEstimates"`
}

// CancelAllOrdersData stores order data for all cancelled orders
type CancelAllOrdersData struct {
	CancelStatus struct {
		ReceivedTime    time.Time `json:"receivedTime"`
		CancelOnly      string    `json:"cancelOnly"`
		Status          string    `json:"status"`
		CancelledOrders []struct {
			OrderID string `json:"order_id"`
		} `json:"cancelledOrders"`
		OrderEvents []struct {
			UID string `json:"uid"`
		} `json:"uid"`
		Order    FuturesOrderData `json:"order"`
		DataType string           `json:"type"`
	} `json:"cancelStatus"`
	ServerTime time.Time `json:"serverTime"`
}

// CancelOrdersAfterData stores data of all orders after a certain time that are cancelled
type CancelOrdersAfterData struct {
	Result string `json:"result"`
	Status struct {
		CurrentTime time.Time `json:"currentTime"`
		TriggerTime time.Time `json:"triggerTime"`
	} `json:"status"`
	ServerTime time.Time `json:"serverTime"`
}

// RecentOrderData stores order data of a recent order
type RecentOrderData struct {
	UID        string    `json:"uid"`
	AccountID  string    `json:"accountId"`
	Tradeable  string    `json:"tradeable"`
	Direction  string    `json:"direction"`
	Quantity   float64   `json:"quantity,string"`
	Filled     float64   `json:"filled,string"`
	Timestamp  time.Time `json:"timestamp"`
	LimitPrice float64   `json:"limitPrice,string"`
	OrderType  string    `json:"orderType"`
	ClientID   string    `json:"clientId"`
	StopPrice  float64   `json:"stopPrice,string"`
}

// FOpenOrdersData stores open orders data for futures
type FOpenOrdersData struct {
	OrderID        string    `json:"order_id"`
	ClientOrderID  string    `json:"cliOrdId"`
	Symbol         string    `json:"symbol"`
	Side           string    `json:"side"`
	OrderType      string    `json:"orderType"`
	LimitPrice     float64   `json:"limitPrice"`
	StopPrice      float64   `json:"stopPrice"`
	UnfilledSize   float64   `json:"unfilledSize"`
	ReceivedTime   time.Time `json:"receivedTime"`
	Status         string    `json:"status"`
	FilledSize     float64   `json:"filledSize"`
	ReduceOnly     bool      `json:"reduceOnly"`
	TriggerSignal  string    `json:"triggerSignal"`
	LastUpdateTime time.Time `json:"lastUpdateTime"`
}

// FuturesRecentOrdersData stores recent orders data
type FuturesRecentOrdersData struct {
	OrderEvents []struct {
		Timestamp int64 `json:"timestamp"`
		Event     struct {
			Timestamp   string `json:"timestamp"`
			UID         string `json:"uid"`
			OrderPlaced struct {
				RecentOrder RecentOrderData `json:"order"`
				Reason      string          `json:"reason"`
			} `json:"orderPlaced"`
			OrderCancelled struct {
				RecentOrder RecentOrderData `json:"order"`
				Reason      string          `json:"reason"`
			} `json:"orderCancelled"`
			OrderRejected struct {
				RecentOrder RecentOrderData `json:"order"`
				Reason      string          `json:"reason"`
			} `json:"orderRejected"`
			ExecutionEvent struct {
				Execution   ExecutionData `json:"execution"`
				Timestamp   string        `json:"timestamp"`
				Quantity    float64       `json:"quantity"`
				Price       float64       `json:"price"`
				MarkPrice   float64       `json:"markPrice"`
				LimitFilled bool          `json:"limitFilled"`
			} `json:"executionEvent"`
		} `json:"event"`
	} `json:"orderEvents"`
}

// BatchOrderData stores batch order data
type BatchOrderData struct {
	Result      string    `json:"result"`
	ServerTime  time.Time `json:"serverTime"`
	BatchStatus []struct {
		Status           string    `json:"status"`
		OrderTag         string    `json:"order_tag"`
		OrderID          string    `json:"order_id"`
		DateTimeReceived time.Time `json:"dateTimeReceived"`
		OrderEvents      []struct {
			OrderPlaced    FuturesOrderData `json:"orderPlaced"`
			ReduceOnly     bool             `json:"reduceOnly"`
			Timestamp      time.Time        `json:"timestamp"`
			OldEditedOrder FuturesOrderData `json:"old"`
			NewEditedOrder FuturesOrderData `json:"new"`
			UID            string           `json:"uid"`
			RequestType    string           `json:"requestType"`
		} `json:"orderEvents"`
	} `json:"batchStatus"`
}

// PlaceBatchOrderData stores data required to place a batch order
type PlaceBatchOrderData struct {
	PlaceOrderType string  `json:"order,omitempty"`
	OrderType      string  `json:"orderType,omitempty"`
	OrderTag       string  `json:"order_tag,omitempty"`
	Symbol         string  `json:"symbol,omitempty"`
	Side           string  `json:"side,omitempty"`
	Size           float64 `json:"size,omitempty"`
	LimitPrice     float64 `json:"limitPrice,omitempty"`
	StopPrice      float64 `json:"stopPrice,omitempty"`
	ClientOrderID  int64   `json:"cliOrdId,omitempty"`
	ReduceOnly     string  `json:"reduceOnly,omitempty"`
	OrderID        string  `json:"order_id,omitempty"`
}

// ExecutionData stores execution data
type ExecutionData struct {
	UID        string          `json:"uid"`
	TakerOrder RecentOrderData `json:"takerOrder"`
}

// FuturesOpenOrdersData stores open orders data for futures
type FuturesOpenOrdersData struct {
	Result     string            `json:"result"`
	OpenOrders []FOpenOrdersData `json:"openOrders"`
	ServerTime time.Time         `json:"serverTime"`
}

// FuturesPublicTrades returns public trade data
// a terrible type for parsing data
type FuturesPublicTrades struct {
	ContinuationToken string `json:"continuationToken"`
	Elements          []struct {
		ExecutionEvent struct {
			OuterExecutionHolder struct {
				Execution struct {
					LimitFilled    bool                  `json:"limitFilled"`
					MakerOrder     FutureTradeOrder      `json:"makerOrder"`
					MakerOrderData FuturesTradeOrderData `json:"makerOrderData"`
					MarkPrice      float64               `json:"markPrice,string"`
					OldTakerOrder  FutureTradeOrder      `json:"oldTakerOrder"`
					Price          float64               `json:"price,string"`
					Quantity       float64               `json:"quantity,string"`
					TakerOrder     FutureTradeOrder      `json:"takerOrder"`
					TakerOrderData FuturesTradeOrderData `json:"takerOrderData"`
					Timestamp      types.Time            `json:"timestamp"`
					UID            string                `json:"uid"`
					UsdValue       float64               `json:"usdValue,string"`
				} `json:"execution"`
				TakerReducedQuantity string `json:"takerReducedQuantity"`
			} `json:"execution"`
		} `json:"event"`
		Timestamp types.Time `json:"timestamp"`
		UID       string     `json:"uid"`
	} `json:"elements"`
	Len int64 `json:"len"`
}

// FutureTradeOrder holds details about the order for a futures trade
type FutureTradeOrder struct {
	AccountUID          string     `json:"accountUid"`
	ClientID            string     `json:"clientId"`
	Direction           string     `json:"direction"`
	Filled              string     `json:"filled"`
	LastUpdateTimestamp types.Time `json:"lastUpdateTimestamp"`
	LimitPrice          float64    `json:"limitPrice,string"`
	OrderType           string     `json:"orderType"`
	Quantity            float64    `json:"quantity,string"`
	ReduceOnly          bool       `json:"reduceOnly"`
	SpotData            string     `json:"spotData"`
	Timestamp           types.Time `json:"timestamp"`
	Tradeable           string     `json:"tradeable"`
	UID                 string     `json:"uid"`
}

// FuturesTradeOrderData holds additional order details for a futures trade order
type FuturesTradeOrderData struct {
	Fee          float64 `json:"fee,string"`
	PositionSize float64 `json:"positionSize,string"`
}

// FuturesCandles is the response when requesting futures candles
type FuturesCandles struct {
	Candles     []FuturesCandle `json:"candles"`
	MoreCandles bool            `json:"more_candles"`
}

// FuturesCandle is an individual candle
type FuturesCandle struct {
	Close  float64    `json:"close,string"`
	High   float64    `json:"high,string"`
	Low    float64    `json:"low,string"`
	Open   float64    `json:"open,string"`
	Time   types.Time `json:"time"`
	Volume float64    `json:"volume,string"`
}
