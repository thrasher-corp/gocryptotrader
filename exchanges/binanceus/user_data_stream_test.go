package binanceus

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testListenKey is the listen key the recorded user data stream responses hold
const testListenKey = "pqia91ma19a5s61cv6a81va65sdf19v8a65a1a5s61cv6a81va65sdf19v8a65a1"

func TestCreateUserDataStream(t *testing.T) {
	t.Parallel()
	t.Run("sends the API key without a signature", func(t *testing.T) {
		t.Parallel()
		ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method, "CreateUserDataStream should send a POST request")
			assert.Equal(t, "/api/v3/userDataStream", r.URL.Path, "CreateUserDataStream should request the user data stream path")
			assert.Equal(t, "test-api-key", r.Header.Get("X-MBX-APIKEY"), "CreateUserDataStream should send the API key")
			assert.Empty(t, r.URL.RawQuery, "CreateUserDataStream should send neither a timestamp nor a signature")
			writeResponse(t, w, `{"listenKey":"test-listen-key"}`)
		})
		result, err := ex.CreateUserDataStream(t.Context())
		require.NoError(t, err, "CreateUserDataStream must not error")
		assert.Equal(t, &ListenKeyResponse{ListenKey: "test-listen-key"}, result, "CreateUserDataStream should decode the listen key")
	})

	skipIfCredentialsUnset(t)
	result, err := e.CreateUserDataStream(t.Context())
	require.NoError(t, err, "CreateUserDataStream must not error")
	require.NotNil(t, result, "CreateUserDataStream must return a listen key")
	if mockTests {
		assert.Equal(t, &ListenKeyResponse{ListenKey: testListenKey}, result, "CreateUserDataStream should decode the listen key")
		return
	}
	assert.NotEmpty(t, result.ListenKey, "CreateUserDataStream should return a listen key")
}

// TestExtendUserDataStream is not parallel: an account has one listen key at a time, which TestCloseUserDataStream
// closes in live tests
func TestExtendUserDataStream(t *testing.T) {
	require.ErrorIs(t, e.ExtendUserDataStream(t.Context(), ""), errListenKeyRequired, "ExtendUserDataStream must reject an empty listen key")

	t.Run("sends the listen key with the API key alone", func(t *testing.T) {
		ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPut, r.Method, "ExtendUserDataStream should send a PUT request")
			assert.Equal(t, "/api/v3/userDataStream", r.URL.Path, "ExtendUserDataStream should request the user data stream path")
			assert.Equal(t, "test-api-key", r.Header.Get("X-MBX-APIKEY"), "ExtendUserDataStream should send the API key")
			assert.Equal(t, url.Values{"listenKey": {"test-listen-key"}}, r.URL.Query(), "ExtendUserDataStream should send the listen key without a timestamp or signature")
			writeResponse(t, w, `{}`)
		})
		require.NoError(t, ex.ExtendUserDataStream(t.Context(), "test-listen-key"), "ExtendUserDataStream must not error")
	})

	skipIfCredentialsUnset(t)
	listenKey := testListenKey
	if !mockTests {
		resp, err := e.CreateUserDataStream(t.Context())
		require.NoError(t, err, "CreateUserDataStream must not error")
		require.NotNil(t, resp, "CreateUserDataStream must return a listen key")
		listenKey = resp.ListenKey
	}
	assert.NoError(t, e.ExtendUserDataStream(t.Context(), listenKey), "ExtendUserDataStream should not error")
}

// TestCloseUserDataStream is not parallel: an account has one listen key at a time, which TestExtendUserDataStream
// extends in live tests
func TestCloseUserDataStream(t *testing.T) {
	require.ErrorIs(t, e.CloseUserDataStream(t.Context(), ""), errListenKeyRequired, "CloseUserDataStream must reject an empty listen key")

	t.Run("sends the listen key with the API key alone", func(t *testing.T) {
		ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodDelete, r.Method, "CloseUserDataStream should send a DELETE request")
			assert.Equal(t, "/api/v3/userDataStream", r.URL.Path, "CloseUserDataStream should request the user data stream path")
			assert.Equal(t, "test-api-key", r.Header.Get("X-MBX-APIKEY"), "CloseUserDataStream should send the API key")
			assert.Equal(t, url.Values{"listenKey": {"test-listen-key"}}, r.URL.Query(), "CloseUserDataStream should send the listen key without a timestamp or signature")
			writeResponse(t, w, `{}`)
		})
		require.NoError(t, ex.CloseUserDataStream(t.Context(), "test-listen-key"), "CloseUserDataStream must not error")
	})

	skipIfCredentialsUnset(t)
	listenKey := testListenKey
	if !mockTests {
		resp, err := e.CreateUserDataStream(t.Context())
		require.NoError(t, err, "CreateUserDataStream must not error")
		require.NotNil(t, resp, "CreateUserDataStream must return a listen key")
		listenKey = resp.ListenKey
	}
	assert.NoError(t, e.CloseUserDataStream(t.Context(), listenKey), "CloseUserDataStream should not error")
}

func TestSendListenKeyRequest(t *testing.T) {
	t.Parallel()
	ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method, "a REST request should use the method")
		assert.Equal(t, "/api/v3/userDataStream", r.URL.Path, "a REST request should go to the user data stream path")
		assert.Equal(t, "test-api-key", r.Header.Get("X-MBX-APIKEY"), "a REST request should carry the API key")
		assert.Empty(t, r.URL.Query().Get("signature"), "a REST request should not be signed")
		writeResponse(t, w, `{"listenKey":"key"}`)
	})
	var resp *ListenKeyResponse
	require.NoError(t, ex.sendListenKeyRequest(t.Context(), false, http.MethodPost, "userDataStream.start", nil, &resp), "sendListenKeyRequest must not error over REST")
	assert.Equal(t, "key", resp.ListenKey, "sendListenKeyRequest should decode the REST response")

	resp = nil
	ws := newWebsocketAPITestExchange(t, true)
	require.NoError(t, ws.sendListenKeyRequest(t.Context(), true, http.MethodPost, "userDataStream.start", nil, &resp), "sendListenKeyRequest must not error over the WebSocket API")
	assert.NotEmpty(t, resp.ListenKey, "sendListenKeyRequest should decode the WebSocket API result")
}
