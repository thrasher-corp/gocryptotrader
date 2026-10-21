package htx

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// SwapWebsocketKlineResponse stores req kline data for swap websocket
type SwapWebsocketKlineResponse struct {
	Rep         string                `json:"rep"`
	ID          string                `json:"id"`
	WebsocketID uint64                `json:"wsid"`
	Data        []SwapWebsocketCandle `json:"data"`
}

// SwapWebsocketTradesResponse stores requested trade detail data for swap websocket
type SwapWebsocketTradesResponse struct {
	Rep       string               `json:"rep"`
	ID        string               `json:"id"`
	Timestamp types.Time           `json:"ts"`
	Data      []SwapWebsocketTrade `json:"data"`
}

// SwapWebsocketPremiumKline stores subscribed premium kline data for futures websocket
type SwapWebsocketPremiumKline struct {
	Channel   string                     `json:"ch"`
	Timestamp types.Time                 `json:"ts"`
	Tick      SwapWebsocketPremiumCandle `json:"tick"`
}

// SwapWebsocketPremiumKlineResponse stores requested premium kline data for futures websocket
type SwapWebsocketPremiumKlineResponse struct {
	Rep         string                       `json:"rep"`
	ID          string                       `json:"id"`
	WebsocketID uint64                       `json:"wsid"`
	Timestamp   types.Time                   `json:"ts"`
	Data        []SwapWebsocketPremiumCandle `json:"data"`
}

// SwapWebsocketEstimatedFunding stores estimated funding rate data for swap websocket
type SwapWebsocketEstimatedFunding struct {
	Channel   string                     `json:"ch"`
	Timestamp types.Time                 `json:"ts"`
	Tick      SwapWebsocketFundingCandle `json:"tick"`
}

// SwapWebsocketEstimatedFundingResponse stores requested estimated funding data for swap websocket
type SwapWebsocketEstimatedFundingResponse struct {
	Rep         string     `json:"rep"`
	ID          string     `json:"id"`
	WebsocketID uint64     `json:"wsid"`
	Timestamp   types.Time `json:"ts"`
	Data        []SwapWebsocketFundingCandle
}

// SwapWebsocketBasis stores subscribed basis data for swap websocket
type SwapWebsocketBasis struct {
	Channel   string                   `json:"ch"`
	Timestamp types.Time               `json:"ts"`
	Tick      []SwapWebsocketBasisTick `json:"tick"`
}

// SwapWebsocketBasisResponse stores requested basis data for swap websocket
type SwapWebsocketBasisResponse struct {
	Rep         string     `json:"rep"`
	ID          string     `json:"id"`
	WebsocketID uint64     `json:"wsid"`
	Timestamp   types.Time `json:"ts"`
	Data        []SwapWebsocketBasisSnapshot
}

// SwapWebsocketLiquidationOrders stores subscribed liquidation orders data for swap futures
type SwapWebsocketLiquidationOrders struct {
	Operation  string                          `json:"op"`
	Topic      string                          `json:"topic"`
	Timestamp  types.Time                      `json:"ts"`
	OrdersData []SwapWebsocketLiquidationOrder `json:"data"`
}

// SwapWebsocketFundingRates stores funding rate data for swap websocket
type SwapWebsocketFundingRates struct {
	Operation   string                     `json:"op"`
	Topic       string                     `json:"topic"`
	Timestamp   types.Time                 `json:"ts"`
	FundingData []SwapWebsocketFundingRate `json:"data"`
}

// SwapWebsocketContractInfo stores funding rate data for swap websocket
type SwapWebsocketContractInfo struct {
	Operation    string                  `json:"op"`
	Topic        string                  `json:"topic"`
	Timestamp    types.Time              `json:"ts"`
	Event        string                  `json:"event"`
	ContractData []SwapWebsocketContract `json:"data"`
}

// SwapWebsocketCandle contains a perpetual swap market kline.
type SwapWebsocketCandle struct {
	Volume float64 `json:"vol"`
	Count  float64 `json:"count"`
	ID     uint64  `json:"id"`
	Open   float64 `json:"open"`
	Close  float64 `json:"close"`
	Low    float64 `json:"low"`
	High   float64 `json:"high"`
	Amount float64 `json:"amount"`
}

// SwapWebsocketTrade contains a perpetual swap trade execution.
type SwapWebsocketTrade struct {
	ID        uint64       `json:"id"`
	Price     types.Number `json:"price"`
	Amount    types.Number `json:"amount"`
	Direction string       `json:"direction"`
	Timestamp types.Time   `json:"ts"`
}

// SwapWebsocketPremiumCandle contains a premium index kline, whose prices may be negative.
type SwapWebsocketPremiumCandle struct {
	ID     uint64       `json:"id"`
	Volume types.Number `json:"vol"`
	Count  types.Number `json:"count"`
	Open   types.Number `json:"open"`
	Close  types.Number `json:"close"`
	Low    types.Number `json:"low"`
	High   types.Number `json:"high"`
	Amount types.Number `json:"amount"`
}

// SwapWebsocketFundingCandle contains an estimated funding rate kline.
type SwapWebsocketFundingCandle struct {
	ID     uint64       `json:"id"`
	Volume types.Number `json:"vol"`
	Count  types.Number `json:"count"`
	Open   types.Number `json:"open"`
	Close  types.Number `json:"close"`
	Low    types.Number `json:"low"`
	High   types.Number `json:"high"`
	Amount types.Number `json:"amount"`
}

// SwapWebsocketBasisTick contains a contract basis subscription update.
type SwapWebsocketBasisTick struct {
	ID            uint64       `json:"id"`
	ContractPrice types.Number `json:"contract_price"`
	IndexPrice    types.Number `json:"index_price"`
	Basis         types.Number `json:"basis"`
	BasisRate     types.Number `json:"basis_rate"`
}

// SwapWebsocketBasisSnapshot contains a contract basis request result.
type SwapWebsocketBasisSnapshot struct {
	ID            uint64       `json:"id"`
	ContractPrice types.Number `json:"contract_price"`
	IndexPrice    types.Number `json:"index_price"`
	Basis         types.Number `json:"basis"`
	BasisRate     types.Number `json:"basis_rate"`
}

// SwapWebsocketLiquidationOrder contains a liquidated perpetual swap order.
type SwapWebsocketLiquidationOrder struct {
	Symbol       string     `json:"symbol"`
	ContractCode string     `json:"contract_code"`
	Direction    string     `json:"direction"`
	Offset       string     `json:"offset"`
	Volume       float64    `json:"volume"`
	Price        float64    `json:"price"`
	CreatedAt    types.Time `json:"created_at"`
}

// SwapWebsocketFundingRate contains a perpetual swap funding rate notification.
type SwapWebsocketFundingRate struct {
	Symbol         string        `json:"symbol"`
	ContractCode   string        `json:"contract_code"`
	FeeAsset       currency.Code `json:"fee_asset"`
	FundingTime    types.Time    `json:"funding_time"`
	FundingRate    types.Number  `json:"funding_rate"`
	EstimatedRate  types.Number  `json:"estimated_rate"`
	SettlementTime types.Time    `json:"settlement_time"`
}

// SwapWebsocketContract contains a perpetual swap contract definition.
type SwapWebsocketContract struct {
	Symbol         string     `json:"symbol"`
	ContractCode   string     `json:"contract_code"`
	ContractSize   float64    `json:"contract_size"`
	PriceTick      float64    `json:"price_tick"`
	SettlementDate types.Time `json:"settlement_date"`
	CreateDate     types.Time `json:"create_date"`
	ContractStatus uint64     `json:"contract_status"`
}
