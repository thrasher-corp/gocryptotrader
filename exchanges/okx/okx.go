package okx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Okx is the overarching type across this package
type Okx struct {
	exchange.Base
	WsResponseMultiplexer wsRequestDataChannelsMultiplexer

	// WsRequestSemaphore channel is used to block write operation on the websocket connection to reduce contention; a kind of bounded parallelism.
	// it is made to hold up to 20 integers so that up to 20 write operations can be called over the websocket connection at a time.
	// and when the operation is completed the thread releases (consumes) one value from the channel so that the other waiting operation can enter.
	// ok.WsRequestSemaphore <- 1
	// defer func() { <-ok.WsRequestSemaphore }()
	WsRequestSemaphore chan int
}

const (
	baseURL       = "https://www.okx.com/"
	okxAPIURL     = baseURL + okxAPIPath
	okxAPIVersion = "/v5/"
	tradeSpot     = "trade-spot/"
	tradeMargin   = "trade-margin/"
	tradeFutures  = "trade-futures/"
	tradePerps    = "trade-swap/"
	tradeOptions  = "trade-option/"

	okxAPIPath      = "api" + okxAPIVersion
	okxWebsocketURL = "wss://ws.okx.com:8443/ws" + okxAPIVersion

	okxAPIWebsocketPublicURL  = okxWebsocketURL + "public"
	okxAPIWebsocketPrivateURL = okxWebsocketURL + "private"
)

var (
	errNo24HrTradeVolumeFound                 = errors.New("no trade record found in the 24 trade volume")
	errOracleInformationNotFound              = errors.New("oracle information not found")
	errExchangeInfoNotFound                   = errors.New("exchange information not found")
	errIndexComponentNotFound                 = errors.New("unable to fetch index components")
	errLimitValueExceedsMaxOf100              = errors.New("limit value exceeds the maximum value 100")
	errMissingInstrumentID                    = errors.New("missing instrument id")
	errFundingRateHistoryNotFound             = errors.New("funding rate history not found")
	errMissingRequiredUnderlying              = errors.New("error missing required parameter underlying")
	errMissingRequiredParamInstID             = errors.New("missing required parameter instrument id")
	errLiquidationOrderResponseNotFound       = errors.New("liquidation order not found")
	errEitherInstIDOrCcyIsRequired            = errors.New("either parameter instId or ccy is required")
	errIncorrectRequiredParameterTradeMode    = errors.New("unacceptable required argument, trade mode")
	errInterestRateAndLoanQuotaNotFound       = errors.New("interest rate and loan quota not found")
	errInsuranceFundInformationNotFound       = errors.New("insurance fund information not found")
	errMissingExpiryTimeParameter             = errors.New("missing expiry date parameter")
	errInvalidTradeModeValue                  = errors.New("invalid trade mode value")
	errMissingClientOrderIDOrOrderID          = errors.New("client order id or order id is missing")
	errWebsocketStreamNotAuthenticated        = errors.New("websocket stream not authenticated")
	errInvalidNewSizeOrPriceInformation       = errors.New("invalid the new size or price information")
	errSizeOrPriceIsRequired                  = errors.New("either size or price is required")
	errMissingNewSize                         = errors.New("missing the order size information")
	errMissingMarginMode                      = errors.New("missing required param margin mode 'mgnMode'")
	errInvalidTriggerPrice                    = errors.New("invalid trigger price value")
	errInvalidPriceLimit                      = errors.New("invalid price limit value")
	errMissingIntervalValue                   = errors.New("missing interval value")
	errMissingTakeProfitTriggerPrice          = errors.New("missing take profit trigger price")
	errMissingTakeProfitOrderPrice            = errors.New("missing take profit order price")
	errMissingSizeLimit                       = errors.New("missing required parameter 'szLimit'")
	errMissingEitherAlgoIDOrState             = errors.New("either algo id or order state is required")
	errUnacceptableAmount                     = errors.New("amount must be greater than 0")
	errMissingValidWithdrawalID               = errors.New("missing valid withdrawal id")
	errInstrumentFamilyRequired               = errors.New("instrument family is required")
	errCountdownTimeoutRequired               = errors.New("countdown timeout is required")
	errInstrumentIDorFamilyRequired           = errors.New("either instrumen id or instrument family is required")
	errInvalidQuantityLimit                   = errors.New("invalid quantity limit")
	errInstrumentTypeRequired                 = errors.New("instrument type required")
	errInvalidInstrumentType                  = errors.New("invalid instrument type")
	errMissingValidGreeksType                 = errors.New("missing valid greeks type")
	errMissingIsolatedMarginTradingSetting    = errors.New("missing isolated margin trading setting, isolated margin trading settings automatic:Auto transfers autonomy:Manual transfers")
	errInvalidCounterParties                  = errors.New("missing counter parties")
	errMissingRfqIDAndClientRfqID             = errors.New("missing rfq id or client rfq id")
	errMissingRfqIDOrQuoteID                  = errors.New("either Rfq ID or Quote ID is missing")
	errMissingRfqID                           = errors.New("error missing rfq id")
	errMissingLegs                            = errors.New("missing legs")
	errMissingSizeOfQuote                     = errors.New("missing size of quote leg")
	errMossingLegsQuotePrice                  = errors.New("error missing quote price")
	errMissingQuoteIDOrClientQuoteID          = errors.New("missing quote id or client quote id")
	errMissingEitherQuoteIDAOrClientQuoteIDs  = errors.New("missing either quote ids or client quote ids")
	errMissingRequiredParameterSubaccountName = errors.New("missing required parameter subaccount name")
	errInvalidLoanAllocationValue             = errors.New("invalid loan allocation value, must be between 0 to 100")
	errInvalidSubaccount                      = errors.New("invalid account type")
	errMissingDestinationSubaccountName       = errors.New("missing destination subaccount name")
	errMissingInitialSubaccountName           = errors.New("missing initial subaccount name")
	errMissingAlgoOrderType                   = errors.New("missing algo order type 'grid': Spot grid, \"contract_grid\": Contract grid")
	errInvalidMaximumPrice                    = errors.New("invalid maximum price")
	errInvalidMinimumPrice                    = errors.New("invalid minimum price")
	errInvalidGridQuantity                    = errors.New("invalid grid quantity (grid number)")
	errMissingRequiredArgumentDirection       = errors.New("missing required argument, direction")
	errRequiredParameterMissingLeverage       = errors.New("missing required parameter, leverage")
	errMissingValidStopType                   = errors.New("invalid grid order stop type, only values are \"1\" and \"2\" ")
	errMissingSubOrderType                    = errors.New("missing sub order type")
	errMissingQuantity                        = errors.New("invalid quantity to buy or sell")
	errDepositAddressNotFound                 = errors.New("deposit address with the specified currency code and chain not found")
	errAddressRequired                        = errors.New("address is required")
	errInvalidWebsocketEvent                  = errors.New("invalid websocket event")
	errMissingValidChannelInformation         = errors.New("missing channel information")
	errMaxRfqOrdersToCancel                   = errors.New("no more than 100 Rfq cancel order parameter is allowed")
	errMalformedData                          = errors.New("malformed data")
	errInvalidUnderlying                      = errors.New("invalid underlying")
	errInstrumentFamilyOrUnderlyingRequired   = errors.New("either underlying or instrument family is required")
	errMissingRequiredParameter               = errors.New("missing required parameter")
	errMissingMakerInstrumentSettings         = errors.New("missing maker instrument settings")
	errInvalidSubAccountName                  = errors.New("invalid sub-account name")
	errInvalidAPIKey                          = errors.New("invalid api key")
	errInvalidMarginTypeAdjust                = errors.New("invalid margin type adjust, only 'add' and 'reduce' are allowed")
	errInvalidAlgoOrderType                   = errors.New("invalid algo order type")
	errInvalidIPAddress                       = errors.New("invalid ip address")
	errInvalidAPIKeyPermission                = errors.New("invalid API Key permission")
	errInvalidResponseParam                   = errors.New("invalid response parameter, response must be non-nil pointer")
	errTooManyArgument                        = errors.New("too many cancel request params")
	errInvalidDuration                        = errors.New("invalid grid contract duration, only '7D', '30D', and '180D' are allowed")
	errInvalidProtocolType                    = errors.New("invalid protocol type, only 'staking' and 'defi' allowed")
	errExceedLimit                            = errors.New("limit exceeded")
	errOnlyThreeMonthsSupported               = errors.New("only three months of trade data retrieval supported")
	errOnlyOneResponseExpected                = errors.New("one response item expected")
	errNoInstrumentFound                      = errors.New("no instrument found")
	errStrategyNameRequired                   = errors.New("strategy name required")
	errSubPositionIDRequired                  = errors.New("sub position id is required")
	errUserIDRequired                         = errors.New("uid is required")
	errSubPositionCloseTypeRequired           = errors.New("sub position close type")
	errUniqueCodeRequired                     = errors.New("unique code is required")
	errLastDaysRequired                       = errors.New("last days required")
	errCopyInstrumentIDTypeRequired           = errors.New("copy instrument ID type is required")
	errInvalidChecksum                        = errors.New("invalid checksum")
	errInvalidPositionMode                    = errors.New("invalid position mode")
	errLendingTermIsRequired                  = errors.New("lending term is required")
	errLendingRateRequired                    = errors.New("lending rate is required")
)

/************************************ MarketData Endpoints *************************************************/

// OrderTypeFromString returns order.Type instance from string
func (ok *Okx) OrderTypeFromString(orderType string) (order.Type, error) {
	switch strings.ToLower(orderType) {
	case OkxOrderMarket:
		return order.Market, nil
	case OkxOrderLimit:
		return order.Limit, nil
	case OkxOrderPostOnly:
		return order.PostOnly, nil
	case OkxOrderFOK:
		return order.FillOrKill, nil
	case OkxOrderIOC:
		return order.ImmediateOrCancel, nil
	case OkxOrderOptimalLimitIOC:
		return order.OptimalLimitIOC, nil
	default:
		return order.UnknownType, fmt.Errorf("%w %v", order.ErrTypeIsInvalid, orderType)
	}
}

// OrderTypeString returns a string representation of order.Type instance
func (ok *Okx) OrderTypeString(orderType order.Type) (string, error) {
	switch orderType {
	case order.Market:
		return OkxOrderMarket, nil
	case order.Limit:
		return OkxOrderLimit, nil
	case order.PostOnly:
		return OkxOrderPostOnly, nil
	case order.FillOrKill:
		return OkxOrderFOK, nil
	case order.IOS:
		return OkxOrderIOC, nil
	case order.OptimalLimitIOC:
		return OkxOrderOptimalLimitIOC, nil
	default:
		return "", fmt.Errorf("%w %v", order.ErrTypeIsInvalid, orderType)
	}
}

// PlaceOrder place an order only if you have sufficient funds.
func (ok *Okx) PlaceOrder(ctx context.Context, arg *PlaceOrderRequestParam, a asset.Item) (*OrderData, error) {
	if arg == nil || *arg == (PlaceOrderRequestParam{}) {
		return nil, common.ErrNilPointer
	}
	arg.AssetType = a
	err := ok.validatePlaceOrderParams(arg)
	if err != nil {
		return nil, err
	}
	var resp *OrderData
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, placeOrderEPL, http.MethodPost, "trade/order", &arg, &resp, true)
}

func (ok *Okx) validatePlaceOrderParams(arg *PlaceOrderRequestParam) error {
	if arg == nil || *arg == (PlaceOrderRequestParam{}) {
		return common.ErrNilPointer
	}
	if arg.InstrumentID == "" {
		return errMissingInstrumentID
	}
	arg.Side = strings.ToLower(arg.Side)
	if arg.Side != order.Buy.Lower() && arg.Side != order.Sell.Lower() {
		return fmt.Errorf("%w %s", order.ErrSideIsInvalid, arg.Side)
	}
	if arg.TradeMode != "" &&
		arg.TradeMode != TradeModeCross &&
		arg.TradeMode != TradeModeIsolated &&
		arg.TradeMode != TradeModeCash {
		return fmt.Errorf("%w %s", errInvalidTradeModeValue, arg.TradeMode)
	}
	if arg.PositionSide != "" {
		if (arg.PositionSide == positionSideLong || arg.PositionSide == positionSideShort) &&
			(arg.AssetType != asset.Futures && arg.AssetType != asset.PerpetualSwap) {
			return errInvalidPositionMode
		}
	}
	arg.OrderType = strings.ToLower(arg.OrderType)
	if arg.OrderType == order.OptimalLimitIOC.Lower() &&
		(arg.AssetType != asset.Futures && arg.AssetType != asset.PerpetualSwap) {
		return errors.New("\"optimal_limit_ioc\": market order with immediate-or-cancel order (applicable only to Futures and Perpetual swap)")
	}
	if arg.OrderType != OkxOrderMarket &&
		arg.OrderType != OkxOrderLimit &&
		arg.OrderType != OkxOrderPostOnly &&
		arg.OrderType != OkxOrderFOK &&
		arg.OrderType != OkxOrderIOC &&
		arg.OrderType != OkxOrderOptimalLimitIOC {
		return fmt.Errorf("%w %v", order.ErrTypeIsInvalid, arg.OrderType)
	}
	if arg.Amount <= 0 {
		return order.ErrAmountBelowMin
	}
	if arg.QuantityType != "" && arg.QuantityType != "base_ccy" && arg.QuantityType != "quote_ccy" {
		return errors.New("only base_ccy and quote_ccy quantity types are supported")
	}
	return nil
}

// PlaceMultipleOrders  to place orders in batches. Maximum 20 orders can be placed at a time. Request parameters should be passed in the form of an array.
func (ok *Okx) PlaceMultipleOrders(ctx context.Context, args []PlaceOrderRequestParam) ([]OrderData, error) {
	if len(args) == 0 {
		return nil, order.ErrSubmissionIsNil
	}
	var err error
	for x := range args {
		err = ok.validatePlaceOrderParams(&args[x])
		if err != nil {
			return nil, err
		}
	}
	var resp []OrderData
	err = ok.SendHTTPRequest(ctx, exchange.RestSpot, placeMultipleOrdersEPL, http.MethodPost, "trade/batch-orders", &args, &resp, true)
	if err != nil {
		if len(resp) == 0 {
			return nil, err
		}
		var errs error
		for x := range resp {
			errs = common.AppendError(errs, fmt.Errorf("error code:%s message: %v", resp[x].SCode, resp[x].SMessage))
		}
		return nil, errs
	}
	return resp, nil
}

// CancelSingleOrder cancel an incomplete order.
func (ok *Okx) CancelSingleOrder(ctx context.Context, arg *CancelOrderRequestParam) (*OrderData, error) {
	if arg == nil || *arg == (CancelOrderRequestParam{}) {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	if arg.OrderID == "" && arg.ClientOrderID == "" {
		return nil, errors.New("either order id or client id is required")
	}
	var resp []OrderData
	err := ok.SendHTTPRequest(ctx, exchange.RestSpot, cancelOrderEPL, http.MethodPost, "trade/cancel-order", &arg, &resp, true)
	if err != nil {
		if len(resp) != 1 {
			return nil, err
		}
		return nil, fmt.Errorf("error code:%s message: %v", resp[0].SCode, resp[0].SMessage)
	}
	if len(resp) == 1 {
		return &resp[0], nil
	}
	return nil, fmt.Errorf("%w, received invalid response", common.ErrNoResponse)
}

// CancelMultipleOrders cancel incomplete orders in batches. Maximum 20 orders can be canceled at a time.
// Request parameters should be passed in the form of an array.
func (ok *Okx) CancelMultipleOrders(ctx context.Context, args []CancelOrderRequestParam) ([]OrderData, error) {
	for x := range args {
		arg := args[x]
		if arg.InstrumentID == "" {
			return nil, errMissingInstrumentID
		}
		if arg.OrderID == "" && arg.ClientOrderID == "" {
			return nil, errors.New("either order id or client id is required")
		}
	}
	var resp []OrderData
	err := ok.SendHTTPRequest(ctx, exchange.RestSpot, cancelMultipleOrdersEPL,
		http.MethodPost, "trade/cancel-batch-orders", args, &resp, true)
	if err != nil {
		if len(resp) == 0 {
			return nil, err
		}
		var errs error
		for x := range resp {
			if resp[x].SCode != "0" {
				errs = common.AppendError(errs, fmt.Errorf("error code:%s message: %v", resp[x].SCode, resp[x].SMessage))
			}
		}
		return nil, errs
	}
	return resp, nil
}

// AmendOrder an incomplete order.
func (ok *Okx) AmendOrder(ctx context.Context, arg *AmendOrderRequestParams) (*OrderData, error) {
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	if arg.ClientOrderID == "" && arg.OrderID == "" {
		return nil, errMissingClientOrderIDOrOrderID
	}
	if arg.NewQuantity < 0 && arg.NewPrice < 0 {
		return nil, errInvalidNewSizeOrPriceInformation
	}
	var resp *OrderData
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, amendOrderEPL, http.MethodPost, "trade/amend-order", arg, &resp, true)
}

// AmendMultipleOrders amend incomplete orders in batches. Maximum 20 orders can be amended at a time. Request parameters should be passed in the form of an array.
func (ok *Okx) AmendMultipleOrders(ctx context.Context, args []AmendOrderRequestParams) ([]OrderData, error) {
	for x := range args {
		if args[x].InstrumentID == "" {
			return nil, errMissingInstrumentID
		}
		if args[x].ClientOrderID == "" && args[x].OrderID == "" {
			return nil, errMissingClientOrderIDOrOrderID
		}
		if args[x].NewQuantity < 0 && args[x].NewPrice < 0 {
			return nil, errInvalidNewSizeOrPriceInformation
		}
	}
	var resp []OrderData
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, amendMultipleOrdersEPL, http.MethodPost, "trade/amend-batch-orders", &args, &resp, true)
}

// ClosePositions close all positions of an instrument via a market order.
func (ok *Okx) ClosePositions(ctx context.Context, arg *ClosePositionsRequestParams) (*ClosePositionResponse, error) {
	if arg == nil || *arg == (ClosePositionsRequestParams{}) {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	if arg.MarginMode != "" &&
		arg.MarginMode != TradeModeCross &&
		arg.MarginMode != TradeModeIsolated {
		return nil, errMissingMarginMode
	}
	var resp *ClosePositionResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, closePositionEPL, http.MethodPost, "trade/close-position", arg, &resp, true)
}

// GetOrderDetail retrieves order details given instrument id and order identification
func (ok *Okx) GetOrderDetail(ctx context.Context, arg *OrderDetailRequestParam) (*OrderDetail, error) {
	if arg == nil || *arg == (OrderDetailRequestParam{}) {
		return nil, common.ErrNilPointer
	}
	params := url.Values{}
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	params.Set("instId", arg.InstrumentID)
	switch {
	case arg.OrderID == "" && arg.ClientOrderID == "":
		return nil, errMissingClientOrderIDOrOrderID
	case arg.ClientOrderID == "":
		params.Set("ordId", arg.OrderID)
	default:
		params.Set("clOrdId", arg.ClientOrderID)
	}
	var resp *OrderDetail
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getOrderDetEPL, http.MethodGet, common.EncodeURLValues("trade/order", params), nil, &resp, true)
}

// GetOrderList retrieves all incomplete orders under the current account.
func (ok *Okx) GetOrderList(ctx context.Context, arg *OrderListRequestParams) ([]OrderDetail, error) {
	if arg == nil || *arg == (OrderListRequestParams{}) {
		return nil, common.ErrNilPointer
	}
	params := url.Values{}
	if arg.InstrumentType != "" {
		params.Set("instType", arg.InstrumentType)
	}
	if arg.InstrumentID != "" {
		params.Set("instId", arg.InstrumentID)
	}
	if arg.Underlying != "" {
		params.Set("uly", arg.Underlying)
	}
	if arg.OrderType != "" {
		params.Set("orderType", strings.ToLower(arg.OrderType))
	}
	if arg.State != "" {
		params.Set("state", arg.State)
	}
	if arg.Before != "" {
		params.Set("before", arg.Before)
	}
	if arg.After != "" {
		params.Set("after", arg.After)
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatInt(arg.Limit, 10))
	}
	var resp []OrderDetail
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getOrderListEPL, http.MethodGet, common.EncodeURLValues("trade/orders-pending", params), nil, &resp, true)
}

// Get7DayOrderHistory retrieves the completed order data for the last 7 days, and the incomplete orders that have been cancelled are only reserved for 2 hours.
func (ok *Okx) Get7DayOrderHistory(ctx context.Context, arg *OrderHistoryRequestParams) ([]OrderDetail, error) {
	return ok.getOrderHistory(ctx, arg, "trade/orders-history", getOrderHistory7DaysEPL)
}

// Get3MonthOrderHistory retrieves the completed order data for the last 7 days, and the incomplete orders that have been cancelled are only reserved for 2 hours.
func (ok *Okx) Get3MonthOrderHistory(ctx context.Context, arg *OrderHistoryRequestParams) ([]OrderDetail, error) {
	return ok.getOrderHistory(ctx, arg, "trade/orders-history-archive", getOrderHistory3MonthsEPL)
}

// getOrderHistory retrieves the order history of the past limited times
func (ok *Okx) getOrderHistory(ctx context.Context, arg *OrderHistoryRequestParams, route string, rateLimit request.EndpointLimit) ([]OrderDetail, error) {
	if arg == nil || *arg == (OrderHistoryRequestParams{}) {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentType == "" {
		return nil, errInstrumentTypeRequired
	}
	params := url.Values{}
	params.Set("instType", strings.ToUpper(arg.InstrumentType))
	if arg.InstrumentID != "" {
		params.Set("instId", arg.InstrumentID)
	}
	if arg.Underlying != "" {
		params.Set("uly", arg.Underlying)
	}
	if arg.OrderType != "" {
		params.Set("orderType", strings.ToLower(arg.OrderType))
	}
	if arg.State != "" {
		params.Set("state", arg.State)
	}
	if arg.Before != "" {
		params.Set("before", arg.Before)
	}
	if arg.After != "" {
		params.Set("after", arg.After)
	}
	if !arg.Start.IsZero() {
		params.Set("begin", strconv.FormatInt(arg.Start.UnixMilli(), 10))
	}
	if !arg.End.IsZero() {
		params.Set("end", strconv.FormatInt(arg.End.UnixMilli(), 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatInt(arg.Limit, 10))
	}
	if arg.Category != "" {
		params.Set("category", strings.ToLower(arg.Category))
	}
	var resp []OrderDetail
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, rateLimit, http.MethodGet, common.EncodeURLValues(route, params), nil, &resp, true)
}

// GetTransactionDetailsLast3Days retrieves recently-filled transaction details in the last 3 day.
func (ok *Okx) GetTransactionDetailsLast3Days(ctx context.Context, arg *TransactionDetailRequestParams) ([]TransactionDetail, error) {
	return ok.getTransactionDetails(ctx, arg, "trade/fills", getTransactionDetail3DaysEPL)
}

// GetTransactionDetailsLast3Months Retrieve recently-filled transaction details in the last 3 months.
func (ok *Okx) GetTransactionDetailsLast3Months(ctx context.Context, arg *TransactionDetailRequestParams) ([]TransactionDetail, error) {
	return ok.getTransactionDetails(ctx, arg, "trade/fills-history", getTransactionDetail3MonthsEPL)
}

// SetTransactionDetailIntervalFor2Years to apply for recently-filled transaction details in the past 2 years except for last 3 months.
// returns download link generation time
func (ok *Okx) SetTransactionDetailIntervalFor2Years(ctx context.Context, arg *FillArchiveParam) (time.Time, error) {
	if arg == nil || *arg == (FillArchiveParam{}) {
		return time.Time{}, common.ErrNilPointer
	}
	resp := &struct {
		Timestamp convert.ExchangeTime `json:"ts"`
	}{}
	return resp.Timestamp.Time(), ok.SendHTTPRequest(ctx, exchange.RestSpot, setTransactionDetail2YearIntervalEPL, http.MethodPost, "trade/fills-archive", arg, &resp, true)
}

// GetTransactionDetailsLast2Year retrieve recently-filled transaction details in the past 2 years except for last 3 months.
func (ok *Okx) GetTransactionDetailsLast2Year(ctx context.Context, year int64, quarter string) ([]ArchiveReference, error) {
	if year == 0 {
		return nil, errors.New("year is required")
	}
	if quarter == "" {
		return nil, errors.New("quarter is required; possible values are Q1, Q2, Q3, and Q4")
	}
	params := url.Values{}
	params.Set("year", strconv.FormatInt(year, 10))
	params.Set("quarter", quarter)
	var resp []ArchiveReference
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getTransactionDetailLast2YearsEPL, http.MethodGet, common.EncodeURLValues("trade/fills-archive", params), nil, &resp, true)
}

// GetTransactionDetails retrieves recently-filled transaction details.
func (ok *Okx) getTransactionDetails(ctx context.Context, arg *TransactionDetailRequestParams, route string, rateLimit request.EndpointLimit) ([]TransactionDetail, error) {
	if arg == nil || *arg == (TransactionDetailRequestParams{}) {
		return nil, common.ErrNilPointer
	}
	arg.InstrumentType = strings.ToUpper(arg.InstrumentType)
	if arg.InstrumentType == "" {
		return nil, errInstrumentTypeRequired
	}
	params := url.Values{}
	params.Set("instType", arg.InstrumentType)
	if arg.InstrumentID != "" {
		params.Set("instId", arg.InstrumentID)
	}
	if arg.Underlying != "" {
		params.Set("uly", arg.Underlying)
	}
	if !arg.Begin.IsZero() {
		params.Set("begin", strconv.FormatInt(arg.Begin.UnixMilli(), 10))
	}
	if !arg.End.IsZero() {
		params.Set("end", strconv.FormatInt(arg.End.UnixMilli(), 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatInt(arg.Limit, 10))
	}
	if arg.InstrumentID != "" {
		params.Set("instId", arg.InstrumentID)
	}
	if arg.After != "" {
		params.Set("after", arg.After)
	}
	if arg.Before != "" {
		params.Set("before", arg.Before)
	}
	var resp []TransactionDetail
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, rateLimit, http.MethodGet, common.EncodeURLValues(route, params), nil, &resp, true)
}

// PlaceAlgoOrder order includes trigger order, oco order, conditional order,iceberg order, twap order and trailing order.
func (ok *Okx) PlaceAlgoOrder(ctx context.Context, arg *AlgoOrderParams) (*AlgoOrder, error) {
	if arg == nil || *arg == (AlgoOrderParams{}) {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	arg.TradeMode = strings.ToLower(arg.TradeMode)
	if arg.TradeMode != TradeModeCross &&
		arg.TradeMode != TradeModeIsolated {
		return nil, errInvalidTradeModeValue
	}
	if arg.Side != order.Buy &&
		arg.Side != order.Sell {
		return nil, order.ErrSideIsInvalid
	}
	if arg.OrderType == "" {
		return nil, order.ErrTypeIsInvalid
	}
	if arg.Size <= 0 {
		return nil, errMissingNewSize
	}
	var resp *AlgoOrder
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, placeAlgoOrderEPL, http.MethodGet, "trade/order-algo", arg, &resp, true)
}

// PlaceStopOrder to place stop order
func (ok *Okx) PlaceStopOrder(ctx context.Context, arg *AlgoOrderParams) (*AlgoOrder, error) {
	if arg == nil || *arg == (AlgoOrderParams{}) {
		return nil, common.ErrNilPointer
	}
	if arg.OrderType != "conditional" {
		return nil, order.ErrTypeIsInvalid
	}
	if arg.TakeProfitTriggerPrice == 0 {
		return nil, errMissingTakeProfitTriggerPrice
	}
	if arg.TakeProfitTriggerPriceType == "" {
		return nil, errMissingTakeProfitOrderPrice
	}
	return ok.PlaceAlgoOrder(ctx, arg)
}

// PlaceTrailingStopOrder to place trailing stop order
func (ok *Okx) PlaceTrailingStopOrder(ctx context.Context, arg *AlgoOrderParams) (*AlgoOrder, error) {
	if arg == nil || *arg == (AlgoOrderParams{}) {
		return nil, common.ErrNilPointer
	}
	if arg.OrderType != "move_order_stop" {
		return nil, order.ErrTypeIsInvalid
	}
	if arg.CallbackRatio == 0 && arg.CallbackSpreadVariance == "" {
		return nil, errors.New("either \"callbackRatio\" or \"callbackSpread\" is allowed to be passed")
	}
	return ok.PlaceAlgoOrder(ctx, arg)
}

// PlaceIcebergOrder to place iceburg algo order
func (ok *Okx) PlaceIcebergOrder(ctx context.Context, arg *AlgoOrderParams) (*AlgoOrder, error) {
	if arg == nil || *arg == (AlgoOrderParams{}) {
		return nil, common.ErrNilPointer
	}
	if arg.OrderType != "iceberg" {
		return nil, order.ErrTypeIsInvalid
	}
	if arg.SizeLimit <= 0 {
		return nil, errMissingSizeLimit
	}
	if arg.PriceLimit <= 0 {
		return nil, errInvalidPriceLimit
	}
	return ok.PlaceAlgoOrder(ctx, arg)
}

