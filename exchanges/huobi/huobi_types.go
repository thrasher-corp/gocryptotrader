package huobi

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

type errorCapture struct {
	Status      string     `json:"status"`
	CodeType1   any        `json:"err-code"` // can be either a string or int depending on the endpoint
	ErrMsgType1 string     `json:"err-msg"`
	CodeType2   any        `json:"err_code"`
	ErrMsgType2 string     `json:"err_msg"`
	Timestamp   types.Time `json:"ts"`
}

// MarketSummary24Hr stores past 24hr market summary data of a given symbol
type MarketSummary24Hr struct {
	Tick struct {
		Amount  float64 `json:"amount"`
		Open    float64 `json:"open"`
		Close   float64 `json:"close"`
		High    float64 `json:"high"`
		ID      int64   `json:"id"`
		Count   float64 `json:"count"`
		Low     float64 `json:"low"`
		Version int64   `json:"version"`
		Volume  float64 `json:"vol"`
	}
}

// CurrenciesChainData stores currency and chain info
type CurrenciesChainData struct {
	Currency   string `json:"currency"`
	AssetType  uint8  `json:"assetType"`
	InstStatus string `json:"instStatus"`
	ChainData  []*struct {
		Chain                     string  `json:"chain"`
		DisplayName               string  `json:"displayName"`
		BaseChain                 string  `json:"baseChain"`
		BaseChainProtocol         string  `json:"baseChainProtocol"`
		IsDynamic                 bool    `json:"isDynamic"`
		NumberOfConfirmations     uint16  `json:"numOfConfirmations"`
		NumberOfFastConfirmations uint16  `json:"numOfFastConfirmations"`
		DepositStatus             string  `json:"depositStatus"`
		MinimumDepositAmount      float64 `json:"minDepositAmt,string"`
		WithdrawStatus            string  `json:"withdrawStatus"`
		MinimumWithdrawalAmount   float64 `json:"minWithdrawAmt,string"`
		WithdrawPrecision         int16   `json:"withdrawPrecision"`
		MaximumWithdrawAmount     float64 `json:"maxWithdrawwAmt,string"`
		WithdrawQuotaPerDay       float64 `json:"withdrawQuotaPerDay,string"`
		WithdrawQuotaPerYear      float64 `json:"withdrawQuotaPerYear,string"`
		WithdrawQuotaTotal        float64 `json:"withdrawQuotaTotal,string"`
		WithdrawFeeType           string  `json:"withdrawFeeType"`
		TransactFeeWithdraw       float64 `json:"transactFeeWithdraw,string"`
		AddressWithTag            bool    `json:"addrWithTag"`
		AddressDepositTag         bool    `json:"addrDepositTag"`
	} `json:"chains"`
}

// WsKlineData stores kline data for futures and swap websocket
type WsKlineData struct {
	Channel   string     `json:"ch"`
	Timestamp types.Time `json:"ts"`
	Tick      struct {
		ID     int64   `json:"id"`
		MRID   int64   `json:"mrid"`
		Volume float64 `json:"vol"`
		Count  float64 `json:"count"`
		Open   float64 `json:"open"`
		Close  float64 `json:"close"`
		Low    float64 `json:"low"`
		High   float64 `json:"high"`
		Amount float64 `json:"amount"`
	} `json:"tick"`
}

// WsMarketDepth stores market depth data for futures and swap websocket
type WsMarketDepth struct {
	Channel   string     `json:"ch"`
	Timestamp types.Time `json:"ts"`
	Tick      struct {
		MRID      int64        `json:"mrid"`
		ID        int64        `json:"id"`
		Bids      [][2]float64 `json:"bids"`
		Asks      [][2]float64 `json:"asks"`
		Timestamp types.Time   `json:"ts"`
		Version   int64        `json:"version"`
		Channel   string       `json:"ch"`
	} `json:"tick"`
}

// WsIncrementalMarketDepth stores incremental market depth data for swap and futures websocket
type WsIncrementalMarketDepth struct {
	Channel   string     `json:"ch"`
	Timestamp types.Time `json:"ts"`
	Tick      struct {
		MRID      int64        `json:"mrid"`
		ID        int64        `json:"id"`
		Bids      [][2]float64 `json:"bids"`
		Asks      [][2]float64 `json:"asks"`
		Timestamp types.Time   `json:"ts"`
		Version   int64        `json:"version"`
		Channel   string       `json:"ch"`
		Event     string       `json:"event"`
	} `json:"tick"`
}

// WsMarketDetail stores market detail data for futures and swap websocket
type WsMarketDetail struct {
	Channel   string     `json:"ch"`
	Timestamp types.Time `json:"ts"`
	Tick      struct {
		ID     int64   `json:"id"`
		MRID   int64   `json:"mrid"`
		Open   float64 `json:"open"`
		Close  float64 `json:"close"`
		High   float64 `json:"high"`
		Low    float64 `json:"low"`
		Amount float64 `json:"amount"`
		Volume float64 `json:"vol"`
		Count  float64 `json:"count"`
	} `json:"tick"`
}

// WsMarketBBOData stores BBO data for futures and swap websocket
type WsMarketBBOData struct {
	Channel   string     `json:"ch"`
	Timestamp types.Time `json:"ts"`
	Tick      struct {
		Channel   string     `json:"ch"`
		MRID      int64      `json:"mrid"`
		ID        int64      `json:"id"`
		Bid       [2]float64 `json:"bid"`
		Ask       [2]float64 `json:"ask"`
		Timestamp types.Time `json:"ts"`
		Version   int64      `json:":version"`
	} `json:"tick"`
}

