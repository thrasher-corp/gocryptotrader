package request

// WithBackoff configures the backoff strategy for a Requester.
func WithBackoff(b Backoff) RequesterOption {
	return func(r *Requester) {
		r.backoff = b
	}
}

// WithLimiter configures the rate limiter for a Requester.
func WithLimiter(def RateLimitDefinitions) RequesterOption {
	return func(r *Requester) {
		r.limiter = def
	}
}

// WithRetryPolicy configures the retry policy for a Requester.
func WithRetryPolicy(p RetryPolicy) RequesterOption {
	return func(r *Requester) {
		r.retryPolicy = p
	}
}

// WithReporter configures the reporter for a Requester.
func WithReporter(rep Reporter) RequesterOption {
	return func(r *Requester) {
		r.reporter = rep
	}
}
