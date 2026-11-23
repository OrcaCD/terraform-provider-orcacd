package provider

import (
	"context"
	"strings"

	"github.com/OrcaCD/terraform-provider-orcacd/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type repositoryDetails struct {
	objectID
	URL                               types.String `tfsdk:"url"`
	Provider                          types.String `tfsdk:"git_provider"`
	AuthMethod                        types.String `tfsdk:"auth_method"`
	SyncType                          types.String `tfsdk:"sync_type"`
	PollingIntervalSeconds            types.Int64  `tfsdk:"polling_interval_seconds"`
	GitHubActionsOIDCEnabled          types.Bool   `tfsdk:"github_actions_oidc_enabled"`
	GitHubActionsOIDCAllowRepoSync    types.Bool   `tfsdk:"github_actions_oidc_allow_repo_sync"`
	GitHubActionsOIDCAllowImageSync   types.Bool   `tfsdk:"github_actions_oidc_allow_image_sync"`
	GitHubActionsOIDCAllowedBranches  types.Set    `tfsdk:"github_actions_oidc_allowed_branches"`
	GitHubActionsOIDCAllowedWorkflows types.Set    `tfsdk:"github_actions_oidc_allowed_workflows"`
	RequireSignedCommits              types.Bool   `tfsdk:"require_signed_commits"`
}

type repositoryModel struct {
	repositoryDetails
	AuthUser      types.String `tfsdk:"auth_user"`
	AuthToken     types.String `tfsdk:"auth_token"`
	WebhookSecret types.String `tfsdk:"webhook_secret"`
	WebhookURL    types.String `tfsdk:"webhook_url"`
}

func newRepository() resource.Resource {
	return &managedResource[repositoryModel]{
		kind: "repository",
		schema: schema.Schema{
			Description: "Register and configure a Git repository on the OrcaCD hub.",
			Attributes: map[string]schema.Attribute{
				"id":             idAttribute(),
				"name":           schema.StringAttribute{Computed: true, Description: "Repository owner and name derived from the URL."},
				"url":            replacementString("Git repository URL. Provide credentials separately.", true, false, equivalentString{normalizeRepositoryURL}),
				"git_provider":   replacementString("Git provider, such as github, gitlab, gitea, bitbucket, azure_devops or generic.", true, false),
				"auth_method":    schema.StringAttribute{Optional: true, Computed: true, Description: "Authentication method supported by the Git provider. Defaults to none at creation."},
				"auth_user":      schema.StringAttribute{Optional: true, Sensitive: true, Description: "Git authentication username. Unavailable after import."},
				"auth_token":     schema.StringAttribute{Optional: true, Sensitive: true, Description: "Git access token or password. Unavailable after import."},
				"sync_type":      schema.StringAttribute{Optional: true, Computed: true, Description: "Sync mode. Use manual, polling or webhook. Defaults to manual at creation."},
				"webhook_secret": schema.StringAttribute{Computed: true, Sensitive: true, Description: "Webhook secret returned when webhook sync is enabled."},
				"webhook_url":    schema.StringAttribute{Computed: true, Description: "Webhook URL returned when webhook sync is enabled."},
				"polling_interval_seconds": schema.Int64Attribute{
					Optional: true, Computed: true, Description: "Repository polling interval in seconds. The hub defaults to 60 for polling.",
				},
				"github_actions_oidc_enabled":           optionalBool("Allow GitHub Actions to authenticate with OIDC."),
				"github_actions_oidc_allow_repo_sync":   optionalBool("Allow GitHub Actions to trigger repository syncs."),
				"github_actions_oidc_allow_image_sync":  optionalBool("Allow GitHub Actions to trigger image syncs."),
				"github_actions_oidc_allowed_branches":  optionalSet("Branches allowed for GitHub Actions OIDC. An empty set allows any branch.", equivalentSet{strings.TrimSpace}),
				"github_actions_oidc_allowed_workflows": optionalSet("Workflow files allowed for GitHub Actions OIDC. An empty set allows any workflow.", equivalentSet{strings.TrimSpace}),
				"require_signed_commits":                schema.BoolAttribute{Computed: true, Description: "Whether the hub requires commits signed by a trusted key."},
			},
		},
		create: createRepository,
		read:   readRepository,
		update: updateRepository,
		delete: func(ctx context.Context, client *api.Client, id string) error {
			return api.Delete(client.DeleteRepository(ctx, id))
		},
	}
}

