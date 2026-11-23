//go:build windows

package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"testing"
)

func TestWindowsOpenStartsServerWithOneJSONResult(t *testing.T) {
	reserve, e := net.Listen("tcp4", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	addr := reserve.Addr().String()
	port := reserve.Addr().(*net.TCPAddr).Port
	reserve.Close()
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, `{"version":"test","accounts":[]}`) })}
	defer srv.Close()
	started := false
	var out, errOut bytes.Buffer
	code := Run([]string{"open", "--json"}, Options{Port: port, Stdout: &out, Stderr: &errOut, Getenv: func(key string) string {
		if key == "SSH_CONNECTION" {
			return "synthetic"
		}
		return ""
	}, Start: func() error {
		started = true
		ln, e := net.Listen("tcp4", addr)
		if e != nil {
			return e
		}
		go srv.Serve(ln)
		return nil
	}})
	if code != 0 || !started {
		t.Fatalf("code=%d started=%v err=%s", code, started, errOut.String())
	}
	d := json.NewDecoder(&out)
	var result map[string]any
	if e = d.Decode(&result); e != nil {
		t.Fatal(e)
	}
	if e = d.Decode(&result); e != io.EOF {
		t.Fatalf("extra command result: %v", e)
	}
}
