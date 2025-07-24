// package models defines the models used in the provider.
package models

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/verily-src/terraform-provider-workbench/internal/openapi/wsm"
)

// workspaceIamMemberModel is the description of a single workspace IAM member.
type workspaceIamMemberModel struct {
	// The user-facing ID of the workspace.
	WorkspaceID types.String `tfsdk:"workspace_id"`
	// Role is the IAM role of the member.
	Role types.String `tfsdk:"role"`
	// Member is the email of the member.
	Member types.String `tfsdk:"member"`
}

// WorkspaceRoleBinding is the description of a workspace IAM role binding.
type WorkspaceRoleBinding struct {
	// Role is the IAM role of the binding.
	Role types.String `tfsdk:"role"`
	// Members is the list of members in the IAM binding.
	Members *[]types.String `tfsdk:"members"`
}

// WorkspaceIamBindingModel is the description of a workspace IAM binding.
type WorkspaceIamBindingModel struct {
	WorkspaceRoleBinding
	// The user-facing ID of the workspace.
	WorkspaceID types.String `tfsdk:"workspace_id"`
}

// WorkspaceIamPolicyModel is the description of a workspace iam policy resource.
type WorkspaceIamPolicyModel struct {
	// The user-facing ID of the workspace.
	WorkspaceID types.String `tfsdk:"workspace_id"`
	// Iams is the list of IAM policies attached to the workspace.
	Iams *[]WorkspaceRoleBinding `tfsdk:"iams"`
}

// BuildGrantRequest builds a SetAccessRequest for granting access to a member
func (m *workspaceIamMemberModel) BuildGrantRequest() (wsm.WorkspaceIdParam, wsm.SetAccessRequest) {
	return m.buildSetAccessRequest(wsm.GRANT)
}

// BuildRevokeRequest builds a SetAccessRequest for revoking access from a
// member
func (m *workspaceIamMemberModel) BuildRevokeRequest() (wsm.WorkspaceIdParam, wsm.SetAccessRequest) {
	return m.buildSetAccessRequest(wsm.REVOKE)
}

func (m *workspaceIamMemberModel) buildSetAccessRequest(op wsm.SetAccessOperation) (wsm.WorkspaceIdParam, wsm.SetAccessRequest) {
	return wsm.WorkspaceIdParam(m.WorkspaceID.ValueString()), wsm.SetAccessRequest{
		Operation: op,
		Role:      wsm.IamRole(m.Role.ValueString()),
		Principal: wsm.Principal{
			UserPrincipal: &wsm.PrincipalUser{
				Email: m.Member.ValueString(),
			},
		},
	}
}

func convertRoleBinding(roleBinding *wsm.RoleBinding) WorkspaceRoleBinding {
	return WorkspaceRoleBinding{
		Role:    types.StringValue(string(roleBinding.Role)),
		Members: convertStringArray(roleBinding.Members),
	}
}

func (b *WorkspaceRoleBinding) compareRole(other *WorkspaceRoleBinding) bool {
	if b == nil && other == nil {
		return true
	}
	if b == nil || other == nil {
		return false
	}
	if !b.Role.Equal(other.Role) {
		return false
	}
	return true
}

func (b *WorkspaceRoleBinding) buildSetAccessRequests(workspaceID types.String, op wsm.SetAccessOperation) []wsm.SetAccessRequest {
	if b == nil || b.Members == nil || len(*b.Members) == 0 {
		return nil
	}

	requests := make([]wsm.SetAccessRequest, 0, len(*b.Members))
	for _, member := range *b.Members {
		iamMember := &workspaceIamMemberModel{
			WorkspaceID: workspaceID,
			Role:        b.Role,
			Member:      member,
		}
		_, req := iamMember.buildSetAccessRequest(op)
		requests = append(requests, req)
	}
	return requests
}

