package huobi

import "github.com/thrasher-corp/gocryptotrader/types"

// FContractInfoData gets contract info data for futures
type FContractInfoData struct {
	Data []struct {
		Symbol         string     `json:"symbol"`
		ContractCode   string     `json:"contract_code"`
		ContractType   string     `json:"contract_type"`
		ContractSize   float64    `json:"contract_size"`
		PriceTick      float64    `json:"price_tick"`
		DeliveryDate   string     `json:"delivery_date"`
		DeliveryTime   types.Time `json:"delivery_time"`
		CreateDate     string     `json:"create_date"`
		ContractStatus int64      `json:"contract_status"`
		SettlementTime types.Time `json:"settlement_time"`
	}
}

// FContractIndexPriceInfo stores contract index price
type FContractIndexPriceInfo struct {
	Data []struct {
		Symbol     string  `json:"symbol"`
		IndexPrice float64 `json:"index_price"`
	} `json:"data"`
	Timestamp types.Time `json:"ts"`
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
	Timestamp types.Time `json:"ts"`
}

// FContractOIData stores open interest data for futures contracts
type FContractOIData struct {
	Data      []UContractOpenInterest `json:"data"`
	Timestamp types.Time              `json:"ts"`
}

// UContractOpenInterest stores open interest data for futures contracts
type UContractOpenInterest struct {
	Volume        float64 `json:"volume"`
	Amount        float64 `json:"amount"`
	Symbol        string  `json:"symbol"`
	Value         float64 `json:"value"`
	ContractCode  string  `json:"contract_code"`
	TradeAmount   float64 `json:"trade_amount"`
	TradeVolume   float64 `json:"trade_volume"`
	TradeTurnover float64 `json:"trade_turnover"`
	BusinessType  string  `json:"business_type"`
	Pair          string  `json:"pair"`
	ContractType  string  `json:"contract_type"`
}

// FEstimatedDeliveryPriceInfo stores estimated delivery price data for futures
type FEstimatedDeliveryPriceInfo struct {
	Data struct {
		DeliveryPrice float64 `json:"delivery_price"`
	} `json:"data"`
	Timestamp types.Time `json:"ts"`
}

// FMarketDepth gets orderbook data for futures
type FMarketDepth struct {
	Ch        string     `json:"ch"`
	Timestamp types.Time `json:"ts"`
	Tick      struct {
		MRID      int64        `json:"mrid"`
		ID        int64        `json:"id"`
		Bids      [][2]float64 `json:"bids"`
		Asks      [][2]float64 `json:"asks"`
		Timestamp types.Time   `json:"ts"`
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
		Volume      float64    `json:"vol"`
		Close       float64    `json:"close"`
		Count       float64    `json:"count"`
		High        float64    `json:"high"`
		IDTimestamp types.Time `json:"id"`
		Low         float64    `json:"low"`
		Open        float64    `json:"open"`
		Amount      float64    `json:"amount"`
	} `json:"data"`
	Timestamp types.Time `json:"ts"`
}

// FMarketOverviewData stores overview data for futures
type FMarketOverviewData struct {
	Ch   string `json:"ch"`
	Tick struct {
		Vol       float64 `json:"vol,string"`
		Ask       [2]float64
		Bid       [2]float64
		Close     float64    `json:"close,string"`
		Count     float64    `json:"count"`
		High      float64    `json:"high,string"`
		ID        int64      `jso:"id"`
		Low       float64    `json:"low,string"`
		Open      float64    `json:"open,string"`
		Timestamp types.Time `json:"ts"`
		Amount    float64    `json:"amount,string"`
	} `json:"tick"`
	Timestamp types.Time `json:"ts"`
}

// FLastTradeData stores last trade's data for a contract
type FLastTradeData struct {
	Ch   string `json:"ch"`
	Tick struct {
		Data []struct {
			Amount    float64    `json:"amount,string"`
			Direction string     `json:"direction"`
			ID        int64      `json:"id"`
			Price     float64    `json:"price,string"`
			Timestamp types.Time `json:"ts"`
		} `json:"data"`
		ID        int64      `json:"id"`
		Timestamp types.Time `json:"ts"`
	} `json:"tick"`
	Timestamp types.Time `json:"ts"`
}

// FBatchTradesForContractData stores batch of trades data for a contract
type FBatchTradesForContractData struct {
	Ch        string     `json:"ch"`
	Timestamp types.Time `json:"ts"`
	Data      []struct {
		ID        int64          `json:"id"`
		Timestamp types.Time     `json:"ts"`
		Data      []FuturesTrade `json:"data"`
	} `json:"data"`
}

