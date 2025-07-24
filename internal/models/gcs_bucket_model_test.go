package models

import (
	"fmt"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/verily-src/terraform-provider-workbench/internal/client"
	"github.com/verily-src/terraform-provider-workbench/internal/openapi/wsm"
	"testing"
	"time"
)

func TestBuildGcsBucketCreateRequest(t *testing.T) {
	folderId := uuid.New()
	tests := []struct {
		name  string
		model ControlledGCSBucketModel
		want  wsm.CreateControlledGcpGcsBucketRequestBody
	}{
		{
			name: "minimal",
			model: ControlledGCSBucketModel{
				GCSBucketModel: GCSBucketModel{
					ResourceModel: ResourceModel{
						Name:             types.StringValue("test-bucket"),
						CloneInstruction: types.StringValue("COPY_NOTHING"),
					},
				},
			},
			want: wsm.CreateControlledGcpGcsBucketRequestBody{
				GcsBucket: wsm.GcpGcsBucketCreationParameters{},
				Common: wsm.ControlledResourceCommonFields{
					AccessScope:         wsm.SHAREDACCESS,
					Name:                client.Ptr(wsm.Name("test-bucket")),
					CloningInstructions: wsm.COPYNOTHING,
					ManagedBy:           wsm.USER,
				},
			},
		},
		{
			name: "full",
			model: ControlledGCSBucketModel{
				GCSBucketModel: GCSBucketModel{
					ResourceModel: ResourceModel{
						Name:             types.StringValue("test-bucket"),
						DisplayName:      types.StringValue("Test Bucket"),
						Description:      types.StringValue("This is a test GCS bucket"),
						CloneInstruction: types.StringValue("COPY_NOTHING"),
						FolderID:         types.StringValue(folderId.String()),
						Properties: &[]PropertyModel{
							{Key: types.StringValue("key1"), Value: types.StringValue("value1")},
						},
					},
					BucketName: types.StringValue("project-123-test-bucket"),
				},
				Location:     types.StringValue("us-central1"),
				StorageClass: types.StringValue("STANDARD"),
			},
			want: wsm.CreateControlledGcpGcsBucketRequestBody{
				GcsBucket: wsm.GcpGcsBucketCreationParameters{
					Name:                client.Ptr("project-123-test-bucket"),
					Location:            client.Ptr("us-central1"),
					DefaultStorageClass: client.Ptr(wsm.GcpGcsBucketDefaultStorageClass("STANDARD")),
				},
				Common: wsm.ControlledResourceCommonFields{
					AccessScope:         wsm.SHAREDACCESS,
					Name:                client.Ptr(wsm.Name("test-bucket")),
					DisplayName:         client.Ptr(wsm.Name("Test Bucket")),
					Description:         client.Ptr("This is a test GCS bucket"),
					CloningInstructions: wsm.COPYNOTHING,
					ManagedBy:           wsm.USER,
					FolderId:            client.Ptr(folderId),
					Properties:          client.Ptr([]wsm.Property{{Key: "key1", Value: "value1"}}),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.model.ToCreateRequest()
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("ToCreateRequest() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewGCSBucketModel(t *testing.T) {
	resourceId := uuid.New()
	now := time.Now()
	tests := []struct {
		name string
		b    wsm.GcpGcsBucketResource
		want GCSBucketModel
	}{
		{
			name: "minimal",
			b: wsm.GcpGcsBucketResource{
				Metadata: wsm.ResourceMetadata{
					ResourceId:      resourceId,
					Name:            "test-bucket",
					ResourceType:    wsm.GCSBUCKET,
					StewardshipType: wsm.CONTROLLED,
					CreatedDate:     now,
					CreatedBy:       "user@example.com",
					LastUpdatedDate: now,
					LastUpdatedBy:   "user@example.com",
				},
				Attributes: wsm.GcpGcsBucketAttributes{
					BucketName: "project-123-test-bucket",
				},
			},
			want: GCSBucketModel{
				ResourceModel: ResourceModel{
					ID:              types.StringValue(resourceId.String()),
					WorkspaceID:     types.StringValue("workspace-id"),
					Name:            types.StringValue("test-bucket"),
					CreatedAt:       timetypes.NewRFC3339TimeValue(now),
					CreatedBy:       types.StringValue("user@example.com"),
					UpdatedAt:       timetypes.NewRFC3339TimeValue(now),
					UpdatedBy:       types.StringValue("user@example.com"),
					ResourceType:    types.StringValue(string(wsm.GCSBUCKET)),
					StewardshipType: types.StringValue(string(wsm.CONTROLLED)),
					ResourceLineage: BuildResourceLineageModels(nil),
				},
				BucketName: types.StringValue("project-123-test-bucket"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewGCSBucketModel(tt.b, types.StringValue("workspace-id"))
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("NewGCSBucketModel() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestToUpdateControlledGcpGcsBucketRequestBody(t *testing.T) {
	resourceId := uuid.New()
	newFolderId := uuid.New()
	workspaceId := uuid.New()
	newWorkspaceId := uuid.New()
	tests := []struct {
		name string
		m    ControlledGCSBucketModel
		new  ControlledGCSBucketModel
		want *wsm.UpdateControlledGcpGcsBucketRequestBody
		err  error
	}{
		{
			name: "all fields updated",
			m: ControlledGCSBucketModel{
				GCSBucketModel: GCSBucketModel{
					ResourceModel: ResourceModel{
						ID:               types.StringValue(resourceId.String()),
						Name:             types.StringValue("test-bucket"),
						DisplayName:      types.StringValue("Test Bucket"),
						Description:      types.StringValue("This is a test GCS bucket"),
						CloneInstruction: types.StringValue("COPY_NOTHING"),
						FolderID:         types.StringValue(uuid.New().String()),
					},
					BucketName: types.StringValue("project-123-test-bucket"),
				},
				Location:     types.StringValue("us-central1"),
				StorageClass: types.StringValue("STANDARD"),
			},
			new: ControlledGCSBucketModel{
				GCSBucketModel: GCSBucketModel{
					ResourceModel: ResourceModel{
						ID:               types.StringValue(resourceId.String()),
						Name:             types.StringValue("updated-bucket"),
						DisplayName:      types.StringValue("Updated Bucket"),
						Description:      types.StringValue("This is an updated test GCS bucket"),
						CloneInstruction: types.StringValue("COPY_REFERENCE"),
						FolderID:         types.StringValue(newFolderId.String()),
					},
					BucketName: types.StringValue("project-123-test-bucket"),
				},
				Location:     types.StringValue("us-central1"),
				StorageClass: types.StringValue("NEARLINE"),
			},
			want: &wsm.UpdateControlledGcpGcsBucketRequestBody{
				Description:    client.Ptr("This is an updated test GCS bucket"),
				DisplayName:    client.Ptr(wsm.Name("Updated Bucket")),
				Name:           client.Ptr(wsm.Name("updated-bucket")),
				UpdateFolderId: &wsm.UpdateFolderId{FolderId: client.Ptr(newFolderId)},
				UpdateParameters: &wsm.GcpGcsBucketUpdateParameters{
					DefaultStorageClass: client.Ptr(wsm.GcpGcsBucketDefaultStorageClass("NEARLINE")),
					CloningInstructions: client.Ptr(wsm.COPYREFERENCE),
				},
			},
		},
		{
			name: "minimal update",
			m: ControlledGCSBucketModel{
				GCSBucketModel: GCSBucketModel{
					ResourceModel: ResourceModel{
						ID:               types.StringValue(resourceId.String()),
						Name:             types.StringValue("test-bucket"),
						DisplayName:      types.StringValue("Test Bucket"),
						Description:      types.StringValue("This is a test GCS bucket"),
						CloneInstruction: types.StringValue("COPY_NOTHING"),
					},
					BucketName: types.StringValue("project-123-test-bucket"),
				},
				Location:     types.StringValue("us-central1"),
				StorageClass: types.StringValue("STANDARD"),
			},
			new: ControlledGCSBucketModel{
				GCSBucketModel: GCSBucketModel{
					ResourceModel: ResourceModel{
						ID:               types.StringValue(resourceId.String()),
						Name:             types.StringValue("update-bucket"),
						DisplayName:      types.StringValue("Test Bucket"),
						Description:      types.StringValue("This is a test GCS bucket"),
						CloneInstruction: types.StringValue("COPY_NOTHING"),
					},
					BucketName: types.StringValue("project-123-test-bucket"),
				},
				Location:     types.StringValue("us-central1"),
				StorageClass: types.StringValue("STANDARD"),
			},
			want: &wsm.UpdateControlledGcpGcsBucketRequestBody{
				Name: client.Ptr(wsm.Name("update-bucket")),
			},
		},
		{
			name: "error update bucket name",
			m: ControlledGCSBucketModel{
				GCSBucketModel: GCSBucketModel{
					ResourceModel: ResourceModel{
						ID:               types.StringValue(resourceId.String()),
						Name:             types.StringValue("test-bucket"),
						DisplayName:      types.StringValue("Test Bucket"),
						Description:      types.StringValue("This is a test GCS bucket"),
						CloneInstruction: types.StringValue("COPY_NOTHING"),
					},
					BucketName: types.StringValue("project-123-test-bucket"),
				},
				Location:     types.StringValue("us-central1"),
				StorageClass: types.StringValue("STANDARD"),
			},
			new: ControlledGCSBucketModel{
				GCSBucketModel: GCSBucketModel{
					ResourceModel: ResourceModel{
						ID:               types.StringValue(resourceId.String()),
						Name:             types.StringValue("update-bucket"),
						DisplayName:      types.StringValue("Test Bucket"),
						Description:      types.StringValue("This is a test GCS bucket"),
						CloneInstruction: types.StringValue("COPY_NOTHING"),
					},
					BucketName: types.StringValue("project-123-update-bucket"),
				},
				Location:     types.StringValue("us-central1"),
				StorageClass: types.StringValue("STANDARD"),
			},
			want: nil,
			err:  fmt.Errorf("cannot update the bucket name of a controlled GCS bucket: project-123-test-bucket != project-123-update-bucket"),
		},
		{
			name: "error update location",
			m: ControlledGCSBucketModel{
				GCSBucketModel: GCSBucketModel{
					ResourceModel: ResourceModel{
						ID:               types.StringValue(resourceId.String()),
						Name:             types.StringValue("test-bucket"),
						DisplayName:      types.StringValue("Test Bucket"),
						Description:      types.StringValue("This is a test GCS bucket"),
						CloneInstruction: types.StringValue("COPY_NOTHING"),
					},
					BucketName: types.StringValue("project-123-test-bucket"),
				},
				Location:     types.StringValue("us-central1"),
				StorageClass: types.StringValue("STANDARD"),
			},
			new: ControlledGCSBucketModel{
				GCSBucketModel: GCSBucketModel{
					ResourceModel: ResourceModel{
						ID:               types.StringValue(resourceId.String()),
						Name:             types.StringValue("update-bucket"),
						DisplayName:      types.StringValue("Test Bucket"),
						Description:      types.StringValue("This is a test GCS bucket"),
						CloneInstruction: types.StringValue("COPY_NOTHING"),
					},
					BucketName: types.StringValue("project-123-test-bucket"),
				},
				Location:     types.StringValue("europe-west1"),
				StorageClass: types.StringValue("STANDARD"),
			},
			want: nil,
			err:  fmt.Errorf("cannot update the location of a controlled GCS bucket: us-central1 != europe-west1"),
		},
		{
			name: "error update workspace id",
			m: ControlledGCSBucketModel{
				GCSBucketModel: GCSBucketModel{
					ResourceModel: ResourceModel{
						ID:               types.StringValue(resourceId.String()),
						WorkspaceID:      types.StringValue(workspaceId.String()),
						Name:             types.StringValue("test-bucket"),
						DisplayName:      types.StringValue("Test Bucket"),
						Description:      types.StringValue("This is a test GCS bucket"),
						CloneInstruction: types.StringValue("COPY_NOTHING"),
					},
					BucketName: types.StringValue("project-123-test-bucket"),
				},
				Location:     types.StringValue("us-central1"),
				StorageClass: types.StringValue("STANDARD"),
			},
			new: ControlledGCSBucketModel{
				GCSBucketModel: GCSBucketModel{
					ResourceModel: ResourceModel{
						ID:               types.StringValue(resourceId.String()),
						WorkspaceID:      types.StringValue(newWorkspaceId.String()),
						Name:             types.StringValue("update-bucket"),
						DisplayName:      types.StringValue("Test Bucket"),
						Description:      types.StringValue("This is a test GCS bucket"),
						CloneInstruction: types.StringValue("COPY_NOTHING"),
					},
					BucketName: types.StringValue("project-123-test-bucket"),
				},
				Location:     types.StringValue("us-central1"),
				StorageClass: types.StringValue("STANDARD"),
			},
			want: nil,
			err:  fmt.Errorf("cannot update a controlled GCS bucket to a different workspace: %s != %s", workspaceId.String(), newWorkspaceId.String()),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.m.ToUpdateControlledGcpGcsBucketRequestBody(tt.new)
			if err != nil {
				if tt.err == nil {
					t.Errorf("ToUpdateControlledGcpGcsBucketRequestBody() unexpected error: %v", err)
					return
				}
				if tt.err.Error() != err.Error() {
					t.Errorf("ToUpdateControlledGcpGcsBucketRequestBody() error mismatch: got %v, want %v", err, tt.err)
					return
				}
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("ToUpdateControlledGcpGcsBucketRequestBody() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
