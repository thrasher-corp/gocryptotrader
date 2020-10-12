package modules

import "testing"

func TestToFloat64(t *testing.T) {
	type args struct {
		data interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    float64
		wantErr bool
	}{
		{
			"valid-int",
			args{
				42,
			},
			42.0,
			false,
		},
		{
			"valid-int32",
			args{
				int32(42),
			},
			42.0,
			false,
		},
		{
			"valid-int64",
			args{
				int64(42),
			},
			42.0,
			false,
		},
		{
			"valid-float64",
			args{
				42.0,
			},
			42.0,
			false,
		},
		{
			"invalid-string",
			args{
				"helloworld",
			},
			0,
			true,
		},
	}
	for x := range tests {
		tt := tests[x]
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToFloat64(tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("modules.ToFloat64() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("modules.ToFloat64() got = %v, want %v", got, tt.want)
			}
		})
	}
}
