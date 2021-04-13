package engine

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/config"
)

func TestLoadConfigWithSettings(t *testing.T) {
	empty := ""
	somePath := "somePath"
	// Clean up after the tests
	defer os.RemoveAll(somePath)
	tests := []struct {
		name     string
		flags    []string
		settings *Settings
		want     *string
		wantErr  bool
	}{
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
			name:  "data dir in settings overrides config data dir",
			flags: []string{"datadir"},
			settings: &Settings{
				ConfigFile:   config.TestFile,
				DataDir:      somePath,
				EnableDryRun: true,
			},
			want:    &somePath,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			// prepare the 'flags'
			flagSet := make(map[string]bool)
			for _, v := range tt.flags {
				flagSet[v] = true
			}
			// Run the test
			got, err := loadConfigWithSettings(tt.settings, flagSet)
			if (err != nil) != tt.wantErr {
				t.Errorf("loadConfigWithSettings() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != nil || tt.want != nil {
				if (got == nil && tt.want != nil) || (got != nil && tt.want == nil) {
					t.Errorf("loadConfigWithSettings() = is nil %v, want nil %v", got == nil, tt.want == nil)
				} else if got.DataDirectory != *tt.want {
					t.Errorf("loadConfigWithSettings() = %v, want %v", got.DataDirectory, *tt.want)
				}
			}
		})
	}
}

func TestStartStopDoesNotCausePanic(t *testing.T) {
	t.Parallel()
	botOne, err := NewFromSettings(&Settings{
		ConfigFile:   config.TestFile,
		EnableDryRun: true,
	}, nil)
	if err != nil {
		t.Error(err)
	}

	if err = botOne.Start(); err != nil {
		t.Error(err)
	}

	botOne.Stop()
}

func TestStartStopTwoDoesNotCausePanic(t *testing.T) {
	t.Parallel()
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("Problem creating temp dir at %s: %s\n", tempDir, err)
	}
	tempDir2, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("Problem creating temp dir at %s: %s\n", tempDir, err)
	}
	defer func() {
		err := os.RemoveAll(tempDir)
		if err != nil {
			t.Error(err)
		}
		err = os.RemoveAll(tempDir2)
		if err != nil {
			t.Error(err)
		}
	}()
	botOne, err := NewFromSettings(&Settings{
		ConfigFile:   config.TestFile,
		EnableDryRun: true,
		DataDir:      tempDir,
	}, nil)
	if err != nil {
		t.Error(err)
	}
	botTwo, err := NewFromSettings(&Settings{
		ConfigFile:   config.TestFile,
		EnableDryRun: true,
		DataDir:      tempDir2,
	}, nil)
	if err != nil {
		t.Error(err)
	}

	if err = botOne.Start(); err != nil {
		t.Error(err)
	}
	if err = botTwo.Start(); err != nil {
		t.Error(err)
	}

	botOne.Stop()
	botTwo.Stop()
}
