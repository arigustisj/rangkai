package pack

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Finding describes a leak detected by the linter.
type Finding struct {
	Path   string
	Kind   string
	Detail string
}

// secretPatterns are generic credential shapes. Personal infra is NOT hardcoded;
// it comes from user-supplied extraInfra patterns.
var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`ghp_[A-Za-z0-9]{36,}`),
	regexp.MustCompile(`sk-ant-[A-Za-z0-9_-]{20,}`),
	regexp.MustCompile(`sk-[A-Za-z0-9]{20,}`),
	regexp.MustCompile(`xox[baprs]-[A-Za-z0-9-]{10,}`),
	regexp.MustCompile(`glpat-[A-Za-z0-9_-]{20,}`),
	regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
	regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----`),
}

// forbiddenNames should never appear in a shared pack.
var forbiddenNames = []string{".env", ".env.local", "id_rsa", "id_ed25519", "state.db", "state.db-shm", "state.db-wal"}

// LintPath scans a directory for leaks. extraInfra adds user infra regexes.
func LintPath(root string, extraInfra []string) ([]Finding, error) {
	var findings []Finding
	extra := make([]*regexp.Regexp, 0, len(extraInfra))
	for _, p := range extraInfra {
		re, err := regexp.Compile(p)
		if err != nil {
			return nil, fmt.Errorf("bad infra pattern %q: %w", p, err)
		}
		extra = append(extra, re)
	}
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		lower := strings.ToLower(filepath.Base(path))
		for _, n := range forbiddenNames {
			if lower == n {
				findings = append(findings, Finding{Path: rel, Kind: "forbidden-file", Detail: "file name " + n})
			}
		}
		if info.IsDir() {
			return nil
		}
		if strings.HasPrefix(rel, "memory"+string(filepath.Separator)) {
			base := filepath.Base(rel)
			if strings.HasPrefix(base, "private-") || strings.HasPrefix(base, "session-") {
				findings = append(findings, Finding{Path: rel, Kind: "private-memory", Detail: "private memory file"})
			}
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		content := string(data)
		for _, re := range secretPatterns {
			if re.MatchString(content) {
				findings = append(findings, Finding{Path: rel, Kind: "credential", Detail: re.String()})
			}
		}
		for _, re := range extra {
			if m := re.FindString(content); m != "" {
				findings = append(findings, Finding{Path: rel, Kind: "infra", Detail: "matched " + re.String()})
			}
		}
		return nil
	})
	return findings, err
}

// LintPack extracts a .tar.gz pack to a temp dir and lints it.
func LintPack(packPath string, extraInfra []string) ([]Finding, error) {
	f, err := os.Open(packPath)
	if err != nil {
		return nil, fmt.Errorf("open pack: %w", err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("open gzip: %w", err)
	}
	defer gz.Close()
	tmp, err := os.MkdirTemp("", "rangkai-lint-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read pack: %w", err)
		}
		name := filepath.Clean(h.Name)
		if name == ".." || strings.HasPrefix(name, ".."+string(filepath.Separator)) || filepath.IsAbs(name) {
			return nil, fmt.Errorf("unsafe path in pack: %s", h.Name)
		}
		target := filepath.Join(tmp, name)
		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return nil, err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return nil, err
			}
			out, err := os.Create(target)
			if err != nil {
				return nil, err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return nil, err
			}
			out.Close()
		}
	}
	return LintPath(tmp, extraInfra)
}
