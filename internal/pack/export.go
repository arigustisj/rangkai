package pack

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/arigustisj/rangkai/internal/skill"
)

// Layer identifies a pack layer. Structure is always exported; the rest are opt-in.
type Layer string

const (
	LayerStructure Layer = "structure"
	LayerMemory    Layer = "memory"
	LayerSecrets   Layer = "secrets"
	LayerSSOT      Layer = "ssot"
)

// ExportOptions selects which layers an export includes.
type ExportOptions struct {
	Layers []Layer
}

// Has reports whether a layer is enabled.
func (o ExportOptions) Has(l Layer) bool {
	for _, x := range o.Layers {
		if x == l {
			return true
		}
	}
	return false
}

// DefaultExportOptions returns the shareable default: structure only.
func DefaultExportOptions() ExportOptions {
	return ExportOptions{Layers: []Layer{LayerStructure}}
}

// MethodologyMemory lists the memory files that belong to the structure layer.
var MethodologyMemory = []string{
	"conventions.md", "server-map.md", "git-rules.md",
	"deploy-patterns.md", "agent-rules.md", "mcp-registry.md",
}

type Exporter struct {
	SourceProfile string
	SourceAgent   string
	OutputPath    string
	Sanitizer     *Sanitizer
	Manifest      *Manifest
	SkillRegistry *skill.SkillRegistry
	Options       ExportOptions
	// Home overrides home directory resolution when non-empty (tests).
	Home string
}

// home resolves the home directory, honoring the Home test seam.
func (e *Exporter) home() string {
	if e.Home != "" {
		return e.Home
	}
	h, _ := os.UserHomeDir()
	return h
}

// NewExporter keeps the historical behaviour: structure + memory.
func NewExporter(sourceProfile, sourceAgent, outputPath string, sanitizer *Sanitizer, manifest *Manifest, registry *skill.SkillRegistry) *Exporter {
	return NewExporterWithOptions(sourceProfile, sourceAgent, outputPath, sanitizer, manifest, registry, ExportOptions{Layers: []Layer{LayerStructure, LayerMemory}})
}

func NewExporterWithOptions(sourceProfile, sourceAgent, outputPath string, sanitizer *Sanitizer, manifest *Manifest, registry *skill.SkillRegistry, opts ExportOptions) *Exporter {
	return &Exporter{
		SourceProfile: sourceProfile,
		SourceAgent:   sourceAgent,
		OutputPath:    outputPath,
		Sanitizer:     sanitizer,
		Manifest:      manifest,
		SkillRegistry: registry,
		Options:       opts,
	}
}

func (e *Exporter) Export() error {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "rangkai-pack-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Export skills
	if err := e.exportSkills(tmpDir); err != nil {
		return fmt.Errorf("export skills: %w", err)
	}

	// Export memory (structure methodology files always; personal memory opt-in)
	if err := e.exportMemory(tmpDir, e.Options.Has(LayerMemory)); err != nil {
		return fmt.Errorf("export memory: %w", err)
	}

	// Export secrets (opt-in, private)
	if e.Options.Has(LayerSecrets) {
		if err := e.exportSecrets(tmpDir); err != nil {
			return fmt.Errorf("export secrets: %w", err)
		}
	}

	// Export SSOT (opt-in, private)
	if e.Options.Has(LayerSSOT) {
		if err := e.exportSSOT(tmpDir); err != nil {
			return fmt.Errorf("export ssot: %w", err)
		}
	}

	// Export MCP
	if err := e.exportMCP(tmpDir); err != nil {
		return fmt.Errorf("export mcp: %w", err)
	}

	// Export hooks
	if err := e.exportHooks(tmpDir); err != nil {
		return fmt.Errorf("export hooks: %w", err)
	}

	// Export config template
	if err := e.exportConfigTemplate(tmpDir); err != nil {
		return fmt.Errorf("export config template: %w", err)
	}

	// Write manifest
	if err := e.Manifest.Write(tmpDir); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}

	// Create tarball
	if err := e.createTarGz(tmpDir); err != nil {
		return fmt.Errorf("create tarball: %w", err)
	}

	return nil
}

