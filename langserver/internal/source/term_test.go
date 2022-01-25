package source_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kitagry/regols/langserver/internal/source"
	"github.com/open-policy-agent/opa/ast"
)

func TestProject_SearchTargetTerm(t *testing.T) {
	tests := map[string]struct {
		files      map[string]source.File
		updateFile map[string]source.File
		location   *ast.Location
		expectTerm *ast.Term
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
			location: &ast.Location{
				Row: 4,
				Col: 2,
				Offset: len("package main\n\nviolation[msg] {\n	m"),
				Text: []byte("m"),
				File: "main.rego",
			},
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
			location: &ast.Location{
				Row: 6,
				Col: 5,
				Offset: len("package main\n\nimport data.lib\n\nviolation[msg] {\n	lib."),
				Text: []byte("."),
				File: "main.rego",
			},
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
			location: &ast.Location{
				Row: 6,
				Col: 5,
				Offset: len("package main\n\nimport data.lib\n\nviolation[msg] {\n	lib."),
				Text: []byte("."),
				File: "main.rego",
			},
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
			location: &ast.Location{
				Row: 6,
				Col: 2,
				Offset: len("package main\n\nimport data.lib\n\nviolation[msg] {\n	l"),
				Text: []byte("l"),
				File: "main.rego",
			},
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
			location: &ast.Location{
				Row: 6,
				Col: 2,
				Offset: len("package main\n\nauthorize = \"allow\" {\n	msg ==\"allow\"\n} else = \"deny\"  {\n	m"),
				Text: []byte("m"),
				File: "src.rego",
			},
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
			location: &ast.Location{
				Row: 8,
				Col: 2,
				Offset: len("package main\n\nauthorize = \"allow\" {\n	msg ==\"allow\"\n} else = \"deny\"  {\n	msg ==\"deny\"\n} else = \"out\"  {\n	m"),
				Text: []byte("m"),
				File: "src.rego",
			},
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
			location: &ast.Location{
				Row:    3,
				Col:    13,
				Offset: len("package main\n\nauthorize = i"),
				Text:   []byte("i"),
				File:   "src.rego",
			},
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
			location: &ast.Location{
				Row: 4,
				Col: 12,
				Offset: len("package main\n\ntest_hoge {\n	violation w"),
				Text: []byte("w"),
				File: "src_test.rego",
			},
			expectTerm: nil,
		},
		"searchTerm in import": {
			files: map[string]source.File{
				"src.rego": {
					RowText: `package main

import data.lib`,
				},
			},
			location: &ast.Location{
				Row:    3,
				Col:    13,
				Offset: len("package main\n\nimport data.l"),
				Text:   []byte("l"),
				File:   "src.rego",
			},
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

			term, err := project.SearchTargetTerm(tt.location)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(tt.expectTerm, term); diff != "" {
				t.Errorf("SearchTargetTerm result diff (-expect, +got)\n%s", diff)
			}
		})
	}
}
