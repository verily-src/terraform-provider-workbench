package models

import (
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/verily-src/terraform-provider-workbench/internal/openapi/wsm"
)

const (
	SHORT_DESCRIPTION_KEY = "terra-short-description"
	RELEASE_NOTES_URL_KEY = "terra-release-notes-url"
	PUBLISHED_DATE_KEY    = "terra-published-date"
)

// FolderModel is the description of the folder resource.
type VersionModel struct {
	// The unique ID of the workspace.
	FolderModel
	// ReleaseNotesURL is the URL to the release notes for the version.
	ReleaseNotesURL types.String `tfsdk:"release_notes_url"`
	// Published indicates whether the version is published.
	Published types.Bool `tfsdk:"published"`
	// PublishedDate is the date when the version was published.
	PublishedDate timetypes.RFC3339 `tfsdk:"published_date"`
}

func NewVersionModel(f *wsm.Folder, workspaceId string) (*VersionModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	if f == nil {
		diags.AddError("Invalid folder object", "Folder is nil")
		return nil, diags
	}
	folderModel := NewFolderModel(f, workspaceId)

	// Handle publishing properties extraction
	publish_date_str := GetValue(f.Properties, PUBLISHED_DATE_KEY)
	var published bool
	var publish_date_time time.Time

	// If there is no published date property, we assume the version is not published.
	if publish_date_str != "" {
		parsedTime, err := time.Parse(time.RFC3339, GetValue(f.Properties, PUBLISHED_DATE_KEY))
		if err != nil {
			diags.AddError("Invalid published date", fmt.Sprintf("Failed to parse \"%s published date in RFC3339", GetValue(f.Properties, PUBLISHED_DATE_KEY)))
			return nil, diags
		}
		publish_date_time = parsedTime
		published = true
	} else {
		publish_date_time = time.Time{}
		published = false
	}
	// Extract release notes URL, if available
	release_url_str := GetValue(f.Properties, RELEASE_NOTES_URL_KEY)
	var release_url = types.StringValue(release_url_str)
	if release_url_str == "" {
		release_url = types.StringNull()
	}

	return &VersionModel{
		FolderModel:     *folderModel,
		ReleaseNotesURL: release_url,
		Published:       types.BoolValue(published),
		PublishedDate:   timetypes.NewRFC3339TimeValue(publish_date_time),
	}, diags
}

// ToCreateRequest converts the VersionModel to a CreateFolderJSONRequestBody.
func (v *VersionModel) ToCreateRequest() *wsm.CreateFolderJSONRequestBody {
	payload := v.FolderModel.ToCreateRequest()
	if payload.Properties == nil {
		payload.Properties = &wsm.Properties{}
	}
	payload.Properties = packVersionProperties(v, payload.Properties)
	return payload
}

func packVersionProperties(v *VersionModel, payloadProps *wsm.Properties) *wsm.Properties {
	var packedProps = *payloadProps
	if !v.ReleaseNotesURL.IsNull() {
		packedProps = append(packedProps, wsm.Property{
			Key:   RELEASE_NOTES_URL_KEY,
			Value: v.ReleaseNotesURL.ValueString(),
		})
	}
	if v.Published.ValueBool() {
		packedProps = append(packedProps, wsm.Property{
			Key:   PUBLISHED_DATE_KEY,
			Value: time.Now().Format(time.RFC3339),
		})
	}
	return &packedProps
}
