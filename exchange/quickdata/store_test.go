package quickdata

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
}

func TestUpsert(t *testing.T) {
	t.Parallel()
	fs := NewFocusStore()
	require.NotNil(t, fs, "NewFocusStore must not return nil")
	fs.Upsert(TickerFocusType, nil)
	require.Empty(t, fs.s)

	fd := &FocusData{}
	fs.Upsert(TickerFocusType, fd)
	require.Len(t, fs.s, 1)
	require.Equal(t, TickerFocusType, fs.s[TickerFocusType].focusType)

	fd2 := &FocusData{}
	fs.Upsert(TickerFocusType, fd2)
	require.Len(t, fs.s, 1)
	require.Equal(t, TickerFocusType, fs.s[TickerFocusType].focusType)
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

	fs.Remove(TickerFocusType)
	require.Empty(t, fs.s)
}

func TestGetByFocusType(t *testing.T) {
	t.Parallel()
	fs := NewFocusStore()
	require.NotNil(t, fs, "NewFocusStore must not return nil")
	result := fs.GetByFocusType(TickerFocusType)
	require.Nil(t, result)

	fd := &FocusData{}
	fs.Upsert(TickerFocusType, fd)
	result = fs.GetByFocusType(TickerFocusType)
	require.NotNil(t, result)
	require.Equal(t, TickerFocusType, result.focusType)
}

func TestList(t *testing.T) {
	t.Parallel()
	fs := NewFocusStore()
	require.NotNil(t, fs, "NewFocusStore must not return nil")
	list := fs.List()
	require.Empty(t, list)

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
	fs.DisableWebsocketFocuses()

	fd1 := &FocusData{useWebsocket: true}
	fd2 := &FocusData{useWebsocket: true}
	fs.Upsert(TickerFocusType, fd1)
	fs.Upsert(TradesFocusType, fd2)
	fs.DisableWebsocketFocuses()
	list := fs.List()
	for _, fd := range list {
		t.Run(fd.focusType.String(), func(t *testing.T) {
			t.Parallel()
			assert.False(t, fd.useWebsocket)
		})
	}
}
