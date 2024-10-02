package subscription

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestResult(t *testing.T) {
	var result *Result
	result.add(nil, nil)
	require.Empty(t, result.GetSuccessful())
	require.Empty(t, result.GetUnsuccessful())

	result = &Result{}
	result.add(nil, nil)
	require.Empty(t, result.GetSuccessful())
	require.Empty(t, result.GetUnsuccessful())

	sub := &Subscription{}
	result.add(sub, nil)
	require.Len(t, result.GetSuccessful(), 1)
	require.Empty(t, result.GetUnsuccessful())

	badSub := &Subscription{}
	err := errors.New("silly")
	result.add(badSub, err)
	require.Len(t, result.GetSuccessful(), 1)
	bad := result.GetUnsuccessful()
	require.Len(t, bad, 1)
	require.ErrorIs(t, err, bad[badSub])

	result = &Result{}
	result.RunRoutine(sub, func() error { time.Sleep(time.Millisecond); return nil })
	result.ReturnWhenFinished()
	require.Len(t, result.GetSuccessful(), 1)
	require.Empty(t, result.GetUnsuccessful())
}
