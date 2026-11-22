package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadProfile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.yaml")
	os.WriteFile(p, []byte("# Profile: personal\n# Stack: go\n# Model: openrouter:a\nmodel:\n  provider: \"openrouter:a\"\n"), 0o644)
	cfg, err := ReadProfile(p)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ProfileName != "personal" || cfg.Stack != "go" || cfg.Model != "openrouter:a" {
		t.Fatalf("parsed %+v", cfg)
	}
}

func TestReadProfileMissing(t *testing.T) {
	if _, err := ReadProfile(filepath.Join(t.TempDir(), "nope.yaml")); err == nil {
		t.Fatal("expected error")
	}
}

func TestReadProfileIncomplete(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.yaml")
	os.WriteFile(p, []byte("# Profile: personal\n"), 0o644)
	if _, err := ReadProfile(p); err == nil {
		t.Fatal("expected error for an incomplete config")
	}
}
