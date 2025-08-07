package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/cenkalti/backoff/v4"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/verily-src/terraform-provider-workbench/internal/api"
	"github.com/verily-src/terraform-provider-workbench/internal/models"
	"github.com/verily-src/terraform-provider-workbench/internal/openapi/wsm"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &FolderResource{}
	_ resource.ResourceWithConfigure   = &FolderResource{}
	_ resource.ResourceWithImportState = &FolderResource{}
)

// NewFolderResource initializes a new folder resource.
func NewFolderResource() resource.Resource {
	return &FolderResource{}
}

// FolderResource defines the resource implementation.
type FolderResource struct {
	client      *ClientConfig
	retryClient *RetryClient
}

// Metadata returns the resource type name.
func (r *FolderResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_folder"
}

// Schema defines the resource-level schema for configuration.
func (r *FolderResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Workbench provisioned folder resource",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Workbench folder unique identifier",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"parent_folder_id": schema.StringAttribute{
				MarkdownDescription: "Parent folder ID of the Folder, if any",
				Optional:            true,
			},
			"workspace_id": schema.StringAttribute{
				MarkdownDescription: "Folder ID to which the folder belongs",
				Required:            true,
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "Workbench Folder display name",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Workbench Folder description",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("terraform-managed-folder"),
			},
			"properties": schema.SetNestedAttribute{
				MarkdownDescription: "Workbench Folder properties in key-value pair",
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_by": schema.StringAttribute{
				MarkdownDescription: "Workbench Folder created by",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Configure initializes the client for the resource with the configuration.
func (r *FolderResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *FolderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve provider data from the configuration.
	var data models.FolderModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	f, err := r.createFolderWithResponse(ctx, data.WorkspaceId.ValueString(), data.ToCreateRequest())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating folder",
			fmt.Sprintf("Unable to create folder: %s", err),
		)
		return
	}
	folderState := models.NewFolderModel(f, data.WorkspaceId.ValueString())

	resp.Diagnostics.Append(resp.State.Set(ctx, folderState)...)
}

func (r *FolderResource) createFolderWithResponse(ctx context.Context, workspaceID string, request *wsm.CreateFolderJSONRequestBody) (*wsm.Folder, error) {
	c, err := api.NewWSMClient(ctx, r.client.Host)
	if err != nil {
		return nil, fmt.Errorf("unable to create Workbench client, unexpected error: %w", err)
	}
	tflog.Trace(ctx, "Creating a folder")
	// Note that the UUID param to this function is the workspace ID, not the folder ID\
	folder, err := api.CreateFolder(ctx, c, workspaceID, *request)
	if err != nil {
		b, _ := json.MarshalIndent(request, "", "  ") // Pretty-print JSON
		return nil, fmt.Errorf("CreateFolder on workspaceID (%s) failed: %w\nRequest: %s", workspaceID, err, b)
	}
	tflog.Trace(ctx, "Created a folder")
	return folder, nil
}

// Update updates the resource.
func (r *FolderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state models.FolderModel

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
	r.updateMetadata(ctx, c, resp, &data, &state)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update properties
	deletedProperties, addedProperties := models.DiffProperties(state.Properties, data.Properties)
	r.updateProperties(ctx, c, resp, data.WorkspaceId.ValueString(), data.ID.ValueString(), &addedProperties, &deletedProperties)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "Updated a folder")
	f, err := api.GetFolder(ctx, c, data.WorkspaceId.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to retrieve updated folder: %s", err))
		return
	}
	folderState := models.NewFolderModel(f, data.WorkspaceId.ValueString())
	data = *folderState

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FolderResource) updateMetadata(ctx context.Context, c *wsm.ClientWithResponses, resp *resource.UpdateResponse, data *models.FolderModel, state *models.FolderModel) {
	if state.DisplayName.ValueString() != data.DisplayName.ValueString() ||
		state.Description.ValueString() != data.Description.ValueString() {
		if _, err := api.UpdateFolder(ctx, c, data.WorkspaceId.ValueString(), data.ID.ValueString(), data.BuildUpdateRequest()); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to update folder metadata: %s", err))
		}
	}
}

