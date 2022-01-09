package langserver

import (
	"context"

	"github.com/open-policy-agent/opa/ast"
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
	result := make(map[lsp.DocumentURI][]lsp.Diagnostic)

	pathToErrs := h.project.GetErrors(documentURIToURI(uri))
	for path, errs := range pathToErrs {
		uri := uriToDocumentURI(path)
		result[uri] = convertErrorsToDiagnostics(errs)
	}

	return result, nil
}

func convertErrorsToDiagnostics(errs ast.Errors) []lsp.Diagnostic {
	result := make([]lsp.Diagnostic, len(errs))
	for i, e := range errs {
		result[i] = convertErrorToDiagnostic(e)
	}
	return result
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
