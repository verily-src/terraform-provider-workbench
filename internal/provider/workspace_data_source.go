package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/verily-src/terraform-provider-workbench/internal/api"
	"github.com/verily-src/terraform-provider-workbench/internal/models"
	"github.com/verily-src/terraform-provider-workbench/internal/openapi/wsm"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ datasource.DataSource              = &WorkspaceDataSource{}
	_ datasource.DataSourceWithConfigure = &WorkspaceDataSource{}
)

// NewWorkspaceDataSource initializes a new workspace data source.
func NewWorkspaceDataSource() datasource.DataSource {
	return &WorkspaceDataSource{}
}

// WorkspaceDataSource defines the data source implementation.
type WorkspaceDataSource struct {
	client *ClientConfig
}

// Metadata returns the data source type name.
func (d *WorkspaceDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workspace"
}

// Schema defines the data source-level schema for configuration.
func (d *WorkspaceDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Workbench workspace UUID",
				Optional:            true,
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "Workbench workspace display name",
				Optional:            true,
			},
			"user_facing_id": schema.StringAttribute{
				MarkdownDescription: "Workbench workspace user facing id",
				Optional:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Workbench workspace description",
				Optional:            true,
			},
			"location": schema.StringAttribute{
				MarkdownDescription: "Workbench workspace default resource location",
				Optional:            true,
			},
			"policies": schema.ListNestedAttribute{
				MarkdownDescription: "Workbench workspace policies",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"namespace": schema.StringAttribute{
							MarkdownDescription: "Namespace of the policy",
							Required:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Name of the policy",
							Required:            true,
						},
						"additional_data": schema.ListNestedAttribute{
							MarkdownDescription: "Additional data for the policy",
							Optional:            true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"key": schema.StringAttribute{
										MarkdownDescription: "Key of the additional data",
										Required:            true,
									},
									"value": schema.StringAttribute{
										MarkdownDescription: "Value of the additional data",
										Required:            true,
									},
								},
							},
						},
					},
				},
			},
			"pod_id": schema.StringAttribute{
				MarkdownDescription: "Workbench workspace pod id",
				Optional:            true,
			},
			"organization_id": schema.StringAttribute{
				MarkdownDescription: "Workbench workspace organization id",
				Optional:            true,
			},
			"properties": schema.ListNestedAttribute{
				MarkdownDescription: "Workbench workspace properties in key-value pair",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"key": schema.StringAttribute{
							MarkdownDescription: "Key of the property",
							Required:            true,
						},
						"value": schema.StringAttribute{
							MarkdownDescription: "Value of the property",
							Required:            true,
						},
					},
				},
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
			"gcp_project_id": schema.StringAttribute{
				MarkdownDescription: "GCP project ID associated with the workspace (if GCP workspace)",
				Computed:            true,
			},
			"aws_account_id": schema.StringAttribute{
				MarkdownDescription: "AWS account ID associated with the workspace (if AWS workspace)",
				Computed:            true,
			},
		},
	}
}

// Configure initializes the client for the data source with the configuration.
func (d *WorkspaceDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *WorkspaceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data models.WorkspaceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Validate that at least one identifier is provided
	idValue := data.ID.ValueString()
	userFacingIdValue := data.UserFacingId.ValueString()

	if idValue == "" && userFacingIdValue == "" {
		resp.Diagnostics.AddError(
			"Missing Required Field",
			"Either 'id' or 'user_facing_id' must be provided to query the workspace",
		)
		return
	}

	c, err := api.NewWSMClient(ctx, d.client.Host, d.client.UseIdToken, d.client.ImpersonateServiceAccount)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Workbench client, unexpected error: %s", err))
		return
	}

	var w *wsm.WorkspaceDescription
	if idValue != "" {
		// Query by UUID
		w, err = api.GetWorkspace(ctx, c, idValue)
	} else {
		// Query by user-facing ID
		w, err = api.GetWorkspaceByUserFacingId(ctx, c, userFacingIdValue)
	}

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Reading workspace into data, got error: %s", err))
		return
	}
	workspaceState := models.NewWorkspaceModel(w)
	data = *workspaceState

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
