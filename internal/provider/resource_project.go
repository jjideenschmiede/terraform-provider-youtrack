// Copyright 2026 J&J Ideenschmiede GmbH. All rights reserved.

package provider

import (
	"context"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jjideenschmiede/terraform-provider-youtrack/internal/client"
)

var (
	_ resource.Resource                = &projectResource{}
	_ resource.ResourceWithImportState = &projectResource{}
)

type projectResource struct {
	clientHolder
}

type projectModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	ShortName       types.String `tfsdk:"short_name"`
	LeaderID        types.String `tfsdk:"leader_id"`
	Description     types.String `tfsdk:"description"`
	Archived        types.Bool   `tfsdk:"archived"`
	FromEmail       types.String `tfsdk:"from_email"`
	ReplyToEmail    types.String `tfsdk:"reply_to_email"`
	StartingNumber  types.Int64  `tfsdk:"starting_number"`
	ProjectTemplate types.String `tfsdk:"project_template"`
}

func NewProjectResource() resource.Resource {
	return &projectResource{}
}

func (r *projectResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (r *projectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a YouTrack project. Destroying this resource deletes the project including all of its issues.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Database ID of the project.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the project.",
				Required:    true,
			},
			"short_name": schema.StringAttribute{
				Description: "Short name (ID) of the project, used as prefix for issue IDs.",
				Required:    true,
			},
			"leader_id": schema.StringAttribute{
				Description: "Database ID of the user who owns the project.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Description shown on the project overview.",
				Optional:    true,
			},
			"archived": schema.BoolAttribute{
				Description: "Whether the project is archived.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"from_email": schema.StringAttribute{
				Description: "Sender address for notifications of this project. Uses the server default if not set.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"reply_to_email": schema.StringAttribute{
				Description: "Reply-to address for notifications of this project.",
				Optional:    true,
			},
			"starting_number": schema.Int64Attribute{
				Description: "Number of the first issue. Only used when the project is created, later changes are ignored.",
				Optional:    true,
			},
			"project_template": schema.StringAttribute{
				Description: "Template the project is created from, either `scrum` or `kanban`. Only used when the project is created, later changes are ignored.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("scrum", "kanban"),
				},
			},
		},
	}
}

func (r *projectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan projectModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := projectBody(plan)
	setIfKnown(body, "startingNumber", plan.StartingNumber)
	query := url.Values{}
	if !plan.ProjectTemplate.IsNull() {
		query.Set("template", plan.ProjectTemplate.ValueString())
	}

	var project client.Project
	if err := r.client.PostWithQuery(ctx, "admin/projects", client.ProjectFields, query, body, &project); err != nil {
		addAPIError(&resp.Diagnostics, "create", "project", err)
		return
	}

	plan.ID = types.StringValue(project.ID)
	if plan.Archived.ValueBool() != project.Archived {
		if err := r.client.Post(ctx, projectPath(project.ID), client.ProjectFields, map[string]any{"archived": plan.Archived.ValueBool()}, &project); err != nil {
			addAPIError(&resp.Diagnostics, "archive", "project", err)
		}
	}
	applyProject(&plan, project)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *projectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state projectModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var project client.Project
	err := r.client.Get(ctx, projectPath(state.ID.ValueString()), client.ProjectFields, &project)
	if isNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		addAPIError(&resp.Diagnostics, "read", "project", err)
		return
	}

	applyProject(&state, project)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *projectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state projectModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := projectBody(plan)
	body["archived"] = plan.Archived.ValueBool()
	if plan.Description.IsNull() && !state.Description.IsNull() {
		body["description"] = ""
	}
	if plan.ReplyToEmail.IsNull() && !state.ReplyToEmail.IsNull() {
		body["replyToEmail"] = nil
	}

	var project client.Project
	if err := r.client.Post(ctx, projectPath(state.ID.ValueString()), client.ProjectFields, body, &project); err != nil {
		addAPIError(&resp.Diagnostics, "update", "project", err)
		return
	}

	plan.ID = state.ID
	applyProject(&plan, project)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *projectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state projectModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.Delete(ctx, projectPath(state.ID.ValueString())); err != nil && !isNotFound(err) {
		addAPIError(&resp.Diagnostics, "delete", "project", err)
	}
}

func (r *projectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func projectBody(plan projectModel) map[string]any {
	body := map[string]any{
		"name":      plan.Name.ValueString(),
		"shortName": plan.ShortName.ValueString(),
		"leader":    client.Ref{ID: plan.LeaderID.ValueString()},
	}
	setIfKnown(body, "description", plan.Description)
	setIfKnown(body, "fromEmail", plan.FromEmail)
	setIfKnown(body, "replyToEmail", plan.ReplyToEmail)
	return body
}

func applyProject(model *projectModel, project client.Project) {
	model.ID = types.StringValue(project.ID)
	model.Name = types.StringValue(project.Name)
	model.ShortName = types.StringValue(project.ShortName)
	model.LeaderID = refID(project.Leader)
	model.Description = optionalString(project.Description)
	model.Archived = types.BoolValue(project.Archived)
	model.FromEmail = types.StringValue(project.FromEmail)
	model.ReplyToEmail = optionalString(project.ReplyToEmail)
}

func projectPath(id string) string {
	return "admin/projects/" + url.PathEscape(id)
}
