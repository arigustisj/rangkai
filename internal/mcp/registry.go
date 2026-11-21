package mcp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type MCPServer struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Command     string            `json:"command"`
	Args        []string          `json:"args,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
	Enabled     bool              `json:"enabled"`
	Source      string            `json:"source"` // "global", "local", "reference", "custom"
	Path        string            `json:"path"`
}

type MCPRegistry struct {
	GlobalDir    string
	LocalDir     string
	ReferenceDir string
	ProfileDir   string
}

func NewMCPRegistry(globalDir, localDir, referenceDir, profileDir string) *MCPRegistry {
	return &MCPRegistry{
		GlobalDir:    globalDir,
		LocalDir:     localDir,
		ReferenceDir: referenceDir,
		ProfileDir:   profileDir,
	}
}

func (r *MCPRegistry) ListInstalled() ([]MCPServer, error) {
	var servers []MCPServer

	if globalServers, err := r.scanDir(r.GlobalDir, "global"); err == nil {
		servers = append(servers, globalServers...)
	}
	if localServers, err := r.scanDir(r.LocalDir, "local"); err == nil {
		servers = append(servers, localServers...)
	}
	if referenceServers, err := r.scanDir(r.ReferenceDir, "reference"); err == nil {
		servers = append(servers, referenceServers...)
	}
	if profileServers, err := r.scanDir(r.ProfileDir, "profile"); err == nil {
		servers = append(servers, profileServers...)
	}

	return servers, nil
}

func (r *MCPRegistry) scanDir(dir, source string) ([]MCPServer, error) {
	var servers []MCPServer

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return servers, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		serverPath := filepath.Join(dir, entry.Name())
		serverLink := ""

		info, err := os.Lstat(serverPath)
		if err != nil {
			continue
		}

		isDir := entry.IsDir()
		if !isDir && info.Mode()&os.ModeSymlink != 0 {
			if target, err := os.Readlink(serverPath); err == nil {
				serverLink = target
				if !filepath.IsAbs(target) {
					target = filepath.Join(filepath.Dir(serverPath), target)
				}
				if targetInfo, err := os.Stat(target); err == nil {
					isDir = targetInfo.IsDir()
				}
			}
		}

		if !isDir {
			continue
		}

		manifest, err := r.readManifest(serverPath, entry.Name())
		if err != nil {
			manifest = MCPServer{
				Name:        entry.Name(),
				Description: "No manifest found",
				Source:      source,
				Path:        serverPath,
				Enabled:     true,
			}
			if serverLink != "" {
				manifest.Source = "symlink:" + source
				manifest.Path = serverLink
			}
		} else {
			manifest.Source = source
			manifest.Path = serverPath
			if serverLink != "" {
				manifest.Source = "symlink:" + source
				manifest.Path = serverLink
			}
		}

		servers = append(servers, manifest)
	}

	return servers, nil
}

func (r *MCPRegistry) readManifest(serverPath, serverName string) (MCPServer, error) {
	manifestPath := filepath.Join(serverPath, "MCP.md")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		return MCPServer{}, fmt.Errorf("no MCP.md found")
	}

	content, err := os.ReadFile(manifestPath)
	if err != nil {
		return MCPServer{}, err
	}

	lines := strings.Split(string(content), "\n")
	if len(lines) < 3 || !strings.HasPrefix(lines[0], "---") {
		return MCPServer{}, fmt.Errorf("invalid frontmatter")
	}

	var frontmatterLines []string
	for i := 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "---") {
			break
		}
		frontmatterLines = append(frontmatterLines, lines[i])
	}

	frontmatter := strings.Join(frontmatterLines, "\n")
	var manifest MCPServer
	if err := parseFrontmatter(frontmatter, &manifest); err != nil {
		return MCPServer{}, err
	}

	manifest.Name = serverName
	manifest.Path = serverPath

	return manifest, nil
}

func parseFrontmatter(frontmatter string, manifest *MCPServer) error {
	lines := strings.Split(frontmatter, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, `"`)

		switch key {
		case "name":
			manifest.Name = value
		case "description":
			manifest.Description = value
		case "command":
			manifest.Command = value
		case "args":
			value = strings.Trim(value, "[]")
			if value != "" {
				args := strings.Split(value, ",")
				for i := range args {
					args[i] = strings.TrimSpace(args[i])
					args[i] = strings.Trim(args[i], `"`)
				}
				manifest.Args = args
			}
		case "enabled":
			manifest.Enabled = value == "true"
		}
	}
	return nil
}

