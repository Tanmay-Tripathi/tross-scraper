// Package response holds the outbound DTOs. Keeping them separate from the models
// stops a schema change leaking a new column into the API.
package response

// ComponentResponse reports a dependency's state; the error is omitted deliberately.
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
