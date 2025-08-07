package models

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/verily-src/terraform-provider-workbench/internal/client"
	"github.com/verily-src/terraform-provider-workbench/internal/openapi/user"
)

func TestNewGroupModel(t *testing.T) {
	orgUfid := "test-org-ufid"
	orgId := fmt.Sprintf("~%s", orgUfid)
	now := time.Now()
	tests := []struct {
		name  string
		group *user.GroupDescriptionAndRoles
		want  *GroupModel
	}{
		{
			name: "Test New WorkspaceModel new",
			group: &user.GroupDescriptionAndRoles{
				GroupDescription: &user.GroupDescription{
					GroupEmail:             "test-group@gmail.com",
					GroupName:              client.Ptr("test-group"),
					InternalName:           client.Ptr("test-group-internal"),
					OrgId:                  orgId,
					OrgUfid:                orgUfid,
					CreatedBy:              "test-user",
					CreatedDate:            now,
					LastUpdatedBy:          "test-user",
					LastUpdatedDate:        now,
					ExpirationDays:         30,
					ExpirationNotification: boolPtr(true),
					RequireGrantReason:     boolPtr(true),
					SyncGroup:              true,
					Description:            client.Ptr("This is a test group"),
				},
			},
			want: &GroupModel{
				GroupEmail:             types.StringValue("test-group@gmail.com"),
				GroupName:              types.StringValue("test-group"),
				InternalName:           types.StringValue("test-group-internal"),
				OrgId:                  types.StringValue(orgId),
				OrgUfid:                types.StringValue(orgUfid),
				CreatedBy:              types.StringValue("test-user"),
				CreatedDate:            timetypes.NewRFC3339TimeValue(now),
				LastUpdatedBy:          types.StringValue("test-user"),
				LastUpdatedDate:        timetypes.NewRFC3339TimeValue(now),
				ExpirationDays:         types.Int64Value(30),
				ExpirationNotification: types.BoolValue(true),
				RequireGrantReason:     types.BoolValue(true),
				SyncGroup:              types.BoolValue(true),
				Description:            types.StringValue("This is a test group"),
			},
		},
		{
			name: "Test New WorkspaceModel new minimum",
			group: &user.GroupDescriptionAndRoles{
				GroupDescription: &user.GroupDescription{
					GroupEmail:      "test-group@gmail.com",
					OrgId:           orgId,
					OrgUfid:         orgUfid,
					CreatedBy:       "test-user",
					CreatedDate:     now,
					LastUpdatedBy:   "test-user",
					LastUpdatedDate: now,
					SyncGroup:       true,
					ExpirationDays:  30,
				},
			},
			want: &GroupModel{
				GroupEmail:      types.StringValue("test-group@gmail.com"),
				OrgId:           types.StringValue(orgId),
				OrgUfid:         types.StringValue(orgUfid),
				CreatedBy:       types.StringValue("test-user"),
				CreatedDate:     timetypes.NewRFC3339TimeValue(now),
				LastUpdatedBy:   types.StringValue("test-user"),
				LastUpdatedDate: timetypes.NewRFC3339TimeValue(now),
				SyncGroup:       types.BoolValue(true),
				ExpirationDays:  types.Int64Value(30),
			},
		},
		{
			name: "Test New GroupModel old minimum",
			group: &user.GroupDescriptionAndRoles{
				GroupAndRoles: user.GroupAndRoles{
					GroupEmail: "test-group@gmail.com",
					GroupName:  "test-group",
				},
			},
			want: &GroupModel{
				GroupEmail: types.StringValue("test-group@gmail.com"),
				GroupName:  types.StringValue("test-group"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewGroupModel(tt.group)
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("NewGroupModel() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func boolPtr(b bool) *bool {
	return &b
}