// WsSubTradeDetail stores trade detail data for futures websocket
type WsSubTradeDetail struct {
	Channel   string     `json:"ch"`
	Timestamp types.Time `json:"ts"`
	Tick      struct {
		ID        int64      `json:"id"`
		Timestamp types.Time `json:"ts"`
		Data      []struct {
			Amount    float64    `json:"amount"`
			Timestamp types.Time `json:"ts"`
			ID        int64      `json:"id"`
			Price     float64    `json:"price"`
			Direction string     `json:"direction"`
		} `json:"data"`
	} `json:"tick"`
}

//

// Futures

// FWsRequestKline stores requested kline data for futures websocket
type FWsRequestKline struct {
	Rep  string `json:"rep"`
	ID   string `json:"id"`
	WsID int64  `json:"wsid"`
	Tick []struct {
		Volume float64 `json:"vol"`
		Count  float64 `json:"count"`
		ID     int64   `json:"id"`
		Open   float64 `json:"open"`
		Close  float64 `json:"close"`
		Low    float64 `json:"low"`
		High   float64 `json:"high"`
		Amount float64 `json:"amount"`
	} `json:"tick"`
}

// FWsReqTradeDetail stores requested trade detail data for futures websocket
type FWsReqTradeDetail struct {
	Rep       string     `json:"rep"`
	ID        string     `json:"id"`
	Timestamp types.Time `json:"ts"`
	Data      []struct {
		ID        int64      `json:"id"`
		Price     float64    `json:"price"`
		Amount    float64    `json:"amount"`
		Direction string     `json:"direction"`
		Timestamp types.Time `json:"ts"`
	} `json:"data"`
}

// FWsSubKlineIndex stores subscribed kline index data for futures websocket
type FWsSubKlineIndex struct {
	Channel   string     `json:"ch"`
	Timestamp types.Time `json:"ts"`
	Tick      struct {
		ID     string  `json:"id"`
		Open   float64 `json:"open,string"`
		Close  float64 `json:"close,string"`
		High   float64 `json:"high,string"`
		Low    float64 `json:"low,string"`
		Amount float64 `json:"amount,string"`
		Volume float64 `json:"vol,string"`
		Count  float64 `json:"count,string"`
	} `json:"tick"`
}

// FWsReqKlineIndex stores requested kline index data for futures websocket
type FWsReqKlineIndex struct {
	ID        string     `json:"id"`
	Rep       string     `json:"rep"`
	WsID      int64      `json:"wsid"`
	Timestamp types.Time `json:"ts"`
	Data      []struct {
		ID     int64   `json:"id"`
		Open   float64 `json:"open"`
		Close  float64 `json:"close"`
		Low    float64 `json:"low"`
		High   float64 `json:"high"`
		Amount float64 `json:"amount"`
		Volume float64 `json:"vol"`
		Count  float64 `json:"count"`
	} `json:"data"`
}

// FWsSubBasisData stores subscribed basis data for futures websocket
type FWsSubBasisData struct {
	Channel   string     `json:"ch"`
	Timestamp types.Time `json:"ts"`
	Tick      struct {
		ID            int64   `json:"id"`
		IndexPrice    float64 `json:"index_price,string"`
		ContractPrice float64 `json:"contract_price,string"`
		Basis         float64 `json:"basis,string"`
		BasisRate     float64 `json:"basis_rate,string"`
	}
}

// FWsReqBasisData stores requested basis data for futures websocket
type FWsReqBasisData struct {
	ID        string     `json:"id"`
	Rep       string     `json:"rep"`
	Timestamp types.Time `json:"ts"`
	WsID      int64      `json:"wsid"`
	Tick      struct {
		ID            int64   `json:"id"`
		IndexPrice    float64 `json:"index_price,string"`
		ContractPrice float64 `json:"contract_price,string"`
		Basis         float64 `json:"basis,string"`
		BasisRate     float64 `json:"basis_rate,string"`
	} `json:"tick"`
}

// FWsSubOrderData stores subscribed order data for futures websocket
type FWsSubOrderData struct {
	Operation      string     `json:"op"`
	Topic          string     `json:"topic"`
	UID            string     `json:"uid"`
	Timestamp      types.Time `json:"ts"`
	Symbol         string     `json:"symbol"`
	ContractType   string     `json:"contract_type"`
	ContractCode   string     `json:"contract_code"`
	Volume         float64    `json:"volume"`
	Price          float64    `json:"price"`
	OrderPriceType string     `json:"order_price_type"`
	Direction      string     `json:"direction"`
	Offset         string     `json:"offset"`
	Status         int64      `json:"status"`
	LeverageRate   int64      `json:"lever_rate"`
	OrderID        int64      `json:"order_id"`
	OrderIDString  string     `json:"order_id_string"`
	ClientOrderID  int64      `json:"client_order_id"`
	OrderSource    string     `json:"order_source"`
	OrderType      int64      `json:"order_type"`
	CreatedAt      int64      `json:"created_at"`
	TradeVolume    float64    `json:"trade_volume"`
	TradeTurnover  float64    `json:"trade_turnover"`
	Fee            float64    `json:"fee"`
	TradeAvgPrice  float64    `json:"trade_avg_price"`
	MarginFrozen   float64    `json:"margin_frozen"`
	Profit         float64    `json:"profit"`
	FeeAsset       string     `json:"fee_asset"`
	CancelledAt    int64      `json:"canceled_at"`
	Trade          []struct {
		ID            string  `json:"id"`
		TradeID       int64   `json:"trade_id"`
		TradeVolume   float64 `json:"trade_volume"`
		TradePrice    float64 `json:"trade_price"`
		TradeFee      float64 `json:"trade_fee"`
		TradeTurnover float64 `json:"trade_turnover"`
		CreatedAt     int64   `json:"created_at"`
		Role          string  `json:"role"`
		FeeAsset      string  `json:"fee_asset"`
	} `json:"trade"`
}

