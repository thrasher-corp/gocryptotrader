package gct

import (
	"reflect"
	"testing"

	"github.com/d5/tengo/v2"
)

func Test_generateChart(t *testing.T) {
	type args struct {
		args []tengo.Object
	}
	tests := []struct {
		name    string
		args    args
		want    tengo.Object
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				[]tengo.Object{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := generateChart(tt.args.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("generateChart() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("generateChart() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_toFloat64(t *testing.T) {
	type args struct {
		data interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    float64
		wantErr bool
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toFloat64(tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("toFloat64() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("toFloat64() got = %v, want %v", got, tt.want)
			}
		})
	}
}
