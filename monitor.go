package status_neko

import "context"

// Monitor is the interface that wraps the basic methods for monitoring the status of a service.
type Monitor interface {
	// Name returns the name of the monitor.
	Name() string

	// Check returns the status of the service.
	Check(ctx context.Context) (interface{}, error)
}
