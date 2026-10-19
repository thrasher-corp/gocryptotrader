package coinbase

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"

	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Coinbase App list orders
const (
	AppOrderAscending  = "asc"
	AppOrderDescending = "desc"
)

const (
	// appMaximumPageLimit is the most records a Coinbase App list page holds; it holds 25 by default
	appMaximumPageLimit = 100
)

var (
	errInvalidAppOrder = errors.New("invalid list order, want asc or desc")
	errAppPageCursors  = errors.New("starting after and ending before cannot both be set")
)

// AppAPI calls the Coinbase App API, which holds an account's wallets, addresses, transfers and transaction history;
// Advanced Trade has no endpoints for these
type AppAPI struct {
	e *Exchange
}

// App returns the methods called on the Coinbase App API
func (e *Exchange) App() AppAPI {
	return AppAPI{e: e}
}

// AppPageRequest selects a page of a Coinbase App list; lists are newest first by default
type AppPageRequest struct {
	// Limit is how many records the page holds, at most 100; Coinbase holds 25 when it is unset
	Limit uint64
	// Order is asc or desc, the default
	Order string
	// StartingAfter and EndingBefore are the IDs of the records the page starts after or ends before
	StartingAfter string
	EndingBefore  string
}

// AppPagination is the cursor of a Coinbase App list page; the list is exhausted when NextURI is empty
type AppPagination struct {
	EndingBefore  string `json:"ending_before"`
	StartingAfter string `json:"starting_after"`
	// PreviousEndingBefore and NextStartingAfter are the cursors of the adjacent pages, which some lists send
	PreviousEndingBefore string       `json:"previous_ending_before"`
	NextStartingAfter    string       `json:"next_starting_after"`
	Limit                uint64       `json:"limit"`
	Order                string       `json:"order"`
	PreviousURI          string       `json:"previous_uri"`
	NextURI              string       `json:"next_uri"`
	Page                 uint64       `json:"page"`
	TotalCount           types.Number `json:"total_count"`
}

// AppWarning is a warning the Coinbase App API sends with a response, such as a deprecated parameter's
type AppWarning struct {
	ID      string `json:"id"`
	Message string `json:"message"`
	URL     string `json:"url"`
}

// appResponse is the envelope of every Coinbase App response
type appResponse[T any] struct {
	Data       T              `json:"data"`
	Pagination *AppPagination `json:"pagination"`
	Warnings   []AppWarning   `json:"warnings"`
}

// AppListResponse is a page of a Coinbase App list, with the cursor of the pages around it
type AppListResponse[T any] struct {
	Data       []T
	Pagination AppPagination
	Warnings   []AppWarning
}

// params validates a page request and returns its query; a nil request selects the first page
func (p *AppPageRequest) params() (url.Values, error) {
	params := url.Values{}
	if p == nil {
		return params, nil
	}
	if p.Limit > appMaximumPageLimit {
		return nil, fmt.Errorf("%w: %d records, at most %d", errInvalidLimit, p.Limit, appMaximumPageLimit)
	}
	if p.StartingAfter != "" && p.EndingBefore != "" {
		return nil, errAppPageCursors
	}
	switch p.Order {
	case "", AppOrderAscending, AppOrderDescending:
	default:
		return nil, fmt.Errorf("%w: %q", errInvalidAppOrder, p.Order)
	}
	if p.Limit != 0 {
		params.Set("limit", strconv.FormatUint(p.Limit, 10))
	}
	if p.Order != "" {
		params.Set("order", p.Order)
	}
	if p.StartingAfter != "" {
		params.Set("starting_after", p.StartingAfter)
	}
	if p.EndingBefore != "" {
		params.Set("ending_before", p.EndingBefore)
	}
	return params, nil
}

// sendPublicAppRequest sends an unauthenticated GET request for a Coinbase App path under /v2/ and decodes the response
// envelope
func sendPublicAppRequest[T any](ctx context.Context, a AppAPI, path string, params url.Values) (*appResponse[T], error) {
	var resp appResponse[T]
	if err := a.e.SendHTTPRequest(ctx, exchange.RestSpot, appEPL, appPath+path, params, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// sendAppRequest sends a request for a Coinbase App path under /v2/ with a JWT, and decodes the response envelope
func sendAppRequest[T any](ctx context.Context, a AppAPI, method, path string, params url.Values, body any) (*appResponse[T], error) {
	var resp appResponse[T]
	if err := a.e.SendAuthenticatedHTTPRequest(ctx, appEPL, method, appPath+path, params, body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// listResponse returns a list envelope as a list page
func listResponse[T any](resp *appResponse[[]T]) *AppListResponse[T] {
	page := &AppListResponse[T]{Data: resp.Data, Warnings: resp.Warnings}
	if resp.Pagination != nil {
		page.Pagination = *resp.Pagination
	}
	return page
}
