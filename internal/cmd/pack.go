package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/arigustisj/rangkai/internal/pack"
	"github.com/arigustisj/rangkai/internal/skill"
	"github.com/spf13/cobra"
)

var (
	packProfile     string
	packAgent       string
	packOutput      string
	packSource      string
	packDryRun      bool
	packRules       string
	packWithMemory  bool
	packWithSecrets bool
	packWithSSOT    bool

	packLintSource string
	packLintInfra  []string
)

// packReferenceProfile is the reference profile used for linking skills.
// Defaults to "reference" so a missing profile is handled gracefully.
var packReferenceProfile = "reference"

var packCmd = &cobra.Command{
	Use:   "pack",
	Short: "Manage portable context engineering packs",
	Long: `Pack commands for exporting, importing, sanitizing, and onboarding
context engineering packages (skills, memory, MCP, hooks).

Examples:
  rangkai pack export --profile personal --agent hermes --output team-pack.tar.gz
  rangkai pack import --profile personal --agent hermes --source team-pack.tar.gz
  rangkai pack sanitize --input ./config --output ./clean --rules strict
  rangkai pack onboard --profile new-hire --source team-pack.tar.gz`,
}

var packExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export skills, memory, MCP, and hooks to a portable pack",
	RunE: func(cmd *cobra.Command, args []string) error {
		if packOutput == "" {
			return fmt.Errorf("output path required (--output)")
		}

		home, _ := os.UserHomeDir()
		globalDir := filepath.Join(home, ".agents", "skills")
		localDir := filepath.Join(home, ".hermes", "skills")
		profileDir := filepath.Join(home, ".hermes", "profiles", packProfile)
		referenceDir := filepath.Join(home, ".hermes", "profiles", packReferenceProfile, "skills")

		registry := skill.NewSkillRegistry(globalDir, localDir, profileDir, referenceDir)

		sanitizer := pack.NewSanitizer(pack.DefaultSanitizeRules(), pack.NewManifest(packProfile, packAgent))
		manifest := pack.NewManifest(packProfile, packAgent)

		opts := pack.DefaultExportOptions()
		if packWithMemory {
			opts.Layers = append(opts.Layers, pack.LayerMemory)
		}
		if packWithSecrets {
			opts.Layers = append(opts.Layers, pack.LayerSecrets)
		}
		if packWithSSOT {
			opts.Layers = append(opts.Layers, pack.LayerSSOT)
		}
		exporter := pack.NewExporterWithOptions(packProfile, packAgent, packOutput, sanitizer, manifest, registry, opts)

		fmt.Printf("📦 Exporting pack from profile '%s' (%s)...\n", packProfile, packAgent)
		if packWithMemory || packWithSecrets || packWithSSOT {
			fmt.Println("⚠️  Private layers included — do NOT share this pack publicly.")
		}

		if err := exporter.Export(); err != nil {
			return fmt.Errorf("export failed: %w", err)
		}

		fmt.Printf("✅ Pack exported to: %s\n", packOutput)
		fmt.Printf("   Skills: %d, Memory files: %d, MCP: %d, Hooks: %d\n",
			len(manifest.Skills), len(manifest.MemoryFiles), len(manifest.MCPServers), len(manifest.Hooks))

		return nil
	},
}

var packImportCmd = &cobra.Command{
	Use:   "import",
	Short: "Import a pack into a target profile and agent",
	RunE: func(cmd *cobra.Command, args []string) error {
		if packSource == "" {
			return fmt.Errorf("source pack required (--source)")
		}

		if _, err := os.Stat(packSource); os.IsNotExist(err) {
			return fmt.Errorf("source pack not found: %s", packSource)
		}

		home, _ := os.UserHomeDir()
		globalDir := filepath.Join(home, ".agents", "skills")
		localDir := filepath.Join(home, ".hermes", "skills")
		profileDir := filepath.Join(home, ".hermes", "profiles", packProfile)
		referenceDir := filepath.Join(home, ".hermes", "profiles", packReferenceProfile, "skills")

		registry := skill.NewSkillRegistry(globalDir, localDir, profileDir, referenceDir)

		importer := pack.NewImporter(packProfile, packAgent, packSource, registry)

		fmt.Printf("📥 Importing pack to profile '%s' (%s)...\n", packProfile, packAgent)

		if err := importer.Import(); err != nil {
			return fmt.Errorf("import failed: %w", err)
		}

		return nil
	},
}

var packSanitizeCmd = &cobra.Command{
	Use:   "sanitize",
	Short: "Sanitize a directory (strip credentials, replace patterns)",
	RunE: func(cmd *cobra.Command, args []string) error {
		input := ""
		output := ""

		// Try to get from flags first
		if cmd.Flags().Changed("input") {
			input, _ = cmd.Flags().GetString("input")
		}
		if cmd.Flags().Changed("output") {
			output, _ = cmd.Flags().GetString("output")
		}

		// Fallback to positional args
		if input == "" && len(args) > 0 {
			input = args[0]
		}
		if output == "" && len(args) > 1 {
			output = args[1]
		}

		if input == "" || output == "" {
			return fmt.Errorf("usage: rangkai pack sanitize --input <src> --output <dst> [--rules <file>]")
		}

		manifest := pack.NewManifest("sanitize", "manual")
		sanitizer := pack.NewSanitizer(pack.DefaultSanitizeRules(), manifest)

		fmt.Printf("🧹 Sanitizing %s -> %s\n", input, output)

		if err := sanitizer.SanitizeDir(input, output); err != nil {
			return fmt.Errorf("sanitize failed: %w", err)
		}

		// Write sanitization manifest
		if err := manifest.Write(output); err != nil {
			return fmt.Errorf("write manifest: %w", err)
		}

		fmt.Printf("✅ Sanitization complete. Log: %d entries\n", len(manifest.SanitizationLog))
		return nil
	},
}

