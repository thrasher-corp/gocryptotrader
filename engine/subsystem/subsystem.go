package subsystem

const (
	// ErrSubSystemAlreadyStarted message to return when a subsystem is already started
	ErrSubSystemAlreadyStarted = "manager already started"
	// ErrSubSystemAlreadyStopped message to return when a subsystem is already stopped
	ErrSubSystemAlreadyStopped = "already stopped"
	// ErrSubSystemNotStarted message to return when subsystem not started
	ErrSubSystemNotStarted = "not started"

	// MsgSubSystemStarting message to return when subsystem is starting up
	MsgSubSystemStarting = "manager starting..."
	// MsgSubSystemStarted message to return when subsystem has started
	MsgSubSystemStarted = "started."

	// MsgSubSystemShuttingDown message to return when a subsystem is shutting down
	MsgSubSystemShuttingDown = "shutting down..."
	// MsgSubSystemShutdown message to return when a subsystem has shutdown
	MsgSubSystemShutdown = "manager shutdown."
)
