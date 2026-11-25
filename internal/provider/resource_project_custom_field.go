// Copyright 2026 J&J Ideenschmiede GmbH
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jjideenschmiede/terraform-provider-youtrack/internal/client"
)

var (
	_ resource.Resource                = &projectCustomFieldResource{}
	_ resource.ResourceWithImportState = &projectCustomFieldResource{}
)

type projectCustomFieldResource struct {
	clientHolder
}

type projectCustomFieldModel struct {
	ID             types.String `tfsdk:"id"`
	ProjectID      types.String `tfsdk:"project_id"`
	FieldID        types.String `tfsdk:"field_id"`
	BundleID       types.String `tfsdk:"bundle_id"`
	CanBeEmpty     types.Bool   `tfsdk:"can_be_empty"`
	EmptyFieldText types.String `tfsdk:"empty_field_text"`
	IsPublic       types.Bool   `tfsdk:"is_public"`
}

func NewProjectCustomFieldResource() resource.Resource {
	return &projectCustomFieldResource{}
}

func (r *projectCustomFieldResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_custom_field"
}

func (r *projectCustomFieldResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Attaches a custom field to a project. Destroying this resource removes the field and its values from all issues of the project.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Database ID of the project custom field.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "Database ID of the project.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"field_id": schema.StringAttribute{
				Description: "Database ID of the custom field prototype.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"bundle_id": schema.StringAttribute{
				Description: "Database ID of the bundle with the values of the field. Only for enum, state, build, version, owned and user fields. Defaults to the bundle chosen by YouTrack.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"can_be_empty": schema.BoolAttribute{
				Description: "Whether the field can be empty.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"empty_field_text": schema.StringAttribute{
				Description: "Text shown when the field is empty.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"is_public": schema.BoolAttribute{
				Description: "Whether the field is visible to users without the permission to read private fields.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *projectCustomFieldResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan projectCustomFieldModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var field client.CustomField
	if err := r.client.Get(ctx, customFieldPath(plan.FieldID.ValueString()), "id,fieldType(id)", &field); err != nil {
		addAPIError(&resp.Diagnostics, "read", "custom field", err)
		return
	}
	if field.FieldType == nil {
		resp.Diagnostics.AddError("Unable to read custom field", "YouTrack returned no field type for the custom field.")
		return
	}

	fieldType := field.FieldType.ID
	body := projectCustomFieldBody(plan, fieldType)
	body["$type"] = projectCustomFieldType(fieldType)
	body["field"] = client.Ref{ID: field.ID, Type: "CustomField"}

	var projectField client.ProjectCustomField
	if err := r.client.Post(ctx, projectCustomFieldsPath(plan.ProjectID.ValueString()), client.ProjectCustomFieldFields, body, &projectField); err != nil {
		addAPIError(&resp.Diagnostics, "attach", "custom field", err)
		return
	}

	applyProjectCustomField(&plan, projectField)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *projectCustomFieldResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state projectCustomFieldModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var projectField client.ProjectCustomField
	err := r.client.Get(ctx, projectCustomFieldPath(state.ProjectID.ValueString(), state.ID.ValueString()), client.ProjectCustomFieldFields, &projectField)
	if isNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		addAPIError(&resp.Diagnostics, "read", "project custom field", err)
		return
	}

	applyProjectCustomField(&state, projectField)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *projectCustomFieldResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state projectCustomFieldModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var current client.ProjectCustomField
	if err := r.client.Get(ctx, projectCustomFieldPath(state.ProjectID.ValueString(), state.ID.ValueString()), client.ProjectCustomFieldFields, &current); err != nil {
		addAPIError(&resp.Diagnostics, "read", "project custom field", err)
		return
	}
	fieldType := ""
	if current.Field != nil && current.Field.FieldType != nil {
		fieldType = current.Field.FieldType.ID
	}

	body := projectCustomFieldBody(plan, fieldType)
	body["$type"] = current.Type

	var projectField client.ProjectCustomField
	if err := r.client.Post(ctx, projectCustomFieldPath(state.ProjectID.ValueString(), state.ID.ValueString()), client.ProjectCustomFieldFields, body, &projectField); err != nil {
		addAPIError(&resp.Diagnostics, "update", "project custom field", err)
		return
	}

	plan.ID = state.ID
	applyProjectCustomField(&plan, projectField)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *projectCustomFieldResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state projectCustomFieldModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.Delete(ctx, projectCustomFieldPath(state.ProjectID.ValueString(), state.ID.ValueString())); err != nil && !isNotFound(err) {
		addAPIError(&resp.Diagnostics, "detach", "custom field", err)
	}
}

func (r *projectCustomFieldResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	projectID, id, ok := strings.Cut(req.ID, "/")
	if !ok || projectID == "" || id == "" {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected <project_id>/<project_custom_field_id>, got %q.", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), projectID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

func projectCustomFieldBody(plan projectCustomFieldModel, fieldType string) map[string]any {
	body := map[string]any{}
	setIfKnown(body, "canBeEmpty", plan.CanBeEmpty)
	setIfKnown(body, "emptyFieldText", plan.EmptyFieldText)
	setIfKnown(body, "isPublic", plan.IsPublic)
	if bundleType, ok := bundleTypes[fieldTypeKind(fieldType)]; ok && !plan.BundleID.IsNull() && !plan.BundleID.IsUnknown() {
		body["bundle"] = client.Ref{ID: plan.BundleID.ValueString(), Type: bundleType}
	}
	return body
}

func applyProjectCustomField(model *projectCustomFieldModel, projectField client.ProjectCustomField) {
	model.ID = types.StringValue(projectField.ID)
	if projectField.Field != nil {
		model.FieldID = types.StringValue(projectField.Field.ID)
	}
	model.BundleID = refID(projectField.Bundle)
	model.CanBeEmpty = types.BoolValue(projectField.CanBeEmpty)
	model.EmptyFieldText = types.StringValue(projectField.EmptyFieldText)
	model.IsPublic = types.BoolValue(projectField.IsPublic)
}

func projectCustomFieldsPath(projectID string) string {
	return projectPath(projectID) + "/customFields"
}

func projectCustomFieldPath(projectID, id string) string {
	return projectCustomFieldsPath(projectID) + "/" + url.PathEscape(id)
}
