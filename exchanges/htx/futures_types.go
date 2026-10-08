package htx

import (
	"bytes"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// FContractInfoData gets contract info data for futures
type FContractInfoData struct {
	Data []FContractInfoDataData `json:"data"`
}

// FContractIndexPriceInfo stores contract index price
type FContractIndexPriceInfo struct {
	Data      []FContractIndexPriceInfoData `json:"data"`
	Timestamp types.Time                    `json:"ts"`
}

// FContractOIData stores open interest data for futures contracts
type FContractOIData struct {
	Data      []FContractOpenInterest `json:"data"`
	Timestamp types.Time              `json:"ts"`
}

// FContractOpenInterest stores open interest data for futures contracts.
type FContractOpenInterest struct {
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
	Data      FEstimatedDeliveryPriceInfoData `json:"data"`
	Timestamp types.Time                      `json:"ts"`
}

// FMarketDepth gets orderbook data for futures
type FMarketDepth struct {
	Ch        string           `json:"ch"`
	Timestamp types.Time       `json:"ts"`
	Tick      FMarketDepthTick `json:"tick"`
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
	Ch        string         `json:"ch"`
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

// FMarketOverviewData stores overview data for futures
type FMarketOverviewData struct {
	Ch        string                  `json:"ch"`
	Tick      FMarketOverviewDataTick `json:"tick"`
	Timestamp types.Time              `json:"ts"`
}

// FLastTradeData stores last trade's data for a contract
type FLastTradeData struct {
	Ch        string             `json:"ch"`
	Tick      FLastTradeDataTick `json:"tick"`
	Timestamp types.Time         `json:"ts"`
}

// FBatchTradesForContractData stores batch of trades data for a contract
type FBatchTradesForContractData struct {
	Ch        string                            `json:"ch"`
	Timestamp types.Time                        `json:"ts"`
	Data      []FBatchTradesForContractDataData `json:"data"`
}

// FuturesTrade is futures trade data
type FuturesTrade struct {
	Amount    float64    `json:"amount"`
	Direction string     `json:"direction"`
	ID        int64      `json:"id"`
	Price     float64    `json:"price"`
	Timestamp types.Time `json:"ts"`
}

// FTieredAdjustmentFactorInfo stores info on adjustment factor for futures
type FTieredAdjustmentFactorInfo struct {
	Data      []FTieredAdjustmentFactorInfoData `json:"data"`
	Timestamp types.Time                        `json:"ts"`
}

// FOIData gets oi data on futures
type FOIData struct {
	Data      FOIDataData `json:"data"`
	Timestamp types.Time  `json:"ts"`
}

// FTopAccountsLongShortRatio stores long/short ratio for top futures accounts
type FTopAccountsLongShortRatio struct {
	Data      FTopAccountsLongShortRatioData `json:"data"`
	Timestamp types.Time                     `json:"ts"`
}

// FTopPositionsLongShortRatio stores long short ratio for top futures positions
type FTopPositionsLongShortRatio struct {
	Data      FTopPositionsLongShortRatioData `json:"data"`
	Timestamp types.Time                      `json:"timestamp"`
}

// FIndexKlineData stores index kline data for futures
type FIndexKlineData struct {
	Ch        string                `json:"ch"`
	Data      []FIndexKlineDataData `json:"data"`
	Timestamp types.Time            `json:"ts"`
}

// FBasisData stores basis data for futures
type FBasisData struct {
	Ch        string           `json:"ch"`
	Data      []FBasisDataData `json:"data"`
	Timestamp types.Time       `json:"ts"`
}

// FUserAccountData stores user account data info for futures
type FUserAccountData struct {
	AccData   []FUserAccountDataAccData `json:"data"`
	Timestamp types.Time                `json:"ts"`
}

// FUsersPositionsInfo stores positions data for futures
type FUsersPositionsInfo struct {
	PosInfo   []FUsersPositionsInfoPosInfo `json:"data"`
	Timestamp types.Time                   `json:"ts"`
}

// FSubAccountAssetsInfo gets subaccounts asset data
type FSubAccountAssetsInfo struct {
	Timestamp types.Time                  `json:"ts"`
	Data      []FSubAccountAssetsInfoData `json:"data"`
}

// FSingleSubAccountAssetsInfo stores futures assets info for a single subaccount
type FSingleSubAccountAssetsInfo struct {
	AssetsData []FSingleSubAccountAssetsInfoAssetsData `json:"data"`
	Timestamp  types.Time                              `json:"ts"`
}

// FSingleSubAccountPositionsInfo stores futures positions' info for a single subaccount
type FSingleSubAccountPositionsInfo struct {
	PositionsData []FSingleSubAccountPositionsInfoPositionsData `json:"data"`
	Timestamp     types.Time                                    `json:"ts"`
}

// FFinancialRecord stores a financial record entry for futures.
type FFinancialRecord struct {
	QueryID      int64      `json:"query_id"`
	ID           int64      `json:"id"`
	Timestamp    types.Time `json:"ts"`
	Symbol       string     `json:"symbol"`
	ContractCode string     `json:"contract_code"`
	RecordType   int64      `json:"type"`
	Amount       float64    `json:"amount"`
}

// FFinancialRecordsData stores financial record data and legacy pagination values.
type FFinancialRecordsData struct {
	FinancialRecord []FFinancialRecord `json:"financial_record"`
	TotalPage       int64              `json:"total_page"`
	CurrentPage     int64              `json:"current_page"`
	TotalSize       int64              `json:"total_size"`
}

// FFinancialRecords stores financial records data for futures.
type FFinancialRecords struct {
	Data      FFinancialRecordsData `json:"data"`
	Timestamp types.Time            `json:"ts"`
}

// UnmarshalJSON supports the documented v3 array response while preserving the
// legacy paged-object shape used by older responses.
func (f *FFinancialRecords) UnmarshalJSON(data []byte) error {
	var response FFinancialRecords
	if err := unmarshalV3FuturesResponse(data, &response.Timestamp, &response.Data.FinancialRecord, &response.Data); err != nil {
		return err
	}
	*f = response
	return nil
}

// FSettlementRecords stores user's futures settlement records
type FSettlementRecords struct {
	Data      FSettlementRecordsData `json:"data"`
	Timestamp types.Time             `json:"ts"`
}

// FContractInfoOnOrderLimit stores contract info on futures order limit
type FContractInfoOnOrderLimit struct {
	ContractData []FContractInfoOnOrderLimitContractData `json:"data"`
	Timestamp    types.Time                              `json:"ts"`
}

// FContractTradingFeeData stores contract trading fee data
type FContractTradingFeeData struct {
	ContractTradingFeeData []FContractTradingFeeDataContractTradingFeeData `json:"data"`
	Timestamp              types.Time                                      `json:"ts"`
}

// FTransferLimitData stores transfer limit data for futures
type FTransferLimitData struct {
	Data      []FTransferLimitDataData `json:"data"`
	Timestamp types.Time               `json:"ts"`
}

// FPositionLimitData stores information on futures positions limit
type FPositionLimitData struct {
	Data      []FPositionLimitDataData `json:"data"`
	Timestamp types.Time               `json:"ts"`
}

// FAssetsAndPositionsData stores assets and positions data for futures
type FAssetsAndPositionsData struct {
	Data []FAssetsAndPositionsDataData `json:"data"`
}

// FAccountTransferData stores internal transfer data for futures
type FAccountTransferData struct {
	Status    string                   `json:"status"`
	Timestamp types.Time               `json:"ts"`
	Data      FAccountTransferDataData `json:"data"`
}

// FTransferRecords gets transfer records data
type FTransferRecords struct {
	Timestamp types.Time           `json:"ts"`
	Data      FTransferRecordsData `json:"data"`
}

// FAvailableLeverageData stores available leverage data for futures
type FAvailableLeverageData struct {
	Data      []FAvailableLeverageDataData `json:"data"`
	Timestamp types.Time                   `json:"timestamp"`
}

// FSwitchLeverageRequest defines a delivery-contract leverage change.
type FSwitchLeverageRequest struct {
	Symbol       string `json:"symbol"`
	LeverageRate uint64 `json:"lever_rate"`
}

// FSwitchLeverageResponse contains the leverage accepted by HTX.
type FSwitchLeverageResponse struct {
	Response
	Data FSwitchLeverageResponseData `json:"data"`
}

// FOrderData stores order data for futures
type FOrderData struct {
	Data      FOrderDataData `json:"data"`
	Timestamp types.Time     `json:"ts"`
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
	Data      FCancelOrderDataData `json:"data"`
	Timestamp types.Time           `json:"ts"`
}

// FOrderInfo stores order info
type FOrderInfo struct {
	Data      []FOrderInfoData `json:"data"`
	Timestamp types.Time       `json:"timestamp"`
}

// FOrderDetailsData stores order details for futures orders
type FOrderDetailsData struct {
	Data      FOrderDetailsDataData `json:"data"`
	Timestamp types.Time            `json:"ts"`
}

// FOpenOrdersData stores open orders data for futures
type FOpenOrdersData struct {
	Data      FOpenOrdersDataData `json:"data"`
	Timestamp types.Time          `json:"ts"`
}

// FOrderHistoryEntry stores an order history entry for futures.
type FOrderHistoryEntry struct {
	QueryID         int64                `json:"query_id"`
	Symbol          string               `json:"symbol"`
	ContractType    string               `json:"contract_type"`
	ContractCode    string               `json:"contract_code"`
	Volume          float64              `json:"volume"`
	Price           float64              `json:"price"`
	OrderPriceType  LegacyOrderPriceType `json:"order_price_type"`
	Direction       string               `json:"direction"`
	Offset          string               `json:"offset"`
	LeverageRate    float64              `json:"lever_rate"`
	OrderID         int64                `json:"order_id"`
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
	Status          int64                `json:"status"`
	OrderType       int64                `json:"order_type"`
	FeeAsset        string               `json:"fee_asset"`
	LiquidationType types.Number         `json:"liquidation_type"`
}

// FOrderHistoryResponseData stores order history data and legacy pagination values.
type FOrderHistoryResponseData struct {
	Orders      []FOrderHistoryEntry `json:"orders"`
	TotalPage   int64                `json:"total_page"`
	CurrentPage int64                `json:"current_page"`
	TotalSize   int64                `json:"total_size"`
}

// FOrderHistoryData stores order history data.
type FOrderHistoryData struct {
	Data      FOrderHistoryResponseData `json:"data"`
	Timestamp types.Time                `json:"ts"`
}

// UnmarshalJSON supports the documented v3 array response while preserving the
// legacy paged-object shape used by older responses.
func (f *FOrderHistoryData) UnmarshalJSON(data []byte) error {
	var response FOrderHistoryData
	if err := unmarshalV3FuturesResponse(data, &response.Timestamp, &response.Data.Orders, &response.Data); err != nil {
		return err
	}
	*f = response
	return nil
}

// FTradeHistoryEntry stores a trade history entry for futures.
type FTradeHistoryEntry struct {
	QueryID       int64   `json:"query_id"`
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
}

// FTradeHistoryResponseData stores trade history data and legacy pagination values.
type FTradeHistoryResponseData struct {
	TotalPage   int64                `json:"total_page"`
	CurrentPage int64                `json:"current_page"`
	TotalSize   int64                `json:"total_size"`
	Trades      []FTradeHistoryEntry `json:"trades"`
}

// FTradeHistoryData stores trade history data for futures.
type FTradeHistoryData struct {
	Data      FTradeHistoryResponseData `json:"data"`
	Timestamp types.Time                `json:"ts"`
}

// UnmarshalJSON supports the documented v3 array response while preserving the
// legacy paged-object shape used by older responses.
func (f *FTradeHistoryData) UnmarshalJSON(data []byte) error {
	var response FTradeHistoryData
	if err := unmarshalV3FuturesResponse(data, &response.Timestamp, &response.Data.Trades, &response.Data); err != nil {
		return err
	}
	*f = response
	return nil
}

// unmarshalV3FuturesResponse preserves HTX's legacy paginated response shape
// while accepting the array and empty-data forms returned by current V3 APIs.
func unmarshalV3FuturesResponse[T, P any](data []byte, timestamp *types.Time, arrayResponse *[]T, legacyResponse *P) error {
	var raw struct {
		Data      json.RawMessage `json:"data"`
		Timestamp types.Time      `json:"ts"`
	}
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

// FTriggerOrderData stores trigger order data
type FTriggerOrderData struct {
	Data      FTriggerOrderDataData `json:"data"`
	Timestamp types.Time            `json:"ts"`
}

// FTriggerOpenOrders stores trigger open orders data
type FTriggerOpenOrders struct {
	Data      FTriggerOpenOrdersData `json:"data"`
	Timestamp types.Time             `json:"ts"`
}

// FTriggerOrderHistoryData stores trigger order history for futures
type FTriggerOrderHistoryData struct {
	Data      FTriggerOrderHistoryDataData `json:"data"`
	Timestamp types.Time                   `json:"ts"`
}

// FContractInfoDataData contains data fields from FContractInfoData.
type FContractInfoDataData struct {
	Symbol         string     `json:"symbol"`
	ContractCode   string     `json:"contract_code"`
	ContractType   string     `json:"contract_type"`
	ContractSize   float64    `json:"contract_size"`
	PriceTick      float64    `json:"price_tick"`
	DeliveryDate   string     `json:"delivery_date"`
	DeliveryTime   types.Time `json:"delivery_time"`
	CreateDate     types.Time `json:"create_date"`
	ContractStatus int64      `json:"contract_status"`
	SettlementTime types.Time `json:"settlement_time"`
}

// FContractIndexPriceInfoData contains data fields from FContractIndexPriceInfo.
type FContractIndexPriceInfoData struct {
	Symbol     string  `json:"symbol"`
	IndexPrice float64 `json:"index_price"`
}

// FEstimatedDeliveryPriceInfoData contains data fields from FEstimatedDeliveryPriceInfo.
type FEstimatedDeliveryPriceInfoData struct {
	DeliveryPrice float64 `json:"delivery_price"`
}

// FMarketDepthTick contains tick fields from FMarketDepth.
type FMarketDepthTick struct {
	MRID      int64        `json:"mrid"`
	ID        int64        `json:"id"`
	Bids      [][2]float64 `json:"bids"`
	Asks      [][2]float64 `json:"asks"`
	Timestamp types.Time   `json:"ts"`
	Version   int64        `json:"version"`
	Ch        string       `json:"ch"`
}

// FMarketOverviewDataTick contains tick fields from FMarketOverviewData.
type FMarketOverviewDataTick struct {
	Vol       types.Number `json:"vol"`
	Ask       []float64    `json:"ask"`
	Bid       []float64    `json:"bid"`
	Close     types.Number `json:"close"`
	Count     float64      `json:"count"`
	High      types.Number `json:"high"`
	ID        int64        `json:"id"`
	Low       types.Number `json:"low"`
	Open      types.Number `json:"open"`
	Timestamp types.Time   `json:"ts"`
	Amount    types.Number `json:"amount"`
}

// FLastTradeDataData contains data fields from FLastTradeData.
type FLastTradeDataData struct {
	Amount    types.Number `json:"amount"`
	Direction string       `json:"direction"`
	ID        int64        `json:"id"`
	Price     types.Number `json:"price"`
	Timestamp types.Time   `json:"ts"`
}

// FLastTradeDataTick contains tick fields from FLastTradeData.
type FLastTradeDataTick struct {
	Data      []FLastTradeDataData `json:"data"`
	ID        int64                `json:"id"`
	Timestamp types.Time           `json:"ts"`
}

// FBatchTradesForContractDataData contains data fields from FBatchTradesForContractData.
type FBatchTradesForContractDataData struct {
	ID        int64          `json:"id"`
	Timestamp types.Time     `json:"ts"`
	Data      []FuturesTrade `json:"data"`
}

// FTieredAdjustmentFactorInfoLadders contains ladders fields from FTieredAdjustmentFactorInfo.
type FTieredAdjustmentFactorInfoLadders struct {
	Ladder       int64   `json:"ladder"`
	MinSize      float64 `json:"min_size"`
	MaxSize      float64 `json:"max_size"`
	AdjustFactor float64 `json:"adjust_factor"`
}

// FTieredAdjustmentFactorInfoList contains list fields from FTieredAdjustmentFactorInfo.
type FTieredAdjustmentFactorInfoList struct {
	LeverageRate float64                              `json:"lever_rate"`
	Ladders      []FTieredAdjustmentFactorInfoLadders `json:"ladders"`
}

// FTieredAdjustmentFactorInfoData contains data fields from FTieredAdjustmentFactorInfo.
type FTieredAdjustmentFactorInfoData struct {
	Symbol string                            `json:"symbol"`
	List   []FTieredAdjustmentFactorInfoList `json:"list"`
}

// FOIDataTick contains tick fields from FOIData.
type FOIDataTick struct {
	Volume     types.Number `json:"volume"`
	AmountType int64        `json:"amount_type"`
	Timestamp  types.Time   `json:"ts"`
}

// FOIDataData contains data fields from FOIData.
type FOIDataData struct {
	Symbol       string        `json:"symbol"`
	ContractType string        `json:"contract_type"`
	Tick         []FOIDataTick `json:"tick"`
}

// FTopAccountsLongShortRatioList contains list fields from FTopAccountsLongShortRatio.
type FTopAccountsLongShortRatioList struct {
	BuyRatio    float64    `json:"buy_ratio"`
	SellRatio   float64    `json:"sell_ratio"`
	LockedRatio float64    `json:"locked_ratio"`
	Timestamp   types.Time `json:"ts"`
}

// FTopAccountsLongShortRatioData contains data fields from FTopAccountsLongShortRatio.
type FTopAccountsLongShortRatioData struct {
	List   []FTopAccountsLongShortRatioList `json:"list"`
	Symbol string                           `json:"symbol"`
}

// FTopPositionsLongShortRatioList contains list fields from FTopPositionsLongShortRatio.
type FTopPositionsLongShortRatioList struct {
	BuyRatio  float64    `json:"buy_ratio"`
	SellRatio float64    `json:"sell_ratio"`
	Timestamp types.Time `json:"timestamp"`
}

// FTopPositionsLongShortRatioData contains data fields from FTopPositionsLongShortRatio.
type FTopPositionsLongShortRatioData struct {
	Symbol string                            `json:"symbol"`
	List   []FTopPositionsLongShortRatioList `json:"list"`
}

// FIndexKlineDataData contains data fields from FIndexKlineData.
type FIndexKlineDataData struct {
	Vol    float64 `json:"vol"`
	Close  float64 `json:"close"`
	Count  float64 `json:"count"`
	High   float64 `json:"high"`
	ID     int64   `json:"id"`
	Low    float64 `json:"low"`
	Open   float64 `json:"open"`
	Amount float64 `json:"amount"`
}

// FBasisDataData contains data fields from FBasisData.
type FBasisDataData struct {
	Basis         types.Number `json:"basis"`
	BasisRate     types.Number `json:"basis_rate"`
	ContractPrice types.Number `json:"contract_price"`
	ID            int64        `json:"id"`
	IndexPrice    types.Number `json:"index_price"`
}

// FUserAccountDataAccData contains accdata fields from FUserAccountData.
type FUserAccountDataAccData struct {
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

// FUsersPositionsInfoPosInfo contains posinfo fields from FUsersPositionsInfo.
type FUsersPositionsInfoPosInfo struct {
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
}

// FSubAccountAssetsInfoList contains list fields from FSubAccountAssetsInfo.
type FSubAccountAssetsInfoList struct {
	Symbol           string  `json:"symbol"`
	MarginBalance    float64 `json:"margin_balance"`
	LiquidationPrice float64 `json:"liquidation_price"`
	RiskRate         float64 `json:"risk_rate"`
}

// FSubAccountAssetsInfoData contains data fields from FSubAccountAssetsInfo.
type FSubAccountAssetsInfoData struct {
	SubUID int64                       `json:"sub_uid"`
	List   []FSubAccountAssetsInfoList `json:"list"`
}

// FSingleSubAccountAssetsInfoAssetsData contains assetsdata fields from FSingleSubAccountAssetsInfo.
type FSingleSubAccountAssetsInfoAssetsData struct {
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

// FSingleSubAccountPositionsInfoPositionsData contains positionsdata fields from FSingleSubAccountPositionsInfo.
type FSingleSubAccountPositionsInfoPositionsData struct {
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
}

// FSettlementRecordsPositions contains positions fields from FSettlementRecords.
type FSettlementRecordsPositions struct {
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

// FSettlementRecordsSettlementRecords contains settlementrecords fields from FSettlementRecords.
type FSettlementRecordsSettlementRecords struct {
	Symbol               string                        `json:"symbol"`
	MarginBalanceInit    float64                       `json:"margin_balance_init"`
	MarginBalance        int64                         `json:"margin_balance"`
	SettlementProfitReal float64                       `json:"settlement_profit_real"`
	SettlementTime       types.Time                    `json:"settlement_time"`
	Clawback             float64                       `json:"clawback"`
	DeliveryFee          float64                       `json:"delivery_fee"`
	OffsetProfitLoss     float64                       `json:"offset_profitloss"`
	Fee                  float64                       `json:"fee"`
	FeeAsset             string                        `json:"fee_asset"`
	Positions            []FSettlementRecordsPositions `json:"positions"`
}

// FSettlementRecordsData contains data fields from FSettlementRecords.
type FSettlementRecordsData struct {
	SettlementRecords []FSettlementRecordsSettlementRecords `json:"settlement_records"`
	CurrentPage       int64                                 `json:"current_page"`
	TotalPage         int64                                 `json:"total_page"`
	TotalSize         int64                                 `json:"total_size"`
}

// FContractInfoOnOrderLimitContractTypes contains contracttypes fields from FContractInfoOnOrderLimit.
type FContractInfoOnOrderLimitContractTypes struct {
	ContractType string `json:"contract_type"`
	OpenLimit    int64  `json:"open_limit"`
	CloseLimit   int64  `json:"close_limit"`
}

// FContractInfoOnOrderLimitList contains list fields from FContractInfoOnOrderLimit.
type FContractInfoOnOrderLimitList struct {
	Symbol        string                                   `json:"symbol"`
	ContractTypes []FContractInfoOnOrderLimitContractTypes `json:"types"`
}

// FContractInfoOnOrderLimitContractData contains contractdata fields from FContractInfoOnOrderLimit.
type FContractInfoOnOrderLimitContractData struct {
	OrderPriceType string                          `json:"order_price_type"`
	List           []FContractInfoOnOrderLimitList `json:"list"`
}

// FContractTradingFeeDataContractTradingFeeData contains contracttradingfeedata fields from FContractTradingFeeData.
type FContractTradingFeeDataContractTradingFeeData struct {
	Symbol        string       `json:"symbol"`
	OpenMakerFee  types.Number `json:"open_maker_fee"`
	OpenTakerFee  types.Number `json:"open_taker_fee"`
	CloseMakerFee types.Number `json:"close_maker_fee"`
	CloseTakerFee types.Number `json:"close_taker_fee"`
	DeliveryFee   types.Number `json:"delivery_fee"`
	FeeAsset      string       `json:"fee_asset"`
}

// FTransferLimitDataData contains data fields from FTransferLimitData.
type FTransferLimitDataData struct {
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

// FPositionLimitDataList contains list fields from FPositionLimitData.
type FPositionLimitDataList struct {
	ContractType string  `json:"contract_type"`
	BuyLimit     float64 `json:"buy_limit"`
	SellLimit    float64 `json:"sell_limit"`
}

// FPositionLimitDataData contains data fields from FPositionLimitData.
type FPositionLimitDataData struct {
	Symbol string                   `json:"symbol"`
	List   []FPositionLimitDataList `json:"list"`
}

// FAssetsAndPositionsDataData contains data fields from FAssetsAndPositionsData.
type FAssetsAndPositionsDataData struct {
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

// FAccountTransferDataData contains data fields from FAccountTransferData.
type FAccountTransferDataData struct {
	OrderID string `json:"order_id"`
}

// FTransferRecordsTransferRecord contains transferrecord fields from FTransferRecords.
type FTransferRecordsTransferRecord struct {
	ID             int64      `json:"id"`
	Timestamp      types.Time `json:"ts"`
	Symbol         string     `json:"symbol"`
	SubUID         int64      `json:"sub_uid"`
	SubAccountName string     `json:"sub_account_name"`
	TransferType   int64      `json:"transfer_type"`
	Amount         float64    `json:"amount"`
}

// FTransferRecordsData contains data fields from FTransferRecords.
type FTransferRecordsData struct {
	TransferRecord []FTransferRecordsTransferRecord `json:"transfer_record"`
	TotalPage      int64                            `json:"total_page"`
	CurrentPage    int64                            `json:"current_page"`
	TotalSize      int64                            `json:"total_size"`
}

// FAvailableLeverageDataData contains data fields from FAvailableLeverageData.
type FAvailableLeverageDataData struct {
	Symbol                string `json:"symbol"`
	AvailableLeverageRate string `json:"available_level_rate"`
}

// FSwitchLeverageResponseData contains data fields from FSwitchLeverageResponse.
type FSwitchLeverageResponseData struct {
	Symbol       string `json:"symbol"`
	LeverageRate uint64 `json:"lever_rate"`
}

// FOrderDataData contains data fields from FOrderData.
type FOrderDataData struct {
	OrderID       int64  `json:"order_id"`
	OrderIDStr    string `json:"order_id_str"`
	ClientOrderID int64  `json:"client_order_id"`
}

// FCancelOrderDataErrors contains errors fields from FCancelOrderData.
type FCancelOrderDataErrors struct {
	OrderID int64  `json:"order_id,string"`
	ErrCode int64  `json:"err_code"`
	ErrMsg  string `json:"err_msg"`
}

// FCancelOrderDataData contains data fields from FCancelOrderData.
type FCancelOrderDataData struct {
	Errors    []FCancelOrderDataErrors `json:"errors"`
	Successes string                   `json:"successes"`
}

// FOrderInfoData contains data fields from FOrderInfo.
type FOrderInfoData struct {
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
	OrderIDString   string  `json:"order_id_str"`
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
}

// FOrderDetailsDataTrades contains trades fields from FOrderDetailsData.
type FOrderDetailsDataTrades struct {
	ID            string  `json:"id"`
	TradeID       int64   `json:"trade_id"`
	TradeVolume   float64 `json:"trade_volume"`
	TradePrice    float64 `json:"trade_price"`
	TradeFee      float64 `json:"trade_fee"`
	TradeTurnover float64 `json:"trade_turnover"`
	Role          string  `json:"role"`
	CreatedAt     int64   `json:"created_at"`
}

// FOrderDetailsDataData contains data fields from FOrderDetailsData.
type FOrderDetailsDataData struct {
	Symbol         string                    `json:"symbol"`
	ContractType   string                    `json:"contract_type"`
	ContractCode   string                    `json:"contract_code"`
	Volume         float64                   `json:"volume"`
	Price          float64                   `json:"price"`
	OrderPriceType string                    `json:"order_price_type"`
	Direction      string                    `json:"direction"`
	Offset         string                    `json:"offset"`
	LeverageRate   float64                   `json:"lever_rate"`
	MarginFrozen   float64                   `json:"margin_frozen"`
	Profit         float64                   `json:"profit"`
	OrderSource    string                    `json:"order_source"`
	OrderID        int64                     `json:"order_id"`
	OrderIDString  string                    `json:"order_id_str"`
	ClientOrderID  int64                     `json:"client_order_id"`
	OrderType      int64                     `json:"order_type"`
	Status         int64                     `json:"status"`
	TradeVolume    float64                   `json:"trade_volume"`
	TradeTurnover  int64                     `json:"trade_turnover"`
	TradeAvgPrice  float64                   `json:"trade_avg_price"`
	Fee            float64                   `json:"fee"`
	CreatedAt      int64                     `json:"created_at"`
	CanceledAt     int64                     `json:"canceled_at"`
	FinalInterest  float64                   `json:"final_interest"`
	AdjustValue    int64                     `json:"adjust_value"`
	FeeAsset       string                    `json:"fee_asset"`
	Trades         []FOrderDetailsDataTrades `json:"trades"`
	TotalPage      int64                     `json:"total_page"`
	TotalSize      int64                     `json:"total_size"`
	CurrentPage    int64                     `json:"current_page"`
}

// FOpenOrdersDataOrders contains orders fields from FOpenOrdersData.
type FOpenOrdersDataOrders struct {
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
}

// FOpenOrdersDataData contains data fields from FOpenOrdersData.
type FOpenOrdersDataData struct {
	Orders      []FOpenOrdersDataOrders `json:"orders"`
	TotalPage   int64                   `json:"total_page"`
	CurrentPage int64                   `json:"current_page"`
	TotalSize   int64                   `json:"total_size"`
}

// FTriggerOrderDataData contains data fields from FTriggerOrderData.
type FTriggerOrderDataData struct {
	OrderID    int64  `json:"order_id"`
	OrderIDStr string `json:"order_id_str"`
}

// FTriggerOpenOrdersOrders contains orders fields from FTriggerOpenOrders.
type FTriggerOpenOrdersOrders struct {
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
}

// FTriggerOpenOrdersData contains data fields from FTriggerOpenOrders.
type FTriggerOpenOrdersData struct {
	Orders      []FTriggerOpenOrdersOrders `json:"orders"`
	TotalPage   int64                      `json:"total_page"`
	CurrentPage int64                      `json:"current_page"`
	TotalSize   int64                      `json:"total_size"`
}

// FTriggerOrderHistoryDataOrders contains orders fields from FTriggerOrderHistoryData.
type FTriggerOrderHistoryDataOrders struct {
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
}

// FTriggerOrderHistoryDataData contains data fields from FTriggerOrderHistoryData.
type FTriggerOrderHistoryDataData struct {
	Orders      []FTriggerOrderHistoryDataOrders `json:"orders"`
	TotalPage   int64                            `json:"total_page"`
	CurrentPage int64                            `json:"current_page"`
	TotalSize   int64                            `json:"total_size"`
}
