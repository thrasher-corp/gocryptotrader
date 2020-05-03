package gct

import (
	"testing"

	objects "github.com/d5/tengo/v2"
)

var matrix = &objects.Array{
	Value: []objects.Object{
		&objects.Array{
			Value: []objects.Object{
				&objects.String{Value: "hello"},
				&objects.String{Value: "world"},
			},
		},
		&objects.Array{
			Value: []objects.Object{
				&objects.String{Value: "one"},
				&objects.String{Value: "two"},
			},
		},
		&objects.Array{
			Value: []objects.Object{
				&objects.String{Value: "3"},
				&objects.String{Value: "4"},
			},
		},
	},
}

var badMatrix = &objects.Array{
	Value: []objects.Object{
		&objects.Array{
			Value: []objects.Object{
				&objects.String{Value: "hello"},
				&objects.String{Value: "world"},
			},
		},
		&objects.Array{
			Value: []objects.Object{
				&objects.String{Value: "one"},
				&objects.String{Value: "two"},
			},
		},
		&objects.Array{
			Value: []objects.Object{
				&objects.String{Value: "3"},
				&objects.String{Value: "4"},
				&objects.String{Value: "LOLOLOLOLOL"},
			},
		},
	},
}

func TestCommonWriteToCSV(t *testing.T) {
	t.Parallel()

	// tempDir := filepath.Join(os.TempDir(), "script-temp")
	// testFile := filepath.Join(tempDir, "script-test.csv")

	_, err := WriteAsCSV(matrix)
	if err != nil {
		t.Fatal(err)
	}
}
