package codexcfg

import (
	"os"
	"path/filepath"
	"strings"
	"switcher/internal/winfs"
	"testing"
)

func TestWindowsCodexInstallRestore(t *testing.T) {
	r, e := winfs.OpenRoot(t.TempDir(), false, true)
	if e != nil {
		t.Fatal(e)
	}
	defer r.Close()
	path := filepath.Join(r.Name(), "config.toml")
	original := []byte("# keep my settings\r\nmodel = \"test-model\"\r\n")
	if e = winfs.Atomic(r, "config.toml", original); e != nil {
		t.Fatal(e)
	}
	if e = Install(path); e != nil {
		t.Fatal(e)
	}
	out, e := os.ReadFile(path)
	if e != nil || !strings.Contains(string(out), `model_provider = "switcher"`) {
		t.Fatalf("install err=%v config=%s", e, out)
	}
	if e = Uninstall(path); e != nil {
		t.Fatal(e)
	}
	out, e = os.ReadFile(path)
	if e != nil || string(out) != string(original) {
		t.Fatalf("restore err=%v config=%q", e, out)
	}
}
func TestWindowsConfigRejectsExternalEdit(t *testing.T) {
	r, e := winfs.OpenRoot(t.TempDir(), false, true)
	if e != nil {
		t.Fatal(e)
	}
	defer r.Close()
	path := filepath.Join(r.Name(), "config.toml")
	if e = winfs.Atomic(r, "config.toml", []byte("model = \"first\"\n")); e != nil {
		t.Fatal(e)
	}
	f, e := openConfigFiles(path, false, true)
	if e != nil {
		t.Fatal(e)
	}
	defer f.close()
	before, _, e := f.read("config.toml", false)
	if e != nil {
		t.Fatal(e)
	}
	if e = winfs.Atomic(r, "config.toml", []byte("model = \"external\"\n")); e != nil {
		t.Fatal(e)
	}
	e = f.replace("config.toml", []byte("overwrite"), expectedFile{digest: digest(before), exists: true})
	if e == nil {
		t.Fatal("external edit overwritten")
	}
}