// FWsSubMatchOrderData stores subscribed match order data for futures websocket
type FWsSubMatchOrderData struct {
	Operation     string     `json:"op"`
	Topic         string     `json:"topic"`
	UID           string     `json:"uid"`
	Timestamp     types.Time `json:"ts"`
	Symbol        string     `json:"symbol"`
	ContractType  string     `json:"contract_type"`
	ContractCode  string     `json:"contract_code"`
	Status        int64      `json:"status"`
	OrderID       int64      `json:"order_id"`
	OrderIDString string     `json:"order_id_string"`
	OrderType     string     `json:"order_type"`
	Volume        float64    `json:"volume"`
	TradeVolume   float64    `json:"trade_volume"`
	ClientOrderID int64      `json:"client_order_id"`
	Trade         []struct {
		ID            string  `json:"id"`
		TradeID       int64   `json:"trade_id"`
		TradeVolume   float64 `json:"trade_volume"`
		TradePrice    float64 `json:"trade_price"`
		TradeTurnover float64 `json:"trade_turnover"`
		CreatedAt     int64   `json:"created_at"`
		Role          string  `json:"role"`
	}
}

// FWsSubEquityUpdates stores account equity updates data for futures websocket
type FWsSubEquityUpdates struct {
	Operation string     `json:"op"`
	Topic     string     `json:"topic"`
	UID       string     `json:"uid"`
	Timestamp types.Time `json:"ts"`
	Event     string     `json:"event"`
	Data      []struct {
		Symbol            string  `json:"symbol"`
		MarginBalance     float64 `json:"margin_balance"`
		MarginStatic      int64   `json:"margin_static"`
		MarginPosition    float64 `json:"margin_position"`
		MarginFrozen      float64 `json:"margin_frozen"`
		MarginAvailable   float64 `json:"margin_available"`
		ProfitReal        float64 `json:"profit_real"`
		ProfitUnreal      float64 `json:"profit_unreal"`
		WithdrawAvailable float64 `json:"withdraw_available"`
		RiskRate          float64 `json:"risk_rate"`
		LiquidationPrice  float64 `json:"liquidation_price"`
		LeverageRate      float64 `json:"lever_rate"`
		AdjustFactor      float64 `json:"adjust_factor"`
	} `json:"data"`
}

// FWsSubPositionUpdates stores subscribed position updates data for futures websocket
type FWsSubPositionUpdates struct {
	Operation     string     `json:"op"`
	Topic         string     `json:"topic"`
	UID           string     `json:"uid"`
	Timestamp     types.Time `json:"ts"`
	Event         string     `json:"event"`
	PositionsData []struct {
		Symbol         string  `json:"symbol"`
		ContractCode   string  `json:"contract_code"`
		ContractType   string  `json:"contract_type"`
		Volume         float64 `json:"volume"`
		Available      float64 `json:"available"`
		Frozen         float64 `json:"frozen"`
		CostOpen       float64 `json:"cost_open"`
		CostHold       float64 `json:"cost_hold"`
		ProfitUnreal   float64 `json:"profit_unreal"`
		ProfitRate     float64 `json:"profit_rate"`
		Profit         float64 `json:"profit"`
		PositionMargin float64 `json:"position_margin"`
		LeverageRate   float64 `json:"lever_rate"`
		Direction      string  `json:"direction"`
		LastPrice      float64 `json:"last_price"`
	} `json:"data"`
}

// FWsSubLiquidationOrders stores subscribed liquidation orders data for futures websocket
type FWsSubLiquidationOrders struct {
	Operation  string     `json:"op"`
	Topic      string     `json:"topic"`
	Timestamp  types.Time `json:"ts"`
	OrdersData []struct {
		Symbol       string     `json:"symbol"`
		ContractCode string     `json:"contract_code"`
		Direction    string     `json:"direction"`
		Offset       string     `json:"offset"`
		Volume       float64    `json:"volume"`
		Price        float64    `json:"price"`
		CreatedAt    types.Time `json:"created_at"`
	} `json:"data"`
}

// FWsSubContractInfo stores contract info data for futures websocket
type FWsSubContractInfo struct {
	Operation    string     `json:"op"`
	Topic        string     `json:"topic"`
	Timestamp    types.Time `json:"ts"`
	Event        string     `json:"event"`
	ContractData []struct {
		Symbol         string  `json:"symbol"`
		ContractCode   string  `json:"contract_code"`
		ContractType   string  `json:"contract_type"`
		ContractSize   float64 `json:"contract_size"`
		PriceTick      float64 `json:"price_tick"`
		DeliveryDate   string  `json:"delivery_date"`
		CreateDate     string  `json:"create_date"`
		ContractStatus int64   `json:"contract_status"`
	} `json:"data"`
}

