package gateio

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const (
	p2pTestTransactionID   = 40000001
	p2pTestAdvertisementID = 2124000001
	p2pTestCounterpartyUID = "biz_uid_demo_b84d21"
	p2pTestChatFileKey     = "c2cchat_image/c2ctrade-demo-receipt|s3-gateio-payments"
)

func TestGetP2PAccountInfo(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetP2PAccountInfo(t.Context())
	require.NoError(t, err, "GetP2PAccountInfo must not error")
	if mockTests {
		exp := &P2PMerchantInfoResponse{
			IsSelf:                    true,
			UserRegistrationDate:      Date(time.Date(2025, 11, 19, 0, 0, 0, 0, time.UTC)),
			CounterpartiesNumber:      12,
			EmailVerified:             true,
			Verified:                  true,
			UserName:                  "merchant_demo",
			UserNote:                  "Preferred counterparty",
			CompleteTransactions:      128,
			PaidTransactions:          68,
			AcceptedTransactions:      60,
			TransactionsUsedTime:      300,
			CancelledUsedTimeMonth:    2,
			CompleteTransactionsMonth: 32,
			CompleteRateMonth:         96,
			OrdersBuyRateMonth:        53,
			IsFollow:                  true,
			HaveTraded:                true,
			BizUID:                    "biz_uid_demo_9f3a7c",
			BlueVIP:                   1,
			WorkStatus:                1,
			RegistrationDays:          42,
			FirstTradeDays:            30,
			MerchantInfo:              &P2PMerchantMarketInfo{Type: "0", Market: "USD"},
			OnlineStatus:              1,
			WorkHours:                 json.RawMessage(`null`),
			TransactionsMonth:         6400.5,
			TransactionsAll:           28600.75,
		}
		assert.Equal(t, exp, result, "GetP2PAccountInfo should decode every field")
		return
	}
	assert.NotNil(t, result, "GetP2PAccountInfo should return account information")
}

