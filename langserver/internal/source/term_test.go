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