// FWsSubTriggerOrderUpdates stores subscribed trigger order updates data for futures websocket
type FWsSubTriggerOrderUpdates struct {
	Operation string `json:"op"`
	Topic     string `json:"topic"`
	UID       string `json:"uid"`
	Event     string `json:"event"`
	Data      []struct {
		Symbol          string  `json:"symbol"`
		ContractCode    string  `json:"contract_code"`
		ContractType    string  `json:"contract_type"`
		TriggerType     string  `json:"trigger_type"`
		Volume          float64 `json:"volume"`
		OrderType       int64   `json:"order_type"`
		Direction       string  `json:"direction"`
		Offset          string  `json:"offset"`
		LeverageRate    int64   `json:"lever_rate"`
		OrderID         int64   `json:"order_id"`
		OrderIDString   string  `json:"order_id_str"`
		RelationOrderID string  `json:"relation_order_id"`
		OrderPriceType  string  `json:"order_price_type"`
		Status          int64   `json:"status"`
		OrderSource     string  `json:"order_source"`
		TriggerPrice    float64 `json:"trigger_price"`
		TriggeredPrice  float64 `json:"triggered_price"`
		OrderPrice      float64 `json:"order_price"`
		CreatedAt       int64   `json:"created_at"`
		TriggeredAt     int64   `json:"triggered_at"`
		OrderInsertAt   int64   `json:"order_insert_at"`
		CancelledAt     int64   `json:"canceled_at"`
		FailCode        int64   `json:"fail_code"`
		FailReason      string  `json:"fail_reason"`
	} `json:"data"`
}

// --------------------------------Spot-----------------------------------------

// Response stores the Huobi response information
type Response struct {
	Status       string     `json:"status"`
	Channel      string     `json:"ch"`
	Timestamp    types.Time `json:"ts"`
	ErrorCode    string     `json:"err-code"`
	ErrorMessage string     `json:"err-msg"`
}

// MarginRatesData stores margin rates data
type MarginRatesData struct {
	Data []struct {
		Symbol     string `json:"symbol"`
		Currencies []struct {
			Currency       string  `json:"currency"`
			InterestRate   float64 `json:"interest-rate,string"`
			MinLoanAmount  float64 `json:"min-loan-amt,string"`
			MaxLoanAmount  float64 `json:"max-loan-amt,string"`
			LoanableAmount float64 `json:"loanable-amt,string"`
			ActualRate     float64 `json:"actual-rate,string"`
		} `json:"currencies"`
	} `json:"data"`
}

// ResponseV2 stores the Huobi generic response info
type ResponseV2 struct {
	Code    int32  `json:"code"`
	Message string `json:"message"`
}

// SwapMarketsData stores market data for swaps
type SwapMarketsData struct {
	Symbol         string     `json:"symbol"`
	ContractCode   string     `json:"contract_code"`
	ContractSize   float64    `json:"contract_size"`
	PriceTick      float64    `json:"price_tick"`
	SettlementDate types.Time `json:"settlement_date"`
	CreateDate     string     `json:"create_date"`
	DeliveryTime   types.Time `json:"delivery_time"`
	ContractStatus int64      `json:"contract_status"`
}

// KlineItem stores a kline item
type KlineItem struct {
	IDTimestamp types.Time `json:"id"`
	Open        float64    `json:"open"`
	Close       float64    `json:"close"`
	Low         float64    `json:"low"`
	High        float64    `json:"high"`
	Amount      float64    `json:"amount"`
	Volume      float64    `json:"vol"`
	Count       int        `json:"count"`
}

// CancelOpenOrdersBatch stores open order batch response data
type CancelOpenOrdersBatch struct {
	Data struct {
		FailedCount  int `json:"failed-count"`
		NextID       int `json:"next-id"`
		SuccessCount int `json:"success-count"`
	} `json:"data"`
	Status       string `json:"status"`
	ErrorMessage string `json:"err-msg"`
}

// DetailMerged stores the ticker detail merged data
type DetailMerged struct {
	Detail
	Version int64     `json:"version"`
	Ask     []float64 `json:"ask"`
	Bid     []float64 `json:"bid"`
}

// Tickers contain all tickers
type Tickers struct {
	Data []Ticker `json:"data"`
}

// FuturesBatchTicker holds ticker data
type FuturesBatchTicker struct {
	ID             float64      `json:"id"`
	Timestamp      types.Time   `json:"ts"`
	Ask            [2]float64   `json:"ask"`
	Bid            [2]float64   `json:"bid"`
	BusinessType   string       `json:"business_type"`
	ContractCode   string       `json:"contract_code"`
	Open           types.Number `json:"open"`
	Close          types.Number `json:"close"`
	Low            types.Number `json:"low"`
	High           types.Number `json:"high"`
	Amount         types.Number `json:"amount"`
	Count          float64      `json:"count"`
	Volume         types.Number `json:"vol"`
	TradeTurnover  types.Number `json:"trade_turnover"`
	TradePartition string       `json:"trade_partition"`
	Symbol         string       `json:"symbol"` // If ContractCode is empty, Symbol is populated
}

// Ticker latest ticker data
type Ticker struct {
	Symbol  string  `json:"symbol"`
	Open    float64 `json:"open"`
	High    float64 `json:"high"`
	Low     float64 `json:"low"`
	Close   float64 `json:"close"`
	Amount  float64 `json:"amount"`
	Volume  float64 `json:"vol"`
	Count   float64 `json:"count"`
	Bid     float64 `json:"bid"`
	BidSize float64 `json:"bidSize"`
	Ask     float64 `json:"ask"`
	AskSize float64 `json:"askSize"`
}

// OrderBookDataRequestParamsType var for request param types
type OrderBookDataRequestParamsType string

// vars for OrderBookDataRequestParamsTypes
var (
	OrderBookDataRequestParamsTypeNone  = OrderBookDataRequestParamsType("")
	OrderBookDataRequestParamsTypeStep0 = OrderBookDataRequestParamsType("step0")
	OrderBookDataRequestParamsTypeStep1 = OrderBookDataRequestParamsType("step1")
	OrderBookDataRequestParamsTypeStep2 = OrderBookDataRequestParamsType("step2")
	OrderBookDataRequestParamsTypeStep3 = OrderBookDataRequestParamsType("step3")
	OrderBookDataRequestParamsTypeStep4 = OrderBookDataRequestParamsType("step4")
	OrderBookDataRequestParamsTypeStep5 = OrderBookDataRequestParamsType("step5")
)

