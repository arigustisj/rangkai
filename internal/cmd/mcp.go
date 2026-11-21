package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/arigustisj/rangkai/internal/mcp"
	"github.com/spf13/cobra"
)

var (
	mcpProfile     string
	mcpSource      string
	mcpShowDetails bool
)

// mcpReferenceProfile is the reference profile used for linking MCP servers.
// Defaults to "reference" so a missing profile is handled gracefully.
var mcpReferenceProfile = "reference"

var addMCPCmd = &cobra.Command{
	Use:   "add-mcp [name]",
	Short: "Add an MCP server to the environment",
	Long: `Add an MCP server to the current profile.

Sources:
  global:<name>     - Link from global MCP servers (~/.agents/mcp)
  local:<name>      - Link from local MCP servers (~/.hermes/mcp)
  reference:<name>  - Link from reference profile (~/.hermes/profiles/reference/mcp)
  git:<url>         - Clone from git repository (not implemented)
  path:<path>       - Copy from local filesystem

Examples:
  rangkai add-mcp github --source reference:github
  rangkai add-mcp my-mcp --source path:~/my-mcp/my-mcp
  rangkai add-mcp postgres --source reference:postgres`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		home, _ := os.UserHomeDir()
		globalDir := filepath.Join(home, ".agents", "mcp")
		localDir := filepath.Join(home, ".hermes", "mcp")
		profileDir := filepath.Join(home, ".hermes", "profiles", mcpProfile)
		referenceDir := filepath.Join(home, ".hermes", "profiles", mcpReferenceProfile, "mcp")

		registry := mcp.NewMCPRegistry(globalDir, localDir, referenceDir, profileDir)

		if mcpSource == "" {
			source := detectMCPSource(name, registry)
			if source == "" {
				return fmt.Errorf("MCP server source not specified and auto-detect failed. Use --source flag")
			}
			mcpSource = source
			fmt.Printf("   Auto-detected source: %s\n", source)
		}

		fmt.Printf("🔌 Installing MCP server: %s from %s\n", name, mcpSource)

		if err := registry.InstallServer(name, mcpSource); err != nil {
			return fmt.Errorf("failed to install MCP server: %w", err)
		}

		fmt.Printf("✅ MCP server '%s' installed to profile '%s'\n", name, mcpProfile)
		return nil
	},
}

