package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
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

func TestDataHistoryJobRequest(t *testing.T) {
	tests := []struct {
		name        string
		commandName string
		args        []string
		want        *gctrpc.GetDataHistoryJobDetailsRequest
		wantErr     error
	}{
		{name: "by ID", commandName: "getajob", args: []string{"--id", "job-id"}, want: &gctrpc.GetDataHistoryJobDetailsRequest{Id: "job-id"}},
		{name: "by nickname", commandName: "getajob", args: []string{"--nickname", "job-name"}, want: &gctrpc.GetDataHistoryJobDetailsRequest{Nickname: "job-name"}},
		{name: "detailed results", commandName: "getjobwithdetailedresults", args: []string{"--nickname", "job-name"}, want: &gctrpc.GetDataHistoryJobDetailsRequest{Nickname: "job-name", FullDetails: true}},
		{name: "both identifiers", commandName: "getajob", args: []string{"--id", "job-id", "--nickname", "job-name"}, wantErr: errDataHistoryJobIdentifiersConflict},
		{name: "empty identifier", commandName: "getajob", args: []string{"--id", ""}, wantErr: errDataHistoryJobIdentifierRequired},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got *gctrpc.GetDataHistoryJobDetailsRequest
			command := &cli.Command{Name: tc.commandName, Flags: specificJobSubCommands, Action: func(c *cli.Context) error {
				var err error
				got, err = dataHistoryJobRequest(c)
				return err
			}}
			app := &cli.App{Commands: []*cli.Command{command}}
			err := app.Run(append([]string{"gctcli", tc.commandName}, tc.args...))
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr, "request builder must reject invalid identifiers")
				return
			}
			require.NoError(t, err, "request builder must accept a valid identifier")
			assert.Equal(t, tc.want, got, "request should use the selected identifier")
		})
	}
}

func TestDataHistoryJobIdentifier(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantID       string
		wantNickname string
		wantErr      error
	}{
		{name: "ID", args: []string{"--id", "job-id"}, wantID: "job-id"},
		{name: "nickname", args: []string{"--nickname", "job-name"}, wantNickname: "job-name"},
		{name: "missing", wantErr: errDataHistoryJobIdentifierRequired},
		{name: "empty ID", args: []string{"--id", ""}, wantErr: errDataHistoryJobIdentifierRequired},
		{name: "empty nickname", args: []string{"--nickname", ""}, wantErr: errDataHistoryJobIdentifierRequired},
		{name: "both", args: []string{"--id", "job-id", "--nickname", "job-name"}, wantErr: errDataHistoryJobIdentifiersConflict},
		{name: "ID and empty nickname", args: []string{"--id", "job-id", "--nickname", ""}, wantErr: errDataHistoryJobIdentifiersConflict},
		{name: "empty ID and nickname", args: []string{"--id", "", "--nickname", "job-name"}, wantErr: errDataHistoryJobIdentifiersConflict},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			command := &cli.Command{Name: "selectjob", Flags: specificJobSubCommands, Action: func(c *cli.Context) error {
				id, nickname, err := dataHistoryJobIdentifier(c)
				if err != nil {
					return err
				}
				assert.Equal(t, tc.wantID, id, "job ID should match selected value")
				assert.Equal(t, tc.wantNickname, nickname, "nickname should match selected value")
				return nil
			}}
			app := &cli.App{Commands: []*cli.Command{command}}
			err := app.Run(append([]string{"gctcli", "selectjob"}, tc.args...))
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr, "invalid identifiers should return the expected error")
				return
			}
			require.NoError(t, err, "valid identifier must be accepted")
		})
	}
}

func TestSetDataHistoryJobStatus(t *testing.T) {
	for _, name := range []string{"deletejob", "pausejob", "unpausejob"} {
		t.Run(name, func(t *testing.T) {
			for _, tc := range []struct {
				name    string
				args    []string
				wantErr error
			}{
				{name: "empty ID", args: []string{"--id", ""}, wantErr: errDataHistoryJobIdentifierRequired},
				{name: "both", args: []string{"--id", "job-id", "--nickname", "job-name"}, wantErr: errDataHistoryJobIdentifiersConflict},
				{name: "ID and empty nickname", args: []string{"--id", "job-id", "--nickname", ""}, wantErr: errDataHistoryJobIdentifiersConflict},
				{name: "valid ID", args: []string{"--id", "job-id"}, wantErr: os.ErrNotExist},
			} {
				t.Run(tc.name, func(t *testing.T) {
					app := &cli.App{Commands: []*cli.Command{dataHistoryCommands}}
					err := app.Run(append([]string{"gctcli", "datahistory", name}, tc.args...))
					assert.ErrorIs(t, err, tc.wantErr, "status command should validate the selected identifier")
				})
			}
		})
	}
}