// PlaceTWAPOrder to place TWAP algo orders
func (ok *Okx) PlaceTWAPOrder(ctx context.Context, arg *AlgoOrderParams) (*AlgoOrder, error) {
	if arg == nil || *arg == (AlgoOrderParams{}) {
		return nil, common.ErrNilPointer
	}
	if arg.OrderType != "twap" {
		return nil, order.ErrTypeIsInvalid
	}
	if arg.SizeLimit <= 0 {
		return nil, errMissingSizeLimit
	}
	if arg.PriceLimit <= 0 {
		return nil, errInvalidPriceLimit
	}
	if ok.GetIntervalEnum(arg.TimeInterval, true) == "" {
		return nil, errMissingIntervalValue
	}
	return ok.PlaceAlgoOrder(ctx, arg)
}

// TriggerAlgoOrder fetches algo trigger orders for SWAP market types.
func (ok *Okx) TriggerAlgoOrder(ctx context.Context, arg *AlgoOrderParams) (*AlgoOrder, error) {
	if arg == nil || *arg == (AlgoOrderParams{}) {
		return nil, common.ErrNilPointer
	}
	if arg.OrderType != "trigger" {
		return nil, order.ErrTypeIsInvalid
	}
	if arg.TriggerPrice <= 0 {
		return nil, errInvalidTriggerPrice
	}
	if arg.TriggerPriceType != "" &&
		arg.TriggerPriceType != "last" &&
		arg.TriggerPriceType != "index" &&
		arg.TriggerPriceType != "mark" {
		return nil, errors.New("only last, index and mark trigger price types are allowed")
	}
	return ok.PlaceAlgoOrder(ctx, arg)
}

// CancelAdvanceAlgoOrder Cancel unfilled algo orders
// A maximum of 10 orders can be canceled at a time.
// Request parameters should be passed in the form of an array.
func (ok *Okx) CancelAdvanceAlgoOrder(ctx context.Context, args []AlgoOrderCancelParams) ([]AlgoOrder, error) {
	if len(args) == 0 {
		return nil, common.ErrNilPointer
	}
	return ok.cancelAlgoOrder(ctx, args, "trade/cancel-advance-algos", cancelAdvanceAlgoOrderEPL)
}

// CancelAlgoOrder to cancel unfilled algo orders (not including Iceberg order, TWAP order, Trailing Stop order).
// A maximum of 10 orders can be canceled at a time.
// Request parameters should be passed in the form of an array.
func (ok *Okx) CancelAlgoOrder(ctx context.Context, args []AlgoOrderCancelParams) ([]AlgoOrder, error) {
	if len(args) == 0 {
		return nil, common.ErrNilPointer
	}
	return ok.cancelAlgoOrder(ctx, args, "trade/cancel-algos", cancelAlgoOrderEPL)
}

// cancelAlgoOrder to cancel unfilled algo orders.
func (ok *Okx) cancelAlgoOrder(ctx context.Context, args []AlgoOrderCancelParams, route string, rateLimit request.EndpointLimit) ([]AlgoOrder, error) {
	if len(args) == 0 {
		return nil, errors.New("no parameter")
	}
	for x := range args {
		arg := args[x]
		if arg.AlgoOrderID == "" {
			return nil, fmt.Errorf("%w, algoId is required", order.ErrOrderIDNotSet)
		} else if arg.InstrumentID == "" {
			return nil, errMissingInstrumentID
		}
	}
	var resp []AlgoOrder
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, rateLimit, http.MethodPost, route, &args, &resp, true)
}

// AmendAlgoOrder amend unfilled algo orders (Support stop order only, not including Move_order_stop order, Trigger order, Iceberg order, TWAP order, Trailing Stop order).
// Only applicable to Futures and Perpetual swap.
func (ok *Okx) AmendAlgoOrder(ctx context.Context, arg *AmendAlgoOrderParam) (*AmendAlgoResponse, error) {
	if arg == nil || *arg == (AmendAlgoOrderParam{}) {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	if arg.AlgoID == "" && arg.ClientSuppliedAlgoOrderID == "" {
		return nil, fmt.Errorf("%w either 'algoId' or 'algoClOrdId' is required", order.ErrOrderIDNotSet)
	}
	var resp *AmendAlgoResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, amendAlgoOrderEPL, http.MethodPost, "trade/amend-algos", arg, &resp, true)
}

// GetAlgoOrderDetail retrieves algo order details.
func (ok *Okx) GetAlgoOrderDetail(ctx context.Context, algoID, algoClientOrderID string) (*AlgoOrderDetail, error) {
	if algoID == "" && algoClientOrderID == "" {
		return nil, fmt.Errorf("%w either 'algoId' or 'algoClOrdId' is required", order.ErrOrderIDNotSet)
	}
	params := url.Values{}
	params.Set("algoId", algoID)
	params.Set("algoClOrdId", algoClientOrderID)
	var resp *AlgoOrderDetail
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getAlgoOrderDetailEPL, http.MethodGet, common.EncodeURLValues("trade/order-algo", params), nil, &resp, true)
}

// GetAlgoOrderList retrieves a list of untriggered Algo orders under the current account.
func (ok *Okx) GetAlgoOrderList(ctx context.Context, orderType, algoOrderID, clientOrderID, instrumentType, instrumentID string, after, before time.Time, limit int64) ([]AlgoOrderResponse, error) {
	orderType = strings.ToLower(orderType)
	if orderType == "" {
		return nil, errors.New("order type is required")
	}
	params := url.Values{}
	params.Set("ordType", orderType)
	if algoOrderID != "" {
		params.Set("algoId", algoOrderID)
	}
	if clientOrderID != "" {
		params.Set("clOrdId", clientOrderID)
	}
	instrumentType = strings.ToUpper(instrumentType)
	if instrumentType != "" {
		params.Set("instType", instrumentType)
	}
	if instrumentID != "" {
		params.Set("instId", instrumentID)
	}
	if !before.IsZero() && before.Before(time.Now()) {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if !after.IsZero() && after.Before(time.Now()) {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []AlgoOrderResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getAlgoOrderListEPL, http.MethodGet, common.EncodeURLValues("trade/orders-algo-pending", params), nil, &resp, true)
}

// GetAlgoOrderHistory load a list of all algo orders under the current account in the last 3 months.
func (ok *Okx) GetAlgoOrderHistory(ctx context.Context, orderType, state, algoOrderID, instrumentType, instrumentID string, after, before time.Time, limit int64) ([]AlgoOrderResponse, error) {
	if orderType == "" {
		return nil, errors.New("order type is required")
	}
	if algoOrderID == "" &&
		state != "effective" &&
		state != "order_failed" &&
		state != "canceled" {
		return nil, errMissingEitherAlgoIDOrState
	}
	params := url.Values{}
	params.Set("ordType", strings.ToLower(orderType))
	if algoOrderID != "" {
		params.Set("algoId", algoOrderID)
	} else {
		params.Set("state", state)
	}
	instrumentType = strings.ToUpper(instrumentType)
	if instrumentType != "" {
		params.Set("instType", instrumentType)
	}
	if instrumentID != "" {
		params.Set("instId", instrumentID)
	}
	if !before.IsZero() && before.Before(time.Now()) {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if !after.IsZero() && after.Before(time.Now()) {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []AlgoOrderResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getAlgoOrderHistoryEPL, http.MethodGet, common.EncodeURLValues("trade/orders-algo-history", params), nil, &resp, true)
}

// GetEasyConvertCurrencyList retrieve list of small convertibles and mainstream currencies. Only applicable to the crypto balance less than $10.
func (ok *Okx) GetEasyConvertCurrencyList(ctx context.Context) (*EasyConvertDetail, error) {
	var resp *EasyConvertDetail
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getEasyConvertCurrencyListEPL, http.MethodGet,
		"trade/easy-convert-currency-list", nil, &resp, true)
}

// PlaceEasyConvert converts small currencies to mainstream currencies. Only applicable to the crypto balance less than $10.
func (ok *Okx) PlaceEasyConvert(ctx context.Context, arg PlaceEasyConvertParam) ([]EasyConvertItem, error) {
	if len(arg.FromCurrency) == 0 {
		return nil, fmt.Errorf("%w, missing 'fromCcy'", errMissingRequiredParameter)
	}
	if arg.ToCurrency == "" {
		return nil, fmt.Errorf("%w, missing 'toCcy'", errMissingRequiredParameter)
	}
	var resp []EasyConvertItem
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, placeEasyConvertEPL, http.MethodPost, "trade/easy-convert", &arg, &resp, true)
}

// GetEasyConvertHistory retrieves the history and status of easy convert trades.
func (ok *Okx) GetEasyConvertHistory(ctx context.Context, after, before time.Time, limit int64) ([]EasyConvertItem, error) {
	params := url.Values{}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.Unix(), 10))
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.Unix(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []EasyConvertItem
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getEasyConvertHistoryEPL, http.MethodGet, "trade/easy-convert-history", nil, &resp, true)
}

// GetOneClickRepayCurrencyList retrieves list of debt currency data and repay currencies. Debt currencies include both cross and isolated debts.
// debt level "cross", and "isolated" are allowed
func (ok *Okx) GetOneClickRepayCurrencyList(ctx context.Context, debtType string) ([]CurrencyOneClickRepay, error) {
	params := url.Values{}
	if debtType != "" {
		params.Set("debtType", debtType)
	}
	var resp []CurrencyOneClickRepay
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, oneClickRepayCurrencyListEPL, http.MethodGet, common.EncodeURLValues("trade/one-click-repay-currency-list", params), nil, &resp, true)
}

// TradeOneClickRepay trade one-click repay to repay cross debts. Isolated debts are not applicable. The maximum repayment amount is based on the remaining available balance of funding and trading accounts.
func (ok *Okx) TradeOneClickRepay(ctx context.Context, arg TradeOneClickRepayParam) ([]CurrencyOneClickRepay, error) {
	if len(arg.DebtCurrency) == 0 {
		return nil, fmt.Errorf("%w, missing 'debtCcy'", errMissingRequiredParameter)
	}
	if arg.RepayCurrency == "" {
		return nil, fmt.Errorf("%w, missing 'repayCcy'", errMissingRequiredParameter)
	}
	var resp []CurrencyOneClickRepay
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, tradeOneClickRepayEPL, http.MethodPost, "trade/one-click-repay", &arg, &resp, true)
}

// GetOneClickRepayHistory get the history and status of one-click repay trades.
func (ok *Okx) GetOneClickRepayHistory(ctx context.Context, after, before time.Time, limit int64) ([]CurrencyOneClickRepay, error) {
	params := url.Values{}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.Unix(), 10))
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.Unix(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []CurrencyOneClickRepay
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getOneClickRepayHistoryEPL, http.MethodGet, common.EncodeURLValues("trade/one-click-repay-history", params), nil, &resp, true)
}

// MassCancelOrder cancel all the MMP pending orders of an instrument family.
// Only applicable to Option in Portfolio Margin mode, and MMP privilege is required.
func (ok *Okx) MassCancelOrder(ctx context.Context, instrumentType, instrumentFamily string) (*CancelMMPResponse, error) {
	if instrumentType == "" {
		return nil, errInstrumentTypeRequired
	}
	if instrumentFamily == "" {
		return nil, errInstrumentFamilyRequired
	}
	arg := &struct {
		InstrumentType   string `json:"instType,omitempty"`
		InstrumentFamily string `json:"instFamily,omitempty"`
	}{
		InstrumentType:   instrumentType,
		InstrumentFamily: instrumentFamily,
	}
	var resp *CancelMMPResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, tradeOneClickRepayEPL, http.MethodPost, "trade/mass-cancel", arg, &resp, true)
}

// CancelAllMMPOrdersAfterCountdown cancel all MMP pending orders after the countdown timeout.
// Only applicable to Option in Portfolio Margin mode, and MMP privilege is required.
func (ok *Okx) CancelAllMMPOrdersAfterCountdown(ctx context.Context, timeout int64, orderTag string) (*CancelMMPAfterCountdownResponse, error) {
	if (timeout != 0) && (timeout < 10 || timeout > 120) {
		return nil, fmt.Errorf("%w, Range of value can be 0, [10, 120]", errCountdownTimeoutRequired)
	}
	arg := &struct {
		TimeOut  int64  `json:"timeOut,string"`
		OrderTag string `json:"tag,omitempty"`
	}{
		TimeOut:  timeout,
		OrderTag: orderTag,
	}
	var resp *CancelMMPAfterCountdownResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, cancelAllAfterCountdownEPL, http.MethodPost, "trade/cancel-all-after", arg, &resp, true)
}

/*************************************** Block trading ********************************/

// GetCounterparties retrieves the list of counterparties that the user has permissions to trade with.
func (ok *Okx) GetCounterparties(ctx context.Context) ([]CounterpartiesResponse, error) {
	var resp []CounterpartiesResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getCounterpartiesEPL, http.MethodGet, "rfq/counterparties", nil, &resp, true)
}

// CreateRfq Creates a new Rfq
func (ok *Okx) CreateRfq(ctx context.Context, arg CreateRfqInput) (*RfqResponse, error) {
	if len(arg.CounterParties) == 0 {
		return nil, errInvalidCounterParties
	}
	if len(arg.Legs) == 0 {
		return nil, errMissingLegs
	}
	var resp *RfqResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, createRfqEPL, http.MethodPost, "rfq/create-rfq", &arg, &resp, true)
}

// CancelRfq Cancel an existing active Rfq that you has previously created.
func (ok *Okx) CancelRfq(ctx context.Context, rfqID, clientRfqID string) (*CancelRfqResponse, error) {
	if rfqID == "" && clientRfqID == "" {
		return nil, errMissingRfqIDAndClientRfqID
	}
	var resp *CancelRfqResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, cancelRfqEPL, http.MethodPost, "rfq/cancel-rfq", &CancelRfqRequestParam{
		RfqID:       rfqID,
		ClientRfqID: clientRfqID,
	}, &resp, true)
}

// CancelMultipleRfqs cancel multiple active Rfqs in a single batch. Maximum 100 Rfq orders can be canceled at a time.
func (ok *Okx) CancelMultipleRfqs(ctx context.Context, arg *CancelRfqRequestsParam) ([]CancelRfqResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if len(arg.RfqIDs) == 0 && len(arg.ClientRfqIDs) == 0 {
		return nil, errMissingRfqIDAndClientRfqID
	} else if len(arg.RfqIDs)+len(arg.ClientRfqIDs) > 100 {
		return nil, errMaxRfqOrdersToCancel
	}
	var resp []CancelRfqResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, cancelMultipleRfqEPL, http.MethodPost, "rfq/cancel-batch-rfqs", &arg, &resp, true)
}

// CancelAllRfqs cancels all active Rfqs.
func (ok *Okx) CancelAllRfqs(ctx context.Context) (time.Time, error) {
	resp := &TimestampResponse{}
	return resp.Timestamp.Time(), ok.SendHTTPRequest(ctx, exchange.RestSpot, cancelAllRfqsEPL, http.MethodPost, "rfq/cancel-all-rfqs", nil, &resp, true)
}

// ExecuteQuote executes a Quote. It is only used by the creator of the Rfq
func (ok *Okx) ExecuteQuote(ctx context.Context, rfqID, quoteID string) (*ExecuteQuoteResponse, error) {
	if rfqID == "" || quoteID == "" {
		return nil, errMissingRfqIDOrQuoteID
	}
	var resp *ExecuteQuoteResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, executeQuoteEPL, http.MethodPost, "rfq/execute-quote", &ExecuteQuoteParams{
		RfqID:   rfqID,
		QuoteID: quoteID,
	}, &resp, true)
}

// GetQuoteProducts retrieve the products which makers want to quote and receive RFQs for, and the corresponding price and size limit.
func (ok *Okx) GetQuoteProducts(ctx context.Context) ([]QuoteProduct, error) {
	var resp []QuoteProduct
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getQuoteProductsEPL, http.MethodGet, "rfq/maker-instrument-settings", nil, &resp, true)
}

// SetQuoteProducts customize the products which makers want to quote and receive Rfqs for, and the corresponding price and size limit.
func (ok *Okx) SetQuoteProducts(ctx context.Context, args []SetQuoteProductParam) (*SetQuoteProductsResult, error) {
	if len(args) == 0 {
		return nil, common.ErrNilPointer
	}
	for x := range args {
		args[x].InstrumentType = strings.ToUpper(args[x].InstrumentType)
		if args[x].InstrumentType != okxInstTypeSwap &&
			args[x].InstrumentType != okxInstTypeSpot &&
			args[x].InstrumentType != okxInstTypeFutures &&
			args[x].InstrumentType != okxInstTypeOption {
			return nil, fmt.Errorf("%w received %v", errInvalidInstrumentType, args[x].InstrumentType)
		}
		if len(args[x].Data) == 0 {
			return nil, errMissingMakerInstrumentSettings
		}
		for y := range args[x].Data {
			if (args[x].InstrumentType == okxInstTypeSwap ||
				args[x].InstrumentType == okxInstTypeFutures ||
				args[x].InstrumentType == okxInstTypeOption) && args[x].Data[y].Underlying == "" {
				return nil, fmt.Errorf("%w, for instrument type %s and %s", errInvalidUnderlying, args[x].InstrumentType, args[x].Data[x].Underlying)
			}
			if (args[x].InstrumentType == okxInstTypeSpot) && args[x].Data[x].InstrumentID == "" {
				return nil, fmt.Errorf("%w, for instrument type %s and %s", errMissingInstrumentID, args[x].InstrumentType, args[x].Data[x].InstrumentID)
			}
		}
	}
	var resp *SetQuoteProductsResult
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, setQuoteProductsEPL, http.MethodPost, "rfq/maker-instrument-settings", &args, &resp, true)
}

// ResetRFQMMPStatus reset the MMP status to be inactive.
func (ok *Okx) ResetRFQMMPStatus(ctx context.Context) (time.Time, error) {
	resp := &struct {
		Timestamp convert.ExchangeTime `json:"ts"`
	}{}
	return resp.Timestamp.Time(), ok.SendHTTPRequest(ctx, exchange.RestSpot, resetRFQMMPEPL, http.MethodPost, "rfq/mmp-reset", nil, resp, true)
}

// CreateQuote allows the user to Quote an Rfq that they are a counterparty to. The user MUST quote
// the entire Rfq and not part of the legs or part of the quantity. Partial quoting or partial fills are not allowed.
func (ok *Okx) CreateQuote(ctx context.Context, arg CreateQuoteParams) (*QuoteResponse, error) {
	arg.QuoteSide = strings.ToLower(arg.QuoteSide)
	switch {
	case arg.RfqID == "":
		return nil, errMissingRfqID
	case arg.QuoteSide != "buy" && arg.QuoteSide != "sell":
		return nil, order.ErrSideIsInvalid
	case len(arg.Legs) == 0:
		return nil, errMissingLegs
	}
	for x := range arg.Legs {
		switch {
		case arg.Legs[x].InstrumentID == "":
			return nil, errMissingInstrumentID
		case arg.Legs[x].SizeOfQuoteLeg <= 0:
			return nil, errMissingSizeOfQuote
		case arg.Legs[x].Price <= 0:
			return nil, errMossingLegsQuotePrice
		case arg.Legs[x].Side == order.UnknownSide:
			return nil, order.ErrSideIsInvalid
		}
	}
	var resp *QuoteResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, createQuoteEPL, http.MethodPost, "rfq/create-quote", &arg, &resp, true)
}

// CancelQuote cancels an existing active quote you have created in response to an Rfq.
// rfqCancelQuote = "rfq/cancel-quote"
func (ok *Okx) CancelQuote(ctx context.Context, quoteID, clientQuoteID string) (*CancelQuoteResponse, error) {
	if clientQuoteID == "" && quoteID == "" {
		return nil, errMissingQuoteIDOrClientQuoteID
	}
	var resp *CancelQuoteResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, cancelQuoteEPL, http.MethodPost, "rfq/cancel-quote", &CancelQuoteRequestParams{
		QuoteID:       quoteID,
		ClientQuoteID: clientQuoteID,
	}, &resp, true)
}

// CancelMultipleQuote cancel multiple active Quotes in a single batch. Maximum 100 quote orders can be canceled at a time.
func (ok *Okx) CancelMultipleQuote(ctx context.Context, arg CancelQuotesRequestParams) ([]CancelQuoteResponse, error) {
	if len(arg.QuoteIDs) == 0 && len(arg.ClientQuoteIDs) == 0 {
		return nil, errMissingEitherQuoteIDAOrClientQuoteIDs
	}
	var resp []CancelQuoteResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, cancelMultipleQuotesEPL, http.MethodPost, "rfq/cancel-batch-quotes", &arg, &resp, true)
}

// CancelAllQuotes cancels all active Quotes.
func (ok *Okx) CancelAllQuotes(ctx context.Context) (time.Time, error) {
	var resp *TimestampResponse
	err := ok.SendHTTPRequest(ctx, exchange.RestSpot, cancelAllQuotesEPL, http.MethodPost, "rfq/cancel-all-quotes", nil, &resp, true)
	if err != nil {
		return time.Time{}, err
	}
	if resp == nil {
		return time.Time{}, common.ErrNoResponse
	}
	return resp.Timestamp.Time(), nil
}

// GetRfqs retrieves details of Rfqs that the user is a counterparty to (either as the creator or the receiver of the Rfq).
func (ok *Okx) GetRfqs(ctx context.Context, arg *RfqRequestParams) ([]RfqResponse, error) {
	if arg == nil || *arg == (RfqRequestParams{}) {
		return nil, common.ErrNilPointer
	}
	params := url.Values{}
	if arg.RfqID != "" {
		params.Set("rfqId", arg.RfqID)
	}
	if arg.ClientRfqID != "" {
		params.Set("clRfqId", arg.ClientRfqID)
	}
	if arg.State != "" {
		params.Set("state", strings.ToLower(arg.State))
	}
	if arg.BeginningID != "" {
		params.Set("beginId", arg.BeginningID)
	}
	if arg.EndID != "" {
		params.Set("endId", arg.EndID)
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatInt(arg.Limit, 10))
	}
	var resp []RfqResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getRfqsEPL, http.MethodGet, common.EncodeURLValues("rfq/rfqs", params), nil, &resp, true)
}

// GetQuotes retrieves all Quotes that the user is a counterparty to (either as the creator or the receiver).
func (ok *Okx) GetQuotes(ctx context.Context, arg *QuoteRequestParams) ([]QuoteResponse, error) {
	if arg == nil || *arg == (QuoteRequestParams{}) {
		return nil, common.ErrNilPointer
	}
	params := url.Values{}
	if arg.RfqID != "" {
		params.Set("rfqId", arg.RfqID)
	}
	if arg.ClientRfqID != "" {
		params.Set("clRfqId", arg.ClientRfqID)
	}
	if arg.QuoteID != "" {
		params.Set("quoteId", arg.QuoteID)
	}
	if arg.ClientQuoteID != "" {
		params.Set("clQuoteId", arg.ClientQuoteID)
	}
	if arg.State != "" {
		params.Set("state", strings.ToLower(arg.State))
	}
	if arg.BeginID != "" {
		params.Set("beginId", arg.BeginID)
	}
	if arg.EndID != "" {
		params.Set("endId", arg.EndID)
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatInt(arg.Limit, 10))
	}
	var resp []QuoteResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getQuotesEPL, http.MethodGet, common.EncodeURLValues("rfq/quotes", params), nil, &resp, true)
}

// GetRfqTrades retrieves the executed trades that the user is a counterparty to (either as the creator or the receiver).
func (ok *Okx) GetRfqTrades(ctx context.Context, arg *RfqTradesRequestParams) ([]RfqTradeResponse, error) {
	if arg == nil || *arg == (RfqTradesRequestParams{}) {
		return nil, common.ErrNilPointer
	}
	params := url.Values{}
	if arg.RfqID != "" {
		params.Set("rfqId", arg.RfqID)
	}
	if arg.ClientRfqID != "" {
		params.Set("clRfqId", arg.ClientRfqID)
	}
	if arg.QuoteID != "" {
		params.Set("quoteId", arg.QuoteID)
	}
	if arg.ClientQuoteID != "" {
		params.Set("clQuoteId", arg.ClientQuoteID)
	}
	if arg.State != "" {
		params.Set("state", strings.ToLower(arg.State))
	}
	if arg.BlockTradeID != "" {
		params.Set("blockTdId", arg.BlockTradeID)
	}
	if arg.BeginID != "" {
		params.Set("beginId", arg.BeginID)
	}
	if arg.EndID != "" {
		params.Set("endId", arg.EndID)
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatInt(arg.Limit, 10))
	}
	var resp []RfqTradeResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getTradesEPL, http.MethodGet, common.EncodeURLValues("rfq/trades", params), nil, &resp, true)
}

// GetPublicRFQTrades retrieves the recent executed block trades.
func (ok *Okx) GetPublicRFQTrades(ctx context.Context, beginID, endID string, limit int64) ([]PublicTradesResponse, error) {
	params := url.Values{}
	if beginID != "" {
		params.Set("beginId", beginID)
	}
	if endID != "" {
		params.Set("endId", endID)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []PublicTradesResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getPublicTradesEPL, http.MethodGet, common.EncodeURLValues("rfq/public-trades", params), nil, &resp, false)
}

/*************************************** Funding Tradings ********************************/

// GetFundingCurrencies retrieve a list of all currencies.
func (ok *Okx) GetFundingCurrencies(ctx context.Context, ccy currency.Code) ([]CurrencyResponse, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	var resp []CurrencyResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getCurrenciesEPL, http.MethodGet, common.EncodeURLValues("asset/currencies", params), nil, &resp, true)
}

// GetBalance retrieves the funding account balances of all the assets and the amount that is available or on hold.
func (ok *Okx) GetBalance(ctx context.Context, ccy currency.Code) ([]AssetBalance, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	var resp []AssetBalance
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getBalanceEPL, http.MethodGet, common.EncodeURLValues("asset/balances", params), nil, &resp, true)
}

// GetNonTradableAssets retrieves non tradable assets.
func (ok *Okx) GetNonTradableAssets(ctx context.Context, ccy currency.Code) ([]NonTradableAsset, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	var resp []NonTradableAsset
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getNonTradableAssetsEPL, http.MethodGet, common.EncodeURLValues("asset/non-tradable-assets", params), nil, &resp, true)
}

// GetAccountAssetValuation view account asset valuation
func (ok *Okx) GetAccountAssetValuation(ctx context.Context, ccy currency.Code) ([]AccountAssetValuation, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.Upper().String())
	}
	var resp []AccountAssetValuation
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getAccountAssetValuationEPL, http.MethodGet, common.EncodeURLValues("asset/asset-valuation", params), nil, &resp, true)
}

// FundingTransfer transfer of funds between your funding account and trading account,
// and from the master account to sub-accounts.
func (ok *Okx) FundingTransfer(ctx context.Context, arg *FundingTransferRequestInput) ([]FundingTransferResponse, error) {
	if arg == nil || *arg == (FundingTransferRequestInput{}) {
		return nil, common.ErrNilPointer
	}
	if arg.Amount <= 0 {
		return nil, fmt.Errorf("%w, funding amount must be greater than 0", order.ErrAmountBelowMin)
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.FundingSourceAddress != "6" && arg.FundingSourceAddress != "18" {
		return nil, fmt.Errorf("%w sending address is required", errAddressRequired)
	}
	if arg.FundingReceipentAddress == "" {
		return nil, fmt.Errorf("%w receipent address is required", errAddressRequired)
	}
	var resp []FundingTransferResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, fundsTransferEPL, http.MethodPost, "asset/transfer", arg, &resp, true)
}

// GetFundsTransferState get funding rate response.
func (ok *Okx) GetFundsTransferState(ctx context.Context, transferID, clientID string, transferType int64) ([]TransferFundRateResponse, error) {
	if transferID == "" && clientID == "" {
		return nil, fmt.Errorf("%w, 'transfer id' or 'client id' is required", order.ErrOrderIDNotSet)
	}
	params := url.Values{}
	if transferID != "" {
		params.Set("transId", transferID)
	}
	if clientID != "" {
		params.Set("clientId", clientID)
	}
	if transferType > 0 && transferType <= 4 {
		params.Set("type", strconv.FormatInt(transferType, 10))
	}
	var resp []TransferFundRateResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getFundsTransferStateEPL, http.MethodGet, common.EncodeURLValues("asset/transfer-state", params), nil, &resp, true)
}

// GetAssetBillsDetails Query the billing record, you can get the latest 1 month historical data
func (ok *Okx) GetAssetBillsDetails(ctx context.Context, ccy currency.Code, clientID string, after, before time.Time, billType, limit int64) ([]AssetBillDetail, error) {
	params := url.Values{}
	billTypeMap := map[int64]bool{1: true, 2: true, 13: true, 20: true, 21: true, 28: true, 47: true, 48: true, 49: true, 50: true, 51: true, 52: true, 53: true, 54: true, 61: true, 68: true, 69: true, 72: true, 73: true, 74: true, 75: true, 76: true, 77: true, 78: true, 79: true, 80: true, 81: true, 82: true, 83: true, 84: true, 85: true, 86: true, 87: true, 88: true, 89: true, 90: true, 91: true, 92: true, 93: true, 94: true, 95: true, 96: true, 97: true, 98: true, 99: true, 102: true, 103: true, 104: true, 105: true, 106: true, 107: true, 108: true, 109: true, 110: true, 111: true, 112: true, 113: true, 114: true, 115: true, 116: true, 117: true, 118: true, 119: true, 120: true, 121: true, 122: true, 123: true, 124: true, 125: true, 126: true, 127: true, 128: true, 129: true, 130: true, 131: true, 132: true, 133: true, 134: true, 135: true, 136: true, 137: true, 138: true, 139: true, 141: true, 142: true, 143: true, 144: true, 145: true, 146: true, 147: true, 150: true, 151: true, 152: true, 153: true, 154: true, 155: true, 156: true, 157: true, 160: true, 161: true, 162: true, 163: true, 169: true, 170: true, 171: true, 172: true, 173: true, 174: true, 175: true, 176: true, 177: true, 178: true, 179: true, 180: true, 181: true, 182: true, 183: true, 184: true, 185: true, 186: true, 187: true, 188: true, 189: true, 193: true, 194: true, 195: true, 196: true, 197: true, 198: true, 199: true, 200: true, 211: true}
	if _, okay := billTypeMap[billType]; okay {
		params.Set("type", strconv.FormatInt(billType, 10))
	}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if clientID != "" {
		params.Set("clientId", clientID)
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []AssetBillDetail
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, assetBillsDetailsEPL, http.MethodGet, common.EncodeURLValues("asset/bills", params), nil, &resp, true)
}

