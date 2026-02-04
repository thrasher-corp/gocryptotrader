package livetest

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnvIsTrue(t *testing.T) {
	for _, tc := range []struct {
		name     string
		value    string
		expected bool
	}{
		{name: "empty", value: "", expected: false},
		{name: "whitespace", value: "  ", expected: false},
		{name: "true lowercase", value: "true", expected: true},
		{name: "true uppercase", value: "TRUE", expected: true},
		{name: "true mixed", value: "tRuE", expected: true},
		{name: "one", value: "1", expected: true},
		{name: "zero", value: "0", expected: false},
		{name: "other", value: "yes", expected: false},
		{name: "true with whitespace", value: " true ", expected: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("GCT_TEST_ENV_TRUE", tc.value)
			got := envIsTrue("GCT_TEST_ENV_TRUE")
			assert.Equalf(t, tc.expected, got, "envIsTrue should be %v for value %q", tc.expected, tc.value)
		})
	}
}

func TestShouldSkip(t *testing.T) {
	for _, tc := range []struct {
		name     string
		gctSkip  string
		expected bool
	}{
		{name: "none", expected: false},
		{name: "gct skip true", gctSkip: "true", expected: true},
		{name: "gct skip one", gctSkip: "1", expected: true},
		{name: "gct skip false", gctSkip: "false", expected: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("GCT_SKIP_LIVE_TESTS", tc.gctSkip)
			got := ShouldSkip()
			assert.Equalf(t, tc.expected, got, "ShouldSkip should be %v (GCT_SKIP_LIVE_TESTS=%q)", tc.expected, tc.gctSkip)
		})
	}
}
