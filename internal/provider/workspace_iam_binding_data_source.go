package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/verily-src/terraform-provider-workbench/internal/api"
	"github.com/verily-src/terraform-provider-workbench/internal/models"
	"github.com/verily-src/terraform-provider-workbench/internal/openapi/wsm"
	"github.com/verily-src/terraform-provider-workbench/internal/schemas"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ datasource.DataSource              = &WorkspaceIamBindingDataSource{}
	_ datasource.DataSourceWithConfigure = &WorkspaceIamBindingDataSource{}
)

// NewWorkspaceIamDataSource iniitializes a new WSM IAM member data source.
func NewWorkspaceIamBindingDataSource() datasource.DataSource {
	return &WorkspaceIamBindingDataSource{}
}

// WorkspaceIamBindingDataSource defines the data source implementation.
type WorkspaceIamBindingDataSource struct {
	client *ClientConfig
}

// Metadata returns the data source type name.
func (d *WorkspaceIamBindingDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workspace_iam_binding"
}

// Schema defines the data source-level schema for configuration.
func (d *WorkspaceIamBindingDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"workspace_id": schemas.WorkspaceDataSourceSchema,
			"role": schema.StringAttribute{
				MarkdownDescription: "Workbench IAM role of the member",
				Required:            true,
			},
			"members": schema.SetAttribute{
				MarkdownDescription: "List of members in the IAM binding",
				Computed:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

// Configure initializes the client for the data source with the configuration.
func (d *WorkspaceIamBindingDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *WorkspaceIamBindingDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data models.WorkspaceIamBindingModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create a new client
	c, err := api.NewWSMClient(ctx, d.client.Host)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Workbench client, unexpected error: %s", err))
		return
	}

	r, err := api.GetRoles(ctx, c, data.WorkspaceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Reading workspace roles into data, got error: %s", err))
		return
	}

	roleBinding := &wsm.RoleBinding{
		Role:    wsm.IamRole(data.Role.ValueString()),
		Members: nil,
	}

	for _, rb := range *r {
		if rb.Role == roleBinding.Role {
			roleBinding.Members = rb.Members
			break
		}
	}

	state := models.NewWorkspaceIamBindingModel(data.WorkspaceID.ValueString(), roleBinding)
	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}
