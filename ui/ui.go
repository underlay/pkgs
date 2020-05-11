package ui

import (
	"html/template"
	"strings"
)

var pageTemplate = `<!DOCTYPE html>
<html lang="en">
	<head>
		<meta charset="UTF-8" />
		<title>● /{{ join .Key "/" }}</title>
		<meta name="viewport" content="width=device-width, initial-scale=1" />
	</head>
	<style>
		html {
			background: #fffff8;
			color: #111;
		}
		body {
			width: 640px;
			margin: 1em auto;
		}
		table {
			border-collapse: collapse;
		}
		td {
			vertical-align: top;
			padding: 0;
		}
		.label {
			padding-right: 1em;
		}
	</style>
	<body>
		<header>
			<h1>
				<a href="/">●</a>
				{{ $key := .Key }}
				{{ range $i, $name := $key }}
					/ <a class="label" href="{{ join (slice $key 0 $i) "/"}}/{{ $name }}">{{ $name }}</a>
				{{ end }}
			</h1>
		</header>
		<h1>{{ .Pkg.Title }}</h1>
		<table>
			<tr><td><span class="label">Resource</span></td><td>{{ .Pkg.Resource }}</td></tr>
			<tr><td><span class="label">ID</span></td><td>{{ .Pkg.ID }}</td></tr>
			<tr><td><span class="label">Parent</span></td><td>{{ .Pkg.Parent }}</td></tr>
			<tr><td><span class="label">Description</span></td><td>{{ .Pkg.Description }}</td></tr>
			<tr><td rowspan="2"><span class="label">Contents</span></td><td>{{ .Pkg.Value.ID }}</td></tr>
			<tr><td>{{ .Pkg.Value.Extent }} bytes</td></tr>
			<tr><td><span class="label">Created</span></td><td>{{ .Pkg.Created }}</td></tr>
			<tr><td><span class="label">Modified</span></td><td>{{ .Pkg.Modified }}</td></tr>
		</table>
		<hr />
		{{ $base := .Pkg.Title }}
		{{ if eq (len .Key) 0 }}
			{{ $base = "." }}
		{{ end }}
		<section>
		<h2>Packages</h2>
		{{ if gt (len .Pkg.Members.Packages) 0 }}
		<table>
			{{ range $i, $pkg := .Pkg.Members.Packages }}
			<tr>
				<td><a class="label" href="{{ $base }}/{{ $pkg.Title }}">{{ $pkg.Title }}</a></td>
				<td>{{ $pkg.ID }}</td>
			</tr>
			{{ end }}
		</table>
		{{ else }}
		No packages
		{{ end }}
		</section>
		<section>
		<h2>Assertions</h2>
		{{ if gt (len .Pkg.Members.Assertions) 0 }}
		<table>
			{{ range $i, $assertion := .Pkg.Members.Assertions }}
			<tr>
				<td rowspan="2">
					{{ if ne $assertion.Resource "" }}
					<a class="label" href="{{ $base }}/{{ $assertion.Title }}">{{ $assertion.Title }}</a>
					{{ end }}
				</td>
				<td>{{ $assertion.ID }}</td>
			</tr>
			<tr>
				<td>
					created <span class="created">{{ $assertion.Created }}</span>
					{{- if ne $assertion.Resource "" }}, modified <span class="modified">{{ $assertion.Modified }}</span>{{ end }}
				</td>
			</tr>
			{{ end }}
		</table>
		{{ else }}
		No assertions
		{{ end }}
		</section>
		<section>
		<h2>Files</h2>
		{{ if gt (len .Pkg.Members.Files) 0 }}
		<table>
			{{ range $i, $file := .Pkg.Members.Files }}
			<tr>
				<td rowspan="3">
					{{ if ne $file.Resource "" }}
					<a class="label" href="{{ $base }}/{{ $file.Title }}">{{ $file.Title }}</a>
					{{ end }}
				</td>
				<td>{{ $file.ID }}</td>
			</tr>
			<tr>
				<td>
					created <span class="created">{{ $file.Created }}</span>
					{{- if ne $file.Resource "" }}, modified <span class="modified">{{ $file.Modified }}</span>{{ end }}
				</td>
			</tr>
			<tr>
				<td><span class="format">{{ $file.Format }}</span>, <span class="extent">{{ $file.Extent }}</span> bytes</td>
			</tr>
			{{ end }}
		</table>
		{{ else }}
		No files
		{{ end }}
		</section>
	</body>
</html>`

var funcs = template.FuncMap{"join": strings.Join}

// PageTemplate is the template for HTML package pages
var PageTemplate = template.Must(template.New("page").Funcs(funcs).Parse(pageTemplate))
