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
	_ resource.Resource                = &issueLinkTypeResource{}
	_ resource.ResourceWithImportState = &issueLinkTypeResource{}
)

type issueLinkTypeResource struct {
	clientHolder
}

type issueLinkTypeModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	SourceToTarget types.String `tfsdk:"source_to_target"`
	TargetToSource types.String `tfsdk:"target_to_source"`
	Directed       types.Bool   `tfsdk:"directed"`
	Aggregation    types.Bool   `tfsdk:"aggregation"`
}

func NewIssueLinkTypeResource() resource.Resource {
	return &issueLinkTypeResource{}
}

func (r *issueLinkTypeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_issue_link_type"
}

func (r *issueLinkTypeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an issue link type.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Database ID of the link type.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the link type.",
				Required:    true,
			},
			"source_to_target": schema.StringAttribute{
				Description: "Outward name of the link, for example `subtask of`.",
				Required:    true,
			},
			"target_to_source": schema.StringAttribute{
				Description: "Inward name of the link, for example `parent for`. Ignored by YouTrack for undirected links.",
				Required:    true,
			},
			"directed": schema.BoolAttribute{
				Description: "Whether the link is directed.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"aggregation": schema.BoolAttribute{
				Description: "Whether the link expresses an aggregation, which is used for issue hierarchies.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
		},
	}
}

func (r *issueLinkTypeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan issueLinkTypeModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var linkType client.IssueLinkType
	if err := r.client.Post(ctx, "issueLinkTypes", client.IssueLinkTypeFields, issueLinkTypeBody(plan), &linkType); err != nil {
		addAPIError(&resp.Diagnostics, "create", "issue link type", err)
		return
	}

	applyIssueLinkType(&plan, linkType)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *issueLinkTypeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state issueLinkTypeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var linkType client.IssueLinkType
	err := r.client.Get(ctx, issueLinkTypePath(state.ID.ValueString()), client.IssueLinkTypeFields, &linkType)
	if isNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		addAPIError(&resp.Diagnostics, "read", "issue link type", err)
		return
	}

	applyIssueLinkType(&state, linkType)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *issueLinkTypeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state issueLinkTypeModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var linkType client.IssueLinkType
	if err := r.client.Post(ctx, issueLinkTypePath(state.ID.ValueString()), client.IssueLinkTypeFields, issueLinkTypeBody(plan), &linkType); err != nil {
		addAPIError(&resp.Diagnostics, "update", "issue link type", err)
		return
	}

	applyIssueLinkType(&plan, linkType)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *issueLinkTypeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state issueLinkTypeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.Delete(ctx, issueLinkTypePath(state.ID.ValueString())); err != nil && !isNotFound(err) {
		addAPIError(&resp.Diagnostics, "delete", "issue link type", err)
	}
}

func (r *issueLinkTypeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func issueLinkTypeBody(plan issueLinkTypeModel) map[string]any {
	return map[string]any{
		"name":           plan.Name.ValueString(),
		"sourceToTarget": plan.SourceToTarget.ValueString(),
		"targetToSource": plan.TargetToSource.ValueString(),
		"directed":       plan.Directed.ValueBool(),
		"aggregation":    plan.Aggregation.ValueBool(),
	}
}

func applyIssueLinkType(model *issueLinkTypeModel, linkType client.IssueLinkType) {
	model.ID = types.StringValue(linkType.ID)
	model.Name = types.StringValue(linkType.Name)
	model.SourceToTarget = types.StringValue(linkType.SourceToTarget)
	if linkType.Directed || linkType.TargetToSource != "" {
		model.TargetToSource = types.StringValue(linkType.TargetToSource)
	}
	model.Directed = types.BoolValue(linkType.Directed)
	model.Aggregation = types.BoolValue(linkType.Aggregation)
}

func issueLinkTypePath(id string) string {
	return "issueLinkTypes/" + url.PathEscape(id)
}