// GetLightningDeposits users can create up to 10 thousand different invoices within 24 hours.
// this method fetches list of lightning deposits filtered by a currency and amount.
func (ok *Okx) GetLightningDeposits(ctx context.Context, ccy currency.Code, amount float64, to int64) ([]LightningDepositItem, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("ccy", ccy.String())
	if amount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	params.Set("amt", strconv.FormatFloat(amount, 'f', 0, 64))
	if to == 6 || to == 18 {
		params.Set("to", strconv.FormatInt(to, 10))
	}
	var resp []LightningDepositItem
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, lightningDepositsEPL, http.MethodGet, common.EncodeURLValues("asset/deposit-lightning", params), nil, &resp, true)
}

// GetCurrencyDepositAddress returns the deposit address and related information for the provided currency information.
func (ok *Okx) GetCurrencyDepositAddress(ctx context.Context, ccy currency.Code) ([]CurrencyDepositResponseItem, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("ccy", ccy.String())
	var resp []CurrencyDepositResponseItem
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getDepositAddressEPL, http.MethodGet, common.EncodeURLValues("asset/deposit-address", params), nil, &resp, true)
}

// GetCurrencyDepositHistory retrieves deposit records and withdrawal status information depending on the currency, timestamp, and chronological order.
func (ok *Okx) GetCurrencyDepositHistory(ctx context.Context, ccy currency.Code, depositID, transactionID string, after, before time.Time, state, limit int64) ([]DepositHistoryResponseItem, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if depositID != "" {
		params.Set("depId", depositID)
	}
	if transactionID != "" {
		params.Set("txId", transactionID)
	}
	params.Set("state", strconv.FormatInt(state, 10))
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []DepositHistoryResponseItem
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getDepositHistoryEPL, http.MethodGet, common.EncodeURLValues("asset/deposit-history", params), nil, &resp, true)
}

// Withdrawal to perform a withdrawal action. Sub-account does not support withdrawal.
func (ok *Okx) Withdrawal(ctx context.Context, input *WithdrawalInput) (*WithdrawalResponse, error) {
	if input == nil {
		return nil, common.ErrNilPointer
	}
	switch {
	case input.Currency.IsEmpty():
		return nil, currency.ErrCurrencyCodeEmpty
	case input.Amount <= 0:
		return nil, fmt.Errorf("%w, withdrawal amount required", order.ErrAmountBelowMin)
	case input.WithdrawalDestination == "":
		return nil, fmt.Errorf("%w, withdrawal destination required", errAddressRequired)
	case input.ToAddress == "":
		return nil, errors.New("missing verified digital currency address \"toAddr\" information")
	}
	var resp *WithdrawalResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, withdrawalEPL, http.MethodPost, "asset/withdrawal", &input, &resp, true)
}

/*
 This API function service is only open to some users. If you need this function service, please send an email to `liz.jensen@okg.com` to apply
*/

// LightningWithdrawal to withdraw a currency from an invoice.
func (ok *Okx) LightningWithdrawal(ctx context.Context, arg LightningWithdrawalRequestInput) (*LightningWithdrawalResponse, error) {
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	} else if arg.Invoice == "" {
		return nil, errors.New("missing invoice text")
	}
	var resp *LightningWithdrawalResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, lightningWithdrawalsEPL, http.MethodPost, "asset/withdrawal-lightning", &arg, &resp, true)
}

// CancelWithdrawal You can cancel normal withdrawal, but can not cancel the withdrawal on Lightning.
func (ok *Okx) CancelWithdrawal(ctx context.Context, withdrawalID string) (string, error) {
	if withdrawalID == "" {
		return "", errMissingValidWithdrawalID
	}
	type withdrawData struct {
		WithdrawalID string `json:"wdId"`
	}
	request := &withdrawData{
		WithdrawalID: withdrawalID,
	}
	var response withdrawData
	return response.WithdrawalID, ok.SendHTTPRequest(ctx, exchange.RestSpot, cancelWithdrawalEPL, http.MethodPost, "asset/cancel-withdrawal", request, &response, true)
}

// GetWithdrawalHistory retrieves the withdrawal records according to the currency, withdrawal status, and time range in reverse chronological order.
// The 100 most recent records are returned by default.
func (ok *Okx) GetWithdrawalHistory(ctx context.Context, ccy currency.Code, withdrawalID, clientID, transactionID, state string, after, before time.Time, limit int64) ([]WithdrawalHistoryResponse, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if withdrawalID != "" {
		params.Set("wdId", withdrawalID)
	}
	if clientID != "" {
		params.Set("clientId", clientID)
	}
	if transactionID != "" {
		params.Set("txId", transactionID)
	}
	if state != "" {
		params.Set("state", state)
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []WithdrawalHistoryResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getWithdrawalHistoryEPL, http.MethodGet, common.EncodeURLValues("asset/withdrawal-history", params), nil, &resp, true)
}

// GetDepositWithdrawalStatus retrieve deposit's and withdrawal's detailed status and estimated complete time.
func (ok *Okx) GetDepositWithdrawalStatus(ctx context.Context, ccy currency.Code, withdrawalID, transactionID, addressTo, chain string) ([]DepositWithdrawStatus, error) {
	if withdrawalID == "" && transactionID == "" {
		return nil, fmt.Errorf("%w, either withdrawal id or transaction id is required", order.ErrOrderIDNotSet)
	}
	if withdrawalID == "" {
		return nil, errMissingValidWithdrawalID
	}
	params := url.Values{}
	params.Set("wdId", withdrawalID)
	if transactionID != "" {
		params.Set("txId", transactionID)
	}
	if withdrawalID == "" && ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	} else if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if withdrawalID == "" && addressTo == "" {
		return nil, fmt.Errorf("%w, 'addressTo' is empty", errAddressRequired)
	} else if !ccy.IsEmpty() {
		params.Set("to", addressTo)
	}
	if withdrawalID == "" && chain == "" {
		return nil, common.ErrNoResults
	} else if !ccy.IsEmpty() {
		params.Set("chain", chain)
	}
	var resp []DepositWithdrawStatus
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getDepositWithdrawalStatusEPL, http.MethodGet, common.EncodeURLValues("asset/deposit-withdraw-status", params), nil, &resp, true)
}

// SmallAssetsConvert Convert small assets in funding account to OKB. Only one convert is allowed within 24 hours.
func (ok *Okx) SmallAssetsConvert(ctx context.Context, currency []string) (*SmallAssetConvertResponse, error) {
	input := map[string][]string{"ccy": currency}
	var resp *SmallAssetConvertResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, smallAssetsConvertEPL, http.MethodPost, "asset/convert-dust-assets", input, &resp, true)
}

// GetPublicExchangeList retrieves exchanges
func (ok *Okx) GetPublicExchangeList(ctx context.Context) ([]ExchangeInfo, error) {
	var resp []ExchangeInfo
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getPublicExchangeListEPL, http.MethodGet, "asset/exchange-list", nil, &resp, false)
}

// GetSavingBalance returns saving balance, and only assets in the funding account can be used for saving.
func (ok *Okx) GetSavingBalance(ctx context.Context, ccy currency.Code) ([]SavingBalanceResponse, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	var resp []SavingBalanceResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getSavingBalanceEPL, http.MethodGet, common.EncodeURLValues("finance/savings/balance", params), nil, &resp, true)
}

// SavingsPurchaseOrRedemption creates a purchase or redemption instance
func (ok *Okx) SavingsPurchaseOrRedemption(ctx context.Context, arg *SavingsPurchaseRedemptionInput) (*SavingsPurchaseRedemptionResponse, error) {
	if arg == nil || *arg == (SavingsPurchaseRedemptionInput{}) {
		return nil, common.ErrNilPointer
	}
	arg.ActionType = strings.ToLower(arg.ActionType)
	switch {
	case arg.Currency.IsEmpty():
		return nil, currency.ErrCurrencyCodeEmpty
	case arg.Amount <= 0:
		return nil, errUnacceptableAmount
	case arg.ActionType != "purchase" && arg.ActionType != "redempt":
		return nil, fmt.Errorf("%w, side has to be either \"redempt\" or \"purchase\"", order.ErrSideIsInvalid)
	case arg.ActionType == "purchase" && (arg.Rate < 0.01 || arg.Rate > 3.65):
		return nil, errors.New("the rate value range is between 1% (0.01) and 365% (3.65)")
	}
	var resp *SavingsPurchaseRedemptionResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, savingsPurchaseRedemptionEPL, http.MethodPost, "finance/savings/purchase-redempt", &arg, &resp, true)
}

// GetLendingHistory lending history
func (ok *Okx) GetLendingHistory(ctx context.Context, ccy currency.Code, before, after time.Time, limit int64) ([]LendingHistory, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []LendingHistory
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getLendingHistoryEPL, http.MethodGet, common.EncodeURLValues("finance/savings/lending-history", params), nil, &resp, true)
}

// SetLendingRate sets an assets lending rate
func (ok *Okx) SetLendingRate(ctx context.Context, arg LendingRate) (*LendingRate, error) {
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	} else if arg.Rate < 0.01 || arg.Rate > 3.65 {
		return nil, fmt.Errorf("%w, rate value range is between 1 percent (0.01) and 365 percent (3.65)", errLendingRateRequired)
	}
	var resp *LendingRate
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, setLendingRateEPL, http.MethodPost, "finance/savings/set-lending-rate", &arg, &resp, true)
}

// GetPublicBorrowInfo returns the public borrow info.
func (ok *Okx) GetPublicBorrowInfo(ctx context.Context, ccy currency.Code) ([]PublicBorrowInfo, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	var resp []PublicBorrowInfo
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getPublicBorrowInfoEPL, http.MethodGet, common.EncodeURLValues("finance/savings/lending-rate-summary", params), nil, &resp, false)
}

// GetPublicBorrowHistory return list of publix borrow history.
func (ok *Okx) GetPublicBorrowHistory(ctx context.Context, ccy currency.Code, before, after time.Time, limit int64) ([]PublicBorrowHistory, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []PublicBorrowHistory
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getPublicBorrowHistoryEPL, http.MethodGet, common.EncodeURLValues("finance/savings/lending-rate-history", params), nil, &resp, false)
}

/***********************************Convert Endpoints | Authenticated s*****************************************/

// GetConvertCurrencies retrieves the currency conversion information.
func (ok *Okx) GetConvertCurrencies(ctx context.Context) ([]ConvertCurrency, error) {
	var resp []ConvertCurrency
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getConvertCurrenciesEPL, http.MethodGet, "asset/convert/currencies", nil, &resp, true)
}

// GetConvertCurrencyPair retrieves the currency conversion response detail given the 'currency from' and 'currency to'
func (ok *Okx) GetConvertCurrencyPair(ctx context.Context, fromCcy, toCcy currency.Code) (*ConvertCurrencyPair, error) {
	if fromCcy.IsEmpty() {
		return nil, fmt.Errorf("%w, source currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if toCcy.IsEmpty() {
		return nil, fmt.Errorf("%w, target currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	params.Set("fromCcy", fromCcy.String())
	params.Set("toCcy", toCcy.String())
	var resp *ConvertCurrencyPair
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getConvertCurrencyPairEPL, http.MethodGet, common.EncodeURLValues("asset/convert/currency-pair", params), nil, &resp, true)
}

// EstimateQuote retrieves quote estimation detail result given the base and quote currency.
func (ok *Okx) EstimateQuote(ctx context.Context, arg *EstimateQuoteRequestInput) (*EstimateQuoteResponse, error) {
	if arg == nil || *arg == (EstimateQuoteRequestInput{}) {
		return nil, common.ErrNilPointer
	}
	if arg.BaseCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w, base currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if arg.QuoteCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w, quote currency is required", currency.ErrCurrencyCodeEmpty)
	}
	arg.Side = strings.ToLower(arg.Side)
	if arg.Side != order.Buy.Lower() && arg.Side != order.Sell.Lower() {
		return nil, order.ErrSideIsInvalid
	}
	if arg.RfqAmount <= 0 {
		return nil, fmt.Errorf("%w, rfq amount required", order.ErrAmountBelowMin)
	}
	if arg.RfqSzCurrency == "" {
		return nil, fmt.Errorf("%w, missing rfq currency", currency.ErrCurrencyCodeEmpty)
	}
	var resp *EstimateQuoteResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, estimateQuoteEPL, http.MethodPost, "asset/convert/estimate-quote", arg, &resp, true)
}

// ConvertTrade converts a base currency to quote currency.
func (ok *Okx) ConvertTrade(ctx context.Context, arg *ConvertTradeInput) (*ConvertTradeResponse, error) {
	if arg == nil || *arg == (ConvertTradeInput{}) {
		return nil, common.ErrNilPointer
	}
	if arg.BaseCurrency == "" {
		return nil, fmt.Errorf("%w, base currency required", currency.ErrCurrencyCodeEmpty)
	}
	if arg.QuoteCurrency == "" {
		return nil, fmt.Errorf("%w, quote currency required", currency.ErrCurrencyCodeEmpty)
	}
	arg.Side = strings.ToLower(arg.Side)
	if arg.Side != order.Buy.Lower() &&
		arg.Side != order.Sell.Lower() {
		return nil, order.ErrSideIsInvalid
	}
	if arg.Size <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	if arg.SizeCurrency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.QuoteID == "" {
		return nil, fmt.Errorf("%w, quote id required", order.ErrOrderIDNotSet)
	}
	var resp *ConvertTradeResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, convertTradeEPL, http.MethodPost, "asset/convert/trade", &arg, &resp, true)
}

// GetConvertHistory gets the recent history.
func (ok *Okx) GetConvertHistory(ctx context.Context, before, after time.Time, limit int64, tag string) ([]ConvertHistory, error) {
	params := url.Values{}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if tag != "" {
		params.Set("tag", tag)
	}
	var resp []ConvertHistory
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getConvertHistoryEPL, http.MethodGet, common.EncodeURLValues("asset/convert/history", params), nil, &resp, true)
}

/********************************** Account endpoints ***************************************************/

// GetAccountInstruments retrieve available instruments info of current account.
func (ok *Okx) GetAccountInstruments(ctx context.Context, instrumentType, underlying, instrumentFamily, instrumentID string) ([]AccountInstrument, error) {
	if instrumentType == "" {
		return nil, errInstrumentTypeRequired
	}
	params := url.Values{}
	switch instrumentType {
	case "MARGIN", "SWAP", "FUTURES":
		if underlying == "" {
			return nil, fmt.Errorf("%w, underlying is required", errInvalidUnderlying)
		}
		params.Set("uly", underlying)
	case "OPTION":
		if underlying == "" && instrumentFamily == "" {
			return nil, errInstrumentFamilyOrUnderlyingRequired
		}
		if underlying != "" {
			params.Set("uly", underlying)
		}
		if instrumentFamily != "" {
			params.Set("instFamily", instrumentFamily)
		}
	}
	params.Set("instType", instrumentType)
	if instrumentID != "" {
		params.Set("instId", instrumentID)
	}
	var resp []AccountInstrument
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getAccountInstrumentsEPL, http.MethodGet, common.EncodeURLValues("account/instruments", params), nil, &resp, true)
}

// AccountBalance retrieves a list of assets (with non-zero balance), remaining balance, and available amount in the trading account.
// Interest-free quota and discount rates are public data and not displayed on the account interface.
func (ok *Okx) AccountBalance(ctx context.Context, ccy currency.Code) ([]Account, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	var resp []Account
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getAccountBalanceEPL, http.MethodGet, common.EncodeURLValues("account/balance", params), nil, &resp, true)
}

// GetPositions retrieves information on your positions. When the account is in net mode, net positions will be displayed, and when the account is in long/short mode, long or short positions will be displayed.
func (ok *Okx) GetPositions(ctx context.Context, instrumentType, instrumentID, positionID string) ([]AccountPosition, error) {
	params := url.Values{}
	if instrumentType != "" {
		instrumentType = strings.ToUpper(instrumentType)
		params.Set("instType", instrumentType)
	}
	if instrumentID != "" {
		params.Set("instId", instrumentID)
	}
	if positionID != "" {
		params.Set("posId", positionID)
	}
	var resp []AccountPosition
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getPositionsEPL, http.MethodGet, common.EncodeURLValues("account/positions", params), nil, &resp, true)
}

// GetPositionsHistory retrieves the updated position data for the last 3 months.
func (ok *Okx) GetPositionsHistory(ctx context.Context, instrumentType, instrumentID, marginMode string, closePositionType, limit int64, after, before time.Time) ([]AccountPositionHistory, error) {
	params := url.Values{}
	instrumentType = strings.ToUpper(instrumentType)
	if instrumentType != "" {
		params.Set("instType", instrumentType)
	}
	if instrumentID != "" {
		params.Set("instId", instrumentID)
	}
	if marginMode != "" {
		params.Set("mgnMode", marginMode) // margin mode 'cross' and 'isolated' are allowed
	}
	// The type of closing position
	// 1：Close position partially;2：Close all;3：Liquidation;4：Partial liquidation; 5：ADL;
	// It is the latest type if there are several types for the same position.
	if closePositionType > 0 {
		params.Set("type", strconv.FormatInt(closePositionType, 10))
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []AccountPositionHistory
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getPositionsHistoryEPL, http.MethodGet, common.EncodeURLValues("account/positions-history", params), nil, &resp, true)
}

// GetAccountAndPositionRisk  get account and position risks.
func (ok *Okx) GetAccountAndPositionRisk(ctx context.Context, instrumentType string) ([]AccountAndPositionRisk, error) {
	params := url.Values{}
	if instrumentType != "" {
		params.Set("instType", strings.ToUpper(instrumentType))
	}
	var resp []AccountAndPositionRisk
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getAccountAndPositionRiskEPL, http.MethodGet, common.EncodeURLValues("account/account-position-risk", params), nil, &resp, true)
}

// GetBillsDetailLast7Days The bill refers to all transaction records that result in changing the balance of an account. Pagination is supported, and the response is sorted with the most recent first. This endpoint can retrieve data from the last 7 days.
func (ok *Okx) GetBillsDetailLast7Days(ctx context.Context, arg *BillsDetailQueryParameter) ([]BillsDetailResponse, error) {
	return ok.GetBillsDetail(ctx, arg, "account/bills", getBillsDetailsEPL)
}

// GetBillsDetail3Months retrieves the account’s bills.
// The bill refers to all transaction records that result in changing the balance of an account.
// Pagination is supported, and the response is sorted with most recent first.
// This endpoint can retrieve data from the last 3 months.
func (ok *Okx) GetBillsDetail3Months(ctx context.Context, arg *BillsDetailQueryParameter) ([]BillsDetailResponse, error) {
	return ok.GetBillsDetail(ctx, arg, "account/bills-archive", getBillsDetailArchiveEPL)
}

// GetBillsDetail retrieves the bills of the account.
func (ok *Okx) GetBillsDetail(ctx context.Context, arg *BillsDetailQueryParameter, route string, epl request.EndpointLimit) ([]BillsDetailResponse, error) {
	if arg == nil || *arg == (BillsDetailQueryParameter{}) {
		return nil, common.ErrNilPointer
	}
	params := url.Values{}
	if arg.InstrumentType != "" {
		params.Set("instType", strings.ToUpper(arg.InstrumentType))
	}
	if !arg.Currency.IsEmpty() {
		params.Set("ccy", arg.Currency.Upper().String())
	}
	if arg.MarginMode != "" {
		params.Set("mgnMode", arg.MarginMode)
	}
	if arg.ContractType != "" {
		params.Set("ctType", arg.ContractType)
	}
	if arg.BillType >= 1 && arg.BillType <= 13 {
		params.Set("type", strconv.Itoa(int(arg.BillType)))
	}
	if arg.BillSubType != 0 {
		params.Set("subType", strconv.FormatInt(arg.BillSubType, 10))
	}
	if arg.After != "" {
		params.Set("after", arg.After)
	}
	if arg.Before != "" {
		params.Set("before", arg.Before)
	}
	if !arg.BeginTime.IsZero() {
		params.Set("begin", strconv.FormatInt(arg.BeginTime.UnixMilli(), 10))
	}
	if !arg.EndTime.IsZero() {
		params.Set("end", strconv.FormatInt(arg.EndTime.UnixMilli(), 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatInt(arg.Limit, 10))
	}
	var resp []BillsDetailResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, epl, http.MethodGet, common.EncodeURLValues(route, params), nil, &resp, true)
}

// GetAccountConfiguration retrieves current account configuration.
func (ok *Okx) GetAccountConfiguration(ctx context.Context) ([]AccountConfigurationResponse, error) {
	var resp []AccountConfigurationResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getAccountConfigurationEPL, http.MethodGet, "account/config", nil, &resp, true)
}

// SetPositionMode FUTURES and SWAP support both long/short mode and net mode. In net mode, users can only have positions in one direction; In long/short mode, users can hold positions in long and short directions.
func (ok *Okx) SetPositionMode(ctx context.Context, positionMode string) (*PositionMode, error) {
	if positionMode != "long_short_mode" && positionMode != "net_mode" {
		return nil, errInvalidPositionMode
	}
	input := &PositionMode{
		PositionMode: positionMode,
	}
	var resp *PositionMode
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, setPositionModeEPL, http.MethodPost, "account/set-position-mode", input, &resp, true)
}

// SetLeverageRate sets a leverage setting for instrument id.
func (ok *Okx) SetLeverageRate(ctx context.Context, arg SetLeverageInput) (*SetLeverageResponse, error) {
	if arg.InstrumentID == "" && arg.Currency.IsEmpty() {
		return nil, errEitherInstIDOrCcyIsRequired
	}
	arg.PositionSide = strings.ToLower(arg.PositionSide)
	var resp *SetLeverageResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, setLeverageEPL, http.MethodPost, "account/set-leverage", &arg, &resp, true)
}

// GetMaximumBuySellAmountOROpenAmount retrieves the maximum buy or sell amount for a specific instrument id
func (ok *Okx) GetMaximumBuySellAmountOROpenAmount(ctx context.Context, ccy currency.Code, instrumentID, tradeMode, leverage string, price float64) ([]MaximumBuyAndSell, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	if tradeMode != TradeModeCross && tradeMode != TradeModeIsolated && tradeMode != TradeModeCash {
		return nil, errInvalidTradeModeValue
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	params.Set("tdMode", tradeMode)
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if price > 0 {
		params.Set("px", strconv.FormatFloat(price, 'f', -1, 64))
	}
	if leverage != "" {
		params.Set("leverage", leverage)
	}
	var resp []MaximumBuyAndSell
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getMaximumBuyOrSellAmountEPL, http.MethodGet, common.EncodeURLValues("account/max-size", params), nil, &resp, true)
}

// GetMaximumAvailableTradableAmount retrieves the maximum tradable amount for specific instrument id, and/or currency
func (ok *Okx) GetMaximumAvailableTradableAmount(ctx context.Context, ccy currency.Code, instrumentID, tradeMode string, reduceOnly bool, price float64) ([]MaximumTradableAmount, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	if tradeMode != TradeModeIsolated &&
		tradeMode != TradeModeCross &&
		tradeMode != TradeModeCash {
		return nil, errInvalidTradeModeValue
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if reduceOnly {
		params.Set("reduceOnly", "true")
	}
	params.Set("tdMode", tradeMode)
	params.Set("px", strconv.FormatFloat(price, 'f', 0, 64))
	var resp []MaximumTradableAmount
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getMaximumAvailableTradableAmountEPL, http.MethodGet, common.EncodeURLValues("account/max-avail-size", params), nil, &resp, true)
}

// IncreaseDecreaseMargin Increase or decrease the margin of the isolated position. Margin reduction may result in the change of the actual leverage.
func (ok *Okx) IncreaseDecreaseMargin(ctx context.Context, arg *IncreaseDecreaseMarginInput) (*IncreaseDecreaseMargin, error) {
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	if arg.PositionSide != positionSideLong &&
		arg.PositionSide != positionSideShort &&
		arg.PositionSide != positionSideNet {
		return nil, fmt.Errorf("%w, position side is required", order.ErrSideIsInvalid)
	}
	if arg.Type != "add" && arg.Type != "reduce" {
		return nil, errors.New("missing valid 'type', 'add': add margin 'reduce': reduce margin are allowed")
	}
	if arg.Amount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	var resp *IncreaseDecreaseMargin
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, increaseOrDecreaseMarginEPL, http.MethodPost, "account/position/margin-balance", &arg, &resp, true)
}

// GetLeverageRate retrieves leverage data for different instrument id or margin mode.
func (ok *Okx) GetLeverageRate(ctx context.Context, instrumentID, marginMode string) ([]LeverageResponse, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	if marginMode != TradeModeCross && marginMode != TradeModeIsolated {
		return nil, errMissingMarginMode
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	params.Set("mgnMode", marginMode)
	var resp []LeverageResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getLeverageEPL, http.MethodGet, common.EncodeURLValues("account/leverage-info", params), nil, &resp, true)
}

// GetLeverateEstimatedInfo retrieves leverage estimated information.
// Instrument type: possible values are MARGIN, SWAP, FUTURES
func (ok *Okx) GetLeverateEstimatedInfo(ctx context.Context, instrumentType, marginMode, leverage, positionSide, instrumentID string, ccy currency.Code) ([]LeverageEstimatedInfo, error) {
	if instrumentType == "" {
		return nil, errInstrumentTypeRequired
	}
	if marginMode == "" ||
		(marginMode != TradeModeCross &&
			marginMode != TradeModeIsolated) {
		return nil, errMissingMarginMode
	}
	if leverage == "" {
		return nil, errRequiredParameterMissingLeverage
	}
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("instType", instrumentType)
	params.Set("lever", leverage)
	params.Set("mgnMode", marginMode)
	params.Set("ccy", ccy.String())
	if instrumentID != "" {
		params.Set("instId", instrumentID)
	}
	if positionSide != "" {
		params.Set("posSide", positionSide)
	}
	var resp []LeverageEstimatedInfo
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getLeverateEstimatedInfoEPL, http.MethodGet, common.EncodeURLValues("account/adjust-leverage-info", params), nil, &resp, true)
}

// GetMaximumLoanOfInstrument returns list of maximum loan of instruments.
func (ok *Okx) GetMaximumLoanOfInstrument(ctx context.Context, instrumentID, marginMode string, mgnCurrency currency.Code) ([]MaximumLoanInstrument, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	if marginMode == "" {
		return nil, errMissingMarginMode
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	params.Set("mgnMode", marginMode)
	if !mgnCurrency.IsEmpty() {
		params.Set("mgnCcy", mgnCurrency.String())
	}
	var resp []MaximumLoanInstrument
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getTheMaximumLoanOfInstrumentEPL, http.MethodGet, common.EncodeURLValues("account/max-loan", params), nil, &resp, true)
}

// GetFee returns Cryptocurrency trade fee, and offline trade fee
func (ok *Okx) GetFee(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	// Here the Asset Type for the instrument Type is needed for getting the CryptocurrencyTrading Fee.
	var fee float64
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		uly, err := ok.GetUnderlying(feeBuilder.Pair, asset.Spot)
		if err != nil {
			return 0, err
		}
		responses, err := ok.GetTradeFee(ctx, okxInstTypeSpot, uly, "")
		if err != nil {
			return 0, err
		} else if len(responses) == 0 {
			return 0, common.ErrNoResponse
		}
		if feeBuilder.IsMaker {
			if feeBuilder.Pair.Quote.Equal(currency.USDC) {
				fee = responses[0].FeeRateMakerUSDC.Float64()
			} else if fee = responses[0].FeeRateMaker.Float64(); fee == 0 {
				fee = responses[0].FeeRateMakerUSDT.Float64()
			}
		} else {
			if feeBuilder.Pair.Quote.Equal(currency.USDC) {
				fee = responses[0].FeeRateTakerUSDC.Float64()
			} else if fee = responses[0].FeeRateTaker.Float64(); fee == 0 {
				fee = responses[0].FeeRateTakerUSDT.Float64()
			}
		}
		if fee != 0 {
			fee = -fee // Negative fee rate means commission else rebate.
		}
		return fee * feeBuilder.Amount * feeBuilder.PurchasePrice, nil
	case exchange.OfflineTradeFee:
		return 0.0015 * feeBuilder.PurchasePrice * feeBuilder.Amount, nil
	}
	return fee, nil
}

// GetTradeFee query trade fee rate of various instrument types and instrument ids.
func (ok *Okx) GetTradeFee(ctx context.Context, instrumentType, instrumentID, underlying string) ([]TradeFeeRate, error) {
	if instrumentType == "" {
		return nil, errInstrumentTypeRequired
	}
	params := url.Values{}
	instrumentType = strings.ToUpper(instrumentType)
	params.Set("instType", instrumentType)
	if instrumentID != "" {
		params.Set("instId", instrumentID)
	}
	if underlying != "" {
		params.Set("uly", underlying)
	}
	var resp []TradeFeeRate
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getFeeRatesEPL, http.MethodGet, common.EncodeURLValues("account/trade-fee", params), nil, &resp, true)
}

