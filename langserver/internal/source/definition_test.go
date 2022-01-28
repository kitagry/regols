package source_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kitagry/regols/langserver/internal/source"
	"github.com/open-policy-agent/opa/ast"
)

func TestLookupDefinition(t *testing.T) {
	tests := map[string]struct {
		files          map[string]source.File
		createLocation createLocationFunc
		expectResult   []*ast.Location
		expectErr      error
	}{
		"Should return variable definition in the rule": {
			files: map[string]source.File{
				"src.rego": {
					RowText: `package main

violation[msg] {
	m := "hello"
	msg = m
}`,
				},
			},
			createLocation: createLocation(5, 8, "src.rego"),
			expectResult: []*ast.Location{
				{
					Row: 4,
					Col: 2,
					Offset: len("package main\n\nviolation[msg] {\n	"),
					Text: []byte("m"),
					File: "src.rego",
				},
			},
		},
		"Should return definition in the rule's key": {
			files: map[string]source.File{
				"src.rego": {
					RowText: `package main

violation[msg] {
	m := "hello"
	msg = m
}`,
				},
			},
			createLocation: createLocation(5, 2, "src.rego"),
			expectResult: []*ast.Location{
				{
					Row:    3,
					Col:    11,
					Offset: len("package main\n\nviolation["),
					Text:   []byte("msg"),
					File:   "src.rego",
				},
			},
		},
		"Should return definition in the rule's value": {
			files: map[string]source.File{
				"src.rego": {
					RowText: `package main

test(msg) = test {
	msg == "hello"
	test = "hello"
}`,
				},
			},
			createLocation: createLocation(5, 2, "src.rego"),
			expectResult: []*ast.Location{
				{
					Row:    3,
					Col:    13,
					Offset: len("package main\n\ntest(msg) = "),
					Text:   []byte("test"),
					File:   "src.rego",
				},
			},
		},
		"Should return definition in the other file but same package": {
			files: map[string]source.File{
				"src.rego": {
					RowText: `package main

violation[msg] {
	other_method("hello")
	msg := "hello"
}`,
				},
				"src2.rego": {
					RowText: `package main

other_method(msg) {
	msg == "hello"
}`,
				},
			},
			createLocation: createLocation(4, 5, "src.rego"),
			expectResult: []*ast.Location{
				{
					Row:    3,
					Col:    1,
					Offset: len("package main\n\n"),
					Text: []byte("other_method(msg) {\n	msg == \"hello\"\n}"),
					File: "src2.rego",
				},
			},
		},
		"Should return definition in the other package": {
			files: map[string]source.File{
				"src.rego": {
					RowText: `package main

import data.lib

violation[msg] {
	lib.method("hello")
	msg := "hello"
}`,
				},
				"lib.rego": {
					RowText: `package lib

method(msg) {
	msg == "hello"
}`,
				},
			},
			createLocation: createLocation(6, 6, "src.rego"),
			expectResult: []*ast.Location{
				{
					Row:    3,
					Col:    1,
					Offset: len("package lib\n\n"),
					Text: []byte("method(msg) {\n	msg == \"hello\"\n}"),
					File: "lib.rego",
				},
			},
		},
		"Should return import sentense definition": {
			files: map[string]source.File{
				"src.rego": {
					RowText: `package main

import data.lib

violation[msg] {
	lib.method("hello")
	msg := "hello"
}`,
				},
			},
			createLocation: createLocation(6, 2, "src.rego"),
			expectResult: []*ast.Location{
				{
					Row:    3,
					Col:    13,
					Offset: len("package main\n\nimport data."),
					Text:   []byte("lib"),
					File:   "src.rego",
				},
			},
		},
		"Should not return definition when itself is definition": {
			files: map[string]source.File{
				"src.rego": {
					RowText: `package main

violation[msg] {
	m := "hello"
	msg := m
}`,
				},
			},
			createLocation: createLocation(4, 2, "src.rego"),
			expectResult:   []*ast.Location{},
			expectErr:      nil,
		},
		`Should not return definition when the item has "." but not library`: {
			files: map[string]source.File{
				"src.rego": {
					RowText: `package main

violation[msg] {
	containers[container]
	container.name
}

containers[container] {
	container := input.resource.container
}`,
				},
			},
			createLocation: createLocation(5, 12, "src.rego"),
			expectResult:   nil,
		},
		"Should return definition when the rule has else clause": {
			files: map[string]source.File{
				"src.rego": {
					RowText: `package main

authorize = "allow" {
	msg := "allow"
	trace(msg)
} else = "deny" {
	msg := "deny"
	trace(msg)
} else = "out" {
	msg := "out"
	trace(msg)
}`,
				},
			},
			createLocation: createLocation(5, 8, "src.rego"),
			expectResult: []*ast.Location{
				{
					Row: 4,
					Col: 2,
					Offset: len("package main\n\nauthorize = \"allow\" {\n	"),
					Text: []byte("msg"),
					File: "src.rego",
				},
			},
		},
		"Should return definition when the term is in the else clause": {
			files: map[string]source.File{
				"src.rego": {
					RowText: `package main

authorize = "allow" {
	msg := "allow"
	trace(msg)
} else = "deny" {
	msg := "deny"
	trace(msg)
} else = "out" {
	msg := "out"
	trace(msg)
}`,
				},
			},
			createLocation: createLocation(8, 8, "src.rego"),
			expectResult: []*ast.Location{
				{
					Row: 7,
					Col: 2,
					Offset: len("package main\n\nauthorize = \"allow\" {\n	msg := \"allow\"\n	trace(msg)\n} else = \"deny\" {\n	"),
					Text: []byte("msg"),
					File: "src.rego",
				},
			},
		},
		"Should return definition when the term is in the else of else clause": {
			files: map[string]source.File{
				"src.rego": {
					RowText: `package main

authorize = "allow" {
	msg := "allow"
	trace(msg)
} else = "deny" {
	msg := "deny"
	trace(msg)
} else = "out" {
	msg := "out"
	trace(msg)
}`,
				},
			},
			createLocation: createLocation(11, 8, "src.rego"),
			expectResult: []*ast.Location{
				{
					Row: 10,
					Col: 2,
					Offset: len("package main\n\nauthorize = \"allow\" {\n	msg := \"allow\"\n	trace(msg)\n} else = \"deny\" {\n	msg := \"deny\"\n	trace(msg)\n} else = \"out\" {\n	"),
					Text: []byte("msg"),
					File: "src.rego",
				},
			},
		},
		"Should return definition from import sentense to the library file": {
			files: map[string]source.File{
				"src.rego": {
					RowText: `package main

import data.lib`,
				},
				"lib.rego": {
					RowText: `package lib`,
				},
			},
			createLocation: createLocation(3, 13, "src.rego"),
			expectResult: []*ast.Location{
				{
					Row:    1,
					Col:    1,
					Offset: len(""),
					Text:   []byte("package"),
					File:   "lib.rego",
				},
			},
		},
	}

	for n, tt := range tests {
		t.Run(n, func(t *testing.T) {
			p, err := source.NewProjectWithFiles(tt.files)
			if err != nil {
				t.Fatalf("failed to create project: %v", err)
			}

			location := tt.createLocation(tt.files)
			got, err := p.LookupDefinition(location)
			if !errors.Is(err, tt.expectErr) {
				t.Fatalf("LookupDefinition should return error expect %v, but got %v", tt.expectErr, err)
			}

			if diff := cmp.Diff(tt.expectResult, got, cmp.Comparer(func(x, y []*ast.Location) bool {
				return reflect.DeepEqual(x, y)
			})); diff != "" {
				t.Errorf("LookupDefinition result diff (-expect +got):\n%s", diff)
			}
		})
	}
}
