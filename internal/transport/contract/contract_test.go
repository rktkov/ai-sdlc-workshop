package contract

import (
	"encoding/json"
	"os"
	"testing"
)

func TestHealthValidateCases(t *testing.T) {
	tests := []struct {
		name   string
		health Health
		wantOK bool
	}{
		{name: "valid", health: Health{Status: "ok", Version: "dev", SchemaVersion: 1}, wantOK: true},
		{name: "status", health: Health{Status: "down", Version: "dev", SchemaVersion: 1}},
		{name: "version", health: Health{Status: "ok", Version: "", SchemaVersion: 1}},
		{name: "schema", health: Health{Status: "ok", Version: "dev", SchemaVersion: 0}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.health.Validate()
			if tt.wantOK && err != nil {
				t.Fatalf("Validate() = %v", err)
			}
			if !tt.wantOK && err == nil {
				t.Fatal("Validate() = nil, want an error")
			}
		})
	}
}

func TestHealthContractHappy(t *testing.T) {
	raw, err := os.ReadFile("testdata/health.json")
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}

	var health Health
	if err := json.Unmarshal(raw, &health); err != nil {
		t.Fatalf("decode golden: %v", err)
	}
	if err := health.Validate(); err != nil {
		t.Fatalf("Validate() = %v", err)
	}
}

func TestHealthContractUnhappy(t *testing.T) {
	raw, err := os.ReadFile("testdata/health-missing-schema.json")
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}

	var health Health
	if err := json.Unmarshal(raw, &health); err != nil {
		t.Fatalf("decode golden: %v", err)
	}
	if err := health.Validate(); err == nil {
		t.Fatal("Validate() = nil, want an error")
	}
}

func TestHealthGolden(t *testing.T) {
	raw, err := os.ReadFile("testdata/health.json")
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("decode golden: %v", err)
	}

	for _, key := range []string{"status", "version", "schema_version"} {
		if _, ok := payload[key]; !ok {
			t.Fatalf("golden is missing %q", key)
		}
	}
}

func TestHealthGoldenRejectsMissingField(t *testing.T) {
	raw, err := os.ReadFile("testdata/health-missing-schema.json")
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("decode golden: %v", err)
	}

	if _, ok := payload["schema_version"]; ok {
		t.Fatal("invalid golden unexpectedly has schema_version")
	}
}

func TestGoldenRoundTrip(t *testing.T) {
	raw, err := os.ReadFile("testdata/health.json")
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("decode golden: %v", err)
	}

	out, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode golden: %v", err)
	}

	var again map[string]any
	if err := json.Unmarshal(out, &again); err != nil {
		t.Fatalf("decode round trip: %v", err)
	}
	if again["status"] != payload["status"] {
		t.Fatalf("status changed: %v -> %v", payload["status"], again["status"])
	}
}
