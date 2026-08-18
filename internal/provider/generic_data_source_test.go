//nolint:testpackage // tests for unexported helpers
package provider

import (
	"net/url"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestAppendQueryValueListBrackets(t *testing.T) {
	query := url.Values{}
	typ := tftypes.List{ElementType: tftypes.String}
	v := tftypes.NewValue(typ, []tftypes.Value{
		tftypes.NewValue(tftypes.String, "a"),
		tftypes.NewValue(tftypes.String, "b"),
	})
	if err := appendQueryValue(query, "assignees", v, true); err != nil {
		t.Fatal(err)
	}
	got := query["assignees[]"]
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("unexpected query values: %v", got)
	}
}

func TestAppendQueryValueListNoBrackets(t *testing.T) {
	query := url.Values{}
	typ := tftypes.List{ElementType: tftypes.String}
	v := tftypes.NewValue(typ, []tftypes.Value{
		tftypes.NewValue(tftypes.String, "id1"),
		tftypes.NewValue(tftypes.String, "id2"),
	})
	if err := appendQueryValue(query, "task_ids", v, false); err != nil {
		t.Fatal(err)
	}
	got := query["task_ids"]
	if len(got) != 2 || got[0] != "id1" || got[1] != "id2" {
		t.Fatalf("unexpected query values: %v", got)
	}
}
