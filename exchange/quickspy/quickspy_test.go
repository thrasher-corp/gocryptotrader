package quickspy

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func TestNewQuickSpy(t *testing.T) {
	_, err := NewQuickSpy(nil, nil)
	require.ErrorIs(t, err, errNoKey)

	_, err = NewQuickSpy(&CredentialsKey{}, nil)
	require.ErrorIs(t, err, errNoFocus)

	_, err = NewQuickSpy(&CredentialsKey{}, []FocusData{{}})
	require.ErrorIs(t, err, ErrUnsetFocusType)

	_, err = NewQuickSpy(&CredentialsKey{}, []FocusData{{Type: OrderBookFocusType, RESTPollTime: -1}})
	require.ErrorIs(t, err, ErrInvalidRESTPollTime)

	_, err = NewQuickSpy(&CredentialsKey{Key: key.NewExchangeAssetPair("hi", asset.Binary, currency.NewBTCUSD())}, []FocusData{{Type: OpenInterestFocusType, RESTPollTime: 10}})
	require.ErrorIs(t, err, ErrInvalidAssetForFocusType)

	_, err = NewQuickSpy(&CredentialsKey{Key: key.NewExchangeAssetPair("hi", asset.Futures, currency.NewBTCUSD())}, []FocusData{{Type: AccountHoldingsFocusType, RESTPollTime: 10}})
	require.ErrorIs(t, err, ErrCredentialsRequiredForFocusType)

	qs, err := NewQuickSpy(&CredentialsKey{Key: key.NewExchangeAssetPair("hi", asset.Futures, currency.NewBTCUSD()), Credentials: &account.Credentials{}}, []FocusData{{Type: AccountHoldingsFocusType, RESTPollTime: 10}})
	require.NoError(t, err)
	require.NotNil(t, qs)
}
