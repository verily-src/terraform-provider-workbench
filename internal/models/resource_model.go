package models

import (
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/verily-src/terraform-provider-workbench/internal/openapi/wsm"
)

// ResourceModel is a generic model for resources in Workbench.
type ResourceModel struct {
	// ID is the unique ID of the resource.
	ID types.String `tfsdk:"id"`
	// WorkspaceID is the ID of the workspace that this resource belongs to.
	WorkspaceID types.String `tfsdk:"workspace_id"`
	// Name is the name of the resource.
	Name types.String `tfsdk:"name"`
	// DisplayName is the display name of the resource.
	DisplayName types.String `tfsdk:"display_name"`
	// Description is a description of the resource.
	Description types.String `tfsdk:"description"`
	// CreatedAt is the timestamp when the resource was created.
	CreatedAt timetypes.RFC3339 `tfsdk:"created_date"`
	// CreatedBy is the user who created the resource.
	CreatedBy types.String `tfsdk:"created_by"`
	// UpdatedAt is the timestamp when the resource was last updated.
	UpdatedAt timetypes.RFC3339 `tfsdk:"last_updated_date"`
	// UpdatedBy is the user who last updated the resource.
	UpdatedBy types.String `tfsdk:"last_updated_by"`
	// ResourceType is the type of the resource.
	ResourceType types.String `tfsdk:"resource_type"`
	// StewardshipType is the stewardship type of the resource.
	StewardshipType types.String `tfsdk:"stewardship_type"`
	// Properties is a map of additional properties for the resource.
	Properties *[]PropertyModel `tfsdk:"properties"`
	// FolderID is the ID of the folder that contains this resource.
	FolderID types.String `tfsdk:"folder_id"`
	// ResourceLineage is the lineage of the resource.
	ResourceLineage types.List `tfsdk:"resource_lineage"`
	// CloneInstruction is the instruction for cloning the resource.
	CloneInstruction types.String `tfsdk:"clone_instruction"`
}

// ResourceLineageEntry represents an entry in the resource lineage, representing where the resource is cloned from.
type ResourceLineageEntry struct {
	// SourceResourceID is the ID of the source resource.
	SourceResourceID types.String `tfsdk:"source_resource_id"`
	// SourceWorkspaceID is the ID of the workspace that contains the source resource.
	SourceWorkspaceID types.String `tfsdk:"source_workspace_id"`
}

func (m *ResourceModel) toControlledResourceCreateRequest() wsm.ControlledResourceCommonFields {

	return wsm.ControlledResourceCommonFields{
		AccessScope:         wsm.SHAREDACCESS,
		Name:                m.Name.ValueStringPointer(),
		DisplayName:         m.DisplayName.ValueStringPointer(),
		Description:         m.Description.ValueStringPointer(),
		Properties:          BuildWSMProperties(m.Properties),
		CloningInstructions: wsm.CloningInstructionsEnum(m.CloneInstruction.ValueString()),
		ManagedBy:           wsm.USER,
		FolderId:            parseUuid(m.FolderID),
	}
}

func NewResourceModel(r wsm.ResourceMetadata, workspaceID types.String) ResourceModel {
	return ResourceModel{
		ID:               types.StringValue(r.ResourceId.String()),
		WorkspaceID:      workspaceID,
		Name:             types.StringValue(r.Name),
		DisplayName:      types.StringPointerValue(r.DisplayName),
		Description:      types.StringPointerValue(r.Description),
		CreatedAt:        timetypes.NewRFC3339TimeValue(r.CreatedDate),
		CreatedBy:        types.StringValue(r.CreatedBy),
		UpdatedAt:        timetypes.NewRFC3339TimeValue(r.LastUpdatedDate),
		UpdatedBy:        types.StringValue(r.LastUpdatedBy),
		ResourceType:     types.StringValue(string(r.ResourceType)),
		StewardshipType:  types.StringValue(string(r.StewardshipType)),
		Properties:       convertProperties(r.Properties),
		FolderID:         uuidToStringType(r.FolderId),
		ResourceLineage:  BuildResourceLineageModels(r.ResourceLineage),
		CloneInstruction: safeConvertCloningInstructions(r.CloningInstructions),
	}
}

func safeConvertCloningInstructions(c *wsm.CloningInstructionsEnum) types.String {
	if c == nil {
		return types.StringNull()
	}
	return types.StringValue(string(*c))
}

func BuildResourceLineageModels(lineage *[]wsm.ResourceLineageEntry) types.List {
	objectType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"source_resource_id":  types.StringType,
			"source_workspace_id": types.StringType,
		},
	}

	// If lineage is nil or empty, return an empty list
	if lineage == nil || len(*lineage) == 0 {
		emptyList, _ := types.ListValue(objectType, []attr.Value{})
		return emptyList
	}

	values := make([]attr.Value, 0, len(*lineage))
	for _, entry := range *lineage {
		obj, _ := types.ObjectValue(
			objectType.AttrTypes,
			map[string]attr.Value{
				"source_resource_id":  types.StringValue(entry.SourceResourceId.String()),
				"source_workspace_id": types.StringValue(entry.SourceWorkspaceId.String()),
			},
		)
		values = append(values, obj)
	}

	listVal, _ := types.ListValue(objectType, values)
	return listVal
}