func (e *Exporter) exportSkills(dst string) error {
	skillsDir := filepath.Join(dst, "skills")
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		return err
	}

	home := e.home()
	var sourceSkillsDir string
	switch e.SourceAgent {
	case "hermes":
		sourceSkillsDir = filepath.Join(home, ".hermes", "profiles", e.SourceProfile, "skills")
	case "claude-code":
		sourceSkillsDir = filepath.Join(home, ".claude", "skills")
	default:
		sourceSkillsDir = filepath.Join(home, ".hermes", "profiles", e.SourceProfile, "skills")
	}

	if _, err := os.Stat(sourceSkillsDir); os.IsNotExist(err) {
		return nil // No skills to export
	}

	entries, err := os.ReadDir(sourceSkillsDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		// Skip hidden/backup directories
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		if entry.IsDir() || (entry.Type()&os.ModeSymlink != 0) {
			src := filepath.Join(sourceSkillsDir, entry.Name())
			// Resolve symlink
			if info, err := os.Lstat(src); err == nil && info.Mode()&os.ModeSymlink != 0 {
				if target, err := os.Readlink(src); err == nil {
					if !filepath.IsAbs(target) {
						target = filepath.Join(filepath.Dir(src), target)
					}
					src = target
				}
			}

			dstPath := filepath.Join(skillsDir, entry.Name())
			if err := copyDir(src, dstPath); err != nil {
				return fmt.Errorf("copy skill %s: %w", entry.Name(), err)
			}
			e.Manifest.Skills = append(e.Manifest.Skills, entry.Name())
		}
	}

	return nil
}

func (e *Exporter) exportMemory(dst string, all bool) error {
	memDir := filepath.Join(dst, "memory")
	if err := os.MkdirAll(memDir, 0755); err != nil {
		return err
	}

	home := e.home()
	var sourceMemDir string
	switch e.SourceAgent {
	case "hermes":
		sourceMemDir = filepath.Join(home, ".hermes", "profiles", e.SourceProfile, "memories")
	case "claude-code":
		sourceMemDir = filepath.Join(home, ".claude", "memory")
	default:
		sourceMemDir = filepath.Join(home, ".hermes", "profiles", e.SourceProfile, "memories")
	}

	if _, err := os.Stat(sourceMemDir); os.IsNotExist(err) {
		return nil
	}

	isMethodology := func(rel string) bool {
		base := filepath.Base(rel)
		for _, m := range MethodologyMemory {
			if base == m {
				return true
			}
		}
		return false
	}

	// Walk source directory and copy files, tracking them in manifest.
	// When all is false, only the structure methodology files are copied.
	return filepath.Walk(sourceMemDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(sourceMemDir, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}

		// Skip lock files
		if strings.HasSuffix(relPath, ".lock") {
			return nil
		}

		if !all && !isMethodology(relPath) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		dstPath := filepath.Join(memDir, relPath)

		// Use sanitizer for files, direct copy for directories
		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		// Check if should strip
		if e.Sanitizer.shouldStrip(relPath, path, info) {
			return nil
		}

		if err := e.Sanitizer.processFile(path, dstPath); err != nil {
			return err
		}

		e.Manifest.MemoryFiles = append(e.Manifest.MemoryFiles, relPath)
		return nil
	})
}

func (e *Exporter) exportSecrets(dst string) error {
	src := filepath.Join(e.home(), "secrets-central")
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return nil
	}
	return copyTree(src, filepath.Join(dst, "secrets"), false)
}

func (e *Exporter) exportSSOT(dst string) error {
	src := filepath.Join(e.home(), "Dev", "00-system")
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return nil
	}
	return copyTree(src, filepath.Join(dst, "ssot"), true)
}

// copyTree copies a directory tree. When skipGit is true, .git is omitted.
func copyTree(src, dst string, skipGit bool) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if skipGit && info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode())
	})
}

func (e *Exporter) exportMCP(dst string) error {
	mcpDir := filepath.Join(dst, "mcp")
	if err := os.MkdirAll(mcpDir, 0755); err != nil {
		return err
	}

	home := e.home()
	var sourceMCPDir string
	switch e.SourceAgent {
	case "hermes":
		sourceMCPDir = filepath.Join(home, ".hermes", "profiles", e.SourceProfile, "mcp")
	case "claude-code":
		sourceMCPDir = filepath.Join(home, ".claude", "mcp")
	default:
		sourceMCPDir = filepath.Join(home, ".hermes", "profiles", e.SourceProfile, "mcp")
	}

	if _, err := os.Stat(sourceMCPDir); os.IsNotExist(err) {
		return nil
	}

	entries, err := os.ReadDir(sourceMCPDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			src := filepath.Join(sourceMCPDir, entry.Name())
			dstPath := filepath.Join(mcpDir, entry.Name())
			// Copy MCP directories directly (they're config files with placeholders)
			if err := copyDir(src, dstPath); err != nil {
				return fmt.Errorf("copy MCP %s: %w", entry.Name(), err)
			}
			e.Manifest.MCPServers = append(e.Manifest.MCPServers, entry.Name())
		}
	}

	return nil
}

