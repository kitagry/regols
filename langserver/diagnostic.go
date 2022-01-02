package langserver

import (
	"context"
	"fmt"

	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/loader"
	"github.com/sourcegraph/go-lsp"
)

func (h *handler) diagnostic() {
	running := make(map[lsp.DocumentURI]context.CancelFunc)

	for {
		uri, ok := <-h.diagnosticRequest
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
			diagnostics, err := h.diagnose(ctx, uri)
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

func (h *handler) diagnose(ctx context.Context, uri lsp.DocumentURI) (map[lsp.DocumentURI][]lsp.Diagnostic, error) {
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
	// Change module in order to use no saved file.
	modules[documentURIToURI(uri)] = module

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
