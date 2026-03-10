package bitmex

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParamsToURLValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		params      Parameter
		expect      url.Values
		expectEmpty bool
	}{
		{
			name:        "nil parameter",
			params:      nil,
			expectEmpty: true,
		},
		{
			name:        "nil typed pointer",
			params:      (*ChatGetParams)(nil),
			expectEmpty: true,
		},
		{
			name:   "pointer param fields encoded",
			params: &ChatGetParams{ChannelID: 1.25, Count: 2, Reverse: true, Start: 3},
			expect: url.Values{
				"channelID": []string{"1.2500"},
				"count":     []string{"2"},
				"reverse":   []string{"true"},
				"start":     []string{"3"},
			},
		},
		{
			name:   "value param converted to pointer",
			params: UserTokenParams{Token: "abc123"},
			expect: url.Values{
				"token": []string{"abc123"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := paramsToURLValues(tc.params)
			require.NoError(t, err)
			if tc.expectEmpty {
				assert.Empty(t, got)
				return
			}
			assert.Equal(t, tc.expect, got)
		})
	}
}

func TestParamsToRequestPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		params   Parameter
		expected string
	}{
		{
			name:     "empty values returns path unchanged",
			path:     "/api/v1/user",
			params:   &APIKeyParams{},
			expected: "/api/v1/user",
		},
		{
			name:     "appends encoded query values",
			path:     "/api/v1/chat",
			params:   &ChatGetParams{Count: 10, Reverse: true},
			expected: "/api/v1/chat?count=10&reverse=true",
		},
		{
			name:     "encodes escaped values",
			path:     "/api/v1/chat",
			params:   &APIKeyParams{APIKeyID: "client credentials"},
			expected: "/api/v1/chat?apiKeyID=client+credentials",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := paramsToRequestPath(tc.params, tc.path)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, got)
		})
	}
}
