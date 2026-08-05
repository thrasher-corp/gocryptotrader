package request

import "context"

const contextVerboseFlag verbosity = "verbose"

type verbosity string

// WithVerbose adds verbosity to a request context so that specific requests
// can have distinct verbosity without impacting all requests.
func WithVerbose(ctx context.Context) context.Context {
	return context.WithValue(ctx, contextVerboseFlag, true)
}

// IsVerbose checks main verbosity first then checks context verbose values
// for specific request verbosity.
func IsVerbose(ctx context.Context, verbose bool) bool {
	if !verbose {
		verbose, _ = ctx.Value(contextVerboseFlag).(bool)
	}
	return verbose
}

type delayNotAllowedKey struct{}

// WithDelayNotAllowed adds a value to the context that indicates that no delay is allowed for rate limiting.
func WithDelayNotAllowed(ctx context.Context) context.Context {
	return context.WithValue(ctx, delayNotAllowedKey{}, struct{}{})
}

func hasDelayNotAllowed(ctx context.Context) bool {
	_, ok := ctx.Value(delayNotAllowedKey{}).(struct{})
	return ok
}

type additionalRateLimitsKey struct{}

// WithAdditionalRateLimits returns a child context that applies the supplied limiters in addition to the endpoint limiter.
// Repeated calls append limits, so callers must not add the same limiter more than once.
func WithAdditionalRateLimits(ctx context.Context, rateLimits ...AdditionalRateLimit) context.Context {
	if len(rateLimits) == 0 {
		return ctx
	}
	existing := additionalRateLimitsFromContext(ctx)
	combined := make([]AdditionalRateLimit, 0, len(existing)+len(rateLimits))
	combined = append(combined, existing...)
	combined = append(combined, rateLimits...)
	return context.WithValue(ctx, additionalRateLimitsKey{}, combined)
}

func additionalRateLimitsFromContext(ctx context.Context) []AdditionalRateLimit {
	rateLimits, _ := ctx.Value(additionalRateLimitsKey{}).([]AdditionalRateLimit)
	return rateLimits
}

type retryNotAllowedKey struct{}

// WithRetryNotAllowed adds a value to the context that indicates that no retries are allowed for requests.
func WithRetryNotAllowed(ctx context.Context) context.Context {
	return context.WithValue(ctx, retryNotAllowedKey{}, struct{}{})
}

func hasRetryNotAllowed(ctx context.Context) bool {
	_, ok := ctx.Value(retryNotAllowedKey{}).(struct{})
	return ok
}
