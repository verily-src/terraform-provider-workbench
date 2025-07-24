package models

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/verily-src/terraform-provider-workbench/internal/client"
	"github.com/verily-src/terraform-provider-workbench/internal/openapi/wsm"
)

// GCSBucketModel represents a Google Cloud Storage (GCS) bucket in Workbench.
type GCSBucketModel struct {
	ResourceModel
	// BucketName is the name of the GCS bucket.
	BucketName types.String `tfsdk:"bucket_name"`
}

// ControlledGCSBucketModel represents a controlled GCS bucket in Workbench.
type ControlledGCSBucketModel struct {
	GCSBucketModel
	// Location is the location of the GCS bucket.
	Location types.String `tfsdk:"location"`
	// StorageClass is the storage class of the GCS bucket.
	StorageClass types.String `tfsdk:"storage_class"`
}

// ToCreateRequest converts the ControlledGCSBucketModel to a CreateControlledGcpGcsBucketRequestBody for creating a controlled GCS bucket in Workbench.
func (m *ControlledGCSBucketModel) ToCreateRequest() wsm.CreateControlledGcpGcsBucketRequestBody {
	return wsm.CreateControlledGcpGcsBucketRequestBody{
		GcsBucket: m.toGcsBucketCreateRequestParam(),
		Common:    m.toControlledResourceCreateRequest(),
	}
}

func (m *ControlledGCSBucketModel) toGcsBucketCreateRequestParam() wsm.GcpGcsBucketCreationParameters {
	return wsm.GcpGcsBucketCreationParameters{
		Name:                m.BucketName.ValueStringPointer(),
		Location:            m.Location.ValueStringPointer(),
		DefaultStorageClass: safeStorageClass(m.StorageClass),
	}
}

// NewGCSBucketModel creates a new GCSBucketModel from a GcpGcsBucketResource and workspace ID.
func NewGCSBucketModel(b wsm.GcpGcsBucketResource, workspaceID types.String) GCSBucketModel {
	model := GCSBucketModel{
		ResourceModel: NewResourceModel(b.Metadata, workspaceID),
		BucketName:    types.StringValue(b.Attributes.BucketName),
	}
	return model
}

// NewControlledGcsBucketModel creates a new ControlledGCSBucketModel from a CreatedControlledGcpGcsBucket and workspace ID.
func NewControlledGcsBucketModel(b wsm.CreatedControlledGcpGcsBucket, workspaceID, storageClass types.String) ControlledGCSBucketModel {
	return ControlledGCSBucketModel{
		GCSBucketModel: NewGCSBucketModel(b.GcpBucket, workspaceID),
		Location:       types.StringPointerValue(b.GcpBucket.Metadata.ControlledResourceMetadata.Region),
		StorageClass:   storageClass,
	}
}

// ToUpdateControlledControlledGcpGcsBucketRequestBody converts the ControlledGCSBucketModel to an UpdateControlledGcpGcsBucketRequestBody for updating a controlled GCS bucket in Workbench.
func (m *ControlledGCSBucketModel) ToUpdateControlledGcpGcsBucketRequestBody(new ControlledGCSBucketModel) (*wsm.UpdateControlledGcpGcsBucketRequestBody, error) {
	if m.ID.ValueString() != new.ID.ValueString() {
		return nil, fmt.Errorf("cannot update a controlled GCS bucket with a different ID: %s != %s", m.ID.ValueString(), new.ID.ValueString())
	}
	if m.WorkspaceID.ValueString() != new.WorkspaceID.ValueString() {
		return nil, fmt.Errorf("cannot update a controlled GCS bucket to a different workspace: %s != %s", m.WorkspaceID.ValueString(), new.WorkspaceID.ValueString())
	}
	if m.BucketName.ValueString() != new.BucketName.ValueString() {
		return nil, fmt.Errorf("cannot update the bucket name of a controlled GCS bucket: %s != %s", m.BucketName.ValueString(), new.BucketName.ValueString())
	}
	if new.Location.ValueString() != "" && m.Location.ValueString() != new.Location.ValueString() {
		return nil, fmt.Errorf("cannot update the location of a controlled GCS bucket: %s != %s", m.Location.ValueString(), new.Location.ValueString())
	}
	var newDescription *string
	var newDisplayName *string
	var newName *string
	var updateFolderId *wsm.UpdateFolderId
	if !new.Description.IsNull() && m.Description.ValueString() != new.Description.ValueString() {
		newDescription = new.Description.ValueStringPointer()
	}
	if !new.DisplayName.IsNull() && m.DisplayName.ValueString() != new.DisplayName.ValueString() {
		newDisplayName = new.DisplayName.ValueStringPointer()
	}
	if !new.Name.IsNull() && m.Name.ValueString() != new.Name.ValueString() {
		newName = new.Name.ValueStringPointer()
	}
	if new.FolderID.IsNull() && !m.FolderID.IsNull() {
		// not set FolderId in UpdateFolderId to remove the folder id from the resource
		updateFolderId = &wsm.UpdateFolderId{}
	} else if m.FolderID.ValueString() != new.FolderID.ValueString() {
		updateFolderId = &wsm.UpdateFolderId{
			FolderId: parseUuid(new.FolderID),
		}
	}

	var bucketUpdateParam *wsm.GcpGcsBucketUpdateParameters
	if m.CloneInstruction.ValueString() != new.CloneInstruction.ValueString() ||
		m.StorageClass.ValueString() != new.StorageClass.ValueString() {
		bucketUpdateParam = &wsm.GcpGcsBucketUpdateParameters{}

		if m.CloneInstruction.ValueString() != new.CloneInstruction.ValueString() {
			bucketUpdateParam.CloningInstructions = safeCloningInstruction(new.CloneInstruction)
		}
		if m.StorageClass.ValueString() != new.StorageClass.ValueString() {
			bucketUpdateParam.DefaultStorageClass = safeStorageClass(new.StorageClass)
		}
	}

	return &wsm.UpdateControlledGcpGcsBucketRequestBody{
		Description:      newDescription,
		DisplayName:      newDisplayName,
		Name:             newName,
		UpdateFolderId:   updateFolderId,
		UpdateParameters: bucketUpdateParam,
	}, nil
}

func safeStorageClass(storageClass types.String) *wsm.GcpGcsBucketDefaultStorageClass {
	if storageClass.IsNull() || storageClass.ValueString() == "" {
		return nil
	}
	return client.Ptr(wsm.GcpGcsBucketDefaultStorageClass(storageClass.ValueString()))
}

func safeCloningInstruction(cloningInstruction types.String) *wsm.CloningInstructionsEnum {
	if cloningInstruction.IsNull() || cloningInstruction.ValueString() == "" {
		return nil
	}
	return client.Ptr(wsm.CloningInstructionsEnum(cloningInstruction.ValueString()))
}
