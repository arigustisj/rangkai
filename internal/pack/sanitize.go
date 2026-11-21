package pack

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type SanitizeRule struct {
	Strip   []StripRule   `yaml:"strip"`
	Replace []ReplaceRule `yaml:"replace"`
	Keep    []string      `yaml:"keep"`
}

type StripRule struct {
	Pattern string `yaml:"pattern"`
	Path    string `yaml:"path"`
}

type ReplaceRule struct {
	Pattern     string `yaml:"pattern"`
	Replacement string `yaml:"replacement"`
	Path        string `yaml:"path"`
	Key         string `yaml:"key"`
}

type Sanitizer struct {
	rules SanitizeRule
	log   *Manifest
}

func NewSanitizer(rules SanitizeRule, manifest *Manifest) *Sanitizer {
	return &Sanitizer{
		rules: rules,
		log:   manifest,
	}
}

func DefaultSanitizeRules() SanitizeRule {
	return SanitizeRule{
		Strip: []StripRule{
			{Pattern: `ghp_[a-zA-Z0-9]{36}`, Path: ""},
			{Pattern: `sk-[a-zA-Z0-9]{48}`, Path: ""},
			{Pattern: `sk-ant-[a-zA-Z0-9\-_]{95}`, Path: ""},
			{Pattern: `xoxb-[a-zA-Z0-9\-]{50,}`, Path: ""},
			{Path: "memory/private-*"},
			{Path: "memory/session-*"},
			{Path: "**/.env"},
			{Path: "**/state.db*"},
			{Path: "**/state.db-*"},
			{Path: "**/*.ORIGINAL"},
		},
		Replace: []ReplaceRule{
			// Example infrastructure patterns (RFC 5737 TEST-NET addresses + example.com).
			// Replace these with your own infrastructure patterns before exporting a pack.
			{Pattern: `192\.0\.2\.10`, Replacement: "staging-server", Path: ""},
			{Pattern: `198\.51\.100\.20`, Replacement: "data-server", Path: ""},
			{Pattern: `203\.0\.113\.30`, Replacement: "edge-server", Path: ""},
			{Pattern: `ssh\.example\.com`, Replacement: "gateway-server", Path: ""},
			{Path: "**/config.yaml", Key: "model.provider", Replacement: "openrouter:${MODEL}"},
		},
		Keep: []string{
			"memory/conventions.md",
			"memory/server-map.md",
			"memory/git-rules.md",
			"memory/deploy-patterns.md",
			"memory/agent-rules.md",
			"memory/mcp-registry.md",
			"skills/**",
			"mcp/**",
			"hooks/**",
		},
	}
}

func (s *Sanitizer) SanitizeDir(src, dst string) error {
	// Ensure destination root exists
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel(src, path)
		if relPath == "." {
			return nil
		}

		// Check if should keep
		if s.shouldKeep(relPath) {
			return s.copyIfFile(path, filepath.Join(dst, relPath), info)
		}

		// Check if should strip (remove entirely)
		if s.shouldStrip(relPath, path, info) {
			s.log.AddLog("strip", relPath, "", "", "Removed by strip rule")
			return nil // Skip this file/dir
		}

		// Process file content for replacements
		if !info.IsDir() {
			return s.processFile(path, filepath.Join(dst, relPath))
		}

		// Create directory
		return os.MkdirAll(filepath.Join(dst, relPath), info.Mode())
	})
}

func (s *Sanitizer) shouldKeep(relPath string) bool {
	for _, keep := range s.rules.Keep {
		matched, _ := filepath.Match(keep, relPath)
		if matched {
			return true
		}
		// Check prefix for directory patterns
		if strings.HasSuffix(keep, "**") {
			prefix := strings.TrimSuffix(keep, "**")
			if strings.HasPrefix(relPath, prefix) {
				return true
			}
		}
	}
	return false
}

func (s *Sanitizer) shouldStrip(relPath, fullPath string, info os.FileInfo) bool {
	for _, rule := range s.rules.Strip {
		if rule.Path != "" {
			matched, _ := filepath.Match(rule.Path, relPath)
			if matched {
				return true
			}
		}
		if rule.Pattern != "" && !info.IsDir() {
			content, err := os.ReadFile(fullPath)
			if err == nil {
				matched, _ := regexp.MatchString(rule.Pattern, string(content))
				if matched {
					return true
				}
			}
		}
	}
	return false
}

func (s *Sanitizer) processFile(src, dst string) error {
	content, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	original := string(content)
	modified := original

	// Apply replacements
	for _, rule := range s.rules.Replace {
		if rule.Path != "" {
			// For YAML key replacement, we'd need a YAML parser
			// For now, simple string replacement
			if rule.Key != "" && strings.HasSuffix(src, "config.yaml") {
				// Simple line-based replacement for model.provider
				lines := strings.Split(modified, "\n")
				for i, line := range lines {
					if strings.Contains(line, rule.Key) && strings.Contains(line, ":") {
						lines[i] = strings.Replace(line, line, "  provider: \""+rule.Replacement+"\"", 1)
						s.log.AddLog("replace", filepath.Base(src), rule.Pattern, rule.Replacement, "Replaced YAML key: "+rule.Key)
					}
				}
				modified = strings.Join(lines, "\n")
			}
		} else if rule.Pattern != "" {
			re := regexp.MustCompile(rule.Pattern)
			if re.MatchString(modified) {
				modified = re.ReplaceAllString(modified, rule.Replacement)
				s.log.AddLog("replace", filepath.Base(src), rule.Pattern, rule.Replacement, "Pattern replacement")
			}
		}
	}

	if modified != original {
		s.log.AddLog("sanitize", filepath.Base(dst), "", "", "Content sanitized")
	}

	return os.WriteFile(dst, []byte(modified), 0644)
}

func (s *Sanitizer) copyIfFile(src, dst string, info os.FileInfo) error {
	if info.IsDir() {
		return os.MkdirAll(dst, info.Mode())
	}
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	content, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, content, info.Mode())
}
