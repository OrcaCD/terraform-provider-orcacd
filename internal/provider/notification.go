package provider

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/OrcaCD/terraform-provider-orcacd/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type notificationDetails struct {
	objectID
	Type            types.String `tfsdk:"type"`
	Enabled         types.Bool   `tfsdk:"enabled"`
	AllApplications types.Bool   `tfsdk:"all_applications"`
	AllAgents       types.Bool   `tfsdk:"all_agents"`
	AllRepositories types.Bool   `tfsdk:"all_repositories"`
	ApplicationIDs  types.Set    `tfsdk:"application_ids"`
	AgentIDs        types.Set    `tfsdk:"agent_ids"`
	RepositoryIDs   types.Set    `tfsdk:"repository_ids"`
	Events          types.Set    `tfsdk:"events"`
}

type notificationModel struct {
	notificationDetails
	ConfigJSON types.String `tfsdk:"config_json"`
}

func newNotification() resource.Resource {
	return &managedResource[notificationModel]{
		kind: "notification",
		schema: schema.Schema{
			Description: "Manage an OrcaCD notification and its subscriptions.",
			Attributes: map[string]schema.Attribute{
				"id":   idAttribute(),
				"name": replacementString("Notification name.", true, false, equivalentString{strings.TrimSpace}),
				"type": replacementString("Notification provider, such as webhook, slack or discord.", true, false),
				"config_json": replacementString(
					"Provider settings as a JSON object or JSON-encoded string. Required to create a notification. Unavailable after import.", false, true,
				),
				"enabled":          optionalBool("Whether notifications are enabled. The hub defaults to true."),
				"all_applications": optionalBool("Subscribe to all applications."),
				"all_agents":       optionalBool("Subscribe to all agents."),
				"all_repositories": optionalBool("Subscribe to all repositories."),
				"application_ids":  optionalSet("Application IDs to subscribe to. Use with all_applications unset or false."),
				"agent_ids":        optionalSet("Agent IDs to subscribe to. Use with all_agents unset or false."),
				"repository_ids":   optionalSet("Repository IDs to subscribe to. Use with all_repositories unset or false."),
				"events":           optionalSet("Events to send. The hub subscribes to all events when omitted at creation."),
			},
		},
		create: createNotification,
		read:   readNotification,
		update: updateNotification,
		delete: func(ctx context.Context, client *api.Client, id string) error {
			return api.Delete(client.DeleteNotification(ctx, id))
		},
	}
}

func createNotification(ctx context.Context, client *api.Client, model *notificationModel) error {
	config := strings.TrimSpace(model.ConfigJSON.ValueString())
	if !json.Valid([]byte(config)) || (config[0] != '{' && config[0] != '"') {
		return errors.New("config_json must be a JSON object or JSON-encoded string when creating a notification")
	}

	request := api.CreateNotificationRequest{
		Name: model.Name.ValueString(), Type: model.Type.ValueString(), Config: json.RawMessage(config),
		Enabled: boolPtr(model.Enabled), AllApplications: boolPtr(model.AllApplications),
		AllAgents: boolPtr(model.AllAgents), AllRepositories: boolPtr(model.AllRepositories),
	}
	if err := notificationSets(ctx, model.notificationDetails, &request); err != nil {
		return err
	}

	reply, err := api.Decode[api.NotificationResponse](client.CreateNotification(ctx, request))
	if err != nil {
		return err
	}
	return model.setNotification(reply)
}

func updateNotification(ctx context.Context, client *api.Client, plan *notificationModel, config, prior notificationModel) error {
	id := prior.ID.ValueString()
	current, err := getNotification(ctx, client, id)
	if err != nil {
		return err
	}

	// The hub requires the complete subscription when writing any part of it.
	request := api.CreateNotificationRequest{
		AllApplications: boolPtr(config.AllApplications),
		AllAgents:       boolPtr(config.AllAgents),
		AllRepositories: boolPtr(config.AllRepositories),
	}
	if err := notificationSets(ctx, config.notificationDetails, &request); err != nil {
		return err
	}
	allApplications := valueOr(config.AllApplications, current.AllApplications)
	allAgents := valueOr(config.AllAgents, current.AllAgents)
	allRepositories := valueOr(config.AllRepositories, current.AllRepositories)

	applicationIDs, err := notificationTargets("applications", allApplications, request.ApplicationIds, current.ApplicationIds)
	if err != nil {
		return err
	}
	agentIDs, err := notificationTargets("agents", allAgents, request.AgentIds, current.AgentIds)
	if err != nil {
		return err
	}
	repositoryIDs, err := notificationTargets("repositories", allRepositories, request.RepositoryIds, current.RepositoryIds)
	if err != nil {
		return err
	}

	events := choose(request.Events, current.Events)
	if events == nil {
		return errors.New("the hub did not return the existing notification events; configure events explicitly")
	}
	reply, err := api.Decode[api.NotificationResponse](client.UpdateNotification(ctx, id, api.UpdateNotificationRequest{
		Enabled: valueOr(config.Enabled, current.Enabled), Events: *events,
		AllApplications: allApplications, AllAgents: allAgents, AllRepositories: allRepositories,
		ApplicationIds: applicationIDs, AgentIds: agentIDs, RepositoryIds: repositoryIDs,
	}))
	if err != nil {
		return err
	}
	return plan.setNotification(reply)
}

