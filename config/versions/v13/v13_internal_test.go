package v13

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReplaceAccountPlan(t *testing.T) {
	t.Parallel()

	replacements := map[string]string{hobbyist: builder}
	for _, tc := range []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "replace",
			input:    `{"currencyConfig":{"cryptocurrencyProvider":{"accountPlan":"hobbyist"}}}`,
			expected: `{"currencyConfig":{"cryptocurrencyProvider":{"accountPlan":"builder"}}}`,
		},
		{
			name:     "normalised replace",
			input:    `{"currencyConfig":{"cryptocurrencyProvider":{"accountPlan":" Hobbyist "}}}`,
			expected: `{"currencyConfig":{"cryptocurrencyProvider":{"accountPlan":"builder"}}}`,
		},
		{
			name:     "missing",
			input:    `{"currencyConfig":{"cryptocurrencyProvider":{}}}`,
			expected: `{"currencyConfig":{"cryptocurrencyProvider":{}}}`,
		},
		{
			name:     "non-string",
			input:    `{"currencyConfig":{"cryptocurrencyProvider":{"accountPlan":null}}}`,
			expected: `{"currencyConfig":{"cryptocurrencyProvider":{"accountPlan":null}}}`,
		},
		{
			name:     "unmatched",
			input:    `{"currencyConfig":{"cryptocurrencyProvider":{"accountPlan":"growth"}}}`,
			expected: `{"currencyConfig":{"cryptocurrencyProvider":{"accountPlan":"growth"}}}`,
		},
		{
			name:    "malformed",
			input:   `{"currencyConfig":{"cryptocurrencyProvider":{"accountPlan":`,
			wantErr: true,
		},
		{
			name:    "invalid string escape",
			input:   `{"currencyConfig":{"cryptocurrencyProvider":{"accountPlan":"\x"}}}`,
			wantErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			input := []byte(tc.input)
			out, err := replaceAccountPlan(input, replacements)
			if tc.wantErr {
				require.Error(t, err, "replaceAccountPlan must reject malformed input")
				assert.Equal(t, input, out, "replaceAccountPlan should return the original config on error")
				return
			}
			require.NoError(t, err, "replaceAccountPlan must not error")
			assert.JSONEq(t, tc.expected, string(out), "replaceAccountPlan should return the correct config")
		})
	}
}