// GetInterestAccruedData returns accrued interest data
func (ok *Okx) GetInterestAccruedData(ctx context.Context, loanType, limit int64, ccy currency.Code, instrumentID, marginMode string, after, before time.Time) ([]InterestAccruedData, error) {
	params := url.Values{}
	if loanType == 1 || loanType == 2 {
		params.Set("type", strconv.FormatInt(loanType, 10))
	}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if instrumentID != "" {
		params.Set("instId", instrumentID)
	}
	if marginMode != "" {
		params.Set("mgnMode", strings.ToLower(marginMode))
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []InterestAccruedData
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getInterestAccruedDataEPL, http.MethodGet, common.EncodeURLValues("account/interest-accrued", params), nil, &resp, true)
}

// GetInterestRate get the user's current leveraged currency borrowing interest rate
func (ok *Okx) GetInterestRate(ctx context.Context, ccy currency.Code) ([]InterestRateResponse, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	var resp []InterestRateResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getInterestRateEPL, http.MethodGet, common.EncodeURLValues("account/interest-rate", params), nil, &resp, true)
}

// SetGreeks set the display type of Greeks. PA: Greeks in coins BS: Black-Scholes Greeks in dollars
func (ok *Okx) SetGreeks(ctx context.Context, greeksType string) (*GreeksType, error) {
	greeksType = strings.ToUpper(greeksType)
	if greeksType != "PA" && greeksType != "BS" {
		return nil, errMissingValidGreeksType
	}
	input := &GreeksType{
		GreeksType: greeksType,
	}
	var resp *GreeksType
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, setGreeksEPL, http.MethodPost, "account/set-greeks", input, &resp, true)
}

// IsolatedMarginTradingSettings to set the currency margin and futures/perpetual Isolated margin trading mode.
func (ok *Okx) IsolatedMarginTradingSettings(ctx context.Context, arg IsolatedMode) (*IsolatedMode, error) {
	arg.IsoMode = strings.ToLower(arg.IsoMode)
	if arg.IsoMode != "automatic" &&
		arg.IsoMode != "autonomy" {
		return nil, errMissingIsolatedMarginTradingSetting
	}
	arg.InstrumentType = strings.ToUpper(arg.InstrumentType)
	if arg.InstrumentType != okxInstTypeMargin &&
		arg.InstrumentType != okxInstTypeContract {
		return nil, fmt.Errorf("%w, received '%v' only margin and contract instrument types are allowed", errInvalidInstrumentType, arg.InstrumentType)
	}
	var resp *IsolatedMode
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, isolatedMarginTradingSettingsEPL, http.MethodPost, "account/set-isolated-mode", &arg, &resp, true)
}

// ManualBorrowAndRepayInQuickMarginMode creates a new manual borrow and repay
func (ok *Okx) ManualBorrowAndRepayInQuickMarginMode(ctx context.Context, arg *BorrowAndRepay) (*BorrowAndRepay, error) {
	if arg.Amount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	if arg.LoanCcy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Side == "" {
		return nil, fmt.Errorf("%w, possible values are 'borrow' and 'repay'", order.ErrSideIsInvalid)
	}
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	var resp *BorrowAndRepay
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, manualBorrowAndRepayEPL, http.MethodPost, "account/quick-margin-borrow-repay", arg, &resp, true)
}

// GetBorrowAndRepayHistoryInQuickMarginMode retrieves borrow and repay history in quick margin mode.
func (ok *Okx) GetBorrowAndRepayHistoryInQuickMarginMode(ctx context.Context, instrumentID currency.Pair, ccy currency.Code, side, afterPaginationID, beforePaginationID string, beginTime, endTime time.Time, limit int64) ([]BorrowRepayHistoryItem, error) {
	params := url.Values{}
	if !instrumentID.IsEmpty() {
		params.Set("instId", instrumentID.String())
	}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if side != "" {
		params.Set("side", side)
	}
	if afterPaginationID != "" {
		params.Set("after", afterPaginationID)
	}
	if beforePaginationID != "" {
		params.Set("before", beforePaginationID)
	}
	if !beginTime.IsZero() {
		params.Set("begin", strconv.FormatInt(beginTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("end", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []BorrowRepayHistoryItem
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getBorrowAndRepayHistoryEPL, http.MethodGet, common.EncodeURLValues("account/quick-margin-borrow-repay-history", params), nil, &resp, true)
}

// GetMaximumWithdrawals retrieves the maximum transferable amount from trading account to funding account.
func (ok *Okx) GetMaximumWithdrawals(ctx context.Context, ccy currency.Code) ([]MaximumWithdrawal, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	var resp []MaximumWithdrawal
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getMaximumWithdrawalsEPL, http.MethodGet, common.EncodeURLValues("account/max-withdrawal", params), nil, &resp, true)
}

// GetAccountRiskState gets the account risk status.
// only applicable to Portfolio margin account
func (ok *Okx) GetAccountRiskState(ctx context.Context) ([]AccountRiskState, error) {
	var resp []AccountRiskState
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getAccountRiskStateEPL, http.MethodGet, "account/risk-state", nil, &resp, true)
}

// VIPLoansBorrowAndRepay creates VIP borrow or repay for a currency.
func (ok *Okx) VIPLoansBorrowAndRepay(ctx context.Context, arg LoanBorrowAndReplayInput) (*LoanBorrowAndReplay, error) {
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.Amount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	var resp *LoanBorrowAndReplay
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, vipLoansBorrowAnsRepayEPL, http.MethodPost, "account/borrow-repay", &arg, &resp, true)
}

// GetBorrowAndRepayHistoryForVIPLoans retrieves borrow and repay history for VIP loans.
func (ok *Okx) GetBorrowAndRepayHistoryForVIPLoans(ctx context.Context, ccy currency.Code, after, before time.Time, limit int64) ([]BorrowRepayHistory, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []BorrowRepayHistory
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getBorrowAnsRepayHistoryHistoryEPL, http.MethodGet, common.EncodeURLValues("account/borrow-repay-history", params), nil, &resp, true)
}

// GetVIPInterestAccruedData retrieves VIP interest accrued data
func (ok *Okx) GetVIPInterestAccruedData(ctx context.Context, ccy currency.Code, orderID string, after, before time.Time, limit int64) ([]VIPInterestData, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if orderID != "" {
		params.Set("ordId", orderID)
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []VIPInterestData
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getVIPInterestAccruedDataEPL, http.MethodGet, common.EncodeURLValues("account/vip-interest-accrued", params), nil, &resp, true)
}

// GetVIPInterestDeductedData retrieves a VIP interest deducted data
func (ok *Okx) GetVIPInterestDeductedData(ctx context.Context, ccy currency.Code, orderID string, after, before time.Time, limit int64) ([]VIPInterestData, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if orderID != "" {
		params.Set("ordId", orderID)
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []VIPInterestData
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getVIPInterestDeductedDataEPL, http.MethodGet, common.EncodeURLValues("account/vip-interest-deducted", params), nil, &resp, true)
}

// GetVIPLoanOrderList retrieves VIP loan order list
// state: possible values are 1:Borrowing 2:Borrowed 3:Repaying 4:Repaid 5:Borrow failed
func (ok *Okx) GetVIPLoanOrderList(ctx context.Context, orderID, state string, ccy currency.Code, after, before time.Time, limit int64) ([]VIPLoanOrder, error) {
	params := url.Values{}
	if orderID != "" {
		params.Set("ordId", orderID)
	}
	if state != "" {
		params.Set("state", state)
	}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []VIPLoanOrder
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getVIPLoanOrderListEPL, http.MethodGet, common.EncodeURLValues("account/vip-loan-order-list", params), nil, &resp, true)
}

// GetVIPLoanOrderDetail retrieves list of loan order details.
func (ok *Okx) GetVIPLoanOrderDetail(ctx context.Context, orderID string, ccy currency.Code, after, before time.Time, limit int64) (*VIPLoanOrderDetail, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("ordId", orderID)
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *VIPLoanOrderDetail
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getVIPLoanOrderDetailEPL, http.MethodGet, common.EncodeURLValues("account/vip-loan-order-detail", params), nil, &resp, true)
}

// GetBorrowInterestAndLimit borrow interest and limit
func (ok *Okx) GetBorrowInterestAndLimit(ctx context.Context, loanType int64, ccy currency.Code) ([]BorrowInterestAndLimitResponse, error) {
	params := url.Values{}
	if loanType == 1 || loanType == 2 {
		params.Set("type", strconv.FormatInt(loanType, 10))
	}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	var resp []BorrowInterestAndLimitResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getBorrowInterestAndLimitEPL, http.MethodGet, common.EncodeURLValues("account/interest-limits", params), nil, &resp, true)
}

// PositionBuilder calculates portfolio margin information for simulated position or current position of the user. You can add up to 200 simulated positions in one request.
// Instrument type SWAP  FUTURES, and OPTION are supported
func (ok *Okx) PositionBuilder(ctx context.Context, arg PositionBuilderInput) ([]PositionBuilderResponse, error) {
	arg.InstrumentType = strings.ToUpper(arg.InstrumentType)
	var resp []PositionBuilderResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, positionBuilderEPL, http.MethodPost, "account/simulated_margin", &arg, &resp, true)
}

// GetGreeks retrieves a greeks list of all assets in the account.
func (ok *Okx) GetGreeks(ctx context.Context, ccy currency.Code) ([]GreeksItem, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	var resp []GreeksItem
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getGreeksEPL, http.MethodGet, common.EncodeURLValues("account/greeks", params), nil, &resp, true)
}

// GetPMPositionLimitation retrieve cross position limitation of SWAP/FUTURES/OPTION under Portfolio margin mode.
func (ok *Okx) GetPMPositionLimitation(ctx context.Context, instrumentType, underlying string) ([]PMLimitationResponse, error) {
	if instrumentType == "" {
		return nil, errInstrumentTypeRequired
	}
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	params := url.Values{}
	params.Set("instType", strings.ToUpper(instrumentType))
	params.Set("uly", underlying)
	var resp []PMLimitationResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getPMLimitationEPL, http.MethodGet, common.EncodeURLValues("account/position-tiers", params), nil, &resp, true)
}

// SetRiskOffsetType configure the risk offset type in portfolio margin mode.
// riskOffsetType possible values are:
// 1: Spot-derivatives (USDT) risk offset
// 2: Spot-derivatives (Crypto) risk offset
// 3:Derivatives only mode
func (ok *Okx) SetRiskOffsetType(ctx context.Context, riskOffsetType string) (*RiskOffsetType, error) {
	if riskOffsetType == "" {
		return nil, errors.New("missing risk offset type")
	}
	var resp *RiskOffsetType
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, setRiskOffsetLimiterEPL, http.MethodPost, "account/set-riskOffset-type", &map[string]string{"type": riskOffsetType}, &resp, true)
}

// ActivateOption activates option
func (ok *Okx) ActivateOption(ctx context.Context) (time.Time, error) {
	resp := &struct {
		Timestamp convert.ExchangeTime `json:"ts"`
	}{}
	return resp.Timestamp.Time(), ok.SendHTTPRequest(ctx, exchange.RestSpot, activateOptionEPL, http.MethodPost, "account/activate-option", nil, &resp, true)
}

// SetAutoLoan only applicable to Multi-currency margin and Portfolio margin
func (ok *Okx) SetAutoLoan(ctx context.Context, autoLoan bool) (*AutoLoan, error) {
	var resp *AutoLoan
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, setAutoLoanEPL, http.MethodPost, "account/set-auto-loan", &AutoLoan{AutoLoan: autoLoan}, &resp, true)
}

// SetAccountMode to set on the Web/App for the first set of every account mode.
// Account mode 1: Simple mode 2: Single-currency margin mode  3: Multi-currency margin code  4: Portfolio margin mode
func (ok *Okx) SetAccountMode(ctx context.Context, accountLevel string) (*AccountMode, error) {
	var resp *AccountMode
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, setAccountLevelEPL, http.MethodPost, "account/set-account-level", &map[string]string{"acctLv": accountLevel}, &resp, true)
}

// ResetMMPStatus reset the MMP status to be inactive.
// you can unfreeze by this endpoint once MMP is triggered.
// Only applicable to Option in Portfolio Margin mode, and MMP privilege is required.
func (ok *Okx) ResetMMPStatus(ctx context.Context, instrumentType, instrumentFamily string) (*MMPStatusResponse, error) {
	if instrumentFamily == "" {
		return nil, errInstrumentFamilyRequired
	}
	arg := &struct {
		InstrumentType   string `json:"instType,omitempty"`
		InstrumentFamily string `json:"instFamily"`
	}{
		InstrumentType:   instrumentType,
		InstrumentFamily: instrumentFamily,
	}
	var resp *MMPStatusResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, resetMMPStatusEPL, http.MethodPost, "account/mmp-reset", arg, &resp, true)
}

// SetMMP set MMP configure
// Only applicable to Option in Portfolio Margin mode, and MMP privilege is required.
func (ok *Okx) SetMMP(ctx context.Context, arg *MMPConfig) (*MMPConfig, error) {
	if arg == nil || *arg == (MMPConfig{}) {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentFamily == "" {
		return nil, errInstrumentFamilyRequired
	}
	if arg.QuantityLimit <= 0 {
		return nil, errInvalidQuantityLimit
	}
	var resp *MMPConfig
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, setMMPEPL, http.MethodPost, "account/mmp-config", arg, &resp, true)
}

// GetMMPConfig retrieves MMP configure information
// Only applicable to Option in Portfolio Margin mode, and MMP privilege is required.
func (ok *Okx) GetMMPConfig(ctx context.Context, instrumentFamily string) ([]MMPConfigDetail, error) {
	if instrumentFamily == "" {
		return nil, errInstrumentFamilyRequired
	}
	var resp []MMPConfigDetail
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getMMPConfigEPL, http.MethodGet, "account/mmp-config?instFamily="+instrumentFamily, nil, &resp, true)
}

/********************************** Subaccount Endpoints ***************************************************/

// ViewSubAccountList applies to master accounts only
func (ok *Okx) ViewSubAccountList(ctx context.Context, enable bool, subaccountName string, after, before time.Time, limit int64) ([]SubaccountInfo, error) {
	params := url.Values{}
	params.Set("enable", strconv.FormatBool(enable))
	if subaccountName != "" {
		params.Set("subAcct", subaccountName)
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []SubaccountInfo
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, viewSubaccountListEPL, http.MethodGet, common.EncodeURLValues("users/subaccount/list", params), nil, &resp, true)
}

// ResetSubAccountAPIKey applies to master accounts only and master accounts APIKey must be linked to IP addresses.
func (ok *Okx) ResetSubAccountAPIKey(ctx context.Context, arg *SubAccountAPIKeyParam) (*SubAccountAPIKeyResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.SubAccountName == "" {
		return nil, errInvalidSubAccountName
	}
	if arg.APIKey == "" {
		return nil, errInvalidAPIKey
	}
	if arg.IP != "" && net.ParseIP(arg.IP).To4() == nil {
		return nil, errInvalidIPAddress
	}
	params := url.Values{}
	params.Set("subAcct", arg.SubAccountName)
	if arg.APIKeyPermission == "" && len(arg.Permissions) != 0 {
		for x := range arg.Permissions {
			if arg.Permissions[x] != "read" &&
				arg.Permissions[x] != "withdraw" &&
				arg.Permissions[x] != "trade" &&
				arg.Permissions[x] != "read_only" {
				return nil, errInvalidAPIKeyPermission
			}
			if x != 0 {
				arg.APIKeyPermission += ","
			}
			arg.APIKeyPermission += arg.Permissions[x]
		}
	} else if arg.APIKeyPermission != "read" && arg.APIKeyPermission != "withdraw" && arg.APIKeyPermission != "trade" && arg.APIKeyPermission != "read_only" {
		return nil, errInvalidAPIKeyPermission
	}
	var resp *SubAccountAPIKeyResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, resetSubAccountAPIKeyEPL, http.MethodPost, "users/subaccount/modify-apikey", &arg, &resp, true)
}

// GetSubaccountTradingBalance query detailed balance info of Trading Account of a sub-account via the master account (applies to master accounts only)
func (ok *Okx) GetSubaccountTradingBalance(ctx context.Context, subaccountName string) ([]SubaccountBalanceResponse, error) {
	if subaccountName == "" {
		return nil, errMissingRequiredParameterSubaccountName
	}
	params := url.Values{}
	params.Set("subAcct", subaccountName)
	var resp []SubaccountBalanceResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getSubaccountTradingBalanceEPL, http.MethodGet, common.EncodeURLValues("account/subaccount/balances", params), nil, &resp, true)
}

// GetSubaccountFundingBalance query detailed balance info of Funding Account of a sub-account via the master account (applies to master accounts only)
func (ok *Okx) GetSubaccountFundingBalance(ctx context.Context, subaccountName string, ccy currency.Code) ([]FundingBalance, error) {
	if subaccountName == "" {
		return nil, errMissingRequiredParameterSubaccountName
	}
	params := url.Values{}
	params.Set("subAcct", subaccountName)
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	var resp []FundingBalance
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getSubaccountFundingBalanceEPL, http.MethodGet, common.EncodeURLValues("asset/subaccount/balances", params), nil, &resp, true)
}

// GetSubAccountMaximumWithdrawal retrieve the maximum withdrawal information of a sub-account via the master account (applies to master accounts only). If no currency is specified, the transferable amount of all owned currencies will be returned.
func (ok *Okx) GetSubAccountMaximumWithdrawal(ctx context.Context, subAccountName string, ccy currency.Code) ([]SubAccountMaximumWithdrawal, error) {
	if subAccountName == "" {
		return nil, errMissingRequiredParameterSubaccountName
	}
	params := url.Values{}
	params.Set("subAcct", subAccountName)
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	var resp []SubAccountMaximumWithdrawal
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getSubAccountMaxWithdrawalEPL, http.MethodGet, "account/subaccount/max-withdrawal", nil, &resp, true)
}

// HistoryOfSubaccountTransfer retrieves subaccount transfer histories; applies to master accounts only.
// Retrieve the transfer data for the last 3 months.
func (ok *Okx) HistoryOfSubaccountTransfer(ctx context.Context, ccy currency.Code, subaccountType, subaccountName string, before, after time.Time, limit int64) ([]SubaccountBillItem, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if subaccountType != "" {
		params.Set("type", subaccountType)
	}
	if subaccountName != "" {
		params.Set("subacct", subaccountName)
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []SubaccountBillItem
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, historyOfSubaccountTransferEPL, http.MethodGet, common.EncodeURLValues("asset/subaccount/bills", params), nil, &resp, true)
}

// GetHistoryOfManagedSubAccountTransfer retrieves managed sub-account transfers.
// nly applicable to the trading team's master account to getting transfer records of managed sub accounts entrusted to oneself.
func (ok *Okx) GetHistoryOfManagedSubAccountTransfer(ctx context.Context, ccy currency.Code, transferType, subAccountName, subAccountUID string, after, before time.Time, limit int64) ([]SubAccountTransfer, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if transferType != "" {
		params.Set("type", transferType)
	}
	if subAccountName != "" {
		params.Set("subAcct", subAccountName)
	}
	if subAccountUID != "" {
		params.Set("subUid", subAccountUID)
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []SubAccountTransfer
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, managedSubAccountTransferEPL, http.MethodGet, common.EncodeURLValues("asset/subaccount/managed-subaccount-bills", params), nil, &resp, true)
}

// MasterAccountsManageTransfersBetweenSubaccounts master accounts manage the transfers between sub-accounts applies to master accounts only
func (ok *Okx) MasterAccountsManageTransfersBetweenSubaccounts(ctx context.Context, arg *SubAccountAssetTransferParams) ([]TransferIDInfo, error) {
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Amount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	if arg.From == 0 {
		return nil, errInvalidSubaccount
	}
	if arg.To != 6 && arg.To != 18 {
		return nil, errInvalidSubaccount
	}
	if arg.FromSubAccount == "" {
		return nil, errMissingInitialSubaccountName
	}
	if arg.ToSubAccount == "" {
		return nil, errMissingDestinationSubaccountName
	}
	var resp []TransferIDInfo
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, masterAccountsManageTransfersBetweenSubaccountEPL, http.MethodPost, "asset/subaccount/transfer", &arg, &resp, true)
}

// SetPermissionOfTransferOut set permission of transfer out for sub-account(only applicable to master account). Sub-account can transfer out to master account by default.
func (ok *Okx) SetPermissionOfTransferOut(ctx context.Context, arg PermissionOfTransfer) ([]PermissionOfTransfer, error) {
	if arg.SubAcct == "" {
		return nil, errMissingRequiredParameterSubaccountName
	}
	var resp []PermissionOfTransfer
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, setPermissionOfTransferOutEPL, http.MethodPost, "users/subaccount/set-transfer-out", &arg, &resp, true)
}

// GetCustodyTradingSubaccountList the trading team uses this interface to view the list of sub-accounts currently under escrow
// usersEntrustSubaccountList ="users/entrust-subaccount-list"
func (ok *Okx) GetCustodyTradingSubaccountList(ctx context.Context, subaccountName string) ([]SubaccountName, error) {
	params := url.Values{}
	if subaccountName != "" {
		params.Set("setAcct", subaccountName)
	}
	var resp []SubaccountName
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getCustodyTradingSubaccountListEPL, http.MethodGet, common.EncodeURLValues("users/entrust-subaccount-list", params), nil, &resp, true)
}

// SetSubAccountVIPLoanAllocation set the VIP loan allocation of sub-accounts. Only Applicable to master account API keys with Trade access.
func (ok *Okx) SetSubAccountVIPLoanAllocation(ctx context.Context, arg *SubAccountLoanAllocationParam) (bool, error) {
	if arg == nil || len(arg.Alloc) == 0 {
		return false, common.ErrNilPointer
	}
	for a := range arg.Alloc {
		if arg.Alloc[a].SubAcct == "" {
			return false, errMissingRequiredParameterSubaccountName
		}
		if arg.Alloc[a].LoanAlloc < 0 {
			return false, errInvalidLoanAllocationValue
		}
	}
	resp := &struct {
		Result bool `json:"result"`
	}{}
	return resp.Result, ok.SendHTTPRequest(ctx, exchange.RestSpot, setSubAccountVIPLoanAllocationEPL, http.MethodPost, "account/subaccount/set-loan-allocation", arg, resp, true)
}

// GetSubAccountBorrowInterestAndLimit retrieves sub-account borrow interest and limit
// Only applicable to master account API keys. Only return VIP loan information
func (ok *Okx) GetSubAccountBorrowInterestAndLimit(ctx context.Context, subAccount string, ccy currency.Code) ([]SubAccounBorrowInterestAndLimit, error) {
	if subAccount == "" {
		return nil, errMissingRequiredParameterSubaccountName
	}
	params := url.Values{}
	params.Set("subAcct", subAccount)
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	var resp []SubAccounBorrowInterestAndLimit
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getSubAccountBorrowInterestAndLimitEPL, http.MethodGet, common.EncodeURLValues("account/subaccount/interest-limits", params), nil, &resp, true)
}

/*************************************** Grid Trading Endpoints ***************************************************/

// PlaceGridAlgoOrder place spot grid algo order.
func (ok *Okx) PlaceGridAlgoOrder(ctx context.Context, arg *GridAlgoOrder) (*GridAlgoOrderIDResponse, error) {
	if arg == nil || *arg == (GridAlgoOrder{}) {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	arg.AlgoOrdType = strings.ToLower(arg.AlgoOrdType)
	if arg.AlgoOrdType != AlgoOrdTypeGrid && arg.AlgoOrdType != AlgoOrdTypeContractGrid {
		return nil, errMissingAlgoOrderType
	}
	if arg.MaxPrice <= 0 {
		return nil, errInvalidMaximumPrice
	}
	if arg.MinPrice < 0 {
		return nil, errInvalidMinimumPrice
	}
	if arg.GridQuantity < 0 {
		return nil, errInvalidGridQuantity
	}
	isSpotGridOrder := arg.QuoteSize > 0 || arg.BaseSize > 0
	if !isSpotGridOrder {
		if arg.Size <= 0 {
			return nil, fmt.Errorf("%w 'size' is required", order.ErrAmountMustBeSet)
		}
		arg.Direction = strings.ToLower(arg.Direction)
		if arg.Direction != positionSideLong && arg.Direction != positionSideShort && arg.Direction != "neutral" {
			return nil, errMissingRequiredArgumentDirection
		}
		if arg.Lever == "" {
			return nil, errRequiredParameterMissingLeverage
		}
	}
	var resp []GridAlgoOrderIDResponse
	err := ok.SendHTTPRequest(ctx, exchange.RestSpot, gridTradingEPL, http.MethodPost, "tradingBot/grid/order-algo", &arg, &resp, true)
	if err != nil {
		if len(resp) != 1 {
			return nil, err
		}
		return nil, fmt.Errorf("error code:%s message: %v", resp[0].SCode, resp[0].SMsg)
	}
	if len(resp) == 1 {
		return &resp[0], nil
	}
	return nil, fmt.Errorf("%w, received invalid response", common.ErrNoResponse)
}

// AmendGridAlgoOrder supported contract grid algo order amendment.
func (ok *Okx) AmendGridAlgoOrder(ctx context.Context, arg GridAlgoOrderAmend) (*GridAlgoOrderIDResponse, error) {
	if arg.AlgoID == "" {
		return nil, fmt.Errorf("%w, algoId is required", order.ErrOrderIDNotSet)
	}
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	var resp []GridAlgoOrderIDResponse
	err := ok.SendHTTPRequest(ctx, exchange.RestSpot, amendGridAlgoOrderEPL, http.MethodPost, "tradingBot/grid/amend-order-algo", &arg, &resp, true)
	if err != nil {
		if len(resp) != 1 {
			return nil, err
		}
		return nil, fmt.Errorf("error code:%s message: %v", resp[0].SCode, resp[0].SMsg)
	}
	if len(resp) == 1 {
		return &resp[0], nil
	}
	return nil, fmt.Errorf("%w, received invalid response", common.ErrNoResponse)
}

// StopGridAlgoOrder stop a batch of grid algo orders.
func (ok *Okx) StopGridAlgoOrder(ctx context.Context, arg []StopGridAlgoOrderRequest) ([]GridAlgoOrderIDResponse, error) {
	if len(arg) == 0 {
		return nil, common.ErrNilPointer
	} else if len(arg) > 10 {
		return nil, fmt.Errorf("%w, a maximum of 10 orders can be canceled per request", errTooManyArgument)
	}
	for x := range arg {
		if arg[x].AlgoID == "" {
			return nil, fmt.Errorf("%w, algoId is required", order.ErrOrderIDNotSet)
		}
		if arg[x].InstrumentID == "" {
			return nil, errMissingInstrumentID
		}
		arg[x].AlgoOrderType = strings.ToLower(arg[x].AlgoOrderType)
		if arg[x].AlgoOrderType != AlgoOrdTypeGrid && arg[x].AlgoOrderType != AlgoOrdTypeContractGrid {
			return nil, errMissingAlgoOrderType
		}
		if arg[x].StopType != 1 && arg[x].StopType != 2 {
			return nil, errMissingValidStopType
		}
	}
	var resp []GridAlgoOrderIDResponse
	err := ok.SendHTTPRequest(ctx, exchange.RestSpot, stopGridAlgoOrderEPL, http.MethodPost, "tradingBot/grid/stop-order-algo", arg, &resp, true)
	if err != nil {
		if len(resp) == 0 {
			return nil, err
		}
		return nil, fmt.Errorf("error code:%s message: %v", resp[0].SCode, resp[0].SMsg)
	}
	return resp, nil
}

// ClosePositionForContractrid close position when the contract grid stop type is 'keep position'.
func (ok *Okx) ClosePositionForContractrid(ctx context.Context, arg *ClosePositionParams) (*ClosePositionContractGridResponse, error) {
	if arg == nil || *arg == (ClosePositionParams{}) {
		return nil, common.ErrNilPointer
	}
	if arg.AlgoID == "" {
		return nil, fmt.Errorf("%w, algoId is required", order.ErrOrderIDNotSet)
	}
	if !arg.MarketCloseAllPositions && arg.Size <= 0 {
		return nil, fmt.Errorf("%w 'size' is required", order.ErrAmountMustBeSet)
	}
	if !arg.MarketCloseAllPositions && arg.Price <= 0 {
		return nil, errInvalidMinimumPrice
	}
	var resp *ClosePositionContractGridResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, closePositionForForContractGridEPL, http.MethodPost, "tradingBot/grid/close-position", arg, &resp, true)
}

// CancelClosePositionOrderForContractGrid cancels close position order for contract grid
func (ok *Okx) CancelClosePositionOrderForContractGrid(ctx context.Context, arg *CancelClosePositionOrder) (*ClosePositionContractGridResponse, error) {
	if arg == nil || *arg == (CancelClosePositionOrder{}) {
		return nil, common.ErrNilPointer
	}
	if arg.AlgoID == "" {
		return nil, fmt.Errorf("%w, algoId is required", order.ErrOrderIDNotSet)
	}
	if arg.OrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp *ClosePositionContractGridResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, cancelClosePositionOrderForContractGridEPL, http.MethodPost, "tradingBot/grid/cancel-close-order", arg, &resp, true)
}