func (b *WorkspaceRoleBinding) toIamBindingModel(workspaceID types.String) *WorkspaceIamBindingModel {
	return &WorkspaceIamBindingModel{
		WorkspaceID:          workspaceID,
		WorkspaceRoleBinding: *b,
	}
}

// NewWorkspaceIamBindingModel creates a new WorkspaceIamBindingModel with the given
// workspace ID and role binding.
func NewWorkspaceIamBindingModel(workspaceID string, roleBinding *wsm.RoleBinding) *WorkspaceIamBindingModel {
	return &WorkspaceIamBindingModel{
		WorkspaceID:          types.StringValue(workspaceID),
		WorkspaceRoleBinding: convertRoleBinding(roleBinding),
	}
}

// DiffWorkspaceIamBindings compares the state and plan WorkspaceIamBindingModel and returns
// the members that were deleted and added.
func DiffWorkspaceIamBindings(oldBinding, newBinding *WorkspaceIamBindingModel) (deleted, added *WorkspaceIamBindingModel) {
	if oldBinding == nil && newBinding == nil {
		return nil, nil
	}
	if oldBinding == nil {
		added = newBinding
		return deleted, added
	}
	if newBinding == nil {
		deleted = oldBinding
		return deleted, added
	}
	if !oldBinding.compareWorkspaceAndRole(newBinding) {
		deleted = oldBinding
		added = newBinding
		return deleted, added
	}

	deletedMembers, addedMembers := diffArrays(oldBinding.Members, newBinding.Members, func(oldMember, newMember types.String) bool {
		return oldMember.Equal(newMember)
	})

	if len(deletedMembers) > 0 {
		deleted = &WorkspaceIamBindingModel{
			WorkspaceID: oldBinding.WorkspaceID,
			WorkspaceRoleBinding: WorkspaceRoleBinding{
				Role:    oldBinding.Role,
				Members: &deletedMembers,
			},
		}
	}
	if len(addedMembers) > 0 {
		added = &WorkspaceIamBindingModel{
			WorkspaceID: oldBinding.WorkspaceID,
			WorkspaceRoleBinding: WorkspaceRoleBinding{
				Role:    newBinding.Role,
				Members: &addedMembers,
			},
		}
	}

	return deleted, added
}

// BuildGrantRequests builds a list of SetAccessRequest for granting access to
// members of an IAM binding.
func (m *WorkspaceIamBindingModel) BuildGrantRequests() []wsm.SetAccessRequest {
	return m.buildSetAccessRequests(wsm.GRANT)
}

// BuildRevokeRequests builds a list of SetAccessRequest for revoking access
// from members of an IAM binding.
func (m *WorkspaceIamBindingModel) BuildRevokeRequests() []wsm.SetAccessRequest {
	return m.buildSetAccessRequests(wsm.REVOKE)
}

func (m *WorkspaceIamBindingModel) buildSetAccessRequests(op wsm.SetAccessOperation) []wsm.SetAccessRequest {
	if m == nil {
		return nil
	}
	return m.WorkspaceRoleBinding.buildSetAccessRequests(m.WorkspaceID, op)
}

// NewWorkspaceIamPolicyModel creates a new WorkspaceModel with a given description.
func NewWorkspaceIamPolicyModel(workspaceID string, roles *wsm.RoleBindingList) *WorkspaceIamPolicyModel {
	return (&WorkspaceIamPolicyModel{
		WorkspaceID: types.StringValue(workspaceID),
		Iams:        convertRolesBindingList(roles),
	}).Normalized()
}

func convertRolesBindingList(roles *wsm.RoleBindingList) *[]WorkspaceRoleBinding {
	if roles == nil {
		return nil
	}
	iamModels := make([]WorkspaceRoleBinding, 0, len(*roles))
	for _, r := range *roles {
		iamModels = append(iamModels, convertRoleBinding(&r))
	}
	return &iamModels
}

