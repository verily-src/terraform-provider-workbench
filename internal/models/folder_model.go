package models

import (
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/verily-src/terraform-provider-workbench/internal/client"
	"github.com/verily-src/terraform-provider-workbench/internal/openapi/wsm"
)

// FolderModel is the description of the folder resource.
type FolderModel struct {
	// The unique ID of the workspace.
	ID types.String `tfsdk:"id"`
	// ParentFolderId is the ID of the parent folder, if any.
	ParentFolderId types.String `tfsdk:"parent_folder_id"`
	// WorkspaceId is the unique ID of the workspace.
	WorkspaceId types.String `tfsdk:"workspace_id"`
	// DisplayName is the display name of the workspace.
	DisplayName types.String `tfsdk:"display_name"`
	// Description is the description of the workspace.
	Description types.String `tfsdk:"description"`
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
}

func NewFolderModel(f *wsm.Folder, workspaceId string) *FolderModel {
	if f == nil {
		return nil
	}
	var parentFolderId *string
	if f.ParentFolderId != nil {
		parentFolderId = client.Ptr(f.ParentFolderId.String())
	}
	return &FolderModel{
		ID:              types.StringValue(f.Id.String()),
		ParentFolderId:  types.StringPointerValue(parentFolderId),
		WorkspaceId:     types.StringValue(workspaceId),
		DisplayName:     types.StringValue(f.DisplayName),
		Description:     types.StringPointerValue(f.Description),
		Properties:      convertProperties(f.Properties),
		LastUpdatedDate: timetypes.NewRFC3339TimeValue(f.LastUpdatedDate),
		LastUpdatedBy:   types.StringValue(f.LastUpdatedBy),
		CreatedDate:     timetypes.NewRFC3339TimeValue(f.CreatedDate),
		CreatedBy:       types.StringValue(f.CreatedBy),
	}
}

// ToCreateRequest converts the FolderModel to a CreateFolderJSONRequestBody.
func (f *FolderModel) ToCreateRequest() *wsm.CreateFolderJSONRequestBody {
	var payload wsm.CreateFolderJSONRequestBody
	payload.DisplayName = f.DisplayName.ValueString()
	payload.Description = f.Description.ValueStringPointer()
	if f.ParentFolderId.IsUnknown() || f.ParentFolderId.IsNull() {
		payload.ParentFolderId = nil
	} else {
		fID, err := uuid.Parse(f.ParentFolderId.ValueString())
		if err != nil {
			payload.ParentFolderId = nil
		} else {
			payload.ParentFolderId = &fID
		}
	}
	payload.Properties = f.getProperties()
	return &payload
}

func (f *FolderModel) getProperties() *[]wsm.Property {
	properties := f.Properties
	var propertyModels []wsm.Property
	if properties == nil {
		return nil
	}
	for _, p := range *properties {
		propertyModels = append(propertyModels, wsm.Property{
			Key:   p.Key.ValueString(),
			Value: p.Value.ValueString(),
		})
	}
	return &propertyModels
}

func (workspace *FolderModel) BuildUpdateRequest() wsm.UpdateFolderJSONRequestBody {
	return wsm.UpdateFolderJSONRequestBody{
		DisplayName: workspace.DisplayName.ValueStringPointer(),
		Description: workspace.Description.ValueStringPointer(),
	}
}
