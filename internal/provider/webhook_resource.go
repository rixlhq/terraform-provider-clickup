package provider

import (
	"context"
	"net/url"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/rixlhq/terraform-provider-clickup/internal/clickupclient"
)

func (r *webhookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data webhookResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, err := r.buildCreateBody(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("Request Body Error", err.Error())
		return
	}

	path := "/v2/team/" + strconv.FormatInt(data.TeamId.ValueInt64(), 10) + "/webhook"
	raw, err := r.client.Post(ctx, path, body)
	if err != nil {
		resp.Diagnostics.AddError("ClickUp API Error", err.Error())
		return
	}

	id, err := extractWebhookID(raw)
	if err != nil {
		resp.Diagnostics.AddError("Create Response Error", err.Error())
		return
	}
	data.WebhookId = types.StringValue(id)

	if !r.refresh(ctx, &data, &resp.Diagnostics) {
		if !resp.Diagnostics.HasError() {
			resp.Diagnostics.AddError("ClickUp API Error", "newly created webhook was not found in list")
		}
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *webhookResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data webhookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !r.refresh(ctx, &data, &resp.Diagnostics) {
		if !resp.Diagnostics.HasError() {
			resp.State.RemoveResource(ctx)
		}
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *webhookResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var stateData, planData webhookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Start from the prior state so the computed webhook_id used in the path is
	// known, then overlay any values the practitioner changed in the plan.
	data := stateData
	if !planData.Endpoint.IsUnknown() {
		data.Endpoint = planData.Endpoint
	}
	if !planData.Events.IsUnknown() {
		data.Events = planData.Events
	}
	if !planData.Status.IsUnknown() {
		data.Status = planData.Status
	}

	body, err := r.buildUpdateBody(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("Request Body Error", err.Error())
		return
	}

	path := "/v2/webhook/" + url.PathEscape(stateData.WebhookId.ValueString())
	if _, err := r.client.Put(ctx, path, body); err != nil {
		resp.Diagnostics.AddError("ClickUp API Error", err.Error())
		return
	}

	if !r.refresh(ctx, &data, &resp.Diagnostics) {
		if !resp.Diagnostics.HasError() {
			resp.Diagnostics.AddError("ClickUp API Error", "updated webhook was not found in list")
		}
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *webhookResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data webhookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	path := "/v2/webhook/" + url.PathEscape(data.WebhookId.ValueString())
	if _, err := r.client.Delete(ctx, path); err != nil {
		if !clickupclient.IsNotFound(err) {
			resp.Diagnostics.AddError("ClickUp API Error", err.Error())
		}
	}
}

func (r *webhookResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("Invalid ID", "expected team_id:webhook_id")
		return
	}

	teamID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid team_id", err.Error())
		return
	}

	data := webhookResourceModel{
		TeamId:    types.Int64Value(teamID),
		WebhookId: types.StringValue(parts[1]),
	}

	if !r.refresh(ctx, &data, &resp.Diagnostics) {
		if !resp.Diagnostics.HasError() {
			resp.Diagnostics.AddError("Not Found", "webhook not found")
		}
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
