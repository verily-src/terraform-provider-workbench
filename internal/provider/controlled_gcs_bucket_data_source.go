package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/verily-src/terraform-provider-workbench/internal/api"
	"github.com/verily-src/terraform-provider-workbench/internal/models"
	"github.com/verily-src/terraform-provider-workbench/internal/schemas"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ datasource.DataSource              = &ControlledGcsBucketDataSource{}
	_ datasource.DataSourceWithConfigure = &ControlledGcsBucketDataSource{}
)

// NewControlledGcsBucketDataSource iniitializes a new ControlledGcsBucket data source.
func NewControlledGcsBucketDataSource() datasource.DataSource {
	return &ControlledGcsBucketDataSource{}
}

// ControlledGcsBucketDataSource defines the data source implementation.
type ControlledGcsBucketDataSource struct {
	client *ClientConfig
}

// Metadata returns the data source type name.
func (d *ControlledGcsBucketDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_controlled_gcs_bucket"
}

// Schema defines the data source-level schema for configuration.
func (d *ControlledGcsBucketDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique ID of the controlled GCS bucket",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the controlled GCS bucket",
				Computed:            true,
			},
			"workspace_id": schemas.WorkspaceDataSourceSchema,
			"display_name": schema.StringAttribute{
				MarkdownDescription: "Workbench controlled GCS bucket display name",
				Optional:            true,
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Workbench controlled GCS bucket description",
				Optional:            true,
				Computed:            true,
			},
			"bucket_name": schema.StringAttribute{
				MarkdownDescription: "Workbench controlled GCS bucket name",
				Computed:            true,
			},
			"resource_type": schema.StringAttribute{
				MarkdownDescription: "Workbench controlled GCS bucket resource type",
				Computed:            true,
			},
			"stewardship_type": schema.StringAttribute{
				MarkdownDescription: "Workbench controlled GCS bucket stewardship type",
				Computed:            true,
			},
			"clone_instruction": schema.StringAttribute{
				MarkdownDescription: "Workbench controlled GCS bucket cloning instruction",
				Computed:            true,
			},
			"location": schema.StringAttribute{
				MarkdownDescription: "Workbench controlled GCS bucket location",
				Optional:            true,
				Computed:            true,
			},
			"properties": schema.ListNestedAttribute{
				MarkdownDescription: "Workbench controlled GCS bucket properties in key-value pair",
				Optional:            true,
				Computed:            true,
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
				MarkdownDescription: "Workbench controlled GCS bucket last updated date",
				Computed:            true,
				CustomType:          timetypes.RFC3339Type{},
			},
			"last_updated_by": schema.StringAttribute{
				MarkdownDescription: "Workbench controlled GCS bucket last updated by",
				Computed:            true,
			},
			"created_date": schema.StringAttribute{
				MarkdownDescription: "Workbench controlled GCS bucket created date",
				Computed:            true,
				CustomType:          timetypes.RFC3339Type{},
			},
			"created_by": schema.StringAttribute{
				MarkdownDescription: "Workbench controlled GCS bucket created by",
				Computed:            true,
			},
			"resource_lineage": schema.ListNestedAttribute{
				MarkdownDescription: "Workbench controlled GCS bucket resource lineage",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"source_resource_id": schema.StringAttribute{
							MarkdownDescription: "The ID of the source resource from which this controlled GCS bucket was cloned",
							Computed:            true,
						},
						"source_workspace_id": schema.StringAttribute{
							MarkdownDescription: "The ID of the workspace that contains the source resource from which this controlled GCS bucket was cloned",
							Computed:            true,
						},
					},
				},
			},
			"folder_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the folder that contains this controlled GCS bucket",
				Optional:            true,
				Computed:            true,
			},
			"storage_class": schema.StringAttribute{
				MarkdownDescription: "The default storage class of the controlled GCS bucket",
				Optional:            true,
				Computed:            true,
			},
		},
	}
}

// Configure initializes the client for the data source with the configuration.
func (d *ControlledGcsBucketDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *ControlledGcsBucketDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data models.ControlledGCSBucketModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	c, err := api.NewWSMClient(ctx, d.client.Host)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Workbench client, unexpected error: %s", err))
		return
	}
	b, err := api.GetControlledGcsBucket(ctx, c, data.WorkspaceID.ValueString(), data.ID.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Reading ControlledGcsBucket into data, got error: %s", err))
		return
	}
	bucket := models.NewGCSBucketModel(*b, data.WorkspaceID)
	data.GCSBucketModel = bucket

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
