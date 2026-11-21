package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose and fix environment issues",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("🏥 Running diagnostics... (stub)")
		return nil
	},
}

var addMCPcmd = &cobra.Command{
	Use:   "add-mcp [name]",
	Short: "Add an MCP server",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("🔌 Adding MCP: %s (stub)\n", args[0])
		return nil
	},
}

var switchModelCmd = &cobra.Command{
	Use:   "switch-model [provider:model]",
	Short: "Switch the active model",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("🔄 Switching model to: %s (stub)\n", args[0])
		return nil
	},
}

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Backup profile, memory, and skills",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("💾 Creating backup... (stub)")
		return nil
	},
}

var restoreCmd = &cobra.Command{
	Use:   "restore [backup-file]",
	Short: "Restore from backup",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("📥 Restoring from: %s (stub)\n", args[0])
		return nil
	},
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update Hermes, skills, and MCP servers",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("⬆️ Updating... (stub)")
		return nil
	},
}

var ejectCmd = &cobra.Command{
	Use:   "eject",
	Short: "Eject config as plain files (no lock-in)",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("📤 Ejecting config... (stub)")
		return nil
	},
}
