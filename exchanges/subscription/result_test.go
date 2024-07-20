package subscription

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResult(t *testing.T) {
	var result *Result
	result.Add(nil, nil)
	require.Empty(t, result.GetSuccessful())
	require.Empty(t, result.GetUnsuccessful())

	result = &Result{}
	result.Add(nil, nil)
	require.Empty(t, result.GetSuccessful())
	require.Empty(t, result.GetUnsuccessful())

	sub := &Subscription{}
	result.Add(sub, nil)
	require.Len(t, result.GetSuccessful(), 1)
	require.Empty(t, result.GetUnsuccessful())

	badSub := &Subscription{}
	err := errors.New("silly")
	result.Add(badSub, err)
	require.Len(t, result.GetSuccessful(), 1)
	bad := result.GetUnsuccessful()
	require.Len(t, bad, 1)
	require.ErrorIs(t, err, bad[badSub])
}
