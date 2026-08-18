//nolint:testpackage // tests for unexported helpers
package clickupcommon

import (
	"context"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestJSONToTfListEmpty(t *testing.T) {
	ctx := context.Background()
	typ := tftypes.List{ElementType: tftypes.String}
	got, err := jsonToTfValue(ctx, typ, []any{})
	if err != nil {
		t.Fatal(err)
	}
	if got.IsNull() {
		t.Fatal("empty list became null")
	}
	var out []tftypes.Value
	if err := got.As(&out); err != nil {
		t.Fatal(err)
	}
	if len(out) != 0 {
		t.Fatalf("expected 0 elements, got %d", len(out))
	}
}

func TestJSONToTfValueStringCoercesNumber(t *testing.T) {
	ctx := context.Background()
	got, err := jsonToTfValue(ctx, tftypes.String, json.Number("123"))
	if err != nil {
		t.Fatal(err)
	}
	var s string
	if err := got.As(&s); err != nil {
		t.Fatal(err)
	}
	if s != "123" {
		t.Fatalf("expected \"123\", got %q", s)
	}
}

func TestJSONToTfValueStringRejectsObject(t *testing.T) {
	ctx := context.Background()
	_, err := jsonToTfValue(ctx, tftypes.String, map[string]any{"foo": "bar"})
	if err == nil {
		t.Fatal("expected error for object-to-string conversion")
	}
}

func TestJSONToTfValueNumberCoercesString(t *testing.T) {
	ctx := context.Background()
	got, err := jsonToTfValue(ctx, tftypes.Number, "456")
	if err != nil {
		t.Fatal(err)
	}
	var n big.Float
	if err := got.As(&n); err != nil {
		t.Fatal(err)
	}
	if n.Cmp(big.NewFloat(456)) != 0 {
		t.Fatalf("expected 456, got %v", &n)
	}
}

func TestJSONToTfValueNumberRejectsInvalidString(t *testing.T) {
	ctx := context.Background()
	_, err := jsonToTfValue(ctx, tftypes.Number, "not-a-number")
	if err == nil {
		t.Fatal("expected error for non-numeric string")
	}
}

func TestToString(t *testing.T) {
	if s, ok := toString("hello"); !ok || s != "hello" {
		t.Fatalf("unexpected toString result: %q %v", s, ok)
	}
	if s, ok := toString(json.Number("42")); !ok || s != "42" {
		t.Fatalf("unexpected toString number result: %q %v", s, ok)
	}
	if _, ok := toString(map[string]any{}); ok {
		t.Fatal("expected toString to reject object")
	}
}

func TestToBigFloat(t *testing.T) {
	if n, ok := toBigFloat(json.Number("3.14")); !ok || n == nil {
		t.Fatalf("unexpected toBigFloat result: %v %v", n, ok)
	}
	if n, ok := toBigFloat("2.718"); !ok || n == nil {
		t.Fatalf("unexpected toBigFloat string result: %v %v", n, ok)
	}
	if _, ok := toBigFloat("abc"); ok {
		t.Fatal("expected toBigFloat to reject non-numeric string")
	}
	if _, ok := toBigFloat([]any{}); ok {
		t.Fatal("expected toBigFloat to reject list")
	}
}

func TestJSONToTfListAllNull(t *testing.T) {
	ctx := context.Background()
	typ := tftypes.List{ElementType: tftypes.Object{AttributeTypes: map[string]tftypes.Type{"x": tftypes.String}}}
	got, err := jsonToTfValue(ctx, typ, []any{nil, nil})
	if err != nil {
		t.Fatal(err)
	}
	if got.IsNull() {
		t.Fatal("all-null list became null")
	}
	var out []tftypes.Value
	if err := got.As(&out); err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(out))
	}
}
