package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/verily-src/terraform-provider-workbench/internal/api"
	"github.com/verily-src/terraform-provider-workbench/internal/models"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &VersionResource{}
	_ resource.ResourceWithConfigure   = &VersionResource{}
	_ resource.ResourceWithImportState = &VersionResource{}
)

// NewVersionResource initializes a new version resource.
func NewVersionResource() resource.Resource {
	return &VersionResource{
		FolderResource: &FolderResource{},
	}
}

// VersionResource defines the resource implementation.
type VersionResource struct {
	*FolderResource
}

// Metadata returns the resource type name.
func (r *VersionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_data_collection_version"
}

// Schema defines the resource-level schema for configuration.
func (r *VersionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Workbench provisioned Data Collection Version resource",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Workbench Data Collection Version unique identifier",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"parent_folder_id": schema.StringAttribute{
				MarkdownDescription: "Version is toplevel folder, default to empty string",
				Computed:            true,
			},
			"workspace_id": schema.StringAttribute{
				MarkdownDescription: "Workspace ID to which the version belongs",
				Required:            true,
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "Workbench Data Collection Version display name",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Workbench Data Collection Version description",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("terraform-managed-version"),
			},
			"properties": schema.SetNestedAttribute{
				MarkdownDescription: "Workbench Data Collection Version properties in key-value pair",
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
			"release_notes_url": schema.StringAttribute{
				MarkdownDescription: "URL to the release notes for the version",
				Optional:            true,
			},
			"published": schema.BoolAttribute{
				MarkdownDescription: "Indicates whether the version is published, cannot be unset",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"published_date": schema.StringAttribute{
				MarkdownDescription: "Date when the version was published",
				Computed:            true,
				CustomType:          timetypes.RFC3339Type{},
			},
			"last_updated_date": schema.StringAttribute{
				MarkdownDescription: "Workbench Data Collection Version last updated date",
				Computed:            true,
				CustomType:          timetypes.RFC3339Type{},
			},
			"last_updated_by": schema.StringAttribute{
				MarkdownDescription: "Workbench Data Collection Version last updated by",
				Computed:            true,
			},
			"created_date": schema.StringAttribute{
				MarkdownDescription: "Workbench Data Collection Version created date",
				Computed:            true,
				CustomType:          timetypes.RFC3339Type{},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_by": schema.StringAttribute{
				MarkdownDescription: "Workbench Data Collection Version created by",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Configure initializes the client for the resource with the configuration.
func (r *VersionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*ClientConfig)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *ClientConfig, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.retryClient = NewRetryClient()
	r.client = client
}

// Create creates the resource.
func (r *VersionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve provider data from the configuration.
	var data models.VersionModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	// Check if the provided workspaceID is a valid data collection ID
	c, _ := api.NewWSMClient(ctx, r.client.Host)
	w, err := api.GetWorkspace(ctx, c, data.WorkspaceId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating version",
			fmt.Sprintf("Invalid Data Collection ID: %s", data.WorkspaceId.ValueString()),
		)
		return
	}
	if models.GetValue(w.Properties, "terra-type") != "data-collection" {
		resp.Diagnostics.AddError(
			"Error creating version",
			fmt.Sprintf("Invalid Data Collection ID: %s, expected a data collection, got workspace", data.WorkspaceId.ValueString()),
		)
		return
	}

	f, err := r.createFolderWithResponse(ctx, data.WorkspaceId.ValueString(), data.ToCreateRequest())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating version",
			fmt.Sprintf("Unable to create version: %s", err),
		)
		return
	}
	versionState, diags := models.NewVersionModel(f, data.WorkspaceId.ValueString())
	resp.Diagnostics.Append(diags...)

	resp.Diagnostics.Append(resp.State.Set(ctx, versionState)...)
}

// Update updates the version.
func (r *VersionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state models.VersionModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Read Terraform state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	c, err := api.NewWSMClient(ctx, r.client.Host)

	// Update metadata
	r.updateMetadata(ctx, c, resp, &data.FolderModel, &state.FolderModel)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update properties
	// Diff the properties to find out which ones to add and delete
	deletedProperties, addedProperties := models.DiffProperties(state.Properties, data.Properties)
	deletedVersionProperties, addedVersionProperties := diffVersionProperties(&state, &data, &resp.Diagnostics)
	deletedProperties = append(deletedProperties, deletedVersionProperties...)
	addedProperties = append(addedProperties, addedVersionProperties...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Send update request to the API
	r.updateProperties(ctx, c, resp, data.WorkspaceId.ValueString(), data.ID.ValueString(), &addedProperties, &deletedProperties)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "Updated a version")
	f, err := api.GetFolder(ctx, c, data.WorkspaceId.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to retrieve updated version: %s", err))
		return
	}
	versionState, diags := models.NewVersionModel(f, data.WorkspaceId.ValueString())
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	data = *versionState

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func diffVersionProperties(state *models.VersionModel, data *models.VersionModel, diags *diag.Diagnostics) (deletedProperties, addedProperties []models.PropertyModel) {
	if state.ReleaseNotesURL != data.ReleaseNotesURL {
		deletedProperties = append(deletedProperties, models.PropertyModel{
			Key:   types.StringValue(models.RELEASE_NOTES_URL_KEY),
			Value: state.ReleaseNotesURL,
		})
		addedProperties = append(addedProperties, models.PropertyModel{
			Key:   types.StringValue(models.RELEASE_NOTES_URL_KEY),
			Value: data.ReleaseNotesURL,
		})
	}
	if state.Published != data.Published {
		if state.Published.ValueBool() {
			diags.AddError("Invalid Update", "Cannot unset published property, it can only be set to true once.")
			return nil, nil
		}
		addedProperties = append(addedProperties, models.PropertyModel{
			Key:   types.StringValue(models.PUBLISHED_DATE_KEY),
			Value: types.StringValue(time.Now().Format(time.RFC3339)),
		})
	}
	return deletedProperties, addedProperties
}

// Read reads the resource.
func (r *VersionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data models.VersionModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Create a new client
	c, err := api.NewWSMClient(ctx, r.client.Host)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Workbench client, unexpected error: %s", err))
		return
	}
	// Get the workspace
	f, err := api.GetFolder(ctx, c, data.WorkspaceId.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Reading version, got error: %s", err))
		return
	}
	versionState, diags := models.NewVersionModel(f, data.WorkspaceId.ValueString())
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	data = *versionState
	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the Version resource.
func (r *VersionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data models.VersionModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Create a new client
	c, err := api.NewWSMClient(ctx, r.client.Host)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Workbench client, unexpected error: %s", err))
		return
	}
	// Delete the Data Collection Version
	jobID, err := api.DeleteFolderAsync(ctx, c, data.WorkspaceId.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Creating version delete job, got error: %s", err))
		return
	}
	if jobID != nil {
		if err := r.pullForFolderDeleteStatus(ctx, c, data.WorkspaceId.ValueString(), data.ID.ValueString(), *jobID); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Deleting version job failed, got error: %s", err))
			return
		}
	} else {
		tflog.Trace(ctx, "Version already deleted")
	}

	tflog.Trace(ctx, "Deleted a version")
}

func (r *VersionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	r.FolderResource.ImportState(ctx, req, resp)
}
