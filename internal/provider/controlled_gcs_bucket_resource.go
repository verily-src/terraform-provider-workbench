package provider

import (
	"context"
	"fmt"
	"regexp"

	"github.com/cenkalti/backoff/v4"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/verily-src/terraform-provider-workbench/internal/api"
	"github.com/verily-src/terraform-provider-workbench/internal/models"
	"github.com/verily-src/terraform-provider-workbench/internal/openapi/wsm"
	"github.com/verily-src/terraform-provider-workbench/internal/schemas"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &ControlledGcsBucketResource{}
	_ resource.ResourceWithConfigure   = &ControlledGcsBucketResource{}
	_ resource.ResourceWithImportState = &ControlledGcsBucketResource{}
)

// NewControlledGcsBucketResource initializes a new workspace resource.
func NewControlledGcsBucketResource() resource.Resource {
	return &ControlledGcsBucketResource{}
}

// ControlledGcsBucketResource defines the resource implementation.
type ControlledGcsBucketResource struct {
	client      *ClientConfig
	retryClient *RetryClient
}

// Metadata returns the resource type name.
func (r *ControlledGcsBucketResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_controlled_gcs_bucket"
}

// Schema defines the resource-level schema for configuration.
func (r *ControlledGcsBucketResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Workbench provisioned controlled GCS bucket resource",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Workbench controlled GCS bucket unique identifier",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"workspace_id": schemas.WorkspaceResourceSchema,
			"name": schema.StringAttribute{
				MarkdownDescription: "Workbench controlled GCS bucket user facing display name",
				Optional:            true,
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "Workbench controlled GCS bucket display name",
				Optional:            true,
			},
			"bucket_name": schema.StringAttribute{
				MarkdownDescription: "Workbench controlled GCS bucket cloud storage bucket name",
				Optional:            true,
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Workbench controlled GCS bucket description",
				Optional:            true,
			},
			"location": schema.StringAttribute{
				MarkdownDescription: "Workbench controlled GCS bucket default resource location, e.g. us-central1, europe-west4",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(), // Prevents updates
				},
			},
			"storage_class": schema.StringAttribute{
				MarkdownDescription: "Workbench controlled GCS bucket default storage class, e.g. STANDARD, NEARLINE, COLDLINE, ARCHIVE",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					storageClassValidator(),
				},
				Default: stringdefault.StaticString("STANDARD"),
			},
			"properties": schemas.PropertiesResourceSchema,
			"resource_type": schema.StringAttribute{
				MarkdownDescription: "Workbench controlled GCS bucket resource type",
				Computed:            true,
			},
			"stewardship_type": schema.StringAttribute{
				MarkdownDescription: "Workbench controlled GCS bucket stewardship type",
				Computed:            true,
			},
			"clone_instruction": schema.StringAttribute{
				MarkdownDescription: "Workbench controlled GCS bucket cloning instruction, e.g. COPY_RESOURCE, COPY_DEFINITION, COPY_LINK_REFERENCE, COPY_NOTHING, COPY_REFERENCE",
				Computed:            true,
				Optional:            true,
				Validators: []validator.String{
					cloningInstructionValidator(),
				},
				Default: stringdefault.StaticString("COPY_RESOURCE"),
			},
			"folder_id": schema.StringAttribute{
				MarkdownDescription: "Workbench controlled GCS bucket folder ID",
				Optional:            true,
			},
			"resource_lineage": schema.ListNestedAttribute{
				MarkdownDescription: "Workbench controlled GCS bucket resource lineage",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"source_resource_id": schema.StringAttribute{
							MarkdownDescription: "The ID of the source resource from which this controlled GCS bucket was cloned",
							Required:            true,
						},
						"source_workspace_id": schema.StringAttribute{
							MarkdownDescription: "The ID of the workspace that contains the source resource from which this controlled GCS bucket was cloned",
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_by": schema.StringAttribute{
				MarkdownDescription: "Workbench controlled GCS bucket created by",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Configure initializes the client for the resource with the configuration.
func (r *ControlledGcsBucketResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *ControlledGcsBucketResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve provider data from the configuration.
	var data models.ControlledGCSBucketModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	c, err := api.NewWSMClient(ctx, r.client.Host, r.client.UseIdToken)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Workbench client, unexpected error: %s", err))
		return
	}

	tflog.Trace(ctx, "Creating a GCS bucket")
	rsp, err := api.CreateControlledGcsBucket(ctx, c, data.WorkspaceID.ValueString(), data.ToCreateRequest())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating gcs bucket",
			fmt.Sprintf("Unable to create gcs bucket: %s", err),
		)
		return
	}
	newState := models.NewControlledGcsBucketModel(*rsp, data.WorkspaceID, data.StorageClass)

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

