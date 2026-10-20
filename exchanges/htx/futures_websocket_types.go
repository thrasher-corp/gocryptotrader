package htx

import (
	"github.com/thrasher-corp/gocryptotrader/types"
)

// FuturesWebsocketKlineResponse stores requested kline data for futures websocket
type FuturesWebsocketKlineResponse struct {
	Rep         string                  `json:"rep"`
	ID          string                  `json:"id"`
	WebsocketID uint64                  `json:"wsid"`
	Tick        []FuturesWebsocketKline `json:"tick"`
}

// FuturesWebsocketTradeDetailResponse stores requested trade detail data for futures websocket
type FuturesWebsocketTradeDetailResponse struct {
	Rep       string                  `json:"rep"`
	ID        string                  `json:"id"`
	Timestamp types.Time              `json:"ts"`
	Data      []FuturesWebsocketTrade `json:"data"`
}

// FuturesWebsocketSubKlineIndex stores subscribed kline index data for futures websocket
type FuturesWebsocketSubKlineIndex struct {
	Channel   string                         `json:"ch"`
	Timestamp types.Time                     `json:"ts"`
	Tick      FuturesWebsocketIndexKlineTick `json:"tick"`
}

// FuturesWebsocketKlineIndexResponse stores requested kline index data for futures websocket
type FuturesWebsocketKlineIndexResponse struct {
	ID          string                       `json:"id"`
	Rep         string                       `json:"rep"`
	WebsocketID uint64                       `json:"wsid"`
	Timestamp   types.Time                   `json:"ts"`
	Data        []FuturesWebsocketIndexKline `json:"data"`
}

// FuturesWebsocketSubBasis stores subscribed basis data for futures websocket
type FuturesWebsocketSubBasis struct {
	Channel   string                `json:"ch"`
	Timestamp types.Time            `json:"ts"`
	Tick      FuturesWebsocketBasis `json:"tick"`
}

// FuturesWebsocketBasisResponse stores requested basis data for futures websocket
type FuturesWebsocketBasisResponse struct {
	ID          string                `json:"id"`
	Rep         string                `json:"rep"`
	Timestamp   types.Time            `json:"ts"`
	WebsocketID uint64                `json:"wsid"`
	Tick        FuturesWebsocketBasis `json:"tick"`
}

// FuturesWebsocketSubLiquidationOrders stores subscribed liquidation orders data for futures websocket
type FuturesWebsocketSubLiquidationOrders struct {
	Operation  string                             `json:"op"`
	Topic      string                             `json:"topic"`
	Timestamp  types.Time                         `json:"ts"`
	OrdersData []FuturesWebsocketLiquidationOrder `json:"data"`
}

// FuturesWebsocketSubContractInfo stores contract info data for futures websocket
type FuturesWebsocketSubContractInfo struct {
	Operation    string                         `json:"op"`
	Topic        string                         `json:"topic"`
	Timestamp    types.Time                     `json:"ts"`
	Event        string                         `json:"event"`
	ContractData []FuturesWebsocketContractInfo `json:"data"`
}

// FuturesWebsocketKline contains the documented delivery-futures websocket fields.
type FuturesWebsocketKline struct {
	Volume float64 `json:"vol"`
	Count  float64 `json:"count"`
	ID     uint64  `json:"id"`
	Open   float64 `json:"open"`
	Close  float64 `json:"close"`
	Low    float64 `json:"low"`
	High   float64 `json:"high"`
	Amount float64 `json:"amount"`
}

// FuturesWebsocketTrade contains the documented delivery-futures websocket fields.
type FuturesWebsocketTrade struct {
	ID        uint64     `json:"id"`
	Price     float64    `json:"price"`
	Amount    float64    `json:"amount"`
	Direction string     `json:"direction"`
	Timestamp types.Time `json:"ts"`
}

// FuturesWebsocketIndexKlineTick contains the documented delivery-futures websocket fields.
type FuturesWebsocketIndexKlineTick struct {
	ID     string       `json:"id"`
	Open   types.Number `json:"open"`
	Close  types.Number `json:"close"`
	High   types.Number `json:"high"`
	Low    types.Number `json:"low"`
	Amount types.Number `json:"amount"`
	Volume types.Number `json:"vol"`
	Count  types.Number `json:"count"`
}

// FuturesWebsocketIndexKline contains the documented delivery-futures websocket fields.
type FuturesWebsocketIndexKline struct {
	ID     uint64  `json:"id"`
	Open   float64 `json:"open"`
	Close  float64 `json:"close"`
	Low    float64 `json:"low"`
	High   float64 `json:"high"`
	Amount float64 `json:"amount"`
	Volume float64 `json:"vol"`
	Count  float64 `json:"count"`
}

// FuturesWebsocketBasis contains the documented delivery-futures websocket fields.
type FuturesWebsocketBasis struct {
	ID            uint64       `json:"id"`
	IndexPrice    types.Number `json:"index_price"`
	ContractPrice types.Number `json:"contract_price"`
	Basis         types.Number `json:"basis"`
	BasisRate     types.Number `json:"basis_rate"`
}

// FuturesWebsocketLiquidationOrder contains the documented delivery-futures websocket fields.
type FuturesWebsocketLiquidationOrder struct {
	Symbol       string     `json:"symbol"`
	ContractCode string     `json:"contract_code"`
	Direction    string     `json:"direction"`
	Offset       string     `json:"offset"`
	Volume       float64    `json:"volume"`
	Price        float64    `json:"price"`
	CreatedAt    types.Time `json:"created_at"`
}

// FuturesWebsocketContractInfo contains the documented delivery-futures websocket fields.
type FuturesWebsocketContractInfo struct {
	Symbol         string     `json:"symbol"`
	ContractCode   string     `json:"contract_code"`
	ContractType   string     `json:"contract_type"`
	ContractSize   float64    `json:"contract_size"`
	PriceTick      float64    `json:"price_tick"`
	DeliveryDate   types.Time `json:"delivery_date"`
	CreateDate     string     `json:"create_date"`
	ContractStatus uint64     `json:"contract_status"`
}
