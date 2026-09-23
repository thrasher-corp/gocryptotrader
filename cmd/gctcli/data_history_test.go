package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestGetDataHistoryJob(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr error
	}{
		{name: "by ID", args: []string{"--id", "job-id"}, wantErr: os.ErrNotExist},
		{name: "by nickname", args: []string{"--nickname", "job-name"}, wantErr: os.ErrNotExist},
		{name: "both identifiers", args: []string{"--id", "job-id", "--nickname", "job-name"}, wantErr: errDataHistoryJobIdentifiersConflict},
		{name: "both identifiers with empty ID", args: []string{"--id", "", "--nickname", "job-name"}, wantErr: errDataHistoryJobIdentifiersConflict},
		{name: "both identifiers with empty nickname", args: []string{"--id", "job-id", "--nickname", ""}, wantErr: errDataHistoryJobIdentifiersConflict},
		{name: "empty ID", args: []string{"--id", ""}, wantErr: errDataHistoryJobIdentifierRequired},
		{name: "empty nickname", args: []string{"--nickname", ""}, wantErr: errDataHistoryJobIdentifierRequired},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			app := &cli.App{Commands: []*cli.Command{dataHistoryCommands}}
			args := append([]string{"gctcli", "datahistory", "getajob"}, tc.args...)
			require.ErrorIs(t, app.Run(args), tc.wantErr)
		})
	}
}
