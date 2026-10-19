package coinbase

import (
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	// testConvertTradeID is the trade the convert fixtures quote, commit and return
	testConvertTradeID = "0b2c5a0e-2a6c-4ee3-9d0a-0c3f8a1b5e77"
	// testConvertFromAccount and testConvertToAccount are the accounts the convert tests convert between, named by
	// their currencies as Coinbase documents
	testConvertFromAccount = "USD"
	testConvertToAccount   = "USDC"
	// testConvertAmount is the USD the mock fixtures convert, and testLiveConvertAmount the USD live tests convert and
	// convert back
	testConvertAmount     = 100
	testLiveConvertAmount = 1
)

// testConvertTrade returns the convert trade the convert fixtures return, in a status
func testConvertTrade(status string) *ConvertTrade {
	return &ConvertTrade{
		ID:     "0b2c5a0e-2a6c-4ee3-9d0a-0c3f8a1b5e77",
		Status: status,
		UserEnteredAmount: Amount{
			Value:    100,
			Currency: currency.USD,
		},
		Amount: Amount{
			Value:    100,
			Currency: currency.USDC,
		},
		Subtotal: Amount{
			Value:    100,
			Currency: currency.USD,
		},
		Total: Amount{
			Value:    100,
			Currency: currency.USD,
		},
		Fees: []ConvertFee{
			{
				Title:       "Coinbase fee",
				Description: "No fee is charged to convert USD to USDC",
				Amount: Amount{
					Currency: currency.USD,
				},
				Label: "Coinbase fee",
				Disclosure: &ConvertFeeDisclosure{
					Title:       "Fees",
					Description: "Learn how Coinbase charges fees",
					Link: &ConvertLink{
						Text: "Learn more",
						URL:  "https://help.coinbase.com/en/coinbase/trading-and-funding/pricing-and-fees/fees",
					},
				},
			},
		},
		TotalFee: &ConvertFee{
			Title: "Total fee",
			Amount: Amount{
				Currency: currency.USD,
			},
			Label: "Total fee",
		},
		Source: &ConvertPaymentMethod{
			Type: "LEDGER_ACCOUNT",
			LedgerAccount: &ConvertLedgerAccount{
				AccountID: "81b5dca0-5775-4a21-b872-fc850046c99b",
				Currency:  currency.USD,
				Owner: &ConvertLedgerAccountOwner{
					UUID:     "3e44ed34-0be4-5e1b-a1f3-2b9e0bd4d5b5",
					UserUUID: "3e44ed34-0be4-5e1b-a1f3-2b9e0bd4d5b5",
					Type:     "RETAIL",
				},
				AccountUUID: "81b5dca0-5775-4a21-b872-fc850046c99b",
				AccountName: "Cash (USD)",
			},
		},
		Target: &ConvertPaymentMethod{
			Type: "LEDGER_ACCOUNT",
			LedgerAccount: &ConvertLedgerAccount{
				AccountID: "66f975a6-bb2e-44be-82e9-cd8669e404b0",
				Currency:  currency.USDC,
				Owner: &ConvertLedgerAccountOwner{
					UUID:     "3e44ed34-0be4-5e1b-a1f3-2b9e0bd4d5b5",
					UserUUID: "3e44ed34-0be4-5e1b-a1f3-2b9e0bd4d5b5",
					Type:     "RETAIL",
				},
				AccountUUID: "66f975a6-bb2e-44be-82e9-cd8669e404b0",
				AccountName: "USDC Wallet",
			},
		},
		UnitPrice: &ConvertUnitPrice{
			TargetToFiat: &ConvertScaledAmount{
				Amount: Amount{
					Value:    1,
					Currency: currency.USD,
				},
				Scale: 2,
			},
			TargetToSource: &ConvertScaledAmount{
				Amount: Amount{
					Value:    1,
					Currency: currency.USD,
				},
				Scale: 2,
			},
			SourceToFiat: &ConvertScaledAmount{
				Amount: Amount{
					Value:    1,
					Currency: currency.USD,
				},
				Scale: 2,
			},
		},
		UserWarnings:   []ConvertUserWarning{},
		UserReference:  "K7QX2M4P",
		SourceCurrency: currency.USD,
		TargetCurrency: currency.USDC,
		SourceID:       "81b5dca0-5775-4a21-b872-fc850046c99b",
		TargetID:       "66f975a6-bb2e-44be-82e9-cd8669e404b0",
		SubscriptionInfo: &ConvertSubscriptionInfo{
			UsedZeroFeeTrading: Amount{
				Currency: currency.USD,
			},
			RemainingFreeTradingVolume: Amount{
				Currency: currency.USD,
			},
			MaximumFreeTradingVolume: Amount{
				Currency: currency.USD,
			},
			FeeWithoutSubscriptionBenefit: Amount{
				Currency: currency.USD,
			},
			PaymentMethodFeeWithoutSubscriptionBenefit: Amount{
				Currency: currency.USD,
			},
		},
		ExchangeRate: Amount{
			Value:    1,
			Currency: currency.USDC,
		},
		TaxDetails: []ConvertTax{},
		TradeIncentiveInfo: &ConvertTradeIncentive{
			FeeWithoutIncentive: Amount{
				Currency: currency.USD,
			},
		},
		TotalFeeWithoutTax: &ConvertFee{
			Title: "Total fee",
			Amount: Amount{
				Currency: currency.USD,
			},
			Label: "Total fee",
		},
		FiatDenotedTotal: Amount{
			Value:    100,
			Currency: currency.USD,
		},
	}
}

