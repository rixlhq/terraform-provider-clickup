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
		body["space_id"] = parseIntString(data.SpaceId.ValueString())
	}
	if !data.FolderId.IsNull() && !data.FolderId.IsUnknown() {
		body["folder_id"] = parseIntString(data.FolderId.ValueString())
	}
	if !data.ListId.IsNull() && !data.ListId.IsUnknown() {
		body["list_id"] = parseIntString(data.ListId.ValueString())
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

func (r *webhookResource) refresh(ctx context.Context, data *webhookResourceModel, diags *diag.Diagnostics) bool {
	webhook, err := r.getWebhook(ctx, data.TeamId.ValueInt64(), data.WebhookId.ValueString())
	if err != nil {
		if errors.Is(err, errWebhookNotFound) {
			data.WebhookId = types.StringNull()
			return false
		}
		diags.AddError("ClickUp API Error", err.Error())
		return false
	}
	r.mapWebhookToModel(ctx, webhook, data)
	return true
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
	data.Health = jsonOrNull(wh["health"])
	if h, ok := wh["health"].(map[string]any); ok {
		data.HealthStatus = stringOrNull(h["status"])
	} else {
		data.HealthStatus = types.StringNull()
	}
	data.Secret = stringOrNull(wh["secret"])
	data.Status = stringOrNull(wh["status"])
	data.TaskId = stringOrNull(wh["task_id"])
	data.UserId = int64OrNull(wh["userid"])
	data.TeamId = int64OrNull(wh["team_id"])
	data.SpaceId = stringOrNull(wh["space_id"])
	data.FolderId = stringOrNull(wh["folder_id"])
	data.ListId = stringOrNull(wh["list_id"])

	if events, ok := wh["events"].([]any); ok {
		data.Events = listValueFromStrings(ctx, events)
	} else if s, ok := wh["events"].(string); ok {
		data.Events = listValueFromStrings(ctx, []any{s})
	} else {
		data.Events = types.ListNull(types.StringType)
	}
}
