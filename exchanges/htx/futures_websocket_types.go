package htx

import (
	"github.com/thrasher-corp/gocryptotrader/types"
)

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
		ID     string       `json:"id"`
		Open   types.Number `json:"open"`
		Close  types.Number `json:"close"`
		High   types.Number `json:"high"`
		Low    types.Number `json:"low"`
		Amount types.Number `json:"amount"`
		Volume types.Number `json:"vol"`
		Count  types.Number `json:"count"`
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
		ID            int64        `json:"id"`
		IndexPrice    types.Number `json:"index_price"`
		ContractPrice types.Number `json:"contract_price"`
		Basis         types.Number `json:"basis"`
		BasisRate     types.Number `json:"basis_rate"`
	}
}

// FWsReqBasisData stores requested basis data for futures websocket
type FWsReqBasisData struct {
	ID        string     `json:"id"`
	Rep       string     `json:"rep"`
	Timestamp types.Time `json:"ts"`
	WsID      int64      `json:"wsid"`
	Tick      struct {
		ID            int64        `json:"id"`
		IndexPrice    types.Number `json:"index_price"`
		ContractPrice types.Number `json:"contract_price"`
		Basis         types.Number `json:"basis"`
		BasisRate     types.Number `json:"basis_rate"`
	} `json:"tick"`
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