// OrderBookDataRequestParams represents Klines request data.
type OrderBookDataRequestParams struct {
	Symbol currency.Pair                  // Required; example LTCBTC,BTCUSDT
	Type   OrderBookDataRequestParamsType `json:"type"` // step0, step1, step2, step3, step4, step5 (combined depth 0-5); when step0, no depth is merged
}

// Orderbook stores the orderbook data
type Orderbook struct {
	ID         int64        `json:"id"`
	Timetstamp types.Time   `json:"ts"`
	Bids       [][2]float64 `json:"bids"`
	Asks       [][2]float64 `json:"asks"`
}

// Trade stores the trade data
type Trade struct {
	TradeID   float64    `json:"trade-id"`
	Price     float64    `json:"price"`
	Amount    float64    `json:"amount"`
	Direction string     `json:"direction"`
	Timestamp types.Time `json:"ts"`
}

// TradeHistory stores the trade history data
type TradeHistory struct {
	ID        int64      `json:"id"`
	Timestamp types.Time `json:"ts"`
	Trades    []Trade    `json:"data"`
}

// Detail stores the ticker detail data
type Detail struct {
	Amount    float64    `json:"amount"`
	Open      float64    `json:"open"`
	Close     float64    `json:"close"`
	High      float64    `json:"high"`
	Timestamp types.Time `json:"timestamp"`
	ID        int64      `json:"id"`
	Count     int        `json:"count"`
	Low       float64    `json:"low"`
	Volume    float64    `json:"vol"`
}

// Symbol stores the symbol data
type Symbol struct {
	BaseCurrency             string  `json:"base-currency"`
	QuoteCurrency            string  `json:"quote-currency"`
	PricePrecision           float64 `json:"price-precision"`
	AmountPrecision          float64 `json:"amount-precision"`
	SymbolPartition          string  `json:"symbol-partition"`
	Symbol                   string  `json:"symbol"`
	State                    string  `json:"state"`
	ValuePrecision           float64 `json:"value-precision"`
	MinOrderAmt              float64 `json:"min-order-amt"`
	MaxOrderAmt              float64 `json:"max-order-amt"`
	MinOrderValue            float64 `json:"min-order-value"`
	LimitOrderMinOrderAmt    float64 `json:"limit-order-min-order-amt"`
	LimitOrderMaxOrderAmt    float64 `json:"limit-order-max-order-amt"`
	SellMarketMinOrderAmt    float64 `json:"sell-market-min-order-amt"`
	SellMarketMaxOrderAmt    float64 `json:"sell-market-max-order-amt"`
	BuyMarketMaxOrderAmt     float64 `json:"buy-market-max-order-amt"`
	LeverageRatio            float64 `json:"leverage-ratio"`
	SuperMarginLeverageRatio float64 `json:"super-margin-leverage-ratio"`
	FundingLeverageRatio     float64 `json:"funding-leverage-ratio"`
}

// Account stores the account data
type Account struct {
	ID     int64  `json:"id"`
	Type   string `json:"type"`
	State  string `json:"state"`
	UserID int64  `json:"user-id"`
}

// AccountBalance stores the user all account balance
type AccountBalance struct {
	ID                    int64                  `json:"id"`
	Type                  string                 `json:"type"`
	State                 string                 `json:"state"`
	AccountBalanceDetails []AccountBalanceDetail `json:"list"`
}

// AccountBalanceDetail stores the user account balance
type AccountBalanceDetail struct {
	Currency currency.Code `json:"currency"`
	Type     string        `json:"type"`
	Balance  float64       `json:"balance,string"`
}

// AggregatedBalance stores balances of all the sub-account
type AggregatedBalance struct {
	Currency string  `json:"currency"`
	Balance  float64 `json:"balance,string"`
}

// CancelOrderBatch stores the cancel order batch data
type CancelOrderBatch struct {
	Success []string `json:"success"`
	Failed  []struct {
		OrderID      string `json:"order-id"`
		ErrorCode    string `json:"err-code"`
		ErrorMessage string `json:"err-msg"`
	} `json:"failed"`
}

// OrderInfo stores the order info
type OrderInfo struct {
	ID               int64      `json:"id"`
	Symbol           string     `json:"symbol"`
	AccountID        int64      `json:"account-id"`
	Amount           float64    `json:"amount,string"`
	Price            float64    `json:"price,string"`
	CreatedAt        types.Time `json:"created-at"`
	Type             string     `json:"type"`
	FieldAmount      float64    `json:"field-amount,string"`
	FieldCashAmount  float64    `json:"field-cash-amount,string"`
	FilledAmount     float64    `json:"filled-amount,string"`
	FilledCashAmount float64    `json:"filled-cash-amount,string"`
	FilledFees       float64    `json:"filled-fees,string"`
	FinishedAt       types.Time `json:"finished-at"`
	UserID           int64      `json:"user-id"`
	Source           string     `json:"source"`
	State            string     `json:"state"`
	CanceledAt       int64      `json:"canceled-at"`
	Exchange         string     `json:"exchange"`
	Batch            string     `json:"batch"`
}

// OrderMatchInfo stores the order match info
type OrderMatchInfo struct {
	ID           int        `json:"id"`
	OrderID      int        `json:"order-id"`
	MatchID      int        `json:"match-id"`
	Symbol       string     `json:"symbol"`
	Type         string     `json:"type"`
	Source       string     `json:"source"`
	Price        string     `json:"price"`
	FilledAmount string     `json:"filled-amount"`
	FilledFees   string     `json:"filled-fees"`
	CreatedAt    types.Time `json:"created-at"`
}

