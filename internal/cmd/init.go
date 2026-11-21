package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/arigustisj/rangkai/internal/config"
	"github.com/spf13/cobra"
)

var (
	initProfile      string
	initStack        string
	initModel        string
	initServer       string
	initForce        bool
	initSharedSkills string
	initVariant      string // team-member, ci-bot, personal
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new Rangkai environment",
	Long: `Initialize a new Rangkai AI development environment.

Examples:
  rangkai init --profile personal --stack go
  rangkai init --profile team-member --shared-skills https://github.com/you/skills
  rangkai init --variant ci-bot --stack go`,
	RunE: func(cmd *cobra.Command, args []string) error {
		home, _ := os.UserHomeDir()
		targetDir := filepath.Join(home, ".hermes", "profiles", initProfile)

		if !initForce {
			if _, err := os.Stat(targetDir); err == nil {
				return fmt.Errorf("profile %q already exists at %s (use --force to overwrite)", initProfile, targetDir)
			}
		}

		// Apply variant defaults
		variant := initVariant
		if variant == "" {
			variant = initProfile // fallback to profile name
		}

		fmt.Printf("🔧 Initializing Rangkai environment...\n")
		fmt.Printf("   Profile: %s\n", initProfile)
		fmt.Printf("   Variant: %s\n", variant)
		fmt.Printf("   Stack:   %s\n", initStack)
		fmt.Printf("   Model:   %s\n", initModel)
		fmt.Printf("   Target:  %s\n", targetDir)

		// Create directory structure (variant-specific)
		dirs := getVariantDirs(variant)
		for _, d := range dirs {
			if err := os.MkdirAll(filepath.Join(targetDir, d), 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", d, err)
			}
		}

		// Build config
		cfg := config.ProfileConfig{
			ProfileName: initProfile,
			Stack:       initStack,
			Model:       initModel,
			Server:      initServer,
			HomeDir:     home,
			Variant:     variant,
		}

		// Generate config.yaml
		configYAML, err := config.GenerateConfig(cfg)
		if err != nil {
			return fmt.Errorf("failed to generate config: %w", err)
		}
		configPath := filepath.Join(targetDir, "config.yaml")
		if err := os.WriteFile(configPath, []byte(configYAML), 0644); err != nil {
			return fmt.Errorf("failed to write config.yaml: %w", err)
		}
		fmt.Printf("   ✓ Generated config.yaml\n")

		// Generate install.sh
		installScript, err := config.GenerateInstallScript(cfg)
		if err != nil {
			return fmt.Errorf("failed to generate install.sh: %w", err)
		}
		installPath := filepath.Join(targetDir, "install.sh")
		if err := os.WriteFile(installPath, []byte(installScript), 0755); err != nil {
			return fmt.Errorf("failed to write install.sh: %w", err)
		}
		fmt.Printf("   ✓ Generated install.sh\n")

		// Generate verify.sh
		verifyScript, err := config.GenerateVerifyScript(cfg)
		if err != nil {
			return fmt.Errorf("failed to generate verify.sh: %w", err)
		}
		verifyPath := filepath.Join(targetDir, "verify.sh")
		if err := os.WriteFile(verifyPath, []byte(verifyScript), 0755); err != nil {
			return fmt.Errorf("failed to write verify.sh: %w", err)
		}
		fmt.Printf("   ✓ Generated verify.sh\n")

		// Generate .env.example (variant-specific)
		envExample := generateEnvExample(variant)
		envPath := filepath.Join(targetDir, ".env.example")
		if err := os.WriteFile(envPath, []byte(envExample), 0644); err != nil {
			return fmt.Errorf("failed to write .env.example: %w", err)
		}
		fmt.Printf("   ✓ Generated .env.example\n")

		// Generate git hooks (variant-specific)
		if err := generateHooks(targetDir, variant); err != nil {
			return fmt.Errorf("failed to generate hooks: %w", err)
		}
		fmt.Printf("   ✓ Generated git hooks\n")

		// If team-member with shared skills, generate skill sync script
		if variant == "team-member" && initSharedSkills != "" {
			if err := generateSkillSyncScript(targetDir, initSharedSkills); err != nil {
				return fmt.Errorf("failed to generate skill sync script: %w", err)
			}
			fmt.Printf("   ✓ Generated skill-sync.sh\n")
		}

		fmt.Println("\n✅ Rangkai environment initialized!")
		fmt.Printf("   Location: %s\n", targetDir)
		fmt.Println("\nNext steps:")
		fmt.Printf("  1. cd %s\n", targetDir)
		fmt.Println("  2. cp .env.example .env  # Add your API tokens")
		fmt.Println("  3. ./install.sh          # Run full installation")
		fmt.Println("  4. ./verify.sh           # Verify everything works")
		if variant == "team-member" && initSharedSkills != "" {
			fmt.Println("  5. ./skill-sync.sh       # Sync shared skills from team repo")
		}
		fmt.Printf("\nOr start coding directly:\n")
		fmt.Printf("  hermes --profile %s \"create a GET /users endpoint\"\n", initProfile)

		return nil
	},
}

