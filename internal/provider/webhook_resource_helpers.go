package provider

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/rixlhq/terraform-provider-clickup/internal/provider/clickupcommon"
)

var errWebhookNotFound = errors.New("webhook not found in list")

func (r *webhookResource) buildCreateBody(ctx context.Context, data webhookResourceModel) ([]byte, error) {
	body := map[string]any{
		"endpoint": data.Endpoint.ValueString(),
		"events":   r.eventsAsStrings(ctx, data.Events),
	}

	if !data.SpaceId.IsNull() && !data.SpaceId.IsUnknown() {
		body["space_id"] = data.SpaceId.ValueInt64()
	}
	if !data.FolderId.IsNull() && !data.FolderId.IsUnknown() {
		body["folder_id"] = data.FolderId.ValueInt64()
	}
	if !data.ListId.IsNull() && !data.ListId.IsUnknown() {
		body["list_id"] = data.ListId.ValueInt64()
	}
	if !data.TaskId.IsNull() && !data.TaskId.IsUnknown() {
		body["task_id"] = data.TaskId.ValueString()
	}

	return json.Marshal(body)
}

func (r *webhookResource) buildUpdateBody(ctx context.Context, data webhookResourceModel) ([]byte, error) {
	// The ClickUp spec documents update "events" as a single string value.
	// Send a comma-separated string if multiple events are configured.
	events := r.eventsAsStrings(ctx, data.Events)
	eventsStr := "*"
	if len(events) > 0 {
		eventsStr = strings.Join(events, ",")
	}
	status := data.Status.ValueString()
	if status == "" {
		status = "active"
	}
	body := map[string]any{
		"endpoint": data.Endpoint.ValueString(),
		"events":   eventsStr,
		statusKey:  status,
	}
	return json.Marshal(body)
}

func (r *webhookResource) eventsAsStrings(_ context.Context, events types.List) []string {
	if events.IsNull() || events.IsUnknown() {
		return nil
	}
	var out []string
	for _, v := range events.Elements() {
		if s, ok := v.(types.String); ok {
			out = append(out, s.ValueString())
		}
	}
	return out
}

func (r *webhookResource) refresh(ctx context.Context, data *webhookResourceModel, diags *diag.Diagnostics) {
	webhook, err := r.getWebhook(ctx, data.TeamId.ValueInt64(), data.WebhookId.ValueString())
	if err != nil {
		if errors.Is(err, errWebhookNotFound) {
			data.WebhookId = types.StringNull()
			return
		}
		diags.AddError("ClickUp API Error", err.Error())
		return
	}
	r.mapWebhookToModel(ctx, webhook, data)
}

func (r *webhookResource) getWebhook(ctx context.Context, teamID int64, webhookID string) (map[string]any, error) {
	path := "/v2/team/" + strconv.FormatInt(teamID, 10) + "/webhook"
	raw, err := r.client.Get(ctx, path, nil)
	if err != nil {
		return nil, err
	}
	jv, err := clickupcommon.DecodeJSONResponse(raw)
	if err != nil {
		return nil, err
	}
	root, ok := jv.(map[string]any)
	if !ok {
		return nil, errors.New("webhook list response is not an object")
	}
	items, ok := root["webhooks"].([]any)
	if !ok {
		return nil, errors.New("webhook list response did not contain webhooks array")
	}
	for _, item := range items {
		wh, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if s, ok := wh["id"].(string); ok && s == webhookID {
			return wh, nil
		}
		if n, ok := wh["id"].(json.Number); ok && n.String() == webhookID {
			return wh, nil
		}
	}
	return nil, errWebhookNotFound
}

func (r *webhookResource) mapWebhookToModel(ctx context.Context, wh map[string]any, data *webhookResourceModel) {
	data.WebhookId = stringOrNull(wh["id"])
	data.ClientId = stringOrNull(wh["client_id"])
	data.Endpoint = stringOrNull(wh["endpoint"])
	data.Health = stringOrNull(wh["health"])
	data.Secret = stringOrNull(wh["secret"])
	data.Status = stringOrNull(wh["status"])
	data.TaskId = stringOrNull(wh["task_id"])
	data.UserId = int64OrNull(wh["userid"])
	data.TeamId = int64OrNull(wh["team_id"])
	data.SpaceId = int64OrNull(wh["space_id"])
	data.FolderId = int64OrNull(wh["folder_id"])
	data.ListId = int64OrNull(wh["list_id"])

	if events, ok := wh["events"].([]any); ok {
		data.Events = listValueFromStrings(ctx, events)
	} else if s, ok := wh["events"].(string); ok {
		data.Events = listValueFromStrings(ctx, []any{s})
	} else {
		data.Events = types.ListNull(types.StringType)
	}
}

func extractWebhookID(raw []byte) (string, error) {
	jv, err := clickupcommon.DecodeJSONResponse(raw)
	if err != nil {
		return "", err
	}
	root, ok := jv.(map[string]any)
	if !ok {
		return "", errors.New("create webhook response is not an object")
	}
	if id, ok := root["id"].(string); ok {
		return id, nil
	}
	if n, ok := root["id"].(json.Number); ok {
		return n.String(), nil
	}
	if webhook, ok := root["webhook"].(map[string]any); ok {
		if id, ok := webhook["id"].(string); ok {
			return id, nil
		}
		if n, ok := webhook["id"].(json.Number); ok {
			return n.String(), nil
		}
	}
	return "", errors.New("create webhook response did not contain an id")
}

func stringOrNull(v any) types.String {
	switch x := v.(type) {
	case string:
		return types.StringValue(x)
	case json.Number:
		return types.StringValue(x.String())
	default:
		return types.StringNull()
	}
}

func int64OrNull(v any) types.Int64 {
	switch x := v.(type) {
	case json.Number:
		i, err := x.Int64()
		if err == nil {
			return types.Int64Value(i)
		}
	case int64:
		return types.Int64Value(x)
	case int:
		return types.Int64Value(int64(x))
	case float64:
		return types.Int64Value(int64(x))
	}
	return types.Int64Null()
}

func listValueFromStrings(ctx context.Context, raw []any) types.List {
	if len(raw) == 0 {
		return types.ListNull(types.StringType)
	}
	values := make([]string, 0, len(raw))
	for _, e := range raw {
		switch x := e.(type) {
		case string:
			values = append(values, x)
		case json.Number:
			values = append(values, x.String())
		}
	}
	lv, _ := types.ListValueFrom(ctx, types.StringType, values)
	return lv
}