func notificationSets(ctx context.Context, input notificationDetails, request *api.CreateNotificationRequest) error {
	var err error
	if request.ApplicationIds, err = optionalStrings(ctx, input.ApplicationIDs); err != nil {
		return err
	}
	if request.AgentIds, err = optionalStrings(ctx, input.AgentIDs); err != nil {
		return err
	}
	if request.RepositoryIds, err = optionalStrings(ctx, input.RepositoryIDs); err != nil {
		return err
	}
	if request.Events, err = optionalStrings(ctx, input.Events); err != nil {
		return err
	}

	if request.Events != nil && len(*request.Events) == 0 {
		return errors.New("events must contain at least one event when configured")
	}
	for _, scope := range []struct {
		name string
		all  *bool
		ids  *[]string
	}{
		{"applications", request.AllApplications, request.ApplicationIds},
		{"agents", request.AllAgents, request.AgentIds},
		{"repositories", request.AllRepositories, request.RepositoryIds},
	} {
		if scope.all != nil {
			if _, err := notificationTargets(scope.name, *scope.all, scope.ids, nil); err != nil {
				return err
			}
		}
	}
	return nil
}

func notificationTargets(kind string, all bool, configured, current *[]string) (*[]string, error) {
	if !all {
		return choose(configured, current), nil
	}
	if configured != nil && len(*configured) != 0 {
		return nil, errors.New("individual " + kind + " cannot be configured while subscribed to all " + kind)
	}
	return nil, nil
}

func getNotification(ctx context.Context, client *api.Client, id string) (api.NotificationResponse, error) {
	items, err := api.Decode[[]api.NotificationResponse](client.ListNotifications(ctx))
	if err != nil {
		return api.NotificationResponse{}, err
	}
	for _, item := range items {
		if item.Id == id {
			return item, nil
		}
	}
	return api.NotificationResponse{}, api.ErrNotFound
}

func readNotification(ctx context.Context, client *api.Client, id string, model *notificationModel) error {
	reply, err := getNotification(ctx, client, id)
	if err != nil {
		return err
	}
	return model.setNotification(reply)
}

func (m *notificationDetails) setNotification(reply api.NotificationResponse) error {
	name := reply.Name
	if known(m.Name) && strings.TrimSpace(m.Name.ValueString()) == name {
		name = m.Name.ValueString()
	}
	m.Type = types.StringValue(reply.Type)
	m.Enabled = types.BoolValue(reply.Enabled)
	m.AllApplications = types.BoolValue(reply.AllApplications)
	m.AllAgents = types.BoolValue(reply.AllAgents)
	m.AllRepositories = types.BoolValue(reply.AllRepositories)
	m.ApplicationIDs = stringsValue(reply.ApplicationIds)
	m.AgentIDs = stringsValue(reply.AgentIds)
	m.RepositoryIDs = stringsValue(reply.RepositoryIds)
	m.Events = stringsValue(reply.Events)
	return m.set(reply.Id, name)
}

func newNotificationDataSource() datasource.DataSource {
	return &lookupSource[notificationDetails]{
		kind: "notification",
		attributes: map[string]dsschema.Attribute{
			"type":             dsschema.StringAttribute{Computed: true, Description: "Notification provider."},
			"enabled":          dsschema.BoolAttribute{Computed: true, Description: "Whether notifications are enabled."},
			"all_applications": dsschema.BoolAttribute{Computed: true, Description: "Whether all applications are subscribed."},
			"all_agents":       dsschema.BoolAttribute{Computed: true, Description: "Whether all agents are subscribed."},
			"all_repositories": dsschema.BoolAttribute{Computed: true, Description: "Whether all repositories are subscribed."},
			"application_ids":  computedSet("Application IDs visible to the token."),
			"agent_ids":        computedSet("Agent IDs visible to the token."),
			"repository_ids":   computedSet("Repository IDs visible to the token."),
			"events":           computedSet("Events this notification listens for."),
		},
		find: func(ctx context.Context, client *api.Client, name string) (string, error) {
			items, err := api.Decode[[]api.NotificationResponse](client.ListNotifications(ctx))
			if err != nil {
				return "", err
			}
			return findByName(items, name, func(item api.NotificationResponse) (string, string) { return item.Id, item.Name })
		},
		read: func(ctx context.Context, client *api.Client, id string, model *notificationDetails) error {
			reply, err := getNotification(ctx, client, id)
			if err != nil {
				return err
			}
			return model.setNotification(reply)
		},
	}
}

func replacementString(description string, required, sensitive bool, modifiers ...planmodifier.String) schema.StringAttribute {
	return schema.StringAttribute{
		Required: required, Optional: !required, Sensitive: sensitive, Description: description,
		PlanModifiers: append(modifiers, stringplanmodifier.RequiresReplace()),
	}
}

func optionalSet(description string, modifiers ...planmodifier.Set) schema.SetAttribute {
	return schema.SetAttribute{Optional: true, Computed: true, ElementType: types.StringType, Description: description, PlanModifiers: modifiers}
}

func computedSet(description string) dsschema.SetAttribute {
	return dsschema.SetAttribute{Computed: true, ElementType: types.StringType, Description: description}
}

func optionalBool(description string) schema.BoolAttribute {
	return schema.BoolAttribute{Optional: true, Computed: true, Description: description}
}

func valueOr(value types.Bool, fallback bool) bool {
	if !known(value) {
		return fallback
	}
	return value.ValueBool()
}

func choose[T any](preferred, fallback *T) *T {
	if preferred != nil {
		return preferred
	}
	return fallback
}
