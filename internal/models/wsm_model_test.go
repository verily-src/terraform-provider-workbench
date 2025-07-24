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

func TestNewWorkspaceModel(t *testing.T) {
	id := uuid.New()
	crgId := uuid.New()
	orgId := uuid.New()
	now := time.Now()
	tests := []struct {
		name      string
		workspace *wsm.WorkspaceDescription
		want      *WorkspaceModel
	}{
		{
			name: "Test New WorkspaceModel",
			workspace: &wsm.WorkspaceDescription{
				Id:              id,
				UserFacingId:    "test-workspace",
				Description:     client.Ptr("This is a test workspace"),
				DisplayName:     client.Ptr("Test Workspace"),
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
				},
				),
				Properties: buildProperties(map[string]string{
					WorkspacePropertyDefaultLocation: "us-central1",
					"key2":                           "value2",
				}),
			},
			want: &WorkspaceModel{
				ID:              types.StringValue(id.String()),
				UserFacingId:    types.StringValue("test-workspace"),
				Description:     types.StringValue("This is a test workspace"),
				DisplayName:     types.StringValue("Test Workspace"),
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
		},
		{
			name: "Test New WorkspaceModel minimum",
			workspace: &wsm.WorkspaceDescription{
				Id:              id,
				UserFacingId:    "test-workspace",
				Description:     nil,
				DisplayName:     nil,
				OrgId:           &orgId,
				CrgId:           &crgId,
				CreatedBy:       "test-user",
				CreatedDate:     now,
				LastUpdatedBy:   "test-user",
				LastUpdatedDate: now,
			},
			want: &WorkspaceModel{
				ID:              types.StringValue(id.String()),
				UserFacingId:    types.StringValue("test-workspace"),
				OrganizationID:  types.StringValue(orgId.String()),
				PodID:           types.StringValue(crgId.String()),
				CreatedBy:       types.StringValue("test-user"),
				CreatedDate:     timetypes.NewRFC3339TimeValue(now),
				LastUpdatedBy:   types.StringValue("test-user"),
				LastUpdatedDate: timetypes.NewRFC3339TimeValue(now),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewWorkspaceModel(tt.workspace)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("NewWorkspaceModel() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestConvertToCreateRequest(t *testing.T) {
	crgId := uuid.New().String()
	orgId := uuid.New().String()
	stage := wsm.CRGWORKSPACE
	tests := []struct {
		name      string
		workspace *WorkspaceModel
		want      wsm.CreateWorkspaceV2JSONRequestBody
	}{
		{
			name: "Test ConvertToCreateRequest all fields",
			workspace: &WorkspaceModel{
				UserFacingId:   types.StringValue("test-workspace"),
				Description:    types.StringValue("This is a test workspace"),
				DisplayName:    types.StringValue("Test Workspace"),
				OrganizationID: types.StringValue(orgId),
				PodID:          types.StringValue(crgId),
				Policies: &[]PolicyModel{
					{
						Namespace: types.StringValue("terra"),
						Name:      types.StringValue("exfil-perimeter-constraint"),
						AdditionalData: &[]AdditionalDataModel{
							{
								Key:   types.StringValue("perimeter-id"),
								Value: types.StringValue("fake-perimeter-id"),
							},
						},
					},
				},
				Properties: &[]PropertyModel{
					{
						Key:   types.StringValue("key1"),
						Value: types.StringValue("value1"),
					},
				},
				Location: types.StringValue("us"),
			},
			want: wsm.CreateWorkspaceV2JSONRequestBody{
				UserFacingId:         client.Ptr("test-workspace"),
				Description:          client.Ptr("This is a test workspace"),
				DisplayName:          client.Ptr("Test Workspace"),
				CloudResourceGroupId: &crgId,
				OrganizationId:       &orgId,
				Properties: buildProperties(map[string]string{
					"key1":                   "value1",
					"terra-default-location": "us",
				}),
				Policies: &wsm.WsmPolicyInputs{
					Inputs: *buildPolicies([]testPolicy{
						{
							Namespace: "terra",
							Name:      "exfil-perimeter-constraint",
							AdditionalData: map[string]string{
								"perimeter-id": "fake-perimeter-id",
							},
						},
					}),
				},
				Stage: &stage,
			},
		},
		{
			name: "Test ConvertToCreateRequest minimum fields",
			workspace: &WorkspaceModel{
				UserFacingId:   types.StringValue("test-workspace"),
				OrganizationID: types.StringValue(orgId),
				PodID:          types.StringValue(crgId),
			},
			want: wsm.CreateWorkspaceV2JSONRequestBody{
				UserFacingId:         client.Ptr("test-workspace"),
				CloudResourceGroupId: &crgId,
				OrganizationId:       &orgId,
				Properties: buildProperties(map[string]string{
					WorkspacePropertyDefaultLocation: "us-central1",
				}),
				Stage: &stage,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.workspace.ConvertToCreateRequest()

			tt.want.Id = got.Id
			tt.want.JobControl.Id = got.JobControl.Id

			if ok := compareProperties(tt.want.Properties, got.Properties); !ok {
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

func compareProperties(a, b *[]wsm.Property) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if len(*a) != len(*b) {
		return false
	}
	mapA := make(map[string]string)
	for _, p := range *a {
		mapA[p.Key] = p.Value
	}
	for _, p := range *b {
		if val, ok := mapA[p.Key]; !ok || val != p.Value {
			return false
		}
	}
	return true
}

type testPolicy struct {
	Namespace      string
	Name           string
	AdditionalData map[string]string
}

func buildPolicies(policies []testPolicy) *[]wsm.WsmPolicyInput {
	var inputs []wsm.WsmPolicyInput
	for _, p := range policies {
		var additionalData []wsm.WsmPolicyPair
		for k, v := range p.AdditionalData {
			additionalData = append(additionalData, wsm.WsmPolicyPair{Key: client.Ptr(k), Value: client.Ptr(v)})
		}
		inputs = append(inputs, wsm.WsmPolicyInput{
			Namespace:      p.Namespace,
			Name:           p.Name,
			AdditionalData: &additionalData,
		})
	}
	return &inputs
}

func buildProperties(properties map[string]string) *[]wsm.Property {
	var inputs []wsm.Property
	for k, v := range properties {
		inputs = append(inputs, wsm.Property{
			Key:   k,
			Value: v,
		})
	}
	return &inputs
}