// FuturesTrade is futures trade data
type FuturesTrade struct {
	Amount    float64    `json:"amount"`
	Direction string     `json:"direction"`
	ID        int64      `json:"id"`
	Price     float64    `json:"price"`
	Timestamp types.Time `json:"ts"`
}

// FClawbackRateAndInsuranceData stores clawback rate and insurance data for futures
type FClawbackRateAndInsuranceData struct {
	Timestamp types.Time `json:"ts"`
	Data      []struct {
		Symbol            string  `json:"symbol"`
		InsuranceFund     float64 `json:"insurance_fund"`
		EstimatedClawback float64 `json:"estimated_clawback"`
	} `json:"data"`
}

// FHistoricalInsuranceRecordsData stores historical records of insurance fund balances for futures
type FHistoricalInsuranceRecordsData struct {
	Timestamp types.Time `json:"timestamp"`
	Data      struct {
		Symbol string `json:"symbol"`
		Tick   []struct {
			InsuranceFund float64    `json:"insurance_fund"`
			Timestamp     types.Time `json:"ts"`
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
	Timestamp types.Time `json:"ts"`
}

// FOIData gets oi data on futures
type FOIData struct {
	Data struct {
		Symbol       string `json:"symbol"`
		ContractType string `json:"contract_type"`
		Tick         []struct {
			Volume     float64    `json:"volume,string"`
			AmountType int64      `json:"amount_type"`
			Timestamp  types.Time `json:"ts"`
		} `json:"tick"`
	} `json:"data"`
	Timestamp types.Time `json:"ts"`
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
	Timestamp types.Time `json:"ts"`
}

// FTopAccountsLongShortRatio stores long/short ratio for top futures accounts
type FTopAccountsLongShortRatio struct {
	Data struct {
		List []struct {
			BuyRatio    float64    `json:"buy_ratio"`
			SellRatio   float64    `json:"sell_ratio"`
			LockedRatio float64    `json:"locked_ratio"`
			Timestamp   types.Time `json:"ts"`
		} `json:"list"`
		Symbol string `json:"symbol"`
	} `json:"data"`
	Timestamp types.Time `json:"ts"`
}

// FTopPositionsLongShortRatio stores long short ratio for top futures positions
type FTopPositionsLongShortRatio struct {
	Data struct {
		Symbol string `json:"symbol"`
		List   []struct {
			BuyRatio  float64    `json:"buy_ratio"`
			SellRatio float64    `json:"sell_ratio"`
			Timestamp types.Time `json:"timestamp"`
		} `json:"list"`
	} `json:"data"`
	Timestamp types.Time `json:"timestamp"`
}

// FLiquidationOrdersInfo stores data of futures liquidation orders
type FLiquidationOrdersInfo struct {
	Data struct {
		Orders []struct {
			Symbol       string     `json:"symbol"`
			ContractCode string     `json:"contract_code"`
			Direction    string     `json:"direction"`
			Offset       string     `json:"offset"`
			Volume       float64    `json:"volume"`
			Price        float64    `json:"price"`
			CreatedAt    types.Time `json:"created_at"`
		} `json:"orders"`
		TotalPage   int64 `json:"total_page"`
		CurrentPage int64 `json:"current_page"`
		TotalSize   int64 `json:"total_size"`
	} `json:"data"`
	Timestamp types.Time `json:"ts"`
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
	Timestamp types.Time `json:"ts"`
}

// FBasisData stores basis data for futures
type FBasisData struct {
	Ch   string `json:"ch"`
	Data []struct {
		Basis         float64 `json:"basis,string"`
		BasisRate     float64 `json:"basis_rate,string"`
		ContractPrice float64 `json:"contract_price,string"`
		ID            int64   `json:"id"`
		IndexPrice    float64 `json:"index_price,string"`
	} `json:"data"`
	Timestamp types.Time `json:"ts"`
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
	Timestamp types.Time `json:"ts"`
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
	Timestamp types.Time `json:"ts"`
}

// FSubAccountAssetsInfo gets subaccounts asset data
type FSubAccountAssetsInfo struct {
	Timestamp types.Time `json:"ts"`
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
	Timestamp types.Time `json:"ts"`
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
	Timestamp types.Time `json:"ts"`
}

// FFinancialRecords stores financial records data for futures
type FFinancialRecords struct {
	Data struct {
		FinancialRecord []struct {
			ID         int64      `json:"id"`
			Timestamp  types.Time `json:"ts"`
			Symbol     string     `json:"symbol"`
			RecordType int64      `json:"type"`
			Amount     float64    `json:"amount"`
		} `json:"financial_record"`
		TotalPage   int64 `json:"total_page"`
		CurrentPage int64 `json:"current_page"`
		TotalSize   int64 `json:"total_size"`
	} `json:"data"`
	Timestamp types.Time `json:"ts"`
}

// FSettlementRecords stores user's futures settlement records
type FSettlementRecords struct {
	Data struct {
		SettlementRecords []struct {
			Symbol               string     `json:"symbol"`
			MarginBalanceInit    float64    `json:"margin_balance_init"`
			MarginBalance        int64      `json:"margin_balance"`
			SettlementProfitReal float64    `json:"settlement_profit_real"`
			SettlementTime       types.Time `json:"settlement_time"`
			Clawback             float64    `json:"clawback"`
			DeliveryFee          float64    `json:"delivery_fee"`
			OffsetProfitLoss     float64    `json:"offset_profitloss"`
			Fee                  float64    `json:"fee"`
			FeeAsset             string     `json:"fee_asset"`
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
	Timestamp types.Time `json:"ts"`
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
	Timestamp types.Time `json:"ts"`
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
	Timestamp types.Time `json:"ts"`
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
	Timestamp types.Time `json:"ts"`
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
	Timestamp types.Time `json:"ts"`
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
	Status    string     `json:"status"`
	Timestamp types.Time `json:"ts"`
	Data      struct {
		OrderID string `json:"order_id"`
	} `json:"data"`
}

// FTransferRecords gets transfer records data
type FTransferRecords struct {
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

// FAvailableLeverageData stores available leverage data for futures
type FAvailableLeverageData struct {
	Data []struct {
		Symbol                string `json:"symbol"`
		AvailableLeverageRate string `json:"available_level_rate"`
	} `json:"data"`
	Timestamp types.Time `json:"timestamp"`
}

// FOrderData stores order data for futures
type FOrderData struct {
	Data struct {
		OrderID       int64  `json:"order_id"`
		OrderIDStr    string `json:"order_id_str"`
		ClientOrderID int64  `json:"client_order_id"`
	} `json:"data"`
	Timestamp types.Time `json:"ts"`
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
			OrderID int64  `json:"order_id,string"`
			ErrCode int64  `json:"err_code"`
			ErrMsg  string `json:"err_msg"`
		} `json:"errors"`
		Successes string `json:"successes"`
	} `json:"data"`
	Timestamp types.Time `json:"ts"`
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
	Timestamp types.Time `json:"timestamp"`
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
	Timestamp types.Time `json:"ts"`
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
	Timestamp types.Time `json:"ts"`
}

// FOrderHistoryData stores order history data
type FOrderHistoryData struct {
	Data struct {
		Orders []struct {
			Symbol          string     `json:"symbol"`
			ContractType    string     `json:"contract_type"`
			ContractCode    string     `json:"contract_code"`
			Volume          float64    `json:"volume"`
			Price           float64    `json:"price"`
			OrderPriceType  string     `json:"order_price_type"`
			Direction       string     `json:"direction"`
			Offset          string     `json:"offset"`
			LeverageRate    float64    `json:"lever_rate"`
			OrderID         int64      `json:"order_id"`
			OrderIDString   string     `json:"order_id_str"`
			OrderSource     string     `json:"order_source"`
			CreateDate      types.Time `json:"create_date"`
			TradeVolume     float64    `json:"trade_volume"`
			TradeTurnover   float64    `json:"trade_turnover"`
			Fee             float64    `json:"fee"`
			TradeAvgPrice   float64    `json:"trade_avg_price"`
			MarginFrozen    float64    `json:"margin_frozen"`
			Profit          float64    `json:"profit"`
			Status          int64      `json:"status"`
			OrderType       int64      `json:"order_type"`
			FeeAsset        string     `json:"fee_asset"`
			LiquidationType int64      `json:"liquidation_type"`
		} `json:"orders"`
		TotalPage   int64 `json:"total_page"`
		CurrentPage int64 `json:"current_page"`
		TotalSize   int64 `json:"total_size"`
	} `json:"data"`
	Timestamp types.Time `json:"ts"`
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
	Timestamp types.Time `json:"ts"`
}

// FTriggerOrderData stores trigger order data
type FTriggerOrderData struct {
	Data struct {
		OrderID    int64  `json:"order_id"`
		OrderIDStr string `json:"order_id_str"`
	} `json:"data"`
	Timestamp types.Time `json:"ts"`
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
	Timestamp types.Time `json:"ts"`
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
	Timestamp types.Time `json:"ts"`
}
