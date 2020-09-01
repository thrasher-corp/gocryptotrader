package huobi

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
	Ts   int64 `json:"ts"`
	Data struct {
		OrderID string `json:"order_id"`
	} `json:"data"`
}

// InternalAccountTransferRecords stores data for transfer records within the account
type InternalAccountTransferRecords struct {
	Timestamp string `json:"ts"`
	Data      struct {
		TransferRecord []struct {
			ID             int64   `json:"id"`
			Timestamp      int64   `json:"ts"`
			Symbol         string  `json:"symbol"`
			SubUID         string  `json:"sub_uid"`
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
		OrderID       int64 `json:"order_id"`
		OrderIDString int64 `json:"order_id_string"`
		ClientOrderID int64 `json:"client_order_id"`
	} `json:"data"`
	Timestamp string `json:"ts"`
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
	Timestamp string `json:"ts"`
}

// LightningCloseOrderData stores order data from a lightning close order
type LightningCloseOrderData struct {
	Data struct {
		OrderID       int64  `json:"order_id"`
		OrderIDString string `json:"order_id_str"`
		ClientOrderID int64  `json:"client_order_id"`
	}
	Timestamp string `json:"ts"`
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
			LeverageRate   int64   `json:"lever_rate"`
			OrderID        int64   `json:"order_id"`
			OrderIDString  int64   `json:"order_id_str"`
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
			Symbol         string  `json:"symbol"`
			ContractCode   string  `json:"contract_code"`
			Volume         float64 `json:"volume"`
			Price          float64 `json:"price"`
			OrderPriceType string  `json:"order_price_type"`
			Direction      string  `json:"direction"`
			Offset         string  `json:"offset"`
			LeverageRate   float64 `json:"lever_rate"`
		} `json:"orders"`
	} `json:"data"`
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

// *******************************************************

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
	ID        float64 `json:"id"`
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
			ID        float64 `json:"id"`
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
