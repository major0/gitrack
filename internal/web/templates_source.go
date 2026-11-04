package web

import "html/template"

// crumbsHTML is the shared breadcrumb trail: ancestors link and the leaf is the
// current page. The separators between entries come from the theme.
const crumbsHTML = `<nav class="crumbs" aria-label="Breadcrumb"><ol>{{range .Crumbs}}<li{{if not .Href}} aria-current="page"{{end}}>{{if .Href}}<a href="{{.Href}}">{{.Name}}</a>{{else}}{{.Name}}{{end}}</li>{{end}}</ol></nav>`

// treeBodyTmpl lists one directory of the snapshot (ADR-0042).
var treeBodyTmpl = template.Must(template.New("tree").Parse(`
` + crumbsHTML + `
<h1>{{.Heading}}</h1>
<ul class="meta">
<li class="ref">{{.Ref}} @ {{if .CommitHref}}<a href="{{.CommitHref}}"><code>{{.Short}}</code></a>{{else}}<code>{{.Short}}</code>{{end}}</li>
<li><a href="{{.LogHref}}">commit log</a></li>
</ul>
<table class="tree">
<thead><tr><th>name</th><th>size</th></tr></thead>
<tbody>
{{range .Rows}}<tr><td>{{if .Href}}<a href="{{.Href}}">{{.Name}}</a>{{else}}{{.Name}}{{end}}{{if .Note}} <span class="note">{{.Note}}</span>{{end}}</td><td class="size">{{.Size}}</td></tr>
{{else}}<tr><td colspan="2" class="empty">empty</td></tr>{{end}}
</tbody>
</table>`))

// blobBodyTmpl shows one file: breadcrumbs, a size/notes list, and numbered
// code. CodeHTML is pre-escaped.
var blobBodyTmpl = template.Must(template.New("blob").Parse(`
` + crumbsHTML + `
<h1>{{.Heading}}</h1>
<ul class="meta">
<li class="size">{{.Size}}</li>
{{range .Notes}}<li class="note">{{.}}</li>
{{end}}</ul>
{{if .CodeHTML}}<pre class="blob{{if .Highlighted}} chroma{{end}}">{{.CodeHTML}}</pre>{{end}}`))

// logBodyTmpl lists the recent commits, newest first.
var logBodyTmpl = template.Must(template.New("log").Funcs(tmplFuncs).Parse(`
<nav class="crumbs" aria-label="Breadcrumb"><ol><li><a href="{{.SourceHref}}">source</a></li><li aria-current="page">log</li></ol></nav>
<h1>Commits</h1>
<table class="tree">
<thead><tr><th>date</th><th>commit</th><th>author</th></tr></thead>
<tbody>
{{range .Rows}}<tr><td>{{if .Date}}<time datetime="{{isotime .Date}}">{{.Date}}</time>{{end}}</td><td><a href="{{.Href}}">{{.Subject}}</a> <code class="sha">{{.Short}}</code></td><td>{{.Author}}</td></tr>
{{else}}<tr><td colspan="3" class="empty">no commits</td></tr>{{end}}
</tbody>
</table>`))

// commitBodyTmpl shows one commit: message, parents, changed files, and patch.
// SubjectHTML, BodyHTML, and DiffHTML are pre-escaped.
var commitBodyTmpl = template.Must(template.New("commit").Funcs(tmplFuncs).Parse(`
<nav class="crumbs" aria-label="Breadcrumb"><ol><li><a href="{{.SourceHref}}">source</a></li><li><a href="{{.LogHref}}">log</a></li><li aria-current="page"><code>{{.Short}}</code></li></ol></nav>
<article class="commit">
<h1>{{.SubjectHTML}}</h1>
<ul class="meta">
<li class="author">{{.Author}}</li>
{{if .Date}}<li><time datetime="{{isotime .Date}}">{{.Date}}</time></li>
{{end}}<li class="commit-id">commit <code>{{.SHA}}</code></li>
</ul>
{{if .Parents}}<p class="parents">{{if gt (len .Parents) 1}}parents{{else}}parent{{end}}: {{range $i, $p := .Parents}}{{if $i}}, {{end}}{{if $p.Href}}<a href="{{$p.Href}}"><code>{{$p.Short}}</code></a>{{else}}<code>{{$p.Short}}</code>{{end}}{{end}}</p>{{end}}
{{if .BodyHTML}}<pre class="msg">{{.BodyHTML}}</pre>{{end}}
<h2>Changed files <span class="count">{{len .Changes}}</span></h2>
{{if .Changes}}<table class="tree changes">
<thead><tr><th>added</th><th>removed</th><th>file</th></tr></thead>
<tbody>
{{range .Changes}}<tr>{{if .Binary}}<td colspan="2" class="note">binary</td>{{else}}<td class="stat-add">{{.Added}}</td><td class="stat-del">{{.Deleted}}</td>{{end}}<td>{{.Path}}</td></tr>
{{end}}</tbody>
</table>{{else}}<p class="empty">no file changes</p>{{end}}
{{if .DiffHTML}}<h2>Diff</h2>
<pre class="diff">{{.DiffHTML}}</pre>{{end}}
{{if .Truncated}}<p class="note">The patch is longer than the limit and has been cut short.</p>{{end}}
</article>`))
