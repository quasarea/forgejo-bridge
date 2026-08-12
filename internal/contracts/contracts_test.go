package contracts

import (
	"encoding/json"
	"testing"
)

func TestSuccessEnvelopeContract(t *testing.T) {
	envelope := Success("repo.get", "request-1", "work", map[string]string{"name": "service"})
	raw, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["schema_version"] != SchemaVersion {
		t.Fatalf("schema version = %v", decoded["schema_version"])
	}
	if decoded["ok"] != true {
		t.Fatalf("ok = %v", decoded["ok"])
	}
	if warnings, ok := decoded["warnings"].([]any); !ok || len(warnings) != 0 {
		t.Fatalf("warnings must be an empty array: %#v", decoded["warnings"])
	}
}

func TestFailureEnvelopeDoesNotExposeCause(t *testing.T) {
	err := WrapError("authentication_failed", "authentication failed", assertSecretError("token-secret"))
	raw, marshalErr := json.Marshal(Failure("repo.list", "request-1", "work", err))
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	if string(raw) == "" || contains(string(raw), "token-secret") {
		t.Fatalf("serialized error exposed its cause: %s", raw)
	}
}

type assertSecretError string

func (e assertSecretError) Error() string { return string(e) }

func contains(value, substring string) bool {
	for i := 0; i+len(substring) <= len(value); i++ {
		if value[i:i+len(substring)] == substring {
			return true
		}
	}
	return false
}