func createRepository(ctx context.Context, client *api.Client, model *repositoryModel) error {
	branches, err := optionalStrings(ctx, model.GitHubActionsOIDCAllowedBranches)
	if err != nil {
		return err
	}
	workflows, err := optionalStrings(ctx, model.GitHubActionsOIDCAllowedWorkflows)
	if err != nil {
		return err
	}

	authMethod := "none"
	if known(model.AuthMethod) {
		authMethod = model.AuthMethod.ValueString()
	}
	syncType := "manual"
	if known(model.SyncType) {
		syncType = model.SyncType.ValueString()
	}

	reply, err := api.Decode[api.RepositoryResponse](client.CreateRepository(ctx, api.CreateRepositoryRequest{
		Url: model.URL.ValueString(), Provider: model.Provider.ValueString(), AuthMethod: authMethod,
		AuthUser: stringPtr(model.AuthUser), AuthToken: stringPtr(model.AuthToken),
		SyncType: syncType, PollingIntervalSeconds: intPtr(model.PollingIntervalSeconds),
		GithubActionsOIDCEnabled:         boolPtr(model.GitHubActionsOIDCEnabled),
		GithubActionsOIDCAllowRepoSync:   boolPtr(model.GitHubActionsOIDCAllowRepoSync),
		GithubActionsOIDCAllowImageSync:  boolPtr(model.GitHubActionsOIDCAllowImageSync),
		GithubActionsOIDCAllowedBranches: branches, GithubActionsOIDCAllowedWorkflows: workflows,
	}))
	if err != nil {
		return err
	}
	return model.setRepository(ctx, reply)
}

func updateRepository(ctx context.Context, client *api.Client, plan *repositoryModel, config, prior repositoryModel) error {
	branches, err := optionalStrings(ctx, config.GitHubActionsOIDCAllowedBranches)
	if err != nil {
		return err
	}
	workflows, err := optionalStrings(ctx, config.GitHubActionsOIDCAllowedWorkflows)
	if err != nil {
		return err
	}

	request := api.UpdateRepositoryRequest{
		GithubActionsOIDCEnabled:         boolPtr(config.GitHubActionsOIDCEnabled),
		GithubActionsOIDCAllowRepoSync:   boolPtr(config.GitHubActionsOIDCAllowRepoSync),
		GithubActionsOIDCAllowImageSync:  boolPtr(config.GitHubActionsOIDCAllowImageSync),
		GithubActionsOIDCAllowedBranches: branches, GithubActionsOIDCAllowedWorkflows: workflows,
	}

	// Sending authMethod replaces both credentials. Only send it for auth changes.
	if (known(config.AuthMethod) && !config.AuthMethod.Equal(prior.AuthMethod)) ||
		!plan.AuthUser.Equal(prior.AuthUser) || !plan.AuthToken.Equal(prior.AuthToken) {
		request.AuthMethod = choose(stringPtr(config.AuthMethod), stringPtr(prior.AuthMethod))
		request.AuthUser = stringPtr(plan.AuthUser)
		request.AuthToken = stringPtr(plan.AuthToken)
	}

	// The hub updates the polling interval only when syncType is also sent.
	if (known(config.SyncType) && !config.SyncType.Equal(prior.SyncType)) ||
		(known(config.PollingIntervalSeconds) && !config.PollingIntervalSeconds.Equal(prior.PollingIntervalSeconds)) {
		request.SyncType = choose(stringPtr(config.SyncType), stringPtr(prior.SyncType))
		request.PollingIntervalSeconds = intPtr(config.PollingIntervalSeconds)
	}

	plan.WebhookSecret = prior.WebhookSecret
	plan.WebhookURL = prior.WebhookURL
	id := prior.ID.ValueString()
	if request == (api.UpdateRepositoryRequest{}) {
		return readRepository(ctx, client, id, plan)
	}
	reply, err := api.Decode[api.RepositoryResponse](client.UpdateRepository(ctx, id, request))
	if err != nil {
		return err
	}
	return plan.setRepository(ctx, reply)
}

func readRepository(ctx context.Context, client *api.Client, id string, model *repositoryModel) error {
	reply, err := api.Decode[api.RepositoryResponse](client.GetRepository(ctx, id))
	if err != nil {
		return err
	}
	return model.setRepository(ctx, reply)
}

