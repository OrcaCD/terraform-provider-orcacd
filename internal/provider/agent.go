package provider

import (
	"context"
	"net/netip"
	"strings"

	"github.com/OrcaCD/terraform-provider-orcacd/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	defaultAgentIcon      = "server"
	allowedIPsDescription = "IP addresses and CIDR ranges the agent may connect from. An empty set allows any IP."
)

type agentDetails struct {
	objectID
	Icon       types.String `tfsdk:"icon"`
	AllowedIPs types.Set    `tfsdk:"allowed_ips"`
}

type agentModel struct {
	agentDetails
	Token types.String `tfsdk:"token"`
}

type agentLookup struct {
	agentDetails
	Status  types.String `tfsdk:"status"`
	Version types.String `tfsdk:"version"`
}

func newAgent() resource.Resource {
	return &managedResource[agentModel]{
		kind: "agent",
		schema: schema.Schema{
			Description: "Register an agent and obtain its one-time connection token.",
			Attributes: map[string]schema.Attribute{
				"id":          idAttribute(),
				"name":        schema.StringAttribute{Required: true, Description: "Agent name."},
				"icon":        schema.StringAttribute{Optional: true, Computed: true, Description: "Agent icon. Defaults to server at creation.", PlanModifiers: []planmodifier.String{equivalentString{normalizeAgentIcon}}},
				"allowed_ips": optionalSet(allowedIPsDescription, equivalentSet{normalizeIP}),
				"token": schema.StringAttribute{
					Computed: true, Sensitive: true, Description: "Connection token returned at creation. Unavailable after import.",
				},
			},
		},
		create: createAgent,
		read:   readAgent,
		update: updateAgent,
		delete: func(ctx context.Context, client *api.Client, id string) error {
			return api.Delete(client.DeleteAgent(ctx, id))
		},
	}
}

func createAgent(ctx context.Context, client *api.Client, model *agentModel) error {
	ips, err := optionalStrings(ctx, model.AllowedIPs)
	if err != nil {
		return err
	}

	reply, err := api.Decode[api.AgentWithTokenResponse](client.CreateAgent(ctx, api.CreateAgentRequest{
		Name: model.Name.ValueString(), Icon: agentIconRequest(model.Icon), AllowedIps: ips,
	}))
	if err != nil {
		return err
	}
	model.Token = types.StringPointerValue(reply.AuthToken)
	return model.setAgent(ctx, reply.Id, reply.Name, reply.Icon, reply.AllowedIps)
}

func updateAgent(ctx context.Context, client *api.Client, plan *agentModel, config, prior agentModel) error {
	ips, err := optionalStrings(ctx, config.AllowedIPs)
	if err != nil {
		return err
	}
	reply, err := api.Decode[api.AgentResponse](client.UpdateAgent(ctx, prior.ID.ValueString(), api.UpdateAgentRequest{
		Name: plan.Name.ValueString(), Icon: agentIconRequest(config.Icon), AllowedIps: ips,
	}))
	if err != nil {
		return err
	}
	plan.Token = prior.Token
	return plan.setAgent(ctx, reply.Id, reply.Name, reply.Icon, reply.AllowedIps)
}

func readAgent(ctx context.Context, client *api.Client, id string, model *agentModel) error {
	reply, err := api.Decode[api.AgentResponse](client.GetAgent(ctx, id))
	if err != nil {
		return err
	}
	return model.setAgent(ctx, reply.Id, reply.Name, reply.Icon, reply.AllowedIps)
}

func (d *agentDetails) setAgent(ctx context.Context, id, name, icon string, ips *[]string) error {
	if err := d.set(id, name); err != nil {
		return err
	}
	if !known(d.Icon) || normalizeAgentIcon(d.Icon.ValueString()) != icon {
		d.Icon = types.StringValue(icon)
	}
	actual := stringsValue(ips)
	// Keep the configured spelling when the hub stores the same addresses in canonical form.
	if !equivalentStrings(ctx, d.AllowedIPs, actual, normalizeIP) {
		d.AllowedIPs = actual
	}
	return nil
}

func agentIconRequest(icon types.String) *string {
	if !known(icon) {
		return nil
	}
	value := normalizeAgentIcon(icon.ValueString())
	return &value
}

func normalizeAgentIcon(icon string) string {
	if icon == "" {
		return defaultAgentIcon
	}
	return icon
}

func normalizeIP(value string) string {
	value = strings.TrimSpace(value)
	if prefix, err := netip.ParsePrefix(value); err == nil {
		if prefix.Addr().Is4In6() && prefix.Bits() >= 96 {
			prefix = netip.PrefixFrom(prefix.Addr().Unmap(), prefix.Bits()-96)
		}
		return prefix.Masked().String()
	}
	address, _, _ := strings.Cut(value, "%")
	if ip, err := netip.ParseAddr(address); err == nil {
		return ip.Unmap().String()
	}
	return value
}

func newAgentDataSource() datasource.DataSource {
	return &lookupSource[agentLookup]{
		kind: "agent",
		attributes: map[string]dsschema.Attribute{
			"icon":        dsschema.StringAttribute{Computed: true, Description: "Agent icon."},
			"allowed_ips": dsschema.SetAttribute{Computed: true, ElementType: types.StringType, Description: allowedIPsDescription},
			"status":      dsschema.StringAttribute{Computed: true, Description: "Current connection status."},
			"version":     dsschema.StringAttribute{Computed: true, Description: "Version reported by the agent."},
		},
		find: func(ctx context.Context, client *api.Client, name string) (string, error) {
			items, err := api.Decode[[]api.AgentResponse](client.ListAgents(ctx))
			if err != nil {
				return "", err
			}
			return findByName(items, name, func(item api.AgentResponse) (string, string) { return item.Id, item.Name })
		},
		read: func(ctx context.Context, client *api.Client, id string, model *agentLookup) error {
			reply, err := api.Decode[api.AgentResponse](client.GetAgent(ctx, id))
			if err != nil {
				return err
			}
			model.Status = types.StringValue(string(reply.Status))
			model.Version = types.StringPointerValue(reply.Version)
			return model.setAgent(ctx, reply.Id, reply.Name, reply.Icon, reply.AllowedIps)
		},
	}
}
