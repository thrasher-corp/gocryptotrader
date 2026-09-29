package messages

import (
	"testing"

	"github.com/stretchr/testify/assert"
	r "github.com/stretchr/testify/require"
)

func messages(t *testing.T, dynamic string, args []any) {
	assert.Equal(t, "must contain %s", "must contain %s", "values should match")
	r.NoError(t, nil, "response must be valid")
	assert.Equalf(t, 1, 1, "values should match %d", 1)
	assert.Equalf(t, 1, 1, "values should match %[1]*.[2]*[3]f", 5, 2, 1.0)
	assert.Equalf(t, 1, 1, "progress should be 100%%")
	assert.Equalf(t, 1, 1, "custom formatter should accept %☃", 1)
	assert.Equal(t, 1, 1, "progress should be 100%")
	assert.Equal(t, 1, 1, "mustard should match")
	assert.Equal(t, 1, 1)
	assert.Equal(t, 1, 1, "matching values")
	assert.Equal(t, 1, 1, dynamic)
	assert.Equal(t, 1, 1, args...)
	assert.Equalf(t, 1, 1, dynamic, 1)
	assert.Equalf(t, 1, 1, "should match %[")     // Malformed formats belong to testifylint.
	assert.Equalf(t, 1, 1, "should match")        // want "has a plain message; use Equal"
	assert.Equal(t, 1, 1, "should match %d", 1)   // want "has a format message; use Equalf"
	assert.Equal(t, 1, 1, "should match %%")      // want "has a format message; use Equalf"
	assert.Equal(t, 1, 1, "should match %%%d", 1) // want "has a format message; use Equalf"
	r.NoErrorf(t, nil, "must succeed")            // want "has a plain message; use NoError"
	r.NoError(t, nil, "operation Should succeed") // want "message uses .should.; use .must."
	assert.Equal(
		t, 1, 1,
		"values MUST match", // want "message uses .must.; use .should."
	)
	const message = "values " + "must match"
	assert.Equal(t, 1, 1, message) // want "message uses .must.; use .should."
	assert.PanicsWithError(
		t, "must panic", func() {},
		"operation must panic", // want "message uses .must.; use .should."
	)
	assert.New(t).Equalf(1, 1, "should match")                       // want "has a plain message; use Equal"
	r.New(t).NoError(nil, "should work")                             // want "message uses .should.; use .must."
	(*assert.Assertions).Equalf(assert.New(t), 1, 1, "should match") // want "has a plain message; use Equal"
	var suite struct{ *assert.Assertions }
	suite.Equal(1, 1, "should match %d", 1)    // want "has a format message; use Equalf"
	new(assert.CollectT).Errorf("should fail") // No non-f counterpart exists.
	custom(t, "must be left alone %s")
	{
		assert := struct{ Equal func(...any) }{}
		assert.Equal(t, 1, 1, "must be left alone %s")
	}
}

func custom(t any, message string) {}
