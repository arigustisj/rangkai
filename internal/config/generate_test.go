package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateModel(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.yaml")
	os.WriteFile(p, []byte("model:\n  provider: \"openrouter:a\"\nskills:\n  core: []\n"), 0o644)
	if err := UpdateModel(p, "ollama:llama3"); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(p)
	if !strings.Contains(string(b), `provider: "ollama:llama3"`) {
		t.Fatalf("not updated: %s", b)
	}
	if !strings.Contains(string(b), "skills:") {
		t.Fatalf("rest lost: %s", b)
	}
}

func TestUpdateModelMissing(t *testing.T) {
	if err := UpdateModel(filepath.Join(t.TempDir(), "nope.yaml"), "x:y"); err == nil {
		t.Fatal("expected error")
	}
}

func TestUpdateModelNoModelBlock(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.yaml")
	os.WriteFile(p, []byte("skills:\n  core: []\n"), 0o644)
	if err := UpdateModel(p, "x:y"); err == nil {
		t.Fatal("expected error when no model block")
	}
}
