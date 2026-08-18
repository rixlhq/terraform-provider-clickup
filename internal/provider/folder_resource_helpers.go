package provider

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/rixlhq/terraform-provider-clickup/internal/clickupclient"
	"github.com/rixlhq/terraform-provider-clickup/internal/provider/clickupcommon"
)

func (r *folderResource) buildCreateBody(data folderResourceModel) ([]byte, error) {
	body := map[string]any{
		"name": data.Name.ValueString(),
	}
	if !data.ParentFolderId.IsNull() && !data.ParentFolderId.IsUnknown() && data.ParentFolderId.ValueString() != "" {
		body["parent_folder_id"] = data.ParentFolderId.ValueString()
	}
	return json.Marshal(body)
}

func (r *folderResource) buildUpdateBody(data folderResourceModel) ([]byte, error) {
	body := map[string]any{
		"name": data.Name.ValueString(),
	}
	return json.Marshal(body)
}

func (r *folderResource) refresh(ctx context.Context, data *folderResourceModel) error {
	path := "/v2/folder/" + data.FolderId.ValueString()
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

	if p, ok := folder["parent_folder_id"].(string); ok && p != "" {
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
