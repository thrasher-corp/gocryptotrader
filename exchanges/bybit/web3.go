package bybit

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// Web3 (formerly Alpha) endpoint validation errors
var (
	errTradeTypeRequired          = errors.New("trade type required")
	errTokenCodeRequired          = errors.New("token code required")
	errQuoteDataRequired          = errors.New("quote data required")
	errCorrectingCodeRequired     = errors.New("correcting code required")
	errGasRequired                = errors.New("gas from the quote required")
	errChainCodeRequired          = errors.New("chain code required")
	errTokenAddressRequired       = errors.New("token address required")
	errPageLimitRequired          = errors.New("page limit required")
	errPageIndexRequired          = errors.New("page index required")
	errPoolAddressRequired        = errors.New("pool address required")
	errLPPositionIDRequired       = errors.New("LP position ID required")
	errDercRatioInvalid           = errors.New("redeem ratio must be greater than zero and at most 1")
	errRangeAndPriceBoundsBothSet = errors.New("range and price bounds are mutually exclusive")
	errOutcomeTokenIDRequired     = errors.New("outcome token ID required")
	errEventIDRequired            = errors.New("event ID required")
	errPriceIntervalRequired      = errors.New("price interval required")
	errEventTypeRequired          = errors.New("event type required")
)

// GetWeb3TradeQuote returns a quote for an on-chain purchase or redeem trade, which Execute Purchase and Execute Redeem
// require (Get Trade Quote)
func (e *Exchange) GetWeb3TradeQuote(ctx context.Context, r *Web3TradeQuoteRequest) (*Web3TradeQuoteResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.TradeType == 0 {
		return nil, errTradeTypeRequired
	}
	if r.FromTokenCode == "" {
		return nil, fmt.Errorf("%w: from token", errTokenCodeRequired)
	}
	if r.FromTokenAmount <= 0 {
		return nil, fmt.Errorf("%w: from token amount", order.ErrAmountIsInvalid)
	}
	if r.ToTokenCode == "" {
		return nil, fmt.Errorf("%w: to token", errTokenCodeRequired)
	}
	body := make(map[string]any)
	body["tradeType"] = r.TradeType
	body["fromTokenCode"] = r.FromTokenCode
	body["fromTokenAmount"] = strconv.FormatFloat(r.FromTokenAmount, 'f', -1, 64)
	body["toTokenCode"] = r.ToTokenCode
	if r.QuoteMode != 0 {
		body["quoteMode"] = r.QuoteMode
	}
	var resp *Web3TradeQuoteResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/alpha/trade/quote", nil, body, &resp, defaultEPL)
}

// ExecuteWeb3Purchase buys an on-chain token with a payment token at a quote from GetWeb3TradeQuote. The order is only
// acknowledged; GetWeb3OrderList reports its outcome (Execute Purchase)
func (e *Exchange) ExecuteWeb3Purchase(ctx context.Context, r *Web3TradeRequest) (*Web3OrderResponse, error) {
	body, err := web3TradeBody(r)
	if err != nil {
		return nil, err
	}
	var resp *Web3OrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/alpha/trade/purchase", nil, body, &resp, defaultEPL)
}

// ExecuteWeb3Redeem sells an on-chain token for a payment token at a quote from GetWeb3TradeQuote. The order is only
// acknowledged; GetWeb3OrderList reports its outcome (Execute Redeem)
func (e *Exchange) ExecuteWeb3Redeem(ctx context.Context, r *Web3TradeRequest) (*Web3OrderResponse, error) {
	body, err := web3TradeBody(r)
	if err != nil {
		return nil, err
	}
	var resp *Web3OrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/alpha/trade/redeem", nil, body, &resp, defaultEPL)
}

