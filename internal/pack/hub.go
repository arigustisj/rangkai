package pack

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// runSCP and runSSH are injectable so unit tests run without a real server.
// Defaults fail closed (BatchMode) and trust-on-first-use host keys.
var runSCP = func(args ...string) error {
	full := append([]string{"-o", "BatchMode=yes", "-o", "StrictHostKeyChecking=accept-new"}, args...)
	cmd := exec.Command("scp", full...)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

var runSSH = func(host, script string) error {
	cmd := exec.Command("ssh", "-o", "BatchMode=yes", "-o", "StrictHostKeyChecking=accept-new", host, script)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// SetTransportForTest overrides the SCP/SSH runners and returns a restore func.
func SetTransportForTest(scp func(...string) error, ssh func(string, string) error) func() {
	oldSCP, oldSSH := runSCP, runSSH
	runSCP, runSSH = scp, ssh
	return func() { runSCP, runSSH = oldSCP, oldSSH }
}

func splitHub(hub string) (string, string, error) {
	parts := strings.SplitN(hub, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid hub %q (want host:path)", hub)
	}
	return parts[0], parts[1], nil
}

// PushToHub gathers and encrypts the personal environment, then uploads the
// ciphertext archive to hub/<profile>.tar.gz.age over SCP. The passphrase stays
// local; only ciphertext reaches the hub.
func PushToHub(src PersonalSource, passphrase, hub, profile string) error {
	if passphrase == "" {
		return fmt.Errorf("passphrase required")
	}
	host, dir, err := splitHub(hub)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp("", "rangkai-push-*.tar.gz.age")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if err := GatherPersonalToEncrypted(src, tmp, passphrase); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("gather/encrypt failed: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := runSSH(host, "mkdir -p "+dir); err != nil {
		return fmt.Errorf("hub mkdir failed: %w", err)
	}
	remote := hub + "/" + profile + ".tar.gz.age"
	if err := runSCP(tmp.Name(), remote); err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}
	return nil
}

// PullFromHub downloads hub/<profile>.tar.gz.age over SCP and restores it.
// RestorePersonal decrypts fully before writing, so a wrong passphrase or a
// missing remote archive leaves destHome untouched.
func PullFromHub(hub, profile, passphrase, destHome string, force bool) error {
	if passphrase == "" {
		return fmt.Errorf("passphrase required")
	}
	if _, _, err := splitHub(hub); err != nil {
		return err
	}
	tmp, err := os.CreateTemp("", "rangkai-pull-*.tar.gz.age")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	_ = tmp.Close()
	remote := hub + "/" + profile + ".tar.gz.age"
	if err := runSCP(remote, tmp.Name()); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	f, err := os.Open(tmp.Name())
	if err != nil {
		return err
	}
	defer f.Close()
	if err := RestorePersonal(f, passphrase, destHome, force); err != nil {
		return fmt.Errorf("restore failed: %w", err)
	}
	return nil
}
