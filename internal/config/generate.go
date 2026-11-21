package config

import (
	"bytes"
	"embed"
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
