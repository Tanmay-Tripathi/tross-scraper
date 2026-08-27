// Package response holds the outbound DTOs controllers send to clients. Keeping
// them separate from internal/models means a schema change never silently
// leaks a new database column into the API.
package response

// ComponentResponse reports one dependency's state. The underlying error is
// deliberately omitted so internal detail never reaches a client.
type ComponentResponse struct {
	Status string `json:"status"`
}

// HealthResponse is the payload of the health endpoints.
type HealthResponse struct {
	Status       string                       `json:"status"`
	Service      string                       `json:"service"`
	Version      string                       `json:"version"`
	Environment  string                       `json:"environment"`
	Dependencies map[string]ComponentResponse `json:"dependencies"`
}
