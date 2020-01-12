package ui

import (
	"fmt"
	"html/template"
	"strings"

	badger "github.com/dgraph-io/badger/v2"

	types "github.com/underlay/pkgs/types"
)

// Page is the local struct we use for representing the data that gets rendered on an HTML page
type Page struct {
	Pathname string
	P        *types.Package
	Packages map[string]*types.Package
	Messages map[string]types.Message
	Files    map[string]*types.File
}

// Path splits the pathname into a slice of path elements
func (p *Page) Path() []string {
	if p.Pathname == "/" {
		return nil
	}
	return strings.Split(p.Pathname[1:], "/")
}

var pageTemplate = `<!DOCTYPE html>
<html lang="en">
	<head>
		<meta charset="UTF-8" />
		<title>● {{ .Pathname }}</title>
		<meta name="viewport" content="width=device-width, initial-scale=1" />
	</head>
	<style>
		html {
			background: #fffff8;
			color: #111;
		}
		body {
			max-width: max-content;
			margin: auto;
		}
		table {
			border-spacing: 0 2px;
		}
		table tr td:nth-child(2) {
			padding-left: 1em;
		}
		pre {
			margin: 0;
		}
	</style>
	<body>
		<header>
			<h1>
				<a href="/">●</a>
				{{ range $index, $element := .Path }} / <a href={{ $element }}>{{ $element }}</a>{{ end }}
			</h1>
		</header>
		<table>
			<tr><td>Resource</td><td><pre>{{ .P.Resource }}</pre></td></tr>
			<tr><td>Version</td><td><pre>{{ .P.URI }}</pre></td></tr>
			<tr><td>Value</td><td><pre>{{ .P.ValueURI }}</pre></td></tr>
			<tr><td>Created</td><td>{{ .P.PrintCreated }}</td></tr>
			<tr><td>Modified</td><td>{{ .P.PrintModified }}</td></tr>
		</table>
		<hr />
		<section>
			<h2>Packages</h2>
			<dl>
				{{ range $key, $value := .Packages }}
				<dt><a href="{{ $key }}">{{ $key }}</a></dt>
				<dd><pre>{{ $value.URI }}</pre></dd>
				{{ else }}
				No packages
				{{ end }}
			</dl>
		</section>
		<section>
			<h2>Messages</h2>
			<dl>
				{{ $p := .Pathname }}
				{{ range $key, $value := .Messages }}
				<dt><a href="{{ $p }}/{{ $key }}">{{ $key }}</a></dt>
				<dd><pre>{{ $value.URI }}</pre></dd>
				{{ else }}
				No messages
				{{ end }}
			</dl>
		</section>
		<section>
			<h2>Files</h2>
			<dl>
				{{ range $key, $value := .Files }}
				<dt><a href="{{ $key }}">{{ $key }}</a></dt>
				<dd>
					<pre>{{ $value.URI }}</pre>
					<pre>{{ $value.Format }}</pre>
					<pre>{{ $value.Extent }} B</pre>
				</dd>
				{{ else }}
				No files
				{{ end }}
			</dl>
		</section>
		<hr />
		<form method="POST">
			<label for="add-file">
				Add file
			</label>
			<input id="add-file" type="file" />
			<input type="submit" />
		</form>
	</body>
</html>`

// PageTemplate is the template for HTML package pages
var PageTemplate = template.Must(template.New("page").Parse(pageTemplate))

// MakePage generates a page given a read-only badger transaction and a pathname
func MakePage(pathname string, p *types.Package, txn *badger.Txn) (*Page, error) {
	packagePage := &Page{
		Pathname: pathname,
		P:        p,
		Packages: make(map[string]*types.Package),
		Messages: make(map[string]types.Message),
		Files:    make(map[string]*types.File),
	}

	for _, member := range p.Member {
		var memberPath string
		if pathname == "/" {
			memberPath = "/" + member
		} else {
			memberPath = fmt.Sprintf("%s/%s", pathname, member)
		}
		resource, _, err := types.GetResource(memberPath, txn)
		if err != nil {
			return nil, err
		}

		switch t := resource.(type) {
		case *types.Package:
			packagePage.Packages[member] = t
		case types.Message:
			packagePage.Messages[member] = t
		case *types.File:
			packagePage.Files[member] = t
		}
	}
	return packagePage, nil
}