func (m *repositoryModel) setRepository(ctx context.Context, reply api.RepositoryResponse) error {
	if reply.WebhookSecret != nil || reply.SyncType != "webhook" {
		m.WebhookSecret = types.StringPointerValue(reply.WebhookSecret)
		m.WebhookURL = types.StringPointerValue(reply.WebhookUrl)
	}
	return m.repositoryDetails.setRepository(ctx, reply)
}

func (d *repositoryDetails) setRepository(ctx context.Context, reply api.RepositoryResponse) error {
	if !known(d.URL) || normalizeRepositoryURL(d.URL.ValueString()) != reply.Url {
		d.URL = types.StringValue(reply.Url)
	}
	d.Provider = types.StringValue(reply.Provider)
	d.AuthMethod = types.StringValue(reply.AuthMethod)
	d.SyncType = types.StringValue(reply.SyncType)
	d.PollingIntervalSeconds = types.Int64PointerValue(reply.PollingIntervalSeconds)
	d.GitHubActionsOIDCEnabled = types.BoolValue(reply.GithubActionsOIDCEnabled)
	d.GitHubActionsOIDCAllowRepoSync = types.BoolValue(reply.GithubActionsOIDCAllowRepoSync)
	d.GitHubActionsOIDCAllowImageSync = types.BoolValue(reply.GithubActionsOIDCAllowImageSync)
	d.RequireSignedCommits = types.BoolValue(reply.RequireSignedCommits)

	branches := stringsValue(reply.GithubActionsOIDCAllowedBranches)
	if !equivalentStrings(ctx, d.GitHubActionsOIDCAllowedBranches, branches, strings.TrimSpace) {
		d.GitHubActionsOIDCAllowedBranches = branches
	}
	workflows := stringsValue(reply.GithubActionsOIDCAllowedWorkflows)
	if !equivalentStrings(ctx, d.GitHubActionsOIDCAllowedWorkflows, workflows, strings.TrimSpace) {
		d.GitHubActionsOIDCAllowedWorkflows = workflows
	}
	return d.set(reply.Id, reply.Name)
}

func normalizeRepositoryURL(raw string) string {
	return strings.TrimRight(strings.TrimSuffix(strings.TrimRight(strings.TrimSpace(raw), "/"), ".git"), "/")
}

func newRepositoryDataSource() datasource.DataSource {
	return &lookupSource[repositoryDetails]{
		kind: "repository",
		attributes: map[string]dsschema.Attribute{
			"url":                                   dsschema.StringAttribute{Computed: true, Description: "Git repository URL."},
			"git_provider":                          dsschema.StringAttribute{Computed: true, Description: "Git provider."},
			"auth_method":                           dsschema.StringAttribute{Computed: true, Description: "Git authentication method."},
			"sync_type":                             dsschema.StringAttribute{Computed: true, Description: "Repository sync mode."},
			"polling_interval_seconds":              dsschema.Int64Attribute{Computed: true, Description: "Effective repository polling interval in seconds."},
			"github_actions_oidc_enabled":           dsschema.BoolAttribute{Computed: true, Description: "Whether GitHub Actions OIDC is enabled."},
			"github_actions_oidc_allow_repo_sync":   dsschema.BoolAttribute{Computed: true, Description: "Whether GitHub Actions may trigger repository syncs."},
			"github_actions_oidc_allow_image_sync":  dsschema.BoolAttribute{Computed: true, Description: "Whether GitHub Actions may trigger image syncs."},
			"github_actions_oidc_allowed_branches":  computedSet("Branches allowed for GitHub Actions OIDC."),
			"github_actions_oidc_allowed_workflows": computedSet("Workflow files allowed for GitHub Actions OIDC."),
			"require_signed_commits":                dsschema.BoolAttribute{Computed: true, Description: "Whether commits must be signed by a trusted key."},
		},
		find: func(ctx context.Context, client *api.Client, name string) (string, error) {
			items, err := api.Decode[[]api.RepositoryResponse](client.ListRepositories(ctx))
			if err != nil {
				return "", err
			}
			return findByName(items, name, func(item api.RepositoryResponse) (string, string) { return item.Id, item.Name })
		},
		read: func(ctx context.Context, client *api.Client, id string, model *repositoryDetails) error {
			reply, err := api.Decode[api.RepositoryResponse](client.GetRepository(ctx, id))
			if err != nil {
				return err
			}
			return model.setRepository(ctx, reply)
		},
	}
}
