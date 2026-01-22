package provider

import (
	"context"
	"fmt"

	"github.com/cenkalti/backoff/v4"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/verily-src/terraform-provider-workbench/internal/api"
	"github.com/verily-src/terraform-provider-workbench/internal/models"
	"github.com/verily-src/terraform-provider-workbench/internal/openapi/wsm"
	"github.com/verily-src/terraform-provider-workbench/internal/schemas"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &WorkspaceResource{}
	_ resource.ResourceWithConfigure   = &WorkspaceResource{}
	_ resource.ResourceWithImportState = &WorkspaceResource{}
)

// NewWorkspaceResource initializes a new workspace resource.
func NewWorkspaceResource() resource.Resource {
	return &WorkspaceResource{}
}

// WorkspaceResource defines the resource implementation.
type WorkspaceResource struct {
	client      *ClientConfig
	retryClient *RetryClient
}

// Metadata returns the resource type name.
func (r *WorkspaceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workspace"
}

// Schema defines the resource-level schema for configuration.
func (r *WorkspaceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Workbench provisioned workspace resource",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Workbench workspace unique identifier",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "Workbench workspace display name",
				Optional:            true,
			},
			"user_facing_id": schema.StringAttribute{
				MarkdownDescription: "Workbench workspace user facing id",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Workbench workspace description",
				Optional:            true,
			},
			"location": schema.StringAttribute{
				MarkdownDescription: "Workbench workspace default resource location",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("us-central1"),
			},
			"policies": schema.SetNestedAttribute{
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
				MarkdownDescription: "Workbench workspace pod id",
				Required:            true,
			},
			"organization_id": schema.StringAttribute{
				MarkdownDescription: "Workbench workspace organization id",
				Required:            true,
			},
			"properties": schemas.PropertiesResourceSchema,
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_by": schema.StringAttribute{
				MarkdownDescription: "Workbench workspace created by",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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

// Configure initializes the client for the resource with the configuration.
func (r *WorkspaceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *WorkspaceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve provider data from the configuration.
	var data models.WorkspaceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	id := uuid.New().String()
	data.ID = types.StringValue(id)
	w, err := r.createWorkspaceAndWaitForComplete(ctx, data.ConvertToCreateRequest())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating workspace",
			fmt.Sprintf("Unable to create workspace: %s", err),
		)
		return
	}
	workspaceState := models.NewWorkspaceModel(w)
	data = *workspaceState

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkspaceResource) createWorkspaceAndWaitForComplete(ctx context.Context, request wsm.CreateWorkspaceV2JSONRequestBody) (*wsm.WorkspaceDescription, error) {
	c, err := api.NewWSMClient(ctx, r.client.Host, r.client.UseIdToken, r.client.ImpersonateServiceAccount)
	if err != nil {
		return nil, fmt.Errorf("unable to create Workbench client, unexpected error: %w", err)
	}
	tflog.Trace(ctx, "Creating a workspace")
	jobID, err := api.CreateWorkspace(ctx, c, request)
	if err != nil {
		return nil, err
	}
	if err := r.pullForWorkspaceCreationStatus(ctx, c, jobID); err != nil {
		return nil, err
	}
	tflog.Trace(ctx, "Created a workspace")
	return api.GetWorkspace(ctx, c, request.Id.String())
}

func (r *WorkspaceResource) pullForWorkspaceCreationStatus(ctx context.Context, client *wsm.ClientWithResponses, jobID string) error {
	return r.retryClient.Retry(func() error {
		rsp, err := client.GetCreateWorkspaceV2ResultWithResponse(ctx, jobID)
		if err != nil {
			return backoff.Permanent(fmt.Errorf("unable to check request status: %s", err))
		}
		if rsp.JSON202 != nil {
			return fmt.Errorf("Continue pulling for workspace creation status")
		}
		if rsp.JSON200 != nil {
			if rsp.JSON200.JobReport != nil && rsp.JSON200.JobReport.Status == "SUCCEEDED" {
				return nil
			}
			if rsp.JSON200.ErrorReport != nil {
				return backoff.Permanent(fmt.Errorf("pulling workspace creation status failed: %v", rsp.JSON200.ErrorReport.Message))
			}
		}
		return backoff.Permanent(fmt.Errorf("pulling workspace creation status failed, unexpected status: %d", rsp.StatusCode()))
	})
}

