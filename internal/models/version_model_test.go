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

func TestNewVersionModel(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	id := uuid.New()
	workspaceId := "workspace-id"

	t.Run("unpublished version", func(t *testing.T) {
		folder := &wsm.Folder{
			Id:          id,
			DisplayName: "Test Version",
			Description: client.Ptr("A test version"),
			Properties: &wsm.Properties{
				{
					Key:   SHORT_DESCRIPTION_KEY,
					Value: "Short desc",
				},
				{
					Key:   RELEASE_NOTES_URL_KEY,
					Value: "https://example.com/notes",
				},
			},
			LastUpdatedDate: now,
			LastUpdatedBy:   "user1",
			CreatedDate:     now,
			CreatedBy:       "user1",
		}
		got, diags := NewVersionModel(folder, workspaceId)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		want := &VersionModel{
			FolderModel: FolderModel{
				ID:              types.StringValue(id.String()),
				WorkspaceId:     types.StringValue(workspaceId),
				DisplayName:     types.StringValue("Test Version"),
				Description:     types.StringPointerValue(client.Ptr("A test version")),
				Properties:      convertProperties(folder.Properties),
				LastUpdatedDate: timetypes.NewRFC3339TimeValue(now),
				LastUpdatedBy:   types.StringValue("user1"),
				CreatedDate:     timetypes.NewRFC3339TimeValue(now),
				CreatedBy:       types.StringValue("user1"),
			},
			ReleaseNotesURL: types.StringValue("https://example.com/notes"),
			Published:       types.BoolValue(false),
			PublishedDate:   timetypes.NewRFC3339TimeValue(time.Time{}),
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("NewVersionModel() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("published version", func(t *testing.T) {
		publishDate := now.Format(time.RFC3339)
		folder := &wsm.Folder{
			Id:          id,
			DisplayName: "Test Version",
			Description: client.Ptr("A test version"),
			Properties: &wsm.Properties{
				{
					Key:   RELEASE_NOTES_URL_KEY,
					Value: "https://example.com/notes",
				},
				{
					Key:   PUBLISHED_DATE_KEY,
					Value: publishDate,
				},
			},
			LastUpdatedDate: now,
			LastUpdatedBy:   "user1",
			CreatedDate:     now,
			CreatedBy:       "user1",
		}
		got, diags := NewVersionModel(folder, workspaceId)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		parsedPublishDate, _ := time.Parse(time.RFC3339, publishDate)
		want := &VersionModel{
			FolderModel: FolderModel{
				ID:              types.StringValue(id.String()),
				WorkspaceId:     types.StringValue(workspaceId),
				DisplayName:     types.StringValue("Test Version"),
				Description:     types.StringPointerValue(client.Ptr("A test version")),
				Properties:      convertProperties(folder.Properties),
				LastUpdatedDate: timetypes.NewRFC3339TimeValue(now),
				LastUpdatedBy:   types.StringValue("user1"),
				CreatedDate:     timetypes.NewRFC3339TimeValue(now),
				CreatedBy:       types.StringValue("user1"),
			},
			ReleaseNotesURL: types.StringValue("https://example.com/notes"),
			Published:       types.BoolValue(true),
			PublishedDate:   timetypes.NewRFC3339TimeValue(parsedPublishDate),
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("NewVersionModel() mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestVersionToCreateRequest(t *testing.T) {
	id := uuid.New()
	workspaceId := "workspace-id"
	m := &VersionModel{
		FolderModel: FolderModel{
			ID:          types.StringValue(id.String()),
			WorkspaceId: types.StringValue(workspaceId),
			DisplayName: types.StringValue("Test Version"),
			Description: types.StringPointerValue(client.Ptr("A test version")),
			Properties: &[]PropertyModel{
				{
					Key:   types.StringValue("key1"),
					Value: types.StringValue("value1"),
				},
			},
		},
		ReleaseNotesURL: types.StringValue("https://example.com/notes"),
		Published:       types.BoolValue(true),
	}
	got := m.ToCreateRequest()
	if got.DisplayName != "Test Version" {
		t.Errorf("DisplayName = %q, want %q", got.DisplayName, "Test Version")
	}
	if got.Description == nil || *got.Description != "A test version" {
		t.Errorf("Description = %v, want %q", got.Description, "A test version")
	}
	if got.Properties == nil || len(*got.Properties) == 0 {
		t.Errorf("Properties should not be nil or empty")
	}
	foundReleaseNotes := false
	foundPublishedDate := false
	for _, p := range *got.Properties {
		if p.Key == RELEASE_NOTES_URL_KEY && p.Value == "https://example.com/notes" {
			foundReleaseNotes = true
		}
		if p.Key == PUBLISHED_DATE_KEY {
			foundPublishedDate = true
		}
	}
	if !foundReleaseNotes {
		t.Errorf("RELEASE_NOTES_URL_KEY not found in properties")
	}
	if !foundPublishedDate {
		t.Errorf("PUBLISHED_DATE_KEY not found in properties")
	}
}