// MarginOrder stores the margin order info
type MarginOrder struct {
	Currency        string `json:"currency"`
	Symbol          string `json:"symbol"`
	AccruedAt       int64  `json:"accrued-at"`
	LoanAmount      string `json:"loan-amount"`
	LoanBalance     string `json:"loan-balance"`
	InterestBalance string `json:"interest-balance"`
	CreatedAt       int64  `json:"created-at"`
	InterestAmount  string `json:"interest-amount"`
	InterestRate    string `json:"interest-rate"`
	AccountID       int    `json:"account-id"`
	UserID          int    `json:"user-id"`
	UpdatedAt       int64  `json:"updated-at"`
	ID              int    `json:"id"`
	State           string `json:"state"`
}

// MarginAccountBalance stores the margin account balance info
type MarginAccountBalance struct {
	ID       int              `json:"id"`
	Type     string           `json:"type"`
	State    string           `json:"state"`
	Symbol   string           `json:"symbol"`
	FlPrice  string           `json:"fl-price"`
	FlType   string           `json:"fl-type"`
	RiskRate string           `json:"risk-rate"`
	List     []AccountBalance `json:"list"`
}

// SpotNewOrderRequestParams holds the params required to place an order
type SpotNewOrderRequestParams struct {
	AccountID int                           `json:"account-id,string"` // Account ID, obtained using the accounts method. Currency trades use the accountid of the ‘spot’ account; for loan asset transactions, please use the accountid of the ‘margin’ account.
	Amount    float64                       `json:"amount"`            // The limit price indicates the quantity of the order, the market price indicates how much to buy when the order is paid, and the market price indicates how much the coin is sold when the order is sold.
	Price     float64                       `json:"price"`             // Order price, market price does not use  this parameter
	Source    string                        `json:"source"`            // Order source, api: API call, margin-api: loan asset transaction
	Symbol    currency.Pair                 `json:"symbol"`            // The symbol to use; example btcusdt, bccbtc......
	Type      SpotNewOrderRequestParamsType `json:"type"`              // 订单类型, buy-market: 市价买, sell-market: 市价卖, buy-limit: 限价买, sell-limit: 限价卖
}

// DepositAddress stores the users deposit address info
type DepositAddress struct {
	UserID     int64  `json:"userId"`
	Currency   string `json:"currency"`
	Address    string `json:"address"`
	AddressTag string `json:"addressTag"`
	Chain      string `json:"chain"`
}

// ChainQuota stores the users currency chain quota
type ChainQuota struct {
	Chain                         string  `json:"chain"`
	MaxWithdrawAmount             float64 `json:"maxWithdrawAmt,string"`
	WithdrawQuotaPerDay           float64 `json:"withdrawQuotaPerDay,string"`
	RemainingWithdrawQuotaPerDay  float64 `json:"remainWithdrawQuotaPerDay,string"`
	WithdrawQuotaPerYear          float64 `json:"withdrawQuotaPerYear,string"`
	RemainingWithdrawQuotaPerYear float64 `json:"remainWithdrawQuotaPerYear,string"`
	WithdrawQuotaTotal            float64 `json:"withdrawQuotaTotal,string"`
	RemainingWithdrawQuotaTotal   float64 `json:"remainWithdrawQuotaTotal,string"`
}

// WithdrawQuota stores the users withdraw quotas
type WithdrawQuota struct {
	Currency string       `json:"currency"`
	Chains   []ChainQuota `json:"chains"`
}

// SpotNewOrderRequestParamsType order type
type SpotNewOrderRequestParamsType string

var (
	// SpotNewOrderRequestTypeBuyMarket buy market order
	SpotNewOrderRequestTypeBuyMarket = SpotNewOrderRequestParamsType("buy-market")

	// SpotNewOrderRequestTypeSellMarket sell market order
	SpotNewOrderRequestTypeSellMarket = SpotNewOrderRequestParamsType("sell-market")

	// SpotNewOrderRequestTypeBuyLimit buy limit order
	SpotNewOrderRequestTypeBuyLimit = SpotNewOrderRequestParamsType("buy-limit")

	// SpotNewOrderRequestTypeSellLimit sell limit order
	SpotNewOrderRequestTypeSellLimit = SpotNewOrderRequestParamsType("sell-limit")
)

//-----------

// KlinesRequestParams represents Klines request data.
type KlinesRequestParams struct {
	Symbol currency.Pair // Symbol to be used; example btcusdt, bccbtc......
	Period string        // Kline time interval; 1min, 5min, 15min......
	Size   uint64        // Size; [1-2000]
}

// wsSubReq is a request to subscribe to or unubscribe from a topic for public channels (private channels use generic wsReq)
type wsSubReq struct {
	ID    string `json:"id,omitempty"`
	Sub   string `json:"sub,omitempty"`
	Unsub string `json:"unsub,omitempty"`
}

// WsHeartBeat defines a heartbeat request
type WsHeartBeat struct {
	ClientNonce int64 `json:"ping"`
}

// WsDepth defines market depth websocket response
type WsDepth struct {
	Channel   string     `json:"ch"`
	Timestamp types.Time `json:"ts"`
	Tick      struct {
		Bids      [][]any    `json:"bids"`
		Asks      [][]any    `json:"asks"`
		Timestamp types.Time `json:"ts"`
		Version   int64      `json:"version"`
	} `json:"tick"`
}

// WsKline defines market kline websocket response
type WsKline struct {
	Channel   string     `json:"ch"`
	Timestamp types.Time `json:"ts"`
	Tick      struct {
		ID     int64   `json:"id"`
		Open   float64 `json:"open"`
		Close  float64 `json:"close"`
		Low    float64 `json:"low"`
		High   float64 `json:"high"`
		Amount float64 `json:"amount"`
		Volume float64 `json:"vol"`
		Count  int64   `json:"count"`
	} `json:"tick"`
}

