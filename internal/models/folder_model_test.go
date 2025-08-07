package models

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/verily-src/terraform-provider-workbench/internal/client"
	"github.com/verily-src/terraform-provider-workbench/internal/openapi/wsm"
)

func TestNewFolder(t *testing.T) {
	now := time.Now()
	id := uuid.New()
	parentFolderId := uuid.New()

	tests := []struct {
		name string
		m    *wsm.Folder
		want *FolderModel
	}{
		{
			name: "Test New FolderModel",
			m: &wsm.Folder{
				Id:          id,
				DisplayName: "Test Folder",
				Description: client.Ptr("This is a test folder"),
				Properties: &wsm.Properties{
					{
						Key:   SHORT_DESCRIPTION_KEY,
						Value: "This is a short description",
					},
					{
						Key:   RELEASE_NOTES_URL_KEY,
						Value: "https://example.com/release-notes",
					},
					{
						Key:   "key1",
						Value: "value1",
					},
				},
				LastUpdatedDate: now,
				LastUpdatedBy:   "test-user",
				CreatedDate:     now,
				CreatedBy:       "test-user",
			},
			want: &FolderModel{
				ID:          types.StringValue(id.String()),
				WorkspaceId: types.StringValue("workspace-id"),
				DisplayName: types.StringValue("Test Folder"),
				Description: types.StringPointerValue(client.Ptr("This is a test folder")),
				Properties: &[]PropertyModel{
					{
						Key:   types.StringValue("key1"),
						Value: types.StringValue("value1"),
					},
				},
				LastUpdatedDate: timetypes.NewRFC3339TimeValue(now),
				LastUpdatedBy:   types.StringValue("test-user"),
				CreatedDate:     timetypes.NewRFC3339TimeValue(now),
				CreatedBy:       types.StringValue("test-user"),
			},
		},
		{
			name: "Test New FolderModel, have parent folder",
			m: &wsm.Folder{
				Id:          id,
				DisplayName: "Test Folder",
				Description: client.Ptr("This is a test folder"),
				Properties: &wsm.Properties{
					{
						Key:   SHORT_DESCRIPTION_KEY,
						Value: "This is a short description",
					},
					{
						Key:   RELEASE_NOTES_URL_KEY,
						Value: "https://example.com/release-notes",
					},
					{
						Key:   "key1",
						Value: "value1",
					},
				},
				ParentFolderId:  client.Ptr(parentFolderId),
				LastUpdatedDate: now,
				LastUpdatedBy:   "test-user",
				CreatedDate:     now,
				CreatedBy:       "test-user",
			},
			want: &FolderModel{
				ID:          types.StringValue(id.String()),
				WorkspaceId: types.StringValue("workspace-id"),
				DisplayName: types.StringValue("Test Folder"),
				Description: types.StringPointerValue(client.Ptr("This is a test folder")),
				Properties: &[]PropertyModel{
					{
						Key:   types.StringValue("key1"),
						Value: types.StringValue("value1"),
					},
				},
				ParentFolderId:  types.StringValue(parentFolderId.String()),
				LastUpdatedDate: timetypes.NewRFC3339TimeValue(now),
				LastUpdatedBy:   types.StringValue("test-user"),
				CreatedDate:     timetypes.NewRFC3339TimeValue(now),
				CreatedBy:       types.StringValue("test-user"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewFolderModel(tt.m, "workspace-id")
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("NewFolderModel() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFolderToCreateRequest(t *testing.T) {
	parentFolderId := uuid.New()

	tests := []struct {
		name string
		m    *FolderModel
		want *wsm.CreateFolderJSONRequestBody
	}{
		{
			name: "Test ToCreateRequest with properties",
			m: &FolderModel{
				WorkspaceId: types.StringValue("workspace-id"),
				DisplayName: types.StringValue("Test Folder"),
				Description: types.StringPointerValue(client.Ptr("This is a test folder")),
				Properties: &[]PropertyModel{
					{
						Key:   types.StringValue("key1"),
						Value: types.StringValue("value1"),
					},
					{
						Key:   types.StringValue("key2"),
						Value: types.StringValue("value2"),
					},
				},
			},
			want: &wsm.CreateFolderJSONRequestBody{
				DisplayName: "Test Folder",
				Description: client.Ptr("This is a test folder"),
				Properties: &wsm.Properties{
					{
						Key:   "key1",
						Value: "value1",
					},
					{
						Key:   "key2",
						Value: "value2",
					},
				},
			},
		},
		{
			name: "Test ToCreateRequest with parent folder ID",
			m: &FolderModel{
				WorkspaceId:    types.StringValue("workspace-id"),
				DisplayName:    types.StringValue("Test Folder"),
				Description:    types.StringPointerValue(client.Ptr("This is a test folder")),
				ParentFolderId: types.StringValue(parentFolderId.String()),
				Properties: &[]PropertyModel{
					{
						Key:   types.StringValue("key1"),
						Value: types.StringValue("value1"),
					},
					{
						Key:   types.StringValue("key2"),
						Value: types.StringValue("value2"),
					},
				},
			},
			want: &wsm.CreateFolderJSONRequestBody{
				DisplayName:    "Test Folder",
				Description:    client.Ptr("This is a test folder"),
				ParentFolderId: &parentFolderId,
				Properties: &wsm.Properties{
					{
						Key:   "key1",
						Value: "value1",
					},
					{
						Key:   "key2",
						Value: "value2",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.m.ToCreateRequest()
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("ToCreateRequest() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
