package okex

// ContractPrice holds date and ticker price price for contracts.
type ContractPrice struct {
	Date   string `json:"date"`
	Ticker struct {
		Buy        float64 `json:"buy"`
		ContractID int     `json:"contract_id"`
		High       float64 `json:"high"`
		Low        float64 `json:"low"`
		Last       float64 `json:"last"`
		Sell       float64 `json:"sell"`
		UnitAmount float64 `json:"unit_amount"`
		Vol        float64 `json:"vol"`
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
		Price  float64
		Volume float64
	}
	Bids []struct {
		Price  float64
		Volume float64
	}
}

// ActualContractTradeHistory holds contract trade history
type ActualContractTradeHistory struct {
	Amount   float64 `json:"amount"`
	DateInMS float64 `json:"date_ms"`
	Date     float64 `json:"date"`
	Price    float64 `json:"price"`
	TID      float64 `json:"tid"`
	Type     string  `json:"buy"`
}

// CandleStickData holds candlestick data
type CandleStickData struct {
	Timestamp float64 `json:"timestamp"`
	Open      float64 `json:"open"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Close     float64 `json:"close"`
	Volume    float64 `json:"volume"`
	Amount    float64 `json:"amount"`
}

// Info holds individual information
type Info struct {
	AccountRights float64 `json:"account_rights"`
	KeepDeposit   float64 `json:"keep_deposit"`
	ProfitReal    float64 `json:"profit_real"`
	ProfitUnreal  float64 `json:"profit_unreal"`
	RiskRate      float64 `json:"risk_rate"`
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
	BuyAmount      float64 `json:"buy_amount"`
	BuyAvailable   float64 `json:"buy_available"`
	BuyPriceAvg    float64 `json:"buy_price_avg"`
	BuyPriceCost   float64 `json:"buy_price_cost"`
	BuyProfitReal  float64 `json:"buy_profit_real"`
	ContractID     int     `json:"contract_id"`
	ContractType   string  `json:"contract_type"`
	CreateDate     int     `json:"create_date"`
	LeverRate      float64 `json:"lever_rate"`
	SellAmount     float64 `json:"sell_amount"`
	SellAvailable  float64 `json:"sell_available"`
	SellPriceAvg   float64 `json:"sell_price_avg"`
	SellPriceCost  float64 `json:"sell_price_cost"`
	SellProfitReal float64 `json:"sell_profit_real"`
	Symbol         string  `json:"symbol"`
}

// FuturePosition contains an array of holding types
type FuturePosition struct {
	ForceLiquidationPrice float64    `json:"force_liqu_price"`
	Holding               []HoldData `json:"holding"`
}

// FutureTradeHistory will contain futures trade data
type FutureTradeHistory struct {
	Amount float64 `json:"amount"`
	Date   int     `json:"date"`
	Price  float64 `json:"price"`
	TID    float64 `json:"tid"`
	Type   string  `json:"type"`
}

// SpotPrice holds date and ticker price price for contracts.
type SpotPrice struct {
	Date   string `json:"date"`
	Ticker struct {
		Buy        float64 `json:"buy,string"`
		ContractID int     `json:"contract_id"`
		High       float64 `json:"high,string"`
		Low        float64 `json:"low,string"`
		Last       float64 `json:"last,string"`
		Sell       float64 `json:"sell,string"`
		UnitAmount float64 `json:"unit_amount,string"`
		Vol        float64 `json:"vol,string"`
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

// OrderBookDataRequestParams represents Klines request data.
type ActualSpotDepthRequestParams struct {
	Symbol string `json:"symbol"` //必填项，币对如ltc_btc
	Size   int    `json:"size"`   // value: 1-200
}

// ActualSpotDepth better manipulated structure to return
type ActualSpotDepth struct {
	Asks []struct {
		Price  float64
		Volume float64
	}
	Bids []struct {
		Price  float64
		Volume float64
	}
}

// ActualSpotTradeHistoryRequestParams represents Klines request data.
type ActualSpotTradeHistoryRequestParams struct {
	Symbol string `json:"symbol"` //必填项，币对如ltc_btc
	Since  int    `json:"since"`  // tid:交易记录ID(返回数据不包括当前tid值,最多返回600条数据)
}

// ActualSpotTradeHistory holds contract trade history
type ActualSpotTradeHistory struct {
	Amount   float64 `json:"amount"`
	DateInMS float64 `json:"date_ms"`
	Date     float64 `json:"date"`
	Price    float64 `json:"price"`
	TID      float64 `json:"tid"`
	Type     string  `json:"buy"`
}

//---------- UserInfo ----------

// SpotUserInfo 获取用户信息
type SpotUserInfo struct {
	Result bool                                    `json:"result"`
	Info   map[string]map[string]map[string]string `json:"info"`
}

//---------- 下订单 ----------

// SpotNewOrderRequestParams 下单买入/卖出请求参数
type SpotNewOrderRequestParams struct {
	Amount float64                 `json:"amount"` // 下单数量
	Price  float64                 `json:"price"`  // 下单价格
	Symbol string                  `json:"symbol"` // 交易对, btc_usdt, eth_btc......
	Type   SpotNewOrderRequestType `json:"type"`   // 订单类型,
}

// SpotNewOrderRequestType 交易类型
type SpotNewOrderRequestType string

var (
	// SpotNewOrderRequestTypeBuy 限价买
	SpotNewOrderRequestTypeBuy = SpotNewOrderRequestType("buy")

	// SpotNewOrderRequestTypeSell 限价卖
	SpotNewOrderRequestTypeSell = SpotNewOrderRequestType("sell")

	// SpotNewOrderRequestTypeBuyMarket 市价买
	SpotNewOrderRequestTypeBuyMarket = SpotNewOrderRequestType("buy_market")

	// SpotNewOrderRequestTypeSellMarket 市价卖
	SpotNewOrderRequestTypeSellMarket = SpotNewOrderRequestType("sell_market")
)

//---------Kline

// KlinesRequestParams represents Klines request data.
type KlinesRequestParams struct {
	Symbol string       //交易对, btcusdt, bccbtc......
	Type   TimeInterval //K线类型, 1min, 5min, 15min......
	Size   int          //获取数量, [1-2000]
	Since  int64        //时间戳，返回该时间戳以后的数据(例如1417536000000)
}

// TimeInterval represents interval enum.
type TimeInterval string

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
