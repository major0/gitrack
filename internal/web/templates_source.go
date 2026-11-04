package web

import "html/template"

// crumbsHTML is the shared breadcrumb line: ancestors link, the leaf is bold.
const crumbsHTML = `<p class="crumbs">{{range $i, $c := .Crumbs}}{{if $i}} / {{end}}{{if $c.Href}}<a href="{{$c.Href}}">{{$c.Name}}</a>{{else}}<b>{{$c.Name}}</b>{{end}}{{end}}</p>`

// treeBodyTmpl lists one directory of the snapshot (ADR-0042). Pure HTML.
var treeBodyTmpl = template.Must(template.New("tree").Parse(`
` + crumbsHTML + `
<p class="muted">{{.Ref}} @ {{if .CommitHref}}<a href="{{.CommitHref}}"><code>{{.Short}}</code></a>{{else}}<code>{{.Short}}</code>{{end}}
&middot; <a href="{{.LogHref}}">commit log</a></p>
<table class="tree">
<tr><th>name</th><th>size</th></tr>
{{range .Rows}}<tr><td>{{if .Href}}<a href="{{.Href}}">{{.Name}}</a>{{else}}{{.Name}}{{end}}{{if .Note}} <span class="muted">{{.Note}}</span>{{end}}</td><td class="size">{{.Size}}</td></tr>
{{else}}<tr><td colspan="2" class="muted">empty</td></tr>{{end}}
</table>`))

// blobBodyTmpl shows one file: breadcrumbs, a size/notes line, and numbered
// code. CodeHTML is pre-escaped. Pure HTML.
var blobBodyTmpl = template.Must(template.New("blob").Parse(`
` + crumbsHTML + `
<p class="muted">{{.Size}}{{range .Notes}} &middot; {{.}}{{end}}</p>
{{if .CodeHTML}}<pre class="blob">{{.CodeHTML}}</pre>{{end}}`))

// logBodyTmpl lists the recent commits, newest first. Pure HTML.
var logBodyTmpl = template.Must(template.New("log").Parse(`
<p class="crumbs"><a href="{{.SourceHref}}">source</a> / <b>log</b></p>
<h1>Commits</h1>
<table class="tree">
<tr><th>date</th><th>commit</th><th>author</th></tr>
{{range .Rows}}<tr><td>{{.Date}}</td><td><a href="{{.Href}}">{{.Subject}}</a> <code class="muted">{{.Short}}</code></td><td>{{.Author}}</td></tr>
{{else}}<tr><td colspan="3" class="muted">no commits</td></tr>{{end}}
</table>`))

// commitBodyTmpl shows one commit: message, parents, changed files, and patch.
// SubjectHTML, BodyHTML, and DiffHTML are pre-escaped. Pure HTML.
var commitBodyTmpl = template.Must(template.New("commit").Parse(`
<p class="crumbs"><a href="{{.SourceHref}}">source</a> / <a href="{{.LogHref}}">log</a> / <b><code>{{.Short}}</code></b></p>
<h1>{{.SubjectHTML}}</h1>
<p class="post-meta">{{.Author}}{{if .Date}} &middot; {{.Date}}{{end}} &middot; commit <code>{{.SHA}}</code></p>
{{if .Parents}}<p class="muted">{{if gt (len .Parents) 1}}parents{{else}}parent{{end}}: {{range $i, $p := .Parents}}{{if $i}}, {{end}}{{if $p.Href}}<a href="{{$p.Href}}"><code>{{$p.Short}}</code></a>{{else}}<code>{{$p.Short}}</code>{{end}}{{end}}</p>{{end}}
{{if .BodyHTML}}<pre class="msg">{{.BodyHTML}}</pre>{{end}}
<h2>Changed files ({{len .Changes}})</h2>
{{if .Changes}}<table class="tree">
<tr><th>+</th><th>&minus;</th><th>file</th></tr>
{{range .Changes}}<tr>{{if .Binary}}<td colspan="2" class="muted">binary</td>{{else}}<td class="stat-add">+{{.Added}}</td><td class="stat-del">&minus;{{.Deleted}}</td>{{end}}<td>{{.Path}}</td></tr>
{{end}}</table>{{else}}<p class="muted">no file changes</p>{{end}}
{{if .DiffHTML}}<h2>Diff</h2>
<pre class="diff">{{.DiffHTML}}</pre>{{end}}
{{if .Truncated}}<p class="muted">The patch is longer than the limit and has been cut short.</p>{{end}}`))
