package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/rixlhq/terraform-provider-clickup/internal/clickupclient"
	"github.com/rixlhq/terraform-provider-clickup/internal/providerdata"
)

// --- team_guest (hand-written) ---
// The invite POST returns { "team": { "members": [...] } } where the last
// member is the newly invited guest. The generic resource cannot extract
// the ID from this nested structure.

func newTeamGuestResource() resource.Resource {
	return &teamGuestResource{}
}

type teamGuestResource struct {
	client *clickupclient.Client
}

type teamGuestModel struct {
	GuestID               types.String `tfsdk:"guest_id"`
	TeamID                types.String `tfsdk:"team_id"`
	Email                 types.String `tfsdk:"email"`
	CanEditTags           types.Bool   `tfsdk:"can_edit_tags"`
	CanSeeTimeSpent       types.Bool   `tfsdk:"can_see_time_spent"`
	CanSeeTimeEstimated   types.Bool   `tfsdk:"can_see_time_estimated"`
	CanCreateViews        types.Bool   `tfsdk:"can_create_views"`
	CanSeePointsEstimated types.Bool   `tfsdk:"can_see_points_estimated"`
	CustomRoleID          types.Int64  `tfsdk:"custom_role_id"`
}

func (r *teamGuestResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team_guest"
}

func (r *teamGuestResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = teamGuestSchema(ctx)
}

func (r *teamGuestResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	pd, ok := req.ProviderData.(*providerdata.Data)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Configure Type", fmt.Sprintf("Expected *providerdata.Data, got %T", req.ProviderData))
		return
	}
	r.client = pd.ClickUp
}

func (r *teamGuestResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Missing ClickUp Client", "Configure the provider with api_token.")
		return
	}
	var plan teamGuestModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{"email": plan.Email.ValueString()}
	if !plan.CanEditTags.IsNull() {
		body["can_edit_tags"] = plan.CanEditTags.ValueBool()
	}
	if !plan.CanSeeTimeSpent.IsNull() {
		body["can_see_time_spent"] = plan.CanSeeTimeSpent.ValueBool()
	}
	if !plan.CanSeeTimeEstimated.IsNull() {
		body["can_see_time_estimated"] = plan.CanSeeTimeEstimated.ValueBool()
	}
	if !plan.CanCreateViews.IsNull() {
		body["can_create_views"] = plan.CanCreateViews.ValueBool()
	}
	if !plan.CanSeePointsEstimated.IsNull() {
		body["can_see_points_estimated"] = plan.CanSeePointsEstimated.ValueBool()
	}
	if !plan.CustomRoleID.IsNull() {
		body["custom_role_id"] = plan.CustomRoleID.ValueInt64()
	}

	path := "/v2/team/" + plan.TeamID.ValueString() + "/guest"
	raw, err := r.client.Post(ctx, path, mustJSON(body))
	if err != nil {
		resp.Diagnostics.AddError("ClickUp API Error", err.Error())
		return
	}

	guestID, err := extractGuestIDFromTeamResponse(raw)
	if err != nil {
		resp.Diagnostics.AddError("Create Response Error", fmt.Sprintf("could not extract guest_id: %s", err))
		return
	}
	plan.GuestID = types.StringValue(guestID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *teamGuestResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state teamGuestModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Guest GET has no defined schema; return state as-is.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *teamGuestResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Missing ClickUp Client", "Configure the provider with api_token.")
		return
	}
	var plan teamGuestModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{}
	if !plan.CanEditTags.IsNull() {
		body["can_edit_tags"] = plan.CanEditTags.ValueBool()
	}
	if !plan.CanSeeTimeSpent.IsNull() {
		body["can_see_time_spent"] = plan.CanSeeTimeSpent.ValueBool()
	}
	if !plan.CanSeeTimeEstimated.IsNull() {
		body["can_see_time_estimated"] = plan.CanSeeTimeEstimated.ValueBool()
	}
	if !plan.CanCreateViews.IsNull() {
		body["can_create_views"] = plan.CanCreateViews.ValueBool()
	}
	if !plan.CanSeePointsEstimated.IsNull() {
		body["can_see_points_estimated"] = plan.CanSeePointsEstimated.ValueBool()
	}
	if !plan.CustomRoleID.IsNull() {
		body["custom_role_id"] = plan.CustomRoleID.ValueInt64()
	}

	path := "/v2/team/" + plan.TeamID.ValueString() + "/guest/" + plan.GuestID.ValueString()
	if _, err := r.client.Put(ctx, path, mustJSON(body)); err != nil {
		resp.Diagnostics.AddError("ClickUp API Error", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *teamGuestResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Missing ClickUp Client", "Configure the provider with api_token.")
		return
	}
	var state teamGuestModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	path := "/v2/team/" + state.TeamID.ValueString() + "/guest/" + state.GuestID.ValueString()
	if _, err := r.client.Delete(ctx, path); err != nil {
		if !clickupclient.IsNotFound(err) {
			resp.Diagnostics.AddError("ClickUp API Error", err.Error())
		}
	}
}