// InstantTriggerGridAlgoOrder triggers grid algo order
func (ok *Okx) InstantTriggerGridAlgoOrder(ctx context.Context, algoID string) (*TriggeredGridAlgoOrderInfo, error) {
	var resp *TriggeredGridAlgoOrderInfo
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, instantTriggerGridAlgoOrderEPL, http.MethodPost, "tradingBot/grid/order-instant-trigger", &map[string]string{"algoId": algoID}, &resp, true)
}

// GetGridAlgoOrdersList retrieves list of pending grid algo orders with the complete data.
func (ok *Okx) GetGridAlgoOrdersList(ctx context.Context, algoOrderType, algoID,
	instrumentID, instrumentType,
	after, before string, limit int64) ([]GridAlgoOrderResponse, error) {
	return ok.getGridAlgoOrders(ctx, algoOrderType, algoID,
		instrumentID, instrumentType,
		after, before, "tradingBot/grid/orders-algo-pending", limit)
}

// GetGridAlgoOrderHistory retrieves list of grid algo orders with the complete data including the stopped orders.
func (ok *Okx) GetGridAlgoOrderHistory(ctx context.Context, algoOrderType, algoID,
	instrumentID, instrumentType,
	after, before string, limit int64) ([]GridAlgoOrderResponse, error) {
	return ok.getGridAlgoOrders(ctx, algoOrderType, algoID,
		instrumentID, instrumentType,
		after, before, "tradingBot/grid/orders-algo-history", limit)
}

// getGridAlgoOrderList retrieves list of grid algo orders with the complete data.
func (ok *Okx) getGridAlgoOrders(ctx context.Context, algoOrderType, algoID,
	instrumentID, instrumentType,
	after, before, route string, limit int64) ([]GridAlgoOrderResponse, error) {
	algoOrderType = strings.ToLower(algoOrderType)
	if algoOrderType != AlgoOrdTypeGrid && algoOrderType != AlgoOrdTypeContractGrid {
		return nil, errMissingAlgoOrderType
	}
	params := url.Values{}
	params.Set("algoOrdType", algoOrderType)
	if algoID != "" {
		params.Set("algoId", algoID)
	}
	if instrumentID != "" {
		params.Set("instId", instrumentID)
	}
	if instrumentType != "" {
		params.Set("instType", strings.ToUpper(instrumentType))
	}
	if after != "" {
		params.Set("after", after)
	}
	if before != "" {
		params.Set("before", before)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	epl := getGridAlgoOrderListEPL
	if route == "tradingBot/grid/orders-algo-history" {
		epl = getGridAlgoOrderHistoryEPL
	}
	var resp []GridAlgoOrderResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, epl, http.MethodGet, common.EncodeURLValues(route, params), nil, &resp, true)
}

// GetGridAlgoOrderDetails retrieves grid algo order details
func (ok *Okx) GetGridAlgoOrderDetails(ctx context.Context, algoOrderType, algoID string) (*GridAlgoOrderResponse, error) {
	if algoOrderType != AlgoOrdTypeGrid &&
		algoOrderType != AlgoOrdTypeContractGrid {
		return nil, errMissingAlgoOrderType
	}
	if algoID == "" {
		return nil, fmt.Errorf("%w, algoId is required", order.ErrOrderIDNotSet)
	}
	params := url.Values{}
	params.Set("algoOrdType", algoOrderType)
	params.Set("algoId", algoID)
	var resp *GridAlgoOrderResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getGridAlgoOrderDetailsEPL, http.MethodGet, common.EncodeURLValues("tradingBot/grid/orders-algo-details", params), nil, &resp, true)
}

// GetGridAlgoSubOrders retrieves grid algo sub orders
func (ok *Okx) GetGridAlgoSubOrders(ctx context.Context, algoOrderType, algoID, subOrderType, groupID, after, before string, limit int64) ([]GridAlgoOrderResponse, error) {
	if algoOrderType != AlgoOrdTypeGrid &&
		algoOrderType != AlgoOrdTypeContractGrid {
		return nil, errMissingAlgoOrderType
	}
	if algoID == "" {
		return nil, fmt.Errorf("%w, algoId is required", order.ErrOrderIDNotSet)
	}
	if subOrderType != "live" && subOrderType != order.Filled.String() {
		return nil, errMissingSubOrderType
	}
	params := url.Values{}
	params.Set("algoOrdType", algoOrderType)
	params.Set("algoId", algoID)
	params.Set("type", subOrderType)
	if groupID != "" {
		params.Set("groupId", groupID)
	}
	if after != "" {
		params.Set("after", after)
	}
	if before != "" {
		params.Set("before", before)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []GridAlgoOrderResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getGridAlgoSubOrdersEPL, http.MethodGet, common.EncodeURLValues("tradingBot/grid/sub-orders", params), nil, &resp, true)
}

// GetGridAlgoOrderPositions retrieves grid algo order positions.
func (ok *Okx) GetGridAlgoOrderPositions(ctx context.Context, algoOrderType, algoID string) ([]AlgoOrderPosition, error) {
	if algoOrderType != AlgoOrdTypeGrid && algoOrderType != AlgoOrdTypeContractGrid {
		return nil, errInvalidAlgoOrderType
	}
	if algoID == "" {
		return nil, fmt.Errorf("%w, algoId is required", order.ErrOrderIDNotSet)
	}
	params := url.Values{}
	params.Set("algoOrdType", algoOrderType)
	params.Set("algoId", algoID)
	var resp []AlgoOrderPosition
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getGridAlgoOrderPositionsEPL, http.MethodGet, common.EncodeURLValues("tradingBot/grid/positions", params), nil, &resp, true)
}

// SpotGridWithdrawProfit returns the spot grid orders withdrawal profit given an instrument id.
func (ok *Okx) SpotGridWithdrawProfit(ctx context.Context, algoID string) (*AlgoOrderWithdrawalProfit, error) {
	if algoID == "" {
		return nil, fmt.Errorf("%w, algoId is required", order.ErrOrderIDNotSet)
	}
	input := &struct {
		AlgoID string `json:"algoId"`
	}{
		AlgoID: algoID,
	}
	var resp *AlgoOrderWithdrawalProfit
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, spotGridWithdrawIncomeEPL, http.MethodPost, "tradingBot/grid/withdraw-income", input, &resp, true)
}

// ComputeMarginBalance computes margin balance with 'add' and 'reduce' balance type
func (ok *Okx) ComputeMarginBalance(ctx context.Context, arg MarginBalanceParam) (*ComputeMarginBalance, error) {
	if arg.AlgoID == "" {
		return nil, fmt.Errorf("%w, algoId is required", order.ErrOrderIDNotSet)
	}
	if arg.Type != "add" && arg.Type != "reduce" {
		return nil, errInvalidMarginTypeAdjust
	}
	var resp *ComputeMarginBalance
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, computeMarginBalanceEPL, http.MethodPost, "tradingBot/grid/compute-margin-balance", &arg, &resp, true)
}

// AdjustMarginBalance retrieves adjust margin balance with 'add' and 'reduce' balance type
func (ok *Okx) AdjustMarginBalance(ctx context.Context, arg MarginBalanceParam) (*AdjustMarginBalanceResponse, error) {
	if arg.AlgoID == "" {
		return nil, fmt.Errorf("%w, algoId is required", order.ErrOrderIDNotSet)
	}
	if arg.Type != "add" && arg.Type != "reduce" {
		return nil, errInvalidMarginTypeAdjust
	}
	if arg.Percentage <= 0 && arg.Amount < 0 {
		return nil, errors.New("either percentage or amount is required")
	}
	var resp *AdjustMarginBalanceResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, adjustMarginBalanceEPL, http.MethodPost, "tradingBot/grid/margin-balance", &arg, &resp, true)
}

// GetGridAIParameter retrieves grid AI parameter
func (ok *Okx) GetGridAIParameter(ctx context.Context, algoOrderType, instrumentID, direction, duration string) ([]GridAIParameterResponse, error) {
	if algoOrderType != "moon_grid" && algoOrderType != "contract_grid" && algoOrderType != "grid" {
		return nil, errInvalidAlgoOrderType
	}
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	if algoOrderType == "contract_grid" && direction != positionSideLong && direction != positionSideShort && direction != "neutral" {
		return nil, fmt.Errorf("%w, required for 'contract_grid' algo order type", errMissingRequiredArgumentDirection)
	}
	params := url.Values{}
	params.Set("direction", direction)
	params.Set("algoOrdType", algoOrderType)
	params.Set("instId", instrumentID)
	if duration != "" && duration != "7D" && duration != "30D" && duration != "180D" {
		return nil, errInvalidDuration
	}
	var resp []GridAIParameterResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getGridAIParameterEPL, http.MethodGet, common.EncodeURLValues("tradingBot/grid/ai-param", params), nil, &resp, false)
}

// ComputeMinInvestment computes minimum investment
func (ok *Okx) ComputeMinInvestment(ctx context.Context, arg *ComputeInvestmentDataParam) (*InvestmentResult, error) {
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	switch arg.AlgoOrderType {
	case "grid", "contract_grid":
	default:
		return nil, errInvalidAlgoOrderType
	}
	if arg.MaxPrice <= 0 {
		return nil, fmt.Errorf("%w, maxPrice = %f", order.ErrPriceBelowMin, arg.MaxPrice)
	}
	if arg.MinPrice < 0 {
		return nil, fmt.Errorf("%w, minPrice = %f", order.ErrPriceBelowMin, arg.MaxPrice)
	}
	if arg.GridNumber == 0 {
		return nil, errors.New("grid number is required")
	}
	if arg.RunType == "" {
		return nil, errors.New("runType is required; possible values are 1: Arithmetic, 2: Geometric")
	}
	if arg.AlgoOrderType == "contract_grid" {
		switch arg.Direction {
		case "long", "short", "neutral":
		default:
			return nil, fmt.Errorf("%w, invalid grid direction %s", errMissingRequiredArgumentDirection, arg.Direction)
		}
		if arg.Leverage <= 0 {
			return nil, errRequiredParameterMissingLeverage
		}
	}
	for x := range arg.InvestmentData {
		if arg.InvestmentData[x].Amount <= 0 {
			return nil, fmt.Errorf("%w, investment amt = %f", order.ErrAmountBelowMin, arg.InvestmentData[x].Amount)
		}
		if arg.InvestmentData[x].Currency.IsEmpty() {
			return nil, currency.ErrCurrencyCodeEmpty
		}
	}
	var resp *InvestmentResult
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, computeMinInvestmentEPL, http.MethodPost, "tradingBot/grid/min-investment", arg, &resp, false)
}

// RSIBackTesting relative strength index(RSI) backtesting
// Parameters:
//
//	TriggerCondition: possible values are "cross_up" "cross_down" "above" "below" "cross" Default is cross_down
//
// Threshold: The value should be an integer between 1 to 100
func (ok *Okx) RSIBackTesting(ctx context.Context, instrumentID, triggerCondition, duration string, threshold, timePeriod int64, timeFrame kline.Interval) (*RSIBacktestingResponse, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	if threshold > 100 || threshold < 1 {
		return nil, errors.New("threshold should be an integer between 1 to 100")
	}
	timeFrameString := ok.GetIntervalEnum(timeFrame, false)
	if timeFrameString == "" {
		return nil, errors.New("timeframe is required")
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	params.Set("timeframe", timeFrameString)
	params.Set("thold", strconv.FormatInt(threshold, 10))
	if timePeriod > 0 {
		params.Set("timePeriod", strconv.FormatInt(timePeriod, 10))
	}
	if triggerCondition != "" {
		params.Set("triggerCond", triggerCondition)
	}
	if duration != "" {
		params.Set("duration", duration)
	}
	var resp *RSIBacktestingResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, rsiBackTestingEPL, http.MethodGet, common.EncodeURLValues("tradingBot/public/rsi-back-testing", params), nil, &resp, false)
}

// ****************************************** Signal bot trading **************************************************

// GetSignalBotOrderDetail create and customize your own signals while gaining access to a diverse selection of signals from top providers.
// Empower your trading strategies and stay ahead of the game with our comprehensive signal trading platform.
func (ok *Okx) GetSignalBotOrderDetail(ctx context.Context, algoOrderType, algoID string) (*SignalBotOrderDetail, error) {
	if algoOrderType == "" {
		return nil, errInvalidAlgoOrderType
	}
	if algoID == "" {
		return nil, fmt.Errorf("%w, algoId is required", order.ErrOrderIDNotSet)
	}
	params := url.Values{}
	params.Set("algoId", algoID)
	params.Set("algoOrdType", algoOrderType)
	var resp *SignalBotOrderDetail
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, signalBotOrderDetailsEPL, http.MethodGet, common.EncodeURLValues("tradingBot/signal/orders-algo-details", params), nil, &resp, true)
}

// GetSignalOrderPositions retrieves signal bot order positions
func (ok *Okx) GetSignalOrderPositions(ctx context.Context, algoOrderType, algoID string) (*SignalBotPosition, error) {
	if algoOrderType == "" {
		return nil, errInvalidAlgoOrderType
	}
	if algoID == "" {
		return nil, fmt.Errorf("%w, algoId is required", order.ErrOrderIDNotSet)
	}
	params := url.Values{}
	params.Set("algoId", algoID)
	params.Set("algoOrdType", algoOrderType)
	var resp *SignalBotPosition
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, signalBotOrderPositionsEPL, http.MethodGet, common.EncodeURLValues("tradingBot/signal/positions", params), nil, &resp, true)
}

// GetSignalBotSubOrders retrieves historical filled sub orders and designated sub orders
func (ok *Okx) GetSignalBotSubOrders(ctx context.Context, algoID, algoOrderType, subOrderType, clientOrderID, afterPaginationID, beforePaginationID string, begin, end time.Time, limit int64) ([]SubOrder, error) {
	if algoID == "" {
		return nil, fmt.Errorf("%w, algoId is required", order.ErrOrderIDNotSet)
	}
	if algoOrderType == "" {
		return nil, errInvalidAlgoOrderType
	}
	if subOrderType == "" && clientOrderID == "" {
		return nil, errors.New("either client order ID or sub-order state is required")
	}
	params := url.Values{}
	params.Set("algoId", algoID)
	params.Set("algoOrdType", algoOrderType)
	if subOrderType != "" {
		params.Set("type", subOrderType)
	} else {
		params.Set("clOrdId", clientOrderID)
	}
	if afterPaginationID != "" {
		params.Set("after", afterPaginationID)
	}
	if beforePaginationID != "" {
		params.Set("before", beforePaginationID)
	}
	if !begin.IsZero() {
		params.Set("begin", strconv.FormatInt(begin.UnixMilli(), 10))
	}
	if !end.IsZero() {
		params.Set("end", strconv.FormatInt(end.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []SubOrder
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, signalBotSubOrdersEPL, http.MethodGet, common.EncodeURLValues("tradingBot/signal/sub-orders", params), nil, &resp, true)
}

// GetSignalBotEventHistory retrieves signal bot event history
func (ok *Okx) GetSignalBotEventHistory(ctx context.Context, algoID string, after, before time.Time, limit int64) ([]SignalBotEventHistory, error) {
	if algoID == "" {
		return nil, fmt.Errorf("%w, algoId is required", order.ErrOrderIDNotSet)
	}
	params := url.Values{}
	params.Set("algoId", algoID)
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []SignalBotEventHistory
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, signalBotEventHistoryEPL, http.MethodGet, common.EncodeURLValues("tradingBot/signal/event-history", params), nil, &resp, true)
}

// ****************************************** Recurring Buy *****************************************

// PlaceRecurringBuyOrder recurring buy is a strategy for investing a fixed amount in crypto at fixed intervals.
// An appropriate recurring approach in volatile markets allows you to buy crypto at lower costs. Learn more
// The API endpoints of Recurring buy require authentication.
func (ok *Okx) PlaceRecurringBuyOrder(ctx context.Context, arg *PlaceRecurringBuyOrderParam) (*RecurringOrderResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.StrategyName == "" {
		return nil, errStrategyNameRequired
	}
	if len(arg.RecurringList) == 0 {
		return nil, errors.New("no recurring list is provided")
	}
	for x := range arg.RecurringList {
		if arg.RecurringList[x].Currency.IsEmpty() {
			return nil, currency.ErrCurrencyCodeEmpty
		}
	}
	if arg.RecurringDay == "" {
		return nil, errors.New("recurring day is required")
	}
	if arg.RecurringTime > 23 || arg.RecurringTime < 0 {
		return nil, errors.New("recurring buy time, the value range is an integer with value between 0 and 23")
	}
	if arg.TradeMode == "" {
		return nil, errInvalidTradeModeValue
	}
	var resp *RecurringOrderResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, placeRecurringBuyOrderEPL, http.MethodPost, "tradingBot/recurring/order-algo", arg, &resp, true)
}

// AmendRecurringBuyOrder amends recurring order
func (ok *Okx) AmendRecurringBuyOrder(ctx context.Context, arg *AmendRecurringOrderParam) (*RecurringOrderResponse, error) {
	if arg == nil || (*arg) == (AmendRecurringOrderParam{}) {
		return nil, common.ErrNilPointer
	}
	if arg.AlgoID == "" {
		return nil, fmt.Errorf("%w, algoId is required", order.ErrOrderIDNotSet)
	}
	if arg.StrategyName == "" {
		return nil, errStrategyNameRequired
	}
	var resp *RecurringOrderResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, amendRecurringBuyOrderEPL, http.MethodPost, "tradingBot/recurring/amend-order-algo", arg, &resp, true)
}

// StopRecurringBuyOrder stops recurring buy order. A maximum of 10 orders can be stopped per request.
func (ok *Okx) StopRecurringBuyOrder(ctx context.Context, arg []StopRecurringBuyOrder) ([]RecurringOrderResponse, error) {
	if len(arg) == 0 {
		return nil, common.ErrNilPointer
	}
	for x := range arg {
		if arg[x].AlgoID == "" {
			return nil, fmt.Errorf("%w, algoId is required", order.ErrOrderIDNotSet)
		}
	}
	var resp []RecurringOrderResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, stopRecurringBuyOrderEPL, http.MethodGet, "tradingBot/recurring/stop-order-algo", arg, &resp, true)
}

// GetRecurringBuyOrderList retrieves recurring buy order list.
func (ok *Okx) GetRecurringBuyOrderList(ctx context.Context, algoID, algoOrderState string, after, before time.Time, limit int64) ([]RecurringOrderItem, error) {
	params := url.Values{}
	if algoOrderState != "" {
		params.Set("state", algoOrderState)
	}
	if algoID != "" {
		params.Set("algoId", algoID)
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []RecurringOrderItem
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getRecurringBuyOrderListEPL, http.MethodGet, common.EncodeURLValues("tradingBot/recurring/orders-algo-pending", params), nil, &resp, true)
}

// GetRecurringBuyOrderHistory retrieves recurring buy order history.
func (ok *Okx) GetRecurringBuyOrderHistory(ctx context.Context, algoID string, after, before time.Time, limit int64) ([]RecurringOrderItem, error) {
	params := url.Values{}
	if algoID != "" {
		params.Set("algoId", algoID)
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []RecurringOrderItem
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getRecurringBuyOrderHistoryEPL, http.MethodGet, common.EncodeURLValues("tradingBot/recurring/orders-algo-history", params), nil, &resp, true)
}

// GetRecurringOrderDetails retrieves a single recurring order detail.
func (ok *Okx) GetRecurringOrderDetails(ctx context.Context, algoID, algoOrderState string) (*RecurringOrderDeail, error) {
	if algoID == "" {
		return nil, fmt.Errorf("%w, algoId is required", order.ErrOrderIDNotSet)
	}
	params := url.Values{}
	params.Set("algoId", algoID)
	if algoOrderState != "" {
		params.Set("state", algoOrderState)
	}
	var resp *RecurringOrderDeail
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getRecurringBuyOrderDetailEPL, http.MethodGet, common.EncodeURLValues("tradingBot/recurring/orders-algo-details", params), nil, &resp, true)
}

// GetRecurringSubOrders retrieves recurring buy sub orders.
func (ok *Okx) GetRecurringSubOrders(ctx context.Context, algoID, orderID string, after, before time.Time, limit int64) ([]RecurringBuySubOrder, error) {
	if algoID == "" {
		return nil, common.ErrNilPointer
	}
	params := url.Values{}
	params.Set("algoId", algoID)
	if orderID != "" {
		params.Set("ordId", orderID)
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []RecurringBuySubOrder
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getRecurringBuySubOrdersEPL, http.MethodGet, common.EncodeURLValues("tradingBot/recurring/sub-orders", params), nil, &resp, true)
}

// ****************************************** Earn **************************************************

// GetExistingLeadingPositions retrieves leading positions that are not closed.
func (ok *Okx) GetExistingLeadingPositions(ctx context.Context, instrumentType, instrumentID string, before, after time.Time, limit int64) ([]PositionInfo, error) {
	params := url.Values{}
	if instrumentType != "" {
		params.Set("instType", instrumentType)
	}
	if instrumentID != "" {
		params.Set("instId", instrumentID)
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []PositionInfo
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getExistingLeadingPositionsEPL, http.MethodGet, common.EncodeURLValues("copytrading/current-subpositions", params), nil, &resp, true)
}

// GetLeadingPositionsHistory leading trader retrieves the completed leading position of the last 3 months.
// Returns reverse chronological order with subPosId.
func (ok *Okx) GetLeadingPositionsHistory(ctx context.Context, instrumentType, instrumentID string, before, after time.Time, limit int64) ([]PositionInfo, error) {
	params := url.Values{}
	if instrumentType != "" {
		params.Set("instType", instrumentType)
	}
	if instrumentID != "" {
		params.Set("instId", instrumentID)
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []PositionInfo
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getLeadingPositionHistoryEPL, http.MethodGet, common.EncodeURLValues("copytrading/subpositions-history", params), nil, &resp, true)
}

// PlaceLeadingStopOrder holds leading trader sets TP/SL for the current leading position that are not closed.
func (ok *Okx) PlaceLeadingStopOrder(ctx context.Context, arg *TPSLOrderParam) (*PositionIDInfo, error) {
	if arg == nil || *arg == (TPSLOrderParam{}) {
		return nil, common.ErrNilPointer
	}
	if arg.SubPositionID == "" {
		return nil, errSubPositionIDRequired
	}
	var resp *PositionIDInfo
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, placeLeadingStopOrderEPL, http.MethodGet, "copytrading/algo-order", arg, &resp, true)
}

// CloseLeadingPosition close a leading position once a time.
func (ok *Okx) CloseLeadingPosition(ctx context.Context, arg *CloseLeadingPositionParam) (*PositionIDInfo, error) {
	if arg == nil || *arg == (CloseLeadingPositionParam{}) {
		return nil, common.ErrNilPointer
	}
	if arg.SubPositionID == "" {
		return nil, errSubPositionIDRequired
	}
	var resp *PositionIDInfo
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, closeLeadingPositionEPL, http.MethodPost, "copytrading/close-subposition", arg, &resp, true)
}

// GetLeadingInstrument retrieves leading instruments
func (ok *Okx) GetLeadingInstrument(ctx context.Context, instrumentType string) ([]LeadingInstrumentItem, error) {
	params := url.Values{}
	if instrumentType == "" {
		params.Set("instType", instrumentType)
	}
	var resp []LeadingInstrumentItem
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getLeadingInstrumentsEPL, http.MethodGet, common.EncodeURLValues("copytrading/instruments", params), nil, &resp, true)
}

// AmendLeadingInstruments amend current leading instruments, need to set initial leading instruments while applying to become a leading trader.
// All non-leading contracts can't have position or pending orders for the current request when setting non-leading contracts as leading contracts.
func (ok *Okx) AmendLeadingInstruments(ctx context.Context, instrumentID, instrumentType string) ([]LeadingInstrumentItem, error) {
	if instrumentID == "" {
		return nil, errInstrumentTypeRequired
	}
	var resp []LeadingInstrumentItem
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getLeadingInstrumentsEPL, http.MethodPost, "copytrading/set-instruments", &struct {
		InstrumentType string `json:"instType,omitempty"`
		InstrumentID   string `json:"instId"`
	}{
		InstrumentID:   instrumentID,
		InstrumentType: instrumentType,
	}, &resp, true)
}

// GetProfitSharingDetails gets profits shared details for the last 3 months.
func (ok *Okx) GetProfitSharingDetails(ctx context.Context, instrumentType string, before, after time.Time, limit int64) ([]ProfitSharingItem, error) {
	params := url.Values{}
	if instrumentType != "" {
		params.Set("instType", instrumentType)
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []ProfitSharingItem
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getProfitSharingLimitEPL, http.MethodGet, common.EncodeURLValues("copytrading/profit-sharing-details", params), nil, &resp, true)
}

// GetTotalProfitSharing gets the total amount of profit shared since joining the platform.
// Instrument type 'SPOT' 'SWAP' It returns all types by default.
func (ok *Okx) GetTotalProfitSharing(ctx context.Context, instrumentType string) ([]TotalProfitSharing, error) {
	params := url.Values{}
	if instrumentType != "" {
		params.Set("instType", instrumentType)
	}
	var resp []TotalProfitSharing
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getTotalProfitSharingEPL, http.MethodGet, common.EncodeURLValues("copytrading/total-profit-sharing", params), nil, &resp, true)
}

// GetUnrealizedProfitSharingDetails gets leading trader gets the profit sharing details that are expected to be shared in the next settlement cycle.
// The unrealized profit sharing details will update once there copy position is closed.
func (ok *Okx) GetUnrealizedProfitSharingDetails(ctx context.Context, instrumentType string) ([]ProfitSharingItem, error) {
	params := url.Values{}
	if instrumentType != "" {
		params.Set("instType", instrumentType)
	}
	var resp []ProfitSharingItem
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getTotalProfitSharingEPL, http.MethodGet, common.EncodeURLValues("copytrading/unrealized-profit-sharing-details", params), nil, &resp, true)
}

// SetFirstCopySettings set first copy settings for the certain lead trader. You need to first copy settings after stopping copying.
func (ok *Okx) SetFirstCopySettings(ctx context.Context, arg *FirstCopySettings) (*ResponseSuccess, error) {
	err := validateFirstCopySettings(arg)
	if err != nil {
		return nil, err
	}
	var resp *ResponseSuccess
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, setFirstCopySettingsEPL, http.MethodPost, "copytrading/first-copy-settings", arg, &resp, true)
}

// AmendCopySettings amends need to use this endpoint for amending copy settings
func (ok *Okx) AmendCopySettings(ctx context.Context, arg *FirstCopySettings) (*ResponseSuccess, error) {
	err := validateFirstCopySettings(arg)
	if err != nil {
		return nil, err
	}
	var resp *ResponseSuccess
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, amendFirstCopySettingsEPL, http.MethodPost, "copytrading/amend-copy-settings", arg, &resp, true)
}

func validateFirstCopySettings(arg *FirstCopySettings) error {
	if arg == nil || *arg == (FirstCopySettings{}) {
		return common.ErrNilPointer
	}
	if arg.UniqueCode == "" {
		return errUniqueCodeRequired
	}
	if arg.CopyInstrumentIDType == "" {
		return errCopyInstrumentIDTypeRequired
	}
	if arg.CopyTotalAmount <= 0 {
		return fmt.Errorf("%w, copyTotalAmount value %f", order.ErrAmountBelowMin, arg.CopyTotalAmount)
	}
	if arg.SubPosCloseType == "" {
		return errSubPositionCloseTypeRequired
	}
	return nil
}

// StopCopying need to use this endpoint for amending copy settings
func (ok *Okx) StopCopying(ctx context.Context, arg *StopCopyingParameter) (*ResponseSuccess, error) {
	if arg == nil || *arg == (StopCopyingParameter{}) {
		return nil, common.ErrNilPointer
	}
	if arg.UniqueCode == "" {
		return nil, errUniqueCodeRequired
	}
	if arg.SubPositionCloseType == "" {
		return nil, errSubPositionCloseTypeRequired
	}
	var resp *ResponseSuccess
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, stopCopyingEPL, http.MethodPost, "copytrading/stop-copy-trading", arg, &resp, true)
}

// GetCopySettings retrieve the copy settings about certain lead trader.
func (ok *Okx) GetCopySettings(ctx context.Context, instrumentType, uniqueCode string) (*CopySetting, error) {
	if uniqueCode == "" {
		return nil, errUniqueCodeRequired
	}
	params := url.Values{}
	params.Set("uniqueCode", uniqueCode)
	if instrumentType == "" {
		params.Set("instType", instrumentType)
	}
	var resp *CopySetting
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getCopySettingsEPL, http.MethodGet, common.EncodeURLValues("copytrading/copy-settings", params), nil, &resp, true)
}

// GetMultipleLeverages retrieve leverages that belong to the lead trader and you.
func (ok *Okx) GetMultipleLeverages(ctx context.Context, marginMode, uniqueCode, instrumentID string) ([]Leverages, error) {
	if marginMode == "" {
		return nil, errMissingMarginMode
	}
	if uniqueCode == "" {
		return nil, errUniqueCodeRequired
	}
	params := url.Values{}
	params.Set("mgnMode", marginMode)
	params.Set("uniqueCode", uniqueCode)
	if instrumentID != "" {
		params.Set("instId", instrumentID)
	}
	var resp []Leverages
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getMultipleLeveragesEPL, http.MethodGet, common.EncodeURLValues("copytrading/batch-leverage-info", params), nil, &resp, true)
}

