package gateio

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

var (
	errTableIDEmpty     = errors.New("tableID cannot be empty")
	errInvalidRiskLimit = errors.New("invalid risk limit")
	errPagingNotAllowed = errors.New("limit/offset pagination params not allowed when contract supplied")
)

// GetUnifiedUserRiskUnitDetails retrieves the user's risk unit details
func (e *Exchange) GetUnifiedUserRiskUnitDetails(ctx context.Context) (*UserRiskUnitDetails, error) {
	var result *UserRiskUnitDetails
	return result, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, unifiedUserRiskUnitDetailsEPL, http.MethodGet, "unified/risk_units", nil, nil, &result)
}

// GetFuturesRiskTable retrieves the futures risk table for a given settlement currency and table ID
func (e *Exchange) GetFuturesRiskTable(ctx context.Context, settleCurrency currency.Code, tableID string) ([]RiskTable, error) {
	if settleCurrency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if tableID == "" {
		return nil, errTableIDEmpty
	}
	var result []RiskTable
	path := futuresPath + settleCurrency.Lower().String() + "/risk_limit_table?table_id=" + tableID
	return result, e.SendHTTPRequest(ctx, exchange.RestSpot, publicFuturesRiskTableEPL, path, &result)
}

// GetFuturesRiskLimitTiers retrieves the futures risk limit tiers
// NOTE: 'limit' and 'offset' correspond to pagination queries at the market level, not to the length of the returned
// array. This only takes effect when the contract parameter is empty.
func (e *Exchange) GetFuturesRiskLimitTiers(ctx context.Context, settleCurrency currency.Code, contract currency.Pair, limit, offset uint64) ([]RiskTable, error) {
	return e.getRiskLimitTiers(ctx, futuresPath, publicFuturesRiskLimitTiersEPL, settleCurrency, contract, limit, offset)
}

// GetDeliveryRiskLimitTiers retrieves the delivery risk limit tiers
// NOTE: 'limit' and 'offset' correspond to pagination queries at the market level, not to the length of the returned
// array. This only takes effect when the contract parameter is empty.
func (e *Exchange) GetDeliveryRiskLimitTiers(ctx context.Context, settleCurrency currency.Code, contract currency.Pair, limit, offset uint64) ([]RiskTable, error) {
	return e.getRiskLimitTiers(ctx, deliveryPath, publicDeliveryRiskLimitTiersEPL, settleCurrency, contract, limit, offset)
}

func (e *Exchange) getRiskLimitTiers(ctx context.Context, assetPath string, epl request.EndpointLimit, settleCurrency currency.Code, contract currency.Pair, limit, offset uint64) ([]RiskTable, error) {
	if settleCurrency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}

	params := url.Values{}
	if !contract.IsEmpty() {
		if limit > 0 || offset > 0 {
			return nil, errPagingNotAllowed
		}
		params.Set("contract", contract.Upper().String())
	} else {
		if limit > 0 {
			params.Set("limit", strconv.FormatUint(limit, 10))
		}
		if offset > 0 {
			params.Set("offset", strconv.FormatUint(offset, 10))
		}
	}

	path := common.EncodeURLValues(assetPath+settleCurrency.Lower().String()+"/risk_limit_tiers", params)

	var result []RiskTable
	return result, e.SendHTTPRequest(ctx, exchange.RestSpot, epl, path, &result)
}

// DeliveryUpdatePositionRiskLimit updates the position risk limit for a delivery contract
func (e *Exchange) DeliveryUpdatePositionRiskLimit(ctx context.Context, settleCurrency currency.Code, contract currency.Pair, riskLimit float64) (*Position, error) {
	return e.updatePositionRiskLimit(ctx, deliveryPath, positionsPath, deliveryUpdateRiskLimitEPL, settleCurrency, contract, riskLimit)
}

// FuturesUpdatePositionRiskLimit updates the position risk limit for a futures contract
func (e *Exchange) FuturesUpdatePositionRiskLimit(ctx context.Context, settleCurrency currency.Code, contract currency.Pair, riskLimit float64) (*Position, error) {
	return e.updatePositionRiskLimit(ctx, futuresPath, positionsPath, perpetualUpdateRiskEPL, settleCurrency, contract, riskLimit)
}

// FuturesUpdatePositionRiskLimitDualMode updates the position risk limit for a futures contract in dual/hedge mode
func (e *Exchange) FuturesUpdatePositionRiskLimitDualMode(ctx context.Context, settleCurrency currency.Code, contract currency.Pair, riskLimit float64) (*Position, error) {
	return e.updatePositionRiskLimit(ctx, futuresPath, hedgeModePath, perpetualUpdateRiskDualModeEPL, settleCurrency, contract, riskLimit)
}

func (e *Exchange) updatePositionRiskLimit(ctx context.Context, assetPath, positionsTypePath string, epl request.EndpointLimit, settleCurrency currency.Code, contract currency.Pair, riskLimit float64) (*Position, error) {
	if settleCurrency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if contract.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if riskLimit <= 0 {
		return nil, errInvalidRiskLimit
	}
	path := assetPath + settleCurrency.Lower().String() + positionsTypePath + contract.Upper().String() + "/risk_limit"
	param := url.Values{}
	param.Set("risk_limit", strconv.FormatFloat(riskLimit, 'f', -1, 64))

	if positionsTypePath == hedgeModePath {
		var result []Position
		if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, epl, http.MethodPost, path, param, nil, &result); err != nil {
			return nil, err
		}
		// Endpoint returns an array but only one position is expected
		if len(result) != 1 {
			return nil, common.ErrNoResults
		}
		return &result[0], nil
	}

	var result Position
	return &result, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, epl, http.MethodPost, path, param, nil, &result)
}
