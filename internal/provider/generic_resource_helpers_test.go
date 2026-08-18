//nolint:testpackage // tests for unexported helpers
package provider

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestMergeTfValuesEmptyList(t *testing.T) {
	typ := tftypes.List{ElementType: tftypes.String}
	base := tftypes.NewValue(typ, nil)
	latest := tftypes.NewValue(typ, []tftypes.Value{})
	got, err := mergeTfValues(base, latest)
	if err != nil {
		t.Fatal(err)
	}
	if got.IsNull() {
		t.Fatal("empty latest list became null")
	}
	var out []tftypes.Value
	if err := got.As(&out); err != nil {
		t.Fatal(err)
	}
	if len(out) != 0 {
		t.Fatalf("expected 0 elements, got %d", len(out))
	}
}

func TestMergeTfValuesAllNullList(t *testing.T) {
	typ := tftypes.List{ElementType: tftypes.String}
	base := tftypes.NewValue(typ, []tftypes.Value{tftypes.NewValue(tftypes.String, "x")})
	latest := tftypes.NewValue(typ, []tftypes.Value{tftypes.NewValue(tftypes.String, nil), tftypes.NewValue(tftypes.String, nil)})
	got, err := mergeTfValues(base, latest)
	if err != nil {
		t.Fatal(err)
	}
	if got.IsNull() {
		t.Fatal("result became null")
	}
	var out []tftypes.Value
	if err := got.As(&out); err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 element, got %d", len(out))
	}
	var s string
	if err := out[0].As(&s); err != nil {
		t.Fatal(err)
	}
	if s != "x" {
		t.Fatalf("expected x, got %s", s)
	}
}

func TestUserObjectsToIntList(t *testing.T) {
	got := userObjectsToIntList([]any{
		map[string]any{"id": json.Number("1"), "name": "Alice"},
		map[string]any{"user_id": json.Number("2"), "name": "Bob"},
		"3",
	})
	want := []any{json.Number("1"), json.Number("2"), "3"}
	if len(got.([]any)) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if fmt.Sprint(got.([]any)[i]) != fmt.Sprint(want[i]) {
			t.Fatalf("at %d: expected %v, got %v", i, want[i], got.([]any)[i])
		}
	}
}

func TestUserObjectsToIntListNumeric(t *testing.T) {
	got := userObjectsToIntList([]any{json.Number("42"), "43"})
	want := []any{json.Number("42"), "43"}
	if len(got.([]any)) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}
