// Copyright 2026 J&J Ideenschmiede GmbH
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jjideenschmiede/terraform-provider-youtrack/internal/client"
)

var (
	_ resource.Resource                = &customFieldResource{}
	_ resource.ResourceWithImportState = &customFieldResource{}
)

var fieldTypes = []string{
	"enum[1]", "enum[*]",
	"build[1]", "build[*]",
	"state[1]",
	"version[1]", "version[*]",
	"ownedField[1]", "ownedField[*]",
	"user[1]", "user[*]",
	"group[1]", "group[*]",
	"integer", "float", "date", "date and time", "period", "string", "text",
}

var projectCustomFieldTypes = map[string]string{
	"enum":       "EnumProjectCustomField",
	"build":      "BuildProjectCustomField",
	"state":      "StateProjectCustomField",
	"version":    "VersionProjectCustomField",
	"ownedField": "OwnedProjectCustomField",
	"user":       "UserProjectCustomField",
	"group":      "GroupProjectCustomField",
	"period":     "PeriodProjectCustomField",
	"text":       "TextProjectCustomField",
}

var bundleTypes = map[string]string{
	"enum":       "EnumBundle",
	"build":      "BuildBundle",
	"state":      "StateBundle",
	"version":    "VersionBundle",
	"ownedField": "OwnedBundle",
	"user":       "UserBundle",
}

func fieldTypeKind(fieldType string) string {
	kind, _, _ := strings.Cut(fieldType, "[")
	return kind
}

func projectCustomFieldType(fieldType string) string {
	if t, ok := projectCustomFieldTypes[fieldTypeKind(fieldType)]; ok {
		return t
	}
	return "SimpleProjectCustomField"
}

type customFieldResource struct {
	clientHolder
}

type customFieldModel struct {
	ID                     types.String `tfsdk:"id"`
	Name                   types.String `tfsdk:"name"`
	Type                   types.String `tfsdk:"type"`
	Aliases                types.String `tfsdk:"aliases"`
	IsAutoAttached         types.Bool   `tfsdk:"is_auto_attached"`
	IsDisplayedInIssueList types.Bool   `tfsdk:"is_displayed_in_issue_list"`
}

func NewCustomFieldResource() resource.Resource {
	return &customFieldResource{}
}

func (r *customFieldResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_field"
}

func (r *customFieldResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a global custom field prototype. Use `youtrack_project_custom_field` to attach it to a project.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Database ID of the custom field.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the custom field.",
				Required:    true,
			},
			"type": schema.StringAttribute{
				Description: "Field type, for example `enum[1]`, `state[1]`, `user[*]` or `string`. `[1]` stores a single value, `[*]` multiple values. Changing it recreates the field.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(fieldTypes...),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"aliases": schema.StringAttribute{
				Description: "Comma-separated aliases that can be used in commands and queries.",
				Optional:    true,
			},
			"is_auto_attached": schema.BoolAttribute{
				Description: "Whether the field is attached to new projects automatically.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"is_displayed_in_issue_list": schema.BoolAttribute{
				Description: "Whether the field is shown in the issue list.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *customFieldResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan customFieldModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := customFieldBody(plan)
	body["fieldType"] = client.Ref{ID: plan.Type.ValueString()}

	var field client.CustomField
	if err := r.client.Post(ctx, "admin/customFieldSettings/customFields", client.CustomFieldFields, body, &field); err != nil {
		addAPIError(&resp.Diagnostics, "create", "custom field", err)
		return
	}

	applyCustomField(&plan, field)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *customFieldResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state customFieldModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var field client.CustomField
	err := r.client.Get(ctx, customFieldPath(state.ID.ValueString()), client.CustomFieldFields, &field)
	if isNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		addAPIError(&resp.Diagnostics, "read", "custom field", err)
		return
	}

	applyCustomField(&state, field)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *customFieldResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state customFieldModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := customFieldBody(plan)
	if plan.Aliases.IsNull() && !state.Aliases.IsNull() {
		body["aliases"] = ""
	}

	var field client.CustomField
	if err := r.client.Post(ctx, customFieldPath(state.ID.ValueString()), client.CustomFieldFields, body, &field); err != nil {
		addAPIError(&resp.Diagnostics, "update", "custom field", err)
		return
	}

	applyCustomField(&plan, field)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *customFieldResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state customFieldModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.Delete(ctx, customFieldPath(state.ID.ValueString())); err != nil && !isNotFound(err) {
		addAPIError(&resp.Diagnostics, "delete", "custom field", err)
	}
}

func (r *customFieldResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func customFieldBody(plan customFieldModel) map[string]any {
	body := map[string]any{"name": plan.Name.ValueString()}
	setIfKnown(body, "aliases", plan.Aliases)
	setIfKnown(body, "isAutoAttached", plan.IsAutoAttached)
	setIfKnown(body, "isDisplayedInIssueList", plan.IsDisplayedInIssueList)
	return body
}

func applyCustomField(model *customFieldModel, field client.CustomField) {
	model.ID = types.StringValue(field.ID)
	model.Name = types.StringValue(field.Name)
	if field.FieldType != nil {
		model.Type = types.StringValue(field.FieldType.ID)
	}
	model.Aliases = optionalString(field.Aliases)
	model.IsAutoAttached = types.BoolValue(field.IsAutoAttached)
	model.IsDisplayedInIssueList = types.BoolValue(field.IsDisplayedInIssueList)
}

func customFieldPath(id string) string {
	return "admin/customFieldSettings/customFields/" + url.PathEscape(id)
}
