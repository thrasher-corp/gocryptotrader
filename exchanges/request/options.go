package request

// WithBackoff configures the backoff strategy for a Requester.
func WithBackoff(b Backoff) RequesterOption {
	return func(r *Requester) {
		r.backoff = b
	}
}

// WithLimiter configures the rate limiter for a Requester.
func WithLimiter(l Limiter) RequesterOption {
	return func(r *Requester) {
		r.limiter = l
	}
}

// WithRetryPolicy configures the retry policy for a Requester.
func WithRetryPolicy(p RetryPolicy) RequesterOption {
	return func(r *Requester) {
		r.retryPolicy = p
	}
}
