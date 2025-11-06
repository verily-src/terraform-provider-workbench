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
	_ datasource.DataSource              = &FolderDataSource{}
	_ datasource.DataSourceWithConfigure = &FolderDataSource{}
)

// NewFolderDataSource initializes a new folder data source.
func NewFolderDataSource() datasource.DataSource {
	return &FolderDataSource{}
}

// FolderDataSource defines the data source implementation.
type FolderDataSource struct {
	client *ClientConfig
}

// Metadata returns the data source type name.
func (d *FolderDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_folder"
}

// Schema defines the data source-level schema for configuration.
func (d *FolderDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Workbench folder unique identifier",
				Required:            true,
			},
			"parent_folder_id": schema.StringAttribute{
				MarkdownDescription: "Parent folder ID of the Folder, if any",
				Computed:            true,
			},
			"workspace": schema.StringAttribute{
				MarkdownDescription: "Workspace ID to which the folder belongs",
				Computed:            true,
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "Workbench Folder display name",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Workbench Folder description",
				Computed:            true,
			},
			"properties": schema.SetNestedAttribute{
				MarkdownDescription: "Workbench Folder properties in key-value pair",
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
				MarkdownDescription: "Workbench Folder last updated date",
				Computed:            true,
				CustomType:          timetypes.RFC3339Type{},
			},
			"last_updated_by": schema.StringAttribute{
				MarkdownDescription: "Workbench Folder last updated by",
				Computed:            true,
			},
			"created_date": schema.StringAttribute{
				MarkdownDescription: "Workbench Folder created date",
				Computed:            true,
				CustomType:          timetypes.RFC3339Type{},
			},
			"created_by": schema.StringAttribute{
				MarkdownDescription: "Workbench Folder created by",
				Computed:            true,
			},
		},
	}
}

// Configure initializes the client for the data source with the configuration.
func (d *FolderDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*ClientConfig)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *ClientConfig, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

// Read is called to read the data source.
func (d *FolderDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data models.FolderModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...) // Get the id from config

	if resp.Diagnostics.HasError() {
		return
	}

	c, err := api.NewWSMClient(ctx, d.client.Host, d.client.UseIdToken, d.client.ImpersonateServiceAccount)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Workbench client, unexpected error: %s", err))
		return
	}
	f, err := api.GetFolder(ctx, c, data.WorkspaceId.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Reading folder into data, got error: %s", err))
		return
	}
	folderState := models.NewFolderModel(f, data.WorkspaceId.ValueString())
	data = *folderState

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...) // Set all attributes
}
