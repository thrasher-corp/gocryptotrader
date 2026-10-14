package htx

import (
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
		ID     int64        `json:"id"`
		Volume types.Number `json:"vol"`
		Count  types.Number `json:"count"`
		Open   types.Number `json:"open"`
		Close  types.Number `json:"close"`
		Low    types.Number `json:"low"`
		High   types.Number `json:"high"`
		Amount types.Number `json:"amount"`
	} `json:"tick"`
}

// SwapWsReqEstimatedFunding stores requested estimated funding data for swap websocket
type SwapWsReqEstimatedFunding struct {
	Rep       string     `json:"rep"`
	ID        string     `json:"id"`
	WsID      int64      `json:"wsid"`
	Timestamp types.Time `json:"ts"`
	Data      []struct {
		Volume types.Number `json:"vol"`
		Count  types.Number `json:"count"`
		ID     int64        `json:"id"`
		Open   types.Number `json:"open"`
		Close  types.Number `json:"close"`
		Low    types.Number `json:"low"`
		High   types.Number `json:"high"`
		Amount types.Number `json:"amount"`
	}
}

// SwapWsSubBasisData stores subscribed basis data for swap websocket
type SwapWsSubBasisData struct {
	Channel   string     `json:"ch"`
	Timestamp types.Time `json:"ts"`
	Tick      []struct {
		ID            int64        `json:"id"`
		ContractPrice types.Number `json:"contract_price"`
		IndexPrice    types.Number `json:"index_price"`
		Basis         types.Number `json:"basis"`
		BasisRate     types.Number `json:"basis_rate"`
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
		Symbol         string       `json:"symbol"`
		ContractCode   string       `json:"contract_code"`
		FeeAsset       string       `json:"fee_asset"`
		FundingTime    types.Time   `json:"funding_time"`
		FundingRate    types.Number `json:"funding_rate"`
		EstimatedRate  types.Number `json:"estimated_rate"`
		SettlementTime types.Time   `json:"settlement_time"`
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