// SetMultipleLeverages set Multiple leverages
func (ok *Okx) SetMultipleLeverages(ctx context.Context, arg *SetLeveragesParam) (*SetMultipleLeverageResponse, error) {
	if arg == nil || *arg == (SetLeveragesParam{}) {
		return nil, common.ErrNilPointer
	}
	if arg.MarginMode == "" {
		return nil, errMissingMarginMode
	}
	if arg.Leverage < 0 {
		return nil, errRequiredParameterMissingLeverage
	}
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	var resp *SetMultipleLeverageResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, setBatchLeverageEPL, http.MethodPost, "copytrading/batch-set-leverage", arg, &resp, true)
}

// GetMyLeadTraders retrieve my lead traders.
func (ok *Okx) GetMyLeadTraders(ctx context.Context, instrumentType string) ([]CopyTradingLeadTrader, error) {
	params := url.Values{}
	if instrumentType != "" {
		params.Set("instType", instrumentType)
	}
	var resp []CopyTradingLeadTrader
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getMyLeadTradersEPL, http.MethodGet, common.EncodeURLValues("copytrading/current-lead-traders", params), nil, &resp, true)
}

// GetHistoryLeadTraders retrieve my history lead traders.
func (ok *Okx) GetHistoryLeadTraders(ctx context.Context, instrumentType, after, before string, limit int64) ([]CopyTradingLeadTrader, error) {
	params := url.Values{}
	if instrumentType != "" {
		params.Set("instType", instrumentType)
	}
	if after != "" {
		params.Set("after", after)
	}
	if before != "" {
		params.Set("before", before)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []CopyTradingLeadTrader
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getMyLeadTradersEPL, http.MethodGet, common.EncodeURLValues("copytrading/lead-traders-history", params), nil, &resp, true)
}

// GetLeadTradersRanks retrieve lead trader ranks.
// Instrument type: SWAP, the default value
// Sort type"overview": overview, the default value "pnl": profit and loss "aum": assets under management "win_ratio": win ratio "pnl_ratio": pnl ratio "current_copy_trader_pnl": current copy trader pnl
// Lead trader state: "0": All lead traders, the default, including vacancy and non-vacancy "1": lead traders who have vacancy
// Minimum lead days '1': 7 days '2': 30 days '3': 90 days '4': 180 days
func (ok *Okx) GetLeadTradersRanks(ctx context.Context, instrumentType, sortType, state,
	minLeadDays, minAssets, maxAssets, minAssetUnderManagement, maxAssetUnderManagement,
	dataVersion, page string, limit int64) ([]LeadTradersRank, error) {
	params := url.Values{}
	if instrumentType != "" {
		params.Set("instType", instrumentType)
	}
	if sortType != "" {
		params.Set("sortType", sortType)
	}
	if state != "" {
		params.Set("state", state)
	}
	if minLeadDays != "" {
		params.Set("minLeadDays", minLeadDays)
	}
	if minAssets != "" {
		params.Set("minAssets", minAssets)
	}
	if maxAssets != "" {
		params.Set("maxAssets", maxAssets)
	}
	if minAssetUnderManagement != "" {
		params.Set("minAum", minAssetUnderManagement)
	}
	if maxAssetUnderManagement != "" {
		params.Set("maxAum", maxAssetUnderManagement)
	}
	if dataVersion != "" {
		params.Set("dataVer", dataVersion)
	}
	if page != "" {
		params.Set("page", page)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []LeadTradersRank
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getLeadTraderRanksEPL, http.MethodGet, common.EncodeURLValues("copytrading/public-lead-traders", params), nil, &resp, false)
}

// GetWeeklyTraderProfitAndLoss retrieve lead trader weekly pnl. Results are returned in counter chronological order.
func (ok *Okx) GetWeeklyTraderProfitAndLoss(ctx context.Context, instrumentType, uniqueCode string) ([]TraderWeeklyProfitAndLoss, error) {
	if uniqueCode == "" {
		return nil, errUniqueCodeRequired
	}
	params := url.Values{}
	params.Set("uniqueCode", uniqueCode)
	if instrumentType != "" {
		params.Set("instType", instrumentType)
	}
	var resp []TraderWeeklyProfitAndLoss
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getLeadTraderWeeklyPNLEPL, http.MethodGet, common.EncodeURLValues("copytrading/public-weekly-pnl", params), nil, &resp, false)
}

// GetDailyLeadTraderPNL retrieve lead trader daily pnl. Results are returned in counter chronological order.
// Last days "1": last 7 days  "2": last 30 days "3": last 90 days  "4": last 365 days
func (ok *Okx) GetDailyLeadTraderPNL(ctx context.Context, instrumentType, uniqueCode, lastDays string) ([]TraderWeeklyProfitAndLoss, error) {
	if uniqueCode == "" {
		return nil, errUniqueCodeRequired
	}
	if lastDays == "" {
		return nil, errLastDaysRequired
	}
	params := url.Values{}
	params.Set("uniqueCode", uniqueCode)
	params.Set("lastDays", lastDays)
	if instrumentType != "" {
		params.Set("instType", instrumentType)
	}
	var resp []TraderWeeklyProfitAndLoss
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getLeadTraderDailyPNLEPL, http.MethodGet, common.EncodeURLValues("copytrading/public-weekly-pnl", params), nil, &resp, false)
}

// GetLeadTraderStats retrieves key data related to lead trader performance.
func (ok *Okx) GetLeadTraderStats(ctx context.Context, instrumentType, uniqueCode, lastDays string) ([]LeadTraderStat, error) {
	if uniqueCode == "" {
		return nil, errUniqueCodeRequired
	}
	if lastDays == "" {
		return nil, errLastDaysRequired
	}
	params := url.Values{}
	params.Set("uniqueCode", uniqueCode)
	params.Set("lastDays", lastDays)
	if instrumentType != "" {
		params.Set("instType", instrumentType)
	}
	var resp []LeadTraderStat
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getLeadTraderStatsEPL, http.MethodGet, common.EncodeURLValues("copytrading/public-stats", params), nil, &resp, false)
}

// GetLeadTraderCurrencyPreferences retrieves the most frequently traded crypto of this lead trader. Results are sorted by ratio from large to small.
func (ok *Okx) GetLeadTraderCurrencyPreferences(ctx context.Context, instrumentType, uniqueCode, lastDays string) ([]LeadTraderCurrencyPreference, error) {
	if uniqueCode == "" {
		return nil, errUniqueCodeRequired
	}
	if lastDays == "" {
		return nil, errLastDaysRequired
	}
	params := url.Values{}
	params.Set("uniqueCode", uniqueCode)
	params.Set("lastDays", lastDays)
	if instrumentType != "" {
		params.Set("instType", instrumentType)
	}
	var resp []LeadTraderCurrencyPreference
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getLeadTraderCurrencyPreferencesEPL, http.MethodGet, common.EncodeURLValues("copytrading/public-preference-currency", params), nil, &resp, false)
}

// GetLeadTraderCurrentLeadPositions get current leading positions of lead trader
// Instrument type "SPOT" "SWAP"
func (ok *Okx) GetLeadTraderCurrentLeadPositions(ctx context.Context, instrumentType, uniqueCode, afterSubPositionID,
	beforeSubPositionID string, limit int64) ([]LeadTraderCurrentLeadPosition, error) {
	if instrumentType != "SWAP" {
		return nil, fmt.Errorf("%w asset: %s", asset.ErrNotSupported, instrumentType)
	}
	if uniqueCode == "" {
		return nil, errUniqueCodeRequired
	}
	params := url.Values{}
	params.Set("uniqueCode", uniqueCode)
	if instrumentType != "" {
		params.Set("instType", instrumentType)
	}
	if afterSubPositionID != "" {
		params.Set("after", afterSubPositionID)
	}
	if beforeSubPositionID != "" {
		params.Set("before", beforeSubPositionID)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []LeadTraderCurrentLeadPosition
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getTraderCurrentLeadPositionsEPL, http.MethodGet, common.EncodeURLValues("copytrading/public-current-subpositions", params), nil, &resp, false)
}

// GetLeadTraderLeadPositionHistory retrieve the lead trader completed leading position of the last 3 months. Returns reverse chronological order with subPosId.
func (ok *Okx) GetLeadTraderLeadPositionHistory(ctx context.Context, instrumentType, uniqueCode, afterSubPositionID, beforeSubPositionID string, limit int64) ([]LeadPosition, error) {
	if instrumentType != "SWAP" {
		return nil, fmt.Errorf("%w asset: %s", asset.ErrNotSupported, instrumentType)
	}
	if uniqueCode == "" {
		return nil, errUniqueCodeRequired
	}
	params := url.Values{}
	params.Set("uniqueCode", uniqueCode)
	if instrumentType != "" {
		params.Set("instType", instrumentType)
	}
	if afterSubPositionID != "" {
		params.Set("after", afterSubPositionID)
	}
	if beforeSubPositionID != "" {
		params.Set("before", beforeSubPositionID)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []LeadPosition
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getLeadTraderLeadPositionHistoryEPL, http.MethodGet, common.EncodeURLValues("copytrading/public-subpositions-history", params), nil, &resp, false)
}

// ****************************************** Earn **************************************************

// GetOffers retrieves list of offers for different protocols.
func (ok *Okx) GetOffers(ctx context.Context, productID, protocolType string, ccy currency.Code) ([]Offer, error) {
	params := url.Values{}
	if productID != "" {
		params.Set("productId", productID)
	}
	if protocolType != "" {
		params.Set("protocolType", protocolType)
	}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	var resp []Offer
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getOfferEPL, http.MethodGet, common.EncodeURLValues("finance/staking-defi/offers", params), nil, &resp, true)
}

// Purchase invest on specific product
func (ok *Okx) Purchase(ctx context.Context, arg PurchaseRequestParam) (*OrderIDResponse, error) {
	if arg.ProductID == "" {
		return nil, fmt.Errorf("%w, missing product id", errMissingRequiredParameter)
	}
	for x := range arg.InvestData {
		if arg.InvestData[x].Currency.IsEmpty() {
			return nil, fmt.Errorf("%w, currency information for investment is required", errMissingRequiredParameter)
		}
		if arg.InvestData[x].Amount <= 0 {
			return nil, errUnacceptableAmount
		}
	}
	var resp *OrderIDResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, purchaseEPL, http.MethodPost, "finance/staking-defi/purchase", &arg, &resp, true)
}

// Redeem redemption of investment
func (ok *Okx) Redeem(ctx context.Context, arg RedeemRequestParam) (*OrderIDResponse, error) {
	if arg.OrderID == "" {
		return nil, fmt.Errorf("%w, missing 'orderId'", errMissingRequiredParameter)
	}
	if arg.ProtocolType != "staking" && arg.ProtocolType != "defi" {
		return nil, errInvalidProtocolType
	}
	var resp *OrderIDResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, redeemEPL, http.MethodPost, "finance/staking-defi/redeem", &arg, &resp, true)
}

// CancelPurchaseOrRedemption cancels Purchases or Redemptions
// after cancelling, returning funds will go to the funding account.
func (ok *Okx) CancelPurchaseOrRedemption(ctx context.Context, arg CancelFundingParam) (*OrderIDResponse, error) {
	if arg.OrderID == "" {
		return nil, fmt.Errorf("%w, missing 'orderId'", errMissingRequiredParameter)
	}
	if arg.ProtocolType != "staking" && arg.ProtocolType != "defi" {
		return nil, errInvalidProtocolType
	}
	var resp *OrderIDResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, cancelPurchaseOrRedemptionEPL, http.MethodPost, "finance/staking-defi/cancel", &arg, &resp, true)
}

// GetEarnActiveOrders retrieves active orders.
func (ok *Okx) GetEarnActiveOrders(ctx context.Context, productID, protocolType, state string, ccy currency.Code) ([]ActiveFundingOrder, error) {
	params := url.Values{}
	if productID != "" {
		params.Set("productId", productID)
	}

	if protocolType != "" {
		// protocol type 'staking' and 'defi' is allowed by default
		if protocolType != "staking" && protocolType != "defi" {
			return nil, errInvalidProtocolType
		}
		params.Set("protocolType", protocolType)
	}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if state != "" {
		params.Set("state", state)
	}
	var resp []ActiveFundingOrder
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getEarnActiveOrdersEPL, http.MethodGet, common.EncodeURLValues("finance/staking-defi/orders-active", params), nil, &resp, true)
}

// GetFundingOrderHistory retrieves funding order history
// valid protocol types are 'staking' and 'defi'
func (ok *Okx) GetFundingOrderHistory(ctx context.Context, productID, protocolType string, ccy currency.Code, after, before time.Time, limit int64) ([]ActiveFundingOrder, error) {
	params := url.Values{}
	if productID != "" {
		params.Set("productId", productID)
	}
	if protocolType != "" {
		params.Set("protocolType", protocolType)
	}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []ActiveFundingOrder
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getFundingOrderHistoryEPL, http.MethodGet, common.EncodeURLValues("finance/staking-defi/orders-history", params), nil, &resp, true)
}

// **************************************************************** ETH Staking ****************************************************************

// PurcahseETHStaking staking ETH for BETH
// Only the assets in the funding account can be used.
func (ok *Okx) PurcahseETHStaking(ctx context.Context, amount float64) error {
	if amount <= 0 {
		return order.ErrAmountBelowMin
	}
	var resp []string
	return ok.SendHTTPRequest(ctx, exchange.RestSpot, purchaseETHStakingEPL, http.MethodPost, "finance/staking-defi/eth/purchase", map[string]string{"amt": strconv.FormatFloat(amount, 'f', -1, 64)}, &resp, true)
}

// RedeemETHStaking only the assets in the funding account can be used. If your BETH is in your trading account, you can make funding transfer first.
func (ok *Okx) RedeemETHStaking(ctx context.Context, amount float64) error {
	if amount <= 0 {
		return order.ErrAmountBelowMin
	}
	var resp []string
	return ok.SendHTTPRequest(ctx, exchange.RestSpot, redeemETHStakingEPL, http.MethodPost, "finance/staking-defi/eth/redeem", map[string]string{"amt": strconv.FormatFloat(amount, 'f', -1, 64)}, &resp, true)
}

// GetBETHAssetsBalance balance is a snapshot summarized all BETH assets in trading and funding accounts. Also, the snapshot updates hourly.
func (ok *Okx) GetBETHAssetsBalance(ctx context.Context) (*BETHAssetsBalance, error) {
	var resp *BETHAssetsBalance
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getBETHBalanceEPL, http.MethodGet, "finance/staking-defi/eth/balance", nil, &resp, true)
}

// GetPurchaseAndRedeemHistory retrieves purchase and redeem history
// kind possible values are 'purchase' and 'redeem'
// Status 'pending' 'success' 'failed'
func (ok *Okx) GetPurchaseAndRedeemHistory(ctx context.Context, kind, status string, after, before time.Time, limit int64) ([]PurchaseRedeemHistory, error) {
	if kind == "" {
		return nil, errors.New("kind is required: possible values are 'purchase' and 'redeem'")
	}
	params := url.Values{}
	params.Set("type", kind)
	if status != "" {
		params.Set("status", status)
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []PurchaseRedeemHistory
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getPurchaseRedeemHistoryEPL, http.MethodGet, common.EncodeURLValues("finance/staking-defi/eth/purchase-redeem-history", params), nil, &resp, true)
}

// GetAPYHistory retrieves Annual percentage yield(APY) history
func (ok *Okx) GetAPYHistory(ctx context.Context, days int64) ([]APYItem, error) {
	if days == 0 || days > 365 {
		return nil, errors.New("field days is required; possible values from 1 to 365")
	}
	var resp []APYItem
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getAPYHistoryEPL, http.MethodGet, fmt.Sprintf("finance/staking-defi/eth/apy-history?days=%d", days), nil, &resp, false)
}

// GetTickers retrieves the latest price snapshots best bid/ ask price, and trading volume in the last 24 hours.
func (ok *Okx) GetTickers(ctx context.Context, instType, uly, instID string) ([]TickerResponse, error) {
	params := url.Values{}
	instType = strings.ToUpper(instType)
	switch {
	case instType != "":
		params.Set("instType", instType)
		if (instType == okxInstTypeSwap || instType == okxInstTypeFutures || instType == okxInstTypeOption) && uly != "" {
			params.Set("uly", uly)
		}
	case instID != "":
		params.Set("instId", instID)
	default:
		return nil, errInstrumentTypeRequired
	}
	var response []TickerResponse
	return response, ok.SendHTTPRequest(ctx, exchange.RestSpot, getTickersEPL, http.MethodGet, common.EncodeURLValues("market/tickers", params), nil, &response, false)
}

// GetTicker retrieves the latest price snapshot, best bid/ask price, and trading volume in the last 24 hours.
func (ok *Okx) GetTicker(ctx context.Context, instrumentID string) (*TickerResponse, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	var resp *TickerResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getTickerEPL, http.MethodGet, common.EncodeURLValues("market/ticker", params), nil, &resp, false)
}

// GetIndexTickers Retrieves index tickers.
func (ok *Okx) GetIndexTickers(ctx context.Context, quoteCurrency currency.Code, instID string) ([]IndexTicker, error) {
	if instID == "" && quoteCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w, quoteCcy or instId is required", errEitherInstIDOrCcyIsRequired)
	}
	params := url.Values{}
	if !quoteCurrency.IsEmpty() {
		params.Set("quoteCcy", quoteCurrency.String())
	} else if instID != "" {
		params.Set("instId", instID)
	}
	response := []IndexTicker{}
	return response, ok.SendHTTPRequest(ctx, exchange.RestSpot, getIndexTickersEPL, http.MethodGet, common.EncodeURLValues("market/index-tickers", params), nil, &response, false)
}

// GetInstrumentTypeFromAssetItem returns a string representation of asset.Item; which is an equivalent term for InstrumentType in Okx exchange.
func (ok *Okx) GetInstrumentTypeFromAssetItem(a asset.Item) string {
	switch a {
	case asset.PerpetualSwap:
		return okxInstTypeSwap
	case asset.Options:
		return okxInstTypeOption
	default:
		return strings.ToUpper(a.String())
	}
}

// GetUnderlying returns the instrument ID for the corresponding asset pairs and asset type( Instrument Type )
func (ok *Okx) GetUnderlying(pair currency.Pair, a asset.Item) (string, error) {
	if !pair.IsPopulated() {
		return "", currency.ErrCurrencyPairsEmpty
	}
	format, err := ok.GetPairFormat(a, false)
	if err != nil {
		return "", err
	}
	return pair.Base.String() + format.Delimiter + pair.Quote.String(), nil
}

// GetPairFromInstrumentID returns a currency pair give an instrument ID and asset Item, which represents the instrument type.
func (ok *Okx) GetPairFromInstrumentID(instrumentID string) (currency.Pair, error) {
	codes := strings.Split(instrumentID, currency.DashDelimiter)
	if len(codes) >= 2 {
		instrumentID = codes[0] + currency.DashDelimiter + strings.Join(codes[1:], currency.DashDelimiter)
	}
	return currency.NewPairFromString(instrumentID)
}

// GetOrderBookDepth returns the recent order asks and bids before specified timestamp.
func (ok *Okx) GetOrderBookDepth(ctx context.Context, instrumentID string, depth int64) (*OrderBookResponse, error) {
	params := url.Values{}
	params.Set("instId", instrumentID)
	if depth > 0 {
		params.Set("sz", strconv.FormatInt(depth, 10))
	}
	var resp *OrderBookResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getOrderBookEPL, http.MethodGet, common.EncodeURLValues("market/books", params), nil, &resp, false)
}

// GetIntervalEnum allowed interval params by Okx Exchange
func (ok *Okx) GetIntervalEnum(interval kline.Interval, appendUTC bool) string {
	switch interval {
	case kline.OneMin:
		return "1m"
	case kline.ThreeMin:
		return "3m"
	case kline.FiveMin:
		return "5m"
	case kline.FifteenMin:
		return "15m"
	case kline.ThirtyMin:
		return "30m"
	case kline.OneHour:
		return "1H"
	case kline.TwoHour:
		return "2H"
	case kline.FourHour:
		return "4H"
	}

	duration := ""
	switch interval {
	case kline.SixHour: // NOTE: Cases here and below can either be local Hong Kong time or UTC time.
		duration = "6H"
	case kline.TwelveHour:
		duration = "12H"
	case kline.OneDay:
		duration = "1D"
	case kline.TwoDay:
		duration = "2D"
	case kline.ThreeDay:
		duration = "3D"
	case kline.FiveDay:
		duration = "5D"
	case kline.OneWeek:
		duration = "1W"
	case kline.OneMonth:
		duration = "1M"
	case kline.ThreeMonth:
		duration = "3M"
	case kline.SixMonth:
		duration = "6M"
	case kline.OneYear:
		duration = "1Y"
	default:
		return duration
	}

	if appendUTC {
		duration += "utc"
	}

	return duration
}

// GetCandlesticks Retrieve the candlestick charts. This endpoint can retrieve the latest 1,440 data entries. Charts are returned in groups based on the requested bar.
func (ok *Okx) GetCandlesticks(ctx context.Context, instrumentID string, interval kline.Interval, before, after time.Time, limit int64) ([]CandleStick, error) {
	return ok.GetCandlestickData(ctx, instrumentID, interval, before, after, limit, "market/candles", getCandlesticksEPL)
}

// GetCandlesticksHistory Retrieve history candlestick charts from recent years.
func (ok *Okx) GetCandlesticksHistory(ctx context.Context, instrumentID string, interval kline.Interval, before, after time.Time, limit int64) ([]CandleStick, error) {
	return ok.GetCandlestickData(ctx, instrumentID, interval, before, after, limit, "market/history-candles", getCandlestickHistoryEPL)
}

// GetIndexCandlesticks Retrieve the candlestick charts of the index. This endpoint can retrieve the latest 1,440 data entries. Charts are returned in groups based on the requested bar.
// the response is a list of Candlestick data.
func (ok *Okx) GetIndexCandlesticks(ctx context.Context, instrumentID string, interval kline.Interval, before, after time.Time, limit int64) ([]CandleStick, error) {
	return ok.GetCandlestickData(ctx, instrumentID, interval, before, after, limit, "market/index-candles", getIndexCandlesticksEPL)
}

// GetMarkPriceCandlesticks Retrieve the candlestick charts of mark price. This endpoint can retrieve the latest 1,440 data entries. Charts are returned in groups based on the requested bar.
func (ok *Okx) GetMarkPriceCandlesticks(ctx context.Context, instrumentID string, interval kline.Interval, before, after time.Time, limit int64) ([]CandleStick, error) {
	return ok.GetCandlestickData(ctx, instrumentID, interval, before, after, limit, "market/mark-price-candles", getCandlestickHistoryEPL)
}

// GetHistoricIndexCandlesticksHistory retrieve the candlestick charts of the index from recent years.
func (ok *Okx) GetHistoricIndexCandlesticksHistory(ctx context.Context, instrumentID string, after, before time.Time, bar kline.Interval, limit int64) ([]CandlestickHistoryItem, error) {
	return ok.getHistoricCandlesticks(ctx, instrumentID, "market/history-index-candles", after, before, bar, limit, getIndexCandlesticksHistoryEPL)
}

// GetMarkPriceCandlestickHistory retrieve the candlestick charts of the mark price from recent years.
func (ok *Okx) GetMarkPriceCandlestickHistory(ctx context.Context, instrumentID string, after, before time.Time, bar kline.Interval, limit int64) ([]CandlestickHistoryItem, error) {
	return ok.getHistoricCandlesticks(ctx, instrumentID, "market/history-mark-price-candles", after, before, bar, limit, getMarkPriceCandlesticksHistoryEPL)
}

func (ok *Okx) getHistoricCandlesticks(ctx context.Context, instrumentID, path string, after, before time.Time, bar kline.Interval, limit int64, epl request.EndpointLimit) ([]CandlestickHistoryItem, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	barString := ok.GetIntervalEnum(bar, false)
	if barString != "" {
		params.Set("bar", barString)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp IndexCandlestickSlices
	err := ok.SendHTTPRequest(ctx, exchange.RestSpot, epl, http.MethodGet, common.EncodeURLValues(path, params), nil, &resp, false)
	if err != nil {
		return nil, err
	}
	return resp.ExtractIndexCandlestick()
}

// GetEconomicCalendarData retrieves the macro-economic calendar data within 3 months. Historical data from 3 months ago is only available to users with trading fee tier VIP1 and above.
func (ok *Okx) GetEconomicCalendarData(ctx context.Context, region, importance string, before, after time.Time, limit int64) ([]EconomicCalendar, error) {
	params := url.Values{}
	if region != "" {
		params.Set("region", region)
	}
	if importance != "" {
		params.Set("importance", importance)
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []EconomicCalendar
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getEconomicCalendarEPL, http.MethodGet, "public/economic-calendar", nil, &resp, true)
}

// GetCandlestickData handles fetching the data for both the default GetCandlesticks, GetCandlesticksHistory, and GetIndexCandlesticks() methods.
func (ok *Okx) GetCandlestickData(ctx context.Context, instrumentID string, interval kline.Interval, before, after time.Time, limit int64, route string, rateLimit request.EndpointLimit) ([]CandleStick, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	params.Set("limit", strconv.FormatInt(limit, 10))
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	bar := ok.GetIntervalEnum(interval, true)
	if bar != "" {
		params.Set("bar", bar)
	}
	var resp [][7]types.Number
	err := ok.SendHTTPRequest(ctx, exchange.RestSpot, rateLimit, http.MethodGet, common.EncodeURLValues(route, params), nil, &resp, false)
	if err != nil {
		return nil, err
	}
	klineData := make([]CandleStick, len(resp))
	for x := range resp {
		klineData[x] = CandleStick{
			OpenTime:         time.UnixMilli(resp[x][0].Int64()),
			OpenPrice:        resp[x][1].Float64(),
			HighestPrice:     resp[x][2].Float64(),
			LowestPrice:      resp[x][3].Float64(),
			ClosePrice:       resp[x][4].Float64(),
			Volume:           resp[x][5].Float64(),
			QuoteAssetVolume: resp[x][6].Float64(),
		}
	}
	return klineData, nil
}

// GetTrades Retrieve the recent transactions of an instrument.
func (ok *Okx) GetTrades(ctx context.Context, instrumentID string, limit int64) ([]TradeResponse, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []TradeResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getTradesRequestEPL, http.MethodGet, common.EncodeURLValues("market/trades", params), nil, &resp, false)
}

// GetTradesHistory retrieves the recent transactions of an instrument from the last 3 months with pagination.
func (ok *Okx) GetTradesHistory(ctx context.Context, instrumentID, before, after string, limit int64) ([]TradeResponse, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	if before != "" {
		params.Set("before", before)
	}
	if after != "" {
		params.Set("after", after)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []TradeResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getTradesHistoryEPL, http.MethodGet, common.EncodeURLValues("market/history-trades", params), nil, &resp, false)
}

// GetOptionTradesByInstrumentFamily retrieve the recent transactions of an instrument under same instFamily. The maximum is 100.
func (ok *Okx) GetOptionTradesByInstrumentFamily(ctx context.Context, instrumentFamily string) ([]InstrumentFamilyTrade, error) {
	if instrumentFamily == "" {
		return nil, errInstrumentFamilyRequired
	}
	var resp []InstrumentFamilyTrade
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, optionInstrumentTradeFamilyEPL, http.MethodGet, "market/option/instrument-family-trades?instFamily="+instrumentFamily, nil, &resp, false)
}

// GetOptionTrades retrieves option trades
// Option type, 'C': Call 'P': put
func (ok *Okx) GetOptionTrades(ctx context.Context, instrumentID, instrumentFamily, optionType string) ([]OptionTrade, error) {
	if instrumentID == "" && instrumentFamily == "" {
		return nil, errInstrumentIDorFamilyRequired
	}
	params := url.Values{}
	if instrumentID != "" {
		params.Set("instId", instrumentID)
	}
	if instrumentFamily != "" {
		params.Set("instFamily", instrumentFamily)
	}
	if optionType != "" {
		params.Set("optType", optionType)
	}
	var resp []OptionTrade
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, optionTradesEPL, http.MethodGet, common.EncodeURLValues("public/option-trades", params), nil, &resp, false)
}

// Get24HTotalVolume The 24-hour trading volume is calculated on a rolling basis, using USD as the pricing unit.
func (ok *Okx) Get24HTotalVolume(ctx context.Context) (*TradingVolumeIn24HR, error) {
	var resp []TradingVolumeIn24HR
	err := ok.SendHTTPRequest(ctx, exchange.RestSpot, get24HTotalVolumeEPL, http.MethodGet, "market/platform-24-volume", nil, &resp, false)
	if err != nil {
		return nil, err
	}
	if len(resp) == 1 {
		return &resp[0], nil
	}
	return nil, errNo24HrTradeVolumeFound
}

// GetOracle Get the crypto price of signing using Open Oracle smart contract.
func (ok *Okx) GetOracle(ctx context.Context) (*OracleSmartContractResponse, error) {
	var resp []OracleSmartContractResponse
	err := ok.SendHTTPRequest(ctx, exchange.RestSpot, getOracleEPL, http.MethodGet, "market/open-oracle", nil, &resp, false)
	if err != nil {
		return nil, err
	}
	if len(resp) == 1 {
		return &resp[0], nil
	}
	return nil, errOracleInformationNotFound
}

