package contract

import "fmt"

// Health is the response of GET /v1/health.
type Health struct {
	Status        string `json:"status"`
	Version       string `json:"version"`
	SchemaVersion int    `json:"schema_version"`
}

// Validate checks the health contract.
func (h Health) Validate() error {
	if h.Status != "ok" {
		return fmt.Errorf("status must be ok")
	}
	if h.Version == "" {
		return fmt.Errorf("version must not be empty")
	}
	if h.SchemaVersion < 1 {
		return fmt.Errorf("schema_version must be positive")
	}
	return nil
}
