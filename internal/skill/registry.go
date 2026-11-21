package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type SkillManifest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Version     string   `json:"version,omitempty"`
	Category    string   `json:"category,omitempty"`
	Source      string   `json:"source"` // "global", "local", "remote"
	Path        string   `json:"path"`
	Tags        []string `json:"tags,omitempty"`
}

type SkillRegistry struct {
	GlobalSkillsDir    string
	LocalSkillsDir     string
	ProfileSkillsDir   string
	ReferenceSkillsDir string // Main profile with all skills
}

func NewSkillRegistry(globalDir, localDir, profileDir, referenceDir string) *SkillRegistry {
	return &SkillRegistry{
		GlobalSkillsDir:    globalDir,
		LocalSkillsDir:     localDir,
		ProfileSkillsDir:   profileDir,
		ReferenceSkillsDir: referenceDir,
	}
}

func (r *SkillRegistry) ListInstalled() ([]SkillManifest, error) {
	var skills []SkillManifest

	// Scan global skills
	if globalSkills, err := r.scanDir(r.GlobalSkillsDir, "global"); err == nil {
		skills = append(skills, globalSkills...)
	}

	// Scan local skills
	if localSkills, err := r.scanDir(r.LocalSkillsDir, "local"); err == nil {
		skills = append(skills, localSkills...)
	}

	// Scan profile skills
	if profileSkills, err := r.scanDir(r.ProfileSkillsDir, "profile"); err == nil {
		skills = append(skills, profileSkills...)
	}

	return skills, nil
}

func (r *SkillRegistry) scanDir(dir, source string) ([]SkillManifest, error) {
	var skills []SkillManifest

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return skills, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		// Skip hidden/backup directories
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		skillPath := filepath.Join(dir, entry.Name())
		skillLink := ""

		// Check if it's a symlink
		info, err := os.Lstat(skillPath)
		if err != nil {
			continue
		}

		isDir := entry.IsDir()
		if !isDir && info.Mode()&os.ModeSymlink != 0 {
			// For symlinks, check if target is a directory
			if target, err := os.Readlink(skillPath); err == nil {
				skillLink = target
				// Resolve relative symlinks
				if !filepath.IsAbs(target) {
					target = filepath.Join(filepath.Dir(skillPath), target)
				}
				if targetInfo, err := os.Stat(target); err == nil {
					isDir = targetInfo.IsDir()
				}
			}
		}

		if !isDir {
			continue
		}

		manifest, err := r.readManifest(skillPath, entry.Name())
		if err != nil {
			// Try to create basic manifest from directory name
			manifest = SkillManifest{
				Name:        entry.Name(),
				Description: "No manifest found",
				Source:      source,
				Path:        skillPath,
			}
			if skillLink != "" {
				manifest.Source = "symlink:" + source
				manifest.Path = skillLink
			}
		} else {
			manifest.Source = source
			manifest.Path = skillPath
			if skillLink != "" {
				manifest.Source = "symlink:" + source
				manifest.Path = skillLink
			}
		}

		skills = append(skills, manifest)
	}

	return skills, nil
}

func (r *SkillRegistry) readManifest(skillPath, skillName string) (SkillManifest, error) {
	manifestPath := filepath.Join(skillPath, "SKILL.md")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		return SkillManifest{}, fmt.Errorf("no SKILL.md found")
	}

	content, err := os.ReadFile(manifestPath)
	if err != nil {
		return SkillManifest{}, err
	}

	// Parse frontmatter
	lines := strings.Split(string(content), "\n")
	if len(lines) < 3 || !strings.HasPrefix(lines[0], "---") {
		return SkillManifest{}, fmt.Errorf("invalid frontmatter")
	}

	var frontmatterLines []string
	for i := 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "---") {
			break
		}
		frontmatterLines = append(frontmatterLines, lines[i])
	}

	frontmatter := strings.Join(frontmatterLines, "\n")
	var manifest SkillManifest
	if err := parseFrontmatter(frontmatter, &manifest); err != nil {
		return SkillManifest{}, err
	}

	manifest.Name = skillName // Use directory name as fallback
	manifest.Path = skillPath

	return manifest, nil
}

func parseFrontmatter(frontmatter string, manifest *SkillManifest) error {
	// Simple YAML-like parsing for key: value pairs
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
		case "version":
			manifest.Version = value
		case "category":
			manifest.Category = value
		case "tags":
			// Parse array: [tag1, tag2]
			value = strings.Trim(value, "[]")
			if value != "" {
				tags := strings.Split(value, ",")
				for i := range tags {
					tags[i] = strings.TrimSpace(tags[i])
				}
				manifest.Tags = tags
			}
		}
	}
	return nil
}

