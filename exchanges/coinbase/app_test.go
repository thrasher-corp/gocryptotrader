package coinbase

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppPageRequestParams(t *testing.T) {
	t.Parallel()
	params, err := (*AppPageRequest)(nil).params()
	require.NoError(t, err, "params must not error for a nil request")
	assert.Empty(t, params, "params should select the first page for a nil request")
	params, err = (&AppPageRequest{Limit: 100, Order: AppOrderAscending, StartingAfter: "a"}).params()
	require.NoError(t, err, "params must not error")
	assert.Equal(t, "limit=100&order=asc&starting_after=a", params.Encode(), "params should send every field")
	params, err = (&AppPageRequest{EndingBefore: "b"}).params()
	require.NoError(t, err, "params must not error")
	assert.Equal(t, "ending_before=b", params.Encode(), "params should send the ending cursor")

	_, err = (&AppPageRequest{Limit: appMaximumPageLimit + 1}).params()
	assert.ErrorIs(t, err, errInvalidLimit, "params should refuse a limit above the maximum")
	_, err = (&AppPageRequest{StartingAfter: "a", EndingBefore: "b"}).params()
	assert.ErrorIs(t, err, errAppPageCursors, "params should refuse both cursors")
	_, err = (&AppPageRequest{Order: "up"}).params()
	assert.ErrorIs(t, err, errInvalidAppOrder, "params should refuse an unknown order")
}

func TestListResponse(t *testing.T) {
	t.Parallel()
	page := listResponse(&appResponse[[]string]{Data: []string{"a"}, Pagination: &AppPagination{NextURI: "/v2/accounts?starting_after=a"}, Warnings: []AppWarning{{ID: "deprecated"}}})
	assert.Equal(t, &AppListResponse[string]{Data: []string{"a"}, Pagination: AppPagination{NextURI: "/v2/accounts?starting_after=a"}, Warnings: []AppWarning{{ID: "deprecated"}}}, page, "listResponse should keep the data, cursor and warnings")
	assert.Equal(t, &AppListResponse[string]{Data: []string{"a"}}, listResponse(&appResponse[[]string]{Data: []string{"a"}}), "listResponse should leave the cursor empty without pagination")
}
