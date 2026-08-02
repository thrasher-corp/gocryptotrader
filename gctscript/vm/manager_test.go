package vm

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
)

func TestNewManager(t *testing.T) {
	t.Parallel()
	sharedConf := &Config{
		AllowImports: true,
	}
	for _, tc := range []struct {
		name    string
		config  *Config
		want    *GctScriptManager
		wantErr bool
	}{
		{
			name:    "nil config gives error",
			wantErr: true,
		},
		{
			name:   "config is applied",
			config: sharedConf,
			want: &GctScriptManager{
				config: sharedConf,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := NewManager(tc.config)
			if tc.wantErr {
				require.Error(t, err, "NewManager must return an error for invalid configuration")
				assert.Equal(t, tc.want, got, "NewManager should return the expected manager for invalid configuration")
				return
			}
			require.NoError(t, err, "NewManager must accept valid configuration")
			assert.Equal(t, tc.want, got, "NewManager should return the configured manager")
		})
	}
}

func TestGctScriptManagerStartStopNominal(t *testing.T) {
	t.Parallel()
	mgr, err := NewManager(&Config{AllowImports: true})
	require.NoError(t, err, "NewManager must create the manager")
	var wg sync.WaitGroup
	err = mgr.Start(&wg)
	require.NoError(t, err, "Start must start the manager")
	assert.Equal(t, int32(1), mgr.started, "Start should mark the manager as started")
	err = mgr.Stop()
	require.NoError(t, err, "Stop must stop the manager")
	wg.Wait()
	assert.Zero(t, mgr.started, "Stop should mark the manager as stopped")
}

func TestGctScriptManagerStartStopErrors(t *testing.T) {
	mgr, err := NewManager(&Config{AllowImports: true})
	require.NoError(t, err, "NewManager must create the manager")
	require.ErrorIs(t, mgr.Start(nil), common.ErrNilPointer, "Start must reject a nil wait group")
	require.EqualError(t, mgr.Stop(), "GCTScript not running", "Stop must reject a manager that is not running")

	var wg sync.WaitGroup
	require.NoError(t, mgr.Start(&wg), "Start must start the manager")
	require.EqualError(t, mgr.Start(&wg), "GCTScript validation failed", "Start must reject an already running manager")
	require.NoError(t, mgr.Stop(), "Stop must stop the manager")
	wg.Wait()

	var nilManager *GctScriptManager
	require.ErrorIs(t, nilManager.Stop(), ErrNilSubsystem, "Stop must reject a nil manager")
}

func TestGctScriptManagerGetMaxVirtualMachines(t *testing.T) {
	var value uint64 = 6
	for _, tc := range []struct {
		name               string
		config             *Config
		started            int32
		shutdown           chan struct{}
		maxVirtualMachines *uint64
		want               uint64
	}{
		{
			name: "get from config",
			config: &Config{
				MaxVirtualMachines: 7,
			},
			want: 7,
		},
		{
			name: "get from manager",
			config: &Config{
				MaxVirtualMachines: 7,
			},
			maxVirtualMachines: &value,
			want:               6,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := &GctScriptManager{
				config:             tc.config,
				started:            tc.started,
				shutdown:           tc.shutdown,
				MaxVirtualMachines: tc.maxVirtualMachines,
			}
			assert.Equal(t, tc.want, g.GetMaxVirtualMachines(), "GetMaxVirtualMachines should return the configured limit")
		})
	}
}
