package domain

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestCampus_TimezoneJSONField(t *testing.T) {
	c := Campus{
		ID:       uuid.New(),
		Name:     "Central",
		Region:   "SOUTHEAST",
		Country:  "BRA",
		Timezone: "America/Sao_Paulo",
	}

	raw, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("marshal campus: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal campus: %v", err)
	}

	got, ok := decoded["timezone"]
	if !ok {
		t.Fatalf("expected campus JSON to contain %q key, got keys: %v", "timezone", decoded)
	}
	if got != "America/Sao_Paulo" {
		t.Errorf("timezone = %v, want %q", got, "America/Sao_Paulo")
	}
}
