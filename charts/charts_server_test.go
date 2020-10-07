package charts

import (
	"io"
	"net/http"
	"testing"
)

func TestChart_Serve(t *testing.T) {
	type fields struct {
		template     string
		TemplatePath string
		output       string
		OutputPath   string
		Data         Data
		w            io.ReadWriter
		WriteFile    bool
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chart{
				template:     tt.fields.template,
				TemplatePath: tt.fields.TemplatePath,
				output:       tt.fields.output,
				OutputPath:   tt.fields.OutputPath,
				Data:         tt.fields.Data,
				w:            tt.fields.w,
				WriteFile:    tt.fields.WriteFile,
			}
			t.Log(c)
		})
	}
}

func TestChart_handler(t *testing.T) {
	type fields struct {
		template     string
		TemplatePath string
		output       string
		OutputPath   string
		Data         Data
		w            io.ReadWriter
		WriteFile    bool
	}
	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chart{
				template:     tt.fields.template,
				TemplatePath: tt.fields.TemplatePath,
				output:       tt.fields.output,
				OutputPath:   tt.fields.OutputPath,
				Data:         tt.fields.Data,
				w:            tt.fields.w,
				WriteFile:    tt.fields.WriteFile,
			}
			t.Log(c)
		})
	}
}