func generateEnvExample(variant string) string {
	base := `# Rangkai Environment Variables
# Copy this file to .env and fill in your credentials

# GitHub
GITHUB_TOKEN=ghp_xxx

# Jira
JIRA_URL=https://your-domain.atlassian.net
JIRA_EMAIL=your@email.com
JIRA_TOKEN=xxx

# Postgres (shared instance)
POSTGRES_DSN=postgres://user:***@host:5432/db?sslmode=disable

# Cloudflare
CF_API_TOKEN=xxx
CF_ACCOUNT_ID=xxx

# Doris
DORIS_URL=http://host:8030
DORIS_USER=root
DORIS_PASSWORD=xxx

# N8N
N8N_URL=https://n8n.yourdomain.com
N8N_API_KEY=xxx

# SSH (uses ~/.ssh/config)
# No extra config needed
`

	switch variant {
	case "ci-bot":
		return base + `
# CI Bot - minimal credentials
# Only GitHub token needed for CI operations
`
	case "team-member":
		return base + `
# Team Member - shared skills repo
# RANGKAI_SKILLS_REPO=https://github.com/your-org/rangkai-skills
`
	default: // personal
		return base
	}
}

func generateHooks(targetDir, variant string) error {
	scriptsDir := filepath.Join(targetDir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		return err
	}

	var preCommit, prePush string

	switch variant {
	case "ci-bot":
		preCommit = `#!/usr/bin/env bash
# Pre-commit hook - CI Bot (minimal)
set -e
# Go formatting
if command -v go >/dev/null 2>&1; then
    go fmt ./...
    go vet ./...
fi
# Tests only
if command -v go >/dev/null 2>&1; then
    go test ./...
fi
`
		prePush = `#!/usr/bin/env bash
# Pre-push hook - CI Bot
set -e
if command -v go >/dev/null 2>&1; then
    go build ./...
    go test ./...
fi
`
	case "team-member":
		preCommit = `#!/usr/bin/env bash
# Pre-commit hook - Team Member
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
`
		prePush = `#!/usr/bin/env bash
# Pre-push hook - Team Member
set -e
if command -v go >/dev/null 2>&1; then
    go build ./...
    go test ./...
fi
`
	default: // personal
		preCommit = `#!/usr/bin/env bash
# Pre-commit hook - Personal (full)
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
`
		prePush = `#!/usr/bin/env bash
# Pre-push hook - Personal
set -e
if command -v go >/dev/null 2>&1; then
    go build ./...
    go test ./...
fi
`
	}

	if err := os.WriteFile(filepath.Join(scriptsDir, "pre-commit"), []byte(preCommit), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(scriptsDir, "pre-push"), []byte(prePush), 0755); err != nil {
		return err
	}

	return nil
}

func generateSkillSyncScript(targetDir, sharedSkillsURL string) error {
	script := fmt.Sprintf(`#!/usr/bin/env bash
# Skill Sync Script - Team Member
# Syncs shared skills from team repository

set -e

SHARED_SKILLS_REPO="%s"
SKILLS_DIR="$(dirname "$0")/../skills/shared"

echo "Syncing shared skills from %s..."

if [ -d "$SKILLS_DIR" ]; then
    cd "$SKILLS_DIR"
    git pull origin main
else
    git clone "$SHARED_SKILLS_REPO" "$SKILLS_DIR"
fi

echo "Shared skills synced to $SKILLS_DIR"
`, sharedSkillsURL, sharedSkillsURL)

	scriptPath := filepath.Join(targetDir, "skill-sync.sh")
	return os.WriteFile(scriptPath, []byte(script), 0755)
}

func getVariantDirs(variant string) []string {
	base := []string{
		"data", "cache", "memory", "skills", "plugins", "cron", "scripts",
		"data/kanban", "data/sessions", "data/swarm",
	}

	switch variant {
	case "ci-bot":
		// Minimal: no kanban, sessions, swarm
		return []string{
			"data", "cache", "memory", "skills", "plugins", "cron", "scripts",
		}
	case "team-member":
		// Full + shared skills mount point
		dirs := append(base, "skills/shared")
		return dirs
	default: // personal
		return base
	}
}

func init() {
	initCmd.Flags().StringVarP(&initProfile, "profile", "p", "personal", "Profile name (personal, team-member, ci-bot)")
	initCmd.Flags().StringVarP(&initStack, "stack", "s", "go", "Tech stack (go, node, laravel, python)")
	initCmd.Flags().StringVarP(&initModel, "model", "m", "openrouter:nemotron", "Model provider:model (openrouter:..., ollama:..., vllm:...)")
	initCmd.Flags().StringVar(&initServer, "server", "", "Target server alias (e.g. staging, data, edge, gateway)")
	initCmd.Flags().StringVar(&initSharedSkills, "shared-skills", "", "Git repo URL for shared skills")
	initCmd.Flags().StringVar(&initVariant, "variant", "", "Variant: personal, team-member, ci-bot (defaults to profile name)")
	initCmd.Flags().BoolVar(&initForce, "force", false, "Overwrite existing profile")
}
