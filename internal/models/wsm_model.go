// package models defines the models used in the provider.
package models

import (
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/verily-src/terraform-provider-workbench/internal/openapi/wsm"
)

const (
	// WorkspacePropertyDefaultLocation is the property key for a workspace's default location.
	WorkspacePropertyDefaultLocation = "terra-default-location"
)

// WorkspaceModel is the description of the workspace resource.s
type WorkspaceModel struct {
	// The unique ID of the workspace.
	ID types.String `tfsdk:"id"`
	// UserFacingId is the unique user-facing ID of the workspace.
	UserFacingId types.String `tfsdk:"user_facing_id"`
	// DisplayName is the display name of the workspace.
	DisplayName types.String `tfsdk:"display_name"`
	// Description is the description of the workspace.
	Description types.String `tfsdk:"description"`
	// Location is the default location where resources are created.
	Location types.String `tfsdk:"location"`
	// Policies is the policies attached to the workspace.
	Policies *[]PolicyModel `tfsdk:"policies"`
	// PodId is the UUID of the pod where the workspace is created.
	PodID types.String `tfsdk:"pod_id"`
	// OrganizationId is the UUID of the organization that owns the workspace.
	OrganizationID types.String `tfsdk:"organization_id"`
	// Properties is a key-value pair for the workspace.
	Properties *[]PropertyModel `tfsdk:"properties"`
	// LastUpdatedDate is the date when the workspace was last updated.
	LastUpdatedDate timetypes.RFC3339 `tfsdk:"last_updated_date"`
	// LastUpdatedBy is the user who last updated the workspace.
	LastUpdatedBy types.String `tfsdk:"last_updated_by"`
	// CreatedDate is the date when the workspace was created.
	CreatedDate timetypes.RFC3339 `tfsdk:"created_date"`
	// CreatedBy is the user who created the workspace.
	CreatedBy types.String `tfsdk:"created_by"`
	// GcpProjectId is the GCP project ID associated with the workspace (if GCP workspace).
	GcpProjectId types.String `tfsdk:"gcp_project_id"`
	// AwsAccountId is the AWS account ID associated with the workspace (if AWS workspace).
	AwsAccountId types.String `tfsdk:"aws_account_id"`
}

// NewWorkspaceModel creates a new WorkspaceModel with a given description.
func NewWorkspaceModel(workspace *wsm.WorkspaceDescription) *WorkspaceModel {
	var gcpProjectId types.String
	var awsAccountId types.String

	// Extract GCP project ID if GCP context is present
	if workspace.GcpContext != nil {
		gcpProjectId = types.StringValue(workspace.GcpContext.ProjectId)
	} else {
		gcpProjectId = types.StringNull()
	}

	// Extract AWS account ID if AWS context is present
	if workspace.AwsContext != nil && workspace.AwsContext.AccountId != nil {
		awsAccountId = types.StringPointerValue(workspace.AwsContext.AccountId)
	} else {
		awsAccountId = types.StringNull()
	}

	return &WorkspaceModel{
		ID:              types.StringValue(workspace.Id.String()),
		DisplayName:     types.StringPointerValue(workspace.DisplayName),
		Description:     types.StringPointerValue(workspace.Description),
		UserFacingId:    types.StringValue(workspace.UserFacingId),
		LastUpdatedDate: timetypes.NewRFC3339TimeValue(workspace.LastUpdatedDate),
		LastUpdatedBy:   types.StringValue(workspace.LastUpdatedBy),
		CreatedDate:     timetypes.NewRFC3339TimeValue(workspace.CreatedDate),
		CreatedBy:       types.StringValue(workspace.CreatedBy),
		PodID:           uuidToStringType(workspace.CrgId),
		OrganizationID:  uuidToStringType(workspace.OrgId),
		Policies:        convertPolicies(workspace.Policies),
		Properties:      convertProperties(workspace.Properties),
		Location:        types.StringPointerValue(getDefaultLocation(workspace.Properties)),
		GcpProjectId:    gcpProjectId,
		AwsAccountId:    awsAccountId,
	}
}

// ConvertToCreateRequest converts the WorkspaceModel to a CreateWorkspaceV2JSONRequestBody.
// This is used to create a new workspace.
func (workspace *WorkspaceModel) ConvertToCreateRequest() wsm.CreateWorkspaceV2JSONRequestBody {
	workspaceID := uuid.New()
	jobID := uuid.New()
	stage := wsm.CRGWORKSPACE
	stagePtr := &stage
	return wsm.CreateWorkspaceV2JSONRequestBody{
		Id:                   workspaceID,
		UserFacingId:         workspace.UserFacingId.ValueStringPointer(),
		DisplayName:          workspace.DisplayName.ValueStringPointer(),
		Description:          workspace.Description.ValueStringPointer(),
		OrganizationId:       workspace.OrganizationID.ValueStringPointer(),
		CloudResourceGroupId: workspace.PodID.ValueStringPointer(),
		Properties:           workspace.GetProperties(),
		Policies:             workspace.getPolicies(),
		Stage:                stagePtr,
		JobControl: wsm.JobControl{
			Id: jobID.String(),
		},
	}
}

// BuildUpdateRequest converts the WorkspaceModel to a UpdateWorkspaceJSONRequestBody.
func (workspace *WorkspaceModel) BuildUpdateRequest() wsm.UpdateWorkspaceJSONRequestBody {
	return wsm.UpdateWorkspaceJSONRequestBody{
		DisplayName:  workspace.DisplayName.ValueStringPointer(),
		Description:  workspace.Description.ValueStringPointer(),
		UserFacingId: workspace.UserFacingId.ValueStringPointer(),
	}
}

// GetProperties converts the properties of the workspace to a slice of wsm.Property.
func (w *WorkspaceModel) GetProperties() *[]wsm.Property {
	properties := w.Properties
	var propertyModels []wsm.Property
	if properties != nil {
		for _, p := range *properties {
			propertyModels = append(propertyModels, wsm.Property{
				Key:   p.Key.ValueString(),
				Value: p.Value.ValueString(),
			})
		}
	}
	location := w.Location
	if location.IsNull() {
		location = types.StringValue("us-central1")
	}
	propertyModels = append(propertyModels, wsm.Property{
		Key:   WorkspacePropertyDefaultLocation,
		Value: location.ValueString(),
	})
	return &propertyModels
}

func (w *WorkspaceModel) getPolicies() *wsm.WsmPolicyInputs {
	return GetPoliciesInput(w.Policies)
}

func convertPolicies(policies *[]wsm.WsmPolicyInput) *[]PolicyModel {
	if policies == nil {
		return nil
	}
	var policyModels []PolicyModel
	for _, p := range *policies {
		policyModels = append(policyModels, PolicyModel{
			Namespace:      types.StringValue(p.Namespace),
			Name:           types.StringValue(p.Name),
			AdditionalData: convertAdditionalData(p.AdditionalData),
		})
	}
	return &policyModels
}

func convertAdditionalData(additionalData *[]wsm.WsmPolicyPair) *[]AdditionalDataModel {
	if additionalData == nil {
		return nil
	}
	var additionalDataModels []AdditionalDataModel
	for _, ad := range *additionalData {
		additionalDataModels = append(additionalDataModels, AdditionalDataModel{
			Key:   types.StringPointerValue(ad.Key),
			Value: types.StringPointerValue(ad.Value),
		})
	}
	return &additionalDataModels
}

func getDefaultLocation(properties *[]wsm.Property) *string {
	if properties == nil {
		return nil
	}
	for _, p := range *properties {
		if p.Key == WorkspacePropertyDefaultLocation {
			return &p.Value
		}
	}
	return nil
}
