package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/arigustisj/rangkai/internal/config"
	"github.com/spf13/cobra"
)

var verifyProfile string

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Run health checks on the current environment",
	Long: `Verify that the Rangkai environment is properly configured.

Checks:
  - Profile directory and config.yaml
  - Model provider set
  - Skills present
  - MCP section configured
  - Git hooks executable`,
	RunE: func(cmd *cobra.Command, args []string) error {
		home, _ := os.UserHomeDir()
		return runDiagnostics(filepath.Join(home, ".hermes", "profiles", verifyProfile), "🔍 Verifying Rangkai environment...")
	},
}

// runDiagnostics prints each check and returns an error if any failed.
func runDiagnostics(dir, header string) error {
	fmt.Println(header)
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
}

func init() {
	verifyCmd.Flags().StringVarP(&verifyProfile, "profile", "p", "personal", "Profile name")
}
