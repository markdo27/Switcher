package winfs

import (
	"golang.org/x/sys/windows"
	"os"
	"path/filepath"
	"testing"
)

func privateRoot(t *testing.T) *os.Root {
	t.Helper()
	r, e := OpenRoot(t.TempDir(), false, true)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { r.Close() })
	return r
}
func TestWindowsPrivateAtomicRoundTrip(t *testing.T) {
	r := privateRoot(t)
	for _, v := range []string{"first", "replacement"} {
		if e := Atomic(r, "token.json", []byte(v)); e != nil {
			t.Fatal(e)
		}
		b, e := Read(r, "token.json", 100)
		if e != nil || string(b) != v {
			t.Fatalf("read=%q err=%v", b, e)
		}
	}
	f, e := r.Open("token.json")
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	if e = Check(f); e != nil {
		t.Fatal(e)
	}
}
func TestWindowsRejectsHardlinksAndTraversal(t *testing.T) {
	r := privateRoot(t)
	if e := Atomic(r, "token.json", []byte("synthetic")); e != nil {
		t.Fatal(e)
	}
	if e := r.Link("token.json", "alias.json"); e != nil {
		t.Fatal(e)
	}
	if _, e := Read(r, "token.json", 100); e == nil {
		t.Fatal("hard link accepted")
	}
	for _, name := range []string{"../secret", "C:secret", "sub/file", "."} {
		if _, e := Open(r, name, os.O_RDONLY, 0); e == nil {
			t.Fatalf("unsafe name accepted: %s", name)
		}
	}
}
func TestWindowsRejectsPublicACL(t *testing.T) {
	r := privateRoot(t)
	if e := Atomic(r, "token.json", []byte("synthetic")); e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(r.Name(), "token.json")
	sd, e := windows.SecurityDescriptorFromString("D:P(A;;FA;;;WD)")
	if e != nil {
		t.Fatal(e)
	}
	acl, _, e := sd.DACL()
	if e != nil {
		t.Fatal(e)
	}
	if e = windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, acl, nil); e != nil {
		t.Fatal(e)
	}
	if _, e := Read(r, "token.json", 100); e == nil {
		t.Fatal("world-readable token accepted")
	}
}
func TestWindowsReadSizeLimit(t *testing.T) {
	r := privateRoot(t)
	if e := Atomic(r, "large", []byte("12345")); e != nil {
		t.Fatal(e)
	}
	if _, e := Read(r, "large", 4); e == nil {
		t.Fatal("oversized private file accepted")
	}
}
