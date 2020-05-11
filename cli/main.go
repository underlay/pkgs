package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"

	jsonrpc2 "github.com/sourcegraph/jsonrpc2"
	cli "github.com/urfave/cli/v2"

	rdf "github.com/underlay/go-rdfjs"
	types "github.com/underlay/pkgs/types"
)

var resource = "http://example.com"
var base = "http://localhost:8086"

func main() {
	app := &cli.App{
		Name:                 "pkgs",
		Usage:                "interact with resources on pkgs",
		EnableBashCompletion: true,
		Commands: []*cli.Command{
			{
				Name:      "ls",
				Usage:     "list the members of a package",
				UsageText: "ls [resource]",
				Action: func(c *cli.Context) error {
					arg := c.Args().First()
					key := types.ParsePath(arg)
					url := types.GetURI(base, key)
					req, err := http.NewRequest("GET", url, nil)
					if err != nil {
						return err
					}

					req.Header.Add("Accept", "application/ld+json")
					res, err := http.DefaultClient.Do(req)
					if err != nil {
						return err
					}

					if res.StatusCode != 200 {
						return errors.New(res.Status)
					}

					self, t := types.ParseLinks(res.Header["Link"])
					if self == "" || t != types.PackageType {
						return fmt.Errorf("Resource %s is not a package", arg)
					}

					pkg := &types.Package{}
					err = json.NewDecoder(res.Body).Decode(pkg)
					if err != nil {
						return err
					}

					w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
					if len(pkg.Members.Packages) > 0 {
						fmt.Fprintln(w, "Packages\tURI\t\t\t\t")
						for _, r := range pkg.Members.Packages {
							fmt.Fprintf(w, "%s\t%s\t\t\t\t\n", r.Title, r.ID)
						}
						fmt.Fprintln(w, "\t\t\t\t\t")
					}
					if len(pkg.Members.Assertions) > 0 {
						fmt.Fprintln(w, "Assertions\tURI\tCreated\tModified")
						for _, a := range pkg.Members.Assertions {
							fmt.Fprintf(w, "%s\t%s\t%s\t%s\t\t\n", a.Title, a.ID, a.Created, a.Modified)
						}
						fmt.Fprintln(w, "\t\t\t\t\t")
					}
					if len(pkg.Members.Files) > 0 {
						fmt.Fprintln(w, "Files\tURI\tCreated\tModified\tFormat\tSize")
						for _, f := range pkg.Members.Files {
							fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\n", f.Title, f.ID, f.Created, f.Modified, f.Format, f.Extent)
						}
						fmt.Fprintln(w, "\t\t\t\t\t")
					}
					return w.Flush()
				},
			},
			{
				Name:      "get",
				Usage:     "print a representation of a resource",
				UsageText: "get --format [format] [resource]",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "format",
						Value: "application/n-quads",
						Usage: "application/n-quads, application/ld+json, or application/json",
					},
				},
				Action: func(c *cli.Context) error {
					key := types.ParsePath(c.Args().First())
					url := types.GetURI(base, key)
					req, err := http.NewRequest("GET", url, nil)
					if err != nil {
						return err
					}

					req.Header.Add("Accept", c.String("format"))
					res, err := http.DefaultClient.Do(req)
					if err != nil {
						return err
					}

					_, err = io.Copy(os.Stdout, res.Body)

					if res.StatusCode != 200 {
						return errors.New(res.Status)
					}

					return err
				},
			},
			{
				Name:  "mkpkg",
				Usage: "create a new package",
				Action: func(c *cli.Context) error {
					arg := c.Args().First()
					key := types.ParsePath(arg)
					url := types.GetURI(base, key)
					req, err := http.NewRequest("MKCOL", url, nil)
					if err != nil {
						return err
					}

					res, err := http.DefaultClient.Do(req)
					if err != nil {
						return err
					}

					if res.StatusCode != 201 {
						return errors.New(res.Status)
					}
					return nil
				},
			},
			{
				Name:  "delete",
				Usage: "delete a resource",
				Action: func(c *cli.Context) error {
					arg := c.Args().First()
					key := types.ParsePath(arg)
					url := types.GetURI(base, key)
					req, err := http.NewRequest("DELETE", url, nil)
					if err != nil {
						return err
					}

					res, err := http.DefaultClient.Do(req)
					if err != nil {
						return err
					}

					if res.StatusCode != 204 {
						return errors.New(res.Status)
					}
					return nil
				},
			},
			{
				Name:  "put",
				Usage: "put a named resource",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "package",
						Aliases: []string{"p"},
					},
					&cli.BoolFlag{
						Name:    "assertion",
						Aliases: []string{"a"},
					},
					&cli.BoolFlag{
						Name:    "file",
						Aliases: []string{"f"},
					},
					&cli.StringFlag{
						Name: "format",
					},
				},
				Action: func(c *cli.Context) error {
					format := c.String("format")
					t, linkType := parseType(c)
					if t == types.PackageType || t == types.AssertionType {
						if format == "" {
							log.Println("Defaulting to --format application/n-quads")
							format = "application/n-quads"
						}
					} else if t == types.FileType {
						if format == "" {
							return errors.New("--format required for --file resources")
						}
					} else {
						return errors.New("must specifiy exactly one of --package, --asssertion, or --file")
					}

					path, resource := c.Args().Get(0), c.Args().Get(1)

					if path == "" {
						return errors.New("File path required")
					} else if resource == "" {
						return errors.New("Resource path required")
					} else if resource == "/" {
						return errors.New("Cannot PUT the root resource")
					} else if strings.HasSuffix(resource, "/") {
						return errors.New("PUT resource paths cannot end in a trailing slash")
					} else if strings.HasSuffix(resource, ".") {
						terms := strings.Split(path, "/")
						name := terms[len(terms)-1]
						if name != "" {
							resource = strings.TrimSuffix(resource, ".") + name
						}
					}

					key := types.ParsePath(resource)
					url := types.GetURI(base, key)

					body, err := os.Open(path)
					if err != nil {
						return err
					}

					req, err := http.NewRequest("PUT", url, body)
					if err != nil {
						return err
					}

					req.Header.Add("Link", types.MakeLinkType(linkType))
					req.Header.Add("Content-Type", format)
					res, err := http.DefaultClient.Do(req)
					if err != nil {
						return err
					}

					if res.StatusCode != 204 {
						return errors.New(res.Status)
					}
					return nil
				},
			},
			{
				Name:  "post",
				Usage: "post an unnamed resource",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "assertion",
						Aliases: []string{"a"},
					},
					&cli.BoolFlag{
						Name:    "file",
						Aliases: []string{"f"},
					},
					&cli.StringFlag{
						Name: "format",
					},
				},
				Action: func(c *cli.Context) error {
					format := c.String("format")
					t, rdfType := parseType(c)
					if t == types.AssertionType {
						if format == "" {
							log.Println("Defaulting to --format application/n-quads")
							format = "application/n-quads"
						}
					} else if t == types.FileType {
						if format == "" {
							return errors.New("--format required for --file resources")
						}
					} else {
						return errors.New("must specifiy exactly one of --asssertion or --file")
					}

					path, resource := c.Args().Get(0), c.Args().Get(1)

					if path == "" {
						return errors.New("File path required")
					} else if resource == "" {
						return errors.New("Resource path required")
					} else if !strings.HasSuffix(resource, "/") {
						return errors.New("POST resource paths must end in a trailing slash")
					}

					key := types.ParsePath(resource)
					url := types.GetURI(base, key)

					body, err := os.Open(path)
					if err != nil {
						return err
					}

					req, err := http.NewRequest("POST", url, body)
					if err != nil {
						return err
					}

					req.Header.Add("Link", types.MakeLinkType(rdfType))
					req.Header.Add("Content-Type", format)
					res, err := http.DefaultClient.Do(req)
					if err != nil {
						return err
					}

					if res.StatusCode != 201 {
						return errors.New(res.Status)
					}

					return nil
				},
			},
			{
				Name:  "query",
				Usage: "query the package server",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "format",
						Value: "application/n-quads",
					},
				},
				Action: func(c *cli.Context) error {
					arg := c.Args().First()
					query := strings.TrimSuffix(arg, "\n")
					lines := strings.Split(query, "\n")
					if query == "" || len(lines) == 0 {
						return errors.New("Empty query")
					}

					quads := make([]*rdf.Quad, len(lines))
					for i, line := range lines {
						quads[i] = rdf.ParseQuad(line)
						if quads[i] == nil {
							return errors.New("Invalid query")
						}
					}

					conn, err := net.Dial("tcp", ":8087")
					if err != nil {
						return err
					}

					stream := newJSONObjectStream(conn)

					ctx := context.Background()
					rpc := jsonrpc2.NewConn(ctx, stream, nil)
					defer rpc.Close()

					var result json.RawMessage
					err = rpc.Call(ctx, "query", []interface{}{quads}, &result)
					if err != nil {
						return err
					}

					terms, err := rdf.UnmarshalTerms(result)
					if err != nil {
						return err
					}

					domain := make([]string, len(terms))
					for i, term := range terms {
						domain[i] = term.String()
					}

					fmt.Print(strings.Join(domain, "\t\t"))

					reader := bufio.NewReader(os.Stdin)
					// fmt.Print("Next: ")
					text, err := reader.ReadString('\n')
					for ; err == nil; text, err = reader.ReadString('\n') {
						params := []interface{}{}
						text = strings.TrimSuffix(text, "\n")
						if text != "" {
							var node rdf.Term
							node, err = rdf.ParseTerm(text)
							if err != nil {
								return err
							}
							params = append(params, node)
						}

						var result json.RawMessage
						err = rpc.Call(ctx, "next", params, &result)
						if err != nil {
							return err
						}

						if result == nil || string(result) == "null" {
							break
						}

						terms, err := rdf.UnmarshalTerms(result)
						if err != nil {
							return err
						}

						start := len(domain) - len(terms)
						values := make([]string, len(domain))
						for i := range values {
							values[i] = "\t"
						}
						for i, term := range terms {
							values[start+i] = term.String()
						}
						fmt.Print(strings.Join(values, "\t"))
					}
					return nil
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func add(path, resource, linkType, format string) error {
	var method string
	var success int
	if resource == "" || strings.HasSuffix(resource, "/") {
		if linkType == types.LinkTypeDirectContainer {
			return errors.New("Cannot add a package anonymously")
		}
		method, success = "POST", 201
	} else {
		method, success = "PUT", 204
	}

	key := types.ParsePath(resource)
	url := types.GetURI(base, key)

	body, err := os.Open(path)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return err
	}

	req.Header.Add("Link", linkType)
	req.Header.Add("Content-Type", format)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode != success {
		return errors.New(res.Status)
	}
	return nil
}

func parseType(c *cli.Context) (t types.ResourceType, rdfType string) {
	if c.Bool("package") {
		t |= types.PackageType
		rdfType = types.LDPDirectContainer
	}
	if c.Bool("assertion") {
		t |= types.AssertionType
		rdfType = types.LDPRDFSource
	}
	if c.Bool("file") {
		t |= types.FileType
		rdfType = types.LDPNonRDFSource
	}
	return
}

type jsonObjectStream struct {
	conn    net.Conn
	encoder *json.Encoder
	decoder *json.Decoder
}

func newJSONObjectStream(conn net.Conn) jsonrpc2.ObjectStream {
	return &jsonObjectStream{
		conn:    conn,
		encoder: json.NewEncoder(conn),
		decoder: json.NewDecoder(conn),
	}
}

func (os *jsonObjectStream) Close() error {
	return os.conn.Close()
}

// WriteObject writes a JSON object to the stream
func (os *jsonObjectStream) WriteObject(obj interface{}) error { return os.encoder.Encode(obj) }

// ReadObject reads a JSON object from the stream
func (os *jsonObjectStream) ReadObject(v interface{}) error { return os.decoder.Decode(v) }
