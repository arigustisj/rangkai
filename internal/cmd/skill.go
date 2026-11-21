package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/arigustisj/rangkai/internal/skill"
	"github.com/spf13/cobra"
)

var (
	skillProfile     string
	skillSource      string
	skillStack       string
	skillShowDetails bool
)

// skillReferenceProfile is the reference profile used for linking skills.
// Defaults to "reference" so a missing profile is handled gracefully.
var skillReferenceProfile = "reference"

var addSkillCmd = &cobra.Command{
	Use:   "add-skill [name]",
	Short: "Add a skill to the environment",
	Long: `Add a skill to the current profile.

Sources:
  global:<name>     - Link from global skills (~/.agents/skills)
  local:<name>      - Link from local skills (~/.hermes/skills)
  reference:<name>  - Link from reference profile (~/.hermes/profiles/reference/skills)
  git:<url>         - Clone from git repository (not implemented)
  path:<path>       - Copy from local filesystem

Examples:
  rangkai add-skill orchestrator --source reference:orchestrator
  rangkai add-skill my-skill --source path:~/my-skills/my-skill
  rangkai add-skill go --source reference:go`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		home, _ := os.UserHomeDir()
		globalDir := filepath.Join(home, ".agents", "skills")
		localDir := filepath.Join(home, ".hermes", "skills")
		profileDir := filepath.Join(home, ".hermes", "profiles", skillProfile)
		referenceDir := filepath.Join(home, ".hermes", "profiles", skillReferenceProfile, "skills")

		registry := skill.NewSkillRegistry(globalDir, localDir, profileDir, referenceDir)

		if skillSource == "" {
			// Auto-detect: try core skills first, then stack skills, then project skills
			source := detectSkillSource(name, registry)
			if source == "" {
				return fmt.Errorf("skill source not specified and auto-detect failed. Use --source flag")
			}
			skillSource = source
			fmt.Printf("   Auto-detected source: %s\n", source)
		}

		fmt.Printf("📦 Installing skill: %s from %s\n", name, skillSource)

		if err := registry.InstallSkill(name, skillSource, profileDir); err != nil {
			return fmt.Errorf("failed to install skill: %w", err)
		}

		fmt.Printf("✅ Skill '%s' installed to profile '%s'\n", name, skillProfile)
		return nil
	},
}

var listSkillsCmd = &cobra.Command{
	Use:   "list-skills",
	Short: "List installed and available skills",
	RunE: func(cmd *cobra.Command, args []string) error {
		home, _ := os.UserHomeDir()
		globalDir := filepath.Join(home, ".agents", "skills")
		localDir := filepath.Join(home, ".hermes", "skills")
		profileDir := filepath.Join(home, ".hermes", "profiles", skillProfile)
		referenceDir := filepath.Join(home, ".hermes", "profiles", skillReferenceProfile, "skills")

		registry := skill.NewSkillRegistry(globalDir, localDir, profileDir, referenceDir)

		fmt.Println("╔══════════════════════════════════════════════════════════════╗")
		fmt.Println("║                    Skill Registry                             ║")
		fmt.Println("╚══════════════════════════════════════════════════════════════╝")

		// Installed in profile
		fmt.Printf("\n📁 Profile skills (%s):\n", skillProfile)
		profileSkills, _ := registry.ScanDir(filepath.Join(profileDir, "skills"), "profile")
		if len(profileSkills) == 0 {
			fmt.Println("   (none)")
		} else {
			for _, s := range profileSkills {
				src := s.Source
				if skillShowDetails {
					fmt.Printf("   • %s [%s] - %s\n", s.Name, src, s.Description)
				} else {
					fmt.Printf("   • %s [%s]\n", s.Name, src)
				}
			}
		}

		// Global skills
		fmt.Println("\n🌐 Global skills (~/.agents/skills):")
		globalSkills, _ := registry.ScanDir(globalDir, "global")
		coreCount := 0
		for _, s := range globalSkills {
			isCore := false
			for _, core := range skill.CoreSkills {
				if s.Name == core {
					isCore = true
					coreCount++
					break
				}
			}
			marker := ""
			if isCore {
				marker = " ★"
			}
			if skillShowDetails {
				fmt.Printf("   • %s%s - %s\n", s.Name, marker, s.Description)
			} else {
				fmt.Printf("   • %s%s\n", s.Name, marker)
			}
		}
		fmt.Printf("   (%d core skills)\n", coreCount)

		// Stack skills
		if skillStack != "" {
			fmt.Printf("\n🔧 Stack skills (%s):\n", skillStack)
			if stackSkills, ok := skill.StackSkills[skillStack]; ok {
				for _, name := range stackSkills {
					found := false
					for _, s := range globalSkills {
						if s.Name == name {
							if skillShowDetails {
								fmt.Printf("   • %s - %s\n", s.Name, s.Description)
							} else {
								fmt.Printf("   • %s\n", s.Name)
							}
							found = true
							break
						}
					}
					if !found {
						fmt.Printf("   • %s (not in global)\n", name)
					}
				}
			}
		}

		// Project skills
		fmt.Println("\n📦 Project skills (available on demand):")
		for _, name := range skill.ProjectSkills {
			found := false
			for _, s := range globalSkills {
				if s.Name == name {
					if skillShowDetails {
						fmt.Printf("   • %s - %s\n", s.Name, s.Description)
					} else {
						fmt.Printf("   • %s\n", s.Name)
					}
					found = true
					break
				}
			}
			if !found {
				fmt.Printf("   • %s (not in global)\n", name)
			}
		}

		return nil
	},
}

