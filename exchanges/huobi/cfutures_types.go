package huobi

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// WsSwapReqKline stores req kline data for swap websocket
type WsSwapReqKline struct {
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

// WsSwapReqTradeDetail stores requested trade detail data for swap websocket
type WsSwapReqTradeDetail struct {
	Rep       string     `json:"rep"`
	ID        int64      `json:"id"`
	Timestamp types.Time `json:"ts"`
	Data      []struct {
		ID        int64      `json:"id"`
		Price     float64    `json:"price"`
		Amount    float64    `json:"amount"`
		Direction string     `json:"direction"`
		Timestamp types.Time `json:"ts"`
	} `json:"data"`
}

// SwapWsSubPremiumKline stores subscribed premium kline data for futures websocket
type SwapWsSubPremiumKline struct {
	Channel   string     `json:"ch"`
	Timestamp types.Time `json:"ts"`
	Tick      struct {
		ID     int64   `json:"id"`
		Volume float64 `json:"vol"`
		Count  float64 `json:"count"`
		Open   float64 `json:"open"`
		Close  float64 `json:"close"`
		Low    float64 `json:"low"`
		High   float64 `json:"high"`
		Amount float64 `json:"amount"`
	} `json:"tick"`
}

// SwapWsReqPremiumKline stores requested premium kline data for futures websocket
type SwapWsReqPremiumKline struct {
	Rep       string     `json:"rep"`
	ID        string     `json:"id"`
	WsID      int64      `json:"wsid"`
	Timestamp types.Time `json:"ts"`
	Data      []struct {
		Volume float64 `json:"vol"`
		Count  float64 `json:"count"`
		ID     int64   `json:"id"`
		Open   float64 `json:"open"`
		Close  float64 `json:"close"`
		Low    float64 `json:"low"`
		High   float64 `json:"high"`
		Amount float64 `json:"amount"`
	} `json:"data"`
}

// SwapWsSubEstimatedFunding stores estimated funding rate data for swap websocket
type SwapWsSubEstimatedFunding struct {
	Channel   string     `json:"ch"`
	Timestamp types.Time `json:"ts"`
	Tick      struct {
		ID     int64   `json:"id"`
		Volume float64 `json:"vol,string"`
		Count  float64 `json:"count,string"`
		Open   float64 `json:"open,string"`
		Close  float64 `json:"close,string"`
		Low    float64 `json:"low,string"`
		High   float64 `json:"high,string"`
		Amount float64 `json:"amount,string"`
	} `json:"tick"`
}

// SwapWsReqEstimatedFunding stores requested estimated funding data for swap websocket
type SwapWsReqEstimatedFunding struct {
	Rep       string     `json:"rep"`
	ID        string     `json:"id"`
	WsID      int64      `json:"wsid"`
	Timestamp types.Time `json:"ts"`
	Data      []struct {
		Volume float64 `json:"vol,string"`
		Count  float64 `json:"count,string"`
		ID     int64   `json:"id"`
		Open   float64 `json:"open,string"`
		Close  float64 `json:"close,string"`
		Low    float64 `json:"low,string"`
		High   float64 `json:"high,string"`
		Amount float64 `json:"amount,string"`
	}
}

// SwapWsSubBasisData stores subscribed basis data for swap websocket
type SwapWsSubBasisData struct {
	Channel   string     `json:"ch"`
	Timestamp types.Time `json:"ts"`
	Tick      []struct {
		ID            int64   `json:"id"`
		ContractPrice float64 `json:"contract_price,string"`
		IndexPrice    float64 `json:"index_price,string"`
		Basis         float64 `json:"basis,string"`
		BasisRate     float64 `json:"basis_rate,string"`
	} `json:"tick"`
}

// SwapWsReqBasisData stores requested basis data for swap websocket
type SwapWsReqBasisData struct {
	Rep       string     `json:"rep"`
	ID        string     `json:"id"`
	WsID      int64      `json:"wsid"`
	Timestamp types.Time `json:"ts"`
	Data      []struct {
		ID            int64   `json:"id"`
		ContractPrice float64 `json:"contract_price"`
		IndexPrice    float64 `json:"index_price"`
		Basis         float64 `json:"basis"`
		BasisRate     float64 `json:"basis_rate"`
	}
}

// SwapWsSubOrderData stores subscribed order data for swap websocket
type SwapWsSubOrderData struct {
	Operation      string     `json:"op"`
	Topic          string     `json:"topic"`
	UID            string     `json:"uid"`
	Timestamp      types.Time `json:"ts"`
	Symbol         string     `json:"symbol"`
	ContractCode   string     `json:"contract_code"`
	Volume         float64    `json:"volume"`
	Price          float64    `json:"price"`
	OrderPriceType string     `json:"order_price_type"`
	Direction      string     `json:"direction"`
	Offset         string     `json:"offset"`
	Status         int64      `json:"status"`
	LeverateRate   float64    `json:"lever_rate"`
	OrderID        int64      `json:"order_id"`
	OrderIDString  string     `json:"order_id_str"`
	ClientOrderID  int64      `json:"client_order_id"`
	OrderSource    string     `json:"order_source"`
	OrderType      int64      `json:"order_type"`
	CreatedAt      int64      `json:"created_at"`
	CanceledAt     int64      `json:"canceled_at"`
	TradeVolume    float64    `json:"trade_volume"`
	TradeTurnover  float64    `json:"trade_turnover"`
	Fee            float64    `json:"fee"`
	FeeAsset       string     `json:"fee_asset"`
	TradeAvgPrice  float64    `json:"trade_avg_price"`
	MarginFrozen   float64    `json:"margin_frozen"`
	Profit         float64    `json:"profit"`
	Trade          []struct {
		ID            string  `json:"id"`
		TradeID       int64   `json:"trade_id"`
		TradeVolume   float64 `json:"trade_volume"`
		TradePrice    float64 `json:"trade_price"`
		TradeFee      float64 `json:"trade_fee"`
		TradeTurnover float64 `json:"trade_turnover"`
		CreatedAt     int64   `json:"created_at"`
		FeeAsset      string  `json:"fee_asset"`
		Role          string  `json:"role"`
	} `json:"trade"`
	LiquidationType string `json:"liquidation_type"`
}

// SwapWsSubMatchOrderData stores subscribed match order data for swap websocket
type SwapWsSubMatchOrderData struct {
	Operation     string     `json:"op"`
	Topic         string     `json:"topic"`
	UID           string     `json:"uid"`
	Timestamp     types.Time `json:"ts"`
	Symbol        string     `json:"symbol"`
	ContractCode  string     `json:"contract_code"`
	Status        int64      `json:"status"`
	OrderID       int64      `json:"order_id"`
	OrderIDString string     `json:"order_id_str"`
	ClientOrderID int64      `json:"client_order_id"`
	OrderType     string     `json:"order_type"`
	TradeVolume   int64      `json:"trade_volume"`
	Volume        float64    `json:"volume"`
	Trade         []struct {
		ID            string  `json:"id"`
		TradeID       int64   `json:"trade_id"`
		TradeVolume   float64 `json:"trade_volume"`
		TradePrice    float64 `json:"trade_price"`
		TradeTurnover float64 `json:"trade_turnover"`
		CreatedAt     int64   `json:"created_at"`
		Role          string  `json:"role"`
	} `json:"trade"`
}

// SwapWsSubEquityData stores subscribed account data for swap account equity updates through websocket
type SwapWsSubEquityData struct {
	Operation string     `json:"op"`
	Topic     string     `json:"topic"`
	Timestamp types.Time `json:"ts"`
	UID       string     `json:"uid"`
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

// SwapWsSubPositionUpdates stores subscribed position updates data for swap websocket
type SwapWsSubPositionUpdates struct {
	Operation string     `json:"op"`
	Topic     string     `json:"topic"`
	UID       string     `json:"uid"`
	Timestamp types.Time `json:"ts"`
	Event     string     `json:"event"`
	Data      []struct {
		Symbol         string  `json:"symbol"`
		ContractCode   string  `json:"contract_code"`
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
	}
}

// SwapWsSubLiquidationOrders stores subscribed liquidation orders data for swap futures
type SwapWsSubLiquidationOrders struct {
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

// SwapWsSubFundingData stores funding rate data for swap websocket
type SwapWsSubFundingData struct {
	Operation   string     `json:"op"`
	Topic       string     `json:"topic"`
	Timestamp   types.Time `json:"ts"`
	FundingData []struct {
		Symbol         string     `json:"symbol"`
		ContractCode   string     `json:"contract_code"`
		FeeAsset       string     `json:"fee_asset"`
		FundingTime    types.Time `json:"funding_time"`
		FundingRate    float64    `json:"funding_rate,string"`
		EstimatedRate  float64    `json:"estimated_rate,string"`
		SettlementTime types.Time `json:"settlement_time"`
	} `json:"data"`
}

// SwapWsSubContractInfo stores funding rate data for swap websocket
type SwapWsSubContractInfo struct {
	Operation    string     `json:"op"`
	Topic        string     `json:"topic"`
	Timestamp    types.Time `json:"ts"`
	Event        string     `json:"event"`
	ContractData []struct {
		Symbol         string  `json:"symbol"`
		ContractCode   string  `json:"contract_code"`
		ContractSize   float64 `json:"contract_size"`
		PriceTick      float64 `json:"price_tick"`
		SettlementDate string  `json:"settlement_date"`
		CreateDate     string  `json:"create_date"`
		ContractStatus int64   `json:"contract_status"`
	} `json:"data"`
}

// SwapWsSubTriggerOrderUpdates stores subscribed trigger order updates data for swap websocket
type SwapWsSubTriggerOrderUpdates struct {
	Operation string `json:"op"`
	Topic     string `json:"topic"`
	UID       string `json:"uid"`
	Event     string `json:"event"`
	Data      []struct {
		Symbol          string  `json:"symbol"`
		ContractCode    string  `json:"contract_code"`
		ContractType    string  `json:"contract_type"`
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

// SwapIndexPriceData gets price of a perpetual swap
type SwapIndexPriceData struct {
	Data []struct {
		ContractCode   string     `json:"contract_code"`
		IndexPrice     float64    `json:"index_price"`
		IndexTimestamp types.Time `json:"index_ts"`
	} `json:"data"`
}

// SwapPriceLimitsData gets price restrictions on perpetual swaps
type SwapPriceLimitsData struct {
	Data []struct {
		Symbol       string  `json:"symbol"`
		HighLimit    float64 `json:"high_limit"`
		LowLimit     float64 `json:"low_limit"`
		ContractCode string  `json:"contract_code"`
	} `json:"data"`
}

// SwapOpenInterestData stores open interest data for swaps
type SwapOpenInterestData struct {
	Data []struct {
		Symbol       string  `json:"symbol"`
		Volume       float64 `json:"volume"`
		Amount       float64 `json:"amount"`
		ContractCode string  `json:"contract_code"`
	} `json:"data"`
}

// SwapMarketDepthData stores market depth data
type SwapMarketDepthData struct {
	Tick struct {
		Asks      [][]float64 `json:"asks"`
		Bids      [][]float64 `json:"bids"`
		Channel   string      `json:"ch"`
		ID        int64       `json:"id"`
		MRID      int64       `json:"mrid"`
		Timestamp types.Time  `json:"ts"`
		Version   int64       `json:"version"`
	} `json:"tick"`
}

// SwapKlineData stores kline data for perpetual swaps
type SwapKlineData struct {
	Data []struct {
		Volume      float64    `json:"vol"`
		Close       float64    `json:"close"`
		Count       float64    `json:"count"`
		High        float64    `json:"high"`
		IDTimestamp types.Time `json:"id"`
		Low         float64    `json:"low"`
		Open        float64    `json:"open"`
		Amount      float64    `json:"amount"`
	} `json:"data"`
}

// MarketOverviewData stores market overview data
type MarketOverviewData struct {
	Channel string `json:"ch"`
	Tick    struct {
		Vol       float64    `json:"vol,string"`
		Ask       []float64  `json:"ask"`
		Bid       []float64  `json:"bid"`
		Close     float64    `json:"close,string"`
		Count     float64    `json:"count"`
		High      float64    `json:"high,string"`
		ID        int64      `json:"id"`
		Low       float64    `json:"low,string"`
		Open      float64    `json:"open,string"`
		Timestamp types.Time `json:"ts"`
		Amount    float64    `json:"amount,string"`
	} `json:"tick"`
}

// LastTradeData stores last trade's data of a contract
type LastTradeData struct {
	Ch   string `json:"ch"`
	Tick struct {
		Data []struct {
			Amount    float64    `json:"amount,string"`
			Direction string     `json:"direction"`
			ID        int64      `json:"id"`
			Price     float64    `json:"price,string"`
			Timestamp types.Time `json:"ts"`
		} `json:"data"`
	} `json:"tick"`
}

// BatchTradesData stores batch trades for a given swap contract
type BatchTradesData struct {
	ID        int64                      `json:"id"`
	Timestamp types.Time                 `json:"ts"`
	Data      []CoinMarginedFuturesTrade `json:"data"`
}

// CoinMarginedFuturesTrade holds coinmarginedfutures trade data
type CoinMarginedFuturesTrade struct {
	Amount    float64    `json:"amount"`
	Direction string     `json:"direction"`
	ID        int64      `json:"id"`
	Price     float64    `json:"price"`
	Timestamp types.Time `json:"ts"`
}

// InsuranceAndClawbackData stores insurance fund's and clawback rate's data
type InsuranceAndClawbackData struct {
	Timestamp types.Time `json:"timestamp"`
	Data      []struct {
		ContractCode      string  `json:"contract_code"`
		InsuranceFund     float64 `json:"insurance_fund"`
		EstimatedClawback float64 `json:"estimated_clawback"`
	} `json:"data"`
}

// HistoricalInsuranceFundBalance stores insurance fund balance data in the past
type HistoricalInsuranceFundBalance struct {
	Data struct {
		Symbol       string `json:"symbol"`
		ContractCode string `json:"contract_code"`
		Tick         []struct {
			InsuranceFund float64    `json:"insurance_fund"`
			Timestamp     types.Time `json:"ts"`
		} `json:"tick"`
		TotalPage   int64 `json:"total_page"`
		TotalSize   int64 `json:"total_size"`
		CurrentPage int64 `json:"current_page"`
	} `json:"data"`
}

// TieredAdjustmentFactorData stores tiered adjustment factor data
type TieredAdjustmentFactorData struct {
	Data []struct {
		Symbol       string `json:"symbol"`
		ContractCode string `json:"contract_code"`
		List         []struct {
			LeverRate float64 `json:"lever_rate"`
			Ladders   []struct {
				Ladder       float64 `json:"ladder"`
				MinSize      float64 `json:"min_size"`
				MaxSize      float64 `json:"max_size"`
				AdjustFactor float64 `json:"adjust_factor"`
			} `json:"ladders"`
		} `json:"list"`
	} `json:"data"`
}

// OpenInterestData stores open interest data
type OpenInterestData struct {
	Data struct {
		Symbol       string `json:"symbol"`
		ContractCode string `json:"contract_code"`
		Tick         []struct {
			Volume     float64    `json:"volume"`
			AmountType float64    `json:"amountType"`
			Timestamp  types.Time `json:"ts"`
		} `json:"tick"`
	} `json:"data"`
}

// SystemStatusData stores information on system status
type SystemStatusData struct {
	Data []struct {
		Symbol            string  `json:"symbol"`
		ContractCode      string  `json:"contract_code"`
		Open              float64 `json:"open"`
		Close             float64 `json:"close"`
		Cancel            float64 `json:"cancel"`
		TransferIn        float64 `json:"transfer_in"`
		TransferOut       float64 `json:"transfer_out"`
		MasterTransferSub float64 `json:"master_transfer_sub"`
		SubTransferMaster float64 `json:"sub_transfer_master"`
	} `json:"data"`
}

// TraderSentimentIndexAccountData stores trader sentiment index data
type TraderSentimentIndexAccountData struct {
	Data struct {
		Symbol       string `json:"symbol"`
		ContractCode string `json:"contract_code"`
		List         []struct {
			BuyRatio    float64    `json:"buy_ratio"`
			SellRatio   float64    `json:"sell_ratio"`
			LockedRatio float64    `json:"locked_ratio"`
			Timestamp   types.Time `json:"ts"`
		} `json:"list"`
	} `json:"data"`
}

// TraderSentimentIndexPositionData stores trader sentiment index data
type TraderSentimentIndexPositionData struct {
	Data struct {
		Symbol       string `json:"symbol"`
		ContractCode string `json:"contract_code"`
		List         []struct {
			BuyRatio  float64    `json:"buy_ratio"`
			SellRatio float64    `json:"sell_ratio"`
			Timestamp types.Time `json:"ts"`
		} `json:"list"`
	} `json:"data"`
}

// LiquidationOrdersData stores data of liquidation orders
type LiquidationOrdersData struct {
	Data []struct {
		QueryID      int64      `json:"query_id"`
		ContractCode string     `json:"contract_code"`
		Symbol       string     `json:"symbol"`
		Direction    string     `json:"direction"`
		Offset       string     `json:"offset"`
		Volume       float64    `json:"volume"`
		Price        float64    `json:"price"`
		CreatedAt    types.Time `json:"created_at"`
		Amount       float64    `json:"amount"`
	} `json:"data"`
}

// SwapFundingRatesResponse holds funding rates and data response
type SwapFundingRatesResponse struct {
	Response
	Data []FundingRatesData `json:"data"`
}

// FundingRatesData stores funding rates data
type FundingRatesData struct {
	EstimatedRate   float64    `json:"estimated_rate,string"`
	FundingRate     float64    `json:"funding_rate,string"`
	ContractCode    string     `json:"contractCode"`
	Symbol          string     `json:"symbol"`
	FeeAsset        string     `json:"fee_asset"`
	FundingTime     types.Time `json:"fundingTime"`
	NextFundingTime types.Time `json:"next_funding_time"`
}

// HistoricalFundingRateData stores historical funding rates for perpetuals
type HistoricalFundingRateData struct {
	Data struct {
		TotalPage   int64                `json:"total_page"`
		CurrentPage int64                `json:"current_page"`
		TotalSize   int64                `json:"total_size"`
		Data        []HistoricalRateData `json:"data"`
	}
}

// HistoricalRateData stores historical rates data
type HistoricalRateData struct {
	FundingRate     float64    `json:"funding_rate,string"`
	RealizedRate    float64    `json:"realized_rate,string"`
	FundingTime     types.Time `json:"fundingTime"`
	ContractCode    string     `json:"contract_code"`
	Symbol          string     `json:"symbol"`
	FeeAsset        string     `json:"fee_asset"`
	AvgPremiumIndex float64    `json:"avg_premium_index,string"`
}

// PremiumIndexKlineData stores kline data for premium
type PremiumIndexKlineData struct {
	Channel string `json:"ch"`
	Data    []struct {
		Volume float64 `json:"vol,string"`
		Close  float64 `json:"close,string"`
		Count  float64 `json:"count,string"`
		High   float64 `json:"high,string"`
		ID     int64   `json:"id"`
		Low    float64 `json:"low,string"`
		Open   float64 `json:"open,string"`
		Amount float64 `json:"amount,string"`
	} `json:"data"`
	Timestamp types.Time `json:"ts"`
}

// EstimatedFundingRateData stores estimated funding rate data
type EstimatedFundingRateData struct {
	Channel string `json:"ch"`
	Data    []struct {
		Volume float64 `json:"vol"`
		Close  float64 `json:"close"`
		Count  float64 `json:"count"`
		High   float64 `json:"high"`
		ID     int64   `json:"id"`
		Low    float64 `json:"low"`
		Open   float64 `json:"open"`
		Amount float64 `json:"amount"`
	} `json:"data"`
	Timestamp types.Time `json:"ts"`
}

// BasisData stores basis data for swaps
type BasisData struct {
	Channel string `json:"ch"`
	Data    []struct {
		Basis         string `json:"basis"`
		BasisRate     string `json:"basis_rate"`
		ContractPrice string `json:"contract_price"`
		ID            int64  `json:"id"`
		IndexPrice    string `json:"index_price"`
	} `json:"data"`
	Timestamp types.Time `json:"ts"`
}

// SwapAccountInformation stores swap account information
type SwapAccountInformation struct {
	Data []struct {
		Symbol            currency.Code `json:"symbol"`
		ContractCode      string        `json:"contract_code"`
		MarginBalance     float64       `json:"margin_balance"`
		MarginPosition    float64       `json:"margin_position"`
		MarginFrozen      float64       `json:"margin_frozen"`
		MarginAvailable   float64       `json:"margin_available"`
		ProfitReal        float64       `json:"profit_real"`
		ProfitUnreal      float64       `json:"profit_unreal"`
		WithdrawAvailable float64       `json:"withdraw_available"`
		RiskRate          float64       `json:"risk_rate"`
		LiquidationPrice  float64       `json:"liquidation_price"`
		AdjustFactor      float64       `json:"adjust_factor"`
		LeverageRate      float64       `json:"lever_rate"`
		MarginStatic      float64       `json:"margin_static"`
	} `json:"data"`
}

// SwapPositionInfo stores user's swap positions' info
type SwapPositionInfo struct {
	Data []struct {
		Symbol         string  `json:"symbol"`
		ContractCode   string  `json:"contract_code"`
		Volume         float64 `json:"volume"`
		Available      float64 `json:"available"`
		Frozen         float64 `json:"frozen"`
		CostOpen       float64 `json:"cost_open"`
		CostHold       float64 `json:"cost_hold"`
		ProfitUnreal   float64 `json:"profit_unreal"`
		ProfitRate     float64 `json:"profit_rate"`
		Profit         float64 `json:"profit"`
		PositionMargin float64 `json:"position_margin"`
		LeverRate      float64 `json:"lever_rate"`
		Direction      string  `json:"direction"`
		LastPrice      float64 `json:"last_price"`
	} `json:"data"`
}

// SwapAssetsAndPositionsData stores positions and assets data for swaps
type SwapAssetsAndPositionsData struct {
	Timestamp types.Time `json:"ts"`
	Data      []struct {
		Symbol            string  `json:"symbol"`
		ContractCode      string  `json:"contract_code"`
		MarginBalance     float64 `json:"margin_balance"`
		MarginPosition    float64 `json:"margin_position"`
		MarginFrozen      float64 `json:"margin_frozen"`
		MarginAvailable   float64 `json:"margin_available"`
		ProfitReal        float64 `json:"profit_real"`
		ProfitUnreal      float64 `json:"profit_unreal"`
		WithdrawAvailable float64 `json:"withdraw_available"`
		RiskRate          float64 `json:"risk_rate"`
		LiquidationPrice  float64 `json:"liquidation_price"`
		AdjustFactor      float64 `json:"adjust_factor"`
		LeverageRate      float64 `json:"lever_rate"`
		MarginStatic      float64 `json:"margin_static"`
		Positions         []struct {
			Symbol         string  `json:"symbol"`
			ContractCode   string  `json:"contract_code"`
			Volume         float64 `json:"volume"`
			Available      float64 `json:"available"`
			Frozen         float64 `json:"frozen"`
			CostOpen       float64 `json:"cost_open"`
			CostHold       float64 `json:"cost_hold"`
			ProfitUnreal   float64 `json:"profit_unreal"`
			ProfitRate     float64 `json:"profit_rate"`
			Profit         float64 `json:"profit"`
			PositionMargin float64 `json:"position_margin"`
			LeverRate      float64 `json:"lever_rate"`
			Direction      string  `json:"direction"`
			LastPrice      float64 `json:"last_price"`
		} `json:"positions"`
	} `json:"data"`
}

// SubAccountsAssetData stores asset data for all subaccounts
type SubAccountsAssetData struct {
	Timestamp types.Time `json:"ts"`
	Data      []struct {
		SubUID int64 `json:"sub_uid"`
		List   []struct {
			Symbol           string  `json:"symbol"`
			ContractCode     string  `json:"contract_code"`
			MarginBalance    int64   `json:"margin_balance"`
			LiquidationPrice float64 `json:"liquidation_price"`
			RiskRate         float64 `json:"risk_rate"`
		} `json:"list"`
	} `json:"data"`
}

// SingleSubAccountAssetsInfo stores asset info for a single subaccount
type SingleSubAccountAssetsInfo struct {
	Timestamp types.Time `json:"ts"`
	Data      []struct {
		Symbol            currency.Code `json:"symbol"`
		ContractCode      string        `json:"contract_code"`
		MarginBalance     float64       `json:"margin_balance"`
		MarginPosition    float64       `json:"margin_position"`
		MarginFrozen      float64       `json:"margin_frozen"`
		MarginAvailable   float64       `json:"margin_available"`
		ProfitReal        float64       `json:"profit_real"`
		ProfitUnreal      float64       `json:"profit_unreal"`
		WithdrawAvailable float64       `json:"withdraw_available"`
		RiskRate          float64       `json:"risk_rate"`
		LiquidationPrice  float64       `json:"liquidation_price"`
		AdjustFactor      float64       `json:"adjust_factor"`
		LeverageRate      float64       `json:"lever_rate"`
		MarginStatic      float64       `json:"margin_static"`
	} `json:"data"`
}

// SingleSubAccountPositionsInfo stores single subaccount's positions data
type SingleSubAccountPositionsInfo struct {
	Timestamp types.Time `json:"ts"`
	Data      []struct {
		Symbol         string  `json:"symbol"`
		ContractCode   string  `json:"contract_code"`
		Volume         float64 `json:"volume"`
		Available      float64 `json:"available"`
		Frozen         float64 `json:"frozen"`
		CostOpen       float64 `json:"cost_open"`
		CostHold       float64 `json:"cost_hold"`
		ProfitUnreal   float64 `json:"profit_unreal"`
		ProfitRate     float64 `json:"profit_rate"`
		Profit         float64 `json:"profit"`
		PositionMargin float64 `json:"position_margin"`
		LeverRate      float64 `json:"lever_rate"`
		Direction      string  `json:"direction"`
		LastPrice      float64 `json:"last_price"`
	} `json:"data"`
}

// AvailableLeverageData stores data of available leverage for account
type AvailableLeverageData struct {
	Data []struct {
		ContractCode      string `json:"contract_code"`
		AvailableLeverage string `json:"available_level_rate"`
	} `json:"data"`
	Timestamp types.Time `json:"timestamp"`
}

// FinancialRecordData stores an accounts financial records
type FinancialRecordData struct {
	Data struct {
		FinancialRecord []struct {
			ID           int64      `json:"id"`
			Timestamp    types.Time `json:"ts"`
			Symbol       string     `json:"symbol"`
			ContractCode string     `json:"contract_code"`
			OrderType    int64      `json:"type"`
			Amount       float64    `json:"amount"`
		} `json:"financial_record"`
		TotalPage   int64 `json:"total_page"`
		CurrentPage int64 `json:"current_page"`
		TotalSize   int64 `json:"total_size"`
	} `json:"data"`
}

// SwapOrderLimitInfo stores information about order limits on a perpetual swap
type SwapOrderLimitInfo struct {
	Data struct {
		OrderPriceType string `json:"order_price_type"`
		List           []struct {
			Symbol       string  `json:"symbol"`
			ContractCode string  `json:"contract_code"`
			OpenLimit    float64 `json:"open_limit"`
			CloseLimit   float64 `json:"close_limit"`
		} `json:"list"`
	} `json:"data"`
	Timestamp types.Time `json:"ts"`
}

// SwapTradingFeeData stores trading fee data for swaps
type SwapTradingFeeData struct {
	Data []struct {
		Symbol        string  `json:"symbol"`
		ContractCode  string  `json:"contract_code"`
		FeeAsset      string  `json:"fee_asset"`
		OpenMakerFee  float64 `json:"open_maker_fee,string"`
		OpenTakerFee  float64 `json:"open_taker_fee,string"`
		CloseMakerFee float64 `json:"close_maker_fee,string"`
		CloseTakerFee float64 `json:"close_taker_fee,string"`
	} `json:"data"`
	Timestamp types.Time `json:"ts"`
}

// TransferLimitData stores transfer limits
type TransferLimitData struct {
	Data []struct {
		Symbol                 string  `json:"symbol"`
		ContractCode           string  `json:"contract_code"`
		MaxTransferIn          float64 `json:"transfer_in_max_each"`
		MinTransferIn          float64 `json:"transfer_in_min_each"`
		MaxTransferOut         float64 `json:"transfer_out_max_each"`
		MinTransferOut         float64 `json:"transfer_out_min_each"`
		MaxTransferInDaily     float64 `json:"transfer_in_max_daily"`
		MinTransferInDaily     float64 `json:"transfer_in_min_daily"`
		MaxTransferOutDaily    float64 `json:"transfer_out_max_daily"`
		MinTransferOutDaily    float64 `json:"transfer_out_min_daily"`
		NetTransferInMaxDaily  float64 `json:"net_transfer_in_max_daily"`
		NetTransferOutMaxDaily float64 `json:"net_transfer_out_max_daily"`
	} `json:"data"`
	Timestamp types.Time `json:"timestamp"`
}

// PositionLimitData stores position limit data
type PositionLimitData struct {
	Data []struct {
		Symbol       string  `json:"symbol"`
		ContractCode string  `json:"contract_code"`
		BuyLimit     float64 `json:"buy_limit"`
		SellLimit    float64 `json:"sell_limit"`
	} `json:"data"`
	Timestamp types.Time `json:"ts"`
}

// InternalAccountTransferData stores transfer data between subaccounts and main account
type InternalAccountTransferData struct {
	TS   int64 `json:"ts"`
	Data struct {
		OrderID string `json:"order_id"`
	} `json:"data"`
}

// InternalAccountTransferRecords stores data for transfer records within the account
type InternalAccountTransferRecords struct {
	Timestamp types.Time `json:"ts"`
	Data      struct {
		TransferRecord []struct {
			ID             int64      `json:"id"`
			Timestamp      types.Time `json:"ts"`
			Symbol         string     `json:"symbol"`
			SubUID         int64      `json:"sub_uid"`
			SubAccountName string     `json:"sub_account_name"`
			TransferType   int64      `json:"transfer_type"`
			Amount         float64    `json:"amount"`
		} `json:"transfer_record"`
		TotalPage   int64 `json:"total_page"`
		CurrentPage int64 `json:"current_page"`
		TotalSize   int64 `json:"total_size"`
	} `json:"data"`
}

// SwapOrderData stores swap order data
type SwapOrderData struct {
	Data struct {
		OrderID       int64  `json:"order_id"`
		OrderIDString string `json:"order_id_string"`
		ClientOrderID int64  `json:"client_order_id"`
	} `json:"data"`
	Timestamp types.Time `json:"ts"`
}

// BatchOrderData stores data for batch orders
type BatchOrderData struct {
	Data struct {
		Errors []struct {
			ErrCode int64  `json:"err_code"`
			ErrMsg  string `json:"err_msg"`
			Index   int64  `json:"index"`
		} `json:"errors"`
		Success []struct {
			Index         int64  `json:"index"`
			OrderID       int64  `json:"order_id"`
			OrderIDString string `json:"order_id_str"`
		} `json:"success"`
	} `json:"data"`
	Timestamp types.Time `json:"ts"`
}

// BatchOrderRequestType stores batch order request data
type BatchOrderRequestType struct {
	Data []batchOrderData
}

type batchOrderData struct {
	ContractCode   string  `json:"contract_code"`
	ClientOrderID  string  `json:"client_order_id"`
	Price          float64 `json:"price"`
	Volume         float64 `json:"volume"`
	Direction      string  `json:"direction"`
	Offset         string  `json:"offset"`
	LeverageRate   float64 `json:"leverRate"`
	OrderPriceType string  `json:"orderPriceType"`
}

// CancelOrdersData stores order cancellation data
type CancelOrdersData struct {
	Errors []struct {
		OrderID string `json:"order_id"`
		ErrCode int64  `json:"err_code"`
		ErrMsg  string `json:"err_msg"`
	} `json:"errors"`
	Successes string     `json:"successes"`
	Timestamp types.Time `json:"ts"`
}

// LightningCloseOrderData stores order data from a lightning close order
type LightningCloseOrderData struct {
	Data struct {
		OrderID       int64  `json:"order_id"`
		OrderIDString string `json:"order_id_str"`
		ClientOrderID int64  `json:"client_order_id"`
	}
	Timestamp types.Time `json:"ts"`
}

// SwapOrderInfo stores info for swap orders
type SwapOrderInfo struct {
	Data []struct {
		Symbol          string  `json:"symbol"`
		ContractCode    string  `json:"contract_code"`
		Volume          float64 `json:"volume"`
		Price           float64 `json:"price"`
		OrderPriceType  string  `json:"order_price_type"`
		Direction       string  `json:"direction"`
		Offset          string  `json:"offset"`
		LeverRate       int64   `json:"lever_rate"`
		OrderID         int64   `json:"order_id"`
		OrderIDString   string  `json:"order_id_string"`
		ClientOrderID   int64   `json:"client_order_id"`
		OrderSource     string  `json:"order_source"`
		CreatedAt       int64   `json:"created_at"`
		CancelledAt     int64   `json:"cancelled_at"`
		TradeVolume     float64 `json:"trade_volume"`
		TradeTurnover   float64 `json:"trade_turnover"`
		Fee             float64 `json:"fee"`
		TradeAvgPrice   float64 `json:"trade_avg_price"`
		MarginFrozen    float64 `json:"margin_frozen"`
		Profit          float64 `json:"profit"`
		Status          int64   `json:"status"`
		FeeAsset        float64 `json:"fee_asset"`
		LiquidationType int64   `json:"liquidation_type"`
	}
	Timestamp types.Time `json:"ts"`
}

// OrderDetailData acquires order details
type OrderDetailData struct {
	Data struct {
		Symbol          string  `json:"symbol"`
		ContractCode    string  `json:"contract_code"`
		Volume          float64 `json:"volume"`
		Price           float64 `json:"price"`
		OrderPriceType  string  `json:"order_price_type"`
		Direction       string  `json:"direction"`
		Offset          string  `json:"offset"`
		LeverRate       float64 `json:"lever_rate"`
		MarginFrozen    float64 `json:"margin_frozen"`
		Profit          float64 `json:"profit"`
		OrderSource     string  `json:"order_source"`
		CreatedAt       int64   `json:"created_at"`
		FinalInterest   float64 `json:"final_interest"`
		AdjustValue     float64 `json:"adjust_value"`
		FeeAsset        string  `json:"fee_asset"`
		LiquidationType string  `json:"liquidation_type"`
		OrderID         int64   `json:"order_id"`
		OrderIDStr      string  `json:"order_id_str"`
		ClientOrderID   int     `json:"client_order_id"`
		TradeVolume     float64 `json:"trade_volume"`
		TradeTurnover   float64 `json:"trade_turnover"`
		OrderType       int     `json:"order_type"`
		Status          int     `json:"status"`
		TradeAvgPrice   float64 `json:"trade_avg_price"`
		Trades          []struct {
			ID            string  `json:"id"`
			TradeID       float64 `json:"trade_id"`
			TradeVolume   float64 `json:"trade_volume"`
			TradePrice    float64 `json:"trade_price"`
			TradeFee      float64 `json:"trade_fee"`
			TradeTurnover float64 `json:"trade_turnover"`
			Role          string  `json:"role"`
			CreatedAt     int64   `json:"created_at"`
		} `json:"trades"`
		TotalPage   int64 `json:"total_page"`
		TotalSize   int64 `json:"total_size"`
		CurrentPage int64 `json:"current_page"`
	} `json:"data"`
	Timestamp types.Time `json:"ts"`
}

// SwapOpenOrdersData stores open orders data for swaps
type SwapOpenOrdersData struct {
	Data struct {
		Orders []struct {
			Symbol         string  `json:"symbol"`
			ContractCode   string  `json:"contract_code"`
			Volume         float64 `json:"volume"`
			Price          float64 `json:"price"`
			OrderPriceType string  `json:"order_price_type"`
			OrderType      int64   `json:"order_type"`
			Direction      string  `json:"direction"`
			Offset         string  `json:"offset"`
			LeverageRate   float64 `json:"lever_rate"`
			OrderID        int64   `json:"order_id"`
			OrderIDString  string  `json:"order_id_str"`
			OrderSource    string  `json:"order_source"`
			CreatedAt      int64   `json:"created_at"`
			TradeVolume    float64 `json:"trade_volume"`
			TradeTurnover  float64 `json:"trade_turnover"`
			Fee            float64 `json:"fee"`
			TradeAvgPrice  float64 `json:"trade_avg_price"`
			MarginFrozen   int64   `json:"margin_frozen"`
			Profit         float64 `json:"profit"`
			Status         int64   `json:"status"`
			FeeAsset       string  `json:"fee_asset"`
		} `json:"orders"`
		TotalPage   int64 `json:"total_page"`
		CurrentPage int64 `json:"current_page"`
		TotalSize   int64 `json:"total_size"`
	} `json:"data"`
	Timestamp types.Time `json:"ts"`
}

// SwapOrderHistory gets order history for swaps
type SwapOrderHistory struct {
	Data struct {
		Orders []struct {
			Symbol            string  `json:"symbol"`
			ContractCode      string  `json:"contract_code"`
			Volume            float64 `json:"volume"`
			Price             float64 `json:"price"`
			OrderPriceType    string  `json:"order_price_type"`
			Direction         string  `json:"direction"`
			Offset            string  `json:"offset"`
			LeverageRate      float64 `json:"lever_rate"`
			OrderID           int64   `json:"order_id"`
			OrderIDString     string  `json:"order_id_str"`
			OrderSource       string  `json:"order_source"`
			CreateDate        int64   `json:"create_date"`
			TradeVolume       float64 `json:"trade_volume"`
			TradeTurnover     float64 `json:"trade_turnover"`
			Fee               float64 `json:"fee"`
			TradeAveragePrice float64 `json:"trade_avg_price"`
			MarginFrozen      float64 `json:"margin_frozen"`
			Profit            float64 `json:"profit"`
			Status            int64   `json:"status"`
			OrderType         int64   `json:"order_type"`
			FeeAsset          string  `json:"fee_asset"`
			LiquidationType   string  `json:"liquidation_type"`
		} `json:"orders"`
		TotalPage   int64 `json:"total_page"`
		CurrentPage int64 `json:"current_page"`
		TotalSize   int64 `json:"total_size"`
	} `json:"data"`
	Timestamp types.Time `json:"ts"`
}

// AccountTradeHistoryData stores account trade history for swaps
type AccountTradeHistoryData struct {
	Data struct {
		CurrentPage int64 `json:"current_page"`
		TotalPage   int64 `json:"total_page"`
		TotalSize   int64 `json:"total_size"`
		Trades      []struct {
			ID               string  `json:"id"`
			ContractCode     string  `json:"contract_code"`
			CreateDate       string  `json:"create_date"`
			Direction        string  `json:"direction"`
			MatchID          int64   `json:"match_id"`
			Offset           string  `json:"offset"`
			OffsetProfitloss float64 `json:"offset_profitloss"`
			OrderID          int64   `json:"order_id"`
			OrderIDString    string  `json:"order_id_str"`
			Symbol           string  `json:"symbol"`
			OrderSource      string  `json:"order_source"`
			TradeFee         float64 `json:"trade_fee"`
			TradePrice       float64 `json:"trade_price"`
			TradeTurnover    float64 `json:"trade_turnover"`
			TradeVolume      float64 `json:"trade_volume"`
			Role             string  `json:"role"`
			FeeAsset         string  `json:"fee_asset"`
		} `json:"trades"`
	} `json:"data"`
	Timestamp types.Time `json:"ts"`
}

// TriggerOrderData stores trigger order data
type TriggerOrderData struct {
	Data struct {
		OrderID       int64  `json:"order_id"`
		OrderIDString string `json:"order_id_str"`
	} `json:"data"`
}

// CancelTriggerOrdersData stores trigger order cancel data
type CancelTriggerOrdersData struct {
	Data struct {
		Errors []struct {
			OrderID int64  `json:"order_id"`
			ErrCode int64  `json:"err_code"`
			ErrMsg  string `json:"err_msg"`
		} `json:"errors"`
		Successes string `json:"successes"`
	} `json:"data"`
	Timestamp types.Time `json:"ts"`
}

// TriggerOpenOrdersData stores trigger open orders data
type TriggerOpenOrdersData struct {
	Data struct {
		Orders []struct {
			Symbol         string  `json:"symbol"`
			ContractCode   string  `json:"contract_code"`
			TriggerType    string  `json:"trigger_type"`
			Volume         float64 `json:"volume"`
			OrderType      int64   `json:"order_type"`
			Direction      string  `json:"direction"`
			Offset         string  `json:"offset"`
			LeverageRate   float64 `json:"lever_rate"`
			OrderID        int64   `json:"order_id"`
			OrderIDString  string  `json:"order_id_str"`
			OrderSource    string  `json:"order_source"`
			TriggerPrice   float64 `json:"trigger_price"`
			OrderPrice     float64 `json:"order_price"`
			CreatedAt      int64   `json:"created_at"`
			OrderPriceType string  `json:"order_price_type"`
			Status         int64   `json:"status"`
		} `json:"orders"`
		TotalPage   int64 `json:"total_page"`
		CurrentPage int64 `json:"current_page"`
		TotalSize   int64 `json:"total_size"`
	} `json:"data"`
	Timestamp types.Time `json:"ts"`
}

// TriggerOrderHistory stores trigger order history data for swaps
type TriggerOrderHistory struct {
	Data struct {
		Orders []struct {
			Symbol          string  `json:"symbol"`
			ContractCode    string  `json:"contract_code"`
			TriggerType     string  `json:"trigger_type"`
			Volume          float64 `json:"volume"`
			OrderType       int64   `json:"order_type"`
			Direction       string  `json:"direction"`
			Offset          string  `json:"offset"`
			LeverageRate    float64 `json:"lever_rate"`
			OrderID         int64   `json:"order_id"`
			OrderIDString   string  `json:"order_id_string"`
			RelationOrderID string  `json:"relation_order_id"`
			OrderPriceType  string  `json:"order_price_type"`
			Status          int64   `json:"status"`
			OrderSource     string  `json:"order_source"`
			TriggerPrice    float64 `json:"trigger_price"`
			TriggeredPrice  float64 `json:"triggered_price"`
			OrderPrice      float64 `json:"order_price"`
			CreatedAt       int64   `json:"created_at"`
			TriggeredAt     int64   `json:"triggered_at"`
			OrderInsertAt   float64 `json:"order_insert_at"`
			CancelledAt     int64   `json:"cancelled_at"`
			FailCode        int64   `json:"fail_code"`
			FailReason      string  `json:"fail_reason"`
		} `json:"orders"`
		TotalPage   int64 `json:"total_page"`
		CurrentPage int64 `json:"current_page"`
		TotalSize   int64 `json:"total_size"`
	} `json:"data"`
	Timestamp types.Time `json:"ts"`
}

// TransferMarginBetweenAccountsData stores margin transfer data between spot and swap accounts
type TransferMarginBetweenAccountsData struct {
	Code    int64  `json:"code"`
	Data    int64  `json:"data"`
	Message string `json:"message"`
	Success bool   `json:"success"`
}
