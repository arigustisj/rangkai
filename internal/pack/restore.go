package pack

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type restoreEntry struct {
	data []byte
	mode int64
}

// RestorePersonal decrypts an archive fully into memory first, so a wrong
// passphrase or corrupt input fails before the destination is touched. It reads
// the embedded manifest, aborts on an existing profile unless force is set, and
// rejects any entry resolving outside destHome (path traversal).
func RestorePersonal(archive io.Reader, passphrase string, destHome string, force bool) error {
	var plain bytes.Buffer
	if err := DecryptStream(&plain, archive, passphrase); err != nil {
		return fmt.Errorf("decrypt failed: %w", err)
	}
	gz, err := gzip.NewReader(&plain)
	if err != nil {
		return fmt.Errorf("not a gzip archive: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	entries := map[string]restoreEntry{}
	var manifest Manifest
	haveManifest := false
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		data, err := io.ReadAll(tr)
		if err != nil {
			return err
		}
		if hdr.Name == "manifest.json" {
			if err := json.Unmarshal(data, &manifest); err != nil {
				return fmt.Errorf("bad manifest: %w", err)
			}
			haveManifest = true
			continue
		}
		entries[hdr.Name] = restoreEntry{data: data, mode: hdr.Mode}
	}
	if !haveManifest {
		return fmt.Errorf("archive missing manifest.json")
	}
	if manifest.SourceProfile == "" {
		return fmt.Errorf("manifest missing profile")
	}
	destClean := filepath.Clean(destHome)
	profileDir := filepath.Join(destClean, ".hermes", "profiles", manifest.SourceProfile)
	if _, err := os.Stat(profileDir); err == nil && !force {
		return fmt.Errorf("profile already exists at %s (use --force to overwrite)", profileDir)
	}
	for name, e := range entries {
		var target string
		switch {
		case strings.HasPrefix(name, "profile/"):
			target = filepath.Join(profileDir, strings.TrimPrefix(name, "profile/"))
		case strings.HasPrefix(name, "secrets/"):
			target = filepath.Join(destClean, "secrets-central", strings.TrimPrefix(name, "secrets/"))
		case strings.HasPrefix(name, "ssot/"):
			target = filepath.Join(destClean, "Dev", "00-system", strings.TrimPrefix(name, "ssot/"))
		default:
			continue
		}
		target = filepath.Clean(target)
		if !strings.HasPrefix(target, destClean+string(os.PathSeparator)) {
			return fmt.Errorf("unsafe path in archive: %s", name)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		mode := os.FileMode(e.mode)
		if mode == 0 {
			mode = 0o600
		}
		if err := os.WriteFile(target, e.data, mode); err != nil {
			return err
		}
	}
	return nil
}
