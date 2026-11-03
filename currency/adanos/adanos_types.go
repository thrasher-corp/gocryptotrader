package adanos

import (
	"net/http"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Client provides on-demand Reddit crypto sentiment without engine integration.
// Construct a client with New and reuse it for subsequent requests.
// Call Shutdown once when retiring it, after all requests have finished.
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	requester  *request.Requester
}

// TokenRequest selects a symbol and optional inclusive UTC date bounds.
// Zero bounds are omitted. Times within a UTC date do not affect the query.
type TokenRequest struct {
	Symbol string
	From   time.Time
	To     time.Time
}

// TokenResponse is the complete v1 token sentiment report. Nil fields mean that
// the API did not supply a value, rather than a neutral score or a zero count.
type TokenResponse struct {
	Symbol            string           `json:"symbol"`
	Name              *string          `json:"name"`
	Found             bool             `json:"found"` // Legacy v1 data availability, not symbol support.
	BuzzScore         *float64         `json:"buzz_score"`
	Mentions          *uint64          `json:"mentions"`
	SentimentScore    *float64         `json:"sentiment_score"`
	PositiveCount     *uint64          `json:"positive_count"`
	NegativeCount     *uint64          `json:"negative_count"`
	NeutralCount      *uint64          `json:"neutral_count"`
	TotalUpvotes      *int64           `json:"total_upvotes"`
	UniquePosts       *uint64          `json:"unique_posts"`
	SubredditCount    *uint64          `json:"subreddit_count"`
	Trend             *string          `json:"trend"`
	BullishPercentage *uint64          `json:"bullish_pct"`
	BearishPercentage *uint64          `json:"bearish_pct"`
	PeriodDays        *uint64          `json:"period_days"`
	TopSubreddits     []SubredditCount `json:"top_subreddits"`
	DailyTrend        []DailyTrendItem `json:"daily_trend"`
	TopMentions       []TopMention     `json:"top_mentions"`
}

// SubredditCount contains contributor metrics for the selected period.
type SubredditCount struct {
	Subreddit      string   `json:"subreddit"`
	Mentions       uint64   `json:"mentions"`
	Count          uint64   `json:"count"` // Deprecated API alias for mentions, retained for v1 compatibility.
	SentimentScore *float64 `json:"sentiment_score"`
	BuzzScore      *float64 `json:"buzz_score"`
}

// DailyTrendItem retains the API's date-only string without inventing a timestamp.
type DailyTrendItem struct {
	Date              string   `json:"date"`
	Mentions          uint64   `json:"mentions"`
	SentimentScore    *float64 `json:"sentiment_score"`
	BuzzScore         *float64 `json:"buzz_score"`
	BullishPercentage *uint64  `json:"bullish_pct"`
	BearishPercentage *uint64  `json:"bearish_pct"`
}

// TopMention contains discussion evidence. Reddit scores can be negative.
type TopMention struct {
	TextSnippet    string    `json:"text_snippet"`
	SentimentScore *float64  `json:"sentiment_score"`
	SentimentLabel *string   `json:"sentiment_label"`
	Upvotes        int64     `json:"upvotes"`
	Subreddit      *string   `json:"subreddit"`
	CreatedUTC     time.Time `json:"created_utc"`
}

type tokenError struct {
	err    error
	apiKey string
}
