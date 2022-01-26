package source_test

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kitagry/regols/langserver/internal/source"
	"github.com/open-policy-agent/opa/ast"
)

type createLocationFunc func(files map[string]source.File) *ast.Location

func TestProject_SearchTargetTerm(t *testing.T) {
	tests := map[string]struct {
		files          map[string]source.File
		updateFile     map[string]source.File
		createLocation createLocationFunc
		expectTerm     *ast.Term
	}{
		"search term": {
			files: map[string]source.File{
				"main.rego": {
					RowText: `package main

violation[msg] {
	msg = "hello"
}`,
				},
			},
			createLocation: createLocation(4, 2, "m", "main.rego"),
			expectTerm: &ast.Term{
				Location: &ast.Location{
					Row: 4,
					Col: 2,
					Offset: len("package main\n\nviolation[msg] {\n	m"),
					Text: []byte("msg"),
					File: "main.rego",
				},
				Value: ast.Var("msg"),
			},
		},
		"search ref of library": {
			files: map[string]source.File{
				"main.rego": {
					RowText: `package main

import data.lib

violation[msg] {
	lib
}`,
				},
			},
			updateFile: map[string]source.File{
				"main.rego": {
					RowText: `package main

import data.lib

violation[msg] {
	lib.
}`,
				},
			},
			createLocation: createLocation(6, 5, ".", "main.rego"),
			expectTerm: &ast.Term{
				Location: &ast.Location{
					Row: 6,
					Col: 2,
					Offset: len("package main\n\nimport data.lib\n\nviolation[msg] {\n	l"),
					Text: []byte("lib."),
					File: "main.rego",
				},
				Value: ast.Ref{
					{
						Location: &ast.Location{
							Row: 6,
							Col: 2,
							Offset: len("package main\n\nimport data.lib\n\nviolation[msg] {\n	l"),
							Text: []byte("lib"),
							File: "main.rego",
						},
						Value: ast.Var("lib"),
					},
					{
						Location: &ast.Location{
							Row: 6,
							Col: 2,
							Offset: len("package main\n\nimport data.lib\n\nviolation[msg] {\n	lib."),
							Text: []byte(""),
							File: "main.rego",
						},
						Value: ast.String(""),
					},
				},
			},
		},
		"when parse error first, can't find term": {
			files: map[string]source.File{
				"main.rego": {
					RowText: `package main

import data.lib

violation[msg] {
	lib
}`,
				},
			},
			createLocation: createLocation(6, 5, ".", "main.rego"),
		},
		"library term is itself": {
			files: map[string]source.File{
				"main.rego": {
					RowText: `package main

import data.lib

violation[msg] {
	lib.method()
}`,
				},
			},
			createLocation: createLocation(6, 2, "l", "main.rego"),
			expectTerm: &ast.Term{
				Location: &ast.Location{
					Row: 6,
					Col: 2,
					Offset: len("package main\n\nimport data.lib\n\nviolation[msg] {\n	l"),
					Text: []byte("lib"),
					File: "main.rego",
				},
				Value: ast.Var("lib"),
			},
		},
		"searchTerm in else": {
			files: map[string]source.File{
				"src.rego": {
					RowText: `package main

authorize = "allow" {
	msg == "allow"
} else = "deny" {
	msg == "deny"
} else = "out" {
	msg == "out"
}`,
				},
			},
			createLocation: createLocation(6, 2, "m", "src.rego"),
			expectTerm: &ast.Term{
				Location: &ast.Location{
					Row: 6,
					Col: 2,
					Offset: len("package main\n\nauthorize = \"allow\" {\n	msg ==\"allow\"\n} else = \"deny\" {\n	m"),
					Text: []byte("msg"),
					File: "src.rego",
				},
				Value: ast.Var("msg"),
			},
		},
		"searchTerm in else of else": {
			files: map[string]source.File{
				"src.rego": {
					RowText: `package main

authorize = "allow" {
	msg == "allow"
} else = "deny" {
	msg == "deny"
} else = "out" {
	msg == "out"
}`,
				},
			},
			createLocation: createLocation(8, 2, "m", "src.rego"),
			expectTerm: &ast.Term{
				Location: &ast.Location{
					Row: 8,
					Col: 2,
					Offset: len("package main\n\nauthorize = \"allow\" {\n	msg ==\"allow\"\n} else = \"deny\"  {\n	msg ==\"deny\"\n} else = \"out\"  {\n	m"),
					Text: []byte("msg"),
					File: "src.rego",
				},
				Value: ast.Var("msg"),
			},
		},
		"searchTerm in head value": {
			files: map[string]source.File{
				"src.rego": {
					RowText: `package main

authorize = input {
	input.message == "allow"
}`,
				},
			},
			createLocation: createLocation(3, 13, "i", "src.rego"),
			expectTerm: &ast.Term{
				Location: &ast.Location{
					Row:    3,
					Col:    13,
					Offset: len("package main\n\nauthorize = i"),
					Text:   []byte("input"),
					File:   "src.rego",
				},
				Value: ast.Var("input"),
			},
		},
		"searchTerm with `with`": {
			files: map[string]source.File{
				"src_test.rego": {
					RowText: `package main

test_hoge {
	violation with input as "{}"
}

violation[msg] {
	msg := "hello"
}`,
				},
			},
			createLocation: createLocation(4, 12, "w", "src_test.rego"),
			expectTerm:     nil,
		},
		"searchTerm in import": {
			files: map[string]source.File{
				"src.rego": {
					RowText: `package main

import data.lib`,
				},
			},
			createLocation: createLocation(3, 13, "i", "src.rego"),
			expectTerm: &ast.Term{
				Location: &ast.Location{
					Row:    3,
					Col:    8,
					Offset: len("package main\n\nimport d"),
					Text:   []byte("data.lib"),
					File:   "src.rego",
				},
				Value: ast.Ref{
					&ast.Term{
						Value: ast.Var("data"),
						Location: &ast.Location{
							Row:    3,
							Col:    8,
							Offset: len("package main\n\nimport d"),
							Text:   []byte("data"),
							File:   "src.rego",
						},
					},
					&ast.Term{
						Value: ast.String("lib"),
						Location: &ast.Location{
							Row:    3,
							Col:    13,
							Offset: len("package main\n\nimport data.l"),
							Text:   []byte("lib"),
							File:   "src.rego",
						},
					},
				},
			},
		},
	}

	for n, tt := range tests {
		t.Run(n, func(t *testing.T) {
			project, err := source.NewProjectWithFiles(tt.files)
			if err != nil {
				t.Fatal(err)
			}

			for path, file := range tt.updateFile {
				err := project.UpdateFile(path, file.RowText, file.Version)
				if err != nil {
					t.Fatal(err)
				}
			}

			location := tt.createLocation(tt.files)
			term, err := project.SearchTargetTerm(location)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(tt.expectTerm, term); diff != "" {
				t.Errorf("SearchTargetTerm result diff (-expect, +got)\n%s", diff)
			}
		})
	}
}

func createLocation(row, col int, text, file string) createLocationFunc {
	return func(files map[string]source.File) *ast.Location {
		rawText := files[file].RowText

		offset := 0
		for i := 1; i < row; i++ {
			offset += strings.Index(rawText[offset:], "\n") + 1
		}
		offset += col

		return &ast.Location{
			Row:    row,
			Col:    col,
			Offset: offset,
			Text:   []byte(text),
			File:   file,
		}
	}
}