// WsTick stores websocket ticker data
type WsTick struct {
	Channel   string     `json:"ch"`
	Rep       string     `json:"rep"`
	Timestamp types.Time `json:"ts"`
	Tick      struct {
		Amount    float64    `json:"amount"`
		Close     float64    `json:"close"`
		Count     float64    `json:"count"`
		High      float64    `json:"high"`
		ID        float64    `json:"id"`
		Low       float64    `json:"low"`
		Open      float64    `json:"open"`
		Timestamp types.Time `json:"ts"`
		Volume    float64    `json:"vol"`
	} `json:"tick"`
}

// WsTrade defines market trade websocket response
type WsTrade struct {
	Channel   string     `json:"ch"`
	Timestamp types.Time `json:"ts"`
	Tick      struct {
		ID        int64      `json:"id"`
		Timestamp types.Time `json:"ts"`
		Data      []struct {
			Amount    float64    `json:"amount"`
			Timestamp types.Time `json:"ts"`
			TradeID   float64    `json:"tradeId"`
			Price     float64    `json:"price"`
			Direction string     `json:"direction"`
		} `json:"data"`
	}
}

// wsReq contains authentication login fields
type wsReq struct {
	Action  string `json:"action"`
	Channel string `json:"ch"`
	Params  any    `json:"params"`
}

// wsAuthReq contains authentication login fields
type wsAuthReq struct {
	AuthType         string `json:"authType"`
	AccessKey        string `json:"accessKey"`
	SignatureMethod  string `json:"signatureMethod"`
	SignatureVersion string `json:"signatureVersion"`
	Timestamp        string `json:"timestamp"`
	Signature        string `json:"signature"`
}

type wsAccountUpdateMsg struct {
	Data WsAccountUpdate `json:"data"`
}

// WsAccountUpdate contains account updates to balances
type WsAccountUpdate struct {
	Currency    string     `json:"currency"`
	AccountID   int64      `json:"accountId"`
	Balance     float64    `json:"balance,string"`
	Available   float64    `json:"available,string"`
	ChangeType  string     `json:"changeType"`
	AccountType string     `json:"accountType"`
	ChangeTime  types.Time `json:"changeTime"`
	SeqNum      int64      `json:"seqNum"`
}

type wsOrderUpdateMsg struct {
	Data WsOrderUpdate `json:"data"`
}

// WsOrderUpdate contains updates to orders
type WsOrderUpdate struct {
	EventType       string     `json:"eventType"`
	Symbol          string     `json:"symbol"`
	AccountID       int64      `json:"accountId"`
	OrderID         int64      `json:"orderId"`
	TradeID         int64      `json:"tradeId"`
	ClientOrderID   string     `json:"clientOrderId"`
	Source          string     `json:"orderSource"`
	Price           float64    `json:"orderPrice,string"`
	Size            float64    `json:"orderSize,string"`
	Value           float64    `json:"orderValue,string"`
	OrderType       string     `json:"type"`
	TradePrice      float64    `json:"tradePrice,string"`
	TradeVolume     float64    `json:"tradeVolume,string"`
	RemainingAmount float64    `json:"remainAmt,string"`
	ExecutedAmount  float64    `json:"execAmt,string"`
	IsTaker         bool       `json:"aggressor"`
	Side            order.Side `json:"orderSide"`
	OrderStatus     string     `json:"orderStatus"`
	LastActTime     types.Time `json:"lastActTime"`
	CreateTime      types.Time `json:"orderCreateTime"`
	TradeTime       types.Time `json:"tradeTime"`
	ErrCode         int64      `json:"errCode"`
	ErrMessage      string     `json:"errMessage"`
}

type wsTradeUpdateMsg struct {
	Data WsTradeUpdate `json:"data"`
}

// WsTradeUpdate contains trade updates to orders
type WsTradeUpdate struct {
	EventType       string     `json:"eventType"`
	Symbol          string     `json:"symbol"`
	OrderID         int64      `json:"orderId"`
	TradePrice      float64    `json:"tradePrice,string"`
	TradeVolume     float64    `json:"tradeVolume,string"`
	Side            order.Side `json:"orderSide"`
	OrderType       string     `json:"orderType"`
	IsTaker         bool       `json:"aggressor"`
	TradeID         int64      `json:"tradeId"`
	TradeTime       types.Time `json:"tradeTime"`
	TransactFee     float64    `json:"transactFee,string"`
	FeeCurrency     string     `json:"feeCurrency"`
	FeeDeduct       string     `json:"feeDeduct"`
	FeeDeductType   string     `json:"feeDeductType"`
	AccountID       int64      `json:"accountId"`
	Source          string     `json:"orderSource"`
	OrderPrice      float64    `json:"orderPrice,string"`
	OrderSize       float64    `json:"orderSize,string"`
	Value           float64    `json:"orderValue,string"`
	ClientOrderID   string     `json:"clientOrderId"`
	StopPrice       string     `json:"stopPrice"`
	Operator        string     `json:"operator"`
	OrderCreateTime types.Time `json:"orderCreateTime"`
	OrderStatus     string     `json:"orderStatus"`
}

// OrderVars stores side, status and type for any order/trade
type OrderVars struct {
	Side        order.Side
	Status      order.Status
	OrderType   order.Type
	TimeInForce order.TimeInForce
	Fee         float64
}

// Variables below are used to check api requests being sent out

