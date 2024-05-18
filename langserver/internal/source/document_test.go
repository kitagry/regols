package source_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kitagry/regols/langserver/internal/source"
	"github.com/kitagry/regols/langserver/internal/source/helper"
)

func TestProject_TermDocument(t *testing.T) {
	tests := map[string]struct {
		files      map[string]source.File
		expectDocs []source.Document
	}{
		"Should document rule in same file method": {
			files: map[string]source.File{
				"src.rego": {
					RawText: `package src

violation[msg] {
	m|ethod(msg)
}

method(msg) {
	msg == "hello"
}`,
				},
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
		"Should document rule which has default": {
			files: map[string]source.File{
				"src.rego": {
					RawText: `package src

violation[msg] {
	i|tem
}

default item = "hello"`,
				},
			},
			expectDocs: []source.Document{
				{
					Content:  `default item = "hello"`,
					Language: "rego",
				},
			},
		},
		"Should document builtin function": {
			files: map[string]source.File{
				"src.rego": {
					RawText: `package src

violation[msg] {
	s|printf("msg: %s", [msg])
}`,
				},
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
		`Should document builtin function which has "." text`: {
			files: map[string]source.File{
				"src.rego": {
					RawText: `package src

violation[msg] {
	json.i|s_valid("{}")
}`,
				},
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
			files, location, err := helper.GetAstLocation(tt.files)
			if err != nil {
				t.Fatal(err)
			}

			project, err := source.NewProjectWithFiles(files)
			if err != nil {
				t.Fatal(err)
			}

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
