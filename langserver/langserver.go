package langserver

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/open-policy-agent/opa/ast"
	"github.com/sourcegraph/go-lsp"
	"github.com/sourcegraph/jsonrpc2"
)

type handler struct {
	conn        *jsonrpc2.Conn
	logger      *log.Logger
	lintRequest chan lsp.DocumentURI
	files       map[lsp.DocumentURI]document
}

type document struct {
	Text    regoText
	Version int
}

type regoText struct {
	statements []ast.Statement
	comments   []*ast.Comment
	err        ast.Errors
}

func NewRegoText(text string) regoText {
	parser := ast.NewParser()
	parser.WithReader(strings.NewReader(text))
	statements, comments, err := parser.Parse()
	return regoText{
		statements, comments, err,
	}
}

func NewHandler() jsonrpc2.Handler {
	handler := &handler{
		files:       make(map[lsp.DocumentURI]document),
		logger:      log.New(os.Stderr, "", log.LstdFlags),
		lintRequest: make(chan lsp.DocumentURI, 3),
	}
	go handler.linter()
	return jsonrpc2.HandlerWithError(handler.handle)
}

func (h *handler) linter() {
	running := make(map[lsp.DocumentURI]context.CancelFunc)

	for {
		uri, ok := <-h.lintRequest
		if !ok {
			break
		}

		cancel, ok := running[uri]
		if ok {
			cancel()
		}

		ctx, cancel := context.WithCancel(context.Background())
		running[uri] = cancel

		go func() {
			diagnostics, err := h.lint(ctx, uri)
			if err != nil {
				h.logger.Println(err)
				return
			}

			for uri, d := range diagnostics {
				h.conn.Notify(ctx, "textDocument/publishDiagnostics", lsp.PublishDiagnosticsParams{
					URI:         uri,
					Diagnostics: d,
				})
			}
		}()
	}
}

func (h *handler) lint(ctx context.Context, uri lsp.DocumentURI) (map[lsp.DocumentURI][]lsp.Diagnostic, error) {
	document, ok := h.files[uri]
	if !ok {
		return nil, fmt.Errorf("file not found: %s", uri)
	}

	result := make(map[lsp.DocumentURI][]lsp.Diagnostic)
	diagnostics := make([]lsp.Diagnostic, len(document.Text.err))
	for i, e := range document.Text.err {
		diagnostics[i] = lsp.Diagnostic{
			Severity: lsp.Error,
			Range: lsp.Range{
				Start: lsp.Position{
					Line:      e.Location.Row,
					Character: e.Location.Col,
				},
				End: lsp.Position{
					Line:      e.Location.Row,
					Character: e.Location.Col + e.Location.Offset,
				},
			},
			Message: e.Message,
		}
	}
	result[uri] = diagnostics

	return result, nil
}

func (h *handler) handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result interface{}, err error) {
	switch req.Method {
	case "initialize":
		return h.handleInitialize(ctx, conn, req)
	case "initialized":
		return
	case "textDocument/didOpen":
		return h.handleTextDocumentDidOpen(ctx, conn, req)
	case "textDocument/didChange":
		return h.handleTextDocumentDidChange(ctx, conn, req)
	case "textDocument/didClose":
		return h.handleTextDocumentDidClose(ctx, conn, req)
	case "textDocument/didSave":
		return h.handleTextDocumentDidSave(ctx, conn, req)
	}
	return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeMethodNotFound, Message: fmt.Sprintf("method not supported: %s", req.Method)}
}
