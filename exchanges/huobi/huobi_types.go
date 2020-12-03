package huobi

import "github.com/thrasher-corp/gocryptotrader/exchanges/order"

// WsKlineData stores kline data for futures and swap websocket
type WsKlineData struct {
	Channel   string `json:"ch"`
	Timestamp int64  `json:"ts"`
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
	Channel   string `json:"ch"`
	Timestamp int64  `json:"ts"`
	Tick      struct {
		MRID      int64        `json:"mrid"`
		ID        int64        `json:"id"`
		Bids      [][2]float64 `json:"bids"`
		Asks      [][2]float64 `json:"asks"`
		Timestamp int64        `json:"ts"`
		Version   int64        `json:"version"`
		Channel   string       `json:"ch"`
	} `json:"tick"`
}

// WsIncrementalMarketDepth stores incremental market depth data for swap and futures websocket
type WsIncrementalMarketDepth struct {
	Channel   string `json:"ch"`
	Timestamp int64  `json:"ts"`
	Tick      struct {
		MRID      int64        `json:"mrid"`
		ID        int64        `json:"id"`
		Bids      [][2]float64 `json:"bids"`
		Asks      [][2]float64 `json:"asks"`
		Timestamp int64        `json:"ts"`
		Version   int64        `json:"version"`
		Channel   string       `json:"ch"`
		Event     string       `json:"event"`
	} `json:"tick"`
}

// WsMarketDetail stores market detail data for futures and swap websocket
type WsMarketDetail struct {
	Channel   string `json:"ch"`
	Timestamp int64  `json:"ts"`
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
	Channel   string `json:"ch"`
	Timestamp int64  `json:"ts"`
	Tick      struct {
		Channel   string     `json:"ch"`
		MRID      int64      `json:"mrid"`
		ID        int64      `json:"id"`
		Bid       [2]float64 `json:"bid"`
		Ask       [2]float64 `json:"ask"`
		Timestamp int64      `json:"ts"`
		Version   int64      `json:":version"`
	} `json:"tick"`
}

