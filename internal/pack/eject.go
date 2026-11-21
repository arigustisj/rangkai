package pack

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Report lists what eject removed (or would remove in dry-run) and the residue
// it cannot control.
type Report struct {
	Removed []string
	Residue []string
}

// Eject removes the resolved profile directory and, if set, the secrets
// directory. It refuses an empty or traversal profile name and any resolved path
// equal to HOME or root. dryRun lists targets without deleting. Residue lines are
// informational: traces outside this tool's control on a borrowed machine.
func Eject(profile string, hermesHome string, secretsDir string, dryRun bool) (Report, error) {
	rep := Report{Removed: []string{}, Residue: []string{}}
	if profile == "" || strings.Contains(profile, "/") || strings.Contains(profile, "..") {
		return rep, fmt.Errorf("invalid profile name: %q", profile)
	}
	homeClean := filepath.Clean(hermesHome)
	profileDir := filepath.Join(homeClean, ".hermes", "profiles", profile)
	if profileDir == homeClean || profileDir == string(os.PathSeparator) {
		return rep, fmt.Errorf("refusing to eject: resolved path equals HOME or root")
	}
	targets := []string{profileDir}
	if secretsDir != "" {
		sd := filepath.Clean(secretsDir)
		if sd == homeClean || sd == string(os.PathSeparator) {
			return rep, fmt.Errorf("refusing to eject secrets: equals HOME or root")
		}
		targets = append(targets, sd)
	}
	for _, t := range targets {
		rep.Removed = append(rep.Removed, t)
		if !dryRun {
			if err := os.RemoveAll(t); err != nil {
				return rep, fmt.Errorf("remove %s: %w", t, err)
			}
		}
	}
	rep.Residue = []string{
		"shell history (~/.bash_history, ~/.zsh_history) may contain commands",
		"~/.ssh/known_hosts may list servers you connected to",
		"swap may hold memory pages (needs root: swapoff -a)",
		"host journal/auth.log and RAM residue are outside this tool's control",
	}
	return rep, nil
}
