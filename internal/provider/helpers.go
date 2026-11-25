// Copyright 2026 J&J Ideenschmiede GmbH
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jjideenschmiede/terraform-provider-youtrack/internal/client"
)

type clientHolder struct {
	client *client.Client
}

func (h *clientHolder) configure(providerData any, diags *diag.Diagnostics) {
	if providerData == nil {
		return
	}
	c, ok := providerData.(*client.Client)
	if !ok {
		diags.AddError("Unexpected provider data", fmt.Sprintf("Expected *client.Client, got %T.", providerData))
		return
	}
	h.client = c
}

func (h *clientHolder) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	h.configure(req.ProviderData, &resp.Diagnostics)
}

type dataSourceClientHolder struct {
	client *client.Client
}

func (h *dataSourceClientHolder) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	holder := clientHolder{}
	holder.configure(req.ProviderData, &resp.Diagnostics)
	h.client = holder.client
}

func isNotFound(err error) bool {
	return errors.Is(err, client.ErrNotFound)
}

func addAPIError(diags *diag.Diagnostics, action, kind string, err error) {
	diags.AddError(fmt.Sprintf("Unable to %s %s", action, kind), err.Error())
}

func refOrNil(value types.String) any {
	if value.IsNull() || value.IsUnknown() || value.ValueString() == "" {
		return nil
	}
	return client.Ref{ID: value.ValueString()}
}

func refID(ref *client.Ref) types.String {
	if ref == nil || ref.ID == "" {
		return types.StringNull()
	}
	return types.StringValue(ref.ID)
}

func optionalString(value string) types.String {
	if value == "" {
		return types.StringNull()
	}
	return types.StringValue(value)
}

func colorID(style *client.FieldStyle) types.String {
	if style == nil || style.ID == "" {
		return types.StringNull()
	}
	return types.StringValue(style.ID)
}

func setIfKnown(body map[string]any, key string, value attrValue) {
	if value.IsNull() || value.IsUnknown() {
		return
	}
	switch v := value.(type) {
	case types.String:
		body[key] = v.ValueString()
	case types.Bool:
		body[key] = v.ValueBool()
	case types.Int64:
		body[key] = v.ValueInt64()
	}
}

type attrValue interface {
	IsNull() bool
	IsUnknown() bool
}

var colorPattern = regexp.MustCompile(`^([0-9]|[12][0-9]|3[0-5])$`)

func colorValidators() []validator.String {
	return []validator.String{
		stringvalidator.RegexMatches(colorPattern, "must be a palette index from 0 to 35"),
	}
}
