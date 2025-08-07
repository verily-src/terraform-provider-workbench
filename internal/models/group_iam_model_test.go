package models

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/verily-src/terraform-provider-workbench/internal/client"
	"github.com/verily-src/terraform-provider-workbench/internal/openapi/user"
)

func TestNewGroupIdentifier(t *testing.T) {
	tests := []struct {
		name      string
		groupName string
		orgID     string
		want      GroupIdentifier
	}{
		{
			name:      "workbench group",
			groupName: "test-group",
			orgID:     "test-org",
			want: GroupIdentifier{
				state:     attr.ValueStateKnown,
				GroupName: types.StringValue("test-group"),
				OrgID:     types.StringValue("test-org"),
			},
		},
		{
			name:      "global group",
			groupName: "global-group",
			orgID:     "",
			want: GroupIdentifier{
				state:     attr.ValueStateKnown,
				GroupName: types.StringValue("global-group"),
				OrgID:     types.StringNull(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewGroupIdentifier(tt.groupName, tt.orgID)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("NewGroupIdentifier() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGroupIdentifierEquals(t *testing.T) {
	tests := []struct {
		name string
		a    GroupIdentifier
		b    GroupIdentifier
		want bool
	}{
		{
			name: "workbench groups",
			a: GroupIdentifier{
				state:     attr.ValueStateKnown,
				GroupName: types.StringValue("group-name"),
				OrgID:     types.StringValue("org-id"),
			},
			b: GroupIdentifier{
				state:     attr.ValueStateKnown,
				GroupName: types.StringValue("group-name"),
				OrgID:     types.StringValue("org-id"),
			},
			want: true,
		},
		{
			name: "global groups",
			a: GroupIdentifier{
				state:     attr.ValueStateKnown,
				GroupName: types.StringValue("group-name"),
				OrgID:     types.StringNull(),
			},
			b: GroupIdentifier{
				state:     attr.ValueStateKnown,
				GroupName: types.StringValue("group-name"),
				OrgID:     types.StringNull(),
			},
			want: true,
		},
		{
			name: "different group name",
			a: GroupIdentifier{
				state:     attr.ValueStateKnown,
				GroupName: types.StringValue("group-name"),
				OrgID:     types.StringValue("org-id"),
			},
			b: GroupIdentifier{
				GroupName: types.StringValue("different-group-name"),
				OrgID:     types.StringValue("org-id"),
			},
			want: false,
		},
		{
			name: "different org ID",
			a: GroupIdentifier{
				state:     attr.ValueStateKnown,
				GroupName: types.StringValue("group-name"),
				OrgID:     types.StringValue("org-id"),
			},
			b: GroupIdentifier{
				state:     attr.ValueStateKnown,
				GroupName: types.StringValue("group-name"),
				OrgID:     types.StringValue("different-org-id"),
			},
			want: false,
		},
		{
			name: "different state",
			a: GroupIdentifier{
				state:     attr.ValueStateUnknown,
				GroupName: types.StringValue("group-name"),
				OrgID:     types.StringValue("org-id"),
			},
			b: GroupIdentifier{
				state:     attr.ValueStateKnown,
				GroupName: types.StringValue("group-name"),
				OrgID:     types.StringValue("org-id"),
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.a.Equal(tt.b)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("Equal() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGroupPrincipalConvert(t *testing.T) {
	tests := []struct {
		name        string
		umPrincipal user.Principal
		principal   GroupPrincipal
		skipToTf    bool
		skipToUm    bool
	}{
		{
			name: "user",
			umPrincipal: user.Principal{
				UserPrincipal: &user.PrincipalUser{
					Email: "test-user",
				},
			},
			principal: GroupPrincipal{
				User: types.StringValue("test-user"),
			},
		},
		{
			name: "group",
			umPrincipal: user.Principal{
				GroupPrincipal: &user.PrincipalWorkbenchGroup{
					GroupName:      "test-group",
					OrganizationId: "test-org",
				},
			},
			principal: GroupPrincipal{
				Group: &GroupIdentifier{
					state:     attr.ValueStateKnown,
					GroupName: types.StringValue("test-group"),
					OrgID:     types.StringValue("test-org"),
				},
			},
		},
		{
			name: "global legacy group",
			umPrincipal: user.Principal{
				GlobalGroupPrincipal: &user.PrincipalGlobalGroup{
					GroupName: "global-group",
				},
			},
			principal: GroupPrincipal{
				Group: &GroupIdentifier{
					state:     attr.ValueStateKnown,
					GroupName: types.StringValue("global-group"),
					OrgID:     types.StringNull(),
				},
			},
			// Don't test conversion from terrafom to user manager global group,
			// since we will always convert to a WorkbenchGroup and never a
			// GlobalGroup (verified with the test below)
			skipToUm: true,
		},
		{
			name: "global group",
			umPrincipal: user.Principal{
				GroupPrincipal: &user.PrincipalWorkbenchGroup{
					GroupName:      "global-group",
					OrganizationId: "",
				},
			},
			principal: GroupPrincipal{
				Group: &GroupIdentifier{
					state:     attr.ValueStateKnown,
					GroupName: types.StringValue("global-group"),
					OrgID:     types.StringNull(),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.skipToTf {
				got := ConvertGroupPrincipalToTf(tt.umPrincipal)
				if diff := cmp.Diff(tt.principal, got); diff != "" {
					t.Errorf("ConvertGroupPrincipalToTf() mismatch (-want +got):\n%s", diff)
				}
			}

			if !tt.skipToUm {
				gotUm := convertGroupPrincipalToUm(tt.principal)
				if diff := cmp.Diff(tt.umPrincipal, gotUm); diff != "" {
					t.Errorf("convertGroupPrincipalToUm() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestNewGroupIamBindingModel(t *testing.T) {
	tests := []struct {
		name    string
		members *user.GroupMemberList
		role    user.GroupRole
		want    *GroupIamBindingModel
	}{
		{
			name: "succeeds",
			members: &user.GroupMemberList{
				{
					Principal: user.Principal{
						UserPrincipal: &user.PrincipalUser{
							Email: "test-user1",
						},
					},
					Roles: []user.GroupRole{
						user.GroupRoleADMIN,
						user.GroupRoleMEMBER,
						user.GroupRoleREADER,
					},
				},
				{
					Principal: user.Principal{
						UserPrincipal: &user.PrincipalUser{
							Email: "test-user2",
						},
					},
					Roles: []user.GroupRole{
						user.GroupRoleMEMBER,
						user.GroupRoleREADER,
					},
				},
				{
					Principal: user.Principal{
						GroupPrincipal: &user.PrincipalWorkbenchGroup{
							GroupName:      "test-group",
							OrganizationId: "test-org",
						},
					},
					Roles: []user.GroupRole{
						user.GroupRoleADMIN,
						user.GroupRoleMEMBER,
					},
				},
				{
					Principal: user.Principal{
						PublicPrincipal: client.Ptr(true),
					},
					Roles: []user.GroupRole{
						user.GroupRoleREADER,
					},
				},
			},
			role: user.GroupRoleMEMBER,
			want: &GroupIamBindingModel{
				GroupIdentifier: NewGroupIdentifier("test-group", "test-org"),
				GroupRoleBinding: buildGroupBinding(user.GroupRoleMEMBER,
					groupUserPrincipal("test-user1"),
					groupUserPrincipal("test-user2"),
					groupGroupPrincipal("test-group", "test-org"),
				),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewGroupIamBindingModel("test-group", "test-org", tt.members, tt.role)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("NewGroupIamBindingModel() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDiffGroupIamBindings(t *testing.T) {
	tests := []struct {
		name        string
		oldBinding  *GroupIamBindingModel
		newBinding  *GroupIamBindingModel
		wantDeleted *GroupIamBindingModel
		wantAdded   *GroupIamBindingModel
	}{
		{
			name: "no change",
			oldBinding: &GroupIamBindingModel{
				GroupRoleBinding: buildGroupBinding(user.GroupRoleMEMBER,
					groupUserPrincipal("test-user1"),
				),
			},
			newBinding: &GroupIamBindingModel{
				GroupRoleBinding: buildGroupBinding(user.GroupRoleMEMBER,
					groupUserPrincipal("test-user1"),
				),
			},
			wantDeleted: nil,
			wantAdded:   nil,
		},
		{
			name: "new members",
			oldBinding: &GroupIamBindingModel{
				GroupRoleBinding: buildGroupBinding(user.GroupRoleMEMBER,
					groupUserPrincipal("test-user1"),
				),
			},
			newBinding: &GroupIamBindingModel{
				GroupRoleBinding: buildGroupBinding(user.GroupRoleMEMBER,
					groupUserPrincipal("test-user1"),
					groupUserPrincipal("new-user1"),
					groupUserPrincipal("new-user2"),
				),
			},
			wantDeleted: nil,
			wantAdded: &GroupIamBindingModel{
				GroupRoleBinding: buildGroupBinding(user.GroupRoleMEMBER,
					groupUserPrincipal("new-user1"),
					groupUserPrincipal("new-user2"),
				),
			},
		},
		{
			name: "deleted members",
			oldBinding: &GroupIamBindingModel{
				GroupRoleBinding: buildGroupBinding(user.GroupRoleMEMBER,
					groupUserPrincipal("test-user1"),
					groupUserPrincipal("old-user1"),
					groupUserPrincipal("old-user2"),
				),
			},
			newBinding: &GroupIamBindingModel{
				GroupRoleBinding: buildGroupBinding(user.GroupRoleMEMBER,
					groupUserPrincipal("test-user1"),
				),
			},
			wantDeleted: &GroupIamBindingModel{
				GroupRoleBinding: buildGroupBinding(user.GroupRoleMEMBER,
					groupUserPrincipal("old-user1"),
					groupUserPrincipal("old-user2"),
				),
			},
			wantAdded: nil,
		},
		{
			name: "new and deleted members",
			oldBinding: &GroupIamBindingModel{
				GroupRoleBinding: buildGroupBinding(user.GroupRoleMEMBER,
					groupUserPrincipal("test-user1"),
					groupUserPrincipal("old-user1"),
				),
			},
			newBinding: &GroupIamBindingModel{
				GroupRoleBinding: buildGroupBinding(user.GroupRoleMEMBER,
					groupUserPrincipal("test-user1"),
					groupUserPrincipal("new-user1"),
				),
			},
			wantDeleted: &GroupIamBindingModel{
				GroupRoleBinding: buildGroupBinding(user.GroupRoleMEMBER,
					groupUserPrincipal("old-user1"),
				),
			},
			wantAdded: &GroupIamBindingModel{
				GroupRoleBinding: buildGroupBinding(user.GroupRoleMEMBER,
					groupUserPrincipal("new-user1"),
				),
			},
		},
		{
			name: "different role",
			oldBinding: &GroupIamBindingModel{
				GroupRoleBinding: buildGroupBinding(user.GroupRoleMEMBER,
					groupUserPrincipal("test-user1"),
				),
			},
			newBinding: &GroupIamBindingModel{
				GroupRoleBinding: buildGroupBinding(user.GroupRoleREADER,
					groupUserPrincipal("test-user1"),
				),
			},
			wantDeleted: &GroupIamBindingModel{
				GroupRoleBinding: buildGroupBinding(user.GroupRoleMEMBER,
					groupUserPrincipal("test-user1"),
				),
			},
			wantAdded: &GroupIamBindingModel{
				GroupRoleBinding: buildGroupBinding(user.GroupRoleREADER,
					groupUserPrincipal("test-user1"),
				),
			},
		},
		{
			name: "different group",
			oldBinding: &GroupIamBindingModel{
				GroupIdentifier: NewGroupIdentifier("test-group-1", "test-org"),
				GroupRoleBinding: buildGroupBinding(user.GroupRoleMEMBER,
					groupUserPrincipal("test-user1"),
				),
			},
			newBinding: &GroupIamBindingModel{
				GroupIdentifier: NewGroupIdentifier("test-group-2", "test-org"),
				GroupRoleBinding: buildGroupBinding(user.GroupRoleREADER,
					groupUserPrincipal("test-user1"),
				),
			},
			wantDeleted: &GroupIamBindingModel{
				GroupIdentifier: NewGroupIdentifier("test-group-1", "test-org"),
				GroupRoleBinding: buildGroupBinding(user.GroupRoleMEMBER,
					groupUserPrincipal("test-user1"),
				),
			},
			wantAdded: &GroupIamBindingModel{
				GroupIdentifier: NewGroupIdentifier("test-group-2", "test-org"),
				GroupRoleBinding: buildGroupBinding(user.GroupRoleREADER,
					groupUserPrincipal("test-user1"),
				),
			},
		},
		{
			name: "nil members",
			oldBinding: &GroupIamBindingModel{
				GroupRoleBinding: GroupRoleBinding{
					Role:       types.StringValue(string(user.GroupRoleMEMBER)),
					Principals: nil,
				},
			},
			newBinding: &GroupIamBindingModel{
				GroupRoleBinding: buildGroupBinding(user.GroupRoleMEMBER,
					groupUserPrincipal("test-user1"),
				),
			},
			wantDeleted: nil,
			wantAdded: &GroupIamBindingModel{
				GroupRoleBinding: buildGroupBinding(user.GroupRoleMEMBER,
					groupUserPrincipal("test-user1"),
				),
			},
		},
		{
			name: "empty members",
			oldBinding: &GroupIamBindingModel{
				GroupRoleBinding: GroupRoleBinding{
					Role:       types.StringValue(string(user.GroupRoleMEMBER)),
					Principals: []GroupPrincipal{},
				},
			},
			newBinding: &GroupIamBindingModel{
				GroupRoleBinding: buildGroupBinding(user.GroupRoleMEMBER,
					groupUserPrincipal("test-user1"),
				),
			},
			wantDeleted: nil,
			wantAdded: &GroupIamBindingModel{
				GroupRoleBinding: buildGroupBinding(user.GroupRoleMEMBER,
					groupUserPrincipal("test-user1"),
				),
			},
		},
		{
			name:       "from nil",
			oldBinding: nil,
			newBinding: &GroupIamBindingModel{
				GroupRoleBinding: buildGroupBinding(user.GroupRoleMEMBER,
					groupUserPrincipal("test-user1"),
				),
			},
			wantDeleted: nil,
			wantAdded: &GroupIamBindingModel{
				GroupRoleBinding: buildGroupBinding(user.GroupRoleMEMBER,
					groupUserPrincipal("test-user1"),
				),
			},
		},
		{
			name: "to nil",
			oldBinding: &GroupIamBindingModel{
				GroupRoleBinding: buildGroupBinding(user.GroupRoleMEMBER,
					groupUserPrincipal("test-user1"),
				),
			},
			newBinding: nil,
			wantDeleted: &GroupIamBindingModel{
				GroupRoleBinding: buildGroupBinding(user.GroupRoleMEMBER,
					groupUserPrincipal("test-user1"),
				),
			},
			wantAdded: nil,
		},
		{
			name:        "both nil",
			oldBinding:  nil,
			newBinding:  nil,
			wantDeleted: nil,
			wantAdded:   nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDeleted, gotAdded := DiffGroupIamBindings(tt.oldBinding, tt.newBinding)
			if diff := cmp.Diff(tt.wantDeleted, gotDeleted); diff != "" {
				t.Errorf("DiffGroupIamBindings() deleted mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantAdded, gotAdded); diff != "" {
				t.Errorf("DiffGroupIamBindings() added mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewGroupIamPolicyModelIams(t *testing.T) {
	tests := []struct {
		name    string
		members user.GroupMemberList
		want    *[]GroupRoleBinding
	}{
		{
			name: "succeeds",
			members: user.GroupMemberList{
				{
					Principal: user.Principal{
						UserPrincipal: &user.PrincipalUser{
							Email: "test-user1",
						},
					},
					Roles: []user.GroupRole{
						user.GroupRoleADMIN,
						user.GroupRoleMEMBER,
						user.GroupRoleREADER,
					},
				},
				{
					Principal: user.Principal{
						UserPrincipal: &user.PrincipalUser{
							Email: "test-user2",
						},
					},
					Roles: []user.GroupRole{
						user.GroupRoleMEMBER,
						user.GroupRoleREADER,
					},
				},
				{
					Principal: user.Principal{
						GroupPrincipal: &user.PrincipalWorkbenchGroup{
							GroupName:      "test-group",
							OrganizationId: "test-org",
						},
					},
					Roles: []user.GroupRole{
						user.GroupRoleADMIN,
						user.GroupRoleMEMBER,
					},
				},
				{
					Principal: user.Principal{
						PublicPrincipal: client.Ptr(true),
					},
					Roles: []user.GroupRole{
						user.GroupRoleREADER,
					},
				},
			},
			want: &[]GroupRoleBinding{
				buildGroupBinding(user.GroupRoleADMIN,
					groupUserPrincipal("test-user1"),
					groupGroupPrincipal("test-group", "test-org"),
				),
				buildGroupBinding(user.GroupRoleMEMBER,
					groupUserPrincipal("test-user1"),
					groupUserPrincipal("test-user2"),
					groupGroupPrincipal("test-group", "test-org"),
				),
				buildGroupBinding(user.GroupRoleREADER,
					groupUserPrincipal("test-user1"),
					groupUserPrincipal("test-user2"),
					groupPublicPrincipal(),
				),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewGroupIamPolicyModel("test-group", "test-org", client.Ptr(tt.members)).Iams
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("NewGroupIamPolicyModel() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGroupIamPolicyBuildGrantRequests(t *testing.T) {
	tests := []struct {
		name   string
		policy GroupIamPolicyModel
		want   []user.SetAccessRequest
	}{
		{
			name: "succeeds",
			policy: GroupIamPolicyModel{
				GroupIdentifier: NewGroupIdentifier("test-group", "test-org"),
				Iams: &[]GroupRoleBinding{
					buildGroupBinding(user.GroupRoleADMIN,
						groupUserPrincipal("test-user1"),
					),
					buildGroupBinding(user.GroupRoleMEMBER,
						groupUserPrincipal("test-user1"),
						groupUserPrincipal("test-user2"),
						groupGroupPrincipal("test-group", "test-org"),
					),
					buildGroupBinding(user.GroupRoleREADER,
						groupGlobalGroupPrincipal("test-global-group"),
						groupPublicPrincipal(),
					),
				},
			},
			want: []user.SetAccessRequest{
				buildSetAccessRequest(user.GRANT,
					user.GroupRoleADMIN,
					convertGroupPrincipalToUm(groupUserPrincipal("test-user1"))),
				buildSetAccessRequest(user.GRANT,
					user.GroupRoleMEMBER,
					convertGroupPrincipalToUm(groupUserPrincipal("test-user1"))),
				buildSetAccessRequest(user.GRANT,
					user.GroupRoleMEMBER,
					convertGroupPrincipalToUm(groupUserPrincipal("test-user2"))),
				buildSetAccessRequest(user.GRANT,
					user.GroupRoleMEMBER,
					convertGroupPrincipalToUm(groupGroupPrincipal("test-group", "test-org"))),
				buildSetAccessRequest(user.GRANT,
					user.GroupRoleREADER,
					convertGroupPrincipalToUm(groupGlobalGroupPrincipal("test-global-group"))),
				buildSetAccessRequest(user.GRANT,
					user.GroupRoleREADER,
					convertGroupPrincipalToUm(groupPublicPrincipal())),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.policy.BuildGrantRequests()
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("BuildGrantRequests() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDiffGroupIamPolicies(t *testing.T) {
	empty := []GroupRoleBinding{}
	tests := []struct {
		name        string
		oldPolicy   *GroupIamPolicyModel
		newPolicy   *GroupIamPolicyModel
		wantDeleted []GroupRoleBinding
		wantAdded   []GroupRoleBinding
	}{
		{
			name: "no change",
			oldPolicy: &GroupIamPolicyModel{
				Iams: &[]GroupRoleBinding{
					buildGroupBinding(user.GroupRoleADMIN,
						groupUserPrincipal("test-user1")),
					buildGroupBinding(user.GroupRoleMEMBER,
						groupUserPrincipal("test-user1"),
						groupUserPrincipal("test-user2")),
				},
			},
			newPolicy: &GroupIamPolicyModel{
				Iams: &[]GroupRoleBinding{
					buildGroupBinding(user.GroupRoleADMIN,
						groupUserPrincipal("test-user1")),
					buildGroupBinding(user.GroupRoleMEMBER,
						groupUserPrincipal("test-user1"),
						groupUserPrincipal("test-user2")),
				},
			},
			wantDeleted: empty,
			wantAdded:   empty,
		},
		{
			name: "reordered but equivalent",
			oldPolicy: &GroupIamPolicyModel{
				Iams: &[]GroupRoleBinding{
					buildGroupBinding(user.GroupRoleADMIN,
						groupUserPrincipal("test-user1")),
					buildGroupBinding(user.GroupRoleMEMBER,
						groupUserPrincipal("test-user1"),
						groupUserPrincipal("test-user2")),
				},
			},
			newPolicy: &GroupIamPolicyModel{
				Iams: &[]GroupRoleBinding{
					buildGroupBinding(user.GroupRoleMEMBER,
						groupUserPrincipal("test-user2"),
						groupUserPrincipal("test-user1")),
					buildGroupBinding(user.GroupRoleADMIN,
						groupUserPrincipal("test-user1")),
				},
			},
			wantDeleted: empty,
			wantAdded:   empty,
		},
		{
			name: "changed iams",
			oldPolicy: &GroupIamPolicyModel{
				Iams: &[]GroupRoleBinding{
					buildGroupBinding(user.GroupRoleADMIN,
						groupUserPrincipal("test-user1")),
					buildGroupBinding(user.GroupRoleMEMBER,
						groupUserPrincipal("test-user1"),
						groupUserPrincipal("test-user2")),
				},
			},
			newPolicy: &GroupIamPolicyModel{
				Iams: &[]GroupRoleBinding{
					buildGroupBinding(user.GroupRoleADMIN,
						groupUserPrincipal("test-user3")),
					buildGroupBinding(user.GroupRoleREADER,
						groupUserPrincipal("test-user1"),
						groupUserPrincipal("test-user2")),
				},
			},
			wantDeleted: []GroupRoleBinding{
				buildGroupBinding(user.GroupRoleMEMBER,
					groupUserPrincipal("test-user1"),
					groupUserPrincipal("test-user2")),
				buildGroupBinding(user.GroupRoleADMIN,
					groupUserPrincipal("test-user1")),
			},
			wantAdded: []GroupRoleBinding{
				buildGroupBinding(user.GroupRoleREADER,
					groupUserPrincipal("test-user1"),
					groupUserPrincipal("test-user2")),
				buildGroupBinding(user.GroupRoleADMIN,
					groupUserPrincipal("test-user3")),
			},
		},
		{
			name: "nil iams",
			oldPolicy: &GroupIamPolicyModel{
				Iams: nil,
			},
			newPolicy: &GroupIamPolicyModel{
				Iams: &[]GroupRoleBinding{
					buildGroupBinding(user.GroupRoleADMIN,
						groupUserPrincipal(" test-user1")),
				},
			},
			wantDeleted: empty,
			wantAdded: []GroupRoleBinding{
				buildGroupBinding(user.GroupRoleADMIN,
					groupUserPrincipal(" test-user1")),
			},
		},
		{
			name: "empty iams",
			oldPolicy: &GroupIamPolicyModel{
				Iams: &[]GroupRoleBinding{},
			},
			newPolicy: &GroupIamPolicyModel{
				Iams: &[]GroupRoleBinding{
					buildGroupBinding(user.GroupRoleADMIN,
						groupUserPrincipal(" test-user1")),
				},
			},
			wantDeleted: empty,
			wantAdded: []GroupRoleBinding{
				buildGroupBinding(user.GroupRoleADMIN,
					groupUserPrincipal(" test-user1")),
			},
		},
		{
			name:      "from nil",
			oldPolicy: nil,
			newPolicy: &GroupIamPolicyModel{
				Iams: &[]GroupRoleBinding{
					buildGroupBinding(user.GroupRoleADMIN,
						groupUserPrincipal(" test-user1")),
				},
			},
			wantDeleted: empty,
			wantAdded: []GroupRoleBinding{
				buildGroupBinding(user.GroupRoleADMIN,
					groupUserPrincipal(" test-user1")),
			},
		},
		{
			name: "to nil",
			oldPolicy: &GroupIamPolicyModel{
				Iams: &[]GroupRoleBinding{
					buildGroupBinding(user.GroupRoleADMIN,
						groupUserPrincipal(" test-user1")),
				},
			},
			newPolicy: nil,
			wantDeleted: []GroupRoleBinding{
				buildGroupBinding(user.GroupRoleADMIN,
					groupUserPrincipal(" test-user1")),
			},
			wantAdded: empty,
		},
		{
			name:        "both nil",
			oldPolicy:   nil,
			newPolicy:   nil,
			wantDeleted: empty,
			wantAdded:   empty,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDeleted, gotAdded := DiffGroupIamPolicies(tt.oldPolicy, tt.newPolicy)
			if diff := cmp.Diff(tt.wantDeleted, gotDeleted); diff != "" {
				t.Errorf("DiffGroupIamPolicies() deleted mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantAdded, gotAdded); diff != "" {
				t.Errorf("DiffGroupIamPolicies() added mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPrincipalMatchesTf(t *testing.T) {
	tests := []struct {
		name        string
		principal   user.Principal
		tfPrincipal GroupPrincipal
		want        bool
	}{
		{
			name: "user principal matches",
			principal: user.Principal{
				UserPrincipal: &user.PrincipalUser{
					Email: "test-user",
				},
			},
			tfPrincipal: groupUserPrincipal("test-user"),
			want:        true,
		},
		{
			name: "user principal email does not match",
			principal: user.Principal{
				UserPrincipal: &user.PrincipalUser{
					Email: "test-user",
				},
			},
			tfPrincipal: groupUserPrincipal("test-user-2"),
			want:        false,
		},
		{
			name: "group principal matches",
			principal: user.Principal{
				GroupPrincipal: &user.PrincipalWorkbenchGroup{
					GroupName:      "test-group",
					OrganizationId: "test-org",
				},
			},
			tfPrincipal: groupGroupPrincipal("test-group", "test-org"),
			want:        true,
		},
		{
			name: "group principal group name does not match",
			principal: user.Principal{
				GroupPrincipal: &user.PrincipalWorkbenchGroup{
					GroupName:      "test-group",
					OrganizationId: "test-org",
				},
			},
			tfPrincipal: groupGroupPrincipal("test-group-2", "test-org"),
			want:        false,
		},
		{
			name: "group principal different org does not match",
			principal: user.Principal{
				GroupPrincipal: &user.PrincipalWorkbenchGroup{
					GroupName:      "test-group",
					OrganizationId: "test-org",
				},
			},
			tfPrincipal: groupGroupPrincipal("test-group", "test-org-2"),
			want:        false,
		},
		{
			name: "global group principal matches",
			principal: user.Principal{
				GlobalGroupPrincipal: &user.PrincipalGlobalGroup{
					GroupName: "global-group",
				},
			},
			tfPrincipal: groupGlobalGroupPrincipal("global-group"),
			want:        true,
		},
		{
			name: "global group principal not match non-global group",
			principal: user.Principal{
				GlobalGroupPrincipal: &user.PrincipalGlobalGroup{
					GroupName: "global-group",
				},
			},
			tfPrincipal: groupGroupPrincipal("test-group", "test-org"),
			want:        false,
		},
		{
			name: "public principal matches",
			principal: user.Principal{
				PublicPrincipal: client.Ptr(true),
			},
			tfPrincipal: groupPublicPrincipal(),
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PrincipalMatchesTf(tt.principal, tt.tfPrincipal)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("PrincipalMatchesTf() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func buildSetAccessRequest(op user.SetAccessOperation, role user.GroupRole, principal user.Principal) user.SetAccessRequest {
	return user.SetAccessRequest{
		Operation: op,
		Role:      role,
		Principal: principal,
	}
}

func buildGroupBinding(role user.GroupRole, principals ...GroupPrincipal) GroupRoleBinding {
	return GroupRoleBinding{
		Role:       types.StringValue(string(role)),
		Principals: principals,
	}
}

func groupUserPrincipal(email string) GroupPrincipal {
	return GroupPrincipal{
		User: types.StringValue(email),
	}
}

func groupGroupPrincipal(groupName, orgID string) GroupPrincipal {
	return GroupPrincipal{
		Group: &GroupIdentifier{
			state:     attr.ValueStateKnown,
			GroupName: types.StringValue(groupName),
			OrgID:     types.StringValue(orgID),
		},
	}
}

func groupGlobalGroupPrincipal(groupName string) GroupPrincipal {
	return GroupPrincipal{
		Group: &GroupIdentifier{
			state:     attr.ValueStateKnown,
			GroupName: types.StringValue(groupName),
			OrgID:     types.StringNull(),
		},
	}
}

func groupPublicPrincipal() GroupPrincipal {
	return GroupPrincipal{
		Public: types.BoolValue(true),
	}
}
