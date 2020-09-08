package engine

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/config"
)

func Test_loadConfigWithSettings(t *testing.T) {
	empty := ""
	somePath := "somePath"
	tests := []struct {
		name     string
		settings *Settings
		want     *string
		wantErr  bool
	}{
		{
			name:     "empty",
			settings: &Settings{},
			wantErr:  true,
		},
		{
			name: "invalid file",
			settings: &Settings{
				ConfigFile: "nonExistent.json",
			},
			wantErr: true,
		},
		{
			name: "test file",
			settings: &Settings{
				ConfigFile:   config.TestFile,
				EnableDryRun: true,
			},
			want:    &empty,
			wantErr: false,
		},
		{
			name: "data dir in settings overrides config data dir",
			settings: &Settings{
				ConfigFile:   config.TestFile,
				DataDir:      somePath,
				EnableDryRun: true,
			},
			want:    &somePath,
			wantErr: false,
		},
	}
	config.TestBypass = true
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := loadConfigWithSettings(tt.settings)
			if (err != nil) != tt.wantErr {
				t.Errorf("loadConfigWithSettings() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != nil || tt.want != nil {
				if (got == nil && tt.want != nil) || (got != nil && tt.want == nil) {
					t.Errorf("loadConfigWithSettings() = is nil %v, want nil %v", got == nil, tt.want == nil)
				} else if got.DataDir != *tt.want {
					t.Errorf("loadConfigWithSettings() = %v, want %v", got.DataDir, *tt.want)
				}
			}
		})
	}
}