// web3TradeBody builds the body Execute Purchase and Execute Redeem share. Slippage and QuoteMode are always sent: the
// API requires them and zero is a valid value for each. Gas is sent as the quote's own string, since Bybit requires it
// unchanged
func web3TradeBody(r *Web3TradeRequest) (map[string]any, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.FromTokenCode == "" {
		return nil, fmt.Errorf("%w: from token", errTokenCodeRequired)
	}
	if r.FromTokenAmount <= 0 {
		return nil, fmt.Errorf("%w: from token amount", order.ErrAmountIsInvalid)
	}
	if r.ToTokenCode == "" {
		return nil, fmt.Errorf("%w: to token", errTokenCodeRequired)
	}
	if r.QuoteData == "" {
		return nil, errQuoteDataRequired
	}
	if r.CorrectingCode == "" {
		return nil, errCorrectingCodeRequired
	}
	if r.Gas == "" {
		return nil, errGasRequired
	}
	body := make(map[string]any)
	body["fromTokenCode"] = r.FromTokenCode
	body["fromTokenAmount"] = strconv.FormatFloat(r.FromTokenAmount, 'f', -1, 64)
	body["toTokenCode"] = r.ToTokenCode
	body["slippage"] = strconv.FormatFloat(r.Slippage, 'f', -1, 64)
	body["quoteData"] = r.QuoteData
	body["gas"] = r.Gas
	body["quoteMode"] = r.QuoteMode
	body["correctingCode"] = r.CorrectingCode
	if r.Tenant != "" {
		body["tenant"] = r.Tenant
	}
	return body, nil
}

// GetWeb3PaymentTokenList returns the payment tokens, with their CEX_ token codes, that can buy the on-chain token at
// tokenAddress on chainCode (Get Payment Token List)
func (e *Exchange) GetWeb3PaymentTokenList(ctx context.Context, chainCode, tokenAddress string) ([]Web3PaymentToken, error) {
	if chainCode == "" {
		return nil, errChainCodeRequired
	}
	if tokenAddress == "" {
		return nil, errTokenAddressRequired
	}
	body := make(map[string]any)
	body["chainCode"] = chainCode
	body["tokenAddress"] = tokenAddress
	var resp []Web3PaymentToken
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/alpha/trade/pay-token-list", nil, body, &resp, defaultEPL)
}

// GetWeb3OrderList returns on-chain purchase and redeem orders from the last 90 days at most, newest first (Get Order
// List)
func (e *Exchange) GetWeb3OrderList(ctx context.Context, r *Web3OrderListRequest) (*Web3OrderListResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.Limit == 0 {
		return nil, errPageLimitRequired
	}
	if r.PageIndex == 0 {
		return nil, errPageIndexRequired
	}
	body := make(map[string]any)
	body["limit"] = r.Limit
	body["pageIndex"] = r.PageIndex
	if r.TradeType != 0 {
		body["tradeType"] = r.TradeType
	}
	if r.TokenCode != "" {
		body["tokenCode"] = r.TokenCode
	}
	if len(r.OrderStatus) != 0 {
		body["orderStatus"] = r.OrderStatus
	}
	if r.Days != 0 {
		body["days"] = r.Days
	}
	if r.Direction != "" {
		body["direction"] = r.Direction
	}
	var resp *Web3OrderListResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/alpha/trade/order-list", nil, body, &resp, defaultEPL)
}

// GetWeb3BizTokenList returns the on-chain tokens available for trading, with their DEX_ token codes, risk flags and
// order limits. tokenTag 1 returns new token sniping, 2 on-chain hot tokens, and 0 all tokens (Get Biz Token List)
func (e *Exchange) GetWeb3BizTokenList(ctx context.Context, tokenTag uint64) ([]Web3BizToken, error) {
	body := make(map[string]any)
	if tokenTag != 0 {
		body["tokenTag"] = tokenTag
	}
	var resp []Web3BizToken
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/alpha/trade/biz-token-list", nil, body, &resp, defaultEPL)
}

