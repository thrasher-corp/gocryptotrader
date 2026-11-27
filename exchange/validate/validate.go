package validate

// Checker defines validation check functionality
type Checker interface {
	Check() error
}

// Check defines a validation check function to close over individual validation
// check methods
type Check func() error

// Check initiates the Check functionality
func (v Check) Check() error {
	return v()
}
