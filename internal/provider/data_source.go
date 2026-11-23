package provider

import (
	"context"
	"errors"
	"maps"
	"strings"

	"github.com/OrcaCD/terraform-provider-orcacd/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type lookupSource[T any] struct {
	kind       string
	attributes map[string]schema.Attribute
	client     *api.Client
	find       func(context.Context, *api.Client, string) (string, error)
	read       func(context.Context, *api.Client, string, *T) error
}

var (
	_ datasource.DataSourceWithConfigure      = (*lookupSource[objectID])(nil)
	_ datasource.DataSourceWithValidateConfig = (*lookupSource[objectID])(nil)
)

func (d *lookupSource[T]) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + d.kind
}

func (d *lookupSource[T]) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attributes := maps.Clone(d.attributes)
	attributes["id"] = schema.StringAttribute{
		Optional: true, Computed: true, Description: "Stable OrcaCD " + d.kind + " ID. Set either id or name.",
	}
	attributes["name"] = schema.StringAttribute{
		Optional: true, Computed: true, Description: "Exact " + d.kind + " name. Set either id or name.",
	}
	resp.Schema = schema.Schema{
		Description: "Look up an existing OrcaCD " + d.kind + ".",
		Attributes:  attributes,
	}
}

func (d *lookupSource[T]) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configuredClient(req.ProviderData, &resp.Diagnostics)
}

func (d *lookupSource[T]) ValidateConfig(ctx context.Context, req datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	var id, name types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, idPath, &id)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, namePath, &name)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if id.IsNull() == name.IsNull() || blankString(name) {
		resp.Diagnostics.AddError("Invalid OrcaCD lookup", "Set exactly one of id or name to a nonempty value.")
	} else if known(id) && !validID(id.ValueString()) {
		resp.Diagnostics.AddAttributeError(idPath, "Invalid OrcaCD ID", "Use a single resource ID from the OrcaCD hub.")
	}
}

func (d *lookupSource[T]) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var id, name types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, idPath, &id)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, namePath, &name)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resourceID := id.ValueString()
	if id.IsNull() {
		var err error
		resourceID, err = d.find(ctx, d.client, name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Look up OrcaCD "+d.kind, err.Error())
			return
		}
	}

	if !validID(resourceID) {
		resp.Diagnostics.AddError("Invalid OrcaCD ID", "Use a single resource ID from the OrcaCD hub.")
		return
	}
	var model T
	if err := d.read(ctx, d.client, resourceID, &model); err != nil {
		resp.Diagnostics.AddError("Read OrcaCD "+d.kind, err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func blankString(value types.String) bool {
	return !value.IsUnknown() && !value.IsNull() && strings.TrimSpace(value.ValueString()) == ""
}

func findByName[T any](items []T, name string, identity func(T) (string, string)) (string, error) {
	id := ""
	for _, item := range items {
		itemID, itemName := identity(item)
		if itemName != name {
			continue
		}
		if id != "" {
			return "", errors.New("multiple visible resources share this name; look up the resource by ID")
		}
		id = itemID
	}
	if id == "" {
		return "", errors.New("no visible resource matches this name")
	}
	return id, nil
}
