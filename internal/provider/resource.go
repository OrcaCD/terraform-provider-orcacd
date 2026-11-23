package provider

import (
	"context"
	"errors"

	"github.com/OrcaCD/terraform-provider-orcacd/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type objectID struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

type managedResource[T any] struct {
	kind   string
	schema schema.Schema
	client *api.Client
	create func(context.Context, *api.Client, *T) error
	read   func(context.Context, *api.Client, string, *T) error
	update func(context.Context, *api.Client, *T, T, T) error
	delete func(context.Context, *api.Client, string) error
}

var (
	_ resource.ResourceWithConfigure   = (*managedResource[objectID])(nil)
	_ resource.ResourceWithImportState = (*managedResource[objectID])(nil)
)

func (r *managedResource[T]) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + r.kind
}

func (r *managedResource[T]) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = r.schema
}

func (r *managedResource[T]) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configuredClient(req.ProviderData, &resp.Diagnostics)
}

func (r *managedResource[T]) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model T
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.create(ctx, r.client, &model); err != nil {
		resp.Diagnostics.AddError("Create OrcaCD "+r.kind, err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *managedResource[T]) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model T
	var id types.String
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, idPath, &id)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.read(ctx, r.client, id.ValueString(), &model)
	if errors.Is(err, api.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Read OrcaCD "+r.kind, err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *managedResource[T]) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, config, prior T
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.update(ctx, r.client, &plan, config, prior); err != nil {
		resp.Diagnostics.AddError("Update OrcaCD "+r.kind, err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *managedResource[T]) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var id types.String
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, idPath, &id)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.delete(ctx, r.client, id.ValueString()); err != nil && !errors.Is(err, api.ErrNotFound) {
		resp.Diagnostics.AddError("Delete OrcaCD "+r.kind, err.Error())
	}
}

func (r *managedResource[T]) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if !validID(req.ID) {
		resp.Diagnostics.AddError("Invalid OrcaCD import ID", "Use a single resource ID from the OrcaCD hub.")
		return
	}
	resource.ImportStatePassthroughID(ctx, idPath, req, resp)
}

func idAttribute() schema.StringAttribute {
	return schema.StringAttribute{
		Computed: true, Description: "Stable ID assigned by the OrcaCD hub.",
		PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
	}
}

func configuredClient(data any, diagnostics *diag.Diagnostics) *api.Client {
	if data == nil {
		return nil
	}
	client, ok := data.(*api.Client)
	if !ok {
		diagnostics.AddError("Invalid OrcaCD client", "The provider supplied an unexpected client type.")
	}
	return client
}
