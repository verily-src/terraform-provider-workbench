package provider

import (
	"context"
	"fmt"

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
	"github.com/verily-src/terraform-provider-workbench/internal/schemas"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &DataCollectionResource{}
	_ resource.ResourceWithConfigure   = &DataCollectionResource{}
	_ resource.ResourceWithImportState = &DataCollectionResource{}
)

// NewDataCollection initializes a new data collection resource.
func NewDataCollectionResource() resource.Resource {
	return &DataCollectionResource{
		WorkspaceResource: &WorkspaceResource{},
	}
}

// DataCollectionResource defines the resource implementation.
type DataCollectionResource struct {
	*WorkspaceResource
}

// Metadata returns the resource type name.
func (r *DataCollectionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_data_collection"
}

// Schema defines the resource-level schema for configuration.
func (r *DataCollectionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Workbench provisioned datacollection resource",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Workbench datacollection unique identifier",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "Workbench datacollection display name",
				Optional:            true,
			},
			"user_facing_id": schema.StringAttribute{
				MarkdownDescription: "Workbench datacollection user facing id",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Workbench datacollection description",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("Managed by Terraform"),
			},
			"location": schema.StringAttribute{
				MarkdownDescription: "Workbench datacollection default resource location",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("us-central1"),
			},
			"policies": schema.SetNestedAttribute{
				MarkdownDescription: "Workbench datacollection policies",
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
				MarkdownDescription: "Workbench datacollection pod id",
				Required:            true,
			},
			"organization_id": schema.StringAttribute{
				MarkdownDescription: "Workbench datacollection organization id",
				Required:            true,
			},
			"properties": schemas.PropertiesResourceSchema,
			"support_email": schema.StringAttribute{
				MarkdownDescription: "Support email for the maintainer of this Data Collection",
				Required:            true,
			},
			"organization_name": schema.StringAttribute{
				MarkdownDescription: "Organization name maintaining this Data Collection",
				Required:            true,
			},
			"therapeutic_tags": schema.SetAttribute{
				MarkdownDescription: "Therapeutic tags for this Data Collection",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"update_frequency": schema.StringAttribute{
				MarkdownDescription: "Data Update frequency for this data collection",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("No Update Frequency Provided"),
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_by": schema.StringAttribute{
				MarkdownDescription: "Workbench data collection created by",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Configure initializes the client for the resource with the configuration.
func (r *DataCollectionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *DataCollectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve provider data from the configuration.
	var data models.DataCollectionModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	id := uuid.New().String()
	data.ID = types.StringValue(id)
	api_create_req, diags := data.ConvertToCreateRequest()
	resp.Diagnostics.Append(diags...)
	w, err := r.createWorkspaceAndWaitForComplete(ctx, api_create_req)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating data collection",
			fmt.Sprintf("Unable to create data collection: %s", err),
		)
		return
	}
	datacollectionState, diags := models.NewDataCollectionModel(w)
	resp.Diagnostics.Append(diags...)

	data = *datacollectionState

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource.
func (r *DataCollectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state models.DataCollectionModel

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
	r.updateMetadata(ctx, c, resp, &data.WorkspaceModel, &state.WorkspaceModel)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update policies
	deleted, added := models.DiffPolicies(state.Policies, data.Policies)
	r.updatePolicies(ctx, c, resp, data.ID.ValueString(), &deleted, &added)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update properties
	deletedProperties, addedProperties := models.DiffProperties(state.Properties, data.Properties)
	deletedDCProperties, addedDCProperties := r.diffDCProperties(ctx, &state, &data)
	addedProperties = append(addedProperties, addedDCProperties...)
	deletedProperties = append(deletedProperties, deletedDCProperties...)
	r.updateProperties(ctx, c, resp, data.ID.ValueString(), &addedProperties, &deletedProperties)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "Updated a data collection")
	w, err := api.GetWorkspace(ctx, c, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to retrieve updated data collection: %s", err))
		return
	}
	datacollectionState, diags := models.NewDataCollectionModel(w)
	resp.Diagnostics.Append(diags...)

	data = *datacollectionState

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DataCollectionResource) diffDCProperties(ctx context.Context, state *models.DataCollectionModel, data *models.DataCollectionModel) (deletedProperties, addedProperties []models.PropertyModel) {
	if state.SupportEmail != data.SupportEmail {
		deletedProperties = append(deletedProperties, models.PropertyModel{
			Key:   types.StringValue(models.SUPPORT_EMAIL_KEY),
			Value: state.SupportEmail,
		})
		addedProperties = append(addedProperties, models.PropertyModel{
			Key:   types.StringValue(models.SUPPORT_EMAIL_KEY),
			Value: data.SupportEmail,
		})
	}
	if state.OrganizationName != data.OrganizationName {
		deletedProperties = append(deletedProperties, models.PropertyModel{
			Key:   types.StringValue(models.ORGANIZATION_NAME_TAG),
			Value: state.OrganizationName,
		})
		addedProperties = append(addedProperties, models.PropertyModel{
			Key:   types.StringValue(models.ORGANIZATION_NAME_TAG),
			Value: data.OrganizationName,
		})
	}
	if state.UpdateFrequency != data.UpdateFrequency {
		deletedProperties = append(deletedProperties, models.PropertyModel{
			Key:   types.StringValue(models.UPDATE_FREQUENCY_KEY),
			Value: state.UpdateFrequency,
		})
		addedProperties = append(addedProperties, models.PropertyModel{
			Key:   types.StringValue(models.UPDATE_FREQUENCY_KEY),
			Value: data.UpdateFrequency,
		})
	}
	if !state.TherapeuticTags.Equal(data.TherapeuticTags) {
		deletedProperties = append(deletedProperties, models.PropertyModel{
			Key:   types.StringValue(models.THERAPEUTIC_TAGS_KEY),
			Value: types.StringNull(),
		})
		therapeuticTags, err := models.ConvertTypeSetToJsonString(data.TherapeuticTags)
		if err != nil {
			tflog.Error(ctx, fmt.Sprintf("Error marshalling therapeutic tags: %s", err))
			return deletedProperties, addedProperties
		}
		addedProperties = append(addedProperties, models.PropertyModel{
			Key:   types.StringValue(models.THERAPEUTIC_TAGS_KEY),
			Value: types.StringValue(therapeuticTags),
		})
	}
	return deletedProperties, addedProperties
}

// Read reads the resource.
func (r *DataCollectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data models.DataCollectionModel

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
	datacollectionState, diags := models.NewDataCollectionModel(w)
	resp.Diagnostics.Append(diags...)

	data = *datacollectionState
	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the data collection resource.
func (r *DataCollectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data models.DataCollectionModel

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
	// Delete the data collection
	jobID, err := api.DeleteWorkspace(ctx, c, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Deleting Data Collection into data, got error: %s", err))
		return
	}
	if jobID != nil {
		if err := r.pullForWorkspaceDeleteStatus(ctx, c, data.ID.ValueString(), *jobID); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Deleting Data Collection into data, got error: %s", err))
			return
		}
	} else {
		tflog.Trace(ctx, "Data Collection already deleted")
	}

	tflog.Trace(ctx, "Deleted a data collection")
}

func (r *DataCollectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