var removeSkillCmd = &cobra.Command{
	Use:   "remove-skill [name]",
	Short: "Remove a skill from the profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		home, _ := os.UserHomeDir()
		profileDir := filepath.Join(home, ".hermes", "profiles", skillProfile)
		referenceDir := filepath.Join(home, ".hermes", "profiles", skillReferenceProfile, "skills")

		registry := skill.NewSkillRegistry("", "", profileDir, referenceDir)

		fmt.Printf("🗑️  Removing skill: %s\n", name)

		if err := registry.RemoveSkill(name, profileDir); err != nil {
			return fmt.Errorf("failed to remove skill: %w", err)
		}

		fmt.Printf("✅ Skill '%s' removed from profile '%s'\n", name, skillProfile)
		return nil
	},
}

var syncSkillsCmd = &cobra.Command{
	Use:   "sync-skills",
	Short: "Sync core and stack skills to profile",
	Long: `Sync all core skills and stack-specific skills to the current profile.

This ensures the profile has all required skills for the configured stack.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		home, _ := os.UserHomeDir()
		globalDir := filepath.Join(home, ".agents", "skills")
		localDir := filepath.Join(home, ".hermes", "skills")
		profileDir := filepath.Join(home, ".hermes", "profiles", skillProfile)
		referenceDir := filepath.Join(home, ".hermes", "profiles", skillReferenceProfile, "skills")

		registry := skill.NewSkillRegistry(globalDir, localDir, profileDir, referenceDir)

		fmt.Printf("🔄 Syncing skills for profile '%s' (stack: %s)...\n", skillProfile, skillStack)

		// Sync core skills
		synced := 0
		failed := 0
		for _, name := range skill.CoreSkills {
			if err := registry.InstallSkill(name, "reference:"+name, profileDir); err != nil {
				fmt.Printf("   ❌ %s: %v\n", name, err)
				failed++
			} else {
				fmt.Printf("   ✅ %s (core)\n", name)
				synced++
			}
		}

		// Sync stack skills
		if stackSkills, ok := skill.StackSkills[skillStack]; ok {
			for _, name := range stackSkills {
				if err := registry.InstallSkill(name, "reference:"+name, profileDir); err != nil {
					fmt.Printf("   ❌ %s: %v\n", name, err)
					failed++
				} else {
					fmt.Printf("   ✅ %s (stack: %s)\n", name, skillStack)
					synced++
				}
			}
		}

		fmt.Printf("\n✅ Synced: %d, Failed: %d\n", synced, failed)
		return nil
	},
}

func detectSkillSource(name string, registry *skill.SkillRegistry) string {
	// Check core skills
	for _, core := range skill.CoreSkills {
		if core == name {
			return "reference:" + name
		}
	}

	// Check stack skills
	for _, skills := range skill.StackSkills {
		for _, s := range skills {
			if s == name {
				return "reference:" + name
			}
		}
	}

	// Check project skills
	for _, proj := range skill.ProjectSkills {
		if proj == name {
			return "reference:" + name
		}
	}

	// Check if exists in reference profile
	home, _ := os.UserHomeDir()
	refDir := filepath.Join(home, ".hermes", "profiles", skillReferenceProfile, "skills")
	if _, err := os.Stat(filepath.Join(refDir, name)); err == nil {
		return "reference:" + name
	}

	// Check if exists in global
	globalDir := filepath.Join(os.Getenv("HOME"), ".agents", "skills")
	if _, err := os.Stat(filepath.Join(globalDir, name)); err == nil {
		return "global:" + name
	}

	return ""
}

func init() {
	addSkillCmd.Flags().StringVar(&skillSource, "source", "", "Skill source (global:<name>, local:<name>, reference:<name>, git:<url>, path:<path>)")
	addSkillCmd.Flags().StringVarP(&skillProfile, "profile", "p", "personal", "Profile name")

	listSkillsCmd.Flags().StringVarP(&skillProfile, "profile", "p", "personal", "Profile name")
	listSkillsCmd.Flags().StringVar(&skillStack, "stack", "go", "Stack to show stack-specific skills")
	listSkillsCmd.Flags().BoolVar(&skillShowDetails, "details", false, "Show skill descriptions")

	removeSkillCmd.Flags().StringVarP(&skillProfile, "profile", "p", "personal", "Profile name")

	syncSkillsCmd.Flags().StringVarP(&skillProfile, "profile", "p", "personal", "Profile name")
	syncSkillsCmd.Flags().StringVar(&skillStack, "stack", "go", "Stack for stack-specific skills")
}
