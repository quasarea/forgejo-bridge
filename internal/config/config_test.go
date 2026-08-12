package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAndSelect(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	contents := []byte(`default_instance = "work"
[instances.work]
base_url = "https://forge.example/forgejo/"
credential = "env:TEST_FORGEJO_TOKEN"
allowed_repositories = ["team/z", "team/a"]
read_only = true
`)
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TEST_FORGEJO_TOKEN", "secret")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := cfg.Select("")
	if err != nil {
		t.Fatal(err)
	}
	if selected.Alias != "work" || selected.APIURL != "https://forge.example/forgejo/api/v1" {
		t.Fatalf("unexpected selection: %#v", selected)
	}
	if !selected.RepositoryAllowed("team/a") || selected.RepositoryAllowed("team/nope") {
		t.Fatal("repository allowlist was not enforced")
	}
	token, err := selected.ResolveCredential()
	if err != nil || token != "secret" {
		t.Fatalf("credential resolution failed: token=%q err=%v", token, err)
	}
}

func TestRejectsCrossOriginAPI(t *testing.T) {
	_, err := normalize(Instance{BaseURL: "https://forge.example", APIURL: "https://attacker.example/api/v1"})
	if err == nil {
		t.Fatal("expected cross-origin API URL to be rejected")
	}
}

func TestAmbiguousSelectionFails(t *testing.T) {
	cfg := &Config{Instances: map[string]Instance{"one": {}, "two": {}}}
	t.Setenv("FORGEJO_BRIDGE_INSTANCE", "")
	if _, err := cfg.Select(""); err == nil {
		t.Fatal("expected ambiguous selection to fail")
	}
}
