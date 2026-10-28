// Copyright 2026 J&J Ideenschmiede GmbH. All rights reserved.

package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jjideenschmiede/terraform-provider-youtrack/internal/client"
)

var _ provider.Provider = &youtrackProvider{}

type youtrackProvider struct {
	version string
}

type youtrackProviderModel struct {
	URL   types.String `tfsdk:"url"`
	Token types.String `tfsdk:"token"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &youtrackProvider{version: version}
	}
}

func (p *youtrackProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "youtrack"
	resp.Version = p.version
}

func (p *youtrackProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the configuration of a JetBrains YouTrack instance through its REST API.",
		Attributes: map[string]schema.Attribute{
			"url": schema.StringAttribute{
				Description: "Base URL of the YouTrack instance, for example `https://example.youtrack.cloud`. Can also be set with the `YOUTRACK_URL` environment variable.",
				Optional:    true,
			},
			"token": schema.StringAttribute{
				Description: "Permanent token used for authentication. Can also be set with the `YOUTRACK_TOKEN` environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
		},
	}
}

func (p *youtrackProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config youtrackProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.URL.IsUnknown() || config.Token.IsUnknown() {
		return
	}

	baseURL := os.Getenv("YOUTRACK_URL")
	if !config.URL.IsNull() {
		baseURL = config.URL.ValueString()
	}
	token := os.Getenv("YOUTRACK_TOKEN")
	if !config.Token.IsNull() {
		token = config.Token.ValueString()
	}

	if baseURL == "" {
		resp.Diagnostics.AddAttributeError(path.Root("url"), "Missing YouTrack URL", "Set the url attribute or the YOUTRACK_URL environment variable.")
	}
	if token == "" {
		resp.Diagnostics.AddAttributeError(path.Root("token"), "Missing YouTrack token", "Set the token attribute or the YOUTRACK_TOKEN environment variable.")
	}
	if resp.Diagnostics.HasError() {
		return
	}

	c, err := client.New(baseURL, token, "terraform-provider-youtrack/"+p.version)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("url"), "Invalid YouTrack URL", err.Error())
		return
	}

	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *youtrackProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewNotificationTemplateResource,
		NewProjectResource,
		NewUserGroupResource,
		NewCustomFieldResource,
		NewEnumBundleResource,
		NewStateBundleResource,
		NewProjectCustomFieldResource,
		NewTagResource,
		NewSavedQueryResource,
		NewIssueLinkTypeResource,
	}
}

func (p *youtrackProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewUserDataSource,
		NewUserGroupDataSource,
		NewCustomFieldDataSource,
	}
}
