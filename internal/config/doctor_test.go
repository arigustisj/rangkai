package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeDoctorConfig(t *testing.T, dir string) {
	t.Helper()
	os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("model:\n  provider: \"openrouter:a\"\nmcp:\n  enabled: [github]\n"), 0o644)
}

func TestDiagnoseHealthy(t *testing.T) {
	dir := t.TempDir()
	writeDoctorConfig(t, dir)
	os.MkdirAll(filepath.Join(dir, "skills"), 0o755)
	os.WriteFile(filepath.Join(dir, "skills", "a.md"), []byte("x"), 0o600)
	os.MkdirAll(filepath.Join(dir, "scripts"), 0o755)
	os.WriteFile(filepath.Join(dir, "scripts", "pre-commit"), []byte("#!/bin/sh\n"), 0o755)
	for _, c := range Diagnose(dir) {
		if !c.OK {
			t.Fatalf("check %s failed: %s", c.Name, c.Detail)
		}
	}
}

func TestDiagnoseMissingProfile(t *testing.T) {
	res := Diagnose(filepath.Join(t.TempDir(), "nope"))
	if len(res) == 0 || res[0].OK {
		t.Fatal("expected the profile-dir check to fail")
	}
}

func TestDiagnoseEmptySkills(t *testing.T) {
	dir := t.TempDir()
	writeDoctorConfig(t, dir)
	os.MkdirAll(filepath.Join(dir, "skills"), 0o755)
	for _, c := range Diagnose(dir) {
		if c.Name == "skills" && c.OK {
			t.Fatal("expected skills check to fail on an empty dir")
		}
	}
}