// GetExchangeRate this interface provides the average exchange rate data for 2 weeks
// from USD to CNY
func (ok *Okx) GetExchangeRate(ctx context.Context) (*UsdCnyExchangeRate, error) {
	var resp []UsdCnyExchangeRate
	err := ok.SendHTTPRequest(ctx, exchange.RestSpot, getExchangeRateRequestEPL, http.MethodGet, "market/exchange-rate", nil, &resp, false)
	if err != nil {
		return nil, err
	}
	if len(resp) == 1 {
		return &resp[0], nil
	}
	return nil, errExchangeInfoNotFound
}

// GetIndexComponents returns the index component information data on the market
func (ok *Okx) GetIndexComponents(ctx context.Context, index string) (*IndexComponent, error) {
	params := url.Values{}
	params.Set("index", index)
	var resp *IndexComponent
	err := ok.SendHTTPRequest(ctx, exchange.RestSpot, getIndexComponentsEPL, http.MethodGet, common.EncodeURLValues("market/index-components", params), nil, &resp, false, true)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, errIndexComponentNotFound
	}
	return resp, nil
}

// GetBlockTickers retrieves the latest block trading volume in the last 24 hours.
// Instrument Type Is Mandatory, and Underlying is Optional.
func (ok *Okx) GetBlockTickers(ctx context.Context, instrumentType, underlying string) ([]BlockTicker, error) {
	instrumentType = strings.ToUpper(instrumentType)
	if instrumentType == "" {
		return nil, errInstrumentTypeRequired
	}
	params := url.Values{}
	params.Set("instType", instrumentType)
	if underlying != "" {
		params.Set("uly", underlying)
	}
	var resp []BlockTicker
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getBlockTickersEPL, http.MethodGet, common.EncodeURLValues("market/block-tickers", params), nil, &resp, false)
}

// GetBlockTicker retrieves the latest block trading volume in the last 24 hours.
func (ok *Okx) GetBlockTicker(ctx context.Context, instrumentID string) (*BlockTicker, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	var resp *BlockTicker
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getBlockTickersEPL, http.MethodGet, common.EncodeURLValues("market/block-ticker", params), nil, &resp, false)
}

// GetPublicBlockTrades retrieves the recent block trading transactions of an instrument. Descending order by tradeId.
func (ok *Okx) GetPublicBlockTrades(ctx context.Context, instrumentID string) ([]BlockTrade, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	var resp []BlockTrade
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getBlockTradesEPL, http.MethodGet, common.EncodeURLValues("public/block-trades", params), nil, &resp, false)
}

// ********************************************* Spread Trading ***********************************************

// Spread Trading: As Introduced in the Okx exchange. URL: https://www.okx.com/docs-v5/en/#spread-trading-introduction

// PlaceSpreadOrder places new spread order.
func (ok *Okx) PlaceSpreadOrder(ctx context.Context, arg *SpreadOrderParam) (*SpreadOrderResponse, error) {
	err := ok.validatePlaceSpreadOrderParam(arg)
	if err != nil {
		return nil, err
	}
	var resp *SpreadOrderResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, placeSpreadOrderEPL, http.MethodPost, "sprd/order", arg, &resp, true)
}

func (ok *Okx) validatePlaceSpreadOrderParam(arg *SpreadOrderParam) error {
	if arg == nil || *arg == (SpreadOrderParam{}) {
		return common.ErrNilPointer
	}
	if arg.SpreadID == "" {
		return fmt.Errorf("%w, spread ID missing", order.ErrOrderIDNotSet)
	}
	if arg.OrderType == "" {
		return fmt.Errorf("%w spread order type is required", order.ErrTypeIsInvalid)
	}
	if arg.Size <= 0 {
		return errMissingNewSize
	}
	if arg.Price <= 0 {
		return errInvalidMinimumPrice
	}
	arg.Side = strings.ToLower(arg.Side)
	if arg.Side != "buy" && arg.Side != "sell" {
		return fmt.Errorf("%w %s", order.ErrSideIsInvalid, arg.Side)
	}
	return nil
}

// CancelSpreadOrder cancels an incomplete spread order
func (ok *Okx) CancelSpreadOrder(ctx context.Context, orderID, clientOrderID string) (*SpreadOrderResponse, error) {
	if orderID == "" && clientOrderID == "" {
		return nil, errMissingClientOrderIDOrOrderID
	}
	arg := make(map[string]string)
	if orderID != "" {
		arg["ordId"] = orderID
	}
	if clientOrderID != "" {
		arg["clOrdId"] = clientOrderID
	}
	var resp *SpreadOrderResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, cancelSpreadOrderEPL, http.MethodPost, "sprd/cancel-order", arg, &resp, true)
}

// CancelAllSpreadOrders cancels all spread orders and return success message
// spreadID is optional
// the function returns success status and error message
func (ok *Okx) CancelAllSpreadOrders(ctx context.Context, spreadID string) (bool, error) {
	arg := make(map[string]string, 1)
	if spreadID != "" {
		arg["sprdId"] = spreadID
	}
	resp := &struct {
		Result bool `json:"result"`
	}{}
	return resp.Result, ok.SendHTTPRequest(ctx, exchange.RestSpot, cancelAllSpreadOrderEPL, http.MethodPost, "sprd/mass-cancel", arg, resp, true)
}

// AmendSpreadOrder amends incomplete spread order
func (ok *Okx) AmendSpreadOrder(ctx context.Context, arg *AmendSpreadOrderParam) (*SpreadOrderResponse, error) {
	if arg == nil || *arg == (AmendSpreadOrderParam{}) {
		return nil, common.ErrNilPointer
	}
	if arg.OrderID == "" && arg.ClientOrderID == "" {
		return nil, errMissingClientOrderIDOrOrderID
	}
	if arg.NewPrice == 0 && arg.NewSize == 0 {
		return nil, errSizeOrPriceIsRequired
	}
	var resp *SpreadOrderResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, amendSpreadOrderEPL, http.MethodPost, "sprd/amend-order", arg, &resp, true)
}

// GetSpreadOrderDetails retrieves spread order details.
func (ok *Okx) GetSpreadOrderDetails(ctx context.Context, orderID, clientOrderID string) (*SpreadOrder, error) {
	if orderID == "" && clientOrderID == "" {
		return nil, errMissingClientOrderIDOrOrderID
	}
	params := url.Values{}
	if orderID != "" {
		params.Set("ordId", orderID)
	}
	if clientOrderID != "" {
		params.Set("clOrdId", clientOrderID)
	}
	var resp *SpreadOrder
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getSpreadOrderDetailsEPL, http.MethodGet, common.EncodeURLValues("sprd/order", params), nil, &resp, true)
}

// GetActiveSpreadOrders retrieves list of incomplete spread orders.
func (ok *Okx) GetActiveSpreadOrders(ctx context.Context, spreadID, orderType, state, beginID, endID string, limit int64) ([]SpreadOrder, error) {
	params := url.Values{}
	if spreadID != "" {
		params.Set("sprdId", spreadID)
	}
	if orderType != "" {
		params.Set("ordType", orderType)
	}
	if state != "" {
		params.Set("state", state)
	}
	if beginID != "" {
		params.Set("beginId", beginID)
	}
	if endID != "" {
		params.Set("endId", endID)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []SpreadOrder
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getActiveSpreadOrdersEPL, http.MethodGet, common.EncodeURLValues("sprd/orders-pending", params), nil, &resp, true)
}

// GetCompletedSpreadOrdersLast7Days retrieve the completed order data for the last 7 days, and the incomplete orders (filledSz =0 & state = canceled) that have been canceled are only reserved for 2 hours. Results are returned in counter chronological order.
func (ok *Okx) GetCompletedSpreadOrdersLast7Days(ctx context.Context, spreadID, orderType, state, beginID, endID string, begin, end time.Time, limit int64) ([]SpreadOrder, error) {
	params := url.Values{}
	if spreadID != "" {
		params.Set("sprdId", spreadID)
	}
	if orderType != "" {
		params.Set("ordType", orderType)
	}
	if state != "" {
		params.Set("state", state)
	}
	if beginID != "" {
		params.Set("beginId", beginID)
	}
	if endID != "" {
		params.Set("endId", endID)
	}
	if !begin.IsZero() {
		params.Set("begin", strconv.FormatInt(begin.UnixMilli(), 10))
	}
	if !end.IsZero() {
		params.Set("end", strconv.FormatInt(end.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []SpreadOrder
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getSpreadOrders7DaysEPL, http.MethodGet, common.EncodeURLValues("sprd/orders-history", params), nil, &resp, true)
}

// GetSpreadTradesOfLast7Days retrieve historical transaction details for the last 7 days. Results are returned in counter chronological order.
func (ok *Okx) GetSpreadTradesOfLast7Days(ctx context.Context, spreadID, tradeID, orderID, beginID, endID string, begin, end time.Time, limit int64) ([]SpreadTrade, error) {
	params := url.Values{}
	if spreadID != "" {
		params.Set("sprdId", spreadID)
	}
	if tradeID != "" {
		params.Set("tradeId", tradeID)
	}
	if orderID != "" {
		params.Set("ordId", orderID)
	}
	if beginID != "" {
		params.Set("beginId", beginID)
	}
	if endID != "" {
		params.Set("endId", endID)
	}
	if !begin.IsZero() {
		params.Set("begin", strconv.FormatInt(begin.UnixMilli(), 10))
	}
	if !end.IsZero() {
		params.Set("end", strconv.FormatInt(end.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []SpreadTrade
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getSpreadOrderTradesEPL, http.MethodGet, common.EncodeURLValues("sprd/trades", params), nil, &resp, true)
}

// GetPublicSpreads retrieve all available spreads based on the request parameters.
func (ok *Okx) GetPublicSpreads(ctx context.Context, baseCurrency, instrumentID, spreadID, state string) ([]SpreadInstrument, error) {
	params := url.Values{}
	if baseCurrency != "" {
		params.Set("baseCcy", baseCurrency)
	}
	if instrumentID != "" {
		params.Set("instId", instrumentID)
	}
	if spreadID != "" {
		params.Set("sprdId", spreadID)
	}
	if state != "" {
		params.Set("state", state)
	}
	var resp []SpreadInstrument
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getSpreadsEPL, http.MethodGet, common.EncodeURLValues("sprd/spreads", params), nil, &resp, false)
}

// GetPublicSpreadOrderBooks retrieve the order book of the spread.
func (ok *Okx) GetPublicSpreadOrderBooks(ctx context.Context, spreadID string, orderbookSize int64) ([]SpreadOrderbook, error) {
	if spreadID == "" {
		return nil, fmt.Errorf("%w, spread ID missing", order.ErrOrderIDNotSet)
	}
	params := url.Values{}
	params.Set("sprdId", spreadID)
	if orderbookSize != 0 {
		params.Set("size", strconv.FormatInt(orderbookSize, 10))
	}
	var resp []SpreadOrderbook
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getSpreadOrderbookEPL, http.MethodGet, common.EncodeURLValues("sprd/books", params), nil, &resp, false)
}

// GetPublicSpreadTickers retrieve the latest price snapshot, best bid/ask price, and trading volume in the last 24 hours.
func (ok *Okx) GetPublicSpreadTickers(ctx context.Context, spreadID string) ([]SpreadTicker, error) {
	if spreadID == "" {
		return nil, fmt.Errorf("%w, spread ID missing", order.ErrOrderIDNotSet)
	}
	params := url.Values{}
	params.Set("sprdId", spreadID)
	var resp []SpreadTicker
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getSpreadTickerEPL, http.MethodGet, common.EncodeURLValues("sprd/ticker", params), nil, &resp, false)
}

// GetPublicSpreadTrades retrieve the recent transactions of an instrument (at most 500 records per request). Results are returned in counter chronological order.
func (ok *Okx) GetPublicSpreadTrades(ctx context.Context, spreadID string) ([]SpreadPublicTradeItem, error) {
	params := url.Values{}
	if spreadID != "" {
		params.Set("sprdId", spreadID)
	}
	var resp []SpreadPublicTradeItem
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getSpreadPublicTradesEPL, http.MethodGet, common.EncodeURLValues("sprd/public-trades", params), nil, &resp, false)
}

// CancelAllSpreadOrdersAfterCountdown cancel all pending orders after the countdown timeout. Only applicable to spread trading.
func (ok *Okx) CancelAllSpreadOrdersAfterCountdown(ctx context.Context, timeoutDuration int64) (*SpreadOrderCancellationResponse, error) {
	if (timeoutDuration != 0) && (timeoutDuration < 10 || timeoutDuration > 120) {
		return nil, fmt.Errorf("%w, Range of value can be 0, [10, 120]", errCountdownTimeoutRequired)
	}
	arg := &struct {
		TimeOut int64 `json:"timeOut,string"`
	}{
		TimeOut: timeoutDuration,
	}
	var resp *SpreadOrderCancellationResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, cancelAllSpreadOrdersAfterEPL, http.MethodPost, "sprd/cancel-all-after", arg, &resp, true)
}

/************************************ Public Data Endpoinst *************************************************/

// GetInstruments Retrieve a list of instruments with open contracts.
func (ok *Okx) GetInstruments(ctx context.Context, arg *InstrumentsFetchParams) ([]Instrument, error) {
	if arg.InstrumentType == "" {
		return nil, errInstrumentTypeRequired
	}
	params := url.Values{}
	arg.InstrumentType = strings.ToUpper(arg.InstrumentType)
	params.Set("instType", arg.InstrumentType)
	if arg.Underlying != "" {
		params.Set("uly", arg.Underlying)
	}
	if arg.InstrumentID != "" {
		params.Set("instId", arg.InstrumentID)
	}
	var resp []Instrument
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getInstrumentsEPL, http.MethodGet, common.EncodeURLValues("public/instruments", params), nil, &resp, false)
}

// GetDeliveryHistory retrieves the estimated delivery price of the last 3 months, which will only have a return value one hour before the delivery/exercise.
func (ok *Okx) GetDeliveryHistory(ctx context.Context, instrumentType, underlying string, after, before time.Time, limit int64) ([]DeliveryHistory, error) {
	if underlying == "" {
		return nil, errMissingRequiredUnderlying
	}
	if limit <= 0 || limit > 100 {
		return nil, errLimitValueExceedsMaxOf100
	}
	params := url.Values{}
	instrumentType = strings.ToUpper(instrumentType)
	if instrumentType != "" {
		params.Set("instType", instrumentType)
	}
	params.Set("uly", underlying)
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	params.Set("limit", strconv.FormatInt(limit, 10))
	var resp []DeliveryHistory
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getDeliveryExerciseHistoryEPL, http.MethodGet, common.EncodeURLValues("public/delivery-exercise-history", params), nil, &resp, false)
}

// GetOpenInterestData retrieves the total open interest for contracts on OKX
func (ok *Okx) GetOpenInterestData(ctx context.Context, instType, uly, instID string) ([]OpenInterest, error) {
	if instType == "" {
		return nil, errInstrumentTypeRequired
	}
	instType = strings.ToUpper(instType)
	params := url.Values{}
	params.Set("instType", instType)
	if uly != "" {
		params.Set("uly", uly)
	}
	if instID != "" {
		params.Set("instId", instID)
	}
	var resp []OpenInterest
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getOpenInterestEPL, http.MethodGet, common.EncodeURLValues("public/open-interest", params), nil, &resp, false)
}

// GetSingleFundingRate returns the latest funding rate
func (ok *Okx) GetSingleFundingRate(ctx context.Context, instrumentID string) (*FundingRateResponse, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	var resp *FundingRateResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getFundingEPL, http.MethodGet, common.EncodeURLValues("public/funding-rate", params), nil, &resp, false)
}

// GetFundingRateHistory retrieves funding rate history. This endpoint can retrieve data from the last 3 months.
func (ok *Okx) GetFundingRateHistory(ctx context.Context, instrumentID string, before, after time.Time, limit int64) ([]FundingRateResponse, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []FundingRateResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getFundingRateHistoryEPL, http.MethodGet, common.EncodeURLValues("public/funding-rate-history", params), nil, &resp, false)
}

// GetLimitPrice retrieves the highest buy limit and lowest sell limit of the instrument.
func (ok *Okx) GetLimitPrice(ctx context.Context, instrumentID string) (*LimitPriceResponse, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	var resp []LimitPriceResponse
	if err := ok.SendHTTPRequest(ctx, exchange.RestSpot, getLimitPriceEPL, http.MethodGet, common.EncodeURLValues("public/price-limit", params), nil, &resp, false); err != nil {
		return nil, err
	}
	if len(resp) == 1 {
		return &resp[0], nil
	}
	return nil, errFundingRateHistoryNotFound
}

// GetOptionMarketData retrieves option market data.
func (ok *Okx) GetOptionMarketData(ctx context.Context, underlying string, expTime time.Time) ([]OptionMarketDataResponse, error) {
	if underlying == "" {
		return nil, errMissingRequiredUnderlying
	}
	params := url.Values{}
	params.Set("uly", underlying)
	if !expTime.IsZero() {
		params.Set("expTime", fmt.Sprintf("%d%d%d", expTime.Year(), expTime.Month(), expTime.Day()))
	}
	var resp []OptionMarketDataResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getOptionMarketDateEPL, http.MethodGet, common.EncodeURLValues("public/opt-summary", params), nil, &resp, false)
}

// GetEstimatedDeliveryPrice retrieves the estimated delivery price which will only have a return value one hour before the delivery/exercise.
func (ok *Okx) GetEstimatedDeliveryPrice(ctx context.Context, instrumentID string) ([]DeliveryEstimatedPrice, error) {
	if instrumentID == "" {
		return nil, errMissingRequiredParamInstID
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	var resp []DeliveryEstimatedPrice
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getEstimatedDeliveryPriceEPL, http.MethodGet, common.EncodeURLValues("public/estimated-price", params), nil, &resp, false)
}

// GetDiscountRateAndInterestFreeQuota retrieves discount rate level and interest-free quota.
func (ok *Okx) GetDiscountRateAndInterestFreeQuota(ctx context.Context, ccy currency.Code, discountLevel int8) ([]DiscountRate, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if discountLevel > 0 {
		params.Set("discountLv", strconv.Itoa(int(discountLevel)))
	}
	var response []DiscountRate
	return response, ok.SendHTTPRequest(ctx, exchange.RestSpot, getDiscountRateAndInterestFreeQuotaEPL, http.MethodGet, common.EncodeURLValues("public/discount-rate-interest-free-quota", params), nil, &response, false)
}

// GetSystemTime Retrieve API server time.
func (ok *Okx) GetSystemTime(ctx context.Context) (time.Time, error) {
	resp := &ServerTime{}
	return resp.Timestamp.Time(), ok.SendHTTPRequest(ctx, exchange.RestSpot, getSystemTimeEPL, http.MethodGet, "public/time", nil, &resp, false)
}

// GetLiquidationOrders retrieves information on liquidation orders in the last day.
func (ok *Okx) GetLiquidationOrders(ctx context.Context, arg *LiquidationOrderRequestParams) (*LiquidationOrder, error) {
	arg.InstrumentType = strings.ToUpper(arg.InstrumentType)
	if arg.InstrumentType == "" {
		return nil, errInstrumentTypeRequired
	}
	params := url.Values{}
	params.Set("instType", arg.InstrumentType)
	arg.MarginMode = strings.ToLower(arg.MarginMode)
	if arg.MarginMode != "" {
		params.Set("mgnMode", arg.MarginMode)
	}
	switch {
	case arg.InstrumentType == okxInstTypeMargin && arg.InstrumentID != "":
		params.Set("instId", arg.InstrumentID)
	case arg.InstrumentType == okxInstTypeMargin && arg.Currency.String() != "":
		params.Set("ccy", arg.Currency.String())
	default:
		return nil, errEitherInstIDOrCcyIsRequired
	}
	if arg.InstrumentType != okxInstTypeMargin && arg.Underlying != "" {
		params.Set("uly", arg.Underlying)
	}
	if arg.InstrumentType == okxInstTypeFutures && arg.Alias != "" {
		params.Set("alias", arg.Alias)
	}
	if !arg.Before.IsZero() {
		params.Set("before", strconv.FormatInt(arg.Before.UnixMilli(), 10))
	}
	if !arg.After.IsZero() {
		params.Set("after", strconv.FormatInt(arg.After.UnixMilli(), 10))
	}
	if arg.Limit > 0 && arg.Limit < 100 {
		params.Set("limit", strconv.FormatInt(arg.Limit, 10))
	}
	var response []LiquidationOrder
	err := ok.SendHTTPRequest(ctx, exchange.RestSpot, getLiquidationOrdersEPL, http.MethodGet, common.EncodeURLValues("public/liquidation-orders", params), nil, &response, false)
	if err != nil {
		return nil, err
	}
	if len(response) == 1 {
		return &response[0], nil
	}
	return nil, errLiquidationOrderResponseNotFound
}

// GetMarkPrice  Retrieve mark price.
func (ok *Okx) GetMarkPrice(ctx context.Context, instrumentType, underlying, instrumentID string) ([]MarkPrice, error) {
	if instrumentType == "" {
		return nil, errInstrumentTypeRequired
	}
	params := url.Values{}
	params.Set("instType", strings.ToUpper(instrumentType))
	if underlying != "" {
		params.Set("uly", underlying)
	}
	if instrumentID != "" {
		params.Set("instId", instrumentID)
	}
	var response []MarkPrice
	return response, ok.SendHTTPRequest(ctx, exchange.RestSpot, getMarkPriceEPL, http.MethodGet, common.EncodeURLValues("public/mark-price", params), nil, &response, false)
}

// GetPositionTiers retrieves position tiers information，maximum leverage depends on your borrowings and margin ratio.
func (ok *Okx) GetPositionTiers(ctx context.Context, instrumentType, tradeMode, underlying, instrumentID, tiers string) ([]PositionTiers, error) {
	instrumentType = strings.ToUpper(instrumentType)
	if instrumentType == "" {
		return nil, errInstrumentTypeRequired
	}
	params := url.Values{}
	params.Set("instType", strings.ToUpper(instrumentType))
	tradeMode = strings.ToLower(tradeMode)
	if tradeMode != TradeModeCross && tradeMode != TradeModeIsolated {
		return nil, errIncorrectRequiredParameterTradeMode
	}
	params.Set("tdMode", tradeMode)
	if underlying != "" {
		params.Set("uly", underlying)
	}
	if instrumentType == okxInstTypeMargin && instrumentID != "" {
		params.Set("instId", instrumentID)
	} else if instrumentType == okxInstTypeMargin {
		return nil, errMissingInstrumentID
	}
	if tiers != "" {
		params.Set("tiers", tiers)
	}
	var response []PositionTiers
	return response, ok.SendHTTPRequest(ctx, exchange.RestSpot, getPositionTiersEPL, http.MethodGet, common.EncodeURLValues("public/position-tiers", params), nil, &response, false)
}

// GetInterestRateAndLoanQuota retrieves an interest rate and loan quota information for various currencies.
func (ok *Okx) GetInterestRateAndLoanQuota(ctx context.Context) (map[string][]InterestRateLoanQuotaItem, error) {
	var response []map[string][]InterestRateLoanQuotaItem
	err := ok.SendHTTPRequest(ctx, exchange.RestSpot, getInterestRateAndLoanQuotaEPL, http.MethodGet, "public/interest-rate-loan-quota", nil, &response, false)
	if err != nil {
		return nil, err
	} else if len(response) == 1 {
		return response[0], nil
	}
	return nil, errInterestRateAndLoanQuotaNotFound
}

// GetInterestRateAndLoanQuotaForVIPLoans retrieves an interest rate and loan quota information for VIP users of various currencies.
func (ok *Okx) GetInterestRateAndLoanQuotaForVIPLoans(ctx context.Context) ([]VIPInterestRateAndLoanQuotaInformation, error) {
	var response []VIPInterestRateAndLoanQuotaInformation
	return response, ok.SendHTTPRequest(ctx, exchange.RestSpot, getInterestRateAndLoanQuoteForVIPLoansEPL, http.MethodGet, "public/vip-interest-rate-loan-quota", nil, &response, false)
}

// GetPublicUnderlyings returns list of underlyings for various instrument types.
func (ok *Okx) GetPublicUnderlyings(ctx context.Context, instrumentType string) ([]string, error) {
	params := url.Values{}
	instrumentType = strings.ToUpper(instrumentType)
	if instrumentType == "" {
		return nil, errInstrumentTypeRequired
	}
	params.Set("instType", strings.ToUpper(instrumentType))
	var resp []string
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getUnderlyingEPL, http.MethodGet, common.EncodeURLValues("public/underlying", params), nil, &resp, false, false)
}

// GetInsuranceFundInformation returns insurance fund balance information.
func (ok *Okx) GetInsuranceFundInformation(ctx context.Context, arg *InsuranceFundInformationRequestParams) (*InsuranceFundInformation, error) {
	if arg == nil || *arg == (InsuranceFundInformationRequestParams{}) {
		return nil, common.ErrNilPointer
	}
	if arg.InstrumentType == "" {
		return nil, errInstrumentTypeRequired
	}
	params := url.Values{}
	params.Set("instType", strings.ToUpper(arg.InstrumentType))
	arg.Type = strings.ToLower(arg.Type)
	if arg.Type != "" {
		params.Set("type", arg.Type)
	}
	if arg.Underlying != "" {
		params.Set("uly", arg.Underlying)
	} else if arg.InstrumentType != okxInstTypeMargin {
		return nil, errMissingRequiredUnderlying
	}
	if !arg.Currency.IsEmpty() {
		params.Set("ccy", arg.Currency.String())
	}
	if !arg.Before.IsZero() {
		params.Set("before", strconv.FormatInt(arg.Before.UnixMilli(), 10))
	}
	if !arg.After.IsZero() {
		params.Set("after", strconv.FormatInt(arg.After.UnixMilli(), 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatInt(arg.Limit, 10))
	}
	var response []InsuranceFundInformation
	if err := ok.SendHTTPRequest(ctx, exchange.RestSpot, getInsuranceFundEPL, http.MethodGet, common.EncodeURLValues("public/insurance-fund", params), nil, &response, false); err != nil {
		return nil, err
	}
	if len(response) == 1 {
		return &response[0], nil
	}
	return nil, errInsuranceFundInformationNotFound
}

// CurrencyUnitConvert convert currency to contract, or contract to currency.
func (ok *Okx) CurrencyUnitConvert(ctx context.Context, instrumentID string, quantity, orderPrice float64, convertType uint, unitOfCcy currency.Code) (*UnitConvertResponse, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	if quantity <= 0 {
		return nil, errMissingQuantity
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	params.Set("sz", strconv.FormatFloat(quantity, 'f', 0, 64))
	if orderPrice > 0 {
		params.Set("px", strconv.FormatFloat(orderPrice, 'f', 0, 64))
	}
	if convertType > 0 {
		params.Set("type", strconv.Itoa(int(convertType)))
	}
	if !unitOfCcy.IsEmpty() {
		params.Set("unit", unitOfCcy.String())
	}
	var resp *UnitConvertResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, unitConvertEPL, http.MethodGet, common.EncodeURLValues("public/convert-contract-coin", params), nil, &resp, false)
}

// GetOptionsTickBands retrieves option tick bands information.
// Instrument type OPTION
func (ok *Okx) GetOptionsTickBands(ctx context.Context, instrumentType, instrumentFamily string) ([]OptionTickBand, error) {
	if instrumentType == "" {
		return nil, errInstrumentTypeRequired
	}
	params := url.Values{}
	params.Set("instType", instrumentType)
	if instrumentFamily != "" {
		params.Set("instFamily", instrumentFamily)
	}
	var resp []OptionTickBand
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, optionTickBandsEPL, http.MethodGet, common.EncodeURLValues("public/instrument-tick-bands", params), nil, &resp, false)
}

// Trading Data Endpoints

// GetSupportCoins retrieves the currencies supported by the trading data endpoints
func (ok *Okx) GetSupportCoins(ctx context.Context) (*SupportedCoinsData, error) {
	var response *SupportedCoinsData
	return response, ok.SendHTTPRequest(ctx, exchange.RestSpot, getSupportCoinEPL, http.MethodGet, "rubik/stat/trading-data/support-coin", nil, &response, false, true)
}

// GetTakerVolume retrieves the taker volume for both buyers and sellers.
func (ok *Okx) GetTakerVolume(ctx context.Context, ccy currency.Code, instrumentType string, begin, end time.Time, period kline.Interval) ([]TakerVolume, error) {
	if instrumentType == "" {
		return nil, errInstrumentTypeRequired
	}
	params := url.Values{}
	params.Set("instType", strings.ToUpper(instrumentType))
	interval := ok.GetIntervalEnum(period, false)
	if interval != "" {
		params.Set("period", interval)
	}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if !begin.IsZero() {
		params.Set("begin", strconv.FormatInt(begin.UnixMilli(), 10))
	}
	if !end.IsZero() {
		params.Set("end", strconv.FormatInt(end.UnixMilli(), 10))
	}
	var response [][3]types.Number
	err := ok.SendHTTPRequest(ctx, exchange.RestSpot, getTakerVolumeEPL, http.MethodGet, common.EncodeURLValues("rubik/stat/taker-volume", params), nil, &response, false)
	if err != nil {
		return nil, err
	}
	takerVolumes := make([]TakerVolume, len(response))
	for x := range response {
		takerVolumes[x] = TakerVolume{
			Timestamp:  time.UnixMilli(response[x][0].Int64()),
			SellVolume: response[x][1].Float64(),
			BuyVolume:  response[x][2].Float64(),
		}
	}
	return takerVolumes, nil
}