// Update updates the resource.
func (r *WorkspaceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state models.WorkspaceModel

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

	c, err := api.NewWSMClient(ctx, r.client.Host, r.client.UseIdToken, r.client.ImpersonateServiceAccount)

	// Update metadata
	r.updateMetadata(ctx, c, resp, &data, &state)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update policies
	deleted, added := models.DiffPolicies(state.Policies, data.Policies)
	r.updatePolicies(ctx, c, resp, data.ID.ValueString(), &added, &deleted)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update properties
	deletedProperties, addedProperties := models.DiffProperties(state.Properties, data.Properties)
	r.updateProperties(ctx, c, resp, data.ID.ValueString(), &addedProperties, &deletedProperties)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "Updated a workspace")
	w, err := api.GetWorkspace(ctx, c, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to retrieve updated workspace: %s", err))
		return
	}
	workspaceState := models.NewWorkspaceModel(w)
	data = *workspaceState

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkspaceResource) updateMetadata(ctx context.Context, c *wsm.ClientWithResponses, resp *resource.UpdateResponse, data *models.WorkspaceModel, state *models.WorkspaceModel) {
	if state.UserFacingId.ValueString() != data.UserFacingId.ValueString() ||
		state.DisplayName.ValueString() != data.DisplayName.ValueString() ||
		state.Description.ValueString() != data.Description.ValueString() {
		if _, err := api.UpdateWorkspace(ctx, c, data.ID.ValueString(), data.BuildUpdateRequest()); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to update workspace metadata: %s", err))
		}
	}
}

func (r *WorkspaceResource) updatePolicies(ctx context.Context, c *wsm.ClientWithResponses, resp *resource.UpdateResponse, UUID string, added *[]models.PolicyModel, deleted *[]models.PolicyModel) {

	if len(*deleted) > 0 {
		for _, d := range *deleted {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Delete policy %s not supported", d.String()))
		}
		return
	}

	if len(*added) > 0 {
		request := wsm.WsmPolicyUpdateRequest{
			AddAttributes: models.GetPoliciesInput(added),
			UpdateMode:    wsm.FAILONCONFLICT,
		}
		if err := api.UpdateWorkspacePolicies(ctx, c, UUID, request); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to update workspace policies: %s", err))
			return
		}
	}
}

func (r *WorkspaceResource) updateProperties(ctx context.Context, c *wsm.ClientWithResponses, resp *resource.UpdateResponse, UUID string, addedProperties *[]models.PropertyModel, deletedProperties *[]models.PropertyModel) {
	if len(*deletedProperties) > 0 {
		if err := api.DeleteWorkspaceProperties(ctx, c, UUID, models.GetKeys(*deletedProperties)); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to delete workspace properties: %s", err))
			return
		}
	}
	if len(*addedProperties) > 0 {
		if err := api.UpdateWorkspaceProperties(ctx, c, UUID, *models.BuildWSMProperties(addedProperties)); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to update workspace properties: %s", err))
			return
		}
	}
}

// Read reads the resource.
func (r *WorkspaceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data models.WorkspaceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Create a new client
	c, err := api.NewWSMClient(ctx, r.client.Host, r.client.UseIdToken, r.client.ImpersonateServiceAccount)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Workbench client, unexpected error: %s", err))
		return
	}
	// Get the workspace
	w, err := api.GetWorkspace(ctx, c, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Reading workspace into data, got error: %s", err))
		return
	}
	workspaceState := models.NewWorkspaceModel(w)
	data = *workspaceState
	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the workspace resource.
func (r *WorkspaceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data models.WorkspaceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Create a new client
	c, err := api.NewWSMClient(ctx, r.client.Host, r.client.UseIdToken, r.client.ImpersonateServiceAccount)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Workbench client, unexpected error: %s", err))
		return
	}
	// Delete the workspace
	jobID, err := api.DeleteWorkspace(ctx, c, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Deleting workspace into data, got error: %s", err))
		return
	}
	if jobID != nil {
		if err := r.pullForWorkspaceDeleteStatus(ctx, c, data.ID.ValueString(), *jobID); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Deleting workspace into data, got error: %s", err))
			return
		}
	} else {
		tflog.Trace(ctx, "Workspace already deleted")
	}

	tflog.Trace(ctx, "Deleted a workspace")
}

func (r *WorkspaceResource) pullForWorkspaceDeleteStatus(ctx context.Context, client *wsm.ClientWithResponses, workspaceID string, jobID string) error {
	return r.retryClient.Retry(func() error {
		rsp, err := client.GetDeleteWorkspaceV2ResultWithResponse(ctx, workspaceID, jobID)
		if err != nil {
			return backoff.Permanent(fmt.Errorf("unable to check request status: %s", err))
		}
		if rsp.JSON202 != nil {
			return fmt.Errorf("Continue pulling for workspace deletion status")
		}
		// Catch 403 errors to avoid throwing errors resulting from race condition on already deleted workspaces
		// Do not think this works currently, see below
		if rsp.JSON404 != nil || rsp.JSON403 != nil {
			return nil
		}
		if rsp.JSON200 != nil {
			if rsp.JSON200.JobReport.Status == "SUCCEEDED" {
				return nil
			}
			if rsp.JSON200.ErrorReport != nil {
				// catch race condition on already deleted workspaces - can return 200 with error report 403
				// ensure JSON200.ErrorReport is not nil before checking
				if rsp.JSON200.ErrorReport.StatusCode == 403 {
					return nil
				}
				return backoff.Permanent(fmt.Errorf("pulling workspace deletion status failed: %v", rsp.JSON200.ErrorReport.Message))
			}
		}
		return backoff.Permanent(fmt.Errorf("pulling workspace deletion status failed, unexpected status: %d", rsp.StatusCode()))
	})
}

func (r *WorkspaceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
