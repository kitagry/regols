package helper

import (
	"errors"
	"strings"

	"github.com/kitagry/regols/langserver/internal/lsp"
	"github.com/kitagry/regols/langserver/internal/source"
	"github.com/open-policy-agent/opa/ast"
)

var ErrNoPosition = errors.New("no position")

func GetAstLocation(files map[string]source.File) (formattedFiles map[string]source.File, location *ast.Location, err error) {
	formattedFiles = make(map[string]source.File)
	for filePath, file := range files {
		if ind := strings.Index(file.RawText, "|"); ind != -1 {
			file.RawText = strings.Replace(file.RawText, "|", "", 1)
			position := indexToPosition(file.RawText, ind)

			offset := 0
			for i := 0; i < position.Line; i++ {
				offset += strings.Index(file.RawText[offset:], "\n") + 1
			}
			offset += position.Character

			location = &ast.Location{
				Row:    position.Line + 1,
				Col:    position.Character + 1,
				Offset: offset,
				File:   filePath,
				Text:   []byte(file.RawText),
			}
		}
		formattedFiles[filePath] = file
	}

	if location == nil {
		return nil, nil, ErrNoPosition
	}

	return
}

func indexToPosition(file string, index int) lsp.Position {
	col, row := 0, 0
	lines := strings.Split(file, "\n")
	for _, line := range lines {
		if index <= len(line) {
			col = index
			break
		}
		index -= len(line) + 1
		row++
	}
	return lsp.Position{Line: row, Character: col}
}