// GetWeb3TokenPriceList returns market data for up to 20 on-chain tokens (Get Token Price List)
func (e *Exchange) GetWeb3TokenPriceList(ctx context.Context, tokens []Web3TokenAddress) (*Web3TokenPriceListResponse, error) {
	if len(tokens) == 0 {
		return nil, fmt.Errorf("%w: no tokens given", errTokenAddressRequired)
	}
	tokenAddressInfo := make([]map[string]string, len(tokens))
	for i := range tokens {
		if tokens[i].ChainCode == "" {
			return nil, errChainCodeRequired
		}
		if tokens[i].TokenAddress == "" {
			return nil, errTokenAddressRequired
		}
		tokenAddressInfo[i] = make(map[string]string)
		tokenAddressInfo[i]["chainCode"] = tokens[i].ChainCode
		tokenAddressInfo[i]["tokenAddress"] = tokens[i].TokenAddress
	}
	body := make(map[string]any)
	body["tokenAddressInfo"] = tokenAddressInfo
	var resp *Web3TokenPriceListResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/alpha/trade/biz-token-price-list", nil, body, &resp, defaultEPL)
}

// GetWeb3TokenDetails returns the project details of the listed or delisting on-chain token at tokenAddress on
// chainCode (Get Token Details)
func (e *Exchange) GetWeb3TokenDetails(ctx context.Context, chainCode, tokenAddress string) (*Web3TokenDetailsResponse, error) {
	if chainCode == "" {
		return nil, errChainCodeRequired
	}
	if tokenAddress == "" {
		return nil, errTokenAddressRequired
	}
	body := make(map[string]any)
	body["chainCode"] = chainCode
	body["tokenAddress"] = tokenAddress
	var resp *Web3TokenDetailsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/alpha/trade/biz-token-details", nil, body, &resp, defaultEPL)
}

// GetWeb3AssetList returns the on-chain tokens held, largest USD value first (Get Asset List)
func (e *Exchange) GetWeb3AssetList(ctx context.Context) (*Web3AssetListResponse, error) {
	var resp *Web3AssetListResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/alpha/trade/asset-list", nil, nil, &resp, defaultEPL)
}

// GetWeb3AssetDetail returns the holding of the on-chain token at tokenAddress on chainCode (Get Asset Detail)
func (e *Exchange) GetWeb3AssetDetail(ctx context.Context, chainCode, tokenAddress string) (*Web3AssetDetailResponse, error) {
	if chainCode == "" {
		return nil, errChainCodeRequired
	}
	if tokenAddress == "" {
		return nil, errTokenAddressRequired
	}
	body := make(map[string]any)
	body["chainCode"] = chainCode
	body["tokenAddress"] = tokenAddress
	var resp *Web3AssetDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/alpha/trade/asset-detail", nil, body, &resp, defaultEPL)
}

// ExecuteWeb3LPStake stakes a payment token into a liquidity pool, opening a position or adding to one. The order is
// only acknowledged; GetWeb3LPOrderList reports its outcome (Execute LP Stake)
func (e *Exchange) ExecuteWeb3LPStake(ctx context.Context, r *Web3LPStakeRequest) (*Web3LPStakeResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.PoolAddress == "" {
		return nil, errPoolAddressRequired
	}
	if r.PayTokenAmount <= 0 {
		return nil, fmt.Errorf("%w: pay token amount", order.ErrAmountIsInvalid)
	}
	if r.PayTokenCode == "" {
		return nil, errTokenCodeRequired
	}
	if r.RangeUpper != 0 && r.PriceUpper != 0 {
		return nil, fmt.Errorf("%w: upper bound", errRangeAndPriceBoundsBothSet)
	}
	if r.RangeLower != 0 && r.PriceLower != 0 {
		return nil, fmt.Errorf("%w: lower bound", errRangeAndPriceBoundsBothSet)
	}
	body := make(map[string]any)
	body["positionId"] = r.PositionID
	body["poolAddress"] = r.PoolAddress
	body["payTokenAmount"] = strconv.FormatFloat(r.PayTokenAmount, 'f', -1, 64)
	body["payTokenCode"] = r.PayTokenCode
	if r.RangeUpper != 0 {
		body["rangeUpper"] = strconv.FormatFloat(r.RangeUpper, 'f', -1, 64)
	}
	if r.RangeLower != 0 {
		body["rangeLower"] = strconv.FormatFloat(r.RangeLower, 'f', -1, 64)
	}
	if r.PriceUpper != 0 {
		body["priceUpper"] = strconv.FormatFloat(r.PriceUpper, 'f', -1, 64)
	}
	if r.PriceLower != 0 {
		body["priceLower"] = strconv.FormatFloat(r.PriceLower, 'f', -1, 64)
	}
	var resp *Web3LPStakeResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/alpha/lp/stake", nil, body, &resp, executeWeb3LPStakeEPL)
}

