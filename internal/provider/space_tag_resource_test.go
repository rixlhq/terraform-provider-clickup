//nolint:testpackage // tests for unexported helpers
package provider

import (
	"reflect"
	"testing"
)

func TestSpaceTagUpdateColorNames(t *testing.T) {
	in := map[string]any{
		"name":   "blocked",
		"tag_fg": "#ffffff",
		"tag_bg": "#ff0000",
	}

	got := spaceTagUpdateColorNames(in)
	want := map[string]any{
		"name":     "blocked",
		"fg_color": "#ffffff",
		"bg_color": "#ff0000",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestSpaceTagUpdateColorNamesNonObject(t *testing.T) {
	if got := spaceTagUpdateColorNames("not an object"); got != "not an object" {
		t.Fatalf("expected passthrough, got %v", got)
	}
}
