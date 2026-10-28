// Copyright 2026 J&J Ideenschmiede GmbH. All rights reserved.

package provider

import (
	"context"
	"fmt"
	"net/url"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	_ resource.Resource                = &bundleResource{}
	_ resource.ResourceWithImportState = &bundleResource{}
)

type bundleKind struct {
	typeName    string
	path        string
	elementType string
	label       string
	isState     bool
}

var (
	enumBundleKind = bundleKind{
		typeName:    "_enum_bundle",
		path:        "enum",
		elementType: "EnumBundleElement",
		label:       "enum bundle",
	}
	stateBundleKind = bundleKind{
		typeName:    "_state_bundle",
		path:        "state",
		elementType: "StateBundleElement",
		label:       "state bundle",
		isState:     true,
	}
)

type bundleResource struct {
	clientHolder
	kind bundleKind
}

type bundleModel struct {
	ID       types.String `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	Values   types.List   `tfsdk:"values"`
	ValueIDs types.Map    `tfsdk:"value_ids"`
}

type bundleValueModel struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Archived    types.Bool   `tfsdk:"archived"`
	ColorID     types.String `tfsdk:"color_id"`
	IsResolved  types.Bool   `tfsdk:"is_resolved"`
}

func NewEnumBundleResource() resource.Resource {
	return &bundleResource{kind: enumBundleKind}
}

func NewStateBundleResource() resource.Resource {
	return &bundleResource{kind: stateBundleKind}
}

func (r *bundleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.kind.typeName
}

func (r *bundleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	valueAttributes := map[string]schema.Attribute{
		"name": schema.StringAttribute{
			Description: "Name of the value. Values are matched by name, so renaming a value deletes it and creates a new one. Archive values that are still used by issues instead of removing them.",
			Required:    true,
		},
		"description": schema.StringAttribute{
			Description: "Description of the value.",
			Optional:    true,
		},
		"archived": schema.BoolAttribute{
			Description: "Whether the value is archived and can no longer be selected.",
			Optional:    true,
			Computed:    true,
			Default:     booldefault.StaticBool(false),
		},
		"color_id": schema.StringAttribute{
			Description: "Index of the color in the YouTrack palette, from `1` to `35`. Not set means no color.",
			Optional:    true,
			Validators:  colorValidators(),
		},
		"is_resolved": schema.BoolAttribute{
			Description: "Whether issues in this state count as resolved. Only used by state bundles.",
			Optional:    true,
			Computed:    true,
			Default:     booldefault.StaticBool(false),
		},
	}
	if !r.kind.isState {
		delete(valueAttributes, "is_resolved")
	}

	resp.Schema = schema.Schema{
		Description: fmt.Sprintf("Manages an %s, the set of values of a custom field. The order of `values` defines the order in YouTrack.", r.kind.label),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Database ID of the bundle.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the bundle.",
				Required:    true,
			},
			"values": schema.ListNestedAttribute{
				Description: "Values of the bundle in display order.",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: valueAttributes,
				},
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
			},
			"value_ids": schema.MapAttribute{
				Description: "Database IDs of the values, keyed by name.",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *bundleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan bundleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	values := r.values(ctx, plan.Values, &resp.Diagnostics)
	if resp.Diagnostics.HasError() || !uniqueValueNames(values, &resp.Diagnostics) {
		return
	}

	var bundle client.Bundle
	if err := r.client.Post(ctx, r.collectionPath(), "id", map[string]any{"name": plan.Name.ValueString()}, &bundle); err != nil {
		addAPIError(&resp.Diagnostics, "create", r.kind.label, err)
		return
	}
	plan.ID = types.StringValue(bundle.ID)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), plan.ID)...)

	r.syncValues(ctx, plan.ID.ValueString(), values, nil, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.refresh(ctx, &plan, nil, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *bundleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state bundleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	found := r.refresh(ctx, &state, nil, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *bundleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state bundleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	values := r.values(ctx, plan.Values, &resp.Diagnostics)
	if resp.Diagnostics.HasError() || !uniqueValueNames(values, &resp.Diagnostics) {
		return
	}
	plan.ID = state.ID

	if !plan.Name.Equal(state.Name) {
		if err := r.client.Post(ctx, r.bundlePath(plan.ID.ValueString()), "id", map[string]any{"name": plan.Name.ValueString()}, nil); err != nil {
			addAPIError(&resp.Diagnostics, "update", r.kind.label, err)
			return
		}
	}

	var current client.Bundle
	if err := r.client.Get(ctx, r.bundlePath(plan.ID.ValueString()), r.fields(), &current); err != nil {
		addAPIError(&resp.Diagnostics, "read", r.kind.label, err)
		return
	}
	deleted := r.syncValues(ctx, plan.ID.ValueString(), values, current.Values, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.refresh(ctx, &plan, deleted, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *bundleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state bundleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.Delete(ctx, r.bundlePath(state.ID.ValueString())); err != nil && !isNotFound(err) {
		addAPIError(&resp.Diagnostics, "delete", r.kind.label, err)
	}
}

func (r *bundleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *bundleResource) syncValues(ctx context.Context, bundleID string, values []bundleValueModel, existing []client.BundleElement, diags *diag.Diagnostics) map[string]bool {
	byName := make(map[string]client.BundleElement, len(existing))
	for _, element := range existing {
		byName[element.Name] = element
	}

	for index, value := range values {
		body := r.valueBody(value, index)
		name := value.Name.ValueString()
		valuesPath := r.bundlePath(bundleID) + "/values"
		if element, ok := byName[name]; ok {
			delete(byName, name)
			valuesPath += "/" + url.PathEscape(element.ID)
		}
		if err := r.client.Post(ctx, valuesPath, "id", body, nil); err != nil {
			addAPIError(diags, "save value "+name+" of", r.kind.label, err)
			return nil
		}
	}

	deleted := make(map[string]bool, len(byName))
	for name, element := range byName {
		if err := r.client.Delete(ctx, r.bundlePath(bundleID)+"/values/"+url.PathEscape(element.ID)); err != nil && !isNotFound(err) {
			addAPIError(diags, "delete value "+name+" of", r.kind.label, err)
			return nil
		}
		deleted[element.ID] = true
	}
	return deleted
}

func (r *bundleResource) valueBody(value bundleValueModel, ordinal int) map[string]any {
	body := map[string]any{
		"$type":       r.kind.elementType,
		"name":        value.Name.ValueString(),
		"description": value.Description.ValueString(),
		"archived":    value.Archived.ValueBool(),
		"ordinal":     ordinal,
		"color":       client.Ref{ID: "0"},
	}
	if !value.ColorID.IsNull() {
		body["color"] = client.Ref{ID: value.ColorID.ValueString()}
	}
	if r.kind.isState {
		body["isResolved"] = value.IsResolved.ValueBool()
	}
	return body
}

func (r *bundleResource) refresh(ctx context.Context, model *bundleModel, deleted map[string]bool, diags *diag.Diagnostics) bool {
	var bundle client.Bundle
	err := r.client.Get(ctx, r.bundlePath(model.ID.ValueString()), r.fields(), &bundle)
	if isNotFound(err) {
		return false
	}
	if err != nil {
		addAPIError(diags, "read", r.kind.label, err)
		return false
	}

	sort.SliceStable(bundle.Values, func(i, j int) bool {
		return bundle.Values[i].Ordinal < bundle.Values[j].Ordinal
	})

	model.ID = types.StringValue(bundle.ID)
	model.Name = types.StringValue(bundle.Name)
	values := make([]bundleValueModel, 0, len(bundle.Values))
	ids := make(map[string]attr.Value, len(bundle.Values))
	for _, element := range bundle.Values {
		if deleted[element.ID] {
			continue
		}
		value := bundleValueModel{
			Name:        types.StringValue(element.Name),
			Description: optionalString(element.Description),
			Archived:    types.BoolValue(element.Archived),
			ColorID:     colorID(element.Color),
			IsResolved:  types.BoolNull(),
		}
		if value.ColorID.ValueString() == "0" {
			value.ColorID = types.StringNull()
		}
		if r.kind.isState {
			value.IsResolved = types.BoolValue(element.IsResolved)
		}
		values = append(values, value)
		ids[element.Name] = types.StringValue(element.ID)
	}

	model.Values = r.valuesList(values, diags)
	valueIDs, d := types.MapValue(types.StringType, ids)
	diags.Append(d...)
	model.ValueIDs = valueIDs
	return true
}

func (r *bundleResource) fields() string {
	if r.kind.isState {
		return client.BundleFields
	}
	return "id,name,values(id,name,description,archived,ordinal,color(id))"
}

func (r *bundleResource) collectionPath() string {
	return "admin/customFieldSettings/bundles/" + r.kind.path
}

func (r *bundleResource) bundlePath(id string) string {
	return r.collectionPath() + "/" + url.PathEscape(id)
}

func (r *bundleResource) valueAttrTypes() map[string]attr.Type {
	attrTypes := map[string]attr.Type{
		"name":        types.StringType,
		"description": types.StringType,
		"archived":    types.BoolType,
		"color_id":    types.StringType,
	}
	if r.kind.isState {
		attrTypes["is_resolved"] = types.BoolType
	}
	return attrTypes
}

func (r *bundleResource) values(ctx context.Context, list types.List, diags *diag.Diagnostics) []bundleValueModel {
	objects := make([]types.Object, 0, len(list.Elements()))
	diags.Append(list.ElementsAs(ctx, &objects, false)...)
	values := make([]bundleValueModel, 0, len(objects))
	for _, object := range objects {
		attributes := object.Attributes()
		value := bundleValueModel{
			Name:        attributes["name"].(types.String),
			Description: attributes["description"].(types.String),
			Archived:    attributes["archived"].(types.Bool),
			ColorID:     attributes["color_id"].(types.String),
			IsResolved:  types.BoolNull(),
		}
		if r.kind.isState {
			value.IsResolved = attributes["is_resolved"].(types.Bool)
		}
		values = append(values, value)
	}
	return values
}

func (r *bundleResource) valuesList(values []bundleValueModel, diags *diag.Diagnostics) types.List {
	attrTypes := r.valueAttrTypes()
	objects := make([]attr.Value, 0, len(values))
	for _, value := range values {
		attributes := map[string]attr.Value{
			"name":        value.Name,
			"description": value.Description,
			"archived":    value.Archived,
			"color_id":    value.ColorID,
		}
		if r.kind.isState {
			attributes["is_resolved"] = value.IsResolved
		}
		object, d := types.ObjectValue(attrTypes, attributes)
		diags.Append(d...)
		objects = append(objects, object)
	}
	list, d := types.ListValue(types.ObjectType{AttrTypes: attrTypes}, objects)
	diags.Append(d...)
	return list
}

func uniqueValueNames(values []bundleValueModel, diags *diag.Diagnostics) bool {
	seen := make(map[string]bool, len(values))
	for index, value := range values {
		name := value.Name.ValueString()
		if seen[name] {
			diags.AddAttributeError(path.Root("values").AtListIndex(index).AtName("name"), "Duplicate value name", fmt.Sprintf("The value %q is defined more than once.", name))
			return false
		}
		seen[name] = true
	}
	return true
}
