package coinbene

import (
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
}

// Limit limits outbound requests
func (r *RateLimit) Limit(f request.EndpointLimit) error {
	switch f {
	case contractOrderbook:
		time.Sleep(r.ContractOrderbook.Reserve().Delay())
	case contractTickers:
		time.Sleep(r.ContractTickers.Reserve().Delay())
	case contractKline:
		time.Sleep(r.ContractKline.Reserve().Delay())
	case contractTrades:
		time.Sleep(r.ContractTrades.Reserve().Delay())
	case contractInstruments:
		time.Sleep(r.ContractInstruments.Reserve().Delay())
	case contractAccountInfo:
		time.Sleep(r.ContractAccountInfo.Reserve().Delay())
	case contractPositionInfo:
		time.Sleep(r.ContractPositionInfo.Reserve().Delay())
	case contractPlaceOrder:
		time.Sleep(r.ContractPlaceOrder.Reserve().Delay())
	case contractCancelOrder:
		time.Sleep(r.ContractCancelOrder.Reserve().Delay())
	case contractGetOpenOrders:
		time.Sleep(r.ContractGetOpenOrders.Reserve().Delay())
	case contractOpenOrdersByPage:
		time.Sleep(r.ContractOpenOrdersByPage.Reserve().Delay())
	case contractGetOrderInfo:
		time.Sleep(r.ContractGetOrderInfo.Reserve().Delay())
	case contractGetClosedOrders:
		time.Sleep(r.ContractGetClosedOrders.Reserve().Delay())
	case contractGetClosedOrdersbyPage:
		time.Sleep(r.ContractGetClosedOrdersbyPage.Reserve().Delay())
	case contractCancelMultipleOrders:
		time.Sleep(r.ContractCancelMultipleOrders.Reserve().Delay())
	case contractGetOrderFills:
		time.Sleep(r.ContractGetOrderFills.Reserve().Delay())
	case contractGetFundingRates:
		time.Sleep(r.ContractGetFundingRates.Reserve().Delay())
	case spotPairs:
		time.Sleep(r.SpotPairs.Reserve().Delay())
	case spotPairInfo:
		time.Sleep(r.SpotPairInfo.Reserve().Delay())
	case spotOrderbook:
		time.Sleep(r.SpotOrderbook.Reserve().Delay())
	case spotTickerList:
		time.Sleep(r.SpotTickerList.Reserve().Delay())
	case spotSpecificTicker:
		time.Sleep(r.SpotSpecificTicker.Reserve().Delay())
	case spotMarketTrades:
		time.Sleep(r.SpotMarketTrades.Reserve().Delay())
	// case spotKline: // Not implemented yet
	// 	time.Sleep(r.SpotKline.Reserve().Delay())
	// case spotExchangeRate:
	// 	time.Sleep(r.SpotExchangeRate.Reserve().Delay())
	case spotAccountInfo:
		time.Sleep(r.SpotAccountInfo.Reserve().Delay())
	case spotAccountAssetInfo:
		time.Sleep(r.SpotAccountAssetInfo.Reserve().Delay())
	case spotPlaceOrder:
		time.Sleep(r.SpotPlaceOrder.Reserve().Delay())
	case spotBatchOrder:
		time.Sleep(r.SpotBatchOrder.Reserve().Delay())
	case spotQueryOpenOrders:
		time.Sleep(r.SpotQueryOpenOrders.Reserve().Delay())
	case spotQueryClosedOrders:
		time.Sleep(r.SpotQueryClosedOrders.Reserve().Delay())
	case spotQuerySpecficOrder:
		time.Sleep(r.SpotQuerySpecficOrder.Reserve().Delay())
	case spotQueryTradeFills:
		time.Sleep(r.SpotQueryTradeFills.Reserve().Delay())
	case spotCancelOrder:
		time.Sleep(r.SpotCancelOrder.Reserve().Delay())
	case spotCancelOrdersBatch:
		time.Sleep(r.SpotCancelOrdersBatch.Reserve().Delay())
	default:
		return errors.New("rate limit error endpoint functionality not set")
	}
	return nil
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
	}
}
