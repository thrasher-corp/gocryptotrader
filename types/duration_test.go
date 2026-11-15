package types

import (
	"math"
	"math/big"
	"math/rand/v2"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

func TestDurationSecondsUnmarshalJSON(t *testing.T) {
	t.Parallel()

	t.Run("fraction a million digits long", func(t *testing.T) {
		t.Parallel()
		// big.Rat refuses a fraction this long, as it needs ten to the power of its length
		input := "1." + strings.Repeat("0", 1_000_001) + "1"
		var d DurationSeconds
		require.NoError(t, d.UnmarshalJSON([]byte(input)), "UnmarshalJSON must read a fraction a million digits long")
		assert.Equal(t, time.Second, d.Duration(), "UnmarshalJSON should drop the digits below a nanosecond")
	})

	// Cases run in order, before the invalid inputs below call UnmarshalJSON again
	for _, tc := range []struct {
		input    string
		expected time.Duration
	}{
		{`30`, 30 * time.Second},
		{`"30"`, 30 * time.Second},
		{`"1.5"`, 1500 * time.Millisecond},
		{`0.000000001`, time.Nanosecond},
		{`-3`, -3 * time.Second},
		{`"+3"`, 3 * time.Second},
		{`".5"`, 500 * time.Millisecond},
		{`1e3`, 1000 * time.Second},
		{`"2.5E-1"`, 250 * time.Millisecond},
		{`0`, 0},
		{`""`, 0},
		{`null`, 0},
		{`9223372036`, 9223372036 * time.Second},
		{`9223372036.854775807`, math.MaxInt64},
		{`9223372036.854775807e0`, math.MaxInt64},
		{`-9223372036.854775808`, math.MinInt64},
		{`0.0000000005`, 0},
		{`5e-10`, 0},
		{`0.0000000019`, time.Nanosecond},
		{`-0.0000000019`, -time.Nanosecond},
		{`"\u0035"`, 5 * time.Second},
		{`007`, 7 * time.Second},
		{`1E+2`, 100 * time.Second},
		{`100e-11`, time.Nanosecond},
		{`"0.000000001e9"`, time.Second},
		{`1e-1000001`, 0},
		{`-1e-10000001`, 0},
		{`"1e-99999999999999999999"`, 0},
		{`0e99999999999`, 0},
	} {
		t.Run(tc.input, func(t *testing.T) {
			d := DurationSeconds(time.Hour)
			require.NoErrorf(t, d.UnmarshalJSON([]byte(tc.input)), "UnmarshalJSON must not error for %s", tc.input)
			assert.Equalf(t, tc.expected, d.Duration(), "UnmarshalJSON should read %s as %s", tc.input, tc.expected)
		})
	}
	for _, tc := range []struct {
		name, input string
	}{
		{"empty input", ``},
		{"true", `true`},
		{"false", `false`},
		{"truncated null", `nul`},
		{"unterminated string", `"30`},
		{"letters", `"abc"`},
		{"hour unit", `"1h"`},
		{"second unit", `"1s"`},
		{"two decimal points", `"1.2.3"`},
		{"sign only", `"-"`},
		{"decimal point only", `"."`},
		{"exponent without digits", `"1e"`},
		{"signed exponent without digits", `"1e+"`},
		{"NaN", `"NaN"`},
		{"infinity", `"Inf"`},
		{"hexadecimal", `"0x10"`},
		{"ratio", `"1/2"`},
		{"seconds above the maximum", `9223372037`},
		{"a nanosecond above the maximum", `9223372036.854775808`},
		{"large exponent", `1e400`},
		{"below the minimum", `-1e19`},
		{"exponent above a million", `1e1000001`},
		{"exponent beyond int64", `1e99999999999999999999`},
		{"characters after the exponent", `"1e5x"`},
		{"fractional exponent", `"1.5e2.0"`},
		{"array", `[1]`},
		{"object", `{}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			d := DurationSeconds(time.Hour)
			assert.ErrorIsf(t, d.UnmarshalJSON([]byte(tc.input)), errInvalidDurationValue, "UnmarshalJSON should error for %s", tc.input)
			assert.Equalf(t, time.Hour, d.Duration(), "UnmarshalJSON should leave the value unchanged for %s", tc.input)
		})
	}
}

func TestDurationMillisecondsUnmarshalJSON(t *testing.T) {
	t.Parallel()
	// Cases run in order, before the invalid inputs below call UnmarshalJSON again
	for _, tc := range []struct {
		input    string
		expected time.Duration
	}{
		{`500`, 500 * time.Millisecond},
		{`"1500"`, 1500 * time.Millisecond},
		{`"0.000001"`, time.Nanosecond},
		{`-250`, -250 * time.Millisecond},
		{`1e3`, time.Second},
		{`""`, 0},
		{`null`, 0},
	} {
		t.Run(tc.input, func(t *testing.T) {
			d := DurationMilliseconds(time.Hour)
			require.NoErrorf(t, d.UnmarshalJSON([]byte(tc.input)), "UnmarshalJSON must not error for %s", tc.input)
			assert.Equalf(t, tc.expected, d.Duration(), "UnmarshalJSON should read %s as %s", tc.input, tc.expected)
		})
	}
	for _, tc := range []struct {
		name, input string
	}{
		{"true", `true`},
		{"millisecond unit", `"500ms"`},
		{"minute unit", `"1m"`},
		{"letters", `"abc"`},
		{"milliseconds above the maximum", `9223372036855`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			d := DurationMilliseconds(time.Hour)
			assert.ErrorIsf(t, d.UnmarshalJSON([]byte(tc.input)), errInvalidDurationValue, "UnmarshalJSON should error for %s", tc.input)
		})
	}
}

func TestDurationSecondsMarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		duration time.Duration
		seconds  string
	}{
		{0, `0`},
		{30 * time.Second, `30`},
		{1500 * time.Millisecond, `1.5`},
		{time.Nanosecond, `0.000000001`},
		{-1500 * time.Microsecond, `-0.0015`},
		{-time.Nanosecond, `-0.000000001`},
		{math.MinInt64, `-9223372036.854775808`},
		{math.MaxInt64, `9223372036.854775807`},
	} {
		t.Run(tc.duration.String(), func(t *testing.T) {
			t.Parallel()
			got, err := DurationSeconds(tc.duration).MarshalJSON()
			require.NoError(t, err, "DurationSeconds.MarshalJSON must not error")
			assert.Equalf(t, tc.seconds, string(got), "DurationSeconds.MarshalJSON should write %s as seconds", tc.duration)
			var seconds DurationSeconds
			require.NoErrorf(t, json.Unmarshal(got, &seconds), "DurationSeconds must read back %s", got)
			assert.Equalf(t, tc.duration, seconds.Duration(), "DurationSeconds should round trip %s", tc.duration)
		})
	}
}

func TestDurationMillisecondsMarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		duration     time.Duration
		milliseconds string
	}{
		{0, `0`},
		{30 * time.Second, `30000`},
		{1500 * time.Millisecond, `1500`},
		{time.Nanosecond, `0.000001`},
		{-1500 * time.Microsecond, `-1.5`},
		{-time.Nanosecond, `-0.000001`},
		{math.MinInt64, `-9223372036854.775808`},
		{math.MaxInt64, `9223372036854.775807`},
	} {
		t.Run(tc.duration.String(), func(t *testing.T) {
			t.Parallel()
			got, err := DurationMilliseconds(tc.duration).MarshalJSON()
			require.NoError(t, err, "DurationMilliseconds.MarshalJSON must not error")
			assert.Equalf(t, tc.milliseconds, string(got), "DurationMilliseconds.MarshalJSON should write %s as milliseconds", tc.duration)
			var milliseconds DurationMilliseconds
			require.NoErrorf(t, json.Unmarshal(got, &milliseconds), "DurationMilliseconds must read back %s", got)
			assert.Equalf(t, tc.duration, milliseconds.Duration(), "DurationMilliseconds should round trip %s", tc.duration)
		})
	}
}

func TestDurationSecondsDuration(t *testing.T) {
	t.Parallel()
	assert.Equal(t, -1500*time.Millisecond, DurationSeconds(-1500*time.Millisecond).Duration(), "DurationSeconds.Duration should return the underlying time.Duration")
}

func TestDurationMillisecondsDuration(t *testing.T) {
	t.Parallel()
	assert.Equal(t, -1500*time.Millisecond, DurationMilliseconds(-1500*time.Millisecond).Duration(), "DurationMilliseconds.Duration should return the underlying time.Duration")
}

func TestDurationSecondsString(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "1m30s", DurationSeconds(90*time.Second).String(), "DurationSeconds.String should use time.Duration's format")
}

func TestDurationMillisecondsString(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "1.5s", DurationMilliseconds(1500*time.Millisecond).String(), "DurationMilliseconds.String should use time.Duration's format")
}

func TestUnmarshalDuration(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		input    string
		unit     time.Duration
		expected time.Duration
		err      error
	}{
		{name: "empty input", input: ``, unit: time.Second, err: errInvalidDurationValue},
		{name: "null", input: `null`, unit: time.Second},
		{name: "truncated null", input: `nul`, unit: time.Second, err: errInvalidDurationValue},
		{name: "empty string", input: `""`, unit: time.Second},
		{name: "unterminated string", input: `"30`, unit: time.Second, err: errInvalidDurationValue},
		{name: "quoted seconds", input: `"1.5"`, unit: time.Second, expected: 1500 * time.Millisecond},
		{name: "bare milliseconds", input: `1500`, unit: time.Millisecond, expected: 1500 * time.Millisecond},
		{name: "invalid number", input: `"1h"`, unit: time.Second, err: errInvalidDurationValue},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := unmarshalDuration([]byte(tc.input), tc.unit)
			require.ErrorIs(t, err, tc.err, "unmarshalDuration must return the expected error")
			assert.Equal(t, tc.expected, got, "unmarshalDuration should return the expected duration")
		})
	}
}

