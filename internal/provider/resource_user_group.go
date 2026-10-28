// Copyright 2026 J&J Ideenschmiede GmbH. All rights reserved.

package provider

import (
	"context"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jjideenschmiede/terraform-provider-youtrack/internal/client"
)

var (
	_ resource.Resource                = &userGroupResource{}
	_ resource.ResourceWithImportState = &userGroupResource{}
)

type userGroupResource struct {
	clientHolder
}

type userGroupModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	RingID           types.String `tfsdk:"ring_id"`
	SuccessorGroupID types.String `tfsdk:"successor_group_id"`
}

func NewUserGroupResource() resource.Resource {
	return &userGroupResource{}
}

func (r *userGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_group"
}

func (r *userGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a YouTrack user group. Requires YouTrack 2026.1 or later.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Database ID of the group.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the group.",
				Required:    true,
			},
			"successor_group_id": schema.StringAttribute{
				Description: "Database ID of the group that takes over the permissions and visibility settings of this group when it is deleted. YouTrack requires it, so the group can only be destroyed if it is set.",
				Optional:    true,
			},
			"ring_id": schema.StringAttribute{
				Description: "ID of the group in Hub.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *userGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan userGroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var group client.UserGroup
	if err := r.client.Post(ctx, "groups", client.UserGroupFields, map[string]any{"name": plan.Name.ValueString()}, &group); err != nil {
		addAPIError(&resp.Diagnostics, "create", "user group", err)
		return
	}

	applyUserGroup(&plan, group)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *userGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state userGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var group client.UserGroup
	err := r.client.Get(ctx, groupPath(state.ID.ValueString()), client.UserGroupFields, &group)
	if isNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		addAPIError(&resp.Diagnostics, "read", "user group", err)
		return
	}

	applyUserGroup(&state, group)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *userGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state userGroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var group client.UserGroup
	if err := r.client.Post(ctx, groupPath(state.ID.ValueString()), client.UserGroupFields, map[string]any{"name": plan.Name.ValueString()}, &group); err != nil {
		addAPIError(&resp.Diagnostics, "update", "user group", err)
		return
	}

	applyUserGroup(&plan, group)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *userGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state userGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if state.SuccessorGroupID.IsNull() || state.SuccessorGroupID.ValueString() == "" {
		resp.Diagnostics.AddAttributeError(path.Root("successor_group_id"), "Missing successor group", "YouTrack only deletes a group together with a successor group. Set successor_group_id, apply, and destroy again.")
		return
	}
	query := url.Values{"successor": {state.SuccessorGroupID.ValueString()}}
	if err := r.client.DeleteWithQuery(ctx, groupPath(state.ID.ValueString()), query); err != nil && !isNotFound(err) {
		addAPIError(&resp.Diagnostics, "delete", "user group", err)
	}
}

func (r *userGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func applyUserGroup(model *userGroupModel, group client.UserGroup) {
	model.ID = types.StringValue(group.ID)
	model.Name = types.StringValue(group.Name)
	model.RingID = types.StringValue(group.RingID)
}

func groupPath(id string) string {
	return "groups/" + url.PathEscape(id)
}
