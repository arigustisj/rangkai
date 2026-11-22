package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/arigustisj/rangkai/internal/config"
	"github.com/arigustisj/rangkai/internal/pack"
	"github.com/spf13/cobra"
)

var doctorProfile string

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose and fix environment issues",
	RunE: func(cmd *cobra.Command, args []string) error {
		home, _ := os.UserHomeDir()
		dir := filepath.Join(home, ".hermes", "profiles", doctorProfile)
		fmt.Println("🏥 Running diagnostics...")
		failed := 0
		for _, c := range config.Diagnose(dir) {
			status := "PASS"
			if !c.OK {
				status = "FAIL"
				failed++
			}
			fmt.Printf("   [%s] %s: %s\n", status, c.Name, c.Detail)
		}
		if failed > 0 {
			return fmt.Errorf("%d check(s) failed", failed)
		}
		fmt.Println("✅ All checks passed")
		return nil
	},
}

var switchModelProfile string

var switchModelCmd = &cobra.Command{
	Use:   "switch-model [provider:model]",
	Short: "Switch the active model",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		m := args[0]
		if !strings.Contains(m, ":") {
			return fmt.Errorf("model must be provider:model (got %q)", m)
		}
		home, _ := os.UserHomeDir()
		p := filepath.Join(home, ".hermes", "profiles", switchModelProfile, "config.yaml")
		if _, err := os.Stat(p); err != nil {
			return fmt.Errorf("profile config not found: %s (run rangkai init first)", p)
		}
		if err := config.UpdateModel(p, m); err != nil {
			return fmt.Errorf("switch model: %w", err)
		}
		fmt.Printf("✅ Model switched to %s\n", m)
		return nil
	},
}

var (
	backupProfile    string
	backupSecrets    string
	backupSSOT       string
	backupOutput     string
	backupPassphrase string
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Back up the full personal environment to an encrypted archive",
	RunE: func(cmd *cobra.Command, args []string) error {
		pass := backupPassphrase
		if pass == "" {
			pass = os.Getenv("RANGKAI_PASSPHRASE")
		}
		if pass == "" {
			return fmt.Errorf("passphrase required (--passphrase or RANGKAI_PASSPHRASE)")
		}
		if backupOutput == "" {
			return fmt.Errorf("output path required (--output)")
		}
		home, _ := os.UserHomeDir()
		secrets := backupSecrets
		if secrets == "" {
			secrets = filepath.Join(home, "secrets-central")
		}
		ssot := backupSSOT
		if ssot == "" {
			ssot = filepath.Join(home, "Dev", "00-system")
		}
		src := pack.PersonalSource{Profile: backupProfile, HermesHome: home, SecretsDir: secrets, SSOTDir: ssot}
		f, err := os.Create(backupOutput)
		if err != nil {
			return err
		}
		if err := pack.GatherPersonalToEncrypted(src, f, pass); err != nil {
			_ = f.Close()
			_ = os.Remove(backupOutput)
			return fmt.Errorf("backup failed: %w", err)
		}
		if err := f.Close(); err != nil {
			_ = os.Remove(backupOutput)
			return err
		}
		fmt.Printf("Encrypted backup written to %s\n", backupOutput)
		return nil
	},
}

func init() {
	backupCmd.Flags().StringVarP(&backupProfile, "profile", "p", "personal", "Profile name")
	backupCmd.Flags().StringVar(&backupSecrets, "secrets", "", "Secrets directory (default $HOME/secrets-central)")
	backupCmd.Flags().StringVar(&backupSSOT, "ssot", "", "SSOT directory (default $HOME/Dev/00-system)")
	backupCmd.Flags().StringVarP(&backupOutput, "output", "o", "", "Output archive (.tar.gz.age)")
	backupCmd.Flags().StringVar(&backupPassphrase, "passphrase", "", "Encryption passphrase (or RANGKAI_PASSPHRASE)")
	restoreCmd.Flags().StringVarP(&restoreSource, "source", "s", "", "Source archive (.tar.gz.age)")
	restoreCmd.Flags().StringVar(&restorePassphrase, "passphrase", "", "Decryption passphrase (or RANGKAI_PASSPHRASE)")
	restoreCmd.Flags().BoolVar(&restoreForce, "force", false, "Overwrite an existing profile")
	ejectCmd.Flags().StringVarP(&ejectProfile, "profile", "p", "personal", "Profile name")
	ejectCmd.Flags().StringVar(&ejectSecrets, "secrets", "", "Secrets directory (default $HOME/secrets-central)")
	ejectCmd.Flags().BoolVar(&ejectDryRun, "dry-run", false, "List without deleting")
	ejectCmd.Flags().BoolVar(&ejectYes, "yes", false, "Actually delete (without this, dry-run)")
	switchModelCmd.Flags().StringVarP(&switchModelProfile, "profile", "p", "personal", "Profile name")
	doctorCmd.Flags().StringVarP(&doctorProfile, "profile", "p", "personal", "Profile name")
	updateCmd.Flags().StringVarP(&updateProfile, "profile", "p", "personal", "Profile name")
}

