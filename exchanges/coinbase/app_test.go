package coinbase

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testAppLiveUnknownID is the nil UUID, which RFC 9562 sets apart from every generated UUID, so no payment method or
// transfer has it and a live request naming it is refused
const testAppLiveUnknownID = "00000000-0000-0000-0000-000000000000"

// appRequestRefusalIDs are the ids of the Coinbase App's errors that refuse or fail a request itself rather than answer
// what it asks for, by the App API's error reference: a missing parameter or a malformed request, refused credentials
// or scopes, an account verification not completed, a rate limit or an outage. Any other error is an answer, such as
// not_found for a resource no account holds, validation_error for a value the App refuses, or two_factor_required for
// a transfer over the account's limit
var appRequestRefusalIDs = []string{
	"param_required", "invalid_request", "personal_details_required", "identity_verification_required",
	"jumio_verification_required", "jumio_face_match_verification_required", "unverified_email", "authentication_error",
	"invalid_token", "revoked_token", "expired_token", "invalid_scope", "unauthorized", "rate_limit_exceeded",
	"resource_exhausted", "internal_server_error",
}

// requireAppRefusedOnMerits fails a live case built to be refused unless the Coinbase App refused its request with
// errors that answer it on its merits
func requireAppRefusedOnMerits(t *testing.T, err error, method string) {
	t.Helper()
	require.Errorf(t, err, "%s must be refused, as its request is built to be refused", method)
	appErrs, ok := errors.AsType[AppErrors](err)
	require.Truef(t, ok, "%s must return the Coinbase App's errors, not %v", method, err)
	require.NotEmptyf(t, appErrs, "%s must return the Coinbase App's errors", method)
	for i := range appErrs {
		require.NotContainsf(t, appRequestRefusalIDs, appErrs[i].ID, "%s must be refused on its merits rather than as a request", method)
	}
}

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
	page := listResponse(&AppResponse[[]string]{Data: []string{"a"}, Pagination: &AppPagination{NextURI: "/v2/accounts?starting_after=a"}, Warnings: []AppWarning{{ID: "deprecated"}}})
	assert.Equal(t, &AppListResponse[string]{Data: []string{"a"}, Pagination: AppPagination{NextURI: "/v2/accounts?starting_after=a"}, Warnings: []AppWarning{{ID: "deprecated"}}}, page, "listResponse should keep the data, cursor and warnings")
	assert.Equal(t, &AppListResponse[string]{Data: []string{"a"}}, listResponse(&AppResponse[[]string]{Data: []string{"a"}}), "listResponse should leave the cursor empty without pagination")
}
