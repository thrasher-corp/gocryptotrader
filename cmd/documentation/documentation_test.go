package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetContributorList(t *testing.T) {
	t.Parallel()

	c, err := GetContributorList(t.Context(), DefaultRepo, true)
	require.NoError(t, err, "GetContributorList must not error")
	require.NotEmpty(t, c, "GetContributorList must not return empty list")
}
