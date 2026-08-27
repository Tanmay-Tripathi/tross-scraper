package models

// ComponentState is the health of a single dependency.
type ComponentState string

const (
	ComponentUp       ComponentState = "up"
	ComponentDown     ComponentState = "down"
	ComponentDisabled ComponentState = "disabled"
)

// ComponentHealth records whether one dependency answered, and why not if it
// did not. Error is an operator-facing detail and is never sent to clients.
type ComponentHealth struct {
	State ComponentState
	Error string
}

// Up returns the healthy state for a dependency.
func Up() ComponentHealth {
	return ComponentHealth{State: ComponentUp}
}

// Down returns the failing state for a dependency, carrying the cause.
func Down(err error) ComponentHealth {
	health := ComponentHealth{State: ComponentDown}
	if err != nil {
		health.Error = err.Error()
	}
	return health
}

// HealthStatus is the aggregate health of the service and its dependencies.
type HealthStatus struct {
	Service     string
	Version     string
	Environment string
	Database    ComponentHealth
	Cache       ComponentHealth
}

// IsHealthy reports whether every non-disabled dependency answered.
func (h HealthStatus) IsHealthy() bool {
	for _, component := range []ComponentHealth{h.Database, h.Cache} {
		if component.State == ComponentDown {
			return false
		}
	}
	return true
}
