package pack

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/arigustisj/rangkai/internal/skill"
)

type Importer struct {
	TargetProfile string
	TargetAgent   string
	SourcePath    string
	SkillRegistry *skill.SkillRegistry
	DryRun        bool
}

func NewImporter(targetProfile, targetAgent, sourcePath string, registry *skill.SkillRegistry) *Importer {
	return &Importer{
		TargetProfile: targetProfile,
		TargetAgent:   targetAgent,
		SourcePath:    sourcePath,
		SkillRegistry: registry,
	}
}

func (i *Importer) Import() error {
	// Extract tarball
	tmpDir, err := i.extractTarGz()
	if err != nil {
		return fmt.Errorf("extract tarball: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Read manifest
	manifest, err := ReadManifest(tmpDir)
	if err != nil {
		return fmt.Errorf("read manifest: %w", err)
	}

	fmt.Printf("📦 Importing pack: %s (from %s, %s)\n", manifest.Version, manifest.SourceProfile, manifest.SourceAgent)
	fmt.Printf("   Created: %s\n", manifest.CreatedAt)
	fmt.Printf("   Skills: %d, Memory: %d, MCP: %d, Hooks: %d\n",
		len(manifest.Skills), len(manifest.MemoryFiles), len(manifest.MCPServers), len(manifest.Hooks))

	// Import skills
	if err := i.importSkills(tmpDir); err != nil {
		return fmt.Errorf("import skills: %w", err)
	}

	// Import memory
	if err := i.importMemory(tmpDir); err != nil {
		return fmt.Errorf("import memory: %w", err)
	}

	// Import MCP
	if err := i.importMCP(tmpDir); err != nil {
		return fmt.Errorf("import mcp: %w", err)
	}

	// Import hooks
	if err := i.importHooks(tmpDir); err != nil {
		return fmt.Errorf("import hooks: %w", err)
	}

	// Import config template
	if err := i.importConfigTemplate(tmpDir); err != nil {
		return fmt.Errorf("import config template: %w", err)
	}

	fmt.Printf("✅ Import complete for profile '%s' (%s)\n", i.TargetProfile, i.TargetAgent)
	return nil
}

func (i *Importer) extractTarGz() (string, error) {
	tmpDir, err := os.MkdirTemp("", "rangkai-import-*")
	if err != nil {
		return "", err
	}

	file, err := os.Open(i.SourcePath)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", err
	}
	defer file.Close()

	gz, err := gzip.NewReader(file)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			os.RemoveAll(tmpDir)
			return "", err
		}

		targetPath := filepath.Join(tmpDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				os.RemoveAll(tmpDir)
				return "", err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				os.RemoveAll(tmpDir)
				return "", err
			}
			outFile, err := os.Create(targetPath)
			if err != nil {
				os.RemoveAll(tmpDir)
				return "", err
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				os.RemoveAll(tmpDir)
				return "", err
			}
			outFile.Close()
		}
	}

	return tmpDir, nil
}

func (i *Importer) importSkills(srcDir string) error {
	skillsSrc := filepath.Join(srcDir, "skills")
	if _, err := os.Stat(skillsSrc); os.IsNotExist(err) {
		return nil
	}

	home, _ := os.UserHomeDir()
	var targetSkillsDir string
	switch i.TargetAgent {
	case "hermes":
		targetSkillsDir = filepath.Join(home, ".hermes", "profiles", i.TargetProfile, "skills")
	case "claude-code":
		targetSkillsDir = filepath.Join(home, ".claude", "skills")
	case "cursor":
		targetSkillsDir = filepath.Join(home, ".cursor", "rules")
	default:
		targetSkillsDir = filepath.Join(home, ".hermes", "profiles", i.TargetProfile, "skills")
	}

	if err := os.MkdirAll(targetSkillsDir, 0755); err != nil {
		return err
	}

	entries, err := os.ReadDir(skillsSrc)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			src := filepath.Join(skillsSrc, entry.Name())
			dst := filepath.Join(targetSkillsDir, entry.Name())
			if err := copyDir(src, dst); err != nil {
				return fmt.Errorf("copy skill %s: %w", entry.Name(), err)
			}
			fmt.Printf("   ✅ Skill: %s\n", entry.Name())
		}
	}

	return nil
}

