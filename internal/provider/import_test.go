//nolint:testpackage // tests for unexported helpers
package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestFolderResourceImportState(t *testing.T) {
	ctx := context.Background()
	r := newFolderResource().(resource.ResourceWithImportState)

	req := resource.ImportStateRequest{
		ID: "12345",
	}
	resp := &resource.ImportStateResponse{
		State: tfsdk.State{
			Schema: schema.Schema{
				Attributes: map[string]schema.Attribute{
					"folder_id": schema.StringAttribute{
						Computed: true,
					},
				},
			},
			Raw: tftypes.NewValue(tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"folder_id": tftypes.String,
				},
			}, map[string]tftypes.Value{
				"folder_id": tftypes.NewValue(tftypes.String, nil),
			}),
		},
	}

	r.ImportState(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected import error: %v", resp.Diagnostics)
	}

	var folderID types.String
	diags := resp.State.GetAttribute(ctx, path.Root("folder_id"), &folderID)
	if diags.HasError() {
		t.Fatalf("failed to get imported attribute: %v", diags)
	}
	if folderID.ValueString() != "12345" {
		t.Fatalf("expected folder_id 12345, got %s", folderID.ValueString())
	}
}

func TestTeamUserResourceImportState(t *testing.T) {
	ctx := context.Background()
	r := newTeamUserResource().(resource.ResourceWithImportState)

	schemaObj := teamUserSchema(ctx)
	req := resource.ImportStateRequest{
		ID: "team99:user123",
	}
	resp := &resource.ImportStateResponse{
		State: tfsdk.State{
			Schema: schemaObj,
			Raw: tftypes.NewValue(schemaObj.Type().TerraformType(ctx), map[string]tftypes.Value{
				"team_id":        tftypes.NewValue(tftypes.String, nil),
				"user_id":        tftypes.NewValue(tftypes.String, nil),
				"email":          tftypes.NewValue(tftypes.String, nil),
				"username":       tftypes.NewValue(tftypes.String, nil),
				"admin":          tftypes.NewValue(tftypes.Bool, nil),
				"custom_role_id": tftypes.NewValue(tftypes.Number, nil),
			}),
		},
	}

	r.ImportState(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected import error: %v", resp.Diagnostics)
	}

	var teamID, userID types.String
	_ = resp.State.GetAttribute(ctx, path.Root("team_id"), &teamID)
	_ = resp.State.GetAttribute(ctx, path.Root("user_id"), &userID)
	if teamID.ValueString() != "team99" || userID.ValueString() != "user123" {
		t.Fatalf("expected team99:user123, got %s:%s", teamID.ValueString(), userID.ValueString())
	}
}

func TestGenericResourceImportState_SingleID(t *testing.T) {
	ctx := context.Background()
	r := &genericResource{
		name:       "task",
		createPath: "/v2/list/{list_id}/task",
		readPath:   "/v2/task/{task_id}",
		deletePath: "/v2/task/{task_id}",
		schemaFunc: func(_ context.Context) schema.Schema {
			return schema.Schema{
				Attributes: map[string]schema.Attribute{
					"id":      schema.StringAttribute{Computed: true},
					"task_id": schema.StringAttribute{Computed: true},
					"name":    schema.StringAttribute{Optional: true},
				},
			}
		},
	}

	req := resource.ImportStateRequest{ID: "task888"}
	resp := &resource.ImportStateResponse{}

	r.ImportState(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected import error: %v", resp.Diagnostics)
	}

	obj, err := asObject(resp.State.Raw)
	if err != nil {
		t.Fatalf("failed to decode raw state: %v", err)
	}

	taskID, err := valueAsString(obj["task_id"])
	if err != nil || taskID != "task888" {
		t.Fatalf("expected task_id task888, got %q (err: %v)", taskID, err)
	}
}

func TestGenericResourceImportState_CompositePath(t *testing.T) {
	ctx := context.Background()
	r := &genericResource{
		name:       "space_tag",
		createPath: "/v2/space/{space_id}/tag",
		readPath:   "/v2/space/{space_id}/tag",
		updatePath: "/v2/space/{space_id}/tag/{tag_name}",
		deletePath: "/v2/space/{space_id}/tag/{tag_name}",
		idField:    "tag_name",
		schemaFunc: func(_ context.Context) schema.Schema {
			return schema.Schema{
				Attributes: map[string]schema.Attribute{
					"space_id": schema.StringAttribute{Required: true},
					"tag_name": schema.StringAttribute{Required: true},
				},
			}
		},
	}

	req := resource.ImportStateRequest{ID: "sp100:backend"}
	resp := &resource.ImportStateResponse{}

	r.ImportState(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected import error: %v", resp.Diagnostics)
	}

	obj, err := asObject(resp.State.Raw)
	if err != nil {
		t.Fatalf("failed to decode raw state: %v", err)
	}

	spaceID, _ := valueAsString(obj["space_id"])
	tagName, _ := valueAsString(obj["tag_name"])
	if spaceID != "sp100" || tagName != "backend" {
		t.Fatalf("expected sp100:backend, got %s:%s", spaceID, tagName)
	}
}
