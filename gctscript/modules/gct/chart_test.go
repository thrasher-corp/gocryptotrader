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
		{
			name: "invalid-incorrect args",
			args: args{
				[]tengo.Object{},
			},
			wantErr: true,
		},
		{
			name: "invalid-chartName conversion Failed",
			args: args{
				[]tengo.Object{
					tengo.FalseValue,
					tengo.FalseValue,
					&ohlcvData,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid-write file conversion",
			args: args{
				[]tengo.Object{
					&tengo.String{Value: "valid"},
					&tengo.Float{Value: 420.69},
					&ohlcvData,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid-invalid data",
			args: args{
				[]tengo.Object{
					&tengo.String{Value: "valid"},
					tengo.FalseValue,
					nil,
				},
			},
			wantErr: true,
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
