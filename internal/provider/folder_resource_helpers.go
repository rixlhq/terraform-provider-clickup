package provider

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/rixlhq/terraform-provider-clickup/internal/clickupclient"
	"github.com/rixlhq/terraform-provider-clickup/internal/provider/clickupcommon"
)

// buildCreateBody constructs the JSON payload for creating a ClickUp folder.
func (r *folderResource) buildCreateBody(data folderResourceModel) ([]byte, error) {
	body := map[string]any{
		"name": data.Name.ValueString(),
	}
	if !data.ParentFolderId.IsNull() && !data.ParentFolderId.IsUnknown() && data.ParentFolderId.ValueString() != "" {
		body["parent_folder_id"] = data.ParentFolderId.ValueString()
	}
	return json.Marshal(body)
}

// buildUpdateBody constructs the JSON payload for updating a ClickUp folder.
func (r *folderResource) buildUpdateBody(data folderResourceModel) ([]byte, error) {
	body := map[string]any{
		"name": data.Name.ValueString(),
	}
	return json.Marshal(body)
}

// folderParentID returns the parent folder ID from the ClickUp response.
// The API uses "parent_folder" in read responses and "parent_folder_id" in
// create/update request bodies, so both keys are checked.
func folderParentID(folder map[string]any) string {
	for _, key := range []string{"parent_folder_id", "parent_folder"} {
		switch v := folder[key].(type) {
		case string:
			return v
		case json.Number:
			return v.String()
		}
	}
	return ""
}

// refresh retrieves the current state of a folder from the ClickUp API.
func (r *folderResource) refresh(ctx context.Context, data *folderResourceModel) error {
	path := "/v2/folder/" + url.PathEscape(data.FolderId.ValueString())
	raw, err := r.client.Get(ctx, path, nil)
	if err != nil {
		if clickupclient.IsNotFound(err) {
			data.FolderId = types.StringNull()
			return nil
		}
		return err
	}
	return r.mapResponse(raw, data)
}

// mapResponse parses the API response JSON and populates the folder resource model.
func (r *folderResource) mapResponse(raw []byte, data *folderResourceModel) error {
	jv, err := clickupcommon.DecodeJSONResponse(raw)
	if err != nil {
		return err
	}
	folder, ok := jv.(map[string]any)
	if !ok {
		return errors.New("folder response is not an object")
	}
	if id, ok := folder["id"].(string); ok {
		data.FolderId = types.StringValue(id)
	} else if n, ok := folder["id"].(json.Number); ok {
		data.FolderId = types.StringValue(n.String())
	}
	data.Name = stringOrNull(folder["name"])
	data.Hidden = boolOrNull(folder["hidden"])
	data.OverrideStatuses = boolOrNull(folder["override_statuses"])
	data.Orderindex = int64OrNull(folder["orderindex"])
	data.TaskCount = int64OrNull(folder["task_count"])

	if p := folderParentID(folder); p != "" {
		data.ParentFolderId = types.StringValue(p)
	}

	if space, ok := folder["space"].(map[string]any); ok {
		data.SpaceId = stringOrNull(space["id"])
	}

	if data.FolderId.IsNull() {
		return errors.New("folder response did not contain an id")
	}
	return nil
}

// boolOrNull converts an arbitrary response value into a Terraform Bool.
func boolOrNull(v any) types.Bool {
	switch x := v.(type) {
	case bool:
		return types.BoolValue(x)
	case string:
		return types.BoolValue(x == "true")
	case json.Number:
		return types.BoolValue(x.String() != "0")
	}
	return types.BoolNull()
}