func (r *SkillRegistry) InstallSkill(name, source string, profileDir string) error {
	// source can be: "global:<name>", "local:<name>", "reference:<name>", "git:<url>", "path:<path>"
	parts := strings.SplitN(source, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid source format: %s (use global:<name>, local:<name>, reference:<name>, git:<url>, or path:<path>)", source)
	}

	sourceType := parts[0]
	sourceValue := parts[1]

	// Target directory name (last component of name)
	targetName := filepath.Base(name)
	targetDir := filepath.Join(profileDir, "skills", targetName)

	// Remove existing if any
	os.RemoveAll(targetDir)

	switch sourceType {
	case "global":
		return r.linkGlobalSkill(sourceValue, targetDir)
	case "local":
		return r.linkLocalSkill(sourceValue, targetDir)
	case "reference":
		return r.linkReferenceSkill(sourceValue, targetDir)
	case "git":
		return r.cloneGitSkill(sourceValue, targetDir)
	case "path":
		return r.copyLocalSkill(sourceValue, targetDir)
	default:
		return fmt.Errorf("unknown source type: %s", sourceType)
	}
}

func (r *SkillRegistry) linkGlobalSkill(name, targetDir string) error {
	sourcePath := filepath.Join(r.GlobalSkillsDir, name)
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return fmt.Errorf("global skill not found: %s", name)
	}

	// Ensure parent directory exists
	parentDir := filepath.Dir(targetDir)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	return os.Symlink(sourcePath, targetDir)
}

func (r *SkillRegistry) linkLocalSkill(name, targetDir string) error {
	sourcePath := filepath.Join(r.LocalSkillsDir, name)
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return fmt.Errorf("local skill not found: %s", name)
	}

	// Ensure parent directory exists
	parentDir := filepath.Dir(targetDir)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	return os.Symlink(sourcePath, targetDir)
}

func (r *SkillRegistry) linkReferenceSkill(name, targetDir string) error {
	if r.ReferenceSkillsDir == "" {
		return fmt.Errorf("reference skills directory not configured")
	}
	sourcePath := filepath.Join(r.ReferenceSkillsDir, name)
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return fmt.Errorf("reference skill not found: %s", name)
	}

	// Resolve symlink if it's a symlink
	if info, err := os.Lstat(sourcePath); err == nil && info.Mode()&os.ModeSymlink != 0 {
		if target, err := os.Readlink(sourcePath); err == nil {
			// If relative, make it absolute
			if !filepath.IsAbs(target) {
				target = filepath.Join(filepath.Dir(sourcePath), target)
			}
			sourcePath = target
		}
	}

	// Ensure parent directory exists
	parentDir := filepath.Dir(targetDir)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	return os.Symlink(sourcePath, targetDir)
}

func (r *SkillRegistry) cloneGitSkill(url, targetDir string) error {
	// This would use git clone - implement with exec.Command
	return fmt.Errorf("git clone not implemented yet")
}

func (r *SkillRegistry) copyLocalSkill(sourcePath, targetDir string) error {
	// Copy directory recursively
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

func (r *SkillRegistry) RemoveSkill(name, profileDir string) error {
	targetDir := filepath.Join(profileDir, "skills", name)
	return os.RemoveAll(targetDir)
}

func (r *SkillRegistry) GetSkillPath(name, profileDir string) (string, error) {
	// Check profile skills first
	profilePath := filepath.Join(profileDir, "skills", name)
	if _, err := os.Stat(profilePath); err == nil {
		return profilePath, nil
	}

	// Check local
	localPath := filepath.Join(r.LocalSkillsDir, name)
	if _, err := os.Stat(localPath); err == nil {
		return localPath, nil
	}

	// Check reference
	if r.ReferenceSkillsDir != "" {
		refPath := filepath.Join(r.ReferenceSkillsDir, name)
		if _, err := os.Stat(refPath); err == nil {
			return refPath, nil
		}
	}

	// Check global
	globalPath := filepath.Join(r.GlobalSkillsDir, name)
	if _, err := os.Stat(globalPath); err == nil {
		return globalPath, nil
	}

	return "", fmt.Errorf("skill not found: %s", name)
}

// Core skills that should always be available (matching actual skill names in the reference profile)
var CoreSkills = []string{
	"orchestrator",
	"ponytail-bundle/ponytail",
	"executive-agent-mode",
	"antislop",
	"antislop-code",
	"antislop-copywriting",
	"antislop-human",
	"antislop-layoutmobile",
	"antislop-ui",
	"grilling",
	"grill-me",
	"grill-with-docs",
}

// Stack-specific skills (matching actual skill names)
var StackSkills = map[string][]string{
	"go": {
		"go",
		"go-http-service-performance",
	},
	"node": {
		"setup-ts-deep-modules",
	},
	"laravel": {},
	"python":  {},
}

// Project skills (optional, loaded on demand - only those that exist in reference profile)
var ProjectSkills = []string{
	"authorized-app-security-testing",
	"code-review",
	"github",
	"devops",
	"diagnosing-bugs",
	"systematic-debugging",
	"test-driven-development",
	"git-pr-deploy-workflow",
	"safe-refactor",
	"implement",
	"implement-spec",
	"investigate-first",
	"prototype",
	"wayfinder",
	"writing-skills",
	"grilling",
	"grill-me",
	"grill-with-docs",
}

// ScanDir is a public wrapper for scanning a skill directory
func (r *SkillRegistry) ScanDir(dir, source string) ([]SkillManifest, error) {
	return r.scanDir(dir, source)
}
