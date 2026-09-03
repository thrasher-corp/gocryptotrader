package htx

// htxError provides comparable package errors without allocating errors at declaration sites.
type htxError string

// Error implements the error interface.
func (e htxError) Error() string {
	return string(e)
}

const (
	errWithdrawDetailsUnset          htxError = "currency, address and amount must be set"
	errInvalidEndpoint               htxError = "invalid endpoint"
	errInvalidContractType           htxError = "invalid contract type"
	errInconsistentContractExpiry    htxError = "inconsistent contract expiry date codes"
	errInvalidPeriod                 htxError = "invalid period"
	errStartTimeAfterEndTime         htxError = "start time cannot be after end time"
	errInvalidSize                   htxError = "invalid size"
	errInvalidTradeType              htxError = "invalid trade type"
	errInvalidTradeID                htxError = "invalid trade ID"
	errInvalidAmountType             htxError = "invalid amount type"
	errInvalidRecordType             htxError = "invalid record type"
	errInvalidOrderType              htxError = "invalid order type"
	errInvalidTransferType           htxError = "invalid transfer type"
	errInvalidCreateDate             htxError = "invalid create date"
	errHistoryTimeRangeExceeded      htxError = "history time range cannot exceed 48 hours"
	errInvalidOffsetAmounts          htxError = "invalid offset amounts"
	errInvalidLeverage               htxError = "invalid leverage"
	errBatchAssetMismatch            htxError = "batch orders must use one asset type"
	errBatchPairMismatch             htxError = "batch orders must use one currency pair"
	errUnexpectedBatchResponseCount  htxError = "unexpected batch response count"
	errInvalidOrderPriceType         htxError = "invalid order price type"
	errBatchOrderLimitExceeded       htxError = "a maximum of 10 batch orders is supported"
	errContractCodeLimitExceeded     htxError = "between 1 and 10 contract codes must be supplied"
	errInvalidRequestType            htxError = "invalid request type"
	errInvalidOrderStatus            htxError = "invalid order status"
	errInvalidPositionMode           htxError = "invalid position mode"
	errInvalidTriggerType            htxError = "invalid trigger type"
	errInvalidOffset                 htxError = "invalid offset"
	errExpectedResponseBody          htxError = "expected response body"
	errUnexpectedResponseBody        htxError = "expected no response body"
	errEmptyResult                   htxError = "result contains no data"
	errDepositAddressMissing         htxError = "deposit address data is not populated"
	errBidPriceTypeAssertion         htxError = "unable to type assert bid price"
	errBidAmountTypeAssertion        htxError = "unable to type assert bid amount"
	errAskPriceTypeAssertion         htxError = "unable to type assert ask price"
	errAskAmountTypeAssertion        htxError = "unable to type assert ask amount"
	errUnrecognisedOrderStatus       htxError = "unrecognised order status"
	errUnrecognisedOrderSide         htxError = "unrecognised order side"
	errUnrecognisedOrderType         htxError = "unrecognised order type"
	errInvalidBidData                htxError = "invalid bid data"
	errInvalidAskData                htxError = "invalid ask data"
	errNoAccountReturned             htxError = "no account returned"
	errDepositAddressNotFound        htxError = "unable to match deposit address currency or chain"
	errCurrencyNotSupplied           htxError = "currency must be supplied"
	errNoTransferChains              htxError = "no chains returned from currencies API"
	errUnhandledMockWebsocketMessage htxError = "unhandled mock websocket message"
)
