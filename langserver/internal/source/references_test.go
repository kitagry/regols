package source_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kitagry/regols/langserver/internal/source"
	"github.com/open-policy-agent/opa/ast"
)

func TestLookupReferences(t *testing.T) {
	tests := map[string]struct {
		files          map[string]source.File
		createLocation createLocationFunc
		expectResult   []*ast.Location
		expectErr      error
	}{
		"Should list self": {
			files: map[string]source.File{
				"src.rego": {
					RawText: `package src

violation[msg] {
	hello := "hello"
}`,
				},
			},
			createLocation: createLocation(4, 2, "src.rego"),
			expectResult: []*ast.Location{
				{
					Row: 4,
					Col: 2,
					Offset: len("package src\n\nviolation[msg] {\n	"),
					Text: []byte("hello"),
					File: "src.rego",
				},
			},
		},
		"Should list above item": {
			files: map[string]source.File{
				"src.rego": {
					RawText: `package src

violation[msg] {
	hello := "hello"
	is_hello(hello)
}`,
				},
			},
			createLocation: createLocation(5, 10, "src.rego"),
			expectResult: []*ast.Location{
				{
					Row: 4,
					Col: 2,
					Offset: len("package src\n\nviolation[msg] {\n	"),
					Text: []byte("hello"),
					File: "src.rego",
				},
				{
					Row: 5,
					Col: 11,
					Offset: len("package src\n\nviolation[msg] {\n	hello := \"hello\"\n	is_hello("),
					Text: []byte("hello"),
					File: "src.rego",
				},
			},
		},
		"Should list rule's key": {
			files: map[string]source.File{
				"src.rego": {
					RawText: `package src

violation[msg] {
	trace(msg)
}`,
				},
			},
			createLocation: createLocation(4, 8, "src.rego"),
			expectResult: []*ast.Location{
				{
					Row:    3,
					Col:    11,
					Offset: len("package src\n\nviolation["),
					Text:   []byte("msg"),
					File:   "src.rego",
				},
				{
					Row: 4,
					Col: 8,
					Offset: len("package src\n\nviolation[msg] {\n	trace("),
					Text: []byte("msg"),
					File: "src.rego",
				},
			},
		},
		"Should list rule's args": {
			files: map[string]source.File{
				"src.rego": {
					RawText: `package src

violation(msg) {
	trace(msg)
}`,
				},
			},
			createLocation: createLocation(4, 8, "src.rego"),
			expectResult: []*ast.Location{
				{
					Row:    3,
					Col:    11,
					Offset: len("package src\n\nviolation("),
					Text:   []byte("msg"),
					File:   "src.rego",
				},
				{
					Row: 4,
					Col: 8,
					Offset: len("package src\n\nviolation(msg) {\n	trace("),
					Text: []byte("msg"),
					File: "src.rego",
				},
			},
		},
		"Should list rule's value": {
			files: map[string]source.File{
				"src.rego": {
					RawText: `package src

violation = msg {
	trace(msg)
}`,
				},
			},
			createLocation: createLocation(4, 8, "src.rego"),
			expectResult: []*ast.Location{
				{
					Row:    3,
					Col:    13,
					Offset: len("package src\n\nviolation = "),
					Text:   []byte("msg"),
					File:   "src.rego",
				},
				{
					Row: 4,
					Col: 8,
					Offset: len("package src\n\nviolation = msg {\n	trace("),
					Text: []byte("msg"),
					File: "src.rego",
				},
			},
		},
		"Should list function references": {
			files: map[string]source.File{
				"src.rego": {
					RawText: `package src

violation[msg] {
	is_hello(msg)
}

is_hello(msg) {
	msg == "hello"
}`,
				},
			},
			createLocation: createLocation(4, 2, "src.rego"),
			expectResult: []*ast.Location{
				{
					Row: 4,
					Col: 2,
					Offset: len("package src\n\nviolation[msg] {\n	"),
					Text: []byte("is_hello"),
					File: "src.rego",
				},
				{
					Row: 7,
					Col: 1,
					Offset: len("package src\n\nviolation[msg] {\n	is_hello(msg)\n}\n\n"),
					Text: []byte("is_hello(msg) {\n	msg == \"hello\"\n}"),
					File: "src.rego",
				},
			},
		},
	}

	for n, tt := range tests {
		t.Run(n, func(t *testing.T) {
			p, err := source.NewProjectWithFiles(tt.files)
			if err != nil {
				t.Fatal(err)
			}

			location := tt.createLocation(tt.files)
			got, err := p.LookupReferences(location)
			if !errors.Is(err, tt.expectErr) {
				t.Fatalf("LookupDefinition should return error expect %v, but got %v", tt.expectErr, err)
			}

			if diff := cmp.Diff(tt.expectResult, got, cmp.Comparer(func(x, y []*ast.Location) bool {
				return reflect.DeepEqual(x, y)
			})); diff != "" {
				t.Errorf("LookupReferences result diff (-expect +got):\n%s", diff)
			}
		})
	}
}