// ExecuteWeb3LPRedeem withdraws part or all of an LP position. The order is only acknowledged; GetWeb3LPOrderList
// reports its outcome (Execute LP Redeem)
func (e *Exchange) ExecuteWeb3LPRedeem(ctx context.Context, r *Web3LPRedeemRequest) (*Web3OrderResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.PositionID == 0 {
		return nil, errLPPositionIDRequired
	}
	if r.PoolAddress == "" {
		return nil, errPoolAddressRequired
	}
	if r.DercRatio <= 0 || r.DercRatio > 1 {
		return nil, fmt.Errorf("%w: %v", errDercRatioInvalid, r.DercRatio)
	}
	body := make(map[string]any)
	body["positionId"] = r.PositionID
	body["poolAddress"] = r.PoolAddress
	body["dercRatio"] = strconv.FormatFloat(r.DercRatio, 'f', -1, 64)
	if r.ReceiveTokenCode != "" {
		body["receiveTokenCode"] = r.ReceiveTokenCode
	}
	var resp *Web3OrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/alpha/lp/redeem", nil, body, &resp, executeWeb3LPRedeemEPL)
}

// GetWeb3LPOrderList returns LP stake and redeem orders from the last 365 days at most, newest first (Get LP Order
// List)
func (e *Exchange) GetWeb3LPOrderList(ctx context.Context, r *Web3LPOrderListRequest) (*Web3LPOrderListResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	body := make(map[string]any)
	if r.OrderType != 0 {
		body["orderType"] = r.OrderType
	}
	if r.TokenCode != "" {
		body["tokenCode"] = r.TokenCode
	}
	if len(r.OrderStatus) != 0 {
		body["orderStatus"] = r.OrderStatus
	}
	if r.Days != 0 {
		body["days"] = r.Days
	}
	if r.Limit != 0 {
		body["limit"] = r.Limit
	}
	if r.PageIndex != 0 {
		body["pageIndex"] = r.PageIndex
	}
	if r.PoolAddress != "" {
		body["poolAddress"] = r.PoolAddress
	}
	var resp *Web3LPOrderListResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/alpha/lp/order-list", nil, body, &resp, getWeb3LPOrderListEPL)
}

// GetWeb3LPPayTokenList returns the payment tokens accepted for LP staking with their available balances, optionally
// filtered by chainCode and tokenAddress (Get LP Pay Token List)
func (e *Exchange) GetWeb3LPPayTokenList(ctx context.Context, chainCode, tokenAddress string) (*Web3LPPayTokenListResponse, error) {
	body := make(map[string]any)
	if chainCode != "" {
		body["chainCode"] = chainCode
	}
	if tokenAddress != "" {
		body["tokenAddress"] = tokenAddress
	}
	var resp *Web3LPPayTokenListResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/alpha/lp/pay-token-list", nil, body, &resp, getWeb3LPPayTokenListEPL)
}

