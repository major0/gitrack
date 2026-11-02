package server

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/major0/gitrack/internal/core"
)

func TestIsLoopbackAddr(t *testing.T) {
	cases := map[string]bool{
		"127.0.0.1:0":    true,
		"127.0.0.1:8080": true,
		"localhost:0":    true,
		"[::1]:0":        true,
		"0.0.0.0:0":      false, // all interfaces
		":0":             false, // empty host = all interfaces
		"192.168.1.5:80": false,
		"example.com:80": false,
		"garbage":        false,
	}
	for addr, want := range cases {
		if got := isLoopbackAddr(addr); got != want {
			t.Errorf("isLoopbackAddr(%q) = %v, want %v", addr, got, want)
		}
	}
}

func TestServeRefusesNonLoopback(t *testing.T) {
	repo := &core.Repo{Root: t.TempDir()}
	err := Serve(repo, "0.0.0.0:0", nil)
	if err == nil || !strings.Contains(err.Error(), "loopback-only") {
		t.Fatalf("Serve(0.0.0.0) err = %v, want loopback refusal", err)
	}
}

func TestServeLiveRead(t *testing.T) {
	dir := t.TempDir()
	repo, err := core.Init(dir)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}

	urlCh := make(chan string, 1)
	go func() {
		_ = Serve(repo, "127.0.0.1:0", func(u string) { urlCh <- u })
	}()

	var base string
	select {
	case base = <-urlCh:
	case <-time.After(3 * time.Second):
		t.Fatal("server did not announce its URL")
	}

	// index is reachable and loopback
	if !strings.Contains(base, "127.0.0.1:") {
		t.Fatalf("server bound non-loopback: %s", base)
	}

	// create a track AFTER startup; it must appear on the next request (live)
	id, err := repo.Create(core.NewTrackOpts{Title: "Live Track Example"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	body := get(t, base+"/")
	if !strings.Contains(body, "Live Track Example") {
		t.Errorf("index did not show the track created after startup")
	}
	// per-track page resolves
	page := get(t, base+"/track/"+id)
	if !strings.Contains(page, "Live Track Example") {
		t.Errorf("track page missing title")
	}
}

func get(t *testing.T, url string) string {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(b)
}
