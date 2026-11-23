//go:build windows

package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWindowsListenerChangesRejectedBeforeSaving(t *testing.T) {
	a := &API{}
	for _, body := range []string{`{"bind_lan":true}`, `{"tls":true}`} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPatch, "/api/settings", strings.NewReader(body))
		r.RemoteAddr = "127.0.0.1:12345"
		r.Host = "127.0.0.1"
		a.handleSettingsPatch(w, r)
		if w.Code != http.StatusNotImplemented {
			t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
		}
	}
}
