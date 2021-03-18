package subsystem

import "errors"

const (
	// MsgSubSystemStarting message to return when subsystem is starting up
	MsgSubSystemStarting = "manager starting..."
	// MsgSubSystemStarted message to return when subsystem has started
	MsgSubSystemStarted = "started."
	// MsgSubSystemShuttingDown message to return when a subsystem is shutting down
	MsgSubSystemShuttingDown = "shutting down..."
	// MsgSubSystemShutdown message to return when a subsystem has shutdown
	MsgSubSystemShutdown = "manager shutdown."
)

var (
	// ErrSubSystemAlreadyStarted message to return when a subsystem is already started
	ErrSubSystemAlreadyStarted = errors.New("manager already started")
	// ErrSubSystemNotStarted message to return when subsystem not started
	ErrSubSystemNotStarted = errors.New("not started")
)
