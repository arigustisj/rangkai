package pack

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/arigustisj/rangkai/internal/skill"
)

type Onboarder struct {
	ProfileName   string
	TargetAgent   string
	SourcePack    string
	SkillRegistry *skill.SkillRegistry
}

func NewOnboarder(profileName, targetAgent, sourcePack string, registry *skill.SkillRegistry) *Onboarder {
	return &Onboarder{
		ProfileName:   profileName,
		TargetAgent:   targetAgent,
		SourcePack:    sourcePack,
		SkillRegistry: registry,
	}
}

func (o *Onboarder) Run() error {
	fmt.Println("🎯 Rangkai Onboarding Wizard")
	fmt.Println("═══════════════════════════")
	fmt.Println()

	// 1. Select agent
	agent := o.selectAgent()
	if agent == "" {
		return fmt.Errorf("no agent selected")
	}
	o.TargetAgent = agent

	// 2. Select model access
	modelAccess := o.selectModelAccess()
	if modelAccess == "" {
		return fmt.Errorf("no model access selected")
	}

	// 3. Get API key if needed
	apiKey := ""
	if modelAccess == "openrouter" || modelAccess == "anthropic" || modelAccess == "openai" {
		apiKey = o.promptAPIKey(modelAccess)
		if apiKey == "" {
			return fmt.Errorf("API key required")
		}
	}

	// 4. Confirm profile name
	profile := o.promptProfileName()
	if profile == "" {
		return fmt.Errorf("profile name required")
	}
	o.ProfileName = profile

	// 5. Select stack
	stack := o.promptStack()

	// 6. Run import
	fmt.Println()
	fmt.Printf("📦 Installing pack for %s (%s)...\n", o.TargetAgent, modelAccess)

	importer := NewImporter(o.ProfileName, o.TargetAgent, o.SourcePack, o.SkillRegistry)
	if err := importer.Import(); err != nil {
		return fmt.Errorf("import failed: %w", err)
	}

	// 7. Post-setup: inject API key into .env
	if apiKey != "" {
		if err := o.injectAPIKey(apiKey, modelAccess); err != nil {
			fmt.Printf("⚠️  Warning: failed to inject API key: %v\n", err)
		}
	}

	// 8. Print next steps
	o.printNextSteps(modelAccess, stack)

	return nil
}

func (o *Onboarder) selectAgent() string {
	fmt.Println("1. Which AI agent do you use daily?")
	fmt.Println("   [1] Hermes (recommended - full features)")
	fmt.Println("   [2] Claude Code (Anthropic subscription)")
	fmt.Println("   [3] Cursor (IDE with AI)")
	fmt.Println("   [4] Generic (JSON config)")
	fmt.Print("   > ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	switch input {
	case "1", "hermes":
		return "hermes"
	case "2", "claude-code", "claude":
		return "claude-code"
	case "3", "cursor":
		return "cursor"
	case "4", "generic":
		return "generic"
	default:
		fmt.Println("   Invalid selection, defaulting to Hermes")
		return "hermes"
	}
}

func (o *Onboarder) selectModelAccess() string {
	fmt.Println()
	fmt.Println("2. Which model access do you have?")
	fmt.Println("   [1] OpenRouter (pay-as-you-go, all models: Claude, GPT, Nemotron, Llama)")
	fmt.Println("   [2] Anthropic API direct (Claude Opus/Sonnet)")
	fmt.Println("   [3] OpenAI API direct (GPT-4o)")
	fmt.Println("   [4] Local Ollama (free, needs GPU/RAM)")
	fmt.Print("   > ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	switch input {
	case "1", "openrouter":
		return "openrouter"
	case "2", "anthropic":
		return "anthropic"
	case "3", "openai":
		return "openai"
	case "4", "ollama", "local":
		return "ollama"
	default:
		fmt.Println("   Invalid selection, defaulting to OpenRouter")
		return "openrouter"
	}
}

