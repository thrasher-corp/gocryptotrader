//go:build udecimal_on

package orderbook

// Match the fractional precision enforced by types/decimal's udecimal backend.
const executionFractionalDigits = 19
