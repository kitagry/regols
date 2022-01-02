package langserver

import (
	"context"

	"github.com/sourcegraph/jsonrpc2"
)

func (h *handler) handleTextDocumentDefinition(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result interface{}, err error) {
	return nil, nil
}
