package source_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kitagry/regols/langserver/internal/source"
	"github.com/kitagry/regols/langserver/internal/source/helper"
	"github.com/open-policy-agent/opa/ast"
)

func TestLookupReferences(t *testing.T) {
	tests := map[string]struct {
		files        map[string]source.File
		expectResult []*ast.Location
		expectErr    error
	}{
		"Should list self": {
			files: map[string]source.File{
				"src.rego": {
					RawText: `package src

violation[msg] {
	h|ello := "hello"
}`,
				},
			},
			expectResult: []*ast.Location{
				{
					Row:    4,
					Col:    2,
					Offset: len("package src\n\nviolation[msg] {\n	"),
					Text:   []byte("hello"),
					File:   "src.rego",
				},
			},
		},
		"Should list above item": {
			files: map[string]source.File{
				"src.rego": {
					RawText: `package src

violation[msg] {
	hello := "hello"
	is_hello(h|ello)
}`,
				},
			},
			expectResult: []*ast.Location{
				{
					Row:    4,
					Col:    2,
					Offset: len("package src\n\nviolation[msg] {\n	"),
					Text:   []byte("hello"),
					File:   "src.rego",
				},
				{
					Row:    5,
					Col:    11,
					Offset: len("package src\n\nviolation[msg] {\n	hello := \"hello\"\n	is_hello("),
					Text:   []byte("hello"),
					File:   "src.rego",
				},
			},
		},
		"Should list term which is declared in function": {
			files: map[string]source.File{
				"src.rego": {
					RawText: `package src

violation[msg] {
	containers[c|ontainer]
	msg := sprintf("%s", [container])
}

containers[container] {
	container = "c"
}`,
				},
			},
			expectResult: []*ast.Location{
				{
					Row:    4,
					Col:    13,
					Offset: len("package src\n\nviolation[msg] {\n	containers["),
					Text:   []byte("container"),
					File:   "src.rego",
				},
				{
					Row:    5,
					Col:    24,
					Offset: len("package src\n\nviolation[msg] {\n	containers[container]\n	msg := sprintf(\"%s\", ["),
					Text:   []byte("container"),
					File:   "src.rego",
				},
			},
		},
		"Should list term in array": {
			files: map[string]source.File{
				"src.rego": {
					RawText: `package src

violation[msg] {
	h|ello := "hello"
	msg := sprintf("%s", [hello])
}`,
				},
			},
			expectResult: []*ast.Location{
				{
					Row:    4,
					Col:    2,
					Offset: len("package src\n\nviolation[msg] {\n	"),
					Text:   []byte("hello"),
					File:   "src.rego",
				},
				{
					Row:    5,
					Col:    24,
					Offset: len("package src\n\nviolation[msg] {\n	hello := \"hello\"\n	msg := sprintf(\"%s\", ["),
					Text:   []byte("hello"),
					File:   "src.rego",
				},
			},
		},
		"Should list rule itself": {
			files: map[string]source.File{
				"src.rego": {
					RawText: `package src

i|s_hello(msg) {
	msg == "hello"
}`,
				},
			},
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
	trace(m|sg)
}`,
				},
			},
			expectResult: []*ast.Location{
				{
					Row:    3,
					Col:    11,
					Offset: len("package src\n\nviolation["),
					Text:   []byte("msg"),
					File:   "src.rego",
				},
				{
					Row:    4,
					Col:    8,
					Offset: len("package src\n\nviolation[msg] {\n	trace("),
					Text:   []byte("msg"),
					File:   "src.rego",
				},
			},
		},
		"Should list rule's args": {
			files: map[string]source.File{
				"src.rego": {
					RawText: `package src

violation(msg) {
	trace(m|sg)
}`,
				},
			},
			expectResult: []*ast.Location{
				{
					Row:    3,
					Col:    11,
					Offset: len("package src\n\nviolation("),
					Text:   []byte("msg"),
					File:   "src.rego",
				},
				{
					Row:    4,
					Col:    8,
					Offset: len("package src\n\nviolation(msg) {\n	trace("),
					Text:   []byte("msg"),
					File:   "src.rego",
				},
			},
		},
		"Should list rule's value": {
			files: map[string]source.File{
				"src.rego": {
					RawText: `package src

violation = msg {
	trace(m|sg)
}`,
				},
			},
			expectResult: []*ast.Location{
				{
					Row:    3,
					Col:    13,
					Offset: len("package src\n\nviolation = "),
					Text:   []byte("msg"),
					File:   "src.rego",
				},
				{
					Row:    4,
					Col:    8,
					Offset: len("package src\n\nviolation = msg {\n	trace("),
					Text:   []byte("msg"),
					File:   "src.rego",
				},
			},
		},
		"Should list function references": {
			files: map[string]source.File{
				"src.rego": {
					RawText: `package src

violation[msg] {
	i|s_hello(msg)
}

is_hello(msg) {
	msg == "hello"
}`,
				},
			},
			expectResult: []*ast.Location{
				{
					Row:    4,
					Col:    2,
					Offset: len("package src\n\nviolation[msg] {\n	"),
					Text:   []byte("is_hello"),
					File:   "src.rego",
				},
				{
					Row:    7,
					Col:    1,
					Offset: len("package src\n\nviolation[msg] {\n	is_hello(msg)\n}\n\n"),
					Text:   []byte("is_hello"),
					File:   "src.rego",
				},
			},
		},
		"Should list function references in other function": {
			files: map[string]source.File{
				"src.rego": {
					RawText: `package src

violation[msg] {
	i|s_hello(msg)
}

violation[msg] {
	is_hello(msg)
}

is_hello(msg) {
	msg == "hello"
}`,
				},
			},
			expectResult: []*ast.Location{
				{
					Row:    4,
					Col:    2,
					Offset: len("package src\n\nviolation[msg] {\n	"),
					Text:   []byte("is_hello"),
					File:   "src.rego",
				},
				{
					Row:    8,
					Col:    2,
					Offset: len("package src\n\nviolation[msg] {\n	is_hello(msg)\n}\n\nviolation[msg] {\n	"),
					Text:   []byte("is_hello"),
					File:   "src.rego",
				},
				{
					Row:    11,
					Col:    1,
					Offset: len("package src\n\nviolation[msg] {\n	is_hello(msg)\n}\n\nviolation[msg] {\n	is_hello(msg)\n}\n\n"),
					Text:   []byte("is_hello"),
					File:   "src.rego",
				},
			},
		},
		"Should list library definition": {
			files: map[string]source.File{
				"src.rego": {
					RawText: `package src

import data.lib

violation[msg] {
	lib.i|s_hello(msg)
}`,
				},
				"lib.rego": {
					RawText: `package lib

is_hello(msg) {
	msg == "hello"
}`,
				},
			},
			expectResult: []*ast.Location{
				{
					Row:    3,
					Col:    1,
					Offset: len("package lib\n\n"),
					Text:   []byte("is_hello"),
					File:   "lib.rego",
				},
				{
					Row:    6,
					Col:    6,
					Offset: len("package src\n\nimport data.lib\n\nviolation[msg] {\n	lib."),
					Text:   []byte("is_hello"),
					File:   "src.rego",
				},
			},
		},
		"Should list used in library": {
			files: map[string]source.File{
				"src.rego": {
					RawText: `package src

import data.lib

violation[msg] {
	lib.i|s_hello(msg)
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
			expectResult: []*ast.Location{
				{
					Row:    3,
					Col:    1,
					Offset: len("package lib\n\n"),
					Text:   []byte("is_hello"),
					File:   "lib.rego",
				},
				{
					Row:    8,
					Col:    2,
					Offset: len("package lib\n\nis_hello(msg) {\n	msg == \"hello\"\n}\n\nviolation[msg] {\n	"),
					Text:   []byte("is_hello"),
					File:   "lib.rego",
				},
				{
					Row:    6,
					Col:    6,
					Offset: len("package src\n\nimport data.lib\n\nviolation[msg] {\n	lib."),
					Text:   []byte("is_hello"),
					File:   "src.rego",
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
	c|ontainers[container]
	container == "a"
}

violation[msg] {
	containers[container]
	container == "b"
}`,
				},
			},
			expectResult: []*ast.Location{
				{
					Row:    3,
					Col:    1,
					Offset: len("package src\n\n"),
					Text:   []byte("containers"),
					File:   "src.rego",
				},
				{
					Row:    8,
					Col:    2,
					Offset: len("package src\n\ncontainers[container] {\n	container = input.containers[_].name\n}\n\nviolation[msg] {\n	"),
					Text:   []byte("containers"),
					File:   "src.rego",
				},
				{
					Row:    13,
					Col:    2,
					Offset: len("package src\n\ncontainers[container] {\n	container = input.containers[_].name\n}\n\nviolation[msg] {\n	containers[container]\n	container == \"a\"\n}\n\nviolation[msg] {\n	"),
					Text:   []byte("containers"),
					File:   "src.rego",
				},
			},
		},
		"Should list library function which have args": {
			files: map[string]source.File{
				"src.rego": {
					RawText: `package src

import data.lib

violation[msg] {
	lib.c|ontainers[container]
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
			expectResult: []*ast.Location{
				{
					Row:    3,
					Col:    1,
					Offset: len("package lib\n\n"),
					Text:   []byte("containers"),
					File:   "lib.rego",
				},
				{
					Row:    6,
					Col:    6,
					Offset: len("package src\n\nimport data.lib\n\nviolation[msg] {\n	lib."),
					Text:   []byte("containers"),
					File:   "src.rego",
				},
				{
					Row:    11,
					Col:    6,
					Offset: len("package src\n\nimport data.lib\n\nviolation[msg] {\n	lib.containers[container]\n	container == \"a\"\n}\n\nviolation[msg] {\n	lib."),
					Text:   []byte("containers"),
					File:   "src.rego",
				},
			},
		},
		"Should list package name": {
			files: map[string]source.File{
				"src.rego": {
					RawText: `package src

import data.lib

violation[msg] {
	l|ib.is_hello(msg)
}`,
				},
				"lib.rego": {
					RawText: `package lib

is_hello(msg) {
	msg == "hello"
}`,
				},
			},
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
					Row:    6,
					Col:    2,
					File:   "src.rego",
					Offset: len("package src\n\nimport data.lib\n\nviolation[msg] {\n	"),
					Text:   []byte("lib"),
				},
			},
		},
		"Should list package name which use alias": {
			files: map[string]source.File{
				"src.rego": {
					RawText: `package src

import data.lib as alib

violation[msg] {
	a|lib.is_hello(msg)
}`,
				},
				"lib.rego": {
					RawText: `package lib

is_hello(msg) {
	msg == "hello"
}`,
				},
			},
			expectResult: []*ast.Location{
				{
					Row:    3,
					Col:    20,
					File:   "src.rego",
					Offset: len("package src\n\nimport data.lib as "),
					Text:   []byte("alib"),
				},
				{
					Row:    6,
					Col:    2,
					File:   "src.rego",
					Offset: len("package src\n\nimport data.lib as alib\n\nviolation[msg] {\n	"),
					Text:   []byte("alib"),
				},
			},
		},
		"Should list alias function": {
			files: map[string]source.File{
				"src.rego": {
					RawText: `package src

import data.lib as alib

violation[msg] {
	alib.i|s_hello(msg)
}`,
				},
				"lib.rego": {
					RawText: `package lib

is_hello(msg) {
	msg == "hello"
}`,
				},
			},
			expectResult: []*ast.Location{
				{
					Row:    3,
					Col:    1,
					File:   "lib.rego",
					Offset: len("package lib\n\n"),
					Text:   []byte("is_hello"),
				},
				{
					Row:    6,
					Col:    7,
					File:   "src.rego",
					Offset: len("package src\n\nimport data.lib as alib\n\nviolation[msg] {\n	alib."),
					Text:   []byte("is_hello"),
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

i|s_hello(msg) {
	msg == "hello"
}`,
				},
			},
			expectResult: []*ast.Location{
				{
					Row:    3,
					Col:    1,
					File:   "lib.rego",
					Offset: len("package lib\n\n"),
					Text:   []byte("is_hello"),
				},
				{
					Row:    6,
					Col:    7,
					File:   "src.rego",
					Offset: len("package src\n\nimport data.lib as alib\n\nviolation[msg] {\n	alib."),
					Text:   []byte("is_hello"),
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

i|s_hello(msg) {
	msg == "hello"
}`,
				},
			},
			expectResult: []*ast.Location{
				{
					Row:    3,
					Col:    1,
					File:   "lib.rego",
					Offset: len("package lib\n\n"),
					Text:   []byte("is_hello"),
				},
				{
					Row:    6,
					Col:    7,
					File:   "src.rego",
					Offset: len("package src\n\nimport data.lib as alib\n\nviolation[msg] {\n	alib."),
					Text:   []byte("is_hello"),
				},
				{
					Row:    6,
					Col:    7,
					File:   "src2.rego",
					Offset: len("package src2\n\nimport data.lib as blib\n\nviolation[msg] {\n	blib."),
					Text:   []byte("is_hello"),
				},
			},
		},
		"Should list imports only in a file": {
			files: map[string]source.File{
				"src.rego": {
					RawText: `package src

import data.a.lib

f() {
	l|ib.is_hello("hello")
}`,
				},
				"lib.rego": {
					RawText: `package a.lib

is_hello(msg) {
	msg == "hello"
}`,
				},
			},
			expectResult: []*ast.Location{
				{
					Row:    1,
					Col:    11,
					Offset: len("package a."),
					Text:   []byte("lib"),
					File:   "lib.rego",
				},
				{
					Row:    3,
					Col:    15,
					Offset: len("package src\n\nimport data.a."),
					Text:   []byte("lib"),
					File:   "src.rego",
				},
				{
					Row:    6,
					Col:    2,
					Offset: len("package src\n\nimport data.a.lib\n\nf() {\n	"),
					Text:   []byte("lib"),
					File:   "src.rego",
				},
			},
		},
		"Should list references when cursor is args": {
			files: map[string]source.File{
				"src.rego": {
					RawText: `package src

func(a|) {
	trace(a)
}`,
				},
			},
			expectResult: []*ast.Location{
				{
					Row:    3,
					Col:    6,
					Offset: len("package src\n\nfunc("),
					Text:   []byte("a"),
					File:   "src.rego",
				},
				{
					Row:    4,
					Col:    8,
					Offset: len("package src\n\nfunc(a) {\n\ttrace("),
					Text:   []byte("a"),
					File:   "src.rego",
				},
			},
		},
		"Should not list references when name is same": {
			files: map[string]source.File{
				"src.rego": {
					RawText: `package src

same_name| := "hello"

func(same_name) {
	trace(same_name)
}`,
				},
			},
			expectResult: []*ast.Location{
				{
					Row:    3,
					Col:    1,
					Offset: len("package src\n\n"),
					Text:   []byte("same_name"),
					File:   "src.rego",
				},
			},
		},
		"Should not list references when name is same2": {
			files: map[string]source.File{
				"src.rego": {
					RawText: `package src

same_name| := "hello"

func {
	same_name := "hoge"
	trace(same_name)
}`,
				},
			},
			expectResult: []*ast.Location{
				{
					Row:    3,
					Col:    1,
					Offset: len("package src\n\n"),
					Text:   []byte("same_name"),
					File:   "src.rego",
				},
			},
		},
		"Should list references when used in assigned": {
			files: map[string]source.File{
				"src.rego": {
					RawText: `package src

same_name| := "hello"

func {
	other := same_name
	trace(other)
}`,
				},
			},
			expectResult: []*ast.Location{
				{
					Row:    3,
					Col:    1,
					Offset: len("package src\n\n"),
					Text:   []byte("same_name"),
					File:   "src.rego",
				},
				{
					Row:    6,
					Col:    11,
					Offset: len("package src\n\nsame_name := \"hello\"\n\nfunc {\n	other := "),
					Text:   []byte("same_name"),
					File:   "src.rego",
				},
			},
		},
		"Should not list same name function in other package": {
			files: map[string]source.File{
				"hoge.rego": {
					RawText: `package hoge

h|ello() {
	trace("hello")
}

hoge() {
	hello()
}`,
				},
				"fuga.rego": {
					RawText: `package fuga

hello() {
	trace("hello")
}

fuga() {
	hello()
}`,
				},
			},
			expectResult: []*ast.Location{
				{
					Row:    3,
					Col:    1,
					Offset: len("package hoge\n\n"),
					Text:   []byte("hello"),
					File:   "hoge.rego",
				},
				{
					Row:    8,
					Col:    2,
					Offset: len("package hoge\n\nhello() {\n	trace(\"hello\")\n}\n\nfuga() {\n	"),
					Text:   []byte("hello"),
					File:   "hoge.rego",
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

			p, err := source.NewProjectWithFiles(files)
			if err != nil {
				t.Fatal(err)
			}

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
