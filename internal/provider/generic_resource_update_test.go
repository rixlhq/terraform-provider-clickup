//nolint:testpackage,goconst // tests use unexported helpers and repeated strings
package provider

import (
	"context"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestGenericResourceUpdatePathMergesStateID(t *testing.T) {
	typ := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"name":     tftypes.String,
			"list_id":  tftypes.String,
			"space_id": tftypes.String,
		},
	}
	state := tftypes.NewValue(typ, map[string]tftypes.Value{
		"name":     tftypes.NewValue(tftypes.String, "Old"),
		"list_id":  tftypes.NewValue(tftypes.String, "abc123"),
		"space_id": tftypes.NewValue(tftypes.String, "456"),
	})
	plan := tftypes.NewValue(typ, map[string]tftypes.Value{
		"name":     tftypes.NewValue(tftypes.String, "New"),
		"list_id":  tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"space_id": tftypes.NewValue(tftypes.String, "456"),
	})

	merged, err := mergeTfValues(state, plan)
	if err != nil {
		t.Fatal(err)
	}

	r := &genericResource{updatePath: "/v2/list/{list_id}"}
	path, diags := r.buildPath(r.updatePath, merged)
	if diags.HasError() {
		t.Fatalf("expected path build to succeed: %s", diags)
	}
	if path != "/v2/list/abc123" {
		t.Fatalf("expected /v2/list/abc123, got %s", path)
	}

	id, diags := r.idFromValue(merged)
	if diags.HasError() {
		t.Fatalf("expected id extraction to succeed: %s", diags)
	}
	if id != "abc123" {
		t.Fatalf("expected id abc123, got %s", id)
	}
}

func TestGenericResourceUpdateBodyOmitsUnknownPlanFields(t *testing.T) {
	typ := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"name":     tftypes.String,
			"priority": tftypes.Number,
			"list_id":  tftypes.String,
		},
	}
	state := tftypes.NewValue(typ, map[string]tftypes.Value{
		"name":     tftypes.NewValue(tftypes.String, "Old"),
		"priority": tftypes.NewValue(tftypes.Number, big.NewFloat(3).SetPrec(128)),
		"list_id":  tftypes.NewValue(tftypes.String, "abc"),
	})
	plan := tftypes.NewValue(typ, map[string]tftypes.Value{
		"name":     tftypes.NewValue(tftypes.String, "New"),
		"priority": tftypes.NewValue(tftypes.Number, tftypes.UnknownValue),
		"list_id":  tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
	})

	merged, err := mergeTfValues(state, plan)
	if err != nil {
		t.Fatal(err)
	}

	r := &genericResource{}
	ctx := context.Background()
	body, diags := r.buildUpdateBody(ctx, merged, plan, []string{"name", "priority"}, nil, nil)
	if diags.HasError() {
		t.Fatalf("expected body build to succeed: %s", diags)
	}

	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatal(err)
	}
	if m["name"] != "New" {
		t.Fatalf("expected name New, got %v", m["name"])
	}
	if _, ok := m["priority"]; ok {
		t.Fatalf("expected priority to be omitted, got %v", m["priority"])
	}
}