// GetWeb3LPPayTokenPrice returns the USD prices of up to 50 payment tokens, optionally filtered by chainCode (Get LP
// Pay Token Price)
func (e *Exchange) GetWeb3LPPayTokenPrice(ctx context.Context, tokenCodes []string, chainCode string) (*Web3LPPayTokenPriceResponse, error) {
	if len(tokenCodes) == 0 || slices.Contains(tokenCodes, "") {
		return nil, errTokenCodeRequired
	}
	body := make(map[string]any)
	body["tokenCode"] = tokenCodes
	if chainCode != "" {
		body["chainCode"] = chainCode
	}
	var resp *Web3LPPayTokenPriceResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/alpha/lp/pay-token-price", nil, body, &resp, getWeb3LPPayTokenPriceEPL)
}

// GetWeb3LPPoolInfo returns a liquidity pool's details (Get LP Pool Info)
func (e *Exchange) GetWeb3LPPoolInfo(ctx context.Context, poolAddress string) (*Web3LPPoolInfoResponse, error) {
	if poolAddress == "" {
		return nil, errPoolAddressRequired
	}
	body := make(map[string]any)
	body["poolAddress"] = poolAddress
	var resp *Web3LPPoolInfoResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/alpha/lp/pool-info", nil, body, &resp, getWeb3LPPoolInfoEPL)
}

// GetWeb3LPPoolList returns the available liquidity pools, optionally only those containing tokenSymbol (Get LP Pool
// List)
func (e *Exchange) GetWeb3LPPoolList(ctx context.Context, tokenSymbol currency.Code) (*Web3LPPoolListResponse, error) {
	body := make(map[string]any)
	if !tokenSymbol.IsEmpty() {
		body["tokenSymbol"] = tokenSymbol.String()
	}
	var resp *Web3LPPoolListResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/alpha/lp/pool-list", nil, body, &resp, getWeb3LPPoolListEPL)
}

// GetWeb3LPPositionList returns the user's LP positions (Get LP Position List)
func (e *Exchange) GetWeb3LPPositionList(ctx context.Context) (*Web3LPPositionListResponse, error) {
	var resp *Web3LPPositionListResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/alpha/lp/position-list", nil, nil, &resp, getWeb3LPPositionListEPL)
}

// GetPredictionEngineStatus reports whether the prediction market matching engine accepts buy and sell orders (Get
// Engine Status)
func (e *Exchange) GetPredictionEngineStatus(ctx context.Context) (*PredictionEngineStatusResponse, error) {
	var resp *PredictionEngineStatusResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/alpha/prediction/engine-status", nil, nil, &resp, defaultEPL)
}

// GetPredictionPaymentTokenList returns the payment tokens the prediction market accepts (Get Payment Token List)
func (e *Exchange) GetPredictionPaymentTokenList(ctx context.Context) ([]PredictionPaymentToken, error) {
	var resp []PredictionPaymentToken
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/alpha/prediction/pay-token-list", nil, nil, &resp, defaultEPL)
}

// GetPredictionEventDetail returns a prediction event and its outcome markets, including the token IDs orders need
// (Get Event Detail)
func (e *Exchange) GetPredictionEventDetail(ctx context.Context, r *PredictionEventDetailRequest) (*PredictionEventDetailResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	body := make(map[string]any)
	if r.EventID != "" {
		body["eventId"] = r.EventID
	}
	if r.Slug != "" {
		body["slug"] = r.Slug
	}
	if r.HasMoreMarkets {
		body["hasMoreMarkets"] = true
	}
	var resp *PredictionEventDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/alpha/prediction/event-detail", nil, body, &resp, defaultEPL)
}

