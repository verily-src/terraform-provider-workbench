package models

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/verily-src/terraform-provider-workbench/internal/openapi/wsm"
	wsmClient "github.com/verily-src/terraform-provider-workbench/internal/workspace-manager/goclient"
)

func TestResourceModelToControlledResourceCreateRequest(t *testing.T) {
	folderId := uuid.New()
	tests := []struct {
		name  string
		model ResourceModel
		want  wsm.ControlledResourceCommonFields
	}{
		{
			name: "all fields set",
			model: ResourceModel{
				Name:             types.StringValue("test-resource"),
				DisplayName:      types.StringValue("Test Resource"),
				Description:      types.StringValue("This is a test resource"),
				Properties:       &[]PropertyModel{{Key: types.StringValue("key1"), Value: types.StringValue("value1")}},
				CloneInstruction: types.StringValue("COPY_NOTHING"),
				FolderID:         types.StringValue(folderId.String()),
			},
			want: wsm.ControlledResourceCommonFields{
				AccessScope:         wsm.SHAREDACCESS,
				Name:                wsmClient.Ptr(wsm.Name("test-resource")),
				DisplayName:         wsmClient.Ptr(wsm.Name("Test Resource")),
				Description:         wsmClient.Ptr("This is a test resource"),
				CloningInstructions: wsm.COPYNOTHING,
				Properties:          wsmClient.Ptr([]wsm.Property{{Key: "key1", Value: "value1"}}),
				FolderId:            wsmClient.Ptr(folderId),
				ManagedBy:           wsm.USER,
			},
		},
		{
			name: "optional fields nil",
			model: ResourceModel{
				Name:             types.StringValue("test-resource"),
				CloneInstruction: types.StringValue("COPY_NOTHING"),
			},
			want: wsm.ControlledResourceCommonFields{
				AccessScope:         wsm.SHAREDACCESS,
				Name:                wsmClient.Ptr(wsm.Name("test-resource")),
				CloningInstructions: wsm.COPYNOTHING,
				ManagedBy:           wsm.USER,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.model.toControlledResourceCreateRequest()
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("toControlledResourceCreateRequest() mismatch (-want +got):\n%s", diff)
			}
		})
	}

}

func TestNewResourceModel(t *testing.T) {
	resourceId := uuid.New()
	folderId := uuid.New()
	now := time.Now()
	rLineage := []wsm.ResourceLineageEntry{
		{
			SourceResourceId:  uuid.New(),
			SourceWorkspaceId: uuid.New(),
		},
		{
			SourceResourceId:  uuid.New(),
			SourceWorkspaceId: uuid.New(),
		},
	}
	rLinageModel := BuildResourceLineageModels(&rLineage)

	tests := []struct {
		name string
		b    wsm.ResourceMetadata
		want ResourceModel
	}{
		{
			name: "all fields set",
			b: wsm.ResourceMetadata{
				ResourceId:          resourceId,
				Name:                wsm.Name("test-resource"),
				DisplayName:         wsmClient.Ptr("Test Resource"),
				Description:         wsmClient.Ptr("This is a test resource"),
				CreatedDate:         now,
				CreatedBy:           "user@example.com",
				LastUpdatedDate:     now,
				LastUpdatedBy:       "user@example.com",
				ResourceType:        wsm.GCSOBJECT,
				StewardshipType:     wsm.CONTROLLED,
				Properties:          wsmClient.Ptr([]wsm.Property{{Key: "key1", Value: "value1"}}),
				ResourceLineage:     wsmClient.Ptr(rLineage),
				FolderId:            wsmClient.Ptr(folderId),
				CloningInstructions: wsmClient.Ptr(wsm.COPYNOTHING),
			},
			want: ResourceModel{
				ID:               types.StringValue(resourceId.String()),
				WorkspaceID:      types.StringValue("workspace-id"),
				Name:             types.StringValue("test-resource"),
				DisplayName:      types.StringValue("Test Resource"),
				Description:      types.StringValue("This is a test resource"),
				CreatedAt:        timetypes.NewRFC3339TimeValue(now),
				CreatedBy:        types.StringValue("user@example.com"),
				UpdatedAt:        timetypes.NewRFC3339TimeValue(now),
				UpdatedBy:        types.StringValue("user@example.com"),
				ResourceType:     types.StringValue("GCS_OBJECT"),
				StewardshipType:  types.StringValue("CONTROLLED"),
				Properties:       &[]PropertyModel{{Key: types.StringValue("key1"), Value: types.StringValue("value1")}},
				ResourceLineage:  rLinageModel,
				FolderID:         types.StringValue(folderId.String()),
				CloneInstruction: types.StringValue("COPY_NOTHING"),
			},
		},
		{
			name: "minimal fields",
			b: wsm.ResourceMetadata{
				ResourceId:      resourceId,
				Name:            wsm.Name("test-resource"),
				CreatedDate:     now,
				CreatedBy:       "user@example.com",
				LastUpdatedDate: now,
				LastUpdatedBy:   "user@example.com",
				ResourceType:    wsm.GCSOBJECT,
				StewardshipType: wsm.CONTROLLED,
			},
			want: ResourceModel{
				ID:              types.StringValue(resourceId.String()),
				WorkspaceID:     types.StringValue("workspace-id"),
				Name:            types.StringValue("test-resource"),
				CreatedAt:       timetypes.NewRFC3339TimeValue(now),
				CreatedBy:       types.StringValue("user@example.com"),
				UpdatedAt:       timetypes.NewRFC3339TimeValue(now),
				UpdatedBy:       types.StringValue("user@example.com"),
				ResourceType:    types.StringValue("GCS_OBJECT"),
				StewardshipType: types.StringValue("CONTROLLED"),
				ResourceLineage: BuildResourceLineageModels(nil),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewResourceModel(tt.b, types.StringValue("workspace-id"))
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("NewResourceModel() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
