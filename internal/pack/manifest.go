package pack

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type Manifest struct {
	Version         string              `json:"version"`
	CreatedAt       string              `json:"created_at"`
	SourceProfile   string              `json:"source_profile"`
	SourceAgent     string              `json:"source_agent"`
	SanitizationLog []SanitizationEntry `json:"sanitization_log"`
	Skills          []string            `json:"skills"`
	MemoryFiles     []string            `json:"memory_files"`
	MCPServers      []string            `json:"mcp_servers"`
	Hooks           []string            `json:"hooks"`
	Files           []string            `json:"files,omitempty"`
	Checksum        string              `json:"checksum,omitempty"`
}

type SanitizationEntry struct {
	Timestamp   string `json:"timestamp"`
	Action      string `json:"action"` // "strip", "replace", "remove"
	Path        string `json:"path"`
	Pattern     string `json:"pattern,omitempty"`
	Replacement string `json:"replacement,omitempty"`
	Details     string `json:"details"`
}

func NewManifest(sourceProfile, sourceAgent string) *Manifest {
	return &Manifest{
		Version:         "1.0",
		CreatedAt:       time.Now().Format(time.RFC3339),
		SourceProfile:   sourceProfile,
		SourceAgent:     sourceAgent,
		SanitizationLog: []SanitizationEntry{},
		Skills:          []string{},
		MemoryFiles:     []string{},
		MCPServers:      []string{},
		Hooks:           []string{},
		Files:           []string{},
	}
}

func (m *Manifest) AddLog(action, path, pattern, replacement, details string) {
	m.SanitizationLog = append(m.SanitizationLog, SanitizationEntry{
		Timestamp:   time.Now().Format(time.RFC3339),
		Action:      action,
		Path:        path,
		Pattern:     pattern,
		Replacement: replacement,
		Details:     details,
	})
}

func (m *Manifest) Write(path string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(path, "manifest.json"), data, 0644)
}

func ReadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(filepath.Join(path, "manifest.json"))
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}
