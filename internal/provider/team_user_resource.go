package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/rixlhq/terraform-provider-clickup/internal/clickupclient"
	"github.com/rixlhq/terraform-provider-clickup/internal/providerdata"
)

// --- team_user (hand-written) ---
// Same pattern as team_guest: invite returns team.members array.

func newTeamUserResource() resource.Resource {
	return &teamUserResource{}
}

type teamUserResource struct {
	client *clickupclient.Client
}

type teamUserModel struct {
	UserID       types.String `tfsdk:"user_id"`
	TeamID       types.String `tfsdk:"team_id"`
	Email        types.String `tfsdk:"email"`
	Username     types.String `tfsdk:"username"`
	Admin        types.Bool   `tfsdk:"admin"`
	CustomRoleID types.Int64  `tfsdk:"custom_role_id"`
}

func (r *teamUserResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team_user"
}

func (r *teamUserResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = teamUserSchema(ctx)
}

func (r *teamUserResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *teamUserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Missing ClickUp Client", "Configure the provider with api_token.")
		return
	}
	var plan teamUserModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"email": plan.Email.ValueString(),
		"admin": plan.Admin.ValueBool(),
	}
	if !plan.CustomRoleID.IsNull() {
		body["custom_role_id"] = plan.CustomRoleID.ValueInt64()
	}

	path := "/v2/team/" + plan.TeamID.ValueString() + "/user"
	raw, err := r.client.Post(ctx, path, mustJSON(body))
	if err != nil {
		resp.Diagnostics.AddError("ClickUp API Error", err.Error())
		return
	}

	userID, err := extractUserIDFromTeamResponse(raw)
	if err != nil {
		resp.Diagnostics.AddError("Create Response Error", fmt.Sprintf("could not extract user_id: %s", err))
		return
	}
	plan.UserID = types.StringValue(userID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *teamUserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state teamUserModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *teamUserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Missing ClickUp Client", "Configure the provider with api_token.")
		return
	}
	var plan teamUserModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"username": plan.Username.ValueString(),
		"admin":    plan.Admin.ValueBool(),
	}
	if !plan.CustomRoleID.IsNull() {
		body["custom_role_id"] = plan.CustomRoleID.ValueInt64()
	}

	path := "/v2/team/" + plan.TeamID.ValueString() + "/user/" + plan.UserID.ValueString()
	if _, err := r.client.Put(ctx, path, mustJSON(body)); err != nil {
		resp.Diagnostics.AddError("ClickUp API Error", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *teamUserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Missing ClickUp Client", "Configure the provider with api_token.")
		return
	}
	var state teamUserModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	path := "/v2/team/" + state.TeamID.ValueString() + "/user/" + state.UserID.ValueString()
	if _, err := r.client.Delete(ctx, path); err != nil {
		if !clickupclient.IsNotFound(err) {
			resp.Diagnostics.AddError("ClickUp API Error", err.Error())
		}
	}
}

// --- helpers ---

//nolint:errchkjson // map[string]any is safe to marshal
func mustJSON(v map[string]any) []byte {
	b, _ := json.Marshal(v)
	return b
}

// extractGuestIDFromTeamResponse extracts the guest ID from the invite response.
// The response is { "team": { "members": [{ "user": { "id": 123 }, ... }] } }.
// The last member is the newly invited guest.
func extractGuestIDFromTeamResponse(raw []byte) (string, error) {
	var resp struct {
		Team struct {
			Members []struct {
				User struct {
					ID int64 `json:"id"`
				} `json:"user"`
			} `json:"members"`
		} `json:"team"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return "", fmt.Errorf("decoding response: %w", err)
	}
	if len(resp.Team.Members) == 0 {
		return "", errors.New("team.members is empty")
	}
	last := resp.Team.Members[len(resp.Team.Members)-1]
	return strconv.FormatInt(last.User.ID, 10), nil
}

// extractUserIDFromTeamResponse extracts the user ID from the invite response.
// Same structure as guest: { "team": { "members": [{ "user": { "id": 123 } }] } }.
func extractUserIDFromTeamResponse(raw []byte) (string, error) {
	return extractGuestIDFromTeamResponse(raw) // same structure
}
