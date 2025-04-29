package core

import (
	"context"
)

// Interface defines a common interface for all services
type Interface interface {
	Start(ctx context.Context) error
	Stop()
}

// Registry manages all services
type Registry struct {
	services []Interface
}

// NewRegistry creates a new core registry
func NewRegistry() *Registry {
	return &Registry{
		services: make([]Interface, 0),
	}
}

// Register adds a core to the registry
func (sr *Registry) Register(service Interface) {
	sr.services = append(sr.services, service)
}

// StartAll starts all registered services
func (sr *Registry) StartAll(ctx context.Context) error {
	for _, service := range sr.services {
		if err := service.Start(ctx); err != nil {
			return err
		}
	}
	return nil
}

// StopAll stops all registered services
func (sr *Registry) StopAll() {
	// Stop in reverse order
	for i := len(sr.services) - 1; i >= 0; i-- {
		sr.services[i].Stop()
	}
}
