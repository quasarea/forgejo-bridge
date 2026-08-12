package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/quasarea/forgejo-bridge/internal/contracts"
)

func TestVersionCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	runner := Runner{Stdout: &stdout, Stderr: &stderr}
	if code := runner.Run(context.Background(), []string{"version"}); code != 0 {
		t.Fatalf("exit code = %d stderr=%s", code, stderr.String())
	}
	var envelope contracts.Envelope
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if !envelope.OK || envelope.Operation != "version" {
		t.Fatalf("unexpected envelope: %#v", envelope)
	}
}

func TestUnknownCommandIsStructured(t *testing.T) {
	var stdout, stderr bytes.Buffer
	runner := Runner{Stdout: &stdout, Stderr: &stderr}
	if code := runner.Run(context.Background(), []string{"wat"}); code != 2 {
		t.Fatalf("exit code = %d", code)
	}
	var envelope contracts.Envelope
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.OK || envelope.Error == nil || envelope.Error.Code != "invalid_input" {
		t.Fatalf("unexpected envelope: %#v", envelope)
	}
}
