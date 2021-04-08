package subsystems

import "errors"

const (
	// MsgSubSystemStarting message to return when subsystem is starting up
	MsgSubSystemStarting = "starting..."
	// MsgSubSystemStarted message to return when subsystem has started
	MsgSubSystemStarted = "started."
	// MsgSubSystemShuttingDown message to return when a subsystem is shutting down
	MsgSubSystemShuttingDown = "shutting down..."
	// MsgSubSystemShutdown message to return when a subsystem has shutdown
	MsgSubSystemShutdown = "shutdown."
)

var (
	// ErrSubSystemAlreadyStarted message to return when a subsystem is already started
	ErrSubSystemAlreadyStarted = errors.New("manager already started")
	// ErrSubSystemNotStarted message to return when subsystem not started
	ErrSubSystemNotStarted = errors.New("not started")
	// ErrNilSubsystem is returned when a subsystem hasn't had its Setup() func run
	ErrNilSubsystem = errors.New("subsystem not setup")
)
