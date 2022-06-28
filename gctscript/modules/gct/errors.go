package gct

import (
	"fmt"

	objects "github.com/d5/tengo/v2"
)

// errorResponse is a helper function to apply error details to a return object
// for better script side error handling
func errorResponse(format string, a ...interface{}) (objects.Object, error) {
	return &objects.Error{
		Value: &objects.String{Value: fmt.Sprintf(format, a...)},
	}, nil
}