func (e *Exporter) exportHooks(dst string) error {
	hooksDir := filepath.Join(dst, "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return err
	}

	home := e.home()
	var sourceHooksDir string
	switch e.SourceAgent {
	case "hermes":
		sourceHooksDir = filepath.Join(home, ".hermes", "profiles", e.SourceProfile, "scripts")
	case "claude-code":
		sourceHooksDir = filepath.Join(home, ".claude", "hooks")
	default:
		sourceHooksDir = filepath.Join(home, ".hermes", "profiles", e.SourceProfile, "scripts")
	}

	// Copy existing hooks that match pattern
	if _, err := os.Stat(sourceHooksDir); err == nil {
		entries, err := os.ReadDir(sourceHooksDir)
		if err == nil {
			for _, entry := range entries {
				if !entry.IsDir() && (strings.HasSuffix(entry.Name(), "-commit") || strings.HasSuffix(entry.Name(), "-push")) {
					src := filepath.Join(sourceHooksDir, entry.Name())
					dstPath := filepath.Join(hooksDir, entry.Name())
					content, err := os.ReadFile(src)
					if err == nil {
						if err := os.WriteFile(dstPath, content, 0755); err == nil {
							e.Manifest.Hooks = append(e.Manifest.Hooks, entry.Name())
						}
					}
				}
			}
		}
	}

	// Always ensure standard hooks exist (generate if not present)
	standardHooks := map[string]string{
		"pre-commit": `#!/usr/bin/env bash
# Pre-commit hook - generated by Rangkai

set -e

# Go formatting
if command -v go >/dev/null 2>&1; then
    go fmt ./...
    go vet ./...
fi

# Linting
if command -v golangci-lint >/dev/null 2>&1; then
    golangci-lint run
fi

# Tests
if command -v go >/dev/null 2>&1; then
    go test ./...
fi
`,
		"pre-push": `#!/usr/bin/env bash
# Pre-push hook - generated by Rangkai

set -e

# Build check
if command -v go >/dev/null 2>&1; then
    go build ./...
    go test ./...
fi
`,
	}

	for name, content := range standardHooks {
		dstPath := filepath.Join(hooksDir, name)
		if _, err := os.Stat(dstPath); os.IsNotExist(err) {
			if err := os.WriteFile(dstPath, []byte(content), 0755); err == nil {
				e.Manifest.Hooks = append(e.Manifest.Hooks, name)
			}
		}
	}

	return nil
}

func (e *Exporter) exportConfigTemplate(dst string) error {
	template := `# Rangkai Config Template
# Generated from profile: ` + e.SourceProfile + ` (` + e.SourceAgent + `)
# Date: ` + time.Now().Format(time.RFC3339) + `

version: "1.0"

model:
  provider: "openrouter:${MODEL}"
  # fallback:
  #   - provider: "ollama:llama3.1:8b"

context_budget:
  max_tokens: 120000
  reserve_for_output: 8000
  warning_threshold: 0.8

tools:
  allow:
    - "read_file"
    - "write_file"
    - "patch"
    - "search_files"
    - "terminal"
    - "web_search"
    - "web_extract"
    - "skill_view"
    - "skill_manage"
    - "memory"
    - "delegate_task"
    - "todo_list"
  confirm_destructive: true

skills:
  core:
    - orchestrator
    - ponytail
    - executive-agent-mode
    - antislop
    - antislop-code
    - antislop-copywriting
    - antislop-human
    - antislop-layoutmobile
    - antislop-ui
    - grilling
    - grill-me
    - grill-with-docs

mcp:
  enabled:
    - github
    - jira
    - postgres
    - cloudflare
    - doris
    - n8n
    - ssh

profile:
  name: "${PROFILE_NAME}"
  home: "${HOME}/.hermes/profiles/${PROFILE_NAME}"

servers:
  staging:
    host: "your-server.example.com"
    user: "your-user"
    alias: "generic"
  data:
    host: "your-server.example.com"
    user: "your-user"
    alias: "generic"
  edge:
    host: "your-server.example.com"
    user: "your-user"
    alias: "generic"
  gateway:
    host: "your-server.example.com"
    user: "your-user"
    alias: "generic"
  worker-1:
    host: "your-server.example.com"
    user: "your-user"
    alias: "generic"
  worker-2:
    host: "your-server.example.com"
    user: "your-user"
    alias: "generic"
`

	return os.WriteFile(filepath.Join(dst, "config.yaml.template"), []byte(template), 0644)
}

func (e *Exporter) createTarGz(srcDir string) error {
	file, err := os.Create(e.OutputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	gz := gzip.NewWriter(file)
	defer gz.Close()

	tw := tar.NewWriter(gz)
	defer tw.Close()

	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = relPath

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		_, err = tw.Write(data)
		return err
	})
}