func (b *WorkspaceIamBindingModel) compareWorkspaceAndRole(other *WorkspaceIamBindingModel) bool {
	if b == nil && other == nil {
		return true
	}
	if b == nil || other == nil {
		return false
	}
	if !b.WorkspaceID.Equal(other.WorkspaceID) {
		return false
	}
	return (&b.WorkspaceRoleBinding).compareRole(&other.WorkspaceRoleBinding)
}

// Normalized removes empty bindings.
func (p *WorkspaceIamPolicyModel) Normalized() *WorkspaceIamPolicyModel {
	if p == nil || p.Iams == nil {
		return p
	}
	newPolicy := *p
	newIams := make([]WorkspaceRoleBinding, 0, len(*p.Iams))
	for _, iam := range *p.Iams {
		if iam.Members == nil || len(*iam.Members) == 0 {
			continue // Skip empty bindings
		}
		newIams = append(newIams, iam)
	}
	newPolicy.Iams = &newIams
	return &newPolicy
}

// BuildGrantRequests builds a list of SetAccessRequest for granting access to
// members of an IAM policy.
func (m *WorkspaceIamPolicyModel) BuildGrantRequests() []wsm.SetAccessRequest {
	return m.buildSetAccessRequests(wsm.GRANT)
}

// BuildRevokeRequests builds a list of SetAccessRequest for revoking access
// from members of an IAM policy.
func (m *WorkspaceIamPolicyModel) BuildRevokeRequests() []wsm.SetAccessRequest {
	return m.buildSetAccessRequests(wsm.REVOKE)
}

func (m *WorkspaceIamPolicyModel) buildSetAccessRequests(op wsm.SetAccessOperation) []wsm.SetAccessRequest {
	if m == nil || m.Iams == nil {
		return nil
	}

	requests := make([]wsm.SetAccessRequest, 0)
	for _, iam := range *m.Iams {
		reqs := iam.buildSetAccessRequests(m.WorkspaceID, op)
		requests = append(requests, reqs...)
	}
	return requests
}

func WorkspaceIamBindingsToPolicy(workspaceID string, iams []WorkspaceRoleBinding) *WorkspaceIamPolicyModel {
	policy := &WorkspaceIamPolicyModel{
		WorkspaceID: types.StringValue(workspaceID),
	}
	if iams != nil {
		policy.Iams = &iams
	}
	return policy.Normalized()
}

// DiffWorkspaceIamPolicies compares the state and plan WorkspaceIamPolicyModel and returns
// the bindings that were deleted and added.
func DiffWorkspaceIamPolicies(oldPolicy, newPolicy *WorkspaceIamPolicyModel) (deleted, added []WorkspaceRoleBinding) {
	oldPolicy = oldPolicy.Normalized()
	newPolicy = newPolicy.Normalized()

	oldIams := make([]WorkspaceRoleBinding, 0)
	newIams := make([]WorkspaceRoleBinding, 0)
	if oldPolicy != nil && oldPolicy.Iams != nil {
		oldIams = *oldPolicy.Iams
	}
	if newPolicy != nil && newPolicy.Iams != nil {
		newIams = *newPolicy.Iams
	}

	if len(oldIams) == 0 || len(newIams) == 0 || (oldPolicy.WorkspaceID != newPolicy.WorkspaceID) {
		return oldIams, newIams
	}

	deleted, added = diffArrays(&oldIams, &newIams, func(old, new WorkspaceRoleBinding) bool {
		return old.compareRole(&new)
	})

	// Detect changes in members
	for _, o := range oldIams {
		for _, n := range newIams {
			if o.compareRole(&n) {
				deletedBindings, addedBindings := DiffWorkspaceIamBindings(o.toIamBindingModel(oldPolicy.WorkspaceID), n.toIamBindingModel(newPolicy.WorkspaceID))
				if deletedBindings != nil {
					deleted = append(deleted, deletedBindings.WorkspaceRoleBinding)
				}
				if addedBindings != nil {
					added = append(added, addedBindings.WorkspaceRoleBinding)
				}
			}
		}
	}

	return deleted, added
}