func TestParseDuration(t *testing.T) {
	t.Parallel()

	t.Run("matches rational arithmetic", func(t *testing.T) {
		t.Parallel()
		for _, unit := range []time.Duration{time.Second, time.Millisecond} {
			t.Run(unit.String(), func(t *testing.T) {
				t.Parallel()
				r := rand.New(rand.NewPCG(1, 2)) //nolint:gosec // Test inputs need a fixed seed, not a strong random number
				randomDigits := func(n int) string {
					b := make([]byte, n)
					for i := range b {
						b[i] = "0123456789"[r.IntN(10)]
					}
					return string(b)
				}
				for range 10000 {
					var b strings.Builder
					b.WriteString([]string{"", "-", "+"}[r.IntN(3)])
					integer, fraction := randomDigits(r.IntN(13)), randomDigits(r.IntN(13))
					if integer == "" && fraction == "" {
						integer = randomDigits(1)
					}
					b.WriteString(integer)
					if fraction != "" || r.IntN(4) == 0 {
						b.WriteString("." + fraction)
					}
					if r.IntN(2) == 0 {
						b.WriteString([]string{"e", "E"}[r.IntN(2)] + []string{"", "-", "+"}[r.IntN(3)] + strconv.Itoa(r.IntN(41)))
					}
					s := b.String()
					exact, ok := new(big.Rat).SetString(s)
					require.Truef(t, ok, "big.Rat must read %q", s)
					ns := new(big.Int).Quo(exact.Mul(exact, new(big.Rat).SetInt64(int64(unit))).Num(), exact.Denom())
					got, err := parseDuration(s, unit)
					if !ns.IsInt64() {
						require.ErrorIsf(t, err, errInvalidDurationValue, "parseDuration must refuse %q in units of %s as out of range", s, unit)
						continue
					}
					require.NoErrorf(t, err, "parseDuration must read %q in units of %s", s, unit)
					require.Equalf(t, time.Duration(ns.Int64()), got, "parseDuration must match exact arithmetic for %q in units of %s", s, unit)
				}
			})
		}
	})
}

