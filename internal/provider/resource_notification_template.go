// Copyright 2026 J&J Ideenschmiede GmbH. All rights reserved.

package provider

import (
	"context"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jjideenschmiede/terraform-provider-youtrack/internal/client"
)

var (
	_ resource.Resource                = &notificationTemplateResource{}
	_ resource.ResourceWithImportState = &notificationTemplateResource{}
)

type notificationTemplateResource struct {
	clientHolder
}

type notificationTemplateModel struct {
	ID      types.String `tfsdk:"id"`
	Content types.String `tfsdk:"content"`
}

func NewNotificationTemplateResource() resource.Resource {
	return &notificationTemplateResource{}
}

func (r *notificationTemplateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_notification_template"
}

func (r *notificationTemplateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the content of a built-in YouTrack notification template (FreeMarker). " +
			"YouTrack has no API to reset a template, so destroying this resource only removes it from the state. " +
			"Use **Reset to default** in Administration → Notification Templates to restore the original content.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "ID of the template, for example `helpdesk_footer`. This matches the file name without the `.ftl` extension.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"content": schema.StringAttribute{
				Description: "FreeMarker source of the template.",
				Required:    true,
			},
		},
	}
}

func (r *notificationTemplateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan notificationTemplateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.write(ctx, &plan, &resp.Diagnostics, "create")
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *notificationTemplateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state notificationTemplateModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var template client.NotificationTemplate
	err := r.client.Get(ctx, templatePath(state.ID.ValueString()), client.NotificationTemplateFields, &template)
	if isNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		addAPIError(&resp.Diagnostics, "read", "notification template", err)
		return
	}

	state.ID = types.StringValue(template.ID)
	state.Content = types.StringValue(template.Content)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *notificationTemplateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan notificationTemplateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.write(ctx, &plan, &resp.Diagnostics, "update")
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *notificationTemplateResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning(
		"Notification template was not reset",
		"YouTrack has no API to reset a notification template. The template keeps its current content until it is reset with Reset to default in the YouTrack administration.",
	)
}

func (r *notificationTemplateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *notificationTemplateResource) write(ctx context.Context, plan *notificationTemplateModel, diags *diag.Diagnostics, action string) {
	body := client.NotificationTemplate{ID: plan.ID.ValueString(), Content: plan.Content.ValueString()}
	if err := r.client.Post(ctx, templatePath(body.ID), client.NotificationTemplateFields, body, nil); err != nil {
		addAPIError(diags, action, "notification template", err)
	}
}

func templatePath(id string) string {
	return "admin/notificationTemplates/" + url.PathEscape(id)
}
