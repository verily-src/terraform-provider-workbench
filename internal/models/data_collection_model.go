package models

import (
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/verily-src/terraform-provider-workbench/internal/openapi/wsm"
)

const (
	SUPPORT_EMAIL_KEY     = "terra-support-email"
	THERAPEUTIC_TAGS_KEY  = "terra-therapeutic-tags"
	UPDATE_FREQUENCY_KEY  = "terra-update-frequency"
	ORGANIZATION_NAME_TAG = "terra-organization-name"
)

type DataCollectionModel struct {
	WorkspaceModel
	SupportEmail     types.String `tfsdk:"support_email"`
	OrganizationName types.String `tfsdk:"organization_name"`
	TherapeuticTags  types.Set    `tfsdk:"therapeutic_tags"`
	UpdateFrequency  types.String `tfsdk:"update_frequency"`
}

// NewDataCollectionModel creates a new DataCollectionModel with a given workspace description.
func NewDataCollectionModel(workspace *wsm.WorkspaceDescription) (*DataCollectionModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	raw_marshalled_string := GetValue(workspace.Properties, THERAPEUTIC_TAGS_KEY)
	unmarshalled_ttags, err := unmarshallJsonString(raw_marshalled_string)
	if err != nil {
		diags.AddError("Error Unmarshalling Therapeutic Tags", "Failed to unmarshal therapeutic tags from JSON string: "+raw_marshalled_string)
	}
	return &DataCollectionModel{
		WorkspaceModel:   *NewWorkspaceModel(workspace),
		SupportEmail:     types.StringValue(GetValue(workspace.Properties, SUPPORT_EMAIL_KEY)),
		OrganizationName: types.StringValue(GetValue(workspace.Properties, ORGANIZATION_NAME_TAG)),
		TherapeuticTags:  unmarshalled_ttags,
		UpdateFrequency:  types.StringValue(GetValue(workspace.Properties, UPDATE_FREQUENCY_KEY)),
	}, diags
}

// GetValue retrieves the value of a property by its key from a slice of wsm.Property.
// If the property is not found, it returns an empty string.
func GetValue(properties *[]wsm.Property, key string) string {
	if properties == nil {
		return ""
	}
	for _, property := range *properties {
		if property.Key == key {
			return property.Value
		}
	}
	return ""
}

func unmarshallJsonString(jsonStr string) (types.Set, error) {
	// Treat empty or "[]" as empty set
	if jsonStr == "" || jsonStr == "[]" {
		return types.SetNull(types.StringType), nil
	}
	var stringSlice []string
	if err := json.Unmarshal([]byte(jsonStr), &stringSlice); err != nil {
		return types.SetNull(types.StringType), err
	}
	tf_strings := make([]attr.Value, len(stringSlice))
	for i, s := range stringSlice {
		tf_strings[i] = types.StringValue(strings.ToLower(s))
	}
	ret, _ := types.SetValue(types.StringType, tf_strings)
	return ret, nil
}

// ConvertTypeSetToJsonString converts a types.Set of types.String to a JSON string representation.
func ConvertTypeSetToJsonString(ttags types.Set) (string, error) {
	if ttags.IsNull() || ttags.IsUnknown() {
		return "[]", nil // Return empty array instead of "null"
	}
	var stringSlice []string
	elems := ttags.Elements()
	if len(elems) == 0 {
		return "[]", nil
	}
	for _, s := range elems {
		strVal, _ := s.(types.String)
		stringSlice = append(stringSlice, strVal.ValueString())
	}
	jsonBytes, err := json.Marshal(stringSlice)
	return string(jsonBytes), err
}

// ConvertToCreateRequest converts the DataCollectionModel to a CreateWorkspaceV2JSONRequestBody.
// This is used to create a new data collection
func (datacollection *DataCollectionModel) ConvertToCreateRequest() (wsm.CreateWorkspaceV2JSONRequestBody, diag.Diagnostics) {
	var diags diag.Diagnostics
	datacollectionID, _ := uuid.Parse(datacollection.ID.ValueString())
	jobID := uuid.New()
	stage := wsm.CRGWORKSPACE
	stagePtr := &stage
	return wsm.CreateWorkspaceV2JSONRequestBody{
		Id:                   datacollectionID,
		UserFacingId:         datacollection.UserFacingId.ValueStringPointer(),
		DisplayName:          datacollection.DisplayName.ValueStringPointer(),
		Description:          datacollection.Description.ValueStringPointer(),
		OrganizationId:       datacollection.OrganizationID.ValueStringPointer(),
		CloudResourceGroupId: datacollection.PodID.ValueStringPointer(),
		Properties:           datacollection.GetProperties(diags),
		Policies:             datacollection.getPolicies(),
		Stage:                stagePtr,
		JobControl: wsm.JobControl{
			Id: jobID.String(),
		},
	}, diags
}

// GetProperties converts the properties of the DataCollectionModel to a slice of wsm.Property.
// Data Collection first party fields such as (support_email, therapeutic_tags, etc.) are added to the nested properties
// 'terra-type' is also injected here as "data-collection" to identify this as a Data Collection
func (datacollection *DataCollectionModel) GetProperties(diags diag.Diagnostics) *[]wsm.Property {
	workspace_props := datacollection.WorkspaceModel.GetProperties()
	marshalled, err := ConvertTypeSetToJsonString(datacollection.TherapeuticTags)
	if err != nil {
		marshalled = "[]"
		diags.AddError("Error Marshalling Therapeutic Tags", "Failed to marshal therapeutic tags to JSON string: "+err.Error())
	}
	*workspace_props = append(*workspace_props, wsm.Property{
		Key:   SUPPORT_EMAIL_KEY,
		Value: datacollection.SupportEmail.ValueString(),
	}, wsm.Property{
		Key:   UPDATE_FREQUENCY_KEY,
		Value: datacollection.UpdateFrequency.ValueString(),
	}, wsm.Property{
		Key:   ORGANIZATION_NAME_TAG,
		Value: datacollection.OrganizationName.ValueString(),
	}, wsm.Property{
		Key:   THERAPEUTIC_TAGS_KEY,
		Value: marshalled,
		// Append the terra-type which identifies this as DC
	}, wsm.Property{
		Key:   "terra-type",
		Value: "data-collection",
	})

	return workspace_props
}
