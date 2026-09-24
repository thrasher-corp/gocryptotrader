package htx

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// SwapIndexPriceData gets price of a perpetual swap
type SwapIndexPriceData struct {
	Data []SwapIndexPrice `json:"data"`
}

// SwapIndexPrice contains the index price for a perpetual swap.
type SwapIndexPrice struct {
	ContractCode   string     `json:"contract_code"`
	IndexPrice     float64    `json:"index_price"`
	IndexTimestamp types.Time `json:"index_ts"`
}

// SwapPriceLimitsData gets price restrictions on perpetual swaps
type SwapPriceLimitsData struct {
	Data []SwapPriceLimit `json:"data"`
}

// SwapPriceLimit contains the price restrictions for a perpetual swap.
type SwapPriceLimit struct {
	Symbol       string  `json:"symbol"`
	HighLimit    float64 `json:"high_limit"`
	LowLimit     float64 `json:"low_limit"`
	ContractCode string  `json:"contract_code"`
}

// SwapOpenInterestData stores open interest data for swaps
type SwapOpenInterestData struct {
	Data []SwapOpenInterest `json:"data"`
}

// SwapOpenInterest contains open interest for a perpetual swap.
type SwapOpenInterest struct {
	Symbol       string  `json:"symbol"`
	Volume       float64 `json:"volume"`
	Amount       float64 `json:"amount"`
	ContractCode string  `json:"contract_code"`
}

// SwapMarketDepthData stores market depth data
type SwapMarketDepthData struct {
	Tick SwapMarketDepth `json:"tick"`
}

// SwapMarketDepth contains a perpetual swap orderbook snapshot.
type SwapMarketDepth struct {
	Asks      [][]float64 `json:"asks"`
	Bids      [][]float64 `json:"bids"`
	Channel   string      `json:"ch"`
	ID        int64       `json:"id"`
	MRID      int64       `json:"mrid"`
	Timestamp types.Time  `json:"ts"`
	Version   int64       `json:"version"`
}

// SwapKlineData stores kline data for perpetual swaps
type SwapKlineData struct {
	Data []FuturesKline `json:"data"`
}

// MarketOverviewData stores market overview data
type MarketOverviewData struct {
	Channel string         `json:"ch"`
	Tick    MarketOverview `json:"tick"`
}