// Update updates the resource.
func (r *ControlledGcsBucketResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state models.ControlledGCSBucketModel

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

	updateRequest, err := state.ToUpdateControlledGcpGcsBucketRequestBody(data)
	if err != nil {
		resp.Diagnostics.AddError("Invalid update gcs bucket request", err.Error())
		return
	}

	c, err := api.NewWSMClient(ctx, r.client.Host, r.client.UseIdToken)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Workbench client, unexpected error: %s", err))
		return
	}

	rsp, err := api.UpdateControlledGcsBucket(ctx, c, data.WorkspaceID.ValueString(), data.ID.ValueString(), *updateRequest)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to update gcs bucket: %s", err))
		return
	}

	tflog.Trace(ctx, "Updated a GCS bucket")

	gcsBucketState := models.NewGCSBucketModel(*rsp, data.WorkspaceID)

	data.GCSBucketModel = gcsBucketState

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read reads the resource.
func (r *ControlledGcsBucketResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data models.ControlledGCSBucketModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Create a new client
	c, err := api.NewWSMClient(ctx, r.client.Host, r.client.UseIdToken)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Workbench client, unexpected error: %s", err))
		return
	}
	// Get the bucket
	b, err := api.GetControlledGcsBucket(ctx, c, data.WorkspaceID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Reading bucket into data, got error: %s", err))
		return
	}
	bucket := models.NewGCSBucketModel(*b, data.WorkspaceID)
	data.GCSBucketModel = bucket
	data.Location = types.StringPointerValue(b.Metadata.ControlledResourceMetadata.Region)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the gcs bucket resource.
func (r *ControlledGcsBucketResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data models.ControlledGCSBucketModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Create a new client
	c, err := api.NewWSMClient(ctx, r.client.Host, r.client.UseIdToken)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Workbench client, unexpected error: %s", err))
		return
	}
	// Delete the bucket
	jobID, err := api.DeleteControlledGcsBucketAsync(ctx, c, data.WorkspaceID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Deleting bucket into data, got error: %s", err))
		return
	}
	if jobID != nil {
		if err := r.pullForBucketDeleteStatus(ctx, c, data.WorkspaceID.ValueString(), *jobID); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Deleting bucket into data, got error: %s", err))
			return
		}
	} else {
		tflog.Trace(ctx, "Bucket already deleted")
	}

	tflog.Trace(ctx, "Deleted a bucket")
}

func (r *ControlledGcsBucketResource) pullForBucketDeleteStatus(ctx context.Context, client *wsm.ClientWithResponses, workspaceID, jobID string) error {
	return r.retryClient.Retry(func() error {
		rsp, err := client.GetDeleteBucketResultWithResponse(ctx, workspaceID, jobID)
		if err != nil {
			return backoff.Permanent(fmt.Errorf("unable to check request status: %s", err))
		}
		if rsp.JSON202 != nil {
			return fmt.Errorf("Continue pulling for bucket deletion status")
		}
		// Catch 403 errors to avoid throwing errors resulting from race condition on already deleted buckets
		if rsp.JSON403 != nil {
			tflog.Trace(ctx, "error report status is 403, bucket likely already deleted")
			return nil
		}
		if rsp.JSON200 != nil {
			if rsp.JSON200.JobReport.Status == "SUCCEEDED" {
				return nil
			}
			if rsp.JSON200.ErrorReport != nil {
				if rsp.JSON200.ErrorReport.StatusCode == 403 {
					tflog.Trace(ctx, "error report status is 403, bucket likely already deleted")
					return nil
				}
				return backoff.Permanent(fmt.Errorf("pulling bucket deletion status failed: %v", rsp.JSON200.ErrorReport.Message))
			}
		}
		return backoff.Permanent(fmt.Errorf("pulling bucket deletion status failed, unexpected status: %d", rsp.StatusCode()))
	})
}

var gcsBucketID = regexp.MustCompile("workspaces/(.+)/controlled_gcs_buckets/(.+)")

type GcsBucketID struct {
	WorkspaceID string
	ID          string
}

func ParseGcsBucketID(id string) *GcsBucketID {
	parts := gcsBucketID.FindStringSubmatch(id)
	if parts == nil {
		return nil
	}

	return &GcsBucketID{
		WorkspaceID: parts[1],
		ID:          parts[2],
	}
}

func (r *ControlledGcsBucketResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	gcsBucketID := ParseGcsBucketID(req.ID)
	if gcsBucketID == nil {
		resp.Diagnostics.AddError(
			"Invalid GCS Bucket ID",
			fmt.Sprintf("Unable to parse GCS bucket ID %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("workspace_id"), gcsBucketID.WorkspaceID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), gcsBucketID.ID)...)
}
