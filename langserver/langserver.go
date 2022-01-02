package langserver

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/loader"
	"github.com/sourcegraph/go-lsp"
	"github.com/sourcegraph/jsonrpc2"
)

type handler struct {
	conn        *jsonrpc2.Conn
	logger      *log.Logger
	lintRequest chan lsp.DocumentURI
	files       map[lsp.DocumentURI]document
	rootPath    string
}

type document struct {
	Text    string
	Version int
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

	module, err := ast.ParseModule(documentURIToURI(uri), document.Text)
	if errs, ok := err.(ast.Errors); ok {
		diagnostics := make([]lsp.Diagnostic, len(errs))
		for i, e := range errs {
			diagnostics[i] = convertErrorToDiagnostic(e)
		}
		result[uri] = diagnostics
		return result, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to parse module: %w", err)
	}

	policies, err := loader.AllRegos([]string{h.rootPath})
	if err != nil {
		return nil, err
	}

	modules := policies.ParsedModules()
	modules[documentURIToURI(uri)] = module
	h.logger.Println(modules)
	compiler := ast.NewCompiler()
	compiler.Compile(modules)

	if compiler.Failed() {
		for _, e := range compiler.Errors {
			uri := uriToDocumentURI(e.Location.File)
			result[uri] = append(result[uri], convertErrorToDiagnostic(e))
		}
	}

	// Refresh old diagnostics.
	for uri := range h.files {
		if _, ok := result[uri]; !ok {
			result[uri] = make([]lsp.Diagnostic, 0)
		}
	}

	return result, nil
}

func convertErrorToDiagnostic(err *ast.Error) lsp.Diagnostic {
	return lsp.Diagnostic{
		Severity: lsp.Error,
		Range: lsp.Range{
			Start: lsp.Position{
				Line:      err.Location.Row - 1,
				Character: err.Location.Col - 1,
			},
			End: lsp.Position{
				Line:      err.Location.Row - 1,
				Character: err.Location.Col + err.Location.Offset - 1,
			},
		},
		Message: err.Message,
	}
}

func uriToDocumentURI(uri string) lsp.DocumentURI {
	return lsp.DocumentURI(fmt.Sprintf("file://%s", uri))
}

func documentURIToURI(duri lsp.DocumentURI) string {
	return string(duri)[len("file://"):]
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
