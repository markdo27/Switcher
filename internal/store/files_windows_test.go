package store

import (
	"golang.org/x/sys/windows"
	"os"
	"path/filepath"
	"switcher/internal/winfs"
	"testing"
	"time"
)

func TestWindowsAccountsPrivateAndUnsafeExistingRejected(t *testing.T) {
	root, e := winfs.OpenRoot(t.TempDir(), false, true)
	if e != nil {
		t.Fatal(e)
	}
	defer root.Close()
	st := New(root.Name())
	a := Account{ID: "test-account", Provider: "codex", Email: "synthetic@example.invalid", Token: Token{AccessToken: "synthetic-not-a-real-token"}}
	if e = st.Save(a); e != nil {
		t.Fatal(e)
	}
	if _, e = st.Get(a.ID); e != nil {
		t.Fatal(e)
	}
	if _, e = st.List(); e != nil {
		t.Fatal(e)
	}
	sd, e := windows.SecurityDescriptorFromString("D:P(A;;FA;;;WD)")
	if e != nil {
		t.Fatal(e)
	}
	acl, _, e := sd.DACL()
	if e != nil {
		t.Fatal(e)
	}
	if e = windows.SetNamedSecurityInfo(filepath.Join(root.Name(), "accounts", a.ID+".json"), windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, acl, nil); e != nil {
		t.Fatal(e)
	}
	if _, e = st.Get(a.ID); e == nil {
		t.Fatal("public account credential accepted")
	}
	accounts, e := st.List()
	if e != nil {
		t.Fatal(e)
	}
	if len(accounts) != 0 {
		t.Fatal("cached credential returned after its ACL became unsafe")
	}
}

func TestWindowsAbandonedTokenTempsAreCleaned(t *testing.T) {
	st := New(t.TempDir())
	r, e := winfs.OpenRoot(st.accountsDir, false, false)
	if e != nil {
		t.Fatal(e)
	}
	defer r.Close()
	old, e := winfs.Temp(r, []byte("synthetic"))
	if e != nil {
		t.Fatal(e)
	}
	fresh, e := winfs.Temp(r, []byte("synthetic"))
	if e != nil {
		t.Fatal(e)
	}
	oldPath := filepath.Join(r.Name(), old)
	when := time.Now().Add(-2 * time.Hour)
	if e = os.Chtimes(oldPath, when, when); e != nil {
		t.Fatal(e)
	}
	st.CleanTemp()
	if _, e = os.Stat(oldPath); !os.IsNotExist(e) {
		t.Fatal("abandoned credential temp remains")
	}
	if _, e = os.Stat(filepath.Join(r.Name(), fresh)); e != nil {
		t.Fatal("fresh in-progress temp removed")
	}
}
