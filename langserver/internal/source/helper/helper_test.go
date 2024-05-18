package helper_test

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kitagry/regols/langserver/internal/source"
	"github.com/kitagry/regols/langserver/internal/source/helper"
	"github.com/open-policy-agent/opa/ast"
)

func TestGetAstLocation(t *testing.T) {
	tests := map[string]struct {
		files            map[string]source.File
		expectedFiles    map[string]source.File
		expectedLocation *ast.Location
		expectedErr      error
	}{
		"position exists": {
			files:         map[string]source.File{"a.rego": {RawText: "package src|"}},
			expectedFiles: map[string]source.File{"a.rego": {RawText: "package src"}}, // remove "|"
			expectedLocation: &ast.Location{
				File:   "a.rego",
				Row:    1,
				Col:    12,
				Text:   []byte("package src"),
				Offset: 12,
			},
		},
		"position exists in multiline": {
			files: map[string]source.File{"a.rego": {RawText: `package src

import data.lib|`}},
			expectedFiles: map[string]source.File{"a.rego": {RawText: `package src

import data.lib`}}, // remove "|"
			expectedLocation: &ast.Location{
				File:   "a.rego",
				Row:    3,
				Col:    16,
				Text:   []byte("package src\n\nimport data.lib"),
				Offset: 28,
			},
		},
		"no position": {
			files:            map[string]source.File{"a.rego": {RawText: "package src"}},
			expectedFiles:    nil,
			expectedLocation: nil,
			expectedErr:      helper.ErrNoPosition,
		},
	}

	for n, tt := range tests {
		t.Run(n, func(t *testing.T) {
			gotFiles, gotLocation, gotErr := helper.GetAstLocation(tt.files)
			if diff := cmp.Diff(gotFiles, tt.expectedFiles); diff != "" {
				t.Errorf("files mismatch (-got +want):\n%s", diff)
			}

			if tt.expectedLocation != nil {
				if diff := cmp.Diff(gotLocation, tt.expectedLocation); diff != "" {
					t.Errorf("location mismatch (-got +want):\n%s", diff)
				}
			} else {
				if gotLocation != nil {
					t.Errorf("location should be nil, but got %v", gotLocation)
				}
			}

			if !errors.Is(gotErr, tt.expectedErr) {
				t.Errorf("error mismatch: got %v, want %v", gotErr, tt.expectedErr)
			}
		})
	}
}