// WsSubTradeDetail stores trade detail data for futures websocket
type WsSubTradeDetail struct {
	Channel   string `json:"ch"`
	Timestamp int64  `json:"ts"`
	Tick      struct {
		ID        int64 `json:"id"`
		Timestamp int64 `json:"ts"`
		Data      []struct {
			Amount    float64 `json:"amount"`
			Timestamp int64   `json:"ts"`
			ID        int64   `json:"id"`
			Price     float64 `json:"price"`
			Direction string  `json:"direction"`
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
	Rep       string `json:"rep"`
	ID        string `json:"id"`
	Timestamp int64  `json:"ts"`
	Data      []struct {
		ID        int64   `json:"id"`
		Price     float64 `json:"price"`
		Amount    float64 `json:"amount"`
		Direction string  `json:"direction"`
		Timestamp int64   `json:"ts"`
	} `json:"data"`
}

// FWsSubKlineIndex stores subscribed kline index data for futures websocket
type FWsSubKlineIndex struct {
	Channel   string `json:"ch"`
	Timestamp int64  `json:"ts"`
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
	ID        string `json:"id"`
	Rep       string `json:"rep"`
	WsID      int64  `json:"wsid"`
	Timestamp int64  `json:"ts"`
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
	Channel   string `json:"ch"`
	Timestamp int64  `json:"ts"`
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
	ID        string `json:"id"`
	Rep       string `json:"rep"`
	Timestamp int64  `json:"ts"`
	WsID      int64  `json:"wsid"`
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
	Operation      string  `json:"op"`
	Topic          string  `json:"topic"`
	UID            string  `json:"uid"`
	Timestamp      int64   `json:"ts"`
	Symbol         string  `json:"symbol"`
	ContractType   string  `json:"contract_type"`
	ContractCode   string  `json:"contract_code"`
	Volume         float64 `json:"volume"`
	Price          float64 `json:"price"`
	OrderPriceType string  `json:"order_price_type"`
	Direction      string  `json:"direction"`
	Offset         string  `json:"offset"`
	Status         int64   `json:"status"`
	LeverageRate   int64   `json:"lever_rate"`
	OrderID        int64   `json:"order_id"`
	OrderIDString  string  `json:"order_id_string"`
	ClientOrderID  int64   `json:"client_order_id"`
	OrderSource    string  `json:"order_source"`
	OrderType      int64   `json:"order_type"`
	CreatedAt      int64   `json:"created_at"`
	TradeVolume    float64 `json:"trade_volume"`
	TradeTurnover  float64 `json:"trade_turnover"`
	Fee            float64 `json:"fee"`
	TradeAvgPrice  float64 `json:"trade_avg_price"`
	MarginFrozen   float64 `json:"margin_frozen"`
	Profit         float64 `json:"profit"`
	FeeAsset       string  `json:"fee_asset"`
	CancelledAt    int64   `json:"canceled_at"`
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
	Operation     string  `json:"op"`
	Topic         string  `json:"topic"`
	UID           string  `json:"uid"`
	Timestamp     int64   `json:"ts"`
	Symbol        string  `json:"symbol"`
	ContractType  string  `json:"contract_type"`
	ContractCode  string  `json:"contract_code"`
	Status        int64   `json:"status"`
	OrderID       int64   `json:"order_id"`
	OrderIDString string  `json:"order_id_string"`
	OrderType     string  `json:"order_type"`
	Volume        float64 `json:"volume"`
	TradeVolume   float64 `json:"trade_volume"`
	ClientOrderID int64   `json:"client_order_id"`
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
	Operation string `json:"op"`
	Topic     string `json:"topic"`
	UID       string `json:"uid"`
	Timestamp int64  `json:"ts"`
	Event     string `json:"event"`
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
	Operation     string `json:"op"`
	Topic         string `json:"topic"`
	UID           string `json:"uid"`
	Timestamp     int64  `json:"ts"`
	Event         string `json:"event"`
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
	Operation  string `json:"op"`
	Topic      string `json:"topic"`
	Timestamp  int64  `json:"ts"`
	OrdersData []struct {
		Symbol       string  `json:"symbol"`
		ContractCode string  `json:"contract_code"`
		Direction    string  `json:"direction"`
		Offset       string  `json:"offset"`
		Volume       float64 `json:"volume"`
		Price        float64 `json:"price"`
		CreatedAt    int64   `json:"created_at"`
	} `json:"data"`
}

// FWsSubContractInfo stores contract info data for futures websocket
type FWsSubContractInfo struct {
	Operation    string `json:"op"`
	Topic        string `json:"topic"`
	Timestamp    int64  `json:"ts"`
	Event        string `json:"event"`
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

// FContractInfoData gets contract info data for futures
type FContractInfoData struct {
	Data []struct {
		Symbol         string  `json:"symbol"`
		ContractCode   string  `json:"contract_code"`
		ContractType   string  `json:"contract_type"`
		ContractSize   float64 `json:"contract_size"`
		PriceTick      float64 `json:"price_tick"`
		DeliveryDate   string  `json:"delivery_date"`
		CreateDate     string  `json:"create_date"`
		ContractStatus int64   `json:"contract_status"`
	}
}

// FContractIndexPriceInfo stores contract index price
type FContractIndexPriceInfo struct {
	Data []struct {
		Symbol     string  `json:"symbol"`
		IndexPrice float64 `json:"index_price"`
	} `json:"data"`
	Timestamp int64 `json:"ts"`
}

// FContractPriceLimits gets limits for futures contracts
type FContractPriceLimits struct {
	Data struct {
		Symbol       string  `json:"symbol"`
		HighLimit    float64 `json:"high_limit"`
		LowLimit     float64 `json:"low_limit"`
		ContractCode string  `json:"contract_code"`
		ContractType string  `json:"contract_type"`
	} `json:"data"`
	Timestamp int64 `json:"ts"`
}

// FContractOIData stores open interest data for futures contracts
type FContractOIData struct {
	Data []struct {
		Symbol       string  `json:"symbol"`
		ContractType string  `json:"contract_type"`
		Volume       float64 `json:"volume"`
		Amount       float64 `json:"amount"`
		ContractCode string  `json:"contract_code"`
	} `json:"data"`
	Timestamp int64 `json:"ts"`
}

// FEstimatedDeliveryPriceInfo stores estimated delivery price data for futures
type FEstimatedDeliveryPriceInfo struct {
	Data struct {
		DeliveryPrice float64 `json:"delivery_price"`
	} `json:"data"`
	Timestamp int64 `json:"ts"`
}

// FMarketDepth gets orderbook data for futures
type FMarketDepth struct {
	Ch        string `json:"ch"`
	Timestamp int64  `json:"ts"`
	Tick      struct {
		MRID      int64        `json:"mrid"`
		ID        int64        `json:"id"`
		Bids      [][2]float64 `json:"bids"`
		Asks      [][2]float64 `json:"asks"`
		Timestamp int64        `json:"ts"`
		Version   int64        `json:"version"`
		Ch        string       `json:"ch"`
	} `json:"tick"`
}

// OBData stores market depth data
type OBData struct {
	Symbol string
	Asks   []obItem
	Bids   []obItem
}

type obItem struct {
	Price    float64
	Quantity float64
}

// FKlineData stores kline data for futures
type FKlineData struct {
	Ch   string `json:"ch"`
	Data []struct {
		Vol    float64 `json:"vol"`
		Close  float64 `json:"close"`
		Count  float64 `json:"count"`
		High   float64 `json:"high"`
		ID     int64   `json:"id"`
		Low    float64 `json:"low"`
		Open   float64 `json:"open"`
		Amount float64 `json:"amount"`
	} `json:"data"`
	Timestamp int64 `json:"ts"`
}

// FMarketOverviewData stores overview data for futures
type FMarketOverviewData struct {
	Ch   string `json:"ch"`
	Tick struct {
		Vol       float64 `json:"vol,string"`
		Ask       [2]float64
		Bid       [2]float64
		Close     float64 `json:"close,string"`
		Count     float64 `json:"count"`
		High      float64 `json:"high,string"`
		ID        int64   `jso:"id"`
		Low       float64 `json:"low,string"`
		Open      float64 `json:"open,string"`
		Timestamp int64   `json:"ts"`
		Amount    float64 `json:"amount,string"`
	} `json:"tick"`
	Timestamp int64 `json:"ts"`
}

// FLastTradeData stores last trade's data for a contract
type FLastTradeData struct {
	Ch   string `json:"ch"`
	Tick struct {
		Data []struct {
			Amount    float64 `json:"amount,string"`
			Direction string  `json:"direction"`
			ID        int64   `json:"id"`
			Price     float64 `json:"price,string"`
			Timestamp int64   `json:"ts"`
		} `json:"data"`
		ID        int64 `json:"id"`
		Timestamp int64 `json:"ts"`
	} `json:"tick"`
	Timestamp int64 `json:"ts"`
}

// FBatchTradesForContractData stores batch of trades data for a contract
type FBatchTradesForContractData struct {
	Ch        string `json:"ch"`
	Timestamp int64  `json:"ts"`
	Data      []struct {
		ID        int64 `json:"id"`
		Timestamp int64 `json:"ts"`
		Data      []struct {
			Amount    float64 `json:"amount"`
			Direction string  `json:"direction"`
			ID        int64   `json:"id"`
			Price     float64 `json:"price"`
			Timestamp int64   `json:"ts"`
		} `json:"data"`
	} `json:"data"`
}

// FClawbackRateAndInsuranceData stores clawback rate and insurance data for futures
type FClawbackRateAndInsuranceData struct {
	Timestamp int64 `json:"ts"`
	Data      []struct {
		Symbol            string  `json:"symbol"`
		InsuranceFund     float64 `json:"insurance_fund"`
		EstimatedClawback float64 `json:"estimated_clawback"`
	} `json:"data"`
}

// FHistoricalInsuranceRecordsData stores historical records of insurance fund balances for futures
type FHistoricalInsuranceRecordsData struct {
	Timestamp int64 `json:"timestamp"`
	Data      struct {
		Symbol string `json:"symbol"`
		Tick   []struct {
			InsuranceFund float64 `json:"insurance_fund"`
			Timestamp     int64   `json:"ts"`
		} `json:"tick"`
	} `json:"data"`
}

// FTieredAdjustmentFactorInfo stores info on adjustment factor for futures
type FTieredAdjustmentFactorInfo struct {
	Data []struct {
		Symbol string `json:"symbol"`
		List   []struct {
			LeverageRate float64 `json:"lever_rate"`
			Ladders      []struct {
				Ladder       int64   `json:"ladder"`
				MinSize      float64 `json:"min_size"`
				MaxSize      float64 `json:"max_size"`
				AdjustFactor float64 `json:"adjust_factor"`
			} `json:"ladders"`
		} `json:"list"`
	} `json:"data"`
	Timestamp int64 `json:"ts"`
}

// FOIData gets oi data on futures
type FOIData struct {
	Data struct {
		Symbol       string `json:"symbol"`
		ContractType string `json:"contract_type"`
		Tick         []struct {
			Volume     float64 `json:"volume,string"`
			AmountType int64   `json:"amount_type"`
			Timestamp  int64   `json:"ts"`
		} `json:"tick"`
	} `json:"data"`
	Timestamp int64 `json:"ts"`
}

// FInfoSystemStatusData stores system status info for futures
type FInfoSystemStatusData struct {
	Data []struct {
		Symbol      string `json:"symbol"`
		Open        int64  `json:"open"`
		Close       int64  `json:"close"`
		Cancel      int64  `json:"cancel"`
		TransferIn  int64  `json:"transfer_in"`
		TransferOut int64  `json:"transfer_out"`
	} `json:"data"`
	Timestamp int64 `json:"ts"`
}

// FTopAccountsLongShortRatio stores long/short ratio for top futures accounts
type FTopAccountsLongShortRatio struct {
	Data struct {
		List []struct {
			BuyRatio    float64 `json:"buy_ratio"`
			SellRatio   float64 `json:"sell_ratio"`
			LockedRatio float64 `json:"locked_ratio"`
			Timestamp   int64   `json:"ts"`
		} `json:"list"`
		Symbol string `json:"symbol"`
	} `json:"data"`
	Timestamp int64 `json:"ts"`
}

// FTopPositionsLongShortRatio stores long short ratio for top futures positions
type FTopPositionsLongShortRatio struct {
	Data struct {
		Symbol string `json:"symbol"`
		List   []struct {
			BuyRatio  float64 `json:"buy_ratio"`
			SellRatio float64 `json:"sell_ratio"`
			Timestamp int64   `json:"timestamp"`
		} `json:"list"`
	} `json:"data"`
	Timestamp int64 `json:"timestamp"`
}

// FLiquidationOrdersInfo stores data of futures liquidation orders
type FLiquidationOrdersInfo struct {
	Data struct {
		Orders []struct {
			Symbol       string  `json:"symbol"`
			ContractCode string  `json:"contract_code"`
			Direction    string  `json:"direction"`
			Offset       string  `json:"offset"`
			Volume       float64 `json:"volume"`
			Price        float64 `json:"price"`
			CreatedAt    int64   `json:"created_at"`
		} `json:"orders"`
		TotalPage   int64 `json:"total_page"`
		CurrentPage int64 `json:"current_page"`
		TotalSize   int64 `json:"total_size"`
	} `json:"data"`
	Timestamp int64 `json:"ts"`
}

// FIndexKlineData stores index kline data for futures
type FIndexKlineData struct {
	Ch   string `json:"ch"`
	Data []struct {
		Vol    float64 `json:"vol"`
		Close  float64 `json:"close"`
		Count  float64 `json:"count"`
		High   float64 `json:"high"`
		ID     int64   `json:"id"`
		Low    float64 `json:"low"`
		Open   float64 `json:"open"`
		Amount float64 `json:"amount"`
	} `json:"data"`
	Timestamp int64 `json:"ts"`
}

// FBasisData stores basis data for futures
type FBasisData struct {
	Ch   string `json:"ch"`
	Data []struct {
		Basis         float64 `json:"basis"`
		BasisRate     float64 `json:"basis_rate"`
		ContractPrice float64 `json:"contract_price"`
		ID            int64   `json:"id"`
		IndexPrice    float64 `json:"index_price"`
	} `json:"data"`
	Timestamp int64 `json:"ts"`
}

// FUserAccountData stores user account data info for futures
type FUserAccountData struct {
	AccData []struct {
		Symbol            string  `json:"symbol"`
		MarginBalance     float64 `json:"margin_balance"`
		MarginPosition    float64 `json:"margin_position"`
		MarginFrozen      float64 `json:"margin_frozen"`
		MarginAvailable   float64 `json:"margin_available"`
		ProfitReal        float64 `json:"profit_real"`
		ProfitUnreal      float64 `json:"profit_unreal"`
		RiskRate          float64 `json:"risk_rate"`
		LiquidationPrice  float64 `json:"liquidation_price"`
		WithdrawAvailable float64 `json:"withdraw_available"`
		LeverageRate      float64 `json:"lever_rate"`
		AdjustFactor      float64 `json:"adjust_factor"`
		MarginStatic      float64 `json:"margin_static"`
	} `json:"data"`
	Timestamp int64 `json:"ts"`
}

// FUsersPositionsInfo stores positions data for futures
type FUsersPositionsInfo struct {
	PosInfo []struct {
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
	Timestamp int64 `json:"ts"`
}

// FSubAccountAssetsInfo gets subaccounts asset data
type FSubAccountAssetsInfo struct {
	Timestamp int64 `json:"ts"`
	Data      []struct {
		SubUID int64 `json:"sub_uid"`
		List   []struct {
			Symbol           string  `json:"symbol"`
			MarginBalance    float64 `json:"margin_balance"`
			LiquidationPrice float64 `json:"liquidation_price"`
			RiskRate         float64 `json:"risk_rate"`
		} `json:"list"`
	} `json:"data"`
}

// FSingleSubAccountAssetsInfo stores futures assets info for a single subaccount
type FSingleSubAccountAssetsInfo struct {
	AssetsData []struct {
		Symbol            string  `json:"symbol"`
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
	} `json:"data"`
	Timestamp int64 `json:"ts"`
}

// FSingleSubAccountPositionsInfo stores futures positions' info for a single subaccount
type FSingleSubAccountPositionsInfo struct {
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
	Timestamp int64 `json:"ts"`
}

// FFinancialRecords stores financial records data for futures
type FFinancialRecords struct {
	Data struct {
		FinancialRecord []struct {
			ID         int64   `json:"id"`
			Timestamp  int64   `json:"ts"`
			Symbol     string  `json:"symbol"`
			RecordType int64   `json:"type"`
			Amount     float64 `json:"amount"`
		} `json:"financial_record"`
		TotalPage   int64 `json:"total_page"`
		CurrentPage int64 `json:"current_page"`
		TotalSize   int64 `json:"total_size"`
	} `json:"data"`
	Timestamp int64 `json:"ts"`
}

// FSettlementRecords stores user's futures settlement records
type FSettlementRecords struct {
	Data struct {
		SettlementRecords []struct {
			Symbol               string  `json:"symbol"`
			MarginBalanceInit    float64 `json:"margin_balance_init"`
			MarginBalance        int64   `json:"margin_balance"`
			SettlementProfitReal float64 `json:"settlement_profit_real"`
			SettlementTime       int64   `json:"settlement_time"`
			Clawback             float64 `json:"clawback"`
			DeliveryFee          float64 `json:"delivery_fee"`
			OffsetProfitLoss     float64 `json:"offset_profitloss"`
			Fee                  float64 `json:"fee"`
			FeeAsset             string  `json:"fee_asset"`
			Positions            []struct {
				Symbol                 string  `json:"symbol"`
				ContractCode           string  `json:"contract_code"`
				Direction              string  `json:"direction"`
				Volume                 float64 `json:"volume"`
				CostOpen               float64 `json:"cost_open"`
				CostHoldPre            float64 `json:"cost_hold_pre"`
				CostHold               float64 `json:"cost_hold"`
				SettlementProfitUnreal float64 `json:"settlement_profit_unreal"`
				SettlementPrice        float64 `json:"settlement_price"`
				SettlmentType          string  `json:"settlement_type"`
			} `json:"positions"`
		} `json:"settlement_records"`
		CurrentPage int64 `json:"current_page"`
		TotalPage   int64 `json:"total_page"`
		TotalSize   int64 `json:"total_size"`
	} `json:"data"`
	Timestamp int64 `json:"ts"`
}

// FContractInfoOnOrderLimit stores contract info on futures order limit
type FContractInfoOnOrderLimit struct {
	ContractData []struct {
		OrderPriceType string `json:"order_price_type"`
		List           []struct {
			Symbol        string `json:"symbol"`
			ContractTypes []struct {
				ContractType string `json:"contract_type"`
				OpenLimit    int64  `json:"open_limit"`
				CloseLimit   int64  `json:"close_limit"`
			} `json:"types"`
		} `json:"list"`
	} `json:"data"`
	Timestamp int64 `json:"ts"`
}

// FContractTradingFeeData stores contract trading fee data
type FContractTradingFeeData struct {
	ContractTradingFeeData []struct {
		Symbol        string  `json:"symbol"`
		OpenMakerFee  float64 `json:"open_maker_fee,string"`
		OpenTakerFee  float64 `json:"open_taker_fee,string"`
		CloseMakerFee float64 `json:"close_maker_fee,string"`
		CloseTakerFee float64 `json:"close_taker_fee,string"`
		DeliveryFee   float64 `json:"delivery_fee,string"`
		FeeAsset      string  `json:"fee_asset"`
	} `json:"data"`
	Timestamp int64 `json:"ts"`
}

// FTransferLimitData stores transfer limit data for futures
type FTransferLimitData struct {
	Data []struct {
		Symbol                 string  `json:"symbol"`
		MaxTransferIn          float64 `json:"transfer_in_max_each"`
		MinTransferIn          float64 `json:"transfer_in_min_each"`
		MaxTransferOut         float64 `json:"transfer_out_max_each"`
		MinTransferOut         float64 `json:"transfer_out_min_each"`
		MaxTransferInDaily     float64 `json:"transfer_in_max_daily"`
		MaxTransferOutDaily    float64 `json:"transfer_out_max_daily"`
		NetTransferInMaxDaily  float64 `json:"net_transfer_in_max_daily"`
		NetTransferOutMaxDaily float64 `json:"net_transfer_out_max_daily"`
	} `json:"data"`
	Timestamp int64 `json:"ts"`
}

// FPositionLimitData stores information on futures positions limit
type FPositionLimitData struct {
	Data []struct {
		Symbol string `json:"symbol"`
		List   []struct {
			ContractType string  `json:"contract_type"`
			BuyLimit     float64 `json:"buy_limit"`
			SellLimit    float64 `json:"sell_limit"`
		} `json:"list"`
	} `json:"data"`
	Timestamp int64 `json:"ts"`
}

// FAssetsAndPositionsData stores assets and positions data for futures
type FAssetsAndPositionsData struct {
	Data []struct {
		Symbol            string  `json:"symbol"`
		MarginBalance     float64 `json:"margin_balance"`
		MarginPosition    float64 `json:"margin_position"`
		MarginFrozen      float64 `json:"margin_frozen"`
		MarginAvailable   float64 `json:"margin_available"`
		ProfitReal        float64 `json:"profit_real"`
		ProfitUnreal      float64 `json:"profit_unreal"`
		RiskRate          float64 `json:"risk_rate"`
		WithdrawAvailable float64 `json:"withdraw_available"`
	} `json:"data"`
}

// FAccountTransferData stores internal transfer data for futures
type FAccountTransferData struct {
	Status    string `json:"status"`
	Timestamp int64  `json:"ts"`
	Data      struct {
		OrderID string `json:"order_id"`
	} `json:"data"`
}

// FTransferRecords gets transfer records data
type FTransferRecords struct {
	Timestamp int64 `json:"ts"`
	Data      struct {
		TransferRecord []struct {
			ID             int64   `json:"id"`
			Timestamp      int64   `json:"ts"`
			Symbol         string  `json:"symbol"`
			SubUID         int64   `json:"sub_uid"`
			SubAccountName string  `json:"sub_account_name"`
			TransferType   int64   `json:"transfer_type"`
			Amount         float64 `json:"amount"`
		} `json:"transfer_record"`
		TotalPage   int64 `json:"total_page"`
		CurrentPage int64 `json:"current_page"`
		TotalSize   int64 `json:"total_size"`
	} `json:"data"`
}

// FAvailableLeverageData stores available leverage data for futures
type FAvailableLeverageData struct {
	Data []struct {
		Symbol                string `json:"symbol"`
		AvailableLeverageRate string `json:"available_level_rate"`
	} `json:"data"`
	Timestamp int64 `json:"timestamp"`
}

// FOrderData stores order data for futures
type FOrderData struct {
	Data struct {
		OrderID       int64  `json:"order_id"`
		OrderIDStr    string `json:"order_id_str"`
		ClientOrderID int64  `json:"client_order_id"`
	} `json:"data"`
	Timestamp int64 `json:"ts"`
}

type fBatchOrderData struct {
	Symbol         string  `json:"symbol"`
	ContractType   string  `json:"contract_type"`
	ContractCode   string  `json:"contract_code"`
	ClientOrderID  string  `json:"client_order_id"`
	Price          float64 `json:"price"`
	Volume         float64 `json:"volume"`
	Direction      string  `json:"direction"`
	Offset         string  `json:"offset"`
	LeverageRate   float64 `json:"leverRate"`
	OrderPriceType string  `json:"orderPriceType"`
}

// FBatchOrderResponse stores batch order data
type FBatchOrderResponse struct {
	OrdersData []FOrderData `json:"orders_data"`
}

// FCancelOrderData stores cancel order data
type FCancelOrderData struct {
	Data struct {
		Errors []struct {
			OrderID int64  `json:"order_id"`
			ErrCode int64  `json:"err_code,string"`
			ErrMsg  string `json:"err_msg"`
		} `json:"errors"`
		Successes string `json:"successes"`
	} `json:"data"`
	Timestamp int64 `json:"ts"`
}

// FOrderInfo stores order info
type FOrderInfo struct {
	Data []struct {
		ClientOrderID   int64   `json:"client_order_id"`
		ContractCode    string  `json:"contract_code"`
		ContractType    string  `json:"contract_type"`
		CreatedAt       int64   `json:"created_at"`
		CanceledAt      int64   `json:"canceled_at"`
		Direction       string  `json:"direction"`
		Fee             float64 `json:"fee"`
		FeeAsset        string  `json:"fee_asset"`
		LeverRate       int64   `json:"lever_rate"`
		MarginFrozen    float64 `json:"margin_frozen"`
		Offset          string  `json:"offset"`
		OrderID         int64   `json:"order_id"`
		OrderIDString   string  `json:"order_id_string"`
		OrderPriceType  string  `json:"order_price_type"`
		OrderSource     string  `json:"order_source"`
		OrderType       int64   `json:"order_type"`
		Price           float64 `json:"price"`
		Profit          float64 `json:"profit"`
		Status          int64   `json:"status"`
		Symbol          string  `json:"symbol"`
		TradeAvgPrice   float64 `json:"trade_avg_price"`
		TradeTurnover   float64 `json:"trade_turnover"`
		TradeVolume     float64 `json:"trade_volume"`
		Volume          float64 `json:"volume"`
		LiquidationType int64   `json:"liquidation_type"`
	} `json:"data"`
	Timestamp int64 `json:"timestamp"`
}

// FOrderDetailsData stores order details for futures orders
type FOrderDetailsData struct {
	Data struct {
		Symbol         string  `json:"symbol"`
		ContractType   string  `json:"contract_type"`
		ContractCode   string  `json:"contract_code"`
		Volume         float64 `json:"volume"`
		Price          float64 `json:"price"`
		OrderPriceType string  `json:"order_price_type"`
		Direction      string  `json:"direction"`
		Offset         string  `json:"offset"`
		LeverageRate   float64 `json:"lever_rate"`
		MarginFrozen   float64 `json:"margin_frozen"`
		Profit         float64 `json:"profit"`
		OrderSource    string  `json:"order_source"`
		OrderID        int64   `json:"order_id"`
		OrderIDString  string  `json:"order_id_str"`
		ClientOrderID  int64   `json:"client_order_id"`
		OrderType      int64   `json:"order_type"`
		Status         int64   `json:"status"`
		TradeVolume    float64 `json:"trade_volume"`
		TradeTurnover  int64   `json:"trade_turnover"`
		TradeAvgPrice  float64 `json:"trade_avg_price"`
		Fee            float64 `json:"fee"`
		CreatedAt      int64   `json:"created_at"`
		CanceledAt     int64   `json:"canceled_at"`
		FinalInterest  float64 `json:"final_interest"`
		AdjustValue    int64   `json:"adjust_value"`
		FeeAsset       string  `json:"fee_asset"`
		Trades         []struct {
			ID            string  `json:"id"`
			TradeID       int64   `json:"trade_id"`
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
	Timestamp int64 `json:"ts"`
}

// FOpenOrdersData stores open orders data for futures
type FOpenOrdersData struct {
	Data struct {
		Orders []struct {
			Symbol         string  `json:"symbol"`
			ContractType   string  `json:"contract_type"`
			ContractCode   string  `json:"contract_code"`
			Volume         float64 `json:"volume"`
			Price          float64 `json:"price"`
			OrderPriceType string  `json:"order_price_type"`
			OrderType      int64   `json:"order_type"`
			Direction      string  `json:"direction"`
			Offset         string  `json:"offset"`
			LeverageRate   float64 `json:"lever_rate"`
			OrderID        int64   `json:"order_id"`
			OrderIDString  string  `json:"order_id_string"`
			ClientOrderID  int64   `json:"client_order_id"`
			OrderSource    string  `json:"order_source"`
			CreatedAt      int64   `json:"created_at"`
			TradeVolume    float64 `json:"trade_volume"`
			Fee            float64 `json:"fee"`
			TradeAvgPrice  float64 `json:"trade_avg_price"`
			MarginFrozen   float64 `json:"margin_frozen"`
			Profit         float64 `json:"profit"`
			Status         int64   `json:"status"`
			FeeAsset       string  `json:"fee_asset"`
		} `json:"orders"`
		TotalPage   int64 `json:"total_page"`
		CurrentPage int64 `json:"current_page"`
		TotalSize   int64 `json:"total_size"`
	} `json:"data"`
	Timestamp int64 `json:"ts"`
}

// FOrderHistoryData stores order history data
type FOrderHistoryData struct {
	Data struct {
		Orders []struct {
			Symbol          string  `json:"symbol"`
			ContractType    string  `json:"contract_type"`
			ContractCode    string  `json:"contract_code"`
			Volume          float64 `json:"volume"`
			Price           float64 `json:"price"`
			OrderPriceType  string  `json:"order_price_type"`
			Direction       string  `json:"direction"`
			Offset          string  `json:"offset"`
			LeverageRate    float64 `json:"lever_rate"`
			OrderID         int64   `json:"order_id"`
			OrderIDString   string  `json:"order_id_str"`
			OrderSource     string  `json:"order_source"`
			CreateDate      int64   `json:"create_date"`
			TradeVolume     float64 `json:"trade_volume"`
			TradeTurnover   float64 `json:"trade_turnover"`
			Fee             float64 `json:"fee"`
			TradeAvgPrice   float64 `json:"trade_avg_price"`
			MarginFrozen    float64 `json:"margin_frozen"`
			Profit          float64 `json:"profit"`
			Status          int64   `json:"status"`
			OrderType       int64   `json:"order_type"`
			FeeAsset        string  `json:"fee_asset"`
			LiquidationType int64   `json:"liquidation_type"`
		} `json:"orders"`
		TotalPage   int64 `json:"total_page"`
		CurrentPage int64 `json:"current_page"`
		TotalSize   int64 `json:"total_size"`
	} `json:"data"`
	Timestamp int64 `json:"ts"`
}

// FTradeHistoryData stores trade history data for futures
type FTradeHistoryData struct {
	Data struct {
		TotalPage   int64 `json:"total_page"`
		CurrentPage int64 `json:"current_page"`
		TotalSize   int64 `json:"total_size"`
		Trades      []struct {
			ID            string  `json:"id"`
			ContractCode  string  `json:"contract_code"`
			ContractType  string  `json:"contract_type"`
			CreateDate    int64   `json:"create_date"`
			Direction     string  `json:"direction"`
			MatchID       int64   `json:"match_id"`
			Offset        string  `json:"offset"`
			OffsetPNL     float64 `json:"offset_profitloss"`
			OrderID       int64   `json:"order_id"`
			OrderIDString string  `json:"order_id_str"`
			Symbol        string  `json:"symbol"`
			OrderSource   string  `json:"order_source"`
			TradeFee      float64 `json:"trade_fee"`
			TradePrice    float64 `json:"trade_price"`
			TradeTurnover float64 `json:"trade_turnover"`
			TradeVolume   float64 `json:"trade_volume"`
			Role          string  `json:"role"`
			FeeAsset      string  `json:"fee_asset"`
		} `json:"trades"`
	} `json:"data"`
	Timestamp int64 `json:"ts"`
}

// FTriggerOrderData stores trigger order data
type FTriggerOrderData struct {
	Data struct {
		OrderID    int64  `json:"order_id"`
		OrderIDStr string `json:"order_id_str"`
	} `json:"data"`
	Timestamp int64 `json:"ts"`
}

// FTriggerOpenOrders stores trigger open orders data
type FTriggerOpenOrders struct {
	Data struct {
		Orders []struct {
			Symbol         string  `json:"symbol"`
			ContractCode   string  `json:"contract_code"`
			ContractType   string  `json:"contract_type"`
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
	Timestamp int64 `json:"ts"`
}

// FTriggerOrderHistoryData stores trigger order history for futures
type FTriggerOrderHistoryData struct {
	Data struct {
		Orders []struct {
			Symbol          string  `json:"symbol"`
			ContractCode    string  `json:"contract_code"`
			ContractType    string  `json:"contract_type"`
			TriggerType     string  `json:"trigger_type"`
			Volume          float64 `json:"volume"`
			OrderType       int64   `json:"order_type"`
			Direction       string  `json:"direction"`
			Offset          string  `json:"offset"`
			LeverageRate    float64 `json:"lever_rate"`
			OrderID         int64   `json:"order_id"`
			OrderIDString   string  `json:"order_id_str"`
			RelationOrderID string  `json:"relation_order_id"`
			OrderPriceType  string  `json:"order_price_type"`
			Status          string  `json:"status"`
			OrderSource     string  `json:"order_source"`
			TriggerPrice    int64   `json:"trigger_price"`
			TriggeredPrice  float64 `json:"triggered_price"`
			OrderPrice      float64 `json:"order_price"`
			CreatedAt       int64   `json:"created_at"`
			TriggeredAt     int64   `json:"triggered_at"`
			OrderInsertAt   float64 `json:"order_insert_at"`
			CancelledAt     int64   `json:"canceled_at"`
			FailCode        int64   `json:"fail_code"`
			FailReason      string  `json:"fail_reason"`
		} `json:"orders"`
		TotalPage   int64 `json:"total_page"`
		CurrentPage int64 `json:"current_page"`
		TotalSize   int64 `json:"total_size"`
	} `json:"data"`
	Timestamp int64 `json:"ts"`
}

// Coin Margined Swaps

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
	Rep       string `json:"rep"`
	ID        int64  `json:"id"`
	Timestamp int64  `json:"ts"`
	Data      []struct {
		ID        int64   `json:"id"`
		Price     float64 `json:"price"`
		Amount    float64 `json:"amount"`
		Direction string  `json:"direction"`
		Timestamp int64   `json:"ts"`
	} `json:"data"`
}

// SwapWsSubPremiumKline stores subscribed premium kline data for futures websocket
type SwapWsSubPremiumKline struct {
	Channel   string `json:"ch"`
	Timestamp int64  `json:"ts"`
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
	Rep       string `json:"rep"`
	ID        string `json:"id"`
	WsID      int64  `json:"wsid"`
	Timestamp int64  `json:"ts"`
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
	Channel   string `json:"ch"`
	Timestamp int64  `json:"ts"`
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
	Rep       string `json:"rep"`
	ID        string `json:"id"`
	WsID      int64  `json:"wsid"`
	Timestamp int64  `json:"ts"`
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
	Channel   string `json:"ch"`
	Timestamp int64  `json:"ts"`
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
	Rep       string `json:"rep"`
	ID        string `json:"id"`
	WsID      int64  `json:"wsid"`
	Timestamp int64  `json:"ts"`
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
	Operation      string  `json:"op"`
	Topic          string  `json:"topic"`
	UID            string  `json:"uid"`
	Timestamp      int64   `json:"ts"`
	Symbol         string  `json:"symbol"`
	ContractCode   string  `json:"contract_code"`
	Volume         float64 `json:"volume"`
	Price          float64 `json:"price"`
	OrderPriceType string  `json:"order_price_type"`
	Direction      string  `json:"direction"`
	Offset         string  `json:"offset"`
	Status         int64   `json:"status"`
	LeverateRate   float64 `json:"lever_rate"`
	OrderID        int64   `json:"order_id"`
	OrderIDString  string  `json:"order_id_str"`
	ClientOrderID  int64   `json:"client_order_id"`
	OrderSource    string  `json:"order_source"`
	OrderType      int64   `json:"order_type"`
	CreatedAt      int64   `json:"created_at"`
	CanceledAt     int64   `json:"canceled_at"`
	TradeVolume    float64 `json:"trade_volume"`
	TradeTurnover  float64 `json:"trade_turnover"`
	Fee            float64 `json:"fee"`
	FeeAsset       string  `json:"fee_asset"`
	TradeAvgPrice  float64 `json:"trade_avg_price"`
	MarginFrozen   float64 `json:"margin_frozen"`
	Profit         float64 `json:"profit"`
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
	Operation     string  `json:"op"`
	Topic         string  `json:"topic"`
	UID           string  `json:"uid"`
	Timestamp     int64   `json:"ts"`
	Symbol        string  `json:"symbol"`
	ContractCode  string  `json:"contract_code"`
	Status        int64   `json:"status"`
	OrderID       int64   `json:"order_id"`
	OrderIDString string  `json:"order_id_str"`
	ClientOrderID int64   `json:"client_order_id"`
	OrderType     string  `json:"order_type"`
	TradeVolume   int64   `json:"trade_volume"`
	Volume        float64 `json:"volume"`
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
	Operation string `json:"op"`
	Topic     string `json:"topic"`
	Timestamp int64  `json:"ts"`
	UID       string `json:"uid"`
	Event     string `json:"event"`
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
	Operation string `json:"op"`
	Topic     string `json:"topic"`
	UID       string `json:"uid"`
	Timestamp int64  `json:"ts"`
	Event     string `json:"event"`
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
	Operation  string `json:"op"`
	Topic      string `json:"topic"`
	Timestamp  int64  `json:"ts"`
	OrdersData []struct {
		Symbol       string  `json:"symbol"`
		ContractCode string  `json:"contract_code"`
		Direction    string  `json:"direction"`
		Offset       string  `json:"offset"`
		Volume       float64 `json:"volume"`
		Price        float64 `json:"price"`
		CreatedAt    int64   `json:"created_at"`
	} `json:"data"`
}

// SwapWsSubFundingData stores funding rate data for swap websocket
type SwapWsSubFundingData struct {
	Operation   string `json:"op"`
	Topic       string `json:"topic"`
	Timestamp   int64  `json:"ts"`
	FundingData []struct {
		Symbol         string  `json:"symbol"`
		ContractCode   string  `json:"contract_code"`
		FeeAsset       string  `json:"fee_asset"`
		FundingTime    int64   `json:"funding_time,string"`
		FundingRate    float64 `json:"funding_rate,string"`
		EstimatedRate  float64 `json:"estimated_rate,string"`
		SettlementTime int64   `json:"settlement_time,string"`
	} `json:"data"`
}

// SwapWsSubContractInfo stores funding rate data for swap websocket
type SwapWsSubContractInfo struct {
	Operation    string `json:"op"`
	Topic        string `json:"topic"`
	Timestamp    int64  `json:"ts"`
	Event        string `json:"event"`
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
		ContractCode   string  `json:"contract_code"`
		IndexPrice     float64 `json:"index_price"`
		IndexTimestamp int64   `json:"index_ts"`
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
		Timestamp int64       `json:"ts"`
		Version   int64       `json:"version"`
	} `json:"tick"`
}

// SwapKlineData stores kline data for perpetual swaps
type SwapKlineData struct {
	Data []struct {
		Volume float64 `json:"vol"`
		Close  float64 `json:"close"`
		Count  float64 `json:"count"`
		High   float64 `json:"high"`
		ID     int64   `json:"id"`
		Low    float64 `json:"low"`
		Open   float64 `json:"open"`
		Amount float64 `json:"amount"`
	} `json:"data"`
}

// MarketOverviewData stores market overview data
type MarketOverviewData struct {
	Channel string `json:"ch"`
	Tick    struct {
		Vol       float64   `json:"vol,string"`
		Ask       []float64 `json:"ask"`
		Bid       []float64 `json:"bid"`
		Close     float64   `json:"close,string"`
		Count     float64   `json:"count"`
		High      float64   `json:"high,string"`
		ID        int64     `json:"id"`
		Low       float64   `json:"low,string"`
		Open      float64   `json:"open,string"`
		Timestamp int64     `json:"ts"`
		Amount    float64   `json:"amount,string"`
	} `json:"tick"`
}

// LastTradeData stores last trade's data of a contract
type LastTradeData struct {
	Ch   string `json:"ch"`
	Tick struct {
		Data []struct {
			Amount    float64 `json:"amount,string"`
			Direction string  `json:"direction"`
			ID        int64   `json:"id"`
			Price     float64 `json:"price,string"`
			Timestamp int64   `json:"ts"`
		} `json:"data"`
	} `json:"tick"`
}

// BatchTradesData stores batch trades for a given swap contract
type BatchTradesData struct {
	Channel string `json:"ch"`
	Data    []struct {
		ID        int64 `json:"id"`
		Timestamp int64 `json:"ts"`
		Data      []struct {
			Amount    float64 `json:"amount"`
			Direction string  `json:"direction"`
			ID        int64   `json:"id"`
			Price     float64 `json:"price"`
			Timestamp int64   `json:"ts"`
		} `json:"data"`
	} `json:"data"`
}

// InsuranceAndClawbackData stores insurance fund's and clawback rate's data
type InsuranceAndClawbackData struct {
	Timestamp string `json:"timestamp"`
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
			InsuranceFund float64 `json:"insurance_fund"`
			Timestamp     int64   `json:"ts"`
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
			Volume     float64 `json:"volume"`
			AmountType float64 `json:"amountType"`
			Timestamp  int64   `json:"ts"`
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
			BuyRatio    float64 `json:"buy_ratio"`
			SellRatio   float64 `json:"sell_ratio"`
			LockedRatio float64 `json:"locked_ratio"`
			Timestamp   int64   `json:"ts"`
		} `json:"list"`
	} `json:"data"`
}

// TraderSentimentIndexPositionData stores trader sentiment index data
type TraderSentimentIndexPositionData struct {
	Data struct {
		Symbol       string `json:"symbol"`
		ContractCode string `json:"contract_code"`
		List         []struct {
			BuyRatio  float64 `json:"buy_ratio"`
			SellRatio float64 `json:"sell_ratio"`
			Timestamp int64   `json:"ts"`
		} `json:"list"`
	} `json:"data"`
}

// LiquidationOrdersData stores data of liquidation orders
type LiquidationOrdersData struct {
	Data struct {
		Orders []struct {
			Symbol       string  `json:"symbol"`
			ContractCode string  `json:"contract_code"`
			Direction    string  `json:"buy"`
			Offset       string  `json:"offset"`
			Volume       float64 `json:"volume"`
			Price        float64 `json:"price"`
			CreatedAt    int64   `json:"created_at"`
		} `json:"orders"`
		TotalPage   int64 `json:"totalPage"`
		CurrentPage int64 `json:"current_page"`
		TotalSize   int64 `json:"total_size"`
	} `json:"data"`
}

// FundingRatesData stores funding rates data
type FundingRatesData struct {
	EstimatedRate   float64 `json:"estimated_rate,string"`
	FundingRate     float64 `json:"funding_rate,string"`
	ContractCode    string  `json:"contractCode"`
	Symbol          string  `json:"symbol"`
	FeeAsset        string  `json:"fee_asset"`
	FundingTime     string  `json:"fundingTime"`
	NextFundingTime string  `json:"next_funding_time"`
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
	FundingRate     float64 `json:"funding_rate,string"`
	RealizedRate    float64 `json:"realized_rate,string"`
	FundingTime     int64   `json:"fundingTime,string"`
	ContractCode    string  `json:"contract_code"`
	Symbol          string  `json:"symbol"`
	FeeAsset        string  `json:"fee_asset"`
	AvgPremiumIndex float64 `json:"avg_premium_index,string"`
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
	Timestamp int64 `json:"ts"`
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
	Timestamp int64 `json:"ts"`
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
	Timestamp int64 `json:"ts"`
}

// SwapAccountInformation stores swap account information
type SwapAccountInformation struct {
	Data []struct {
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
	Timestamp int64 `json:"ts"`
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
	Timestamp int64 `json:"ts"`
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
	Timestamp int64 `json:"ts"`
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
	} `json:"data"`
}

// SingleSubAccountPositionsInfo stores single subaccount's positions data
type SingleSubAccountPositionsInfo struct {
	Timestamp int64 `json:"ts"`
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
	Timestamp int64 `json:"timestamp"`
}

// FinancialRecordData stores an accounts financial records
type FinancialRecordData struct {
	Data struct {
		FinancialRecord []struct {
			ID           int64   `json:"id"`
			Timestamp    int64   `json:"ts"`
			Symbol       string  `json:"symbol"`
			ContractCode string  `json:"contract_code"`
			OrderType    int64   `json:"type"`
			Amount       float64 `json:"amount"`
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
	Timestamp int64 `json:"ts"`
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
	Timestamp int64 `json:"ts"`
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
	Timestamp int64 `json:"timestamp"`
}

// PositionLimitData stores position limit data
type PositionLimitData struct {
	Data []struct {
		Symbol       string  `json:"symbol"`
		ContractCode string  `json:"contract_code"`
		BuyLimit     float64 `json:"buy_limit"`
		SellLimit    float64 `json:"sell_limit"`
	} `json:"data"`
	Timestamp int64 `json:"ts"`
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
	Timestamp int64 `json:"ts"`
	Data      struct {
		TransferRecord []struct {
			ID             int64   `json:"id"`
			Timestamp      int64   `json:"ts"`
			Symbol         string  `json:"symbol"`
			SubUID         int64   `json:"sub_uid"`
			SubAccountName string  `json:"sub_account_name"`
			TransferType   int64   `json:"transfer_type"`
			Amount         float64 `json:"amount"`
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
	Timestamp int64 `json:"ts"`
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
	Timestamp int64 `json:"ts"`
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
	Successes string `json:"successes"`
	Timestamp int64  `json:"ts"`
}

// LightningCloseOrderData stores order data from a lightning close order
type LightningCloseOrderData struct {
	Data struct {
		OrderID       int64  `json:"order_id"`
		OrderIDString string `json:"order_id_str"`
		ClientOrderID int64  `json:"client_order_id"`
	}
	Timestamp int64 `json:"ts"`
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
	Timestamp int64 `json:"ts"`
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
	Timestamp int64 `json:"ts"`
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
	Timestamp int64 `json:"ts"`
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
	Timestamp int64 `json:"ts"`
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
	Timestamp int64 `json:"ts"`
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
	Timestamp int64 `json:"ts"`
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
	Timestamp int64 `json:"ts"`
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
	Timestamp int64 `json:"ts"`
}

// TransferMarginBetweenAccountsData stores margin transfer data between spot and swap accounts
type TransferMarginBetweenAccountsData struct {
	Code    int64  `json:"code"`
	Data    int64  `json:"data"`
	Message string `json:"message"`
	Success bool   `json:"success"`
}

// --------------------------------Spot-----------------------------------------

// Response stores the Huobi response information
type Response struct {
	Status       string `json:"status"`
	Channel      string `json:"ch"`
	Timestamp    int64  `json:"ts"`
	ErrorCode    string `json:"err-code"`
	ErrorMessage string `json:"err-msg"`
}

// MarginRatesData stores margin rates data
type MarginRatesData struct {
	Data []struct {
		Symbol     string `json:"symbol"`
		Currencies []struct {
			Currency       string  `json:"currency"`
			InterestRate   float64 `json:"interestRate,string"`
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
	Symbol         string  `json:"symbol"`
	ContractCode   string  `json:"contract_code"`
	ContractSize   float64 `json:"contract_size"`
	PriceTick      float64 `json:"price_tick"`
	SettlementDate string  `json:"settlement_date"`
	CreateDate     string  `json:"create_date"`
	ContractStatus int64   `json:"contract_status"`
}

// KlineItem stores a kline item
type KlineItem struct {
	ID     int64   `json:"id"`
	Open   float64 `json:"open"`
	Close  float64 `json:"close"`
	Low    float64 `json:"low"`
	High   float64 `json:"high"`
	Amount float64 `json:"amount"`
	Volume float64 `json:"vol"`
	Count  int     `json:"count"`
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

// Ticker latest ticker data
type Ticker struct {
	Amount float64 `json:"amount"`
	Close  float64 `json:"close"`
	Count  int64   `json:"count"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Open   float64 `json:"open"`
	Symbol string  `json:"symbol"`
	Volume float64 `json:"vol"`
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
	Symbol string                         `json:"symbol"` // Required; example LTCBTC,BTCUSDT
	Type   OrderBookDataRequestParamsType `json:"type"`   // step0, step1, step2, step3, step4, step5 (combined depth 0-5); when step0, no depth is merged
}

// Orderbook stores the orderbook data
type Orderbook struct {
	ID         int64       `json:"id"`
	Timetstamp int64       `json:"ts"`
	Bids       [][]float64 `json:"bids"`
	Asks       [][]float64 `json:"asks"`
}

// Trade stores the trade data
type Trade struct {
	TradeID   float64 `json:"trade-id"`
	Price     float64 `json:"price"`
	Amount    float64 `json:"amount"`
	Direction string  `json:"direction"`
	Timestamp int64   `json:"ts"`
}

// TradeHistory stores the the trade history data
type TradeHistory struct {
	ID        int64   `json:"id"`
	Timestamp int64   `json:"ts"`
	Trades    []Trade `json:"data"`
}

// Detail stores the ticker detail data
type Detail struct {
	Amount    float64 `json:"amount"`
	Open      float64 `json:"open"`
	Close     float64 `json:"close"`
	High      float64 `json:"high"`
	Timestamp int64   `json:"timestamp"`
	ID        int64   `json:"id"`
	Count     int     `json:"count"`
	Low       float64 `json:"low"`
	Volume    float64 `json:"vol"`
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
	Currency string  `json:"currency"`
	Type     string  `json:"type"`
	Balance  float64 `json:"balance,string"`
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
		OrderID      int64  `json:"order-id,string"`
		ErrorCode    string `json:"err-code"`
		ErrorMessage string `json:"err-msg"`
	} `json:"failed"`
}

// OrderInfo stores the order info
type OrderInfo struct {
	ID               int64   `json:"id"`
	Symbol           string  `json:"symbol"`
	AccountID        int64   `json:"account-id"`
	Amount           float64 `json:"amount,string"`
	Price            float64 `json:"price,string"`
	CreatedAt        int64   `json:"created-at"`
	Type             string  `json:"type"`
	FieldAmount      float64 `json:"field-amount,string"`
	FieldCashAmount  float64 `json:"field-cash-amount,string"`
	FilledAmount     float64 `json:"filled-amount,string"`
	FilledCashAmount float64 `json:"filled-cash-amount,string"`
	FilledFees       float64 `json:"filled-fees,string"`
	FinishedAt       int64   `json:"finished-at"`
	UserID           int64   `json:"user-id"`
	Source           string  `json:"source"`
	State            string  `json:"state"`
	CanceledAt       int64   `json:"canceled-at"`
	Exchange         string  `json:"exchange"`
	Batch            string  `json:"batch"`
}

// OrderMatchInfo stores the order match info
type OrderMatchInfo struct {
	ID           int    `json:"id"`
	OrderID      int    `json:"order-id"`
	MatchID      int    `json:"match-id"`
	Symbol       string `json:"symbol"`
	Type         string `json:"type"`
	Source       string `json:"source"`
	Price        string `json:"price"`
	FilledAmount string `json:"filled-amount"`
	FilledFees   string `json:"filled-fees"`
	CreatedAt    int64  `json:"created-at"`
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

// SpotNewOrderRequestParams holds the params required to place
// an order
type SpotNewOrderRequestParams struct {
	AccountID int                           `json:"account-id,string"` // Account ID, obtained using the accounts method. Curency trades use the accountid of the ‘spot’ account; for loan asset transactions, please use the accountid of the ‘margin’ account.
	Amount    float64                       `json:"amount"`            // The limit price indicates the quantity of the order, the market price indicates how much to buy when the order is paid, and the market price indicates how much the coin is sold when the order is sold.
	Price     float64                       `json:"price"`             // Order price, market price does not use  this parameter
	Source    string                        `json:"source"`            // Order source, api: API call, margin-api: loan asset transaction
	Symbol    string                        `json:"symbol"`            // The symbol to use; example btcusdt, bccbtc......
	Type      SpotNewOrderRequestParamsType `json:"type"`              // 订单类型, buy-market: 市价买, sell-market: 市价卖, buy-limit: 限价买, sell-limit: 限价卖
}

// DepositAddress stores the users deposit address info
type DepositAddress struct {
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

	// SpotNewOrderRequestTypeSellLimit sell lmit order
	SpotNewOrderRequestTypeSellLimit = SpotNewOrderRequestParamsType("sell-limit")
)

//-----------

// KlinesRequestParams represents Klines request data.
type KlinesRequestParams struct {
	Symbol string // Symbol to be used; example btcusdt, bccbtc......
	Period string // Kline time interval; 1min, 5min, 15min......
	Size   int    // Size; [1-2000]
}

// WsRequest defines a request data structure
type WsRequest struct {
	Topic       string `json:"req,omitempty"`
	Subscribe   string `json:"sub,omitempty"`
	Unsubscribe string `json:"unsub,omitempty"`
	ClientID    int64  `json:"cid,string,omitempty"`
}

// WsResponse defines a response from the websocket connection when there
// is an error
type WsResponse struct {
	Op     string `json:"op"`
	TS     int64  `json:"ts"`
	Status string `json:"status"`
	// ErrorCode returns either an integer or a string
	ErrorCode    interface{} `json:"err-code"`
	ErrorMessage string      `json:"err-msg"`
	Ping         int64       `json:"ping"`
	Channel      string      `json:"ch"`
	Rep          string      `json:"rep"`
	Topic        string      `json:"topic"`
	Subscribed   string      `json:"subbed"`
	UnSubscribed string      `json:"unsubbed"`
	ClientID     int64       `json:"cid,string"`
}

// WsHeartBeat defines a heartbeat request
type WsHeartBeat struct {
	ClientNonce int64 `json:"ping"`
}

// WsDepth defines market depth websocket response
type WsDepth struct {
	Channel   string `json:"ch"`
	Timestamp int64  `json:"ts"`
	Tick      struct {
		Bids      [][]interface{} `json:"bids"`
		Asks      [][]interface{} `json:"asks"`
		Timestamp int64           `json:"ts"`
		Version   int64           `json:"version"`
	} `json:"tick"`
}

// WsKline defines market kline websocket response
type WsKline struct {
	Channel   string `json:"ch"`
	Timestamp int64  `json:"ts"`
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
	Channel   string `json:"ch"`
	Rep       string `json:"rep"`
	Timestamp int64  `json:"ts"`
	Tick      struct {
		Amount    float64 `json:"amount"`
		Close     float64 `json:"close"`
		Count     float64 `json:"count"`
		High      float64 `json:"high"`
		ID        float64 `json:"id"`
		Low       float64 `json:"low"`
		Open      float64 `json:"open"`
		Timestamp float64 `json:"ts"`
		Volume    float64 `json:"vol"`
	} `json:"tick"`
}

// WsTrade defines market trade websocket response
type WsTrade struct {
	Channel   string `json:"ch"`
	Timestamp int64  `json:"ts"`
	Tick      struct {
		ID        int64 `json:"id"`
		Timestamp int64 `json:"ts"`
		Data      []struct {
			Amount    float64 `json:"amount"`
			Timestamp int64   `json:"ts"`
			TradeID   float64 `json:"tradeId"`
			Price     float64 `json:"price"`
			Direction string  `json:"direction"`
		} `json:"data"`
	}
}

// WsAuthenticationRequest data for login
type WsAuthenticationRequest struct {
	Op               string `json:"op"`
	AccessKeyID      string `json:"AccessKeyId"`
	SignatureMethod  string `json:"SignatureMethod"`
	SignatureVersion string `json:"SignatureVersion"`
	Timestamp        string `json:"Timestamp"`
	Signature        string `json:"Signature"`
	ClientID         int64  `json:"cid,string,omitempty"`
}

// WsMessage defines read data from the websocket connection
type WsMessage struct {
	Raw []byte
	URL string
}

// WsAuthenticatedSubscriptionRequest request for subscription on authenticated connection
type WsAuthenticatedSubscriptionRequest struct {
	Op               string `json:"op"`
	AccessKeyID      string `json:"AccessKeyId"`
	SignatureMethod  string `json:"SignatureMethod"`
	SignatureVersion string `json:"SignatureVersion"`
	Timestamp        string `json:"Timestamp"`
	Signature        string `json:"Signature"`
	Topic            string `json:"topic"`
	ClientID         int64  `json:"cid,string,omitempty"`
}

// WsAuthenticatedAccountsListRequest request for account list authenticated connection
type WsAuthenticatedAccountsListRequest struct {
	Op               string `json:"op"`
	AccessKeyID      string `json:"AccessKeyId"`
	SignatureMethod  string `json:"SignatureMethod"`
	SignatureVersion string `json:"SignatureVersion"`
	Timestamp        string `json:"Timestamp"`
	Signature        string `json:"Signature"`
	Topic            string `json:"topic"`
	Symbol           string `json:"symbol"`
	ClientID         int64  `json:"cid,string,omitempty"`
}

// WsAuthenticatedOrderDetailsRequest request for order details authenticated connection
type WsAuthenticatedOrderDetailsRequest struct {
	Op               string `json:"op"`
	AccessKeyID      string `json:"AccessKeyId"`
	SignatureMethod  string `json:"SignatureMethod"`
	SignatureVersion string `json:"SignatureVersion"`
	Timestamp        string `json:"Timestamp"`
	Signature        string `json:"Signature"`
	Topic            string `json:"topic"`
	OrderID          string `json:"order-id"`
	ClientID         int64  `json:"cid,string,omitempty"`
}

// WsAuthenticatedOrdersListRequest request for orderslist authenticated connection
type WsAuthenticatedOrdersListRequest struct {
	Op               string `json:"op"`
	AccessKeyID      string `json:"AccessKeyId"`
	SignatureMethod  string `json:"SignatureMethod"`
	SignatureVersion string `json:"SignatureVersion"`
	Timestamp        string `json:"Timestamp"`
	Signature        string `json:"Signature"`
	Topic            string `json:"topic"`
	States           string `json:"states"`
	AccountID        int64  `json:"account-id"`
	Symbol           string `json:"symbol"`
	ClientID         int64  `json:"cid,string,omitempty"`
}

// WsAuthenticatedAccountsResponse response from Accounts authenticated subscription
type WsAuthenticatedAccountsResponse struct {
	WsResponse
	Data WsAuthenticatedAccountsResponseData `json:"data"`
}

// WsAuthenticatedAccountsResponseData account data
type WsAuthenticatedAccountsResponseData struct {
	Event string                                    `json:"event"`
	List  []WsAuthenticatedAccountsResponseDataList `json:"list"`
}

// WsAuthenticatedAccountsResponseDataList detailed account data
type WsAuthenticatedAccountsResponseDataList struct {
	AccountID int64   `json:"account-id"`
	Currency  string  `json:"currency"`
	Type      string  `json:"type"`
	Balance   float64 `json:"balance,string"`
}

// WsAuthenticatedOrdersUpdateResponse response from OrdersUpdate authenticated subscription
type WsAuthenticatedOrdersUpdateResponse struct {
	WsResponse
	Data WsAuthenticatedOrdersUpdateResponseData `json:"data"`
}

// WsAuthenticatedOrdersUpdateResponseData order  update data
type WsAuthenticatedOrdersUpdateResponseData struct {
	UnfilledAmount   float64 `json:"unfilled-amount,string"`
	FilledAmount     float64 `json:"filled-amount,string"`
	Price            float64 `json:"price,string"`
	OrderID          int64   `json:"order-id"`
	Symbol           string  `json:"symbol"`
	MatchID          int64   `json:"match-id"`
	FilledCashAmount float64 `json:"filled-cash-amount,string"`
	Role             string  `json:"role"`
	OrderState       string  `json:"order-state"`
	OrderType        string  `json:"order-type"`
}

// WsAuthenticatedOrdersResponse response from Orders authenticated subscription
type WsAuthenticatedOrdersResponse struct {
	WsResponse
	Data []WsAuthenticatedOrdersResponseData `json:"data"`
}

// WsOldOrderUpdate response from Orders authenticated subscription
type WsOldOrderUpdate struct {
	WsResponse
	Data WsAuthenticatedOrdersResponseData `json:"data"`
}

// WsAuthenticatedOrdersResponseData order data
type WsAuthenticatedOrdersResponseData struct {
	SeqID            int64   `json:"seq-id"`
	OrderID          int64   `json:"order-id"`
	Symbol           string  `json:"symbol"`
	AccountID        int64   `json:"account-id"`
	OrderAmount      float64 `json:"order-amount,string"`
	OrderPrice       float64 `json:"order-price,string"`
	CreatedAt        int64   `json:"created-at"`
	OrderType        string  `json:"order-type"`
	OrderSource      string  `json:"order-source"`
	OrderState       string  `json:"order-state"`
	Role             string  `json:"role"`
	Price            float64 `json:"price,string"`
	FilledAmount     float64 `json:"filled-amount,string"`
	UnfilledAmount   float64 `json:"unfilled-amount,string"`
	FilledCashAmount float64 `json:"filled-cash-amount,string"`
	FilledFees       float64 `json:"filled-fees,string"`
}

// WsAuthenticatedAccountsListResponse response from AccountsList authenticated endpoint
type WsAuthenticatedAccountsListResponse struct {
	WsResponse
	Data []WsAuthenticatedAccountsListResponseData `json:"data"`
}

// WsAuthenticatedAccountsListResponseData account data
type WsAuthenticatedAccountsListResponseData struct {
	ID    int64                                         `json:"id"`
	Type  string                                        `json:"type"`
	State string                                        `json:"state"`
	List  []WsAuthenticatedAccountsListResponseDataList `json:"list"`
}

// WsAuthenticatedAccountsListResponseDataList detailed account data
type WsAuthenticatedAccountsListResponseDataList struct {
	Currency string  `json:"currency"`
	Type     string  `json:"type"`
	Balance  float64 `json:"balance,string"`
}

// WsAuthenticatedOrdersListResponse response from OrdersList authenticated endpoint
type WsAuthenticatedOrdersListResponse struct {
	WsResponse
	Data []OrderInfo `json:"data"`
}

// WsAuthenticatedOrderDetailResponse response from OrderDetail authenticated endpoint
type WsAuthenticatedOrderDetailResponse struct {
	WsResponse
	Data OrderInfo `json:"data"`
}

// WsPong sent for pong messages
type WsPong struct {
	Pong int64 `json:"pong"`
}

type wsKlineResponse struct {
	Data []struct {
		Amount float64 `json:"amount"`
		Close  float64 `json:"close"`
		Count  float64 `json:"count"`
		High   float64 `json:"high"`
		ID     int64   `json:"id"`
		Low    float64 `json:"low"`
		Open   float64 `json:"open"`
		Volume float64 `json:"vol"`
	} `json:"data"`
	Rep    string `json:"rep"`
	Status string `json:"status"`
}

type authenticationPing struct {
	OP string `json:"op"`
	TS int64  `json:"ts"`
}

// OrderVars stores side, status and type for any order/trade
type OrderVars struct {
	Side      order.Side
	Status    order.Status
	OrderType order.Type
	Fee       float64
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

	validContractTypes = []string{
		"this_week", "next_week", "quarter", "next_quarter",
	}

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
