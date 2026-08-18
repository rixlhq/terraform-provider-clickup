package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/rixlhq/terraform-provider-clickup/internal/clickupclient"
	resource_chat_channel "github.com/rixlhq/terraform-provider-clickup/internal/provider/generated/resource_chat_channel"
)

func (r *ChatChannelResource) workspaceID(model resource_chat_channel.ChatChannelModel) (string, diag.Diagnostics) {
	var diags diag.Diagnostics
	if model.WorkspaceId.IsNull() || model.WorkspaceId.IsUnknown() {
		diags.AddError("Missing workspace_id", "workspace_id is required for chat channel operations.")
		return "", diags
	}
	return strconv.FormatInt(model.WorkspaceId.ValueInt64(), 10), diags
}

func (r *ChatChannelResource) buildCreateBody(model resource_chat_channel.ChatChannelModel) ([]byte, error) {
	body := map[string]any{
		"name": model.Name.ValueString(),
	}
	if !model.Description.IsNull() && !model.Description.IsUnknown() {
		body["description"] = model.Description.ValueString()
	}
	if !model.Topic.IsNull() && !model.Topic.IsUnknown() {
		body["topic"] = model.Topic.ValueString()
	}
	if !model.Visibility.IsNull() && !model.Visibility.IsUnknown() {
		body["visibility"] = model.Visibility.ValueString()
	}
	if !model.UserIds.IsNull() && !model.UserIds.IsUnknown() {
		var userIDs []string
		for _, elem := range model.UserIds.Elements() {
			if s, ok := elem.(types.String); ok {
				userIDs = append(userIDs, s.ValueString())
			}
		}
		body["user_ids"] = userIDs
	}
	return json.Marshal(body)
}

func (r *ChatChannelResource) buildUpdateBody(model resource_chat_channel.ChatChannelModel) ([]byte, error) {
	body := map[string]any{}
	if !model.Name.IsNull() && !model.Name.IsUnknown() {
		body["name"] = model.Name.ValueString()
	}
	if !model.Description.IsNull() && !model.Description.IsUnknown() {
		body["description"] = model.Description.ValueString()
	}
	if !model.Topic.IsNull() && !model.Topic.IsUnknown() {
		body["topic"] = model.Topic.ValueString()
	}
	if !model.Visibility.IsNull() && !model.Visibility.IsUnknown() {
		body["visibility"] = model.Visibility.ValueString()
	}
	return json.Marshal(body)
}

func (r *ChatChannelResource) parseCreatedChannelID(raw []byte) (string, error) {
	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", err
	}
	data, ok := result["data"].(map[string]any)
	if !ok {
		return "", errors.New("create response did not contain a 'data' object")
	}
	id, ok := data["id"].(string)
	if !ok {
		return "", errors.New("create response 'data.id' was not a string")
	}
	return id, nil
}

func (r *ChatChannelResource) readModel(ctx context.Context, workspaceID, channelID string) (resource_chat_channel.ChatChannelModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	path := strings.ReplaceAll(chatChannelReadPath, "{workspace_id}", workspaceID)
	path = strings.ReplaceAll(path, "{channel_id}", channelID)

	raw, err := r.client.Get(ctx, path, nil)
	if err != nil {
		if clickupclient.IsNotFound(err) {
			diags.AddWarning("Not Found", fmt.Sprintf("ClickUp API returned 404 for chat channel %s: %s", channelID, err.Error()))
			return resource_chat_channel.ChatChannelModel{}, diags
		}
		diags.AddError("ClickUp API Error", err.Error())
		return resource_chat_channel.ChatChannelModel{}, diags
	}

	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		diags.AddError("Response Decode Error", err.Error())
		return resource_chat_channel.ChatChannelModel{}, diags
	}
	data, ok := result["data"].(map[string]any)
	if !ok {
		diags.AddError("Response Decode Error", "read response did not contain a 'data' object")
		return resource_chat_channel.ChatChannelModel{}, diags
	}

	ws, err := strconv.ParseInt(workspaceID, 10, 64)
	if err != nil {
		diags.AddError("Workspace ID Error", err.Error())
		return resource_chat_channel.ChatChannelModel{}, diags
	}

	model := resource_chat_channel.ChatChannelModel{
		ChannelId:   types.StringValue(channelID),
		WorkspaceId: types.Int64Value(ws),
		Data:        resource_chat_channel.NewDataValueNull(),
	}
	model.Name = stringFromMap(data, "name")
	model.Description = stringFromMap(data, "description")
	model.Topic = stringFromMap(data, "topic")
	model.Visibility = stringFromMap(data, "visibility")

	return model, diags
}

func stringFromMap(data map[string]any, key string) types.String {
	if v, ok := data[key].(string); ok {
		return types.StringValue(v)
	}
	return types.StringNull()
}