var listMCPCmd = &cobra.Command{
	Use:   "list-mcp",
	Short: "List installed and available MCP servers",
	RunE: func(cmd *cobra.Command, args []string) error {
		home, _ := os.UserHomeDir()
		globalDir := filepath.Join(home, ".agents", "mcp")
		localDir := filepath.Join(home, ".hermes", "mcp")
		profileDir := filepath.Join(home, ".hermes", "profiles", mcpProfile)
		referenceDir := filepath.Join(home, ".hermes", "profiles", mcpReferenceProfile, "mcp")

		registry := mcp.NewMCPRegistry(globalDir, localDir, referenceDir, profileDir)

		fmt.Println("╔══════════════════════════════════════════════════════════════╗")
		fmt.Println("║                      MCP Server Registry                      ║")
		fmt.Println("╚══════════════════════════════════════════════════════════════╝")

		// Installed in profile
		fmt.Printf("\n📁 Profile MCP servers (%s):\n", mcpProfile)
		profileServers, _ := registry.ScanDir(filepath.Join(profileDir, "mcp"), "profile")
		if len(profileServers) == 0 {
			fmt.Println("   (none)")
		} else {
			for _, s := range profileServers {
				if mcpShowDetails {
					fmt.Printf("   • %s [%s] - %s\n", s.Name, s.Source, s.Description)
					if len(s.Args) > 0 {
						fmt.Printf("     Command: %s %v\n", s.Command, s.Args)
					} else {
						fmt.Printf("     Command: %s\n", s.Command)
					}
				} else {
					fmt.Printf("   • %s [%s]\n", s.Name, s.Source)
				}
			}
		}

		// Global servers
		fmt.Println("\n🌐 Global MCP servers (~/.agents/mcp):")
		globalServers, _ := registry.ScanDir(globalDir, "global")
		coreCount := 0
		for _, s := range globalServers {
			isCore := false
			for _, core := range mcp.CoreMCPServers {
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
			if mcpShowDetails {
				fmt.Printf("   • %s%s - %s\n", s.Name, marker, s.Description)
			} else {
				fmt.Printf("   • %s%s\n", s.Name, marker)
			}
		}
		fmt.Printf("   (%d core servers)\n", coreCount)

		// Reference servers
		fmt.Println("\n📚 Reference MCP servers (~/.hermes/profiles/" + mcpReferenceProfile + "/mcp):")
		referenceServers, _ := registry.ScanDir(referenceDir, "reference")
		for _, s := range referenceServers {
			if mcpShowDetails {
				fmt.Printf("   • %s - %s\n", s.Name, s.Description)
			} else {
				fmt.Printf("   • %s\n", s.Name)
			}
		}

		// Project servers
		fmt.Println("\n📦 Project MCP servers (available on demand):")
		for _, name := range mcp.ProjectMCPServers {
			found := false
			for _, s := range globalServers {
				if s.Name == name {
					if mcpShowDetails {
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

var removeMCPCmd = &cobra.Command{
	Use:   "remove-mcp [name]",
	Short: "Remove an MCP server from the profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		home, _ := os.UserHomeDir()
		profileDir := filepath.Join(home, ".hermes", "profiles", mcpProfile)
		referenceDir := filepath.Join(home, ".hermes", "profiles", mcpReferenceProfile, "mcp")

		registry := mcp.NewMCPRegistry("", "", referenceDir, profileDir)

		fmt.Printf("🗑️  Removing MCP server: %s\n", name)

		if err := registry.RemoveServer(name, profileDir); err != nil {
			return fmt.Errorf("failed to remove MCP server: %w", err)
		}

		fmt.Printf("✅ MCP server '%s' removed from profile '%s'\n", name, mcpProfile)
		return nil
	},
}

var syncMCPCmd = &cobra.Command{
	Use:   "sync-mcp",
	Short: "Sync core MCP servers to profile",
	Long: `Sync all core MCP servers to the current profile.

This ensures the profile has all required MCP servers for the environment.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		home, _ := os.UserHomeDir()
		globalDir := filepath.Join(home, ".agents", "mcp")
		localDir := filepath.Join(home, ".hermes", "mcp")
		profileDir := filepath.Join(home, ".hermes", "profiles", mcpProfile)
		referenceDir := filepath.Join(home, ".hermes", "profiles", mcpReferenceProfile, "mcp")

		registry := mcp.NewMCPRegistry(globalDir, localDir, referenceDir, profileDir)

		fmt.Printf("🔄 Syncing MCP servers for profile '%s'...\n", mcpProfile)

		synced := 0
		failed := 0
		for _, name := range mcp.CoreMCPServers {
			if err := registry.InstallServer(name, "reference:"+name); err != nil {
				fmt.Printf("   ❌ %s: %v\n", name, err)
				failed++
			} else {
				fmt.Printf("   ✅ %s (core)\n", name)
				synced++
			}
		}

		fmt.Printf("\n✅ Synced: %d, Failed: %d\n", synced, failed)
		return nil
	},
}

func detectMCPSource(name string, registry *mcp.MCPRegistry) string {
	for _, core := range mcp.CoreMCPServers {
		if core == name {
			return "reference:" + name
		}
	}

	for _, proj := range mcp.ProjectMCPServers {
		if proj == name {
			return "reference:" + name
		}
	}

	home, _ := os.UserHomeDir()
	refDir := filepath.Join(home, ".hermes", "profiles", mcpReferenceProfile, "mcp")
	if _, err := os.Stat(filepath.Join(refDir, name)); err == nil {
		return "reference:" + name
	}

	globalDir := filepath.Join(os.Getenv("HOME"), ".agents", "mcp")
	if _, err := os.Stat(filepath.Join(globalDir, name)); err == nil {
		return "global:" + name
	}

	return ""
}

func init() {
	addMCPCmd.Flags().StringVar(&mcpSource, "source", "", "MCP source (global:<name>, local:<name>, reference:<name>, git:<url>, path:<path>)")
	addMCPCmd.Flags().StringVarP(&mcpProfile, "profile", "p", "personal", "Profile name")

	listMCPCmd.Flags().StringVarP(&mcpProfile, "profile", "p", "personal", "Profile name")
	listMCPCmd.Flags().BoolVar(&mcpShowDetails, "details", false, "Show server descriptions")

	removeMCPCmd.Flags().StringVarP(&mcpProfile, "profile", "p", "personal", "Profile name")

	syncMCPCmd.Flags().StringVarP(&mcpProfile, "profile", "p", "personal", "Profile name")
}