// testConvertQuote quotes a live conversion of testLiveConvertAmount between two accounts
func testConvertQuote(t *testing.T, fromAccount, toAccount string) *ConvertTrade {
	t.Helper()
	quote, err := e.CreateConvertQuote(t.Context(), &CreateConvertQuoteRequest{FromAccount: fromAccount, ToAccount: toAccount, Amount: testLiveConvertAmount})
	require.NoError(t, err, "CreateConvertQuote must not error")
	return quote
}

func TestCommitConvertTrade(t *testing.T) {
	t.Parallel()
	_, err := e.CommitConvertTrade(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CommitConvertTrade must reject a nil request")
	_, err = e.CommitConvertTrade(t.Context(), &ConvertTradeRequest{FromAccount: testConvertFromAccount, ToAccount: testConvertToAccount})
	require.ErrorIs(t, err, errConvertTradeIDRequired, "CommitConvertTrade must require a trade ID")
	_, err = e.CommitConvertTrade(t.Context(), &ConvertTradeRequest{TradeID: testConvertTradeID, ToAccount: testConvertToAccount})
	require.ErrorIs(t, err, errConvertAccountRequired, "CommitConvertTrade must require the account converted from")
	_, err = e.CommitConvertTrade(t.Context(), &ConvertTradeRequest{TradeID: testConvertTradeID, FromAccount: testConvertFromAccount})
	require.ErrorIs(t, err, errConvertAccountRequired, "CommitConvertTrade must require the account converted to")

	skipLiveUnlessCanManipulate(t)
	req := &ConvertTradeRequest{TradeID: testConvertTradeID, FromAccount: testConvertFromAccount, ToAccount: testConvertToAccount}
	if !mockTests {
		req.TradeID = testConvertQuote(t, testConvertFromAccount, testConvertToAccount).ID
	}
	resp, err := e.CommitConvertTrade(t.Context(), req)
	require.NoError(t, err, "CommitConvertTrade must not error")
	if mockTests {
		exp := testConvertTrade(ConvertTradeStatusStarted)
		assert.Equal(t, exp, resp, "CommitConvertTrade should decode every field")
	} else {
		assert.Equal(t, req.TradeID, resp.ID, "CommitConvertTrade should commit the quoted trade")
		back := testConvertQuote(t, testConvertToAccount, testConvertFromAccount)
		_, err = e.CommitConvertTrade(t.Context(), &ConvertTradeRequest{TradeID: back.ID, FromAccount: testConvertToAccount, ToAccount: testConvertFromAccount})
		assert.NoError(t, err, "CommitConvertTrade should convert the funds back")
	}

	req = &ConvertTradeRequest{TradeID: testConvertTradeID, FromAccount: testConvertFromAccount, ToAccount: testConvertToAccount}
	_, err = newResponseServerExchange(t, `null`).CommitConvertTrade(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "CommitConvertTrade should error for a null response")
	_, err = newResponseServerExchange(t, `{"trade":null}`).CommitConvertTrade(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "CommitConvertTrade should error for a null trade")
	_, err = newUnavailableServerExchange(t).CommitConvertTrade(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "CommitConvertTrade should return the request failure")
}

func TestCreateConvertQuote(t *testing.T) {
	t.Parallel()
	_, err := e.CreateConvertQuote(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateConvertQuote must reject a nil request")
	_, err = e.CreateConvertQuote(t.Context(), &CreateConvertQuoteRequest{ToAccount: testConvertToAccount, Amount: testConvertAmount})
	require.ErrorIs(t, err, errConvertAccountRequired, "CreateConvertQuote must require the account converted from")
	_, err = e.CreateConvertQuote(t.Context(), &CreateConvertQuoteRequest{FromAccount: testConvertFromAccount, Amount: testConvertAmount})
	require.ErrorIs(t, err, errConvertAccountRequired, "CreateConvertQuote must require the account converted to")
	_, err = e.CreateConvertQuote(t.Context(), &CreateConvertQuoteRequest{FromAccount: testConvertFromAccount, ToAccount: testConvertToAccount})
	require.ErrorIs(t, err, errInvalidAmount, "CreateConvertQuote must require an amount")

	skipLiveUnlessCanManipulate(t)
	req := &CreateConvertQuoteRequest{FromAccount: testConvertFromAccount, ToAccount: testConvertToAccount, Amount: testConvertAmount}
	if !mockTests {
		req.Amount = testLiveConvertAmount
	}
	resp, err := e.CreateConvertQuote(t.Context(), req)
	require.NoError(t, err, "CreateConvertQuote must not error")
	if mockTests {
		exp := testConvertTrade(ConvertTradeStatusCreated)
		assert.Equal(t, exp, resp, "CreateConvertQuote should decode every field")
	} else {
		assert.NotEmpty(t, resp.ID, "CreateConvertQuote should return the quote's trade ID")
	}

	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if assert.NoError(t, err, "Reading the request body should not error") {
			assert.JSONEq(t, `{"from_account":"USD","to_account":"USDC","amount":"0.5","trade_incentive_metadata":{"user_incentive_id":"incentive","code_val":"PROMO"}}`, string(body), "CreateConvertQuote should send the trade incentive")
		}
		_, err = w.Write([]byte(`{"trade":{"id":"quote"}}`))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	_, err = ex.CreateConvertQuote(t.Context(), &CreateConvertQuoteRequest{FromAccount: testConvertFromAccount, ToAccount: testConvertToAccount, Amount: 0.5, UserIncentiveID: "incentive", CodeValue: "PROMO"})
	assert.NoError(t, err, "CreateConvertQuote should not error with a trade incentive")

	req = &CreateConvertQuoteRequest{FromAccount: testConvertFromAccount, ToAccount: testConvertToAccount, Amount: testConvertAmount}
	_, err = newResponseServerExchange(t, `null`).CreateConvertQuote(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "CreateConvertQuote should error for a null response")
	_, err = newResponseServerExchange(t, `{"trade":null}`).CreateConvertQuote(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "CreateConvertQuote should error for a null trade")
	_, err = newUnavailableServerExchange(t).CreateConvertQuote(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "CreateConvertQuote should return the request failure")
}

func TestGetConvertTrade(t *testing.T) {
	t.Parallel()
	_, err := e.GetConvertTrade(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetConvertTrade must reject a nil request")
	_, err = e.GetConvertTrade(t.Context(), &ConvertTradeRequest{FromAccount: testConvertFromAccount, ToAccount: testConvertToAccount})
	require.ErrorIs(t, err, errConvertTradeIDRequired, "GetConvertTrade must require a trade ID")
	_, err = e.GetConvertTrade(t.Context(), &ConvertTradeRequest{TradeID: testConvertTradeID, ToAccount: testConvertToAccount})
	require.ErrorIs(t, err, errConvertAccountRequired, "GetConvertTrade must require the account converted from")
	_, err = e.GetConvertTrade(t.Context(), &ConvertTradeRequest{TradeID: testConvertTradeID, FromAccount: testConvertFromAccount})
	require.ErrorIs(t, err, errConvertAccountRequired, "GetConvertTrade must require the account converted to")

	// A live trade to get needs a quote, which needs the trade permission
	skipLiveUnlessCanManipulate(t)
	req := &ConvertTradeRequest{TradeID: testConvertTradeID, FromAccount: testConvertFromAccount, ToAccount: testConvertToAccount}
	if !mockTests {
		req.TradeID = testConvertQuote(t, testConvertFromAccount, testConvertToAccount).ID
	}
	resp, err := e.GetConvertTrade(t.Context(), req)
	require.NoError(t, err, "GetConvertTrade must not error")
	if mockTests {
		exp := testConvertTrade(ConvertTradeStatusCompleted)
		assert.Equal(t, exp, resp, "GetConvertTrade should decode every field")
	} else {
		assert.Equal(t, req.TradeID, resp.ID, "GetConvertTrade should return the quoted trade")
	}

	req = &ConvertTradeRequest{TradeID: testConvertTradeID, FromAccount: testConvertFromAccount, ToAccount: testConvertToAccount}
	_, err = newResponseServerExchange(t, `null`).GetConvertTrade(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetConvertTrade should error for a null response")
	_, err = newResponseServerExchange(t, `{"trade":null}`).GetConvertTrade(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetConvertTrade should error for a null trade")
	_, err = newUnavailableServerExchange(t).GetConvertTrade(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetConvertTrade should return the request failure")
}

func TestCheckConvertAccounts(t *testing.T) {
	t.Parallel()
	assert.NoError(t, checkConvertAccounts(testConvertFromAccount, testConvertToAccount), "checkConvertAccounts should accept both accounts")
	assert.ErrorIs(t, checkConvertAccounts("", testConvertToAccount), errConvertAccountRequired, "checkConvertAccounts should require the account converted from")
	assert.ErrorIs(t, checkConvertAccounts(testConvertFromAccount, ""), errConvertAccountRequired, "checkConvertAccounts should require the account converted to")
}