func TestGetP2PCounterpartyInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetP2PCounterpartyInfo(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetP2PCounterpartyInfo must reject a nil request")
	_, err = e.GetP2PCounterpartyInfo(t.Context(), &GetCounterpartyInfoRequest{})
	require.ErrorIs(t, err, errBizUIDRequired, "GetP2PCounterpartyInfo must require a biz UID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetP2PCounterpartyInfo(t.Context(), &GetCounterpartyInfoRequest{BizUID: p2pTestCounterpartyUID})
	require.NoError(t, err, "GetP2PCounterpartyInfo must not error")
	if mockTests {
		exp := &P2PCounterpartyInfoResponse{
			UserRegistrationDate:      Date(time.Date(2025, 11, 19, 0, 0, 0, 0, time.UTC)),
			EmailVerified:             true,
			Verified:                  true,
			HasPhone:                  true,
			UserName:                  "counterparty_demo",
			CompleteTransactions:      86,
			PaidTransactions:          44,
			AcceptedTransactions:      42,
			TransactionsUsedTime:      420,
			CancelledUsedTimeMonth:    1,
			CompleteTransactionsMonth: 18,
			CompleteRateMonth:         95,
			BizUID:                    p2pTestCounterpartyUID,
			RegistrationDays:          180,
			FirstTradeDays:            90,
			TradeVersatile:            true,
		}
		assert.Equal(t, exp, result, "GetP2PCounterpartyInfo should decode every field")
		return
	}
	assert.NotNil(t, result, "GetP2PCounterpartyInfo should return counterparty information")
}

func TestGetP2PPaymentMethods(t *testing.T) {
	t.Parallel()
	_, err := e.GetP2PPaymentMethods(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetP2PPaymentMethods must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetP2PPaymentMethods(t.Context(), &GetP2PPaymentMethodsRequest{Fiat: currency.USD})
	require.NoError(t, err, "GetP2PPaymentMethods must not error")
	if !mockTests {
		assert.NotNil(t, result, "GetP2PPaymentMethods should return payment methods")
		return
	}
	exp := []*P2PPaymentMethodGroupResponse{
		{
			PayType: "bank",
			PayName: "Bank Transfer",
			IDs:     []uint64{10001},
			List: []*P2PPaymentMethod{{
				UID:             1000001,
				BankID:          "10001",
				Nickname:        1000001,
				BankName:        "Demo Bank",
				BankBranch:      "Main Branch",
				BankCity:        "New York",
				BankProvince:    "NY",
				BankCardNumber:  "****1234",
				BankDescription: "Corporate settlement account",
				HolderUID:       1000001,
				HolderUsername:  "merchant_demo",
				RealName:        "Merchant Demo",
			}},
		},
		{
			PayType: "swift",
			PayName: "SWIFT International Remittance",
			IDs:     []uint64{10002},
			List: []*P2PPaymentMethod{{
				ID:                    "10002",
				AccountDescription:    "Business USD account",
				PaymentMethodType:     "swift",
				PaymentMethodFileLink: "https://example.com/p2p/swift.png",
				PaymentMethodFileKey:  "swift-file-key",
				Account:               "****5678",
				Memo:                  "Use order txid as reference",
				Code:                  "SWIFTCODE",
				MemoExtended:          "Extra note",
				TradeTips:             "Please pay from an account under your real name",
				RealName:              "Merchant Demo",
			}},
		},
	}
	assert.Equal(t, exp, result, "GetP2PPaymentMethods should decode every field")

	// Without a fiat filter the request body is empty, returning every payment method
	result, err = e.GetP2PPaymentMethods(t.Context(), &GetP2PPaymentMethodsRequest{})
	require.NoError(t, err, "GetP2PPaymentMethods must not error without a fiat filter")
	assert.Len(t, result, 2, "GetP2PPaymentMethods should return every payment method group")
}

func TestSetMerchantWorkingStatusAndCustomWorking(t *testing.T) {
	t.Parallel()
	_, err := e.SetMerchantWorkingStatusAndCustomWorking(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SetMerchantWorkingStatusAndCustomWorking must reject a nil request")
	_, err = e.SetMerchantWorkingStatusAndCustomWorking(t.Context(), &SetMerchantWorkHoursRequest{WorkStatus: 3})
	require.ErrorIs(t, err, errP2PWorkStatusInvalid, "SetMerchantWorkingStatusAndCustomWorking must reject an unknown work status")
	arg := &SetMerchantWorkHoursRequest{WorkStatus: P2PWorkStatusCustomHours}
	_, err = e.SetMerchantWorkingStatusAndCustomWorking(t.Context(), arg)
	require.ErrorIs(t, err, errP2PCycleTypeRequired, "SetMerchantWorkingStatusAndCustomWorking must require a cycle type for custom hours")
	arg.CycleType = "Weekly"
	_, err = e.SetMerchantWorkingStatusAndCustomWorking(t.Context(), arg)
	require.ErrorIs(t, err, errP2PDayOfWeekRequired, "SetMerchantWorkingStatusAndCustomWorking must require working days for a weekly cycle")
	arg.DayOfWeek = "1,2,3,4,5"
	_, err = e.SetMerchantWorkingStatusAndCustomWorking(t.Context(), arg)
	require.ErrorIs(t, err, errP2PTimeZoneRequired, "SetMerchantWorkingStatusAndCustomWorking must require a time zone for custom hours")
	arg.TimeZone = "+8"
	_, err = e.SetMerchantWorkingStatusAndCustomWorking(t.Context(), arg)
	require.ErrorIs(t, err, errP2PWorkingHoursRequired, "SetMerchantWorkingStatusAndCustomWorking must require working hours for custom hours")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	arg.StartTime, arg.EndTime = "09:00", "18:00"
	result, err := e.SetMerchantWorkingStatusAndCustomWorking(t.Context(), arg)
	require.NoError(t, err, "SetMerchantWorkingStatusAndCustomWorking must not error")
	if mockTests {
		assert.Equal(t, &P2PMerchantWorkStatusResponse{WorkStatus: 3}, result, "SetMerchantWorkingStatusAndCustomWorking should decode every field")
		return
	}
	assert.NotNil(t, result, "SetMerchantWorkingStatusAndCustomWorking should return the working status")
}

func TestGetPendingP2POrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetPendingP2POrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPendingP2POrders must reject a nil request")
	_, err = e.GetPendingP2POrders(t.Context(), &PendingP2POrderRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetPendingP2POrders must require a crypto currency")
	_, err = e.GetPendingP2POrders(t.Context(), &PendingP2POrderRequest{CryptoCurrency: currency.USDT})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetPendingP2POrders must require a fiat currency")
	startTime, endTime := getTime()
	_, err = e.GetPendingP2POrders(t.Context(), &PendingP2POrderRequest{CryptoCurrency: currency.USDT, FiatCurrency: currency.USD, StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetPendingP2POrders must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPendingP2POrders(t.Context(), &PendingP2POrderRequest{
		CryptoCurrency: currency.USDT,
		FiatCurrency:   currency.USD,
		OrderTab:       "pending",
		SelectType:     "sell",
		Status:         "open",
		TransactionID:  p2pTestTransactionID,
		StartTime:      startTime,
		EndTime:        endTime,
	})
	require.NoError(t, err, "GetPendingP2POrders must not error")
	if mockTests {
		exp := &P2POrderListResponse{
			List: []*P2POrderInfo{{
				IsBuy:                true,
				CreationTime:         "2025-12-15 08:27:09",
				ExpiryTime:           "2025-12-15 10:27:09",
				Timestamp:            types.Time(time.Unix(1765787229, 0)),
				Rate:                 1.1,
				Amount:               8,
				Total:                8.8,
				TransactionID:        p2pTestTransactionID,
				Status:               "paid",
				CounterpartyRealName: "Counterparty Demo",
				CounterpartyUID:      p2pTestCounterpartyUID,
				CounterpartyNickname: "counterparty_demo",
				SellerRealName:       "Merchant Demo",
				BuyerRealName:        "Counterparty Demo",
				Cancelable:           true,
				CurrencyType:         currency.USDT,
				WantType:             currency.USD,
				SelectedPayType:      "bank",
				Countdown:            types.DurationSeconds(10 * time.Minute),
				OrderType:            3,
				OrderTags:            []string{"fast"},
				ConvertInfo: &P2PConvertInfo{
					ConvertType:         "BTC",
					ConvertStatus:       "DONE",
					ExpectedPriceRate:   1.1,
					ExecutionRate:       1.09,
					ExpectedFiatPrice:   1.12,
					FiatRate:            1.11,
					Amount:              8,
					SwapAmount:          0.0001,
					SlippageCalculation: 0.009,
					Status:              "finished",
				},
			}},
			TransactionTime: []*P2PTransactionTimeMarker{{OrderCountdown: types.DurationSeconds(10 * time.Minute)}},
			Count:           1,
		}
		assert.Equal(t, exp, result, "GetPendingP2POrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPendingP2POrders should return an order list")
}

func TestPendingP2POrderRequestMarshalJSON(t *testing.T) {
	t.Parallel()
	got, err := (&PendingP2POrderRequest{CryptoCurrency: currency.USDT, FiatCurrency: currency.USD}).MarshalJSON()
	require.NoError(t, err, "MarshalJSON must not error")
	assert.JSONEq(t, `{"crypto_currency":"USDT","fiat_currency":"USD"}`, string(got), "unset optional fields should be omitted")

	got, err = (&PendingP2POrderRequest{CryptoCurrency: currency.USDT, FiatCurrency: currency.USD, StartTime: time.Unix(1764547200, 0), EndTime: time.Unix(1767139199, 0)}).MarshalJSON()
	require.NoError(t, err, "MarshalJSON must not error")
	assert.JSONEq(t, `{"crypto_currency":"USDT","fiat_currency":"USD","start_time":1764547200,"end_time":1767139199}`, string(got), "times should be sent as Unix seconds")
}

func TestGetHistoricalP2POrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricalP2POrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetHistoricalP2POrders must reject a nil request")
	_, err = e.GetHistoricalP2POrders(t.Context(), &P2PCompletedOrderRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetHistoricalP2POrders must require a crypto currency")
	_, err = e.GetHistoricalP2POrders(t.Context(), &P2PCompletedOrderRequest{CryptoCurrency: currency.USDT})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetHistoricalP2POrders must require a fiat currency")
	startTime, endTime := getTime()
	_, err = e.GetHistoricalP2POrders(t.Context(), &P2PCompletedOrderRequest{CryptoCurrency: currency.USDT, FiatCurrency: currency.USD, StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetHistoricalP2POrders must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetHistoricalP2POrders(t.Context(), &P2PCompletedOrderRequest{
		CryptoCurrency: currency.USDT,
		FiatCurrency:   currency.USD,
		SelectType:     "buy",
		Status:         "closed",
		StartTime:      startTime,
		EndTime:        endTime,
		QueryDispute:   true,
		Page:           1,
		PerPage:        20,
	})
	require.NoError(t, err, "GetHistoricalP2POrders must not error")
	if mockTests {
		exp := &P2POrderListResponse{
			List: []*P2POrderInfo{{
				CreationTime:         "2025-12-15 08:27:09",
				ExpiryTime:           "2025-12-15 10:27:09",
				Timestamp:            types.Time(time.Unix(1765787229, 0)),
				Rate:                 1.1,
				Amount:               8,
				Total:                8.8,
				TransactionID:        40000002,
				Status:               "finished",
				CounterpartyRealName: "Counterparty Demo",
				CounterpartyUID:      p2pTestCounterpartyUID,
				CounterpartyNickname: "counterparty_demo",
				SellerRealName:       "Merchant Demo",
				BuyerRealName:        "Counterparty Demo",
				CurrencyType:         currency.USDT,
				WantType:             currency.USD,
				SelectedPayType:      "swift",
				PayOthers:            []*P2POtherPaymentMethod{{PayType: "swift", PayName: "SWIFT International Remittance"}},
				OrderType:            1,
				OrderTags:            []string{},
				ConvertInfo:          &P2PConvertInfo{},
			}},
			TransactionTime: []*P2PTransactionTimeMarker{{}},
			Count:           1,
			ExportedNumber:  1,
		}
		assert.Equal(t, exp, result, "GetHistoricalP2POrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetHistoricalP2POrders should return an order list")
}

func TestP2PCompletedOrderRequestMarshalJSON(t *testing.T) {
	t.Parallel()
	got, err := (&P2PCompletedOrderRequest{CryptoCurrency: currency.USDT, FiatCurrency: currency.USD}).MarshalJSON()
	require.NoError(t, err, "MarshalJSON must not error")
	assert.JSONEq(t, `{"crypto_currency":"USDT","fiat_currency":"USD"}`, string(got), "unset optional fields should be omitted")

	got, err = (&P2PCompletedOrderRequest{CryptoCurrency: currency.USDT, FiatCurrency: currency.USD, StartTime: time.Unix(1764547200, 0), EndTime: time.Unix(1767139199, 0), QueryDispute: true}).MarshalJSON()
	require.NoError(t, err, "MarshalJSON must not error")
	assert.JSONEq(t, `{"crypto_currency":"USDT","fiat_currency":"USD","start_time":1764547200,"end_time":1767139199,"query_dispute":1}`, string(got), "times should be sent as Unix seconds and the dispute flag as 1")
}

func TestGetP2POrderDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetP2POrderDetails(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetP2POrderDetails must reject a nil request")
	_, err = e.GetP2POrderDetails(t.Context(), &GetP2POrderDetailsRequest{})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetP2POrderDetails must require a transaction ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetP2POrderDetails(t.Context(), &GetP2POrderDetailsRequest{TransactionID: p2pTestTransactionID})
	require.NoError(t, err, "GetP2POrderDetails must not error")
	if mockTests {
		exp := &P2POrderDetailResponse{
			IsSell:               true,
			TransactionID:        p2pTestTransactionID,
			OrderID:              p2pTestAdvertisementID,
			CreationTime:         types.Time(time.Unix(1767530241, 0)),
			PaymentDeadline:      types.Time(time.Unix(1767531441, 0)),
			RemainingPaymentTime: types.DurationSeconds(-30 * time.Second),
			CurrencyType:         currency.USDT,
			WantType:             currency.USD,
			FiatSymbol:           "$",
			Rate:                 1.23,
			Amount:               3,
			Total:                3.69,
			Status:               "paid",
			DisputeTime:          types.Time(time.Unix(1767531441, 0)),
			Cancelable:           true,
			TradeTips:            "Please pay from an account under your real name",
			ShowBank:             true,
			BankName:             "Demo Bank",
			BankBranch:           "Main Branch",
			BankAccount:          "****1234",
			BankHolderRealName:   "Merchant Demo",
			ShowOthers:           true,
			PayOthers: []*P2POtherPaymentMethod{{
				ID:                 "10002",
				AccountDescription: "Business USD account",
				PayType:            "swift",
				PayName:            "SWIFT International Remittance",
				Account:            "****5678",
				Memo:               "Use order txid as reference",
				TradeTips:          "Pay before the deadline",
			}},
			SelectedPayType:      "bank",
			CounterpartyUID:      p2pTestCounterpartyUID,
			CounterpartyNickname: "counterparty_demo",
			CounterpartyRealName: "Counterparty Demo",
			HaveTraded:           true,
			UnreadMessages:       2,
			PaymentVoucherURLs:   []string{"https://example.com/p2p/voucher.png"},
			PaidTime:             types.Time(time.Unix(1767530257, 0)),
			OwnRealName:          "Merchant Demo",
			OrderType:            1,
			SupportedPayTypes:    []string{"bank", "swift"},
		}
		assert.Equal(t, exp, result, "GetP2POrderDetails should decode every field")
		return
	}
	assert.NotNil(t, result, "GetP2POrderDetails should return an order")
}

func TestConfirmP2PPayment(t *testing.T) {
	t.Parallel()
	err := e.ConfirmP2PPayment(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ConfirmP2PPayment must reject a nil request")
	err = e.ConfirmP2PPayment(t.Context(), &ConfirmP2PPaymentRequest{})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "ConfirmP2PPayment must require a transaction ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	err = e.ConfirmP2PPayment(t.Context(), &ConfirmP2PPaymentRequest{TransactionID: "40000001", PaymentMethod: "bank"})
	assert.NoError(t, err, "ConfirmP2PPayment should not error")
}

func TestConfirmP2PReceipt(t *testing.T) {
	t.Parallel()
	err := e.ConfirmP2PReceipt(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ConfirmP2PReceipt must reject a nil request")
	err = e.ConfirmP2PReceipt(t.Context(), &ConfirmP2PReceiptRequest{})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "ConfirmP2PReceipt must require a transaction ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	err = e.ConfirmP2PReceipt(t.Context(), &ConfirmP2PReceiptRequest{TransactionID: "40000001"})
	assert.NoError(t, err, "ConfirmP2PReceipt should not error")
}

func TestCancelP2POrder(t *testing.T) {
	t.Parallel()
	err := e.CancelP2POrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelP2POrder must reject a nil request")
	err = e.CancelP2POrder(t.Context(), &CancelP2POrderRequest{})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelP2POrder must require a transaction ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	err = e.CancelP2POrder(t.Context(), &CancelP2POrderRequest{TransactionID: "40000001", ReasonID: 7, ReasonMemo: "Canceled after agreement with the counterparty"})
	assert.NoError(t, err, "CancelP2POrder should not error")
}

func TestCancelP2POrderRequestMarshalJSON(t *testing.T) {
	t.Parallel()
	got, err := json.Marshal(&CancelP2POrderRequest{TransactionID: "40000001", ReasonID: 12})
	require.NoError(t, err, "Marshal must not error")
	assert.JSONEq(t, `{"txid":"40000001","reason_id":"12"}`, string(got), "reason ID should be sent as a string")

	got, err = json.Marshal(&CancelP2POrderRequest{TransactionID: "40000001"})
	require.NoError(t, err, "Marshal must not error")
	assert.JSONEq(t, `{"txid":"40000001"}`, string(got), "an unset reason should be omitted")
}

func TestPublishP2PAdOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PublishP2PAdOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PublishP2PAdOrder must reject a nil request")

	arg := &PublishP2PAdRequest{}
	_, err = e.PublishP2PAdOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "PublishP2PAdOrder must require a crypto currency")
	arg.CurrencyType = currency.USDT
	_, err = e.PublishP2PAdOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "PublishP2PAdOrder must require a fiat currency")
	arg.ExchangeType = currency.USD
	arg.Type = 4
	_, err = e.PublishP2PAdOrder(t.Context(), arg)
	require.ErrorIs(t, err, errP2PAdOperationInvalid, "PublishP2PAdOrder must reject an unknown operation")
	arg.Type = P2PAdOperationEditSell
	_, err = e.PublishP2PAdOrder(t.Context(), arg)
	require.ErrorIs(t, err, errP2PAdIDRequired, "PublishP2PAdOrder must require the ad ID when editing")
	arg.Type = P2PAdOperationPublishSell
	_, err = e.PublishP2PAdOrder(t.Context(), arg)
	require.ErrorIs(t, err, errP2PUnitPriceRequired, "PublishP2PAdOrder must require a unit price")
	arg.UnitPrice = 1.1
	_, err = e.PublishP2PAdOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountMustBeSet, "PublishP2PAdOrder must require an ad amount")
	arg.Number = 100
	_, err = e.PublishP2PAdOrder(t.Context(), arg)
	require.ErrorIs(t, err, errP2PPayTypeRequired, "PublishP2PAdOrder must require payment types")
	arg.PayType = "bank,swift"
	_, err = e.PublishP2PAdOrder(t.Context(), arg)
	require.ErrorIs(t, err, errP2PTradeLimitRequired, "PublishP2PAdOrder must require crypto per-order limits by default")
	arg.LimitBasis = P2PLimitBasisFiat
	arg.MinAmount, arg.MaxAmount = 10, 500
	_, err = e.PublishP2PAdOrder(t.Context(), arg)
	require.ErrorIs(t, err, errP2PTradeLimitRequired, "PublishP2PAdOrder must require fiat per-order limits for a fiat limit basis")
	arg.LimitBasis = 2
	_, err = e.PublishP2PAdOrder(t.Context(), arg)
	require.ErrorIs(t, err, errP2PLimitBasisInvalid, "PublishP2PAdOrder must reject an unknown limit basis")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	arg.LimitBasis = P2PLimitBasisFiat
	arg.MinAmount, arg.MaxAmount = 0, 0
	arg.FiatMinAmount, arg.FiatMaxAmount = 100, 1000
	arg.PayTypeJSON = `{"bank":"10001","swift":"10002"}`
	arg.RateFixed = "1"
	arg.ExpireMinutes = 20
	arg.MinCompletedLimit = -1
	arg.TradeTips = "Please pay from an account under your own name"
	result, err := e.PublishP2PAdOrder(t.Context(), arg)
	require.NoError(t, err, "PublishP2PAdOrder must not error")
	if !mockTests {
		return
	}
	assert.Equal(t, &P2PPublishAdResponse{}, result, "PublishP2PAdOrder should return empty risk details when accepted")

	arg.TradeTips = "Contact me off platform"
	result, err = e.PublishP2PAdOrder(t.Context(), arg)
	require.ErrorContains(t, err, "70305102", "PublishP2PAdOrder must return the risk control error")
	exp := &P2PPublishAdResponse{
		RiskEvent: &P2PRiskEvent{
			Type:            "modal",
			Title:           "Risk warning",
			Message:         "Trade terms must not divert users off the platform",
			Actions:         []*P2PRiskEventAction{{ActionType: "close", Title: "Close", ActionData: json.RawMessage(`{}`)}},
			ContentRiskType: "trade_tips",
			TradeTips:       "Trade terms must not divert users off the platform",
		},
	}
	assert.Equal(t, exp, result, "PublishP2PAdOrder should return the risk details alongside the error")
}

func TestPublishP2PAdRequestMarshalJSON(t *testing.T) {
	t.Parallel()
	got, err := json.Marshal(&PublishP2PAdRequest{CurrencyType: currency.USDT, ExchangeType: currency.USD, UnitPrice: 1.1, Number: 100, PayType: "bank", MinAmount: 10, MaxAmount: 500})
	require.NoError(t, err, "Marshal must not error")
	assert.JSONEq(t, `{"currencyType":"USDT","exchangeType":"USD","type":"0","unitPrice":"1.1","number":"100","payType":"bank","minAmount":"10","maxAmount":"500"}`, string(got), "a publish sell ad should send its operation type and omit unset fields")
}

func TestUpdateP2PAdStatus(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateP2PAdStatus(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "UpdateP2PAdStatus must reject a nil request")
	_, err = e.UpdateP2PAdStatus(t.Context(), &UpdateP2PAdStatusRequest{Status: 1})
	require.ErrorIs(t, err, errP2PAdIDRequired, "UpdateP2PAdStatus must require an ad ID")
	_, err = e.UpdateP2PAdStatus(t.Context(), &UpdateP2PAdStatusRequest{AdvertisementID: p2pTestAdvertisementID, Status: 2})
	require.ErrorIs(t, err, errP2PAdStatusInvalid, "UpdateP2PAdStatus must reject an unknown status")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.UpdateP2PAdStatus(t.Context(), &UpdateP2PAdStatusRequest{AdvertisementID: p2pTestAdvertisementID, Status: 3})
	require.NoError(t, err, "UpdateP2PAdStatus must not error")
	if mockTests {
		assert.Equal(t, &P2PUpdateAdStatusResponse{Status: 3}, result, "UpdateP2PAdStatus should decode every field")
		return
	}
	assert.NotNil(t, result, "UpdateP2PAdStatus should return the ad status")
}

func TestGetP2PAdDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetP2PAdDetails(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetP2PAdDetails must reject a nil request")
	_, err = e.GetP2PAdDetails(t.Context(), &GetP2PAdDetailsRequest{})
	require.ErrorIs(t, err, errP2PAdIDRequired, "GetP2PAdDetails must require an ad ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetP2PAdDetails(t.Context(), &GetP2PAdDetailsRequest{AdvertisementID: "2124000001"})
	require.NoError(t, err, "GetP2PAdDetails must not error")
	if mockTests {
		exp := &P2PAdDetailResponse{
			Rate:                  0.908,
			Type:                  "sell",
			Amount:                100,
			MinAmount:             10,
			MaxAmount:             500,
			FiatMinAmount:         9.08,
			FiatMaxAmount:         454,
			LimitBasisText:        "crypto",
			Total:                 90.8,
			PayBank:               true,
			PayTypeNumbers:        "2,5",
			PayTypeJSON:           `{"bank":"10001","swift":"10002"}`,
			OrderID:               p2pTestAdvertisementID,
			Timestamp:             types.Time(time.Unix(1766988136, 0)),
			CurrencyType:          currency.USDT,
			WantType:              currency.USD,
			TradeTips:             "Please pay from an account under your real name",
			AutoReply:             "Thanks for your order. I will process it soon.",
			RateReferenceID:       3,
			RateOffset:            0.5,
			Status:                "OPEN",
			FloatTrend:            1,
			ExpireMinutes:         20,
			TierLimit:             1,
			RegistrationTimeLimit: 30,
			AdvertisersLimit:      true,
			PolymarketLimit:       true,
			MinCompletedLimit:     -1,
			MaxCompletedLimit:     100,
			UserOrdersLimit:       -1,
			CompletedRateLimit:    -1,
			HidePayment:           true,
		}
		assert.Equal(t, exp, result, "GetP2PAdDetails should decode every field")
		return
	}
	assert.NotNil(t, result, "GetP2PAdDetails should return the ad")
}

func TestGetMyP2PAds(t *testing.T) {
	t.Parallel()
	_, err := e.GetMyP2PAds(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetMyP2PAds must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMyP2PAds(t.Context(), &GetMyP2PAdsRequest{Asset: currency.USDT, FiatUnit: currency.USD, TradeType: "sell"})
	require.NoError(t, err, "GetMyP2PAds must not error")
	if !mockTests {
		assert.NotNil(t, result, "GetMyP2PAds should return the user's ads")
		return
	}
	exp := &P2PMyAdsResponse{Lists: []*P2PMyAdItem{{
		Type:                  "sell",
		Rate:                  1.27,
		OriginalRate:          1.27,
		Amount:                100,
		Total:                 127,
		LimitTotal:            "10~500",
		LimitFiat:             "12.7~635",
		MinAmount:             10,
		MaxAmount:             500,
		PayTypeNumbers:        "2,4",
		PayTypeJSON:           `{"bank":"10001","wu":"10003"}`,
		ExpireMinutes:         45,
		TierLimit:             1,
		AdvertisersLimit:      true,
		RegistrationTimeLimit: 90,
		MinCompletedLimit:     8,
		MaxCompletedLimit:     9,
		UserCountryLimit:      4,
		CompletedRateLimit:    6,
		UserOrdersLimit:       7,
		HidePayment:           true,
		CurrencyType:          currency.USDT,
		WantType:              currency.USD,
		TradeTips:             "Please pay from an account under your real name",
		ID:                    "2124000001",
		Status:                "OFFLIN",
		RateReferenceID:       -1,
		RateFixed:             true,
		AutoReply:             "Thanks for your order. I will process it soon.",
		Timestamp:             types.Time(time.Unix(1767008930, 0)),
	}}}
	assert.Equal(t, exp, result, "GetMyP2PAds should decode every field")

	// Without filters the request body is empty, returning every ad
	result, err = e.GetMyP2PAds(t.Context(), &GetMyP2PAdsRequest{})
	require.NoError(t, err, "GetMyP2PAds must not error without filters")
	assert.Len(t, result.Lists, 1, "GetMyP2PAds should return the user's ads")
}

func TestGetP2PAdList(t *testing.T) {
	t.Parallel()
	_, err := e.GetP2PAdList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetP2PAdList must reject a nil request")
	_, err = e.GetP2PAdList(t.Context(), &GetP2PAdsListRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetP2PAdList must require an asset")
	_, err = e.GetP2PAdList(t.Context(), &GetP2PAdsListRequest{Asset: currency.USDT})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetP2PAdList must require a fiat unit")
	_, err = e.GetP2PAdList(t.Context(), &GetP2PAdsListRequest{Asset: currency.USDT, FiatUnit: currency.USD})
	require.ErrorIs(t, err, errP2PTradeTypeRequired, "GetP2PAdList must require a trade type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetP2PAdList(t.Context(), &GetP2PAdsListRequest{Asset: currency.USDT, FiatUnit: currency.USD, TradeType: "sell"})
	require.NoError(t, err, "GetP2PAdList must not error")
	if mockTests {
		exp := []*P2PAdListItemResponse{
			{
				Index:                   1,
				Asset:                   currency.USDT,
				FiatUnit:                currency.USD,
				AdvertisementID:         p2pTestAdvertisementID,
				Price:                   0.8,
				SurplusAmount:           4975.5,
				MaxSingleTransferAmount: 4975.5,
				MinSingleTransferAmount: 1,
				FiatMinAmount:           0.8,
				FiatMaxAmount:           3980.4,
				LimitBasisText:          "crypto",
				TradeMethods: []*P2PAdsTradeMethod{{
					IconURL:         "https://www.gate.com/images/payment/bank.png",
					Identifier:      "bank",
					PayID:           "10001",
					PayType:         "bank",
					TradeMethodName: "Bank Transfer",
				}},
				AdvertiserNickname: "merchant_demo_01",
			},
			{
				Index:                   2,
				Asset:                   currency.USDT,
				FiatUnit:                currency.USD,
				AdvertisementID:         2124000002,
				Price:                   0.88,
				SurplusAmount:           12379.15,
				MaxSingleTransferAmount: 12379.15,
				MinSingleTransferAmount: 10,
				FiatMinAmount:           8.8,
				FiatMaxAmount:           10893.65,
				LimitBasis:              P2PLimitBasisFiat,
				LimitBasisText:          "fiat",
				TradeMethods: []*P2PAdsTradeMethod{{
					IconURL:         "https://www.gate.com/images/payment/swift.png",
					Identifier:      "swift",
					PayID:           "10002",
					PayType:         "swift",
					TradeMethodName: "SWIFT",
				}},
				AdvertiserNickname: "merchant_demo_02",
			},
		}
		assert.Equal(t, exp, result, "GetP2PAdList should decode every field")
		return
	}
	assert.NotNil(t, result, "GetP2PAdList should return ads")
}

func TestGetP2PChatHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetP2PChatHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetP2PChatHistory must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetP2PChatHistory(t.Context(), &GetP2PChatHistoryRequest{
		TransactionID: p2pTestTransactionID,
		LastReceived:  time.Unix(1767009884, 0),
		FirstReceived: time.Unix(1767009000, 0),
	})
	require.NoError(t, err, "GetP2PChatHistory must not error")
	if mockTests {
		exp := &P2PChatMessagesResponse{
			Messages: []*P2PMessageDetail{
				{
					IsSell:       true,
					Message:      "Please contact me outside Gate",
					Username:     "Self",
					Timestamp:    types.Time(time.Unix(1767009001, 0)),
					RiskType:     1,
					ToastMessage: "This message may contain security risks.",
				},
				{
					IsSell:      true,
					MessageType: 5,
					Username:    "Other",
					Timestamp:   types.Time(time.Unix(1767009300, 0)),
					MessageObject: &P2PMessageObject{
						Status:         "CANCEL",
						Text:           "The buyer has canceled the order.",
						PaymentVoucher: []string{"https://example.com/p2p/voucher.png"},
						ReasonID:       1,
						ToastID:        1,
						ReasonMemo:     "I don't want to buy the coins anymore.",
						CancelTime:     types.Time(time.Unix(1767009300, 0)),
						SellerConfirm:  1,
					},
				},
				{
					IsSell:      true,
					MessageType: 4,
					Username:    "Other",
					UID:         p2pTestCounterpartyUID,
					Timestamp:   types.Time(time.Unix(1767009500, 0)),
					MessageObject: &P2PMessageObject{
						ID:                 "10002",
						AccountDescription: "Business USD account",
						PaymentType:        "swift",
						File:               "https://example.com/p2p/swift.png",
						FileKey:            "swift-file-key",
						Account:            "****5678",
						Memo:               "Use order txid as reference",
						Code:               "SWIFTCODE",
						MemoExtended:       "Extra note",
						TradeTips:          "Please pay from an account under your real name",
						RealName:           "Merchant Demo",
						IsDeleted:          true,
						PayName:            "SWIFT International Remittance",
					},
				},
				{
					IsSell:      true,
					MessageType: 1,
					Message:     "https://example.com/p2p/chat/receipt.png",
					Username:    "Other",
					Timestamp:   types.Time(time.Unix(1767009700, 0)),
					Picture:     "https://example.com/p2p/chat/receipt.png",
					FileKey:     p2pTestChatFileKey,
					FileType:    "image",
					Type:        1,
				},
			},
			Memo:              "Pay within 15 minutes",
			HasHistory:        true,
			TransactionID:     p2pTestTransactionID,
			LatestMessageTime: types.Time(time.Unix(1767009700, 0)),
			OrderStatus:       "CANCEL",
		}
		assert.Equal(t, exp, result, "GetP2PChatHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetP2PChatHistory should return chat history")
}

func TestGetP2PChatHistoryRequestMarshalJSON(t *testing.T) {
	t.Parallel()
	got, err := (&GetP2PChatHistoryRequest{}).MarshalJSON()
	require.NoError(t, err, "MarshalJSON must not error")
	assert.JSONEq(t, `{}`, string(got), "unset fields should be omitted")

	got, err = (&GetP2PChatHistoryRequest{TransactionID: p2pTestTransactionID, LastReceived: time.Unix(1767009884, 0), FirstReceived: time.Unix(1767009000, 0)}).MarshalJSON()
	require.NoError(t, err, "MarshalJSON must not error")
	assert.JSONEq(t, `{"txid":40000001,"lastreceived":1767009884,"firstreceived":1767009000}`, string(got), "message times should be sent as Unix seconds")
}

func TestSendP2PChatMessage(t *testing.T) {
	t.Parallel()
	_, err := e.SendP2PChatMessage(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SendP2PChatMessage must reject a nil request")
	_, err = e.SendP2PChatMessage(t.Context(), &SendP2PChatMessageRequest{})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "SendP2PChatMessage must require a transaction ID")
	_, err = e.SendP2PChatMessage(t.Context(), &SendP2PChatMessageRequest{TransactionID: p2pTestTransactionID})
	require.ErrorIs(t, err, errP2PMessageRequired, "SendP2PChatMessage must require a message")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SendP2PChatMessage(t.Context(), &SendP2PChatMessageRequest{TransactionID: p2pTestTransactionID, Message: "Payment completed, please check"})
	require.NoError(t, err, "SendP2PChatMessage must not error")
	if mockTests {
		exp := &P2PSendMessageResponse{
			SentTime:       types.Time(time.UnixMilli(1767009886638)),
			TransactionID:  p2pTestTransactionID,
			ConversationID: "1000001_1000002",
			RiskType:       1,
			ToastMessage:   "This message may contain security risks.",
		}
		assert.Equal(t, exp, result, "SendP2PChatMessage should decode every field")
		return
	}
	assert.NotNil(t, result, "SendP2PChatMessage should return the sent message")
}

func TestUploadP2PChatFile(t *testing.T) {
	t.Parallel()
	_, err := e.UploadP2PChatFile(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "UploadP2PChatFile must reject a nil request")
	_, err = e.UploadP2PChatFile(t.Context(), &UploadP2PChatFileRequest{})
	require.ErrorIs(t, err, errP2PImageTypeRequired, "UploadP2PChatFile must require a content type")
	_, err = e.UploadP2PChatFile(t.Context(), &UploadP2PChatFileRequest{ImageContentType: "image/png"})
	require.ErrorIs(t, err, errP2PImageDataRequired, "UploadP2PChatFile must require file content")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.UploadP2PChatFile(t.Context(), &UploadP2PChatFileRequest{
		ImageContentType: "image/png",
		Base64Img:        "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==",
	})
	require.NoError(t, err, "UploadP2PChatFile must not error")
	if mockTests {
		assert.Equal(t, &P2PUploadFileResponse{FileKey: p2pTestChatFileKey}, result, "UploadP2PChatFile should decode every field")
		return
	}
	assert.NotEmpty(t, result.FileKey, "UploadP2PChatFile should return a file key")
}
