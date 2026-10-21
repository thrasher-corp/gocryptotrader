package htx

import (
	"bytes"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// FuturesContractInfoResponse gets contract info data for futures
type FuturesContractInfoResponse struct {
	Data []FuturesContractInfo `json:"data"`
}

// FuturesContractIndexPriceInfoResponse stores contract index price
type FuturesContractIndexPriceInfoResponse struct {
	Data      []FuturesContractIndexPriceInfo `json:"data"`
	Timestamp types.Time                      `json:"ts"`
}

// FuturesContractOpenInterestResponse stores open interest data for futures contracts
type FuturesContractOpenInterestResponse struct {
	Data      []FuturesContractOpenInterest `json:"data"`
	Timestamp types.Time                    `json:"ts"`
}

// FuturesContractOpenInterest stores open interest data for futures contracts.
type FuturesContractOpenInterest struct {
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

// FuturesEstimatedDeliveryPriceResponse stores estimated delivery price data for futures
type FuturesEstimatedDeliveryPriceResponse struct {
	Data      FuturesEstimatedDeliveryPrice `json:"data"`
	Timestamp types.Time                    `json:"ts"`
}

// FuturesMarketDepthResponse gets orderbook data for futures
type FuturesMarketDepthResponse struct {
	Channel   string                 `json:"ch"`
	Timestamp types.Time             `json:"ts"`
	Tick      FuturesMarketDepthTick `json:"tick"`
}

// FuturesOrderbook stores market depth data
type FuturesOrderbook struct {
	Symbol string
	Asks   []obItem
	Bids   []obItem
}

type obItem struct {
	Price    float64
	Quantity float64
}

// FuturesKlineResponse stores kline data for futures
type FuturesKlineResponse struct {
	Channel   string         `json:"ch"`
	Data      []FuturesKline `json:"data"`
	Timestamp types.Time     `json:"ts"`
}

// FuturesKline stores the common delivery and perpetual futures candlestick fields.
type FuturesKline struct {
	Volume      float64    `json:"vol"`
	Close       float64    `json:"close"`
	Count       float64    `json:"count"`
	High        float64    `json:"high"`
	IDTimestamp types.Time `json:"id"`
	Low         float64    `json:"low"`
	Open        float64    `json:"open"`
	Amount      float64    `json:"amount"`
}

// FuturesMarketOverviewResponse stores overview data for futures
type FuturesMarketOverviewResponse struct {
	Channel   string                    `json:"ch"`
	Tick      FuturesMarketOverviewTick `json:"tick"`
	Timestamp types.Time                `json:"ts"`
}

// FuturesLastTradeResponse stores last trade's data for a contract
type FuturesLastTradeResponse struct {
	Channel   string               `json:"ch"`
	Tick      FuturesLastTradeTick `json:"tick"`
	Timestamp types.Time           `json:"ts"`
}

// FuturesBatchTradesForContractResponse stores batch of trades data for a contract
type FuturesBatchTradesForContractResponse struct {
	Channel   string                          `json:"ch"`
	Timestamp types.Time                      `json:"ts"`
	Data      []FuturesBatchTradesForContract `json:"data"`
}

// FuturesTrade is futures trade data
type FuturesTrade struct {
	Amount    float64    `json:"amount"`
	Direction string     `json:"direction"`
	ID        uint64     `json:"id"`
	Price     float64    `json:"price"`
	Timestamp types.Time `json:"ts"`
}

// FuturesTieredAdjustmentFactorInfoResponse stores info on adjustment factor for futures
type FuturesTieredAdjustmentFactorInfoResponse struct {
	Data      []FuturesTieredAdjustmentFactorInfo `json:"data"`
	Timestamp types.Time                          `json:"ts"`
}

// FuturesOpenInterestResponse gets oi data on futures
type FuturesOpenInterestResponse struct {
	Data      FuturesOpenInterest `json:"data"`
	Timestamp types.Time          `json:"ts"`
}

// FuturesTopAccountsLongShortRatioResponse stores long/short ratio for top futures accounts
type FuturesTopAccountsLongShortRatioResponse struct {
	Data      FuturesTopAccountsLongShortRatio `json:"data"`
	Timestamp types.Time                       `json:"ts"`
}

// FuturesTopPositionsLongShortRatioResponse stores long short ratio for top futures positions
type FuturesTopPositionsLongShortRatioResponse struct {
	Data      FuturesTopPositionsLongShortRatio `json:"data"`
	Timestamp types.Time                        `json:"timestamp"`
}

// FuturesIndexKlineResponse stores index kline data for futures
type FuturesIndexKlineResponse struct {
	Channel   string              `json:"ch"`
	Data      []FuturesIndexKline `json:"data"`
	Timestamp types.Time          `json:"ts"`
}

// FuturesBasisResponse stores basis data for futures
type FuturesBasisResponse struct {
	Channel   string         `json:"ch"`
	Data      []FuturesBasis `json:"data"`
	Timestamp types.Time     `json:"ts"`
}

// FuturesUserAccountResponse stores user account data info for futures
type FuturesUserAccountResponse struct {
	AccData   []FuturesUserAccount `json:"data"`
	Timestamp types.Time           `json:"ts"`
}

// FuturesUsersPositionsInfoResponse stores positions data for futures
type FuturesUsersPositionsInfoResponse struct {
	PosInfo   []FuturesPosition `json:"data"`
	Timestamp types.Time        `json:"ts"`
}

// FuturesSubAccountAssetsInfoResponse gets subaccounts asset data
type FuturesSubAccountAssetsInfoResponse struct {
	Timestamp types.Time                    `json:"ts"`
	Data      []FuturesSubAccountAssetsInfo `json:"data"`
}

// FuturesSingleSubAccountAssetsInfoResponse stores futures assets info for a single subaccount
type FuturesSingleSubAccountAssetsInfoResponse struct {
	AssetsData []FuturesSingleSubAccountAsset `json:"data"`
	Timestamp  types.Time                     `json:"ts"`
}

// FuturesSingleSubAccountPositionsInfoResponse stores futures positions' info for a single subaccount
type FuturesSingleSubAccountPositionsInfoResponse struct {
	PositionsData []FuturesSingleSubAccountPosition `json:"data"`
	Timestamp     types.Time                        `json:"ts"`
}

// FuturesFinancialRecord stores a financial record entry for futures.
type FuturesFinancialRecord struct {
	QueryID      uint64     `json:"query_id"`
	ID           uint64     `json:"id"`
	Timestamp    types.Time `json:"ts"`
	Symbol       string     `json:"symbol"`
	ContractCode string     `json:"contract_code"`
	RecordType   uint64     `json:"type"`
	Amount       float64    `json:"amount"`
}

// FuturesFinancialRecords stores financial record data and legacy pagination values.
type FuturesFinancialRecords struct {
	FinancialRecord []FuturesFinancialRecord `json:"financial_record"`
	TotalPage       uint64                   `json:"total_page"`
	CurrentPage     uint64                   `json:"current_page"`
	TotalSize       uint64                   `json:"total_size"`
}

// FuturesFinancialRecordsResponse stores financial records data for futures.
type FuturesFinancialRecordsResponse struct {
	Data      FuturesFinancialRecords `json:"data"`
	Timestamp types.Time              `json:"ts"`
}

// UnmarshalJSON supports the documented v3 array response while preserving the
// legacy paged-object shape used by older responses.
func (f *FuturesFinancialRecordsResponse) UnmarshalJSON(data []byte) error {
	var response FuturesFinancialRecordsResponse
	if err := unmarshalV3FuturesResponse(data, &response.Timestamp, &response.Data.FinancialRecord, &response.Data); err != nil {
		return err
	}
	*f = response
	return nil
}

// FuturesSettlementRecordsResponse stores user's futures settlement records
type FuturesSettlementRecordsResponse struct {
	Data      FuturesSettlementRecords `json:"data"`
	Timestamp types.Time               `json:"ts"`
}

// FuturesContractOrderLimitResponse stores contract info on futures order limit
type FuturesContractOrderLimitResponse struct {
	ContractData []FuturesContractOrderLimit `json:"data"`
	Timestamp    types.Time                  `json:"ts"`
}

// FuturesContractTradingFeeResponse stores contract trading fee data
type FuturesContractTradingFeeResponse struct {
	ContractTradingFeeData []FuturesContractTradingFee `json:"data"`
	Timestamp              types.Time                  `json:"ts"`
}

// FuturesTransferLimitResponse stores transfer limit data for futures
type FuturesTransferLimitResponse struct {
	Data      []FuturesTransferLimit `json:"data"`
	Timestamp types.Time             `json:"ts"`
}

// FuturesPositionLimitResponse stores information on futures positions limit
type FuturesPositionLimitResponse struct {
	Data      []FuturesPositionLimit `json:"data"`
	Timestamp types.Time             `json:"ts"`
}

// FuturesAssetsAndPositionsResponse stores assets and positions data for futures
type FuturesAssetsAndPositionsResponse struct {
	Data []FuturesAssetsAndPositions `json:"data"`
}

// FuturesAccountTransferResponse stores internal transfer data for futures
type FuturesAccountTransferResponse struct {
	Status    string                 `json:"status"`
	Timestamp types.Time             `json:"ts"`
	Data      FuturesAccountTransfer `json:"data"`
}

// FuturesTransferRecordsResponse gets transfer records data
type FuturesTransferRecordsResponse struct {
	Timestamp types.Time             `json:"ts"`
	Data      FuturesTransferRecords `json:"data"`
}

// FuturesAvailableLeverageResponse stores available leverage data for futures
type FuturesAvailableLeverageResponse struct {
	Data      []FuturesAvailableLeverage `json:"data"`
	Timestamp types.Time                 `json:"timestamp"`
}

// FuturesSwitchLeverageRequest defines a delivery-contract leverage change.
type FuturesSwitchLeverageRequest struct {
	Symbol       string `json:"symbol"`
	LeverageRate uint64 `json:"lever_rate"`
}

// FuturesSwitchLeverageResponse contains the leverage accepted by HTX.
type FuturesSwitchLeverageResponse struct {
	Response
	Data FuturesLeverage `json:"data"`
}

// FuturesOrderResponse stores order data for futures
type FuturesOrderResponse struct {
	Data      FuturesOrder `json:"data"`
	Timestamp types.Time   `json:"ts"`
}

// FuturesBatchOrderRequest contains one order in HTX's batch placement request.
type FuturesBatchOrderRequest struct {
	Symbol         string  `json:"symbol"`
	ContractType   string  `json:"contract_type"`
	ContractCode   string  `json:"contract_code"`
	ClientOrderID  string  `json:"client_order_id"`
	Price          float64 `json:"price"`
	Volume         float64 `json:"volume"`
	Direction      string  `json:"direction"`
	Offset         string  `json:"offset"`
	LeverageRate   uint64  `json:"leverRate"`
	OrderPriceType string  `json:"orderPriceType"`
}

// FuturesBatchOrderResponse stores batch order data
type FuturesBatchOrderResponse struct {
	OrdersData []FuturesOrderResponse `json:"orders_data"`
}

// FuturesCancelOrderResponse stores cancel order data
type FuturesCancelOrderResponse struct {
	Data      FuturesCancelOrder `json:"data"`
	Timestamp types.Time         `json:"ts"`
}

// FuturesOrderInfoResponse stores order info
type FuturesOrderInfoResponse struct {
	Data      []FuturesOrderInfo `json:"data"`
	Timestamp types.Time         `json:"timestamp"`
}

// FuturesOrderDetailsResponse stores order details for futures orders
type FuturesOrderDetailsResponse struct {
	Data      FuturesOrderDetails `json:"data"`
	Timestamp types.Time          `json:"ts"`
}

// FuturesOpenOrdersResponse stores open orders data for futures
type FuturesOpenOrdersResponse struct {
	Data      FuturesOpenOrders `json:"data"`
	Timestamp types.Time        `json:"ts"`
}

// FuturesOrderHistoryEntry stores an order history entry for futures.
type FuturesOrderHistoryEntry struct {
	QueryID         uint64               `json:"query_id"`
	Symbol          string               `json:"symbol"`
	ContractType    string               `json:"contract_type"`
	ContractCode    string               `json:"contract_code"`
	Volume          float64              `json:"volume"`
	Price           float64              `json:"price"`
	OrderPriceType  LegacyOrderPriceType `json:"order_price_type"`
	Direction       string               `json:"direction"`
	Offset          string               `json:"offset"`
	LeverageRate    uint64               `json:"lever_rate"`
	OrderID         uint64               `json:"order_id"`
	OrderIDString   string               `json:"order_id_str"`
	OrderSource     string               `json:"order_source"`
	CreateDate      types.Time           `json:"create_date"`
	UpdateTime      types.Time           `json:"update_time"`
	TradeVolume     float64              `json:"trade_volume"`
	TradeTurnover   float64              `json:"trade_turnover"`
	Fee             float64              `json:"fee"`
	TradeAvgPrice   float64              `json:"trade_avg_price"`
	MarginFrozen    float64              `json:"margin_frozen"`
	Profit          float64              `json:"profit"`
	Status          uint64               `json:"status"`
	OrderType       uint64               `json:"order_type"`
	FeeAsset        currency.Code        `json:"fee_asset"`
	LiquidationType types.Number         `json:"liquidation_type"`
}

// FuturesOrderHistory stores order history data and legacy pagination values.
type FuturesOrderHistory struct {
	Orders      []FuturesOrderHistoryEntry `json:"orders"`
	TotalPage   uint64                     `json:"total_page"`
	CurrentPage uint64                     `json:"current_page"`
	TotalSize   uint64                     `json:"total_size"`
}

// FuturesOrderHistoryResponse stores order history data.
type FuturesOrderHistoryResponse struct {
	Data      FuturesOrderHistory `json:"data"`
	Timestamp types.Time          `json:"ts"`
}

// UnmarshalJSON supports the documented v3 array response while preserving the
// legacy paged-object shape used by older responses.
func (f *FuturesOrderHistoryResponse) UnmarshalJSON(data []byte) error {
	var response FuturesOrderHistoryResponse
	if err := unmarshalV3FuturesResponse(data, &response.Timestamp, &response.Data.Orders, &response.Data); err != nil {
		return err
	}
	*f = response
	return nil
}

// FuturesTradeHistoryEntry stores a trade history entry for futures.
type FuturesTradeHistoryEntry struct {
	QueryID       uint64        `json:"query_id"`
	ID            string        `json:"id"`
	ContractCode  string        `json:"contract_code"`
	ContractType  string        `json:"contract_type"`
	CreateDate    types.Time    `json:"create_date"`
	Direction     string        `json:"direction"`
	MatchID       uint64        `json:"match_id"`
	Offset        string        `json:"offset"`
	OffsetPNL     float64       `json:"offset_profitloss"`
	OrderID       uint64        `json:"order_id"`
	OrderIDString string        `json:"order_id_str"`
	Symbol        string        `json:"symbol"`
	OrderSource   string        `json:"order_source"`
	TradeFee      float64       `json:"trade_fee"`
	TradePrice    float64       `json:"trade_price"`
	TradeTurnover float64       `json:"trade_turnover"`
	TradeVolume   float64       `json:"trade_volume"`
	Role          string        `json:"role"`
	FeeAsset      currency.Code `json:"fee_asset"`
}

// FuturesTradeHistory stores trade history data and legacy pagination values.
type FuturesTradeHistory struct {
	TotalPage   uint64                     `json:"total_page"`
	CurrentPage uint64                     `json:"current_page"`
	TotalSize   uint64                     `json:"total_size"`
	Trades      []FuturesTradeHistoryEntry `json:"trades"`
}

// FuturesTradeHistoryResponse stores trade history data for futures.
type FuturesTradeHistoryResponse struct {
	Data      FuturesTradeHistory `json:"data"`
	Timestamp types.Time          `json:"ts"`
}

// UnmarshalJSON supports the documented v3 array response while preserving the
// legacy paged-object shape used by older responses.
func (f *FuturesTradeHistoryResponse) UnmarshalJSON(data []byte) error {
	var response FuturesTradeHistoryResponse
	if err := unmarshalV3FuturesResponse(data, &response.Timestamp, &response.Data.Trades, &response.Data); err != nil {
		return err
	}
	*f = response
	return nil
}

// unmarshalV3FuturesResponse preserves HTX's legacy paginated response shape
// while accepting the array and empty-data forms returned by current V3 APIs.
func unmarshalV3FuturesResponse[T, P any](data []byte, timestamp *types.Time, arrayResponse *[]T, legacyResponse *P) error {
	var raw futuresV3Response
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*timestamp = raw.Timestamp
	if isEmptyHTXData(raw.Data) {
		return nil
	}
	if bytes.HasPrefix(bytes.TrimSpace(raw.Data), []byte("[")) {
		return json.Unmarshal(raw.Data, arrayResponse)
	}
	return json.Unmarshal(raw.Data, legacyResponse)
}

// FuturesTriggerOrderResponse stores trigger order data
type FuturesTriggerOrderResponse struct {
	Data      FuturesTriggerOrder `json:"data"`
	Timestamp types.Time          `json:"ts"`
}

// FuturesTriggerOpenOrdersResponse stores trigger open orders data
type FuturesTriggerOpenOrdersResponse struct {
	Data      FuturesTriggerOpenOrders `json:"data"`
	Timestamp types.Time               `json:"ts"`
}

// FuturesTriggerOrderHistoryResponse stores trigger order history for futures
type FuturesTriggerOrderHistoryResponse struct {
	Data      FuturesTriggerOrderHistory `json:"data"`
	Timestamp types.Time                 `json:"ts"`
}

// FuturesContractInfo describes the size, price tick and lifecycle of a delivery contract.
type FuturesContractInfo struct {
	Symbol         string     `json:"symbol"`
	ContractCode   string     `json:"contract_code"`
	ContractType   string     `json:"contract_type"`
	ContractSize   float64    `json:"contract_size"`
	PriceTick      float64    `json:"price_tick"`
	DeliveryDate   types.Time `json:"delivery_date"`
	DeliveryTime   types.Time `json:"delivery_time"`
	CreateDate     types.Time `json:"create_date"`
	ContractStatus uint64     `json:"contract_status"`
	SettlementTime types.Time `json:"settlement_time"`
}

// FuturesContractIndexPriceInfo pairs an underlying symbol with its current index price.
type FuturesContractIndexPriceInfo struct {
	Symbol     string  `json:"symbol"`
	IndexPrice float64 `json:"index_price"`
}

// FuturesEstimatedDeliveryPrice is the estimated settlement price as a contract approaches delivery.
type FuturesEstimatedDeliveryPrice struct {
	DeliveryPrice float64 `json:"delivery_price"`
}

// FuturesMarketDepthTick is a depth snapshot whose levels contain price and contract quantity.
type FuturesMarketDepthTick struct {
	MRID      uint64       `json:"mrid"`
	ID        uint64       `json:"id"`
	Bids      [][2]float64 `json:"bids"`
	Asks      [][2]float64 `json:"asks"`
	Timestamp types.Time   `json:"ts"`
	Version   uint64       `json:"version"`
	Channel   string       `json:"ch"`
}

// FuturesMarketOverviewTick combines the rolling delivery summary with the best bid and ask.
type FuturesMarketOverviewTick struct {
	Vol       types.Number `json:"vol"`
	Ask       []float64    `json:"ask"`
	Bid       []float64    `json:"bid"`
	Close     types.Number `json:"close"`
	Count     float64      `json:"count"`
	High      types.Number `json:"high"`
	ID        uint64       `json:"id"`
	Low       types.Number `json:"low"`
	Open      types.Number `json:"open"`
	Timestamp types.Time   `json:"ts"`
	Amount    types.Number `json:"amount"`
}

// FuturesLastTrade is a recent delivery execution; quoted and bare prices are both accepted.
type FuturesLastTrade struct {
	Amount    types.Number `json:"amount"`
	Direction string       `json:"direction"`
	ID        uint64       `json:"id"`
	Price     types.Number `json:"price"`
	Timestamp types.Time   `json:"ts"`
}

// FuturesLastTradeTick groups recent executions under their exchange update ID.
type FuturesLastTradeTick struct {
	Data      []FuturesLastTrade `json:"data"`
	ID        uint64             `json:"id"`
	Timestamp types.Time         `json:"ts"`
}

// FuturesBatchTradesForContract groups the delivery executions published at one exchange timestamp.
type FuturesBatchTradesForContract struct {
	ID        uint64         `json:"id"`
	Timestamp types.Time     `json:"ts"`
	Data      []FuturesTrade `json:"data"`
}

// FuturesTieredAdjustmentFactorInfoLadders defines the position-size interval for one margin adjustment factor.
type FuturesTieredAdjustmentFactorInfoLadders struct {
	Ladder       uint64  `json:"ladder"`
	MinSize      float64 `json:"min_size"`
	MaxSize      float64 `json:"max_size"`
	AdjustFactor float64 `json:"adjust_factor"`
}

// FuturesTieredAdjustmentFactorInfoList groups margin adjustment tiers by leverage.
type FuturesTieredAdjustmentFactorInfoList struct {
	LeverageRate float64                                    `json:"lever_rate"`
	Ladders      []FuturesTieredAdjustmentFactorInfoLadders `json:"ladders"`
}

// FuturesTieredAdjustmentFactorInfo groups the available margin adjustment tiers for an underlying symbol.
type FuturesTieredAdjustmentFactorInfo struct {
	Symbol string                                  `json:"symbol"`
	List   []FuturesTieredAdjustmentFactorInfoList `json:"list"`
}

// FuturesOpenInterestTick records outstanding contract volume at an exchange timestamp.
type FuturesOpenInterestTick struct {
	Volume     types.Number `json:"volume"`
	AmountType uint64       `json:"amount_type"`
	Timestamp  types.Time   `json:"ts"`
}

// FuturesOpenInterest groups historical open interest by symbol and expiry type.
type FuturesOpenInterest struct {
	Symbol       string                    `json:"symbol"`
	ContractType string                    `json:"contract_type"`
	Tick         []FuturesOpenInterestTick `json:"tick"`
}

// FuturesTopAccountsLongShortRatioList records the proportions of long, short and locked accounts at a sample time.
type FuturesTopAccountsLongShortRatioList struct {
	BuyRatio    float64    `json:"buy_ratio"`
	SellRatio   float64    `json:"sell_ratio"`
	LockedRatio float64    `json:"locked_ratio"`
	Timestamp   types.Time `json:"ts"`
}

// FuturesTopAccountsLongShortRatio groups account sentiment samples for an underlying symbol.
type FuturesTopAccountsLongShortRatio struct {
	List   []FuturesTopAccountsLongShortRatioList `json:"list"`
	Symbol string                                 `json:"symbol"`
}

// FuturesTopPositionsLongShortRatioList records long and short position proportions at a sample time.
type FuturesTopPositionsLongShortRatioList struct {
	BuyRatio  float64    `json:"buy_ratio"`
	SellRatio float64    `json:"sell_ratio"`
	Timestamp types.Time `json:"timestamp"`
}

// FuturesTopPositionsLongShortRatio groups position sentiment samples for an underlying symbol.
type FuturesTopPositionsLongShortRatio struct {
	Symbol string                                  `json:"symbol"`
	List   []FuturesTopPositionsLongShortRatioList `json:"list"`
}

// FuturesIndexKline is an index-price candle whose ID is its start time in Unix seconds.
type FuturesIndexKline struct {
	Vol    float64 `json:"vol"`
	Close  float64 `json:"close"`
	Count  float64 `json:"count"`
	High   float64 `json:"high"`
	ID     uint64  `json:"id"`
	Low    float64 `json:"low"`
	Open   float64 `json:"open"`
	Amount float64 `json:"amount"`
}

// FuturesBasis compares the delivery contract price with its index; basis values may be negative.
type FuturesBasis struct {
	Basis         types.Number `json:"basis"`
	BasisRate     types.Number `json:"basis_rate"`
	ContractPrice types.Number `json:"contract_price"`
	ID            uint64       `json:"id"`
	IndexPrice    types.Number `json:"index_price"`
}

// FuturesUserAccount reports margin, available collateral and profit for a delivery settlement currency.
type FuturesUserAccount struct {
	Symbol            currency.Code `json:"symbol"`
	MarginBalance     float64       `json:"margin_balance"`
	MarginPosition    float64       `json:"margin_position"`
	MarginFrozen      float64       `json:"margin_frozen"`
	MarginAvailable   float64       `json:"margin_available"`
	ProfitReal        float64       `json:"profit_real"`
	ProfitUnreal      float64       `json:"profit_unreal"`
	RiskRate          float64       `json:"risk_rate"`
	LiquidationPrice  float64       `json:"liquidation_price"`
	WithdrawAvailable float64       `json:"withdraw_available"`
	LeverageRate      float64       `json:"lever_rate"`
	AdjustFactor      float64       `json:"adjust_factor"`
	MarginStatic      float64       `json:"margin_static"`
}

// FuturesPosition describes a delivery position with its entry costs, margin and available contracts.
type FuturesPosition struct {
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
	LeverageRate   uint64  `json:"lever_rate"`
	Direction      string  `json:"direction"`
	LastPrice      float64 `json:"last_price"`
}

// FuturesSubAccountAsset reports a subaccount balance and liquidation risk for one settlement currency.
type FuturesSubAccountAsset struct {
	Symbol           string  `json:"symbol"`
	MarginBalance    float64 `json:"margin_balance"`
	LiquidationPrice float64 `json:"liquidation_price"`
	RiskRate         float64 `json:"risk_rate"`
}

// FuturesSubAccountAssetsInfo groups balances by subaccount identifier.
type FuturesSubAccountAssetsInfo struct {
	SubUID uint64                   `json:"sub_uid"`
	List   []FuturesSubAccountAsset `json:"list"`
}

// FuturesSingleSubAccountAsset reports the collateral and profit components of a subaccount balance.
type FuturesSingleSubAccountAsset struct {
	Symbol            currency.Code `json:"symbol"`
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
}

// FuturesSingleSubAccountPosition describes one subaccount delivery position.
type FuturesSingleSubAccountPosition struct {
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
	LeverageRate   uint64  `json:"lever_rate"`
	Direction      string  `json:"direction"`
	LastPrice      float64 `json:"last_price"`
}

// FuturesSettlementPosition records position costs and profit at an account settlement.
type FuturesSettlementPosition struct {
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
}

// FuturesSettlementRecord records the balance and position values used for one account settlement.
type FuturesSettlementRecord struct {
	Symbol               string                      `json:"symbol"`
	MarginBalanceInit    float64                     `json:"margin_balance_init"`
	MarginBalance        float64                     `json:"margin_balance"`
	SettlementProfitReal float64                     `json:"settlement_profit_real"`
	SettlementTime       types.Time                  `json:"settlement_time"`
	Clawback             float64                     `json:"clawback"`
	DeliveryFee          float64                     `json:"delivery_fee"`
	OffsetProfitLoss     float64                     `json:"offset_profitloss"`
	Fee                  float64                     `json:"fee"`
	FeeAsset             currency.Code               `json:"fee_asset"`
	Positions            []FuturesSettlementPosition `json:"positions"`
}

// FuturesSettlementRecords is a page of account settlements with the server pagination totals.
type FuturesSettlementRecords struct {
	SettlementRecords []FuturesSettlementRecord `json:"settlement_records"`
	CurrentPage       uint64                    `json:"current_page"`
	TotalPage         uint64                    `json:"total_page"`
	TotalSize         uint64                    `json:"total_size"`
}

// FuturesContractTypeOrderLimit sets opening and closing order limits for an expiry type.
type FuturesContractTypeOrderLimit struct {
	ContractType string `json:"contract_type"`
	OpenLimit    uint64 `json:"open_limit"`
	CloseLimit   uint64 `json:"close_limit"`
}

// FuturesSymbolOrderLimit groups expiry-specific order limits for an underlying symbol.
type FuturesSymbolOrderLimit struct {
	Symbol        string                          `json:"symbol"`
	ContractTypes []FuturesContractTypeOrderLimit `json:"types"`
}

// FuturesContractOrderLimit groups order limits by price instruction.
type FuturesContractOrderLimit struct {
	OrderPriceType string                    `json:"order_price_type"`
	List           []FuturesSymbolOrderLimit `json:"list"`
}

// FuturesContractTradingFee lists maker, taker and delivery fees in their settlement currency.
type FuturesContractTradingFee struct {
	Symbol        string        `json:"symbol"`
	OpenMakerFee  types.Number  `json:"open_maker_fee"`
	OpenTakerFee  types.Number  `json:"open_taker_fee"`
	CloseMakerFee types.Number  `json:"close_maker_fee"`
	CloseTakerFee types.Number  `json:"close_taker_fee"`
	DeliveryFee   types.Number  `json:"delivery_fee"`
	FeeAsset      currency.Code `json:"fee_asset"`
}

// FuturesTransferLimit sets per-transfer and daily limits for moving delivery collateral.
type FuturesTransferLimit struct {
	Symbol                 string  `json:"symbol"`
	MaxTransferIn          float64 `json:"transfer_in_max_each"`
	MinTransferIn          float64 `json:"transfer_in_min_each"`
	MaxTransferOut         float64 `json:"transfer_out_max_each"`
	MinTransferOut         float64 `json:"transfer_out_min_each"`
	MaxTransferInDaily     float64 `json:"transfer_in_max_daily"`
	MaxTransferOutDaily    float64 `json:"transfer_out_max_daily"`
	NetTransferInMaxDaily  float64 `json:"net_transfer_in_max_daily"`
	NetTransferOutMaxDaily float64 `json:"net_transfer_out_max_daily"`
}

// FuturesContractPositionLimit sets long and short position limits for an expiry type.
type FuturesContractPositionLimit struct {
	ContractType string  `json:"contract_type"`
	BuyLimit     float64 `json:"buy_limit"`
	SellLimit    float64 `json:"sell_limit"`
}

// FuturesPositionLimit groups expiry-specific position limits for an underlying symbol.
type FuturesPositionLimit struct {
	Symbol string                         `json:"symbol"`
	List   []FuturesContractPositionLimit `json:"list"`
}

// FuturesAssetsAndPositions reports collateral, margin and profit totals for a delivery account.
type FuturesAssetsAndPositions struct {
	Symbol            string  `json:"symbol"`
	MarginBalance     float64 `json:"margin_balance"`
	MarginPosition    float64 `json:"margin_position"`
	MarginFrozen      float64 `json:"margin_frozen"`
	MarginAvailable   float64 `json:"margin_available"`
	ProfitReal        float64 `json:"profit_real"`
	ProfitUnreal      float64 `json:"profit_unreal"`
	RiskRate          float64 `json:"risk_rate"`
	WithdrawAvailable float64 `json:"withdraw_available"`
}

// FuturesAccountTransfer identifies a transfer between a master account and subaccount.
type FuturesAccountTransfer struct {
	OrderID string `json:"order_id"`
}

// FuturesTransferRecord records a collateral transfer and its participating subaccount.
type FuturesTransferRecord struct {
	ID             uint64     `json:"id"`
	Timestamp      types.Time `json:"ts"`
	Symbol         string     `json:"symbol"`
	SubUID         uint64     `json:"sub_uid"`
	SubAccountName string     `json:"sub_account_name"`
	TransferType   uint64     `json:"transfer_type"`
	Amount         float64    `json:"amount"`
}

// FuturesTransferRecords is a page of collateral transfers with the server pagination totals.
type FuturesTransferRecords struct {
	TransferRecord []FuturesTransferRecord `json:"transfer_record"`
	TotalPage      uint64                  `json:"total_page"`
	CurrentPage    uint64                  `json:"current_page"`
	TotalSize      uint64                  `json:"total_size"`
}

// FuturesAvailableLeverage lists the leverage levels available for an underlying symbol.
type FuturesAvailableLeverage struct {
	Symbol                string `json:"symbol"`
	AvailableLeverageRate string `json:"available_level_rate"`
}

// FuturesLeverage reports the leverage selected for an underlying symbol.
type FuturesLeverage struct {
	Symbol       string `json:"symbol"`
	LeverageRate uint64 `json:"lever_rate"`
}

// FuturesOrder identifies an accepted delivery order using exchange and client identifiers.
type FuturesOrder struct {
	OrderID       uint64 `json:"order_id"`
	OrderIDStr    string `json:"order_id_str"`
	ClientOrderID uint64 `json:"client_order_id"`
}

// FuturesCancelOrderError identifies an order rejected within a cancellation request.
type FuturesCancelOrderError struct {
	OrderID uint64 `json:"order_id,string"`
	ErrCode int64  `json:"err_code"`
	ErrMsg  string `json:"err_msg"`
}

// FuturesCancelOrder retains successful identifiers and individual failures from a cancellation request.
type FuturesCancelOrder struct {
	Errors    []FuturesCancelOrderError `json:"errors"`
	Successes string                    `json:"successes"`
}

// FuturesOrderInfo combines delivery order state with its execution and fee totals.
type FuturesOrderInfo struct {
	ClientOrderID   uint64        `json:"client_order_id"`
	ContractCode    string        `json:"contract_code"`
	ContractType    string        `json:"contract_type"`
	CreatedAt       types.Time    `json:"created_at"`
	CanceledAt      types.Time    `json:"canceled_at"`
	Direction       string        `json:"direction"`
	Fee             float64       `json:"fee"`
	FeeAsset        currency.Code `json:"fee_asset"`
	LeverRate       uint64        `json:"lever_rate"`
	MarginFrozen    float64       `json:"margin_frozen"`
	Offset          string        `json:"offset"`
	OrderID         uint64        `json:"order_id"`
	OrderIDString   string        `json:"order_id_str"`
	OrderPriceType  string        `json:"order_price_type"`
	OrderSource     string        `json:"order_source"`
	OrderType       uint64        `json:"order_type"`
	Price           float64       `json:"price"`
	Profit          float64       `json:"profit"`
	Status          uint64        `json:"status"`
	Symbol          string        `json:"symbol"`
	TradeAvgPrice   float64       `json:"trade_avg_price"`
	TradeTurnover   float64       `json:"trade_turnover"`
	TradeVolume     float64       `json:"trade_volume"`
	Volume          float64       `json:"volume"`
	LiquidationType types.Number  `json:"liquidation_type"`
}

// FuturesOrderTrade records an individual fill and its fee within a delivery order.
type FuturesOrderTrade struct {
	ID            string     `json:"id"`
	TradeID       uint64     `json:"trade_id"`
	TradeVolume   float64    `json:"trade_volume"`
	TradePrice    float64    `json:"trade_price"`
	TradeFee      float64    `json:"trade_fee"`
	TradeTurnover float64    `json:"trade_turnover"`
	Role          string     `json:"role"`
	CreatedAt     types.Time `json:"created_at"`
}

// FuturesOrderDetails combines a delivery order with its individual fills and pagination totals.
type FuturesOrderDetails struct {
	Symbol         string              `json:"symbol"`
	ContractType   string              `json:"contract_type"`
	ContractCode   string              `json:"contract_code"`
	Volume         float64             `json:"volume"`
	Price          float64             `json:"price"`
	OrderPriceType string              `json:"order_price_type"`
	Direction      string              `json:"direction"`
	Offset         string              `json:"offset"`
	LeverageRate   uint64              `json:"lever_rate"`
	MarginFrozen   float64             `json:"margin_frozen"`
	Profit         float64             `json:"profit"`
	OrderSource    string              `json:"order_source"`
	OrderID        uint64              `json:"order_id"`
	OrderIDString  string              `json:"order_id_str"`
	ClientOrderID  uint64              `json:"client_order_id"`
	OrderType      uint64              `json:"order_type"`
	Status         uint64              `json:"status"`
	TradeVolume    float64             `json:"trade_volume"`
	TradeTurnover  float64             `json:"trade_turnover"`
	TradeAvgPrice  float64             `json:"trade_avg_price"`
	Fee            float64             `json:"fee"`
	CreatedAt      types.Time          `json:"created_at"`
	CanceledAt     types.Time          `json:"canceled_at"`
	FinalInterest  float64             `json:"final_interest"`
	AdjustValue    float64             `json:"adjust_value"`
	FeeAsset       currency.Code       `json:"fee_asset"`
	Trades         []FuturesOrderTrade `json:"trades"`
	TotalPage      uint64              `json:"total_page"`
	TotalSize      uint64              `json:"total_size"`
	CurrentPage    uint64              `json:"current_page"`
}

// FuturesOpenOrder describes an unfilled or partially filled delivery order.
type FuturesOpenOrder struct {
	Symbol         string        `json:"symbol"`
	ContractType   string        `json:"contract_type"`
	ContractCode   string        `json:"contract_code"`
	Volume         float64       `json:"volume"`
	Price          float64       `json:"price"`
	OrderPriceType string        `json:"order_price_type"`
	OrderType      uint64        `json:"order_type"`
	Direction      string        `json:"direction"`
	Offset         string        `json:"offset"`
	LeverageRate   uint64        `json:"lever_rate"`
	OrderID        uint64        `json:"order_id"`
	OrderIDString  string        `json:"order_id_str"`
	ClientOrderID  uint64        `json:"client_order_id"`
	OrderSource    string        `json:"order_source"`
	CreatedAt      types.Time    `json:"created_at"`
	TradeVolume    float64       `json:"trade_volume"`
	Fee            float64       `json:"fee"`
	TradeAvgPrice  float64       `json:"trade_avg_price"`
	MarginFrozen   float64       `json:"margin_frozen"`
	Profit         float64       `json:"profit"`
	Status         uint64        `json:"status"`
	FeeAsset       currency.Code `json:"fee_asset"`
}

// FuturesOpenOrders is a page of unfilled or partially filled delivery orders.
type FuturesOpenOrders struct {
	Orders      []FuturesOpenOrder `json:"orders"`
	TotalPage   uint64             `json:"total_page"`
	CurrentPage uint64             `json:"current_page"`
	TotalSize   uint64             `json:"total_size"`
}

// FuturesTriggerOrder identifies an accepted trigger order before it creates a regular order.
type FuturesTriggerOrder struct {
	OrderID    uint64 `json:"order_id"`
	OrderIDStr string `json:"order_id_str"`
}

// FuturesOpenTriggerOrder describes a trigger order still awaiting its price condition.
type FuturesOpenTriggerOrder struct {
	Symbol         string     `json:"symbol"`
	ContractCode   string     `json:"contract_code"`
	ContractType   string     `json:"contract_type"`
	TriggerType    string     `json:"trigger_type"`
	Volume         float64    `json:"volume"`
	OrderType      uint64     `json:"order_type"`
	Direction      string     `json:"direction"`
	Offset         string     `json:"offset"`
	LeverageRate   uint64     `json:"lever_rate"`
	OrderID        uint64     `json:"order_id"`
	OrderIDString  string     `json:"order_id_str"`
	OrderSource    string     `json:"order_source"`
	TriggerPrice   float64    `json:"trigger_price"`
	OrderPrice     float64    `json:"order_price"`
	CreatedAt      types.Time `json:"created_at"`
	OrderPriceType string     `json:"order_price_type"`
	Status         uint64     `json:"status"`
}

// FuturesTriggerOpenOrders is a page of trigger orders still awaiting their price conditions.
type FuturesTriggerOpenOrders struct {
	Orders      []FuturesOpenTriggerOrder `json:"orders"`
	TotalPage   uint64                    `json:"total_page"`
	CurrentPage uint64                    `json:"current_page"`
	TotalSize   uint64                    `json:"total_size"`
}

// FuturesTriggerOrderHistoryEntry records a trigger lifecycle and any resulting order or failure.
type FuturesTriggerOrderHistoryEntry struct {
	Symbol          string     `json:"symbol"`
	ContractCode    string     `json:"contract_code"`
	ContractType    string     `json:"contract_type"`
	TriggerType     string     `json:"trigger_type"`
	Volume          float64    `json:"volume"`
	OrderType       uint64     `json:"order_type"`
	Direction       string     `json:"direction"`
	Offset          string     `json:"offset"`
	LeverageRate    uint64     `json:"lever_rate"`
	OrderID         uint64     `json:"order_id"`
	OrderIDString   string     `json:"order_id_str"`
	RelationOrderID string     `json:"relation_order_id"`
	OrderPriceType  string     `json:"order_price_type"`
	Status          string     `json:"status"`
	OrderSource     string     `json:"order_source"`
	TriggerPrice    float64    `json:"trigger_price"`
	TriggeredPrice  float64    `json:"triggered_price"`
	OrderPrice      float64    `json:"order_price"`
	CreatedAt       types.Time `json:"created_at"`
	TriggeredAt     types.Time `json:"triggered_at"`
	OrderInsertAt   types.Time `json:"order_insert_at"`
	CancelledAt     types.Time `json:"canceled_at"`
	FailCode        int64      `json:"fail_code"`
	FailReason      string     `json:"fail_reason"`
}

// FuturesTriggerOrderHistory is a page of completed or cancelled trigger orders.
type FuturesTriggerOrderHistory struct {
	Orders      []FuturesTriggerOrderHistoryEntry `json:"orders"`
	TotalPage   uint64                            `json:"total_page"`
	CurrentPage uint64                            `json:"current_page"`
	TotalSize   uint64                            `json:"total_size"`
}

// futuresV3Response retains the data shape until the API version is identified.
type futuresV3Response struct {
	Data      json.RawMessage `json:"data"`
	Timestamp types.Time      `json:"ts"`
}
