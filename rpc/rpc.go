package rpc

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"log"
	"net"
	"os"
	"os/signal"

	ld "github.com/piprate/json-gold/ld"
	jsonrpc2 "github.com/sourcegraph/jsonrpc2"

	indices "github.com/underlay/pkgs/indices"
	query "github.com/underlay/pkgs/query"
	rdf "github.com/underlay/pkgs/rdf"
)

type method interface {
	call(handler *rpcHandler) (interface{}, int64, error)
}

var methods = map[string](func() method){
	"cursor":        func() method { return &newCursorParams{} },
	"cursor.get":    func() method { return &cursorGetParams{} },
	"cursor.graph":  func() method { return &cursorGraphParams{} },
	"cursor.domain": func() method { return &cursorDomainParams{} },
	"cursor.index":  func() method { return &cursorIndexParams{} },
	"cursor.next":   func() method { return &cursorNextParams{} },
	"cursor.seek":   func() method { return &cursorSeekParams{} },
}

type rpcHandler struct {
	cursor query.Cursor
}

type newCursorParams struct {
	Query  []*rdf.Quad       `json:"query"`
	Domain []json.RawMessage `json:"domain"`
	Index  []json.RawMessage `json:"index"`
}

func (params *newCursorParams) call(handler *rpcHandler) (interface{}, int64, error) {
	query := make([]*ld.Quad, len(params.Query))
	for i, quad := range params.Query {
		query[i] = rdf.FromQuad(quad)
	}

	signature, assignments := getSignature(query)
	if signature == nil {
		return nil, jsonrpc2.CodeInvalidRequest, nil
	}

	domain := make([]*ld.BlankNode, len(params.Domain))
	for i, data := range params.Domain {
		term, err := rdf.UnmarshalTerm(data)
		if err != nil {
			return nil, jsonrpc2.CodeInvalidParams, err
		}
		domain[i] = ld.NewBlankNode(term.Value())
	}

	index := make([]ld.Node, len(params.Index))
	for i, data := range params.Index {
		term, err := rdf.UnmarshalTerm(data)
		if err != nil {
			return nil, jsonrpc2.CodeInvalidParams, err
		}
		index[i] = rdf.FromTerm(term)
	}

	cursor, err := signature.Query(query, assignments, domain, index)
	handler.cursor = cursor
	return nil, 0, err
}

type cursorGetParams struct {
	Node json.RawMessage `json:"node"`
}

func (params *cursorGetParams) call(handler *rpcHandler) (interface{}, int64, error) {
	term, err := rdf.UnmarshalTerm(params.Node)
	if err != nil {
		return nil, jsonrpc2.CodeInvalidParams, err
	}
	node := ld.NewBlankNode(term.Value())
	value := handler.cursor.Get(node)
	return rdf.ToTerm(value), 0, nil
}

type cursorGraphParams struct{}

func (params *cursorGraphParams) call(handler *rpcHandler) (interface{}, int64, error) {
	quads := make([]*rdf.Quad, handler.cursor.Len())
	for i, quad := range handler.cursor.Graph() {
		quads[i] = rdf.ToQuad(quad)
	}
	return quads, 0, nil
}

type cursorDomainParams struct{}

func (params *cursorDomainParams) call(handler *rpcHandler) (interface{}, int64, error) {
	domain := make([]rdf.Term, handler.cursor.Len())
	for i, node := range handler.cursor.Domain() {
		domain[i] = rdf.ToTerm(node)
	}
	return domain, 0, nil
}

type cursorIndexParams struct{}

func (params *cursorIndexParams) call(handler *rpcHandler) (interface{}, int64, error) {
	index := make([]rdf.Term, handler.cursor.Len())
	for i, node := range handler.cursor.Index() {
		index[i] = rdf.ToTerm(node)
	}
	return index, 0, nil
}

type cursorNextParams struct {
	Node json.RawMessage `json:"node"`
}

func (params *cursorNextParams) call(handler *rpcHandler) (interface{}, int64, error) {
	term, err := rdf.UnmarshalTerm(params.Node)
	if err != nil {
		return nil, jsonrpc2.CodeInvalidParams, err
	}

	var node *ld.BlankNode
	if term != nil {
		if blank, is := rdf.FromTerm(term).(*ld.BlankNode); is {
			node = blank
		} else {
			return nil, jsonrpc2.CodeInvalidParams, err
		}
	}

	tail, err := handler.cursor.Next(node)
	if err != nil {
		return nil, jsonrpc2.CodeInternalError, err
	}

	terms := make([]rdf.Term, len(tail))
	for i, blank := range tail {
		terms[i] = rdf.ToTerm(blank)
	}

	return terms, 0, nil
}

type cursorSeekParams struct {
	Index []json.RawMessage `json:"index"`
}

func (params *cursorSeekParams) call(handler *rpcHandler) (interface{}, int64, error) {
	index := make([]ld.Node, len(params.Index))
	for i, node := range params.Index {
		term, err := rdf.UnmarshalTerm(node)
		if err != nil {
			return nil, jsonrpc2.CodeInvalidParams, err
		}
		index[i] = rdf.FromTerm(term)
	}
	err := handler.cursor.Seek(index)
	if err != nil {
		return nil, jsonrpc2.CodeInternalError, nil
	}
	return nil, 0, nil
}

func (handler *rpcHandler) Handle(ctx context.Context, conn *jsonrpc2.Conn, request *jsonrpc2.Request) {
	var result interface{}
	var code int64
	var err error

	if method, has := methods[request.Method]; !has {
		code = jsonrpc2.CodeMethodNotFound
	} else {
		params := method()
		err = json.Unmarshal(*request.Params, params)
		if err != nil {
			code = jsonrpc2.CodeInvalidParams
		} else {
			result, code, err = params.call(handler)
		}
	}

	if code != 0 {
		respErr := &jsonrpc2.Error{Code: code}
		if err != nil {
			respErr.Message = err.Error()
		}
		_ = conn.ReplyWithError(ctx, request.ID, respErr)
	} else {
		conn.Reply(ctx, request.ID, result)
	}
}

// ServeRPC is the exported entrypoint into the RPC server
func ServeRPC() {
	ln, err := net.Listen("tcp", ":8087")
	if err != nil {
		// handle error
	}

	log.Println("Opening indices")
	for _, index := range indices.INDICES {
		index.Open()
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		_ = <-c
		log.Println("Closing indices")
		for _, index := range indices.INDICES {
			index.Close()
		}
		os.Exit(1)
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			// handle error
		}
		go handleRPC(conn)
	}
}

func handleRPC(conn io.ReadWriteCloser) {
	ctx := context.Background()
	stream := jsonrpc2.NewBufferedStream(conn, jsonObjectStream{})
	handler := &rpcHandler{}
	c := jsonrpc2.NewConn(ctx, stream, handler)
	<-c.DisconnectNotify()
	if handler.cursor != nil {
		handler.cursor.Close()
	}
}

type jsonObjectStream struct{}

// WriteObject writes a JSON object to the stream
func (os jsonObjectStream) WriteObject(stream io.Writer, obj interface{}) error {
	return json.NewEncoder(stream).Encode(obj)
}

// ReadObject reads a JSON object from the stream
func (os jsonObjectStream) ReadObject(stream *bufio.Reader, v interface{}) error {
	return json.NewDecoder(stream).Decode(v)
}
