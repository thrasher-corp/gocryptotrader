package main

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestBuildFileList(t *testing.T) {
	tests := []struct {
		name         string
		templatePath string
		want         []string
		wantErr      bool
	}{
		{
			"valid",
			filepath.Join("testdata"),
			[]string{
				"testdata/base.tmpl",
			},
			false,
		},
	}
	for x := range tests {
		tt := tests[x]
		t.Run(tt.name, func(t *testing.T) {
			templatePath = tt.templatePath
			got, err := buildFileList()
			if (err != nil) != tt.wantErr {
				t.Errorf("buildFileList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildFileList() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestByteJoin(t *testing.T) {
	type args struct {
		b []byte
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"valid",
			args{
				[]byte("Hello"),
			},
			"{72,101,108,108,111}",
		},
	}
	for x := range tests {
		tt := tests[x]
		t.Run(tt.name, func(t *testing.T) {
			if got := byteJoin(tt.args.b); got != tt.want {
				t.Errorf("byteJoin() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateMap(t *testing.T) {
	tests := []struct {
		name         string
		templatePath string
		want         []templateData
		wantErr      bool
	}{
		{
			"valid",
			filepath.Join("testdata"),
			[]templateData{
				{
					"base.tmpl",
					"{40,226,149,175,194,176,226,150,161,194,176,239,188,137,226,149,175,239,184,181,32,226,148,187,226,148,129,226,148,187,10}",
				},
			},
			false,
		},
	}
	for x := range tests {
		tt := tests[x]
		t.Run(tt.name, func(t *testing.T) {
			templatePath = tt.templatePath
			got, err := generateMap()
			if (err != nil) != tt.wantErr {
				t.Errorf("generateMap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("generateMap() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReadTemplateToByte(t *testing.T) {
	type args struct {
		input string
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			"valid",
			args{
				filepath.Join("testdata", "base.tmpl"),
			},
			[]byte{40, 226, 149, 175, 194, 176, 226, 150, 161, 194, 176, 239, 188, 137, 226, 149, 175, 239, 184, 181, 32, 226, 148, 187, 226, 148, 129, 226, 148, 187, 10},
			false,
		},
	}
	for x := range tests {
		tt := tests[x]
		t.Run(tt.name, func(t *testing.T) {
			got, err := readTemplateToByte(tt.args.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("readTemplateToByte() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("readTemplateToByte() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStripPath(t *testing.T) {
	type args struct {
		in string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"valid",
			args{
				"valid/path",
			},
			"path",
		},
	}
	for x := range tests {
		tt := tests[x]
		templatePath = tt.name
		t.Run(tt.name, func(t *testing.T) {
			if got := stripPath(tt.args.in); got != tt.want {
				t.Errorf("stripPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
