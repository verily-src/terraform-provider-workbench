package models

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/verily-src/terraform-provider-workbench/internal/openapi/wsm"
)

func TestWorkspaceIamMemberBuildGrantRequest(t *testing.T) {
	id := uuid.New()
	tests := []struct {
		name      string
		iamMember workspaceIamMemberModel
		wantWsID  string
		wantReq   wsm.SetAccessRequest
	}{
		{
			name: "succeeds",
			iamMember: workspaceIamMemberModel{
				WorkspaceID: types.StringValue(id.String()),
				Role:        types.StringValue(string(wsm.IamRoleREADER)),
				Member:      types.StringValue("foo@bar.com"),
			},
			wantWsID: id.String(),
			wantReq: wsm.SetAccessRequest{
				Operation: wsm.GRANT,
				Principal: wsm.Principal{
					UserPrincipal: &wsm.PrincipalUser{
						Email: "foo@bar.com",
					},
				},
				Role: wsm.IamRoleREADER,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotWsID, gotReq := tt.iamMember.BuildGrantRequest()
			if diff := cmp.Diff(tt.wantWsID, gotWsID); diff != "" {
				t.Errorf("BuildGrantRequest() workspace ID mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantReq, gotReq); diff != "" {
				t.Errorf("BuildGrantRequest() request mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestWorkspaceIamMemberBuildRevokeRequest(t *testing.T) {
	id := uuid.New()
	tests := []struct {
		name      string
		iamMember workspaceIamMemberModel
		wantWsID  string
		wantReq   wsm.SetAccessRequest
	}{
		{
			name: "succeeds",
			iamMember: workspaceIamMemberModel{
				WorkspaceID: types.StringValue(id.String()),
				Role:        types.StringValue(string(wsm.IamRoleWRITER)),
				Member:      types.StringValue("foo@bar.com"),
			},
			wantWsID: id.String(),
			wantReq: wsm.SetAccessRequest{
				Operation: wsm.REVOKE,
				Principal: wsm.Principal{
					UserPrincipal: &wsm.PrincipalUser{
						Email: "foo@bar.com",
					},
				},
				Role: wsm.IamRoleWRITER,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotWsID, gotReq := tt.iamMember.BuildRevokeRequest()
			if diff := cmp.Diff(tt.wantWsID, gotWsID); diff != "" {
				t.Errorf("BuildRevokeRequest() workspace ID mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantReq, gotReq); diff != "" {
				t.Errorf("BuildRevokeRequest() request mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewWorkspaceIamBindingModel(t *testing.T) {
	id := uuid.New()
	tests := []struct {
		name        string
		workspaceID string
		roleBinding *wsm.RoleBinding
		want        *WorkspaceIamBindingModel
	}{
		{
			name:        "succeeds",
			workspaceID: id.String(),
			roleBinding: &wsm.RoleBinding{
				Role:    wsm.IamRoleOWNER,
				Members: &[]string{"user1@bar.com", "user2@bar.com"},
			},
			want: &WorkspaceIamBindingModel{
				WorkspaceID: types.StringValue(id.String()),
				WorkspaceRoleBinding: WorkspaceRoleBinding{
					Role: types.StringValue(string(wsm.IamRoleOWNER)),
					Members: &[]types.String{
						types.StringValue("user1@bar.com"),
						types.StringValue("user2@bar.com"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewWorkspaceIamBindingModel(tt.workspaceID, tt.roleBinding)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("NewWorkspaceIamBindingModel() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDiffWorkspaceIamBindings(t *testing.T) {
	id := types.StringValue(uuid.New().String())
	id2 := types.StringValue(uuid.New().String())
	role := types.StringValue(string(wsm.IamRoleOWNER))
	role2 := types.StringValue(string(wsm.IamRoleREADER))
	tests := []struct {
		name        string
		oldBindings *WorkspaceIamBindingModel
		newBindings *WorkspaceIamBindingModel
		wantDeleted *WorkspaceIamBindingModel
		wantAdded   *WorkspaceIamBindingModel
	}{
		{
			name: "no change",
			oldBindings: &WorkspaceIamBindingModel{
				WorkspaceID: id,
				WorkspaceRoleBinding: WorkspaceRoleBinding{
					Role:    role,
					Members: &[]types.String{types.StringValue("foo@bar.com")},
				},
			},
			newBindings: &WorkspaceIamBindingModel{
				WorkspaceID: id,
				WorkspaceRoleBinding: WorkspaceRoleBinding{
					Role:    role,
					Members: &[]types.String{types.StringValue("foo@bar.com")},
				},
			},
			wantDeleted: nil,
			wantAdded:   nil,
		},
		{
			name: "new members",
			oldBindings: &WorkspaceIamBindingModel{
				WorkspaceID: id,
				WorkspaceRoleBinding: WorkspaceRoleBinding{
					Role:    role,
					Members: &[]types.String{types.StringValue("foo@bar.com")},
				},
			},
			newBindings: &WorkspaceIamBindingModel{
				WorkspaceID: id,
				WorkspaceRoleBinding: WorkspaceRoleBinding{
					Role: role,
					Members: &[]types.String{
						types.StringValue("foo@bar.com"),
						types.StringValue("newuser1@bar.com"),
						types.StringValue("newuser2@bar.com"),
					},
				},
			},
			wantDeleted: nil,
			wantAdded: &WorkspaceIamBindingModel{
				WorkspaceID: id,
				WorkspaceRoleBinding: WorkspaceRoleBinding{
					Role: role,
					Members: &[]types.String{
						types.StringValue("newuser1@bar.com"),
						types.StringValue("newuser2@bar.com"),
					},
				},
			},
		},
		{
			name: "deleted members",
			oldBindings: &WorkspaceIamBindingModel{
				WorkspaceID: id,
				WorkspaceRoleBinding: WorkspaceRoleBinding{
					Role: role,
					Members: &[]types.String{
						types.StringValue("foo@bar.com"),
						types.StringValue("olduser1@bar.com"),
						types.StringValue("olduser2@bar.com"),
					},
				},
			},
			newBindings: &WorkspaceIamBindingModel{
				WorkspaceID: id,
				WorkspaceRoleBinding: WorkspaceRoleBinding{
					Role:    role,
					Members: &[]types.String{types.StringValue("foo@bar.com")},
				},
			},
			wantDeleted: &WorkspaceIamBindingModel{
				WorkspaceID: id,
				WorkspaceRoleBinding: WorkspaceRoleBinding{
					Role: role,
					Members: &[]types.String{
						types.StringValue("olduser1@bar.com"),
						types.StringValue("olduser2@bar.com"),
					},
				},
			},
			wantAdded: nil,
		},
		{
			name: "new and deleted members",
			oldBindings: &WorkspaceIamBindingModel{
				WorkspaceID: id,
				WorkspaceRoleBinding: WorkspaceRoleBinding{
					Role: role,
					Members: &[]types.String{
						types.StringValue("foo@bar.com"),
						types.StringValue("olduser@bar.com"),
					},
				},
			},
			newBindings: &WorkspaceIamBindingModel{
				WorkspaceID: id,
				WorkspaceRoleBinding: WorkspaceRoleBinding{
					Role: role,
					Members: &[]types.String{
						types.StringValue("foo@bar.com"),
						types.StringValue("newuser@bar.com"),
					},
				},
			},
			wantDeleted: &WorkspaceIamBindingModel{
				WorkspaceID: id,
				WorkspaceRoleBinding: WorkspaceRoleBinding{
					Role: role,
					Members: &[]types.String{
						types.StringValue("olduser@bar.com"),
					},
				},
			},
			wantAdded: &WorkspaceIamBindingModel{
				WorkspaceID: id,
				WorkspaceRoleBinding: WorkspaceRoleBinding{
					Role: role,
					Members: &[]types.String{
						types.StringValue("newuser@bar.com"),
					},
				},
			},
		},
		{
			name: "different role",
			oldBindings: &WorkspaceIamBindingModel{
				WorkspaceID: id,
				WorkspaceRoleBinding: WorkspaceRoleBinding{
					Role: role,
					Members: &[]types.String{
						types.StringValue("foo@bar.com"),
					},
				},
			},
			newBindings: &WorkspaceIamBindingModel{
				WorkspaceID: id,
				WorkspaceRoleBinding: WorkspaceRoleBinding{
					Role: role2,
					Members: &[]types.String{
						types.StringValue("foo@bar.com"),
					},
				},
			},
			wantDeleted: &WorkspaceIamBindingModel{
				WorkspaceID: id,
				WorkspaceRoleBinding: WorkspaceRoleBinding{
					Role: role,
					Members: &[]types.String{
						types.StringValue("foo@bar.com"),
					},
				},
			},
			wantAdded: &WorkspaceIamBindingModel{
				WorkspaceID: id,
				WorkspaceRoleBinding: WorkspaceRoleBinding{
					Role: role2,
					Members: &[]types.String{
						types.StringValue("foo@bar.com"),
					},
				},
			},
		},
		{
			name: "different workspace",
			oldBindings: &WorkspaceIamBindingModel{
				WorkspaceID: id,
				WorkspaceRoleBinding: WorkspaceRoleBinding{
					Role: role,
					Members: &[]types.String{
						types.StringValue("foo@bar.com"),
					},
				},
			},
			newBindings: &WorkspaceIamBindingModel{
				WorkspaceID: id2,
				WorkspaceRoleBinding: WorkspaceRoleBinding{
					Role: role,
					Members: &[]types.String{
						types.StringValue("foo@bar.com"),
					},
				},
			},
			wantDeleted: &WorkspaceIamBindingModel{
				WorkspaceID: id,
				WorkspaceRoleBinding: WorkspaceRoleBinding{
					Role: role,
					Members: &[]types.String{
						types.StringValue("foo@bar.com"),
					},
				},
			},
			wantAdded: &WorkspaceIamBindingModel{
				WorkspaceID: id2,
				WorkspaceRoleBinding: WorkspaceRoleBinding{
					Role: role,
					Members: &[]types.String{
						types.StringValue("foo@bar.com"),
					},
				},
			},
		},
		{
			name: "nil members",
			oldBindings: &WorkspaceIamBindingModel{
				WorkspaceID: id,
				WorkspaceRoleBinding: WorkspaceRoleBinding{
					Role:    role,
					Members: nil,
				},
			},
			newBindings: &WorkspaceIamBindingModel{
				WorkspaceID: id,
				WorkspaceRoleBinding: WorkspaceRoleBinding{
					Role: role,
					Members: &[]types.String{
						types.StringValue("foo@bar.com"),
					},
				},
			},
			wantDeleted: nil,
			wantAdded: &WorkspaceIamBindingModel{
				WorkspaceID: id,
				WorkspaceRoleBinding: WorkspaceRoleBinding{
					Role: role,
					Members: &[]types.String{
						types.StringValue("foo@bar.com"),
					},
				},
			},
		},
		{
			name: "empty members",
			oldBindings: &WorkspaceIamBindingModel{
				WorkspaceID: id,
				WorkspaceRoleBinding: WorkspaceRoleBinding{
					Role:    role,
					Members: &[]types.String{},
				},
			},
			newBindings: &WorkspaceIamBindingModel{
				WorkspaceID: id,
				WorkspaceRoleBinding: WorkspaceRoleBinding{
					Role: role,
					Members: &[]types.String{
						types.StringValue("foo@bar.com"),
					},
				},
			},
			wantDeleted: nil,
			wantAdded: &WorkspaceIamBindingModel{
				WorkspaceID: id,
				WorkspaceRoleBinding: WorkspaceRoleBinding{
					Role: role,
					Members: &[]types.String{
						types.StringValue("foo@bar.com"),
					},
				},
			},
		},
		{
			name:        "from nil",
			oldBindings: nil,
			newBindings: &WorkspaceIamBindingModel{
				WorkspaceID: id,
				WorkspaceRoleBinding: WorkspaceRoleBinding{
					Role: role,
					Members: &[]types.String{
						types.StringValue("foo@bar.com"),
					},
				},
			},
			wantDeleted: nil,
			wantAdded: &WorkspaceIamBindingModel{
				WorkspaceID: id,
				WorkspaceRoleBinding: WorkspaceRoleBinding{
					Role: role,
					Members: &[]types.String{
						types.StringValue("foo@bar.com"),
					},
				},
			},
		},
		{
			name: "to nil",
			oldBindings: &WorkspaceIamBindingModel{
				WorkspaceID: id,
				WorkspaceRoleBinding: WorkspaceRoleBinding{
					Role: role,
					Members: &[]types.String{
						types.StringValue("foo@bar.com"),
					},
				},
			},
			newBindings: nil,
			wantDeleted: &WorkspaceIamBindingModel{
				WorkspaceID: id,
				WorkspaceRoleBinding: WorkspaceRoleBinding{
					Role: role,
					Members: &[]types.String{
						types.StringValue("foo@bar.com"),
					},
				},
			},
			wantAdded: nil,
		},
		{
			name:        "both nil",
			oldBindings: nil,
			newBindings: nil,
			wantDeleted: nil,
			wantAdded:   nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDeleted, gotAdded := DiffWorkspaceIamBindings(tt.oldBindings, tt.newBindings)
			if diff := cmp.Diff(tt.wantDeleted, gotDeleted); diff != "" {
				t.Errorf("DiffWorkspaceIamBindings() deleted mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantAdded, gotAdded); diff != "" {
				t.Errorf("DiffWorkspaceIamBindings() added mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestWorkspaceIamBindingBuildGrantRequests(t *testing.T) {
	id := uuid.New()
	tests := []struct {
		name       string
		iamBinding WorkspaceIamBindingModel
		want       []wsm.SetAccessRequest
	}{
		{
			name: "succeeds",
			iamBinding: WorkspaceIamBindingModel{
				WorkspaceID: types.StringValue(id.String()),
				WorkspaceRoleBinding: WorkspaceRoleBinding{
					Role: types.StringValue(string(wsm.IamRoleREADER)),
					Members: &[]types.String{
						types.StringValue("user1@bar.com"),
						types.StringValue("user2@bar.com"),
						types.StringValue("user3@bar.com"),
					},
				},
			},
			want: []wsm.SetAccessRequest{
				{
					Operation: wsm.GRANT,
					Principal: wsm.Principal{
						UserPrincipal: &wsm.PrincipalUser{
							Email: "user1@bar.com",
						},
					},
					Role: wsm.IamRoleREADER,
				},
				{
					Operation: wsm.GRANT,
					Principal: wsm.Principal{
						UserPrincipal: &wsm.PrincipalUser{
							Email: "user2@bar.com",
						},
					},
					Role: wsm.IamRoleREADER,
				},
				{
					Operation: wsm.GRANT,
					Principal: wsm.Principal{
						UserPrincipal: &wsm.PrincipalUser{
							Email: "user3@bar.com",
						},
					},
					Role: wsm.IamRoleREADER,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.iamBinding.BuildGrantRequests()
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("BuildGrantRequests() request mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestWorkspaceIamBindingBuildRevokeRequests(t *testing.T) {
	id := uuid.New()
	tests := []struct {
		name       string
		iamBinding WorkspaceIamBindingModel
		want       []wsm.SetAccessRequest
	}{
		{
			name: "succeeds",
			iamBinding: WorkspaceIamBindingModel{
				WorkspaceID: types.StringValue(id.String()),
				WorkspaceRoleBinding: WorkspaceRoleBinding{
					Role: types.StringValue(string(wsm.IamRoleWRITER)),
					Members: &[]types.String{
						types.StringValue("user1@bar.com"),
						types.StringValue("user2@bar.com"),
						types.StringValue("user3@bar.com"),
					},
				},
			},
			want: []wsm.SetAccessRequest{
				{
					Operation: wsm.REVOKE,
					Principal: wsm.Principal{
						UserPrincipal: &wsm.PrincipalUser{
							Email: "user1@bar.com",
						},
					},
					Role: wsm.IamRoleWRITER,
				},
				{
					Operation: wsm.REVOKE,
					Principal: wsm.Principal{
						UserPrincipal: &wsm.PrincipalUser{
							Email: "user2@bar.com",
						},
					},
					Role: wsm.IamRoleWRITER,
				},
				{
					Operation: wsm.REVOKE,
					Principal: wsm.Principal{
						UserPrincipal: &wsm.PrincipalUser{
							Email: "user3@bar.com",
						},
					},
					Role: wsm.IamRoleWRITER,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.iamBinding.BuildRevokeRequests()
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("BuildGrantRequests() request mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewWorkspaceIamPolicyModel(t *testing.T) {
	id := uuid.New()
	tests := []struct {
		name         string
		workspaceID  string
		roleBindings *wsm.RoleBindingList
		want         *WorkspaceIamPolicyModel
	}{
		{
			name:        "succeeds",
			workspaceID: id.String(),
			roleBindings: &wsm.RoleBindingList{
				{
					Role:    wsm.IamRoleOWNER,
					Members: &[]string{"foo@bar.com"},
				},
				{
					Role:    wsm.IamRoleWRITER,
					Members: &[]string{"writer1@bar.com", "writer2@bar.com"},
				},
				{
					Role:    wsm.IamRoleREADER,
					Members: &[]string{"reader1@bar.com", "reader2@bar.com"},
				},
			},
			want: &WorkspaceIamPolicyModel{
				WorkspaceID: types.StringValue(id.String()),
				Iams: &[]WorkspaceRoleBinding{
					{
						Role: types.StringValue(string(wsm.IamRoleOWNER)),
						Members: &[]types.String{
							types.StringValue("foo@bar.com"),
						},
					},
					{
						Role: types.StringValue(string(wsm.IamRoleWRITER)),
						Members: &[]types.String{
							types.StringValue("writer1@bar.com"),
							types.StringValue("writer2@bar.com"),
						},
					},
					{
						Role: types.StringValue(string(wsm.IamRoleREADER)),
						Members: &[]types.String{
							types.StringValue("reader1@bar.com"),
							types.StringValue("reader2@bar.com"),
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewWorkspaceIamPolicyModel(tt.workspaceID, tt.roleBindings)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("NewWorkspaceIamPolicyModel() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestWorkspaceIamPolicyBuildGrantRequests(t *testing.T) {
	id := uuid.New()
	tests := []struct {
		name      string
		iamPolicy WorkspaceIamPolicyModel
		want      []wsm.SetAccessRequest
	}{
		{
			name: "succeeds",
			iamPolicy: WorkspaceIamPolicyModel{
				WorkspaceID: types.StringValue(id.String()),
				Iams: &[]WorkspaceRoleBinding{
					{
						Role: types.StringValue(string(wsm.IamRoleREADER)),
						Members: &[]types.String{
							types.StringValue("user1@bar.com"),
							types.StringValue("user2@bar.com"),
						},
					},
					{
						Role: types.StringValue(string(wsm.IamRoleWRITER)),
						Members: &[]types.String{
							types.StringValue("user1@bar.com"),
							types.StringValue("user3@bar.com"),
						},
					},
				},
			},
			want: []wsm.SetAccessRequest{
				{
					Operation: wsm.GRANT,
					Principal: wsm.Principal{
						UserPrincipal: &wsm.PrincipalUser{
							Email: "user1@bar.com",
						},
					},
					Role: wsm.IamRoleREADER,
				},
				{
					Operation: wsm.GRANT,
					Principal: wsm.Principal{
						UserPrincipal: &wsm.PrincipalUser{
							Email: "user2@bar.com",
						},
					},
					Role: wsm.IamRoleREADER,
				},
				{
					Operation: wsm.GRANT,
					Principal: wsm.Principal{
						UserPrincipal: &wsm.PrincipalUser{
							Email: "user1@bar.com",
						},
					},
					Role: wsm.IamRoleWRITER,
				},
				{
					Operation: wsm.GRANT,
					Principal: wsm.Principal{
						UserPrincipal: &wsm.PrincipalUser{
							Email: "user3@bar.com",
						},
					},
					Role: wsm.IamRoleWRITER,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.iamPolicy.BuildGrantRequests()
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("BuildGrantRequests() request mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestWorkspaceIamPolicyBuildRevokeRequests(t *testing.T) {
	id := uuid.New()
	tests := []struct {
		name      string
		iamPolicy WorkspaceIamPolicyModel
		want      []wsm.SetAccessRequest
	}{
		{
			name: "succeeds",
			iamPolicy: WorkspaceIamPolicyModel{
				WorkspaceID: types.StringValue(id.String()),
				Iams: &[]WorkspaceRoleBinding{
					{
						Role: types.StringValue(string(wsm.IamRoleREADER)),
						Members: &[]types.String{
							types.StringValue("user1@bar.com"),
							types.StringValue("user2@bar.com"),
						},
					},
					{
						Role: types.StringValue(string(wsm.IamRoleWRITER)),
						Members: &[]types.String{
							types.StringValue("user1@bar.com"),
							types.StringValue("user3@bar.com"),
						},
					},
				},
			},
			want: []wsm.SetAccessRequest{
				{
					Operation: wsm.REVOKE,
					Principal: wsm.Principal{
						UserPrincipal: &wsm.PrincipalUser{
							Email: "user1@bar.com",
						},
					},
					Role: wsm.IamRoleREADER,
				},
				{
					Operation: wsm.REVOKE,
					Principal: wsm.Principal{
						UserPrincipal: &wsm.PrincipalUser{
							Email: "user2@bar.com",
						},
					},
					Role: wsm.IamRoleREADER,
				},
				{
					Operation: wsm.REVOKE,
					Principal: wsm.Principal{
						UserPrincipal: &wsm.PrincipalUser{
							Email: "user1@bar.com",
						},
					},
					Role: wsm.IamRoleWRITER,
				},
				{
					Operation: wsm.REVOKE,
					Principal: wsm.Principal{
						UserPrincipal: &wsm.PrincipalUser{
							Email: "user3@bar.com",
						},
					},
					Role: wsm.IamRoleWRITER,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.iamPolicy.BuildRevokeRequests()
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("BuildGrantRequests() request mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDiffWorkspaceIamPolicies(t *testing.T) {
	id := types.StringValue(uuid.New().String())
	role := types.StringValue(string(wsm.IamRoleOWNER))
	role2 := types.StringValue(string(wsm.IamRoleREADER))
	role3 := types.StringValue(string(wsm.IamRoleWRITER))
	empty := make([]WorkspaceRoleBinding, 0)
	tests := []struct {
		name        string
		oldPolicy   *WorkspaceIamPolicyModel
		newPolicy   *WorkspaceIamPolicyModel
		wantDeleted []WorkspaceRoleBinding
		wantAdded   []WorkspaceRoleBinding
	}{
		{
			name: "no change",
			oldPolicy: &WorkspaceIamPolicyModel{
				WorkspaceID: id,
				Iams: &[]WorkspaceRoleBinding{
					{
						Role:    role,
						Members: &[]types.String{types.StringValue("foo@bar.com")},
					},
				},
			},
			newPolicy: &WorkspaceIamPolicyModel{
				WorkspaceID: id,
				Iams: &[]WorkspaceRoleBinding{
					{
						Role:    role,
						Members: &[]types.String{types.StringValue("foo@bar.com")},
					},
				},
			},
			wantDeleted: empty,
			wantAdded:   empty,
		},
		{
			name: "reordered but equivalent",
			oldPolicy: &WorkspaceIamPolicyModel{
				WorkspaceID: id,
				Iams: &[]WorkspaceRoleBinding{
					{
						Role:    role,
						Members: &[]types.String{types.StringValue("user1@bar.com")},
					},
					{
						Role:    role2,
						Members: &[]types.String{types.StringValue("user2@bar.com")},
					},
				},
			},
			newPolicy: &WorkspaceIamPolicyModel{
				WorkspaceID: id,
				Iams: &[]WorkspaceRoleBinding{
					{
						Role:    role2,
						Members: &[]types.String{types.StringValue("user2@bar.com")},
					},
					{
						Role:    role,
						Members: &[]types.String{types.StringValue("user1@bar.com")},
					},
				},
			},
			wantDeleted: empty,
			wantAdded:   empty,
		},
		{
			name: "changed iams",
			oldPolicy: &WorkspaceIamPolicyModel{
				WorkspaceID: id,
				Iams: &[]WorkspaceRoleBinding{
					{
						Role: role,
						Members: &[]types.String{
							types.StringValue("user1@bar.com"),
							types.StringValue("user2@bar.com"),
						},
					},
					{
						Role: role2,
						Members: &[]types.String{
							types.StringValue("user3@bar.com"),
						},
					},
				},
			},
			newPolicy: &WorkspaceIamPolicyModel{
				WorkspaceID: id,
				Iams: &[]WorkspaceRoleBinding{
					{
						Role: role,
						Members: &[]types.String{
							types.StringValue("user1@bar.com"),
						},
					},
					{
						Role: role3,
						Members: &[]types.String{
							types.StringValue("user2@bar.com"),
							types.StringValue("user3@bar.com"),
						},
					},
				},
			},
			wantDeleted: []WorkspaceRoleBinding{
				{
					Role: role2,
					Members: &[]types.String{
						types.StringValue("user3@bar.com"),
					},
				},
				{
					Role: role,
					Members: &[]types.String{
						types.StringValue("user2@bar.com"),
					},
				},
			},
			wantAdded: []WorkspaceRoleBinding{
				{
					Role: role3,
					Members: &[]types.String{
						types.StringValue("user2@bar.com"),
						types.StringValue("user3@bar.com"),
					},
				},
			},
		},
		{
			name: "nil iams",
			oldPolicy: &WorkspaceIamPolicyModel{
				WorkspaceID: id,
				Iams:        nil,
			},
			newPolicy: &WorkspaceIamPolicyModel{
				WorkspaceID: id,
				Iams: &[]WorkspaceRoleBinding{
					{
						Role:    role,
						Members: &[]types.String{types.StringValue("foo@bar.com")},
					},
				},
			},
			wantDeleted: empty,
			wantAdded: []WorkspaceRoleBinding{
				{
					Role:    role,
					Members: &[]types.String{types.StringValue("foo@bar.com")},
				},
			},
		},
		{
			name: "empty iams",
			oldPolicy: &WorkspaceIamPolicyModel{
				WorkspaceID: id,
				Iams:        &[]WorkspaceRoleBinding{},
			},
			newPolicy: &WorkspaceIamPolicyModel{
				WorkspaceID: id,
				Iams: &[]WorkspaceRoleBinding{
					{
						Role:    role,
						Members: &[]types.String{types.StringValue("foo@bar.com")},
					},
				},
			},
			wantDeleted: empty,
			wantAdded: []WorkspaceRoleBinding{
				{
					Role:    role,
					Members: &[]types.String{types.StringValue("foo@bar.com")},
				},
			},
		},
		{
			name:      "from nil",
			oldPolicy: nil,
			newPolicy: &WorkspaceIamPolicyModel{
				WorkspaceID: id,
				Iams: &[]WorkspaceRoleBinding{
					{
						Role:    role,
						Members: &[]types.String{types.StringValue("foo@bar.com")},
					},
				},
			},
			wantDeleted: empty,
			wantAdded: []WorkspaceRoleBinding{
				{
					Role:    role,
					Members: &[]types.String{types.StringValue("foo@bar.com")},
				},
			},
		},
		{
			name: "to nil",
			oldPolicy: &WorkspaceIamPolicyModel{
				WorkspaceID: id,
				Iams: &[]WorkspaceRoleBinding{
					{
						Role:    role,
						Members: &[]types.String{types.StringValue("foo@bar.com")},
					},
				},
			},
			newPolicy: nil,
			wantDeleted: []WorkspaceRoleBinding{
				{
					Role:    role,
					Members: &[]types.String{types.StringValue("foo@bar.com")},
				},
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
			gotDeleted, gotAdded := DiffWorkspaceIamPolicies(tt.oldPolicy, tt.newPolicy)
			if diff := cmp.Diff(tt.wantDeleted, gotDeleted); diff != "" {
				t.Errorf("DiffWorkspaceIamPolicies() deleted mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantAdded, gotAdded); diff != "" {
				t.Errorf("DiffWorkspaceIamPolicies() added mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
