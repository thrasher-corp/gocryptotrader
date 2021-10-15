package coinbene

import (
	"context"
	"errors"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"golang.org/x/time/rate"
)

const (
	// Contract rate limit time interval and request rates
	contractRateInterval                 = time.Second * 2
	orderbookContractReqRate             = 20
	tickersContractReqRate               = 20
	klineContractReqRate                 = 20
	tradesContractReqRate                = 20
	contractInstrumentsReqRate           = 20
	contractAccountInfoContractReqRate   = 10
	positionInfoContractReqRate          = 10
	placeOrderContractReqRate            = 20
	cancelOrderContractReqRate           = 20
	getOpenOrdersContractReqRate         = 5
	openOrdersByPageContractReqRate      = 5
	getOrderInfoContractReqRate          = 10
	getClosedOrdersContractReqRate       = 5
	getClosedOrdersbyPageContractReqRate = 5
	cancelMultipleOrdersContractReqRate  = 5
	getOrderFillsContractReqRate         = 10
	getFundingRatesContractReqRate       = 10

	// Spot rate limit time interval and request rates
	spotRateInterval             = time.Second
	getPairsSpotReqRate          = 2
	getPairsInfoSpotReqRate      = 3
	getOrderbookSpotReqRate      = 6
	getTickerListSpotReqRate     = 6
	getSpecificTickerSpotReqRate = 6
	getMarketTradesSpotReqRate   = 3
	// getKlineSpotReqRate              = 1
	// getExchangeRateSpotReqRate       = 1
	getAccountInfoSpotReqRate        = 3
	queryAccountAssetInfoSpotReqRate = 6
	placeOrderSpotReqRate            = 6
	batchOrderSpotReqRate            = 3
	queryOpenOrdersSpotReqRate       = 3
	queryClosedOrdersSpotReqRate     = 3
	querySpecficOrderSpotReqRate     = 6
	queryTradeFillsSpotReqRate       = 3
	cancelOrderSpotReqRate           = 6
	cancelOrdersBatchSpotReqRate     = 3
	capitalDepositReqRate            = 1
	capitalWithdrawReqRate           = 1

	// Rate limit functionality
	contractOrderbook request.EndpointLimit = iota
	contractTickers
	contractKline
	contractTrades
	contractInstruments
	contractAccountInfo
	contractPositionInfo
	contractPlaceOrder
	contractCancelOrder
	contractGetOpenOrders
	contractOpenOrdersByPage
	contractGetOrderInfo
	contractGetClosedOrders
	contractGetClosedOrdersbyPage
	contractCancelMultipleOrders
	contractGetOrderFills
	contractGetFundingRates

	spotPairs
	spotPairInfo
	spotOrderbook
	spotTickerList
	spotSpecificTicker
	spotMarketTrades
	spotKline        // Not implemented yet
	spotExchangeRate // Not implemented yet
	spotAccountInfo
	spotAccountAssetInfo
	spotPlaceOrder
	spotBatchOrder
	spotQueryOpenOrders
	spotQueryClosedOrders
	spotQuerySpecficOrder
	spotQueryTradeFills
	spotCancelOrder
	spotCancelOrdersBatch
	capitalDeposit
	capitalWithdraw
)

// RateLimit implements the request.Limiter interface
type RateLimit struct {
	ContractOrderbook             *rate.Limiter
	ContractTickers               *rate.Limiter
	ContractKline                 *rate.Limiter
	ContractTrades                *rate.Limiter
	ContractInstruments           *rate.Limiter
	ContractAccountInfo           *rate.Limiter
	ContractPositionInfo          *rate.Limiter
	ContractPlaceOrder            *rate.Limiter
	ContractCancelOrder           *rate.Limiter
	ContractGetOpenOrders         *rate.Limiter
	ContractOpenOrdersByPage      *rate.Limiter
	ContractGetOrderInfo          *rate.Limiter
	ContractGetClosedOrders       *rate.Limiter
	ContractGetClosedOrdersbyPage *rate.Limiter
	ContractCancelMultipleOrders  *rate.Limiter
	ContractGetOrderFills         *rate.Limiter
	ContractGetFundingRates       *rate.Limiter
	SpotPairs                     *rate.Limiter
	SpotPairInfo                  *rate.Limiter
	SpotOrderbook                 *rate.Limiter
	SpotTickerList                *rate.Limiter
	SpotSpecificTicker            *rate.Limiter
	SpotMarketTrades              *rate.Limiter
	// spotKline        // Not implemented yet
	// spotExchangeRate // Not implemented yet
	SpotAccountInfo       *rate.Limiter
	SpotAccountAssetInfo  *rate.Limiter
	SpotPlaceOrder        *rate.Limiter
	SpotBatchOrder        *rate.Limiter
	SpotQueryOpenOrders   *rate.Limiter
	SpotQueryClosedOrders *rate.Limiter
	SpotQuerySpecficOrder *rate.Limiter
	SpotQueryTradeFills   *rate.Limiter
	SpotCancelOrder       *rate.Limiter
	SpotCancelOrdersBatch *rate.Limiter
	CapitalDeposit        *rate.Limiter
	CapitalWithdraw       *rate.Limiter
}

