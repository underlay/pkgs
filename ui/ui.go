package ui

import (
	"bytes"
	"fmt"
	"html/template"
	"io"

	badger "github.com/dgraph-io/badger/v2"

	types "github.com/underlay/pkgs/types"
)

type page struct {
	Pathname string
	P        *types.Package
	Packages map[string]*types.Package
	Messages map[string]types.Message
	Files    map[string]*types.File
}

var pageTemplate = `<!DOCTYPE html>
<html lang="en">
	<head>
		<meta charset="UTF-8" />
		<title>pkgs {{.Pathname}}</title>
		<meta name="viewport" content="width=device-width, initial-scale=1" />
	</head>
	<body>
		<h1>{{.P.Resource}}</h1>
		<section>
			<h2>Packages</h2>
			<ul>
				{{ range $key, $value := .Packages }}
				<li><strong><a href="{{ $key }}">{{ $key }}</a></strong> - {{ $value.URI }}</li>
				{{ end }}
			</ul>
		</section>
		<section>
			<h2>Messages</h2>
			<ul>
				{{ range $key, $value := .Messages }}
				<li><strong>{{ $key }}</strong> - {{ $value.URI }}</li>
				{{ end }}
			</ul>
		</section>
		<section>
			<h2>Files</h2>
			<ul>
				{{ range $key, $value := .Files }}
				<li><strong>{{ $key }}</strong> - {{ $value.URI }}</li>
				{{ end }}
			</ul>
		</section>
	</body>
</html>`

var parsedTemplate = template.Must(template.New("page").Parse(pageTemplate))

// RenderPackage is the only export from the UI package
func RenderPackage(pathname string, p *types.Package, txn *badger.Txn) (io.Reader, error) {
	packagePage := &page{
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

	buf := bytes.NewBuffer(nil)
	return buf, parsedTemplate.Execute(buf, packagePage)
}