var packLintCmd = &cobra.Command{
	Use:   "lint",
	Short: "Scan a directory or pack for leaks (credentials, private memory, infra)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if packLintSource == "" {
			return fmt.Errorf("--source is required (a directory or a .tar.gz pack)")
		}
		var findings []pack.Finding
		var err error
		if strings.HasSuffix(packLintSource, ".tar.gz") {
			findings, err = pack.LintPack(packLintSource, packLintInfra)
		} else {
			findings, err = pack.LintPath(packLintSource, packLintInfra)
		}
		if err != nil {
			return fmt.Errorf("lint failed: %w", err)
		}
		if len(findings) == 0 {
			fmt.Printf("✅ No leaks found in %s\n", packLintSource)
			return nil
		}
		fmt.Printf("❌ %d leak(s) found in %s:\n", len(findings), packLintSource)
		for _, f := range findings {
			fmt.Printf("  - [%s] %s: %s\n", f.Kind, f.Path, f.Detail)
		}
		return fmt.Errorf("%d leak(s) found", len(findings))
	},
}

var packOnboardCmd = &cobra.Command{
	Use:   "onboard",
	Short: "Interactive onboarding wizard for new team members",
	RunE: func(cmd *cobra.Command, args []string) error {
		if packSource == "" {
			return fmt.Errorf("source pack required (--source)")
		}

		if _, err := os.Stat(packSource); os.IsNotExist(err) {
			return fmt.Errorf("source pack not found: %s", packSource)
		}

		home, _ := os.UserHomeDir()
		globalDir := filepath.Join(home, ".agents", "skills")
		localDir := filepath.Join(home, ".hermes", "skills")
		profileDir := filepath.Join(home, ".hermes", "profiles", packProfile)
		referenceDir := filepath.Join(home, ".hermes", "profiles", packReferenceProfile, "skills")

		registry := skill.NewSkillRegistry(globalDir, localDir, profileDir, referenceDir)

		onboarder := pack.NewOnboarder(packProfile, packAgent, packSource, registry)

		fmt.Printf("🎯 Starting onboarding for '%s'...\n", packProfile)

		return onboarder.Run()
	},
}

func init() {
	packCmd.AddCommand(packExportCmd)
	packCmd.AddCommand(packImportCmd)
	packCmd.AddCommand(packSanitizeCmd)
	packCmd.AddCommand(packLintCmd)
	packCmd.AddCommand(packOnboardCmd)

	packExportCmd.Flags().StringVarP(&packProfile, "profile", "p", "personal", "Source profile name")
	packExportCmd.Flags().StringVar(&packAgent, "agent", "hermes", "Source agent (hermes, claude-code, cursor)")
	packExportCmd.Flags().StringVarP(&packOutput, "output", "o", "", "Output pack file (.tar.gz)")
	packExportCmd.Flags().BoolVar(&packWithMemory, "with-memory", false, "Include personal memory (private layer)")
	packExportCmd.Flags().BoolVar(&packWithSecrets, "with-secrets", false, "Include secrets-central (private layer)")
	packExportCmd.Flags().BoolVar(&packWithSSOT, "with-ssot", false, "Include personal SSOT (private layer)")

	packImportCmd.Flags().StringVarP(&packProfile, "profile", "p", "personal", "Target profile name")
	packImportCmd.Flags().StringVar(&packAgent, "agent", "hermes", "Target agent (hermes, claude-code, cursor)")
	packImportCmd.Flags().StringVarP(&packSource, "source", "s", "", "Source pack file (.tar.gz)")

	packSanitizeCmd.Flags().StringVar(&packRules, "rules", "default", "Sanitization rules (default, strict, minimal)")
	packSanitizeCmd.Flags().StringVar(&packSource, "input", "", "Input directory to sanitize")
	packSanitizeCmd.Flags().StringVar(&packOutput, "output", "", "Output directory for sanitized files")
	packLintCmd.Flags().StringVar(&packLintSource, "source", "", "Directory or .tar.gz pack to scan")
	packLintCmd.Flags().StringArrayVar(&packLintInfra, "infra-pattern", nil, "Extra infra regex to flag (repeatable)")

	packOnboardCmd.Flags().StringVarP(&packProfile, "profile", "p", "new-hire", "Target profile name")
	packOnboardCmd.Flags().StringVar(&packAgent, "agent", "hermes", "Target agent (hermes, claude-code, cursor)")
	packOnboardCmd.Flags().StringVarP(&packSource, "source", "s", "", "Source pack file (.tar.gz)")

	rootCmd.AddCommand(packCmd)
}
