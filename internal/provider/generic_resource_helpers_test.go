//nolint:testpackage // tests for unexported helpers
package provider

import (
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