var (
	restoreSource     string
	restorePassphrase string
	restoreForce      bool
)

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore a personal environment from an encrypted archive",
	RunE: func(cmd *cobra.Command, args []string) error {
		pass := restorePassphrase
		if pass == "" {
			pass = os.Getenv("RANGKAI_PASSPHRASE")
		}
		if pass == "" {
			return fmt.Errorf("passphrase required (--passphrase or RANGKAI_PASSPHRASE)")
		}
		if restoreSource == "" {
			return fmt.Errorf("source archive required (--source)")
		}
		f, err := os.Open(restoreSource)
		if err != nil {
			return err
		}
		defer f.Close()
		home, _ := os.UserHomeDir()
		if err := pack.RestorePersonal(f, pass, home, restoreForce); err != nil {
			return fmt.Errorf("restore failed: %w", err)
		}
		fmt.Println("Environment restored.")
		return nil
	},
}

var updateProfile string

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update Hermes, skills, and MCP servers",
	RunE: func(cmd *cobra.Command, args []string) error {
		home, _ := os.UserHomeDir()
		targetDir := filepath.Join(home, ".hermes", "profiles", updateProfile)
		cfgPath := filepath.Join(targetDir, "config.yaml")
		cfg, err := config.ReadProfile(cfgPath)
		if err != nil {
			return fmt.Errorf("cannot read profile config %s: %w", cfgPath, err)
		}
		cfg.HomeDir = home
		cfg.Variant = inferVariant(targetDir)
		fmt.Printf("⬆️ Updating profile '%s' (stack %s, model %s)...\n", cfg.ProfileName, cfg.Stack, cfg.Model)
		if err := generateProfile(cfg, targetDir, ""); err != nil {
			return err
		}
		fmt.Println("✅ Profile refreshed. Sync skills/MCP with: rangkai sync-skills / rangkai sync-mcp")
		return nil
	},
}

// inferVariant derives the variant from the profile's directory layout.
func inferVariant(dir string) string {
	if _, err := os.Stat(filepath.Join(dir, "skills", "shared")); err == nil {
		return "team-member"
	}
	if _, err := os.Stat(filepath.Join(dir, "data", "kanban")); err != nil {
		return "ci-bot"
	}
	return "personal"
}

var (
	ejectProfile string
	ejectSecrets string
	ejectDryRun  bool
	ejectYes     bool
)

var ejectCmd = &cobra.Command{
	Use:   "eject",
	Short: "Wipe the installed environment and secrets from this machine",
	RunE: func(cmd *cobra.Command, args []string) error {
		home, _ := os.UserHomeDir()
		secrets := ejectSecrets
		if secrets == "" {
			secrets = filepath.Join(home, "secrets-central")
		}
		dry := ejectDryRun || !ejectYes
		rep, err := pack.Eject(ejectProfile, home, secrets, dry)
		if err != nil {
			return err
		}
		if dry {
			fmt.Println("DRY RUN (use --yes to actually remove). Would remove:")
		} else {
			fmt.Println("Removed:")
		}
		for _, r := range rep.Removed {
			fmt.Println("  " + r)
		}
		fmt.Println("Residue this tool cannot remove (informational):")
		for _, r := range rep.Residue {
			fmt.Println("  - " + r)
		}
		return nil
	},
}
