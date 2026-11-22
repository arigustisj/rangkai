package config

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed templates/*
var templateFS embed.FS

type ProfileConfig struct {
	ProfileName string
	Stack       string
	Model       string
	Server      string
	HomeDir     string
	Variant     string // personal, team-member, ci-bot
}

func executeTemplate(name, tmplPath string, cfg ProfileConfig) (string, error) {
	tmplBytes, err := templateFS.ReadFile(tmplPath)
	if err != nil {
		return "", err
	}
	t, err := template.New(name).Parse(string(tmplBytes))
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	err = t.Execute(&buf, cfg)
	return buf.String(), err
}

func GenerateConfig(cfg ProfileConfig) (string, error) {
	return executeTemplate("config", "templates/config.yaml.tmpl", cfg)
}

func GenerateInstallScript(cfg ProfileConfig) (string, error) {
	return executeTemplate("install", "templates/install.sh.tmpl", cfg)
}

func GenerateVerifyScript(cfg ProfileConfig) (string, error) {
	return executeTemplate("verify", "templates/verify.sh.tmpl", cfg)
}

// UpdateModel rewrites the model provider in a profile's config.yaml,
// preserving every other byte. It errors if the file is missing or has no
// model block.
func UpdateModel(configPath, newModel string) error {
	if newModel == "" {
		return fmt.Errorf("model must not be empty")
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}
	info, err := os.Stat(configPath)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	inModel := false
	found := false
	for i, ln := range lines {
		if strings.HasPrefix(ln, "# Model:") {
			lines[i] = "# Model: " + newModel
			continue
		}
		if ln == "model:" {
			inModel = true
			continue
		}
		if !inModel {
			continue
		}
		trimmed := strings.TrimSpace(ln)
		if strings.HasPrefix(trimmed, "provider:") {
			indent := ln[:len(ln)-len(strings.TrimLeft(ln, " 	"))]
			lines[i] = indent + "provider: \"" + newModel + "\""
			found = true
			inModel = false
			continue
		}
		// The model block ends at the next top-level key.
		if trimmed != "" && !strings.HasPrefix(ln, " ") && !strings.HasPrefix(ln, "	") && !strings.HasPrefix(ln, "#") {
			inModel = false
		}
	}
	if !found {
		return fmt.Errorf("no model provider found in %s", configPath)
	}
	return os.WriteFile(configPath, []byte(strings.Join(lines, "\n")), info.Mode())
}

// modelProvider extracts the provider value from a config.yaml body.
func modelProvider(cfg string) string {
	inModel := false
	for _, ln := range strings.Split(cfg, "\n") {
		if ln == "model:" {
			inModel = true
			continue
		}
		if !inModel {
			continue
		}
		t := strings.TrimSpace(ln)
		if strings.HasPrefix(t, "provider:") {
			v := strings.TrimSpace(strings.TrimPrefix(t, "provider:"))
			return strings.Trim(v, "\"")
		}
		if t != "" && !strings.HasPrefix(ln, " ") && !strings.HasPrefix(ln, "	") && !strings.HasPrefix(ln, "#") {
			inModel = false
		}
	}
	return ""
}

// Check is one diagnostic result.
type Check struct {
	Name   string
	OK     bool
	Detail string
}

// Diagnose runs read-only checks over a profile directory. It never mutates it.
func Diagnose(profileDir string) []Check {
	var out []Check
	emit := func(name string, ok bool, detail string) {
		out = append(out, Check{Name: name, OK: ok, Detail: detail})
	}

	info, err := os.Stat(profileDir)
	emit("profile-dir", err == nil && info.IsDir(), profileDir)
	if err != nil || !info.IsDir() {
		for _, n := range []string{"config", "model", "skills", "mcp", "hooks"} {
			emit(n, false, "profile missing")
		}
		return out
	}

	cfgPath := filepath.Join(profileDir, "config.yaml")
	cfgData, cfgErr := os.ReadFile(cfgPath)
	cfg := string(cfgData)
	emit("config", cfgErr == nil, cfgPath)
	provider := ""
	if cfgErr == nil {
		provider = modelProvider(cfg)
	}
	emit("model", provider != "", "provider: \""+provider+"\"")

	entries, skillsErr := os.ReadDir(filepath.Join(profileDir, "skills"))
	emit("skills", skillsErr == nil && len(entries) > 0, fmt.Sprintf("%d entries", len(entries)))

	emit("mcp", cfgErr == nil && strings.Contains(cfg, "mcp:"), "config.yaml mcp section")

	hookPath := filepath.Join(profileDir, "scripts", "pre-commit")
	hinfo, herr := os.Stat(hookPath)
	emit("hooks", herr == nil && !hinfo.IsDir() && hinfo.Mode()&0o111 != 0, hookPath)

	return out
}

// ReadProfile parses a profile's config.yaml back into a ProfileConfig.
func ReadProfile(configPath string) (ProfileConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return ProfileConfig{}, err
	}
	s := string(data)
	cfg := ProfileConfig{}
	for _, ln := range strings.Split(s, "\n") {
		if strings.HasPrefix(ln, "# Profile:") {
			cfg.ProfileName = strings.TrimSpace(strings.TrimPrefix(ln, "# Profile:"))
		}
		if strings.HasPrefix(ln, "# Stack:") {
			cfg.Stack = strings.TrimSpace(strings.TrimPrefix(ln, "# Stack:"))
		}
	}
	cfg.Model = modelProvider(s)
	if cfg.ProfileName == "" || cfg.Stack == "" || cfg.Model == "" {
		return ProfileConfig{}, fmt.Errorf("incomplete profile config: %s", configPath)
	}
	return cfg, nil
}
