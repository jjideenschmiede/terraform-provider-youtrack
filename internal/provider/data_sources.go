// Copyright 2026 J&J Ideenschmiede GmbH
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jjideenschmiede/terraform-provider-youtrack/internal/client"
)

var (
	_ datasource.DataSource = &userDataSource{}
	_ datasource.DataSource = &userGroupDataSource{}
	_ datasource.DataSource = &customFieldDataSource{}
)

type userDataSource struct {
	dataSourceClientHolder
}

type userDataSourceModel struct {
	ID       types.String `tfsdk:"id"`
	Login    types.String `tfsdk:"login"`
	FullName types.String `tfsdk:"full_name"`
	Email    types.String `tfsdk:"email"`
	RingID   types.String `tfsdk:"ring_id"`
	Banned   types.Bool   `tfsdk:"banned"`
}

func NewUserDataSource() datasource.DataSource {
	return &userDataSource{}
}

func (d *userDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (d *userDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up a user by login, for example to use it as project leader.",
		Attributes: map[string]schema.Attribute{
			"login":     schema.StringAttribute{Description: "Login of the user.", Required: true},
			"id":        schema.StringAttribute{Description: "Database ID of the user.", Computed: true},
			"full_name": schema.StringAttribute{Description: "Full name of the user.", Computed: true},
			"email":     schema.StringAttribute{Description: "Email address of the user.", Computed: true},
			"ring_id":   schema.StringAttribute{Description: "ID of the user in Hub.", Computed: true},
			"banned":    schema.BoolAttribute{Description: "Whether the user is banned.", Computed: true},
		},
	}
}

func (d *userDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config userDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	users, err := client.List[client.User](ctx, d.client, "users", client.UserFields)
	if err != nil {
		addAPIError(&resp.Diagnostics, "list", "users", err)
		return
	}
	for _, user := range users {
		if user.Login != config.Login.ValueString() {
			continue
		}
		config.ID = types.StringValue(user.ID)
		config.FullName = types.StringValue(user.FullName)
		config.Email = types.StringValue(user.Email)
		config.RingID = types.StringValue(user.RingID)
		config.Banned = types.BoolValue(user.Banned)
		resp.Diagnostics.Append(resp.State.Set(ctx, config)...)
		return
	}
	resp.Diagnostics.AddError("User not found", fmt.Sprintf("No user with login %q exists.", config.Login.ValueString()))
}

type userGroupDataSource struct {
	dataSourceClientHolder
}

type userGroupDataSourceModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	RingID     types.String `tfsdk:"ring_id"`
	UsersCount types.Int64  `tfsdk:"users_count"`
}

func NewUserGroupDataSource() datasource.DataSource {
	return &userGroupDataSource{}
}

func (d *userGroupDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_group"
}

func (d *userGroupDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up a user group by name, for example `All Users` or `Registered Users`.",
		Attributes: map[string]schema.Attribute{
			"name":        schema.StringAttribute{Description: "Name of the group.", Required: true},
			"id":          schema.StringAttribute{Description: "Database ID of the group.", Computed: true},
			"ring_id":     schema.StringAttribute{Description: "ID of the group in Hub.", Computed: true},
			"users_count": schema.Int64Attribute{Description: "Number of users in the group.", Computed: true},
		},
	}
}

func (d *userGroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config userGroupDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groups, err := client.List[client.UserGroup](ctx, d.client, "groups", client.UserGroupFields)
	if err != nil {
		addAPIError(&resp.Diagnostics, "list", "user groups", err)
		return
	}
	for _, group := range groups {
		if group.Name != config.Name.ValueString() {
			continue
		}
		config.ID = types.StringValue(group.ID)
		config.RingID = types.StringValue(group.RingID)
		config.UsersCount = types.Int64Value(group.UsersCount)
		resp.Diagnostics.Append(resp.State.Set(ctx, config)...)
		return
	}
	resp.Diagnostics.AddError("User group not found", fmt.Sprintf("No user group with name %q exists.", config.Name.ValueString()))
}

type customFieldDataSource struct {
	dataSourceClientHolder
}

type customFieldDataSourceModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Type types.String `tfsdk:"type"`
}

func NewCustomFieldDataSource() datasource.DataSource {
	return &customFieldDataSource{}
}

func (d *customFieldDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_field"
}

func (d *customFieldDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up an existing custom field prototype by name, for example the built-in `Priority` or `State`.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{Description: "Name of the custom field.", Required: true},
			"id":   schema.StringAttribute{Description: "Database ID of the custom field.", Computed: true},
			"type": schema.StringAttribute{Description: "Field type, for example `enum[1]`.", Computed: true},
		},
	}
}

func (d *customFieldDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config customFieldDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	fields, err := client.List[client.CustomField](ctx, d.client, "admin/customFieldSettings/customFields", client.CustomFieldFields)
	if err != nil {
		addAPIError(&resp.Diagnostics, "list", "custom fields", err)
		return
	}
	for _, field := range fields {
		if field.Name != config.Name.ValueString() {
			continue
		}
		config.ID = types.StringValue(field.ID)
		if field.FieldType != nil {
			config.Type = types.StringValue(field.FieldType.ID)
		}
		resp.Diagnostics.Append(resp.State.Set(ctx, config)...)
		return
	}
	resp.Diagnostics.AddError("Custom field not found", fmt.Sprintf("No custom field with name %q exists.", config.Name.ValueString()))
}
