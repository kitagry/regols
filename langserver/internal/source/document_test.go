package source_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kitagry/regols/langserver/internal/source"
)

func TestProject_TermDocument(t *testing.T) {
	tests := map[string]struct {
		files          map[string]source.File
		createLocation createLocationFunc
		expectDocs     []source.Document
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
			createLocation: createLocation(4, 2, "src.rego"),
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
			createLocation: createLocation(4, 2, "src.rego"),
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
			createLocation: createLocation(4, 2, "src.rego"),
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
			createLocation: createLocation(4, 7, "src.rego"),
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

			location := tt.createLocation(tt.files)
			docs, err := project.TermDocument(location)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(tt.expectDocs, docs); diff != "" {
				t.Errorf("TermDocument result diff (-expect, +got)\n%s", diff)
			}
		})
	}
}
