// package provider provides the implementation of the Workbench provider for Terraform.
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
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure WorkbenchProvider satisfies various provider interfaces.
var _ provider.Provider = &WorkbenchProvider{}

// WorkbenchProvider defines the provider implementation.
type WorkbenchProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// ClientConfig holds the configuration for the Workbench client.
type ClientConfig struct {
	// Host is the Workbench server host URL.
	// It defaults to "https://workbench.verily.com" if not set.
	Host string
	// UseIdToken indicates whether to use an ID token for authentication.
	// Local development should set this to false.
	UseIdToken bool
	// ImpersonateServiceAccount is the service account email to impersonate.
	// If set, the provider will use the default credentials to impersonate this service account.
	ImpersonateServiceAccount string
}

type workbenchProviderModel struct {
	Host                      types.String `tfsdk:"host"`
	UseIdToken                types.Bool   `tfsdk:"use_id_token"`
	ImpersonateServiceAccount types.String `tfsdk:"impersonate_service_account"`
}

// Metadata returns the provider type name and version.
func (p *WorkbenchProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "workbench"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration.
func (p *WorkbenchProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				MarkdownDescription: "example of a Workbench server is https://workbench.verily.com",
				Optional:            true,
			},
			"use_id_token": schema.BoolAttribute{
				MarkdownDescription: "Set to true to use an ID token for authentication. Set to false for local development.",
				Optional:            true,
			},
			"impersonate_service_account": schema.StringAttribute{
				MarkdownDescription: "Service account email to impersonate using default credentials. When set, the provider will use application default credentials to impersonate this service account.",
				Optional:            true,
			},
		},
	}
}

// Configure initializes the client for the provider with the configuration.
func (p *WorkbenchProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring Workbench Provider")

	// Retrieve provider data from the configuration.
	var data workbenchProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.Host.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("host"),
			"Unknown Workbench API Host",
			"The provider cannot create the Workbench API client as there is an unknown configuration value for the Workbench API host. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the WORKBENCH_HOST environment variable.",
		)
	}

	host := os.Getenv("WORKBENCH_HOST")
	// Configuration values are now available.
	if !data.Host.IsNull() {
		host = data.Host.ValueString()
	}

	if host == "" {
		host = "https://workbench.verily.com"
	}

	// Handle UseIdToken with default value of false
	useIdToken := false
	if !data.UseIdToken.IsNull() {
		useIdToken = data.UseIdToken.ValueBool()
	}

	// Handle ImpersonateServiceAccount
	impersonateServiceAccount := ""
	if !data.ImpersonateServiceAccount.IsNull() {
		impersonateServiceAccount = data.ImpersonateServiceAccount.ValueString()
	}

	ctx = tflog.SetField(ctx, "host", host)
	ctx = tflog.SetField(ctx, "use_id_token", useIdToken)
	ctx = tflog.SetField(ctx, "impersonate_service_account", impersonateServiceAccount)
	tflog.Debug(ctx, "Creating Workbench client")

	client := &ClientConfig{
		Host:                      host,
		UseIdToken:                useIdToken,
		ImpersonateServiceAccount: impersonateServiceAccount,
	}

	resp.DataSourceData = client
	resp.ResourceData = client

	tflog.Info(ctx, "Configured Workbench client", map[string]any{"success": true})
}

// Resources defines the resources available in the provider.
func (p *WorkbenchProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewWorkspaceResource,
		NewDataCollectionResource,
		NewGroupResource,
		NewGroupIamPolicyResource,
		NewGroupIamBindingResource,
		NewGroupIamMemberResource,
		NewWorkspaceIamPolicyResource,
		NewWorkspaceIamBindingResource,
		NewWorkspaceIamMemberResource,
		NewFolderResource,
		NewVersionResource,
		NewControlledGcsBucketResource,
		NewPerimeterResource,
	}
}

// DataSources defines the data sources available in the provider.
func (p *WorkbenchProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewWorkspaceDataSource,
		NewDataCollectionDataSource,
		NewGroupDataSource,
		NewGroupIamBindingDataSource,
		NewGroupIamPolicyDataSource,
		NewWorkspaceIamPolicyDataSource,
		NewWorkspaceIamBindingDataSource,
		NewControlledGcsBucketDataSource,
	}
}

// New creates a new Workbench provider with the specified version.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &WorkbenchProvider{
			version: version,
		}
	}
}