var (
	validPeriods = []string{"5min", "15min", "30min", "60min", "4hour", "1day"}

	validBasisPriceTypes = []string{"open", "close", "high", "low", "average"}

	validAmountType = map[string]int64{
		"cont":           1,
		"cryptocurrency": 2,
	}

	validTransferType = []string{
		"master_to_sub", "sub_to_master",
	}

	validTradeTypes = map[string]int64{
		"filled": 0,
		"closed": 5,
		"open":   6,
	}

	validOrderType = map[string]int64{
		"quotation":         1,
		"cancelledOrder":    2,
		"forcedLiquidation": 3,
		"deliveryOrder":     4,
	}

	validOrderTypes = []string{
		"limit", "opponent", "lightning", "optimal_5", "optimal_10", "optimal_20",
		"fok", "ioc", "opponent_ioc", "lightning_ioc", "optimal_5_ioc",
		"optimal_10_ioc", "optimal_20_ioc", "opponent_fok", "optimal_20_fok",
	}

	validTriggerType = map[string]string{
		"greaterOrEqual": "ge",
		"smallerOrEqual": "le",
	}

	validOrderPriceType = []string{
		"limit", "optimal_5", "optimal_10", "optimal_20",
	}

	validLightningOrderPriceType = []string{
		"lightning", "lightning_fok", "lightning_ioc",
	}

	validTradeType = map[string]int64{
		"all":            0,
		"openLong":       1,
		"openShort":      2,
		"closeShort":     3,
		"closeLong":      4,
		"liquidateLong":  5,
		"liquidateShort": 6,
	}

	validFuturesTradeType = map[string]int64{
		"all":            0,
		"openLong":       1,
		"openShort":      2,
		"closeShort":     3,
		"closeLong":      4,
		"liquidateLong":  5,
		"liquidateShort": 6,
		"deliveryLong":   7,
		"deliveryShort":  8,
		"reduceLong":     11,
		"reduceShort":    12,
	}

	contractExpiryNames = map[string]string{
		"this_week":    "CW",
		"next_week":    "NW",
		"quarter":      "CQ",
		"next_quarter": "NQ",
	}

	validContractExpiryCodes = []string{"CW", "NW", "CQ", "NQ"}

	validFuturesPeriods = []string{
		"1min", "5min", "15min", "30min", "60min", "1hour", "4hour", "1day",
	}

	validFuturesOrderPriceTypes = []string{
		"limit", "opponent", "lightning", "optimal_5", "optimal_10",
		"optimal_20", "fok", "ioc", "opponent_ioc", "lightning_ioc",
		"optimal_5_ioc", "optimal_10_ioc", "optimal_20_ioc", "opponent_fok",
		"lightning_fok", "optimal_5_fok", "optimal_10_fok", "optimal_20_fok",
	}

	validFuturesRecordTypes = map[string]string{
		"closeLong":                   "3",
		"closeShort":                  "4",
		"openOpenPositionsTakerFees":  "5",
		"openPositionsMakerFees":      "6",
		"closePositionsTakerFees":     "7",
		"closePositionsMakerFees":     "8",
		"closeLongDelivery":           "9",
		"closeShortDelivery":          "10",
		"deliveryFee":                 "11",
		"longLiquidationClose":        "12",
		"shortLiquidationClose":       "13",
		"transferFromSpotToContracts": "14",
		"transferFromContractsToSpot": "15",
		"settleUnrealizedLongPNL":     "16",
		"settleUnrealizedShortPNL":    "17",
		"clawback":                    "19",
		"system":                      "26",
		"activityPrizeRewards":        "28",
		"rebate":                      "29",
		"transferToSub":               "34",
		"transferFromSub":             "35",
		"transferToMaster":            "36",
		"transferFromMaster":          "37",
	}

	validOffsetTypes = []string{
		"open", "close",
	}

	validOPTypes = []string{
		"lightning", "lightning_fok", "lightning_ioc",
	}

	validFuturesReqType = map[string]int64{
		"all":            1,
		"finishedStatus": 2,
	}

	validFuturesOrderTypes = map[string]int64{
		"limit":        1,
		"opponent":     3,
		"lightning":    4,
		"triggerOrder": 5,
		"postOnly":     6,
		"optimal_5":    7,
		"optimal_10":   8,
		"optimal_20":   9,
		"fok":          10,
		"ioc":          11,
	}

	validOrderStatus = map[order.Status]int64{
		order.AnyStatus:          0,
		order.Active:             3,
		order.PartiallyFilled:    4,
		order.PartiallyCancelled: 5,
		order.Filled:             6,
		order.Cancelled:          7,
	}

	validStatusTypes = map[string]int64{
		"all":       0,
		"success":   4,
		"failed":    5,
		"cancelled": 6,
	}
)

// WithdrawalHistory holds withdrawal history data
type WithdrawalHistory struct {
	Status string           `json:"status"`
	Data   []WithdrawalData `json:"data"`
}

// WithdrawalData contains details of a withdrawal
type WithdrawalData struct {
	ID              int64         `json:"id"`
	Type            string        `json:"type"`
	Currency        currency.Code `json:"currency"`
	TransactionHash string        `json:"tx-hash"`
	Chain           string        `json:"chain"`
	Amount          float64       `json:"amount"`
	SubType         string        `json:"sub-type"`
	Address         string        `json:"address"`
	AddressTag      string        `json:"address-tag"`
	FromAddressTag  string        `json:"from-addr-tag"`
	Fee             float64       `json:"fee"`
	State           string        `json:"state"`
	ErrorCode       string        `json:"error-code"`
	ErrorMessage    string        `json:"error-message"`
	CreatedAt       types.Time    `json:"created-at"`
	UpdatedAt       types.Time    `json:"updated-at"`
}
