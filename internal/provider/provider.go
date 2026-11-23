package provider

import (
	"context"
	"os"
	"strings"

	"github.com/OrcaCD/terraform-provider-orcacd/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type orcaProvider struct {
	version string
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &orcaProvider{version: version}
	}
}

func (p *orcaProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "orcacd"
	resp.Version = p.version
}

type providerConfig struct {
	URL   types.String `tfsdk:"url"`
	Token types.String `tfsdk:"token"`
}

func (p *orcaProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage repositories, applications, notifications, and agents on an OrcaCD hub.",
		Attributes: map[string]schema.Attribute{
			"url": schema.StringAttribute{
				Optional: true, Description: "Base URL of the OrcaCD hub. Defaults to ORCACD_URL.",
			},
			"token": schema.StringAttribute{
				Optional: true, Sensitive: true, Description: "OrcaCD API token. Defaults to ORCACD_TOKEN.",
			},
		},
	}
}

func (p *orcaProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config providerConfig
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.URL.IsUnknown() {
		resp.Diagnostics.AddAttributeError(path.Root("url"), "Unknown OrcaCD hub URL", "Set the hub URL to a value known during planning.")
	}
	if config.Token.IsUnknown() {
		resp.Diagnostics.AddAttributeError(path.Root("token"), "Unknown OrcaCD API token", "Set the API token to a value known during planning.")
	}
	if resp.Diagnostics.HasError() {
		return
	}

	hubURL := os.Getenv("ORCACD_URL")
	if !config.URL.IsNull() {
		hubURL = config.URL.ValueString()
	}
	token := os.Getenv("ORCACD_TOKEN")
	if !config.Token.IsNull() {
		token = config.Token.ValueString()
	}

	if strings.TrimSpace(hubURL) == "" {
		resp.Diagnostics.AddAttributeError(path.Root("url"), "Missing OrcaCD hub URL", "Set url or the ORCACD_URL environment variable.")
	}
	token = strings.TrimSpace(token)
	if token == "" {
		resp.Diagnostics.AddAttributeError(path.Root("token"), "Missing OrcaCD API token", "Set token or the ORCACD_TOKEN environment variable.")
	} else if strings.ContainsAny(token, " \t\r\n") {
		resp.Diagnostics.AddAttributeError(path.Root("token"), "Invalid OrcaCD API token", "Use a single token value.")
	}
	if resp.Diagnostics.HasError() {
		return
	}

	client, err := api.New(hubURL, token)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("url"), "Invalid OrcaCD hub URL", err.Error())
		return
	}
	resp.ResourceData = client
	resp.DataSourceData = client
}

func (p *orcaProvider) Resources(context.Context) []func() resource.Resource {
	return []func() resource.Resource{newAgent, newApplication, newNotification, newRepository}
}

func (p *orcaProvider) DataSources(context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{newAgentDataSource, newApplicationDataSource, newNotificationDataSource, newRepositoryDataSource}
}
