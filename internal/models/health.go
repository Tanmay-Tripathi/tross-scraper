package models

// ComponentState is the health of a single dependency.
type ComponentState string

const (
	ComponentUp       ComponentState = "up"
	ComponentDown     ComponentState = "down"
	ComponentDisabled ComponentState = "disabled"
)

// ComponentHealth records whether a dependency answered. Error is operator-only.
type ComponentHealth struct {
	State ComponentState
	Error string
}

// Up returns the healthy state for a dependency.
func Up() ComponentHealth {
	return ComponentHealth{State: ComponentUp}
}

// Disabled marks a dependency that is intentionally absent; it stays healthy.
func Disabled() ComponentHealth {
	return ComponentHealth{State: ComponentDisabled}
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
	// LinkedIn reports whether the scraper session still works — the one failure
	// that needs a human, so it belongs on a dashboard.
	LinkedIn ComponentHealth
}

// IsHealthy reports whether every non-disabled dependency answered.
func (h HealthStatus) IsHealthy() bool {
	for _, component := range []ComponentHealth{h.Database, h.Cache, h.LinkedIn} {
		if component.State == ComponentDown {
			return false
		}
	}
	return true
}
