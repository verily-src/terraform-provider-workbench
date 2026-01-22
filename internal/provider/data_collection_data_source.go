package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/verily-src/terraform-provider-workbench/internal/api"
	"github.com/verily-src/terraform-provider-workbench/internal/models"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ datasource.DataSource              = &DataCollectionDataSource{}
	_ datasource.DataSourceWithConfigure = &DataCollectionDataSource{}
)

// NewDataCollectionDataSource initializes a new data collection data source.
func NewDataCollectionDataSource() datasource.DataSource {
	return &DataCollectionDataSource{}
}

// DataCollectionDataSource defines the data source implementation.
type DataCollectionDataSource struct {
	client *ClientConfig
}

// Metadata returns the data source type name.
func (d *DataCollectionDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_data_collection"
}

// Schema defines the data source-level schema for configuration.
func (d *DataCollectionDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Workbench datacollection unique identifier",
				Required:            true,
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "Workbench datacollection display name",
				Computed:            true,
			},
			"user_facing_id": schema.StringAttribute{
				MarkdownDescription: "Workbench datacollection user facing id",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Workbench datacollection description",
				Computed:            true,
			},
			"location": schema.StringAttribute{
				MarkdownDescription: "Workbench datacollection default resource location",
				Computed:            true,
			},
			"policies": schema.SetNestedAttribute{
				MarkdownDescription: "Workbench datacollection policies",
				Computed:            true,
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
						"additional_data": schema.SetNestedAttribute{
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
				MarkdownDescription: "Workbench datacollection pod id",
				Computed:            true,
			},
			"organization_id": schema.StringAttribute{
				MarkdownDescription: "Workbench datacollection organization id",
				Computed:            true,
			},
			"properties": schema.SetNestedAttribute{
				MarkdownDescription: "Workbench data collection properties in key-value pair",
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
			"support_email": schema.StringAttribute{
				MarkdownDescription: "Support email for the maintainer of this Data Collection",
				Computed:            true,
			},
			"organization_name": schema.StringAttribute{
				MarkdownDescription: "Organization name maintaining this Data Collection",
				Computed:            true,
			},
			"therapeutic_tags": schema.SetAttribute{
				MarkdownDescription: "Therapeutic tags for this Data Collection",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"update_frequency": schema.StringAttribute{
				MarkdownDescription: "Data Update frequency for this data collection",
				Computed:            true,
			},
			"last_updated_date": schema.StringAttribute{
				MarkdownDescription: "Workbench data collection last updated date",
				Computed:            true,
				CustomType:          timetypes.RFC3339Type{},
			},
			"last_updated_by": schema.StringAttribute{
				MarkdownDescription: "Workbench data collection last updated by",
				Computed:            true,
			},
			"created_date": schema.StringAttribute{
				MarkdownDescription: "Workbench data collection created date",
				Computed:            true,
				CustomType:          timetypes.RFC3339Type{},
			},
			"created_by": schema.StringAttribute{
				MarkdownDescription: "Workbench data collection created by",
				Computed:            true,
			},
			"gcp_project_id": schema.StringAttribute{
				MarkdownDescription: "GCP project ID associated with the data collection (if GCP workspace)",
				Computed:            true,
			},
			"aws_account_id": schema.StringAttribute{
				MarkdownDescription: "AWS account ID associated with the data collection (if AWS workspace)",
				Computed:            true,
			},
		},
	}
}

// Configure initializes the client for the data source with the configuration.
func (d *DataCollectionDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *DataCollectionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data models.DataCollectionModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	c, err := api.NewWSMClient(ctx, d.client.Host, d.client.UseIdToken, d.client.ImpersonateServiceAccount)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Workbench client, unexpected error: %s", err))
		return
	}
	w, err := api.GetWorkspace(ctx, c, data.ID.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Reading data collection into data, got error: %s", err))
		return
	}
	datacollectionState, diags := models.NewDataCollectionModel(w)
	resp.Diagnostics.Append(diags...)

	data = *datacollectionState

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