func (o *Onboarder) promptAPIKey(provider string) string {
	fmt.Printf("\n3. Enter your %s API key (hidden):\n", provider)
	fmt.Print("   > ")

	// Note: In real implementation, use termios to hide input
	reader := bufio.NewReader(os.Stdin)
	key, _ := reader.ReadString('\n')
	return strings.TrimSpace(key)
}

func (o *Onboarder) promptProfileName() string {
	fmt.Printf("\n4. Target profile name [%s]: ", o.ProfileName)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return o.ProfileName
	}
	return input
}

func (o *Onboarder) promptStack() string {
	fmt.Println("\n5. Tech stack:")
	fmt.Println("   [1] Go (clean architecture)")
	fmt.Println("   [2] Node.js/TypeScript")
	fmt.Println("   [3] PHP/Laravel")
	fmt.Println("   [4] Python/FastAPI")
	fmt.Print("   > ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	switch input {
	case "1", "go":
		return "go"
	case "2", "node":
		return "node"
	case "3", "laravel":
		return "laravel"
	case "4", "python":
		return "python"
	default:
		return "go"
	}
}

func (o *Onboarder) injectAPIKey(apiKey, provider string) error {
	home, _ := os.UserHomeDir()
	var envPath string

	switch o.TargetAgent {
	case "hermes":
		envPath = filepath.Join(home, ".hermes", "profiles", o.ProfileName, ".env")
	case "claude-code":
		envPath = filepath.Join(home, ".claude", ".env")
	default:
		envPath = filepath.Join(home, ".hermes", "profiles", o.ProfileName, ".env")
	}

	var keyName string
	switch provider {
	case "openrouter":
		keyName = "OPENROUTER_API_KEY"
	case "anthropic":
		keyName = "ANTHROPIC_API_KEY"
	case "openai":
		keyName = "OPENAI_API_KEY"
	default:
		keyName = "OPENROUTER_API_KEY"
	}

	// Read existing .env or create new
	var content string
	if data, err := os.ReadFile(envPath); err == nil {
		content = string(data)
	}

	// Check if key already exists
	if strings.Contains(content, keyName+"=") {
		return nil // Already set
	}

	// Append
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += fmt.Sprintf("%s=%s\n", keyName, apiKey)

	return os.WriteFile(envPath, []byte(content), 0644)
}

func (o *Onboarder) printNextSteps(modelAccess, stack string) {
	fmt.Println()
	fmt.Println("✅ Onboarding complete!")
	fmt.Println()
	fmt.Println("Next steps:")

	switch o.TargetAgent {
	case "hermes":
		fmt.Printf("  1. cd ~/.hermes/profiles/%s\n", o.ProfileName)
		fmt.Println("  2. cp .env.example .env  # Add remaining credentials")
		fmt.Println("  3. ./install.sh          # Run full installation")
		fmt.Println("  4. ./verify.sh           # Verify everything works")
		fmt.Printf("  5. hermes --profile %s \"grill me on my first task\"\n", o.ProfileName)
	case "claude-code":
		fmt.Printf("  1. cd ~/.claude\n")
		fmt.Printf("  2. claude \"grill me on my first task\"\n")
	case "cursor":
		fmt.Printf("  1. Open Cursor in your project\n")
		fmt.Printf("  2. Cmd/Ctrl + Shift + P → \"Cursor: Open Rules\"\n")
		fmt.Printf("  3. Start coding with AI assistance\n")
	}

	fmt.Println()
	fmt.Println("📚 Your pack includes:")
	fmt.Println("   • Skills: orchestrator, ponytail, grilling, antislop (+4), go, go-http-perf")
	fmt.Println("   • Memory: conventions, server-map, git-rules, deploy-patterns, agent-rules, mcp-registry")
	fmt.Println("   • MCP: github, jira, postgres, cloudflare, doris, n8n, ssh")
	fmt.Println("   • Hooks: pre-commit, pre-push")
	fmt.Println()
	fmt.Println("Welcome to the team! 🚀")
}