// GetPredictionOrderEstimate estimates the cost, proceeds and fees of a prediction market order, which must be shown
// to the user before Execute Buy or Execute Sell (Get Order Estimate)
func (e *Exchange) GetPredictionOrderEstimate(ctx context.Context, r *PredictionOrderEstimateRequest) (*PredictionOrderEstimateResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.TokenID == "" {
		return nil, errOutcomeTokenIDRequired
	}
	if r.Side == 0 {
		return nil, order.ErrSideIsInvalid
	}
	if r.EventID == "" {
		return nil, errEventIDRequired
	}
	if r.Amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	if r.OrderType == 0 {
		return nil, order.ErrTypeIsInvalid
	}
	if r.Side == 1 && r.PayTokenCode == "" {
		return nil, fmt.Errorf("%w: buy orders need a pay token code", errTokenCodeRequired)
	}
	body := make(map[string]any)
	body["tokenId"] = r.TokenID
	body["side"] = r.Side
	body["eventId"] = r.EventID
	body["amount"] = strconv.FormatFloat(r.Amount, 'f', -1, 64)
	body["orderType"] = r.OrderType
	if r.PayTokenCode != "" {
		body["payTokenCode"] = r.PayTokenCode
	}
	var resp *PredictionOrderEstimateResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/alpha/prediction/order-estimate", nil, body, &resp, defaultEPL)
}

// ExecutePredictionBuy buys outcome token shares with USDC. The order is only acknowledged; GetPredictionOrderList
// reports its outcome (Execute Buy)
func (e *Exchange) ExecutePredictionBuy(ctx context.Context, r *PredictionBuyRequest) (*Web3OrderResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.TokenID == "" {
		return nil, errOutcomeTokenIDRequired
	}
	if r.Amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	if r.PayTokenCode == "" {
		return nil, errTokenCodeRequired
	}
	if r.OrderType == 0 {
		return nil, order.ErrTypeIsInvalid
	}
	if r.EventID == "" {
		return nil, errEventIDRequired
	}
	body := make(map[string]any)
	body["tokenId"] = r.TokenID
	body["amount"] = strconv.FormatFloat(r.Amount, 'f', -1, 64)
	body["payTokenCode"] = r.PayTokenCode
	body["orderType"] = r.OrderType
	body["slippage"] = strconv.FormatFloat(r.Slippage, 'f', -1, 64)
	body["eventId"] = r.EventID
	var resp *Web3OrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/alpha/prediction/buy", nil, body, &resp, defaultEPL)
}

// ExecutePredictionSell sells outcome token shares for USDC. The order is only acknowledged; GetPredictionOrderList
// reports its outcome (Execute Sell)
func (e *Exchange) ExecutePredictionSell(ctx context.Context, r *PredictionSellRequest) (*Web3OrderResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.TokenID == "" {
		return nil, errOutcomeTokenIDRequired
	}
	if r.Size <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	if r.OrderType == 0 {
		return nil, order.ErrTypeIsInvalid
	}
	if r.EventID == "" {
		return nil, errEventIDRequired
	}
	body := make(map[string]any)
	body["tokenId"] = r.TokenID
	body["size"] = strconv.FormatFloat(r.Size, 'f', -1, 64)
	body["orderType"] = r.OrderType
	body["slippage"] = strconv.FormatFloat(r.Slippage, 'f', -1, 64)
	body["eventId"] = r.EventID
	if r.ToTokenCode != "" {
		body["toTokenCode"] = r.ToTokenCode
	}
	var resp *Web3OrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/alpha/prediction/sell", nil, body, &resp, defaultEPL)
}

// GetPredictionOrderList returns prediction market orders (Get Order List)
func (e *Exchange) GetPredictionOrderList(ctx context.Context, r *PredictionOrderListRequest) (*PredictionOrderListResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	body := make(map[string]any)
	if r.Status != 0 {
		body["status"] = r.Status
	}
	if r.TokenID != "" {
		body["tokenId"] = r.TokenID
	}
	if r.EventID != "" {
		body["eventId"] = r.EventID
	}
	if r.Side != 0 {
		body["side"] = r.Side
	}
	if r.Days != 0 {
		body["days"] = r.Days
	}
	if r.Limit != 0 {
		body["limit"] = r.Limit
	}
	if r.PageIndex != 0 {
		body["pageIndex"] = r.PageIndex
	}
	var resp *PredictionOrderListResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/alpha/prediction/order-list", nil, body, &resp, defaultEPL)
}

