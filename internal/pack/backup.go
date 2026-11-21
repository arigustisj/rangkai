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

// PersonalSource locates the private environment to carry. SecretsDir and
// SSOTDir are optional; an empty or absent path is skipped. Profile is required.
type PersonalSource struct {
	Profile    string
	HermesHome string
	SecretsDir string
	SSOTDir    string
}

// personalSkipDirs are runtime/regenerable directories excluded from a carry.
var personalSkipDirs = map[string]bool{
	"cache": true, "sessions": true, "logs": true, "runtime": true,
	"browser-profile": true, "audio_cache": true, "image_cache": true,
	"telemetry": true, ".git": true,
}

func skipPersonal(rel string, isDir bool) bool {
	for _, seg := range strings.Split(rel, "/") {
		if personalSkipDirs[seg] {
			return true
		}
	}
	if !isDir {
		base := filepath.Base(rel)
		if strings.HasSuffix(base, ".lock") || strings.HasSuffix(base, ".log") ||
			strings.HasSuffix(base, "-cache.json") || strings.HasPrefix(base, "state.db") ||
			base == "processes.json" {
			return true
		}
	}
	return false
}

// GatherPersonal writes a gzip tar of the resolved existing private paths into
// dst and returns a Manifest listing every included relative path. The profile
// directory is required; missing optional roots are skipped. An embedded
// manifest.json is written last so restore can verify completeness.
func GatherPersonal(src PersonalSource, dst io.Writer) (Manifest, error) {
	m := NewManifest(src.Profile, "hermes")
	m.Files = []string{}
	profileDir := filepath.Join(src.HermesHome, ".hermes", "profiles", src.Profile)
	if fi, err := os.Stat(profileDir); err != nil || !fi.IsDir() {
		return *m, fmt.Errorf("profile dir not found: %s", profileDir)
	}
	gz := gzip.NewWriter(dst)
	tw := tar.NewWriter(gz)
	type root struct{ prefix, dir string }
	roots := []root{{"profile", profileDir}}
	if src.SecretsDir != "" {
		roots = append(roots, root{"secrets", src.SecretsDir})
	}
	if src.SSOTDir != "" {
		roots = append(roots, root{"ssot", src.SSOTDir})
	}
	for _, rt := range roots {
		if fi, err := os.Stat(rt.dir); err != nil || !fi.IsDir() {
			continue
		}
		err := filepath.Walk(rt.dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(rt.dir, path)
			if rel == "." {
				return nil
			}
			rel = filepath.ToSlash(rel)
			if skipPersonal(rel, info.IsDir()) {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if info.IsDir() || !info.Mode().IsRegular() {
				return nil
			}
			name := rt.prefix + "/" + rel
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if err := tw.WriteHeader(&tar.Header{Name: name, Mode: int64(info.Mode().Perm()), Size: int64(len(data))}); err != nil {
				return err
			}
			if _, err := tw.Write(data); err != nil {
				return err
			}
			m.Files = append(m.Files, name)
			return nil
		})
		if err != nil {
			return *m, err
		}
	}
	mdata, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return *m, err
	}
	if err := tw.WriteHeader(&tar.Header{Name: "manifest.json", Mode: 0o644, Size: int64(len(mdata))}); err != nil {
		return *m, err
	}
	if _, err := tw.Write(mdata); err != nil {
		return *m, err
	}
	if err := tw.Close(); err != nil {
		return *m, err
	}
	if err := gz.Close(); err != nil {
		return *m, err
	}
	return *m, nil
}

// GatherPersonalToEncrypted composes GatherPersonal and EncryptStream so the
// backup command and tests share one path. The tar is buffered then encrypted.
func GatherPersonalToEncrypted(src PersonalSource, dst io.Writer, passphrase string) error {
	var buf bytes.Buffer
	if _, err := GatherPersonal(src, &buf); err != nil {
		return err
	}
	return EncryptStream(dst, &buf, passphrase)
}
