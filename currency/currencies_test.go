package currency

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

func TestCurrenciesUnmarshalJSON(t *testing.T) {
	var unmarshalHere Currencies
	expected := "btc,usd,ltc,bro,things"
	encoded, err := json.Marshal(expected)
	require.NoError(t, err)

	err = json.Unmarshal(encoded, &unmarshalHere)
	require.NoError(t, err)

	err = json.Unmarshal(encoded, &unmarshalHere)
	require.NoError(t, err)
	require.Equal(t, expected, unmarshalHere.Join())

	j := []byte(`["btc","usd","ltc","bro","things"]`)
	err = json.Unmarshal(j, &unmarshalHere)
	require.NoError(t, err)
	require.Len(t, unmarshalHere, 5)
}

func TestCurrenciesMarshalJSON(t *testing.T) {
	quickStruct := struct {
		C Currencies `json:"amazingCurrencies"`
	}{
		C: NewCurrenciesFromStringArray([]string{"btc", "usd", "ltc", "bro", "things"}),
	}

	encoded, err := json.Marshal(quickStruct)
	require.NoError(t, err)

	expected := `{"amazingCurrencies":"btc,usd,ltc,bro,things"}`
	require.Equal(t, expected, string(encoded))
}

func TestMatch(t *testing.T) {
	matchString := []string{"btc", "usd", "ltc", "bro", "things"}
	c := NewCurrenciesFromStringArray(matchString)
	require.True(t, c.Match(NewCurrenciesFromStringArray(matchString)))
	require.False(t, c.Match(NewCurrenciesFromStringArray([]string{"btc", "usd", "ltc", "bro"})))
	require.False(t, c.Match(NewCurrenciesFromStringArray([]string{"btc", "usd", "ltc", "bro", "garbo"})))
}

func TestCurrenciesAdd(t *testing.T) {
	c := Currencies{}
	c = c.Add(BTC)
	assert.Len(t, c, 1, "Should have one currency")
	c = c.Add(ETH)
	assert.Len(t, c, 2, "Should have two currencies")
	c = c.Add(BTC)
	assert.Len(t, c, 2, "Adding a duplicate should not change anything")
}