func (r *FolderResource) updateProperties(ctx context.Context, c *wsm.ClientWithResponses, resp *resource.UpdateResponse, workspaceID string, folderID string, addedProperties *[]models.PropertyModel, deletedProperties *[]models.PropertyModel) {
	if deletedProperties != nil && len(*deletedProperties) > 0 {
		if err := api.DeleteFolderProperties(ctx, c, workspaceID, folderID, models.GetKeys(*deletedProperties)); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to delete Folder properties: %s", err))
			return
		}
	}
	if len(*addedProperties) > 0 {
		if err := api.UpdateFolderProperties(ctx, c, workspaceID, folderID, *models.BuildWSMProperties(addedProperties)); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to update Folder properties: %s", err))
			return
		}
	}
}

// Read reads the resource.
func (r *FolderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data models.FolderModel

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
	// Get the Folder
	f, err := api.GetFolder(ctx, c, data.WorkspaceId.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Reading folder into data, got error: %s", err))
		return
	}
	folderState := models.NewFolderModel(f, data.WorkspaceId.ValueString())
	data = *folderState
	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the Folder resource.
func (r *FolderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data models.FolderModel

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
	// Delete the Folder
	jobID, err := api.DeleteFolderAsync(ctx, c, data.WorkspaceId.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Creating folder delete job, got error: %s", err))
		return
	}
	if jobID != nil {
		if err := r.pullForFolderDeleteStatus(ctx, c, data.WorkspaceId.ValueString(), data.ID.ValueString(), *jobID); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Deleting folder job failed, got error: %s", err))
			return
		}
	} else {
		tflog.Trace(ctx, "Folder already deleted")
	}

	tflog.Trace(ctx, "Deleted a Folder")
}

func (r *FolderResource) pullForFolderDeleteStatus(ctx context.Context, client *wsm.ClientWithResponses, workspaceID string, folderID string, jobID string) error {
	return r.retryClient.Retry(func() error {
		folder_uuid, err := uuid.Parse(folderID)
		if err != nil {
			return backoff.Permanent(fmt.Errorf("Retrieving folder delete job: invalid folder ID %s: %v", folderID, err))
		}
		rsp, err := client.GetDeleteFolderResultWithResponse(ctx, workspaceID, folder_uuid, jobID)
		if err != nil {
			return backoff.Permanent(fmt.Errorf("unable to check request status: %s", err))
		}
		if rsp.JSON202 != nil {
			return fmt.Errorf("Continue pulling for Folder deletion status")
		}
		// Catch 403 errors to avoid throwing errors resulting from race condition on already deleted Folders
		if rsp.JSON404 != nil || rsp.JSON403 != nil {
			return nil
		}
		if rsp.JSON200 != nil {
			if rsp.JSON200.JobReport.Status == "SUCCEEDED" {
				return nil
			}
			if rsp.JSON200.ErrorReport != nil {
				if rsp.JSON200.ErrorReport.StatusCode == 403 {
					return backoff.Permanent(fmt.Errorf("user does not have permission to delete the folder: %s", rsp.JSON200.ErrorReport.Message))
				}
				return backoff.Permanent(fmt.Errorf("pulling Folder deletion status failed: %v", rsp.JSON200.ErrorReport.Message))
			}
		}
		return backoff.Permanent(fmt.Errorf("pulling Folder deletion status failed, unexpected status: %d", rsp.StatusCode()))
	})
}

type ResourceID struct {
	WorkspaceID string
	ResourceID  string
}

func ParseResourceID(id string) *ResourceID {
	var resourceID = regexp.MustCompile("workspaces/(.+)/resources/(.+)")
	parts := resourceID.FindStringSubmatch(id)
	if parts == nil {
		return nil
	}

	return &ResourceID{
		WorkspaceID: parts[1],
		ResourceID:  parts[2],
	}
}

func (r *FolderResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resourceID := ParseResourceID(req.ID)
	if resourceID == nil {
		resp.Diagnostics.AddError(
			"Invalid Folder ID",
			fmt.Sprintf("Unable to parse Folder/Version ID in the format 'workspaces/<workspace_id>/resources/<folder_id>', got: %s", req.ID),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("workspace_id"), resourceID.WorkspaceID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), resourceID.ResourceID)...)
}