// GetMarginLendingRatio retrieves the ratio of cumulative amount between currency margin quote currency and base currency.
func (ok *Okx) GetMarginLendingRatio(ctx context.Context, ccy currency.Code, begin, end time.Time, period kline.Interval) ([]MarginLendRatioItem, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if !begin.IsZero() {
		params.Set("begin", strconv.FormatInt(begin.UnixMilli(), 10))
	}
	if !end.IsZero() {
		params.Set("end", strconv.FormatInt(begin.UnixMilli(), 10))
	}
	interval := ok.GetIntervalEnum(period, false)
	if interval != "" {
		params.Set("period", interval)
	}
	var response [][2]types.Number
	err := ok.SendHTTPRequest(ctx, exchange.RestSpot, getMarginLendingRatioEPL, http.MethodGet, common.EncodeURLValues("rubik/stat/margin/loan-ratio", params), nil, &response, false)
	if err != nil {
		return nil, err
	}
	lendingRatios := make([]MarginLendRatioItem, len(response))
	for x := range response {
		lendingRatios[x] = MarginLendRatioItem{
			Timestamp:       time.UnixMilli(response[x][0].Int64()),
			MarginLendRatio: response[x][1].Float64(),
		}
	}
	return lendingRatios, nil
}

// GetLongShortRatio retrieves the ratio of users with net long vs net short positions for futures and perpetual swaps.
func (ok *Okx) GetLongShortRatio(ctx context.Context, ccy currency.Code, begin, end time.Time, period kline.Interval) ([]LongShortRatio, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if !begin.IsZero() {
		params.Set("begin", strconv.FormatInt(begin.UnixMilli(), 10))
	}
	if !end.IsZero() {
		params.Set("end", strconv.FormatInt(begin.UnixMilli(), 10))
	}
	interval := ok.GetIntervalEnum(period, false)
	if interval != "" {
		params.Set("period", interval)
	}
	var response [][2]types.Number
	err := ok.SendHTTPRequest(ctx, exchange.RestSpot, getLongShortRatioEPL, http.MethodGet, common.EncodeURLValues("rubik/stat/contracts/long-short-account-ratio", params), nil, &response, false)
	if err != nil {
		return nil, err
	}
	ratios := make([]LongShortRatio, len(response))
	for x := range response {
		if response[x][0].Int64() <= 0 {
			return nil, fmt.Errorf("%w, expecting non zero timestamp, but found %d", errMalformedData, response[x][0].Int64())
		}
		ratios[x] = LongShortRatio{
			Timestamp:       time.UnixMilli(response[x][0].Int64()),
			MarginLendRatio: response[x][1].Float64(),
		}
	}
	return ratios, nil
}

// GetContractsOpenInterestAndVolume retrieves the open interest and trading volume for futures and perpetual swaps.
func (ok *Okx) GetContractsOpenInterestAndVolume(ctx context.Context, ccy currency.Code, begin, end time.Time, period kline.Interval) ([]OpenInterestVolume, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if !begin.IsZero() {
		params.Set("begin", strconv.FormatInt(begin.UnixMilli(), 10))
	}
	if !end.IsZero() {
		params.Set("end", strconv.FormatInt(begin.UnixMilli(), 10))
	}
	interval := ok.GetIntervalEnum(period, false)
	if interval != "" {
		params.Set("period", interval)
	}
	var response [][3]types.Number
	err := ok.SendHTTPRequest(ctx, exchange.RestSpot, getContractsOpenInterestAndVolumeEPL, http.MethodGet, common.EncodeURLValues("rubik/stat/contracts/open-interest-volume", params), nil, &response, false)
	if err != nil {
		return nil, err
	}
	openInterestVolumes := make([]OpenInterestVolume, len(response))
	for x := range response {
		if response[x][0].Int64() <= 0 {
			return nil, fmt.Errorf("%w, invalid Timestamp value %d", errMalformedData, response[x][0].Int64())
		}
		if response[x][1].Float64() <= 0 {
			return nil, fmt.Errorf("%w, OpendInterest has to be non-zero positive value, but found %f", errMalformedData, response[x][1].Float64())
		}
		openInterestVolumes[x] = OpenInterestVolume{
			Timestamp:    time.UnixMilli(response[x][0].Int64()),
			Volume:       response[x][2].Float64(),
			OpenInterest: response[x][1].Float64(),
		}
	}
	return openInterestVolumes, nil
}

// GetOptionsOpenInterestAndVolume retrieves the open interest and trading volume for options.
func (ok *Okx) GetOptionsOpenInterestAndVolume(ctx context.Context, ccy currency.Code, period kline.Interval) ([]OpenInterestVolume, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	interval := ok.GetIntervalEnum(period, false)
	if interval != "" {
		params.Set("period", interval)
	}
	var response [][3]types.Number
	err := ok.SendHTTPRequest(ctx, exchange.RestSpot, getOptionsOpenInterestAndVolumeEPL, http.MethodGet, common.EncodeURLValues("rubik/stat/option/open-interest-volume", params), nil, &response, false)
	if err != nil {
		return nil, err
	}
	openInterestVolumes := make([]OpenInterestVolume, len(response))
	for x := range response {
		if response[x][0].Int64() <= 0 {
			return nil, fmt.Errorf("%w, expecting non zero timestamp, but found %d", errMalformedData, response[x][0].Int64())
		}
		if response[x][1].Float64() <= 0 {
			return nil, fmt.Errorf("%w, OpendInterest has to be non-zero positive value, but found %f", errMalformedData, response[x][1].Float64())
		}
		openInterestVolumes[x] = OpenInterestVolume{
			Timestamp:    time.UnixMilli(response[x][0].Int64()),
			Volume:       response[x][2].Float64(),
			OpenInterest: response[x][1].Float64(),
		}
	}
	return openInterestVolumes, nil
}

// GetPutCallRatio retrieves the open interest ration and trading volume ratio of calls vs puts.
func (ok *Okx) GetPutCallRatio(ctx context.Context, ccy currency.Code,
	period kline.Interval) ([]OpenInterestVolumeRatio, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	interval := ok.GetIntervalEnum(period, false)
	if interval != "" {
		params.Set("period", interval)
	}
	var response [][3]types.Number
	err := ok.SendHTTPRequest(ctx, exchange.RestSpot, getPutCallRatioEPL, http.MethodGet, common.EncodeURLValues("rubik/stat/option/open-interest-volume-ratio", params), nil, &response, false)
	if err != nil {
		return nil, err
	}
	openInterestVolumeRatios := make([]OpenInterestVolumeRatio, len(response))
	for x := range response {
		openInterestVolumeRatios[x] = OpenInterestVolumeRatio{
			Timestamp:         time.UnixMilli(response[x][0].Int64()),
			VolumeRatio:       response[x][2].Float64(),
			OpenInterestRatio: response[x][1].Float64(),
		}
	}
	return openInterestVolumeRatios, nil
}

// GetOpenInterestAndVolumeExpiry retrieves the open interest and trading volume of calls and puts for each upcoming expiration.
func (ok *Okx) GetOpenInterestAndVolumeExpiry(ctx context.Context, ccy currency.Code, period kline.Interval) ([]ExpiryOpenInterestAndVolume, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	interval := ok.GetIntervalEnum(period, false)
	if interval != "" {
		params.Set("period", interval)
	}
	var resp [][6]string
	err := ok.SendHTTPRequest(ctx, exchange.RestSpot, getOpenInterestAndVolumeEPL, http.MethodGet, common.EncodeURLValues("rubik/stat/option/open-interest-volume-expiry", params), nil, &resp, false)
	if err != nil {
		return nil, err
	}
	volumes := make([]ExpiryOpenInterestAndVolume, len(resp))
	for x := range resp {
		var timestamp int64
		timestamp, err = strconv.ParseInt(resp[x][0], 10, 64)
		if err != nil {
			return nil, err
		}
		var expiryTime time.Time
		expTime := resp[x][1]
		if expTime != "" && len(expTime) == 8 {
			year, err := strconv.ParseInt(expTime[0:4], 10, 64)
			if err != nil {
				return nil, err
			}
			month, err := strconv.ParseInt(expTime[4:6], 10, 64)
			if err != nil {
				return nil, err
			}
			var months string
			var days string
			if month <= 9 {
				months = "0" + strconv.FormatInt(month, 10)
			} else {
				months = strconv.FormatInt(month, 10)
			}
			day, err := strconv.ParseInt(expTime[6:], 10, 64)
			if err != nil {
				return nil, err
			}
			if day <= 9 {
				days = "0" + strconv.FormatInt(day, 10)
			} else {
				days = strconv.FormatInt(day, 10)
			}
			expiryTime, err = time.Parse("2006-01-02", strconv.FormatInt(year, 10)+"-"+months+"-"+days)
			if err != nil {
				return nil, err
			}
		}
		callOpenInterest, err := strconv.ParseFloat(resp[x][2], 64)
		if err != nil {
			return nil, err
		}
		putOpenInterest, err := strconv.ParseFloat(resp[x][3], 64)
		if err != nil {
			return nil, err
		}
		callVol, err := strconv.ParseFloat(resp[x][4], 64)
		if err != nil {
			return nil, err
		}
		putVol, err := strconv.ParseFloat(resp[x][5], 64)
		if err != nil {
			return nil, err
		}
		volumes[x] = ExpiryOpenInterestAndVolume{
			Timestamp:        time.UnixMilli(timestamp),
			ExpiryTime:       expiryTime,
			CallOpenInterest: callOpenInterest,
			PutOpenInterest:  putOpenInterest,
			CallVolume:       callVol,
			PutVolume:        putVol,
		}
	}
	return volumes, nil
}

// GetOpenInterestAndVolumeStrike retrieves the taker volume for both buyers and sellers of calls and puts.
func (ok *Okx) GetOpenInterestAndVolumeStrike(ctx context.Context, ccy currency.Code,
	expTime time.Time, period kline.Interval) ([]StrikeOpenInterestAndVolume, error) {
	if expTime.IsZero() {
		return nil, errMissingExpiryTimeParameter
	}
	params := url.Values{}
	params.Set("expTime", expTime.Format("20060102"))
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	interval := ok.GetIntervalEnum(period, false)
	if interval != "" {
		params.Set("period", interval)
	}
	var resp [][6]types.Number
	err := ok.SendHTTPRequest(ctx, exchange.RestSpot, getOpenInterestAndVolumeEPL, http.MethodGet, common.EncodeURLValues("rubik/stat/option/open-interest-volume-strike", params), nil, &resp, false)
	if err != nil {
		return nil, err
	}
	volumes := make([]StrikeOpenInterestAndVolume, len(resp))
	for x := range resp {
		volumes[x].Timestamp = time.UnixMilli(resp[x][0].Int64())
		volumes[x].Strike = resp[x][1].Int64()
		volumes[x].CallOpenInterest = resp[x][2].Float64()
		volumes[x].PutOpenInterest = resp[x][3].Float64()
		volumes[x].CallVolume = resp[x][4].Float64()
		volumes[x].PutVolume = resp[x][5].Float64()
	}
	return volumes, nil
}

// GetTakerFlow shows the relative buy/sell volume for calls and puts.
// It shows whether traders are bullish or bearish on price and volatility
func (ok *Okx) GetTakerFlow(ctx context.Context, ccy currency.Code, period kline.Interval) (*CurrencyTakerFlow, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	interval := ok.GetIntervalEnum(period, false)
	if interval != "" {
		params.Set("period", interval)
	}
	var resp [7]types.Number
	err := ok.SendHTTPRequest(ctx, exchange.RestSpot, getTakerFlowEPL, http.MethodGet, common.EncodeURLValues("rubik/stat/option/taker-block-volume", params), nil, &resp, false, true)
	if err != nil {
		return nil, err
	}
	return &CurrencyTakerFlow{
		Timestamp:       time.UnixMilli(resp[0].Int64()),
		CallBuyVolume:   resp[1].Float64(),
		CallSellVolume:  resp[2].Float64(),
		PutBuyVolume:    resp[3].Float64(),
		PutSellVolume:   resp[4].Float64(),
		CallBlockVolume: resp[5].Float64(),
		PutBlockVolume:  resp[6].Float64(),
	}, nil
}

// ********************************************************** Affiliate **********************************************************************

// The Affiliate API offers affiliate users a flexible function to query the invitee information.
// Simply enter the UID of your direct invitee to access their relevant information, empowering your affiliate business growth and day-to-day business operation.
// If you have additional data requirements regarding the Affiliate API, please don't hesitate to contact your BD.
// We will reach out to you through your BD to provide more comprehensive API support.

// GetInviteesDetail retrieves affiliate invitees details.
func (ok *Okx) GetInviteesDetail(ctx context.Context, uid string) (*AffilateInviteesDetail, error) {
	if uid == "" {
		return nil, errUserIDRequired
	}
	var resp *AffilateInviteesDetail
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getAffilateInviteesDetailEPL, http.MethodGet, "affiliate/invitee/detail?uid="+uid, nil, &resp, true)
}

// GetUserAffilateRebateInformation this endpoint is used to get the user's affiliate rebate information for affiliate.
func (ok *Okx) GetUserAffilateRebateInformation(ctx context.Context, apiKey string) (*AffilateRebateInfo, error) {
	if apiKey == "" {
		return nil, errInvalidAPIKey
	}
	var resp *AffilateRebateInfo
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getUserAffilateRebateInformationEPL, http.MethodGet, "users/partner/if-rebate?apiKey="+apiKey, nil, &resp, true)
}

// SendHTTPRequest sends an authenticated http request to a desired
// path with a JSON payload (of present)
// URL arguments must be in the request path and not as url.URL values
func (ok *Okx) SendHTTPRequest(ctx context.Context, ep exchange.URL, f request.EndpointLimit, httpMethod, requestPath string, data, result interface{}, authenticated bool, useAsItIs ...bool) (err error) {
	rv := reflect.ValueOf(result)
	if rv.Kind() != reflect.Pointer {
		return errInvalidResponseParam
	}
	endpoint, err := ok.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	var respResult interface{}
	switch {
	case rv.Elem().Kind() == reflect.Slice && len(useAsItIs) > 0 && !useAsItIs[0]:
		respResult = &[]interface{}{&result}
	case rv.Elem().Kind() == reflect.Slice ||
		// When needed to use the result as it is.
		len(useAsItIs) > 0 && useAsItIs[0]:
		respResult = result
	default:
		respResult = &[]interface{}{result}
	}
	resp := struct {
		Code types.Number `json:"code"`
		Msg  string       `json:"msg"`
		Data interface{}  `json:"data"`
	}{
		Data: respResult,
	}
	requestType := request.AuthType(request.UnauthenticatedRequest)
	if authenticated {
		requestType = request.AuthenticatedRequest
	}
	newRequest := func() (*request.Item, error) {
		utcTime := time.Now().UTC().Format(time.RFC3339)
		payload := []byte("")

		if data != nil {
			payload, err = json.Marshal(data)
			if err != nil {
				return nil, err
			}
		}
		path := endpoint + requestPath
		headers := make(map[string]string)
		headers["Content-Type"] = "application/json"
		if _, okay := ctx.Value(testNetVal).(bool); okay {
			headers["x-simulated-trading"] = "1"
		}
		if authenticated {
			var creds *account.Credentials
			creds, err = ok.GetCredentials(ctx)
			if err != nil {
				return nil, err
			}
			signPath := "/" + okxAPIPath + requestPath
			var hmac []byte
			hmac, err = crypto.GetHMAC(crypto.HashSHA256,
				[]byte(utcTime+httpMethod+signPath+string(payload)),
				[]byte(creds.Secret))
			if err != nil {
				return nil, err
			}
			headers["OK-ACCESS-KEY"] = creds.Key
			headers["OK-ACCESS-SIGN"] = crypto.Base64Encode(hmac)
			headers["OK-ACCESS-TIMESTAMP"] = utcTime
			headers["OK-ACCESS-PASSPHRASE"] = creds.ClientID
		}
		return &request.Item{
			Method:        strings.ToUpper(httpMethod),
			Path:          path,
			Headers:       headers,
			Body:          bytes.NewBuffer(payload),
			Result:        &resp,
			Verbose:       ok.Verbose,
			HTTPDebugging: ok.HTTPDebugging,
			HTTPRecording: ok.HTTPRecording,
		}, nil
	}
	err = ok.SendPayload(ctx, f, newRequest, requestType)
	if err != nil {
		return err
	}
	switch {
	case rv.Kind() == reflect.Slice:
		value, okay := result.([]interface{})
		if !okay || result == nil || len(value) == 0 {
			return fmt.Errorf("%w, received invalid response", common.ErrNoResponse)
		}
	case rv.Kind() == reflect.Pointer && rv.Elem().Kind() != reflect.Slice:
	}
	if err == nil && resp.Code.Int64() != 0 {
		if resp.Msg != "" {
			return fmt.Errorf("%w error code: %d message: %s", request.ErrAuthRequestFailed, resp.Code.Int64(), resp.Msg)
		}
		err, okay := ErrorCodes[resp.Code.String()]
		if okay {
			return err
		}
		return fmt.Errorf("%w error code: %d", request.ErrAuthRequestFailed, resp.Code.Int64())
	}
	return nil
}

// Status

// SystemStatusResponse retrieves the system status.
// state supports valid values 'scheduled', 'ongoing', 'pre_open', 'completed', and 'canceled'.
func (ok *Okx) SystemStatusResponse(ctx context.Context, state string) ([]SystemStatusResponse, error) {
	params := url.Values{}
	params.Set("state", state)
	var resp []SystemStatusResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, getEventStatusEPL, http.MethodGet, common.EncodeURLValues("system/status", params), nil, &resp, false)
}

// GetAssetTypeFromInstrumentType returns an asset Item instance given and Instrument Type string.
func GetAssetTypeFromInstrumentType(instrumentType string) asset.Item {
	switch strings.ToUpper(instrumentType) {
	case okxInstTypeSwap, okxInstTypeContract:
		return asset.PerpetualSwap
	case okxInstTypeSpot:
		return asset.Spot
	case okxInstTypeMargin:
		return asset.Margin
	case okxInstTypeFutures:
		return asset.Futures
	case okxInstTypeOption:
		return asset.Options
	}
	return asset.Empty
}

// GetAssetsFromInstrumentTypeOrID parses an instrument type and instrument ID and returns a list of assets
// that the currency pair is associated with.
func (ok *Okx) GetAssetsFromInstrumentTypeOrID(instType, instrumentID string) ([]asset.Item, error) {
	if instType != "" {
		a := GetAssetTypeFromInstrumentType(instType)
		if a != asset.Empty {
			return []asset.Item{a}, nil
		}
	}
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	pf, err := ok.CurrencyPairs.GetFormat(asset.Spot, true)
	if err != nil {
		return nil, err
	}
	splitSymbol := strings.Split(instrumentID, pf.Delimiter)
	if len(splitSymbol) <= 1 {
		return nil, fmt.Errorf("%w %v", currency.ErrCurrencyNotSupported, instrumentID)
	}
	pair, err := currency.NewPairDelimiter(instrumentID, pf.Delimiter)
	if err != nil {
		return nil, err
	}
	switch {
	case len(splitSymbol) == 2:
		resp := make([]asset.Item, 0, 2)
		enabled, err := ok.IsPairEnabled(pair, asset.Spot)
		if err != nil {
			return nil, err
		}
		if enabled {
			resp = append(resp, asset.Spot)
		}
		enabled, err = ok.IsPairEnabled(pair, asset.Margin)
		if err != nil {
			return nil, err
		}
		if enabled {
			resp = append(resp, asset.Margin)
		}
		if len(resp) > 0 {
			return resp, nil
		}
	case len(splitSymbol) > 2:
		switch splitSymbol[len(splitSymbol)-1] {
		case "SWAP", "swap":
			enabled, err := ok.IsPairEnabled(pair, asset.PerpetualSwap)
			if err != nil {
				return nil, err
			}
			if enabled {
				return []asset.Item{asset.PerpetualSwap}, nil
			}
		case "C", "P", "c", "p":
			enabled, err := ok.IsPairEnabled(pair, asset.Options)
			if err != nil {
				return nil, err
			}
			if enabled {
				return []asset.Item{asset.Options}, nil
			}
		default:
			enabled, err := ok.IsPairEnabled(pair, asset.Futures)
			if err != nil {
				return nil, err
			}
			if enabled {
				return []asset.Item{asset.Futures}, nil
			}
		}
	}
	return nil, fmt.Errorf("%w '%v' or currency not enabled '%v'", asset.ErrNotSupported, instType, instrumentID)
}

// -------------------------------------------------------  Lending Orders  ------------------------------------------------------

// PlaceLendingOrder places a lending order
func (ok *Okx) PlaceLendingOrder(ctx context.Context, arg *LendingOrderParam) (*LendingOrderResponse, error) {
	if arg == nil || *arg == (LendingOrderParam{}) {
		return nil, common.ErrNilPointer
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Amount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	if arg.Rate <= 0 {
		return nil, errLendingRateRequired
	}
	if arg.Term == "" {
		return nil, errLendingTermIsRequired
	}
	var resp *LendingOrderResponse
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, placeLendingOrderEPL, http.MethodPost, "finance/fixed-loan/lending-order", arg, &resp, true, false)
}

// Note: the documentation for Amending lending order has similar url, request method, and parameters to the placing order. Therefore, the implementation is skipped for now.

// GetLendingOrders retrieves list of lending orders.
// State: possible values are 'pending', 'earning', 'expired', 'settled'
func (ok *Okx) GetLendingOrders(ctx context.Context, orderID, term, state string, ccy currency.Code, startAt, endAt time.Time, limit int64) ([]LendingOrderDetail, error) {
	params := url.Values{}
	if orderID != "" {
		params.Set("ordId", orderID)
	}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if term != "" {
		params.Set("term", term)
	}
	if !startAt.IsZero() && !endAt.IsZero() {
		err := common.StartEndTimeCheck(startAt, endAt)
		if err != nil {
			return nil, err
		}
		params.Set("after", strconv.FormatInt(startAt.UnixMilli(), 10))
		params.Set("before", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	if state != "" {
		params.Set("state", state)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []LendingOrderDetail
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, lendingOrderListEPL, http.MethodGet, common.EncodeURLValues("finance/fixed-loan/lending-orders-list", params), nil, &resp, true)
}

// GetLendingSubOrderList retrieves a lending sub-orders list.
func (ok *Okx) GetLendingSubOrderList(ctx context.Context, orderID, state string, startAt, endAt time.Time, limit int64) ([]LendingSubOrder, error) {
	params := url.Values{}
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if state != "" {
		params.Set("state", state)
	}
	if !startAt.IsZero() && !endAt.IsZero() {
		err := common.StartEndTimeCheck(startAt, endAt)
		if err != nil {
			return nil, err
		}
		params.Set("after", strconv.FormatInt(startAt.UnixMilli(), 10))
		params.Set("before", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []LendingSubOrder
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, lendingSubOrderListEPL, http.MethodGet, common.EncodeURLValues("finance/fixed-loan/lending-sub-orders", params), nil, &resp, true)
}

// GetLendingOffers get lending-supported currencies and estimated APY.
func (ok *Okx) GetLendingOffers(ctx context.Context, ccy currency.Code, term string) ([]PublicLendingOffer, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if term != "" {
		params.Set("term", term)
	}
	var resp []PublicLendingOffer
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, lendingPublicOfferEPL, http.MethodGet, common.EncodeURLValues("finance/fixed-loan/lending-offers", params), nil, &resp, false)
}

// GetLendingAPYHistory retrieves a lending history.
func (ok *Okx) GetLendingAPYHistory(ctx context.Context, ccy currency.Code, term string) ([]LendingAPIHistoryItem, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if term == "" {
		return nil, errLendingTermIsRequired
	}
	params := url.Values{}
	params.Set("term", term)
	params.Set("ccy", ccy.String())
	var resp []LendingAPIHistoryItem
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, lendingAPYHistoryEPL, http.MethodGet, common.EncodeURLValues("finance/fixed-loan/lending-apy-history", params), nil, &resp, false)
}

// GetLendingVolume retrieves a lending volume
func (ok *Okx) GetLendingVolume(ctx context.Context, ccy currency.Code, term string) ([]LendingVolume, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if term == "" {
		return nil, errLendingTermIsRequired
	}
	params := url.Values{}
	params.Set("term", term)
	params.Set("ccy", ccy.String())
	var resp []LendingVolume
	return resp, ok.SendHTTPRequest(ctx, exchange.RestSpot, lendingVolumeEPL, http.MethodGet, common.EncodeURLValues("finance/fixed-loan/pending-lending-volume", params), nil, &resp, false)
}

// Trading Statistics endpoints.

// GetFuturesContractsOpenInterestHistory retrieve the contract open interest statistics of futures and perp. This endpoint returns a maximum of 1440 records.
func (ok *Okx) GetFuturesContractsOpenInterestHistory(ctx context.Context, instrumentID string, period kline.Interval, startAt, endAt time.Time, limit int64) ([]ContractOpenInterestHistoryItem, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	if period != kline.Interval(0) {
		params.Set("period", ok.GetIntervalEnum(period, true))
	}
	if !startAt.IsZero() {
		params.Set("begin", strconv.FormatInt(startAt.UnixMilli(), 10))
	}
	if !endAt.IsZero() {
		params.Set("end", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp [][4]types.Number
	err := ok.SendHTTPRequest(ctx, exchange.RestSpot, rubikGetContractOpenInterestHistoryEPL, http.MethodGet, common.EncodeURLValues("rubik/stat/contracts/open-interest-history", params), nil, &resp, false)
	if err != nil {
		return nil, err
	}
	return extractOpenInterestHistoryFromSlice(resp), nil
}

// GetFuturesContractTakerVolume retrieve the contract taker volume for both buyers and sellers. This endpoint returns a maximum of 1440 records.
// The unit of buy/sell volume, the default is 1. '0': Crypto '1': Contracts '2': U
func (ok *Okx) GetFuturesContractTakerVolume(ctx context.Context, instrumentID string, period kline.Interval, unit, limit int64, startAt, endAt time.Time) ([]ContractTakerVolume, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	if period != kline.Interval(0) {
		params.Set("period", ok.GetIntervalEnum(period, true))
	}
	if !startAt.IsZero() {
		params.Set("begin", strconv.FormatInt(startAt.UnixMilli(), 10))
	}
	if !endAt.IsZero() {
		params.Set("end", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	if unit != 1 {
		params.Set("unit", strconv.FormatInt(unit, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp [][3]types.Number
	err := ok.SendHTTPRequest(ctx, exchange.RestSpot, rubikContractTakerVolumeEPL, http.MethodGet, common.EncodeURLValues("rubik/stat/taker-volume-contract", params), nil, &resp, false)
	if err != nil {
		return nil, err
	}
	return extractTakerVolumesFromSlice(resp), nil
}

// GetFuturesContractLongShortAccountRatio retrieve the account long/short ratio of a contract. This endpoint returns a maximum of 1440 records.
func (ok *Okx) GetFuturesContractLongShortAccountRatio(ctx context.Context, instrumentID string, period kline.Interval, startAt, endAt time.Time, limit int64) ([]TopTraderContractsLongShortRatio, error) {
	return ok.getTopTradersFuturesContractLongShortRatio(ctx, instrumentID, "rubik/stat/contracts/long-short-account-ratio-contract", period, startAt, endAt, limit)
}

// GetTopTradersFuturesContractLongShortAccountRatio retrieve the account net long/short ratio of a contract for top traders.
// Top traders refer to the top 5% of traders with the largest open position value.
// This endpoint returns a maximum of 1440 records.
func (ok *Okx) GetTopTradersFuturesContractLongShortAccountRatio(ctx context.Context, instrumentID string, period kline.Interval, startAt, endAt time.Time, limit int64) ([]TopTraderContractsLongShortRatio, error) {
	return ok.getTopTradersFuturesContractLongShortRatio(ctx, instrumentID, "rubik/stat/contracts/long-short-account-ratio-contract-top-trader", period, startAt, endAt, limit)
}

// GetTopTradersFuturesContractLongShortPositionRatio retrieve the position long/short ratio of a contract for top traders. Top traders refer to the top 5% of traders with the largest open position value. This endpoint returns a maximum of 1440 records.
func (ok *Okx) GetTopTradersFuturesContractLongShortPositionRatio(ctx context.Context, instrumentID string, period kline.Interval, startAt, endAt time.Time, limit int64) ([]TopTraderContractsLongShortRatio, error) {
	return ok.getTopTradersFuturesContractLongShortRatio(ctx, instrumentID, "rubik/stat/contracts/long-short-position-ratio-contract-top-trader", period, startAt, endAt, limit)
}

func (ok *Okx) getTopTradersFuturesContractLongShortRatio(ctx context.Context, instrumentID, path string, period kline.Interval, startAt, endAt time.Time, limit int64) ([]TopTraderContractsLongShortRatio, error) {
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	}
	params := url.Values{}
	params.Set("instId", instrumentID)
	if period != kline.Interval(0) {
		params.Set("period", ok.GetIntervalEnum(period, true))
	}
	if !startAt.IsZero() {
		params.Set("begin", strconv.FormatInt(startAt.UnixMilli(), 10))
	}
	if !endAt.IsZero() {
		params.Set("end", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp [][2]types.Number
	err := ok.SendHTTPRequest(ctx, exchange.RestSpot, rubikTopTradersContractLongShortRatioEPL, http.MethodGet, common.EncodeURLValues(path, params), nil, &resp, false)
	if err != nil {
		return nil, err
	}
	return extractLongShortRatio(resp), nil
}

// func (ok *Okx)
