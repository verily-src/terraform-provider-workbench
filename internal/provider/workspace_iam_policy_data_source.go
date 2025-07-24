package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/verily-src/terraform-provider-workbench/internal/api"
	"github.com/verily-src/terraform-provider-workbench/internal/models"
	"github.com/verily-src/terraform-provider-workbench/internal/schemas"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ datasource.DataSource              = &WorkspaceIamPolicyDataSource{}
	_ datasource.DataSourceWithConfigure = &WorkspaceIamPolicyDataSource{}
)

// NewWorkspaceIamDataSource iniitializes a new WSM IAM member data source.
func NewWorkspaceIamPolicyDataSource() datasource.DataSource {
	return &WorkspaceIamPolicyDataSource{}
}

// WorkspaceIamPolicyDataSource defines the data source implementation.
type WorkspaceIamPolicyDataSource struct {
	client *ClientConfig
}

// Metadata returns the data source type name.
func (d *WorkspaceIamPolicyDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workspace_iam_policy"
}

// Schema defines the data source-level schema for configuration.
func (d *WorkspaceIamPolicyDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"workspace_id": schemas.WorkspaceDataSourceSchema,
			"iams": schema.SetNestedAttribute{
				MarkdownDescription: "Workbench workspace user IAM policies",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"role": schema.StringAttribute{
							MarkdownDescription: "Workbench IAM role of the member",
							Computed:            true,
						},
						"members": schema.SetAttribute{
							MarkdownDescription: "List of members in the IAM binding",
							Computed:            true,
							ElementType:         types.StringType,
						},
					},
				},
			},
		},
	}
}

// Configure initializes the client for the data source with the configuration.
func (d *WorkspaceIamPolicyDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*ClientConfig)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *wsm.ClientWithResponse, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

// Read is called to read the data source.
func (d *WorkspaceIamPolicyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config models.WorkspaceIamPolicyModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create a new client
	c, err := api.NewWSMClient(ctx, d.client.Host)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Workbench client, unexpected error: %s", err))
		return
	}

	r, err := api.GetRoles(ctx, c, config.WorkspaceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Reading workspace roles into data, got error: %s", err))
		return
	}

	state := models.NewWorkspaceIamPolicyModel(config.WorkspaceID.ValueString(), r)
	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}