// MarketOverview contains the latest aggregate market values for a perpetual swap.
type MarketOverview struct {
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

// LastTradeData stores last trade's data of a contract
type LastTradeData struct {
	Ch   string        `json:"ch"`
	Tick LastTradeTick `json:"tick"`
}

// LastTradeTick contains the latest trades for a perpetual swap.
type LastTradeTick struct {
	Data []LastTrade `json:"data"`
}

// LastTrade contains a single latest trade for a perpetual swap.
type LastTrade struct {
	Amount    types.Number `json:"amount"`
	Direction string       `json:"direction"`
	ID        int64        `json:"id"`
	Price     types.Number `json:"price"`
	Timestamp types.Time   `json:"ts"`
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

// TieredAdjustmentFactorData stores tiered adjustment factor data
type TieredAdjustmentFactorData struct {
	Data []TieredAdjustmentFactor `json:"data"`
}

// TieredAdjustmentFactor contains leverage tiers for a perpetual swap.
type TieredAdjustmentFactor struct {
	Symbol       string                           `json:"symbol"`
	ContractCode string                           `json:"contract_code"`
	List         []TieredAdjustmentFactorLeverage `json:"list"`
}

// TieredAdjustmentFactorLeverage contains adjustment ladders for a leverage rate.
type TieredAdjustmentFactorLeverage struct {
	LeverRate float64                        `json:"lever_rate"`
	Ladders   []TieredAdjustmentFactorLadder `json:"ladders"`
}

// TieredAdjustmentFactorLadder contains the adjustment factor for a size range.
type TieredAdjustmentFactorLadder struct {
	Ladder       float64 `json:"ladder"`
	MinSize      float64 `json:"min_size"`
	MaxSize      float64 `json:"max_size"`
	AdjustFactor float64 `json:"adjust_factor"`
}

// OpenInterestData stores open interest data
type OpenInterestData struct {
	Data OpenInterest `json:"data"`
}

// OpenInterest contains historical open interest for a perpetual swap.
type OpenInterest struct {
	Symbol       string             `json:"symbol"`
	ContractCode string             `json:"contract_code"`
	Tick         []OpenInterestTick `json:"tick"`
}

// OpenInterestTick contains open interest at a point in time.
type OpenInterestTick struct {
	Volume     float64    `json:"volume"`
	AmountType float64    `json:"amountType"`
	Timestamp  types.Time `json:"ts"`
}

// SystemStatusData stores information on system status
type SystemStatusData struct {
	Data []SystemStatus `json:"data"`
}

// SystemStatus contains the enabled operations for a perpetual swap.
type SystemStatus struct {
	Symbol            string  `json:"symbol"`
	ContractCode      string  `json:"contract_code"`
	Open              float64 `json:"open"`
	Close             float64 `json:"close"`
	Cancel            float64 `json:"cancel"`
	TransferIn        float64 `json:"transfer_in"`
	TransferOut       float64 `json:"transfer_out"`
	MasterTransferSub float64 `json:"master_transfer_sub"`
	SubTransferMaster float64 `json:"sub_transfer_master"`
}

// TraderSentimentIndexAccountData stores trader sentiment index data
type TraderSentimentIndexAccountData struct {
	Data TraderSentimentIndexAccount `json:"data"`
}

// TraderSentimentIndexAccount contains account sentiment for a perpetual swap.
type TraderSentimentIndexAccount struct {
	Symbol       string                                `json:"symbol"`
	ContractCode string                                `json:"contract_code"`
	List         []TraderSentimentIndexAccountSnapshot `json:"list"`
}

// TraderSentimentIndexAccountSnapshot contains account sentiment at a point in time.
type TraderSentimentIndexAccountSnapshot struct {
	BuyRatio    float64    `json:"buy_ratio"`
	SellRatio   float64    `json:"sell_ratio"`
	LockedRatio float64    `json:"locked_ratio"`
	Timestamp   types.Time `json:"ts"`
}

// TraderSentimentIndexPositionData stores trader sentiment index data
type TraderSentimentIndexPositionData struct {
	Data TraderSentimentIndexPosition `json:"data"`
}

// TraderSentimentIndexPosition contains position sentiment for a perpetual swap.
type TraderSentimentIndexPosition struct {
	Symbol       string                                 `json:"symbol"`
	ContractCode string                                 `json:"contract_code"`
	List         []TraderSentimentIndexPositionSnapshot `json:"list"`
}

// TraderSentimentIndexPositionSnapshot contains position sentiment at a point in time.
type TraderSentimentIndexPositionSnapshot struct {
	BuyRatio  float64    `json:"buy_ratio"`
	SellRatio float64    `json:"sell_ratio"`
	Timestamp types.Time `json:"ts"`
}

// LiquidationOrdersData stores data of liquidation orders
type LiquidationOrdersData struct {
	Data []LiquidationOrder `json:"data"`
}

// LiquidationOrder contains a perpetual swap liquidation order.
type LiquidationOrder struct {
	QueryID      int64      `json:"query_id"`
	ContractCode string     `json:"contract_code"`
	Symbol       string     `json:"symbol"`
	Direction    string     `json:"direction"`
	Offset       string     `json:"offset"`
	Volume       float64    `json:"volume"`
	Price        float64    `json:"price"`
	CreatedAt    types.Time `json:"created_at"`
	Amount       float64    `json:"amount"`
}

// SwapFundingRatesResponse holds funding rates and data response
type SwapFundingRatesResponse struct {
	Response
	Data []FundingRatesData `json:"data"`
}

// FundingRatesData stores funding rates data
type FundingRatesData struct {
	EstimatedRate   types.Number `json:"estimated_rate"`
	FundingRate     types.Number `json:"funding_rate"`
	ContractCode    string       `json:"contract_code"`
	Symbol          string       `json:"symbol"`
	FeeAsset        string       `json:"fee_asset"`
	FundingTime     types.Time   `json:"funding_time"`
	NextFundingTime types.Time   `json:"next_funding_time"`
}

// HistoricalFundingRateData stores historical funding rates for perpetuals
type HistoricalFundingRateData struct {
	Data HistoricalFundingRatePage `json:"data"`
}

// HistoricalFundingRatePage contains paginated perpetual swap funding rates.
type HistoricalFundingRatePage struct {
	TotalPage   int64                `json:"total_page"`
	CurrentPage int64                `json:"current_page"`
	TotalSize   int64                `json:"total_size"`
	Data        []HistoricalRateData `json:"data"`
}

// HistoricalRateData stores historical rates data
type HistoricalRateData struct {
	FundingRate     types.Number `json:"funding_rate"`
	RealizedRate    types.Number `json:"realized_rate"`
	FundingTime     types.Time   `json:"funding_time"`
	ContractCode    string       `json:"contract_code"`
	Symbol          string       `json:"symbol"`
	FeeAsset        string       `json:"fee_asset"`
	AvgPremiumIndex types.Number `json:"avg_premium_index"`
}

// PremiumIndexKlineData stores kline data for premium
type PremiumIndexKlineData struct {
	Channel   string           `json:"ch"`
	Data      []IndexKlineData `json:"data"`
	Timestamp types.Time       `json:"ts"`
}

// EstimatedFundingRateData stores estimated funding rate data
type EstimatedFundingRateData struct {
	Channel   string           `json:"ch"`
	Data      []IndexKlineData `json:"data"`
	Timestamp types.Time       `json:"ts"`
}

// IndexKlineData contains a premium index or estimated funding rate kline.
type IndexKlineData struct {
	Volume types.Number `json:"vol"`
	Close  types.Number `json:"close"`
	Count  types.Number `json:"count"`
	High   types.Number `json:"high"`
	ID     int64        `json:"id"`
	Low    types.Number `json:"low"`
	Open   types.Number `json:"open"`
	Amount types.Number `json:"amount"`
}

// BasisData stores basis data for swaps
type BasisData struct {
	Channel   string      `json:"ch"`
	Data      []BasisTick `json:"data"`
	Timestamp types.Time  `json:"ts"`
}

// BasisTick contains a perpetual swap basis value.
type BasisTick struct {
	Basis         string `json:"basis"`
	BasisRate     string `json:"basis_rate"`
	ContractPrice string `json:"contract_price"`
	ID            int64  `json:"id"`
	IndexPrice    string `json:"index_price"`
}

// SwapAccountInformation stores swap account information
type SwapAccountInformation struct {
	Data []SwapAccount `json:"data"`
}

// SwapAccount contains account balances and risk values for a perpetual swap.
type SwapAccount struct {
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
}

// SwapPositionInfo stores user's swap positions' info
type SwapPositionInfo struct {
	Data []SwapPosition `json:"data"`
}

// SwapPosition contains a perpetual swap position.
type SwapPosition struct {
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
}

// SwapAssetsAndPositionsData stores positions and assets data for swaps
type SwapAssetsAndPositionsData struct {
	Timestamp types.Time                 `json:"ts"`
	Data      []SwapAssetAndPositionData `json:"data"`
}

// SwapAssetAndPositionData contains account values and positions for a perpetual swap.
type SwapAssetAndPositionData struct {
	Symbol            string         `json:"symbol"`
	ContractCode      string         `json:"contract_code"`
	MarginBalance     float64        `json:"margin_balance"`
	MarginPosition    float64        `json:"margin_position"`
	MarginFrozen      float64        `json:"margin_frozen"`
	MarginAvailable   float64        `json:"margin_available"`
	ProfitReal        float64        `json:"profit_real"`
	ProfitUnreal      float64        `json:"profit_unreal"`
	WithdrawAvailable float64        `json:"withdraw_available"`
	RiskRate          float64        `json:"risk_rate"`
	LiquidationPrice  float64        `json:"liquidation_price"`
	AdjustFactor      float64        `json:"adjust_factor"`
	LeverageRate      float64        `json:"lever_rate"`
	MarginStatic      float64        `json:"margin_static"`
	Positions         []SwapPosition `json:"positions"`
}

// SubAccountsAssetData stores asset data for all subaccounts
type SubAccountsAssetData struct {
	Timestamp types.Time        `json:"ts"`
	Data      []SubAccountAsset `json:"data"`
}

// SubAccountAsset contains perpetual swap assets for a subaccount.
type SubAccountAsset struct {
	SubUID int64                    `json:"sub_uid"`
	List   []SubAccountAssetBalance `json:"list"`
}

// SubAccountAssetBalance contains a subaccount balance and risk values.
type SubAccountAssetBalance struct {
	Symbol           string  `json:"symbol"`
	ContractCode     string  `json:"contract_code"`
	MarginBalance    int64   `json:"margin_balance"`
	LiquidationPrice float64 `json:"liquidation_price"`
	RiskRate         float64 `json:"risk_rate"`
}

// SingleSubAccountAssetsInfo stores asset info for a single subaccount
type SingleSubAccountAssetsInfo struct {
	Timestamp types.Time    `json:"ts"`
	Data      []SwapAccount `json:"data"`
}

// SingleSubAccountPositionsInfo stores single subaccount's positions data
type SingleSubAccountPositionsInfo struct {
	Timestamp types.Time     `json:"ts"`
	Data      []SwapPosition `json:"data"`
}

// AvailableLeverageData stores data of available leverage for account
type AvailableLeverageData struct {
	Data      []AvailableLeverage `json:"data"`
	Timestamp types.Time          `json:"timestamp"`
}

// AvailableLeverage contains the leverage rates available for a contract.
type AvailableLeverage struct {
	ContractCode      string `json:"contract_code"`
	AvailableLeverage string `json:"available_level_rate"`
}

// SwitchCoinMarginedLeverageRequest defines a coin-margined perpetual leverage change.
type SwitchCoinMarginedLeverageRequest struct {
	ContractCode string `json:"contract_code"`
	LeverageRate uint64 `json:"lever_rate"`
}

// SwitchCoinMarginedLeverageResponse contains the leverage accepted by HTX.
type SwitchCoinMarginedLeverageResponse struct {
	Response
	Data SwitchCoinMarginedLeverageData `json:"data"`
}

// SwitchCoinMarginedLeverageData contains the accepted contract leverage.
type SwitchCoinMarginedLeverageData struct {
	ContractCode string `json:"contract_code"`
	LeverageRate uint64 `json:"lever_rate"`
}

// FinancialRecord stores a coin-margined financial record entry.
type FinancialRecord struct {
	QueryID      int64      `json:"query_id"`
	ID           int64      `json:"id"`
	Timestamp    types.Time `json:"ts"`
	Symbol       string     `json:"symbol"`
	ContractCode string     `json:"contract_code"`
	OrderType    int64      `json:"type"`
	Amount       float64    `json:"amount"`
}

// FinancialRecordResponseData stores financial record data and legacy pagination values.
type FinancialRecordResponseData struct {
	FinancialRecord []FinancialRecord `json:"financial_record"`
	TotalPage       int64             `json:"total_page"`
	CurrentPage     int64             `json:"current_page"`
	TotalSize       int64             `json:"total_size"`
}

// FinancialRecordData stores an accounts financial records.
type FinancialRecordData struct {
	Data      FinancialRecordResponseData `json:"data"`
	Timestamp types.Time                  `json:"ts"`
}

// UnmarshalJSON supports the documented v3 array response while preserving the
// legacy paged-object shape used by older responses.
func (f *FinancialRecordData) UnmarshalJSON(data []byte) error {
	var response FinancialRecordData
	if err := unmarshalV3FuturesResponse(data, &response.Timestamp, &response.Data.FinancialRecord, &response.Data); err != nil {
		return err
	}
	*f = response
	return nil
}

// SwapOrderLimitInfo stores information about order limits on a perpetual swap
type SwapOrderLimitInfo struct {
	Data      SwapOrderLimitData `json:"data"`
	Timestamp types.Time         `json:"ts"`
}

// SwapOrderLimitData contains order limits grouped by order price type.
type SwapOrderLimitData struct {
	OrderPriceType string           `json:"order_price_type"`
	List           []SwapOrderLimit `json:"list"`
}

// SwapOrderLimit contains open and close limits for a perpetual swap.
type SwapOrderLimit struct {
	Symbol       string  `json:"symbol"`
	ContractCode string  `json:"contract_code"`
	OpenLimit    float64 `json:"open_limit"`
	CloseLimit   float64 `json:"close_limit"`
}

// SwapTradingFeeData stores trading fee data for swaps
type SwapTradingFeeData struct {
	Data      []SwapTradingFee `json:"data"`
	Timestamp types.Time       `json:"ts"`
}

// SwapTradingFee contains maker and taker fees for a perpetual swap.
type SwapTradingFee struct {
	Symbol        string       `json:"symbol"`
	ContractCode  string       `json:"contract_code"`
	FeeAsset      string       `json:"fee_asset"`
	OpenMakerFee  types.Number `json:"open_maker_fee"`
	OpenTakerFee  types.Number `json:"open_taker_fee"`
	CloseMakerFee types.Number `json:"close_maker_fee"`
	CloseTakerFee types.Number `json:"close_taker_fee"`
}

// TransferLimitData stores transfer limits
type TransferLimitData struct {
	Data      []TransferLimit `json:"data"`
	Timestamp types.Time      `json:"timestamp"`
}

// TransferLimit contains transfer restrictions for a perpetual swap.
type TransferLimit struct {
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
}

// PositionLimitData stores position limit data
type PositionLimitData struct {
	Data      []PositionLimit `json:"data"`
	Timestamp types.Time      `json:"ts"`
}

// PositionLimit contains buy and sell position limits for a perpetual swap.
type PositionLimit struct {
	Symbol       string  `json:"symbol"`
	ContractCode string  `json:"contract_code"`
	BuyLimit     float64 `json:"buy_limit"`
	SellLimit    float64 `json:"sell_limit"`
}

// InternalAccountTransferData stores transfer data between subaccounts and main account
type InternalAccountTransferData struct {
	TS   int64                       `json:"ts"`
	Data InternalAccountTransferInfo `json:"data"`
}

// InternalAccountTransferInfo contains an internal transfer identifier.
type InternalAccountTransferInfo struct {
	OrderID string `json:"order_id"`
}

// SwapOrderData stores swap order data
type SwapOrderData struct {
	Data      SwapOrderResponseData `json:"data"`
	Timestamp types.Time            `json:"ts"`
}

// SwapOrderResponseData contains identifiers returned for a perpetual swap order.
type SwapOrderResponseData struct {
	OrderID       int64  `json:"order_id"`
	OrderIDString string `json:"order_id_str"`
	ClientOrderID int64  `json:"client_order_id"`
}

// BatchOrderData stores data for batch orders
type BatchOrderData struct {
	Data      BatchOrderResponseData `json:"data"`
	Timestamp types.Time             `json:"ts"`
}

// BatchOrderResponseData contains successful and failed batch order results.
type BatchOrderResponseData struct {
	Errors  []BatchOrderError   `json:"errors"`
	Success []BatchOrderSuccess `json:"success"`
}

// BatchOrderError contains a failed batch order result.
type BatchOrderError struct {
	ErrCode int64  `json:"err_code"`
	ErrMsg  string `json:"err_msg"`
	Index   int64  `json:"index"`
}

// BatchOrderSuccess contains a successful batch order result.
type BatchOrderSuccess struct {
	Index         int64  `json:"index"`
	OrderID       int64  `json:"order_id"`
	OrderIDString string `json:"order_id_str"`
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
	Data      CancelOrdersResponseData `json:"data"`
	Timestamp types.Time               `json:"ts"`
}

// CancelOrdersResponseData contains successful and failed order cancellations.
type CancelOrdersResponseData struct {
	Errors    []CancelOrderError `json:"errors"`
	Successes string             `json:"successes"`
}

// CancelOrderError contains a failed order cancellation.
type CancelOrderError struct {
	OrderID string `json:"order_id"`
	ErrCode int64  `json:"err_code"`
	ErrMsg  string `json:"err_msg"`
}

// LightningCloseOrderData stores order data from a lightning close order
type LightningCloseOrderData struct {
	Data      SwapOrderResponseData `json:"data"`
	Timestamp types.Time            `json:"ts"`
}

// SwapOrderInfo stores info for swap orders
type SwapOrderInfo struct {
	Data      []SwapOrder `json:"data"`
	Timestamp types.Time  `json:"ts"`
}

// SwapOrder contains perpetual swap order information.
type SwapOrder struct {
	Symbol          string  `json:"symbol"`
	ContractCode    string  `json:"contract_code"`
	Volume          float64 `json:"volume"`
	Price           float64 `json:"price"`
	OrderPriceType  string  `json:"order_price_type"`
	Direction       string  `json:"direction"`
	Offset          string  `json:"offset"`
	LeverRate       int64   `json:"lever_rate"`
	OrderID         int64   `json:"order_id"`
	OrderIDString   string  `json:"order_id_str"`
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
	FeeAsset        string  `json:"fee_asset"`
	LiquidationType int64   `json:"liquidation_type"`
}

// SwapOpenOrdersData stores open orders data for swaps
type SwapOpenOrdersData struct {
	Data      SwapOpenOrdersResponseData `json:"data"`
	Timestamp types.Time                 `json:"ts"`
}

// SwapOpenOrdersResponseData contains open orders and pagination values.
type SwapOpenOrdersResponseData struct {
	Orders      []SwapOpenOrder `json:"orders"`
	TotalPage   int64           `json:"total_page"`
	CurrentPage int64           `json:"current_page"`
	TotalSize   int64           `json:"total_size"`
}

// SwapOpenOrder contains an open perpetual swap order.
type SwapOpenOrder struct {
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
}

// SwapOrderHistoryEntry stores an order history entry for coin-margined swaps.
type SwapOrderHistoryEntry struct {
	QueryID           int64   `json:"query_id"`
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
	UpdateTime        int64   `json:"update_time"`
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
}

// SwapOrderHistoryResponseData stores order history data and legacy pagination values.
type SwapOrderHistoryResponseData struct {
	Orders      []SwapOrderHistoryEntry `json:"orders"`
	TotalPage   int64                   `json:"total_page"`
	CurrentPage int64                   `json:"current_page"`
	TotalSize   int64                   `json:"total_size"`
}

// SwapOrderHistory gets order history for swaps.
type SwapOrderHistory struct {
	Data      SwapOrderHistoryResponseData `json:"data"`
	Timestamp types.Time                   `json:"ts"`
}

// UnmarshalJSON supports the documented v3 array response while preserving the
// legacy paged-object shape used by older responses.
func (s *SwapOrderHistory) UnmarshalJSON(data []byte) error {
	var response SwapOrderHistory
	if err := unmarshalV3FuturesResponse(data, &response.Timestamp, &response.Data.Orders, &response.Data); err != nil {
		return err
	}
	*s = response
	return nil
}

// AccountTradeHistoryEntry stores a trade history entry for coin-margined swaps.
type AccountTradeHistoryEntry struct {
	QueryID          int64   `json:"query_id"`
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
}

// AccountTradeHistoryResponseData stores trade history data and legacy pagination values.
type AccountTradeHistoryResponseData struct {
	CurrentPage int64                      `json:"current_page"`
	TotalPage   int64                      `json:"total_page"`
	TotalSize   int64                      `json:"total_size"`
	Trades      []AccountTradeHistoryEntry `json:"trades"`
}

// AccountTradeHistoryData stores account trade history for swaps.
type AccountTradeHistoryData struct {
	Data      AccountTradeHistoryResponseData `json:"data"`
	Timestamp types.Time                      `json:"ts"`
}

// UnmarshalJSON supports the documented v3 array response while preserving the
// legacy paged-object shape used by older responses.
func (a *AccountTradeHistoryData) UnmarshalJSON(data []byte) error {
	var response AccountTradeHistoryData
	if err := unmarshalV3FuturesResponse(data, &response.Timestamp, &response.Data.Trades, &response.Data); err != nil {
		return err
	}
	*a = response
	return nil
}

// CancelTriggerOrdersData stores trigger order cancel data
type CancelTriggerOrdersData struct {
	Data      CancelTriggerOrdersResponseData `json:"data"`
	Timestamp types.Time                      `json:"ts"`
}

// CancelTriggerOrdersResponseData contains successful and failed trigger order cancellations.
type CancelTriggerOrdersResponseData struct {
	Errors    []CancelTriggerOrderError `json:"errors"`
	Successes string                    `json:"successes"`
}

// CancelTriggerOrderError contains a failed trigger order cancellation.
type CancelTriggerOrderError struct {
	OrderID int64  `json:"order_id"`
	ErrCode int64  `json:"err_code"`
	ErrMsg  string `json:"err_msg"`
}

// TriggerOrderHistory stores trigger order history data for swaps
type TriggerOrderHistory struct {
	Data      TriggerOrderHistoryResponseData `json:"data"`
	Timestamp types.Time                      `json:"ts"`
}

// TriggerOrderHistoryResponseData contains trigger orders and pagination values.
type TriggerOrderHistoryResponseData struct {
	Orders      []TriggerOrder `json:"orders"`
	TotalPage   int64          `json:"total_page"`
	CurrentPage int64          `json:"current_page"`
	TotalSize   int64          `json:"total_size"`
}

// TriggerOrder contains a historical perpetual swap trigger order.
type TriggerOrder struct {
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
}
