package htx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsEmptyHTXData(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		data []byte
		want bool
	}{
		{name: "empty", want: true},
		{name: "empty string", data: []byte(`""`), want: true},
		{name: "null", data: []byte(` null `), want: true},
		{name: "array", data: []byte(`[]`)},
		{name: "object", data: []byte(`{}`)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, isEmptyHTXData(tt.data), "isEmptyHTXData should identify empty HTX data values")
		})
	}
}
