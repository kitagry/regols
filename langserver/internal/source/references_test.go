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
		"Should list term which is declared in function": {
			files: map[string]source.File{
				"src.rego": {
					RawText: `package src

violation[msg] {
	containers[container]
	msg := sprintf("%s", [container])
}

containers[container] {
	container = "c"
}`,
				},
			},
			createLocation: createLocation(4, 13, "src.rego"),
			expectResult: []*ast.Location{
				{
					Row: 4,
					Col: 13,
					Offset: len("package src\n\nviolation[msg] {\n	containers["),
					Text: []byte("container"),
					File: "src.rego",
				},
				{
					Row: 5,
					Col: 24,
					Offset: len("package src\n\nviolation[msg] {\n	containers[container]\n	msg := sprintf(\"%s\", ["),
					Text: []byte("container"),
					File: "src.rego",
				},
			},
		},
		"Should list term in array": {
			files: map[string]source.File{
				"src.rego": {
					RawText: `package src

violation[msg] {
	hello := "hello"
	msg := sprintf("%s", [hello])
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
				{
					Row: 5,
					Col: 24,
					Offset: len("package src\n\nviolation[msg] {\n	hello := \"hello\"\n	msg := sprintf(\"%s\", ["),
					Text: []byte("hello"),
					File: "src.rego",
				},
			},
		},
		"Should list rule itself": {
			files: map[string]source.File{
				"src.rego": {
					RawText: `package src

is_hello(msg) {
	msg == "hello"
}`,
				},
			},
			createLocation: createLocation(3, 1, "src.rego"),
			expectResult: []*ast.Location{
				{
					Row:    3,
					Col:    1,
					Offset: len("package src\n\n"),
					Text:   []byte("is_hello"),
					File:   "src.rego",
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
					Text: []byte("is_hello"),
					File: "src.rego",
				},
			},
		},
		"Should list function references in other function": {
			files: map[string]source.File{
				"src.rego": {
					RawText: `package src

violation[msg] {
	is_hello(msg)
}

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
					Row: 8,
					Col: 2,
					Offset: len("package src\n\nviolation[msg] {\n	is_hello(msg)\n}\n\nviolation[msg] {\n	"),
					Text: []byte("is_hello"),
					File: "src.rego",
				},
				{
					Row: 11,
					Col: 1,
					Offset: len("package src\n\nviolation[msg] {\n	is_hello(msg)\n}\n\nviolation[msg] {\n	is_hello(msg)\n}\n\n"),
					Text: []byte("is_hello"),
					File: "src.rego",
				},
			},
		},
		"Should list library definition": {
			files: map[string]source.File{
				"src.rego": {
					RawText: `package src

import data.lib

violation[msg] {
	lib.is_hello(msg)
}`,
				},
				"lib.rego": {
					RawText: `package lib

is_hello(msg) {
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
					Text:   []byte("is_hello"),
					File:   "lib.rego",
				},
				{
					Row: 6,
					Col: 6,
					Offset: len("package src\n\nimport data.lib\n\nviolation[msg] {\n	lib."),
					Text: []byte("is_hello"),
					File: "src.rego",
				},
			},
		},
		"Should list used in library": {
			files: map[string]source.File{
				"src.rego": {
					RawText: `package src

import data.lib

violation[msg] {
	lib.is_hello(msg)
}`,
				},
				"lib.rego": {
					RawText: `package lib

is_hello(msg) {
	msg == "hello"
}

violation[msg] {
	is_hello(msg)
}`,
				},
			},
			createLocation: createLocation(6, 6, "src.rego"),
			expectResult: []*ast.Location{
				{
					Row:    3,
					Col:    1,
					Offset: len("package lib\n\n"),
					Text:   []byte("is_hello"),
					File:   "lib.rego",
				},
				{
					Row: 8,
					Col: 2,
					Offset: len("package lib\n\nis_hello(msg) {\n	msg == \"hello\"\n}\n\nviolation[msg] {\n	"),
					Text: []byte("is_hello"),
					File: "lib.rego",
				},
				{
					Row: 6,
					Col: 6,
					Offset: len("package src\n\nimport data.lib\n\nviolation[msg] {\n	lib."),
					Text: []byte("is_hello"),
					File: "src.rego",
				},
			},
		},
		"Should list function which have args": {
			files: map[string]source.File{
				"src.rego": {
					RawText: `package src

containers[container] {
	container = input.containers[_].name
}

violation[msg] {
	containers[container]
	container == "a"
}

violation[msg] {
	containers[container]
	container == "b"
}`,
				},
			},
			createLocation: createLocation(8, 2, "src.rego"),
			expectResult: []*ast.Location{
				{
					Row:    3,
					Col:    1,
					Offset: len("package src\n\n"),
					Text:   []byte("containers"),
					File:   "src.rego",
				},
				{
					Row: 8,
					Col: 2,
					Offset: len("package src\n\ncontainers[container] {\n	container = input.containers[_].name\n}\n\nviolation[msg] {\n	"),
					Text: []byte("containers"),
					File: "src.rego",
				},
				{
					Row: 13,
					Col: 2,
					Offset: len("package src\n\ncontainers[container] {\n	container = input.containers[_].name\n}\n\nviolation[msg] {\n	containers[container]\n	container == \"a\"\n}\n\nviolation[msg] {\n	"),
					Text: []byte("containers"),
					File: "src.rego",
				},
			},
		},
		"Should list library function which have args": {
			files: map[string]source.File{
				"src.rego": {
					RawText: `package src

import data.lib

violation[msg] {
	lib.containers[container]
	container == "a"
}

violation[msg] {
	lib.containers[container]
	container == "b"
}`,
				},
				"lib.rego": {
					RawText: `package lib

containers[container] {
	container = input.containers[_].name
}`,
				},
			},
			createLocation: createLocation(6, 6, "src.rego"),
			expectResult: []*ast.Location{
				{
					Row:    3,
					Col:    1,
					Offset: len("package lib\n\n"),
					Text:   []byte("containers"),
					File:   "lib.rego",
				},
				{
					Row: 6,
					Col: 6,
					Offset: len("package src\n\nimport data.lib\n\nviolation[msg] {\n	lib."),
					Text: []byte("containers"),
					File: "src.rego",
				},
				{
					Row: 11,
					Col: 6,
					Offset: len("package src\n\nimport data.lib\n\nviolation[msg] {\n	lib.containers[container]\n	container == \"a\"\n}\n\nviolation[msg] {\n	lib."),
					Text: []byte("containers"),
					File: "src.rego",
				},
			},
		},
		"Should list package name": {
			files: map[string]source.File{
				"src.rego": {
					RawText: `package src

import data.lib

violation[msg] {
	lib.is_hello(msg)
}`,
				},
				"lib.rego": {
					RawText: `package lib

is_hello(msg) {
	msg == "hello"
}`,
				},
			},
			createLocation: createLocation(6, 2, "src.rego"),
			expectResult: []*ast.Location{
				{
					Row:    1,
					Col:    9,
					File:   "lib.rego",
					Offset: len("package "),
					Text:   []byte("lib"),
				},
				{
					Row:    3,
					Col:    13,
					File:   "src.rego",
					Offset: len("package src\n\nimport data."),
					Text:   []byte("lib"),
				},
				{
					Row:  6,
					Col:  2,
					File: "src.rego",
					Offset: len("package src\n\nimport data.lib\n\nviolation[msg] {\n	"),
					Text: []byte("lib"),
				},
			},
		},
		"Should list package name which use alias": {
			files: map[string]source.File{
				"src.rego": {
					RawText: `package src

import data.lib as alib

violation[msg] {
	alib.is_hello(msg)
}`,
				},
				"lib.rego": {
					RawText: `package lib

is_hello(msg) {
	msg == "hello"
}`,
				},
			},
			createLocation: createLocation(6, 2, "src.rego"),
			expectResult: []*ast.Location{
				{
					Row:    3,
					Col:    20,
					File:   "src.rego",
					Offset: len("package src\n\nimport data.lib as "),
					Text:   []byte("alib"),
				},
				{
					Row:  6,
					Col:  2,
					File: "src.rego",
					Offset: len("package src\n\nimport data.lib as alib\n\nviolation[msg] {\n	"),
					Text: []byte("alib"),
				},
			},
		},
		"Should list alias function": {
			files: map[string]source.File{
				"src.rego": {
					RawText: `package src

import data.lib as alib

violation[msg] {
	alib.is_hello(msg)
}`,
				},
				"lib.rego": {
					RawText: `package lib

is_hello(msg) {
	msg == "hello"
}`,
				},
			},
			createLocation: createLocation(6, 7, "src.rego"),
			expectResult: []*ast.Location{
				{
					Row:    3,
					Col:    1,
					File:   "lib.rego",
					Offset: len("package lib\n\n"),
					Text:   []byte("is_hello"),
				},
				{
					Row:  6,
					Col:  7,
					File: "src.rego",
					Offset: len("package src\n\nimport data.lib as alib\n\nviolation[msg] {\n	alib."),
					Text: []byte("is_hello"),
				},
			},
		},
		"Should list alias function for not source file": {
			files: map[string]source.File{
				"src.rego": {
					RawText: `package src

import data.lib as alib

violation[msg] {
	alib.is_hello(msg)
}`,
				},
				"lib.rego": {
					RawText: `package lib

is_hello(msg) {
	msg == "hello"
}`,
				},
			},
			createLocation: createLocation(3, 1, "lib.rego"),
			expectResult: []*ast.Location{
				{
					Row:    3,
					Col:    1,
					File:   "lib.rego",
					Offset: len("package lib\n\n"),
					Text:   []byte("is_hello"),
				},
				{
					Row:  6,
					Col:  7,
					File: "src.rego",
					Offset: len("package src\n\nimport data.lib as alib\n\nviolation[msg] {\n	alib."),
					Text: []byte("is_hello"),
				},
			},
		},
		"Should list different alias function": {
			files: map[string]source.File{
				"src.rego": {
					RawText: `package src

import data.lib as alib

violation[msg] {
	alib.is_hello(msg)
}`,
				},
				"src2.rego": {
					RawText: `package src2

import data.lib as blib

violation[msg] {
	blib.is_hello(msg)
}`,
				},
				"lib.rego": {
					RawText: `package lib

is_hello(msg) {
	msg == "hello"
}`,
				},
			},
			createLocation: createLocation(3, 1, "lib.rego"),
			expectResult: []*ast.Location{
				{
					Row:    3,
					Col:    1,
					File:   "lib.rego",
					Offset: len("package lib\n\n"),
					Text:   []byte("is_hello"),
				},
				{
					Row:  6,
					Col:  7,
					File: "src.rego",
					Offset: len("package src\n\nimport data.lib as alib\n\nviolation[msg] {\n	alib."),
					Text: []byte("is_hello"),
				},
				{
					Row:  6,
					Col:  7,
					File: "src2.rego",
					Offset: len("package src2\n\nimport data.lib as blib\n\nviolation[msg] {\n	blib."),
					Text: []byte("is_hello"),
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
