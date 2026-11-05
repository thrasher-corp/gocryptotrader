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
		d := DurationSeconds(time.Hour)
		require.NoErrorf(t, d.UnmarshalJSON([]byte(tc.input)), "UnmarshalJSON must not error for %s", tc.input)
		assert.Equalf(t, tc.expected, d.Duration(), "UnmarshalJSON should read %s as %s", tc.input, tc.expected)
	}
	for _, input := range []string{``, `true`, `false`, `nul`, `"30`, `"abc"`, `"1h"`, `"1s"`, `"1.2.3"`, `"-"`, `"."`, `"1e"`, `"1e+"`, `"NaN"`, `"Inf"`, `"0x10"`, `"1/2"`, `9223372037`, `9223372036.854775808`, `1e400`, `-1e19`, `1e1000001`, `1e99999999999999999999`, `"1e5x"`, `"1.5e2.0"`, `[1]`, `{}`} {
		d := DurationSeconds(time.Hour)
		assert.ErrorIsf(t, d.UnmarshalJSON([]byte(input)), errInvalidDurationValue, "UnmarshalJSON should error for %s", input)
		assert.Equalf(t, time.Hour, d.Duration(), "UnmarshalJSON should leave the value unchanged for %s", input)
	}
}

func TestDurationSecondsUnmarshalJSONLongFraction(t *testing.T) {
	t.Parallel()
	// big.Rat refuses a fraction this long, as it needs ten to the power of its length
	input := "1." + strings.Repeat("0", 1_000_001) + "1"
	var d DurationSeconds
	require.NoError(t, d.UnmarshalJSON([]byte(input)), "UnmarshalJSON must read a fraction a million digits long")
	assert.Equal(t, time.Second, d.Duration(), "UnmarshalJSON should drop the digits below a nanosecond")
}

func TestParseDurationMatchesRationalArithmetic(t *testing.T) {
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
		for _, unit := range []time.Duration{time.Second, time.Millisecond} {
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
	}
}

func TestDurationMillisecondsUnmarshalJSON(t *testing.T) {
	t.Parallel()
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
		d := DurationMilliseconds(time.Hour)
		require.NoErrorf(t, d.UnmarshalJSON([]byte(tc.input)), "UnmarshalJSON must not error for %s", tc.input)
		assert.Equalf(t, tc.expected, d.Duration(), "UnmarshalJSON should read %s as %s", tc.input, tc.expected)
	}
	for _, input := range []string{`true`, `"500ms"`, `"1m"`, `"abc"`, `9223372036855`} {
		d := DurationMilliseconds(time.Hour)
		assert.ErrorIsf(t, d.UnmarshalJSON([]byte(input)), errInvalidDurationValue, "UnmarshalJSON should error for %s", input)
	}
}

func TestDurationMarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		duration              time.Duration
		seconds, milliseconds string
	}{
		{0, `0`, `0`},
		{30 * time.Second, `30`, `30000`},
		{1500 * time.Millisecond, `1.5`, `1500`},
		{time.Nanosecond, `0.000000001`, `0.000001`},
		{-1500 * time.Microsecond, `-0.0015`, `-1.5`},
		{-time.Nanosecond, `-0.000000001`, `-0.000001`},
		{math.MinInt64, `-9223372036.854775808`, `-9223372036854.775808`},
		{math.MaxInt64, `9223372036.854775807`, `9223372036854.775807`},
	} {
		got, err := DurationSeconds(tc.duration).MarshalJSON()
		require.NoError(t, err, "DurationSeconds.MarshalJSON must not error")
		assert.Equalf(t, tc.seconds, string(got), "DurationSeconds.MarshalJSON should write %s as seconds", tc.duration)
		var seconds DurationSeconds
		require.NoErrorf(t, json.Unmarshal(got, &seconds), "DurationSeconds must read back %s", got)
		assert.Equalf(t, tc.duration, seconds.Duration(), "DurationSeconds should round trip %s", tc.duration)

		got, err = DurationMilliseconds(tc.duration).MarshalJSON()
		require.NoError(t, err, "DurationMilliseconds.MarshalJSON must not error")
		assert.Equalf(t, tc.milliseconds, string(got), "DurationMilliseconds.MarshalJSON should write %s as milliseconds", tc.duration)
		var milliseconds DurationMilliseconds
		require.NoErrorf(t, json.Unmarshal(got, &milliseconds), "DurationMilliseconds must read back %s", got)
		assert.Equalf(t, tc.duration, milliseconds.Duration(), "DurationMilliseconds should round trip %s", tc.duration)
	}
}

func TestDurationString(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "1m30s", DurationSeconds(90*time.Second).String(), "DurationSeconds.String should use time.Duration's format")
	assert.Equal(t, "1.5s", DurationMilliseconds(1500*time.Millisecond).String(), "DurationMilliseconds.String should use time.Duration's format")
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
