// Copyright 2026 J&J Ideenschmiede GmbH
// SPDX-License-Identifier: MPL-2.0

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
	_ resource.Resource                = &savedQueryResource{}
	_ resource.ResourceWithImportState = &savedQueryResource{}
)

type savedQueryResource struct {
	clientHolder
}

type savedQueryModel struct {
	ID                  types.String `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	Query               types.String `tfsdk:"query"`
	VisibleForGroupID   types.String `tfsdk:"visible_for_group_id"`
	UpdateableByGroupID types.String `tfsdk:"updateable_by_group_id"`
}

func NewSavedQueryResource() resource.Resource {
	return &savedQueryResource{}
}

func (r *savedQueryResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_saved_query"
}

func (r *savedQueryResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a saved search. The search is owned by the user of the token.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Database ID of the saved search.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the saved search.",
				Required:    true,
			},
			"query": schema.StringAttribute{
				Description: "Search query in YouTrack query syntax.",
				Required:    true,
			},
			"visible_for_group_id": schema.StringAttribute{
				Description: "Database ID of the group that can see the saved search.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updateable_by_group_id": schema.StringAttribute{
				Description: "Database ID of the group that can edit the saved search.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *savedQueryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan savedQueryModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var query client.SavedQuery
	if err := r.client.Post(ctx, "savedQueries", client.SavedQueryFields, savedQueryBody(plan), &query); err != nil {
		addAPIError(&resp.Diagnostics, "create", "saved query", err)
		return
	}

	applySavedQuery(&plan, query)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *savedQueryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state savedQueryModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var query client.SavedQuery
	err := r.client.Get(ctx, savedQueryPath(state.ID.ValueString()), client.SavedQueryFields, &query)
	if isNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		addAPIError(&resp.Diagnostics, "read", "saved query", err)
		return
	}

	applySavedQuery(&state, query)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *savedQueryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state savedQueryModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var query client.SavedQuery
	if err := r.client.Post(ctx, savedQueryPath(state.ID.ValueString()), client.SavedQueryFields, savedQueryBody(plan), &query); err != nil {
		addAPIError(&resp.Diagnostics, "update", "saved query", err)
		return
	}

	applySavedQuery(&plan, query)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *savedQueryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state savedQueryModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.Delete(ctx, savedQueryPath(state.ID.ValueString())); err != nil && !isNotFound(err) {
		addAPIError(&resp.Diagnostics, "delete", "saved query", err)
	}
}

func (r *savedQueryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func savedQueryBody(plan savedQueryModel) map[string]any {
	body := map[string]any{
		"name":  plan.Name.ValueString(),
		"query": plan.Query.ValueString(),
	}
	if ref := refOrNil(plan.VisibleForGroupID); ref != nil {
		body["visibleFor"] = ref
	}
	if ref := refOrNil(plan.UpdateableByGroupID); ref != nil {
		body["updateableBy"] = ref
	}
	return body
}

func applySavedQuery(model *savedQueryModel, query client.SavedQuery) {
	model.ID = types.StringValue(query.ID)
	model.Name = types.StringValue(query.Name)
	model.Query = types.StringValue(query.Query)
	model.VisibleForGroupID = refID(query.VisibleFor)
	model.UpdateableByGroupID = refID(query.UpdateableBy)
}

func savedQueryPath(id string) string {
	return "savedQueries/" + url.PathEscape(id)
}
