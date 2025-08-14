package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/verily-src/terraform-provider-workbench/internal/api"
	"github.com/verily-src/terraform-provider-workbench/internal/models"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ datasource.DataSource              = &GroupIamPolicyDataSource{}
	_ datasource.DataSourceWithConfigure = &GroupIamPolicyDataSource{}
)

// NewGroupIamPolicyDataSource initializes a new WSM IAM member data source.
func NewGroupIamPolicyDataSource() datasource.DataSource {
	return &GroupIamPolicyDataSource{}
}

// GroupIamPolicyDataSource defines the data source implementation.
type GroupIamPolicyDataSource struct {
	client *ClientConfig
}

// Metadata returns the data source type name.
func (d *GroupIamPolicyDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_iam_policy"
}

// Schema defines the data source-level schema for configuration.
func (d *GroupIamPolicyDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"group": schema.StringAttribute{
				MarkdownDescription: "Workbench group name",
				Required:            true,
			},
			"organization": schema.StringAttribute{
				MarkdownDescription: "Workbench organization ID, either UUID or UFID. If it is a UFID, it must be prefixed with a tilde (~).",
				Optional:            true,
			},
			"iams": schema.SetNestedAttribute{
				MarkdownDescription: "Workbench group user IAM policies",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"role": schema.StringAttribute{
							MarkdownDescription: "Workbench IAM role of the member",
							Computed:            true,
						},
						"principals": schema.SetNestedAttribute{
							Description: "List of principals (user, group, or public) in the IAM binding.",
							Computed:    true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"user": schema.StringAttribute{
										Description: "Email of a user.",
										Computed:    true,
									},
									"group": schema.SingleNestedAttribute{
										Description: "Identifier of a group.",
										Computed:    true,
										Attributes: map[string]schema.Attribute{
											"group": schema.StringAttribute{
												Description: "Name of the group.",
												Computed:    true,
											},
											"organization": schema.StringAttribute{
												Description: "UxID of the organization. If it is a UFID, it must be prefixed with a tilde (~).",
												Computed:    true,
											},
										},
									},
									"public": schema.BoolAttribute{
										Description: "True if the group is public.",
										Computed:    true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// Configure initializes the client for the data source with the configuration.
func (d *GroupIamPolicyDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *GroupIamPolicyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data models.GroupIamPolicyModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create a new client
	c, err := api.NewUserClient(ctx, d.client.Host, d.client.UseIdToken)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Workbench client, unexpected error: %s", err))
		return
	}

	groupNameParam, orgIDParams := data.Params()
	roles, err := api.GetGroupRoles(ctx, c, groupNameParam, orgIDParams)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Reading group roles into data, got error: %s", err))
		return
	}

	state := models.NewGroupIamPolicyModel(data.GroupName.ValueString(), data.OrgID.ValueString(), roles)
	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}