// GetPredictionOrderBook returns the order books of up to 20 outcome tokens (Get Order Book)
func (e *Exchange) GetPredictionOrderBook(ctx context.Context, tokenIDs []string) ([]PredictionOrderBook, error) {
	body, err := predictionTokenIDsBody(tokenIDs)
	if err != nil {
		return nil, err
	}
	var resp []PredictionOrderBook
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/alpha/prediction/order-book", nil, body, &resp, defaultEPL)
}

// GetPredictionTokenPrice returns the best bid, best ask and last price of up to 20 outcome tokens (Get Token Price)
func (e *Exchange) GetPredictionTokenPrice(ctx context.Context, tokenIDs []string) ([]PredictionTokenPrice, error) {
	body, err := predictionTokenIDsBody(tokenIDs)
	if err != nil {
		return nil, err
	}
	var resp []PredictionTokenPrice
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/alpha/prediction/token-price", nil, body, &resp, defaultEPL)
}

// predictionTokenIDsBody builds the body Get Order Book and Get Token Price share
func predictionTokenIDsBody(tokenIDs []string) (map[string]any, error) {
	if len(tokenIDs) == 0 || slices.Contains(tokenIDs, "") {
		return nil, errOutcomeTokenIDRequired
	}
	body := make(map[string]any)
	body["tokenIds"] = tokenIDs
	return body, nil
}

// GetPredictionPriceHistory returns an outcome token's candles at interval (1H, 6H, 1D, 1W, 1M or ALL), newest first
// (Get Price History)
func (e *Exchange) GetPredictionPriceHistory(ctx context.Context, tokenID, interval string) (*PredictionPriceHistoryResponse, error) {
	if tokenID == "" {
		return nil, errOutcomeTokenIDRequired
	}
	if interval == "" {
		return nil, errPriceIntervalRequired
	}
	body := make(map[string]any)
	body["tokenId"] = tokenID
	body["interval"] = interval
	var resp *PredictionPriceHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/alpha/prediction/price-history", nil, body, &resp, defaultEPL)
}

// GetPredictionPositionList returns the user's open prediction market positions (Get Position List)
func (e *Exchange) GetPredictionPositionList(ctx context.Context, r *PredictionPositionListRequest) (*PredictionPositionListResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	body := make(map[string]any)
	if r.TokenID != "" {
		body["tokenId"] = r.TokenID
	}
	if r.EventID != "" {
		body["eventId"] = r.EventID
	}
	if r.Limit != 0 {
		body["limit"] = r.Limit
	}
	if r.PageIndex != 0 {
		body["pageIndex"] = r.PageIndex
	}
	var resp *PredictionPositionListResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/alpha/prediction/position-list", nil, body, &resp, defaultEPL)
}

// GetPredictionPositionHistory returns the user's closed prediction market positions with their settlement results
// (Get Position History)
func (e *Exchange) GetPredictionPositionHistory(ctx context.Context, r *PredictionPositionHistoryRequest) (*PredictionPositionHistoryResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	body := make(map[string]any)
	if r.TokenID != "" {
		body["tokenId"] = r.TokenID
	}
	if r.EventID != "" {
		body["eventId"] = r.EventID
	}
	if r.Result != 0 {
		body["result"] = r.Result
	}
	if r.Days != 0 {
		body["days"] = r.Days
	}
	if r.Limit != 0 {
		body["limit"] = r.Limit
	}
	if r.PageIndex != 0 {
		body["pageIndex"] = r.PageIndex
	}
	var resp *PredictionPositionHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/alpha/prediction/position-history", nil, body, &resp, defaultEPL)
}

