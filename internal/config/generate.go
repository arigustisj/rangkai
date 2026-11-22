package config

import (
	"bytes"
	"embed"
	"fmt"
	"os"
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
