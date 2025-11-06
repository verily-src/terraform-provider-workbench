package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/verily-src/terraform-provider-workbench/internal/api"
	"github.com/verily-src/terraform-provider-workbench/internal/models"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ datasource.DataSource              = &GroupDataSource{}
	_ datasource.DataSourceWithConfigure = &GroupDataSource{}
)

// NewGroupDataSource initializes a new workspace data source.
func NewGroupDataSource() datasource.DataSource {
	return &GroupDataSource{}
}

// GroupDataSource defines the data source implementation.
type GroupDataSource struct {
	client *ClientConfig
}

// Metadata returns the data source type name.
func (d *GroupDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

// Schema defines the data source-level schema for configuration.
func (d *GroupDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"group_name": schema.StringAttribute{
				MarkdownDescription: "Name of the group",
				Required:            true,
			},
			"internal_name": schema.StringAttribute{
				MarkdownDescription: "Internal name of the group",
				Computed:            true,
			},
			"group_email": schema.StringAttribute{
				MarkdownDescription: "Email of the group",
				Computed:            true,
			},
			"organization_id": schema.StringAttribute{
				MarkdownDescription: "Organization ID of the group",
				Computed:            true,
			},
			"organization_user_facing_id": schema.StringAttribute{
				MarkdownDescription: "Organization UFID of the group",
				Optional:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the group",
				Computed:            true,
			},
			"expiration_days": schema.Int64Attribute{
				MarkdownDescription: "Number of days until the group expires",
				Computed:            true,
			},
			"expiration_notification": schema.BoolAttribute{
				MarkdownDescription: "Whether to notify the user when the group expires",
				Computed:            true,
			},
			"require_grant_reason": schema.BoolAttribute{
				MarkdownDescription: "Whether to require a reason for granting access",
				Computed:            true,
			},
			"sync_group": schema.BoolAttribute{
				MarkdownDescription: "Whether to sync the google group",
				Computed:            true,
			},
			"last_updated_date": schema.StringAttribute{
				MarkdownDescription: "Workbench workspace last updated date",
				Computed:            true,
				CustomType:          timetypes.RFC3339Type{},
			},
			"last_updated_by": schema.StringAttribute{
				MarkdownDescription: "Workbench workspace last updated by",
				Computed:            true,
			},
			"created_date": schema.StringAttribute{
				MarkdownDescription: "Workbench workspace created date",
				Computed:            true,
				CustomType:          timetypes.RFC3339Type{},
			},
			"created_by": schema.StringAttribute{
				MarkdownDescription: "Workbench workspace created by",
				Computed:            true,
			},
		},
	}
}

// Configure initializes the client for the data source with the configuration.
func (d *GroupDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*ClientConfig)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *user.ClientWithResponse, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

// Read is called to read the data source.
func (d *GroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data models.GroupModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	c, err := api.NewUserClient(ctx, d.client.Host, d.client.UseIdToken, d.client.ImpersonateServiceAccount)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating user client",
			fmt.Sprintf("Unable to create user client: %s", err),
		)
		return
	}

	group, err := api.DescribeGroup(ctx, c, data.GroupName.ValueString(), data.OrgUfid.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting group",
			fmt.Sprintf("Unable to get group: %s", err),
		)
		return
	}

	groupState := models.NewGroupModel(group)
	resp.Diagnostics.Append(resp.State.Set(ctx, groupState)...)
}
