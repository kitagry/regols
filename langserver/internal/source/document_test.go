package source_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kitagry/regols/langserver/internal/source"
	"github.com/open-policy-agent/opa/ast"
)

func TestProject_TermDocument(t *testing.T) {
	tests := map[string]struct {
		files      map[string]source.File
		location   *ast.Location
		expectDocs []source.Document
	}{
		"document in same file method": {
			files: map[string]source.File{
				"src.rego": {
					RowText: `package src

violation[msg] {
	method(msg)
}

method(msg) {
	msg == "hello"
}`,
				},
			},
			location: &ast.Location{
				Row: 4,
				Col: 2,
				Offset: len("package src\n\nviolation[msg] {	m"),
				Text: []byte("m"),
				File: "src.rego",
			},
			expectDocs: []source.Document{
				{
					Content: `method(msg) {
	msg == "hello"
}`,
					Language: "rego",
				},
			},
		},
		"default can show all": {
			files: map[string]source.File{
				"src.rego": {
					RowText: `package src

violation[msg] {
	item
}

default item = "hello"`,
				},
			},
			location: &ast.Location{
				Row: 4,
				Col: 2,
				Offset: len("package src\n\nviolation[msg] {	i"),
				Text: []byte("i"),
				File: "src.rego",
			},
			expectDocs: []source.Document{
				{
					Content:  `default item = "hello"`,
					Language: "rego",
				},
			},
		},
		"builtin function": {
			files: map[string]source.File{
				"src.rego": {
					RowText: `package src

violation[msg] {
	sprintf("msg: %s", [msg])
}`,
				},
			},
			location: &ast.Location{
				Row: 4,
				Col: 2,
				Offset: len("package src\n\nviolation[msg] {	s"),
				Text: []byte("s"),
				File: "src.rego",
			},
			expectDocs: []source.Document{
				{
					Content:  "sprintf(string, array[any])",
					Language: "rego",
				},
				{
					Content: `built-in function

See https://www.openpolicyagent.org/docs/latest/policy-reference/#built-in-functions`,
					Language: "markdown",
				},
			},
		},
		"builtin function with ref": {
			files: map[string]source.File{
				"src.rego": {
					RowText: `package src

violation[msg] {
	json.is_valid("{}")
}`,
				},
			},
			location: &ast.Location{
				Row: 4,
				Col: 7,
				Offset: len("package src\n\nviolation[msg] {	json.i"),
				Text: []byte("i"),
				File: "src.rego",
			},
			expectDocs: []source.Document{
				{
					Content:  "json.is_valid(string)",
					Language: "rego",
				},
				{
					Content: `built-in function

See https://www.openpolicyagent.org/docs/latest/policy-reference/#built-in-functions`,
					Language: "markdown",
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

			docs, err := project.TermDocument(tt.location)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(tt.expectDocs, docs); diff != "" {
				t.Errorf("TermDocument result diff (-expect, +got)\n%s", diff)
			}
		})
	}
}