func (r *MCPRegistry) InstallServer(name, source string) error {
	parts := strings.SplitN(source, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid source format: %s (use global:<name>, local:<name>, reference:<name>, git:<url>, path:<path>)", source)
	}

	sourceType := parts[0]
	sourceValue := parts[1]

	targetName := filepath.Base(name)
	targetDir := filepath.Join(r.ProfileDir, "mcp", targetName)

	os.RemoveAll(targetDir)

	switch sourceType {
	case "global":
		return r.linkGlobal(sourceValue, targetDir)
	case "local":
		return r.linkLocal(sourceValue, targetDir)
	case "reference":
		return r.linkReference(sourceValue, targetDir)
	case "git":
		return fmt.Errorf("git clone not implemented yet")
	case "path":
		return r.copyLocal(sourceValue, targetDir)
	default:
		return fmt.Errorf("unknown source type: %s", sourceType)
	}
}

func (r *MCPRegistry) linkGlobal(name, targetDir string) error {
	sourcePath := filepath.Join(r.GlobalDir, name)
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return fmt.Errorf("global MCP server not found: %s", name)
	}
	parentDir := filepath.Dir(targetDir)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}
	return os.Symlink(sourcePath, targetDir)
}

func (r *MCPRegistry) linkLocal(name, targetDir string) error {
	sourcePath := filepath.Join(r.LocalDir, name)
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return fmt.Errorf("local MCP server not found: %s", name)
	}
	parentDir := filepath.Dir(targetDir)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}
	return os.Symlink(sourcePath, targetDir)
}

func (r *MCPRegistry) linkReference(name, targetDir string) error {
	if r.ReferenceDir == "" {
		return fmt.Errorf("reference MCP directory not configured")
	}
	sourcePath := filepath.Join(r.ReferenceDir, name)
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return fmt.Errorf("reference MCP server not found: %s", name)
	}

	if info, err := os.Lstat(sourcePath); err == nil && info.Mode()&os.ModeSymlink != 0 {
		if target, err := os.Readlink(sourcePath); err == nil {
			if !filepath.IsAbs(target) {
				target = filepath.Join(filepath.Dir(sourcePath), target)
			}
			sourcePath = target
		}
	}

	parentDir := filepath.Dir(targetDir)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}
	return os.Symlink(sourcePath, targetDir)
}

func (r *MCPRegistry) copyLocal(sourcePath, targetDir string) error {
	return filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, _ := filepath.Rel(sourcePath, path)
		targetPath := filepath.Join(targetDir, relPath)
		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(targetPath, data, info.Mode())
	})
}

func (r *MCPRegistry) RemoveServer(name, profileDir string) error {
	targetDir := filepath.Join(profileDir, "mcp", name)
	return os.RemoveAll(targetDir)
}

func (r *MCPRegistry) GetServerPath(name string) (string, error) {
	if r.ProfileDir != "" {
		profilePath := filepath.Join(r.ProfileDir, "mcp", name)
		if _, err := os.Stat(profilePath); err == nil {
			return profilePath, nil
		}
	}

	if r.ReferenceDir != "" {
		refPath := filepath.Join(r.ReferenceDir, name)
		if _, err := os.Stat(refPath); err == nil {
			return refPath, nil
		}
	}

	if r.LocalDir != "" {
		localPath := filepath.Join(r.LocalDir, name)
		if _, err := os.Stat(localPath); err == nil {
			return localPath, nil
		}
	}

	if r.GlobalDir != "" {
		globalPath := filepath.Join(r.GlobalDir, name)
		if _, err := os.Stat(globalPath); err == nil {
			return globalPath, nil
		}
	}

	return "", fmt.Errorf("MCP server not found: %s", name)
}

// Core MCP servers (matching user's setup)
var CoreMCPServers = []string{
	"github",
	"jira",
	"postgres",
	"cloudflare",
	"doris",
	"n8n",
	"ssh",
}

// Project-specific MCP servers
var ProjectMCPServers = []string{
	"slack",
	"notion",
	"linear",
	"vercel",
	"aws",
	"gcp",
}

// ScanDir is a public wrapper for scanning an MCP directory
func (r *MCPRegistry) ScanDir(dir, source string) ([]MCPServer, error) {
	return r.scanDir(dir, source)
}
