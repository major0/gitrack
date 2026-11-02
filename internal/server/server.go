// Package server implements `git track serve` (ADR-0024): a loopback-only live
// preview of the tracker, rendering the current tree on each request. It is
// read-only and exposes no mutation endpoint.
package server

import (
	"fmt"
	"html/template"
	"net"
	"net/http"
	"sort"
	"strings"

	"github.com/major0/gitrack/internal/core"
)

// listTmpl renders the track index.
var listTmpl = template.Must(template.New("list").Parse(`<!doctype html>
<html><head><meta charset="utf-8"><title>gitrack</title></head>
<body>
<h1>gitrack</h1>
<table>
<tr><th>id</th><th>state</th><th>status</th><th>kind</th><th>title</th></tr>
{{range .}}<tr>
<td><a href="/track/{{.ID}}">{{slice .ID 0 8}}</a></td>
<td>{{.State}}</td><td>{{.Status}}</td><td>{{.Kind}}</td>
<td>{{.Title}}</td>
</tr>
{{end}}</table>
</body></html>`))

// trackTmpl renders a single track.
var trackTmpl = template.Must(template.New("track").Parse(`<!doctype html>
<html><head><meta charset="utf-8"><title>{{.Title}}</title></head>
<body>
<p><a href="/">&larr; all tracks</a></p>
<h1>{{.Title}}</h1>
<dl>
<dt>id</dt><dd>{{.ID}}</dd>
<dt>slug</dt><dd>{{.Slug}}</dd>
<dt>kind</dt><dd>{{.Kind}}</dd>
<dt>state</dt><dd>{{.State}}</dd>
<dt>status</dt><dd>{{.Status}}</dd>
</dl>
{{if .Description}}<h2>Description</h2><pre>{{.Description}}</pre>{{end}}
</body></html>`))

// isLoopbackAddr reports whether addr's host is a loopback interface. A bare
// ":port" or empty host is rejected (it would bind all interfaces).
func isLoopbackAddr(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	if host == "" {
		return false // ":0" binds every interface — refused (ADR-0024)
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// Serve starts the loopback preview server on addr (default "127.0.0.1:0" when
// empty) and blocks. It refuses any non-loopback bind. The chosen URL is passed
// to announce once the listener is up.
func Serve(repo *core.Repo, addr string, announce func(url string)) error {
	if addr == "" {
		addr = "127.0.0.1:0"
	}
	if !isLoopbackAddr(addr) {
		return fmt.Errorf("refusing non-loopback bind %q: serve is loopback-only (ADR-0024)", addr)
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listening on %s: %w", addr, err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", handleIndex(repo))
	mux.HandleFunc("/track/", handleTrack(repo))
	if announce != nil {
		announce("http://" + ln.Addr().String())
	}
	srv := &http.Server{Handler: mux}
	return srv.Serve(ln)
}

// handleIndex renders the track list, read live from the tree on each request.
func handleIndex(repo *core.Repo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		ids, err := repo.ListIDs()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		tracks := make([]*core.Track, 0, len(ids))
		for _, id := range ids {
			t, err := repo.Load(id)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			tracks = append(tracks, t)
		}
		sort.Slice(tracks, func(i, j int) bool { return tracks[i].ID < tracks[j].ID })
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := listTmpl.Execute(w, tracks); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// handleTrack renders a single track page, read live on each request.
func handleTrack(repo *core.Repo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ref := strings.TrimPrefix(r.URL.Path, "/track/")
		if ref == "" {
			http.NotFound(w, r)
			return
		}
		id, err := repo.Resolve(ref)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		t, err := repo.Load(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := trackTmpl.Execute(w, t); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
