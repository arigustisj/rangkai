package pack

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// runKeygen and sshHomeDir are injectable so tests never touch the real ~/.ssh.
var runKeygen = func(keyPath, comment string) error {
	cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-N", "", "-f", keyPath, "-C", comment)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

var sshHomeDir = func() string {
	h, _ := os.UserHomeDir()
	return h
}

// HubTarget is a parsed --hub value: [user@]host:path plus the local alias.
type HubTarget struct {
	User  string
	Host  string
	Path  string
	Alias string
}

// ParseHubTarget parses "host:path" or "user@host:path".
func ParseHubTarget(s, alias string) (HubTarget, error) {
	hostPart, path, ok := strings.Cut(s, ":")
	if !ok || hostPart == "" || path == "" {
		return HubTarget{}, fmt.Errorf("invalid hub %q (want [user@]host:path)", s)
	}
	t := HubTarget{Path: path, Alias: alias}
	if at := strings.LastIndex(hostPart, "@"); at >= 0 {
		t.User = hostPart[:at]
		t.Host = hostPart[at+1:]
	} else {
		t.Host = hostPart
	}
	if t.Host == "" {
		return HubTarget{}, fmt.Errorf("invalid hub %q (empty host)", s)
	}
	return t, nil
}

// Dest returns the ssh destination (user@host, or host).
func (t HubTarget) Dest() string {
	if t.User != "" {
		return t.User + "@" + t.Host
	}
	return t.Host
}

// SSHPublicKey reads the public key as a single authorized-keys line.
func (t HubTarget) SSHPublicKey(pubPath string) (string, error) {
	b, err := os.ReadFile(pubPath)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

// EnsureSSHKey returns the private and public key paths for a profile,
// generating an ed25519 keypair when missing. It never overwrites an existing
// key. The key has no passphrase so `pull` stays non-interactive.
func EnsureSSHKey(profile string) (keyPath, pubPath string, created bool, err error) {
	dir := filepath.Join(sshHomeDir(), ".ssh")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", "", false, err
	}
	keyPath = filepath.Join(dir, "rangkai_"+profile)
	pubPath = keyPath + ".pub"
	if _, err := os.Stat(keyPath); err == nil {
		return keyPath, pubPath, false, nil
	}
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		return "", "", false, fmt.Errorf("ssh-keygen not found in PATH: %w", err)
	}
	if err := runKeygen(keyPath, "rangkai-"+profile); err != nil {
		return "", "", false, fmt.Errorf("ssh-keygen failed: %w", err)
	}
	if _, err := os.Stat(keyPath); err != nil {
		return "", "", false, fmt.Errorf("ssh-keygen did not create %s", keyPath)
	}
	return keyPath, pubPath, true, nil
}

// WriteHubAlias appends an idempotent "Host <alias>" stanza to ~/.ssh/config.
func WriteHubAlias(t HubTarget, keyPath string) error {
	if t.Alias == "" {
		return fmt.Errorf("hub alias required")
	}
	dir := filepath.Join(sshHomeDir(), ".ssh")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	cfgPath := filepath.Join(dir, "config")
	data, _ := os.ReadFile(cfgPath)
	if hasHostStanza(string(data), t.Alias) {
		return nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "\nHost %s\n", t.Alias)
	fmt.Fprintf(&b, "    HostName %s\n", t.Host)
	if t.User != "" {
		fmt.Fprintf(&b, "    User %s\n", t.User)
	}
	fmt.Fprintf(&b, "    IdentityFile %s\n", keyPath)
	b.WriteString("    StrictHostKeyChecking accept-new\n")
	f, err := os.OpenFile(cfgPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(b.String())
	return err
}

func hasHostStanza(data, alias string) bool {
	for _, ln := range strings.Split(data, "\n") {
		f := strings.Fields(ln)
		if len(f) >= 2 && f[0] == "Host" && f[1] == alias {
			return true
		}
	}
	return false
}

// BootstrapProfile provisions the SSH key and hub alias, returning the alias
// and the public key to authorize on the hub.
func BootstrapProfile(t HubTarget, profile string) (alias, pub string, err error) {
	t.Alias = "rangkai-" + profile
	keyPath, pubPath, _, err := EnsureSSHKey(profile)
	if err != nil {
		return "", "", err
	}
	if err := WriteHubAlias(t, keyPath); err != nil {
		return "", "", err
	}
	pub, err = t.SSHPublicKey(pubPath)
	if err != nil {
		return "", "", err
	}
	return t.Alias, pub, nil
}
