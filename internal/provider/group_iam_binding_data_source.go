package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/verily-src/terraform-provider-workbench/internal/api"
	"github.com/verily-src/terraform-provider-workbench/internal/models"
	"github.com/verily-src/terraform-provider-workbench/internal/openapi/user"
	"github.com/verily-src/terraform-provider-workbench/internal/schemas"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ datasource.DataSource              = &GroupIamBindingDataSource{}
	_ datasource.DataSourceWithConfigure = &GroupIamBindingDataSource{}
)

// NewGroupIamDataSource initializes a new WSM IAM member data source.
func NewGroupIamBindingDataSource() datasource.DataSource {
	return &GroupIamBindingDataSource{}
}

// GroupIamBindingDataSource defines the data source implementation.
type GroupIamBindingDataSource struct {
	client *ClientConfig
}

// Metadata returns the data source type name.
func (d *GroupIamBindingDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_iam_binding"
}

// Schema defines the data source-level schema for configuration.
func (d *GroupIamBindingDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"group": schemas.GroupDataSourceSchema,
			"organization": schema.StringAttribute{
				MarkdownDescription: "Workbench organization ID, either UUID or UFID. If it is a UFID, it must be prefixed with a tilde (~).",
				Optional:            true,
			},
			"role": schema.StringAttribute{
				MarkdownDescription: "Workbench IAM role of the member",
				Required:            true,
				Validators: []validator.String{
					groupIamRoleValidator(),
				},
			},
			"principals": schema.ListNestedAttribute{
				Description: "List of principals (users, groups, or public) in the IAM binding.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"user": schema.StringAttribute{
							Description: "Email of a user.",
							Optional:    true,
							Computed:    true,
						},
						"group": schema.SingleNestedAttribute{
							Description: "Identifier of a group.",
							Optional:    true,
							Computed:    true,
							Attributes: map[string]schema.Attribute{
								"group": schema.StringAttribute{
									Description: "Name of the group.",
									Computed:    true,
								},
								"organization": schema.StringAttribute{
									Description: "Workbench organization ID.",
									Computed:    true,
								},
							},
						},
						"public": schema.BoolAttribute{
							Description: "True if the group is public.",
							Optional:    true,
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// Configure initializes the client for the data source with the configuration.
func (d *GroupIamBindingDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *GroupIamBindingDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data models.GroupIamBindingModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create a new client
	c, err := api.NewUserClient(ctx, d.client.Host, d.client.UseIdToken, d.client.ImpersonateServiceAccount)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Workbench client, unexpected error: %s", err))
		return
	}

	groupNameParam, orgIDParam := data.Params()
	roles, err := api.GetGroupRoles(ctx, c, groupNameParam, orgIDParam)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Reading workspace roles into data, got error: %s", err))
		return
	}

	state := models.NewGroupIamBindingModel(data.GroupName.ValueString(), data.OrgID.ValueString(), roles, user.GroupRole(data.Role.ValueString()))
	tflog.Debug(ctx, fmt.Sprintf("Read group IAM binding data source: %v", state))
	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}
