package okex

import "github.com/shopspring/decimal"

// ContractPrice holds date and ticker price price for contracts.
type ContractPrice struct {
	Date   string `json:"date"`
	Ticker struct {
		Buy        decimal.Decimal `json:"buy"`
		ContractID int             `json:"contract_id"`
		High       decimal.Decimal `json:"high"`
		Low        decimal.Decimal `json:"low"`
		Last       decimal.Decimal `json:"last"`
		Sell       decimal.Decimal `json:"sell"`
		UnitAmount decimal.Decimal `json:"unit_amount"`
		Vol        decimal.Decimal `json:"vol"`
	} `json:"ticker"`
	Result bool        `json:"result"`
	Error  interface{} `json:"error_code"`
}

// ContractDepth response depth
type ContractDepth struct {
	Asks   []interface{} `json:"asks"`
	Bids   []interface{} `json:"bids"`
	Result bool          `json:"result"`
	Error  interface{}   `json:"error_code"`
}

// ActualContractDepth better manipulated structure to return
type ActualContractDepth struct {
	Asks []struct {
		Price  decimal.Decimal
		Volume decimal.Decimal
	}
	Bids []struct {
		Price  decimal.Decimal
		Volume decimal.Decimal
	}
}

// ActualContractTradeHistory holds contract trade history
type ActualContractTradeHistory struct {
	Amount   decimal.Decimal `json:"amount"`
	DateInMS decimal.Decimal `json:"date_ms"`
	Date     decimal.Decimal `json:"date"`
	Price    decimal.Decimal `json:"price"`
	TID      decimal.Decimal `json:"tid"`
	Type     string          `json:"buy"`
}

// CandleStickData holds candlestick data
type CandleStickData struct {
	Timestamp decimal.Decimal `json:"timestamp"`
	Open      decimal.Decimal `json:"open"`
	High      decimal.Decimal `json:"high"`
	Low       decimal.Decimal `json:"low"`
	Close     decimal.Decimal `json:"close"`
	Volume    decimal.Decimal `json:"volume"`
	Amount    decimal.Decimal `json:"amount"`
}

// Info holds individual information
type Info struct {
	AccountRights decimal.Decimal `json:"account_rights"`
	KeepDeposit   decimal.Decimal `json:"keep_deposit"`
	ProfitReal    decimal.Decimal `json:"profit_real"`
	ProfitUnreal  decimal.Decimal `json:"profit_unreal"`
	RiskRate      decimal.Decimal `json:"risk_rate"`
}

// UserInfo holds a collection of user data
type UserInfo struct {
	Info struct {
		BTC Info `json:"btc"`
		LTC Info `json:"ltc"`
	} `json:"info"`
	Result bool `json:"result"`
}

// HoldData is a sub type for FuturePosition
type HoldData struct {
	BuyAmount      decimal.Decimal `json:"buy_amount"`
	BuyAvailable   decimal.Decimal `json:"buy_available"`
	BuyPriceAvg    decimal.Decimal `json:"buy_price_avg"`
	BuyPriceCost   decimal.Decimal `json:"buy_price_cost"`
	BuyProfitReal  decimal.Decimal `json:"buy_profit_real"`
	ContractID     int             `json:"contract_id"`
	ContractType   string          `json:"contract_type"`
	CreateDate     int             `json:"create_date"`
	LeverRate      decimal.Decimal `json:"lever_rate"`
	SellAmount     decimal.Decimal `json:"sell_amount"`
	SellAvailable  decimal.Decimal `json:"sell_available"`
	SellPriceAvg   decimal.Decimal `json:"sell_price_avg"`
	SellPriceCost  decimal.Decimal `json:"sell_price_cost"`
	SellProfitReal decimal.Decimal `json:"sell_profit_real"`
	Symbol         string          `json:"symbol"`
}

// FuturePosition contains an array of holding types
type FuturePosition struct {
	ForceLiquidationPrice decimal.Decimal `json:"force_liqu_price"`
	Holding               []HoldData      `json:"holding"`
}

// FutureTradeHistory will contain futures trade data
type FutureTradeHistory struct {
	Amount decimal.Decimal `json:"amount"`
	Date   int             `json:"date"`
	Price  decimal.Decimal `json:"price"`
	TID    decimal.Decimal `json:"tid"`
	Type   string          `json:"type"`
}

