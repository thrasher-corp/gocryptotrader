package coinbase

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

func TestGetCurrencies(t *testing.T) {
	t.Parallel()
	resp, err := e.GetCurrencies(t.Context())
	require.NoError(t, err, "GetCurrencies must not error")
	if mockTests {
		exp := []ExchangeCurrencyResponse{
			{
				ID:               currency.NewCode("ACX"),
				Name:             "Across Protocol",
				MinimumSize:      0.00000001,
				Status:           "delisted",
				MaximumPrecision: 0.00000001,
				ConvertibleTo:    []currency.Code{},
				Details: ExchangeCurrencyDetails{
					Type:                    ExchangeCurrencyTypeCrypto,
					NetworkConfirmations:    14,
					SortOrder:               500,
					CryptoAddressLink:       "https://etherscan.io/token/0x44108f0223A3C3028F5Fe7AEC7f9bb2E66beF82F?a={{address}}",
					CryptoTransactionLink:   "https://etherscan.io/tx/0x{{txId}}",
					PushPaymentMethods:      []string{"crypto"},
					GroupTypes:              []string{},
					MinimumWithdrawalAmount: 0.00000001,
					MaximumWithdrawalAmount: 17000000,
				},
				DefaultNetwork: "ethereum",
				SupportedNetworks: []ExchangeCurrencyNetwork{
					{
						ID:                      "ethereum",
						Name:                    "Ethereum",
						Status:                  "delisted",
						ContractAddress:         "0x44108f0223A3C3028F5Fe7AEC7f9bb2E66beF82F",
						CryptoAddressLink:       "https://etherscan.io/token/0x44108f0223A3C3028F5Fe7AEC7f9bb2E66beF82F?a={{address}}",
						CryptoTransactionLink:   "https://etherscan.io/tx/0x{{txId}}",
						MinimumWithdrawalAmount: 0.00000001,
						MaximumWithdrawalAmount: 17000000,
						NetworkConfirmations:    14,
						IsEVMNetwork:            true,
					},
				},
				DisplayName: "ACX",
			},
			{
				ID:               currency.NewCode("LINEA"),
				Name:             "Linea",
				MinimumSize:      0.00000001,
				Status:           "online",
				Message:          "We’ve started accepting deposits, trading will begin shortly.",
				MaximumPrecision: 0.00000001,
				ConvertibleTo:    []currency.Code{},
				Details: ExchangeCurrencyDetails{
					Type:                    ExchangeCurrencyTypeCrypto,
					NetworkConfirmations:    14,
					SortOrder:               500,
					CryptoAddressLink:       "https://etherscan.io/token/0x1789e0043623282d5dcc7f213d703c6d8bafbb04?a={{address}}",
					CryptoTransactionLink:   "https://etherscan.io/tx/0x{{txId}}",
					PushPaymentMethods:      []string{"crypto"},
					GroupTypes:              []string{},
					MinimumWithdrawalAmount: 0.00000001,
					MaximumWithdrawalAmount: 14490000,
				},
				DefaultNetwork: "ethereum",
				SupportedNetworks: []ExchangeCurrencyNetwork{
					{
						ID:                      "ethereum",
						Name:                    "Ethereum",
						Status:                  "online",
						ContractAddress:         "0x1789e0043623282d5dcc7f213d703c6d8bafbb04",
						CryptoAddressLink:       "https://etherscan.io/token/0x1789e0043623282d5dcc7f213d703c6d8bafbb04?a={{address}}",
						CryptoTransactionLink:   "https://etherscan.io/tx/0x{{txId}}",
						MinimumWithdrawalAmount: 0.00000001,
						MaximumWithdrawalAmount: 14490000,
						NetworkConfirmations:    14,
						IsEVMNetwork:            true,
					},
				},
				DisplayName: "LINEA",
			},
			{
				ID:               currency.NewCode("POL"),
				Name:             "Polygon Ecosystem Token",
				MinimumSize:      0.00000001,
				Status:           "online",
				MaximumPrecision: 0.00000001,
				ConvertibleTo:    []currency.Code{},
				Details: ExchangeCurrencyDetails{
					Type:                    ExchangeCurrencyTypeCrypto,
					NetworkConfirmations:    14,
					SortOrder:               500,
					CryptoAddressLink:       "https://etherscan.io/token/0x455e53cbb86018ac2b8092fdcd39d8444affc3f6?a={{address}}",
					CryptoTransactionLink:   "https://etherscan.io/tx/0x{{txId}}",
					PushPaymentMethods:      []string{"crypto"},
					GroupTypes:              []string{},
					MinimumWithdrawalAmount: 0.00000001,
					MaximumWithdrawalAmount: 5000000,
				},
				DefaultNetwork: "polygon",
				SupportedNetworks: []ExchangeCurrencyNetwork{
					{
						ID:                      "ethereum",
						Name:                    "Ethereum",
						Status:                  "online",
						ContractAddress:         "0x455e53cbb86018ac2b8092fdcd39d8444affc3f6",
						CryptoAddressLink:       "https://etherscan.io/token/0x455e53cbb86018ac2b8092fdcd39d8444affc3f6?a={{address}}",
						CryptoTransactionLink:   "https://etherscan.io/tx/0x{{txId}}",
						MinimumWithdrawalAmount: 0.00000001,
						MaximumWithdrawalAmount: 2500000,
						NetworkConfirmations:    14,
						IsEVMNetwork:            true,
					},
					{
						ID:                      "polygon",
						Name:                    "Polygon",
						Status:                  "online",
						ContractAddress:         "0x0000000000000000000000000000000000001010",
						CryptoAddressLink:       "https://polygonscan.com/address/{{address}}",
						CryptoTransactionLink:   "https://polygonscan.com/tx/{{txId}}",
						MinimumWithdrawalAmount: 0.00000001,
						MaximumWithdrawalAmount: 5000000,
						NetworkConfirmations:    300,
						ProcessingTimeSeconds:   21,
						IsEVMNetwork:            true,
					},
				},
				DisplayName: "POL",
			},
			{
				ID:               currency.USD,
				Name:             "United States Dollar",
				MinimumSize:      0.01,
				Status:           "online",
				MaximumPrecision: 0.01,
				ConvertibleTo:    []currency.Code{currency.USDC, currency.PYUSD, currency.OUSD},
				Details: ExchangeCurrencyDetails{
					Type:               ExchangeCurrencyTypeFiat,
					Symbol:             "$",
					SortOrder:          1,
					PushPaymentMethods: []string{"bank_wire", "fedwire", "swift_bank_account", "intra_bank_account"},
					GroupTypes:         []string{"fiat", "usd"},
					DisplayName:        "US Dollar",
				},
				SupportedNetworks: []ExchangeCurrencyNetwork{},
				DisplayName:       "USD",
			},
			{
				ID:               currency.XRP,
				Name:             "XRP",
				MinimumSize:      1,
				Status:           "online",
				MaximumPrecision: 0.000001,
				ConvertibleTo:    []currency.Code{},
				Details: ExchangeCurrencyDetails{
					Type:                    ExchangeCurrencyTypeCrypto,
					Symbol:                  "$",
					SortOrder:               27,
					CryptoAddressLink:       "https://bithomp.com/explorer/{{address}}",
					CryptoTransactionLink:   "https://bithomp.com/explorer/{{txId}}",
					PushPaymentMethods:      []string{"crypto"},
					GroupTypes:              []string{},
					ProcessingTimeSeconds:   600,
					MinimumWithdrawalAmount: 0.35,
					MaximumWithdrawalAmount: 10000000,
				},
				DefaultNetwork: "ripple",
				SupportedNetworks: []ExchangeCurrencyNetwork{
					{
						ID:                      "base",
						Name:                    "Base",
						Status:                  "online",
						CryptoAddressLink:       "https://basescan.org/address/{{address}}",
						CryptoTransactionLink:   "https://basescan.org/tx/{{txId}}",
						MinimumWithdrawalAmount: 1,
						MaximumWithdrawalAmount: 10000000,
						IsEVMNetwork:            true,
					},
					{
						ID:                      "ripple",
						Name:                    "Ripple",
						Status:                  "online",
						CryptoAddressLink:       "https://bithomp.com/explorer/{{address}}",
						CryptoTransactionLink:   "https://bithomp.com/explorer/{{txId}}",
						MinimumWithdrawalAmount: 0.35,
						MaximumWithdrawalAmount: 10000000,
						DestinationTagRegex:     `^\d{1,10}$`,
					},
				},
				DisplayName: "XRP",
			},
		}
		assert.Equal(t, exp, resp, "GetCurrencies should decode every field")
	} else {
		require.NotEmpty(t, resp, "GetCurrencies must return currencies")
		assert.NotEmpty(t, resp[0].ID, "GetCurrencies should return each currency's ID")
	}

	_, err = newUnavailableServerExchange(t).GetCurrencies(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetCurrencies should return the request failure")
}

func TestGetCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.GetCurrency(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetCurrency must require a currency")

	// The currency is sent upper case, as Coinbase Exchange names currencies
	resp, err := e.GetCurrency(t.Context(), currency.BTC.Lower())
	require.NoError(t, err, "GetCurrency must not error")
	if mockTests {
		exp := &ExchangeCurrencyResponse{
			ID:               currency.BTC,
			Name:             "Bitcoin",
			MinimumSize:      0.00000001,
			Status:           "online",
			MaximumPrecision: 0.00000001,
			ConvertibleTo:    []currency.Code{},
			Details: ExchangeCurrencyDetails{
				Type:                    ExchangeCurrencyTypeCrypto,
				Symbol:                  "₿",
				NetworkConfirmations:    2,
				SortOrder:               20,
				CryptoAddressLink:       "https://live.blockcypher.com/btc/address/{{address}}",
				CryptoTransactionLink:   "https://live.blockcypher.com/btc/tx/{{txId}}",
				PushPaymentMethods:      []string{"crypto"},
				GroupTypes:              []string{"btc", "crypto"},
				MinimumWithdrawalAmount: 0.0001,
				MaximumWithdrawalAmount: 2400,
			},
			DefaultNetwork: "bitcoin",
			SupportedNetworks: []ExchangeCurrencyNetwork{
				{
					ID:                      "arbitrum",
					Name:                    "Arbitrum",
					Status:                  "online",
					CryptoAddressLink:       "https://arbiscan.io/address/{{address}}",
					CryptoTransactionLink:   "https://arbiscan.io/tx/{{txId}}",
					MinimumWithdrawalAmount: 0.00002,
					MaximumWithdrawalAmount: 200,
					IsEVMNetwork:            true,
				},
				{
					ID:                      "base",
					Name:                    "Base",
					Status:                  "online",
					CryptoAddressLink:       "https://basescan.org/address/{{address}}",
					CryptoTransactionLink:   "https://basescan.org/tx/{{txId}}",
					MinimumWithdrawalAmount: 0.00002,
					MaximumWithdrawalAmount: 200,
					IsEVMNetwork:            true,
				},
				{
					ID:                      "bitcoin",
					Name:                    "Bitcoin",
					Status:                  "online",
					CryptoAddressLink:       "https://live.blockcypher.com/btc/address/{{address}}",
					CryptoTransactionLink:   "https://live.blockcypher.com/btc/tx/{{txId}}",
					MinimumWithdrawalAmount: 0.0001,
					MaximumWithdrawalAmount: 2400,
					NetworkConfirmations:    2,
				},
				{
					ID:                      "ethereum",
					Name:                    "Ethereum",
					Status:                  "online",
					CryptoAddressLink:       "https://etherscan.io/token/0xcbB7C0000aB88B473b1f5aFd9ef808440eed33Bf?a={{address}}",
					CryptoTransactionLink:   "https://etherscan.io/tx/0x{{txId}}",
					MinimumWithdrawalAmount: 0.00002,
					MaximumWithdrawalAmount: 200,
					NetworkConfirmations:    35,
					IsEVMNetwork:            true,
				},
				{
					ID:                      "solana",
					Name:                    "Solana",
					Status:                  "online",
					CryptoAddressLink:       "https://explorer.solana.com/address/{{address}}",
					CryptoTransactionLink:   "https://explorer.solana.com/tx/{{txId}}",
					MinimumWithdrawalAmount: 0.00002,
					MaximumWithdrawalAmount: 200,
					NetworkConfirmations:    31,
				},
			},
			DisplayName: "BTC",
		}
		assert.Equal(t, exp, resp, "GetCurrency should decode every field")
	} else {
		assert.True(t, resp.ID.Equal(currency.BTC), "GetCurrency should return the currency")
		assert.NotEmpty(t, resp.SupportedNetworks, "GetCurrency should return the currency's networks")
	}

	_, err = newResponseServerExchange(t, `null`).GetCurrency(t.Context(), currency.BTC)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetCurrency should error for a null response")
	_, err = newUnavailableServerExchange(t).GetCurrency(t.Context(), currency.BTC)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetCurrency should return the request failure")
}