// Limit limits outbound requests
func (r *RateLimit) Limit(ctx context.Context, f request.EndpointLimit) error {
	switch f {
	case contractOrderbook:
		return r.ContractOrderbook.Wait(ctx)
	case contractTickers:
		return r.ContractTickers.Wait(ctx)
	case contractKline:
		return r.ContractKline.Wait(ctx)
	case contractTrades:
		return r.ContractTrades.Wait(ctx)
	case contractInstruments:
		return r.ContractInstruments.Wait(ctx)
	case contractAccountInfo:
		return r.ContractAccountInfo.Wait(ctx)
	case contractPositionInfo:
		return r.ContractPositionInfo.Wait(ctx)
	case contractPlaceOrder:
		return r.ContractPlaceOrder.Wait(ctx)
	case contractCancelOrder:
		return r.ContractCancelOrder.Wait(ctx)
	case contractGetOpenOrders:
		return r.ContractGetOpenOrders.Wait(ctx)
	case contractOpenOrdersByPage:
		return r.ContractOpenOrdersByPage.Wait(ctx)
	case contractGetOrderInfo:
		return r.ContractGetOrderInfo.Wait(ctx)
	case contractGetClosedOrders:
		return r.ContractGetClosedOrders.Wait(ctx)
	case contractGetClosedOrdersbyPage:
		return r.ContractGetClosedOrdersbyPage.Wait(ctx)
	case contractCancelMultipleOrders:
		return r.ContractCancelMultipleOrders.Wait(ctx)
	case contractGetOrderFills:
		return r.ContractGetOrderFills.Wait(ctx)
	case contractGetFundingRates:
		return r.ContractGetFundingRates.Wait(ctx)
	case spotPairs:
		return r.SpotPairs.Wait(ctx)
	case spotPairInfo:
		return r.SpotPairInfo.Wait(ctx)
	case spotOrderbook:
		return r.SpotOrderbook.Wait(ctx)
	case spotTickerList:
		return r.SpotTickerList.Wait(ctx)
	case spotSpecificTicker:
		return r.SpotSpecificTicker.Wait(ctx)
	case spotMarketTrades:
		return r.SpotMarketTrades.Wait(ctx)
	case capitalDeposit:
		return r.CapitalDeposit.Wait(ctx)
	case capitalWithdraw:
		return r.CapitalWithdraw.Wait(ctx)
	// case spotKline: // Not implemented yet
	// 	return r.SpotKline.Wait(ctx)
	// case spotExchangeRate:
	// 	return r.SpotExchangeRate.Wait(ctx)
	case spotAccountInfo:
		return r.SpotAccountInfo.Wait(ctx)
	case spotAccountAssetInfo:
		return r.SpotAccountAssetInfo.Wait(ctx)
	case spotPlaceOrder:
		return r.SpotPlaceOrder.Wait(ctx)
	case spotBatchOrder:
		return r.SpotBatchOrder.Wait(ctx)
	case spotQueryOpenOrders:
		return r.SpotQueryOpenOrders.Wait(ctx)
	case spotQueryClosedOrders:
		return r.SpotQueryClosedOrders.Wait(ctx)
	case spotQuerySpecficOrder:
		return r.SpotQuerySpecficOrder.Wait(ctx)
	case spotQueryTradeFills:
		return r.SpotQueryTradeFills.Wait(ctx)
	case spotCancelOrder:
		return r.SpotCancelOrder.Wait(ctx)
	case spotCancelOrdersBatch:
		return r.SpotCancelOrdersBatch.Wait(ctx)
	default:
		return errors.New("rate limit endpoint functionality not set")
	}
}

