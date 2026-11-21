package pack

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSplitHub(t *testing.T) {
	h, d, err := splitHub("srv:rangkai-hub")
	if err != nil || h != "srv" || d != "rangkai-hub" {
		t.Fatalf("%q %q %v", h, d, err)
	}
	if _, _, err := splitHub("no-colon"); err == nil {
		t.Fatal("expected error")
	}
	if _, _, err := splitHub(":empty"); err == nil {
		t.Fatal("expected error")
	}
}

func fakeTransport(t *testing.T, fakeHub string) {
	t.Helper()
	oldSCP, oldSSH := runSCP, runSSH
	t.Cleanup(func() { runSCP, runSSH = oldSCP, oldSSH })
	resolve := func(p string) string {
		if i := strings.Index(p, ":"); i >= 0 {
			return filepath.Join(fakeHub, p[i+1:])
		}
		return p
	}
	runSSH = func(host, script string) error {
		if strings.HasPrefix(script, "mkdir -p ") {
			return os.MkdirAll(filepath.Join(fakeHub, strings.TrimPrefix(script, "mkdir -p ")), 0o755)
		}
		return nil
	}
	runSCP = func(args ...string) error {
		data, err := os.ReadFile(resolve(args[len(args)-2]))
		if err != nil {
			return err
		}
		return os.WriteFile(resolve(args[len(args)-1]), data, 0o600)
	}
}

func TestPushPullHubFakeTransport(t *testing.T) {
	fakeHub := t.TempDir()
	fakeTransport(t, fakeHub)
	src := t.TempDir()
	os.MkdirAll(filepath.Join(src, ".hermes", "profiles", "personal", "memory"), 0o755)
	os.WriteFile(filepath.Join(src, ".hermes", "profiles", "personal", "memory", "MEMORY.md"), []byte("hub-carry"), 0o600)
	if err := PushToHub(PersonalSource{Profile: "personal", HermesHome: src}, "pw", "fakehost:rangkai-hub", "personal"); err != nil {
		t.Fatal(err)
	}
	arc, err := os.ReadFile(filepath.Join(fakeHub, "rangkai-hub", "personal.tar.gz.age"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(arc, []byte("age-encryption.org/")) {
		t.Fatal("hub archive not ciphertext")
	}
	dest := t.TempDir()
	if err := PullFromHub("fakehost:rangkai-hub", "personal", "pw", dest, false); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(filepath.Join(dest, ".hermes", "profiles", "personal", "memory", "MEMORY.md"))
	if string(got) != "hub-carry" {
		t.Fatalf("got %q", got)
	}
}

func TestPullWrongPassphraseNoTouch(t *testing.T) {
	fakeHub := t.TempDir()
	fakeTransport(t, fakeHub)
	src := t.TempDir()
	os.MkdirAll(filepath.Join(src, ".hermes", "profiles", "personal"), 0o755)
	if err := PushToHub(PersonalSource{Profile: "personal", HermesHome: src}, "right", "fakehost:rangkai-hub", "personal"); err != nil {
		t.Fatal(err)
	}
	dest := t.TempDir()
	if err := PullFromHub("fakehost:rangkai-hub", "personal", "wrong", dest, false); err == nil {
		t.Fatal("expected error")
	}
	if e, _ := os.ReadDir(dest); len(e) != 0 {
		t.Fatal("dest touched")
	}
}

func TestPushRejectsBadHub(t *testing.T) {
	src := t.TempDir()
	os.MkdirAll(filepath.Join(src, ".hermes", "profiles", "personal"), 0o755)
	if err := PushToHub(PersonalSource{Profile: "personal", HermesHome: src}, "pw", "no-colon", "personal"); err == nil {
		t.Fatal("expected bad hub error")
	}
}