func TestFormatDuration(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		duration time.Duration
		unit     time.Duration
		expected string
	}{
		{name: "zero", duration: 0, unit: time.Second, expected: `0`},
		{name: "whole units", duration: 30 * time.Second, unit: time.Second, expected: `30`},
		{name: "negative whole units", duration: -2 * time.Millisecond, unit: time.Millisecond, expected: `-2`},
		{name: "fraction", duration: 1500 * time.Millisecond, unit: time.Second, expected: `1.5`},
		{name: "fraction padded with zeros", duration: time.Nanosecond, unit: time.Second, expected: `0.000000001`},
		{name: "negative fraction", duration: -1500 * time.Microsecond, unit: time.Millisecond, expected: `-1.5`},
		{name: "negative fraction below one unit", duration: -time.Nanosecond, unit: time.Second, expected: `-0.000000001`},
		{name: "minimum duration", duration: math.MinInt64, unit: time.Second, expected: `-9223372036.854775808`},
		{name: "maximum duration", duration: math.MaxInt64, unit: time.Millisecond, expected: `9223372036854.775807`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, string(formatDuration(tc.duration, tc.unit)), "formatDuration should write the duration as an exact number of units")
		})
	}
}

func TestDurationInStruct(t *testing.T) {
	t.Parallel()
	type payload struct {
		Timeout DurationSeconds      `json:"timeout"`
		ValidMS DurationMilliseconds `json:"valid_ms"`
		Omitted DurationSeconds      `json:"omitted,omitempty"`
	}
	var p payload
	require.NoError(t, json.Unmarshal([]byte(`{"timeout":"30","valid_ms":1500}`), &p), "Unmarshal must not error")
	assert.Equal(t, 30*time.Second, p.Timeout.Duration(), "Timeout should read as seconds")
	assert.Equal(t, 1500*time.Millisecond, p.ValidMS.Duration(), "ValidMS should read as milliseconds")
	got, err := json.Marshal(p)
	require.NoError(t, err, "Marshal must not error")
	assert.JSONEq(t, `{"timeout":30,"valid_ms":1500}`, string(got), "Marshal should write bare numbers and omit an empty omitempty duration")
}