func (i *Importer) importMemory(srcDir string) error {
	memSrc := filepath.Join(srcDir, "memory")
	if _, err := os.Stat(memSrc); os.IsNotExist(err) {
		return nil
	}

	home, _ := os.UserHomeDir()
	var targetMemDir string
	switch i.TargetAgent {
	case "hermes":
		targetMemDir = filepath.Join(home, ".hermes", "profiles", i.TargetProfile, "memory")
	case "claude-code":
		targetMemDir = filepath.Join(home, ".claude", "memory")
	case "cursor":
		targetMemDir = filepath.Join(home, ".cursor", "memory")
	default:
		targetMemDir = filepath.Join(home, ".hermes", "profiles", i.TargetProfile, "memory")
	}

	if err := os.MkdirAll(targetMemDir, 0755); err != nil {
		return err
	}

	return copyDir(memSrc, targetMemDir)
}

func (i *Importer) importMCP(srcDir string) error {
	mcpSrc := filepath.Join(srcDir, "mcp")
	if _, err := os.Stat(mcpSrc); os.IsNotExist(err) {
		return nil
	}

	home, _ := os.UserHomeDir()
	var targetMCPDir string
	switch i.TargetAgent {
	case "hermes":
		targetMCPDir = filepath.Join(home, ".hermes", "profiles", i.TargetProfile, "mcp")
	case "claude-code":
		targetMCPDir = filepath.Join(home, ".claude", "mcp")
	case "cursor":
		targetMCPDir = filepath.Join(home, ".cursor", "mcp")
	default:
		targetMCPDir = filepath.Join(home, ".hermes", "profiles", i.TargetProfile, "mcp")
	}

	if err := os.MkdirAll(targetMCPDir, 0755); err != nil {
		return err
	}

	return copyDir(mcpSrc, targetMCPDir)
}

func (i *Importer) importHooks(srcDir string) error {
	hooksSrc := filepath.Join(srcDir, "hooks")
	if _, err := os.Stat(hooksSrc); os.IsNotExist(err) {
		return nil
	}

	home, _ := os.UserHomeDir()
	var targetHooksDir string
	switch i.TargetAgent {
	case "hermes":
		targetHooksDir = filepath.Join(home, ".hermes", "profiles", i.TargetProfile, "scripts")
	case "claude-code":
		targetHooksDir = filepath.Join(home, ".claude", "hooks")
	case "cursor":
		targetHooksDir = filepath.Join(home, ".cursor", "hooks")
	default:
		targetHooksDir = filepath.Join(home, ".hermes", "profiles", i.TargetProfile, "scripts")
	}

	if err := os.MkdirAll(targetHooksDir, 0755); err != nil {
		return err
	}

	return copyDir(hooksSrc, targetHooksDir)
}

func (i *Importer) importConfigTemplate(srcDir string) error {
	templateSrc := filepath.Join(srcDir, "config.yaml.template")
	if _, err := os.Stat(templateSrc); os.IsNotExist(err) {
		return nil
	}

	home, _ := os.UserHomeDir()
	var targetConfigPath string
	switch i.TargetAgent {
	case "hermes":
		targetConfigPath = filepath.Join(home, ".hermes", "profiles", i.TargetProfile, "config.yaml")
	case "claude-code":
		// For Claude Code, we'd generate settings.json instead
		targetConfigPath = filepath.Join(home, ".claude", "settings.json")
	case "cursor":
		targetConfigPath = filepath.Join(home, ".cursor", "config.yaml")
	default:
		targetConfigPath = filepath.Join(home, ".hermes", "profiles", i.TargetProfile, "config.yaml")
	}

	content, err := os.ReadFile(templateSrc)
	if err != nil {
		return err
	}

	// Replace placeholders
	contentStr := string(content)
	contentStr = strings.ReplaceAll(contentStr, "${PROFILE_NAME}", i.TargetProfile)
	contentStr = strings.ReplaceAll(contentStr, "${HOME}", home)
	contentStr = strings.ReplaceAll(contentStr, "${MODEL}", "anthropic/claude-3.5-sonnet")

	if i.TargetAgent == "claude-code" {
		// Convert to Claude Code settings.json format
		contentStr = i.convertToClaudeSettings(contentStr)
	}

	return os.WriteFile(targetConfigPath, []byte(contentStr), 0644)
}

func (i *Importer) convertToClaudeSettings(yamlContent string) string {
	// Simple conversion for now - in practice would use a proper YAML parser
	return `{
  "model": "claude-3-5-sonnet-20241022",
  "permissions": {
    "allow": ["read_file", "write_file", "bash", "grep", "task", "web_search"],
    "deny": []
  },
  "hooks": {
    "pre-commit": ["go fmt ./...", "go vet ./...", "go test ./..."]
  }
}`
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(dstPath, data, info.Mode())
	})
}