// SetRateLimit returns the rate limit for the exchange
func SetRateLimit() *RateLimit {
	return &RateLimit{
		ContractOrderbook:             request.NewRateLimit(contractRateInterval, orderbookContractReqRate),
		ContractTickers:               request.NewRateLimit(contractRateInterval, tickersContractReqRate),
		ContractKline:                 request.NewRateLimit(contractRateInterval, klineContractReqRate),
		ContractTrades:                request.NewRateLimit(contractRateInterval, tradesContractReqRate),
		ContractInstruments:           request.NewRateLimit(contractRateInterval, contractInstrumentsReqRate),
		ContractAccountInfo:           request.NewRateLimit(contractRateInterval, contractAccountInfoContractReqRate),
		ContractPositionInfo:          request.NewRateLimit(contractRateInterval, positionInfoContractReqRate),
		ContractPlaceOrder:            request.NewRateLimit(contractRateInterval, placeOrderContractReqRate),
		ContractCancelOrder:           request.NewRateLimit(contractRateInterval, cancelOrderContractReqRate),
		ContractGetOpenOrders:         request.NewRateLimit(contractRateInterval, getOpenOrdersContractReqRate),
		ContractOpenOrdersByPage:      request.NewRateLimit(contractRateInterval, openOrdersByPageContractReqRate),
		ContractGetOrderInfo:          request.NewRateLimit(contractRateInterval, getOrderInfoContractReqRate),
		ContractGetClosedOrders:       request.NewRateLimit(contractRateInterval, getClosedOrdersContractReqRate),
		ContractGetClosedOrdersbyPage: request.NewRateLimit(contractRateInterval, getClosedOrdersbyPageContractReqRate),
		ContractCancelMultipleOrders:  request.NewRateLimit(contractRateInterval, cancelMultipleOrdersContractReqRate),
		ContractGetOrderFills:         request.NewRateLimit(contractRateInterval, getOrderFillsContractReqRate),
		ContractGetFundingRates:       request.NewRateLimit(contractRateInterval, getFundingRatesContractReqRate),
		SpotPairs:                     request.NewRateLimit(spotRateInterval, getPairsSpotReqRate),
		SpotPairInfo:                  request.NewRateLimit(spotRateInterval, getPairsInfoSpotReqRate),
		SpotOrderbook:                 request.NewRateLimit(spotRateInterval, getOrderbookSpotReqRate),
		SpotTickerList:                request.NewRateLimit(spotRateInterval, getTickerListSpotReqRate),
		SpotSpecificTicker:            request.NewRateLimit(spotRateInterval, getSpecificTickerSpotReqRate),
		SpotMarketTrades:              request.NewRateLimit(spotRateInterval, getMarketTradesSpotReqRate),
		SpotAccountInfo:               request.NewRateLimit(spotRateInterval, getAccountInfoSpotReqRate),
		SpotAccountAssetInfo:          request.NewRateLimit(spotRateInterval, queryAccountAssetInfoSpotReqRate),
		SpotPlaceOrder:                request.NewRateLimit(spotRateInterval, placeOrderSpotReqRate),
		SpotBatchOrder:                request.NewRateLimit(spotRateInterval, batchOrderSpotReqRate),
		SpotQueryOpenOrders:           request.NewRateLimit(spotRateInterval, queryOpenOrdersSpotReqRate),
		SpotQueryClosedOrders:         request.NewRateLimit(spotRateInterval, queryClosedOrdersSpotReqRate),
		SpotQuerySpecficOrder:         request.NewRateLimit(spotRateInterval, querySpecficOrderSpotReqRate),
		SpotQueryTradeFills:           request.NewRateLimit(spotRateInterval, queryTradeFillsSpotReqRate),
		SpotCancelOrder:               request.NewRateLimit(spotRateInterval, cancelOrderSpotReqRate),
		SpotCancelOrdersBatch:         request.NewRateLimit(spotRateInterval, cancelOrdersBatchSpotReqRate),
		CapitalDeposit:                request.NewRateLimit(spotRateInterval, capitalDepositReqRate),
		CapitalWithdraw:               request.NewRateLimit(spotRateInterval, capitalWithdrawReqRate),
	}
}
