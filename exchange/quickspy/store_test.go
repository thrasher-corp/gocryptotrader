package quickspy

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStore(t *testing.T) {
	t.Parallel()
	fs := NewFocusStore()
	require.NotNil(t, fs)
	require.NotNil(t, fs.s)
	require.NotNil(t, fs.m)
}

func TestUpsert(t *testing.T) {
	t.Parallel()
	fs := NewFocusStore()
	require.NotNil(t, fs, "NewFocusStore must not return nil")
	// test nil data does nothing
	fs.Upsert(TickerFocusType, nil)
	require.Empty(t, fs.s)
	// success
	fd := &FocusData{}
	fs.Upsert(TickerFocusType, fd)
	require.Len(t, fs.s, 1)
	require.Equal(t, TickerFocusType, fs.s[TickerFocusType].Type)
	require.NotNil(t, fs.s[TickerFocusType].m)
	// test update existing key
	fd2 := &FocusData{}
	fs.Upsert(TickerFocusType, fd2)
	require.Len(t, fs.s, 1)
	require.Equal(t, TickerFocusType, fs.s[TickerFocusType].Type)
}

func TestRemove(t *testing.T) {
	t.Parallel()
	fs := NewFocusStore()
	require.NotNil(t, fs, "NewFocusStore must not return nil")
	fd := &FocusData{}
	fs.Upsert(TickerFocusType, fd)
	require.Len(t, fs.s, 1)
	fs.Remove(TickerFocusType)
	require.Empty(t, fs.s)
	// test removing non-existent key does nothing
	fs.Remove(TickerFocusType)
	require.Empty(t, fs.s)
}

func TestGetByFocusType(t *testing.T) {
	t.Parallel()
	fs := NewFocusStore()
	require.NotNil(t, fs, "NewFocusStore must not return nil")
	// test non-existent key returns nil
	result := fs.GetByFocusType(TickerFocusType)
	require.Nil(t, result)
	// success
	fd := &FocusData{}
	fs.Upsert(TickerFocusType, fd)
	result = fs.GetByFocusType(TickerFocusType)
	require.NotNil(t, result)
	require.NotNil(t, result.m)
	require.Equal(t, TickerFocusType, result.Type)
}

func TestList(t *testing.T) {
	t.Parallel()
	fs := NewFocusStore()
	require.NotNil(t, fs, "NewFocusStore must not return nil")
	// test empty store returns empty slice
	list := fs.List()
	require.NotNil(t, list)
	require.Empty(t, list)
	// success
	fd1 := &FocusData{}
	fd2 := &FocusData{}
	fs.Upsert(TickerFocusType, fd1)
	fs.Upsert(TradesFocusType, fd2)
	list = fs.List()
	require.Len(t, list, 2)
}

func TestDisableWebsocketFocuses(t *testing.T) {
	t.Parallel()
	fs := NewFocusStore()
	require.NotNil(t, fs, "NewFocusStore must not return nil")
	// test empty store does nothing
	fs.DisableWebsocketFocuses()
	// success
	fd1 := &FocusData{UseWebsocket: true}
	fd2 := &FocusData{UseWebsocket: true}
	fs.Upsert(TickerFocusType, fd1)
	fs.Upsert(TradesFocusType, fd2)
	fs.DisableWebsocketFocuses()
	list := fs.List()
	for _, fd := range list {
		t.Run(fd.Type.String(), func(t *testing.T) {
			t.Parallel()
			assert.False(t, fd.UseWebsocket)
		})
	}
}
