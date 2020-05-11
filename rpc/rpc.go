package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net"
	"os"
	"os/signal"

	jsonrpc2 "github.com/sourcegraph/jsonrpc2"
	rdf "github.com/underlay/go-rdfjs"
	indices "github.com/underlay/pkgs/indices"
)

// ServeRPC is the exported entrypoint into the RPC server
func ServeRPC() {
	ln, err := net.Listen("tcp", ":8087")
	if err != nil {
		log.Fatalln(err)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		_ = <-c
		os.Exit(1)
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			// handle error
			log.Println(err)
			log.Println(conn.Close())
			continue
		}

		go handleRPC(conn)
	}
}

func handleRPC(conn net.Conn) {
	ctx := context.Background()
	stream := newJSONObjectStream(conn)

	handler := &rpcHandler{}
	c := jsonrpc2.NewConn(ctx, stream, handler)
	<-c.DisconnectNotify()
	if handler.Iterator != nil {
		handler.Iterator.Close()
		handler.Iterator = nil
	}
}

type method func(params []json.RawMessage, handler *rpcHandler) (interface{}, int64, error)

var methods = map[string]method{
	"query": callQuery,
	"next":  callNext,
	"seek":  callSeek,
	"close": callClose,
}

func callQuery(params []json.RawMessage, handler *rpcHandler) (interface{}, int64, error) {
	if len(params) == 0 || len(params) > 3 {
		return nil, jsonrpc2.CodeInvalidParams, nil
	}

	quads := make([]*rdf.Quad, 0)
	err := json.Unmarshal(params[0], &quads)
	if err != nil || len(quads) == 0 {
		return nil, jsonrpc2.CodeInvalidParams, err
	}

	var domain []rdf.Term
	if len(params) > 1 {
		domain, err = rdf.UnmarshalTerms(params[1])
		if err != nil {
			return nil, jsonrpc2.CodeInvalidParams, err
		}
	}

	var index []rdf.Term
	if len(params) > 2 {
		index, err = rdf.UnmarshalTerms(params[2])
		if err != nil {
			return nil, jsonrpc2.CodeInvalidParams, err
		}
	}

	signature, signatureDomain, signatureIndex := getSignature(quads, domain)
	if signature == nil {
		return nil, jsonrpc2.CodeInternalError, errors.New("No matching query signature found")
	}

	handler.Iterator, err = makeIterator(
		quads, domain, index,
		signature, signatureDomain, signatureIndex,
	)

	if err != nil {
		return nil, jsonrpc2.CodeInternalError, err
	}

	return handler.Iterator.Domain(), 0, nil
}

func callClose(params []json.RawMessage, handler *rpcHandler) (interface{}, int64, error) {
	if handler.Iterator == nil {
		return nil, jsonrpc2.CodeInvalidRequest, nil
	}

	if len(params) > 0 {
		return nil, jsonrpc2.CodeInvalidParams, nil
	}

	handler.Iterator.Close()
	handler.Iterator = nil
	return nil, 0, nil
}

func callNext(params []json.RawMessage, handler *rpcHandler) (interface{}, int64, error) {
	if handler.Iterator == nil {
		return nil, jsonrpc2.CodeInvalidRequest, nil
	}

	if len(params) > 1 {
		return nil, jsonrpc2.CodeInvalidParams, nil
	}

	var err error
	var term rdf.Term
	if len(params) > 0 {
		term, err = rdf.UnmarshalTerm(params[0])
		if err != nil {
			return nil, jsonrpc2.CodeInvalidParams, nil
		}

		t := term.TermType()
		if t != rdf.BlankNodeType && t != rdf.VariableType {
			return nil, jsonrpc2.CodeInvalidParams, nil
		}
	}

	delta, err := handler.Iterator.Next(term)

	if err != nil {
		return nil, jsonrpc2.CodeInternalError, err
	}

	return delta, 0, nil
}

type seekParams [][]json.RawMessage

func callSeek(params []json.RawMessage, handler *rpcHandler) (interface{}, int64, error) {
	if handler.Iterator == nil {
		return nil, jsonrpc2.CodeInvalidRequest, nil
	}

	if len(params) > 1 {
		return nil, jsonrpc2.CodeInvalidParams, nil
	}

	var index []rdf.Term
	var err error
	if len(params) > 0 {
		index, err = rdf.UnmarshalTerms(params[0])
		if err != nil {
			return nil, jsonrpc2.CodeInvalidParams, err
		}
	}

	err = handler.Iterator.Seek(index)
	if err != nil {
		return nil, jsonrpc2.CodeInternalError, err
	}

	return nil, 0, nil
}

type rpcHandler struct{ indices.Iterator }

func (handler *rpcHandler) Handle(ctx context.Context, conn *jsonrpc2.Conn, request *jsonrpc2.Request) {
	var result interface{}
	var code int64
	var err error

	if method, has := methods[request.Method]; !has {
		code = jsonrpc2.CodeMethodNotFound
	} else {
		params := make([]json.RawMessage, 0)
		if request.Params != nil {
			err = json.Unmarshal(*request.Params, &params)
			if err != nil {
				code = jsonrpc2.CodeInvalidParams
			}
		}

		if code == 0 && err == nil {
			result, code, err = method(params, handler)
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

type jsonObjectStream struct {
	conn    net.Conn
	encoder *json.Encoder
	decoder *json.Decoder
}

func newJSONObjectStream(conn net.Conn) *jsonObjectStream {
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