// GetPredictionPortfolioSummary summarises the user's prediction market portfolio, optionally for one event type, such
// as 1 for the 2026 FIFA World Cup (Get Portfolio Summary)
func (e *Exchange) GetPredictionPortfolioSummary(ctx context.Context, eventType uint64) (*PredictionPortfolioSummaryResponse, error) {
	body := make(map[string]any)
	if eventType != 0 {
		body["eventType"] = eventType
	}
	var resp *PredictionPortfolioSummaryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/alpha/prediction/portfolio-summary", nil, body, &resp, defaultEPL)
}

// GetPredictionSideMarketList returns the markets of a prediction event (Get Side Market List)
func (e *Exchange) GetPredictionSideMarketList(ctx context.Context, r *PredictionSideMarketListRequest) (*PredictionSideMarketListResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.EventID == "" {
		return nil, errEventIDRequired
	}
	body := make(map[string]any)
	body["eventId"] = r.EventID
	if r.SortBy != "" {
		body["sortBy"] = r.SortBy
	}
	if r.MarketType != 0 {
		body["marketType"] = r.MarketType
	}
	if r.Limit != 0 {
		body["limit"] = r.Limit
	}
	if r.PageIndex != 0 {
		body["pageIndex"] = r.PageIndex
	}
	var resp *PredictionSideMarketListResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/alpha/prediction/side-market-list", nil, body, &resp, defaultEPL)
}

// GetPredictionSportsMatchList returns the sports matches of a prediction event type (Get Sports Match List)
func (e *Exchange) GetPredictionSportsMatchList(ctx context.Context, r *PredictionSportsMatchListRequest) (*PredictionSportsMatchListResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.EventType == 0 {
		return nil, errEventTypeRequired
	}
	// The window bounds the matches' start times, which are in the future for upcoming matches, so only its order is
	// checked
	if !r.StartTime.IsZero() && !r.EndTime.IsZero() && r.StartTime.After(r.EndTime) {
		return nil, common.ErrStartAfterEnd
	}
	body := make(map[string]any)
	body["eventType"] = r.EventType
	if r.Status != 0 {
		body["status"] = r.Status
	}
	if !r.StartTime.IsZero() {
		body["startTime"] = r.StartTime.UnixMilli()
	}
	if !r.EndTime.IsZero() {
		body["endTime"] = r.EndTime.UnixMilli()
	}
	if r.StageCode != "" {
		body["stageCode"] = r.StageCode
	}
	if r.Limit != 0 {
		body["limit"] = r.Limit
	}
	if r.PageIndex != 0 {
		body["pageIndex"] = r.PageIndex
	}
	var resp *PredictionSportsMatchListResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/alpha/prediction/sports/match-list", nil, body, &resp, defaultEPL)
}

// GetPredictionSportsTimelineStages returns the tournament stages of a sports prediction event type (Get Sports
// Timeline Stages)
func (e *Exchange) GetPredictionSportsTimelineStages(ctx context.Context, eventType uint64) (*PredictionSportsTimelineStagesResponse, error) {
	if eventType == 0 {
		return nil, errEventTypeRequired
	}
	params := url.Values{}
	params.Set("eventType", strconv.FormatUint(eventType, 10))
	var resp *PredictionSportsTimelineStagesResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/alpha/prediction/sports/timeline-stages", params, nil, &resp, defaultEPL)
}

// GetPredictionSportsGroupStageDetail returns the group standings and matches of a sports prediction event type,
// optionally for one group (Get Sports Group Stage Detail)
func (e *Exchange) GetPredictionSportsGroupStageDetail(ctx context.Context, eventType uint64, groupName string) (*PredictionSportsGroupStageDetailResponse, error) {
	if eventType == 0 {
		return nil, errEventTypeRequired
	}
	body := make(map[string]any)
	body["eventType"] = eventType
	if groupName != "" {
		body["groupName"] = groupName
	}
	var resp *PredictionSportsGroupStageDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/alpha/prediction/sports/group-stage-detail", nil, body, &resp, defaultEPL)
}
