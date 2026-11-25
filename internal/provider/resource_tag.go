// Copyright 2026 J&J Ideenschmiede GmbH
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jjideenschmiede/terraform-provider-youtrack/internal/client"
)

var (
	_ resource.Resource                = &tagResource{}
	_ resource.ResourceWithImportState = &tagResource{}
)

type tagResource struct {
	clientHolder
}

type tagModel struct {
	ID                  types.String `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	UntagOnResolve      types.Bool   `tfsdk:"untag_on_resolve"`
	ColorID             types.String `tfsdk:"color_id"`
	VisibleForGroupID   types.String `tfsdk:"visible_for_group_id"`
	UpdateableByGroupID types.String `tfsdk:"updateable_by_group_id"`
}

func NewTagResource() resource.Resource {
	return &tagResource{}
}

func (r *tagResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tag"
}

func (r *tagResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an issue tag. The tag is owned by the user of the token.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Database ID of the tag.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the tag.",
				Required:    true,
			},
			"untag_on_resolve": schema.BoolAttribute{
				Description: "Whether the tag is removed when an issue is resolved.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"color_id": schema.StringAttribute{
				Description: "Index of the color in the YouTrack palette, from `0` to `35`.",
				Optional:    true,
				Computed:    true,
				Validators:  colorValidators(),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"visible_for_group_id": schema.StringAttribute{
				Description: "Database ID of the group that can see the tag.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updateable_by_group_id": schema.StringAttribute{
				Description: "Database ID of the group that can edit the tag.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *tagResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan tagModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var tag client.Tag
	if err := r.client.Post(ctx, "tags", client.TagFields, tagBody(plan), &tag); err != nil {
		addAPIError(&resp.Diagnostics, "create", "tag", err)
		return
	}

	applyTag(&plan, tag)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *tagResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state tagModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var tag client.Tag
	err := r.client.Get(ctx, tagPath(state.ID.ValueString()), client.TagFields, &tag)
	if isNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		addAPIError(&resp.Diagnostics, "read", "tag", err)
		return
	}

	applyTag(&state, tag)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *tagResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state tagModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var tag client.Tag
	if err := r.client.Post(ctx, tagPath(state.ID.ValueString()), client.TagFields, tagBody(plan), &tag); err != nil {
		addAPIError(&resp.Diagnostics, "update", "tag", err)
		return
	}

	applyTag(&plan, tag)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *tagResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state tagModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.Delete(ctx, tagPath(state.ID.ValueString())); err != nil && !isNotFound(err) {
		addAPIError(&resp.Diagnostics, "delete", "tag", err)
	}
}

func (r *tagResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func tagBody(plan tagModel) map[string]any {
	body := map[string]any{
		"name":           plan.Name.ValueString(),
		"untagOnResolve": plan.UntagOnResolve.ValueBool(),
	}
	if ref := refOrNil(plan.ColorID); ref != nil {
		body["color"] = ref
	}
	if ref := refOrNil(plan.VisibleForGroupID); ref != nil {
		body["visibleFor"] = ref
	}
	if ref := refOrNil(plan.UpdateableByGroupID); ref != nil {
		body["updateableBy"] = ref
	}
	return body
}

func applyTag(model *tagModel, tag client.Tag) {
	model.ID = types.StringValue(tag.ID)
	model.Name = types.StringValue(tag.Name)
	model.UntagOnResolve = types.BoolValue(tag.UntagOnResolve)
	model.ColorID = colorID(tag.Color)
	model.VisibleForGroupID = refID(tag.VisibleFor)
	model.UpdateableByGroupID = refID(tag.UpdateableBy)
}

func tagPath(id string) string {
	return "tags/" + url.PathEscape(id)
}
