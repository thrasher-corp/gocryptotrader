package gct

import (
	"reflect"
	"testing"
	"time"

	"github.com/d5/tengo/v2"
)

var (
	ohlcvData = tengo.Array{
		Value: []tengo.Object{
			&tengo.Array{
				Value: []tengo.Object{
					&tengo.Int{Value: time.Now().Unix()},
					&tengo.Float{Value: 1},
					&tengo.Float{Value: 2},
					&tengo.Float{Value: 3},
					&tengo.Float{Value: 4},
					&tengo.Float{Value: 5},
				},
			},
			&tengo.Array{
				Value: []tengo.Object{
					&tengo.Int{Value: time.Now().Unix()},
					&tengo.Float{Value: 1},
					&tengo.Float{Value: 2},
					&tengo.Float{Value: 3},
					&tengo.Float{Value: 4},
					&tengo.Float{Value: 5},
				},
			},
			&tengo.Array{
				Value: []tengo.Object{
					&tengo.Int{Value: time.Now().Unix()},
					&tengo.Float{Value: 1},
					&tengo.Float{Value: 2},
					&tengo.Float{Value: 3},
					&tengo.Float{Value: 4},
					&tengo.Float{Value: 5},
				},
			},
		},
	}
)

func TestGenerateChart(t *testing.T) {
	type args struct {
		data []tengo.Object
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				[]tengo.Object{
					&tengo.String{Value: "valid"},
					tengo.FalseValue,
					&ohlcvData,
				},
			},
			wantErr: false,
		},
	}
	for x := range tests {
		tt := tests[x]
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateChart(tt.args.data...)
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
