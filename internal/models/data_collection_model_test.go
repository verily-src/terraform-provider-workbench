package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/verily-src/terraform-provider-workbench/internal/client"
	"github.com/verily-src/terraform-provider-workbench/internal/openapi/wsm"
)

func TestNewDataCollectionModel(t *testing.T) {
	id := uuid.New()
	crgId := uuid.New()
	orgId := uuid.New()
	now := time.Now()
	therapeuticTags := []string{"cardiology", "oncology"}
	therapeuticTagsJSON, _ := json.Marshal(therapeuticTags)
	therapeuticTagsStr := string(therapeuticTagsJSON)

	therapeuticTagsList := make([]attr.Value, len(therapeuticTags))
	for i, tag := range therapeuticTags {
		therapeuticTagsList[i] = types.StringValue(tag)
	}
	tfTagsList, _ := types.SetValue(types.StringType, therapeuticTagsList)

	var nilPropertiesSlice []PropertyModel

	tests := []struct {
		name      string
		workspace *wsm.WorkspaceDescription
		want      *DataCollectionModel
	}{
		{
			name: "Test New DataCollectionModel",
			workspace: &wsm.WorkspaceDescription{
				Id:              id,
				UserFacingId:    "test-data-collection",
				Description:     client.Ptr("This is a test data collection"),
				DisplayName:     client.Ptr("Test Data Collection"),
				OrgId:           &orgId,
				CrgId:           &crgId,
				CreatedBy:       "test-user",
				CreatedDate:     now,
				LastUpdatedBy:   "test-user",
				LastUpdatedDate: now,
				Policies: buildPolicies([]testPolicy{
					{
						Namespace: "test-namespace",
						Name:      "test-policy",
						AdditionalData: map[string]string{
							"key1": "value1",
						},
					},
				}),
				Properties: buildProperties(map[string]string{
					WorkspacePropertyDefaultLocation: "us-central1",
					SUPPORT_EMAIL_KEY:                "support@example.com",
					ORGANIZATION_NAME_TAG:            "Test Org",
					THERAPEUTIC_TAGS_KEY:             therapeuticTagsStr,
					UPDATE_FREQUENCY_KEY:             "weekly",
					"terra-type":                     "data-collection",
					"key2":                           "value2",
				}),
			},
			want: &DataCollectionModel{
				WorkspaceModel: WorkspaceModel{
					ID:              types.StringValue(id.String()),
					UserFacingId:    types.StringValue("test-data-collection"),
					Description:     types.StringValue("This is a test data collection"),
					DisplayName:     types.StringValue("Test Data Collection"),
					OrganizationID:  types.StringValue(orgId.String()),
					PodID:           types.StringValue(crgId.String()),
					CreatedBy:       types.StringValue("test-user"),
					CreatedDate:     timetypes.NewRFC3339TimeValue(now),
					LastUpdatedBy:   types.StringValue("test-user"),
					LastUpdatedDate: timetypes.NewRFC3339TimeValue(now),
					Policies: &[]PolicyModel{
						{
							Namespace: types.StringValue("test-namespace"),
							Name:      types.StringValue("test-policy"),
							AdditionalData: &[]AdditionalDataModel{
								{
									Key:   types.StringValue("key1"),
									Value: types.StringValue("value1"),
								},
							},
						},
					},
					Properties: &[]PropertyModel{
						{
							Key:   types.StringValue("key2"),
							Value: types.StringValue("value2"),
						},
					},
					Location: types.StringValue("us-central1"),
				},
				SupportEmail:     types.StringValue("support@example.com"),
				OrganizationName: types.StringValue("Test Org"),
				TherapeuticTags:  tfTagsList,
				UpdateFrequency:  types.StringValue("weekly"),
			},
		},
		{
			name: "Test New DataCollectionModel minimum",
			workspace: &wsm.WorkspaceDescription{
				Id:              id,
				UserFacingId:    "test-data-collection",
				OrgId:           &orgId,
				CrgId:           &crgId,
				CreatedBy:       "test-user",
				CreatedDate:     now,
				LastUpdatedBy:   "test-user",
				LastUpdatedDate: now,
				Properties: buildProperties(map[string]string{
					"terra-type": "data-collection",
				}),
			},
			want: &DataCollectionModel{
				WorkspaceModel: WorkspaceModel{
					ID:              types.StringValue(id.String()),
					UserFacingId:    types.StringValue("test-data-collection"),
					OrganizationID:  types.StringValue(orgId.String()),
					PodID:           types.StringValue(crgId.String()),
					CreatedBy:       types.StringValue("test-user"),
					CreatedDate:     timetypes.NewRFC3339TimeValue(now),
					LastUpdatedBy:   types.StringValue("test-user"),
					LastUpdatedDate: timetypes.NewRFC3339TimeValue(now),
					// Update the Properties field to expect a pointer to a slice containing the terra-type property
					// Note that terra-type is never tracked in the model, only injected when converting to a create request
					Properties: &nilPropertiesSlice,
				},
				SupportEmail:     types.StringValue(""),
				OrganizationName: types.StringValue(""),
				TherapeuticTags:  types.SetNull(types.StringType),
				UpdateFrequency:  types.StringValue(""),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := NewDataCollectionModel(tt.workspace)
			t.Logf("Properties type: %T, value: %#v", got.WorkspaceModel.Properties, got.WorkspaceModel.Properties)

			// Handle the case where TherapeuticTags may be empty list vs null
			if tt.want.TherapeuticTags.IsNull() && !got.TherapeuticTags.IsNull() {
				if len(got.TherapeuticTags.Elements()) == 0 {
					got.TherapeuticTags = types.SetNull(types.StringType)
				}
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("NewDataCollectionModel() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDataCollectionConvertToCreateRequest(t *testing.T) {
	crgId := uuid.New().String()
	orgId := uuid.New().String()
	id := uuid.New().String()
	stage := wsm.CRGWORKSPACE
	therapeuticTags := []string{"cardiology", "oncology"}
	therapeuticTagsJSON, _ := json.Marshal(therapeuticTags)
	therapeuticTagsStr := string(therapeuticTagsJSON)

	therapeuticTagsList := make([]attr.Value, len(therapeuticTags))
	for i, tag := range therapeuticTags {
		therapeuticTagsList[i] = types.StringValue(tag)
	}
	tfTagsSet, _ := types.SetValue(types.StringType, therapeuticTagsList)

	tests := []struct {
		name           string
		dataCollection *DataCollectionModel
		want           wsm.CreateWorkspaceV2JSONRequestBody
	}{
		{
			name: "Test DataCollectionConvertToCreateRequest all fields",
			dataCollection: &DataCollectionModel{
				WorkspaceModel: WorkspaceModel{
					ID:             types.StringValue(id),
					UserFacingId:   types.StringValue("test-data-collection"),
					Description:    types.StringValue("This is a test data collection"),
					DisplayName:    types.StringValue("Test Data Collection"),
					OrganizationID: types.StringValue(orgId),
					PodID:          types.StringValue(crgId),
					Location:       types.StringValue("us-central1"),
					Properties: &[]PropertyModel{
						{
							Key:   types.StringValue("key1"),
							Value: types.StringValue("value1"),
						},
					},
				},
				SupportEmail:     types.StringValue("support@example.com"),
				OrganizationName: types.StringValue("Test Org"),
				TherapeuticTags:  tfTagsSet,
				UpdateFrequency:  types.StringValue("weekly"),
			},
			want: wsm.CreateWorkspaceV2JSONRequestBody{
				Id:                   uuid.MustParse(id),
				UserFacingId:         client.Ptr("test-data-collection"),
				Description:          client.Ptr("This is a test data collection"),
				DisplayName:          client.Ptr("Test Data Collection"),
				CloudResourceGroupId: &crgId,
				OrganizationId:       &orgId,
				Properties: buildProperties(map[string]string{
					"key1":                           "value1",
					WorkspacePropertyDefaultLocation: "us-central1",
					SUPPORT_EMAIL_KEY:                "support@example.com",
					ORGANIZATION_NAME_TAG:            "Test Org",
					THERAPEUTIC_TAGS_KEY:             therapeuticTagsStr,
					UPDATE_FREQUENCY_KEY:             "weekly",
					"terra-type":                     "data-collection",
				}),
				Stage: &stage,
			},
		},
		{
			name: "Test DataCollectionConvertToCreateRequest minimum fields",
			dataCollection: &DataCollectionModel{
				WorkspaceModel: WorkspaceModel{
					ID:             types.StringValue(id),
					UserFacingId:   types.StringValue("test-data-collection"),
					OrganizationID: types.StringValue(orgId),
					PodID:          types.StringValue(crgId),
				},
			},
			want: wsm.CreateWorkspaceV2JSONRequestBody{
				Id:                   uuid.MustParse(id),
				UserFacingId:         client.Ptr("test-data-collection"),
				CloudResourceGroupId: &crgId,
				OrganizationId:       &orgId,
				Properties: buildProperties(map[string]string{
					WorkspacePropertyDefaultLocation: "us-central1",
					SUPPORT_EMAIL_KEY:                "",
					UPDATE_FREQUENCY_KEY:             "",
					ORGANIZATION_NAME_TAG:            "",
					THERAPEUTIC_TAGS_KEY:             "[]",
					"terra-type":                     "data-collection",
				}),
				Stage: &stage,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := tt.dataCollection.ConvertToCreateRequest()

			// JobControl.Id will be different each time
			tt.want.JobControl.Id = got.JobControl.Id

			if ok := compareProperties(tt.want.Properties, got.Properties); !ok {
				t.Errorf("%+v\n", got.Properties)
				t.Errorf("ConvertToCreateRequest() properties mismatch")
			}

			// Set the properties in the want to match the got because the map ordering may differ when
			// converting to a slice.
			tt.want.Properties = got.Properties

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("ConvertToCreateRequest() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestConvertTypeSetToJsonString(t *testing.T) {
	tests := []struct {
		name    string
		input   types.Set
		want    string
		wantErr bool
	}{
		{
			name: "Empty set",
			input: func() types.Set {
				set, _ := types.SetValue(types.StringType, []attr.Value{})
				return set
			}(),
			want:    "[]",
			wantErr: false,
		},
		{
			name: "Set with values",
			input: func() types.Set {
				set, _ := types.SetValue(types.StringType,
					[]attr.Value{
						types.StringValue("tag1"),
						types.StringValue("tag2"),
					},
				)
				return set
			}(),
			want:    `["tag1","tag2"]`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertTypeSetToJsonString(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("marshallTfStrings() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("marshallTfStrings() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUnmarshallJsonString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    types.Set
		wantErr bool
	}{
		{
			name:    "Empty array",
			input:   "[]",
			want:    types.SetNull(types.StringType),
			wantErr: false,
		},
		{
			name:  "Array with values",
			input: `["tag1","tag2"]`,
			want: types.SetValueMust(types.StringType, []attr.Value{
				types.StringValue("tag1"),
				types.StringValue("tag2"),
			}),
			wantErr: false,
		},
		{
			name:    "Invalid JSON",
			input:   `not json`,
			want:    types.SetNull(types.StringType),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := unmarshallJsonString(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("unmarshallJsonString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if !got.Equal(tt.want) {
					t.Errorf("unmarshallJsonString() values mismatch, got: %v of type %T want: %v of type %T", got.Elements(), got.Elements(), tt.want.Elements(), tt.want.Elements())
				}
			}
		})
	}
}
