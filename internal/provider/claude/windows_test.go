//go:build windows

package claude

import (
	"context"
	"switcher/internal/store"
	"testing"
)

func TestWindowsClaudeProxyOnly(t *testing.T) {
	p := New()
	if e := p.ConfigureNative(store.New(t.TempDir()), t.TempDir()); e != nil {
		t.Fatal(e)
	}
	if p.NativeEnabled() {
		t.Fatal("Windows attached an unsupported native manager")
	}
	if changed, e := p.SyncNative(context.Background(), &store.Account{}); changed || e != nil {
		t.Fatalf("proxy sync changed=%v err=%v", changed, e)
	}
	if _, e := p.ImportFromKeychain(context.Background()); e == nil {
		t.Fatal("native credential import was not refused")
	}
}
