package source_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kitagry/regols/langserver/internal/source"
	"github.com/kitagry/regols/langserver/internal/source/helper"
	"github.com/open-policy-agent/opa/ast"
)

type createLocationFunc func(files map[string]source.File) *ast.Location

func TestProject_SearchTargetTerm(t *testing.T) {
	tests := map[string]struct {
		files      map[string]source.File
		updateFile map[string]source.File
		expectTerm *ast.Term
	}{
		"Should find term in the body": {
			files: map[string]source.File{
				"main.rego": {
					RawText: `package main

violation[msg] {
	m|sg = "hello"
}`,
				},
			},
			expectTerm: &ast.Term{
				Location: &ast.Location{
					Row:    4,
					Col:    2,
					Offset: len("package main\n\nviolation[msg] {\n	"),
					Text:   []byte("msg"),
					File:   "main.rego",
				},
				Value: ast.Var("msg"),
			},
		},
		"Should find term when the update has not correct ast": {
			files: map[string]source.File{
				"main.rego": {
					RawText: `package main

import data.lib

violation[msg] {
	lib|
}`,
				},
			},
			updateFile: map[string]source.File{
				"main.rego": {
					RawText: `package main

import data.lib

violation[msg] {
	lib.|
}`,
				},
			},
			expectTerm: &ast.Term{
				Location: &ast.Location{
					Row:    6,
					Col:    2,
					Offset: len("package main\n\nimport data.lib\n\nviolation[msg] {\n	"),
					Text:   []byte("lib."),
					File:   "main.rego",
				},
				Value: ast.Ref{
					{
						Location: &ast.Location{
							Row:    6,
							Col:    2,
							Offset: len("package main\n\nimport data.lib\n\nviolation[msg] {\n	"),
							Text:   []byte("lib"),
							File:   "main.rego",
						},
						Value: ast.Var("lib"),
					},
					{
						Location: &ast.Location{
							Row:    6,
							Col:    6,
							Offset: len("package main\n\nimport data.lib\n\nviolation[msg] {\n	lib"),
							Text:   []byte(""),
							File:   "main.rego",
						},
						Value: ast.String(""),
					},
				},
			},
		},
		"Should not find term when the file has not correct ast at first": {
			files: map[string]source.File{
				"main.rego": {
					RawText: `package main

import data.lib

violation[msg] {
	lib |
}`,
				},
			},
		},
		"Should return only library name when the location is on the left side": {
			files: map[string]source.File{
				"main.rego": {
					RawText: `package main

import data.lib

violation[msg] {
	l|ib.method()
}`,
				},
			},
			expectTerm: &ast.Term{
				Location: &ast.Location{
					Row:    6,
					Col:    2,
					Offset: len("package main\n\nimport data.lib\n\nviolation[msg] {\n	"),
					Text:   []byte("lib"),
					File:   "main.rego",
				},
				Value: ast.Var("lib"),
			},
		},
		"Should find term in the else clause": {
			files: map[string]source.File{
				"src.rego": {
					RawText: `package main

authorize = "allow" {
	msg == "allow"
} else = "deny" {
	m|sg == "deny"
} else = "out" {
	msg == "out"
}`,
				},
			},
			expectTerm: &ast.Term{
				Location: &ast.Location{
					Row:    6,
					Col:    2,
					Offset: len("package main\n\nauthorize = \"allow\" {\n	msg ==\"allow\"\n} else = \"deny\" {\n	m"),
					Text:   []byte("msg"),
					File:   "src.rego",
				},
				Value: ast.Var("msg"),
			},
		},
		"Should find term in else of else clause": {
			files: map[string]source.File{
				"src.rego": {
					RawText: `package main

authorize = "allow" {
	msg == "allow"
} else = "deny" {
	msg == "deny"
} else = "out" {
	m|sg == "out"
}`,
				},
			},
			expectTerm: &ast.Term{
				Location: &ast.Location{
					Row:    8,
					Col:    2,
					Offset: len("package main\n\nauthorize = \"allow\" {\n	msg ==\"allow\"\n} else = \"deny\"  {\n	msg ==\"deny\"\n} else = \"out\"  {\n	"),
					Text:   []byte("msg"),
					File:   "src.rego",
				},
				Value: ast.Var("msg"),
			},
		},
		"Should find term in the rule's value": {
			files: map[string]source.File{
				"src.rego": {
					RawText: `package main

authorize = i|nput {
	input.message == "allow"
}`,
				},
			},
			expectTerm: &ast.Term{
				Location: &ast.Location{
					Row:    3,
					Col:    13,
					Offset: len("package main\n\nauthorize = "),
					Text:   []byte("input"),
					File:   "src.rego",
				},
				Value: ast.Var("input"),
			},
		},
		"Should not find term `with` clause": {
			files: map[string]source.File{
				"src_test.rego": {
					RawText: `package main

test_hoge {
	violation w|ith input as "{}"
}

violation[msg] {
	msg := "hello"
}`,
				},
			},
			expectTerm: nil,
		},
		"Should find term in the import sentense": {
			files: map[string]source.File{
				"src.rego": {
					RawText: `package main

import data.li|b`,
				},
			},
			expectTerm: &ast.Term{
				Location: &ast.Location{
					Row:    3,
					Col:    8,
					Offset: len("package main\n\nimport "),
					Text:   []byte("data.lib"),
					File:   "src.rego",
				},
				Value: ast.Ref{
					&ast.Term{
						Value: ast.Var("data"),
						Location: &ast.Location{
							Row:    3,
							Col:    8,
							Offset: len("package main\n\nimport "),
							Text:   []byte("data"),
							File:   "src.rego",
						},
					},
					&ast.Term{
						Value: ast.String("lib"),
						Location: &ast.Location{
							Row:    3,
							Col:    13,
							Offset: len("package main\n\nimport data."),
							Text:   []byte("lib"),
							File:   "src.rego",
						},
					},
				},
			},
		},
		"Should return empty term with empty file": {
			files: map[string]source.File{
				"src.rego": {
					RawText: `|`,
				},
			},
			expectTerm: nil,
		},
		"Should return ast.Var when ast.Ref's final item is ast.Var": {
			files: map[string]source.File{
				"src.rego": {
					RawText: `package src

is_hello() {
	messages[m|sg]
	msg == "hello"
}

messages[msg] {
	msg = input[_]
}`,
				},
			},
			expectTerm: &ast.Term{
				Value: ast.Var("msg"),
				Location: &ast.Location{
					Row:    4,
					Col:    11,
					File:   "src.rego",
					Offset: len("package src\n\nis_hello() {\n	messages["),
					Text:   []byte("msg"),
				},
			},
		},
		"Should list function itself": {
			files: map[string]source.File{
				"src.rego": {
					RawText: `package src

|is_hello(msg) {
	msg == "hello"
}`,
				},
			},
			expectTerm: &ast.Term{
				Value: ast.Var("is_hello"),
				Location: &ast.Location{
					Row:    3,
					Col:    1,
					File:   "src.rego",
					Offset: len("package src\n\n"),
					Text:   []byte("is_hello"),
				},
			},
		},
	}

	for n, tt := range tests {
		t.Run(n, func(t *testing.T) {
			files, location, err := helper.GetAstLocation(tt.files)
			if err != nil {
				t.Fatal(err)
			}

			project, err := source.NewProjectWithFiles(files)
			if err != nil {
				t.Fatal(err)
			}

			if len(tt.updateFile) != 0 {
				var updateFile map[string]source.File
				updateFile, location, err = helper.GetAstLocation(tt.updateFile)
				if err != nil {
					t.Fatal(err)
				}

				for path, file := range updateFile {
					err := project.UpdateFile(path, file.RawText, file.Version)
					if err != nil {
						t.Fatal(err)
					}
				}
			}

			term, err := project.SearchTargetTerm(location)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(tt.expectTerm, term, cmp.Comparer(func(x, y *ast.Term) bool {
				return reflect.DeepEqual(x, y)
			})); diff != "" {
				t.Errorf("SearchTargetTerm result diff (-expect, +got)\n%s", diff)
			}
		})
	}
}

func createLocation(row, col int, file string) createLocationFunc {
	return func(files map[string]source.File) *ast.Location {
		rawText := files[file].RawText

		offset := 0
		for i := 1; i < row; i++ {
			offset += strings.Index(rawText[offset:], "\n") + 1
		}
		offset += col

		var text []byte
		if offset > 0 && offset <= len(rawText) {
			text = []byte{rawText[offset-1]}
		}

		return &ast.Location{
			Row:    row,
			Col:    col,
			Offset: offset,
			Text:   text,
			File:   file,
		}
	}
}
