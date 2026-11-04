package contract

import (
	"encoding/json"
	"os"
	"testing"
)

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