// SpotPrice holds date and ticker price price for contracts.
type SpotPrice struct {
	Date   string `json:"date"`
	Ticker struct {
		Buy        decimal.Decimal `json:"buy,string"`
		ContractID int             `json:"contract_id"`
		High       decimal.Decimal `json:"high,string"`
		Low        decimal.Decimal `json:"low,string"`
		Last       decimal.Decimal `json:"last,string"`
		Sell       decimal.Decimal `json:"sell,string"`
		UnitAmount decimal.Decimal `json:"unit_amount,string"`
		Vol        decimal.Decimal `json:"vol,string"`
	} `json:"ticker"`
	Result bool        `json:"result"`
	Error  interface{} `json:"error_code"`
}

// SpotDepth response depth
type SpotDepth struct {
	Asks   []interface{} `json:"asks"`
	Bids   []interface{} `json:"bids"`
	Result bool          `json:"result"`
	Error  interface{}   `json:"error_code"`
}

// ActualSpotDepthRequestParams represents Klines request data.
type ActualSpotDepthRequestParams struct {
	Symbol string `json:"symbol"` // Symbol; example ltc_btc
	Size   int    `json:"size"`   // value: 1-200
}

// ActualSpotDepth better manipulated structure to return
type ActualSpotDepth struct {
	Asks []struct {
		Price  decimal.Decimal
		Volume decimal.Decimal
	}
	Bids []struct {
		Price  decimal.Decimal
		Volume decimal.Decimal
	}
}

// ActualSpotTradeHistoryRequestParams represents Klines request data.
type ActualSpotTradeHistoryRequestParams struct {
	Symbol string `json:"symbol"` // Symbol; example ltc_btc
	Since  int    `json:"since"`  // TID; transaction record ID (return data does not include the current TID value, returning up to 600 items)
}

// ActualSpotTradeHistory holds contract trade history
type ActualSpotTradeHistory struct {
	Amount   decimal.Decimal `json:"amount"`
	DateInMS decimal.Decimal `json:"date_ms"`
	Date     decimal.Decimal `json:"date"`
	Price    decimal.Decimal `json:"price"`
	TID      decimal.Decimal `json:"tid"`
	Type     string          `json:"buy"`
}

// SpotUserInfo holds the spot user info
type SpotUserInfo struct {
	Result bool                                    `json:"result"`
	Info   map[string]map[string]map[string]string `json:"info"`
}

// SpotNewOrderRequestParams holds the params for making a new spot order
type SpotNewOrderRequestParams struct {
	Amount decimal.Decimal         `json:"amount"` // Order quantity
	Price  decimal.Decimal         `json:"price"`  // Order price
	Symbol string                  `json:"symbol"` // Symbol; example btc_usdt, eth_btc......
	Type   SpotNewOrderRequestType `json:"type"`   // Order type (see below)
}

// SpotNewOrderRequestType order type
type SpotNewOrderRequestType string

var (
	// SpotNewOrderRequestTypeBuy buy order
	SpotNewOrderRequestTypeBuy = SpotNewOrderRequestType("buy")

	// SpotNewOrderRequestTypeSell sell order
	SpotNewOrderRequestTypeSell = SpotNewOrderRequestType("sell")

	// SpotNewOrderRequestTypeBuyMarket buy market order
	SpotNewOrderRequestTypeBuyMarket = SpotNewOrderRequestType("buy_market")

	// SpotNewOrderRequestTypeSellMarket sell market order
	SpotNewOrderRequestTypeSellMarket = SpotNewOrderRequestType("sell_market")
)

// KlinesRequestParams represents Klines request data.
type KlinesRequestParams struct {
	Symbol string       // Symbol; example btcusdt, bccbtc......
	Type   TimeInterval // Kline data time interval; 1min, 5min, 15min......
	Size   int          // Size; [1-2000]
	Since  int64        // Since timestamp, return data after the specified timestamp (for example, 1417536000000)
}

// TimeInterval represents interval enum.
type TimeInterval string

// vars for time intervals
var (
	TimeIntervalMinute         = TimeInterval("1min")
	TimeIntervalThreeMinutes   = TimeInterval("3min")
	TimeIntervalFiveMinutes    = TimeInterval("5min")
	TimeIntervalFifteenMinutes = TimeInterval("15min")
	TimeIntervalThirtyMinutes  = TimeInterval("30min")
	TimeIntervalHour           = TimeInterval("1hour")
	TimeIntervalFourHours      = TimeInterval("4hour")
	TimeIntervalSixHours       = TimeInterval("6hour")
	TimeIntervalTwelveHours    = TimeInterval("12hour")
	TimeIntervalDay            = TimeInterval("1day")
	TimeIntervalThreeDays      = TimeInterval("3day")
	TimeIntervalWeek           = TimeInterval("1week")
)
