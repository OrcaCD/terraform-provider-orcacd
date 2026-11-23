package provider

import (
	"context"

	"github.com/OrcaCD/terraform-provider-orcacd/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type applicationModel struct {
	objectID
	RepositoryID types.String `tfsdk:"repository_id"`
	AgentID      types.String `tfsdk:"agent_id"`
	Branch       types.String `tfsdk:"branch"`
	Path         types.String `tfsdk:"path"`
}

type applicationLookup struct {
	applicationModel
	Icon                     types.String `tfsdk:"icon"`
	HealthStatus             types.String `tfsdk:"health_status"`
	SyncStatus               types.String `tfsdk:"sync_status"`
	ImagePollEnabled         types.Bool   `tfsdk:"image_poll_enabled"`
	ImagePollIntervalSeconds types.Int64  `tfsdk:"image_poll_interval_seconds"`
	DeleteOldImages          types.Bool   `tfsdk:"delete_old_images"`
	RollbackEnabled          types.Bool   `tfsdk:"rollback_enabled"`
}

func newApplication() resource.Resource {
	return &managedResource[applicationModel]{
		kind: "application",
		schema: schema.Schema{
			Description: "Register a Compose application from a Git repository on an OrcaCD agent.",
			Attributes: map[string]schema.Attribute{
				"id":            idAttribute(),
				"name":          schema.StringAttribute{Required: true, Description: "Application name."},
				"repository_id": schema.StringAttribute{Required: true, Description: "ID of the application's OrcaCD repository."},
				"agent_id":      schema.StringAttribute{Required: true, Description: "ID of the OrcaCD agent for deployment."},
				"branch":        schema.StringAttribute{Required: true, Description: "Git branch containing the Compose file."},
				"path":          schema.StringAttribute{Required: true, Description: "Path to the Compose file relative to the repository root."},
			},
		},
		create: createApplication,
		read:   readApplication,
		update: updateApplication,
		delete: func(ctx context.Context, client *api.Client, id string) error {
			return api.Delete(client.DeleteApplication(ctx, id))
		},
	}
}

func createApplication(ctx context.Context, client *api.Client, model *applicationModel) error {
	reply, err := api.Decode[api.ApplicationResponse](client.CreateApplication(ctx, api.CreateApplicationRequest{
		Name: model.Name.ValueString(), RepositoryId: model.RepositoryID.ValueString(),
		AgentId: model.AgentID.ValueString(), Branch: model.Branch.ValueString(), Path: model.Path.ValueString(),
	}))
	if err != nil {
		return err
	}
	return model.setApplication(reply)
}

func updateApplication(ctx context.Context, client *api.Client, plan *applicationModel, _ applicationModel, prior applicationModel) error {
	id := prior.ID.ValueString()
	current, err := api.Decode[api.ApplicationResponse](client.GetApplication(ctx, id))
	if err != nil {
		return err
	}

	// The hub's PUT replaces these settings even for a source-only update.
	reply, err := api.Decode[api.ApplicationResponse](client.UpdateApplication(ctx, id, api.UpdateApplicationRequest{
		Name: plan.Name.ValueString(), RepositoryId: plan.RepositoryID.ValueString(),
		AgentId: plan.AgentID.ValueString(), Branch: plan.Branch.ValueString(), Path: plan.Path.ValueString(),
		ImagePollEnabled: &current.ImagePollEnabled, ImagePollIntervalSeconds: &current.ImagePollIntervalSeconds,
		DeleteOldImages: &current.DeleteOldImages,
	}))
	if err != nil {
		return err
	}
	return plan.setApplication(reply)
}

func readApplication(ctx context.Context, client *api.Client, id string, model *applicationModel) error {
	reply, err := api.Decode[api.ApplicationResponse](client.GetApplication(ctx, id))
	if err != nil {
		return err
	}
	return model.setApplication(reply)
}

func (m *applicationModel) setApplication(reply api.ApplicationResponse) error {
	m.RepositoryID = types.StringValue(reply.RepositoryId)
	m.AgentID = types.StringValue(reply.AgentId)
	m.Branch = types.StringValue(reply.Branch)
	m.Path = types.StringValue(reply.Path)
	return m.set(reply.Id, reply.Name)
}

func newApplicationDataSource() datasource.DataSource {
	return &lookupSource[applicationLookup]{
		kind: "application",
		attributes: map[string]dsschema.Attribute{
			"repository_id":               dsschema.StringAttribute{Computed: true, Description: "ID of the application's repository."},
			"agent_id":                    dsschema.StringAttribute{Computed: true, Description: "ID of the agent for deployment."},
			"branch":                      dsschema.StringAttribute{Computed: true, Description: "Git branch containing the Compose file."},
			"path":                        dsschema.StringAttribute{Computed: true, Description: "Path to the Compose file in the repository."},
			"icon":                        dsschema.StringAttribute{Computed: true, Description: "Effective application icon."},
			"health_status":               dsschema.StringAttribute{Computed: true, Description: "Current deployment health."},
			"sync_status":                 dsschema.StringAttribute{Computed: true, Description: "Current sync status."},
			"image_poll_enabled":          dsschema.BoolAttribute{Computed: true, Description: "Whether image polling is enabled."},
			"image_poll_interval_seconds": dsschema.Int64Attribute{Computed: true, Description: "Effective image polling interval in seconds."},
			"delete_old_images":           dsschema.BoolAttribute{Computed: true, Description: "Whether the agent deletes old images."},
			"rollback_enabled":            dsschema.BoolAttribute{Computed: true, Description: "Whether automatic rollback is enabled."},
		},
		find: func(ctx context.Context, client *api.Client, name string) (string, error) {
			items, err := api.Decode[[]api.ApplicationListResponse](client.ListApplications(ctx))
			if err != nil {
				return "", err
			}
			return findByName(items, name, func(item api.ApplicationListResponse) (string, string) { return item.Id, item.Name })
		},
		read: func(ctx context.Context, client *api.Client, id string, model *applicationLookup) error {
			reply, err := api.Decode[api.ApplicationResponse](client.GetApplication(ctx, id))
			if err != nil {
				return err
			}
			model.Icon = types.StringValue(reply.Icon)
			model.HealthStatus = types.StringValue(reply.HealthStatus)
			model.SyncStatus = types.StringValue(reply.SyncStatus)
			model.ImagePollEnabled = types.BoolValue(reply.ImagePollEnabled)
			model.ImagePollIntervalSeconds = types.Int64Value(reply.ImagePollIntervalSeconds)
			model.DeleteOldImages = types.BoolValue(reply.DeleteOldImages)
			model.RollbackEnabled = types.BoolValue(reply.Rollback.Enabled)
			return model.setApplication(reply)
		},
	}
}
