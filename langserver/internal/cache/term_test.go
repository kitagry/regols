package cache_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kitagry/regols/langserver/internal/cache"
	"github.com/open-policy-agent/opa/ast"
)

func TestProject_SearchTargetTerm(t *testing.T) {
	tests := map[string]struct {
		files      map[string]cache.File
		updateFile map[string]cache.File
		location   *ast.Location
		expectTerm *ast.Term
	}{
		"search term": {
			files: map[string]cache.File{
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
			files: map[string]cache.File{
				"main.rego": {
					RowText: `package main

import data.lib

violation[msg] {
	lib
}`,
				},
			},
			updateFile: map[string]cache.File{
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
	}

	for n, tt := range tests {
		t.Run(n, func(t *testing.T) {
			project, err := cache.NewProjectWithFiles(tt.files)
			if err != nil {
				t.Fatal(err)
			}

			for path, file := range tt.updateFile {
				project.UpdateFile(path, file.RowText, file.Version)
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
