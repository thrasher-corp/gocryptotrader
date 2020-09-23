package charts

import (
	"io"
	"reflect"
	"testing"
)

func TestChart_Generate(t *testing.T) {
	type fields struct {
		template  string
		output    string
		Data      data
		w         io.ReadWriter
		writeFile bool
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			"basic",
			fields{
				template: "basic.tmpl",
				writeFile: true,
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chart{
				template:  tt.fields.template,
				output:    tt.fields.output,
				Data:      tt.fields.Data,
				w:         tt.fields.w,
				writeFile: tt.fields.writeFile,
			}
			if err := c.Generate(); (err != nil) != tt.wantErr {
				t.Errorf("Generate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChart_Result(t *testing.T) {
	type fields struct {
		template  string
		output    string
		Data      data
		w         io.ReadWriter
		writeFile bool
	}
	tests := []struct {
		name    string
		fields  fields
		want    []byte
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chart{
				template:  tt.fields.template,
				output:    tt.fields.output,
				Data:      tt.fields.Data,
				w:         tt.fields.w,
				writeFile: tt.fields.writeFile,
			}
			got, err := c.Result()
			if (err != nil) != tt.wantErr {
				t.Errorf("Result() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Result() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewBasic(t *testing.T) {
	tests := []struct {
		name string
		want Chart
	}{
		{
			"basic",
			Chart{
				template: "basic.tmpl",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := New(tt.name); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewBasic() = %v, want %v", got, tt.want)
			}
		})
	}
}
