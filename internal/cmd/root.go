package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "rangkai",
	Short: "Rangkai - Opinionated AI development environment builder",
	Long: `Rangkai assembles Hermes/Claude Code with curated skills, 
MCP servers, hooks, and model routing in one command.

  rangkai init    # Setup new environment
  rangkai verify  # Health check
  rangkai doctor  # Diagnose issues`,
	Version: "0.1.0",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(verifyCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(addSkillCmd)
	rootCmd.AddCommand(listSkillsCmd)
	rootCmd.AddCommand(removeSkillCmd)
	rootCmd.AddCommand(syncSkillsCmd)
	rootCmd.AddCommand(addMCPCmd)
	rootCmd.AddCommand(listMCPCmd)
	rootCmd.AddCommand(removeMCPCmd)
	rootCmd.AddCommand(syncMCPCmd)
	rootCmd.AddCommand(switchModelCmd)
	rootCmd.AddCommand(backupCmd)
	rootCmd.AddCommand(restoreCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(ejectCmd)
	rootCmd.AddCommand(pushCmd)
	rootCmd.AddCommand(pullCmd)
}
