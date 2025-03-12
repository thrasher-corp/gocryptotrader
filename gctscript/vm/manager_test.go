package vm

import (
	"reflect"
	"sync"
	"testing"
)

func TestNewManager(t *testing.T) {
	t.Parallel()
	type args struct {
		config *Config
	}
	sharedConf := &Config{
		AllowImports: true,
	}
	tests := []struct {
		name    string
		args    args
		want    *GctScriptManager
		wantErr bool
	}{
		{
			name:    "nil config gives error",
			wantErr: true,
		},
		{
			name: "config is applied",
			args: args{
				config: sharedConf,
			},
			want: &GctScriptManager{
				config: sharedConf,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := NewManager(tt.args.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewManager() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewManager() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGctScriptManagerStartStopNominal(t *testing.T) {
	t.Parallel()
	mgr, err := NewManager(&Config{AllowImports: true})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	err = mgr.Start(&wg)
	if err != nil {
		t.Fatal(err)
	}
	if mgr.started != 1 {
		t.Errorf("Manager should be started (%v)", mgr.started)
	}
	err = mgr.Stop()
	if err != nil {
		t.Fatal(err)
	}
	wg.Wait()
	if mgr.started != 0 {
		t.Errorf("Manager should be stopped, expected=%v, got %v", 0, mgr.started)
	}
}

func TestGctScriptManagerGetMaxVirtualMachines(t *testing.T) {
	type fields struct {
		config             *Config
		started            int32
		shutdown           chan struct{}
		MaxVirtualMachines *uint64
	}
	var value uint64 = 6
	tests := []struct {
		name   string
		fields fields
		want   uint64
	}{
		{
			name: "get from config",
			fields: fields{
				config: &Config{
					MaxVirtualMachines: 7,
				},
			},
			want: 7,
		},
		{
			name: "get from manager",
			fields: fields{
				config: &Config{
					MaxVirtualMachines: 7,
				},
				MaxVirtualMachines: &value,
			},
			want: 6,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &GctScriptManager{
				config:             tt.fields.config,
				started:            tt.fields.started,
				shutdown:           tt.fields.shutdown,
				MaxVirtualMachines: tt.fields.MaxVirtualMachines,
			}
			if got := g.GetMaxVirtualMachines(); got != tt.want {
				t.Errorf("GctScriptManager.GetMaxVirtualMachines() = %v, want %v", got, tt.want)
			}
		})
	}
}
